# ---------------------------------------------------------------------------
# Resource group
# ---------------------------------------------------------------------------
output "resource_group_name" {
  description = "Name of the platform resource group"
  value       = azurerm_resource_group.main.name
}

# ---------------------------------------------------------------------------
# AKS cluster
# ---------------------------------------------------------------------------
output "cluster_name" {
  description = "Name of the AKS cluster"
  value       = azurerm_kubernetes_cluster.main.name
}

output "cluster_id" {
  description = "Resource ID of the AKS cluster"
  value       = azurerm_kubernetes_cluster.main.id
}

output "cluster_fqdn" {
  description = "FQDN of the AKS API server"
  value       = azurerm_kubernetes_cluster.main.fqdn
}

output "oidc_issuer_url" {
  description = "OIDC issuer URL — used when adding new federated credentials"
  value       = azurerm_kubernetes_cluster.main.oidc_issuer_url
}

output "kube_config_raw" {
  description = "Raw kubeconfig (sensitive) — use 'az aks get-credentials' instead where possible"
  sensitive   = true
  value       = azurerm_kubernetes_cluster.main.kube_config_raw
}

output "kube_config_command" {
  description = "az CLI command to merge cluster credentials into local kubeconfig"
  value       = "az aks get-credentials --resource-group ${azurerm_resource_group.main.name} --name ${azurerm_kubernetes_cluster.main.name} --overwrite-existing"
}

# ---------------------------------------------------------------------------
# ACR
# ---------------------------------------------------------------------------
output "acr_login_server" {
  description = "ACR login server hostname (used in image references and Argo CD)"
  value       = azurerm_container_registry.main.login_server
}

output "acr_id" {
  description = "Resource ID of the container registry"
  value       = azurerm_container_registry.main.id
}

# ---------------------------------------------------------------------------
# Bootstrap Key Vault
# ---------------------------------------------------------------------------
output "keyvault_uri" {
  description = "URI of the bootstrap Key Vault (used in ESO ClusterSecretStore)"
  value       = azurerm_key_vault.bootstrap.vault_uri
}

output "keyvault_id" {
  description = "Resource ID of the bootstrap Key Vault"
  value       = azurerm_key_vault.bootstrap.id
}

# ---------------------------------------------------------------------------
# Crossplane managed identity
# ---------------------------------------------------------------------------
output "crossplane_identity_client_id" {
  description = "Client ID — annotate the Crossplane provider ServiceAccount with this value"
  value       = azurerm_user_assigned_identity.crossplane.client_id
}

output "crossplane_identity_principal_id" {
  description = "Principal ID of the Crossplane managed identity"
  value       = azurerm_user_assigned_identity.crossplane.principal_id
}

# ---------------------------------------------------------------------------
# ESO managed identity
# ---------------------------------------------------------------------------
output "eso_identity_client_id" {
  description = "Client ID — annotate the ESO controller ServiceAccount with this value"
  value       = azurerm_user_assigned_identity.eso.client_id
}

output "eso_identity_principal_id" {
  description = "Principal ID of the ESO managed identity"
  value       = azurerm_user_assigned_identity.eso.principal_id
}

# ---------------------------------------------------------------------------
# Azure context (consumed by downstream automation / docs)
# ---------------------------------------------------------------------------
output "subscription_id" {
  description = "Azure subscription ID"
  sensitive   = true
  value       = data.azurerm_client_config.current.subscription_id
}

output "tenant_id" {
  description = "Azure tenant ID"
  sensitive   = true
  value       = data.azurerm_client_config.current.tenant_id
}
