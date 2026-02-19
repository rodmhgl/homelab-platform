# ---------------------------------------------------------------------------
# AKS control-plane identity (used by AKS for VNet/subnet operations)
# ---------------------------------------------------------------------------
resource "azurerm_user_assigned_identity" "aks_control_plane" {
  name                = "id-${local.cluster_name}-cp"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  tags                = local.tags
}

# AKS needs Network Contributor on the subnet so it can attach NICs / manage
# IP configurations for nodes when using Azure CNI.
resource "azurerm_role_assignment" "aks_subnet_network_contributor" {
  principal_id         = azurerm_user_assigned_identity.aks_control_plane.principal_id
  role_definition_name = "Network Contributor"
  scope                = azurerm_subnet.aks.id
}

# ---------------------------------------------------------------------------
# AKS cluster
# ---------------------------------------------------------------------------
resource "azurerm_kubernetes_cluster" "main" {
  name                = local.cluster_name
  dns_prefix          = local.cluster_name
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  kubernetes_version  = var.kubernetes_version
  sku_tier            = "Free"

  # Managed (non-static) credential upgrade — patch-level auto-upgrades only.
  automatic_upgrade_channel = "patch"
  node_os_upgrade_channel   = "NodeImage"

  # Disable local (static-password) cluster accounts; Entra-only access.
  local_account_disabled = true

  azure_active_directory_role_based_access_control {
    tenant_id          = data.azurerm_client_config.current.tenant_id
    azure_rbac_enabled = true
  }

  # Workload Identity + OIDC issuer (required for Crossplane + ESO federation)
  oidc_issuer_enabled       = true
  workload_identity_enabled = true

  identity {
    type         = "UserAssigned"
    identity_ids = [azurerm_user_assigned_identity.aks_control_plane.id]
  }

  # Single node pool — system + workload on the same pool for cost efficiency.
  default_node_pool {
    name                         = "system"
    node_count                   = var.node_pool.node_count
    vm_size                      = var.node_pool.vm_size
    os_disk_size_gb              = var.node_pool.os_disk_size_gb
    os_disk_type                 = var.node_pool.os_disk_type
    max_pods                     = var.node_pool.max_pods
    vnet_subnet_id               = azurerm_subnet.aks.id
    only_critical_addons_enabled = false

    upgrade_settings {
      max_surge = var.node_pool.max_surge
    }
  }

  # Azure CNI Powered by Cilium (overlay mode)
  # Nodes draw IPs from aks_subnet_cidr; pods get IPs from pod_cidr overlay.
  network_profile {
    network_plugin      = "azure"
    network_plugin_mode = "overlay"
    network_data_plane  = "cilium"
    network_policy      = "cilium"
    load_balancer_sku   = "standard"
    service_cidr        = var.service_cidr
    dns_service_ip      = var.dns_service_ip
    pod_cidr            = var.pod_cidr
  }

  # ACR integration — grants AcrPull to the kubelet identity automatically.
  # The kubelet_identity block is populated by AKS; we wire the role assignment
  # to it in acr.tf after the cluster is created.

  tags = local.tags

  depends_on = [
    azurerm_role_assignment.aks_subnet_network_contributor,
  ]
}