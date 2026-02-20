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
| `platform/platform-api/` | ✅ Phase D — Platform API Deployment + Service + RBAC (wave 10) |
| `platform/` (remaining) | ⬜ ESO, Trivy, Falco, monitoring, kagent, HolmesGPT |
| `scaffolds/go-service/` | ✅ Copier template — complete (23 template files: copier.yml, main.go, Dockerfile, k8s/, claims/, CI/CD, Makefile, supporting files) |
| `scaffolds/python-service/` | ⬜ Copier template (not started) |
| `api/` | ✅ Platform API (Go + Chi) — scaffold endpoint implemented (task #51) |
| `cli/` | 🔨 rdp CLI (Go + Cobra) — Cobra root command + Viper config management complete |

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

`ApplicationSet` generator watches `apps/*/config.json` in the platform repo to auto-onboard new scaffold repos.

## Platform API (`api/`)

**Status:** Foundation complete, scaffold endpoint implemented (task #51)

- **Language:** Go
- **Router:** Chi
- **Logging:** Structured logging with `slog`
- **Configuration:** Environment variables via `envconfig`

**Implemented endpoints:**

- `GET /health`, `GET /ready` — Health checks
- `POST /api/v1/scaffold` — ✅ Implemented (task #51)
  - Runs Copier to generate project from template
  - Creates GitHub repository
  - Initializes git, commits, and pushes to new repo
  - Commits `apps/<name>/config.json` to platform repo for Argo CD discovery
  - See `api/internal/scaffold/README.md` for full documentation

**Pending endpoints:**

- `/api/v1/apps/*` — Argo CD application management
- `/api/v1/infra/*` — Crossplane Claim management (commits YAML to Git, not direct cluster mutations)
- `/api/v1/compliance/*` — Gatekeeper + Trivy + Falco aggregation
- `/api/v1/secrets/*` — ExternalSecrets + connection secrets
- `/api/v1/investigate/*` — HolmesGPT integration
- `/api/v1/agent/ask` — kagent CRD-based interaction
- `/api/v1/webhooks/*` — Falco and Argo CD webhooks

**Key architectural patterns:**

- GitOps for infrastructure: `/api/v1/infra` endpoints commit Claim YAML to app repos, not direct cluster mutations
- Falco events arrive at `POST /api/v1/webhooks/falco` via Falcosidekick
- kagent interaction is CRD-based: Platform API creates `Agent`/`Task` resources, not direct HTTP to an LLM

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
