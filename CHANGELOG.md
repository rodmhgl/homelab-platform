# Changelog

All notable changes to the Homelab Platform IDP will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### Added - Portal UI (2025-02-20)

**Portal UI React Application** (#78)

- Vite + React 18.3.1 + TypeScript project scaffold
- Tailwind CSS 3.4 with custom color palette
- React Router 6.28 for SPA routing
- TanStack Query 5.62 for server state management
- 22 TypeScript files implementing API client layer, layout, routing, common components
- Multi-stage Dockerfile (Node 22 → Nginx 1.27-alpine)
- Security-hardened deployment: non-root user, read-only rootfs, emptyDir volumes
- Kubernetes manifests: Deployment (2 replicas, wave 11), Service (ClusterIP)
- Comprehensive documentation in portal/README.md and platform/portal-ui/README.md

### Pending

- Dashboard panels (#79-#84): Applications, Infrastructure, Compliance Score, Policy Violations, Vulnerability Feed, Security Events
- Scaffold form (#85): Interactive project creation
- CORS middleware, authentication, Ingress, detail pages, AI Ops panel (#86)

### Changed

- Updated homelab-platform/CLAUDE.md with Portal UI status
- Updated CLAUDE.md (root) with Portal UI in repository structure
- Updated homelab-platform/README.md with Portal UI entry

## Earlier Work

See homelab-platform/README.md for full platform infrastructure and application layer implementation status.

[Unreleased]: https://github.com/rodmhgl/homelab-platform/compare/main...HEAD
