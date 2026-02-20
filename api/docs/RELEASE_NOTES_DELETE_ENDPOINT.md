# Release Notes: Infrastructure DELETE Endpoint

**Date:** 2026-02-20
**Task:** #47 - DELETE /api/v1/infra/:kind/:name
**Status:** ✅ Complete

## Summary

Implemented the final piece of CRUD operations for infrastructure management in the Platform API. The DELETE endpoint follows the same GitOps pattern as CREATE, ensuring Git remains the single source of truth for all infrastructure state.

## What's New

### DELETE /api/v1/infra/:kind/:name

Delete a Crossplane Claim by removing its YAML definition from the application's Git repository.

**Request:**
```bash
DELETE /api/v1/infra/storage/my-bucket?namespace=default
Content-Type: application/json
Authorization: Bearer YOUR_TOKEN

{
  "repoOwner": "myorg",
  "repoName": "my-app"
}
```

**Response:**
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

## GitOps Workflow

The deletion follows a multi-step reconciliation loop:

1. **API Request** → Validates request body (repoOwner + repoName required)
2. **Cluster Verification** → Checks if Claim exists (warns if missing, proceeds anyway)
3. **Git Deletion** → Removes `k8s/claims/<name>.yaml` from repository
4. **Argo CD Sync** → Detects file deletion and removes Claim from cluster
5. **Crossplane Cleanup** → Deprovisions all Azure resources (ResourceGroup, StorageAccount/KeyVault, etc.)
6. **Secret Cleanup** → Kubernetes automatically removes connection secrets

## Implementation Details

### Files Modified

- `api/internal/infra/handler.go` - Added `HandleDeleteClaim()` method
- `api/internal/infra/github.go` - Added `DeleteClaim()` GitHub API method
- `api/internal/infra/types.go` - Added `DeleteClaimRequest` and `DeleteClaimResponse`
- `api/main.go` - Wired up DELETE route (replaced `notImplementedHandler`)

### Files Created

- `api/internal/infra/delete_test.go` - Test suite for delete functionality
- `api/docs/DELETE_INFRA.md` - Comprehensive endpoint documentation
- `api/docs/INFRA_API_SUMMARY.md` - Complete infrastructure API reference

### Key Features

✅ **GitOps Pattern** - Never touches cluster directly, only modifies Git
✅ **Defensive Programming** - Verifies Claim existence, warns if missing, continues anyway
✅ **Cascading Deletion** - Crossplane automatically cleans up all managed resources
✅ **Audit Trail** - Every deletion creates a Git commit with full attribution
✅ **Error Handling** - Proper 404 handling for missing files, clear error messages
✅ **Test Coverage** - JSON marshaling tests, validation tests, all passing

## Breaking Changes

None. This is a new endpoint with no impact on existing functionality.

## Migration Guide

No migration needed. If you were manually deleting Claims via `kubectl delete`, you can now use the API:

**Old way:**
```bash
kubectl delete storagebucket my-bucket -n default
```

**New way (via API):**
```bash
curl -X DELETE 'http://platform-api/api/v1/infra/storage/my-bucket?namespace=default' \
  -H 'Authorization: Bearer TOKEN' \
  -d '{"repoOwner": "myorg", "repoName": "my-app"}'
```

**New way (via CLI - coming soon):**
```bash
rdp infra delete storage my-bucket
```

## Testing

All tests passing:

```bash
$ go test ./internal/infra -v
=== RUN   TestDeleteClaimRequest_JSONMarshaling
--- PASS: TestDeleteClaimRequest_JSONMarshaling (0.00s)
=== RUN   TestDeleteClaimResponse_JSONMarshaling
--- PASS: TestDeleteClaimResponse_JSONMarshaling (0.00s)
=== RUN   TestDeleteClaimRequest_Validation
--- PASS: TestDeleteClaimRequest_Validation (0.00s)
PASS
ok      github.com/rodmhgl/homelab-platform/api/internal/infra  0.008s
```

## Documentation

- **Endpoint Reference:** [DELETE_INFRA.md](./DELETE_INFRA.md)
- **API Summary:** [INFRA_API_SUMMARY.md](./INFRA_API_SUMMARY.md)
- **Package README:** [../internal/infra/README.md](../internal/infra/README.md)

## What's Next

With full CRUD operations now complete for infrastructure, the next priorities are:

1. **Secrets Management** (#50) - `GET /api/v1/secrets/:namespace`
2. **Webhooks** (#49) - Falco and Argo CD event receivers
3. **AI Operations** (#52, #53) - HolmesGPT and kagent integration
4. **CLI Commands** (#67-#77) - Interactive infrastructure management
5. **Portal UI** (#78-#86) - React dashboard implementation

## Contributors

- Platform API implementation
- GitOps pattern adherence
- Comprehensive testing and documentation

---

**Complete Infrastructure API Status:**

| Operation | Endpoint | Status |
|-----------|----------|--------|
| List All | `GET /api/v1/infra` | ✅ Complete |
| List Storage | `GET /api/v1/infra/storage` | ✅ Complete |
| List Vaults | `GET /api/v1/infra/vaults` | ✅ Complete |
| Get Details | `GET /api/v1/infra/:kind/:name` | ✅ Complete |
| Create | `POST /api/v1/infra` | ✅ Complete |
| Delete | `DELETE /api/v1/infra/:kind/:name` | ✅ Complete |

**The Platform API infrastructure layer is now feature-complete!** 🎉
