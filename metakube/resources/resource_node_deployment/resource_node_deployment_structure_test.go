package resource_node_deployment

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/syseleven/go-metakube/models"
	"github.com/syseleven/terraform-provider-metakube/metakube/common"
	"k8s.io/utils/ptr"
)

func TestMetakubeNodeDeploymentFlatten(t *testing.T) {
	cases := []struct {
		Input          *models.NodeDeploymentSpec
		ExpectedOutput []interface{}
	}{
		{
			&models.NodeDeploymentSpec{
				Replicas: common.Int32ToPtr(1),
				Template: &models.NodeSpec{},
			},
			[]interface{}{
				map[string]interface{}{
					"replicas": int32(1),
					"template": []interface{}{map[string]interface{}{}},
				},
			},
		},
		{
			&models.NodeDeploymentSpec{},
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
		output := metakubeNodeDeploymentFlattenSpec(tc.Input)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from flattener: mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestMetakubeNodeDeploymentSpecFlatten(t *testing.T) {
	cases := []struct {
		Input          *models.NodeSpec
		ExpectedOutput []interface{}
	}{
		{
			&models.NodeSpec{
				OperatingSystem: &models.OperatingSystemSpec{
					Ubuntu: &models.UbuntuSpec{},
				},
				Taints: []*models.TaintSpec{
					{
						Key:    "key1",
						Value:  "value1",
						Effect: "NoSchedule",
					},
					{
						Key:    "key2",
						Value:  "value2",
						Effect: "NoSchedule",
					},
				},
				Cloud: &models.NodeCloudSpec{
					Aws: &models.AWSNodeSpec{},
				},
				Labels: map[string]string{
					"foo": "bar",
				},
				Versions: &models.NodeVersionInfo{
					Kubelet: "1.18.8",
				},
			},
			[]interface{}{
				map[string]interface{}{
					"operating_system": []interface{}{
						map[string]interface{}{
							"ubuntu": []interface{}{
								map[string]interface{}{
									"dist_upgrade_on_boot": false,
								},
							},
						},
					},
					"taints": []interface{}{
						map[string]interface{}{
							"key":    "key1",
							"value":  "value1",
							"effect": "NoSchedule",
						},
						map[string]interface{}{
							"key":    "key2",
							"value":  "value2",
							"effect": "NoSchedule",
						},
					},
					"cloud": []interface{}{
						map[string]interface{}{
							"aws": []interface{}{
								map[string]interface{}{
									"assign_public_ip": false,
								},
							},
						},
					},
					"labels": map[string]string{
						"foo": "bar",
					},
					"versions": []interface{}{
						map[string]interface{}{
							"kubelet": "1.18.8",
						},
					},
				},
			},
		},
		{
			&models.NodeSpec{
				Versions: &models.NodeVersionInfo{
					Kubelet: "",
				},
			},
			[]interface{}{
				map[string]interface{}{},
			},
		},
		{
			&models.NodeSpec{},
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
		output := metakubeNodeDeploymentFlattenNodeSpec(tc.Input)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from flattener: mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestMetakubeNodeDeploymentFlattenOperatingSystem(t *testing.T) {
	cases := []struct {
		Input          *models.OperatingSystemSpec
		ExpectedOutput []interface{}
	}{
		{
			&models.OperatingSystemSpec{
				Ubuntu: &models.UbuntuSpec{
					DistUpgradeOnBoot: true,
				},
			},
			[]interface{}{
				map[string]interface{}{
					"ubuntu": []interface{}{
						map[string]interface{}{
							"dist_upgrade_on_boot": true,
						},
					},
				},
			},
		},
		{
			&models.OperatingSystemSpec{
				Flatcar: &models.FlatcarSpec{
					DisableAutoUpdate: true,
				},
			},
			[]interface{}{
				map[string]interface{}{
					"flatcar": []interface{}{
						map[string]interface{}{
							"disable_auto_update": true,
						},
					},
				},
			},
		},
		{
			&models.OperatingSystemSpec{},
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
		output := metakubeNodeDeploymentFlattenOperatingSystem(tc.Input)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from flattener: mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestMetakubeNodeDeploymentFlattenAWSSpec(t *testing.T) {
	cases := []struct {
		Input          *models.AWSNodeSpec
		ExpectedOutput []interface{}
	}{
		{
			&models.AWSNodeSpec{
				AMI:              "ami-5731123e",
				AssignPublicIP:   true,
				AvailabilityZone: "eu-west-1",
				InstanceType:     common.StrToPtr("t3.small"),
				SubnetID:         "subnet-53485",
				Tags: map[string]string{
					"foo": "bar",
				},
				VolumeSize: common.IntToInt32Ptr(25),
				VolumeType: common.StrToPtr("standard"),
			},
			[]interface{}{
				map[string]interface{}{
					"ami":               "ami-5731123e",
					"assign_public_ip":  true,
					"availability_zone": "eu-west-1",
					"instance_type":     "t3.small",
					"subnet_id":         "subnet-53485",
					"tags": map[string]string{
						"foo": "bar",
					},
					"disk_size":   int32(25),
					"volume_type": "standard",
				},
			},
		},
		{
			&models.AWSNodeSpec{},
			[]interface{}{
				map[string]interface{}{
					"assign_public_ip": false,
				},
			},
		},
		{
			nil,
			[]interface{}{},
		},
	}

	for _, tc := range cases {
		output := metakubeNodeDeploymentFlattenAWSSpec(tc.Input)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from flattener: mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestFlattenAzureNodeSpec(t *testing.T) {
	cases := []struct {
		Input          *models.AzureNodeSpec
		ExpectedOutput []interface{}
	}{
		{
			&models.AzureNodeSpec{
				ImageID:        "ImageID",
				Size:           common.StrToPtr("Size"),
				AssignPublicIP: false,
				DataDiskSize:   1,
				OSDiskSize:     2,
				Tags: map[string]string{
					"tag-k": "tag-v",
				},
				Zones: []string{"Zone-x"},
			},
			[]interface{}{
				map[string]interface{}{
					"image_id":         "ImageID",
					"size":             "Size",
					"assign_public_ip": false,
					"disk_size_gb":     int32(1),
					"os_disk_size_gb":  int32(2),
					"tags": map[string]string{
						"tag-k": "tag-v",
					},
					"zones": []string{"Zone-x"},
				},
			},
		},
		{
			&models.AzureNodeSpec{},
			[]interface{}{
				map[string]interface{}{
					"assign_public_ip": false,
					"disk_size_gb":     int32(0),
					"os_disk_size_gb":  int32(0),
				},
			},
		},
		{
			nil,
			[]interface{}{},
		},
	}

	for _, tc := range cases {
		output := metakubeNodeDeploymentFlattenAzureSpec(tc.Input)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from flattener: mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestFlattenOpenstackNodeSpec(t *testing.T) {
	cases := []struct {
		Input          *models.OpenstackNodeSpec
		ExpectedOutput []interface{}
	}{
		{
			&models.OpenstackNodeSpec{
				Flavor:                    common.StrToPtr("big"),
				Image:                     common.StrToPtr("Ubuntu"),
				UseFloatingIP:             ptr.To(true),
				InstanceReadyCheckPeriod:  "10s",
				InstanceReadyCheckTimeout: "120s",
				Tags: map[string]string{
					"foo": "bar",
				},
				RootDiskSizeGB: ptr.To(int64(999)),
			},
			[]interface{}{
				map[string]interface{}{
					"flavor":                       "big",
					"image":                        "Ubuntu",
					"instance_ready_check_period":  "10s",
					"instance_ready_check_timeout": "120s",
					"use_floating_ip":              ptr.To(true),
					"tags": map[string]string{
						"foo": "bar",
					},
					"disk_size": int64(999),
				},
			},
		},
		{
			&models.OpenstackNodeSpec{},
			[]interface{}{
				map[string]interface{}{
					"use_floating_ip": (*bool)(nil),
				},
			},
		},
		{
			nil,
			[]interface{}{},
		},
	}

	for _, tc := range cases {
		output := metakubeNodeDeploymentFlattenOpenstackSpec(tc.Input)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from flattener: mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestExpandNodeDeploymentSpec(t *testing.T) {
	cases := []struct {
		Input          []interface{}
		ExpectedOutput *models.NodeDeploymentSpec
	}{
		{
			[]interface{}{
				map[string]interface{}{
					"replicas": 1,
					"template": []interface{}{map[string]interface{}{}},
				},
			},
			&models.NodeDeploymentSpec{
				Replicas: common.Int32ToPtr(1),
				Template: &models.NodeSpec{},
			},
		},
		{

			[]interface{}{
				map[string]interface{}{},
			},
			&models.NodeDeploymentSpec{},
		},
		{
			[]interface{}{},
			nil,
		},
	}

	for _, tc := range cases {
		output := metakubeNodeDeploymentExpandSpec(tc.Input, false)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from expander: mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestExpandNodeSpec(t *testing.T) {
	cases := []struct {
		Input          []interface{}
		ExpectedOutput *models.NodeSpec
	}{
		{
			[]interface{}{
				map[string]interface{}{
					"operating_system": []interface{}{
						map[string]interface{}{
							"ubuntu": []interface{}{
								map[string]interface{}{
									"dist_upgrade_on_boot": false,
								},
							},
						},
					},
					"taints": []interface{}{
						map[string]interface{}{
							"key":    "key1",
							"value":  "value1",
							"effect": "NoSchedule",
						},
						map[string]interface{}{
							"key":    "key2",
							"value":  "value2",
							"effect": "NoSchedule",
						},
					},
					"cloud": []interface{}{
						map[string]interface{}{
							"aws": []interface{}{
								map[string]interface{}{
									"assign_public_ip": false,
								},
							},
						},
					},
					"labels": map[string]interface{}{
						"foo": "bar",
					},
					"versions": []interface{}{
						map[string]interface{}{
							"kubelet": "1.18.8",
						},
					},
				},
			},
			&models.NodeSpec{
				OperatingSystem: &models.OperatingSystemSpec{
					Ubuntu: &models.UbuntuSpec{},
				},
				Taints: []*models.TaintSpec{
					{
						Key:    "key1",
						Value:  "value1",
						Effect: "NoSchedule",
					},
					{
						Key:    "key2",
						Value:  "value2",
						Effect: "NoSchedule",
					},
				},
				Cloud: &models.NodeCloudSpec{
					Aws: &models.AWSNodeSpec{},
				},
				Labels: map[string]string{
					"foo": "bar",
				},
				Versions: &models.NodeVersionInfo{
					Kubelet: "1.18.8",
				},
			},
		},
		{

			[]interface{}{
				map[string]interface{}{
					"versions": []interface{}{
						map[string]interface{}{
							"kubelet": "",
						},
					},
				},
			},
			&models.NodeSpec{},
		},
		{
			[]interface{}{},
			nil,
		},
	}

	for _, tc := range cases {
		output := metakubeNodeDeploymentExpandNodeSpec(tc.Input)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from expander: mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestExpandOperatingSystem(t *testing.T) {
	cases := []struct {
		Input          []interface{}
		ExpectedOutput *models.OperatingSystemSpec
	}{
		{
			[]interface{}{
				map[string]interface{}{
					"ubuntu": []interface{}{
						map[string]interface{}{
							"dist_upgrade_on_boot": true,
						},
					},
				},
			},
			&models.OperatingSystemSpec{
				Ubuntu: &models.UbuntuSpec{
					DistUpgradeOnBoot: true,
				},
			},
		},
		{
			[]interface{}{
				map[string]interface{}{
					"flatcar": []interface{}{
						map[string]interface{}{
							"disable_auto_update": true,
						},
					},
				},
			},
			&models.OperatingSystemSpec{
				Flatcar: &models.FlatcarSpec{
					DisableAutoUpdate: true,
				},
			},
		},
		{
			[]interface{}{
				map[string]interface{}{},
			},
			&models.OperatingSystemSpec{},
		},
		{
			[]interface{}{},
			nil,
		},
	}

	for _, tc := range cases {
		output := metakubeNodeDeploymentExpandOS(tc.Input)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from expander: mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestExpandAWSNodeSpec(t *testing.T) {
	cases := []struct {
		Input          []interface{}
		ExpectedOutput *models.AWSNodeSpec
	}{
		{
			[]interface{}{
				map[string]interface{}{
					"ami":               "ami-5731123e",
					"assign_public_ip":  true,
					"availability_zone": "eu-west-1",
					"instance_type":     "t3.small",
					"subnet_id":         "subnet-53485",
					"tags": map[string]interface{}{
						"foo": "bar",
					},
					"disk_size":   25,
					"volume_type": "standard",
				},
			},
			&models.AWSNodeSpec{
				AMI:              "ami-5731123e",
				AssignPublicIP:   true,
				AvailabilityZone: "eu-west-1",
				InstanceType:     common.StrToPtr("t3.small"),
				SubnetID:         "subnet-53485",
				Tags: map[string]string{
					"foo": "bar",
				},
				VolumeSize: common.IntToInt32Ptr(25),
				VolumeType: common.StrToPtr("standard"),
			},
		},
		{

			[]interface{}{
				map[string]interface{}{},
			},
			&models.AWSNodeSpec{},
		},
		{
			[]interface{}{},
			nil,
		},
	}

	for _, tc := range cases {
		output := metakubeNodeDeploymentExpandAWSSpec(tc.Input)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from expander: mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestExpandOpenstackNodeSpec(t *testing.T) {
	cases := []struct {
		Input          []interface{}
		ExpectedOutput *models.OpenstackNodeSpec
	}{
		{
			[]interface{}{
				map[string]interface{}{
					"flavor":          "tiny",
					"image":           "Ubuntu",
					"use_floating_ip": false,
					"tags": map[string]interface{}{
						"foo": "bar",
					},
					"disk_size": 999,
				},
			},
			&models.OpenstackNodeSpec{
				Flavor:        common.StrToPtr("tiny"),
				Image:         common.StrToPtr("Ubuntu"),
				UseFloatingIP: ptr.To(false),
				Tags: map[string]string{
					"foo": "bar",
				},
				RootDiskSizeGB: ptr.To(int64(999)),
			},
		},
		{

			[]interface{}{
				map[string]interface{}{},
			},
			&models.OpenstackNodeSpec{},
		},
		{
			[]interface{}{},
			nil,
		},
	}

	for _, tc := range cases {
		output := metakubeNodeDeploymentExpandOpenstackSpec(tc.Input)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from expander: mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestExpandAzureNodeSpec(t *testing.T) {
	cases := []struct {
		Input          []interface{}
		ExpectedOutput *models.AzureNodeSpec
	}{
		{
			[]interface{}{
				map[string]interface{}{
					"image_id":         "ImageID",
					"size":             "Size",
					"assign_public_ip": false,
					"disk_size_gb":     int32(1),
					"os_disk_size_gb":  int32(2),
					"tags": map[string]string{
						"tag-k": "tag-v",
					},
					"zones": []string{"Zone-x"},
				},
			},
			&models.AzureNodeSpec{
				ImageID:        "ImageID",
				Size:           common.StrToPtr("Size"),
				AssignPublicIP: false,
				DataDiskSize:   1,
				OSDiskSize:     2,
				Tags: map[string]string{
					"tag-k": "tag-v",
				},
				Zones: []string{"Zone-x"},
			},
		},
		{

			[]interface{}{
				map[string]interface{}{},
			},
			&models.AzureNodeSpec{},
		},
		{
			[]interface{}{},
			nil,
		},
	}

	for _, tc := range cases {
		output := metakubeNodeDeploymentExpandAzureSpec(tc.Input)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from expander: mismatch (-want +got):\n%s", diff)
		}
	}
}
