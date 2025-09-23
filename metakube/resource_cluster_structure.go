package metakube

import (
	"github.com/syseleven/go-metakube/models"
)

// flatteners

func metakubeResourceClusterFlattenSpec(values clusterPreserveValues, in *models.ClusterSpec) []interface{} {
	if in == nil {
		return []interface{}{}
	}

	att := make(map[string]interface{})

	if in.Version != "" {
		att["version"] = string(in.Version)
	}

	if in.UpdateWindow != nil && (in.UpdateWindow.Start != "" || in.UpdateWindow.Length != "") {
		att["update_window"] = flattenUpdateWindow(in.UpdateWindow)
	}

	if in.EnableUserSSHKeyAgent != nil {
		att["enable_ssh_agent"] = *in.EnableUserSSHKeyAgent
	}

	att["audit_logging"] = false
	if in.AuditLogging != nil {
		att["audit_logging"] = in.AuditLogging.Enabled
	}

	att["pod_security_policy"] = in.UsePodSecurityPolicyAdmissionPlugin

	att["pod_node_selector"] = in.UsePodNodeSelectorAdmissionPlugin

	if network := in.ClusterNetwork; network != nil {
		if v := network.Pods; len(v.CIDRBlocks) > 0 && v.CIDRBlocks[0] != "" {
			att["pods_cidr"] = v.CIDRBlocks[0]
		}
		if v := network.Services; len(v.CIDRBlocks) > 0 && v.CIDRBlocks[0] != "" {
			att["services_cidr"] = v.CIDRBlocks[0]
		}
		if network.IPFamily != "" {
			att["ip_family"] = string(network.IPFamily)
		}
	}

	if in.CniPlugin != nil && in.CniPlugin.Type != "" {
		att["cni_plugin"] = flattenCniPlugin(in.CniPlugin)
	}

	if in.Cloud != nil {
		att["cloud"] = flattenClusterCloudSpec(values, in.Cloud)
	}

	if in.Sys11auth != nil {
		att["syseleven_auth"] = flattenClusterSys11Auth(in.Sys11auth)
	}

	return []interface{}{att}
}

func flattenUpdateWindow(in *models.UpdateWindow) []interface{} {
	m := make(map[string]interface{})
	m["start"] = in.Start
	m["length"] = in.Length
	return []interface{}{m}
}

func flattenCniPlugin(in *models.CNIPluginSettings) []interface{} {
	if in == nil {
		return []interface{}{}
	}

	m := make(map[string]interface{})
	m["type"] = string(in.Type)

	return []interface{}{m}
}

func flattenClusterCloudSpec(values clusterPreserveValues, in *models.CloudSpec) []interface{} {
	if in == nil {
		return []interface{}{}
	}

	att := make(map[string]interface{})

	if in.Aws != nil {
		att["aws"] = flattenAWSCloudSpec(values.aws)
	}

	if in.Openstack != nil {
		att["openstack"] = flattenOpenstackSpec(values.openstack, in.Openstack)
	}

	if in.Azure != nil {
		att["azure"] = flattenAzureSpec(values.azure)
	}

	return []interface{}{att}
}

func flattenClusterSys11Auth(in *models.Sys11AuthSettings) []interface{} {
	if in == nil || in.Realm == "" {
		return nil
	}

	return []interface{}{map[string]interface{}{
		"realm": in.Realm,
	}}
}

func flattenAWSCloudSpec(in *models.AWSCloudSpec) []interface{} {
	if in == nil {
		return []interface{}{}
	}

	att := make(map[string]interface{})

	if in.AccessKeyID != "" {
		att["access_key_id"] = in.AccessKeyID
	}

	if in.SecretAccessKey != "" {
		att["secret_access_key"] = in.SecretAccessKey
	}

	if in.VPCID != "" {
		att["vpc_id"] = in.VPCID
	}

	if in.SecurityGroupID != "" {
		att["security_group_id"] = in.SecurityGroupID
	}

	if in.InstanceProfileName != "" {
		att["instance_profile_name"] = in.InstanceProfileName
	}

	if in.ControlPlaneRoleARN != "" {
		att["role_arn"] = in.ControlPlaneRoleARN
	}

	if in.OpenstackBillingTenant != "" {
		att["openstack_billing_tenant"] = in.OpenstackBillingTenant
	}

	if in.RouteTableID != "" {
		att["route_table_id"] = in.RouteTableID
	}

	return []interface{}{att}
}

func flattenOpenstackSpec(values *clusterOpenstackPreservedValues, in *models.OpenstackCloudSpec) []interface{} {
	if in == nil {
		return []interface{}{}
	}

	att := make(map[string]interface{})

	if in.FloatingIPPool != "" {
		att["floating_ip_pool"] = in.FloatingIPPool
	}

	if in.SecurityGroups != "" {
		att["security_group"] = in.SecurityGroups
	}

	if in.Network != "" {
		att["network"] = in.Network
	}

	if in.SubnetID != "" {
		att["subnet_id"] = in.SubnetID
	}

	if in.SubnetCIDR != "" {
		att["subnet_cidr"] = in.SubnetCIDR
	}

	if in.ServerGroupID != "" {
		att["server_group_id"] = in.ServerGroupID
	}

	if values != nil {
		if _, ok := att["server_group_id"]; !ok && values.openstackServerGroupID != nil {
			att["server_group_id"] = values.openstackServerGroupID
		}
		if values.openstackProjectID != nil || values.openstackProjectName != nil || values.openstackUsername != nil || values.openstackPassword != nil {
			m := make(map[string]interface{})
			if values.openstackProjectID != nil {
				if v := values.openstackProjectID.(string); v != "" {
					m["project_id"] = values.openstackProjectID
				}
			}
			if values.openstackProjectName != nil {
				if v := values.openstackProjectName.(string); v != "" {
					m["project_name"] = values.openstackProjectName
				}
			}
			if values.openstackUsername != nil {
				if v := values.openstackUsername.(string); v != "" {
					m["username"] = values.openstackUsername
				}
			}
			if values.openstackPassword != nil {
				if v := values.openstackPassword.(string); v != "" {
					m["password"] = values.openstackPassword
				}
			}
			if len(m) > 0 {
				att["user_credentials"] = []interface{}{m}
			}
		}
		if values.openstackApplicationCredentialsID != nil || values.openstackApplicationCredentialsSecret != nil {
			m := make(map[string]interface{})
			if values.openstackApplicationCredentialsID != nil {
				id := values.openstackApplicationCredentialsID.(string)
				if id != "" {
					m["id"] = values.openstackApplicationCredentialsID
				}
			}
			if values.openstackApplicationCredentialsSecret != nil {
				secret := values.openstackApplicationCredentialsSecret.(string)
				if secret != "" {
					m["secret"] = values.openstackApplicationCredentialsSecret
				}
			}
			if len(m) > 0 {
				att["application_credentials"] = []interface{}{m}
			}
		}
	}

	return []interface{}{att}
}

func flattenAzureSpec(in *models.AzureCloudSpec) []interface{} {
	if in == nil {
		return []interface{}{}
	}

	// API returns empty spec for Azure clusters, so we just preserve values used for creation

	att := make(map[string]interface{})

	if in.AvailabilitySet != "" {
		att["availability_set"] = in.AvailabilitySet
	}

	if in.ClientID != "" {
		att["client_id"] = in.ClientID
	}

	if in.ClientSecret != "" {
		att["client_secret"] = in.ClientSecret
	}

	if in.SubscriptionID != "" {
		att["subscription_id"] = in.SubscriptionID
	}

	if in.TenantID != "" {
		att["tenant_id"] = in.TenantID
	}

	if in.ResourceGroup != "" {
		att["resource_group"] = in.ResourceGroup
	}

	if in.RouteTableName != "" {
		att["route_table"] = in.RouteTableName
	}

	if in.OpenstackBillingTenant != "" {
		att["openstack_billing_tenant"] = in.OpenstackBillingTenant
	}

	if in.SecurityGroup != "" {
		att["security_group"] = in.SecurityGroup
	}

	if in.SubnetName != "" {
		att["subnet"] = in.SubnetName
	}

	if in.VNetName != "" {
		att["vnet"] = in.VNetName
	}

	return []interface{}{att}
}

// expanders

func metakubeResourceClusterExpandSpec(p []interface{}, dcName string, include func(string) bool) *models.ClusterSpec {
	if len(p) < 1 {
		return nil
	}
	obj := &models.ClusterSpec{}
	if p[0] == nil {
		return obj
	}
	in := p[0].(map[string]interface{})

	if v, ok := in["version"]; ok {
		if vv, ok := v.(string); ok && include("version") {
			obj.Version = models.Semver(vv)
		}
	}

	if v, ok := in["update_window"]; ok {
		if vv, ok := v.([]interface{}); ok && include("update_window") {
			obj.UpdateWindow = expandUpdateWindow(vv)
		}
	}

	if v, ok := in["enable_ssh_agent"]; ok && include("enable_ssh_agent") {
		if vv, ok := v.(bool); ok {
			obj.EnableUserSSHKeyAgent = &vv
		}
	}

	if v, ok := in["audit_logging"]; ok && include("audit_logging") {
		if vv, ok := v.(bool); ok {
			obj.AuditLogging = expandAuditLogging(vv)
		}
	}

	if v, ok := in["pod_security_policy"]; ok && include("pod_security_policy") {
		if vv, ok := v.(bool); ok {
			obj.UsePodSecurityPolicyAdmissionPlugin = vv
		}
	}

	if v, ok := in["pod_node_selector"]; ok && include("pod_node_selector") {
		if vv, ok := v.(bool); ok {
			obj.UsePodNodeSelectorAdmissionPlugin = vv
		}
	}

	if v, ok := in["services_cidr"]; ok && include("services_cidr") {
		if vv, ok := v.(string); ok && vv != "" {
			if obj.ClusterNetwork == nil {
				obj.ClusterNetwork = &models.ClusterNetworkingConfig{}
			}
			obj.ClusterNetwork.Services = &models.NetworkRanges{
				CIDRBlocks: []string{vv},
			}
		}
	}

	if v, ok := in["pods_cidr"]; ok && include("pods_cidr") {
		if vv, ok := v.(string); ok && vv != "" {
			if obj.ClusterNetwork == nil {
				obj.ClusterNetwork = &models.ClusterNetworkingConfig{}
			}
			obj.ClusterNetwork.Pods = &models.NetworkRanges{
				CIDRBlocks: []string{vv},
			}
		}
	}

	if v, ok := in["cni_plugin"]; ok {
		if vv, ok := v.([]interface{}); ok {
			obj.CniPlugin = expandCniPlugin(vv)
		}
	}

	if v, ok := in["ip_family"]; ok && include("ip_family") {
		if vv, ok := v.(string); ok && vv != "" {
			if obj.ClusterNetwork == nil {
				obj.ClusterNetwork = &models.ClusterNetworkingConfig{}
			}
			obj.ClusterNetwork.IPFamily = models.IPFamily(vv)
		}
	}

	if v, ok := in["cloud"]; ok && include("cloud") {
		if vv, ok := v.([]interface{}); ok {
			obj.Cloud = expandClusterCloudSpec(vv, dcName, func(k string) bool { return include("cloud.0." + k) })
		}
	}

	// FIXME once we have proper server side validation for spec.BillingTenant we can remove this
	// for now copy it from cloud spec
	if obj.Cloud != nil && obj.Cloud.Aws != nil {
		obj.BillingTenant = obj.Cloud.Aws.OpenstackBillingTenant
	}

	if v, ok := in["syseleven_auth"]; ok && include("syseleven_auth") {
		if vv, ok := v.([]interface{}); ok {
			obj.Sys11auth = expandClusterSys11Auth(vv)
		}
	}

	return obj
}

func expandUpdateWindow(p []interface{}) *models.UpdateWindow {
	if len(p) < 1 {
		return nil
	}

	m := p[0].(map[string]interface{})
	ret := new(models.UpdateWindow)
	if v, ok := m["start"]; ok {
		ret.Start = v.(string)
	}
	if v, ok := m["length"]; ok {
		ret.Length = v.(string)
	}
	return ret
}

func expandAuditLogging(enabled bool) *models.AuditLoggingSettings {
	return &models.AuditLoggingSettings{
		Enabled: enabled,
	}
}

func expandCniPlugin(p []interface{}) *models.CNIPluginSettings {
	if len(p) < 1 {
		return nil
	}
	obj := &models.CNIPluginSettings{}
	if p[0] == nil {
		return obj
	}
	in := p[0].(map[string]interface{})

	if v, ok := in["type"]; ok {
		if vv, ok := v.(string); ok && vv != "" {
			obj.Type = models.CNIPluginType(vv)
		}
	}

	return obj
}

func expandClusterCloudSpec(p []interface{}, dcName string, include func(string) bool) *models.CloudSpec {
	if len(p) < 1 {
		return nil
	}
	obj := &models.CloudSpec{}
	if p[0] == nil {
		return obj
	}
	in := p[0].(map[string]interface{})

	obj.DatacenterName = dcName

	if v, ok := in["aws"]; ok && include("aws") {
		if vv, ok := v.([]interface{}); ok {
			obj.Aws = expandAWSCloudSpec(vv, func(k string) bool { return include("aws.0." + k) })
		}
	}

	if v, ok := in["openstack"]; ok && include("openstack") {
		if vv, ok := v.([]interface{}); ok {
			obj.Openstack = expandOpenstackCloudSpec(vv, func(k string) bool { return include("openstack.0." + k) })
		}
	}

	if v, ok := in["azure"]; ok && include("azure") {
		if vv, ok := v.([]interface{}); ok {
			obj.Azure = expandAzureCloudSpec(vv, func(k string) bool { return include("azure.0." + k) })
		}
	}

	return obj
}

func expandClusterSys11Auth(p []interface{}) *models.Sys11AuthSettings {
	if len(p) < 1 {
		return nil
	}
	if p[0] == nil {
		return nil
	}
	in := p[0].(map[string]interface{})
	if v := in["realm"].(string); v != "" {
		return &models.Sys11AuthSettings{Realm: v}
	}
	return nil
}

func expandAWSCloudSpec(p []interface{}, include func(string) bool) *models.AWSCloudSpec {
	if len(p) < 1 {
		return nil
	}
	obj := &models.AWSCloudSpec{}
	if p[0] == nil {
		return obj
	}
	in := p[0].(map[string]interface{})

	if v, ok := in["access_key_id"]; ok && include("access_key_id") {
		if vv, ok := v.(string); ok && vv != "" {
			obj.AccessKeyID = vv
		}
	}

	if v, ok := in["secret_access_key"]; ok && include("secret_access_key") {
		if vv, ok := v.(string); ok && vv != "" {
			obj.SecretAccessKey = vv
		}
	}

	if v, ok := in["vpc_id"]; ok && include("vpc_id") {
		if vv, ok := v.(string); ok && vv != "" {
			obj.VPCID = vv
		}
	}

	if v, ok := in["security_group_id"]; ok && include("security_group_id") {
		if vv, ok := v.(string); ok && vv != "" {
			obj.SecurityGroupID = vv
		}
	}

	if v, ok := in["instance_profile_name"]; ok && include("instance_profile_name") {
		if vv, ok := v.(string); ok && vv != "" {
			obj.InstanceProfileName = vv
		}
	}

	if v, ok := in["role_arn"]; ok && include("role_arn") {
		if vv, ok := v.(string); ok && vv != "" {
			obj.ControlPlaneRoleARN = vv
		}
	}

	if v, ok := in["openstack_billing_tenant"]; ok && include("openstack_billing_tenant") {
		if vv, ok := v.(string); ok && vv != "" {
			obj.OpenstackBillingTenant = vv
		}
	}

	if v, ok := in["route_table_id"]; ok && include("route_table_id") {
		if vv, ok := v.(string); ok && vv != "" {
			obj.RouteTableID = vv
		}
	}

	return obj
}

func expandOpenstackCloudSpec(p []interface{}, include func(string) bool) *models.OpenstackCloudSpec {
	if len(p) < 1 {
		return nil
	}

	obj := &models.OpenstackCloudSpec{}
	if p[0] == nil {
		return obj
	}
	in := p[0].(map[string]interface{})

	if v, ok := in["floating_ip_pool"]; ok && include("floating_ip_pool") {
		if vv, ok := v.(string); ok && vv != "" {
			obj.FloatingIPPool = vv
		}
	}

	if v, ok := in["security_group"]; ok && include("security_group") {
		if vv, ok := v.(string); ok && vv != "" {
			obj.SecurityGroups = vv
		}
	}

	if v, ok := in["network"]; ok && include("network") {
		if vv, ok := v.(string); ok && vv != "" {
			obj.Network = vv
		}
	}

	if v, ok := in["subnet_id"]; ok && include("subnet_id") {
		if vv, ok := v.(string); ok && vv != "" {
			obj.SubnetID = vv
		}
	}

	if v, ok := in["subnet_cidr"]; ok && include("subnet_cidr") {
		if vv, ok := v.(string); ok && vv != "" {
			obj.SubnetCIDR = vv
		}
	}

	if v, ok := in["server_group_id"]; ok && include("server_group_id") {
		if vv, ok := v.(string); ok && vv != "" {
			obj.ServerGroupID = vv
		}
	}

	if v, ok := in["application_credentials"]; ok {
		if vv, ok := v.([]interface{}); ok && len(vv) > 0 && vv[0] != nil {
			if m, ok := vv[0].(map[string]interface{}); ok {
				if v, ok := m["id"]; ok && include("application_credentials.0.id") {
					if vv, ok := v.(string); ok && vv != "" {
						obj.ApplicationCredentialID = vv
					}
				}

				if v, ok := m["secret"]; ok && include("application_credentials.0.secret") {
					if vv, ok := v.(string); ok && vv != "" {
						obj.ApplicationCredentialSecret = vv
					}
				}
			}
		}
	}

	if v, ok := in["user_credentials"]; ok {
		if vv, ok := v.([]interface{}); ok && len(vv) > 0 && vv[0] != nil {
			if m, ok := vv[0].(map[string]interface{}); ok {
				if v, ok := m["username"]; ok {
					if vv, ok := v.(string); ok && vv != "" {
						obj.Username = vv
					}
				}
				if v, ok := m["password"]; ok {
					if vv, ok := v.(string); ok && vv != "" {
						obj.Password = vv
					}
				}
				if v, ok := m["project_id"]; ok {
					if vv, ok := v.(string); ok && vv != "" {
						obj.ProjectID = vv
					}
				}

				if v, ok := m["project_name"]; ok {
					if vv, ok := v.(string); ok && vv != "" {
						obj.Project = vv
					}
				}

			}
		}
	}

	// HACK(furkhat): API doesn't return domain for cluster. Use 'Default' all the time.
	obj.Domain = "Default"

	return obj
}

func expandAzureCloudSpec(p []interface{}, include func(string) bool) *models.AzureCloudSpec {
	if len(p) < 1 {
		return nil
	}

	obj := &models.AzureCloudSpec{}

	if p[0] == nil {
		return obj
	}

	in := p[0].(map[string]interface{})

	if v, ok := in["availability_set"]; ok && include("availability_set") {
		if vv, ok := v.(string); ok && vv != "" {
			obj.AvailabilitySet = vv
		}
	}

	if v, ok := in["client_id"]; ok && include("client_id") {
		if vv, ok := v.(string); ok && vv != "" {
			obj.ClientID = vv
		}
	}

	if v, ok := in["client_secret"]; ok && include("client_secret") {
		if vv, ok := v.(string); ok && vv != "" {
			obj.ClientSecret = vv
		}
	}

	if v, ok := in["subscription_id"]; ok && include("subscription_id") {
		if vv, ok := v.(string); ok && vv != "" {
			obj.SubscriptionID = vv
		}
	}

	if v, ok := in["tenant_id"]; ok && include("tenant_id") {
		if vv, ok := v.(string); ok && vv != "" {
			obj.TenantID = vv
		}
	}

	if v, ok := in["resource_group"]; ok && include("resource_group") {
		if vv, ok := v.(string); ok && vv != "" {
			obj.ResourceGroup = vv
		}
	}

	if v, ok := in["route_table"]; ok && include("route_table") {
		if vv, ok := v.(string); ok && vv != "" {
			obj.RouteTableName = vv
		}
	}

	if v, ok := in["openstack_billing_tenant"]; ok && include("openstack_billing_tenant") {
		if vv, ok := v.(string); ok && vv != "" {
			obj.OpenstackBillingTenant = vv
		}
	}

	if v, ok := in["security_group"]; ok && include("security_group") {
		if vv, ok := v.(string); ok && vv != "" {
			obj.SecurityGroup = vv
		}
	}

	if v, ok := in["subnet"]; ok && include("subnet") {
		if vv, ok := v.(string); ok && vv != "" {
			obj.SubnetName = vv
		}
	}

	if v, ok := in["vnet"]; ok && include("vnet") {
		if vv, ok := v.(string); ok && vv != "" {
			obj.VNetName = vv
		}
	}

	return obj
}
