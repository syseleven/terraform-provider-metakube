# role_binding Resource

Role binding resource gives a quick and easy way to manage user and group namespaced role bindings. This is useful for clusters using [SysEleven Login](https://docs.syseleven.de/metakube/en/tutorials/external-authentication).

## Example Usage

```hcl
resource "metakube_role_binding" "example" {
  project_id = "project id"
  cluster_id = "cluster id"
  
  role_name = "kube-admin"
  namespace = "kube-system"
  
  subject {
    kind = "user"
    name = "foo@example.com"
  }
  
  subject {
    kind = "group"
    name = "SRE-team"
  }

  timeouts {
    create = "5m"
  }
}
```

## Argument Reference

The following arguments are supported:

* `project_id` - (Required) Reference project identifier.
* `cluster_id` - (Required) Cluster ID.
* `namespace` - (Required) The namespace to create binding for.
* `role_name` - (Required) The name of the role in the namespace to bind to.
* `subject` - (Required) List of users and groups to bind cluster role to. At least one subject must be specified.

## Nested Blocks

### `subject`

#### Arguments

* `kind` - (Required) Either 'group' or 'user'.
* `name` - (Optional) Name of the group or user's email.

### `timeouts`

#### Arguments

* `create` - (Optional) Timeout for creating bindings. Defaults to `20m`. Applies per subject.
