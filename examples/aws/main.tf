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
  minor = var.k8s_minor_version
}
resource "metakube_cluster" "example_cluster" {
  name       = var.cluster_name
  dc_name    = "syseleven-aws-eu-central-1a"
  project_id = var.project_id
  spec {
    enable_ssh_agent = true
    version          = data.metakube_k8s_version.cluster.version
    cloud {
      aws {
        access_key_id     = var.aws_access_key_id
        secret_access_key = var.aws_secret_access_key
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
        aws {
          instance_type     = "t3.small"
          disk_size         = 25
          volume_type       = "standard"
          availability_zone = "eu-central-1c"
          subnet_id         = var.aws_subnet_id
          assign_public_ip  = true

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
