package metakube

import (
	"fmt"
	"os"
	"regexp"
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

		MaintenanceCronJobName:     "test_maintenance_cron_job_name",
		MaintenanceJobTemplateName: "test_maintenance_job_template_name",
		MaintenanceJobType:         "test_maintenance_job_type",
	}
	var config strings.Builder
	if err := testAccCheckMetaKubeMaintenanceCronJobBasicTemplate.Execute(&config, params); err != nil {
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
		CheckDestroy: testAccCheckMetaKubeMaintenanceCronJobDestroy,
		Steps: []resource.TestStep{
			{
				Config: config.String(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMetaKubeMaintenanceCronJobExists(&maintenanceCronJob),
					resource.TestCheckResourceAttr(resourceName, "maintenance_cron_job_name", params.MaintenanceCronJobName),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.schedule", "5 4 * * *"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.maintenance_job_template.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.maintenance_job_template.0.rollback", "false"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.maintenance_job_template.0.type", params.MaintenanceJobType),
					resource.TestCheckResourceAttrSet(resourceName, "creation_timestamp"),
					resource.TestCheckResourceAttrSet(resourceName, "deletion_timestamp"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: false,
				ImportStateId:     "x:y:123xyz",
				ExpectError:       regexp.MustCompile(`(Please verify the ID is correct|Cannot import non-existent remote object)`),
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
}

var testAccCheckMetaKubeMaintenanceCronJobBasicTemplate = mustParseTemplate("maintenanceCronJobBasic", `
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
		project_id = "{{ .ProjectID }}"
		cluster_id = metakube_cluster.acctest.id
		name = "{{ .MaintenanceCronJobName }}"
		timeouts {
			create = "15m"
			update = "15m"
			delete = "15m"
		}
		spec {
			schedule						= "5 4 * * *"
			maintenance_job_template {
				options = {
					"a" = "b"
					"c" = "d"
				}
				rollback 	= false
				type		= "{{ .MaintenanceJobType }}"
			}
		}
	}`)

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
		p := project.NewGetMaintenanceCronJobParams().WithProjectID(projectID).WithMaintenanceCronJobID(rs.Primary.ID)
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
