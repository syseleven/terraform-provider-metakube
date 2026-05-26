package resource_cluster_test

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/syseleven/go-metakube/client/project"
	"github.com/syseleven/go-metakube/models"
	"github.com/syseleven/terraform-provider-metakube/metakube/common"
	"github.com/syseleven/terraform-provider-metakube/metakube/common/testutil"
)

func TestMain(m *testing.M) {
	resource.TestMain(m)
}

func TestAccMetakubeCluster_Openstack_Basic(t *testing.T) {
	t.Parallel()
	var cluster models.Cluster

	networkResourceName := "openstack_networking_network_v2.network_tf_test"
	resourceName := "metakube_cluster.acctest_cluster"
	securityGroupResourceName := "openstack_networking_secgroup_v2.cluster-net"
	subnetResourceName := "openstack_networking_subnet_v2.subnet_tf_test"
	auditLoggingPath := tfjsonpath.New("spec").AtMapKey("audit_logging")
	podNodeSelectorPath := tfjsonpath.New("spec").AtMapKey("pod_node_selector")
	data := &clusterOpenstackBasicData{
		Name:                                  testutil.MakeRandomName() + "-cluster-os-basic",
		OpenstackAuthURL:                      os.Getenv(common.TestEnvOpenstackAuthURL),
		OpenstackApplicationCredentialsID:     common.GetSACredentialId(),
		OpenstackApplicationCredentialsSecret: os.Getenv(common.TestEnvServiceAccountCredential),
		OpenstackProjectID:                    os.Getenv(common.TestEnvProjectID),
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
	data2.Cilium = true
	data2.Clustermesh = true
	if err := clusterOpenstackBasicTemplate.Execute(&config2, &data2); err != nil {
		t.Fatal(err)
	}

	var config3 strings.Builder
	data3 := *data
	data3.CNIPlugin = "cilium"
	data3.Cilium = true
	data3.Clustermesh = false
	data3.IPFamily = "IPv4"
	data3.SyselevenAuth = true
	data3.IAMAuthentication = true
	data3.AuditLogging = true
	data3.PodNodeSelector = true
	if err := clusterOpenstackBasicTemplate.Execute(&config3, &data3); err != nil {
		t.Fatal(err)
	}

	t.Log("Generated randomname: ", data.Name)
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testutil.TestAccPreCheckForOpenstack(t)
		},
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
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
						plancheck.ExpectResourceAction(networkResourceName, plancheck.ResourceActionCreate),
						plancheck.ExpectResourceAction(securityGroupResourceName, plancheck.ResourceActionCreate),
						plancheck.ExpectResourceAction(subnetResourceName, plancheck.ResourceActionCreate),
						plancheck.ExpectResourceAction("metakube_cluster.acctest_cluster", plancheck.ResourceActionCreate),
						plancheck.ExpectKnownValue(resourceName, auditLoggingPath, knownvalue.Bool(false)),
						plancheck.ExpectKnownValue(resourceName, podNodeSelectorPath, knownvalue.Bool(false)),
					},
					PostApplyPreRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
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
					resource.TestCheckResourceAttr(resourceName, "spec.version", data.Version),
					resource.TestCheckResourceAttr(resourceName, "spec.update_window.start", "Tue 02:00"),
					resource.TestCheckResourceAttr(resourceName, "spec.update_window.length", "2h"),
					resource.TestCheckResourceAttr(resourceName, "spec.services_cidr", "10.240.16.0/18"),
					resource.TestCheckResourceAttr(resourceName, "spec.pods_cidr", "172.25.0.0/18"),
					resource.TestCheckResourceAttr(resourceName, "spec.cni_plugin.type", "cilium"),
					resource.TestCheckResourceAttr(resourceName, "spec.ip_family", "IPv4"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.cloud.openstack.security_group"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.cloud.openstack.network"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.cloud.openstack.subnet_id"),
					resource.TestCheckResourceAttr(resourceName, "spec.cloud.openstack.subnet_cidr", "192.168.2.0/24"),
					resource.TestCheckResourceAttrSet(resourceName, "kube_config"),
					resource.TestCheckResourceAttr(resourceName, "spec.audit_logging", "false"),
					resource.TestCheckResourceAttrSet(resourceName, "creation_timestamp"),
					resource.TestCheckResourceAttrSet(resourceName, "deletion_timestamp"),
				),
			},
			{
				Config: config2.String(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMetaKubeClusterExists(&cluster),
					testAccCheckMetaKubeClusterOpenstackAttributes(&cluster, data2.Name, data2.DatacenterName, data2.Version, false),
					resource.TestCheckResourceAttr(resourceName, "spec.cni_plugin.cilium.clustermesh.enable", "true"),
					resource.TestCheckResourceAttr(resourceName, "spec.cni_plugin.cilium.clustermesh.cluster_id", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.cni_plugin.cilium.clustermesh.ipv4_native_routing_cidr", "172.0.0.0/15"),
				),
			},
			{
				Config: config3.String(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMetaKubeClusterExists(&cluster),
					testAccCheckMetaKubeClusterOpenstackAttributes(&cluster, data3.Name, data3.DatacenterName, data3.Version, true),
					resource.TestCheckResourceAttr(resourceName, "dc_name", data.DatacenterName),
					resource.TestCheckResourceAttr(resourceName, "name", data.Name),
					resource.TestCheckResourceAttr(resourceName, "labels.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "labels.a", "b"),
					resource.TestCheckResourceAttr(resourceName, "labels.c", "d"),
					resource.TestCheckResourceAttr(resourceName, "spec.version", data.Version),
					resource.TestCheckResourceAttr(resourceName, "spec.update_window.start", "Tue 02:00"),
					resource.TestCheckResourceAttr(resourceName, "spec.update_window.length", "2h"),
					resource.TestCheckResourceAttr(resourceName, "spec.services_cidr", "10.240.16.0/18"),
					resource.TestCheckResourceAttr(resourceName, "spec.pods_cidr", "172.25.0.0/18"),
					resource.TestCheckResourceAttr(resourceName, "spec.cni_plugin.type", "cilium"),
					resource.TestCheckResourceAttr(resourceName, "spec.ip_family", "IPv4"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.cloud.openstack.security_group"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.cloud.openstack.network"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.cloud.openstack.subnet_id"),
					resource.TestCheckResourceAttr(resourceName, "spec.cloud.openstack.subnet_cidr", "192.168.2.0/24"),
					resource.TestCheckResourceAttrSet(resourceName, "kube_config"),
					resource.TestCheckResourceAttr(resourceName, "spec.audit_logging", "true"),
					resource.TestCheckResourceAttr(resourceName, "spec.pod_node_selector", "true"),
					resource.TestCheckResourceAttr(resourceName, "spec.syseleven_auth.realm", "syseleven"),
					resource.TestCheckResourceAttr(resourceName, "spec.syseleven_auth.iam_authentication", "true"),
					resource.TestCheckResourceAttrSet(resourceName, "creation_timestamp"),
					resource.TestCheckResourceAttrSet(resourceName, "deletion_timestamp"),
				),
			},
			{
				Config: config3.String(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(networkResourceName, plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction(securityGroupResourceName, plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction(subnetResourceName, plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction("metakube_cluster.acctest_cluster", plancheck.ResourceActionNoop),
						plancheck.ExpectKnownValue(resourceName, auditLoggingPath, knownvalue.Bool(true)),
						plancheck.ExpectKnownValue(resourceName, podNodeSelectorPath, knownvalue.Bool(true)),
					},
					PostApplyPreRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"spec.cloud.openstack.application_credentials", "kube_login_kube_config", "oidc_kube_config"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return data.ProjectID + ":" + s.RootModule().Resources[resourceName].Primary.ID, nil
				},
			},
			{
				Config:   config3.String(),
				PlanOnly: true,
			},
			{
				Config: config.String(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(networkResourceName, plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction(securityGroupResourceName, plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction(subnetResourceName, plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction("metakube_cluster.acctest_cluster", plancheck.ResourceActionUpdate),
						plancheck.ExpectKnownValue(resourceName, auditLoggingPath, knownvalue.Bool(false)),
						plancheck.ExpectKnownValue(resourceName, podNodeSelectorPath, knownvalue.Bool(false)),
					},
					PostApplyPreRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMetaKubeClusterExists(&cluster),
				),
			},
			{
				Config: config.String(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(networkResourceName, plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction(securityGroupResourceName, plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction(subnetResourceName, plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction("metakube_cluster.acctest_cluster", plancheck.ResourceActionNoop),
						plancheck.ExpectKnownValue(resourceName, auditLoggingPath, knownvalue.Bool(false)),
						plancheck.ExpectKnownValue(resourceName, podNodeSelectorPath, knownvalue.Bool(false)),
					},
					PostApplyPreRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
			// Test importing non-existent resource provides expected error.
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: false,
				ImportStateId:     data.ProjectID + ":123abc",
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
	dataWithoutSecret := *data
	dataWithoutSecret.OmitApplicationCredentialSecret = true
	var configWithoutSecret strings.Builder
	if err := clusterOpenstackApplicationCredentialsBasicTemplate.Execute(&configWithoutSecret, &dataWithoutSecret); err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutil.TestAccPreCheckForOpenstack(t) },
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
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
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionCreate),
					},
					PostApplyPreRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMetaKubeClusterExists(&cluster),
					resource.TestCheckResourceAttr(resourceName, "spec.cloud.openstack.application_credentials.id", data.OpenstackApplicationCredentialID),
					resource.TestCheckResourceAttr(resourceName, "spec.cloud.openstack.application_credentials.secret", data.OpenstackApplicationCredentialSecret),
				),
			},
			{
				Config:   configWithoutSecret.String(),
				PlanOnly: true,
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
			OpenstackProjectID:                    os.Getenv(common.TestEnvProjectID),
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
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		ExternalProviders: map[string]resource.ExternalProvider{
			"openstack": {
				Source: "terraform-provider-openstack/openstack",
			},
		},
		CheckDestroy: testutil.TestAccCheckMetaKubeClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: versionedConfig(versionK8s1),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionCreate),
					},
					PostApplyPreRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMetaKubeClusterExists(&cluster),
					resource.TestCheckResourceAttr(resourceName, "spec.version", versionK8s1),
				),
			},
			{
				Config: versionedConfig(versionK8s2),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
					PostApplyPreRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMetaKubeClusterExists(&cluster),
					resource.TestCheckResourceAttr(resourceName, "spec.version", versionK8s2),
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
	Cilium            bool
	Clustermesh       bool
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

	spec = {
		version = "{{ .Version }}"
		update_window = {
		  start = "Tue 02:00"
		  length = "2h"
		}
		cloud = {
			openstack = {
			    application_credentials = {
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
		syseleven_auth = {
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
		cni_plugin = {
			type = "{{ .CNIPlugin }}"
			{{ if .Cilium }}
			cilium = {
				clustermesh = {
					enable = {{ .Clustermesh }}
					cluster_id = 1
					ipv4_native_routing_cidr = "172.0.0.0/15"
				}
			}
			{{ end }}
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
	OmitApplicationCredentialSecret      bool
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

	spec = {
		version = "{{ .Version }}"
		update_window = {
		  start = "Tue 02:00"
		  length = "2h"
		}
		cloud = {
			openstack = {
				application_credentials = {
					id="{{ .OpenstackApplicationCredentialID }}"
					{{ if not .OmitApplicationCredentialSecret }}
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
	sshKey1ResourceName := "metakube_sshkey.acctest_sshkey1"
	sshKey2ResourceName := "metakube_sshkey.acctest_sshkey2"

	data := &clusterOpenstackWithSSHKeyData{
		Name:                                  testutil.MakeRandomName() + "-sshkeys",
		SSHKey1Name:                           testutil.MakeRandomName() + "-sshkey1",
		SSHKey2Name:                           testutil.MakeRandomName() + "-sshkey2",
		OpenstackApplicationCredentialsID:     common.GetSACredentialId(),
		OpenstackApplicationCredentialsSecret: os.Getenv(common.TestEnvServiceAccountCredential),
		OpenstackProjectID:                    os.Getenv(common.TestEnvProjectID),
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
		PreCheck:                 func() { testutil.TestAccPreCheckForOpenstack(t) },
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testutil.TestAccCheckMetaKubeClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: config1.String(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionCreate),
						plancheck.ExpectResourceAction(sshKey1ResourceName, plancheck.ResourceActionCreate),
					},
					PostApplyPreRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMetaKubeClusterExists(&cluster),
					testutil.TestAccCheckMetaKubeSSHKeyExists("metakube_sshkey.acctest_sshkey1", &sshkey),
					resource.TestCheckResourceAttr(resourceName, "sshkeys.#", "1"),
					testAccCheckMetaKubeClusterHasSSHKey(&cluster.ID, &sshkey.ID),
				),
			},
			{
				Config: config2.String(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
						plancheck.ExpectResourceAction(sshKey1ResourceName, plancheck.ResourceActionDestroy),
						plancheck.ExpectResourceAction(sshKey2ResourceName, plancheck.ResourceActionCreate),
					},
					PostApplyPreRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
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
	SSHKey1Name                           string
	SSHKey2Name                           string
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

	spec = {
		version = "{{ .Version }}"
		enable_ssh_agent = true
		cloud = {
			openstack = {
				application_credentials = {
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
	name = "{{ .SSHKey1Name }}"
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

	spec = {
		version = "{{ .Version }}"
		enable_ssh_agent = true
		cloud = {
			openstack = {
				application_credentials = {
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
	name = "{{ .SSHKey2Name }}"
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
