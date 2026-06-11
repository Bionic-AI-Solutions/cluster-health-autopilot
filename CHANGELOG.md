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

### Added — watcher health probes + opt-in multi-replica via leader election (P1.9)

The watcher Deployment shipped with no liveness/readiness probes (every sibling deployment — approval-server, qdrant, operator — had them) because its only HTTP `/healthz` lived inside the `--webhook-listen` branch, so an install without the M6 webhook trigger had no health endpoint to probe. The watcher now starts an **always-on health server** (`--health-listen`, default `:8081`; chart value `watcher.healthListen`) serving `GET /healthz` unconditionally, independent of the webhook receiver, and the chart wires `livenessProbe` + `readinessProbe` against it. The watcher Deployment's hard-coded `replicas: 1` is now `watcher.replicas` (default 1); raising it above 1 is only safe with leader election on, so the chart **fails the render** when `watcher.replicas > 1` and `watcher.leaderElection.enabled=false` (otherwise replicas race on DriftReports and double-post Slack).

### Fixed — watcher: pending approval-URL cache grew unbounded (P1.9)

The `pendingURLs` map (approval URLs keyed by ActionID for the AI tier) evicted entries only on lookup (`approvalURLFor`). A recorded-but-never-rendered ActionID — e.g. a diagnostic that resolved before its next post — persisted for the whole process lifetime, a slow memory leak on long-running watchers with the AI tier enabled. `recordApprovalURL` now sweeps entries older than a 24h TTL on every insert (and lookup still evicts on access), via an injectable clock seam.

### Fixed — operator: cross-namespace approval events Role/RoleBinding leaked on CR deletion (P1.9)

When a CR pinned `spec.approval.auditNamespace` to a namespace other than its own, the operator created the `<name>-events` Role + RoleBinding there **without** an ownerRef (cross-namespace ownerRefs are illegal), so Kubernetes GC never reaped them. Teardown only ran on disable-while-alive — a straight `kubectl delete` of the CR skipped that path, leaking the cross-namespace RBAC pair for the cluster's lifetime. The operator's finalizer now also deletes those objects (NotFound ignored, so the same-namespace owner-ref'd case is a harmless no-op).

### Changed — webhook trigger sources now FAIL CLOSED on missing HMAC secret (P1.1, breaking-ish)

Before this change a `--webhook-source=<name>=<env-var>` whose env var was unset or empty (secret not mounted, ESO key drift, typo, or a spec entry without `=`) silently registered the source with HMAC verification DISABLED — any unauthenticated POST to `/webhook/<name>` triggered a full diagnose cycle (and fixer churn under `--remedy`). Now:

- Registration fails closed: a missing/empty env var or malformed spec logs an `ERROR … source disabled (fail-closed)` and the source is NOT registered (requests 404).
- Defense in depth: should a source ever be registered with an empty secret, the handler rejects every request for it with 401 instead of skipping verification.
- Explicit opt-out: the literal spec `<name>=insecure-no-hmac` registers a deliberately unauthenticated source and logs a loud `UNAUTHENTICATED webhook source` warning at startup.

**Migration:** deployments that (knowingly or not) relied on an empty secret to run an unauthenticated source must either mount a real secret or switch the spec to `<name>=insecure-no-hmac`.

### Fixed — feeder: workload digest index collided across workloads in a namespace (P1.6)

The workload feeder's pod-digest index was keyed by (namespace, container-name) only, despite a comment claiming it was scoped to the owning controller. Two workloads in one namespace that both name their container e.g. `app` (extremely common) silently received each other's `image_digest` — first pod observed won — so a digest-pin PR proposal built downstream could previously cite a **sibling workload's digest** and pin the wrong image. The index is now scoped to the owning workload (each pod's controller ownerReference, with Deployment names recovered from the ReplicaSet `<deployment>-<pod-template-hash>` convention), and as a second guard a digest only attaches when the repo it was pulled from matches the workload's declared image repo (so a mid-rollout pod still running the old repo's image can no longer stamp its digest onto the new spec). Pods whose owner can't be resolved (bare pods, Jobs, bare ReplicaSets) now contribute no digest — fail-closed; the entry simply omits `image_digest` until a resolvable pod is observed.

### Fixed — operator: `spec.externalDNS` was accepted but did nothing (P1.5)

The CRD documented `spec.externalDNS.cloudflare.*` (incl. `apiTokenSecretRef`) and the operator accepted it — but consumed it nowhere. The DNSChainDrift analyzer only wires its Cloudflare client when `CHA_CLOUDFLARE_TOKEN` is set at registration time, and nothing supplied that env on operator-managed installs, so external-hop DNS verification silently never ran. The operator's watcher Deployment now injects `CHA_CLOUDFLARE_TOKEN` via `secretKeyRef` from `apiTokenSecretRef.{name,key}` (key defaults to `token`) when `cloudflare.enabled=true`. The token value never appears in any manifest.

### Fixed — operator: `spec.watcher.triggers.webhook.serviceEnabled` was accepted but did nothing (P1.5)

The chart has shipped `watcher-webhook-service.yaml` since v1.23.0, but the operator built neither the ClusterIP Service nor the named `webhook` containerPort — an operator-managed webhook receiver was reachable only by pod IP. The operator now reconciles a `<cr>-webhook` ClusterIP Service (port = `servicePort`, default 8090; `targetPort: webhook`; selects the watcher pods) when `serviceEnabled=true`, owner-ref'd to the CR and torn down when the field flips off or the watcher is disabled, and declares the `webhook` containerPort whenever `webhook.listen` is set — both mirroring the chart's semantics exactly.

### Added — optional timestamped HMAC scheme (replay window)

Webhook senders can now include `X-CHA-Timestamp: <unix-seconds>` and sign `timestamp + "." + body` (`X-CHA-Signature: sha256=hex(hmac-sha256(secret, ts+"."+body))`). Timestamped requests more than 5 minutes from server time are rejected with 401, so a captured request can no longer be replayed forever. Requests without the header keep the legacy body-only HMAC check (existing senders unaffected); a once-per-source log notice recommends adopting the timestamp header. New `webhook.SignWithTimestamp` helper for integrators.

## [1.25.1] — 2026-06-11

### Fixed — goreleaser disk-OOM on GH-hosted runner

v1.25.0 goreleaser failed at the docker buildx multi-arch build stage with `no space left on device`. The OSS workflow's transitive deps (AWS SDK v2 + k8s.io + buildx cache) overshoot the ~14 GiB free disk on the GH-hosted runner. v1.24.x and earlier just happened to fit; v1.25.0's added KEDA + extra ownerRef walker pushed past the limit.

Same fix that the CHA-com workflow shipped in v1.20.0: pre-checkout cleanup step removes ~25 GiB of preinstalled .NET / Android SDK / Haskell / Swift / CodeQL toolchains that the workflow doesn't use.

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
- `owner_chart = "<crkind-lowercase>-<crname>"` (e.g. `clusterhealthautopilot-bionic`)
- `owner_release = <CR name>`
- `owner_release_namespace = workload namespace`

Built-in workload parents (apps/v1 ReplicaSet, batch/v1 Job, core/v1) are explicitly skipped — they mean "this Pod is owned by a Deployment", not "this Deployment is operator-managed". 2 new tests cover both the positive (operator CR owner → synthesized) and negative (apps ReplicaSet owner → still nil) cases.

`detectOwner` no longer early-returns on nil annotations — operator-managed workloads typically have NO annotations at all, so the nil-anns path must still walk the OwnerReferences fallback.

## [1.24.1] — 2026-06-10

### Fixed — CRD schema for `spec.watcher.triggers` (v1.24.0 was unusable on schema-strict K8s)

v1.24.0 added the Go types + operator reconciler for `spec.watcher.triggers.{prom,webhook}` but did NOT update the CRD's OpenAPIv3 schema. K8s 1.27+ structural-schema pruning stripped the field at the API server, so any `kubectl apply` of a CR with `triggers` set silently dropped the data. The operator then rendered the watcher Deployment with no trigger args.

This patch adds the matching schema to both `bundle/manifests/cha.bionicaisolutions.com_clusterhealthautopilots.yaml` and `charts/cluster-health-autopilot/templates/crd-clusterhealthautopilot.yaml`. Verified live: `kubectl explain clusterhealthautopilots.spec.watcher.triggers` now resolves and the field persists on `kubectl get`.

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

### Pairs with CHA-com

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

v1.23.0 shipped `Config.PromTriggerURL` but `cmd/cha/watch` never
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
via `CHA_PROBE_KONG_ROUTES=off`.

### Added — M3 `GPUNodes` probe + `LogPatternMatcher` analyzer

- **GPUNodes** — critical on NotReady / zero-allocatable, warning
  on cordoned, for each GPU-advertising Node. Opt out:
  `CHA_PROBE_GPU_NODES=off`.
- **LogPatternMatcher** — scans Events for ImagePullBackOff,
  OOMKilled, probe-failed, volume-attach-failed, RBAC Forbidden.
  Dedup'd per (involved-object, pattern). Opt out:
  `CHA_ANALYZER_LOG_PATTERN_MATCHER=off`.

### Added — M4 Endpoints probe Layer-7 mode

`EndpointTarget.L7` populated from three Ingress annotations
(`cha.bionicaisolutions.com/probe-l7-{path,expect,status}`). When
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
- `charts/cluster-health-autopilot/templates/clusterrole-reader.yaml`
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
  simultaneously). Opts out via `CHA_ANALYZER_OOMKILL_RECURRENCE=off`.
- **`PVOrphan`** (warning) — PersistentVolume in `Released` phase for
  >7d. Underlying cloud disk (EBS / GCE-PD / Azure-Disk) may still be
  billing. Message surfaces storageClass + capacity + reclaimPolicy
  for cost-sizing. Opts out via `CHA_ANALYZER_PV_ORPHAN=off`.
- **`CronJobStuck`** (warning/critical) — CronJob whose lastSuccessfulTime
  is >24h old OR has never succeeded OR is suspended. Each cause gets
  tailored remediation guidance. Opts out via `CHA_ANALYZER_CRONJOB_STUCK=off`.

### Added — `spec.ai.metrics` + `spec.ai.llmProposer` typed CR fields (Phase 3.D)

Promotes two Phase 2 surfaces from chart-only / extraArgs-hatch into
typed CR fields so operator-managed installs (ArgoCD/Flux/kubectl apply)
don't need escape hatches.

- `AIMetricsSpec {Addr, Port}` — operator renders `--metrics-addr` arg +
  named container port + headless Service. Selectors target aiwatch pods
  so Prometheus pod-discovery sees per-pod endpoints (leader vs follower
  stay distinct in `cha_cycle_total{leader=...}`).
- `AILLMProposerSpec {Enabled}` — typed switch for the Phase 2.D LLM
  fallback proposer.

CRD schema additions on both chart-side template and OLM bundle manifest.
3 helm-template invariants preserved: legacy installs (no Metrics / no
LLMProposer fields) render byte-identical to v1.21.1.

### Pairs with CHA-com

The CHA-com binary `--metrics-addr` + `--llm-proposer` flags ship since
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
When set, the chart mounts the Secret at `/etc/cha/attestation/` and
passes `--digest-pin-attestation-key` + `--digest-pin-attestation-kid`
to the aiwatch container. Operator reconciler mirrors the chart.
Mount path is separate from `/etc/cha/keys/` so attestation key
rotation is independent of the approval-server signing key.

### Fixed — `internal/report.DeltaDiag` class-URL docs (Phase 2.B.6)

The render-only class-URL fields shipped in v1.21.0 — `ApproveClassURL`,
`DenyClassURL`, `SilenceClassURL` — now carry a doc clarifying that
the OSS enrich pipeline does NOT mint class-action JWTs (the signer
lives in CHA-com's `ai/approval`). The CHA-com aiwatch's renderer
(`cmd/cha-com/render.go`) is the active surface; the OSS render is
preparatory for a future shared-signer extraction.

### Pairs with CHA-com

`v1.16.0+` (binary-side surfaces are unchanged from v1.21.0 →
v1.21.1; only the chart's wiring of an existing CHA-com flag is new).

## [1.21.0] — 2026-06-08

Phase 2 closure on the OSS side. Pairs with CHA-com `v1.16.0`
for the paid-tier binary half.

### Added — `spec.ai.replicas` for HA aiwatch (Phase 2.F)

`ClusterHealthAutopilot.spec.ai.replicas` (`int32`, min 1 max 5).
Default 1 (single-replica, noop elector — byte-identical to pre-2.F).
When `>1`, the chart turns on `--leader-election=true` + binds the
SA to a scoped Lease Role; the binary races for a
`coordination.k8s.io/v1.Lease` named `<release>-aiwatch-leader`.
Failover within ~30s on lease loss.

### Added — Prometheus instrumentation + Grafana dashboard + canary alerts (Phase 2.G)

`ai.metrics.{addr,port,serviceMonitor,grafanaDashboard,prometheusRule}`
values opt in to: aiwatch `/metrics:9090` headless Service +
optional `ServiceMonitor` + `dashboards/cha-overview.json` ConfigMap
(Grafana sidecar labels) + `PrometheusRule` canaries
(`ChaWatcherStuck`, `ChaBreakerOpen`, `ChaAutonomyRejectionSpike`).

All gated on `ai.enabled` + non-empty `ai.metrics.addr` — pure-OSS
deploys see no new resources.

### Added — Slack class-button render row (Phase 2.B.6)

`internal/report.DeltaDiag` gains `ApproveClassURL` /
`DenyClassURL` / `SilenceClassURL`. When populated, `FormatSlackDelta`
renders an extra row under the Approve/Deny pair. Render-only on
OSS — the OSS enrich pipeline does NOT yet mint class-action JWTs
(the signer lives in CHA-com). CHA-com aiwatch's renderer
(`cmd/cha-com/render.go`) is the active surface in production.

### Added — `Silence.spec.matcher.messagePattern` (Phase 2.B.9)

Substring-match on `Diagnostic.Message`. Enables class-scoped
silences from the CHA-com `/silence-class` click. `pkg/silence.Matches`
ANDs MessagePattern alongside Source + Subject + Severity.

### Added — `DisruptionDrift` analyzer (Phase 2.E)

Three new signals: **PDB blocks all evictions** (`critical`),
**stuck Indexed Job failed indexes** (`warning`), **stale
ResourceQuota at 100%** (`warning`). Opts out via
`CHA_ANALYZER_DISRUPTION_DRIFT=off`.

### Added — `spec.ai.digestPinAttestation` chart wiring (Phase 2.H)

`DigestPinAttestationSpec {SecretName, SecretKey, KeyID}` on AISpec.
When set, chart mounts the Secret at `/etc/cha/attestation/` and
passes `--digest-pin-attestation-key` + `--digest-pin-attestation-kid`
to the aiwatch container. Operator reconciler mirrors the chart.
Mount path is separate from `/etc/cha/keys/` so attestation key
rotation is independent of the approval-server signing key.

### Pairs with CHA-com

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

Chart values block used nested `ticketing.openproject.{mcpURL, projectID, …}`; CRD uses flat `ticketing.{mcpURL, project, …}`. Users could not move YAML between `helm upgrade -f values.yaml` and `kubectl patch cha …` without reshaping. Fixed by flattening the chart shape to mirror the CRD exactly; legacy nested form honored as a fallback (will be removed in the next major chart bump).

## [1.20.0] — 2026-06-07

### Added — Operator-managed `spec.ticketing` (Phase 1.D, PR #167)

Closes the Helm-values-vs-operator-managed-CR gap for issue-tracker delivery. Before this release, the chart had full `ticketing.*` Helm values + the `cmd/cha` flags + the `pkg/ticketing/openproject.Sink` implementation, but the operator's `BuildWatcherDeployment` never emitted any `--ticketing-*` flag and the CRD had no `spec.ticketing` field. Operators who set Helm values saw no effect on operator-managed installs.

New `spec.ticketing` (TicketingSpec) on the CR drives flags + env injection:

  - `--ticketing-provider` ← `spec.ticketing.provider` (openproject / jira / servicenow enum)
  - `--ticketing-mcp-url` ← `spec.ticketing.mcpURL`
  - `--ticketing-project` / `--ticketing-type-id` / `--ticketing-closed-status-id` ← matching CR fields
  - `--ticketing-priority-{critical,warning,info}` ← `spec.ticketing.severityPriority.*`
  - `--ticketing-web-url-prefix` ← `spec.ticketing.webURLPrefix`
  - `--ticketing-labels=<L>` (one flag per label) ← `spec.ticketing.labels[]`
  - `--ticketing-dry-run` ← `spec.ticketing.dryRun: true`
  - `TICKETING_MCP_API_KEY` env via secretKeyRef ← `spec.ticketing.auth.{secretName,secretKey}` (only when `auth.enabled`)

Schema is OpenProject-shaped (OSS's only supported provider); the enum allows jira/servicenow so CHA-com can add them additively without a v1alpha2 bump.

Disabled (the default) emits zero flags so existing CRs stay byte-identical.

## [1.19.0] — 2026-06-05

### Added — Per-cycle delta render: "🆕 New this cycle" + stable-collapse + opt-in no-change digest (PR #165)

Operators reading #ceph-critical can't tell at-a-glance which findings just appeared vs. which are stale-but-getting-reposted. With 50+ findings per cycle the "what should I look at right now" signal drowns. Three signal:noise improvements layered together:

- **"🆕 New this cycle (N):" section** renders ABOVE the legacy critical/diagnostics list. `watcher.diff()` marks `entry.isNewThisCycle=true` on new-subject + fp-changed paths, false on repeat-interval re-posts. Zero-count section headers are suppressed (no `(0)` clutter).
- **Stable-collapse** (always on when new findings exist): the steady-state list collapses to a single `"…and N other stable finding(s) already posted in earlier cycles"` line. Cycles with 0 new findings render stable in full (operator is reading the periodic re-post — every finding matters).
- **"✨ No new issues" digest** (opt-in via `--slack-no-change-digest=true`): on cycles where `newCount == 0 && resolved == 0 && stable > 0`, replaces the full re-post with a compact steady-state confirmation. Default OFF to preserve byte-identical legacy behaviour.

Wiring chain: `cmd/cha --slack-no-change-digest` → `watcher.Config.NoChangeSlackDigest` → `report.RouteAndPostConfig` → `report.SplitCriticalPayloadsConfig` → `report.emitNoChangeDigest`. Legacy entry points (`SplitCriticalPayloads`, `RouteAndPost`) preserved as zero-config delegates.

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

- **The workload feeder is now importable from external Go modules** (the paid cha-com binary in particular). Go's `internal/` visibility rule was blocking the cha-com aiwatch from instantiating `WorkloadFeeder` — meaning `kind=workload` entries were never being written to RAG, meaning the v1.11.0 cha-com `DigestPinProposer` would always miss its RAG lookup, meaning **no Approve/Deny buttons would have appeared on digest-pin findings even after the cluster rolled to v1.11.0**.
- **Mechanical move**: `git mv internal/feeder pkg/feeder`. The 4 Kubernetes GVRs (`Pod`, `Deployment`, `StatefulSet`, `DaemonSet`) the feeder needs are now defined locally in `pkg/feeder/workload.go` since `pkg/snapshot` doesn't carry them and `pkg/` cannot import `internal/snapshot`. No logic changes.
- All 13 existing feeder tests still pass.


### Added — `spec.ai.extraArgs` + `spec.ai.extraEnv` escape hatches on the operator (v1.18.0)

- **`api/v1alpha1/clusterhealthautopilot_types.go`** — new `AISpec.ExtraArgs []string` + `AISpec.ExtraEnv []AIExtraEnv` (with `AIExtraEnvSource` + `AIExtraEnvSecretKeyRef`).
- **Why**: cha-com v1.11.0 ships new flags (`--cloudflare-feeder`, `--rag-store-url`, `--cluster-name`, `--digest-pin-proposer`, `--forge-token-env`, `--digest-pin-repo-map`) and env vars (`GITHUB_PAT`, `CLOUDFLARE_API_TOKEN`) that the operator's typed schema doesn't yet model. The escape hatches let operators wire them today via the existing CR-patch flow while typed fields land in subsequent minor releases.
- **`internal/operator/builders.go::aiArgs`** — appends `ai.ExtraArgs` AFTER the typed args so a typed flag wins on duplicate keys (later args override earlier in pflag).
- **`internal/operator/builders.go::aiEnv`** — appends `ai.ExtraEnv` entries as `corev1.EnvVar`, supporting either literal `Value` or `ValueFrom.SecretKeyRef`. ConfigMapKeyRef / FieldRef / ResourceFieldRef are deliberately out of scope (aiwatch never needs them).
- **CRD schema updated** — both the chart-managed `crd-clusterhealthautopilot.yaml` and the bundled `bundle/manifests/cha.bionicaisolutions.com_clusterhealthautopilots.yaml` accept the new fields with kubebuilder validators (`minLength=1` on `name`/`key`).
- **3 new operator builder tests** — `ExtraArgs_AppendedAfterTypedArgs` (order check), `ExtraEnv_SecretRefAppended` (both `ValueFrom.SecretKeyRef` + literal `Value` paths), `ExtraArgsEmpty_NoChange` (defensive baseline).

### Added — `ActionProposePullRequest` ActionKind (Phase 2d-γ-3 slice 3a)

- **`pkg/ai/types.go`** — new `ActionProposePullRequest ActionKind = "ProposePullRequest"` for proposals that carry a forge PR URL instead of a cluster-side mutation. The cluster itself is NOT changed when the proposal is approved; only when the PR is merged + the next normal deploy runs.
- **`AIProposedAction.PullRequestURL string`** — new field holding the forge URL the proposer already opened (the digest-pin proposer in slice 3b will populate it via the cha-com forge client).
- **`Validate()` rules for the new kind** —
  - `PullRequestURL` non-empty (rejects with `ErrPullRequestURLEmpty`)
  - URL must parse as a well-formed HTTPS URL with a non-empty host (`ErrPullRequestURLInvalid`) — guards against an `http://` downgrade or a `https:///path` malformed link rendering as a phishing target in Slack
  - `Target.Namespace` still subject to the protected-NS check — CHA never proposes PRs that would mutate `kube-system` / `vault` / `cnpg-system` infra
  - `Rollback.Description` still required (PR rollback = "close PR + delete branch")
  - Tier still must be `T1`/`T2`/`T3` (T0 = narration-only)
  - `ManifestYAML` and `PatchPayload` MUST be empty on a `ProposePullRequest` (else `ErrInvalidActionKind` — proposer can't smuggle a cluster mutation through this kind)
- **Self-hosted forges supported** — no OSS-side host allowlist; operators run self-hosted GitLab / Gitea / Forgejo with arbitrary hostnames. Allowlist enforcement (if needed for a specific deployment) belongs in the approval-server's per-CR policy layer in a future slice.
- **12 test cases** (`pkg/ai/propose_pull_request_test.go`) — happy path, empty/whitespace URL, http downgrade, missing host, garbled URL, self-hosted-GitLab accepted, protected-namespace rejection, missing rollback, wrong-kind URL field, T0-tier rejection, ManifestYAML-on-PR-kind rejection.
- Not yet wired into an executor — `pkg/ai` types only. cha-com slice 3b/3c lands the approval-server executor handler (Approve → post-merge comment / auto-merge per CR policy; Deny → close PR + record outcome to RAG) plus the `DigestPinProposer` that emits proposals of this kind.

### Added — Release-source detection (`pkg/releasesrc`, Phase 2d-γ-3 slice 1)

- **`pkg/releasesrc`** — new public package finds the file + line in a release-source repo where a workload's image tag is declared. Keystone for the paid-tier digest-pin proposer: without knowing which `values.yaml` line holds `image.tag`, the proposer can't construct a one-line patch.
- **`DetectInHelmValues(ctx, files, chartName, expectRepository) → *ImageRef`** — probes `charts/<chartName>/values.yaml` (umbrella layout) then `values.yaml` (single-chart root). Decodes the conventional `image: {repository, tag}` shape via `sigs.k8s.io/yaml`, requires `image.repository` to match `expectRepository` (guards against false matches in umbrella charts that ship multiple subchart blocks). Returns `ErrNotFound` cleanly when nothing matches; transport errors propagate unchanged.
- **`ImageRef{File, Line, KeyPath, CurrentTag, Repository}`** — `Line` is 1-based for editor/`git blame` parity. `KeyPath` is a dot-separated YAML walk (today: always `"image.tag"`). Line lookup uses a regex anchor (`image:` header → first `tag:` line below it) because `sigs.k8s.io/yaml` doesn't preserve positions.
- **`RepoFiles` interface** — minimal `Get(path) → bytes` + `List(patterns) → []string`. Defined in OSS so cha-com's forge client can implement it via a per-`(owner, repo, ref)` adapter without OSS taking a forge dependency.
- **Security defenses** — chart-name input is sanitized to `path.Base()` so a hostile `"../../etc"` chart name can't escape the chart dir. Empty `expectRepository` is rejected (would match every image block). Garbled YAML in one candidate file doesn't abort the probe — falls through to the next path.
- **13 test cases** — happy-path umbrella + happy-path root layouts, repo-mismatch silently skipped, missing tag returns NotFound, garbled YAML doesn't crash, all-paths-missing returns NotFound, true transport error propagates, nil-files / empty-expectRepository guards, empty chart name falls back to root only, path-traversal sanitization, exact line-number calculation, unquoted-numeric-tag handling.
- **Not yet wired** into any proposer — pure library slice. Foundation for **slice 2** (cha-com Forge → RepoFiles adapter + DigestPinProposer that consumes the v1.16.0+ workload feeder's `kind=workload` entries + this detector → forge.CreatePullRequest → Slack Approve/Deny buttons on every digest-pin warning). Argo CD Application + Kustomize + Flux HelmRelease detectors will join in follow-up slices.

### Added — Workload feeder (Phase 2d-γ-2, RAG foundation slice)

- **`internal/feeder/workload.go`** — new `WorkloadFeeder` walks Deployments / StatefulSets / DaemonSets each cycle and upserts one `rag.Entry{Kind: KindWorkload}` per workload. Features captured: `kind` (controller type), `namespace`, `name`, `replicas`, `containers: [{name, image, image_digest}, ...]`, and best-effort `owner_kind`/`owner_release`/`owner_release_namespace`/`owner_chart` derived from the conventional Helm + Argo CD annotations.
- **Digest resolution** — `image_digest` is read from the owning Pod's `status.containerStatuses[].imageID` (kubelet writes the resolved `sha256:` after a successful pull). Pods that haven't pulled yet (ImagePullBackOff, pending) contribute nothing, and that container's `image_digest` is simply omitted — the correct signal for a downstream proposer to skip the cycle and retry next time. Extraction tolerates `docker.io/`, `docker-pullable://`, and private-registry imageID formats.
- **Owner detection** — reads `meta.helm.sh/release-name` + `meta.helm.sh/release-namespace` for Helm-managed workloads; `argocd.argoproj.io/instance` for Argo CD (`<namespace>_<name>` form). The `helm.sh/chart` label is parsed to extract the chart name with the trailing version stripped. Empty when neither annotation is set — the proposer slice will fall back to a PR-template path.
- **System-namespace skip list** — `kube-system`, `kube-public`, `kube-node-lease`, `cnpg-system`, `rook-ceph`, `vault`, `external-secrets`, `calico-system`, `tigera-operator`, `calico-apiserver`, `local-path-storage`. Matches the digest-pin analyzer's system-namespace set so feeder and analyzer agree on "is this workload SRE-relevant".
- **Fail-open everywhere** — nil receiver / missing Source / missing Writer errors at the contract boundary; per-workload parse + Upsert failures are silently skipped so one bad workload can't stall the sweep. Mirrors the cha-com `CloudflareFeeder` discipline.
- **13 test cases** — happy path with digest, no-pod-no-digest, three-controller-kinds sweep, system-namespace skip, Helm annotations populate owner, Argo CD annotation parses `<ns>_<name>` form, no-annotations omits owner, multi-container with partial digests, degenerate empty workload skipped, writer error doesn't abort sweep, digest extraction across 5 imageID formats, default importance fallback, nil-guards table.
- **Not yet wired** into `cha watch` — pure library slice. Next slice activates it via `cfg.RAGWriter rag.Writer` + a `--workload-feeder` flag on cmd/cha + an operator `spec.feeder.workload.enabled` knob on the CR. Foundation for Phase 2d-γ-3 (release-source detection enrichment) and Phase 2d-γ-4 (digest-pin proposer that consumes these entries).

### Added — Watcher mints approve/deny URLs directly (Path B)

- **`pkg/ai/manifest_bridge.go`** — new public `ManifestBridge` (implements `FixProposer`) that converts `Diagnostic.ProposedPolicyYAML` into a signed `ApplyManifest` `AIProposedAction` via the existing safe-apply validator (closed Kind whitelist + per-Kind shape; NetworkPolicy is the v1.15.0 entry). Refusal classes — egress in `policyTypes`, unsupported Kind, protected namespace, non-yaml — quietly return `nil` (no URL minted on dangerous YAML).
- **`pkg/ai/signer.go`** — Ed25519 signer ported from cha-com (was proprietary, now Apache-2.0). Disk-backed (base64 raw bytes), trailing-whitespace tolerant, env-var fallback (`CHA_SIGNING_KEY_PATH`), `ErrSigningKeyMissing` sentinel for graceful fall-through. `GenerateAndPersistSigningKey()` for bootstrap.
- **`cmd/cha/main.go`** — `cha watch` gains `--approval-server-url` + `--signing-key-path` flags. When both resolve, loads signer + registers `ManifestBridge` as fallback `FixProposer` (only when registry has no programmatically-registered proposer — keeps cha-com's LLM-backed proposer primary). Wires `Config.ApprovalBaseURL` so `enrichDiagnostics` mints URLs in the existing T1 path.
- **`internal/operator/builders.go`** — `BuildWatcherDeployment` passes the new flags + mounts the signing-key Secret when both `cr.Spec.AI.ApprovalServerURL` AND `cr.Spec.Approval.SigningKey.SecretName` are set. Guards against half-configured installs (no key → no flags → no broken pod).
- **Closes the architectural gap** where ProposedPolicyYAML-bearing diagnostics (NetworkPolicyProposer) had URLs minted in the cha-com aiwatch process but NEVER reached the user-facing Slack / Alertmanager / OpenProject surfaces — those are written by the OSS watcher, which had no URL-minting capability. After this change the OSS watcher mints URLs itself; they flow through the existing `d.ApprovalURL` field every adapter already renders.
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

### Added — typed `AISpec` on `ClusterHealthAutopilot` v1alpha1 (operator Phase-2 schema slice)

- `api/v1alpha1`: new `AISpec` + `AIAPIKeySpec` + `AIT3Spec` + `AIMemorySpec` + `AIMemoryStorageSpec` + `AIEmbeddingsSpec` types mirroring the chart's `ai.*` helm values surface. Exposed as `ClusterHealthAutopilotSpec.AI` (optional). DeepCopy methods hand-written matching the Phase-1 pattern.
- CRD YAML extended to accept `spec.ai` so the apiserver validates the schema. Tier uses `kubebuilder:validation:Enum=off;t0;t1;t2;t3`.
- **Controller does NOT yet consume these fields.** The reconciler still relies on the chart's `ai.*` helm values for the running aiwatch + approval-server. Schema lands now so operator-managed manifests are forward-compatible with the Phase 2 reconciler wiring; the misleading comment that previously claimed the fields were "opaque pass-throughs" is corrected.

### Fixed

- Stale package docs in `pkg/cloud/gcp/client.go` and `pkg/cloud/azure/client.go` that claimed "Live wrapper deferred to a follow-up PR" — both Live wrappers shipped (v1.7 baseline; v1.9 Cloud Monitoring / Azure Monitor / BackendHealth LRO additions in PRs #103–#106). Comments now reflect what's on `main`.

### Added — Operator Phase 1c (slice B) — OLM bundle scaffolding

- New `bundle/` directory and `bundle.Dockerfile` carrying the OLM ClusterServiceVersion + the four shipped CRDs (`ClusterHealthAutopilot`, `Silence`, `ResolutionRecord`, `DriftReport`). The CSV's `install.spec` mirrors `templates/operator-deployment.yaml` (image, args, env, ports, probes, securityContext) so `operator-sdk run bundle <image>` produces structurally the same operator as `helm install`.
- `installModes`: `OwnNamespace` + `SingleNamespace` + `AllNamespaces` (true); `MultiNamespace` (false) — explicit because the reconciler scope decisions in Phase 1b watch all-namespaces.
- New parity-guard tests in `internal/operator/bundle_parity_test.go`: (1) CSV is valid YAML + kind=ClusterServiceVersion; (2) every CRD shipped in `bundle/manifests/` is declared under CSV `customresourcedefinitions.owned` (no orphan CRDs); (3) the chart's operator ClusterRole rules and the CSV's `clusterPermissions[0].rules` carry the same `(apiGroup, resource)` set (catches the common drift pattern where someone adds a rule to one file and forgets the other).
- **NOT in this slice**: CI bundle-smoke job using `operator-sdk run bundle` against kind — Phase 1c slice C, separate PR.

### Added — Operator Phase 1c (slice A) — operator-provisioned reader RBAC

- `api/v1alpha1`: new `ConditionReaderRBACReady` condition + `FinalizerOperatorRBAC` (`cha.bionicaisolutions.com/operator-rbac`) — cluster-scoped resources can't carry namespaced ownerRefs, so the finalizer drives cleanup.
- `internal/operator/rbac_builders.go`: `BuildReaderClusterRole()` returns a shared cluster-scoped role mirroring `templates/clusterrole-reader.yaml`'s verb set. `BuildReaderClusterRoleBinding(cr)` returns a per-CR binding labeled `managed-by-cr` + `cr-namespace` for safe identification by the finalizer.
- `internal/operator/reconciler.go`: adds `reconcileReaderRBAC()` (idempotent CreateOrUpdate on the shared role + per-CR binding), `finalizeReaderRBAC()` (deletes ONLY bindings we labeled; defense-in-depth against name collisions), and finalizer add/remove paths on every reconcile. `ReaderRBACReady` is computed from observed state: ClusterRole present + binding present + subject targets the CR's SA. `Ready` is now `(no reconcile error) AND ReaderRBACReady`; `WatcherRunning` continues to track availability orthogonally.
- Chart: operator ClusterRole extended with cluster-wide CRUD on `rbac.authorization.k8s.io/{clusterroles,clusterrolebindings}`.
- 6 new reconciler tests with the controller-runtime fake client: creates-both / finalizer-add / on-delete-removes-binding-and-finalizer / shared-ClusterRole-survives-CR-delete / defense-in-depth-skips-unlabeled-bindings / ReaderRBACReady-True / WrongSubject-detected-and-corrected.
- Coexistence contract: operator-managed binding lands ALONGSIDE the chart-managed binding (RBAC unions across bindings; same SA + same role from two managers is harmless). Operators can run both side-by-side; chart binding stays helm-owned until `helm uninstall`.
- **NOT in this slice**: OLM bundle (Phase 1c slice B) + CI bundle-smoke (Phase 1c slice C). Each is a separate PR per `docs/design/2026-05-v1.9-operator-phase-1c.md`.

### Added — `Silence` CRD + watch-loop suppression

- New `Silence` CRD (`silences.cha.bionicaisolutions.com`, namespace-scoped, `sil` short name). Operators create a Silence to mute a known-benign-but-unfixable finding for a bounded window. Matcher fields: `source` / `subject` / `severity` (empty = wildcard); CRD validation rejects an entirely-empty matcher. `spec.until` is required; past expiry the Silence becomes a no-op but is NOT auto-deleted (audit trail). Optional `reason` + `createdBy` for "why is this muted?" answers.
- New pure `pkg/silence.Filter()` + `Matches()` — namespaced lookup, exact-field matching, expired-silence-never-matches guard. Order-preserving, doesn't mutate the caller's slice. 8 unit tests.
- New `pkg/silence.K8sLister` (dynamic-client backed) lists active Silences cluster-wide once per watcher cycle. Soft-fails on a missing CRD (returns nil, nil) so a chart < 1.9 install still works.
- Watcher integration: `Watcher.WithSilenceLister(lister)` — wired in `cmd/cha/main.go`. Silenced diagnostics are dropped in `runDiagnose()` BEFORE downstream emission (DriftReport / Slack / Alertmanager / ticketing), so a muted finding never re-pages.
- Chart: `templates/crd-silence.yaml` (gated on `silence.installCRD`, default ON) + `templates/clusterrole-silence.yaml` (cluster-wide list/watch on `silences`, gated on `silence.enabled`, default ON). Reserves `silences/status` write permission for a future matchCount/lastMatchAt updater.
- Closes post-v1.9 adversarial-review finding #2: previously CHA had only endpoint-probe flake debounce — no user-controlled per-fingerprint, time-bounded suppression. Now Silence is a first-class, K8s-native concept matching the Alertmanager-silences pattern.

(Reserve for v1.9+ — operator Phase 1c per `docs/design/2026-05-v1.9-operator-phase-1c.md`; Phase 2 reconciler consumption of the AISpec; remaining cloud Monitoring-API signals; trigger-classes C/E.)

---

## [1.8.12] — 2026-05-30

Chart wiring for the approval-server HA backend introduced in CHA-com PR #16.

### Added

- `approval.store.backend=configmap` (and `.namespace`, `.replayConfigMap`, `.runbookConfigMap`): when set, passes the matching `--store-*` flags to `cha-com approval-server`, switches the Deployment to `RollingUpdate`, and provisions a per-namespace Role + RoleBinding granting the approval-server SA `get/update/create` on the named ConfigMaps (minimum-privilege: NOT granted in the default in-memory posture). With this set + `approval.replicas > 1`, the approval-server is HA-safe (a JTI used on replica A cannot be replayed on B; T3 dual-approval state cannot fork).

---

## [1.8.11] — 2026-05-30

Chart-only fix: the RAG Qdrant StatefulSet (added in 1.8.9) CrashLooped on first deploy because `securityContext.readOnlyRootFilesystem: true` made Qdrant's default snapshots/temp paths unwritable. Redirected both under the mounted storage PVC via `QDRANT__STORAGE__SNAPSHOTS_PATH` and `QDRANT__STORAGE__TEMP_PATH` env vars — single volume now serves all writes.

### Fixed

- RAG Qdrant snapshots + temp paths point inside the storage PVC (was: read-only root FS → CrashLoopBackOff `"Can't create Snapshots directory: ReadOnlyFilesystem"`).

---

## [1.8.10] — 2026-05-29

P2/G5c chart wiring — connects the deployed aiwatch to the RAG store.

### Added

- When `ai.memory.enabled`, `cha.aiArgs` now passes `--memory-store-url` (defaults to the in-namespace Qdrant service `http://<release>-rag.<ns>.svc:6333`), `--memory-embeddings-endpoint` (defaults to `ai.endpoint`), `--memory-embeddings-model` (required), and `--memory-topk` to the aiwatch. With this, the commercial binary's RAG grounding (CHA-com G5c retrieve half) is reachable end-to-end; off by default.

---

## [1.8.9] — 2026-05-29

P1/G4 foundation for the AI-remediation memory loop. Chart-only effect (new CRD + RBAC); the recorder library is dormant until the AI write-path wires it (P2/G5c).

### Added — ResolutionRecord CRD + recorder

- **`ResolutionRecord` CRD** (`resolutionrecords.cha.bionicaisolutions.com`, cluster-scoped, `rr` short name) — the append-only outcome log: one CR per applied+verified (or rejected/reverted) remediation, capturing `{fingerprint, source, subjectKind, diagnosticDigest, proposal{actionKind,target,rationale,rollback}, delivery, applied, verified}`. This is the durable system-of-record the dedicated RAG memory layer (1.8.8 Qdrant) embeds + retrieves.
- **`internal/resolution` recorder** — `Recorder.Record()` appends a CR through the snapshot.Mutator (nil-safe / no-op in dry-run); stable `Fingerprint(source, subject)` join key to DriftReport.
- **ResolutionRecord ClusterRole** — create/get/list/watch + status patch (for the RAG layer's `embeddedAt`/`vectorId` stamp), bound to the chart SA. Append-only (no delete verb).

---

## [1.8.8] — 2026-05-29

P2/G5a — the dedicated RAG vector store deployment (chart-only; off by default).

### Added — in-namespace Qdrant RAG

- **`ai.memory.enabled`** stands up a dedicated **Qdrant** vector store (StatefulSet + PVC + ClusterIP Service) in the install namespace, alongside the other CHA objects. The aiwatch (P2/G5b–c, CHA-com) embeds ResolutionRecords via the in-cluster gpu-ai embeddings endpoint and retrieves prior resolutions to ground T1–T3 proposals. The ResolutionRecord CRD (1.8.9) is the system-of-record; Qdrant is the rebuildable semantic index over it.
- New `ai.memory.*` values: `image`, `storage.{size,className}`, `resources`, `embeddings.{endpoint,model}`, `storeUrl`, `topK`. Off by default and independent of `ai.enabled` so it can be rolled out separately.

---

## [1.8.6] — 2026-05-29

P0 signal-hygiene from the AI-remediation plan (`docs/design/` in CHA-com), plus the chart arg that activates commercial click-to-fix delivery.

### Fixed — false-positive criticals (alert-fatigue de-noising)

- **HPAScaling: `ScalingActive=False` / `reason=ScalingDisabled` is now Warning, not Critical.** That condition is the *expected* state when an HPA's target is intentionally at zero (KEDA scale-to-zero, or a Deployment scaled to 0) — the autoscaler simply goes dormant. Flagging it CRITICAL was a false positive that trained operators to ignore the board. Every other reason (`FailedGetScale`, `FailedGetResourceMetric`, quota/PDB blocks) stays CRITICAL.
- **Endpoint discovery skips `cm-acme-http-solver-*` Ingresses.** cert-manager spawns these transient HTTP-01 challenge solvers and deletes them on completion; probing them produced churning false-criticals and ticket spam for hosts that aren't real services.

### Added

- **`ai.approvalServerUrl`** chart value → `--approval-server-url` arg on the aiwatch (via `cha.aiArgs`). When set (with `approval.enabled`), the commercial CHA-com binary emits signed one-click click-to-fix links for T1/T2 proposals.

---

## [1.8.5] — 2026-05-28

Chart-only fix found while enabling the paid tier on a live cluster.

### Fixed — approval-server keygen-job image tag

The `approval-server-keygen-job` (a Helm pre-install/upgrade hook that mints the Ed25519 signing key) still defaulted its image tag to `.Chart.AppVersion` (e.g. `1.8.2`), but cha-com images are tagged with a leading `v` (`v1.8.2`). On a fresh paid enable the keygen hook hit `ImagePullBackOff` and stalled the whole `helm upgrade` in `pending-upgrade`. Now uses the same `v<AppVersion>` default as the approval-server Deployment (fixed in 1.8.4). Without this, enabling `approval.enabled=true` required a manual `approval.image.tag` override.

---

## [1.8.4] — 2026-05-28

Corrects the AI-tier deployment model shipped in 1.8.3. No Go changes.

### Fixed — AI tier deploys as an additive companion, not an OSS-binary flag-swap

1.8.3 folded the `--ai-*` flags into the **OSS** watcher Deployment and diagnose CronJob args, on the assumption that the commercial binary is a flag-superset of OSS. It is not: `cha-com watch` / `cha-com diagnose` are the **AI-layered counterparts** with a deliberately reduced flag surface — they reject the OSS operational flags (`--live`, `--write-driftreports`, `--slack-*`, `--remedy`, `--ticketing-*`, `--cloud-*`). Enabling 1.8.3's wiring + the cha-com image would have crash-looped the watcher on an unknown flag. (1.8.3 was gated on `ai.enabled=true`, default false, so no OSS or default install was affected.)

The corrected, **purely additive** model:

- The OSS watcher Deployment + diagnose / remediate CronJobs are **never swapped or modified** — they keep the OSS image and own the event-driven probe → fix → Slack / ticketing / DriftReport pipeline.
- Setting `ai.enabled=true` stands up a **separate `aiwatch` Deployment** (new `templates/aiwatch-deployment.yaml`) running `cha-com watch` — the AI-layered counterpart that polls the same merged catalog on `ai.interval` and fires the AI tiers (t0→t3) against new diagnostics, signing click-to-fix URLs for the approval-server at t1+.
- `cha.aiArgs` now emits exactly the `cha-com watch` flag surface (incl. `--interval`); `cha.aiImage` resolves the commercial image (`docker4zerocool/cha-com:v<AppVersion>`).
- New `ai.image.*`, `ai.interval`, `ai.resources` values. The aiwatch pod reuses the chart's read-only reader SA (it only reads + proposes; never mutates).
- Fixed the approval-server image-tag default to the `v`-prefixed cha-com convention (`v<AppVersion>`), which previously resolved to a non-existent tag.

**Go-to-market:** OSS install + the single flag `ai.enabled=true` = the paid tier. No image-swap-and-pray. Full setup in `docs/DEPLOYMENT.md`.

---

## [1.8.3] — 2026-05-28

Chart-only release that completes the AI-tier (commercial CHA-com) deployment path. No Go changes.

### Added — AI-tier flag wiring in the chart

The chart now renders the commercial `--ai-*` flag surface onto the watcher Deployment and diagnose CronJob when `ai.enabled=true`, via two new nil-safe helpers (`cha.aiArgs`, `cha.aiEnv`). Previously the `ai:` values block existed but was consumed by no template, so the paid tier could not actually be turned on through Helm.

- **`cha.aiArgs`** emits `--ai-tier`, `--ai-endpoint`, `--ai-model`, `--ai-api-key-header`, `--ai-allow-saas`, `--ai-llm-fixer-matcher`, `--ai-audit-log`, and (for t3) repeatable `--t3-vault-allowed-prefix`. `ai.endpoint` and `ai.model` are `required` when enabled. The OSS `cha` binary does not understand these flags, so the block is inert unless `image.repository` points at `docker4zerocool/cha-com`.
- **`cha.aiEnv`** injects the LLM bearer token into the env var the binary reads (`ai.apiKey.envName`, default `AI_API_KEY`) via `secretKeyRef` — never inlined, ESO-managed.
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

When a `ClusterHealthAutopilot` CR pins `spec.serviceAccountName` (the supported path for giving an operator-managed watcher the probe RBAC it needs — point it at the chart's reader-bound SA), the reconciler used to still create+own that SA, grafting an owner-ref onto a pre-existing object and garbage-collecting it on CR deletion. The reconciler now skips SA creation entirely when `spec.serviceAccountName` is set.

**Known limitation (tracked for Phase 1c):** the operator does not yet provision its own reader `ClusterRoleBinding`, so an operator-managed watcher gets probe RBAC **only** via the BYO-SA path above. Documented on the CRD field and in the operator design doc.

### Added — M2 probe-class Helm toggles (roadmap AC parity)

`probes.{kong,hpaScaling,argocdApp,velero}.enabled` now exist in `values.yaml` and emit `CHA_PROBE_*=off` via the new `cha.probeToggleEnv` helper (mirrors `cha.analyzerToggleEnv`). Closes the v1.8 acceptance criterion that promised per-probe Helm values; the probes were previously gated only by env opt-out + CRD auto-skip. All default ON (auto-skip when the CRD is absent).

### Changed

- Cleared stale "not shipped yet" / "M1 follow-up" / "Azure remains a stub" comments in `values.yaml`, `cmd/cha/main.go`, and `catalog/cloud.go` — all three cloud providers and the M2 probe set shipped in v1.8.

---

## [1.8.1] — 2026-05-28

Patch release fixing two issues found while deploying v1.8.0 to a live cluster. Both are chart-only; no Go changes.

### Fixed

- **Diagnose / remediate `activeDeadlineSeconds` raised 120 → 300.** The v1.8 analyzer + M2-probe set adds a meaningful number of cluster List calls (CRDs, HPAs, all namespaces + pods + NetworkPolicies, Kong / Velero / Argo CRDs). A live diagnose on a busy cluster (~80 KongPlugins, ~250 drift records) was measured at ~157s — past the old 120s cap, so the CronJob was killed mid-run with `DeadlineExceeded`. 300s gives headroom while still failing fast on a genuinely hung cluster API.
- **Operator templates made nil-safe for `--reuse-values` upgrades.** `operator-deployment.yaml` (`.Values.operator.enabled`) and `crd-clusterhealthautopilot.yaml` (`.Values.operator.installCRD`) dereferenced `.Values.operator` directly, so a `helm upgrade --reuse-values` from a pre-v1.8 install (whose stored values predate the `operator:` block) hit `nil pointer evaluating interface {}.enabled`. Now guarded with `(.Values.operator).enabled` / `(.Values.operator).installCRD`, matching the existing `(.Values.analyzers).*` pattern.

### Verified on live cluster

v1.8.0 deployed to the dev cluster (helm rev 23) and a live diagnose confirmed the new probes fire against real resources: **Kong** (80 KongPlugins inspected), **HPAScaling** (flagged 3 real scaling-disabled HPAs), **ArgoCD-Application**, **Velero**, and **SecurityDrift** (PSS / image-pin / NetworkPolicy gaps in the `kong` namespace). 255 DriftReports reconciled.

---

## [1.8.0] — 2026-05-28

Drift-class completion + operator port + full multi-cloud release. Closes the bulk of `docs/design/2026-05-v1.8-roadmap.md`: the remaining drift classes (config / capacity / security), the controller-runtime operator port (Phase 1 + 1b), the M2 K8s probe slice (Kong / HPA / ArgoCD / Velero), and a complete 30-probe multi-cloud surface (10 each AWS / GCP / Azure) with all three Live SDK wrappers wired so the probes execute against real clouds.

### Added — Azure cloud-probe Live SDK wrapper (probes now execute against real Azure) — all 3 clouds live

- **`internal/cloud/azure/live.go`** — `LiveClient` implements all 10 `pkgazure.Client` methods against `azure-sdk-for-go` (armsql, armcompute, armcontainerservice, armmsi, armnetwork, armappservice, armstorage, armkeyvault, armauthorization). Auth via `DefaultAzureCredential` (AAD Workload Identity in-cluster, `az login` locally). Read-only. Resolves server→database, vnet→subnet, and cluster→nodepool nesting; extracts resource group from ARM IDs; counts role assignments per managed-identity principal.
- **`cmd/cha buildCloudSource()`** — `--cloud-azure-enabled` now constructs the live client (requires `--cloud-azure-subscription-id`; optional `--cloud-azure-location`) instead of erroring. **With this, all three providers (AWS, GCP, Azure) execute against real clouds.**
- Two documented limitations populated conservatively (no false-positives): VNet subnet free-IP (Network API exposes none → CIDR-derived total, available=total) and App Gateway backend health (per-gateway LRO too heavy per cycle → reports pool size as healthy). Both have Monitoring/LRO follow-ups noted in code.
- **Verification boundary:** compiles cleanly against the real `azure-sdk-for-go` ARM modules (API-surface correctness), but **not** integration-tested against a live Azure subscription — needs credentials. Probe evaluation logic remains unit-tested against fakes.

### Added — GCP cloud-probe Live SDK wrapper (probes now execute against real GCP)

- **`internal/cloud/gcp/live.go`** — `LiveClient` implements all 10 `pkggcp.Client` methods against `google.golang.org/api` (sqladmin, compute, container, iam, cloudkms, storage). Auth via Application Default Credentials (GKE Workload Identity in-cluster). Read-only. Compiles against the real SDK surface.
- **`cmd/cha buildCloudSource()`** — `--cloud-gcp-enabled` now constructs the live client (requires `--cloud-gcp-project`; optional `--cloud-gcp-region`) instead of erroring. The GCP probes are no longer dormant — they run against a real project when enabled.
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
- Default ON; flip `analyzers.configDrift.enabled=false` to disable, or set `CHA_ANALYZER_CONFIG_DRIFT=off`. 16 unit tests.

### Added — Operator port Phase 1 (foundations)

- **`api/v1alpha1/`** — `ClusterHealthAutopilot` CRD types (Spec, Status, Conditions) with hand-written DeepCopy methods. Foundations only; the controller-runtime Reconcile loop, the manager binary, and the chart wiring for operator-managed installs all come in Phase 1b. See `docs/design/2026-05-v1.8-operator-phase-1.md` for the staged-release rationale.
- **`internal/operator/builders.go`** — pure-function builders that translate `ClusterHealthAutopilotSpec` → `*appsv1.Deployment` (watcher) and `*batchv1.CronJob` (diagnose, remediate). Mirror the existing chart's CLI argument format so an operator-managed install behaves identically to a Helm-managed install. 19 unit tests cover defaults, overrides, image-policy inference, pull-secret round-trip, and alerting-flag emission.
- **`charts/.../templates/crd-clusterhealthautopilot.yaml`** — CRD shipped via the chart, gated behind `operator.installCRD` (default `true`). Installing the CRD on a cluster without the operator binary is harmless: the resource is queryable state with no controller acting on it.

### Added — Workstream B5 (capacity drift)

- **`CapacityDrift`** analyzer (B5) — capacity-tier signals that the basic resource-health probes miss. Five signals across HPAs and PVCs, none requiring metrics-server (the metrics-dependent signals — pod request vs usage, PVC growth-trajectory — defer to a v1.8.x follow-up):
  - **HPA pinned at maxReplicas** — `status.currentReplicas == spec.maxReplicas` past the saturation grace (24h default), excluding `min==max` static configurations. Workload is chronically under-provisioned. Critical.
  - **HPA pinned at minReplicas, not load-driven** — current replicas held at `minReplicas` for > 30 days with `maxReplicas > minReplicas + 1`. HPA range is decorative; the workload could be a static Deployment. Warning.
  - **HPA AbleToScale=False** — `status.conditions[type=AbleToScale,status=False]` past grace (15-min default). Typically a ResourceQuota cap or PDB blocking the controller. Critical.
  - **HPA FailedGetResourceMetric** — `ScalingActive=False` with that reason. Metrics-server is missing or unreachable; the HPA can't decide. Warning. This is the v1.8 R1 risk-mitigation signal so operators notice without us depending on metrics-server.
  - **PVC volume-expansion stuck** — `FileSystemResizePending=True` past grace, OR `spec.resources.requests.storage > status.capacity.storage` past grace. Volume-expansion got requested but the CSI driver didn't complete it. Critical.
- Skips kube-system / kube-public / kube-node-lease.
- Reader ClusterRole extended with read on `autoscaling/horizontalpodautoscalers`; PVC reads already covered by the core probe rule.
- Default ON; flip `analyzers.capacityDrift.enabled=false` to disable, or set `CHA_ANALYZER_CAPACITY_DRIFT=off`. 17 unit tests.

### Added — Workstream B6 (security drift)

- **`SecurityDrift`** analyzer (B6) — three observational signals on security posture:
  - **PSS posture gap** — user namespaces with no `pod-security.kubernetes.io/enforce` label (admission applies the cluster-wide default, typically `privileged`), or with `enforce=privileged` explicitly (the most-permissive PSS profile). Warning.
  - **Mutable image tag** — Pods whose containers reference images by tag only (`<image>:<tag>`) rather than by digest (`<image>@sha256:<digest>`). Mutable tags break the image-attestation signature chain — the runtime image can be re-published behind the same tag. Warning. Skipped for `:latest` (other code paths already flag that).
  - **NetworkPolicy coverage gap** — user namespaces running pods with zero NetworkPolicies. K8s default networking is fully permissive without at least one policy. Warning per namespace.
- Skips kube-system / kube-public / kube-node-lease / cnpg-system / rook-ceph / vault / external-secrets — system namespaces whose security posture is controller-managed.
- Reader ClusterRole extended with `networking.k8s.io/networkpolicies`; namespaces already covered by the core probe rule.
- Default ON; flip `analyzers.securityDrift.enabled=false` to disable, or set `CHA_ANALYZER_SECURITY_DRIFT=off`. 16 unit tests.
- Out of scope for v1.8 (deferred to a v1.8.x follow-up): true PSS-downgrade detection (requires label history) and active Cosign / Notation signature verification (admission-time concern; CHA is observational).

### Added — Operator port Phase 1b (controller-runtime + Reconciler + manager binary)

- **`sigs.k8s.io/controller-runtime v0.24.1`** added — chosen for compatibility with the current `k8s.io v0.36` baseline (controller-runtime v0.21 had a `ResourceEventHandlerRegistration` interface mismatch with newer client-go).
- **`internal/operator/reconciler.go`** — `Reconciler` implementation. Reconcile flow: fetch CR → validate `spec.image.tag` → reconcile ServiceAccount + watcher Deployment + diagnose CronJob + remediate CronJob via createOrUpdate (delete-on-disable) → compute `Ready` and `WatcherRunning` conditions from observed Deployment state → patch status. Uses controller-runtime CreateOrUpdate rather than server-side-apply to keep the cutover boring (existing chart installs are not disturbed unless an operator explicitly creates a `ClusterHealthAutopilot` CR).
- **`cmd/cha-operator/main.go`** — manager binary: leader-election lease (`cha-operator.cha.bionicaisolutions.com`, namespace from downward-API `MY_POD_NAMESPACE`), `:8080` Prometheus metrics, `:8081` healthz/readyz probes, structured zap logging.
- **`api/v1alpha1/groupversion_info.go`** — `AddToScheme` wired via `runtime.NewSchemeBuilder` directly (sidesteps the deprecated `controller-runtime/pkg/scheme.Builder`).
- **`charts/.../templates/operator-deployment.yaml`** — operator Deployment + ServiceAccount + ClusterRole + ClusterRoleBinding. Gated behind `operator.enabled` (default `false`). Operator has the read+write+delete verbs on ServiceAccount / Deployment / CronJob in any namespace; status-subresource write on the CR; Lease verbs for leader-election; events create+patch for `kubectl describe`. SecurityContext: `runAsNonRoot`, `readOnlyRootFilesystem`, drops all capabilities.
- **`Dockerfile`** — second `go build` step compiles `/cha-operator` alongside `/usr/local/bin/cha`. Single image hosts both binaries; the operator Deployment overrides `command:` to invoke `/cha-operator` instead of the watcher.
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

> **Note on cloud-probe execution:** the GCP + Azure probe *detection logic* is complete and unit-tested (10 probes each, parity with AWS), but neither provider has a Live SDK wrapper yet (`internal/cloud/{gcp,azure}/live.go` absent; `cloud.google.com/go` / `azure-sdk-for-go` not in go.mod). `cmd/cha buildCloudSource()` still errors for `--cloud-gcp-enabled` / `--cloud-azure-enabled`. Until the Live wrappers land, only **AWS** cloud probes execute against a real cloud; GCP/Azure are dormant. The Live wrappers are the remaining v1.8 cloud item.

### Added — M2 K8s probes (Kong / HPA / ArgoCD / Velero)

Four new resource-event-driven probes from `docs/design/2026-05-trigger-expansion-roadmap.md` M2/M3 and v1.8 roadmap §A5. Each auto-skips when its CRD is absent (or no-ops on an empty list for HPA), so default-on is safe. Each is independently disablable via `CHA_PROBE_<NAME>=off`.

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
