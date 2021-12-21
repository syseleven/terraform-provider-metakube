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
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
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
func Provider() *schema.Provider {
	p := &schema.Provider{
		Schema: map[string]*schema.Schema{
			"host": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("METAKUBE_HOST", "https://metakube.syseleven.de"),
				Description: "The hostname of MetaKube API (in form of URI)",
			},
			"token": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("METAKUBE_TOKEN", ""),
				Description: "Authentication token",
			},
			"token_path": {
				Type:     schema.TypeString,
				Optional: true,
				DefaultFunc: schema.MultiEnvDefaultFunc(
					[]string{
						"METAKUBE_TOKEN_PATH",
					},
					"~/.metakube/auth"),
				Description: "Path to the MetaKube authentication token, defaults to ~/.metakube/auth",
			},
			"development": {
				Type:        schema.TypeBool,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("METAKUBE_DEV", false),
				Description: "Run development mode.",
			},
			"debug": {
				Type:        schema.TypeBool,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("METAKUBE_DEBUG", false),
				Description: "Run debug mode.",
			},
			"log_path": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("METAKUBE_LOG_PATH", ""),
				Description: "Path to store logs",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"metakube_cluster":              metakubeResourceCluster(),
			"metakube_cluster_role_binding": metakubeResourceClusterRoleBinding(),
			"metakube_node_deployment":      metakubeResourceNodeDeployment(),
			"metakube_sshkey":               metakubeResourceSSHKey(),
		},

		DataSourcesMap: map[string]*schema.Resource{
			"metakube_k8s_version": dataSourceMetakubeK8sClusterVersion(),
		},
	}

	// copying stderr because of https://github.com/hashicorp/go-plugin/issues/93
	// as an example the standard log pkg points to the "old" stderr
	stderr := os.Stderr

	p.ConfigureContextFunc = func(_ context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
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

func configure(d *schema.ResourceData, terraformVersion string, fd *os.File) (interface{}, diag.Diagnostics) {
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

func newLogger(d *schema.ResourceData, fd *os.File) (*zap.SugaredLogger, diag.Diagnostics) {
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
