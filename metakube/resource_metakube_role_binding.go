package metakube

import (
	"context"
	"fmt"
	"strings"

	"github.com/syseleven/go-metakube/client/project"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func metakubeResourceRoleBinding() *schema.Resource {
	return &schema.Resource{
		CreateContext: metakubeResourceRoleBindingCreate,
		ReadContext:   metakubeResourceRoleBindingRead,
		DeleteContext: metakubeResourceRoleBindingDelete,
		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
				parts := strings.Split(d.Id(), ":")
				if len(parts) != 4 {
					return nil, fmt.Errorf("please provide resource identifier in format 'project_id:cluster_id:role_namespace:role_name'")
				}
				d.Set("project_id", parts[0])
				d.Set("cluster_id", parts[1])
				d.SetId(parts[2] + ":" + parts[3])
				return []*schema.ResourceData{d}, nil
			},
		},

		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
				ForceNew:     true,
				Description:  "The id of the project resource belongs to",
			},
			"cluster_id": {
				Type:         schema.TypeString,
				ValidateFunc: validation.NoZeroValues,
				Required:     true,
				ForceNew:     true,
				Description:  "The id of the cluster resource belongs to",
			},
			"namespace": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
				ForceNew:     true,
				Description:  "The name of the namespace",
			},
			"role_name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
				ForceNew:     true,
				Description:  "The name of the role to bind to",
			},
			"subject": {
				Type:        schema.TypeList,
				Required:    true,
				ForceNew:    true,
				Description: "Users and groups to bind for",
				MinItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"kind": {
							Type:         schema.TypeString,
							Required:     true,
							Description:  "Can be either 'user' or 'group'",
							ValidateFunc: validation.StringInSlice([]string{"user", "group"}, false),
						},
						"name": {
							Type:         schema.TypeString,
							Optional:     true,
							Description:  "Subject name",
							ValidateFunc: validation.NoZeroValues,
						},
					},
				},
			},
		},
	}
}

func metakubeResourceRoleBindingCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	k := m.(*metakubeProviderMeta)

	subjects := metakubeRoleBindingExpandSubjects(d.Get("subject"))
	for _, sub := range subjects {
		params := project.NewBindUserToRoleV2Params().
			WithContext(ctx).
			WithProjectID(d.Get("project_id").(string)).
			WithClusterID(d.Get("cluster_id").(string)).
			WithNamespace(d.Get("namespace").(string)).
			WithRoleID(d.Get("role_name").(string)).
			WithBody(&sub)
		_, err := k.client.Project.BindUserToRoleV2(params, k.auth)
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to create role bindings: %s", stringifyResponseError(err)))
		}
	}
	d.SetId(d.Get("namespace").(string) + ":" + d.Get("role_name").(string))
	return metakubeResourceRoleBindingRead(ctx, d, m)
}

func metakubeResourceRoleBindingRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	k := m.(*metakubeProviderMeta)

	params := project.NewListRoleBindingV2Params().
		WithContext(ctx).
		WithProjectID(d.Get("project_id").(string)).
		WithClusterID(d.Get("cluster_id").(string))
	ret, err := k.client.Project.ListRoleBindingV2(params, k.auth)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to list role bindings: %s", stringifyResponseError(err)))
	}

	idParts := strings.Split(d.Id(), ":")
	namespace := idParts[0]
	roleName := idParts[1]
	for _, item := range ret.Payload {
		if item.Namespace == namespace && item.RoleRefName == roleName && len(item.Subjects) != 0 {
			err := d.Set("subject", metakubeClusterRoleBindingFlattenSubjects(item.Subjects))
			if err != nil {
				return diag.FromErr(err)
			}
			d.SetId(item.Namespace + ":" + item.RoleRefName)
			err = d.Set("namespace", item.Namespace)
			if err != nil {
				return diag.FromErr(err)
			}
			err = d.Set("role_name", item.RoleRefName)
			if err != nil {
				return diag.FromErr(err)
			}
			return nil
		}
	}

	// Signal record was not found by setting id = ""
	d.SetId("")
	return nil
}

func metakubeResourceRoleBindingDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	k := m.(*metakubeProviderMeta)

	subjects := metakubeRoleBindingExpandSubjects(d.Get("subject"))
	idParts := strings.Split(d.Id(), ":")
	namespace := idParts[0]
	roleName := idParts[1]
	for _, sub := range subjects {
		params := project.NewUnbindUserFromRoleBindingV2Params().
			WithContext(ctx).
			WithProjectID(d.Get("project_id").(string)).
			WithClusterID(d.Get("cluster_id").(string)).
			WithNamespace(namespace).
			WithRoleID(roleName).
			WithBody(&sub)
		_, err := k.client.Project.UnbindUserFromRoleBindingV2(params, k.auth)
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to delete role binding: %s", stringifyResponseError(err)))
		}
	}
	return nil
}
