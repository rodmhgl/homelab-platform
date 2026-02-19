# ---------------------------------------------------------------------------
# Provider / subscription
# ---------------------------------------------------------------------------
variable "subscription_id" {
  description = "Azure subscription ID"
  type        = string
  sensitive   = true
}

# ---------------------------------------------------------------------------
# Naming & location
# ---------------------------------------------------------------------------
variable "location" {
  description = "Azure region for all resources"
  type        = string
  default     = "southcentralus"
}

variable "cluster_name" {
  description = "Base name for the AKS cluster and derived resources"
  type        = string
  default     = "homelab-aks"
}

variable "environment" {
  description = "Short environment label used in resource naming and tags (e.g. dev, prod)"
  type        = string
  default     = "dev"

  validation {
    condition     = can(regex("^[a-z0-9-]+$", var.environment))
    error_message = "environment must be lowercase alphanumeric with hyphens only."
  }
}

# ---------------------------------------------------------------------------
# AKS
# ---------------------------------------------------------------------------
variable "kubernetes_version" {
  description = "Kubernetes version. null = AKS-managed latest patch for the minor version."
  type        = string
  default     = null
}

variable "node_pool" {
  description = "Default (single) node pool configuration"
  type = object({
    node_count      = optional(number, 3)
    vm_size         = optional(string, "Standard_B4ms")
    os_disk_size_gb = optional(number, 128)
    os_disk_type    = optional(string, "Managed")
    max_pods        = optional(number, 50)
    max_surge       = optional(string, "1")
  })
  default = {}
}

# ---------------------------------------------------------------------------
# Networking
# ---------------------------------------------------------------------------
variable "vnet_address_space" {
  description = "Address space for the VNet"
  type        = list(string)
  default     = ["10.10.0.0/16"]
}

variable "aks_subnet_cidr" {
  description = "CIDR for the AKS node subnet (node IPs are drawn from here)"
  type        = string
  default     = "10.10.0.0/22"
}

variable "service_cidr" {
  description = "CIDR for Kubernetes service IPs (must not overlap VNet)"
  type        = string
  default     = "172.16.0.0/16"
}

variable "dns_service_ip" {
  description = "IP address within service_cidr reserved for kube-dns"
  type        = string
  default     = "172.16.0.10"
}

variable "pod_cidr" {
  description = "Pod overlay CIDR (Cilium overlay mode; not consumed from VNet)"
  type        = string
  default     = "192.168.0.0/16"
}

# ---------------------------------------------------------------------------
# ACR
# ---------------------------------------------------------------------------
variable "acr_name" {
  description = "Globally unique name for the Azure Container Registry (lowercase alphanumeric, 5-50 chars)"
  type        = string
  default     = "homelabplatformacr"
}

# ---------------------------------------------------------------------------
# Bootstrap Key Vault
# ---------------------------------------------------------------------------
variable "keyvault_name" {
  description = "Globally unique name for the bootstrap Key Vault (3-24 chars)"
  type        = string
  default     = "homelab-bootstrap-kv"
}

# ---------------------------------------------------------------------------
# Crossplane workload identity
# ---------------------------------------------------------------------------
variable "crossplane_namespace" {
  description = "Kubernetes namespace where Crossplane providers run"
  type        = string
  default     = "crossplane-system"
}

variable "crossplane_service_account" {
  description = "Name of the Crossplane provider ServiceAccount"
  type        = string
  default     = "provider-azure"
}

# ---------------------------------------------------------------------------
# ESO workload identity
# ---------------------------------------------------------------------------
variable "eso_namespace" {
  description = "Kubernetes namespace where External Secrets Operator runs"
  type        = string
  default     = "external-secrets"
}

variable "eso_service_account" {
  description = "Name of the ESO controller ServiceAccount"
  type        = string
  default     = "external-secrets"
}

# ---------------------------------------------------------------------------
# Tagging
# ---------------------------------------------------------------------------
variable "tags" {
  description = "Additional tags merged with the default tag set"
  type        = map(string)
  default     = {}
}