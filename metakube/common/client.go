package common

import (
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	k8client "github.com/syseleven/go-metakube/client"
)

func NewClient(host string) (*k8client.MetaKubeAPI, diag.Diagnostics) {
	var diagnostics diag.Diagnostics

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
