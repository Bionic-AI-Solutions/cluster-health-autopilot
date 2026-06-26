# Phase 2.G — Grafana Dashboards for Srenix Enterprise

**Status:** active sub-plan; execution deferred to a focused observability session.

**Parent:** [2026-06-07-srenix-phase-2-master.md](2026-06-07-srenix-phase-2-master.md)

---

## Goal

Operators currently observe Srenix Enterprise behavior via:
- Watcher stdout (the cycle log lines: `[cycle=N]`, `outcomes:`, `policies:`, `llm-proposer:`)
- K8s Events in the install namespace (the audit sink)

Phase 2.G ships Grafana dashboards backed by Prometheus metrics so operators see the same signals as time-series + alerting.

## Anti-goals

- Don't build a custom log shipper. If Loki isn't deployed, we use Prometheus. If neither, dashboards can't ship.
- No new persistence layer. Metrics are in-memory + scraped.

## Sub-tasks

### 2.G.1 — Prometheus instrumentation

Per signal, the watch loop already emits, add a `prometheus.Counter` or `Gauge`:

| Metric | Type | Labels | Source |
|---|---|---|---|
| `srenix_cycle_total` | counter | — | tick header |
| `srenix_diagnostic_total` | counter | source, severity | per fresh diag |
| `srenix_outcome_total` | counter | verdict, delivery | outcomes line |
| `srenix_outcome_revert_total` | counter | source | revert observer |
| `srenix_policy_active` | gauge | — | policy counter |
| `srenix_policy_muted_total` | counter | — | policy counter |
| `srenix_llm_proposer_total` | counter | outcome | llm-proposer line (outcome=succeeded/refused/invalid/errored/rejected) |
| `srenix_autonomy_decision_total` | counter | result | autonomy.Consider (result=applied/declined/skipped/error) |
| `srenix_breaker_open` | gauge | — | circuit breaker |
| `srenix_confidence_histogram` | histogram | source, action_kind | DecideAutonomyWithInputs |

### 2.G.2 — /metrics endpoint on watcher + approval-server

- Add `prometheus.Handler` on `:9090/metrics`
- Helm template adds a `ServiceMonitor` (Prometheus Operator) OR a `PodMonitor` per deployment

### 2.G.3 — Grafana dashboard JSON

One dashboard `srenix-enterprise-overview.json` with rows:
- **Throughput** — cycle rate, diagnostic rate, outcome rate
- **Autonomy** — auto-apply success vs decline; confidence distribution
- **LLM Proposer** — attempts vs success; refuse-rate trend
- **Class Policies** — active count, mute count
- **Reverts** — revert counter; revert rate by source
- **Breaker** — open/closed timeline

### 2.G.4 — Alertmanager rules

`PrometheusRule` CR shipping a small set of canary alerts:
- `chaWatcherStuck` — `srenix_cycle_total` not increasing for 5m
- `chaBreakerOpen` — `srenix_breaker_open == 1` for 10m
- `chaAutonomyRejectionSpike` — sustained `srenix_llm_proposer_total{outcome="rejected"} / total > 0.5` for 30m

### 2.G.5 — Local verify + screenshots

- Install dashboard JSON via Grafana Provisioning ConfigMap
- Trigger a manual approve click → verify counters increment + panel renders
- Document in CHANGELOG

## Risk

- **Metric label cardinality** — `source` × `action_kind` could explode. Cap at the analyzer enum (currently ~10 sources) + closed action kind list.
- **No Prometheus on target cluster** — operator's bring-your-own. Document the dependency in Helm chart prereqs.
