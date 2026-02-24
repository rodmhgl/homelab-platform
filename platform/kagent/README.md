# kagent — Natural Language Cluster Queries

**Status:** ✅ Installed (v0.7.0)
**Wave:** 13 (after HolmesGPT)
**Namespace:** `kagent-system`

## Overview

kagent is a Kubernetes-native AI agent framework that enables natural language interaction with the cluster. It provides a conversational interface for exploring cluster state, troubleshooting issues, and understanding platform architecture.

**Key Features:**
- **Agent CRDs:** Pre-configured AI assistants with platform-specific knowledge
- **Task CRDs:** Ephemeral query objects for natural language questions
- **Read-only RBAC:** Safe cluster introspection without mutation risk
- **Anthropic Claude Sonnet 4.5:** Advanced reasoning for complex queries

## Architecture

```
Platform API (/api/v1/agent/ask)
  ↓ creates
Task CRD (ephemeral)
  ↓ references
Agent CRD (platform-agent)
  ↓ uses
Anthropic Provider (Claude Sonnet 4.5)
  ↓ queries
Kubernetes API (read-only)
  ↓ returns
Structured response
```

**Integration Points:**
- **Platform API:** Task #53 — `POST /api/v1/agent/ask` creates Task CRDs, streams responses
- **CLI:** Task #76 — `rdp ask <query>` wraps Platform API calls
- **Portal UI:** AI Operations panel (#86) — web-based chat interface

## Default Agent: `platform-agent`

The `platform-agent` Agent CRD is pre-configured with comprehensive platform knowledge:

**Model Configuration:**
- Model: `claude-sonnet-4-5-20250929`
- Temperature: `0.1` (deterministic for factual queries)
- Max tokens: `4096`

**Platform Context Includes:**
- Argo CD GitOps architecture (App of Apps, ApplicationSets)
- Crossplane self-service infrastructure (StorageBucket, Vault XRDs)
- Compliance scoring formula (Gatekeeper + Trivy + Falco)
- Secrets management patterns (ESO + Workload Identity)
- Container registry requirements (homelabplatformacr.azurecr.io)

**Query Capabilities:**
- Application health/sync status
- Infrastructure provisioning status (Crossplane resource trees)
- Compliance violations (Gatekeeper audit)
- CVE vulnerabilities (Trivy reports)
- Runtime security events (Falco via Platform API)

## RBAC Scope

kagent operates with **read-only cluster access** via the `kagent-reader` ClusterRole:

**Allowed Operations:**
- `get`, `list`, `watch` on core resources (Pods, Services, Deployments, etc.)
- `get`, `list` on Secrets (metadata only, no data access)
- `get`, `list`, `watch` on Crossplane resources (Claims, XRs, Managed Resources)
- `get`, `list`, `watch` on compliance resources (Gatekeeper, Trivy, Falco)
- `get`, `list`, `watch` on GitOps resources (Argo CD Applications)

**Prohibited Operations:**
- No `create`, `update`, `delete`, `patch` on any resources
- No exec into Pods
- No port-forwarding

**Rationale:** Natural language queries should be read-only introspection. Mutations go through validated Platform API GitOps workflows.

## Usage Examples

### Manual Task Creation (Testing)

```bash
# Create a Task CRD
cat <<EOF | kubectl apply -f -
apiVersion: kagent.dev/v1alpha1
kind: Task
metadata:
  name: test-query-001
  namespace: kagent-system
spec:
  agentRef:
    name: platform-agent
  prompt: "List all unhealthy pods in the platform namespace"
EOF

# Watch Task status
kubectl get task -n kagent-system test-query-001 -w
# Expected: STATUS transitions Pending → Running → Completed

# Retrieve response
kubectl get task -n kagent-system test-query-001 -o jsonpath='{.status.response}' | jq .

# Cleanup
kubectl delete task -n kagent-system test-query-001
```

### Via Platform API (Task #53)

```bash
# POST /api/v1/agent/ask
curl -X POST http://platform-api.platform/api/v1/agent/ask \
  -H "Authorization: Bearer homelab-portal-token" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "Why is the portal-ui deployment unhealthy?"
  }'

# Response (Server-Sent Events stream)
# {"status": "running", "message": "Analyzing cluster state..."}
# {"status": "completed", "response": "The portal-ui deployment shows 0/2 replicas ready..."}
```

### Via CLI (Task #76)

```bash
# rdp ask <natural language query>
rdp ask "Show me all Gatekeeper policy violations"
rdp ask "What CVEs are in the platform-api image?"
rdp ask "List all Crossplane Claims with connection secret errors"
rdp ask "Why is my app failing to deploy?"
```

### Via Portal UI (Already Implemented)

Navigate to **AI Operations** panel at `http://portal.rdp.azurelaboratory.com`:
- Enter natural language queries in chat interface
- View streaming responses with Kubernetes resource citations
- Trigger HolmesGPT investigations for incidents

## Example Queries kagent Can Handle

**Application Health:**
- "Why is my app unhealthy?"
- "Show me all failing pods in production"
- "What's the sync status of the platform-api Application?"

**Infrastructure:**
- "List all Crossplane Claims in the workloads namespace"
- "Show me the resource tree for storage-bucket-demo"
- "What Azure resources were created by Crossplane today?"

**Compliance:**
- "List all Gatekeeper policy violations"
- "Show me deployments missing CPU limits"
- "Which images are using the :latest tag?"

**Security:**
- "What CVEs are in the portal-ui deployment?"
- "List all CRITICAL vulnerabilities across the cluster"
- "Are there any Falco security alerts?" (via Platform API)

**GitOps:**
- "Which Argo CD apps are OutOfSync?"
- "Show me recent deployment failures"
- "What's the last deployed version of platform-api?"

## Secrets Management

**ExternalSecret Resource:**
- Name: `kagent-secrets`
- Target Secret: `kagent-anthropic-secret`
- Key Vault Secret: `anthropic-api-key` (shared with HolmesGPT)
- Refresh Interval: 1 hour

**Verification:**
```bash
# Check ExternalSecret sync status
kubectl get externalsecret -n kagent-system kagent-secrets
# Expected: STATUS=SecretSynced, READY=True

# Verify target secret exists
kubectl get secret -n kagent-system kagent-anthropic-secret
# Expected: DATA=1 (apiKey key present)
```

## Monitoring

**Controller Logs:**
```bash
# View kagent controller logs
kubectl logs -n kagent-system -l app.kubernetes.io/name=kagent --tail=100 -f
```

**Metrics (Prometheus):**
- ServiceMonitor enabled in `monitoring` namespace
- Metrics interval: 30s
- Key metrics:
  - `kagent_task_total` — Total tasks created
  - `kagent_task_duration_seconds` — Task completion time
  - `kagent_provider_requests_total` — LLM API calls
  - `kagent_provider_errors_total` — LLM API errors

**Task CRD Lifecycle:**
```bash
# List all tasks (including completed)
kubectl get tasks -n kagent-system

# View task details
kubectl describe task -n kagent-system <task-name>

# Delete completed tasks (manual cleanup)
kubectl delete tasks -n kagent-system --field-selector status.completed=true
```

## Troubleshooting

### Task Stuck in Pending

**Symptoms:** Task CRD created but never transitions to Running

**Diagnosis:**
```bash
# Check controller logs
kubectl logs -n kagent-system -l app.kubernetes.io/name=kagent --tail=50

# Check Agent status
kubectl get agent -n kagent-system platform-agent -o yaml
# Look for status.conditions
```

**Common Causes:**
1. **Missing API key:** ExternalSecret not synced
   ```bash
   kubectl get externalsecret -n kagent-system kagent-secrets
   kubectl describe externalsecret -n kagent-system kagent-secrets
   ```

2. **Provider not ready:** Check Provider CRD status
   ```bash
   kubectl get provider -n kagent-system anthropic-claude -o yaml
   ```

3. **RBAC issues:** ServiceAccount lacks permissions
   ```bash
   kubectl auth can-i list pods --as=system:serviceaccount:kagent-system:kagent-sa
   ```

### Task Fails with Error Status

**Symptoms:** Task transitions to Completed but with error status

**Diagnosis:**
```bash
# View task error details
kubectl get task -n kagent-system <task-name> -o jsonpath='{.status.error}'
```

**Common Errors:**
- **"rate limit exceeded":** Anthropic API rate limit hit (60 req/min)
  - Wait 1 minute and retry
  - Check if multiple tasks are running concurrently
- **"context length exceeded":** Query result too large for 4096 token limit
  - Narrow query scope (e.g., filter by namespace)
  - Use `kubectl` directly for large list operations
- **"permission denied":** Query requires unauthorized verb (e.g., delete)
  - kagent is read-only; use Platform API for mutations

### ExternalSecret Not Syncing

**Symptoms:** `kagent-anthropic-secret` not created

**Diagnosis:**
```bash
# Check ExternalSecret status
kubectl get externalsecret -n kagent-system kagent-secrets -o yaml
# Look for status.conditions

# Check ClusterSecretStore
kubectl get clustersecretstore azure-backend -o yaml

# Verify Key Vault secret exists
az keyvault secret show --vault-name <vault-name> --name anthropic-api-key
```

**Resolution:**
1. Ensure External Secrets Operator is running:
   ```bash
   kubectl get pods -n external-secrets
   ```

2. Verify ESO ServiceAccount has Workload Identity annotation:
   ```bash
   kubectl get sa -n external-secrets external-secrets -o yaml
   # Should have azure.workload.identity/client-id annotation
   ```

3. Check ESO controller logs:
   ```bash
   kubectl logs -n external-secrets -l app.kubernetes.io/name=external-secrets --tail=50
   ```

### Agent CRD Not Found on First Sync

**Symptoms:** Argo CD Application shows `Agent.kagent.dev` CRD missing

**Expected Behavior:** This is normal on first deployment. The `kagent-crds` Helm chart registers the Agent CRD asynchronously.

**Resolution:**
- Wait 1-2 minutes for CRD registration
- Argo CD will auto-retry sync (5 retries with exponential backoff)
- `SkipDryRunOnMissingResource=true` annotation allows sync to proceed

**Verification:**
```bash
# Check CRD registration
kubectl get crds | grep kagent.dev
# Expected: agents.kagent.dev, tasks.kagent.dev, providers.kagent.dev

# Force Argo CD retry (if needed)
kubectl patch application -n argocd kagent -p '{"metadata":{"annotations":{"argocd.argoproj.io/refresh":"hard"}}}' --type merge
```

## Cost Considerations

**Anthropic API Pricing (Claude Sonnet 4.5):**
- Input: $3.00 / 1M tokens
- Output: $15.00 / 1M tokens

**Estimated Costs:**
- Average query: 4K input + 2K output tokens
- Cost per query: ~$0.03
- 100 queries/day: ~$3/month
- 1000 queries/day: ~$30/month

**Rate Limits (configured in values.yaml):**
- 60 requests/minute
- 100K tokens/minute

**Recommendations:**
- Platform API should rate-limit `/api/v1/agent/ask` (e.g., 10 queries/user/hour)
- Monitor usage via Anthropic dashboard
- Set alerts for unexpected cost spikes

## Integration Patterns

### Platform API (Task #53)

**Endpoint:** `POST /api/v1/agent/ask`

**Request:**
```json
{
  "query": "Why is my app unhealthy?",
  "namespace": "workloads",  // optional context
  "timeout": 60               // seconds, default 60
}
```

**Response (Server-Sent Events):**
```json
{"status": "running", "message": "Analyzing cluster state..."}
{"status": "completed", "response": "The app shows 0/2 replicas ready due to ImagePullBackOff..."}
```

**Implementation Pattern:**
```go
// Create Task CRD via dynamic client
task := &unstructured.Unstructured{
    Object: map[string]interface{}{
        "apiVersion": "kagent.dev/v1alpha1",
        "kind":       "Task",
        "metadata": map[string]interface{}{
            "name":      fmt.Sprintf("task-%s", uuid.New().String()[:8]),
            "namespace": "kagent-system",
        },
        "spec": map[string]interface{}{
            "agentRef": map[string]interface{}{"name": "platform-agent"},
            "prompt":   req.Query,
        },
    },
}

// Watch status until completion
watch, _ := dynamicClient.Resource(taskGVR).Namespace("kagent-system").Watch(ctx, ...)
for event := range watch.ResultChan() {
    // Stream status updates via SSE
    // Delete Task CRD after completion to prevent accumulation
}
```

### CLI (Task #76)

**Command:** `rdp ask <query>`

**Behavior:**
- Wraps `POST /api/v1/agent/ask` API call
- Streams response to stdout with spinner
- Color-coded output for resource citations
- Timeout: 60 seconds (configurable via `--timeout` flag)

### Portal UI (Already Implemented)

**Component:** `AIOperationsPanel.tsx`

**Features:**
- Chat interface with message history
- Streaming response rendering
- Markdown formatting for structured output
- HolmesGPT investigation trigger button
- Auto-scroll to latest message

## Comparison: kagent vs HolmesGPT

| Feature | kagent | HolmesGPT |
|---------|--------|-----------|
| **Purpose** | Natural language cluster queries | Root cause analysis for incidents |
| **Trigger** | User-initiated questions | Alertmanager webhooks (auto) + manual |
| **Scope** | General cluster exploration | Focused incident investigation |
| **Model** | Claude Sonnet 4.5 | Claude Sonnet 4.5 |
| **Data Sources** | Kubernetes API (read-only) | Prometheus, Logs, K8s API |
| **Output** | Conversational responses | Structured investigation reports |
| **RBAC** | Read-only ClusterRole | Read-only ClusterRole |
| **Use Case** | "Why is my app unhealthy?" | "Root cause of high latency incident" |

**Complementary Usage:**
1. Alertmanager detects incident → HolmesGPT investigates → generates report
2. Developer reviews HolmesGPT report → asks kagent follow-up questions
3. kagent provides conversational exploration of findings

## Next Steps

### Immediate (After This Deployment)

1. **Verify kagent installation:**
   ```bash
   # Run Phase 2 validation from implementation plan
   kubectl get application -n argocd kagent
   kubectl get pods -n kagent-system
   kubectl get agent -n kagent-system platform-agent
   ```

2. **Test manual Task creation:**
   ```bash
   # Run Phase 3 validation
   # Create test Task, verify response, cleanup
   ```

### Downstream Tasks (Blocked on This)

1. **Task #53:** Platform API `/api/v1/agent/ask` endpoint
   - Implement dynamic client for Task CRD creation
   - Add SSE streaming for real-time responses
   - Add rate limiting (10 queries/user/hour)
   - Implement Task CRD lifecycle management (auto-delete after response)

2. **Task #76:** CLI `rdp ask` command
   - Wrap Platform API calls with streaming output
   - Add color-coded formatting for resource citations
   - Support `--namespace` flag for scoped queries

3. **Portal UI integration:**
   - Update AIOperationsPanel to use `/api/v1/agent/ask` backend
   - Remove mock data from API client

### Optional Enhancements

- **Task TTL controller:** Auto-delete completed Tasks after 1 hour
- **Query templates:** Pre-defined queries for common troubleshooting scenarios
- **Multi-turn conversations:** Task CRDs with conversation history
- **Custom tools:** Platform API endpoints as kagent tools (Falco events, Prometheus queries)

## References

- **kagent Documentation:** https://github.com/kagent-dev/kagent
- **Helm Charts:** `oci://ghcr.io/kagent-dev/kagent/helm` (kagent-crds, kagent)
- **Anthropic API:** https://docs.anthropic.com/claude/reference
- **Platform API Integration:** See `api/internal/agent/` (Task #53)
- **CLI Integration:** See `cli/cmd/ask.go` (Task #76)
