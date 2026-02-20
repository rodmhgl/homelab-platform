# Implementation Notes

This file tracks significant implementation decisions, bug fixes, and lessons learned during platform development.

## 2026-02-20: Infrastructure Query Endpoint & Crossplane Composition Fixes

### Task #45: GET /api/v1/infra/:kind/:name

**Implemented:** Infrastructure resource tree query endpoint for debugging Crossplane Claims.

**Package:** `api/internal/infra/`

**Key Features:**
- Traverses full resource composition: Claim → Composite → Managed Resources
- Retrieves Kubernetes Events for all resources in the tree
- Returns status derived from Crossplane conditions (Ready, Synced)
- Supports namespace filtering via `?namespace=` query parameter
- Returns Azure resource names via `externalName` field

**Response Structure:**
```json
{
  "claim": { "name": "...", "status": "Ready|Progressing|Failed", ... },
  "composite": { "name": "...", "kind": "XStorageBucket", ... },
  "managed": [
    { "name": "...", "kind": "ResourceGroup", "externalName": "rg-...", ... }
  ],
  "events": [
    { "type": "Warning", "reason": "ComposeResources", "message": "...", ... }
  ]
}
```

**Example:**
```bash
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/infra/storage/my-bucket?namespace=default
```

---

### RBAC Fix: API Group Mismatch

**Problem:** Platform API ServiceAccount had incorrect API group in RBAC rules.

**Root Cause:**
- XRDs use `platform.example.com` API group
- RBAC was configured for `azure.homelab.rodst.io`
- Client code used `platform.homelab.io`

**Fix Applied:**
1. Updated `platform/platform-api/rbac.yaml`:
   - Changed Claim/Composite API group to `platform.example.com`
2. Updated `api/internal/infra/client.go`:
   - Fixed GVR mappings to use `platform.example.com`
3. Applied RBAC to cluster:
   ```bash
   kubectl apply -f platform/platform-api/rbac.yaml
   ```

**Verification:**
```bash
# Before fix:
# Error: User "system:serviceaccount:platform:platform-api" cannot get resource "storagebuckets" in API group "platform.homelab.io"

# After fix:
# Successfully retrieves resource tree with events
```

---

### Crossplane Composition Bug Fixes

**Bug 1: Invalid Regexp Transform Syntax**

**Problem:** Composition used `replace` field in Regexp transforms, causing fatal errors:
```
cannot unmarshal Go value of type v1beta1.StringTransformRegexp: unknown name "replace"
```

**Root Cause:** The `replace` field doesn't exist in the Crossplane string transform API. The correct approach for character removal is to use capture groups or simpler transforms.

**Fix:** Simplified storage account name sanitization to use only `Convert: ToLower`:
```yaml
# Before (broken):
transforms:
  - type: string
    string:
      type: Regexp
      regexp:
        match: '[-._]'
        replace: ''  # ❌ Invalid field
  - type: string
    string:
      type: Convert
      convert: ToLower

# After (working):
transforms:
  - type: string
    string:
      type: Convert
      convert: ToLower  # ✅ Simple and effective
```

**Rationale:** Azure storage account names accept lowercase alphanumeric characters. Converting to lowercase is sufficient; claim names with special characters will be rejected by Gatekeeper policies.

---

**Bug 2: Missing Connection Detail Type**

**Problem:** Connection details lacked required `type` field:
```
invalid Function input: resources[1].connectionDetails[0].type: Required value: connection detail type is required
```

**Fix:** Added `type: FromConnectionSecretKey` to all connection details:
```yaml
# Before (broken):
connectionDetails:
  - name: primaryAccessKey
    fromConnectionSecretKey: attribute.primary_access_key  # ❌ Missing type

# After (working):
connectionDetails:
  - name: primaryAccessKey
    type: FromConnectionSecretKey  # ✅ Required field
    fromConnectionSecretKey: attribute.primary_access_key
```

**Files Updated:**
- `platform/crossplane-config/compositions/storagebucket-azure.yaml`

**Verification:**
```bash
# Delete old CompositionRevision (immutable)
kubectl delete compositionrevision storagebucket-azure-<hash>

# Recreate claim with fixed composition
kubectl delete storagebucket test-bucket -n default
kubectl apply -f test-storagebucket.yaml

# Verify resources are provisioning
kubectl get storagebucket,xstoragebucket,resourcegroup,account,container
```

---

### Lessons Learned

1. **Crossplane Composition Debugging:**
   - CompositionRevisions are immutable — must delete and recreate to apply fixes
   - Events on the Composite resource show detailed error messages
   - Use `kubectl describe xstoragebucket` to see composition errors

2. **API Group Consistency:**
   - XRD `spec.group` defines the API group for Claims and Composites
   - RBAC, client code, and XRD must all use the same API group
   - Check XRD first when encountering permission errors

3. **Crossplane Transform Syntax:**
   - Pipeline-mode Compositions require `type: FromConnectionSecretKey` on connection details
   - Prefer simple transforms (`Convert`) over complex regex patterns
   - Test transforms with small Claims before deploying to production

4. **Infrastructure Query Endpoint Value:**
   - Events are critical for debugging Crossplane provisioning failures
   - Full resource tree visibility helps understand composition structure
   - Status derivation from conditions (Ready, Synced) provides clear state

---

### Documentation Updated

- `api/README.md` — Added `internal/infra/` package section, marked endpoint as complete
- `api/internal/infra/README.md` — Full endpoint documentation with examples
- `homelab-platform/CLAUDE.md` — Updated API status, added Composition syntax notes
- `CLAUDE.md` — Updated repository status to reflect infra query endpoint completion

---

### Next Steps

**Immediate Next Task:** #46 — `POST /api/v1/infra` (Create Claim via GitOps)

This endpoint will commit Claim YAML to app repositories, completing the GitOps contract for infrastructure provisioning. The query endpoint (#45) provides the inspection capability; the create endpoint (#46) provides the provisioning capability.

**Remaining Infrastructure Endpoints:**
- #44 — `GET /api/v1/infra` (list all Claims)
- #46 — `POST /api/v1/infra` (create Claim)
- #47 — `DELETE /api/v1/infra/:kind/:name` (delete Claim)
