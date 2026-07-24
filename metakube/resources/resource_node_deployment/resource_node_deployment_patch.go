package resource_node_deployment

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/syseleven/terraform-provider-metakube/metakube/common"
)

// buildNodeDeploymentPatch derives an RFC 7396 JSON merge patch from the
// configuration, planned, and prior Terraform values. Configuration
// distinguishes an omitted optional-computed value from a computed plan value.
func buildNodeDeploymentPatch(
	ctx context.Context,
	config, plan, state NodeDeploymentModel,
) (map[string]any, diag.Diagnostics) {
	configSpec, diags := common.ObjectAs[NodeDeploymentSpecModel](ctx, config.Spec)
	planSpec, childDiags := common.ObjectAs[NodeDeploymentSpecModel](ctx, plan.Spec)
	diags.Append(childDiags...)
	stateSpec, childDiags := common.ObjectAs[NodeDeploymentSpecModel](ctx, state.Spec)
	diags.Append(childDiags...)
	if diags.HasError() {
		return nil, diags
	}

	specPatch := make(map[string]any)
	common.SetChangedInt64(specPatch, "replicas", planSpec.Replicas, stateSpec.Replicas)
	common.SetChangedInt64(specPatch, "minReplicas", planSpec.MinReplicas, stateSpec.MinReplicas)
	common.SetChangedInt64(specPatch, "maxReplicas", planSpec.MaxReplicas, stateSpec.MaxReplicas)

	templatePatch, childDiags := patchTemplate(ctx, configSpec.Template, planSpec.Template, stateSpec.Template)
	diags.Append(childDiags...)
	if diags.HasError() {
		return nil, diags
	}
	if len(templatePatch) > 0 {
		specPatch["template"] = templatePatch
	}

	if len(specPatch) == 0 {
		return map[string]any{}, diags
	}
	return map[string]any{"spec": specPatch}, diags
}

func patchTemplate(ctx context.Context, config, plan, state types.Object) (map[string]any, diag.Diagnostics) {
	configModel, diags := common.ObjectAs[NodeSpecModel](ctx, config)
	planModel, childDiags := common.ObjectAs[NodeSpecModel](ctx, plan)
	diags.Append(childDiags...)
	stateModel, childDiags := common.ObjectAs[NodeSpecModel](ctx, state)
	diags.Append(childDiags...)
	if diags.HasError() {
		return nil, diags
	}

	result := make(map[string]any)
	diags.Append(setChangedStringMap(result, "labels", planModel.Labels, stateModel.Labels)...)
	diags.Append(setChangedStringMap(result, "node_annotations", planModel.NodeAnnotations, stateModel.NodeAnnotations)...)
	diags.Append(setChangedStringMap(result, "machine_annotations", planModel.MachineAnnotations, stateModel.MachineAnnotations)...)
	value, include, valueDiags := patchTaints(ctx, planModel.Taints, stateModel.Taints)
	diags.Append(valueDiags...)
	if include {
		result["taints"] = value
	}

	cloud, childDiags := patchCloud(ctx, configModel.Cloud, planModel.Cloud, stateModel.Cloud)
	diags.Append(childDiags...)
	if len(cloud) > 0 {
		result["cloud"] = cloud
	}
	operatingSystem, childDiags := patchOperatingSystem(ctx, planModel.OperatingSystem, stateModel.OperatingSystem)
	diags.Append(childDiags...)
	if len(operatingSystem) > 0 {
		result["operatingSystem"] = operatingSystem
	}
	value, include, valueDiags = patchVersions(ctx, planModel.Versions, stateModel.Versions)
	diags.Append(valueDiags...)
	if include {
		result["versions"] = value
	}
	return result, diags
}

func patchCloud(ctx context.Context, config, plan, state types.Object) (map[string]any, diag.Diagnostics) {
	configModel, diags := common.ObjectAs[CloudSpecModel](ctx, config)
	planModel, childDiags := common.ObjectAs[CloudSpecModel](ctx, plan)
	diags.Append(childDiags...)
	stateModel, childDiags := common.ObjectAs[CloudSpecModel](ctx, state)
	diags.Append(childDiags...)
	if diags.HasError() {
		return nil, diags
	}

	result := make(map[string]any)
	value, include, childDiags := patchOpenstack(ctx, configModel.OpenStack, planModel.OpenStack, stateModel.OpenStack)
	diags.Append(childDiags...)
	if include {
		result["openstack"] = value
	}
	return result, diags
}

func patchOperatingSystem(ctx context.Context, plan, state types.Object) (map[string]any, diag.Diagnostics) {
	planModel, diags := common.ObjectAs[OperatingSystemModel](ctx, plan)
	stateModel, childDiags := common.ObjectAs[OperatingSystemModel](ctx, state)
	diags.Append(childDiags...)
	if diags.HasError() {
		return nil, diags
	}

	result := make(map[string]any)
	value, include, valueDiags := patchUbuntu(ctx, planModel.Ubuntu, stateModel.Ubuntu)
	diags.Append(valueDiags...)
	if include {
		result["ubuntu"] = value
	}
	value, include, valueDiags = patchFlatcar(ctx, planModel.Flatcar, stateModel.Flatcar)
	diags.Append(valueDiags...)
	if include {
		result["flatcar"] = value
	}
	return result, diags
}

// setChangedConfiguredString uses configuration to detect removal of an
// optional-computed value. A value omitted from configuration is cleared only
// when prior state contains a known value; otherwise ordinary plan/state
// comparison applies.
func setChangedConfiguredString(target map[string]any, key string, config, plan, state types.String) {
	if config.IsNull() {
		if !state.IsNull() && !state.IsUnknown() {
			target[key] = nil
		}
		return
	}
	common.SetChangedString(target, key, plan, state)
}

// setChangedStringMap patches maps by key. Removed keys become null, changed or
// added keys receive planned values, and an unknown map produces no patch.
func setChangedStringMap(target map[string]any, key string, plan, state types.Map) diag.Diagnostics {
	var diags diag.Diagnostics
	if plan.IsUnknown() || plan.Equal(state) || state.IsUnknown() {
		return diags
	}
	patch, err := common.StringMapMergePatch(plan, state)
	if err != nil {
		diags.AddError("Unable to build "+key+" patch", err.Error())
		return diags
	}
	target[key] = patch
	return diags
}

// patchTaints replaces the complete taints list when it changes. A null
// plan deletes the list, while an unknown plan leaves the remote value intact.
func patchTaints(ctx context.Context, plan, state types.List) (any, bool, diag.Diagnostics) {
	if plan.IsUnknown() || plan.Equal(state) {
		return nil, false, nil
	}
	if plan.IsNull() {
		return nil, !state.IsNull() && !state.IsUnknown(), nil
	}

	var taints []TaintModel
	diags := plan.ElementsAs(ctx, &taints, false)
	if diags.HasError() {
		return nil, false, diags
	}
	patch := make([]any, 0, len(taints))
	for _, taint := range taints {
		value := make(map[string]any)
		if !taint.Effect.IsNull() && !taint.Effect.IsUnknown() {
			value["effect"] = taint.Effect.ValueString()
		}
		if !taint.Key.IsNull() && !taint.Key.IsUnknown() {
			value["key"] = taint.Key.ValueString()
		}
		if !taint.Value.IsNull() && !taint.Value.IsUnknown() {
			value["value"] = taint.Value.ValueString()
		}
		patch = append(patch, value)
	}
	return patch, true, nil
}

func patchOpenstack(ctx context.Context, config, plan, state types.Object) (any, bool, diag.Diagnostics) {
	if plan.Equal(state) {
		return nil, false, nil
	}
	if plan.IsUnknown() {
		return nil, false, nil
	}
	if plan.IsNull() {
		return nil, !state.IsNull() && !state.IsUnknown(), nil
	}

	configModel, diags := common.ObjectAs[OpenStackCloudSpecModel](ctx, config)
	planModel, childDiags := common.ObjectAs[OpenStackCloudSpecModel](ctx, plan)
	diags.Append(childDiags...)
	stateModel, childDiags := common.ObjectAs[OpenStackCloudSpecModel](ctx, state)
	diags.Append(childDiags...)
	if diags.HasError() {
		return nil, false, diags
	}

	patch := make(map[string]any)
	common.SetChangedString(patch, "flavor", planModel.Flavor, stateModel.Flavor)
	common.SetChangedString(patch, "image", planModel.Image, stateModel.Image)
	common.SetChangedInt64(patch, "diskSize", planModel.DiskSize, stateModel.DiskSize)
	common.SetChangedBool(patch, "useFloatingIP", planModel.UseFloatingIP, stateModel.UseFloatingIP)
	common.SetChangedString(patch, "instanceReadyCheckPeriod", planModel.InstanceReadyCheckPeriod, stateModel.InstanceReadyCheckPeriod)
	common.SetChangedString(patch, "instanceReadyCheckTimeout", planModel.InstanceReadyCheckTimeout, stateModel.InstanceReadyCheckTimeout)
	setChangedConfiguredString(
		patch,
		"serverGroupID",
		configModel.ServerGroupID,
		planModel.ServerGroupID,
		stateModel.ServerGroupID,
	)
	diags.Append(setChangedStringMap(patch, "tags", planModel.Tags, stateModel.Tags)...)
	if diags.HasError() {
		return nil, false, diags
	}
	if len(patch) == 0 {
		return nil, false, diags
	}
	return patch, true, diags
}

func patchUbuntu(ctx context.Context, plan, state types.Object) (any, bool, diag.Diagnostics) {
	if plan.Equal(state) {
		return nil, false, nil
	}
	if plan.IsUnknown() {
		return nil, false, nil
	}
	if plan.IsNull() {
		return nil, !state.IsNull() && !state.IsUnknown(), nil
	}

	planModel, diags := common.ObjectAs[UbuntuModel](ctx, plan)
	stateModel, childDiags := common.ObjectAs[UbuntuModel](ctx, state)
	diags.Append(childDiags...)
	if diags.HasError() {
		return nil, false, diags
	}
	patch := make(map[string]any)
	common.SetChangedBool(patch, "distUpgradeOnBoot", planModel.DistUpgradeOnBoot, stateModel.DistUpgradeOnBoot)
	if len(patch) == 0 {
		return nil, false, diags
	}
	return patch, true, diags
}

func patchFlatcar(ctx context.Context, plan, state types.Object) (any, bool, diag.Diagnostics) {
	if plan.Equal(state) {
		return nil, false, nil
	}
	if plan.IsUnknown() {
		return nil, false, nil
	}
	if plan.IsNull() {
		return nil, !state.IsNull() && !state.IsUnknown(), nil
	}

	planModel, diags := common.ObjectAs[FlatcarModel](ctx, plan)
	stateModel, childDiags := common.ObjectAs[FlatcarModel](ctx, state)
	diags.Append(childDiags...)
	if diags.HasError() {
		return nil, false, diags
	}
	patch := make(map[string]any)
	common.SetChangedBool(patch, "disableAutoUpdate", planModel.DisableAutoUpdate, stateModel.DisableAutoUpdate)
	if len(patch) == 0 {
		return nil, false, diags
	}
	return patch, true, diags
}

func patchVersions(ctx context.Context, plan, state types.Object) (any, bool, diag.Diagnostics) {
	if plan.Equal(state) {
		return nil, false, nil
	}
	if plan.IsUnknown() {
		return nil, false, nil
	}
	if plan.IsNull() {
		return nil, !state.IsNull() && !state.IsUnknown(), nil
	}

	planModel, diags := common.ObjectAs[VersionsModel](ctx, plan)
	stateModel, childDiags := common.ObjectAs[VersionsModel](ctx, state)
	diags.Append(childDiags...)
	if diags.HasError() {
		return nil, false, diags
	}
	patch := make(map[string]any)
	common.SetChangedString(patch, "kubelet", planModel.Kubelet, stateModel.Kubelet)
	if len(patch) == 0 {
		return nil, false, diags
	}
	return patch, true, diags
}
