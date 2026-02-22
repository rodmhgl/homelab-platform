# Grafana Dashboards

This directory contains Grafana dashboards for the homelab platform, delivered via ConfigMaps using the Grafana sidecar pattern.

## How It Works

The kube-prometheus-stack Helm chart includes a Grafana sidecar container that automatically discovers and loads dashboards from ConfigMaps labeled with `grafana_dashboard: "1"`.

When Argo CD syncs the monitoring Application, it deploys these ConfigMaps to the `monitoring` namespace, and the Grafana sidecar automatically loads them.

## Available Dashboards

### 1. Platform Compliance Overview (`compliance-overview.json`)

**UID:** `platform-compliance`
**ConfigMap:** `grafana-dashboard-compliance`

Provides a comprehensive view of platform compliance posture:

- **Compliance Score Gauge** — Real-time compliance score (0-100) with color-coded thresholds
- **Compliance Score Over Time** — Historical trend line
- **Policy Violations by Type** — Pie chart breaking down Gatekeeper constraint violations
- **CVE Count by Severity** — Stacked bar chart showing Critical/High/Medium/Low vulnerabilities over time
- **Falco Security Events** — Recent runtime security event count (5-minute window)
- **Policy Violations by Namespace** — Bar chart showing which namespaces have the most violations
- **Top 10 Vulnerable Images** — Table view of images with the most Critical/High CVEs

**Metrics Used:**
- `gatekeeper_violations{enforcementAction="deny"}` — Gatekeeper policy violations
- `vulnerabilityreport_vulnerability_count` — Trivy Operator CVE counts by severity
- `falcosidekick_inputs_total` — Falco security event counter

**Refresh Rate:** 30 seconds
**Default Time Range:** Last 6 hours

### 2. Crossplane Claim Status (`crossplane-status.json`)

**UID:** `crossplane-status`
**ConfigMap:** `grafana-dashboard-crossplane`

Monitors Crossplane-managed infrastructure health and reconciliation:

- **Total Claims** — Count of all existing Claims
- **Ready Claims** — Count of Claims in Ready state
- **Synced Claims** — Count of Claims successfully synced to Azure
- **Not Ready Claims** — Count of Claims with issues (red threshold)
- **Reconcile Errors** — 5-minute rate of reconciliation failures
- **Ready Percentage** — Overall infrastructure health percentage
- **Claim Status Over Time** — Trend lines for Ready/Not Ready/Synced states
- **Reconcile Rate** — Success vs. error rate for reconciliation loops
- **All Claims Table** — Detailed list of all Claims with Name, Namespace, Kind
- **Claims by Type** — Pie chart showing distribution of StorageBucket vs. Vault Claims
- **Claims by Namespace** — Pie chart showing which namespaces consume infrastructure

**Metrics Used:**
- `crossplane_managed_resource_exists{state="true"}` — Claim existence
- `crossplane_managed_resource_ready{state="true|false"}` — Ready state
- `crossplane_managed_resource_synced{state="true"}` — Sync state
- `crossplane_managed_resource_reconcile_total{result="success|error"}` — Reconciliation counters

**Refresh Rate:** 30 seconds
**Default Time Range:** Last 6 hours

## Adding New Dashboards

1. **Export dashboard JSON** from Grafana UI (Settings → JSON Model)
2. **Remove `id` field** from JSON (set to `null`) — allows dashboard to be portable
3. **Set a unique `uid`** field (e.g., `my-dashboard`)
4. **Create a new dashboard JSON file** in this directory (e.g., `my-dashboard.json`)
5. **Create a ConfigMap** wrapping the JSON:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: grafana-dashboard-<name>
  namespace: monitoring
  labels:
    grafana_dashboard: "1"
data:
  my-dashboard.json: |
    <indent JSON with 4 spaces>
```

6. **Name the ConfigMap file** `configmap-<name>.yaml`
7. **Commit to Git** — Argo CD will sync it automatically (thanks to the `dashboards/configmap-*.yaml` glob pattern in `application.yaml`)

## Accessing Dashboards

**Via Ingress (Production):**
```
http://grafana.rdp.azurelaboratory.com/
```

**Via Port-Forward (Development):**
```bash
kubectl port-forward -n monitoring svc/monitoring-grafana 3000:80
# Open http://localhost:3000
```

**Login Credentials:**
- Username: `admin`
- Password: Retrieved from ExternalSecret `grafana-admin-creds` (stored in Azure Key Vault)

## Troubleshooting

### Dashboards Not Appearing

1. **Check ConfigMap exists:**
   ```bash
   kubectl get cm -n monitoring -l grafana_dashboard=1
   ```

2. **Check ConfigMap has correct label:**
   ```bash
   kubectl get cm grafana-dashboard-compliance -n monitoring -o yaml | grep grafana_dashboard
   ```

3. **Check Grafana sidecar logs:**
   ```bash
   kubectl logs -n monitoring deployment/monitoring-grafana -c grafana-sc-dashboard
   ```

4. **Restart Grafana pod to force reload:**
   ```bash
   kubectl rollout restart deployment/monitoring-grafana -n monitoring
   ```

### Metrics Not Showing

1. **Check Prometheus scrape targets:**
   ```
   http://grafana.rdp.azurelaboratory.com/prometheus/targets
   ```

2. **Verify metric exists in Prometheus:**
   ```bash
   kubectl port-forward -n monitoring svc/monitoring-prometheus 9090:9090
   # Open http://localhost:9090 and run a PromQL query
   ```

3. **Common missing metrics:**
   - `gatekeeper_violations` — Ensure Gatekeeper has violations (create a test violation)
   - `vulnerabilityreport_vulnerability_count` — Ensure Trivy Operator has scanned images (check VulnerabilityReport CRDs)
   - `crossplane_managed_resource_*` — Ensure Crossplane metrics port is exposed and scraped (check `values.yaml` scrape config)

## Dashboard Design Principles

- **Auto-refresh** — All dashboards refresh every 30 seconds to show near-real-time data
- **Color coding** — Red = bad/critical, Yellow/Orange = warning, Green = good
- **Threshold-based visuals** — Gauges and stats change color based on severity
- **Time series + instant values** — Trends over time plus current state
- **Drill-down friendly** — Tables with sortable columns for detailed investigation
- **Self-documenting** — Panel titles clearly describe what's shown

## Next Steps

Planned dashboards (not yet implemented):

- **Platform API Metrics** — Request rates, latencies, error rates by endpoint
- **Argo CD Sync Status** — Application health, sync duration, sync failures
- **Cluster Resource Usage** — Node CPU/memory, pod counts, PV usage
- **External Secrets Status** — ExternalSecret sync state, Key Vault access errors
