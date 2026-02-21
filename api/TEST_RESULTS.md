# POST /api/v1/infra - Test Results

**Date:** 2026-02-20
**API Version:** v1
**Endpoint:** `POST /api/v1/infra`

## Test Environment

- **API URL:** <http://localhost:8080> (port-forwarded from Kubernetes)
- **Authentication:** Bearer token (validation not yet implemented)
- **GitHub Integration:** ✅ Live commits to rodmhgl/homelab-platform

---

## Test Results Summary

| Test | Status | Description |
| ---- | ------ | ----------- |
| 1. Valid StorageBucket | ✅ PASS | Creates Claim with default parameters |
| 2. Valid Vault | ✅ PASS | Creates Vault with custom parameters |
| 3. Invalid Location | ✅ PASS | Correctly rejects westeurope |
| 4. Public Access Block | ✅ PASS | Correctly rejects publicAccess: true |
| 5. Invalid Name (uppercase) | ✅ PASS | Correctly rejects uppercase names |
| 6. Invalid Tier | ✅ PASS | Correctly rejects invalid tier |
| 7. Invalid Retention | ✅ PASS | Correctly rejects retention < 7 days |

**Result:** 7/7 tests passed ✅

---

## Test Details

### Test 1: Valid StorageBucket Request

**Request:**

```json
{
  "kind": "StorageBucket",
  "name": "test-storage",
  "namespace": "default",
  "repoOwner": "rodmhgl",
  "repoName": "homelab-platform",
  "parameters": {
    "location": "southcentralus",
    "tier": "Standard",
    "redundancy": "LRS"
  }
}
```

**Response:** `201 Created`

```json
{
  "success": true,
  "message": "Claim committed successfully. Argo CD will sync it to the cluster.",
  "kind": "StorageBucket",
  "name": "test-storage",
  "namespace": "default",
  "commitSha": "d40f23bea2c59b69645ba8a0c460194353c2cd79",
  "filePath": "k8s/claims/test-storage.yaml",
  "repoUrl": "https://github.com/rodmhgl/homelab-platform",
  "connectionSecret": "test-storage"
}
```

**GitHub Verification:**

- Commit: <https://github.com/rodmhgl/homelab-platform/commit/d40f23bea2c59b69645ba8a0c460194353c2cd79>
- File: <https://github.com/rodmhgl/homelab-platform/blob/d40f23bea2c59b69645ba8a0c460194353c2cd79/k8s/claims/test-storage.yaml>
- Author: Rod Stewart
- Message: "Add StorageBucket Claim: test-storage\n\nProvisions StorageBucket infrastructure for default namespace.\n\nLocation: southcentralus\nManaged by: Platform API"

**Generated YAML:**

```yaml
apiVersion: platform.example.com/v1alpha1
kind: StorageBucket
metadata:
  name: test-storage
  namespace: default
  labels:
    app.kubernetes.io/component: infrastructure
    app.kubernetes.io/instance: test-storage
    app.kubernetes.io/managed-by: crossplane
    app.kubernetes.io/name: homelab-platform
    app.kubernetes.io/part-of: homelab-platform
    app.kubernetes.io/version: 1.0.0
spec:
  parameters:
    location: southcentralus
    tier: Standard
    redundancy: LRS
    enableVersioning: false
    publicAccess: false  # Enforced by Gatekeeper CrossplaneNoPublicAccess constraint
  writeConnectionSecretToRef:
    name: test-storage
    namespace: default
  compositionSelector:
    matchLabels:
      provider: azure
      type: storage
```

✅ **Validation:** All required labels present, parameters correct, publicAccess hardcoded to false

---

### Test 2: Valid Vault Request

**Request:**

```json
{
  "kind": "Vault",
  "name": "test-vault",
  "namespace": "default",
  "repoOwner": "rodmhgl",
  "repoName": "homelab-platform",
  "parameters": {
    "location": "eastus2",
    "skuName": "premium",
    "softDeleteRetentionDays": 30
  }
}
```

**Response:** `201 Created`

```json
{
  "success": true,
  "message": "Claim committed successfully. Argo CD will sync it to the cluster.",
  "kind": "Vault",
  "name": "test-vault",
  "namespace": "default",
  "commitSha": "800a5ae1da33...",
  "filePath": "k8s/claims/test-vault.yaml",
  "repoUrl": "https://github.com/rodmhgl/homelab-platform",
  "connectionSecret": "test-vault"
}
```

**Generated YAML:**

```yaml
apiVersion: platform.example.com/v1alpha1
kind: Vault
metadata:
  name: test-vault
  namespace: default
  labels:
    app.kubernetes.io/component: infrastructure
    app.kubernetes.io/instance: test-vault
    app.kubernetes.io/managed-by: crossplane
    app.kubernetes.io/name: homelab-platform
    app.kubernetes.io/part-of: homelab-platform
    app.kubernetes.io/version: 1.0.0
spec:
  parameters:
    location: eastus2
    skuName: premium
    publicAccess: false  # Enforced by Gatekeeper CrossplaneNoPublicAccess constraint
    softDeleteRetentionDays: 30
  writeConnectionSecretToRef:
    name: test-vault
    namespace: default
  compositionSelector:
    matchLabels:
      provider: azure
      type: keyvault
```

✅ **Validation:** Custom parameters applied correctly (eastus2, premium, 30 days)

---

### Test 3: Invalid Location (westeurope)

**Request:**

```json
{
  "kind": "StorageBucket",
  "name": "test-invalid-location",
  "namespace": "default",
  "repoOwner": "rodmhgl",
  "repoName": "homelab-platform",
  "parameters": {
    "location": "westeurope"
  }
}
```

**Response:** `400 Bad Request`

```json
{
  "error": "policy violation: location 'westeurope' is not allowed (allowed: southcentralus, eastus2)"
}
```

✅ **Validation:** Correctly enforces Gatekeeper CrossplaneClaimLocation constraint

---

### Test 4: Public Access Blocked

**Request:**

```json
{
  "kind": "StorageBucket",
  "name": "test-public",
  "namespace": "default",
  "repoOwner": "rodmhgl",
  "repoName": "homelab-platform",
  "parameters": {
    "location": "southcentralus",
    "publicAccess": true
  }
}
```

**Response:** `400 Bad Request`

```json
{
  "error": "policy violation: publicAccess: true is not allowed (enforced by Gatekeeper)"
}
```

✅ **Validation:** Correctly enforces Gatekeeper CrossplaneNoPublicAccess constraint

---

### Test 5: Invalid Name (uppercase)

**Request:**

```json
{
  "kind": "StorageBucket",
  "name": "Test-Storage",
  "namespace": "default",
  "repoOwner": "rodmhgl",
  "repoName": "homelab-platform",
  "parameters": {
    "location": "southcentralus"
  }
}
```

**Response:** `400 Bad Request`

```json
{
  "error": "validation failed: name must be a valid DNS label (lowercase alphanumeric + hyphens, cannot start/end with hyphen)"
}
```

✅ **Validation:** Correctly validates DNS label format

---

### Test 6: Invalid Tier

**Request:**

```json
{
  "kind": "StorageBucket",
  "name": "test-tier",
  "namespace": "default",
  "repoOwner": "rodmhgl",
  "repoName": "homelab-platform",
  "parameters": {
    "location": "southcentralus",
    "tier": "InvalidTier"
  }
}
```

**Response:** `400 Bad Request`

```json
{
  "error": "policy violation: invalid tier 'InvalidTier' (allowed: Standard, Premium)"
}
```

✅ **Validation:** Correctly validates StorageBucket tier parameter

---

### Test 7: Invalid Vault Retention (too low)

**Request:**

```json
{
  "kind": "Vault",
  "name": "test-retention",
  "namespace": "default",
  "repoOwner": "rodmhgl",
  "repoName": "homelab-platform",
  "parameters": {
    "location": "southcentralus",
    "softDeleteRetentionDays": 5
  }
}
```

**Response:** `400 Bad Request`

```json
{
  "error": "policy violation: softDeleteRetentionDays must be between 7 and 90, got 5"
}
```

✅ **Validation:** Correctly validates Vault retention days range

---

## Structured Logging Verification

API logs show proper structured logging with slog:

```json
{
  "time": "2026-02-20T15:44:34.517906622Z",
  "level": "INFO",
  "msg": "Creating infrastructure Claim",
  "kind": "StorageBucket",
  "name": "test-public",
  "namespace": "default",
  "repo": "rodmhgl/homelab-platform"
}
```

```json
{
  "time": "2026-02-20T15:44:34.651021635Z",
  "level": "ERROR",
  "msg": "Request validation failed",
  "error": "name must be a valid DNS label (lowercase alphanumeric + hyphens, cannot start/end with hyphen)"
}
```

✅ **All critical paths emit structured logs**

---

## Feature Verification

| Feature | Status | Notes |
| ------- | ------ | ----- |
| GitOps Contract | ✅ | Commits to Git, not direct cluster creation |
| Gatekeeper Pre-Validation | ✅ | Mirrors cluster constraints client-side |
| Smart Defaults | ✅ | southcentralus, Standard/LRS, standard/7 days |
| Label Merging | ✅ | Auto-injects required Gatekeeper labels |
| Connection Secret Name | ✅ | Matches Claim name |
| Comprehensive Errors | ✅ | Clear validation failure messages |
| Structured Logging | ✅ | All critical paths emit slog events |
| YAML Generation | ✅ | Go text/template with proper formatting |

---

## Code Quality

| Metric | Result |
| ------ | ------ |
| Unit Tests | 4 test functions, 27 test cases |
| Test Pass Rate | 100% (27/27) |
| Build Status | ✅ Compiles successfully |
| Lint Status | ⚠️ Minor modernize suggestions (interface{} → any) |
| Documentation | ✅ Complete API docs + examples |

---

## Next Steps

This endpoint **unblocks 3 downstream tasks:**

1. **#69** - `rdp infra create storage` CLI command
2. **#70** - `rdp infra create vault` CLI command
3. **#68** - `rdp infra list` CLI command

The platform is now ready for the **Act 5 demo** (Self-Service Infrastructure):

```text
rdp infra create storage → Platform API → Git commit → Argo CD sync → Crossplane → Azure
```

---

## Known Limitations

1. **Authentication:** Bearer token validation not yet implemented (TODO in authMiddleware)
2. **Dry-run mode:** Not implemented (future enhancement)
3. **Namespace validation:** Doesn't check if namespace exists before commit
4. **Branch strategy:** Commits to main (future: PR workflow)

---

## Clean Up Test Claims

To remove test Claims from the repo:

```bash
# Delete the test files
git rm k8s/claims/test-storage.yaml k8s/claims/test-vault.yaml
git commit -m "Remove test Claims"
git push
```
