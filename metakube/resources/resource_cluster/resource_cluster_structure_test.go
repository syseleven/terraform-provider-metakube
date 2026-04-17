package resource_cluster

import (
	"context"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/syseleven/go-metakube/models"
	"k8s.io/utils/ptr"
)

func TestFlattenSpecIntoModel(t *testing.T) {
	ctx := context.Background()
	trueBool := true

	cases := []struct {
		name           string
		Input          *models.ClusterSpec
		ExpectedSpec   ClusterSpecModel
		ExpectNullSpec bool
	}{
		{
			name: "full spec",
			Input: &models.ClusterSpec{
				Version: "1.18.8",
				UpdateWindow: &models.UpdateWindow{
					Start:  "Tue 02:00",
					Length: "3h",
				},
				EnableUserSSHKeyAgent: &trueBool,
				AuditLogging:          &models.AuditLoggingSettings{},
				Cloud: &models.CloudSpec{
					DatacenterName: "eu-west-1",
					Openstack:      &models.OpenstackCloudSpec{},
				},
				Sys11auth: &models.Sys11AuthSettings{
					Realm: ptr.To("testrealm"),
				},
				ClusterNetwork: &models.ClusterNetworkingConfig{
					Services: &models.NetworkRanges{
						CIDRBlocks: []string{"1.1.1.0/20"},
					},
					Pods: &models.NetworkRanges{
						CIDRBlocks: []string{"2.2.0.0/16"},
					},
					IPFamily: models.IPFamily("IPv4"),
				},
				CniPlugin: &models.CNIPluginSettings{
					Type: models.CNIPluginType("canal"),
				},
			},
			ExpectedSpec: ClusterSpecModel{
				Version:           types.StringValue("1.18.8"),
				EnableSSHAgent:    types.BoolValue(true),
				AuditLogging:      types.BoolValue(false),
				PodSecurityPolicy: types.BoolValue(false),
				PodNodeSelector:   types.BoolValue(false),
				ServicesCIDR:      types.StringValue("1.1.1.0/20"),
				PodsCIDR:          types.StringValue("2.2.0.0/16"),
				IPFamily:          types.StringValue("IPv4"),
			},
		},
		{
			name: "empty update window",
			Input: &models.ClusterSpec{
				UpdateWindow: &models.UpdateWindow{},
			},
			ExpectedSpec: ClusterSpecModel{
				Version:           types.StringNull(),
				EnableSSHAgent:    types.BoolNull(),
				AuditLogging:      types.BoolValue(false),
				PodSecurityPolicy: types.BoolValue(false),
				PodNodeSelector:   types.BoolValue(false),
				ServicesCIDR:      types.StringNull(),
				PodsCIDR:          types.StringNull(),
				IPFamily:          types.StringNull(),
			},
		},
		{
			name:           "nil spec",
			Input:          nil,
			ExpectNullSpec: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			model := &ClusterModel{}
			diags := metakubeResourceClusterFlattenSpec(ctx, model, tc.Input)
			if diags.HasError() {
				t.Fatalf("Unexpected error: %v", diags)
			}

			if tc.ExpectNullSpec {
				if !model.Spec.IsNull() {
					t.Fatalf("Expected null spec, got %v", model.Spec)
				}
				return
			}

			spec, ok := getClusterSpecModel(ctx, model)
			if !ok {
				t.Fatal("Expected spec object to be set")
			}

			if spec.Version.ValueString() != tc.ExpectedSpec.Version.ValueString() {
				t.Errorf("Version mismatch: got %v, want %v", spec.Version.ValueString(), tc.ExpectedSpec.Version.ValueString())
			}
			if spec.AuditLogging.ValueBool() != tc.ExpectedSpec.AuditLogging.ValueBool() {
				t.Errorf("AuditLogging mismatch: got %v, want %v", spec.AuditLogging.ValueBool(), tc.ExpectedSpec.AuditLogging.ValueBool())
			}
			if spec.PodSecurityPolicy.ValueBool() != tc.ExpectedSpec.PodSecurityPolicy.ValueBool() {
				t.Errorf("PodSecurityPolicy mismatch: got %v, want %v", spec.PodSecurityPolicy.ValueBool(), tc.ExpectedSpec.PodSecurityPolicy.ValueBool())
			}
			if spec.PodNodeSelector.ValueBool() != tc.ExpectedSpec.PodNodeSelector.ValueBool() {
				t.Errorf("PodNodeSelector mismatch: got %v, want %v", spec.PodNodeSelector.ValueBool(), tc.ExpectedSpec.PodNodeSelector.ValueBool())
			}
		})
	}
}

func TestFlattenCniPlugin(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name         string
		Input        *models.CNIPluginSettings
		ExpectedType string
		ExpectNull   bool
	}{
		{
			name:         "API returns cilium",
			Input:        &models.CNIPluginSettings{Type: models.CNIPluginType("cilium")},
			ExpectedType: "cilium",
		},
		{
			name:         "API returns canal",
			Input:        &models.CNIPluginSettings{Type: models.CNIPluginType("canal")},
			ExpectedType: "canal",
		},
		{
			name:       "API returns empty type - should be null",
			Input:      &models.CNIPluginSettings{},
			ExpectNull: true,
		},
		{
			name:       "API returns nil - should be null",
			Input:      nil,
			ExpectNull: true,
		},
		{
			name:       "API returns none - should be null",
			Input:      &models.CNIPluginSettings{Type: models.CNIPluginType("none")},
			ExpectNull: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			specModel := &ClusterSpecModel{}
			diags := flattenCniPlugin(ctx, specModel, tc.Input)
			if diags.HasError() {
				t.Fatalf("Unexpected error: %v", diags)
			}

			if tc.ExpectNull {
				if !specModel.CNIPlugin.IsNull() {
					t.Fatalf("Expected CNI plugin to be null")
				}
				return
			}

			if specModel.CNIPlugin.IsNull() {
				t.Fatalf("Expected CNI plugin to not be null")
			}

			var plugin CNIPluginModel
			if d := specModel.CNIPlugin.As(ctx, &plugin, basetypes.ObjectAsOptions{}); d.HasError() {
				t.Fatalf("Failed to get CNI plugin: %v", d)
			}
			if plugin.Type.ValueString() != tc.ExpectedType {
				t.Errorf("Type mismatch: got %v, want %v", plugin.Type.ValueString(), tc.ExpectedType)
			}
		})
	}
}

func TestFlattenClusterCloudSpec(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name       string
		Input      *models.CloudSpec
		ExpectNull bool
	}{
		{
			name:  "empty cloud",
			Input: &models.CloudSpec{},
		},
		{
			name:       "nil cloud",
			Input:      nil,
			ExpectNull: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			specModel := &ClusterSpecModel{}
			diags := flattenClusterCloudSpec(ctx, specModel, clusterPreserveValues{}, tc.Input)
			if diags.HasError() {
				t.Fatalf("Unexpected error: %v", diags)
			}

			if tc.ExpectNull {
				if !specModel.Cloud.IsNull() {
					t.Fatalf("Expected null cloud, got %v", specModel.Cloud)
				}
				return
			}

			var cloud ClusterCloudSpecModel
			if d := specModel.Cloud.As(ctx, &cloud, basetypes.ObjectAsOptions{}); d.HasError() {
				t.Fatalf("Failed to get cloud object: %v", d)
			}
		})
	}
}

func TestFlattenOpenstackCloudSpec(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name           string
		Input          *models.OpenstackCloudSpec
		PreserveValues *clusterOpenstackPreservedValues
		ExpectNull     bool
	}{
		{
			name: "with application credentials preserved",
			Input: &models.OpenstackCloudSpec{
				FloatingIPPool: "FloatingIPPool",
				Network:        "Network",
				SecurityGroups: "SecurityGroups",
				SubnetID:       "SubnetID",
				ServerGroupID:  "ServerGroupID",
			},
			PreserveValues: &clusterOpenstackPreservedValues{
				openstackApplicationCredentialsID:     types.StringValue("id"),
				openstackApplicationCredentialsSecret: types.StringValue("secret"),
			},
		},
		{
			name: "with user credentials preserved",
			Input: &models.OpenstackCloudSpec{
				FloatingIPPool: "FloatingIPPool",
				Network:        "Network",
				SecurityGroups: "SecurityGroups",
				SubnetID:       "SubnetID",
				ServerGroupID:  "ServerGroupID",
			},
			PreserveValues: &clusterOpenstackPreservedValues{
				openstackUsername:    types.StringValue("Username"),
				openstackPassword:    types.StringValue("Password"),
				openstackProjectID:   types.StringValue("ProjectID"),
				openstackProjectName: types.StringValue("ProjectName"),
			},
		},
		{
			name:  "empty spec",
			Input: &models.OpenstackCloudSpec{},
		},
		{
			name:       "nil spec",
			Input:      nil,
			ExpectNull: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cloudModel := &ClusterCloudSpecModel{
				Openstack: types.ObjectNull(openstackCloudSpecAttrTypes()),
			}
			diags := flattenOpenstackSpec(ctx, cloudModel, tc.PreserveValues, tc.Input)
			if diags.HasError() {
				t.Fatalf("Unexpected error: %v", diags)
			}

			if tc.ExpectNull {
				if !cloudModel.Openstack.IsNull() {
					t.Fatalf("Expected null Openstack, got %v", cloudModel.Openstack)
				}
				return
			}

			var osSpec OpenstackCloudSpecModel
			if d := cloudModel.Openstack.As(ctx, &osSpec, basetypes.ObjectAsOptions{}); d.HasError() {
				t.Fatalf("Failed to get Openstack object: %v", d)
			}

			if tc.Input != nil && tc.Input.FloatingIPPool != "" {
				if osSpec.FloatingIPPool.ValueString() != tc.Input.FloatingIPPool {
					t.Errorf("FloatingIPPool mismatch: got %v, want %v", osSpec.FloatingIPPool.ValueString(), tc.Input.FloatingIPPool)
				}
			}
		})
	}
}

func TestExpandClusterSpecFromModel(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name           string
		setupModel     func() *ClusterModel
		DCName         string
		ExpectedOutput *models.ClusterSpec
	}{
		{
			name: "full spec",
			setupModel: func() *ClusterModel {
				return createTestClusterModel(ctx, t, ClusterSpecModel{
					Version:           types.StringValue("1.18.8"),
					AuditLogging:      types.BoolValue(false),
					PodSecurityPolicy: types.BoolValue(true),
					PodNodeSelector:   types.BoolValue(true),
					ServicesCIDR:      types.StringValue("1.1.1.0/20"),
					PodsCIDR:          types.StringValue("2.2.0.0/16"),
					IPFamily:          types.StringValue("IPv4"),
					UpdateWindow:      createUpdateWindowList(ctx, t, "Tue 02:00", "3h"),
					CNIPlugin:         createCNIPluginObject(ctx, t, "canal"),
					Cloud:             createOpenstackCloudList(ctx, t),
					SyselevenAuth:     createSyselevenAuthList(ctx, t, "testrealm"),
				})
			},
			DCName: "eu-west-1",
			ExpectedOutput: &models.ClusterSpec{
				Version: "1.18.8",
				UpdateWindow: &models.UpdateWindow{
					Start:  "Tue 02:00",
					Length: "3h",
				},
				AuditLogging:                        &models.AuditLoggingSettings{},
				UsePodSecurityPolicyAdmissionPlugin: true,
				UsePodNodeSelectorAdmissionPlugin:   true,
				ClusterNetwork: &models.ClusterNetworkingConfig{
					Services: &models.NetworkRanges{
						CIDRBlocks: []string{"1.1.1.0/20"},
					},
					Pods: &models.NetworkRanges{
						CIDRBlocks: []string{"2.2.0.0/16"},
					},
					IPFamily: models.IPFamily("IPv4"),
				},
				Cloud: &models.CloudSpec{
					DatacenterName: "eu-west-1",
					Openstack: &models.OpenstackCloudSpec{
						Domain: "Default",
					},
				},
				CniPlugin: &models.CNIPluginSettings{
					Type:   models.CNIPluginType("canal"),
					Cilium: &models.CiliumCNISettings{},
				},
				Sys11auth: &models.Sys11AuthSettings{
					IAMAuthentication: ptr.To(false),
					Realm:             ptr.To("testrealm"),
				},
			},
		},
		{
			name: "empty spec",
			setupModel: func() *ClusterModel {
				return createTestClusterModel(ctx, t, ClusterSpecModel{
					Version:           types.StringNull(),
					EnableSSHAgent:    types.BoolNull(),
					AuditLogging:      types.BoolNull(),
					PodSecurityPolicy: types.BoolNull(),
					PodNodeSelector:   types.BoolNull(),
					ServicesCIDR:      types.StringNull(),
					PodsCIDR:          types.StringNull(),
					IPFamily:          types.StringNull(),
					UpdateWindow:      types.ObjectNull(updateWindowAttrTypes()),
					CNIPlugin:         createCNIPluginObject(ctx, t, "canal"),
					Cloud:             types.ObjectNull(clusterCloudSpecAttrTypes()),
					SyselevenAuth:     types.ObjectNull(syselevenAuthAttrTypes()),
				})
			},
			DCName: "",
			ExpectedOutput: &models.ClusterSpec{
				CniPlugin: &models.CNIPluginSettings{
					Type:   models.CNIPluginType("canal"),
					Cilium: &models.CiliumCNISettings{},
				},
			},
		},
		{
			name: "nil spec",
			setupModel: func() *ClusterModel {
				return &ClusterModel{
					Spec: types.ObjectNull(clusterSpecAttrTypes()),
				}
			},
			DCName:         "",
			ExpectedOutput: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			model := tc.setupModel()
			output := metakubeResourceClusterExpandSpec(ctx, model, tc.DCName, func(string) bool { return true })
			if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
				t.Fatalf("Unexpected output from expander: mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestExpandClusterCloudSpec(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name           string
		setupObject    func() types.Object
		DCName         string
		ExpectedOutput *models.CloudSpec
	}{
		{
			name: "openstack cloud",
			setupObject: func() types.Object {
				return createOpenstackCloudList(ctx, t)
			},
			DCName: "eu-west-1",
			ExpectedOutput: &models.CloudSpec{
				DatacenterName: "eu-west-1",
				Openstack: &models.OpenstackCloudSpec{
					Domain: "Default",
				},
			},
		},
		{
			name: "empty cloud",
			setupObject: func() types.Object {
				cloudModel := ClusterCloudSpecModel{
					Openstack: types.ObjectNull(openstackCloudSpecAttrTypes()),
				}
				objVal, _ := types.ObjectValueFrom(ctx, clusterCloudSpecAttrTypes(), cloudModel)
				return objVal
			},
			DCName: "eu-west-1",
			ExpectedOutput: &models.CloudSpec{
				DatacenterName: "eu-west-1",
			},
		},
		{
			name: "null cloud",
			setupObject: func() types.Object {
				return types.ObjectNull(clusterCloudSpecAttrTypes())
			},
			DCName:         "eu-west-1",
			ExpectedOutput: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			object := tc.setupObject()
			output := expandClusterCloudSpec(ctx, object, tc.DCName, func(string) bool { return true })
			if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
				t.Fatalf("Unexpected output from expander: mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestExpandCniPlugin(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name           string
		setupObject    func() types.Object
		ExpectedOutput *models.CNIPluginSettings
	}{
		{
			name: "canal",
			setupObject: func() types.Object {
				return createCNIPluginObject(ctx, t, "canal")
			},
			ExpectedOutput: &models.CNIPluginSettings{
				Type:   "canal",
				Cilium: &models.CiliumCNISettings{},
			},
		},
		{
			name: "cilium",
			setupObject: func() types.Object {
				return createCNIPluginObject(ctx, t, "cilium")
			},
			ExpectedOutput: &models.CNIPluginSettings{
				Type:   "cilium",
				Cilium: &models.CiliumCNISettings{},
			},
		},
		{
			name: "empty type - should be nil",
			setupObject: func() types.Object {
				cniModel := CNIPluginModel{
					Type: types.StringNull(),
				}
				objVal, _ := types.ObjectValueFrom(ctx, cniPluginAttrTypes(), cniModel)
				return objVal
			},
			ExpectedOutput: nil,
		},
		{
			name: "null object - should be nil",
			setupObject: func() types.Object {
				return types.ObjectNull(cniPluginAttrTypes())
			},
			ExpectedOutput: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			obj := tc.setupObject()
			output := expandCniPlugin(ctx, obj)
			if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
				t.Fatalf("Unexpected output from expander: mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestExpandOpenstackCloudSpec(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name           string
		setupObject    func() types.Object
		ExpectedOutput *models.OpenstackCloudSpec
	}{
		{
			name: "with user credentials",
			setupObject: func() types.Object {
				userCredsModel := OpenstackUserCredentialsModel{
					Username:    types.StringValue("Username"),
					Password:    types.StringValue("Password"),
					ProjectID:   types.StringValue("ProjectID"),
					ProjectName: types.StringValue("ProjectName"),
				}
				userCredsObjVal, _ := types.ObjectValueFrom(ctx, openstackUserCredentialsAttrTypes(), userCredsModel)

				osModel := OpenstackCloudSpecModel{
					FloatingIPPool:         types.StringValue("FloatingIPPool"),
					ServerGroupID:          types.StringValue("ServerGroupID"),
					UserCredentials:        userCredsObjVal,
					ApplicationCredentials: types.ObjectNull(openstackApplicationCredentialsAttrTypes()),
					SecurityGroup:          types.StringNull(),
					Network:                types.StringNull(),
					SubnetID:               types.StringNull(),
					SubnetCIDR:             types.StringNull(),
				}
				objVal, _ := types.ObjectValueFrom(ctx, openstackCloudSpecAttrTypes(), osModel)
				return objVal
			},
			ExpectedOutput: &models.OpenstackCloudSpec{
				Domain:         "Default",
				FloatingIPPool: "FloatingIPPool",
				Username:       "Username",
				Password:       "Password",
				ProjectID:      "ProjectID",
				Project:        "ProjectName",
				ServerGroupID:  "ServerGroupID",
			},
		},
		{
			name: "with application credentials",
			setupObject: func() types.Object {
				appCredsModel := OpenstackApplicationCredentialsModel{
					ID:     types.StringValue("id"),
					Secret: types.StringValue("secret"),
				}
				appCredsObjVal, _ := types.ObjectValueFrom(ctx, openstackApplicationCredentialsAttrTypes(), appCredsModel)

				osModel := OpenstackCloudSpecModel{
					FloatingIPPool:         types.StringValue("FloatingIPPool"),
					ServerGroupID:          types.StringValue("ServerGroupID"),
					ApplicationCredentials: appCredsObjVal,
					UserCredentials:        types.ObjectNull(openstackUserCredentialsAttrTypes()),
					SecurityGroup:          types.StringNull(),
					Network:                types.StringNull(),
					SubnetID:               types.StringNull(),
					SubnetCIDR:             types.StringNull(),
				}
				objVal, _ := types.ObjectValueFrom(ctx, openstackCloudSpecAttrTypes(), osModel)
				return objVal
			},
			ExpectedOutput: &models.OpenstackCloudSpec{
				Domain:                      "Default",
				FloatingIPPool:              "FloatingIPPool",
				ApplicationCredentialID:     "id",
				ApplicationCredentialSecret: "secret",
				ServerGroupID:               "ServerGroupID",
			},
		},
		{
			name: "empty openstack spec",
			setupObject: func() types.Object {
				osModel := OpenstackCloudSpecModel{
					FloatingIPPool:         types.StringNull(),
					SecurityGroup:          types.StringNull(),
					Network:                types.StringNull(),
					SubnetID:               types.StringNull(),
					SubnetCIDR:             types.StringNull(),
					ServerGroupID:          types.StringNull(),
					UserCredentials:        types.ObjectNull(openstackUserCredentialsAttrTypes()),
					ApplicationCredentials: types.ObjectNull(openstackApplicationCredentialsAttrTypes()),
				}
				objVal, _ := types.ObjectValueFrom(ctx, openstackCloudSpecAttrTypes(), osModel)
				return objVal
			},
			ExpectedOutput: &models.OpenstackCloudSpec{
				Domain: "Default",
			},
		},
		{
			name: "null list",
			setupObject: func() types.Object {
				return types.ObjectNull(openstackCloudSpecAttrTypes())
			},
			ExpectedOutput: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			object := tc.setupObject()
			output := expandOpenstackCloudSpec(ctx, object, func(string) bool { return true })
			if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
				t.Fatalf("Unexpected output from expander: mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestExpandAuditLogging(t *testing.T) {
	want := &models.AuditLoggingSettings{
		Enabled: true,
	}
	got := expandAuditLogging(true)
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("want %+v, got %+v", want, got)
	}
}

func TestGetPreservedValuesFromModel(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name       string
		setupModel func() *ClusterModel
		expectOS   *clusterOpenstackPreservedValues
	}{
		{
			name: "null spec returns empty values",
			setupModel: func() *ClusterModel {
				return &ClusterModel{
					Spec: types.ObjectNull(clusterSpecAttrTypes()),
				}
			},
			expectOS: nil,
		},
		{
			name: "model with OpenStack user credentials preserves them",
			setupModel: func() *ClusterModel {
				return createModelWithOpenstackUserCredentials(ctx, t, "testuser", "testpass", "project-123", "myproject")
			},
			expectOS: &clusterOpenstackPreservedValues{
				openstackUsername:    types.StringValue("testuser"),
				openstackPassword:    types.StringValue("testpass"),
				openstackProjectID:   types.StringValue("project-123"),
				openstackProjectName: types.StringValue("myproject"),
			},
		},
		{
			name: "model with OpenStack application credentials preserves them",
			setupModel: func() *ClusterModel {
				return createModelWithOpenstackAppCredentials(ctx, t, "app-id-123", "app-secret-456")
			},
			expectOS: &clusterOpenstackPreservedValues{
				openstackApplicationCredentialsID:     types.StringValue("app-id-123"),
				openstackApplicationCredentialsSecret: types.StringValue("app-secret-456"),
			},
		},
		{
			name: "model with null cloud returns empty values",
			setupModel: func() *ClusterModel {
				specModel := ClusterSpecModel{
					Version:       types.StringValue("1.18.8"),
					Cloud:         types.ObjectNull(clusterCloudSpecAttrTypes()),
					UpdateWindow:  types.ObjectNull(updateWindowAttrTypes()),
					CNIPlugin:     createCNIPluginObject(ctx, t, "canal"),
					SyselevenAuth: types.ObjectNull(syselevenAuthAttrTypes()),
				}
				return createTestClusterModel(ctx, t, specModel)
			},
			expectOS: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			model := tc.setupModel()
			result := getPreservedValuesFromModel(ctx, model)

			// Check OpenStack
			if tc.expectOS == nil {
				if result.openstack != nil {
					t.Errorf("Expected nil OpenStack, got %+v", result.openstack)
				}
			} else {
				if result.openstack == nil {
					t.Fatal("Expected OpenStack to be set, got nil")
				}
				if result.openstack.openstackUsername.ValueString() != tc.expectOS.openstackUsername.ValueString() {
					t.Errorf("OpenStack Username mismatch: got %v, want %v", result.openstack.openstackUsername.ValueString(), tc.expectOS.openstackUsername.ValueString())
				}
				if result.openstack.openstackPassword.ValueString() != tc.expectOS.openstackPassword.ValueString() {
					t.Errorf("OpenStack Password mismatch: got %v, want %v", result.openstack.openstackPassword.ValueString(), tc.expectOS.openstackPassword.ValueString())
				}
				if result.openstack.openstackProjectID.ValueString() != tc.expectOS.openstackProjectID.ValueString() {
					t.Errorf("OpenStack ProjectID mismatch: got %v, want %v", result.openstack.openstackProjectID.ValueString(), tc.expectOS.openstackProjectID.ValueString())
				}
				if result.openstack.openstackProjectName.ValueString() != tc.expectOS.openstackProjectName.ValueString() {
					t.Errorf("OpenStack ProjectName mismatch: got %v, want %v", result.openstack.openstackProjectName.ValueString(), tc.expectOS.openstackProjectName.ValueString())
				}
				if result.openstack.openstackApplicationCredentialsID.ValueString() != tc.expectOS.openstackApplicationCredentialsID.ValueString() {
					t.Errorf("OpenStack AppCredID mismatch: got %v, want %v", result.openstack.openstackApplicationCredentialsID.ValueString(), tc.expectOS.openstackApplicationCredentialsID.ValueString())
				}
				if result.openstack.openstackApplicationCredentialsSecret.ValueString() != tc.expectOS.openstackApplicationCredentialsSecret.ValueString() {
					t.Errorf("OpenStack AppCredSecret mismatch: got %v, want %v", result.openstack.openstackApplicationCredentialsSecret.ValueString(), tc.expectOS.openstackApplicationCredentialsSecret.ValueString())
				}
			}
		})
	}
}

func TestFlattenOpenstackSpecPreservesCredentials(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name                  string
		apiResponse           *models.OpenstackCloudSpec
		preservedValues       *clusterOpenstackPreservedValues
		expectedUsername      string
		expectedPassword      string
		expectedProjectID     string
		expectedProjectName   string
		expectedAppCredID     string
		expectedAppCredSecret string
		expectedServerGroupID string
		expectUserCredentials bool
		expectAppCredentials  bool
	}{
		{
			name: "preserves user credentials when API returns empty",
			apiResponse: &models.OpenstackCloudSpec{
				FloatingIPPool: "ext-net",
				Network:        "my-network",
				// API doesn't return credentials
			},
			preservedValues: &clusterOpenstackPreservedValues{
				openstackUsername:    types.StringValue("preserved-user"),
				openstackPassword:    types.StringValue("preserved-pass"),
				openstackProjectID:   types.StringValue("preserved-proj-id"),
				openstackProjectName: types.StringValue("preserved-proj-name"),
			},
			expectedUsername:      "preserved-user",
			expectedPassword:      "preserved-pass",
			expectedProjectID:     "preserved-proj-id",
			expectedProjectName:   "preserved-proj-name",
			expectUserCredentials: true,
			expectAppCredentials:  false,
		},
		{
			name: "preserves application credentials when API returns empty",
			apiResponse: &models.OpenstackCloudSpec{
				FloatingIPPool: "ext-net",
			},
			preservedValues: &clusterOpenstackPreservedValues{
				openstackApplicationCredentialsID:     types.StringValue("preserved-app-id"),
				openstackApplicationCredentialsSecret: types.StringValue("preserved-app-secret"),
			},
			expectedAppCredID:     "preserved-app-id",
			expectedAppCredSecret: "preserved-app-secret",
			expectUserCredentials: false,
			expectAppCredentials:  true,
		},
		{
			name: "preserves ServerGroupID when API returns empty but state has value",
			apiResponse: &models.OpenstackCloudSpec{
				FloatingIPPool: "ext-net",
				ServerGroupID:  "", // API returns empty
			},
			preservedValues: &clusterOpenstackPreservedValues{
				openstackServerGroupID: types.StringValue("preserved-server-group"),
			},
			expectedServerGroupID: "preserved-server-group",
			expectUserCredentials: false,
			expectAppCredentials:  false,
		},
		{
			name: "API ServerGroupID takes precedence over preserved value",
			apiResponse: &models.OpenstackCloudSpec{
				FloatingIPPool: "ext-net",
				ServerGroupID:  "api-server-group",
			},
			preservedValues: &clusterOpenstackPreservedValues{
				openstackServerGroupID: types.StringValue("preserved-server-group"),
			},
			expectedServerGroupID: "api-server-group",
			expectUserCredentials: false,
			expectAppCredentials:  false,
		},
		{
			name: "nil preserved values results in null credentials",
			apiResponse: &models.OpenstackCloudSpec{
				FloatingIPPool: "ext-net",
			},
			preservedValues:       nil,
			expectUserCredentials: false,
			expectAppCredentials:  false,
		},
		{
			name: "partial user credentials - only password set",
			apiResponse: &models.OpenstackCloudSpec{
				FloatingIPPool: "ext-net",
			},
			preservedValues: &clusterOpenstackPreservedValues{
				openstackPassword: types.StringValue("only-password"),
				openstackUsername: types.StringNull(),
			},
			expectedPassword:      "only-password",
			expectUserCredentials: true,
			expectAppCredentials:  false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cloudModel := &ClusterCloudSpecModel{
				Openstack: types.ObjectNull(openstackCloudSpecAttrTypes()),
			}
			diags := flattenOpenstackSpec(ctx, cloudModel, tc.preservedValues, tc.apiResponse)
			if diags.HasError() {
				t.Fatalf("Unexpected error: %v", diags)
			}

			var os OpenstackCloudSpecModel
			if d := cloudModel.Openstack.As(ctx, &os, basetypes.ObjectAsOptions{}); d.HasError() {
				t.Fatalf("Failed to get Openstack object: %v", d)
			}

			// Check ServerGroupID
			if tc.expectedServerGroupID != "" {
				if os.ServerGroupID.ValueString() != tc.expectedServerGroupID {
					t.Errorf("ServerGroupID mismatch: got %v, want %v", os.ServerGroupID.ValueString(), tc.expectedServerGroupID)
				}
			}

			// Check user credentials
			if tc.expectUserCredentials {
				if os.UserCredentials.IsNull() {
					t.Fatal("Expected UserCredentials to be set, got null")
				}
				var userCreds OpenstackUserCredentialsModel
				if d := os.UserCredentials.As(ctx, &userCreds, basetypes.ObjectAsOptions{}); d.HasError() {
					t.Fatalf("Failed to get UserCredentials object: %v", d)
				}
				if tc.expectedUsername != "" && userCreds.Username.ValueString() != tc.expectedUsername {
					t.Errorf("Username mismatch: got %v, want %v", userCreds.Username.ValueString(), tc.expectedUsername)
				}
				if tc.expectedPassword != "" && userCreds.Password.ValueString() != tc.expectedPassword {
					t.Errorf("Password mismatch: got %v, want %v", userCreds.Password.ValueString(), tc.expectedPassword)
				}
				if tc.expectedProjectID != "" && userCreds.ProjectID.ValueString() != tc.expectedProjectID {
					t.Errorf("ProjectID mismatch: got %v, want %v", userCreds.ProjectID.ValueString(), tc.expectedProjectID)
				}
				if tc.expectedProjectName != "" && userCreds.ProjectName.ValueString() != tc.expectedProjectName {
					t.Errorf("ProjectName mismatch: got %v, want %v", userCreds.ProjectName.ValueString(), tc.expectedProjectName)
				}
			} else {
				if !os.UserCredentials.IsNull() {
					t.Errorf("Expected UserCredentials to be null, got %v", os.UserCredentials)
				}
			}

			// Check application credentials
			if tc.expectAppCredentials {
				if os.ApplicationCredentials.IsNull() {
					t.Fatal("Expected ApplicationCredentials to be set, got null")
				}
				var appCreds OpenstackApplicationCredentialsModel
				if d := os.ApplicationCredentials.As(ctx, &appCreds, basetypes.ObjectAsOptions{}); d.HasError() {
					t.Fatalf("Failed to get ApplicationCredentials object: %v", d)
				}
				if appCreds.ID.ValueString() != tc.expectedAppCredID {
					t.Errorf("AppCred ID mismatch: got %v, want %v", appCreds.ID.ValueString(), tc.expectedAppCredID)
				}
				if appCreds.Secret.ValueString() != tc.expectedAppCredSecret {
					t.Errorf("AppCred Secret mismatch: got %v, want %v", appCreds.Secret.ValueString(), tc.expectedAppCredSecret)
				}
			} else {
				if !os.ApplicationCredentials.IsNull() {
					t.Errorf("Expected ApplicationCredentials to be null, got %v", os.ApplicationCredentials)
				}
			}
		})
	}
}

func TestFlattenSpecIntoModelPreservesCloudCredentials(t *testing.T) {
	ctx := context.Background()

	// Test that flattenSpecIntoModel properly extracts and uses preserved values
	cases := []struct {
		name            string
		setupModel      func() *ClusterModel
		apiSpec         *models.ClusterSpec
		verifyPreserved func(t *testing.T, model *ClusterModel)
	}{
		{
			name: "OpenStack credentials preserved through full flatten",
			setupModel: func() *ClusterModel {
				return createModelWithOpenstackUserCredentials(ctx, t, "state-user", "state-pass", "state-proj-id", "state-proj-name")
			},
			apiSpec: &models.ClusterSpec{
				Version: "1.20.0",
				Cloud: &models.CloudSpec{
					DatacenterName: "dc1",
					Openstack: &models.OpenstackCloudSpec{
						FloatingIPPool: "ext-net",
						// API returns empty credentials
					},
				},
			},
			verifyPreserved: func(t *testing.T, model *ClusterModel) {
				spec, ok := getClusterSpecModel(ctx, model)
				if !ok {
					t.Fatal("expected spec object")
				}
				var cloud ClusterCloudSpecModel
				spec.Cloud.As(ctx, &cloud, basetypes.ObjectAsOptions{})
				var osSpec OpenstackCloudSpecModel
				cloud.Openstack.As(ctx, &osSpec, basetypes.ObjectAsOptions{})

				if osSpec.UserCredentials.IsNull() {
					t.Fatal("UserCredentials should not be null")
				}
				var userCreds OpenstackUserCredentialsModel
				osSpec.UserCredentials.As(ctx, &userCreds, basetypes.ObjectAsOptions{})
				if userCreds.Username.ValueString() != "state-user" {
					t.Errorf("Username not preserved: got %v, want state-user", userCreds.Username.ValueString())
				}
				if userCreds.Password.ValueString() != "state-pass" {
					t.Errorf("Password not preserved: got %v, want state-pass", userCreds.Password.ValueString())
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			model := tc.setupModel()
			diags := metakubeResourceClusterFlattenSpec(ctx, model, tc.apiSpec)
			if diags.HasError() {
				t.Fatalf("Unexpected error: %v", diags)
			}
			tc.verifyPreserved(t, model)
		})
	}
}

// TestFlattenSpecIntoModelPopulatesCNIPluginFromAPI tests that CNI plugin
// is populated from the API response when API returns a valid value, or null when API returns nil/empty/none.
func TestFlattenSpecIntoModelPopulatesCNIPluginFromAPI(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name        string
		setupModel  func() *ClusterModel
		apiSpec     *models.ClusterSpec
		expectedCNI string
		expectNull  bool
	}{
		{
			name: "Config has no CNI block, API returns cilium - state should have cilium",
			setupModel: func() *ClusterModel {
				specModel := ClusterSpecModel{
					Version:       types.StringValue("1.20.0"),
					CNIPlugin:     createCNIPluginObject(ctx, t, "canal"),
					Cloud:         createOpenstackCloudList(ctx, t),
					UpdateWindow:  types.ObjectNull(updateWindowAttrTypes()),
					SyselevenAuth: types.ObjectNull(syselevenAuthAttrTypes()),
				}
				return createTestClusterModel(ctx, t, specModel)
			},
			apiSpec: &models.ClusterSpec{
				Version: "1.20.0",
				CniPlugin: &models.CNIPluginSettings{
					Type: models.CNIPluginType("cilium"),
				},
				Cloud: &models.CloudSpec{
					DatacenterName: "dc1",
					Openstack:      &models.OpenstackCloudSpec{},
				},
			},
			expectedCNI: "cilium",
		},
		{
			name: "Config has CNI block with canal, API returns canal - state should have canal",
			setupModel: func() *ClusterModel {
				specModel := ClusterSpecModel{
					Version:       types.StringValue("1.20.0"),
					CNIPlugin:     createCNIPluginObject(ctx, t, "canal"),
					Cloud:         createOpenstackCloudList(ctx, t),
					UpdateWindow:  types.ObjectNull(updateWindowAttrTypes()),
					SyselevenAuth: types.ObjectNull(syselevenAuthAttrTypes()),
				}
				return createTestClusterModel(ctx, t, specModel)
			},
			apiSpec: &models.ClusterSpec{
				Version: "1.20.0",
				CniPlugin: &models.CNIPluginSettings{
					Type: models.CNIPluginType("canal"),
				},
				Cloud: &models.CloudSpec{
					DatacenterName: "dc1",
					Openstack:      &models.OpenstackCloudSpec{},
				},
			},
			expectedCNI: "canal",
		},
		{
			name: "Config has CNI block with cilium, API returns nil - state should be null",
			setupModel: func() *ClusterModel {
				specModel := ClusterSpecModel{
					Version:       types.StringValue("1.20.0"),
					CNIPlugin:     createCNIPluginObject(ctx, t, "cilium"),
					Cloud:         createOpenstackCloudList(ctx, t),
					UpdateWindow:  types.ObjectNull(updateWindowAttrTypes()),
					SyselevenAuth: types.ObjectNull(syselevenAuthAttrTypes()),
				}
				return createTestClusterModel(ctx, t, specModel)
			},
			apiSpec: &models.ClusterSpec{
				Version:   "1.20.0",
				CniPlugin: nil,
				Cloud: &models.CloudSpec{
					DatacenterName: "dc1",
					Openstack:      &models.OpenstackCloudSpec{},
				},
			},
			expectNull: true,
		},
		{
			name: "Config has CNI block with cilium, API returns cilium - state should have cilium",
			setupModel: func() *ClusterModel {
				specModel := ClusterSpecModel{
					Version:       types.StringValue("1.20.0"),
					CNIPlugin:     createCNIPluginObject(ctx, t, "cilium"),
					Cloud:         createOpenstackCloudList(ctx, t),
					UpdateWindow:  types.ObjectNull(updateWindowAttrTypes()),
					SyselevenAuth: types.ObjectNull(syselevenAuthAttrTypes()),
				}
				return createTestClusterModel(ctx, t, specModel)
			},
			apiSpec: &models.ClusterSpec{
				Version: "1.20.0",
				CniPlugin: &models.CNIPluginSettings{
					Type: models.CNIPluginType("cilium"),
				},
				Cloud: &models.CloudSpec{
					DatacenterName: "dc1",
					Openstack:      &models.OpenstackCloudSpec{},
				},
			},
			expectedCNI: "cilium",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			model := tc.setupModel()
			diags := metakubeResourceClusterFlattenSpec(ctx, model, tc.apiSpec)
			if diags.HasError() {
				t.Fatalf("Unexpected error: %v", diags)
			}

			spec, ok := getClusterSpecModel(ctx, model)
			if !ok {
				t.Fatal("Expected spec object to be set")
			}

			if tc.expectNull {
				if !spec.CNIPlugin.IsNull() {
					t.Fatal("Expected CNI plugin to be null")
				}
				return
			}

			if spec.CNIPlugin.IsNull() {
				t.Fatal("Expected CNI plugin to be set, got null")
			}
			var plugin CNIPluginModel
			if d := spec.CNIPlugin.As(ctx, &plugin, basetypes.ObjectAsOptions{}); d.HasError() {
				t.Fatalf("Failed to get CNI plugin: %v", d)
			}
			if plugin.Type.ValueString() != tc.expectedCNI {
				t.Errorf("CNI type mismatch: got %v, want %v", plugin.Type.ValueString(), tc.expectedCNI)
			}
		})
	}
}

// Helper functions for creating test data

func createTestClusterModel(ctx context.Context, t *testing.T, specModel ClusterSpecModel) *ClusterModel {
	t.Helper()
	specObjVal, err := types.ObjectValueFrom(ctx, clusterSpecAttrTypes(), specModel)
	if err != nil {
		t.Fatalf("Failed to create spec object: %v", err)
	}
	return &ClusterModel{
		Spec: specObjVal,
	}
}

func createUpdateWindowList(ctx context.Context, t *testing.T, start, length string) types.Object {
	t.Helper()
	uwModel := UpdateWindowModel{
		Start:  types.StringValue(start),
		Length: types.StringValue(length),
	}
	objVal, _ := types.ObjectValueFrom(ctx, updateWindowAttrTypes(), uwModel)
	return objVal
}

func createCNIPluginObject(ctx context.Context, t *testing.T, pluginType string) types.Object {
	t.Helper()
	cniModel := CNIPluginModel{
		Type:   types.StringValue(pluginType),
		Cilium: types.ObjectNull(ciliumAttrTypes()),
	}
	objVal, _ := types.ObjectValueFrom(ctx, cniPluginAttrTypes(), cniModel)
	return objVal
}

func createSyselevenAuthList(ctx context.Context, t *testing.T, realm string) types.Object {
	t.Helper()
	authModel := SyselevenAuthModel{
		Realm:             types.StringValue(realm),
		IAMAuthentication: types.BoolValue(false),
	}
	objVal, _ := types.ObjectValueFrom(ctx, syselevenAuthAttrTypes(), authModel)
	return objVal
}

func createOpenstackCloudList(ctx context.Context, t *testing.T) types.Object {
	t.Helper()
	osModel := OpenstackCloudSpecModel{
		FloatingIPPool:         types.StringNull(),
		SecurityGroup:          types.StringNull(),
		Network:                types.StringNull(),
		SubnetID:               types.StringNull(),
		SubnetCIDR:             types.StringNull(),
		ServerGroupID:          types.StringNull(),
		UserCredentials:        types.ObjectNull(openstackUserCredentialsAttrTypes()),
		ApplicationCredentials: types.ObjectNull(openstackApplicationCredentialsAttrTypes()),
	}
	osObjVal, _ := types.ObjectValueFrom(ctx, openstackCloudSpecAttrTypes(), osModel)

	cloudModel := ClusterCloudSpecModel{
		Openstack: osObjVal,
	}
	cloudObjVal, _ := types.ObjectValueFrom(ctx, clusterCloudSpecAttrTypes(), cloudModel)
	return cloudObjVal
}

func createModelWithOpenstackUserCredentials(ctx context.Context, t *testing.T, username, password, projectID, projectName string) *ClusterModel {
	t.Helper()
	userCredsModel := OpenstackUserCredentialsModel{
		Username:    types.StringValue(username),
		Password:    types.StringValue(password),
		ProjectID:   types.StringValue(projectID),
		ProjectName: types.StringValue(projectName),
	}
	userCredsObjVal, _ := types.ObjectValueFrom(ctx, openstackUserCredentialsAttrTypes(), userCredsModel)

	osModel := OpenstackCloudSpecModel{
		FloatingIPPool:         types.StringNull(),
		SecurityGroup:          types.StringNull(),
		Network:                types.StringNull(),
		SubnetID:               types.StringNull(),
		SubnetCIDR:             types.StringNull(),
		ServerGroupID:          types.StringNull(),
		UserCredentials:        userCredsObjVal,
		ApplicationCredentials: types.ObjectNull(openstackApplicationCredentialsAttrTypes()),
	}
	osObjVal, _ := types.ObjectValueFrom(ctx, openstackCloudSpecAttrTypes(), osModel)

	cloudModel := ClusterCloudSpecModel{
		Openstack: osObjVal,
	}
	cloudObjVal, _ := types.ObjectValueFrom(ctx, clusterCloudSpecAttrTypes(), cloudModel)

	specModel := ClusterSpecModel{
		Version:           types.StringValue("1.20.0"),
		EnableSSHAgent:    types.BoolNull(),
		AuditLogging:      types.BoolNull(),
		PodSecurityPolicy: types.BoolNull(),
		PodNodeSelector:   types.BoolNull(),
		ServicesCIDR:      types.StringNull(),
		PodsCIDR:          types.StringNull(),
		IPFamily:          types.StringNull(),
		UpdateWindow:      types.ObjectNull(updateWindowAttrTypes()),
		CNIPlugin:         createCNIPluginObject(ctx, t, "canal"),
		Cloud:             cloudObjVal,
		SyselevenAuth:     types.ObjectNull(syselevenAuthAttrTypes()),
	}
	return createTestClusterModel(ctx, t, specModel)
}

func createModelWithOpenstackAppCredentials(ctx context.Context, t *testing.T, appCredID, appCredSecret string) *ClusterModel {
	t.Helper()
	appCredsModel := OpenstackApplicationCredentialsModel{
		ID:     types.StringValue(appCredID),
		Secret: types.StringValue(appCredSecret),
	}
	appCredsObjVal, _ := types.ObjectValueFrom(ctx, openstackApplicationCredentialsAttrTypes(), appCredsModel)

	osModel := OpenstackCloudSpecModel{
		FloatingIPPool:         types.StringNull(),
		SecurityGroup:          types.StringNull(),
		Network:                types.StringNull(),
		SubnetID:               types.StringNull(),
		SubnetCIDR:             types.StringNull(),
		ServerGroupID:          types.StringNull(),
		UserCredentials:        types.ObjectNull(openstackUserCredentialsAttrTypes()),
		ApplicationCredentials: appCredsObjVal,
	}
	osObjVal, _ := types.ObjectValueFrom(ctx, openstackCloudSpecAttrTypes(), osModel)

	cloudModel := ClusterCloudSpecModel{
		Openstack: osObjVal,
	}
	cloudObjVal, _ := types.ObjectValueFrom(ctx, clusterCloudSpecAttrTypes(), cloudModel)

	specModel := ClusterSpecModel{
		Version:           types.StringValue("1.20.0"),
		EnableSSHAgent:    types.BoolNull(),
		AuditLogging:      types.BoolNull(),
		PodSecurityPolicy: types.BoolNull(),
		PodNodeSelector:   types.BoolNull(),
		ServicesCIDR:      types.StringNull(),
		PodsCIDR:          types.StringNull(),
		IPFamily:          types.StringNull(),
		UpdateWindow:      types.ObjectNull(updateWindowAttrTypes()),
		CNIPlugin:         createCNIPluginObject(ctx, t, "canal"),
		Cloud:             cloudObjVal,
		SyselevenAuth:     types.ObjectNull(syselevenAuthAttrTypes()),
	}
	return createTestClusterModel(ctx, t, specModel)
}

func TestUpgradeClusterLegacyNestedSpecState_ListToObject(t *testing.T) {
	rawState := map[string]any{
		"spec": []any{
			map[string]any{
				"version": "1.31.4",
				"cni_plugin": []any{
					map[string]any{
						"type": "cilium",
					},
				},
			},
		},
	}

	upgradeClusterLegacyNestedSpecState(rawState)

	specMap, ok := rawState["spec"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected spec value after upgrade: %#v", rawState["spec"])
	}

	cniPlugin, ok := specMap["cni_plugin"].(map[string]any)
	if !ok {
		t.Fatalf("expected cni_plugin object after upgrade, got: %#v", specMap["cni_plugin"])
	}

	if got, want := cniPlugin["type"], "cilium"; got != want {
		t.Fatalf("unexpected cni_plugin.type after upgrade: got %v, want %v", got, want)
	}
}

func TestUpgradeClusterLegacyNestedSpecState_EmptyListToNull(t *testing.T) {
	rawState := map[string]any{
		"spec": []any{
			map[string]any{
				"cni_plugin": []any{},
			},
		},
	}

	upgradeClusterLegacyNestedSpecState(rawState)

	specMap := rawState["spec"].(map[string]any)
	if val, ok := specMap["cni_plugin"]; !ok || val != nil {
		t.Fatalf("expected cni_plugin to be null after upgrade, got: %#v", specMap["cni_plugin"])
	}
}

func TestUpgradeClusterLegacyNestedSpecState_RemovesLegacyAzureCloud(t *testing.T) {
	rawState := map[string]any{
		"spec": []any{
			map[string]any{
				"cloud": []any{
					map[string]any{
						"openstack": []any{},
						"azure": []any{
							map[string]any{
								"tenant_id": "legacy-tenant",
							},
						},
					},
				},
			},
		},
	}

	upgradeClusterLegacyNestedSpecState(rawState)

	specMap := rawState["spec"].(map[string]any)
	cloudMap := specMap["cloud"].(map[string]any)

	if _, ok := cloudMap["azure"]; ok {
		t.Fatalf("expected legacy cloud.azure to be removed, got: %#v", cloudMap["azure"])
	}
}

func TestUpgradeClusterLegacyNestedSpecState_FromV5SDKClusterState(t *testing.T) {
	rawState := map[string]any{
		"spec": []any{
			map[string]any{
				"version": "1.31.4",
				"update_window": []any{
					map[string]any{
						"start":  "Tue 02:00",
						"length": "2h",
					},
				},
				"cni_plugin": []any{
					map[string]any{
						"type": "canal",
					},
				},
				"syseleven_auth": []any{
					map[string]any{
						"realm": "syseleven",
					},
				},
				"cloud": []any{
					map[string]any{
						"openstack": []any{
							map[string]any{
								"floating_ip_pool": "ext-net",
								"application_credentials": []any{
									map[string]any{
										"id":     "s11auth:project-id",
										"secret": "secret-value",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	upgradeClusterLegacyNestedSpecState(rawState)

	specMap := rawState["spec"].(map[string]any)

	if _, ok := specMap["update_window"].(map[string]any); !ok {
		t.Fatalf("expected update_window object after upgrade, got: %#v", specMap["update_window"])
	}

	if _, ok := specMap["cni_plugin"].(map[string]any); !ok {
		t.Fatalf("expected cni_plugin object after upgrade, got: %#v", specMap["cni_plugin"])
	}

	sys11Auth, ok := specMap["syseleven_auth"].(map[string]any)
	if !ok {
		t.Fatalf("expected syseleven_auth object after upgrade, got: %#v", specMap["syseleven_auth"])
	}
	if got, want := sys11Auth["realm"], "syseleven"; got != want {
		t.Fatalf("unexpected syseleven_auth.realm after upgrade: got %v, want %v", got, want)
	}

	cloudMap, ok := specMap["cloud"].(map[string]any)
	if !ok {
		t.Fatalf("expected cloud object after upgrade, got: %#v", specMap["cloud"])
	}

	openstackMap, ok := cloudMap["openstack"].(map[string]any)
	if !ok {
		t.Fatalf("expected openstack object after upgrade, got: %#v", cloudMap["openstack"])
	}
	if got, want := openstackMap["floating_ip_pool"], "ext-net"; got != want {
		t.Fatalf("unexpected openstack.floating_ip_pool after upgrade: got %v, want %v", got, want)
	}

	appCreds, ok := openstackMap["application_credentials"].(map[string]any)
	if !ok {
		t.Fatalf("expected application_credentials object after upgrade, got: %#v", openstackMap["application_credentials"])
	}
	if got, want := appCreds["id"], "s11auth:project-id"; got != want {
		t.Fatalf("unexpected application_credentials.id after upgrade: got %v, want %v", got, want)
	}
	if got, want := appCreds["secret"], "secret-value"; got != want {
		t.Fatalf("unexpected application_credentials.secret after upgrade: got %v, want %v", got, want)
	}
}

func TestClusterSpecPatchBody(t *testing.T) {
	t.Run("nil spec returns nil map", func(t *testing.T) {
		got, err := clusterSpecPatchBody(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != nil {
			t.Fatalf("expected nil map, got: %#v", got)
		}
	})

	t.Run("false booleans are forced into the body", func(t *testing.T) {
		spec := &models.ClusterSpec{
			UsePodNodeSelectorAdmissionPlugin:   false,
			UsePodSecurityPolicyAdmissionPlugin: false,
			AuditLogging:                        &models.AuditLoggingSettings{Enabled: false},
			CniPlugin: &models.CNIPluginSettings{
				Type: models.CNIPluginType("cilium"),
				Cilium: &models.CiliumCNISettings{
					EnableHubble:  false,
					EnableL7Proxy: false,
				},
			},
		}

		got, err := clusterSpecPatchBody(spec)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got["usePodNodeSelectorAdmissionPlugin"] != false {
			t.Errorf("expected usePodNodeSelectorAdmissionPlugin=false, got: %#v", got["usePodNodeSelectorAdmissionPlugin"])
		}
		if got["usePodSecurityPolicyAdmissionPlugin"] != false {
			t.Errorf("expected usePodSecurityPolicyAdmissionPlugin=false, got: %#v", got["usePodSecurityPolicyAdmissionPlugin"])
		}

		auditLogging, ok := got["auditLogging"].(map[string]interface{})
		if !ok {
			t.Fatalf("expected auditLogging map, got: %#v", got["auditLogging"])
		}
		if auditLogging["enabled"] != false {
			t.Errorf("expected auditLogging.enabled=false, got: %#v", auditLogging["enabled"])
		}

		cni, ok := got["cniPlugin"].(map[string]interface{})
		if !ok {
			t.Fatalf("expected cniPlugin map, got: %#v", got["cniPlugin"])
		}
		cilium, ok := cni["cilium"].(map[string]interface{})
		if !ok {
			t.Fatalf("expected cniPlugin.cilium map, got: %#v", cni["cilium"])
		}
		if cilium["enableHubble"] != false {
			t.Errorf("expected cniPlugin.cilium.enableHubble=false, got: %#v", cilium["enableHubble"])
		}
		if cilium["enableL7Proxy"] != false {
			t.Errorf("expected cniPlugin.cilium.enableL7Proxy=false, got: %#v", cilium["enableL7Proxy"])
		}
	})

	t.Run("true booleans are preserved", func(t *testing.T) {
		spec := &models.ClusterSpec{
			UsePodNodeSelectorAdmissionPlugin:   true,
			UsePodSecurityPolicyAdmissionPlugin: true,
			AuditLogging:                        &models.AuditLoggingSettings{Enabled: true},
			CniPlugin: &models.CNIPluginSettings{
				Type: models.CNIPluginType("cilium"),
				Cilium: &models.CiliumCNISettings{
					EnableHubble:  true,
					EnableL7Proxy: true,
				},
			},
		}

		got, err := clusterSpecPatchBody(spec)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got["usePodNodeSelectorAdmissionPlugin"] != true {
			t.Errorf("expected usePodNodeSelectorAdmissionPlugin=true, got: %#v", got["usePodNodeSelectorAdmissionPlugin"])
		}
		if got["usePodSecurityPolicyAdmissionPlugin"] != true {
			t.Errorf("expected usePodSecurityPolicyAdmissionPlugin=true, got: %#v", got["usePodSecurityPolicyAdmissionPlugin"])
		}

		auditLogging := got["auditLogging"].(map[string]interface{})
		if auditLogging["enabled"] != true {
			t.Errorf("expected auditLogging.enabled=true, got: %#v", auditLogging["enabled"])
		}

		cilium := got["cniPlugin"].(map[string]interface{})["cilium"].(map[string]interface{})
		if cilium["enableHubble"] != true {
			t.Errorf("expected cniPlugin.cilium.enableHubble=true, got: %#v", cilium["enableHubble"])
		}
		if cilium["enableL7Proxy"] != true {
			t.Errorf("expected cniPlugin.cilium.enableL7Proxy=true, got: %#v", cilium["enableL7Proxy"])
		}
	})

	t.Run("absent auditLogging and cniPlugin are not fabricated", func(t *testing.T) {
		spec := &models.ClusterSpec{
			UsePodNodeSelectorAdmissionPlugin:   false,
			UsePodSecurityPolicyAdmissionPlugin: true,
		}

		got, err := clusterSpecPatchBody(spec)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if _, ok := got["auditLogging"]; ok {
			t.Errorf("expected auditLogging to be absent, got: %#v", got["auditLogging"])
		}
		if _, ok := got["cniPlugin"]; ok {
			t.Errorf("expected cniPlugin to be absent, got: %#v", got["cniPlugin"])
		}
		if got["usePodNodeSelectorAdmissionPlugin"] != false {
			t.Errorf("expected usePodNodeSelectorAdmissionPlugin=false, got: %#v", got["usePodNodeSelectorAdmissionPlugin"])
		}
		if got["usePodSecurityPolicyAdmissionPlugin"] != true {
			t.Errorf("expected usePodSecurityPolicyAdmissionPlugin=true, got: %#v", got["usePodSecurityPolicyAdmissionPlugin"])
		}
	})

	t.Run("cniPlugin without cilium is left untouched", func(t *testing.T) {
		spec := &models.ClusterSpec{
			CniPlugin: &models.CNIPluginSettings{
				Type: models.CNIPluginType("canal"),
			},
		}

		got, err := clusterSpecPatchBody(spec)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cni, ok := got["cniPlugin"].(map[string]interface{})
		if !ok {
			t.Fatalf("expected cniPlugin map, got: %#v", got["cniPlugin"])
		}
		if _, ok := cni["cilium"]; ok {
			t.Errorf("expected cniPlugin.cilium to remain absent, got: %#v", cni["cilium"])
		}
	})
}
