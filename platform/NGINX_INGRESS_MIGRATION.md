# NGINX Ingress Controller - Migration to Manifest-Based Deployment

## Background

**Date:** 2026-02-21
**Issue:** Helm-based NGINX Ingress Controller deployment caused persistent Argo CD sync failures due to admission webhook Jobs with TTL-based cleanup.

**Root Cause:** The NGINX Ingress Helm chart creates admission webhook Jobs (`create` and `patch`) that have `ttlSecondsAfterFinished` set, causing Kubernetes to delete them shortly after completion. Argo CD tracks these Jobs as sync hooks and waits for completion, but by the time it checks status, the Jobs have been cleaned up by the TTL controller. This results in Argo CD being stuck in "Running" state indefinitely.

**Attempted Fixes (unsuccessful):**
1. Set `ttlSecondsAfterFinished: null` in Helm values — insufficient schema coverage
2. Added `createSecretJob.ttlSecondsAfterFinished: null` — still missed some Jobs
3. Added `patchWebhookJob.ttlSecondsAfterFinished: null` — Jobs continued to have TTL

**Decision:** Abandon Helm-based installation, migrate to official Kubernetes manifest-based deployment.

---

## New Architecture

### Manifest-Based NGINX Ingress Controller

**Source:** https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.14.3/deploy/static/provider/cloud/deploy.yaml

**Benefits:**
- No Helm hooks → No Job TTL issues
- Simpler to debug (plain YAML, no templating)
- Official AKS deployment pattern
- Identical functionality to Helm chart

**Deployment:**
- Directory: `platform/nginx-ingress-controller/`
- Files: `deploy.yaml` (customized), `application.yaml` (Argo CD), `servicemonitor.yaml` (Prometheus)
- Sync wave: 3 (unchanged)
- LoadBalancer IP: `20.165.21.39`

### Application-Owned Ingress Resources

**Previous (wrong):** All Ingress resources centralized in `platform/nginx-ingress/ingresses/`

**New (correct):** Each application owns its Ingress resource
- `platform/portal-ui/ingress.yaml` — Portal UI routes
- `platform/platform-api/ingress.yaml` — Platform API routes
- `platform/monitoring/ingress-grafana.yaml` — Grafana routes

**Benefits:**
- Applications control their own routing
- GitOps-friendly: app owns all its K8s resources
- Easier to understand ownership
- Independent lifecycle management

### Hostname-Based Routing

**DNS:** `*.rdp.azurelaboratory.com` → `20.165.21.39` (configured by user)

**Hostnames:**
- `portal.rdp.azurelaboratory.com` → Portal UI
- `api.rdp.azurelaboratory.com` → Platform API
- `grafana.rdp.azurelaboratory.com` → Grafana

**Why hostname-based over path-based:**
- Cleaner URLs (no `/api` or `/grafana` paths)
- TLS/HTTPS ready (each hostname gets its own certificate)
- Production standard pattern
- No path rewriting needed

---

## Migration Tasks

Created tasks #94-#98 to implement this migration:

**#94:** Clean up Helm-based NGINX Ingress files
- Delete `platform/nginx-ingress/` directory
- Remove Helm repo from `platform/argocd/projects.yaml`

**#95:** Deploy NGINX Ingress Controller using manifests
- Download and customize official deploy.yaml
- Create Argo CD Application (wave 3)
- Add ServiceMonitor for Prometheus
- Verify LoadBalancer IP assignment

**#96:** Add Ingress resource to Portal UI
- Create `platform/portal-ui/ingress.yaml`
- Hostname: `portal.rdp.azurelaboratory.com`

**#97:** Add Ingress resource to Platform API
- Create `platform/platform-api/ingress.yaml`
- Hostname: `api.rdp.azurelaboratory.com`

**#98:** Add Ingress resource to Grafana
- Create `platform/monitoring/ingress-grafana.yaml`
- Hostname: `grafana.rdp.azurelaboratory.com`
- Update Grafana `root_url` config

**#92 (updated):** Update Portal UI API URL
- Change from in-cluster DNS to `https://api.rdp.azurelaboratory.com`
- Rebuild and redeploy container

---

## Lessons Learned

1. **Helm is not always better** — Manifests are simpler and more predictable for infrastructure controllers
2. **Argo CD + Helm hooks don't mix well** — TTL-based Job cleanup breaks Argo CD's sync tracking
3. **Co-locate resources with apps** — Applications should own all their Kubernetes resources, including Ingress
4. **Test before committing patterns** — The Helm approach looked good on paper but failed in practice
5. **Official docs are reliable** — The NGINX Ingress project recommends manifests for cloud providers (not Helm)

---

## References

- [NGINX Ingress Controller - Bare Metal/Cloud Deployment](https://kubernetes.github.io/ingress-nginx/deploy/#bare-metal-clusters)
- [Official Cloud Provider Manifest](https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.14.3/deploy/static/provider/cloud/deploy.yaml)
- [Argo CD Sync Phases and Waves](https://argo-cd.readthedocs.io/en/stable/user-guide/sync-waves/)
- Implementation plan: `/home/rodst/.claude/plans/humble-sauteeing-sketch.md`
