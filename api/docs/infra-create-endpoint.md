# POST /api/v1/infra - Create Infrastructure Claim

## Overview

This endpoint implements the platform's **GitOps contract for infrastructure provisioning**. Instead of creating Claims directly in the Kubernetes cluster, it commits Claim YAML to the application's Git repository, where Argo CD will sync it.

**Demo Flow:**
```
rdp infra create storage → Platform API → Git commit → Argo CD sync → Crossplane → Azure resources
```

This is **Task #46**, the critical blocker for Act 5 of the platform demo (Self-Service Infrastructure).

---

## Request

**Method:** `POST`
**Path:** `/api/v1/infra`
**Content-Type:** `application/json`

### Request Body

```json
{
  "kind": "StorageBucket" | "Vault",
  "name": "claim-name",
  "namespace": "target-namespace",
  "repoOwner": "github-org-or-user",
  "repoName": "app-repo-name",
  "parameters": {
    // Kind-specific parameters (see below)
  },
  "labels": {
    // Optional custom labels
  }
}
```

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `kind` | string | Yes | Must be `StorageBucket` or `Vault` |
| `name` | string | Yes | DNS label format (lowercase alphanumeric + hyphens, max 63 chars) |
| `namespace` | string | Yes | Target Kubernetes namespace |
| `repoOwner` | string | Yes | GitHub organization or user (e.g., `rodmhgl`) |
| `repoName` | string | Yes | App repository name (e.g., `demo-app`) |
| `parameters` | object | Yes | XRD-specific parameters (see below) |
| `labels` | object | No | Optional custom labels (merged with required labels) |

---

## Parameters

### StorageBucket Parameters

| Parameter | Type | Required | Default | Allowed Values |
|-----------|------|----------|---------|----------------|
| `location` | string | No | `southcentralus` | `southcentralus`, `eastus2` |
| `tier` | string | No | `Standard` | `Standard`, `Premium` |
| `redundancy` | string | No | `LRS` | `LRS`, `ZRS`, `GRS`, `GZRS`, `RAGRS`, `RAGZRS` |
| `enableVersioning` | boolean | No | `false` | `true`, `false` |
| `publicAccess` | boolean | No | `false` | **Always `false`** (Gatekeeper blocks `true`) |

### Vault Parameters

| Parameter | Type | Required | Default | Allowed Values |
|-----------|------|----------|---------|----------------|
| `location` | string | No | `southcentralus` | `southcentralus`, `eastus2` |
| `skuName` | string | No | `standard` | `standard`, `premium` |
| `softDeleteRetentionDays` | integer | No | `7` | `7` to `90` |
| `publicAccess` | boolean | No | `false` | **Always `false`** (Gatekeeper blocks `true`) |

---

## Response

**Success Status Code:** `201 Created`

```json
{
  "success": true,
  "message": "Claim committed successfully. Argo CD will sync it to the cluster.",
  "kind": "StorageBucket",
  "name": "demo-app-storage",
  "namespace": "demo-app",
  "commitSha": "a1b2c3d4e5f6...",
  "filePath": "k8s/claims/demo-app-storage.yaml",
  "repoUrl": "https://github.com/rodmhgl/demo-app",
  "connectionSecret": "demo-app-storage"
}
```

### Response Fields

| Field | Description |
|-------|-------------|
| `success` | Always `true` for 201 responses |
| `message` | Human-readable success message |
| `kind` | The Claim kind (`StorageBucket` or `Vault`) |
| `name` | The Claim name |
| `namespace` | The target namespace |
| `commitSha` | Git commit SHA for audit trail |
| `filePath` | Path in the repo where YAML was committed |
| `repoUrl` | GitHub repository URL |
| `connectionSecret` | Name of the Kubernetes Secret containing connection details |

---

## Error Responses

### 400 Bad Request - Invalid Request

```json
{
  "error": "validation failed: name must be a valid DNS label (lowercase alphanumeric + hyphens, cannot start/end with hyphen)"
}
```

**Common validation errors:**
- Invalid `kind` (must be `StorageBucket` or `Vault`)
- Invalid `name` (uppercase, special characters, too long)
- Missing required fields (`namespace`, `repoOwner`, `repoName`)

### 400 Bad Request - Gatekeeper Constraint Violation

```json
{
  "error": "policy violation: location 'westeurope' is not allowed (allowed: southcentralus, eastus2)"
}
```

**Gatekeeper-enforced constraints:**
- Location must be `southcentralus` or `eastus2`
- `publicAccess: true` is blocked
- `tier` must be `Standard` or `Premium` (StorageBucket)
- `redundancy` must be valid (StorageBucket)
- `skuName` must be `standard` or `premium` (Vault)
- `softDeleteRetentionDays` must be 7-90 (Vault)

### 500 Internal Server Error - GitHub API Failure

```json
{
  "error": "failed to commit to repository: 404 Not Found"
}
```

**Common GitHub errors:**
- Repository not found (404) → Scaffold the app first
- No write access (403) → Check GitHub token permissions
- Rate limit exceeded (429) → Wait and retry

---

## Examples

### Example 1: Create StorageBucket with Defaults

**Request:**
```bash
curl -X POST http://localhost:8080/api/v1/infra \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "kind": "StorageBucket",
    "name": "demo-app-storage",
    "namespace": "demo-app",
    "repoOwner": "rodmhgl",
    "repoName": "demo-app",
    "parameters": {
      "location": "southcentralus"
    }
  }'
```

**Response:**
```json
{
  "success": true,
  "message": "Claim committed successfully. Argo CD will sync it to the cluster.",
  "kind": "StorageBucket",
  "name": "demo-app-storage",
  "namespace": "demo-app",
  "commitSha": "abc123...",
  "filePath": "k8s/claims/demo-app-storage.yaml",
  "repoUrl": "https://github.com/rodmhgl/demo-app",
  "connectionSecret": "demo-app-storage"
}
```

**Generated YAML** (committed to `k8s/claims/demo-app-storage.yaml`):
```yaml
apiVersion: platform.example.com/v1alpha1
kind: StorageBucket
metadata:
  name: demo-app-storage
  namespace: demo-app
  labels:
    app.kubernetes.io/name: demo-app
    app.kubernetes.io/instance: demo-app-storage
    app.kubernetes.io/version: "1.0.0"
    app.kubernetes.io/component: infrastructure
    app.kubernetes.io/part-of: demo-app
    app.kubernetes.io/managed-by: crossplane
spec:
  parameters:
    location: southcentralus
    tier: Standard
    redundancy: LRS
    enableVersioning: false
    publicAccess: false
  writeConnectionSecretToRef:
    name: demo-app-storage
    namespace: demo-app
  compositionSelector:
    matchLabels:
      provider: azure
      type: storage
```

---

### Example 2: Create Vault with Custom SKU

**Request:**
```bash
curl -X POST http://localhost:8080/api/v1/infra \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "kind": "Vault",
    "name": "demo-app-vault",
    "namespace": "demo-app",
    "repoOwner": "rodmhgl",
    "repoName": "demo-app",
    "parameters": {
      "location": "eastus2",
      "skuName": "premium",
      "softDeleteRetentionDays": 30
    }
  }'
```

**Response:**
```json
{
  "success": true,
  "message": "Claim committed successfully. Argo CD will sync it to the cluster.",
  "kind": "Vault",
  "name": "demo-app-vault",
  "namespace": "demo-app",
  "commitSha": "def456...",
  "filePath": "k8s/claims/demo-app-vault.yaml",
  "repoUrl": "https://github.com/rodmhgl/demo-app",
  "connectionSecret": "demo-app-vault"
}
```

---

### Example 3: Validation Failure - Invalid Location

**Request:**
```bash
curl -X POST http://localhost:8080/api/v1/infra \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "kind": "StorageBucket",
    "name": "test-storage",
    "namespace": "default",
    "repoOwner": "rodmhgl",
    "repoName": "test-app",
    "parameters": {
      "location": "westeurope"
    }
  }'
```

**Response:** `400 Bad Request`
```json
{
  "error": "policy violation: location 'westeurope' is not allowed (allowed: southcentralus, eastus2)"
}
```

---

### Example 4: Validation Failure - Public Access Blocked

**Request:**
```bash
curl -X POST http://localhost:8080/api/v1/infra \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "kind": "StorageBucket",
    "name": "test-storage",
    "namespace": "default",
    "repoOwner": "rodmhgl",
    "repoName": "test-app",
    "parameters": {
      "location": "southcentralus",
      "publicAccess": true
    }
  }'
```

**Response:** `400 Bad Request`
```json
{
  "error": "policy violation: publicAccess: true is not allowed (enforced by Gatekeeper)"
}
```

---

## End-to-End Workflow

**Timeline:** ~5-10 minutes from API call to live Azure resources

1. **API commits YAML** (< 1 second)
   - Validates request
   - Validates against Gatekeeper constraints
   - Generates YAML
   - Commits to `k8s/claims/{name}.yaml` in app repo

2. **Argo CD detects change** (3-5 minutes, or immediate if manually synced)
   - ApplicationSet watches app repo
   - Syncs Claim to cluster

3. **Gatekeeper validates Claim** (< 1 second)
   - Checks location constraint
   - Checks publicAccess constraint
   - Admits or rejects

4. **Crossplane provisions Azure resources** (2-5 minutes)
   - Creates ResourceGroup
   - Creates Storage Account or Key Vault
   - Creates connection secret

5. **Connection secret appears** (< 1 second after Crossplane finishes)
   - Secret created in target namespace
   - Contains storage account keys / vault URI
   - Ready for consumption by app Pods

---

## Verification

After creating a Claim, verify the workflow:

```bash
# 1. Check the commit in GitHub
open "https://github.com/${REPO_OWNER}/${REPO_NAME}/blob/main/k8s/claims/${NAME}.yaml"

# 2. Trigger Argo CD sync (or wait for auto-sync)
curl -X POST http://localhost:8080/api/v1/apps/${APP_NAME}/sync \
  -H "Authorization: Bearer $TOKEN"

# 3. Verify Claim exists in cluster
kubectl get storagebucket ${NAME} -n ${NAMESPACE}

# 4. Check Crossplane provisioned Azure resources
kubectl get managed -l crossplane.io/claim-name=${NAME}

# 5. Verify connection secret exists
kubectl get secret ${NAME} -n ${NAMESPACE}
kubectl get secret ${NAME} -n ${NAMESPACE} -o jsonpath='{.data}' | jq .

# 6. Query resource tree via API
curl http://localhost:8080/api/v1/infra/storage/${NAME}?namespace=${NAMESPACE} \
  -H "Authorization: Bearer $TOKEN" | jq .
```

---

## Implementation Details

### Files Created

| File | Purpose |
|------|---------|
| `api/internal/infra/types.go` | Added `CreateClaimRequest` & `CreateClaimResponse` |
| `api/internal/infra/github.go` | GitHub API client with `CommitClaim` function |
| `api/internal/infra/handler.go` | `HandleCreateClaim` implementation |
| `api/internal/infra/validation.go` | Request validation & Gatekeeper constraint validation |
| `api/internal/infra/templates.go` | YAML generation for StorageBucket & Vault Claims |
| `api/internal/infra/validation_test.go` | Comprehensive test suite |

### Validation Layers

1. **Request Validation**
   - Kind ∈ {`StorageBucket`, `Vault`}
   - Name is valid DNS label
   - Namespace, RepoOwner, RepoName non-empty

2. **Gatekeeper Pre-Validation** (mirrors cluster admission rules)
   - Location ∈ {`southcentralus`, `eastus2`}
   - `publicAccess: true` is blocked
   - StorageBucket: `tier` ∈ {`Standard`, `Premium`}, `redundancy` valid
   - Vault: `skuName` ∈ {`standard`, `premium`}, `softDeleteRetentionDays` ∈ [7-90]

3. **GitHub API Validation**
   - Repository exists
   - Token has write access
   - Commit succeeds

---

## Future Enhancements (Out of Scope)

- **PATCH /api/v1/infra/{kind}/{name}** — Update existing Claims
- **Dry-run mode** (`?dryRun=true`) — Return YAML without committing
- **Namespace pre-validation** — Check namespace exists before commit
- **Branch strategy** — Commit to feature branch and return PR URL
- **Claim templates** — User-defined templates beyond StorageBucket/Vault

---

## Related Endpoints

- **GET /api/v1/infra** — List all Claims (Task #44)
- **GET /api/v1/infra/{kind}/{name}** — Query Claim resource tree (Task #45)
- **DELETE /api/v1/infra/{kind}/{name}** — Delete Claim (Task #47, pending)
- **POST /api/v1/apps/{name}/sync** — Trigger Argo CD sync (Task #43)

---

## References

- **XRD Schemas:**
  - `/platform/crossplane-config/xrds/xstoragebucket.yaml`
  - `/platform/crossplane-config/xrds/xkeyvault.yaml`

- **Gatekeeper Constraints:**
  - `/platform/gatekeeper-constraints/constraints/crossplane-claim-location.yaml`
  - `/platform/gatekeeper-constraints/constraints/crossplane-no-public-access.yaml`

- **Scaffold Templates:**
  - `/scaffolds/go-service/{{project_name}}/k8s/claims/storage.yaml.jinja`
  - `/scaffolds/go-service/{{project_name}}/k8s/claims/vault.yaml.jinja`
