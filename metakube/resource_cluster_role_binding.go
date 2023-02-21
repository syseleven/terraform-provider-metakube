package metakube

import (
	"context"
	"fmt"

	"github.com/syseleven/go-metakube/client/project"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func metakubeResourceClusterRoleBinding() *schema.Resource {
	return &schema.Resource{
		CreateContext: metakubeResourceClusterRoleBindingCreate,
		ReadContext:   metakubeResourceClusterRoleBindingRead,
		DeleteContext: metakubeResourceClusterRoleBindingDelete,
		Importer: &schema.ResourceImporter{
			StateContext: importResourceWithProjectAndClusterID("cluster_role_binding_name"),
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
			"cluster_role_name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
				ForceNew:     true,
				Description:  "The name of the cluster role to bind to",
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

func metakubeResourceClusterRoleBindingCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	k := m.(*metakubeProviderMeta)

	subjects := metakubeClusterRoleBindingExpandSubjects(d.Get("subject"))
	for _, sub := range subjects {
		params := project.NewBindUserToClusterRoleV2Params().
			WithContext(ctx).
			WithProjectID(d.Get("project_id").(string)).
			WithClusterID(d.Get("cluster_id").(string)).
			WithRoleID(d.Get("cluster_role_name").(string)).
			WithBody(&sub)
		_, err := k.client.Project.BindUserToClusterRoleV2(params, k.auth)
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to create cluster role bindings: %s", stringifyResponseError(err)))
		}
	}
	d.SetId(d.Get("cluster_role_name").(string))
	return metakubeResourceClusterRoleBindingRead(ctx, d, m)
}

func metakubeResourceClusterRoleBindingRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	k := m.(*metakubeProviderMeta)

	params := project.NewListClusterRoleBindingV2Params().
		WithContext(ctx).
		WithProjectID(d.Get("project_id").(string)).
		WithClusterID(d.Get("cluster_id").(string))
	ret, err := k.client.Project.ListClusterRoleBindingV2(params, k.auth)
	if err != nil {
		return diag.FromErr(err)
	}

	for _, item := range ret.Payload {
		if item.RoleRefName == d.Id() && len(item.Subjects) != 0 {
			err := d.Set("subject", metakubeClusterRoleBindingFlattenSubjects(item.Subjects))
			if err != nil {
				return diag.FromErr(err)
			}
			d.SetId(item.RoleRefName)
			err = d.Set("cluster_role_name", item.RoleRefName)
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

func metakubeResourceClusterRoleBindingDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	k := m.(*metakubeProviderMeta)

	subjects := metakubeClusterRoleBindingExpandSubjects(d.Get("subject"))
	for _, sub := range subjects {
		params := project.NewUnbindUserFromClusterRoleBindingV2Params().
			WithContext(ctx).
			WithProjectID(d.Get("project_id").(string)).
			WithClusterID(d.Get("cluster_id").(string)).
			WithRoleID(d.Id()).
			WithBody(&sub)
		_, err := k.client.Project.UnbindUserFromClusterRoleBindingV2(params, k.auth)
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to delete cluster role binding: %s", stringifyResponseError(err)))
		}
	}
	return nil
}
