# cluster Resource

Cluster resource in the provider defines the corresponding cluster in MetaKube.

## Example Usage

```hcl
resource "metakube_cluster" "example" {
  project_id = "example-project-id"
  name = "example"
  dc_name = "europe-west3-c"

  spec {
    version = "1.18.8"
    cloud {

      aws {
        instance_profile_name = "example-profile-name"
      }
    }
  }
}

# create admin.conf file
resource "local_file" "kubeconfig" {
  content     = metakube_cluster.cluster.kube_config
  filename = "${path.module}/admin.conf"
}
```

## Argument Reference

The following arguments are supported:

* `project_id` - (Required) Reference project identifier.
* `dc_name` - (Required) Data center name. To list of available options you can run the following command: `curl -s -H "authorization: Bearer $METAKUBE_TOKEN" https://metakube.syseleven.de/api/v1/dc | jq -r '.[] | select(.seed!=true) | .metadata.name'`
* `name` - (Required) Cluster name.
* `spec` - (Required) Cluster specification.
* `labels` - (Optional) Labels added to cluster.
* `sshkeys` - (Optional) IDs of SSH keys to be attached to nodes. Ideally you want to use this along with [metakube_sshkey](./sshkey.md).

### Timeouts

`metakube_cluster` provides the following Timeouts configuration options:
  * create - (Default 20 minutes) Used for Creating cluster control plane, etcd, api server etc.
  * update - (Default 20 minutes) Used for cluster modifications.
  * delete - (Default 20 minutes) Used for destroying clusters.

## Attributes

* `id` - Cluster identifier.
* `kube_config` - Admin kube config raw content which can be dumped to a file using [local_file](https://registry.terraform.io/providers/hashicorp/local/latest/docs/resources/file). You might want to use `oidc_kube_config` or `kube_login_kube_config` together with `syseleven_auth` configured for better security.
* `oidc_kube_config` - Plain Open ID Connect kube config raw content which can be dumped to a file using [local_file](https://registry.terraform.io/providers/hashicorp/local/latest/docs/resources/file). To use `syseleven_auth` should be configured too.
* `kube_login_kube_config` - The `kubelogin` config content which can be dumped to a file using [local_file](https://registry.terraform.io/providers/hashicorp/local/latest/docs/resources/file). To use `syseleven_auth` should be configured too.
* `creation_timestamp` - Timestamp of resource creation.
* `deletion_timestamp` - Timestamp of resource deletion.

## Nested Blocks

### `spec`

#### Arguments

* `version` - (Required) Cloud orchestrator version. You can use [metakube_k8s_version](../data-sources/k8s_version.md) to query available versions.
* `enable_ssh_agent` - (Optional) User SSH Agent runs on each node and manages ssh keys. You can disable it if you prefer to manage ssh keys manually.
* `cloud` - (Required) Cloud provider specification.
* `update_window` - (Optional) Node reboot window. Currently used only for Flatcar node deployments.
* `machine_networks` - (Optional) Machine networks, optionally specifies the parameters for IPAM.
* `audit_logging` - (Optional) Audit logging settings.
* `pod_security_policy` - (Optional) Pod security policies allow detailed authorization of pod creation and updates.
* `pod_node_selector` - (Optional) Configure PodNodeSelector admission plugin at the apiserver
* `syseleven_auth` - (Optional) Useful for authenticating against [SysEleven Login](https://docs.syseleven.de/metakube/en/tutorials/external-authentication).
* `services_cidr` - (Optional) Internal IP range for ClusterIP Services.
* `pods_cidr` - (Optional) Internal IP range for Pods.

### `cloud`

One of the following must be selected.

#### Arguments

* `openstack` - (Optional) Opestack infrastructure.
* `aws` - (Optional) Amazon Web Services infrastructure.


### `update_window`

When set, start time and length must be configured.

#### Arguments
* `start` - (Required) Node reboot window start time. Example: `Thu 02:35`.
* `length` - (Required) Node reboot window duration. Example: `1h30m`

### `openstack`

#### Arguments
* `floating_ip_pool` - (Required) The floating ip pool used by all worker nodes to receive a public ip.
* `security_group` - (Optional) When specified, all worker nodes will be attached to this security group. If not specified, a security group will be created.
* `network` - (Optional) When specified, all worker nodes will be attached to this network. If not specified, a network, subnet & router will be created.
* `subnet_id` - (Optional) When specified, all worker nodes will be attached to this subnet of specified network. If not specified, a network, subnet & router will be created.
* `subnet_cidr` - (Optional) Change this to configure a different internal IP range for Nodes. Default: `192.168.1.0/24`.
When using password based auth
* `server_group_id` - (Optional) Server group id to use for all machines within a cluster. You can use openstack server groups to group or seperate servers using soft/hard affinity/anti-affinity rules. When not set explicitly, the default soft anti-affinity server group will be created and used. 
* `user_credentials` - (Conditional) connect to Openstack using user credentials. May be omitted if `application_credentials` being used.
* `application_credentials` - (Conditional) connect to Openstack using Application Credentials. May be omitted if `user_credentials` being used.

### `user_credentials`

Openstack user credentials.

#### Arguments
* `project_id` - (Required) The id of project to use for billing. You can set it using environment variable `OS_PROJECT_ID`.
* `project_name` - (Required) The name of project to use for billing. You can set it using environment variable `OS_PROJECT_NAME`.
* `username` - (Required) The account's username. You can set it using environment variable `OS_USERNAME`.
* `password` - (Required) The account's password. You can set it using environment variable `OS_PASSWORD`.

### `application_credentials`

Openstack Application Credentials.

#### Arguments
* `id` - (Required) Application Credentials id to use.
* `secret` - (Required) Application Credentials secret to use.

### `aws`

#### Arguments

* `access_key_id` - (Required) Access key id, can be passed as AWS_ACCESS_KEY_ID env.
* `secret_access_key` - (Required) Secret access key, can be passed as AWS_SECRET_ACCESS_KEY env.
* `vpc_id` - (Required) Virtual private cloud identifier.
* `security_group_id` - (Optional) Security group identifier.
* `route_table_id` - (Optional) Route table identifier.
* `instance_profile_name` - (Optional) Instance profile name.
* `role_arn` - (Optional) The IAM role that the control plane will use.
* `openstack_billing_tenant` - (Required) Openstack Tenant/Project name for the account.

### syseleven_auth

Configure [SysEleven Login](https://docs.syseleven.de/metakube/en/tutorials/external-authentication) Realm to use.

#### Arguments
* `realm` - (Required) The name of the realm.
