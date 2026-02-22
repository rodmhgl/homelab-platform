# Monitoring Stack (kube-prometheus-stack)

Self-hosted observability platform for the AKS homelab IDP, providing Prometheus metrics collection, Alertmanager alerting, and Grafana visualization.

## Components

### Prometheus
- **Retention**: 15 days / 10GB
- **Storage**: 20Gi PVC
- **Resources**: 500m-1000m CPU, 2-4Gi RAM
- **Scrape targets**:
  - Kubernetes API server
  - Kubelets (cAdvisor metrics)
  - Node Exporter (host metrics)
  - Kube-State-Metrics (K8s object state)
  - Crossplane controller metrics
  - Gatekeeper audit metrics
  - Trivy Operator vulnerability counts
  - Platform API custom metrics

### Alertmanager
- **Storage**: 5Gi PVC
- **Resources**: 50m-100m CPU, 128-256Mi RAM
- **Routing**:
  - Critical/High severity alerts → HolmesGPT webhook (AI-powered root cause analysis)
  - All other alerts → null receiver (logging only)
- **Configuration**: `config` section in values.yaml

### Grafana
- **Storage**: 5Gi PVC
- **Resources**: 100m-200m CPU, 256-512Mi RAM
- **Admin password**: `admin` (CHANGE IN PRODUCTION)
- **Datasources**: Prometheus (pre-configured)
- **Dashboard folders**:
  - Platform — Cluster health, resource usage
  - Crossplane — Claim status, provisioning time, error rates
  - Compliance — Gatekeeper violations, Trivy CVEs, Falco alerts

### Exporters
- **Node Exporter**: Host-level metrics (CPU, memory, disk, network)
- **Kube-State-Metrics**: Kubernetes object state (Deployments, Pods, Claims, etc.)

## Deployment

**Prerequisites:**
- Argo CD installed (task #6)
- External Secrets Operator installed (task #31)
- NGINX Ingress Controller installed (task #90, #95)
- Grafana admin credentials in bootstrap Key Vault (see `externalsecrets/README.md`)

**Setup Grafana Credentials:**

Before deploying, create the required secrets in Azure Key Vault:

```bash
# Get Key Vault name from Terraform output
KEYVAULT_NAME=$(cd homelab-platform/infra && terraform output -raw keyvault_name)

# Create Grafana admin username
az keyvault secret set --vault-name "$KEYVAULT_NAME" --name "grafana-admin-username" --value "admin"

# Create Grafana admin password (generate strong password)
GRAFANA_PASSWORD=$(openssl rand -base64 32)
az keyvault secret set --vault-name "$KEYVAULT_NAME" --name "grafana-admin-password" --value "$GRAFANA_PASSWORD"
echo "Grafana password: $GRAFANA_PASSWORD" > ~/.grafana-admin-password.txt
chmod 600 ~/.grafana-admin-password.txt
```

**Install:**

Argo CD auto-discovers the monitoring Application via the root App of Apps pattern. No manual apply needed.

**Namespace:** `monitoring` (auto-created)

**Verify:**
```bash
# Check Application sync status
kubectl get application monitoring -n argocd

# Check ExternalSecret sync (must succeed before Grafana starts)
kubectl get externalsecret -n monitoring grafana-admin-creds

# Check Prometheus Operator
kubectl get pods -n monitoring -l app.kubernetes.io/name=prometheus-operator

# Check Prometheus instance
kubectl get prometheus -n monitoring

# Check Alertmanager instance
kubectl get alertmanager -n monitoring

# Check Grafana (depends on grafana-admin-creds Secret)
kubectl get pods -n monitoring -l app.kubernetes.io/name=grafana
```

## Access

### Prometheus UI
```bash
kubectl port-forward -n monitoring svc/monitoring-prometheus 9090:9090
# Open http://localhost:9090
```

### Alertmanager UI
```bash
kubectl port-forward -n monitoring svc/monitoring-alertmanager 9093:9093
# Open http://localhost:9093
```

### Grafana UI

**Via Ingress (Production):**
```bash
# Access Grafana at public URL
open http://grafana.rdp.azurelaboratory.com

# Get credentials from Secret (synced from Key Vault)
kubectl get secret -n monitoring grafana-admin-creds -o jsonpath='{.data.admin-user}' | base64 -d
echo ""
kubectl get secret -n monitoring grafana-admin-creds -o jsonpath='{.data.admin-password}' | base64 -d
echo ""

# Or retrieve from the saved file (if you followed setup instructions)
cat ~/.grafana-admin-password.txt
```

**Via Port-Forward (Development):**
```bash
kubectl port-forward -n monitoring svc/monitoring-grafana 3000:80
# Open http://localhost:3000
```

## ServiceMonitor Pattern

The Prometheus Operator uses **ServiceMonitor** CRDs for declarative scrape target discovery. To expose metrics from any platform component:

**1. Expose metrics endpoint in your Service:**
```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-app
  namespace: workloads
  labels:
    app: my-app
spec:
  ports:
    - name: metrics  # Name MUST be "metrics"
      port: 8080
      targetPort: 8080
  selector:
    app: my-app
```

**2. Create a ServiceMonitor:**
```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: my-app
  namespace: workloads
spec:
  selector:
    matchLabels:
      app: my-app
  endpoints:
    - port: metrics
      interval: 30s
      path: /metrics
```

**3. Verify scrape target:**
```bash
# Check ServiceMonitor exists
kubectl get servicemonitor -n workloads

# Check Prometheus targets (port-forward to 9090 first)
# Visit http://localhost:9090/targets
```

## Platform Component Metrics

### Crossplane
- **Endpoint**: `http://crossplane.crossplane-system:8080/metrics`
- **Key metrics**:
  - `crossplane_managed_resource_exists` — Number of managed resources
  - `crossplane_managed_resource_ready` — Ready status (0 or 1)
  - `crossplane_managed_resource_synced` — Sync status (0 or 1)
  - `crossplane_claim_exists` — Number of Claims

### Gatekeeper
- **Endpoint**: `http://gatekeeper-webhook-service.gatekeeper-system:8443/metrics`
- **Key metrics**:
  - `gatekeeper_violations` — Total violations by enforcement action
  - `gatekeeper_constraint_template_ingestion_count` — Template load success/failure
  - `gatekeeper_audit_duration_seconds` — Audit loop timing

### Trivy Operator
- **Endpoint**: `http://trivy-operator.trivy-system:80/metrics`
- **Key metrics**:
  - `trivy_vulnerability_id` — CVE count by severity
  - `trivy_image_vulnerabilities` — Vulnerabilities per image
  - `trivy_resource_vulnerabilities` — Vulnerabilities per workload

### Platform API
- **Endpoint**: `http://platform-api.platform:8080/metrics`
- **Key metrics** (to be implemented):
  - `platform_api_http_requests_total` — Request count by endpoint/status
  - `platform_api_http_request_duration_seconds` — Request latency histogram
  - `platform_api_scaffold_operations_total` — Scaffold creation count by template
  - `platform_api_infra_claims_total` — Infra Claim operations by kind

## Custom Alerts

Default alerting rules are enabled for:
- Node resource exhaustion (CPU, memory, disk)
- Pod CrashLoopBackOff
- Persistent Volume usage
- Kubernetes API server availability

**Add custom alerts** by creating PrometheusRule resources:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: crossplane-alerts
  namespace: monitoring
spec:
  groups:
    - name: crossplane
      interval: 30s
      rules:
        - alert: CrossplaneClaimNotReady
          expr: crossplane_managed_resource_ready{kind="Claim"} == 0
          for: 5m
          labels:
            severity: critical
          annotations:
            summary: "Crossplane Claim not ready"
            description: "Claim {{ $labels.name }} in namespace {{ $labels.namespace }} has been unready for 5 minutes"
```

## HolmesGPT Integration

Alertmanager is configured to forward **critical** and **high** severity alerts to HolmesGPT for AI-powered root cause analysis:

**Webhook config** (in values.yaml):
```yaml
alertmanager:
  config:
    receivers:
      - name: 'holmesgpt'
        webhook_configs:
          - url: 'http://holmesgpt.holmesgpt.svc.cluster.local:8080/webhook'
            send_resolved: true
```

**Prerequisites:**
- HolmesGPT installed (task #39)
- HolmesGPT Service listening on port 8080

## Grafana Dashboards

Dashboard ConfigMaps are referenced in values.yaml but need to be created separately (task #37):

**Expected ConfigMaps:**
- `grafana-dashboards-platform` — Cluster overview, resource usage
- `grafana-dashboards-crossplane` — Claim status, provisioning times, error rates
- `grafana-dashboards-compliance` — Gatekeeper violations, Trivy CVEs, Falco alerts

**Create dashboard ConfigMaps:**
```bash
# Export a dashboard from Grafana UI as JSON
# Create ConfigMap:
kubectl create configmap grafana-dashboards-platform \
  -n monitoring \
  --from-file=cluster-overview.json=dashboard.json
```

## Troubleshooting

### Prometheus not scraping targets

**Check ServiceMonitor exists:**
```bash
kubectl get servicemonitor -A
```

**Check Prometheus Operator logs:**
```bash
kubectl logs -n monitoring -l app.kubernetes.io/name=prometheus-operator
```

**Verify target in Prometheus UI:**
Port-forward to Prometheus and check http://localhost:9090/targets

### Alertmanager not sending alerts

**Check Alertmanager config:**
```bash
kubectl get secret -n monitoring alertmanager-monitoring-kube-prometheus-alertmanager -o jsonpath='{.data.alertmanager\.yaml}' | base64 -d
```

**Check Alertmanager logs:**
```bash
kubectl logs -n monitoring -l app.kubernetes.io/name=alertmanager
```

**Test webhook manually:**
```bash
kubectl run -it --rm curl --image=curlimages/curl --restart=Never -- \
  curl -X POST http://holmesgpt.holmesgpt.svc.cluster.local:8080/webhook \
  -H "Content-Type: application/json" \
  -d '{"status":"firing","alerts":[{"labels":{"alertname":"test"}}]}'
```

### Grafana dashboards not loading

**Check ConfigMap exists:**
```bash
kubectl get configmap -n monitoring grafana-dashboards-platform
```

**Check Grafana logs:**
```bash
kubectl logs -n monitoring -l app.kubernetes.io/name=grafana
```

**Verify dashboard provider config:**
```bash
kubectl exec -it -n monitoring deployment/monitoring-grafana -- cat /etc/grafana/provisioning/dashboards/dashboardproviders.yaml
```

## Storage Management

All three components use Persistent Volumes:

**Check PVCs:**
```bash
kubectl get pvc -n monitoring
```

**Expected PVCs:**
- `prometheus-monitoring-kube-prometheus-prometheus-db-prometheus-monitoring-kube-prometheus-prometheus-0` — 20Gi
- `alertmanager-monitoring-kube-prometheus-alertmanager-db-alertmanager-monitoring-kube-prometheus-alertmanager-0` — 5Gi
- `monitoring-grafana` — 5Gi

**Storage class**: Default (Azure Disk for AKS)

## Security

**RBAC:**
- Prometheus Operator has cluster-wide read permissions (ServiceMonitors, PodMonitors)
- Prometheus has read-only access to Kubernetes API for service discovery
- Grafana has no cluster permissions (metrics queried via Prometheus datasource)

**Network Policies:**
- Default deny ingress/egress (if NetworkPolicy enforcement enabled)
- Allow Prometheus → API server (scraping)
- Allow Grafana → Prometheus (queries)
- Allow Alertmanager → HolmesGPT (webhooks)

**Secrets:**
- Grafana admin credentials: synced from Azure Key Vault via ESO → Secret `grafana-admin-creds`
- Zero hardcoded credentials — all secrets managed via ExternalSecrets
- See `externalsecrets/README.md` for credential rotation and troubleshooting

## Retention and Cost

**Prometheus retention**: 15 days (configurable via `prometheus.prometheusSpec.retention`)

**Estimated storage usage:**
- ~1GB/day for 3-node AKS cluster with ~20 workloads
- 15-day retention = ~15GB (20Gi PVC gives headroom)

**Reduce retention** if storage costs are a concern:
```yaml
prometheus:
  prometheusSpec:
    retention: 7d  # Reduce to 7 days
    retentionSize: "5GB"
```

## Next Steps

1. **Install HolmesGPT** (task #39) to enable AI-powered alert analysis
2. **Create Grafana dashboards** (task #37) for platform, Crossplane, and compliance views
3. **Add custom PrometheusRules** for platform-specific alerts (Claim provisioning failures, policy violations, etc.)
4. **Implement Platform API metrics** (task #48) to expose scaffold/infra/compliance operation counts

## References

- [kube-prometheus-stack Helm Chart](https://github.com/prometheus-community/helm-charts/tree/main/charts/kube-prometheus-stack)
- [Prometheus Operator Documentation](https://prometheus-operator.dev/)
- [Grafana Documentation](https://grafana.com/docs/grafana/latest/)
- [Alertmanager Configuration](https://prometheus.io/docs/alerting/latest/configuration/)
