---
page_title: "MetaKube: metakube_k8s_version"
---

# metakube_k8s_version

Get the latest supported kubernetes version matching `major` & `minor`.

## Example Usage

Get the latest supported patch of Kubernetes v1.21.x:

```hcl
data "metakube_k8s_version" "example" {
  major = "1"
  minor = "21"
}

resource "metakube_cluster" "foo" {
  # ...
  spec = {
    version = data.metakube_k8s_version.example.version
    # ...
  }
  # ...
}
```
## Argument Reference

The following arguments are supported:

* `major` - (Optional) Major version, defaults to the latest available.
* `minor` - (Optional) Minor version, cannot be specified without `major`, defaults to the latest available.

## Attributes Reference

The only attribute exported is:
* `version`:  The latest Kubernetes version supported by MetaKube matching `major` and `minor`.
