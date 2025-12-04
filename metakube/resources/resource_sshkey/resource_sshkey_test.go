package resource_sshkey_test

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
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

func TestAccMetakubeSSHKey_Basic(t *testing.T) {
	t.Parallel()
	var sshkey models.SSHKey
	testName := testutil.MakeRandomName()
	projectID := os.Getenv(common.TestEnvProjectID)
	resourceName := "metakube_sshkey.acctest_sshkey"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutil.TestAccPreCheck(t) },
		ProtoV5ProviderFactories: testutil.TestAccProtoV5ProviderFactories,
		CheckDestroy:             testutil.TestAccCheckMetaKubeSSHKeyDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccCheckMetaKubeSSHKeyConfigBasic, projectID, testName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutil.TestAccCheckMetaKubeSSHKeyExists(resourceName, &sshkey),
					testutil.TestAccCheckMetaKubeSSHKeyAttributes(&sshkey, testName),
					resource.TestCheckResourceAttr(resourceName, "name", testName),
					resource.TestCheckResourceAttr(resourceName, "public_key", testutil.TestSSHPubKey),
					resource.TestCheckResourceAttr(resourceName, "project_id", projectID),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return projectID + ":" + s.RootModule().Resources[resourceName].Primary.ID, nil
				},
			},
			// Test importing non-existent resource provides expected error.
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: false,
				ImportStateId:     projectID + ":123abc",
				ExpectError:       regexp.MustCompile(`(Please verify the ID is correct|Cannot import non-existent remote object)`),
			},
		},
	})
}

const (
	testAccCheckMetaKubeSSHKeyConfigBasic = `
resource "metakube_sshkey" "acctest_sshkey" {
	project_id = "%s"

	name = "%s"
	public_key = "` + testutil.TestSSHPubKey + `"
}
`
)
