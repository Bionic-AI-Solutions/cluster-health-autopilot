# Phase 2.A — RAG Memory Write Path on Action Outcomes

**Status:** active — execution started 2026-06-07 immediately after Phase 1 close and the master Phase 2 plan.

**Parent:** [2026-06-07-cha-phase-2-master.md](2026-06-07-cha-phase-2-master.md)

**Branch:** `phase2a/rag-outcomes` (CHA-com) + `phase2a/rag-outcomes` (OSS, if any analyzer-side wiring needed)

---

## Goal

Every applied / denied / reverted action writes a structured outcome to Qdrant `kind=outcome`. One proposer (DigestPin) reads memory before proposing and surfaces "we tried this before" rationale. Sets up 2.B (policy reads) and 2.C (confidence reads).

## Anti-goals

- Not a generic event bus. Outcome is a single shape.
- RAG is for similarity search; canonical truth still K8s events + DriftReport status.
- Only ONE proposer reads memory in 2.A. Other proposers in 2.D's wave.

## Outcome record shape

```go
type Outcome struct {
    AppliedAt     time.Time     // When the action was executed (or denied/reverted)
    ActionKind    pkgai.ActionKind  // ProposePullRequest | ApplyManifest | RunRunbook | …
    Target        pkgai.ObjectRef   // What the action operated on
    DiagSubject   string        // The diagnostic that triggered this
    DiagSource    string        // SecurityDrift | CapacityDrift | …
    Decision      string        // "auto-applied" | "approved-by-human" | "denied-by-human"
    Result        string        // "succeeded" | "reverted" | "failed" | "denied"
    RevertedAt    *time.Time    // nil unless Result=reverted
    Approver      string        // X-Forwarded-User; empty for auto-applied
    Rationale     string        // What the proposer said; for memory-grounded follow-ups
}
```

## Sub-tasks (TDD; bite-size)

### 2.A.1 — Outcome type in `ai/memory/outcome.go`
- [ ] Define `Outcome` struct
- [ ] Define `kindOutcome rag.EntryKind = "outcome"`
- [ ] Define helper `OutcomeKey(actionID) string` for deterministic Qdrant point IDs (so re-records overwrite)
- [ ] Failing test in `ai/memory/outcome_test.go`: `TestOutcome_KeyDeterministic` — same actionID → same key
- [ ] Run, confirm fail, implement, run, pass

### 2.A.2 — `OutcomeRecorder` implementation
- [ ] In `ai/memory/outcome.go`: `type Recorder struct { rag rag.Writer }` with method `Record(ctx, Outcome) error`
- [ ] Record serializes Outcome to a `rag.Entry` (features map) and calls `Upsert` keyed on `OutcomeKey(actionID)`
- [ ] Failing test `TestRecorder_RoundTrip` — record + read via fake-rag → identical Outcome
- [ ] Run, confirm fail, implement, run, pass

### 2.A.3 — `OutcomeReader` query helpers
- [ ] In `ai/memory/outcome.go`: `Reader` with methods:
  - [ ] `RecentByClass(ctx, source, kind, since) ([]Outcome, error)` — for confidence model
  - [ ] `RecentByTarget(ctx, target, since) ([]Outcome, error)` — for "we tried this before"
- [ ] Failing test `TestReader_RecentByClass` — 5 outcomes seeded, query returns expected subset, sorted recent-first
- [ ] Failing test `TestReader_RecentByTarget` — same shape, target-scoped
- [ ] Implement (both go through Qdrant scroll + filter), run, pass

### 2.A.4 — Wire `Record` into `approval-server` (Approve handler)
- [ ] In `ai/approval/server.go::handleApprove`: after `Executor.Apply` succeeds, call `recorder.Record(Outcome{Decision: "approved-by-human", Result: "succeeded", Approver: hdrUser, …})`
- [ ] Recorder is optional (Server has `recorder OutcomeRecorder` field; nil = no-op — matches existing pattern)
- [ ] Failing test `TestHandleApprove_RecordsOutcome` — fake recorder asserts a Record call with the right Outcome shape
- [ ] Implement, pass

### 2.A.5 — Wire `Record` into `approval-server` (Deny handler)
- [ ] In `ai/approval/server.go::handleDeny`: record `Outcome{Decision: "denied-by-human", Result: "denied", Approver: hdrUser, …}`
- [ ] Failing test, implement, pass

### 2.A.6 — Wire `Record` into autonomy auto-apply path
- [ ] In `cmd/cha-com/watch_cmd.go::tick()`: after `autonomy.Consider` returns `Decision.AutoApply=true` AND the apply succeeds, record `Outcome{Decision: "auto-applied", Result: "succeeded", Approver: "", …}`
- [ ] Failing test in `cmd/cha-com/watch_cmd_test.go`: fake autonomy + fake recorder, assert Record called once per auto-apply
- [ ] Implement, pass

### 2.A.7 — Wire revert detection
- [ ] Revert = a finding that was auto-applied/approved in cycle N reappears identically in cycle N+1 within 5 minutes
- [ ] In `cmd/cha-com/watch_cmd.go::tick()`: after building the new diagnostic set, query recorder for outcomes <5 min old; for any whose DiagSubject re-appears with `Result=succeeded`, write a follow-up `Outcome{Result: "reverted", RevertedAt: time.Now()}`
- [ ] Failing test, implement, pass

### 2.A.8 — DigestPinProposer reads memory before proposing
- [ ] In `ai/proposer/digest_pin.go::Propose`: before opening a new PR, query `reader.RecentByTarget(podTarget, 7days)` for prior outcomes
- [ ] If a prior outcome's Result=reverted within 24h → include "previously attempted, was reverted" in the proposal's Rationale field
- [ ] Failing test `TestDigestPin_MemoryAwareRationale` — fake reader returns a reverted outcome; assert the new proposal's Rationale mentions it
- [ ] Implement, pass

### 2.A.9 — Helm + operator wiring
- [ ] `ai.outcomeRecorder.enabled: true` default-on Helm value
- [ ] Operator wires the OutcomeRecorder into both watchLoop and approval-server when CR.spec.ai.enabled=true (no new CR field needed — reuses existing RAG store config)
- [ ] Failing operator test `TestBuildAIWatch_RecorderWiredWhenRAGEnabled`, implement, pass

### 2.A.10 — Per-cycle observability
- [ ] New log line each cycle: `outcomes: recorded=N (auto-applied=A approved=B denied=C reverted=D)`
- [ ] Failing test on the log assertion, implement, pass

### 2.A.11 — Field-travels-end-to-end integration test
- [ ] Per the master plan's Phase-1-lesson refinement: an explicit test asserting a recorded outcome can be queried back by both RecentByClass and RecentByTarget. Catches the DriftReport.spec.remediation class of bug where one side writes and the other side reads but the field is dropped between them.
- [ ] Test name: `TestOutcome_WrittenByRecorderReadBackByReader`. Lives in `ai/memory/outcome_integration_test.go` (build-tag `integration`).

### 2.A.12 — Local build + dev tag + cluster verify
- [ ] CGO_ENABLED=0 build cha-com binary
- [ ] Push docker4zerocool/cha-com:1.15.0-dev1
- [ ] Patch CR ai.image.tag, wait rollout
- [ ] Force a few proposals (click Approve on real Slack URLs)
- [ ] Verify outcomes in Qdrant via `cha rag-debug query --kind=outcome` (new debug subcommand, or curl directly)

### 2.A.13 — Open PR + release
- [ ] Open CHA-com PR; CI green; merge
- [ ] Tag v1.15.0; goreleaser (~80 min); confirm multi-arch manifest; cluster roll

---

## Acceptance for 2.A

- Auto-applied action → outcome row visible in Qdrant within 1 cycle
- Approve click → outcome row with Approver populated from `X-Forwarded-User`
- Deny click → outcome row with Decision=denied
- Re-appearance within 5 min → outcome row with Result=reverted
- DigestPin proposal for a previously-reverted target → Rationale mentions the prior attempt
- `outcomes: recorded=N (…)` log line on every cycle

## Risk + mitigation

- **Qdrant write failures shouldn't block apply.** Recorder errors are logged, not propagated.
- **Outcome volume.** ~50 actions/day × 30 days = ~1500 entries; well within Qdrant comfort.
- **Race between recorder write + reader query.** Recorder uses Upsert (sync to Qdrant before returning); reader queries within next watch cycle (~1 min later). No race in practice.
