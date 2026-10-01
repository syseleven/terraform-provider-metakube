# Migrate resource blocks to nested attributes

This is a breaking configuration change on `master`. Update configurations and reusable modules when adopting a provider release containing this migration.

The following fields now use attributes:

| Resource | Field | New syntax |
| --- | --- | --- |
| `metakube_role_binding` | `subject` | List of objects: `subject = [{ ... }]` |
| `metakube_cluster_role_binding` | `subject` | List of objects: `subject = [{ ... }]` |
| `metakube_maintenance_cron_job` | `spec` | Object: `spec = { ... }` |
| `metakube_maintenance_cron_job` | `spec.maintenance_job_template` | Object: `maintenance_job_template = { ... }` |
| `metakube_maintenance_cron_job` | `spec.maintenance_job_template.options` | Optional object: `options = { ... }` |

`timeouts { ... }` remains a block. Resource names and import identifiers do not change.

## Binding subjects

Before, subjects were repeated blocks:

```hcl
subject {
  kind = "user"
  name = "user@example.com"
}
subject {
  kind = "group"
  name = "operators"
}
```

After, subjects are entries in one list:

```hcl
subject = [
  {
    kind = "user"
    name = "user@example.com"
  },
  {
    kind = "group"
    name = "operators"
  },
]
```

Keep the original subject order. At least one subject is required. Changing subjects still requires resource replacement. Replace `dynamic "subject"` blocks with a list expression, such as `subject = [for subject in var.subjects : { kind = subject.kind, name = subject.name }]`.

## Maintenance cron jobs

Before:

```hcl
spec {
  schedule = "5 4 * * *"
  maintenance_job_template {
    type = "kubernetesPatchUpdate"
    options {
      options = {
        version = "1.32.1"
      }
    }
  }
}
```

After:

```hcl
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
```

The inner `options` field remains a map of strings. Omit the outer `options` object, or set it to `null`, when no options are needed. `spec` and `maintenance_job_template` are required. `rollback` remains API-controlled and must not be set in configuration.

Remove singleton list indices from references. For example, change `job.spec[0].maintenance_job_template[0].type` to `job.spec.maintenance_job_template.type`. Subject references retain their list indices.

## Existing state

The provider automatically upgrades maintenance cron job state from schema version 0 to version 1. It converts the three singleton lists into objects and an absent options block into `null`. Resource IDs, timestamps, option values, rollback, and timeouts are preserved. Binding subjects retain their list representation and need no state transformation.

Back up state before upgrading. Update the provider version and configuration together, then review a Terraform plan before applying. With equivalent configuration and no remote drift, this syntax migration should not cause an infrastructure change or resource replacement. Manual state editing, removal, and re-import are not required for this migration.
