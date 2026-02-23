# rdp CLI

The `rdp` (RNLabs Developer Platform) CLI is the primary interface for developers to interact with the Internal Developer Platform.

## Installation

```bash
# Build from source
cd homelab-platform/cli
go build -o rdp .

# Install to $GOPATH/bin
go install .
```

## Configuration

The CLI requires configuration to connect to the Platform API:

### Initialize Configuration

```bash
# Interactive setup
rdp config init

# Or provide values directly
rdp config init --api-url https://api.platform.rnlabs.local --token <your-token>
```

This creates `~/.rdp/config.yaml`:

```yaml
api_base_url: https://api.platform.rnlabs.local
auth_token: <your-token>
```

### Configuration Precedence

Configuration values are resolved in this order (highest to lowest priority):

1. **Command-line flags**: `--api-url`, `--token`
2. **Environment variables**: `RDP_API_BASE_URL`, `RDP_AUTH_TOKEN`
3. **Config file**: `~/.rdp/config.yaml` (or path from `--config`)

### Managing Configuration

```bash
# View current configuration
rdp config view

# Set individual values
rdp config set api_base_url https://api.platform.rnlabs.local
rdp config set auth_token <your-token>

# Use custom config file
rdp --config /path/to/config.yaml <command>
```

## Usage

### Platform Status

```bash
# Display comprehensive platform health summary
rdp status
```

Shows:
- API health and readiness
- Compliance score and violation count
- Application health status
- Infrastructure resources (Claims)

### Version Information

```bash
# Check version
rdp version
```

### Application Management

```bash
# List all Argo CD applications
rdp apps list

# Filter by project
rdp apps list -p platform
rdp apps list --project workloads

# Output as JSON
rdp apps list --json

# Show detailed application status
rdp apps status platform-api
rdp apps status argocd

# JSON output for detailed status
rdp apps status platform-api --json

# Trigger application sync
rdp apps sync platform-api

# Sync with pruning
rdp apps sync platform-api --prune

# Dry run (preview changes)
rdp apps sync platform-api --dry-run

# Sync specific revision
rdp apps sync platform-api --revision abc1234
```

Shows:
- **list**: Tabular view of all applications with name, project, sync/health status, repository, path, age, and last deployed time
- **status**: Detailed view including metadata, source info, sync/health status, managed resources, deployment history, and conditions
- **sync**: Initiates async sync operation with optional flags (prune, dry-run, revision)

### Infrastructure Management

```bash
# List all infrastructure Claims
rdp infra list

# List only StorageBucket Claims
rdp infra list storage

# List only Vault Claims
rdp infra list vaults

# Filter by namespace
rdp infra list --namespace production

# Output as JSON
rdp infra list --json

# Show detailed status for a specific Claim
rdp infra status storage my-bucket
rdp infra status vault my-vault --namespace production

# JSON output for detailed status
rdp infra status storage my-bucket --json

# Create infrastructure resources (interactive)
rdp infra create storage  # Interactive wizard for StorageBucket
rdp infra create vault    # Interactive wizard for Vault

# Delete infrastructure resources (GitOps)
rdp infra delete storage my-bucket --repo-owner myorg --repo-name myapp
rdp infra delete vault my-vault --namespace production --repo-owner myorg --repo-name myapp --force
```

Shows:
- **list**: Tabular view of all Claims with name, namespace, kind, status, ready/synced flags, age, and connection secret
- **status**: Detailed view including Claim details, Composite resource, Managed Azure resources, and recent Kubernetes events
- **create storage**: Interactive TUI wizard that guides you through creating an Azure Storage Account via Crossplane
  - Collects: name, namespace, location, tier, redundancy, versioning
  - Validates DNS labels, location whitelist, and field constraints
  - Auto-detects Git repository from current directory
  - Commits Claim YAML to app repo via Platform API
  - Argo CD syncs within 60 seconds
- **create vault**: Interactive TUI wizard that guides you through creating an Azure Key Vault via Crossplane
  - Collects: name, namespace, location, SKU, soft delete retention days
  - Validates DNS labels, retention range (7-90 days)
  - Auto-detects Git repository from current directory
  - Commits Claim YAML to app repo via Platform API
  - Argo CD syncs within 60 seconds
- **delete**: Deletes a Claim via GitOps (requires explicit repo flags for safety)
  - Confirmation prompt: User must type Claim name to confirm (unless --force)
  - Removes `k8s/claims/<name>.yaml` from Git repository
  - Argo CD syncs within 60 seconds and removes Claim from cluster
  - Crossplane deletes all managed Azure resources (ResourceGroup, StorageAccount, Key Vault, etc.)
  - ⚠️ **WARNING**: This is a destructive operation - Azure resources and data are permanently deleted

### Compliance Commands

```bash
# View overall compliance score and metrics
rdp compliance summary

# JSON output
rdp compliance summary --json

# List active Gatekeeper policies
rdp compliance policies

# View policy violations
rdp compliance violations

# Filter violations by namespace
rdp compliance violations --namespace platform

# List vulnerabilities from Trivy scans
rdp compliance vulns

# Filter by severity
rdp compliance vulns --severity CRITICAL
rdp compliance vulns --severity HIGH

# View Falco security events
rdp compliance events

# Filter events by time window and limit
rdp compliance events --since 1h --limit 20

# Filter by severity
rdp compliance events --severity ERROR

# Filter by namespace
rdp compliance events --namespace production

# JSON output for scripting
rdp compliance events --json
```

Shows:
- **summary**: Compliance score (0-100) with color coding, policy violations count, vulnerabilities breakdown (Critical/High/Medium/Low), security events count
- **policies**: Tabular view of Gatekeeper ConstraintTemplates with name, kind, scope, and description
- **violations**: Policy violations from Gatekeeper audits with constraint name, resource kind, resource path, namespace, and violation message
- **vulns**: CVE list from Trivy Operator scans with severity (color-coded), CVE ID, image, affected package, fixed version, and workload. Summary shows breakdown across CRITICAL/HIGH/MEDIUM/LOW.
- **events**: Falco runtime security events with timestamp (human-readable age), severity (color-coded), rule name, resource, and message. Supports time window filtering (`--since 1h`), severity filtering, and result limiting.

### Secrets Management

```bash
# List all secrets in a namespace
rdp secrets list default
rdp secrets list platform

# Filter by kind
rdp secrets list default --kind external      # ExternalSecrets only
rdp secrets list default --kind connection    # Connection secrets only

# JSON output
rdp secrets list platform --json
```

Shows:
- **list**: Unified view of both ExternalSecrets (ESO-managed) and Crossplane connection secrets
  - Table: NAME, NAMESPACE, KIND, STATUS, KEYS, SOURCE CLAIM, AGE
  - Status: ✓ Ready/Synced (green), ✗ Error (red), ○ Unknown (gray)
  - Keys: Displays key names (not values) - comma-separated for ≤3 keys, "N keys" for >3
  - Source Claim: Shows originating Crossplane Claim for connection secrets
  - Summary footer: Total count with breakdown (ExternalSecrets vs connection secrets)
  - ⚠️ **Security Note**: Never displays secret values, only metadata

### Scaffold a New Service

Create a new service from a template:

```bash
rdp scaffold create
```

The wizard will guide you through:
- **Template selection** — Currently supports `go-service` (Python template coming soon)
- **Project metadata** — Name (DNS label), description (optional)
- **Port configuration** — HTTP port (default: 8080), gRPC port (default: 9090, if enabled)
- **Feature flags** — gRPC, database, storage (StorageBucket Claim), Key Vault (Vault Claim)
- **GitHub configuration** — Org/owner (auto-detected from Git), repository name

**What happens next:**
1. Platform API executes Copier template
2. Creates GitHub repository (`github.com/{org}/{repo}`)
3. Pushes scaffolded code
4. Adds `apps/{name}/config.json` to platform repo
5. Argo CD auto-discovers application within 60 seconds

**Example output:**
```
✓ Service Scaffolded Successfully!

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

### Portal UI

Open the Platform Portal web interface in your default browser:

```bash
# Open production Portal
rdp portal open

# Open local dev Portal (port-forward)
rdp portal open --url http://localhost:8080

# Print URL without opening browser
rdp portal open --print
```

The Portal URL is determined in the following order of precedence:
1. `--url` flag (highest priority)
2. `portal_url` in config file (`~/.rdp/config.yaml`)
3. Derived from `api_base_url` (replaces 'api'/'platform' with 'portal')
4. Default: `http://portal.rdp.azurelaboratory.com`

**What's in the Portal:**
- Application status and health dashboard
- Infrastructure resources (Crossplane Claims)
- Compliance score with breakdown
- Policy violations table
- Vulnerability feed (CVE scanning)
- Security events timeline (Falco alerts)

### Other Commands

```bash
# Get help
rdp help
```

## Development

### Building with Version Information

```bash
# Build with version details
go build -ldflags "\
  -X github.com/rodmhgl/homelab-platform/cli/cmd.Version=0.1.0 \
  -X github.com/rodmhgl/homelab-platform/cli/cmd.GitCommit=$(git rev-parse --short HEAD) \
  -X github.com/rodmhgl/homelab-platform/cli/cmd.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o rdp .
```

### Project Structure

```
cli/
├── main.go                      # Entry point
├── cmd/
│   ├── root.go                  # Root command and config management
│   ├── config.go                # Config subcommands (init, view, set)
│   ├── version.go               # Version command
│   ├── status.go                # Platform health summary
│   ├── apps.go                  # Application commands (list, status, sync)
│   ├── infra.go                 # Infrastructure commands (list, status)
│   ├── infra_create.go          # Infrastructure create commands (storage, vault)
│   ├── infra_delete.go          # Infrastructure delete command (GitOps with confirmation)
│   ├── compliance.go            # Compliance commands (parent + helpers)
│   ├── compliance_summary.go    # Compliance summary command
│   ├── compliance_policies.go   # List policies command
│   ├── compliance_violations.go # List violations command
│   ├── compliance_vulns.go      # List vulnerabilities command
│   ├── compliance_events.go     # List security events command
│   ├── secrets.go               # Secrets commands (parent + helpers)
│   ├── secrets_list.go          # List secrets command
│   ├── scaffold.go              # Scaffold commands (create)
│   └── ...                      # Future command groups (investigate, ask, portal)
├── internal/
│   └── tui/
│       ├── shared.go            # Shared TUI styles, validators, helpers
│       ├── create_storage.go    # Storage creation TUI model
│       ├── create_vault.go      # Vault creation TUI model
│       └── create_scaffold.go   # Scaffold creation TUI model
├── go.mod
└── README.md
```

## Architecture

The CLI is built with:
- **[Cobra](https://github.com/spf13/cobra)**: Command structure and parsing
- **[Viper](https://github.com/spf13/viper)**: Configuration management (files, env vars, flags)
- **[Bubbletea](https://github.com/charmbracelet/bubbletea)**: Terminal UI framework for interactive commands
- **[Lipgloss](https://github.com/charmbracelet/lipgloss)**: Terminal styling for TUI components
- **[Bubbles](https://github.com/charmbracelet/bubbles)**: Common TUI components (text inputs, etc.)

All commands validate configuration before execution and communicate with the Platform API as a stateless client. Interactive commands use bubbletea for guided, user-friendly experiences.
