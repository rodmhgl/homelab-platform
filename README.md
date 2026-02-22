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
| `platform/trivy-operator/` | ✅ Complete | Trivy Operator v0.32.0 (wave 7). **Fixed:** DB repo config (uses `mirror.gcr.io`, no version tags) + containerd socket mount for CRI access. VulnerabilityReport CRDs generating successfully. Integrates with Platform API compliance scoring. |
| `platform/monitoring/` | ✅ Complete | kube-prometheus-stack Helm install (Prometheus + Alertmanager + Grafana, wave 8). Grafana admin credentials via ESO from bootstrap Key Vault. Alertmanager pre-configured for HolmesGPT webhook. Custom scrape configs for Crossplane, Gatekeeper, Trivy, Platform API. **Ingress configured** — Grafana accessible at `grafana.rdp.azurelaboratory.com`. |
| `platform/falco/` | ✅ Complete | Falco v8.0.0 Helm install + 4 custom rules (wave 8). Modern eBPF driver for runtime security monitoring. Custom rules: shell spawning, sensitive file access, container drift, suspicious network connections. Monitors all namespaces except kube-system. HTTP output enabled to Falcosidekick. |
| `platform/falcosidekick/` | ✅ Complete | Falcosidekick v0.10.0 Helm install (wave 9). Routes Falco events to Platform API webhook. Prometheus metrics enabled via ServiceMonitor. |
| `platform/portal-ui/` | ✅ Complete | Portal UI Kubernetes manifests (Deployment, Service, application.yaml, wave 11). React app runtime (2 replicas, ClusterIP, security-hardened). |
| `platform/kagent/` | ⬜ Pending | Natural language cluster interaction |
| `platform/holmesgpt/` | ⬜ Pending | AI-powered root cause analysis |
| `scaffolds/go-service/` | ✅ Complete | Copier template — 23 production-ready template files (copier.yml, main.go, Dockerfile, k8s/ manifests, Crossplane Claims, CI/CD pipeline, Makefile, golangci-lint, Dependabot, CODEOWNERS). Generates Gatekeeper-compliant apps with optional Azure infrastructure. |
| `scaffolds/python-service/` | ⬜ Pending | Copier template (not started) |
| `portal/` | ✅ Complete | Portal UI React app — Vite + React 18 + TypeScript + Tailwind CSS. 24 TypeScript files. API client with full Platform API integration (apps, infra, compliance, scaffold, health). Layout (Sidebar, Header, AppShell), routing (React Router 6.28), common components (Badge, LoadingSpinner, StatusCard). **Dashboard panels (3 of 6 complete):** Applications (#79) ✅ — Argo CD apps with sync status, health, last deployed. Infrastructure (#80) ✅ — Crossplane Claims with ready/synced status, connection secrets. Compliance Score (#81) ✅ — Donut chart (Recharts) with color-coded severity (green/amber/red), violations + vulnerabilities breakdown with severity badges, auto-refresh 30s. Remaining: Policy Violations (#82), Vulnerability Feed (#83), Security Events (#84), Scaffold form (#85). Multi-stage Dockerfile (Node 22 → Nginx 1.27-alpine). Security: non-root user, read-only rootfs, emptyDir volumes. TanStack Query for server state management. Bearer token authentication via `VITE_API_TOKEN`. Deployed at `portal.rdp.azurelaboratory.com`. |
| `api/` | ✅ Complete | Platform API — Go + Chi router, structured logging, graceful shutdown. Endpoints: scaffold (#51), Argo CD apps (#42, #43, #89), compliance (#48), infra full CRUD (#44, #45, #46, #47), Falco webhook (#49). Complete GitOps infrastructure management. RBAC configured. Secrets via ESO. Event store (in-memory, 1000 events) for Falco runtime security events. Argo CD integration requires one-time token bootstrap (see `platform/platform-api/setup-argocd-token.sh`). |
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
Trivy + Falco + Falcosidekick — CVE scanning + runtime security + event routing
Platform API (api/)         — Go + Chi; all CLI/UI operations go through here
Portal UI (portal/)         — React + TypeScript; browser-based dashboard (thin client over Platform API)
rdp CLI (cli/)              — Go + Cobra; terminal-based client (thin client over Platform API)
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
