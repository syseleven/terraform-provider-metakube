package resource_maintenance_cronjob_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
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

func TestAccMetakubeCluster_MaintenanceCronJob_Basic(t *testing.T) {
	t.Parallel()
	var maintenanceCronJob models.MaintenanceCronJob

	resourceName := "metakube_maintenance_cron_job.acctest"
	params := &testAccCheckMetaKubeMaintenanceCronJobBasicParams{
		ClusterName:                          testutil.MakeRandomName() + "-maint-cron-job",
		DatacenterName:                       os.Getenv(common.TestEnvOpenstackNodeDC),
		ProjectID:                            os.Getenv(common.TestEnvProjectID),
		Version:                              os.Getenv(common.TestEnvK8sVersionOpenstack),
		OpenstackApplicationCredentialID:     common.GetSACredentialId(),
		OpenstackApplicationCredentialSecret: os.Getenv(common.TestEnvServiceAccountCredential),

		MaintenanceCronJobName:     testutil.RandomName("test-maintenancecronjob", 5),
		MaintenanceJobTemplateName: testutil.RandomName("test-maintenancecronjob-template", 5),
		MaintenanceJobType:         "kubernetesPatchUpdate",
		Schedule:                   "5 4 * * *",
		UpdatedSchedule:            "0 2 * * *",
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testutil.TestAccPreCheckForOpenstack(t)
		},
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckMetaKubeMaintenanceCronJobDestroy,
		Steps: []resource.TestStep{
			// Create
			{
				Config: testAccCheckMetaKubeMaintenanceCronJobBasicConfig(t, params),
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
					testAccCheckMetaKubeMaintenanceCronJobExists(&maintenanceCronJob),
					testAccCheckMetaKubeMaintenanceCronJobFields(&maintenanceCronJob, params.MaintenanceCronJobName, params.Schedule, params.MaintenanceJobType),
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("project_id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("name"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("creation_timestamp"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("deletion_timestamp"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName,
						tfjsonpath.New("spec").AtSliceIndex(0).AtMapKey("maintenance_job_template").AtSliceIndex(0).AtMapKey("rollback"),
						knownvalue.Bool(false),
					),
				},
			},
		// Update
		{
			Config: testAccCheckMetaKubeMaintenanceCronJobUpdateConfig(t, params),
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
					testAccCheckMetaKubeMaintenanceCronJobExists(&maintenanceCronJob),
					testAccCheckMetaKubeMaintenanceCronJobFields(&maintenanceCronJob, params.MaintenanceCronJobName, params.UpdatedSchedule, params.MaintenanceJobType),
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("project_id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("name"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("creation_timestamp"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("deletion_timestamp"), knownvalue.NotNull()),
				statecheck.ExpectKnownValue(resourceName,
					tfjsonpath.New("spec").AtSliceIndex(0).AtMapKey("maintenance_job_template").AtSliceIndex(0).AtMapKey("rollback"),
					knownvalue.Bool(false),
				),
				},
			},
			// Import
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"timeouts"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					for _, rs := range s.RootModule().Resources {
						if rs.Type == "metakube_maintenance_cron_job" {
							return fmt.Sprintf("%s:%s:%s", rs.Primary.Attributes["project_id"], rs.Primary.Attributes["cluster_id"], rs.Primary.ID), nil
						}
					}
					return "", fmt.Errorf("not found")
				},
			},

		},
	})
}

type testAccCheckMetaKubeMaintenanceCronJobBasicParams struct {
	ClusterName                          string
	DatacenterName                       string
	ProjectID                            string
	Version                              string
	OpenstackApplicationCredentialID     string
	OpenstackApplicationCredentialSecret string

	MaintenanceCronJobName     string
	MaintenanceJobTemplateName string
	MaintenanceJobType         string
	Schedule                   string
	UpdatedSchedule            string
}

func testAccCheckMetaKubeMaintenanceCronJobBasicConfig(t *testing.T, params *testAccCheckMetaKubeMaintenanceCronJobBasicParams) string {
	t.Helper()

	var result strings.Builder
	err := testutil.MustParseTemplate("metakube maintenance cron job test template", `
	resource "metakube_cluster" "acctest" {
		name = "{{ .ClusterName }}"
		dc_name = "{{ .DatacenterName }}"
		project_id = "{{ .ProjectID }}"
	
		spec {
			version = "{{ .Version }}"
			cloud {
				openstack {
					application_credentials {
						id = "{{ .OpenstackApplicationCredentialID }}"
						secret ="{{ .OpenstackApplicationCredentialSecret }}"
					}
				}
			}
		}
	}

	resource "metakube_maintenance_cron_job" "acctest" {
		cluster_id = metakube_cluster.acctest.id
		name = "{{ .MaintenanceCronJobName }}"
		project_id = "{{ .ProjectID }}"

		spec {
			schedule		= "{{ .Schedule }}"
			maintenance_job_template {
				rollback 	= false
				type		= "{{ .MaintenanceJobType }}"
			}
		}
	}
`).Execute(&result, params)
	if err != nil {
		t.Fatal(err)
	}
	return result.String()
}

func testAccCheckMetaKubeMaintenanceCronJobUpdateConfig(t *testing.T, params *testAccCheckMetaKubeMaintenanceCronJobBasicParams) string {
	t.Helper()

	var result strings.Builder
	err := testutil.MustParseTemplate("metakube maintenance cron job update template", `
	resource "metakube_cluster" "acctest" {
		name = "{{ .ClusterName }}"
		dc_name = "{{ .DatacenterName }}"
		project_id = "{{ .ProjectID }}"
	
		spec {
			version = "{{ .Version }}"
			cloud {
				openstack {
					application_credentials {
						id = "{{ .OpenstackApplicationCredentialID }}"
						secret ="{{ .OpenstackApplicationCredentialSecret }}"
					}
				}
			}
		}
	}

	resource "metakube_maintenance_cron_job" "acctest" {
		cluster_id = metakube_cluster.acctest.id
		name = "{{ .MaintenanceCronJobName }}"
		project_id = "{{ .ProjectID }}"

		spec {
			schedule		= "{{ .UpdatedSchedule }}"
			maintenance_job_template {
				rollback 	= true
				type		= "{{ .MaintenanceJobType }}"
			}
		}
	}
`).Execute(&result, params)
	if err != nil {
		t.Fatal(err)
	}
	return result.String()
}

func testAccCheckMetaKubeMaintenanceCronJobDestroy(s *terraform.State) error {
	k, err := testutil.GetTestClient()
	if err != nil {
		return fmt.Errorf("failed to get test client: %v", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "metakube_maintenance_cron_job" {
			continue
		}

		// Try to find the maintenance cron job
		projectID := rs.Primary.Attributes["project_id"]
		p := project.NewGetMaintenanceCronJobParams().WithProjectID(projectID).WithMaintenanceCronJobID(rs.Primary.ID)
		r, err := k.Client.Project.GetMaintenanceCronJob(p, k.Auth)
		if err == nil && r.Payload != nil {
			return fmt.Errorf("MaintenanceCronJob still exists")
		}
	}

	return nil
}

func testAccCheckMetaKubeMaintenanceCronJobExists(maintenanceCronJob *models.MaintenanceCronJob) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceName := "metakube_maintenance_cron_job.acctest"
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
		clusterID := rs.Primary.Attributes["cluster_id"]
		p := project.NewGetMaintenanceCronJobParams().WithProjectID(projectID).WithClusterID(clusterID).WithMaintenanceCronJobID(rs.Primary.ID)
		ret, err := k.Client.Project.GetMaintenanceCronJob(p, k.Auth)
		if err != nil {
			return fmt.Errorf("GetMaintenanceCronJob %v", err)
		}
		if ret.Payload == nil {
			return fmt.Errorf("Record not found")
		}

		*maintenanceCronJob = *ret.Payload

		return nil
	}
}

func testAccCheckMetaKubeMaintenanceCronJobFields(mcj *models.MaintenanceCronJob, name, schedule, maintenanceJobType string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if mcj == nil {
			return fmt.Errorf("No Record")
		}

		if mcj.Spec == nil {
			return fmt.Errorf("No Maintenance Cron Job spec present")
		}

		if mcj.Spec.Schedule == "" {
			return fmt.Errorf("No Maintenance Cron Job schedule present")
		}

		if mcj.Spec.MaintenanceJobTemplate == nil {
			return fmt.Errorf("No Maintenance Job Template present")
		}

		if mcj.Name != name {
			return fmt.Errorf("want MaintenanceCronJob.Name=%s, got %s", name, mcj.Name)
		}

		if mcj.Spec.Schedule != schedule {
			return fmt.Errorf("want MaintenanceCronJob.Schedule=%s, got %s", schedule, mcj.Spec.Schedule)
		}

		maintenanceJobTemplate := mcj.Spec.MaintenanceJobTemplate

		if maintenanceJobTemplate.Type != maintenanceJobType {
			return fmt.Errorf("want MaintenanceJobTemplate.Type=%s, got %s", maintenanceJobType, maintenanceJobTemplate.Type)
		}

		return nil
	}
}
