# Compliance Package

This package provides HTTP handlers for Platform API compliance endpoints that aggregate Gatekeeper policy violations and Trivy Operator vulnerability scans.

## Architecture

The compliance package follows the **Argo CD handler pattern** with three layers:

```
compliance/
├── handler.go  # HTTP handlers (5 endpoints)
├── client.go   # Kubernetes dynamic client wrapper
├── types.go    # Request/response DTOs
└── README.md   # This file
```

### Data Flow

```
HTTP Request → Handler → Kubernetes Dynamic Client → CRDs → Transform → JSON Response
```

All endpoints are **read-only** operations that query existing Kubernetes CRDs. There is no GitOps involvement.

## Endpoints

### GET /api/v1/compliance/summary

Returns overall compliance score and aggregated metrics.

**Response:**
```json
{
  "complianceScore": 85.5,
  "totalViolations": 3,
  "totalVulnerabilities": 12,
  "violationsBySeverity": {
    "policy": 1,
    "config": 1,
    "security": 1
  },
  "vulnerabilitiesBySeverity": {
    "CRITICAL": 0,
    "HIGH": 2,
    "MEDIUM": 5,
    "LOW": 5
  }
}
```

**Compliance Score Formula:**
```
score = max(0, 100 - (violations × 5) - (critical_cves × 10) - (high_cves × 5))
```

**Data Sources:**
- All 8 deployed Constraint kinds (queries `.status.violations[]`)
- VulnerabilityReports in workload namespaces (excludes platform namespaces)

---

### GET /api/v1/compliance/policies

Returns list of active Gatekeeper policies (ConstraintTemplates).

**Response:**
```json
{
  "policies": [
    {
      "name": "k8srequiredlabels",
      "kind": "K8sRequiredLabels",
      "description": "Validates k8srequiredlabels policy",
      "scope": ["cluster"],
      "parameters": {
        "labels": {
          "type": "array",
          "description": "List of required label keys"
        }
      }
    }
  ]
}
```

**Data Sources:**
- ConstraintTemplate CRDs (`templates.gatekeeper.sh/v1beta1`)

---

### GET /api/v1/compliance/violations

Returns Gatekeeper audit violations with optional filtering.

**Query Parameters:**
- `namespace` (optional) — Filter by namespace
- `kind` (optional) — Filter by resource kind
- `constraint` (optional) — Filter by constraint name

**Response:**
```json
{
  "violations": [
    {
      "constraintName": "require-labels",
      "constraintKind": "K8sRequiredLabels",
      "resource": "workloads/Deployment/my-app",
      "namespace": "workloads",
      "message": "Missing required label: app.kubernetes.io/name"
    }
  ]
}
```

**Data Sources:**
- All 8 Constraint kinds (queries `.status.violations[]`)

**Constraint Kinds Queried:**
1. `K8sRequiredLabels` (policy)
2. `ContainerLimitsRequired` (config)
3. `NoLatestTag` (policy)
4. `AllowedRepos` (security)
5. `NoPrivilegedContainers` (security)
6. `RequireProbes` (config)
7. `CrossplaneClaimLocation` (policy)
8. `CrossplaneNoPublicAccess` (security)

---

### GET /api/v1/compliance/vulnerabilities

Returns Trivy Operator CVE scan results with optional filtering.

**Query Parameters:**
- `namespace` (optional) — Filter by namespace
- `severity` (optional) — Filter by severity (CRITICAL, HIGH, MEDIUM, LOW)
- `image` (optional) — Filter by image name substring

**Response:**
```json
{
  "vulnerabilities": [
    {
      "image": "myregistry.azurecr.io/my-app:v1.2.3",
      "namespace": "workloads",
      "workload": "replicaset-my-app-7d9f8c",
      "cveId": "CVE-2024-1234",
      "severity": "HIGH",
      "score": 7.5,
      "affectedPackage": "openssl",
      "fixedVersion": "1.1.1w",
      "primaryLink": "https://nvd.nist.gov/vuln/detail/CVE-2024-1234"
    }
  ]
}
```

**Data Sources:**
- VulnerabilityReport CRDs (`aquasecurity.github.io/v1alpha1`)

**Namespaces:**
- **Included:** All workload namespaces
- **Excluded:** `kube-system`, `argocd`, `crossplane-system`, `gatekeeper-system`, `trivy-system`, `external-secrets`, `monitoring`, `platform`

---

### GET /api/v1/compliance/events

Returns security events (Falco). **Placeholder implementation** until Falco is deployed (task #33).

**Response:**
```json
{
  "events": []
}
```

**Future Data Sources:**
- Falco events via webhook (POST /api/v1/webhooks/falco)

## Client Implementation

### Kubernetes Dynamic Client

Uses `k8s.io/client-go/dynamic` to query CRDs without generated Go types.

**Authentication:**
- **In-cluster:** Uses Pod ServiceAccount (default)
- **Out-of-cluster:** Uses kubeconfig from `KUBECONFIG` env var

**RBAC:**
Platform API ServiceAccount already has read permissions for:
- `constraints.gatekeeper.sh/v1beta1/*`
- `templates.gatekeeper.sh/v1beta1/constrainttemplates`
- `aquasecurity.github.io/v1alpha1/vulnerabilityreports`

See `platform/platform-api/rbac.yaml` for details.

### GVR Mapping

The client maps constraint kinds to Group/Version/Resource identifiers:

```go
// Example: K8sRequiredLabels
gvr := schema.GroupVersionResource{
    Group:    "constraints.gatekeeper.sh",
    Version:  "v1beta1",
    Resource: "k8srequiredlabels",  // lowercase plural
}
```

**Naming Convention:**
Gatekeeper constraint resources use lowercase, no hyphens (e.g., `k8srequiredlabels`, not `k8s-required-labels`).

## Error Handling

### Graceful Degradation

If CRDs are missing or inaccessible, endpoints return partial data instead of failing completely:

- Missing Gatekeeper CRDs → `violations: []`
- Missing Trivy CRDs → `vulnerabilities: []`
- Cluster unreachable → 503 Service Unavailable

### Logging

All operations use structured logging:

```go
slog.Error("Failed to list constraints",
    "constraintKind", kind,
    "error", err)
```

## Local Testing

### Prerequisites

1. AKS cluster with Gatekeeper + Trivy Operator deployed
2. Platform API ServiceAccount with proper RBAC
3. `kubectl` configured with cluster credentials

### Port-Forward to Cluster

```bash
# Forward Platform API service
kubectl port-forward -n platform svc/platform-api 8080:8080
```

### Test Endpoints

```bash
# Summary
curl http://localhost:8080/api/v1/compliance/summary \
  -H "Authorization: Bearer test-token"

# Policies
curl http://localhost:8080/api/v1/compliance/policies \
  -H "Authorization: Bearer test-token"

# Violations (all)
curl http://localhost:8080/api/v1/compliance/violations \
  -H "Authorization: Bearer test-token"

# Violations (filtered by namespace)
curl "http://localhost:8080/api/v1/compliance/violations?namespace=workloads" \
  -H "Authorization: Bearer test-token"

# Vulnerabilities (all)
curl http://localhost:8080/api/v1/compliance/vulnerabilities \
  -H "Authorization: Bearer test-token"

# Vulnerabilities (filtered by severity)
curl "http://localhost:8080/api/v1/compliance/vulnerabilities?severity=CRITICAL" \
  -H "Authorization: Bearer test-token"

# Events (placeholder)
curl http://localhost:8080/api/v1/compliance/events \
  -H "Authorization: Bearer test-token"
```

### Local Development (Out-of-Cluster)

Set environment variables to run locally:

```bash
export KUBECONFIG=~/.kube/config
export IN_CLUSTER=false
export ARGOCD_SERVER_URL=https://argocd.example.com
export ARGOCD_TOKEN=your-token
export GITHUB_TOKEN=your-token
export GITHUB_ORG=your-org

go run main.go
```

## CRD Query Patterns

### Constraint Status Structure

```yaml
apiVersion: constraints.gatekeeper.sh/v1beta1
kind: K8sRequiredLabels
metadata:
  name: require-labels
status:
  violations:
    - enforcementAction: deny
      kind: Deployment
      message: "Missing required label: app.kubernetes.io/name"
      name: my-app
      namespace: workloads
```

**Extraction:**
```go
violations, found, err := unstructured.NestedSlice(
    constraint.Object,
    "status", "violations",
)
```

### VulnerabilityReport Structure

```yaml
apiVersion: aquasecurity.github.io/v1alpha1
kind: VulnerabilityReport
metadata:
  name: replicaset-my-app-7d9f8c
  namespace: workloads
report:
  artifact:
    repository: myregistry.azurecr.io/my-app
    tag: v1.2.3
  vulnerabilities:
    - vulnerabilityID: CVE-2024-1234
      severity: HIGH
      resource: openssl
      fixedVersion: 1.1.1w
      score:
        nvd:
          V3Score: 7.5
```

**Extraction:**
```go
vulns, found, err := unstructured.NestedSlice(
    report.Object,
    "report", "vulnerabilities",
)
```

## Performance Considerations

### Query Optimization

- **ListAllConstraints:** Queries 8 constraint kinds in parallel (future optimization opportunity)
- **ListVulnerabilityReportsInWorkloads:** Filters at application layer (not Kubernetes API) to exclude platform namespaces
- **Caching:** None currently — all queries hit Kubernetes API on every request

### Response Times

Typical query times (depends on cluster size):
- `/summary`: 1-2s (queries constraints + vulnerability reports)
- `/policies`: < 500ms (queries constraint templates)
- `/violations`: 500ms-1s (queries 8 constraint kinds)
- `/vulnerabilities`: 500ms-1s (queries vulnerability reports)

### Future Optimizations

- Add in-memory caching with TTL (30s)
- Batch constraint queries in parallel
- Use Kubernetes watch API for real-time updates
- Add pagination for large result sets

## Dependencies

### Go Modules

```
k8s.io/client-go@v0.31.0
k8s.io/apimachinery@v0.31.0
```

### Kubernetes CRDs

**Required:**
- Gatekeeper (deployed via `platform/gatekeeper/`)
- Trivy Operator (deployed via `platform/trivy-operator/`)

**Optional:**
- Falco (not yet deployed — task #33)

## Integration Points

### Portal UI

Compliance endpoints power the Portal UI dashboard:
- `/summary` → Compliance Score donut chart (task #81)
- `/violations` → Policy Violations table (task #82)
- `/vulnerabilities` → Vulnerability Feed (task #83)
- `/events` → Security Events timeline (task #84)

### CLI

`rdp` CLI commands use these endpoints:
- `rdp compliance summary` → GET /summary
- `rdp compliance policies` → GET /policies
- `rdp compliance violations` → GET /violations (task #73)
- `rdp compliance vulnerabilities` → GET /vulnerabilities (task #73)

## Future Enhancements

Out of scope for this implementation (future tasks):

- WebSocket streaming for real-time compliance updates
- Historical trend data (violation counts over time)
- Falco event integration (task #33)
- CSV/PDF export
- Alerting thresholds (webhook triggers when score drops)
- Prometheus metrics export
