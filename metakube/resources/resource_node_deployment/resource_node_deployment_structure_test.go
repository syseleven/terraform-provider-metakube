package resource_node_deployment

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/syseleven/go-metakube/models"
	"k8s.io/utils/ptr"
)

func TestNodeDeploymentSpecConversion(t *testing.T) {
	ctx := context.Background()
	input := &models.NodeDeploymentSpec{
		Replicas: ptr.To(int32(2)),
		Template: &models.NodeSpec{
			Labels:             map[string]string{"team": "platform"},
			NodeAnnotations:    map[string]string{"node": "value"},
			MachineAnnotations: map[string]string{"machine": "value"},
			Cloud: &models.NodeCloudSpec{Openstack: &models.OpenstackNodeSpec{
				Flavor:                    ptr.To("m1.small"),
				Image:                     ptr.To("ubuntu"),
				Tags:                      map[string]string{"role": "worker"},
				UseFloatingIP:             ptr.To(false),
				InstanceReadyCheckPeriod:  "5s",
				InstanceReadyCheckTimeout: "120s",
			}},
			OperatingSystem: &models.OperatingSystemSpec{
				Ubuntu: &models.UbuntuSpec{DistUpgradeOnBoot: false},
			},
			Versions: &models.NodeVersionInfo{Kubelet: "1.32.0"},
		},
	}

	value, diags := flattenNodeDeploymentSpec(ctx, input)
	if diags.HasError() {
		t.Fatal(diags)
	}
	output, diags := expandNodeDeploymentSpec(ctx, value, false)
	if diags.HasError() {
		t.Fatal(diags)
	}

	if output.Replicas == nil || *output.Replicas != 2 {
		t.Fatalf("unexpected replicas: %#v", output.Replicas)
	}
	if diff := cmp.Diff(input.Template.Labels, output.Template.Labels); diff != "" {
		t.Fatalf("labels mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(input.Template.MachineAnnotations, output.Template.MachineAnnotations); diff != "" {
		t.Fatalf("annotations mismatch (-want +got):\n%s", diff)
	}
	if output.Template.Cloud.Openstack.UseFloatingIP == nil || *output.Template.Cloud.Openstack.UseFloatingIP {
		t.Fatalf("explicit false was lost: %#v", output.Template.Cloud.Openstack.UseFloatingIP)
	}
	if output.Template.OperatingSystem.Ubuntu == nil || output.Template.OperatingSystem.Ubuntu.DistUpgradeOnBoot {
		t.Fatalf("Ubuntu boolean was lost: %#v", output.Template.OperatingSystem)
	}
}

func TestNodeDeploymentEmptyMapsAreCanonical(t *testing.T) {
	value, diags := flattenNodeDeploymentSpec(context.Background(), &models.NodeDeploymentSpec{
		Template: &models.NodeSpec{},
	})
	if diags.HasError() {
		t.Fatal(diags)
	}
	spec := mustModel[NodeDeploymentSpecModel](t, value)
	template := mustModel[NodeSpecModel](t, spec.Template)
	for name, value := range map[string]types.Map{
		"labels":              template.Labels,
		"all_labels":          template.AllLabels,
		"node_annotations":    template.NodeAnnotations,
		"machine_annotations": template.MachineAnnotations,
	} {
		if value.IsNull() || value.IsUnknown() || len(value.Elements()) != 0 {
			t.Fatalf("%s is not a known empty map: %#v", name, value)
		}
	}
}

func TestExpandStringMapRejectsUnknownElements(t *testing.T) {
	value := types.MapValueMust(types.StringType, map[string]attr.Value{
		"unknown": types.StringUnknown(),
	})
	if _, diags := expandStringMap(context.Background(), value); !diags.HasError() {
		t.Fatal("expected unknown map element to produce diagnostics")
	}
}

func TestBuildNodeDeploymentPatch(t *testing.T) {
	t.Run("removed annotations and tags become null entries", func(t *testing.T) {
		state := nodeDeploymentModel(t,
			map[string]string{"annotation": "old"},
			map[string]string{"tag": "old"},
			types.StringValue("server-group"),
		)
		config := nodeDeploymentModel(t, map[string]string{}, map[string]string{}, types.StringNull())
		plan := nodeDeploymentModel(t, map[string]string{}, map[string]string{}, types.StringUnknown())

		patch, diags := buildNodeDeploymentPatch(context.Background(), config, plan, state)
		if diags.HasError() {
			t.Fatal(diags)
		}
		spec := patch["spec"].(map[string]any)
		template := spec["template"].(map[string]any)
		annotations := template["machine_annotations"].(map[string]any)
		if value, exists := annotations["annotation"]; !exists || value != nil {
			t.Fatalf("annotation deletion missing: %#v", annotations)
		}
		openstack := template["cloud"].(map[string]any)["openstack"].(map[string]any)
		tags := openstack["tags"].(map[string]any)
		if value, exists := tags["tag"]; !exists || value != nil {
			t.Fatalf("tag deletion missing: %#v", tags)
		}
		if value, exists := openstack["serverGroupID"]; !exists || value != nil {
			t.Fatalf("server group deletion missing: %#v", openstack)
		}
	})

	t.Run("unknown values are not deletions", func(t *testing.T) {
		state := nodeDeploymentModel(t, nil, nil, types.StringValue("server-group"))
		config := nodeDeploymentModel(t, nil, nil, types.StringUnknown())
		plan := config

		patch, diags := buildNodeDeploymentPatch(context.Background(), config, plan, state)
		if diags.HasError() {
			t.Fatal(diags)
		}
		if len(patch) != 0 {
			t.Fatalf("expected empty patch, got %#v", patch)
		}
	})
}

func TestNodeDeploymentSchemaUsesNestedAttributes(t *testing.T) {
	resourceSchema := NodeDeploymentSchema(context.Background())
	spec, ok := resourceSchema.Attributes["spec"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("spec is %T, want SingleNestedAttribute", resourceSchema.Attributes["spec"])
	}
	template, ok := spec.Attributes["template"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("template is %T, want SingleNestedAttribute", spec.Attributes["template"])
	}
	cloud, ok := template.Attributes["cloud"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("cloud is %T, want SingleNestedAttribute", template.Attributes["cloud"])
	}
	if _, ok := cloud.Attributes["openstack"].(schema.SingleNestedAttribute); !ok {
		t.Fatalf("openstack is %T, want SingleNestedAttribute", cloud.Attributes["openstack"])
	}
	for _, name := range []string{"labels", "node_annotations", "machine_annotations"} {
		attribute := template.Attributes[name].(schema.MapAttribute)
		if !attribute.Optional || !attribute.Computed || attribute.Default == nil {
			t.Fatalf("%s must be optional+computed when using an empty default", name)
		}
	}
}

func TestUpgradeNodeDeploymentLegacyState(t *testing.T) {
	rawState := map[string]any{
		"spec": []any{map[string]any{
			"template": []any{map[string]any{
				"cloud": []any{map[string]any{
					"aws":       []any{map[string]any{"instance_type": "legacy"}},
					"openstack": []any{map[string]any{"flavor": "m1.small"}},
				}},
				"operating_system": []any{map[string]any{
					"ubuntu": []any{map[string]any{"dist_upgrade_on_boot": false}},
				}},
				"versions": []any{map[string]any{"kubelet": "1.32.0"}},
			}},
		}},
	}

	upgradeNodeDeploymentLegacyState(rawState)
	spec := rawState["spec"].(map[string]any)
	template := spec["template"].(map[string]any)
	cloud := template["cloud"].(map[string]any)
	if _, exists := cloud["aws"]; exists {
		t.Fatalf("legacy AWS state was retained: %#v", cloud)
	}
	if _, ok := cloud["openstack"].(map[string]any); !ok {
		t.Fatalf("OpenStack was not upgraded: %#v", cloud)
	}
	operatingSystem := template["operating_system"].(map[string]any)
	if _, ok := operatingSystem["ubuntu"].(map[string]any); !ok {
		t.Fatalf("Ubuntu was not upgraded: %#v", operatingSystem)
	}
	if _, ok := template["versions"].(map[string]any); !ok {
		t.Fatalf("versions were not upgraded: %#v", template["versions"])
	}
}

func nodeDeploymentModel(
	t *testing.T,
	machineAnnotations map[string]string,
	tags map[string]string,
	serverGroupID types.String,
) NodeDeploymentModel {
	t.Helper()
	openstack := mustObject(t, openstackCloudSpecAttrTypes(), OpenStackCloudSpecModel{
		Flavor:                    types.StringValue("m1.small"),
		Image:                     types.StringValue("ubuntu"),
		DiskSize:                  types.Int64Null(),
		Tags:                      mustMap(t, tags),
		UseFloatingIP:             types.BoolValue(true),
		InstanceReadyCheckPeriod:  types.StringValue("5s"),
		InstanceReadyCheckTimeout: types.StringValue("120s"),
		ServerGroupID:             serverGroupID,
	})
	cloud := mustObject(t, cloudSpecAttrTypes(), CloudSpecModel{OpenStack: openstack})
	operatingSystem := mustObject(t, operatingSystemAttrTypes(), OperatingSystemModel{
		Ubuntu:  types.ObjectNull(ubuntuAttrTypes()),
		Flatcar: types.ObjectNull(flatcarAttrTypes()),
	})
	template := mustObject(t, nodeSpecAttrTypes(), NodeSpecModel{
		Cloud:              cloud,
		OperatingSystem:    operatingSystem,
		Versions:           types.ObjectNull(versionsAttrTypes()),
		Labels:             mustMap(t, nil),
		AllLabels:          mustMap(t, nil),
		Taints:             types.ListNull(types.ObjectType{AttrTypes: taintAttrTypes()}),
		NodeAnnotations:    mustMap(t, nil),
		MachineAnnotations: mustMap(t, machineAnnotations),
	})
	spec := mustObject(t, nodeDeploymentSpecAttrTypes(), NodeDeploymentSpecModel{
		Replicas:    types.Int64Value(1),
		MinReplicas: types.Int64Null(),
		MaxReplicas: types.Int64Null(),
		Template:    template,
	})
	return NodeDeploymentModel{Spec: spec}
}

func mustObject[T any](t *testing.T, attrTypes map[string]attr.Type, model T) types.Object {
	t.Helper()
	value, diags := types.ObjectValueFrom(context.Background(), attrTypes, model)
	if diags.HasError() {
		t.Fatal(diags)
	}
	return value
}

func mustModel[T any](t *testing.T, value types.Object) T {
	t.Helper()
	var model T
	diags := value.As(context.Background(), &model, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		t.Fatal(diags)
	}
	return model
}

func mustMap(t *testing.T, values map[string]string) types.Map {
	t.Helper()
	elements := make(map[string]attr.Value, len(values))
	for key, value := range values {
		elements[key] = types.StringValue(value)
	}
	result, diags := types.MapValue(types.StringType, elements)
	if diags.HasError() {
		t.Fatal(diags)
	}
	return result
}
