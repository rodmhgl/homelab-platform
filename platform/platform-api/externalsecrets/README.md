# Platform API External Secrets

This directory contains ExternalSecret resources that sync secrets from the bootstrap Azure Key Vault to Kubernetes Secrets consumed by the Platform API.

## Architecture

```text
┌────────────────────────────────────────────────────────┐
│ Azure Key Vault (Bootstrap)                            │
│ - github-pat         (GitHub personal access token)    │
│ - openai-api-key     (OpenAI API key)                  │
│ - argocd-token       (Argo CD API token)               │
└───────────────────────┬────────────────────────────────┘
                        │ OIDC Federation (Workload Identity)
                        │ via ClusterSecretStore: azure-bootstrap-kv
                        │
┌───────────────────────▼────────────────────────────────┐
│ ExternalSecret: platform-api-secrets                   │
│ (watches Key Vault, creates Kubernetes Secret)         │
└───────────────────────┬────────────────────────────────┘
                        │
┌───────────────────────▼────────────────────────────────┐
│ Secret: platform-api-secrets                           │
│ - GITHUB_TOKEN                                         │
│ - OPENAI_API_KEY                                       │
│ - ARGOCD_TOKEN                                         │
└───────────────────────┬────────────────────────────────┘
                        │ envFrom in Deployment
                        │
┌───────────────────────▼────────────────────────────────┐
│ Platform API Pod                                       │
│ (environment variables available to application)       │
└────────────────────────────────────────────────────────┘
```

## Prerequisites

1. **External Secrets Operator installed** — Deployed at wave 3.5 via `platform/external-secrets/`
2. **ClusterSecretStore configured** — `azure-bootstrap-kv` pointing to the bootstrap Key Vault
3. **Azure Key Vault secrets provisioned** — See "Provisioning Secrets" below

## Required Key Vault Secrets

The following secrets must exist in the bootstrap Azure Key Vault before the ExternalSecret will sync successfully:

| Key Vault Secret Name | Environment Variable | Purpose | How to Obtain |
|----------------------|---------------------|---------|---------------|
| `github-pat` | `GITHUB_TOKEN` | GitHub API access for repo creation | Create at https://github.com/settings/tokens (scope: `repo`) |
| `openai-api-key` | `OPENAI_API_KEY` | OpenAI API for AI operations | Create at https://platform.openai.com/api-keys |
| `argocd-token` | `ARGOCD_TOKEN` | Argo CD API for app management | Generate via `argocd account generate-token --account platform-api` |

## Provisioning Secrets

### 1. GitHub Personal Access Token

```bash
# Create a PAT at https://github.com/settings/tokens with 'repo' scope
# Then set it in Key Vault:
az keyvault secret set \
  --vault-name homelab-bootstrap-kv \
  --name github-pat \
  --value "ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
```

**Required scopes:** `repo` (full control of private repositories)

### 2. OpenAI API Key

```bash
# Create an API key at https://platform.openai.com/api-keys
# Then set it in Key Vault:
az keyvault secret set \
  --vault-name homelab-bootstrap-kv \
  --name openai-api-key \
  --value "sk-proj-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
```

### 3. Argo CD API Token

```bash
# First, create a ServiceAccount in Argo CD for the Platform API
kubectl -n argocd patch configmap argocd-cm --type merge -p '
data:
  accounts.platform-api: apiKey
'

# Grant appropriate permissions (adjust RBAC as needed)
kubectl -n argocd patch configmap argocd-rbac-cm --type merge -p '
data:
  policy.csv: |
    p, role:platform-api, applications, *, */*, allow
    p, role:platform-api, applicationsets, *, */*, allow
    g, platform-api, role:platform-api
'

# Restart Argo CD server to pick up the config changes
kubectl -n argocd rollout restart deployment argocd-server

# Generate the token (expires after 90 days by default)
argocd login <argocd-server-url> --username admin --password <admin-password>
argocd account generate-token --account platform-api

# Set it in Key Vault
az keyvault secret set \
  --vault-name homelab-bootstrap-kv \
  --name argocd-token \
  --value "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

## Verification

### 1. Check ExternalSecret Status

```bash
# Check if ExternalSecret is syncing correctly
kubectl get externalsecret platform-api-secrets -n platform
kubectl describe externalsecret platform-api-secrets -n platform

# Expected status: SecretSynced (Ready: True)
```

### 2. Verify Kubernetes Secret Created

```bash
# Check that the Secret was created by ESO
kubectl get secret platform-api-secrets -n platform
kubectl describe secret platform-api-secrets -n platform

# Verify the keys exist (don't print values!)
kubectl get secret platform-api-secrets -n platform -o jsonpath='{.data}' | jq 'keys'
# Expected: ["ARGOCD_TOKEN", "GITHUB_TOKEN", "OPENAI_API_KEY"]
```

### 3. Test Platform API Access

```bash
# Check Platform API logs for any auth errors
kubectl logs -n platform -l app.kubernetes.io/name=platform-api --tail=50

# Test scaffold endpoint (requires GITHUB_TOKEN)
curl -X POST http://platform-api.platform.svc.cluster.local:8080/api/v1/scaffold \
  -H "Authorization: Bearer <platform-api-token>" \
  -H "Content-Type: application/json" \
  -d '{"name": "test-service", "template": "go-service", ...}'
```

## Troubleshooting

### ExternalSecret stuck in SecretSyncedError

**Symptoms:**
```bash
$ kubectl describe externalsecret platform-api-secrets -n platform
Status:
  Conditions:
    Type:    SecretSyncedError
    Status:  False
    Reason:  SecretNotFound
```

**Common causes:**

1. **Key Vault secret doesn't exist** — Check `remoteRef.key` matches the actual secret name in Key Vault:
   ```bash
   az keyvault secret list --vault-name homelab-bootstrap-kv -o table
   ```

2. **ClusterSecretStore not ready** — Verify ESO has access to Key Vault:
   ```bash
   kubectl get clustersecretstore azure-bootstrap-kv
   kubectl describe clustersecretstore azure-bootstrap-kv
   ```

3. **ESO identity missing Key Vault RBAC** — Check the managed identity has "Key Vault Secrets User" role:
   ```bash
   # Should be provisioned by Terraform
   az role assignment list --scope /subscriptions/<sub-id>/resourceGroups/<rg>/providers/Microsoft.KeyVault/vaults/homelab-bootstrap-kv
   ```

### Secret not updating after Key Vault change

ExternalSecrets refresh on the interval specified by `refreshInterval` (default: 1h). To force an immediate sync:

```bash
# Delete and recreate the ExternalSecret
kubectl delete externalsecret platform-api-secrets -n platform
# Argo CD will automatically recreate it (selfHeal: true)
```

Or trigger a manual refresh:
```bash
kubectl annotate externalsecret platform-api-secrets -n platform \
  force-sync="$(date +%s)" --overwrite
```

### Platform API pod can't read environment variables

**Symptoms:** Platform API logs show "missing required configuration"

**Verify the Deployment references the Secret:**
```bash
kubectl get deployment platform-api -n platform -o yaml | grep -A 5 envFrom
# Should include:
#   envFrom:
#     - secretRef:
#         name: platform-api-secrets
```

**Check Secret exists and has correct keys:**
```bash
kubectl get secret platform-api-secrets -n platform -o json | jq '.data | keys'
```

## Files

```text
platform/platform-api/externalsecrets/
├── README.md                      # This file
├── kustomization.yaml             # Kustomize config for ExternalSecrets
└── platform-api-secrets.yaml     # ExternalSecret for all Platform API secrets
```

## Migration from Static Secrets

The previous implementation used a static `Secret` resource (`secret.yaml`) with placeholder values. That file has been deprecated and renamed to `secret.yaml.deprecated`.

**Migration steps:**
1. ✅ ExternalSecret resources created (this directory)
2. ✅ Kustomization updated to include `externalsecrets/` instead of `secret.yaml`
3. ⬜ Provision secrets in Azure Key Vault (see "Provisioning Secrets" above)
4. ⬜ Argo CD sync will create the ExternalSecret and corresponding Secret
5. ⬜ Platform API pods will automatically pick up the new Secret (via envFrom)

## Security Considerations

- **Zero secrets in Git** — All sensitive values live only in Azure Key Vault
- **Automatic rotation** — Update secrets in Key Vault; ESO syncs within `refreshInterval`
- **Workload Identity** — No static credentials; OIDC federation via Azure AD
- **Namespace isolation** — ExternalSecrets create Secrets in their own namespace only
- **Owner deletion policy** — ESO owns the Secret; deleting the ExternalSecret also deletes the Secret
- **Retain deletion policy** — If ExternalSecret is accidentally deleted, Secret is retained (safety measure)

## Next Steps

1. **Provision all three secrets** in the bootstrap Key Vault (see "Provisioning Secrets")
2. **Verify ExternalSecret sync** — Check `kubectl get externalsecret -n platform`
3. **Deploy Platform API** — Argo CD will sync the updated configuration
4. **Test API endpoints** — Verify GitHub/OpenAI/Argo CD integrations work

## References

- [External Secrets Operator Docs](https://external-secrets.io/)
- [Azure Key Vault Provider](https://external-secrets.io/latest/provider/azure-key-vault/)
- [Workload Identity Best Practices](https://azure.github.io/azure-workload-identity/docs/topics/best-practices.html)
