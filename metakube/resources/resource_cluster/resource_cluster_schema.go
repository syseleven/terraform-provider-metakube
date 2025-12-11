package resource_cluster

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	fwpath "github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
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

type boolDiffSuppressPlanModifier struct{}

func (m boolDiffSuppressPlanModifier) Description(ctx context.Context) string {
	return "Suppresses diff when config is null but state has a value"
}

func (m boolDiffSuppressPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m boolDiffSuppressPlanModifier) PlanModifyBool(ctx context.Context, req planmodifier.BoolRequest, resp *planmodifier.BoolResponse) {
	if req.ConfigValue.IsNull() && !req.StateValue.IsNull() {
		resp.PlanValue = req.StateValue
	}
}

func BoolDiffSuppress() planmodifier.Bool {
	return boolDiffSuppressPlanModifier{}
}

func ClusterResourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description: "Cluster resource in MetaKube",
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Create: true,
				Update: true,
				Delete: true,
			}),
			"spec": schema.ListNestedBlock{
				Description: "Cluster specification",
				NestedObject: schema.NestedBlockObject{
					Attributes: metakubeResourceClusterSpecAttributes(),
					Blocks:     metakubeResourceClusterSpecBlocks(),
				},
			},
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
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"pods_cidr": schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Internal IP range for Pods",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
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
			Optional:    true,
			Computed:    true,
			Description: "Contains the spec of the CNI plugin used by the Cluster. Defaults to canal if not specified.",
			Attributes: map[string]schema.Attribute{
				"type": schema.StringAttribute{
					Optional:    true,
					Computed:    true,
					Default:     stringdefault.StaticString("canal"),
					Description: "Define the type of CNI plugin",
					Validators: []validator.String{
						stringvalidator.OneOf("cilium", "canal", "none"),
					},
				},
			},
			Default: objectdefault.StaticValue(types.ObjectValueMust(
				map[string]attr.Type{"type": types.StringType},
				map[string]attr.Value{"type": types.StringValue("canal")},
			)),
		},
	}
}

func metakubeResourceClusterSpecBlocks() map[string]schema.Block {
	return map[string]schema.Block{
		"update_window": schema.ListNestedBlock{
			Description: "Flatcar nodes reboot window",
			Validators: []validator.List{
				listvalidator.SizeAtMost(1),
			},
			NestedObject: schema.NestedBlockObject{
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
		},
		"cloud": schema.ListNestedBlock{
			Description: "Cloud provider specification",
			Validators: []validator.List{
				listvalidator.SizeAtLeast(1),
				listvalidator.SizeAtMost(1),
			},
			NestedObject: schema.NestedBlockObject{
				Blocks: map[string]schema.Block{
					"aws": schema.ListNestedBlock{
						Description: "AWS cluster specification",
						Validators: []validator.List{
							listvalidator.SizeAtMost(1),
							listvalidator.ConflictsWith(
								fwpath.MatchRelative().AtParent().AtName("openstack"),
								fwpath.MatchRelative().AtParent().AtName("azure"),
							),
						},
						NestedObject: metakubeResourceClusterAWSCloudSpecFields(),
					},
					"openstack": schema.ListNestedBlock{
						Description: "OpenStack cluster specification",
						Validators: []validator.List{
							listvalidator.SizeAtMost(1),
							listvalidator.ConflictsWith(
								fwpath.MatchRelative().AtParent().AtName("aws"),
								fwpath.MatchRelative().AtParent().AtName("azure"),
							),
						},
						NestedObject: metakubeResourceClusterOpenstackCloudSpecFields(),
					},
					"azure": schema.ListNestedBlock{
						Description: "Azure cluster specification",
						Validators: []validator.List{
							listvalidator.SizeAtMost(1),
							listvalidator.ConflictsWith(
								fwpath.MatchRelative().AtParent().AtName("aws"),
								fwpath.MatchRelative().AtParent().AtName("openstack"),
							),
						},
						NestedObject: metakubeResourceClusterAzureSpecFields(),
					},
				},
			},
		},
		"syseleven_auth": schema.ListNestedBlock{
			Description: "Configuration of SysEleven Login over OpenID Connect to authenticate against this cluster",
			Validators: []validator.List{
				listvalidator.SizeAtMost(1),
			},
			NestedObject: schema.NestedBlockObject{
				Attributes: map[string]schema.Attribute{
					"realm": schema.StringAttribute{
						Optional:    true,
						Description: "Realm name",
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(1),
						},
					},
					"iam_authentication": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
						Description: "Enable Authentication against Syseleven IAM system",
						PlanModifiers: []planmodifier.Bool{
							BoolDiffSuppress(),
						},
					},
				},
			},
		},
	}
}

func metakubeResourceClusterAzureSpecFields() schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"availability_set": schema.StringAttribute{
				Optional: true,
			},
			"client_id": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"client_secret": schema.StringAttribute{
				Required:  true,
				Sensitive: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"subscription_id": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"tenant_id": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"resource_group": schema.StringAttribute{
				Computed: true,
				Optional: true,
			},
			"route_table": schema.StringAttribute{
				Computed: true,
				Optional: true,
			},
			"security_group": schema.StringAttribute{
				Computed: true,
				Optional: true,
			},
			"subnet": schema.StringAttribute{
				Computed: true,
				Optional: true,
			},
			"vnet": schema.StringAttribute{
				Computed: true,
				Optional: true,
			},
			"openstack_billing_tenant": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				Description: "Openstack tenant/project name for the account",
				PlanModifiers: []planmodifier.String{
					EnvDefaultWithDiffSuppress("OS_PROJECT_NAME"),
				},
			},
		},
	}
}

func metakubeResourceClusterAWSCloudSpecFields() schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"access_key_id": schema.StringAttribute{
				Required:  true,
				Sensitive: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				Description: "Access key identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"secret_access_key": schema.StringAttribute{
				Required:  true,
				Sensitive: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				Description: "Secret access key",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"vpc_id": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				Description: "Virtual private cloud identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"security_group_id": schema.StringAttribute{
				Optional:    true,
				Description: "Security group identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"route_table_id": schema.StringAttribute{
				Optional:    true,
				Description: "Route table identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"instance_profile_name": schema.StringAttribute{
				Optional:    true,
				Description: "Instance profile name",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"role_arn": schema.StringAttribute{
				Optional:    true,
				Description: "The IAM role the control plane will use over assume-role",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"openstack_billing_tenant": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				Description: "Openstack tenant/project name for the account",
				PlanModifiers: []planmodifier.String{
					EnvDefault("OS_PROJECT_NAME"),
				},
			},
		},
	}
}

func metakubeResourceClusterOpenstackCloudSpecFields() schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"floating_ip_pool": schema.StringAttribute{
				Computed:    true,
				Optional:    true,
				Description: "The floating ip pool used by all worker nodes to receive a public ip",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"security_group": schema.StringAttribute{
				Computed:    true,
				Optional:    true,
				Description: "When specified, all worker nodes will be attached to this security group. If not specified, a security group will be created",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"network": schema.StringAttribute{
				Computed:    true,
				Optional:    true,
				Description: "When specified, all worker nodes will be attached to this network. If not specified, a network, subnet & router will be created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"subnet_id": schema.StringAttribute{
				Computed:    true,
				Optional:    true,
				Description: "When specified, all worker nodes will be attached to this subnet of specified network. If not specified, a network, subnet & router will be created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.AlsoRequires(fwpath.MatchRoot("spec").AtListIndex(0).AtName("cloud").AtListIndex(0).AtName("openstack").AtListIndex(0).AtName("network")),
				},
			},
			"subnet_cidr": schema.StringAttribute{
				Computed: true,
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Description: "Change this to configure a different internal IP range for Nodes. Default: 192.168.1.0/24",
			},
			"server_group_id": schema.StringAttribute{
				Computed:    true,
				Optional:    true,
				Description: "Server group to use for all machines within a cluster",
			},
		},
		Blocks: map[string]schema.Block{
			"user_credentials": schema.ListNestedBlock{
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
					listvalidator.ConflictsWith(
						fwpath.MatchRoot("spec").AtListIndex(0).AtName("cloud").AtListIndex(0).AtName("openstack").AtListIndex(0).AtName("application_credentials"),
					),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: metakubeResourceClusterOpenstackCloudSpecUserCredentialsFields(),
				},
			},
			"application_credentials": schema.ListNestedBlock{
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
					listvalidator.ConflictsWith(
						fwpath.MatchRoot("spec").AtListIndex(0).AtName("cloud").AtListIndex(0).AtName("openstack").AtListIndex(0).AtName("user_credentials"),
					),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: metakubeResourceClusterOpenstackCloudSpecApplicationCredentialsFields(),
				},
			},
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

// ClusterModel represents the Terraform resource model for a cluster.
type ClusterModel struct {
	ID                  types.String   `tfsdk:"id"`
	ProjectID           types.String   `tfsdk:"project_id"`
	DCName              types.String   `tfsdk:"dc_name"`
	Name                types.String   `tfsdk:"name"`
	Labels              types.Map      `tfsdk:"labels"`
	SSHKeys             types.Set      `tfsdk:"sshkeys"`
	Spec                types.List     `tfsdk:"spec"` // []ClusterSpecModel
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
	UpdateWindow      types.List   `tfsdk:"update_window"`  // []UpdateWindowModel
	CNIPlugin         types.Object `tfsdk:"cni_plugin"`     // CNIPluginModel
	Cloud             types.List   `tfsdk:"cloud"`          // []ClusterCloudSpecModel
	SyselevenAuth     types.List   `tfsdk:"syseleven_auth"` // []SyselevenAuthModel
}

// UpdateWindowModel represents the update_window block.
type UpdateWindowModel struct {
	Start  types.String `tfsdk:"start"`
	Length types.String `tfsdk:"length"`
}

// CNIPluginModel represents the cni_plugin block.
type CNIPluginModel struct {
	Type types.String `tfsdk:"type"`
}

// SyselevenAuthModel represents the syseleven_auth block.
type SyselevenAuthModel struct {
	Realm             types.String `tfsdk:"realm"`
	IAMAuthentication types.Bool   `tfsdk:"iam_authentication"`
}

// ClusterCloudSpecModel represents the cloud block.
type ClusterCloudSpecModel struct {
	AWS       types.List `tfsdk:"aws"`       // []AWSCloudSpecModel
	Openstack types.List `tfsdk:"openstack"` // []OpenstackCloudSpecModel
	Azure     types.List `tfsdk:"azure"`     // []AzureCloudSpecModel
}

// AWSCloudSpecModel represents the AWS cloud specification.
type AWSCloudSpecModel struct {
	AccessKeyID            types.String `tfsdk:"access_key_id"`
	SecretAccessKey        types.String `tfsdk:"secret_access_key"`
	VPCID                  types.String `tfsdk:"vpc_id"`
	SecurityGroupID        types.String `tfsdk:"security_group_id"`
	RouteTableID           types.String `tfsdk:"route_table_id"`
	InstanceProfileName    types.String `tfsdk:"instance_profile_name"`
	RoleARN                types.String `tfsdk:"role_arn"`
	OpenstackBillingTenant types.String `tfsdk:"openstack_billing_tenant"`
}

// OpenstackCloudSpecModel represents the OpenStack cloud specification.
type OpenstackCloudSpecModel struct {
	UserCredentials        types.List   `tfsdk:"user_credentials"`        // []OpenstackUserCredentialsModel
	ApplicationCredentials types.List   `tfsdk:"application_credentials"` // []OpenstackApplicationCredentialsModel
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

// AzureCloudSpecModel represents the Azure cloud specification.
type AzureCloudSpecModel struct {
	AvailabilitySet        types.String `tfsdk:"availability_set"`
	ClientID               types.String `tfsdk:"client_id"`
	ClientSecret           types.String `tfsdk:"client_secret"`
	SubscriptionID         types.String `tfsdk:"subscription_id"`
	TenantID               types.String `tfsdk:"tenant_id"`
	ResourceGroup          types.String `tfsdk:"resource_group"`
	RouteTable             types.String `tfsdk:"route_table"`
	SecurityGroup          types.String `tfsdk:"security_group"`
	Subnet                 types.String `tfsdk:"subnet"`
	VNet                   types.String `tfsdk:"vnet"`
	OpenstackBillingTenant types.String `tfsdk:"openstack_billing_tenant"`
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
		"update_window":       types.ListType{ElemType: types.ObjectType{AttrTypes: updateWindowAttrTypes()}},
		"cni_plugin":          types.ObjectType{AttrTypes: cniPluginAttrTypes()},
		"cloud":               types.ListType{ElemType: types.ObjectType{AttrTypes: clusterCloudSpecAttrTypes()}},
		"syseleven_auth":      types.ListType{ElemType: types.ObjectType{AttrTypes: syselevenAuthAttrTypes()}},
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
		"aws":       types.ListType{ElemType: types.ObjectType{AttrTypes: awsCloudSpecAttrTypes()}},
		"openstack": types.ListType{ElemType: types.ObjectType{AttrTypes: openstackCloudSpecAttrTypes()}},
		"azure":     types.ListType{ElemType: types.ObjectType{AttrTypes: azureCloudSpecAttrTypes()}},
	}
}

func awsCloudSpecAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"access_key_id":            types.StringType,
		"secret_access_key":        types.StringType,
		"vpc_id":                   types.StringType,
		"security_group_id":        types.StringType,
		"route_table_id":           types.StringType,
		"instance_profile_name":    types.StringType,
		"role_arn":                 types.StringType,
		"openstack_billing_tenant": types.StringType,
	}
}

func openstackCloudSpecAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"user_credentials":        types.ListType{ElemType: types.ObjectType{AttrTypes: openstackUserCredentialsAttrTypes()}},
		"application_credentials": types.ListType{ElemType: types.ObjectType{AttrTypes: openstackApplicationCredentialsAttrTypes()}},
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

func azureCloudSpecAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"availability_set":         types.StringType,
		"client_id":                types.StringType,
		"client_secret":            types.StringType,
		"subscription_id":          types.StringType,
		"tenant_id":                types.StringType,
		"resource_group":           types.StringType,
		"route_table":              types.StringType,
		"security_group":           types.StringType,
		"subnet":                   types.StringType,
		"vnet":                     types.StringType,
		"openstack_billing_tenant": types.StringType,
	}
}
