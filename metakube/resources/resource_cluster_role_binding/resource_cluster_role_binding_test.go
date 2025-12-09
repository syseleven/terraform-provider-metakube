package resource_cluster_role_binding_test

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
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

func TestAccMetakubeClusterRoleBinding(t *testing.T) {
	t.Parallel()
	resourceName := "metakube_cluster_role_binding.acctest"
	params := &testAccCheckMetaKubeClusterRoleBindingBasicParams{
		ClusterName:                          testutil.MakeRandomName() + "-cluster-role-binding",
		DatacenterName:                       os.Getenv(common.TestEnvOpenstackNodeDC),
		ProjectID:                            os.Getenv(common.TestEnvProjectID),
		Version:                              os.Getenv(common.TestEnvK8sVersionOpenstack),
		OpenstackApplicationCredentialID:     common.GetSACredentialId(),
		OpenstackApplicationCredentialSecret: os.Getenv(common.TestEnvServiceAccountCredential),

		ClusterRoleName:  "view",
		UserSubjectName:  "foo.bar@mycompany.xyz",
		GroupSubjectName: "support-team",
	}
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testutil.CheckEnv(t, "METAKUBE_HOST")
			testutil.CheckEnv(t, common.TestEnvServiceAccountCredential)
			testutil.CheckEnv(t, common.TestEnvK8sVersionOpenstack)
			testutil.CheckEnv(t, common.TestEnvOpenstackNodeDC)
			testutil.CheckEnv(t, common.TestEnvK8sVersionOpenstack)
			testutil.CheckEnv(t, common.TestEnvProjectID)
		},
		ProtoV5ProviderFactories: testutil.TestAccProtoV5ProviderFactories,
		CheckDestroy:             testutil.TestAccCheckMetaKubeSSHKeyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckMetaKubeClusterRoleBindingBasicConfig(t, params),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "cluster_role_name", params.ClusterRoleName),
					resource.TestCheckResourceAttr(resourceName, "subject.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.kind", "user"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.name", params.UserSubjectName),
					resource.TestCheckResourceAttr(resourceName, "subject.1.kind", "group"),
					resource.TestCheckResourceAttr(resourceName, "subject.1.name", params.GroupSubjectName),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					for _, rs := range s.RootModule().Resources {
						if rs.Type == "metakube_cluster_role_binding" {
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
				ImportStateId:     "123abc",
				ExpectError:       regexp.MustCompile(`please provide resource identifier in format 'project_id:cluster_id:cluster_role_binding_name'`),
			},
		},
	})
}

type testAccCheckMetaKubeClusterRoleBindingBasicParams struct {
	ClusterName                          string
	DatacenterName                       string
	ProjectID                            string
	Version                              string
	OpenstackApplicationCredentialID     string
	OpenstackApplicationCredentialSecret string

	ClusterRoleName  string
	UserSubjectName  string
	GroupSubjectName string
}

func testAccCheckMetaKubeClusterRoleBindingBasicConfig(t *testing.T, params *testAccCheckMetaKubeClusterRoleBindingBasicParams) string {
	t.Helper()

	var result strings.Builder
	err := testutil.MustParseTemplate("cluster role binding test template", `
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

resource "metakube_cluster_role_binding" "acctest" {
	project_id = "{{ .ProjectID }}"
	cluster_id = metakube_cluster.acctest.id
	cluster_role_name = "{{ .ClusterRoleName }}"

    subject {
		kind = "user"
		name = "{{ .UserSubjectName }}"
	}

    subject {
		kind = "group"
		name = "{{ .GroupSubjectName }}"
	}
}
`).Execute(&result, params)
	if err != nil {
		t.Fatal(err)
	}
	return result.String()
}
