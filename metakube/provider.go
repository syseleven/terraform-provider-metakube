package metakube

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"time"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	fwdiag "github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	frameworkSchema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	pluginSchema "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mitchellh/go-homedir"
	k8client "github.com/syseleven/go-metakube/client"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	// wait this time before starting resource checks
	requestDelay = time.Second
)

type metakubeProviderMeta struct {
	client *k8client.MetaKubeAPI
	auth   runtime.ClientAuthInfoWriter
	log    *zap.SugaredLogger
}

// Provider returns a schema.Provider for MetaKube.
func Provider() *pluginSchema.Provider {
	p := &pluginSchema.Provider{
		Schema: map[string]*pluginSchema.Schema{
			"host": {
				Type:        pluginSchema.TypeString,
				Description: "The hostname of MetaKube API (in form of URI)",
				Optional:    true,
				DefaultFunc: pluginSchema.EnvDefaultFunc("METAKUBE_HOST", "https://metakube.syseleven.de"),
			},
			"token": {
				Type:        pluginSchema.TypeString,
				Description: "The MetaKube authentication token",
				Optional:    true,
				DefaultFunc: pluginSchema.EnvDefaultFunc("METAKUBE_TOKEN", ""),
			},
			"token_path": {
				Type:        pluginSchema.TypeString,
				Description: "Path to the MetaKube authentication token, defaults to ~/.metakube/auth",
				Optional:    true,
				DefaultFunc: pluginSchema.MultiEnvDefaultFunc(
					[]string{
						"METAKUBE_TOKEN_PATH",
					}, "~/.metakube/auth"),
			},
			"development": {
				Type:        pluginSchema.TypeBool,
				Description: "Run development mode.",
				Optional:    true,
				Default:     false,
			},
			"debug": {
				Type:        pluginSchema.TypeBool,
				Description: "Run debug mode.",
				Optional:    true,
				Default:     false,
			},
			"log_path": {
				Type:        pluginSchema.TypeString,
				Description: "Path to store logs",
				Optional:    true,
				Default:     "",
			},
		},

		ResourcesMap: map[string]*pluginSchema.Resource{
			"metakube_cluster":              metakubeResourceCluster(),
			"metakube_cluster_role_binding": metakubeResourceClusterRoleBinding(),
			"metakube_role_binding":         metakubeResourceRoleBinding(),
			"metakube_node_deployment":      metakubeResourceNodeDeployment(),
			"metakube_sshkey":               metakubeResourceSSHKey(),
			"metakube_maintenance_cron_job": metakubeResourceMaintenanceCronJob(),
		},

		DataSourcesMap: map[string]*pluginSchema.Resource{
			"metakube_k8s_version": dataSourceMetakubeK8sClusterVersion(),
			"metakube_sshkey":      dataSourceMetakubeSSHKey(),
			"metakube_project":     dataSourceMetakubeProject(),
		},
	}

	// copying stderr because of https://github.com/hashicorp/go-plugin/issues/93
	// as an example the standard log pkg points to the "old" stderr
	stderr := os.Stderr

	p.ConfigureContextFunc = func(_ context.Context, d *pluginSchema.ResourceData) (interface{}, diag.Diagnostics) {
		terraformVersion := p.TerraformVersion
		if terraformVersion == "" {
			// Terraform 0.12 introduced this field to the protocol
			// We can therefore assume that if it's missing it's 0.10 or 0.11
			terraformVersion = "0.11+compatible"
		}
		return configure(d, terraformVersion, stderr)
	}

	return p
}

func configure(d *pluginSchema.ResourceData, terraformVersion string, fd *os.File) (interface{}, diag.Diagnostics) {
	var (
		k                metakubeProviderMeta
		diagnostics, tmp diag.Diagnostics
	)

	k.log, tmp = newLogger(d, fd)
	diagnostics = append(diagnostics, tmp...)
	k.client, tmp = newClient(d.Get("host").(string))
	diagnostics = append(diagnostics, tmp...)

	k.auth, tmp = newAuth(d.Get("token").(string), d.Get("token_path").(string), terraformVersion)
	diagnostics = append(diagnostics, tmp...)

	return &k, diagnostics
}

func newLogger(d *pluginSchema.ResourceData, fd *os.File) (*zap.SugaredLogger, diag.Diagnostics) {
	var (
		ec    zapcore.EncoderConfig
		cores []zapcore.Core
		level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	)

	logDev := d.Get("development").(bool)
	logDebug := d.Get("debug").(bool)
	logPath := d.Get("log_path").(string)

	if logDev || logDebug {
		level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	}

	if logDev {
		ec = zap.NewDevelopmentEncoderConfig()
		ec.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		ec = zap.NewProductionEncoderConfig()
		ec.EncodeLevel = func(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString("[" + level.CapitalString() + "]")
		}
	}
	ec.EncodeTime = zapcore.ISO8601TimeEncoder
	ec.EncodeDuration = zapcore.StringDurationEncoder

	if logPath != "" {
		jsonEC := ec
		jsonEC.EncodeLevel = zapcore.LowercaseLevelEncoder
		sink, _, err := zap.Open(logPath)
		if err != nil {
			return nil, diag.Diagnostics{{
				Severity:      diag.Error,
				Summary:       fmt.Sprintf("Can't access location: %v", err),
				AttributePath: cty.Path{cty.GetAttrStep{Name: "log_path"}},
			}}
		}
		cores = append(cores, zapcore.NewCore(zapcore.NewJSONEncoder(jsonEC), sink, level))
	}

	cores = append(cores, zapcore.NewCore(zapcore.NewConsoleEncoder(ec), zapcore.AddSync(fd), level))
	core := zapcore.NewTee(cores...)
	return zap.New(core).Sugar(), nil
}

func newClient(host string) (*k8client.MetaKubeAPI, diag.Diagnostics) {
	u, err := url.Parse(host)
	if err != nil {
		return nil, diag.Diagnostics{{
			Severity:      diag.Error,
			Summary:       fmt.Sprintf("Can't parse host: %v", err),
			AttributePath: cty.Path{cty.GetAttrStep{Name: "host"}},
		}}
	}

	return k8client.NewHTTPClientWithConfig(nil, &k8client.TransportConfig{
		Host:     u.Host,
		BasePath: u.Path,
		Schemes:  []string{u.Scheme},
	}), nil
}

func newAuth(token, tokenPath, terraformVersion string) (runtime.ClientAuthInfoWriter, diag.Diagnostics) {
	if token == "" && tokenPath != "" {
		p, err := homedir.Expand(tokenPath)
		if err != nil {
			return nil, diag.Diagnostics{{
				Severity:      diag.Error,
				Summary:       fmt.Sprintf("Can't parse path: %v", err),
				AttributePath: cty.Path{cty.GetAttrStep{Name: "token_path"}},
			}}
		}
		rawToken, err := ioutil.ReadFile(p)
		if err != nil {
			return nil, diag.Diagnostics{{
				Severity:      diag.Error,
				Summary:       fmt.Sprintf("Can't read token file: %v", err),
				AttributePath: cty.Path{cty.GetAttrStep{Name: "token_path"}},
			}}
		}
		token = string(bytes.Trim(rawToken, "\n"))
	} else if token == "" {
		return nil, diag.Diagnostics{{
			Severity:      diag.Error,
			Summary:       "Missing authorization token",
			AttributePath: cty.Path{cty.GetAttrStep{Name: "token_path"}, cty.GetAttrStep{Name: "token"}},
		}}
	}

	auth := runtime.ClientAuthInfoWriterFunc(func(r runtime.ClientRequest, _ strfmt.Registry) error {
		err := r.SetHeaderParam("Authorization", "Bearer "+token)
		if err != nil {
			return err
		}
		return r.SetHeaderParam("User-Agent", fmt.Sprintf("Terraform/%s", terraformVersion))
	})
	return auth, nil
}

// Terraform Plugin Framework Provider

type metakubeProviderConfig struct {
	host        types.String `tfsdk:"host"`
	token       types.String `tfsdk:"token"`
	token_path  types.String `tfsdk:"token_path"`
	development types.Bool   `tfsdk:"development"`
	debug       types.Bool   `tfsdk:"debug"`
	log_path    types.String `tfsdk:"log_path"`
}

var _ provider.Provider = (*metakubeProvider)(nil)

type metakubeProvider struct{}

func NewFrameworkProvider() provider.Provider {
	return &metakubeProvider{}
}

func newLoggerFramework(config metakubeProviderConfig, fd *os.File) (*zap.SugaredLogger, fwdiag.Diagnostics) {
	var (
		ec          zapcore.EncoderConfig
		cores       []zapcore.Core
		level       = zap.NewAtomicLevelAt(zapcore.InfoLevel)
		diagnostics fwdiag.Diagnostics
	)

	logDev := config.development.ValueBool()
	logDebug := config.debug.ValueBool()
	logPath := config.log_path.ValueString()

	if logDev || logDebug {
		level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	}

	if logDev {
		ec = zap.NewDevelopmentEncoderConfig()
		ec.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		ec = zap.NewProductionEncoderConfig()
		ec.EncodeLevel = func(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString("[" + level.CapitalString() + "]")
		}
	}
	ec.EncodeTime = zapcore.ISO8601TimeEncoder
	ec.EncodeDuration = zapcore.StringDurationEncoder

	if logPath != "" {
		jsonEC := ec
		jsonEC.EncodeLevel = zapcore.LowercaseLevelEncoder
		sink, _, err := zap.Open(logPath)
		if err != nil {
			diagnostics.AddAttributeError(
				path.Root("log_path"),
				"Cannot access log location",
				fmt.Sprintf("Can't access location: %v", err),
			)
			return nil, diagnostics
		}
		cores = append(cores, zapcore.NewCore(zapcore.NewJSONEncoder(jsonEC), sink, level))
	}

	cores = append(cores, zapcore.NewCore(zapcore.NewConsoleEncoder(ec), zapcore.AddSync(fd), level))
	core := zapcore.NewTee(cores...)
	return zap.New(core).Sugar(), diagnostics
}

func newClientFramework(host string) (*k8client.MetaKubeAPI, fwdiag.Diagnostics) {
	var diagnostics fwdiag.Diagnostics

	u, err := url.Parse(host)
	if err != nil {
		diagnostics.AddAttributeError(
			path.Root("host"),
			"Cannot parse host",
			fmt.Sprintf("Can't parse host: %v", err),
		)
		return nil, diagnostics
	}

	return k8client.NewHTTPClientWithConfig(nil, &k8client.TransportConfig{
		Host:     u.Host,
		BasePath: u.Path,
		Schemes:  []string{u.Scheme},
	}), diagnostics
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
	var config metakubeProviderConfig

	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check for unknown values
	if config.host.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("host"),
			"Unknown MetaKube host",
			"The MetaKube host is unknown",
		)
	}

	if config.token.IsUnknown() || config.token_path.IsUnknown() {
		if config.token.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root("token"),
				"Unknown MetaKube token",
				"The MetaKube token is unknown",
			)
		}
		if config.token_path.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root("token_path"),
				"Unknown MetaKube token path",
				"The MetaKube token path is unknown",
			)
		}
	}

	if config.development.IsUnknown() {
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
	host := config.host.ValueString()
	if host == "" {
		host = os.Getenv("METAKUBE_HOST")
		if host == "" {
			host = "https://metakube.syseleven.de"
		}
	}

	token := config.token.ValueString()
	if token == "" {
		token = os.Getenv("METAKUBE_TOKEN")
	}

	tokenPath := config.token_path.ValueString()
	if tokenPath == "" {
		tokenPath = os.Getenv("METAKUBE_TOKEN_PATH")
		if tokenPath == "" {
			tokenPath = "~/.metakube/auth"
		}
	}

	var k metakubeProviderMeta

	k.log, diags = newLoggerFramework(config, os.Stderr)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	k.client, diags = newClientFramework(host)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.DataSourceData = k.client
	resp.ResourceData = k.client
}

func (p *metakubeProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		// dataSourceMetakubeK8sClusterVersion,
		// dataSourceMetakubeSSHKey,
		// dataSourceMetakubeProject,
	}
}

func (p *metakubeProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "metakube"
}

func (p *metakubeProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		// metakubeResourceCluster,
		// metakubeResourceClusterRoleBinding,
		// metakubeResourceRoleBinding,
		// metakubeResourceNodeDeployment,
		// metakubeResourceSSHKey,
		// metakubeResourceMaintenanceCronJob,
	}
}
