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
