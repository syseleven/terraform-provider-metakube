package datasource_project

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/syseleven/go-metakube/client/project"
	"github.com/syseleven/terraform-provider-metakube/metakube/common"
)

func DataSourceMetakubeProject() *schema.Resource {
	return &schema.Resource{
		ReadContext: metakubeDataSourceProjectRead,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
			},
		},
	}
}

func metakubeDataSourceProjectRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	meta := m.(*common.MetaKubeProviderMeta)

	p := project.NewListProjectsParams().WithContext(ctx)
	res, err := meta.Client.Project.ListProjects(p, meta.Auth)
	if err != nil {
		return diag.FromErr(err)
	}

	name := d.Get("name").(string)
	matches := 0
	for _, r := range res.Payload {
		if r != nil && r.Name == name {
			d.SetId(r.ID)
			matches++
		}
	}

	if matches == 0 {
		return diag.Errorf("Could not find a project with name: %s", name)
	} else if matches > 1 {
		return diag.Errorf("Found multiple projects with name: %s", name)
	}

	return nil
}
