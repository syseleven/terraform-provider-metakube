# OpenStack Basic Example

This examples shows how to create MetaKube project and deploy Kubernetes cluster on OpenStack using Terraform.

To run this example:
1. set up your OpenStack provider credentials. You can do that by downloading and sourcing "OpenStack RC File v3"
for your account at https://cloud.syseleven.de or configure it manually by uncommenting [corresponding section in
the file](./main.tf#{L12:L23}).
2. configure MetaKube host and token as described in the [documentation](https://registry.terraform.io/providers/syseleven/metakube/latest/docs)

Running the example
```
terraform init .
terraform apply
```
