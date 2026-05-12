# Design: Investigator Agent (Layer 2)

Status: **Draft / pre-implementation**
Tracked: planned for v1.5 or later
Sibling of: T0 Enricher, T1 Fix Proposer (CHA-com)

## Problem

CHA's probes produce one bit of information per cycle: pass or fail, with a
short error string. When a probe fails — say `https://comfy.baisoln.com:
context deadline exceeded` — the operator gets enough to know *something*
broke but not enough to triage. They have to manually run `dig`, `curl -v`,
`kubectl describe pod`, `openssl s_client`, etc., to figure out whether it's
DNS, Kong routing, an OOMKilled backend, an expired cert, or a real outage.

Layer 1 (v1.4) addressed *false-positive* alerts via retry + N-of-M streak
suppression. Layer 2 addresses *underspecified* alerts that survive Layer 1:
when an alert does fire, make it dramatically more useful by attaching the
diagnostic exploration that a human SRE would do anyway.

## Goal

When a probe failure escalates to a CRITICAL DriftReport, run an
**Investigator Agent** that issues a closed set of follow-up read-only probes
and attaches their findings to the alert. Operators open Slack and see not
just *what* failed but the most likely *why* — synthesized from the same
checks they would have run by hand.

## Non-goals

- Mutating anything. The investigator is strictly read-only.
- Replacing Layer 1. The investigator runs only on alerts that have already
  passed the streak/retry gate.
- Becoming a generic agentic shell. The action space is a fixed enum.

## Interface

```go
// In pkg/ai — sibling of Enricher and FixProposer.

// Investigator runs deep-dive read-only diagnostics on a CRITICAL finding.
// Returns a narrative + structured observations to attach to the DriftReport
// before it surfaces to operators.
type Investigator interface {
    // Investigate returns Findings the agent observed and a short summary.
    // ctx carries a hard deadline (default 20s); the implementation MUST
    // respect it and return whatever was gathered if cancelled.
    Investigate(ctx context.Context, finding probe.Finding, env Environment) (InvestigationResult, error)
}

// Environment exposes the read-only tools the agent may invoke. Closed enum
// — agent cannot exec arbitrary commands.
type Environment interface {
    DNSLookup(ctx context.Context, host string) (DNSResult, error)
    HTTPProbe(ctx context.Context, url string, opts HTTPProbeOpts) (HTTPProbeResult, error)
    TLSInspect(ctx context.Context, host string, port int) (TLSResult, error)
    KubectlDescribe(ctx context.Context, gvr, ns, name string) (string, error)
    KubectlGetEvents(ctx context.Context, ns string, since time.Duration) ([]Event, error)
    KubectlLogs(ctx context.Context, ns, pod, container string, tailLines int) (string, error)
}

type InvestigationResult struct {
    Summary      string         // 2–4 sentence narrative (LLM-synthesized)
    Observations []Observation  // structured findings (each one tool call's result)
    Cost         Cost           // tokens / wall time — for audit
}
```

Renderers display `Summary` as a `🤖 Investigation` block on the DriftReport
and the Slack message. `Observations` ship in the audit log for replay.

## Action space (closed enum)

The agent CANNOT invent commands. It selects from:

| Tool | What it does | Mutates? |
|------|--------------|----------|
| `dns_lookup(host)` | A, AAAA, CNAME, NS lookups + timing | no |
| `http_probe(url, method, expect_status)` | One HTTP request; full response headers | no |
| `tls_inspect(host:port)` | Cert chain, SANs, validity dates, issuer | no |
| `kubectl_describe(gvr, ns, name)` | Read-only `kubectl describe` output | no |
| `kubectl_get_events(ns, since)` | Recent events in a namespace | no |
| `kubectl_logs(ns, pod, container, tail)` | Last N lines of container logs | no |

Each tool has a hard timeout and a max-output budget. The LLM picks the next
tool given prior observations; the harness enforces the budget.

## Safety guardrails (mirror the existing AI tier)

| Risk | Mitigation |
|------|------------|
| Prompt injection via cluster data | All observed data wrapped in `<observed_data>` markers; tool names are pre-tokenised in the prompt schema |
| Excessive token use | Per-investigation token cap; hard wall-clock deadline (default 20s); circuit breaker after 3 consecutive over-budget cycles |
| Unbounded recursion | Max 6 tool calls per investigation; explicit halt condition (LLM must emit `<conclude>` or harness terminates) |
| RBAC overreach | Investigator runs as a dedicated ServiceAccount with read-only verbs on a narrow GVR list — strictly smaller than the watcher's ClusterRole |
| Sensitive data in narrative | Pre-output scrubber (existing audit-trail scrubber: base64≥40, hex≥32) |
| Cost runaway | Per-cluster `investigations/hour` rate limit + token bucket; soft-fail to "no investigation attached" on exceedence |

## How it composes with existing CHA tiers

```
                      ┌──────────────────────────────────────┐
                      │  Probe / Analyzer  →  Finding         │
                      └──────────────┬───────────────────────┘
                                     │
                       ┌─────────────▼─────────────┐
   v1.4 (Layer 1)     │  In-cycle retry            │
                       │  N-of-M streak suppression │
                       └─────────────┬─────────────┘
                                     │ (escalated)
                       ┌─────────────▼─────────────┐
   v1.5+ (Layer 2)    │  Investigator Agent        │  ← T1.5 — read-only
                       │  Attaches Summary +        │     between T0 and T1
                       │  Observations to Finding   │
                       └─────────────┬─────────────┘
                                     │
                       ┌─────────────▼─────────────┐
   Existing T0+       │  Enricher / FixProposer    │
                       │  (mutation tier from CHA-com)│
                       └────────────────────────────┘
```

The investigator sits **between** the probe layer and the existing T0/T1
tiers. It runs only when Layer 1 has confirmed a real outage, and its output
becomes part of the Finding/Diagnostic that subsequent AI tiers see — so T1
fix proposals get richer context for free.

## Why not just give it bash?

The temptation is to hand the LLM a shell. It would be more flexible. It
would also:

- Have unbounded action space (audit complexity explodes)
- Run arbitrary code from prompt-inject-able cluster data (a hostile pod log
  could contain instructions like "run `kubectl delete …`")
- Make RBAC scoping impossible (a single ClusterRole has to be loose enough
  for every conceivable shell command)
- Be impossible to soft-fail safely

The closed action enum is the wedge that lets CHA ship this and a customer
trust it. Same pattern the existing T1/T2 already use for mutation.

## Open questions

- Should the investigator be opt-in per-Helm-value, or on by default once it
  stabilizes? (Lean: opt-in via `ai.investigator.enabled=true`, like T1/T2.)
- LLM model choice — same provider as enricher/proposer, or smaller/cheaper?
  (Lean: configurable; default to whatever T0 enricher uses for consistency.)
- Where do investigation results persist? In the DriftReport CR (bloats it)
  or in a sibling `Investigation` CR with a reference? (Lean: sibling CR;
  size-bounded DriftReports keep `kubectl get driftreports` readable.)
- Should investigations run in parallel for multiple concurrent CRITICALs?
  (Lean: per-finding worker pool with a global concurrency cap of 2 initially.)

## Implementation milestones

1. Define `pkg/ai.Investigator` interface + `Environment` + tool result types.
2. Implement the harness (action-loop with tool budget and timeout).
3. Wire one tool (`dns_lookup`) end-to-end as a tracer.
4. Add the closed-enum prompt template in `ai/prompts/system_investigator.md`.
5. Wire to watcher post-probe path: when a finding escalates, kick off an
   investigation, attach result to the Finding before DriftReport creation.
6. Helm: `ai.investigator.{enabled,maxToolCalls,maxTokens,timeoutSeconds}`.
7. Audit trail: emit `ai.investigation.started`, `ai.investigation.tool_call`,
   `ai.investigation.completed`, `ai.investigation.budget_exceeded` events.
8. Reference customer flag-and-flip rollout: dry-run first (investigation
   runs but result is logged only, not posted), then live.

## Out of scope (later if needed)

- Agent learning / fine-tuning from past investigations
- Multi-cluster correlation ("this same pattern fired in staging last week")
- Auto-issue-creation in GitHub / Linear from investigation findings

These belong in a v2+ phase once the single-cluster, single-finding case is
solid.
