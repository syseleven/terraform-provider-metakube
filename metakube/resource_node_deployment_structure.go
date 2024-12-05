package metakube

import (
	"github.com/syseleven/go-metakube/models"
	"k8s.io/utils/ptr"
)

// flatteners
func metakubeNodeDeploymentFlattenSpec(in *models.NodeDeploymentSpec) []interface{} {
	if in == nil {
		return []interface{}{}
	}

	att := make(map[string]interface{})

	if in.Replicas != nil {
		att["replicas"] = *in.Replicas
	}

	if in.MinReplicas != nil {
		att["min_replicas"] = *in.MinReplicas
	}

	if in.MaxReplicas != nil {
		att["max_replicas"] = *in.MaxReplicas
	}

	if in.Template != nil {
		att["template"] = metakubeNodeDeploymentFlattenNodeSpec(in.Template)
	}

	return []interface{}{att}
}

func metakubeNodeDeploymentFlattenNodeSpec(in *models.NodeSpec) []interface{} {
	if in == nil {
		return []interface{}{}
	}

	att := make(map[string]interface{})

	if l := len(in.Labels); l > 0 {
		labels := make(map[string]string, l)
		for key, val := range in.Labels {
			labels[key] = val
		}
		att["labels"] = labels
	}

	if in.OperatingSystem != nil {
		att["operating_system"] = metakubeNodeDeploymentFlattenOperatingSystem(in.OperatingSystem)
	}

	if in.Versions != nil && in.Versions.Kubelet != "" {
		att["versions"] = metakubeNodeDeploymentFlattenVersion(in.Versions)
	}

	if l := len(in.Taints); l > 0 {
		t := make([]interface{}, l)
		for i, v := range in.Taints {
			t[i] = metakubeNodeDeploymentFlattenTaintSpec(v)
		}
		att["taints"] = t
	}

	if in.Cloud != nil {
		att["cloud"] = metakubeNodeDeploymentFlattenCloudSpec(in.Cloud)
	}

	return []interface{}{att}
}

func metakubeNodeDeploymentFlattenOperatingSystem(in *models.OperatingSystemSpec) []interface{} {
	if in == nil {
		return []interface{}{}
	}

	att := make(map[string]interface{})

	if in.Ubuntu != nil {
		att["ubuntu"] = metakubeNodeDeploymentFlattenUbuntu(in.Ubuntu)
	}

	if in.Flatcar != nil {
		att["flatcar"] = metakubeNodeDeploymentFlattenFlatcar(in.Flatcar)
	}

	return []interface{}{att}
}

func metakubeNodeDeploymentFlattenUbuntu(in *models.UbuntuSpec) []interface{} {
	if in == nil {
		return []interface{}{}
	}

	att := make(map[string]interface{})

	att["dist_upgrade_on_boot"] = in.DistUpgradeOnBoot

	return []interface{}{att}
}

func metakubeNodeDeploymentFlattenFlatcar(in *models.FlatcarSpec) []interface{} {
	if in == nil {
		return []interface{}{}
	}

	att := make(map[string]interface{})

	att["disable_auto_update"] = in.DisableAutoUpdate

	return []interface{}{att}
}

func metakubeNodeDeploymentFlattenVersion(in *models.NodeVersionInfo) []interface{} {
	if in == nil {
		return []interface{}{}
	}

	att := make(map[string]interface{})

	if in.Kubelet != "" {
		att["kubelet"] = in.Kubelet
	}

	return []interface{}{att}
}

func metakubeNodeDeploymentFlattenTaintSpec(in *models.TaintSpec) map[string]interface{} {
	if in == nil {
		return map[string]interface{}{}
	}

	att := make(map[string]interface{})

	if in.Key != "" {
		att["key"] = in.Key
	}

	if in.Value != "" {
		att["value"] = in.Value
	}

	if in.Effect != "" {
		att["effect"] = in.Effect
	}

	return att
}

func metakubeNodeDeploymentFlattenCloudSpec(in *models.NodeCloudSpec) []interface{} {
	if in == nil {
		return []interface{}{}
	}

	att := make(map[string]interface{})

	if in.Aws != nil {
		att["aws"] = metakubeNodeDeploymentFlattenAWSSpec(in.Aws)
	}

	if in.Openstack != nil {
		att["openstack"] = metakubeNodeDeploymentFlattenOpenstackSpec(in.Openstack)
	}

	if in.Azure != nil {
		att["azure"] = metakubeNodeDeploymentFlattenAzureSpec(in.Azure)
	}

	return []interface{}{att}
}

func metakubeNodeDeploymentFlattenAWSSpec(in *models.AWSNodeSpec) []interface{} {
	if in == nil {
		return []interface{}{}
	}

	att := make(map[string]interface{})

	att["assign_public_ip"] = in.AssignPublicIP

	if l := len(in.Tags); l > 0 {
		t := make(map[string]string, l)
		for key, val := range in.Tags {
			t[key] = val
		}
		att["tags"] = t
	}

	if in.AMI != "" {
		att["ami"] = in.AMI
	}

	if in.AvailabilityZone != "" {
		att["availability_zone"] = in.AvailabilityZone
	}

	if in.SubnetID != "" {
		att["subnet_id"] = in.SubnetID
	}

	if in.VolumeType != nil {
		att["volume_type"] = *in.VolumeType
	}

	if in.VolumeSize != nil {
		att["disk_size"] = *in.VolumeSize
	}

	if in.InstanceType != nil {
		att["instance_type"] = *in.InstanceType
	}

	return []interface{}{att}
}

func metakubeNodeDeploymentFlattenOpenstackSpec(in *models.OpenstackNodeSpec) []interface{} {
	if in == nil {
		return []interface{}{}
	}

	att := make(map[string]interface{})

	if in.Flavor != nil {
		att["flavor"] = *in.Flavor
	}

	if in.Image != nil {
		att["image"] = *in.Image
	}

	att["use_floating_ip"] = in.UseFloatingIP

	if in.InstanceReadyCheckPeriod != "" {
		att["instance_ready_check_period"] = in.InstanceReadyCheckPeriod
	}

	if in.InstanceReadyCheckTimeout != "" {
		att["instance_ready_check_timeout"] = in.InstanceReadyCheckTimeout
	}

	if in.Tags != nil {
		att["tags"] = in.Tags
	}

	if in.RootDiskSizeGB != nil && *in.RootDiskSizeGB != 0 {
		att["disk_size"] = *in.RootDiskSizeGB
	}

	if in.ServerGroupID != "" {
		att["server_group_id"] = in.ServerGroupID
	}

	return []interface{}{att}
}

func metakubeNodeDeploymentFlattenAzureSpec(in *models.AzureNodeSpec) []interface{} {
	if in == nil {
		return []interface{}{}
	}

	att := make(map[string]interface{})

	if in.ImageID != "" {
		att["image_id"] = in.ImageID
	}

	if in.Size != nil {
		att["size"] = *in.Size
	}

	att["assign_public_ip"] = in.AssignPublicIP

	att["disk_size_gb"] = in.DataDiskSize

	att["os_disk_size_gb"] = in.OSDiskSize

	if in.Tags != nil {
		att["tags"] = in.Tags
	}

	if in.Zones != nil {
		att["zones"] = in.Zones
	}

	return []interface{}{att}
}

// expanders

func metakubeNodeDeploymentExpandSpec(p []interface{}, isCreate bool) *models.NodeDeploymentSpec {
	if len(p) < 1 {
		return nil
	}
	obj := &models.NodeDeploymentSpec{}
	if p[0] == nil {
		return obj
	}

	in, ok := p[0].(map[string]interface{})
	if !ok {
		return obj
	}

	if v, ok := in["min_replicas"]; ok {
		if vv, ok := v.(int); ok {
			obj.MinReplicas = ptr.To(int32(vv))
			if isCreate {
				obj.Replicas = int32ToPtr(*obj.MinReplicas)
			}
		}
	}

	if v, ok := in["max_replicas"]; ok {
		if vv, ok := v.(int); ok {
			obj.MaxReplicas = ptr.To(int32(vv))
		}
	}

	if v, ok := in["replicas"]; ok && *obj.MinReplicas == 0 {
		if vv, ok := v.(int); ok {
			obj.Replicas = int32ToPtr(int32(vv))
		}
	}

	if v, ok := in["template"]; ok {
		if vv, ok := v.([]interface{}); ok {
			obj.Template = metakubeNodeDeploymentExpandNodeSpec(vv)
		}
	}

	return obj
}

func metakubeNodeDeploymentExpandNodeSpec(p []interface{}) *models.NodeSpec {
	if len(p) < 1 {
		return nil
	}
	obj := &models.NodeSpec{}
	if p[0] == nil {
		return obj
	}

	in, ok := p[0].(map[string]interface{})
	if !ok {
		return obj
	}

	if v, ok := in["labels"]; ok {
		obj.Labels = make(map[string]string)
		if vv, ok := v.(map[string]interface{}); ok {
			for key, val := range vv {
				if s, ok := val.(string); ok && s != "" {
					obj.Labels[key] = s
				}
			}
		}
	}

	if v, ok := in["operating_system"]; ok {
		if vv, ok := v.([]interface{}); ok {
			obj.OperatingSystem = metakubeNodeDeploymentExpandOS(vv)
		}
	}

	if v, ok := in["versions"]; ok {
		if vv, ok := v.([]interface{}); ok {
			obj.Versions = metakubeNodeDeploymentExpandVersion(vv)
		}
	}

	if v, ok := in["taints"]; ok {
		if vv, ok := v.([]interface{}); ok {
			for _, t := range vv {
				if tt, ok := t.(map[string]interface{}); ok {
					obj.Taints = append(obj.Taints, metakubeNodeDeploymentExpandTaintSpec(tt))
				}
			}
		}
	}

	if v, ok := in["cloud"]; ok {
		if vv, ok := v.([]interface{}); ok {
			obj.Cloud = metakubeNodeDeploymentExpandCloudSpec(vv)
		}
	}

	return obj
}

func metakubeNodeDeploymentExpandOS(p []interface{}) *models.OperatingSystemSpec {
	if len(p) < 1 {
		return nil
	}
	obj := &models.OperatingSystemSpec{}
	if p[0] == nil {
		return obj
	}

	in, ok := p[0].(map[string]interface{})
	if !ok {
		return obj
	}

	if v, ok := in["ubuntu"]; ok {
		if vv, ok := v.([]interface{}); ok {
			obj.Ubuntu = metakubeNodeDeploymentExpandUbuntu(vv)
		}
	}

	if v, ok := in["flatcar"]; ok {
		if vv, ok := v.([]interface{}); ok {
			obj.Flatcar = metakubeNodeDeploymentExpandFlatcar(vv)
		}
	}

	return obj
}

func metakubeNodeDeploymentExpandUbuntu(p []interface{}) *models.UbuntuSpec {
	if len(p) < 1 {
		return nil
	}
	obj := &models.UbuntuSpec{}
	if p[0] == nil {
		return obj
	}

	in, ok := p[0].(map[string]interface{})
	if !ok {
		return obj
	}

	if v, ok := in["dist_upgrade_on_boot"]; ok {
		if vv, ok := v.(bool); ok {
			obj.DistUpgradeOnBoot = vv
		}
	}

	return obj
}

func metakubeNodeDeploymentExpandFlatcar(p []interface{}) *models.FlatcarSpec {
	if len(p) < 1 {
		return nil
	}
	obj := &models.FlatcarSpec{}
	if p[0] == nil {
		return obj
	}

	in, ok := p[0].(map[string]interface{})
	if !ok {
		return obj
	}

	if v, ok := in["disable_auto_update"]; ok {
		if vv, ok := v.(bool); ok {
			obj.DisableAutoUpdate = vv
		}
	}

	return obj
}

func metakubeNodeDeploymentExpandVersion(p []interface{}) *models.NodeVersionInfo {
	if len(p) < 1 {
		return nil
	}
	if p[0] == nil {
		return nil
	}

	in, ok := p[0].(map[string]interface{})
	if !ok {
		return nil
	}

	v, ok := in["kubelet"]
	if !ok {
		return nil
	}

	if vv, ok := v.(string); ok && vv != "" {
		return &models.NodeVersionInfo{Kubelet: vv}
	}
	return nil
}

func metakubeNodeDeploymentExpandTaintSpec(in map[string]interface{}) *models.TaintSpec {
	obj := &models.TaintSpec{}

	if v, ok := in["key"]; ok {
		if vv, ok := v.(string); ok && vv != "" {
			obj.Key = vv
		}
	}

	if v, ok := in["value"]; ok {
		if vv, ok := v.(string); ok && vv != "" {
			obj.Value = vv
		}
	}

	if v, ok := in["effect"]; ok {
		if vv, ok := v.(string); ok && vv != "" {
			obj.Effect = vv
		}
	}

	return obj
}

func metakubeNodeDeploymentExpandCloudSpec(p []interface{}) *models.NodeCloudSpec {
	if len(p) < 1 {
		return nil
	}
	obj := &models.NodeCloudSpec{}
	if p[0] == nil {
		return obj
	}

	in, ok := p[0].(map[string]interface{})
	if !ok {
		return obj
	}

	if v, ok := in["aws"]; ok {
		if vv, ok := v.([]interface{}); ok {
			obj.Aws = metakubeNodeDeploymentExpandAWSSpec(vv)
		}
	}

	if v, ok := in["openstack"]; ok {
		if vv, ok := v.([]interface{}); ok {
			obj.Openstack = metakubeNodeDeploymentExpandOpenstackSpec(vv)
		}
	}

	if v, ok := in["azure"]; ok {
		if vv, ok := v.([]interface{}); ok {
			obj.Azure = metakubeNodeDeploymentExpandAzureSpec(vv)
		}
	}

	return obj
}

func metakubeNodeDeploymentExpandAWSSpec(p []interface{}) *models.AWSNodeSpec {
	if len(p) < 1 {
		return nil
	}
	obj := &models.AWSNodeSpec{}
	if p[0] == nil {
		return obj
	}

	in, ok := p[0].(map[string]interface{})
	if !ok {
		return obj
	}

	if v, ok := in["instance_type"]; ok {
		if vv, ok := v.(string); ok && vv != "" {
			obj.InstanceType = strToPtr(vv)
		}
	}

	if v, ok := in["disk_size"]; ok {
		if vv, ok := v.(int); ok {
			obj.VolumeSize = intToInt32Ptr(vv)
		}
	}

	if v, ok := in["volume_type"]; ok {
		if vv, ok := v.(string); ok && vv != "" {
			obj.VolumeType = strToPtr(vv)
		}
	}

	if v, ok := in["availability_zone"]; ok {
		if vv, ok := v.(string); ok && vv != "" {
			obj.AvailabilityZone = vv
		}
	}

	if v, ok := in["subnet_id"]; ok {
		if vv, ok := v.(string); ok && vv != "" {
			obj.SubnetID = vv
		}
	}

	if v, ok := in["assign_public_ip"]; ok {
		if vv, ok := v.(bool); ok {
			obj.AssignPublicIP = vv
		}
	}

	if v, ok := in["ami"]; ok {
		if vv, ok := v.(string); ok && vv != "" {
			obj.AMI = vv
		}
	}

	if v, ok := in["tags"]; ok {
		obj.Tags = make(map[string]string)
		if vv, ok := v.(map[string]interface{}); ok {
			for key, val := range vv {
				if s, ok := val.(string); ok && s != "" {
					obj.Tags[key] = s
				}
			}
		}
	}

	return obj
}

func metakubeNodeDeploymentExpandOpenstackSpec(p []interface{}) *models.OpenstackNodeSpec {
	if len(p) < 1 {
		return nil
	}
	obj := &models.OpenstackNodeSpec{}
	if p[0] == nil {
		return obj
	}

	in, ok := p[0].(map[string]interface{})
	if !ok {
		return obj
	}

	if v, ok := in["flavor"]; ok {
		if vv, ok := v.(string); ok && vv != "" {
			obj.Flavor = strToPtr(vv)
		}
	}

	if v, ok := in["image"]; ok {
		if vv, ok := v.(string); ok && vv != "" {
			obj.Image = strToPtr(vv)
		}
	}

	if v, ok := in["use_floating_ip"]; ok {
		if vv, ok := v.(bool); ok {
			obj.UseFloatingIP = ptr.To(vv)
		}
	}

	if v, ok := in["instance_ready_check_period"]; ok {
		if vv, ok := v.(string); ok {
			obj.InstanceReadyCheckPeriod = vv
		}
	}

	if v, ok := in["instance_ready_check_timeout"]; ok {
		if vv, ok := v.(string); ok {
			obj.InstanceReadyCheckTimeout = vv
		}
	}

	if v, ok := in["tags"]; ok {
		obj.Tags = make(map[string]string)
		for key, val := range v.(map[string]interface{}) {
			if s, ok := val.(string); ok && s != "" {
				obj.Tags[key] = s
			}
		}
	}

	if v, ok := in["disk_size"]; ok {
		if vv, ok := v.(int); ok {
			obj.RootDiskSizeGB = ptr.To(int64(vv))
		}
	}

	if v, ok := in["server_group_id"]; ok {
		if vv, ok := v.(string); ok {
			obj.ServerGroupID = vv
		}
	}

	return obj
}

func metakubeNodeDeploymentExpandAzureSpec(p []interface{}) *models.AzureNodeSpec {
	if len(p) < 1 {
		return nil
	}

	obj := &models.AzureNodeSpec{}

	if p[0] == nil {
		return obj
	}

	in, ok := p[0].(map[string]interface{})
	if !ok {
		return obj
	}

	if v, ok := in["image_id"]; ok {
		if vv, ok := v.(string); ok && vv != "" {
			obj.ImageID = vv
		}
	}

	if v, ok := in["size"]; ok {
		if vv, ok := v.(string); ok && vv != "" {
			obj.Size = strToPtr(vv)
		}
	}

	if v, ok := in["assign_public_ip"]; ok {
		if vv, ok := v.(bool); ok {
			obj.AssignPublicIP = vv
		}
	}

	if v, ok := in["disk_size_gb"]; ok {
		if vv, ok := v.(int32); ok {
			obj.DataDiskSize = vv
		}
	}

	if v, ok := in["os_disk_size_gb"]; ok {
		if vv, ok := v.(int32); ok {
			obj.OSDiskSize = vv
		}
	}

	if v, ok := in["tags"]; ok {
		if vv, ok := v.(map[string]string); ok {
			obj.Tags = vv
		}
	}

	if v, ok := in["zones"]; ok {
		if vv, ok := v.([]string); ok && len(vv) > 0 {
			obj.Zones = vv
		}
	}

	return obj
}
