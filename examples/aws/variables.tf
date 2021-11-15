variable "project_id" {
  description = "ID of the MetaKube project"
  type        = string
}

variable "cluster_name" {
  description = "Name of the MetaKube cluster"
  type        = string
}

variable "aws_access_key_id" {
  description = "AWS Access Key ID"
  type        = string
}

variable "aws_secret_access_key" {
  description = "AWS Access Key Secret"
  type        = string
}

variable "openstack_billing_tenant" {
  description = "Openstack project that metakube fees should be belled to"
  type        = string
}

