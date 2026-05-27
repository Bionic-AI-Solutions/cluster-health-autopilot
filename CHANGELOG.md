# Changelog

All notable changes to this project will be documented in this file. The
format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

The Helm chart `cluster-health-autopilot` follows the same version line as the
`cha` binary (`appVersion == version`). Released chart artifacts are tagged
`cluster-health-autopilot-X.Y.Z`; the binary itself is tagged `vX.Y.Z`. The
published Helm repository at
`https://bionic-ai-solutions.github.io/cluster-health-autopilot/` always
serves the latest tagged chart cut.

## [Unreleased]

### Added — Workstream B4 (config drift)

- **`ConfigDrift`** analyzer (B4) — three signals the basic resource-health probes miss:
  - **CRD multi-storedVersions** — `apiextensions.k8s.io/v1` CRDs whose `status.storedVersions` lists more than one apiserver storage version. Storage migration is pending; future drops of the old version will fail. Critical.
  - **Deployment rollout stuck** — `metadata.generation` ahead of `status.observedGeneration` past the grace window (the controller hasn't reconciled the latest spec; critical), or `status.updatedReplicas` < `spec.replicas` past the grace window (rollout stuck mid-flight; warning if some replicas still available, critical if zero). Default 15-minute grace.
  - **`checksum/config` annotation drift** — Pods of the same Deployment carrying disagreeing `checksum/config` annotation values, indicating a rolling update from the last config change didn't propagate to all replicas. Warning. Skipped on workloads that don't carry the annotation.
- Walks via owner-reference chain Pod → ReplicaSet → Deployment.
- Skips system namespaces (kube-system, kube-public, kube-node-lease).
- Reader ClusterRole extended with read on `apiextensions.k8s.io/customresourcedefinitions`.
- Default ON; flip `analyzers.configDrift.enabled=false` to disable, or set `CHA_ANALYZER_CONFIG_DRIFT=off`. 16 unit tests.

### Deferred (still on the v1.8 plan)

Reserve for v1.8 — capacity / security drift classes, GCP+Azure cloud probes, M2 K8s probes (Kong, HPA, ArgoCD Application, Velero), operator port (controller-runtime Phase 1). See `docs/design/2026-05-v1.8-roadmap.md`.

---

## [1.7.0] — 2026-05-27

Drift-class expansion release. Closes Workstream B of the AI SRE positioning plan (`docs/design/2026-05-ai-sre-positioning.md`): the agent's investigation surface broadens from secret/credential drift to three additional classes that page oncall in practice.

### Added — three new drift-class analyzers

- **`GitOpsDrift`** (B1, [#69](https://github.com/Bionic-AI-Solutions/cluster-health-autopilot/pull/69)) — Argo CD `Application` out-of-sync / Degraded health + Flux `Kustomization` / `HelmRelease` Ready=False past grace. Reasons matching `*Failed` (BuildFailed, UpgradeFailed, InstallFailed) escalate to critical. 10-minute default grace period (controllers reconcile continuously). Reader ClusterRole extended with read on `argoproj.io/applications`, `kustomize.toolkit.fluxcd.io/kustomizations`, `helm.toolkit.fluxcd.io/helmreleases`. Default ON; flip `analyzers.gitopsDrift.enabled=false` on clusters without Argo/Flux. 15 unit tests.

- **`WorkloadStateDrift`** (B2, [#70](https://github.com/Bionic-AI-Solutions/cluster-health-autopilot/pull/70)) — state-tier health drift the basic "X/Y ready" probe misses. CNPG cluster: non-healthy phase (warning, or critical if failover/failed), follower-degraded-while-phase-healthy (early signal), primary switchover stuck (critical, names both endpoints). StatefulSet ordinal-zero: pod-0 missing while other ordinals running (critical), pod-0 unready while higher ordinals Ready (warning). 5-minute default grace. Default ON; flip `analyzers.workloadStateDrift.enabled=false` to disable. 12 unit tests.

- **`RBACDrift`** (B3, [#71](https://github.com/Bionic-AI-Solutions/cluster-health-autopilot/pull/71)) — RBAC posture changes that are audit-relevant. Wildcard verbs in user-defined Role/ClusterRole (warning) — skips system canonical roles (`cluster-admin`, `system:*`) and kube-system / kube-public / kube-node-lease namespaces. Unbound ServiceAccount mounted by a Pod (warning) — skips the `default` SA in every namespace + kube-system Pods. Remediation includes the exact `kubectl create rolebinding` command. Reader ClusterRole extended with read on `rbac.authorization.k8s.io/{roles,rolebindings,clusterroles,clusterrolebindings}` + `core/serviceaccounts`. Default ON; flip `analyzers.rbacDrift.enabled=false` to disable. 12 unit tests.

### Added — chart wiring

- New `analyzers.gitopsDrift.enabled` / `analyzers.workloadStateDrift.enabled` / `analyzers.rbacDrift.enabled` values (all default `true`)
- New `cha.analyzerToggleEnv` chart helper emits `CHA_ANALYZER_<NAME>=off` env when an analyzer is disabled
- Watcher Deployment + diagnose CronJob both pick up the helper

### Demo

- `demo/run-demo-v3.sh` (Workstream A4, [#68](https://github.com/Bionic-AI-Solutions/cluster-health-autopilot/pull/68)) — sales/stakeholder walkthrough leading with the AI SRE agent flow rather than the OSS engine bootstrap. T0 narration → T1 fix proposer → T3 vault break-glass → JSONL audit. 510 lines, six narration sections.

### Out of scope (deliberately deferred)

- **Config drift** (CM hash divergence, CRD version mismatch, Helm release values vs cluster-live) — v1.8
- **Capacity drift** (HPA min/max divergence, PVC growth trajectory, pod resource-request vs actual usage) — v1.8 (needs metrics-server integration)
- **Security drift** (Pod Security Standards downgrade, image attestation, NetworkPolicy coverage gaps) — v1.8
- **RBAC out-of-band edits** (annotation-vs-spec diff) — v1.8 (diff logic significantly more complex than wildcards / binding walks)
- **GCP + Azure cloud probes** — v1.7+ (`pkg/cloud/{gcp,azure}` scaffolds in place)
- **Operator port** (controller-runtime / kubebuilder) — v1.7+

### Companion CHA-com release

CHA-com v1.7.0 (separate repo) lands the C5 stretch: `LLMFixerMatcher` replaces the keyword `DefaultFixerMatcher` switch with an opt-in LLM classification call (`--ai-llm-fixer-matcher`). Same action_kind whitelist, but the decision of which fixer to invoke becomes LLM-driven. Falls back to keyword on LLM error / invalid response — worst case is identical to v1.6 behavior.

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
  flag on `cha watch` lets operators keep critical alerts loud (e.g. `4h`)
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
- Configurable Services-probe targets via `CHA_CRITICAL_SERVICES` env var (semicolon-separated `ns/selector|Display` pairs) and the `cha.bionicaisolutions.com/probe-critical: "true"` annotation on any Deployment / StatefulSet. The compiled-in defaults remain the baseline; set `CHA_CRITICAL_SERVICES_REPLACE=true` to fully replace them.
- New `IsProtectedNamespace` helper in `internal/probe/` (duplicated from `internal/fix/protected.go` for package isolation; consolidation tracked under Sprint 5).
- `GVRDaemonSet` exposed by `internal/snapshot/` and wired into both `snapshot.CaptureGVRs` and the watcher's `watchedGVRs` so the new probe sees changes in real time and is captured by `cha snapshot capture`.
- Per-probe opt-out env vars: `CHA_PROBE_NODE_PRESSURE`, `CHA_PROBE_DAEMONSETS`, `CHA_PROBE_PENDING_PODS`, `CHA_PROBE_CRASHLOOP`, `CHA_PROBE_ETCD`, `CHA_PROBE_FAILED_MOUNTS` (set to `off` to silence individual probes without forking).
- **Sprint 3 — AI safety hardening (CHA-com paid binary).** Patch-payload semantic validator (`ai/approval/patch_validator.go`) — the closed-enum `ActionKind` whitelist now gates *shape* as well as verbs; LLM-proposed `{"spec":{"replicas":0}}` on a StatefulSet is rejected at admission. Investigation rate limiter (`ai/rate_limit.go::TakeInvestigation`) with per-diagnostic-class budgets, independent from the proposal budget. Cold-start bucket initialization (default 0 tokens) closes the pod-restart burst attack. Hash-chained audit sink (`ai/audit/hash_chain.go`) makes audit-trail tampering detectable via `VerifyChain`. See [CHA-com commits at d38287d..552004b](https://github.com/Bionic-AI-Solutions/CHA-com/commits/main) for the private repo history.
- **Sprint 3.4 — Event-message secret scrubbing in OSS.** New helpers `pkg/ai.RedactEventMessage` and `pkg/ai.RedactEvents` apply both identifier redaction and the existing secret heuristics (AWS access keys, Vault tokens, JWTs, GitHub PATs, Slack tokens) to event `.Message` strings. Wired into `internal/investigator.LiveEnvironment.GetEvents` so any LLM-backed investigator sees scrubbed events.
- **Sprint 4.1 — Watcher unit tests.** 12 new tests covering `fingerprint()`, `buildCurrentState()`, `diff()`, `updateSeen()` — the dedup logic that previously had zero unit coverage. Brings the watcher package up from 2 to 14 tests, and any future refactor of the seen-map or post-fix-state handling now has a regression net.
- **Sprint 4.2 — Ticketing flag validation.** `--ticketing-provider=openproject` now fails fast with a descriptive error when `--ticketing-mcp-url`, `--ticketing-project`, or `$TICKETING_MCP_API_KEY` are missing — instead of silently constructing a misconfigured client that errors at first-ticket time.
- **Sprint 4.3 — Lease-based leader election.** `internal/watcher/leader.go` wraps the watcher loop with `k8s.io/client-go/tools/leaderelection`. Default lease name `cha-watcher` in the install namespace; 30s LeaseDuration / 20s RenewDeadline / 5s RetryPeriod (kube-controller-manager defaults). Two watcher replicas now race for the lease — only the holder runs the probe/fix/post cycle. Set `CHA_LEADER_ELECTION=off` or `watcher.leaderElection.enabled=false` to disable for single-pod dev. New namespace-scoped `Role` for the `cha-watcher` Lease minimizes blast radius. Downward-API env (`MY_POD_NAMESPACE`, `MY_POD_NAME`) wired in the watcher deployment template.
- **Sprint 4.4 — Multi-registry image default.** Helm chart now pulls `ghcr.io/bionic-ai-solutions/cluster-health-autopilot` by default. `docker4zerocool/cluster-health-autopilot` remains as a mirror (the GoReleaser config publishes to both registries on every tag). Operators who can't reach GHCR continue to work unchanged.
- **Sprint 4.5 — OSS/paid boundary exerciser.** CHA-com's `catalog/paid.go` now registers a no-op `PaidBoundaryAnalyzer` whose only purpose is to fail the paid build at CI time if the OSS `pkg/diagnose.Analyzer` interface or `pkg/registry.Registry` shape drifts.

### Changed
- README architecture section now describes the actual Go-binary-on-distroless image and the three ClusterRoles (reader, remediator, driftreport) — the old description of a bash/jq/curl container and "two ClusterRoles" was inherited from a v0.x iteration.
- README and `docs/CHA_OVERVIEW.md` clarify that `VaultPathMissing` source code is Apache-2.0 OSS but ships unwired (you supply the Vault client); the paid CHA Enterprise binary auto-wires it.
- README roadmap section replaced the user-local path with links to `docs/design/`.
- `docs/FAILURE_MODES.md` analyzer count corrected from "seven" to "eight"; intro now distinguishes "source ships OSS" vs. "auto-wired in paid."

### Fixed
- **StuckRSPods** now refuses to `kubectl rollout restart` a Deployment that is GitOps-managed (Argo CD / Flux / Helm via `app.kubernetes.io/managed-by` or the per-controller annotations) or has `spec.paused=true`. Previously CHA would patch the restart annotation and the GitOps controller would revert it on the next reconcile, locking the two into a tight fight loop.
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

## [1.5.0] — 2026-05-12

### Added
- Layer-2 Investigator: read-only deep-dive on CRITICAL findings.
- OSS ships a deterministic, rule-based investigator (TLS expiry, TLS SAN mismatch, DNS, slow-DNS, status mismatch, ExternalSecret, Certificate state).
- CHA-com paid binary swaps in an LLM-backed investigator via the same `Environment` interface.

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
- `cha watch --live` event-driven watcher with Slack dedup (Phase 1).

---

For releases earlier than 0.9.0, see the git tag list and PR titles on GitHub.
