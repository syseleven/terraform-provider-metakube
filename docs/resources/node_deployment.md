# node_deployment Resource

Node deployment resource in the provider defines the corresponding deployment of nodes.

## Example usage

```hcl
resource "metakube_cluster" "example_cluster" {
  # [...]
}

resource "metakube_node_deployment" "example_node" {
  cluster_id = metakube_cluster.example_cluster.id
  spec {
    replicas = 1
    template {
      cloud {
        aws {
          instance_type     = "t3.small"
          disk_size         = 25
          volume_type       = "standard"
          subnet_id         = "subnet-04f2f551bbc697db3"
          availability_zone = "eu-central-1c"
          assign_public_ip  = true
        }
      }
      operating_system {
        ubuntu {
          dist_upgrade_on_boot = true
        }
      }
      versions {
        kubelet = "1.21.3"
      }
    }
  }
}
```

### Example with machine annotations

To configure the user-data plugin for Ubuntu nodes with sysext:

```hcl
resource "metakube_node_deployment" "example_with_annotations" {
  cluster_id = metakube_cluster.example_cluster.id
  spec {
    replicas = 1
    template {
      cloud {
        openstack {
          flavor = "m1.small"
          image  = "Ubuntu 22.04"
        }
      }
      operating_system {
        ubuntu {
          dist_upgrade_on_boot = false
        }
      }
      machine_annotations = {
        "machines.metakube.syseleven.de/user-data-plugin" = "ubuntu-sysext"
      }
    }
  }
}
```

## Argument reference

The following arguments are supported:

* `cluster_id` - (Required) Reference cluster id.
* `name` - (Optional) Node deployment name.
* `spec` - (Required) Node deployment specification.

### Timeouts

`metakube_node_deployment` provides the following Timeouts configuration options:

* create - (Default 20 minutes) Used for Creating node deployments and waiting for them to join the cluster.
* update - (Default 20 minutes) Used for node deployment modifications.
* delete - (Default 20 minutes) Used for destroying node deployment.

Example usage:

```hcl
resource "metakube_node_deployment" "example" {
  # ... other configuration ...

  timeouts {
    create = "45m"
    update = "45m"
    delete = "30m"
  }
}
```

## Attributes

* `creation_timestamp` - Timestamp of resource creation.
* `deletion_timestamp` - Timestamp of resource deletion.

## Nested Blocks

### `spec`

#### Arguments

* `replicas` - (Optional) Number of replicas, default = 1.
* `template` - (Required) Template specification.
* `min_replicas` - (Optional) Minimum number of replicas to downscale node deployment to. Be aware that:
  * downscaling is not supported for kubernetes versions below `1.18.0`.
  * downscaling to `0` is not supported.
* `max_replicas` - (Optional) Maximum number of replicas to upscale node deployment to.

### `template`

#### Arguments

* `cloud` - (Required) Cloud specification.
* `operating_system` - (Required) Operating system settings.
* `versions` - (Optional) K8s components versions.
* `labels` - (Optional) Map of string keys and values that can be used to organize and categorize (scope and select) objects. It will be applied to Nodes allowing users run their apps on specific Node using labelSelector. Note: The server adds system labels (`system/cluster`, `system/project`) which are available in `all_labels`.
* `all_labels` - (Computed) All labels on the node deployment, including user-specified labels and system-managed labels (`system/cluster`, `system/project`) added by the server. This is read-only.
* `taints` - (Optional) List of taints to set on nodes.
* `node_annotations` - (Optional) Map of annotations to set on nodes.
* `machine_annotations` - (Optional) Map of annotations to set on machine objects. Use this to configure machine-controller behavior, such as the user-data plugin.

### `cloud`

One of the following must be selected.

#### Arguments

* `openstack` - (Optional) Openstack node deployment specification.
* `aws` - (Optional) AWS node deployment specification.

### `operating_system`

One of the following must be selected.

#### Arguments

* `ubuntu` - (Exactly one choice, this or another required) Ubuntu operating system and its settings.
* `flatcar` - (Exactly one choice, this or another required) Flatcar operating system and its settings.

### `versions`

#### Arguments

* `kubelet` - (Optional) Kubelet version.

### `taints`

#### Arguments

* `effect` - (Required) Effect for taint. Accepted values are NoSchedule, PreferNoSchedule, and NoExecute.
* `key` - (Required) Key for taint.
* `value` - (Required) Value for taint.

### `openstack`
* `flavor` - (Required) Instance type.
* `image` - (Required) Image to use.
* `disk_size` - (Optional) If set, the rootDisk will be a cinder volume of that size in GiB. If unset, the rootDisk will be ephemeral nova root storage and its size will be derived from the flavor.
* `tags` - (Optional) Additional instance tags.
* `use_floating_ip` - (Optional) Indicate use of floating ip in case of floating_ip_pool presense. Defaults to true.
* `instance_ready_check_period` - (Optional) Specify custom value for how often to check if instance is ready before timing out.
* `instance_ready_check_timeout` - (Optional) Specifies custom value for how long to check if instance is ready before timing out.
* `server_group_id` - (Optional) Specifies custom value for the Openstack server group ID to use for the nodes. Defaults to a cluster-wide group.

### `aws`

#### Arguments

* `instance_type` - (Required) EC2 instance type
* `disk_size` - (Required) Size of the volume in GBs.
* `volume_type` -  (Required) EBS volume type.
* `availability_zone` - (Required) Availability zone in which to place the node. It is coupled with the subnet to which the node will belong.
* `subnet_id` - (Required) The VPC subnet to which the node shall be connected.
* `assign_public_ip` - (Optional) When set the AWS instance will get a public IP address assigned during launch overriding a possible setting in the used AWS subnet.
* `ami` - (Optional) Amazon Machine Image to use. Will be defaulted to an AMI of your selected operating system and region.
* `tags`- (Optional) Additional EC2 instance tags.

### `ubuntu`

#### Arguments

* `dist_upgrade_on_boot` - (Optional) Upgrade operating system on boot, default to false.

### `flatcar`

#### Arguments

* `disable_auto_update` - (Optional) Disable Flatcar auto update feature. Defaults to false.
