# Srenix Phase 0 + Phase 1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Retire 7 dev-tag fixes into canonical releases (Phase 0), then close the top 5 user-visible gaps revealed in the 2026-06-04 adversarial review (Phase 1) — bringing pillars 2/3/4/5 each up one full grade.

**Architecture:**
- Phase 0 is release hygiene — clean-cut PRs against `main` for each fix already in working dev tags, then goreleaser for canonical multi-arch images, then roll cluster.
- Phase 1 layers 5 independently-shippable features: srenix-enterprise→Slack bridge, OSS analyzer placeholder substitution, Forge rate-limiter + workload-key dedup, OpenProject delivery, per-cycle delta render.
- Every Phase 1 deliverable is additive — no breaking changes to OSS public surface; srenix-enterprise side gets new flags (default off where risky).

**Tech Stack:**
- Go 1.26 (agentic-sre OSS + Srenix Enterprise proprietary, two separate repos)
- Helm 3 chart at `charts/agentic-sre/`
- Kubernetes operator (controller-runtime) at `internal/operator/`
- Qdrant for RAG (`bionic-rag` StatefulSet, `srenix-rag` collection)
- GitHub REST API v3 via `ai/forge/forge.go` for PR creation
- Slack incoming webhooks (write-only) for delivery
- ExternalSecrets Operator → Vault for secret material
- `goreleaser v2` for release builds (~80 min multi-arch)
- Local docker build + push for dev iteration (per `local-docker-build-iteration` memory)

**Repos involved:**
- OSS: `/home/skadam/Srenix/agentic-sre` (`Srenix/agentic-sre`)
- Paid: `/home/skadam/Srenix/Srenix Enterprise` (`Srenix/Srenix Enterprise`)
- Deploy: `/home/skadam/deploy` (`Srenix/Deploy`)
- K8s infrastructure: subdir of Deploy (`/home/skadam/deploy/k8s-infrastructure`)

**Branch convention:** `phase0/<short-desc>` and `phase1/<deliverable>-<sub-desc>` for trackability.

---

## File Structure Decisions

### Phase 0 — touches existing files only; no new modules.

### Phase 1 — new modules + their tests:

**srenix-enterprise (1.A bridge):**
- `cmd/srenix-enterprise/ai_slack_digest.go` — NEW (~200 LOC). Renderer that produces `🤖 AI Tier Activity` SlackPayload from `proposalRecord[]`. Three sections: Auto-applied (autonomy fired) / Awaiting approval (autonomy declined for safety) / Declined (autonomy declined with reason). Includes chunking + sort to stay under Slack's 40K char attachment cap.
- `cmd/srenix-enterprise/ai_slack_digest_test.go` — NEW (~250 LOC). Six unit tests covering all three sections, chunk-boundary, dedup, and no-activity-no-post behavior.
- `cmd/srenix-enterprise/ai_slack_wiring.go` — NEW (~100 LOC). `aiSlackFlags` struct, `register()`, `buildAISlackPoster()`.
- `cmd/srenix-enterprise/ai_slack_wiring_test.go` — NEW (~80 LOC). Flag-binding + builder tests.
- `cmd/srenix-enterprise/watch_cmd.go` — MODIFY (~30 LOC added). Thread aiSlackPoster into watchLoop; call after autonomy.Consider in tick().

**OSS (1.B placeholders):**
- `internal/diagnose/capacity_drift.go` — MODIFY (~30 LOC). PVC expansion-stuck remediation looks up StorageClass + branches text based on allowVolumeExpansion.
- `internal/diagnose/capacity_drift_test.go` — MODIFY (~50 LOC). Add 2 tests for both branches.
- `internal/diagnose/security_drift.go` — MODIFY (~25 LOC). digest-pin remediation includes actual image tag + suggested `@sha256:` recipe (uses prior outcome data from RAG if available, otherwise generic).
- `internal/diagnose/security_drift_test.go` — MODIFY (~30 LOC). Add test for substituted text.
- `internal/diagnose/netpol_proposer.go` — already produces concrete YAML; verify + add test if missing.
- `internal/diagnose/dns_chain_drift.go` — MODIFY (~20 LOC). Include actual Ingress + Service paths in remediation.
- `internal/diagnose/dns_chain_drift_test.go` — MODIFY (~25 LOC).

**srenix-enterprise (1.C rate-limiter + dedup):**
- `ai/forge/forge.go` — MODIFY (~80 LOC). Add `RateLimiter` field (golang.org/x/time/rate.Limiter). All HTTP methods acquire 1 token before sending. On HTTP 403 with secondary-rate-limit body: sleep + retry once. On HTTP 429: respect Retry-After header.
- `ai/forge/forge_test.go` — MODIFY (~80 LOC). Test 100-call burst rate-limited; 403-with-secondary-rate-limit-body retries.
- `ai/proposer/digest_pin.go` — MODIFY (~25 LOC). Add per-cycle workload-key dedup map. Caller responsibility to reset between cycles.
- `ai/proposer/digest_pin_test.go` — MODIFY (~40 LOC). Test: 3 calls for same workloadKey → 1 forge interaction.
- `cmd/srenix-enterprise/watch_cmd.go` — MODIFY (~10 LOC). Reset proposer dedup map at top of each tick.

**OSS + Srenix Deploy (1.D OpenProject):**
- `api/v1alpha1/agenticsre_types.go` — MODIFY (~20 LOC). Add `spec.ticketing.openProject{url, secretName}`.
- `api/v1alpha1/zz_generated.deepcopy.go` — REGENERATE.
- `internal/operator/builders.go` — MODIFY (~30 LOC). CR field → deployment env wiring.
- `charts/agentic-sre/templates/watcher-deployment.yaml` — MODIFY (~15 LOC). Add env: OPENPROJECT_URL, OPENPROJECT_API_TOKEN.
- `cmd/srenix/main.go` — MODIFY (~15 LOC). Bind --openproject-url + --openproject-token-env flags; instantiate sink.
- `pkg/ticketing/openproject/integration_test.go` — NEW (~150 LOC). Integration test against mock OpenProject server.
- **External**: `/home/skadam/deploy/k8s-infrastructure/srenix-paid-tier/04-externalsecret-srenix-openproject.yaml` — NEW. ESO for OpenProject API token.
- **External Vault**: `vault kv put secret/t6-apps/srenix/openproject-config api_token=<user-provided>`.

**OSS (1.E per-cycle delta render):**
- `internal/report/routing.go` — MODIFY (~40 LOC). Add "🆕 New this cycle" section above "⚠️ Diagnostics"; collapse stable findings into summary line.
- `internal/report/routing_test.go` — MODIFY (~60 LOC). 3 new tests covering new/stable/resolved sectioning.
- `internal/watcher/watcher.go` — MODIFY (~10 LOC). Thread `firstSeenCycle` through seenEntry to support stable-since-N-cycles summary.

---

## Phase 0 — Stabilize (3 sessions of work, ~3 days wall-clock)

> **Wait condition between tasks:** goreleaser takes ~80 min. Do other tasks while it builds.

### Task 0.1 — Open OSS PR #A: `feat(watcher): seenEntryToDeltaDiag helper`

**Branch:** `phase0/seen-entry-helper`

**Source code:** already on local branch `slack-toPostDiags-approval-fields` (verified in 1.18.3-dev1).

**Steps:**
- [ ] Checkout the existing branch: `git -C /home/skadam/Srenix/agentic-sre checkout slack-toPostDiags-approval-fields`
- [ ] Rebase onto latest main: `git pull --rebase origin main`
- [ ] Push the branch: `git push -u origin slack-toPostDiags-approval-fields`
- [ ] Open PR via gh: title `feat(watcher): collapse toPostDiags + allActive into seenEntryToDeltaDiag helper`, body explains 2026-06-04 outage diagnosed during session + regression test rationale
- [ ] Wait for CI green (~16 min for build+test, ~6 min for lint, ~20s for oss-freshness)
- [ ] Merge with squash + delete branch

**Verification:** PR merged, main now contains `seenEntryToDeltaDiag` function in `internal/watcher/watcher.go`.

### Task 0.2 — Open OSS PR #B: `feat(report): SplitCriticalPayloads chunking + actionable-first sort + diagnostic logs`

**Branch:** `phase0/slack-chunking-sort`

**Source code:** already on local branch with dev2/dev3/dev4/dev5 changes (verified in 1.18.3-dev5).

**Steps:**
- [ ] On the branch, ensure the commit includes: `SplitCriticalPayloads` helper + sort + log lines (the dev3 instrumentation) + 3 regression tests
- [ ] Rebase onto post-PR-A main
- [ ] Push, open PR via gh
- [ ] Wait for CI
- [ ] Merge

### Task 0.3 — Open OSS PR #C: `fix(releasesrc): Detect falls through on any Helm probe error`

**Branch:** `phase0/detect-fallthrough`

**Source code:** already on local branch `helm-probe-fallthrough-on-403` (verified in 1.18.3-dev6).

**Steps:**
- [ ] Push branch, open PR
- [ ] Wait for CI
- [ ] Merge

### Task 0.4 — Tag + release `agentic-sre v1.18.3`

**Steps:**
- [ ] On main, after all three PRs merged: `git tag -a v1.18.3 -m "..."`
- [ ] Push tag: `git push origin v1.18.3`
- [ ] Wait for goreleaser run (~80 min). Spawn poller in background.
- [ ] Verify image: `docker manifest inspect docker4zerocool/agentic-sre:1.18.3` shows multi-arch list.

### Task 0.5 — Open srenix-enterprise PR #D: `fix(ai/memory): paginated Qdrant scroll`

**Source:** dev1 changes already on `/home/skadam/Srenix/Srenix Enterprise` branch `qdrant-scroll-pagination`.

**Steps:**
- [ ] Push + open PR
- [ ] Wait for CI (~16 min)
- [ ] Merge

### Task 0.6 — Open srenix-enterprise PR #E: `feat(ai/forge): per-forge read cache TTL=5m`

**Source:** dev3 changes on local branch `forge-read-cache`.

**Steps:**
- [ ] Push + open PR (rebased onto post-D main)
- [ ] Wait for CI
- [ ] Merge

### Task 0.7 — Tag + release `srenix-enterprise v1.11.4`

**Steps:**
- [ ] `git tag -a v1.11.4`
- [ ] Push tag
- [ ] Wait for goreleaser (~80 min)
- [ ] Verify image

### Task 0.8 — Roll cluster to canonical tags

**Steps:**
- [ ] `helm upgrade srenix srenix/agentic-sre --reset-then-reuse-values --version 1.18.3 --set image.tag=1.18.3`
- [ ] `kubectl patch agenticsre/bionic -n agentic-sre --type=merge --patch='{"spec":{"image":{"tag":"1.18.3"},"ai":{"image":{"tag":"1.11.4"}}}}'`
- [ ] Wait for both rollouts (bionic-watcher + bionic-aiwatch)
- [ ] Verify: `kubectl get deploy -n agentic-sre -o jsonpath=...` shows all on canonical tags
- [ ] Wait for one cycle to complete; verify ≥1 PR created
- [ ] Commit + document: add memory entry summarizing the 7 session fixes

**Phase 0 done when:** cluster runs `agentic-sre:1.18.3` + `srenix-enterprise:1.11.4`, all dev tags retired from use, one verification cycle completed cleanly.

---

## Phase 1 — Critical UX Gaps (~3 weeks)

### Deliverable 1.A — srenix-enterprise → Slack Bridge

**Branch:** `phase1/srenix-enterprise-slack-bridge`

#### Task 1.A.1 — Write failing test: renderAISlackDigest auto-applied section
- [ ] Create `cmd/srenix-enterprise/ai_slack_digest_test.go` with `TestRenderAISlackDigest_AutoAppliedSectionListsActionedItems`
- [ ] Test constructs 3 proposalRecords where `Autonomy.AutoApply=true` + `Action.PullRequestURL` set
- [ ] Asserts payload contains `🔧 Auto-applied (3)` section header
- [ ] Asserts each proposal's PR URL appears as markdown link
- [ ] Run test to confirm it fails (helper doesn't exist yet)

#### Task 1.A.2 — Implement renderAISlackDigest minimal scaffold
- [ ] Create `cmd/srenix-enterprise/ai_slack_digest.go`
- [ ] Define `SlackPayload` (local, mirrors `internal/report.SlackPayload` shape; CANNOT import OSS internal pkg)
- [ ] Define `renderAISlackDigest(proposals []proposalRecord) SlackPayload`
- [ ] Implement auto-applied section only (sufficient to make test 1.A.1 pass)
- [ ] Run test to confirm it passes

#### Task 1.A.3 — Write failing test: awaiting-approval section
- [ ] Add `TestRenderAISlackDigest_AwaitingApprovalSectionHasApproveDenyLinks`
- [ ] Proposals where `Autonomy=nil OR Autonomy.AutoApply=false AND Autonomy.Reason starts with "human approval required"`
- [ ] Asserts `✅ <approveURL|Approve> · ❌ <denyURL|Deny>` markdown links present
- [ ] Run, confirm failure

#### Task 1.A.4 — Implement awaiting-approval section
- [ ] Extend `renderAISlackDigest` to include this section
- [ ] Run tests, both 1.A.1 and 1.A.3 should pass

#### Task 1.A.5 — Write failing test: declined section
- [ ] Add `TestRenderAISlackDigest_DeclinedSectionShowsReason`
- [ ] Proposals where `Autonomy.AutoApply=false AND Reason ≠ "human approval required"` (e.g., circuit breaker open, action kind not allowed)
- [ ] Asserts section header + per-item reason
- [ ] Run, confirm failure

#### Task 1.A.6 — Implement declined section
- [ ] Extend renderer
- [ ] Run tests, all 3 pass

#### Task 1.A.7 — Write failing test: chunking behavior
- [ ] Add `TestRenderAISlackDigest_ChunksOverSlackLimit`
- [ ] 200 large proposals (each ~600 bytes rendered)
- [ ] Asserts ≥2 returned payloads; each under 35K chars
- [ ] Run, confirm failure (renderer returns 1 huge payload today)

#### Task 1.A.8 — Implement chunking
- [ ] Refactor renderer to return `[]SlackPayload`
- [ ] Adopt the same chunking pattern as OSS `SplitCriticalPayloads` (local copy — different package)
- [ ] Update earlier tests to assert at least one payload contains expected content
- [ ] Run, all 4 tests pass

#### Task 1.A.9 — Write failing test: no activity → no payload
- [ ] Add `TestRenderAISlackDigest_NoActivity_ReturnsEmpty`
- [ ] Empty proposal list → 0 payloads returned
- [ ] Run, confirm

#### Task 1.A.10 — Implement no-activity guard
- [ ] Early-return on empty input
- [ ] Run, all 5 tests pass

#### Task 1.A.11 — Write failing test: wiring
- [ ] Create `cmd/srenix-enterprise/ai_slack_wiring_test.go`
- [ ] `TestAISlackFlags_RegisterBindsFlags` — invoke `aiSlackFlags.register(...)` with a captured slice; assert `--ai-slack-url-env` registered
- [ ] `TestAISlackFlags_BuildPosterFromEnv` — set env, invoke buildAISlackPoster, assert non-nil poster
- [ ] `TestAISlackFlags_BuildPosterNoEnv_ReturnsNil` — no env → nil poster (feature off)
- [ ] Run, confirm failure

#### Task 1.A.12 — Implement aiSlackFlags + buildAISlackPoster
- [ ] Create `cmd/srenix-enterprise/ai_slack_wiring.go`
- [ ] Mirror the digestPinFlags pattern
- [ ] `aiSlackPoster` struct with `webhookURL string` + `Post(ctx, []SlackPayload) error` method
- [ ] Run, tests pass

#### Task 1.A.13 — Wire into watch_cmd
- [ ] Modify `cmd/srenix-enterprise/watch_cmd.go`:
  - Add `aiSlackPoster *aiSlackPoster` to watchLoop struct
  - In RunE: bind flags via aiSlackFlags.register; build poster; thread into loop
  - In `tick()` after autonomy.Consider loop: render proposals → poster.Post(payloads); log post outcome
- [ ] Run full test suite: `go test ./...`

#### Task 1.A.14 — Local build + push dev tag
- [ ] Build: `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /tmp/srenix-enterprise-1.12.0-dev1 ./cmd/srenix-enterprise`
- [ ] Wrap in distroless: docker build + push `docker4zerocool/srenix-enterprise:1.12.0-dev1`

#### Task 1.A.15 — Deploy to cluster + verify
- [ ] Patch CR: `spec.ai.image.tag=1.12.0-dev1` + add `--ai-slack-url-env=SLACK_CRITICAL_URL` to `spec.ai.extraArgs`
- [ ] Wait rollout
- [ ] Wait one watch cycle (~5 min)
- [ ] Verify: a new Slack message with `🤖 AI Tier Activity` header appears in #ceph-critical
- [ ] Verify: message contains the 11 PRs from this morning under "Auto-applied"

#### Task 1.A.16 — Open PR + tag canonical release
- [ ] Open srenix-enterprise PR; CHANGELOG entry; CI green; merge
- [ ] Tag `v1.12.0`; goreleaser (~80 min); deploy

---

### Deliverable 1.B — OSS Analyzer Placeholder Substitution

**Branch:** `phase1/analyzer-placeholder-substitution`

> Per-analyzer tasks; each is independent so they can be parallelized.

#### Task 1.B.1 — Substitute placeholders in capacity_drift PVC remediation
- [ ] Failing test in `capacity_drift_test.go`: feed a PVC with StorageClass `rook-ceph-block`, snapshot Source providing allowVolumeExpansion=true; assert remediation contains literal "rook-ceph-block allows expansion = true" not "<name>"
- [ ] Implement lookup in `capacity_drift.go:checkPVCExpansionStuck`
- [ ] Run test, pass
- [ ] Add second test: allowVolumeExpansion=false branch
- [ ] Implement branch text
- [ ] Commit

#### Task 1.B.2 — Substitute placeholders in security_drift digest-pin
- [ ] Failing test: pod with image `docker4zerocool/foo:v1`; assert remediation contains "Pin to @sha256:<digest>" with actual digest (or generic-instruction fallback if no observed digest available)
- [ ] Implement (use workload RAG entry if available, else fall back to current generic text)
- [ ] Commit

#### Task 1.B.3 — Substitute placeholders in dns_chain_drift
- [ ] Failing test: Ingress/Service mismatch; assert remediation includes actual Ingress name + Service name + suggested fix command
- [ ] Implement
- [ ] Commit

#### Task 1.B.4 — Audit + substitute remaining analyzers
- [ ] `grep -rn '<name>\|<placeholder>\|<image>' internal/diagnose/*.go` to enumerate remaining
- [ ] For each: write failing test → implement → commit
- [ ] Final grep returns 0 hits

#### Task 1.B.5 — Open PR + release
- [ ] Open OSS PR; CI; merge; tag `v1.19.0`; goreleaser; deploy

---

### Deliverable 1.C — Forge Rate-Limiter + Workload-Key Dedup

**Branch:** `phase1/forge-rate-limiter-and-dedup`

#### Task 1.C.1 — Add rate limiter dependency
- [ ] Run `go get golang.org/x/time/rate` in srenix-enterprise
- [ ] `go mod tidy`
- [ ] Commit

#### Task 1.C.2 — Failing test: 5000-token-per-hour cap
- [ ] In `ai/forge/forge_test.go`: simulate 100 GET calls in 100ms via mocked HTTP server
- [ ] Assert duration ≥ X based on rate limit (using a low rate cap like 10 tokens/sec for the test)
- [ ] Run, confirm failure

#### Task 1.C.3 — Add RateLimiter field + acquisition
- [ ] Add `RateLimiter *rate.Limiter` to GitHubForge
- [ ] `NewGitHubForge`: default to `rate.NewLimiter(rate.Limit(5000.0/3600.0), 50)` (5000/hour, burst 50)
- [ ] Insert `g.RateLimiter.Wait(ctx)` at top of `g.get()` and `g.put()` and `g.post()` etc.
- [ ] Run test, passes

#### Task 1.C.4 — Failing test: HTTP 403 with secondary-rate-limit body → retry once
- [ ] Mock server returns 403 with body `{"message":"You have exceeded a secondary rate limit"}` on first call, 200 on second
- [ ] Assert: GetFileContent succeeds after 1 retry; backoff observed
- [ ] Run, confirm failure

#### Task 1.C.5 — Implement secondary-rate-limit retry
- [ ] In `g.get()`: detect 403 with "secondary rate limit" in body → sleep configurable interval + retry once
- [ ] Run test, pass

#### Task 1.C.6 — Failing test: HTTP 429 honors Retry-After header
- [ ] Mock server returns 429 with `Retry-After: 2` header
- [ ] Assert retry observed after 2 seconds
- [ ] Run, confirm failure

#### Task 1.C.7 — Implement Retry-After
- [ ] Parse Retry-After header on 429; sleep + retry
- [ ] Run test, pass

#### Task 1.C.8 — Failing test: per-cycle workload-key dedup
- [ ] In `ai/proposer/digest_pin_test.go`: instantiate proposer, call Propose 3 times for same workloadKey
- [ ] Assert only 1 forge interaction (via mocked forge call counter)
- [ ] Run, confirm failure

#### Task 1.C.9 — Implement per-cycle dedup
- [ ] Add `proposedThisCycle map[string]bool` to DigestPinProposer
- [ ] Top of Propose(): if already-proposed, return (nil, nil)
- [ ] Add `ResetCycle()` method for caller
- [ ] Run test, pass

#### Task 1.C.10 — Wire ResetCycle into watch_cmd tick
- [ ] In `cmd/srenix-enterprise/watch_cmd.go::tick()`: at top, call `l.digestPin.ResetCycle()` if non-nil
- [ ] Run full test suite

#### Task 1.C.11 — Local build + deploy + verify
- [ ] Build, push dev tag
- [ ] Roll cluster
- [ ] Verify: cycle log shows reduced proposer call count (3 replicas → 1 proposer call)
- [ ] Verify: GitHub API rate-limit graph stays well under 5000/hour

#### Task 1.C.12 — Open PR + release
- [ ] CI green; merge; tag `v1.12.1`; goreleaser; deploy

---

### Deliverable 1.D — OpenProject Delivery to Production

**Branch:** `phase1/openproject-delivery-live`

**External pre-requisites (USER ACTION REQUIRED before code work):**
- [ ] User confirms target OpenProject URL (e.g., `https://openproject.srenix.ai`)
- [ ] User provisions OpenProject API token (with permission to create work packages in target project)
- [ ] User stores token: `vault kv put secret/t6-apps/srenix/openproject-config api_token=<value>` (uses kv patch if other keys exist)

#### Task 1.D.1 — Failing test: CR validates openProject sub-spec
- [ ] In `api/v1alpha1/agenticsre_types_test.go`: test that `spec.ticketing.openProject{url, secretName}` validates correctly
- [ ] Run, confirm failure (field doesn't exist)

#### Task 1.D.2 — Add openProject sub-spec to CR
- [ ] Modify `api/v1alpha1/agenticsre_types.go` to add ticketing.openProject
- [ ] `make generate` (regenerates deepcopy)
- [ ] Update `bundle/manifests/srenix.ai_agenticsres.yaml` CRD schema
- [ ] Run test, pass

#### Task 1.D.3 — Failing test: operator builds watcher with openproject env vars
- [ ] In `internal/operator/builders_test.go`: test that CR with `spec.ticketing.openProject.url=X` produces deployment with env OPENPROJECT_URL=X
- [ ] Run, confirm failure

#### Task 1.D.4 — Wire CR field → deployment env in operator
- [ ] Modify `internal/operator/builders.go::buildWatcherDeployment` to add env when openProject configured
- [ ] Run test, pass

#### Task 1.D.5 — Failing test: cmd/srenix binary instantiates OpenProject sink on flag
- [ ] In a new test for cmd/srenix: with --openproject-url + --openproject-token-env set, assert RouteTickets receives a non-nil OpenProject sink
- [ ] Run, confirm failure

#### Task 1.D.6 — Wire --openproject-* flags in cmd/srenix/main.go
- [ ] Bind flags via cobra
- [ ] Instantiate `openproject.NewClient(url, token)` and pass to RouteTickets configuration
- [ ] Run test, pass

#### Task 1.D.7 — Integration test against mock OpenProject server
- [ ] Create `pkg/ticketing/openproject/integration_test.go` if not present
- [ ] Spin up httptest.Server that mimics OpenProject API contract for work-package create
- [ ] Test: given a DeltaDiag, the sink POSTs the right shape; gets back a work-package URL; emits audit
- [ ] Run, pass

#### Task 1.D.8 — Helm chart updates
- [ ] Modify `charts/agentic-sre/templates/watcher-deployment.yaml` to include OPENPROJECT_URL + OPENPROJECT_API_TOKEN env vars (from secretKeyRef)
- [ ] Update chart values.yaml with default off + documentation

#### Task 1.D.9 — Provision ESO in user's k8s-infrastructure
- [ ] Create `/home/skadam/deploy/k8s-infrastructure/srenix-paid-tier/04-externalsecret-srenix-openproject.yaml`
- [ ] Wire to `secret/t6-apps/srenix/openproject-config` Vault path
- [ ] `kubectl apply`; verify ESO sync ready

#### Task 1.D.10 — Patch CR to enable
- [ ] `kubectl patch agenticsre/bionic ... --patch '{"spec":{"ticketing":{"openProject":{"url":"...","secretName":"srenix-openproject-token"}}}}'`
- [ ] Wait rollout

#### Task 1.D.11 — Live verification
- [ ] Force a fresh finding (e.g., re-delete storethesoup DriftReport)
- [ ] Wait one watcher cycle
- [ ] Verify: work-package appears in user's OpenProject instance with finding context
- [ ] Verify: subsequent occurrences update (not duplicate) the same work-package

#### Task 1.D.12 — Open OSS PR + release
- [ ] CI; merge; tag `v1.20.0` (minor bump — new feature); goreleaser; deploy

---

### Deliverable 1.E — Per-Cycle Delta View

**Branch:** `phase1/per-cycle-delta-render`

#### Task 1.E.1 — Failing test: new-this-cycle section appears
- [ ] In `internal/report/routing_test.go`: feed unfixable with 5 entries where 2 have `firstSeenCycle == currentCycle`
- [ ] Assert payload contains `🆕 New this cycle (2):` section with those subjects
- [ ] Run, confirm failure

#### Task 1.E.2 — Thread firstSeenCycle through DeltaDiag
- [ ] Add `FirstSeenCycle int` to `DeltaDiag`
- [ ] In `internal/watcher/watcher.go`: populate from seenEntry
- [ ] Run test, partial pass

#### Task 1.E.3 — Implement new-this-cycle section in routing.go
- [ ] Modify `FormatCriticalPayload` / `SplitCriticalPayloads`: separate findings into "new this cycle" and "stable" buckets; render NEW first
- [ ] Run test, pass

#### Task 1.E.4 — Failing test: stable findings collapse to summary
- [ ] 50 unfixable entries, 48 stable, 2 new
- [ ] Assert "stable" section is a single line: "...and 48 other stable findings (last posted 12 cycles ago)" summary
- [ ] Run, confirm failure

#### Task 1.E.5 — Implement stable-collapse
- [ ] Group stable findings; emit summary
- [ ] Run test, pass

#### Task 1.E.6 — Failing test: cycle with 0 changes posts "no change" digest (configurable)
- [ ] Empty new + empty resolved + N stable → returns "✨ No new issues since last cycle (steady state at N findings)"
- [ ] Run, confirm failure

#### Task 1.E.7 — Implement no-change digest
- [ ] Add config flag to opt out of the no-change digest (default: enabled)
- [ ] Run test, pass

#### Task 1.E.8 — Local build + deploy + verify
- [ ] Build agentic-sre dev tag
- [ ] Deploy; wait 2 cycles
- [ ] Verify: cycle 1 message has clear "🆕 New" section; cycle 2 (no changes) shows compact "no change" digest
- [ ] User feedback: confirm signal:noise ratio improved

#### Task 1.E.9 — Open OSS PR + release
- [ ] CI; merge; tag `v1.21.0`; goreleaser; deploy

---

## Verification — Phase 1 Complete

After all 5 deliverables merged + deployed:

- [ ] Slack #ceph-critical shows `🤖 AI Tier Activity` digest with auto-applied PRs (1.A)
- [ ] All Slack remediations contain real values (no `<name>` placeholders) (1.B)
- [ ] GitHub API rate-limit dashboard shows steady-state well under 5000/hour (1.C)
- [ ] OpenProject work-packages auto-created for new unfixable findings (1.D)
- [ ] Cycles with no changes post a compact "no change" digest; new findings stand out (1.E)
- [ ] Operator-pillar scoring re-run: Pillars 2/3/4/5 each +1 grade from baseline

## Roll-Back Plan

Each deliverable is independently revertable:
- 1.A: remove `--ai-slack-url-env` from CR → no posts; revert ai_slack_*.go files in next release
- 1.B: revert per-analyzer commit; older `<name>` text returns
- 1.C: Set `RateLimiter` to no-op limiter; or revert
- 1.D: clear `spec.ticketing.openProject` from CR; OpenProject delivery disabled
- 1.E: revert routing.go changes; previous render returns

## Out of Scope (Phase 2+ work)

- LLM-driven proposer for arbitrary findings
- Cross-cluster RAG federation
- Cosign-signed PRs
- HA aiwatch
- Grafana dashboards
- 15 new analyzers
- Auto-merge of Srenix PRs
- "Approve + remember class" workflow
- "Ignore via click" workflow

These move to Phase 2 + 3 + 4 plans (separate files).
