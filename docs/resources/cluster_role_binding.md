# cluster_role_binding Resource

Cluster role binding resource gives a quick and easy way to manage user and group cluster role bindings. This is useful for clusters using [SysEleven Login](https://docs.syseleven.de/metakube/en/tutorials/external-authentication).

Each resource manages only its configured subjects. Other users or groups can have the same cluster role. The provider does not add those subjects to this resource's state.

During refresh, the provider checks all bindings for the role. The provider preserves the configured subject order and detects missing managed subjects.

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

  timeouts {
    create = "5m"
  }
}
```

For an OIDC group, use the exact group name that Kubernetes receives. Include any configured group prefix.

```hcl
resource "metakube_cluster_role_binding" "role_binding_name" {
  project_id        = "project id"
  cluster_id        = "cluster id"
  cluster_role_name = "cluster-admin"

  subject {
    kind = "group"
    name = "role_binding_name"
  }
}
```

You do not need to include existing administrators in this resource.

## Argument Reference

The following arguments are supported:

* `project_id` - (Required) Reference project identifier.
* `cluster_id` - (Required) Cluster ID.
* `cluster_role_name` - (Required) The name of the cluster role to bind to.
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

The import identifier is `project_id:cluster_id:cluster_role_name`:

```shell
terraform import metakube_cluster_role_binding.example 'project-id:cluster-id:cluster-admin'
```

This identifier does not select individual subjects. During an import, the provider selects the first nonempty binding that the API returns for the specified role. The provider adds that binding's user and group subjects to Terraform state.

Import only if this resource should manage all imported subjects. Terraform can unbind these subjects when it replaces or destroys the resource.

The resource does not support subjects with kind `ServiceAccount`. The API selects bindings by role. The resource cannot independently manage multiple named Kubernetes bindings for the same role.

## Recover state after an incorrect refresh

Earlier provider versions read subjects from the first binding for the role during refresh. This could record another administrator in Terraform state instead of the configured OIDC group.

Do not apply a replacement that uses the incorrect state. Terraform uses the subjects in state when it removes access.

Use the following procedure only if `terraform state show metakube_cluster_role_binding.role_binding_name` shows an unrelated subject:

1. Create a backup of Terraform state.

   ```shell
   terraform state pull > terraform-state-backup.json
   ```

2. Remove only the affected resource from Terraform state.

   ```shell
   terraform state rm metakube_cluster_role_binding.role_binding_name
   ```

   `terraform state rm` does not change the remote access bindings.

3. Create a Terraform plan.

   ```shell
   terraform plan
   ```

4. Review the Terraform plan.

5. Apply the configuration to record the configured subjects again.

During creation, the provider accepts the API's "already connected" response for an existing binding.
