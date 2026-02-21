# NGINX Ingress Controller

## Overview

The NGINX Ingress Controller provides external HTTP/HTTPS access to platform services via a single Azure LoadBalancer IP address. This eliminates the need for manual `kubectl port-forward` commands and provides a production-ready ingress pattern for the homelab environment.

**Deployment Wave:** 3 (after Crossplane ProviderConfig/XRDs, before Gatekeeper)

**Key Features:**
- Path-based routing on a single external IP
- Azure LoadBalancer integration (automatic public IP provisioning)
- Prometheus metrics integration with kube-prometheus-stack
- Security hardening (non-root, seccomp, minimal capabilities)
- TLS-ready configuration (certificates deferred to task #93)

---

## Architecture

### Service Exposure Strategy

**Type:** Azure LoadBalancer with path-based routing

```
External Traffic
    ↓
Azure LoadBalancer (public IP)
    ↓
NGINX Ingress Controller (ingress-nginx namespace)
    ↓
┌───────────────────┬───────────────────┬───────────────────┐
│   /api/*          │   /grafana/*      │   /*              │
│   Platform API    │   Grafana         │   Portal UI       │
│   (platform ns)   │   (monitoring ns) │   (platform ns)   │
└───────────────────┴───────────────────┴───────────────────┘
```

### Routing Rules

| Path Pattern | Backend Service | Namespace | Port | Purpose |
|---|---|---|---|---|
| `/api/*` | `platform-api` | `platform` | 80 | Platform API endpoints |
| `/grafana/*` | `monitoring-grafana` | `monitoring` | 80 | Monitoring dashboards |
| `/*` | `portal-ui` | `platform` | 80 | Portal UI (React SPA) |

**Order matters:** `/api` and `/grafana` must be defined before `/*` to match specific paths first.

---

## File Structure

```
platform/nginx-ingress/
├── application.yaml          # Argo CD Application (wave 3)
├── values.yaml              # Helm chart overrides
├── ingresses/
│   └── platform.yaml        # Ingress resource for platform services
└── README.md                # This file
```

---

## Access Instructions

### 1. Get External LoadBalancer IP

```bash
kubectl get svc -n ingress-nginx ingress-nginx-controller
```

Expected output:
```
NAME                       TYPE           CLUSTER-IP     EXTERNAL-IP      PORT(S)
ingress-nginx-controller   LoadBalancer   172.16.x.x     <Azure-IP>       80:xxxxx/TCP,443:xxxxx/TCP
```

If `EXTERNAL-IP` shows `<pending>`, wait 1-2 minutes for Azure to provision the public IP.

### 2. Access Platform Services

**Portal UI (React SPA):**
```bash
EXTERNAL_IP=$(kubectl get svc -n ingress-nginx ingress-nginx-controller -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
echo "Portal UI: http://$EXTERNAL_IP/"
```

**Platform API:**
```bash
curl http://$EXTERNAL_IP/api/v1/health
# Expected: {"status":"ok"}
```

**Grafana (monitoring dashboards):**
```bash
echo "Grafana: http://$EXTERNAL_IP/grafana/"
# Default credentials: admin / prom-operator
```

### 3. (Optional) Add DNS Entry

For easier access, add a DNS record in your homelab DNS server (or `/etc/hosts`):

```bash
# /etc/hosts example
<EXTERNAL-IP>  homelab.local
```

Then access services via:
- Portal UI: `http://homelab.local/`
- Platform API: `http://homelab.local/api/v1/health`
- Grafana: `http://homelab.local/grafana/`

---

## Configuration Summary

### Helm Chart Details

- **Chart:** `ingress-nginx/ingress-nginx`
- **Version:** 4.14.3
- **Controller App Version:** v1.14.3
- **Namespace:** `ingress-nginx` (auto-created)

### Resource Limits (Homelab-Optimized)

**Controller:**
- Requests: 100m CPU, 256Mi memory
- Limits: 500m CPU, 512Mi memory
- Replicas: 1 (scale to 2+ for HA)

**Default Backend:**
- Requests: 10m CPU, 20Mi memory
- Limits: 20m CPU, 40Mi memory

### Security Configuration

**Pod Security:**
- Non-root user (UID 101)
- Seccomp profile: `RuntimeDefault`
- Capabilities: Drop ALL, add only `NET_BIND_SERVICE`

**Container Security:**
- Read-only rootfs: `false` (NGINX requires writable `/tmp` for buffering)
- `allowPrivilegeEscalation: false`

### NGINX Configuration Highlights

| Setting | Value | Purpose |
|---|---|---|
| `proxy-body-size` | 50m | Platform API may accept scaffold template uploads |
| `proxy-read-timeout` | 60s | Standard timeout for API requests |
| `limit-rps` | 20 | Rate limiting per IP (homelab-sized) |
| `limit-connections` | 100 | Connection limit per IP |
| `ssl-protocols` | TLSv1.2, TLSv1.3 | TLS-ready for cert-manager (task #93) |
| `use-forwarded-headers` | true | Preserve client IP through proxy chain |

### Monitoring Integration

**Prometheus metrics enabled:**
- ServiceMonitor with label `prometheus: monitoring` (matches kube-prometheus-stack selector)
- Metrics endpoint: `:10254/metrics`
- Scraped automatically by Prometheus (wave 8)

**Available metrics:**
- `nginx_ingress_controller_requests` — Request count by status code
- `nginx_ingress_controller_request_duration_seconds` — Request latency
- `nginx_ingress_controller_nginx_process_connections` — Active connections
- `nginx_ingress_controller_nginx_process_cpu_seconds_total` — CPU usage

---

## Routing Logic Explained

### Path Matching Order

NGINX Ingress evaluates rules **in the order they appear** in the manifest. For path-based routing to work correctly:

1. **Specific paths first:** `/api`, `/grafana`
2. **Catch-all path last:** `/` (matches everything)

If `/` were defined first, all requests would route to Portal UI (no API or Grafana access).

### Why No `rewrite-target` Annotation?

The Ingress manifest **does not** use `nginx.ingress.kubernetes.io/rewrite-target` because all backend services expect the full path:

- **Platform API** serves under `/api/v1/*` → Ingress forwards `/api/v1/health` as-is
- **Grafana** configured with `serve_from_sub_path: true` → expects `/grafana/*` paths
- **Portal UI** uses Nginx internally with `try_files $uri /index.html` → handles React Router routes

### Portal UI Client-Side Routing

React Router in the Portal UI handles client-side routes (e.g., `/apps`, `/infra`, `/compliance`). When a user navigates to `http://<EXTERNAL-IP>/apps`:

1. Request reaches NGINX Ingress Controller
2. Path `/apps` matches catch-all rule `/*` → routes to `portal-ui` service
3. Portal UI's internal Nginx config serves `index.html` for all non-file requests
4. React Router renders the `ApplicationsPage` component

**No special Ingress configuration needed** for SPA routing; it's handled by Portal UI's container.

---

## Verification Steps

### 1. Check Argo CD Application Status

```bash
kubectl get application -n argocd nginx-ingress
# Expected: STATUS=Synced, HEALTH=Healthy

kubectl describe application -n argocd nginx-ingress
# Check Events for successful sync
```

### 2. Verify NGINX Ingress Controller Pods

```bash
kubectl get pods -n ingress-nginx
# Expected:
# - ingress-nginx-controller-xxx        Running (1/1)
# - ingress-nginx-defaultbackend-xxx    Running (1/1)

kubectl logs -n ingress-nginx -l app.kubernetes.io/name=ingress-nginx --tail=50
# Check for ERROR logs (should be none)
```

### 3. Check Ingress Resource Status

```bash
kubectl get ingress -n platform platform-ingress
# Expected: ADDRESS column shows LoadBalancer IP

kubectl describe ingress -n platform platform-ingress
# Check Events for successful backend configuration
```

### 4. Test HTTP Routes

```bash
EXTERNAL_IP=$(kubectl get svc -n ingress-nginx ingress-nginx-controller -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

# Test Platform API
curl http://$EXTERNAL_IP/api/v1/health
# Expected: {"status":"ok"}

# Test Portal UI (HTML response)
curl -I http://$EXTERNAL_IP/
# Expected: HTTP/1.1 200 OK, Content-Type: text/html

# Test Grafana (redirect to login)
curl -I http://$EXTERNAL_IP/grafana/
# Expected: HTTP/1.1 302 Found
```

### 5. Verify Prometheus Metrics

```bash
# Check ServiceMonitor exists
kubectl get servicemonitor -n ingress-nginx
# Expected: ingress-nginx-controller ServiceMonitor

# Port-forward to Prometheus UI
kubectl port-forward -n monitoring svc/monitoring-kube-prometheus-prometheus 9090:9090

# Open http://localhost:9090/targets
# Expected: ingress-nginx/ingress-nginx-controller target is UP
```

### 6. Test Rate Limiting (Optional)

```bash
# Trigger rate limit (20 requests per second)
EXTERNAL_IP=$(kubectl get svc -n ingress-nginx ingress-nginx-controller -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
for i in {1..30}; do curl -s -o /dev/null -w "%{http_code}\n" http://$EXTERNAL_IP/api/v1/health; done

# Expected: First ~20 requests return 200, remaining return 503 (rate limit exceeded)
```

---

## Troubleshooting

### Issue: LoadBalancer IP stays in `<pending>` state

**Cause:** Azure provisioning delay or insufficient permissions

**Fix:**
1. Wait 2-3 minutes (Azure LoadBalancer provisioning can be slow)
2. Check AKS service principal has Network Contributor role on VNet:
   ```bash
   az role assignment list --scope /subscriptions/<sub-id>/resourceGroups/<rg-name>/providers/Microsoft.Network/virtualNetworks/<vnet-name>
   ```
3. Check Azure Portal → Load Balancers for provisioning errors

### Issue: Portal UI shows 404 for client-side routes (e.g., `/apps`, `/infra`)

**Cause:** Portal UI's internal Nginx config not serving `index.html` for non-file requests

**Fix:**
Verify Portal UI's `nginx/default.conf` contains:
```nginx
location / {
    root /usr/share/nginx/html;
    try_files $uri /index.html;
}
```
Rebuild Portal UI container if missing.

### Issue: Platform API returns 404 for `/api/v1/*` requests

**Cause:** Incorrect path rewriting in Ingress annotations

**Fix:**
- Remove `nginx.ingress.kubernetes.io/rewrite-target` annotation if present
- Platform API expects full paths (e.g., `/api/v1/health`, not `/v1/health`)

### Issue: Grafana shows "Origin not allowed" or 404 errors

**Cause:** Grafana's `root_url` setting doesn't match Ingress path

**Fix:**
Verify `platform/monitoring/values.yaml` contains:
```yaml
grafana:
  grafana.ini:
    server:
      root_url: http://localhost:3000/grafana
      serve_from_sub_path: true
```
Update and re-sync Argo CD Application if missing.

### Issue: Prometheus doesn't scrape NGINX metrics

**Cause:** ServiceMonitor label mismatch with Prometheus operator selector

**Fix:**
1. Check Prometheus `serviceMonitorSelector`:
   ```bash
   kubectl get prometheus -n monitoring -o yaml | grep -A5 serviceMonitorSelector
   ```
2. Verify NGINX ServiceMonitor has matching label:
   ```bash
   kubectl get servicemonitor -n ingress-nginx ingress-nginx-controller -o yaml | grep prometheus:
   # Expected: prometheus: monitoring
   ```

### Issue: NGINX controller logs show "backend not found" errors

**Cause:** Backend service doesn't exist or isn't in the expected namespace

**Fix:**
```bash
# Verify all backend services exist
kubectl get svc -n platform platform-api portal-ui
kubectl get svc -n monitoring monitoring-grafana

# Check Ingress resource references correct services
kubectl get ingress -n platform platform-ingress -o yaml
```

---

## Future Enhancements

### Task #92: Update Portal UI API Configuration

**Current state:** Portal UI uses in-cluster DNS (`http://platform-api.platform.svc.cluster.local`)

**Future state:** Use relative path (`/api`) to route through Ingress

**File to update:** `portal/src/utils/config.ts`
```typescript
// Change:
apiUrl: import.meta.env.VITE_API_URL || '/api'
```

**Rebuild Portal UI container after this change** to apply new configuration.

### Task #93: Install cert-manager for TLS

**Scope:** Automated TLS certificate provisioning with Let's Encrypt

**Benefits:**
- HTTPS for all platform services
- Automatic certificate rotation
- Browser trust (no self-signed cert warnings)

**Implementation:**
- Install cert-manager (wave 2)
- Create ClusterIssuer (Let's Encrypt)
- Update Ingress with TLS annotations
- cert-manager automatically provisions Certificate resources

**Configuration ready:** `values.yaml` already has TLS protocol settings (TLSv1.2+, secure ciphers).

### Task #37: Add NGINX Ingress Grafana Dashboard

**Purpose:** Visualize NGINX Ingress metrics in Grafana

**Dashboard ID:** 9614 (Kubernetes Ingress-NGINX) from grafana.com

**Metrics to visualize:**
- Request rate by status code (2xx, 4xx, 5xx)
- Request latency (p50, p95, p99)
- Active connections
- Bandwidth usage
- Error rate trends

### Additional Future Work

**Rate limiting refinement:**
- Per-service rate limits (API stricter than Portal UI)
- Whitelist for platform monitoring tools
- Burst allowance for legitimate traffic spikes

**Access control:**
- Basic auth for Grafana (instead of default admin password)
- OAuth2 proxy for SSO integration
- IP whitelisting for sensitive endpoints

**Observability:**
- Distributed tracing headers (X-Request-ID)
- Access log shipping to Loki
- Alert rules for 5xx errors, high latency

---

## References

- [NGINX Ingress Controller Documentation](https://kubernetes.github.io/ingress-nginx/)
- [Helm Chart Repository](https://github.com/kubernetes/ingress-nginx/tree/main/charts/ingress-nginx)
- [Azure LoadBalancer Integration](https://kubernetes.github.io/ingress-nginx/deploy/#azure)
- [Prometheus Metrics](https://kubernetes.github.io/ingress-nginx/user-guide/monitoring/)
- [Path-based Routing](https://kubernetes.github.io/ingress-nginx/user-guide/multiple-ingress/)
- [TLS Configuration](https://kubernetes.github.io/ingress-nginx/user-guide/tls/)
