package datasource_k8s_version

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func DataSourceK8sVersionSchema() schema.Schema {
	return schema.Schema{
		Attributes: map[string]schema.Attribute{
			"major": schema.StringAttribute{
				Optional:    true,
				Description: "Kubernetes cluster major version",
			},
			"minor": schema.StringAttribute{
				Optional:    true,
				Description: "Kubernetes cluster minor version",
				Validators: []validator.String{
					stringvalidator.AlsoRequires(path.MatchRoot("major")),
				},
			},
			"version": schema.StringAttribute{
				Computed:    true,
				Description: "The latest version of kubernetes cluster that satisfies specification and supported by MetaKube",
			},
		},
	}
}

type metakubeK8sVersionDataSourceModel struct {
	Major   types.String `tfsdk:"major"`
	Minor   types.String `tfsdk:"minor"`
	Version types.String `tfsdk:"version"`
}
