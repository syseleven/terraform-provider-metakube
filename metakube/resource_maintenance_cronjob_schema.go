package metakube

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func metakubeResourceMaintenanceCronJobSpecFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
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
				Schema: metakubeMaintenanceJobTemplateSchema(),
			},
		},
	}
}

func metakubeMaintenanceJobTemplateSchema() map[string]*schema.Schema {
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
