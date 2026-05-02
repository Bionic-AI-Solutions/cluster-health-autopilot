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
helm repo add cha https://<org>.github.io/cluster-health-autopilot
helm install cha cha/cluster-health-autopilot \
  --namespace cluster-health-autopilot --create-namespace \
  --set slackWebhookSecretName=cha-slack-webhook
```

Full Helm chart at [`charts/cluster-health-autopilot/`](charts/cluster-health-autopilot).

## What it checks (probes)

- Distributed storage health (Ceph)
- Database health (CloudNativePG, Zalando Spilo/Patroni — auto-detected)
- Critical workloads (configurable list, counted by `READY` column not `phase=Running`)
- Cluster nodes
- PVC binding state
- API connectivity sanity (so transient API problems become a reported `PROBE_FAILED`, not a silent green light)

## What it diagnoses (analyzers — read-only)

- **Missing-Secret-key** — when a pod is stuck in `CreateContainerConfigError`, names the missing key, the consuming Deployment, and the owning ExternalSecret.
- **Failing-ExternalSecret** — walks every ExternalSecret cluster-wide whose `Ready=False`, surfaces the controller's specific error message (the missing Vault property name).

## What it auto-fixes (whitelisted)

- Stale `Error`/`Failed` pods owned by a Job or unowned (debug leftovers).
- Frozen `Job` whose pod template references a Secret key that no longer exists; the parent CronJob's template has been corrected — the fixer deletes the Job so the CronJob respawns clean.
- ReplicaSet pods stuck on a stale revision when the Deployment has rolled forward — `kubectl rollout restart`.

**Never auto-applied:** edits to Secrets, ConfigMaps, or CRDs (those changes need a human + git).

## Architecture

- One CronJob, one ConfigMap (the script), one ServiceAccount, two ClusterRoles.
- Container image: `kubectl + bash + jq + curl` (no proprietary registry).
- Webhook: ExternalSecret from Vault — no plaintext credentials in any manifest.
- <100 MB RAM, <100 ms CPU, <60 s wall-clock per run.

## License

[Apache License 2.0](LICENSE) for the engine and the default signature library.

The **Verified Signature Library** (curated, regression-tested patterns added monthly) ships as a separate signed bundle under a commercial license. See [LICENSE-VERIFIED-LIBRARY.md] *(to be added before public launch)*.

## Security

To report a vulnerability, email **cha-security@baisoln.com**. See [SECURITY.md](SECURITY.md).

## Roadmap

See [/home/skadam/.claude/plans/i-have-been-adviced-hashed-lecun.md] for the WS-A → WS-D rollout plan.
