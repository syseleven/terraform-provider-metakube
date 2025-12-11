package resource_cluster

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/syseleven/go-metakube/models"
	"k8s.io/utils/ptr"
)

// clusterPreserveValues holds values that need to be preserved during flatten operations
// because the API doesn't return sensitive data or to maintain consistency with planned state
type clusterPreserveValues struct {
	aws       *models.AWSCloudSpec
	openstack *clusterOpenstackPreservedValues
	azure     *models.AzureCloudSpec
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
		model.Spec = types.ListNull(types.ObjectType{AttrTypes: clusterSpecAttrTypes()})
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

	diags.Append(flattenCniPlugin(ctx, &specModel, in.CniPlugin)...)

	if in.Cloud != nil {
		diags.Append(flattenClusterCloudSpec(ctx, &specModel, preservedValues, in.Cloud)...)
	} else {
		specModel.Cloud = types.ListNull(types.ObjectType{AttrTypes: clusterCloudSpecAttrTypes()})
	}

	if in.Sys11auth != nil {
		diags.Append(flattenClusterSys11Auth(ctx, &specModel, in.Sys11auth)...)
	} else {
		specModel.SyselevenAuth = types.ListNull(types.ObjectType{AttrTypes: syselevenAuthAttrTypes()})
	}

	specObjVal, d := types.ObjectValueFrom(ctx, clusterSpecAttrTypes(), specModel)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}

	specList, d := types.ListValue(types.ObjectType{AttrTypes: clusterSpecAttrTypes()}, []attr.Value{specObjVal})
	diags.Append(d...)
	model.Spec = specList

	return diags
}

func getPreservedValuesFromModel(ctx context.Context, model *ClusterModel) clusterPreserveValues {
	values := clusterPreserveValues{}

	if model.Spec.IsNull() || model.Spec.IsUnknown() {
		return values
	}

	var specs []ClusterSpecModel
	if diags := model.Spec.ElementsAs(ctx, &specs, false); diags.HasError() || len(specs) == 0 {
		return values
	}

	if specs[0].Cloud.IsNull() || specs[0].Cloud.IsUnknown() {
		return values
	}

	var clouds []ClusterCloudSpecModel
	if diags := specs[0].Cloud.ElementsAs(ctx, &clouds, false); diags.HasError() || len(clouds) == 0 {
		return values
	}

	if !clouds[0].AWS.IsNull() && !clouds[0].AWS.IsUnknown() {
		var awsSpecs []AWSCloudSpecModel
		if diags := clouds[0].AWS.ElementsAs(ctx, &awsSpecs, false); !diags.HasError() && len(awsSpecs) > 0 {
			values.aws = &models.AWSCloudSpec{
				AccessKeyID:            awsSpecs[0].AccessKeyID.ValueString(),
				SecretAccessKey:        awsSpecs[0].SecretAccessKey.ValueString(),
				VPCID:                  awsSpecs[0].VPCID.ValueString(),
				SecurityGroupID:        awsSpecs[0].SecurityGroupID.ValueString(),
				RouteTableID:           awsSpecs[0].RouteTableID.ValueString(),
				InstanceProfileName:    awsSpecs[0].InstanceProfileName.ValueString(),
				ControlPlaneRoleARN:    awsSpecs[0].RoleARN.ValueString(),
				OpenstackBillingTenant: awsSpecs[0].OpenstackBillingTenant.ValueString(),
			}
		}
	}

	if !clouds[0].Openstack.IsNull() && !clouds[0].Openstack.IsUnknown() {
		var osSpecs []OpenstackCloudSpecModel
		if diags := clouds[0].Openstack.ElementsAs(ctx, &osSpecs, false); !diags.HasError() && len(osSpecs) > 0 {
			values.openstack = &clusterOpenstackPreservedValues{
				openstackServerGroupID: osSpecs[0].ServerGroupID,
			}

			if !osSpecs[0].UserCredentials.IsNull() && !osSpecs[0].UserCredentials.IsUnknown() {
				var userCreds []OpenstackUserCredentialsModel
				if diags := osSpecs[0].UserCredentials.ElementsAs(ctx, &userCreds, false); !diags.HasError() && len(userCreds) > 0 {
					values.openstack.openstackProjectID = userCreds[0].ProjectID
					values.openstack.openstackProjectName = userCreds[0].ProjectName
					values.openstack.openstackUsername = userCreds[0].Username
					values.openstack.openstackPassword = userCreds[0].Password
				}
			}

			if !osSpecs[0].ApplicationCredentials.IsNull() && !osSpecs[0].ApplicationCredentials.IsUnknown() {
				var appCreds []OpenstackApplicationCredentialsModel
				if diags := osSpecs[0].ApplicationCredentials.ElementsAs(ctx, &appCreds, false); !diags.HasError() && len(appCreds) > 0 {
					values.openstack.openstackApplicationCredentialsID = appCreds[0].ID
					values.openstack.openstackApplicationCredentialsSecret = appCreds[0].Secret
				}
			}
		}
	}

	if !clouds[0].Azure.IsNull() && !clouds[0].Azure.IsUnknown() {
		var azureSpecs []AzureCloudSpecModel
		if diags := clouds[0].Azure.ElementsAs(ctx, &azureSpecs, false); !diags.HasError() && len(azureSpecs) > 0 {
			values.azure = &models.AzureCloudSpec{
				AvailabilitySet:        azureSpecs[0].AvailabilitySet.ValueString(),
				ClientID:               azureSpecs[0].ClientID.ValueString(),
				ClientSecret:           azureSpecs[0].ClientSecret.ValueString(),
				SubscriptionID:         azureSpecs[0].SubscriptionID.ValueString(),
				TenantID:               azureSpecs[0].TenantID.ValueString(),
				ResourceGroup:          azureSpecs[0].ResourceGroup.ValueString(),
				RouteTableName:         azureSpecs[0].RouteTable.ValueString(),
				SecurityGroup:          azureSpecs[0].SecurityGroup.ValueString(),
				SubnetName:             azureSpecs[0].Subnet.ValueString(),
				VNetName:               azureSpecs[0].VNet.ValueString(),
				OpenstackBillingTenant: azureSpecs[0].OpenstackBillingTenant.ValueString(),
			}
		}
	}

	return values
}

func flattenUpdateWindow(ctx context.Context, specModel *ClusterSpecModel, in *models.UpdateWindow) diag.Diagnostics {
	var diags diag.Diagnostics

	if in == nil || (in.Start == "" && in.Length == "") {
		specModel.UpdateWindow = types.ListNull(types.ObjectType{AttrTypes: updateWindowAttrTypes()})
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

	listVal, d := types.ListValue(types.ObjectType{AttrTypes: updateWindowAttrTypes()}, []attr.Value{objVal})
	diags.Append(d...)
	specModel.UpdateWindow = listVal

	return diags
}

func flattenCniPlugin(ctx context.Context, specModel *ClusterSpecModel, in *models.CNIPluginSettings) diag.Diagnostics {
	var diags diag.Diagnostics

	cniType := "canal"
	if in != nil && in.Type != "" && in.Type != "none" {
		cniType = string(in.Type)
	}

	cniModel := CNIPluginModel{
		Type: types.StringValue(cniType),
	}

	objVal, d := types.ObjectValueFrom(ctx, cniPluginAttrTypes(), cniModel)
	diags.Append(d...)
	specModel.CNIPlugin = objVal

	return diags
}

func flattenClusterCloudSpec(ctx context.Context, specModel *ClusterSpecModel, values clusterPreserveValues, in *models.CloudSpec) diag.Diagnostics {
	var diags diag.Diagnostics

	if in == nil {
		specModel.Cloud = types.ListNull(types.ObjectType{AttrTypes: clusterCloudSpecAttrTypes()})
		return diags
	}

	cloudModel := ClusterCloudSpecModel{
		AWS:       types.ListNull(types.ObjectType{AttrTypes: awsCloudSpecAttrTypes()}),
		Openstack: types.ListNull(types.ObjectType{AttrTypes: openstackCloudSpecAttrTypes()}),
		Azure:     types.ListNull(types.ObjectType{AttrTypes: azureCloudSpecAttrTypes()}),
	}

	if in.Aws != nil {
		awsSpec := in.Aws
		if values.aws != nil {
			awsSpec = values.aws
		}
		diags.Append(flattenAWSCloudSpec(ctx, &cloudModel, awsSpec)...)
	}

	if in.Openstack != nil {
		diags.Append(flattenOpenstackSpec(ctx, &cloudModel, values.openstack, in.Openstack)...)
	}

	if in.Azure != nil {
		azureSpec := in.Azure
		if values.azure != nil {
			azureSpec = values.azure
		}
		diags.Append(flattenAzureSpec(ctx, &cloudModel, azureSpec)...)
	}

	objVal, d := types.ObjectValueFrom(ctx, clusterCloudSpecAttrTypes(), cloudModel)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}

	listVal, d := types.ListValue(types.ObjectType{AttrTypes: clusterCloudSpecAttrTypes()}, []attr.Value{objVal})
	diags.Append(d...)
	specModel.Cloud = listVal

	return diags
}

func flattenClusterSys11Auth(ctx context.Context, specModel *ClusterSpecModel, in *models.Sys11AuthSettings) diag.Diagnostics {
	var diags diag.Diagnostics

	if in == nil || (in.Realm == "" && in.IAMAuthentication == nil) {
		specModel.SyselevenAuth = types.ListNull(types.ObjectType{AttrTypes: syselevenAuthAttrTypes()})
		return diags
	}

	authModel := SyselevenAuthModel{}

	if in.Realm != "" {
		authModel.Realm = types.StringValue(in.Realm)
	} else {
		authModel.Realm = types.StringNull()
	}

	if in.IAMAuthentication != nil {
		authModel.IAMAuthentication = types.BoolValue(*in.IAMAuthentication)
	} else {
		authModel.IAMAuthentication = types.BoolNull()
	}

	objVal, d := types.ObjectValueFrom(ctx, syselevenAuthAttrTypes(), authModel)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}

	listVal, d := types.ListValue(types.ObjectType{AttrTypes: syselevenAuthAttrTypes()}, []attr.Value{objVal})
	diags.Append(d...)
	specModel.SyselevenAuth = listVal

	return diags
}

func flattenAWSCloudSpec(ctx context.Context, cloudModel *ClusterCloudSpecModel, in *models.AWSCloudSpec) diag.Diagnostics {
	var diags diag.Diagnostics

	if in == nil {
		cloudModel.AWS = types.ListNull(types.ObjectType{AttrTypes: awsCloudSpecAttrTypes()})
		return diags
	}

	awsModel := AWSCloudSpecModel{}

	if in.AccessKeyID != "" {
		awsModel.AccessKeyID = types.StringValue(in.AccessKeyID)
	} else {
		awsModel.AccessKeyID = types.StringNull()
	}

	if in.SecretAccessKey != "" {
		awsModel.SecretAccessKey = types.StringValue(in.SecretAccessKey)
	} else {
		awsModel.SecretAccessKey = types.StringNull()
	}

	if in.VPCID != "" {
		awsModel.VPCID = types.StringValue(in.VPCID)
	} else {
		awsModel.VPCID = types.StringNull()
	}

	if in.SecurityGroupID != "" {
		awsModel.SecurityGroupID = types.StringValue(in.SecurityGroupID)
	} else {
		awsModel.SecurityGroupID = types.StringNull()
	}

	if in.InstanceProfileName != "" {
		awsModel.InstanceProfileName = types.StringValue(in.InstanceProfileName)
	} else {
		awsModel.InstanceProfileName = types.StringNull()
	}

	if in.ControlPlaneRoleARN != "" {
		awsModel.RoleARN = types.StringValue(in.ControlPlaneRoleARN)
	} else {
		awsModel.RoleARN = types.StringNull()
	}

	if in.OpenstackBillingTenant != "" {
		awsModel.OpenstackBillingTenant = types.StringValue(in.OpenstackBillingTenant)
	} else {
		awsModel.OpenstackBillingTenant = types.StringNull()
	}

	if in.RouteTableID != "" {
		awsModel.RouteTableID = types.StringValue(in.RouteTableID)
	} else {
		awsModel.RouteTableID = types.StringNull()
	}

	objVal, d := types.ObjectValueFrom(ctx, awsCloudSpecAttrTypes(), awsModel)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}

	listVal, d := types.ListValue(types.ObjectType{AttrTypes: awsCloudSpecAttrTypes()}, []attr.Value{objVal})
	diags.Append(d...)
	cloudModel.AWS = listVal

	return diags
}

func flattenOpenstackSpec(ctx context.Context, cloudModel *ClusterCloudSpecModel, values *clusterOpenstackPreservedValues, in *models.OpenstackCloudSpec) diag.Diagnostics {
	var diags diag.Diagnostics

	if in == nil {
		cloudModel.Openstack = types.ListNull(types.ObjectType{AttrTypes: openstackCloudSpecAttrTypes()})
		return diags
	}

	osModel := OpenstackCloudSpecModel{
		UserCredentials:        types.ListNull(types.ObjectType{AttrTypes: openstackUserCredentialsAttrTypes()}),
		ApplicationCredentials: types.ListNull(types.ObjectType{AttrTypes: openstackApplicationCredentialsAttrTypes()}),
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
				listVal, d := types.ListValue(types.ObjectType{AttrTypes: openstackUserCredentialsAttrTypes()}, []attr.Value{objVal})
				diags.Append(d...)
				osModel.UserCredentials = listVal
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
				listVal, d := types.ListValue(types.ObjectType{AttrTypes: openstackApplicationCredentialsAttrTypes()}, []attr.Value{objVal})
				diags.Append(d...)
				osModel.ApplicationCredentials = listVal
			}
		}
	}

	objVal, d := types.ObjectValueFrom(ctx, openstackCloudSpecAttrTypes(), osModel)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}

	listVal, d := types.ListValue(types.ObjectType{AttrTypes: openstackCloudSpecAttrTypes()}, []attr.Value{objVal})
	diags.Append(d...)
	cloudModel.Openstack = listVal

	return diags
}

func flattenAzureSpec(ctx context.Context, cloudModel *ClusterCloudSpecModel, in *models.AzureCloudSpec) diag.Diagnostics {
	var diags diag.Diagnostics

	if in == nil {
		cloudModel.Azure = types.ListNull(types.ObjectType{AttrTypes: azureCloudSpecAttrTypes()})
		return diags
	}

	azureModel := AzureCloudSpecModel{}

	if in.AvailabilitySet != "" {
		azureModel.AvailabilitySet = types.StringValue(in.AvailabilitySet)
	} else {
		azureModel.AvailabilitySet = types.StringNull()
	}

	if in.ClientID != "" {
		azureModel.ClientID = types.StringValue(in.ClientID)
	} else {
		azureModel.ClientID = types.StringNull()
	}

	if in.ClientSecret != "" {
		azureModel.ClientSecret = types.StringValue(in.ClientSecret)
	} else {
		azureModel.ClientSecret = types.StringNull()
	}

	if in.SubscriptionID != "" {
		azureModel.SubscriptionID = types.StringValue(in.SubscriptionID)
	} else {
		azureModel.SubscriptionID = types.StringNull()
	}

	if in.TenantID != "" {
		azureModel.TenantID = types.StringValue(in.TenantID)
	} else {
		azureModel.TenantID = types.StringNull()
	}

	if in.ResourceGroup != "" {
		azureModel.ResourceGroup = types.StringValue(in.ResourceGroup)
	} else {
		azureModel.ResourceGroup = types.StringNull()
	}

	if in.RouteTableName != "" {
		azureModel.RouteTable = types.StringValue(in.RouteTableName)
	} else {
		azureModel.RouteTable = types.StringNull()
	}

	if in.OpenstackBillingTenant != "" {
		azureModel.OpenstackBillingTenant = types.StringValue(in.OpenstackBillingTenant)
	} else {
		azureModel.OpenstackBillingTenant = types.StringNull()
	}

	if in.SecurityGroup != "" {
		azureModel.SecurityGroup = types.StringValue(in.SecurityGroup)
	} else {
		azureModel.SecurityGroup = types.StringNull()
	}

	if in.SubnetName != "" {
		azureModel.Subnet = types.StringValue(in.SubnetName)
	} else {
		azureModel.Subnet = types.StringNull()
	}

	if in.VNetName != "" {
		azureModel.VNet = types.StringValue(in.VNetName)
	} else {
		azureModel.VNet = types.StringNull()
	}

	objVal, d := types.ObjectValueFrom(ctx, azureCloudSpecAttrTypes(), azureModel)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}

	listVal, d := types.ListValue(types.ObjectType{AttrTypes: azureCloudSpecAttrTypes()}, []attr.Value{objVal})
	diags.Append(d...)
	cloudModel.Azure = listVal

	return diags
}

// expanders

func metakubeResourceClusterExpandSpec(ctx context.Context, model *ClusterModel, dcName string, include func(string) bool) *models.ClusterSpec {
	if model.Spec.IsNull() || model.Spec.IsUnknown() {
		return nil
	}

	var specs []ClusterSpecModel
	if diags := model.Spec.ElementsAs(ctx, &specs, false); diags.HasError() || len(specs) == 0 {
		return nil
	}

	spec := specs[0]
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

	if !spec.CNIPlugin.IsUnknown() {
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
		obj.Cloud = expandClusterCloudSpec(ctx, spec.Cloud, dcName, func(k string) bool { return include("cloud.0." + k) })
	}

	// FIXME once we have proper server side validation for spec.BillingTenant we can remove this
	// for now copy it from cloud spec
	if obj.Cloud != nil && obj.Cloud.Aws != nil {
		obj.BillingTenant = obj.Cloud.Aws.OpenstackBillingTenant
	}

	if !spec.SyselevenAuth.IsNull() && !spec.SyselevenAuth.IsUnknown() && include("syseleven_auth") {
		obj.Sys11auth = expandClusterSys11Auth(ctx, spec.SyselevenAuth)
	}

	return obj
}

func expandUpdateWindow(ctx context.Context, list types.List) *models.UpdateWindow {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}

	var windows []UpdateWindowModel
	if diags := list.ElementsAs(ctx, &windows, false); diags.HasError() || len(windows) == 0 {
		return nil
	}

	ret := &models.UpdateWindow{}
	if !windows[0].Start.IsNull() && !windows[0].Start.IsUnknown() {
		ret.Start = windows[0].Start.ValueString()
	}
	if !windows[0].Length.IsNull() && !windows[0].Length.IsUnknown() {
		ret.Length = windows[0].Length.ValueString()
	}
	return ret
}

func expandAuditLogging(enabled bool) *models.AuditLoggingSettings {
	return &models.AuditLoggingSettings{
		Enabled: enabled,
	}
}

func expandCniPlugin(ctx context.Context, obj types.Object) *models.CNIPluginSettings {
	defaultCNI := &models.CNIPluginSettings{
		Type: models.CNIPluginType("canal"),
	}

	if obj.IsNull() || obj.IsUnknown() {
		return defaultCNI
	}

	var plugin CNIPluginModel
	if diags := obj.As(ctx, &plugin, basetypes.ObjectAsOptions{}); diags.HasError() {
		return defaultCNI
	}

	if plugin.Type.IsNull() || plugin.Type.IsUnknown() {
		return defaultCNI
	}

	v := plugin.Type.ValueString()
	if v == "" {
		return defaultCNI
	}

	return &models.CNIPluginSettings{
		Type: models.CNIPluginType(v),
	}
}

func expandClusterCloudSpec(ctx context.Context, list types.List, dcName string, include func(string) bool) *models.CloudSpec {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}

	var clouds []ClusterCloudSpecModel
	if diags := list.ElementsAs(ctx, &clouds, false); diags.HasError() || len(clouds) == 0 {
		return nil
	}

	obj := &models.CloudSpec{
		DatacenterName: dcName,
	}

	if !clouds[0].AWS.IsNull() && !clouds[0].AWS.IsUnknown() && include("aws") {
		obj.Aws = expandAWSCloudSpec(ctx, clouds[0].AWS, func(k string) bool { return include("aws.0." + k) })
	}

	if !clouds[0].Openstack.IsNull() && !clouds[0].Openstack.IsUnknown() && include("openstack") {
		obj.Openstack = expandOpenstackCloudSpec(ctx, clouds[0].Openstack, func(k string) bool { return include("openstack.0." + k) })
	}

	if !clouds[0].Azure.IsNull() && !clouds[0].Azure.IsUnknown() && include("azure") {
		obj.Azure = expandAzureCloudSpec(ctx, clouds[0].Azure, func(k string) bool { return include("azure.0." + k) })
	}

	return obj
}

func expandClusterSys11Auth(ctx context.Context, list types.List) *models.Sys11AuthSettings {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}

	var auths []SyselevenAuthModel
	if diags := list.ElementsAs(ctx, &auths, false); diags.HasError() || len(auths) == 0 {
		return nil
	}

	obj := &models.Sys11AuthSettings{}

	if !auths[0].IAMAuthentication.IsNull() && !auths[0].IAMAuthentication.IsUnknown() {
		obj.IAMAuthentication = ptr.To(auths[0].IAMAuthentication.ValueBool())
	}

	if !auths[0].Realm.IsNull() && !auths[0].Realm.IsUnknown() {
		v := auths[0].Realm.ValueString()
		if v != "" {
			obj.Realm = v
		}
	}

	return obj
}

func expandAWSCloudSpec(ctx context.Context, list types.List, include func(string) bool) *models.AWSCloudSpec {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}

	var awsSpecs []AWSCloudSpecModel
	if diags := list.ElementsAs(ctx, &awsSpecs, false); diags.HasError() || len(awsSpecs) == 0 {
		return nil
	}

	obj := &models.AWSCloudSpec{}
	aws := awsSpecs[0]

	if !aws.AccessKeyID.IsNull() && !aws.AccessKeyID.IsUnknown() && include("access_key_id") {
		v := aws.AccessKeyID.ValueString()
		if v != "" {
			obj.AccessKeyID = v
		}
	}

	if !aws.SecretAccessKey.IsNull() && !aws.SecretAccessKey.IsUnknown() && include("secret_access_key") {
		v := aws.SecretAccessKey.ValueString()
		if v != "" {
			obj.SecretAccessKey = v
		}
	}

	if !aws.VPCID.IsNull() && !aws.VPCID.IsUnknown() && include("vpc_id") {
		v := aws.VPCID.ValueString()
		if v != "" {
			obj.VPCID = v
		}
	}

	if !aws.SecurityGroupID.IsNull() && !aws.SecurityGroupID.IsUnknown() && include("security_group_id") {
		v := aws.SecurityGroupID.ValueString()
		if v != "" {
			obj.SecurityGroupID = v
		}
	}

	if !aws.InstanceProfileName.IsNull() && !aws.InstanceProfileName.IsUnknown() && include("instance_profile_name") {
		v := aws.InstanceProfileName.ValueString()
		if v != "" {
			obj.InstanceProfileName = v
		}
	}

	if !aws.RoleARN.IsNull() && !aws.RoleARN.IsUnknown() && include("role_arn") {
		v := aws.RoleARN.ValueString()
		if v != "" {
			obj.ControlPlaneRoleARN = v
		}
	}

	if !aws.OpenstackBillingTenant.IsNull() && !aws.OpenstackBillingTenant.IsUnknown() && include("openstack_billing_tenant") {
		v := aws.OpenstackBillingTenant.ValueString()
		if v != "" {
			obj.OpenstackBillingTenant = v
		}
	}

	if !aws.RouteTableID.IsNull() && !aws.RouteTableID.IsUnknown() && include("route_table_id") {
		v := aws.RouteTableID.ValueString()
		if v != "" {
			obj.RouteTableID = v
		}
	}

	return obj
}

func expandOpenstackCloudSpec(ctx context.Context, list types.List, include func(string) bool) *models.OpenstackCloudSpec {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}

	var osSpecs []OpenstackCloudSpecModel
	if diags := list.ElementsAs(ctx, &osSpecs, false); diags.HasError() || len(osSpecs) == 0 {
		return nil
	}

	obj := &models.OpenstackCloudSpec{}
	os := osSpecs[0]

	if !os.FloatingIPPool.IsNull() && !os.FloatingIPPool.IsUnknown() && include("floating_ip_pool") {
		v := os.FloatingIPPool.ValueString()
		if v != "" {
			obj.FloatingIPPool = v
		}
	}

	if !os.SecurityGroup.IsNull() && !os.SecurityGroup.IsUnknown() && include("security_group") {
		v := os.SecurityGroup.ValueString()
		if v != "" {
			obj.SecurityGroups = v
		}
	}

	if !os.Network.IsNull() && !os.Network.IsUnknown() && include("network") {
		v := os.Network.ValueString()
		if v != "" {
			obj.Network = v
		}
	}

	if !os.SubnetID.IsNull() && !os.SubnetID.IsUnknown() && include("subnet_id") {
		v := os.SubnetID.ValueString()
		if v != "" {
			obj.SubnetID = v
		}
	}

	if !os.SubnetCIDR.IsNull() && !os.SubnetCIDR.IsUnknown() && include("subnet_cidr") {
		v := os.SubnetCIDR.ValueString()
		if v != "" {
			obj.SubnetCIDR = v
		}
	}

	if !os.ServerGroupID.IsNull() && !os.ServerGroupID.IsUnknown() && include("server_group_id") {
		v := os.ServerGroupID.ValueString()
		if v != "" {
			obj.ServerGroupID = v
		}
	}

	if !os.ApplicationCredentials.IsNull() && !os.ApplicationCredentials.IsUnknown() {
		var appCreds []OpenstackApplicationCredentialsModel
		if diags := os.ApplicationCredentials.ElementsAs(ctx, &appCreds, false); !diags.HasError() && len(appCreds) > 0 {
			if !appCreds[0].ID.IsNull() && !appCreds[0].ID.IsUnknown() && include("application_credentials.0.id") {
				v := appCreds[0].ID.ValueString()
				if v != "" {
					obj.ApplicationCredentialID = v
				}
			}

			if !appCreds[0].Secret.IsNull() && !appCreds[0].Secret.IsUnknown() && include("application_credentials.0.secret") {
				v := appCreds[0].Secret.ValueString()
				if v != "" {
					obj.ApplicationCredentialSecret = v
				}
			}
		}
	}

	if !os.UserCredentials.IsNull() && !os.UserCredentials.IsUnknown() {
		var userCreds []OpenstackUserCredentialsModel
		if diags := os.UserCredentials.ElementsAs(ctx, &userCreds, false); !diags.HasError() && len(userCreds) > 0 {
			if !userCreds[0].Username.IsNull() && !userCreds[0].Username.IsUnknown() {
				v := userCreds[0].Username.ValueString()
				if v != "" {
					obj.Username = v
				}
			}
			if !userCreds[0].Password.IsNull() && !userCreds[0].Password.IsUnknown() {
				v := userCreds[0].Password.ValueString()
				if v != "" {
					obj.Password = v
				}
			}
			if !userCreds[0].ProjectID.IsNull() && !userCreds[0].ProjectID.IsUnknown() {
				v := userCreds[0].ProjectID.ValueString()
				if v != "" {
					obj.ProjectID = v
				}
			}
			if !userCreds[0].ProjectName.IsNull() && !userCreds[0].ProjectName.IsUnknown() {
				v := userCreds[0].ProjectName.ValueString()
				if v != "" {
					obj.Project = v
				}
			}
		}
	}

	// HACK(furkhat): API doesn't return domain for cluster. Use 'Default' all the time.
	obj.Domain = "Default"

	return obj
}

func expandAzureCloudSpec(ctx context.Context, list types.List, include func(string) bool) *models.AzureCloudSpec {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}

	var azureSpecs []AzureCloudSpecModel
	if diags := list.ElementsAs(ctx, &azureSpecs, false); diags.HasError() || len(azureSpecs) == 0 {
		return nil
	}

	obj := &models.AzureCloudSpec{}
	azure := azureSpecs[0]

	if !azure.AvailabilitySet.IsNull() && !azure.AvailabilitySet.IsUnknown() && include("availability_set") {
		v := azure.AvailabilitySet.ValueString()
		if v != "" {
			obj.AvailabilitySet = v
		}
	}

	if !azure.ClientID.IsNull() && !azure.ClientID.IsUnknown() && include("client_id") {
		v := azure.ClientID.ValueString()
		if v != "" {
			obj.ClientID = v
		}
	}

	if !azure.ClientSecret.IsNull() && !azure.ClientSecret.IsUnknown() && include("client_secret") {
		v := azure.ClientSecret.ValueString()
		if v != "" {
			obj.ClientSecret = v
		}
	}

	if !azure.SubscriptionID.IsNull() && !azure.SubscriptionID.IsUnknown() && include("subscription_id") {
		v := azure.SubscriptionID.ValueString()
		if v != "" {
			obj.SubscriptionID = v
		}
	}

	if !azure.TenantID.IsNull() && !azure.TenantID.IsUnknown() && include("tenant_id") {
		v := azure.TenantID.ValueString()
		if v != "" {
			obj.TenantID = v
		}
	}

	if !azure.ResourceGroup.IsNull() && !azure.ResourceGroup.IsUnknown() && include("resource_group") {
		v := azure.ResourceGroup.ValueString()
		if v != "" {
			obj.ResourceGroup = v
		}
	}

	if !azure.RouteTable.IsNull() && !azure.RouteTable.IsUnknown() && include("route_table") {
		v := azure.RouteTable.ValueString()
		if v != "" {
			obj.RouteTableName = v
		}
	}

	if !azure.OpenstackBillingTenant.IsNull() && !azure.OpenstackBillingTenant.IsUnknown() && include("openstack_billing_tenant") {
		v := azure.OpenstackBillingTenant.ValueString()
		if v != "" {
			obj.OpenstackBillingTenant = v
		}
	}

	if !azure.SecurityGroup.IsNull() && !azure.SecurityGroup.IsUnknown() && include("security_group") {
		v := azure.SecurityGroup.ValueString()
		if v != "" {
			obj.SecurityGroup = v
		}
	}

	if !azure.Subnet.IsNull() && !azure.Subnet.IsUnknown() && include("subnet") {
		v := azure.Subnet.ValueString()
		if v != "" {
			obj.SubnetName = v
		}
	}

	if !azure.VNet.IsNull() && !azure.VNet.IsUnknown() && include("vnet") {
		v := azure.VNet.ValueString()
		if v != "" {
			obj.VNetName = v
		}
	}

	return obj
}
