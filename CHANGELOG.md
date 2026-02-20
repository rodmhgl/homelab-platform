# Changelog

All notable changes to the homelab-platform project.

## [Unreleased]

### Added

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
- ✅ Argo CD Apps (`GET /api/v1/apps`, `GET /api/v1/apps/{name}`, `POST /api/v1/apps/{name}/sync`)
- ✅ Compliance (`GET /api/v1/compliance/summary|policies|violations|vulnerabilities`)
- ✅ Infrastructure Query (`GET /api/v1/infra/{kind}/{name}`)

**Scaffolds:**
- ✅ go-service (23 production-ready template files)

**CLI:**
- ✅ Root command + config management

### Pending Components

**Platform Infrastructure:**
- ⬜ Falco + Falcosidekick
- ⬜ kagent
- ⬜ HolmesGPT

**Platform API Endpoints:**
- ⬜ Infrastructure List (`GET /api/v1/infra`, `GET /api/v1/infra/storage`, `GET /api/v1/infra/vaults`)
- ⬜ Infrastructure Create/Delete (`POST /api/v1/infra`, `DELETE /api/v1/infra/{kind}/{name}`)
- ⬜ Secrets (`GET /api/v1/secrets/{namespace}`)
- ⬜ Investigation (`POST /api/v1/investigate`, `GET /api/v1/investigate/{id}`)
- ⬜ AI Agent (`POST /api/v1/agent/ask`)
- ⬜ Webhooks (`POST /api/v1/webhooks/falco`, `POST /api/v1/webhooks/argocd`)

**Scaffolds:**
- ⬜ python-service

**CLI:**
- ⬜ All subcommands (apps, infra, compliance, secrets, investigate, ask)

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

Infrastructure mutation endpoints (create/delete) will:
- Commit Claim YAML to app Git repositories
- NOT apply resources directly to the cluster
- Rely on Argo CD to sync from Git
- Maintain Git as the single source of truth

---

## Next Steps

**Immediate Priority:** Task #46 — `POST /api/v1/infra` (Create Claim via GitOps)

This will complete the infrastructure provisioning story by enabling developers to create Claims through the API, which commits them to Git for Argo CD to sync.
