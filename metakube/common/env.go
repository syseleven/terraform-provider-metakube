package common

import "os"

const (
	TestNamePrefix = "tf-acc-test-"

	TestEnvK8sVersionOpenstack = "METAKUBE_K8S_VERSION_OS"
	TestEnvK8sOlderVersion     = "METAKUBE_K8S_OLDER_VERSION"

	TestEnvProjectID                = "METAKUBE_NCS_PROJECT_ID"
	TestEnvServiceAccountCredential = "METAKUBE_SERVICE_ACCOUNT_CREDENTIAL"

	TestEnvOpenstackAuthURL     = "METAKUBE_OPENSTACK_AUTH_URL"
	TestEnvOpenstackProjectName = "METAKUBE_OPENSTACK_PROJECT_NAME"
	TestEnvOpenstackNodeDC      = "METAKUBE_NCS_OPENSTACK_NODE_DC"
	TestEnvOpenstackRegion      = "METAKUBE_NCS_OPENSTACK_REGION"
	TestEnvOpenstackImage       = "METAKUBE_OPENSTACK_IMAGE"
	TestEnvOpenstackImage2      = "METAKUBE_OPENSTACK_IMAGE2"
	TestEnvOpenstackFlavor      = "METAKUBE_OPENSTACK_FLAVOR"
)

// GetSaCredentialId returns the credential ID for service accounts
func GetSACredentialId() string {
	return "s11auth:" + os.Getenv(TestEnvProjectID)
}
