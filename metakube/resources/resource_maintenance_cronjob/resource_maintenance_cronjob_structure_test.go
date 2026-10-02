package resource_maintenance_cronjob

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/syseleven/go-metakube/models"
)

func TestMaintenanceCronJobSpecConversion(t *testing.T) {
	for _, tc := range []struct {
		name  string
		api   *models.MaintenanceCronJobSpec
		value types.Object
	}{
		{
			name: "full spec",
			api: &models.MaintenanceCronJobSpec{
				Schedule: "5 4 * * *",
				MaintenanceJobTemplate: &models.MaintenanceJobTemplate{
					Type:     "kubernetesPatchUpdate",
					Rollback: true,
					Options:  map[string]string{"version": "1.32.1"},
				},
			},
			value: maintenanceSpecValue("5 4 * * *", true, types.ObjectValueMust(optionsAttrTypes(), map[string]attr.Value{
				"options": types.MapValueMust(types.StringType, map[string]attr.Value{"version": types.StringValue("1.32.1")}),
			})),
		},
		{
			name: "omitted options",
			api: &models.MaintenanceCronJobSpec{
				Schedule:               "5 4 * * *",
				MaintenanceJobTemplate: &models.MaintenanceJobTemplate{Type: "kubernetesPatchUpdate"},
			},
			value: maintenanceSpecValue("5 4 * * *", false, types.ObjectNull(optionsAttrTypes())),
		},
		{
			name: "nil template",
			api:  &models.MaintenanceCronJobSpec{Schedule: "5 4 * * *"},
			value: types.ObjectValueMust(specAttrTypes(), map[string]attr.Value{
				"schedule":                 types.StringValue("5 4 * * *"),
				"maintenance_job_template": types.ObjectNull(maintenanceJobTemplateAttrTypes()),
			}),
		},
		{name: "nil spec", value: types.ObjectNull(specAttrTypes())},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var model MaintenanceCronJobModel
			if diags := metakubeMaintenanceCronJobFlattenSpec(t.Context(), &model, tc.api); diags.HasError() {
				t.Fatalf("flatten spec: %v", diags)
			}
			if !model.Spec.Equal(tc.value) {
				t.Errorf("flatten spec = %s, want %s", model.Spec, tc.value)
			}
			// Expand an independently specified Terraform value, not the flattener output.
			got, diags := metakubeMaintenanceCronJobExpandSpec(t.Context(), tc.value)
			if diags.HasError() {
				t.Fatalf("expand spec: %v", diags)
			}
			if diff := cmp.Diff(tc.api, got); diff != "" {
				t.Errorf("expand spec mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMaintenanceCronJobExpandUnknownSpec(t *testing.T) {
	got, diags := metakubeMaintenanceCronJobExpandSpec(t.Context(), types.ObjectUnknown(specAttrTypes()))
	if diags.HasError() || got != nil {
		t.Errorf("expand unknown spec = %v, %v, want nil without errors", got, diags)
	}
}

func TestMaintenanceCronJobBuildPatch(t *testing.T) {
	value := maintenanceSpecValue("0 2 * * *", false, types.ObjectValueMust(optionsAttrTypes(), map[string]attr.Value{
		"options": types.MapValueMust(types.StringType, map[string]attr.Value{"version": types.StringValue("1.32.1")}),
	}))
	got, diags := metakubeMaintenanceCronJobBuildPatch(t.Context(), value)
	if diags.HasError() {
		t.Fatal(diags)
	}
	want := map[string]any{"spec": map[string]any{
		"schedule": "0 2 * * *",
		"maintenanceJobTemplate": map[string]any{
			"type":     "kubernetesPatchUpdate",
			"rollback": false,
			"options":  map[string]string{"version": "1.32.1"},
		},
	}}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("build patch mismatch (-want +got):\n%s", diff)
	}
}

func TestMaintenanceCronJobOptionsChanged(t *testing.T) {
	populated := types.ObjectValueMust(optionsAttrTypes(), map[string]attr.Value{
		"options": types.MapValueMust(types.StringType, map[string]attr.Value{"version": types.StringValue("1.32.1")}),
	})
	empty := types.ObjectValueMust(optionsAttrTypes(), map[string]attr.Value{
		"options": types.MapValueMust(types.StringType, map[string]attr.Value{}),
	})
	null := types.ObjectNull(optionsAttrTypes())
	for _, tc := range []struct {
		name        string
		plan, state types.Object
		want        bool
	}{
		{name: "schedule alone changes", plan: populated, state: populated},
		{name: "options added", plan: populated, state: null, want: true},
		{name: "options removed", plan: null, state: populated, want: true},
		{name: "options cleared", plan: empty, state: populated, want: true},
		{name: "empty and absent options", plan: empty, state: null},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, diags := metakubeMaintenanceCronJobOptionsChanged(t.Context(),
				maintenanceSpecValue("0 2 * * *", false, tc.plan),
				maintenanceSpecValue("5 4 * * *", false, tc.state))
			if diags.HasError() {
				t.Fatal(diags)
			}
			if got != tc.want {
				t.Errorf("options changed = %t, want %t", got, tc.want)
			}
		})
	}
}

func TestMaintenanceCronJobInvalidOptionsStopPatch(t *testing.T) {
	value := maintenanceSpecValue("5 4 * * *", false, types.ObjectValueMust(optionsAttrTypes(), map[string]attr.Value{
		"options": types.MapValueMust(types.StringType, map[string]attr.Value{"version": types.StringUnknown()}),
	}))
	patch, diags := metakubeMaintenanceCronJobBuildPatch(t.Context(), value)
	if !diags.HasError() || patch != nil {
		t.Errorf("patch with unknown option = %v, %v, want nil and an error", patch, diags)
	}
}

func maintenanceSpecValue(schedule string, rollback bool, options types.Object) types.Object {
	return types.ObjectValueMust(specAttrTypes(), map[string]attr.Value{
		"schedule": types.StringValue(schedule),
		"maintenance_job_template": types.ObjectValueMust(maintenanceJobTemplateAttrTypes(), map[string]attr.Value{
			"type":     types.StringValue("kubernetesPatchUpdate"),
			"rollback": types.BoolValue(rollback),
			"options":  options,
		}),
	})
}

func TestMaintenanceCronJobRefreshPreservesEmptyOptions(t *testing.T) {
	for name, options := range map[string]types.Object{
		"omitted":      types.ObjectNull(optionsAttrTypes()),
		"empty object": types.ObjectValueMust(optionsAttrTypes(), map[string]attr.Value{"options": types.MapNull(types.StringType)}),
		"empty map":    types.ObjectValueMust(optionsAttrTypes(), map[string]attr.Value{"options": types.MapValueMust(types.StringType, map[string]attr.Value{})}),
	} {
		t.Run(name, func(t *testing.T) {
			want := maintenanceSpecValue("5 4 * * *", false, options)
			model := MaintenanceCronJobModel{Spec: want}
			api := &models.MaintenanceCronJobSpec{
				Schedule:               "5 4 * * *",
				MaintenanceJobTemplate: &models.MaintenanceJobTemplate{Type: "kubernetesPatchUpdate"},
			}
			if diags := metakubeMaintenanceCronJobFlattenSpec(t.Context(), &model, api); diags.HasError() {
				t.Fatal(diags)
			}
			if !model.Spec.Equal(want) {
				t.Errorf("refresh changed empty options: got %s, want %s", model.Spec, want)
			}
		})
	}
}

func TestMaintenanceCronJobRefreshDetectsRemovedOptions(t *testing.T) {
	options := types.ObjectValueMust(optionsAttrTypes(), map[string]attr.Value{
		"options": types.MapValueMust(types.StringType, map[string]attr.Value{"version": types.StringValue("1.32.1")}),
	})
	model := MaintenanceCronJobModel{Spec: maintenanceSpecValue("5 4 * * *", false, options)}
	api := &models.MaintenanceCronJobSpec{
		Schedule:               "5 4 * * *",
		MaintenanceJobTemplate: &models.MaintenanceJobTemplate{Type: "kubernetesPatchUpdate"},
	}
	if diags := metakubeMaintenanceCronJobFlattenSpec(t.Context(), &model, api); diags.HasError() {
		t.Fatal(diags)
	}
	want := maintenanceSpecValue("5 4 * * *", false, types.ObjectValueMust(optionsAttrTypes(), map[string]attr.Value{
		"options": types.MapValueMust(types.StringType, map[string]attr.Value{}),
	}))
	if !model.Spec.Equal(want) {
		t.Errorf("refresh after remote option removal = %s, want %s", model.Spec, want)
	}
}
