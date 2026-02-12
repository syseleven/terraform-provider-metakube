package resource_maintenance_cronjob

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/syseleven/go-metakube/models"
)

// flatteners

func metakubeMaintenanceCronJobFlattenSpec(ctx context.Context, model *MaintenanceCronJobModel, in *models.MaintenanceCronJobSpec) diag.Diagnostics {
	var diags diag.Diagnostics

	if in == nil {
		model.Spec = types.ListNull(types.ObjectType{AttrTypes: specAttrTypes()})
		return diags
	}

	specModel := SpecModel{
		Schedule: types.StringValue(in.Schedule),
	}

	diags.Append(metakubeMaintenanceCronJobFlattenMaintenanceJobTemplate(ctx, &specModel, in.MaintenanceJobTemplate)...)
	if diags.HasError() {
		return diags
	}

	specObj, d := types.ObjectValueFrom(ctx, specAttrTypes(), specModel)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}

	specList, d := types.ListValue(types.ObjectType{AttrTypes: specAttrTypes()}, []attr.Value{specObj})
	diags.Append(d...)
	model.Spec = specList

	return diags
}

func metakubeMaintenanceCronJobFlattenMaintenanceJobTemplate(ctx context.Context, specModel *SpecModel, in *models.MaintenanceJobTemplate) diag.Diagnostics {
	var diags diag.Diagnostics

	if in == nil {
		specModel.MaintenanceJobTemplate = types.ListNull(types.ObjectType{AttrTypes: maintenanceJobTemplateAttrTypes()})
		return diags
	}

	tmplModel := MaintenanceJobTemplateModel{
		Rollback: types.BoolValue(in.Rollback),
		Type:     types.StringValue(in.Type),
	}

	diags.Append(metakubeMaintenanceCronJobFlattenOptions(ctx, &tmplModel, in.Options)...)
	if diags.HasError() {
		return diags
	}

	tmplObj, d := types.ObjectValueFrom(ctx, maintenanceJobTemplateAttrTypes(), tmplModel)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}

	tmplList, d := types.ListValue(types.ObjectType{AttrTypes: maintenanceJobTemplateAttrTypes()}, []attr.Value{tmplObj})
	diags.Append(d...)
	specModel.MaintenanceJobTemplate = tmplList

	return diags
}

func metakubeMaintenanceCronJobFlattenOptions(ctx context.Context, tmplModel *MaintenanceJobTemplateModel, in map[string]string) diag.Diagnostics {
	var diags diag.Diagnostics

	if len(in) == 0 {
		tmplModel.Options = types.ListNull(types.ObjectType{AttrTypes: optionsBlockAttrTypes()})
		return diags
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

	optionsBlockModel := OptionsBlockModel{
		Options: mapVal,
	}

	optObj, d := types.ObjectValueFrom(ctx, optionsBlockAttrTypes(), optionsBlockModel)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}

	optList, d := types.ListValue(types.ObjectType{AttrTypes: optionsBlockAttrTypes()}, []attr.Value{optObj})
	diags.Append(d...)
	tmplModel.Options = optList

	return diags
}

// metakubeMaintenanceCronJobBuildPatch builds a map[string]any patch from the plan spec
func metakubeMaintenanceCronJobBuildPatch(ctx context.Context, specList types.List) map[string]any {
	if specList.IsNull() || specList.IsUnknown() || len(specList.Elements()) == 0 {
		return map[string]any{}
	}

	var specModels []SpecModel
	if diags := specList.ElementsAs(ctx, &specModels, false); diags.HasError() || len(specModels) == 0 {
		return map[string]any{}
	}

	spec := specModels[0]

	tmpl := map[string]any{}
	if !spec.MaintenanceJobTemplate.IsNull() && !spec.MaintenanceJobTemplate.IsUnknown() {
		var tmplModels []MaintenanceJobTemplateModel
		if diags := spec.MaintenanceJobTemplate.ElementsAs(ctx, &tmplModels, false); diags == nil || !diags.HasError() {
			if len(tmplModels) > 0 {
				t := tmplModels[0]
				tmpl["type"] = t.Type.ValueString()
				tmpl["rollback"] = t.Rollback.ValueBool()
				if opts := metakubeMaintenanceCronJobExpandOptions(ctx, t.Options); opts != nil {
					tmpl["options"] = opts
				}
			}
		}
	}

	return map[string]any{
		"spec": map[string]any{
			"schedule":                spec.Schedule.ValueString(),
			"maintenanceJobTemplate": tmpl,
		},
	}
}

// expanders

func metakubeMaintenanceCronJobExpandSpec(ctx context.Context, specList types.List) *models.MaintenanceCronJobSpec {
	if specList.IsNull() || specList.IsUnknown() || len(specList.Elements()) == 0 {
		return nil
	}

	var specModels []SpecModel
	if diags := specList.ElementsAs(ctx, &specModels, false); diags.HasError() || len(specModels) == 0 {
		return nil
	}

	spec := specModels[0]
	obj := &models.MaintenanceCronJobSpec{
		Schedule: spec.Schedule.ValueString(),
	}

	obj.MaintenanceJobTemplate = metakubeMaintenanceCronJobExpandMaintenanceJobTemplate(ctx, spec.MaintenanceJobTemplate)

	return obj
}

func metakubeMaintenanceCronJobExpandMaintenanceJobTemplate(ctx context.Context, tmplList types.List) *models.MaintenanceJobTemplate {
	if tmplList.IsNull() || tmplList.IsUnknown() || len(tmplList.Elements()) == 0 {
		return nil
	}

	var tmplModels []MaintenanceJobTemplateModel
	if diags := tmplList.ElementsAs(ctx, &tmplModels, false); diags.HasError() || len(tmplModels) == 0 {
		return nil
	}

	tmpl := tmplModels[0]
	obj := &models.MaintenanceJobTemplate{
		Rollback: tmpl.Rollback.ValueBool(),
		Type:     tmpl.Type.ValueString(),
	}

	obj.Options = metakubeMaintenanceCronJobExpandOptions(ctx, tmpl.Options)

	return obj
}

func metakubeMaintenanceCronJobExpandOptions(ctx context.Context, optionsList types.List) map[string]string {
	if optionsList.IsNull() || optionsList.IsUnknown() || len(optionsList.Elements()) == 0 {
		return nil
	}

	var optModels []OptionsBlockModel
	if diags := optionsList.ElementsAs(ctx, &optModels, false); diags.HasError() || len(optModels) == 0 {
		return nil
	}

	opts := optModels[0]
	if opts.Options.IsNull() || opts.Options.IsUnknown() {
		return nil
	}

	result := make(map[string]string)
	for k, v := range opts.Options.Elements() {
		if sv, ok := v.(types.String); ok {
			result[k] = sv.ValueString()
		}
	}

	return result
}
