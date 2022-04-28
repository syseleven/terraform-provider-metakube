package metakube

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
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

		t := true
		p := project.NewDeleteClusterV2Params().
			WithProjectID(projectID).
			WithDeleteLoadBalancers(&t).
			WithDeleteVolumes(&t).
			WithClusterID(rec.ID)
		if _, err := meta.client.Project.DeleteClusterV2(p, meta.auth); err != nil {
			return fmt.Errorf("delete cluster: %v", stringifyResponseError(err))
		}
	}

	return nil
}

func TestAccMetakubeCluster_Openstack_Basic(t *testing.T) {
	var cluster models.Cluster

	resourceName := "metakube_cluster.acctest_cluster"
	data := &clusterOpenstackBasicData{
		Name:              makeRandomName(),
		OpenstackAuthURL:  os.Getenv(testEnvOpenstackAuthURL),
		OpenstackUser:     os.Getenv(testEnvOpenstackUsername),
		OpenstackPassword: os.Getenv(testEnvOpenstackPassword),
		OpenstackTenant:   os.Getenv(testEnvOpenstackTenant),
		DatacenterName:    os.Getenv(testEnvOpenstackNodeDC),
		ProjectID:         os.Getenv(testEnvProjectID),
		Version:           os.Getenv(testEnvK8sVersion),
	}
	var config strings.Builder
	if err := clusterOpenstackBasicTemplate.Execute(&config, data); err != nil {
		t.Fatal(err)
	}
	var config2 strings.Builder
	if err := clusterOpenstackBasicTemplate2.Execute(&config2, data); err != nil {
		t.Fatal(err)
	}
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
					resource.TestCheckResourceAttr(resourceName, "spec.0.cloud.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.cloud.0.aws.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.cloud.0.openstack.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.cloud.0.openstack.0.security_group"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.cloud.0.openstack.0.network"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.cloud.0.openstack.0.subnet_id"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.cloud.0.openstack.0.subnet_cidr", "192.168.2.0/24"),
					resource.TestCheckResourceAttrSet(resourceName, "kube_config"),
					// Test spec.0.machine_networks value
					testResourceInstanceState(resourceName, func(is *terraform.InstanceState) error {
						n, err := strconv.Atoi(is.Attributes["spec.0.machine_networks.#"])
						if err != nil {
							return err
						}

						if want := len(cluster.Spec.MachineNetworks); n != want {
							return fmt.Errorf("want len(cluster.Spec.MachineNetworks)=%d, got %d", want, n)
						}

						for i, networks := range cluster.Spec.MachineNetworks {
							prefix := fmt.Sprintf("spec.0.machine_networks.%d.", i)

							var k string

							k = prefix + "cidr"
							if v := is.Attributes[k]; v != networks.CIDR {
								return fmt.Errorf("want %s=%s, got %s", k, networks.CIDR, v)
							}

							k = prefix + "gateway"
							if v := is.Attributes[k]; v != networks.Gateway {
								return fmt.Errorf("want %s=%s, got %s", k, networks.Gateway, v)
							}

							k = prefix + "dns_servers.#"
							n, err = strconv.Atoi(is.Attributes[k])
							if err != nil {
								return err
							}
							if w := len(networks.DNSServers); n != w {
								return fmt.Errorf("want %s=%d, got %d", k, w, n)
							}
							for i, want := range networks.DNSServers {
								k = prefix + fmt.Sprintf("dns_server.%d", i)
								if v := is.Attributes[k]; v != want {
									return fmt.Errorf("want %s=%s, got %s", k, want, v)
								}
							}
						}

						return nil
					}),
					resource.TestCheckResourceAttr(resourceName, "spec.0.audit_logging", "false"),
					resource.TestCheckResourceAttrSet(resourceName, "creation_timestamp"),
					resource.TestCheckResourceAttrSet(resourceName, "deletion_timestamp"),
				),
			},
			{
				Config: config2.String(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMetaKubeClusterExists(&cluster),
					testAccCheckMetaKubeClusterOpenstackAttributes(&cluster, data.Name+"-changed", data.DatacenterName, data.Version, true),
					resource.TestCheckResourceAttr(resourceName, "name", data.Name+"-changed"),
					resource.TestCheckResourceAttr(resourceName, "labels.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "labels.foo", "bar"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.version", data.Version),
					resource.TestCheckResourceAttr(resourceName, "spec.0.update_window.0.start", "Wed 12:00"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.update_window.0.length", "3h"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.services_cidr", "10.240.16.0/18"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pods_cidr", "172.25.0.0/18"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_node_selector", "true"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_security_policy", "true"),
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
					resource.TestCheckResourceAttrSet(resourceName, "oidc_kube_config"),
					resource.TestCheckResourceAttrSet(resourceName, "kube_login_kube_config"),
					// Test spec.0.machine_networks value
					testResourceInstanceState(resourceName, func(is *terraform.InstanceState) error {
						n, err := strconv.Atoi(is.Attributes["spec.0.machine_networks.#"])
						if err != nil {
							return err
						}

						if want := len(cluster.Spec.MachineNetworks); n != want {
							return fmt.Errorf("want len(cluster.Spec.MachineNetworks)=%d, got %d", want, n)
						}

						for i, networks := range cluster.Spec.MachineNetworks {
							prefix := fmt.Sprintf("spec.0.machine_networks.%d.", i)

							var k string

							k = prefix + "cidr"
							if v := is.Attributes[k]; v != networks.CIDR {
								return fmt.Errorf("want %s=%s, got %s", k, networks.CIDR, v)
							}

							k = prefix + "gateway"
							if v := is.Attributes[k]; v != networks.Gateway {
								return fmt.Errorf("want %s=%s, got %s", k, networks.Gateway, v)
							}

							k = prefix + "dns_servers.#"
							n, err = strconv.Atoi(is.Attributes[k])
							if err != nil {
								return err
							}
							if w := len(networks.DNSServers); n != w {
								return fmt.Errorf("want %s=%d, got %d", k, w, n)
							}
							for i, want := range networks.DNSServers {
								k = prefix + fmt.Sprintf("dns_server.%d", i)
								if v := is.Attributes[k]; v != want {
									return fmt.Errorf("want %s=%s, got %s", k, want, v)
								}
							}
						}

						return nil
					}),
					resource.TestCheckResourceAttr(resourceName, "spec.0.audit_logging", "true"),
					resource.TestCheckResourceAttrSet(resourceName, "creation_timestamp"),
					resource.TestCheckResourceAttrSet(resourceName, "deletion_timestamp"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"spec.0.cloud.0.openstack.0.username",
					"spec.0.cloud.0.openstack.0.password",
					"spec.0.cloud.0.openstack.0.tenant",
				},
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
	var cluster models.Cluster
	resourceName := "metakube_cluster.acctest_cluster"
	data := &clusterOpenstackApplicationCredentailsData{
		Name:                                 makeRandomName(),
		DatacenterName:                       os.Getenv(testEnvOpenstackNodeDC),
		ProjectID:                            os.Getenv(testEnvProjectID),
		Version:                              os.Getenv(testEnvK8sVersion),
		OpenstackApplicationCredentialID:     os.Getenv(testEnvOpenstackApplicationCredentialsID),
		OpenstackApplicationCredentialSecret: os.Getenv(testEnvOpenstackApplicationCredentialsSecret),
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
					resource.TestCheckResourceAttr(resourceName, "spec.0.cloud.0.openstack.0.application_credentials_id", data.OpenstackApplicationCredentialID),
					resource.TestCheckResourceAttr(resourceName, "spec.0.cloud.0.openstack.0.application_credentials_secret", data.OpenstackApplicationCredentialSecret),
				),
			},
		},
	})
}

func TestAccMetakubeCluster_Openstack_UpgradeVersion(t *testing.T) {
	var cluster models.Cluster
	resourceName := "metakube_cluster.acctest_cluster"
	versionedConfig := func(version string) string {
		data := &clusterOpenstackBasicData{
			Name:              makeRandomName(),
			Version:           version,
			OpenstackAuthURL:  os.Getenv(testEnvOpenstackAuthURL),
			OpenstackUser:     os.Getenv(testEnvOpenstackUsername),
			OpenstackPassword: os.Getenv(testEnvOpenstackPassword),
			OpenstackTenant:   os.Getenv(testEnvOpenstackTenant),
			DatacenterName:    os.Getenv(testEnvOpenstackNodeDC),
			ProjectID:         os.Getenv(testEnvProjectID),
		}
		var result strings.Builder
		if err := clusterOpenstackBasicTemplate.Execute(&result, data); err != nil {
			t.Fatal(err)
		}
		return result.String()
	}
	versionK8s1 := os.Getenv(testEnvK8sOlderVersion)
	versionK8s2 := os.Getenv(testEnvK8sVersion)

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
	OpenstackAuthURL  string
	OpenstackUser     string
	OpenstackPassword string
	OpenstackTenant   string

	Name           string
	DatacenterName string
	ProjectID      string
	Version        string
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
	user_name = "{{ .OpenstackUser }}"
	password = "{{ .OpenstackPassword }}"
	tenant_name = "{{ .OpenstackTenant }}"
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
				tenant = "{{ .OpenstackTenant }}"
				username = "{{ .OpenstackUser }}"
				password = "{{ .OpenstackPassword }}"
				floating_ip_pool = "ext-net"
				security_group = openstack_networking_secgroup_v2.cluster-net.name
				network = openstack_networking_network_v2.network_tf_test.name
				subnet_id = openstack_networking_subnet_v2.subnet_tf_test.id
				subnet_cidr = "192.168.2.0/24"
			}
		}
		services_cidr = "10.240.16.0/18"
		pods_cidr = "172.25.0.0/18"
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
	Name                                 string
	DatacenterName                       string
	ProjectID                            string
	Version                              string
	OpenstackApplicationCredentialID     string
	OpenstackApplicationCredentialSecret string
}

var clusterOpenstackApplicationCredentialsBasicTemplate = mustParseTemplate("clusterOpenstackApplicationCredentials", `
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
				application_credentials_id="{{ .OpenstackApplicationCredentialID }}"
				application_credentials_secret="{{ .OpenstackApplicationCredentialSecret }}"
			}
		}
	}
}
`)

var clusterOpenstackBasicTemplate2 = mustParseTemplate("clusterOpenstackBasic2", `
terraform {
	required_providers {
		openstack = {
			source = "terraform-provider-openstack/openstack"
		}
	}
}

provider "openstack" {
	auth_url = "{{ .OpenstackAuthURL }}"
	user_name = "{{ .OpenstackUser }}"
	password = "{{ .OpenstackPassword }}"
	tenant_name = "{{ .OpenstackTenant }}"
}

resource "metakube_cluster" "acctest_cluster" {
	name = "{{ .Name }}-changed"
	dc_name = "{{ .DatacenterName }}"
	project_id = "{{ .ProjectID }}"

	# add labels
	labels = {
		"foo" = "bar"
	}

	spec {
		version = "{{ .Version }}"
		update_window {
		  start = "Wed 12:00"
		  length = "3h"
		}
		cloud {
			openstack {
				tenant = "{{ .OpenstackTenant }}"
				username = "{{ .OpenstackUser }}"
				password = "{{ .OpenstackPassword }}"
				floating_ip_pool = "ext-net"
				security_group = openstack_networking_secgroup_v2.cluster-net.name
				network = openstack_networking_network_v2.network_tf_test.name
				subnet_id = openstack_networking_subnet_v2.subnet_tf_test.id
				subnet_cidr = "192.168.2.0/24"
			}
		}

		syseleven_auth {
			realm = "syseleven"
		}

		# enable audit logging
		audit_logging = true

		pod_node_selector = true
		pod_security_policy = true
		services_cidr = "10.240.16.0/18"
		pods_cidr = "172.25.0.0/18"
	}
}

resource "openstack_networking_secgroup_v2" "cluster-net" {
  name = "{{ .Name }}-tf-test"
}

resource "openstack_networking_network_v2" "network_tf_test" {
  name = "{{ .Name }}-network_tf_test"
}

resource "openstack_networking_subnet_v2" "subnet_tf_test" {
  name = "{{ .Name }}-subnet_tf_test"
  network_id = openstack_networking_network_v2.network_tf_test.id
  cidr = "192.168.0.0/16"
  ip_version = 4
}`)

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

		if v, ok := cluster.Spec.Version.(string); !ok || v == "" {
			return fmt.Errorf("cluster version is empty")
		} else if v != k8sVersion {
			return fmt.Errorf("want .Spec.Version=%s, got %s", k8sVersion, v)
		}

		openstack := cluster.Spec.Cloud.Openstack

		if openstack == nil {
			return fmt.Errorf("Cluster cloud is not Openstack")
		}

		if openstack.FloatingIPPool != "ext-net" {
			return fmt.Errorf("want .Spec.Cloud.Openstack.FloatingIPPool=%s, got %s", "ext-net", openstack.FloatingIPPool)
		}

		return nil
	}
}

func TestAccMetakubeCluster_SSHKeys(t *testing.T) {
	var cluster models.Cluster
	var sshkey models.SSHKey
	resourceName := "metakube_cluster.acctest_cluster"

	data := &clusterOpenstackWithSSHKeyData{
		Name:              makeRandomName(),
		OpenstackUser:     os.Getenv(testEnvOpenstackUsername),
		OpenstackPassword: os.Getenv(testEnvOpenstackPassword),
		OpenstackTenant:   os.Getenv(testEnvOpenstackTenant),
		DatacenterName:    os.Getenv(testEnvOpenstackNodeDC),
		ProjectID:         os.Getenv(testEnvProjectID),
		Version:           os.Getenv(testEnvK8sVersion),
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
	Name              string
	DatacenterName    string
	ProjectID         string
	Version           string
	OpenstackTenant   string
	OpenstackUser     string
	OpenstackPassword string
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
				tenant = "{{ .OpenstackTenant }}"
				username = "{{ .OpenstackUser }}"
				password = "{{ .OpenstackPassword }}"
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
				tenant = "{{ .OpenstackTenant }}"
				username = "{{ .OpenstackUser }}"
				password = "{{ .OpenstackPassword }}"
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

func TestAccMetakubeCluster_Azure_Basic(t *testing.T) {
	var cluster models.Cluster
	resourceName := "metakube_cluster.acctest_cluster"

	data := &clusterAzureBasicData{
		Name:            makeRandomName(),
		ProjectID:       os.Getenv(testEnvProjectID),
		ClientID:        os.Getenv(testEnvAzureClientID),
		ClientSecret:    os.Getenv(testEnvAzureClientSecret),
		TenantID:        os.Getenv(testEnvAzureTenantID),
		SubscriptionID:  os.Getenv(testEnvAzureSubscriptionID),
		DatacenterName:  os.Getenv(testEnvAzureNodeDC),
		Version:         os.Getenv(testEnvK8sVersion),
		OpenstackTenant: os.Getenv(testEnvOpenstackTenant),
	}
	var config strings.Builder
	if err := testAccCheckMetaKubeClusterAzureBasic.Execute(&config, data); err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheckForAzure(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckMetaKubeClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: config.String(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMetaKubeClusterExists(&cluster),
					resource.TestCheckResourceAttr(resourceName, "spec.0.cloud.0.azure.0.client_id", data.ClientID),
					resource.TestCheckResourceAttr(resourceName, "spec.0.cloud.0.azure.0.client_secret", data.ClientSecret),
					resource.TestCheckResourceAttr(resourceName, "spec.0.cloud.0.azure.0.tenant_id", data.TenantID),
					resource.TestCheckResourceAttr(resourceName, "spec.0.cloud.0.azure.0.subscription_id", data.SubscriptionID),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"spec.0.cloud.0.azure.#",
					"spec.0.cloud.0.azure.0.%",
					"spec.0.cloud.0.azure.0.availability_set",
					"spec.0.cloud.0.azure.0.openstack_billing_tenant",
					"spec.0.cloud.0.azure.0.resource_group",
					"spec.0.cloud.0.azure.0.route_table",
					"spec.0.cloud.0.azure.0.security_group",
					"spec.0.cloud.0.azure.0.subnet",
					"spec.0.cloud.0.azure.0.vnet",
					"spec.0.cloud.0.azure.0.client_id",
					"spec.0.cloud.0.azure.0.client_secret",
					"spec.0.cloud.0.azure.0.tenant_id",
					"spec.0.cloud.0.azure.0.subscription_id",
				},
			},
		},
	})
}

type clusterAzureBasicData struct {
	Name            string
	DatacenterName  string
	ProjectID       string
	Version         string
	ClientID        string
	ClientSecret    string
	TenantID        string
	SubscriptionID  string
	OpenstackTenant string
}

var testAccCheckMetaKubeClusterAzureBasic = mustParseTemplate("clusterAzureBasic", `
resource "metakube_cluster" "acctest_cluster" {
	name = "{{ .Name }}"
	dc_name = "{{ .DatacenterName }}"
	project_id = "{{ .ProjectID }}"

	spec {
		version = "{{ .Version }}"
		cloud {
			azure {
				client_id = "{{ .ClientID }}"
				client_secret = "{{ .ClientSecret }}"
				tenant_id = "{{ .TenantID }}"
				subscription_id = "{{ .SubscriptionID }}"
				openstack_billing_tenant = "{{ .OpenstackTenant }}"
			}
		}
	}
}`)

func TestAccMetakubeCluster_AWS_Basic(t *testing.T) {
	var cluster models.Cluster
	resourceName := "metakube_cluster.acctest_cluster"
	data := &clusterAWSBasicData{
		Name:            makeRandomName(),
		ProjectID:       os.Getenv(testEnvProjectID),
		AccessID:        os.Getenv(testEnvAWSAccessKeyID),
		AccessSecret:    os.Getenv(testAWSSecretAccessKey),
		VpcID:           os.Getenv(testEnvAWSVPCID),
		DatacenterName:  os.Getenv(testEnvAWSNodeDC),
		Version:         os.Getenv(testEnvK8sVersion),
		OpenstackTenant: os.Getenv(testEnvOpenstackTenant),
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
	Name            string
	DatacenterName  string
	ProjectID       string
	Version         string
	AccessID        string
	AccessSecret    string
	VpcID           string
	OpenstackTenant string
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
				openstack_billing_tenant = "{{ .OpenstackTenant }}"
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
