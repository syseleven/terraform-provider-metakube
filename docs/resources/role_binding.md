# role_binding Resource

Role binding resource gives a quick and easy way to manage user and group namespaced role bindings. This is useful for clusters using [SysEleven Login](https://docs.syseleven.de/metakube/en/tutorials/external-authentication).

Each resource manages only its configured subjects. Other users or groups can have the same role in the same namespace. The provider does not add those subjects to this resource's state.

During refresh, the provider checks all bindings for the specified namespace and role. It preserves the subject order in state and detects missing managed subjects.

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

## Import

The import identifier is `project_id:cluster_id:namespace:role_name`:

```shell
terraform import metakube_role_binding.example 'project-id:cluster-id:kube-system:kube-admin'
```

During import, the provider selects the first binding with supported subjects for the specified namespace and role. It adds that binding's users and groups to Terraform state. The resource does not support subjects with kind `ServiceAccount`.

An import cannot select individual subjects. Import only if this resource should manage all imported subjects. Terraform can unbind these subjects when it replaces or destroys the resource.

The API selects bindings by namespace and role. This resource cannot independently manage multiple named Kubernetes bindings for the same namespace and role.

## Recover state after an incorrect refresh

Earlier provider versions could record unrelated subjects in Terraform state during refresh. Use a provider build with the subject matching fix before you recover state.

If state contains unrelated subjects, follow the [state recovery procedure](cluster_role_binding.md#recover-state-after-an-incorrect-refresh). Replace the resource address in those commands with `metakube_role_binding.example`.
