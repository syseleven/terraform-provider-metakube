package datasource_project_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/syseleven/terraform-provider-metakube/metakube/common"
	"github.com/syseleven/terraform-provider-metakube/metakube/common/testutil"
)

func TestAccMetakubeProjectDataSource(t *testing.T) {
	t.Parallel()

	config := fmt.Sprintf(`
		data "metakube_project" "project" {
			name = "%s"
		}
	`, os.Getenv(common.TestEnvProjectName))

	projectID := os.Getenv(common.TestEnvProjectID)
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testutil.TestAccPreCheck(t)
			testutil.CheckEnv(t, common.TestEnvProjectName)
		},
		ProtoV5ProviderFactories: testutil.TestAccProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.metakube_project.project", "id", projectID),
				),
			},
		},
	})
}
