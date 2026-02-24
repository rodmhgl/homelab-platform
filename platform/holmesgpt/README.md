# HolmesGPT

AI-powered root cause analysis for Kubernetes incidents using Anthropic Claude Sonnet 4.5.

## Overview

HolmesGPT is a CNCF Sandbox project that provides agentic troubleshooting for Kubernetes clusters. It analyzes cluster state, logs, metrics, and events to provide data-backed insights in plain language.

**Key capabilities:**
- Automatic investigation of critical/high alerts from Alertmanager
- Manual investigation via Platform API `/api/v1/investigate` endpoint
- Read-only cluster access (respects RBAC)
- Comprehensive context gathering (pods, logs, events, Crossplane Claims, Trivy CVEs, Gatekeeper violations)
- Prometheus metrics integration for performance data
- Evidence-based recommendations

## Architecture

**Integration flow:**
```
Alertmanager (critical/high alerts)
    ↓
HolmesGPT /api/investigate endpoint
    ↓
Investigation (Claude Sonnet 4.5 LLM + Kubernetes toolsets)
    ↓
Platform API (future: stores results for CLI/Portal retrieval)
```

**Deployment model:**
- **Namespace:** `holmesgpt`
- **Sync wave:** 12 (after monitoring wave 8, platform-api wave 10)
- **Replicas:** 1 (stateless FastAPI server)
- **Image:** `homelabplatformacr.azurecr.io/holmesgpt:v1.0.0` (built from source)
- **Service:** ClusterIP at `holmesgpt.holmesgpt.svc.cluster.local:5050`

## Configuration

HolmesGPT is configured via **environment variables** (not config files in Kubernetes deployments).

### LLM Configuration

```yaml
MODEL: anthropic/claude-sonnet-4-20250514
API_KEY: <from ExternalSecret holmesgpt-secrets>
```

**Why Anthropic Claude Sonnet 4.5:**
- Consistency with platform design (same LLM used by Platform API)
- Superior reasoning for complex troubleshooting scenarios
- Large 200K context window for comprehensive cluster analysis

### Enabled Toolsets

```yaml
ENABLED_BY_DEFAULT_TOOLSETS: kubernetes/core,kubernetes/logs,internet
```

**Available toolsets:**
- `kubernetes/core` — Describe/find resources (pods, deployments, services, nodes, etc.)
- `kubernetes/logs` — Read pod logs with filtering
- `internet` — Search documentation and knowledge bases
- `prometheus` — Query metrics (enabled via `PROMETHEUS_URL`)

### Prometheus Integration

```yaml
PROMETHEUS_URL: http://monitoring-kube-prometheus-prometheus.monitoring:9090
DISABLE_PROMETHEUS_TOOLSET: false
```

HolmesGPT can query Prometheus for CPU/memory usage, pod restarts, and other metrics to correlate with incidents.

## RBAC Permissions

HolmesGPT has **read-only cluster access** via ClusterRole:

**Core resources:** pods, logs, events, services, configmaps, namespaces, nodes, PVCs, PVs
**Workloads:** deployments, replicasets, statefulsets, daemonsets, jobs, cronjobs
**Crossplane:** Claims, XRs, Managed Resources (infrastructure context)
**Compliance:** VulnerabilityReports (Trivy), Constraints (Gatekeeper)
**Argo CD:** Applications, AppProjects (GitOps context)
**External Secrets:** ExternalSecrets, SecretStores (secrets management context)
**Metrics:** pods, nodes (metrics-server, if installed)

**NO write permissions** — safe for production use.

## Secrets Management

**API key storage:** Azure Key Vault (`anthropic-api-key` secret)
**Sync mechanism:** External Secrets Operator (ClusterSecretStore `azure-bootstrap-kv`)
**Refresh interval:** 1 hour
**Target secret:** `holmesgpt-secrets` in `holmesgpt` namespace

### Adding the Anthropic API Key

**Prerequisite:** Obtain API key from https://console.anthropic.com/

```bash
# Add to bootstrap Key Vault (one-time setup)
az keyvault secret set \
  --vault-name homelab-kv-dev \
  --name anthropic-api-key \
  --value "sk-ant-api03-YOUR_KEY_HERE"

# Verify sync
kubectl get externalsecret -n holmesgpt holmesgpt-secrets
kubectl get secret -n holmesgpt holmesgpt-secrets
```

## API Endpoints

HolmesGPT exposes a FastAPI server on port 5050:

### Health Endpoints

- `GET /healthz` — Liveness probe
- `GET /readyz` — Readiness probe (validates LLM model access)

### Investigation Endpoints

- `POST /api/investigate` — Trigger investigation (called by Alertmanager webhook)
  - Request: `{ "title": "...", "context": "..." }`
  - Response: `{ "investigation_id": "...", "status": "...", "result": "..." }`
  - Synchronous (blocks until investigation completes)

- `POST /api/stream/investigate` — Streaming investigation (SSE)
  - Same request format as `/api/investigate`
  - Returns Server-Sent Events for real-time progress

### Chat Endpoints (Future Use)

- `POST /api/chat` — General-purpose chat
- `POST /api/issue_chat` — Issue-specific conversation

## Alertmanager Webhook

**Configuration:** `platform/monitoring/values.yaml` (lines 178-182)

```yaml
receivers:
  - name: 'holmesgpt'
    webhook_configs:
      - url: 'http://holmesgpt.holmesgpt.svc.cluster.local:5050/api/investigate'
        send_resolved: true
```

**Trigger conditions:**
- Severity: `critical` (immediate investigation)
- Severity: `high` (investigation with continue=true, also goes to other receivers)

**Alertmanager payload format:**
```json
{
  "alerts": [
    {
      "labels": { "alertname": "...", "severity": "critical", ... },
      "annotations": { "summary": "...", "description": "..." },
      "status": "firing"
    }
  ]
}
```

HolmesGPT extracts alert context and initiates investigation automatically.

## Building the Docker Image

**Why build from source:** No official public Docker image exists at `ghcr.io/robusta-dev/holmesgpt`.

### Prerequisites

```bash
# Clone HolmesGPT repository
git clone https://github.com/robusta-dev/holmesgpt.git
cd holmesgpt

# Authenticate to homelab ACR
az acr login --name homelabplatformacr
```

### Build and Push

```bash
# Build multi-arch image (AMD64 + ARM64)
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --tag homelabplatformacr.azurecr.io/holmesgpt:v1.0.0 \
  --tag homelabplatformacr.azurecr.io/holmesgpt:latest \
  --push \
  .

# Verify push
docker pull homelabplatformacr.azurecr.io/holmesgpt:v1.0.0
```

### Image Details

**Base image:** `python:3.11-slim-bookworm`
**Entrypoint:** `uvicorn` FastAPI server (`server.py`)
**Includes:** kubectl, argocd CLI, kube-lineage (dependency graph tool)
**Size:** ~500MB (includes Python dependencies + Kubernetes tools)

## Deployment

### Via Argo CD (Recommended)

```bash
# Push manifests to Git
git add platform/holmesgpt/
git commit -m "feat(platform): install HolmesGPT for AI root cause analysis"
git push origin main

# Argo CD auto-syncs (wave 12)
# Wait for sync completion
kubectl get application -n argocd holmesgpt
```

### Manual Deployment (Testing)

```bash
# Apply manifests directly
kubectl apply -k platform/holmesgpt/

# Check deployment status
kubectl get pods -n holmesgpt
kubectl logs -n holmesgpt -l app.kubernetes.io/name=holmesgpt
```

## Verification

### 1. Pod Status

```bash
kubectl get pods -n holmesgpt
# Expected: 1/1 Running
```

### 2. ExternalSecret Sync

```bash
kubectl get externalsecret -n holmesgpt holmesgpt-secrets -o yaml
# Check status.conditions: type=Ready, status=True

kubectl get secret -n holmesgpt holmesgpt-secrets -o jsonpath='{.data.ANTHROPIC_API_KEY}' | base64 -d
# Should return: sk-ant-api03-...
```

### 3. Service DNS

```bash
kubectl run -it --rm debug --image=curlimages/curl --restart=Never -- \
  curl -v http://holmesgpt.holmesgpt.svc.cluster.local:5050/healthz

# Expected: HTTP 200 OK, {"status": "healthy"}
```

### 4. Readiness Probe (Validates LLM Access)

```bash
kubectl run -it --rm debug --image=curlimages/curl --restart=Never -- \
  curl -v http://holmesgpt.holmesgpt.svc.cluster.local:5050/readyz

# Expected: HTTP 200 OK, {"status": "ready", "models": [...]}
```

### 5. Manual Investigation Test

```bash
kubectl port-forward -n holmesgpt svc/holmesgpt 5050:5050

# In another terminal
curl -X POST http://localhost:5050/api/investigate \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Test Investigation",
    "context": "Platform API pods are restarting frequently"
  }'

# Expected: JSON response with investigation results
```

### 6. Alertmanager Webhook Test

```bash
# Port-forward to Alertmanager
kubectl port-forward -n monitoring svc/monitoring-alertmanager 9093:9093

# Access http://localhost:9093
# Create a test alert with severity: critical
# Check HolmesGPT logs for webhook receipt

kubectl logs -n holmesgpt -l app.kubernetes.io/name=holmesgpt --tail=50 -f
# Look for: "Received /api/investigate request: title=..."
```

## Troubleshooting

### Pod Not Starting

**Symptom:** Pod stuck in `ImagePullBackOff` or `ErrImagePull`

**Cause:** Image not built or not pushed to ACR

**Solution:**
```bash
# Verify image exists
az acr repository show-tags --name homelabplatformacr --repository holmesgpt

# If missing, build and push image (see "Building the Docker Image" above)
```

### 401 Unauthorized from Anthropic API

**Symptom:** Readiness probe fails, logs show `AuthenticationError`

**Cause:** Invalid or missing API key in Key Vault

**Solution:**
```bash
# Check Key Vault secret
az keyvault secret show --vault-name homelab-kv-dev --name anthropic-api-key

# If wrong/missing, update it
az keyvault secret set \
  --vault-name homelab-kv-dev \
  --name anthropic-api-key \
  --value "sk-ant-api03-YOUR_CORRECT_KEY"

# Force ExternalSecret refresh
kubectl delete secret -n holmesgpt holmesgpt-secrets
# ESO will recreate it automatically

# Restart pod
kubectl rollout restart deployment -n holmesgpt holmesgpt
```

### ExternalSecret Not Syncing

**Symptom:** `kubectl get externalsecret -n holmesgpt holmesgpt-secrets` shows `Ready=False`

**Cause:** ESO ServiceAccount lacks permissions or ClusterSecretStore misconfigured

**Solution:**
```bash
# Check ESO logs
kubectl logs -n external-secrets -l app.kubernetes.io/name=external-secrets

# Verify ClusterSecretStore exists
kubectl get clustersecretstore azure-bootstrap-kv -o yaml

# Check ESO ServiceAccount has Workload Identity annotation
kubectl get sa -n external-secrets external-secrets -o yaml
# Look for: azure.workload.identity/client-id annotation

# Verify Key Vault RBAC (ESO identity needs "Key Vault Secrets User" role)
az role assignment list --assignee <eso-identity-client-id> --scope /subscriptions/.../resourceGroups/.../providers/Microsoft.KeyVault/vaults/homelab-kv-dev
```

### Investigations Timing Out

**Symptom:** `/api/investigate` requests return 500 errors, logs show timeout

**Cause:** LLM taking too long or tool execution exceeding limits

**Solution:**
```bash
# Increase timeout (edit deployment.yaml)
# Change: LLM_REQUEST_TIMEOUT: "600" → "900"

# Apply changes
kubectl apply -k platform/holmesgpt/

# Alternative: Reduce investigation scope
# Edit: ENABLED_BY_DEFAULT_TOOLSETS (disable prometheus if not needed)
```

### Alertmanager Webhook Failures

**Symptom:** Alertmanager logs show webhook delivery errors

**Cause:** Service DNS name mismatch or port incorrect

**Solution:**
```bash
# Verify Service exists and port is 5050
kubectl get svc -n holmesgpt holmesgpt -o yaml

# Test DNS resolution from monitoring namespace
kubectl run -it --rm debug -n monitoring --image=curlimages/curl --restart=Never -- \
  nslookup holmesgpt.holmesgpt.svc.cluster.local

# Test HTTP connectivity
kubectl run -it --rm debug -n monitoring --image=curlimages/curl --restart=Never -- \
  curl -v http://holmesgpt.holmesgpt.svc.cluster.local:5050/healthz
```

### High Memory Usage

**Symptom:** Pod OOMKilled, restarts frequently

**Cause:** Large investigations with many tool calls

**Solution:**
```bash
# Increase memory limit (edit deployment.yaml)
# Change: limits.memory: "2Gi" → "4Gi"

# Reduce tool memory limit (reduces parallelism)
# Change: TOOL_MEMORY_LIMIT_MB: "1500" → "1000"

# Apply changes
kubectl apply -k platform/holmesgpt/
```

## Performance Tuning

### Investigation Speed

**Fast investigations:**
- Disable Prometheus toolset if metrics not needed
- Use streaming endpoint (`/api/stream/investigate`) for real-time feedback
- Reduce `TOOL_MAX_ALLOCATED_CONTEXT_WINDOW_PCT` to limit tool output verbosity

### Cost Optimization

**Reduce LLM API costs:**
- Set `TEMPERATURE: "0.00000001"` (deterministic, fewer retry calls)
- Enable `fast_model` for tool output summarization (edit env: `FAST_MODEL: "anthropic/claude-haiku-4-5-20241022"`)
- Limit enabled toolsets to only what's needed

### Resource Limits

**Current settings:**
- CPU: 200m request, 1000m limit
- Memory: 512Mi request, 2Gi limit

**For high-volume environments:**
- Increase replicas to 2+ (stateless, scales horizontally)
- Add HPA based on CPU/memory usage
- Consider separate HolmesGPT instance per cluster region

## Integration with Platform API (Task #52)

**Next steps after HolmesGPT deployment:**

1. **Platform API HTTP client** (`api/internal/investigate/client.go`)
   - Call `POST http://holmesgpt.holmesgpt:5050/api/investigate`
   - Parse investigation results
   - Store in investigation store (in-memory or persistent)

2. **Platform API endpoints** (`api/internal/investigate/handler.go`)
   - `POST /api/v1/investigate` — Create investigation
   - `GET /api/v1/investigate/:id` — Get investigation status/results
   - `GET /api/v1/investigate` — List investigations

3. **CLI integration** (Task #75: `rdp investigate`)
   - Interactive app selector (bubbletea TUI)
   - Issue description prompt
   - Real-time status polling
   - Display investigation findings + recommendations

4. **Portal UI integration** (AI Ops panel, already complete)
   - Manual investigation trigger form
   - Investigation history table
   - Results viewer with findings/recommendations

## Sources

- [HolmesGPT GitHub Repository](https://github.com/robusta-dev/holmesgpt)
- [CNCF Blog: HolmesGPT Agentic Troubleshooting](https://www.cncf.io/blog/2026/01/07/holmesgpt-agentic-troubleshooting-built-for-the-cloud-native-era/)
- [Robusta Documentation: Kubernetes Toolsets](https://docs.robusta.dev/master/configuration/holmesgpt/toolsets/kubernetes.html)
- [Anthropic Claude Models](https://docs.anthropic.com/en/docs/models-overview)
