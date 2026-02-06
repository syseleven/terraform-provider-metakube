package resource_node_deployment

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type NodeDeploymentModel struct {
	ID                types.String   `tfsdk:"id"`
	ProjectID         types.String   `tfsdk:"project_id"`
	ClusterID         types.String   `tfsdk:"cluster_id"`
	Name              types.String   `tfsdk:"name"`
	Spec              types.List     `tfsdk:"spec"`
	CreationTimestamp types.String   `tfsdk:"creation_timestamp"`
	DeletionTimestamp types.String   `tfsdk:"deletion_timestamp"`
	Timeouts          timeouts.Value `tfsdk:"timeouts"`
}

type NodeDeploymentSpecModel struct {
	Replicas    types.Int64 `tfsdk:"replicas"`
	MinReplicas types.Int64 `tfsdk:"min_replicas"`
	MaxReplicas types.Int64 `tfsdk:"max_replicas"`
	Template    types.List  `tfsdk:"template"`
}

type NodeSpecModel struct {
	Cloud              types.List `tfsdk:"cloud"`
	OperatingSystem    types.List `tfsdk:"operating_system"`
	Versions           types.List `tfsdk:"versions"`
	Labels             types.Map  `tfsdk:"labels"`
	AllLabels          types.Map  `tfsdk:"all_labels"`
	Taints             types.List `tfsdk:"taints"`
	NodeAnnotations    types.Map  `tfsdk:"node_annotations"`
	MachineAnnotations types.Map  `tfsdk:"machine_annotations"`
}

type CloudSpecModel struct {
	AWS       types.List `tfsdk:"aws"`
	OpenStack types.List `tfsdk:"openstack"`
}

type AWSCloudSpecModel struct {
	InstanceType     types.String `tfsdk:"instance_type"`
	DiskSize         types.Int64  `tfsdk:"disk_size"`
	VolumeType       types.String `tfsdk:"volume_type"`
	AvailabilityZone types.String `tfsdk:"availability_zone"`
	SubnetID         types.String `tfsdk:"subnet_id"`
	AssignPublicIP   types.Bool   `tfsdk:"assign_public_ip"`
	AMI              types.String `tfsdk:"ami"`
	Tags             types.Map    `tfsdk:"tags"`
}

type OpenStackCloudSpecModel struct {
	Flavor                    types.String `tfsdk:"flavor"`
	Image                     types.String `tfsdk:"image"`
	DiskSize                  types.Int64  `tfsdk:"disk_size"`
	Tags                      types.Map    `tfsdk:"tags"`
	UseFloatingIP             types.Bool   `tfsdk:"use_floating_ip"`
	InstanceReadyCheckPeriod  types.String `tfsdk:"instance_ready_check_period"`
	InstanceReadyCheckTimeout types.String `tfsdk:"instance_ready_check_timeout"`
	ServerGroupID             types.String `tfsdk:"server_group_id"`
}

type OperatingSystemModel struct {
	Ubuntu  types.List `tfsdk:"ubuntu"`
	Flatcar types.List `tfsdk:"flatcar"`
}

type UbuntuModel struct {
	DistUpgradeOnBoot types.Bool `tfsdk:"dist_upgrade_on_boot"`
}

type FlatcarModel struct {
	DisableAutoUpdate types.Bool `tfsdk:"disable_auto_update"`
}

type VersionsModel struct {
	Kubelet types.String `tfsdk:"kubelet"`
}

type TaintModel struct {
	Effect types.String `tfsdk:"effect"`
	Key    types.String `tfsdk:"key"`
	Value  types.String `tfsdk:"value"`
}

// Attr type helpers for constructing types.Object and types.List values

func nodeDeploymentSpecAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"replicas":     types.Int64Type,
		"min_replicas": types.Int64Type,
		"max_replicas": types.Int64Type,
		"template":     types.ListType{ElemType: types.ObjectType{AttrTypes: nodeSpecAttrTypes()}},
	}
}

func nodeSpecAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"cloud":               types.ListType{ElemType: types.ObjectType{AttrTypes: cloudSpecAttrTypes()}},
		"operating_system":    types.ListType{ElemType: types.ObjectType{AttrTypes: operatingSystemAttrTypes()}},
		"versions":            types.ListType{ElemType: types.ObjectType{AttrTypes: versionsAttrTypes()}},
		"labels":              types.MapType{ElemType: types.StringType},
		"all_labels":          types.MapType{ElemType: types.StringType},
		"taints":              types.ListType{ElemType: types.ObjectType{AttrTypes: taintAttrTypes()}},
		"node_annotations":    types.MapType{ElemType: types.StringType},
		"machine_annotations": types.MapType{ElemType: types.StringType},
	}
}

func cloudSpecAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"aws":       types.ListType{ElemType: types.ObjectType{AttrTypes: awsCloudSpecAttrTypes()}},
		"openstack": types.ListType{ElemType: types.ObjectType{AttrTypes: openstackCloudSpecAttrTypes()}},
	}
}

func awsCloudSpecAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"instance_type":     types.StringType,
		"disk_size":         types.Int64Type,
		"volume_type":       types.StringType,
		"availability_zone": types.StringType,
		"subnet_id":         types.StringType,
		"assign_public_ip":  types.BoolType,
		"ami":               types.StringType,
		"tags":              types.MapType{ElemType: types.StringType},
	}
}

func openstackCloudSpecAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"flavor":                       types.StringType,
		"image":                        types.StringType,
		"disk_size":                    types.Int64Type,
		"tags":                         types.MapType{ElemType: types.StringType},
		"use_floating_ip":              types.BoolType,
		"instance_ready_check_period":  types.StringType,
		"instance_ready_check_timeout": types.StringType,
		"server_group_id":              types.StringType,
	}
}

func operatingSystemAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"ubuntu":  types.ListType{ElemType: types.ObjectType{AttrTypes: ubuntuAttrTypes()}},
		"flatcar": types.ListType{ElemType: types.ObjectType{AttrTypes: flatcarAttrTypes()}},
	}
}

func ubuntuAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"dist_upgrade_on_boot": types.BoolType,
	}
}

func flatcarAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"disable_auto_update": types.BoolType,
	}
}

func versionsAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"kubelet": types.StringType,
	}
}

func taintAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"effect": types.StringType,
		"key":    types.StringType,
		"value":  types.StringType,
	}
}

// NodeDeploymentSchema returns the framework schema for metakube_node_deployment
func NodeDeploymentSchema(ctx context.Context) schema.Schema {
	blocks := nodeDeploymentBlocks()
	blocks["timeouts"] = timeouts.Block(ctx, timeouts.Opts{
		Create: true,
		Update: true,
		Delete: true,
	})

	return schema.Schema{
		Description: "Node deployment resource for MetaKube clusters",
		Attributes:  nodeDeploymentAttributes(),
		Blocks:      blocks,
	}
}

func nodeDeploymentAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:    true,
			Description: "The ID of the node deployment",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"project_id": schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Project the cluster belongs to",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"cluster_id": schema.StringAttribute{
			Required:    true,
			Description: "Cluster that node deployment belongs to",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
			Validators: []validator.String{
				stringvalidator.LengthAtLeast(1),
			},
		},
		"name": schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Node deployment name",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"creation_timestamp": schema.StringAttribute{
			Computed:    true,
			Description: "Creation timestamp",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"deletion_timestamp": schema.StringAttribute{
			Computed:    true,
			Description: "Deletion timestamp",
		},
	}
}

func nodeDeploymentBlocks() map[string]schema.Block {
	return map[string]schema.Block{
		"spec": schema.ListNestedBlock{
			Description: "Node deployment specification",
			Validators: []validator.List{
				listvalidator.SizeAtMost(1),
				listvalidator.SizeAtLeast(1),
			},
			NestedObject: schema.NestedBlockObject{
				Attributes: nodeDeploymentSpecAttributes(),
				Blocks:     nodeDeploymentSpecBlocks(),
			},
		},
	}
}

func nodeDeploymentSpecAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"replicas": schema.Int64Attribute{
			Optional:    true,
			Computed:    true,
			Description: "Number of replicas",
		},
		"min_replicas": schema.Int64Attribute{
			Optional:    true,
			Computed:    true,
			Description: "Minimum number of replicas to downscale",
			Validators: []validator.Int64{
				int64validator.AtLeast(0),
			},
		},
		"max_replicas": schema.Int64Attribute{
			Optional:    true,
			Description: "Maximum number of replicas to scale up",
			Validators: []validator.Int64{
				int64validator.AtLeast(1),
			},
		},
	}
}

func nodeDeploymentSpecBlocks() map[string]schema.Block {
	return map[string]schema.Block{
		"template": schema.ListNestedBlock{
			Description: "Template specification",
			Validators: []validator.List{
				listvalidator.SizeAtMost(1),
				listvalidator.SizeAtLeast(1),
			},
			NestedObject: schema.NestedBlockObject{
				Attributes: nodeSpecAttributes(),
				Blocks:     nodeSpecBlocks(),
			},
		},
	}
}

func nodeSpecAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"labels": schema.MapAttribute{
			Optional:    true,
			ElementType: types.StringType,
			Description: "Map of string keys and values that can be used to organize and categorize (scope and select) objects. It will be applied to Nodes allowing users run their apps on specific Node using labelSelector. Note: The server may add additional system labels (system/cluster, system/project) which are available in the `all_labels` attribute.",
		},
		"all_labels": schema.MapAttribute{
			Computed:    true,
			ElementType: types.StringType,
			Description: "All labels on the node deployment, including user-specified labels and system-managed labels (system/cluster, system/project, etc.) added by the server.",
		},
		"node_annotations": schema.MapAttribute{
			Optional:    true,
			ElementType: types.StringType,
			Description: "Map of annotations to set on nodes.",
		},
		"machine_annotations": schema.MapAttribute{
			Optional:    true,
			ElementType: types.StringType,
			Description: "Map of annotations to set on machine objects.",
		},
	}
}

func nodeSpecBlocks() map[string]schema.Block {
	return map[string]schema.Block{
		"cloud": schema.ListNestedBlock{
			Description: "Cloud specification",
			Validators: []validator.List{
				listvalidator.SizeAtMost(1),
				listvalidator.SizeAtLeast(1),
			},
			NestedObject: schema.NestedBlockObject{
				Blocks: cloudSpecBlocks(),
			},
		},
		"operating_system": schema.ListNestedBlock{
			Description: "Operating system",
			Validators: []validator.List{
				listvalidator.SizeAtMost(1),
				listvalidator.SizeAtLeast(1),
			},
			NestedObject: schema.NestedBlockObject{
				Blocks: operatingSystemBlocks(),
			},
		},
		"versions": schema.ListNestedBlock{
			Description: "Cloud components versions",
			Validators: []validator.List{
				listvalidator.SizeAtMost(1),
			},
			NestedObject: schema.NestedBlockObject{
				Attributes: versionsAttributes(),
			},
		},
		"taints": schema.ListNestedBlock{
			Description: "List of taints to set on new nodes",
			NestedObject: schema.NestedBlockObject{
				Attributes: taintAttributes(),
			},
		},
	}
}

func cloudSpecBlocks() map[string]schema.Block {
	return map[string]schema.Block{
		"aws": schema.ListNestedBlock{
			Description: "AWS node deployment specification",
			Validators: []validator.List{
				listvalidator.SizeAtMost(1),
			},
			NestedObject: schema.NestedBlockObject{
				Attributes: awsCloudSpecAttributes(),
			},
		},
		"openstack": schema.ListNestedBlock{
			Description: "OpenStack node deployment specification",
			Validators: []validator.List{
				listvalidator.SizeAtMost(1),
			},
			NestedObject: schema.NestedBlockObject{
				Attributes: openstackCloudSpecAttributes(),
			},
		},
	}
}

func awsCloudSpecAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"instance_type": schema.StringAttribute{
			Required:    true,
			Description: "EC2 instance type",
			Validators: []validator.String{
				stringvalidator.LengthAtLeast(1),
			},
		},
		"disk_size": schema.Int64Attribute{
			Required:    true,
			Description: "Size of the volume in GBs. Only one volume will be created",
			Validators: []validator.Int64{
				int64validator.AtLeast(1),
			},
		},
		"volume_type": schema.StringAttribute{
			Required:    true,
			Description: "EBS volume type",
			Validators: []validator.String{
				stringvalidator.LengthAtLeast(1),
			},
		},
		"availability_zone": schema.StringAttribute{
			Required:    true,
			Description: "Availability zone in which to place the node. It is coupled with the subnet to which the node will belong",
			Validators: []validator.String{
				stringvalidator.LengthAtLeast(1),
			},
		},
		"subnet_id": schema.StringAttribute{
			Required:    true,
			Description: "The VPC subnet to which the node shall be connected",
			Validators: []validator.String{
				stringvalidator.LengthAtLeast(1),
			},
		},
		"assign_public_ip": schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Default:     booldefault.StaticBool(true),
			Description: "Flag which controls a property of the AWS instance. When set the AWS instance will get a public IP address assigned during launch overriding a possible setting in the used AWS subnet.",
		},
		"ami": schema.StringAttribute{
			Optional:    true,
			Description: "Amazon Machine Image to use. Will be defaulted to an AMI of your selected operating system and region",
		},
		"tags": schema.MapAttribute{
			Optional:    true,
			Computed:    true,
			ElementType: types.StringType,
			Description: "Additional instance tags",
		},
	}
}

func openstackCloudSpecAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"flavor": schema.StringAttribute{
			Required:    true,
			Description: "Instance type",
			Validators: []validator.String{
				stringvalidator.LengthAtLeast(1),
			},
		},
		"image": schema.StringAttribute{
			Required:    true,
			Description: "Image to use",
			Validators: []validator.String{
				stringvalidator.LengthAtLeast(1),
			},
		},
		"disk_size": schema.Int64Attribute{
			Optional:    true,
			Description: "If set, the rootDisk will be a cinder volume of that size in GiB. If unset, the rootDisk will be ephemeral nova root storage and its size will be derived from the flavor",
			Validators: []validator.Int64{
				int64validator.AtLeast(1),
			},
		},
		"tags": schema.MapAttribute{
			Optional:    true,
			Computed:    true,
			ElementType: types.StringType,
			Description: "Additional instance tags",
		},
		"use_floating_ip": schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Default:     booldefault.StaticBool(true),
			Description: "Indicate use of floating ip in case of floating_ip_pool presence",
		},
		"instance_ready_check_period": schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Default:     stringdefault.StaticString("5s"),
			Description: "Specifies how often should the controller check if instance is ready before timing out",
			Validators: []validator.String{
				durationValidator{},
			},
		},
		"instance_ready_check_timeout": schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Default:     stringdefault.StaticString("120s"),
			Description: "Specifies how long should the controller check if instance is ready before timing out",
			Validators: []validator.String{
				durationValidator{},
			},
		},
		"server_group_id": schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Specifies the ID of the server group for nodes in the nodes deployment. Defaults to the cluster setting",
			PlanModifiers: []planmodifier.String{
				serverGroupIDPlanModifier{},
			},
		},
	}
}

func operatingSystemBlocks() map[string]schema.Block {
	return map[string]schema.Block{
		"ubuntu": schema.ListNestedBlock{
			Description: "Ubuntu operating system",
			Validators: []validator.List{
				listvalidator.SizeAtMost(1),
			},
			NestedObject: schema.NestedBlockObject{
				Attributes: ubuntuAttributes(),
			},
		},
		"flatcar": schema.ListNestedBlock{
			Description: "Flatcar operating system",
			Validators: []validator.List{
				listvalidator.SizeAtMost(1),
			},
			NestedObject: schema.NestedBlockObject{
				Attributes: flatcarAttributes(),
			},
		},
	}
}

func ubuntuAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"dist_upgrade_on_boot": schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Default:     booldefault.StaticBool(false),
			Description: "Upgrade operating system on boot",
		},
	}
}

func flatcarAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"disable_auto_update": schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Default:     booldefault.StaticBool(false),
			Description: "Disable flatcar auto update feature",
		},
	}
}

func versionsAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"kubelet": schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Kubelet version",
		},
	}
}

func taintAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"effect": schema.StringAttribute{
			Required:    true,
			Description: "Taint effect",
			Validators: []validator.String{
				stringvalidator.OneOf("NoSchedule", "PreferNoSchedule", "NoExecute"),
			},
		},
		"key": schema.StringAttribute{
			Required:    true,
			Description: "Taint key",
			Validators: []validator.String{
				stringvalidator.LengthAtLeast(1),
			},
		},
		"value": schema.StringAttribute{
			Required:    true,
			Description: "Taint value",
			Validators: []validator.String{
				stringvalidator.LengthAtLeast(1),
			},
		},
	}
}

// Custom validators

// durationValidator validates that a string is a valid duration
type durationValidator struct{}

func (v durationValidator) Description(ctx context.Context) string {
	return "Must be a valid duration string (e.g., '5s', '120s')"
}

func (v durationValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v durationValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	value := req.ConfigValue.ValueString()
	if value == "" {
		return
	}

	_, err := time.ParseDuration(value)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Duration",
			fmt.Sprintf("Value %q cannot be parsed as a duration: %s", value, err),
		)
	}
}

// serverGroupIDPlanModifier preserves state value when config is empty
type serverGroupIDPlanModifier struct{}

func (m serverGroupIDPlanModifier) Description(ctx context.Context) string {
	return "Preserves state value when config is empty"
}

func (m serverGroupIDPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m serverGroupIDPlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.ConfigValue.IsUnknown() {
		return
	}

	if (req.ConfigValue.IsNull() || req.ConfigValue.ValueString() == "") &&
		!req.StateValue.IsNull() && req.StateValue.ValueString() != "" {
		resp.PlanValue = req.StateValue
	}
}

// int64RequiresReplacePlanModifier forces replacement when value changes
type int64RequiresReplacePlanModifier struct{}

func (m int64RequiresReplacePlanModifier) Description(ctx context.Context) string {
	return "Requires replacement when value changes"
}

func (m int64RequiresReplacePlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m int64RequiresReplacePlanModifier) PlanModifyInt64(ctx context.Context, req planmodifier.Int64Request, resp *planmodifier.Int64Response) {
	if req.StateValue.IsNull() {
		return
	}

	if !req.PlanValue.Equal(req.StateValue) {
		resp.RequiresReplace = true
	}
}

// isSystemKey checks for labels created by Metakube.
func isSystemKey(key string) bool {
	switch key {
	case "system/cluster", "system/project":
		return true
	default:
		return false
	}
}
