package metakube

import (
	"context"
	"strings"

	"golang.org/x/mod/semver"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/syseleven/go-metakube/client/versions"
)

func dataSourceMetakubeK8sClusterVersion() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceMetakubeK8sClusterVersionRead,
		Schema: map[string]*schema.Schema{
			"major": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Kubernetes cluster major version",
			},
			"minor": {
				Type:         schema.TypeString,
				Optional:     true,
				RequiredWith: []string{"major"},
				Description:  "Kubernetes cluster minor version",
			},
			"version": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The latest version of kubernetes cluster that satisfies specification and supported by MetaKube",
			},
		},
	}
}

func dataSourceMetakubeK8sClusterVersionRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	k := meta.(*metakubeProviderMeta)

	partialVersionSpec := ""
	if v, ok := d.GetOk("major"); ok {
		partialVersionSpec = v.(string)
	}
	if v, ok := d.GetOk("minor"); ok {
		partialVersionSpec += "." + v.(string)
	}

	p := versions.NewGetMasterVersionsParams().WithContext(ctx)
	r, err := k.client.Versions.GetMasterVersions(p, k.auth)
	if err != nil {
		return diag.Errorf("%s", stringifyResponseError(err))
	}

	var all []string
	for _, item := range r.Payload {
		if item != nil {
			all = append(all, item.Version.(string))
		}
	}

	var available []string
	for _, v := range all {
		if strings.Index(v, partialVersionSpec) == 0 {
			available = append(available, v)
		}
	}

	if len(available) == 0 {
		return diag.Errorf("found following versions but did not match specification: %s", strings.Join(all, " "))
	}

	latest := available[0]
	for _, v := range available {
		if semver.Compare("v"+v, "v"+latest) > 0 {
			latest = v
		}
	}

	d.SetId(latest)
	d.Set("version", latest)

	return nil
}
