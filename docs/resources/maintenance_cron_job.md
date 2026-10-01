# maintenance_cron_job Resource

Manage scheduled maintenance for a MetaKube cluster.

## Example Usage

```hcl
resource "metakube_maintenance_cron_job" "example" {
  project_id = "project-id"
  cluster_id = "cluster-id"
  name       = "scheduled-patch-update"

  spec = {
    schedule = "5 4 * * *"
    maintenance_job_template = {
      type = "kubernetesPatchUpdate"
      options = {
        options = {
          version = "1.32.1"
        }
      }
    }
  }

  timeouts {
    create = "20m"
    update = "20m"
    delete = "20m"
  }
}
```

See the [nested attribute migration guide](../guides/nested-attributes-migration.md) for existing configurations.

## Arguments

- `cluster_id` (Required): Cluster that owns the maintenance cron job. Changes require replacement.
- `project_id` (Optional): Owning project. The provider discovers it from the cluster when omitted.
- `name` (Optional): Maintenance cron job name. Changes require replacement.
- `spec` (Required): Object containing `schedule` and `maintenance_job_template`.
- `spec.schedule` (Required): Cron schedule.
- `spec.maintenance_job_template.type` (Required): Maintenance job type.
- `spec.maintenance_job_template.options` (Optional): Object containing an optional `options` map of strings. Available map keys depend on the maintenance type. Changing option values causes the provider to delete and recreate the cron job.

`spec.maintenance_job_template.rollback` is controlled by the API. Leave it unset; it is planned as `false` on creation and follows API state afterward.

## Read-only attributes

- `id`: Maintenance cron job identifier.
- `creation_timestamp`: Creation timestamp from the API.
- `deletion_timestamp`: Deletion timestamp from the API.

## Timeouts

The `timeouts` block supports `create`, `update`, and `delete`, each defaulting to `20m`.

## Import

```shell
terraform import metakube_maintenance_cron_job.example 'project-id:cluster-id:maintenance-cron-job-id'
```
