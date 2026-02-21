# Scaffold Implementation Summary

## Overview

Implemented the POST `/api/v1/scaffold` endpoint (Task #51) to provide automated project scaffolding with full GitOps integration.

## Implementation Details

### Core Components

#### 1. Handler Package (`internal/scaffold/`)

- `types.go` — Request/response structures and Argo CD config schema
- `handler.go` — Main HTTP handler with validation, defaults, and workflow orchestration
- `github.go` — GitHub API operations (repo creation, platform config commits)
- `git.go` — Git CLI operations (init, commit, push)
- `README.md` — Comprehensive documentation

#### 2. Main API Integration

- Updated `main.go` to initialize and wire the scaffold handler
- Added `Config` fields for scaffold paths and work directory
- Connected handler to the `/api/v1/scaffold` route

#### 3. Dependencies

- `github.com/google/go-github/v66` — GitHub API client
- `golang.org/x/oauth2` — OAuth2 token authentication
- External: `git` CLI, `copier` CLI (Python)

#### 4. Docker Image

- Updated `Dockerfile` to install `git`, `python3`, `copier`
- Created directories for scaffold templates and work directory
- Non-root user (uid 1000) with proper permissions

#### 5. Kubernetes Deployment

- Added init container to clone scaffold templates from GitHub
- Mounted three volumes:
  - `scaffold-templates` — Read-only, populated by init container
  - `scaffold-work` — Writable, for temporary Copier output
  - `tmp` — Writable, for general temporary files
- Updated ConfigMap with scaffold configuration

## Workflow

```text
1. POST /api/v1/scaffold with template + project config
   ↓
2. Validate request (template exists, project name valid, etc.)
   ↓
3. Run Copier CLI to generate project files
   ↓
4. Create GitHub repository via API
   ↓
5. Initialize git repo, commit all files, push to GitHub
   ↓
6. Commit apps/<name>/config.json to platform repo
   ↓
7. Argo CD ApplicationSet discovers new app and deploys it
```

## API Contract

### Request Fields

**Core:**

- `template` — "go-service" or "python-service"
- `project_name` — Lowercase, hyphens only, 3-63 chars

**Go-specific:**

- `go_module_path`, `http_port`, `grpc_port`

**Features:**

- `enable_grpc`, `enable_database`, `enable_storage`, `enable_keyvault`

**Storage config (if enabled):**

- `storage_location`, `storage_replication`, `storage_public_access`, `storage_container_name`, `storage_connection_env`

**Vault config (if enabled):**

- `vault_location`, `vault_sku`, `vault_public_access`, `vault_connection_env`

**Metadata:**

- `team_name`, `team_email`, `owners`

**GitHub (optional overrides):**

- `github_org`, `github_repo`, `repo_private`

### Response Fields

- `success` — Boolean
- `message` — Human-readable result
- `repo_url` — HTTPS clone URL
- `repo_name` — Repository name
- `platform_config_path` — Path to config.json in platform repo
- `argocd_app_name` — Argo CD Application name

## Configuration

**Environment Variables:**

| Variable | Required | Default | Description |
| --- | --- | --- | --- |
| `GITHUB_TOKEN` | Yes | - | GitHub PAT (repo scope) |
| `GITHUB_ORG` | Yes | - | GitHub org for new repos |
| `PLATFORM_REPO` | Yes | `homelab-platform` | Platform repo name |
| `SCAFFOLD_TEMPLATES` | No | `/app/scaffolds` | Path to templates |
| `SCAFFOLD_WORK_DIR` | No | `/tmp/scaffold` | Temporary work dir |

**Kubernetes Secrets:**

The `GITHUB_TOKEN` is stored in `platform-api-secrets`. In production, this should come from ExternalSecrets (see task #87).

## Security Considerations

1. **GitHub token** — Sensitive credential, use ExternalSecrets in production
2. **Input validation** — Strict validation to prevent command injection
3. **Read-only rootfs** — All writable directories are explicitly mounted volumes
4. **Non-root user** — Both API pod and init container run as uid 1000
5. **Git URL authentication** — Uses token in HTTPS URL for push (cleaned from logs)

## Error Handling

**HTTP Status Codes:**

- `201 Created` — Success
- `400 Bad Request` — Invalid input
- `409 Conflict` — Repository already exists
- `500 Internal Server Error` — Execution failure

All errors logged with structured logging (slog).

## Testing Checklist

Before deploying:

- [ ] Build succeeds: `go build -o platform-api .`
- [ ] Docker image builds: `docker build -t platform-api:test .`
- [ ] Environment variables set in Secret/ConfigMap
- [ ] GitHub token has `repo` scope
- [ ] Scaffold templates exist at `/app/scaffolds` in pod
- [ ] Init container can clone platform repo
- [ ] Platform repo has ApplicationSet watching `apps/*/config.json`

## Integration Points

**Upstream Dependencies:**

- Copier templates (go-service, python-service) must exist
- GitHub API must be reachable
- Platform repo must exist and be writable

**Downstream Consumers:**

- Argo CD ApplicationSet watches `apps/*/config.json`
- Newly created repos are auto-deployed to the cluster
- Gatekeeper validates all generated manifests

## Future Enhancements

1. **Async scaffold execution** — Long-running operations should return immediately with a job ID
2. **Scaffold status endpoint** — GET /api/v1/scaffold/:id to check progress
3. **Template validation** — Pre-validate templates on startup
4. **Git commit signing** — Sign commits for auditability
5. **GitHub App integration** — Replace PAT with GitHub App for better security
6. **Webhook notifications** — Notify Slack/Teams on scaffold completion
7. **Template versioning** — Allow pinning template versions

## Files Changed

```text
homelab-platform/api/
├── main.go                              # ✅ Wired scaffold handler
├── go.mod                               # ✅ Added GitHub/OAuth2 deps
├── Dockerfile                           # ✅ Added git/copier
├── internal/scaffold/
│   ├── types.go                         # ✅ Request/response structs
│   ├── handler.go                       # ✅ Main handler logic
│   ├── github.go                        # ✅ GitHub API operations
│   ├── git.go                           # ✅ Git CLI operations
│   └── README.md                        # ✅ Documentation
└── SCAFFOLD_IMPLEMENTATION.md           # ✅ This file

homelab-platform/platform/platform-api/
├── deployment.yaml                      # ✅ Init container + volumes
└── configmap.yaml                       # ✅ Scaffold config vars
```

## Task Dependencies

**Completed prerequisites:**

- ✅ Task #41 — Platform API foundation (Go + Chi router)
- ✅ Task #54 — Kubernetes manifests (Deployment, Service, ConfigMap, Secret)
- ✅ Task #55 — go-service Copier template (all 23 files)

**Unblocked by completion:**

- Task #64 — Scaffold post-action (already implemented as part of #51)
- Task #72 — `rdp scaffold create` CLI command (can now call this API)
- Task #85 — Portal scaffold form (can now call this API)

**Related future work:**

- Task #63 — python-service scaffold template
- Task #87 — ExternalSecret for Platform API secrets (GitHub token)
