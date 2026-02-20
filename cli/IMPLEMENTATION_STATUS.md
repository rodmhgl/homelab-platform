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

## Pending Commands

### ⬜ `rdp apps`
**Task:** #67
**Dependencies:** Platform API `/api/v1/apps` endpoints (#42, #43), Task #89 fix

Subcommands:
- `rdp apps list` — List all Argo CD applications
- `rdp apps status <name>` — Get application details
- `rdp apps sync <name>` — Trigger application sync
- `rdp apps logs <name>` — View application logs

### ⬜ `rdp infra`
**Tasks:** #68, #69, #70, #71
**Dependencies:** Platform API `/api/v1/infra` endpoints (#44, #45, #46, #47)

Subcommands:
- `rdp infra list` — List all Claims (tabular view)
- `rdp infra status <kind> <name>` — Get Claim details + resource tree
- `rdp infra create storage` — Create StorageBucket Claim (bubbletea interactive)
- `rdp infra create vault` — Create Vault Claim (bubbletea interactive)
- `rdp infra delete <kind> <name>` — Delete Claim (commits removal to app repo)

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

## Known Issues

1. **Task #89**: Platform API `/api/v1/apps` endpoint needs Argo CD token configuration
   - Impact: `rdp status` shows error for Applications section
   - Workaround: All other sections display correctly with graceful degradation
