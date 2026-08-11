package resource_cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/syseleven/go-metakube/client/datacenter"
	"github.com/syseleven/go-metakube/client/project"
	"github.com/syseleven/go-metakube/models"
	"github.com/syseleven/terraform-provider-metakube/metakube/common"
)

var (
	_ resource.Resource                 = &clusterResource{}
	_ resource.ResourceWithConfigure    = &clusterResource{}
	_ resource.ResourceWithImportState  = &clusterResource{}
	_ resource.ResourceWithUpgradeState = &clusterResource{}
	_ resource.ResourceWithModifyPlan   = &clusterResource{}
)

// NewClusterResource returns a MetaKube cluster resource.
func NewClusterResource() resource.Resource {
	return &clusterResource{}
}

type clusterResource struct {
	meta *common.MetaKubeProviderMeta
}

func (r *clusterResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster"
}

func (r *clusterResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = ClusterResourceSchema(ctx)
}

// ModifyPlan preserves computed kubeconfigs when the authentication settings
// are unchanged. When authentication changes, their unknown values remain in
// the plan so Read can replace them with credentials for the new settings.
func (r *clusterResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Skip create and destroy plans because no prior computed value can be reused.
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}
	// Decode the planned and prior cluster models.
	var plan, state ClusterModel
	if diags := req.Plan.Get(ctx, &plan); diags.HasError() {
		return
	}
	if diags := req.State.Get(ctx, &state); diags.HasError() {
		return
	}

	planSpec, planOk := getClusterSpecModel(ctx, &plan)
	stateSpec, stateOk := getClusterSpecModel(ctx, &state)

	var authChanged bool
	switch {
	case planOk && stateOk:
		authChanged = !planSpec.SyselevenAuth.Equal(stateSpec.SyselevenAuth)
	case planOk != stateOk:
		authChanged = true
	default:
		authChanged = false
	}

	if !authChanged {
		// Reuse prior kubeconfigs to avoid a diff when authentication is unchanged.
		if plan.OIDCKubeConfig.IsUnknown() {
			resp.Plan.SetAttribute(ctx, path.Root("oidc_kube_config"), state.OIDCKubeConfig)
		}
		if plan.KubeLoginKubeConfig.IsUnknown() {
			resp.Plan.SetAttribute(ctx, path.Root("kube_login_kube_config"), state.KubeLoginKubeConfig)
		}
	}
}

func (r *clusterResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *clusterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Decode the planned cluster from the request.
	var plan ClusterModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(metakubeResourceClusterValidateClusterFields(ctx, &plan, r.meta, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Expand Terraform values into the API create payload.
	dcname := plan.DCName.ValueString()
	clusterSpec := metakubeResourceClusterExpandSpec(ctx, &plan, dcname, func(_ string) bool { return true })
	clusterLabels := expandLabelsFromModel(plan.Labels)

	createClusterSpec := &models.CreateClusterSpec{
		Cluster: &models.Cluster{
			Name:   plan.Name.ValueString(),
			Spec:   clusterSpec,
			Labels: clusterLabels,
		},
	}

	if n := clusterSpec.ClusterNetwork; n != nil {
		if v := clusterSpec.ClusterNetwork.Pods; v != nil {
			if len(v.CIDRBlocks) == 1 {
				createClusterSpec.PodsCIDR = v.CIDRBlocks[0]
			}
			if len(v.CIDRBlocks) > 1 {
				resp.Diagnostics.AddWarning("Multiple pods CIDRs", "API returned multiple pods CIDRs")
			}
		}
		if v := clusterSpec.ClusterNetwork.Services; v != nil {
			if len(v.CIDRBlocks) == 1 {
				createClusterSpec.ServicesCIDR = v.CIDRBlocks[0]
			}
			if len(v.CIDRBlocks) > 1 {
				resp.Diagnostics.AddWarning("Multiple services CIDRs", "API returned multiple services CIDRs")
			}
		}
	}

	// Require the node-side SSH agent before assigning managed SSH keys.
	sshkeys := expandSSHKeysFromModel(plan.SSHKeys)
	if len(sshkeys) > 0 {
		sshAgentEnabled := getSSHAgentEnabled(ctx, &plan)
		if !sshAgentEnabled {
			resp.Diagnostics.AddAttributeError(
				path.Root("spec").AtName("enable_ssh_agent"),
				"SSH Agent must be enabled",
				"SSH Agent must be enabled in order to automatically manage ssh keys",
			)
			return
		}
	}

	if resp.Diagnostics.HasError() {
		return
	}

	projectID := plan.ProjectID.ValueString()
	p := project.NewCreateClusterV2Params().WithProjectID(projectID).WithBody(createClusterSpec)
	result, err := r.meta.Client.Project.CreateClusterV2(p, r.meta.Auth)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create cluster",
			fmt.Sprintf("Unable to create cluster for project '%s': %s", projectID, common.StringifyResponseError(err)),
		)
		return
	}

	plan.ID = types.StringValue(result.Payload.ID)

	if err := r.assignSSHKeysToCluster(projectID, result.Payload.ID, sshkeys); err != nil {
		resp.Diagnostics.AddError("Failed to assign SSH keys", err.Error())
		return
	}

	createTimeout, diags := plan.Timeouts.Create(ctx, 20*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := common.MetakubeResourceClusterWaitForReady(ctx, r.meta, createTimeout, projectID, plan.ID.ValueString(), ""); err != nil {
		// read from API and set the partial state on error, so Terraform can reconcile or delete the created cluster.
		resp.Diagnostics.Append(r.readClusterIntoModel(ctx, &plan)...)
		resp.Diagnostics.AddError(
			"Cluster not ready",
			fmt.Sprintf("Cluster '%s' is not ready: %v", result.Payload.ID, err),
		)
		resp.State.Set(ctx, &plan)
		return
	}

	// read from API and set the state
	resp.Diagnostics.Append(r.readClusterIntoModel(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r *clusterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Decode the current cluster from Terraform state.
	var state ClusterModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Fetch and map the remote cluster into the Terraform model.
	resp.Diagnostics.Append(r.readClusterIntoModel(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.ID.IsNull() || state.ID.ValueString() == "" {
		// Remove state only when the refresh proves the remote cluster is absent.
		resp.State.RemoveResource(ctx)
		return
	}

	// Persist the refreshed cluster to Terraform state.
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// Update sends desired cluster fields as one JSON merge patch, then reconciles
// SSH-key assignments through separate API calls. Those operations are not
// atomic: if SSH-key reconciliation fails after the patch succeeds, Terraform
// keeps the prior state and the next refresh observes the remote cluster before
// a retry.
func (r *clusterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Decode the planned and prior cluster models from the request.
	var plan, state ClusterModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID := plan.ProjectID.ValueString()

	// Fetch the live cluster before validating an in-place update.
	cluster, ok, err := common.MetakubeGetCluster(ctx, projectID, state.ID.ValueString(), r.meta)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get cluster", err.Error())
		return
	}
	if !ok {
		resp.State.RemoveResource(ctx)
		return
	}

	planVersion := getVersionFromModel(ctx, &plan)
	stateVersion := getVersionFromModel(ctx, &state)
	if planVersion != stateVersion {
		// Validate version transitions only when the requested version changes.
		r.meta.Log.Debugf("validating version change")
		resp.Diagnostics.Append(metakubeResourceClusterValidateVersionUpgrade(ctx, projectID, planVersion, cluster, r.meta)...)
	}

	resp.Diagnostics.Append(metakubeResourceClusterValidateClusterFields(ctx, &plan, r.meta, true)...)
	resp.Diagnostics.Append(r.validateDatacenter(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build and send an explicit JSON merge patch for changed cluster fields.
	patch, patchDiags := buildClusterPatch(ctx, plan, state)
	resp.Diagnostics.Append(patchDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(patch) > 0 {
		if err := r.sendPatchRequest(ctx, projectID, state.ID.ValueString(), patch); err != nil {
			resp.Diagnostics.AddError("Failed to patch cluster", err.Error())
			return
		}
	}

	if !plan.SSHKeys.Equal(state.SSHKeys) {
		if err := r.updateClusterSSHKeys(ctx, &plan, &state); err != nil {
			resp.Diagnostics.AddError("Failed to update SSH keys", err.Error())
			return
		}
	}

	configuredVersion := getVersionFromModel(ctx, &plan)
	updateTimeout, diags := plan.Timeouts.Update(ctx, 20*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := common.MetakubeResourceClusterWaitForReady(ctx, r.meta, updateTimeout, projectID, state.ID.ValueString(), configuredVersion); err != nil {
		resp.Diagnostics.AddError(
			"Cluster not ready",
			fmt.Sprintf("Cluster '%s' is not ready: %v", state.ID.ValueString(), err),
		)
		return
	}

	plan.ID = state.ID

	// Refresh the updated cluster to capture computed API values.
	resp.Diagnostics.Append(r.readClusterIntoModel(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Persist the updated cluster to Terraform state.
	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

// Delete retries conflicts while another cluster operation is active, then
// polls an accepted deletion until the API returns not found. A not-found
// response means deletion is complete. The API's dedicated forbidden response
// is also treated as terminal during request submission and as transient while
// polling; returning without diagnostics allows Terraform to remove the state.
func (r *clusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Decode cluster identifiers from Terraform state.
	var state ClusterModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID := state.ProjectID.ValueString()
	clusterID := state.ID.ValueString()

	deleteTimeout, diags := state.Timeouts.Delete(ctx, 20*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	timeoutTimer := time.NewTimer(deleteTimeout)
	defer timeoutTimer.Stop()

	deleteSent := false
	deleteStartTime := time.Now()

	// Retry request conflicts, then poll until not found confirms deletion.
	for {
		shouldWait := false

		if !deleteSent {
			deleteParams := project.NewDeleteClusterV2Params()
			deleteParams.SetContext(ctx)
			deleteParams.SetProjectID(projectID)
			deleteParams.SetClusterID(clusterID)

			_, err := r.meta.Client.Project.DeleteClusterV2(deleteParams, r.meta.Auth)
			if err != nil {
				if e, ok := err.(*project.DeleteClusterV2Default); ok {
					if e.Code() == http.StatusConflict {
						shouldWait = true
					} else if e.Code() == http.StatusNotFound {
						return
					} else {
						resp.Diagnostics.AddError(
							"Unable to delete cluster",
							fmt.Sprintf("Unable to delete cluster '%s': %s", clusterID, common.StringifyResponseError(err)),
						)
						return
					}
				} else if _, ok := err.(*project.DeleteClusterV2Forbidden); ok {
					return
				} else {
					resp.Diagnostics.AddError(
						"Unable to delete cluster",
						fmt.Sprintf("Unable to delete cluster '%s': %s", clusterID, common.StringifyResponseError(err)),
					)
					return
				}
			} else {
				deleteSent = true
			}
		}

		if deleteSent && !shouldWait {
			// Check whether the accepted asynchronous deletion has completed.
			getParams := project.NewGetClusterV2Params()
			getParams.SetContext(ctx)
			getParams.SetProjectID(projectID)
			getParams.SetClusterID(clusterID)

			result, err := r.meta.Client.Project.GetClusterV2(getParams, r.meta.Auth)
			if err != nil {
				if e, ok := err.(*project.GetClusterV2Default); ok {
					if e.Code() == http.StatusNotFound {
						r.meta.Log.Debugf("cluster '%s' has been destroyed, returned http code: %d", clusterID, e.Code())
						return
					}
					if e.Code() == http.StatusInternalServerError {
						shouldWait = true
					}
				}
				if _, ok := err.(*project.GetClusterV2Forbidden); ok {
					shouldWait = true
				}
				if !shouldWait {
					resp.Diagnostics.AddError(
						"Unable to get cluster",
						fmt.Sprintf("Unable to get cluster '%s': %v", clusterID, err),
					)
					return
				}
			} else {
				r.meta.Log.Debugf("cluster '%s' deletion in progress, deletionTimestamp: %s, elapsed: %s",
					clusterID, result.Payload.DeletionTimestamp.String(), time.Since(deleteStartTime).Round(time.Second))
			}
		}

		select {
		case <-ctx.Done():
			resp.Diagnostics.AddError(
				"Operation cancelled",
				fmt.Sprintf("Deletion of cluster '%s' was cancelled: %v", clusterID, ctx.Err()),
			)
			return
		case <-timeoutTimer.C:
			resp.Diagnostics.AddError(
				"Timeout deleting cluster",
				fmt.Sprintf("Timeout waiting for cluster '%s' to be deleted after %s. You can configure a longer timeout using the 'timeouts' block in your Terraform configuration.",
					clusterID, time.Since(deleteStartTime).Round(time.Second)),
			)
			return
		case <-ticker.C:
		}
	}
}

func (r *clusterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse the supported project-qualified or cluster-only import ID.
	parts := strings.Split(req.ID, ":")

	switch len(parts) {
	case 1:
		// find the project ID for the cluster.
		clusterID := parts[0]
		projectID, err := common.MetakubeResourceClusterFindProjectID(ctx, clusterID, r.meta)
		if err != nil {
			resp.Diagnostics.AddError("Failed to find project", err.Error())
			return
		}
		if projectID == "" {
			resp.Diagnostics.AddError("Project not found", fmt.Sprintf("Could not find project for cluster '%s'", clusterID))
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), clusterID)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), projectID)...)
	case 2:
		// Seed identity attributes for the first post-import Read.
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), parts[0])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
	default:
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Please provide resource identifier in format 'project_id:cluster_id' or 'cluster_id'",
		)
	}
}

// UpgradeState upgrades schema versions 0 and 1 directly to version 2. Both
// legacy versions represented singleton nested blocks as lists and could retain
// the now-unsupported AWS and Azure cloud fields.
func (r *clusterResource) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	// Register every supported legacy schema version for direct migration to v2.
	return map[int64]resource.StateUpgrader{
		0: {
			StateUpgrader: upgradeClusterStateToV2,
		},
		1: {
			StateUpgrader: upgradeClusterStateToV2,
		},
	}
}

// upgradeClusterStateToV2 rewrites legacy raw state into the version 2 object
// shape. Unsupported AWS and Azure fields are intentionally discarded because
// the current schema cannot represent them.
func upgradeClusterStateToV2(_ context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
	// Skip migration when Terraform supplied no legacy state.
	if req.RawState == nil || len(req.RawState.JSON) == 0 {
		return
	}

	// Decode the legacy JSON without requiring its retired framework schema.
	var rawState map[string]any
	if err := json.Unmarshal(req.RawState.JSON, &rawState); err != nil {
		resp.Diagnostics.AddError(
			"Failed to unmarshal legacy cluster state",
			fmt.Sprintf("Unable to unmarshal existing metakube_cluster state for upgrade: %v", err),
		)
		return
	}

	// Rewrite legacy nested blocks and discard unsupported cloud providers.
	upgradeClusterLegacyNestedSpecState(rawState)

	// Encode the migrated object in the current dynamic-state format.
	upgradedJSON, err := json.Marshal(rawState)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to marshal upgraded cluster state",
			fmt.Sprintf("Unable to marshal upgraded metakube_cluster state: %v", err),
		)
		return
	}

	// Return the migrated value to the framework.
	resp.DynamicValue = &tfprotov6.DynamicValue{JSON: upgradedJSON}
}

// upgradeClusterLegacyNestedSpecState converts each legacy singleton list into
// its current nested-object representation.
func upgradeClusterLegacyNestedSpecState(rawState map[string]any) {
	spec, ok := rawState["spec"].(map[string]any)
	if !ok {
		specList, listOK := rawState["spec"].([]any)
		if !listOK {
			return
		}
		if len(specList) == 0 {
			rawState["spec"] = nil
			return
		}
		spec, ok = specList[0].(map[string]any)
		if !ok {
			return
		}
		rawState["spec"] = spec
	}

	for _, key := range []string{"cni_plugin", "update_window", "cloud", "syseleven_auth"} {
		upgradeSingleItemListToObject(spec, key)
	}
	cloud, ok := spec["cloud"].(map[string]any)
	if !ok {
		return
	}
	delete(cloud, "azure")
	delete(cloud, "aws")
	upgradeSingleItemListToObject(cloud, "openstack")

	openstack, ok := cloud["openstack"].(map[string]any)
	if !ok {
		return
	}
	upgradeSingleItemListToObject(openstack, "user_credentials")
	upgradeSingleItemListToObject(openstack, "application_credentials")
}

// upgradeSingleItemListToObject maps an empty legacy block to null and a
// populated singleton block to its first object.
func upgradeSingleItemListToObject(parent map[string]any, key string) {
	value, ok := parent[key].([]any)
	if !ok {
		return
	}
	if len(value) == 0 {
		parent[key] = nil
		return
	}
	if object, ok := value[0].(map[string]any); ok {
		parent[key] = object
	}
}

// clusterOmittedState records prior values for fields the cluster API omits
// from reads. Reconciliation restores only known values after flattening.
type clusterOmittedState struct {
	openstack *clusterOpenstackOmittedState
}

type clusterOpenstackOmittedState struct {
	openstackProjectID                    types.String
	openstackProjectName                  types.String
	openstackUsername                     types.String
	openstackPassword                     types.String
	openstackApplicationCredentialsID     types.String
	openstackApplicationCredentialsSecret types.String
	openstackServerGroupID                types.String
}

func clusterOmittedStateValues(ctx context.Context, model *ClusterModel) clusterOmittedState {
	var values clusterOmittedState
	spec, ok := getClusterSpecModel(ctx, model)
	if !ok || spec.Cloud.IsNull() || spec.Cloud.IsUnknown() {
		return values
	}

	var cloud ClusterCloudSpecModel
	if diags := spec.Cloud.As(ctx, &cloud, basetypes.ObjectAsOptions{}); diags.HasError() ||
		cloud.Openstack.IsNull() || cloud.Openstack.IsUnknown() {
		return values
	}

	var openstack OpenstackCloudSpecModel
	if diags := cloud.Openstack.As(ctx, &openstack, basetypes.ObjectAsOptions{}); diags.HasError() {
		return values
	}
	values.openstack = &clusterOpenstackOmittedState{
		openstackServerGroupID: openstack.ServerGroupID,
	}

	if !openstack.UserCredentials.IsNull() && !openstack.UserCredentials.IsUnknown() {
		var credentials OpenstackUserCredentialsModel
		if diags := openstack.UserCredentials.As(ctx, &credentials, basetypes.ObjectAsOptions{}); !diags.HasError() {
			values.openstack.openstackProjectID = credentials.ProjectID
			values.openstack.openstackProjectName = credentials.ProjectName
			values.openstack.openstackUsername = credentials.Username
			values.openstack.openstackPassword = credentials.Password
		}
	}
	if !openstack.ApplicationCredentials.IsNull() && !openstack.ApplicationCredentials.IsUnknown() {
		var credentials OpenstackApplicationCredentialsModel
		if diags := openstack.ApplicationCredentials.As(ctx, &credentials, basetypes.ObjectAsOptions{}); !diags.HasError() {
			values.openstack.openstackApplicationCredentialsID = credentials.ID
			values.openstack.openstackApplicationCredentialsSecret = credentials.Secret
		}
	}
	return values
}

// reconcileClusterOmittedState restores configured OpenStack values that the
// API does not return, preventing a read from creating a false Terraform diff.
func reconcileClusterOmittedState(ctx context.Context, model *ClusterModel, omitted clusterOmittedState) diag.Diagnostics {
	var diags diag.Diagnostics
	if omitted.openstack == nil {
		return diags
	}

	spec, ok := getClusterSpecModel(ctx, model)
	if !ok || spec.Cloud.IsNull() || spec.Cloud.IsUnknown() {
		return diags
	}
	var cloud ClusterCloudSpecModel
	diags.Append(spec.Cloud.As(ctx, &cloud, basetypes.ObjectAsOptions{})...)
	if diags.HasError() || cloud.Openstack.IsNull() || cloud.Openstack.IsUnknown() {
		return diags
	}
	var openstack OpenstackCloudSpecModel
	diags.Append(cloud.Openstack.As(ctx, &openstack, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return diags
	}

	values := omitted.openstack
	if openstack.ServerGroupID.IsNull() && knownString(values.openstackServerGroupID) {
		openstack.ServerGroupID = values.openstackServerGroupID
	}
	if knownString(values.openstackProjectID) || knownString(values.openstackProjectName) ||
		knownString(values.openstackUsername) || knownString(values.openstackPassword) {
		var childDiags diag.Diagnostics
		openstack.UserCredentials, childDiags = types.ObjectValueFrom(ctx, openstackUserCredentialsAttrTypes(), OpenstackUserCredentialsModel{
			ProjectID:   values.openstackProjectID,
			ProjectName: values.openstackProjectName,
			Username:    values.openstackUsername,
			Password:    values.openstackPassword,
		})
		diags.Append(childDiags...)
	}
	if knownString(values.openstackApplicationCredentialsID) ||
		knownString(values.openstackApplicationCredentialsSecret) {
		var childDiags diag.Diagnostics
		openstack.ApplicationCredentials, childDiags = types.ObjectValueFrom(
			ctx,
			openstackApplicationCredentialsAttrTypes(),
			OpenstackApplicationCredentialsModel{
				ID:     values.openstackApplicationCredentialsID,
				Secret: values.openstackApplicationCredentialsSecret,
			},
		)
		diags.Append(childDiags...)
	}
	if diags.HasError() {
		return diags
	}

	var childDiags diag.Diagnostics
	cloud.Openstack, childDiags = types.ObjectValueFrom(ctx, openstackCloudSpecAttrTypes(), openstack)
	diags.Append(childDiags...)
	spec.Cloud, childDiags = types.ObjectValueFrom(ctx, clusterCloudSpecAttrTypes(), cloud)
	diags.Append(childDiags...)
	model.Spec, childDiags = types.ObjectValueFrom(ctx, clusterSpecAttrTypes(), spec)
	diags.Append(childDiags...)
	return diags
}

func knownString(value types.String) bool {
	return !value.IsNull() && !value.IsUnknown() && value.ValueString() != ""
}

// readClusterIntoModel refreshes Terraform state from the API. Only a not-found
// response from the cluster lookup proves remote absence; other lookup errors
// return diagnostics and retain the resource ID. OpenStack credentials and the
// server group ID may be omitted by the API, so their known prior values are
// captured before flattening and restored afterward.
func (r *clusterResource) readClusterIntoModel(ctx context.Context, model *ClusterModel) diag.Diagnostics {
	var diags diag.Diagnostics
	omittedState := clusterOmittedStateValues(ctx, model)

	projectID := model.ProjectID.ValueString()
	if projectID == "" {
		var err error
		projectID, err = common.MetakubeResourceClusterFindProjectID(ctx, model.ID.ValueString(), r.meta)
		if err != nil {
			diags.AddError("Failed to find project", err.Error())
			return diags
		}
		if projectID == "" {
			model.ID = types.StringNull()
			return diags
		}
		r.meta.Log.Debugf("found cluster in project '%s'", projectID)
	}

	p := project.NewGetClusterV2Params().WithContext(ctx).WithProjectID(projectID).WithClusterID(model.ID.ValueString())
	result, err := r.meta.Client.Project.GetClusterV2(p, r.meta.Auth)
	if isNotFoundError(err) {
		r.meta.Log.Infof("removing cluster '%s', could not find the resource", model.ID.ValueString())
		model.ID = types.StringNull()
		return diags
	}
	if err != nil {
		r.meta.Log.Debugf("get cluster: %v", err)
		diags.AddError(
			"Unable to get cluster",
			fmt.Sprintf("Unable to get cluster '%s/%s': %s", projectID, model.ID.ValueString(), common.StringifyResponseError(err)),
		)
		return diags
	}

	model.ProjectID = types.StringValue(projectID)
	model.DCName = types.StringValue(result.Payload.Spec.Cloud.DatacenterName)
	model.Name = types.StringValue(result.Payload.Name)

	if len(result.Payload.Labels) > 0 {
		labels := make(map[string]attr.Value)
		for k, v := range result.Payload.Labels {
			if !common.MetakubeResourceSystemLabelOrTag(k) {
				labels[k] = types.StringValue(v)
			}
		}
		labelsValue, d := types.MapValue(types.StringType, labels)
		diags.Append(d...)
		if diags.HasError() {
			return diags
		}
		model.Labels = labelsValue
	} else {
		model.Labels = types.MapValueMust(types.StringType, map[string]attr.Value{})
	}

	diags.Append(metakubeResourceClusterFlattenSpec(ctx, model, result.Payload.Spec)...)
	diags.Append(reconcileClusterOmittedState(ctx, model, omittedState)...)
	if diags.HasError() {
		return diags
	}

	model.CreationTimestamp = types.StringValue(result.Payload.CreationTimestamp.String())
	model.DeletionTimestamp = types.StringValue(result.Payload.DeletionTimestamp.String())

	keys, err := r.metakubeClusterGetAssignedSSHKeys(ctx, projectID, model.ID.ValueString())
	if err != nil {
		diags.AddError("Failed to get SSH keys", err.Error())
		return diags
	}
	if len(keys) > 0 {
		sshKeyValues := make([]attr.Value, len(keys))
		for i, k := range keys {
			sshKeyValues[i] = types.StringValue(k)
		}
		model.SSHKeys = types.SetValueMust(types.StringType, sshKeyValues)
	} else {
		model.SSHKeys = types.SetValueMust(types.StringType, []attr.Value{})
	}

	if conf, err := r.metakubeClusterUpdateKubeconfig(ctx, projectID, model.ID.ValueString()); err != nil {
		diags.AddWarning("Could not get kubeconfig", fmt.Sprintf("could not update kubeconfig: %v", err))
		model.KubeConfig = types.StringValue("")
	} else {
		model.KubeConfig = types.StringValue(conf)
	}

	if hasSyselevenAuth(ctx, model) {
		if conf, err := r.metakubeClusterUpdateOIDCKubeconfig(ctx, projectID, model.ID.ValueString()); err != nil {
			diags.AddWarning("Could not get OIDC kubeconfig", fmt.Sprintf("could not update OIDC kubeconfig: %s", common.StringifyResponseError(err)))
			model.OIDCKubeConfig = types.StringNull()
		} else {
			model.OIDCKubeConfig = types.StringValue(conf)
		}

		if conf, err := r.metakubeClusterUpdateKubeloginKubeconfig(ctx, projectID, model.ID.ValueString()); err != nil {
			diags.AddWarning("Could not get kubelogin kubeconfig", fmt.Sprintf("could not update kubelogin kubeconfig: %v", err))
			model.KubeLoginKubeConfig = types.StringNull()
		} else {
			model.KubeLoginKubeConfig = types.StringValue(conf)
		}
	} else {
		model.OIDCKubeConfig = types.StringNull()
		model.KubeLoginKubeConfig = types.StringNull()
	}

	return diags
}

func (r *clusterResource) metakubeClusterUpdateKubeconfig(ctx context.Context, projectID, clusterID string) (string, error) {
	kubeConfigParams := project.NewGetClusterKubeconfigV2Params()
	kubeConfigParams.SetContext(ctx)
	kubeConfigParams.SetProjectID(projectID)
	kubeConfigParams.SetClusterID(clusterID)
	ret, err := r.meta.Client.Project.GetClusterKubeconfigV2(kubeConfigParams, r.meta.Auth)
	if err != nil {
		return "", fmt.Errorf("failed to get kube_config: %s", common.StringifyResponseError(err))
	}
	return string(ret.Payload), nil
}

func (r *clusterResource) metakubeClusterUpdateOIDCKubeconfig(ctx context.Context, projectID, clusterID string) (string, error) {
	kubeConfigParams := project.NewGetOidcClusterKubeconfigV2Params()
	kubeConfigParams.SetContext(ctx)
	kubeConfigParams.SetProjectID(projectID)
	kubeConfigParams.SetClusterID(clusterID)
	ret, err := r.meta.Client.Project.GetOidcClusterKubeconfigV2(kubeConfigParams, r.meta.Auth)
	if err != nil {
		return "", fmt.Errorf("failed to get oidc_kube_config: %s", common.StringifyResponseError(err))
	}
	return string(ret.Payload), nil
}

func (r *clusterResource) metakubeClusterUpdateKubeloginKubeconfig(ctx context.Context, projectID, clusterID string) (string, error) {
	kubeConfigParams := project.NewGetKubeLoginClusterKubeconfigV2Params()
	kubeConfigParams.SetContext(ctx)
	kubeConfigParams.SetProjectID(projectID)
	kubeConfigParams.SetClusterID(clusterID)
	ret, err := r.meta.Client.Project.GetKubeLoginClusterKubeconfigV2(kubeConfigParams, r.meta.Auth)
	if err != nil {
		return "", fmt.Errorf("failed to get kube_login_kube_config: %s", common.StringifyResponseError(err))
	}
	return string(ret.Payload), nil
}

func (r *clusterResource) metakubeClusterGetAssignedSSHKeys(ctx context.Context, projectID, clusterID string) ([]string, error) {
	p := project.NewListSSHKeysAssignedToClusterV2Params().WithProjectID(projectID).WithClusterID(clusterID).WithContext(ctx)
	ret, err := r.meta.Client.Project.ListSSHKeysAssignedToClusterV2(p, r.meta.Auth)
	if err != nil {
		return nil, fmt.Errorf("list project keys error %v", common.StringifyResponseError(err))
	}

	var ids []string
	for _, v := range ret.Payload {
		ids = append(ids, v.ID)
	}
	return ids, nil
}

func (r *clusterResource) assignSSHKeysToCluster(projectID, clusterID string, sshkeyIDs []string) error {
	for _, id := range sshkeyIDs {
		p := project.NewAssignSSHKeyToClusterV2Params().WithProjectID(projectID).WithClusterID(clusterID).WithKeyID(id)
		_, err := r.meta.Client.Project.AssignSSHKeyToClusterV2(p, r.meta.Auth)
		if err != nil {
			return fmt.Errorf("can't assign sshkeys to cluster '%s': %v", clusterID, err)
		}
	}
	return nil
}

func (r *clusterResource) validateDatacenter(ctx context.Context, model *ClusterModel) diag.Diagnostics {
	var diags diag.Diagnostics

	name := model.DCName.ValueString()
	p := datacenter.NewListDatacentersV2Params().WithContext(ctx)
	result, err := r.meta.Client.Datacenter.ListDatacentersV2(p, r.meta.Auth)
	if err != nil {
		diags.AddError("Can't list datacenters", common.StringifyResponseError(err))
		return diags
	}

	available := make([]string, 0)
	openstackCluster := hasOpenstackConfig(ctx, model)

	for _, dc := range result.Payload {
		openstackDatacenter := dc.Spec.Openstack != nil
		if openstackCluster && openstackDatacenter {
			available = append(available, dc.Metadata.Name)
		}
		if dc.Metadata.Name == name {
			return diags
		}
	}

	summary := fmt.Sprintf("Could not find datacenter with name '%s'", name)
	var details string
	if name == "" {
		summary = "Datacenter name not set"
	}
	if len(available) > 0 {
		details = fmt.Sprintf("Please set one of available datacenters for the provider - %v", available)
	}

	diags.AddAttributeError(path.Root("dc_name"), summary, details)
	return diags
}

// sendPatchRequest retries conflict responses because they indicate another
// cluster operation is still active. Any other API error stops the update.
func (r *clusterResource) sendPatchRequest(ctx context.Context, projectID, clusterID string, patch map[string]any) error {
	p := project.NewPatchClusterV2Params()
	p.SetContext(ctx)
	p.SetProjectID(projectID)
	p.SetClusterID(clusterID)
	p.SetPatch(patch)

	timeout := 20 * time.Minute
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		_, err := r.meta.Client.Project.PatchClusterV2(p, r.meta.Auth)
		if err != nil {
			if e, ok := err.(*project.PatchClusterV2Default); ok && e.Code() == http.StatusConflict {
				time.Sleep(5 * time.Second)
				continue
			}
			return fmt.Errorf("patch cluster '%s': %v", clusterID, common.StringifyResponseError(err))
		}

		return nil
	}

	return fmt.Errorf("timeout patching cluster '%s'", clusterID)
}

func (r *clusterResource) updateClusterSSHKeys(ctx context.Context, plan, state *ClusterModel) error {
	projectID := plan.ProjectID.ValueString()
	clusterID := state.ID.ValueString()

	planKeys := expandSSHKeysFromModel(plan.SSHKeys)
	prevKeys, err := r.metakubeClusterGetAssignedSSHKeys(ctx, projectID, clusterID)
	if err != nil {
		return err
	}

	planKeySet := make(map[string]bool)
	for _, k := range planKeys {
		planKeySet[k] = true
	}

	for _, id := range prevKeys {
		if !planKeySet[id] {
			p := project.NewDetachSSHKeyFromClusterV2Params()
			p.SetProjectID(projectID)
			p.SetClusterID(clusterID)
			p.SetKeyID(id)
			_, err := r.meta.Client.Project.DetachSSHKeyFromClusterV2(p, r.meta.Auth)
			if err != nil {
				if e, ok := err.(*project.DetachSSHKeyFromClusterV2Default); ok && e.Code() == http.StatusNotFound {
					continue
				}
				return fmt.Errorf("failed to unassign sshkey: %s", common.StringifyResponseError(err))
			}
		}
	}

	prevKeySet := make(map[string]bool)
	for _, k := range prevKeys {
		prevKeySet[k] = true
	}

	var toAssign []string
	for _, id := range planKeys {
		if !prevKeySet[id] {
			toAssign = append(toAssign, id)
		}
	}

	return r.assignSSHKeysToCluster(projectID, clusterID, toAssign)
}

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	e, ok := err.(*project.GetClusterV2Default)
	if !ok {
		return false
	}
	return e.Code() == http.StatusNotFound
}

func getSSHAgentEnabled(ctx context.Context, model *ClusterModel) bool {
	spec, ok := getClusterSpecModel(ctx, model)
	if !ok {
		return true // default
	}
	if spec.EnableSSHAgent.IsNull() || spec.EnableSSHAgent.IsUnknown() {
		return true
	}
	return spec.EnableSSHAgent.ValueBool()
}

func getVersionFromModel(ctx context.Context, model *ClusterModel) string {
	spec, ok := getClusterSpecModel(ctx, model)
	if !ok {
		return ""
	}
	return spec.Version.ValueString()
}

func hasSyselevenAuth(ctx context.Context, model *ClusterModel) bool {
	spec, ok := getClusterSpecModel(ctx, model)
	if !ok {
		return false
	}
	if spec.SyselevenAuth.IsNull() || spec.SyselevenAuth.IsUnknown() {
		return false
	}
	var sysAuth SyselevenAuthModel
	if diags := spec.SyselevenAuth.As(ctx, &sysAuth, basetypes.ObjectAsOptions{}); diags.HasError() {
		return false
	}
	return !sysAuth.Realm.IsNull() && sysAuth.Realm.ValueString() != ""
}

func expandLabelsFromModel(labels types.Map) map[string]string {
	result := make(map[string]string)
	if labels.IsNull() || labels.IsUnknown() {
		return result
	}
	elements := labels.Elements()
	for k, v := range elements {
		if sv, ok := v.(types.String); ok {
			result[k] = sv.ValueString()
		}
	}
	return result
}

func expandSSHKeysFromModel(sshkeys types.Set) []string {
	var result []string
	if sshkeys.IsNull() || sshkeys.IsUnknown() {
		return result
	}
	for _, v := range sshkeys.Elements() {
		if sv, ok := v.(types.String); ok {
			result = append(result, sv.ValueString())
		}
	}
	return result
}
