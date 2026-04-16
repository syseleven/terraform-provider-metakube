package resource_cluster

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/boolvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	fwpath "github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type durationValidator struct{}

func (v durationValidator) Description(ctx context.Context) string {
	return "Must be a valid duration"
}

func (v durationValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	value := req.ConfigValue.ValueString()
	_, err := time.ParseDuration(value)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Time Duration",
			fmt.Sprintf("Value %q cannot be parsed as a duration: %s", value, err),
		)
	}
}

func (v durationValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func DurationValidator() validator.String {
	return durationValidator{}
}

// envDefaultPlanModifier is a plan modifier that sets a default value from an environment variable.
// When diffSuppress is enabled, it also uses the prior state value when the config value is null/empty.
type envDefaultPlanModifier struct {
	envVar       string
	diffSuppress bool
}

func (m envDefaultPlanModifier) Description(ctx context.Context) string {
	if m.diffSuppress {
		return fmt.Sprintf("Uses environment variable %s as default, preserves prior state when config is empty", m.envVar)
	}
	return fmt.Sprintf("Uses environment variable %s as default", m.envVar)
}

func (m envDefaultPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m envDefaultPlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if !req.ConfigValue.IsNull() && req.ConfigValue.ValueString() != "" {
		return
	}

	if m.diffSuppress && !req.StateValue.IsNull() && req.StateValue.ValueString() != "" {
		resp.PlanValue = req.StateValue
		return
	}

	if envVal := os.Getenv(m.envVar); envVal != "" {
		resp.PlanValue = types.StringValue(envVal)
	}
}

func EnvDefault(envVar string) planmodifier.String {
	return envDefaultPlanModifier{envVar: envVar, diffSuppress: false}
}

func EnvDefaultWithDiffSuppress(envVar string) planmodifier.String {
	return envDefaultPlanModifier{envVar: envVar, diffSuppress: true}
}

func ClusterResourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description: "Cluster resource in MetaKube",
		Version:     2,
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Create: true,
				Update: true,
				Delete: true,
			}),
		},
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Cluster identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Required:    true,
				Description: "Reference project identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"dc_name": schema.StringAttribute{
				Required:    true,
				Description: "Data center name",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Cluster name",
			},
			"spec": schema.SingleNestedAttribute{
				Required:    true,
				Description: "Cluster specification",
				Attributes:  metakubeResourceClusterSpecAttributes(),
			},
			"labels": schema.MapAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Labels added to cluster",
				Default:     mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
			},
			"sshkeys": schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "SSH keys attached to nodes",
				Default:     setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
			},
			"creation_timestamp": schema.StringAttribute{
				Computed:    true,
				Description: "Creation timestamp",
			},
			"deletion_timestamp": schema.StringAttribute{
				Computed:    true,
				Description: "Deletion timestamp",
			},
			"kube_config": schema.StringAttribute{
				Sensitive:   true,
				Computed:    true,
				Description: "Kubeconfig for the cluster",
			},
			"oidc_kube_config": schema.StringAttribute{
				Sensitive:   true,
				Computed:    true,
				Description: "OIDC Kubeconfig for the cluster",
			},
			"kube_login_kube_config": schema.StringAttribute{
				Computed:    true,
				Description: "Kubelogin Kubeconfig for the cluster",
			},
		},
	}
}

func metakubeResourceClusterSpecAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"version": schema.StringAttribute{
			Required:    true,
			Description: "Cloud orchestrator version, either Kubernetes or OpenShift",
			Validators: []validator.String{
				stringvalidator.LengthAtLeast(1),
			},
		},
		"enable_ssh_agent": schema.BoolAttribute{
			Computed:    true,
			Default:     booldefault.StaticBool(true),
			Optional:    true,
			Description: "SSH Agent as a daemon running on each node that can manage ssh keys. Disable it if you want to manage keys manually",
		},
		"audit_logging": schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Default:     booldefault.StaticBool(false),
			Description: "Whether to enable audit logging or not",
		},
		"pod_security_policy": schema.BoolAttribute{
			Optional:           true,
			Computed:           true,
			Default:            booldefault.StaticBool(false),
			DeprecationMessage: "PodSecurityPolicy deprecated by Kubernetes since version 1.21 and will be removed in version 1.25",
			Description:        "Pod security policies allow detailed authorization of pod creation and updates",
		},
		"pod_node_selector": schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Default:     booldefault.StaticBool(false),
			Description: "Configure PodNodeSelector admission plugin at the apiserver",
		},
		"services_cidr": schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Internal IP range for ClusterIP Services",
		},
		"pods_cidr": schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Internal IP range for Pods",
		},
		"ip_family": schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Default:     stringdefault.StaticString("IPv4"),
			Description: "Represents IP address family to use for the Cluster",
			Validators: []validator.String{
				stringvalidator.OneOf("IPv4", "IPv4+IPv6"),
			},
		},
		"cni_plugin": schema.SingleNestedAttribute{
			Description: "Contains the spec of the CNI plugin used by the Cluster. Defaults to canal if not specified.",
			Optional:    true,
			Computed:    true,
			Attributes: map[string]schema.Attribute{
				"type": schema.StringAttribute{
					Optional:    true,
					Computed:    true,
					Description: "Define the type of CNI plugin. Defaults to canal if not specified.",
					Validators: []validator.String{
						stringvalidator.OneOf("cilium", "canal", "none"),
					},
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
				"cilium": schema.SingleNestedAttribute{
					Optional:    true,
					Computed:    true,
					Description: "Cilium clustermesh",
					Attributes:  metakubeResourceClusterCNICiliumAttributes(),
					PlanModifiers: []planmodifier.Object{
						objectplanmodifier.UseStateForUnknown(),
					},
				},
			},
		},
		"update_window": schema.SingleNestedAttribute{
			Optional:    true,
			Description: "Flatcar nodes reboot window",
			Attributes: map[string]schema.Attribute{
				"start": schema.StringAttribute{
					Required:    true,
					Description: "Node reboot window start time",
					Validators: []validator.String{
						stringvalidator.RegexMatches(regexp.MustCompile("(Mon |Tue |Wed |Thu |Fri |Sat )*([0-1][0-9]|2[0-4]):[0-5][0-9]"), "Example: 'Thu 02:00' or '02:00'"),
					},
				},
				"length": schema.StringAttribute{
					Required:    true,
					Description: "Node reboot window duration",
					Validators: []validator.String{
						DurationValidator(),
					},
				},
			},
		},
		"cloud": schema.SingleNestedAttribute{
			Required:    true,
			Description: "Cloud provider specification",
			Attributes: map[string]schema.Attribute{
				"openstack": schema.SingleNestedAttribute{
					Required:    true,
					Description: "OpenStack cluster specification",
					Attributes:  metakubeResourceClusterOpenstackCloudSpecFields(),
				},
			},
		},
		"syseleven_auth": schema.SingleNestedAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Configuration of SysEleven Login over OpenID Connect to authenticate against this cluster",
			Attributes: map[string]schema.Attribute{
				"realm": schema.StringAttribute{
					Optional:    true,
					Default:     stringdefault.StaticString(""),
					Description: "Realm name",
				},
				"iam_authentication": schema.BoolAttribute{
					Optional:    true,
					Computed:    true,
					Default:     booldefault.StaticBool(false),
					Description: "Enable Authentication against Syseleven IAM system",
					PlanModifiers: []planmodifier.Bool{
						boolplanmodifier.UseStateForUnknown(),
					},
				},
			},
			PlanModifiers: []planmodifier.Object{
				objectplanmodifier.UseStateForUnknown(),
			},
		},
	}
}

func metakubeResourceClusterOpenstackCloudSpecFields() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"floating_ip_pool": schema.StringAttribute{
			Computed:    true,
			Optional:    true,
			Description: "The floating ip pool used by all worker nodes to receive a public ip",
		},
		"security_group": schema.StringAttribute{
			Computed:    true,
			Optional:    true,
			Description: "When specified, all worker nodes will be attached to this security group. If not specified, a security group will be created",
		},
		"network": schema.StringAttribute{
			Computed:    true,
			Optional:    true,
			Description: "When specified, all worker nodes will be attached to this network. If not specified, a network, subnet & router will be created.",
		},
		"subnet_id": schema.StringAttribute{
			Computed:    true,
			Optional:    true,
			Description: "When specified, all worker nodes will be attached to this subnet of specified network. If not specified, a network, subnet & router will be created.",
			Validators: []validator.String{
				stringvalidator.AlsoRequires(fwpath.MatchRoot("spec").AtName("cloud").AtName("openstack").AtName("network")),
			},
		},
		"subnet_cidr": schema.StringAttribute{
			Computed:    true,
			Optional:    true,
			Description: "Change this to configure a different internal IP range for Nodes. Default: 192.168.1.0/24",
		},
		"server_group_id": schema.StringAttribute{
			Computed:    true,
			Optional:    true,
			Description: "Server group to use for all machines within a cluster",
		},
		"user_credentials": schema.SingleNestedAttribute{
			Optional: true,
			Validators: []validator.Object{
				objectvalidator.ConflictsWith(
					fwpath.MatchRelative().AtParent().AtName("application_credentials"),
				),
			},
			Attributes: metakubeResourceClusterOpenstackCloudSpecUserCredentialsFields(),
		},
		"application_credentials": schema.SingleNestedAttribute{
			Optional: true,
			Validators: []validator.Object{
				objectvalidator.ConflictsWith(
					fwpath.MatchRelative().AtParent().AtName("user_credentials"),
				),
			},
			Attributes: metakubeResourceClusterOpenstackCloudSpecApplicationCredentialsFields(),
		},
	}
}

func metakubeResourceClusterOpenstackCloudSpecUserCredentialsFields() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"project_id": schema.StringAttribute{
			Optional:    true,
			Description: "The id of openstack project",
			PlanModifiers: []planmodifier.String{
				EnvDefaultWithDiffSuppress("OS_PROJECT_ID"),
			},
		},
		"project_name": schema.StringAttribute{
			Optional:           true,
			DeprecationMessage: "use project_id or switch to application_credentials",
			Description:        "The name of openstack project",
			PlanModifiers: []planmodifier.String{
				EnvDefaultWithDiffSuppress("OS_PROJECT_NAME"),
			},
		},
		"username": schema.StringAttribute{
			Optional:    true,
			Sensitive:   true,
			Description: "The openstack account's username",
			PlanModifiers: []planmodifier.String{
				EnvDefaultWithDiffSuppress("OS_USERNAME"),
			},
		},
		"password": schema.StringAttribute{
			Optional:    true,
			Sensitive:   true,
			Description: "The openstack account's password",
			PlanModifiers: []planmodifier.String{
				EnvDefaultWithDiffSuppress("OS_PASSWORD"),
			},
		},
	}
}

func metakubeResourceClusterOpenstackCloudSpecApplicationCredentialsFields() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Optional:    true,
			Description: "Openstack application credentials ID",
			PlanModifiers: []planmodifier.String{
				EnvDefaultWithDiffSuppress("OS_APPLICATION_CREDENTIAL_ID"),
			},
		},
		"secret": schema.StringAttribute{
			Optional:    true,
			Sensitive:   true,
			Description: "Openstack application credentials secret",
			PlanModifiers: []planmodifier.String{
				EnvDefaultWithDiffSuppress("OS_APPLICATION_CREDENTIAL_SECRET"),
			},
		},
	}
}

func metakubeResourceClusterCNICiliumAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"enable_hubble": schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Enable Hubble relay",
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseNonNullStateForUnknown(),
			},
		},
		"enable_l7_proxy": schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Enable L7 Proxy",
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseNonNullStateForUnknown(),
			},
		},
		"clustermesh": schema.SingleNestedAttribute{
			Optional: true,
			Attributes: map[string]schema.Attribute{
				"enable": schema.BoolAttribute{
					Optional:    true,
					Description: "Enale clustermesh",
					Validators: []validator.Bool{
						boolvalidator.AlsoRequires(path.Expressions{
							path.MatchRelative().AtParent().AtName("cluster_id"),
							path.MatchRelative().AtParent().AtName("ipv4_native_routing_cidr"),
						}...),
					},
				},
				"cluster_id": schema.Int32Attribute{
					Optional:    true,
					Description: "Set cilium cluster ID",
				},
				"ipv4_native_routing_cidr": schema.StringAttribute{
					Optional:    true,
					Description: "Set ipv4 native routing cidr",
				},
			},
		},
	}
}

// ClusterModel represents the Terraform resource model for a cluster.
type ClusterModel struct {
	ID                  types.String   `tfsdk:"id"`
	ProjectID           types.String   `tfsdk:"project_id"`
	DCName              types.String   `tfsdk:"dc_name"`
	Name                types.String   `tfsdk:"name"`
	Labels              types.Map      `tfsdk:"labels"`
	SSHKeys             types.Set      `tfsdk:"sshkeys"`
	Spec                types.Object   `tfsdk:"spec"` // ClusterSpecModel
	CreationTimestamp   types.String   `tfsdk:"creation_timestamp"`
	DeletionTimestamp   types.String   `tfsdk:"deletion_timestamp"`
	KubeConfig          types.String   `tfsdk:"kube_config"`
	OIDCKubeConfig      types.String   `tfsdk:"oidc_kube_config"`
	KubeLoginKubeConfig types.String   `tfsdk:"kube_login_kube_config"`
	Timeouts            timeouts.Value `tfsdk:"timeouts"`
}

// ClusterSpecModel represents the spec block of a cluster.
type ClusterSpecModel struct {
	Version           types.String `tfsdk:"version"`
	EnableSSHAgent    types.Bool   `tfsdk:"enable_ssh_agent"`
	AuditLogging      types.Bool   `tfsdk:"audit_logging"`
	PodSecurityPolicy types.Bool   `tfsdk:"pod_security_policy"`
	PodNodeSelector   types.Bool   `tfsdk:"pod_node_selector"`
	ServicesCIDR      types.String `tfsdk:"services_cidr"`
	PodsCIDR          types.String `tfsdk:"pods_cidr"`
	IPFamily          types.String `tfsdk:"ip_family"`
	UpdateWindow      types.Object `tfsdk:"update_window"`  // UpdateWindowModel
	CNIPlugin         types.Object `tfsdk:"cni_plugin"`     // CNIPluginModel
	Cloud             types.Object `tfsdk:"cloud"`          // ClusterCloudSpecModel
	SyselevenAuth     types.Object `tfsdk:"syseleven_auth"` // SyselevenAuthModel
}

// UpdateWindowModel represents the update_window block.
type UpdateWindowModel struct {
	Start  types.String `tfsdk:"start"`
	Length types.String `tfsdk:"length"`
}

// CNIPluginModel represents the cni_plugin block.
type CNIPluginModel struct {
	Type   types.String `tfsdk:"type"`
	Cilium types.Object `tfsdk:"cilium"` // CiliumSpecModel
}

// CiliumModel
type CiliumModel struct {
	Clustermesh   types.Object `tfsdk:"clustermesh"` // CiliumClustermeshSpecModel
	EnableHubble  types.Bool   `tfsdk:"enable_hubble"`
	EnableL7Proxy types.Bool   `tfsdk:"enable_l7_proxy"`
}

// CiliumSpecModel
type CiliumClustermeshModel struct {
	Enable                types.Bool   `tfsdk:"enable"`
	ClusterID             types.Int32  `tfsdk:"cluster_id"`
	IPv4NativeRoutingCIDR types.String `tfsdk:"ipv4_native_routing_cidr"`
}

// SyselevenAuthModel represents the syseleven_auth block.
type SyselevenAuthModel struct {
	Realm             types.String `tfsdk:"realm"`
	IAMAuthentication types.Bool   `tfsdk:"iam_authentication"`
}

// ClusterCloudSpecModel represents the cloud block.
type ClusterCloudSpecModel struct {
	Openstack types.Object `tfsdk:"openstack"` // OpenstackCloudSpecModel
}

// OpenstackCloudSpecModel represents the OpenStack cloud specification.
type OpenstackCloudSpecModel struct {
	UserCredentials        types.Object `tfsdk:"user_credentials"`        // OpenstackUserCredentialsModel
	ApplicationCredentials types.Object `tfsdk:"application_credentials"` // OpenstackApplicationCredentialsModel
	FloatingIPPool         types.String `tfsdk:"floating_ip_pool"`
	SecurityGroup          types.String `tfsdk:"security_group"`
	Network                types.String `tfsdk:"network"`
	SubnetID               types.String `tfsdk:"subnet_id"`
	SubnetCIDR             types.String `tfsdk:"subnet_cidr"`
	ServerGroupID          types.String `tfsdk:"server_group_id"`
}

// OpenstackUserCredentialsModel represents OpenStack user credentials.
type OpenstackUserCredentialsModel struct {
	ProjectID   types.String `tfsdk:"project_id"`
	ProjectName types.String `tfsdk:"project_name"`
	Username    types.String `tfsdk:"username"`
	Password    types.String `tfsdk:"password"`
}

// OpenstackApplicationCredentialsModel represents OpenStack application credentials.
type OpenstackApplicationCredentialsModel struct {
	ID     types.String `tfsdk:"id"`
	Secret types.String `tfsdk:"secret"`
}

// Attribute type helper functions

func clusterSpecAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"version":             types.StringType,
		"enable_ssh_agent":    types.BoolType,
		"audit_logging":       types.BoolType,
		"pod_security_policy": types.BoolType,
		"pod_node_selector":   types.BoolType,
		"services_cidr":       types.StringType,
		"pods_cidr":           types.StringType,
		"ip_family":           types.StringType,
		"update_window":       types.ObjectType{AttrTypes: updateWindowAttrTypes()},
		"cni_plugin":          types.ObjectType{AttrTypes: cniPluginAttrTypes()},
		"cloud":               types.ObjectType{AttrTypes: clusterCloudSpecAttrTypes()},
		"syseleven_auth":      types.ObjectType{AttrTypes: syselevenAuthAttrTypes()},
	}
}

func updateWindowAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"start":  types.StringType,
		"length": types.StringType,
	}
}

func cniPluginAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"type": types.StringType,
		"cilium": types.ObjectType{
			AttrTypes: ciliumAttrTypes(),
		},
	}
}

func ciliumAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"clustermesh": types.ObjectType{
			AttrTypes: ciliumClustermeshAttrTypes(),
		},
		"enable_hubble":   types.BoolType,
		"enable_l7_proxy": types.BoolType,
	}
}

func ciliumClustermeshAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"enable":                   types.BoolType,
		"cluster_id":               types.Int32Type,
		"ipv4_native_routing_cidr": types.StringType,
	}
}

func syselevenAuthAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"realm":              types.StringType,
		"iam_authentication": types.BoolType,
	}
}

func clusterCloudSpecAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"openstack": types.ObjectType{AttrTypes: openstackCloudSpecAttrTypes()},
	}
}

func openstackCloudSpecAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"user_credentials":        types.ObjectType{AttrTypes: openstackUserCredentialsAttrTypes()},
		"application_credentials": types.ObjectType{AttrTypes: openstackApplicationCredentialsAttrTypes()},
		"floating_ip_pool":        types.StringType,
		"security_group":          types.StringType,
		"network":                 types.StringType,
		"subnet_id":               types.StringType,
		"subnet_cidr":             types.StringType,
		"server_group_id":         types.StringType,
	}
}

func openstackUserCredentialsAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"project_id":   types.StringType,
		"project_name": types.StringType,
		"username":     types.StringType,
		"password":     types.StringType,
	}
}

func openstackApplicationCredentialsAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":     types.StringType,
		"secret": types.StringType,
	}
}
