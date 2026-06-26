# Changelog

All notable changes to this project will be documented in this file. The
format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

The Helm chart `agentic-sre` follows the same version line as the
`srenix` binary (`appVersion == version`). Released chart artifacts are tagged
`agentic-sre-X.Y.Z`; the binary itself is tagged `vX.Y.Z`. The
published Helm repository at
`https://srenix-ai.github.io/agentic-sre/` always
serves the latest tagged chart cut.

## [Unreleased]

## [0.2.0-alpha.6] — 2026-06-23

### Fixed
- **NetworkPolicy approval links now honor `--approval-ttl`.** `ManifestBridge` and `BuildApplyManifestProposalWithTTL` (new) accept a configurable TTL; the old `BuildApplyManifestProposal` is preserved for backward compatibility (delegates to TTL=0 → `DefaultProposalTTL`). Previously every NetworkPolicy approve/deny link expired in 15 minutes regardless of the operator-configured TTL, because `BuildApplyManifestProposal` hardcoded `DefaultProposalTTL`. The Srenix Enterprise aiwatch now passes `--approval-ttl` (default 4h) here so on-call SREs have a full window to respond. `ManifestBridge` gains a `ProposalTTL` field for OSS callers that register it directly.
- **Probes: Spilo / Zalando PGO probe now scans all namespaces.** Previously hardcoded to the `pg` namespace; rewritten to cluster-wide list + per-cluster grouping by `cluster-name` label so multi-cluster and non-default-namespace Spilo deployments are detected.
- **Probes: CrashLoopBackOff no longer mismatches container and restart count.** `podMaxRestartCount` previously took the highest restart count across all containers but the reason from the last container in a waiting state — completely unrelated containers. Now only waiting containers (reason != "") contribute and the winner is the waiting container with the most restarts.
- **Probes: PVC `Lost` phase detected as critical.** Only `Pending` was previously checked. Lost PVCs (data-loss scenario) now emit a CRITICAL finding with PVC names and remediation.
- **Probes: ETCD restarts on Ready members downgraded to WARNING.** Any restart count > 0 previously triggered CRITICAL, causing false pages after routine node maintenance where the pod came back healthy. A Ready member with restarts is now WARNING; CRITICAL is reserved for non-Ready + restarted (crash-loop) or not-Ready (quorum risk).
- **Probes: Velero staleness check uses `completionTimestamp`.** Was using `creationTimestamp`, which could fire a "stale" critical for a long-running backup that had completed hours earlier. Now uses `status.completionTimestamp` (falls back to `startedAt` for in-progress).
- **Probes: ArgoCD `Suspended` health is WARNING, not CRITICAL.** Suspension is an intentional operator action (`argocd app suspend`). Was incorrectly emitting CRITICAL alongside Degraded/Missing states.
- **Analyzers: ImagePullAuth subject is now `Pod/ns/name`.** Was `image-pull-auth/ns/pod/container`, breaking the OSS investigator's `Kind/ns/name` parse and the Srenix Enterprise `runEnvTools` lookup.
- **Analyzers: CertExpiry subject is now `Certificate/ns/name`.** Was `cert-expiry/ns/name`.
- **Analyzers: WorkloadStateDrift subjects no longer carry `(followers)`/`(primary)` suffixes.** These suffixes broke Srenix Enterprise's `runEnvTools` subject parse.
- **Analyzers: ImagePullAuth `notFoundSignals` guard narrows false-positive suppression.** Guard now restricted to `manifest unknown` and `name unknown` — unambiguous "image not found" signals. Removed `repository does not exist` (ambiguous on Docker Hub: also returned for private repos with wrong credentials).
- **Analyzers: ImagePullAuth message truncation increased to 300 chars** (was 160) to retain enough registry error detail for diagnosis.
- **Analyzers: ConfigDrift rollout remediation uses correct kubectl syntax.** Was emitting `kubectl rollout restart Workload/ns/name`; now emits `kubectl rollout restart deployment/name -n ns`.
- **Report: `renderAIBlocks` now renders the `Investigation` field.** Was silently dropped — only `Enrichment` and `ApprovalURL` were rendered. Investigation (🔬) now renders before Enrichment (🤖).
- **Report: `hasActionableFindings` returns true for critical severity** even without an `ApprovalURL`. Criticals were previously routed to "Advisory — Review (no action required)" when no approve/deny link was minted, giving a misleading low-urgency label to real incidents.
- **Report: First-seen criticals no longer render green.** The Slack attachment color was computed only from stable (`critRendered`/`diagRendered`) findings, ignoring new-this-cycle criticals. A first-time critical fired green. Fixed by tracking `hasNewCritical` across the render loop.
- **Report: `renderSilenceSnippet` no longer shown for brand-new findings.** A finding that just appeared this cycle should be investigated, not immediately silenced. The silence affordance (kubectl one-liner or signed click-link) is now gated on `!IsNewThisCycle`.

## [0.2.0-alpha.5] — 2026-06-19

### Changed
- **Alertmanager push is now limited to critical + actionable findings.** Purely-advisory warnings (no approve/deny action) are no longer pushed to Alertmanager — they stay on Srenix's native Slack with the honest "Srenix Advisory" title. This stops the duplicate, misleadingly-titled "Human Action Required" alerts that the cluster Alertmanager's Slack receiver rendered for advisory findings. Criticals (always) and findings with a signed approve/deny URL still reach Alertmanager for paging/escalation.

## [0.2.0-alpha.4] — 2026-06-19

### Fixed
- **Terminating pods are no longer flagged as stuck/not-ready.** Pod probes (CrashLoopBackOff, PendingPods, FailedMounts, Services, ETCD) now skip pods with a `metadata.deletionTimestamp` (i.e. already Terminating) — they're intentionally being deleted, so flagging them (and proposing DeletePod, which then "already resolved" on click) was pure noise. Most visible during rollouts, including Srenix's own pods. Non-terminating pods in the same state are still detected.

## [0.2.0-alpha.3] — 2026-06-19

### Added
- **Configurable RBAC allowlist** — extend the built-in expected-system allowlist at runtime (no rebuild) via an optional `srenix-rbac-allowlist` ConfigMap in the install namespace. Data keys `allowNamespaces`, `allowRolePrefixes`, `allowRoleNames` (comma/newline-separated) are merged with the built-ins per cycle; absent ConfigMap → built-ins only.
- **Suppressed-RBAC digest** — instead of silently dropping allowlisted findings, `RBACDrift` emits one `info` summary per cycle (`RBACDrift/cluster/suppressed-expected-system`) listing how many wildcard-verb / unbound-SA findings were suppressed and which, so the silenced set stays visible (one info line + a DriftReport, replacing N paging warnings).

## [0.2.0-alpha.2] — 2026-06-19

### Fixed
- **RBAC-drift noise reduction.** The `RBACDrift` analyzer no longer pages "wildcard verb" / "unbound ServiceAccount" warnings for well-known third-party operator and system components that legitimately hold wildcard verbs. The allowlist (previously only `kube-system`/`system:`/`cluster-admin`) now also skips operator namespaces (calico-system, tigera-operator, minio-operator, kasten-io, olm, rook-ceph, cert-manager, longhorn-system, cnpg-system, external-secrets, vault, local-path-storage, cattle-system, openshift-operators) and role-name prefixes (k10-, kasten-, calico-, tigera-, minio-, olm., k3s-, local-path-, console-, rook-, cert-manager, velero, longhorn-, cnpg-, external-secrets, vault-, openshift-), plus the canonical `admin`/`edit`/`view`/`cluster-owner`/`local-clusterowner` roles by exact name. Genuine user-defined wildcard roles (and roles like `custom-admin`/`payments-admin` that merely end in `-admin`) are still flagged.

## [0.2.0-alpha.1] — 2026-06-19

### Fixed
- Slack alert headers are now **conditional on actionability**: the alarming `*Srenix Alert — Human Action Required*` title is used only when a finding in the payload carries a signed Approve/Deny button. Purely advisory findings (e.g. RBAC-drift warnings: wildcard verbs, unbound ServiceAccounts) now render `*Srenix Advisory — Review (no action required)*` so on-call engineers aren't paged to "act" on items that have no action. One-click Silence links are unchanged and still attached to every posted finding.

## [0.1.0-alpha.1] — 2026-06-18

**Version re-baseline.** This project is pre-launch; releases through v1.26.3
were internal pre-alpha iterations mis-numbered as 1.x. Versioning is reset to
SemVer 0.x with `-alpha.N` pre-releases. No code regression — 0.1.0-alpha.1 is
the v1.26.3 tree under honest pre-alpha numbering.

This release also moves per-checkin verification from GitHub CI to a local
`make verify` flow (see `RELEASING.md`); the GitHub `ci.yml` / `bundle-smoke` /
`helm-publish` workflows are now manual-only or release-tag-triggered.

Content carried from the v1.26.3 tree (what the prior 1.x line shipped):

- **Live watcher one-click silence links (OSS foundation, ex-1.26.3).** The
  per-cycle watcher Slack post renders two signed one-click silence links
  (24h subject-scoped snooze + 90d class-scoped mute) when a signer + approval
  base URL are configured; falls back to the 24h kubectl one-liner when
  unconfigured. Signed `SilenceTokenClaims` (EdDSA compact-JWS) protect the
  window/scope/matcher from URL tampering; durations configurable via
  `approval.silence.{shortDuration,longDuration}`.
- **Execution-gate fix (ex-1.26.2).** New `ValidateForExecution()` enforces
  every safety/structural invariant except the creation-time rollback-description
  requirement, so human-approved token-based executions no longer fail with
  "ai proposal lacks rollback info". `Validate()` (creation/sign-time) unchanged.
- **Standby `/healthz` fix (ex-1.26.1).** `srenix watch` binds the health listener
  before leader election (process lifetime, not lease lifetime); `/readyz` is an
  unconditional 200 alias. Fixes the `maxUnavailable=0` rolling-upgrade deadlock.
- The full operator port, drift-class + M2 probes, supply-chain provenance,
  ticketing, dashboard/playground surfaces, and the CHANGELOG↔tag CI gate that
  accumulated across the 1.x line (see the headings below for detail).

## [1.26.3] — 2026-06-17

### Added — one-click Silence links on the LIVE watcher "needs human" Slack path (OSS foundation)

Non-actionable "human intervention" findings previously had no in-Slack way to dismiss them. The **live per-cycle watcher Slack post** (the `srenix watch` delta path — `internal/watcher` → `report.RouteAndPost` → `SplitCriticalPayloads` / `FormatSlackDelta`, which is what the operator/watcher actually posts) now renders **two signed one-click silence links** under each posted finding when a signer + approval base URL are configured: **🔕 Silence 24h** (subject-scoped snooze, `matcher.subject` = the finding's real `Subject`) and **🔕 Silence class (90d)** (class-scoped mute, `matcher.source` = the finding's real `Source`). Clicking either hits the Srenix Enterprise approval-server's `/silence` endpoint, which consumes the signed token and creates a real OSS `Silence` CR (`api/v1alpha1`, `pkg/silence`) with the right `Matcher` + `Until`. Durations are configurable (default 24h / 90d) and the link label tracks the actual window. This PR ships the OSS foundation; the approval-server `/silence` handler is a separate Srenix Enterprise task.

When NO signer/base-URL is configured (OSS-only / air-gapped), the renderer keeps the existing **24h kubectl one-liner** fallback so the air-gapped affordance is never lost.

- **Watcher delta-path wiring** (`internal/watcher/watcher.go`): `attachApprovalURLs` now also mints the two links for every posted finding via `pkgai.MintSilenceLinks` when `Config.SilenceLinks` is fully configured. `DeltaDiag` gained `Source`, `SilenceSubjectURL`, `SilenceClassLongURL`, `SilenceShortDur`, `SilenceLongDur`; the finding `Source` is threaded from `diagnose.Diagnostic.Source` through `seenEntry` into the rendered `DeltaDiag` (probe findings use their component as Source). The shared `renderSilenceSnippet` (used by both `SplitCriticalPayloads` and `FormatSlackDelta`) emits the click-links when present and the kubectl heredoc otherwise; the legacy hardcoded "Silence class (7d)" link is replaced by the single configurable-duration class link (no duplicate class links).
- **Durations flow to `watch`**: `srenix watch` gains `--silence-short-duration` (24h) / `--silence-long-duration` (90d), reusing the already-loaded approval signing key. The operator wires `spec.approval.silence.{shortDuration,longDuration}` → the watcher Deployment's `--silence-{short,long}-duration` flags (rendered only when explicitly set; unset keeps binary defaults).
- **Signed silence token** (`pkg/ai/silence_token.go`): `SilenceTokenClaims{Scope, Source, Subject, MessagePattern, UntilUnix, …}` + `SignSilenceToken` / `VerifySilenceToken`, mirroring the existing approval `TokenClaims` EdDSA compact-JWS (same `kid`, same verify-before-unmarshal discipline). The silence WINDOW (`UntilUnix`), `Scope`, and the whole matcher are SIGNED — an attacker cannot widen a 24h subject snooze into a 90d cluster-wide class mute by editing the URL (it flips the signature). Security model is doc-commented.
- **Link minter** (`pkg/ai/silence_link.go`): `MintSilenceLinks(priv, kid, baseURL, req, now)` returns the two signed `<baseURL>/silence?token=…` URLs (subject-scoped `now+ShortDur`, class-scoped `now+LongDur`), each with a unique JTI; token `exp` extends a clickability buffer past the silence window (mirrors approve-token exp policy).
- **Legacy diagnose path** (`internal/report/slack.go`): the earlier `srenix diagnose --slack-webhook` wiring (`FormatSlackWithSilence`, `FormatSlack` kept as a thin no-link wrapper, `--approval-server-url` / `--signing-key-path` / `--silence-{short,long}-duration` flags on `srenix diagnose`) is retained as harmless secondary coverage; the watcher delta path above is the primary surface.
- **Config knobs**: `approval.silence.{shortDuration,longDuration}` Helm values + `spec.approval.silence.{shortDuration,longDuration}` CR fields (defaults 24h / 90d).
- **RBAC**: the approval-server SA gets `create,get,list` on `silences.srenix.ai` via a namespace-local Role in BOTH the chart (`approval-server-rbac.yaml`) and the operator (`BuildApprovalSilenceWriterRole`, reconciled + finalizer-owned). The operator/CSV/chart operator-ClusterRole also hold the silences verbs so RBAC escalation prevention passes when materializing that Role (chart↔operator↔bundle parity preserved).
- **Tests**: silence token sign/verify round-trip + tamper (UntilUnix/Scope/Subject) + expiry + malformed; minter two-well-formed-URLs with correct scope/until/jti + messagePattern propagation; delta renderer (`SplitCriticalPayloads` + `FormatSlackDelta`) shows BOTH click-links with correct scope + configurable-duration labels when minted, falls back to the kubectl one-liner when unconfigured, and emits exactly one class link; `attachApprovalURLs` mints both links with the matcher built from real `Subject`/`Source` (not Component), verified by signature; operator watcher gets `--silence-{short,long}-duration` only when `spec.approval.silence.*` is set; legacy `FormatSlackWithSilence` configured/no-link regression; operator silence-writer Role/RoleBinding unit tests.

## [1.26.2] — 2026-06-17

### Fixed — human-approved (token-based) executions no longer fail with "ai proposal lacks rollback info" (OF1)

The Srenix Enterprise approval-server executor reconstructs an `AIProposedAction` from the signed approval JWT and validated it with `(*AIProposedAction).Validate()` before applying the mutation. But the signed token deliberately carries only the safety-relevant identity (`action_id` / `tier` / `action_kind` / `target` / `diag_subject`) and intentionally OMITS the rollback description — while `Validate()` requires `Rollback.Description != ""` (`ErrMissingRollback`). So every human-approved, token-based execution failed at the execution gate even though the proposal was a fully-valid, approved action. Rollback is a proposal-CREATION quality gate (the LLM must supply a rollback plan, rendered to the approver in Slack/the ticket) — it is not an execution-time invariant and cannot be re-checked against a token that omits it by design.

- **New `(*AIProposedAction).ValidateForExecution()`** (`pkg/ai/validate.go`) enforces every safety/structural invariant `Validate()` does — action_kind closed enum, target presence/shape, protected-namespace boundary, patch-payload/kind pairing, manifest validity for `ApplyManifest`, pull-request URL shape, expiry window, proposal-tier check — EXCEPT the rollback-description requirement. This is the correct check for executing an already-approved, reconstructed action; the Srenix Enterprise approval-server executor is the intended caller.
- **`Validate()` behavior is UNCHANGED** (creation/sign-time contract stays strict, including `ErrMissingRollback`). The shared checks are factored into an unexported `validateStructural()` helper; `Validate()` = shared checks + rollback requirement, `ValidateForExecution()` = shared checks only. No signature or external-behavior change for existing callers/tests.
- **Tests** pin the exact failing case (empty-rollback proposal passes `ValidateForExecution`, fails `Validate`) across every executor-reachable kind (`DeletePod` / `DeleteJob` / `DeleteCertRequest` / `DeleteACMEOrder` / `PatchDeployment` / `ApplyManifest`), assert `ValidateForExecution` still rejects the genuinely-unsafe cases (invalid/empty action_kind, missing target, protected namespace, patch on non-patch kind, invalid manifest, expired window, disallowed tier), and assert Validate↔ValidateForExecution parity when rollback is present.

The T3/runbook validation path (`VaultRunbook.Validate`, which reuses `ErrMissingRollback` for "incomplete runbook") is untouched — scope is strictly `AIProposedAction` execution.

## [1.26.1] — 2026-06-12

### Fixed — watcher standby pods now serve `/healthz`; rolling upgrades no longer deadlock (O11, production 1.26.0 upgrade incident)

PR #186 (1.26.0) introduced the always-on `:8081` health server **and** the chart/operator liveness+readiness probes that target it — but started the listener inside `Watcher.Run`, which `srenix watch` wraps in `RunWithLeader`, i.e. inside the leader-election `OnStartedLeading` callback. A standby (non-leader) pod therefore served nothing on `:8081`: the liveness probe got `connection refused` and kubelet kill-looped it. Under the operator-built Deployment's `RollingUpdate maxUnavailable=0` strategy this deadlocked **every** upgrade — the new pod could never pass its probes while the old leader held the `srenix-watcher` lease, and the old pod was never terminated. Production recovery required deleting the old leader pod and temporarily relaxing `maxUnavailable`.

- **`srenix watch` now binds the health listener BEFORE entering leader election**, on the command context (process lifetime, not lease lifetime), via the new idempotent `Watcher.StartHealthServer`. `/healthz` returns 200 as soon as the process is up — leader, standby, or still-acquiring. A bind failure is a hard startup error (loud exit beats a silent probe kill-loop). `Watcher.Run` still calls `StartHealthServer` defensively (idempotent no-op when already started), so direct `Run` callers and the `SRENIX_LEADER_ELECTION=off` path keep the listener.
- **`/readyz` added as an unconditional 200 alias of `/healthz`** — deliberately NOT gated on holding the leader lease. The chart and operator point both probes at `/healthz`; the watcher serves no Service traffic that readiness needs to gate, and a leadership-gated readiness would re-create the same deadlock (a standby pod that never goes Ready blocks `maxUnavailable=0` rollouts and single-replica rollover). Documented in the chart template, the operator's `watcherHealthProbes`, and the handler.
- **Regression test** `TestStartHealthServer_Serves200WhileStandby_NotLeader` (internal/watcher) pins the incident: health server started the way `cmd/srenix` does, leader election entered against a lease already held by another identity, asserts `/healthz` answers 200 while the watch-loop body has never run. Plus an idempotency test (second `StartHealthServer` must not re-bind).

No probe endpoints, ports, or chart values changed — `helm upgrade` from 1.26.0 picks up the fixed binary and rolls cleanly (this release is itself the first upgrade that no longer needs the manual leader-pod delete).

## [1.26.0] — 2026-06-12

Release cut covering everything merged since v1.25.1 (PRs #186–#203 plus the O9 release PR): the operator CronJob unknown-flag production fix, the housekeeping/honesty batches (O6–O8), the trigger/security hardening from the adversarial review (P1.x/P2.x), supply-chain provenance (P6.2), ticketing M2, the dashboard + playground deploy surfaces, and the new CHANGELOG↔tag CI gate.

### Added — CI: CHANGELOG ↔ git-tag parity gate (`scripts/changelog-tag-check.sh`)

Companion to `changelog-lint.sh` (which checks heading format): every released `## [x.y.z]` heading except the topmost (the release in flight) must have a matching `vx.y.z` tag (or chart-releaser `agentic-sre-x.y.z` tag — the 1.8.x line shipped chart-only cuts under that form), and `[Unreleased]` content may not present a version as already shipped (no dated `### [x.y.z] — YYYY-MM-DD` headings, no version-numbered headings). Both `[Unreleased]` checks are anchored to the markdown HEADING signature, so prose that merely cross-references a version and a date (e.g. `- Backport of [1.25.1] fix from 2026-05-11`) passes instead of false-positiving. Catches the claimed-but-never-tagged release class. Runs in the lint CI job next to changelog-lint (the job's checkout now fetches tags), followed by `scripts/changelog-tag-check_test.sh` — positive/negative fixture selftests for the gate itself, including the prose-mention case.

### Added — cloud-probe message join keys for Srenix Enterprise's cross-resource RCA matchers

Srenix Enterprise's cross-resource RCA matchers (ai/cloudcontext, PR #65) join Kubernetes resources to cloud findings via tokens parsed out of the finding MESSAGE. Three LB probes and the Azure cert probe omitted the join keys; they now carry them. **Message-only enrichment** — subjects, severities, and finding counts are unchanged, and the suffix format is a frozen cross-repo contract: single space, literal `(lb: ` / `(domains: `, comma-separated domains with no spaces, closing paren.

- **`aws-alb-target-health`** — the 0-healthy-targets finding appends ` (lb: <load balancer DNS name>)`. The live wrapper resolves the target group's `LoadBalancerArns` via ONE `elbv2.DescribeLoadBalancers` per probe cycle (not per target group); new optional `ALBTargetGroup.LoadBalancerDNS` (`loadBalancerDNS` snapshot field).
- **`gcp-lb-backends`** — the 0-healthy-backends finding appends ` (lb: <forwarding-rule IP or name>)`, falling back to the backend-service name when unmapped. The live wrapper adds ONE `compute.ForwardingRules.AggregatedList` per probe cycle, joining passthrough-LB rules on `rule.BackendService` (proxy-based rules would need a target-proxy + URL-map walk and are deliberately left to the name fallback); new optional `BackendService.ForwardingRule` (`forwardingRule` snapshot field).
- **`azure-appgw-backends`** — the 0-healthy-members finding appends ` (lb: <AppGW public hostname>)` from the already-fetched HTTP-listener config (`HostName`/`HostNames`; no extra API call), falling back to the gateway name; new optional `AppGatewayBackend.FrontendHostname` (`frontendHostname` snapshot field).
- **`azure-certs`** — both cert findings (`expires` / `is not issued`) append ` (domains: <d1>,<d2>)` from the certificate resource's `HostNames` (SANs/CN, already in the fetched data); new optional `Certificate.Domains` (`domains` snapshot field). Omitted entirely when no domains are known.
- **Backward/offline compat** — all four fields are optional snapshot additions; live-wrapper enrichment fetches are best-effort and never fail the probe.
  - AWS (`aws-alb-target-health`) and Azure certs (`azure-certs`): absent or failed enrichment → the pre-enrichment message with no suffix added (no `(lb: ...)` or `(domains: ...)`) (name is not enough — DNS/IP needs a separate API call, so without it there is nothing safe to emit).
  - GCP (`gcp-lb-backends`) and Azure AppGW (`azure-appgw-backends`): absent or failed enrichment → name-only suffix, e.g. `(lb: my-forwarding-rule)` or `(lb: my-appgw)`, because the backend/gateway name is already present in local data (name is in-process and always available; DNS/IP needs a separate API call). Never a panic or an empty `(lb: )`.
- **Contract pinned** — shared suffix builders in `internal/cloud/joinkeys.go` (`JoinKeyLB`, `JoinKeyDomains`) + `internal/cloud/contract_test.go` freezing the literal `" (lb: %s)"` / `" (domains: %s)"` formats with a pointer at the Srenix Enterprise dependency; per-probe tests assert the exact enriched message with the data present AND the unsuffixed shape when absent; live-layer helper tests cover the ARN→DNS map, the forwarding-rule index, listener-hostname extraction, and SAN flattening. No chart/CRD changes.

### Added — hash-chained audit-trail primitive in OSS `pkg/audit` (closes the features/policy source-citation gap)

The website's features/policy page cites `pkg/audit/hash_chain.go` as the auditable open-source implementation of the tamper-evident audit trail, but the hash-chain primitive previously lived only in the private Srenix Enterprise repo — the cited file did not exist here. The primitive is now ported (first-party code, Apache-2.0), so "audit the envelope before you install" includes the chain itself.

- **`pkg/audit/hash_chain.go`** (new, stdlib-only) — the full chain core operating directly on the OSS `ai.AuditEvent`/`ai.AuditSink` types: `ChainedSink` (canonical-JSON hashing, `prev_hash`/`entry_hash` linking, tamper-evident `entry_time` stamping via `EntryTimeKey`, failed inner write does NOT advance the chain), `NewChainedSinkResuming(inner, resumeHash, ChainOptions{Signer, CheckpointEvery})` (a restart continues the existing chain instead of re-anchoring at `""`; periodic signed checkpoints), `WriteCheckpoint` (caller-triggered tail anchor, e.g. on close), `CheckpointSigner` (narrow signing interface; concrete Ed25519 adapters live with the caller), `VerifyChain` (first broken index or -1), and `VerifyChainWithCheckpoints`/`ChainVerification` (tail-truncation detection: the hash links alone cannot catch lopping off the tail; only a signed checkpoint as the final entry anchors it).
- **OSS vs paid boundary** — the chain primitive + verification are OSS; the richer sinks the chain wraps (JSONL chained file with rotation, Loki, OTLP/SIEM) remain in the paid Srenix Enterprise binary, registered via the registry without removing the defaults. The package doc states the split and the intended Srenix Enterprise adapter path (import swap + `NewChainedSinkResuming` from the store's last persisted `entry_hash` + `WriteCheckpoint` on close; the Srenix Enterprise swap itself is a separate follow-up after the next OSS release).
- **Tests** (`pkg/audit/hash_chain_test.go`, 23 cases) — chain integrity and link consistency; tamper detection at EVERY position for five tamper classes (struct field, Details key, `entry_hash` forgery, `prev_hash` forgery, timestamp edit); reordering detection; resumption continues across sink lifetimes with no `""` re-anchor; checkpoint cadence; signed-checkpoint tail-truncation detection (plain `VerifyChain` accepts the truncated prefix, the checkpoint-aware verifier flags it); unsigned/absent checkpoints report an unanchored tail; a broken chain reports `BrokenIndex` AND the last trustworthy `LastCheckpointIndex` (scanned over the verified prefix only) together; Ed25519 checkpoint signatures verify against the head hash; canonicalization stability (map insertion order, nested maps, unicode — CJK/RTL/emoji/combining marks — and fixed-clock determinism); and a golden-bytes canonical-format contract test (`TestCanonicalJSON_FormatContract`) freezing the exact `encoding/json.Marshal` form — declaration-order struct fields, sorted map keys, HTML-escaping ON — that production chains already use, so any accidental format change (e.g. `SetEscapeHTML(false)`) fails loudly instead of silently breaking cross-version verification.

### Added — append-only protected-namespace extension (`SRENIX_PROTECTED_NAMESPACES_EXTRA`)

The website's policy page promised the protected-namespace list is "configurable; the operator extends this list per cluster", but both compiled-in lists (`internal/fix/protected.go` for the fixer guard, `pkg/ai/validate.go` for the AI-action validator) had no extension mechanism. Now there is one — APPEND-ONLY by construction: operators can ADD protected namespaces; nothing can remove the compiled-in floor (`kube-system`, `kube-public`, `kube-node-lease`, `rook-ceph`, `vault`, `external-secrets`, `cnpg-system`).

- **Binary** — new env var `SRENIX_PROTECTED_NAMESPACES_EXTRA` (comma-separated; entries trimmed, empties dropped, duplicates collapsed) consumed by BOTH act-side guards. `pkg/ai` (new `protected.go`) owns the shared extra set — `EnvProtectedNamespacesExtra`, `ParseProtectedNamespacesExtra`, `SetExtraProtectedNamespaces(...)` (host/test initializer), `LoadExtraProtectedNamespacesFromEnv()`, `IsExtraProtectedNamespace`, `ExtraProtectedNamespaces()` — loaded lazily on first check, so the srenix-enterprise aiwatch (which links `pkg/ai`) inherits the widened floor with zero code change. `internal/fix.IsProtectedNamespace` keeps its compiled floor and ORs in the shared extras, so the fixer guard and the AI validator can never disagree. The detect side honors the same knob: `internal/probe.IsProtectedNamespace` (severity ESCALATION — in probe-land protected namespaces are "always-critical", e.g. the CrashLoop probe) ORs in the shared extras too, so an issue in an extra-protected namespace is escalated, not just shielded from auto-fix. **Backward compatible**: `ai.IsProtectedNamespace` / `ai.ProtectedNamespaces` signatures unchanged; the exported map is now documented as the compiled-in FLOOR (extras live alongside, never inside it).
- **Helm** — new `protectedNamespaces.extra: []` value; a non-empty list renders `SRENIX_PROTECTED_NAMESPACES_EXTRA` on the watcher Deployment, diagnose + remediate CronJobs, AND the aiwatch Deployment (empty = byte-identical render). The Gatekeeper third-layer constraint (`gatekeeper.constraints.protectedNamespaces`) now appends `protectedNamespaces.extra` (deduped) so the admission gate enforces the same widened boundary.
- **Operator** — new top-level CR field `spec.protectedNamespacesExtra []string` (ONE field feeding both consumers — deliberately not under `spec.remediate` or `spec.ai`, because splitting the knob per consumer would invite the two safety floors to diverge; rationale in the field's doc comment). Rendered by the builders onto the watcher Deployment, both CronJobs, AND `aiEnv` (aiwatch). CRD schema added to both the chart template and `bundle/manifests` (the Go↔CRD and bundle↔chart parity gates pin them); `bundle/tests/sample-cr-full.yaml` extended; hand-maintained DeepCopy updated.
- **Safety tests** — floor preserved under empty/garbage/whitespace env values and under attempts to "replace" the list (`pkg/ai/protected_test.go`, `internal/fix/protected_test.go`, `internal/probe/protected_test.go`); extension visible to the fixer guard, the probe severity escalation, the proposal validator (`Validate()` → `ErrProtectedNamespace`), and the safe-apply manifest validator (`ErrManifestProtectedNS`); lazy env load can never clobber a racing `SetExtraProtectedNamespaces` (double-checked locking, pinned by `TestLazyEnvLoad_DoesNotClobberRacingSetter`); operator builder env propagation + absent-when-unset (`internal/operator/protected_namespaces_test.go`); helm-unittest for all four workloads; the P1.8 toggle-drift gate now scans `pkg/ai/protected.go`.
- **Docs** — `docs/SETUP_GUIDE.md` values appendix + TLSSecretMismatch safety constraints describe the floor + extension (and no longer claim a `srenix.ai/protected` namespace-label mechanism that never existed in the code).

### Added — toggles for the 6 base probes + 7 core analyzers (docs said "each probe independently togglable"; now it's true)

The public docs promised every probe/analyzer is independently disablable, but the six base probes (Ceph, Nodes, Postgres, PVCs, Critical Services, Endpoints) and seven core analyzers (SecretKeyMissing, FailingExternalSecrets, ProactiveSecretKeyCheck, UnprovisionedSecret, ImagePullAuth, CertExpiry, TLSSecretMismatch) were registered unconditionally in `catalog/catalog.go` — no env gate, no chart toggle.

- **catalog** — each now follows the exact `os.Getenv("SRENIX_X") != "off"` opt-out pattern the 15 existing gated probes/analyzers use. New env vars: `SRENIX_PROBE_CEPH`, `SRENIX_PROBE_NODES`, `SRENIX_PROBE_POSTGRES`, `SRENIX_PROBE_PVCS`, `SRENIX_PROBE_CRITICAL_WORKLOADS` (gates the Critical Services probe — the documented env name), `SRENIX_PROBE_ENDPOINTS`, `SRENIX_ANALYZER_SECRET_KEY_MISSING`, `SRENIX_ANALYZER_FAILING_EXTERNAL_SECRETS`, `SRENIX_ANALYZER_PROACTIVE_SECRET_KEY_CHECK`, `SRENIX_ANALYZER_UNPROVISIONED_SECRET`, `SRENIX_ANALYZER_IMAGE_PULL_AUTH`, `SRENIX_ANALYZER_CERT_EXPIRY`, `SRENIX_ANALYZER_TLS_SECRET_MISMATCH`. **All 13 default ON — no behavior change for existing installs** (these probes/analyzers have shipped default-on since v1.0; the toggle only adds the documented opt-out, so the P3.3a default-off discipline records a status-quo soak rationale in the golden rather than shipping the secret-chain core signal default-off).
- **chart** — `probes.{ceph,nodes,postgres,pvcs,criticalWorkloads,endpoints}.enabled` and `analyzers.{secretKeyMissing,failingExternalSecrets,proactiveSecretKeyCheck,unprovisionedSecret,imagePullAuth,certExpiry,tlsSecretMismatch}.enabled` (all default `true`), wired through the existing `srenix.probeToggleEnv` / `srenix.analyzerToggleEnv` helpers — `enabled: false` emits `SRENIX_*=off` on the watcher + diagnose containers, byte-identical render at defaults.
- **tests** — `catalog/catalog_test.go` (new): every toggle registers by default, skips on `=off`, doesn't drop siblings, and non-`off` values (`true`/`OFF`/garbage) do NOT disable; helm-unittest coverage extended (`probe_toggle_test.yaml` + new `analyzer_toggle_test.yaml`); the P1.8 toggle-drift and P3.3a default-off chartgates pass with the new inventory.
- **operator** — no change needed: the CR has no probe/analyzer toggle surface today (the existing 15 toggles aren't operator-settable either); tracked as a follow-up alongside a watcher `extraEnv` passthrough rather than growing the CRD here.
- **docs** — `docs/SETUP_GUIDE.md` env-var tables list the 13 new toggles with their Helm values.

### Added — chart: deploy the read-only hosted dashboard (P6.6)

P6.6 shipped a `srenix-enterprise dashboard` subcommand (a read-only, server-rendered HTML view of findings/approvals/history). This wires the OSS Helm chart to actually deploy it, mirroring the approval-server's chart pattern.

- **values.yaml** — a new `dashboard:` block, default `enabled: false` (byte-identical render for existing installs). Carries `image` (defaults to the `docker4zerocool/srenix-enterprise` image like `approval`/`ai`), `replicas`, `approvalBaseURL` (REQUIRED when enabled — the approval-server URL the Approve/Deny/Ignore links target), `authHeader` (default `X-Forwarded-User`), optional `auditLogPath`, `historyLimit`/`approvalsLimit`, `ingress.{host,ingressClassName,annotations,tls}`, and `networkPolicy.{enabled,gatewayNamespaceSelector}` (default-off, required-selector-when-enabled — the same P2.6b fail-closed contract as `approval.networkPolicy`).
- **Templates** (all gated on `dashboard.enabled`) — `dashboard-deployment.yaml` (runs `srenix-enterprise dashboard` with `--approval-base-url` + `--auth-header`), `dashboard-service.yaml` (ClusterIP :8444), `dashboard-serviceaccount.yaml` (a DEDICATED `<release>-dashboard` SA), `dashboard-rbac.yaml` (a **read-only** ClusterRole — `get/list/watch` on `driftreports` only, plus `resolutionrecords` read when that CRD is enabled — bound to the dashboard SA; **no signing key, no mutate verbs, no Secret access**), `dashboard-ingress.yaml`, and `dashboard-networkpolicy.yaml` (mirrors the approval-server P2.6b NetworkPolicy: restricts ingress to the gateway namespace).
- **Guard test** — `chart_dashboard_binding_sa_test.go` asserts the dashboard ClusterRoleBinding targets the dashboard's OWN SA (`<fullname>-dashboard`, not the watcher SA — the silence-binding-bug class) AND that the ClusterRole carries ONLY `get/list/watch` with no mutate verb and no `secrets` reference.

Operator path: this follow-up is **chart-only by design**. Wiring the operator (a `DashboardSpec` CRD field + DeepCopy + reconcile/teardown + a cluster-scoped ClusterRole/Binding requiring finalizer cleanup) would touch the CRD and trip the CRD/RBAC/bundle parity gates; per the P6.6 deploy note the operator path is tracked as a separate follow-up so the parity gates stay green and the CRD surface is unchanged.

### Added — hosted playground bundle: live synthetic-drift demo (P6.8) — **shipped** (deploy + DNS are the operator's final step)

The website's "Try it now" / playground CTA now has a real, kind-verified, deployable implementation under `examples/playground/`. A visitor watches Srenix detect synthetic K8s drift **live** — real watcher, real `DriftReport` CRs, real broken workloads — inside a fully isolated namespace that cannot disturb anything else.

- **Drift injector** (`drift-injector.yaml`): a CronJob (stock `alpine/k8s` image + inline bash, no custom build) that creates and rotates **four** synthetic scenarios every 15 min, each mapped 1:1 to a **shipped OSS analyzer/probe** (verified firing on kind): (1) Deployment with a bad imagePullSecret pulling a private Docker Hub repo → `ImagePullAuth` (`internal/diagnose/image_pull_auth.go`); (2) Job referencing a missing Secret key → `SecretKeyMissing` (`internal/diagnose/secret_key_missing.go`); (3) Ingress serving an **expired** TLS secret while a Ready cert-manager `Certificate` renews the same host into a **different** secret → `TLSSecretMismatch` (`internal/diagnose/tls_secret_mismatch.go`); (4) CrashLoopBackOff Deployment → `CrashLoopBackOff` probe (`internal/probe/crashloop.go`).
- **Isolation** (`namespace.yaml`): dedicated `srenix-playground` namespace (PSA `baseline`) + **ResourceQuota** (30 pods / 1 CPU / 1Gi) + **LimitRange** + **default-deny NetworkPolicy** (only DNS / in-namespace / HTTPS-out re-opened) + **namespaced injector RBAC** (acts only in-namespace) + a viewer ClusterRole granting **only** `get/list/watch` on `driftreports`. Every workload carries `nodeAffinity` excluding GPU nodes (`nvidia.com/gpu.present DoesNotExist`).
- **Viewer** (`viewer/`, ~180 lines Go): a self-contained **OSS** read-only page (not the Srenix Enterprise dashboard, so anyone can `kind`-run it) that lists the cluster-scoped `DriftReport` CRs and renders them with `html/template` (**XSS-safe** — tested). Deployment + Service + Ingress (`playground.asre.baisoln.com`, Kong ingressClass + `cert-manager.io/cluster-issuer: letsencrypt-prod`, mirroring the cluster's website/grafana ingress).
- **Srenix watcher** (`srenix-values.yaml`): the OSS chart installed scoped to `srenix-playground`, **diagnose-only** (`watcher.remedy.enabled=false`), AI/approval/ticketing/cloud all off, single pod, GPU-excluded. The playground only DETECTS drift; it never mutates.
- **Runbook** (`README.md`): kind quick-try (anyone can run locally), prod deploy, the **DNS step documented but NOT executed** (Cloudflare A record + `deploy/lib/dns.sh` `DNS_DOMAINS` entry per the dns-new-subdomains rule), and teardown. Honest scope note: `DriftReport` is cluster-scoped and `srenix watch` lists cluster-wide, so the reader ClusterRole is cluster-wide read-only (safe with remedy off; only `srenix-playground` ever contains injected drift).
- **kind verification:** created a kind cluster, installed the chart + bundle, injected all four scenarios, and confirmed the watcher produced DriftReports for each (`ImagePullAuth`, `SecretKeyMissing`, `TLSSecretMismatch`, `CrashLoopBackOff` probe) and the viewer rendered them (29 active reports, `/healthz` 200). `helm lint` + `kubeconform` + `kubectl apply --dry-run=server` clean; viewer `go build`/`vet`/`test`/`gofmt` clean incl. an XSS test. Hosted deploy + DNS remain the operator's final manual step.

### Added — chart + operator: ticketing.{jira,servicenow,route} values → Srenix Enterprise ticketing env (makes the paid Jira/ServiceNow sinks deployable)

The Srenix Enterprise Jira/ServiceNow ticketing sinks just shipped but were undeployable end-to-end: nothing populated the `SRENIX_JIRA_*` / `SRENIX_SERVICENOW_*` / `SRENIX_TICKETING_ROUTE` env vars the aiwatch (srenix-enterprise) container reads. This wires them through both render paths the OSS chart/operator own.

- **values.yaml** — the `ticketing:` block gains `route` (→ `SRENIX_TICKETING_ROUTE`), `jira.{url,project,email,issueType,priority.{critical,warning,info},webUrlBase}` + `jira.tokenSecret.{name,key}` (→ `SRENIX_JIRA_TOKEN`), and `servicenow.{url,user,urgency.*,impact.*,webUrlBase}` + `servicenow.passwordSecret.{name,key}` / `servicenow.bearerSecret.{name,key}` (→ `SRENIX_SERVICENOW_PASSWORD` / `SRENIX_SERVICENOW_BEARER`). All default empty/unset — byte-identical render for existing installs.
- **Render (chart + operator).** New `srenix.ticketingProviderEnv` helper (mirroring `srenix.aiEnv`) and `internal/operator` `ticketingProviderEnv` add the env to the aiwatch container in lockstep. Plain (non-secret) values render as `value:`; **credentials (Jira token, ServiceNow password/bearer) render as `valueFrom.secretKeyRef` ONLY — the literal never appears in the manifest**, mirroring the existing `ai.apiKey.secretName` pattern. Each env var is emitted only when its source is set: no empty `SRENIX_JIRA_TOKEN` when no secret-ref is configured.
- **CRD parity.** `TicketingSpec` gains `Route`, `Jira` (`TicketingJiraSpec`), and `ServiceNow` (`TicketingServiceNowSpec`) with secret-refs via a new `TicketingSecretRef`; hand-ported into the chart CRD, the bundle CRD, and `bundle/tests/sample-cr-full.yaml` (full-surface coverage). DeepCopy hand-written per repo convention. All parity gates (CRD↔Go-types, bundle↔chart, full-surface sample, RBAC, toggle/flag) stay green. No RBAC change — the kubelet resolves the secret-ref at pod admission; the operator only emits the reference.

### Added — CycloneDX SBOM + cosign keyless image signing + attestation (P6.2)

The release pipeline now emits verifiable supply-chain provenance, making the website's "SBOM (paid)" and "Cosign-signed container images with attestation" claims real and customer-verifiable. `.goreleaser.yaml` gains three pipes: `sboms:` (syft → one **CycloneDX JSON** SBOM per binary archive, attached to each GitHub Release), `signs:` (cosign **keyless** `sign-blob` over `checksums.txt` → `checksums.txt.sigstore.json`, transitively covering every archive + SBOM), and `docker_signs:` (cosign **keyless** `sign` over every container image + multi-arch manifest on both Docker Hub and GHCR, recorded in the Rekor transparency log). Keyless = no private key on disk: the release workflow's GitHub OIDC token (`permissions: id-token: write`) is exchanged for a short-lived Fulcio certificate. The release workflow installs syft (`anchore/sbom-action/download-syft`) + cosign (`sigstore/cosign-installer`) and runs goreleaser with `--timeout 2h` (the 1h internal default is too tight once SBOM + signing pipes are added on top of multi-arch buildx). New `docs/RELEASE_VERIFICATION.md` documents the exact `cosign verify` / `cosign verify-blob` commands (certificate-identity-regexp + GitHub OIDC issuer) and how to download + inspect a CycloneDX SBOM. Verified locally: `goreleaser check` passes and `goreleaser release --snapshot` produces six valid CycloneDX 1.6 SBOMs (104 components each); the signing pipes only execute in CI under a real OIDC token.

### Added — ticketing: resolve-on-clear + debounced comment-on-recurrence (M2, P6.5) — tickets now auto-close

`Sink.Resolve` and `Sink.Comment` shipped in M1 with **zero call sites** — tickets never auto-closed and never got a recurrence comment, which trains operators to ignore the ticket sink. M2 wires both.

- **Resolve-on-clear (default ON).** When a previously-ticketed finding is no longer present in the diagnose cycle, Srenix closes the ticket with reason `Srenix: condition cleared as of <ts>`. The cleared-subject set is computed in the watcher (seen last cycle, absent now) **independently of the Slack `postOnResolved` setting**, and `report.RouteResolves` runs **before** `Reconcile` deletes the DriftReport CR — the only point at which the persisted `TicketRef` on `status.ticket` is still readable. Idempotent: a resolved ticket is stamped `status.ticket.resolved=true` + `resolvedAt`, and Srenix never re-resolves it.
- **Comment-on-recurrence (debounced).** A finding that already has a ticket (still-open, or a resolved ticket whose finding reappeared) now comments on the **existing** ticket instead of the M1 no-op, debounced by `ticketing.commentInterval` (default `1h`) keyed on `status.ticket.lastCommentedAt` so a flapping finding can't spam the tracker. A recurrence after a clear also clears the `resolved` flag so the next clear resolves the ticket again. **After-interval recurrence reuses the existing ticket (no new ticket is opened)** — one ticket per finding keeps the operator's investigation history together.
- **Severity-transition comment.** A still-open ticketed finding that changes severity gets a transition comment (debounced like recurrence). Severity is stamped on `status.ticket.severity` at open / last comment so the next transition is detectable.
- **CRD/status:** new `status.ticket` fields `severity`, `resolved`, `resolvedAt`, `lastCommentedAt` (chart DriftReport CRD + bundle CRD).
- **Config:** new `ticketing.resolveOnClear` (default `true`) and `ticketing.commentInterval` (default `1h`) chart values; binary flags `--ticketing-resolve-on-clear` / `--ticketing-comment-interval` (+ `TICKETING_RESOLVE_ON_CLEAR` / `TICKETING_COMMENT_INTERVAL` env); operator `spec.ticketing.{resolveOnClear,commentInterval}` (chart + bundle CRD, full-surface sample CR, builder emits the flags — `resolveOnClear` nil defaults to `=true`). No-op when no ticketing provider is configured.

### Added — explicit OWASP K8s Top-10 mapping + posture-non-regression guard (G2)

The fixer safety envelope (protected namespaces, GitOps-aware skip, dry-run, minimal RBAC) was real but the "we don't violate OWASP" property was unlabeled and untested. It is now explicit and locked. New `docs/OWASP_MAPPING.md` maps every fixer (`internal/fix/*.go`) and every security-relevant analyzer signal (`SecurityDrift`, `RBACDrift`, `TLSSecretMismatch`, `NetworkPolicyProposer`, the digest-pin signal, …) to the OWASP Kubernetes Top-10 item(s) it **respects** (fixers — proven not to violate) or **detects** (analyzers — observational, detection ≠ enforcement). Each fixer's doc comment now names the OWASP item it respects. The key deliverable is `internal/fix/owasp_posture_test.go`: a table-driven guard that runs every fixer against a fixture, captures each Delete/Patch, and asserts no mutation ever removes/weakens a NetworkPolicy (K07), adds `privileged`/`hostPath`/`hostNetwork`/a capability (K01), broadens RBAC (K03), downgrades a TLS secret reference (K08), or deletes in a protected namespace. A meta-check scans the package for every `Fixer`-implementing type and fails the build if a fixer has no posture-test entry — a new fixer **cannot** silently skip the guard. Wires nothing new into CI beyond the existing `go test ./...`.

### Added — operator + chart: approval-server NetworkPolicy closes the X-Forwarded-User bypass (P2.6b)

The approval-server trusts the `X-Forwarded-User` header for audit attribution. That header is injected by oauth2-proxy at the OIDC ingress after a successful login — but the approval-server's `ClusterIP` Service is reachable by any pod in the cluster, and a pod hitting it directly bypasses the ingress and can forge an arbitrary `X-Forwarded-User`. The approve/deny click still requires a valid one-time signed token, so this was defense-in-depth for attribution honesty, not an auth bypass — but it let any pod corrupt the audit trail's "who approved this" field.

A new **opt-in NetworkPolicy** restricts ingress to the approval-server pods (port 8443/TCP) to **only the gateway/oauth2-proxy namespace**, so the only `X-Forwarded-User` the server ever sees is the one oauth2-proxy set.

- Operator: `BuildApprovalServerNetworkPolicy(cr)` (owner-ref'd, reconciled alongside the approval-server Deployment/Service; torn down when disabled). `podSelector` matches the Deployment's pod labels exactly.
- CRD/types: new `spec.approval.networkPolicy.{enabled, gatewayNamespaceSelector}` (chart CRD + bundle CRD + full-surface sample CR).
- Chart: `templates/approval-server-networkpolicy.yaml` + `approval.networkPolicy.{enabled, gatewayNamespaceSelector}` values.
- RBAC: the operator ClusterRole (chart + bundle CSV) gains `networkpolicies` create/update/patch/delete on `networking.k8s.io`.
- **Default OFF**, strongly recommended in production. A NetworkPolicy is fail-closed: defaulting on with a wrong/absent `gatewayNamespaceSelector` (or on a CNI that doesn't enforce NetworkPolicy) would silently 0-route every approval click — a worse outcome than the bug it closes. `gatewayNamespaceSelector` is **REQUIRED** when enabled (operator fails the CR `Ready=False/InvalidSpec`; chart `fail`s the render) — there is no safe default selector.

### Added — watcher health probes + opt-in multi-replica via leader election (P1.9)

The watcher Deployment shipped with no liveness/readiness probes (every sibling deployment — approval-server, qdrant, operator — had them) because its only HTTP `/healthz` lived inside the `--webhook-listen` branch, so an install without the M6 webhook trigger had no health endpoint to probe. The watcher now starts an **always-on health server** (`--health-listen`, default `:8081`; chart value `watcher.healthListen`) serving `GET /healthz` unconditionally, independent of the webhook receiver, and the chart wires `livenessProbe` + `readinessProbe` against it. The watcher Deployment's hard-coded `replicas: 1` is now `watcher.replicas` (default 1); raising it above 1 is only safe with leader election on, so the chart **fails the render** when `watcher.replicas > 1` and `watcher.leaderElection.enabled=false` (otherwise replicas race on DriftReports and double-post Slack).

### Added — optional timestamped HMAC scheme (replay window)

Webhook senders can now include `X-Srenix-Timestamp: <unix-seconds>` and sign `timestamp + "." + body` (`X-Srenix-Signature: sha256=hex(hmac-sha256(secret, ts+"."+body))`). Timestamped requests more than 5 minutes from server time are rejected with 401, so a captured request can no longer be replayed forever. Requests without the header keep the legacy body-only HMAC check (existing senders unaffected); a once-per-source log notice recommends adopting the timestamp header. New `webhook.SignWithTimestamp` helper for integrators.

### Changed — drive-by polish from the #203 quality review

- `pkg/cloud/gcp.Subnet.SecondaryIPCount` doc comment now states it ships DATA-ONLY (snapshot capture surface; no probe consumes it), and new `internal/cloud/gcp/cidr_test.go` pins the full-size (`rangeSizeFromCIDR`, secondary ranges, no reservations) vs usable (`usableIPsFromCIDR`, primary range, minus GCP's 4 reserved) semantics against each other.
- watcher: the `snapshot.AsMutator` gate is hoisted ABOVE `silence.CountMatches`, so snapshot/dry-run sources skip the per-silence match counting entirely (counts feed only the status writer; without a Mutator they were dead work).
- `catalog/cloud.go`: comment pinning that the literal `os.Getenv(...) != "off"` form is load-bearing for the chartgate regex scanners; `gcpSubnetSmallPrefix()` now logs invalid `SRENIX_CLOUD_PROBE_GCP_SUBNETS_SMALL_PREFIX` values before falling back to the compiled-in /26 default (was: silent).
- test hygiene: `strings.HasPrefix` replaces the panicky `n[:4]` slice in `catalog/cloud_test.go`; `crd_printcolumn_parity_test.go` derives the chart CRD path from the `markerSources` entry instead of hardcoding `crd-silence.yaml` (a second CRD entry can no longer silently compare against the wrong file).

### Changed — housekeeping batch: cloud values wired-or-deleted (O6), GCP subnet capacity contract (O7), Silence UNTIL column + status writer (O8)

Three doc-vs-code honesty fixes shipped as one batch. After this change every key in the chart's `cloud:` block does something, the GCP subnet probe's docs match what it can actually measure, and `kubectl get silences` renders all five printer columns correctly.

**O6 — dead Helm cloud values: wired or deleted.**

- **Wired: `cloud.<provider>.probes.*`** (previously "informational" — the comment admitted only `aws.enabled` did anything; the binary registered all 10 probes per provider unconditionally). Each cloud probe is now independently disablable via `SRENIX_CLOUD_PROBE_<PROVIDER>_<NAME>=off` (default ON — these probes have registered unconditionally since v1.8, so default-on preserves the status quo exactly; soak rationale recorded in the P3.3a golden), following the exact `os.Getenv(...) != "off"` pattern of the K8s `SRENIX_PROBE_*` gates. The chart renders the envs from `cloud.{aws,gcp,azure}.probes.*` via the new `srenix.cloudProbeToggleEnv` helper (GCP and Azure gain `probes:` blocks to match AWS). The `eks` / `gke` / `aks` keys each gate BOTH the control-plane and node-pool probes (one asset, one key). Gates extended: P1.8 toggle-drift and P3.3a default-off now scan `catalog/cloud.go`; new `catalog/cloud_test.go` (default-on / `=off` skips / sibling isolation / non-off values keep registration) + helm-unittest `cloud_probe_toggle_test.yaml`; SETUP_GUIDE env-var table lists all 27 toggles.
- **Deleted: `cloud.rateLimitPerMin`** — consumed by nothing (rate protection is, and was, the `cloud.cadence` interval). Removed from values.yaml; design-doc sketch annotated as not-shipped.
- **Deleted: `cloud.{aws,gcp,azure}.auth.mode` + `cloud.aws.auth.assumeRoleArn` + `cloud.*.auth.credentialsSecret`** — the `assumeRole` / `staticCredentials` modes were never implemented; only `roleArn` / `serviceAccount` / `clientId` are consumed (→ workload-identity SA annotations, which remain). `serviceaccount.yaml` simplified accordingly. Assume-role / static-credential support would be a net-new FEATURE, not a doc fix — recorded here so it can be reintroduced deliberately if demanded.
- **Fixed: stale `catalog/cloud.go` comment** describing the v1.9 monitoring wiring as future work — rewritten in present tense to the actual live-mode coverage (Cloud SQL / Azure SQL storage-% and App Gateway health are Monitoring-API-backed best-effort; GCP subnets are capacity-only, see O7).
- **Operator**: no change — the CR has no per-probe toggle surface for the existing K8s probe toggles either; same follow-up as PR #198.

**O7 — GCP Subnet probe: honest capacity-only contract (the last inert cloud signal).**

- Decision (researched, documented in `internal/cloud/gcp/live.go`): GCP exposes **no cheap used-IP count** for a subnetwork — the allocation-ratio insight lives in Network Analyzer behind the **Recommender API** (`google.networkanalyzer.vpcnetwork.ipAddressInsight`; separate SDK + IAM surface + Network Analyzer dependency), there is **no Cloud Monitoring metric** for it, and deriving "used" from instance NICs would need an Instances aggregated fan-out per cycle. So instead of a signal that silently returns -1, the probe/docs contract changed honestly:
- **Live mode is capacity-only**: `TotalIPCount` from the primary CIDR (minus GCP's 4 reserved), new `Subnet.SecondaryIPCount` summing secondary (alias) ranges, `AvailableIPCount` stays -1 — and the probe now FLAGS unmeasured subnets whose primary CIDR is smaller than /26 (warning; threshold configurable end-to-end: the chart's `cloud.gcp.subnetsSmallPrefixThreshold` renders `SRENIX_CLOUD_PROBE_GCP_SUBNETS_SMALL_PREFIX`, which the catalog feeds into `Subnets.SmallPrefixThreshold` at registration — 0 / unset / invalid = the compiled-in /26 default), pointing at Network Analyzer for the real allocation ratio. Measured mode (snapshot files / future clients with `AvailableIPCount >= 0`) keeps the existing <25%/<10% free-IP thresholds unchanged.
- Probe Detail + README + values.yaml + design-doc claims updated to the capacity-only wording (no more "pending the Monitoring API (v1.9)"); tests cover small-CIDR warn, large-CIDR silent, threshold override, and CIDR-prefix parsing.

**O8 — Silence CRD: UNTIL printer column + status writer (K1 findings).**

- **`kubectl get silences` UNTIL showed `<invalid>`** for every active silence: the printer column was `type: date`, which kubectl renders as age-SINCE — negative for the future expiry an active Silence has by definition. Fixed to `type: string` (raw RFC3339) in all 3 places: the kubebuilder marker (`api/v1alpha1/silence_types.go`), the chart CRD, and the bundle CRD.
- **ACTIVE / MATCHED columns were always empty**: nothing wrote `status.active` / `status.matchCount` despite the CRD comment claiming "status.active flips to false" on expiry. New `pkg/silence/status.go` `UpdateStatuses` (narrow `StatusPatcher` interface, satisfied by the snapshot Mutators): per watcher cycle, `status.active = until > now`, `status.matchCount += this cycle's matches` (running total), `status.lastMatchAt` stamped on match — via merge patch to `/status` (the CRD declares the status subresource; same gotcha as DriftReport reconcile). Writes are cheap: a Silence is patched ONLY when its active flag flipped or it matched this cycle — steady-state cycles patch nothing; failures are soft (collected + logged, never abort the cycle). Wired in `runDiagnose` next to the existing filter using `silence.CountMatches` on the pre-filter diagnostics.
- **New printer-column parity gates** (`internal/operator/crd_printcolumn_parity_test.go`): chart ↔ bundle printer columns pinned per CRD/version; Go `+kubebuilder:printcolumn` markers pinned against the chart CRD; and a `type: date`-on-future-field rule (until/expiry/deadline/notAfter) so the `<invalid>` bug class cannot recur. Status-writer unit tests: active flips on/off, running matchCount + lastMatchAt, no-op patch suppression, soft scoped errors, nil patcher.
- **Operator RBAC**: `BuildReaderClusterRole()` now grants `update`/`patch` on `silences/status` (the chart's `clusterrole-silence.yaml` already did) — without it the status writer 403s on every operator-managed install. The `silences` parent resource stays read-only (SREs own the spec); the stale "only READ by the watcher" builder comment and the chart template's "reserved for a future status-updater" header were rewritten to describe the shipped change-only writer. RBAC builder status-subresource test extended to cover `silences/status`.

### Changed — webhook trigger sources now FAIL CLOSED on missing HMAC secret (P1.1, breaking-ish)

Before this change a `--webhook-source=<name>=<env-var>` whose env var was unset or empty (secret not mounted, ESO key drift, typo, or a spec entry without `=`) silently registered the source with HMAC verification DISABLED — any unauthenticated POST to `/webhook/<name>` triggered a full diagnose cycle (and fixer churn under `--remedy`). Now:

- Registration fails closed: a missing/empty env var or malformed spec logs an `ERROR … source disabled (fail-closed)` and the source is NOT registered (requests 404).
- Defense in depth: should a source ever be registered with an empty secret, the handler rejects every request for it with 401 instead of skipping verification.
- Explicit opt-out: the literal spec `<name>=insecure-no-hmac` registers a deliberately unauthenticated source and logs a loud `UNAUTHENTICATED webhook source` warning at startup.

**Migration:** deployments that (knowingly or not) relied on an empty secret to run an unauthenticated source must either mount a real secret or switch the spec to `<name>=insecure-no-hmac`.

### Fixed — operator: diagnose/remediate CronJobs NEVER succeeded when `spec.alerting` was configured (watch-only flags leaked)

PRODUCTION BUG: the operator's `buildCronJobCommon` appended the watch-only alerting flags (`--alertmanager-url`, `--cluster-name`, `--slack-alerts`, `--slack-critical`) to BOTH CronJobs. `srenix diagnose` / `srenix remediate` register none of them, so on any operator-managed install with `spec.alerting` set the diagnose/remediate Jobs exited 1 with "unknown flag" on every run — the CronJobs had never succeeded on the live cluster. The chart's `cronjob-diagnose.yaml` / `cronjob-remediate.yaml` were always correct and are the reference shape:

- **diagnose CronJob** now renders `diagnose --live --format=daily` (the previously missing `--format=daily` is what produces the #healthinfo daily digest) plus `--slack-healthinfo=$(SLACK_HEALTHINFO_URL)` when `spec.alerting.slack.healthInfo` is configured; its env carries ONLY `SLACK_HEALTHINFO_URL` (was: all three `SLACK_*_URL`s).
- **remediate CronJob** renders `remediate --live [--dry-run=true]` with no alerting flags and no `SLACK_*` env (mirroring the chart).
- **Bug-class guard** — new `cmd/srenix/operatorflags_test.go` builds the watcher Deployment + both CronJobs from the bundle's full-surface sample CR (features force-enabled, resolved through the shared `internal/chartgate.SampleCRFullPath` fixture locator so a future `bundle/tests` move fails loudly in one place) and asserts every operator-rendered arg is registered on the REAL cobra subcommand (`newRootCmd()` in-process) — the operator-render sibling of the v1.23.0 chart↔flags parity gate, so ANY future builder emitting an unregistered flag fails CI instead of CrashLooping in production. Exact-args + role-scoped-env regression tests in `internal/operator/cronjob_args_test.go`.

### Fixed — operator: watcher Deployment carried a `SLACK_HEALTHINFO_URL` secretKeyRef nothing in watch mode reads

Pre-existing operator bug (since the alerting env was introduced, NOT introduced by the CronJob fix above): the builders' shared `alertingEnv` injected `SLACK_HEALTHINFO_URL` onto the WATCHER Deployment, but `srenix watch` registers no `--slack-healthinfo` (it is a diagnose-only flag) and nothing reads the env var directly. Because the operator emits a NON-optional `secretKeyRef`, an absent healthinfo secret hard-failed watcher pod creation (`CreateContainerConfigError`) over an env var that could never be consumed. `alertingEnv` now emits only the two channels the watcher actually expands (`SLACK_ALERTS_URL` / `SLACK_CRITICAL_URL`); the diagnose CronJob keeps its `SLACK_HEALTHINFO_URL` via the dedicated `healthinfoEnv`. The Helm chart never had this issue — `watcher-deployment.yaml` renders `srenix.slackAlertsEnv` + `srenix.slackCriticalEnv` only, and is the reference shape. Pinned by `TestWatcherDeploymentEnv_NoHealthinfoSecretRef`.

### Fixed — operator: `spec.diagnose.backoffLimit: 0` was silently overridden to 1

`DiagnoseSpec.BackoffLimit` was a plain `int32`, so an explicit `backoffLimit: 0` (retry-never — a legitimate posture for a read-only diagnose Job) was indistinguishable from unset and silently replaced with the default 1. The field is now `*int32`: nil (unset) defaults to 1 as before, explicit `0` is honored, explicit `N` passes through. Hand-maintained DeepCopy updated for the pointer; CRD schema is unchanged (`integer`, `minimum: 0` — pointer-ness is a Go-side concern), so the Go↔CRD and bundle↔chart parity gates pin the same shape. `BuildRemediateCronJob` no longer passes a misleading literal `0` (it passes nil — `RemediateSpec` has no backoffLimit knob). Table-driven nil/0/2 regression test in `internal/operator/builders_test.go`.

### Fixed — deployed watcher/aiwatch spammed ~2.5 log lines/sec of `v1 Endpoints is deprecated in v1.33+` client-go warnings

The DNSChainDrift analyzer issued a `Get` on deprecated core/v1 Endpoints per Ingress backend per diagnose cycle (plus the KongRoutes legacy fallback and `srenix snapshot capture`); every call made the API server attach the deprecation warning header, which client-go printed verbatim — pure noise drowning real signal. (The watcher's `watchedGVRs` never actually watched core/v1 Endpoints, so the volume was list/get-driven, not watch-driven.)

- **Migrated (mechanical, semantics preserved)** — `DNSChainDrift` now resolves the endpoint layer EndpointSlice-first (`internal/diagnose/dns_chain_drift.go`, new `serviceReadyAddressCount`): lists `discovery.k8s.io/v1` EndpointSlices in the Service namespace, links shards back via the `kubernetes.io/service-name` label, and counts ready addresses (`conditions.ready == nil` is treated as ready per the API contract). The deprecated core/v1 Endpoints `Get` survives ONLY as a fallback when no slice for the Service is visible — old snapshot captures and clusters without the EndpointSlice mirroring controller — keeping pre-migration semantics for those sources. `KongRoutes` (already slice-first since v1.23.0) now reuses the shared `snapshot.GVREndpointSlice`/`GVREndpoints` constants.
- **Watch trigger** — `watchedGVRs` gains `discovery.k8s.io/v1` EndpointSlice, so an endpoint-membership change (pod ready/unready behind a Service) triggers a debounced diagnose cycle directly via the canonical API; a guard test pins that the deprecated core/v1 Endpoints GVR is never (re)added to the watch set.
- **Kept + suppressed (deliberate)** — the remaining legacy reads (DNSChainDrift/KongRoutes fallbacks, `srenix snapshot capture` parity for old tooling) are intentional, so `pkg/snapshot.SuppressEndpointsDeprecationWarnings` installs a `rest.Config` WarningHandler that drops ONLY the `v1 Endpoints is deprecated` message and routes every other API-server warning through a deduplicating stderr writer (once per unique message per process instead of once per call). Wired into `internal/snapshot.buildConfig` (LoadLive + BuildKubeClientset), cmd/srenix's silence kubeconfig builder, and `pkg/snapshot.NewLiveSource` (applied to a config COPY, only when the caller hasn't installed a handler — this is the constructor srenix-enterprise's aiwatch uses, so the paid binary inherits the fix without a code change).
- **Snapshot parity** — `CaptureGVRs` + the File-source kind map gain EndpointSlice so offline diagnose feeds the new slice-first read path; legacy Endpoints stays captured for the fallback path and old tooling.
- **RBAC** — no change needed: both the chart's reader ClusterRole and the operator's `readerPolicyRules()` have carried `discovery.k8s.io/endpointslices` `get/list/watch` since v1.23.1 (verified against the chart↔operator parity gate).
- **Tests** — watcher: EndpointSlice in `watchedGVRs`, never-legacy-Endpoints guard, and an end-to-end `watchGVR` test (fake dynamic client) asserting an EndpointSlice ADDED event reaches the trigger channel; DNSChainDrift: slice-preferred without any legacy object, zero-ready slices NOT overridden by a stale legacy object, nil-ready-counts-as-ready, and legacy-fallback-still-works fixtures; pkg/snapshot: warning filter drops both current and future-version Endpoints deprecation texts, passes other warnings through, dedups passthroughs, respects caller-provided handlers, and never mutates the caller's config.

### Fixed — DNSChainDrift emitted non-enum severities; every DriftReport reconcile cycle failed CRD validation

Production bug: `DNSChainDrift` emitted diagnostics with `severity: "warn"` (duplicate-ingress-host, service-external-name-mismatch) and `severity: "error"` (missing-cloudflare-record, cloudflare-points-elsewhere, missing-ingress, ingress-orphan-service, service-no-endpoints), but the DriftReport CRD enum only allows `info|warning|critical` — so the watcher's driftreport reconcile failed on those subjects every cycle with `spec.severity: Unsupported value: "warn"`.

- **Source fix** (`internal/diagnose/dns_chain_drift.go`) — all seven emit sites now use enum values: `warn` → `warning`, `error` → `critical` (broken-chain findings — host unreachable / traffic dropped — are critical by the same scale the rest of the catalog uses).
- **Defense-in-depth** (`internal/report/driftreport.go`) — new `report.NormalizeSeverity(severity, source)` applied at the single choke point in `Reconcile` where both the create spec and the per-cycle spec-refresh patch are built: `warn`→`warning`, `error`/`err`/`fatal`/`crit`→`critical`, empty→`warning` (the existing AssembleEntries default), anything else→`warning` with a log line naming the offending source. A future emitter with a bad literal can no longer break reconcile.
- **Guard tests** (`internal/report/severity_test.go`) — (1) a table-driven regression test for the normalizer mapping; (2) `TestReconcile_NormalizesNonEnumSeverity` asserting both the create path and the spec-patch path send enum values; (3) `TestSeverityLiteralsAreEnumValues`, a static AST lint that walks every non-test `.go` file under `internal/`, `catalog/`, `pkg/`, `cmd/`, `api/` and fails on any `Severity` string literal outside the enum — this test caught exactly the seven production sites before the fix.

### Fixed — chart: silence ClusterRoleBinding bound the wrong ServiceAccount (live-verification finding)

The chart's `clusterrole-silence` binding referenced the watcher SA via `srenix.fullname` instead of `srenix.serviceAccountName` (`<fullname>-sa`) — the SA the watcher Deployment actually runs as. So on chart installs the watcher could not `list silences`, and silence-filtering was **skipped on every diagnose cycle** (live symptom: `silences.srenix.ai is forbidden`). Found by deploying the merged build to a real (RBAC-enforcing) kind cluster; unit tests missed it because the chart↔operator RBAC parity test excludes the `srenix.ai` group. Fixed the binding + added `internal/operator/chart_watcher_binding_sa_test.go` asserting every watcher read-role binding uses `srenix.serviceAccountName`.

### Fixed — watcher: pending approval-URL cache grew unbounded (P1.9)

The `pendingURLs` map (approval URLs keyed by ActionID for the AI tier) evicted entries only on lookup (`approvalURLFor`). A recorded-but-never-rendered ActionID — e.g. a diagnostic that resolved before its next post — persisted for the whole process lifetime, a slow memory leak on long-running watchers with the AI tier enabled. `recordApprovalURL` now sweeps entries older than a 24h TTL on every insert (and lookup still evicts on access), via an injectable clock seam.

### Fixed — operator: cross-namespace approval events Role/RoleBinding leaked on CR deletion (P1.9)

When a CR pinned `spec.approval.auditNamespace` to a namespace other than its own, the operator created the `<name>-events` Role + RoleBinding there **without** an ownerRef (cross-namespace ownerRefs are illegal), so Kubernetes GC never reaped them. Teardown only ran on disable-while-alive — a straight `kubectl delete` of the CR skipped that path, leaking the cross-namespace RBAC pair for the cluster's lifetime. The operator's finalizer now also deletes those objects (NotFound ignored, so the same-namespace owner-ref'd case is a harmless no-op).

### Fixed — feeder: workload digest index collided across workloads in a namespace (P1.6)

The workload feeder's pod-digest index was keyed by (namespace, container-name) only, despite a comment claiming it was scoped to the owning controller. Two workloads in one namespace that both name their container e.g. `app` (extremely common) silently received each other's `image_digest` — first pod observed won — so a digest-pin PR proposal built downstream could previously cite a **sibling workload's digest** and pin the wrong image. The index is now scoped to the owning workload (each pod's controller ownerReference, with Deployment names recovered from the ReplicaSet `<deployment>-<pod-template-hash>` convention), and as a second guard a digest only attaches when the repo it was pulled from matches the workload's declared image repo (so a mid-rollout pod still running the old repo's image can no longer stamp its digest onto the new spec). Pods whose owner can't be resolved (bare pods, Jobs, bare ReplicaSets) now contribute no digest — fail-closed; the entry simply omits `image_digest` until a resolvable pod is observed.

### Fixed — operator: `spec.externalDNS` was accepted but did nothing (P1.5)

The CRD documented `spec.externalDNS.cloudflare.*` (incl. `apiTokenSecretRef`) and the operator accepted it — but consumed it nowhere. The DNSChainDrift analyzer only wires its Cloudflare client when `SRENIX_CLOUDFLARE_TOKEN` is set at registration time, and nothing supplied that env on operator-managed installs, so external-hop DNS verification silently never ran. The operator's watcher Deployment now injects `SRENIX_CLOUDFLARE_TOKEN` via `secretKeyRef` from `apiTokenSecretRef.{name,key}` (key defaults to `token`) when `cloudflare.enabled=true`. The token value never appears in any manifest.

### Fixed — operator: `spec.watcher.triggers.webhook.serviceEnabled` was accepted but did nothing (P1.5)

The chart has shipped `watcher-webhook-service.yaml` since v1.23.0, but the operator built neither the ClusterIP Service nor the named `webhook` containerPort — an operator-managed webhook receiver was reachable only by pod IP. The operator now reconciles a `<cr>-webhook` ClusterIP Service (port = `servicePort`, default 8090; `targetPort: webhook`; selects the watcher pods) when `serviceEnabled=true`, owner-ref'd to the CR and torn down when the field flips off or the watcher is disabled, and declares the `webhook` containerPort whenever `webhook.listen` is set — both mirroring the chart's semantics exactly.

## [1.25.1] — 2026-06-11

### Fixed — goreleaser disk-OOM on GH-hosted runner

v1.25.0 goreleaser failed at the docker buildx multi-arch build stage with `no space left on device`. The OSS workflow's transitive deps (AWS SDK v2 + k8s.io + buildx cache) overshoot the ~14 GiB free disk on the GH-hosted runner. v1.24.x and earlier just happened to fit; v1.25.0's added KEDA + extra ownerRef walker pushed past the limit.

Same fix that the Srenix Enterprise workflow shipped in v1.20.0: pre-checkout cleanup step removes ~25 GiB of preinstalled .NET / Android SDK / Haskell / Swift / CodeQL toolchains that the workflow doesn't use.

Chart 1.25.0 was published successfully to gh-pages before goreleaser exited; this patch re-publishes the chart at 1.25.1 alongside the new image so operators always pull a coherent pair. No code changes — v1.25.0 and v1.25.1 ship byte-identical Go binaries.

### Added — `spec.remediate.activeDeadlineSeconds`

The diagnose CronJob already had `spec.diagnose.activeDeadlineSeconds` (default 300s). The remediate counterpart was hardcoded at 120s in the operator builder. Busy clusters with many SecurityDrift proposals + DigestPin candidates queued up routinely overshoot 120s and hit BackoffLimitExceeded. Live observation on the dev cluster (2026-06-11): 4 of 4 most-recent runs had `cond: Failed=True reason=BackoffLimitExceeded`.

Operators can now set `spec.remediate.activeDeadlineSeconds: 900` (or whatever their workload needs). Default 120s preserved for low-finding clusters where it's fine.

## [1.25.0] — 2026-06-11

Two follow-ups after live deployment surfaced operator-managed gaps + Slack-flood symptoms.

### Added — Per-workload dedup in SecurityDrift digest-pin findings

Before v1.25.0 the SecurityDrift `checkMutableImageTags` analyzer emitted one Diagnostic per Pod. A 3-replica Deployment + 9-DaemonSet-Pod calico-node → 12 Slack alerts every cycle, each identical except for the Pod-name suffix. v1.25.0 collapses by `(namespace, controller-owner-name, sorted-unpinned-image-set)`:

- 3 Pods of one ReplicaSet → 1 diagnostic, `Subject="Workload/ns/rs-name"`, message says "(across 3 replica pods)"
- Different image versions during a rolling update → 2 diagnostics (one per RS), correctly distinct
- Standalone Pods (no controller) → fall back to per-Pod identity so they still surface
- Severity is the union: any one warning-class image in the group upgrades the whole group

2 new tests cover dedup + rolling-update distinctness. Existing SecurityDrift tests still pass with the new Subject shape.

### Added — Operator-managed workloads synthesize `owner_chart`

The workload-feeder previously read `owner_chart` only from Helm release labels (`helm.sh/chart` + `meta.helm.sh/release-name`) + ArgoCD `instance` annotation. Operator-managed Deployments (created directly by a Custom Resource controller, no Helm labels) had `owner_chart=None` in RAG → the DigestPinProposer couldn't find a `values.yaml` to target → silently skipped → no PR opened → no Approve/Reject buttons in Slack.

v1.25.0 walks the OwnerReferences chain and synthesizes:

- `owner_kind = "Operator"`
- `owner_chart = "<crkind-lowercase>-<crname>"` (e.g. `agenticsre-bionic`)
- `owner_release = <CR name>`
- `owner_release_namespace = workload namespace`

Built-in workload parents (apps/v1 ReplicaSet, batch/v1 Job, core/v1) are explicitly skipped — they mean "this Pod is owned by a Deployment", not "this Deployment is operator-managed". 2 new tests cover both the positive (operator CR owner → synthesized) and negative (apps ReplicaSet owner → still nil) cases.

`detectOwner` no longer early-returns on nil annotations — operator-managed workloads typically have NO annotations at all, so the nil-anns path must still walk the OwnerReferences fallback.

## [1.24.1] — 2026-06-10

### Fixed — CRD schema for `spec.watcher.triggers` (v1.24.0 was unusable on schema-strict K8s)

v1.24.0 added the Go types + operator reconciler for `spec.watcher.triggers.{prom,webhook}` but did NOT update the CRD's OpenAPIv3 schema. K8s 1.27+ structural-schema pruning stripped the field at the API server, so any `kubectl apply` of a CR with `triggers` set silently dropped the data. The operator then rendered the watcher Deployment with no trigger args.

This patch adds the matching schema to both `bundle/manifests/srenix.ai_agenticsres.yaml` and `charts/agentic-sre/templates/crd-agenticsre.yaml`. Verified live: `kubectl explain agenticsres.spec.watcher.triggers` now resolves and the field persists on `kubectl get`.

Caught during live activation of M5 on the dev cluster (kubectl apply succeeded with a warning, but the field was stripped silently — operator rendered no trigger args).

## [1.24.0] — 2026-06-10

Adversarial-review follow-up: operator-CR triggers + KEDA expansion.

### Added — `spec.watcher.triggers.{prom,webhook}` typed CR fields

The chart's v1.23.1 `watcher.triggers.*` values knobs activated M5/M6 for chart-managed installs, but operator-managed (ArgoCD/Flux/kubectl-apply) installs couldn't reach them from the CR — they had to thread Helm values around. This release adds the typed surface:

- `WatcherTriggersSpec.Prom {URL, Interval, AlertNameFilter}` → renders `--prom-trigger-url/interval/alert-filter`
- `WatcherTriggersSpec.Webhook {Listen, Sources, SecretName, ServiceEnabled, ServicePort}` → renders `--webhook-listen/source` + projects every `<src>=<env-var>` source's env-var from the named Secret + (optionally) a ClusterIP Service

Operator's `BuildWatcherDeployment` reads from `cr.Spec.Watcher.Triggers` via two new helpers (`watcherTriggerArgs`, `watcherTriggerEnv`). Legacy CRs (no Triggers stanza) render byte-identical to v1.23.1.

### Added — KEDA `ScaledObject` in `watchedGVRs` (M1 follow-up)

The v1.6.0 M1 expansion added HPA + Ingress + ArgoCD + DaemonSet to the watcher's inform-loop set but missed KEDA's `keda.sh/v1alpha1/ScaledObject`. The memory note `keda-paused-scaledobject` documents the production failure mode: paused annotation set out-of-band → silent 502-after-oauth-login cascade. This release adds it:

- `GVRScaledObject` constant in `internal/snapshot`
- `watchedGVRs` includes it (auto-skip when KEDA isn't installed)
- Chart `clusterrole-reader.yaml` + operator `rbac_builders.go` both grant `keda.sh/scaledobjects` get/list/watch (no-op when KEDA absent)

1 new test asserts the GVR is present.

### Pairs with Srenix Enterprise

`v1.21.0+` adds observability log lines for Phase 3.B (auto-merge gate armed at startup) and Phase 3.C (`ai.target_history.applied` audit event when the prompt block fires).

## [1.23.1] — 2026-06-10

Adversarial-review fixes after v1.23.0 went out.

### Fixed — webhook HTTP server actually starts now (M6 wiring gap)

v1.23.0 shipped `internal/server/webhook.Handler` + tests but
nothing in the watcher's `Run()` instantiated an HTTP server, so
M6 was compiled-but-never-loaded in production. This release wires
the receiver: `watcher.Config.WebhookListen` + `WebhookSourceSpec`
fields, `--webhook-listen` + `--webhook-source` CLI flags, an
`http.Server` mux serving `/webhook/` + `/healthz` with graceful
shutdown on `ctx.Done()`, Helm `watcher.triggers.webhook.*` values
knobs, and a new `watcher-webhook-service.yaml` Service template
with `secretKeyRef` env-var projection per registered source.

### Fixed — Prometheus trigger CLI flags actually exist (M5 wiring gap)

v1.23.0 shipped `Config.PromTriggerURL` but `cmd/srenix/watch` never
registered matching CLI flags, so M5 was unreachable from
`helm install` or `kubectl apply`. This release adds
`--prom-trigger-url`, `--prom-trigger-interval`, and
`--prom-trigger-alert-filter` + Helm `watcher.triggers.prom.*`.

### Fixed — `endpointslices` RBAC for KongRoutes (M2)

KongRoutes prefers `discovery.k8s.io/v1.EndpointSlice` for
backend-readiness, but neither chart nor operator RBAC granted it.
Silently fell back to legacy `v1.Endpoints` (still works) so this
wasn't fatal — but the slice fast-path was dead code. Now granted
in `clusterrole-reader.yaml` and `internal/operator/rbac_builders.go`.

## [1.23.0] — 2026-06-09

Trigger-expansion roadmap M1-M7 bundled. Closes the
`docs/design/2026-05-trigger-expansion-roadmap.md` plan that v1.6.0
opened M1 against.

### Added — M1 expanded `watchedGVRs`

Adds `Ingress`, `HorizontalPodAutoscaler`, and `ArgoCD Application`
to the watcher's inform-loop set.

### Added — M2 `KongRoutes` probe

For each Kong-managed Ingress, verifies the backend Service has ≥1
ready Endpoint + KongPlugin / KongConsumer annotation references
resolve. Silent on clusters without Kong-managed Ingresses. Opt out
via `SRENIX_PROBE_KONG_ROUTES=off`.

### Added — M3 `GPUNodes` probe + `LogPatternMatcher` analyzer

- **GPUNodes** — critical on NotReady / zero-allocatable, warning
  on cordoned, for each GPU-advertising Node. Opt out:
  `SRENIX_PROBE_GPU_NODES=off`.
- **LogPatternMatcher** — scans Events for ImagePullBackOff,
  OOMKilled, probe-failed, volume-attach-failed, RBAC Forbidden.
  Dedup'd per (involved-object, pattern). Opt out:
  `SRENIX_ANALYZER_LOG_PATTERN_MATCHER=off`.

### Added — M4 Endpoints probe Layer-7 mode

`EndpointTarget.L7` populated from three Ingress annotations
(`srenix.ai/probe-l7-{path,expect,status}`). When
set, second GET asserts both status + body content. Closes the
"Kong returns 200 but body is wrong" failure class.

### Added — M5 Prometheus class-C trigger

`internal/trigger/prom`. Polls Alertmanager `/api/v2/alerts` and
pushes a debounced signal on new firing-alert fingerprints. Closes
the slow-drift gap (disk fill, cert expiry creep, error-budget
burn, GPU ECC accumulation). New `Config` fields:
`PromTriggerURL`, `PromTriggerInterval` (clamped ≥5s),
`PromTriggerAlertFilter`.

### Added — M6 external webhook receiver (class E)

`internal/server/webhook`. HMAC-SHA256-authenticated POST to
`/webhook/<source>` triggers an immediate diagnose cycle. `Sign()`
exported for external integrators. Closes "rotation → probe within
seconds" loop.

### Added — M7 `pkg/probe.GVRWatcher` foundation

Optional interface for probes to declare consumed GVRs.
`GVRsOf(probe)` reads them; nil = "run on every trigger" (back-
compat). Applied to KongRoutes + GPUNodes as exemplars. Sets up
phase 2 (per-probe dispatch) and phase 3 (controller-runtime
migration) without changing today's semantics.

### Tests

35+ new tests across the milestones; full regression green.

## [1.22.2] — 2026-06-09

### Fixed — PVOrphan needs `persistentvolumes` in RBAC

v1.22.1 added PV capture (`GVRPV` in CaptureGVRs) but the watcher's
reader ClusterRole still only granted `persistentvolumeclaims`. The
PV list call silently failed with RBAC denial, so PVOrphan kept
emitting nothing on live clusters.

Adds `persistentvolumes` to:
- `charts/agentic-sre/templates/clusterrole-reader.yaml`
- `internal/operator/rbac_builders.go` (used in operator-managed installs)

Verified live: with the live ClusterRole patched and the watcher
restarted, PVOrphan now fires on the dev cluster's 117 Released PVs.

## [1.22.1] — 2026-06-09

### Fixed — PVOrphan needs `persistentvolumes` in CaptureGVRs

The v1.22.0 PVOrphan analyzer was silent on live clusters because
`internal/snapshot.CaptureGVRs` didn't include PVs (PVCs were
captured separately; PVs are their own cluster-scoped GVR). Adds
`GVRPV` to the capture list and refactors PVOrphan to consume the
shared constant. Verified live: with 117 Released PVs on the dev
cluster, the analyzer now fires the expected warnings.

## [1.22.0] — 2026-06-09

Phase 3.E + 3.D bundled.

### Added — 3 new workload-tier analyzers (Phase 3.E)

The 3 most-requested signals from the deferred wishlist:

- **`OOMKillRecurrence`** (warning) — Pod container with ≥3 OOMKilled
  restarts in 24h. Catches the sizing problem masquerading as a crash
  loop. One finding per pod (operator's edit pass fixes all containers
  simultaneously). Opts out via `SRENIX_ANALYZER_OOMKILL_RECURRENCE=off`.
- **`PVOrphan`** (warning) — PersistentVolume in `Released` phase for
  >7d. Underlying cloud disk (EBS / GCE-PD / Azure-Disk) may still be
  billing. Message surfaces storageClass + capacity + reclaimPolicy
  for cost-sizing. Opts out via `SRENIX_ANALYZER_PV_ORPHAN=off`.
- **`CronJobStuck`** (warning/critical) — CronJob whose lastSuccessfulTime
  is >24h old OR has never succeeded OR is suspended. Each cause gets
  tailored remediation guidance. Opts out via `SRENIX_ANALYZER_CRONJOB_STUCK=off`.

### Added — `spec.ai.metrics` + `spec.ai.llmProposer` typed CR fields (Phase 3.D)

Promotes two Phase 2 surfaces from chart-only / extraArgs-hatch into
typed CR fields so operator-managed installs (ArgoCD/Flux/kubectl apply)
don't need escape hatches.

- `AIMetricsSpec {Addr, Port}` — operator renders `--metrics-addr` arg +
  named container port + headless Service. Selectors target aiwatch pods
  so Prometheus pod-discovery sees per-pod endpoints (leader vs follower
  stay distinct in `srenix_cycle_total{leader=...}`).
- `AILLMProposerSpec {Enabled}` — typed switch for the Phase 2.D LLM
  fallback proposer.

CRD schema additions on both chart-side template and OLM bundle manifest.
3 helm-template invariants preserved: legacy installs (no Metrics / no
LLMProposer fields) render byte-identical to v1.21.1.

### Pairs with Srenix Enterprise

The Srenix Enterprise binary `--metrics-addr` + `--llm-proposer` flags ship since
v1.16.0; this release wires them through the operator schema. Cluster
operators can now drop their `extraArgs: ["--metrics-addr=:9090"]`
escape hatch in favor of `spec.ai.metrics.addr: ":9090"`.

## [1.21.1] — 2026-06-08

Follow-up to the v1.21.0 Phase 2 closure. Adds the
`spec.ai.digestPinAttestation` field that the v1.21.0 merge missed
(the chart-version bump from v1.20.1 → v1.21.0 was also missed at
tag time; this release bumps both together).

### Added — `spec.ai.digestPinAttestation` chart wiring (Phase 2.H)

`DigestPinAttestationSpec {SecretName, SecretKey, KeyID}` on AISpec.
When set, the chart mounts the Secret at `/etc/srenix/attestation/` and
passes `--digest-pin-attestation-key` + `--digest-pin-attestation-kid`
to the aiwatch container. Operator reconciler mirrors the chart.
Mount path is separate from `/etc/srenix/keys/` so attestation key
rotation is independent of the approval-server signing key.

### Fixed — `internal/report.DeltaDiag` class-URL docs (Phase 2.B.6)

The render-only class-URL fields shipped in v1.21.0 — `ApproveClassURL`,
`DenyClassURL`, `SilenceClassURL` — now carry a doc clarifying that
the OSS enrich pipeline does NOT mint class-action JWTs (the signer
lives in Srenix Enterprise's `ai/approval`). The Srenix Enterprise aiwatch's renderer
(`cmd/srenix-enterprise/render.go`) is the active surface; the OSS render is
preparatory for a future shared-signer extraction.

### Pairs with Srenix Enterprise

`v1.16.0+` (binary-side surfaces are unchanged from v1.21.0 →
v1.21.1; only the chart's wiring of an existing Srenix Enterprise flag is new).

## [1.21.0] — 2026-06-08

Phase 2 closure on the OSS side. Pairs with Srenix Enterprise `v1.16.0`
for the paid-tier binary half.

### Added — `spec.ai.replicas` for HA aiwatch (Phase 2.F)

`AgenticSRE.spec.ai.replicas` (`int32`, min 1 max 5).
Default 1 (single-replica, noop elector — byte-identical to pre-2.F).
When `>1`, the chart turns on `--leader-election=true` + binds the
SA to a scoped Lease Role; the binary races for a
`coordination.k8s.io/v1.Lease` named `<release>-aiwatch-leader`.
Failover within ~30s on lease loss.

### Added — Prometheus instrumentation + Grafana dashboard + canary alerts (Phase 2.G)

`ai.metrics.{addr,port,serviceMonitor,grafanaDashboard,prometheusRule}`
values opt in to: aiwatch `/metrics:9090` headless Service +
optional `ServiceMonitor` + `dashboards/srenix-overview.json` ConfigMap
(Grafana sidecar labels) + `PrometheusRule` canaries
(`ChaWatcherStuck`, `ChaBreakerOpen`, `ChaAutonomyRejectionSpike`).

All gated on `ai.enabled` + non-empty `ai.metrics.addr` — pure-OSS
deploys see no new resources.

### Added — Slack class-button render row (Phase 2.B.6)

`internal/report.DeltaDiag` gains `ApproveClassURL` /
`DenyClassURL` / `SilenceClassURL`. When populated, `FormatSlackDelta`
renders an extra row under the Approve/Deny pair. Render-only on
OSS — the OSS enrich pipeline does NOT yet mint class-action JWTs
(the signer lives in Srenix Enterprise). Srenix Enterprise aiwatch's renderer
(`cmd/srenix-enterprise/render.go`) is the active surface in production.

### Added — `Silence.spec.matcher.messagePattern` (Phase 2.B.9)

Substring-match on `Diagnostic.Message`. Enables class-scoped
silences from the Srenix Enterprise `/silence-class` click. `pkg/silence.Matches`
ANDs MessagePattern alongside Source + Subject + Severity.

### Added — `DisruptionDrift` analyzer (Phase 2.E)

Three new signals: **PDB blocks all evictions** (`critical`),
**stuck Indexed Job failed indexes** (`warning`), **stale
ResourceQuota at 100%** (`warning`). Opts out via
`SRENIX_ANALYZER_DISRUPTION_DRIFT=off`.

### Added — `spec.ai.digestPinAttestation` chart wiring (Phase 2.H)

`DigestPinAttestationSpec {SecretName, SecretKey, KeyID}` on AISpec.
When set, chart mounts the Secret at `/etc/srenix/attestation/` and
passes `--digest-pin-attestation-key` + `--digest-pin-attestation-kid`
to the aiwatch container. Operator reconciler mirrors the chart.
Mount path is separate from `/etc/srenix/keys/` so attestation key
rotation is independent of the approval-server signing key.

### Pairs with Srenix Enterprise

`v1.16.0+` carries the binary-side surfaces this chart drives:
class-action JWT routes, `/metrics` endpoint, attestation signer,
lease elector, LLM proposer, autonomy class-policy bypass.

## [1.20.1] — 2026-06-07

### Fixed — Finish Phase 1.B placeholder substitution across analyzers (PR #169)

PR #164 (shipped in v1.18.3 / re-stated in v1.19.0) addressed Phase 1.B by substituting `<name>` / `<selector>` placeholders in `capacity_drift.go` + `config_drift.go`. The post-Phase-1 adversarial audit caught 4 more analyzers still leaking literal `<placeholder>` tokens that neither the AI tier could parse nor operators could action without manual lookups:

  - **`security_drift.go`** digest-pin remediation — now reads `status.containerStatuses[].imageID` (kubelet has already resolved every running image to a sha256 at pull time) and renders per-container substitution `Replace foo:1.2.3 with foo@sha256:…`. Strips the `docker-pullable://` kubelet prefix. Falls back to a concrete `crane digest <actual-image>` invocation when the Pod hasn't been scheduled.
  - **`dns_chain_drift.go`** missing-ingress remediation — refactored into `renderMissingIngressRemediation(host)` helper that renders a copy-pasteable Ingress YAML skeleton with the actual host substituted.
  - **`rbac_drift.go`** unbound-SA remediation — reworded "Pick a Role (list candidates with `kubectl get role`) … `--role=NAME (substitute NAME …)`" — no bare `<role-name>` token.
  - **`workload_state_drift.go`** CNPG follower remediation — reworded "Identify the non-Ready follower, then `<that-pod-name>` (substitute the pod name from the prior list)" — no bare `<follower-pod>` token.

Audit grep across `internal/diagnose/*.go` (non-test) Remediation strings for the strict token set `<name>|<placeholder>|<image>:<tag>|<selector>|<digest>|<svc-name>|<port>|<role-name>|<follower-pod>` now returns **0 hits**.

### Fixed — Align ticketing values shape with CRD (PR #170)

Chart values block used nested `ticketing.openproject.{mcpURL, projectID, …}`; CRD uses flat `ticketing.{mcpURL, project, …}`. Users could not move YAML between `helm upgrade -f values.yaml` and `kubectl patch srenix …` without reshaping. Fixed by flattening the chart shape to mirror the CRD exactly; legacy nested form honored as a fallback (will be removed in the next major chart bump).

## [1.20.0] — 2026-06-07

### Added — Operator-managed `spec.ticketing` (Phase 1.D, PR #167)

Closes the Helm-values-vs-operator-managed-CR gap for issue-tracker delivery. Before this release, the chart had full `ticketing.*` Helm values + the `cmd/srenix` flags + the `pkg/ticketing/openproject.Sink` implementation, but the operator's `BuildWatcherDeployment` never emitted any `--ticketing-*` flag and the CRD had no `spec.ticketing` field. Operators who set Helm values saw no effect on operator-managed installs.

New `spec.ticketing` (TicketingSpec) on the CR drives flags + env injection:

  - `--ticketing-provider` ← `spec.ticketing.provider` (openproject / jira / servicenow enum)
  - `--ticketing-mcp-url` ← `spec.ticketing.mcpURL`
  - `--ticketing-project` / `--ticketing-type-id` / `--ticketing-closed-status-id` ← matching CR fields
  - `--ticketing-priority-{critical,warning,info}` ← `spec.ticketing.severityPriority.*`
  - `--ticketing-web-url-prefix` ← `spec.ticketing.webURLPrefix`
  - `--ticketing-labels=<L>` (one flag per label) ← `spec.ticketing.labels[]`
  - `--ticketing-dry-run` ← `spec.ticketing.dryRun: true`
  - `TICKETING_MCP_API_KEY` env via secretKeyRef ← `spec.ticketing.auth.{secretName,secretKey}` (only when `auth.enabled`)

Schema is OpenProject-shaped (OSS's only supported provider); the enum allows jira/servicenow so Srenix Enterprise can add them additively without a v1alpha2 bump.

Disabled (the default) emits zero flags so existing CRs stay byte-identical.

## [1.19.0] — 2026-06-05

### Added — Per-cycle delta render: "🆕 New this cycle" + stable-collapse + opt-in no-change digest (PR #165)

Operators reading #ceph-critical can't tell at-a-glance which findings just appeared vs. which are stale-but-getting-reposted. With 50+ findings per cycle the "what should I look at right now" signal drowns. Three signal:noise improvements layered together:

- **"🆕 New this cycle (N):" section** renders ABOVE the legacy critical/diagnostics list. `watcher.diff()` marks `entry.isNewThisCycle=true` on new-subject + fp-changed paths, false on repeat-interval re-posts. Zero-count section headers are suppressed (no `(0)` clutter).
- **Stable-collapse** (always on when new findings exist): the steady-state list collapses to a single `"…and N other stable finding(s) already posted in earlier cycles"` line. Cycles with 0 new findings render stable in full (operator is reading the periodic re-post — every finding matters).
- **"✨ No new issues" digest** (opt-in via `--slack-no-change-digest=true`): on cycles where `newCount == 0 && resolved == 0 && stable > 0`, replaces the full re-post with a compact steady-state confirmation. Default OFF to preserve byte-identical legacy behaviour.

Wiring chain: `cmd/srenix --slack-no-change-digest` → `watcher.Config.NoChangeSlackDigest` → `report.RouteAndPostConfig` → `report.SplitCriticalPayloadsConfig` → `report.emitNoChangeDigest`. Legacy entry points (`SplitCriticalPayloads`, `RouteAndPost`) preserved as zero-config delegates.

### Fixed — Substitute placeholders in remediations (PVC StorageClass, Deployment selector) (PR #164)

Analyzer remediations contained literal `<placeholder>` tokens the operator was expected to substitute by hand. The AI tier surfacing these diagnostics had no way to interpret them, and operators reading Slack alerts had no way to act without manual lookup.

- **`capacity_drift.go` (PVC expansion stuck)**: looks up the PVC's `spec.storageClassName`, fetches the StorageClass from the snapshot, and emits one branch-collapsed remediation per `allowVolumeExpansion` value. `<name>` placeholder no longer leaks even in fallback paths (SC unknown / no `spec.storageClassName`).
- **`config_drift.go` (Deployment rollout stuck)**: reads `spec.selector.matchLabels` and renders the actual `-l key=val,...` flag (keys sorted for stable output). The kubectl proposer can now generate concrete commands instead of templates.

Out-of-scope tokens audited + kept as-is when they represent operator-intent input (e.g. `<svc-name>:<port>` for an Ingress whose backend Service doesn't exist yet) or legitimate format literals (`@sha256:<digest>`).

## [1.18.3] — 2026-06-05

### Fixed — Slack-bound AI tier fields restored via seenEntryToDeltaDiag helper (PR #160)

- `internal/watcher/watcher.go::runCycle` had two inline mappings from `seenEntry → DeltaDiag` — one for Alertmanager and one for Slack-bound `toPostDiags`. PR #59 (ticketing M1) updated the AM path to carry `ProposedActionID` + `ApprovalURL` but missed the Slack path. No test pinned the AI-field flow through to Slack so it went unnoticed.
- **Operator impact (diagnosed 2026-06-04)**: every AI-tier proposal (NetworkPolicy ManifestBridge, DigestPin) had a working signed approve/deny URL at AM but no `✅ Approve · ❌ Deny` line in Slack. Operators couldn't action proposals.
- Fix: collapse both mappings into single `seenEntryToDeltaDiag` helper. 2 regression tests added.

### Added — SplitCriticalPayloads chunking + actionable-first sort (PR #161)

- **Chunking**: Slack silently truncates webhook attachment text past ~40K chars. A cycle with 118 findings rendered to 115K → alphabetically-late findings (incl. storethesoup with a real Approve URL) cut from displayed message. `SplitCriticalPayloads(unfixable, resolved)` greedily packs per-finding strings into chunks ≤ 35K, posts each as own SlackPayload, adds `_(part N/M)_` marker.
- **Actionable-first sort**: within each severity block, findings with `ApprovalURL` sort ahead of findings without — promotes them into Slack's ~3-4K inline-preview window. Alphabetical-by-subject is secondary key.
- `FormatCriticalPayload` retained for backwards-compat. 3 regression tests added.

### Fixed — Detect always falls through to raw scan on Helm probe error (PR #162)

- Previous: `Detect` halted on any non-`ErrNotFound` error from `DetectInHelmValues`. GitHub's secondary rate limit returns HTTP 403 indistinguishably from real auth scope failure to the forge layer, so transient bursts blocked all downstream digest-pin work.
- Diagnosed 2026-06-04: 35 of 41 candidates erroring on `charts/X/values.yaml: HTTP 403` while direct curl returned clean 404 seconds later.
- Fix: `Detect` always falls through to `DetectInRawManifests` regardless of Helm's error. Real auth failures still surface via raw scan's own `ListRepoFiles` call.
- Test renamed `TestDetect_HelmTransportError_FallsThroughToRaw` with updated contract.

---

### Added — Raw-YAML inline-image detector (`releasesrc.DetectInRawManifests`, v1.18.2)

- **Problem**: many in-house repos ship plain Kubernetes YAML (Deployment / StatefulSet specs with `image: <repo>:<tag>` inline) instead of a Helm chart with `image.repository`/`tag` keys. The v1.18.1 `DetectInHelmValues` returned `ErrNotFound` for those, causing the digest-pin proposer to silently skip — the gap that prevented buttons from appearing on `docker4zerocool/storethesoup-wordpress` even after the cluster rolled to v1.18.1.
- **New `DetectInRawManifests(ctx, files, expectRepository)`** — lists every `*.yaml`/`*.yml` in the repo (via `RepoFiles.List(["**/*.yaml", "**/*.yml"])` with no-pattern fallback for forge backends that ignore the glob), scans each for an `image: <repo>:<tag>` line, returns first hit with `File` + 1-based `Line` + `CurrentTag`. Anchors on `^\s*-?\s*image\s*:` so it doesn't match keys whose name happens to end in `image:`. Accepts both quoted (`image: "repo:tag"`) and unquoted forms; tag charset matches OCI (`a-z A-Z 0-9 . - _`). Skips non-YAML files (a `Dockerfile` or `README.md` with an `image:` line is not a K8s manifest).
- **New `Detect(ctx, files, chartName, expectRepository)`** — single entry that tries `DetectInHelmValues` first (preferred edit anchor; tag value substitutes cleanly into the chart template), then falls back to `DetectInRawManifests`. Transport errors from the Helm probe propagate — Detect does NOT silently paper over genuine forge outages by trying the raw scan.
- **9 new tests**: storethesoup-k8s shape (wordpress Deployment), quoted image, multi-document YAML (matches first), no-match repo, empty repo, nil-args guards, non-YAML file skip, Helm priority over raw, Helm transport error propagation through `Detect`.
- **Chart bump 1.18.1 → 1.18.2** (patch — new function, no behavior change in existing API).

---


### Changed — Promoted `internal/feeder` → `pkg/feeder` (v1.18.1)

- **The workload feeder is now importable from external Go modules** (the paid srenix-enterprise binary in particular). Go's `internal/` visibility rule was blocking the srenix-enterprise aiwatch from instantiating `WorkloadFeeder` — meaning `kind=workload` entries were never being written to RAG, meaning the v1.11.0 srenix-enterprise `DigestPinProposer` would always miss its RAG lookup, meaning **no Approve/Deny buttons would have appeared on digest-pin findings even after the cluster rolled to v1.11.0**.
- **Mechanical move**: `git mv internal/feeder pkg/feeder`. The 4 Kubernetes GVRs (`Pod`, `Deployment`, `StatefulSet`, `DaemonSet`) the feeder needs are now defined locally in `pkg/feeder/workload.go` since `pkg/snapshot` doesn't carry them and `pkg/` cannot import `internal/snapshot`. No logic changes.
- All 13 existing feeder tests still pass.


### Added — `spec.ai.extraArgs` + `spec.ai.extraEnv` escape hatches on the operator (v1.18.0)

- **`api/v1alpha1/agenticsre_types.go`** — new `AISpec.ExtraArgs []string` + `AISpec.ExtraEnv []AIExtraEnv` (with `AIExtraEnvSource` + `AIExtraEnvSecretKeyRef`).
- **Why**: srenix-enterprise v1.11.0 ships new flags (`--cloudflare-feeder`, `--rag-store-url`, `--cluster-name`, `--digest-pin-proposer`, `--forge-token-env`, `--digest-pin-repo-map`) and env vars (`GITHUB_PAT`, `CLOUDFLARE_API_TOKEN`) that the operator's typed schema doesn't yet model. The escape hatches let operators wire them today via the existing CR-patch flow while typed fields land in subsequent minor releases.
- **`internal/operator/builders.go::aiArgs`** — appends `ai.ExtraArgs` AFTER the typed args so a typed flag wins on duplicate keys (later args override earlier in pflag).
- **`internal/operator/builders.go::aiEnv`** — appends `ai.ExtraEnv` entries as `corev1.EnvVar`, supporting either literal `Value` or `ValueFrom.SecretKeyRef`. ConfigMapKeyRef / FieldRef / ResourceFieldRef are deliberately out of scope (aiwatch never needs them).
- **CRD schema updated** — both the chart-managed `crd-agenticsre.yaml` and the bundled `bundle/manifests/srenix.ai_agenticsres.yaml` accept the new fields with kubebuilder validators (`minLength=1` on `name`/`key`).
- **3 new operator builder tests** — `ExtraArgs_AppendedAfterTypedArgs` (order check), `ExtraEnv_SecretRefAppended` (both `ValueFrom.SecretKeyRef` + literal `Value` paths), `ExtraArgsEmpty_NoChange` (defensive baseline).

### Added — `ActionProposePullRequest` ActionKind (Phase 2d-γ-3 slice 3a)

- **`pkg/ai/types.go`** — new `ActionProposePullRequest ActionKind = "ProposePullRequest"` for proposals that carry a forge PR URL instead of a cluster-side mutation. The cluster itself is NOT changed when the proposal is approved; only when the PR is merged + the next normal deploy runs.
- **`AIProposedAction.PullRequestURL string`** — new field holding the forge URL the proposer already opened (the digest-pin proposer in slice 3b will populate it via the srenix-enterprise forge client).
- **`Validate()` rules for the new kind** —
  - `PullRequestURL` non-empty (rejects with `ErrPullRequestURLEmpty`)
  - URL must parse as a well-formed HTTPS URL with a non-empty host (`ErrPullRequestURLInvalid`) — guards against an `http://` downgrade or a `https:///path` malformed link rendering as a phishing target in Slack
  - `Target.Namespace` still subject to the protected-NS check — Srenix never proposes PRs that would mutate `kube-system` / `vault` / `cnpg-system` infra
  - `Rollback.Description` still required (PR rollback = "close PR + delete branch")
  - Tier still must be `T1`/`T2`/`T3` (T0 = narration-only)
  - `ManifestYAML` and `PatchPayload` MUST be empty on a `ProposePullRequest` (else `ErrInvalidActionKind` — proposer can't smuggle a cluster mutation through this kind)
- **Self-hosted forges supported** — no OSS-side host allowlist; operators run self-hosted GitLab / Gitea / Forgejo with arbitrary hostnames. Allowlist enforcement (if needed for a specific deployment) belongs in the approval-server's per-CR policy layer in a future slice.
- **12 test cases** (`pkg/ai/propose_pull_request_test.go`) — happy path, empty/whitespace URL, http downgrade, missing host, garbled URL, self-hosted-GitLab accepted, protected-namespace rejection, missing rollback, wrong-kind URL field, T0-tier rejection, ManifestYAML-on-PR-kind rejection.
- Not yet wired into an executor — `pkg/ai` types only. srenix-enterprise slice 3b/3c lands the approval-server executor handler (Approve → post-merge comment / auto-merge per CR policy; Deny → close PR + record outcome to RAG) plus the `DigestPinProposer` that emits proposals of this kind.

### Added — Release-source detection (`pkg/releasesrc`, Phase 2d-γ-3 slice 1)

- **`pkg/releasesrc`** — new public package finds the file + line in a release-source repo where a workload's image tag is declared. Keystone for the paid-tier digest-pin proposer: without knowing which `values.yaml` line holds `image.tag`, the proposer can't construct a one-line patch.
- **`DetectInHelmValues(ctx, files, chartName, expectRepository) → *ImageRef`** — probes `charts/<chartName>/values.yaml` (umbrella layout) then `values.yaml` (single-chart root). Decodes the conventional `image: {repository, tag}` shape via `sigs.k8s.io/yaml`, requires `image.repository` to match `expectRepository` (guards against false matches in umbrella charts that ship multiple subchart blocks). Returns `ErrNotFound` cleanly when nothing matches; transport errors propagate unchanged.
- **`ImageRef{File, Line, KeyPath, CurrentTag, Repository}`** — `Line` is 1-based for editor/`git blame` parity. `KeyPath` is a dot-separated YAML walk (today: always `"image.tag"`). Line lookup uses a regex anchor (`image:` header → first `tag:` line below it) because `sigs.k8s.io/yaml` doesn't preserve positions.
- **`RepoFiles` interface** — minimal `Get(path) → bytes` + `List(patterns) → []string`. Defined in OSS so srenix-enterprise's forge client can implement it via a per-`(owner, repo, ref)` adapter without OSS taking a forge dependency.
- **Security defenses** — chart-name input is sanitized to `path.Base()` so a hostile `"../../etc"` chart name can't escape the chart dir. Empty `expectRepository` is rejected (would match every image block). Garbled YAML in one candidate file doesn't abort the probe — falls through to the next path.
- **13 test cases** — happy-path umbrella + happy-path root layouts, repo-mismatch silently skipped, missing tag returns NotFound, garbled YAML doesn't crash, all-paths-missing returns NotFound, true transport error propagates, nil-files / empty-expectRepository guards, empty chart name falls back to root only, path-traversal sanitization, exact line-number calculation, unquoted-numeric-tag handling.
- **Not yet wired** into any proposer — pure library slice. Foundation for **slice 2** (srenix-enterprise Forge → RepoFiles adapter + DigestPinProposer that consumes the v1.16.0+ workload feeder's `kind=workload` entries + this detector → forge.CreatePullRequest → Slack Approve/Deny buttons on every digest-pin warning). Argo CD Application + Kustomize + Flux HelmRelease detectors will join in follow-up slices.

### Added — Workload feeder (Phase 2d-γ-2, RAG foundation slice)

- **`internal/feeder/workload.go`** — new `WorkloadFeeder` walks Deployments / StatefulSets / DaemonSets each cycle and upserts one `rag.Entry{Kind: KindWorkload}` per workload. Features captured: `kind` (controller type), `namespace`, `name`, `replicas`, `containers: [{name, image, image_digest}, ...]`, and best-effort `owner_kind`/`owner_release`/`owner_release_namespace`/`owner_chart` derived from the conventional Helm + Argo CD annotations.
- **Digest resolution** — `image_digest` is read from the owning Pod's `status.containerStatuses[].imageID` (kubelet writes the resolved `sha256:` after a successful pull). Pods that haven't pulled yet (ImagePullBackOff, pending) contribute nothing, and that container's `image_digest` is simply omitted — the correct signal for a downstream proposer to skip the cycle and retry next time. Extraction tolerates `docker.io/`, `docker-pullable://`, and private-registry imageID formats.
- **Owner detection** — reads `meta.helm.sh/release-name` + `meta.helm.sh/release-namespace` for Helm-managed workloads; `argocd.argoproj.io/instance` for Argo CD (`<namespace>_<name>` form). The `helm.sh/chart` label is parsed to extract the chart name with the trailing version stripped. Empty when neither annotation is set — the proposer slice will fall back to a PR-template path.
- **System-namespace skip list** — `kube-system`, `kube-public`, `kube-node-lease`, `cnpg-system`, `rook-ceph`, `vault`, `external-secrets`, `calico-system`, `tigera-operator`, `calico-apiserver`, `local-path-storage`. Matches the digest-pin analyzer's system-namespace set so feeder and analyzer agree on "is this workload SRE-relevant".
- **Fail-open everywhere** — nil receiver / missing Source / missing Writer errors at the contract boundary; per-workload parse + Upsert failures are silently skipped so one bad workload can't stall the sweep. Mirrors the srenix-enterprise `CloudflareFeeder` discipline.
- **13 test cases** — happy path with digest, no-pod-no-digest, three-controller-kinds sweep, system-namespace skip, Helm annotations populate owner, Argo CD annotation parses `<ns>_<name>` form, no-annotations omits owner, multi-container with partial digests, degenerate empty workload skipped, writer error doesn't abort sweep, digest extraction across 5 imageID formats, default importance fallback, nil-guards table.
- **Not yet wired** into `srenix watch` — pure library slice. Next slice activates it via `cfg.RAGWriter rag.Writer` + a `--workload-feeder` flag on cmd/srenix + an operator `spec.feeder.workload.enabled` knob on the CR. Foundation for Phase 2d-γ-3 (release-source detection enrichment) and Phase 2d-γ-4 (digest-pin proposer that consumes these entries).

### Added — Watcher mints approve/deny URLs directly (Path B)

- **`pkg/ai/manifest_bridge.go`** — new public `ManifestBridge` (implements `FixProposer`) that converts `Diagnostic.ProposedPolicyYAML` into a signed `ApplyManifest` `AIProposedAction` via the existing safe-apply validator (closed Kind whitelist + per-Kind shape; NetworkPolicy is the v1.15.0 entry). Refusal classes — egress in `policyTypes`, unsupported Kind, protected namespace, non-yaml — quietly return `nil` (no URL minted on dangerous YAML).
- **`pkg/ai/signer.go`** — Ed25519 signer ported from srenix-enterprise (was proprietary, now Apache-2.0). Disk-backed (base64 raw bytes), trailing-whitespace tolerant, env-var fallback (`SRENIX_SIGNING_KEY_PATH`), `ErrSigningKeyMissing` sentinel for graceful fall-through. `GenerateAndPersistSigningKey()` for bootstrap.
- **`cmd/srenix/main.go`** — `srenix watch` gains `--approval-server-url` + `--signing-key-path` flags. When both resolve, loads signer + registers `ManifestBridge` as fallback `FixProposer` (only when registry has no programmatically-registered proposer — keeps srenix-enterprise's LLM-backed proposer primary). Wires `Config.ApprovalBaseURL` so `enrichDiagnostics` mints URLs in the existing T1 path.
- **`internal/operator/builders.go`** — `BuildWatcherDeployment` passes the new flags + mounts the signing-key Secret when both `cr.Spec.AI.ApprovalServerURL` AND `cr.Spec.Approval.SigningKey.SecretName` are set. Guards against half-configured installs (no key → no flags → no broken pod).
- **Closes the architectural gap** where ProposedPolicyYAML-bearing diagnostics (NetworkPolicyProposer) had URLs minted in the srenix-enterprise aiwatch process but NEVER reached the user-facing Slack / Alertmanager / OpenProject surfaces — those are written by the OSS watcher, which had no URL-minting capability. After this change the OSS watcher mints URLs itself; they flow through the existing `d.ApprovalURL` field every adapter already renders.
- **22 new test cases**: `pkg/ai/manifest_bridge_test.go` (10 — happy path, FixProposer compliance, empty-YAML → nil, refusal classes, missing-metadata variants), `pkg/ai/signer_test.go` (10 — construction, sign happy path, validation errors, key load round-trip with trailing whitespace / missing / bad / wrong-size, env-var fallback, generate-and-persist), `internal/operator/builders_test.go` (2 — watcher wires flags + volume when both spec fields set; watcher omits flags when only ApprovalServerURL set without signing key).
- **Backward compatible**: watcher built from a CR without `ai.approvalServerUrl` stays byte-identical to v1.15.0; new flags default empty; existing scripts/manifests unaffected.

### Added — Cloud Monitoring wiring, P4/G9

- **GCP Cloud SQL disk utilization** — `internal/cloud/gcp`: new `monitoringQuerier` interface + `cloudMonitoringQuerier` impl backed by `google.golang.org/api/monitoring/v3`. Queries `cloudsql.googleapis.com/database/disk/utilization` (ALIGN_MEAN over a 5-min window). `LiveClient.ListCloudSQLInstances` now populates `DiskUsedPercent` from the querier, falling back to `-1` "not measured" on failure. Non-fatal `monitoring.NewService` errors keep install working on partial credential grants.
- **Azure SQL DB storage_percent** — `internal/cloud/azure`: same shape; `monitoringQuerier` interface + `azureMonitoringQuerier` impl backed by `github.com/Azure/azure-sdk-for-go/sdk/monitor/azquery`. Queries the `storage_percent` metric (Average over 5-min window) for each `Microsoft.Sql/servers/databases` ARM ID. `LiveClient.ListSQLDatabases` populates `UsedPercent` from the querier; `-1` fallback preserved. Non-fatal `azquery.NewMetricsClient` errors keep install working when the SP lacks the Monitoring Reader role.
- Both impls use a small `metricsClient`/equivalent interface so unit tests stub the SDK without spinning up its transport. Parsing functions (`latestDiskUsedPercent`, `latestStoragePercent`) are pure + table-tested (nil / empty / multi-point / rounding / over-100 cap / defensive negatives).
- Pattern (interface + injectable querier + soft-fail + pure parser) is now the template for the remaining "not measured" signals: GCP `AvailableIPCount`, Azure IP-pool / AppGW backend health.

### Added — Azure App Gateway backend health via BackendHealth LRO (P4/G9 slice 3)

- `internal/cloud/azure`: new `backendHealthClient` interface + `liveBackendHealthClient` impl that wraps `ApplicationGatewaysClient.BeginBackendHealth` + `PollUntilDone` with a 60s poll cap (so a single misbehaving gateway can't stretch a probe cycle). Pure `aggregateBackendHealth` flattens the LRO response into per-pool `{Healthy, Total}` counts; **dedup**s the same backend address across HTTP settings (preferring the strongest Health observation: Up > Partial > Draining > Down > Unknown) and counts `Up`/`Partial` as healthy. `ListAppGatewayBackends` now populates `HealthyCount` from the aggregated result instead of leaving the `-1` "not measured" sentinel; failures keep the `-1` fallback so the probe skips the check.
- Different shape from the Monitor-API slices (LRO, not time-series) but the same overall pattern: injectable interface for testability + production impl + soft-fail.

### Added — Azure subnet IP-pool free count (P4/G9 slice 4)

- `internal/cloud/azure`: `ListSubnets` now computes `AvailableIPCount = TotalIPCount - used`, summing every IP-consuming resource type attached to the subnet (`IPConfigurations` NIC refs, `ApplicationGatewayIPConfigurations`, `IPConfigurationProfiles`, `PrivateEndpoints`). These fields are READ-ONLY on the Subnet resource and populated by the apiserver automatically — no `$expand` needed, no extra API call. The IP-exhaustion probe now fires on real data instead of skipping. Pure `subnetUsedIPCount` helper is fully unit-tested.

### Added — typed `AISpec` on `AgenticSRE` v1alpha1 (operator Phase-2 schema slice)

- `api/v1alpha1`: new `AISpec` + `AIAPIKeySpec` + `AIT3Spec` + `AIMemorySpec` + `AIMemoryStorageSpec` + `AIEmbeddingsSpec` types mirroring the chart's `ai.*` helm values surface. Exposed as `AgenticSRESpec.AI` (optional). DeepCopy methods hand-written matching the Phase-1 pattern.
- CRD YAML extended to accept `spec.ai` so the apiserver validates the schema. Tier uses `kubebuilder:validation:Enum=off;t0;t1;t2;t3`.
- **Controller does NOT yet consume these fields.** The reconciler still relies on the chart's `ai.*` helm values for the running aiwatch + approval-server. Schema lands now so operator-managed manifests are forward-compatible with the Phase 2 reconciler wiring; the misleading comment that previously claimed the fields were "opaque pass-throughs" is corrected.

### Fixed

- Stale package docs in `pkg/cloud/gcp/client.go` and `pkg/cloud/azure/client.go` that claimed "Live wrapper deferred to a follow-up PR" — both Live wrappers shipped (v1.7 baseline; v1.9 Cloud Monitoring / Azure Monitor / BackendHealth LRO additions in PRs #103–#106). Comments now reflect what's on `main`.

### Added — Operator Phase 1c (slice B) — OLM bundle scaffolding

- New `bundle/` directory and `bundle.Dockerfile` carrying the OLM ClusterServiceVersion + the four shipped CRDs (`AgenticSRE`, `Silence`, `ResolutionRecord`, `DriftReport`). The CSV's `install.spec` mirrors `templates/operator-deployment.yaml` (image, args, env, ports, probes, securityContext) so `operator-sdk run bundle <image>` produces structurally the same operator as `helm install`.
- `installModes`: `OwnNamespace` + `SingleNamespace` + `AllNamespaces` (true); `MultiNamespace` (false) — explicit because the reconciler scope decisions in Phase 1b watch all-namespaces.
- New parity-guard tests in `internal/operator/bundle_parity_test.go`: (1) CSV is valid YAML + kind=ClusterServiceVersion; (2) every CRD shipped in `bundle/manifests/` is declared under CSV `customresourcedefinitions.owned` (no orphan CRDs); (3) the chart's operator ClusterRole rules and the CSV's `clusterPermissions[0].rules` carry the same `(apiGroup, resource)` set (catches the common drift pattern where someone adds a rule to one file and forgets the other).
- **NOT in this slice**: CI bundle-smoke job using `operator-sdk run bundle` against kind — Phase 1c slice C, separate PR.

### Added — Operator Phase 1c (slice A) — operator-provisioned reader RBAC

- `api/v1alpha1`: new `ConditionReaderRBACReady` condition + `FinalizerOperatorRBAC` (`srenix.ai/operator-rbac`) — cluster-scoped resources can't carry namespaced ownerRefs, so the finalizer drives cleanup.
- `internal/operator/rbac_builders.go`: `BuildReaderClusterRole()` returns a shared cluster-scoped role mirroring `templates/clusterrole-reader.yaml`'s verb set. `BuildReaderClusterRoleBinding(cr)` returns a per-CR binding labeled `managed-by-cr` + `cr-namespace` for safe identification by the finalizer.
- `internal/operator/reconciler.go`: adds `reconcileReaderRBAC()` (idempotent CreateOrUpdate on the shared role + per-CR binding), `finalizeReaderRBAC()` (deletes ONLY bindings we labeled; defense-in-depth against name collisions), and finalizer add/remove paths on every reconcile. `ReaderRBACReady` is computed from observed state: ClusterRole present + binding present + subject targets the CR's SA. `Ready` is now `(no reconcile error) AND ReaderRBACReady`; `WatcherRunning` continues to track availability orthogonally.
- Chart: operator ClusterRole extended with cluster-wide CRUD on `rbac.authorization.k8s.io/{clusterroles,clusterrolebindings}`.
- 6 new reconciler tests with the controller-runtime fake client: creates-both / finalizer-add / on-delete-removes-binding-and-finalizer / shared-ClusterRole-survives-CR-delete / defense-in-depth-skips-unlabeled-bindings / ReaderRBACReady-True / WrongSubject-detected-and-corrected.
- Coexistence contract: operator-managed binding lands ALONGSIDE the chart-managed binding (RBAC unions across bindings; same SA + same role from two managers is harmless). Operators can run both side-by-side; chart binding stays helm-owned until `helm uninstall`.
- **NOT in this slice**: OLM bundle (Phase 1c slice B) + CI bundle-smoke (Phase 1c slice C). Each is a separate PR per `docs/design/2026-05-v1.9-operator-phase-1c.md`.

### Added — `Silence` CRD + watch-loop suppression

- New `Silence` CRD (`silences.srenix.ai`, namespace-scoped, `sil` short name). Operators create a Silence to mute a known-benign-but-unfixable finding for a bounded window. Matcher fields: `source` / `subject` / `severity` (empty = wildcard); CRD validation rejects an entirely-empty matcher. `spec.until` is required; past expiry the Silence becomes a no-op but is NOT auto-deleted (audit trail). Optional `reason` + `createdBy` for "why is this muted?" answers.
- New pure `pkg/silence.Filter()` + `Matches()` — namespaced lookup, exact-field matching, expired-silence-never-matches guard. Order-preserving, doesn't mutate the caller's slice. 8 unit tests.
- New `pkg/silence.K8sLister` (dynamic-client backed) lists active Silences cluster-wide once per watcher cycle. Soft-fails on a missing CRD (returns nil, nil) so a chart < 1.9 install still works.
- Watcher integration: `Watcher.WithSilenceLister(lister)` — wired in `cmd/srenix/main.go`. Silenced diagnostics are dropped in `runDiagnose()` BEFORE downstream emission (DriftReport / Slack / Alertmanager / ticketing), so a muted finding never re-pages.
- Chart: `templates/crd-silence.yaml` (gated on `silence.installCRD`, default ON) + `templates/clusterrole-silence.yaml` (cluster-wide list/watch on `silences`, gated on `silence.enabled`, default ON). Reserves `silences/status` write permission for a future matchCount/lastMatchAt updater.
- Closes post-v1.9 adversarial-review finding #2: previously Srenix had only endpoint-probe flake debounce — no user-controlled per-fingerprint, time-bounded suppression. Now Silence is a first-class, K8s-native concept matching the Alertmanager-silences pattern.

(Reserve for v1.9+ — operator Phase 1c per `docs/design/2026-05-v1.9-operator-phase-1c.md`; Phase 2 reconciler consumption of the AISpec; remaining cloud Monitoring-API signals; trigger-classes C/E.)

---

## [1.8.12] — 2026-05-30

Chart wiring for the approval-server HA backend introduced in Srenix Enterprise PR #16.

### Added

- `approval.store.backend=configmap` (and `.namespace`, `.replayConfigMap`, `.runbookConfigMap`): when set, passes the matching `--store-*` flags to `srenix-enterprise approval-server`, switches the Deployment to `RollingUpdate`, and provisions a per-namespace Role + RoleBinding granting the approval-server SA `get/update/create` on the named ConfigMaps (minimum-privilege: NOT granted in the default in-memory posture). With this set + `approval.replicas > 1`, the approval-server is HA-safe (a JTI used on replica A cannot be replayed on B; T3 dual-approval state cannot fork).

---

## [1.8.11] — 2026-05-30

Chart-only fix: the RAG Qdrant StatefulSet (added in 1.8.9) CrashLooped on first deploy because `securityContext.readOnlyRootFilesystem: true` made Qdrant's default snapshots/temp paths unwritable. Redirected both under the mounted storage PVC via `QDRANT__STORAGE__SNAPSHOTS_PATH` and `QDRANT__STORAGE__TEMP_PATH` env vars — single volume now serves all writes.

### Fixed

- RAG Qdrant snapshots + temp paths point inside the storage PVC (was: read-only root FS → CrashLoopBackOff `"Can't create Snapshots directory: ReadOnlyFilesystem"`).

---

## [1.8.10] — 2026-05-29

P2/G5c chart wiring — connects the deployed aiwatch to the RAG store.

### Added

- When `ai.memory.enabled`, `srenix.aiArgs` now passes `--memory-store-url` (defaults to the in-namespace Qdrant service `http://<release>-rag.<ns>.svc:6333`), `--memory-embeddings-endpoint` (defaults to `ai.endpoint`), `--memory-embeddings-model` (required), and `--memory-topk` to the aiwatch. With this, the commercial binary's RAG grounding (Srenix Enterprise G5c retrieve half) is reachable end-to-end; off by default.

---

## [1.8.9] — 2026-05-29

P1/G4 foundation for the AI-remediation memory loop. Chart-only effect (new CRD + RBAC); the recorder library is dormant until the AI write-path wires it (P2/G5c).

### Added — ResolutionRecord CRD + recorder

- **`ResolutionRecord` CRD** (`resolutionrecords.srenix.ai`, cluster-scoped, `rr` short name) — the append-only outcome log: one CR per applied+verified (or rejected/reverted) remediation, capturing `{fingerprint, source, subjectKind, diagnosticDigest, proposal{actionKind,target,rationale,rollback}, delivery, applied, verified}`. This is the durable system-of-record the dedicated RAG memory layer (1.8.8 Qdrant) embeds + retrieves.
- **`internal/resolution` recorder** — `Recorder.Record()` appends a CR through the snapshot.Mutator (nil-safe / no-op in dry-run); stable `Fingerprint(source, subject)` join key to DriftReport.
- **ResolutionRecord ClusterRole** — create/get/list/watch + status patch (for the RAG layer's `embeddedAt`/`vectorId` stamp), bound to the chart SA. Append-only (no delete verb).

---

## [1.8.8] — 2026-05-29

P2/G5a — the dedicated RAG vector store deployment (chart-only; off by default).

### Added — in-namespace Qdrant RAG

- **`ai.memory.enabled`** stands up a dedicated **Qdrant** vector store (StatefulSet + PVC + ClusterIP Service) in the install namespace, alongside the other Srenix objects. The aiwatch (P2/G5b–c, Srenix Enterprise) embeds ResolutionRecords via the in-cluster gpu-ai embeddings endpoint and retrieves prior resolutions to ground T1–T3 proposals. The ResolutionRecord CRD (1.8.9) is the system-of-record; Qdrant is the rebuildable semantic index over it.
- New `ai.memory.*` values: `image`, `storage.{size,className}`, `resources`, `embeddings.{endpoint,model}`, `storeUrl`, `topK`. Off by default and independent of `ai.enabled` so it can be rolled out separately.

---

## [1.8.6] — 2026-05-29

P0 signal-hygiene from the AI-remediation plan (`docs/design/` in Srenix Enterprise), plus the chart arg that activates commercial click-to-fix delivery.

### Fixed — false-positive criticals (alert-fatigue de-noising)

- **HPAScaling: `ScalingActive=False` / `reason=ScalingDisabled` is now Warning, not Critical.** That condition is the *expected* state when an HPA's target is intentionally at zero (KEDA scale-to-zero, or a Deployment scaled to 0) — the autoscaler simply goes dormant. Flagging it CRITICAL was a false positive that trained operators to ignore the board. Every other reason (`FailedGetScale`, `FailedGetResourceMetric`, quota/PDB blocks) stays CRITICAL.
- **Endpoint discovery skips `cm-acme-http-solver-*` Ingresses.** cert-manager spawns these transient HTTP-01 challenge solvers and deletes them on completion; probing them produced churning false-criticals and ticket spam for hosts that aren't real services.

### Added

- **`ai.approvalServerUrl`** chart value → `--approval-server-url` arg on the aiwatch (via `srenix.aiArgs`). When set (with `approval.enabled`), the commercial Srenix Enterprise binary emits signed one-click click-to-fix links for T1/T2 proposals.

---

## [1.8.5] — 2026-05-28

Chart-only fix found while enabling the paid tier on a live cluster.

### Fixed — approval-server keygen-job image tag

The `approval-server-keygen-job` (a Helm pre-install/upgrade hook that mints the Ed25519 signing key) still defaulted its image tag to `.Chart.AppVersion` (e.g. `1.8.2`), but srenix-enterprise images are tagged with a leading `v` (`v1.8.2`). On a fresh paid enable the keygen hook hit `ImagePullBackOff` and stalled the whole `helm upgrade` in `pending-upgrade`. Now uses the same `v<AppVersion>` default as the approval-server Deployment (fixed in 1.8.4). Without this, enabling `approval.enabled=true` required a manual `approval.image.tag` override.

---

## [1.8.4] — 2026-05-28

Corrects the AI-tier deployment model shipped in 1.8.3. No Go changes.

### Fixed — AI tier deploys as an additive companion, not an OSS-binary flag-swap

1.8.3 folded the `--ai-*` flags into the **OSS** watcher Deployment and diagnose CronJob args, on the assumption that the commercial binary is a flag-superset of OSS. It is not: `srenix-enterprise watch` / `srenix-enterprise diagnose` are the **AI-layered counterparts** with a deliberately reduced flag surface — they reject the OSS operational flags (`--live`, `--write-driftreports`, `--slack-*`, `--remedy`, `--ticketing-*`, `--cloud-*`). Enabling 1.8.3's wiring + the srenix-enterprise image would have crash-looped the watcher on an unknown flag. (1.8.3 was gated on `ai.enabled=true`, default false, so no OSS or default install was affected.)

The corrected, **purely additive** model:

- The OSS watcher Deployment + diagnose / remediate CronJobs are **never swapped or modified** — they keep the OSS image and own the event-driven probe → fix → Slack / ticketing / DriftReport pipeline.
- Setting `ai.enabled=true` stands up a **separate `aiwatch` Deployment** (new `templates/aiwatch-deployment.yaml`) running `srenix-enterprise watch` — the AI-layered counterpart that polls the same merged catalog on `ai.interval` and fires the AI tiers (t0→t3) against new diagnostics, signing click-to-fix URLs for the approval-server at t1+.
- `srenix.aiArgs` now emits exactly the `srenix-enterprise watch` flag surface (incl. `--interval`); `srenix.aiImage` resolves the commercial image (`docker4zerocool/srenix-enterprise:v<AppVersion>`).
- New `ai.image.*`, `ai.interval`, `ai.resources` values. The aiwatch pod reuses the chart's read-only reader SA (it only reads + proposes; never mutates).
- Fixed the approval-server image-tag default to the `v`-prefixed srenix-enterprise convention (`v<AppVersion>`), which previously resolved to a non-existent tag.

**Go-to-market:** OSS install + the single flag `ai.enabled=true` = the paid tier. No image-swap-and-pray. Full setup in `docs/DEPLOYMENT.md`.

---

## [1.8.3] — 2026-05-28

Chart-only release that completes the AI-tier (commercial Srenix Enterprise) deployment path. No Go changes.

### Added — AI-tier flag wiring in the chart

The chart now renders the commercial `--ai-*` flag surface onto the watcher Deployment and diagnose CronJob when `ai.enabled=true`, via two new nil-safe helpers (`srenix.aiArgs`, `srenix.aiEnv`). Previously the `ai:` values block existed but was consumed by no template, so the paid tier could not actually be turned on through Helm.

- **`srenix.aiArgs`** emits `--ai-tier`, `--ai-endpoint`, `--ai-model`, `--ai-api-key-header`, `--ai-allow-saas`, `--ai-llm-fixer-matcher`, `--ai-audit-log`, and (for t3) repeatable `--t3-vault-allowed-prefix`. `ai.endpoint` and `ai.model` are `required` when enabled. The OSS `srenix` binary does not understand these flags, so the block is inert unless `image.repository` points at `docker4zerocool/srenix-enterprise`.
- **`srenix.aiEnv`** injects the LLM bearer token into the env var the binary reads (`ai.apiKey.envName`, default `AI_API_KEY`) via `secretKeyRef` — never inlined, ESO-managed.
- New `ai:` keys: `model`, `llmFixerMatcher`, `auditLog`, `apiKey.{envName,header}`, and `t3.vaultAllowedPrefixes` (gates the Vault blast radius the t3 runbook proposer may reference).
- Approval-server templates (deployment / service / ingress / rbac / keygen-job) were already present; this release makes the watcher/diagnose side consumable.

---

## [1.8.2] — 2026-05-28

Hardening release from a post-v1.8 adversarial review. Corrects honesty/correctness defects found in the cloud Live wrappers and the operator, and closes one roadmap acceptance-criteria gap. No new probes; behavior changes are confined to live cloud mode and the opt-in operator.

### Fixed — cloud Live wrappers no longer silently pass un-measurable signals

The GCP/Azure Live wrappers previously populated placeholder values for metrics the SDK list-calls don't expose, which made several probe checks **silently never fire in live mode** (green unit tests masked it because they inject values the live wrapper can't produce). Each is now reported as **"not measured"** via a `-1` sentinel, and the probe **skips** that specific check (surfacing the gap in the component Detail) instead of evaluating it as healthy:

- **GCP Subnets** — IP-exhaustion check (`AvailableIPCount` was set to the total → always 100% free).
- **Azure Subnets** — same IP-exhaustion check.
- **Azure App Gateway backends** — backend-health check (`HealthyCount` was set to the total).
- **GCP Cloud SQL / Azure SQL** — storage-utilization check (`DiskUsedPercent`/`UsedPercent` was never populated → treated as 0%).

These require the cloud Monitoring API / a long-running BackendHealth operation and are wired for real in v1.9. AWS already fetches all of these for real and is unaffected. (Azure SQL automated PITR backup is a genuine platform invariant, not a placeholder — comment clarified, behavior unchanged.)

### Fixed — operator BYO-ServiceAccount no longer adopts a pre-existing SA

When a `AgenticSRE` CR pins `spec.serviceAccountName` (the supported path for giving an operator-managed watcher the probe RBAC it needs — point it at the chart's reader-bound SA), the reconciler used to still create+own that SA, grafting an owner-ref onto a pre-existing object and garbage-collecting it on CR deletion. The reconciler now skips SA creation entirely when `spec.serviceAccountName` is set.

**Known limitation (tracked for Phase 1c):** the operator does not yet provision its own reader `ClusterRoleBinding`, so an operator-managed watcher gets probe RBAC **only** via the BYO-SA path above. Documented on the CRD field and in the operator design doc.

### Added — M2 probe-class Helm toggles (roadmap AC parity)

`probes.{kong,hpaScaling,argocdApp,velero}.enabled` now exist in `values.yaml` and emit `SRENIX_PROBE_*=off` via the new `srenix.probeToggleEnv` helper (mirrors `srenix.analyzerToggleEnv`). Closes the v1.8 acceptance criterion that promised per-probe Helm values; the probes were previously gated only by env opt-out + CRD auto-skip. All default ON (auto-skip when the CRD is absent).

### Changed

- Cleared stale "not shipped yet" / "M1 follow-up" / "Azure remains a stub" comments in `values.yaml`, `cmd/srenix/main.go`, and `catalog/cloud.go` — all three cloud providers and the M2 probe set shipped in v1.8.

---

## [1.8.1] — 2026-05-28

Patch release fixing two issues found while deploying v1.8.0 to a live cluster. Both are chart-only; no Go changes.

### Fixed

- **Diagnose / remediate `activeDeadlineSeconds` raised 120 → 300.** The v1.8 analyzer + M2-probe set adds a meaningful number of cluster List calls (CRDs, HPAs, all namespaces + pods + NetworkPolicies, Kong / Velero / Argo CRDs). A live diagnose on a busy cluster (~80 KongPlugins, ~250 drift records) was measured at ~157s — past the old 120s cap, so the CronJob was killed mid-run with `DeadlineExceeded`. 300s gives headroom while still failing fast on a genuinely hung cluster API.
- **Operator templates made nil-safe for `--reuse-values` upgrades.** `operator-deployment.yaml` (`.Values.operator.enabled`) and `crd-agenticsre.yaml` (`.Values.operator.installCRD`) dereferenced `.Values.operator` directly, so a `helm upgrade --reuse-values` from a pre-v1.8 install (whose stored values predate the `operator:` block) hit `nil pointer evaluating interface {}.enabled`. Now guarded with `(.Values.operator).enabled` / `(.Values.operator).installCRD`, matching the existing `(.Values.analyzers).*` pattern.

### Verified on live cluster

v1.8.0 deployed to the dev cluster (helm rev 23) and a live diagnose confirmed the new probes fire against real resources: **Kong** (80 KongPlugins inspected), **HPAScaling** (flagged 3 real scaling-disabled HPAs), **ArgoCD-Application**, **Velero**, and **SecurityDrift** (PSS / image-pin / NetworkPolicy gaps in the `kong` namespace). 255 DriftReports reconciled.

---

## [1.8.0] — 2026-05-28

Drift-class completion + operator port + full multi-cloud release. Closes the bulk of `docs/design/2026-05-v1.8-roadmap.md`: the remaining drift classes (config / capacity / security), the controller-runtime operator port (Phase 1 + 1b), the M2 K8s probe slice (Kong / HPA / ArgoCD / Velero), and a complete 30-probe multi-cloud surface (10 each AWS / GCP / Azure) with all three Live SDK wrappers wired so the probes execute against real clouds.

### Added — Azure cloud-probe Live SDK wrapper (probes now execute against real Azure) — all 3 clouds live

- **`internal/cloud/azure/live.go`** — `LiveClient` implements all 10 `pkgazure.Client` methods against `azure-sdk-for-go` (armsql, armcompute, armcontainerservice, armmsi, armnetwork, armappservice, armstorage, armkeyvault, armauthorization). Auth via `DefaultAzureCredential` (AAD Workload Identity in-cluster, `az login` locally). Read-only. Resolves server→database, vnet→subnet, and cluster→nodepool nesting; extracts resource group from ARM IDs; counts role assignments per managed-identity principal.
- **`cmd/srenix buildCloudSource()`** — `--cloud-azure-enabled` now constructs the live client (requires `--cloud-azure-subscription-id`; optional `--cloud-azure-location`) instead of erroring. **With this, all three providers (AWS, GCP, Azure) execute against real clouds.**
- Two documented limitations populated conservatively (no false-positives): VNet subnet free-IP (Network API exposes none → CIDR-derived total, available=total) and App Gateway backend health (per-gateway LRO too heavy per cycle → reports pool size as healthy). Both have Monitoring/LRO follow-ups noted in code.
- **Verification boundary:** compiles cleanly against the real `azure-sdk-for-go` ARM modules (API-surface correctness), but **not** integration-tested against a live Azure subscription — needs credentials. Probe evaluation logic remains unit-tested against fakes.

### Added — GCP cloud-probe Live SDK wrapper (probes now execute against real GCP)

- **`internal/cloud/gcp/live.go`** — `LiveClient` implements all 10 `pkggcp.Client` methods against `google.golang.org/api` (sqladmin, compute, container, iam, cloudkms, storage). Auth via Application Default Credentials (GKE Workload Identity in-cluster). Read-only. Compiles against the real SDK surface.
- **`cmd/srenix buildCloudSource()`** — `--cloud-gcp-enabled` now constructs the live client (requires `--cloud-gcp-project`; optional `--cloud-gcp-region`) instead of erroring. The GCP probes are no longer dormant — they run against a real project when enabled.
- Two documented SDK limitations populated conservatively so probes never false-positive: VPC subnet free-IP count (Compute API exposes no free-IP field → reports fully-free pending a Monitoring-API follow-up) and per-backend LB health (aggregated via `BackendServices.GetHealth`).
- **Verification boundary:** the wrapper compiles cleanly against `google.golang.org/api v0.282.0` (proves API-surface correctness) but is **not** integration-tested against a live GCP project — that needs credentials. Probe evaluation logic remains unit-tested against fakes.
- Azure Live wrapper follows in the next PR; until then `--cloud-azure-enabled` still errors.

### Added — Workstream B4 (config drift)

- **`ConfigDrift`** analyzer (B4) — three signals the basic resource-health probes miss:
  - **CRD multi-storedVersions** — `apiextensions.k8s.io/v1` CRDs whose `status.storedVersions` lists more than one apiserver storage version. Storage migration is pending; future drops of the old version will fail. Critical.
  - **Deployment rollout stuck** — `metadata.generation` ahead of `status.observedGeneration` past the grace window (the controller hasn't reconciled the latest spec; critical), or `status.updatedReplicas` < `spec.replicas` past the grace window (rollout stuck mid-flight; warning if some replicas still available, critical if zero). Default 15-minute grace.
  - **`checksum/config` annotation drift** — Pods of the same Deployment carrying disagreeing `checksum/config` annotation values, indicating a rolling update from the last config change didn't propagate to all replicas. Warning. Skipped on workloads that don't carry the annotation.
- Walks via owner-reference chain Pod → ReplicaSet → Deployment.
- Skips system namespaces (kube-system, kube-public, kube-node-lease).
- Reader ClusterRole extended with read on `apiextensions.k8s.io/customresourcedefinitions`.
- Default ON; flip `analyzers.configDrift.enabled=false` to disable, or set `SRENIX_ANALYZER_CONFIG_DRIFT=off`. 16 unit tests.

### Added — Operator port Phase 1 (foundations)

- **`api/v1alpha1/`** — `AgenticSRE` CRD types (Spec, Status, Conditions) with hand-written DeepCopy methods. Foundations only; the controller-runtime Reconcile loop, the manager binary, and the chart wiring for operator-managed installs all come in Phase 1b. See `docs/design/2026-05-v1.8-operator-phase-1.md` for the staged-release rationale.
- **`internal/operator/builders.go`** — pure-function builders that translate `AgenticSRESpec` → `*appsv1.Deployment` (watcher) and `*batchv1.CronJob` (diagnose, remediate). Mirror the existing chart's CLI argument format so an operator-managed install behaves identically to a Helm-managed install. 19 unit tests cover defaults, overrides, image-policy inference, pull-secret round-trip, and alerting-flag emission.
- **`charts/.../templates/crd-agenticsre.yaml`** — CRD shipped via the chart, gated behind `operator.installCRD` (default `true`). Installing the CRD on a cluster without the operator binary is harmless: the resource is queryable state with no controller acting on it.

### Added — Workstream B5 (capacity drift)

- **`CapacityDrift`** analyzer (B5) — capacity-tier signals that the basic resource-health probes miss. Five signals across HPAs and PVCs, none requiring metrics-server (the metrics-dependent signals — pod request vs usage, PVC growth-trajectory — defer to a v1.8.x follow-up):
  - **HPA pinned at maxReplicas** — `status.currentReplicas == spec.maxReplicas` past the saturation grace (24h default), excluding `min==max` static configurations. Workload is chronically under-provisioned. Critical.
  - **HPA pinned at minReplicas, not load-driven** — current replicas held at `minReplicas` for > 30 days with `maxReplicas > minReplicas + 1`. HPA range is decorative; the workload could be a static Deployment. Warning.
  - **HPA AbleToScale=False** — `status.conditions[type=AbleToScale,status=False]` past grace (15-min default). Typically a ResourceQuota cap or PDB blocking the controller. Critical.
  - **HPA FailedGetResourceMetric** — `ScalingActive=False` with that reason. Metrics-server is missing or unreachable; the HPA can't decide. Warning. This is the v1.8 R1 risk-mitigation signal so operators notice without us depending on metrics-server.
  - **PVC volume-expansion stuck** — `FileSystemResizePending=True` past grace, OR `spec.resources.requests.storage > status.capacity.storage` past grace. Volume-expansion got requested but the CSI driver didn't complete it. Critical.
- Skips kube-system / kube-public / kube-node-lease.
- Reader ClusterRole extended with read on `autoscaling/horizontalpodautoscalers`; PVC reads already covered by the core probe rule.
- Default ON; flip `analyzers.capacityDrift.enabled=false` to disable, or set `SRENIX_ANALYZER_CAPACITY_DRIFT=off`. 17 unit tests.

### Added — Workstream B6 (security drift)

- **`SecurityDrift`** analyzer (B6) — three observational signals on security posture:
  - **PSS posture gap** — user namespaces with no `pod-security.kubernetes.io/enforce` label (admission applies the cluster-wide default, typically `privileged`), or with `enforce=privileged` explicitly (the most-permissive PSS profile). Warning.
  - **Mutable image tag** — Pods whose containers reference images by tag only (`<image>:<tag>`) rather than by digest (`<image>@sha256:<digest>`). Mutable tags break the image-attestation signature chain — the runtime image can be re-published behind the same tag. Warning. Skipped for `:latest` (other code paths already flag that).
  - **NetworkPolicy coverage gap** — user namespaces running pods with zero NetworkPolicies. K8s default networking is fully permissive without at least one policy. Warning per namespace.
- Skips kube-system / kube-public / kube-node-lease / cnpg-system / rook-ceph / vault / external-secrets — system namespaces whose security posture is controller-managed.
- Reader ClusterRole extended with `networking.k8s.io/networkpolicies`; namespaces already covered by the core probe rule.
- Default ON; flip `analyzers.securityDrift.enabled=false` to disable, or set `SRENIX_ANALYZER_SECURITY_DRIFT=off`. 16 unit tests.
- Out of scope for v1.8 (deferred to a v1.8.x follow-up): true PSS-downgrade detection (requires label history) and active Cosign / Notation signature verification (admission-time concern; Srenix is observational).

### Added — Operator port Phase 1b (controller-runtime + Reconciler + manager binary)

- **`sigs.k8s.io/controller-runtime v0.24.1`** added — chosen for compatibility with the current `k8s.io v0.36` baseline (controller-runtime v0.21 had a `ResourceEventHandlerRegistration` interface mismatch with newer client-go).
- **`internal/operator/reconciler.go`** — `Reconciler` implementation. Reconcile flow: fetch CR → validate `spec.image.tag` → reconcile ServiceAccount + watcher Deployment + diagnose CronJob + remediate CronJob via createOrUpdate (delete-on-disable) → compute `Ready` and `WatcherRunning` conditions from observed Deployment state → patch status. Uses controller-runtime CreateOrUpdate rather than server-side-apply to keep the cutover boring (existing chart installs are not disturbed unless an operator explicitly creates a `AgenticSRE` CR).
- **`cmd/srenix-operator/main.go`** — manager binary: leader-election lease (`srenix-operator.srenix.ai`, namespace from downward-API `MY_POD_NAMESPACE`), `:8080` Prometheus metrics, `:8081` healthz/readyz probes, structured zap logging.
- **`api/v1alpha1/groupversion_info.go`** — `AddToScheme` wired via `runtime.NewSchemeBuilder` directly (sidesteps the deprecated `controller-runtime/pkg/scheme.Builder`).
- **`charts/.../templates/operator-deployment.yaml`** — operator Deployment + ServiceAccount + ClusterRole + ClusterRoleBinding. Gated behind `operator.enabled` (default `false`). Operator has the read+write+delete verbs on ServiceAccount / Deployment / CronJob in any namespace; status-subresource write on the CR; Lease verbs for leader-election; events create+patch for `kubectl describe`. SecurityContext: `runAsNonRoot`, `readOnlyRootFilesystem`, drops all capabilities.
- **`Dockerfile`** — second `go build` step compiles `/srenix-operator` alongside `/usr/local/bin/srenix`. Single image hosts both binaries; the operator Deployment overrides `command:` to invoke `/srenix-operator` instead of the watcher.
- **11 reconciler unit tests** using the controller-runtime fake client — covers create-all-subresources, owner-ref attachment, condition computation, watcher disabled (no-create + delete-on-disable), validation short-circuit (empty image tag), update-existing-deployment, remediate flow, ServiceAccountName override, post-delete reconcile silence.

### Added — GCP cloud probes (Sprint 1 slice)

First two probes of the M2 GCP slice from `docs/design/2026-05-cloud-probe-framework.md`. The remaining 8 probes (GKE control plane, GKE nodepool, IAM SA, LB backend, Google-managed certs, GCS public-access, KMS, subnet capacity) ship on follow-up PRs against `feat/gcp-cloud-probes`.

- **`pkg/cloud/gcp/`** — `Client` interface fleshed out from scaffold (now has `Project()`, `Region()`, `ListCloudSQLInstances()`, `ListPersistentDisks()`). `types.go` adds narrow projections of `CloudSQLInstance` + `PersistentDisk` so probes don't depend on `cloud.google.com/go/...` directly.
- **`internal/cloud/gcp/CloudSQL`** — reports drift on Cloud SQL instances: non-RUNNABLE state (FAILED/SUSPENDED critical; transitional warning), disk usage ≥ 80%/90% (warn/critical; suppressed when `StorageAutoResize=true`), missing automated backups (warning). Subject convention: `gcp-cloudsql/<project>/<instance>`.
- **`internal/cloud/gcp/PersistentDisks`** — reports drift on Persistent Disks: FAILED state (critical), transitional state (CREATING/RESTORING/DELETING) past 1h grace (warning), detached-but-READY past 24h cleanup grace (warning — billing leak / orphaned PV). Subject convention: `gcp-pd/<project>/<zone-or-region>/<disk>`.
- **`catalog/cloud.go`** — `RegisterCloudOSS` now registers the GCP probes when `gcpEnabled=true` (parameter was previously unused).
- 21 unit tests via fake client (11 Cloud SQL + 10 Persistent Disks).

### Added — GCP cloud probes (Sprint 2 slice)

Four more GCP probes (6 of 10 now shipped). Remaining 4 (LB backend, Google-managed certs, GCS public-access, KMS) follow in Sprint 3.

- **`GKEControlPlane`** — flags the configured GKE cluster (env `CLOUD_GCP_GKE_CLUSTER`) when status is not RUNNING (ERROR/DEGRADED critical, transitional warning, not-found critical). Mirrors AWS `EKSControlPlane`. Subject `gcp-gke/<project>/<cluster>`.
- **`GKENodePools`** — flags node pools in ERROR / RUNNING_WITH_ERROR (critical) or other non-RUNNING state (warning) for the configured cluster. Mirrors AWS `EKSNodeGroups`. Subject `gcp-gke-nodepool/<project>/<cluster>/<pool>`.
- **`IAMServiceAccounts`** — posture drift: disabled SA still carrying user-managed keys (warning), > 2 user-managed keys (key sprawl; warning). Mirrors AWS `IAMRoles`. Subject `gcp-iam-sa/<project>/<email>`.
- **`Subnets`** — IP-exhaustion: < 10% free critical, < 25% free warning; zero-total subnets skipped (no div-by-zero). Mirrors AWS `VPCSubnets`. Subject `gcp-subnet/<project>/<region>/<name>`.
- `pkg/cloud/gcp` client + types extended; `catalog/cloud.go` registers all 6 GCP probes when `gcpEnabled=true`.
- 18 unit tests.

### Added — GCP cloud probes (Sprint 3 slice — 10/10 GCP parity)

Final 4 GCP probes — completes 10-probe parity with the AWS set.

- **`LoadBalancerBackends`** — 0 healthy backends critical, partial-unhealthy warning. Mirrors AWS `ALBTargetHealth`. Subject `gcp-lb/<project>/<name>`.
- **`ManagedCertificates`** — PROVISIONING_FAILED* / RENEWAL_FAILED critical, ACTIVE-but-< 21d-to-expiry warning. Mirrors AWS `ACMCertExpiry`. Subject `gcp-cert/<project>/<name>`.
- **`GCSPublicAccess`** — allUsers / allAuthenticatedUsers IAM binding critical, `publicAccessPrevention != enforced` warning. Mirrors AWS `S3BucketPublicAccess`. Subject `gcp-gcs/<project>/<bucket>`.
- **`KMSKeys`** — DESTROY_SCHEDULED / DESTROYED / *_FAILED critical, DISABLED warning, ENABLED-without-rotation warning. Mirrors AWS `KMSKeys`. Subject `gcp-kms/<project>/<key>`.
- `catalog/cloud.go` registers all 10 GCP probes when `gcpEnabled=true`.
- 18 unit tests.

### Added — Azure cloud probes (Sprint 1 slice)

First two probes of the M2 Azure slice. The remaining 8 probes (AKS control plane, AKS nodepool, Managed Identity, App Gateway backend, certs, Storage public-access, Key Vault, VNet/subnet) ship on follow-up PRs against `feat/azure-cloud-probes`.

- **`pkg/cloud/azure/`** — `Client` interface fleshed out from scaffold (now has `SubscriptionID()`, `Location()`, `ListSQLDatabases()`, `ListDisks()`). `types.go` adds narrow projections of `SQLDatabase` + `Disk` so probes don't depend on `azure-sdk-for-go` directly.
- **`internal/cloud/azure/SQLDatabases`** — reports drift on Azure SQL Database: terminal states (Offline / Suspect / EmergencyMode / Inaccessible / Disabled) critical, Paused warning (expected for Serverless tier; flagged for awareness), transitional (Restoring / Scaling / etc.) warning, storage ≥ 80%/90% warn/critical, missing automated backups warning. Subject convention: `azure-sql/<subscription>/<resourceGroup>/<server>/<db>`.
- **`internal/cloud/azure/Disks`** — reports drift on Managed Disks: `ProvisioningState=Failed` critical, transitional past 1h grace warning, `DiskState=Unattached` past 24h cleanup grace warning (billing leak / orphaned PV). Subject convention: `azure-disk/<subscription>/<resourceGroup>/<disk>`.
- **`catalog/cloud.go`** — `RegisterCloudOSS` now registers the Azure probes when `azureEnabled=true` (the last unused parameter).
- 22 unit tests via fake client (12 SQLDatabases + 10 Disks).

### Added — Azure cloud probes (Sprint 2 slice)

Four more Azure probes (6 of 10 now shipped). Mirrors GCP Sprint 2. Remaining 4 (App Gateway backend, certs, Storage public-access, Key Vault) follow in Sprint 3.

- **`AKSControlPlane`** — configured cluster (env `CLOUD_AZURE_AKS_CLUSTER`) `ProvisioningState=Failed` or `PowerState=Stopped` critical, non-Succeeded warning, not-found critical. Mirrors AWS `EKSControlPlane` / GCP `GKEControlPlane`. Subject `azure-aks/<subscription>/<resourceGroup>/<cluster>`.
- **`AKSNodePools`** — Failed provisioning critical, Stopped / non-Succeeded warning. Subject `azure-aks-nodepool/<subscription>/<cluster>/<pool>`.
- **`ManagedIdentities`** — user-assigned identity with zero role assignments warning (orphaned; workloads using it silently lack permissions). Mirrors AWS `IAMRoles` / GCP `IAMServiceAccounts`. Subject `azure-mi/<subscription>/<resourceGroup>/<name>`.
- **`Subnets`** — VNet subnet IP-exhaustion: < 10% free critical, < 25% warning; zero-total skipped. Subject `azure-subnet/<subscription>/<vnet>/<name>`.
- `pkg/cloud/azure` Client + types extended; `catalog/cloud.go` registers all 6 Azure probes when `azureEnabled=true`.
- 17 unit tests.

### Added — Azure cloud probes (Sprint 3 slice — 10/10 Azure parity)

Final 4 Azure probes — completes 10-probe parity with AWS + GCP. All three providers now have a 10-probe detection set with identical contracts.

- **`AppGatewayBackends`** — 0 healthy members critical, partial-unhealthy warning. Mirrors AWS `ALBTargetHealth` / GCP `LoadBalancerBackends`. Subject `azure-appgw/<subscription>/<gateway>/<pool>`.
- **`Certificates`** — not-issued critical, < 21d-to-expiry warning. Mirrors AWS `ACMCertExpiry` / GCP `ManagedCertificates`. Subject `azure-cert/<subscription>/<resourceGroup>/<name>`.
- **`StoragePublicAccess`** — `allowBlobPublicAccess=true` critical, non-HTTPS-only warning. Mirrors AWS `S3BucketPublicAccess` / GCP `GCSPublicAccess`. Subject `azure-storage/<subscription>/<resourceGroup>/<name>`.
- **`KeyVaults`** — no soft-delete critical, soft-delete-without-purge-protection warning (Azure's data-protection-posture analog to AWS/GCP KMS key-state). Subject `azure-keyvault/<subscription>/<resourceGroup>/<name>`.
- `catalog/cloud.go` registers all 10 Azure probes when `azureEnabled=true`.
- 17 unit tests.

> **Note on cloud-probe execution:** the GCP + Azure probe *detection logic* is complete and unit-tested (10 probes each, parity with AWS), but neither provider has a Live SDK wrapper yet (`internal/cloud/{gcp,azure}/live.go` absent; `cloud.google.com/go` / `azure-sdk-for-go` not in go.mod). `cmd/srenix buildCloudSource()` still errors for `--cloud-gcp-enabled` / `--cloud-azure-enabled`. Until the Live wrappers land, only **AWS** cloud probes execute against a real cloud; GCP/Azure are dormant. The Live wrappers are the remaining v1.8 cloud item.

### Added — M2 K8s probes (Kong / HPA / ArgoCD / Velero)

Four new resource-event-driven probes from `docs/design/2026-05-trigger-expansion-roadmap.md` M2/M3 and v1.8 roadmap §A5. Each auto-skips when its CRD is absent (or no-ops on an empty list for HPA), so default-on is safe. Each is independently disablable via `SRENIX_PROBE_<NAME>=off`.

- **`Kong`** — flags `KongPlugin` resources reporting `status.conditions[type=Programmed,status=False]` (the gateway is serving upstream traffic without the intended policy). Critical. Auto-skips when `configuration.konghq.com` CRDs are absent.
- **`HPAScaling`** — fast-path complement to the v1.8 `CapacityDrift` analyzer: any HPA with `ScalingActive=False` or `AbleToScale=False` *right now* (no grace) is critical. Empty cluster → HEALTHY (no opt-out needed).
- **`ArgoCDApplication`** — fast-path complement to the v1.7 `GitOpsDrift` analyzer: `health.status` Degraded/Missing/Suspended critical, `sync.status` OutOfSync/Unknown warning. No grace. Auto-skips when `argoproj.io` CRDs are absent.
- **`Velero`** — most-recent Backup per schedule: `Failed`/`PartiallyFailed` critical, `Completed` but older than the 26h SLA critical, `InProgress` past 4h warning. Groups by `velero.io/schedule-name`. Auto-skips when `velero.io` CRDs are absent.
- `internal/snapshot/file.go` `kindToResource` extended with `HorizontalPodAutoscaler` / `KongPlugin` / `Application` / `Backup` so file-based snapshot capture covers these probes too.
- Reader ClusterRole extended with `configuration.konghq.com/kongplugins` + `velero.io/backups` (HPA + ArgoCD reads already granted by B5/B1).
- 17 unit tests.

### Deferred (still on the v1.8 plan)

Reserve for v1.8 — remaining 8 GCP probes (GKE / IAM / LB / GMSC / GCS / KMS / VNet) + the GCP Live SDK wrapper (`cloud.google.com/go`), Azure probes (all 10) + the Azure Live SDK wrapper (`azure-sdk-for-go`), envtest-driven integration tests for the operator, plus the metrics-server-dependent capacity signals (pod request vs usage, PVC growth-trajectory). See `docs/design/2026-05-v1.8-roadmap.md` and `docs/design/2026-05-v1.8-operator-phase-1.md`.

---

## [1.7.0] — 2026-05-27

Drift-class expansion release. Closes Workstream B of the AI SRE positioning plan (`docs/design/2026-05-ai-sre-positioning.md`): the agent's investigation surface broadens from secret/credential drift to three additional classes that page oncall in practice.

### Added — three new drift-class analyzers

- **`GitOpsDrift`** (B1, [#69](https://github.com/srenix-ai/agentic-sre/pull/69)) — Argo CD `Application` out-of-sync / Degraded health + Flux `Kustomization` / `HelmRelease` Ready=False past grace. Reasons matching `*Failed` (BuildFailed, UpgradeFailed, InstallFailed) escalate to critical. 10-minute default grace period (controllers reconcile continuously). Reader ClusterRole extended with read on `argoproj.io/applications`, `kustomize.toolkit.fluxcd.io/kustomizations`, `helm.toolkit.fluxcd.io/helmreleases`. Default ON; flip `analyzers.gitopsDrift.enabled=false` on clusters without Argo/Flux. 15 unit tests.

- **`WorkloadStateDrift`** (B2, [#70](https://github.com/srenix-ai/agentic-sre/pull/70)) — state-tier health drift the basic "X/Y ready" probe misses. CNPG cluster: non-healthy phase (warning, or critical if failover/failed), follower-degraded-while-phase-healthy (early signal), primary switchover stuck (critical, names both endpoints). StatefulSet ordinal-zero: pod-0 missing while other ordinals running (critical), pod-0 unready while higher ordinals Ready (warning). 5-minute default grace. Default ON; flip `analyzers.workloadStateDrift.enabled=false` to disable. 12 unit tests.

- **`RBACDrift`** (B3, [#71](https://github.com/srenix-ai/agentic-sre/pull/71)) — RBAC posture changes that are audit-relevant. Wildcard verbs in user-defined Role/ClusterRole (warning) — skips system canonical roles (`cluster-admin`, `system:*`) and kube-system / kube-public / kube-node-lease namespaces. Unbound ServiceAccount mounted by a Pod (warning) — skips the `default` SA in every namespace + kube-system Pods. Remediation includes the exact `kubectl create rolebinding` command. Reader ClusterRole extended with read on `rbac.authorization.k8s.io/{roles,rolebindings,clusterroles,clusterrolebindings}` + `core/serviceaccounts`. Default ON; flip `analyzers.rbacDrift.enabled=false` to disable. 12 unit tests.

### Added — chart wiring

- New `analyzers.gitopsDrift.enabled` / `analyzers.workloadStateDrift.enabled` / `analyzers.rbacDrift.enabled` values (all default `true`)
- New `srenix.analyzerToggleEnv` chart helper emits `SRENIX_ANALYZER_<NAME>=off` env when an analyzer is disabled
- Watcher Deployment + diagnose CronJob both pick up the helper

### Demo

- `demo/run-demo-v3.sh` (Workstream A4, [#68](https://github.com/srenix-ai/agentic-sre/pull/68)) — sales/stakeholder walkthrough leading with the AI SRE agent flow rather than the OSS engine bootstrap. T0 narration → T1 fix proposer → T3 vault break-glass → JSONL audit. 510 lines, six narration sections.

### Out of scope (deliberately deferred)

- **Config drift** (CM hash divergence, CRD version mismatch, Helm release values vs cluster-live) — v1.8
- **Capacity drift** (HPA min/max divergence, PVC growth trajectory, pod resource-request vs actual usage) — v1.8 (needs metrics-server integration)
- **Security drift** (Pod Security Standards downgrade, image attestation, NetworkPolicy coverage gaps) — v1.8
- **RBAC out-of-band edits** (annotation-vs-spec diff) — v1.8 (diff logic significantly more complex than wildcards / binding walks)
- **GCP + Azure cloud probes** — v1.7+ (`pkg/cloud/{gcp,azure}` scaffolds in place)
- **Operator port** (controller-runtime / kubebuilder) — v1.7+

### Companion Srenix Enterprise release

Srenix Enterprise v1.7.0 (separate repo) lands the C5 stretch: `LLMFixerMatcher` replaces the keyword `DefaultFixerMatcher` switch with an opt-in LLM classification call (`--ai-llm-fixer-matcher`). Same action_kind whitelist, but the decision of which fixer to invoke becomes LLM-driven. Falls back to keyword on LLM error / invalid response — worst case is identical to v1.6 behavior.

---

## [1.6.2] — 2026-05-27

Pinned chart + binary release reconciling the `feat/cloud-probes` lineage onto `main` (PR #63 + #64). The v1.6.0 binary was previously deployed but its source never landed on `main`; this is the source-of-truth cleanup. Live cluster upgraded from v1.6.1 → v1.6.2 with lease-based leader election now genuinely active (lease transitions = 3, renewing every 5s).

### Added — content from `feat/cloud-probes` merged into main
- All Sprint 1–4 hardening work (see [1.6.0] below) is now reflected on origin/main with file-level history preserved.
- AWS cloud probe code (RDS / EBS / EKS-cluster / EKS-nodegroups / IAM-roles / ALB / ACM / KMS / S3-public-access / VPC-subnets) under `pkg/cloud/aws` + `internal/cloud/aws`. Default `cloud.enabled=false` — operators opt in.
- Lease-based leader election (`internal/watcher/leader.go`), wired by the chart's downward-API env vars (`MY_POD_NAMESPACE`, `MY_POD_NAME`).
- `pkg/ai/redact` Kubernetes Event message scrubbing (Sprint 3.4).
- `internal/fix/gitops.go` shared GitOps detection helpers used by all 5 fixers (Sprint 1).
- `pkg/vault` promotion (Client / HTTPClient / Config / KubernetesAuthConfig moved from `internal/vault`).

### Fixed
- CI helm-unittest setup: drop removed `--verify=false` flag and pin plugin to v1.0.3 so Helm 3.16 + plugin.yaml metadata stay compatible.

---

## [1.6.1] — 2026-05-26

Operator-driven Slack-noise fixes after a stable warning was observed re-posting 6× per day at the default 4h cadence.

### Added
- **Per-severity Slack repeat intervals.** New `--slack-critical-repeat-interval`
  flag on `srenix watch` lets operators keep critical alerts loud (e.g. `4h`)
  while letting warnings calm down (e.g. `--slack-repeat-interval=24h`).
  Zero (default) falls back to `--slack-repeat-interval` so pre-v1.6.1
  callers see identical behavior. Chart value:
  `watcher.slack.criticalRepeatInterval` (empty string = fallback).
  Replaces the single-cadence behavior reported as noisy on long-running
  warnings: a stable warning would previously re-post 6×/day at the 4h
  default.

### Fixed
- **Helm chart: `watcher.slack.postOnResolved=false` was silently flipped
  back to `true`.** The template line
  `{{ ... | default true }}` treats bool `false` as empty under sprig and
  substitutes the default. Operators who set the value via `helm --set`
  or values.yaml override saw the rendered Deployment still carrying
  `--slack-post-on-resolved=true`. Same bug latent on `repeatInterval`
  (string "4h" never tripped the empty check, but the pattern was
  unsafe). values.yaml already provides sane defaults, so we now render
  the values directly. Chart version bumped 1.6.0 → 1.6.1.

---

## [1.6.0] — 2026-05-25

Sprint 1–4 hardening release. Closes 22/23 items from the 2026-05-22 adversarial review (one trigger-expansion roadmap item correctly deferred to v1.7+). Live-deployed to the development cluster on 2026-05-25 as image tag `v1.6.0-aeefa30`.

### Added
- `LICENSE-VERIFIED-LIBRARY.md` — formal terms for the paid Verified Signature Library subscription, replacing the placeholder reference in README.
- README section documenting the AWS cloud probes (RDS, EBS, EKS, IAM, ALB, ACM, KMS, S3, VPC) that were already shipping but undocumented.
- README link to `docs/READINESS.md` so prospects find the pilot-vs-production limits doc before the install step.
- `docs/AI_COST_MODEL.md` — failure-mode amplification section covering flapping-workload cost blowup, pre-Sprint-3 investigation rate-limiter gaps, and the worst-case planning table.
- `docs/design/2026-05-hardening-plan.md` — TDD-driven Sprint 0–4 plan closing the 2026-05-22 adversarial review.
- `internal/fix/gitops.go` — new public, kind-agnostic helpers `GitOpsReason()` and `IsPaused()` and `IsSuspended()` that any fixer can consult before mutating a resource. Lifts the private Ingress-only `isGitOpsManaged()` helper out of `tls_secret_mismatch.go` and broadens it to all kinds.
- Helm value `diagnose.backoffLimit` / `diagnose.activeDeadlineSeconds` (defaults 1 / 120s) and matching `remediation.backoffLimit` / `remediation.activeDeadlineSeconds` to cap CronJob retry storms.
- **Sprint 2 — six new probes** closing the most-impactful blind spots called out in the 2026-05-22 adversarial review:
  - `NodePressure` — surfaces DiskPressure / MemoryPressure / PIDPressure / NetworkUnavailable conditions the basic `Nodes` probe (which only checks `Ready`) misses. DiskPressure and NetworkUnavailable escalate to Critical; the others surface as Warning.
  - `DaemonSets` — checks DaemonSets in system namespaces (kube-system, cilium-system, calico-system, kube-flannel, rook-ceph, longhorn-system, openebs, metallb-system) so a broken CNI/CSI plugin shows up before nodes flip NotReady.
  - `PendingPods` — flags pods whose `PodScheduled` condition is False past a 60s grace window, with reason-aware remediation (Insufficient CPU/Memory, unbound PVC, taint mismatch, nodeSelector). Skips ImagePullBackOff (owned by the existing ImagePullAuth analyzer).
  - `CrashLoopBackOff` — generic crash-loop detector for any namespace, replacing the previous behavior where only workloads on the hardcoded critical list were caught. Severity scales: protected-namespace = Critical immediately; user namespaces = Warning until restart count exceeds the configurable threshold (default 10).
  - `ETCD` — watches the static-pod etcd members in `kube-system` (kubeadm convention) for Ready=False or restartCount>0. Honestly reports Warning ("probe is blind") when no in-cluster etcd is found rather than false-greening on managed control planes.
  - `FailedMounts` — joins Pods stuck in ContainerCreating past a 90s grace window with their kubelet `FailedMount` / `FailedAttachVolume` / `ProvisioningFailed` events to name the volume that's stuck and explain why.
- Configurable Services-probe targets via `SRENIX_CRITICAL_SERVICES` env var (semicolon-separated `ns/selector|Display` pairs) and the `srenix.ai/probe-critical: "true"` annotation on any Deployment / StatefulSet. The compiled-in defaults remain the baseline; set `SRENIX_CRITICAL_SERVICES_REPLACE=true` to fully replace them.
- New `IsProtectedNamespace` helper in `internal/probe/` (duplicated from `internal/fix/protected.go` for package isolation; consolidation tracked under Sprint 5).
- `GVRDaemonSet` exposed by `internal/snapshot/` and wired into both `snapshot.CaptureGVRs` and the watcher's `watchedGVRs` so the new probe sees changes in real time and is captured by `srenix snapshot capture`.
- Per-probe opt-out env vars: `SRENIX_PROBE_NODE_PRESSURE`, `SRENIX_PROBE_DAEMONSETS`, `SRENIX_PROBE_PENDING_PODS`, `SRENIX_PROBE_CRASHLOOP`, `SRENIX_PROBE_ETCD`, `SRENIX_PROBE_FAILED_MOUNTS` (set to `off` to silence individual probes without forking).
- **Sprint 3 — AI safety hardening (Srenix Enterprise paid binary).** Patch-payload semantic validator (`ai/approval/patch_validator.go`) — the closed-enum `ActionKind` whitelist now gates *shape* as well as verbs; LLM-proposed `{"spec":{"replicas":0}}` on a StatefulSet is rejected at admission. Investigation rate limiter (`ai/rate_limit.go::TakeInvestigation`) with per-diagnostic-class budgets, independent from the proposal budget. Cold-start bucket initialization (default 0 tokens) closes the pod-restart burst attack. Hash-chained audit sink (`ai/audit/hash_chain.go`) makes audit-trail tampering detectable via `VerifyChain`. See [Srenix Enterprise commits at d38287d..552004b](https://github.com/srenix-ai/agentic-sre-enterprise/commits/main) for the private repo history.
- **Sprint 3.4 — Event-message secret scrubbing in OSS.** New helpers `pkg/ai.RedactEventMessage` and `pkg/ai.RedactEvents` apply both identifier redaction and the existing secret heuristics (AWS access keys, Vault tokens, JWTs, GitHub PATs, Slack tokens) to event `.Message` strings. Wired into `internal/investigator.LiveEnvironment.GetEvents` so any LLM-backed investigator sees scrubbed events.
- **Sprint 4.1 — Watcher unit tests.** 12 new tests covering `fingerprint()`, `buildCurrentState()`, `diff()`, `updateSeen()` — the dedup logic that previously had zero unit coverage. Brings the watcher package up from 2 to 14 tests, and any future refactor of the seen-map or post-fix-state handling now has a regression net.
- **Sprint 4.2 — Ticketing flag validation.** `--ticketing-provider=openproject` now fails fast with a descriptive error when `--ticketing-mcp-url`, `--ticketing-project`, or `$TICKETING_MCP_API_KEY` are missing — instead of silently constructing a misconfigured client that errors at first-ticket time.
- **Sprint 4.3 — Lease-based leader election.** `internal/watcher/leader.go` wraps the watcher loop with `k8s.io/client-go/tools/leaderelection`. Default lease name `srenix-watcher` in the install namespace; 30s LeaseDuration / 20s RenewDeadline / 5s RetryPeriod (kube-controller-manager defaults). Two watcher replicas now race for the lease — only the holder runs the probe/fix/post cycle. Set `SRENIX_LEADER_ELECTION=off` or `watcher.leaderElection.enabled=false` to disable for single-pod dev. New namespace-scoped `Role` for the `srenix-watcher` Lease minimizes blast radius. Downward-API env (`MY_POD_NAMESPACE`, `MY_POD_NAME`) wired in the watcher deployment template.
- **Sprint 4.4 — Multi-registry image default.** Helm chart now pulls `ghcr.io/srenix-ai/agentic-sre` by default. `docker4zerocool/agentic-sre` remains as a mirror (the GoReleaser config publishes to both registries on every tag). Operators who can't reach GHCR continue to work unchanged.
- **Sprint 4.5 — OSS/paid boundary exerciser.** Srenix Enterprise's `catalog/paid.go` now registers a no-op `PaidBoundaryAnalyzer` whose only purpose is to fail the paid build at CI time if the OSS `pkg/diagnose.Analyzer` interface or `pkg/registry.Registry` shape drifts.

### Changed
- README architecture section now describes the actual Go-binary-on-distroless image and the three ClusterRoles (reader, remediator, driftreport) — the old description of a bash/jq/curl container and "two ClusterRoles" was inherited from a v0.x iteration.
- README and `docs/SRENIX_OVERVIEW.md` clarify that `VaultPathMissing` source code is Apache-2.0 OSS but ships unwired (you supply the Vault client); the paid Srenix Enterprise binary auto-wires it.
- README roadmap section replaced the user-local path with links to `docs/design/`.
- `docs/FAILURE_MODES.md` analyzer count corrected from "seven" to "eight"; intro now distinguishes "source ships OSS" vs. "auto-wired in paid."

### Fixed
- **StuckRSPods** now refuses to `kubectl rollout restart` a Deployment that is GitOps-managed (Argo CD / Flux / Helm via `app.kubernetes.io/managed-by` or the per-controller annotations) or has `spec.paused=true`. Previously Srenix would patch the restart annotation and the GitOps controller would revert it on the next reconcile, locking the two into a tight fight loop.
- **StuckJobsWithBadSecretRef** now fetches the parent CronJob and refuses to delete the broken Job when the CronJob has `spec.suspend=true` (an operator's deliberate freeze) or is GitOps-managed.
- **StaleErrorPods** now skips Failed pods that are either GitOps-annotated themselves or owned by a GitOps-managed Job. When the owning Job isn't in the captured snapshot the fixer falls back to today's cleanup behavior — orphan Failed pods remain garbage-collectable.
- **StuckCertificateRequests** now refuses to delete CRs when the cert-manager controller Deployment is captured in the snapshot and reports `readyReplicas=0`. cert-manager cannot recreate them in that state; the deletion would just nuke the diagnostic evidence without enabling retry.
- Helm chart CronJob Jobs now declare `backoffLimit: 1` (was K8s default 6) and `activeDeadlineSeconds: 120` (was unlimited) so a hung run cannot keep spawning pods for hours.

---

## [1.5.2] — 2026-05-11

### Fixed
- Watcher now re-runs the Layer-2 investigator after fixers apply; the resulting investigation is reflected in the DriftReport CR.
- DriftReport CR severity and message refresh on update, not just on create.

## [1.5.1] — 2026-05-11

### Fixed
- Investigation field now persists on the DriftReport CR.

<!-- NOTE (P3.3): date inversion below — [1.5.0] is dated 2026-05-12 yet its
     patch releases [1.5.1]/[1.5.2] above are dated 2026-05-11. Left as-is:
     it is ambiguous whether 1.5.0's date or the patch dates are the typo, and
     the historical tags can't be retro-corrected from the changelog alone.
     Semver heading order is correct, so changelog-lint.sh (version-order based)
     does not flag this. -->
## [1.5.0] — 2026-05-12

### Added
- Layer-2 Investigator: read-only deep-dive on CRITICAL findings.
- OSS ships a deterministic, rule-based investigator (TLS expiry, TLS SAN mismatch, DNS, slow-DNS, status mismatch, ExternalSecret, Certificate state).
- Srenix Enterprise paid binary swaps in an LLM-backed investigator via the same `Environment` interface.

## [1.4.0] — 2026-05-12

### Added
- Probe flake suppression: retry + N-of-M streak gating before escalation to CRITICAL. Deterministic failures (TLS error, status mismatch) bypass the streak counter.

## [1.3.0] — 2026-05-12

### Added
- `TLSSecretMismatch` analyzer + opt-in fixer that repoints `Ingress.spec.tls[].secretName` to the cert-manager-managed Secret. Skips GitOps-managed Ingresses.

## [1.2.0] — 2026-05-12

### Added
- Ingress host auto-discovery: every Ingress host in the cluster is probed externally by default.

### Removed
- `IngressCoverage` analyzer (replaced by auto-discovery).

## [1.1.0] — 2026-05-12

### Added
- Expanded default endpoint probe coverage.

## [1.0.0] — 2026-05-11

### Fixed
- AI-related Helm templates are now nil-safe for `--reuse-values` upgrades.

## [0.9.5] — 2026-05-11

### Added
- External endpoint probe.
- Ingress coverage analyzer (later superseded in 1.2.0).
- Rewritten `SETUP_GUIDE.md` for v0.9.5; corrected `NOTES.txt` template.

## [0.9.1] — 2026-05-08

### Added
- `StuckCertificateRequests` fixer: deletes terminal-failed cert-manager Certificate Requests so cert-manager re-issues.

## [0.9.0] — 2026-05-07

### Added
- `srenix watch --live` event-driven watcher with Slack dedup (Phase 1).

---

For releases earlier than 0.9.0, see the git tag list and PR titles on GitHub.
