package datasource_k8s_version

import (
	"context"
	"fmt"
	"strings"

	"golang.org/x/mod/semver"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/syseleven/go-metakube/client/versions"
	"github.com/syseleven/terraform-provider-metakube/metakube/common"
)

var (
	_ datasource.DataSource              = &metakubeK8sClusterVersionDataSource{}
	_ datasource.DataSourceWithConfigure = &metakubeK8sClusterVersionDataSource{}
)

func NewK8sClusterVersionDataSource() datasource.DataSource {
	return &metakubeK8sClusterVersionDataSource{}
}

type metakubeK8sClusterVersionDataSource struct {
	meta *common.MetaKubeProviderMeta
}

func (d *metakubeK8sClusterVersionDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "metakube_k8s_version"
}

func (d *metakubeK8sClusterVersionDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = DataSourceK8sVersionSchema()
}

func (d *metakubeK8sClusterVersionDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	meta, ok := req.ProviderData.(*common.MetaKubeProviderMeta)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *common.MetaKubeProviderMeta, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.meta = meta
}

func (d *metakubeK8sClusterVersionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data metakubeK8sVersionDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var partialVersionSpec string
	if !data.Major.IsNull() && !data.Major.IsUnknown() {
		partialVersionSpec = data.Major.ValueString()
	}
	if !data.Minor.IsNull() && !data.Minor.IsUnknown() {
		partialVersionSpec += "." + data.Minor.ValueString()
	}

	if partialVersionSpec == "" {
		resp.Diagnostics.AddError("missing major version", "major version is required")
		return
	}

	p := versions.NewGetMasterVersionsParams().WithContext(ctx)
	r, err := d.meta.Client.Versions.GetMasterVersions(p, d.meta.Auth)
	if err != nil {
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Failed to get master versions: %s", common.StringifyResponseError(err)))
		return
	}

	var all []string
	for _, item := range r.Payload {
		if item != nil {
			all = append(all, item.Version)
		}
	}

	var available []string
	for _, v := range all {
		if strings.Index(v, partialVersionSpec) == 0 {
			available = append(available, v)
		}
	}

	if len(available) == 0 {
		resp.Diagnostics.AddError("No Matching Version", fmt.Sprintf("Found following versions but did not match specification: %s", strings.Join(all, " ")))
		return
	}

	latest := available[0]
	for _, v := range available {
		if semver.Compare("v"+v, "v"+latest) > 0 {
			latest = v
		}
	}

	data.Version = types.StringValue(latest)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
