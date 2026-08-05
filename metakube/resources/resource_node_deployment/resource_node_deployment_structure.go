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

// Framework flatten functions - convert API models to framework types

func userTagValues(tags map[string]string) map[string]attr.Value {
	userTags := make(map[string]attr.Value)
	for k, v := range tags {
		if !common.MetakubeResourceSystemLabelOrTag(k) {
			userTags[k] = types.StringValue(v)
		}
	}
	return userTags
}

func flattenUserTags(tags map[string]string) (types.Map, diag.Diagnostics) {
	var diags diag.Diagnostics
	if len(tags) == 0 {
		return types.MapNull(types.StringType), diags
	}

	userTags := userTagValues(tags)
	if len(userTags) == 0 {
		return types.MapNull(types.StringType), diags
	}

	tagsVal, d := types.MapValue(types.StringType, userTags)
	diags.Append(d...)
	return tagsVal, diags
}

func flattenNodeDeploymentSpec(ctx context.Context, in *models.NodeDeploymentSpec, priorSpec ...types.List) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics

	if in == nil {
		return types.ListNull(types.ObjectType{AttrTypes: nodeDeploymentSpecAttrTypes()}), diags
	}

	specModel := NodeDeploymentSpecModel{}

	if in.Replicas != nil {
		specModel.Replicas = types.Int64Value(int64(*in.Replicas))
	} else {
		specModel.Replicas = types.Int64Null()
	}

	if in.MinReplicas != nil {
		specModel.MinReplicas = types.Int64Value(int64(*in.MinReplicas))
	} else {
		specModel.MinReplicas = types.Int64Null()
	}

	if in.MaxReplicas != nil {
		specModel.MaxReplicas = types.Int64Value(int64(*in.MaxReplicas))
	} else {
		specModel.MaxReplicas = types.Int64Null()
	}

	if in.Template != nil {
		var preserveEmptyNodeAnnotations, preserveEmptyMachineAnnotations bool
		if len(priorSpec) > 0 {
			preserveEmptyNodeAnnotations, preserveEmptyMachineAnnotations = knownEmptyAnnotations(ctx, priorSpec[0])
		}
		templateList, d := flattenNodeSpec(
			ctx,
			in.Template,
			preserveEmptyNodeAnnotations,
			preserveEmptyMachineAnnotations,
		)
		diags.Append(d...)
		specModel.Template = templateList
	} else {
		specModel.Template = types.ListNull(types.ObjectType{AttrTypes: nodeSpecAttrTypes()})
	}

	specObjVal, d := types.ObjectValueFrom(ctx, nodeDeploymentSpecAttrTypes(), specModel)
	diags.Append(d...)
	if diags.HasError() {
		return types.ListNull(types.ObjectType{AttrTypes: nodeDeploymentSpecAttrTypes()}), diags
	}

	specList, d := types.ListValue(types.ObjectType{AttrTypes: nodeDeploymentSpecAttrTypes()}, []attr.Value{specObjVal})
	diags.Append(d...)

	return specList, diags
}

func flattenNodeSpec(
	ctx context.Context,
	in *models.NodeSpec,
	preserveEmptyNodeAnnotations,
	preserveEmptyMachineAnnotations bool,
) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics

	if in == nil {
		return types.ListNull(types.ObjectType{AttrTypes: nodeSpecAttrTypes()}), diags
	}

	nodeSpecModel := NodeSpecModel{}

	// Labels
	if len(in.Labels) > 0 {
		allLabelsMap := make(map[string]attr.Value, len(in.Labels))
		userLabelsMap := make(map[string]attr.Value)

		for k, v := range in.Labels {
			allLabelsMap[k] = types.StringValue(v)
			if !common.MetakubeResourceSystemLabelOrTag(k) {
				userLabelsMap[k] = types.StringValue(v)
			}
		}

		allLabelsVal, d := types.MapValue(types.StringType, allLabelsMap)
		diags.Append(d...)
		nodeSpecModel.AllLabels = allLabelsVal

		if len(userLabelsMap) > 0 {
			labelsVal, d := types.MapValue(types.StringType, userLabelsMap)
			diags.Append(d...)
			nodeSpecModel.Labels = labelsVal
		} else {
			nodeSpecModel.Labels = types.MapNull(types.StringType)
		}
	} else {
		nodeSpecModel.Labels = types.MapNull(types.StringType)
		nodeSpecModel.AllLabels = types.MapNull(types.StringType)
	}

	// Node annotations
	if len(in.NodeAnnotations) > 0 {
		annoMap := make(map[string]attr.Value, len(in.NodeAnnotations))
		for k, v := range in.NodeAnnotations {
			annoMap[k] = types.StringValue(v)
		}
		annoVal, d := types.MapValue(types.StringType, annoMap)
		diags.Append(d...)
		nodeSpecModel.NodeAnnotations = annoVal
	} else if preserveEmptyNodeAnnotations {
		annoVal, d := types.MapValue(types.StringType, map[string]attr.Value{})
		diags.Append(d...)
		nodeSpecModel.NodeAnnotations = annoVal
	} else {
		nodeSpecModel.NodeAnnotations = types.MapNull(types.StringType)
	}

	// Machine annotations
	if len(in.MachineAnnotations) > 0 {
		annoMap := make(map[string]attr.Value, len(in.MachineAnnotations))
		for k, v := range in.MachineAnnotations {
			annoMap[k] = types.StringValue(v)
		}
		annoVal, d := types.MapValue(types.StringType, annoMap)
		diags.Append(d...)
		nodeSpecModel.MachineAnnotations = annoVal
	} else if preserveEmptyMachineAnnotations {
		annoVal, d := types.MapValue(types.StringType, map[string]attr.Value{})
		diags.Append(d...)
		nodeSpecModel.MachineAnnotations = annoVal
	} else {
		nodeSpecModel.MachineAnnotations = types.MapNull(types.StringType)
	}

	// Cloud
	if in.Cloud != nil {
		cloudList, d := flattenCloudSpec(ctx, in.Cloud)
		diags.Append(d...)
		nodeSpecModel.Cloud = cloudList
	} else {
		nodeSpecModel.Cloud = types.ListNull(types.ObjectType{AttrTypes: cloudSpecAttrTypes()})
	}

	// Operating system
	if in.OperatingSystem != nil {
		osList, d := flattenOperatingSystem(ctx, in.OperatingSystem)
		diags.Append(d...)
		nodeSpecModel.OperatingSystem = osList
	} else {
		nodeSpecModel.OperatingSystem = types.ListNull(types.ObjectType{AttrTypes: operatingSystemAttrTypes()})
	}

	// Versions
	if in.Versions != nil && in.Versions.Kubelet != "" {
		versionsList, d := flattenVersions(ctx, in.Versions)
		diags.Append(d...)
		nodeSpecModel.Versions = versionsList
	} else {
		nodeSpecModel.Versions = types.ListNull(types.ObjectType{AttrTypes: versionsAttrTypes()})
	}

	// Taints
	if len(in.Taints) > 0 {
		taintsList, d := flattenTaints(ctx, in.Taints)
		diags.Append(d...)
		nodeSpecModel.Taints = taintsList
	} else {
		nodeSpecModel.Taints = types.ListNull(types.ObjectType{AttrTypes: taintAttrTypes()})
	}

	objVal, d := types.ObjectValueFrom(ctx, nodeSpecAttrTypes(), nodeSpecModel)
	diags.Append(d...)
	if diags.HasError() {
		return types.ListNull(types.ObjectType{AttrTypes: nodeSpecAttrTypes()}), diags
	}

	listVal, d := types.ListValue(types.ObjectType{AttrTypes: nodeSpecAttrTypes()}, []attr.Value{objVal})
	diags.Append(d...)

	return listVal, diags
}

func knownEmptyAnnotations(ctx context.Context, specList types.List) (node, machine bool) {
	if specList.IsNull() || specList.IsUnknown() || len(specList.Elements()) != 1 {
		return false, false
	}

	var specs []NodeDeploymentSpecModel
	if d := specList.ElementsAs(ctx, &specs, false); d.HasError() || len(specs) != 1 {
		return false, false
	}

	templateList := specs[0].Template
	if templateList.IsNull() || templateList.IsUnknown() || len(templateList.Elements()) != 1 {
		return false, false
	}

	var templates []NodeSpecModel
	if d := templateList.ElementsAs(ctx, &templates, false); d.HasError() || len(templates) != 1 {
		return false, false
	}

	return knownEmptyMap(templates[0].NodeAnnotations), knownEmptyMap(templates[0].MachineAnnotations)
}

func knownEmptyMap(value types.Map) bool {
	return !value.IsNull() && !value.IsUnknown() && len(value.Elements()) == 0
}

func flattenCloudSpec(ctx context.Context, in *models.NodeCloudSpec) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics

	if in == nil {
		return types.ListNull(types.ObjectType{AttrTypes: cloudSpecAttrTypes()}), diags
	}

	cloudModel := CloudSpecModel{
		OpenStack: types.ListNull(types.ObjectType{AttrTypes: openstackCloudSpecAttrTypes()}),
	}

	if in.Openstack != nil {
		osList, d := flattenOpenStackCloudSpec(ctx, in.Openstack)
		diags.Append(d...)
		cloudModel.OpenStack = osList
	}

	objVal, d := types.ObjectValueFrom(ctx, cloudSpecAttrTypes(), cloudModel)
	diags.Append(d...)
	if diags.HasError() {
		return types.ListNull(types.ObjectType{AttrTypes: cloudSpecAttrTypes()}), diags
	}

	listVal, d := types.ListValue(types.ObjectType{AttrTypes: cloudSpecAttrTypes()}, []attr.Value{objVal})
	diags.Append(d...)

	return listVal, diags
}

func flattenOpenStackCloudSpec(ctx context.Context, in *models.OpenstackNodeSpec) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics

	if in == nil {
		return types.ListNull(types.ObjectType{AttrTypes: openstackCloudSpecAttrTypes()}), diags
	}

	osModel := OpenStackCloudSpecModel{}

	if in.Flavor != nil {
		osModel.Flavor = types.StringValue(*in.Flavor)
	} else {
		osModel.Flavor = types.StringNull()
	}

	if in.Image != nil {
		osModel.Image = types.StringValue(*in.Image)
	} else {
		osModel.Image = types.StringNull()
	}

	if in.UseFloatingIP != nil {
		osModel.UseFloatingIP = types.BoolValue(*in.UseFloatingIP)
	} else {
		osModel.UseFloatingIP = types.BoolValue(true)
	}

	if in.InstanceReadyCheckPeriod != "" {
		osModel.InstanceReadyCheckPeriod = types.StringValue(in.InstanceReadyCheckPeriod)
	} else {
		osModel.InstanceReadyCheckPeriod = types.StringValue("5s")
	}

	if in.InstanceReadyCheckTimeout != "" {
		osModel.InstanceReadyCheckTimeout = types.StringValue(in.InstanceReadyCheckTimeout)
	} else {
		osModel.InstanceReadyCheckTimeout = types.StringValue("120s")
	}

	if in.RootDiskSizeGB != nil && *in.RootDiskSizeGB != 0 {
		osModel.DiskSize = types.Int64Value(*in.RootDiskSizeGB)
	} else {
		osModel.DiskSize = types.Int64Null()
	}

	if in.ServerGroupID != "" {
		osModel.ServerGroupID = types.StringValue(in.ServerGroupID)
	} else {
		osModel.ServerGroupID = types.StringNull()
	}

	tagsVal, d := flattenUserTags(in.Tags)
	diags.Append(d...)
	osModel.Tags = tagsVal

	objVal, d := types.ObjectValueFrom(ctx, openstackCloudSpecAttrTypes(), osModel)
	diags.Append(d...)
	if diags.HasError() {
		return types.ListNull(types.ObjectType{AttrTypes: openstackCloudSpecAttrTypes()}), diags
	}

	listVal, d := types.ListValue(types.ObjectType{AttrTypes: openstackCloudSpecAttrTypes()}, []attr.Value{objVal})
	diags.Append(d...)

	return listVal, diags
}

func flattenOperatingSystem(ctx context.Context, in *models.OperatingSystemSpec) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics

	if in == nil {
		return types.ListNull(types.ObjectType{AttrTypes: operatingSystemAttrTypes()}), diags
	}

	osModel := OperatingSystemModel{
		Ubuntu:  types.ListNull(types.ObjectType{AttrTypes: ubuntuAttrTypes()}),
		Flatcar: types.ListNull(types.ObjectType{AttrTypes: flatcarAttrTypes()}),
	}

	if in.Ubuntu != nil {
		ubuntuList, d := flattenUbuntu(ctx, in.Ubuntu)
		diags.Append(d...)
		osModel.Ubuntu = ubuntuList
	}

	if in.Flatcar != nil {
		flatcarList, d := flattenFlatcar(ctx, in.Flatcar)
		diags.Append(d...)
		osModel.Flatcar = flatcarList
	}

	objVal, d := types.ObjectValueFrom(ctx, operatingSystemAttrTypes(), osModel)
	diags.Append(d...)
	if diags.HasError() {
		return types.ListNull(types.ObjectType{AttrTypes: operatingSystemAttrTypes()}), diags
	}

	listVal, d := types.ListValue(types.ObjectType{AttrTypes: operatingSystemAttrTypes()}, []attr.Value{objVal})
	diags.Append(d...)

	return listVal, diags
}

func flattenUbuntu(ctx context.Context, in *models.UbuntuSpec) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics

	if in == nil {
		return types.ListNull(types.ObjectType{AttrTypes: ubuntuAttrTypes()}), diags
	}

	ubuntuModel := UbuntuModel{
		DistUpgradeOnBoot: types.BoolValue(in.DistUpgradeOnBoot),
	}

	objVal, d := types.ObjectValueFrom(ctx, ubuntuAttrTypes(), ubuntuModel)
	diags.Append(d...)
	if diags.HasError() {
		return types.ListNull(types.ObjectType{AttrTypes: ubuntuAttrTypes()}), diags
	}

	listVal, d := types.ListValue(types.ObjectType{AttrTypes: ubuntuAttrTypes()}, []attr.Value{objVal})
	diags.Append(d...)

	return listVal, diags
}

func flattenFlatcar(ctx context.Context, in *models.FlatcarSpec) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics

	if in == nil {
		return types.ListNull(types.ObjectType{AttrTypes: flatcarAttrTypes()}), diags
	}

	flatcarModel := FlatcarModel{
		DisableAutoUpdate: types.BoolValue(in.DisableAutoUpdate),
	}

	objVal, d := types.ObjectValueFrom(ctx, flatcarAttrTypes(), flatcarModel)
	diags.Append(d...)
	if diags.HasError() {
		return types.ListNull(types.ObjectType{AttrTypes: flatcarAttrTypes()}), diags
	}

	listVal, d := types.ListValue(types.ObjectType{AttrTypes: flatcarAttrTypes()}, []attr.Value{objVal})
	diags.Append(d...)

	return listVal, diags
}

func flattenVersions(ctx context.Context, in *models.NodeVersionInfo) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics

	if in == nil || in.Kubelet == "" {
		return types.ListNull(types.ObjectType{AttrTypes: versionsAttrTypes()}), diags
	}

	versionsModel := VersionsModel{
		Kubelet: types.StringValue(in.Kubelet),
	}

	objVal, d := types.ObjectValueFrom(ctx, versionsAttrTypes(), versionsModel)
	diags.Append(d...)
	if diags.HasError() {
		return types.ListNull(types.ObjectType{AttrTypes: versionsAttrTypes()}), diags
	}

	listVal, d := types.ListValue(types.ObjectType{AttrTypes: versionsAttrTypes()}, []attr.Value{objVal})
	diags.Append(d...)

	return listVal, diags
}

func flattenTaints(ctx context.Context, in []*models.TaintSpec) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics

	if len(in) == 0 {
		return types.ListNull(types.ObjectType{AttrTypes: taintAttrTypes()}), diags
	}

	taintVals := make([]attr.Value, 0, len(in))
	for _, t := range in {
		if t == nil {
			continue
		}

		taintModel := TaintModel{
			Effect: types.StringValue(t.Effect),
			Key:    types.StringValue(t.Key),
			Value:  types.StringValue(t.Value),
		}

		objVal, d := types.ObjectValueFrom(ctx, taintAttrTypes(), taintModel)
		diags.Append(d...)
		if diags.HasError() {
			return types.ListNull(types.ObjectType{AttrTypes: taintAttrTypes()}), diags
		}

		taintVals = append(taintVals, objVal)
	}

	listVal, d := types.ListValue(types.ObjectType{AttrTypes: taintAttrTypes()}, taintVals)
	diags.Append(d...)

	return listVal, diags
}

// Framework expand functions - convert framework types to API models

func expandNodeDeploymentSpec(ctx context.Context, specList types.List, isCreate bool) (*models.NodeDeploymentSpec, diag.Diagnostics) {
	var diags diag.Diagnostics

	if specList.IsNull() || specList.IsUnknown() || len(specList.Elements()) == 0 {
		return nil, diags
	}

	var specModels []NodeDeploymentSpecModel
	diags.Append(specList.ElementsAs(ctx, &specModels, false)...)
	if diags.HasError() || len(specModels) == 0 {
		return nil, diags
	}

	spec := specModels[0]
	obj := &models.NodeDeploymentSpec{}

	// Handle autoscaler vs replicas
	if !spec.MinReplicas.IsNull() && !spec.MinReplicas.IsUnknown() {
		minVal := int32(spec.MinReplicas.ValueInt64())
		obj.MinReplicas = ptr.To(minVal)
		if isCreate {
			obj.Replicas = ptr.To(minVal)
		}
	}

	if !spec.MaxReplicas.IsNull() && !spec.MaxReplicas.IsUnknown() {
		obj.MaxReplicas = ptr.To(int32(spec.MaxReplicas.ValueInt64()))
	}

	// Only use replicas if autoscaler not configured
	if (obj.MinReplicas == nil || *obj.MinReplicas == 0) && !spec.Replicas.IsNull() && !spec.Replicas.IsUnknown() {
		obj.Replicas = ptr.To(int32(spec.Replicas.ValueInt64()))
	}

	if !spec.Template.IsNull() && !spec.Template.IsUnknown() {
		template, d := expandNodeSpec(ctx, spec.Template)
		diags.Append(d...)
		obj.Template = template
	}

	return obj, diags
}

func expandNodeSpec(ctx context.Context, templateList types.List) (*models.NodeSpec, diag.Diagnostics) {
	var diags diag.Diagnostics

	if templateList.IsNull() || templateList.IsUnknown() || len(templateList.Elements()) == 0 {
		return nil, diags
	}

	var templateModels []NodeSpecModel
	diags.Append(templateList.ElementsAs(ctx, &templateModels, false)...)
	if diags.HasError() || len(templateModels) == 0 {
		return nil, diags
	}

	tmpl := templateModels[0]
	obj := &models.NodeSpec{}

	// Labels
	if !tmpl.Labels.IsNull() && !tmpl.Labels.IsUnknown() {
		obj.Labels = make(map[string]string)
		labelsMap := tmpl.Labels.Elements()
		for k, v := range labelsMap {
			if strVal, ok := v.(types.String); ok && !strVal.IsNull() && !strVal.IsUnknown() {
				obj.Labels[k] = strVal.ValueString()
			}
		}
	}

	// Node annotations
	if !tmpl.NodeAnnotations.IsNull() && !tmpl.NodeAnnotations.IsUnknown() {
		obj.NodeAnnotations = make(map[string]string)
		annoMap := tmpl.NodeAnnotations.Elements()
		for k, v := range annoMap {
			if strVal, ok := v.(types.String); ok && !strVal.IsNull() && !strVal.IsUnknown() {
				obj.NodeAnnotations[k] = strVal.ValueString()
			}
		}
	}

	// Machine annotations
	if !tmpl.MachineAnnotations.IsNull() && !tmpl.MachineAnnotations.IsUnknown() {
		obj.MachineAnnotations = make(map[string]string)
		annoMap := tmpl.MachineAnnotations.Elements()
		for k, v := range annoMap {
			if strVal, ok := v.(types.String); ok && !strVal.IsNull() && !strVal.IsUnknown() {
				obj.MachineAnnotations[k] = strVal.ValueString()
			}
		}
	}

	// Cloud
	if !tmpl.Cloud.IsNull() && !tmpl.Cloud.IsUnknown() {
		cloud, d := expandCloudSpec(ctx, tmpl.Cloud)
		diags.Append(d...)
		obj.Cloud = cloud
	}

	// Operating system
	if !tmpl.OperatingSystem.IsNull() && !tmpl.OperatingSystem.IsUnknown() {
		os, d := expandOperatingSystem(ctx, tmpl.OperatingSystem)
		diags.Append(d...)
		obj.OperatingSystem = os
	}

	// Versions
	if !tmpl.Versions.IsNull() && !tmpl.Versions.IsUnknown() {
		versions, d := expandVersions(ctx, tmpl.Versions)
		diags.Append(d...)
		obj.Versions = versions
	}

	// Taints
	if !tmpl.Taints.IsNull() && !tmpl.Taints.IsUnknown() {
		taints, d := expandTaints(ctx, tmpl.Taints)
		diags.Append(d...)
		obj.Taints = taints
	}

	return obj, diags
}

func expandCloudSpec(ctx context.Context, cloudList types.List) (*models.NodeCloudSpec, diag.Diagnostics) {
	var diags diag.Diagnostics

	if cloudList.IsNull() || cloudList.IsUnknown() || len(cloudList.Elements()) == 0 {
		return nil, diags
	}

	var cloudModels []CloudSpecModel
	diags.Append(cloudList.ElementsAs(ctx, &cloudModels, false)...)
	if diags.HasError() || len(cloudModels) == 0 {
		return nil, diags
	}

	cloud := cloudModels[0]
	obj := &models.NodeCloudSpec{}

	// OpenStack
	if !cloud.OpenStack.IsNull() && !cloud.OpenStack.IsUnknown() && len(cloud.OpenStack.Elements()) > 0 {
		openstack, d := expandOpenStackCloudSpec(ctx, cloud.OpenStack)
		diags.Append(d...)
		obj.Openstack = openstack
	}

	return obj, diags
}

func expandOpenStackCloudSpec(ctx context.Context, osList types.List) (*models.OpenstackNodeSpec, diag.Diagnostics) {
	var diags diag.Diagnostics

	if osList.IsNull() || osList.IsUnknown() || len(osList.Elements()) == 0 {
		return nil, diags
	}

	var osModels []OpenStackCloudSpecModel
	diags.Append(osList.ElementsAs(ctx, &osModels, false)...)
	if diags.HasError() || len(osModels) == 0 {
		return nil, diags
	}

	os := osModels[0]
	obj := &models.OpenstackNodeSpec{}

	if !os.Flavor.IsNull() && !os.Flavor.IsUnknown() {
		obj.Flavor = common.StrToPtr(os.Flavor.ValueString())
	}

	if !os.Image.IsNull() && !os.Image.IsUnknown() {
		obj.Image = common.StrToPtr(os.Image.ValueString())
	}

	if !os.UseFloatingIP.IsNull() && !os.UseFloatingIP.IsUnknown() {
		obj.UseFloatingIP = ptr.To(os.UseFloatingIP.ValueBool())
	}

	if !os.InstanceReadyCheckPeriod.IsNull() && !os.InstanceReadyCheckPeriod.IsUnknown() {
		obj.InstanceReadyCheckPeriod = os.InstanceReadyCheckPeriod.ValueString()
	}

	if !os.InstanceReadyCheckTimeout.IsNull() && !os.InstanceReadyCheckTimeout.IsUnknown() {
		obj.InstanceReadyCheckTimeout = os.InstanceReadyCheckTimeout.ValueString()
	}

	if !os.DiskSize.IsNull() && !os.DiskSize.IsUnknown() {
		obj.RootDiskSizeGB = ptr.To(os.DiskSize.ValueInt64())
	}

	if !os.ServerGroupID.IsNull() && !os.ServerGroupID.IsUnknown() {
		obj.ServerGroupID = os.ServerGroupID.ValueString()
	}

	if !os.Tags.IsNull() && !os.Tags.IsUnknown() {
		obj.Tags = make(map[string]string)
		tagsMap := os.Tags.Elements()
		for k, v := range tagsMap {
			if strVal, ok := v.(types.String); ok && !strVal.IsNull() && !strVal.IsUnknown() {
				obj.Tags[k] = strVal.ValueString()
			}
		}
	}

	return obj, diags
}

func expandOperatingSystem(ctx context.Context, osList types.List) (*models.OperatingSystemSpec, diag.Diagnostics) {
	var diags diag.Diagnostics

	if osList.IsNull() || osList.IsUnknown() || len(osList.Elements()) == 0 {
		return nil, diags
	}

	var osModels []OperatingSystemModel
	diags.Append(osList.ElementsAs(ctx, &osModels, false)...)
	if diags.HasError() || len(osModels) == 0 {
		return nil, diags
	}

	os := osModels[0]
	obj := &models.OperatingSystemSpec{}

	if !os.Ubuntu.IsNull() && !os.Ubuntu.IsUnknown() && len(os.Ubuntu.Elements()) > 0 {
		ubuntu, d := expandUbuntu(ctx, os.Ubuntu)
		diags.Append(d...)
		obj.Ubuntu = ubuntu
	}

	if !os.Flatcar.IsNull() && !os.Flatcar.IsUnknown() && len(os.Flatcar.Elements()) > 0 {
		flatcar, d := expandFlatcar(ctx, os.Flatcar)
		diags.Append(d...)
		obj.Flatcar = flatcar
	}

	return obj, diags
}

func expandUbuntu(ctx context.Context, ubuntuList types.List) (*models.UbuntuSpec, diag.Diagnostics) {
	var diags diag.Diagnostics

	if ubuntuList.IsNull() || ubuntuList.IsUnknown() || len(ubuntuList.Elements()) == 0 {
		return nil, diags
	}

	var ubuntuModels []UbuntuModel
	diags.Append(ubuntuList.ElementsAs(ctx, &ubuntuModels, false)...)
	if diags.HasError() || len(ubuntuModels) == 0 {
		return nil, diags
	}

	ubuntu := ubuntuModels[0]
	obj := &models.UbuntuSpec{}

	if !ubuntu.DistUpgradeOnBoot.IsNull() && !ubuntu.DistUpgradeOnBoot.IsUnknown() {
		obj.DistUpgradeOnBoot = ubuntu.DistUpgradeOnBoot.ValueBool()
	}

	return obj, diags
}

func expandFlatcar(ctx context.Context, flatcarList types.List) (*models.FlatcarSpec, diag.Diagnostics) {
	var diags diag.Diagnostics

	if flatcarList.IsNull() || flatcarList.IsUnknown() || len(flatcarList.Elements()) == 0 {
		return nil, diags
	}

	var flatcarModels []FlatcarModel
	diags.Append(flatcarList.ElementsAs(ctx, &flatcarModels, false)...)
	if diags.HasError() || len(flatcarModels) == 0 {
		return nil, diags
	}

	flatcar := flatcarModels[0]
	obj := &models.FlatcarSpec{}

	if !flatcar.DisableAutoUpdate.IsNull() && !flatcar.DisableAutoUpdate.IsUnknown() {
		obj.DisableAutoUpdate = flatcar.DisableAutoUpdate.ValueBool()
	}

	return obj, diags
}

func expandVersions(ctx context.Context, versionsList types.List) (*models.NodeVersionInfo, diag.Diagnostics) {
	var diags diag.Diagnostics

	if versionsList.IsNull() || versionsList.IsUnknown() || len(versionsList.Elements()) == 0 {
		return nil, diags
	}

	var versionsModels []VersionsModel
	diags.Append(versionsList.ElementsAs(ctx, &versionsModels, false)...)
	if diags.HasError() || len(versionsModels) == 0 {
		return nil, diags
	}

	versions := versionsModels[0]

	if versions.Kubelet.IsNull() || versions.Kubelet.IsUnknown() || versions.Kubelet.ValueString() == "" {
		return nil, diags
	}

	return &models.NodeVersionInfo{
		Kubelet: versions.Kubelet.ValueString(),
	}, diags
}

func expandTaints(ctx context.Context, taintsList types.List) ([]*models.TaintSpec, diag.Diagnostics) {
	var diags diag.Diagnostics

	if taintsList.IsNull() || taintsList.IsUnknown() || len(taintsList.Elements()) == 0 {
		return nil, diags
	}

	var taintModels []TaintModel
	diags.Append(taintsList.ElementsAs(ctx, &taintModels, false)...)
	if diags.HasError() || len(taintModels) == 0 {
		return nil, diags
	}

	taints := make([]*models.TaintSpec, 0, len(taintModels))
	for _, t := range taintModels {
		taint := &models.TaintSpec{}

		if !t.Effect.IsNull() && !t.Effect.IsUnknown() {
			taint.Effect = t.Effect.ValueString()
		}

		if !t.Key.IsNull() && !t.Key.IsUnknown() {
			taint.Key = t.Key.ValueString()
		}

		if !t.Value.IsNull() && !t.Value.IsUnknown() {
			taint.Value = t.Value.ValueString()
		}

		taints = append(taints, taint)
	}

	return taints, diags
}

func getCloudProviderFromModel(ctx context.Context, model *NodeDeploymentModel) (string, diag.Diagnostics) {
	var diags diag.Diagnostics

	if model.Spec.IsNull() || model.Spec.IsUnknown() || len(model.Spec.Elements()) == 0 {
		return "", diags
	}

	var specModels []NodeDeploymentSpecModel
	diags.Append(model.Spec.ElementsAs(ctx, &specModels, false)...)
	if diags.HasError() || len(specModels) == 0 {
		return "", diags
	}

	spec := specModels[0]
	if spec.Template.IsNull() || spec.Template.IsUnknown() || len(spec.Template.Elements()) == 0 {
		return "", diags
	}

	var templateModels []NodeSpecModel
	diags.Append(spec.Template.ElementsAs(ctx, &templateModels, false)...)
	if diags.HasError() || len(templateModels) == 0 {
		return "", diags
	}

	tmpl := templateModels[0]
	if tmpl.Cloud.IsNull() || tmpl.Cloud.IsUnknown() || len(tmpl.Cloud.Elements()) == 0 {
		return "", diags
	}

	var cloudModels []CloudSpecModel
	diags.Append(tmpl.Cloud.ElementsAs(ctx, &cloudModels, false)...)
	if diags.HasError() || len(cloudModels) == 0 {
		return "", diags
	}

	cloud := cloudModels[0]

	if !cloud.OpenStack.IsNull() && !cloud.OpenStack.IsUnknown() && len(cloud.OpenStack.Elements()) > 0 {
		return "openstack", diags
	}

	return "", diags
}

func upgradeNodeDeploymentLegacyUnsupportedCloudState(rawState map[string]any) {
	spec, ok := rawState["spec"]
	if !ok {
		return
	}

	specList, ok := spec.([]any)
	if !ok {
		return
	}

	for _, specElem := range specList {
		specMap, ok := specElem.(map[string]any)
		if !ok {
			continue
		}

		template, ok := specMap["template"]
		if !ok {
			continue
		}

		templateList, ok := template.([]any)
		if !ok {
			continue
		}

		for _, templateElem := range templateList {
			templateMap, ok := templateElem.(map[string]any)
			if !ok {
				continue
			}

			cloud, ok := templateMap["cloud"]
			if !ok {
				continue
			}

			cloudList, ok := cloud.([]any)
			if !ok {
				continue
			}

			for _, cloudElem := range cloudList {
				cloudMap, ok := cloudElem.(map[string]any)
				if !ok {
					continue
				}

				// Legacy SDK state may contain now-unsupported cloud provider blocks.
				delete(cloudMap, "azure")
				delete(cloudMap, "aws")
			}
		}
	}
}
