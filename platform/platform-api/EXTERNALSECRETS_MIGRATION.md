# External Secrets Migration — Platform API

This document describes the migration from static Secret resources to External Secrets Operator (ESO) for the Platform API.

## What Changed

### Before (Static Secrets)
```yaml
# secret.yaml — secrets hardcoded in Git (placeholder values)
apiVersion: v1
kind: Secret
metadata:
  name: platform-api-secrets
stringData:
  GITHUB_TOKEN: ""      # TODO: Populate manually
  OPENAI_API_KEY: ""    # TODO: Populate manually
  ARGOCD_TOKEN: ""      # TODO: Populate manually
```

**Problems:**
- Secrets were committed to Git (even if empty)
- Manual secret rotation required redeploying the Secret
- No audit trail for secret access
- Risk of accidentally committing real credentials

### After (External Secrets Operator)
```yaml
# externalsecrets/platform-api-secrets.yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: platform-api-secrets
spec:
  secretStoreRef:
    name: azure-bootstrap-kv  # References ClusterSecretStore
  target:
    name: platform-api-secrets
  data:
    - secretKey: GITHUB_TOKEN
      remoteRef:
        key: github-pat
    # ... other secrets
```

**Benefits:**
- ✅ Zero secrets in Git
- ✅ Automatic secret rotation (ESO syncs from Key Vault every 1h)
- ✅ Centralized secret management in Azure Key Vault
- ✅ OIDC-based authentication (Workload Identity, no static credentials)
- ✅ Audit trail via Azure Key Vault access logs
- ✅ Namespace isolation (ExternalSecret creates Secret in same namespace)

## Changes Made

1. **Created `externalsecrets/` directory:**
   - `platform-api-secrets.yaml` — ExternalSecret resource
   - `kustomization.yaml` — Kustomize config
   - `README.md` — Comprehensive documentation

2. **Updated `kustomization.yaml`:**
   - Removed reference to `secret.yaml`
   - Added reference to `externalsecrets/` directory

3. **Deprecated `secret.yaml`:**
   - Renamed to `secret.yaml.deprecated` (kept for reference)
   - No longer deployed by Argo CD

4. **Updated `deployment.yaml`:**
   - Removed `checksum/secret` annotation (ESO handles secret updates)
   - Deployment still uses `envFrom.secretRef` (no change to pod spec)

## Required Actions (Operator Checklist)

Before the Platform API will work with ExternalSecrets, you must provision the secrets in Azure Key Vault:

### ☐ 1. Create GitHub Personal Access Token

```bash
# 1. Visit: https://github.com/settings/tokens
# 2. Click "Generate new token (classic)"
# 3. Select scope: repo (full control of private repositories)
# 4. Copy the token (ghp_...)
# 5. Set in Key Vault:

az keyvault secret set \
  --vault-name homelab-bootstrap-kv \
  --name github-pat \
  --value "ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
```

### ☐ 2. Create OpenAI API Key

```bash
# 1. Visit: https://platform.openai.com/api-keys
# 2. Click "Create new secret key"
# 3. Copy the key (sk-proj-...)
# 4. Set in Key Vault:

az keyvault secret set \
  --vault-name homelab-bootstrap-kv \
  --name openai-api-key \
  --value "sk-proj-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
```

### ☐ 3. Create Argo CD API Token

```bash
# 1. Create ServiceAccount in Argo CD
kubectl -n argocd patch configmap argocd-cm --type merge -p '
data:
  accounts.platform-api: apiKey
'

# 2. Grant permissions
kubectl -n argocd patch configmap argocd-rbac-cm --type merge -p '
data:
  policy.csv: |
    p, role:platform-api, applications, *, */*, allow
    p, role:platform-api, applicationsets, *, */*, allow
    g, platform-api, role:platform-api
'

# 3. Restart Argo CD server
kubectl -n argocd rollout restart deployment argocd-server

# 4. Generate token
argocd login <argocd-server-url>
argocd account generate-token --account platform-api

# 5. Set in Key Vault
az keyvault secret set \
  --vault-name homelab-bootstrap-kv \
  --name argocd-token \
  --value "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

### ☐ 4. Verify ESO is Running

```bash
# Check ClusterSecretStore status
kubectl get clustersecretstore azure-bootstrap-kv
# Expected: Ready=True

# Check ESO controller pods
kubectl get pods -n external-secrets
# Expected: 1+ pods Running
```

### ☐ 5. Commit and Deploy

```bash
# Commit the changes
cd homelab-platform
git add platform/platform-api/
git commit -m "Migrate Platform API to External Secrets Operator

- Add ExternalSecret resources for GitHub token, OpenAI key, Argo CD token
- Update kustomization to use externalsecrets/ directory
- Deprecate static secret.yaml
- Add comprehensive documentation

Tasks: #40, #87"

git push origin main
```

### ☐ 6. Verify Deployment

```bash
# Wait for Argo CD sync (wave 10)
kubectl get applications platform-api -n argocd -w

# Check ExternalSecret status
kubectl get externalsecret platform-api-secrets -n platform
kubectl describe externalsecret platform-api-secrets -n platform
# Expected: Conditions.Ready=True, Status=SecretSynced

# Verify Secret was created
kubectl get secret platform-api-secrets -n platform
kubectl get secret platform-api-secrets -n platform -o jsonpath='{.data}' | jq 'keys'
# Expected: ["ARGOCD_TOKEN", "GITHUB_TOKEN", "OPENAI_API_KEY"]

# Check Platform API pods
kubectl get pods -n platform -l app.kubernetes.io/name=platform-api
kubectl logs -n platform -l app.kubernetes.io/name=platform-api --tail=50
# Expected: No "missing configuration" errors
```

## Troubleshooting

### ExternalSecret shows SecretSyncedError

```bash
# Check which secret is failing
kubectl describe externalsecret platform-api-secrets -n platform

# Verify secrets exist in Key Vault
az keyvault secret list --vault-name homelab-bootstrap-kv -o table | grep -E 'github-pat|openai-api-key|argocd-token'

# Check ESO controller logs
kubectl logs -n external-secrets -l app.kubernetes.io/name=external-secrets --tail=100
```

### Platform API pods failing with "missing configuration"

```bash
# Verify the Deployment is using the correct Secret name
kubectl get deployment platform-api -n platform -o yaml | grep -A 5 envFrom

# Check Secret exists and has the right keys
kubectl get secret platform-api-secrets -n platform -o json | jq '.data | keys'

# Restart pods to pick up new secret
kubectl rollout restart deployment platform-api -n platform
```

### Need to rotate a secret

```bash
# Update the secret in Key Vault
az keyvault secret set --vault-name homelab-bootstrap-kv --name github-pat --value "new-value"

# ESO will sync within refreshInterval (1h)
# To force immediate sync:
kubectl annotate externalsecret platform-api-secrets -n platform force-sync="$(date +%s)" --overwrite

# Or delete the ExternalSecret (Argo CD will recreate it):
kubectl delete externalsecret platform-api-secrets -n platform
```

## Architecture Diagram

```text
┌──────────────────────────────────────────────────────────────────┐
│ Developer                                                         │
│ 1. Creates secrets in Azure Key Vault (one-time setup)          │
└───────────────────────────┬──────────────────────────────────────┘
                            │
┌───────────────────────────▼──────────────────────────────────────┐
│ Azure Key Vault (homelab-bootstrap-kv)                          │
│ - github-pat          (GitHub PAT with repo scope)              │
│ - openai-api-key      (OpenAI API key)                          │
│ - argocd-token        (Argo CD API token)                       │
└───────────────────────────┬──────────────────────────────────────┘
                            │ OIDC Federation (Workload Identity)
                            │ Managed Identity: id-homelab-aks-dev-eso
                            │ Role: Key Vault Secrets User
                            │
┌───────────────────────────▼──────────────────────────────────────┐
│ ClusterSecretStore: azure-bootstrap-kv                          │
│ (deployed at wave 3.5, references Key Vault)                   │
└───────────────────────────┬──────────────────────────────────────┘
                            │
┌───────────────────────────▼──────────────────────────────────────┐
│ ExternalSecret: platform-api-secrets                            │
│ (deployed at wave 10 with Platform API)                        │
│ - Watches Key Vault via ClusterSecretStore                     │
│ - Creates Kubernetes Secret: platform-api-secrets              │
└───────────────────────────┬──────────────────────────────────────┘
                            │
┌───────────────────────────▼──────────────────────────────────────┐
│ Secret: platform-api-secrets                                    │
│ - GITHUB_TOKEN: <value from github-pat>                         │
│ - OPENAI_API_KEY: <value from openai-api-key>                  │
│ - ARGOCD_TOKEN: <value from argocd-token>                      │
└───────────────────────────┬──────────────────────────────────────┘
                            │ envFrom.secretRef
                            │
┌───────────────────────────▼──────────────────────────────────────┐
│ Platform API Pod                                                │
│ - Environment variables available to application               │
│ - No code changes required                                     │
└──────────────────────────────────────────────────────────────────┘
```

## Testing the Integration

### 1. Test GitHub Integration

```bash
# The scaffold endpoint should create GitHub repos
curl -X POST http://platform-api.platform.svc.cluster.local:8080/api/v1/scaffold \
  -H "Authorization: Bearer <platform-api-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-service",
    "template": "go-service",
    "enableStorage": false,
    "enableVault": false
  }'

# Check GitHub for the new repository
# https://github.com/<username>/test-service
```

### 2. Test OpenAI Integration (Future)

```bash
# Once kagent/HolmesGPT are deployed, test AI operations
curl -X POST http://platform-api.platform.svc.cluster.local:8080/api/v1/agent/ask \
  -H "Authorization: Bearer <platform-api-token>" \
  -H "Content-Type: application/json" \
  -d '{"query": "What pods are failing in the platform namespace?"}'
```

### 3. Test Argo CD Integration (Future)

```bash
# Once app management endpoints are implemented
curl http://platform-api.platform.svc.cluster.local:8080/api/v1/apps \
  -H "Authorization: Bearer <platform-api-token>"
```

## Rollback Plan

If ExternalSecrets cause issues, you can temporarily rollback:

```bash
# 1. Restore the static secret
cd homelab-platform/platform/platform-api
mv secret.yaml.deprecated secret.yaml

# 2. Update kustomization.yaml
# Change: - externalsecrets
# To:     - secret.yaml

# 3. Populate the static secret with real values (NOT committed to Git)
kubectl edit secret platform-api-secrets -n platform

# 4. Commit and push
git add kustomization.yaml secret.yaml
git commit -m "Temporary rollback to static secrets"
git push
```

**Important:** This is a temporary workaround. Re-enable ExternalSecrets once issues are resolved.

## References

- [External Secrets Operator Documentation](https://external-secrets.io/)
- [Azure Key Vault Provider](https://external-secrets.io/latest/provider/azure-key-vault/)
- [Workload Identity Best Practices](https://azure.github.io/azure-workload-identity/docs/topics/best-practices.html)
- [GitHub PAT Scopes](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens)
- [Argo CD Service Accounts](https://argo-cd.readthedocs.io/en/stable/operator-manual/user-management/#local-usersaccounts-v15)
