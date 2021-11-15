terraform {
  required_providers {
    metakube = {
      source = "syseleven/metakube"
    }
  }
}
provider "metakube" {
}
data "metakube_k8s_version" "cluster" {
  major = "1"
  minor = "21"
}
resource "metakube_cluster" "example_cluster" {
  name       = var.cluster_name
  dc_name    = "syseleven-azure-centralus"
  project_id = var.project_id
  spec {
    enable_ssh_agent = true
    version          = data.metakube_k8s_version.cluster.version
    cloud {
      azure {
        openstack_billing_tenant = var.openstack_billing_tenant
        client_id                = var.azure_client_id
        subscription_id          = var.azure_subscription_id
        tenant_id                = var.azure_tenant_id
        client_secret            = var.azure_client_secret
      }
    }
  }
}
resource "metakube_node_deployment" "example_node" {
  name       = "examplenode"
  cluster_id = metakube_cluster.example_cluster.id
  spec {
    replicas = 2
    template {
      cloud {
        azure {
          size = "Standard_F2"
        }
      }
      operating_system {
        ubuntu {
          dist_upgrade_on_boot = false
        }
      }
      versions {
        kubelet = data.metakube_k8s_version.cluster.version
      }
    }
  }
}
