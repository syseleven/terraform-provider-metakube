package resource_node_deployment

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/syseleven/go-metakube/models"
	"k8s.io/utils/ptr"
)

func TestFrameworkFlattenNodeDeploymentSpec(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		input    *models.NodeDeploymentSpec
		wantNull bool
	}{
		{
			name:     "nil spec",
			input:    nil,
			wantNull: true,
		},
		{
			name: "basic spec with replicas",
			input: &models.NodeDeploymentSpec{
				Replicas: ptr.To(int32(3)),
			},
			wantNull: false,
		},
		{
			name: "autoscaler spec",
			input: &models.NodeDeploymentSpec{
				MinReplicas: ptr.To(int32(1)),
				MaxReplicas: ptr.To(int32(5)),
			},
			wantNull: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, diags := flattenNodeDeploymentSpec(ctx, tt.input)
			if diags.HasError() {
				t.Fatalf("unexpected errors: %v", diags)
			}

			if tt.wantNull {
				if !result.IsNull() {
					t.Errorf("expected null result, got non-null")
				}
				return
			}

			if result.IsNull() {
				t.Error("expected non-null result, got null")
			}
		})
	}
}

func TestFrameworkExpandNodeDeploymentSpec(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		replicas int64
		isCreate bool
		wantSpec *models.NodeDeploymentSpec
	}{
		{
			name:     "basic replicas on create",
			replicas: 3,
			isCreate: true,
			wantSpec: &models.NodeDeploymentSpec{
				Replicas: ptr.To(int32(3)),
			},
		},
		{
			name:     "basic replicas on update",
			replicas: 3,
			isCreate: false,
			wantSpec: &models.NodeDeploymentSpec{
				Replicas: ptr.To(int32(3)),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build a spec model
			specModel := NodeDeploymentSpecModel{
				Replicas:    types.Int64Value(tt.replicas),
				MinReplicas: types.Int64Null(),
				MaxReplicas: types.Int64Null(),
				Template:    types.ListNull(types.ObjectType{AttrTypes: nodeSpecAttrTypes()}),
			}

			// Create list from model
			specObjVal, diags := types.ObjectValueFrom(ctx, nodeDeploymentSpecAttrTypes(), specModel)
			if diags.HasError() {
				t.Fatalf("failed to create object: %v", diags)
			}

			specList, diags := types.ListValue(types.ObjectType{AttrTypes: nodeDeploymentSpecAttrTypes()}, []attr.Value{specObjVal})
			if diags.HasError() {
				t.Fatalf("failed to create list: %v", diags)
			}

			result, diags := expandNodeDeploymentSpec(ctx, specList, tt.isCreate)
			if diags.HasError() {
				t.Fatalf("unexpected errors: %v", diags)
			}

			if result == nil {
				t.Fatal("expected non-nil result")
			}

			// Compare replicas
			if result.Replicas == nil {
				t.Error("expected non-nil replicas")
			} else if *result.Replicas != *tt.wantSpec.Replicas {
				t.Errorf("replicas mismatch: got %d, want %d", *result.Replicas, *tt.wantSpec.Replicas)
			}
		})
	}
}

func TestFlattenAndExpandRoundTrip(t *testing.T) {
	ctx := context.Background()

	// Create a complete spec
	originalSpec := &models.NodeDeploymentSpec{
		Replicas: ptr.To(int32(2)),
		Template: &models.NodeSpec{
			Labels: map[string]string{
				"env":  "test",
				"team": "platform",
			},
			NodeAnnotations: map[string]string{
				"node-anno-key": "node-anno-val",
			},
			MachineAnnotations: map[string]string{
				"a": "b",
			},
			Cloud: &models.NodeCloudSpec{
				Openstack: &models.OpenstackNodeSpec{
					Flavor:                    ptr.To("m1.small"),
					Image:                     ptr.To("Ubuntu 22.04"),
					UseFloatingIP:             ptr.To(true),
					InstanceReadyCheckPeriod:  "5s",
					InstanceReadyCheckTimeout: "120s",
				},
			},
			OperatingSystem: &models.OperatingSystemSpec{
				Ubuntu: &models.UbuntuSpec{
					DistUpgradeOnBoot: false,
				},
			},
			Versions: &models.NodeVersionInfo{
				Kubelet: "1.28.0",
			},
		},
	}

	// Flatten
	flattenedList, diags := flattenNodeDeploymentSpec(ctx, originalSpec)
	if diags.HasError() {
		t.Fatalf("flatten failed: %v", diags)
	}

	// Expand
	expandedSpec, diags := expandNodeDeploymentSpec(ctx, flattenedList, false)
	if diags.HasError() {
		t.Fatalf("expand failed: %v", diags)
	}

	// Compare
	opts := []cmp.Option{
		cmpopts.IgnoreUnexported(models.NodeDeploymentSpec{}),
		cmpopts.IgnoreUnexported(models.NodeSpec{}),
		cmpopts.IgnoreUnexported(models.NodeCloudSpec{}),
		cmpopts.IgnoreUnexported(models.OpenstackNodeSpec{}),
		cmpopts.IgnoreUnexported(models.OperatingSystemSpec{}),
		cmpopts.IgnoreUnexported(models.UbuntuSpec{}),
		cmpopts.IgnoreUnexported(models.NodeVersionInfo{}),
	}

	if diff := cmp.Diff(originalSpec, expandedSpec, opts...); diff != "" {
		t.Errorf("round-trip mismatch (-original +expanded):\n%s", diff)
	}
}

func TestGetCloudProviderFromModel(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name         string
		cloudModel   CloudSpecModel
		wantProvider string
	}{
		{
			name: "AWS",
			cloudModel: CloudSpecModel{
				AWS:       buildMockAWSList(ctx, t),
				OpenStack: types.ListNull(types.ObjectType{AttrTypes: openstackCloudSpecAttrTypes()}),
			},
			wantProvider: "aws",
		},
		{
			name: "OpenStack",
			cloudModel: CloudSpecModel{
				AWS:       types.ListNull(types.ObjectType{AttrTypes: awsCloudSpecAttrTypes()}),
				OpenStack: buildMockOpenStackList(ctx, t),
			},
			wantProvider: "openstack",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := buildMockNodeDeploymentModel(ctx, t, tt.cloudModel)
			provider, diags := getCloudProviderFromModel(ctx, model)
			if diags.HasError() {
				t.Fatalf("unexpected errors: %v", diags)
			}

			if provider != tt.wantProvider {
				t.Errorf("provider mismatch: got %s, want %s", provider, tt.wantProvider)
			}
		})
	}
}

// Helpers

func buildMockAWSList(ctx context.Context, t *testing.T) types.List {
	t.Helper()
	awsModel := AWSCloudSpecModel{
		InstanceType:     types.StringValue("t3.medium"),
		DiskSize:         types.Int64Value(50),
		VolumeType:       types.StringValue("gp2"),
		AvailabilityZone: types.StringValue("us-east-1a"),
		SubnetID:         types.StringValue("subnet-123"),
		AssignPublicIP:   types.BoolValue(true),
		AMI:              types.StringNull(),
		Tags:             types.MapNull(types.StringType),
	}
	objVal, diags := types.ObjectValueFrom(ctx, awsCloudSpecAttrTypes(), awsModel)
	if diags.HasError() {
		t.Fatalf("failed to build AWS object: %v", diags)
	}
	list, diags := types.ListValue(types.ObjectType{AttrTypes: awsCloudSpecAttrTypes()}, []attr.Value{objVal})
	if diags.HasError() {
		t.Fatalf("failed to build AWS list: %v", diags)
	}
	return list
}

func buildMockOpenStackList(ctx context.Context, t *testing.T) types.List {
	t.Helper()
	osModel := OpenStackCloudSpecModel{
		Flavor:                    types.StringValue("m1.small"),
		Image:                     types.StringValue("Ubuntu 22.04"),
		DiskSize:                  types.Int64Null(),
		Tags:                      types.MapNull(types.StringType),
		UseFloatingIP:             types.BoolValue(true),
		InstanceReadyCheckPeriod:  types.StringValue("5s"),
		InstanceReadyCheckTimeout: types.StringValue("120s"),
		ServerGroupID:             types.StringNull(),
	}
	objVal, diags := types.ObjectValueFrom(ctx, openstackCloudSpecAttrTypes(), osModel)
	if diags.HasError() {
		t.Fatalf("failed to build OpenStack object: %v", diags)
	}
	list, diags := types.ListValue(types.ObjectType{AttrTypes: openstackCloudSpecAttrTypes()}, []attr.Value{objVal})
	if diags.HasError() {
		t.Fatalf("failed to build OpenStack list: %v", diags)
	}
	return list
}

func buildMockNodeDeploymentModel(ctx context.Context, t *testing.T, cloudModel CloudSpecModel) *NodeDeploymentModel {
	t.Helper()

	// Build cloud object
	cloudObj, diags := types.ObjectValueFrom(ctx, cloudSpecAttrTypes(), cloudModel)
	if diags.HasError() {
		t.Fatalf("failed to build cloud object: %v", diags)
	}
	cloudList, diags := types.ListValue(types.ObjectType{AttrTypes: cloudSpecAttrTypes()}, []attr.Value{cloudObj})
	if diags.HasError() {
		t.Fatalf("failed to build cloud list: %v", diags)
	}

	// Build node spec
	nodeSpecModel := NodeSpecModel{
		Cloud:              cloudList,
		OperatingSystem:    types.ListNull(types.ObjectType{AttrTypes: operatingSystemAttrTypes()}),
		Versions:           types.ListNull(types.ObjectType{AttrTypes: versionsAttrTypes()}),
		Labels:             types.MapNull(types.StringType),
		AllLabels:          types.MapNull(types.StringType),
		Taints:             types.ListNull(types.ObjectType{AttrTypes: taintAttrTypes()}),
		NodeAnnotations:    types.MapNull(types.StringType),
		MachineAnnotations: types.MapNull(types.StringType),
	}
	nodeSpecObj, diags := types.ObjectValueFrom(ctx, nodeSpecAttrTypes(), nodeSpecModel)
	if diags.HasError() {
		t.Fatalf("failed to build node spec object: %v", diags)
	}
	templateList, diags := types.ListValue(types.ObjectType{AttrTypes: nodeSpecAttrTypes()}, []attr.Value{nodeSpecObj})
	if diags.HasError() {
		t.Fatalf("failed to build template list: %v", diags)
	}

	// Build spec
	specModel := NodeDeploymentSpecModel{
		Replicas:    types.Int64Value(2),
		MinReplicas: types.Int64Null(),
		MaxReplicas: types.Int64Null(),
		Template:    templateList,
	}
	specObj, diags := types.ObjectValueFrom(ctx, nodeDeploymentSpecAttrTypes(), specModel)
	if diags.HasError() {
		t.Fatalf("failed to build spec object: %v", diags)
	}
	specList, diags := types.ListValue(types.ObjectType{AttrTypes: nodeDeploymentSpecAttrTypes()}, []attr.Value{specObj})
	if diags.HasError() {
		t.Fatalf("failed to build spec list: %v", diags)
	}

	return &NodeDeploymentModel{
		ID:                types.StringValue("test-id"),
		ProjectID:         types.StringValue("test-project"),
		ClusterID:         types.StringValue("test-cluster"),
		Name:              types.StringValue("test-nd"),
		Spec:              specList,
		CreationTimestamp: types.StringNull(),
		DeletionTimestamp: types.StringNull(),
	}
}
