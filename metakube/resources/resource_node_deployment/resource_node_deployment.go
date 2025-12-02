package resource_node_deployment

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/syseleven/go-metakube/client/project"
	"github.com/syseleven/go-metakube/client/versions"
	"github.com/syseleven/go-metakube/models"
	"github.com/syseleven/terraform-provider-metakube/metakube/common"
)

func MetakubeResourceNodeDeployment() *schema.Resource {
	return &schema.Resource{
		CreateContext: metakubeResourceNodeDeploymentCreate,
		ReadContext:   metakubeResourceNodeDeploymentRead,
		UpdateContext: metakubeResourceNodeDeploymentUpdate,
		DeleteContext: metakubeResourceNodeDeploymentDelete,
		Importer: &schema.ResourceImporter{
			StateContext: common.ImportResourceWithProjectAndClusterID("node_deployment_name"),
		},
		CustomizeDiff: customdiff.All(
			validateNodeSpecMatchesCluster(),
			validateAutoscalerFields(),
		),

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(20 * time.Minute),
			Update: schema.DefaultTimeout(20 * time.Minute),
			Delete: schema.DefaultTimeout(20 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
				Description: "Project the cluster belongs to",
			},

			"cluster_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				// TODO: update descriptions
				Description: "Cluster that node deployment belongs to",
			},

			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Node deployment name",
			},

			"spec": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Required:    true,
				Description: "Node deployment specification",
				Elem: &schema.Resource{
					Schema: metakubeResourceNodeDeploymentSpecFields(),
				},
			},

			"creation_timestamp": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Creation timestamp",
			},

			"deletion_timestamp": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Deletion timestamp",
			},
		},
	}
}

func metakubeResourceNodeDeploymentCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	k := m.(*common.MetaKubeProviderMeta)
	clusterID := d.Get("cluster_id").(string)
	projectID := d.Get("project_id").(string)
	if projectID == "" {
		var err error
		projectID, err = common.MetakubeResourceClusterFindProjectID(ctx, clusterID, k)
		if err != nil {
			return diag.FromErr(err)
		}
		if projectID == "" {
			k.Log.Info("owner project for cluster '%s' is not found", clusterID)
			return diag.Errorf("could not find owner project for cluster with id '%s'", clusterID)
		}
	}

	nodeDeployment := &models.NodeDeployment{
		Name: d.Get("name").(string),
		Spec: metakubeNodeDeploymentExpandSpec(d.Get("spec").([]interface{}), true),
	}

	if err := metakubeResourceNodeDeploymentVersionCompatibleWithCluster(ctx, k, projectID, clusterID, nodeDeployment); err != nil {
		return diag.FromErr(err)
	}

	if err := common.MetakubeResourceClusterWaitForReady(ctx, k, d.Timeout(schema.TimeoutCreate), projectID, clusterID, ""); err != nil {
		return diag.Errorf("cluster is not ready: %v", err)
	}

	// Some cloud providers, like AWS, take some time to finish initializing.
	err := retry.RetryContext(ctx, d.Timeout(schema.TimeoutCreate), func() *retry.RetryError {
		p := project.NewListMachineDeploymentsParams().
			WithContext(ctx).
			WithProjectID(projectID).
			WithClusterID(clusterID)

		_, err := k.Client.Project.ListMachineDeployments(p, k.Auth)
		if err != nil {
			if e, ok := err.(*project.ListMachineDeploymentsDefault); ok && e.Code() != http.StatusOK {
				return retry.RetryableError(fmt.Errorf("unable to list node deployments %v", err))
			}
			return retry.NonRetryableError(err)
		}

		return nil
	})
	if err != nil {
		return diag.Errorf("nodedeployments API is not ready: %v", err)
	}

	p := project.NewCreateMachineDeploymentParams().
		WithContext(ctx).
		WithProjectID(projectID).
		WithClusterID(clusterID).
		WithBody(nodeDeployment)

	var id string
	err = retry.RetryContext(ctx, d.Timeout(schema.TimeoutCreate), func() *retry.RetryError {
		r, err := k.Client.Project.CreateMachineDeployment(p, k.Auth)
		if err != nil {
			e := common.StringifyResponseError(err)
			if strings.Contains(e, "failed calling webhook") || strings.Contains(e, "Cluster components are not ready yet") {
				return retry.RetryableError(fmt.Errorf("%v", e))
			}
			return retry.NonRetryableError(fmt.Errorf("%v", e))
		}
		id = r.Payload.ID
		return nil
	})
	if err != nil {
		return diag.Errorf("create a node deployment: %v", err)
	}
	d.SetId(id)
	d.Set("project_id", projectID)

	if err := metakubeResourceNodeDeploymentWaitForReady(ctx, k, d.Timeout(schema.TimeoutCreate), projectID, clusterID, id); err != nil {
		return diag.FromErr(err)
	}

	return metakubeResourceNodeDeploymentRead(ctx, d, m)

}

func metakubeResourceNodeDeploymentRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	k := m.(*common.MetaKubeProviderMeta)
	projectID := d.Get("project_id").(string)
	clusterID := d.Get("cluster_id").(string)
	p := project.NewGetMachineDeploymentParams().
		WithContext(ctx).
		WithProjectID(projectID).
		WithClusterID(clusterID).
		WithMachineDeploymentID(d.Id())

	r, err := k.Client.Project.GetMachineDeployment(p, k.Auth)
	if err != nil {
		if e, ok := err.(*project.GetMachineDeploymentDefault); ok && e.Code() == http.StatusNotFound {
			k.Log.Infof("removing node deployment '%s' from terraform state file, could not find the resource", d.Id())
			d.SetId("")
			return nil
		}
		if _, ok := err.(*project.GetMachineDeploymentForbidden); ok {
			k.Log.Infof("removing node deployment '%s' from terraform state file, access forbidden", d.Id())
			d.SetId("")
			return nil
		}
		return diag.Errorf("unable to get node deployment '%s/%s/%s': %s", projectID, clusterID, d.Id(), common.StringifyResponseError(err))
	}

	_ = d.Set("name", r.Payload.Name)

	_ = d.Set("spec", metakubeNodeDeploymentFlattenSpec(r.Payload.Spec))

	_ = d.Set("creation_timestamp", r.Payload.CreationTimestamp.String())

	_ = d.Set("deletion_timestamp", r.Payload.DeletionTimestamp.String())

	return nil
}

func metakubeResourceNodeDeploymentUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	k := m.(*common.MetaKubeProviderMeta)
	projectID := d.Get("project_id").(string)
	clusterID := d.Get("cluster_id").(string)

	nodeDeployment := &models.NodeDeployment{
		Spec: metakubeNodeDeploymentExpandSpec(d.Get("spec").([]interface{}), false),
	}

	if err := metakubeResourceNodeDeploymentVersionCompatibleWithCluster(ctx, k, projectID, clusterID, nodeDeployment); err != nil {
		return diag.FromErr(err)
	}

	p := project.NewPatchMachineDeploymentParams()
	p.SetContext(ctx)
	p.SetProjectID(projectID)
	p.SetClusterID(clusterID)
	p.SetMachineDeploymentID(d.Id())
	p.SetPatch(nodeDeployment)
	_, err := k.Client.Project.PatchMachineDeployment(p, k.Auth)
	if err != nil {
		return diag.Errorf("unable to update a node deployment: %v", common.StringifyResponseError(err))
	}

	if d.HasChange("spec.0.template.0.labels") {
		// To delete a label key we have to send PATCH request with that key set to null.
		// For simplicity we are doing it in a separate PATCH.

		before, now := d.GetChange("spec.0.template.0.labels")

		var beforeMap, nowMap map[string]interface{}
		var ok bool

		if before != nil {
			beforeMap, ok = before.(map[string]interface{})
			if !ok {
				return diag.Errorf("failed to apply labels change")
			}
		}

		if now != nil {
			nowMap, ok = now.(map[string]interface{})
			if !ok {
				return diag.Errorf("failed to apply labels change")
			}
		}

		labelsPatch := make(map[string]interface{})
		for k := range beforeMap {
			if _, ok := nowMap[k]; !ok {
				labelsPatch[k] = nil
			}
		}

		if len(labelsPatch) > 0 {
			patch := map[string]interface{}{
				"spec": map[string]interface{}{
					"template": map[string]interface{}{
						"labels": labelsPatch,
					},
				},
			}
			p := project.NewPatchMachineDeploymentParams()
			p.SetContext(ctx)
			p.SetProjectID(projectID)
			p.SetClusterID(clusterID)
			p.SetMachineDeploymentID(d.Id())
			p.SetPatch(&patch)

			err := retry.RetryContext(ctx, d.Timeout(schema.TimeoutUpdate), func() *retry.RetryError {
				_, err := k.Client.Project.PatchMachineDeployment(p, k.Auth)
				if err != nil {
					if strings.Contains(common.StringifyResponseError(err), "the object has been modified") {
						return retry.RetryableError(fmt.Errorf("machine deployment patch conflict: %v", err))
					}
					return retry.NonRetryableError(fmt.Errorf("patch machine deployment '%s': %v", d.Id(), err))
				}
				return nil
			})
			if err != nil {
				return diag.Errorf("unable to update a node deployment: %v", common.StringifyResponseError(err))
			}
		}
	}

	if err := metakubeResourceNodeDeploymentWaitForReady(ctx, k, d.Timeout(schema.TimeoutUpdate), projectID, clusterID, d.Id()); err != nil {
		return diag.FromErr(err)
	}

	return metakubeResourceNodeDeploymentRead(ctx, d, m)
}

func metakubeResourceNodeDeploymentVersionCompatibleWithCluster(ctx context.Context, k *common.MetaKubeProviderMeta, projectID, clusterID string, ndepl *models.NodeDeployment) error {
	cluster, _, err := common.MetakubeGetCluster(ctx, projectID, clusterID, k)
	if err != nil {
		return err
	}
	clusterVersion := string(cluster.Spec.Version)

	var kubeletVersion string
	if ndepl.Spec.Template != nil && ndepl.Spec.Template.Versions != nil {
		kubeletVersion = ndepl.Spec.Template.Versions.Kubelet
	}
	err = validateVersionAgainstCluster(kubeletVersion, clusterVersion)
	if err != nil {
		return err
	}
	return validateKubeletVersionIsAvailable(k, kubeletVersion, clusterVersion)
}

func validateVersionAgainstCluster(kubeletVersion, clusterVersion string) error {
	if kubeletVersion == "" {
		return nil
	}

	clusterSemverVersion, err := version.NewVersion(clusterVersion)
	if err != nil {
		return err
	}

	v, err := version.NewVersion(kubeletVersion)

	if err != nil {
		return fmt.Errorf("unable to parse node deployment version")
	}

	if clusterSemverVersion.LessThan(v) {
		return fmt.Errorf("node deployment version (%s) cannot be greater than cluster version (%s)", v, clusterVersion)
	}
	return nil
}

func validateKubeletVersionIsAvailable(k *common.MetaKubeProviderMeta, kubeletVersion, clusterVersion string) error {
	if kubeletVersion == "" {
		return nil
	}

	p := versions.NewGetNodeUpgradesParams()
	p.SetControlPlaneVersion(&clusterVersion)
	r, err := k.Client.Versions.GetNodeUpgrades(p, k.Auth)

	if err != nil {
		if e, ok := err.(*versions.GetNodeUpgradesDefault); ok && e.Payload != nil && e.Payload.Error != nil && e.Payload.Error.Message != nil {
			return fmt.Errorf("get node_deployment upgrades: %s", *e.Payload.Error.Message)
		}
		return err
	}

	var availableVersions []string
	for _, v := range r.Payload {
		if v.Version == kubeletVersion && !v.RestrictedByKubeletVersion {
			return nil
		}
		availableVersions = append(availableVersions, v.Version)
	}

	return fmt.Errorf("unknown version for node deployment %s, available versions %v", kubeletVersion, availableVersions)
}

func metakubeResourceNodeDeploymentWaitForReady(ctx context.Context, k *common.MetaKubeProviderMeta, timeout time.Duration, projectID, clusterID, id string) error {
	return retry.RetryContext(ctx, timeout, func() *retry.RetryError {
		p := project.NewGetMachineDeploymentParams().
			WithContext(ctx).
			WithProjectID(projectID).
			WithClusterID(clusterID).
			WithMachineDeploymentID(id)

		r, err := k.Client.Project.GetMachineDeployment(p, k.Auth)
		if err != nil {
			return retry.RetryableError(fmt.Errorf("unable to get node deployment %s", common.StringifyResponseError(err)))
		}

		if r.Payload.Spec.Replicas == nil || r.Payload.Status == nil || r.Payload.Status.ReadyReplicas < *r.Payload.Spec.Replicas || r.Payload.Status.UnavailableReplicas != 0 {
			k.Log.Debugf("waiting for node deployment '%s' to be ready, %+v", id, r.Payload.Status)
			return retry.RetryableError(fmt.Errorf("waiting for node deployment '%s' to be ready", id))
		}

		// Check all nodes are ready
		p2 := project.NewListMachineDeploymentNodesParams().
			WithContext(ctx).
			WithProjectID(projectID).
			WithClusterID(clusterID).
			WithMachineDeploymentID(id)
		r2, err := k.Client.Project.ListMachineDeploymentNodes(p2, k.Auth)
		if err != nil {
			return retry.RetryableError(fmt.Errorf("unable to list nodes %s", common.StringifyResponseError(err)))
		}
		if len(r2.Payload) != int(*r.Payload.Spec.Replicas) {
			k.Log.Debug("node count mismatch, want %v got %v", *r.Payload.Spec.Replicas, len(r2.Payload))
			return retry.RetryableError(fmt.Errorf("want %v nodes, got %v", *r.Payload.Spec.Replicas, len(r2.Payload)))
		}
		for _, node := range r2.Payload {
			if node.Status == nil || node.Status.NodeInfo == nil || node.Status.NodeInfo.KernelVersion == "" {
				k.Log.Debug("found not ready node")
				return retry.RetryableError(fmt.Errorf("some nodes are not ready"))
			}
		}
		return nil
	})
}

func metakubeResourceNodeDeploymentDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	k := m.(*common.MetaKubeProviderMeta)
	projectID := d.Get("project_id").(string)
	clusterID := d.Get("cluster_id").(string)
	p := project.NewDeleteMachineDeploymentParams().
		WithProjectID(projectID).
		WithClusterID(clusterID).
		WithMachineDeploymentID(d.Id())

	_, err := k.Client.Project.DeleteMachineDeployment(p, k.Auth)
	if err != nil {
		if e, ok := err.(*project.DeleteMachineDeploymentDefault); ok && e.Code() == http.StatusNotFound {
			k.Log.Infof("removing node deployment '%s' from terraform state file, could not find the resource", d.Id())
			d.SetId("")
			return nil
		}
		return diag.Errorf("unable to delete node deployment '%s': %s", d.Id(), common.StringifyResponseError(err))
	}

	err = retry.RetryContext(ctx, d.Timeout(schema.TimeoutDelete), func() *retry.RetryError {
		p := project.NewGetMachineDeploymentParams().
			WithContext(ctx).
			WithProjectID(projectID).
			WithClusterID(clusterID).
			WithMachineDeploymentID(d.Id())

		r, err := k.Client.Project.GetMachineDeployment(p, k.Auth)
		if err != nil {
			if e, ok := err.(*project.GetMachineDeploymentDefault); ok && e.Code() == http.StatusNotFound {
				k.Log.Debugf("node deployment '%s' has been destroyed, returned http code: %d", d.Id(), e.Code())
				d.SetId("")
				return nil
			}
			return retry.NonRetryableError(fmt.Errorf("unable to get node deployment '%s': %s", d.Id(), common.StringifyResponseError(err)))
		}

		k.Log.Debugf("node deployment '%s' deletion in progress, deletionTimestamp: %s",
			d.Id(), r.Payload.DeletionTimestamp.String())
		return retry.RetryableError(fmt.Errorf("node deployment '%s' deletion in progress", d.Id()))
	})
	if err != nil {
		return diag.FromErr(err)
	}
	return nil
}
