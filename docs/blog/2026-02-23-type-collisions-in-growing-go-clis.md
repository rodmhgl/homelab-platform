# Type Collisions in Growing Go CLIs: When Your Own Code Becomes the Enemy

**Date:** February 23, 2026
**Author:** Rod Stewart (with Claude Sonnet 4.5)
**Project:** AKS Home Lab Internal Developer Platform
**Task:** Implementing `rdp apps list/status/sync` (#67)

## The Task

Add Argo CD application management to the `rdp` CLI — list apps, show detailed status, trigger syncs. Three subcommands, matching the established `rdp infra` pattern. Should be straightforward.

It wasn't.

## The Collision

The CLI uses Cobra, which conventionally places all commands in a single `cmd` package. When `rdp infra` was the only command, this was fine. Adding `rdp apps` created a namespace collision that broke the entire build:

```text
cmd/status.go:38:6: HealthStatus redeclared in this block
    cmd/apps.go:201:6: other declaration of HealthStatus
cmd/status.go:52:6: ApplicationStatus redeclared in this block
    cmd/apps.go:172:6: other declaration of ApplicationStatus
cmd/status.go:349:6: formatHealthIcon redeclared in this block
    cmd/apps.go:660:6: other declaration of formatHealthIcon
```

Three collisions. `HealthStatus` existed in `status.go` as a simple health check struct (healthy/ready/error). The Argo CD API also has a `HealthStatus` (status/message). Both are correct in their domain — they just can't coexist.

### The Root Cause

Go's package design means **all exported identifiers in a package are global within that package**. Two files, same package, same type name — compiler error. Cobra's `cmd/` convention makes this inevitable as commands grow.

### The Fix

Domain-prefix the types that collide:

```go
// status.go — platform health check (renamed)
type APIHealthStatus struct {
    Healthy bool
    Ready   bool
    Error   string
}

type ApplicationsStatus struct {  // plural to disambiguate
    Total    int
    Healthy  int
    Degraded int
    Error    string
}

func formatAPIHealthIcon(healthy bool) string { ... }
```

```go
// apps.go — Argo CD health (matches API contract)
type HealthStatus struct {
    Status  string `json:"status,omitempty"`
    Message string `json:"message,omitempty"`
}

func formatHealthIcon(healthStatus string) string { ... }
```

**Rule of thumb:** The type name should match the most authoritative source. `HealthStatus` in the API types file wins; the aggregate status check gets prefixed.

## The Hidden Bug

Fixing the collision revealed a pre-existing bug in `status.go` that had been silently producing wrong results:

```go
// WRONG (original status.go)
var apps struct {
    Apps []struct {
        Health string `json:"health"`
    } `json:"apps"`
}
status.Total = len(apps.Apps)
```

The API actually returns:

```go
// api/internal/argocd/types.go
type ListAppsResponse struct {
    Applications []ApplicationSummaryResponse `json:"applications"`
    Total        int                          `json:"total"`
}
```

Two mismatches: `apps` vs `applications`, and `health` vs `healthStatus`. The JSON decoder silently ignored both — it just left the struct fields at their zero values. The Applications section in `rdp status` was always reporting 0 apps/0 healthy/0 degraded, and nobody noticed because the error was *plausible* (looked like empty data, not a bug).

### Why Silent JSON Unmarshalling Failures Are Dangerous

Go's `encoding/json` doesn't error on unknown or missing fields by default. If your struct field is `Apps` but the JSON key is `applications`, the decoder shrugs and moves on. The field stays empty. No error, no warning, no panic. Just wrong data.

```go
// This succeeds without error, but apps.Items is always nil
var apps struct {
    Items []string `json:"items"`
}
json.Unmarshal([]byte(`{"applications":["foo"]}`), &apps)
// apps.Items == nil — silent failure
```

**This is the Go equivalent of the TypeScript type mismatch problem** we've hit three times in the Portal UI. The same root cause — speculative types instead of verified ones — just manifests differently:

- **TypeScript:** Runtime error (`Cannot read properties of undefined`)
- **Go:** Silent data loss (zero values instead of real data)

The Go version is arguably worse because it doesn't crash. It just lies.

## Verification-First Type Derivation

The workflow that prevents both failure modes:

1. **Read the API struct first** — `api/internal/argocd/types.go` is the source of truth
2. **Copy JSON tags exactly** — `json:"applications"` means the field name is `applications`, not `apps`
3. **Match optional patterns** — `omitempty` in Go → optional (`?`) in TypeScript, pointer type → nullable
4. **Build immediately** — `go build` catches type collisions; `npm run build` catches TS mismatches
5. **Never invent types speculatively** — If the API doesn't exist yet, wait

Applied to this implementation:

```go
// API source of truth (api/internal/argocd/types.go)
type ApplicationSummaryResponse struct {
    Name         string     `json:"name"`
    Project      string     `json:"project"`
    SyncStatus   string     `json:"syncStatus"`
    HealthStatus string     `json:"healthStatus"`   // NOT "health"
    RepoURL      string     `json:"repoURL"`
    LastDeployed *time.Time `json:"lastDeployed,omitempty"`  // NOT "lastSyncedAt"
}

// CLI type (cli/cmd/apps.go) — matches exactly
type ApplicationSummary struct {
    Name         string     `json:"name"`
    Project      string     `json:"project"`
    SyncStatus   string     `json:"syncStatus"`
    HealthStatus string     `json:"healthStatus"`
    RepoURL      string     `json:"repoURL"`
    LastDeployed *time.Time `json:"lastDeployed,omitempty"`
}
```

## Architecture: Consistency Over Novelty

The `rdp apps` implementation deliberately copies patterns from `rdp infra` rather than inventing new ones:

**Tabwriter for lists:**

```go
w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
fmt.Fprintln(w, "NAME\tPROJECT\tSYNC\tHEALTH\tREPO\tPATH\tAGE\tLAST DEPLOYED")
fmt.Fprintln(w, "----\t-------\t----\t------\t----\t----\t---\t-------------")
```

**Unicode boxes for detail views:**

```text
╔═══════════════════════════════════════════════════════════╗
║  Application: platform-api                                ║
╚═══════════════════════════════════════════════════════════╝

┌─ Sync Status ─────────────────────────────────────────────┐
│ Status:      ✓ Synced
│ Compared To: github.com/org/platform @ main
│ Last Sync:   2026-02-23 10:45:32
└───────────────────────────────────────────────────────────┘
```

**Shared status icons:** ✓ (good), ⚠ (needs attention), ✗ (error), ○ (unknown)

**Shared formatting:** `formatAge()` from `infra.go` reused for time display.

This matters because **future maintainers can understand new commands instantly** if they follow the established visual language. Adding `rdp compliance` or `rdp secrets` should take hours, not days, because every pattern is already proven.

## Scaling Strategies

The single-package approach works for 3–5 commands. At 10+, these options emerge:

| Strategy | Pros | Cons |
| --- | --- | --- |
| Domain-prefix types (`AppsHealthStatus`) | No restructuring needed | Names get verbose |
| Shared `types.go` | Single source of truth | Must coordinate across commands |
| Package per command (`cmd/apps/`, `cmd/infra/`) | Clean namespaces | Breaks Cobra conventions, more boilerplate |
| Code generation from API types | Perfect alignment | Build tooling overhead |

The current codebase is at the "domain-prefix" stage. The next command that collides should trigger a `types.go` refactor.

## Key Takeaways

**1. Go's `encoding/json` fails silently.** Mismatched field names don't error — they produce zero values. This is the most dangerous class of bug in Go API clients because the code runs without errors while returning wrong data.

**2. Cobra's `cmd/` package is a namespace minefield.** Every file shares the same namespace. Plan type names proactively, not reactively after the compiler complains.

**3. Verification-first beats speculation.** Reading the API types file takes 2 minutes. Debugging silent JSON failures takes hours. The math is unambiguous.

**4. Collisions surface pre-existing bugs.** The type rename in `status.go` forced reading the code carefully, which revealed the `apps`/`applications` mismatch that had been silently failing since day one.

**5. Consistency compounds.** Copying the `infra.go` pattern meant `apps.go` was implemented in a fraction of the time. Every new command that follows the established pattern reinforces the investment.

---

## References

- **Implementation:** [cli/cmd/apps.go](../../cli/cmd/apps.go)
- **API types:** [api/internal/argocd/types.go](../../api/internal/argocd/types.go)
- **Type collision fix:** [cli/cmd/status.go](../../cli/cmd/status.go)
- **Changelog:** [CHANGELOG.md](../../CHANGELOG.md)
