# Completing the Compliance Triad: Building a Runtime Security Events Dashboard

**Date:** February 23, 2026
**Author:** Rod Stewart (with Claude Sonnet 4.5)
**Project:** AKS Home Lab Internal Developer Platform
**Task:** Implementing Dashboard Panel #6 of 6 - Security Events (#84) — Core Dashboard Complete

## The Milestone

This post marks a milestone: the sixth and final core dashboard panel for the Homelab Platform Portal UI. With Security Events complete, the platform now has full-stack security observability across three distinct threat dimensions:

| Panel | Source | Detects | When |
|-------|--------|---------|------|
| Policy Violations | Gatekeeper (OPA) | Misconfigurations | At admission time |
| Vulnerability Feed | Trivy Operator | Known CVEs | At scan time |
| **Security Events** | **Falco** | **Suspicious behavior** | **At runtime** |

This is the **compliance monitoring triad** — and the Security Events panel completes it by surfacing the hardest-to-detect category: things that are happening *right now* in your cluster.

## Why Runtime Events Are Different

Static analysis (Gatekeeper) and vulnerability scanning (Trivy) catch problems before or shortly after deployment. Runtime security is fundamentally different — it detects behaviors that can't be predicted from manifests or images:

- A developer `exec`s a shell into a production container
- A compromised process reads `/etc/shadow`
- A container writes a binary to its filesystem (container drift)
- An unexpected outbound connection reaches a known mining pool port

These are **active threat indicators**, which is why they carry heavier compliance score penalties: Critical events deduct 15 points each (vs. 10 for a Critical CVE) and Error events deduct 8 points (vs. 5 for a High CVE). The scoring reflects a simple reality: a vulnerability *might* be exploited; a Falco alert means something already happened.

## The Type Mismatch Pattern (Third Time's the Charm)

By this point, finding speculative TypeScript types has become a recurring theme. This is the third consecutive panel where we fixed type mismatches before implementation.

### What Was Wrong

```typescript
// WRONG (speculative, written before Go API existed)
export interface SecurityEvent {
  timestamp: string;
  rule: string;
  priority: string;     // ❌ Go struct uses "severity"
  message: string;
  source: string;       // ❌ Go struct uses "resource"
  tags: string[];       // ❌ Doesn't exist in API
  output: string;       // ❌ Doesn't exist in API
  outputFields: Record<string, unknown>;  // ❌ Doesn't exist
  hostname: string;     // ❌ Doesn't exist
}

export interface ListSecurityEventsResponse {
  events: SecurityEvent[];
  count: number;        // ❌ API doesn't return count
}
```

### What the Go API Actually Returns

```go
// api/internal/compliance/types.go (lines 60-71)
type EventsResponse struct {
    Events []SecurityEvent `json:"events"`
}

type SecurityEvent struct {
    Timestamp string `json:"timestamp"`
    Rule      string `json:"rule"`
    Severity  string `json:"severity"`
    Message   string `json:"message"`
    Resource  string `json:"resource,omitempty"`
}
```

Five fields needed correction. Five phantom fields needed removal. One response wrapper field (`count`) didn't exist.

### The Systemic Issue

Looking across all three type-fix sessions:

| Panel | Wrong Fields | Phantom Fields | Root Cause |
|-------|-------------|----------------|------------|
| Vulnerability Feed | 6 | 3 | Speculative types from assumed API shape |
| Security Events | 2 | 5 | Same: wrote TS types before Go implementation |

The pattern is clear: **every TypeScript interface written before the Go API was implemented contained errors**. Not some of them — all of them. The lesson has solidified into a project rule documented in `CLAUDE.md`:

> **MANDATORY RULE:** TypeScript types in `portal/src/api/types.ts` MUST match the Go API struct JSON tags exactly.

With a verification checklist and a known type mappings table that now includes:

| Endpoint | Go Response | Key Corrections |
|----------|-------------|-----------------|
| `GET /api/v1/compliance/events` | `EventsResponse` | `severity` (NOT `priority`), `resource` (NOT `source`), no `count` field |

## Four-Tier Severity Mapping

Falco uses a richer severity model than Trivy. Trivy has 5 levels (CRITICAL through UNKNOWN). Falco has 8 priority levels borrowed from syslog. We mapped these to four visual tiers:

```typescript
function getSeverityVariant(severity: string): 'danger' | 'warning' | 'info' | 'default' {
  const upperSeverity = severity.toUpperCase();
  if (upperSeverity === 'CRITICAL' || upperSeverity === 'ALERT' || upperSeverity === 'EMERGENCY') {
    return 'danger';    // Red — active exploitation indicators
  }
  if (upperSeverity === 'ERROR') {
    return 'warning';   // Yellow — policy violations at runtime
  }
  if (upperSeverity === 'WARNING' || upperSeverity === 'NOTICE') {
    return 'info';      // Blue — suspicious but possibly legitimate
  }
  return 'default';     // Gray — informational/debug noise
}
```

The four tiers map to the incident response mental model:
- **Red (danger):** Drop everything. Something is actively compromised.
- **Yellow (warning):** Investigate soon. A rule fired that shouldn't in normal operation.
- **Blue (info):** Review when convenient. Could be normal behavior that looks suspicious.
- **Gray (default):** Background noise. Useful for forensics but not actionable.

This differs from the Vulnerability Feed's three-tier model (danger/warning/default) because runtime events carry more nuance — a WARNING from Falco ("shell spawned in container") might be a developer debugging, while an ERROR ("sensitive file read in container") almost certainly isn't normal.

## Timeline vs. Table: Choosing the Right Layout

The five preceding panels used two layout patterns:
- **Card grid:** Applications, Infrastructure (entity-centric, show current state)
- **Scrollable table:** Compliance Score, Policy Violations, Vulnerability Feed (list-centric, show collections)

Security Events could have gone either way, but the timeline nature of the data made the table pattern the obvious choice. Events are inherently temporal — "what happened, when, in what order" matters more than any spatial grouping.

The key layout decision: **five columns tuned for security triage.**

| Column | Width | Why |
|--------|-------|-----|
| Timestamp | 140px | "When" — formatted to human-readable ("Feb 23, 2:30 PM") |
| Severity | 100px | "How bad" — color-coded badge for instant triage |
| Rule | 180px | "What fired" — identifies the detection pattern |
| Resource | 150px | "Where" — namespace/pod path in monospace |
| Message | flex | "Details" — truncated to 100 chars with hover tooltip |

The timestamp column was the most debated. Raw RFC3339 (`2026-02-23T14:30:00Z`) is precise but unreadable. We chose locale-aware formatting that drops the year and seconds:

```typescript
function formatTimestamp(timestamp: string): string {
  try {
    const date = new Date(timestamp);
    return date.toLocaleString('en-US', {
      month: 'short', day: 'numeric',
      hour: 'numeric', minute: '2-digit', hour12: true,
    });
  } catch {
    return timestamp;  // Degrade gracefully to raw value
  }
}
```

## The Empty State Matters More Here

For Vulnerability Feed, the empty state was celebratory: "All scanned images are free of known CVEs." For Security Events, the empty state carries a different message:

> ✓ No security events detected
> Falco is monitoring runtime activity across all namespaces

The second line is important. "No events" could mean:
1. Nothing suspicious happened (good)
2. Falco isn't running (bad)
3. The webhook pipeline is broken (bad)

By mentioning Falco explicitly, we reassure the user that the monitoring system is active — the absence of events is a positive signal, not a missing data problem.

## The Full Event Pipeline

One thing that makes this panel interesting architecturally is the length of its data pipeline:

```
Kernel syscall
  → Falco eBPF probe (DaemonSet, every node)
  → Falco rule engine (evaluates against 4 custom + default rules)
  → HTTP output to Falcosidekick
  → Falcosidekick webhook to Platform API
  → POST /api/v1/webhooks/falco (receiver)
  → EventStore (in-memory circular buffer, 1000 events)
  → GET /api/v1/compliance/events (query endpoint)
  → TanStack Query (30s polling)
  → SecurityEventsPanel (render)
```

That's **eight hops** from kernel to pixel. Each one is a potential failure point, and each one was built in a separate task over the course of weeks. The panel itself is just the last mile — but without it, all the upstream infrastructure produces data that nobody sees.

This is a common pattern in platform engineering: **the last 10% of the feature (the UI) makes the first 90% (the infrastructure) actually useful.**

## Consistency as a Velocity Multiplier

The SecurityEventsPanel implementation took roughly 20 minutes of actual coding. Not because it's trivial, but because five panels before it established every pattern:

- TanStack Query setup (queryKey, queryFn, refetchInterval)
- StatusCard wrapper with title
- Loading/error/empty state trio
- Scrollable table with sticky header
- Badge severity mapping
- Footer summary

By the sixth panel, the "creative" decisions were gone. The component is effectively a configuration of established patterns with domain-specific column definitions.

This is the **compound interest of consistency**: each panel that follows the pattern is faster to build and easier to review. The first panel (Applications) took the longest because it established the patterns. By panel six, we're just filling in a template.

## What's Complete, What's Next

### Dashboard Status: 6 of 6 Core Panels Complete

1. ✅ Applications — Argo CD sync status, health, deployments
2. ✅ Infrastructure — Crossplane Claims, connection secrets
3. ✅ Compliance Score — Donut chart with aggregated scoring
4. ✅ Policy Violations — Gatekeeper audit failures
5. ✅ Vulnerability Feed — Trivy CVE scans
6. ✅ **Security Events** — Falco runtime alerts

### Remaining Portal Work

- **Scaffold Form (#85):** Interactive project creation with template selector, storage/vault toggles
- **AI Ops Panel (#86):** kagent chat interface + HolmesGPT investigation triggers
- **Detail Pages:** Drill-down views for individual apps, infra claims, compliance items

The core dashboard is the foundation. Everything else builds on top of it — the scaffold form creates things the dashboard monitors, and the AI ops panel helps investigate what the dashboard surfaces.

## Lessons Reinforced

### 1. The Type Mismatch Is a Systemic Problem, Not a One-Off

Three panels. Three type fixes. Same root cause every time. The project now has a mandatory verification checklist and a growing type mappings table. The process works, but the fact that we needed it three times suggests **the types should have been generated from Go structs** rather than hand-written. Future consideration: `go-typescript` or similar code generation.

### 2. Severity Models Vary by Source

Trivy uses 5 levels. Falco uses 8. Gatekeeper violations have no severity at all (they either violate or they don't). When building a unified compliance dashboard, you need a **mapping layer** that translates source-specific semantics into consistent visual language. The four-tier Badge system (`danger`, `warning`, `info`, `default`) serves as that abstraction.

### 3. The Last Mile Makes the Platform

Falco was deployed weeks ago. The webhook pipeline has been working for days. But until the Security Events panel shipped, **no human could see any of it** without running `kubectl` commands. The dashboard panel isn't technically complex, but it's the difference between "we have runtime security" and "we can *see* our runtime security."

### 4. Empty States Are Communication Design

In security monitoring, what you show when nothing is happening matters as much as what you show when something is. A blank panel is ambiguous. A positive empty state ("No events detected — Falco is monitoring") is reassuring. The small copy difference has a big impact on user confidence.

---

## References

- **Implementation:** [SecurityEventsPanel.tsx](../../portal/src/components/dashboard/SecurityEventsPanel.tsx)
- **Type definitions:** [types.ts](../../portal/src/api/types.ts)
- **Backend types:** [compliance/types.go](../../api/internal/compliance/types.go)
- **Event handler:** [compliance/handler.go](../../api/internal/compliance/handler.go) (lines 446-489)
- **Falco webhook:** [webhooks/falco.go](../../api/internal/webhooks/falco.go)
- **Changelog:** [CHANGELOG.md](../../CHANGELOG.md)
- **Previous post:** [Vulnerability Feed Implementation](./2026-02-23-vulnerability-feed-implementation.md)
- **Live deployment:** https://portal.rdp.azurelaboratory.com
