# AI Tiers — Definitive Specification (v1.0.0)

This document is the single source of truth for the four AI tiers shipped
in CHA-com v1.0.0. Operators reference this when sizing budgets, picking
a deployment posture, or reasoning about blast radius.

**Companion docs**:
- [AI_USAGE.md](AI_USAGE.md) — why we have AI tiers (positioning)
- [THREAT_MODEL_AI.md](THREAT_MODEL_AI.md) — OWASP LLM Top 10 mapping
- [ADVERSARIAL_ANALYSIS.md §8](ADVERSARIAL_ANALYSIS.md#8-ai-tier-attack-surface) — security review of the AI surface
- [SETUP_GUIDE.md §14](SETUP_GUIDE.md#14-ai-tier-setup) — installation walkthrough

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

---

## Tier matrix

| Property | T0 Narration | T1 Single fix | T2 Multi-step | T3 Vault runbook |
|---|---|---|---|---|
| **What ships** | Slack/AM `🤖` enrichment block | "Apply Fix" button | Step-by-step approval | Dual-approval runbook |
| **LLM input** | Diagnostic (redacted) | + matching fixer name | + cross-resource context | + ESO refs |
| **LLM output** | EnrichedDiagnostic JSON | AIProposedAction | up to 5 sequential actions | VaultRunbook |
| **Mutation surface** | None | OSS whitelist (5 verbs) | Same as T1 | Zero — runbook is human-run |
| **Approval gate** | n/a — read-only | One-click signed URL | One-click per step | **Dual** (2 distinct approvers, 30-min audit window) |
| **Click TTL** | n/a | 15 min | 15 min | 90 min |
| **Replay protection** | n/a | JTI one-time-use | JTI one-time-use | JTI one-time-use |
| **Protected NS blocked** | n/a | LLM + validator + admission | Per-step | n/a — Vault path allowlist |
| **Rollback required** | n/a | Yes | Per-step | Manual runbook step |
| **Post-apply verify** | n/a | 60s window | Per-step gate to next | n/a |
| **Audit events** | `ai.enrichment.*` | + `ai.proposal.*`, `ai.approval.*`, `ai.action.*` | + `ai.plan.*` | + `ai.runbook.*` |
| **New RBAC** | none | none (reuses remediator) | none | none |
| **Default** | off | off | off | off |
| **Risk class** | Privacy (sends diagnostic JSON to LLM) | Mutation (with approval) | Mutation × N | Vault knowledge leak (key NAMES only) |

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

| Helm value | T0 | T1 | T2 | T3 |
|---|---|---|---|---|
| `ai.enabled` | true | true | true | true |
| `ai.tier` | t0 | t1 | t2 | t3 |
| `ai.endpoint` | required | required | required | required |
| `approval.enabled` | false (n/a) | true | true | true |
| `approval.ingress.enabled` | false | true | true | true |
| `approval.ingress.host` | — | required | required | required |
| `approval.approvers.minDistinctApprovers` | n/a | 1 | 1 | **2** (auto) |
| `gatekeeper.install` | optional | recommended | recommended | required |

Higher tiers strictly require the prerequisites of all lower tiers
(installation order: T0 → T1 infrastructure → T2 plan state → T3
dual-approval RBAC).

---

## Disabling a tier (downshift)

To temporarily downshift (e.g., during a model rollout):
```sh
helm upgrade cha cha/cluster-health-autopilot --reuse-values --set ai.tier=t0
```
The watcher reads the value each cycle; downshift takes effect within
one watcher resync (default 10 min). In-flight plans (T2) and runbooks
(T3) remain valid until their TTL expires, so downshift never strands
pending approvals.

Full disable: `--set ai.enabled=false`. Returns the cluster to OSS
behavior bit-for-bit.
