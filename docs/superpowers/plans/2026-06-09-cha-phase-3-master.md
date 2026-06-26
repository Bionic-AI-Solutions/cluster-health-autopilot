# Phase 3 — Federation + Autonomous Completion + Operator Polish

**Status:** strawman draft; awaiting scope confirmation before execution.

**Parent (Phase 2 master plan):** [2026-06-07-srenix-phase-2-master.md](2026-06-07-srenix-phase-2-master.md)

---

## Goal

Phase 2 closed the test→learn loop *within one cluster*. Operators see Srenix learn from outcomes, propose fixes that match past resolutions, and silently auto-apply trusted class policies. The promise still half-delivered:

- **Learning stays single-cluster.** A fix that worked on cluster A is invisible to cluster B; each install builds its own memory from scratch. For the fintech/healthtech ICP (running 3-15 clusters), this means N-redundant learning curves instead of one.
- **High-confidence fixes still need a human click.** Even when (class-policy matches) AND (attestation verifies) AND (confidence >0.9), DigestPin opens a PR that waits for an SRE merge. The promised "incidents resolved without paging a human" goal is one merge away.
- **Operator-managed installs still need `extraArgs` escape hatches.** The chart fields I shipped in Phase 2 (`ai.metrics`, `ai.replicas`, `ai.digestPinAttestation`) are operator-aware only for some surfaces. ArgoCD/Flux operators using pure-CR installs hit the escape hatch for `--metrics-addr`, the LLM proposer flags, etc.
- **Coverage gaps in the analyzer catalog.** The Phase 2 plan deferred 10 from the original 15-analyzer wishlist; the field is now asking for OOMKill recurrence, PV orphan, stuck CronJob backoff (top 3 most-requested).

Phase 3 closes those three gaps.

## Non-goals

- Web UI. Still out (Slack-only is the contract).
- Cross-cluster federation that aggregates SECRETS across customer boundaries. Federation is opt-in per signal class + always single-customer.
- General-purpose patch DSL. The closed `ActionKind` enum stays closed.

## Sub-deliverables

### 3.A — Cross-cluster RAG read federation (Srenix Enterprise paid tier)

A single Qdrant tenant per customer holds outcomes from every cluster the customer runs. By default, each cluster reads ONLY its own outcomes (per-cluster `ClusterID` partition — already in `pkg/rag.Entry.ClusterID`). Operators flip a per-class opt-in to widen the read scope.

**Why now:** the bigger an operator's fleet, the more this matters. Two clusters → 2× learning speed when federated. Phase 2.A already partitions on ClusterID; the federation hook is one wider-list call.

**Risks:** cross-cluster signal leakage if an operator widens a class scope and a different cluster's diagnostic accidentally matches (e.g. same Pod name in different namespaces). Mitigation: keep `Target` substring in the Match check, never widen to `Source`-only.

### 3.B — Auto-merge DigestPin PRs at very-high confidence

When ALL of:
- Class-policy exists with `Decision="approve"` + `MessagePattern` matches
- Attestation signature verifies (Phase 2.H) against the embedded public key
- Class success-rate confidence (Phase 2.C Wilson lower bound) ≥ 0.95
- Circuit breaker is closed
- Target repo has Srenix bot listed in branch protection allowlist

…the DigestPin PR auto-merges via GitHub's merge API. No SRE click required.

**Soft fail everywhere:** if any condition fails, PR stays in "awaiting human approval" — the existing Phase 2.B Slack button row is preserved.

### 3.C — Investigator-level RAG grounding

The investigator (the srenix-enterprise module that explains WHY a finding fired) currently re-investigates from scratch every cycle. Phase 3.C grounds its prompt in past investigation outcomes: "we investigated this finding 14 times in the last 30 days; the conclusion was always X." For repeat findings, the investigator's wall-clock + token cost drops ~90% and the conclusion gets sharper.

Uses Phase 2.A's existing `RecentOutcomesByTarget` — no new memory surface needed.

### 3.D — Operator schema parity for Phase 2 chart-only fields

Promote into `api/v1alpha1.AISpec`:
- `Metrics.Addr` + `Metrics.Port` + `Metrics.ServiceMonitor` + `Metrics.GrafanaDashboard` + `Metrics.PrometheusRule` — the operator renders Service + ServiceMonitor + PrometheusRule + ConfigMap CRs
- `LLMProposer.Enabled` — typed flag instead of `extraArgs: ["--llm-proposer=true"]`

So a pure CR-driven install (ArgoCD/Flux/kubectl apply) gets the full Phase 2 surface without escape hatches.

### 3.E — 3 new analyzers (from the deferred 10)

- **OOMKill recurrence** — Pod with ≥3 OOMKilled events in 24h. CapacityDrift already surfaces HPA-pinned-at-max; this catches the case where the pod just needs `resources.limits.memory` raised.
- **PV orphan** — PersistentVolume `Released` for >7d with no claim, costing real $$$. Warning per orphan PV.
- **Stuck CronJob backoff** — CronJob with backoffLimit exhausted + no new run in 24h × `spec.schedule`. The job is silently not running.

### 3.F — Compliance audit-bundle exporter

`srenix audit-bundle --since 30d --output bundle.tar.gz` produces a SOC2-friendly evidence pack:
- Every approval click + the JWT it signed
- Every auto-apply (autonomy decision + verifier result)
- Every digest-pin attestation + the PR URL it landed on
- Every outcome recorded into RAG (with timestamp + cluster + verdict)

Reads the existing audit JSONL + RAG store; emits a flat tarball. Useful for the fintech/healthtech ICP's annual audits.

## Sequencing

Independent — pick by operator pain. Suggested order:
1. **3.E** first (3 new analyzers) — easy wins, expands coverage, fast PR cycle
2. **3.B** next (auto-merge DigestPin) — closes the autonomous-fix loop; high marketing impact
3. **3.D** third (operator schema parity) — unblocks ArgoCD/Flux ICP segment
4. **3.C** (investigator RAG) — incremental quality, paid-tier-only
5. **3.A** (cross-cluster federation) — needed only once operators have >1 cluster; defer until first paying customer hits the limit
6. **3.F** (audit bundle) — needed for first compliance review (estimated 3-6 months out)

## Risks

- **Auto-merge blast radius.** 3.B is the single highest-risk feature in Phase 3 — a buggy auto-merge that breaks production is catastrophic for the GTM moat narrative. Gate behind explicit per-class opt-in + 30-day soak with manual review before auto-merge fires.
- **Federation governance.** 3.A needs careful policy on which signals are eligible for cross-cluster aggregation. Start with `KindFindingOutcome` only; defer `KindWorkload` (might leak namespace structure).
- **Operator schema bloat.** 3.D risks turning `AISpec` into a kitchen-sink. Counter: only fields with active operator-managed-install demand make it in.
