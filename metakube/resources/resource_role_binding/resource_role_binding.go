package resource_role_binding

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
	_ resource.Resource                = &metakubeRoleBinding{}
	_ resource.ResourceWithConfigure   = &metakubeRoleBinding{}
	_ resource.ResourceWithImportState = &metakubeRoleBinding{}
)

func NewRoleBinding() resource.Resource {
	return &metakubeRoleBinding{}
}

type metakubeRoleBinding struct {
	meta *common.MetaKubeProviderMeta
}

func (r *metakubeRoleBinding) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role_binding"
}

func (r *metakubeRoleBinding) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = RoleBindingSchema()
}

func (r *metakubeRoleBinding) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *metakubeRoleBinding) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RoleBindingModel

	diags := req.State.Get(ctx, &data)

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := project.NewListRoleBindingV2Params().
		WithContext(ctx).
		WithProjectID(data.ProjectID.ValueString()).
		WithClusterID(data.ClusterID.ValueString())
	ret, err := r.meta.Client.Project.ListRoleBindingV2(params, r.meta.Auth)
	if err != nil {
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Failed to list role bindings: %s", common.StringifyResponseError(err)))
		return
	}

	namespace := data.Namespace.ValueString()
	roleName := data.RoleName.ValueString()
	for _, item := range ret.Payload {
		if item.Namespace == namespace && item.RoleRefName == roleName && len(item.Subjects) != 0 {
			resp.Diagnostics.Append(metakubeClusterRoleBindingFlattenSubjects(ctx, &data, item.Subjects)...)
			if resp.Diagnostics.HasError() {
				return
			}

			data.ID = types.StringValue(item.Namespace + ":" + item.RoleRefName)
			data.Namespace = types.StringValue(item.Namespace)
			data.RoleName = types.StringValue(item.RoleRefName)

			diags = resp.State.Set(ctx, &data)
			resp.Diagnostics.Append(diags...)

			return
		}
	}

	data.ID = types.StringNull()

	resp.State.RemoveResource(ctx)
}

func (r *metakubeRoleBinding) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RoleBindingModel

	diags := req.Plan.Get(ctx, &plan)

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID := plan.ProjectID.ValueString()
	clusterID := plan.ClusterID.ValueString()
	namespace := plan.Namespace.ValueString()
	roleName := plan.RoleName.ValueString()

	subjects := metakubeRoleBindingExpandSubjects(ctx, plan.Subject)
	for _, sub := range subjects {
		timeout := 20 * time.Minute
		deadline := time.Now().Add(timeout)
		for time.Now().Before(deadline) {
			params := project.NewBindUserToRoleV2Params().
				WithContext(ctx).
				WithProjectID(projectID).
				WithClusterID(clusterID).
				WithNamespace(namespace).
				WithRoleID(roleName).
				WithBody(&sub)
			_, err := r.meta.Client.Project.BindUserToRoleV2(params, r.meta.Auth)
			if err != nil {
				e, ok := err.(*project.BindUserToRoleV2Default)
				if ok && (e.Code() == http.StatusConflict || e.Code() == http.StatusNotFound) {
					time.Sleep(5 * time.Second)
					continue
				}
				resp.Diagnostics.AddError("Timeout creating role binding", fmt.Sprintf(
					"Timeout waiting for role binding for cluster '%s' to be created: %s", clusterID, common.StringifyResponseError(err)))
				return
			}
		}
	}

	plan.ID = types.StringValue(namespace + ":" + roleName)

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r *metakubeRoleBinding) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RoleBindingModel

	diags := req.State.Get(ctx, &state)

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID := state.ProjectID.ValueString()
	clusterID := state.ClusterID.ValueString()
	namespace := state.Namespace.ValueString()
	roleName := state.RoleName.ValueString()

	subjects := metakubeRoleBindingExpandSubjects(ctx, state.Subject)
	for _, sub := range subjects {
		params := project.NewUnbindUserFromRoleBindingV2Params().
			WithContext(ctx).
			WithProjectID(projectID).
			WithClusterID(clusterID).
			WithNamespace(namespace).
			WithRoleID(roleName).
			WithBody(&sub)
		_, err := r.meta.Client.Project.UnbindUserFromRoleBindingV2(params, r.meta.Auth)
		if err != nil {
			resp.Diagnostics.AddError("API Error", fmt.Sprintf("Failed to delete role bindings: %s", common.StringifyResponseError(err)))
			return
		}
	}
}

func (r *metakubeRoleBinding) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")

	if len(parts) != 4 {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Please provide resource identifier in format project_id:cluster_id:role_namespace:role_name'",
		)
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[2]+":"+parts[3])...)
}

func (r *metakubeRoleBinding) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
}
