# Cluster Health Autopilot (`cha`)

A self-healing operational layer for Kubernetes clusters: **detect → remediate → re-verify → report**, on a schedule, without dashboards or pagers.

> **Pre-launch — engineering preview.** This README will be the public face on launch day; treat its current contents as draft.

---

## What it does

`cha` runs a battery of probes against your cluster, applies a whitelist of known-safe fixes for recognized failure patterns, re-probes, and produces a single report listing fixes applied and any residual issues with precise remediation hints. It runs in two modes:

- **Zero-trust offline mode** — point it at a captured `kubectl get … -o json` snapshot. No install, no RBAC, no write permissions. Diagnose your cluster from your laptop in 30 seconds.
- **In-cluster live mode** — installed via Helm; runs as a CronJob with two narrowly-scoped ClusterRoles (read-only + tightly-bounded write); posts to Slack on a schedule.

## Quickstart — zero-trust diagnose

```sh
# 1. Capture a snapshot of your cluster (read-only)
kubectl get pods,events,nodes,pvc,deployments,replicasets,jobs,cronjobs \
  --all-namespaces -o json > snapshot.json
kubectl get externalsecrets,clusters.postgresql.cnpg.io \
  --all-namespaces -o json >> snapshot.json 2>/dev/null  # optional CRDs

# 2. Run cha against the snapshot — never touches your cluster
cha diagnose --snapshot snapshot.json
```

Sample output:
```
🔎 Secret `mcp/mcp-openproject-secrets` missing key `openproject-url`
   referenced by Deployment/mcp-openproject-server.
   Owning ExternalSecret: `mcp/mcp-openproject-secrets`.
   Action: add data/template entry exposing `openproject-url`,
           or remove the env reference if unused.

🔎 ExternalSecret `mail/mail-service-api-key` not Ready:
   cannot find secret data for key: "mail_service_api_key"
   Action: check Vault path / property names.
```

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
