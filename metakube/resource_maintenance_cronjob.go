package metakube

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/syseleven/go-metakube/client/project"
	"github.com/syseleven/go-metakube/models"
)

func metakubeResourceMaintenanceCronJob() *schema.Resource {
	return &schema.Resource{
		CreateContext: metakubeResourceMaintenanceCronJobCreate,
		ReadContext:   metakubeResourceMaintenanceCronJobRead,
		UpdateContext: metakubeResourceMaintenanceCronJobUpdate,
		DeleteContext: metakubeResourceMaintenanceCronJobDelete,

		Importer: &schema.ResourceImporter{
			StateContext: importResourceWithOptionalProject("maintenance_cronjob_id"),
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(20 * time.Minute),
			Update: schema.DefaultTimeout(20 * time.Minute),
			Delete: schema.DefaultTimeout(20 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Reference project identifier",
			},

			"cluster_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				// TODO: update descriptions
				Description: "Cluster that maintenance cron job belongs to",
			},

			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Maintenance cron job name",
			},

			"spec": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Required:    true,
				Description: "Maintenance cron job specification",
				Elem: &schema.Resource{
					Schema: metakubeResourceMaintenanceCronJobSpecFields(),
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

func metakubeResourceMaintenanceCronJobCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
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
			return diag.Errorf("could not find owner project for cluster with id '%s'", clusterID)
		}
	}

	maintenanceCronJob := &models.MaintenanceCronJob{
		Name: d.Get("name").(string),
		Spec: metakubeMaintenanceCronJobExpandSpec(d.Get("spec").([]interface{})),
	}

	// TODO not sure to remove this
	if err := metakubeResourceClusterWaitForReady(ctx, k, d.Timeout(schema.TimeoutCreate), projectID, clusterID, ""); err != nil {
		return diag.Errorf("cluster is not ready: %v", err)
	}

	p := project.NewCreateMaintenanceCronJobParams().
		WithContext(ctx).
		WithProjectID(projectID).
		WithClusterID(clusterID).
		WithBody(maintenanceCronJob)

	var id models.UID
	err := retry.RetryContext(ctx, d.Timeout(schema.TimeoutCreate), func() *retry.RetryError {
		r, err := k.client.Project.CreateMaintenanceCronJob(p, k.auth)
		if err != nil {
			e := stringifyResponseError(err)
			if strings.Contains(e, "failed calling webhook") || strings.Contains(e, "Cluster components are not ready yet") {
				return retry.RetryableError(fmt.Errorf("%v", e))
			}
			return retry.NonRetryableError(fmt.Errorf("%v", e))
		}
		id = models.UID(r.Payload.Name)
		return nil
	})
	if err != nil {
		return diag.Errorf("create a maintenance cron job: %v", err)
	}
	d.SetId(string(id))
	d.Set("project_id", projectID)

	if err := metakubeResourceMaintenanceCronJobWaitForReady(ctx, k, d.Timeout(schema.TimeoutCreate), projectID, clusterID, string(id)); err != nil {
		return diag.FromErr(err)
	}

	return metakubeResourceMaintenanceCronJobRead(ctx, d, m)
}

func metakubeResourceMaintenanceCronJobUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	k := m.(*metakubeProviderMeta)
	projectID := d.Get("project_id").(string)
	clusterID := d.Get("cluster_id").(string)

	maintenanceCronJob := &models.MaintenanceCronJob{
		Spec: metakubeMaintenanceCronJobExpandSpec(d.Get("spec").([]interface{})),
	}

	p := project.NewPatchMaintenanceCronJobParams()
	p.SetContext(ctx)
	p.SetProjectID(projectID)
	p.SetClusterID(clusterID)
	p.SetMaintenanceCronJobID(d.Id())
	p.SetPatch(maintenanceCronJob)
	_, err := k.client.Project.PatchMaintenanceCronJob(p, k.auth)
	if err != nil {
		return diag.Errorf("unable to update a maintenance cron job: %v", stringifyResponseError(err))
	}

	if err := metakubeResourceMaintenanceCronJobWaitForReady(ctx, k, d.Timeout(schema.TimeoutUpdate), projectID, clusterID, d.Id()); err != nil {
		return diag.FromErr(err)
	}

	return metakubeResourceMaintenanceCronJobRead(ctx, d, m)
}

func metakubeResourceMaintenanceCronJobRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	k := m.(*metakubeProviderMeta)
	projectID := d.Get("project_id").(string)
	clusterID := d.Get("cluster_id").(string)
	p := project.NewGetMaintenanceCronJobParams().
		WithContext(ctx).
		WithProjectID(projectID).
		WithClusterID(clusterID).
		WithMaintenanceCronJobID(d.Id())

	r, err := k.client.Project.GetMaintenanceCronJob(p, k.auth)
	if err != nil {
		if e, ok := err.(*project.GetMaintenanceCronJobDefault); ok && e.Code() == http.StatusNotFound {
			k.log.Infof("removing maintenance cron job '%s' from terraform state file, could not find the resource", d.Id())
			d.SetId("")
			return nil
		}
		if _, ok := err.(*project.GetMaintenanceCronJobForbidden); ok {
			k.log.Infof("removing maintenance cron job '%s' from terraform state file, access forbidden", d.Id())
			d.SetId("")
			return nil
		}
		return diag.Errorf("unable to get maintenance cron job '%s/%s/%s': %s", projectID, clusterID, d.Id(), stringifyResponseError(err))
	}

	_ = d.Set("name", r.Payload.Name)

	_ = d.Set("spec", metakubeMaintenanceCronJobFlattenSpec(r.Payload.Spec))

	_ = d.Set("creation_timestamp", r.Payload.CreationTimestamp.String())

	_ = d.Set("deletion_timestamp", r.Payload.DeletionTimestamp.String())

	return nil
}

func metakubeResourceMaintenanceCronJobDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	k := m.(*metakubeProviderMeta)
	projectID := d.Get("project_id").(string)
	clusterID := d.Get("cluster_id").(string)
	p := project.NewDeleteMaintenanceCronJobParams().
		WithProjectID(projectID).
		WithClusterID(clusterID).
		WithMaintenanceCronJobID(d.Id())

	_, err := k.client.Project.DeleteMaintenanceCronJob(p, k.auth)
	if err != nil {
		if e, ok := err.(*project.DeleteMaintenanceCronJobDefault); ok && e.Code() == http.StatusNotFound {
			k.log.Infof("removing maintenance cron job '%s' from terraform state file, could not find the resource", d.Id())
			d.SetId("")
			return nil
		}
		return diag.Errorf("unable to delete maintenance cron job '%s': %s", d.Id(), stringifyResponseError(err))
	}

	err = retry.RetryContext(ctx, d.Timeout(schema.TimeoutDelete), func() *retry.RetryError {
		p := project.NewGetMaintenanceCronJobParams().
			WithContext(ctx).
			WithProjectID(projectID).
			WithClusterID(clusterID).
			WithMaintenanceCronJobID(d.Id())

		r, err := k.client.Project.GetMaintenanceCronJob(p, k.auth)
		if err != nil {
			if e, ok := err.(*project.GetMaintenanceCronJobDefault); ok && e.Code() == http.StatusNotFound {
				k.log.Debugf("maintenance cron job '%s' has been destroyed, returned http code: %d", d.Id(), e.Code())
				d.SetId("")
				return nil
			}
			return retry.NonRetryableError(fmt.Errorf("unable to get maintenance cron job '%s': %s", d.Id(), stringifyResponseError(err)))
		}

		k.log.Debugf("maintenance cron job '%s' deletion in progress, deletionTimestamp: %s",
			d.Id(), r.Payload.DeletionTimestamp)
		return retry.RetryableError(fmt.Errorf("maintenance cron job '%s' deletion in progress", d.Id()))
	})
	if err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func metakubeResourceMaintenanceCronJobWaitForReady(ctx context.Context, k *metakubeProviderMeta, timeout time.Duration, projectID, clusterID, id string) error {
	return retry.RetryContext(ctx, timeout, func() *retry.RetryError {
		p := project.NewGetMaintenanceCronJobParams().
			WithContext(ctx).
			WithProjectID(projectID).
			WithClusterID(clusterID).
			WithMaintenanceCronJobID(id)

		r, err := k.client.Project.GetMaintenanceCronJob(p, k.auth)
		if err != nil {
			return retry.RetryableError(fmt.Errorf("unable to get maintenance cron job %s", stringifyResponseError(err)))
		}

		if r.Payload.Name == "" || r.Payload.Spec.MaintenanceJobTemplate == nil || r.Payload.Spec.MaintenanceJobTemplate.Type == "" {
			return retry.RetryableError(fmt.Errorf("waiting for maintenance cron job '%s' to be ready", id))
		}

		return nil
	})
}
