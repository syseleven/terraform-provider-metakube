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
			Type:        schema.TypeString,
			Required:    true,
			Description: "Cloud orchestrator version, either Kubernetes or OpenShift",
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
		"machine_networks": {
			Type:        schema.TypeList,
			Optional:    true,
			ForceNew:    true,
			Description: "Machine networks optionally specifies the parameters for IPAM",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"cidr": {
						Type:        schema.TypeString,
						Optional:    true,
						Description: "Network CIDR",
					},
					"gateway": {
						Type:        schema.TypeString,
						Optional:    true,
						Description: "Network gateway",
					},
					"dns_servers": {
						Type:        schema.TypeSet,
						Optional:    true,
						Description: "DNS servers",
						Elem:        schema.TypeString,
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
		"domain_name": {
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    true,
			Computed:    true,
			Description: "Internal IP range for ClusterIP Pods",
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
			Type:     schema.TypeString,
			Required: true,
		},
		"client_secret": {
			Type:      schema.TypeString,
			Required:  true,
			Sensitive: true,
		},
		"subscription_id": {
			Type:     schema.TypeString,
			Required: true,
		},
		"tenant_id": {
			Type:     schema.TypeString,
			Required: true,
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
			Type:        schema.TypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Access key identifier",
		},
		"secret_access_key": {
			Type:        schema.TypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Secret access key",
		},
		"vpc_id": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Virtual private cloud identifier",
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
		"tenant": {
			Type:          schema.TypeString,
			Optional:      true,
			ConflictsWith: []string{"spec.0.cloud.0.openstack.0.application_credentials_id", "spec.0.cloud.0.openstack.0.application_credentials_secret"},
			DefaultFunc:   schema.EnvDefaultFunc("OS_PROJECT_NAME", nil),
			Description:   "The opestack project to use for billing",
		},
		"username": {
			Type:          schema.TypeString,
			DefaultFunc:   schema.EnvDefaultFunc("OS_USERNAME", nil),
			Optional:      true,
			ConflictsWith: []string{"spec.0.cloud.0.openstack.0.application_credentials_id", "spec.0.cloud.0.openstack.0.application_credentials_secret"},
			Sensitive:     true,
			Description:   "The openstack account's username",
		},
		"password": {
			Type:          schema.TypeString,
			DefaultFunc:   schema.EnvDefaultFunc("OS_PASSWORD", nil),
			ConflictsWith: []string{"spec.0.cloud.0.openstack.0.application_credentials_id", "spec.0.cloud.0.openstack.0.application_credentials_secret"},
			Optional:      true,
			Sensitive:     true,
			Description:   "The openstack account's password",
		},
		"application_credentials_id": {
			Type:          schema.TypeString,
			ConflictsWith: []string{"spec.0.cloud.0.openstack.0.username", "spec.0.cloud.0.openstack.0.password", "spec.0.cloud.0.openstack.0.tenant"},
			Optional:      true,
			Description:   "Openstack application credentials ID",
		},
		"application_credentials_secret": {
			Type:        schema.TypeString,
			Optional:    true,
			Sensitive:   true,
			Description: "Openstack application credentials secret",
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
	}
}
