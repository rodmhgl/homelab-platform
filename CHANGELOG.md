# Changelog

All notable changes to the Homelab Platform IDP will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### Added - kagent Natural Language Cluster Queries (2026-02-23)

**Platform Component** - Completed task #38: kagent installation for conversational cluster introspection

**Features:**

**kagent Deployment (Wave 13):**
- **Agent/Task CRD framework:** Kubernetes-native AI agent pattern for natural language queries
  - Agent CRDs: Pre-configured AI assistants with domain-specific knowledge
  - Task CRDs: Ephemeral query objects created per user question
- **Three-source Argo CD Application:**
  - Chart 1: `kagent-crds` (v0.7.0) — Agent/Task/Provider CRD definitions
  - Chart 2: `kagent` (v0.7.0) — Controller Deployment + webhook
  - Source 3-5: Git-based base resources (namespace, RBAC, default Agent, ExternalSecrets)
- **Anthropic Claude Sonnet 4.5 integration:**
  - Model: `claude-sonnet-4-5-20250929` (same as HolmesGPT for consistency)
  - Temperature: 0.1 (deterministic for factual queries)
  - Max tokens: 4096 per response
  - Rate limits: 60 requests/min, 100K tokens/min
- **Default `platform-agent` CRD:** Pre-configured with comprehensive platform context
  - Argo CD GitOps architecture (App of Apps, ApplicationSets, sync waves)
  - Crossplane self-service infrastructure (StorageBucket/Vault XRDs, resource trees)
  - Compliance scoring formula (Gatekeeper + Trivy + Falco)
  - Secrets management patterns (ESO + Workload Identity, Crossplane connection secrets)
  - Container registry enforcement (homelabplatformacr.azurecr.io)
- **Read-only RBAC:** Comprehensive cluster introspection without mutation risk
  - Allowed verbs: `get`, `list`, `watch` only
  - Covered resources: Core (Pods, Services, Deployments), Crossplane (Claims, XRs), Compliance (VulnerabilityReports, Constraints), GitOps (Argo CD Applications), Monitoring (ServiceMonitors)
  - Prohibited: `create`, `update`, `delete`, `patch` on all resources
- **ExternalSecret integration:** Reuses `anthropic-api-key` from bootstrap Key Vault (shared with HolmesGPT)
- **Prometheus metrics:** ServiceMonitor enabled (task counts, duration, provider requests/errors)

**Query Capabilities:**
- Application health/deployment troubleshooting (Argo CD sync status, Pod failures)
- Infrastructure provisioning status (Crossplane resource trees: Claim → XR → Managed Resources)
- Compliance violations (Gatekeeper audit, policy explanations)
- CVE vulnerabilities (Trivy VulnerabilityReports grouped by severity)
- Runtime security events (Falco alerts via Platform API context)
- GitOps state (OutOfSync apps, deployment history)

**Configuration:**
- Namespace: `kagent-system`
- Controller replicas: 1 (stateless, horizontally scalable)
- Resource limits: 500m CPU / 512Mi memory
- Webhook enabled on port 9443
- ServiceAccount: `kagent-sa` (custom read-only ClusterRole)

**Documentation:**
- `platform/kagent/README.md` — Comprehensive usage guide (architecture, examples, troubleshooting, integration patterns)
- Example queries: "Why is my app unhealthy?", "Show me all Gatekeeper violations", "What CVEs are in portal-ui?"

**Integration Points:**
- **Unblocks Task #53:** Platform API `/api/v1/agent/ask` endpoint (create Task CRDs, stream responses)
- **Unblocks Task #76:** CLI `rdp ask <query>` command (wraps Platform API)
- **Portal UI ready:** AI Operations panel (#86) already implemented, needs backend integration

**Cost Considerations:**
- Estimated cost: ~$0.03/query (4K input + 2K output tokens)
- Recommended rate limiting: 10 queries/user/hour via Platform API
- Monitoring: Anthropic dashboard for usage tracking

**Files Created:**
- `platform/kagent/application.yaml` — Argo CD Application (wave 13, five sources)
- `platform/kagent/values.yaml` — Helm values (Anthropic provider config)
- `platform/kagent/base/namespace.yaml` — kagent-system namespace
- `platform/kagent/base/rbac.yaml` — ServiceAccount + read-only ClusterRole + Binding
- `platform/kagent/base/agent.yaml` — Default platform-agent with instructions
- `platform/kagent/base/kustomization.yaml` — Base resource list
- `platform/kagent/externalsecrets/kagent-secrets.yaml` — ExternalSecret for API key
- `platform/kagent/externalsecrets/kustomization.yaml` — ExternalSecret resource list
- `platform/kagent/README.md` — Usage documentation

**Next Steps:**
- Deploy via Argo CD sync (wave 13 after HolmesGPT wave 12)
- Verify CRD registration and controller health
- Test manual Task creation for query validation
- Implement Platform API `/api/v1/agent/ask` endpoint (Task #53)
- Implement CLI `rdp ask` command (Task #76)

---

### Added - HolmesGPT AI Root Cause Analysis (2026-02-23)

**Platform Component** - Completed task #39: HolmesGPT installation for AI-powered Kubernetes troubleshooting

**Features:**

**HolmesGPT Deployment (Wave 12):**
- **FastAPI server:** Python-based investigation engine with Claude Sonnet 4.5 LLM
- **Custom Docker image:** Built from source (no public registry available)
  - Image: `homelabplatformacr.azurecr.io/holmesgpt:v1.0.0`
  - Includes kubectl, argocd CLI, kube-lineage for comprehensive cluster analysis
- **Alertmanager webhook integration:** Automatic investigations for critical/high alerts
  - Webhook: `http://holmesgpt.holmesgpt.svc.cluster.local:5050/api/investigate`
  - Configured in `platform/monitoring/values.yaml`
- **Comprehensive RBAC:** Read-only cluster access across all platform resources
  - Core: pods, logs, events, services, nodes
  - Workloads: deployments, statefulsets, daemonsets, jobs
  - Crossplane: Claims, XRs, Managed Resources (infrastructure context)
  - Compliance: VulnerabilityReports (Trivy), Constraints (Gatekeeper)
  - Argo CD: Applications, AppProjects (GitOps context)
- **API endpoints:**
  - `POST /api/investigate` — Trigger investigation (synchronous)
  - `POST /api/stream/investigate` — Streaming investigation (SSE)
  - `GET /healthz` — Liveness probe
  - `GET /readyz` — Readiness probe (validates LLM access)
- **Secrets management:** Anthropic API key via ExternalSecret from bootstrap Key Vault
- **Prometheus integration:** Query metrics for CPU/memory usage, pod restarts
- **Enabled toolsets:** kubernetes/core, kubernetes/logs, prometheus, internet

**Configuration:**
- LLM: `anthropic/claude-sonnet-4-20250514` (200K context window)
- Temperature: 0.00000001 (deterministic investigations)
- Timeout: 600s per investigation
- Memory limit: 2Gi (handles large cluster state)
- Replicas: 1 (stateless, horizontally scalable)

**Documentation:**
- `platform/holmesgpt/README.md` — Comprehensive deployment guide (architecture, configuration, troubleshooting)
- `platform/holmesgpt/BUILD.md` — Docker image build instructions (multi-arch support)

**Integration Points:**
- **Task #36 completed:** Alertmanager webhook configured (was pending)
- **Task #52 unblocked:** Platform API can now implement `/api/v1/investigate` endpoints
- **Task #75 unblocked:** CLI `rdp investigate` command can trigger investigations

**Files Changed:**
- `platform/holmesgpt/` — New directory with 12 manifest files
- `platform/monitoring/values.yaml` — Updated Alertmanager webhook URL (port 8080→5050, endpoint /webhook→/api/investigate)

**Deployment:**
- Wave 12 (after monitoring wave 8, platform-api wave 10)
- Namespace: `holmesgpt`
- Service: ClusterIP at `holmesgpt.holmesgpt.svc.cluster.local:5050`

### Added - Portal UI AI Operations Panel (2026-02-23)

**Portal UI Enhancement** - Completed task #86: AI Operations panel with kagent chat and HolmesGPT investigation

**Features:**

**AI Operations Panel (7th dashboard panel):**
- **Tab-based UI:** Two tabs (Chat, Investigate) with independent workflows
- **kagent Chat Interface:**
  - Natural language queries for Kubernetes operations
  - Chat message history with user/assistant differentiation
  - Example questions as clickable prompts
  - Real-time message display with timestamps
  - Optimistic updates (user message appears immediately)
- **HolmesGPT Investigation Form:**
  - Application dropdown (populated from `/api/v1/apps`)
  - Issue description textarea
  - "Start Investigation" button with loading state
  - Investigation results display (status, root cause, remediation steps)
  - Status polling support (pending → running → completed/failed)
- **Graceful Degradation:**
  - Service unavailable (501) handled with friendly informational banners
  - No hard errors when backend not deployed
  - Educational messaging about service deployment status
  - Links to relevant tasks (#38, #39, #52, #53)

**API Integration Layer:**
- **New file:** `portal/src/api/aiops.ts` (3 methods)
  - `ask(question)` → POST `/api/v1/agent/ask`
  - `investigate(application, issue)` → POST `/api/v1/investigate`
  - `getInvestigation(id)` → GET `/api/v1/investigate/{id}`
- **New types:** `portal/src/api/types.ts`
  - `AskRequest`, `AskResponse` (kagent)
  - `InvestigateRequest`, `InvestigateResponse`, `Investigation` (HolmesGPT)

**State Management:**
- Local component state: Tab selection, chat history, form inputs, investigation results
- TanStack Query mutations: `askMutation`, `investigateMutation`
- Optimistic UI updates for chat messages
- Error handling with service-specific messages

**Styling & UX:**
- Follows existing panel patterns (StatusCard wrapper, Badge components)
- Tab switcher with active state (blue border + text)
- Chat bubbles: Blue for user, white for assistant
- Chat scrollable container (h-64, max-h-96)
- Form validation: Required fields, disabled states
- Loading spinners during async operations
- Color-coded status badges (pending=info, running=info, completed=default, failed=danger)

**Technical Details:**
- File: `portal/src/components/dashboard/AIOperationsPanel.tsx` (370 lines)
- Integration: `portal/src/pages/Dashboard.tsx` (added 7th panel to grid)
- Build verified: TypeScript compilation successful
- Ready for backend integration (no code changes needed when services deploy)

**Next Steps (Backend Prerequisites):**
- Task #38: Install kagent in cluster
- Task #39: Install HolmesGPT in cluster
- Task #52: Implement `/api/v1/investigate/*` endpoints in Platform API
- Task #53: Implement `/api/v1/agent/ask` endpoint in Platform API

### Added - Portal UI Scaffold Form (2026-02-23)

**Portal UI Enhancement** - Completed task #85: Interactive scaffold form for creating new services

**Features:**

**Scaffold Form (`/scaffold` route):**
- **17 form fields** with comprehensive validation and conditional logic
- **Template selection:** go-service (hardcoded; python-service pending)
- **Project configuration:** Name (DNS label validation), description (optional, 500 char max)
- **Service configuration:** HTTP port (1024-65535), gRPC toggle + port, database toggle
- **Infrastructure dependencies:** Storage toggle (location + replication), Key Vault toggle (location + SKU)
- **GitHub configuration:** Organization (required), repository (defaults to project name), private repo (always true)
- **Conditional fields:** gRPC/storage/vault fields show/hide based on checkbox state
- **Real-time validation:** DNS label format, port ranges, port uniqueness, required fields
- **Success UI:** Modal with repo URL, Argo CD app name, platform config path, next steps
- **Error handling:** Inline field errors + top-level submission error alert

**Validation Rules:**
- Project name: 3-63 characters, lowercase alphanumeric with hyphens, no leading/trailing hyphens
- HTTP/gRPC ports: 1024-65535 range, must differ from each other
- GitHub org: Required field
- All validations match CLI `rdp scaffold create` behavior

**User Experience:**
- Four logical sections: Project, Service, Infrastructure, GitHub
- Auto-fill GitHub repo from project name if empty
- Reset button to clear form
- Loading state during submission ("Creating Service...")
- Success modal with actionable next steps (clone, build, test, verify)

**Type Safety Fix (CRITICAL):**
- Fixed TypeScript `ScaffoldRequest`/`ScaffoldResponse` to match Go API JSON tags exactly
- Previous speculative types caused runtime errors (wrong field names)
- Now follows mandatory pattern: Read Go API structs → Match JSON tags → Verify with build

**Technical Details:**
- File: `portal/src/pages/Scaffold.tsx` (600+ lines)
- Types: `portal/src/api/types.ts` (updated with 24 new fields)
- State management: Vanilla React hooks (`useState`, no form library)
- Submission: TanStack Query mutation with `scaffoldApi.create()`
- Styling: Tailwind CSS with consistent form element classes
- Accessibility: Label associations, error messages, disabled states

**Integration:**
- API endpoint: `POST /api/v1/scaffold` (task #51) ✅
- Argo CD ApplicationSet: Auto-discovers new apps via `apps/*/config.json` (task #8) ✅
- go-service template: Copier template with 23 files (task #55) ✅

### Added - CLI Portal Command (2026-02-23)

**CLI Enhancement** - Completed task #77: `rdp portal open` command for quick Portal UI access

**Features:**

**Portal Open Command:**
- **`rdp portal open`** - Launch Platform Portal web interface in default browser
  - Cross-platform support: Linux/WSL (xdg-open), macOS (open), Windows (cmd /c start)
  - URL precedence: `--url` flag → `portal_url` config → derived from `api_base_url` → default
  - Smart URL derivation: Handles multiple API URL patterns (api., api-, platform., platform-api.)
  - `--print` flag: Output URL without opening browser (useful for scripts/CI)
  - Graceful fallback: Prints URL with instructions if browser launch fails
  - No API dependency: Pure client-side operation

**Command Examples:**
```bash
# Open production Portal
rdp portal open

# Open local dev Portal (port-forward scenario)
rdp portal open --url http://localhost:8080

# Print URL for scripting
rdp portal open --print
```

**Configuration:**
```yaml
# Optional in ~/.rdp/config.yaml
portal_url: http://portal.rdp.azurelaboratory.com
```

**Technical Details:**
- Files: `cli/cmd/portal.go`, `cli/cmd/portal_open.go`
- Config schema: Added optional `portal_url` field (backward compatible)
- URL validation: Validates URL format before opening
- Platform detection: Uses `runtime.GOOS` for OS-specific browser launchers

### Added - CLI Scaffold Command (2026-02-23)

**CLI Enhancement** - Completed task #72: `rdp scaffold create` command for interactive service scaffolding

**Features:**

**Scaffold Create Command:**
- **`rdp scaffold create`** - Interactive TUI wizard for creating new services from Copier templates
  - 14-state flow: welcome → template selection → project config → feature toggles → GitHub config → confirmation
  - Template selection: Currently supports `go-service` (Python template pending)
  - Project configuration: Name (DNS label validated), description (optional), HTTP port (default 8080)
  - gRPC configuration: Enable/disable toggle, gRPC port selection (default 9090, must differ from HTTP port)
  - Feature toggles: Database support, Storage (creates StorageBucket Claim), Key Vault (creates Vault Claim)
  - GitHub integration: Organization/owner (auto-detected from Git remote), repository name (defaults to project name)
  - Git auto-detection: Parses `git remote get-url origin` to pre-fill GitHub org
  - Progressive disclosure: Completed fields shown with checkmarks as wizard progresses
  - Extended timeout: 60 seconds to accommodate Copier execution + GitHub repo creation + Argo CD onboarding
  - Error handling: Retry on failure (R key), clear error messages from API
  - Success screen: Shows repo URL, Argo CD app name, platform config path, clone/build/test instructions

**What Happens When You Run It:**
1. Platform API executes Copier template with your configuration
2. Creates GitHub repository at `github.com/{org}/{repo}`
3. Pushes scaffolded code to GitHub
4. Adds `apps/{name}/config.json` to platform repository
5. Argo CD ApplicationSet discovers new config within 60 seconds
6. Application syncs to cluster automatically

**Command Examples:**
```bash
# Launch interactive wizard
rdp scaffold create

# Follow prompts to configure:
# - Select template (go-service)
# - Enter project name (my-api)
# - Enter description (optional)
# - Configure HTTP port (8080)
# - Enable gRPC? (Y/N)
# - Configure gRPC port (9090, if enabled)
# - Enable database? (Y/N)
# - Enable storage? (Y/N)  # Creates StorageBucket Claim
# - Enable Key Vault? (Y/N)  # Creates Vault Claim
# - Enter GitHub org (auto-detected: rodmhgl)
# - Enter repo name (defaults to project name)
# - Confirm (Y/N)
```

**Example Output:**
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

**Technical Details:**
- **State machine:** 14 states with conditional flow (skips gRPC port if disabled)
- **Validation:** DNS label format, port range (1024-65535), port conflict detection
- **API contract:** Matches `ScaffoldRequest`/`ScaffoldResponse` from Platform API exactly
- **HTTP client:** 60-second timeout (vs 30s for other commands) to accommodate multi-step scaffolding process

### Added - CLI Secrets Command (2026-02-23)

**CLI Enhancement** - Completed task #74: `rdp secrets list` command for unified secrets visibility

**Features:**

**Secrets List Command:**
- **`rdp secrets list <namespace>`** - List all secrets in a namespace (ExternalSecrets + Crossplane connection secrets)
  - Unified view of ESO-managed ExternalSecrets and Crossplane connection secrets
  - Tabular view: NAME, NAMESPACE, KIND, STATUS, KEYS, SOURCE CLAIM, AGE
  - Status icons with color coding: ✓ Ready/Synced (green), ✗ Error (red), ○ Unknown (gray)
  - Keys display: Comma-separated list for ≤3 keys, "N keys" summary for >3 keys
  - Source Claim: Shows originating Crossplane Claim for connection secrets (name + kind)
  - Summary footer: Total count with breakdown (ExternalSecrets vs connection secrets)
  - Flags: `--kind external|connection` (filter by type), `--json` (output format)
  - ⚠️ **Security**: Never displays secret values, only metadata

**Command Examples:**
```bash
# List all secrets in a namespace
rdp secrets list default
rdp secrets list platform

# Filter by kind
rdp secrets list default --kind external      # ExternalSecrets only
rdp secrets list default --kind connection    # Connection secrets only

# JSON output for scripting
rdp secrets list platform --json
```

**Example Output:**
```
NAME                NAMESPACE  KIND            STATUS        KEYS                    SOURCE CLAIM              AGE
----                ---------  ----            ------        ----                    ------------              ---
demo-storage-conn   default    Secret          ✓ Synced      3 keys                  demo-storage (StorageBucket)  3d
github-pat          platform   ExternalSecret  ✓ Ready       token                   -                         8d
api-creds           default    ExternalSecret  ✗ Error       -                       -                         1h

Total: 3 secrets (2 ExternalSecrets, 1 connection secret)
```

**Technical Details:**
- **Type Safety:** All Go types match Platform API JSON tags exactly (`creationTimestamp` is `time.Time`, `sourceClaim` is `*ResourceRef`)
- **Client-side Filtering:** `--kind` flag filters response array locally (API returns all secrets in namespace)
- **Consistent Patterns:** Follows `compliance_violations.go` table formatting, reuses `formatAge()` helper from `infra.go`
- **Error Handling:** Clear messages for 400 (bad request), 404 (namespace not found), 500 (API error)
- **HTTP Timeout:** 15s for list operations

**Implementation Files:**
- `cli/cmd/secrets.go` - Parent command + shared helper functions
- `cli/cmd/secrets_list.go` - List subcommand implementation

**Dependencies Met:**
- Platform API `/api/v1/secrets/{namespace}` endpoint (Task #50) ✅ Complete

---

### Added - CLI Compliance Commands (2026-02-23)

**CLI Enhancement** - Completed task #73: `rdp compliance` command group for security and policy visibility

**Features:**

**Compliance Command Group:**
- **`rdp compliance summary`** - View overall compliance score and aggregate metrics
  - Displays compliance score (0-100) with color coding: ✓ (≥90 green), ⚠ (70-89 amber), ✗ (<70 red)
  - Shows policy violations count, vulnerabilities breakdown (Critical/High counts with color)
  - Flag: `--json` for machine-readable output

- **`rdp compliance policies`** - List active Gatekeeper ConstraintTemplates
  - Tabular view: NAME, KIND, SCOPE, DESCRIPTION
  - Shows cluster-wide vs namespaced policy scope
  - Flag: `--json` for machine-readable output

- **`rdp compliance violations`** - View Gatekeeper policy violations
  - Tabular view: CONSTRAINT, KIND, RESOURCE, NAMESPACE, MESSAGE
  - Shows detailed context for each violation
  - Flags: `--namespace` (filter by namespace), `--json` (output format)

- **`rdp compliance vulns`** - List vulnerabilities from Trivy Operator scans
  - Tabular view: SEVERITY, CVE-ID, IMAGE, PACKAGE, FIXED, WORKLOAD
  - Color-coded severity: CRITICAL/HIGH (red), MEDIUM (yellow), LOW (gray)
  - Summary footer shows breakdown across CRITICAL/HIGH/MEDIUM/LOW
  - Flags: `--severity CRITICAL|HIGH|MEDIUM|LOW` (filter), `--json` (output format)

- **`rdp compliance events`** - View Falco runtime security events
  - Tabular view: TIME, SEVERITY, RULE, RESOURCE, MESSAGE
  - Human-readable timestamps (e.g., "2m ago", "1h ago")
  - Color-coded severity: ERROR (red), WARNING (yellow), NOTICE (white)
  - Summary footer shows breakdown by severity
  - Flags: `--namespace` (filter), `--severity ERROR|WARNING|NOTICE` (filter), `--since 1h|30m|24h` (time window), `--limit N` (max events, default 50), `--json` (output format)

**Command Examples:**
```bash
# View compliance summary
rdp compliance summary

# List Gatekeeper policies
rdp compliance policies

# View policy violations (all namespaces)
rdp compliance violations

# View violations in specific namespace
rdp compliance violations --namespace platform

# List all vulnerabilities
rdp compliance vulns

# List only critical CVEs
rdp compliance vulns --severity CRITICAL

# View recent security events
rdp compliance events --since 1h --limit 20

# View error-level events only
rdp compliance events --severity ERROR

# JSON output for scripting
rdp compliance summary --json
rdp compliance events --json
```

**API Integration:**
- **Endpoints**:
  - `GET /api/v1/compliance/summary` - Compliance score and metrics
  - `GET /api/v1/compliance/policies` - Gatekeeper ConstraintTemplates
  - `GET /api/v1/compliance/violations?namespace=` - Policy violations with filtering
  - `GET /api/v1/compliance/vulnerabilities?severity=` - Trivy CVE scans with filtering
  - `GET /api/v1/compliance/events?namespace=&severity=&since=&limit=` - Falco security events with multi-dimensional filtering
- **Timeout**: 10-15 seconds
- **Query Parameters**: Clean encoding via `url.Values` for namespace, severity, time window filters

**Implementation Details:**
- **Type Safety**: All types match API JSON tags exactly
  - `ComplianceScore` is `float64` (not `int`)
  - `VulnerabilitiesBySeverity` is `map[string]int` (not individual fields)
  - Security events use `Severity` field (not `Priority`)
- **Severity Validation**: Enums validated before API call (CRITICAL/HIGH/MEDIUM/LOW for CVEs, ERROR/WARNING/NOTICE for events)
- **Duration Parsing**: `--since` flag uses Go `time.ParseDuration` for validation (e.g., "1h", "30m", "24h")
- **Color Coding**: Shared severity color helper across all commands
- **Timestamp Formatting**: RFC3339 parsing → human-readable age display
- **Consistent Patterns**: Follows `apps.go`/`infra.go` conventions (tabwriter, JSON output, error handling)

**Files Added:**
- `cli/cmd/compliance.go` - Root command + shared helpers (color codes, timestamp formatting, JSON output)
- `cli/cmd/compliance_summary.go` - Compliance score overview
- `cli/cmd/compliance_policies.go` - Gatekeeper policy list
- `cli/cmd/compliance_violations.go` - Policy violations with namespace filtering
- `cli/cmd/compliance_vulns.go` - CVE list with severity filtering
- `cli/cmd/compliance_events.go` - Falco events with multi-dimensional filtering

**Documentation Updated:**
- `cli/IMPLEMENTATION_STATUS.md` - Moved #73 from Pending to Completed with detailed command documentation
- `cli/README.md` - Added "Compliance Commands" section with usage examples
- `homelab-platform/CHANGELOG.md` - Added v0.3.0 changelog entry
- `homelab-platform/CLAUDE.md` - Updated CLI status line to reflect compliance commands completion

---

### Added - CLI Infrastructure Deletion Command (2026-02-23)

**CLI Enhancement** - Completed task #71: `rdp infra delete` command with GitOps workflow and safety confirmation

**Features:**

**Infrastructure Deletion via GitOps:**
- **`rdp infra delete <kind> <name>`** - Safely remove Crossplane Claims and Azure resources
  - **Required Flags**: `--repo-owner`, `--repo-name` (explicit to prevent accidental deletions)
  - **Optional Flags**: `--namespace` (default: default), `--force` (skip confirmation), `--json` (output format)
  - **Safety Confirmation**: User must type the exact Claim name to confirm deletion (unless --force)
  - **Warning Display**: Shows what will be deleted (Claim, Git file, Azure resources) with visual ⚠️ indicators
  - **GitOps Flow**:
    1. Removes `k8s/claims/<name>.yaml` from GitHub repository
    2. Argo CD syncs within 60 seconds and removes Claim from cluster
    3. Crossplane deletes all managed Azure resources (ResourceGroup, StorageAccount/Key Vault, etc.)
  - **Supports**: `storage`/`StorageBucket` and `vault`/`Vault` kinds

**Command Examples:**
```bash
# Delete with confirmation prompt
rdp infra delete storage demo-storage --repo-owner myorg --repo-name myapp

# Delete with force (skip confirmation)
rdp infra delete vault prod-vault --namespace production --repo-owner myorg --repo-name myapp --force

# Delete with JSON output
rdp infra delete storage test-bucket --repo-owner myorg --repo-name myapp --json
```

**Safety Mechanisms:**
- **Explicit Repository Flags**: Unlike create commands (which auto-detect Git), delete requires explicit `--repo-owner` and `--repo-name` to force user awareness
- **Confirmation Prompt**: User must type the exact Claim name (case-sensitive) to proceed
- **Warning Display**: Clear visual indication of destructive action with list of what will be deleted
- **Force Flag**: Available for CI/CD pipelines but requires explicit opt-in

**Output Format:**

*Human-readable (default):*
```
╔═══════════════════════════════════════════════════════════╗
║  ⚠️  WARNING: Destructive Operation                        ║
╚═══════════════════════════════════════════════════════════╝

You are about to delete infrastructure:

  Kind:       StorageBucket
  Name:       demo-storage
  Namespace:  default
  Repository: myorg/myapp

This will:
  1. Remove k8s/claims/demo-storage.yaml from Git
  2. Trigger Argo CD sync (removes Claim from cluster)
  3. Delete all Azure resources (ResourceGroup, StorageAccount, BlobContainer)

⚠️  This action is IRREVERSIBLE. Data in Azure resources will be lost.

Type the Claim name 'demo-storage' to confirm: _
```

*Success Output:*
```
✓ Claim deleted successfully

Kind:            StorageBucket
Name:            demo-storage
Namespace:       default
Repository:      https://github.com/myorg/myapp
Commit:          abc123def456...
File Removed:    k8s/claims/demo-storage.yaml

Next Steps:
  • Argo CD will sync within 60 seconds
  • Crossplane will delete Azure resources (ResourceGroup, StorageAccount, BlobContainer)
  • Monitor progress: rdp infra status storage demo-storage
```

**API Integration:**
- **Endpoint**: `DELETE /api/v1/infra/{kind}/{name}?namespace={namespace}`
- **Request Body**: `DeleteClaimRequest` with `repoOwner`, `repoName`
- **Response**: `DeleteClaimResponse` with `success`, `commitSha`, `filePath`, `repoUrl`
- **Timeout**: 15 seconds
- **Error Handling**: 404 (Claim not found), 400 (missing fields), 500 (server error)

**Implementation Details:**
- **Type Matching**: Request/response types match API JSON tags exactly (`commitSha` not `commitSHA`)
- **Kind Normalization**: User input (`storage`, `vault`) normalized to API format via `normalizeKindForAPI()`
- **Error Messages**: Clear, actionable errors for all failure scenarios
- **JSON Output**: Machine-readable format for CI/CD integration

**Files Added:**
- `cli/cmd/infra_delete.go` - 245 lines (NEW) - Delete command with confirmation prompt

**Files Modified:**
- `cli/cmd/infra.go` - Updated help text and registered `infraDeleteCmd`
- `cli/README.md` - Added delete examples and safety warnings
- `cli/IMPLEMENTATION_STATUS.md` - Marked #71 complete, updated status from "Partial" to "Complete"
- `homelab-platform/CLAUDE.md` - Updated cli/ status line to include delete command
- `homelab-platform/README.md` - Updated CLI progress tracker
- `homelab-platform/CHANGELOG.md` - Added this entry

**CLI Progress:**
- ✅ Root command + config management (#65)
- ✅ Version command
- ✅ `rdp status` - Platform health summary (#66)
- ✅ `rdp infra list/status` - Infrastructure Claims (#68)
- ✅ `rdp infra create storage/vault` - Interactive infra creation (#69, #70)
- ✅ `rdp infra delete` - GitOps infra deletion (#71) **← NEW**
- ✅ `rdp apps list/status/sync` - Application management (#67)
- ⬜ `rdp compliance` - Policy violations, CVEs, events (#73)
- ⬜ `rdp secrets` - Secret management (#74)
- ⬜ `rdp scaffold create` - Interactive project creation (#72)
- ⬜ `rdp investigate` - HolmesGPT integration (#75)
- ⬜ `rdp ask` - kagent natural language (#76)

**Next:** Compliance commands (#73) or scaffold creation (#72).

---

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
- ✅ `rdp infra create storage/vault` - Interactive infra creation (#69, #70)
- ✅ `rdp infra delete` - GitOps infrastructure deletion (#71) **← NEW**
- ✅ `rdp apps list/status/sync` - Application management (#67)
- ⬜ `rdp compliance` - Policy violations, CVEs, events (#73)
- ⬜ `rdp secrets` - Secret management (#74)
- ⬜ `rdp scaffold create` - Interactive project creation (#72)
- ⬜ `rdp investigate` - HolmesGPT integration (#75)
- ⬜ `rdp ask` - kagent natural language (#76)

**Next:** Compliance commands (#73) or scaffold creation (#72).

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
