package resource_cluster_role_binding

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
	"github.com/syseleven/terraform-provider-metakube/metakube/common"
)

var (
	_ resource.Resource                = &metakubeClusterRoleBinding{}
	_ resource.ResourceWithConfigure   = &metakubeClusterRoleBinding{}
	_ resource.ResourceWithImportState = &metakubeClusterRoleBinding{}
)

func NewClusterRoleBinding() resource.Resource {
	return &metakubeClusterRoleBinding{}
}

type metakubeClusterRoleBinding struct {
	meta *common.MetaKubeProviderMeta
}

func (r *metakubeClusterRoleBinding) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster_role_binding"
}

func (r *metakubeClusterRoleBinding) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = ClusterRoleBindingSchema()
}

func (r *metakubeClusterRoleBinding) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *metakubeClusterRoleBinding) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ClusterRoleBindingModel

	diags := req.Plan.Get(ctx, &plan)

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID := plan.ProjectID.ValueString()
	clusterID := plan.ClusterID.ValueString()
	clusterRoleName := plan.ClusterRoleName.ValueString()

	subjects := metakubeClusterRoleBindingExpandSubjects(ctx, plan.Subject)
	for _, sub := range subjects {
		timeout := 20 * time.Minute
		deadline := time.Now().Add(timeout)
		for time.Now().Before(deadline) {
			params := project.NewBindUserToClusterRoleV2Params().
				WithContext(ctx).
				WithProjectID(projectID).
				WithClusterID(clusterID).
				WithRoleID(clusterRoleName).
				WithBody(&sub)
			_, err := r.meta.Client.Project.BindUserToClusterRoleV2(params, r.meta.Auth)
			if err != nil {
				e, ok := err.(*project.BindUserToClusterRoleV2Default)
				if ok && (e.Code() == http.StatusConflict || e.Code() == http.StatusNotFound) {
					time.Sleep(5 * time.Second)
					continue
				}
				resp.Diagnostics.AddError("Timeout creating cluster role binding", fmt.Sprintf(
					"Timeout waiting for cluster role binding for cluster '%s' to be created: %s", clusterID, common.StringifyResponseError(err)))
				return
			}
		}
	}

	plan.ID = types.StringValue(clusterRoleName)

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r *metakubeClusterRoleBinding) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ClusterRoleBindingModel

	diags := req.State.Get(ctx, &data)

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := project.NewListClusterRoleBindingV2Params().
		WithContext(ctx).
		WithProjectID(data.ProjectID.ValueString()).
		WithClusterID(data.ClusterID.ValueString())
	ret, err := r.meta.Client.Project.ListClusterRoleBindingV2(params, r.meta.Auth)
	if err != nil {
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Failed to list cluster role bindings: %s", common.StringifyResponseError(err)))
		return
	}

	clusterRoleName := data.ClusterRoleName.ValueString()
	for _, item := range ret.Payload {
		if item.RoleRefName == clusterRoleName && len(item.Subjects) != 0 {
			resp.Diagnostics.Append(metakubeClusterRoleBindingFlattenSubjects(ctx, &data, item.Subjects)...)
			if resp.Diagnostics.HasError() {
				return
			}

			data.ID = types.StringValue(item.RoleRefName)
			data.ClusterRoleName = types.StringValue(item.RoleRefName)

			diags = resp.State.Set(ctx, &data)
			resp.Diagnostics.Append(diags...)

			return
		}
	}

	data.ID = types.StringNull()

	resp.State.RemoveResource(ctx)
}

func (r *metakubeClusterRoleBinding) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ClusterRoleBindingModel

	diags := req.State.Get(ctx, &state)

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID := state.ProjectID.ValueString()
	clusterID := state.ClusterID.ValueString()
	clusterRoleName := state.ClusterRoleName.ValueString()

	subjects := metakubeClusterRoleBindingExpandSubjects(ctx, state.Subject)
	for _, sub := range subjects {
		params := project.NewUnbindUserFromClusterRoleBindingV2Params().
			WithContext(ctx).
			WithProjectID(projectID).
			WithClusterID(clusterID).
			WithRoleID(clusterRoleName).
			WithBody(&sub)
		_, err := r.meta.Client.Project.UnbindUserFromClusterRoleBindingV2(params, r.meta.Auth)
		if err != nil {
			resp.Diagnostics.AddError("API Error", fmt.Sprintf("Failed to delete cluster role bindings: %s", common.StringifyResponseError(err)))
			return
		}
	}
}

func (r *metakubeClusterRoleBinding) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")

	if len(parts) != 3 {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Please provide resource identifier in format 'project_id:cluster_id:cluster_role_name'",
		)
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[2])...)
}

func (r *metakubeClusterRoleBinding) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
}
