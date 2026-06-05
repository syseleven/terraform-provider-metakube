package resource_node_deployment

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/syseleven/go-metakube/client/project"
	"github.com/syseleven/go-metakube/client/versions"
	"github.com/syseleven/go-metakube/models"
	"github.com/syseleven/terraform-provider-metakube/metakube/common"
)

var (
	_ resource.Resource                 = &nodeDeploymentResource{}
	_ resource.ResourceWithConfigure    = &nodeDeploymentResource{}
	_ resource.ResourceWithImportState  = &nodeDeploymentResource{}
	_ resource.ResourceWithUpgradeState = &nodeDeploymentResource{}
)

// NewNodeDeployment returns a new node deployment resource for the framework provider
func NewNodeDeployment() resource.Resource {
	return &nodeDeploymentResource{}
}

type nodeDeploymentResource struct {
	meta *common.MetaKubeProviderMeta
}

func (r *nodeDeploymentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_node_deployment"
}

func (r *nodeDeploymentResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = NodeDeploymentSchema(ctx)
}

func (r *nodeDeploymentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *nodeDeploymentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan NodeDeploymentModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	clusterID := plan.ClusterID.ValueString()
	projectID := plan.ProjectID.ValueString()

	// If project_id not provided, find it from cluster
	if projectID == "" {
		var err error
		projectID, err = common.MetakubeResourceClusterFindProjectID(ctx, clusterID, r.meta)
		if err != nil {
			resp.Diagnostics.AddError("Failed to find project", fmt.Sprintf("Could not find project for cluster %s: %v", clusterID, err))
			return
		}
		if projectID == "" {
			resp.Diagnostics.AddError("Project not found", fmt.Sprintf("Could not find owner project for cluster with id '%s'", clusterID))
			return
		}
	}

	nodeDeploymentSpec, d := expandNodeDeploymentSpec(ctx, plan.Spec, true)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	nodeDeployment := &models.NodeDeployment{
		Name: plan.Name.ValueString(),
		Spec: nodeDeploymentSpec,
	}

	resp.Diagnostics.Append(r.validateProviderMatchesCluster(ctx, projectID, clusterID, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.validateAutoscalerFields(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.validateVersionCompatibleWithCluster(ctx, projectID, clusterID, nodeDeployment); err != nil {
		resp.Diagnostics.AddError("Version validation failed", err.Error())
		return
	}

	createTimeout, d := plan.Timeouts.Create(ctx, 20*time.Minute)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := common.MetakubeResourceClusterWaitForReady(ctx, r.meta, createTimeout, projectID, clusterID, ""); err != nil {
		resp.Diagnostics.AddError("Cluster not ready", fmt.Sprintf("Cluster is not ready: %v", err))
		return
	}

	deadline := time.Now().Add(createTimeout)
	for {
		if time.Now().After(deadline) {
			resp.Diagnostics.AddError("Timeout", "Timeout waiting for node deployments API to be ready")
			return
		}

		p := project.NewListMachineDeploymentsParams().
			WithContext(ctx).
			WithProjectID(projectID).
			WithClusterID(clusterID)
		_, err := r.meta.Client.Project.ListMachineDeployments(p, r.meta.Auth)
		if err == nil {
			break
		}

		if e, ok := err.(*project.ListMachineDeploymentsDefault); ok && e.Code() != http.StatusOK {
			select {
			case <-ctx.Done():
				resp.Diagnostics.AddError("Context cancelled", "Context cancelled while waiting for API")
				return
			case <-time.After(5 * time.Second):
				continue
			}
		}

		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Failed to list node deployments: %v", err))
		return
	}

	// Create the node deployment with retry
	var nodeDeploymentID string
	deadline = time.Now().Add(createTimeout)
	for {
		if time.Now().After(deadline) {
			resp.Diagnostics.AddError("Timeout", "Timeout waiting to create node deployment")
			return
		}

		p := project.NewCreateMachineDeploymentParams().
			WithContext(ctx).
			WithProjectID(projectID).
			WithClusterID(clusterID).
			WithBody(nodeDeployment)

		result, err := r.meta.Client.Project.CreateMachineDeployment(p, r.meta.Auth)
		if err == nil {
			nodeDeploymentID = result.Payload.ID
			break
		}

		errStr := common.StringifyResponseError(err)
		if strings.Contains(errStr, "failed calling webhook") || strings.Contains(errStr, "Cluster components are not ready yet") {
			select {
			case <-ctx.Done():
				resp.Diagnostics.AddError("Context cancelled", "Context cancelled while creating node deployment")
				return
			case <-time.After(5 * time.Second):
				continue
			}
		}

		resp.Diagnostics.AddError("Failed to create node deployment", errStr)
		return
	}

	if err := r.waitForReady(ctx, createTimeout, projectID, clusterID, nodeDeploymentID); err != nil {
		resp.Diagnostics.AddError("Node deployment not ready", err.Error())
		return
	}

	// Read back the resource
	plan.ID = types.StringValue(nodeDeploymentID)
	plan.ProjectID = types.StringValue(projectID)

	resp.Diagnostics.Append(r.readIntoModel(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r *nodeDeploymentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state NodeDeploymentModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.readIntoModel(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.ID.IsNull() || state.ID.ValueString() == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *nodeDeploymentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var config, plan, state NodeDeploymentModel

	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	diags = req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID := state.ProjectID.ValueString()
	clusterID := state.ClusterID.ValueString()
	nodeDeploymentID := state.ID.ValueString()

	nodeDeploymentSpec, d := expandNodeDeploymentSpec(ctx, plan.Spec, false)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	nodeDeployment := &models.NodeDeployment{
		Spec: nodeDeploymentSpec,
	}

	resp.Diagnostics.Append(r.validateAutoscalerFields(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.validateVersionCompatibleWithCluster(ctx, projectID, clusterID, nodeDeployment); err != nil {
		resp.Diagnostics.AddError("Version validation failed", err.Error())
		return
	}

	patch, err := r.buildPatchWithDeletions(&config, &plan, &state, nodeDeployment)
	if err != nil {
		resp.Diagnostics.AddError("Failed to build patch", err.Error())
		return
	}

	updateTimeout, d := plan.Timeouts.Update(ctx, 20*time.Minute)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Send patch with retry
	deadline := time.Now().Add(updateTimeout)
	for {
		if time.Now().After(deadline) {
			resp.Diagnostics.AddError("Timeout", "Timeout waiting to update node deployment")
			return
		}

		p := project.NewPatchMachineDeploymentParams().
			WithContext(ctx).
			WithProjectID(projectID).
			WithClusterID(clusterID).
			WithMachineDeploymentID(nodeDeploymentID).
			WithPatch(patch)

		_, err := r.meta.Client.Project.PatchMachineDeployment(p, r.meta.Auth)
		if err == nil {
			break
		}

		errStr := common.StringifyResponseError(err)
		if strings.Contains(errStr, "the object has been modified") {
			select {
			case <-ctx.Done():
				resp.Diagnostics.AddError("Context cancelled", "Context cancelled while updating node deployment")
				return
			case <-time.After(2 * time.Second):
				continue
			}
		}

		resp.Diagnostics.AddError("Failed to update node deployment", errStr)
		return
	}

	if err := r.waitForReady(ctx, updateTimeout, projectID, clusterID, nodeDeploymentID); err != nil {
		resp.Diagnostics.AddError("Node deployment not ready", err.Error())
		return
	}

	plan.ID = state.ID
	plan.ProjectID = state.ProjectID

	// Read back the resource
	resp.Diagnostics.Append(r.readIntoModel(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r *nodeDeploymentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state NodeDeploymentModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID := state.ProjectID.ValueString()
	clusterID := state.ClusterID.ValueString()
	nodeDeploymentID := state.ID.ValueString()

	p := project.NewDeleteMachineDeploymentParams().
		WithProjectID(projectID).
		WithClusterID(clusterID).
		WithMachineDeploymentID(nodeDeploymentID)

	_, err := r.meta.Client.Project.DeleteMachineDeployment(p, r.meta.Auth)
	if err != nil {
		if e, ok := err.(*project.DeleteMachineDeploymentDefault); ok && e.Code() == http.StatusNotFound {
			// Already deleted
			return
		}
		resp.Diagnostics.AddError("Failed to delete node deployment", common.StringifyResponseError(err))
		return
	}

	deleteTimeout, d := state.Timeouts.Delete(ctx, 20*time.Minute)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	deadline := time.Now().Add(deleteTimeout)
	for {
		if time.Now().After(deadline) {
			resp.Diagnostics.AddError("Timeout", "Timeout waiting for node deployment to be deleted")
			return
		}

		getParams := project.NewGetMachineDeploymentParams().
			WithContext(ctx).
			WithProjectID(projectID).
			WithClusterID(clusterID).
			WithMachineDeploymentID(nodeDeploymentID)

		_, err := r.meta.Client.Project.GetMachineDeployment(getParams, r.meta.Auth)
		if err != nil {
			if e, ok := err.(*project.GetMachineDeploymentDefault); ok && e.Code() == http.StatusNotFound {
				// Deleted
				return
			}
			resp.Diagnostics.AddError("Failed to check node deployment deletion", common.StringifyResponseError(err))
			return
		}

		select {
		case <-ctx.Done():
			resp.Diagnostics.AddError("Context cancelled", "Context cancelled while waiting for deletion")
			return
		case <-time.After(5 * time.Second):
		}
	}
}

func (r *nodeDeploymentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")

	if len(parts) != 3 {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"please provide resource identifier in format 'project_id:cluster_id:node_deployment_id'",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[2])...)
}

func (r *nodeDeploymentResource) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {
			StateUpgrader: upgradeNodeDeploymentStateToV1,
		},
	}
}

func upgradeNodeDeploymentStateToV1(_ context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
	if req.RawState == nil || len(req.RawState.JSON) == 0 {
		return
	}

	var rawState map[string]any
	if err := json.Unmarshal(req.RawState.JSON, &rawState); err != nil {
		resp.Diagnostics.AddError(
			"Failed to unmarshal legacy node deployment state",
			fmt.Sprintf("Unable to unmarshal existing metakube_node_deployment state for upgrade: %v", err),
		)
		return
	}

	upgradeNodeDeploymentLegacyAzureState(rawState)

	upgradedJSON, err := json.Marshal(rawState)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to marshal upgraded node deployment state",
			fmt.Sprintf("Unable to marshal upgraded metakube_node_deployment state: %v", err),
		)
		return
	}

	resp.DynamicValue = &tfprotov6.DynamicValue{JSON: upgradedJSON}
}

// readIntoModel reads the node deployment from the API and updates the model
func (r *nodeDeploymentResource) readIntoModel(ctx context.Context, model *NodeDeploymentModel) (result diag.Diagnostics) {

	projectID := model.ProjectID.ValueString()
	clusterID := model.ClusterID.ValueString()
	nodeDeploymentID := model.ID.ValueString()

	p := project.NewGetMachineDeploymentParams().
		WithContext(ctx).
		WithProjectID(projectID).
		WithClusterID(clusterID).
		WithMachineDeploymentID(nodeDeploymentID)

	resp, err := r.meta.Client.Project.GetMachineDeployment(p, r.meta.Auth)
	if err != nil {
		if e, ok := err.(*project.GetMachineDeploymentDefault); ok && e.Code() == http.StatusNotFound {
			r.meta.Log.Infof("removing node deployment '%s' from terraform state file, could not find the resource", nodeDeploymentID)
			model.ID = types.StringNull()
			return result
		}
		if _, ok := err.(*project.GetMachineDeploymentForbidden); ok {
			r.meta.Log.Infof("removing node deployment '%s' from terraform state file, access forbidden", nodeDeploymentID)
			model.ID = types.StringNull()
			return result
		}
		result.AddError("Failed to read node deployment", fmt.Sprintf("unable to get node deployment '%s/%s/%s': %s", projectID, clusterID, nodeDeploymentID, common.StringifyResponseError(err)))
		return result
	}

	nd := resp.Payload

	model.Name = types.StringValue(nd.Name)

	specList, d := flattenNodeDeploymentSpec(ctx, nd.Spec)
	result.Append(d...)
	if result.HasError() {
		return result
	}
	model.Spec = specList

	model.CreationTimestamp = types.StringValue(nd.CreationTimestamp.String())
	model.DeletionTimestamp = types.StringValue(nd.DeletionTimestamp.String())

	return result
}

// waitForReady waits for the node deployment to be ready
func (r *nodeDeploymentResource) waitForReady(ctx context.Context, timeout time.Duration, projectID, clusterID, nodeDeploymentID string) error {
	deadline := time.Now().Add(timeout)

	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for node deployment '%s' to be ready", nodeDeploymentID)
		}

		p := project.NewGetMachineDeploymentParams().
			WithContext(ctx).
			WithProjectID(projectID).
			WithClusterID(clusterID).
			WithMachineDeploymentID(nodeDeploymentID)

		resp, err := r.meta.Client.Project.GetMachineDeployment(p, r.meta.Auth)
		if err != nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(5 * time.Second):
				continue
			}
		}

		nd := resp.Payload
		if nd.Spec.Replicas == nil || nd.Status == nil ||
			nd.Status.ReadyReplicas < *nd.Spec.Replicas ||
			nd.Status.UnavailableReplicas != 0 {
			r.meta.Log.Debugf("waiting for node deployment '%s' to be ready, %+v", nodeDeploymentID, nd.Status)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(10 * time.Second):
				continue
			}
		}

		p2 := project.NewListMachineDeploymentNodesParams().
			WithContext(ctx).
			WithProjectID(projectID).
			WithClusterID(clusterID).
			WithMachineDeploymentID(nodeDeploymentID)
		nodesResp, err := r.meta.Client.Project.ListMachineDeploymentNodes(p2, r.meta.Auth)
		if err != nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(5 * time.Second):
				continue
			}
		}

		if len(nodesResp.Payload) != int(*nd.Spec.Replicas) {
			r.meta.Log.Debug("node count mismatch, want %v got %v", *nd.Spec.Replicas, len(nodesResp.Payload))
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(10 * time.Second):
				continue
			}
		}

		allReady := true
		for _, node := range nodesResp.Payload {
			if node.Status == nil || node.Status.NodeInfo == nil || node.Status.NodeInfo.KernelVersion == "" {
				allReady = false
				break
			}
		}

		if !allReady {
			r.meta.Log.Debug("found not ready node")
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(10 * time.Second):
				continue
			}
		}

		return nil
	}
}

// validateProviderMatchesCluster validates that the node deployment cloud provider matches the cluster
func (r *nodeDeploymentResource) validateProviderMatchesCluster(ctx context.Context, projectID, clusterID string, model *NodeDeploymentModel) (result diag.Diagnostics) {
	cluster, _, err := common.MetakubeGetCluster(ctx, projectID, clusterID, r.meta)
	if err != nil {
		result.AddError("Failed to get cluster", err.Error())
		return result
	}

	var clusterProvider string
	switch {
	case cluster.Spec.Cloud.Openstack != nil:
		clusterProvider = "openstack"
	default:
		return result
	}

	nodeProvider, d := getCloudProviderFromModel(ctx, model)
	result.Append(d...)
	if result.HasError() {
		return result
	}

	if nodeProvider != "" && nodeProvider != clusterProvider {
		result.AddError("Provider mismatch", fmt.Sprintf("provider for node deployment must (%s) match cluster provider (%s)", nodeProvider, clusterProvider))
		return result
	}

	return result
}

// validateAutoscalerFields validates min_replicas <= max_replicas
func (r *nodeDeploymentResource) validateAutoscalerFields(ctx context.Context, model *NodeDeploymentModel) (result diag.Diagnostics) {
	if model.Spec.IsNull() || model.Spec.IsUnknown() || len(model.Spec.Elements()) == 0 {
		return result
	}

	var specModels []NodeDeploymentSpecModel
	model.Spec.ElementsAs(ctx, &specModels, false)
	if len(specModels) == 0 {
		return result
	}

	spec := specModels[0]

	if spec.MinReplicas.IsNull() && spec.MaxReplicas.IsNull() {
		return result
	}

	if !spec.MinReplicas.IsNull() && !spec.MaxReplicas.IsNull() {
		min := spec.MinReplicas.ValueInt64()
		max := spec.MaxReplicas.ValueInt64()
		if min > max {
			result.AddError("Invalid autoscaler configuration", "min_replicas must be smaller than max_replicas")
			return result
		}
	}

	return result
}

// validateVersionCompatibleWithCluster validates kubelet version against cluster version
func (r *nodeDeploymentResource) validateVersionCompatibleWithCluster(ctx context.Context, projectID, clusterID string, nd *models.NodeDeployment) error {
	cluster, _, err := common.MetakubeGetCluster(ctx, projectID, clusterID, r.meta)
	if err != nil {
		return err
	}
	clusterVersion := string(cluster.Spec.Version)

	var kubeletVersion string
	if nd.Spec != nil && nd.Spec.Template != nil && nd.Spec.Template.Versions != nil {
		kubeletVersion = nd.Spec.Template.Versions.Kubelet
	}

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

	p := versions.NewGetNodeUpgradesParams()
	p.SetControlPlaneVersion(&clusterVersion)
	versionResp, err := r.meta.Client.Versions.GetNodeUpgrades(p, r.meta.Auth)
	if err != nil {
		if e, ok := err.(*versions.GetNodeUpgradesDefault); ok && e.Payload != nil && e.Payload.Error != nil && e.Payload.Error.Message != nil {
			return fmt.Errorf("get node_deployment upgrades: %s", *e.Payload.Error.Message)
		}
		return err
	}

	var availableVersions []string
	for _, ver := range versionResp.Payload {
		if ver.Version == kubeletVersion && !ver.RestrictedByKubeletVersion {
			return nil
		}
		availableVersions = append(availableVersions, ver.Version)
	}

	return fmt.Errorf("unknown version for node deployment %s, available versions %v", kubeletVersion, availableVersions)
}

// buildPatchWithDeletions builds the SET patch from the expanded API spec and overlays
// merge-patch deletions inferred from framework plan/state map values.
func (r *nodeDeploymentResource) buildPatchWithDeletions(config, plan, state *NodeDeploymentModel, nd *models.NodeDeployment) (map[string]interface{}, error) {
	specPatch, err := nodeDeploymentSpecPatchBody(nd.Spec)
	if err != nil {
		return nil, fmt.Errorf("marshal node deployment spec: %w", err)
	}
	if specPatch == nil {
		specPatch = make(map[string]interface{})
	}

	common.AddMergePatchDeletions(
		specPatch,
		config.Spec,
		plan.Spec,
		state.Spec,
		models.NodeDeploymentSpec{},
		NodeDeploymentSpecModel{},
		NodeSpecModel{},
		CloudSpecModel{},
		OpenStackCloudSpecModel{},
	)

	return map[string]interface{}{"spec": specPatch}, nil
}

func nodeDeploymentSpecPatchBody(spec *models.NodeDeploymentSpec) (map[string]interface{}, error) {
	if spec == nil {
		return map[string]interface{}{}, nil
	}

	payload, err := json.Marshal(spec)
	if err != nil {
		return nil, err
	}

	var out map[string]interface{}
	if err := json.Unmarshal(payload, &out); err != nil {
		return nil, err
	}

	addExplicitOperatingSystemBoolFields(out, spec)

	return out, nil
}

func addExplicitOperatingSystemBoolFields(specMap map[string]interface{}, spec *models.NodeDeploymentSpec) {
	if specMap == nil || spec == nil || spec.Template == nil || spec.Template.OperatingSystem == nil {
		return
	}

	templateMap := common.AsObject(specMap["template"])
	operatingSystemMap := common.AsObject(templateMap["operatingSystem"])

	if ubuntu := spec.Template.OperatingSystem.Ubuntu; ubuntu != nil {
		ubuntuMap := common.AsObject(operatingSystemMap["ubuntu"])
		ubuntuMap["distUpgradeOnBoot"] = ubuntu.DistUpgradeOnBoot
		operatingSystemMap["ubuntu"] = ubuntuMap
	}

	if flatcar := spec.Template.OperatingSystem.Flatcar; flatcar != nil {
		flatcarMap := common.AsObject(operatingSystemMap["flatcar"])
		flatcarMap["disableAutoUpdate"] = flatcar.DisableAutoUpdate
		operatingSystemMap["flatcar"] = flatcarMap
	}

	templateMap["operatingSystem"] = operatingSystemMap
	specMap["template"] = templateMap
}
