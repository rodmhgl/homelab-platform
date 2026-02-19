# Bootstrap Key Vault — holds platform secrets consumed by ESO:
#   • LLM API keys (for kagent / HolmesGPT)
#   • Any other platform-level credentials
#
# Secrets are seeded manually or via CI; Terraform only provisions the vault
# and the RBAC assignments needed for ESO and for the TFC service principal.

resource "azurerm_key_vault" "bootstrap" {
  name                = var.keyvault_name
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  tenant_id           = data.azurerm_client_config.current.tenant_id
  sku_name            = "standard"

  # RBAC-based access control (modern; no per-identity access_policy blocks)
  rbac_authorization_enabled = true

  # Homelab-friendly: allow soft-deleted vaults to be force-purged on destroy
  # and recovered automatically on re-deploy. Purge protection would prevent
  # that, so it stays off.
  purge_protection_enabled   = false
  soft_delete_retention_days = 7

  tags = local.tags
}

# ---------------------------------------------------------------------------
# RBAC assignments on the bootstrap Key Vault
# ---------------------------------------------------------------------------

# The Terraform/TFC service principal needs to write secrets during bootstrap
# (e.g. seeding initial LLM API key placeholders or rotation tokens).
resource "azurerm_role_assignment" "kv_secrets_officer_terraform" {
  principal_id         = data.azurerm_client_config.current.object_id
  role_definition_name = "Key Vault Secrets Officer"
  scope                = azurerm_key_vault.bootstrap.id
}

# ESO managed identity gets read-only access to all secrets.
resource "azurerm_role_assignment" "kv_secrets_user_eso" {
  principal_id                     = azurerm_user_assigned_identity.eso.principal_id
  role_definition_name             = "Key Vault Secrets User"
  scope                            = azurerm_key_vault.bootstrap.id
  skip_service_principal_aad_check = true
}