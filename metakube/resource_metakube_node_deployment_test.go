package metakube

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/syseleven/go-metakube/client/project"
	"github.com/syseleven/go-metakube/models"
)

func TestAccMetakubeNodeDeployment_Openstack_Basic(t *testing.T) {
	var ndepl models.NodeDeployment
	testName := makeRandomName()
	resourceName := "metakube_node_deployment.acctest_nd"

	projectID := os.Getenv(testEnvProjectID)
	username := os.Getenv(testEnvOpenstackUsername)
	password := os.Getenv(testEnvOpenstackPassword)
	osProjectID := os.Getenv(testEnvOpenstackProjectID)
	nodeDC := os.Getenv(testEnvOpenstackNodeDC)
	image := os.Getenv(testEnvOpenstackImage)
	image2 := os.Getenv(testEnvOpenstackImage2)
	imageFlatcar := os.Getenv(testEnvOpenstackImageFlatcar)
	flavor := os.Getenv(testEnvOpenstackFlavor)
	k8sVersionNew := os.Getenv(testEnvK8sVersion)
	k8sVersionOld := os.Getenv(testEnvK8sOlderVersion)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheckForOpenstack(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckMetaKubeNodeDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckMetaKubeNodeDeploymentBasic(projectID, testName, nodeDC, username, password, osProjectID, k8sVersionOld, k8sVersionOld, image, flavor),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMetaKubeNodeDeploymentExists(resourceName, &ndepl),
					testAccCheckMetaKubeNodeDeploymentFields(&ndepl, flavor, image, k8sVersionOld, 1, 0, false),
					resource.TestCheckResourceAttr(resourceName, "name", testName),
					resource.TestCheckResourceAttrPtr(resourceName, "name", &ndepl.Name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.replicas", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.labels.%", "4"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.labels.a", "b"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.labels.c", "d"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.cloud.0.openstack.0.flavor", flavor),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.cloud.0.openstack.0.image", image),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.operating_system.0.ubuntu.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.versions.0.kubelet", k8sVersionOld),
					resource.TestCheckResourceAttr(resourceName, "spec.0.dynamic_config", "false"),
				),
			},
			{
				Config: testAccCheckMetaKubeNodeDeploymentBasic2(projectID, testName, nodeDC, username, password, osProjectID, k8sVersionNew, k8sVersionNew, image2, flavor),
				Check: resource.ComposeAggregateTestCheckFunc(
					testResourceInstanceState(resourceName, func(is *terraform.InstanceState) error {
						// Record IDs to test import
						if is.ID != ndepl.ID {
							return fmt.Errorf("node deployment not updated. Want ID=%v, got %v", ndepl.ID, is.ID)
						}
						return nil
					}),
					testAccCheckMetaKubeNodeDeploymentExists(resourceName, &ndepl),
					testAccCheckMetaKubeNodeDeploymentFields(&ndepl, flavor, image2, k8sVersionNew, 1, 8, true),
					resource.TestCheckResourceAttr(resourceName, "name", testName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.replicas", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.labels.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.labels.foo", "bar"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.cloud.0.openstack.0.flavor", flavor),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.cloud.0.openstack.0.image", image2),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.cloud.0.openstack.0.use_floating_ip", "true"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.cloud.0.openstack.0.disk_size", "8"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.operating_system.0.ubuntu.0.dist_upgrade_on_boot", "true"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.versions.0.kubelet", k8sVersionNew),
					resource.TestCheckResourceAttr(resourceName, "spec.0.dynamic_config", "true"),
				),
			},
			{
				Config: testAccCheckMetaKubeNodeDeploymentBasic3(projectID, testName, nodeDC, username, password, osProjectID, k8sVersionNew, k8sVersionNew, imageFlatcar, flavor),
				Check: resource.ComposeAggregateTestCheckFunc(
					testResourceInstanceState(resourceName, func(is *terraform.InstanceState) error {
						// Record IDs to test import
						if is.ID != ndepl.ID {
							return fmt.Errorf("node deployment not updated. Want ID=%v, got %v", ndepl.ID, is.ID)
						}
						return nil
					}),
					testAccCheckMetaKubeNodeDeploymentExists(resourceName, &ndepl),
					resource.TestCheckResourceAttr(resourceName, "name", testName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.replicas", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.labels.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.labels.foo", "bar"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.cloud.0.openstack.0.flavor", flavor),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.cloud.0.openstack.0.image", imageFlatcar),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.cloud.0.openstack.0.use_floating_ip", "true"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.cloud.0.openstack.0.disk_size", "10"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.operating_system.0.flatcar.0.disable_auto_update", "true"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.versions.0.kubelet", k8sVersionNew),
					resource.TestCheckResourceAttr(resourceName, "spec.0.dynamic_config", "true"),
				),
			},
			{
				Config:   testAccCheckMetaKubeNodeDeploymentBasic3(projectID, testName, nodeDC, username, password, osProjectID, k8sVersionNew, k8sVersionNew, imageFlatcar, flavor),
				PlanOnly: true,
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					for _, rs := range s.RootModule().Resources {
						if rs.Type == "metakube_node_deployment" {
							return fmt.Sprintf("%s:%s:%s", rs.Primary.Attributes["project_id"], rs.Primary.Attributes["cluster_id"], rs.Primary.ID), nil
						}
					}

					return "", fmt.Errorf("not found")
				},
			},
			// Test importing non-existent resource provides expected error.
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: false,
				ImportStateId:     "a:b:123abc",
				ExpectError:       regexp.MustCompile(`(Please verify the ID is correct|Cannot import non-existent remote object)`),
			},
		},
	})
}

func testAccCheckMetaKubeNodeDeploymentBasic(projectID, testName, nodeDC, username, password, tenant, clusterVersion, kubeletVersion, image, flavor string) string {
	return fmt.Sprintf(`
	resource "metakube_cluster" "acctest_cluster" {
		project_id = "%s"
		name = "%s"
		dc_name = "%s"
		spec {
			version = "%s"
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

	resource "metakube_node_deployment" "acctest_nd" {
		cluster_id = metakube_cluster.acctest_cluster.id
		name = "%s"
		timeouts {
			create = "40m"
			update = "40m"
			delete = "40m"
		}
		spec {
			replicas = 1
			template {
				labels = {
					"a" = "b"
					"c" = "d"
				}
				cloud {
					openstack {
						flavor = "%s"
						image = "%s"
						use_floating_ip = false
						instance_ready_check_period = "10s"
						instance_ready_check_timeout = "4m"
					}
				}
				operating_system {
					ubuntu {}
				}
				versions {
					kubelet = "%s"
				}
			}
		}
	}`, projectID, testName, nodeDC, clusterVersion, tenant, username, password, testName, flavor, image, kubeletVersion)
}

func testAccCheckMetaKubeNodeDeploymentBasic2(projectID, testName, nodeDC, username, password, tenant, clusterVersion, kubeletVersion, image, flavor string) string {
	return fmt.Sprintf(`
	resource "metakube_cluster" "acctest_cluster" {
		project_id = "%s"
		name = "%s"
		dc_name = "%s"
		labels = {
			"cluster-label" = "val"
		}
		spec {
			version = "%s"
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

	resource "metakube_node_deployment" "acctest_nd" {
		cluster_id = metakube_cluster.acctest_cluster.id
		name = "%s"
		spec {
			dynamic_config = true
			replicas = 1
			template {
				labels = {
					"foo" = "bar"
				}
				cloud {
					openstack {
						flavor = "%s"
						image = "%s"
						disk_size = 8
						use_floating_ip = true
						instance_ready_check_period = "10s"
						instance_ready_check_timeout = "4m"
					}
				}
				operating_system {
					ubuntu {
						dist_upgrade_on_boot = true
					}
				}
				versions {
					kubelet = "%s"
				}
			}
		}
	}`, projectID, testName, nodeDC, clusterVersion, tenant, username, password, testName, flavor, image, kubeletVersion)
}

func testAccCheckMetaKubeNodeDeploymentBasic3(projectID, testName, nodeDC, username, password, tenant, clusterVersion, kubeletVersion, image, flavor string) string {
	return fmt.Sprintf(`
	resource "metakube_cluster" "acctest_cluster" {
		project_id = "%s"
		name = "%s"
		dc_name = "%s"
		labels = {
			"cluster-label" = "val"
		}
		spec {
			version = "%s"
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

	resource "metakube_node_deployment" "acctest_nd" {
		cluster_id = metakube_cluster.acctest_cluster.id
		name = "%s"
		spec {
			dynamic_config = true
			replicas = 1
			template {
				labels = {
					"foo" = "bar"
				}
				cloud {
					openstack {
						flavor = "%s"
						image = "%s"
						disk_size = 10
						use_floating_ip = true
						instance_ready_check_period = "10s"
						instance_ready_check_timeout = "4m"
					}
				}
				operating_system {
					flatcar {
						disable_auto_update = true
					}
				}
				versions {
					kubelet = "%s"
				}
			}
		}
	}`, projectID, testName, nodeDC, clusterVersion, tenant, username, password, testName, flavor, image, kubeletVersion)
}

func testAccCheckMetaKubeNodeDeploymentDestroy(s *terraform.State) error {
	return nil
}

func testAccCheckMetaKubeNodeDeploymentFields(rec *models.NodeDeployment, flavor, image, kubeletVersion string, replicas, diskSize int, distUpgrade bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if rec == nil {
			return fmt.Errorf("No Record")
		}

		if rec.Spec == nil || rec.Spec.Template == nil || rec.Spec.Template.Cloud == nil || rec.Spec.Template.Cloud.Openstack == nil {
			return fmt.Errorf("No Openstack cloud spec present")
		}

		openstack := rec.Spec.Template.Cloud.Openstack

		if openstack.Flavor == nil {
			return fmt.Errorf("No Flavor spec present")
		}
		if *openstack.Flavor != flavor {
			return fmt.Errorf("Flavor=%s, want %s", *openstack.Flavor, flavor)
		}

		if openstack.Image == nil {
			return fmt.Errorf("No Image spec present")
		}

		if *openstack.Image != image {
			return fmt.Errorf("Image=%s, want %s", *openstack.Image, image)
		}

		if openstack.RootDiskSizeGB != int64(diskSize) {
			return fmt.Errorf("RootDiskSizeGB=%d, want %d", openstack.RootDiskSizeGB, diskSize)
		}

		opSys := rec.Spec.Template.OperatingSystem
		if opSys == nil {
			return fmt.Errorf("No OperatingSystem spec present")
		}

		ubuntu := opSys.Ubuntu
		if ubuntu == nil {
			return fmt.Errorf("No Ubuntu spec present")
		}

		if ubuntu.DistUpgradeOnBoot != distUpgrade {
			return fmt.Errorf("want Ubuntu.DistUpgradeOnBoot=%v, got %v", ubuntu.DistUpgradeOnBoot, distUpgrade)
		}

		versions := rec.Spec.Template.Versions
		if versions == nil {
			return fmt.Errorf("No Versions")
		}

		if versions.Kubelet != kubeletVersion {
			return fmt.Errorf("Versions.Kubelet=%s, want %s", versions.Kubelet, kubeletVersion)
		}

		if rec.Spec.Replicas == nil || *rec.Spec.Replicas != int32(replicas) {
			return fmt.Errorf("Replicas=%d, want %d", rec.Spec.Replicas, replicas)
		}

		return nil
	}
}

func TestAccMetakubeNodeDeployment_Azure_Basic(t *testing.T) {
	var nodedepl models.NodeDeployment
	testName := makeRandomName()

	projectID := os.Getenv(testEnvProjectID)
	clientID := os.Getenv(testEnvAzureClientID)
	clientSecret := os.Getenv(testEnvAzureClientSecret)
	tenantID := os.Getenv(testEnvAzureTenantID)
	subsID := os.Getenv(testEnvAzureSubscriptionID)
	nodeDC := os.Getenv(testEnvAzureNodeDC)
	nodeSize := os.Getenv(testEnvAzureNodeSize)
	k8sVersion17 := os.Getenv(testEnvK8sVersion)
	osProject := os.Getenv(testEnvOpenstackProjectID)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheckForAzure(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckMetaKubeNodeDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckMetaKubeNodeDeploymentAzureBasic(projectID, testName, osProject, clientID, clientSecret, tenantID, subsID, nodeDC, nodeSize, k8sVersion17),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMetaKubeNodeDeploymentExists("metakube_node_deployment.acctest_nd", &nodedepl),
					resource.TestCheckResourceAttr("metakube_node_deployment.acctest_nd", "spec.0.template.0.cloud.0.azure.0.size", nodeSize),
				),
			},
			{
				ResourceName:      "metakube_node_deployment.acctest_nd",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					for _, rs := range s.RootModule().Resources {
						if rs.Type == "metakube_node_deployment" {
							return fmt.Sprintf("%s:%s:%s", rs.Primary.Attributes["project_id"], rs.Primary.Attributes["cluster_id"], rs.Primary.ID), nil
						}
					}

					return "", fmt.Errorf("not found")
				},
			},
		},
	})
}

func testAccCheckMetaKubeNodeDeploymentAzureBasic(projectID, n, billing, clientID, clientSecret, tenantID, subscID, nodeDC, nodeSize, k8sVersion string) string {
	return fmt.Sprintf(`
	resource "metakube_cluster" "acctest_cluster" {
		project_id = "%s"
		name = "%s"
		dc_name = "%s"

		spec {
			version = "%s"
			cloud {
				azure {
					openstack_billing_tenant = "%s"
					client_id = "%s"
					client_secret = "%s"
					tenant_id = "%s"
					subscription_id = "%s"
				}
			}
		}
	}

	resource "metakube_node_deployment" "acctest_nd" {
		cluster_id = metakube_cluster.acctest_cluster.id
		name = "%s"
		spec {
			replicas = 1
			template {
				cloud {
					azure {
						size = "%s"
					}
				}
				operating_system {
					ubuntu {
						dist_upgrade_on_boot = false
					}
				}
				versions {
					kubelet = "%s"
				}
			}
		}
	}`, projectID, n, nodeDC, k8sVersion, billing, clientID, clientSecret, tenantID, subscID, n, nodeSize, k8sVersion)
}

func TestAccMetakubeNodeDeployment_AWS_Basic(t *testing.T) {
	var nodedepl models.NodeDeployment
	testName := makeRandomName()

	projectID := os.Getenv(testEnvProjectID)
	accessKeyID := os.Getenv(testEnvAWSAccessKeyID)
	accessKeySecret := os.Getenv(testAWSSecretAccessKey)
	vpcID := os.Getenv(testEnvAWSVPCID)
	nodeDC := os.Getenv(testEnvAWSNodeDC)
	instanceType := os.Getenv(testEnvAWSInstanceType)
	subnetID := os.Getenv(testEnvAWSSubnetID)
	availabilityZone := os.Getenv(testEnvAWSAvailabilityZone)
	diskSize := os.Getenv(testEnvAWSDiskSize)
	k8sVersion := os.Getenv(testEnvK8sVersion)
	osProject := os.Getenv(testEnvOpenstackProjectID)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheckForAWS(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckMetaKubeNodeDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckMetaKubeNodeDeploymentAWSBasic(projectID, testName, osProject, accessKeyID, accessKeySecret, vpcID, nodeDC, instanceType, subnetID, availabilityZone, diskSize, k8sVersion, k8sVersion),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMetaKubeNodeDeploymentExists("metakube_node_deployment.acctest_nd", &nodedepl),
					resource.TestCheckResourceAttr("metakube_node_deployment.acctest_nd", "spec.0.template.0.cloud.0.aws.0.instance_type", instanceType),
					resource.TestCheckResourceAttr("metakube_node_deployment.acctest_nd", "spec.0.template.0.cloud.0.aws.0.disk_size", diskSize),
					resource.TestCheckResourceAttr("metakube_node_deployment.acctest_nd", "spec.0.template.0.cloud.0.aws.0.volume_type", "standard"),
					resource.TestCheckResourceAttr("metakube_node_deployment.acctest_nd", "spec.0.template.0.cloud.0.aws.0.subnet_id", subnetID),
					resource.TestCheckResourceAttr("metakube_node_deployment.acctest_nd", "spec.0.template.0.cloud.0.aws.0.availability_zone", availabilityZone),
					resource.TestCheckResourceAttr("metakube_node_deployment.acctest_nd", "spec.0.template.0.cloud.0.aws.0.assign_public_ip", "true"),
				),
			},
		},
	})
}

func testAccCheckMetaKubeNodeDeploymentAWSBasic(projectID, n, billing, keyID, keySecret, vpcID, nodeDC, instanceType, subnetID, availabilityZone, diskSize, k8sVersion, kubeletVersion string) string {
	return fmt.Sprintf(`
	resource "metakube_cluster" "acctest_cluster" {
		name = "%s"
		dc_name = "%s"
		project_id = "%s"

		spec {
			version = "%s"
			cloud {
				aws {
					openstack_billing_tenant = "%s"
					access_key_id = "%s"
					secret_access_key = "%s"
					vpc_id = "%s"
				}
			}
		}
	}

	resource "metakube_node_deployment" "acctest_nd" {
		cluster_id = metakube_cluster.acctest_cluster.id
		spec {
			replicas = 1
			template {
				cloud {
					aws {
						instance_type = "%s"
						disk_size = "%s"
						volume_type = "standard"
						subnet_id = "%s"
						availability_zone = "%s"
						assign_public_ip = true
					}
				}
				operating_system {
					ubuntu {
						dist_upgrade_on_boot = false
					}
				}
				versions {
					kubelet = "%s"
				}
			}
		}
	}`, n, nodeDC, projectID, k8sVersion, billing, keyID, keySecret, vpcID, instanceType, diskSize, subnetID, availabilityZone, kubeletVersion)
}

func testAccCheckMetaKubeNodeDeploymentExists(n string, rec *models.NodeDeployment) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		k := testAccProvider.Meta().(*metakubeProviderMeta)

		p := project.NewGetMachineDeploymentParams().
			WithProjectID(rs.Primary.Attributes["project_id"]).
			WithClusterID(rs.Primary.Attributes["cluster_id"]).
			WithMachineDeploymentID(rs.Primary.ID)
		r, err := k.client.Project.GetMachineDeployment(p, k.auth)
		if err != nil {
			return fmt.Errorf("GetNodeDeployment: %v", err)
		}
		*rec = *r.Payload

		return nil
	}
}

func TestAccMetakubeNodeDeployment_ValidationAgainstCluster(t *testing.T) {
	testName := makeRandomName()

	projectID := os.Getenv(testEnvProjectID)
	osProjectID := os.Getenv(testEnvOpenstackProjectID)
	accessKeyID := os.Getenv(testEnvAWSAccessKeyID)
	accessKeySecret := os.Getenv(testAWSSecretAccessKey)
	vpcID := os.Getenv(testEnvAWSVPCID)
	nodeDC := os.Getenv(testEnvAWSNodeDC)
	k8sVersion17 := os.Getenv(testEnvK8sVersion)
	instanceType := os.Getenv(testEnvAWSInstanceType)
	subnetID := os.Getenv(testEnvAWSSubnetID)
	availabilityZone := os.Getenv(testEnvAWSAvailabilityZone)
	diskSize := os.Getenv(testEnvAWSDiskSize)

	unavailableVersion := "1.12.1"
	bigVersion := "3.0.0"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheckForAWS(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckMetaKubeClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckMetaKubeNodeDeploymentBasicValidation(testName, projectID, osProjectID, accessKeyID, accessKeySecret, vpcID, nodeDC, instanceType, subnetID, availabilityZone, diskSize, k8sVersion17, k8sVersion17),
			},
			{
				Config:      testAccCheckMetaKubeNodeDeploymentBasicValidation(testName, projectID, osProjectID, accessKeyID, accessKeySecret, vpcID, nodeDC, instanceType, subnetID, availabilityZone, diskSize, k8sVersion17, unavailableVersion),
				ExpectError: regexp.MustCompile(fmt.Sprintf(`unknown version for node deployment %s, available versions`, unavailableVersion)),
			},
			{
				Config:      testAccCheckMetaKubeNodeDeploymentTypeValidation(testName, projectID, osProjectID, accessKeyID, accessKeySecret, vpcID, nodeDC, k8sVersion17, k8sVersion17),
				ExpectError: regexp.MustCompile(`provider for node deployment must \(.*\) match cluster provider \(.*\)`),
			},
			{
				Config:      testAccCheckMetaKubeNodeDeploymentBasicValidation(testName, projectID, osProjectID, accessKeyID, accessKeySecret, vpcID, nodeDC, instanceType, subnetID, availabilityZone, diskSize, k8sVersion17, bigVersion),
				ExpectError: regexp.MustCompile(`cannot be greater than cluster version`),
			},
		},
	})
}

func testAccCheckMetaKubeNodeDeploymentBasicValidation(n, projectID, billing, keyID, keySecret, vpcID, nodeDC, instanceType, subnetID, availabilityZone, diskSize, k8sVersion, kubeletVersion string) string {
	return fmt.Sprintf(`
	resource "metakube_cluster" "acctest_cluster" {
		name = "%s"
		dc_name = "%s"
		project_id = "%s"

		spec {
			version = "%s"
			cloud {
				aws {
				    openstack_billing_tenant = "%s"
					access_key_id = "%s"
					secret_access_key = "%s"
					vpc_id = "%s"
				}
			}
		}
	}

	resource "metakube_node_deployment" "acctest_nd" {
		cluster_id = metakube_cluster.acctest_cluster.id
		name = "%s"
		spec {
			replicas = 1
			template {
				cloud {
					aws {
						instance_type = "%s"
						disk_size = "%s"
						volume_type = "standard"
						subnet_id = "%s"
						availability_zone = "%s"
						assign_public_ip = true
					}
				}
				operating_system {
					ubuntu {
						dist_upgrade_on_boot = false
					}
				}
				versions {
					kubelet = "%s"
				}
			}
		}
	}`, n, nodeDC, projectID, k8sVersion, billing, keyID, keySecret, vpcID, n, instanceType, diskSize, subnetID, availabilityZone, kubeletVersion)
}

func testAccCheckMetaKubeNodeDeploymentTypeValidation(n, projectID, billing, keyID, keySecret, vpcID, nodeDC, k8sVersion, kubeletVersion string) string {
	return fmt.Sprintf(`
	resource "metakube_cluster" "acctest_cluster" {
		name = "%s"
		dc_name = "%s"
		project_id = "%s"

		spec {
			version = "%s"
			cloud {
				aws {
		            openstack_billing_tenant = "%s"
					access_key_id = "%s"
					secret_access_key = "%s"
					vpc_id = "%s"
				}
			}
		}
	}

	resource "metakube_node_deployment" "acctest_nd" {
		cluster_id = metakube_cluster.acctest_cluster.id
		name = "%s"
		spec {
			replicas = 1
			template {
				cloud {
					azure {
						size = 2
					}
				}
				operating_system {
					ubuntu {
						dist_upgrade_on_boot = false
					}
				}
				versions {
					kubelet = "%s"
				}
			}
		}
	}`, n, nodeDC, projectID, k8sVersion, billing, keyID, keySecret, vpcID, n, kubeletVersion)
}
