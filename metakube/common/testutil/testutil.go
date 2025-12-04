package testutil

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-mux/tf5muxserver"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/syseleven/go-metakube/client/project"
	"github.com/syseleven/go-metakube/models"
	"github.com/syseleven/terraform-provider-metakube/metakube"
	"github.com/syseleven/terraform-provider-metakube/metakube/common"
	"go.uber.org/zap"
)

var TestAccProtoV5ProviderFactories = map[string]func() (tfprotov5.ProviderServer, error){
	"metakube": func() (tfprotov5.ProviderServer, error) {
		providers := []func() tfprotov5.ProviderServer{
			providerserver.NewProtocol5(metakube.NewFrameworkProvider()),
			metakube.Provider().GRPCProvider,
		}
		muxServer, err := tf5muxserver.NewMuxServer(context.Background(), providers...)
		if err != nil {
			return nil, err
		}
		return muxServer.ProviderServer(), nil
	},
}

const (
	TestSSHPubKey = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCzoO6BIidD4Us9a9Kh0GzaUUxosl61GNUZzqcIdmf4EYZDdRtLa+nu88dHPHPQ2dj52BeVV9XVN9EufqdAZCaKpPLj5XxEwMpGcmdrOAl38kk2KKbiswjXkrdhYSBw3w0KkoCPKG/+yNpAUI9z+RJZ9lukeYBvxdDe8nuvUWX7mGRaPaumCpQaBHwYKNn6jMVns2RrumgE9w+Z6jlaKHk1V7T5rCBDcjXwcy6waOX6hKdPPBk84FpUfcfN/SdpwSVGFrcykazrpmzD2nYr71EcOm9T6/yuhBOiIa3H/TOji4G9wr02qtSWuGUpULkqWMFD+BQcYQQA71GSAa+rTZuf user@machine.local"
)

func MakeRandomName() string {
	return RandomName(common.TestNamePrefix, 5)
}

func RandomName(prefix string, length int) string {
	return fmt.Sprintf("%s%s", prefix, acctest.RandString(length))
}

func CheckEnv(t *testing.T, n string) {
	t.Helper()
	if v := os.Getenv(n); v == "" {
		t.Fatalf("%s must be set for acceptance tests", n)
	}
}

func MustParseTemplate(name, text string) *template.Template {
	r, err := template.New(name).Parse(text)
	if err != nil {
		panic(err)
	}
	return r
}

func TestAccPreCheck(t *testing.T) {
	t.Helper()
	CheckEnv(t, "METAKUBE_HOST")
	CheckEnv(t, "METAKUBE_TOKEN")
}

func TestCredentialId(t *testing.T) {
	id := common.GetSACredentialId()
	matched, err := regexp.MatchString(`s11auth:\w`, id)
	if err != nil {
		t.Fatalf("error %v: credential ID could not be matched", err)
	}
	if !matched {
		t.Fatalf("credential ID '%s' did not match service account format 's11auth:<projectId>'", id)
	}
}

func TestAccPreCheckForOpenstack(t *testing.T) {
	t.Helper()
	TestAccPreCheck(t)
	TestCredentialId(t)
	CheckEnv(t, common.TestEnvK8sVersionOpenstack)
	CheckEnv(t, common.TestEnvOpenstackProjectID)
	CheckEnv(t, common.TestEnvServiceAccountCredential)
	CheckEnv(t, common.TestEnvOpenstackProjectName)
	CheckEnv(t, common.TestEnvOpenstackRegion)
	CheckEnv(t, common.TestEnvOpenstackNodeDC)
	CheckEnv(t, common.TestEnvOpenstackImage)
	CheckEnv(t, common.TestEnvOpenstackImage2)
	CheckEnv(t, common.TestEnvOpenstackFlavor)
	CheckEnv(t, common.TestEnvOpenstackAuthURL)
	CheckEnv(t, common.TestEnvK8sOlderVersion)
	CheckEnv(t, common.TestEnvProjectID)
}

func GetTestClient() (*common.MetaKubeProviderMeta, error) {
	host := os.Getenv("METAKUBE_HOST")
	client, err := common.NewClient(host)
	if err != nil {
		return nil, fmt.Errorf("create client: %v", err)
	}
	token := common.TestEnvServiceAccountCredential
	auth, authErr := common.NewAuth(token, "", "")
	if authErr != nil {
		return nil, fmt.Errorf("auth api: %v", authErr)
	}
	log := zap.NewNop().Sugar()
	return &common.MetaKubeProviderMeta{
		Client: client,
		Auth:   auth,
		Log:    log,
	}, nil
}

func TestAccCheckMetaKubeSSHKeyExists(sshkeyN string, rec *models.SSHKey) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[sshkeyN]

		if !ok {
			return fmt.Errorf("Not found: %s", sshkeyN)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		k, err := GetTestClient()
		if err != nil {
			return fmt.Errorf("failed to get test client: %v", err)
		}
		p := project.NewListSSHKeysParams()
		p.SetProjectID(rs.Primary.Attributes["project_id"])

		ret, err := k.Client.Project.ListSSHKeys(p, k.Auth)
		if err != nil {
			return fmt.Errorf("Cannot verify record exist, list sshkeys error: %v", err)
		}

		for _, r := range ret.Payload {
			if r.ID == rs.Primary.ID {
				*rec = *r
				return nil
			}
		}

		return fmt.Errorf("Record not found")
	}
}

func TestAccCheckMetaKubeSSHKeyDestroy(s *terraform.State) error {
	k, err := GetTestClient()
	if err != nil {
		return fmt.Errorf("failed to get test client: %v", err)
	}

	// Check all ssh keys from all projects.
	for _, rsPrj := range s.RootModule().Resources {
		if rsPrj.Type != "metakube_project" {
			continue
		}

		p := project.NewListSSHKeysParams()
		p.SetProjectID(rsPrj.Primary.ID)
		sshkeys, err := k.Client.Project.ListSSHKeys(p, k.Auth)
		if err != nil {
			// API returns 403 if project doesn't exist.
			if _, ok := err.(*project.ListSSHKeysForbidden); ok {
				continue
			}
			if e, ok := err.(*project.ListSSHKeysDefault); ok && e.Code() == http.StatusNotFound {
				continue
			}
			return fmt.Errorf("check destroy: %v", err)
		}

		for _, rs := range s.RootModule().Resources {
			if rs.Type != "metakube_sshkey" {
				continue
			}

			// Try to find sshkey
			for _, r := range sshkeys.Payload {
				if r.ID == rs.Primary.ID {
					return fmt.Errorf("SSHKey still exists")
				}
			}
		}
	}

	return nil
}

func TestResourceInstanceState(name string, check func(*terraform.InstanceState) error) resource.TestCheckFunc {
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

func TestAccCheckMetaKubeClusterDestroy(s *terraform.State) error {
	k, err := GetTestClient()
	if err != nil {
		return fmt.Errorf("failed to get test client: %v", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "metakube_cluster" {
			continue
		}

		// Try to find the cluster
		projectID := rs.Primary.Attributes["project_id"]
		p := project.NewGetClusterV2Params().WithProjectID(projectID).WithClusterID(rs.Primary.ID)
		r, err := k.Client.Project.GetClusterV2(p, k.Auth)
		if err == nil && r.Payload != nil {
			return fmt.Errorf("Cluster still exists")
		}
	}

	return nil
}

func TestAccCheckMetaKubeSSHKeyAttributes(rec *models.SSHKey, name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if rec.Name != name {
			return fmt.Errorf("want SSHKey.Name=%s, got %s", name, rec.Name)
		}

		if rec.Spec.PublicKey != TestSSHPubKey {
			return fmt.Errorf("want SSHKey.PublicKey=%s, got %s", TestSSHPubKey, rec.Spec.PublicKey)
		}

		return nil
	}
}
