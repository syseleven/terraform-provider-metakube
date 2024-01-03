package metakube

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccMetakubeProjectDataSource(t *testing.T) {
	t.Parallel()

	config := fmt.Sprintf(`
		data "metakube_project" "project" {
			name = "%s"
		}
	`, os.Getenv(testEnvProjectName))

	projectID := os.Getenv(testEnvProjectID)
	resource.Test(t, resource.TestCase{
		Providers: map[string]*schema.Provider{
			"metakube": testAccProvider,
		},
		PreCheck: func() {
			testAccPreCheck(t)
			checkEnv(t, testEnvProjectName)
		},
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
