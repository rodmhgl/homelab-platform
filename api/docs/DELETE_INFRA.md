# DELETE /api/v1/infra/:kind/:name

Delete a Crossplane Claim by removing its YAML definition from the application's Git repository.

## Overview

This endpoint implements the **GitOps deletion pattern** for Crossplane Claims. Instead of directly deleting the Claim from the cluster, it removes the Claim YAML file from the application's Git repository. Argo CD then detects the removal and deletes the Claim from the cluster, which triggers Crossplane to deprovision the associated Azure resources.

## Endpoint

```
DELETE /api/v1/infra/:kind/:name
```

## Path Parameters

| Parameter | Type | Description | Example |
|-----------|------|-------------|---------|
| `kind` | string | Type of Claim (case-insensitive) | `storage`, `StorageBucket`, `vault`, `Vault` |
| `name` | string | Name of the Claim to delete | `my-bucket` |

## Query Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `namespace` | string | `default` | Kubernetes namespace containing the Claim |

## Request Body

```json
{
  "repoOwner": "string",
  "repoName": "string"
}
```

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `repoOwner` | string | **Yes** | GitHub organization or username |
| `repoName` | string | **Yes** | Application repository name |

## Response

### Success Response (200 OK)

```json
{
  "success": true,
  "message": "Claim deleted successfully from Git. Argo CD will remove it from the cluster.",
  "kind": "StorageBucket",
  "name": "my-bucket",
  "namespace": "default",
  "commitSha": "abc123def456...",
  "filePath": "k8s/claims/my-bucket.yaml",
  "repoUrl": "https://github.com/myorg/my-app"
}
```

### Error Responses

#### 400 Bad Request - Invalid JSON

```json
{
  "error": "invalid request body: ..."
}
```

#### 400 Bad Request - Missing Required Fields

```json
{
  "error": "repoOwner and repoName are required"
}
```

#### 404 Not Found - File Not Found

```json
{
  "error": "failed to delete from repository: file not found: k8s/claims/my-bucket.yaml"
}
```

#### 500 Internal Server Error - GitHub API Failure

```json
{
  "error": "failed to delete from repository: ..."
}
```

## Example Usage

### cURL

```bash
curl -X DELETE 'http://localhost:8080/api/v1/infra/storage/my-bucket?namespace=default' \
  -H 'Authorization: Bearer YOUR_TOKEN' \
  -H 'Content-Type: application/json' \
  -d '{
    "repoOwner": "myorg",
    "repoName": "my-app"
  }'
```

### Delete a Vault Claim

```bash
curl -X DELETE 'http://localhost:8080/api/v1/infra/vault/my-vault?namespace=production' \
  -H 'Authorization: Bearer YOUR_TOKEN' \
  -H 'Content-Type: application/json' \
  -d '{
    "repoOwner": "myorg",
    "repoName": "my-app"
  }'
```

## GitOps Workflow

The deletion follows the GitOps reconciliation loop:

```
1. API receives DELETE request
   ↓
2. Verify Claim exists in cluster (optional - warns if missing)
   ↓
3. Delete Claim YAML from Git repo (k8s/claims/<name>.yaml)
   ↓
4. Return commit SHA to caller
   ↓
5. Argo CD detects file deletion in Git
   ↓
6. Argo CD removes Claim from cluster
   ↓
7. Crossplane detects Claim deletion
   ↓
8. Crossplane deprovisions Azure resources
   ↓
9. Azure resources deleted
```

## Implementation Details

### File Path Convention

Claims are expected to be stored at:

```
k8s/claims/<claim-name>.yaml
```

### Commit Message Format

```
chore(infra): delete <Kind> Claim <name>

Namespace: <namespace>
Removed via Platform API
```

### Cluster Verification

The endpoint attempts to verify that the Claim exists in the cluster before deleting from Git. If the Claim is not found:

- A warning is logged
- Deletion from Git proceeds anyway
- This handles cases where the Claim was manually deleted

This "delete from Git regardless" behavior ensures GitOps reconciliation can clean up orphaned state.

### Kind Normalization

The `kind` parameter is normalized to support multiple formats:

| Input | Normalized |
|-------|-----------|
| `storage`, `storagebucket`, `storagebuckets` | `StorageBucket` |
| `vault`, `vaults`, `keyvault`, `keyvaults` | `Vault` |

## Security Considerations

1. **Authentication**: Requires Bearer token authentication
2. **GitHub Permissions**: The API's GitHub token must have write access to the repository
3. **No Direct Cluster Deletion**: The API never directly deletes Kubernetes resources - only Git files
4. **Audit Trail**: All deletions create Git commits with full attribution

## Error Handling

The endpoint validates:

1. Request body is valid JSON
2. Both `repoOwner` and `repoName` are provided
3. File exists in the repository before attempting deletion
4. GitHub API operations succeed

## Idempotency

The endpoint is **not idempotent**. Attempting to delete an already-deleted Claim will return a 500 error with "file not found" message.

To check if a Claim exists before deletion, use:

```
GET /api/v1/infra/:kind/:name
```

## Related Endpoints

- `GET /api/v1/infra/:kind/:name` - View Claim status and resource tree
- `POST /api/v1/infra` - Create a new Claim via GitOps
- `GET /api/v1/infra` - List all Claims

## Notes

- **Cascading Deletion**: Deleting a Claim triggers Crossplane to deprovision all managed Azure resources (ResourceGroup, StorageAccount/KeyVault, etc.)
- **Connection Secrets**: Connection secrets are automatically cleaned up by Kubernetes when the Claim is deleted
- **Argo CD Sync**: Deletion will occur on the next Argo CD sync cycle (default: 3 minutes, or trigger manually with `argocd app sync <app-name>`)
