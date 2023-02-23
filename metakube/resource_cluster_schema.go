package metakube

import (
	"regexp"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func metakubeResourceClusterSpecFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"version": {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.NoZeroValues,
			Description:  "Cloud orchestrator version, either Kubernetes or OpenShift",
		},
		"enable_ssh_agent": {
			Type:        schema.TypeBool,
			Default:     true,
			Optional:    true,
			Description: "SSH Agent as a daemon running on each node that can manage ssh keys. Disable it if you want to manage keys manually",
		},
		"update_window": {
			Type:        schema.TypeList,
			Optional:    true,
			MaxItems:    1,
			Description: "Flatcar nodes reboot window",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"start": {
						Type:         schema.TypeString,
						Required:     true,
						Description:  "Node reboot window start time",
						ValidateFunc: validation.StringMatch(regexp.MustCompile("(Mon |Tue |Wed |Thu |Fri |Sat )*([0-1][0-9]|2[0-4]):[0-5][0-9]"), "Example: 'Thu 02:00' or '02:00'"),
					},
					"length": {
						Type:        schema.TypeString,
						Required:    true,
						Description: "Node reboot window duration",
						ValidateFunc: func(i interface{}, _ string) ([]string, []error) {
							s := i.(string)
							_, err := time.ParseDuration(s)
							if err != nil {
								return nil, []error{err}
							}
							return nil, nil
						},
					},
				},
			},
		},
		"cloud": {
			Type:        schema.TypeList,
			Required:    true,
			ForceNew:    true,
			MinItems:    1,
			MaxItems:    1,
			Description: "Cloud provider specification",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"aws": {
						Type:        schema.TypeList,
						Optional:    true,
						MaxItems:    1,
						Description: "AWS cluster specification",
						Elem: &schema.Resource{
							Schema: metakubeResourceCluserAWSCloudSpecFields(),
						},
						ConflictsWith: []string{"spec.0.cloud.0.openstack", "spec.0.cloud.0.azure"},
					},
					"openstack": {
						Type:        schema.TypeList,
						Optional:    true,
						MaxItems:    1,
						Description: "OpenStack cluster specification",
						Elem: &schema.Resource{
							Schema: metakubeResourceClusterOpenstackCloudSpecFields(),
						},
						ConflictsWith: []string{"spec.0.cloud.0.aws", "spec.0.cloud.0.azure"},
					},
					"azure": {
						Type:        schema.TypeList,
						Optional:    true,
						MaxItems:    1,
						Description: "Azire cluster specification",
						Elem: &schema.Resource{
							Schema: metakubeResourceClusterAzureSpecFields(),
						},
						ConflictsWith: []string{"spec.0.cloud.0.aws", "spec.0.cloud.0.openstack"},
					},
				},
			},
		},
		"syseleven_auth": {
			Type:        schema.TypeList,
			Optional:    true,
			MaxItems:    1,
			Description: "Configuration of SysEleven Login over OpenID Connect to authenticate against this cluster",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"realm": {
						Type:         schema.TypeString,
						Required:     true,
						ValidateFunc: validation.NoZeroValues,
						Description:  "Realm name",
					},
				},
			},
		},
		"audit_logging": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: "Whether to enable audit logging or not",
		},
		"pod_security_policy": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: "Pod security policies allow detailed authorization of pod creation and updates",
		},
		"pod_node_selector": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: "Configure PodNodeSelector admission plugin at the apiserver",
		},
		"services_cidr": {
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    true,
			Computed:    true,
			Description: "Internal IP range for ClusterIP Services",
		},
		"pods_cidr": {
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    true,
			Computed:    true,
			Description: "Internal IP range for Pods",
		},
		"cni_plugin": {
			Type:        schema.TypeList,
			Optional:    true,
			Description: "Contains the spec of the CNI plugin used by the Cluster",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"type": {
						Type:         schema.TypeString,
						Required:     true,
						ValidateFunc: validation.StringInSlice([]string{"canal", "none"}, false),
						Description:  "Define the type of CNI plugin",
					},
				},
			},
		},
	}
}

func metakubeResourceClusterAzureSpecFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"availability_set": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"client_id": {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.NoZeroValues,
		},
		"client_secret": {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.NoZeroValues,
			Sensitive:    true,
		},
		"subscription_id": {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.NoZeroValues,
		},
		"tenant_id": {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.NoZeroValues,
		},
		"resource_group": {
			Type:     schema.TypeString,
			Computed: true,
			Optional: true,
		},
		"route_table": {
			Type:     schema.TypeString,
			Computed: true,
			Optional: true,
		},
		"security_group": {
			Type:     schema.TypeString,
			Computed: true,
			Optional: true,
		},
		"subnet": {
			Type:     schema.TypeString,
			Computed: true,
			Optional: true,
		},
		"vnet": {
			Type:     schema.TypeString,
			Computed: true,
			Optional: true,
		},
		"openstack_billing_tenant": {
			Type:         schema.TypeString,
			Required:     true,
			DefaultFunc:  schema.EnvDefaultFunc("OS_PROJECT_NAME", nil),
			ValidateFunc: validation.NoZeroValues,
			Description:  "Openstack tenant/project name for the account",
		},
	}
}

func metakubeResourceCluserAWSCloudSpecFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"access_key_id": {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.NoZeroValues,
			Sensitive:    true,
			Description:  "Access key identifier",
		},
		"secret_access_key": {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.NoZeroValues,
			Sensitive:    true,
			Description:  "Secret access key",
		},
		"vpc_id": {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.NoZeroValues,
			Description:  "Virtual private cloud identifier",
		},
		"security_group_id": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Security group identifier",
		},
		"route_table_id": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Route table identifier",
		},
		"instance_profile_name": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Instance profile name",
		},
		"role_arn": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The IAM role the control plane will use over assume-role",
		},
		"openstack_billing_tenant": {
			Type:         schema.TypeString,
			Required:     true,
			DefaultFunc:  schema.EnvDefaultFunc("OS_PROJECT_NAME", nil),
			ValidateFunc: validation.NoZeroValues,
			Description:  "Openstack tenant/project name for the account",
		},
	}
}

func metakubeResourceClusterOpenstackCloudSpecFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"user_credentials": {
			Type:         schema.TypeList,
			MaxItems:     1,
			Optional:     true,
			ExactlyOneOf: []string{"spec.0.cloud.0.openstack.0.user_credentials", "spec.0.cloud.0.openstack.0.application_credentials"},
			Elem: &schema.Resource{
				Schema: metakubeResourceClusterOpenstackCloudSpecUserCredentialsFields(),
			},
		},
		"application_credentials": {
			Type:         schema.TypeList,
			MaxItems:     1,
			Optional:     true,
			ExactlyOneOf: []string{"spec.0.cloud.0.openstack.0.user_credentials", "spec.0.cloud.0.openstack.0.application_credentials"},
			Elem: &schema.Resource{
				Schema: metakubeResourceClusterOpenstackCloudSpecApplicationCredentialsFields(),
			},
		},
		"floating_ip_pool": {
			Type:        schema.TypeString,
			Computed:    true,
			Optional:    true,
			ForceNew:    true,
			Description: "The floating ip pool used by all worker nodes to receive a public ip",
		},
		"security_group": {
			Type:        schema.TypeString,
			Computed:    true,
			Optional:    true,
			ForceNew:    true,
			Description: "When specified, all worker nodes will be attached to this security group. If not specified, a security group will be created",
		},
		"network": {
			Type:        schema.TypeString,
			Computed:    true,
			Optional:    true,
			ForceNew:    true,
			Description: "When specified, all worker nodes will be attached to this network. If not specified, a network, subnet & router will be created.",
		},
		"subnet_id": {
			Type:         schema.TypeString,
			Computed:     true,
			Optional:     true,
			ForceNew:     true,
			RequiredWith: []string{"spec.0.cloud.0.openstack.0.network"},
			Description:  "When specified, all worker nodes will be attached to this subnet of specified network. If not specified, a network, subnet & router will be created.",
		},
		"subnet_cidr": {
			Type:        schema.TypeString,
			Computed:    true,
			Optional:    true,
			ForceNew:    true,
			Description: "Change this to configure a different internal IP range for Nodes. Default: 192.168.1.0/24",
		},
		"server_group_id": {
			Type:        schema.TypeString,
			Computed:    true,
			Optional:    true,
			Description: "Server group to use for all machines within a cluster",
		},
	}
}

func metakubeResourceClusterOpenstackCloudSpecUserCredentialsFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"project_id": {
			Type:        schema.TypeString,
			Required:    true,
			DefaultFunc: schema.EnvDefaultFunc("OS_PROJECT_ID", nil),
			Description: "The id of openstack project",
		},
		"project_name": {
			Type:        schema.TypeString,
			Required:    true,
			DefaultFunc: schema.EnvDefaultFunc("OS_PROJECT_NAME", nil),
			Description: "The name of openstack project",
		},
		"username": {
			Type:        schema.TypeString,
			DefaultFunc: schema.EnvDefaultFunc("OS_USERNAME", nil),
			Required:    true,
			Sensitive:   true,
			Description: "The openstack account's username",
		},
		"password": {
			Type:        schema.TypeString,
			DefaultFunc: schema.EnvDefaultFunc("OS_PASSWORD", nil),
			Required:    true,
			Sensitive:   true,
			Description: "The openstack account's password",
		},
	}
}

func metakubeResourceClusterOpenstackCloudSpecApplicationCredentialsFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"id": {
			Type:        schema.TypeString,
			DefaultFunc: schema.EnvDefaultFunc("OS_APPLICATION_CREDENTIAL_ID", nil),
			Required:    true,
			Description: "Openstack application credentials ID",
		},
		"secret": {
			Type:        schema.TypeString,
			DefaultFunc: schema.EnvDefaultFunc("OS_APPLICATION_CREDENTIAL_SECRET", nil),
			Required:    true,
			Sensitive:   true,
			Description: "Openstack application credentials secret",
		},
	}
}
