# AI Tiers — Definitive Specification (v1.5.x)

This document is the single source of truth for the AI tiers shipped in
CHA-com. The tier family is **T0 → T1 → T2 → T3**, with a sibling
**Layer-2 Investigator** tier that ships in OSS as a deterministic
rule-based implementation and in CHA-com as an LLM-backed override.
Operators reference this when sizing budgets, picking a deployment
posture, or reasoning about blast radius.

**Companion docs**:
- [AI_USAGE.md](AI_USAGE.md) — why we have AI tiers (positioning)
- [THREAT_MODEL_AI.md](THREAT_MODEL_AI.md) — OWASP LLM Top 10 mapping
- [ADVERSARIAL_ANALYSIS.md §8](ADVERSARIAL_ANALYSIS.md#8-ai-tier-attack-surface) — security review of the AI surface
- [SETUP_GUIDE.md §14](SETUP_GUIDE.md#14-ai-tier-setup) — installation walkthrough
- [design/2026-05-investigator-agent.md](design/2026-05-investigator-agent.md) — Layer-2 design document
- [design/2026-06-19-firecrawl-rca-tier-chaining-design.md](design/2026-06-19-firecrawl-rca-tier-chaining-design.md) — deep-RCA + Firecrawl + tier-chaining increment

---

## The cardinal rule: agency is always human-gated

Across all four tiers, the architecture preserves this invariant:

**AI proposes — humans approve — deterministic Go code executes.**

The OSS engine's Mutator interface is never called directly from an LLM
response. Every mutation passes through:

1. **LLM** produces a structured proposal (closed-enum action_kind +
   target + rollback).
2. **Validator** rejects anything outside the whitelist (incl.
   protected-namespace targets).
3. **Signer** issues a short-lived JWT bound to that specific proposal.
4. **Human** clicks the signed URL from Slack/Alertmanager.
5. **Approval-server** re-verifies signature, expiry, one-time-use,
   approver identity (OIDC).
6. **Executor** re-runs admission policy (defense in depth).
7. **OPA/Gatekeeper** (optional third gate) independently validates.
8. **snapshot.Mutator** applies the mutation — same code path that
   today's `cha remediate --live` uses.

Higher tiers grow *coverage* (what kinds of issues the AI can analyze
and propose), not *autonomy*.

Layer-2 is a special case: it is **read-only by construction** (the
Environment interface is a closed set of read-only tools) and therefore
sits outside the proposal/approval/execution chain. It runs whether or
not any other tier is enabled, and the rule-based implementation that
ships in OSS uses no LLM at all.

---

## Tier matrix

| Property | T0 Narration | **L2 Investigator** | T1 Single fix | T2 Multi-step | T3 Vault runbook |
|---|---|---|---|---|---|
| **What ships** | Slack/AM `🤖` enrichment block | Slack/AM `🔬` investigation block | "Apply Fix" button | Step-by-step approval | Dual-approval runbook |
| **LLM input** | Diagnostic (redacted) | OSS: none. Paid: Finding/Diagnostic + tool transcripts (redacted) | + matching fixer name | + cross-resource context | + ESO refs |
| **LLM output** | EnrichedDiagnostic JSON | OSS: none (rule-driven). Paid: tool selections → InvestigationResult | AIProposedAction | up to 5 sequential actions | VaultRunbook |
| **Mutation surface** | None | **None** (Environment is closed read-only enum) | OSS whitelist (5 verbs) | Same as T1 | Zero — runbook is human-run |
| **Approval gate** | n/a — read-only | n/a — read-only | One-click signed URL | One-click per step | **Dual** (2 distinct approvers, 30-min audit window) |
| **Click TTL** | n/a | n/a | 15 min | 15 min | 90 min |
| **Replay protection** | n/a | n/a | JTI one-time-use | JTI one-time-use | JTI one-time-use |
| **Protected NS blocked** | n/a | n/a (read-only) | LLM + validator + admission | Per-step | n/a — Vault path allowlist |
| **Rollback required** | n/a | n/a | Yes | Per-step | Manual runbook step |
| **Post-apply verify** | n/a | n/a | 60s window | Per-step gate to next | n/a |
| **Audit events** | `ai.enrichment.*` | OSS: none. Paid: `ai.investigator.*` | + `ai.proposal.*`, `ai.approval.*`, `ai.action.*` | + `ai.plan.*` | + `ai.runbook.*` |
| **New RBAC** | none | **none** (reuses watcher's existing snapshot read-access) | none (reuses remediator) | none | none |
| **Default** | off | **on in OSS (rule-based)**; paid override off by default | off | off | off |
| **Risk class** | Privacy (sends diagnostic JSON to LLM) | OSS: none. Paid: same as T0 (redacted diagnostic + tool transcripts) | Mutation (with approval) | Mutation × N | Vault knowledge leak (key NAMES only) |

---

## Per-tier specifications

### T0 — Read-only narration

**Capability**: Adds a 2–4 sentence LLM-generated root-cause narrative
to every Diagnostic. Renders as a `🤖 _{enrichment}_` block under each
issue in Slack/Alertmanager and as the `ai_enrichment` annotation on
Alertmanager alerts. DriftReport CRs gain an `.status.enrichment` field.

**Inputs to LLM**:
- Structured `Diagnostic` (after [redact.go](../pkg/ai/redact.go)):
  - `subject` with namespace/name SHA-256-hashed
  - `severity`, `source` (analyzer name) preserved as enum-like signal
  - `message` with IPs → class labels, UIDs → `<uid>`, internal hostnames
    hashed, cluster domain → `<cluster>`
  - `remediation` redacted same as `message`
- **NEVER sent**: raw event messages, pod logs, Secret bytes, Vault values,
  pre-redaction identifiers, kubeconfig material

**Output schema** (`pkg/ai.EnrichedDiagnostic`):
```json
{
  "enrichment": "<2-4 sentence narrative, ≤500 chars>",
  "related_signals": ["<optional follow-up command>", "..."]
}
```

**Failure modes (handled gracefully)**:
- LLM endpoint unreachable → deterministic diagnostic continues; no
  `🤖` block emitted; `AIEnrichmentFailed` event recorded
- Malformed JSON → drop, log to audit
- Over-length response → truncate to `MaxEnrichmentChars=500`
- Rate-limit hit → serve cached response if subject matches; else drop

**Rate-limit profile**: Each cycle the watcher enriches all current
diagnostics. At ~30 issues × ~1KB prompt = ~30KB egress per cycle. At
the default 10-min resync, that's ~180KB/hour to the LLM. Recommended
budget: 1M tokens/hour.

**Operator decision**: T0 is the lowest-risk tier. It can be enabled
in any cluster that has an LLM endpoint reachable and doesn't have
strict data-residency constraints on the redacted Diagnostic payload.

### Layer-2 — Read-only Investigator

**Capability**: When a Finding or Diagnostic escalates to
`SeverityCritical`, run a follow-up investigation that exercises a
fixed set of read-only tools (DNS lookup, HTTP probe, TLS inspection,
resource describe, recent events) and attach a one-to-four-sentence
narrative summary plus the tool transcripts to the alert. Renders as a
`🔬 _{summary}_` block under each issue in Slack/Alertmanager,
populates the DriftReport CR's `spec.investigation` field, and feeds
downstream T1 prompts when those tiers are enabled.

Layer-2 is the sibling of T0: both are read-only and additive. The
difference is that T0 asks an LLM to *narrate* what the diagnostic
already says; Layer-2 *gathers fresh evidence* via deterministic tools
and synthesizes a conclusion. The OSS implementation is rule-based
(no LLM); the paid CHA-com binary replaces it with an LLM-backed
**deep-RCA investigator** that extends the closed Environment surface
with optional Firecrawl web research.

**CHA-com deep-RCA investigator (v0.2.0-alpha.1, paid)**: The
`ai/investigator.go:LLMInvestigator` replaces the rule-based
investigator when the AI tier is enabled. It:

1. Runs the same read-only cluster tools (`Describe` + `GetEvents` via
   `pkg/ai.Environment`) against the live cluster.
2. Synthesizes a **generic, client-redacted web search query** via an
   LLM call — namespace, pod name, host, IP, and secret identifiers are
   stripped before the query leaves the cluster. `RedactDiagnostic` and
   `RedactEventMessage` provide a regex backstop. Environment observation
   arguments are populated from the REDACTED subject so raw identifiers
   never reach the RCA LLM prompt.
3. Calls **Firecrawl** (opt-in, requires `FIRECRAWL_API_KEY`) to search
   and scrape relevant documentation and runbooks. This is the **one
   deliberate exception** to "payload never leaves the cluster" — see
   [THREAT_MODEL_AI.md](THREAT_MODEL_AI.md) for the egress threat model.
4. Synthesizes a root-cause analysis (summary + citations) from cluster
   tool outputs + web results.
5. Persists the RCA to the **`cha_investigations` Qdrant collection** for
   cross-cycle retrieval.
6. Forwards the RCA into every downstream AI tier via a `<root_cause>`
   prompt block: T0 enricher, T1 proposer, T2 planner, and T3 runbook
   all receive the investigation result when present. T1 also receives
   prior-cycle RCAs for the same finding class (via a
   `<prior_investigation>` block) and the T0 enrichment.

**Flags** (CHA-com binary):

| Flag | Default | Notes |
|---|---|---|
| `--firecrawl-endpoint` | `https://api.firecrawl.dev` | Firecrawl API base URL |
| `--firecrawl-enabled` | `true` | Inert without the key env set |
| `--firecrawl-api-key-env` | `FIRECRAWL_API_KEY` | Name of the env var holding the API key |
| `--investigator-web-timeout` | `8s` | Per-request wall-clock cap for the Firecrawl call |

**ESO setup** (Firecrawl key):
```yaml
# ExternalSecret cha-firecrawl-key
# ClusterSecretStore: vault-backend
# Vault path: secret/data/shared/api-keys  key: firecrawl_api_key
# Produces:  K8s Secret cha-firecrawl-key  key: FIRECRAWL_API_KEY
```
See [AI_OPERATOR_RUNBOOK.md](AI_OPERATOR_RUNBOOK.md) for the full
deployment procedure.

Full design rationale: [design/2026-05-investigator-agent.md](design/2026-05-investigator-agent.md).

**Cardinal rule**: the Environment interface is closed. An Investigator
cannot exec arbitrary commands, write to the cluster, or escape the
read-only tool set. This holds equally for the rule-based and the
LLM-backed implementations.

**Environment** (`pkg/ai.Environment`):
```go
type Environment interface {
    DNSLookup(ctx, host) (DNSResult, error)
    HTTPProbe(ctx, url, opts) (HTTPProbeResult, error)
    TLSInspect(ctx, host, port) (TLSResult, error)
    Describe(ctx, kind, namespace, name) (DescribeResult, error)
    GetEvents(ctx, namespace, kind, name, since) ([]EventInfo, error)
}
```

The concrete `LiveEnvironment` in `internal/investigator/env_live.go`
uses the net stdlib and the watcher's existing `snapshot.Source` —
**no new RBAC is required**; investigation reuses the watcher's
existing read access.

**Inputs** to a rule-based Investigator (no LLM):
- A `probe.Finding` (component, severity, message) **or** a
  `diagnose.Diagnostic` (subject, source, severity, message,
  remediation).
- The `Environment` instance the watcher constructed for this cycle.

**Inputs** to an LLM-backed Investigator (paid):
- Same as above, redacted identically to T0.
- Tool transcripts accumulated during the investigation (also redacted).
- The closed `Environment` action enum, advertised via the prompt
  schema so the model cannot fabricate tool names.

**Output schema** (`pkg/ai.InvestigationResult`):
```json
{
  "summary": "<≤800 chars; capped at MaxInvestigationSummaryChars>",
  "observations": [
    {"tool": "TLSInspect", "args": "...", "result": "...", "elapsed_ms": "..."}
  ],
  "conclusion": "confirmed_outage" | "likely_transient" | "root_cause_identified" | "insufficient_data" | "",
  "cost": {"wall_ms": "...", "tool_calls": "...", "tokens_in": "...", "tokens_out": "..."}
}
```

**Rule coverage** (OSS `internal/investigator/rules.go`):
- TLS verification (cert expiry, SAN mismatch).
- Connection failure (DNS lookup + HTTP probe + insecure-skip-verify
  fallback to distinguish TLS errors from outright unreachability).
- HTTP status mismatch (probe + recent events on the backend).
- Slow-DNS classification (DNS roundtrip > 1.5s flagged as a likely
  resolver issue).
- Diagnostic patterns: `ExternalSecret`, `Secret` missing /
  missing-key, `Certificate` expiry — each calls `Describe` and
  `GetEvents` on the named resource.

**Failure modes (handled gracefully)**:
- Pattern not matched → returns the zero `InvestigationResult`; the
  finding flows through unchanged (no `🔬` block emitted).
- Per-cycle wall-clock cap (default 20s) reached → remaining items
  skipped, completed items still attached.
- Single-item failure → soft-fails for that item only; the rest of
  the cycle proceeds.
- Re-diagnose after fixers runs **before** investigation, so a fixer
  that resolves the underlying issue cleans up the investigation
  surface naturally.

**Rate-limit profile**:
- Rule-based: zero token spend; ~50–500 ms wall-time per item;
  bounded by the 20s per-cycle ceiling.
- LLM-backed (paid): ~2 KB input, ~500 B output per investigation;
  runs only on critical findings (typically <5/cycle). Default budget
  of 5 investigations/hour aligns with the paid-tier rate limit.

**Operator decision**: Layer-2 (rule-based) is on by default in OSS
and is safe in every environment — there is no new RBAC, no network
egress beyond the existing probes, and no token cost. Disable with
`CHA_INVESTIGATOR=off` only if you want bit-identical output to the
pre-v1.5 binary. The paid LLM-backed deep-RCA investigator is opt-in
and lands on the same DriftReport field, the same Slack block, and the
same audit-event taxonomy as the rule-based version — so swapping is a
configuration change, not a re-integration. Firecrawl egress is
explicitly opt-in (key env must be set); without the key, web research
is silently skipped and the investigator falls back to cluster-only
evidence.

### T1 — Approved deterministic fix

**Capability**: When a diagnostic matches an existing OSS whitelisted
fixer, surface an `Apply Fix` button next to the diagnostic. Clicking
verifies the signature, re-validates admission, applies the mutation
via the same code path as today's `cha remediate --live`, runs a 60s
post-apply verification, and posts the outcome.

**LLM JSON schema** (`pkg/ai.AIProposedAction`):
```json
{
  "action_kind": "DeletePod" | "DeleteJob" | "PatchDeployment"
               | "DeleteCertRequest" | "DeleteACMEOrder",
  "target": { "kind": "...", "namespace": "...", "name": "..." },
  "rationale": "<1-2 sentence reason>",
  "rollback": { "description": "...", "action_kind": "<inverse>" }
}
```

The validator rejects:
- `action_kind` outside the closed enum
- `target.namespace` in `pkg/ai.ProtectedNamespaces`
- Missing or empty `rollback.description`
- `PatchPayload` set when `action_kind` ≠ `PatchDeployment`
- `Tier` value that doesn't allow proposals (T0/off)

**Patch shape validation (Sprint 3.1).** Beyond the closed-enum
`ActionKind` whitelist, the `PatchDeployment` payload itself is
allow-listed by JSON-path. The only permitted patch path is
`spec.template.metadata.annotations.kubectl.kubernetes.io/restartedAt`.
Anything else — replicas, selector, container images, env vars,
additional annotations — returns `ErrPatchForbidden` at admission.
Payload size capped at 64 KiB; the restart-annotation value capped
at 256 characters. Closes the StatefulSet-replicas-zero data-loss
vector from the 2026-05-22 threat model. Source:
[`CHA-com/ai/approval/patch_validator.go`](../../CHA-com/ai/approval/patch_validator.go).

**Click flow** (with audit checkpoints):
```
SRE clicks → JWT verify → MarkUsed (replay protection) → audit:granted
          → Executor.Validate → DefaultAdmissionPolicy.Admit
          → snapshot.Mutator.Delete/Patch → PostApplyVerifier.Verify (60s)
          → audit:applied (with post_apply_verified bool)
```

**Negative test coverage** (in `ai/approval/server_test.go`):
- Same URL clicked twice → 409 Conflict, executor not re-called
- URL used 16 minutes after issue → 410 Gone
- URL pointed at `kube-system` namespace → admission rejected
- Tampered token → 403 Forbidden
- No authenticated approver header → 401 Unauthorized
- Executor returns error → 500 Internal, `ai.action.failed` event

**Rate-limit profile**: 5 actions/hour cluster-wide (default). Bursts
allowed up to capacity; sustained rate ~1 fix per 12 min. Operators
with chatty incident environments can raise via
`ai.rateLimit.actionsPerHour`.

**Investigation rate limit (Sprint 3.2).** Layer-2 LLM-backed
investigation calls have an independent budget
(`ai.rateLimit.investigationsPerHour`, default 10) keyed on
diagnostic class. Without this, a flapping workload could uncapped-
burn investigations at ~144/day per resource. Per-class overrides
via `ai.rateLimit.perInvestigationClass`. Source:
[`CHA-com/ai/rate_limit.go`](../../CHA-com/ai/rate_limit.go).

**Cold-start mitigation (Sprint 3.3).** New rate-limit buckets
initialize at 0 tokens by default rather than full capacity.
Operators who can't extract a free burst on each pod restart. Set
`ai.rateLimit.coldStartFull: true` if you need the legacy
burst-on-startup behavior.

**RAG short-circuit (v0.2.0-alpha.1, default ON).** `--rag-short-circuit` is
now **on by default** (inert unless `--memory-store-url` is set). When
a previously-CLEARED, replayable fix for the same diagnostic class
exceeds the `--rag-short-circuit-threshold` (default `0.92`) cosine
similarity, the LLM proposal call is skipped and the prior fix is
replayed directly. Replayed fixes still pass the G6 precondition
re-check, autonomy gate, and post-apply verification, and their
outcomes are recorded to the RAG. Default-on means existing
deployments without a memory store are unaffected; deployments with
`--memory-store-url` automatically benefit once outcome records
accumulate.

**Tamper-evident audit (Sprint 3.6).** Wrapping any `AuditSink` in
`ChainedSink` (from `CHA-com/ai/audit/`) appends `prev_hash` and
`entry_hash` fields to each event's Details. `VerifyChain([]Event)`
walks the chain and returns the first broken-link index — detecting
content mutation, reordering, and insertion/deletion. Layer over an
append-only Vault audit device for full tamper resistance.

### T2 — Approved multi-step plan

**Capability**: When a diagnostic spans multiple correlated issues
(e.g., failing ESO → downstream Deployment in CCE → stuck CronJob),
emit a 1–5 step plan. The Slack/AM post shows the step-1 button only;
each subsequent step's button appears only after the prior step's
post-apply verification passes.

**LLM JSON schema**:
```json
{
  "plan_id": "<uuid>",
  "steps": [
    {
      "step_n": 1,
      "action_kind": "<T1 whitelist enum>",
      "target": { ... },
      "rationale": "...",
      "rollback": { ... },
      "prerequisite_step": 0
    },
    ...
  ]
}
```

**Constraints enforced**:
- `MaxPlanSteps = 5` (hard cap; planner truncates or rejects)
- Each step independently validates as a T1 action
- `prerequisite_step` chains plan ordering; CHA fills `PrerequisiteActionID`
- Plans cannot self-modify (LLM has no `Plan` action_kind)
- Plans with circular dependencies are rejected at construction

**Plan state machine** (per `plan_id`):
- `proposed` → SRE clicks step 1 button
- `step_1_pending` → admission re-validates → execute → verify
- `step_1_success` → step 2 button posted as Slack thread reply
- … repeats …
- `complete` → final summary message
- At any point, `Cancel Plan` button aborts remaining steps

**Operator decision**: T2 produces meaningful uplift when incident
patterns are recurrent (e.g., known cascades). For one-off issues, T1
suffices. Recommended posture: enable T2 for a 2-week pilot, measure
median steps-per-plan vs steps-actually-needed; tune the fixer
matcher heuristics if T2 over-proposes.

### T3 — Break-glass Vault runbook

**Capability**: When a diagnostic source is `VaultPathMissing` or
`FailingExternalSecrets`, emit a Vault recovery runbook in the Slack
post. The runbook specifies:
- Which Vault path needs the missing keys (validated against the
  operator-supplied `AllowedPathPrefixes` allowlist)
- Which key NAMES (never values) must be populated
- A `vault kv patch` command template with `${VALUE_NAME}` placeholders
- Pre/post manual steps (rotate upstream key, verify, etc.)

**CHA-com NEVER executes Vault writes in T3.** The runbook is for
human execution by two distinct SREs.

**Dual-approval flow**:
```
RunbookProposer emits runbook → posts to #ceph-critical with TWO
"Acknowledge Runbook" buttons.

SRE A clicks → state.first set → audit: ai.runbook.approval_recorded
                                  (slot=1, complete=false)
30-minute mandatory delay elapses.
SRE B (≠ A) clicks → state.second set → audit: ai.runbook.approval_recorded
                                         (slot=2, complete=true)
SRE A + B together run `vault kv patch` from THEIR shell using values
they control.
SRE clicks "Mark Resolved" → CHA-com re-runs VaultPathMissing analyzer
                              → if cleared, diagnostic resolves
```

**Constraints**:
- Same approver clicking both slots → `ErrSameApprover` (409 Conflict)
- Second click before 30-min window → `ErrTooEarly` (409 Conflict)
- Vault path outside allowlist → `ai.runbook.rejected` (no Slack post)
- Any field in the runbook matching a Secret-value heuristic
  (`pkg/ai.ContainsSecretLike`: Vault tokens, JWTs, GH PATs, AWS keys,
  Slack tokens, base64 ≥40 chars, hex ≥32 chars) → `ai.runbook.rejected`

**Operator decision**: T3 is the highest-trust tier. Only enable when:
1. You have a SOC2 program or equivalent audit requirement
2. You have two distinct on-call SREs available within a 30-min window
3. Your Vault `AllowedPathPrefixes` allowlist is intentional and reviewed

Recommended posture: keep T3 disabled by default. Enable per-namespace
via Helm value when an incident class repeatedly requires the same
recovery action.

---

## Tier ↔ Helm value mapping

| Helm value / env | T0 | L2 (rule-based) | L2 (LLM-backed, paid) | T1 | T2 | T3 |
|---|---|---|---|---|---|---|
| `ai.enabled` | true | n/a (separate switch) | true | true | true | true |
| `ai.tier` | t0 | n/a | t0+ | t1 | t2 | t3 |
| `ai.endpoint` | required | n/a | required | required | required | required |
| `CHA_INVESTIGATOR` env | unchanged | (default on) | overridden by paid binary | unchanged | unchanged | unchanged |
| `approval.enabled` | false (n/a) | false (n/a) | false (n/a) | true | true | true |
| `approval.ingress.enabled` | false | false | false | true | true | true |
| `approval.ingress.host` | — | — | — | required | required | required |
| `approval.approvers.minDistinctApprovers` | n/a | n/a | n/a | 1 | 1 | **2** (auto) |
| `gatekeeper.install` | optional | optional | optional | recommended | recommended | required |

Layer-2 is orthogonal to the T0–T3 escalation ladder: it can be on with
all paid tiers off, and it stays on through downshift. The proposal /
approval / execution chain only applies to T1+.

Higher T-tiers strictly require the prerequisites of all lower T-tiers
(installation order: T0 → T1 infrastructure → T2 plan state → T3
dual-approval RBAC). Layer-2 has no prerequisites beyond the OSS
watcher itself.

---

## Disabling a tier (downshift)

To temporarily downshift (e.g., during a model rollout):
```sh
helm upgrade cha cha/cluster-health-autopilot --reuse-values --set ai.tier=t0
```
The watcher reads the value each cycle; downshift takes effect within
one watcher resync (default 10 min). In-flight plans (T2) and runbooks
(T3) remain valid until their TTL expires, so downshift never strands
pending approvals. Layer-2 investigation continues unchanged through
T-tier downshifts.

Full T-tier disable: `--set ai.enabled=false`. The rule-based Layer-2
investigator keeps running — its output is part of OSS behavior, not
the paid AI surface.

To return the cluster to bit-identical pre-v1.5 OSS behavior (no
investigator at all), additionally set `CHA_INVESTIGATOR=off` on the
watcher Deployment.
