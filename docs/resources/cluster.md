# cluster Resource

Cluster resource in the provider defines the corresponding cluster in MetaKube.

## Example Usage

```hcl
resource "metakube_cluster" "example" {
  project_id = metakube_project.example.id
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
* `sshkeys` - (Optional) SSH keys attached to nodes.

## Attributes

* `kube_config` - Kube config raw content which can be dumped to a file using [local_file](https://registry.terraform.io/providers/hashicorp/local/latest/docs/resources/file).
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
* `services_cidr` - (Optional) Internal IP range for ClusterIP Services.
* `pods_cidr` - (Optional) Internal IP range for Pods.
* `domain_name` - (Optional) Cluster domain name.

### `cloud`

One of the following must be selected.

#### Arguments

* `bringyourown` - (Optional) User defined infrastructure.
* `openstack` - (Optional) Opestack infrastructure.
* `aws` - (Optional) Amazon Web Services infrastructure.
* `azure` - (Optional) Azure infrastructure.

### `openstack`

#### Arguments
* `floating_ip_pool` - (Required) The floating ip pool used by all worker nodes to receive a public ip.
* `security_group` - (Optional) When specified, all worker nodes will be attached to this security group. If not specified, a security group will be created.
* `network` - (Optional) When specified, all worker nodes will be attached to this network. If not specified, a network, subnet & router will be created.
* `subnet_id` - (Optional) When specified, all worker nodes will be attached to this subnet of specified network. If not specified, a network, subnet & router will be created.
* `subnet_cidr` - Change this to configure a different internal IP range for Nodes. Default: `192.168.1.0/24`.
When using password based auth
* `tenant` - (Optional) The project to use for billing. You can set it using environment variable `OS_PROJECT`. Must be omit if application credentials are used.
* `username` - (Optional) The account's username. You can set it using environment variable `OS_USERNAME`. Must be omit if application credentials are used.
* `password` - (Optional) The account's password. You can set it using environment variable `OS_PASSWORD`. Must be omit if application credentials are used.
When using application credentials
* `application_credentials_id` - (Opitonal) Application credentials ID to use. Must be omit if username/password/tenant are used.
* `application_credentials_secret` - (Opitonal) Application credentials Secret to use. Must be omit if username/password/tenant are used.

### `aws`

#### Arguments

* `access_key_id` - (Required) Access key id, can be passed as AWS_ACCESS_KEY_ID env.
* `secret_access_key` - (Required) Secret access key, can be passed as AWS_SECRET_ACCESS_KEY env.
* `vpc_id` - (Optional) Virtual private cloud identifier.
* `security_group_id` - (Optional) Security group identifier.
* `route_table_id` - (Optional) Route table identifier.
* `instance_profile_name` - (Optional) Instance profile name.
* `role_arn` - (Optional) The IAM role that the control plane will use.
* `openstack_billing_tenant` - (Required) Openstack Tenant/Project name for the account.

### `azure`

#### Arguments
* `availability_set` - (Optional) Availability set name.
* `client_id` - (Required) Client id.
* `client_secret` - (Required) Client secret.
* `subscription_id` - (Required) Subscription id.
* `tenant_id` - (Required) Tenant id.
* `resource_group` - (Optional) Resource group name.
* `route_table` - (Optional) Route table name.
* `security_group` - (Optional) Security group name.
* `subnet` - (Optional) Subnet.
* `vnet` - (Optional) Vnet.
* `openstack_billing_tenant` - (Required) Openstack Tenant/Project name for the account.
