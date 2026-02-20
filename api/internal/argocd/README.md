# Argo CD Handler

This package provides HTTP handlers for interacting with the Argo CD API, enabling the Platform API to manage applications deployed via GitOps.

## Architecture

```text
┌─────────────────────────────────────────────────────┐
│ Platform API                                         │
│                                                      │
│  ┌────────────────┐         ┌──────────────────┐   │
│  │ Handler        │────────>│ Client           │   │
│  │ (HTTP/REST)    │         │ (HTTP client)    │   │
│  └────────────────┘         └──────────────────┘   │
│                                     │               │
└─────────────────────────────────────┼───────────────┘
                                      │ HTTPS + Bearer token
                                      │
┌─────────────────────────────────────▼───────────────┐
│ Argo CD Server                                      │
│ - Applications API (REST)                           │
│ - Authentication via token                          │
└─────────────────────────────────────────────────────┘
```

## Implementation Details

### Client (`client.go`)

Simple HTTP client that wraps the Argo CD REST API:

- **Authentication**: Bearer token passed in `Authorization` header
- **Timeout**: 30-second timeout for all requests
- **Methods**:
  - `ListApplications()` — GET /api/v1/applications
  - `GetApplication(name)` — GET /api/v1/applications/{name}
  - `SyncApplication(name, request)` — POST /api/v1/applications/{name}/sync

**Why REST instead of gRPC?** While Argo CD provides both REST and gRPC APIs, we use REST for simplicity. The gRPC client requires managing protobuf definitions, code generation, and complex connection management. For our use case (occasional API calls from a single-tenant platform), REST is sufficient and easier to maintain.

### Handler (`handler.go`)

Chi-based HTTP handlers that transform Argo CD responses into Platform API responses:

#### GET /api/v1/apps
Lists all applications with simplified response format.

**Response format:**
```json
{
  "applications": [
    {
      "name": "guestbook",
      "namespace": "argocd",
      "project": "default",
      "syncStatus": "Synced",
      "healthStatus": "Healthy",
      "repoURL": "https://github.com/org/repo",
      "path": "manifests",
      "revision": "abc123",
      "lastDeployed": "2024-01-15T10:30:00Z"
    }
  ],
  "total": 1
}
```

**Simplifications from Argo CD response:**
- Extracts only fields needed for dashboard/CLI display
- Flattens nested structures (`spec.source`, `status.sync`, `status.health`)
- Derives `lastDeployed` from operation state or history

#### GET /api/v1/apps/{name}
Returns full application details (unmodified Argo CD Application resource).

Use this endpoint when you need complete information about:
- All resources deployed by the application
- Full sync/operation history
- Detailed health status with messages
- Complete source/destination configuration

#### POST /api/v1/apps/{name}/sync
Triggers an application sync.

**Request body (optional):**
```json
{
  "revision": "main",
  "prune": true,
  "dryRun": false,
  "syncOptions": ["CreateNamespace=true"],
  "resources": [
    {
      "kind": "Deployment",
      "name": "myapp",
      "namespace": "default"
    }
  ]
}
```

**Default behavior (empty body):** Syncs the application using the spec's `targetRevision` with default options.

## Configuration

The handler requires Argo CD connection details via environment variables:

| Variable | Description | Example |
|----------|-------------|---------|
| `ARGOCD_SERVER_URL` | Argo CD API server URL | `https://argocd.example.com` |
| `ARGOCD_TOKEN` | API authentication token | `eyJhbGciOiJIUzI1NiIs...` |

### Generating an Argo CD Token

The Platform API uses a long-lived API token (not session token). Generate one using:

```bash
# 1. Create a service account in Argo CD
kubectl -n argocd patch configmap argocd-cm --type merge -p '
data:
  accounts.platform-api: apiKey
'

# 2. Grant permissions (adjust RBAC as needed)
kubectl -n argocd patch configmap argocd-rbac-cm --type merge -p '
data:
  policy.csv: |
    p, role:platform-api, applications, *, */*, allow
    p, role:platform-api, applicationsets, *, */*, allow
    g, platform-api, role:platform-api
'

# 3. Restart Argo CD server to pick up config changes
kubectl -n argocd rollout restart deployment argocd-server

# 4. Generate the token
argocd login <argocd-server-url> --username admin --password <admin-password>
argocd account generate-token --account platform-api

# 5. Store in Azure Key Vault
az keyvault secret set \
  --vault-name homelab-bootstrap-kv \
  --name argocd-token \
  --value "eyJhbGciOiJIUzI1NiIs..."
```

The token will be synced to the Platform API pod via External Secrets Operator.

## Usage Examples

### List Applications
```bash
curl http://platform-api.platform.svc.cluster.local:8080/api/v1/apps \
  -H "Authorization: Bearer <platform-api-token>"
```

### Get Application Details
```bash
curl http://platform-api.platform.svc.cluster.local:8080/api/v1/apps/guestbook \
  -H "Authorization: Bearer <platform-api-token>"
```

### Sync Application
```bash
# Default sync
curl -X POST http://platform-api.platform.svc.cluster.local:8080/api/v1/apps/guestbook/sync \
  -H "Authorization: Bearer <platform-api-token>"

# Sync with options
curl -X POST http://platform-api.platform.svc.cluster.local:8080/api/v1/apps/guestbook/sync \
  -H "Authorization: Bearer <platform-api-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "prune": true,
    "syncOptions": ["CreateNamespace=true"]
  }'
```

## Error Handling

| Status Code | Condition | Example |
|-------------|-----------|---------|
| 200 | Success | Application found/synced successfully |
| 400 | Bad request | Missing application name in URL |
| 404 | Not found | Application does not exist |
| 500 | Server error | Argo CD API unreachable or returned error |

All error responses use JSON format:
```json
{
  "error": "description of error"
}
```

## Sync Status Values

| Value | Meaning |
|-------|---------|
| `Synced` | Application is synchronized with Git |
| `OutOfSync` | Desired state differs from live state |
| `Unknown` | Sync status cannot be determined |

## Health Status Values

| Value | Meaning |
|-------|---------|
| `Healthy` | All resources are healthy |
| `Progressing` | Resources are being deployed/updated |
| `Degraded` | Some resources are unhealthy |
| `Suspended` | Application is suspended |
| `Missing` | Resources are missing |
| `Unknown` | Health cannot be determined |

## Integration Points

### CLI (`rdp apps`)
The `rdp` CLI will call these endpoints:
- `rdp apps list` → GET /api/v1/apps
- `rdp apps status <name>` → GET /api/v1/apps/{name}
- `rdp apps sync <name>` → POST /api/v1/apps/{name}/sync

### Portal UI
The Portal's Applications panel will:
- Display application cards using GET /api/v1/apps
- Show detailed status using GET /api/v1/apps/{name}
- Trigger syncs via POST /api/v1/apps/{name}/sync

### Future Enhancements
Potential additions not yet implemented:
- GET /api/v1/apps/{name}/logs — Stream application logs
- POST /api/v1/apps/{name}/rollback — Rollback to previous revision
- GET /api/v1/apps/{name}/manifests — Get rendered manifests
- DELETE /api/v1/apps/{name} — Delete application (requires careful RBAC)

## Files

```text
internal/argocd/
├── README.md        # This file
├── types.go         # Argo CD API data structures
├── client.go        # HTTP client for Argo CD API
└── handler.go       # HTTP handlers for Platform API endpoints
```

## References

- [Argo CD API Documentation](https://argo-cd.readthedocs.io/en/stable/developer-guide/api-docs/)
- [Argo CD REST API Examples](https://github.com/argoproj/argo-cd/blob/master/docs/developer-guide/api-docs.md)
