# ---------------------------------------------------------------------------
# Crossplane managed identity
# ---------------------------------------------------------------------------
# Used by the Crossplane Azure provider pods via Workload Identity federation.
# The DeploymentRuntimeConfig in platform/crossplane/providers/runtime-config.yaml
# annotates the provider ServiceAccount with this client ID.

resource "azurerm_user_assigned_identity" "crossplane" {
  name                = "id-${local.cluster_name}-crossplane"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  tags                = local.tags
}

# Federated credential: allows the Crossplane provider ServiceAccount to
# exchange its cluster-issued OIDC token for an Azure access token.
resource "azurerm_federated_identity_credential" "crossplane" {
  name      = "crossplane-provider-azure"
  parent_id = azurerm_user_assigned_identity.crossplane.id
  audience  = ["api://AzureADTokenExchange"]
  issuer    = azurerm_kubernetes_cluster.main.oidc_issuer_url
  subject   = "system:serviceaccount:${var.crossplane_namespace}:${var.crossplane_service_account}"
}

# Crossplane needs Contributor on the subscription to provision resource groups,
# storage accounts, key vaults, etc. on behalf of developers.
# TODO: Scope to a dedicated management resource group in production.
resource "azurerm_role_assignment" "crossplane_contributor" {
  principal_id                     = azurerm_user_assigned_identity.crossplane.principal_id
  role_definition_name             = "Contributor"
  scope                            = "/subscriptions/${data.azurerm_client_config.current.subscription_id}"
  skip_service_principal_aad_check = true
}

# ---------------------------------------------------------------------------
# ESO managed identity
# ---------------------------------------------------------------------------
# Used by the External Secrets Operator controller pod to read secrets from
# the bootstrap Key Vault. The ClusterSecretStore in
# platform/external-secrets/ annotates the ESO ServiceAccount with this
# client ID.

resource "azurerm_user_assigned_identity" "eso" {
  name                = "id-${local.cluster_name}-eso"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  tags                = local.tags
}

# Federated credential: allows the ESO controller ServiceAccount to exchange
# its cluster-issued OIDC token for an Azure access token.
resource "azurerm_federated_identity_credential" "eso" {
  name      = "eso-controller"
  parent_id = azurerm_user_assigned_identity.eso.id
  audience  = ["api://AzureADTokenExchange"]
  issuer    = azurerm_kubernetes_cluster.main.oidc_issuer_url
  subject   = "system:serviceaccount:${var.eso_namespace}:${var.eso_service_account}"
}

# ESO Key Vault access is wired in keyvault.tf (azurerm_role_assignment.kv_secrets_user_eso)
# to avoid a circular dependency between identities.tf and keyvault.tf.
