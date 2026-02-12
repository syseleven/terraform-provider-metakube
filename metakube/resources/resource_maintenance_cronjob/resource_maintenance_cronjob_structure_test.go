package resource_maintenance_cronjob

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/syseleven/go-metakube/models"
)

func TestMetakubeMaintenanceCronJobFlattenSpec(t *testing.T) {
	ctx := context.Background()

	t.Run("full spec with template", func(t *testing.T) {
		model := &MaintenanceCronJobModel{}
		input := &models.MaintenanceCronJobSpec{
			Schedule: "5 4 * * *",
			MaintenanceJobTemplate: &models.MaintenanceJobTemplate{
				Type:     "kubernetesPatchUpdate",
				Rollback: false,
			},
		}

		diags := metakubeMaintenanceCronJobFlattenSpec(ctx, model, input)
		if diags.HasError() {
			t.Fatalf("unexpected error: %v", diags.Errors())
		}

		if model.Spec.IsNull() {
			t.Fatal("spec should not be null")
		}

		var specs []SpecModel
		diags = model.Spec.ElementsAs(ctx, &specs, false)
		if diags.HasError() {
			t.Fatalf("error extracting specs: %v", diags.Errors())
		}
		if len(specs) != 1 {
			t.Fatalf("expected 1 spec, got %d", len(specs))
		}
		if specs[0].Schedule.ValueString() != "5 4 * * *" {
			t.Fatalf("expected schedule '5 4 * * *', got '%s'", specs[0].Schedule.ValueString())
		}

		var tmpls []MaintenanceJobTemplateModel
		diags = specs[0].MaintenanceJobTemplate.ElementsAs(ctx, &tmpls, false)
		if diags.HasError() {
			t.Fatalf("error extracting templates: %v", diags.Errors())
		}
		if len(tmpls) != 1 {
			t.Fatalf("expected 1 template, got %d", len(tmpls))
		}
		if tmpls[0].Type.ValueString() != "kubernetesPatchUpdate" {
			t.Fatalf("expected type 'kubernetesPatchUpdate', got '%s'", tmpls[0].Type.ValueString())
		}
		if tmpls[0].Rollback.ValueBool() != false {
			t.Fatal("expected rollback to be false")
		}
	})

	t.Run("empty spec", func(t *testing.T) {
		model := &MaintenanceCronJobModel{}
		input := &models.MaintenanceCronJobSpec{}

		diags := metakubeMaintenanceCronJobFlattenSpec(ctx, model, input)
		if diags.HasError() {
			t.Fatalf("unexpected error: %v", diags.Errors())
		}

		if model.Spec.IsNull() {
			t.Fatal("spec should not be null for empty input")
		}

		var specs []SpecModel
		diags = model.Spec.ElementsAs(ctx, &specs, false)
		if diags.HasError() {
			t.Fatalf("error extracting specs: %v", diags.Errors())
		}
		if len(specs) != 1 {
			t.Fatalf("expected 1 spec, got %d", len(specs))
		}
	})

	t.Run("nil spec", func(t *testing.T) {
		model := &MaintenanceCronJobModel{}
		diags := metakubeMaintenanceCronJobFlattenSpec(ctx, model, nil)
		if diags.HasError() {
			t.Fatalf("unexpected error: %v", diags.Errors())
		}
		if !model.Spec.IsNull() {
			t.Fatal("spec should be null for nil input")
		}
	})
}

func TestMetakubeMaintenanceCronJobFlattenMaintenanceJobTemplate(t *testing.T) {
	ctx := context.Background()

	t.Run("full template with options", func(t *testing.T) {
		specModel := &SpecModel{}
		input := &models.MaintenanceJobTemplate{
			Options: map[string]string{
				"foo": "bar",
			},
			Rollback: false,
			Type:     "maintenance_job_type",
		}

		diags := metakubeMaintenanceCronJobFlattenMaintenanceJobTemplate(ctx, specModel, input)
		if diags.HasError() {
			t.Fatalf("unexpected error: %v", diags.Errors())
		}

		if specModel.MaintenanceJobTemplate.IsNull() {
			t.Fatal("maintenance_job_template should not be null")
		}

		var tmpls []MaintenanceJobTemplateModel
		diags = specModel.MaintenanceJobTemplate.ElementsAs(ctx, &tmpls, false)
		if diags.HasError() {
			t.Fatalf("error extracting templates: %v", diags.Errors())
		}
		if len(tmpls) != 1 {
			t.Fatalf("expected 1 template, got %d", len(tmpls))
		}
		if tmpls[0].Type.ValueString() != "maintenance_job_type" {
			t.Fatalf("expected type 'maintenance_job_type', got '%s'", tmpls[0].Type.ValueString())
		}
		if tmpls[0].Rollback.ValueBool() != false {
			t.Fatal("expected rollback to be false")
		}
		if tmpls[0].Options.IsNull() {
			t.Fatal("options should not be null when options are provided")
		}
	})

	t.Run("empty template", func(t *testing.T) {
		specModel := &SpecModel{}
		input := &models.MaintenanceJobTemplate{}

		diags := metakubeMaintenanceCronJobFlattenMaintenanceJobTemplate(ctx, specModel, input)
		if diags.HasError() {
			t.Fatalf("unexpected error: %v", diags.Errors())
		}

		if specModel.MaintenanceJobTemplate.IsNull() {
			t.Fatal("maintenance_job_template should not be null for empty input")
		}
	})

	t.Run("nil template", func(t *testing.T) {
		specModel := &SpecModel{}
		diags := metakubeMaintenanceCronJobFlattenMaintenanceJobTemplate(ctx, specModel, nil)
		if diags.HasError() {
			t.Fatalf("unexpected error: %v", diags.Errors())
		}
		if !specModel.MaintenanceJobTemplate.IsNull() {
			t.Fatal("maintenance_job_template should be null for nil input")
		}
	})
}

func TestMetakubeMaintenanceCronJobExpandSpec(t *testing.T) {
	ctx := context.Background()

	t.Run("full spec", func(t *testing.T) {
		tmplModel := MaintenanceJobTemplateModel{
			Options:  types.ListNull(types.ObjectType{AttrTypes: optionsBlockAttrTypes()}),
			Rollback: types.BoolValue(false),
			Type:     types.StringValue("kubernetesPatchUpdate"),
		}
		tmplObj, diags := types.ObjectValueFrom(ctx, maintenanceJobTemplateAttrTypes(), tmplModel)
		if diags.HasError() {
			t.Fatalf("error creating template object: %v", diags.Errors())
		}
		tmplList, diags := types.ListValue(types.ObjectType{AttrTypes: maintenanceJobTemplateAttrTypes()}, []attr.Value{tmplObj})
		if diags.HasError() {
			t.Fatalf("error creating template list: %v", diags.Errors())
		}

		specModel := SpecModel{
			Schedule:               types.StringValue("5 4 * * *"),
			MaintenanceJobTemplate: tmplList,
		}
		specObj, diags := types.ObjectValueFrom(ctx, specAttrTypes(), specModel)
		if diags.HasError() {
			t.Fatalf("error creating spec object: %v", diags.Errors())
		}
		specList, diags := types.ListValue(types.ObjectType{AttrTypes: specAttrTypes()}, []attr.Value{specObj})
		if diags.HasError() {
			t.Fatalf("error creating spec list: %v", diags.Errors())
		}

		result := metakubeMaintenanceCronJobExpandSpec(ctx, specList)

		expected := &models.MaintenanceCronJobSpec{
			Schedule: "5 4 * * *",
			MaintenanceJobTemplate: &models.MaintenanceJobTemplate{
				Type:     "kubernetesPatchUpdate",
				Rollback: false,
			},
		}

		if diff := cmp.Diff(expected, result); diff != "" {
			t.Fatalf("unexpected result: mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("null spec", func(t *testing.T) {
		specList := types.ListNull(types.ObjectType{AttrTypes: specAttrTypes()})
		result := metakubeMaintenanceCronJobExpandSpec(ctx, specList)
		if result != nil {
			t.Fatalf("expected nil result for null spec, got %v", result)
		}
	})

	t.Run("empty spec", func(t *testing.T) {
		specList, diags := types.ListValue(types.ObjectType{AttrTypes: specAttrTypes()}, []attr.Value{})
		if diags.HasError() {
			t.Fatalf("error creating empty spec list: %v", diags.Errors())
		}
		result := metakubeMaintenanceCronJobExpandSpec(ctx, specList)
		if result != nil {
			t.Fatalf("expected nil result for empty spec, got %v", result)
		}
	})
}

func TestMetakubeMaintenanceCronJobExpandMaintenanceJobTemplate(t *testing.T) {
	ctx := context.Background()

	t.Run("full template with options", func(t *testing.T) {
		optionsMap, diags := types.MapValue(types.StringType, map[string]attr.Value{
			"foo": types.StringValue("bar"),
		})
		if diags.HasError() {
			t.Fatalf("error creating options map: %v", diags.Errors())
		}
		optModel := OptionsBlockModel{Options: optionsMap}
		optObj, diags := types.ObjectValueFrom(ctx, optionsBlockAttrTypes(), optModel)
		if diags.HasError() {
			t.Fatalf("error creating options object: %v", diags.Errors())
		}
		optList, diags := types.ListValue(types.ObjectType{AttrTypes: optionsBlockAttrTypes()}, []attr.Value{optObj})
		if diags.HasError() {
			t.Fatalf("error creating options list: %v", diags.Errors())
		}

		tmplModel := MaintenanceJobTemplateModel{
			Options:  optList,
			Rollback: types.BoolValue(false),
			Type:     types.StringValue("maintenance_job_type"),
		}
		tmplObj, diags := types.ObjectValueFrom(ctx, maintenanceJobTemplateAttrTypes(), tmplModel)
		if diags.HasError() {
			t.Fatalf("error creating template object: %v", diags.Errors())
		}
		tmplList, diags := types.ListValue(types.ObjectType{AttrTypes: maintenanceJobTemplateAttrTypes()}, []attr.Value{tmplObj})
		if diags.HasError() {
			t.Fatalf("error creating template list: %v", diags.Errors())
		}

		result := metakubeMaintenanceCronJobExpandMaintenanceJobTemplate(ctx, tmplList)

		expected := &models.MaintenanceJobTemplate{
			Options: map[string]string{
				"foo": "bar",
			},
			Rollback: false,
			Type:     "maintenance_job_type",
		}

		if diff := cmp.Diff(expected, result); diff != "" {
			t.Fatalf("unexpected result: mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("template without options", func(t *testing.T) {
		tmplModel := MaintenanceJobTemplateModel{
			Options:  types.ListNull(types.ObjectType{AttrTypes: optionsBlockAttrTypes()}),
			Rollback: types.BoolValue(false),
			Type:     types.StringValue("maintenance_job_type"),
		}
		tmplObj, diags := types.ObjectValueFrom(ctx, maintenanceJobTemplateAttrTypes(), tmplModel)
		if diags.HasError() {
			t.Fatalf("error creating template object: %v", diags.Errors())
		}
		tmplList, diags := types.ListValue(types.ObjectType{AttrTypes: maintenanceJobTemplateAttrTypes()}, []attr.Value{tmplObj})
		if diags.HasError() {
			t.Fatalf("error creating template list: %v", diags.Errors())
		}

		result := metakubeMaintenanceCronJobExpandMaintenanceJobTemplate(ctx, tmplList)

		expected := &models.MaintenanceJobTemplate{
			Rollback: false,
			Type:     "maintenance_job_type",
		}

		if diff := cmp.Diff(expected, result); diff != "" {
			t.Fatalf("unexpected result: mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("null template", func(t *testing.T) {
		tmplList := types.ListNull(types.ObjectType{AttrTypes: maintenanceJobTemplateAttrTypes()})
		result := metakubeMaintenanceCronJobExpandMaintenanceJobTemplate(ctx, tmplList)
		if result != nil {
			t.Fatalf("expected nil result for null template, got %v", result)
		}
	})
}
