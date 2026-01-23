package main

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6/tf6server"
	"github.com/hashicorp/terraform-plugin-mux/tf5to6server"
	"github.com/hashicorp/terraform-plugin-mux/tf6muxserver"
	"github.com/syseleven/terraform-provider-metakube/metakube"
)

func main() {
	ctx := context.Background()

	upgradedSdkProvider, err := tf5to6server.UpgradeServer(
		ctx,
		func() tfprotov5.ProviderServer {
			return metakube.Provider().GRPCProvider()
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	providers := []func() tfprotov6.ProviderServer{
		providerserver.NewProtocol6(metakube.NewFrameworkProvider()),
		func() tfprotov6.ProviderServer {
			return upgradedSdkProvider
		},
	}

	muxServer, err := tf6muxserver.NewMuxServer(ctx, providers...)
	if err != nil {
		log.Fatal(err)
	}

	var serveOpts []tf6server.ServeOpt

	err = tf6server.Serve("registry.terraform.io/syseleven/metakube", muxServer.ProviderServer, serveOpts...)
	if err != nil {
		log.Fatal(err)
	}
}
