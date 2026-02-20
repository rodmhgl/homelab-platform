# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Overview

AKS Home Lab Internal Developer Platform (IDP) mono-repo.

| Directory | Status |
| --- | --- |
| `infra/` | ✅ Terraform — AKS, networking, ACR, bootstrap KV, managed identities |
| `platform/argocd/` | ✅ Phase A — Argo CD Helm values, self-manage Application, root App of Apps, Workload ApplicationSet, Projects, bootstrap.sh |
| `platform/crossplane/` | ✅ Phase B — Crossplane core Helm install (wave 1) |
| `platform/crossplane-providers/` | ✅ Phase B — DeploymentRuntimeConfig, Providers (family/storage/keyvault), function-patch-and-transform (wave 2) |
| `platform/crossplane-config/` | ✅ Phase B — ProviderConfig (OIDCTokenFile), XRDs (XStorageBucket/XKeyVault), Compositions (Pipeline mode) (wave 3) |
| `platform/gatekeeper/` | ✅ Phase C — Gatekeeper Helm install (wave 4) |
| `platform/gatekeeper-templates/` | ✅ Phase C — 8 ConstraintTemplates (wave 5) |
| `platform/gatekeeper-constraints/` | ✅ Phase C — 8 Constraints with enforcementAction: deny (wave 6) |
| `platform/external-secrets/` | ✅ Phase C — ESO Helm install (v0.11.0) + ClusterSecretStore (Workload Identity, wave 3.5). Platform API ExternalSecret resources deployed. Requires Terraform output placeholders. |
| `platform/trivy-operator/` | ✅ Phase C — Trivy Operator Helm install (v0.32.0) + values.yaml (wave 7). Continuous CVE scanning with VulnerabilityReport CRDs. |
| `platform/platform-api/` | ✅ Phase D — Platform API Deployment + Service + RBAC (wave 10). Secrets managed via ESO ExternalSecret (github-pat, openai-api-key, argocd-token). |
| `platform/falco/` | ✅ Phase C — Falco Helm install (v8.0.0, wave 8) + 4 custom rules. Modern eBPF driver. HTTP output to Falcosidekick. Runtime security monitoring for all namespaces except kube-system. |
| `platform/falcosidekick/` | ✅ Phase C — Falcosidekick Helm install (v0.10.0, wave 9). Webhook output to Platform API. ServiceMonitor enabled for Prometheus. |
| `platform/portal-ui/` | ✅ Phase E — Portal UI (React + TypeScript + Tailwind, wave 11). Vite build, Nginx runtime, TanStack Query for server state. API client layer complete. Layout + routing complete. Dashboard panels pending (#79-#84). |
| `platform/` (remaining) | ⬜ kagent, HolmesGPT |
| `scaffolds/go-service/` | ✅ Copier template — complete (23 template files: copier.yml, main.go, Dockerfile, k8s/, claims/, CI/CD, Makefile, supporting files) |
| `scaffolds/python-service/` | ⬜ Copier template (not started) |
| `portal/` | ✅ Portal UI React app — Vite + React 18 + TypeScript + Tailwind CSS. 22 TypeScript files. API client with full Platform API integration. Layout (Sidebar, Header, AppShell), routing, common components. Multi-stage Dockerfile (Node 22 → Nginx 1.27-alpine). Security: non-root, read-only rootfs, emptyDir volumes. Dashboard panels (#79-#84) + scaffold form (#85) pending. |
| `api/` | ✅ Platform API (Go + Chi) — scaffold (#51), Argo CD (#42, #43, #89), compliance (#48), infra complete CRUD (#44-#47), Falco webhook (#49). Full GitOps infrastructure management (list/get/create/delete) with three-layer validation. Secrets via ESO (#40, #87). RBAC configured. Event store for Falco runtime security events (in-memory, 1000 events). Argo CD integration complete — service account + RBAC via GitOps (values.yaml), token via one-time bootstrap script. |
| `cli/` | 🔨 rdp CLI (Go + Cobra) — Root command, config management, version, `rdp status` (#66), `rdp infra list/status` (#68) complete. Pending: interactive create/delete (#69-#71), apps (#67), compliance (#73), secrets (#74), investigate (#75), ask (#76). |

## Terraform (`infra/`)

**Runs in Terraform Cloud** — org `rnlabs`, workspace `aks-platform`. Push to `main` triggers an apply. There is no local `terraform apply` workflow; all applies go through TFC.

Required sensitive TFC variable: `subscription_id`.

TFC service principal needs `Owner` or `Contributor + User Access Administrator` on the subscription.

```bash
# Trigger apply
git push origin main

# After apply, get cluster credentials
az aks get-credentials --resource-group rg-homelab-aks-dev --name homelab-aks-dev --overwrite-existing

# Format check (run locally before committing)
terraform fmt -check -recursive infra/
```

**Provider versions:** Terraform >= 1.9.0, `azurerm ~> 4.60`, `azuread ~> 3.7`.

### Terraform ↔ Crossplane Responsibility Boundary

This is the critical architectural line:

- **Terraform manages:** foundational platform infra — AKS cluster, VNet, ACR, bootstrap Key Vault, Managed Identities, federated credentials
- **Crossplane manages:** app-level infra that developers consume — storage accounts, app Key Vaults, (future) PostgreSQL, Redis, Service Bus

Do not provision app-level resources in Terraform. Do not provision platform-level resources via Crossplane Claims.

### Identity & Auth Architecture

Zero static credentials — all pod auth via Workload Identity federation (OIDC):

| Identity | Subject | Permission |
| --- | --- | --- |
| `id-{cluster}-crossplane` | `system:serviceaccount:crossplane-system:provider-azure` | Contributor on subscription |
| `id-{cluster}-eso` | `system:serviceaccount:external-secrets:external-secrets` | Key Vault Secrets User on bootstrap KV |
| `id-{cluster}-cp` | AKS control plane | Network Contributor on subnet |

**Crossplane caveat:** Upbound providers generate ServiceAccounts with a hash suffix. After first Crossplane install, check the actual SA name and update `crossplane_service_account` in `variables.tf`, then re-apply.

### Key Outputs (consumed by platform layer)

| Output | Consumer |
| --- | --- |
| `crossplane_identity_client_id` | `DeploymentRuntimeConfig` annotation |
| `eso_identity_client_id` | ESO ServiceAccount annotation |
| `keyvault_uri` | ESO `ClusterSecretStore` spec |
| `oidc_issuer_url` | Future federated credentials |
| `acr_login_server` | Image references, Argo CD |

## Platform Layer (`platform/`)

Argo CD App of Apps pattern. Root app (`platform/argocd/root-app.yaml`) discovers all `platform/*/application.yaml` files.

### GitOps Principle

**Everything that CAN be declarative MUST be declarative and in Git:**

- ✅ Service account definitions (Argo CD values.yaml, not kubectl patches)
- ✅ RBAC policies (Argo CD values.yaml, not kubectl patches)
- ✅ ConfigMaps, Deployments, Services (YAML in platform/)
- ✅ ExternalSecret resources (structure in Git, values in Key Vault)

**Only imperative when impossible to be declarative:**

- ⚠️ Argo CD API tokens (generated via CLI after service account exists)
- ⚠️ Key Vault secret values (never in Git, stored in Azure Key Vault)

**Example:** The Argo CD `platform-api` service account is defined in `platform/argocd/values.yaml` (GitOps), but the token for that account is generated via `setup-argocd-token.sh` (one-time bootstrap) and stored in Key Vault.

### Gatekeeper — Three-Application Pattern (mandatory)

Gatekeeper requires three separate Argo CD Applications due to async CRD registration (same problem as Crossplane):

```text
gatekeeper           (wave 4) — Helm chart; installs core controller + webhook
gatekeeper-templates (wave 5) — ConstraintTemplates; controller registers CRDs asynchronously
gatekeeper-constraints (wave 6) — Constraints; SkipDryRunOnMissingResource=true
```

**Why three and not one or two:** ConstraintTemplates instruct the Gatekeeper controller to register new CRDs (one per template). Constraint objects reference those CRDs. If templates and constraints are in the same Application, Argo CD attempts both in a single sync pass — constraints fail because the CRDs haven't been registered yet. Splitting into separate Applications with inter-Application wave ordering ensures templates fully process before constraints are attempted.

**Rego syntax gotcha:** `contains` is a reserved built-in function in Rego 3.x — do NOT use it as a rule name. Use set comprehension syntax instead:

```rego
# Wrong (causes "var cannot be used for rule name" errors):
input_containers contains container { ... }

# Correct:
input_containers[container] { ... }
```

**8 ConstraintTemplates:**

- `k8srequiredlabels` — enforces ownership labels on Deployments
- `containerlimitsrequired` — CPU + memory limits mandatory
- `nolatesttag` — blocks `:latest` tag or untagged images
- `noprivilegedcontainers` — blocks `privileged: true`
- `allowedrepos` — images only from homelab ACR
- `requireprobes` — readiness + liveness probes mandatory
- `crossplaneclaimlocation` — restricts Claims to allowed Azure regions
- `crossplanenopublicaccess` — blocks `publicAccess: true` on Claims

### Crossplane — Three-Application Pattern (mandatory)

Crossplane requires three separate Argo CD Applications due to async CRD registration:

```text
crossplane          (wave 1) — Helm chart; installs pkg.crossplane.io + apiextensions.crossplane.io CRDs
crossplane-providers (wave 2) — DeploymentRuntimeConfig + Providers + Functions; waits for core CRDs
crossplane-config   (wave 3) — ProviderConfig + XRDs + Compositions; SkipDryRunOnMissingResource=true
```

**Why three and not one:** Provider pods register their own CRDs (azure.upbound.io/*, etc.) asynchronously after becoming `Healthy`. Argo CD has no visibility into CRD registration timing, so `crossplane-config` uses `SkipDryRunOnMissingResource=true` + `selfHeal` to retry until provider CRDs land.

**Known schema facts for Upbound Azure provider v1.9.0:**

- `installConditionFailurePolicy` does not exist in the Provider schema — omit it
- ProviderConfig credential source is `OIDCTokenFile` (not `InjectedIdentity` — renamed in v1.x)

Compositions use `function-patch-and-transform` in **Pipeline mode** — not the legacy `resources` mode.

**Composition transform syntax:**
- String transforms must include `type: FromConnectionSecretKey` for connection details
- For string sanitization, use `type: Convert` with `convert: ToLower` (avoid complex Regexp transforms)
- Storage account names are sanitized by lowercasing only (Azure accepts lowercase alphanumeric)

`ApplicationSet` generator watches `apps/*/config.json` in the platform repo to auto-onboard new scaffold repos.

### Falco — Runtime Security (wave 8)

**Chart version:** falcosecurity/falco 8.0.0 (Falco app version 0.43.0)

**Driver:** Modern eBPF (`modern_ebpf`) — CO-RE (Compile Once, Run Everywhere) with BTF. No kernel module compilation required. Works on AKS Ubuntu nodes (Kernel >= 5.15).

**Custom rules approach:**
- **Do NOT redefine** macros/lists from Falco's default rules (e.g., `shell_binaries`, `container`, `sensitive_files`)
- **Always use `homelab_` prefix** for custom macros to avoid naming conflicts
- Reference Falco's default macros where possible (e.g., `sensitive_files` instead of redefining it)
- Custom rules are defined **inline** in `values.yaml` via `customRules:` section (NOT separate ConfigMap)

**4 Custom Rules:**
1. **Unexpected Shell Spawned in Container** (WARNING) — detects shell execution in containers
2. **Sensitive File Read in Container** (ERROR) — monitors access to /etc/shadow, SSH keys, .kube/config
3. **Binary Written to Container Filesystem** (WARNING) — container drift detection
4. **Unexpected Network Connection from Container** (WARNING) — suspicious outbound ports (IRC, mining, Tor)

**Namespace filtering:** Monitors all namespaces **except kube-system**. This is intentionally broad ("start noisy, tune later"). The `homelab_monitored_namespace` macro can be refined later based on actual usage patterns.

**Priority threshold:** `notice` — all events at NOTICE level and above are captured. This includes both custom rules (WARNING/ERROR) and default Falco rules.

**Output configuration:** HTTP output enabled to Falcosidekick (`http://falcosidekick.falco.svc.cluster.local:2801`). gRPC output disabled due to TLS certificate requirements in Falco v8.0.0.

**Integration architecture:**
```
Falco (DaemonSet)
  → HTTP output
  → Falcosidekick (Deployment, wave 9)
  → Webhook (http://platform-api.platform/api/v1/webhooks/falco)
  → Platform API EventStore (in-memory, 1000 events)
  → GET /api/v1/compliance/events endpoint
```

**Common issues:**
- **Macro name conflicts:** If custom rules redefine Falco's default macros, the default rules will fail compilation with `LOAD_ERR_COMPILE_CONDITION` errors
- **Chart version compatibility:** Falco v8.0.0 has different schema than v4.x — `extraVolumes`/`extraVolumeMounts` are NOT supported at root level; use `customRules:` inline instead
- **gRPC vs HTTP:** Falco's gRPC server requires TLS certs that aren't auto-generated; HTTP output is simpler and works without cert configuration

### Falcosidekick — Event Routing (wave 9)

**Chart version:** falcosecurity/falcosidekick 0.10.0

**Purpose:** Routes Falco security events to external systems. Acts as the bridge between Falco and the Platform API.

**Configuration:**
- **Webhook output:** `http://platform-api.platform.svc.cluster.local/api/v1/webhooks/falco` (internal cluster traffic, no authentication)
- **Resource limits:** 200m CPU / 256Mi memory (homelab-sized)
- **ServiceMonitor:** Enabled for Prometheus metrics (events processed, outputs sent, errors)

**Key architectural decisions:**
- Service port 80 (not pod port 8080) — Falcosidekick connects via K8s Service
- No webhook authentication — internal cluster traffic only; future enhancement: HMAC signature validation
- Modular design — Falcosidekick can route to multiple outputs (Slack, PagerDuty) without touching Falco configuration

**Troubleshooting:**
- DNS name must match Service name (`platform-api.platform.svc.cluster.local`, not `platform-api.platform-api`)
- Falcosidekick logs show webhook delivery status (`POST OK (200)` or errors)
- Config updates require pod restart (Helm values don't trigger automatic rollout)

## Platform API (`api/`)

**Status:** Core endpoints implemented (scaffold, apps, compliance, infra management)

- **Language:** Go
- **Router:** Chi
- **Logging:** Structured logging with `slog`
- **Configuration:** Environment variables via `envconfig`
- **GitOps:** Infrastructure Claims committed to Git, not directly to cluster

**Implemented endpoints:**

- `GET /health`, `GET /ready` — Health checks
- `POST /api/v1/scaffold` — ✅ (#51) Copier template execution, GitHub repo creation, Argo CD onboarding
- `GET /api/v1/apps`, `GET /api/v1/apps/{name}`, `POST /api/v1/apps/{name}/sync` — ✅ (#42, #43) Argo CD app management
- `GET /api/v1/compliance/*` — ✅ (#48) Aggregated compliance view (Gatekeeper + Trivy + Falco)
- `GET /api/v1/infra`, `GET /api/v1/infra/storage`, `GET /api/v1/infra/vaults` — ✅ (#44) List Claims
- `GET /api/v1/infra/{kind}/{name}` — ✅ (#45) Crossplane resource tree query with events
- `POST /api/v1/infra` — ✅ (#46) Create Claim via GitOps (three-layer validation: request → Gatekeeper → GitHub)
- `POST /api/v1/webhooks/falco` — ✅ (#49) Falco event webhook receiver
- `GET /api/v1/compliance/events` — ✅ (#48) Query Falco security events with filtering

**Pending endpoints:**

- `/api/v1/secrets/*` — ExternalSecrets + connection secrets (#50)
- `/api/v1/investigate/*` — HolmesGPT integration (#52)
- `/api/v1/agent/ask` — kagent CRD-based interaction (#53)
- `/api/v1/webhooks/argocd` — Argo CD webhook (#49)

**Key architectural patterns:**

- GitOps for infrastructure: `/api/v1/infra` endpoints commit Claim YAML to app repos, not direct cluster mutations
- Falco integration: Events arrive at `POST /api/v1/webhooks/falco` via Falcosidekick, stored in EventStore (in-memory circular buffer, 1000 events), queryable via `GET /api/v1/compliance/events`
- Compliance scoring: Includes Falco events — Critical events × 15, Error events × 8 (heavier than CVEs because they indicate active threats vs potential vulnerabilities)
- kagent interaction is CRD-based: Platform API creates `Agent`/`Task` resources, not direct HTTP to an LLM

**Event storage notes:**
- EventStore is in-memory per-pod (not shared across replicas)
- Circular buffer drops oldest events when full (max 1000)
- For production: replace with shared persistence (PostgreSQL/Redis/etcd)
- Query filters: namespace, severity, rule name, timestamp (since), limit

## Portal UI (`portal/`)

**Status:** React scaffold complete (task #78); dashboard panels pending

- **Framework:** React 18 + TypeScript + Vite 6
- **Styling:** Tailwind CSS 3.4 with custom color palette
- **State:** TanStack Query 5.62 (server state), React hooks (local state)
- **Routing:** React Router 6.28 (SPA)
- **Charts:** Recharts 2.15 (for compliance donut, task #81)
- **Runtime:** Nginx 1.27-alpine (multi-stage Docker build)

**Architecture:**
- **API-first:** All data fetched from Platform API via TanStack Query
- **Build-time config:** `VITE_API_URL` baked into bundle (default: `http://platform-api.platform.svc.cluster.local`)
- **Security:** Non-root user (UID 1000), read-only rootfs, emptyDir volumes for `/var/cache/nginx` and `/tmp`
- **Deployment:** 2 replicas, wave 11 (after Platform API wave 10), ClusterIP Service port 80 → 8080

**Components implemented:**
- API client layer (9 files): `types.ts`, `client.ts`, endpoint modules (apps, infra, compliance, scaffold, health)
- Layout (3 files): `AppShell.tsx`, `Sidebar.tsx`, `Header.tsx` (with platform health indicator)
- Common components (3 files): `Badge.tsx`, `LoadingSpinner.tsx`, `StatusCard.tsx`
- Pages (6 files): `Dashboard.tsx`, `Applications.tsx`, `Infrastructure.tsx`, `Compliance.tsx`, `Scaffold.tsx`, `NotFound.tsx`

**Pending work:**
- Dashboard panels (#79-#84): Applications panel, Infrastructure panel, Compliance Score donut, Policy Violations table, Vulnerability Feed, Security Events timeline
- Scaffold form (#85): Interactive project creation with template selector, storage/vault toggles
- Detail pages: App detail, Infra detail, Compliance detail
- AI Ops panel (#86): kagent chat + HolmesGPT integration

**Access:**
```bash
kubectl port-forward -n platform svc/portal-ui 8080:80
# Open http://localhost:8080
```

## CLI (`cli/`)

**Status:** Foundation complete (root command + config management)

- **Framework:** Cobra + Viper
- **Config file:** `~/.rdp/config.yaml` (three-tier precedence: flags > env > file)
- **Next:** Implement subcommands that call Platform API endpoints

## Scaffolds (`scaffolds/`)

Uses **Copier** (not Cookiecutter) — Copier supports template updates that propagate to existing projects.

**go-service scaffold status:** ✅ Complete (23 template files ready for production use).

**Copier validator syntax:** Use Jinja2-native filters — `|length`, `|lower`, `|regex_search()` — NOT Python built-ins like `len()`, `.islower()`, `.isalnum()`. Copier runs validators in a restricted Jinja2 environment without Python built-ins available.

Storage account naming rule: `st{claimname}` — lowercase, strip hyphens/dots/underscores to meet Azure constraints.

## CIDR Layout

```text
VNet:         10.10.0.0/16
AKS nodes:    10.10.0.0/22   (drawn from VNet)
Pod overlay:  192.168.0.0/16 (Cilium; not in VNet)
Services:     172.16.0.0/16  (not in VNet)
kube-dns:     172.16.0.10
```
