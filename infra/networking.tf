resource "azurerm_virtual_network" "main" {
  name                = "vnet-${local.cluster_name}"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  address_space       = var.vnet_address_space

  tags = local.tags
}

# AKS node subnet — only node IPs are drawn from here.
# With Azure CNI overlay + Cilium, pod IPs come from var.pod_cidr (not this subnet).
resource "azurerm_subnet" "aks" {
  name                 = "snet-aks"
  resource_group_name  = azurerm_resource_group.main.name
  virtual_network_name = azurerm_virtual_network.main.name
  address_prefixes     = [var.aks_subnet_cidr]
}