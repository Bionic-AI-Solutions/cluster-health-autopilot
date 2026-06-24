# Cluster Health Autopilot — One-Page Brief

*Self-healing operations for Kubernetes — detect, fix, re-verify, report.*

---

## The problem

Every Kubernetes cluster accumulates a tail of silently-degraded state — a Secret-key rename nobody noticed until the next pod eviction, an ExternalSecret quietly failing to sync for weeks, a CronJob that locked itself out a month ago. Existing dashboards (Datadog, Grafana, Prometheus alerts) detect *metric* moves but are blind to this *configuration drift* class across the Vault → ExternalSecret → Deployment chain.

## The wedge

**Zero-trust diagnose, no install, no RBAC, no write access.**

```sh
cha diagnose --snapshot ./your-kubectl-export.json
```

Names the exact Secret + missing key + consuming Deployment + owning ExternalSecret in one line. Run it on your laptop in 30 seconds. Nothing leaves your machine.

## The product

A single Helm chart. Three operational components:

- **diagnose CronJob** — read-only, daily, posts the daily digest to `#healthinfo`. Always enabled.
- **remediate CronJob** — opt-in, runs whitelisted auto-fixers.
- **watcher Deployment** — event-driven; reacts within seconds of a Kubernetes event instead of waiting for a cron tick. Optionally runs fixers each cycle (`--remedy`). Includes Layer-1 flake suppression (one in-cycle retry + 2-of-2 streak before CRITICAL) and Layer-2 Investigator (read-only root-cause classification on critical findings).

A **13 MB distroless container image**. Runs nonroot. Negligible footprint (<100 m CPU request, <100 MB RAM). No inbound traffic; outbound only to the Kubernetes API, optional Vault, optional Alertmanager, optional Slack webhooks.

| What it sees | What it does |
|---|---|
| Ceph health (CephCluster CRD) | **Auto-fix (whitelisted, audited):** |
| CloudNativePG / Spilo (Patroni) Postgres | • delete stale `Error`/`Failed` pods |
| Nodes, PVCs | • delete frozen CronJob Jobs |
| Critical Services (configurable) | • rollout-restart wedged ReplicaSets |
| Public endpoint reachability — every Ingress host **auto-discovered** (v1.2) | • delete terminally-failed cert-manager requests |
| Failing ExternalSecrets + Vault path probe | • repoint Ingress to correct TLS Secret (opt-in, v1.3) |
| Pods stuck in CCE on missing Secret keys | **Diagnose (never acts):** |
| Workloads referencing unprovisioned Secrets | • Secret-key drift across the chain (proactive + reactive) |
| cert-manager Certificate state | • ESO sync errors with `t6-apps/` Vault path hint |
| Ingress + TLS Secret cross-reference | • Cert expiry within 14 days / ACME rate limits |
|  | • ImagePullBackOff with 401/auth signal |
|  | • Ingress.tls.secretName vs cert-manager Certificate target mismatch (v1.3) |
|  | **Investigate on critical (read-only, v1.5):** |
|  | • DNS root cause (no such host / slow CoreDNS) |
|  | • TLS expired / SAN mismatch / fallback cert |
|  | • Transient-recovery (follow-up probe succeeded) |
|  | • ExternalSecret / Certificate / Secret deep-dive |

**Alert routing**: Alertmanager-as-hub (recommended) — CHA posts active issues
to `/api/v2/alerts` every cycle. AM handles dedup, silencing, and fan-out to
all configured receivers. Fallback to direct three-channel Slack
(`#ceph-alerts` / `#ceph-critical` / `#healthinfo`).

## RBAC discipline

Read role and Write role are **separate** ClusterRoles. The Write (remediator) role grants exactly:
`pods/delete`, `jobs/delete`, `deployments/patch`, `certificaterequests/delete`, `orders/delete`. Never Secret/ConfigMap/CRD writes.

Nine platform namespaces are always skipped — `kube-system`, `kube-public`, `kube-node-lease`, `rook-ceph`, `vault`, `external-secrets`, `cnpg-system`, `calico-system`, `tigera-operator`. Enforced both **in code** AND **by RBAC**. The fix list is the source code; an SRE can audit every action the tool will ever take in one afternoon.

## Zero LLM in the hot path

Every probe, analyzer, and fixer is deterministic Go. Same input → same diagnosis, every time, auditably. The OSS Layer-2 Investigator is rule-based (no LLM); each rule pattern-matches the failure mode and runs a closed-enum set of read-only tools (DNS / HTTP / TLS / Describe / Events). No LLM call on cluster data in OSS. No customer state leaves the cluster.

The paid CHA-com binary adds AI tiers (T0–T3) and an optional LLM-backed Investigator, each gated by approval flows + audit + RBAC ceilings inherited from the OSS engine. AI is opt-in, never on the critical path. (See [docs/AI_USAGE.md](AI_USAGE.md) and [docs/AI_TIERS.md](AI_TIERS.md) for the full position.)

## Pricing (open-core)

| Component | License | Tier |
|---|---|---|
| CLI engine + all probes + analyzers + fixers + 30 cloud probes + Helm chart + operator | Apache 2.0 | Free / OSS |
| **CHA-com AI tiers** (T0 narration, T1 fix proposals, T2 multi-step plans, T3 Vault runbooks) + approval-server with signed click-to-fix URLs | Commercial | AI SRE (per-cluster/mo) |
| **CHA-com Enterprise** — above + Jira/ServiceNow ticketing, multi-cluster federation, RAG memory, confidence-gated auto-merge, Loki/OTLP audit sinks | Commercial | Enterprise (per-cluster/mo) |
| **Federal/Sovereign** — air-gap installer, SBOM + signed images, dedicated security eng, SLA | Commercial | Contact sales |

## Real-world validation

Built on a 6-node production GPU/AI cluster running 30+ services. The default catalog was derived from incidents that were already in flight when the project started — including a SIP server stuck 33 days, a CronJob frozen 26 days, and an ExternalSecret silently failing for 6 days. All three are now caught on the next scheduled tick.

## Want the demo?

The 30-second snapshot demo + curated example fixture lives in [`examples/sample-cluster/`](../examples/sample-cluster/). One git clone + one command shows three live diagnostics on a hand-crafted snapshot — without a Kubernetes cluster on your laptop.

For the deeper version of what each fixer does and the failure modes it catches, see [docs/FAILURE_MODES.md](FAILURE_MODES.md).
