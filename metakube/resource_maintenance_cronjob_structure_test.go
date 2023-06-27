package metakube

import (
	"github.com/google/go-cmp/cmp"
	"github.com/syseleven/go-metakube/models"
	"testing"
)

func TestMetakubeMaintenanceCronJobFlattenSpec(t *testing.T) {
	cases := []struct {
		Input          *models.MaintenanceCronJobSpec
		ExpectedOutput []interface{}
	}{
		{
			&models.MaintenanceCronJobSpec{
				FailedJobsHistoryLimit:     1,
				StartingDeadlineSeconds:    1,
				SuccessfulJobsHistoryLimit: 1,
				Schedule:                   "5 4 * * *",
				MaintenanceJobTemplate:     &models.MaintenanceJobTemplateSpec{},
			},
			[]interface{}{
				map[string]interface{}{
					"failed_jobs_history_limit":     int32(1),
					"starting_deadline_seconds":     int32(1),
					"successful_jobs_history_limit": int32(1),
					"schedule":                      "5 4 * * *",
					"maintenance_job_template":      []interface{}{map[string]interface{}{}},
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

func TestMetakubeMaintenanceCronJobFlattenMaintenanceJobTemplateSpec(t *testing.T) {
	cases := []struct {
		Input          *models.MaintenanceJobTemplateSpec
		ExpectedOutput []interface{}
	}{
		{
			&models.MaintenanceJobTemplateSpec{
				Labels: map[string]string{
					"foo": "bar",
				},
				Name: "maintenance_job_template_spec_name",
				Spec: &models.MaintenanceJobSpec{},
			},
			[]interface{}{
				map[string]interface{}{
					"labels": map[string]string{
						"foo": "bar",
					},
					"name": "maintenance_job_template_spec_name",
					"spec": []interface{}{map[string]interface{}{}},
				},
			},
		},
		{
			&models.MaintenanceJobTemplateSpec{},
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
		output := metakubeMaintenanceCronJobFlattenMaintenanceJobTemplateSpec(tc.Input)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from flattener: mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestMetakubeMaintenanceCronJobFlattenMaintenanceJobSpec(t *testing.T) {
	cases := []struct {
		Input          *models.MaintenanceJobSpec
		ExpectedOutput []interface{}
	}{
		{
			&models.MaintenanceJobSpec{
				Options: map[string]string{
					"foo": "bar",
				},
				Rollback: false,
				Type:     "maintenance_job_type",
				Cluster:  &models.ObjectReference{},
			},
			[]interface{}{
				map[string]interface{}{
					"options": map[string]string{
						"foo": "bar",
					},
					"rollback": false,
					"type":     "maintenance_job_type",
					"cluster":  []interface{}{map[string]interface{}{}},
				},
			},
		},
		{
			&models.MaintenanceJobSpec{},
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
		output := metakubeMaintenanceCronJobFlattenMaintenanceJobSpec(tc.Input)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from flattener: mismatch (-want +got):\n%s", diff)
		}
	}
}

// TODO need for a discussion
func TestMetakubeMaintenanceCronJobFlattenClusterObjectReference(t *testing.T) {

}

func TestMetakubeMaintenanceCronJobExpandSpec(t *testing.T) {
	cases := []struct {
		Input          []interface{}
		ExpectedOutput *models.MaintenanceCronJobSpec
	}{
		{[]interface{}{
			map[string]interface{}{
				"failed_jobs_history_limit":     int32(1),
				"starting_deadline_seconds":     int32(1),
				"successful_jobs_history_limit": int32(1),
				"schedule":                      "5 4 * * *",
				"maintenance_job_template":      []interface{}{map[string]interface{}{}},
			},
		},
			&models.MaintenanceCronJobSpec{
				FailedJobsHistoryLimit:     1,
				StartingDeadlineSeconds:    1,
				SuccessfulJobsHistoryLimit: 1,
				Schedule:                   "5 4 * * *",
				MaintenanceJobTemplate:     &models.MaintenanceJobTemplateSpec{},
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
		output := metakubeMaintenanceCronJobExpandMaintenanceJobTemplateSpec(tc.Input)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from expander: mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestMetakubeMaintenanceCronJobExpandMaintenanceJobTemplateSpec(t *testing.T) {
	cases := []struct {
		Input          []interface{}
		ExpectedOutput *models.MaintenanceJobTemplateSpec
	}{
		{[]interface{}{
			map[string]interface{}{
				"labels": map[string]string{
					"foo": "bar",
				},
				"name": "maintenance_job_template_spec_name",
				"spec": []interface{}{map[string]interface{}{}},
			},
		},
			&models.MaintenanceJobTemplateSpec{
				Labels: map[string]string{
					"foo": "bar",
				},
				Name: "maintenance_job_template_spec_name",
				Spec: &models.MaintenanceJobSpec{},
			},
		},
		{
			[]interface{}{
				map[string]interface{}{},
			},
			&models.MaintenanceJobTemplateSpec{},
		},
		{
			[]interface{}{},
			nil,
		},
	}

	for _, tc := range cases {
		output := metakubeMaintenanceCronJobExpandMaintenanceJobTemplateSpec(tc.Input)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from expander: mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestMetakubeMaintenanceCronJobExpandMaintenanceJobSpec(t *testing.T) {
	cases := []struct {
		Input          []interface{}
		ExpectedOutput *models.MaintenanceJobSpec
	}{
		{[]interface{}{
			map[string]interface{}{
				"options": map[string]string{
					"foo": "bar",
				},
				"rollback": false,
				"type":     "maintenance_job_type",
				"cluster":  []interface{}{map[string]interface{}{}},
			},
		},
			&models.MaintenanceJobSpec{
				Options: map[string]string{
					"foo": "bar",
				},
				Rollback: false,
				Type:     "maintenance_job_type",
				Cluster:  &models.ObjectReference{},
			},
		},
		{
			[]interface{}{
				map[string]interface{}{},
			},
			&models.MaintenanceJobSpec{},
		},
		{
			[]interface{}{},
			nil,
		},
	}

	for _, tc := range cases {
		output := metakubeMaintenanceCronJobExpandMaintenanceJobSpec(tc.Input)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from flattener: mismatch (-want +got):\n%s", diff)
		}
	}
}

// TODO need for a discussion
func TestMetakubeExpandClusterObjectReference(t *testing.T) {

}
