# Monitoring Stack ExternalSecrets

This directory contains ExternalSecret resources that sync Grafana admin credentials from the bootstrap Azure Key Vault.

## Prerequisites

**Required Azure Key Vault secrets** (in the bootstrap Key Vault provisioned by Terraform):

| Key Vault Secret Name | Description | Example Value |
|----------------------|-------------|---------------|
| `grafana-admin-username` | Grafana admin username | `admin` |
| `grafana-admin-password` | Grafana admin password | `<strong-random-password>` |

## Setup Instructions

### 1. Create Secrets in Azure Key Vault

The ExternalSecret resources expect secrets to exist in the bootstrap Key Vault. You must create them before deploying the monitoring stack.

**Using Azure CLI:**

```bash
# Get Key Vault name from Terraform output
KEYVAULT_NAME=$(cd homelab-platform/infra && terraform output -raw keyvault_name)

# Create Grafana admin username
az keyvault secret set \
  --vault-name "$KEYVAULT_NAME" \
  --name "grafana-admin-username" \
  --value "admin"

# Create Grafana admin password (generate strong password)
GRAFANA_PASSWORD=$(openssl rand -base64 32)
az keyvault secret set \
  --vault-name "$KEYVAULT_NAME" \
  --name "grafana-admin-password" \
  --value "$GRAFANA_PASSWORD"

# Save password locally for initial login
echo "Grafana admin password: $GRAFANA_PASSWORD" > ~/.grafana-admin-password.txt
chmod 600 ~/.grafana-admin-password.txt
echo "Password saved to ~/.grafana-admin-password.txt"
```

**Using Azure Portal:**

1. Navigate to your bootstrap Key Vault
2. Go to **Secrets** → **Generate/Import**
3. Create two secrets:
   - Name: `grafana-admin-username`, Value: `admin`
   - Name: `grafana-admin-password`, Value: `<strong-password>`

### 2. Deploy the Monitoring Stack

Once the secrets exist in Key Vault, Argo CD will:

1. Deploy the ExternalSecret resources
2. ESO syncs secrets from Key Vault → creates `grafana-admin-creds` Secret in `monitoring` namespace
3. Grafana Helm chart references the Secret via `admin.existingSecret`

**Verify ExternalSecret sync:**

```bash
# Check ExternalSecret status
kubectl get externalsecret -n monitoring grafana-admin-creds

# Expected output:
# NAME                  STORE                  REFRESH INTERVAL   STATUS         READY
# grafana-admin-creds   bootstrap-keyvault     1h                 SecretSynced   True

# Check that the Secret was created
kubectl get secret -n monitoring grafana-admin-creds

# Verify secret contains expected keys
kubectl get secret -n monitoring grafana-admin-creds -o jsonpath='{.data}' | jq
# Should show: {"admin-password":"...", "admin-user":"..."}
```

## ExternalSecret Resources

### grafana-admin-creds.yaml

Syncs Grafana admin username and password from Azure Key Vault to a Kubernetes Secret.

**Source:** Azure Key Vault secrets
- `grafana-admin-username` → Secret key `admin-user`
- `grafana-admin-password` → Secret key `admin-password`

**Target:** Secret `grafana-admin-creds` in `monitoring` namespace

**Refresh:** Every 1 hour (configurable via `refreshInterval`)

**ClusterSecretStore:** `bootstrap-keyvault` (ESO with Workload Identity auth)

## Troubleshooting

### ExternalSecret shows "SecretSyncedError"

**Check ESO logs:**
```bash
kubectl logs -n external-secrets deployment/external-secrets -f
```

**Common causes:**
- Secret doesn't exist in Key Vault
- ESO managed identity lacks "Key Vault Secrets User" role
- Key Vault name mismatch in ClusterSecretStore

**Verify Key Vault access:**
```bash
# Get ESO identity client ID
ESO_CLIENT_ID=$(cd homelab-platform/infra && terraform output -raw eso_identity_client_id)

# Check role assignment
az role assignment list \
  --assignee "$ESO_CLIENT_ID" \
  --scope "/subscriptions/<subscription-id>/resourceGroups/<rg-name>/providers/Microsoft.KeyVault/vaults/<kv-name>" \
  --query "[?roleDefinitionName=='Key Vault Secrets User']"
```

### Grafana Pod CrashLoopBackOff

**Check if Secret exists:**
```bash
kubectl get secret -n monitoring grafana-admin-creds
```

If missing, the ExternalSecret failed to sync (see above).

**Check Grafana logs:**
```bash
kubectl logs -n monitoring deployment/monitoring-grafana
```

### Secret exists but Grafana login fails

**Verify Secret contents:**
```bash
# Decode admin username
kubectl get secret -n monitoring grafana-admin-creds -o jsonpath='{.data.admin-user}' | base64 -d
echo ""

# Decode admin password (CAREFUL: displays password)
kubectl get secret -n monitoring grafana-admin-creds -o jsonpath='{.data.admin-password}' | base64 -d
echo ""
```

Use the decoded values to log in to Grafana.

## Rotating Credentials

To rotate Grafana admin password:

**1. Update secret in Key Vault:**
```bash
KEYVAULT_NAME=$(cd homelab-platform/infra && terraform output -raw keyvault_name)
NEW_PASSWORD=$(openssl rand -base64 32)

az keyvault secret set \
  --vault-name "$KEYVAULT_NAME" \
  --name "grafana-admin-password" \
  --value "$NEW_PASSWORD"

echo "New password: $NEW_PASSWORD"
```

**2. Trigger ExternalSecret refresh:**
```bash
# Force immediate sync (instead of waiting for refreshInterval)
kubectl annotate externalsecret -n monitoring grafana-admin-creds \
  force-sync=$(date +%s) --overwrite
```

**3. Restart Grafana to pick up new password:**
```bash
kubectl rollout restart deployment -n monitoring monitoring-grafana
```

**4. Log in with new password.**

## Security Notes

- **Workload Identity:** ESO uses Azure Workload Identity (OIDC federation) — no static credentials
- **RBAC:** ESO managed identity has "Key Vault Secrets User" role (read-only)
- **Secret rotation:** Secrets auto-refresh every 1 hour; manual rotation via annotation
- **Namespace isolation:** Secret is created in `monitoring` namespace; not accessible from other namespaces

## Production Recommendations

1. **Use a strong password:** Generate with `openssl rand -base64 32` or password manager
2. **Enable MFA:** Configure Grafana LDAP/OAuth/SAML for SSO with MFA
3. **Reduce refresh interval:** For high-security environments, set `refreshInterval: 15m`
4. **Monitor ESO metrics:** Set up alerts on `externalsecret_sync_calls_error` metric
5. **Audit Key Vault access:** Enable Azure Monitor diagnostic logs for Key Vault

## References

- [External Secrets Operator Documentation](https://external-secrets.io/)
- [Azure Key Vault Provider](https://external-secrets.io/latest/provider/azure-key-vault/)
- [Grafana Security Best Practices](https://grafana.com/docs/grafana/latest/setup-grafana/configure-security/)
