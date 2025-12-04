package resource_cluster_test

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/syseleven/go-metakube/client/project"
	"github.com/syseleven/go-metakube/models"
	"github.com/syseleven/terraform-provider-metakube/metakube"
	"github.com/syseleven/terraform-provider-metakube/metakube/common"
	"github.com/syseleven/terraform-provider-metakube/metakube/common/provider_testutil"
	"github.com/syseleven/terraform-provider-metakube/metakube/common/testutil"
)

func TestMain(m *testing.M) {
	provider_testutil.TestAccProvider = metakube.Provider()
	provider_testutil.TestAccProviders = map[string]*schema.Provider{
		"metakube": provider_testutil.TestAccProvider,
	}
	resource.TestMain(m)
}

func TestAccMetakubeCluster_Openstack_Basic(t *testing.T) {
	t.Parallel()
	var cluster models.Cluster

	resourceName := "metakube_cluster.acctest_cluster"
	data := &clusterOpenstackBasicData{
		Name:                                  testutil.MakeRandomName() + "-cluster-os-basic",
		OpenstackAuthURL:                      os.Getenv(common.TestEnvOpenstackAuthURL),
		OpenstackApplicationCredentialsID:     common.GetSACredentialId(),
		OpenstackApplicationCredentialsSecret: os.Getenv(common.TestEnvServiceAccountCredential),
		OpenstackProjectID:                    os.Getenv(common.TestEnvOpenstackProjectID),
		OpenstackRegion:                       os.Getenv(common.TestEnvOpenstackRegion),
		DatacenterName:                        os.Getenv(common.TestEnvOpenstackNodeDC),
		ProjectID:                             os.Getenv(common.TestEnvProjectID),
		Version:                               os.Getenv(common.TestEnvK8sVersionOpenstack),
	}
	var config strings.Builder
	if err := clusterOpenstackBasicTemplate.Execute(&config, data); err != nil {
		t.Fatal(err)
	}
	var config2 strings.Builder
	data2 := *data
	data2.CNIPlugin = "cilium"
	data2.IPFamily = "IPv4"
	data2.SyselevenAuth = true
	data2.IAMAuthentication = true
	data2.AuditLogging = true
	data2.PodNodeSelector = true
	if err := clusterOpenstackBasicTemplate.Execute(&config2, data2); err != nil {
		t.Fatal(err)
	}

	t.Log("Generated randomname: ", data.Name)
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testutil.TestAccPreCheckForOpenstack(t)
		},
		ProtoV5ProviderFactories: testutil.TestAccProtoV5ProviderFactories,
		ExternalProviders: map[string]resource.ExternalProvider{
			"openstack": {
				Source: "terraform-provider-openstack/openstack",
			},
		},
		CheckDestroy: testutil.TestAccCheckMetaKubeClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: config.String(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("metakube_cluster.acctest_cluster", plancheck.ResourceActionCreate),
					},
				},
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
					resource.TestCheckResourceAttr(resourceName, "spec.0.cni_plugin.0.type", "cilium"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ip_family", "IPv4"),
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
					resource.TestCheckResourceAttr(resourceName, "spec.0.cni_plugin.0.type", "cilium"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ip_family", "IPv4"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.cloud.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.cloud.0.aws.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.cloud.0.openstack.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.cloud.0.openstack.0.security_group"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.cloud.0.openstack.0.network"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.cloud.0.openstack.0.subnet_id"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.cloud.0.openstack.0.subnet_cidr", "192.168.2.0/24"),
					resource.TestCheckResourceAttrSet(resourceName, "kube_config"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.audit_logging", "true"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_node_selector", "true"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.syseleven_auth.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.syseleven_auth.0.realm", "syseleven"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.syseleven_auth.0.iam_authentication", "true"),
					resource.TestCheckResourceAttrSet(resourceName, "creation_timestamp"),
					resource.TestCheckResourceAttrSet(resourceName, "deletion_timestamp"),
				),
			},
			{
				Config: config2.String(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("metakube_cluster.acctest_cluster", plancheck.ResourceActionNoop),
					},
				},
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"spec.0.cloud.0.openstack.0.application_credentials", "kube_login_kube_config", "oidc_kube_config"},
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
				ExpectError:       regexp.MustCompile(`(no object exists with the given id|Cannot import non-existent remote object)`),
			},
		},
	})
}

func TestAccMetakubeCluster_Openstack_ApplicationCredentials(t *testing.T) {
	t.Parallel()
	var cluster models.Cluster
	resourceName := "metakube_cluster.acctest_cluster"
	data := &clusterOpenstackApplicationCredentailsData{
		Name:                                 testutil.MakeRandomName() + "-appcred",
		DatacenterName:                       os.Getenv(common.TestEnvOpenstackNodeDC),
		ProjectID:                            os.Getenv(common.TestEnvProjectID),
		Version:                              os.Getenv(common.TestEnvK8sVersionOpenstack),
		OpenstackApplicationCredentialID:     common.GetSACredentialId(),
		OpenstackApplicationCredentialSecret: os.Getenv(common.TestEnvServiceAccountCredential),
	}
	var config strings.Builder
	if err := clusterOpenstackApplicationCredentialsBasicTemplate.Execute(&config, data); err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutil.TestAccPreCheckForOpenstack(t) },
		ProtoV5ProviderFactories: testutil.TestAccProtoV5ProviderFactories,
		ExternalProviders: map[string]resource.ExternalProvider{
			"openstack": {
				Source: "terraform-provider-openstack/openstack",
			},
		},
		CheckDestroy: testutil.TestAccCheckMetaKubeClusterDestroy,
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

func TestAccMetakubeCluster_Openstack_UpgradeVersion(t *testing.T) {
	t.Parallel()
	var cluster models.Cluster
	resourceName := "metakube_cluster.acctest_cluster"
	versionedConfig := func(version string) string {
		data := &clusterOpenstackBasicData{
			Name:                                  testutil.MakeRandomName() + "-cluster-os-upgrade",
			Version:                               version,
			OpenstackAuthURL:                      os.Getenv(common.TestEnvOpenstackAuthURL),
			OpenstackApplicationCredentialsID:     common.GetSACredentialId(),
			OpenstackApplicationCredentialsSecret: os.Getenv(common.TestEnvServiceAccountCredential),
			OpenstackProjectID:                    os.Getenv(common.TestEnvOpenstackProjectID),
			DatacenterName:                        os.Getenv(common.TestEnvOpenstackNodeDC),
			ProjectID:                             os.Getenv(common.TestEnvProjectID),
			OpenstackRegion:                       os.Getenv(common.TestEnvOpenstackRegion),
		}
		var result strings.Builder
		if err := clusterOpenstackBasicTemplate.Execute(&result, data); err != nil {
			t.Fatal(err)
		}
		return result.String()
	}
	versionK8s1 := os.Getenv(common.TestEnvK8sOlderVersion)
	versionK8s2 := os.Getenv(common.TestEnvK8sVersionOpenstack)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutil.TestAccPreCheckForOpenstack(t) },
		ProtoV5ProviderFactories: testutil.TestAccProtoV5ProviderFactories,
		ExternalProviders: map[string]resource.ExternalProvider{
			"openstack": {
				Source: "terraform-provider-openstack/openstack",
			},
		},
		CheckDestroy: testutil.TestAccCheckMetaKubeClusterDestroy,
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

	Name              string
	DatacenterName    string
	ProjectID         string
	Version           string
	CNIPlugin         string
	IPFamily          string
	SyselevenAuth     bool
	AuditLogging      bool
	PodNodeSelector   bool
	IAMAuthentication bool
}

var clusterOpenstackBasicTemplate = testutil.MustParseTemplate("clusterOpenstackBasic", `
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
			iam_authentication = {{ .IAMAuthentication }}
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

		{{ if .IPFamily }}
		ip_family = "{{ .IPFamily }}"
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
	OpenstackAuthURL   string
	OpenstackUser      string
	OpenstackPassword  string
	OpenstackProjectID string
	OpenstackRegion    string

	Name                                 string
	DatacenterName                       string
	ProjectID                            string
	Version                              string
	OpenstackApplicationCredentialID     string
	OpenstackApplicationCredentialSecret string
}

var clusterOpenstackApplicationCredentialsBasicTemplate = testutil.MustParseTemplate("clusterOpenstackApplicationCredentials", `
terraform {
	required_providers {
		openstack = {
			source = "terraform-provider-openstack/openstack"
		}
	}
}

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
					id="{{ .OpenstackApplicationCredentialID }}"
					secret="{{ .OpenstackApplicationCredentialSecret }}"
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
		Name:                                  testutil.MakeRandomName() + "-sshkeys",
		OpenstackApplicationCredentialsID:     common.GetSACredentialId(),
		OpenstackApplicationCredentialsSecret: os.Getenv(common.TestEnvServiceAccountCredential),
		OpenstackProjectID:                    os.Getenv(common.TestEnvOpenstackProjectID),
		DatacenterName:                        os.Getenv(common.TestEnvOpenstackNodeDC),
		ProjectID:                             os.Getenv(common.TestEnvProjectID),
		Version:                               os.Getenv(common.TestEnvK8sVersionOpenstack),
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
		PreCheck:     func() { testutil.TestAccPreCheckForOpenstack(t) },
		Providers:    provider_testutil.TestAccProviders,
		CheckDestroy: testutil.TestAccCheckMetaKubeClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: config1.String(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMetaKubeClusterExists(&cluster),
					testutil.TestAccCheckMetaKubeSSHKeyExists("metakube_sshkey.acctest_sshkey1", &sshkey),
					resource.TestCheckResourceAttr(resourceName, "sshkeys.#", "1"),
					testAccCheckMetaKubeClusterHasSSHKey(&cluster.ID, &sshkey.ID),
				),
			},
			{
				Config: config2.String(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMetaKubeClusterExists(&cluster),
					testutil.TestAccCheckMetaKubeSSHKeyExists("metakube_sshkey.acctest_sshkey2", &sshkey),
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

var clusterOpenstackTemplateWithSSHKey1 = testutil.MustParseTemplate("clusterOpenstackWithSSHKey1", `
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

var clusterOpenstackTemplateWithSSHKey2 = testutil.MustParseTemplate("clusterOpenstackWithSSHKey2", `
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
		k, err := testutil.GetTestClient()
		if err != nil {
			return fmt.Errorf("failed to get test client: %v", err)
		}
		p := project.NewListSSHKeysAssignedToClusterV2Params().WithProjectID(projectID).WithClusterID(*cluster)
		ret, err := k.Client.Project.ListSSHKeysAssignedToClusterV2(p, k.Auth)
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

		k, err := testutil.GetTestClient()
		if err != nil {
			return fmt.Errorf("failed to get test client: %v", err)
		}
		projectID := rs.Primary.Attributes["project_id"]
		p := project.NewGetClusterV2Params().WithProjectID(projectID).WithClusterID(rs.Primary.ID)
		ret, err := k.Client.Project.GetClusterV2(p, k.Auth)
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
