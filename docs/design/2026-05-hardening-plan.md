# CHA Hardening Plan — TDD-driven  ✅ COMPLETED v1.6.0

**Status:** ✅ CLOSED 2026-05-25. All Sprints 0–4 shipped in OSS v1.6.0 (deployed to development cluster) and now on `origin/main` as of v1.6.2. Sprint 5 (operator port to controller-runtime) was intentionally deferred to v1.7+.
**Scope:** Closed 22/23 punch-list items from the 2026-05-22 adversarial review. The 23rd item — full M2–M7 trigger expansion — remains on the roadmap (see [2026-05-trigger-expansion-roadmap.md](2026-05-trigger-expansion-roadmap.md)).
**Owner:** Closed by Sprint 1–4 execution.
**Live verification:** Cluster running v1.6.2 with lease-based leader election active (lease transitions = 3+, renewing every 5s), all 12 K8s probes + 8 analyzers + 5 fixers loaded, OpenProject MCP ticketing wired, per-severity Slack repeat intervals applied.

Original draft below preserved as historical reference.

---

**Status:** Draft, 2026-05-22
**Scope:** Close the 23 punch-list items from the 2026-05-22 adversarial review
**Owner:** TBD
**Companion docs:** [2026-05-trigger-expansion-roadmap.md](2026-05-trigger-expansion-roadmap.md) (parallel track; this plan is hardening, that one is feature expansion)

This plan is **hardening before feature expansion.** M1–M7 of the trigger roadmap stays parked until Sprints 0–3 here are green.

---

## TDD approach

Loop per change:

1. **Red** — write a failing test that asserts the new contract. Run `go test ./internal/fix/...` and confirm it fails.
2. **Green** — make the minimum code change so the test passes.
3. **Refactor** — tighten naming, lift helpers, kill duplication. Tests stay green.

### Tooling

| Concern | Tool | Convention |
|---|---|---|
| Unit test runner | `go test` | one `_test.go` per source file |
| Mocks for cluster mutation | `internal/fix/fake_mutator_test.go` | use the existing `fakeMutator` — do not introduce gomock |
| Fixtures | embedded JSON `const` in test files, or `testdata/` | follow existing pattern in [stuck_rs_pods_test.go](../../internal/fix/stuck_rs_pods_test.go) |
| Assertions | stdlib `testing` + `errors.Is` / `strings.Contains` | no testify dependency unless we add it once and uniformly |
| K8s API integration | `sigs.k8s.io/controller-runtime/pkg/envtest` for new operator-shaped tests | optional; only for Sprint 5 |
| Coverage gate | `go test -cover` in CI; target ≥70% on touched packages | block PR if package drops below threshold |
| CI | GitHub Actions, existing nightly publisher | add `make test`, `make lint`, `make cover` jobs |

### Test layout per touched package

```
internal/fix/
  gitops.go              ← new shared helper (Sprint 1)
  gitops_test.go         ← red-first
  stuck_rs_pods.go
  stuck_rs_pods_test.go  ← add new test cases
internal/probe/
  node_pressure.go       ← new probe (Sprint 2)
  node_pressure_test.go
  ...
```

### Definition of Done (per ticket)

- [ ] Failing test landed first (separate commit, message prefix `test:`)
- [ ] Implementation commit (`fix:` or `feat:`)
- [ ] `go vet ./...` clean
- [ ] `golangci-lint run` clean on touched files
- [ ] Coverage on touched package ≥ pre-change baseline
- [ ] CHANGELOG entry under `## Unreleased`
- [ ] README/docs updated if behavior changed

---

## Sprint 0 — Documentation truth-up (2 days, no tests)

Pure docs/manifest patches. Zero risk. Get this out of the way first so the public-facing surface stops lying.

| # | Change | Files | Acceptance |
|---|---|---|---|
| 0.1 | Replace user-local roadmap link with public path | [README.md:147](../../README.md#L147) | `grep -r '/home/skadam' .` returns 0 in tracked files |
| 0.2 | Decide LICENSE-VERIFIED-LIBRARY.md: write it (commercial OSS-add-on terms) OR delete the reference | [README.md:139](../../README.md#L139) | Reference no longer says "to be added" |
| 0.3 | Rewrite "Container image: kubectl + bash + jq + curl" to reflect Go-binary-on-distroless reality | [README.md:120](../../README.md#L120) | Matches [Dockerfile](../../Dockerfile) |
| 0.4 | Update "two ClusterRoles" → "three (reader, remediator, driftreport)" | [README.md:119](../../README.md#L119) | Matches `charts/.../templates/clusterrole-*.yaml` |
| 0.5 | Add an "AWS cloud probes" subsection to README (list the 10 in [catalog/cloud.go](../../catalog/cloud.go)) | [README.md](../../README.md), `--awsEnabled` Helm value | A user reading the README discovers AWS coverage exists |
| 0.6 | Reconcile VaultPathMissing OSS/paid status — code is OSS; either move it to CHA-com or update docs | [AI_TIERS.md:66](../AI_TIERS.md#L66), [CHA_OVERVIEW.md:94](../CHA_OVERVIEW.md#L94), [FAILURE_MODES.md:78](../FAILURE_MODES.md#L78) | Single consistent claim across all three docs |
| 0.7 | Add `[See docs/READINESS.md for pilot-vs-production limits]` to README near the install section | [README.md](../../README.md) | First-time reader finds the honesty doc |
| 0.8 | Delete the 3 PDFs from git; add them to a `cha-website/assets/` or `release-artifacts/` repo if they're still distributed | `docs/*.pdf` | `git ls-files docs/ \| grep .pdf` returns nothing |
| 0.9 | Write missing [docs/AI_COST_MODEL.md](../AI_COST_MODEL.md) — token counts × prices × per-incident call count, including investigation-cost amplification factor | new file | Operators can validate budget before enabling Layer-2 |
| 0.10 | Fix off-by-one in FAILURE_MODES.md intro ("seven OSS analyzers" → "eight") | [FAILURE_MODES.md:78](../FAILURE_MODES.md#L78) | Count matches the analyzer list below it |
| 0.11 | Document the published-Helm-chart-version discrepancy: either republish at the v1.5.2 line OR set `version` back to 0.9.5 if that's the released cut | [Chart.yaml](../../charts/cluster-health-autopilot/Chart.yaml) | `helm pull` from the public repo matches `Chart.yaml` |

---

## Sprint 1 — Fixer safety nets (1 week, TDD)

**Goal:** No fixer can fight a GitOps controller or violate operator intent (`spec.paused`, `spec.suspend`, namespace allowlist).

### 1.1 Lift GitOps helper into shared `internal/fix/gitops.go`

The helper exists privately in [tls_secret_mismatch.go:149](../../internal/fix/tls_secret_mismatch.go#L149) and only works on Ingress. Lift it to a shared, type-agnostic helper.

**Red — `internal/fix/gitops_test.go`:**
```go
func TestIsGitOpsManaged_Argo(t *testing.T) {
    // unstructured Deployment with annotation argocd.argoproj.io/instance=foo
    // → returns "argocd"
}
func TestIsGitOpsManaged_Flux_KustomizeLabel(t *testing.T) {
    // label kustomize.toolkit.fluxcd.io/name=app → "flux"
}
func TestIsGitOpsManaged_HelmManagedBy(t *testing.T) {
    // label app.kubernetes.io/managed-by=Helm → "helm"
}
func TestIsGitOpsManaged_PlainResource(t *testing.T) {
    // no annotations/labels → "" (empty)
}
func TestIsGitOpsManaged_DeploymentPaused(t *testing.T) {
    // spec.paused=true → "paused" (separate signal but same skip semantics)
}
```

**Green:**
```go
// internal/fix/gitops.go
package fix

func GitOpsReason(u unstructured.Unstructured) string { /* annotations+labels */ }
func IsPaused(u unstructured.Unstructured) bool       { /* spec.paused for Deployments */ }
```

**Refactor:** replace the private helper in `tls_secret_mismatch.go` to delegate. Existing TLS tests must stay green.

### 1.2 Apply GitOps + paused check to `StuckRSPods`

**Red — append to [stuck_rs_pods_test.go](../../internal/fix/stuck_rs_pods_test.go):**
```go
func TestStuckRSPods_SkipsArgoManagedDeployment(t *testing.T) {
    // Pod is stuck-RS, but the owning Deployment has argocd.argoproj.io/instance.
    // Expect: 0 mutations on fakeMutator; one FixSkipped result with reason="gitops:argocd"
}
func TestStuckRSPods_SkipsPausedDeployment(t *testing.T) {
    // Deployment.spec.paused=true → 0 mutations, reason="paused"
}
func TestStuckRSPods_AppliesOnPlainDeployment(t *testing.T) {
    // baseline — keep existing behavior
}
```

**Green:** in [stuck_rs_pods.go](../../internal/fix/stuck_rs_pods.go), fetch the owning Deployment, call `GitOpsReason` and `IsPaused`, skip if either is set. Emit a `Result{Kind: Skipped, Reason: ...}` so the watcher logs it.

### 1.3 `StuckJobsWithBadSecretRef` — honor `CronJob.spec.suspend`

**Red — append to [stuck_jobs_test.go](../../internal/fix/stuck_jobs_test.go):**
```go
func TestStuckJobs_SkipsSuspendedCronJob(t *testing.T) {
    // CronJob.spec.suspend=true → no Job deletion, no CronJob patch
}
func TestStuckJobs_SkipsArgoManagedCronJob(t *testing.T) {
    // argocd-managed CronJob → skipped
}
```

**Green:** Add the two checks before the existing delete/patch logic.

### 1.4 `StaleErrorPods` — GitOps owner check

**Red:**
```go
func TestStaleErrorPods_SkipsGitOpsOwnedPod(t *testing.T) {
    // Pod has owner Job, Job has argocd-managed-by → skipped
}
```

**Green:** When the owner is a Job/CronJob, walk one level up and check GitOps annotations.

### 1.5 `StuckCertificateRequests` — cert-manager health gate

**Red:**
```go
func TestStuckCertRequests_SkipsWhenCertManagerUnhealthy(t *testing.T) {
    // cert-manager Deployment has 0 ready replicas → skip CR deletion
}
```

**Green:** Probe the `cert-manager` namespace for controller readiness; if 0 ready, bail.

### 1.6 Helm chart — runaway job protection

| Change | File |
|---|---|
| Add `spec.activeDeadlineSeconds: 120` to both CronJob templates | [cronjob-diagnose.yaml](../../charts/cluster-health-autopilot/templates/cronjob-diagnose.yaml), [cronjob-remediate.yaml](../../charts/cluster-health-autopilot/templates/cronjob-remediate.yaml) |
| Add explicit `spec.backoffLimit: 1` | same |
| Expose both as Helm values with the above as defaults | [values.yaml](../../charts/cluster-health-autopilot/values.yaml) |

**Test:** `helm template` snapshot test in `charts/cluster-health-autopilot/tests/` using [helm-unittest](https://github.com/helm-unittest/helm-unittest).

---

## Sprint 2 — Probe coverage gaps (1.5 weeks, TDD)

**Goal:** Close the critical blind spots. Each new probe lands with a fixture-driven test.

Pattern for each: drop a captured `kubectl get … -o json` fixture in `testdata/`, write a test that loads it and asserts the probe's verdict, then implement the probe.

### 2.1 Node pressure probe

**Red — `internal/probe/node_pressure_test.go`:**
```go
func TestNodePressure_DiskPressure_True(t *testing.T) {
    // load testdata/nodes_disk_pressure.json (status.conditions[type=DiskPressure].status=True)
    // → Finding{Severity: Critical, Detail: "1 node(s) reporting DiskPressure: gpu-01"}
}
func TestNodePressure_MemoryPressure(t *testing.T) { ... }
func TestNodePressure_PIDPressure(t *testing.T) { ... }
func TestNodePressure_AllHealthy(t *testing.T) { /* baseline */ }
```

**Green:** new file `internal/probe/node_pressure.go`, register in [catalog/catalog.go](../../catalog/catalog.go).

### 2.2 kube-system DaemonSet health probe

**Red:**
```go
func TestKubeSystemDS_CNINotReady(t *testing.T) {
    // DaemonSet status.numberReady < status.desiredNumberScheduled, label k8s-app=cilium
    // → Critical
}
func TestKubeSystemDS_ProxyDegraded(t *testing.T) { ... }
```

Scope: any DaemonSet in `kube-system`, `cilium-system`, `calico-system`, `kube-flannel`, `rook-ceph` (CSI plugins).

### 2.3 Pending-pods (scheduling failure) probe

**Red:**
```go
func TestPendingPods_InsufficientCPU(t *testing.T) {
    // Pod.status.phase=Pending, conditions include PodScheduled=False reason=Unschedulable
    // → Critical with reason text
}
func TestPendingPods_TransientPullBackoffIgnored(t *testing.T) {
    // ImagePullBackOff is not scheduling failure → ignored (ImagePullAuth handles it)
}
```

### 2.4 Generic CrashLoopBackOff probe

**Red:**
```go
func TestCrashLoop_AnyNamespaceFlagged(t *testing.T) {
    // Pod in random namespace with restartCount > 5 + waiting reason CrashLoopBackOff
    // → Warning (not Critical — most CL loops resolve)
}
func TestCrashLoop_ProtectedNamespacesAlwaysSkipped(t *testing.T) {
    // kube-system pods do escalate to Critical (protected = more important, not less)
}
```

### 2.5 ETCD health probe

**Red:**
```go
func TestETCD_AllMembersHealthy(t *testing.T) { /* via kube-system etcd pod status */ }
func TestETCD_MemberDown_Critical(t *testing.T) { ... }
```

If etcd is external (stacked masters with no in-cluster pods), the probe degrades to a warning that says "external etcd; install etcd-exporter for visibility."

### 2.6 Make critical-workload list configurable

**Red — `internal/probe/services_test.go`:**
```go
func TestServiceTargets_FromHelmValue(t *testing.T) {
    // Set CHA_CRITICAL_SERVICES env to "ns1/app=foo;ns2/app=bar"
    // → Services probe targets only those, ignores the hardcoded default
}
func TestServiceTargets_DefaultsWhenUnset(t *testing.T) {
    // hardcoded list returned only when env unset
}
```

**Green:**
- Parse `CHA_CRITICAL_SERVICES` env in [services.go:144](../../internal/probe/services.go#L144)
- Add `criticalWorkloads:` array Helm value, wire to env
- **Decision:** the existing 32-entry hardcoded list moves to `examples/values.bionic-cluster.yaml` as Salil's reference config; the OSS default is **empty + auto-discovery from `cha.bionicaisolutions.com/probe-critical: "true"` annotation** on Deployments/StatefulSets.

### 2.7 Failed-mount probe

**Red:** Pod stuck `ContainerCreating` with event `Unable to attach or mount volumes` → Finding.

---

## Sprint 3 — AI / paid-tier safety (1 week, TDD)

All work in [CHA-com](../../../CHA-com).

### 3.1 Patch payload semantic validation

**Red — `ai/approval/executor_test.go`:**
```go
func TestExecutor_RejectsReplicasZeroOnStatefulSet(t *testing.T) {
    // Action{Kind:PatchDeployment, ResourceKind:StatefulSet, Payload:`{"spec":{"replicas":0}}`}
    // → ErrForbiddenPatchField, audit-trail entry, no API call
}
func TestExecutor_RejectsImmutableFieldPatch_Selector(t *testing.T) { ... }
func TestExecutor_RejectsPayloadAbove64KB(t *testing.T) { ... }
func TestExecutor_AllowsRolloutRestartAnnotation(t *testing.T) {
    // Payload only touches spec.template.metadata.annotations.kubectl.kubernetes.io/restartedAt → allow
}
```

**Green:** Add `validatePatch(action Action, payload []byte) error` with an allow-list of JSON paths per `ActionKind`. Reject anything outside.

### 3.2 Investigation rate limiter

**Red — `ai/rate_limit_test.go`:**
```go
func TestRateLimit_GatesInvestigations(t *testing.T) {
    // 5 consecutive CRITICAL findings on same diagnostic class
    // → 5th investigation blocked, breaker returns deterministic fallback
}
func TestRateLimit_PerDiagnosticClassNotGlobal(t *testing.T) {
    // 5 TLS investigations + 1 DNS investigation in same hour → DNS still allowed
}
```

**Green:** Add `TakeInvestigation(class)` keyed on `(approver_identity, diagnostic_class)`. Default budget configurable via env.

### 3.3 Cold-start burst mitigation

**Red:**
```go
func TestRateLimit_ColdStartBucketFillsGradually(t *testing.T) {
    // On NewBucket, capacity=0; refills at rate over 1 hour
    // → first action allowed after refill_period only
}
```

**Green:** Initialize buckets at 0 tokens, not full. Document the trade-off.

### 3.4 Event-message secret scrubbing

**Red — `pkg/ai/redact_test.go`:**
```go
func TestRedactEvents_SecretLikeValuesScrubbed(t *testing.T) {
    // EventList with .message "STRIPE_API_KEY=sk_live_xyz rejected"
    // → after RedactEvents(), message contains "[REDACTED]" not the key
}
```

**Green:** New `RedactEvents([]Event) []Event` that calls `ContainsSecretLike` on `.message` and `.note`. Wire into [pkg/ai/environment.go](../../pkg/ai/environment.go) `GetEvents()` wrapper.

### 3.5 Compile-time Environment interface assertion

**Red:** Add to CHA-com `ai/environment_impl.go`:
```go
var _ pkg_ai.Environment = (*Impl)(nil)
```

If CHA-com drifts from the OSS interface, the build fails. This is a structural test, not a runtime test.

### 3.6 Approval audit-trail tamper-evidence

**Red:**
```go
func TestAuditTrail_HashChainContinuity(t *testing.T) {
    // emit 3 entries, then mutate entry 2's payload
    // → verifier reports broken chain at entry 3
}
```

**Green:** Each entry stores `prevHash = sha256(prev_entry_serialized)`. Verifier walks chain. Sink remains pluggable (Vault audit log preferred).

---

## Sprint 4 — Operability + watcher tests (1 week, TDD)

### 4.1 Watcher unit tests

The 677-LOC [watcher.go](../../internal/watcher/watcher.go) has zero tests. Highest refactor-risk file in the repo.

**Red — `internal/watcher/watcher_test.go`:**
```go
func TestWatcher_DebouncesRapidEvents(t *testing.T) {
    // Fire 10 events in 100ms → exactly one cycle runs
}
func TestWatcher_DriftReportDedup(t *testing.T) {
    // Same issue surfaced 3 cycles in a row → 1 Slack post, not 3
}
func TestWatcher_PostFixReverification(t *testing.T) {
    // Fixer applied → re-probe → finding gone → report shows "fixed"
}
func TestWatcher_ApprovalURLPropagation(t *testing.T) {
    // Paid mode: pending-approval action → Slack message includes signed URL
}
func TestWatcher_GracefulShutdown(t *testing.T) {
    // ctx.Done() → all goroutines exit within 2s
}
```

**Green:** Likely requires extracting interfaces (`snapshotSource`, `slackSink`) so they can be faked. This is part of the test; do not skip.

### 4.2 Ticketing config upfront validation

**Red — `cmd/cha/main_test.go`:**
```go
func TestBuildTicketingConfig_OpenProjectRequiresProject(t *testing.T) {
    // --ticketing-provider=openproject without --ticketing-project
    // → error from buildTicketingConfig, not silent empty config
}
```

**Green:** Validate required-flag combinations in [cmd/cha/main.go:689](../../cmd/cha/main.go) before returning the config.

### 4.3 Leader election (Option A — decided 2026-05-22)

Use `k8s.io/client-go/tools/leaderelection`. Factor existing watcher loop into `RunAsLeader(ctx)`; wrap startup with a `LeaderElector` racing on a `coordination.k8s.io/v1.Lease`.

**Red — `internal/watcher/leader_test.go`:**
```go
func TestLeader_AcquiresLeaseWhenUncontested(t *testing.T) {
    // single instance, fake client → OnStartedLeading fires within LeaseDuration
}
func TestLeader_FailoverOnLeaderDeath(t *testing.T) {
    // two instances, A holds lease then ctx-cancels → B acquires within RetryPeriod+LeaseDuration
}
func TestLeader_LoserBlocksAndDoesNotRunWatcher(t *testing.T) {
    // B never runs the watcher loop while A holds the lease
}
func TestLeader_GracefulReleaseOnShutdown(t *testing.T) {
    // SIGTERM → leader releases lease cleanly (no 30s wait for new leader)
}
func TestLeader_DisabledByFlag(t *testing.T) {
    // CHA_LEADER_ELECTION=off → runs watcher loop directly, no lease created
}
```

**Green:**
- New file `internal/watcher/leader.go` with the `LeaderElector` setup. Lease name: `cha-watcher`. Namespace: deployment's own namespace (downward-API).
- LeaseDuration: 30s. RenewDeadline: 20s. RetryPeriod: 5s. (Standard controller-manager defaults.)
- Wrap `Watcher.Run(ctx)` so that the existing loop runs only inside `OnStartedLeading`.
- Add `--leader-election-namespace` flag (defaults to pod's namespace).
- Add `CHA_LEADER_ELECTION` env (default `on`, `off` for single-pod dev).

**Chart changes:**
- Add `coordination.k8s.io/leases: [get,list,watch,create,update,patch]` to [clusterrole-remediator.yaml](../../charts/cluster-health-autopilot/templates/clusterrole-remediator.yaml). Scope tight — `resourceNames: ["cha-watcher"]` if possible.
- Add Helm value `watcher.leaderElection.enabled: true` (default).
- Add `PodDisruptionBudget` with `maxUnavailable: 1` — explicit eviction permission during node drains.
- Document that `replicas: 2` is now the recommended HA default; bump chart default.

**Tests for the chart:** helm-unittest assertions on the new ClusterRole verb + PDB.

**Strategic note:** This code is **not throwaway.** When Sprint 5 ports to controller-runtime, `manager.Options{LeaderElection: true}` wraps the same `client-go/tools/leaderelection` primitive. The lease name, namespace, and durations carry over unchanged. Only the startup glue is rewritten.

### 4.4 Publish images to `ghcr.io/bionic-ai-solutions`

- Add GitHub Actions workflow `.github/workflows/release.yml` that pushes to GHCR on tag
- Update [values.yaml](../../charts/cluster-health-autopilot/values.yaml) defaults: `image.repository: ghcr.io/bionic-ai-solutions/cluster-health-autopilot`
- Keep `docker4zerocool/*` as a mirror, not the default

### 4.5 Wire a dummy CHA-com paid analyzer

**Red — `CHA-com/catalog/paid_test.go`:**
```go
func TestPaidCatalog_DummyAnalyzerRegistered(t *testing.T) {
    // After Register(), registry.Analyzers should include "VaultPathDriftPro"
}
```

**Green:** Add a single trivial paid analyzer (even one that always returns no findings). The point is to exercise the OSS/paid boundary in CI before the real paid features land.

---

## Sprint 5 — Operator port (deferred, separate plan)

Out of scope for this hardening pass. Mentioned for sequencing:

- Define `HealthPolicy` CRD (v1alpha2 — bump from v1alpha1 with conversion webhook)
- Wrap probe→diagnose→fix in `Reconcile(*HealthPolicy) Result`
- Add controller-runtime + envtest
- Migrate `DriftReport` to a `status` subresource of `HealthPolicy`
- Estimated: 2–3 weeks, separate roadmap doc

---

## Sprint plan summary

| Sprint | Duration | Risk | Tests added | Tests changed |
|---|---|---|---|---|
| 0 — Docs | 2 days | None | 0 | 0 |
| 1 — Fixer safety | 1 week | Med (touches mutators) | ~15 | ~5 |
| 2 — Probe gaps | 1.5 weeks | Low (additive) | ~25 | 0 |
| 3 — AI safety | 1 week | Med (paid binary) | ~12 | ~3 |
| 4 — Operability | 1 week | High (watcher refactor) | ~10 | ~2 |
| 5 — Operator port | 2–3 weeks | High | TBD | TBD |

**Total Sprint 0–4:** ~5 weeks for one engineer, or ~3 weeks for two with Sprints 2+3 in parallel.

---

## CI hardening (parallel to all sprints)

Land alongside Sprint 0:

- [ ] `.github/workflows/test.yml` — `go test ./... -race -cover` on every PR
- [ ] `.github/workflows/lint.yml` — `golangci-lint` with `errcheck`, `gosec`, `staticcheck`
- [ ] Coverage badge in README
- [ ] `helm lint charts/cluster-health-autopilot/` in CI
- [ ] `helm unittest` for chart templates (after Sprint 1.6)
- [ ] Codecov or coveralls integration
- [ ] Pre-merge gate: no PR merges if any package coverage drops > 2%

---

## Risk register

| Risk | Likelihood | Mitigation |
|---|---|---|
| Watcher refactor (4.1) introduces regression | High | TDD-first; keep behavioral fixture from a real run as integration test |
| Leader election deferral causes prod incident | Med | Sprint 4.3 explicit `replicas: 1` constraint + Pod Disruption Budget |
| Sprint 2 probes generate alert noise | Med | All new probes start at `Warning`, not `Critical`; promote after 1 week pilot |
| CHA-com patch validator over-restricts and breaks legitimate fixes | Low | Allow-list per `ActionKind` reviewed by 2 engineers; emergency env override `CHA_AI_PATCH_VALIDATION=permissive` |

---

## Out of scope (intentional)

- M1–M7 trigger expansion (separate roadmap, parked until this is done)
- Multi-cluster federation
- Web UI / dashboard
- Prometheus/Grafana scraping integration (can be done in parallel by anyone; not blocking)
