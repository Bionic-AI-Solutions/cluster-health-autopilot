# Phase 2.A — RAG Memory READ Path on Action Outcomes

**Status:** active — execution started 2026-06-07; sub-plan REVISED 2026-06-08 after pre-execution survey discovered the WRITE path is already shipped.

**Parent:** [2026-06-07-cha-phase-2-master.md](2026-06-07-cha-phase-2-master.md)

**Branch:** `phase2a/rag-outcomes` (CHA-com)

---

## Goal (revised)

The original 2.A plan assumed both write and read paths needed building. **Pre-execution survey caught that the WRITE path is already wired end-to-end:**

| Site | Status | Evidence |
|---|---|---|
| `OutcomeRecorder` interface | ✅ exists | `ai/approval/executor.go:57` |
| Approve handler records | ✅ wired | `ai/approval/server.go:450` |
| Deny handler records | ✅ wired | `ai/approval/server.go:496` |
| Autonomy auto-apply records | ✅ wired | `cmd/cha-com/autonomy_engine.go:130` |
| `memoryOutcomeRecorder` adapter | ✅ exists | `cmd/cha-com/ai_wiring.go:300` |
| Helm/operator hook | ✅ exists | `ai_wiring.go::outcomeRecorder()` |

**The actual gap is the READ side**: nobody queries outcomes back from Qdrant. `Memory.Retrieve` exists but it's embedding-similarity by digest, not by-class or by-target lookup. That's what 2.B/2.C/2.D will need.

This revised 2.A ships the read helpers, wires DigestPin to use them, and adds revert detection + per-cycle observability.

## Anti-goals

- Don't rewrite the write path. It's done. Touching it only invites regression.
- Read helpers stay narrow: only what 2.B/2.C/2.D need. Future use-cases get their own helpers.

---

## Sub-tasks (TDD; bite-size) — revised numbering

### 2.A.1 — `Recent*` query helpers on `*Memory` (or new `OutcomeReader`)
- [ ] Decide placement: extend `*Memory` with `RecentOutcomesByClass(ctx, source, kind, since)` + `RecentOutcomesByTarget(ctx, target, since)` OR new `ai/memory/outcome_reader.go` wrapper. Lean toward extending `*Memory` (one fewer type).
- [ ] Failing test `TestMemory_RecentOutcomesByTarget` in `ai/memory/memory_test.go`: AppendSignal 5 outcomes for one target + 3 for another; assert query returns only the 5 sorted recent-first.
- [ ] Failing test `TestMemory_RecentOutcomesByClass`: same shape but filter by `(source, action_kind)` tuple.
- [ ] Implement both helpers using `QdrantRAG.List(ctx, rag.Query{Kind: rag.KindFindingOutcome, ...})` + post-filter (Qdrant payload-filter would be ideal but `Query` doesn't expose it yet; post-filter is fine for v1 since outcome volume is ~50/day).
- [ ] Run, pass.

### 2.A.2 — Revert detection in `cmd/cha-com/watch_cmd.go::tick()`
- [ ] After building the new diagnostic set for the cycle, for each newly-seen finding query `mem.RecentOutcomesByTarget(target, 24h)`.
- [ ] If a prior outcome's Verdict=cleared within 24h AND the same finding is back → the prior fix didn't stick. Append a follow-up signal with `Verdict="reverted"`.
- [ ] Failing test in `watch_cmd_test.go`: fake memory returns a 1h-old cleared outcome for the finding's target; assert a "reverted" signal is appended.
- [ ] Implement, pass.

### 2.A.3 — DigestPinProposer reads memory before proposing
- [ ] In `ai/proposer/digest_pin.go::Propose`: after building `workloadKey` but before opening the PR, query `reader.RecentOutcomesByTarget(podTarget, 7days)`.
- [ ] If a prior outcome's Verdict=reverted within 24h → include "previously attempted, was reverted" in the proposal's Rationale field.
- [ ] Failing test `TestDigestPin_MemoryAwareRationale_PriorRevert`: fake reader returns a reverted outcome; assert proposal's Rationale contains "previously attempted, was reverted".
- [ ] Failing test `TestDigestPin_MemoryAwareRationale_PriorSuccess`: fake reader returns a cleared outcome older than 7d; assert Rationale does NOT contain "previously attempted" (stale data).
- [ ] Implement (pass a Reader into the proposer's struct; thread through wiring), pass.

### 2.A.4 — Per-cycle observability log
- [ ] In `cmd/cha-com/watch_cmd.go::tick()`: at cycle end, query `mem.RecentOutcomesByClass` for the current cycle window (~1 min) and log:
  `outcomes: cycle=N applied=A approved=B denied=C reverted=D`
- [ ] Failing test on the log line presence + counts.
- [ ] Implement, pass.

### 2.A.5 — Field-travels-end-to-end integration test
- [ ] Per the master plan's Phase-1-lesson refinement: `TestOutcome_WrittenByApprovalServer_ReadBackByMemory`.
- [ ] Set up an in-process approval-server with a real (test) QdrantRAG (or fake but using the real Outcome serialization helpers).
- [ ] Approve a proposal → query `mem.RecentOutcomesByTarget(target, 1h)` → assert exactly one Outcome returned with the expected fields.
- [ ] Lives in `ai/approval/server_integration_test.go` (build tag `integration`).
- [ ] Catches the DriftReport.spec.remediation class of bug: write side green + read side green + integration silently dropping the field.

### 2.A.6 — Wire memory into DigestPin's construction in `watch_cmd.go`
- [ ] In `cmd/cha-com/watch_cmd.go::RunE`: pass the `*Memory` instance into `dp.buildDigestPinProposer(...)`.
- [ ] Adjust `digest_pin_wiring.go::buildDigestPinProposer` signature to accept a `MemoryReader` interface.
- [ ] Failing test on the wiring (similar to existing `TestDigestPinFlags_BuildDigestPinProposerWiresEverything`).
- [ ] Implement, pass.

### 2.A.7 — Local build + dev tag + cluster verify
- [ ] `CGO_ENABLED=0 go build -o /tmp/cha-com ./cmd/cha-com`
- [ ] `docker build` + push `docker4zerocool/cha-com:1.15.0-dev1`
- [ ] Patch CR `ai.image.tag=1.15.0-dev1`; wait rollout
- [ ] Click Approve/Deny on real Slack URLs in the wild; verify the next cycle's log shows `outcomes: applied=N…`
- [ ] Verify DigestPin proposal rationale (when one fires) mentions prior outcome if any

### 2.A.8 — Open CHA-com PR + tag canonical release
- [ ] PR; CI green; merge
- [ ] Tag `v1.15.0`; goreleaser (~80 min); confirm multi-arch manifest assembled (manual fallback per master-plan refinement); roll cluster

---

## Acceptance for 2.A (revised)

- DigestPin proposal for a recently-reverted target → Rationale carries the prior-revert note
- Watcher log shows `outcomes: cycle=N applied=A approved=B denied=C reverted=D` once per cycle
- A click-Approve outcome is queryable back via `RecentOutcomesByTarget` within 1 cycle
- Revert detection: a finding that was Cleared in cycle N and re-appears in cycle N+1 writes a reverted signal

## Why this is enough for 2.B/2.C/2.D to start

- **2.B (Approve+remember class)**: needs to read per-class outcome history → uses `RecentOutcomesByClass`. ✅
- **2.C (confidence model)**: needs success rate per `(source, action_kind, namespace)` → trivially derived from `RecentOutcomesByClass` + a namespace filter. ✅
- **2.D (LLM proposer)**: prompts include "you tried these recently with this outcome" → uses `RecentOutcomesByTarget`. ✅

## Risk + mitigation

- **Qdrant List performance.** Post-filter pulls all outcomes per query. At ~50/day × 30d = 1500 entries, scanning is microseconds. If volume grows >10K, add Qdrant payload-index on `source`+`action_kind`+`target`.
- **Cold start.** No prior outcomes = empty result = proposers behave exactly as today. Zero regression risk for first deploy.
- **Memory reader nil safety.** Memory may be nil (RAG disabled). All `Recent*` helpers must return `(nil, nil)` on nil receiver, not panic.
