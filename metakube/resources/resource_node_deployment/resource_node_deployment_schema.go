package resource_node_deployment

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/syseleven/terraform-provider-metakube/metakube/common"
)

// NodeDeploymentModel represents the Terraform state for a node deployment.
type NodeDeploymentModel struct {
	ID                types.String   `tfsdk:"id"`
	ProjectID         types.String   `tfsdk:"project_id"`
	ClusterID         types.String   `tfsdk:"cluster_id"`
	Name              types.String   `tfsdk:"name"`
	Spec              types.Object   `tfsdk:"spec"`
	CreationTimestamp types.String   `tfsdk:"creation_timestamp"`
	DeletionTimestamp types.String   `tfsdk:"deletion_timestamp"`
	Timeouts          timeouts.Value `tfsdk:"timeouts"`
}

// NodeDeploymentSpecModel represents the node deployment spec object.
type NodeDeploymentSpecModel struct {
	Replicas    types.Int64  `tfsdk:"replicas"`
	MinReplicas types.Int64  `tfsdk:"min_replicas"`
	MaxReplicas types.Int64  `tfsdk:"max_replicas"`
	Template    types.Object `tfsdk:"template"`
}

// NodeSpecModel represents the machine template within a node deployment.
type NodeSpecModel struct {
	Cloud              types.Object `tfsdk:"cloud"`
	OperatingSystem    types.Object `tfsdk:"operating_system"`
	Versions           types.Object `tfsdk:"versions"`
	Labels             types.Map    `tfsdk:"labels"`
	AllLabels          types.Map    `tfsdk:"all_labels"`
	Taints             types.List   `tfsdk:"taints"`
	NodeAnnotations    types.Map    `tfsdk:"node_annotations"`
	MachineAnnotations types.Map    `tfsdk:"machine_annotations"`
}

// CloudSpecModel represents the cloud-provider configuration.
type CloudSpecModel struct {
	OpenStack types.Object `tfsdk:"openstack"`
}

// OpenStackCloudSpecModel represents the OpenStack machine configuration.
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

// OperatingSystemModel represents the selectable operating-system settings.
type OperatingSystemModel struct {
	Ubuntu  types.Object `tfsdk:"ubuntu"`
	Flatcar types.Object `tfsdk:"flatcar"`
}

// UbuntuModel represents Ubuntu-specific machine settings.
type UbuntuModel struct {
	DistUpgradeOnBoot types.Bool `tfsdk:"dist_upgrade_on_boot"`
}

// FlatcarModel represents Flatcar-specific machine settings.
type FlatcarModel struct {
	DisableAutoUpdate types.Bool `tfsdk:"disable_auto_update"`
}

// VersionsModel represents component-version overrides for worker nodes.
type VersionsModel struct {
	Kubelet types.String `tfsdk:"kubelet"`
}

// TaintModel represents one Kubernetes taint applied to worker nodes.
type TaintModel struct {
	Effect types.String `tfsdk:"effect"`
	Key    types.String `tfsdk:"key"`
	Value  types.String `tfsdk:"value"`
}

// Attribute type helpers mirror the nested model shapes above.

func nodeDeploymentSpecAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"replicas":     types.Int64Type,
		"min_replicas": types.Int64Type,
		"max_replicas": types.Int64Type,
		"template":     types.ObjectType{AttrTypes: nodeSpecAttrTypes()},
	}
}

func nodeSpecAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"cloud":               types.ObjectType{AttrTypes: cloudSpecAttrTypes()},
		"operating_system":    types.ObjectType{AttrTypes: operatingSystemAttrTypes()},
		"versions":            types.ObjectType{AttrTypes: versionsAttrTypes()},
		"labels":              types.MapType{ElemType: types.StringType},
		"all_labels":          types.MapType{ElemType: types.StringType},
		"taints":              types.ListType{ElemType: types.ObjectType{AttrTypes: taintAttrTypes()}},
		"node_annotations":    types.MapType{ElemType: types.StringType},
		"machine_annotations": types.MapType{ElemType: types.StringType},
	}
}

func cloudSpecAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"openstack": types.ObjectType{AttrTypes: openstackCloudSpecAttrTypes()},
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
		"ubuntu":  types.ObjectType{AttrTypes: ubuntuAttrTypes()},
		"flatcar": types.ObjectType{AttrTypes: flatcarAttrTypes()},
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

// NodeDeploymentSchema returns the framework schema for
// metakube_node_deployment.
func NodeDeploymentSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description: "Node deployment resource for MetaKube clusters",
		Version:     3,
		Attributes:  nodeDeploymentAttributes(),
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Create: true,
				Update: true,
				Delete: true,
			}),
		},
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
		"spec": schema.SingleNestedAttribute{
			Required:    true,
			Description: "Node deployment specification",
			Attributes:  nodeDeploymentSpecAttributes(),
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
		"template": schema.SingleNestedAttribute{
			Required:    true,
			Description: "Template specification",
			Attributes:  nodeSpecAttributes(),
		},
	}
}

func nodeSpecAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"labels": schema.MapAttribute{
			Optional:    true,
			Computed:    true,
			ElementType: types.StringType,
			Default:     mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
			Description: "Map of string keys and values that can be used to organize and categorize (scope and select) objects. It will be applied to Nodes allowing users run their apps on specific Node using labelSelector. Note: The server may add additional system labels (system/cluster, system/project) which are available in the `all_labels` attribute.",
			Validators: []validator.Map{
				noSystemManagedKeysValidator(),
			},
		},
		"all_labels": schema.MapAttribute{
			Computed:    true,
			ElementType: types.StringType,
			Description: "All labels on the node deployment, including user-specified labels and system-managed labels (system/cluster, system/project, etc.) added by the server.",
		},
		"node_annotations": schema.MapAttribute{
			Optional:    true,
			Computed:    true,
			ElementType: types.StringType,
			Default:     mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
			Description: "Map of annotations to set on nodes.",
		},
		"machine_annotations": schema.MapAttribute{
			Optional:    true,
			Computed:    true,
			ElementType: types.StringType,
			Default:     mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
			Description: "Map of annotations to set on machine objects.",
		},
		"cloud": schema.SingleNestedAttribute{
			Required:    true,
			Description: "Cloud specification",
			Attributes:  cloudSpecAttributes(),
		},
		"operating_system": schema.SingleNestedAttribute{
			Required:    true,
			Description: "Operating system",
			Attributes:  operatingSystemAttributes(),
		},
		"versions": schema.SingleNestedAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Cloud components versions",
			Attributes:  versionsAttributes(),
		},
		"taints": schema.ListNestedAttribute{
			Optional:    true,
			Description: "List of taints to set on new nodes",
			NestedObject: schema.NestedAttributeObject{
				Attributes: taintAttributes(),
			},
		},
	}
}

func cloudSpecAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"openstack": schema.SingleNestedAttribute{
			Optional:    true,
			Description: "OpenStack node deployment specification",
			Attributes:  openstackCloudSpecAttributes(),
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
			Default:     mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
			Description: "Additional instance tags. Keys matching reserved prefix patterns are ignored in this attribute.",
			Validators: []validator.Map{
				noSystemManagedKeysValidator(),
			},
			PlanModifiers: []planmodifier.Map{
				mapplanmodifier.UseStateForUnknown(),
			},
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
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"server_group_id": schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Specifies the ID of the server group for nodes in the nodes deployment. Defaults to the cluster setting",
		},
	}
}

func operatingSystemAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"ubuntu": schema.SingleNestedAttribute{
			Optional:    true,
			Description: "Ubuntu operating system",
			Attributes:  ubuntuAttributes(),
		},
		"flatcar": schema.SingleNestedAttribute{
			Optional:    true,
			Description: "Flatcar operating system",
			Attributes:  flatcarAttributes(),
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

type noSystemManagedKeysMapValidator struct{}

func (v noSystemManagedKeysMapValidator) Description(ctx context.Context) string {
	return "Map keys must not use reserved patterns (metakube, system-, system/, kubernetes.io, labels.%)."
}

func (v noSystemManagedKeysMapValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v noSystemManagedKeysMapValidator) ValidateMap(ctx context.Context, req validator.MapRequest, resp *validator.MapResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	for key := range req.ConfigValue.Elements() {
		if common.MetakubeResourceSystemLabelOrTag(key) {
			resp.Diagnostics.AddError(
				"Reserved key prefix",
				fmt.Sprintf("Key %q matches a reserved pattern and cannot be configured.", key),
			)
		}
	}
}

func noSystemManagedKeysValidator() validator.Map {
	return noSystemManagedKeysMapValidator{}
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
