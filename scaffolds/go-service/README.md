# Go Service Scaffold

Copier template for generating production-ready Go microservices on the AKS Home Lab Platform.

## Overview

This scaffold generates:

- **Go service** with Chi router, structured logging (slog), health/ready endpoints, graceful shutdown
- **Kubernetes manifests** — Deployment, Service, HPA, PDB, NetworkPolicy (all Gatekeeper-compliant)
- **Crossplane Claims** (optional) — Azure Storage Bucket and/or Key Vault
- **CI/CD pipeline** — GitHub Actions with lint, test, build, Trivy scan, ACR push
- **Database migrations** (optional) — golang-migrate setup
- **gRPC support** (optional)

## Prerequisites

- [Copier](https://copier.readthedocs.io/) installed: `pip install copier`
- GitHub CLI (`gh`) for repo creation
- Access to the platform's Argo CD instance

## Usage

### Via Platform API/CLI (Recommended)

```bash
rdp scaffold create \
  --template go \
  --name my-api \
  --with-storage \
  --with-keyvault
```

The Platform API handles Copier execution, GitHub repo creation, and Argo CD onboarding.

### Direct Copier Usage (Development/Testing)

```bash
copier copy path/to/scaffolds/go-service ./my-new-service
cd my-new-service
```

## Template Variables

### Core Metadata

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `project_name` | str | *required* | Service name (lowercase, hyphens, 3-63 chars) |
| `project_description` | str | Auto-generated | Brief description |
| `go_module_path` | str | `github.com/rodmhgl/{name}` | Go module import path |

### Service Configuration

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `http_port` | int | 8080 | HTTP server port (1024-65535) |
| `include_grpc` | bool | false | Enable gRPC server? |
| `grpc_port` | int | 9090 | gRPC port (if enabled) |
| `include_database` | bool | false | Include golang-migrate setup? |
| `database_type` | str | postgres | Database type (postgres/mysql/mongodb) |

### Infrastructure (Crossplane Claims)

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `include_storage` | bool | false | Generate StorageBucket Claim? |
| `storage_location` | str | southcentralus | Azure region for storage |
| `storage_redundancy` | str | LRS | LRS/ZRS/GRS/GZRS |
| `include_keyvault` | bool | false | Generate Key Vault Claim? |
| `keyvault_location` | str | southcentralus | Azure region for vault |

### Kubernetes Configuration

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `namespace` | str | workloads | Target namespace |
| `replicas` | int | 2 | Static replica count |
| `enable_hpa` | bool | true | Enable Horizontal Pod Autoscaler? |
| `hpa_min_replicas` | int | 2 | HPA minimum replicas |
| `hpa_max_replicas` | int | 6 | HPA maximum replicas |
| `hpa_cpu_threshold` | int | 70 | HPA CPU target (%) |

### Resource Limits (MANDATORY)

These satisfy Gatekeeper's `container-limits-required` policy:

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `cpu_request` | str | 100m | CPU request |
| `cpu_limit` | str | 500m | CPU limit |
| `memory_request` | str | 128Mi | Memory request |
| `memory_limit` | str | 256Mi | Memory limit |

### Container Registry

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `acr_name` | str | rllabs | ACR name (without .azurecr.io) |

### CI/CD & GitHub

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `github_org` | str | rodmhgl | GitHub org/user |
| `enable_dependabot` | bool | true | Enable Dependabot? |
| `enable_codeowners` | bool | true | Generate CODEOWNERS? |
| `codeowners` | str | @rodmhgl | Comma-separated handles |

## Generated Structure

```
my-api/
├── main.go.jinja                    # Chi router, health, graceful shutdown
├── Dockerfile.jinja                 # Multi-stage build (golang:1.23-alpine)
├── Makefile.jinja                   # build, test, lint, docker targets
├── go.mod.jinja
├── .github/
│   └── workflows/
│       └── ci.yml.jinja             # Lint → Test → Build → Trivy → ACR push
├── k8s/
│   ├── kustomization.yaml.jinja
│   ├── deployment.yaml.jinja        # With resource limits, probes, security context
│   ├── service.yaml.jinja
│   ├── hpa.yaml.jinja               # (if enable_hpa)
│   ├── pdb.yaml.jinja               # PodDisruptionBudget (maxUnavailable: 1)
│   ├── networkpolicy.yaml.jinja     # Default deny with explicit egress rules
│   └── claims/
│       ├── storage.yaml.jinja       # (if include_storage) StorageBucket Claim
│       └── vault.yaml.jinja         # (if include_keyvault) Vault Claim
└── README.md.jinja                  # Generated project README

(if include_database)
├── migrations/
│   └── .gitkeep
└── internal/database/migrate.go.jinja

(if include_grpc)
├── proto/
│   └── service.proto.jinja
└── internal/grpc/server.go.jinja
```

## Compliance by Design

The scaffold generates code that **automatically satisfies** all Gatekeeper policies:

| Policy | How Satisfied |
|--------|---------------|
| `k8srequiredlabels` | Standard labels (`app.kubernetes.io/name`, etc.) in all K8s resources |
| `container-limits-required` | Resource requests/limits enforced via validators |
| `no-latest-tag` | CI pipeline uses commit SHA tags |
| `no-privileged-containers` | `securityContext.allowPrivilegeEscalation: false` in Deployment |
| `require-probes` | Liveness + readiness probes pre-configured |
| `allowed-repos` | Image from `{{ acr_name }}.azurecr.io` |
| `CrossplaneClaimLocation` | Claim `location` matches cluster region |
| `CrossplaneNoPublicAccess` | Composition defaults enforce `publicAccess: false` |

**Note:** Gatekeeper runs in **audit mode** during initial deployment, switching to **deny mode** after platform stabilizes. The scaffold ensures zero violations from day one.

## Crossplane Claims Integration

When `include_storage: true`, the scaffold generates:

```yaml
# k8s/claims/storage.yaml
apiVersion: platform.example.com/v1alpha1
kind: StorageBucket
metadata:
  name: my-api-data
spec:
  location: southcentralus
  redundancy: LRS
  publicAccess: false
  writeConnectionSecretToRef:
    name: my-api-storage-creds
```

**What happens after Argo CD syncs this:**

1. Crossplane reconciles the Claim
2. Composition creates: ResourceGroup → StorageAccount → BlobContainer
3. Connection secret (`my-api-storage-creds`) appears in the namespace with keys:
   - `AZURE_STORAGE_ACCOUNT_NAME`
   - `AZURE_STORAGE_ACCOUNT_KEY`
   - `AZURE_STORAGE_CONNECTION_STRING`

The Deployment template auto-wires these as env vars when storage is enabled.

## CI/CD Pipeline

The generated `.github/workflows/ci.yml` performs:

1. **Lint** — golangci-lint
2. **Test** — `go test -race -coverprofile`
3. **Build** — Multi-stage Docker build
4. **Security Scan** — Trivy (fails on HIGH/CRITICAL)
5. **Push to ACR** — Tag: `{acr}.azurecr.io/{project}:git-{sha}`
6. **Image Tag Update** — (Future) Commit updated tag to `k8s/deployment.yaml`, triggering Argo CD sync

## Updating Existing Projects

Copier supports **template updates** (unlike Cookiecutter):

```bash
cd my-existing-service
copier update
```

Copier will:
- Prompt for any new variables added to the template
- Re-render files, preserving your customizations (tracked via `.copier-answers.yml`)
- Show a diff of changes

**Use case:** When the platform team updates the scaffold (e.g., new Gatekeeper policy, security hardening), existing services can pull in those changes.

## Validators

The `copier.yml` includes strict validators to catch errors early:

- **Project name:** lowercase, 3-63 chars, no leading/trailing hyphens (DNS-safe)
- **Ports:** 1024-65535 range, no conflicts
- **Resources:** All limits are required (Gatekeeper enforces this in-cluster too)
- **HPA:** min < max, sane thresholds (20-95% CPU)
- **Regions:** Must match cluster region to satisfy `CrossplaneClaimLocation` policy

## Design Rationale

**Why Chi over standard library?** — Chi provides middleware chaining and route grouping without the weight of Gin/Echo. Performance is comparable to stdlib.

**Why slog over logrus/zap?** — `slog` is stdlib as of Go 1.21, structured by default, and doesn't require vendoring.

**Why multi-stage Dockerfile?** — Smaller images (alpine ~20MB vs. full golang ~800MB), faster pulls, reduced attack surface.

**Why NetworkPolicy?** — Zero-trust default: explicit egress rules for DNS, ACR, Azure APIs. Protects against lateral movement.

**Why PDB?** — Prevents simultaneous termination during node drains (important when `replicas=2` and HPA is scaling).

## Development Notes

### Testing the Scaffold

```bash
# From the scaffolds directory
copier copy go-service /tmp/test-service -d project_name=test-api -d include_storage=true

# Verify generated files
tree /tmp/test-service
```

### Adding New Variables

1. Add to `copier.yml` with type, help, default, and validator
2. Update templates to reference `{{ new_variable }}`
3. Update this README's variable table
4. Test with `copier copy` to ensure validators work

### Conditional File Generation

Copier supports excluding files based on answers:

```yaml
# In copier.yml
_exclude:
  - "{% if not include_grpc %}proto{% endif %}"
  - "{% if not include_database %}migrations{% endif %}"
```

## Future Enhancements

- [ ] Support for message queues (Service Bus, NATS)
- [ ] PostgreSQL/MySQL Crossplane Claims (when XRDs are added)
- [ ] OpenTelemetry tracing setup
- [ ] Feature flag integration (Flagsmith/LaunchDarkly)
- [ ] GraphQL server scaffold option
- [ ] End-to-end tests with Testcontainers

## See Also

- [Python Service Scaffold](../python-service/README.md)
- [Platform Design Document](../../PLATFORM_DESIGN.md)
- [Copier Documentation](https://copier.readthedocs.io/)
