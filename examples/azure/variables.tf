variable "project_id" {
  description = "ID of the MetaKube project"
  type        = string
}

variable "cluster_name" {
  description = "Name of the MetaKube cluster"
  type        = string
}

variable "k8s_minor_version" {
  description = "The minor part of Kubernetes version, eg 21 for v1.21.2"
  type        = string
}

variable "azure_client_id" {
  description = "Azure Client ID"
  type        = string
}

variable "azure_subscription_id" {
  description = "Azure Subscription ID"
  type        = string
}

variable "azure_tenant_id" {
  description = "Azure Tenant ID"
  type        = string
}

variable "azure_client_secret" {
  description = "Azure Client Secret"
  type        = string
}
