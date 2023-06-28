package metakube

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func metakubeResourceMaintenanceCronJobSpecFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"failed_jobs_history_limit": {
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
			Description: "Number of failed finished maintenance jobs to retain",
		},
		"starting_deadline_seconds": {
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
			Description: "An optional deadline in seconds",
		},
		"successful_jobs_history_limit": {
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
			Description: "Number of successful finished maintenance jobs to retain",
		},
		"schedule": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "A schedule in cron format",
		},
		"maintenance_job_template": {
			Type:        schema.TypeList,
			MaxItems:    1,
			Required:    true,
			Description: "MaintenanceJob template specification",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"labels": {
						Type:        schema.TypeMap,
						Optional:    true,
						Description: "Map of string keys and values that can be used to organize and categorize (scope and select) objects.",
						Elem: &schema.Schema{
							Type: schema.TypeString,
						},
					},
					"name": {
						Type:        schema.TypeString,
						Optional:    true,
						Computed:    true,
						ForceNew:    true,
						Description: "Maintenance job template name",
					},
					"spec": {
						Type:        schema.TypeList,
						Required:    true,
						MinItems:    1,
						MaxItems:    1,
						Description: "Maintenance job spec",
						Elem: &schema.Resource{
							Schema: metakubeMaintenanceJobSpecSchema(),
						},
					},
				},
			},
		},
	}
}

func metakubeMaintenanceJobSpecSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"options": {
			Type:        schema.TypeList,
			Optional:    true,
			MinItems:    1,
			MaxItems:    1,
			Description: "Ubuntu operating system",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"options": {
						Type:        schema.TypeMap,
						Optional:    true,
						Description: "Map of string keys and values that can be used to set certain options for the given maintenance type.",
						Elem: &schema.Schema{
							Type: schema.TypeString,
						},
					},
				},
			},
		},
		"rollback": {
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Indicates whether the maintenance done should be rolled back",
		},
		"type": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "Defines the type of maintenance that should be run",
		},
	}
}
