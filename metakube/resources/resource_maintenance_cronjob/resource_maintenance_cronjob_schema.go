package resource_maintenance_cronjob

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func MaintenanceCronJobSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Version:    1,
		Attributes: maintenanceCronJobAttributes(),
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Create: true,
				Update: true,
				Delete: true,
			}),
		},
	}
}

// MaintenanceCronJobModel represents the Terraform resource model for a maintenance cron job.
type MaintenanceCronJobModel struct {
	ID                types.String   `tfsdk:"id"`
	ProjectID         types.String   `tfsdk:"project_id"`
	ClusterID         types.String   `tfsdk:"cluster_id"`
	Name              types.String   `tfsdk:"name"`
	Spec              types.Object   `tfsdk:"spec"`
	CreationTimestamp types.String   `tfsdk:"creation_timestamp"`
	DeletionTimestamp types.String   `tfsdk:"deletion_timestamp"`
	Timeouts          timeouts.Value `tfsdk:"timeouts"`
}

type SpecModel struct {
	Schedule               types.String `tfsdk:"schedule"`
	MaintenanceJobTemplate types.Object `tfsdk:"maintenance_job_template"`
}

type MaintenanceJobTemplateModel struct {
	Options  types.Object `tfsdk:"options"`
	Rollback types.Bool   `tfsdk:"rollback"`
	Type     types.String `tfsdk:"type"`
}

type OptionsModel struct {
	Options types.Map `tfsdk:"options"`
}

func specAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"schedule":                 types.StringType,
		"maintenance_job_template": types.ObjectType{AttrTypes: maintenanceJobTemplateAttrTypes()},
	}
}

func maintenanceJobTemplateAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"options":  types.ObjectType{AttrTypes: optionsAttrTypes()},
		"rollback": types.BoolType,
		"type":     types.StringType,
	}
}

func optionsAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"options": types.MapType{
			ElemType: types.StringType,
		},
	}
}

type rollbackUseAPIValue struct{}

func (m rollbackUseAPIValue) Description(_ context.Context) string {
	return "Uses the prior state value of rollback (which comes from the API) for the plan during updates"
}

func (m rollbackUseAPIValue) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m rollbackUseAPIValue) PlanModifyBool(_ context.Context, req planmodifier.BoolRequest, resp *planmodifier.BoolResponse) {
	if !req.ConfigValue.IsNull() && !req.ConfigValue.IsUnknown() {
		resp.PlanValue = req.StateValue
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid rollback value",
			"Rollback is controlled by the Metakube API and cannot be configured.",
		)
		return
	}

	if req.Plan.Raw.IsNull() {
		return
	}
	// Apply the creation default here so it cannot overwrite an API-controlled
	// true value before the framework decides whether an update is needed.
	if req.State.Raw.IsNull() {
		resp.PlanValue = types.BoolValue(false)
		return
	}
	resp.PlanValue = req.StateValue
}

func maintenanceCronJobAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:    true,
			Description: "The id of the maintenance cron job resource",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"project_id": schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Reference project identifier",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
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
		"spec": schema.SingleNestedAttribute{
			Required:    true,
			Description: "Maintenance cron job specification",
			Attributes: map[string]schema.Attribute{
				"schedule": schema.StringAttribute{
					Required:    true,
					Description: "A schedule in cron format",
				},
				"maintenance_job_template": schema.SingleNestedAttribute{
					Required:    true,
					Description: "MaintenanceJob template specification",
					Attributes: map[string]schema.Attribute{
						"rollback": schema.BoolAttribute{
							Optional:      true,
							Computed:      true,
							Description:   "Indicates whether the maintenance done should be rolled back",
							PlanModifiers: []planmodifier.Bool{rollbackUseAPIValue{}},
						},
						"type": schema.StringAttribute{
							Required:    true,
							Description: "Defines the type of maintenance that should be run",
						},
						"options": schema.SingleNestedAttribute{
							Optional:    true,
							Description: "Options for the maintenance type",
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
	}
}
