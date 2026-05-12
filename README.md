# Cluster Health Autopilot (`cha`)

A self-healing operational layer for Kubernetes clusters: **detect → remediate → re-verify → report**, on a schedule, without dashboards or pagers.

> **Pre-launch — engineering preview.** This README will be the public face on launch day; treat its current contents as draft.

---

## What it does

`cha` runs a battery of probes against your cluster, applies a whitelist of known-safe fixes for recognized failure patterns, re-probes, and produces a single report listing fixes applied and any residual issues with precise remediation hints. It runs in two modes:

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

## What it checks (probes)

- Distributed storage health (Ceph)
- Database health (CloudNativePG, Zalando Spilo/Patroni — auto-detected)
- Critical workloads (configurable list, counted by `READY` column not `phase=Running`)
- Cluster nodes
- PVC binding state
- API connectivity sanity (so transient API problems become a reported `PROBE_FAILED`, not a silent green light)

## What it diagnoses (7 OSS analyzers — read-only)

- **SecretKeyMissing** — pod stuck in `CreateContainerConfigError`; names the missing key + consuming Deployment + owning ExternalSecret.
- **FailingExternalSecrets** — walks every ExternalSecret with `Ready=False`, surfaces the controller's specific error message (the missing Vault property name).
- **ProactiveSecretKeyCheck** — walks workload env references; flags Secret keys that don't exist yet so the next pod restart won't hit ConfigError silently.
- **UnprovisionedSecret** — workload references a Secret with no ExternalSecret provisioning it; suggests the canonical Vault path.
- **ImagePullAuth** — pod in `ImagePullBackOff` with kubelet event auth signals (401, denied, unauthorized).
- **CertExpiry** — cert-manager Certificate not Ready, expiring within 14 days, or already expired.
- **TLSSecretMismatch** — Ingress points at an expired Secret while cert-manager is renewing a healthy cert into a different Secret in the same namespace. (Two-Secret naming drift.)

Plus **VaultPathMissing** in the paid catalog — queries Vault directly to catch drift before ESO's next refresh marks `Ready=False`.

## What it auto-fixes (whitelisted)

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

The OSS catalog ships a deterministic, rule-based Investigator covering TLS expiry, TLS SAN mismatch, DNS failure, slow-DNS classification, transient-recovery detection, status mismatch, ExternalSecret diagnostics, and Certificate state. No new RBAC; reuses the watcher's existing read access. Disable with `CHA_INVESTIGATOR=off`. The paid CHA-com binary replaces it with an LLM-backed Investigator using the same closed-enum `Environment` surface.

## Architecture

- One CronJob, one ConfigMap (the script), one ServiceAccount, two ClusterRoles.
- Container image: `kubectl + bash + jq + curl` (no proprietary registry).
- Webhook: ExternalSecret from Vault — no plaintext credentials in any manifest.
- <100 MB RAM, <100 ms CPU, <60 s wall-clock per run.

## Docs

- **[docs/CHA_OVERVIEW.md](docs/CHA_OVERVIEW.md)** — **two-pager**: what CHA is, OSS vs Paid, what it does/doesn't do, what AI does/doesn't do (and why). Start here.
- **[docs/ONE_PAGER.md](docs/ONE_PAGER.md)** — design-partner brief; the elevator-pitch version of this README with pricing and validation.
- **[docs/FAILURE_MODES.md](docs/FAILURE_MODES.md)** — every fixer + analyzer in the catalog: symptom, root cause, why it's safe, real-world example, source link.
- **[docs/AI_TIERS.md](docs/AI_TIERS.md)** — definitive spec for Layer-2 + T0–T3 (capabilities, inputs, output schemas, safety contracts).
- **[docs/AI_USAGE.md](docs/AI_USAGE.md)** — positioning: LLM-free hot path; rule-based Layer-2 investigator in OSS; LLM AI is paid + opt-in.
- **[docs/SETUP_GUIDE.md](docs/SETUP_GUIDE.md)** — install reference, Helm value catalog, RBAC, troubleshooting.
- **[docs/DEMO_GUIDE.md](docs/DEMO_GUIDE.md)** — storyboarded demo flow with deliberate-failure scenarios.
- **[docs/design/2026-05-investigator-agent.md](docs/design/2026-05-investigator-agent.md)** — architecture rationale for Layer-2.

## License

[Apache License 2.0](LICENSE) for the engine and the default signature library.

The **Verified Signature Library** (curated, regression-tested patterns added monthly) ships as a separate signed bundle under a commercial license. See [LICENSE-VERIFIED-LIBRARY.md] *(to be added before public launch)*.

## Security

To report a vulnerability, email **cha-security@baisoln.com**. See [SECURITY.md](SECURITY.md).

## Roadmap

See [/home/skadam/.claude/plans/i-have-been-adviced-hashed-lecun.md] for the WS-A → WS-D rollout plan.
