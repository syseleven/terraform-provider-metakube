package main

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/syseleven/terraform-provider-metakube/metakube"
)

func main() {
	err := providerserver.Serve(context.Background(), metakube.NewFrameworkProvider, providerserver.ServeOpts{
		Address: "registry.terraform.io/syseleven/metakube",
	})
	if err != nil {
		log.Fatal(err)
	}
}
