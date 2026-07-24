package resource_cluster

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/syseleven/terraform-provider-metakube/metakube/common"
)

func clusterSpecModel(ctx context.Context, model ClusterModel) (ClusterSpecModel, diag.Diagnostics) {
	var result ClusterSpecModel
	if model.Spec.IsNull() || model.Spec.IsUnknown() {
		return result, nil
	}
	diags := model.Spec.As(ctx, &result, basetypes.ObjectAsOptions{})
	return result, diags
}

func objectAs[T any](ctx context.Context, value types.Object) (T, diag.Diagnostics) {
	var result T
	diags := value.As(ctx, &result, basetypes.ObjectAsOptions{})
	return result, diags
}

// buildClusterPatch derives an RFC 7396 JSON merge patch from the planned and
// prior Terraform values. Nulls delete API fields, while omitted fields remain
// unchanged.
func buildClusterPatch(ctx context.Context, plan, state ClusterModel) (map[string]any, diag.Diagnostics) {
	patch := make(map[string]any)
	var diags diag.Diagnostics

	if !plan.Name.Equal(state.Name) {
		patch["name"] = mergePatchString(plan.Name)
	}
	if !plan.Labels.Equal(state.Labels) {
		labels, err := common.StringMapMergePatch(plan.Labels, state.Labels)
		if err != nil {
			diags.AddError("Unable to build labels patch", err.Error())
			return nil, diags
		}
		patch["labels"] = labels
	}
	if plan.Spec.Equal(state.Spec) {
		return patch, diags
	}

	planSpec, childDiags := clusterSpecModel(ctx, plan)
	diags.Append(childDiags...)
	stateSpec, childDiags := clusterSpecModel(ctx, state)
	diags.Append(childDiags...)
	if diags.HasError() {
		return nil, diags
	}
	spec := make(map[string]any)
	setChangedString(spec, "version", planSpec.Version, stateSpec.Version)
	setChangedBool(spec, "enableUserSSHKeyAgent", planSpec.EnableSSHAgent, stateSpec.EnableSSHAgent)
	setChangedBool(spec, "usePodSecurityPolicyAdmissionPlugin", planSpec.PodSecurityPolicy, stateSpec.PodSecurityPolicy)
	setChangedBool(spec, "usePodNodeSelectorAdmissionPlugin", planSpec.PodNodeSelector, stateSpec.PodNodeSelector)

	if !planSpec.AuditLogging.Equal(stateSpec.AuditLogging) {
		if planSpec.AuditLogging.IsNull() {
			spec["auditLogging"] = nil
		} else if !planSpec.AuditLogging.IsUnknown() {
			spec["auditLogging"] = map[string]any{"enabled": planSpec.AuditLogging.ValueBool()}
		}
	}
	if !planSpec.UpdateWindow.Equal(stateSpec.UpdateWindow) {
		switch {
		case planSpec.UpdateWindow.IsNull():
			spec["updateWindow"] = nil
		case !planSpec.UpdateWindow.IsUnknown():
			value, valueDiags := patchUpdateWindow(ctx, planSpec.UpdateWindow)
			diags.Append(valueDiags...)
			spec["updateWindow"] = value
		}
	}
	if !planSpec.CNIPlugin.Equal(stateSpec.CNIPlugin) {
		switch {
		case planSpec.CNIPlugin.IsNull():
			spec["cniPlugin"] = nil
		case !planSpec.CNIPlugin.IsUnknown():
			value, valueDiags := patchCNIPlugin(ctx, planSpec.CNIPlugin)
			diags.Append(valueDiags...)
			spec["cniPlugin"] = value
		}
	}
	if !planSpec.SyselevenAuth.Equal(stateSpec.SyselevenAuth) {
		switch {
		case planSpec.SyselevenAuth.IsNull():
			spec["sys11auth"] = nil
		case !planSpec.SyselevenAuth.IsUnknown():
			value, valueDiags := patchSyselevenAuth(ctx, planSpec.SyselevenAuth)
			diags.Append(valueDiags...)
			spec["sys11auth"] = value
		}
	}

	network := make(map[string]any)
	if !planSpec.ServicesCIDR.Equal(stateSpec.ServicesCIDR) {
		if !planSpec.ServicesCIDR.IsUnknown() {
			network["services"] = patchCIDR(planSpec.ServicesCIDR)
		}
	}
	if !planSpec.PodsCIDR.Equal(stateSpec.PodsCIDR) {
		if !planSpec.PodsCIDR.IsUnknown() {
			network["pods"] = patchCIDR(planSpec.PodsCIDR)
		}
	}
	setChangedString(network, "ipFamily", planSpec.IPFamily, stateSpec.IPFamily)
	if len(network) > 0 {
		spec["clusterNetwork"] = network
	}

	if !planSpec.Cloud.Equal(stateSpec.Cloud) {
		switch {
		case planSpec.Cloud.IsNull():
			spec["cloud"] = nil
		case !planSpec.Cloud.IsUnknown():
			value, valueDiags := patchCloud(ctx, planSpec.Cloud, stateSpec.Cloud)
			diags.Append(valueDiags...)
			spec["cloud"] = value
		}
	}
	if diags.HasError() {
		return nil, diags
	}
	if len(spec) > 0 {
		patch["spec"] = spec
	}
	return patch, diags
}

func patchUpdateWindow(ctx context.Context, value types.Object) (any, diag.Diagnostics) {
	if value.IsNull() {
		return nil, nil
	}
	if value.IsUnknown() {
		return nil, nil
	}
	model, diags := objectAs[UpdateWindowModel](ctx, value)
	if diags.HasError() {
		return nil, diags
	}
	return map[string]any{
		"start":  mergePatchString(model.Start),
		"length": mergePatchString(model.Length),
	}, diags
}

func patchCNIPlugin(ctx context.Context, value types.Object) (any, diag.Diagnostics) {
	if value.IsNull() {
		return nil, nil
	}
	if value.IsUnknown() {
		return nil, nil
	}
	model, diags := objectAs[CNIPluginModel](ctx, value)
	if diags.HasError() {
		return nil, diags
	}
	result := make(map[string]any)
	if model.Type.IsNull() {
		result["type"] = nil
	} else if !model.Type.IsUnknown() {
		result["type"] = model.Type.ValueString()
	}
	if model.Cilium.IsNull() {
		result["cilium"] = nil
		return result, diags
	}
	if model.Cilium.IsUnknown() {
		return result, diags
	}
	cilium, childDiags := objectAs[CiliumModel](ctx, model.Cilium)
	diags.Append(childDiags...)
	if diags.HasError() {
		return nil, diags
	}
	ciliumPatch := make(map[string]any)
	setConfiguredBool(ciliumPatch, "enableHubble", cilium.EnableHubble)
	setConfiguredBool(ciliumPatch, "enableL7Proxy", cilium.EnableL7Proxy)
	if cilium.Clustermesh.IsNull() {
		ciliumPatch["clustermesh"] = nil
	} else if !cilium.Clustermesh.IsUnknown() {
		mesh, meshDiags := objectAs[CiliumClustermeshModel](ctx, cilium.Clustermesh)
		diags.Append(meshDiags...)
		meshPatch := make(map[string]any)
		setConfiguredBool(meshPatch, "enable", mesh.Enable)
		setConfiguredInt32(meshPatch, "clusterID", mesh.ClusterID)
		setConfiguredString(meshPatch, "ipv4NativeRoutingCIDR", mesh.IPv4NativeRoutingCIDR)
		ciliumPatch["clustermesh"] = meshPatch
	}
	result["cilium"] = ciliumPatch
	return result, diags
}

func patchSyselevenAuth(ctx context.Context, value types.Object) (any, diag.Diagnostics) {
	if value.IsNull() {
		return nil, nil
	}
	if value.IsUnknown() {
		return nil, nil
	}
	model, diags := objectAs[SyselevenAuthModel](ctx, value)
	if diags.HasError() {
		return nil, diags
	}
	return map[string]any{
		"realm":             mergePatchString(model.Realm),
		"iamAuthentication": mergePatchBool(model.IAMAuthentication),
	}, diags
}

func patchCloud(ctx context.Context, plan, state types.Object) (any, diag.Diagnostics) {
	if plan.IsNull() {
		return nil, nil
	}
	if plan.IsUnknown() {
		return nil, nil
	}
	planModel, diags := objectAs[ClusterCloudSpecModel](ctx, plan)
	if diags.HasError() {
		return nil, diags
	}
	var stateModel ClusterCloudSpecModel
	if !state.IsNull() && !state.IsUnknown() {
		var childDiags diag.Diagnostics
		stateModel, childDiags = objectAs[ClusterCloudSpecModel](ctx, state)
		diags.Append(childDiags...)
	}
	openstack, childDiags := patchOpenstack(ctx, planModel.Openstack, stateModel.Openstack)
	diags.Append(childDiags...)
	return map[string]any{"openstack": openstack}, diags
}

// patchOpenstack builds the OpenStack fragment of the cluster patch.
//
// Terraform keeps user and application credentials in their own nested
// objects, but the API wants those fields alongside the other OpenStack
// fields, not nested. So after computing each credential patch separately,
// we copy their entries straight into result instead of nesting them.
func patchOpenstack(ctx context.Context, plan, state types.Object) (any, diag.Diagnostics) {
	if plan.IsNull() {
		return nil, nil
	}
	if plan.IsUnknown() {
		return nil, nil
	}
	planModel, diags := objectAs[OpenstackCloudSpecModel](ctx, plan)
	if diags.HasError() {
		return nil, diags
	}
	var stateModel OpenstackCloudSpecModel
	if !state.IsNull() && !state.IsUnknown() {
		var childDiags diag.Diagnostics
		stateModel, childDiags = objectAs[OpenstackCloudSpecModel](ctx, state)
		diags.Append(childDiags...)
	}
	result := make(map[string]any)
	setChangedString(result, "floatingIPPool", planModel.FloatingIPPool, stateModel.FloatingIPPool)
	setChangedString(result, "securityGroups", planModel.SecurityGroup, stateModel.SecurityGroup)
	setChangedString(result, "network", planModel.Network, stateModel.Network)
	setChangedString(result, "subnetID", planModel.SubnetID, stateModel.SubnetID)
	setChangedString(result, "subnetCIDR", planModel.SubnetCIDR, stateModel.SubnetCIDR)
	setChangedString(result, "serverGroupID", planModel.ServerGroupID, stateModel.ServerGroupID)

	userPatch, childDiags := patchUserCredentials(ctx, planModel.UserCredentials, stateModel.UserCredentials)
	diags.Append(childDiags...)
	for key, value := range userPatch {
		result[key] = value
	}
	appPatch, childDiags := patchApplicationCredentials(ctx, planModel.ApplicationCredentials, stateModel.ApplicationCredentials)
	diags.Append(childDiags...)
	for key, value := range appPatch {
		result[key] = value
	}
	return result, diags
}

// patchUserCredentials clears every API credential field when the configured
// credential object is removed. Otherwise, it emits only changed fields.
func patchUserCredentials(ctx context.Context, plan, state types.Object) (map[string]any, diag.Diagnostics) {
	keys := []string{"projectID", "project", "username", "password"}
	if plan.IsNull() {
		if state.IsNull() || state.IsUnknown() {
			return nil, nil
		}
		return nullFields(keys), nil
	}
	if plan.IsUnknown() {
		return nil, nil
	}
	planModel, diags := objectAs[OpenstackUserCredentialsModel](ctx, plan)
	if diags.HasError() {
		return nil, diags
	}
	var stateModel OpenstackUserCredentialsModel
	if !state.IsNull() && !state.IsUnknown() {
		var childDiags diag.Diagnostics
		stateModel, childDiags = objectAs[OpenstackUserCredentialsModel](ctx, state)
		diags.Append(childDiags...)
	}
	result := make(map[string]any)
	setChangedString(result, "projectID", planModel.ProjectID, stateModel.ProjectID)
	setChangedString(result, "project", planModel.ProjectName, stateModel.ProjectName)
	setChangedString(result, "username", planModel.Username, stateModel.Username)
	setChangedString(result, "password", planModel.Password, stateModel.Password)
	return result, diags
}

// patchApplicationCredentials clears every API credential field when the
// configured credential object is removed. Otherwise, it emits only changes.
func patchApplicationCredentials(ctx context.Context, plan, state types.Object) (map[string]any, diag.Diagnostics) {
	keys := []string{"applicationCredentialID", "applicationCredentialSecret"}
	if plan.IsNull() {
		if state.IsNull() || state.IsUnknown() {
			return nil, nil
		}
		return nullFields(keys), nil
	}
	if plan.IsUnknown() {
		return nil, nil
	}
	planModel, diags := objectAs[OpenstackApplicationCredentialsModel](ctx, plan)
	if diags.HasError() {
		return nil, diags
	}
	var stateModel OpenstackApplicationCredentialsModel
	if !state.IsNull() && !state.IsUnknown() {
		var childDiags diag.Diagnostics
		stateModel, childDiags = objectAs[OpenstackApplicationCredentialsModel](ctx, state)
		diags.Append(childDiags...)
	}
	result := make(map[string]any)
	setChangedString(result, "applicationCredentialID", planModel.ID, stateModel.ID)
	setChangedString(result, "applicationCredentialSecret", planModel.Secret, stateModel.Secret)
	return result, diags
}

// patchCIDR converts Terraform's single CIDR string into the API's list shape.
// Null and empty strings both clear the API field; unknown values are omitted by
// the caller.
func patchCIDR(value types.String) any {
	if value.IsNull() {
		return nil
	}
	if value.IsUnknown() {
		return nil
	}
	if value.ValueString() == "" {
		return nil
	}
	return map[string]any{"cidrBlocks": []string{value.ValueString()}}
}

// setChangedString adds a string only when plan differs from state. A null plan
// value is encoded as a JSON merge-patch deletion, while unknown values are
// omitted.
func setChangedString(target map[string]any, key string, plan, state types.String) {
	if plan.Equal(state) {
		return
	}
	if plan.IsNull() {
		target[key] = nil
		return
	}
	if !plan.IsUnknown() {
		target[key] = mergePatchString(plan)
	}
}

// setChangedBool applies the same tri-state merge-patch rules as
// setChangedString for boolean values.
func setChangedBool(target map[string]any, key string, plan, state types.Bool) {
	if plan.Equal(state) {
		return
	}
	if plan.IsNull() {
		target[key] = nil
		return
	}
	if !plan.IsUnknown() {
		target[key] = mergePatchBool(plan)
	}
}

// setConfiguredBool writes every configured child of an object being replaced.
// Null deletes a child and unknown leaves it out of the patch.
func setConfiguredBool(target map[string]any, key string, plan types.Bool) {
	if plan.IsNull() {
		target[key] = nil
		return
	}
	if !plan.IsUnknown() {
		target[key] = plan.ValueBool()
	}
}

// setConfiguredString writes every configured child of an object being
// replaced. Null deletes a child and unknown leaves it out of the patch.
func setConfiguredString(target map[string]any, key string, plan types.String) {
	if plan.IsNull() {
		target[key] = nil
		return
	}
	if !plan.IsUnknown() {
		target[key] = plan.ValueString()
	}
}

// setConfiguredInt32 writes every configured child of an object being replaced.
// Null deletes a child and unknown leaves it out of the patch.
func setConfiguredInt32(target map[string]any, key string, plan types.Int32) {
	if plan.IsNull() {
		target[key] = nil
		return
	}
	if !plan.IsUnknown() {
		target[key] = plan.ValueInt32()
	}
}

func mergePatchString(value types.String) any {
	if value.IsNull() {
		return nil
	}
	return value.ValueString()
}

func mergePatchBool(value types.Bool) any {
	if value.IsNull() {
		return nil
	}
	return value.ValueBool()
}

func nullFields(keys []string) map[string]any {
	result := make(map[string]any, len(keys))
	for _, key := range keys {
		result[key] = nil
	}
	return result
}
