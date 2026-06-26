# AI Audit Trail — Event Schema and Query Guide

Every AI-related operation in Srenix Enterprise v1.0.0 emits a structured audit
event. This doc is the schema reference + query cookbook for SOC 2 /
ISO 27001 compliance reviews and incident response.

**Companion docs**: [AI_TIERS.md](AI_TIERS.md), [THREAT_MODEL_AI.md](THREAT_MODEL_AI.md)

---

## Event lifecycle

A single approved fix produces this audit chain:

```
ai.llm.call              (enricher or proposer requested narration / proposal)
  └─ ai.proposal.created (or ai.enrichment.applied for T0)
       └─ ai.approval.granted  (SRE clicked + JWT verified + replay-checked)
            └─ ai.action.applied  (or ai.action.failed)
```

T2 plans add `ai.plan.created` between proposal and approval. T3
runbooks add `ai.runbook.created` and `ai.runbook.approval_recorded`
(twice: slot 1, slot 2). All events share a `correlation_id` =
ActionID/PlanID/RunbookID for trace linkage.

**Layer-2 Investigator audit events** (paid LLM-backed implementation
only; the OSS rule-based investigator deliberately emits no audit
events to keep its zero-dependency posture):

```
ai.investigator.started      (investigation entered the cycle)
  ├─ ai.investigator.tool_call    (one per Environment method invoked;
  │                                details.tool = "describe" | "events" |
  │                                "firecrawl" | "citation" | ...)
  ├─ ai.investigator.tool_call    ...
  └─ ai.investigator.completed    (or ai.investigator.budget_exceeded)
```

A **Firecrawl web-research call** produces an `ai.investigator.tool_call`
event with `details.tool = "firecrawl"`. The query sent to Firecrawl is
the synthesized, client-redacted string — raw cluster identifiers never
appear in this event.

These events share a `correlation_id` = the DriftReport name being
investigated, so a full per-report trace links cleanly to the enrichment,
proposal, approval, and action chains that may follow.

**Ticket-closure recording** (v0.2.0-alpha.1): when a Jira/ServiceNow ticket is
resolved (finding cleared), an `ai.memory.recorded` event is emitted with
`details.verdict = "cleared"` and `details.delivery = "ticket-closed"`.
This is best-effort; a recording failure emits `ai.memory.record_failed`
but never affects the ticket resolution response.

---

## Event types

| Type | Tier | Severity | When emitted |
|---|---|---|---|
| `ai.llm.call` | any | Normal | Before LLM round-trip |
| `ai.enrichment.applied` | T0 | Normal | T0 narrative successfully applied to a diagnostic |
| `ai.enrichment.failed` | T0 | Warning | LLM endpoint unreachable; deterministic flow continues |
| `ai.enrichment.invalid` | T0 | Warning | LLM response malformed or validator-rejected |
| `ai.proposal.created` | T1+ | Normal | Proposer emitted a valid AIProposedAction |
| `ai.proposal.failed` | T1+ | Warning | LLM call failed during proposal |
| `ai.proposal.refused` | T1+ | Normal | LLM returned `{refuse: "..."}` |
| `ai.proposal.invalid` | T1+ | Warning | Proposal rejected by validator |
| `ai.plan.created` | T2 | Normal | Multi-step plan generated and validated |
| `ai.plan.failed` | T2 | Warning | Plan generation failed |
| `ai.plan.invalid` | T2 | Warning | One or more steps failed validation |
| `ai.runbook.created` | T3 | Normal | Vault recovery runbook generated |
| `ai.runbook.rejected` | T3 | Warning | Runbook violated path allowlist / secret-value heuristics |
| `ai.runbook.invalid` | T3 | Warning | LLM response unparseable |
| `ai.runbook.approval_recorded` | T3 | Normal | One slot of dual approval recorded |
| `ai.runbook.approval_rejected` | T3 | Warning | Same-approver or too-early rejection |
| `ai.approval.granted` | T1+ | Normal | JWT verified, approver identity recorded |
| `ai.approval.rejected` | T1+ | Warning | Token failed verification (signature/expiry/replay) |
| `ai.action.applied` | T1+ | Normal | Mutator call succeeded; includes post_apply_verified |
| `ai.action.failed` | T1+ | Warning | Mutation failed at admission or apply |
| `ai.rate_limited` | T1+ | Normal | Rate limiter denied a proposal |
| `ai.circuit_breaker.tripped` | T1+ | Warning | Auto-disable after N failures |
| `ai.circuit_breaker.reset` | T1+ | Normal | Counter reset (success or manual) |
| `ai.investigator.started` | L2 (paid) | Normal | LLM-backed investigation began for a DriftReport |
| `ai.investigator.tool_call` | L2 (paid) | Normal | One `Environment` method invoked; `details.tool` names which (e.g. `describe`, `events`, `firecrawl`, `citation`) |
| `ai.investigator.completed` | L2 (paid) | Normal | Summary attached to DriftReport |
| `ai.investigator.budget_exceeded` | L2 (paid) | Warning | Per-cycle 20s cap or per-investigation token budget exhausted |
| `ai.memory.recorded` | T1+ / L2 (paid) | Normal | Outcome (proposal result or ticket closure) persisted to RAG; `details.delivery` = `human-approved` \| `autonomy` \| `human-denied` \| `ticket-closed` |
| `ai.memory.record_failed` | T1+ / L2 (paid) | Warning | RAG write failed (best-effort; does not affect the operation that triggered it) |

The OSS rule-based investigator emits no audit events. To audit it,
read the DriftReport CR's `spec.investigation` field directly (the
investigator's only externally observable output).

---

## Event payload schema

```json
{
  "type": "ai.action.applied",
  "correlation_id": "act-a3f0b1c2d3e4",
  "tier": "t1",
  "actor": "approval-server",
  "subject": "Pod/default/demo-abc",
  "details": {
    "approver": "alice@example.com",
    "source_ip": "10.0.5.42",
    "target": "Pod/default/demo-abc",
    "action": "DeletePod",
    "post_apply_verified": true,
    "diff_summary": "Applied DeletePod on Pod/default/demo-abc"
  }
}
```

Kubernetes Events sink (default) maps:
- `type` → Event `reason` (CamelCase: `AIActionApplied`)
- `tier` → annotation `srenix.ai/audit-tier`
- `actor` → annotation `srenix.ai/audit-actor`
- `correlation_id` → annotation `srenix.ai/audit-correlation-id`
- `details` → annotation `srenix.ai/audit-details` (JSON string)

---

## Query examples

### Recent AI events (default Kubernetes Events sink)

```sh
kubectl -n agentic-sre get events --sort-by=lastTimestamp \
  | grep -E "AI(Enrichment|Proposal|Approval|Action|Runbook|RateLimited|CircuitBreaker)"
```

### Filter by tier

```sh
kubectl -n agentic-sre get events -o json | \
  jq '.items[] | select(.metadata.annotations."srenix.ai/audit-tier" == "t1")'
```

### Trace a single approval chain (all events for one correlation_id)

```sh
CID=act-a3f0b1c2d3e4
kubectl -n agentic-sre get events -o json | \
  jq --arg cid "$CID" '.items[] | select(.metadata.annotations."srenix.ai/audit-correlation-id" == $cid)
                                  | {time: .lastTimestamp, reason: .reason, message: .message}'
```

### Trace a single Layer-2 investigation

```sh
# Paid LLM-backed investigator only — rule-based emits none.
# correlation_id is the DriftReport name.
CID=tls-mismatch-prod-shop-2026-05-12
kubectl -n agentic-sre get events -o json | \
  jq --arg cid "$CID" '.items[] | select(.reason | startswith("AIInvestigator"))
                                  | select(.metadata.annotations."srenix.ai/audit-correlation-id" == $cid)
                                  | {time: .lastTimestamp, reason: .reason, message: .message}'

# OSS rule-based investigator audit — read the DriftReport directly
kubectl -n agentic-sre get driftreport "$CID" \
  -o jsonpath='{.spec.investigation}' && echo
```

### Loki LogQL (when LokiSink configured)

```logql
# All approval grants in the last 24h
{job="srenix-ai", event_type="ai.approval.granted"} | json | __error__=""

# Failures by approver
{job="srenix-ai", event_type="ai.action.failed"} | json | line_format "{{.approver}}: {{.reason}}"

# Circuit breaker trips
{job="srenix-ai", event_type=~"ai\\.circuit_breaker\\..*"}

# Firecrawl web-research calls (deep-RCA investigator)
{job="srenix-ai", event_type="ai.investigator.tool_call"} | json | details_tool="firecrawl"

# Ticket-closure outcomes persisted to RAG
{job="srenix-ai", event_type="ai.memory.recorded"} | json | details_delivery="ticket-closed"
```

---

## Compliance evidence packages

### SOC 2 CC7.2 (anomaly detection)

```sh
# Circuit-breaker trip events in the audit period
kubectl -n agentic-sre get events --field-selector reason=AICircuitBreakerTripped \
  --output-version=v1 -o yaml > soc2-cc7.2-circuit-breaker-events.yaml
```

### SOC 2 CC7.3 (security incident handling)

For each incident, gather the full correlation chain:

```sh
# Given an incident's ActionID, dump every event that referenced it
ACTION_ID=act-XXXX
kubectl -n agentic-sre get events -o json | \
  jq --arg id "$ACTION_ID" '[.items[] | select(.metadata.annotations."srenix.ai/audit-correlation-id" == $id)]' \
  > incident-${ACTION_ID}.json
```

### ISO 27001 A.12.4 (logging)

For long-term retention, point Srenix Enterprise at a Loki/OTLP sink (see
SETUP_GUIDE.md §14.9). The Kubernetes Events sink is for short-term
in-cluster review only.
