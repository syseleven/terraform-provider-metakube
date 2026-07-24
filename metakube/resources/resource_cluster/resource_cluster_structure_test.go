package resource_cluster

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/syseleven/go-metakube/models"
	"k8s.io/utils/ptr"
)

func TestClusterSpecConversion(t *testing.T) {
	ctx := context.Background()
	input := &models.ClusterSpec{
		Version:               "1.32.0",
		EnableUserSSHKeyAgent: ptr.To(true),
		AuditLogging:          &models.AuditLoggingSettings{Enabled: true},
		UpdateWindow:          &models.UpdateWindow{Start: "Tue 02:00", Length: "2h"},
		ClusterNetwork: &models.ClusterNetworkingConfig{
			Pods:     &models.NetworkRanges{CIDRBlocks: []string{"10.0.0.0/16"}},
			Services: &models.NetworkRanges{CIDRBlocks: []string{"10.1.0.0/16"}},
			IPFamily: "IPv4",
		},
		CniPlugin: &models.CNIPluginSettings{Type: "canal"},
		Cloud: &models.CloudSpec{
			DatacenterName: "dc-1",
			Openstack:      &models.OpenstackCloudSpec{Network: "network-id"},
		},
	}

	resourceModel := ClusterModel{}
	diags := metakubeResourceClusterFlattenSpec(ctx, &resourceModel, input)
	if diags.HasError() {
		t.Fatal(diags)
	}
	output := metakubeResourceClusterExpandSpec(ctx, &resourceModel, "dc-1", func(string) bool { return true })

	if output.Version != input.Version {
		t.Fatalf("version mismatch: got %q, want %q", output.Version, input.Version)
	}
	if output.Cloud == nil || output.Cloud.Openstack == nil || output.Cloud.Openstack.Network != "network-id" {
		t.Fatalf("unexpected expanded cloud: %#v", output.Cloud)
	}
	if output.ClusterNetwork == nil || output.ClusterNetwork.Pods.CIDRBlocks[0] != "10.0.0.0/16" {
		t.Fatalf("unexpected expanded network: %#v", output.ClusterNetwork)
	}
}

func TestClusterStateValuesAreExplicitlyReconciled(t *testing.T) {
	ctx := context.Background()
	credentials := objectValue(t, openstackApplicationCredentialsAttrTypes(), OpenstackApplicationCredentialsModel{
		ID:     types.StringValue("credential-id"),
		Secret: types.StringValue("secret"),
	})
	openstack := objectValue(t, openstackCloudSpecAttrTypes(), OpenstackCloudSpecModel{
		UserCredentials:        types.ObjectNull(openstackUserCredentialsAttrTypes()),
		ApplicationCredentials: credentials,
		FloatingIPPool:         types.StringNull(),
		SecurityGroup:          types.StringNull(),
		Network:                types.StringNull(),
		SubnetID:               types.StringNull(),
		SubnetCIDR:             types.StringNull(),
		ServerGroupID:          types.StringValue("server-group"),
	})
	cloud := objectValue(t, clusterCloudSpecAttrTypes(), ClusterCloudSpecModel{Openstack: openstack})
	resourceModel := clusterModel(t, baseClusterSpec(cloud))

	omittedState := clusterOmittedStateValues(ctx, &resourceModel)
	if omittedState.openstack == nil ||
		omittedState.openstack.openstackApplicationCredentialsSecret.ValueString() != "secret" {
		t.Fatalf("credentials were not captured: %#v", omittedState.openstack)
	}

	diags := metakubeResourceClusterFlattenSpec(ctx, &resourceModel, &models.ClusterSpec{
		Cloud: &models.CloudSpec{Openstack: &models.OpenstackCloudSpec{}},
	})
	diags.Append(reconcileClusterOmittedState(ctx, &resourceModel, omittedState)...)
	if diags.HasError() {
		t.Fatal(diags)
	}
	spec, _ := getClusterSpecModel(ctx, &resourceModel)
	flattenedCloud := objectModel[ClusterCloudSpecModel](t, spec.Cloud)
	flattenedOpenstack := objectModel[OpenstackCloudSpecModel](t, flattenedCloud.Openstack)
	flattenedCredentials := objectModel[OpenstackApplicationCredentialsModel](t, flattenedOpenstack.ApplicationCredentials)
	if flattenedCredentials.Secret.ValueString() != "secret" {
		t.Fatalf("credential was not reconciled: %#v", flattenedCredentials)
	}
	if flattenedOpenstack.ServerGroupID.ValueString() != "server-group" {
		t.Fatalf("server group was not reconciled: %#v", flattenedOpenstack.ServerGroupID)
	}
}

func TestBuildClusterPatch(t *testing.T) {
	t.Run("map deletion uses JSON merge patch null", func(t *testing.T) {
		spec := objectValue(t, clusterSpecAttrTypes(), baseClusterSpec(types.ObjectNull(clusterCloudSpecAttrTypes())))
		state := ClusterModel{
			Name:   types.StringValue("cluster"),
			Labels: stringMap(t, map[string]string{"keep": "old", "remove": "old"}),
			Spec:   spec,
		}
		plan := ClusterModel{
			Name:   types.StringValue("cluster"),
			Labels: stringMap(t, map[string]string{"keep": "new"}),
			Spec:   spec,
		}

		patch, diags := buildClusterPatch(context.Background(), plan, state)
		if diags.HasError() {
			t.Fatal(diags)
		}
		want := map[string]any{"labels": map[string]any{"keep": "new", "remove": nil}}
		if diff := cmp.Diff(want, patch); diff != "" {
			t.Fatalf("patch mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("false values remain explicit", func(t *testing.T) {
		stateSpec := baseClusterSpec(types.ObjectNull(clusterCloudSpecAttrTypes()))
		stateSpec.PodNodeSelector = types.BoolValue(true)
		planSpec := stateSpec
		planSpec.PodNodeSelector = types.BoolValue(false)
		state := clusterModel(t, stateSpec)
		plan := clusterModel(t, planSpec)

		patch, diags := buildClusterPatch(context.Background(), plan, state)
		if diags.HasError() {
			t.Fatal(diags)
		}
		specPatch := patch["spec"].(map[string]any)
		if got := specPatch["usePodNodeSelectorAdmissionPlugin"]; got != false {
			t.Fatalf("expected explicit false, got %#v", got)
		}
	})

	t.Run("unknown values are not interpreted as deletion", func(t *testing.T) {
		stateSpec := baseClusterSpec(types.ObjectNull(clusterCloudSpecAttrTypes()))
		planSpec := stateSpec
		planSpec.Version = types.StringUnknown()
		patch, diags := buildClusterPatch(
			context.Background(),
			clusterModel(t, planSpec),
			clusterModel(t, stateSpec),
		)
		if diags.HasError() {
			t.Fatal(diags)
		}
		if len(patch) != 0 {
			t.Fatalf("expected empty patch, got %#v", patch)
		}
	})
}

func TestUpgradeClusterLegacyNestedSpecState(t *testing.T) {
	rawState := map[string]any{
		"spec": []any{map[string]any{
			"cni_plugin": []any{map[string]any{"type": "canal"}},
			"cloud": []any{map[string]any{
				"aws":       []any{map[string]any{"access_key": "legacy"}},
				"openstack": []any{map[string]any{"application_credentials": []any{map[string]any{"id": "id"}}}},
			}},
		}},
	}

	upgradeClusterLegacyNestedSpecState(rawState)
	spec := rawState["spec"].(map[string]any)
	cloud := spec["cloud"].(map[string]any)
	if _, exists := cloud["aws"]; exists {
		t.Fatalf("legacy AWS state was retained: %#v", cloud)
	}
	if _, ok := spec["cni_plugin"].(map[string]any); !ok {
		t.Fatalf("CNI plugin was not upgraded: %#v", spec["cni_plugin"])
	}
	openstack := cloud["openstack"].(map[string]any)
	if _, ok := openstack["application_credentials"].(map[string]any); !ok {
		t.Fatalf("credentials were not upgraded: %#v", openstack)
	}
}

func baseClusterSpec(cloud types.Object) ClusterSpecModel {
	return ClusterSpecModel{
		Version:           types.StringValue("1.32.0"),
		EnableSSHAgent:    types.BoolValue(true),
		AuditLogging:      types.BoolValue(false),
		PodSecurityPolicy: types.BoolValue(false),
		PodNodeSelector:   types.BoolValue(false),
		ServicesCIDR:      types.StringNull(),
		PodsCIDR:          types.StringNull(),
		IPFamily:          types.StringValue("IPv4"),
		UpdateWindow:      types.ObjectNull(updateWindowAttrTypes()),
		CNIPlugin:         types.ObjectNull(cniPluginAttrTypes()),
		Cloud:             cloud,
		SyselevenAuth:     types.ObjectNull(syselevenAuthAttrTypes()),
	}
}

func clusterModel(t *testing.T, spec ClusterSpecModel) ClusterModel {
	t.Helper()
	return ClusterModel{
		Name:   types.StringValue("cluster"),
		Labels: stringMap(t, nil),
		Spec:   objectValue(t, clusterSpecAttrTypes(), spec),
	}
}

func objectValue[T any](t *testing.T, attrTypes map[string]attr.Type, model T) types.Object {
	t.Helper()
	value, diags := types.ObjectValueFrom(context.Background(), attrTypes, model)
	if diags.HasError() {
		t.Fatal(diags)
	}
	return value
}

func objectModel[T any](t *testing.T, value types.Object) T {
	t.Helper()
	var model T
	diags := value.As(context.Background(), &model, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		t.Fatal(diags)
	}
	return model
}

func stringMap(t *testing.T, values map[string]string) types.Map {
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
