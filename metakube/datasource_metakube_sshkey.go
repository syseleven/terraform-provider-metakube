package metakube

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/syseleven/go-metakube/client/project"
)

func dataSourceMetakubeSSHKey() *schema.Resource {
	return &schema.Resource{
		ReadContext: metakubeDataSourceSSHKeyRead,
		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
			},

			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
			},

			"public_key": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"fingerprint": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func metakubeDataSourceSSHKeyRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	meta := m.(*metakubeProviderMeta)

	prj := d.Get("project_id").(string)
	prms := project.NewListSSHKeysParams().WithContext(ctx).WithProjectID(prj)
	res, err := meta.client.Project.ListSSHKeys(prms, meta.auth)
	if err != nil {
		return diag.FromErr(err)
	}

	name := d.Get("name").(string)
	for _, r := range res.Payload {
		if r != nil && r.Name == name {
			d.SetId(r.ID)
			d.Set("public_key", r.Spec.PublicKey)
			d.Set("name", name)
			d.Set("project_id", prj)
			d.Set("fingerprint", r.Spec.Fingerprint)
			return nil
		}
	}

	return diag.Errorf("Could not find sshkey with name '%s' in a project with id '%s'", d.Get("name").(string), prj)
}
