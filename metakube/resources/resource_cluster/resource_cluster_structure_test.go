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
			Openstack: &models.OpenstackCloudSpec{
				Domain:            "Default",
				Network:           "network-id",
				SubnetV6ID:        "subnet-v6",
				SubnetV6CIDR:      "2001:db8:1::/64",
				CreatePodSubnetV6: true,
				PodSubnetV6ID:     "pod-subnet-v6",
			},
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
	if diff := cmp.Diff(input.Cloud.Openstack, output.Cloud.Openstack); diff != "" {
		t.Errorf("OpenStack conversion mismatch (-want +got):\n%s", diff)
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
		SubnetV6ID:             types.StringValue("subnet-v6"),
		SubnetV6CIDR:           types.StringValue("2001:db8:1::/64"),
		PodSubnetV6ID:          types.StringValue("pod-subnet-v6"),
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
	if got := flattenedOpenstack.SubnetV6ID; got.ValueString() != "subnet-v6" {
		t.Errorf("reconcileClusterOmittedState(subnet_v6_id=subnet-v6) = %v, want subnet-v6", got)
	}
	if got := flattenedOpenstack.SubnetV6CIDR; got.ValueString() != "2001:db8:1::/64" {
		t.Errorf("reconcileClusterOmittedState(subnet_v6_cidr=2001:db8:1::/64) = %v, want 2001:db8:1::/64", got)
	}
	if got := flattenedOpenstack.PodSubnetV6ID; got.ValueString() != "pod-subnet-v6" {
		t.Errorf("reconcileClusterOmittedState(pod_subnet_v6_id=pod-subnet-v6) = %v, want pod-subnet-v6", got)
	}
}

func TestClusterIPv6StateReconciliation(t *testing.T) {
	ctx := context.Background()
	for _, tt := range []struct {
		name  string
		prior types.String
		api   string
		want  types.String
	}{
		{name: "API takes precedence", prior: types.StringValue("prior"), api: "reported", want: types.StringValue("reported")},
		{name: "unknown state is not restored", prior: types.StringUnknown(), want: types.StringNull()},
		{name: "null state is not restored", prior: types.StringNull(), want: types.StringNull()},
		{name: "empty state is not restored", prior: types.StringValue(""), want: types.StringNull()},
	} {
		t.Run(tt.name, func(t *testing.T) {
			openstack := objectValue(t, openstackCloudSpecAttrTypes(), OpenstackCloudSpecModel{
				SubnetV6ID:             tt.prior,
				SubnetV6CIDR:           tt.prior,
				PodSubnetV6ID:          tt.prior,
				UserCredentials:        types.ObjectNull(openstackUserCredentialsAttrTypes()),
				ApplicationCredentials: types.ObjectNull(openstackApplicationCredentialsAttrTypes()),
			})
			cloud := objectValue(t, clusterCloudSpecAttrTypes(), ClusterCloudSpecModel{Openstack: openstack})
			model := clusterModel(t, baseClusterSpec(cloud))
			omitted := clusterOmittedStateValues(ctx, &model)
			diags := metakubeResourceClusterFlattenSpec(ctx, &model, &models.ClusterSpec{
				Cloud: &models.CloudSpec{Openstack: &models.OpenstackCloudSpec{
					SubnetV6ID:    tt.api,
					SubnetV6CIDR:  tt.api,
					PodSubnetV6ID: tt.api,
				}},
			})
			diags.Append(reconcileClusterOmittedState(ctx, &model, omitted)...)
			if diags.HasError() {
				t.Fatalf("reconcileClusterOmittedState(prior=%v, API=%q) = %v, want no diagnostics", tt.prior, tt.api, diags)
			}
			spec := objectModel[ClusterSpecModel](t, model.Spec)
			gotCloud := objectModel[ClusterCloudSpecModel](t, spec.Cloud)
			got := objectModel[OpenstackCloudSpecModel](t, gotCloud.Openstack)
			if !got.SubnetV6ID.Equal(tt.want) || !got.SubnetV6CIDR.Equal(tt.want) || !got.PodSubnetV6ID.Equal(tt.want) {
				t.Errorf("reconcileClusterOmittedState(prior=%v, API=%q) = (%v, %v, %v), want %v for all IPv6 strings", tt.prior, tt.api, got.SubnetV6ID, got.SubnetV6CIDR, got.PodSubnetV6ID, tt.want)
			}
		})
	}
}

func TestPatchOpenstackIPv6(t *testing.T) {
	ctx := context.Background()
	state := OpenstackCloudSpecModel{
		SubnetV6ID:             types.StringValue("old-node-subnet"),
		SubnetV6CIDR:           types.StringValue("2001:db8:1::/64"),
		CreatePodSubnetV6:      types.BoolValue(true),
		PodSubnetV6ID:          types.StringValue("old-pod-subnet"),
		UserCredentials:        types.ObjectNull(openstackUserCredentialsAttrTypes()),
		ApplicationCredentials: types.ObjectNull(openstackApplicationCredentialsAttrTypes()),
	}
	for _, tt := range []struct {
		name              string
		subnetV6ID        types.String
		subnetV6CIDR      types.String
		createPodSubnetV6 types.Bool
		podSubnetV6ID     types.String
		want              map[string]any
	}{
		{
			name:              "changes include explicit false",
			subnetV6ID:        types.StringValue("new-node-subnet"),
			subnetV6CIDR:      types.StringValue("2001:db8:2::/64"),
			createPodSubnetV6: types.BoolValue(false),
			podSubnetV6ID:     types.StringValue("new-pod-subnet"),
			want: map[string]any{
				"subnetV6ID": "new-node-subnet", "subnetV6CIDR": "2001:db8:2::/64",
				"createPodSubnetV6": false, "podSubnetV6ID": "new-pod-subnet",
			},
		},
		{
			name:              "null deletes prior values",
			subnetV6ID:        types.StringNull(),
			subnetV6CIDR:      types.StringNull(),
			createPodSubnetV6: types.BoolNull(),
			podSubnetV6ID:     types.StringNull(),
			want: map[string]any{
				"subnetV6ID": nil, "subnetV6CIDR": nil,
				"createPodSubnetV6": nil, "podSubnetV6ID": nil,
			},
		},
		{
			name:              "unknown values are omitted",
			subnetV6ID:        types.StringUnknown(),
			subnetV6CIDR:      types.StringUnknown(),
			createPodSubnetV6: types.BoolUnknown(),
			podSubnetV6ID:     types.StringUnknown(),
			want:              map[string]any{},
		},
		{
			name:              "unchanged values are omitted",
			subnetV6ID:        state.SubnetV6ID,
			subnetV6CIDR:      state.SubnetV6CIDR,
			createPodSubnetV6: state.CreatePodSubnetV6,
			podSubnetV6ID:     state.PodSubnetV6ID,
			want:              map[string]any{},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			plan := state
			plan.SubnetV6ID = tt.subnetV6ID
			plan.SubnetV6CIDR = tt.subnetV6CIDR
			plan.CreatePodSubnetV6 = tt.createPodSubnetV6
			plan.PodSubnetV6ID = tt.podSubnetV6ID
			got, diags := patchOpenstack(ctx, objectValue(t, openstackCloudSpecAttrTypes(), plan), objectValue(t, openstackCloudSpecAttrTypes(), state))
			if diags.HasError() {
				t.Fatalf("patchOpenstack(%s) = %v, want no diagnostics", tt.name, diags)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("patchOpenstack(%s) mismatch (-want +got):\n%s", tt.name, diff)
			}
		})
	}
}

func TestCiliumBPFLbSockHostnsOnlyConversion(t *testing.T) {
	ctx := context.Background()
	input := &models.ClusterSpec{
		Version: "1.32.0",
		CniPlugin: &models.CNIPluginSettings{
			Type:   "cilium",
			Cilium: &models.CiliumCNISettings{BpfLbSockHostnsOnly: true},
		},
	}

	resourceModel := ClusterModel{}
	diags := metakubeResourceClusterFlattenSpec(ctx, &resourceModel, input)
	if diags.HasError() {
		t.Fatal(diags)
	}

	spec, ok := getClusterSpecModel(ctx, &resourceModel)
	if !ok {
		t.Fatal("spec was not flattened")
	}
	cni := objectModel[CNIPluginModel](t, spec.CNIPlugin)
	cilium := objectModel[CiliumModel](t, cni.Cilium)
	if !cilium.BPFLbSockHostnsOnly.ValueBool() {
		t.Fatalf("expected bpf_lb_sock_hostns_only=true after flatten, got %#v", cilium)
	}

	output := metakubeResourceClusterExpandSpec(ctx, &resourceModel, "dc-1", func(string) bool { return true })
	if output.CniPlugin == nil || output.CniPlugin.Cilium == nil {
		t.Fatalf("unexpected expanded cni plugin: %#v", output.CniPlugin)
	}
	if !output.CniPlugin.Cilium.BpfLbSockHostnsOnly {
		t.Fatalf("expected expanded bpfLbSockHostnsOnly=true, got %#v", output.CniPlugin.Cilium)
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

	t.Run("cilium bpf_lb_sock_hostns_only change is patched", func(t *testing.T) {
		stateSpec := baseClusterSpec(types.ObjectNull(clusterCloudSpecAttrTypes()))
		stateSpec.CNIPlugin = cniPluginCiliumValue(t, false)
		planSpec := baseClusterSpec(types.ObjectNull(clusterCloudSpecAttrTypes()))
		planSpec.CNIPlugin = cniPluginCiliumValue(t, true)
		state := clusterModel(t, stateSpec)
		plan := clusterModel(t, planSpec)

		patch, diags := buildClusterPatch(context.Background(), plan, state)
		if diags.HasError() {
			t.Fatal(diags)
		}
		specPatch := patch["spec"].(map[string]any)
		cniPatch := specPatch["cniPlugin"].(map[string]any)
		ciliumPatch := cniPatch["cilium"].(map[string]any)
		if got := ciliumPatch["bpfLbSockHostnsOnly"]; got != true {
			t.Fatalf("expected explicit true, got %#v", ciliumPatch)
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

func cniPluginCiliumValue(t *testing.T, bpfLbSockHostnsOnly bool) types.Object {
	t.Helper()
	cilium := objectValue(t, ciliumAttrTypes(), CiliumModel{
		Clustermesh:         types.ObjectNull(ciliumClustermeshAttrTypes()),
		EnableHubble:        types.BoolValue(false),
		EnableL7Proxy:       types.BoolValue(false),
		BPFLbSockHostnsOnly: types.BoolValue(bpfLbSockHostnsOnly),
	})
	return objectValue(t, cniPluginAttrTypes(), CNIPluginModel{
		Type:   types.StringValue("cilium"),
		Cilium: cilium,
	})
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
