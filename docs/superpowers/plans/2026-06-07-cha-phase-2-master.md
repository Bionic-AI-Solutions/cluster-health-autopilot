# CHA Phase 2 — Master Plan

**Status:** DRAFT awaiting approval — written 2026-06-07 immediately after Phase 1 close.

**Phase 1 closed with:** all 5 deliverables LIVE on cluster (1.A cha-com→Slack bridge, 1.B placeholder substitution, 1.C Forge rate-limiter + dedup, 1.D operator-managed OpenProject ticketing, 1.E per-cycle delta view). Operator + watcher on `cluster-health-autopilot:1.20.1`; aiwatch on `cha-com:1.14.0`. OpenProject ticketing creating real work-packages (verified 2026-06-07 with ids 1715+).

**Phase 2 scope (user-confirmed 2026-06-07):** Tier A + B + C from the Phase-1-close adversarial-review proposal. Skipping Tier D housekeeping (rolled into the deliverables instead).

---

## Why these, and why this order

Phase 2 closes the **test → learn → next-cycle-detect** loop that Phase 1 deliberately stopped short of. The audit pattern that surfaced 4 real bugs in Phase 1 (placeholder gaps, chart/CRD shape mismatch, missing register test, **DriftReport.spec.remediation dropped since the field existed**) generalizes to a product feature: when CHA applies a fix, the outcome (worked? reverted? humans complained?) should bend its future behavior. Today nothing learns from outcomes; every cycle is fresh-faced.

**Sequence-locked dependencies:**
- 2.A (RAG memory write path) is the foundation. 2.B + 2.C cannot land without it.
- 2.B + 2.C are independent once 2.A is in.
- 2.D (LLM proposer) benefits hugely from 2.A (memory-grounded prompts) but doesn't require it; ship 2.A first so 2.D's first cut is memory-aware from day 1.
- 2.E/2.F/2.G/2.H are fully independent of A/B/C and of each other.

**Recommended execution order:**

```
2.A RAG memory write (Tier A — foundation)
  ├─ 2.B Approve+remember-class workflow         ┐
  └─ 2.C Confidence model from memory            ┤  parallelizable
                                                  ┘
2.D LLM-driven proposer (Tier B)
2.E 3-5 high-pain new analyzers (Tier B, not all 15)
2.F HA aiwatch (Tier C)
2.G Grafana dashboards (Tier C — after 2.F so HA metrics are real)
2.H Cosign-signed PRs (Tier C, smallest, fits anywhere)
```

Estimated wall-clock: **6-8 weeks for full Tier A+B+C** at ~1-2 weeks per deliverable. Tier A alone (2.A + 2.B + 2.C) is ~3 weeks and delivers the strategic moat.

---

## Per-deliverable scoping (no per-task breakdown yet — those come as sub-plans)

### 2.A — RAG memory write path on every action outcome

**The promise.** Every applied / denied / reverted action writes a structured outcome to Qdrant `kind=outcome`. Proposers query memory before proposing ("class X reverted 3× in the last week"; "applying Y to namespace pg always succeeds").

**Files of interest** (already exist, need wiring):
- `pkg/ai.OutcomeRecorder` — interface defined, no implementation invokes it from the watcher
- `ai/memory/rag_qdrant.go` — has scroll/list/get/upsert; needs an `Outcome` entry type added next to `KindWorkload`
- `ai/approval/server.go` — Approve and Deny handlers should call OutcomeRecorder.Record
- `cmd/cha-com/watch_cmd.go::tick()` — after autonomy.Consider auto-applies, should record an outcome for each applied action

**Anti-goals.**
- Not a generic event bus. Outcome record is single-shape: `{applied_at, action_kind, target, diag_subject, decision, outcome (succeeded|reverted|denied), reverted_at|nil, denied_by|nil}`.
- No RAG-as-database. RAG is for similarity search; canonical record-of-truth is still K8s events + DriftReport status.

**Deliverable boundary.**
- Outcome write path works end-to-end (Slack click → approval-server → record → Qdrant search returns it).
- Bare proposer side: ONE proposer (DigestPin) reads memory and surfaces "we tried this before" rationale in the proposal message. Other proposers updated in 2.D's wave.

**TDD plan target:** ~12 sub-tasks. New file: `ai/memory/outcome.go`. Modified: `ai/approval/server.go`, `cmd/cha-com/watch_cmd.go`, `ai/proposer/digest_pin.go`.

---

### 2.B — Approve+remember-class workflow

**The promise.** Each Slack message's Approve/Deny pair gets a 3rd button: **"Approve and remember this class"**. Clicking it (a) executes the action, (b) records a class-level policy in RAG so the same class auto-approves silently for the next 7 days, (c) surfaces a "muted N items via class policy" line on the next Slack post so the operator knows the policy fired.

Inverse: each post also gets a **"Silence this class via click"** button → creates a 24h Silence CR scoped to the diagnostic class (not subject — Phase 1 silences are subject-scoped, which is a 1-off, not a class policy).

**Files of interest.**
- `ai/approval/server.go` — new routes `/approve-class`, `/deny-class`, `/silence-class`
- `cmd/cha-com/ai_slack_digest.go` — add 3rd button per item
- `internal/report/routing.go` — add the new button to the OSS watcher's Slack render too (parity with cha-com)
- `pkg/silence/lister.go` — already filters by subject; extend to filter by `(source, message-pattern)` for class-scoped silences
- `pkg/ai/policy.go` — NEW. Stores class policies in RAG. `PolicyMatches(diag) bool` queried by autonomy engine.

**Anti-goals.**
- No web UI. Slack-button-only.
- Class definition is `(Source, ActionKind, diag-message-regex)` not free-form. Operators don't write regex; the UI infers it from the clicked item.

**Deliverable boundary.**
- All 3 new buttons render in Slack
- All 3 routes implemented on approval-server
- Autonomy engine consults policy before its existing decision
- Per-cycle observability: "muted N via policy / silenced N via class"

**TDD plan target:** ~15 sub-tasks.

---

### 2.C — Confidence model from memory

**The promise.** Replace static `--autonomy-min-confidence=0.7` with a per-(analyzer, ActionKind, namespace) success-rate computed from outcomes. Cold start = static default. After 5+ samples, use rolling average. After 20+, use exponentially-weighted (recent outcomes matter more).

**Files of interest.**
- `pkg/ai/autonomy.go::DecideAutonomy` — current static gate; new dynamic gate
- `pkg/ai/confidence.go` — NEW. `Compute(analyzer, kind, ns) (float64, samples int)`
- `ai/memory/outcome.go` (from 2.A) — Query the success-rate

**Anti-goals.**
- No ML. Just rolling success rate, optionally exponentially weighted. The operator should be able to reason about the math.
- No per-resource confidence. Only per-class. Resource-specific drift is what the per-subject Silence CR is for.

**Deliverable boundary.**
- Cold-start identical to current static behavior.
- After 20 outcomes accumulate, autonomy gate honors computed confidence.
- New per-cycle log: "autonomy: ActionKind=ApplyManifest ns=default confidence=0.85 (samples=27) → auto-apply".

**TDD plan target:** ~8 sub-tasks.

---

### 2.D — LLM-driven proposer for arbitrary findings

**The promise.** Current proposers are class-specific (NetworkPolicy / DigestPin / VaultRunbook). A general LLM proposer reads any Diagnostic + RAG memory + workload context, calls cha-com's LLM, returns an action (PR / kubectl-patch / silence) or declines.

**Files of interest.**
- `ai/proposer/llm.go` — NEW. Implements pkg/ai.FixProposer
- `ai/prompts/llm_proposer.md` — NEW. System prompt + few-shot examples
- `cmd/cha-com/watch_cmd.go` — register the LLM proposer last in the proposer chain so deterministic proposers (faster, cheaper) win first

**Anti-goals.**
- Doesn't replace existing proposers. They run first; LLM is fallback for findings the deterministic proposers skip.
- No fine-tuning. Standard chat-completion API.
- Decline must be the default. The prompt emphasizes "if you're not confident, decline".

**Deliverable boundary.**
- LLM proposer produces actions for at least 2 finding classes currently unaddressed (e.g., HPA-pinned-low, ResourceQuota-saturated)
- Outcomes flow into the same RAG path as 2.A
- Operator can disable per-class via `--llm-proposer-class-skip=HPADrift`

**TDD plan target:** ~12 sub-tasks.

---

### 2.E — 3-5 high-pain new analyzers (out of the 15 originally listed)

**The promise.** Add high-value detection for failures Phase 1 doesn't catch.

**Confirmed slate (4 analyzers, user-approved 2026-06-07):**
- 2.E.1 — **HPA pinned at min** for >30 days (analyzer pre-existed in CapacityDrift but only fires when min<max; add the always-pinned-at-low case)
- 2.E.2 — **PDB blocks all evictions** (`disruptionsAllowed=0` for >grace; common cause of cluster-upgrade hangs)
- 2.E.3 — **Stuck Job pods** (a Job that ran past `activeDeadlineSeconds` but pods still Running)
- 2.E.4 — **Stale ResourceQuota** (Quota allocated > 80% AND no workload-creation events in 7 days; common before pods stop scheduling)

(Deferred for future: OOMKill recurrence, the other 10 from the original "15 analyzers" wishlist.)

**Anti-goals.**
- Not 15 analyzers. 3-5. The remaining 10 land incrementally based on operator pain.
- No per-analyzer AI tier integration. They emit Diagnostic + Remediation; LLM proposer (2.D) handles them generically.

**Deliverable boundary.**
- Each analyzer ships independently as its own PR
- Each has 6+ unit tests (positive, suppressed-within-grace, all skip-paths)
- Each registered in `catalog/registry.go`

**TDD plan target:** ~10 sub-tasks per analyzer × 5 = ~50 sub-tasks total. Shipped incrementally.

---

### 2.F — HA aiwatch

**The promise.** aiwatch becomes leader-elected (mirroring the OSS watcher). Single Deployment, replicas=2+. Lease in cluster-health-autopilot namespace. Standby pod reads but never writes.

**Files of interest.**
- `cmd/cha-com/watch_cmd.go` — wrap the tick loop in a leader-election gate (same `client-go/tools/leaderelection` as OSS watcher)
- `charts/cluster-health-autopilot/templates/aiwatch-deployment.yaml` — replicas: 2 default
- `internal/operator/builders.go::BuildAIWatch` — same

**Anti-goals.**
- No multi-leader / active-active. Pure HA single-leader.
- No state migration. Standby starts fresh on lease acquisition (matches OSS watcher behavior).

**Deliverable boundary.**
- Two pods running, exactly one ticks at any time
- Lease handoff verified by killing the leader and watching the standby promote within 30s
- No double-Slack-post on handover

**TDD plan target:** ~6 sub-tasks.

---

### 2.G — Grafana dashboards

**The promise.** Two dashboards published as a Helm sub-chart:
- **CHA Operations** — driftreports total by source/severity, cycle latency, slack-post failures, ticketing upsert lag
- **CHA AI Tier** — proposals by ActionKind, autonomy auto-applied vs awaiting-approval vs declined rate, mean confidence, RAG outcome distribution

**Files of interest.**
- `charts/cluster-health-autopilot/dashboards/` — NEW. Grafana JSON files
- `internal/metrics/` — NEW. Prometheus metrics exported by watcher + aiwatch (gauges + histograms)
- `cmd/cha/main.go` + `cmd/cha-com/watch_cmd.go` — start a `:9090/metrics` HTTP listener

**Anti-goals.**
- No Grafana operator. Plain ConfigMaps + the standard `grafana_dashboard=1` label.
- No alerts (those live in Alertmanager via existing routing).

**Deliverable boundary.**
- Metrics exported and scraped (verify via `kubectl port-forward` to :9090/metrics)
- Both dashboards render in Grafana with non-empty panels
- Documented `helm install cha --set dashboards.enabled=true`

**TDD plan target:** ~8 sub-tasks (mostly metrics wiring; dashboard JSON is hand-authored).

---

### 2.H — Cosign-signed PRs from DigestPin proposer

**The promise.** When DigestPin opens a PR pinning `image:tag` → `image@sha256:…`, the PR includes a verification footer: `cosign verify <digest> --certificate-identity=… --certificate-oidc-issuer=…` so the merger can prove the digest was sigstore-attested before merge.

**Files of interest.**
- `ai/proposer/digest_pin.go` — extend prBody with the verification block; query the Rekor API for the certificate identity
- `ai/forge/cosign.go` — NEW. Wraps the `cosign verify` invocation

**Anti-goals.**
- No CHA-side signing of anything. CHA only emits the proof-of-attestation for the digest it observed; the originating image's signer is the one with the keys.
- Optional. Off by default; opt-in via `--digest-pin-cosign-footer=true`.

**Deliverable boundary.**
- DigestPin PRs include the verification footer when cosign-footer is on
- Footer renders correctly for an attested image (verified against a real example)
- Footer omitted for an unattested image (we don't lie)

**TDD plan target:** ~6 sub-tasks.

---

## Methodology — same as Phase 1 with two refinements

Phase 1's methodology held except for the test→learn loop. Phase 2 keeps:
- **Plan first** (this doc + sub-plans)
- **TDD discipline** (red → green → commit → push per unit)
- **Full regression every change** (`go test ./...`, `go vet ./...`, `golangci-lint run`, `helm lint`)
- **Adversarial review at each tier boundary** (after 2.A+B+C, after 2.D+E, after 2.F+G+H)
- **Live cluster verification** is the canonical "TEST" stage

Two refinements based on Phase 1 lessons:
1. **Fold audit findings back into the plan template** so the next plan is born adversarially-hardened. Specifically: every deliverable's "TDD plan target" includes an explicit integration-test sub-task to prove Diagnostic.X actually travels through to all downstream sinks (DriftReport, Slack, Alertmanager, Ticketing). The DriftReport.spec.remediation bug existed for years because no such test existed.
2. **Multi-arch manifest assembly is part of the release runbook.** goreleaser's 1h timeout repeatedly leaves the manifest list unassembled. The post-tag pipeline now always runs `docker manifest inspect && (if missing) docker manifest create + push`.

---

## Acceptance — Phase 2 complete

- [ ] CHA auto-improves: a proposer that gets reverted twice in a row degrades its confidence and stops auto-applying that class (2.A + 2.C)
- [ ] Operators silence classes via Slack click (2.B); the muted-count surfaces on next post
- [ ] LLM proposer handles at least 2 finding classes deterministic proposers don't (2.D)
- [ ] 3-5 new analyzers live (2.E)
- [ ] aiwatch survives leader pod kill, standby takes over <30s (2.F)
- [ ] Both Grafana dashboards render with real data (2.G)
- [ ] DigestPin PR includes cosign verification footer (2.H)
- [ ] DriftReport-style "field travels end-to-end" tests added for every NEW Diagnostic field introduced in Phase 2

---

## Rollback plan — each deliverable independently revertable

Same shape as Phase 1's rollback section:
- 2.A: drop OutcomeRecorder wire-in; outcomes simply not recorded
- 2.B: remove the 3rd button render; policies in RAG become inert
- 2.C: revert confidence.go; autonomy falls back to static threshold
- 2.D: `--llm-proposer-enabled=false`
- 2.E: each analyzer behind its own feature flag in catalog
- 2.F: `replicas: 1` in CR overrides; leader election still works with 1 replica
- 2.G: `dashboards.enabled=false`
- 2.H: `--digest-pin-cosign-footer=false`

---

## Out of scope (Phase 3+)

- Cross-cluster RAG federation (still on the deferred list)
- Auto-merge of CHA PRs (requires policy + branch protection coordination)
- Web UI (Slack-only remains the contract)
- The remaining 10 analyzers (incremental, no Phase boundary)
