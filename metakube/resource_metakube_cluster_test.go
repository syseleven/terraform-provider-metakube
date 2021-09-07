package metakube

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/syseleven/go-metakube/client/project"
	"github.com/syseleven/go-metakube/models"
)

func TestAccMetakubeCluster_Openstack_Basic(t *testing.T) {
	var cluster models.Cluster

	clusterName := makeRandomString()
	resourceName := "metakube_cluster.acctest_cluster"
	authurl := os.Getenv(testEnvOpenstackAuthUrl)
	username := os.Getenv(testEnvOpenstackUsername)
	password := os.Getenv(testEnvOpenstackPassword)
	tenant := os.Getenv(testEnvOpenstackTenant)
	nodeDC := os.Getenv(testEnvOpenstackNodeDC)
	versionK8s17 := os.Getenv(testEnvK8sVersion)
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
				Config: testAccCheckMetaKubeClusterOpenstackBasic(clusterName, authurl, username, password, tenant, nodeDC, versionK8s17),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMetaKubeClusterExists(&cluster),
					testAccCheckMetaKubeClusterOpenstackAttributes(&cluster, clusterName, nodeDC, versionK8s17, false),
					resource.TestCheckResourceAttr(resourceName, "dc_name", nodeDC),
					resource.TestCheckResourceAttr(resourceName, "name", clusterName),
					resource.TestCheckResourceAttr(resourceName, "labels.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "labels.a", "b"),
					resource.TestCheckResourceAttr(resourceName, "labels.c", "d"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.version", versionK8s17),
					resource.TestCheckResourceAttr(resourceName, "spec.0.update_window.0.start", "Tue 02:00"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.update_window.0.length", "2h"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.domain_name", "foodomain.local"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.services_cidr", "10.240.16.0/18"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pods_cidr", "172.25.0.0/18"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.cloud.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.cloud.0.bringyourown.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.cloud.0.aws.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.cloud.0.openstack.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.cloud.0.openstack.0.security_group"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.cloud.0.openstack.0.network"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.cloud.0.openstack.0.subnet_id"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.cloud.0.openstack.0.subnet_cidr", "192.168.2.0/24"),
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
				Config: testAccCheckMetaKubeClusterOpenstackBasic2(clusterName+"-changed", authurl, username, password, tenant, nodeDC, versionK8s17),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMetaKubeClusterExists(&cluster),
					testAccCheckMetaKubeClusterOpenstackAttributes(&cluster, clusterName+"-changed", nodeDC, versionK8s17, true),
					resource.TestCheckResourceAttr(resourceName, "name", clusterName+"-changed"),
					resource.TestCheckResourceAttr(resourceName, "labels.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "labels.foo", "bar"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.version", versionK8s17),
					resource.TestCheckResourceAttr(resourceName, "spec.0.update_window.0.start", "Wed 12:00"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.update_window.0.length", "3h"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.domain_name", "foodomain.local"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.services_cidr", "10.240.16.0/18"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pods_cidr", "172.25.0.0/18"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_node_selector", "true"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_security_policy", "true"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.cloud.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.cloud.0.bringyourown.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.cloud.0.aws.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.cloud.0.openstack.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.cloud.0.openstack.0.floating_ip_pool", "ext-net"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.cloud.0.openstack.0.security_group"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.cloud.0.openstack.0.network"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.cloud.0.openstack.0.subnet_id"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.cloud.0.openstack.0.subnet_cidr", "192.168.2.0/24"),
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
	clusterName := makeRandomString()
	resourceName := "metakube_cluster.acctest_cluster"
	nodeDC := os.Getenv(testEnvOpenstackNodeDC)
	credentialsID := os.Getenv(testEnvOpenstackApplicationCredentialsID)
	credentialsSecret := os.Getenv(testEnvOpenstackApplicationCredentialsSecret)

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
				Config: testAccCheckMetaKubeClusterOpenstackBasicApplicationCredentials(clusterName, credentialsID, credentialsSecret, nodeDC, os.Getenv(testEnvK8sVersion)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMetaKubeClusterExists(&cluster),
					resource.TestCheckResourceAttr(resourceName, "spec.0.cloud.0.openstack.0.application_credentials_id", credentialsID),
					resource.TestCheckResourceAttr(resourceName, "spec.0.cloud.0.openstack.0.application_credentials_secret", credentialsSecret),
				),
			},
		},
	})
}

func TestAccMetakubeCluster_Openstack_UpgradeVersion(t *testing.T) {
	var cluster models.Cluster
	clusterName := makeRandomString()
	resourceName := "metakube_cluster.acctest_cluster"
	authurl := os.Getenv(testEnvOpenstackAuthUrl)
	username := os.Getenv(testEnvOpenstackUsername)
	password := os.Getenv(testEnvOpenstackPassword)
	tenant := os.Getenv(testEnvOpenstackTenant)
	nodeDC := os.Getenv(testEnvOpenstackNodeDC)
	versionedConfig := func(version string) string {
		return testAccCheckMetaKubeClusterOpenstackBasic(clusterName, authurl, username, password, tenant, nodeDC, version)
	}
	versionK8s16 := os.Getenv(testEnvK8sOlderVersion)
	versionK8s17 := os.Getenv(testEnvK8sVersion)

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
				Config: versionedConfig(versionK8s16),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMetaKubeClusterExists(&cluster),
					resource.TestCheckResourceAttr(resourceName, "spec.0.version", versionK8s16),
				),
			},
			{
				Config: versionedConfig(versionK8s17),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMetaKubeClusterExists(&cluster),
					resource.TestCheckResourceAttr(resourceName, "spec.0.version", versionK8s17),
				),
			},
		},
	})
}

func testAccCheckMetaKubeClusterOpenstackBasic(clusterName, authurl, username, password, tenant, nodeDC, version string) string {
	config := `
	terraform {
		required_providers {
			openstack = {
				source = "terraform-provider-openstack/openstack"
			}
		}
	}

	provider "openstack" {
		auth_url = "%s"
		user_name = "%s"
		password = "%s"
		tenant_name = "%s"
	}

	resource "metakube_project" "acctest_project" {
		name = "%s"
	}

	resource "metakube_cluster" "acctest_cluster" {
		name = "%s"
		dc_name = "%s"
		project_id = metakube_project.acctest_project.id
		
		labels = {
			"a" = "b"
		  	"c" = "d"
		}

		spec {
			version = "%s"
			update_window {
			  start = "Tue 02:00"
			  length = "2h"
			}
			cloud {
				openstack {
					tenant = "%s"
					username = "%s"
					password = "%s"
					floating_ip_pool = "ext-net"
					security_group = openstack_networking_secgroup_v2.cluster-net.name
					network = openstack_networking_network_v2.network_tf_test.name
					subnet_id = openstack_networking_subnet_v2.subnet_tf_test.id
					subnet_cidr = "192.168.2.0/24"
				}
			}
			domain_name = "foodomain.local"
			services_cidr = "10.240.16.0/18"
			pods_cidr = "172.25.0.0/18"
		}
	}

	resource "openstack_networking_secgroup_v2" "cluster-net" {
	  name = "%s-tf-test"
	}
	
	resource "openstack_networking_network_v2" "network_tf_test" {
	  name = "%s-network_tf_test"
	}
	
	resource "openstack_networking_subnet_v2" "subnet_tf_test" {
	  name = "%s-subnet_tf_test"
	  network_id = openstack_networking_network_v2.network_tf_test.id
	  cidr = "192.168.0.0/16"
	  ip_version = 4
	}
`

	return fmt.Sprintf(config, authurl, username, password, tenant, clusterName, clusterName, nodeDC, version, tenant, username, password, clusterName, clusterName, clusterName)
}

func testAccCheckMetaKubeClusterOpenstackBasicApplicationCredentials(clusterName, credentialsID, credentialsSecret, nodeDC, version string) string {
	config := `
	terraform {
		required_providers {
			openstack = {
				source = "terraform-provider-openstack/openstack"
			}
		}
	}

	resource "metakube_project" "acctest_project" {
		name = "%s"
	}

	resource "metakube_cluster" "acctest_cluster" {
		name = "%s"
		dc_name = "%s"
		project_id = metakube_project.acctest_project.id
		
		labels = {
			"a" = "b"
		  	"c" = "d"
		}

		spec {
			version = "%s"
			update_window {
			  start = "Tue 02:00"
			  length = "2h"
			}
			cloud {
				openstack {
			                application_credentials_id="%s"
			                application_credentials_secret="%s"
				}
			}
		}
	}
`

	return fmt.Sprintf(config, clusterName, clusterName, nodeDC, version, credentialsID, credentialsSecret)
}

func testAccCheckMetaKubeClusterOpenstackBasic2(clusterName, authurl, username, password, tenant, nodeDC, k8sVersion string) string {
	config := `
	terraform {
		required_providers {
			openstack = {
				source = "terraform-provider-openstack/openstack"
			}
		}
	}

	provider "openstack" {
		auth_url = "%s"
		user_name = "%s"
		password = "%s"
		tenant_name = "%s"
	}

	resource "metakube_project" "acctest_project" {
		name = "%s"
	}

	resource "metakube_cluster" "acctest_cluster" {
		name = "%s"
		dc_name = "%s"
		project_id = metakube_project.acctest_project.id

		# add labels
		labels = {
			"foo" = "bar"
		}

		spec {
			version = "%s"
			update_window {
			  start = "Wed 12:00"
			  length = "3h"
			}
			cloud {
				openstack {
					tenant = "%s"
					username = "%s"
					password = "%s"
					floating_ip_pool = "ext-net"
					security_group = openstack_networking_secgroup_v2.cluster-net.name
					network = openstack_networking_network_v2.network_tf_test.name
					subnet_id = openstack_networking_subnet_v2.subnet_tf_test.id
					subnet_cidr = "192.168.2.0/24"
				}
			}

			# enable audit logging
			audit_logging = true

			pod_node_selector = true
			pod_security_policy = true
			domain_name = "foodomain.local"
			services_cidr = "10.240.16.0/18"
			pods_cidr = "172.25.0.0/18"
		}
	}

	resource "openstack_networking_secgroup_v2" "cluster-net" {
	  name = "%s-tf-test"
	}
	
	resource "openstack_networking_network_v2" "network_tf_test" {
	  name = "%s-network_tf_test"
	}
	
	resource "openstack_networking_subnet_v2" "subnet_tf_test" {
	  name = "%s-subnet_tf_test"
	  network_id = openstack_networking_network_v2.network_tf_test.id
	  cidr = "192.168.0.0/16"
	  ip_version = 4
	}`
	return fmt.Sprintf(config, authurl, username, password, tenant, clusterName, clusterName, nodeDC, k8sVersion, tenant, username, password, clusterName, clusterName, clusterName)
}

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

		// TODO(furkhat): uncomment if API is fixed.
		// if openstack.Username != username {
		// 	return fmt.Errorf("want .Spec.Cloud.Openstack.Username=%s, got %s", openstack.Username, username)
		// }

		// if openstack.Password != password {
		// 	return fmt.Errorf("want .Spec.Cloud.Openstack.Password=%s, got %s", openstack.Password, password)
		// }

		// if openstack.Tenant != tenant {
		// 	return fmt.Errorf("want .Spec.Cloud.Openstack.Tenant=%s, got %s", openstack.Tenant, tenant)
		// }

		return nil
	}
}

func TestAccMetakubeCluster_SSHKeys(t *testing.T) {
	var cluster models.Cluster
	var sshkey models.SSHKey
	clusterName := makeRandomString()
	resourceName := "metakube_cluster.acctest_cluster"
	username := os.Getenv(testEnvOpenstackUsername)
	password := os.Getenv(testEnvOpenstackPassword)
	tenant := os.Getenv(testEnvOpenstackTenant)
	nodeDC := os.Getenv(testEnvOpenstackNodeDC)
	k8sVersion17 := os.Getenv(testEnvK8sVersion)

	configClusterWithKey1 := testAccCheckMetaKubeClusterOpenstackBasicWithSSHKey1(clusterName, username, password, tenant, nodeDC, k8sVersion17)
	configClusterWithKey2 := testAccCheckMetaKubeClusterOpenstackBasicWithSSHKey2(clusterName, username, password, tenant, nodeDC, k8sVersion17)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheckForOpenstack(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckMetaKubeClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: configClusterWithKey1,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMetaKubeClusterExists(&cluster),
					testAccCheckMetaKubeSSHKeyExists("metakube_sshkey.acctest_sshkey1", "metakube_project.acctest_project", &sshkey),
					resource.TestCheckResourceAttr(resourceName, "sshkeys.#", "1"),
					testAccCheckMetaKubeClusterHasSSHKey(&cluster.ID, &sshkey.ID),
				),
			},
			{
				Config: configClusterWithKey2,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMetaKubeClusterExists(&cluster),
					testAccCheckMetaKubeSSHKeyExists("metakube_sshkey.acctest_sshkey2", "metakube_project.acctest_project", &sshkey),
					resource.TestCheckResourceAttr(resourceName, "sshkeys.#", "1"),
					testAccCheckMetaKubeClusterHasSSHKey(&cluster.ID, &sshkey.ID),
				),
			},
		},
	})
}

func testAccCheckMetaKubeClusterOpenstackBasicWithSSHKey1(clusterName, username, password, tenant, nodeDC, k8sVersion string) string {
	config := `
	resource "metakube_project" "acctest_project" {
		name = "%s"
	}

	resource "metakube_cluster" "acctest_cluster" {
		name = "%s"
		dc_name = "%s"
		project_id = metakube_project.acctest_project.id

		sshkeys = [
			metakube_sshkey.acctest_sshkey1.id
		]

		spec {
			version = "%s"
			enable_ssh_agent = true
			cloud {
				openstack {
					tenant = "%s"
					username = "%s"
					password = "%s"
					floating_ip_pool = "ext-net"
				}
			}
		}
	}

	resource "metakube_sshkey" "acctest_sshkey1" {
		project_id = metakube_project.acctest_project.id
		name = "acctest-sshkey-1"
		public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCut5oRyqeqYci3E9m6Z6mtxfqkiyb+xNFJM6+/sllhnMDX0vzrNj8PuIFfGkgtowKY//QWLgoB+RpvXqcD4bb4zPkLdXdJPtUf1eAoMh/qgyThUjBs3n7BXvXMDg1Wdj0gq/sTnPLvXsfrSVPjiZvWN4h0JdID2NLnwYuKIiltIn+IbUa6OnyFfOEpqb5XJ7H7LK1mUKTlQ/9CFROxSQf3YQrR9UdtASIeyIZL53WgYgU31Yqy7MQaY1y0fGmHsFwpCK6qFZj1DNruKl/IR1lLx/Bg3z9sDcoBnHKnzSzVels9EVlDOG6bW738ho269QAIrWQYBtznsvWKu5xZPuuj user@machine"
	}`
	return fmt.Sprintf(config, clusterName, clusterName, nodeDC, k8sVersion, tenant, username, password)
}

func testAccCheckMetaKubeClusterOpenstackBasicWithSSHKey2(clusterName, username, password, tenant, nodeDC, k8sVersion string) string {
	config := `
	resource "metakube_project" "acctest_project" {
		name = "%s"
	}

	resource "metakube_cluster" "acctest_cluster" {
		name = "%s"
		dc_name = "%s"
		project_id = metakube_project.acctest_project.id

		sshkeys = [
			metakube_sshkey.acctest_sshkey2.id
		]

		spec {
			version = "%s"
			enable_ssh_agent = true
			cloud {
				openstack {
					tenant = "%s"
					username = "%s"
					password = "%s"
					floating_ip_pool = "ext-net"
				}
			}
		}
	}

	resource "metakube_sshkey" "acctest_sshkey2" {
		project_id = metakube_project.acctest_project.id
		name = "acctest-sshkey-2"
		public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCut5oRyqeqYci3E9m6Z6mtxfqkiyb+xNFJM6+/sllhnMDX0vzrNj8PuIFfGkgtowKY//QWLgoB+RpvXqcD4bb4zPkLdXdJPtUf1eAoMh/qgyThUjBs3n7BXvXMDg1Wdj0gq/sTnPLvXsfrSVPjiZvWN4h0JdID2NLnwYuKIiltIn+IbUa6OnyFfOEpqb5XJ7H7LK1mUKTlQ/9CFROxSQf3YQrR9UdtASIeyIZL53WgYgU31Yqy7MQaY1y0fGmHsFwpCK6qFZj1DNruKl/IR1lLx/Bg3z9sDcoBnHKnzSzVels9EVlDOG6bW738ho269QAIrWQYBtznsvWKu5xZPuuj user@machine"
	}`
	return fmt.Sprintf(config, clusterName, clusterName, nodeDC, k8sVersion, tenant, username, password)
}

func testAccCheckMetaKubeClusterHasSSHKey(cluster, sshkey *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["metakube_cluster.acctest_cluster"]
		if !ok {
			return fmt.Errorf("Not found: %s", "metakube_project.acctest_project")
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
	clusterName := makeRandomString()
	resourceName := "metakube_cluster.acctest_cluster"
	clientID := os.Getenv(testEnvAzureClientID)
	clientSecret := os.Getenv(testEnvAzureClientSecret)
	tenantID := os.Getenv(testEnvAzureTenantID)
	subsID := os.Getenv(testEnvAzureSubscriptionID)
	nodeDC := os.Getenv(testEnvAzureNodeDC)
	k8sVersion := os.Getenv(testEnvK8sVersion)
	billingTenant := os.Getenv(testEnvOpenstackTenant)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheckForAzure(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckMetaKubeClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckMetaKubeClusterAzureBasic(clusterName, clientID, clientSecret, tenantID, subsID, nodeDC, billingTenant, k8sVersion),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMetaKubeClusterExists(&cluster),
					resource.TestCheckResourceAttr(resourceName, "spec.0.cloud.0.azure.0.client_id", clientID),
					resource.TestCheckResourceAttr(resourceName, "spec.0.cloud.0.azure.0.client_secret", clientSecret),
					resource.TestCheckResourceAttr(resourceName, "spec.0.cloud.0.azure.0.tenant_id", tenantID),
					resource.TestCheckResourceAttr(resourceName, "spec.0.cloud.0.azure.0.subscription_id", subsID),
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

func testAccCheckMetaKubeClusterAzureBasic(n, clientID, clientSecret, tenantID, subscID, nodeDC, billingTenant, k8sVersion string) string {
	return fmt.Sprintf(`
	resource "metakube_project" "acctest_project" {
		name = "%s"
	}

	resource "metakube_cluster" "acctest_cluster" {
		name = "%s"
		dc_name = "%s"
		project_id = metakube_project.acctest_project.id

		spec {
			version = "%s"
			cloud {
				azure {
					client_id = "%s"
					client_secret = "%s"
					tenant_id = "%s"
					subscription_id = "%s"
					openstack_billing_tenant = "%s"
				}
			}
		}
	}`, n, n, nodeDC, k8sVersion, clientID, clientSecret, tenantID, subscID, billingTenant)
}

func TestAccMetakubeCluster_AWS_Basic(t *testing.T) {
	var cluster models.Cluster
	clusterName := makeRandomString()
	resourceName := "metakube_cluster.acctest_cluster"
	awsAccessKeyID := os.Getenv(testEnvAWSAccessKeyID)
	awsSecretAccessKey := os.Getenv(testAWSSecretAccessKey)
	vpcID := os.Getenv(testEnvAWSVPCID)
	nodeDC := os.Getenv(testEnvAWSNodeDC)
	k8sVersion17 := os.Getenv(testEnvK8sVersion)
	billingTenant := os.Getenv(testEnvOpenstackTenant)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheckForAWS(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckMetaKubeClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckMetaKubeClusterAWSBasic(clusterName, awsAccessKeyID, awsSecretAccessKey, vpcID, nodeDC, billingTenant, k8sVersion17),
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

func testAccCheckMetaKubeClusterAWSBasic(n, keyID, keySecret, vpcID, nodeDC, billingTenant, k8sVersion string) string {
	return fmt.Sprintf(`
	resource "metakube_project" "acctest_project" {
		name = "%s"
	}

	resource "metakube_cluster" "acctest_cluster" {
		name = "%s"
		dc_name = "%s"
		project_id = metakube_project.acctest_project.id

		spec {
			version = "%s"
			cloud {
				aws {
					access_key_id = "%s"
					secret_access_key = "%s"
					vpc_id = "%s"
					openstack_billing_tenant = "%s"
				}
			}
		}
	}`, n, n, nodeDC, k8sVersion, keyID, keySecret, vpcID, billingTenant)
}

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
