package metakube

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/syseleven/go-metakube/client/project"
	"github.com/syseleven/go-metakube/models"
)

func TestAccMetakubeNodeDeployment_Openstack_Basic(t *testing.T) {
	var ndepl models.NodeDeployment
	var sgroupID string
	resourceName := "metakube_node_deployment.acctest_nd"
	serverGroupResourceName := "openstack_compute_servergroup_v2.acctest_sg"

	data := &nodeDeploymentBasicData{
		Name:                                  makeRandomName() + "-os-nodedepl",
		OpenstackAuthURL:                      os.Getenv(testEnvOpenstackAuthURL),
		OpenstackApplicationCredentialsID:     os.Getenv(testEnvOpenstackApplicationCredentialsID),
		OpenstackApplicationCredentialsSecret: os.Getenv(testEnvOpenstackApplicationCredentialsSecret),
		OpenstackProjectID:                    os.Getenv(testEnvOpenstackProjectID),
		OpenstackRegion:                       os.Getenv(testEnvOpenstackRegion),
		DatacenterName:                        os.Getenv(testEnvOpenstackNodeDC),
		ProjectID:                             os.Getenv(testEnvProjectID),
		ClusterVersion:                        os.Getenv(testEnvK8sVersionOpenstack),
		KubeletVersion:                        os.Getenv(testEnvK8sOlderVersion),
		NodeFlavor:                            os.Getenv(testEnvOpenstackFlavor),
		OSVersion:                             os.Getenv(testEnvOpenstackImage),
		UseFloatingIP:                         "false",
	}

	var config strings.Builder
	if err := nodeDeploymentBasicTemplate.Execute(&config, data); err != nil {
		t.Fatal(err)
	}
	var config2 strings.Builder
	data2 := *data
	data2.KubeletVersion = os.Getenv(testEnvK8sVersionOpenstack)
	data2.OSVersion = os.Getenv(testEnvOpenstackImage2)
	data2.UseFloatingIP = "true"
	data2.DiskSize = 8
	data2.ServerGroupName = makeRandomName() + "-os-servergroup"
	if err := nodeDeploymentBasicTemplate.Execute(&config2, data2); err != nil {
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
		CheckDestroy: testAccCheckMetaKubeNodeDeploymentDestroy,
		Steps: []resource.TestStep{
			{

				Config: config.String(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMetaKubeNodeDeploymentExists(resourceName, &ndepl),
					testAccCheckMetaKubeNodeDeploymentFields(&ndepl, data.NodeFlavor, data.OSVersion, data.KubeletVersion, 2, 0, false),
					resource.TestCheckResourceAttr(resourceName, "name", data.Name),
					resource.TestCheckResourceAttrPtr(resourceName, "name", &ndepl.Name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.replicas", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.labels.%", "4"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.labels.a", "b"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.labels.c", "d"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.cloud.0.openstack.0.flavor", data.NodeFlavor),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.operating_system.0.ubuntu.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.versions.0.kubelet", data.KubeletVersion),
					resource.TestMatchResourceAttr(resourceName, "spec.0.template.0.cloud.0.openstack.0.server_group_id", regexp.MustCompile(`.+`)),
				),
			},
			{
				Config:   config.String(),
				PlanOnly: true,
			},
			{
				Config: config2.String(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testResourceInstanceState(resourceName, func(is *terraform.InstanceState) error {
						// Record IDs to test import
						if is.ID != ndepl.ID {
							return fmt.Errorf("node deployment not updated. Want ID=%v, got %v", ndepl.ID, is.ID)
						}
						return nil
					}),
					testAccCheckMetaKubeNodeDeploymentExists(resourceName, &ndepl),
					testAccCheckMetaKubeNodeDeploymentFields(&ndepl, data2.NodeFlavor, data2.OSVersion, data2.KubeletVersion, 2, 8, false),
					resource.TestCheckResourceAttr(resourceName, "name", data2.Name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.replicas", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.cloud.0.openstack.0.flavor", data2.NodeFlavor),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.cloud.0.openstack.0.use_floating_ip", "true"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.cloud.0.openstack.0.disk_size", "8"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.versions.0.kubelet", data2.KubeletVersion),
					testMatchAndGetResourceAttr(serverGroupResourceName, "id", regexp.MustCompile(`.+`), &sgroupID),
					resource.TestCheckResourceAttrPtr(resourceName, "spec.0.template.0.cloud.0.openstack.0.server_group_id", &sgroupID),
				),
			},
			{
				Config:   config2.String(),
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

type nodeDeploymentBasicData struct {
	OpenstackAuthURL                      string
	OpenstackApplicationCredentialsID     string
	OpenstackApplicationCredentialsSecret string
	OpenstackProjectID                    string
	OpenstackRegion                       string

	Name            string
	DatacenterName  string
	ProjectID       string
	ClusterVersion  string
	KubeletVersion  string
	NodeFlavor      string
	OSVersion       string
	UseFloatingIP   string
	DiskSize        int
	ServerGroupName string
}

var nodeDeploymentBasicTemplate = mustParseTemplate("nodeDeploymentBasic", `
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

	data "openstack_images_image_v2" "image" {
		most_recent = true

		visibility = "public"
		properties = {
		  os_distro  = "ubuntu"
		  os_version = "{{ .OSVersion }}"
		}
	}

	resource "metakube_cluster" "acctest_cluster" {
		name = "{{ .Name }}"
		dc_name = "{{ .DatacenterName }}"
		project_id = "{{ .ProjectID }}"
		spec {
			version = "{{ .ClusterVersion }}"
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

	{{ if .ServerGroupName }}
	resource "openstack_compute_servergroup_v2" "acctest_sg" {
	  name     = "{{ .ServerGroupName }}"
	  policies = ["soft-anti-affinity"]
	}
	{{ end }}

	resource "metakube_node_deployment" "acctest_nd" {
		cluster_id = metakube_cluster.acctest_cluster.id
		name = "{{ .Name }}"
		timeouts {
			create = "40m"
			update = "40m"
			delete = "40m"
		}
		spec {
			replicas = 2
			template {
				labels = {
					"a" = "b"
					"c" = "d"
				}
				cloud {
					openstack {
						flavor = "{{ .NodeFlavor }}"
						image = data.openstack_images_image_v2.image.name
						use_floating_ip = {{ .UseFloatingIP }}
						{{ if .DiskSize }}
						disk_size  = {{ .DiskSize }}
						{{ end }}
						instance_ready_check_period = "10s"
						instance_ready_check_timeout = "4m"
						{{ if .ServerGroupName }}
						server_group_id = openstack_compute_servergroup_v2.acctest_sg.id
						{{ end }}
					}
				}
				operating_system {
					ubuntu {}
				}
				versions {
					kubelet = "{{ .KubeletVersion }}"
				}
			}
		}
	}`)

func testAccCheckMetaKubeNodeDeploymentDestroy(s *terraform.State) error {
	return nil
}

// testMatchAndGetResourceAttr makes a test function that checks whether the given resource's
// key value matches the given regexp just like TestMatchResourceAttr does, then writes the
// actual value into the string the output pointer points to
func testMatchAndGetResourceAttr(name, key string, r *regexp.Regexp, output *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		*output = rs.Primary.Attributes[key]
		if !r.MatchString(*output) {
			return fmt.Errorf(
				"%s: Attribute '%s' didn't match %q, got %#v",
				name,
				key,
				r.String(),
				rs.Primary.Attributes[key])
		}

		return nil
	}
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

		re := regexp.MustCompile(image)
		if !re.Match([]byte(*openstack.Image)) {
			return fmt.Errorf("Image=%s doesn't match %s", *openstack.Image, image)
		}

		if openstack.RootDiskSizeGB != nil && *openstack.RootDiskSizeGB != int64(diskSize) {
			return fmt.Errorf("RootDiskSizeGB=%v, want %d", openstack.RootDiskSizeGB, diskSize)
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

func TestAccMetakubeNodeDeployment_AWS_Basic(t *testing.T) {
	var nodedepl models.NodeDeployment
	testName := makeRandomName() + "-aws-nodedepl"

	projectID := os.Getenv(testEnvProjectID)
	accessKeyID := os.Getenv(testEnvAWSAccessKeyID)
	accessKeySecret := os.Getenv(testAWSSecretAccessKey)
	vpcID := os.Getenv(testEnvAWSVPCID)
	nodeDC := os.Getenv(testEnvAWSNodeDC)
	instanceType := os.Getenv(testEnvAWSInstanceType)
	subnetID := os.Getenv(testEnvAWSSubnetID)
	availabilityZone := os.Getenv(testEnvAWSAvailabilityZone)
	diskSize := os.Getenv(testEnvAWSDiskSize)
	k8sVersion := os.Getenv(testEnvK8sVersionAWS)
	osProject := os.Getenv(testEnvOpenstackProjectName)
	ami := os.Getenv(testEnvAWSAMI)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheckForAWS(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckMetaKubeNodeDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckMetaKubeNodeDeploymentAWSBasic(projectID, testName, osProject, accessKeyID, accessKeySecret, vpcID, nodeDC, instanceType, subnetID, availabilityZone, diskSize, k8sVersion, ami, k8sVersion),
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

func testAccCheckMetaKubeNodeDeploymentAWSBasic(projectID, n, billing, keyID, keySecret, vpcID, nodeDC, instanceType, subnetID, availabilityZone, diskSize, k8sVersion, ami, kubeletVersion string) string {
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
						ami = "%s"
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
	}`, n, nodeDC, projectID, k8sVersion, billing, keyID, keySecret, vpcID, instanceType, diskSize, subnetID, availabilityZone, ami, kubeletVersion)
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
	testName := makeRandomName() + "-nodedepl-valid"

	projectID := os.Getenv(testEnvProjectID)
	osProjectID := os.Getenv(testEnvOpenstackProjectID)
	accessKeyID := os.Getenv(testEnvAWSAccessKeyID)
	accessKeySecret := os.Getenv(testAWSSecretAccessKey)
	vpcID := os.Getenv(testEnvAWSVPCID)
	nodeDC := os.Getenv(testEnvAWSNodeDC)
	k8sVersion17 := os.Getenv(testEnvK8sVersionAWS)
	instanceType := os.Getenv(testEnvAWSInstanceType)
	subnetID := os.Getenv(testEnvAWSSubnetID)
	availabilityZone := os.Getenv(testEnvAWSAvailabilityZone)
	diskSize := os.Getenv(testEnvAWSDiskSize)
	ami := os.Getenv(testEnvAWSAMI)
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
				Config: testAccCheckMetaKubeNodeDeploymentBasicValidation(testName, projectID, osProjectID, accessKeyID, accessKeySecret, vpcID, nodeDC, instanceType, subnetID, availabilityZone, ami, diskSize, k8sVersion17, k8sVersion17),
			},
			{
				Config:      testAccCheckMetaKubeNodeDeploymentBasicValidation(testName, projectID, osProjectID, accessKeyID, accessKeySecret, vpcID, nodeDC, instanceType, subnetID, availabilityZone, ami, diskSize, k8sVersion17, unavailableVersion),
				ExpectError: regexp.MustCompile(fmt.Sprintf(`unknown version for node deployment %s, available versions`, unavailableVersion)),
			},
			{
				Config:      testAccCheckMetaKubeNodeDeploymentTypeValidation(testName, projectID, osProjectID, accessKeyID, accessKeySecret, vpcID, nodeDC, k8sVersion17, k8sVersion17),
				ExpectError: regexp.MustCompile(`provider for node deployment must \(.*\) match cluster provider \(.*\)`),
			},
			{
				Config:      testAccCheckMetaKubeNodeDeploymentBasicValidation(testName, projectID, osProjectID, accessKeyID, accessKeySecret, vpcID, nodeDC, instanceType, subnetID, availabilityZone, ami, diskSize, k8sVersion17, bigVersion),
				ExpectError: regexp.MustCompile(`cannot be greater than cluster version`),
			},
		},
	})
}

func testAccCheckMetaKubeNodeDeploymentBasicValidation(n, projectID, billing, keyID, keySecret, vpcID, nodeDC, instanceType, subnetID, availabilityZone, ami, diskSize, k8sVersion, kubeletVersion string) string {
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
						ami = "%s"
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
	}`, n, nodeDC, projectID, k8sVersion, billing, keyID, keySecret, vpcID, n, instanceType, diskSize, subnetID, availabilityZone, ami, kubeletVersion)
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
