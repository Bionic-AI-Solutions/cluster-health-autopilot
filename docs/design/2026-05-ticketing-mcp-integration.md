# CHA Ticketing Integration via MCP

> **STATUS: 🚧 PARTIAL — M1 (OpenProject) SHIPPED; M2/M3/M4 NOT started.**
> _(P4.1 honest-header pass, 2026-06-11)_
>
> - **M1 — MCP-driven OpenProject sink for unfixable items: ✅ SHIPPED.** Landed via PR #59 (`ea63875 feat(ticketing): MCP-driven OpenProject sink for unfixable items (M1)`). Operator wiring (`TicketingSpec` on CR → watcher `--ticketing-*` flags) shipped Phase 1.D (PR #167, v1.20.0); chart values shape aligned in PR #170 (v1.20.1); in-cluster MCP bypasses Kong (no API-key requirement, `aeefa30`).
> - **M2 — resolve-on-clear (auto-close the ticket when the finding clears): ❌ NOT started.** Now tracked as **P6.5** in the remediation effort.
> - **M3 — Jira sink (paid): ❌ NOT started.** Now tracked as **P6.3**.
> - **M4 — ServiceNow sink (paid): ❌ NOT started.** Now tracked as **P6.4**.
>
> Body below is the original design, preserved for context.

---

**Status:** Draft
**Author:** skadam
**Date:** 2026-05-22
**Target releases:** OSS v1.7 (M1–M2), CHA-com (M3–M4)

## 1. Summary

When CHA's detect → remediate → re-probe loop identifies an issue that
**cannot be auto-fixed** — no whitelisted fixer, fixer failed, or
re-probe is still red — open a ticket in the team's issue tracker via an
**MCP server** so the item enters a durable human-intervention queue.
Today the unfixable path terminates at Slack `#ceph-critical`,
DriftReport CRs, and Alertmanager. None of those produce trackable work
items with ownership, status, or audit trail.

The integration is **OpenProject-first** (the MCP server is already
running in this cluster) with Jira and ServiceNow following in the paid
tier.

## 2. Background — current state

### 2.1 Where unfixable issues land today

`internal/report/routing.go:167-205` — `RouteAndPost()` is the single
chokepoint. After fixers run and re-probe completes, the function
partitions diagnostics into `fixed` vs `unfixable`, then fans out:

| Sink | Behavior | Lifecycle |
|---|---|---|
| Slack `#ceph-critical` | Posts unfixable summary on each cycle | Webhook fire-and-forget |
| Slack `#ceph-alerts` | Posts auto-fixed summary | Webhook fire-and-forget |
| DriftReport CR | One CR per `(subject, namespace, kind)`, upserted | Auto-deleted when drift clears |
| Alertmanager | Full active state posted every cycle | Deduped by Alertmanager labels |

There is **no notifier/sink interface** today — each output is hardcoded
into `RouteAndPost()`.

### 2.2 Registry pattern

`pkg/registry/registry.go:30-120` registers analyzers, fixers, probes at
init via `RegisterOSS()` (`catalog/catalog.go:40-79`). Sinks are NOT
part of the registry — they're configured per sink in the
`watcher.Config` struct.

### 2.3 No existing MCP usage

`grep` for `mcp`, `fastmcp`, `Model Context Protocol` across both repos
returns only test-fixture pod names. CHA is producer-only today.

## 3. Confirmed prerequisites

**OpenProject MCP server is already running** in this cluster
(verified 2026-05-22):

| Item | Value |
|---|---|
| Pod | `mcp/mcp-openproject-server-69df8b57bd-z5dcr` (uptime 13d) |
| Image | `docker.io/docker4zerocool/mcp-servers-openproject:latest` |
| In-cluster URL | `http://mcp-openproject-server.mcp.svc:8006` |
| External URL | `https://mcp.baisoln.com/openproject/{mcp,sse,messages,health}` |
| Auth (external) | Kong `mcp-key-auth` plugin |
| Backing OpenProject | `openproject.openproject.svc:8080` |
| Secret | `mcp/mcp-openproject-secrets` (keys: `openproject-url`, `openproject-api-key`) |

CHA pods will reach the MCP server **in-cluster** at
`mcp-openproject-server.mcp.svc:8006` over its `/sse` transport,
bypassing Kong key-auth (matches the existing `kong-auth-pattern`
feedback: internal traffic uses ClusterIP, not the gateway).

## 4. Design

### 4.1 Interface — `pkg/ticketing/`

```go
// pkg/ticketing/sink.go
package ticketing

import (
    "context"
    "time"

    "github.com/bionic-ai-solutions/cluster-health-autopilot/pkg/diagnose"
)

type Ticket struct {
    // Stable hash of (Subject, Namespace, Kind, ClusterName).
    // Lets the sink upsert idempotently across cycles.
    Fingerprint string
    Title       string
    Body        string             // markdown — diagnostic + runbook + cluster + ts
    Severity    diagnose.Severity  // Critical | Warning | Info
    Labels      []string
    Source      string             // "cha"
    Cluster     string
    OpenedAt    time.Time
}

type TicketRef struct {
    Provider string // "openproject" | "jira" | "servicenow"
    Key      string // WP-1287 / OPS-42 / INC0012345
    URL      string
}

type Sink interface {
    Upsert(ctx context.Context, t Ticket) (TicketRef, error)
    Resolve(ctx context.Context, ref TicketRef, reason string) error
    Comment(ctx context.Context, ref TicketRef, body string) error
}
```

**Why this shape:**

- `Upsert` (not `Create`) — idempotency is the sink's job. Implementations
  consult their backing system + the DriftReport status to decide
  create-vs-comment.
- `Resolve` — explicit close, called when DriftReport clears.
- `Comment` — used for recurrence and severity transitions without
  flooding the tracker with new tickets.
- `TicketRef` is provider-agnostic; severity → priority mapping happens
  inside each impl using Helm-provided overrides.

### 4.2 MCP client — Go-native, no sidecar

Use `github.com/mark3labs/mcp-go` (mature community Go MCP SDK). The
OpenProject implementation is a thin wrapper that:

1. Establishes one long-lived SSE session per CHA process to
   `http://mcp-openproject-server.mcp.svc:8006/sse`.
2. Discovers tools at startup; logs a warning if the expected toolset
   is missing.
3. Calls MCP `tools/call` for each `Upsert`/`Resolve`/`Comment` op.
4. Maps CHA semantics → OpenProject work-package fields:
   - `Title` → `subject`
   - `Body` → `description` (markdown)
   - `Severity` → `priority` (configurable map)
   - `Labels` → `customField:cha-labels` (or native tags if supported)
   - `Fingerprint` → `customField:cha-fingerprint` (queryable for dedup)

**Sidecar alternative rejected:** CHA already speaks HTTP natively to
Slack and Alertmanager. Adding a Python/Node bridge pod doubles the
failure surface for no benefit. The Go MCP SDK is sufficient.

### 4.3 Idempotency — store ref on DriftReport status

DriftReports already exist one-per-subject and persist across cycles
(`internal/report/driftreport.go:98-150`). Extend the CRD status:

```yaml
status:
  ticket:
    provider: openproject
    key: WP-1287
    url: https://openproject.bionicaisolutions.com/work_packages/1287
    fingerprint: sha256:abc...
    openedAt: 2026-05-22T08:00:00Z
    lastCommentedAt: 2026-05-22T16:00:00Z
```

**Flow:**

1. Unfixable diagnostic on cycle N → CHA reads `DriftReport.status.ticket`.
2. If absent → call `Upsert()` → store the returned `TicketRef` on
   status.
3. If present and N+1 cycle still unfixable → call `Comment()` only if
   the severity or message materially changed (debounce: not more than
   1 comment per `commentInterval`, default 6h).
4. When DriftReport is about to be auto-deleted (drift cleared) → call
   `Resolve(ref, "drift cleared by CHA")` first.

This is the same upsert-by-subject pattern DriftReport already uses;
ticketing just borrows the existing fingerprint.

### 4.4 Wiring into the loop

Surgical insertion in `internal/report/routing.go:RouteAndPost()`:

```go
// After the existing critical-channel Slack post, before Alertmanager:
if cfg.Ticketing.Enabled && ticketSink != nil {
    for _, d := range unfixable {
        ref, err := ticketSink.Upsert(ctx, ticketFromDiagnostic(d, cfg.Cluster))
        if err != nil {
            log.Printf("ticketing: upsert failed for %s: %v", d.Subject, err)
            continue // never abort the cycle on sink failure
        }
        driftReportClient.SetTicketRef(d.Subject, ref)
    }
}

// In the existing toResolve loop:
for _, d := range toResolve {
    if cfg.Ticketing.Enabled && cfg.Ticketing.ResolveOnClear {
        if ref := driftReportClient.GetTicketRef(d.Subject); ref != nil {
            if err := ticketSink.Resolve(ctx, *ref, "drift cleared"); err != nil {
                log.Printf("ticketing: resolve failed for %s: %v", d.Subject, err)
            }
        }
    }
}
```

**Failure posture:** ticket failures NEVER abort the cycle. Matches the
existing Slack/Alertmanager posture — observability for the operator,
not a circuit breaker for CHA itself.

### 4.5 Severity → priority mapping

Defaults — overridable via `values.yaml`:

| CHA severity | OpenProject priority | Jira priority | ServiceNow priority |
|---|---|---|---|
| `Critical` | `Immediate` | `Highest` | `1 - Critical` |
| `Warning`  | `High`      | `High`    | `2 - High`     |
| `Info`     | `Normal`    | `Medium`  | `4 - Low`      |

### 4.6 Helm chart surface

Extend `charts/cluster-health-autopilot/values.yaml`:

```yaml
ticketing:
  enabled: false
  provider: openproject           # openproject | jira | servicenow
  mcp:
    url: http://mcp-openproject-server.mcp.svc:8006/sse
    transport: sse                # sse | streamable-http
    auth:
      enabled: false              # in-cluster: skip Kong key-auth
      secretName: cha-ticketing-mcp
      secretKey: api-key          # ESO-managed when enabled
  routing:
    projectId: 1                  # OpenProject project ID
    workPackageType: "Task"
    defaultAssignee: ""
    labels: ["cha", "auto-filed"]
    severityPriority:
      Critical: Immediate
      Warning: High
      Info: Normal
  dedup:
    fingerprintFields: [subject, namespace, kind, cluster]
    commentInterval: 6h           # min spacing between recurrence comments
  resolveOnClear: true
  dryRun: false                   # log intended ops without calling MCP
```

The `auth.enabled: false` default reflects in-cluster traffic
(`mcp-openproject-server.mcp.svc` bypasses Kong, no key needed). Set
`true` only when pointing CHA at the external `mcp.baisoln.com`
endpoint.

### 4.7 Secrets — Vault → ESO → K8s

Per the project hard rule (`never-hardcode-secrets`): when external auth
is enabled, the API key flows:

```
Vault: secret/cha/ticketing/mcp-api-key
  ↓ ExternalSecret (mcp namespace or cha namespace)
K8s Secret: cha-ticketing-mcp
  ↓ env var
CHA pod: TICKETING_MCP_API_KEY
```

No hardcoded keys, no `--no-verify`, no shortcuts.

## 5. Tier placement

| Component | Tier | Repo |
|---|---|---|
| `pkg/ticketing/` interface | **OSS** | cluster-health-autopilot |
| OpenProject MCP impl | **OSS** | cluster-health-autopilot |
| DriftReport status.ticket extension | **OSS** | cluster-health-autopilot |
| Helm values + wiring | **OSS** | cluster-health-autopilot |
| Jira MCP impl | **Paid** | CHA-com |
| ServiceNow MCP impl | **Paid** | CHA-com |
| Multi-sink routing (e.g. Ceph → SN, Kong → Jira) | **Paid** | CHA-com |

Rationale: matches existing precedent — Slack/Alertmanager/DriftReport
sinks ship in OSS because basic escalation is table-stakes for a
self-hosted health autopilot. Jira and ServiceNow are the enterprise
hooks that drive the paid tier, alongside the existing AI tiers (T0–T3)
and approval-server.

## 6. Rollout phases

### M1 — Interface + OpenProject MCP (OSS v1.7)

- `pkg/ticketing/sink.go` interface
- `pkg/ticketing/openproject/` MCP-backed implementation
- DriftReport CRD extension: `status.ticket`
- Wiring into `internal/report/routing.go`
- Helm values + ExternalSecret template
- Dry-run mode (logs intended ops without calling MCP)
- E2E test: `kind` cluster + mock MCP server + fake unfixable diagnostic
  → assert ticket payload + DriftReport status set

**Acceptance:** unfixable Ceph or Kubelet diagnostic on the live cluster
produces a real work package in OpenProject, DriftReport carries the
ref, and re-running the cycle produces no duplicate.

### M2 — Lifecycle polish (OSS v1.8)

- Resolve-on-clear (auto-close ticket when DriftReport is auto-deleted)
- Comment on recurrence (with `commentInterval` debounce)
- Severity transition handling: post a comment + update priority when
  severity changes
- Backfill: scan existing unfixable DriftReports on startup and ensure
  each has a ticket

**Acceptance:** end-to-end demo: induce drift, ticket opens; clear
drift, ticket closes with comment "drift cleared by CHA on <ts>".

### M3 — Jira MCP (CHA-com)

- `pkg/ticketing/jira/` MCP-backed implementation (paid binary)
- Jira-specific fields: epic links, components, fixVersion
- Test against the official Atlassian MCP server
  (`https://mcp.atlassian.com`)
- Per-team routing rules in `values-paid.yaml`

**Acceptance:** same E2E story as M1 against a Jira sandbox project.

### M4 — ServiceNow + multi-sink routing (CHA-com)

- `pkg/ticketing/servicenow/` MCP implementation
- Routing rules engine: match diagnostic subject/labels → sink
  - example: `analyzer:ceph-*` → ServiceNow Storage queue
  - example: `analyzer:kong-*` → Jira PLATFORM project
  - example: anything else → OpenProject (OSS fallback)
- Fits the v2.0 trigger-expansion roadmap window (more triggers → more
  reason to route by domain)

**Acceptance:** a single CHA install with all three sinks configured
routes synthetic test diagnostics to the correct backend based on
labels.

## 7. Open questions

1. **DriftReport CRD breaking change?** Adding `status.ticket` is
   additive — existing consumers ignore unknown fields. No version bump
   needed, but document in v1.7 release notes.
2. **Multi-cluster ticketing.** If multiple clusters report to the same
   OpenProject project, the `cluster` field in the fingerprint already
   keeps tickets separate. Confirm with operator before M1 GA.
3. **Backoff on persistent MCP failure.** If the MCP server is down for
   N cycles, do we keep logging errors or open a circuit breaker?
   Proposal: simple per-cycle log; no breaker until M2.
4. **PII / sensitive payloads.** CHA diagnostic bodies sometimes
   include namespace names and resource names that could be sensitive.
   Add a `bodyRedactRegex` knob in M2 if any consumer asks.

## 8. Risks

- **Ticket flood on cluster-wide outage** — if 50 pods simultaneously
  go unfixable, we'd open 50 tickets. Mitigation: rely on existing
  DriftReport per-subject deduplication; one ticket per unique
  diagnostic subject.
- **MCP protocol drift** — the OpenProject MCP image is your own
  (`docker4zerocool/mcp-servers-openproject:latest`); pin a digest in
  the CHA values doc and bump deliberately.
- **External Kong dependency** — only relevant if a downstream operator
  configures the external endpoint. Default config is in-cluster.

## 9. Testing strategy

- **Unit:** mock `Sink` for the routing wiring; mock the MCP transport
  for the OpenProject impl.
- **Integration (kind cluster, OSS CI):** stand up a stub MCP server
  that records calls; verify Upsert/Resolve/Comment semantics +
  idempotency.
- **E2E (live cluster, manual gate):** induce a known unfixable
  condition (e.g., delete a ConfigMap key an analyzer expects),
  observe the work package in OpenProject, restore the condition,
  observe auto-close.

## 10. Not in scope

- Approval workflows (already covered by the paid approval-server +
  T1/T2/T3 AI tiers).
- Two-way sync (ticket-status → CHA action). Tickets are write-mostly;
  CHA owns resolution.
- Migration of existing DriftReports to tickets retroactively. M2
  backfill handles only items still active at upgrade time.
