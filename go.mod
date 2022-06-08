module github.com/syseleven/terraform-provider-metakube

go 1.15

replace (
	github.com/hashicorp/terraform-plugin-sdk/v2 => github.com/syseleven/terraform-plugin-sdk/v2 v2.17.0-sys11-1
	k8s.io/api => k8s.io/api v0.0.0-20190918195907-bd6ac527cfd2
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190817020851-f2f3a405f61d
	k8s.io/client-go => k8s.io/client-go v0.0.0-20190918200256-06eb1244587a
)

require (
	github.com/go-openapi/runtime v0.19.30
	github.com/go-openapi/strfmt v0.20.2
	github.com/google/go-cmp v0.5.8
	github.com/hashicorp/go-cty v1.4.1-0.20200723130312-85980079f637
	github.com/hashicorp/go-version v1.5.0
	github.com/hashicorp/terraform-plugin-sdk/v2 v2.17.0
	github.com/mitchellh/go-homedir v1.1.0
	github.com/syseleven/go-metakube v0.0.0-20211112103958-da9c0d9da26c
	go.uber.org/zap v1.19.0
	golang.org/x/mod v0.5.0
)
