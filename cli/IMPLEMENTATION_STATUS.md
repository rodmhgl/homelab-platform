# CLI Implementation Status

## Completed Commands

### ✅ `rdp config`
**Status:** Complete
**Files:** `cli/cmd/config.go`

Subcommands:
- `rdp config init` — Initialize `~/.rdp/config.yaml` (interactive or via flags)
- `rdp config view` — Display current configuration with masked token
- `rdp config set <key> <value>` — Set individual configuration values

Configuration precedence: flags > environment variables > config file

**Environment Variables:**
- `RDP_API_BASE_URL`
- `RDP_AUTH_TOKEN`

### ✅ `rdp version`
**Status:** Complete
**Files:** `cli/cmd/version.go`

Displays version information (set via ldflags during build):
- Version
- Git commit
- Build date

### ✅ `rdp status`
**Status:** Complete (with known issue)
**Files:** `cli/cmd/status.go`, `cli/main.go` (error handling fix)
**Task:** #66

Aggregates platform health from multiple Platform API endpoints:

| Endpoint | Data Displayed |
|----------|----------------|
| `/health` | API health check |
| `/ready` | API readiness |
| `/api/v1/compliance/summary` | Compliance score (0-100), policies count, violations, CVEs |
| `/api/v1/apps` | Application count, healthy vs degraded |
| `/api/v1/infra` | Total Claims, StorageBucket count, Vault count |

**Features:**
- Graceful degradation: Shows available data even when individual endpoints fail
- Professional Unicode box-drawing UI
- Status icons: ✓ (good), ✗ (error), ⚠ (warning)
- Compliance score thresholds: ≥90 (✓), 70-89 (⚠), <70 (✗)
- Overall status: Platform operational vs has issues

**Known Issue:**
- Applications section shows "HTTP 500: failed to list applications" due to Argo CD API configuration
- Tracked in Task #89 (Platform API side)
- Does not block overall command functionality

**Example Output:**
```
╔═══════════════════════════════════════════════════════════╗
║         RNLabs Developer Platform Status                 ║
╚═══════════════════════════════════════════════════════════╝

┌─ Platform API ────────────────────────────────────────────┐
│ Health:      ✓ OK
│ Ready:       ✓ OK
└───────────────────────────────────────────────────────────┘

┌─ Compliance ──────────────────────────────────────────────┐
│ Score:       ✓ 100/100
│ Policies:    0 active
│ Violations:  0
│ CVEs:        0
└───────────────────────────────────────────────────────────┘

┌─ Applications ────────────────────────────────────────────┐
│ Status:      ✗ ERROR
│ Error:       HTTP 500: {"error":"failed to list applications"}
└───────────────────────────────────────────────────────────┘

┌─ Infrastructure ──────────────────────────────────────────┐
│ Total Claims: 1
│   Storage:    1
│   Vaults:     0
└───────────────────────────────────────────────────────────┘

Overall Status: ✓ Platform is operational
```

### ✅ `rdp infra`
**Status:** Complete (CRUD lifecycle)
**Files:** `cli/cmd/infra.go`, `cli/cmd/infra_create.go`, `cli/cmd/infra_delete.go`, `cli/internal/tui/*.go`
**Tasks:** #68, #69, #70, #71 (all complete)

Subcommands:
- ✅ `rdp infra list [storage|vaults]` — List all Claims (tabular view with filters)
  - Flags: `--namespace` (filter), `--json` (output format)
  - Table: NAME, NAMESPACE, KIND, STATUS, READY, SYNCED, AGE, CONNECTION SECRET
  - Status icons: ✓ (ready+synced), ⚠ (issues)
- ✅ `rdp infra status <kind> <name>` — Get Claim details + resource tree
  - Flag: `--namespace` (default: default), `--json` (output format)
  - Unicode box format: Claim details, Composite resource, Managed Azure resources, Recent K8s events
  - Supports: `storage` (StorageBucket), `vault` (Vault)
- ✅ `rdp infra create storage` — Create StorageBucket Claim (bubbletea interactive TUI)
  - Sequential field entry: name, namespace, location, tier, redundancy, versioning
  - DNS label validation, location whitelist enforcement
  - Git repository auto-detection (SSH/HTTPS URL parsing)
  - Commits Claim YAML to app repo via Platform API
- ✅ `rdp infra create vault` — Create Vault Claim (bubbletea interactive TUI)
  - Sequential field entry: name, namespace, location, SKU, retention days
  - DNS label validation, retention range (7-90 days)
  - Git repository auto-detection (SSH/HTTPS URL parsing)
  - Commits Claim YAML to app repo via Platform API
- ✅ `rdp infra delete <kind> <name>` — Delete Claim via GitOps
  - Flags: `--repo-owner` (required), `--repo-name` (required), `--namespace` (default: default), `--force` (skip confirmation), `--json` (output format)
  - Safety confirmation: User must type Claim name to confirm (unless --force)
  - Removes `k8s/claims/<name>.yaml` from Git → Argo CD syncs → Crossplane deletes Azure resources
  - Supports: `storage` (StorageBucket), `vault` (Vault)

**Example Output (list):**
```
NAME              NAMESPACE  KIND           STATUS        READY  SYNCED  AGE  CONNECTION SECRET
----              ---------  ----           ------        -----  ------  ---  -----------------
demo-storage      default    StorageBucket  ✓ Available   ✓      ✓       2d   demo-storage-conn

Total: 1 Claims
```

### ✅ `rdp apps`
**Status:** Complete
**Files:** `cli/cmd/apps.go`
**Task:** #67

Subcommands:
- ✅ `rdp apps list` — List all Argo CD applications
  - Flags: `--project` (filter), `--json` (output format)
  - Table: NAME, PROJECT, SYNC, HEALTH, REPO, PATH, AGE, LAST DEPLOYED
  - Status icons: ✓ (Synced+Healthy), ⚠ (OutOfSync/Progressing), ✗ (Degraded/Unknown)
- ✅ `rdp apps status <name>` — Get application details
  - Flag: `--json` (output format)
  - Unicode box format: App info, Source, Sync status, Health status, Resources (first 10), History (last 5), Conditions
  - 404 handling with clear error message
- ✅ `rdp apps sync <name>` — Trigger application sync
  - Flags: `--prune`, `--dry-run`, `--revision <rev>`
  - Async operation: Returns immediately with phase, guides user to check progress

**Example Output (list):**
```
NAME         PROJECT   SYNC         HEALTH       REPO                     PATH         AGE  LAST DEPLOYED
----         -------   ----         ------       ----                     ----         ---  -------------
platform-api platform  ✓ Synced     ✓ Healthy    github.com/org/platform platform/    2d   2024-02-21 14:32
argocd       platform  ✓ Synced     ✓ Healthy    github.com/org/platform argocd/      5d   2024-02-18 09:15

Total: 2 applications
```

**Critical Implementation Details:**
- **Type Safety:** All types match API JSON tags exactly (`applications` not `apps`, `lastDeployed` not `lastSyncedAt`)
- **Consistent Patterns:** Follows `infra.go` formatting (unicode boxes, status icons, age helpers)
- **Error Handling:** 404 detection, HTTP body capture, graceful degradation
- **HTTP Timeouts:** 15s (list/status), 30s (sync operations)

### ✅ `rdp scaffold`
**Status:** Complete
**Files:** `cli/cmd/scaffold.go`, `cli/internal/tui/create_scaffold.go`
**Task:** #72

Subcommands:
- ✅ `rdp scaffold create` — Create new service from template (bubbletea interactive TUI)
  - Sequential field entry: template selection, project name, description (optional), HTTP port, gRPC enable, gRPC port (if enabled), database enable, storage enable, Key Vault enable, GitHub org, GitHub repo
  - DNS label validation for project name
  - Port validation (1024-65535 range), conflict detection (HTTP ≠ gRPC)
  - Git repository auto-detection (SSH/HTTPS URL parsing)
  - Extended timeout (60s) for Copier + GitHub + Argo CD operations
  - Executes Platform API `/api/v1/scaffold` endpoint
  - Success message includes repo URL, Argo CD app name, platform config path, next steps

**Example Output (success):**
```
✓ Service Scaffolded Successfully!

Argo CD will sync this application within 60 seconds.

Details:
  Repository:        https://github.com/rodmhgl/my-api
  Argo CD App:       my-api
  Platform Config:   apps/my-api/config.json

Next Steps:
  1. Clone repository: git clone https://github.com/rodmhgl/my-api
  2. Build service:    cd my-api && make build
  3. Run tests:        make test
  4. Verify Argo CD:   rdp apps status my-api
```

**Critical Implementation Details:**
- **Type Safety:** ScaffoldRequest/ScaffoldResponse match API JSON tags exactly (`enable_keyvault` not `enableKeyVault`)
- **Conditional Flow:** Skips gRPC port prompt if gRPC disabled
- **Default Values:** HTTP port 8080, gRPC port 9090 (applied if fields empty)
- **GoModulePath:** Auto-constructed as `github.com/{org}/{repo}`
- **RepoPrivate:** Always true (private repos by default)
- **Progressive Disclosure:** Completed fields shown with checkmarks as user progresses through TUI states

## Pending Commands

### ✅ `rdp compliance`
**Status:** Complete
**Files:** `cli/cmd/compliance.go`, `cli/cmd/compliance_summary.go`, `cli/cmd/compliance_policies.go`, `cli/cmd/compliance_violations.go`, `cli/cmd/compliance_vulns.go`, `cli/cmd/compliance_events.go`
**Task:** #73

Subcommands:
- ✅ `rdp compliance summary` — View overall compliance score and metrics
  - Flag: `--json` (output format)
  - Displays: Compliance score (0-100), policy violations count, vulnerabilities by severity (Critical/High breakdown)
  - Color-coded score: ✓ (≥90 green), ⚠ (70-89 amber), ✗ (<70 red)
- ✅ `rdp compliance policies` — List active Gatekeeper ConstraintTemplates
  - Flag: `--json` (output format)
  - Table: NAME, KIND, SCOPE, DESCRIPTION
  - Shows cluster-wide vs namespaced policies
- ✅ `rdp compliance violations` — View policy violations
  - Flags: `--namespace` (filter), `--json` (output format)
  - Table: CONSTRAINT, KIND, RESOURCE, NAMESPACE, MESSAGE
  - Shows Gatekeeper audit violations with detailed context
- ✅ `rdp compliance vulns` — List vulnerabilities from Trivy scans
  - Flags: `--severity` (CRITICAL|HIGH|MEDIUM|LOW filter), `--json` (output format)
  - Table: SEVERITY, CVE-ID, IMAGE, PACKAGE, FIXED, WORKLOAD
  - Color-coded severity: CRITICAL/HIGH (red), MEDIUM (yellow), LOW (gray)
  - Summary footer with severity breakdown
- ✅ `rdp compliance events` — View Falco security events
  - Flags: `--namespace` (filter), `--severity` (ERROR|WARNING|NOTICE filter), `--since` (time window like "1h", "30m"), `--limit` (max events, default 50), `--json` (output format)
  - Table: TIME, SEVERITY, RULE, RESOURCE, MESSAGE
  - Color-coded severity: ERROR (red), WARNING (yellow), NOTICE (white)
  - Summary footer with severity breakdown

**Example Output (summary):**
```
┌─ Compliance Summary ──────────────────────────────────────┐
│                                                            │
│  Compliance Score:  ✓ 92                                  │
│                                                            │
│  Policy Violations: 3                                      │
│  Vulnerabilities:   12 (2 Critical, 5 High)                │
│                                                            │
└────────────────────────────────────────────────────────────┘
```

**Critical Implementation Details:**
- **Type Safety:** All types match API JSON tags exactly (`complianceScore` is `float64`, `vulnerabilitiesBySeverity` is `map[string]int`, `severity` not `priority`)
- **Consistent Patterns:** Follows `apps.go`/`infra.go` formatting (tabwriter, color codes, JSON output)
- **Query Parameters:** Uses `url.Values` for clean parameter encoding (namespace, severity, since, limit filters)
- **Validation:** Severity enums validated before API call, duration parsing for `--since` flag
- **Error Handling:** HTTP status codes, body capture, graceful empty state messages

### ✅ `rdp secrets`
**Status:** Complete
**Files:** `cli/cmd/secrets.go`, `cli/cmd/secrets_list.go`
**Task:** #74

Subcommands:
- ✅ `rdp secrets list <namespace>` — List ExternalSecrets + connection secrets
  - Flags: `--kind` (filter: "external" | "connection" | ""), `--json` (output format)
  - Table: NAME, NAMESPACE, KIND, STATUS, KEYS, SOURCE CLAIM, AGE
  - Status icons: ✓ (Ready/Synced green), ✗ (Error red), ○ (Unknown gray)
  - Keys display: Comma-separated for ≤3 keys, "N keys" summary for >3
  - Source Claim: Shows originating Crossplane Claim for connection secrets
  - Summary footer: Total count with breakdown by type (ExternalSecrets vs connection secrets)
- ⬜ `rdp secrets create` — Create ExternalSecret (future enhancement)

**Example Output (list):**
```
NAME                NAMESPACE  KIND            STATUS        KEYS                    SOURCE CLAIM              AGE
----                ---------  ----            ------        ----                    ------------              ---
demo-storage-conn   default    Secret          ✓ Synced      3 keys                  demo-storage (StorageBucket)  3d
github-pat          platform   ExternalSecret  ✓ Ready       token                   -                         8d
api-creds           default    ExternalSecret  ✗ Error       -                       -                         1h

Total: 3 secrets (2 ExternalSecrets, 1 connection secret)
```

**Critical Implementation Details:**
- **Type Safety:** All types match API JSON tags exactly (`creationTimestamp` is `time.Time`, `sourceClaim` is `*ResourceRef`)
- **Client-side Filtering:** `--kind` flag filters response array (API returns all secrets)
- **Consistent Patterns:** Follows `compliance_violations.go` table formatting, `infra.go` age/status helpers
- **Error Handling:** 400 (bad request), 404 (namespace not found), 500 (API error) with clear messages
- **HTTP Timeout:** 15s (list operations)

### ⬜ `rdp investigate`
**Task:** #75
**Dependencies:** Platform API `/api/v1/investigate` endpoints (#52), HolmesGPT (#39)

Subcommands:
- `rdp investigate <app> --issue <description>` — Trigger HolmesGPT investigation

### ⬜ `rdp ask`
**Task:** #76
**Dependencies:** Platform API `/api/v1/agent/ask` endpoint (#53), kagent (#38)

Usage:
- `rdp ask <natural language question>` — Stream response from kagent

### ✅ `rdp portal`
**Status:** Complete
**Files:** `cli/cmd/portal.go`, `cli/cmd/portal_open.go`
**Task:** #77

Subcommands:
- ✅ `rdp portal open` — Open Portal UI in default browser
  - Flags: `--url` (override Portal URL), `--print` (print URL only, don't open browser)
  - URL precedence: 1) --url flag → 2) portal_url config → 3) derived from api_base_url → 4) default (http://portal.rdp.azurelaboratory.com)
  - Cross-platform: Linux/WSL (xdg-open), macOS (open), Windows (cmd /c start)
  - Graceful fallback: Prints URL with instructions if browser launch fails

**Example Output:**
```
Opening Portal UI in browser: http://portal.rdp.azurelaboratory.com
```

**Critical Implementation Details:**
- **URL Derivation Patterns:** Handles multiple API URL formats: `api.domain.com`, `api-domain.com`, `platform-api.domain.com`, `platform.domain.com`
- **Platform Detection:** Uses `runtime.GOOS` to select appropriate browser launcher
- **No Auth Required:** Pure client-side operation, no API dependency
- **Config Schema:** Added optional `portal_url` to `Config` struct (backward compatible)

## Architecture

**Language:** Go
**Frameworks:** Cobra (commands), Viper (configuration)
**Pattern:** Thin client over Platform API (stateless HTTP calls)

All operations go through the Platform API — the CLI maintains no state beyond configuration.

## Configuration

**Config file:** `~/.rdp/config.yaml`

```yaml
api_base_url: http://localhost:8080
auth_token: <your-token>
```

**Precedence:** Command-line flags > Environment variables > Config file

## Build

```bash
cd cli
go build -o rdp .
```

**With version info:**
```bash
go build -ldflags "\
  -X github.com/rodmhgl/homelab-platform/cli/cmd.Version=1.0.0 \
  -X github.com/rodmhgl/homelab-platform/cli/cmd.GitCommit=$(git rev-parse HEAD) \
  -X github.com/rodmhgl/homelab-platform/cli/cmd.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o rdp .
```

## Implementation Progress

| Command Group | Status | Tasks Complete | Notes |
|---------------|--------|----------------|-------|
| `rdp config` | ✅ Complete | #65 | Config management (init/view/set) |
| `rdp version` | ✅ Complete | - | Build metadata display |
| `rdp status` | ✅ Complete | #66 | Platform health aggregation |
| `rdp infra` | ✅ Complete | #68, #69, #70, #71 | Full CRUD lifecycle (list/status/create/delete) |
| `rdp apps` | ✅ Complete | #67 | List/status/sync all working |
| `rdp scaffold` | ✅ Complete | #72 | Interactive project creation (template selection, config, features, GitHub) |
| `rdp compliance` | ✅ Complete | #73 | Summary/policies/violations/vulns/events all working |
| `rdp secrets` | ✅ Complete | #74 | List command complete; create pending |
| `rdp investigate` | ⬜ Pending | - | HolmesGPT integration (#75) |
| `rdp ask` | ⬜ Pending | - | kagent natural language (#76) |
| `rdp portal` | ✅ Complete | #77 | Browser launcher complete |

## Known Issues

1. ~~**Task #89**: Platform API `/api/v1/apps` endpoint needs Argo CD token configuration~~ **RESOLVED**
   - ~~Impact: `rdp status` shows error for Applications section~~
   - Fixed: Argo CD token bootstrap script created, RBAC via GitOps, integration working
