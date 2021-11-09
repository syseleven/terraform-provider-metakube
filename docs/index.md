# metaKube Provider

The MetaKube provider is used to interact with the resources supported by MetaKube.
The provider needs to be configured with the proper auth token (`~/.metakube/auth` or `METAKUBE_TOKEN` env var) for an existing project before it can be used. You can create a project and an API Account with a token via UI.

Use the navigation to the left to read about the available resources.

## Example Usage

```hcl
provider "metakube" {
  host = "https://metakube-api-address"
}

# Example project configuration
resource "metakube_project" "project" {
  name = "terraform-project"
}
```

## Authentication

The provider tries to read a token from `~/.metakube/auth` by default,
it is possible to change the token location by setting `token_path` argument.
Another way of authentication is to pass `METAKUBE_TOKEN` env or set `token` param,
the last option is not recommended due to possible secret leaking.

At the moment global access tokens are not supported by MetaKube API. We **can't manage resources outside their project** using tokens. We suggest to use UI to create project and API Account/Token that is capable to manage resources **inside the project**.

## Argument Reference

The following arguments are supported:

* `host` - (Optional) The hostname (in form of URI) of MetaKube API. Can be sourced from `METAKUBE_HOST`.
* `token` - (Optional) Authentication token. Can be sourced from `METAKUBE_TOKEN`.
* `token_path` - (Optional) Path to the metakube token. Defaults to `~/.metakube/auth`. Can be sourced from `METAKUBE_TOKEN_PATH`.
* `log_path` - (Optional) Location to store provider logs. Can be sourced from `METAKUBE_LOG_PATH`
* `debug` - (Optional) Set logger to debug level. Can be sourced from `METAKUBE_DEBUG`.
* `development` - (Optional) Run development mode. Useful only for contributors. Can be sourced from `METAKUBE_DEV`.
