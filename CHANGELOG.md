# Changelog

All notable changes to the homelab-platform project.

## [Unreleased]

### Added

**2026-02-20: CLI Infrastructure Commands**

- ✅ **rdp infra list Command** (`cli/cmd/infra.go`)
  - Lists all Crossplane Claims or filters by type (storage/vaults)
  - Tabular output with name, namespace, kind, status, ready/synced flags, age, connection secret
  - Supports `--namespace` flag to filter by namespace
  - Supports `--json` flag for JSON output
  - Calls `GET /api/v1/infra`, `GET /api/v1/infra/storage`, or `GET /api/v1/infra/vaults`
  - Examples: `rdp infra list`, `rdp infra list storage --namespace production`

- ✅ **rdp infra status Command** (`cli/cmd/infra.go`)
  - Shows detailed status for a specific Crossplane Claim
  - Displays Claim details, Composite resource, Managed Azure resources, recent Kubernetes events
  - Supports `--namespace` flag (defaults to "default")
  - Supports `--json` flag for JSON output
  - Calls `GET /api/v1/infra/:kind/:name?namespace=<ns>`
  - Examples: `rdp infra status storage my-bucket`, `rdp infra status vault my-vault --namespace production`

- ✅ **Documentation** (`cli/README.md`)
  - Updated with infrastructure command examples
  - Usage patterns for list and status commands
  - Flag descriptions and JSON output examples
  - Completes task #68 — developers can now view infrastructure Claims via CLI

**2026-02-20: DELETE Infrastructure Endpoint**

- ✅ **DELETE /api/v1/infra/:kind/:name Endpoint** (`api/internal/infra/handler.go`)
  - Implements GitOps deletion pattern for Crossplane Claims
  - Removes Claim YAML from app repository, triggering Argo CD reconciliation
  - Verifies Claim existence before deletion (warns if missing, continues anyway)
  - Supports both StorageBucket and Vault Claims with kind normalization
  - Request validation: requires repoOwner and repoName in body
  - Returns commit SHA, file path, and repo URL in response
  - Completes task #47 — full CRUD operations for infrastructure management

- ✅ **GitHub Delete Operation** (`api/internal/infra/github.go`)
  - New `DeleteClaim()` method using GitHub Contents API
  - Retrieves file SHA before deletion (required by GitHub API)
  - Handles 404 errors gracefully with clear error messages
  - Follows same commit pattern as CREATE operations
  - Comprehensive logging for audit trail

- ✅ **Request/Response Types** (`api/internal/infra/types.go`)
  - `DeleteClaimRequest`: repoOwner, repoName fields
  - `DeleteClaimResponse`: success, message, kind, name, namespace, commitSHA, filePath, repoURL
  - JSON marshaling/unmarshaling validation in tests

- ✅ **Tests** (`api/internal/infra/delete_test.go`)
  - JSON marshaling/unmarshaling tests for request and response types
  - Request validation tests (missing fields, empty requests)
  - All tests passing with comprehensive coverage

- ✅ **Documentation** (`api/docs/DELETE_INFRA.md`)
  - Complete endpoint reference with examples
  - GitOps workflow diagram
  - Security considerations and error handling
  - cURL examples for both StorageBucket and Vault deletions
  - Idempotency notes and related endpoints

**2026-02-20: Argo CD Integration Fix (GitOps Refactoring)**

- ✅ **Argo CD Service Account GitOps Configuration** (`platform/argocd/values.yaml`)
  - Moved service account and RBAC configuration from imperative kubectl patches to declarative Helm values
  - Added `accounts.platform-api: apiKey` to `configs.cm` section
  - Added RBAC policies for platform-api role with full application management permissions
  - **GitOps principle:** Service account and RBAC are now version-controlled and synced via Argo CD
  - Completes the GitOps vision — only token generation remains imperative (cannot be in Git)

- ✅ **Platform API ConfigMap Fix** (`platform/platform-api/configmap.yaml`)
  - Fixed `ARGOCD_SERVER_URL` from `argocd-server.argocd.svc.cluster.local` to `http://argocd-server.argocd.svc.cluster.local`
  - Added clarifying comment about using HTTP for internal cluster communication
  - Argo CD server exposes port 80 for HTTP internally (no TLS needed)

- ✅ **Argo CD Token Bootstrap Script** (`platform/platform-api/setup-argocd-token.sh`)
  - ONE-TIME bootstrap script for token generation and Key Vault storage
  - Removes imperative ConfigMap patches (now in values.yaml)
  - Validates service account exists before attempting token generation
  - Supports `--kv-name` argument for explicit Key Vault specification (TFC-friendly)
  - Auto-detects Key Vault name from Terraform outputs (local state fallback)
  - Verifies ExternalSecret sync and Platform API health after token storage
  - Comprehensive error handling with troubleshooting guidance

- ✅ **Documentation** (`platform/platform-api/ARGOCD_TOKEN_SETUP.md`)
  - Complete setup guide with clear GitOps vs. Imperative separation
  - Step-by-step instructions for token generation and verification
  - Troubleshooting section for common issues
  - Security considerations (token rotation, RBAC scoping)
  - Completes task #89 — `/api/v1/apps` endpoint now functional with proper token configuration

**2026-02-20: CLI Status Command**

- ✅ **rdp status Command** (`cli/cmd/status.go`)
  - Comprehensive platform health dashboard aggregating multiple API endpoints
  - Displays: API health/readiness, compliance score, application health, infrastructure Claims count
  - Graceful degradation: shows available data even when individual endpoints fail
  - Professional formatting with Unicode box-drawing characters and status icons (✓, ✗, ⚠)
  - Three-tier configuration: flags > environment variables > config file (~/.rdp/config.yaml)
  - Error handling fix: main.go now properly displays errors to stderr before exit
  - Documentation: Updated cli/README.md with usage examples and output format
  - Completes task #66 — first operational CLI command beyond config management
  - **Known issue:** Applications section shows error due to Argo CD API configuration (tracked in #89)

**2026-02-20: Infrastructure Create Endpoint (GitOps)**

- ✅ **Infrastructure Create API** (`POST /api/v1/infra`)
  - GitOps-based Claim creation — commits YAML to app repo via GitHub API, not directly to cluster
  - New files: `github.go` (GitHub API client), `validation.go` (three-layer validation), `templates.go` (YAML generation)
  - Three-layer validation: Request structure → Gatekeeper constraints → GitHub API
  - Mirrors Gatekeeper constraints client-side: location (southcentralus, eastus2), no publicAccess: true
  - Smart defaults: location: southcentralus, tier: Standard, redundancy: LRS, skuName: standard, retention: 7 days
  - Go text/template YAML generation with automatic Gatekeeper-required label injection
  - Comprehensive test suite: `validation_test.go` (4 test functions, 27 test cases, 100% pass rate)
  - Live testing: 7/7 tests passed with real GitHub commits (see `api/TEST_RESULTS.md`)
  - Documentation: `api/docs/infra-create-endpoint.md`, `api/examples/create-claim.sh`
  - Completes task #46 — **unblocks Act 5 demo** (Self-Service Infrastructure)
  - **Unblocks downstream tasks:** #69 (rdp infra create storage), #70 (rdp infra create vault), #68 (rdp infra list CLI)

**2026-02-20: Infrastructure List Endpoints**

- ✅ **Infrastructure List API** (`GET /api/v1/infra`, `GET /api/v1/infra/storage`, `GET /api/v1/infra/vaults`)
  - New client methods: `ListClaims()` and `ListAllClaims()` using Kubernetes dynamic client
  - Three new handler methods: `HandleListAllClaims`, `HandleListStorageClaims`, `HandleListVaultClaims`
  - New response type: `ListClaimsResponse` with `ClaimSummary` for lightweight list views
  - Lists Claims across all namespaces using client-go `List()` operations
  - Returns status, ready/synced conditions, connection secret name, and labels
  - Completes task #44 — all infrastructure list endpoints now functional

**2026-02-20: Infrastructure Query Endpoint & Crossplane Composition Fixes**

- ✅ **Infrastructure Query API** (`GET /api/v1/infra/{kind}/{name}`)
  - New package: `api/internal/infra/` with handler, client, and types
  - Traverses complete Crossplane resource tree: Claim → Composite → Managed Resources
  - Retrieves Kubernetes Events for all resources in the tree
  - Returns status derived from Crossplane conditions (Ready, Synced)
  - Supports namespace filtering via `?namespace=` query parameter
  - Documentation: `api/internal/infra/README.md`

- ✅ **Platform API RBAC Fix**
  - Updated `platform/platform-api/rbac.yaml` to use correct API group: `platform.example.com`
  - Fixed client code GVR mappings to match deployed XRDs
  - Platform API ServiceAccount now has correct permissions for Claims and Composites

- ✅ **Implementation Notes Documentation**
  - Created `IMPLEMENTATION_NOTES.md` with detailed bug fixes and lessons learned
  - Documented Crossplane Composition bugs and fixes
  - Included verification commands and debugging tips

### Fixed

**2026-02-20: Crossplane Composition Bug Fixes**

- 🐛 **Regexp Transform Bug** (`platform/crossplane-config/compositions/storagebucket-azure.yaml`)
  - **Problem:** Invalid `replace` field in Regexp transforms caused composition errors
  - **Fix:** Simplified to use only `Convert: ToLower` transform
  - **Impact:** Storage account names are now properly sanitized

- 🐛 **Connection Detail Type Missing** (`platform/crossplane-config/compositions/storagebucket-azure.yaml`)
  - **Problem:** Connection details lacked required `type` field
  - **Fix:** Added `type: FromConnectionSecretKey` to all connection details
  - **Impact:** Crossplane can now properly propagate connection secrets

### Changed

**2026-02-20: Documentation Updates**

- Updated `README.md` — Platform API status reflects completed infrastructure endpoint
- Updated `PLATFORM_DESIGN.md` — API endpoint status table with implementation progress
- Updated `IMPLEMENTATION_PLAN.md` — Task #6.5 marked as complete
- Updated `homelab-platform/README.md` — API status updated
- Updated `homelab-platform/CLAUDE.md` — Composition syntax notes added
- Updated `CLAUDE.md` — Repository status reflects infra query endpoint
- Updated `api/README.md` — Added `internal/infra/` package documentation

---

## Progress Summary

### Completed Components

**Platform Infrastructure:**
- ✅ Terraform (AKS, networking, ACR, bootstrap Key Vault)
- ✅ Argo CD (GitOps control plane, App of Apps)
- ✅ Crossplane (core, providers, XRDs, compositions)
- ✅ Gatekeeper (8 ConstraintTemplates + 8 Constraints)
- ✅ External Secrets Operator (with bootstrap Key Vault)
- ✅ Trivy Operator (CVE scanning)
- ✅ kube-prometheus-stack (monitoring)
- ✅ Platform API Deployment + RBAC

**Platform API Endpoints:**
- ✅ Scaffold (`POST /api/v1/scaffold`)
- ✅ Argo CD Apps (`GET /api/v1/apps`, `GET /api/v1/apps/{name}`, `POST /api/v1/apps/{name}/sync`) — requires one-time token bootstrap
- ✅ Compliance (`GET /api/v1/compliance/summary|policies|violations|vulnerabilities`)
- ✅ Infrastructure List (`GET /api/v1/infra`, `GET /api/v1/infra/storage`, `GET /api/v1/infra/vaults`)
- ✅ Infrastructure Query (`GET /api/v1/infra/{kind}/{name}`)
- ✅ Infrastructure Create (`POST /api/v1/infra`)
- ✅ Infrastructure Delete (`DELETE /api/v1/infra/{kind}/{name}`)

**Scaffolds:**
- ✅ go-service (23 production-ready template files)

**CLI:**
- ✅ Root command + config management
- ✅ `rdp status` — platform health summary
- ✅ `rdp infra list/status` — view infrastructure Claims

### Pending Components

**Platform Infrastructure:**
- ⬜ Falco + Falcosidekick
- ⬜ kagent
- ⬜ HolmesGPT

**Platform API Endpoints:**
- ⬜ Secrets (`GET /api/v1/secrets/{namespace}`)
- ⬜ Investigation (`POST /api/v1/investigate`, `GET /api/v1/investigate/{id}`)
- ⬜ AI Agent (`POST /api/v1/agent/ask`)
- ⬜ Webhooks (`POST /api/v1/webhooks/falco`, `POST /api/v1/webhooks/argocd`)

**Scaffolds:**
- ⬜ python-service

**CLI:**
- ⬜ `rdp infra create storage/vault` — interactive prompts
- ⬜ `rdp infra delete` — interactive deletion
- ⬜ `rdp apps list/status/sync/logs` — Argo CD management
- ⬜ `rdp compliance summary/policies/vulns/events` — compliance views
- ⬜ `rdp secrets list/create` — secrets management
- ⬜ `rdp investigate` — HolmesGPT integration
- ⬜ `rdp ask` — kagent natural language
- ⬜ `rdp scaffold create` — interactive scaffolding

**Portal UI:**
- ⬜ React SPA (not started)

---

## Key Architectural Decisions

### API Group: `platform.example.com`

All Crossplane XRDs, Claims, and Composites use the `platform.example.com` API group. This is configured in:
- XRD `spec.group` field
- RBAC ClusterRole rules
- Client code GVR mappings

### Crossplane Composition Transform Syntax

**Connection Details:**
- Must include `type: FromConnectionSecretKey` field
- Example:
  ```yaml
  connectionDetails:
    - name: primaryAccessKey
      type: FromConnectionSecretKey
      fromConnectionSecretKey: attribute.primary_access_key
  ```

**String Transforms:**
- Prefer simple transforms (`Convert: ToLower`) over complex Regexp patterns
- Azure storage account names: lowercase only (no special character removal needed)

### Infrastructure Query Pattern

The `/api/v1/infra/{kind}/{name}` endpoint:
- Does NOT create or modify resources
- Provides read-only visibility into Crossplane resource trees
- Essential for debugging provisioning issues via Kubernetes Events
- Supports Claims in any namespace via `?namespace=` parameter

### GitOps Contract

**Infrastructure Mutation (API endpoints):**

Infrastructure mutation endpoints (create/delete):
- Commit Claim YAML to app Git repositories
- NOT apply resources directly to the cluster
- Rely on Argo CD to sync from Git
- Maintain Git as the single source of truth

**Platform Configuration (Argo CD, Crossplane, etc.):**

Everything that CAN be declarative MUST be in Git:
- ✅ Service account definitions (Helm values, not kubectl patches)
- ✅ RBAC policies (Helm values, not kubectl patches)
- ✅ Deployments, Services, ConfigMaps (YAML manifests)
- ✅ ExternalSecret resources (structure in Git, values in Key Vault)

Only imperative when impossible to be declarative:
- ⚠️ API tokens (generated after service accounts exist)
- ⚠️ Secret values (never in Git, stored in Azure Key Vault)

**Example:** Argo CD's `platform-api` service account is defined in `platform/argocd/values.yaml` (GitOps), but the API token is generated via `setup-argocd-token.sh` (one-time bootstrap).

---

## Next Steps

**Immediate Priority:** Task #46 — `POST /api/v1/infra` (Create Claim via GitOps)

This will complete the core infrastructure provisioning story by enabling developers to create Claims through the API, which commits them to Git for Argo CD to sync. With the list endpoints now complete, developers can:
1. List all Claims (`GET /api/v1/infra`)
2. View detailed Claim status (`GET /api/v1/infra/{kind}/{name}`)
3. *(Next)* Create new Claims via GitOps (`POST /api/v1/infra`)
4. *(Future)* Delete Claims via GitOps (`DELETE /api/v1/infra/{kind}/{name}`)
