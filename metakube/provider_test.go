package metakube

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/syseleven/go-metakube/client/project"
	"github.com/syseleven/terraform-provider-metakube/metakube/common"
	"github.com/syseleven/terraform-provider-metakube/metakube/common/provider_testutil"
	"go.uber.org/zap"
)

func init() {
	resource.AddTestSweepers("metakube_cluster", &resource.Sweeper{
		Name: "metakube_cluster",
		F:    testSweepClusters,
	})
	resource.AddTestSweepers("metakube_sshkey", &resource.Sweeper{
		Name: "metakube_sshkey",
		F:    testSweepSSHKeys,
	})
}
func TestMain(m *testing.M) {
	provider_testutil.TestAccProvider = Provider()
	provider_testutil.TestAccProviders = map[string]*schema.Provider{
		"metakube": provider_testutil.TestAccProvider,
	}
	resource.TestMain(m)
}

func testSweepClusters(region string) error {
	meta, err := SharedConfigForRegion(region)
	if err != nil {
		return err
	}

	projectID := os.Getenv(common.TestEnvProjectID)
	params := project.NewListClustersV2Params().WithProjectID(projectID)
	records, err := meta.Client.Project.ListClustersV2(params, meta.Auth)
	if err != nil {
		return fmt.Errorf("sweep list clusters: %s", common.StringifyResponseError(err))
	}

	for _, rec := range records.Payload {
		if !strings.HasPrefix(rec.Name, common.TestNamePrefix) || !time.Time(rec.DeletionTimestamp).IsZero() {
			continue
		}

		p := project.NewDeleteClusterV2Params().
			WithProjectID(projectID).
			WithClusterID(rec.ID)
		if _, err := meta.Client.Project.DeleteClusterV2(p, meta.Auth); err != nil {
			return fmt.Errorf("delete cluster: %v", common.StringifyResponseError(err))
		}
	}

	return nil
}

func testSweepSSHKeys(region string) error {
	meta, err := SharedConfigForRegion(region)
	if err != nil {
		return err
	}

	projectID := os.Getenv(common.TestEnvProjectID)
	params := project.NewListSSHKeysParams().WithProjectID(projectID)
	records, err := meta.Client.Project.ListSSHKeys(params, meta.Auth)
	if err != nil {
		return fmt.Errorf("list sshkeys: %v", err)
	}

	for _, rec := range records.Payload {
		if !strings.HasPrefix(rec.Name, common.TestNamePrefix) || !time.Time(rec.DeletionTimestamp).IsZero() {
			continue
		}

		p := project.NewDeleteSSHKeyParams().
			WithProjectID(projectID).
			WithSSHKeyID(rec.ID)
		if _, err := meta.Client.Project.DeleteSSHKey(p, meta.Auth); err != nil {
			return fmt.Errorf("delete sshkey: %v", err)
		}
	}

	return nil
}

func SharedConfigForRegion(region string) (*common.MetaKubeProviderMeta, error) {
	host := os.Getenv("METAKUBE_HOST")
	client, err := common.NewClient(host)
	if err != nil {
		return nil, fmt.Errorf("create client %v", err)
	}

	token := os.Getenv("METAKUBE_TOKEN")
	if regionStringIsNCS(region) {
		token = os.Getenv(common.TestEnvServiceAccountCredential)
	}
	auth, errr := common.NewAuth(token, "", "")
	if errr != nil {
		return nil, fmt.Errorf("auth api %v", common.ToFrameworkDiagnostics(errr))
	}
	log := zap.NewNop().Sugar()
	return &common.MetaKubeProviderMeta{
		Client: client,
		Auth:   auth,
		Log:    log,
	}, nil
}

func regionStringIsNCS(region string) bool {
	for _, prefix := range []string{"aws", "dbl", "cbk", "fes", "ybk", "zbk"} {
		if strings.Contains(region, prefix) {
			return false
		}
	}
	return true
}
