# Platform API

The Platform API is the central nervous system of the AKS Home Lab Internal Developer Platform (IDP). All developer interactions—whether through the `rdp` CLI or the Portal UI—flow through this API.

## Overview

This API provides a unified interface for:

- **Scaffolding** — Create new Go/Python services with batteries included
- **Application Management** — View, sync, and monitor Argo CD applications
- **Infrastructure Provisioning** — Self-service Crossplane Claims (Storage, Key Vaults)
- **Compliance Aggregation** — Unified view of Gatekeeper policies, Trivy CVEs, Falco events
- **Secrets Management** — List ExternalSecrets and Crossplane connection secrets
- **AI Operations** — Natural language cluster interaction (kagent) and root cause analysis (HolmesGPT)

## Architecture

- **Router:** Chi (lightweight, stdlib-compatible)
- **Logging:** Structured logging with `slog` (JSON format)
- **Configuration:** Environment variables via `envconfig`
- **Authentication:** Bearer token (validated against K8s ServiceAccount token or future auth provider)
- **GitOps:** Infrastructure changes commit YAML to Git; Argo CD syncs

## Running Locally

```bash
# Install dependencies
make deps

# Set required environment variables
export ARGOCD_SERVER_URL="https://argocd.example.com"
export ARGOCD_TOKEN="your-token"
export GITHUB_TOKEN="your-github-token"
export GITHUB_ORG="your-org"
export IN_CLUSTER=false
export KUBECONFIG=~/.kube/config

# Run the service
make run

# Or use live reload (requires air: go install github.com/cosmtrek/air@latest)
make dev
```

## Building

```bash
# Build binary
make build

# Build Docker image
make docker-build
```

## Testing

```bash
# Run tests
make test

# Run tests with coverage report
make test-coverage

# Run linter
make lint
```

## Configuration

All configuration is via environment variables:

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `PORT` | No | `8080` | HTTP server port |
| `LOG_LEVEL` | No | `info` | Log level (debug, info, warn, error) |
| `SHUTDOWN_TIMEOUT` | No | `30` | Graceful shutdown timeout (seconds) |
| `IN_CLUSTER` | No | `true` | Use in-cluster K8s client config |
| `KUBECONFIG` | No | — | Path to kubeconfig (if `IN_CLUSTER=false`) |
| `ARGOCD_SERVER_URL` | Yes | — | Argo CD API server URL |
| `ARGOCD_TOKEN` | Yes | — | Argo CD API token |
| `GITHUB_TOKEN` | Yes | — | GitHub API token (for GitOps commits) |
| `GITHUB_ORG` | Yes | — | GitHub organization name |
| `PLATFORM_REPO` | No | `homelab-platform` | Platform repo name (for config.json commits) |
| `OPENAI_API_KEY` | No | — | OpenAI API key (for AI features) |
| `HOLMESGPT_URL` | No | — | HolmesGPT service URL |
| `KAGENT_NAMESPACE` | No | `kagent-system` | Namespace for kagent CRDs |

## API Endpoints

### Health & Readiness

- `GET /health` — Service health (always 200 OK if running)
- `GET /ready` — Service readiness (200 OK when dependencies are available)

### Scaffolding

- `POST /api/v1/scaffold` — Create new service from template

### Applications

- `GET /api/v1/apps` — List all Argo CD applications
- `GET /api/v1/apps/{name}` — Get application details
- `POST /api/v1/apps/{name}/sync` — Trigger application sync

### Infrastructure

- `GET /api/v1/infra` — ⬜ List all Crossplane Claims
- `GET /api/v1/infra/storage` — ⬜ List StorageBucket Claims
- `GET /api/v1/infra/vaults` — ⬜ List Vault Claims
- `GET /api/v1/infra/{kind}/{name}` — ✅ Get Claim details + composed resource tree + events (supports ?namespace=)
- `POST /api/v1/infra` — ⬜ Create Claim (commits YAML to app repo)
- `DELETE /api/v1/infra/{kind}/{name}` — ⬜ Delete Claim (commits removal to app repo)

### Compliance

- `GET /api/v1/compliance/summary` — ✅ Aggregated compliance score (0-100) + violation/vulnerability counts
- `GET /api/v1/compliance/policies` — ✅ Gatekeeper ConstraintTemplates (8 policies)
- `GET /api/v1/compliance/violations` — ✅ Gatekeeper audit violations (supports filtering: ?namespace=, ?kind=, ?constraint=)
- `GET /api/v1/compliance/vulnerabilities` — ✅ Trivy CVE reports (supports filtering: ?namespace=, ?severity=, ?image=)
- `GET /api/v1/compliance/events` — ⬜ Falco security events (placeholder; awaiting Falco deployment)

### Secrets

- `GET /api/v1/secrets/{namespace}` — List ExternalSecrets + connection secrets

### Investigation

- `POST /api/v1/investigate` — Trigger HolmesGPT investigation
- `GET /api/v1/investigate/{id}` — Get investigation results

### AI Agent

- `POST /api/v1/agent/ask` — Ask natural language question (creates kagent Task)

### Webhooks

- `POST /api/v1/webhooks/falco` — Falcosidekick webhook (security events)
- `POST /api/v1/webhooks/argocd` — Argo CD webhook (sync events)

## Package Structure

### `internal/compliance/`

Compliance aggregation endpoints — queries Gatekeeper Constraints and Trivy VulnerabilityReports to provide unified compliance view.

**Files:**
- `handler.go` — HTTP handlers for 5 endpoints
- `client.go` — Kubernetes dynamic client wrapper
- `types.go` — Request/response DTOs
- `README.md` — Full package documentation

**Key Features:**
- Queries all 8 deployed Gatekeeper constraint kinds
- Aggregates Trivy CVE scans from workload namespaces (excludes platform namespaces)
- Calculates compliance score: `max(0, 100 - (violations × 5) - (critical_cves × 10) - (high_cves × 5))`
- Supports query filtering (namespace, severity, kind, constraint, image)
- Graceful degradation when CRDs are missing

See `internal/compliance/README.md` for detailed documentation.

### `internal/argocd/`

Argo CD application management — wraps Argo CD API for listing, viewing, and syncing applications.

**Files:**
- `handler.go` — HTTP handlers
- `client.go` — Argo CD HTTP client wrapper
- `types.go` — Request/response DTOs

### `internal/scaffold/`

Project scaffolding — runs Copier templates, creates GitHub repos, onboards to Argo CD.

**Files:**
- `handler.go` — HTTP handler
- `copier.go` — Copier template execution
- `github.go` — GitHub API client
- `git.go` — Git operations
- `types.go` — Request/response DTOs
- `README.md` — Full package documentation

### `internal/infra/`

Infrastructure management endpoints — queries Crossplane Claims, Composites, and Managed Resources to provide full resource tree visibility.

**Files:**
- `handler.go` — HTTP handler for resource tree queries
- `client.go` — Kubernetes dynamic client wrapper with GVR mappings
- `types.go` — Request/response DTOs
- `README.md` — Full package documentation

**Key Features:**
- Traverses complete Crossplane resource tree: Claim → Composite → Managed Resources
- Retrieves Kubernetes Events for all resources in the tree (essential for debugging provisioning failures)
- Supports Claims in any namespace via `?namespace=` query parameter
- Determines resource status from Crossplane conditions (Ready, Synced)
- Returns Azure resource names via `externalName` field

See `internal/infra/README.md` for detailed documentation.

## Development

This service is built following these patterns:

1. **API-first:** The CLI and Portal are thin clients; all business logic lives here
2. **GitOps for infrastructure:** Claims are committed to Git, not applied directly
3. **Structured logging:** All logs in JSON format with request IDs
4. **Graceful shutdown:** Waits for in-flight requests before terminating
5. **No auth shortcuts:** Even in development, Bearer token is required (except health/ready)

## Deployment

Kubernetes manifests are in the platform repository at `homelab-platform/platform/platform-api/`:

- `deployment.yaml` — Deployment with resource limits, liveness/readiness probes
- `service.yaml` — ClusterIP Service (port 80 → 8080)
- `application.yaml` — Argo CD Application manifest

The API runs in the `platform` namespace with a ServiceAccount that has appropriate RBAC for:
- Crossplane Claims (read, list, watch, create, delete)
- Argo CD Applications (read, list via API)
- Gatekeeper Constraints & ConstraintTemplates (read, list)
- Trivy VulnerabilityReports (read, list)
- ExternalSecrets (read, list)
- kagent Tasks (create, watch)
- Falco events (receive webhooks)
