package metakube

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	frameworkSchema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/syseleven/terraform-provider-metakube/metakube/common"
	"github.com/syseleven/terraform-provider-metakube/metakube/datasources/datasource_k8s_version"
	"github.com/syseleven/terraform-provider-metakube/metakube/datasources/datasource_project"
	"github.com/syseleven/terraform-provider-metakube/metakube/datasources/datasource_sshkey"
	"github.com/syseleven/terraform-provider-metakube/metakube/resources/resource_cluster"
	"github.com/syseleven/terraform-provider-metakube/metakube/resources/resource_cluster_role_binding"
	"github.com/syseleven/terraform-provider-metakube/metakube/resources/resource_maintenance_cronjob"
	"github.com/syseleven/terraform-provider-metakube/metakube/resources/resource_node_deployment"
	"github.com/syseleven/terraform-provider-metakube/metakube/resources/resource_role_binding"
	"github.com/syseleven/terraform-provider-metakube/metakube/resources/resource_sshkey"
)

// Terraform Plugin Framework Provider

var _ provider.Provider = &metakubeProvider{}

type metakubeProvider struct{}

func NewFrameworkProvider() provider.Provider {
	return &metakubeProvider{}
}

func (p *metakubeProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = frameworkSchema.Schema{
		Attributes: map[string]frameworkSchema.Attribute{
			"host": frameworkSchema.StringAttribute{
				Description: "The hostname of MetaKube API (in form of URI)",
				Optional:    true,
			},
			"token": frameworkSchema.StringAttribute{
				Description: "The MetaKube authentication token",
				Optional:    true,
				Sensitive:   true,
			},
			"token_path": frameworkSchema.StringAttribute{
				Description: "Path to the MetaKube authentication token, defaults to ~/.metakube/auth",
				Optional:    true,
			},
			"development": frameworkSchema.BoolAttribute{
				Description: "Run development mode.",
				Optional:    true,
			},
			"debug": frameworkSchema.BoolAttribute{
				Description: "Run debug mode.",
				Optional:    true,
			},
			"log_path": frameworkSchema.StringAttribute{
				Description: "Path to store logs",
				Optional:    true,
			},
		},
	}
}

func (p *metakubeProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config common.MetakubeProviderConfig

	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check for unknown values
	if config.Host.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("host"),
			"Unknown MetaKube host",
			"The MetaKube host is unknown",
		)
	}

	if config.Token.IsUnknown() || config.TokenPath.IsUnknown() {
		if config.Token.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root("token"),
				"Unknown MetaKube token",
				"The MetaKube token is unknown",
			)
		}
		if config.TokenPath.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root("token_path"),
				"Unknown MetaKube token path",
				"The MetaKube token path is unknown",
			)
		}
	}

	if config.Development.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("development"),
			"Unknown MetaKube development mode",
			"The MetaKube development mode is unknown",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Set default values
	host := config.Host.ValueString()
	if host == "" {
		host = os.Getenv("METAKUBE_HOST")
		if host == "" {
			host = "https://metakube.syseleven.de"
		}
	}

	token := config.Token.ValueString()
	if token == "" {
		token = os.Getenv("METAKUBE_TOKEN")
	}

	tokenPath := config.TokenPath.ValueString()
	if tokenPath == "" {
		tokenPath = os.Getenv("METAKUBE_TOKEN_PATH")
		if tokenPath == "" {
			tokenPath = "~/.metakube/auth"
		}
	}

	var k common.MetaKubeProviderMeta

	var err error
	k.Log, err = common.NewLogger(config, os.Stderr)
	resp.Diagnostics.Append(common.LoggerToFrameworkDiagnostics(err)...)
	if resp.Diagnostics.HasError() {
		return
	}

	k.Client, diags = common.NewClient(host)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var authErr error
	k.Auth, authErr = common.NewAuth(token, tokenPath, "1.0+")
	resp.Diagnostics.Append(common.ToFrameworkDiagnostics(authErr)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.DataSourceData = &k
	resp.ResourceData = &k
}

func (p *metakubeProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		datasource_k8s_version.NewK8sClusterVersionDataSource,
		datasource_sshkey.NewSSHKeyDataSource,
		datasource_project.NewProjectDataSource,
	}
}

func (p *metakubeProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "metakube"
}

func (p *metakubeProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		resource_cluster.NewClusterResource,
		resource_cluster_role_binding.NewClusterRoleBinding,
		resource_role_binding.NewRoleBinding,
		resource_node_deployment.NewNodeDeployment,
		resource_sshkey.NewSSHKey,
		resource_maintenance_cronjob.NewMaintenanceCronJob,
	}
}
