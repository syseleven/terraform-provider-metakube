package resource_node_deployment

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/syseleven/go-metakube/models"
	"github.com/syseleven/terraform-provider-metakube/metakube/common"
	"k8s.io/utils/ptr"
)

// Flatteners convert MetaKube API models to Terraform values.

func flattenNodeDeploymentSpec(ctx context.Context, in *models.NodeDeploymentSpec) (types.Object, diag.Diagnostics) {
	if in == nil {
		return types.ObjectNull(nodeDeploymentSpecAttrTypes()), nil
	}

	model := NodeDeploymentSpecModel{
		Replicas:    int64Value(in.Replicas),
		MinReplicas: int64Value(in.MinReplicas),
		MaxReplicas: int64Value(in.MaxReplicas),
		Template:    types.ObjectNull(nodeSpecAttrTypes()),
	}
	var diags diag.Diagnostics
	if in.Template != nil {
		var childDiags diag.Diagnostics
		model.Template, childDiags = flattenNodeSpec(ctx, in.Template)
		diags.Append(childDiags...)
	}
	value, childDiags := objectValue(ctx, nodeDeploymentSpecAttrTypes(), model)
	diags.Append(childDiags...)
	return value, diags
}

func flattenNodeSpec(ctx context.Context, in *models.NodeSpec) (types.Object, diag.Diagnostics) {
	if in == nil {
		return types.ObjectNull(nodeSpecAttrTypes()), nil
	}

	model := NodeSpecModel{
		Cloud:              types.ObjectNull(cloudSpecAttrTypes()),
		OperatingSystem:    types.ObjectNull(operatingSystemAttrTypes()),
		Versions:           types.ObjectNull(versionsAttrTypes()),
		Labels:             flattenStringMap(in.Labels, true),
		AllLabels:          flattenStringMap(in.Labels, false),
		Taints:             types.ListNull(types.ObjectType{AttrTypes: taintAttrTypes()}),
		NodeAnnotations:    flattenStringMap(in.NodeAnnotations, false),
		MachineAnnotations: flattenStringMap(in.MachineAnnotations, false),
	}
	var diags diag.Diagnostics
	if in.Cloud != nil {
		var childDiags diag.Diagnostics
		model.Cloud, childDiags = flattenCloudSpec(ctx, in.Cloud)
		diags.Append(childDiags...)
	}
	if in.OperatingSystem != nil {
		var childDiags diag.Diagnostics
		model.OperatingSystem, childDiags = flattenOperatingSystem(ctx, in.OperatingSystem)
		diags.Append(childDiags...)
	}
	if in.Versions != nil && in.Versions.Kubelet != "" {
		var childDiags diag.Diagnostics
		model.Versions, childDiags = flattenVersions(ctx, in.Versions)
		diags.Append(childDiags...)
	}
	if len(in.Taints) > 0 {
		var childDiags diag.Diagnostics
		model.Taints, childDiags = flattenTaints(ctx, in.Taints)
		diags.Append(childDiags...)
	}
	value, childDiags := objectValue(ctx, nodeSpecAttrTypes(), model)
	diags.Append(childDiags...)
	return value, diags
}

func flattenCloudSpec(ctx context.Context, in *models.NodeCloudSpec) (types.Object, diag.Diagnostics) {
	if in == nil {
		return types.ObjectNull(cloudSpecAttrTypes()), nil
	}
	model := CloudSpecModel{
		OpenStack: types.ObjectNull(openstackCloudSpecAttrTypes()),
	}
	var diags diag.Diagnostics
	if in.Openstack != nil {
		var childDiags diag.Diagnostics
		model.OpenStack, childDiags = flattenOpenStackCloudSpec(ctx, in.Openstack)
		diags.Append(childDiags...)
	}
	value, childDiags := objectValue(ctx, cloudSpecAttrTypes(), model)
	diags.Append(childDiags...)
	return value, diags
}

func flattenOpenStackCloudSpec(ctx context.Context, in *models.OpenstackNodeSpec) (types.Object, diag.Diagnostics) {
	if in == nil {
		return types.ObjectNull(openstackCloudSpecAttrTypes()), nil
	}
	model := OpenStackCloudSpecModel{
		Flavor:                    stringPointerValue(in.Flavor),
		Image:                     stringPointerValue(in.Image),
		DiskSize:                  int64Value(in.RootDiskSizeGB),
		Tags:                      flattenStringMap(in.Tags, true),
		UseFloatingIP:             boolPointerValue(in.UseFloatingIP, true),
		InstanceReadyCheckPeriod:  stringDefaultValue(in.InstanceReadyCheckPeriod, "5s"),
		InstanceReadyCheckTimeout: stringDefaultValue(in.InstanceReadyCheckTimeout, "120s"),
		ServerGroupID:             stringNullIfEmpty(in.ServerGroupID),
	}
	return objectValue(ctx, openstackCloudSpecAttrTypes(), model)
}

func flattenOperatingSystem(ctx context.Context, in *models.OperatingSystemSpec) (types.Object, diag.Diagnostics) {
	if in == nil {
		return types.ObjectNull(operatingSystemAttrTypes()), nil
	}
	model := OperatingSystemModel{
		Ubuntu:  types.ObjectNull(ubuntuAttrTypes()),
		Flatcar: types.ObjectNull(flatcarAttrTypes()),
	}
	var diags diag.Diagnostics
	if in.Ubuntu != nil {
		var childDiags diag.Diagnostics
		model.Ubuntu, childDiags = objectValue(ctx, ubuntuAttrTypes(), UbuntuModel{
			DistUpgradeOnBoot: types.BoolValue(in.Ubuntu.DistUpgradeOnBoot),
		})
		diags.Append(childDiags...)
	}
	if in.Flatcar != nil {
		var childDiags diag.Diagnostics
		model.Flatcar, childDiags = objectValue(ctx, flatcarAttrTypes(), FlatcarModel{
			DisableAutoUpdate: types.BoolValue(in.Flatcar.DisableAutoUpdate),
		})
		diags.Append(childDiags...)
	}
	value, childDiags := objectValue(ctx, operatingSystemAttrTypes(), model)
	diags.Append(childDiags...)
	return value, diags
}

func flattenVersions(ctx context.Context, in *models.NodeVersionInfo) (types.Object, diag.Diagnostics) {
	if in == nil || in.Kubelet == "" {
		return types.ObjectNull(versionsAttrTypes()), nil
	}
	return objectValue(ctx, versionsAttrTypes(), VersionsModel{
		Kubelet: types.StringValue(in.Kubelet),
	})
}

func flattenTaints(ctx context.Context, in []*models.TaintSpec) (types.List, diag.Diagnostics) {
	if len(in) == 0 {
		return types.ListNull(types.ObjectType{AttrTypes: taintAttrTypes()}), nil
	}
	values := make([]attr.Value, 0, len(in))
	var diags diag.Diagnostics
	for _, taint := range in {
		if taint == nil {
			continue
		}
		value, childDiags := objectValue(ctx, taintAttrTypes(), TaintModel{
			Effect: types.StringValue(taint.Effect),
			Key:    types.StringValue(taint.Key),
			Value:  types.StringValue(taint.Value),
		})
		diags.Append(childDiags...)
		values = append(values, value)
	}
	list, childDiags := types.ListValue(types.ObjectType{AttrTypes: taintAttrTypes()}, values)
	diags.Append(childDiags...)
	return list, diags
}

// Expanders convert Terraform values to MetaKube API models.

func expandNodeDeploymentSpec(ctx context.Context, value types.Object, isCreate bool) (*models.NodeDeploymentSpec, diag.Diagnostics) {
	model, diags := common.ObjectAs[NodeDeploymentSpecModel](ctx, value)
	if value.IsNull() || value.IsUnknown() || diags.HasError() {
		return nil, diags
	}
	out := &models.NodeDeploymentSpec{}
	if !model.MinReplicas.IsNull() && !model.MinReplicas.IsUnknown() {
		minimum := int32(model.MinReplicas.ValueInt64())
		out.MinReplicas = &minimum
		if isCreate {
			out.Replicas = &minimum
		}
	}
	if !model.MaxReplicas.IsNull() && !model.MaxReplicas.IsUnknown() {
		out.MaxReplicas = ptr.To(int32(model.MaxReplicas.ValueInt64()))
	}
	if (out.MinReplicas == nil || *out.MinReplicas == 0) && !model.Replicas.IsNull() && !model.Replicas.IsUnknown() {
		out.Replicas = ptr.To(int32(model.Replicas.ValueInt64()))
	}
	if !model.Template.IsNull() && !model.Template.IsUnknown() {
		var childDiags diag.Diagnostics
		out.Template, childDiags = expandNodeSpec(ctx, model.Template)
		diags.Append(childDiags...)
	}
	return out, diags
}

func expandNodeSpec(ctx context.Context, value types.Object) (*models.NodeSpec, diag.Diagnostics) {
	model, diags := common.ObjectAs[NodeSpecModel](ctx, value)
	if value.IsNull() || value.IsUnknown() || diags.HasError() {
		return nil, diags
	}
	out := &models.NodeSpec{}

	var childDiags diag.Diagnostics
	out.Labels, childDiags = expandStringMap(ctx, model.Labels)
	diags.Append(childDiags...)
	out.NodeAnnotations, childDiags = expandStringMap(ctx, model.NodeAnnotations)
	diags.Append(childDiags...)
	out.MachineAnnotations, childDiags = expandStringMap(ctx, model.MachineAnnotations)
	diags.Append(childDiags...)

	if !model.Cloud.IsNull() && !model.Cloud.IsUnknown() {
		out.Cloud, childDiags = expandCloudSpec(ctx, model.Cloud)
		diags.Append(childDiags...)
	}
	if !model.OperatingSystem.IsNull() && !model.OperatingSystem.IsUnknown() {
		out.OperatingSystem, childDiags = expandOperatingSystem(ctx, model.OperatingSystem)
		diags.Append(childDiags...)
	}
	if !model.Versions.IsNull() && !model.Versions.IsUnknown() {
		out.Versions, childDiags = expandVersions(ctx, model.Versions)
		diags.Append(childDiags...)
	}
	if !model.Taints.IsNull() && !model.Taints.IsUnknown() {
		out.Taints, childDiags = expandTaints(ctx, model.Taints)
		diags.Append(childDiags...)
	}
	return out, diags
}

func expandCloudSpec(ctx context.Context, value types.Object) (*models.NodeCloudSpec, diag.Diagnostics) {
	model, diags := common.ObjectAs[CloudSpecModel](ctx, value)
	if value.IsNull() || value.IsUnknown() || diags.HasError() {
		return nil, diags
	}
	out := &models.NodeCloudSpec{}
	if !model.OpenStack.IsNull() && !model.OpenStack.IsUnknown() {
		var childDiags diag.Diagnostics
		out.Openstack, childDiags = expandOpenStackCloudSpec(ctx, model.OpenStack)
		diags.Append(childDiags...)
	}
	return out, diags
}

func expandOpenStackCloudSpec(ctx context.Context, value types.Object) (*models.OpenstackNodeSpec, diag.Diagnostics) {
	model, diags := common.ObjectAs[OpenStackCloudSpecModel](ctx, value)
	if value.IsNull() || value.IsUnknown() || diags.HasError() {
		return nil, diags
	}
	out := &models.OpenstackNodeSpec{}
	if !model.Flavor.IsNull() && !model.Flavor.IsUnknown() {
		out.Flavor = common.StrToPtr(model.Flavor.ValueString())
	}
	if !model.Image.IsNull() && !model.Image.IsUnknown() {
		out.Image = common.StrToPtr(model.Image.ValueString())
	}
	if !model.DiskSize.IsNull() && !model.DiskSize.IsUnknown() {
		out.RootDiskSizeGB = ptr.To(model.DiskSize.ValueInt64())
	}
	if !model.UseFloatingIP.IsNull() && !model.UseFloatingIP.IsUnknown() {
		out.UseFloatingIP = ptr.To(model.UseFloatingIP.ValueBool())
	}
	if !model.InstanceReadyCheckPeriod.IsNull() && !model.InstanceReadyCheckPeriod.IsUnknown() {
		out.InstanceReadyCheckPeriod = model.InstanceReadyCheckPeriod.ValueString()
	}
	if !model.InstanceReadyCheckTimeout.IsNull() && !model.InstanceReadyCheckTimeout.IsUnknown() {
		out.InstanceReadyCheckTimeout = model.InstanceReadyCheckTimeout.ValueString()
	}
	if !model.ServerGroupID.IsNull() && !model.ServerGroupID.IsUnknown() {
		out.ServerGroupID = model.ServerGroupID.ValueString()
	}
	var childDiags diag.Diagnostics
	out.Tags, childDiags = expandStringMap(ctx, model.Tags)
	diags.Append(childDiags...)
	return out, diags
}

func expandOperatingSystem(ctx context.Context, value types.Object) (*models.OperatingSystemSpec, diag.Diagnostics) {
	model, diags := common.ObjectAs[OperatingSystemModel](ctx, value)
	if value.IsNull() || value.IsUnknown() || diags.HasError() {
		return nil, diags
	}
	out := &models.OperatingSystemSpec{}
	if !model.Ubuntu.IsNull() && !model.Ubuntu.IsUnknown() {
		ubuntu, childDiags := common.ObjectAs[UbuntuModel](ctx, model.Ubuntu)
		diags.Append(childDiags...)
		if !ubuntu.DistUpgradeOnBoot.IsNull() && !ubuntu.DistUpgradeOnBoot.IsUnknown() {
			out.Ubuntu = &models.UbuntuSpec{DistUpgradeOnBoot: ubuntu.DistUpgradeOnBoot.ValueBool()}
		}
	}
	if !model.Flatcar.IsNull() && !model.Flatcar.IsUnknown() {
		flatcar, childDiags := common.ObjectAs[FlatcarModel](ctx, model.Flatcar)
		diags.Append(childDiags...)
		if !flatcar.DisableAutoUpdate.IsNull() && !flatcar.DisableAutoUpdate.IsUnknown() {
			out.Flatcar = &models.FlatcarSpec{DisableAutoUpdate: flatcar.DisableAutoUpdate.ValueBool()}
		}
	}
	return out, diags
}

func expandVersions(ctx context.Context, value types.Object) (*models.NodeVersionInfo, diag.Diagnostics) {
	model, diags := common.ObjectAs[VersionsModel](ctx, value)
	if value.IsNull() || value.IsUnknown() || diags.HasError() ||
		model.Kubelet.IsNull() || model.Kubelet.IsUnknown() || model.Kubelet.ValueString() == "" {
		return nil, diags
	}
	return &models.NodeVersionInfo{Kubelet: model.Kubelet.ValueString()}, diags
}

func expandTaints(ctx context.Context, value types.List) ([]*models.TaintSpec, diag.Diagnostics) {
	if value.IsNull() || value.IsUnknown() {
		return nil, nil
	}
	var modelsValue []TaintModel
	diags := value.ElementsAs(ctx, &modelsValue, false)
	if diags.HasError() {
		return nil, diags
	}
	out := make([]*models.TaintSpec, 0, len(modelsValue))
	for _, model := range modelsValue {
		out = append(out, &models.TaintSpec{
			Effect: model.Effect.ValueString(),
			Key:    model.Key.ValueString(),
			Value:  model.Value.ValueString(),
		})
	}
	return out, diags
}

func objectValue[T any](ctx context.Context, attrTypes map[string]attr.Type, model T) (types.Object, diag.Diagnostics) {
	return types.ObjectValueFrom(ctx, attrTypes, model)
}

func flattenStringMap(values map[string]string, userOnly bool) types.Map {
	elements := make(map[string]attr.Value)
	for key, value := range values {
		if !userOnly || !common.MetakubeResourceSystemLabelOrTag(key) {
			elements[key] = types.StringValue(value)
		}
	}
	return types.MapValueMust(types.StringType, elements)
}

func expandStringMap(ctx context.Context, value types.Map) (map[string]string, diag.Diagnostics) {
	if value.IsNull() {
		return nil, nil
	}
	var out map[string]string
	diags := value.ElementsAs(ctx, &out, false)
	return out, diags
}

func int64Value[T ~int32 | ~int64](value *T) types.Int64 {
	if value == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*value))
}

func stringPointerValue(value *string) types.String {
	if value == nil {
		return types.StringNull()
	}
	return types.StringValue(*value)
}

func stringNullIfEmpty(value string) types.String {
	if value == "" {
		return types.StringNull()
	}
	return types.StringValue(value)
}

func stringDefaultValue(value, defaultValue string) types.String {
	if value == "" {
		value = defaultValue
	}
	return types.StringValue(value)
}

func boolPointerValue(value *bool, defaultValue bool) types.Bool {
	if value == nil {
		return types.BoolValue(defaultValue)
	}
	return types.BoolValue(*value)
}
