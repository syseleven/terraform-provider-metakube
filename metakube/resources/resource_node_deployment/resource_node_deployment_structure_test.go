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

func TestMarshalSpecToMapFWIncludesFalseOperatingSystemBooleans(t *testing.T) {
	tests := []struct {
		name      string
		os        *models.OperatingSystemSpec
		osKey     string
		fieldKey  string
		wantValue bool
	}{
		{
			name: "ubuntu dist upgrade on boot",
			os: &models.OperatingSystemSpec{
				Ubuntu: &models.UbuntuSpec{
					DistUpgradeOnBoot: false,
				},
			},
			osKey:     "ubuntu",
			fieldKey:  "distUpgradeOnBoot",
			wantValue: false,
		},
		{
			name: "flatcar disable auto update",
			os: &models.OperatingSystemSpec{
				Flatcar: &models.FlatcarSpec{
					DisableAutoUpdate: false,
				},
			},
			osKey:     "flatcar",
			fieldKey:  "disableAutoUpdate",
			wantValue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patch, err := nodeDeploymentSpecPatchBody(&models.NodeDeploymentSpec{
				Replicas: ptr.To(int32(1)),
				Template: &models.NodeSpec{
					OperatingSystem: tt.os,
				},
			})
			if err != nil {
				t.Fatalf("nodeDeploymentSpecPatchBody failed: %v", err)
			}

			template, ok := patch["template"].(map[string]interface{})
			if !ok {
				t.Fatalf("expected template map, got %#v", patch["template"])
			}
			operatingSystem, ok := template["operatingSystem"].(map[string]interface{})
			if !ok {
				t.Fatalf("expected operatingSystem map, got %#v", template["operatingSystem"])
			}
			osMap, ok := operatingSystem[tt.osKey].(map[string]interface{})
			if !ok {
				t.Fatalf("expected %s map, got %#v", tt.osKey, operatingSystem[tt.osKey])
			}
			if got, ok := osMap[tt.fieldKey].(bool); !ok || got != tt.wantValue {
				t.Fatalf("expected %s.%s=%v, got %#v", tt.osKey, tt.fieldKey, tt.wantValue, osMap[tt.fieldKey])
			}
		})
	}
}

func TestBuildPatchWithDeletions(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name  string
		setup func(t *testing.T) (config, plan, state *NodeDeploymentModel)
		check func(t *testing.T, patch map[string]interface{})
	}{
		{
			name: "removes openstack tags",
			setup: func(t *testing.T) (config, plan, state *NodeDeploymentModel) {
				plan = buildMockNodeDeploymentModel(ctx, t, CloudSpecModel{
					OpenStack: buildMockOpenStackListWithTags(ctx, t, map[string]string{"updated-user-tag": "changed"}),
				})
				state = buildMockNodeDeploymentModel(ctx, t, CloudSpecModel{
					OpenStack: buildMockOpenStackListWithTags(ctx, t, map[string]string{"user-tag": "kept"}),
				})
				return plan, plan, state
			},
			check: func(t *testing.T, patch map[string]interface{}) {
				tagsPatch := cloudTagsPatch(t, patch, "openstack")
				if got := tagsPatch["updated-user-tag"]; got != "changed" {
					t.Fatalf("expected updated tag in patch, got %#v", tagsPatch)
				}
				if got, ok := tagsPatch["user-tag"]; !ok || got != nil {
					t.Fatalf("expected removed tag to be patched as null, got %#v", tagsPatch)
				}
			},
		},
		{
			name: "clears omitted openstack serverGroupID",
			setup: func(t *testing.T) (config, plan, state *NodeDeploymentModel) {
				config = buildMockNodeDeploymentModel(ctx, t, CloudSpecModel{
					OpenStack: buildMockOpenStackListWithServerGroupID(ctx, t, types.StringNull()),
				})
				plan = buildMockNodeDeploymentModel(ctx, t, CloudSpecModel{
					OpenStack: buildMockOpenStackListWithServerGroupID(ctx, t, types.StringUnknown()),
				})
				state = buildMockNodeDeploymentModel(ctx, t, CloudSpecModel{
					OpenStack: buildMockOpenStackListWithServerGroupID(ctx, t, types.StringValue("old-server-group-id")),
				})
				return config, plan, state
			},
			check: func(t *testing.T, patch map[string]interface{}) {
				openStackPatch := cloudProviderPatch(t, patch, "openstack")
				if got, ok := openStackPatch["serverGroupID"]; !ok || got != nil {
					t.Fatalf("expected omitted serverGroupID to be patched as null, got %#v", openStackPatch)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, plan, state := tt.setup(t)
			patch := buildNodeDeploymentPatch(t, ctx, config, plan, state)
			tt.check(t, patch)
		})
	}
}

func TestFlattenOpenStackCloudSpecFiltersSystemTags(t *testing.T) {
	ctx := context.Background()

	result, diags := flattenOpenStackCloudSpec(ctx, &models.OpenstackNodeSpec{
		Flavor: ptr.To("m1.small"),
		Image:  ptr.To("Ubuntu 22.04"),
		Tags: map[string]string{
			"user-tag":         "kept",
			"metakube-cluster": "cluster-id",
			"system-cluster":   "cluster-name",
			"system-project":   "project-name",
			"system/project":   "project-id",
		},
	})
	if diags.HasError() {
		t.Fatalf("flatten failed: %v", diags)
	}

	var flattened []OpenStackCloudSpecModel
	diags = result.ElementsAs(ctx, &flattened, false)
	if diags.HasError() {
		t.Fatalf("failed to read flattened model: %v", diags)
	}
	if len(flattened) != 1 {
		t.Fatalf("expected one openstack model, got %d", len(flattened))
	}

	tags := flattened[0].Tags.Elements()
	if got := tags["user-tag"].(types.String).ValueString(); got != "kept" {
		t.Fatalf("unexpected user tag value: got %q, want %q", got, "kept")
	}
	for _, key := range []string{"metakube-cluster", "system-cluster", "system-project", "system/project"} {
		if _, ok := tags[key]; ok {
			t.Fatalf("expected system tag %q to be filtered from user tags", key)
		}
	}
}

func TestFlattenOpenStackCloudSpecSetsTagsNullWhenOnlySystemTagsExist(t *testing.T) {
	ctx := context.Background()

	result, diags := flattenOpenStackCloudSpec(ctx, &models.OpenstackNodeSpec{
		Flavor: ptr.To("m1.small"),
		Image:  ptr.To("Ubuntu 22.04"),
		Tags: map[string]string{
			"metakube-cluster": "cluster-id",
			"system-cluster":   "cluster-name",
			"system-project":   "project-name",
		},
	})
	if diags.HasError() {
		t.Fatalf("flatten failed: %v", diags)
	}

	var flattened []OpenStackCloudSpecModel
	diags = result.ElementsAs(ctx, &flattened, false)
	if diags.HasError() {
		t.Fatalf("failed to read flattened model: %v", diags)
	}
	if len(flattened) != 1 {
		t.Fatalf("expected one openstack model, got %d", len(flattened))
	}
	if !flattened[0].Tags.IsNull() {
		t.Fatalf("expected tags to be null when only system tags are returned, got %v", flattened[0].Tags)
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
			name: "OpenStack",
			cloudModel: CloudSpecModel{
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

func buildMockOpenStackListWithTags(ctx context.Context, t *testing.T, tags map[string]string) types.List {
	t.Helper()
	osModel := OpenStackCloudSpecModel{
		Flavor:                    types.StringValue("m1.small"),
		Image:                     types.StringValue("Ubuntu 22.04"),
		DiskSize:                  types.Int64Null(),
		Tags:                      stringMapValue(t, tags),
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

func buildMockOpenStackListWithServerGroupID(ctx context.Context, t *testing.T, serverGroupID types.String) types.List {
	t.Helper()
	osModel := OpenStackCloudSpecModel{
		Flavor:                    types.StringValue("m1.small"),
		Image:                     types.StringValue("Ubuntu 22.04"),
		DiskSize:                  types.Int64Null(),
		Tags:                      types.MapNull(types.StringType),
		UseFloatingIP:             types.BoolValue(true),
		InstanceReadyCheckPeriod:  types.StringValue("5s"),
		InstanceReadyCheckTimeout: types.StringValue("120s"),
		ServerGroupID:             serverGroupID,
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

func stringMapValue(t *testing.T, values map[string]string) types.Map {
	t.Helper()
	if values == nil {
		return types.MapNull(types.StringType)
	}

	elements := make(map[string]attr.Value, len(values))
	for key, value := range values {
		elements[key] = types.StringValue(value)
	}

	result, diags := types.MapValue(types.StringType, elements)
	if diags.HasError() {
		t.Fatalf("failed to build string map: %v", diags)
	}
	return result
}

func buildNodeDeploymentPatch(t *testing.T, ctx context.Context, config, plan, state *NodeDeploymentModel) map[string]interface{} {
	t.Helper()

	spec, diags := expandNodeDeploymentSpec(ctx, plan.Spec, false)
	if diags.HasError() {
		t.Fatalf("failed to expand plan spec: %v", diags)
	}

	patch, err := (&nodeDeploymentResource{}).buildPatchWithDeletions(config, plan, state, &models.NodeDeployment{Spec: spec})
	if err != nil {
		t.Fatalf("failed to build patch: %v", err)
	}

	return patch
}

func cloudTagsPatch(t *testing.T, patch map[string]interface{}, provider string) map[string]interface{} {
	t.Helper()

	providerPatch := cloudProviderPatch(t, patch, provider)
	tagsPatch, ok := providerPatch["tags"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected tags patch map, got %#v", providerPatch["tags"])
	}
	return tagsPatch
}

func cloudProviderPatch(t *testing.T, patch map[string]interface{}, provider string) map[string]interface{} {
	t.Helper()

	specPatch, ok := patch["spec"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected spec patch map, got %#v", patch["spec"])
	}
	templatePatch, ok := specPatch["template"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected template patch map, got %#v", specPatch["template"])
	}
	cloudPatch, ok := templatePatch["cloud"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected cloud patch map, got %#v", templatePatch["cloud"])
	}
	providerPatch, ok := cloudPatch[provider].(map[string]interface{})
	if !ok {
		t.Fatalf("expected %s patch map, got %#v", provider, cloudPatch[provider])
	}
	return providerPatch
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

func TestUpgradeNodeDeploymentLegacyUnsupportedCloudState_RemovesUnsupportedClouds(t *testing.T) {
	rawState := map[string]any{
		"spec": []any{
			map[string]any{
				"template": []any{
					map[string]any{
						"cloud": []any{
							map[string]any{
								"openstack": []any{},
								"azure": []any{
									map[string]any{
										"size": "legacy",
									},
								},
								"aws": []any{
									map[string]any{
										"instance_type": "legacy",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	upgradeNodeDeploymentLegacyUnsupportedCloudState(rawState)

	specMap := rawState["spec"].([]any)[0].(map[string]any)
	templateMap := specMap["template"].([]any)[0].(map[string]any)
	cloudMap := templateMap["cloud"].([]any)[0].(map[string]any)
	if _, ok := cloudMap["azure"]; ok {
		t.Fatalf("expected legacy cloud.azure to be removed, got: %#v", cloudMap["azure"])
	}
	if _, ok := cloudMap["aws"]; ok {
		t.Fatalf("expected legacy cloud.aws to be removed, got: %#v", cloudMap["aws"])
	}
}
