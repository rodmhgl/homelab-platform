# homelab-platform/infra

Terraform for foundational AKS platform infrastructure.

Runs in **Terraform Cloud**.

## What this provisions

| Resource | Details |
| --- | --- |
| Resource Group | `rg-homelab-aks-<env>` |
| VNet + AKS subnet | `10.10.0.0/16` / `10.10.0.0/22` |
| AKS cluster | Free tier, Azure CNI Powered by Cilium (overlay), Workload Identity, Entra RBAC only |
| ACR | Basic SKU — `homelabplatformacr` |
| Bootstrap Key Vault | Standard SKU — platform secrets consumed by ESO |
| Managed Identities | `crossplane` (Contributor on sub) + `eso` (KV Secrets User) |

**Not here:** app-level infra (storage accounts, key vaults) — those are Crossplane Claims.

## Prerequisites

Set in TFC workspace as sensitive variables:

- `subscription_id` — Azure subscription

TFC service principal needs `Owner` or `Contributor + User Access Administrator` on the subscription.

## Usage

```bash
# Apply via TFC (push to main triggers run)
git push origin main

# Get credentials after apply
az aks get-credentials --resource-group rg-homelab-aks-dev --name homelab-aks-dev --overwrite-existing
```

## Key outputs

| Output | Use |
| --- | --- |
| `crossplane_identity_client_id` | Annotate Crossplane provider ServiceAccount |
| `eso_identity_client_id` | Annotate ESO controller ServiceAccount |
| `keyvault_uri` | ESO `ClusterSecretStore` spec |
| `oidc_issuer_url` | Adding future federated credentials |
| `acr_login_server` | Image references, Argo CD |

## Crossplane identity caveat

The federated credential subject is `system:serviceaccount:crossplane-system:provider-azure`. Upbound providers generate SAs with a hash suffix — update `crossplane_service_account` in `variables.tf` and re-apply after first Crossplane install.

## CIDR layout

```text
VNet:          10.10.0.0/16
AKS nodes:     10.10.0.0/22   (drawn from VNet)
Pod overlay:   192.168.0.0/16 (Cilium; not in VNet)
Services:      172.16.0.0/16  (not in VNet)
kube-dns:      172.16.0.10
```
