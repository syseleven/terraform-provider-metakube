package resource_node_deployment_test

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
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

func TestAccMetakubeNodeDeployment_Openstack_Basic(t *testing.T) {
	var ndepl models.NodeDeployment
	var sgroupID string
	clusterResourceName := "metakube_cluster.acctest_cluster"
	distUpgradeOnBootPath := tfjsonpath.New("spec").AtSliceIndex(0).AtMapKey("template").AtSliceIndex(0).AtMapKey("operating_system").AtSliceIndex(0).AtMapKey("ubuntu").AtSliceIndex(0).AtMapKey("dist_upgrade_on_boot")
	tagsPath := tfjsonpath.New("spec").AtSliceIndex(0).AtMapKey("template").AtSliceIndex(0).AtMapKey("cloud").AtSliceIndex(0).AtMapKey("openstack").AtSliceIndex(0).AtMapKey("tags")
	resourceName := "metakube_node_deployment.acctest_nd"
	serverGroupResourceName := "openstack_compute_servergroup_v2.acctest_sg"

	data := &nodeDeploymentBasicData{
		Name:                                  testutil.MakeRandomName() + "-os-nodedepl",
		OpenstackAuthURL:                      os.Getenv(common.TestEnvOpenstackAuthURL),
		OpenstackApplicationCredentialsID:     common.GetSACredentialId(),
		OpenstackApplicationCredentialsSecret: os.Getenv(common.TestEnvServiceAccountCredential),
		OpenstackProjectID:                    os.Getenv(common.TestEnvProjectID),
		OpenstackRegion:                       os.Getenv(common.TestEnvOpenstackRegion),
		DatacenterName:                        os.Getenv(common.TestEnvOpenstackNodeDC),
		ProjectID:                             os.Getenv(common.TestEnvProjectID),
		ClusterVersion:                        os.Getenv(common.TestEnvK8sVersionOpenstack),
		KubeletVersion:                        os.Getenv(common.TestEnvK8sOlderVersion),
		NodeFlavor:                            os.Getenv(common.TestEnvOpenstackFlavor),
		OSVersion:                             os.Getenv(common.TestEnvOpenstackImage),
		Replicas:                              2,
		LabelKey:                              "a",
		LabelValue:                            "b",
		SecondLabelKey:                        "c",
		SecondLabelValue:                      "d",
		UseFloatingIP:                         "false",
		InstanceReadyCheckPeriod:              "10s",
		InstanceReadyCheckTimeout:             "4m",
		NodeAnnotationKey:                     "a",
		NodeAnnotationValue:                   "b",
		MachineAnnotationKey:                  "c",
		MachineAnnotationValue:                "d",
		UserTagKey:                            "user-tag",
		UserTagValue:                          "kept",
		DistUpgradeOnBoot:                     true,
	}

	var config strings.Builder
	if err := nodeDeploymentBasicTemplate.Execute(&config, data); err != nil {
		t.Fatal(err)
	}
	var config2 strings.Builder
	data2 := *data
	data2.KubeletVersion = os.Getenv(common.TestEnvK8sVersionOpenstack)
	data2.OSVersion = os.Getenv(common.TestEnvOpenstackImage2)
	data2.Replicas = 1
	data2.LabelKey = "e"
	data2.LabelValue = "f"
	data2.SecondLabelKey = "g"
	data2.SecondLabelValue = "h"
	data2.UseFloatingIP = "true"
	data2.DiskSize = 8
	data2.InstanceReadyCheckPeriod = "15s"
	data2.InstanceReadyCheckTimeout = "5m"
	data2.ServerGroupName = testutil.MakeRandomName() + "-os-servergroup"
	data2.UserTagKey = "updated-user-tag"
	data2.UserTagValue = "changed"
	data2.NodeAnnotationKey = "i"
	data2.NodeAnnotationValue = "j"
	data2.MachineAnnotationKey = "k"
	data2.MachineAnnotationValue = "l"
	data2.DistUpgradeOnBoot = false
	if err := nodeDeploymentBasicTemplate.Execute(&config2, &data2); err != nil {
		t.Fatal(err)
	}

	var config3 strings.Builder
	data3 := *data
	if err := nodeDeploymentBasicTemplate.Execute(&config3, &data3); err != nil {
		t.Fatal(err)
	}

	var invalidTagConfig strings.Builder
	invalidTagData := *data
	invalidTagData.ReservedTagKey = "system-cluster"
	invalidTagData.UserTagKey = ""
	invalidTagData.UserTagValue = ""
	invalidTagData.MinimalConfig = true
	if err := nodeDeploymentBasicTemplate.Execute(&invalidTagConfig, &invalidTagData); err != nil {
		t.Fatal(err)
	}

	var invalidLabelConfig strings.Builder
	invalidLabelData := *data
	invalidLabelData.ReservedLabelKey = "system/project"
	invalidLabelData.UserTagKey = ""
	invalidLabelData.UserTagValue = ""
	invalidLabelData.MinimalConfig = true
	if err := nodeDeploymentBasicTemplate.Execute(&invalidLabelConfig, &invalidLabelData); err != nil {
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
		CheckDestroy: testAccCheckMetaKubeNodeDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config:      invalidTagConfig.String(),
				ExpectError: regexp.MustCompile(`Reserved key |reserved pattern`),
			},
			{
				Config:      invalidLabelConfig.String(),
				ExpectError: regexp.MustCompile(`Reserved key |reserved pattern`),
			},
			{
				Config: config.String(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMetaKubeNodeDeploymentExists(resourceName, &ndepl),
					testAccCheckMetaKubeNodeDeploymentFields(&ndepl, data.NodeFlavor, data.OSVersion, data.KubeletVersion, data.Replicas, data.DiskSize, data.DistUpgradeOnBoot),
					testAccCheckMetaKubeNodeDeploymentOpenstackUserTags(resourceName, data.UserTagKey, data.UserTagValue),
					testAccCheckMetaKubeNodeDeploymentOpenstackAPIHasReservedPrefixTags(&ndepl),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.all_labels.%", "4"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.all_labels."+data.LabelKey, data.LabelValue),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.all_labels."+data.SecondLabelKey, data.SecondLabelValue),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.cloud.0.openstack.0.tags.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.cloud.0.openstack.0.tags."+data.UserTagKey, data.UserTagValue),
					resource.TestCheckNoResourceAttr(resourceName, "spec.0.template.0.cloud.0.openstack.0.tags.metakube-cluster"),
					resource.TestCheckNoResourceAttr(resourceName, "spec.0.template.0.cloud.0.openstack.0.tags.system-cluster"),
					resource.TestCheckNoResourceAttr(resourceName, "spec.0.template.0.cloud.0.openstack.0.tags.system-project"),
					resource.TestMatchResourceAttr(resourceName, "spec.0.template.0.cloud.0.openstack.0.server_group_id", regexp.MustCompile(`.+`)),
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(clusterResourceName, plancheck.ResourceActionCreate),
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionCreate),
						plancheck.ExpectKnownValue(resourceName, distUpgradeOnBootPath, knownvalue.Bool(true)),
					},
					PostApplyPreRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
						plancheck.ExpectKnownValue(resourceName, distUpgradeOnBootPath, knownvalue.Bool(true)),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
						plancheck.ExpectKnownValue(resourceName, distUpgradeOnBootPath, knownvalue.Bool(true)),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("name"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("spec").AtSliceIndex(0).AtMapKey("replicas"),
						knownvalue.Int64Exact(int64(data.Replicas))),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("spec").AtSliceIndex(0).AtMapKey("template").AtSliceIndex(0).AtMapKey("labels"),
						knownvalue.MapExact(map[string]knownvalue.Check{
							data.LabelKey:       knownvalue.StringExact(data.LabelValue),
							data.SecondLabelKey: knownvalue.StringExact(data.SecondLabelValue),
						})),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("spec").AtSliceIndex(0).AtMapKey("template").AtSliceIndex(0).AtMapKey("cloud").AtSliceIndex(0).AtMapKey("openstack").AtSliceIndex(0).AtMapKey("flavor"),
						knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("spec").AtSliceIndex(0).AtMapKey("template").AtSliceIndex(0).AtMapKey("cloud").AtSliceIndex(0).AtMapKey("openstack").AtSliceIndex(0).AtMapKey("image"),
						knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("spec").AtSliceIndex(0).AtMapKey("template").AtSliceIndex(0).AtMapKey("cloud").AtSliceIndex(0).AtMapKey("openstack").AtSliceIndex(0).AtMapKey("use_floating_ip"),
						knownvalue.Bool(false)),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("spec").AtSliceIndex(0).AtMapKey("template").AtSliceIndex(0).AtMapKey("cloud").AtSliceIndex(0).AtMapKey("openstack").AtSliceIndex(0).AtMapKey("instance_ready_check_period"),
						knownvalue.StringExact(data.InstanceReadyCheckPeriod)),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("spec").AtSliceIndex(0).AtMapKey("template").AtSliceIndex(0).AtMapKey("cloud").AtSliceIndex(0).AtMapKey("openstack").AtSliceIndex(0).AtMapKey("instance_ready_check_timeout"),
						knownvalue.StringExact(data.InstanceReadyCheckTimeout)),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("spec").AtSliceIndex(0).AtMapKey("template").AtSliceIndex(0).AtMapKey("operating_system").AtSliceIndex(0).AtMapKey("ubuntu"),
						knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("spec").AtSliceIndex(0).AtMapKey("template").AtSliceIndex(0).AtMapKey("operating_system").AtSliceIndex(0).AtMapKey("ubuntu").AtSliceIndex(0).AtMapKey("dist_upgrade_on_boot"),
						knownvalue.Bool(data.DistUpgradeOnBoot)),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("spec").AtSliceIndex(0).AtMapKey("template").AtSliceIndex(0).AtMapKey("node_annotations"),
						knownvalue.MapExact(map[string]knownvalue.Check{
							data.NodeAnnotationKey: knownvalue.StringExact(data.NodeAnnotationValue),
						})),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("spec").AtSliceIndex(0).AtMapKey("template").AtSliceIndex(0).AtMapKey("machine_annotations"),
						knownvalue.MapExact(map[string]knownvalue.Check{
							"machines.metakube.syseleven.de/user-data-plugin": knownvalue.StringExact("ubuntu-sysext"),
							data.MachineAnnotationKey:                         knownvalue.StringExact(data.MachineAnnotationValue),
						})),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("spec").AtSliceIndex(0).AtMapKey("template").AtSliceIndex(0).AtMapKey("versions").AtSliceIndex(0).AtMapKey("kubelet"),
						knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("spec").AtSliceIndex(0).AtMapKey("template").AtSliceIndex(0).AtMapKey("all_labels"),
						knownvalue.MapPartial(map[string]knownvalue.Check{
							data.LabelKey:       knownvalue.StringExact(data.LabelValue),
							data.SecondLabelKey: knownvalue.StringExact(data.SecondLabelValue),
						})),
					statecheck.ExpectKnownValue(resourceName, tagsPath,
						knownvalue.MapExact(map[string]knownvalue.Check{
							data.UserTagKey: knownvalue.StringExact(data.UserTagValue),
						})),
				},
			},
			{
				Config:   config.String(),
				PlanOnly: true,
			},
			{
				Config: config2.String(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(clusterResourceName, plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction(serverGroupResourceName, plancheck.ResourceActionCreate),
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
						plancheck.ExpectKnownValue(resourceName, distUpgradeOnBootPath, knownvalue.Bool(false)),
					},
					PostApplyPreRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
						plancheck.ExpectKnownValue(resourceName, distUpgradeOnBootPath, knownvalue.Bool(false)),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
						plancheck.ExpectKnownValue(resourceName, distUpgradeOnBootPath, knownvalue.Bool(false)),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.TestResourceInstanceState(resourceName, func(is *terraform.InstanceState) error {
						// Record IDs to test import
						if is.ID != ndepl.ID {
							return fmt.Errorf("node deployment not updated. Want ID=%v, got %v", ndepl.ID, is.ID)
						}
						return nil
					}),
					testAccCheckMetaKubeNodeDeploymentExists(resourceName, &ndepl),
					testAccCheckMetaKubeNodeDeploymentFields(&ndepl, data2.NodeFlavor, data2.OSVersion, data2.KubeletVersion, data2.Replicas, data2.DiskSize, data2.DistUpgradeOnBoot),
					testAccCheckMetaKubeNodeDeploymentOpenstackUserTags(resourceName, data2.UserTagKey, data2.UserTagValue),
					testAccCheckMetaKubeNodeDeploymentOpenstackAPIHasReservedPrefixTags(&ndepl),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.cloud.0.openstack.0.tags.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.cloud.0.openstack.0.tags."+data2.UserTagKey, data2.UserTagValue),
					resource.TestCheckNoResourceAttr(resourceName, "spec.0.template.0.cloud.0.openstack.0.tags."+data.UserTagKey),
					resource.TestCheckNoResourceAttr(resourceName, "spec.0.template.0.cloud.0.openstack.0.tags.metakube-cluster"),
					resource.TestCheckNoResourceAttr(resourceName, "spec.0.template.0.cloud.0.openstack.0.tags.system-cluster"),
					resource.TestCheckNoResourceAttr(resourceName, "spec.0.template.0.cloud.0.openstack.0.tags.system-project"),
					testMatchAndGetResourceAttr(serverGroupResourceName, "id", regexp.MustCompile(`.+`), &sgroupID),
					resource.TestCheckResourceAttrPtr(resourceName, "spec.0.template.0.cloud.0.openstack.0.server_group_id", &sgroupID),
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("name"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("spec").AtSliceIndex(0).AtMapKey("replicas"),
						knownvalue.Int64Exact(int64(data2.Replicas))),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("spec").AtSliceIndex(0).AtMapKey("template").AtSliceIndex(0).AtMapKey("labels"),
						knownvalue.MapExact(map[string]knownvalue.Check{
							data2.LabelKey:       knownvalue.StringExact(data2.LabelValue),
							data2.SecondLabelKey: knownvalue.StringExact(data2.SecondLabelValue),
						})),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("spec").AtSliceIndex(0).AtMapKey("template").AtSliceIndex(0).AtMapKey("cloud").AtSliceIndex(0).AtMapKey("openstack").AtSliceIndex(0).AtMapKey("flavor"),
						knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("spec").AtSliceIndex(0).AtMapKey("template").AtSliceIndex(0).AtMapKey("cloud").AtSliceIndex(0).AtMapKey("openstack").AtSliceIndex(0).AtMapKey("image"),
						knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("spec").AtSliceIndex(0).AtMapKey("template").AtSliceIndex(0).AtMapKey("cloud").AtSliceIndex(0).AtMapKey("openstack").AtSliceIndex(0).AtMapKey("use_floating_ip"),
						knownvalue.Bool(true)),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("spec").AtSliceIndex(0).AtMapKey("template").AtSliceIndex(0).AtMapKey("cloud").AtSliceIndex(0).AtMapKey("openstack").AtSliceIndex(0).AtMapKey("disk_size"),
						knownvalue.Int64Exact(8)),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("spec").AtSliceIndex(0).AtMapKey("template").AtSliceIndex(0).AtMapKey("cloud").AtSliceIndex(0).AtMapKey("openstack").AtSliceIndex(0).AtMapKey("instance_ready_check_period"),
						knownvalue.StringExact(data2.InstanceReadyCheckPeriod)),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("spec").AtSliceIndex(0).AtMapKey("template").AtSliceIndex(0).AtMapKey("cloud").AtSliceIndex(0).AtMapKey("openstack").AtSliceIndex(0).AtMapKey("instance_ready_check_timeout"),
						knownvalue.StringExact(data2.InstanceReadyCheckTimeout)),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("spec").AtSliceIndex(0).AtMapKey("template").AtSliceIndex(0).AtMapKey("cloud").AtSliceIndex(0).AtMapKey("openstack").AtSliceIndex(0).AtMapKey("server_group_id"),
						knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("spec").AtSliceIndex(0).AtMapKey("template").AtSliceIndex(0).AtMapKey("operating_system").AtSliceIndex(0).AtMapKey("ubuntu"),
						knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("spec").AtSliceIndex(0).AtMapKey("template").AtSliceIndex(0).AtMapKey("operating_system").AtSliceIndex(0).AtMapKey("ubuntu").AtSliceIndex(0).AtMapKey("dist_upgrade_on_boot"),
						knownvalue.Bool(data2.DistUpgradeOnBoot)),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("spec").AtSliceIndex(0).AtMapKey("template").AtSliceIndex(0).AtMapKey("node_annotations"),
						knownvalue.MapExact(map[string]knownvalue.Check{
							data2.NodeAnnotationKey: knownvalue.StringExact(data2.NodeAnnotationValue),
						})),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("spec").AtSliceIndex(0).AtMapKey("template").AtSliceIndex(0).AtMapKey("machine_annotations"),
						knownvalue.MapExact(map[string]knownvalue.Check{
							"machines.metakube.syseleven.de/user-data-plugin": knownvalue.StringExact("ubuntu-sysext"),
							data2.MachineAnnotationKey:                        knownvalue.StringExact(data2.MachineAnnotationValue),
						})),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("spec").AtSliceIndex(0).AtMapKey("template").AtSliceIndex(0).AtMapKey("versions").AtSliceIndex(0).AtMapKey("kubelet"),
						knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("spec").AtSliceIndex(0).AtMapKey("template").AtSliceIndex(0).AtMapKey("all_labels"),
						knownvalue.MapPartial(map[string]knownvalue.Check{
							data2.LabelKey:       knownvalue.StringExact(data2.LabelValue),
							data2.SecondLabelKey: knownvalue.StringExact(data2.SecondLabelValue),
						})),
					statecheck.ExpectKnownValue(resourceName, tagsPath,
						knownvalue.MapExact(map[string]knownvalue.Check{
							data2.UserTagKey: knownvalue.StringExact(data2.UserTagValue),
						})),
				},
			},
			{
				Config:   config2.String(),
				PlanOnly: true,
			},
			{
				Config: config3.String(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(clusterResourceName, plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction(serverGroupResourceName, plancheck.ResourceActionDestroy),
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
						plancheck.ExpectKnownValue(resourceName, distUpgradeOnBootPath, knownvalue.Bool(true)),
					},
					PostApplyPreRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
						plancheck.ExpectKnownValue(resourceName, distUpgradeOnBootPath, knownvalue.Bool(true)),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
						plancheck.ExpectKnownValue(resourceName, distUpgradeOnBootPath, knownvalue.Bool(true)),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMetaKubeNodeDeploymentExists(resourceName, &ndepl),
					testAccCheckMetaKubeNodeDeploymentFields(&ndepl, data3.NodeFlavor, data3.OSVersion, data3.KubeletVersion, data3.Replicas, data3.DiskSize, data3.DistUpgradeOnBoot),
					testAccCheckMetaKubeNodeDeploymentOpenstackUserTags(resourceName, data3.UserTagKey, data3.UserTagValue),
					testAccCheckMetaKubeNodeDeploymentOpenstackAPIHasReservedPrefixTags(&ndepl),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.all_labels.%", "4"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.all_labels."+data3.LabelKey, data3.LabelValue),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.all_labels."+data3.SecondLabelKey, data3.SecondLabelValue),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.cloud.0.openstack.0.tags.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.cloud.0.openstack.0.tags."+data3.UserTagKey, data3.UserTagValue),
					resource.TestCheckNoResourceAttr(resourceName, "spec.0.template.0.cloud.0.openstack.0.tags."+data2.UserTagKey),
					resource.TestCheckNoResourceAttr(resourceName, "spec.0.template.0.cloud.0.openstack.0.tags.metakube-cluster"),
					resource.TestCheckNoResourceAttr(resourceName, "spec.0.template.0.cloud.0.openstack.0.tags.system-cluster"),
					resource.TestCheckNoResourceAttr(resourceName, "spec.0.template.0.cloud.0.openstack.0.tags.system-project"),
					resource.TestMatchResourceAttr(resourceName, "spec.0.template.0.cloud.0.openstack.0.server_group_id", regexp.MustCompile(`.+`)),
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("name"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("spec").AtSliceIndex(0).AtMapKey("replicas"),
						knownvalue.Int64Exact(int64(data3.Replicas))),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("spec").AtSliceIndex(0).AtMapKey("template").AtSliceIndex(0).AtMapKey("labels"),
						knownvalue.MapExact(map[string]knownvalue.Check{
							data3.LabelKey:       knownvalue.StringExact(data3.LabelValue),
							data3.SecondLabelKey: knownvalue.StringExact(data3.SecondLabelValue),
						})),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("spec").AtSliceIndex(0).AtMapKey("template").AtSliceIndex(0).AtMapKey("cloud").AtSliceIndex(0).AtMapKey("openstack").AtSliceIndex(0).AtMapKey("flavor"),
						knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("spec").AtSliceIndex(0).AtMapKey("template").AtSliceIndex(0).AtMapKey("cloud").AtSliceIndex(0).AtMapKey("openstack").AtSliceIndex(0).AtMapKey("image"),
						knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("spec").AtSliceIndex(0).AtMapKey("template").AtSliceIndex(0).AtMapKey("cloud").AtSliceIndex(0).AtMapKey("openstack").AtSliceIndex(0).AtMapKey("use_floating_ip"),
						knownvalue.Bool(false)),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("spec").AtSliceIndex(0).AtMapKey("template").AtSliceIndex(0).AtMapKey("cloud").AtSliceIndex(0).AtMapKey("openstack").AtSliceIndex(0).AtMapKey("instance_ready_check_period"),
						knownvalue.StringExact(data3.InstanceReadyCheckPeriod)),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("spec").AtSliceIndex(0).AtMapKey("template").AtSliceIndex(0).AtMapKey("cloud").AtSliceIndex(0).AtMapKey("openstack").AtSliceIndex(0).AtMapKey("instance_ready_check_timeout"),
						knownvalue.StringExact(data3.InstanceReadyCheckTimeout)),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("spec").AtSliceIndex(0).AtMapKey("template").AtSliceIndex(0).AtMapKey("operating_system").AtSliceIndex(0).AtMapKey("ubuntu"),
						knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("spec").AtSliceIndex(0).AtMapKey("template").AtSliceIndex(0).AtMapKey("operating_system").AtSliceIndex(0).AtMapKey("ubuntu").AtSliceIndex(0).AtMapKey("dist_upgrade_on_boot"),
						knownvalue.Bool(data3.DistUpgradeOnBoot)),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("spec").AtSliceIndex(0).AtMapKey("template").AtSliceIndex(0).AtMapKey("node_annotations"),
						knownvalue.MapExact(map[string]knownvalue.Check{
							data3.NodeAnnotationKey: knownvalue.StringExact(data3.NodeAnnotationValue),
						})),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("spec").AtSliceIndex(0).AtMapKey("template").AtSliceIndex(0).AtMapKey("machine_annotations"),
						knownvalue.MapExact(map[string]knownvalue.Check{
							"machines.metakube.syseleven.de/user-data-plugin": knownvalue.StringExact("ubuntu-sysext"),
							data3.MachineAnnotationKey:                        knownvalue.StringExact(data3.MachineAnnotationValue),
						})),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("spec").AtSliceIndex(0).AtMapKey("template").AtSliceIndex(0).AtMapKey("versions").AtSliceIndex(0).AtMapKey("kubelet"),
						knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("spec").AtSliceIndex(0).AtMapKey("template").AtSliceIndex(0).AtMapKey("all_labels"),
						knownvalue.MapPartial(map[string]knownvalue.Check{
							data3.LabelKey:       knownvalue.StringExact(data3.LabelValue),
							data3.SecondLabelKey: knownvalue.StringExact(data3.SecondLabelValue),
						})),
					statecheck.ExpectKnownValue(resourceName, tagsPath,
						knownvalue.MapExact(map[string]knownvalue.Check{
							data3.UserTagKey: knownvalue.StringExact(data3.UserTagValue),
						})),
				},
			},
			{
				Config:   config3.String(),
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

	Name                      string
	DatacenterName            string
	ProjectID                 string
	ClusterVersion            string
	KubeletVersion            string
	NodeFlavor                string
	OSVersion                 string
	Replicas                  int
	LabelKey                  string
	LabelValue                string
	SecondLabelKey            string
	SecondLabelValue          string
	UseFloatingIP             string
	DiskSize                  int
	InstanceReadyCheckPeriod  string
	InstanceReadyCheckTimeout string
	ServerGroupName           string
	UserTagKey                string
	UserTagValue              string
	NodeAnnotationKey         string
	NodeAnnotationValue       string
	MachineAnnotationKey      string
	MachineAnnotationValue    string
	ReservedTagKey            string
	ReservedLabelKey          string
	MinimalConfig             bool

	DistUpgradeOnBoot bool
}

var nodeDeploymentBasicTemplate = testutil.MustParseTemplate("nodeDeploymentBasic", `
	{{ if not .MinimalConfig }}
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
	{{ end }}

	resource "metakube_cluster" "acctest_cluster" {
		name = "{{ .Name }}"
		dc_name = "{{ .DatacenterName }}"
		project_id = "{{ .ProjectID }}"
	timeouts {
		create = "40m"
		update = "40m"
		delete = "40m"
	}
		spec = {
			version = "{{ .ClusterVersion }}"
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

	{{ if .ServerGroupName }}
	resource "openstack_compute_servergroup_v2" "acctest_sg" {
	  name     = "{{ .ServerGroupName }}"
	  policies = ["soft-anti-affinity"]
	}
	{{ end }}

	resource "metakube_node_deployment" "acctest_nd" {
		cluster_id = metakube_cluster.acctest_cluster.id
		project_id = "{{ .ProjectID }}"
		name = "{{ .Name }}"
		timeouts {
			create = "40m"
			update = "40m"
			delete = "40m"
		}
		spec {
			replicas = {{ if .MinimalConfig }}1{{ else }}{{ .Replicas }}{{ end }}
			template {
				labels = {
					"{{ .LabelKey }}" = "{{ .LabelValue }}"
					"{{ .SecondLabelKey }}" = "{{ .SecondLabelValue }}"
					{{ if .ReservedLabelKey }}
					"{{ .ReservedLabelKey }}" = "forbidden"
					{{ end }}
				}
				cloud {
					openstack {
						flavor = "{{ .NodeFlavor }}"
						{{ if .MinimalConfig }}
						image = "Ubuntu {{ .OSVersion }}"
						{{ else }}
						image = data.openstack_images_image_v2.image.name
						use_floating_ip = {{ .UseFloatingIP }}
						{{ if .DiskSize }}
						disk_size  = {{ .DiskSize }}
						{{ end }}
						instance_ready_check_period = "{{ .InstanceReadyCheckPeriod }}"
						instance_ready_check_timeout = "{{ .InstanceReadyCheckTimeout }}"
						{{ if .ServerGroupName }}
						server_group_id = openstack_compute_servergroup_v2.acctest_sg.id
						{{ end }}
						{{ end }}
						{{ if or .UserTagKey .ReservedTagKey }}
						tags = {
							{{ if .ReservedTagKey }}
							"{{ .ReservedTagKey }}" = "forbidden"
							{{ else }}
							"{{ .UserTagKey }}" = "{{ .UserTagValue }}"
							{{ end }}
						}
						{{ end }}
					}
				}
				operating_system {
					ubuntu {
						dist_upgrade_on_boot = {{ .DistUpgradeOnBoot }}
					}
				}
				{{ if not .MinimalConfig }}
				node_annotations = {
					"{{ .NodeAnnotationKey }}" = "{{ .NodeAnnotationValue }}"
				}
				machine_annotations = {
					"machines.metakube.syseleven.de/user-data-plugin" = "ubuntu-sysext"
					"{{ .MachineAnnotationKey }}" = "{{ .MachineAnnotationValue }}"
				}
				{{ end }}
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

func testAccCheckMetaKubeNodeDeploymentOpenstackUserTags(resourceName, key, value string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}

		prefix := "spec.0.template.0.cloud.0.openstack.0.tags."
		got, ok := rs.Primary.Attributes[prefix+key]
		if !ok {
			return fmt.Errorf("expected tag %q in state, got attributes: %#v", key, rs.Primary.Attributes)
		}
		if got != value {
			return fmt.Errorf("tag %q: got %q, want %q", key, got, value)
		}

		for attrKey := range rs.Primary.Attributes {
			if !strings.HasPrefix(attrKey, prefix) {
				continue
			}
			tagKey := strings.TrimPrefix(attrKey, prefix)
			if tagKey == "%" {
				continue
			}
			if common.MetakubeResourceSystemLabelOrTag(tagKey) {
				return fmt.Errorf("reserved-prefix tag %q must not appear in terraform state", tagKey)
			}
		}

		return nil
	}
}

func testAccCheckMetaKubeNodeDeploymentOpenstackAPIHasReservedPrefixTags(rec *models.NodeDeployment) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if rec == nil || rec.Spec == nil || rec.Spec.Template == nil || rec.Spec.Template.Cloud == nil || rec.Spec.Template.Cloud.Openstack == nil {
			return fmt.Errorf("no openstack cloud spec in API response")
		}

		tags := rec.Spec.Template.Cloud.Openstack.Tags
		if len(tags) == 0 {
			return fmt.Errorf("expected API to return tags including reserved-prefix keys, got none")
		}

		for key := range tags {
			if common.MetakubeResourceSystemLabelOrTag(key) {
				return nil
			}
		}

		return fmt.Errorf("expected API tags to include at least one reserved-prefix key, got %#v", tags)
	}
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

		k, err := testutil.GetTestClient()
		if err != nil {
			return fmt.Errorf("failed to get test client: %v", err)
		}

		p := project.NewGetMachineDeploymentParams().
			WithProjectID(rs.Primary.Attributes["project_id"]).
			WithClusterID(rs.Primary.Attributes["cluster_id"]).
			WithMachineDeploymentID(rs.Primary.ID)
		r, err := k.Client.Project.GetMachineDeployment(p, k.Auth)
		if err != nil {
			return fmt.Errorf("GetNodeDeployment: %v", err)
		}
		*rec = *r.Payload

		return nil
	}
}
