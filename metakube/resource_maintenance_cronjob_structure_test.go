package metakube

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/syseleven/go-metakube/models"
)

func TestMetakubeMaintenanceCronJobFlattenSpec(t *testing.T) {
	cases := []struct {
		Input          *models.MaintenanceCronJobSpec
		ExpectedOutput []interface{}
	}{
		{
			&models.MaintenanceCronJobSpec{
				Schedule:               "5 4 * * *",
				MaintenanceJobTemplate: &models.MaintenanceJobTemplate{},
			},
			[]interface{}{
				map[string]interface{}{
					"schedule": "5 4 * * *",
					"maintenance_job_template": []interface{}{map[string]interface{}{
						"rollback": false,
					}},
				},
			},
		},
		{
			&models.MaintenanceCronJobSpec{},
			[]interface{}{
				map[string]interface{}{},
			},
		},
		{
			nil,
			[]interface{}{},
		},
	}

	for _, tc := range cases {
		output := metakubeMaintenanceCronJobFlattenSpec(tc.Input)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from flattener: mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestMetakubeMaintenanceCronJobFlattenMaintenanceJobTemplate(t *testing.T) {
	cases := []struct {
		Input          *models.MaintenanceJobTemplate
		ExpectedOutput []interface{}
	}{
		{
			&models.MaintenanceJobTemplate{
				Options: map[string]string{
					"foo": "bar",
				},
				Rollback: false,
				Type:     "maintenance_job_type",
			},
			[]interface{}{
				map[string]interface{}{
					"options": map[string]string{
						"foo": "bar",
					},
					"rollback": false,
					"type":     "maintenance_job_type",
				},
			},
		},
		{
			&models.MaintenanceJobTemplate{},
			[]interface{}{
				map[string]interface{}{"rollback": false},
			},
		},
		{
			nil,
			[]interface{}{},
		},
	}

	for _, tc := range cases {
		output := metakubeMaintenanceCronJobFlattenMaintenanceJobTemplate(tc.Input)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from flattener: mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestMetakubeMaintenanceCronJobExpandSpec(t *testing.T) {
	cases := []struct {
		Input          []interface{}
		ExpectedOutput *models.MaintenanceCronJobSpec
	}{
		{
			[]interface{}{
				map[string]interface{}{
					"schedule":                 "5 4 * * *",
					"maintenance_job_template": []interface{}{map[string]interface{}{}},
				},
			},
			&models.MaintenanceCronJobSpec{
				Schedule:               "5 4 * * *",
				MaintenanceJobTemplate: &models.MaintenanceJobTemplate{},
			},
		},
		{
			[]interface{}{
				map[string]interface{}{},
			},
			&models.MaintenanceCronJobSpec{},
		},
		{
			[]interface{}{},
			nil,
		},
	}

	for _, tc := range cases {
		output := metakubeMaintenanceCronJobExpandSpec(tc.Input)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from expander: mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestMetakubeMaintenanceCronJobExpandMaintenanceJobTemplate(t *testing.T) {
	cases := []struct {
		Input          []interface{}
		ExpectedOutput *models.MaintenanceJobTemplate
	}{
		{[]interface{}{
			map[string]interface{}{
				"options": map[string]interface{}{
					"foo": "bar",
				},
				"rollback": false,
				"type":     "maintenance_job_type",
			},
		},
			&models.MaintenanceJobTemplate{
				Options: map[string]string{
					"foo": "bar",
				},
				Rollback: false,
				Type:     "maintenance_job_type",
			},
		},
		{
			[]interface{}{
				map[string]interface{}{},
			},
			&models.MaintenanceJobTemplate{},
		},
		{
			[]interface{}{},
			nil,
		},
	}

	for _, tc := range cases {
		output := metakubeMaintenanceCronJobExpandMaintenanceJobTemplate(tc.Input)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from flattener: mismatch (-want +got):\n%s", diff)
		}
	}
}
