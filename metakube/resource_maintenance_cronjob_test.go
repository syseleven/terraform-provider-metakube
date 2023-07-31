package metakube

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/syseleven/go-metakube/client/project"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/syseleven/go-metakube/models"
)

func TestAccMetakubeCluster_MaintenanceCronJob_Basic(t *testing.T) {
	t.Parallel()
	var maintenanceCronJob models.MaintenanceCronJob

	resourceName := "metakube_maintenance_cron_job.acctest"
	params := &testAccCheckMetaKubeMaintenanceCronJobBasicParams{
		ClusterName:                          randomName("testacc", 5),
		DatacenterName:                       os.Getenv(testEnvOpenstackNodeDC),
		ProjectID:                            os.Getenv(testEnvProjectID),
		Version:                              os.Getenv(testEnvK8sVersionOpenstack),
		OpenstackApplicationCredentialID:     os.Getenv(testEnvOpenstackApplicationCredentialsID),
		OpenstackApplicationCredentialSecret: os.Getenv(testEnvOpenstackApplicationCredentialsSecret),

		MaintenanceCronJobName:     randomName("test-maintenancecronjob", 5),
		MaintenanceJobTemplateName: randomName("test-maintenancecronjob-template", 5),
		MaintenanceJobType:         "kubernetesPatchUpdate",
		Schedule:                   "5 4 * * *",
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheckForOpenstack(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckMetaKubeMaintenanceCronJobDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckMetaKubeMaintenanceCronJobBasicConfig(t, params),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMetaKubeMaintenanceCronJobExists(&maintenanceCronJob),
					testAccCheckMetaKubeMaintenanceCronJobFields(&maintenanceCronJob, params.MaintenanceCronJobName, params.Schedule, params.MaintenanceJobType),
					resource.TestCheckResourceAttr(resourceName, "name", params.MaintenanceCronJobName),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.schedule", params.Schedule),
					resource.TestCheckResourceAttr(resourceName, "spec.0.maintenance_job_template.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.maintenance_job_template.0.rollback", "false"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.maintenance_job_template.0.type", params.MaintenanceJobType),
					resource.TestCheckResourceAttrSet(resourceName, "creation_timestamp"),
					resource.TestCheckResourceAttrSet(resourceName, "deletion_timestamp"),
				),
			},
			{
				Config:   testAccCheckMetaKubeMaintenanceCronJobBasicConfig(t, params),
				PlanOnly: true,
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
}

func testAccCheckMetaKubeMaintenanceCronJobBasicConfig(t *testing.T, params *testAccCheckMetaKubeMaintenanceCronJobBasicParams) string {
	t.Helper()

	var result strings.Builder
	err := mustParseTemplate("metakube maintenance cron job test template", `
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

func testAccCheckMetaKubeMaintenanceCronJobBasicSecondConfig(t *testing.T, params *testAccCheckMetaKubeMaintenanceCronJobBasicParams) string {
	t.Helper()

	var result strings.Builder
	err := mustParseTemplate("metakube maintenance cron job test template", `
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

		spec {
			schedule		= "{{ .Schedule }}"
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
	k := testAccProvider.Meta().(*metakubeProviderMeta)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "metakube_maintenance_cron_job" {
			continue
		}

		// Try to find the maintenance cron job
		projectID := rs.Primary.Attributes["project_id"]
		p := project.NewGetMaintenanceCronJobParams().WithProjectID(projectID).WithMaintenanceCronJobID(rs.Primary.ID)
		r, err := k.client.Project.GetMaintenanceCronJob(p, k.auth)
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

		k := testAccProvider.Meta().(*metakubeProviderMeta)
		projectID := rs.Primary.Attributes["project_id"]
		clusterID := rs.Primary.Attributes["cluster_id"]
		p := project.NewGetMaintenanceCronJobParams().WithProjectID(projectID).WithClusterID(clusterID).WithMaintenanceCronJobID(rs.Primary.ID)
		ret, err := k.client.Project.GetMaintenanceCronJob(p, k.auth)
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
