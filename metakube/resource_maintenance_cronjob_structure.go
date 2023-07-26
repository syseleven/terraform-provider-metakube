package metakube

import (
	"github.com/syseleven/go-metakube/models"
)

// flatteners

func metakubeMaintenanceCronJobFlattenSpec(in *models.MaintenanceCronJobSpec) []interface{} {
	if in == nil {
		return []interface{}{}
	}

	att := make(map[string]interface{})

	if in.Schedule != "" {
		att["schedule"] = in.Schedule
	}

	if in.MaintenanceJobTemplate != nil {
		att["maintenance_job_template"] = metakubeMaintenanceCronJobFlattenMaintenanceJobTemplate(in.MaintenanceJobTemplate)
	}

	return []interface{}{att}
}

func metakubeMaintenanceCronJobFlattenMaintenanceJobTemplate(in *models.MaintenanceJobTemplate) []interface{} {
	if in == nil {
		return []interface{}{}
	}

	att := make(map[string]interface{})

	if l := len(in.Options); l > 0 {
		options := make(map[string]string, l)
		for key, val := range in.Options {
			options[key] = val
		}
		att["options"] = options
	}

	att["rollback"] = in.Rollback

	if in.Type != "" {
		att["type"] = in.Type
	}

	return []interface{}{att}
}

// expanders

func metakubeMaintenanceCronJobExpandSpec(p []interface{}) *models.MaintenanceCronJobSpec {
	if len(p) < 1 {
		return nil
	}
	obj := &models.MaintenanceCronJobSpec{}
	if p[0] == nil {
		return obj
	}

	in, ok := p[0].(map[string]interface{})
	if !ok {
		return obj
	}

	if v, ok := in["schedule"]; ok {
		if vv, ok := v.(string); ok {
			obj.Schedule = vv
		}
	}

	if v, ok := in["maintenance_job_template"]; ok {
		if vv, ok := v.([]interface{}); ok {
			obj.MaintenanceJobTemplate = metakubeMaintenanceCronJobExpandMaintenanceJobTemplate(vv)
		}
	}

	return obj
}

func metakubeMaintenanceCronJobExpandMaintenanceJobTemplate(p []interface{}) *models.MaintenanceJobTemplate {
	if len(p) < 1 {
		return nil
	}
	obj := &models.MaintenanceJobTemplate{}
	if p[0] == nil {
		return obj
	}

	in, ok := p[0].(map[string]interface{})
	if !ok {
		return obj
	}

	if v, ok := in["options"]; ok {
		obj.Options = make(map[string]string)
		if vv, ok := v.(map[string]interface{}); ok {
			for key, val := range vv {
				if s, ok := val.(string); ok && s != "" {
					obj.Options[key] = s
				}
			}
		}
	}

	if v, ok := in["rollback"]; ok {
		if vv, ok := v.(bool); ok {
			obj.Rollback = vv
		}
	}

	if v, ok := in["type"]; ok {
		if vv, ok := v.(string); ok {
			obj.Type = vv
		}
	}

	return obj
}
