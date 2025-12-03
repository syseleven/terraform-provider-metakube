package provider_testutil

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

var (
	TestAccProviders map[string]*schema.Provider
	TestAccProvider  *schema.Provider
)
