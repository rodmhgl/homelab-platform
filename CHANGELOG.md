# Changelog

All notable changes to the Homelab Platform IDP will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### Fixed - Portal UI Authentication & API Integration (2026-02-21)

**Portal UI v0.1.4** - Fixed critical runtime errors preventing dashboard from loading

**Issue #1: URL Construction Error**
- Browser error: "Failed to construct 'URL': Invalid URL"
- Root cause: `new URL('/api/v1/apps')` requires absolute URL when `VITE_API_URL` is empty (same-origin requests)
- Fix: Conditional URL building — absolute URLs use `URL` constructor, relative URLs use plain string concatenation
- Affected: `portal/src/api/client.ts`

**Issue #2: Missing Bearer Token Authentication**
- HTTP 401 errors from Platform API (requires Bearer token on all `/api/v1/*` endpoints)
- Fix: Added `Authorization: Bearer` header to all API requests
- Token: Static demo token `homelab-portal-token` (configurable via `VITE_API_TOKEN`)
- TODO: Replace with ExternalSecret + runtime injection when Platform API implements real token validation
- Affected: `portal/src/api/client.ts`, `portal/src/utils/config.ts`, `portal/.env.example`

**Issue #3: TypeScript Type Mismatch with Go API**
- Browser error: "Cannot read properties of undefined (reading 'length')"
- Root cause: Frontend types assumed API structure instead of matching actual Go struct JSON tags
- Mismatches:
  - Go returns `{ applications: [], total: 0 }` but TypeScript expected `{ apps: [], count: 0 }`
  - Go returns `{ lastDeployed: "..." }` but TypeScript expected `{ lastSyncedAt: "..." }`
- Fix: Aligned TypeScript types with actual Platform API response structure
- Affected: `portal/src/api/types.ts`, `portal/src/components/dashboard/ApplicationsPanel.tsx`

**Deployment:**
- v0.1.3: URL construction + Bearer token authentication fixes
- v0.1.4: API type alignment fixes
- Portal UI now successfully displays Argo CD applications at `http://portal.rdp.azurelaboratory.com`

### Added - Portal UI (2025-02-20)

**Portal UI React Application** (#78)

- Vite + React 18.3.1 + TypeScript project scaffold
- Tailwind CSS 3.4 with custom color palette
- React Router 6.28 for SPA routing
- TanStack Query 5.62 for server state management
- 22 TypeScript files implementing API client layer, layout, routing, common components
- Multi-stage Dockerfile (Node 22 → Nginx 1.27-alpine)
- Security-hardened deployment: non-root user, read-only rootfs, emptyDir volumes
- Kubernetes manifests: Deployment (2 replicas, wave 11), Service (ClusterIP), Ingress
- Applications panel (#79): Cards showing app sync status, health, project, last deployed time
- Comprehensive documentation in portal/README.md and platform/portal-ui/README.md

### Pending

- Dashboard panels (#80-#84): Infrastructure panel, Compliance Score donut, Policy Violations table, Vulnerability Feed, Security Events timeline
- Scaffold form (#85): Interactive project creation
- Detail pages: App detail, Infra detail, Compliance detail
- AI Ops panel (#86): kagent chat + HolmesGPT integration

### Changed

- Updated homelab-platform/CLAUDE.md with Portal UI status
- Updated CLAUDE.md (root) with Portal UI in repository structure
- Updated homelab-platform/README.md with Portal UI entry

## Earlier Work

See homelab-platform/README.md for full platform infrastructure and application layer implementation status.

[Unreleased]: https://github.com/rodmhgl/homelab-platform/compare/main...HEAD
