# Basic SKU — the cheapest tier; sufficient for a single-user homelab.
# No geo-replication, no content trust, no private endpoints needed.
resource "azurerm_container_registry" "main" {
  name                = var.acr_name
  resource_group_name = azurerm_resource_group.main.name
  location            = azurerm_resource_group.main.location
  sku                 = "Basic"
  admin_enabled       = false

  tags = local.tags
}

# Grant the AKS kubelet identity AcrPull on this registry.
# AKS populates kubelet_identity after cluster creation; we reference it here.
resource "azurerm_role_assignment" "aks_acr_pull" {
  principal_id                     = azurerm_kubernetes_cluster.main.kubelet_identity[0].object_id
  role_definition_name             = "AcrPull"
  scope                            = azurerm_container_registry.main.id
  skip_service_principal_aad_check = true
}
