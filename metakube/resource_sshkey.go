package metakube

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/syseleven/go-metakube/client/project"
	"github.com/syseleven/go-metakube/models"
)

func metakubeResourceSSHKey() *schema.Resource {
	return &schema.Resource{
		CreateContext: metakubeResourceSSHKeyCreate,
		ReadContext:   metakubeResourceSSHKeyRead,
		DeleteContext: metakubeResourceSSHKeyDelete,
		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
				parts := strings.Split(d.Id(), ":")
				switch len(parts) {
				case 1:
					d.SetId(parts[0])
					return []*schema.ResourceData{d}, nil
				case 2:
					d.Set("project_id", parts[0])
					d.SetId(parts[1])
					return []*schema.ResourceData{d}, nil
				default:
					return nil, fmt.Errorf("please provide resource identifier in format 'project_id:sshkey_id' or 'sshkey_id'")
				}
			},
		},

		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
				ForceNew:     true,
			},

			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
				ForceNew:     true,
			},

			"public_key": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
				DiffSuppressFunc: func(_, old, new string, _ *schema.ResourceData) bool {
					return strings.TrimSpace(old) == strings.TrimSpace(new)
				},
				ForceNew: true,
			},
		},
	}
}

func metakubeResourceSSHKeyCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	k := m.(*metakubeProviderMeta)
	p := project.NewCreateSSHKeyParams()
	p.SetProjectID(d.Get("project_id").(string))
	p.Key = &models.SSHKey{
		Name: d.Get("name").(string),
		Spec: &models.SSHKeySpec{
			PublicKey: d.Get("public_key").(string),
		},
	}
	created, err := k.client.Project.CreateSSHKey(p, k.auth)
	if err != nil {
		return diag.Errorf("unable to create SSH key: %s", stringifyResponseError(err))
	}
	d.SetId(created.Payload.ID)
	return metakubeResourceSSHKeyRead(ctx, d, m)
}

func metakubeResourceSSHKeyRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	meta := m.(*metakubeProviderMeta)
	sshkey, err := metakubeResourceSSHKeyFindByID(ctx, d, meta)
	if err != nil {
		return diag.FromErr(err)
	}
	if sshkey == nil {
		d.SetId("")
		return nil
	}
	_ = d.Set("name", sshkey.Name)
	_ = d.Set("public_key", sshkey.Spec.PublicKey)
	return nil
}

func metakubeResourceSSHKeyFindByID(ctx context.Context, d *schema.ResourceData, meta *metakubeProviderMeta) (*models.SSHKey, error) {
	prj := d.Get("project_id").(string)
	if prj == "" {
		var err error
		prj, err = metakubeResourceSSHKeyFindProjectID(ctx, d.Id(), meta)
		if err != nil {
			return nil, err
		}
		if prj == "" {
			return nil, nil
		}
		_ = d.Set("project_id", prj)
	}

	const pending, target = "Unavailable", "Ready"
	listStateConf := &retry.StateChangeConf{
		Pending: []string{
			pending,
		},
		Target: []string{
			target,
		},
		Refresh: func() (interface{}, string, error) {
			prms := project.NewListSSHKeysParams().WithContext(ctx).WithProjectID(prj)
			res, err := meta.client.Project.ListSSHKeys(prms, meta.auth)
			if err != nil {
				// wait for the RBACs
				if _, ok := err.(*project.ListSSHKeysForbidden); ok {
					return res, pending, nil
				}
				return nil, pending, fmt.Errorf("list ssh keys: %s", stringifyResponseError(err))
			}
			return res, target, nil
		},
		Timeout: d.Timeout(schema.TimeoutRead),
		Delay:   requestDelay,
	}
	s, err := listStateConf.WaitForStateContext(ctx)
	if err != nil {
		meta.log.Debugf("error while waiting for the SSH keys: %v", err)
		return nil, fmt.Errorf("error while waiting for the SSH keys: %v", err)
	}
	keys := s.(*project.ListSSHKeysOK)

	for _, r := range keys.Payload {
		if r.ID == d.Id() {
			return r, nil
		}
	}

	return nil, nil
}

func metakubeResourceSSHKeyFindProjectID(ctx context.Context, id string, meta *metakubeProviderMeta) (string, error) {
	r, err := meta.client.Project.ListProjects(project.NewListProjectsParams(), meta.auth)
	if err != nil {
		return "", fmt.Errorf("list projects: %v", err)
	}

	for _, project := range r.Payload {
		ok, err := metakubeResourceSSHKeyBelongsToProject(ctx, project.ID, id, meta)
		if ok {
			return project.ID, nil
		}
		if err != nil {
			return "", err
		}
	}

	meta.log.Infof("owner project for service account with id(%s) not found", id)
	return "", nil
}

func metakubeResourceSSHKeyBelongsToProject(ctx context.Context, prj, id string, meta *metakubeProviderMeta) (bool, error) {
	prms := project.NewListSSHKeysParams().WithContext(ctx).WithProjectID(prj)
	r, err := meta.client.Project.ListSSHKeys(prms, meta.auth)
	if err != nil {
		return false, fmt.Errorf("list sshkeys: %s", stringifyResponseError(err))
	}

	for _, i := range r.Payload {
		if i.ID == id {
			return true, nil
		}
	}

	return false, nil
}

func metakubeResourceSSHKeyDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	k := m.(*metakubeProviderMeta)
	p := project.NewDeleteSSHKeyParams()
	p.SetContext(ctx)
	p.SetProjectID(d.Get("project_id").(string))
	p.SetSSHKeyID(d.Id())
	_, err := k.client.Project.DeleteSSHKey(p, k.auth)
	if err != nil {
		return diag.Errorf("unable to delete SSH key: %s", stringifyResponseError(err))
	}
	return nil
}
