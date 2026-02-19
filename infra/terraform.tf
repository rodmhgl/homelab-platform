terraform {
  required_version = ">= 1.9.0"

  cloud {
    organization = "rnlabs"

    workspaces {
      name = "aks-platform"
    }
  }

  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 4.60"
    }
    azuread = {
      source  = "hashicorp/azuread"
      version = "~> 3.7"
    }
  }
}