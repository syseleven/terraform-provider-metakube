# metaKube Provider

The MetaKube provider is used to interact with the resources supported by MetaKube.
The provider needs to be configured with the proper auth token (`~/.metakube/auth` or `METAKUBE_TOKEN` env var) for an existing project before it can be used. You can create a project and an API Account with a token via UI.

Use the navigation to the left to read about the available resources.

## Example Usage

```hcl
provider "metakube" {
  host = "https://metakube-api-address"
}

provider "metakube" {
  host     = "https://metakube.syseleven.de"
}

data "metakube_k8s_version" "cluster" {
  major = "1"
  minor = "21"
}

resource "metakube_cluster" "cluster" {
  name       = "cluster-via-terraform"
  dc_name    = "syseleven-cbk1"
  project_id = "YOUR_PROJECT_ID"
  labels = { 
    "foo" = "bar"
  }

  spec {
    enable_ssh_agent = true
    version          = data.metakube_k8s_version.cluster.version
    cloud {
      openstack {
        application_credentials {
			id     = "YOUR_CREDENTIAL_ID"
        	secret  = "YOU_CREDENTIAL_SECRET"
		}
      }
    }
  }
}
```

## Authentication

The provider tries to read a token from `~/.metakube/auth` by default,
it is possible to change the token location by setting `token_path` argument.
Another way of authentication is to pass `METAKUBE_TOKEN` env or set `token` param,
the last option is not recommended due to possible secret leaking.

You have to have a Project and API Account with a token created via UI before using provider.

## Argument Reference

The following arguments are supported:

* `host` - (Optional) The hostname (in form of URI) of MetaKube API. Can be sourced from `METAKUBE_HOST`.
* `token` - (Optional) Authentication token. Can be sourced from `METAKUBE_TOKEN`.
* `token_path` - (Optional) Path to the metakube token. Defaults to `~/.metakube/auth`. Can be sourced from `METAKUBE_TOKEN_PATH`.
* `log_path` - (Optional) Location to store provider logs. Can be sourced from `METAKUBE_LOG_PATH`
* `debug` - (Optional) Set logger to debug level. Can be sourced from `METAKUBE_DEBUG`.
* `development` - (Optional) Run development mode. Useful only for contributors. Can be sourced from `METAKUBE_DEV`.
