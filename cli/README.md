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
│   └── ...              # Future command groups (apps, infra, scaffold, etc.)
├── go.mod
└── README.md
```

## Architecture

The CLI is built with:
- **[Cobra](https://github.com/spf13/cobra)**: Command structure and parsing
- **[Viper](https://github.com/spf13/viper)**: Configuration management (files, env vars, flags)

All commands validate configuration before execution and communicate with the Platform API as a stateless client.
