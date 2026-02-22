# Changelog

All notable changes to the Homelab Platform IDP will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### Added - Portal UI Policy Violations Panel (2026-02-22)

**Dashboard Enhancement** - Completed task #82: Policy Violations table panel

**Features:**
- Scrollable table displaying Gatekeeper audit violations with 5 columns:
  - Constraint name (policy that was violated)
  - Constraint kind (with color-coded severity badges: red for security, yellow for policy, blue for other)
  - Resource path (namespace/kind/name in monospace font)
  - Namespace (or "-" for cluster-scoped resources)
  - Violation message (remediation guidance)
- Auto-refreshes every 30 seconds via TanStack Query
- Empty state: "✓ No policy violations found" when compliant
- Sticky table headers for improved UX when scrolling through 20+ violations
- TypeScript type alignment with Go API (critical bug fix)

**Technical Details:**
- **Component:** `portal/src/components/dashboard/PolicyViolationsPanel.tsx`
- **API Integration:** `GET /api/v1/compliance/violations` (Platform API)
- **Type Safety:** Fixed `Violation` interface to match Go struct JSON tags exactly:
  - `constraintName` (not `constraint`)
  - `constraintKind` (not `kind`)
  - `resource` (not `name`)
- **Dashboard Progress:** 4 of 6 panels complete (#79, #80, #81, #82)

**Files Changed:**
- `portal/src/api/types.ts` - Fixed Violation/ListViolationsResponse types
- `portal/src/components/dashboard/PolicyViolationsPanel.tsx` - NEW
- `portal/src/pages/Dashboard.tsx` - Added PolicyViolationsPanel to grid
- `CLAUDE.md` - Updated panel count (25 TS files, 4/6 complete)

**Deployment:**
- Ready for production (build verified, API tested)
- Next version: `portal-ui:v0.1.5`

---

### Fixed - Trivy Operator CVE Scanning (2026-02-22)

**Trivy Operator Configuration** - Fixed vulnerability scanning to enable real compliance data

**Problem:**
- Trivy Operator was installed but **not generating any VulnerabilityReport CRDs**
- Compliance score showed 100% (misleading — no vulnerability data available)
- Root causes: DB repository configuration + CRI socket access issues

**Fixes Applied:**

1. **DB Repository Configuration** (commit `15ba7fa`)
   - Removed version tags from DB repository URLs (`:2` and `:1` caused MANIFEST_UNKNOWN errors)
   - Updated to use AKS mirror: `mirror.gcr.io/aquasec/trivy-db` (no version tag)
   - Trivy now successfully downloads vulnerability database

2. **Containerd Socket Configuration** (commit `00f5605`)
   - Added `podTemplateVolumeMounts` to mount `/run/containerd/containerd.sock`
   - Added `podTemplateVolumes` to expose host containerd socket as `hostPath`
   - Scan jobs can now access container images from node's CRI
   - Eliminates ACR authentication errors (uses kubelet's managed identity)

**Configuration Changes:**
```yaml
trivy:
  dbRegistry: "mirror.gcr.io"
  dbRepository: "aquasec/trivy-db"  # No :2 tag
  javaDbRegistry: "mirror.gcr.io"
  javaDbRepository: "aquasec/trivy-java-db"  # No :1 tag

scanJob:
  podTemplateVolumeMounts:
    - name: containerd-sock
      mountPath: /run/containerd/containerd.sock
      readOnly: true
  podTemplateVolumes:
    - name: containerd-sock
      hostPath:
        path: /run/containerd/containerd.sock
        type: Socket
```

**Impact:**
- ✅ VulnerabilityReport CRDs are now being generated (8+ reports and counting)
- ✅ Scanned images: nginx, falco, falcoctl, falcosidekick, ingress-nginx, platform-api, portal-ui
- ✅ Compliance score will now reflect **actual CVE data** (expected to drop from 100%)
- ✅ Portal UI Compliance Score panel displays real vulnerability counts
- ⚠️ Some cache lock errors during concurrent scans (non-blocking, reports still generated)

**Verification:**
```bash
# View generated VulnerabilityReports
kubectl get vulnerabilityreports -A

# Check compliance score with real data
curl -H "Authorization: Bearer homelab-portal-token" \
  http://portal.rdp.azurelaboratory.com/api/v1/compliance/summary
```

**Related Files:**
- `platform/trivy-operator/values.yaml` — Updated DB repo + CRI socket configuration
- `platform/trivy-operator/application.yaml` — Argo CD sync wave 7

**Related Tasks:**
- Task #32 ✅ — Trivy Operator install (original)
- Task #81 ✅ — Compliance Score panel (now displays real data)

### Added - Portal UI Compliance Score Panel (2026-02-22)

**Portal UI v0.1.7** - Compliance Score donut chart implementation (#81)

**New Features:**
- Compliance Score panel displays overall platform compliance (0-100 percentage)
- Donut chart visualization with Recharts (PieChart with innerRadius for hollow center)
- Color-coded severity indicators:
  - Green (≥90): Healthy compliance posture
  - Amber (70-89): Moderate risk
  - Red (<70): High risk requiring attention
- Large centered score number above chart (responsive font size)
- Breakdown metrics in 2-column grid:
  - Policy Violations with severity badges (policy, config, security)
  - Vulnerabilities with CRITICAL/HIGH/MEDIUM/LOW severity badges
- Auto-refresh every 30 seconds (consistent with other dashboard panels)
- Loading state, error state, and empty state handling
- Responsive dashboard layout: 1 column (mobile) → 2 columns (desktop) → 3 columns (wide)

**Critical Bug Fix:**
- **BLOCKING:** Fixed critical TypeScript type mismatch in `SummaryResponse` interface
- Previous interface expected speculative field names (`score`, `timestamp`, nested objects)
- Go API returns completely different structure (`complianceScore`, flat totals, severity maps)
- **Root cause:** TypeScript types written before reading Go implementation (same issue as #79)
- **Impact:** Would have caused runtime errors: `Cannot read properties of undefined`
- **Fix:** Replaced `SummaryResponse` interface to match Go struct JSON tags exactly:
  ```typescript
  complianceScore: number;                           // was: score
  totalViolations: number;                           // was: violations.total
  totalVulnerabilities: number;                      // was: vulnerabilities.total
  violationsBySeverity: Record<string, number>;      // new field
  vulnerabilitiesBySeverity: Record<string, number>; // new field
  ```
- Removed unused interfaces: `ViolationSummary`, `VulnerabilitySummary`, `SecurityEventSummary`

**Component Files:**
- New: `portal/src/components/dashboard/CompliancePanel.tsx` (123 lines)
- Updated: `portal/src/pages/Dashboard.tsx` (added CompliancePanel + 3-column grid)
- Updated: `portal/src/api/types.ts` (fixed SummaryResponse interface)

**Chart Implementation Details:**
- Recharts PieChart with `innerRadius={60}`, `outerRadius={80}` for donut effect
- `startAngle={90}` to start at top (12 o'clock position)
- Two-segment data: "Compliant" (colored) + "At Risk" (gray)
- Creates visual "fill meter" effect (100% compliance = full green circle)
- Conditional rendering for severity badges (only show non-zero counts)

**Dashboard Layout:**
- Three-column grid on extra-large screens (xl:grid-cols-3)
- Two-column grid on desktop (lg:grid-cols-2)
- Single column on mobile
- All panels at equal hierarchy (Applications | Infrastructure | Compliance)

**Integration with Platform API:**
- Consumes `GET /api/v1/compliance/summary` endpoint (#48)
- Compliance score formula (from api/internal/compliance/handler.go):
  - `max(0, 100 - (violations × 5) - (critical_cves × 10) - (high_cves × 5) - (critical_events × 15) - (error_events × 8))`
  - Falco Critical events weighted heaviest (15 points) as active threats vs potential vulnerabilities

**Testing:**
- TypeScript compilation successful (`npm run build`)
- No type errors related to SummaryResponse
- Portal UI builds without warnings

**Documentation Updates:**
- Updated homelab-platform/CLAUDE.md with CompliancePanel completion
- Updated homelab-platform/README.md with dashboard panel status (3 of 6 complete)
- Updated portal/README.md Phase 7 progress

**Related Tasks:**
- Task #48 ✅ — Platform API compliance summary endpoint (dependency)
- Task #79 ✅ — Applications panel pattern (reference implementation)
- Task #80 ✅ — Infrastructure panel pattern (reference implementation)
- Task #82 (pending) — Policy Violations table
- Task #83 (pending) — Vulnerability Feed
- Task #84 (pending) — Security Events timeline

### Added - Portal UI Infrastructure Panel (2026-02-22)

**Portal UI v0.1.5** - Infrastructure Panel implementation (#80)

**New Features:**
- Infrastructure panel displays Crossplane Claims (StorageBucket + Vault resources)
- Side-by-side dashboard layout with Applications panel (responsive grid: stacks on mobile, side-by-side on desktop)
- Status indicators: Ready/Progressing/Failed badges
- Ready/Synced status visualization with color-coded badges
- Connection secret name display
- Creation timestamp with human-readable format ("Xd ago")
- Auto-refresh every 30 seconds via TanStack Query
- Summary footer: "Showing X claim(s) (Y ready)"
- Empty state, loading state, and error state handling

**Type Safety Improvements:**
- Fixed critical TypeScript type mismatches between frontend and Go API:
  - `ListClaimsResponse.count` → `ListClaimsResponse.total`
  - Added missing `ClaimSummary` fields: `synced`, `ready`, `labels`
  - Renamed `ClaimSummary.createdAt` → `ClaimSummary.creationTimestamp`
  - Updated `ClaimResource`, `CompositeResource`, `ManagedResource`, `KubernetesEvent` to match Go structs
  - Added `ResourceRef` interface for resource references

**Component Files:**
- New: `portal/src/components/dashboard/InfrastructurePanel.tsx` (155 lines)
- Updated: `portal/src/pages/Dashboard.tsx` (added InfrastructurePanel import + component)
- Updated: `portal/src/api/types.ts` (comprehensive type alignment with Go API)

**Layout Changes:**
- Removed `col-span-2` classes from both ApplicationsPanel and InfrastructurePanel
- Panels now respect two-column grid layout on large screens (>= 1024px)
- Maintains mobile-friendly vertical stacking on smaller screens

**Deployment:**
- Portal UI successfully rebuilt and deployed
- Dashboard now shows both Applications and Infrastructure panels
- Accessible at `http://portal.rdp.azurelaboratory.com`

**Documentation Updates:**
- Updated homelab-platform/CLAUDE.md with Infrastructure panel completion
- Updated homelab-platform/README.md with dashboard panel status
- Updated portal/README.md Phase 7 progress (2 of 6 panels complete)

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
