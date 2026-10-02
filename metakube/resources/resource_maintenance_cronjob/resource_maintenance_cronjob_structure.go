package resource_maintenance_cronjob

import (
	"context"
	"maps"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/syseleven/go-metakube/models"
	"github.com/syseleven/terraform-provider-metakube/metakube/common"
)

// flatteners

func metakubeMaintenanceCronJobFlattenSpec(ctx context.Context, model *MaintenanceCronJobModel, in *models.MaintenanceCronJobSpec) diag.Diagnostics {
	var diags diag.Diagnostics

	if in == nil {
		model.Spec = types.ObjectNull(specAttrTypes())
		return diags
	}

	specModel, d := common.ObjectAs[SpecModel](ctx, model.Spec)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}
	specModel.Schedule = types.StringValue(in.Schedule)

	diags.Append(metakubeMaintenanceCronJobFlattenMaintenanceJobTemplate(ctx, &specModel, in.MaintenanceJobTemplate)...)
	if diags.HasError() {
		return diags
	}

	specObj, d := types.ObjectValueFrom(ctx, specAttrTypes(), specModel)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}

	model.Spec = specObj

	return diags
}

func metakubeMaintenanceCronJobFlattenMaintenanceJobTemplate(ctx context.Context, specModel *SpecModel, in *models.MaintenanceJobTemplate) diag.Diagnostics {
	var diags diag.Diagnostics

	if in == nil {
		specModel.MaintenanceJobTemplate = types.ObjectNull(maintenanceJobTemplateAttrTypes())
		return diags
	}

	tmplModel, d := common.ObjectAs[MaintenanceJobTemplateModel](ctx, specModel.MaintenanceJobTemplate)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}
	tmplModel.Rollback = types.BoolValue(in.Rollback)
	tmplModel.Type = types.StringValue(in.Type)

	diags.Append(metakubeMaintenanceCronJobFlattenOptions(ctx, &tmplModel, in.Options)...)
	if diags.HasError() {
		return diags
	}

	tmplObj, d := types.ObjectValueFrom(ctx, maintenanceJobTemplateAttrTypes(), tmplModel)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}

	specModel.MaintenanceJobTemplate = tmplObj

	return diags
}

func metakubeMaintenanceCronJobFlattenOptions(ctx context.Context, tmplModel *MaintenanceJobTemplateModel, in map[string]string) diag.Diagnostics {
	var diags diag.Diagnostics

	if len(in) == 0 {
		if tmplModel.Options.IsNull() || tmplModel.Options.IsUnknown() {
			tmplModel.Options = types.ObjectNull(optionsAttrTypes())
			return diags
		}
		previous, d := common.ObjectAs[OptionsModel](ctx, tmplModel.Options)
		diags.Append(d...)
		if diags.HasError() {
			return diags
		}
		// The API represents omitted and explicitly empty options identically.
		// Keep the configured empty shape, but never retain removed map entries.
		if !previous.Options.IsUnknown() && len(previous.Options.Elements()) == 0 {
			return diags
		}
	}

	optionsMap := make(map[string]attr.Value, len(in))
	for k, v := range in {
		optionsMap[k] = types.StringValue(v)
	}

	mapVal, d := types.MapValue(types.StringType, optionsMap)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}

	optionsModel := OptionsModel{
		Options: mapVal,
	}

	optObj, d := types.ObjectValueFrom(ctx, optionsAttrTypes(), optionsModel)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}

	tmplModel.Options = optObj

	return diags
}

// metakubeMaintenanceCronJobBuildPatch builds a merge patch from the plan spec.
func metakubeMaintenanceCronJobBuildPatch(ctx context.Context, value types.Object) (map[string]any, diag.Diagnostics) {
	spec, diags := metakubeMaintenanceCronJobExpandSpec(ctx, value)
	if diags.HasError() || spec == nil {
		return nil, diags
	}

	tmpl := map[string]any{}
	if spec.MaintenanceJobTemplate != nil {
		t := spec.MaintenanceJobTemplate
		tmpl["type"] = t.Type
		tmpl["rollback"] = t.Rollback
		if t.Options != nil {
			tmpl["options"] = t.Options
		}
	}

	return map[string]any{
		"spec": map[string]any{
			"schedule":               spec.Schedule,
			"maintenanceJobTemplate": tmpl,
		},
	}, diags
}

func metakubeMaintenanceCronJobOptionsChanged(ctx context.Context, planValue, stateValue types.Object) (bool, diag.Diagnostics) {
	planSpec, diags := metakubeMaintenanceCronJobExpandSpec(ctx, planValue)
	stateSpec, d := metakubeMaintenanceCronJobExpandSpec(ctx, stateValue)
	diags.Append(d...)
	if diags.HasError() {
		return false, diags
	}

	var planOptions, stateOptions map[string]string
	if planSpec != nil && planSpec.MaintenanceJobTemplate != nil {
		planOptions = planSpec.MaintenanceJobTemplate.Options
	}
	if stateSpec != nil && stateSpec.MaintenanceJobTemplate != nil {
		stateOptions = stateSpec.MaintenanceJobTemplate.Options
	}

	return !maps.Equal(planOptions, stateOptions), diags
}

// expanders

func metakubeMaintenanceCronJobExpandSpec(ctx context.Context, value types.Object) (*models.MaintenanceCronJobSpec, diag.Diagnostics) {
	if value.IsNull() || value.IsUnknown() {
		return nil, nil
	}

	spec, diags := common.ObjectAs[SpecModel](ctx, value)
	if diags.HasError() {
		return nil, diags
	}

	template, d := metakubeMaintenanceCronJobExpandMaintenanceJobTemplate(ctx, spec.MaintenanceJobTemplate)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}

	return &models.MaintenanceCronJobSpec{
		Schedule:               spec.Schedule.ValueString(),
		MaintenanceJobTemplate: template,
	}, diags
}

func metakubeMaintenanceCronJobExpandMaintenanceJobTemplate(ctx context.Context, value types.Object) (*models.MaintenanceJobTemplate, diag.Diagnostics) {
	if value.IsNull() || value.IsUnknown() {
		return nil, nil
	}

	tmpl, diags := common.ObjectAs[MaintenanceJobTemplateModel](ctx, value)
	if diags.HasError() {
		return nil, diags
	}

	options, d := metakubeMaintenanceCronJobExpandOptions(ctx, tmpl.Options)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}

	return &models.MaintenanceJobTemplate{
		Rollback: tmpl.Rollback.ValueBool(),
		Type:     tmpl.Type.ValueString(),
		Options:  options,
	}, diags
}

func metakubeMaintenanceCronJobExpandOptions(ctx context.Context, value types.Object) (map[string]string, diag.Diagnostics) {
	opts, diags := common.ObjectAs[OptionsModel](ctx, value)
	if diags.HasError() || opts.Options.IsNull() || opts.Options.IsUnknown() {
		return nil, diags
	}

	var result map[string]string
	diags.Append(opts.Options.ElementsAs(ctx, &result, false)...)
	return result, diags
}
