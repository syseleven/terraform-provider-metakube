---
page_title: "MetaKube: metakube_sshkey"
---

# metakube_sshkey

Get sshkey record's public key and fingerprint data. Useful if you want to reference existing key.

## Example Usage

Dump public key into a file `keydata`.

```hcl
data "metakube_sshkey" "example" {
  project_id = "foo"
  name       = "work-laptop"
}

resource "local_file" "key_data" {
  content = "data.metakube_sshkey.example.public_key"
  filename = "${path.module}/keydata"
}
```
## Argument Reference

The following arguments are supported:

* `project_id` - (Optional) MetaKube Project ID.
* `name` - (Optional) Name of the sshkey record.

## Attributes Reference

The only attribute exported is:
* `public_key`:  The ssh key public key.
* `fingerprint`: The ssh key fingerprint.
