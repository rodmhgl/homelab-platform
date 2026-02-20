# Infrastructure Management API Summary

The Platform API provides a complete CRUD interface for managing Crossplane Claims via GitOps.

## Endpoint Overview

| Method | Endpoint | Description | Status |
|--------|----------|-------------|--------|
| GET | `/api/v1/infra` | List all Claims (StorageBucket + Vault) | ✅ Complete |
| GET | `/api/v1/infra/storage` | List StorageBucket Claims | ✅ Complete |
| GET | `/api/v1/infra/vaults` | List Vault Claims | ✅ Complete |
| GET | `/api/v1/infra/:kind/:name` | Get Claim details with resource tree | ✅ Complete |
| POST | `/api/v1/infra` | Create a new Claim (GitOps) | ✅ Complete |
| DELETE | `/api/v1/infra/:kind/:name` | Delete a Claim (GitOps) | ✅ Complete |

## GitOps Architecture

All write operations (CREATE, DELETE) follow the GitOps pattern:

```
Platform API
    ↓
  GitHub
    ↓
 Argo CD
    ↓
Kubernetes
    ↓
Crossplane
    ↓
   Azure
```

**Key principle:** The API never directly modifies cluster resources. All changes go through Git → Argo CD → Cluster.

## Quick Reference

### List All Claims

```bash
GET /api/v1/infra
```

Returns all Claims across all namespaces with summary information.

### Get Claim Details

```bash
GET /api/v1/infra/storage/my-bucket?namespace=default
```

Returns:
- Claim resource with labels, annotations, status
- Composite resource (XStorageBucket/XKeyVault)
- Managed resources (ResourceGroup, StorageAccount/KeyVault, etc.)
- Events from all resources in the tree

### Create Claim

```bash
POST /api/v1/infra
Content-Type: application/json

{
  "kind": "StorageBucket",
  "name": "my-bucket",
  "namespace": "default",
  "parameters": {
    "location": "southcentralus",
    "storageAccountType": "Standard_LRS"
  },
  "repoOwner": "myorg",
  "repoName": "my-app",
  "labels": {
    "app": "my-app",
    "team": "platform"
  }
}
```

**Three-layer validation:**
1. Request validation (schema, required fields)
2. Gatekeeper constraint validation (location, publicAccess)
3. GitHub commit validation

**Result:**
- Claim YAML committed to `k8s/claims/my-bucket.yaml`
- Argo CD syncs the file to cluster
- Crossplane provisions Azure resources
- Connection secret appears in namespace

### Delete Claim

```bash
DELETE /api/v1/infra/storage/my-bucket?namespace=default
Content-Type: application/json

{
  "repoOwner": "myorg",
  "repoName": "my-app"
}
```

**Result:**
- Claim YAML deleted from `k8s/claims/my-bucket.yaml`
- Argo CD removes Claim from cluster
- Crossplane deprovisions Azure resources
- Connection secret deleted

## Supported Claim Types

### StorageBucket

Creates:
- Azure Resource Group
- Storage Account (V2)
- Blob Container

Parameters:
- `location` (required): Azure region (validated by Gatekeeper)
- `storageAccountType` (optional): `Standard_LRS`, `Standard_GRS`, etc.

### Vault

Creates:
- Azure Resource Group
- Key Vault

Parameters:
- `location` (required): Azure region (validated by Gatekeeper)
- `enableSoftDelete` (optional): boolean, default `true`

## Authentication

All endpoints require Bearer token authentication:

```bash
curl -H "Authorization: Bearer YOUR_TOKEN" \
  http://localhost:8080/api/v1/infra
```

## Error Responses

All endpoints return JSON error responses:

```json
{
  "error": "description of what went wrong"
}
```

Common HTTP status codes:
- `200 OK` - Success
- `201 Created` - Claim created successfully
- `400 Bad Request` - Validation error
- `404 Not Found` - Claim not found
- `500 Internal Server Error` - Server error

## Connection Secrets

When a Claim is provisioned, Crossplane writes connection details to a Kubernetes Secret:

```yaml
kind: Secret
metadata:
  name: <claim-name>  # Same as Claim name
  namespace: <claim-namespace>
type: connection.crossplane.io/v1alpha1
data:
  # StorageBucket secrets:
  storageAccountName: <base64>
  containerName: <base64>
  resourceGroupName: <base64>

  # Vault secrets:
  vaultName: <base64>
  vaultUri: <base64>
  resourceGroupName: <base64>
```

Applications can mount these secrets or use them via ExternalSecrets operator.

## Resource Naming Conventions

### Git Repository

Claims are stored in the app repository at:

```
<repo>/k8s/claims/<claim-name>.yaml
```

### Connection Secrets

Connection secret name = Claim name:

```yaml
apiVersion: platform.homelab.com/v1alpha1
kind: StorageBucket
metadata:
  name: my-bucket  # <-- This is the secret name
spec:
  writeConnectionSecretToRef:
    name: my-bucket  # <-- Always matches metadata.name
```

### Azure Resources

Managed by Crossplane using composition logic:

- **ResourceGroup**: `rg-<claim-name>`
- **StorageAccount**: `st<sanitized-claim-name>` (lowercase, no special chars)
- **KeyVault**: `kv-<claim-name>` (up to 24 chars)

## Next Steps

Pending infrastructure-related endpoints:

- [ ] `GET /api/v1/secrets/:namespace` - List ExternalSecrets + connection secrets (#50)
- [ ] CLI command: `rdp infra list/status` - Tabular Claim view (#68)
- [ ] CLI command: `rdp infra create storage` - Interactive prompts (#69)
- [ ] CLI command: `rdp infra create vault` - Interactive prompts (#70)
- [ ] CLI command: `rdp infra delete` - Interactive deletion (#71)

## See Also

- [DELETE_INFRA.md](./DELETE_INFRA.md) - Detailed DELETE endpoint documentation
- [Infrastructure API source](../internal/infra/) - Implementation code
- [Platform Design](../../PLATFORM_DESIGN.md) - Overall platform architecture
