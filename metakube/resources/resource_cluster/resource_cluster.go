package resource_cluster

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/syseleven/go-metakube/client/datacenter"
	"github.com/syseleven/go-metakube/client/project"
	"github.com/syseleven/go-metakube/models"
	"github.com/syseleven/terraform-provider-metakube/metakube/common"
)

var (
	_ resource.Resource                = &clusterResource{}
	_ resource.ResourceWithConfigure   = &clusterResource{}
	_ resource.ResourceWithImportState = &clusterResource{}
)

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

	dcname := plan.DCName.ValueString()
	clusterSpec := metakubeResourceClusterExpandSpec(ctx, &plan, dcname, func(_ string) bool { return true })
	clusterLabels := expandLabelsFromModel(ctx, plan.Labels)

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

	sshkeys := expandSSHKeysFromModel(ctx, plan.SSHKeys)
	if len(sshkeys) > 0 {
		sshAgentEnabled := getSSHAgentEnabled(ctx, &plan)
		if !sshAgentEnabled {
			resp.Diagnostics.AddAttributeError(
				path.Root("spec").AtListIndex(0).AtName("enable_ssh_agent"),
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
		resp.Diagnostics.Append(r.readClusterIntoModel(ctx, &plan)...)
		resp.Diagnostics.AddError(
			"Cluster not ready",
			fmt.Sprintf("Cluster '%s' is not ready: %v", result.Payload.ID, err),
		)
		resp.State.Set(ctx, &plan)
		return
	}

	resp.Diagnostics.Append(r.readClusterIntoModel(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r *clusterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ClusterModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.readClusterIntoModel(ctx, &state)...)
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

func (r *clusterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ClusterModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID := plan.ProjectID.ValueString()

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
		r.meta.Log.Debugf("validating version change")
		resp.Diagnostics.Append(metakubeResourceClusterValidateVersionUpgrade(ctx, projectID, planVersion, cluster, r.meta)...)
	}

	resp.Diagnostics.Append(metakubeResourceClusterValidateClusterFields(ctx, &plan, r.meta, true)...)
	resp.Diagnostics.Append(r.validateDatacenter(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	nameChanged := !plan.Name.Equal(state.Name)
	labelsChanged := !plan.Labels.Equal(state.Labels)
	specChanged := !plan.Spec.Equal(state.Spec)

	if nameChanged || labelsChanged || specChanged {
		if err := r.sendPatchRequest(ctx, &plan, &state); err != nil {
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

	resp.Diagnostics.Append(r.readClusterIntoModel(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r *clusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ClusterModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID := state.ProjectID.ValueString()
	clusterID := state.ID.ValueString()

	deleteTimeout, diags := state.Timeouts.Delete(ctx, 40*time.Minute)
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
	parts := strings.Split(req.ID, ":")

	switch len(parts) {
	case 1:
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
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), parts[0])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
	default:
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Please provide resource identifier in format 'project_id:cluster_id' or 'cluster_id'",
		)
	}
}

func (r *clusterResource) readClusterIntoModel(ctx context.Context, model *ClusterModel) diag.Diagnostics {
	var diags diag.Diagnostics

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
		model.Labels = labelsValue
	} else {
		model.Labels = types.MapValueMust(types.StringType, map[string]attr.Value{})
	}

	diags.Append(metakubeResourceClusterFlattenSpec(ctx, model, result.Payload.Spec)...)

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
			model.OIDCKubeConfig = types.StringValue("")
		} else {
			model.OIDCKubeConfig = types.StringValue(conf)
		}

		if conf, err := r.metakubeClusterUpdateKubeloginKubeconfig(ctx, projectID, model.ID.ValueString()); err != nil {
			diags.AddWarning("Could not get kubelogin kubeconfig", fmt.Sprintf("could not update kubelogin kubeconfig: %v", err))
			model.KubeLoginKubeConfig = types.StringValue("")
		} else {
			model.KubeLoginKubeConfig = types.StringValue(conf)
		}
	} else {
		model.OIDCKubeConfig = types.StringValue("")
		model.KubeLoginKubeConfig = types.StringValue("")
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
	awsCluster := hasAWSConfig(ctx, model)
	azureCluster := hasAzureConfig(ctx, model)

	for _, dc := range result.Payload {
		openstackDatacenter := dc.Spec.Openstack != nil
		awsDatacenter := dc.Spec.Aws != nil
		azureDatacenter := dc.Spec.Azure != nil
		if (openstackCluster && openstackDatacenter) ||
			(awsCluster && awsDatacenter) ||
			(azureCluster && azureDatacenter) {
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

func (r *clusterResource) sendPatchRequest(ctx context.Context, plan, state *ClusterModel) error {
	projectID := plan.ProjectID.ValueString()
	clusterID := state.ID.ValueString()

	p := project.NewPatchClusterV2Params()
	p.SetContext(ctx)
	p.SetProjectID(projectID)
	p.SetClusterID(clusterID)

	name := plan.Name.ValueString()
	labels := getLabelsChange(ctx, plan, state)
	clusterSpec := metakubeResourceClusterExpandSpec(ctx, plan, plan.DCName.ValueString(), func(_ string) bool { return true })

	p.SetPatch(map[string]interface{}{
		"name":   name,
		"labels": labels,
		"spec":   clusterSpec,
	})

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

	planKeys := expandSSHKeysFromModel(ctx, plan.SSHKeys)
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
	if model.Spec.IsNull() || model.Spec.IsUnknown() {
		return true // default
	}
	var specs []ClusterSpecModel
	if diags := model.Spec.ElementsAs(ctx, &specs, false); diags.HasError() || len(specs) == 0 {
		return true
	}
	if specs[0].EnableSSHAgent.IsNull() || specs[0].EnableSSHAgent.IsUnknown() {
		return true
	}
	return specs[0].EnableSSHAgent.ValueBool()
}

func getVersionFromModel(ctx context.Context, model *ClusterModel) string {
	if model.Spec.IsNull() || model.Spec.IsUnknown() {
		return ""
	}
	var specs []ClusterSpecModel
	if diags := model.Spec.ElementsAs(ctx, &specs, false); diags.HasError() || len(specs) == 0 {
		return ""
	}
	return specs[0].Version.ValueString()
}

func hasSyselevenAuth(ctx context.Context, model *ClusterModel) bool {
	if model.Spec.IsNull() || model.Spec.IsUnknown() {
		return false
	}
	var specs []ClusterSpecModel
	if diags := model.Spec.ElementsAs(ctx, &specs, false); diags.HasError() || len(specs) == 0 {
		return false
	}
	if specs[0].SyselevenAuth.IsNull() || specs[0].SyselevenAuth.IsUnknown() {
		return false
	}
	var sysAuth []SyselevenAuthModel
	if diags := specs[0].SyselevenAuth.ElementsAs(ctx, &sysAuth, false); diags.HasError() || len(sysAuth) == 0 {
		return false
	}
	return !sysAuth[0].Realm.IsNull() && sysAuth[0].Realm.ValueString() != ""
}

func hasAWSConfig(ctx context.Context, model *ClusterModel) bool {
	if model.Spec.IsNull() || model.Spec.IsUnknown() {
		return false
	}
	var specs []ClusterSpecModel
	if diags := model.Spec.ElementsAs(ctx, &specs, false); diags.HasError() || len(specs) == 0 {
		return false
	}
	if specs[0].Cloud.IsNull() || specs[0].Cloud.IsUnknown() {
		return false
	}
	var clouds []ClusterCloudSpecModel
	if diags := specs[0].Cloud.ElementsAs(ctx, &clouds, false); diags.HasError() || len(clouds) == 0 {
		return false
	}
	if clouds[0].AWS.IsNull() || clouds[0].AWS.IsUnknown() {
		return false
	}
	var aws []AWSCloudSpecModel
	if diags := clouds[0].AWS.ElementsAs(ctx, &aws, false); diags.HasError() || len(aws) == 0 {
		return false
	}
	return true
}

func hasAzureConfig(ctx context.Context, model *ClusterModel) bool {
	if model.Spec.IsNull() || model.Spec.IsUnknown() {
		return false
	}
	var specs []ClusterSpecModel
	if diags := model.Spec.ElementsAs(ctx, &specs, false); diags.HasError() || len(specs) == 0 {
		return false
	}
	if specs[0].Cloud.IsNull() || specs[0].Cloud.IsUnknown() {
		return false
	}
	var clouds []ClusterCloudSpecModel
	if diags := specs[0].Cloud.ElementsAs(ctx, &clouds, false); diags.HasError() || len(clouds) == 0 {
		return false
	}
	if clouds[0].Azure.IsNull() || clouds[0].Azure.IsUnknown() {
		return false
	}
	var azure []AzureCloudSpecModel
	if diags := clouds[0].Azure.ElementsAs(ctx, &azure, false); diags.HasError() || len(azure) == 0 {
		return false
	}
	return true
}

func expandLabelsFromModel(ctx context.Context, labels types.Map) map[string]string {
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

func expandSSHKeysFromModel(ctx context.Context, sshkeys types.Set) []string {
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

func getLabelsChange(ctx context.Context, plan, state *ClusterModel) map[string]interface{} {
	oldLabels := expandLabelsFromModel(ctx, state.Labels)
	newLabels := expandLabelsFromModel(ctx, plan.Labels)

	result := make(map[string]interface{})
	for k, v := range newLabels {
		result[k] = v
	}

	// Mark removed labels as nil for API
	for k := range oldLabels {
		if _, ok := newLabels[k]; !ok {
			result[k] = nil
		}
	}

	return result
}
