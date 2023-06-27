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

	if in.FailedJobsHistoryLimit != 0 {
		att["failed_jobs_history_limit"] = in.FailedJobsHistoryLimit
	}

	if in.StartingDeadlineSeconds != 0 {
		att["starting_deadline_seconds"] = in.StartingDeadlineSeconds
	}

	if in.SuccessfulJobsHistoryLimit != 0 {
		att["successful_jobs_history_limit"] = in.SuccessfulJobsHistoryLimit
	}

	if in.Schedule != "" {
		att["schedule"] = in.Schedule
	}

	if in.MaintenanceJobTemplate != nil {
		att["maintenance_job_template"] = metakubeMaintenanceCronJobFlattenMaintenanceJobTemplateSpec(in.MaintenanceJobTemplate)
	}

	return []interface{}{att}
}

func metakubeMaintenanceCronJobFlattenMaintenanceJobTemplateSpec(in *models.MaintenanceJobTemplateSpec) []interface{} {
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

	if in.Name != "" {
		att["name"] = in.Name
	}

	if in.Spec != nil {
		att["spec"] = metakubeMaintenanceCronJobFlattenMaintenanceJobSpec(in.Spec)
	}

	return []interface{}{att}
}
func metakubeMaintenanceCronJobFlattenMaintenanceJobSpec(in *models.MaintenanceJobSpec) []interface{} {
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

	if v, ok := in["failed_jobs_history_limit"]; ok {
		if vv, ok := v.(int); ok {
			obj.FailedJobsHistoryLimit = int32(vv)
		}
	}

	if v, ok := in["starting_deadline_seconds"]; ok {
		if vv, ok := v.(int); ok {
			obj.StartingDeadlineSeconds = int64(vv)
		}
	}

	if v, ok := in["successful_jobs_history_limit"]; ok {
		if vv, ok := v.(int); ok {
			obj.SuccessfulJobsHistoryLimit = int32(vv)
		}
	}

	if v, ok := in["schedule"]; ok {
		if vv, ok := v.(string); ok {
			obj.Schedule = vv
		}
	}

	if v, ok := in["maintenance_job_template"]; ok {
		if vv, ok := v.([]interface{}); ok {
			obj.MaintenanceJobTemplate = metakubeMaintenanceCronJobExpandMaintenanceJobTemplateSpec(vv)
		}
	}

	return obj
}

func metakubeMaintenanceCronJobExpandMaintenanceJobTemplateSpec(p []interface{}) *models.MaintenanceJobTemplateSpec {
	if len(p) < 1 {
		return nil
	}
	obj := &models.MaintenanceJobTemplateSpec{}
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

	if v, ok := in["name"]; ok {
		if vv, ok := v.(string); ok {
			obj.Name = vv
		}
	}

	if v, ok := in["spec"]; ok {
		if vv, ok := v.([]interface{}); ok {
			obj.Spec = metakubeMaintenanceCronJobExpandMaintenanceJobSpec(vv)
		}
	}

	return obj
}

func metakubeMaintenanceCronJobExpandMaintenanceJobSpec(p []interface{}) *models.MaintenanceJobSpec {
	if len(p) < 1 {
		return nil
	}
	obj := &models.MaintenanceJobSpec{}
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
