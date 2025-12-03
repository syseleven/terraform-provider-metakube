package datasource_k8s_version

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func DataSourceK8sVersionSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Attributes: map[string]schema.Attribute{
			"major": schema.StringAttribute{
				Optional:    true,
				Description: "Kubernetes cluster major version",
			},
			"minor": schema.StringAttribute{
				Optional:    true,
				Description: "Kubernetes cluster minor version",
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
