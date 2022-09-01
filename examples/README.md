# MetaKube Provider Examples

This directory contains a set of examples of using MetaKube with
Terraform. The examples each have their own README containing more details
on what the example does.

To run any example, clone the repository and run `terraform apply` within
the example's own directory.

For example:

```
$ git clone https://github.com/syseleven/terraform-provider-metakube
$ cd terraform-provider-metakube/examples/aws
$ terraform apply
...
```

By cloud provider:
- [AWS](./aws)
- [Openstack](./openstack)
