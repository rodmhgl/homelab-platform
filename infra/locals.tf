data "azurerm_client_config" "current" {}

locals {
  cluster_name = "${var.cluster_name}-${var.environment}"

  default_tags = {
    environment = var.environment
    managed_by  = "terraform"
    project     = "homelab-platform"
  }

  tags = merge(local.default_tags, var.tags)
}

resource "azurerm_resource_group" "main" {
  name     = "rg-${local.cluster_name}"
  location = var.location
  tags     = local.tags
}