package metakube

import (
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func metakubeResourceSystemLabelOrTag(key string) bool {
	for _, s := range []string{"labels.%", "metakube", "system-", "system/", "kubernetes.io"} {
		if strings.Contains(key, s) {
			return true
		}
	}
	return false
}

func matakubeResourceNodeDeploymentSpecFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"dynamic_config": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: "Enable metakube kubelete dynamic config",
		},
		"replicas": {
			Type:          schema.TypeInt,
			Optional:      true,
			Default:       3,
			Description:   "Number of replicas",
			ConflictsWith: []string{"spec.0.min_replicas", "spec.0.max_replicas"},
			DiffSuppressFunc: func(_, _, _ string, d *schema.ResourceData) bool {
				minv, ok1 := d.GetOkConfigured("spec.0.min_replicas")
				maxv, ok2 := d.GetOkConfigured("spec.0.max_replicas")
				return ok1 && minv.(int) > 0 && ok2 && maxv.(int) > 0
			},
		},
		"min_replicas": {
			Type:         schema.TypeInt,
			Optional:     true,
			ValidateFunc: validation.IntAtLeast(0),
			Description:  "Minimum number of replicas to downscale",
			RequiredWith: []string{"spec.0.max_replicas"},
			DiffSuppressFunc: func(_, _, _ string, d *schema.ResourceData) bool {
				_, ok := d.GetOk("spec.0.max_replicas")
				return !ok
			},
		},
		"max_replicas": {
			Type:         schema.TypeInt,
			Optional:     true,
			ValidateFunc: validation.IntAtLeast(1),
			Description:  "Maximum number of replicas to scale up",
			RequiredWith: []string{"spec.0.min_replicas"},
		},
		"template": {
			Type:        schema.TypeList,
			MaxItems:    1,
			Required:    true,
			Description: "Template specification",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"cloud": {
						Type:        schema.TypeList,
						MaxItems:    1,
						Required:    true,
						Description: "Cloud specification",
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"aws": {
									Type:        schema.TypeList,
									Optional:    true,
									MaxItems:    1,
									Description: "AWS node deployment specification",
									Elem: &schema.Resource{
										Schema: matakubeResourceNodeDeploymentAWSSchema(),
									},
								},
								"openstack": {
									Type:        schema.TypeList,
									Optional:    true,
									MaxItems:    1,
									Description: "OpenStack node deployment specification",
									Elem: &schema.Resource{
										Schema: matakubeResourceNodeDeploymentCloudOpenstackSchema(),
									},
								},
								"azure": metakubeResourceNodeDeploymentAzureSchema(),
							},
						},
					},
					"operating_system": {
						Type:        schema.TypeList,
						Required:    true,
						MinItems:    1,
						MaxItems:    1,
						Description: "Operating system",
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"ubuntu": {
									Type:         schema.TypeList,
									Optional:     true,
									MinItems:     1,
									MaxItems:     1,
									ExactlyOneOf: []string{"spec.0.template.0.operating_system.0.ubuntu", "spec.0.template.0.operating_system.0.flatcar"},
									Description:  "Ubuntu operating system",
									Elem: &schema.Resource{
										Schema: map[string]*schema.Schema{
											"dist_upgrade_on_boot": {
												Type:        schema.TypeBool,
												Optional:    true,
												Default:     false,
												Description: "Upgrade operating system on boot",
											},
										},
									},
								},
								"flatcar": {
									Type:         schema.TypeList,
									Optional:     true,
									MinItems:     1,
									MaxItems:     1,
									ExactlyOneOf: []string{"spec.0.template.0.operating_system.0.ubuntu", "spec.0.template.0.operating_system.0.flatcar"},
									Description:  "Flatcar operating system",
									Elem: &schema.Resource{
										Schema: map[string]*schema.Schema{
											"disable_auto_update": {
												Type:        schema.TypeBool,
												Optional:    true,
												Default:     false,
												Description: "Disable flatcar auto update feature",
											},
										},
									},
								},
							},
						},
					},
					"versions": {
						Type:        schema.TypeList,
						Optional:    true,
						Computed:    true,
						MaxItems:    1,
						Description: "Cloud components versions",
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"kubelet": {
									Type:        schema.TypeString,
									Optional:    true,
									Computed:    true,
									Description: "Kubelet version",
								},
							},
						},
					},
					"labels": {
						Type:     schema.TypeMap,
						Optional: true,
						Description: "Map of string keys and values that can be used to organize and categorize (scope and select) objects. " +
							"It will be applied to Nodes allowing users run their apps on specific Node using labelSelector.",
						Elem: &schema.Schema{
							Type: schema.TypeString,
						},
						DiffSuppressFunc: func(k, _, _ string, _ *schema.ResourceData) bool {
							return metakubeResourceSystemLabelOrTag(k)
						},
						ValidateFunc: func(v interface{}, k string) (strings []string, errors []error) {
							l := v.(map[string]interface{})
							for key := range l {
								if metakubeResourceSystemLabelOrTag(key) {
									errors = append(errors, fmt.Errorf("%s is reserved for system and can't be used", key))
								}
							}
							return
						},
					},
					"taints": {
						Type:        schema.TypeList,
						Optional:    true,
						Description: "List of taints to set on new nodes",
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"effect": {
									Type:         schema.TypeString,
									Required:     true,
									Description:  "Taint effect",
									ValidateFunc: validation.StringInSlice([]string{"NoSchedule", "PreferNoSchedule", "NoExecute"}, false),
								},
								"key": {
									Type:         schema.TypeString,
									Required:     true,
									ValidateFunc: validation.NoZeroValues,
									Description:  "Taint key",
								},
								"value": {
									Type:         schema.TypeString,
									Required:     true,
									ValidateFunc: validation.NoZeroValues,
									Description:  "Taint value",
								},
							},
						},
					},
				},
			},
		},
	}
}

func matakubeResourceNodeDeploymentAWSSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"instance_type": {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.NoZeroValues,
			Description:  "EC2 instance type",
		},
		"disk_size": {
			Type:         schema.TypeInt,
			Required:     true,
			ValidateFunc: validation.NoZeroValues,
			Description:  "Size of the volume in GBs. Only one volume will be created",
		},
		"volume_type": {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.NoZeroValues,
			Description:  "EBS volume type",
		},
		"availability_zone": {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.NoZeroValues,
			Description:  "Availability zone in which to place the node. It is coupled with the subnet to which the node will belong",
		},
		"subnet_id": {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.NoZeroValues,
			Description:  "The VPC subnet to which the node shall be connected",
		},
		"assign_public_ip": {
			Type:     schema.TypeBool,
			Optional: true,
			Default:  true,
			Description: "Flag which controls a property of the AWS instance. When set the AWS instance will get a public IP address " +
				"assigned during launch overriding a possible setting in the used AWS subnet.",
		},
		"ami": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Amazon Machine Image to use. Will be defaulted to an AMI of your selected operating system and region",
		},
		"tags": {
			Type:        schema.TypeMap,
			Optional:    true,
			Computed:    true,
			Description: "Additional instance tags",
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
			DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
				return metakubeResourceSystemLabelOrTag(k)
			},
			ValidateFunc: func(v interface{}, k string) (strings []string, errors []error) {
				l := v.(map[string]interface{})
				for key := range l {
					if metakubeResourceSystemLabelOrTag(key) {
						errors = append(errors, fmt.Errorf("%s is reserved for system and can't be used", key))
					}
				}
				return
			},
		},
	}
}

func matakubeResourceNodeDeploymentCloudOpenstackSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"flavor": {
			Type:         schema.TypeString,
			Required:     true,
			Description:  "Instance type",
			ValidateFunc: validation.NoZeroValues,
		},
		"image": {
			Type:         schema.TypeString,
			Required:     true,
			Description:  "Image to use",
			ValidateFunc: validation.NoZeroValues,
		},
		"disk_size": {
			Type:         schema.TypeInt,
			Optional:     true,
			ValidateFunc: validation.IntAtLeast(1),
			Description:  "If set, the rootDisk will be a volume. If not, the rootDisk will be on ephemeral storage and its size will be derived from the flavor",
		},
		"tags": {
			Type:        schema.TypeMap,
			Optional:    true,
			Computed:    true,
			Description: "Additional instance tags",
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
			DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
				return metakubeResourceSystemLabelOrTag(k)
			},
			ValidateFunc: func(v interface{}, k string) (strings []string, errors []error) {
				l := v.(map[string]interface{})
				for key := range l {
					if metakubeResourceSystemLabelOrTag(key) {
						errors = append(errors, fmt.Errorf("%s is reserved for system and can't be used", key))
					}
				}
				return
			},
		},
		"use_floating_ip": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     true,
			Description: "Indicate use of floating ip in case of floating_ip_pool presense",
		},
		"instance_ready_check_period": {
			Type:             schema.TypeString,
			Optional:         true,
			Default:          "5s",
			Description:      "Specifies how often should the controller check if instance is ready before timing out",
			ValidateDiagFunc: isNonEmptyDurationString,
		},
		"instance_ready_check_timeout": {
			Type:             schema.TypeString,
			Optional:         true,
			Default:          "120s",
			Description:      "Specifies how long should the controller check if instance is ready before timing out",
			ValidateDiagFunc: isNonEmptyDurationString,
		},
	}
}

func isNonEmptyDurationString(v interface{}, p cty.Path) diag.Diagnostics {
	if vv, ok := v.(string); ok {
		_, err := time.ParseDuration(vv)
		if err != nil {
			return diag.Diagnostics{
				diag.Diagnostic{
					Severity:      diag.Error,
					Summary:       fmt.Sprintf("could not parse '%s': %v", vv, err),
					AttributePath: p,
				},
			}
		}
		return nil
	}
	return diag.Diagnostics{
		diag.Diagnostic{
			Severity:      diag.Error,
			Summary:       fmt.Sprintf("Should be a duration string"),
			AttributePath: p,
		},
	}
}

func metakubeResourceNodeDeploymentAzureSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Optional:    true,
		MaxItems:    1,
		Description: "Azure node deployment specification",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"image_id": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: "Node image id",
				},
				"size": {
					Type:         schema.TypeString,
					Required:     true,
					ValidateFunc: validation.NoZeroValues,
					Description:  "VM size",
				},
				"assign_public_ip": {
					Type:        schema.TypeBool,
					Optional:    true,
					Default:     false,
					Description: "whether to have public facing IP or not",
				},
				"disk_size_gb": {
					Type:        schema.TypeInt,
					Optional:    true,
					Default:     0,
					ForceNew:    true,
					Description: "Data disk size in GB",
				},
				"os_disk_size_gb": {
					Type:        schema.TypeInt,
					Optional:    true,
					Default:     0,
					ForceNew:    true,
					Description: "OS disk size in GB",
				},
				"tags": {
					Type:        schema.TypeMap,
					Optional:    true,
					Computed:    true,
					Description: "Additional metadata to set",
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
					DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
						return metakubeResourceSystemLabelOrTag(k)
					},
					ValidateFunc: func(v interface{}, k string) (strings []string, errors []error) {
						l := v.(map[string]interface{})
						for key := range l {
							if metakubeResourceSystemLabelOrTag(key) {
								errors = append(errors, fmt.Errorf("%s is reserved for system and can't be used", key))
							}
						}
						return
					},
				},
				"zones": {
					Type:        schema.TypeList,
					Optional:    true,
					Computed:    true,
					Description: "Represents the availablity zones for azure vms",
					Elem:        &schema.Schema{Type: schema.TypeString},
				},
			},
		},
	}
}
