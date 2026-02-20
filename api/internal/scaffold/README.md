# Scaffold Service

The scaffold service implements the POST `/api/v1/scaffold` endpoint, which provides automated project scaffolding with full GitOps integration.

## Architecture

The scaffold workflow consists of five sequential steps:

1. **Copier Execution** — Generates project files from a template using the Copier CLI
2. **GitHub Repository Creation** — Creates a new repository via the GitHub API
3. **Git Initialization** — Initializes git, commits all files, and pushes to the new repo
4. **Platform Config Commit** — Commits `apps/<name>/config.json` to the platform repo
5. **Argo CD Discovery** — The ApplicationSet watches `apps/*/config.json` and auto-deploys the new service

## API Contract

### Request

```json
{
  "template": "go-service",
  "project_name": "user-api",
  "project_description": "User management API",

  "go_module_path": "github.com/rodmhgl/user-api",
  "http_port": 8080,
  "grpc_port": 9090,

  "enable_grpc": true,
  "enable_database": false,
  "enable_storage": true,
  "enable_keyvault": true,

  "storage_location": "southcentralus",
  "storage_replication": "LRS",
  "storage_public_access": false,
  "storage_container_name": "data",
  "storage_connection_env": "STORAGE_CONNECTION_STRING",

  "vault_location": "southcentralus",
  "vault_sku": "standard",
  "vault_public_access": false,
  "vault_connection_env": "KEYVAULT_URI",

  "team_name": "platform",
  "team_email": "platform@example.com",
  "owners": "@rodmhgl",

  "github_org": "rodmhgl",
  "github_repo": "user-api",
  "repo_private": false
}
```

### Response

```json
{
  "success": true,
  "message": "Successfully scaffolded user-api from go-service template",
  "repo_url": "https://github.com/rodmhgl/user-api.git",
  "repo_name": "user-api",
  "platform_config_path": "apps/user-api/config.json",
  "argocd_app_name": "user-api"
}
```

## Configuration

Environment variables:

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `GITHUB_TOKEN` | Yes | - | GitHub personal access token (repo scope) |
| `GITHUB_ORG` | Yes | - | GitHub organization/user for new repos |
| `PLATFORM_REPO` | Yes | `homelab-platform` | Platform repo for config commits |
| `SCAFFOLD_TEMPLATES` | No | `/app/scaffolds` | Path to Copier templates |
| `SCAFFOLD_WORK_DIR` | No | `/tmp/scaffold` | Temporary directory for scaffold operations |

## Deployment Considerations

### Scaffold Templates Volume

The scaffold templates must be available at the path specified by `SCAFFOLD_TEMPLATES`. In production, mount the templates via:

1. **Git clone sidecar** (recommended) — Init container that clones the platform repo
2. **ConfigMap** — For small templates (watch ConfigMap size limits)
3. **Persistent Volume** — Shared volume with the platform repo

Example init container (add to deployment.yaml):

```yaml
initContainers:
  - name: clone-templates
    image: alpine/git:latest
    command:
      - sh
      - -c
      - |
        git clone --depth 1 https://github.com/rodmhgl/homelab-platform.git /templates
        cp -r /templates/scaffolds/* /app/scaffolds/
    volumeMounts:
      - name: scaffold-templates
        mountPath: /app/scaffolds
```

### Permissions

The Platform API needs:

1. **GitHub token** with `repo` scope to create repositories and commit to the platform repo
2. **Write access** to `/tmp/scaffold` for Copier output
3. **Git CLI** installed in the container image
4. **Python 3 + Copier** installed (`pip install copier`)

### Security Notes

- The GitHub token is sensitive — store in a Kubernetes Secret or use ExternalSecrets
- Consider using a GitHub App instead of a PAT for better auditability
- Validate all user input to prevent command injection (especially in git operations)
- Consider rate limiting the scaffold endpoint to prevent abuse

## Workflow Diagram

```
┌─────────────┐
│ POST Request│
└──────┬──────┘
       │
       v
┌──────────────────────┐
│ Validate Request     │
│ - Template exists?   │
│ - Project name valid?│
└──────┬───────────────┘
       │
       v
┌──────────────────────┐
│ Run Copier           │
│ /tmp/scaffold/proj/  │
└──────┬───────────────┘
       │
       v
┌──────────────────────┐
│ Create GitHub Repo   │
│ via GitHub API       │
└──────┬───────────────┘
       │
       v
┌──────────────────────┐
│ Git Init & Push      │
│ - git init           │
│ - git add .          │
│ - git commit         │
│ - git push origin    │
└──────┬───────────────┘
       │
       v
┌──────────────────────┐
│ Commit config.json   │
│ to platform repo     │
│ apps/<name>/config   │
└──────┬───────────────┘
       │
       v
┌──────────────────────┐
│ Argo CD Discovers    │
│ ApplicationSet syncs │
└──────────────────────┘
```

## Testing Locally

```bash
# Set required environment variables
export GITHUB_TOKEN="ghp_..."
export GITHUB_ORG="rodmhgl"
export PLATFORM_REPO="homelab-platform"
export SCAFFOLD_TEMPLATES="/path/to/homelab-platform/scaffolds"
export SCAFFOLD_WORK_DIR="/tmp/scaffold"

# Build and run
go build -o platform-api .
./platform-api

# Make a scaffold request
curl -X POST http://localhost:8080/api/v1/scaffold \
  -H "Authorization: Bearer test-token" \
  -H "Content-Type: application/json" \
  -d '{
    "template": "go-service",
    "project_name": "demo-api",
    "enable_storage": true,
    "enable_keyvault": true
  }'
```

## Error Handling

The service returns specific HTTP status codes:

- `201 Created` — Scaffold completed successfully
- `400 Bad Request` — Invalid request (template doesn't exist, invalid project name, etc.)
- `409 Conflict` — GitHub repository already exists
- `500 Internal Server Error` — Copier failed, git operations failed, GitHub API error, etc.

All errors are logged with structured logging for debugging.
