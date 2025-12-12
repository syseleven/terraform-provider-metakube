package datasource_sshkey

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/syseleven/go-metakube/client/project"
	"github.com/syseleven/terraform-provider-metakube/metakube/common"
)

var (
	_ datasource.DataSource = &metakubeSSHKeyDataSource{}
)

func NewSSHKeyDataSource() datasource.DataSource {
	return &metakubeSSHKeyDataSource{}
}

type metakubeSSHKeyDataSource struct {
	meta *common.MetaKubeProviderMeta
}

func (d *metakubeSSHKeyDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "metakube_sshkey"
}

func (d *metakubeSSHKeyDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = DataSourceSSHKeySchema()
}

func (d *metakubeSSHKeyDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *metakubeSSHKeyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data metakubeSSHKeyDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	prj := data.ProjectID.ValueString()
	prms := project.NewListSSHKeysParams().WithContext(ctx).WithProjectID(prj)
	res, err := d.meta.Client.Project.ListSSHKeys(prms, d.meta.Auth)
	if err != nil {
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Failed to list SSH keys: %s", common.StringifyResponseError(err)))
		return
	}

	name := data.Name.ValueString()
	for _, r := range res.Payload {
		if r != nil && r.Name == name {
			data.ID = types.StringValue(r.ID)
			data.ProjectID = types.StringValue(prj)
			data.Name = types.StringValue(r.Name)
			data.PublicKey = types.StringValue(r.Spec.PublicKey)
			data.Fingerprint = types.StringValue(r.Spec.Fingerprint)

			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
	}

	resp.Diagnostics.AddError(fmt.Sprintf("Could not find sshkey with name '%s' in a project with id '%s'", name, prj), "")
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
