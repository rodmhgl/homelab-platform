# Portal UI Deployment

Portal UI is the web-based dashboard for the Homelab Platform IDP. It provides a visual interface for all Platform API functionality.

## Architecture

- **Framework:** React 18.3.1 + TypeScript + Tailwind CSS
- **Build:** Vite 6.x
- **Runtime:** Nginx 1.27-alpine
- **Deployment:** 2 replicas, wave 11 (after Platform API wave 10)

## Local Development

```bash
cd portal
npm install
npm run dev
```

The dev server will start at `http://localhost:5173` by default.

## Building Docker Image

```bash
cd homelab-platform
docker build \
  --build-arg VITE_API_URL=http://platform-api.platform.svc.cluster.local \
  -t homelabplatformacr.azurecr.io/portal-ui:v0.1.0 \
  portal/
```

## Push to ACR

```bash
az acr login --name homelabplatformacr
docker push homelabplatformacr.azurecr.io/portal-ui:v0.1.0
```

## Deployment

Portal UI is deployed via Argo CD. After pushing the image, commit the Kubernetes manifests:

```bash
git add platform/portal-ui/
git commit -m "feat(portal): add Portal UI deployment"
git push origin main
```

Argo CD will automatically sync the application.

## Verify Deployment

```bash
# Check Argo CD application status
kubectl get applications -n argocd portal-ui

# Check pods
kubectl get pods -n platform -l app.kubernetes.io/name=portal-ui

# Check service
kubectl get svc -n platform portal-ui
```

## Access Portal UI

### Via port-forward (development)

```bash
kubectl port-forward -n platform svc/portal-ui 8080:80
```

Then open `http://localhost:8080` in your browser.

### Via Ingress (future)

An Ingress will be added in a future enhancement to expose the Portal UI externally.

## Configuration

The Portal UI is configured with the Platform API URL at build time via the `VITE_API_URL` build arg.

Current configuration:
- **API URL:** `http://platform-api.platform.svc.cluster.local`

## Security

- **Non-root user:** Runs as UID 1000
- **Read-only rootfs:** Filesystem is read-only with emptyDir volumes for cache and tmp
- **Security headers:** CSP, X-Frame-Options, X-Content-Type-Options
- **Capabilities dropped:** All Linux capabilities dropped
- **SeccompProfile:** RuntimeDefault

## Troubleshooting

### Portal UI pods not starting

Check pod logs:
```bash
kubectl logs -n platform -l app.kubernetes.io/name=portal-ui
```

Common issues:
- Image pull errors (check ACR credentials)
- Read-only filesystem errors (check volume mounts for /var/cache/nginx and /tmp)

### Cannot connect to Platform API

Check:
1. Platform API service is running: `kubectl get svc -n platform platform-api`
2. Platform API pods are healthy: `kubectl get pods -n platform -l app.kubernetes.io/name=platform-api`
3. Network policies allow traffic from portal-ui to platform-api

### Health check failing

The health check endpoint is `/healthz`. Test manually:
```bash
kubectl exec -n platform <portal-ui-pod> -- wget -O- http://localhost:8080/healthz
```

## Future Enhancements

- [ ] Add CORS middleware to Platform API
- [ ] Implement authentication (token-based)
- [ ] Add Ingress for external access
- [ ] Add detail pages for apps, infra, compliance
- [ ] Add AI Ops panel (kagent chat + HolmesGPT)
