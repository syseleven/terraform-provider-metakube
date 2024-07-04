package metakube

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/syseleven/go-metakube/client/project"
	"github.com/syseleven/go-metakube/models"
)

func testSweepClusters(region string) error {
	meta, err := sharedConfigForRegion(region)
	if err != nil {
		return err
	}

	projectID := os.Getenv(testEnvProjectID)
	params := project.NewListClustersV2Params().WithProjectID(projectID)
	records, err := meta.client.Project.ListClustersV2(params, meta.auth)
	if err != nil {
		return fmt.Errorf("sweep list clusters: %s", stringifyResponseError(err))
	}

	for _, rec := range records.Payload {
		if !strings.HasPrefix(rec.Name, testNamePrefix) || !time.Time(rec.DeletionTimestamp).IsZero() {
			continue
		}

		p := project.NewDeleteClusterV2Params().
			WithProjectID(projectID).
			WithClusterID(rec.ID)
		if _, err := meta.client.Project.DeleteClusterV2(p, meta.auth); err != nil {
			return fmt.Errorf("delete cluster: %v", stringifyResponseError(err))
		}
	}

	return nil
}

func TestAccMetakubeCluster_Openstack_Basic(t *testing.T) {
	t.Parallel()
	var cluster models.Cluster

	resourceName := "metakube_cluster.acctest_cluster"
	data := &clusterOpenstackBasicData{
		Name:                                  makeRandomName() + "-basic",
		OpenstackAuthURL:                      os.Getenv(testEnvOpenstackAuthURL),
		OpenstackApplicationCredentialsID:     os.Getenv(testEnvOpenstackApplicationCredentialsID),
		OpenstackApplicationCredentialsSecret: os.Getenv(testEnvOpenstackApplicationCredentialsSecret),
		OpenstackProjectID:                    os.Getenv(testEnvOpenstackProjectID),
		OpenstackRegion:                       os.Getenv(testEnvOpenstackRegion),
		DatacenterName:                        os.Getenv(testEnvOpenstackNodeDC),
		ProjectID:                             os.Getenv(testEnvProjectID),
		Version:                               os.Getenv(testEnvK8sVersionOpenstack),
	}
	var config strings.Builder
	if err := clusterOpenstackBasicTemplate.Execute(&config, data); err != nil {
		t.Fatal(err)
	}
	var config2 strings.Builder
	data2 := *data
	data2.CNIPlugin = "canal"
	data2.SyselevenAuth = true
	data2.AuditLogging = true
	data2.PodNodeSelector = true
	if err := clusterOpenstackBasicTemplate.Execute(&config2, data2); err != nil {
		t.Fatal(err)
	}

	t.Log("Generated randomname: ", data.Name)
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheckForOpenstack(t)
		},
		Providers: testAccProviders,
		ExternalProviders: map[string]resource.ExternalProvider{
			"openstack": {
				Source: "terraform-provider-openstack/openstack",
			},
		},
		CheckDestroy: testAccCheckMetaKubeClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: config.String(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMetaKubeClusterExists(&cluster),
					testAccCheckMetaKubeClusterOpenstackAttributes(&cluster, data.Name, data.DatacenterName, data.Version, false),
					resource.TestCheckResourceAttr(resourceName, "dc_name", data.DatacenterName),
					resource.TestCheckResourceAttr(resourceName, "name", data.Name),
					resource.TestCheckResourceAttr(resourceName, "labels.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "labels.a", "b"),
					resource.TestCheckResourceAttr(resourceName, "labels.c", "d"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.version", data.Version),
					resource.TestCheckResourceAttr(resourceName, "spec.0.update_window.0.start", "Tue 02:00"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.update_window.0.length", "2h"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.services_cidr", "10.240.16.0/18"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pods_cidr", "172.25.0.0/18"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.cni_plugin.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.cni_plugin.0.type", "canal"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.cloud.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.cloud.0.aws.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.cloud.0.openstack.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.cloud.0.openstack.0.security_group"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.cloud.0.openstack.0.network"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.cloud.0.openstack.0.subnet_id"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.cloud.0.openstack.0.subnet_cidr", "192.168.2.0/24"),
					resource.TestCheckResourceAttrSet(resourceName, "kube_config"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.audit_logging", "false"),
					resource.TestCheckResourceAttrSet(resourceName, "creation_timestamp"),
					resource.TestCheckResourceAttrSet(resourceName, "deletion_timestamp"),
				),
			},
			{
				Config: config2.String(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("metakube_cluster.acctest_cluster", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMetaKubeClusterExists(&cluster),
					testAccCheckMetaKubeClusterOpenstackAttributes(&cluster, data2.Name, data2.DatacenterName, data2.Version, true),
					resource.TestCheckResourceAttr(resourceName, "name", data2.Name),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.version", data2.Version),
					resource.TestCheckResourceAttr(resourceName, "spec.0.update_window.0.start", "Tue 02:00"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.update_window.0.length", "2h"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.services_cidr", "10.240.16.0/18"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pods_cidr", "172.25.0.0/18"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.cni_plugin.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.cni_plugin.0.type", "canal"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_node_selector", "true"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.cloud.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.cloud.0.aws.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.cloud.0.openstack.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.cloud.0.openstack.0.floating_ip_pool", "ext-net"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.cloud.0.openstack.0.security_group"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.cloud.0.openstack.0.network"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.cloud.0.openstack.0.subnet_id"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.cloud.0.openstack.0.subnet_cidr", "192.168.2.0/24"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.syseleven_auth.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.syseleven_auth.0.realm", "syseleven"),
					resource.TestCheckResourceAttrSet(resourceName, "kube_config"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.audit_logging", "true"),
					resource.TestCheckResourceAttrSet(resourceName, "creation_timestamp"),
					resource.TestCheckResourceAttrSet(resourceName, "deletion_timestamp"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"spec.0.cloud.0.openstack.0.user_credentials", "kube_login_kube_config", "oidc_kube_config"},
			},
			{
				Config:   config2.String(),
				PlanOnly: true,
			},
			// Test importing non-existent resource provides expected error.
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: false,
				ImportStateId:     "123abc",
				ExpectError:       regexp.MustCompile(`(Please verify the ID is correct|Cannot import non-existent remote object)`),
			},
		},
	})
}

func TestAccMetakubeCluster_Openstack_ApplicationCredentials(t *testing.T) {
	t.Parallel()
	var cluster models.Cluster
	resourceName := "metakube_cluster.acctest_cluster"
	data := &clusterOpenstackApplicationCredentailsData{
		Name:                                 makeRandomName() + "-appcred",
		DatacenterName:                       os.Getenv(testEnvOpenstackNodeDC),
		ProjectID:                            os.Getenv(testEnvProjectID),
		Version:                              os.Getenv(testEnvK8sVersionOpenstack),
		OpenstackApplicationCredentialID:     os.Getenv(testEnvOpenstackApplicationCredentialsID),
		OpenstackApplicationCredentialSecret: os.Getenv(testEnvOpenstackApplicationCredentialsSecret),
		Dynamic:                              false,
	}
	var config strings.Builder
	if err := clusterOpenstackApplicationCredentialsBasicTemplate.Execute(&config, data); err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheckForOpenstack(t) },
		Providers: testAccProviders,
		ExternalProviders: map[string]resource.ExternalProvider{
			"openstack": {
				Source: "terraform-provider-openstack/openstack",
			},
		},
		CheckDestroy: testAccCheckMetaKubeClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: config.String(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMetaKubeClusterExists(&cluster),
					resource.TestCheckResourceAttr(resourceName, "spec.0.cloud.0.openstack.0.application_credentials.0.id", data.OpenstackApplicationCredentialID),
					resource.TestCheckResourceAttr(resourceName, "spec.0.cloud.0.openstack.0.application_credentials.0.secret", data.OpenstackApplicationCredentialSecret),
				),
			},
		},
	})
}

func TestAccMetakubeCluster_Openstack_ApplicationCredentials_Dynammic(t *testing.T) {
	t.Parallel()
	var cluster models.Cluster
	resourceName := "metakube_cluster.acctest_cluster"
	data := &clusterOpenstackApplicationCredentailsData{
		Name:                                  makeRandomName() + "-appcred-dynamic",
		OpenstackAuthURL:                      os.Getenv(testEnvOpenstackAuthURL),
		OpenstackApplicationCredentialsID:     os.Getenv(testEnvOpenstackApplicationCredentialsID),
		OpenstackApplicationCredentialsSecret: os.Getenv(testEnvOpenstackApplicationCredentialsSecret),
		OpenstackProjectID:                    os.Getenv(testEnvOpenstackProjectID),
		OpenstackRegion:                       os.Getenv(testEnvOpenstackRegion),
		DatacenterName:                        os.Getenv(testEnvOpenstackNodeDC),
		ProjectID:                             os.Getenv(testEnvProjectID),
		Version:                               os.Getenv(testEnvK8sVersionOpenstack),
		OpenstackApplicationCredentialID:      os.Getenv(testEnvOpenstackApplicationCredentialsID),
		OpenstackApplicationCredentialSecret:  os.Getenv(testEnvOpenstackApplicationCredentialsSecret),
		Dynamic:                               true,
	}
	var config strings.Builder
	if err := clusterOpenstackApplicationCredentialsBasicTemplate.Execute(&config, data); err != nil {
		t.Fatal(err)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheckForOpenstack(t) },
		Providers: testAccProviders,
		ExternalProviders: map[string]resource.ExternalProvider{
			"openstack": {
				Source: "terraform-provider-openstack/openstack",
			},
		},
		CheckDestroy: testAccCheckMetaKubeClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: config.String(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMetaKubeClusterExists(&cluster),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.cloud.0.openstack.0.application_credentials.0.id"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.cloud.0.openstack.0.application_credentials.0.secret"),
				),
			},
		},
	})
}

func TestAccMetakubeCluster_Openstack_UpgradeVersion(t *testing.T) {
	t.Parallel()
	var cluster models.Cluster
	resourceName := "metakube_cluster.acctest_cluster"
	versionedConfig := func(version string) string {
		data := &clusterOpenstackBasicData{
			Name:                                  makeRandomName() + "-upgrade",
			Version:                               version,
			OpenstackAuthURL:                      os.Getenv(testEnvOpenstackAuthURL),
			OpenstackApplicationCredentialsID:     os.Getenv(testEnvOpenstackApplicationCredentialsID),
			OpenstackApplicationCredentialsSecret: os.Getenv(testEnvOpenstackApplicationCredentialsSecret),
			OpenstackProjectID:                    os.Getenv(testEnvOpenstackProjectID),
			DatacenterName:                        os.Getenv(testEnvOpenstackNodeDC),
			ProjectID:                             os.Getenv(testEnvProjectID),
			OpenstackRegion:                       os.Getenv(testEnvOpenstackRegion),
		}
		var result strings.Builder
		if err := clusterOpenstackBasicTemplate.Execute(&result, data); err != nil {
			t.Fatal(err)
		}
		return result.String()
	}
	versionK8s1 := os.Getenv(testEnvK8sOlderVersion)
	versionK8s2 := os.Getenv(testEnvK8sVersionOpenstack)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheckForOpenstack(t) },
		Providers: testAccProviders,
		ExternalProviders: map[string]resource.ExternalProvider{
			"openstack": {
				Source: "terraform-provider-openstack/openstack",
			},
		},
		CheckDestroy: testAccCheckMetaKubeClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: versionedConfig(versionK8s1),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMetaKubeClusterExists(&cluster),
					resource.TestCheckResourceAttr(resourceName, "spec.0.version", versionK8s1),
				),
			},
			{
				Config: versionedConfig(versionK8s2),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMetaKubeClusterExists(&cluster),
					resource.TestCheckResourceAttr(resourceName, "spec.0.version", versionK8s2),
				),
			},
		},
	})
}

type clusterOpenstackBasicData struct {
	OpenstackAuthURL                      string
	OpenstackApplicationCredentialsID     string
	OpenstackApplicationCredentialsSecret string
	OpenstackProjectID                    string
	OpenstackRegion                       string

	Name            string
	DatacenterName  string
	ProjectID       string
	Version         string
	CNIPlugin       string
	SyselevenAuth   bool
	AuditLogging    bool
	PodNodeSelector bool
}

var clusterOpenstackBasicTemplate = mustParseTemplate("clusterOpenstackBasic", `
terraform {
	required_providers {
		openstack = {
			source = "terraform-provider-openstack/openstack"
		}
	}
}

provider "openstack" {
	auth_url = "{{ .OpenstackAuthURL }}"
	application_credential_id = "{{ .OpenstackApplicationCredentialsID }}"
	application_credential_secret = "{{ .OpenstackApplicationCredentialsSecret }}"
	region = "{{ .OpenstackRegion }}"
}

resource "metakube_cluster" "acctest_cluster" {
	name = "{{ .Name }}"
	dc_name = "{{ .DatacenterName }}"
	project_id = "{{ .ProjectID }}"

	labels = {
		"a" = "b"
		"c" = "d"
	}

    timeouts {
        create = "15m"
        update = "15m"
        delete = "15m"
    }

	spec {
		version = "{{ .Version }}"
		update_window {
		  start = "Tue 02:00"
		  length = "2h"
		}
		cloud {
			openstack {
			    application_credentials {
					id = "{{ .OpenstackApplicationCredentialsID }}"
					secret = "{{ .OpenstackApplicationCredentialsSecret }}"
				}
				floating_ip_pool = "ext-net"
				security_group = openstack_networking_secgroup_v2.cluster-net.name
				network = openstack_networking_network_v2.network_tf_test.name
				subnet_id = openstack_networking_subnet_v2.subnet_tf_test.id
				subnet_cidr = "192.168.2.0/24"
			}
		}

		{{ if .SyselevenAuth }}
		syseleven_auth {
			realm = "syseleven"
		}
		{{ end }}

		{{ if .AuditLogging }}
		audit_logging = true
		{{ end }}

		{{ if .PodNodeSelector }}
		pod_node_selector = true
		{{ end }}

		services_cidr = "10.240.16.0/18"
		pods_cidr = "172.25.0.0/18"
		
		{{ if .CNIPlugin }}
		cni_plugin {
			type = "{{ .CNIPlugin }}"
		  }
		{{ end }}
	}
}

resource "openstack_networking_secgroup_v2" "cluster-net" {
  name = "{{ .Name }}"
}

resource "openstack_networking_network_v2" "network_tf_test" {
  name = "{{ .Name }}"
}

resource "openstack_networking_subnet_v2" "subnet_tf_test" {
  name = "{{ .Name }}"
  network_id = openstack_networking_network_v2.network_tf_test.id
  cidr = "192.168.0.0/16"
  ip_version = 4
}`)

type clusterOpenstackApplicationCredentailsData struct {
	OpenstackAuthURL                      string
	OpenstackApplicationCredentialsID     string
	OpenstackApplicationCredentialsSecret string
	OpenstackProjectID                    string
	OpenstackRegion                       string

	Name                                 string
	DatacenterName                       string
	ProjectID                            string
	Version                              string
	OpenstackApplicationCredentialID     string
	OpenstackApplicationCredentialSecret string
	Dynamic                              bool
}

var clusterOpenstackApplicationCredentialsBasicTemplate = mustParseTemplate("clusterOpenstackApplicationCredentials", `
terraform {
	required_providers {
		openstack = {
			source = "terraform-provider-openstack/openstack"
		}
	}
}

{{ if .Dynamic }}
provider "openstack" {
	auth_url = "{{ .OpenstackAuthURL }}"
	application_credential_id = "{{ .OpenstackApplicationCredentialsID }}"
	application_credential_secret = "{{ .OpenstackApplicationCredentialsSecret }}"
	region = "{{ .OpenstackRegion }}"
}

resource "openstack_identity_application_credential_v3" "app_credential" {
	name        = "{{ .Name }}"
}
{{ end }}

resource "metakube_cluster" "acctest_cluster" {
	name = "{{ .Name }}"
	dc_name = "{{ .DatacenterName }}"
	project_id = "{{ .ProjectID }}"
	
	labels = {
		"a" = "b"
		"c" = "d"
	}

	spec {
		version = "{{ .Version }}"
		update_window {
		  start = "Tue 02:00"
		  length = "2h"
		}
		cloud {
			openstack {
				application_credentials {
{{ if .Dynamic }}
					id=openstack_identity_application_credential_v3.app_credential.id
					secret=openstack_identity_application_credential_v3.app_credential.secret
{{ else }}
					id="{{ .OpenstackApplicationCredentialID }}"
					secret="{{ .OpenstackApplicationCredentialSecret }}"
{{ end }}
				}
			}
		}
	}
}
`)

func testAccCheckMetaKubeClusterOpenstackAttributes(cluster *models.Cluster, name, nodeDC, k8sVersion string, auditLogging bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if cluster.Name != name {
			return fmt.Errorf("want .Name=%s, got %s", name, cluster.Name)
		}

		if cluster.Spec.AuditLogging != nil && cluster.Spec.AuditLogging.Enabled != auditLogging {
			return fmt.Errorf("want .Spec.AuditLogging.Enabled=%v, got %v", auditLogging, cluster.Spec.AuditLogging.Enabled)
		}

		if cluster.Spec.Cloud.DatacenterName != nodeDC {
			return fmt.Errorf("want .Spec.Cloud.DatacenterName=%s, got %s", nodeDC, cluster.Spec.Cloud.DatacenterName)
		}

		if cluster.Spec.Version == "" {
			return fmt.Errorf("cluster version is empty")
		} else if string(cluster.Spec.Version) != k8sVersion {
			return fmt.Errorf("want .Spec.Version=%s, got %s", k8sVersion, cluster.Spec.Version)
		}

		openstack := cluster.Spec.Cloud.Openstack

		if openstack == nil {
			return fmt.Errorf("Cluster cloud is not Openstack")
		}

		if openstack.FloatingIPPool != "ext-net" {
			return fmt.Errorf("want .Spec.Cloud.Openstack.FloatingIPPool=%s, got %s", "ext-net", openstack.FloatingIPPool)
		}

		cniPlugin := cluster.Spec.CniPlugin

		if cniPlugin == nil {
			return fmt.Errorf("CNI plugin is not specified")
		}

		return nil
	}
}

func TestAccMetakubeCluster_SSHKeys(t *testing.T) {
	t.Parallel()
	var cluster models.Cluster
	var sshkey models.SSHKey
	resourceName := "metakube_cluster.acctest_cluster"

	data := &clusterOpenstackWithSSHKeyData{
		Name:                                  makeRandomName() + "-sshkeys",
		OpenstackApplicationCredentialsID:     os.Getenv(testEnvOpenstackApplicationCredentialsID),
		OpenstackApplicationCredentialsSecret: os.Getenv(testEnvOpenstackApplicationCredentialsSecret),
		OpenstackProjectID:                    os.Getenv(testEnvOpenstackProjectID),
		DatacenterName:                        os.Getenv(testEnvOpenstackNodeDC),
		ProjectID:                             os.Getenv(testEnvProjectID),
		Version:                               os.Getenv(testEnvK8sVersionOpenstack),
	}

	var config1 strings.Builder
	err := clusterOpenstackTemplateWithSSHKey1.Execute(&config1, data)
	if err != nil {
		t.Fatal(err)
	}
	var config2 strings.Builder
	err = clusterOpenstackTemplateWithSSHKey2.Execute(&config2, data)
	if err != nil {
		t.Fatal(err)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheckForOpenstack(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckMetaKubeClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: config1.String(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMetaKubeClusterExists(&cluster),
					testAccCheckMetaKubeSSHKeyExists("metakube_sshkey.acctest_sshkey1", &sshkey),
					resource.TestCheckResourceAttr(resourceName, "sshkeys.#", "1"),
					testAccCheckMetaKubeClusterHasSSHKey(&cluster.ID, &sshkey.ID),
				),
			},
			{
				Config: config2.String(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMetaKubeClusterExists(&cluster),
					testAccCheckMetaKubeSSHKeyExists("metakube_sshkey.acctest_sshkey2", &sshkey),
					resource.TestCheckResourceAttr(resourceName, "sshkeys.#", "1"),
					testAccCheckMetaKubeClusterHasSSHKey(&cluster.ID, &sshkey.ID),
				),
			},
		},
	})
}

type clusterOpenstackWithSSHKeyData struct {
	Name                                  string
	DatacenterName                        string
	ProjectID                             string
	Version                               string
	OpenstackProjectID                    string
	OpenstackApplicationCredentialsID     string
	OpenstackApplicationCredentialsSecret string
}

var clusterOpenstackTemplateWithSSHKey1 = mustParseTemplate("clusterOpenstackWithSSHKey1", `
resource "metakube_cluster" "acctest_cluster" {
	name = "{{ .Name }}"
	dc_name = "{{ .DatacenterName }}"
	project_id = "{{ .ProjectID }}"

	sshkeys = [
		metakube_sshkey.acctest_sshkey1.id
	]

	spec {
		version = "{{ .Version }}"
		enable_ssh_agent = true
		cloud {
			openstack {
				application_credentials {
					id = "{{ .OpenstackApplicationCredentialsID }}"
					secret = "{{ .OpenstackApplicationCredentialsSecret }}"
				}
				floating_ip_pool = "ext-net"
			}
		}
	}
}

resource "metakube_sshkey" "acctest_sshkey1" {
	project_id = "{{ .ProjectID }}"
	name = "tf-acc-test-sshkey-1"
	public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCut5oRyqeqYci3E9m6Z6mtxfqkiyb+xNFJM6+/sllhnMDX0vzrNj8PuIFfGkgtowKY//QWLgoB+RpvXqcD4bb4zPkLdXdJPtUf1eAoMh/qgyThUjBs3n7BXvXMDg1Wdj0gq/sTnPLvXsfrSVPjiZvWN4h0JdID2NLnwYuKIiltIn+IbUa6OnyFfOEpqb5XJ7H7LK1mUKTlQ/9CFROxSQf3YQrR9UdtASIeyIZL53WgYgU31Yqy7MQaY1y0fGmHsFwpCK6qFZj1DNruKl/IR1lLx/Bg3z9sDcoBnHKnzSzVels9EVlDOG6bW738ho269QAIrWQYBtznsvWKu5xZPuuj user@machine"
	}`)

var clusterOpenstackTemplateWithSSHKey2 = mustParseTemplate("clusterOpenstackWithSSHKey2", `
resource "metakube_cluster" "acctest_cluster" {
	name = "{{ .Name }}"
	dc_name = "{{ .DatacenterName }}"
	project_id = "{{ .ProjectID }}"

	sshkeys = [
		metakube_sshkey.acctest_sshkey2.id
	]

	spec {
		version = "{{ .Version }}"
		enable_ssh_agent = true
		cloud {
			openstack {
				application_credentials {
					id = "{{ .OpenstackApplicationCredentialsID }}"
					secret = "{{ .OpenstackApplicationCredentialsSecret }}"
				}
				floating_ip_pool = "ext-net"
			}
		}
	}
}

resource "metakube_sshkey" "acctest_sshkey2" {
	project_id = "{{ .ProjectID }}"
	name = "tf-acc-sshkey-2"
	public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCut5oRyqeqYci3E9m6Z6mtxfqkiyb+xNFJM6+/sllhnMDX0vzrNj8PuIFfGkgtowKY//QWLgoB+RpvXqcD4bb4zPkLdXdJPtUf1eAoMh/qgyThUjBs3n7BXvXMDg1Wdj0gq/sTnPLvXsfrSVPjiZvWN4h0JdID2NLnwYuKIiltIn+IbUa6OnyFfOEpqb5XJ7H7LK1mUKTlQ/9CFROxSQf3YQrR9UdtASIeyIZL53WgYgU31Yqy7MQaY1y0fGmHsFwpCK6qFZj1DNruKl/IR1lLx/Bg3z9sDcoBnHKnzSzVels9EVlDOG6bW738ho269QAIrWQYBtznsvWKu5xZPuuj user@machine"
}`)

func testAccCheckMetaKubeClusterHasSSHKey(cluster, sshkey *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["metakube_cluster.acctest_cluster"]
		if !ok {
			return fmt.Errorf("Not found: %s", "metakube_cluster.acctest_project")
		}

		projectID := rs.Primary.Attributes["project_id"]
		k := testAccProvider.Meta().(*metakubeProviderMeta)
		p := project.NewListSSHKeysAssignedToClusterV2Params().WithProjectID(projectID).WithClusterID(*cluster)
		ret, err := k.client.Project.ListSSHKeysAssignedToClusterV2(p, k.auth)
		if err != nil {
			return fmt.Errorf("ListSSHKeysAssignedToCluster %v", err)
		}

		var ids []string
		for _, v := range ret.Payload {
			ids = append(ids, v.ID)
		}

		var sshkeys []string
		if *sshkey != "" {
			sshkeys = []string{*sshkey}
		}
		if diff := cmp.Diff(sshkeys, ids); diff != "" {
			return fmt.Errorf("wrong sshkeys: %s, %s", *sshkey, diff)
		}

		return nil
	}
}

func TestAccMetakubeCluster_AWS_Basic(t *testing.T) {
	t.Parallel()
	var cluster models.Cluster
	resourceName := "metakube_cluster.acctest_cluster"
	data := &clusterAWSBasicData{
		Name:                 makeRandomName() + "-aws-basic",
		ProjectID:            os.Getenv(testEnvProjectID),
		AccessID:             os.Getenv(testEnvAWSAccessKeyID),
		AccessSecret:         os.Getenv(testAWSSecretAccessKey),
		VpcID:                os.Getenv(testEnvAWSVPCID),
		DatacenterName:       os.Getenv(testEnvAWSNodeDC),
		Version:              os.Getenv(testEnvK8sVersionAWS),
		OpenstackProjectName: os.Getenv(testEnvOpenstackProjectName),
	}
	var config strings.Builder
	if err := testAccCheckMetaKubeClusterAWSBasic.Execute(&config, data); err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheckForAWS(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckMetaKubeClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: config.String(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMetaKubeClusterExists(&cluster),
					resource.TestCheckResourceAttr(resourceName, "spec.0.cloud.0.aws.#", "1"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"spec.0.cloud.0.aws.#",
					"spec.0.cloud.0.aws.0.%",
					"spec.0.cloud.0.aws.0.instance_profile_name",
					"spec.0.cloud.0.aws.0.role_arn",
					"spec.0.cloud.0.aws.0.route_table_id",
					"spec.0.cloud.0.aws.0.access_key_id",
					"spec.0.cloud.0.aws.0.security_group_id",
					"spec.0.cloud.0.aws.0.secret_access_key",
					"spec.0.cloud.0.aws.0.vpc_id",
					"spec.0.cloud.0.aws.0.openstack_billing_tenant",
				},
			},
		},
	})
}

type clusterAWSBasicData struct {
	Name                 string
	DatacenterName       string
	ProjectID            string
	Version              string
	AccessID             string
	AccessSecret         string
	VpcID                string
	OpenstackProjectName string
}

var testAccCheckMetaKubeClusterAWSBasic = mustParseTemplate("clusterAWSBasic", `
resource "metakube_cluster" "acctest_cluster" {
	name = "{{ .Name }}"
	dc_name = "{{ .DatacenterName }}"
	project_id = "{{ .ProjectID }}"

	spec {
		version = "{{ .Version }}"
		cloud {
			aws {
				access_key_id = "{{ .AccessID }}"
				secret_access_key = "{{ .AccessSecret }}"
				vpc_id = "{{ .VpcID }}"
				openstack_billing_tenant = "{{ .OpenstackProjectName }}"
			}
		}
	}
}`)

func testAccCheckMetaKubeClusterDestroy(s *terraform.State) error {
	k := testAccProvider.Meta().(*metakubeProviderMeta)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "metakube_cluster" {
			continue
		}

		// Try to find the cluster
		projectID := rs.Primary.Attributes["project_id"]
		p := project.NewGetClusterV2Params().WithProjectID(projectID).WithClusterID(rs.Primary.ID)
		r, err := k.client.Project.GetClusterV2(p, k.auth)
		if err == nil && r.Payload != nil {
			return fmt.Errorf("Cluster still exists")
		}
	}

	return nil
}

func testAccCheckMetaKubeClusterExists(cluster *models.Cluster) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceName := "metakube_cluster.acctest_cluster"
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		k := testAccProvider.Meta().(*metakubeProviderMeta)
		projectID := rs.Primary.Attributes["project_id"]
		p := project.NewGetClusterV2Params().WithProjectID(projectID).WithClusterID(rs.Primary.ID)
		ret, err := k.client.Project.GetClusterV2(p, k.auth)
		if err != nil {
			return fmt.Errorf("GetCluster %v", err)
		}
		if ret.Payload == nil {
			return fmt.Errorf("Record not found")
		}

		*cluster = *ret.Payload

		return nil
	}
}
