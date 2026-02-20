# External Secrets Operator

External Secrets Operator (ESO) syncs secrets from the bootstrap Azure Key Vault into Kubernetes `Secret` resources. This provides platform-level secret management for LLM API keys, GitHub tokens, and other credentials needed by platform components.

## Architecture

```text
┌─────────────────────────────────────────────────────────────┐
│ Azure Key Vault (Bootstrap)                                 │
│ - LLM API keys (OpenAI, Anthropic)                          │
│ - GitHub personal access token                              │
│ - Argo CD admin password                                    │
└────────────────┬────────────────────────────────────────────┘
                 │ OIDC Federation (Workload Identity)
                 │
┌────────────────▼────────────────────────────────────────────┐
│ ESO Controller Pod                                          │
│ ServiceAccount: external-secrets                            │
│ Annotation: azure.workload.identity/client-id               │
└────────────────┬────────────────────────────────────────────┘
                 │ Watches ExternalSecret CRDs
                 │
┌────────────────▼────────────────────────────────────────────┐
│ ExternalSecret → Kubernetes Secret                          │
│ (per-namespace, references ClusterSecretStore)              │
└─────────────────────────────────────────────────────────────┘
```

## Prerequisites

1. **Terraform infrastructure applied** — The following resources must exist:
   - Bootstrap Key Vault (`homelab-bootstrap-kv`)
   - ESO managed identity (`id-{cluster}-eso`)
   - Federated credential trusting `system:serviceaccount:external-secrets:external-secrets`
   - RBAC: Key Vault Secrets User role assigned to the managed identity

2. **Argo CD installed** — ESO is deployed via Argo CD Application (wave 3.5)

## Installation Steps

### 1. Replace Placeholders

After Terraform apply completes, retrieve the outputs and replace the placeholders in this directory:

```bash
# Get Terraform outputs
cd homelab-platform/infra
terraform output -raw eso_identity_client_id
terraform output -raw keyvault_uri

# Edit values.yaml — replace REPLACE_WITH_ESO_IDENTITY_CLIENT_ID
# Edit clustersecretstore.yaml — replace REPLACE_WITH_KEYVAULT_URI
```

**values.yaml:**

```yaml
serviceAccount:
  annotations:
    azure.workload.identity/client-id: "abc12345-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
```

**clustersecretstore.yaml:**

```yaml
spec:
  provider:
    azurekv:
      vaultUrl: "https://homelab-bootstrap-kv.vault.azure.net/"
```

### 2. Commit and Push

Argo CD will automatically sync the Application (wave 3.5) after Crossplane config (wave 3) completes.

```bash
git add homelab-platform/platform/external-secrets/
git commit -m "Configure ESO with Terraform outputs"
git push
```

### 3. Verify Installation

```bash
# Check ESO controller pod
kubectl get pods -n external-secrets

# Check ClusterSecretStore status
kubectl get clustersecretstore azure-bootstrap-kv
kubectl describe clustersecretstore azure-bootstrap-kv

# Verify Workload Identity annotation
kubectl get sa external-secrets -n external-secrets -o yaml | grep azure.workload.identity
```

## Usage

### Creating an ExternalSecret

ExternalSecrets reference the `ClusterSecretStore` and specify which Key Vault secrets to sync:

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: github-token
  namespace: platform-api
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: azure-bootstrap-kv
    kind: ClusterSecretStore
  target:
    name: github-token
    creationPolicy: Owner
  data:
    - secretKey: token
      remoteRef:
        key: github-pat
```

This creates a Kubernetes Secret named `github-token` in the `platform-api` namespace with a key `token` containing the value from Key Vault secret `github-pat`.

### Adding Secrets to Key Vault

```bash
# Set a secret in the bootstrap Key Vault
az keyvault secret set \
  --vault-name homelab-bootstrap-kv \
  --name github-pat \
  --value "ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"

# ESO will sync it within refreshInterval (default: 1h)
# To force immediate sync, delete the ExternalSecret and recreate it
```

## Troubleshooting

### Deployment fails with "field not declared in schema"

**Error:** `.spec.template.spec.containers[].securityContext.fsGroup: field not declared in schema`

**Cause:** The `fsGroup` field belongs in `podSecurityContext` (pod-level), not `securityContext` (container-level).

**Solution:** Use the corrected values.yaml structure:

```yaml
podSecurityContext:
  enabled: true
  fsGroup: 1000

securityContext:
  enabled: true
  runAsNonRoot: true
  runAsUser: 1000
  # ... other container-level settings
```

### ClusterSecretStore shows NotReady

Check the ESO controller logs:

```bash
kubectl logs -n external-secrets -l app.kubernetes.io/name=external-secrets
```

Common issues:

- **403 Forbidden**: Managed identity doesn't have Key Vault Secrets User role
- **Invalid client ID**: `azure.workload.identity/client-id` annotation is incorrect
- **OIDC token not injected**: Pod label `azure.workload.identity/use: "true"` is missing

### ExternalSecret stuck in SecretSyncedError

```bash
kubectl describe externalsecret <name> -n <namespace>
```

Common issues:

- Key Vault secret doesn't exist
- Secret name mismatch (check `remoteRef.key`)
- ClusterSecretStore is NotReady

## Files

```text
platform/external-secrets/
├── README.md                    # This file
├── application.yaml             # Argo CD Application (wave 3.5)
├── values.yaml                  # Helm values for ESO chart
└── clustersecretstore.yaml      # ClusterSecretStore for bootstrap Key Vault
```

## Next Steps

After ESO is running:

1. **Task #40**: Create ExternalSecret resources for LLM API keys (kagent, HolmesGPT)
2. **Task #87**: Create ExternalSecret for Platform API secrets (GitHub token)
3. Add secrets to bootstrap Key Vault via `az keyvault secret set`

## References

- [External Secrets Operator Docs](https://external-secrets.io/)
- [Azure Key Vault Provider](https://external-secrets.io/latest/provider/azure-key-vault/)
- [Azure Workload Identity](https://azure.github.io/azure-workload-identity/)
