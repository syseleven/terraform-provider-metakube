# cluster_role_binding Resource

Cluster role binding resource gives a quick and easy way to manage user and group cluster role bindings. This is useful for clusters using [SysEleven Login](https://docs.syseleven.de/metakube/en/tutorials/external-authentication).

## Example Usage

```hcl
resource "metakube_cluster_role_binding" "example" {
  project_id = "project id"
  cluster_id = "cluster id"
  
  cluster_role_name = "kube-admin"
  
  subject {
    kind = "user"
    name = "foo@example.com"
  }
  
  subject {
    kind = "group"
    name = "SRE-team"
  }
}
```

## Argument Reference

The following arguments are supported:

* `project_id` - (Required) Reference project identifier.
* `cluster_id` - (Required) Cluster ID.
* `subject` - (Required) List of users and groups to bind cluster role to. At least one subject must be specified.

## Nested Blocks

### `subject`

#### Arguments

* `kind` - (Required) Either 'group' or 'user'.
* `name` - (Optional) Name of the group or user's email.
