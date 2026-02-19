# ---------------------------------------------------------------------------
# Azure Monitor managed Prometheus + Grafana
#
# All resources are gated on var.enable_monitoring for cost control.
# Deployment order:
#   1. Azure Monitor Workspace (Prometheus-compatible store)
#   2. Azure Managed Grafana  (visualization; linked to the workspace)
#   3. RBAC: Grafana Admin for the TF principal
#   4. RBAC: Monitoring Data Reader so Grafana can read the workspace
#   5. Data Collection Endpoint + Rule (DCR) — routes AKS metrics to the workspace
#   6. Data Collection Rule Association (DCRA) — binds the DCR to the AKS cluster
#   7. monitor_metrics block on the AKS cluster (see aks.tf) is enabled when
#      enable_monitoring = true via a dynamic block
# ---------------------------------------------------------------------------

# ---------------------------------------------------------------------------
# 1. Azure Monitor Workspace  (Prometheus-compatible)
# ---------------------------------------------------------------------------
resource "azurerm_monitor_workspace" "main" {
  count = var.enable_monitoring ? 1 : 0

  name                = "amw-${local.cluster_name}"
  resource_group_name = azurerm_resource_group.main.name
  location            = azurerm_resource_group.main.location
  tags                = local.tags
}

# ---------------------------------------------------------------------------
# 2. Azure Managed Grafana
# ---------------------------------------------------------------------------
resource "azurerm_dashboard_grafana" "main" {
  count = var.enable_monitoring ? 1 : 0

  name                              = "graf-${local.cluster_name}"
  resource_group_name               = azurerm_resource_group.main.name
  location                          = azurerm_resource_group.main.location
  sku                               = "Standard"
  grafana_major_version             = 12
  public_network_access_enabled     = true
  zone_redundancy_enabled           = false
  api_key_enabled                   = false
  deterministic_outbound_ip_enabled = false

  # Link this Grafana instance to the Azure Monitor Workspace so the
  # built-in Azure Monitor data source is pre-configured automatically.
  azure_monitor_workspace_integrations {
    resource_id = azurerm_monitor_workspace.main[0].id
  }

  identity {
    type = "SystemAssigned"
  }

  tags = local.tags
}

# ---------------------------------------------------------------------------
# 3. RBAC — Grafana Admin for the Terraform service principal
#    Allows the TF caller (and therefore the homelab operator) to log in
#    as a Grafana admin without manual portal clicks.
# ---------------------------------------------------------------------------
resource "azurerm_role_assignment" "grafana_admin" {
  count = var.enable_monitoring ? 1 : 0

  principal_id         = data.azurerm_client_config.current.object_id
  role_definition_name = "Grafana Admin"
  scope                = azurerm_dashboard_grafana.main[0].id
}

# ---------------------------------------------------------------------------
# 4. RBAC — Monitoring Data Reader on the Monitor Workspace for Grafana
#    Grafana's system-assigned identity needs read access to pull metrics
#    from the Azure Monitor Workspace.
# ---------------------------------------------------------------------------
resource "azurerm_role_assignment" "grafana_monitor_reader" {
  count = var.enable_monitoring ? 1 : 0

  principal_id                      = azurerm_dashboard_grafana.main[0].identity[0].principal_id
  role_definition_name              = "Monitoring Data Reader"
  scope                             = azurerm_monitor_workspace.main[0].id
  skip_service_principal_aad_check  = true
}

# ---------------------------------------------------------------------------
# 5a. Data Collection Endpoint
#     Required by the Managed Prometheus DCR for AKS scraping.
# ---------------------------------------------------------------------------
resource "azurerm_monitor_data_collection_endpoint" "main" {
  count = var.enable_monitoring ? 1 : 0

  name                = "dce-${local.cluster_name}-prometheus"
  resource_group_name = azurerm_resource_group.main.name
  location            = azurerm_resource_group.main.location
  kind                = "Linux"
  tags                = local.tags
}

# ---------------------------------------------------------------------------
# 5b. Data Collection Rule — wires the AKS Prometheus scraper to the workspace
# ---------------------------------------------------------------------------
resource "azurerm_monitor_data_collection_rule" "prometheus" {
  count = var.enable_monitoring ? 1 : 0

  name                        = "dcr-${local.cluster_name}-prometheus"
  resource_group_name         = azurerm_resource_group.main.name
  location                    = azurerm_resource_group.main.location
  data_collection_endpoint_id = azurerm_monitor_data_collection_endpoint.main[0].id
  kind                        = "Linux"
  tags                        = local.tags

  destinations {
    monitor_account {
      monitor_account_id = azurerm_monitor_workspace.main[0].id
      name               = "MonitoringAccount"
    }
  }

  data_flow {
    streams      = ["Microsoft-PrometheusMetrics"]
    destinations = ["MonitoringAccount"]
  }

  data_sources {
    prometheus_forwarder {
      name    = "PrometheusDataSource"
      streams = ["Microsoft-PrometheusMetrics"]
    }
  }
}

# ---------------------------------------------------------------------------
# 6. Data Collection Rule Association — binds the DCR to the AKS cluster
# ---------------------------------------------------------------------------
resource "azurerm_monitor_data_collection_rule_association" "aks" {
  count = var.enable_monitoring ? 1 : 0

  name                    = "dcra-${local.cluster_name}-prometheus"
  target_resource_id      = azurerm_kubernetes_cluster.main.id
  data_collection_rule_id = azurerm_monitor_data_collection_rule.prometheus[0].id
  description             = "Routes AKS Managed Prometheus metrics to the Azure Monitor Workspace"
}
