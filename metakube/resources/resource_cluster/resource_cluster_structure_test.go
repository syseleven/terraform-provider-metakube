package resource_cluster

import (
	"context"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/syseleven/go-metakube/models"
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
					Realm: "testrealm",
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

			var specs []ClusterSpecModel
			if d := model.Spec.ElementsAs(ctx, &specs, false); d.HasError() {
				t.Fatalf("Failed to get spec elements: %v", d)
			}
			if len(specs) == 0 {
				t.Fatal("Expected spec list to have elements")
			}

			spec := specs[0]

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
			name:         "API returns empty type - defaults to canal",
			Input:        &models.CNIPluginSettings{},
			ExpectedType: "canal",
		},
		{
			name:         "API returns nil - defaults to canal",
			Input:        nil,
			ExpectedType: "canal",
		},
		{
			name:         "API returns none - defaults to canal",
			Input:        &models.CNIPluginSettings{Type: models.CNIPluginType("none")},
			ExpectedType: "canal",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			specModel := &ClusterSpecModel{}
			diags := flattenCniPlugin(ctx, specModel, tc.Input)
			if diags.HasError() {
				t.Fatalf("Unexpected error: %v", diags)
			}

			// CNI should always exist
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
		HasAWS     bool
		ExpectNull bool
	}{
		{
			name: "aws cloud",
			Input: &models.CloudSpec{
				Aws: &models.AWSCloudSpec{},
			},
			HasAWS: true,
		},
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

			var clouds []ClusterCloudSpecModel
			if d := specModel.Cloud.ElementsAs(ctx, &clouds, false); d.HasError() {
				t.Fatalf("Failed to get cloud elements: %v", d)
			}
			if len(clouds) == 0 {
				t.Fatal("Expected cloud list to have elements")
			}

			if tc.HasAWS && clouds[0].AWS.IsNull() {
				t.Error("Expected AWS to be set")
			}
		})
	}
}

func TestFlattenAWSCloudSpec(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name       string
		Input      *models.AWSCloudSpec
		ExpectNull bool
	}{
		{
			name: "full aws spec",
			Input: &models.AWSCloudSpec{
				AccessKeyID:            "AKIAIOSFODNN7EXAMPLE",
				ControlPlaneRoleARN:    "default",
				InstanceProfileName:    "default",
				OpenstackBillingTenant: "foo",
				RouteTableID:           "rtb-09ba434c1bEXAMPLE",
				SecretAccessKey:        "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
				SecurityGroupID:        "sg-51530134",
				VPCID:                  "e5e4b2ef2fe",
			},
		},
		{
			name:  "empty aws spec",
			Input: &models.AWSCloudSpec{},
		},
		{
			name:       "nil aws spec",
			Input:      nil,
			ExpectNull: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cloudModel := &ClusterCloudSpecModel{
				AWS:       types.ListNull(types.ObjectType{AttrTypes: awsCloudSpecAttrTypes()}),
				Openstack: types.ListNull(types.ObjectType{AttrTypes: openstackCloudSpecAttrTypes()}),
				Azure:     types.ListNull(types.ObjectType{AttrTypes: azureCloudSpecAttrTypes()}),
			}
			diags := flattenAWSCloudSpec(ctx, cloudModel, tc.Input)
			if diags.HasError() {
				t.Fatalf("Unexpected error: %v", diags)
			}

			if tc.ExpectNull {
				if !cloudModel.AWS.IsNull() {
					t.Fatalf("Expected null AWS, got %v", cloudModel.AWS)
				}
				return
			}

			var awsSpecs []AWSCloudSpecModel
			if d := cloudModel.AWS.ElementsAs(ctx, &awsSpecs, false); d.HasError() {
				t.Fatalf("Failed to get AWS elements: %v", d)
			}
			if len(awsSpecs) == 0 {
				t.Fatal("Expected AWS list to have elements")
			}

			if tc.Input != nil && tc.Input.AccessKeyID != "" {
				if awsSpecs[0].AccessKeyID.ValueString() != tc.Input.AccessKeyID {
					t.Errorf("AccessKeyID mismatch: got %v, want %v", awsSpecs[0].AccessKeyID.ValueString(), tc.Input.AccessKeyID)
				}
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
				AWS:       types.ListNull(types.ObjectType{AttrTypes: awsCloudSpecAttrTypes()}),
				Openstack: types.ListNull(types.ObjectType{AttrTypes: openstackCloudSpecAttrTypes()}),
				Azure:     types.ListNull(types.ObjectType{AttrTypes: azureCloudSpecAttrTypes()}),
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

			var osSpecs []OpenstackCloudSpecModel
			if d := cloudModel.Openstack.ElementsAs(ctx, &osSpecs, false); d.HasError() {
				t.Fatalf("Failed to get Openstack elements: %v", d)
			}
			if len(osSpecs) == 0 {
				t.Fatal("Expected Openstack list to have elements")
			}

			if tc.Input != nil && tc.Input.FloatingIPPool != "" {
				if osSpecs[0].FloatingIPPool.ValueString() != tc.Input.FloatingIPPool {
					t.Errorf("FloatingIPPool mismatch: got %v, want %v", osSpecs[0].FloatingIPPool.ValueString(), tc.Input.FloatingIPPool)
				}
			}
		})
	}
}

func TestFlattenAzureCloudSpec(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name       string
		Input      *models.AzureCloudSpec
		ExpectNull bool
	}{
		{
			name: "full azure spec",
			Input: &models.AzureCloudSpec{
				ClientID:               "ClientID",
				ClientSecret:           "ClientSecret",
				SubscriptionID:         "SubscriptionID",
				TenantID:               "TenantID",
				ResourceGroup:          "ResourceGroup",
				RouteTableName:         "RouteTableName",
				SecurityGroup:          "SecurityGroup",
				SubnetName:             "SubnetName",
				VNetName:               "VNetName",
				OpenstackBillingTenant: "foo",
			},
		},
		{
			name:  "empty azure spec",
			Input: &models.AzureCloudSpec{},
		},
		{
			name:       "nil azure spec",
			Input:      nil,
			ExpectNull: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cloudModel := &ClusterCloudSpecModel{
				AWS:       types.ListNull(types.ObjectType{AttrTypes: awsCloudSpecAttrTypes()}),
				Openstack: types.ListNull(types.ObjectType{AttrTypes: openstackCloudSpecAttrTypes()}),
				Azure:     types.ListNull(types.ObjectType{AttrTypes: azureCloudSpecAttrTypes()}),
			}
			diags := flattenAzureSpec(ctx, cloudModel, tc.Input)
			if diags.HasError() {
				t.Fatalf("Unexpected error: %v", diags)
			}

			if tc.ExpectNull {
				if !cloudModel.Azure.IsNull() {
					t.Fatalf("Expected null Azure, got %v", cloudModel.Azure)
				}
				return
			}

			var azureSpecs []AzureCloudSpecModel
			if d := cloudModel.Azure.ElementsAs(ctx, &azureSpecs, false); d.HasError() {
				t.Fatalf("Failed to get Azure elements: %v", d)
			}
			if len(azureSpecs) == 0 {
				t.Fatal("Expected Azure list to have elements")
			}

			if tc.Input != nil && tc.Input.ClientID != "" {
				if azureSpecs[0].ClientID.ValueString() != tc.Input.ClientID {
					t.Errorf("ClientID mismatch: got %v, want %v", azureSpecs[0].ClientID.ValueString(), tc.Input.ClientID)
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
					Type: models.CNIPluginType("canal"),
				},
				Sys11auth: &models.Sys11AuthSettings{
					Realm: "testrealm",
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
					UpdateWindow:      types.ListNull(types.ObjectType{AttrTypes: updateWindowAttrTypes()}),
					CNIPlugin:         createCNIPluginObject(ctx, t, "canal"),
					Cloud:             types.ListNull(types.ObjectType{AttrTypes: clusterCloudSpecAttrTypes()}),
					SyselevenAuth:     types.ListNull(types.ObjectType{AttrTypes: syselevenAuthAttrTypes()}),
				})
			},
			DCName: "",
			ExpectedOutput: &models.ClusterSpec{
				CniPlugin: &models.CNIPluginSettings{
					Type: models.CNIPluginType("canal"),
				},
			},
		},
		{
			name: "nil spec",
			setupModel: func() *ClusterModel {
				return &ClusterModel{
					Spec: types.ListNull(types.ObjectType{AttrTypes: clusterSpecAttrTypes()}),
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
		setupList      func() types.List
		DCName         string
		ExpectedOutput *models.CloudSpec
	}{
		{
			name: "aws cloud",
			setupList: func() types.List {
				return createAWSCloudList(ctx, t)
			},
			DCName: "eu-west-1",
			ExpectedOutput: &models.CloudSpec{
				DatacenterName: "eu-west-1",
				Aws:            &models.AWSCloudSpec{},
			},
		},
		{
			name: "empty cloud",
			setupList: func() types.List {
				cloudModel := ClusterCloudSpecModel{
					AWS:       types.ListNull(types.ObjectType{AttrTypes: awsCloudSpecAttrTypes()}),
					Openstack: types.ListNull(types.ObjectType{AttrTypes: openstackCloudSpecAttrTypes()}),
					Azure:     types.ListNull(types.ObjectType{AttrTypes: azureCloudSpecAttrTypes()}),
				}
				objVal, _ := types.ObjectValueFrom(ctx, clusterCloudSpecAttrTypes(), cloudModel)
				listVal, _ := types.ListValue(types.ObjectType{AttrTypes: clusterCloudSpecAttrTypes()}, []attr.Value{objVal})
				return listVal
			},
			DCName: "eu-west-1",
			ExpectedOutput: &models.CloudSpec{
				DatacenterName: "eu-west-1",
			},
		},
		{
			name: "null cloud",
			setupList: func() types.List {
				return types.ListNull(types.ObjectType{AttrTypes: clusterCloudSpecAttrTypes()})
			},
			DCName:         "eu-west-1",
			ExpectedOutput: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			list := tc.setupList()
			output := expandClusterCloudSpec(ctx, list, tc.DCName, func(string) bool { return true })
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
				Type: "canal",
			},
		},
		{
			name: "cilium",
			setupObject: func() types.Object {
				return createCNIPluginObject(ctx, t, "cilium")
			},
			ExpectedOutput: &models.CNIPluginSettings{
				Type: "cilium",
			},
		},
		{
			name: "empty type - defaults to canal",
			setupObject: func() types.Object {
				cniModel := CNIPluginModel{
					Type: types.StringNull(),
				}
				objVal, _ := types.ObjectValueFrom(ctx, cniPluginAttrTypes(), cniModel)
				return objVal
			},
			ExpectedOutput: &models.CNIPluginSettings{
				Type: "canal",
			},
		},
		{
			name: "null object - defaults to canal",
			setupObject: func() types.Object {
				return types.ObjectNull(cniPluginAttrTypes())
			},
			ExpectedOutput: &models.CNIPluginSettings{
				Type: "canal",
			},
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

func TestExpandAWSCloudSpec(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name           string
		setupList      func() types.List
		ExpectedOutput *models.AWSCloudSpec
	}{
		{
			name: "full aws spec",
			setupList: func() types.List {
				awsModel := AWSCloudSpecModel{
					AccessKeyID:            types.StringValue("AKIAIOSFODNN7EXAMPLE"),
					RoleARN:                types.StringValue("default"),
					OpenstackBillingTenant: types.StringValue("foo"),
					InstanceProfileName:    types.StringValue("default"),
					RouteTableID:           types.StringValue("rtb-09ba434c1bEXAMPLE"),
					SecretAccessKey:        types.StringValue("wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"),
					SecurityGroupID:        types.StringValue("sg-51530134"),
					VPCID:                  types.StringValue("e5e4b2ef2fe"),
				}
				objVal, _ := types.ObjectValueFrom(ctx, awsCloudSpecAttrTypes(), awsModel)
				listVal, _ := types.ListValue(types.ObjectType{AttrTypes: awsCloudSpecAttrTypes()}, []attr.Value{objVal})
				return listVal
			},
			ExpectedOutput: &models.AWSCloudSpec{
				AccessKeyID:            "AKIAIOSFODNN7EXAMPLE",
				ControlPlaneRoleARN:    "default",
				OpenstackBillingTenant: "foo",
				InstanceProfileName:    "default",
				RouteTableID:           "rtb-09ba434c1bEXAMPLE",
				SecretAccessKey:        "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
				SecurityGroupID:        "sg-51530134",
				VPCID:                  "e5e4b2ef2fe",
			},
		},
		{
			name: "empty aws spec",
			setupList: func() types.List {
				awsModel := AWSCloudSpecModel{
					AccessKeyID:            types.StringNull(),
					SecretAccessKey:        types.StringNull(),
					VPCID:                  types.StringNull(),
					SecurityGroupID:        types.StringNull(),
					RouteTableID:           types.StringNull(),
					InstanceProfileName:    types.StringNull(),
					RoleARN:                types.StringNull(),
					OpenstackBillingTenant: types.StringNull(),
				}
				objVal, _ := types.ObjectValueFrom(ctx, awsCloudSpecAttrTypes(), awsModel)
				listVal, _ := types.ListValue(types.ObjectType{AttrTypes: awsCloudSpecAttrTypes()}, []attr.Value{objVal})
				return listVal
			},
			ExpectedOutput: &models.AWSCloudSpec{},
		},
		{
			name: "null list",
			setupList: func() types.List {
				return types.ListNull(types.ObjectType{AttrTypes: awsCloudSpecAttrTypes()})
			},
			ExpectedOutput: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			list := tc.setupList()
			output := expandAWSCloudSpec(ctx, list, func(string) bool { return true })
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
		setupList      func() types.List
		ExpectedOutput *models.OpenstackCloudSpec
	}{
		{
			name: "with user credentials",
			setupList: func() types.List {
				userCredsModel := OpenstackUserCredentialsModel{
					Username:    types.StringValue("Username"),
					Password:    types.StringValue("Password"),
					ProjectID:   types.StringValue("ProjectID"),
					ProjectName: types.StringValue("ProjectName"),
				}
				userCredsObjVal, _ := types.ObjectValueFrom(ctx, openstackUserCredentialsAttrTypes(), userCredsModel)
				userCredsList, _ := types.ListValue(types.ObjectType{AttrTypes: openstackUserCredentialsAttrTypes()}, []attr.Value{userCredsObjVal})

				osModel := OpenstackCloudSpecModel{
					FloatingIPPool:         types.StringValue("FloatingIPPool"),
					ServerGroupID:          types.StringValue("ServerGroupID"),
					UserCredentials:        userCredsList,
					ApplicationCredentials: types.ListNull(types.ObjectType{AttrTypes: openstackApplicationCredentialsAttrTypes()}),
					SecurityGroup:          types.StringNull(),
					Network:                types.StringNull(),
					SubnetID:               types.StringNull(),
					SubnetCIDR:             types.StringNull(),
				}
				objVal, _ := types.ObjectValueFrom(ctx, openstackCloudSpecAttrTypes(), osModel)
				listVal, _ := types.ListValue(types.ObjectType{AttrTypes: openstackCloudSpecAttrTypes()}, []attr.Value{objVal})
				return listVal
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
			setupList: func() types.List {
				appCredsModel := OpenstackApplicationCredentialsModel{
					ID:     types.StringValue("id"),
					Secret: types.StringValue("secret"),
				}
				appCredsObjVal, _ := types.ObjectValueFrom(ctx, openstackApplicationCredentialsAttrTypes(), appCredsModel)
				appCredsList, _ := types.ListValue(types.ObjectType{AttrTypes: openstackApplicationCredentialsAttrTypes()}, []attr.Value{appCredsObjVal})

				osModel := OpenstackCloudSpecModel{
					FloatingIPPool:         types.StringValue("FloatingIPPool"),
					ServerGroupID:          types.StringValue("ServerGroupID"),
					ApplicationCredentials: appCredsList,
					UserCredentials:        types.ListNull(types.ObjectType{AttrTypes: openstackUserCredentialsAttrTypes()}),
					SecurityGroup:          types.StringNull(),
					Network:                types.StringNull(),
					SubnetID:               types.StringNull(),
					SubnetCIDR:             types.StringNull(),
				}
				objVal, _ := types.ObjectValueFrom(ctx, openstackCloudSpecAttrTypes(), osModel)
				listVal, _ := types.ListValue(types.ObjectType{AttrTypes: openstackCloudSpecAttrTypes()}, []attr.Value{objVal})
				return listVal
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
			setupList: func() types.List {
				osModel := OpenstackCloudSpecModel{
					FloatingIPPool:         types.StringNull(),
					SecurityGroup:          types.StringNull(),
					Network:                types.StringNull(),
					SubnetID:               types.StringNull(),
					SubnetCIDR:             types.StringNull(),
					ServerGroupID:          types.StringNull(),
					UserCredentials:        types.ListNull(types.ObjectType{AttrTypes: openstackUserCredentialsAttrTypes()}),
					ApplicationCredentials: types.ListNull(types.ObjectType{AttrTypes: openstackApplicationCredentialsAttrTypes()}),
				}
				objVal, _ := types.ObjectValueFrom(ctx, openstackCloudSpecAttrTypes(), osModel)
				listVal, _ := types.ListValue(types.ObjectType{AttrTypes: openstackCloudSpecAttrTypes()}, []attr.Value{objVal})
				return listVal
			},
			ExpectedOutput: &models.OpenstackCloudSpec{
				Domain: "Default",
			},
		},
		{
			name: "null list",
			setupList: func() types.List {
				return types.ListNull(types.ObjectType{AttrTypes: openstackCloudSpecAttrTypes()})
			},
			ExpectedOutput: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			list := tc.setupList()
			output := expandOpenstackCloudSpec(ctx, list, func(string) bool { return true })
			if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
				t.Fatalf("Unexpected output from expander: mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestExpandAzureCloudSpec(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name           string
		setupList      func() types.List
		ExpectedOutput *models.AzureCloudSpec
	}{
		{
			name: "full azure spec",
			setupList: func() types.List {
				azureModel := AzureCloudSpecModel{
					ClientID:               types.StringValue("ClientID"),
					ClientSecret:           types.StringValue("ClientSecret"),
					TenantID:               types.StringValue("TenantID"),
					SubscriptionID:         types.StringValue("SubscriptionID"),
					ResourceGroup:          types.StringValue("ResourceGroup"),
					RouteTable:             types.StringValue("RouteTableName"),
					SecurityGroup:          types.StringValue("SecurityGroup"),
					Subnet:                 types.StringValue("SubnetName"),
					VNet:                   types.StringValue("VNetName"),
					AvailabilitySet:        types.StringNull(),
					OpenstackBillingTenant: types.StringNull(),
				}
				objVal, _ := types.ObjectValueFrom(ctx, azureCloudSpecAttrTypes(), azureModel)
				listVal, _ := types.ListValue(types.ObjectType{AttrTypes: azureCloudSpecAttrTypes()}, []attr.Value{objVal})
				return listVal
			},
			ExpectedOutput: &models.AzureCloudSpec{
				ClientID:       "ClientID",
				ClientSecret:   "ClientSecret",
				SubscriptionID: "SubscriptionID",
				TenantID:       "TenantID",
				ResourceGroup:  "ResourceGroup",
				RouteTableName: "RouteTableName",
				SecurityGroup:  "SecurityGroup",
				SubnetName:     "SubnetName",
				VNetName:       "VNetName",
			},
		},
		{
			name: "empty azure spec",
			setupList: func() types.List {
				azureModel := AzureCloudSpecModel{
					AvailabilitySet:        types.StringNull(),
					ClientID:               types.StringNull(),
					ClientSecret:           types.StringNull(),
					SubscriptionID:         types.StringNull(),
					TenantID:               types.StringNull(),
					ResourceGroup:          types.StringNull(),
					RouteTable:             types.StringNull(),
					SecurityGroup:          types.StringNull(),
					Subnet:                 types.StringNull(),
					VNet:                   types.StringNull(),
					OpenstackBillingTenant: types.StringNull(),
				}
				objVal, _ := types.ObjectValueFrom(ctx, azureCloudSpecAttrTypes(), azureModel)
				listVal, _ := types.ListValue(types.ObjectType{AttrTypes: azureCloudSpecAttrTypes()}, []attr.Value{objVal})
				return listVal
			},
			ExpectedOutput: &models.AzureCloudSpec{},
		},
		{
			name: "null list",
			setupList: func() types.List {
				return types.ListNull(types.ObjectType{AttrTypes: azureCloudSpecAttrTypes()})
			},
			ExpectedOutput: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			list := tc.setupList()
			output := expandAzureCloudSpec(ctx, list, func(string) bool { return true })
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
		name        string
		setupModel  func() *ClusterModel
		expectAWS   *models.AWSCloudSpec
		expectOS    *clusterOpenstackPreservedValues
		expectAzure *models.AzureCloudSpec
	}{
		{
			name: "null spec returns empty values",
			setupModel: func() *ClusterModel {
				return &ClusterModel{
					Spec: types.ListNull(types.ObjectType{AttrTypes: clusterSpecAttrTypes()}),
				}
			},
			expectAWS:   nil,
			expectOS:    nil,
			expectAzure: nil,
		},
		{
			name: "model with AWS credentials preserves them",
			setupModel: func() *ClusterModel {
				return createModelWithAWSCredentials(ctx, t, "AKIATEST", "secretkey123", "vpc-123", "sg-456")
			},
			expectAWS: &models.AWSCloudSpec{
				AccessKeyID:     "AKIATEST",
				SecretAccessKey: "secretkey123",
				VPCID:           "vpc-123",
				SecurityGroupID: "sg-456",
			},
			expectOS:    nil,
			expectAzure: nil,
		},
		{
			name: "model with OpenStack user credentials preserves them",
			setupModel: func() *ClusterModel {
				return createModelWithOpenstackUserCredentials(ctx, t, "testuser", "testpass", "project-123", "myproject")
			},
			expectAWS: nil,
			expectOS: &clusterOpenstackPreservedValues{
				openstackUsername:    types.StringValue("testuser"),
				openstackPassword:    types.StringValue("testpass"),
				openstackProjectID:   types.StringValue("project-123"),
				openstackProjectName: types.StringValue("myproject"),
			},
			expectAzure: nil,
		},
		{
			name: "model with OpenStack application credentials preserves them",
			setupModel: func() *ClusterModel {
				return createModelWithOpenstackAppCredentials(ctx, t, "app-id-123", "app-secret-456")
			},
			expectAWS: nil,
			expectOS: &clusterOpenstackPreservedValues{
				openstackApplicationCredentialsID:     types.StringValue("app-id-123"),
				openstackApplicationCredentialsSecret: types.StringValue("app-secret-456"),
			},
			expectAzure: nil,
		},
		{
			name: "model with Azure credentials preserves them",
			setupModel: func() *ClusterModel {
				return createModelWithAzureCredentials(ctx, t, "client-id", "client-secret", "tenant-id", "subscription-id")
			},
			expectAWS: nil,
			expectOS:  nil,
			expectAzure: &models.AzureCloudSpec{
				ClientID:       "client-id",
				ClientSecret:   "client-secret",
				TenantID:       "tenant-id",
				SubscriptionID: "subscription-id",
			},
		},
		{
			name: "model with null cloud returns empty values",
			setupModel: func() *ClusterModel {
				specModel := ClusterSpecModel{
					Version:       types.StringValue("1.18.8"),
					Cloud:         types.ListNull(types.ObjectType{AttrTypes: clusterCloudSpecAttrTypes()}),
					UpdateWindow:  types.ListNull(types.ObjectType{AttrTypes: updateWindowAttrTypes()}),
					CNIPlugin:     createCNIPluginObject(ctx, t, "canal"),
					SyselevenAuth: types.ListNull(types.ObjectType{AttrTypes: syselevenAuthAttrTypes()}),
				}
				return createTestClusterModel(ctx, t, specModel)
			},
			expectAWS:   nil,
			expectOS:    nil,
			expectAzure: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			model := tc.setupModel()
			result := getPreservedValuesFromModel(ctx, model)

			// Check AWS
			if tc.expectAWS == nil {
				if result.aws != nil {
					t.Errorf("Expected nil AWS, got %+v", result.aws)
				}
			} else {
				if result.aws == nil {
					t.Fatal("Expected AWS to be set, got nil")
				}
				if result.aws.AccessKeyID != tc.expectAWS.AccessKeyID {
					t.Errorf("AWS AccessKeyID mismatch: got %v, want %v", result.aws.AccessKeyID, tc.expectAWS.AccessKeyID)
				}
				if result.aws.SecretAccessKey != tc.expectAWS.SecretAccessKey {
					t.Errorf("AWS SecretAccessKey mismatch: got %v, want %v", result.aws.SecretAccessKey, tc.expectAWS.SecretAccessKey)
				}
				if result.aws.VPCID != tc.expectAWS.VPCID {
					t.Errorf("AWS VPCID mismatch: got %v, want %v", result.aws.VPCID, tc.expectAWS.VPCID)
				}
				if result.aws.SecurityGroupID != tc.expectAWS.SecurityGroupID {
					t.Errorf("AWS SecurityGroupID mismatch: got %v, want %v", result.aws.SecurityGroupID, tc.expectAWS.SecurityGroupID)
				}
			}

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

			// Check Azure
			if tc.expectAzure == nil {
				if result.azure != nil {
					t.Errorf("Expected nil Azure, got %+v", result.azure)
				}
			} else {
				if result.azure == nil {
					t.Fatal("Expected Azure to be set, got nil")
				}
				if result.azure.ClientID != tc.expectAzure.ClientID {
					t.Errorf("Azure ClientID mismatch: got %v, want %v", result.azure.ClientID, tc.expectAzure.ClientID)
				}
				if result.azure.ClientSecret != tc.expectAzure.ClientSecret {
					t.Errorf("Azure ClientSecret mismatch: got %v, want %v", result.azure.ClientSecret, tc.expectAzure.ClientSecret)
				}
				if result.azure.TenantID != tc.expectAzure.TenantID {
					t.Errorf("Azure TenantID mismatch: got %v, want %v", result.azure.TenantID, tc.expectAzure.TenantID)
				}
				if result.azure.SubscriptionID != tc.expectAzure.SubscriptionID {
					t.Errorf("Azure SubscriptionID mismatch: got %v, want %v", result.azure.SubscriptionID, tc.expectAzure.SubscriptionID)
				}
			}
		})
	}
}

func TestFlattenClusterCloudSpecWithAWSPreservedValues(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name                  string
		apiResponse           *models.CloudSpec
		preservedValues       clusterPreserveValues
		expectedAccessKeyID   string
		expectedSecretKey     string
		expectedVPCID         string
		expectedSecurityGroup string
	}{
		{
			name: "API returns empty credentials, preserved values used",
			apiResponse: &models.CloudSpec{
				Aws: &models.AWSCloudSpec{
					// API returns empty credentials (sensitive data not returned)
					VPCID: "api-vpc-id",
				},
			},
			preservedValues: clusterPreserveValues{
				aws: &models.AWSCloudSpec{
					AccessKeyID:     "preserved-access-key",
					SecretAccessKey: "preserved-secret-key",
					VPCID:           "preserved-vpc-id",
					SecurityGroupID: "preserved-sg-id",
				},
			},
			expectedAccessKeyID:   "preserved-access-key",
			expectedSecretKey:     "preserved-secret-key",
			expectedVPCID:         "preserved-vpc-id",
			expectedSecurityGroup: "preserved-sg-id",
		},
		{
			name: "no preserved values, API values used",
			apiResponse: &models.CloudSpec{
				Aws: &models.AWSCloudSpec{
					AccessKeyID:     "api-access-key",
					SecretAccessKey: "api-secret-key",
					VPCID:           "api-vpc-id",
					SecurityGroupID: "api-sg-id",
				},
			},
			preservedValues:       clusterPreserveValues{},
			expectedAccessKeyID:   "api-access-key",
			expectedSecretKey:     "api-secret-key",
			expectedVPCID:         "api-vpc-id",
			expectedSecurityGroup: "api-sg-id",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			specModel := &ClusterSpecModel{}
			diags := flattenClusterCloudSpec(ctx, specModel, tc.preservedValues, tc.apiResponse)
			if diags.HasError() {
				t.Fatalf("Unexpected error: %v", diags)
			}

			var clouds []ClusterCloudSpecModel
			if d := specModel.Cloud.ElementsAs(ctx, &clouds, false); d.HasError() {
				t.Fatalf("Failed to get cloud elements: %v", d)
			}
			if len(clouds) == 0 {
				t.Fatal("Expected cloud list to have elements")
			}

			var awsSpecs []AWSCloudSpecModel
			if d := clouds[0].AWS.ElementsAs(ctx, &awsSpecs, false); d.HasError() {
				t.Fatalf("Failed to get AWS elements: %v", d)
			}
			if len(awsSpecs) == 0 {
				t.Fatal("Expected AWS list to have elements")
			}

			aws := awsSpecs[0]
			if aws.AccessKeyID.ValueString() != tc.expectedAccessKeyID {
				t.Errorf("AccessKeyID mismatch: got %v, want %v", aws.AccessKeyID.ValueString(), tc.expectedAccessKeyID)
			}
			if aws.SecretAccessKey.ValueString() != tc.expectedSecretKey {
				t.Errorf("SecretAccessKey mismatch: got %v, want %v", aws.SecretAccessKey.ValueString(), tc.expectedSecretKey)
			}
			if aws.VPCID.ValueString() != tc.expectedVPCID {
				t.Errorf("VPCID mismatch: got %v, want %v", aws.VPCID.ValueString(), tc.expectedVPCID)
			}
			if aws.SecurityGroupID.ValueString() != tc.expectedSecurityGroup {
				t.Errorf("SecurityGroupID mismatch: got %v, want %v", aws.SecurityGroupID.ValueString(), tc.expectedSecurityGroup)
			}
		})
	}
}

func TestFlattenClusterCloudSpecWithAzurePreservedValues(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name                   string
		apiResponse            *models.CloudSpec
		preservedValues        clusterPreserveValues
		expectedClientID       string
		expectedClientSecret   string
		expectedTenantID       string
		expectedSubscriptionID string
	}{
		{
			name: "API returns empty credentials, preserved values used",
			apiResponse: &models.CloudSpec{
				Azure: &models.AzureCloudSpec{
					// API returns empty credentials (sensitive data not returned)
					ResourceGroup: "api-rg",
				},
			},
			preservedValues: clusterPreserveValues{
				azure: &models.AzureCloudSpec{
					ClientID:       "preserved-client-id",
					ClientSecret:   "preserved-client-secret",
					TenantID:       "preserved-tenant-id",
					SubscriptionID: "preserved-subscription-id",
					ResourceGroup:  "preserved-rg",
				},
			},
			expectedClientID:       "preserved-client-id",
			expectedClientSecret:   "preserved-client-secret",
			expectedTenantID:       "preserved-tenant-id",
			expectedSubscriptionID: "preserved-subscription-id",
		},
		{
			name: "no preserved values, API values used",
			apiResponse: &models.CloudSpec{
				Azure: &models.AzureCloudSpec{
					ClientID:       "api-client-id",
					ClientSecret:   "api-client-secret",
					TenantID:       "api-tenant-id",
					SubscriptionID: "api-subscription-id",
				},
			},
			preservedValues:        clusterPreserveValues{},
			expectedClientID:       "api-client-id",
			expectedClientSecret:   "api-client-secret",
			expectedTenantID:       "api-tenant-id",
			expectedSubscriptionID: "api-subscription-id",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			specModel := &ClusterSpecModel{}
			diags := flattenClusterCloudSpec(ctx, specModel, tc.preservedValues, tc.apiResponse)
			if diags.HasError() {
				t.Fatalf("Unexpected error: %v", diags)
			}

			var clouds []ClusterCloudSpecModel
			if d := specModel.Cloud.ElementsAs(ctx, &clouds, false); d.HasError() {
				t.Fatalf("Failed to get cloud elements: %v", d)
			}
			if len(clouds) == 0 {
				t.Fatal("Expected cloud list to have elements")
			}

			var azureSpecs []AzureCloudSpecModel
			if d := clouds[0].Azure.ElementsAs(ctx, &azureSpecs, false); d.HasError() {
				t.Fatalf("Failed to get Azure elements: %v", d)
			}
			if len(azureSpecs) == 0 {
				t.Fatal("Expected Azure list to have elements")
			}

			azure := azureSpecs[0]
			if azure.ClientID.ValueString() != tc.expectedClientID {
				t.Errorf("ClientID mismatch: got %v, want %v", azure.ClientID.ValueString(), tc.expectedClientID)
			}
			if azure.ClientSecret.ValueString() != tc.expectedClientSecret {
				t.Errorf("ClientSecret mismatch: got %v, want %v", azure.ClientSecret.ValueString(), tc.expectedClientSecret)
			}
			if azure.TenantID.ValueString() != tc.expectedTenantID {
				t.Errorf("TenantID mismatch: got %v, want %v", azure.TenantID.ValueString(), tc.expectedTenantID)
			}
			if azure.SubscriptionID.ValueString() != tc.expectedSubscriptionID {
				t.Errorf("SubscriptionID mismatch: got %v, want %v", azure.SubscriptionID.ValueString(), tc.expectedSubscriptionID)
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
				AWS:       types.ListNull(types.ObjectType{AttrTypes: awsCloudSpecAttrTypes()}),
				Openstack: types.ListNull(types.ObjectType{AttrTypes: openstackCloudSpecAttrTypes()}),
				Azure:     types.ListNull(types.ObjectType{AttrTypes: azureCloudSpecAttrTypes()}),
			}
			diags := flattenOpenstackSpec(ctx, cloudModel, tc.preservedValues, tc.apiResponse)
			if diags.HasError() {
				t.Fatalf("Unexpected error: %v", diags)
			}

			var osSpecs []OpenstackCloudSpecModel
			if d := cloudModel.Openstack.ElementsAs(ctx, &osSpecs, false); d.HasError() {
				t.Fatalf("Failed to get Openstack elements: %v", d)
			}
			if len(osSpecs) == 0 {
				t.Fatal("Expected Openstack list to have elements")
			}

			os := osSpecs[0]

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
				var userCreds []OpenstackUserCredentialsModel
				if d := os.UserCredentials.ElementsAs(ctx, &userCreds, false); d.HasError() {
					t.Fatalf("Failed to get UserCredentials elements: %v", d)
				}
				if len(userCreds) == 0 {
					t.Fatal("Expected UserCredentials list to have elements")
				}
				if tc.expectedUsername != "" && userCreds[0].Username.ValueString() != tc.expectedUsername {
					t.Errorf("Username mismatch: got %v, want %v", userCreds[0].Username.ValueString(), tc.expectedUsername)
				}
				if tc.expectedPassword != "" && userCreds[0].Password.ValueString() != tc.expectedPassword {
					t.Errorf("Password mismatch: got %v, want %v", userCreds[0].Password.ValueString(), tc.expectedPassword)
				}
				if tc.expectedProjectID != "" && userCreds[0].ProjectID.ValueString() != tc.expectedProjectID {
					t.Errorf("ProjectID mismatch: got %v, want %v", userCreds[0].ProjectID.ValueString(), tc.expectedProjectID)
				}
				if tc.expectedProjectName != "" && userCreds[0].ProjectName.ValueString() != tc.expectedProjectName {
					t.Errorf("ProjectName mismatch: got %v, want %v", userCreds[0].ProjectName.ValueString(), tc.expectedProjectName)
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
				var appCreds []OpenstackApplicationCredentialsModel
				if d := os.ApplicationCredentials.ElementsAs(ctx, &appCreds, false); d.HasError() {
					t.Fatalf("Failed to get ApplicationCredentials elements: %v", d)
				}
				if len(appCreds) == 0 {
					t.Fatal("Expected ApplicationCredentials list to have elements")
				}
				if appCreds[0].ID.ValueString() != tc.expectedAppCredID {
					t.Errorf("AppCred ID mismatch: got %v, want %v", appCreds[0].ID.ValueString(), tc.expectedAppCredID)
				}
				if appCreds[0].Secret.ValueString() != tc.expectedAppCredSecret {
					t.Errorf("AppCred Secret mismatch: got %v, want %v", appCreds[0].Secret.ValueString(), tc.expectedAppCredSecret)
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
			name: "AWS credentials preserved through full flatten",
			setupModel: func() *ClusterModel {
				return createModelWithAWSCredentials(ctx, t, "state-access-key", "state-secret-key", "state-vpc", "state-sg")
			},
			apiSpec: &models.ClusterSpec{
				Version: "1.20.0",
				Cloud: &models.CloudSpec{
					DatacenterName: "dc1",
					Aws: &models.AWSCloudSpec{
						// API returns empty credentials
						VPCID: "api-vpc",
					},
				},
			},
			verifyPreserved: func(t *testing.T, model *ClusterModel) {
				var specs []ClusterSpecModel
				model.Spec.ElementsAs(ctx, &specs, false)
				var clouds []ClusterCloudSpecModel
				specs[0].Cloud.ElementsAs(ctx, &clouds, false)
				var awsSpecs []AWSCloudSpecModel
				clouds[0].AWS.ElementsAs(ctx, &awsSpecs, false)

				if awsSpecs[0].AccessKeyID.ValueString() != "state-access-key" {
					t.Errorf("AccessKeyID not preserved: got %v, want state-access-key", awsSpecs[0].AccessKeyID.ValueString())
				}
				if awsSpecs[0].SecretAccessKey.ValueString() != "state-secret-key" {
					t.Errorf("SecretAccessKey not preserved: got %v, want state-secret-key", awsSpecs[0].SecretAccessKey.ValueString())
				}
			},
		},
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
				var specs []ClusterSpecModel
				model.Spec.ElementsAs(ctx, &specs, false)
				var clouds []ClusterCloudSpecModel
				specs[0].Cloud.ElementsAs(ctx, &clouds, false)
				var osSpecs []OpenstackCloudSpecModel
				clouds[0].Openstack.ElementsAs(ctx, &osSpecs, false)

				if osSpecs[0].UserCredentials.IsNull() {
					t.Fatal("UserCredentials should not be null")
				}
				var userCreds []OpenstackUserCredentialsModel
				osSpecs[0].UserCredentials.ElementsAs(ctx, &userCreds, false)
				if userCreds[0].Username.ValueString() != "state-user" {
					t.Errorf("Username not preserved: got %v, want state-user", userCreds[0].Username.ValueString())
				}
				if userCreds[0].Password.ValueString() != "state-pass" {
					t.Errorf("Password not preserved: got %v, want state-pass", userCreds[0].Password.ValueString())
				}
			},
		},
		{
			name: "Azure credentials preserved through full flatten",
			setupModel: func() *ClusterModel {
				return createModelWithAzureCredentials(ctx, t, "state-client-id", "state-client-secret", "state-tenant", "state-sub")
			},
			apiSpec: &models.ClusterSpec{
				Version: "1.20.0",
				Cloud: &models.CloudSpec{
					DatacenterName: "dc1",
					Azure: &models.AzureCloudSpec{
						ResourceGroup: "api-rg",
						// API returns empty credentials
					},
				},
			},
			verifyPreserved: func(t *testing.T, model *ClusterModel) {
				var specs []ClusterSpecModel
				model.Spec.ElementsAs(ctx, &specs, false)
				var clouds []ClusterCloudSpecModel
				specs[0].Cloud.ElementsAs(ctx, &clouds, false)
				var azureSpecs []AzureCloudSpecModel
				clouds[0].Azure.ElementsAs(ctx, &azureSpecs, false)

				if azureSpecs[0].ClientID.ValueString() != "state-client-id" {
					t.Errorf("ClientID not preserved: got %v, want state-client-id", azureSpecs[0].ClientID.ValueString())
				}
				if azureSpecs[0].ClientSecret.ValueString() != "state-client-secret" {
					t.Errorf("ClientSecret not preserved: got %v, want state-client-secret", azureSpecs[0].ClientSecret.ValueString())
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
// is always populated from the API response (defaulting to canal when API returns nil/empty/none).
func TestFlattenSpecIntoModelPopulatesCNIPluginFromAPI(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name        string
		setupModel  func() *ClusterModel
		apiSpec     *models.ClusterSpec
		expectedCNI string
	}{
		{
			name: "Config has no CNI block, API returns cilium - state should have cilium",
			setupModel: func() *ClusterModel {
				specModel := ClusterSpecModel{
					Version:       types.StringValue("1.20.0"),
					CNIPlugin:     createCNIPluginObject(ctx, t, "canal"),
					Cloud:         createOpenstackCloudList(ctx, t),
					UpdateWindow:  types.ListNull(types.ObjectType{AttrTypes: updateWindowAttrTypes()}),
					SyselevenAuth: types.ListNull(types.ObjectType{AttrTypes: syselevenAuthAttrTypes()}),
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
					UpdateWindow:  types.ListNull(types.ObjectType{AttrTypes: updateWindowAttrTypes()}),
					SyselevenAuth: types.ListNull(types.ObjectType{AttrTypes: syselevenAuthAttrTypes()}),
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
			name: "Config has CNI block with cilium, API returns nil - state should default to canal",
			setupModel: func() *ClusterModel {
				specModel := ClusterSpecModel{
					Version:       types.StringValue("1.20.0"),
					CNIPlugin:     createCNIPluginObject(ctx, t, "cilium"),
					Cloud:         createOpenstackCloudList(ctx, t),
					UpdateWindow:  types.ListNull(types.ObjectType{AttrTypes: updateWindowAttrTypes()}),
					SyselevenAuth: types.ListNull(types.ObjectType{AttrTypes: syselevenAuthAttrTypes()}),
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
			expectedCNI: "canal", // When API returns nil, we default to canal
		},
		{
			name: "Config has CNI block with cilium, API returns cilium - state should have cilium",
			setupModel: func() *ClusterModel {
				specModel := ClusterSpecModel{
					Version:       types.StringValue("1.20.0"),
					CNIPlugin:     createCNIPluginObject(ctx, t, "cilium"),
					Cloud:         createOpenstackCloudList(ctx, t),
					UpdateWindow:  types.ListNull(types.ObjectType{AttrTypes: updateWindowAttrTypes()}),
					SyselevenAuth: types.ListNull(types.ObjectType{AttrTypes: syselevenAuthAttrTypes()}),
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

			var specs []ClusterSpecModel
			if d := model.Spec.ElementsAs(ctx, &specs, false); d.HasError() {
				t.Fatalf("Failed to get spec elements: %v", d)
			}
			if len(specs) == 0 {
				t.Fatal("Expected spec list to have elements")
			}

			// CNI plugin should always exist
			if specs[0].CNIPlugin.IsNull() {
				t.Fatal("Expected CNI plugin to be set, got null")
			}
			var plugin CNIPluginModel
			if d := specs[0].CNIPlugin.As(ctx, &plugin, basetypes.ObjectAsOptions{}); d.HasError() {
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
	specList, err := types.ListValue(types.ObjectType{AttrTypes: clusterSpecAttrTypes()}, []attr.Value{specObjVal})
	if err != nil {
		t.Fatalf("Failed to create spec list: %v", err)
	}
	return &ClusterModel{
		Spec: specList,
	}
}

func createUpdateWindowList(ctx context.Context, t *testing.T, start, length string) types.List {
	t.Helper()
	uwModel := UpdateWindowModel{
		Start:  types.StringValue(start),
		Length: types.StringValue(length),
	}
	objVal, _ := types.ObjectValueFrom(ctx, updateWindowAttrTypes(), uwModel)
	listVal, _ := types.ListValue(types.ObjectType{AttrTypes: updateWindowAttrTypes()}, []attr.Value{objVal})
	return listVal
}

func createCNIPluginObject(ctx context.Context, t *testing.T, pluginType string) types.Object {
	t.Helper()
	cniModel := CNIPluginModel{
		Type: types.StringValue(pluginType),
	}
	objVal, _ := types.ObjectValueFrom(ctx, cniPluginAttrTypes(), cniModel)
	return objVal
}

func createSyselevenAuthList(ctx context.Context, t *testing.T, realm string) types.List {
	t.Helper()
	authModel := SyselevenAuthModel{
		Realm:             types.StringValue(realm),
		IAMAuthentication: types.BoolNull(),
	}
	objVal, _ := types.ObjectValueFrom(ctx, syselevenAuthAttrTypes(), authModel)
	listVal, _ := types.ListValue(types.ObjectType{AttrTypes: syselevenAuthAttrTypes()}, []attr.Value{objVal})
	return listVal
}

func createOpenstackCloudList(ctx context.Context, t *testing.T) types.List {
	t.Helper()
	osModel := OpenstackCloudSpecModel{
		FloatingIPPool:         types.StringNull(),
		SecurityGroup:          types.StringNull(),
		Network:                types.StringNull(),
		SubnetID:               types.StringNull(),
		SubnetCIDR:             types.StringNull(),
		ServerGroupID:          types.StringNull(),
		UserCredentials:        types.ListNull(types.ObjectType{AttrTypes: openstackUserCredentialsAttrTypes()}),
		ApplicationCredentials: types.ListNull(types.ObjectType{AttrTypes: openstackApplicationCredentialsAttrTypes()}),
	}
	osObjVal, _ := types.ObjectValueFrom(ctx, openstackCloudSpecAttrTypes(), osModel)
	osListVal, _ := types.ListValue(types.ObjectType{AttrTypes: openstackCloudSpecAttrTypes()}, []attr.Value{osObjVal})

	cloudModel := ClusterCloudSpecModel{
		AWS:       types.ListNull(types.ObjectType{AttrTypes: awsCloudSpecAttrTypes()}),
		Openstack: osListVal,
		Azure:     types.ListNull(types.ObjectType{AttrTypes: azureCloudSpecAttrTypes()}),
	}
	cloudObjVal, _ := types.ObjectValueFrom(ctx, clusterCloudSpecAttrTypes(), cloudModel)
	cloudListVal, _ := types.ListValue(types.ObjectType{AttrTypes: clusterCloudSpecAttrTypes()}, []attr.Value{cloudObjVal})
	return cloudListVal
}

func createAWSCloudList(ctx context.Context, t *testing.T) types.List {
	t.Helper()
	awsModel := AWSCloudSpecModel{
		AccessKeyID:            types.StringNull(),
		SecretAccessKey:        types.StringNull(),
		VPCID:                  types.StringNull(),
		SecurityGroupID:        types.StringNull(),
		RouteTableID:           types.StringNull(),
		InstanceProfileName:    types.StringNull(),
		RoleARN:                types.StringNull(),
		OpenstackBillingTenant: types.StringNull(),
	}
	awsObjVal, _ := types.ObjectValueFrom(ctx, awsCloudSpecAttrTypes(), awsModel)
	awsListVal, _ := types.ListValue(types.ObjectType{AttrTypes: awsCloudSpecAttrTypes()}, []attr.Value{awsObjVal})

	cloudModel := ClusterCloudSpecModel{
		AWS:       awsListVal,
		Openstack: types.ListNull(types.ObjectType{AttrTypes: openstackCloudSpecAttrTypes()}),
		Azure:     types.ListNull(types.ObjectType{AttrTypes: azureCloudSpecAttrTypes()}),
	}
	cloudObjVal, _ := types.ObjectValueFrom(ctx, clusterCloudSpecAttrTypes(), cloudModel)
	cloudListVal, _ := types.ListValue(types.ObjectType{AttrTypes: clusterCloudSpecAttrTypes()}, []attr.Value{cloudObjVal})
	return cloudListVal
}

// Helper functions for creating models with credentials for preserved values tests

func createModelWithAWSCredentials(ctx context.Context, t *testing.T, accessKeyID, secretKey, vpcID, securityGroupID string) *ClusterModel {
	t.Helper()
	awsModel := AWSCloudSpecModel{
		AccessKeyID:            types.StringValue(accessKeyID),
		SecretAccessKey:        types.StringValue(secretKey),
		VPCID:                  types.StringValue(vpcID),
		SecurityGroupID:        types.StringValue(securityGroupID),
		RouteTableID:           types.StringNull(),
		InstanceProfileName:    types.StringNull(),
		RoleARN:                types.StringNull(),
		OpenstackBillingTenant: types.StringNull(),
	}
	awsObjVal, _ := types.ObjectValueFrom(ctx, awsCloudSpecAttrTypes(), awsModel)
	awsListVal, _ := types.ListValue(types.ObjectType{AttrTypes: awsCloudSpecAttrTypes()}, []attr.Value{awsObjVal})

	cloudModel := ClusterCloudSpecModel{
		AWS:       awsListVal,
		Openstack: types.ListNull(types.ObjectType{AttrTypes: openstackCloudSpecAttrTypes()}),
		Azure:     types.ListNull(types.ObjectType{AttrTypes: azureCloudSpecAttrTypes()}),
	}
	cloudObjVal, _ := types.ObjectValueFrom(ctx, clusterCloudSpecAttrTypes(), cloudModel)
	cloudListVal, _ := types.ListValue(types.ObjectType{AttrTypes: clusterCloudSpecAttrTypes()}, []attr.Value{cloudObjVal})

	specModel := ClusterSpecModel{
		Version:           types.StringValue("1.20.0"),
		EnableSSHAgent:    types.BoolNull(),
		AuditLogging:      types.BoolNull(),
		PodSecurityPolicy: types.BoolNull(),
		PodNodeSelector:   types.BoolNull(),
		ServicesCIDR:      types.StringNull(),
		PodsCIDR:          types.StringNull(),
		IPFamily:          types.StringNull(),
		UpdateWindow:      types.ListNull(types.ObjectType{AttrTypes: updateWindowAttrTypes()}),
		CNIPlugin:         createCNIPluginObject(ctx, t, "canal"),
		Cloud:             cloudListVal,
		SyselevenAuth:     types.ListNull(types.ObjectType{AttrTypes: syselevenAuthAttrTypes()}),
	}
	return createTestClusterModel(ctx, t, specModel)
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
	userCredsList, _ := types.ListValue(types.ObjectType{AttrTypes: openstackUserCredentialsAttrTypes()}, []attr.Value{userCredsObjVal})

	osModel := OpenstackCloudSpecModel{
		FloatingIPPool:         types.StringNull(),
		SecurityGroup:          types.StringNull(),
		Network:                types.StringNull(),
		SubnetID:               types.StringNull(),
		SubnetCIDR:             types.StringNull(),
		ServerGroupID:          types.StringNull(),
		UserCredentials:        userCredsList,
		ApplicationCredentials: types.ListNull(types.ObjectType{AttrTypes: openstackApplicationCredentialsAttrTypes()}),
	}
	osObjVal, _ := types.ObjectValueFrom(ctx, openstackCloudSpecAttrTypes(), osModel)
	osListVal, _ := types.ListValue(types.ObjectType{AttrTypes: openstackCloudSpecAttrTypes()}, []attr.Value{osObjVal})

	cloudModel := ClusterCloudSpecModel{
		AWS:       types.ListNull(types.ObjectType{AttrTypes: awsCloudSpecAttrTypes()}),
		Openstack: osListVal,
		Azure:     types.ListNull(types.ObjectType{AttrTypes: azureCloudSpecAttrTypes()}),
	}
	cloudObjVal, _ := types.ObjectValueFrom(ctx, clusterCloudSpecAttrTypes(), cloudModel)
	cloudListVal, _ := types.ListValue(types.ObjectType{AttrTypes: clusterCloudSpecAttrTypes()}, []attr.Value{cloudObjVal})

	specModel := ClusterSpecModel{
		Version:           types.StringValue("1.20.0"),
		EnableSSHAgent:    types.BoolNull(),
		AuditLogging:      types.BoolNull(),
		PodSecurityPolicy: types.BoolNull(),
		PodNodeSelector:   types.BoolNull(),
		ServicesCIDR:      types.StringNull(),
		PodsCIDR:          types.StringNull(),
		IPFamily:          types.StringNull(),
		UpdateWindow:      types.ListNull(types.ObjectType{AttrTypes: updateWindowAttrTypes()}),
		CNIPlugin:         createCNIPluginObject(ctx, t, "canal"),
		Cloud:             cloudListVal,
		SyselevenAuth:     types.ListNull(types.ObjectType{AttrTypes: syselevenAuthAttrTypes()}),
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
	appCredsList, _ := types.ListValue(types.ObjectType{AttrTypes: openstackApplicationCredentialsAttrTypes()}, []attr.Value{appCredsObjVal})

	osModel := OpenstackCloudSpecModel{
		FloatingIPPool:         types.StringNull(),
		SecurityGroup:          types.StringNull(),
		Network:                types.StringNull(),
		SubnetID:               types.StringNull(),
		SubnetCIDR:             types.StringNull(),
		ServerGroupID:          types.StringNull(),
		UserCredentials:        types.ListNull(types.ObjectType{AttrTypes: openstackUserCredentialsAttrTypes()}),
		ApplicationCredentials: appCredsList,
	}
	osObjVal, _ := types.ObjectValueFrom(ctx, openstackCloudSpecAttrTypes(), osModel)
	osListVal, _ := types.ListValue(types.ObjectType{AttrTypes: openstackCloudSpecAttrTypes()}, []attr.Value{osObjVal})

	cloudModel := ClusterCloudSpecModel{
		AWS:       types.ListNull(types.ObjectType{AttrTypes: awsCloudSpecAttrTypes()}),
		Openstack: osListVal,
		Azure:     types.ListNull(types.ObjectType{AttrTypes: azureCloudSpecAttrTypes()}),
	}
	cloudObjVal, _ := types.ObjectValueFrom(ctx, clusterCloudSpecAttrTypes(), cloudModel)
	cloudListVal, _ := types.ListValue(types.ObjectType{AttrTypes: clusterCloudSpecAttrTypes()}, []attr.Value{cloudObjVal})

	specModel := ClusterSpecModel{
		Version:           types.StringValue("1.20.0"),
		EnableSSHAgent:    types.BoolNull(),
		AuditLogging:      types.BoolNull(),
		PodSecurityPolicy: types.BoolNull(),
		PodNodeSelector:   types.BoolNull(),
		ServicesCIDR:      types.StringNull(),
		PodsCIDR:          types.StringNull(),
		IPFamily:          types.StringNull(),
		UpdateWindow:      types.ListNull(types.ObjectType{AttrTypes: updateWindowAttrTypes()}),
		CNIPlugin:         createCNIPluginObject(ctx, t, "canal"),
		Cloud:             cloudListVal,
		SyselevenAuth:     types.ListNull(types.ObjectType{AttrTypes: syselevenAuthAttrTypes()}),
	}
	return createTestClusterModel(ctx, t, specModel)
}

func createModelWithAzureCredentials(ctx context.Context, t *testing.T, clientID, clientSecret, tenantID, subscriptionID string) *ClusterModel {
	t.Helper()
	azureModel := AzureCloudSpecModel{
		AvailabilitySet:        types.StringNull(),
		ClientID:               types.StringValue(clientID),
		ClientSecret:           types.StringValue(clientSecret),
		TenantID:               types.StringValue(tenantID),
		SubscriptionID:         types.StringValue(subscriptionID),
		ResourceGroup:          types.StringNull(),
		RouteTable:             types.StringNull(),
		SecurityGroup:          types.StringNull(),
		Subnet:                 types.StringNull(),
		VNet:                   types.StringNull(),
		OpenstackBillingTenant: types.StringNull(),
	}
	azureObjVal, _ := types.ObjectValueFrom(ctx, azureCloudSpecAttrTypes(), azureModel)
	azureListVal, _ := types.ListValue(types.ObjectType{AttrTypes: azureCloudSpecAttrTypes()}, []attr.Value{azureObjVal})

	cloudModel := ClusterCloudSpecModel{
		AWS:       types.ListNull(types.ObjectType{AttrTypes: awsCloudSpecAttrTypes()}),
		Openstack: types.ListNull(types.ObjectType{AttrTypes: openstackCloudSpecAttrTypes()}),
		Azure:     azureListVal,
	}
	cloudObjVal, _ := types.ObjectValueFrom(ctx, clusterCloudSpecAttrTypes(), cloudModel)
	cloudListVal, _ := types.ListValue(types.ObjectType{AttrTypes: clusterCloudSpecAttrTypes()}, []attr.Value{cloudObjVal})

	specModel := ClusterSpecModel{
		Version:           types.StringValue("1.20.0"),
		EnableSSHAgent:    types.BoolNull(),
		AuditLogging:      types.BoolNull(),
		PodSecurityPolicy: types.BoolNull(),
		PodNodeSelector:   types.BoolNull(),
		ServicesCIDR:      types.StringNull(),
		PodsCIDR:          types.StringNull(),
		IPFamily:          types.StringNull(),
		UpdateWindow:      types.ListNull(types.ObjectType{AttrTypes: updateWindowAttrTypes()}),
		CNIPlugin:         createCNIPluginObject(ctx, t, "canal"),
		Cloud:             cloudListVal,
		SyselevenAuth:     types.ListNull(types.ObjectType{AttrTypes: syselevenAuthAttrTypes()}),
	}
	return createTestClusterModel(ctx, t, specModel)
}
