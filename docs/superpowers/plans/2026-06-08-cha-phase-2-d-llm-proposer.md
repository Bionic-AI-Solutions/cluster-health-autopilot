# Phase 2.D — LLM-Driven Proposer for Unmatched Diagnostics

**Status:** active — starts 2026-06-08 in local-only mode (no `git push` until credit return).

**Parent:** [2026-06-07-srenix-phase-2-master.md](2026-06-07-srenix-phase-2-master.md)

**Branch:** `phase2b/approve-remember-class` (stacking on the 2.B+2.C branch; will split into its own when the 2.B parts get pushed).

---

## Goal

Today's `FixProposerImpl.MatchFixer` returns `("", false)` for any diagnostic that doesn't match one of ~5 hard-coded fixer keywords. Those findings get a Slack post but no Apply button — they need a human to write a kubectl command from scratch.

Phase 2.D: when no deterministic proposer matches, fall back to an LLM proposer that:
1. Builds a prompt with the diagnostic + the operator's recent same-class outcome history (Phase 2.A.1)
2. Asks the LLM to return either a `pkgai.AIProposedAction` shape OR an explicit "no safe action" refusal
3. Validates the returned action through the existing `pkg/ai/validate.go` Kind whitelist + safe-shape checks (the same gate that `ApplyManifest` proposals already pass)
4. Refuses anything outside the whitelist — the LLM cannot escalate the safe-set

## Anti-goals

- LLM does NOT auto-apply. The output is always a click-to-fix URL (gated by the same approve-server pipeline).
- No new action kinds. The LLM picks from the existing closed enum (DeletePod, ApplyManifest, ProposePullRequest, etc.). If it tries to propose something new, validate.go rejects it.
- No multi-turn agent loops in v1. One prompt → one structured response → validate. Future phases may add reflection.

## Sub-tasks (TDD; bite-size)

### 2.D.1 — `LLMProposer` skeleton + happy-path test
- [ ] `ai/proposer/llm.go` with `LLMProposer{client, validator, memory, audit}`
- [ ] `Propose(ctx, diag) (*AIProposedAction, error)` — pure prompt builder + JSON response parser
- [ ] Failing test: stub LLM client returns a valid `DeletePod` action JSON; assert returned action passes Validate
- [ ] Implement, pass

### 2.D.2 — Validator rejects unknown ActionKind
- [ ] Failing test: stub LLM returns `ActionKind: "WipeCluster"`; assert proposer returns nil + error
- [ ] Implement (already done — pkg/ai/validate.go enforces this; just wire it)
- [ ] Pass

### 2.D.3 — Memory-grounded prompt
- [ ] Inject recent same-class outcomes into the prompt: "the operator has approved this exact action 30 times in the last 30 days with 90% cleared rate"
- [ ] Failing test: capture the prompt sent to the stub LLM; assert memory text is present when memory has matching outcomes
- [ ] Implement, pass

### 2.D.4 — Empty / unparseable LLM response
- [ ] Failing test: stub returns malformed JSON, returns "", returns explicit `{"action": "no_safe_action"}`
- [ ] All three return nil action with no error (silent skip — same shape as DigestPin)
- [ ] Implement, pass

### 2.D.5 — Soft-fail on LLM error
- [ ] Failing test: LLM client errors → Propose returns (nil, nil), audit-writes ai.llm.proposer_failed
- [ ] Implement, pass

### 2.D.6 — Wire into `proposeFixes` as the final fallback
- [ ] In `cmd/srenix-enterprise/ai_wiring.go::proposeFixes`: after FixProposer + DigestPin + VaultRunbook all return nil, AND the diagnostic has no ProposedPolicyYAML, invoke LLMProposer
- [ ] Build the URL minter + class URLs + autonomy hook the same way as FixProposer-derived actions
- [ ] Failing test: a diagnostic that none of the deterministic proposers match gets an LLM proposal in the proposalRecord
- [ ] Implement, pass

### 2.D.7 — Per-cycle observability
- [ ] Add `llm-proposer:` line to cycle output: `llm-proposer: cycle=N attempted=N succeeded=M rejected=R errored=E`
- [ ] Failing test, implement, pass

### 2.D.8 — Field-travels integration test
- [ ] Stub LLM + real validator + real autonomy engine; assert the LLM-proposed action ends up in the proposalRecord with valid URLs minted
- [ ] Build-tag `integration`

### 2.D.9 — Local build + cluster verify
- [ ] srenix-enterprise:1.15.2-dev1 image; CR patch; click an LLM-generated Approve URL on a real Slack post; verify action applied + recorded

### 2.D.10 — Doc + CHANGELOG entry

---

## Acceptance for 2.D

- A diagnostic outside the FixProposer keyword set gets an Apply button in Slack (was: ActionID only)
- The proposed action passes `pkg/ai/validate.go` (closed Kind whitelist + safe shape)
- Memory-grounded prompts cite per-class outcome history when available
- `llm-proposer:` cycle line tracks success vs reject vs error counts
- Soft-fail on LLM errors — cycle output never breaks

## Risk + mitigation

- **LLM proposes unsafe action.** The validator's closed enum + safe-shape check is the existing line of defense (Phase 1 already proved it on `ApplyManifest` for NetworkPolicy). Phase 2.D doesn't widen that gate; the LLM's output passes through it.
- **LLM cost spike.** Cap LLM invocations per cycle (e.g., 10) to bound monthly inference cost. Document at deploy time.
- **Stale memory grounding.** The 30-day window from 2.A.1's RecentOutcomesByClass already decays out old data. Reverts (2.A.2) drive the success rate down quickly.
- **JSON parse failures.** Phase 1.A's audit caught this class for the Qwen "thinking mode dumps in content" bug. Same guard: allow null content, schema-validate, soft-fail.
