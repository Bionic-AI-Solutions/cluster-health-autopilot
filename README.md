# Cluster Health Autopilot (`cha`)

A self-healing operational layer for Kubernetes clusters: **detect → remediate → re-verify → report**, on a schedule, without dashboards or pagers.

> **Pre-launch — engineering preview.** This README will be the public face on launch day; treat its current contents as draft.

## Status (v1.8.0 — 2026-05-28)

| Capability | Status |
|---|---|
| K8s probes (12) — Ceph, Postgres, Nodes, PVCs, Critical services, Endpoints, NodePressure, DaemonSets, PendingPods, CrashLoopBackOff, ETCD, FailedMounts | ✅ shipped |
| **M2 K8s probes (4, v1.8)** — Kong route/upstream, HPA scaling-failure, ArgoCD Application sync, Velero backup-completion (each auto-skips when its CRD is absent) | ✅ shipped |
| Diagnose analyzers (8) — secret/cert/ESO/image-pull/TLS classes | ✅ shipped |
| **Drift-class analyzers (6)** — `GitOpsDrift`, `WorkloadStateDrift`, `RBACDrift` (v1.7) + `ConfigDrift`, `CapacityDrift`, `SecurityDrift` (v1.8) | ✅ shipped |
| Fixers (4 default + 1 opt-in) with GitOps + paused + suspended + cert-mgr-health safety gates | ✅ shipped |
| **Cloud probes (30) — 10 each AWS / GCP / Azure**, all with Live SDK wrappers (IRSA / GCP Workload Identity / AAD Workload Identity). Off by default; enable per-provider | ✅ shipped |
| **Operator port (controller-runtime, v1.8)** — `ClusterHealthAutopilot` CRD + `cha-operator` manager reconciling watcher Deployment / CronJobs / RBAC; `Ready` + `WatcherRunning` conditions. Opt-in via `operator.enabled` | ✅ shipped (Phase 1 + 1b) |
| Helm chart (v1.8.0) with lease-based leader election, configurable workloads, narrow RBAC, per-severity Slack repeat intervals, analyzer + cloud-provider toggles | ✅ shipped |
| OpenProject ticketing sink (OSS) via MCP, Slack 3-channel routing with `--slack-critical-repeat-interval`, Alertmanager | ✅ shipped |
| Layer-2 Investigator (OSS: deterministic implementation; CHA-com: LLM-backed agent) with event scrubbing + 6-tool action space | ✅ shipped |
| CHA-com paid binary v1.7.0: approval-server, Ed25519 signing, gen-signing-key, paid catalog | ✅ shipped at `docker4zerocool/cha-com:1.7.0` (multi-arch) |
| Four paid-tier analyzers: `VaultPathDriftPro`, `CertificateChainAnomaly`, `MultiClusterDrift`, `StatefulSetReplicaPressure` | ✅ shipped (CHA-com v1.0.1–v1.0.4) |
| Paid AI tiers — T0 narration, T1 fix proposer (5 operator-policy-bounded action_kinds), T2 multi-step planner, T3 vault break-glass runbook (dual-approval, JWT-signed, hash-chained audit) | ✅ shipped (CHA-com v1.1.0–v1.4.0) |
| `cha-com watch` subcommand — AI-layered poll loop with fingerprint dedup, `--ai-audit-log` JSONL sink | ✅ shipped (CHA-com v1.5.0) |
| **`LLMFixerMatcher`** (CHA-com v1.7.0, opt-in `--ai-llm-fixer-matcher`) — LLM-classified fixer matching with keyword fallback | ✅ shipped |
| Operator OLM bundle (Phase 1c), trigger-classes C (Alertmanager webhook) / E (external webhook) | ⏳ roadmap (v1.9+) |

---

## What it does

`cha` runs a battery of probes against your cluster, applies operator-policy-bounded fixes for recognized failure patterns, re-probes, and produces a single report listing fixes applied and any residual issues with precise remediation hints. The paid CHA-com layer adds an **LLM Investigator agent** and the T0–T3 AI SRE tiers on top — same policy bounds, signed actions, hash-chained audit. CHA runs in two modes:

- **Zero-trust offline mode** — point it at a captured `kubectl get … -o json` snapshot. No install, no RBAC, no write permissions. Diagnose your cluster from your laptop in 30 seconds.
- **In-cluster live mode** — installed via Helm; runs as a CronJob with two narrowly-scoped ClusterRoles (read-only + tightly-bounded write); posts to Slack on a schedule.

## 30-second demo (no cluster needed)

Try the analyzer against the sample fixture in this repo:

```sh
git clone https://github.com/Bionic-AI-Solutions/cluster-health-autopilot.git
cd cluster-health-autopilot
go run ./cmd/cha diagnose --snapshot examples/sample-cluster
```

Expected output:

```
• Ceph Storage:     🟢 HEALTHY    1 cluster(s): rook-ceph@rook-ceph OK (11.5% used)
• Cluster Nodes:    🟢 HEALTHY    All 4 nodes ready
• PostgreSQL:       🟢 HEALTHY    1 CNPG cluster(s): main@data (3/3 ready, primary=main-1)
• Storage Claims:   🟢 HEALTHY    All 3 PVCs bound

Diagnostics (3):
  🔎 Secret `billing/billing-svc-secrets` missing key `STRIPE_API_KEY` (referenced by
     Deployment/billing-svc in ns billing). Owning ExternalSecret: `billing/billing-svc-secrets`
     — add data/template entry exposing `STRIPE_API_KEY`, or remove the env reference if unused.
  🔎 ExternalSecret `billing/billing-svc-secrets` not Ready: error processing spec.data[0]
     (key: shared/billing/config), err: cannot find secret data for key: "stripe_api_key".
  🔎 ExternalSecret `billing/old-payment-gateway` not Ready: error processing spec.data[0]
     (key: shared/legacy/payments), err: vault path not found.
```

That's the headline: a precise diagnosis (which Secret, which key, which Deployment, which ExternalSecret, what Vault property is missing) without an install, without RBAC, without writing anything.

## Run on your own cluster

```sh
# 1. Capture a snapshot of your cluster (read-only — never modifies state)
cha snapshot capture --out ./my-cluster
# or single-tarball form for sharing:
cha snapshot capture --tar my-cluster.tgz

# 2. Diagnose offline against the captured snapshot
cha diagnose --snapshot ./my-cluster

# 3. Or just run it against the live cluster directly
cha diagnose --live
```

`cha snapshot capture` reads only — it cannot modify any cluster state. It writes a directory (or `.tgz`) of `kubectl get -o json` files for the canonical resource set the analyzers need.

## In-cluster install (Helm)

```sh
helm repo add cha https://bionic-ai-solutions.github.io/cluster-health-autopilot
helm repo update
helm install cha cha/cluster-health-autopilot \
  --namespace cluster-health-autopilot --create-namespace \
  --set slackWebhookSecretName=cha-slack-webhook
```

Full Helm chart at [`charts/cluster-health-autopilot/`](charts/cluster-health-autopilot) — published at `https://bionic-ai-solutions.github.io/cluster-health-autopilot/`.

## What it checks (12 probes)

K8s cluster:
- **Ceph storage** — health, capacity, OSD readiness
- **PostgreSQL** — CloudNativePG + Zalando Spilo/Patroni, auto-detected
- **Critical workloads** — configurable list (32 Bionic defaults + `CHA_CRITICAL_SERVICES` env + `cha.bionicaisolutions.com/probe-critical: "true"` annotation auto-discovery)
- **Cluster Nodes** — Ready condition
- **PVC binding** — Bound vs Pending
- **External endpoints** — Ingress host auto-discovery + flake suppression (Layer-1)
- **Node pressure** *(v1.6)* — DiskPressure / MemoryPressure / PIDPressure / NetworkUnavailable; DiskPressure auto-escalates to Critical
- **System DaemonSets** *(v1.6)* — kube-system / cilium-system / calico-system / kube-flannel / rook-ceph / longhorn-system / openebs / metallb-system
- **Pending pods** *(v1.6)* — `PodScheduled=False` past 60s grace; reason-aware remediation (Insufficient CPU/Memory, unbound PVC, taint mismatch, nodeSelector)
- **Generic CrashLoopBackOff** *(v1.6)* — any namespace; protected-NS escalates to Critical immediately, user-NS escalates past restart threshold (default 10)
- **ETCD** *(v1.6)* — kubeadm-style static-pod etcd; honest "blind probe" Warning on external/managed etcd rather than false-greening
- **Failed mounts** *(v1.6)* — joins pods stuck `ContainerCreating` with kubelet `FailedMount` / `FailedAttachVolume` / `ProvisioningFailed` events

Each probe can be disabled independently via `CHA_PROBE_<NAME>=off`.

## What it diagnoses (8 OSS analyzers — read-only)

- **SecretKeyMissing** — pod stuck in `CreateContainerConfigError`; names the missing key + consuming Deployment + owning ExternalSecret.
- **FailingExternalSecrets** — walks every ExternalSecret with `Ready=False`, surfaces the controller's specific error message (the missing Vault property name).
- **ProactiveSecretKeyCheck** — walks workload env references; flags Secret keys that don't exist yet so the next pod restart won't hit ConfigError silently.
- **UnprovisionedSecret** — workload references a Secret with no ExternalSecret provisioning it; suggests the canonical Vault path.
- **ImagePullAuth** — pod in `ImagePullBackOff` with kubelet event auth signals (401, denied, unauthorized).
- **CertExpiry** — cert-manager Certificate not Ready, expiring within 14 days, or already expired.
- **TLSSecretMismatch** — Ingress points at an expired Secret while cert-manager is renewing a healthy cert into a different Secret in the same namespace. (Two-Secret naming drift.)
- **VaultPathMissing** — Apache-2.0 source ships in OSS; requires you to construct a Vault client and register it explicitly (`reg.RegisterAnalyzer(diagnose.VaultPathMissing{Client: vc})`). Queries Vault directly to catch drift before ESO's next refresh marks `Ready=False`. The paid CHA Enterprise binary auto-wires this from your Vault configuration.

## What it checks in your cloud account (AWS — opt-in)

Enable with `--aws-enabled` (or `cloudProbes.aws.enabled: true` in Helm values). The probes use the standard AWS SDK credential chain (IRSA, instance profile, env vars). 10 probes ship today:

- **RDS** — instance/cluster status, storage, multi-AZ, backup retention drift
- **EBSVolumes** — orphan/unattached, snapshot age
- **EKSControlPlane** — version skew vs. node groups, addon staleness
- **EKSNodeGroups** — capacity, scaling activity, version drift
- **IAMRoles** — trust policy drift on cluster service-account roles
- **ALBTargetHealth** — unhealthy targets in Load Balancer Controller-managed TGs
- **ACMCertExpiry** — certs expiring within 14 days
- **KMSKeys** — pending-deletion keys still referenced by cluster resources
- **S3BucketPublicAccess** — public-ACL drift on buckets referenced by cluster IAM
- **VPCSubnets** — exhausted IP space affecting pod CIDR allocation

GCP and Azure probes are scoped for the M2 milestone of the cloud-probe roadmap; see [`docs/design/2026-05-cloud-probe-framework.md`](docs/design/2026-05-cloud-probe-framework.md).

## What it auto-fixes (operator-policy-bounded)

- **StaleErrorPods** — `Error`/`Failed` pods owned by a Job or unowned (debug leftovers).
- **StuckJobsWithBadSecretRef** — frozen Jobs whose pod template references a renamed Secret key; CronJob template is corrected — fixer deletes the Job so the CronJob respawns clean.
- **StuckRSPods** — ReplicaSet pods stuck on a stale revision when the Deployment has rolled forward (`kubectl rollout restart`).
- **StuckCertificateRequests** — cert-manager CRs in terminal `Ready=False`/`Failed`; deletion lets cert-manager re-issue.
- **TLSSecretMismatch** (opt-in) — repoints `Ingress.spec.tls[].secretName` to the cert-manager-managed Secret when the analyzer detects a mismatch. Skips GitOps-managed Ingresses (Argo/Flux/Helm labels) so it doesn't fight a reconcile loop. Enable with `--set fixers.tlsSecretMismatch.enabled=true`.

**Never auto-applied:** edits to Secrets, ConfigMaps, or generic CRDs (those changes need a human + git).

## Probe behavior (v1.2 / v1.4)

- **Auto-discovery** — every Ingress host in the cluster is auto-probed externally. Per-Ingress opt-out via annotation `cha.bionicaisolutions.com/probe-disable: "true"`. Protected namespaces always skipped.
- **Flake suppression** — first failed probe of a target is tagged `[transient, 1/2]` and emits at warning; only a second consecutive failure escalates to CRITICAL. Deterministic failures (TLS error, status mismatch, invalid URL) bypass the streak counter and alert immediately.

## Layer-2 Investigator (v1.5)

When a Finding or Diagnostic reaches CRITICAL, a read-only Investigator runs a deep-dive (DNS, HTTP probe, TLS inspect, kubectl describe, recent events) and attaches a one-line root-cause Summary to the alert. Renders as a 🔬 block in Slack/Alertmanager and persists on `DriftReport.spec.investigation`.

OSS ships a deterministic Investigator implementation covering TLS expiry, TLS SAN mismatch, DNS failure, slow-DNS classification, transient-recovery detection, status mismatch, ExternalSecret diagnostics, and Certificate state — the predictable baseline. No new RBAC; reuses the watcher's existing read access. Disable with `CHA_INVESTIGATOR=off`. The paid CHA-com binary replaces it with an **LLM Investigator agent** using the same closed-enum `Environment` surface — same read-only tool space, same audit shape, more reasoning.

## Architecture

- Two CronJobs (diagnose + remediate), one ServiceAccount, three ClusterRoles (reader, remediator, driftreport). An optional long-running Watcher Deployment and an Approval Server are deployed when their features are enabled.
- Container image: a single `cha` Go binary (~30 MB) on `gcr.io/distroless/static:nonroot`. No shell, no package manager, no proprietary registry. Built via the [Dockerfile](Dockerfile) at the repo root.
- Credentials: ExternalSecret from Vault wherever possible — no plaintext credentials baked into any manifest.
- Footprint: ~30 MB RSS, sub-second CPU, <60 s wall-clock per probe-and-fix cycle.
- See [`docs/READINESS.md`](docs/READINESS.md) for pilot-vs-production limits, RBAC blast-radius notes, and known operational caveats.

## Docs

- **[docs/CHA_OVERVIEW.md](docs/CHA_OVERVIEW.md)** — **two-pager**: what CHA is, OSS vs Paid, what it does/doesn't do, what AI does/doesn't do (and why). Start here.
- **[docs/ONE_PAGER.md](docs/ONE_PAGER.md)** — design-partner brief; the elevator-pitch version of this README with pricing and validation.
- **[docs/FAILURE_MODES.md](docs/FAILURE_MODES.md)** — every fixer + analyzer in the catalog: symptom, root cause, why it's safe, real-world example, source link.
- **[docs/AI_TIERS.md](docs/AI_TIERS.md)** — definitive spec for Layer-2 + T0–T3 (capabilities, inputs, output schemas, safety contracts).
- **[docs/AI_USAGE.md](docs/AI_USAGE.md)** — positioning: LLM-free hot path; deterministic Layer-2 investigator baseline in OSS; LLM-backed Investigator agent + AI tiers in CHA-com (paid + opt-in).
- **[docs/SETUP_GUIDE.md](docs/SETUP_GUIDE.md)** — install reference, Helm value catalog, RBAC, troubleshooting.
- **[docs/DEMO_GUIDE.md](docs/DEMO_GUIDE.md)** — storyboarded demo flow with deliberate-failure scenarios.
- **[docs/design/2026-05-investigator-agent.md](docs/design/2026-05-investigator-agent.md)** — architecture rationale for Layer-2.

## License

[Apache License 2.0](LICENSE) for the engine and the default signature library.

The **Verified Signature Library** (curated, regression-tested patterns added monthly) ships as a separate signed bundle under a commercial subscription license to CHA Enterprise customers. See [`LICENSE-VERIFIED-LIBRARY.md`](LICENSE-VERIFIED-LIBRARY.md) for the terms.

## Security

To report a vulnerability, email **cha-security@baisoln.com**. See [SECURITY.md](SECURITY.md).

## Roadmap

Active design docs live in [`docs/design/`](docs/design/):

- [Hardening plan (Sprints 0–4)](docs/design/2026-05-hardening-plan.md) — TDD-driven punch-list closing the 2026-05 adversarial review
- [Trigger expansion roadmap (v1.6 → v2.0)](docs/design/2026-05-trigger-expansion-roadmap.md) — Kong, GPU, HPA, ArgoCD, Velero, Vault, log-pattern probes
- [Cloud probe framework](docs/design/2026-05-cloud-probe-framework.md) — AWS shipped, GCP/Azure in M2
- [Investigator agent](docs/design/2026-05-investigator-agent.md) — Layer-2 architecture rationale
- [Ticketing MCP integration](docs/design/2026-05-ticketing-mcp-integration.md) — OpenProject + MCP transport
