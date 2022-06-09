variable "project_id" {
  description = "ID of the MetaKube project"
  type        = string
}
variable "public_sshkey_file" {
  description = "Path to public ssh key to add to cluster"
  type        = string
  default     = "~/.ssh/id_rsa.pub"
}
variable "cluster_name" {
  description = "Name of the MetaKube cluster"
  type        = string
}

variable "dc_name" {
  description = "Datacenter where the cluster should be deployed to. Get a list of possible option with `curl -s -H \"authorization: Bearer $METAKUBE_TOKEN\" https://metakube.syseleven.de/api/v1/dc | jq -r '.[] | select(.seed!=true) | .metadata.name'`"
  type        = string
  default     = "syseleven-cbk1"
}

variable "floating_ip_pool" {
  description = "Floating IP pool to use for all worker nodes"
  type        = string
  default     = "ext-net"
}

variable "username" {
  description = "OpenStack username"
  type        = string
  default     = null
}

variable "password" {
  description = "OpenStack password"
  type        = string
  default     = null
}

variable "openstack_project_id" {
  description = "OpenStack project id"
  type        = string
  default     = null
}

variable "openstack_project_name" {
  description = "OpenStack project name"
  type        = string
  default     = null
}

variable "node_flavor" {
  description = "Flavor of the k8s worker node"
  type        = string
  default     = "m1.medium"
}

variable "node_image" {
  description = "Name of the image which should be used for worker nodes"
  type        = string
  default     = null
}

variable "node_replicas" {
  description = "Amount of worker nodes in this node deployment"
  type        = string
  default     = 3
}

variable "use_floating_ip" {
  description = "If the node deployment should use floating IPs"
  type        = bool
  default     = true
}

