package metakube

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/syseleven/go-metakube/client/openstack"
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
							SchemaVersion: 1,
							StateUpgraders: []schema.StateUpgrader{
								{
									Version: 0,
									Type: cty.Object(map[string]cty.Type{
										"username":                   cty.String,
										"tenant":                     cty.String,
										"password":                   cty.String,
										"application_credentials_id": cty.String,
									}),
									Upgrade: func(ctx context.Context, rawState map[string]interface{}, m interface{}) (map[string]interface{}, error) {
										if v, ok := rawState["application_credential_id"].(string); ok && v != "" {
											return rawState, nil
										}

										meta := m.(*metakubeProviderMeta)
										if u, ok := rawState["username"].(string); !ok {
											return nil, fmt.Errorf("could not read 'username' %v from state %v", rawState["username"], rawState)
										} else if p, ok := rawState["password"].(string); !ok {
											return nil, fmt.Errorf("could not read 'password' %v from state %v", rawState["password"], rawState)
										} else if name, ok := rawState["tenant"].(string); !ok {
											return nil, fmt.Errorf("could not read 'tenant' %v from state %v", rawState["tenant"], rawState)
										} else {
											d := "Default"
											params := openstack.NewListOpenstackTenantsParams().WithContext(ctx).WithDomain(&d).WithUsername(&u).WithPassword(&p)
											result, err := meta.client.Openstack.ListOpenstackTenants(params, meta.auth)
											if err != nil {
												return nil, fmt.Errorf("could not get tenant '%s' id: %v", name, err)
											}
											var id string
											for _, v := range result.Payload {
												if v.Name == name {
													id = v.ID
													break
												}
											}
											if id == "" {
												return nil, fmt.Errorf("cound not find tenant '%s'", name)
											}
											delete(rawState, "tenant")
											rawState["project_id"] = id
											return rawState, nil
										}
									},
								},
							},
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
		"project_id": {
			Type:          schema.TypeString,
			Optional:      true,
			ConflictsWith: []string{"spec.0.cloud.0.openstack.0.application_credentials_id", "spec.0.cloud.0.openstack.0.application_credentials_secret"},
			DefaultFunc:   schema.EnvDefaultFunc("OS_PROJECT_ID", nil),
			Description:   "The id of opestack project to use for billing",
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
			ConflictsWith: []string{"spec.0.cloud.0.openstack.0.username", "spec.0.cloud.0.openstack.0.password", "spec.0.cloud.0.openstack.0.project_id"},
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
		"server_group_id": {
			Type:        schema.TypeString,
			Computed:    true,
			Optional:    true,
			Description: "Server group to use for all machines within a cluster",
		},
	}
}
