package resource_maintenance_cronjob

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/syseleven/go-metakube/client/project"
	"github.com/syseleven/go-metakube/models"
	"github.com/syseleven/terraform-provider-metakube/metakube/common"
)

var (
	_ resource.Resource                = &metakubeMaintenanceCronJob{}
	_ resource.ResourceWithConfigure   = &metakubeMaintenanceCronJob{}
	_ resource.ResourceWithImportState = &metakubeMaintenanceCronJob{}
)

func NewMaintenanceCronJob() resource.Resource {
	return &metakubeMaintenanceCronJob{}
}

type metakubeMaintenanceCronJob struct {
	meta *common.MetaKubeProviderMeta
}

func (r *metakubeMaintenanceCronJob) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_maintenance_cron_job"
}

func (r *metakubeMaintenanceCronJob) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = MaintenanceCronJobSchema(ctx)
}

func (r *metakubeMaintenanceCronJob) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	meta, ok := req.ProviderData.(*common.MetaKubeProviderMeta)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *common.MetaKubeProviderMeta, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.meta = meta
}

func (r *metakubeMaintenanceCronJob) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan MaintenanceCronJobModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, diags := plan.Timeouts.Create(ctx, 20*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID := plan.ProjectID.ValueString()
	clusterID := plan.ClusterID.ValueString()

	if projectID == "" {
		var err error
		projectID, err = common.MetakubeResourceClusterFindProjectID(ctx, clusterID, r.meta)
		if err != nil {
			resp.Diagnostics.AddError("Error finding project", err.Error())
			return
		}
		if projectID == "" {
			r.meta.Log.Infof("owner project for cluster '%s' is not found", clusterID)
			resp.Diagnostics.AddError(
				"Project not found",
				fmt.Sprintf("could not find owner project for cluster with id '%s'", clusterID),
			)
			return
		}
	}

	maintenanceCronJob := &models.MaintenanceCronJob{
		Name: plan.Name.ValueString(),
		Spec: metakubeMaintenanceCronJobExpandSpec(ctx, plan.Spec),
	}

	if err := common.MetakubeResourceClusterWaitForReady(ctx, r.meta, createTimeout, projectID, clusterID, ""); err != nil {
		resp.Diagnostics.AddError("Cluster not ready", fmt.Sprintf("cluster is not ready: %v", err))
		return
	}

	p := project.NewCreateMaintenanceCronJobParams().
		WithContext(ctx).
		WithProjectID(projectID).
		WithClusterID(clusterID).
		WithBody(maintenanceCronJob)

	var id string
	err := common.RetryContext(ctx, createTimeout, func() *common.RetryError {
		result, err := r.meta.Client.Project.CreateMaintenanceCronJob(p, r.meta.Auth)
		if err != nil {
			e := common.StringifyResponseError(err)
			if strings.Contains(e, "failed calling webhook") || strings.Contains(e, "Cluster components are not ready yet") {
				return common.RetryableError(fmt.Errorf("%v", e))
			}
			return common.NonRetryableError(fmt.Errorf("%v", e))
		}
		id = result.Payload.Name
		return nil
	})
	if err != nil {
		resp.Diagnostics.AddError("Create failed", fmt.Sprintf("create a maintenance cron job: %v", err))
		return
	}

	plan.ID = types.StringValue(id)
	plan.ProjectID = types.StringValue(projectID)

	if err := metakubeResourceMaintenanceCronJobWaitForReady(ctx, r.meta, createTimeout, projectID, clusterID, id); err != nil {
		resp.Diagnostics.AddError("Wait for ready failed", err.Error())
		return
	}

	// Read back the resource
	readP := project.NewGetMaintenanceCronJobParams().
		WithContext(ctx).
		WithProjectID(projectID).
		WithClusterID(clusterID).
		WithMaintenanceCronJobID(id)

	readResult, err := r.meta.Client.Project.GetMaintenanceCronJob(readP, r.meta.Auth)
	if err != nil {
		resp.Diagnostics.AddError("Read failed", fmt.Sprintf("unable to read maintenance cron job after creation: %s", common.StringifyResponseError(err)))
		return
	}

	plan.Name = types.StringValue(readResult.Payload.Name)
	plan.CreationTimestamp = types.StringValue(readResult.Payload.CreationTimestamp.String())
	plan.DeletionTimestamp = types.StringValue(readResult.Payload.DeletionTimestamp.String())

	resp.Diagnostics.Append(metakubeMaintenanceCronJobFlattenSpec(ctx, &plan, readResult.Payload.Spec)...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r *metakubeMaintenanceCronJob) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data MaintenanceCronJobModel

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID := data.ProjectID.ValueString()
	clusterID := data.ClusterID.ValueString()

	p := project.NewGetMaintenanceCronJobParams().
		WithContext(ctx).
		WithProjectID(projectID).
		WithClusterID(clusterID).
		WithMaintenanceCronJobID(data.ID.ValueString())

	result, err := r.meta.Client.Project.GetMaintenanceCronJob(p, r.meta.Auth)
	if err != nil {
		if e, ok := err.(*project.GetMaintenanceCronJobDefault); ok && e.Code() == http.StatusNotFound {
			r.meta.Log.Infof("removing maintenance cron job '%s' from terraform state file, could not find the resource", data.ID.ValueString())
			resp.State.RemoveResource(ctx)
			return
		}
		if _, ok := err.(*project.GetMaintenanceCronJobForbidden); ok {
			r.meta.Log.Infof("removing maintenance cron job '%s' from terraform state file, access forbidden", data.ID.ValueString())
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("API Error", fmt.Sprintf(
			"unable to get maintenance cron job '%s/%s/%s': %s", projectID, clusterID, data.ID.ValueString(), common.StringifyResponseError(err)))
		return
	}

	data.Name = types.StringValue(result.Payload.Name)
	data.CreationTimestamp = types.StringValue(result.Payload.CreationTimestamp.String())
	data.DeletionTimestamp = types.StringValue(result.Payload.DeletionTimestamp.String())

	resp.Diagnostics.Append(metakubeMaintenanceCronJobFlattenSpec(ctx, &data, result.Payload.Spec)...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *metakubeMaintenanceCronJob) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan MaintenanceCronJobModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateTimeout, diags := plan.Timeouts.Update(ctx, 20*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID := plan.ProjectID.ValueString()
	clusterID := plan.ClusterID.ValueString()
	cronJobID := plan.ID.ValueString()

	maintenanceCronJob := &models.MaintenanceCronJob{
		Spec: metakubeMaintenanceCronJobExpandSpec(ctx, plan.Spec),
	}

	p := project.NewPatchMaintenanceCronJobParams().
		WithContext(ctx).
		WithProjectID(projectID).
		WithClusterID(clusterID).
		WithMaintenanceCronJobID(cronJobID).
		WithPatch(maintenanceCronJob)

	_, err := r.meta.Client.Project.PatchMaintenanceCronJob(p, r.meta.Auth)
	if err != nil {
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("unable to update a maintenance cron job: %v", common.StringifyResponseError(err)))
		return
	}

	if err := metakubeResourceMaintenanceCronJobWaitForReady(ctx, r.meta, updateTimeout, projectID, clusterID, cronJobID); err != nil {
		resp.Diagnostics.AddError("Wait for ready failed", err.Error())
		return
	}

	// Read back the resource
	readP := project.NewGetMaintenanceCronJobParams().
		WithContext(ctx).
		WithProjectID(projectID).
		WithClusterID(clusterID).
		WithMaintenanceCronJobID(cronJobID)

	readResult, err := r.meta.Client.Project.GetMaintenanceCronJob(readP, r.meta.Auth)
	if err != nil {
		resp.Diagnostics.AddError("Read failed", fmt.Sprintf("unable to read maintenance cron job after update: %s", common.StringifyResponseError(err)))
		return
	}

	plan.Name = types.StringValue(readResult.Payload.Name)
	plan.CreationTimestamp = types.StringValue(readResult.Payload.CreationTimestamp.String())
	plan.DeletionTimestamp = types.StringValue(readResult.Payload.DeletionTimestamp.String())

	resp.Diagnostics.Append(metakubeMaintenanceCronJobFlattenSpec(ctx, &plan, readResult.Payload.Spec)...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r *metakubeMaintenanceCronJob) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state MaintenanceCronJobModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteTimeout, diags := state.Timeouts.Delete(ctx, 20*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID := state.ProjectID.ValueString()
	clusterID := state.ClusterID.ValueString()
	cronJobID := state.ID.ValueString()

	p := project.NewDeleteMaintenanceCronJobParams().
		WithProjectID(projectID).
		WithClusterID(clusterID).
		WithMaintenanceCronJobID(cronJobID)

	_, err := r.meta.Client.Project.DeleteMaintenanceCronJob(p, r.meta.Auth)
	if err != nil {
		if e, ok := err.(*project.DeleteMaintenanceCronJobDefault); ok && e.Code() == http.StatusNotFound {
			r.meta.Log.Infof("maintenance cron job '%s' already deleted", cronJobID)
			return
		}
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("unable to delete maintenance cron job '%s': %s", cronJobID, common.StringifyResponseError(err)))
		return
	}

	err = common.RetryContext(ctx, deleteTimeout, func() *common.RetryError {
		getP := project.NewGetMaintenanceCronJobParams().
			WithContext(ctx).
			WithProjectID(projectID).
			WithClusterID(clusterID).
			WithMaintenanceCronJobID(cronJobID)

		result, err := r.meta.Client.Project.GetMaintenanceCronJob(getP, r.meta.Auth)
		if err != nil {
			if e, ok := err.(*project.GetMaintenanceCronJobDefault); ok && e.Code() == http.StatusNotFound {
				r.meta.Log.Debugf("maintenance cron job '%s' has been destroyed, returned http code: %d", cronJobID, e.Code())
				return nil
			}
			return common.NonRetryableError(fmt.Errorf("unable to get maintenance cron job '%s': %s", cronJobID, common.StringifyResponseError(err)))
		}

		r.meta.Log.Debugf("maintenance cron job '%s' deletion in progress, deletionTimestamp: %s",
			cronJobID, result.Payload.DeletionTimestamp)
		return common.RetryableError(fmt.Errorf("maintenance cron job '%s' deletion in progress", cronJobID))
	})
	if err != nil {
		resp.Diagnostics.AddError("Delete failed", err.Error())
	}
}

func (r *metakubeMaintenanceCronJob) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")

	if len(parts) != 3 {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"please provide resource identifier in format 'project_id:cluster_id:maintenance_cronjob_id'",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[2])...)
}

func metakubeResourceMaintenanceCronJobWaitForReady(ctx context.Context, k *common.MetaKubeProviderMeta, timeout time.Duration, projectID, clusterID, id string) error {
	return common.RetryContext(ctx, timeout, func() *common.RetryError {
		p := project.NewGetMaintenanceCronJobParams().
			WithContext(ctx).
			WithProjectID(projectID).
			WithClusterID(clusterID).
			WithMaintenanceCronJobID(id)

		result, err := k.Client.Project.GetMaintenanceCronJob(p, k.Auth)
		if err != nil {
			return common.RetryableError(fmt.Errorf("unable to get maintenance cron job %s", common.StringifyResponseError(err)))
		}

		if result.Payload.Name == "" || result.Payload.Spec.MaintenanceJobTemplate == nil || result.Payload.Spec.MaintenanceJobTemplate.Type == "" {
			return common.RetryableError(fmt.Errorf("waiting for maintenance cron job '%s' to be ready", id))
		}

		return nil
	})
}
