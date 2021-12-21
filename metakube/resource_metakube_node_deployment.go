package metakube

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/syseleven/go-metakube/client/project"
	"github.com/syseleven/go-metakube/client/versions"
	"github.com/syseleven/go-metakube/models"
)

func metakubeResourceNodeDeployment() *schema.Resource {
	return &schema.Resource{
		CreateContext: metakubeResourceNodeDeploymentCreate,
		ReadContext:   metakubeResourceNodeDeploymentRead,
		UpdateContext: metakubeResourceNodeDeploymentUpdate,
		DeleteContext: metakubeResourceNodeDeploymentDelete,
		Importer: &schema.ResourceImporter{
			StateContext: importResourceWithProjectAndClusterID("node_deployment_name"),
		},
		CustomizeDiff: customdiff.All(
			validateNodeSpecMatchesCluster(),
			validateAutoscalerFields(),
		),

		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:        schema.TypeString,
				Computed:    true,
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
					Schema: matakubeResourceNodeDeploymentSpecFields(),
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

func importResourceWithProjectAndClusterID(identifierName string) func(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	return func(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
		parts := strings.Split(d.Id(), ":")
		if len(parts) != 3 {
			return nil, fmt.Errorf("please provide resource identifier in format 'project_id:cluster_id:%s'", identifierName)
		}
		d.Set("project_id", parts[0])
		d.Set("cluster_id", parts[1])
		d.SetId(parts[2])
		return []*schema.ResourceData{d}, nil
	}
}

func metakubeResourceNodeDeploymentCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	k := m.(*metakubeProviderMeta)
	clusterID := d.Get("cluster_id").(string)
	projectID := d.Get("project_id").(string)
	if projectID == "" {
		var err error
		projectID, err = metakubeResourceClusterFindProjectID(ctx, clusterID, k)
		if err != nil {
			return diag.FromErr(err)
		}
		if projectID == "" {
			k.log.Info("owner project for cluster '%s' is not found", clusterID)
			return diag.Errorf("")
		}
	}

	nodeDeployment := &models.NodeDeployment{
		Name: d.Get("name").(string),
		Spec: metakubeNodeDeploymentExpandSpec(d.Get("spec").([]interface{})),
	}

	if err := metakubeResourceNodeDeploymentVersionCompatibleWithCluster(ctx, k, projectID, clusterID, nodeDeployment); err != nil {
		return diag.FromErr(err)
	}

	p := project.NewCreateMachineDeploymentParams().
		WithContext(ctx).
		WithProjectID(projectID).
		WithClusterID(clusterID).
		WithBody(nodeDeployment)

	if err := metakubeResourceClusterWaitForReady(ctx, k, d, projectID, clusterID); err != nil {
		return diag.Errorf("cluster is not ready: %v", err)
	}

	// Some cloud providers, like AWS, take some time to finish initializing.
	err := resource.RetryContext(ctx, d.Timeout(schema.TimeoutCreate), func() *resource.RetryError {
		p := project.NewListMachineDeploymentsParams().
			WithContext(ctx).
			WithProjectID(projectID).
			WithClusterID(clusterID)

		_, err := k.client.Project.ListMachineDeployments(p, k.auth)
		if err != nil {
			if e, ok := err.(*project.ListMachineDeploymentsDefault); ok && e.Code() != http.StatusOK {
				return resource.RetryableError(fmt.Errorf("unable to list node deployments %v", err))
			}
			return resource.NonRetryableError(err)
		}

		return nil
	})
	if err != nil {
		return diag.Errorf("nodedeployments API is not ready: %v", err)
	}

	var id string
	err = resource.RetryContext(ctx, d.Timeout(schema.TimeoutCreate), func() *resource.RetryError {
		r, err := k.client.Project.CreateMachineDeployment(p, k.auth)
		if err != nil {
			e := stringifyResponseError(err)
			if strings.Contains(e, "failed calling webhook") || strings.Contains(e, "Cluster components are not ready yet") {
				return resource.RetryableError(fmt.Errorf(e))
			}
			return resource.NonRetryableError(fmt.Errorf(e))
		}
		id = r.Payload.ID
		return nil
	})
	if err != nil {
		return diag.Errorf("create a node deployment: %v", err)
	}
	d.SetId(id)
	d.Set("project_id", projectID)

	if err := metakubeResourceNodeDeploymentWaitForReady(ctx, k, d.Timeout(schema.TimeoutCreate), projectID, clusterID, id, 0); err != nil {
		return diag.FromErr(err)
	}

	return metakubeResourceNodeDeploymentRead(ctx, d, m)

}

func metakubeResourceNodeDeploymentRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	k := m.(*metakubeProviderMeta)
	projectID := d.Get("project_id").(string)
	clusterID := d.Get("cluster_id").(string)
	p := project.NewGetMachineDeploymentParams().
		WithContext(ctx).
		WithProjectID(projectID).
		WithClusterID(clusterID).
		WithMachineDeploymentID(d.Id())

	r, err := k.client.Project.GetMachineDeployment(p, k.auth)
	if err != nil {
		if e, ok := err.(*project.GetMachineDeploymentDefault); ok && e.Code() == http.StatusNotFound {
			k.log.Infof("removing node deployment '%s' from terraform state file, could not find the resource", d.Id())
			d.SetId("")
			return nil
		}
		if _, ok := err.(*project.GetMachineDeploymentForbidden); ok {
			k.log.Infof("removing node deployment '%s' from terraform state file, access forbidden", d.Id())
			d.SetId("")
			return nil
		}
		return diag.Errorf("unable to get node deployment '%s/%s/%s': %s", projectID, clusterID, d.Id(), stringifyResponseError(err))
	}

	_ = d.Set("name", r.Payload.Name)

	_ = d.Set("spec", metakubeNodeDeploymentFlattenSpec(r.Payload.Spec))

	_ = d.Set("creation_timestamp", r.Payload.CreationTimestamp.String())

	_ = d.Set("deletion_timestamp", r.Payload.DeletionTimestamp.String())

	return nil
}

func metakubeResourceNodeDeploymentUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	k := m.(*metakubeProviderMeta)
	projectID := d.Get("project_id").(string)
	clusterID := d.Get("cluster_id").(string)

	nodeDeployment := &models.NodeDeployment{
		Spec: metakubeNodeDeploymentExpandSpec(d.Get("spec").([]interface{})),
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
	res, err := k.client.Project.PatchMachineDeployment(p, k.auth)
	if err != nil {
		return diag.Errorf("unable to update a node deployment: %v", stringifyResponseError(err))
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

			resource.RetryContext(ctx, d.Timeout(schema.TimeoutUpdate), func() *resource.RetryError {
				_, err := k.client.Project.PatchMachineDeployment(p, k.auth)
				if err != nil {
					if strings.Contains(stringifyResponseError(err), "the object has been modified") {
						return resource.RetryableError(fmt.Errorf("machine deployment patch conflict: %v", err))
					}
					return resource.NonRetryableError(fmt.Errorf("patch machine deployment '%s': %v", d.Id(), err))
				}
				return nil
			})
			if err != nil {
				return diag.Errorf("unable to update a node deployment: %v", stringifyResponseError(err))
			}
		}
	}

	if err := metakubeResourceNodeDeploymentWaitForReady(ctx, k, d.Timeout(schema.TimeoutCreate), projectID, clusterID, d.Id(), res.Payload.Status.ObservedGeneration); err != nil {
		return diag.FromErr(err)
	}

	return metakubeResourceNodeDeploymentRead(ctx, d, m)
}

func metakubeResourceNodeDeploymentVersionCompatibleWithCluster(ctx context.Context, k *metakubeProviderMeta, projectID, clusterID string, ndepl *models.NodeDeployment) error {
	cluster, err := metakubeGetCluster(ctx, projectID, clusterID, k)
	if err != nil {
		return err
	}
	clusterVersion := cluster.Spec.Version.(string)

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

func validateKubeletVersionIsAvailable(k *metakubeProviderMeta, kubeletVersion, clusterVersion string) error {
	if kubeletVersion == "" {
		return nil
	}

	versionType := "kubernetes"

	p := versions.NewGetNodeUpgradesParams()
	p.SetType(&versionType)
	p.SetControlPlaneVersion(&clusterVersion)
	r, err := k.client.Versions.GetNodeUpgrades(p, k.auth)

	if err != nil {
		if e, ok := err.(*versions.GetNodeUpgradesDefault); ok && e.Payload != nil && e.Payload.Error != nil && e.Payload.Error.Message != nil {
			return fmt.Errorf("get node_deployment upgrades: %s", *e.Payload.Error.Message)
		}
		return err
	}

	var availableVersions []string
	for _, v := range r.Payload {
		s, ok := v.Version.(string)
		if ok && s == kubeletVersion && !v.RestrictedByKubeletVersion {
			return nil
		}
		availableVersions = append(availableVersions, s)
	}

	return fmt.Errorf("unknown version for node deployment %s, available versions %v", kubeletVersion, availableVersions)
}

func metakubeResourceNodeDeploymentWaitForReady(ctx context.Context, k *metakubeProviderMeta, timeout time.Duration, projectID, clusterID, id string, generation int64) error {
	ensures := 0
	needed := 2
	return resource.RetryContext(ctx, timeout, func() *resource.RetryError {
		p := project.NewGetMachineDeploymentParams().
			WithContext(ctx).
			WithProjectID(projectID).
			WithClusterID(clusterID).
			WithMachineDeploymentID(id)

		r, err := k.client.Project.GetMachineDeployment(p, k.auth)
		if err != nil {
			return resource.RetryableError(fmt.Errorf("unable to get node deployment %v", err))
		}

		if r.Payload.Status.ReadyReplicas < *r.Payload.Spec.Replicas || r.Payload.Status.UnavailableReplicas != 0 {
			k.log.Debugf("waiting for node deployment '%s' to be ready, %+v", id, r.Payload.Status)
			return resource.RetryableError(fmt.Errorf("waiting for node deployment '%s' to be ready", id))
		} else {
			ensures++
		}
		if ensures <= needed {
			k.log.Debugf("looks ok, ensuring")
			return resource.RetryableError(fmt.Errorf("looks ok but check again to ensure machines are not being reconciled."))
		}
		return nil
	})
}

func metakubeResourceNodeDeploymentDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	k := m.(*metakubeProviderMeta)
	projectID := d.Get("project_id").(string)
	clusterID := d.Get("cluster_id").(string)
	p := project.NewDeleteMachineDeploymentParams().
		WithProjectID(projectID).
		WithClusterID(clusterID).
		WithMachineDeploymentID(d.Id())

	_, err := k.client.Project.DeleteMachineDeployment(p, k.auth)
	if err != nil {
		if e, ok := err.(*project.DeleteMachineDeploymentDefault); ok && e.Code() == http.StatusNotFound {
			k.log.Infof("removing node deployment '%s' from terraform state file, could not find the resource", d.Id())
			d.SetId("")
			return nil
		}
		return diag.Errorf("unable to delete node deployment '%s': %s", d.Id(), stringifyResponseError(err))
	}

	err = resource.RetryContext(ctx, d.Timeout(schema.TimeoutDelete), func() *resource.RetryError {
		p := project.NewGetMachineDeploymentParams().
			WithContext(ctx).
			WithProjectID(projectID).
			WithClusterID(clusterID).
			WithMachineDeploymentID(d.Id())

		r, err := k.client.Project.GetMachineDeployment(p, k.auth)
		if err != nil {
			if e, ok := err.(*project.GetMachineDeploymentDefault); ok && e.Code() == http.StatusNotFound {
				k.log.Debugf("node deployment '%s' has been destroyed, returned http code: %d", d.Id(), e.Code())
				d.SetId("")
				return nil
			}
			return resource.NonRetryableError(fmt.Errorf("unable to get node deployment '%s': %s", d.Id(), stringifyResponseError(err)))
		}

		k.log.Debugf("node deployment '%s' deletion in progress, deletionTimestamp: %s",
			d.Id(), r.Payload.DeletionTimestamp.String())
		return resource.RetryableError(fmt.Errorf("node deployment '%s' deletion in progress", d.Id()))
	})
	if err != nil {
		return diag.FromErr(err)
	}
	return nil
}
