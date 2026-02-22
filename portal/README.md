# Portal UI

**React-based web dashboard for the Homelab Platform IDP.**

The Portal UI is a thin client that renders data from the Platform API. It provides a visual interface for all platform functionality: applications, infrastructure, compliance monitoring, and project scaffolding.

## Technology Stack

| Layer | Technology |
| --- | --- |
| **Framework** | React 18.3.1 + TypeScript |
| **Build** | Vite 6.x |
| **Styling** | Tailwind CSS 3.4.x |
| **Routing** | React Router 6.28 |
| **State** | TanStack Query 5.62 (server state) |
| **Charts** | Recharts 2.15 |
| **Runtime** | Nginx 1.27-alpine |

## Project Structure

```
portal/
├── src/
│   ├── api/              # API client + TypeScript types
│   │   ├── client.ts     # HTTP client wrapper
│   │   ├── types.ts      # Types mirroring Go API
│   │   ├── apps.ts       # Argo CD endpoints
│   │   ├── infra.ts      # Crossplane endpoints
│   │   ├── compliance.ts # Compliance endpoints
│   │   ├── scaffold.ts   # Scaffold endpoint
│   │   └── health.ts     # Health check endpoint
│   ├── components/
│   │   ├── common/       # Reusable components (Badge, LoadingSpinner, StatusCard)
│   │   ├── dashboard/    # Dashboard panels (ApplicationsPanel, InfrastructurePanel, CompliancePanel, PolicyViolationsPanel)
│   │   └── layout/       # Layout components (AppShell, Sidebar, Header)
│   ├── pages/            # Route pages
│   │   ├── Dashboard.tsx
│   │   ├── Applications.tsx
│   │   ├── Infrastructure.tsx
│   │   ├── Compliance.tsx
│   │   ├── Scaffold.tsx
│   │   └── NotFound.tsx
│   ├── hooks/            # Custom hooks (future)
│   ├── utils/            # Config, formatters
│   ├── App.tsx           # Root component with routing
│   ├── main.tsx          # Entry point
│   └── index.css         # Tailwind imports
├── nginx/
│   └── default.conf      # Nginx config (SPA routing, security headers)
├── Dockerfile            # Multi-stage build (Node → Nginx)
└── package.json
```

## Current Status

✅ **Phase 1 (task #78): Project Foundation**
- Vite + React + TypeScript project initialized
- Tailwind CSS configured
- Directory structure created

✅ **Phase 2: API Client Layer**
- TypeScript types mirroring Go API responses
- HTTP client wrapper with error handling
- Endpoint modules for apps, infra, compliance, scaffold, health

✅ **Phase 3: Layout & Routing**
- AppShell, Sidebar, Header components
- React Router with page stubs
- TanStack Query integration

✅ **Phase 4: Common Components**
- Badge, LoadingSpinner, StatusCard

✅ **Phase 5: Containerization**
- Multi-stage Dockerfile (Node 22 → Nginx 1.27-alpine)
- Nginx configuration with SPA routing, security headers, health check
- .dockerignore

✅ **Phase 6: Kubernetes Deployment**
- Deployment manifest (2 replicas, security context, emptyDir volumes)
- Service manifest (ClusterIP port 80 → 8080)
- Argo CD Application (wave 11)

🔨 **Phase 7: Dashboard Components (tasks #79-#84)**
- ✅ Applications panel (#79) — Argo CD apps with sync status, health, project, last deployed time
- ✅ Infrastructure panel (#80) — Crossplane Claims with ready/synced status, connection secrets, creation time
- ✅ Compliance Score donut chart (#81) — Color-coded score with violations + vulnerabilities breakdown
- ✅ Policy Violations table (#82) — Gatekeeper audit violations with constraint details, severity badges
- Pending: Vulnerability Feed (#83)
- Pending: Security Events timeline (#84)

⬜ **Phase 8: Scaffold Form (task #85)**
- Pending: Project creation form with storage/vault toggles

## Development

### Install dependencies

```bash
npm install
```

### Run dev server

```bash
npm run dev
```

Vite dev server starts at `http://localhost:5173` (hot module replacement enabled).

### Build for production

```bash
npm run build
```

Output: `dist/` directory (served by Nginx in container).

### Lint

```bash
npm run lint
```

## Configuration

### API URL

The Platform API URL is configured at **build time** via the `VITE_API_URL` environment variable.

Default: `http://platform-api.platform.svc.cluster.local`

Override:
```bash
VITE_API_URL=http://custom-api-url npm run build
```

Or in Dockerfile:
```dockerfile
ARG VITE_API_URL=http://custom-api-url
```

## Docker

### Build image

```bash
cd homelab-platform
docker build \
  --build-arg VITE_API_URL=http://platform-api.platform.svc.cluster.local \
  -t homelabplatformacr.azurecr.io/portal-ui:v0.1.0 \
  portal/
```

### Push to ACR

```bash
az acr login --name homelabplatformacr
docker push homelabplatformacr.azurecr.io/portal-ui:v0.1.0
```

### Run locally

```bash
docker run -p 8080:8080 homelabplatformacr.azurecr.io/portal-ui:v0.1.0
```

Access at `http://localhost:8080`.

## Deployment

Portal UI is deployed to the `platform` namespace via Argo CD (wave 11).

### Verify deployment

```bash
# Check Argo CD application
kubectl get applications -n argocd portal-ui

# Check pods
kubectl get pods -n platform -l app.kubernetes.io/name=portal-ui

# Check service
kubectl get svc -n platform portal-ui
```

### Access via port-forward

```bash
kubectl port-forward -n platform svc/portal-ui 8080:80
```

Open `http://localhost:8080` in browser.

## Security

- **Non-root user:** Runs as UID 1000
- **Read-only rootfs:** Filesystem is read-only with emptyDir volumes for `/var/cache/nginx` and `/tmp`
- **Capabilities dropped:** All Linux capabilities dropped
- **SeccompProfile:** RuntimeDefault
- **Security headers:** CSP, X-Frame-Options, X-Content-Type-Options, XSS-Protection, Referrer-Policy

## Next Steps

1. **Implement remaining dashboard panels** (4 of 6 complete)
   - ✅ Applications panel (#79) — Argo CD apps with sync status, health, last deployed
   - ✅ Infrastructure panel (#80) — Crossplane Claims with ready/synced status, connection secrets
   - ✅ Compliance Score panel (#81) — Donut chart with color-coded severity, violations + vulnerabilities breakdown
   - ✅ Policy Violations table (#82) — Gatekeeper audit violations with constraint details, severity badges, scrollable table
   - ⬜ Vulnerability Feed (#83) — Trivy CVEs grouped by image
   - ⬜ Security Events timeline (#84) — Falco alerts with real-time polling

2. **Implement scaffold form** (task #85)
   - Template selector (go-service, python-service)
   - Project name validation (DNS label format)
   - Storage/Vault toggles
   - GitHub settings (owner, repo, visibility)

3. **Add CORS middleware to Platform API**
   - Currently missing; will be needed when Portal UI is deployed

4. **Implement authentication**
   - Token-based auth for Portal UI
   - Platform API has TODOs for token validation

5. **Add Ingress**
   - Expose Portal UI externally (currently ClusterIP only)

6. **Detail pages**
   - App detail (`/apps/:name`)
   - Infra detail (`/infra/:kind/:name`)
   - Compliance detail

7. **AI Ops panel** (task #86)
   - kagent chat interface
   - HolmesGPT investigation trigger

## API Integration

Portal UI consumes the following Platform API endpoints:

| Endpoint | Purpose |
| --- | --- |
| `GET /health` | Health check (polled every 30s) |
| `GET /api/v1/apps` | List Argo CD apps |
| `GET /api/v1/apps/:name` | Get app details |
| `POST /api/v1/apps/:name/sync` | Sync app |
| `GET /api/v1/infra` | List all Claims |
| `GET /api/v1/infra/storage` | List StorageBucket Claims |
| `GET /api/v1/infra/vaults` | List Vault Claims |
| `GET /api/v1/infra/:kind/:name` | Get Claim resource tree |
| `POST /api/v1/infra` | Create Claim (GitOps) |
| `DELETE /api/v1/infra/:kind/:name` | Delete Claim (GitOps) |
| `GET /api/v1/compliance/summary` | Compliance score + summary |
| `GET /api/v1/compliance/violations` | Gatekeeper violations |
| `GET /api/v1/compliance/vulnerabilities` | Trivy CVEs |
| `GET /api/v1/compliance/events` | Falco security events |
| `POST /api/v1/scaffold` | Create new project |

## TypeScript Types

All API response types are defined in `src/api/types.ts` and mirror the Go structs from the Platform API:

- `ApplicationSummary`, `Application` (Argo CD)
- `ClaimSummary`, `ClaimResource`, `CompositeResource`, `ManagedResource` (Crossplane)
- `SummaryResponse`, `Violation`, `Vulnerability`, `SecurityEvent` (Compliance)
- `ScaffoldRequest`, `ScaffoldResponse`

## Troubleshooting

### Build errors

```bash
# Clean build artifacts
rm -rf dist node_modules
npm install
npm run build
```

### Type errors

```bash
# Check TypeScript
npm run build
```

Vite runs `tsc -b` before building.

### Health check failing in container

Test manually:
```bash
kubectl exec -n platform <portal-ui-pod> -- wget -O- http://localhost:8080/healthz
```

Should return `ok`.

### Cannot connect to Platform API

Check:
1. Platform API service exists: `kubectl get svc -n platform platform-api`
2. Platform API pods are healthy: `kubectl get pods -n platform -l app.kubernetes.io/name=platform-api`
3. DNS resolution works from Portal UI pod:
   ```bash
   kubectl exec -n platform <portal-ui-pod> -- nslookup platform-api.platform.svc.cluster.local
   ```

### CORS errors in browser

Platform API needs CORS middleware (currently TODO). For development, use `kubectl port-forward` to forward both Portal UI and Platform API to localhost, avoiding CORS issues.

## Contributing

When adding new components:

1. **API types:** Update `src/api/types.ts` if new API endpoints are added
2. **API client:** Add new endpoint modules in `src/api/`
3. **Components:** Follow the existing structure (`common`, `dashboard`, `layout`)
4. **Styling:** Use Tailwind CSS utility classes (avoid inline styles)
5. **State:** Use TanStack Query for server state, React hooks for local state
6. **Testing:** Build the app (`npm run build`) to verify TypeScript types and bundle size

## License

Part of the Homelab Platform IDP mono-repo.
