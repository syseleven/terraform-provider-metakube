package resource_cluster

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/syseleven/go-metakube/models"
	"k8s.io/utils/ptr"
)

// clusterPreserveValues holds values that need to be preserved during flatten operations
// because the API doesn't return sensitive data or to maintain consistency with planned state
type clusterPreserveValues struct {
	openstack *clusterOpenstackPreservedValues
}

type clusterOpenstackPreservedValues struct {
	openstackProjectID                    types.String
	openstackProjectName                  types.String
	openstackUsername                     types.String
	openstackPassword                     types.String
	openstackApplicationCredentialsID     types.String
	openstackApplicationCredentialsSecret types.String
	openstackServerGroupID                types.String
}

// flatteners

func metakubeResourceClusterFlattenSpec(ctx context.Context, model *ClusterModel, in *models.ClusterSpec) diag.Diagnostics {
	var diags diag.Diagnostics

	if in == nil {
		model.Spec = types.ObjectNull(clusterSpecAttrTypes())
		return diags
	}

	preservedValues := getPreservedValuesFromModel(ctx, model)
	specModel := ClusterSpecModel{}

	if in.Version != "" {
		specModel.Version = types.StringValue(string(in.Version))
	} else {
		specModel.Version = types.StringNull()
	}

	diags.Append(flattenUpdateWindow(ctx, &specModel, in.UpdateWindow)...)
	if diags.HasError() {
		return diags
	}

	if in.EnableUserSSHKeyAgent != nil {
		specModel.EnableSSHAgent = types.BoolValue(*in.EnableUserSSHKeyAgent)
	} else {
		specModel.EnableSSHAgent = types.BoolNull()
	}

	if in.AuditLogging != nil {
		specModel.AuditLogging = types.BoolValue(in.AuditLogging.Enabled)
	} else {
		specModel.AuditLogging = types.BoolValue(false)
	}

	specModel.PodSecurityPolicy = types.BoolValue(in.UsePodSecurityPolicyAdmissionPlugin)
	specModel.PodNodeSelector = types.BoolValue(in.UsePodNodeSelectorAdmissionPlugin)

	if network := in.ClusterNetwork; network != nil {
		if v := network.Pods; v != nil && len(v.CIDRBlocks) > 0 && v.CIDRBlocks[0] != "" {
			specModel.PodsCIDR = types.StringValue(v.CIDRBlocks[0])
		} else {
			specModel.PodsCIDR = types.StringNull()
		}
		if v := network.Services; v != nil && len(v.CIDRBlocks) > 0 && v.CIDRBlocks[0] != "" {
			specModel.ServicesCIDR = types.StringValue(v.CIDRBlocks[0])
		} else {
			specModel.ServicesCIDR = types.StringNull()
		}
		if network.IPFamily != "" {
			specModel.IPFamily = types.StringValue(string(network.IPFamily))
		} else {
			specModel.IPFamily = types.StringNull()
		}
	} else {
		specModel.PodsCIDR = types.StringNull()
		specModel.ServicesCIDR = types.StringNull()
		specModel.IPFamily = types.StringNull()
	}
	if in.CniPlugin != nil {
		diags.Append(flattenCniPlugin(ctx, &specModel, in.CniPlugin)...)
		if diags.HasError() {
			return diags
		}
	} else {
		specModel.CNIPlugin = types.ObjectNull(cniPluginAttrTypes())
	}

	if in.Cloud != nil {
		diags.Append(flattenClusterCloudSpec(ctx, &specModel, preservedValues, in.Cloud)...)
		if diags.HasError() {
			return diags
		}
	} else {
		specModel.Cloud = types.ObjectNull(clusterCloudSpecAttrTypes())
	}

	if in.Sys11auth != nil {
		diags.Append(flattenClusterSys11Auth(ctx, &specModel, in.Sys11auth)...)
		if diags.HasError() {
			return diags
		}
	} else {
		specModel.SyselevenAuth = types.ObjectNull(syselevenAuthAttrTypes())
	}

	specObjVal, d := types.ObjectValueFrom(ctx, clusterSpecAttrTypes(), specModel)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}

	model.Spec = specObjVal

	return diags
}

func getClusterSpecModel(ctx context.Context, model *ClusterModel) (ClusterSpecModel, bool) {
	if model == nil || model.Spec.IsNull() || model.Spec.IsUnknown() {
		return ClusterSpecModel{}, false
	}

	var spec ClusterSpecModel
	if diags := model.Spec.As(ctx, &spec, basetypes.ObjectAsOptions{}); diags.HasError() {
		return ClusterSpecModel{}, false
	}

	return spec, true
}

func getPreservedValuesFromModel(ctx context.Context, model *ClusterModel) clusterPreserveValues {
	values := clusterPreserveValues{}

	spec, ok := getClusterSpecModel(ctx, model)
	if !ok {
		return values
	}

	if spec.Cloud.IsNull() || spec.Cloud.IsUnknown() {
		return values
	}

	var cloud ClusterCloudSpecModel
	if diags := spec.Cloud.As(ctx, &cloud, basetypes.ObjectAsOptions{}); diags.HasError() {
		return values
	}

	if !cloud.Openstack.IsNull() && !cloud.Openstack.IsUnknown() {
		var osSpec OpenstackCloudSpecModel
		if diags := cloud.Openstack.As(ctx, &osSpec, basetypes.ObjectAsOptions{}); !diags.HasError() {
			values.openstack = &clusterOpenstackPreservedValues{
				openstackServerGroupID: osSpec.ServerGroupID,
			}

			if !osSpec.UserCredentials.IsNull() && !osSpec.UserCredentials.IsUnknown() {
				var userCreds OpenstackUserCredentialsModel
				if diags := osSpec.UserCredentials.As(ctx, &userCreds, basetypes.ObjectAsOptions{}); !diags.HasError() {
					values.openstack.openstackProjectID = userCreds.ProjectID
					values.openstack.openstackProjectName = userCreds.ProjectName
					values.openstack.openstackUsername = userCreds.Username
					values.openstack.openstackPassword = userCreds.Password
				}
			}

			if !osSpec.ApplicationCredentials.IsNull() && !osSpec.ApplicationCredentials.IsUnknown() {
				var appCreds OpenstackApplicationCredentialsModel
				if diags := osSpec.ApplicationCredentials.As(ctx, &appCreds, basetypes.ObjectAsOptions{}); !diags.HasError() {
					values.openstack.openstackApplicationCredentialsID = appCreds.ID
					values.openstack.openstackApplicationCredentialsSecret = appCreds.Secret
				}
			}
		}
	}

	return values
}

func flattenUpdateWindow(ctx context.Context, specModel *ClusterSpecModel, in *models.UpdateWindow) diag.Diagnostics {
	var diags diag.Diagnostics

	if in == nil || (in.Start == "" && in.Length == "") {
		specModel.UpdateWindow = types.ObjectNull(updateWindowAttrTypes())
		return diags
	}

	uwModel := UpdateWindowModel{
		Start:  types.StringValue(in.Start),
		Length: types.StringValue(in.Length),
	}

	objVal, d := types.ObjectValueFrom(ctx, updateWindowAttrTypes(), uwModel)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}

	specModel.UpdateWindow = objVal

	return diags
}

func flattenCniPlugin(ctx context.Context, specModel *ClusterSpecModel, in *models.CNIPluginSettings) diag.Diagnostics {
	var diags diag.Diagnostics

	if in == nil || in.Type == "" || in.Type == "none" {
		specModel.CNIPlugin = types.ObjectNull(cniPluginAttrTypes())
		return diags
	}

	cniModel := CNIPluginModel{
		Type:   types.StringValue(string(in.Type)),
		Cilium: types.ObjectNull(ciliumAttrTypes()),
	}

	if in.Cilium != nil {
		if diags.Append(flattenCniPluginCilium(ctx, &cniModel, in.Cilium)...); diags.HasError() {
			return diags
		}
	}
	objVal, d := types.ObjectValueFrom(ctx, cniPluginAttrTypes(), cniModel)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}

	specModel.CNIPlugin = objVal

	return diags
}

func flattenCniPluginCilium(ctx context.Context, cniModel *CNIPluginModel, in *models.CiliumCNISettings) diag.Diagnostics {
	var diags diag.Diagnostics

	ciliumModel := CiliumModel{
		EnableHubble:  types.BoolValue(in.EnableHubble),
		EnableL7Proxy: types.BoolValue(in.EnableL7Proxy),
		Clustermesh:   types.ObjectNull(ciliumClustermeshAttrTypes()),
	}

	if in.Clustermesh != nil {
		clustermeshModel := CiliumClustermeshModel{
			Enable:                types.BoolValue(*in.Clustermesh.Enable),
			ClusterID:             types.Int32Value(int32(in.Clustermesh.ClusterID)),
			IPv4NativeRoutingCIDR: types.StringValue(in.Clustermesh.IPV4NativeRoutingCIDR),
		}
		objVal, d := types.ObjectValueFrom(ctx, ciliumClustermeshAttrTypes(), clustermeshModel)
		diags.Append(d...)
		if diags.HasError() {
			return diags
		}

		ciliumModel.Clustermesh = objVal
	}
	objVal, d := types.ObjectValueFrom(ctx, ciliumAttrTypes(), ciliumModel)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}
	cniModel.Cilium = objVal
	return diags
}

func flattenClusterCloudSpec(ctx context.Context, specModel *ClusterSpecModel, values clusterPreserveValues, in *models.CloudSpec) diag.Diagnostics {
	var diags diag.Diagnostics

	if in == nil {
		specModel.Cloud = types.ObjectNull(clusterCloudSpecAttrTypes())
		return diags
	}

	cloudModel := ClusterCloudSpecModel{
		Openstack: types.ObjectNull(openstackCloudSpecAttrTypes()),
	}

	if in.Openstack != nil {
		diags.Append(flattenOpenstackSpec(ctx, &cloudModel, values.openstack, in.Openstack)...)
	}

	objVal, d := types.ObjectValueFrom(ctx, clusterCloudSpecAttrTypes(), cloudModel)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}

	specModel.Cloud = objVal

	return diags
}

func flattenClusterSys11Auth(ctx context.Context, specModel *ClusterSpecModel, in *models.Sys11AuthSettings) diag.Diagnostics {
	var diags diag.Diagnostics

	if in == nil {
		specModel.SyselevenAuth = types.ObjectNull(syselevenAuthAttrTypes())
		return diags
	}

	authModel := SyselevenAuthModel{}

	authModel.Realm = types.StringValue(ptr.Deref(in.Realm, ""))

	authModel.IAMAuthentication = types.BoolValue(ptr.Deref(in.IAMAuthentication, false))

	objVal, d := types.ObjectValueFrom(ctx, syselevenAuthAttrTypes(), authModel)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}

	specModel.SyselevenAuth = objVal

	return diags
}

func flattenOpenstackSpec(ctx context.Context, cloudModel *ClusterCloudSpecModel, values *clusterOpenstackPreservedValues, in *models.OpenstackCloudSpec) diag.Diagnostics {
	var diags diag.Diagnostics

	if in == nil {
		cloudModel.Openstack = types.ObjectNull(openstackCloudSpecAttrTypes())
		return diags
	}

	osModel := OpenstackCloudSpecModel{
		UserCredentials:        types.ObjectNull(openstackUserCredentialsAttrTypes()),
		ApplicationCredentials: types.ObjectNull(openstackApplicationCredentialsAttrTypes()),
	}

	if in.FloatingIPPool != "" {
		osModel.FloatingIPPool = types.StringValue(in.FloatingIPPool)
	} else {
		osModel.FloatingIPPool = types.StringNull()
	}

	if in.SecurityGroups != "" {
		osModel.SecurityGroup = types.StringValue(in.SecurityGroups)
	} else {
		osModel.SecurityGroup = types.StringNull()
	}

	if in.Network != "" {
		osModel.Network = types.StringValue(in.Network)
	} else {
		osModel.Network = types.StringNull()
	}

	if in.SubnetID != "" {
		osModel.SubnetID = types.StringValue(in.SubnetID)
	} else {
		osModel.SubnetID = types.StringNull()
	}

	if in.SubnetCIDR != "" {
		osModel.SubnetCIDR = types.StringValue(in.SubnetCIDR)
	} else {
		osModel.SubnetCIDR = types.StringNull()
	}

	if in.ServerGroupID != "" {
		osModel.ServerGroupID = types.StringValue(in.ServerGroupID)
	} else if values != nil && !values.openstackServerGroupID.IsNull() && values.openstackServerGroupID.ValueString() != "" {
		osModel.ServerGroupID = values.openstackServerGroupID
	} else {
		osModel.ServerGroupID = types.StringNull()
	}

	// Preserve user credentials from state (API doesn't return them)
	if values != nil {
		hasUserCreds := (!values.openstackProjectID.IsNull() && values.openstackProjectID.ValueString() != "") ||
			(!values.openstackProjectName.IsNull() && values.openstackProjectName.ValueString() != "") ||
			(!values.openstackUsername.IsNull() && values.openstackUsername.ValueString() != "") ||
			(!values.openstackPassword.IsNull() && values.openstackPassword.ValueString() != "")

		if hasUserCreds {
			userCredsModel := OpenstackUserCredentialsModel{
				ProjectID:   values.openstackProjectID,
				ProjectName: values.openstackProjectName,
				Username:    values.openstackUsername,
				Password:    values.openstackPassword,
			}

			objVal, d := types.ObjectValueFrom(ctx, openstackUserCredentialsAttrTypes(), userCredsModel)
			diags.Append(d...)
			if !diags.HasError() {
				osModel.UserCredentials = objVal
			}
		}

		hasAppCreds := (!values.openstackApplicationCredentialsID.IsNull() && values.openstackApplicationCredentialsID.ValueString() != "") ||
			(!values.openstackApplicationCredentialsSecret.IsNull() && values.openstackApplicationCredentialsSecret.ValueString() != "")

		if hasAppCreds {
			appCredsModel := OpenstackApplicationCredentialsModel{
				ID:     values.openstackApplicationCredentialsID,
				Secret: values.openstackApplicationCredentialsSecret,
			}

			objVal, d := types.ObjectValueFrom(ctx, openstackApplicationCredentialsAttrTypes(), appCredsModel)
			diags.Append(d...)
			if !diags.HasError() {
				osModel.ApplicationCredentials = objVal
			}
		}
	}

	objVal, d := types.ObjectValueFrom(ctx, openstackCloudSpecAttrTypes(), osModel)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}

	cloudModel.Openstack = objVal

	return diags
}

// expanders

func metakubeResourceClusterExpandSpec(ctx context.Context, model *ClusterModel, dcName string, include func(string) bool) *models.ClusterSpec {
	spec, ok := getClusterSpecModel(ctx, model)
	if !ok {
		return nil
	}
	obj := &models.ClusterSpec{}

	if !spec.Version.IsNull() && !spec.Version.IsUnknown() && include("version") {
		obj.Version = models.Semver(spec.Version.ValueString())
	}

	if !spec.UpdateWindow.IsNull() && !spec.UpdateWindow.IsUnknown() && include("update_window") {
		obj.UpdateWindow = expandUpdateWindow(ctx, spec.UpdateWindow)
	}

	if !spec.EnableSSHAgent.IsNull() && !spec.EnableSSHAgent.IsUnknown() && include("enable_ssh_agent") {
		v := spec.EnableSSHAgent.ValueBool()
		obj.EnableUserSSHKeyAgent = &v
	}

	if !spec.AuditLogging.IsNull() && !spec.AuditLogging.IsUnknown() && include("audit_logging") {
		obj.AuditLogging = expandAuditLogging(spec.AuditLogging.ValueBool())
	}

	if !spec.PodSecurityPolicy.IsNull() && !spec.PodSecurityPolicy.IsUnknown() && include("pod_security_policy") {
		obj.UsePodSecurityPolicyAdmissionPlugin = spec.PodSecurityPolicy.ValueBool()
	}

	if !spec.PodNodeSelector.IsNull() && !spec.PodNodeSelector.IsUnknown() && include("pod_node_selector") {
		obj.UsePodNodeSelectorAdmissionPlugin = spec.PodNodeSelector.ValueBool()
	}

	if !spec.ServicesCIDR.IsNull() && !spec.ServicesCIDR.IsUnknown() && include("services_cidr") {
		v := spec.ServicesCIDR.ValueString()
		if v != "" {
			if obj.ClusterNetwork == nil {
				obj.ClusterNetwork = &models.ClusterNetworkingConfig{}
			}
			obj.ClusterNetwork.Services = &models.NetworkRanges{
				CIDRBlocks: []string{v},
			}
		}
	}

	if !spec.PodsCIDR.IsNull() && !spec.PodsCIDR.IsUnknown() && include("pods_cidr") {
		v := spec.PodsCIDR.ValueString()
		if v != "" {
			if obj.ClusterNetwork == nil {
				obj.ClusterNetwork = &models.ClusterNetworkingConfig{}
			}
			obj.ClusterNetwork.Pods = &models.NetworkRanges{
				CIDRBlocks: []string{v},
			}
		}
	}

	if !spec.CNIPlugin.IsUnknown() && include("cni_plugin") {
		obj.CniPlugin = expandCniPlugin(ctx, spec.CNIPlugin)
	}

	if !spec.IPFamily.IsNull() && !spec.IPFamily.IsUnknown() && include("ip_family") {
		v := spec.IPFamily.ValueString()
		if v != "" {
			if obj.ClusterNetwork == nil {
				obj.ClusterNetwork = &models.ClusterNetworkingConfig{}
			}
			obj.ClusterNetwork.IPFamily = models.IPFamily(v)
		}
	}

	if !spec.Cloud.IsNull() && !spec.Cloud.IsUnknown() && include("cloud") {
		obj.Cloud = expandClusterCloudSpec(ctx, spec.Cloud, dcName, func(k string) bool { return include("cloud." + k) })
	}

	if !spec.SyselevenAuth.IsNull() && !spec.SyselevenAuth.IsUnknown() && include("syseleven_auth") {
		obj.Sys11auth = expandClusterSys11Auth(ctx, spec.SyselevenAuth)
	}

	return obj
}

func expandUpdateWindow(ctx context.Context, obj types.Object) *models.UpdateWindow {
	if obj.IsNull() || obj.IsUnknown() {
		return nil
	}

	var window UpdateWindowModel
	if diags := obj.As(ctx, &window, basetypes.ObjectAsOptions{}); diags.HasError() {
		return nil
	}

	ret := &models.UpdateWindow{}
	if !window.Start.IsNull() && !window.Start.IsUnknown() {
		ret.Start = window.Start.ValueString()
	}
	if !window.Length.IsNull() && !window.Length.IsUnknown() {
		ret.Length = window.Length.ValueString()
	}
	return ret
}

func expandAuditLogging(enabled bool) *models.AuditLoggingSettings {
	return &models.AuditLoggingSettings{
		Enabled: enabled,
	}
}

func expandCniPlugin(ctx context.Context, obj types.Object) *models.CNIPluginSettings {
	if obj.IsNull() || obj.IsUnknown() {
		return nil
	}

	var plugin CNIPluginModel
	if diags := obj.As(ctx, &plugin, basetypes.ObjectAsOptions{}); diags.HasError() {
		return nil
	}

	if plugin.Type.IsNull() || plugin.Type.IsUnknown() {
		return nil
	}

	v := plugin.Type.ValueString()
	if v == "" {
		return nil
	}

	var cniPlugin = models.CNIPluginSettings{
		Type:   models.CNIPluginType(v),
		Cilium: &models.CiliumCNISettings{},
	}

	if !plugin.Cilium.IsNull() {
		var cilium CiliumModel
		if diags := plugin.Cilium.As(ctx, &cilium, basetypes.ObjectAsOptions{}); !diags.HasError() {
			if !cilium.EnableHubble.IsNull() && !cilium.EnableHubble.IsUnknown() {
				cniPlugin.Cilium.EnableHubble = cilium.EnableHubble.ValueBool()
			}
			if !cilium.EnableL7Proxy.IsNull() && !cilium.EnableL7Proxy.IsUnknown() {
				cniPlugin.Cilium.EnableL7Proxy = cilium.EnableL7Proxy.ValueBool()
			}
			var clustermesh CiliumClustermeshModel
			if diags := cilium.Clustermesh.As(ctx, &clustermesh, basetypes.ObjectAsOptions{}); !diags.HasError() {
				cniPlugin.Cilium.Clustermesh = &models.CiliumClustermesh{
					Enable:                ptr.To(clustermesh.Enable.ValueBool()),
					ClusterID:             int64(clustermesh.ClusterID.ValueInt32()),
					IPV4NativeRoutingCIDR: clustermesh.IPv4NativeRoutingCIDR.ValueString(),
				}
			} else {
				return nil
			}
		} else {
			return nil
		}
	}
	return &cniPlugin
}

func expandClusterCloudSpec(ctx context.Context, obj types.Object, dcName string, include func(string) bool) *models.CloudSpec {
	if obj.IsNull() || obj.IsUnknown() {
		return nil
	}

	var cloud ClusterCloudSpecModel
	if diags := obj.As(ctx, &cloud, basetypes.ObjectAsOptions{}); diags.HasError() {
		return nil
	}

	ret := &models.CloudSpec{
		DatacenterName: dcName,
	}

	if !cloud.Openstack.IsNull() && !cloud.Openstack.IsUnknown() && include("openstack") {
		ret.Openstack = expandOpenstackCloudSpec(ctx, cloud.Openstack, func(k string) bool { return include("openstack." + k) })
	}

	return ret
}

func expandClusterSys11Auth(ctx context.Context, obj types.Object) *models.Sys11AuthSettings {
	if obj.IsNull() || obj.IsUnknown() {
		return nil
	}

	var auth SyselevenAuthModel
	if diags := obj.As(ctx, &auth, basetypes.ObjectAsOptions{}); diags.HasError() {
		return nil
	}

	ret := &models.Sys11AuthSettings{}

	if !auth.IAMAuthentication.IsNull() && !auth.IAMAuthentication.IsUnknown() {
		ret.IAMAuthentication = ptr.To(auth.IAMAuthentication.ValueBool())
	}

	if !auth.Realm.IsNull() && !auth.Realm.IsUnknown() {
		ret.Realm = ptr.To(auth.Realm.ValueString())
	}

	return ret
}

func expandOpenstackCloudSpec(ctx context.Context, obj types.Object, include func(string) bool) *models.OpenstackCloudSpec {
	if obj.IsNull() || obj.IsUnknown() {
		return nil
	}

	var os OpenstackCloudSpecModel
	if diags := obj.As(ctx, &os, basetypes.ObjectAsOptions{}); diags.HasError() {
		return nil
	}

	ret := &models.OpenstackCloudSpec{}

	if !os.FloatingIPPool.IsNull() && !os.FloatingIPPool.IsUnknown() && include("floating_ip_pool") {
		v := os.FloatingIPPool.ValueString()
		if v != "" {
			ret.FloatingIPPool = v
		}
	}

	if !os.SecurityGroup.IsNull() && !os.SecurityGroup.IsUnknown() && include("security_group") {
		v := os.SecurityGroup.ValueString()
		if v != "" {
			ret.SecurityGroups = v
		}
	}

	if !os.Network.IsNull() && !os.Network.IsUnknown() && include("network") {
		v := os.Network.ValueString()
		if v != "" {
			ret.Network = v
		}
	}

	if !os.SubnetID.IsNull() && !os.SubnetID.IsUnknown() && include("subnet_id") {
		v := os.SubnetID.ValueString()
		if v != "" {
			ret.SubnetID = v
		}
	}

	if !os.SubnetCIDR.IsNull() && !os.SubnetCIDR.IsUnknown() && include("subnet_cidr") {
		v := os.SubnetCIDR.ValueString()
		if v != "" {
			ret.SubnetCIDR = v
		}
	}

	if !os.ServerGroupID.IsNull() && !os.ServerGroupID.IsUnknown() && include("server_group_id") {
		v := os.ServerGroupID.ValueString()
		if v != "" {
			ret.ServerGroupID = v
		}
	}

	if !os.ApplicationCredentials.IsNull() && !os.ApplicationCredentials.IsUnknown() {
		var appCreds OpenstackApplicationCredentialsModel
		if diags := os.ApplicationCredentials.As(ctx, &appCreds, basetypes.ObjectAsOptions{}); !diags.HasError() {
			if !appCreds.ID.IsNull() && !appCreds.ID.IsUnknown() && include("application_credentials.id") {
				v := appCreds.ID.ValueString()
				if v != "" {
					ret.ApplicationCredentialID = v
				}
			}

			if !appCreds.Secret.IsNull() && !appCreds.Secret.IsUnknown() && include("application_credentials.secret") {
				v := appCreds.Secret.ValueString()
				if v != "" {
					ret.ApplicationCredentialSecret = v
				}
			}
		}
	}

	if !os.UserCredentials.IsNull() && !os.UserCredentials.IsUnknown() {
		var userCreds OpenstackUserCredentialsModel
		if diags := os.UserCredentials.As(ctx, &userCreds, basetypes.ObjectAsOptions{}); !diags.HasError() {
			if !userCreds.Username.IsNull() && !userCreds.Username.IsUnknown() {
				v := userCreds.Username.ValueString()
				if v != "" {
					ret.Username = v
				}
			}
			if !userCreds.Password.IsNull() && !userCreds.Password.IsUnknown() {
				v := userCreds.Password.ValueString()
				if v != "" {
					ret.Password = v
				}
			}
			if !userCreds.ProjectID.IsNull() && !userCreds.ProjectID.IsUnknown() {
				v := userCreds.ProjectID.ValueString()
				if v != "" {
					ret.ProjectID = v
				}
			}
			if !userCreds.ProjectName.IsNull() && !userCreds.ProjectName.IsUnknown() {
				v := userCreds.ProjectName.ValueString()
				if v != "" {
					ret.Project = v
				}
			}
		}
	}

	// HACK(furkhat): API doesn't return domain for cluster. Use 'Default' all the time.
	ret.Domain = "Default"

	return ret
}

func upgradeClusterLegacyNestedSpecState(rawState map[string]any) {
	spec, ok := rawState["spec"]
	if !ok {
		return
	}

	specMap, ok := spec.(map[string]any)
	if !ok {
		specList, ok := spec.([]any)
		if !ok {
			return
		}
		switch len(specList) {
		case 0:
			rawState["spec"] = nil
			return
		default:
			var listSpecMap map[string]any
			listSpecMap, ok = specList[0].(map[string]any)
			if !ok {
				return
			}
			specMap = listSpecMap
			rawState["spec"] = specMap
		}
	}

	upgradeSingleItemListToObject(specMap, "cni_plugin")
	upgradeSingleItemListToObject(specMap, "update_window")
	upgradeSingleItemListToObject(specMap, "cloud")
	upgradeSingleItemListToObject(specMap, "syseleven_auth")

	cloudMap, ok := specMap["cloud"].(map[string]any)
	if !ok {
		return
	}

	// Legacy SDK state may contain a now-unsupported cloud.azure and cloud.aws blocks.
	delete(cloudMap, "azure")
	delete(cloudMap, "aws")
	upgradeSingleItemListToObject(cloudMap, "openstack")

	openstackMap, ok := cloudMap["openstack"].(map[string]any)
	if !ok {
		return
	}

	upgradeSingleItemListToObject(openstackMap, "user_credentials")
	upgradeSingleItemListToObject(openstackMap, "application_credentials")
}

func upgradeSingleItemListToObject(parent map[string]any, key string) {
	value, ok := parent[key]
	if !ok {
		return
	}

	valueList, ok := value.([]any)
	if !ok {
		return
	}

	switch len(valueList) {
	case 0:
		parent[key] = nil
	default:
		if valueMap, ok := valueList[0].(map[string]any); ok {
			parent[key] = valueMap
		}
	}
}
