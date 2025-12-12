package datasource_project

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/syseleven/go-metakube/client/project"
	"github.com/syseleven/terraform-provider-metakube/metakube/common"
)

var (
	_ datasource.DataSource = &metakubeProjectDataSource{}
)

func NewProjectDataSource() datasource.DataSource {
	return &metakubeProjectDataSource{}
}

type metakubeProjectDataSource struct {
	meta *common.MetaKubeProviderMeta
}

func (d *metakubeProjectDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "metakube_project"
}

func (d *metakubeProjectDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = DataSourceProjectSchema()
}

func (d *metakubeProjectDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	meta, ok := req.ProviderData.(*common.MetaKubeProviderMeta)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Provider Data Type",
			fmt.Sprintf("Expected *common.MetaKubeProviderMeta, got: %T", req.ProviderData),
		)
		return
	}

	d.meta = meta
}

func (d *metakubeProjectDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data metakubeProjectDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	p := project.NewListProjectsParams().WithContext(ctx)
	res, err := d.meta.Client.Project.ListProjects(p, d.meta.Auth)
	if err != nil {
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Failed to list projects: %s", common.StringifyResponseError(err)))
		return
	}

	name := data.Name.ValueString()
	matches := 0
	for _, r := range res.Payload {
		if r != nil && r.Name == name {
			data.ID = types.StringValue(r.ID)
			matches++
		}
	}

	if matches == 0 {
		resp.Diagnostics.AddError("Could not find a project with name: %s", name)
	} else if matches > 1 {
		resp.Diagnostics.AddError("Found multiple projects with name: %s", name)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
