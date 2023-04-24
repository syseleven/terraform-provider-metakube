---
page_title: "MetaKube: metakube_project"
---

# metakube_project

Get project ID by Name.

## Example Usage

```hcl
data "metakube_project" "example" {
  name = "staging clusters"
}

resource "metakube_cluster" "foo" {
  project_id = data.metakube_project.id
  # ...
}
```
## Argument Reference

The following arguments are supported:

* `name` - (Required) Name of the project.

## Attributes Reference

The only attribute exported is:
* `id`:  Use this as `project_id` for other resources like `metakube_cluster`
