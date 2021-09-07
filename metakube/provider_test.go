package metakube

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	testNamePrefix = "tf-acc-test-"

	testEnvOtherUserEmail = "METAKUBE_ANOTHER_USER_EMAIL"

	testEnvK8sVersion      = "METAKUBE_K8S_VERSION"
	testEnvK8sOlderVersion = "METAKUBE_K8S_OLDER_VERSION"

	testEnvOpenstackNodeDC                       = "METAKUBE_OPENSTACK_NODE_DC"
	testEnvOpenstackApplicationCredentialsID     = "METAKUBE_OPENSTACK_APPLICATION_CREDENTIALS_ID"
	testEnvOpenstackApplicationCredentialsSecret = "METAKUBE_OPENSTACK_APPLICATION_CREDENTIALS_SECRET"
	testEnvOpenstackUsername                     = "METAKUBE_OPENSTACK_USERNAME"
	testEnvOpenstackAuthUrl                      = "METAKUBE_OPENSTACK_AUTH_URL"
	testEnvOpenstackPassword                     = "METAKUBE_OPENSTACK_PASSWORD"
	testEnvOpenstackTenant                       = "METAKUBE_OPENSTACK_TENANT"
	testEnvOpenstackImage                        = "METAKUBE_OPENSTACK_IMAGE"
	testEnvOpenstackImage2                       = "METAKUBE_OPENSTACK_IMAGE2"
	testEnvOpenstackImageFlatcar                 = "METAKUBE_OPENSTACK_IMAGE_FLATCAR"
	testEnvOpenstackFlavor                       = "METAKUBE_OPENSTACK_FLAVOR"

	testEnvAzureNodeDC         = "METAKUBE_AZURE_NODE_DC"
	testEnvAzureNodeSize       = "METAKUBE_AZURE_NODE_SIZE"
	testEnvAzureClientID       = "METAKUBE_AZURE_CLIENT_ID"
	testEnvAzureClientSecret   = "METAKUBE_AZURE_CLIENT_SECRET"
	testEnvAzureTenantID       = "METAKUBE_AZURE_TENANT_ID"
	testEnvAzureSubscriptionID = "METAKUBE_AZURE_SUBSCRIPTION_ID"

	testEnvAWSAccessKeyID      = "METAKUBE_AWS_ACCESS_KEY_ID"
	testAWSSecretAccessKey     = "METAKUBE_AWS_ACCESS_KEY_SECRET"
	testEnvAWSVPCID            = "METAKUBE_AWS_VPC_ID"
	testEnvAWSNodeDC           = "METAKUBE_AWS_NODE_DC"
	testEnvAWSInstanceType     = "METAKUBE_AWS_INSTANCE_TYPE"
	testEnvAWSSubnetID         = "METAKUBE_AWS_SUBNET_ID"
	testEnvAWSAvailabilityZone = "METAKUBE_AWS_AVAILABILITY_ZONE"
	testEnvAWSDiskSize         = "METAKUBE_AWS_DISK_SIZE"
)

var (
	testAccProviders map[string]*schema.Provider
	testAccProvider  *schema.Provider
)

func TestMain(m *testing.M) {
	testAccProvider = Provider()
	testAccProviders = map[string]*schema.Provider{
		"metakube": testAccProvider,
	}
	resource.TestMain(m)
}

func testAccPreCheckForOpenstack(t *testing.T) {
	t.Helper()
	testAccPreCheck(t)
	checkEnv(t, testEnvOpenstackUsername)
	checkEnv(t, testEnvOpenstackPassword)
	checkEnv(t, testEnvOpenstackTenant)
	checkEnv(t, testEnvOpenstackNodeDC)
	checkEnv(t, testEnvOpenstackImage)
	checkEnv(t, testEnvOpenstackImage2)
	checkEnv(t, testEnvOpenstackFlavor)
}

func testAccPreCheckForAzure(t *testing.T) {
	t.Helper()
	testAccPreCheck(t)
	checkEnv(t, testEnvAzureClientID)
	checkEnv(t, testEnvAzureClientSecret)
	checkEnv(t, testEnvAzureSubscriptionID)
	checkEnv(t, testEnvAzureTenantID)
	checkEnv(t, testEnvAzureNodeDC)
	checkEnv(t, testEnvAzureNodeSize)
}

func testAccPreCheckForAWS(t *testing.T) {
	t.Helper()
	testAccPreCheck(t)
	checkEnv(t, testEnvAWSAccessKeyID)
	checkEnv(t, testAWSSecretAccessKey)
	checkEnv(t, testEnvAWSVPCID)
	checkEnv(t, testEnvAWSNodeDC)
}

func testAccPreCheck(t *testing.T) {
	t.Helper()
	checkEnv(t, "METAKUBE_HOST")
	checkEnv(t, "METAKUBE_TOKEN")
	checkEnv(t, testEnvK8sVersion)
	checkEnv(t, testEnvK8sOlderVersion)
}

func checkEnv(t *testing.T, n string) {
	t.Helper()
	if v := os.Getenv(n); v == "" {
		t.Fatalf("%s must be set for acceptance tests", n)
	}
}

func makeRandomString() string {
	return randomName(testNamePrefix, 10)
}

func randomName(prefix string, length int) string {
	return fmt.Sprintf("%s%s", prefix, acctest.RandString(length))
}

func testResourceInstanceState(name string, check func(*terraform.InstanceState) error) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		m := s.RootModule()
		if rs, ok := m.Resources[name]; ok {
			is := rs.Primary
			if is == nil {
				return fmt.Errorf("no instance: %s", name)
			}

			return check(is)
		}
		return fmt.Errorf("not found: %s", name)

	}
}
