# homelab-platform

AKS Home Lab Internal Developer Platform (IDP) mono-repo.

## Status

| Directory | Status | Notes |
| --- | --- | --- |
| `infra/` | ✅ Complete | Terraform — AKS, VNet, ACR, bootstrap KV, managed identities. TFC: `rnlabs/aks-platform` |
| `platform/argocd/` | ✅ Complete | Argo CD — Helm values, self-manage Application, root App of Apps, Workload ApplicationSet, Projects, `bootstrap.sh` |
| `platform/crossplane/` | ✅ Complete | Crossplane core Helm install (wave 1) |
| `platform/crossplane-providers/` | ✅ Complete | DeploymentRuntimeConfig, Providers (family/storage/keyvault), function-patch-and-transform (wave 2) |
| `platform/crossplane-config/` | ✅ Complete | ProviderConfig (OIDCTokenFile), XRDs (StorageBucket/Vault), Compositions — Pipeline mode (wave 3) |
| `platform/gatekeeper/` | ✅ Complete | Gatekeeper Helm install (wave 4) |
| `platform/gatekeeper-templates/` | ✅ Complete | 8 ConstraintTemplates (wave 5) |
| `platform/gatekeeper-constraints/` | ✅ Complete | 8 Constraints with enforcementAction: deny (wave 6) |
| `platform/platform-api/` | ✅ Complete | Platform API Kubernetes manifests (Deployment, Service, RBAC, application.yaml). Secrets managed via ESO ExternalSecret (github-pat, openai-api-key, argocd-token). |
| `platform/external-secrets/` | ✅ Complete | ESO Helm install + ClusterSecretStore (Workload Identity, wave 3.5). Platform API ExternalSecret resources deployed. Placeholders require Terraform outputs. |
| `platform/trivy-operator/` | ✅ Complete | Trivy Operator v0.32.0 Helm install + values.yaml (wave 7). Continuous CVE scanning with VulnerabilityReport CRDs. |
| `platform/monitoring/` | ✅ Complete | kube-prometheus-stack Helm install (Prometheus + Alertmanager + Grafana, wave 8). Grafana admin credentials via ESO from bootstrap Key Vault. Alertmanager pre-configured for HolmesGPT webhook. Custom scrape configs for Crossplane, Gatekeeper, Trivy, Platform API. |
| `platform/falco/` | ⬜ Pending | Runtime security + Falcosidekick |
| `platform/kagent/` | ⬜ Pending | Natural language cluster interaction |
| `platform/holmesgpt/` | ⬜ Pending | AI-powered root cause analysis |
| `scaffolds/go-service/` | ✅ Complete | Copier template — 23 production-ready template files (copier.yml, main.go, Dockerfile, k8s/ manifests, Crossplane Claims, CI/CD pipeline, Makefile, golangci-lint, Dependabot, CODEOWNERS). Generates Gatekeeper-compliant apps with optional Azure infrastructure. |
| `scaffolds/python-service/` | ⬜ Pending | Copier template (not started) |
| `api/` | ✅ Complete | Platform API — Go + Chi router, structured logging, graceful shutdown. Endpoints: scaffold (#51), Argo CD apps (#42, #43, #89), compliance (#48), infra full CRUD (#44, #45, #46, #47). Complete GitOps infrastructure management (list/get/create/delete) with three-layer validation. RBAC configured. Secrets via ESO. Argo CD integration requires one-time token bootstrap (see `platform/platform-api/setup-argocd-token.sh`). |
| `cli/` | 🔨 In Progress | `rdp` CLI — Root command, config management (init/view/set), version, `status` (#66), and `infra list/status` (#68) complete. Next: interactive create/delete commands (#69-#71), apps/compliance/secrets/investigate/ask commands. |

## Bootstrap

```bash
# Point kubectl at the cluster first
az aks get-credentials --resource-group rg-homelab-aks-dev --name homelab-aks-dev --overwrite-existing

# Seed Argo CD (one-time)
REPO_URL=https://github.com/rodmhgl/homelab-platform ./platform/argocd/bootstrap.sh
```

After bootstrap, all subsequent platform changes are applied via `git push` — Argo CD reconciles automatically.

## Architecture

```text
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

```text
VNet:        10.10.0.0/16
AKS nodes:   10.10.0.0/22   (drawn from VNet)
Pod overlay: 192.168.0.0/16 (Cilium; not in VNet)
Services:    172.16.0.0/16  (not in VNet)
kube-dns:    172.16.0.10
```

## Key Terraform Outputs

| Output | Consumed by |
| --- | --- |
| `crossplane_identity_client_id` | `DeploymentRuntimeConfig` annotation |
| `eso_identity_client_id` | ESO ServiceAccount annotation |
| `keyvault_uri` | ESO `ClusterSecretStore` spec |
| `acr_login_server` | Image references |
