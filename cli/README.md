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
```

Shows:
- **list**: Tabular view of all Claims with name, namespace, kind, status, ready/synced flags, age, and connection secret
- **status**: Detailed view including Claim details, Composite resource, Managed Azure resources, and recent Kubernetes events

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
├── main.go              # Entry point
├── cmd/
│   ├── root.go          # Root command and config management
│   ├── config.go        # Config subcommands (init, view, set)
│   ├── version.go       # Version command
│   ├── status.go        # Platform health summary
│   ├── apps.go          # Application commands (list, status, sync)
│   ├── infra.go         # Infrastructure commands (list, status)
│   └── ...              # Future command groups (scaffold, compliance, etc.)
├── go.mod
└── README.md
```

## Architecture

The CLI is built with:
- **[Cobra](https://github.com/spf13/cobra)**: Command structure and parsing
- **[Viper](https://github.com/spf13/viper)**: Configuration management (files, env vars, flags)

All commands validate configuration before execution and communicate with the Platform API as a stateless client.
