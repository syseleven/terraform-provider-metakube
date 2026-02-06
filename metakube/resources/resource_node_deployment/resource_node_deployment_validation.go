package resource_node_deployment

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/syseleven/go-metakube/models"
	"github.com/syseleven/terraform-provider-metakube/metakube/common"
)

func validateNodeSpecMatchesCluster() schema.CustomizeDiffFunc {
	return func(ctx context.Context, d *schema.ResourceDiff, meta interface{}) error {
		k := meta.(*common.MetaKubeProviderMeta)
		clusterID := d.Get("cluster_id").(string)
		if clusterID == "" {
			return nil
		}
		projectID := d.Get("project_id").(string)
		if projectID == "" {
			return nil
		}
		cluster, _, err := common.MetakubeGetCluster(ctx, projectID, clusterID, k)
		if err != nil {
			return err
		}
		clusterProvider, err := getClusterCloudProvider(cluster)
		if err != nil {
			return err
		}
		err = validateProviderMatchesCluster(d, clusterProvider)
		if err != nil {
			return err
		}
		return nil
	}
}

func getClusterCloudProvider(c *models.Cluster) (string, error) {
	switch {
	case c.Spec.Cloud.Aws != nil:
		return "aws", nil
	case c.Spec.Cloud.Openstack != nil:
		return "openstack", nil
	default:
		return "", fmt.Errorf("could not find cloud provider for cluster")

	}
}

func validateProviderMatchesCluster(d *schema.ResourceDiff, clusterProvider string) error {
	var availableProviders = []string{"aws", "openstack"}
	var provider string

	for _, p := range availableProviders {
		providerField := fmt.Sprintf("spec.0.template.0.cloud.0.%s", p)
		_, ok := d.GetOk(providerField)
		if ok {
			provider = p
			break
		}
	}
	if provider != clusterProvider {
		return fmt.Errorf("provider for node deployment must (%s) match cluster provider (%s)", provider, clusterProvider)
	}
	return nil
}

func validateAutoscalerFields() schema.CustomizeDiffFunc {
	return func(ctx context.Context, d *schema.ResourceDiff, _ interface{}) error {
		minReplicas, ok1 := d.GetOk("spec.0.min_replicas")
		maxReplicas, ok2 := d.GetOk("spec.0.max_replicas")
		if !ok1 && !ok2 {
			return nil
		}

		if minReplicas.(int) > maxReplicas.(int) {
			return fmt.Errorf("min_replicas must be smaller than max_replicas")
		}

		return nil
	}
}
