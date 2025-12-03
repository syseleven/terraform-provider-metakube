package common

import "os"

const (
	TestNamePrefix = "tf-acc-test-"

	TestEnvK8sVersionOpenstack = "METAKUBE_K8S_VERSION_OS"
	TestEnvK8sVersionAWS       = "METAKUBE_K8S_VERSION_AWS"
	TestEnvK8sOlderVersion     = "METAKUBE_K8S_OLDER_VERSION"

	TestEnvProjectID                = "METAKUBE_NCS_PROJECT_ID"
	TestEnvProjectName              = "METAKUBE_NCS_PROJECT_NAME"
	TestEnvServiceAccountCredential = "METAKUBE_SERVICE_ACCOUNT_CREDENTIAL"

	TestEnvOpenstackAuthURL     = "METAKUBE_OPENSTACK_AUTH_URL"
	TestEnvOpenstackProjectID   = "METAKUBE_OPENSTACK_PROJECT_ID"
	TestEnvOpenstackProjectName = "METAKUBE_OPENSTACK_PROJECT_NAME"
	TestEnvOpenstackNodeDC      = "METAKUBE_NCS_OPENSTACK_NODE_DC"
	TestEnvOpenstackRegion      = "METAKUBE_NCS_OPENSTACK_REGION"
	TestEnvOpenstackImage       = "METAKUBE_OPENSTACK_IMAGE"
	TestEnvOpenstackImage2      = "METAKUBE_OPENSTACK_IMAGE2"
	TestEnvOpenstackFlavor      = "METAKUBE_OPENSTACK_FLAVOR"

	TestEnvAWSAccessKeyID      = "METAKUBE_AWS_ACCESS_KEY_ID"
	TestAWSSecretAccessKey     = "METAKUBE_AWS_ACCESS_KEY_SECRET"
	TestEnvAWSVPCID            = "METAKUBE_AWS_VPC_ID"
	TestEnvAWSNodeDC           = "METAKUBE_AWS_NODE_DC"
	TestEnvAWSInstanceType     = "METAKUBE_AWS_INSTANCE_TYPE"
	TestEnvAWSSubnetID         = "METAKUBE_AWS_SUBNET_ID"
	TestEnvAWSAvailabilityZone = "METAKUBE_AWS_AVAILABILITY_ZONE"
	TestEnvAWSDiskSize         = "METAKUBE_AWS_DISK_SIZE"
	TestEnvAWSAMI              = "METAKUBE_AWS_AMI"
)

// GetSaCredentialId returns the credential ID for service accounts
func GetSACredentialId() string {
	return "s11auth:" + os.Getenv(TestEnvProjectID)
}
