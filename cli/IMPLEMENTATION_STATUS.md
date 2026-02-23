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
**Status:** List and status commands complete
**Files:** `cli/cmd/infra.go`
**Tasks:** #68 (complete), #69-#71 (pending)

Subcommands:
- ✅ `rdp infra list [storage|vaults]` — List all Claims (tabular view with filters)
  - Flags: `--namespace` (filter), `--json` (output format)
  - Table: NAME, NAMESPACE, KIND, STATUS, READY, SYNCED, AGE, CONNECTION SECRET
  - Status icons: ✓ (ready+synced), ⚠ (issues)
- ✅ `rdp infra status <kind> <name>` — Get Claim details + resource tree
  - Flag: `--namespace` (default: default), `--json` (output format)
  - Unicode box format: Claim details, Composite resource, Managed Azure resources, Recent K8s events
  - Supports: `storage` (StorageBucket), `vault` (Vault)
- ⬜ `rdp infra create storage` — Create StorageBucket Claim (bubbletea interactive)
- ⬜ `rdp infra create vault` — Create Vault Claim (bubbletea interactive)
- ⬜ `rdp infra delete <kind> <name>` — Delete Claim (commits removal to app repo)

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

## Pending Commands

### ⬜ `rdp scaffold`
**Task:** #72
**Dependencies:** Platform API `/api/v1/scaffold` endpoint (#51)

Subcommands:
- `rdp scaffold create` — Create new service from template (bubbletea interactive)

### ⬜ `rdp compliance`
**Task:** #73
**Dependencies:** Platform API `/api/v1/compliance/*` endpoints (#48)

Subcommands:
- `rdp compliance summary` — Compliance overview
- `rdp compliance policies` — List Gatekeeper policies
- `rdp compliance violations` — List policy violations
- `rdp compliance vulns` — List Trivy CVEs
- `rdp compliance events` — List Falco security events

### ⬜ `rdp secrets`
**Task:** #74
**Dependencies:** Platform API `/api/v1/secrets` endpoint (#50)

Subcommands:
- `rdp secrets list <namespace>` — List ExternalSecrets + connection secrets
- `rdp secrets create` — Create ExternalSecret

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

### ⬜ `rdp portal`
**Task:** #77
**Dependencies:** Portal UI (#78)

Subcommands:
- `rdp portal open` — Open Portal UI in browser

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
| `rdp infra` | 🔨 Partial | #68 | List/status complete, create/delete pending (#69-#71) |
| `rdp apps` | ✅ Complete | #67 | List/status/sync all working |
| `rdp scaffold` | ⬜ Pending | - | Interactive project creation (#72) |
| `rdp compliance` | ⬜ Pending | - | Policy/CVE/event commands (#73) |
| `rdp secrets` | ⬜ Pending | - | Secret management (#74) |
| `rdp investigate` | ⬜ Pending | - | HolmesGPT integration (#75) |
| `rdp ask` | ⬜ Pending | - | kagent natural language (#76) |
| `rdp portal` | ⬜ Pending | - | Browser launcher (#77) |

## Known Issues

1. ~~**Task #89**: Platform API `/api/v1/apps` endpoint needs Argo CD token configuration~~ **RESOLVED**
   - ~~Impact: `rdp status` shows error for Applications section~~
   - Fixed: Argo CD token bootstrap script created, RBAC via GitOps, integration working
