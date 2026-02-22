# NGINX Ingress Controller (Manifest-Based)

**Status:** ✅ Production-ready manifest-based deployment
**Source:** Official Kubernetes manifest (v1.14.3)
**Wave:** 3
**Namespace:** `ingress-nginx`

---

## Architecture

This is the **manifest-based** deployment of NGINX Ingress Controller, replacing the previous Helm-based installation that had Argo CD compatibility issues.

### Why Manifests Instead of Helm?

The Helm chart creates admission webhook Jobs with `ttlSecondsAfterFinished` that get deleted by Kubernetes shortly after completion. This breaks Argo CD's sync tracking because:

1. Argo CD waits for Jobs to complete
2. Kubernetes TTL controller deletes completed Jobs
3. Argo CD can't verify completion → stuck in "Running" state

**Solution:** Use official manifests with Argo CD hook annotations for proper GitOps compatibility.

---

## Components

### Core Resources

- **Namespace:** `ingress-nginx`
- **Deployment:** `ingress-nginx-controller` — 1 replica, 100m/256Mi requests, 500m/512Mi limits
- **LoadBalancer Service:** `ingress-nginx-controller` — ports 80/443, `externalTrafficPolicy: Local`
- **Metrics Service:** `ingress-nginx-controller-metrics` — port 10254 for Prometheus
- **Admission Service:** `ingress-nginx-controller-admission` — internal webhook validation

### Certificate Generation (Argo CD Hooks)

- **Job:** `ingress-nginx-admission-create` — generates TLS cert for admission webhook (PreSync hook)
- **Job:** `ingress-nginx-admission-patch` — patches ValidatingWebhookConfiguration (PreSync hook)

Both Jobs use `argocd.argoproj.io/hook: PreSync` and `argocd.argoproj.io/hook-delete-policy: BeforeHookCreation` to:
- Run before main resources sync
- Auto-delete before re-creation on updates
- NOT count toward Application health status

---

## Customizations

### vs. Vanilla Upstream Manifest

| Change | Reason |
|--------|--------|
| Added resource limits (500m CPU / 512Mi memory) | Prevent resource exhaustion in homelab |
| Added Azure health probe annotation | Proper Azure Load Balancer health checks |
| Added Prometheus port (10254) to Deployment | Metrics scraping |
| Created `ingress-nginx-controller-metrics` Service | Expose metrics for Prometheus |
| Added Argo CD hook annotations to Jobs | GitOps compatibility |

### Azure Integration

LoadBalancer Service includes:
```yaml
metadata:
  annotations:
    service.beta.kubernetes.io/azure-load-balancer-health-probe-request-path: /healthz
spec:
  externalTrafficPolicy: Local  # Preserve source IP
```

---

## Monitoring

**ServiceMonitor:** `servicemonitor.yaml` — targets `ingress-nginx-controller-metrics:10254/metrics`

**Metrics exposed:**
- Request counts, latencies, response codes
- SSL certificate expiry
- Ingress configuration reload count
- Backend health status

**Label selector:** `prometheus: monitoring` (matches kube-prometheus-stack)

---

## Deployment

```bash
# Deployed via Argo CD (wave 3)
kubectl get application -n argocd nginx-ingress-controller

# Verify pods running
kubectl get pods -n ingress-nginx

# Check LoadBalancer IP assignment
kubectl get svc -n ingress-nginx ingress-nginx-controller

# Expected EXTERNAL-IP: 20.165.21.39 (or pending on first deploy)
```

### DNS Configuration (Manual)

Once the LoadBalancer IP is assigned, create DNS A records:

```
*.rdp.azurelaboratory.com → 20.165.21.39
```

This enables hostname-based routing for:
- `portal.rdp.azurelaboratory.com` → Portal UI
- `api.rdp.azurelaboratory.com` → Platform API
- `grafana.rdp.azurelaboratory.com` → Grafana

---

## Application Ingress Resources

**Each application owns its Ingress resource** (not centralized):

- `platform/portal-ui/ingress.yaml` — Portal UI routes
- `platform/platform-api/ingress.yaml` — Platform API routes
- `platform/monitoring/ingress-grafana.yaml` — Grafana routes

**Pattern:** Hostname-based routing (not path-based) for cleaner URLs and TLS readiness.

---

## Troubleshooting

### LoadBalancer stuck in Pending

```bash
# Check AKS node pool for Load Balancer SKU mismatch
az aks show -g rg-homelab-aks-dev -n homelab-aks-dev --query "networkProfile.loadBalancerSku"

# Expected: "standard" (AKS uses Standard LB by default)
```

### Admission webhook errors

```bash
# Check if Jobs ran successfully
kubectl get jobs -n ingress-nginx

# Expected:
# ingress-nginx-admission-create   1/1     (completed)
# ingress-nginx-admission-patch    1/1     (completed)

# If Jobs are missing (deleted by TTL), Argo CD will recreate on next sync
kubectl get app -n argocd nginx-ingress-controller -o yaml | grep -A 5 "hook"
```

### Metrics not showing in Prometheus

```bash
# Verify ServiceMonitor exists
kubectl get servicemonitor -n ingress-nginx

# Check Prometheus targets
kubectl port-forward -n monitoring svc/prometheus-operated 9090:9090
# Open http://localhost:9090/targets
# Look for "ingress-nginx/nginx-ingress-controller/0"
```

---

## References

- [Official NGINX Ingress Docs](https://kubernetes.github.io/ingress-nginx/)
- [Cloud Provider Manifest Source](https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.14.3/deploy/static/provider/cloud/deploy.yaml)
- [Argo CD Sync Waves](https://argo-cd.readthedocs.io/en/stable/user-guide/sync-waves/)
- Migration rationale: `platform/NGINX_INGRESS_MIGRATION.md`
