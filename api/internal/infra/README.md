# Infrastructure API Package

This package implements the Platform API's infrastructure management endpoints for querying Crossplane resources.

## Endpoints

### GET /api/v1/infra/:kind/:name

Retrieves the complete composed resource tree and Kubernetes events for a Crossplane Claim.

**Path Parameters:**
- `kind` - Claim kind (storagebucket, vault, storage, keyvault)
- `name` - Claim name

**Query Parameters:**
- `namespace` (optional) - Namespace of the Claim (defaults to "default")

**Response:**

```json
{
  "claim": {
    "name": "my-bucket",
    "namespace": "default",
    "kind": "StorageBucket",
    "status": "Ready",
    "synced": true,
    "ready": true,
    "connectionSecret": "my-bucket-conn",
    "creationTimestamp": "2026-02-20T10:30:00Z",
    "resourceRef": {
      "name": "my-bucket-xyz123",
      "kind": "XStorageBucket"
    }
  },
  "composite": {
    "name": "my-bucket-xyz123",
    "kind": "XStorageBucket",
    "status": "Ready",
    "synced": true,
    "ready": true,
    "creationTimestamp": "2026-02-20T10:30:01Z",
    "resourceRefs": [
      {
        "name": "my-bucket-rg",
        "kind": "ResourceGroup",
        "apiVersion": "azure.upbound.io/v1beta1"
      },
      {
        "name": "my-bucket-sa",
        "kind": "Account",
        "apiVersion": "storage.azure.upbound.io/v1beta2"
      }
    ]
  },
  "managed": [
    {
      "name": "my-bucket-rg",
      "kind": "ResourceGroup",
      "group": "azure.upbound.io",
      "status": "Ready",
      "synced": true,
      "ready": true,
      "externalName": "rg-my-bucket",
      "creationTimestamp": "2026-02-20T10:30:02Z",
      "message": "Successfully created resource group"
    },
    {
      "name": "my-bucket-sa",
      "kind": "Account",
      "group": "storage.azure.upbound.io",
      "status": "Ready",
      "synced": true,
      "ready": true,
      "externalName": "stmybucket",
      "creationTimestamp": "2026-02-20T10:30:05Z",
      "message": "Successfully provisioned storage account"
    }
  ],
  "events": [
    {
      "type": "Normal",
      "reason": "Synced",
      "message": "Successfully reconciled managed resource",
      "involvedObject": "Account/my-bucket-sa",
      "source": "provider-azure",
      "count": 1,
      "firstTimestamp": "2026-02-20T10:30:10Z",
      "lastTimestamp": "2026-02-20T10:30:10Z"
    }
  ]
}
```

**Status Codes:**
- `200 OK` - Resource tree retrieved successfully
- `404 Not Found` - Claim not found
- `401 Unauthorized` - Missing or invalid Bearer token
- `500 Internal Server Error` - Failed to query Kubernetes API

**Example:**

```bash
# Get StorageBucket claim details
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/infra/storage/my-bucket

# Get Vault claim in a specific namespace
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/infra/vault/my-vault?namespace=app-prod
```

## Resource Hierarchy

The endpoint returns the full Crossplane resource composition tree:

```
Claim (namespaced)
  ↓ resourceRef
Composite (cluster-scoped XR)
  ↓ resourceRefs[]
Managed Resources (Azure resources)
```

## Implementation Details

### Client

The `Client` wraps two Kubernetes clients:
- **Dynamic client** - for querying CRDs (Claims, Composites, Managed Resources)
- **Core client** - for querying Events

### Supported Claim Kinds

- `StorageBucket` → `XStorageBucket` → ResourceGroup + StorageAccount + BlobContainer
- `Vault` → `XKeyVault` → ResourceGroup + KeyVault

### Status Determination

Resource status is derived from Crossplane condition types:
- `Ready=True` + `Synced=True` → **Ready**
- `Ready=False` + `Synced=False` → **Failed**
- Otherwise → **Progressing**

### Events

Events are queried for:
1. The Claim itself (in its namespace)
2. The Composite resource (cluster-scoped)
3. All Managed Resources (cluster-scoped)

Events are sorted by `lastTimestamp` (most recent first).

## Future Work

- Add filtering options (e.g., `?events=false` to skip event retrieval for performance)
- Add pagination for large resource trees
- Cache composite resource lookups to reduce API calls
