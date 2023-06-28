terraform {
  required_providers {
    metakube = {
      source = "syseleven/metakube"
    }
    openstack = {
      source = "terraform-provider-openstack/openstack"
    }
  }
}

resource "metakube_maintenance_cron_job" "acctest" {
  project_id = "h9b8xjcqfd"
  cluster_id = "ztdhg6zw87"
  maintenance_cron_job_name = "maintenance_cron_job_name"
  timeouts {
    create = "15m"
    update = "15m"
    delete = "15m"
  }
  spec {
    failed_jobs_history_limit 		= 1
    starting_deadline_seconds 		= 1
    successful_jobs_history_limit 	= 1
    schedule						= "25 * * * *"
    maintenance_job_template {
      name 	= "maintenance_job_template_name"
      spec {
        rollback 	= false
        type		= "updateKubernetesVersion"
      }
    }
  }
}