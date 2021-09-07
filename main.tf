terraform {
  required_providers {
    metakube = {
      source = "syseleven.de/syseleven/metakube"
    }
    openstack = {
      source = "terraform-provider-openstack/openstack"
    }
  }
}

provider "metakube" {
  host = "https://stage.metakube.de"
}
resource "metakube_project" "project" {
  name = "tf-project"
}

data "metakube_k8s_version" "cluster" {
  major = "1"
  minor = "21"
}

resource "metakube_cluster" "cluster" {
  name       = "cls"
  dc_name    = "syseleven-cbk1"
  project_id = metakube_project.project.id

  spec {
    enable_ssh_agent = true
    version          = data.metakube_k8s_version.cluster.version
    cloud {
      openstack {
        application_credentials_id = "2f96ca8fae55401598d2ae47dbbf74bb"
        application_credentials_secret = "A5P-qtUzBba8QL28XrBeM8NAuWxSDwlerNVzcBM0sSZqLSMqiu1j9KlZar_HYsmHudacilBQKGRgftASjFf80w"
      }
    }
  }
}
