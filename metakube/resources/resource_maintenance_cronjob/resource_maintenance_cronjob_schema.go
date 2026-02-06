package resource_maintenance_cronjob

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func MaintenanceCronJobSchema(ctx context.Context) schema.Schema {
	blocks := maintenanceCronJobBlocks()
	blocks["timeouts"] = timeouts.Block(ctx, timeouts.Opts{
		Create: true,
		Update: true,
		Delete: true,
	})

	return schema.Schema{
		Attributes: maintenanceCronJobAttributes(),
		Blocks:     blocks,
	}
}

// MaintenanceCronJobModel represents the Terraform resource model for a maintenance cron job.
type MaintenanceCronJobModel struct {
	ID                types.String   `tfsdk:"id"`
	ProjectID         types.String   `tfsdk:"project_id"`
	ClusterID         types.String   `tfsdk:"cluster_id"`
	Name              types.String   `tfsdk:"name"`
	Spec              types.List     `tfsdk:"spec"`
	CreationTimestamp types.String   `tfsdk:"creation_timestamp"`
	DeletionTimestamp types.String   `tfsdk:"deletion_timestamp"`
	Timeouts          timeouts.Value `tfsdk:"timeouts"`
}

type SpecModel struct {
	Schedule               types.String `tfsdk:"schedule"`
	MaintenanceJobTemplate types.List   `tfsdk:"maintenance_job_template"`
}

type MaintenanceJobTemplateModel struct {
	Options  types.List   `tfsdk:"options"`
	Rollback types.Bool   `tfsdk:"rollback"`
	Type     types.String `tfsdk:"type"`
}

type OptionsBlockModel struct {
	Options types.Map `tfsdk:"options"`
}

func specAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"schedule": types.StringType,
		"maintenance_job_template": types.ListType{
			ElemType: types.ObjectType{
				AttrTypes: maintenanceJobTemplateAttrTypes(),
			},
		},
	}
}

func maintenanceJobTemplateAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"options": types.ListType{
			ElemType: types.ObjectType{
				AttrTypes: optionsBlockAttrTypes(),
			},
		},
		"rollback": types.BoolType,
		"type":     types.StringType,
	}
}

func optionsBlockAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"options": types.MapType{
			ElemType: types.StringType,
		},
	}
}

func maintenanceCronJobAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:    true,
			Description: "The id of the maintenance cron job resource",
		},
		"project_id": schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Reference project identifier",
		},
		"cluster_id": schema.StringAttribute{
			Required:    true,
			Description: "Cluster that maintenance cron job belongs to",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"name": schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Maintenance cron job name",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"creation_timestamp": schema.StringAttribute{
			Computed:    true,
			Description: "Creation timestamp",
		},
		"deletion_timestamp": schema.StringAttribute{
			Computed:    true,
			Description: "Deletion timestamp",
		},
	}
}

func maintenanceCronJobBlocks() map[string]schema.Block {
	return map[string]schema.Block{
		"spec": schema.ListNestedBlock{
			Validators: []validator.List{
				listvalidator.SizeAtLeast(1),
				listvalidator.SizeAtMost(1),
			},
			Description: "Maintenance cron job specification",
			NestedObject: schema.NestedBlockObject{
				Attributes: map[string]schema.Attribute{
					"schedule": schema.StringAttribute{
						Required:    true,
						Description: "A schedule in cron format",
					},
				},
				Blocks: map[string]schema.Block{
					"maintenance_job_template": schema.ListNestedBlock{
						Validators: []validator.List{
							listvalidator.SizeAtLeast(1),
							listvalidator.SizeAtMost(1),
						},
						Description: "MaintenanceJob template specification",
						NestedObject: schema.NestedBlockObject{
							Attributes: map[string]schema.Attribute{
								"rollback": schema.BoolAttribute{
									Optional:    true,
									Computed:    true,
									Default:     booldefault.StaticBool(false),
									Description: "Indicates whether the maintenance done should be rolled back",
								},
								"type": schema.StringAttribute{
									Required:    true,
									Description: "Defines the type of maintenance that should be run",
								},
							},
							Blocks: map[string]schema.Block{
								"options": schema.ListNestedBlock{
									Validators: []validator.List{
										listvalidator.SizeAtMost(1),
									},
									Description: "Options for the maintenance type",
									NestedObject: schema.NestedBlockObject{
										Attributes: map[string]schema.Attribute{
											"options": schema.MapAttribute{
												Optional:    true,
												ElementType: types.StringType,
												Description: "Map of string keys and values that can be used to set certain options for the given maintenance type.",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
