# homelab-platform

AKS Home Lab Internal Developer Platform (IDP) mono-repo.

## Status

| Directory | Status | Notes |
|---|---|---|
| `infra/` | ✅ Complete | Terraform — AKS, VNet, ACR, bootstrap KV, managed identities. TFC: `rnlabs/aks-platform` |
| `platform/argocd/` | ✅ Complete | Argo CD — Helm values, self-manage Application, root App of Apps, Workload ApplicationSet, Projects, `bootstrap.sh` |
| `platform/crossplane/` | ✅ Complete | Crossplane core Helm install (wave 1) |
| `platform/crossplane-providers/` | ✅ Complete | DeploymentRuntimeConfig, Providers (family/storage/keyvault), function-patch-and-transform (wave 2) |
| `platform/crossplane-config/` | ✅ Complete | ProviderConfig (OIDCTokenFile), XRDs (StorageBucket/Vault), Compositions — Pipeline mode (wave 3) |
| `platform/gatekeeper/` | ⬜ Pending | OPA ConstraintTemplates + Constraints |
| `platform/external-secrets/` | ⬜ Pending | ESO + ClusterSecretStore |
| `platform/trivy-operator/` | ⬜ Pending | CVE scanning |
| `platform/falco/` | ⬜ Pending | Runtime security + Falcosidekick |
| `platform/monitoring/` | ⬜ Pending | kube-prometheus-stack + Grafana dashboards |
| `platform/kagent/` | ⬜ Pending | Natural language cluster interaction |
| `platform/holmesgpt/` | ⬜ Pending | AI-powered root cause analysis |
| `scaffolds/` | ⬜ Pending | Copier templates (go-service, python-service) |
| `api/` | ⬜ Pending | Platform API — Go + Chi |
| `cli/` | ⬜ Pending | `rdp` CLI — Go + Cobra + bubbletea |

## Bootstrap

```bash
# Point kubectl at the cluster first
az aks get-credentials --resource-group rg-homelab-aks-dev --name homelab-aks-dev --overwrite-existing

# Seed Argo CD (one-time)
REPO_URL=https://github.com/rodmhgl/homelab-platform ./platform/argocd/bootstrap.sh
```

After bootstrap, all subsequent platform changes are applied via `git push` — Argo CD reconciles automatically.

## Architecture

```
Terraform (infra/)          — foundational: AKS, VNet, ACR, bootstrap KV, managed identities
Argo CD (platform/argocd/)  — GitOps control plane; App of Apps pattern
Crossplane                  — self-service app infra (storage, key vaults) via Claims
Gatekeeper                  — admission policy for apps AND Crossplane Claims
ESO                         — platform secrets from bootstrap KV via Workload Identity
Trivy + Falco               — CVE scanning + runtime security
Platform API (api/)         — Go + Chi; all CLI/UI operations go through here
rdp CLI (cli/)              — Go + Cobra; thin client over Platform API
```

**Terraform ↔ Crossplane boundary:** Terraform owns platform-level infra. Crossplane owns app-level infra that developers consume via Claims. Do not cross this line.

**GitOps contract:** The `/api/v1/infra` endpoints commit Claim YAML to the app repo — they never write directly to the cluster. Git is the single source of truth.

## CIDR Layout

```
VNet:        10.10.0.0/16
AKS nodes:   10.10.0.0/22   (drawn from VNet)
Pod overlay: 192.168.0.0/16 (Cilium; not in VNet)
Services:    172.16.0.0/16  (not in VNet)
kube-dns:    172.16.0.10
```

## Key Terraform Outputs

| Output | Consumed by |
|---|---|
| `crossplane_identity_client_id` | `DeploymentRuntimeConfig` annotation |
| `eso_identity_client_id` | ESO ServiceAccount annotation |
| `keyvault_uri` | ESO `ClusterSecretStore` spec |
| `acr_login_server` | Image references |
