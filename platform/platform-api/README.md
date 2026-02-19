# Platform API — Kubernetes Manifests

Kubernetes manifests for deploying the Platform API to the AKS cluster via Argo CD.

## Overview

The Platform API is the central nervous system of the IDP, deployed as:

- **Namespace:** `platform` (created by Argo CD)
- **Deployment:** 2 replicas with pod topology spread
- **Service:** ClusterIP on port 80 → 8080
- **RBAC:** ClusterRole with read access to Crossplane Claims, Argo CD Applications, Gatekeeper policies, Trivy reports, ESO secrets
- **Configuration:** ConfigMap (non-sensitive) + Secret (sensitive tokens)

## Prerequisites

Before deploying, you need:

1. **Container image** — Build and push the Platform API image to ACR:
   ```bash
   cd homelab-platform/api
   make docker-build VERSION=v0.1.0
   docker tag homelab/platform-api:v0.1.0 <ACR_LOGIN_SERVER>/platform-api:v0.1.0
   az acr login --name <ACR_NAME>
   docker push <ACR_LOGIN_SERVER>/platform-api:v0.1.0
   ```

2. **Update `deployment.yaml`** — Replace `<ACR_LOGIN_SERVER>` with your actual ACR hostname (from Terraform output `acr_login_server`)

3. **Populate secrets** — Update `secret.yaml` or (recommended) create an ExternalSecret that fetches from Azure Key Vault:
   - `ARGOCD_TOKEN` — Generate via `argocd account generate-token --account platform-api`
   - `GITHUB_TOKEN` — GitHub PAT with `repo` scope
   - `OPENAI_API_KEY` — OpenAI API key (for AI operations)

## Deployment

This Application is deployed via the root App of Apps (`platform-root`). Argo CD auto-discovers it via `platform/platform-api/application.yaml`.

```bash
# Commit manifests to Git
git add platform/platform-api/
git commit -m "Add Platform API Kubernetes manifests"
git push origin main

# Argo CD will auto-sync (or trigger manually)
argocd app sync platform-api
```

## Configuration

### ConfigMap (`configmap.yaml`)

Non-sensitive configuration:
- Server: `PORT`, `LOG_LEVEL`, `SHUTDOWN_TIMEOUT`
- Kubernetes: `IN_CLUSTER=true`
- Argo CD: `ARGOCD_SERVER_URL`
- GitHub: `GITHUB_ORG`, `PLATFORM_REPO`
- AI: `KAGENT_NAMESPACE`, `HOLMESGPT_URL`

### Secret (`secret.yaml`)

Sensitive tokens (TODO: migrate to External Secrets Operator):
- `ARGOCD_TOKEN`
- `GITHUB_TOKEN`
- `OPENAI_API_KEY`

## RBAC Permissions

The Platform API has a ClusterRole with **read-only** access to:
- Crossplane: Claims, Composite Resources, Managed Resources
- Argo CD: Applications
- Gatekeeper: ConstraintTemplates, Constraints, audit violations
- Trivy Operator: VulnerabilityReports
- External Secrets: ExternalSecrets, SecretStores
- Kubernetes core: Secrets (for connection secrets), Events, Pods, Namespaces

**Write operations** (create, update, delete):
- kagent: Agents and Tasks (for `/api/v1/agent/ask`)
- **No direct writes to Claims or Applications** — the API commits YAML to Git; Argo CD syncs

## Health Checks

- **Liveness:** `GET /health` — returns 200 OK if the service is running
- **Readiness:** `GET /ready` — returns 200 OK when K8s API is reachable (TODO: add Argo CD API check)

## Resource Limits

Per pod:
- **Requests:** 100m CPU, 128Mi memory
- **Limits:** 500m CPU, 512Mi memory

## Security

- Non-root user (UID 1000)
- Read-only root filesystem
- Dropped all capabilities
- seccomp profile: `RuntimeDefault`
- Pod Security Standard: `restricted` (satisfies Gatekeeper policies)

## Sync Wave

**Wave 10** — deploys after:
- Argo CD (wave 0)
- Crossplane (waves 1-3)
- Gatekeeper (waves 4-6)
- ESO (wave 7)

## Troubleshooting

```bash
# Check pod status
kubectl get pods -n platform -l app.kubernetes.io/name=platform-api

# View logs
kubectl logs -n platform -l app.kubernetes.io/name=platform-api --tail=100 -f

# Check Argo CD sync status
argocd app get platform-api

# Test health endpoint
kubectl port-forward -n platform svc/platform-api 8080:80
curl http://localhost:8080/health
```

## TODO

- [ ] Replace `secret.yaml` with ExternalSecret (fetch from bootstrap Key Vault)
- [ ] Add HorizontalPodAutoscaler
- [ ] Add PodDisruptionBudget
- [ ] Add NetworkPolicy (restrict egress to Argo CD API, GitHub API, K8s API)
- [ ] Implement actual readiness check (test Argo CD API connectivity)
- [ ] Add ServiceMonitor for Prometheus metrics
