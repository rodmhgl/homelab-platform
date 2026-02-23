# Changelog

All notable changes to the Homelab Platform IDP will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### Added - CLI Interactive Infrastructure Creation (2026-02-23)

**CLI Enhancement** - Completed tasks #69 and #70: Interactive `rdp infra create` commands with bubbletea TUI

**Features:**

**Interactive StorageBucket Creation:**
- **`rdp infra create storage`** - Guided wizard for Azure Storage Account provisioning
  - Sequential field entry: name → namespace → location → tier → redundancy → versioning → repo owner → repo name
  - **DNS Label Validation**: Enforces Kubernetes DNS label format (`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`, max 63 chars)
  - **Location Whitelist**: Only allows `southcentralus` and `eastus2` (matches Gatekeeper constraints)
  - **Git Auto-Detection**: Parses `git remote get-url origin` for both SSH (`git@github.com:owner/repo`) and HTTPS (`https://github.com/owner/repo`) formats
  - **GitOps Flow**: Commits Claim YAML to `k8s/claims/<name>.yaml` via Platform API, Argo CD syncs within 60s
  - **Field Options**:
    - Tier: Standard, Premium
    - Redundancy: LRS, ZRS, GRS, GZRS, RAGRS, RAGZRS
    - Versioning: Y/N toggle
    - Public Access: Always `false` (enforced by Gatekeeper, never exposed to user)

**Interactive Vault Creation:**
- **`rdp infra create vault`** - Guided wizard for Azure Key Vault provisioning
  - Sequential field entry: name → namespace → location → SKU → retention days → repo owner → repo name
  - **DNS Label Validation**: Same strict format as storage
  - **Retention Validation**: Range check (7-90 days)
  - **Git Auto-Detection**: Same SSH/HTTPS parsing as storage
  - **GitOps Flow**: Commits Claim YAML to `k8s/claims/<name>.yaml` via Platform API
  - **Field Options**:
    - SKU: standard, premium
    - Soft Delete Retention: 7-90 days (numeric input with validation)

**TUI Architecture:**
- **Framework**: Bubbletea (Elm architecture) + Lipgloss (styling) + Bubbles (text inputs)
- **State Machine**: Welcome → Input Fields → Confirmation → Submitting → Success/Error
- **Navigation**: Arrow keys (select lists), Enter (advance), Y/N (confirmation), R (retry on error), Q (quit)
- **Visual Design**:
  - Title bar with command name
  - Completed fields shown with ✓ checkmarks above current field
  - Current field highlighted with input box or selection list
  - Inline error messages in red
  - Help text below (gray)
  - Success screen: Green ✓, commit SHA, file path, connection secret name, repo URL
  - Error screen: Red ✗, error message, retry/quit options

**Shared TUI Components** (`cli/internal/tui/shared.go`):
- **Styles**: Title, help, error, success, field label/value, status icons
- **Validators**: `ValidateDNSLabel()`, `ValidateNamespace()`, `ValidateLocation()`, `ValidateRetentionDays()`
- **Git Helpers**: `DetectGitRepo()`, `ParseGitURL()` (handles SSH/HTTPS)
- **View Helpers**: `RenderFieldRow()`, `RenderSpinner()`, `RenderSuccess()`, `RenderError()`

**Platform API Integration:**
- **Endpoint**: `POST /api/v1/infra`
- **Request**: `CreateClaimRequest` with kind, name, namespace, parameters, repoOwner, repoName
- **Response**: `CreateClaimResponse` with commitSha, filePath, connectionSecret, repoUrl
- **Timeout**: 30 seconds (allows time for GitHub API commits)
- **Error Handling**: HTTP status codes (400 validation, 500 server error), structured error messages

**Critical Implementation Details:**
- **No Placeholder Values**: All parameters fully functional (not TODO stubs)
- **Gatekeeper Alignment**: Location whitelist and publicAccess=false match `CrossplaneClaimLocation` and `CrossplaneNoPublicAccess` constraints
- **Config Validation**: `ValidateConfig()` called before TUI launch (ensures API URL + auth token present)
- **Graceful Degradation**: Works outside Git repos (prompts for owner/repo), handles network errors with retry

**Files Added:**
- `cli/cmd/infra_create.go` - 95 lines (NEW) - Cobra commands for `create storage` and `create vault`
- `cli/internal/tui/shared.go` - 207 lines (NEW) - Shared TUI styles, validators, Git helpers
- `cli/internal/tui/create_storage.go` - 480 lines (NEW) - StorageBucket TUI model and logic
- `cli/internal/tui/create_vault.go` - 418 lines (NEW) - Vault TUI model and logic

**Files Modified:**
- `cli/go.mod` - Added bubbletea v0.25.0, lipgloss v0.10.0, bubbles v0.18.0
- `cli/cmd/infra.go` - Updated help text to list `create storage` and `create vault`
- `cli/README.md` - Added interactive create examples with field descriptions
- `cli/IMPLEMENTATION_STATUS.md` - Marked #69 and #70 complete, updated progress table
- `homelab-platform/CLAUDE.md` - Updated `cli/` status line to include bubbletea

**CLI Progress:**
- ✅ Root command + config management (#65)
- ✅ Version command
- ✅ `rdp status` - Platform health summary (#66)
- ✅ `rdp infra list/status` - Infrastructure Claims (#68)
- ✅ `rdp infra create storage/vault` - Interactive infra creation (#69, #70) **← NEW**
- ✅ `rdp apps list/status/sync` - Application management (#67)
- ⬜ `rdp infra delete` - Interactive infra deletion (#71)
- ⬜ `rdp compliance` - Policy violations, CVEs, events (#73)
- ⬜ `rdp secrets` - Secret management (#74)
- ⬜ `rdp scaffold create` - Interactive project creation (#72)
- ⬜ `rdp investigate` - HolmesGPT integration (#75)
- ⬜ `rdp ask` - kagent natural language (#76)

**Next:** Infrastructure deletion command (#71) or compliance commands (#73).

---

### Added - CLI Application Management Commands (2026-02-23)

**CLI Enhancement** - Completed task #67: `rdp apps` command group

**Features:**

**Full Argo CD Application Lifecycle:**
- **`rdp apps list`** - List all applications with filtering
  - Flags: `-p/--project` (filter by Argo CD project), `-j/--json` (machine-readable output)
  - Table format: NAME, PROJECT, SYNC, HEALTH, REPO, PATH, AGE, LAST DEPLOYED
  - Status icons: ✓ (Synced+Healthy), ⚠ (OutOfSync/Progressing), ✗ (Degraded/Unknown)
  - Age formatting: human-readable (2d, 5h, 30m) using shared `formatAge()` helper
  - Empty state: "No applications found."

- **`rdp apps status <name>`** - Detailed application inspection
  - Flag: `-j/--json` (machine-readable output)
  - Unicode box format showing:
    - Application Info: Name, namespace, project, age
    - Source: Repo URL, path, target revision, current revision
    - Sync Status: Status icon + message, compared-to, last sync time
    - Health Status: Status icon + message
    - Resources: Table of first 10 managed resources (kind, namespace, name, status, health)
    - Recent History: Last 5 deployments with revision + timestamp
    - Conditions: Warning/error conditions with messages
  - 404 handling: "Application 'name' not found"
  - Truncation: Long messages/URLs intelligently shortened with "..." suffix

- **`rdp apps sync <name>`** - Trigger async sync operations
  - Flags: `--prune` (delete resources not in Git), `--dry-run` (preview without applying), `--revision <rev>` (sync specific commit)
  - Async operation: Returns immediately with operation phase, guides user to check progress with `rdp apps status`
  - Confirms sync initiated with clear summary (Operation: Sync, Phase: Running, Prune: false, Dry Run: false)

**Critical Type Safety:**
- **All CLI types match API JSON tags exactly**:
  - `ListAppsResponse.applications` (NOT `apps`) ✅
  - `ApplicationSummary.lastDeployed` (NOT `lastSyncedAt`) ✅
  - `Application.status.sync.status` (Synced/OutOfSync/Unknown) ✅
  - `Application.status.health.status` (Healthy/Progressing/Degraded/Suspended/Missing) ✅
- **Root Cause Prevention** - Verified against `api/internal/argocd/types.go` before implementation
- **Build Validation** - `go build` passes with no compilation errors

**Consistent Patterns:**
- **Parent Command Structure** - `appsCmd` with no `RunE` (container only), three subcommands
- **Config Validation** - `ValidateConfig()` before all API calls (matches `infra.go` pattern)
- **HTTP Timeouts** - 15s (list/status), 30s (sync operations)
- **Error Handling** - Structured error wrapping (`fmt.Errorf("context: %w", err)`), HTTP body capture on failure
- **Formatting Helpers**:
  - `formatSyncIcon()` - Status-aware icons (✓/⚠/✗)
  - `formatSyncStatus()` - Prepends icon to status text
  - `formatHealthIcon()` / `formatHealthStatus()` - Health-aware icons and text
  - `truncateString()` - Smart truncation with ellipsis
  - `formatAge()` - Reused from `infra.go` for consistency

**Type Conflict Resolution:**
- **Fixed:** `status.go` type collisions with `apps.go`
  - `HealthStatus` → `APIHealthStatus` ✅
  - `ApplicationStatus` → `ApplicationsStatus` ✅
  - `formatHealthIcon()` → `formatAPIHealthIcon()` ✅
- **Fixed:** `status.go` API response structure to match actual API:
  - `{ apps: [] }` → `{ applications: [] }` ✅
  - `app.health` → `app.healthStatus` ✅

**Integration Points:**
- **Platform API Endpoints:**
  - `GET /api/v1/apps` - List applications (returns `ListAppsResponse`)
  - `GET /api/v1/apps/{name}` - Get application details (returns `Application`)
  - `POST /api/v1/apps/{name}/sync` - Trigger sync (accepts `SyncRequest`, returns `SyncResponse`)
- **Argo CD Backend** - CLI → Platform API → Argo CD API (service account + RBAC configured via GitOps)

**Files Added:**
- `cli/cmd/apps.go` - 700 lines (NEW) - Complete implementation with all subcommands

**Files Modified:**
- `cli/cmd/status.go` - Fixed type collisions (renamed types to avoid `apps.go` conflicts)
- `cli/README.md` - Added Application Management section with examples
- `homelab-platform/CLAUDE.md` - Updated `cli/` status line
- `homelab-platform/README.md` - Updated `cli/` status line

**CLI Progress:**
- ✅ Root command + config management (#65)
- ✅ Version command
- ✅ `rdp status` - Platform health summary (#66)
- ✅ `rdp infra list/status` - Infrastructure Claims (#68)
- ✅ `rdp apps list/status/sync` - Application management (#67) **← NEW**
- ⬜ `rdp infra create/delete` - Interactive infra ops (#69-#71)
- ⬜ `rdp compliance` - Policy violations, CVEs, events (#73)
- ⬜ `rdp secrets` - Secret management (#74)
- ⬜ `rdp scaffold create` - Interactive project creation (#72)
- ⬜ `rdp investigate` - HolmesGPT integration (#75)
- ⬜ `rdp ask` - kagent natural language (#76)

**Next:** Interactive infrastructure operations with bubbletea (#69-#71).

---

### Added - Portal UI Security Events Panel (2026-02-23)

**Portal UI Enhancement** - Completed task #84: Dashboard panel 6 of 6 — **CORE DASHBOARD COMPLETE**

**Features:**

**Runtime Security Visibility:**
- **Falco Integration** - Displays real-time security events from Falco via Platform API EventStore
- **Timeline Table Layout** - 5 columns: Timestamp (formatted), Severity badge, Rule name, Resource (namespace/pod), Message (truncated)
- **Four-tier Severity Color-coding:**
  - Red (danger): Critical/Alert/Emergency
  - Yellow (warning): Error
  - Blue (info): Warning/Notice
  - Gray (default): Debug/Informational
- **Timestamp Formatting** - RFC3339 → human-readable ("Feb 23, 2:30 PM")
- **Message Truncation** - 100-character limit with full message on hover tooltip
- **Event Summary Footer** - Shows total event count with "most recent first" indicator
- **Auto-refresh** - 30-second polling via TanStack Query
- **Empty State** - Positive message: "✓ No security events detected" + "Falco is monitoring runtime activity across all namespaces"

**Critical Type Fix:**
- **TypeScript Alignment** - Fixed 5 incorrect fields in `SecurityEvent` interface:
  - `priority` → `severity` ✅ (matches json:"severity")
  - `source` → `resource` ✅ (matches json:"resource,omitempty")
  - Removed speculative fields: `tags`, `output`, `outputFields`, `hostname` ✅
- **Response Structure Fix** - Removed non-existent `count` field from `ListSecurityEventsResponse`
- **Root Cause Prevention** - Verified against `api/internal/compliance/types.go` JSON tags (lines 64-71)
- **Build Validation** - `npm run build` passes with no TypeScript errors

**Technical Implementation:**
- **Pattern Consistency** - Follows VulnerabilityFeedPanel structure (most recent reference implementation)
- **Helper Functions:**
  - `formatTimestamp()` - Converts ISO 8601 to locale string with month/day/time
  - `truncateMessage()` - Limits output to 100 chars with ellipsis
  - `getSeverityVariant()` - Maps Falco priorities to Badge color variants
- **Resource Display** - Monospace font for `namespace/pod` format, shows "-" if not K8s event

**Integration Architecture:**
```
Falco (DaemonSet, wave 8)
  → HTTP output
  → Falcosidekick (Deployment, wave 9)
  → Platform API Webhook (POST /api/v1/webhooks/falco)
  → EventStore (in-memory, 1000 events)
  → Portal UI (GET /api/v1/compliance/events, 30s polling)
  → SecurityEventsPanel (Dashboard display)
```

**Files Added:**
- `portal/src/components/dashboard/SecurityEventsPanel.tsx` - 136 lines (NEW)

**Files Modified:**
- `portal/src/api/types.ts` - Fixed `SecurityEvent` + `ListSecurityEventsResponse` interfaces to match Go API
- `portal/src/pages/Dashboard.tsx` - Integrated SecurityEventsPanel into dashboard grid (replaces placeholder comment)

**Dashboard Progress:** ✅ **6 of 6 panels complete** — Core compliance monitoring dashboard finished!

**Compliance Monitoring Triad Complete:**
1. **Policy Violations** (static audit) — Gatekeeper admission policy failures
2. **Vulnerability Feed** (image scanning) — Trivy CVE detection
3. **Security Events** (runtime threats) — Falco behavioral monitoring

**Next:** Scaffold form (#85), AI Ops panel (#86), detail pages.

---

### Added - Portal UI Vulnerability Feed Panel (2026-02-23)

**Portal UI Enhancement** - Completed task #83: Dashboard panel 5 of 6

**Features:**

**CVE Visibility Dashboard:**
- **Trivy Integration** - Displays vulnerability scan results from Trivy Operator VulnerabilityReport CRDs
- **Scrollable Table** - 5 columns: Severity badge, CVE ID (clickable link), Image name (truncated), Package, Fixed version
- **Color-coded Severity** - Red (CRITICAL/HIGH), Yellow (MEDIUM/LOW), Gray (UNKNOWN)
- **Smart Image Truncation** - Preserves registry + tag, shortens middle path, full path on hover tooltip
- **Summary Footer** - Shows CVE count across unique images (e.g., "15 CVEs found across 3 images")
- **Auto-refresh** - 30-second polling via TanStack Query

**Critical Type Fix:**
- **TypeScript Alignment** - Fixed 6 incorrect field names in `Vulnerability` interface:
  - `vulnerabilityID` → `cveId` ✅
  - `resource` → `image` ✅
  - `package` → `affectedPackage` ✅
  - Added missing `workload` field ✅
  - Removed speculative fields (`installedVersion`, `title`, `publishedDate`) ✅
- **Root Cause Prevention** - Verified against `api/internal/compliance/types.go` JSON tags before implementation
- **Build Validation** - `npm run build` passes with no TypeScript errors

**Technical Implementation:**
- **Pattern Consistency** - Follows PolicyViolationsPanel table structure for maintainability
- **External Links** - CVE IDs link to NVD/vendor advisories if `primaryLink` exists
- **Empty State** - Positive message: "✓ No vulnerabilities detected" + "All scanned images are free of known CVEs"
- **Loading State** - Spinner with "Scanning container images for CVEs..." message

**Files Added:**
- `portal/src/components/dashboard/VulnerabilityFeedPanel.tsx` - 157 lines (NEW)

**Files Modified:**
- `portal/src/api/types.ts` - Fixed `Vulnerability` interface to match Go API struct
- `portal/src/pages/Dashboard.tsx` - Integrated VulnerabilityFeedPanel into dashboard grid

**Dashboard Progress:** 5 of 6 panels complete. Remaining: Security Events (#84).

---

### Added - Platform API Secrets Endpoint (2026-02-23)

**Platform API Enhancement** - Completed task #50: GET /api/v1/secrets/:namespace endpoint

**Features:**

**Unified Secrets View:**
- **ExternalSecrets** - Lists ESO CRDs with status (Ready/Error), keys, and sync messages
- **Connection Secrets** - Lists Crossplane-generated secrets with source Claim references
- **Security-first** - Exposes metadata only (names, keys, status) — never secret values
- **Graceful degradation** - If ESO not installed, continues with core Secrets only

**Response Structure:**
```json
{
  "secrets": [
    {
      "name": "platform-api-secrets",
      "namespace": "platform",
      "kind": "ExternalSecret",
      "status": "Ready",
      "message": "secret synced",
      "creationTimestamp": "2026-02-20T02:09:22Z",
      "keys": ["ARGOCD_TOKEN", "GITHUB_TOKEN", "OPENAI_API_KEY"],
      "labels": {...}
    }
  ],
  "total": 1
}
```

**Technical Implementation:**
- **Three-layer pattern:** types.go (DTOs) → client.go (K8s queries) → handler.go (HTTP)
- **Dual API access:** dynamic client for ExternalSecrets CRDs + typed client for core Secrets
- **Connection secret linking:** Parses `crossplane.io/claim-name` labels to link back to source Claims
- **Sorted output:** ExternalSecrets first, then alphabetical by name

**Files Added:**
- `api/internal/secrets/types.go` - Response DTOs (SecretSummary, ListSecretsResponse)
- `api/internal/secrets/client.go` - Kubernetes client wrapper (ListExternalSecrets, ListConnectionSecrets)
- `api/internal/secrets/handler.go` - HTTP handler (HandleListSecrets)

**Files Modified:**
- `api/main.go` - Handler initialization and route wiring
- `platform/platform-api/base/deployment.yaml` - Image version v0.1.6 → v0.1.7
- `platform/platform-api/kustomization.yaml` - Image tag v0.1.6 → v0.1.7

**Deployment:**
- Platform API v0.1.7 deployed via GitOps (Argo CD + Kustomize)
- Endpoint tested and verified: `/api/v1/secrets/platform`

**Next Steps:**
- Task #74: `rdp secrets list/create` CLI command
- Future: Portal UI secrets management panel

---

### Added - Grafana Dashboards for Compliance and Infrastructure (2026-02-22)

**Monitoring Stack Enhancement** - Completed task #37: Platform Grafana dashboards

**Features:**

**Dashboard 1: Platform Compliance Overview** (UID: `platform-compliance`)
- **Compliance Score Gauge** - Real-time score (0-100) with color-coded thresholds:
  - 🔴 Red: <70 (critical issues)
  - 🟠 Amber: 70-89 (needs attention)
  - 🟢 Green: ≥90 (healthy)
- **7 Visualization Panels:**
  - Compliance score trend over time (time series)
  - Policy violations by type (pie chart - Gatekeeper constraints)
  - CVE count by severity (stacked bars: Critical/High/Medium/Low)
  - Falco security events counter (5-minute window)
  - Policy violations by namespace (bar chart)
  - Top 10 vulnerable images (sortable table with severity color-coding)
- **Metrics:** `gatekeeper_violations`, `vulnerabilityreport_vulnerability_count`, `falcosidekick_inputs_total`
- **Auto-refresh:** 30 seconds | **Default range:** 6 hours

**Dashboard 2: Crossplane Claim Status** (UID: `crossplane-status`)
- **Infrastructure Health Stats:**
  - Total/Ready/Synced/Not Ready Claims
  - Ready percentage
  - Reconcile error rate (5-minute)
- **11 Visualization Panels:**
  - Claim status trends (Ready/Not Ready/Synced over time)
  - Reconcile success vs error rates
  - All Claims table (Name/Namespace/Kind)
  - Distribution by Type and Namespace (pie charts)
- **Metrics:** `crossplane_managed_resource_exists/ready/synced`, `crossplane_managed_resource_reconcile_total`
- **Auto-refresh:** 30 seconds | **Default range:** 6 hours

**Technical Implementation:**
- **Pattern:** Grafana sidecar auto-discovery (ConfigMaps with label `grafana_dashboard: "1"`)
- **GitOps:** Dashboards sync via Argo CD from Git (`dashboards/configmap-*.yaml`)
- **No manual import required** - Sidecar loads dashboards automatically on ConfigMap creation

**Files Added:**
- `platform/monitoring/dashboards/compliance-overview.json` - 688 lines of Grafana JSON
- `platform/monitoring/dashboards/configmap-compliance.yaml` - ConfigMap wrapper
- `platform/monitoring/dashboards/crossplane-status.json` - Dashboard JSON
- `platform/monitoring/dashboards/configmap-crossplane.yaml` - ConfigMap wrapper
- `platform/monitoring/dashboards/README.md` - Dashboard docs + troubleshooting guide

**Files Modified:**
- `platform/monitoring/application.yaml` - Added dashboard ConfigMaps to Argo CD sync (exclude pattern)
- `platform/monitoring/README.md` - Updated Grafana section with dashboard UIDs

**Access:**
- Production: `http://grafana.rdp.azurelaboratory.com/`
- Port-forward: `kubectl port-forward -n monitoring svc/monitoring-grafana 3000:80`
- Credentials: From ExternalSecret `grafana-admin-creds`

**Commits:**
- `80aa2a9` feat(monitoring): add Grafana dashboards for compliance and infrastructure
- `6601d61` fix(monitoring): prevent Kustomize detection in Argo CD directory source

---

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
