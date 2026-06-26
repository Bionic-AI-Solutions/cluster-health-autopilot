# Phase 2.B — Approve+Remember-Class Workflow

**Status:** active — starts 2026-06-08 once 2.A merges + v1.15.0 lands on the cluster.

**Parent:** [2026-06-07-srenix-phase-2-master.md](2026-06-07-srenix-phase-2-master.md)

**Branch:** `phase2b/approve-remember-class` (Srenix Enterprise), with a paired OSS branch when slack rendering needs operator-server endpoints (TBD per task 2.B.4).

---

## Goal

Each Slack message's per-finding row gains TWO new buttons next to the existing **Approve / Deny** pair:

```
✅ Approve · ❌ Deny · 🧠 Approve+remember class · 🔕 Silence class (24h)
```

- **🧠 Approve+remember class** → (a) executes this action like Approve, (b) writes a class-level **PolicyEntry** to RAG so the next 7 days of findings in this `(source, action_kind, message-pattern)` class auto-approve silently, (c) the post-cycle observability log surfaces a new line: `policies: applied=N matched=M muted=K`.
- **🔕 Silence class (24h)** → creates a class-scoped Silence CR (matcher = `(source, message-regex)` derived from the clicked finding), so the next 24h of matching findings skip Slack entirely. Different from per-subject Silence (Phase 1.A's existing tool — that suppresses ONE pod by exact subject; this suppresses an entire CLASS).

## Anti-goals

- No web UI. Slack-button-only.
- No regex DSL exposed to operators. The class is inferred from the clicked item: source verbatim, message-pattern by literal common-prefix or analyzer-declared template.
- No cross-cluster policy federation. Policies are per-cluster (matches the cluster-isolated RAG store from `pkg/rag/types.go`).

## Why now

The 2.A foundation makes per-class policy queryable. Without policy reads, autonomy would have to decide based on a single click; this PR makes one click apply to a whole class.

## Sub-tasks (TDD; bite-size)

### 2.B.1 — `PolicyEntry` type in `pkg/ai/policy.go` (OSS)
- [ ] Define `PolicyEntry{Source, ActionKind, MessagePattern, ExpiresAt, ClickedBy, ClickedAt}`
- [ ] Failing test in `pkg/ai/policy_test.go`: `TestPolicyEntry_Matches` — same source + action_kind + message-substring → match; mismatched source → no match; expired entry → no match
- [ ] Implement `Matches(diag, kind) bool`
- [ ] Run, pass

### 2.B.2 — `PolicyStore` over RAG (Srenix Enterprise)
- [ ] `ai/policy/store.go`: `Store.Put(ctx, PolicyEntry) error` + `Store.Active(ctx, source, kind) ([]PolicyEntry, error)`
- [ ] Backs onto QdrantRAG with `Kind = "policy"` (new constant in OSS rag/types.go)
- [ ] Failing tests: round-trip Put → Active; expired entries filtered out; nil-store-no-op
- [ ] Implement, pass

### 2.B.3 — Autonomy engine consults policy
- [ ] In `pkg/ai/autonomy.go::DecideAutonomy`: BEFORE existing static gate, query `PolicyStore.Active(source, kind)` and check `.Matches(diag, kind)` on each
- [ ] On match: return `Decision{AutoApply: true, Reason: "matches policy <ID> set by <user> on <date>"}`
- [ ] Failing test in `pkg/ai/autonomy_test.go`: fake store returns a matching policy → DecideAutonomy returns AutoApply=true; no-match → unchanged behavior
- [ ] Implement, pass

### 2.B.4 — `/approve-class` + `/deny-class` + `/silence-class` routes on approval-server (Srenix Enterprise)
- [ ] `ai/approval/server.go`: add 3 handlers
- [ ] Each verifies the JWT (same as `/approve`), then:
  - `/approve-class`: write PolicyEntry{ExpiresAt: now+7d, ClickedBy: hdrUser}, then execute the action like `/approve`
  - `/deny-class`: just write a deny policy (no execute)
  - `/silence-class`: create a Silence CR scoped to the class
- [ ] Failing tests: 1 per route
- [ ] Implement, pass

### 2.B.5 — Slack render: 4-button row (Srenix Enterprise)
- [ ] `cmd/srenix-enterprise/ai_slack_digest.go`: extend `renderApproveDenyURLs` (or equivalent) to emit 4 markdown links
- [ ] Each link's URL is a separately-minted signed JWT scoped to the operation
- [ ] Failing test: rendered output contains all 4 click-targets
- [ ] Implement, pass

### 2.B.6 — Slack render parity in OSS watcher (OSS)
- [ ] `internal/report/routing.go::renderAIBlocks`: same 4-button row when the approve/deny URLs are present
- [ ] Falls back gracefully when the class-URL fields are empty (legacy proposers that don't mint them yet)
- [ ] Failing test, implement, pass

### 2.B.7 — JWT scope check
- [ ] Mint each class-URL with a `scope` claim ("class-approve", "class-deny", "class-silence") so an `/approve` URL can't be replayed against `/approve-class`
- [ ] Approval-server verifier rejects scope-mismatched tokens
- [ ] Failing tests for each route's scope check
- [ ] Implement, pass

### 2.B.8 — Per-cycle policy log
- [ ] Watcher's existing `outcomes: cycle=N …` line gains a sibling: `policies: cycle=N active=A muted=M`
- [ ] `active` counts non-expired entries; `muted` counts diagnostics suppressed by an active policy this cycle
- [ ] Failing test, implement, pass

### 2.B.9 — Class-scoped Silence CR support in OSS
- [ ] `pkg/silence/lister.go`: extend to match on `(source, messageRegex)` not just exact subject
- [ ] CRD `srenix.ai/Silence` v1alpha1 gains optional `spec.matcher.source` + `spec.matcher.messagePattern`
- [ ] Operator + Helm CRD update
- [ ] Failing test, implement, pass

### 2.B.10 — Field-travels integration test
- [ ] Methodology rule: prove the click → policy write → next-cycle autonomy → autoApply path travels end-to-end
- [ ] In-process approval-server + in-process PolicyStore + fake autonomy gate; click `/approve-class`, then run a diag that should match, assert it auto-applied
- [ ] Build-tag `integration`

### 2.B.11 — Local build + cluster verify
- [ ] Build srenix-enterprise + ship dev tag
- [ ] Click "Approve+remember class" on a real Slack message
- [ ] Verify next cycle's matching findings auto-apply silently
- [ ] Verify `policies: …` log line shows `active=1`

### 2.B.12 — Open PR + release
- [ ] Srenix Enterprise PR; CI green; merge; tag `v1.16.0`; goreleaser; cluster roll
- [ ] Pair with OSS PR (Silence CRD extension + Slack render parity); tag OSS `v1.21.0`

---

## Acceptance for 2.B

- Operator clicks "Approve+remember class" on a Slack message → action executes AND a PolicyEntry persists in RAG
- Within the next ~24h, matching findings in the same `(source, action_kind, message-pattern)` class auto-apply silently (no Slack post for that class)
- Operator clicks "🔕 Silence class (24h)" → class-scoped Silence CR appears + matching findings drop out of Slack for 24h
- New per-cycle log line: `policies: cycle=N active=A muted=M` matches reality
- Field-travels integration test green (`go test -tags=integration ./cmd/srenix-enterprise/...`)

## Risk + mitigation

- **Class-policy too broad.** A single "Approve+remember" could auto-apply across many distinct findings. Mitigation: the message-pattern is the literal common-prefix of the clicked item's message, not a regex. Operators see in Slack exactly what's about to be matched.
- **Replay risk.** Class-URL JWTs are still one-shot. The PolicyEntry itself is the "remembered" half; the URL can't be re-clicked.
- **Silence-CR class scope vs existing per-subject.** Both coexist; the lister tries class match first, falls back to subject match. No behavioral change for existing per-subject silences.
