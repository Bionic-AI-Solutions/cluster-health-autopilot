# CHA Consolidated Roadmap

**Last updated:** 2026-06-11 (P4.2 roadmap reconciliation added)

This document supersedes the per-phase plans for forward planning purposes —
the per-phase docs (`2026-06-04-cha-phase-0-and-1.md` through
`2026-06-09-cha-phase-3-master.md`) remain as historical records of what was
designed and shipped.

---

## ✅ Shipped (live on production cluster as of 2026-06-09)

### OSS (`cluster-health-autopilot`) — running v1.22.2

| Phase | What | Tag |
|---|---|---|
| Phase 0 | Bootstrap | v1.0.0 |
| Phase 1 | Drift coverage (initial analyzers + probes) | through v1.18.x |
| Phase 1.B | Placeholder substitution across analyzers | v1.18.3 / v1.19.0 / v1.20.1 |
| Phase 1.6 trigger M1 | Initial expanded `watchedGVRs` | v1.6.0 |
| Phase 2.B.6 | OSS Slack class-button render row | v1.21.0 |
| Phase 2.B.9 | `Silence.spec.matcher.messagePattern` | v1.21.0 |
| Phase 2.E | `DisruptionDrift` analyzer (PDB / Indexed-Job / ResourceQuota) | v1.21.0 |
| Phase 2.F | `spec.ai.replicas` HA aiwatch + leader-election | v1.21.0 |
| Phase 2.G | `ai.metrics.*` chart values (Service/ServiceMonitor/PrometheusRule/Grafana) | v1.21.0 |
| Phase 2.H | `spec.ai.digestPinAttestation` chart wiring | v1.21.1 |
| Phase 3.D | `spec.ai.metrics` + `spec.ai.llmProposer` typed CR fields + operator reconciler | v1.22.0 |
| Phase 3.E | `OOMKillRecurrence`, `PVOrphan`, `CronJobStuck` analyzers | v1.22.0 |
| Phase 3.E fixes | `GVRPV` capture + `persistentvolumes` RBAC | v1.22.1, v1.22.2 |

### CHA-com (`cha-com`) — running v1.18.0

| Phase | What | Tag |
|---|---|---|
| Phase 1 | Initial paid-tier extensions | through v1.14.0 |
| Phase 2.A | RAG memory READ + revert detection + per-cycle observability | v1.15.0 |
| Phase 2.B-H | Approve+remember class + confidence + LLM proposer + lease elector + Prometheus + attestation | v1.16.0 |
| Phase 3.B | `Forge.MergePullRequest` + `AutoMergeGate` interface + concrete gate | v1.17.0 |
| Phase 3.C | `TargetHistoryRetriever` + `<target_history>` prompt block | v1.18.0 |
| Phase 3.F | `cha-com audit-bundle` subcommand | v1.18.0 |

### Live cluster smoke (2026-06-09)

- HA aiwatch: 2 replicas, lease-elected leader, failover ~20s
- `/metrics` + `/healthz` endpoints serving 8 `cha_*` metric families
- Attestation: 64-byte Ed25519 priv mounted at `/etc/cha/attestation/`
- DigestPinAttestation chart wiring honored by operator
- CronJobStuck firing on `bionic-diagnose` / `bionic-remediate` /
  `kube-system/namespace-cleanup*` (never-succeeded) + suspended
  `pg/pg-password-*`
- PVOrphan: firing after RBAC fix shipped in v1.22.2
- OOMKillRecurrence: silent (no qualifying pods — correct)

---

## 🚧 Outstanding work — in order

### 1. Phase 3.B production wiring (~1h)

**Gap:** `digestPinAutoMergeGate` exists as a fully-tested impl but is
never CONSTRUCTED at runtime. `digestPinFlags.autoMergeGate` is always
nil → `DigestPinProposer.AutoMerge` is nil → auto-merge never fires.

**What's needed:**

- In `cmd/cha-com/digest_pin_wiring.go`, when `d.autoMerge == true`:
  - Look up the breaker (already constructed in autonomy wiring)
  - Look up the policies retriever (already constructed)
  - Look up the classHistory retriever (already constructed)
  - Confirm `att != nil` for attestationConfigured
  - Set `d.autoMergeGate = &digestPinAutoMergeGate{...}`
- Add `--digest-pin-min-success-rate` flag (default 0.95)
- Existing `proposer.DigestPinProposer.AutoMerge` field already plumbed

**Tests:** 1 wiring test asserting the gate is constructed when the flag
is on + all deps are present; nil otherwise.

**Release:** `cha-com:1.19.0`.

---

### 2-8. Trigger-expansion roadmap M1-M7 — ✅ MOSTLY SHIPPED (status corrected 2026-06-11)

These come from `docs/design/2026-05-trigger-expansion-roadmap.md`. **M1–M6 shipped** (bundled in v1.23.0–v1.24.1, NOT the speculative one-per-minor schedule below); **M3 shipped with scope-shrink** (see DELTA); **M7 leftovers remain**.

| M | Scope | As-shipped status |
|---|---|---|
| **M1** | Add `Ingress`, `DaemonSet`, `HPA`, `ArgoCD Application` to `watchedGVRs` + RBAC + Helm gates + tests | ✅ **v1.23.0** (PR #179); KEDA `ScaledObject` added v1.24.0 (PR #182) |
| **M2** | Kong / HPA / ArgoCD / Velero probes | ✅ **v1.7.0** (shipped earlier than this table assumed) |
| **M3** | New probe `GPUNodes{}` + new analyzer `LogPatternMatcher{}` | 🚧 **v1.23.0 — SCOPE-SHRINK.** `LogPatternMatcher` shipped as an **Events scanner**, not a pod-log tailer (vLLM `cumem_allocator.cpp:145` NOT catchable). `GPUNodes` shipped Ready/cordoned/zero-allocatable only (no device-plugin/driver-drift/DCGM). → **M3-v2 + GPUNodes-v2 open** (see Q3 forward plan) |
| **M4** | Endpoints probe optional L7 mode (HTTP, not just TCP) | ✅ landed in the M1–M6 bundle |
| **M5** | Class C: PrometheusRule / Alertmanager consumer | ✅ **v1.23.1** (PR #180); operator-CR `triggers.prom` v1.24.0 |
| **M6** | Class E: External webhook receiver | ✅ **v1.23.1** (PR #180); operator-CR `triggers.webhook` + Service v1.24.0 |
| **M7** | Operator-mode refactor (controller-runtime) — largely done in Phase 1c; this completes leftovers | 🔭 **leftovers open → OSS v2.0.0** |

**RBAC philosophy carried through M1-M7:** each new GVR is gated by a
Helm `watcher.gvrs.<kind>=true` value defaulting to true ONLY where the
GVR is broadly safe (no enterprise-only CRDs). Each new probe gets a
`CHA_PROBE_<NAME>=off` env-var opt-out, matching the analyzer pattern.

---

## 🧭 Roadmap reconciliation (P4.2, 2026-06-11)

Three roadmaps had been diverging:
- **(a)** `CHA-com/docs/competitive/roadmap.md` — competitive-informed H1/H2/H3 strategy (Datadog connector, knowledge graph, public benchmark, FedRAMP pack, per-tenant RBAC, multi-account federation, etc.).
- **(b)** this engineering-only consolidated roadmap.
- **(c)** the website roadmap — now **generated from CHANGELOGs**; left as-is (source of truth = the changelogs this repo + CHA-com ship).

**This doc (b) is now the single forward plan.** `CHA-com/docs/competitive/roadmap.md` (a) is to be marked **SUPERSEDED** with a pointer here (see "CHA-com doc action" note at the bottom of this section). Below, every competitive-roadmap H2/H3 item is dispositioned KEEP (with a milestone/quarter) or DROP (with reason). Items the in-flight remediation (Phase **P6**, 2026-06) is actively building are marked **IN PROGRESS** so they are not lost.

### Disposition of competitive-roadmap (a) items

| Competitive item (horizon) | Disposition | Where it lands |
|---|---|---|
| Datadog connector (H2) — read-only Events + Monitor state as trigger source | **KEEP** | Q4 2026; extends the shipped M5 Alertmanager-consumer pattern |
| Lightweight knowledge graph (H2) — in-memory K8s+cloud topology, no Neo4j | **KEEP** | Q4 2026; feeds LLM tiers richer context (compounds with cluster-knowledge RAG) |
| Public benchmark (H2) — "MTTR with remediation" + "% incidents resolved without human page" | **KEEP** | Q3 2026 (strategic, in forward plan below) — sets eval terms in our favor |
| FedRAMP / sovereign pack (H2) — SBOM, signed images, air-gap, no-egress proof | **KEEP, split** — the SBOM + cosign-signed-images slice is **IN PROGRESS (Phase P6)**; the full FedRAMP evidence package + SOC2 track is Q3/Q4 strategic |
| Per-tenant RBAC (H3) — namespace-scoped CHA instances, tenant isolation | **KEEP** | Q4 2026+; prerequisite for multi-tenant SaaS (currently "not formally planned" below) |
| Multi-account / multi-subscription / multi-project cloud federation (H3) | **KEEP** | Q4 2026+ (cloud-probe v2); the single-account caveat is real but ICP-gated |
| Multi-cluster federation (H3) — ArgoCD ApplicationSet + cross-cluster DriftReport hub | **KEEP as Federation MVP — IN PROGRESS (Phase P6)**; full hub aggregation Q4. Substrate ready (`pkg/rag.Entry.ClusterID`), see Parked §3.A |
| Hosted playground / sandbox (H1) — kind + synthetic drift + LocalStack cloud sub-account | **KEEP as Hosted playground — IN PROGRESS (Phase P6)** |
| Hosted dashboard (implied by SaaS surface; competitive "self-serve hosted") | **KEEP — IN PROGRESS (Phase P6)** |
| Jira / ServiceNow connectors (paid ticketing sinks, ticketing doc M3/M4) | **KEEP — IN PROGRESS (Phase P6.3 / P6.4)** |
| Loki / OTLP audit sinks (observability ingestion + audit export) | **KEEP — IN PROGRESS (Phase P6)** |
| Comparison-engine connectors (H3) — Splunk, Grafana Loki, NewRelic, Dynatrace read-only | **PARTIAL KEEP** — Loki via the P6 audit-sink work; Splunk/NewRelic/Dynatrace **DROP for now** (no validated demand; revisit per-objection when a buyer names one) |
| Datadog/Prom/Grafana/Splunk observability *ingestion* as trigger sources (gap §2) | **KEEP (Datadog)** above; rest **DROP** until asked |
| Tighten security claims / "Security" top-nav page (H1) | **KEEP** — GTM/docs track, not engineering; folds into the SOC2/FedRAMP evidence narrative |
| "Deliberately NOT do" items (RCA-accuracy benchmark vs CloudOpsBench, etc.) | **DROP (by design)** — unchanged; we do not chase Tracer's integrity-gate benchmark |

> **CHA-com doc action (not edited in this OSS commit):** mark `CHA-com/docs/competitive/roadmap.md` as **SUPERSEDED**, add a header pointer to this consolidated roadmap, and note that forward planning now lives here while the competitive *analysis* (wedges, honest assessment) remains a historical strategy record.

---

## 🔭 Q3 2026 forward plan (genuinely-open items)

Honest list of what is actually open — verified against git/CHANGELOG/code on 2026-06-11. Excludes everything already in "✅ Shipped" above.

### Restore-scope follow-ups (from the P4.1 design-doc audit)
- **M3-LogPatternMatcher-v2** — real **pod-log tailing**. As shipped (v1.23.0) `LogPatternMatcher` scans K8s **Events only** (`internal/diagnose/log_pattern_matcher.go`); the motivating vLLM `cumem_allocator.cpp:145` pattern (and `CUDA out of memory`, kernel-OOM, `i/o timeout`) live in container stdout and are NOT catchable. Needs `Pod.GetLogs`/`tailLines` + the `watch-logs` annotation from the design.
- **GPUNodes-v2** — restore the dropped depth. As shipped (v1.23.0) `internal/probe/gpu_nodes.go` checks only Ready/cordoned/zero-allocatable; the design's nvidia-device-plugin Pod presence, **driver-version drift**, **DCGM ECC/XID**, and per-GPU >95% memory are un-shipped.
- **Cloud M3 — cloud-aware AI tier** — dropped silently from the cloud-probe framework. The 30 cloud probes shipped (v1.8.0) but the T0/T1/T2 cloud-context AI enrichment (`docs/design/2026-05-cloud-probe-framework.md` §6 M3) has no code and no CHANGELOG entry. The shipped AI tiers carry K8s context only.
- **Ticketing M2 — resolve-on-clear** — auto-close the OpenProject/Jira/ServiceNow ticket when the underlying finding clears. M1 (OpenProject) shipped (PR #59); M2 never started. Tracked as **P6.5**.

### Phase P6 features (IN PROGRESS, 2026-06)
- **Jira sink (P6.3)** + **ServiceNow sink (P6.4)** — paid ticketing connectors (ticketing-doc M3/M4).
- **Loki / OTLP audit sinks** — structured audit export beyond the JSONL `--ai-audit-log`.
- **SBOM + cosign-signed images** — supply-chain attestation (FedRAMP-pack slice).
- **Hosted dashboard** + **hosted playground** — self-serve surfaces (competitive H1).
- **Federation MVP** — cross-cluster DriftReport aggregation (competitive H3 first slice).

### Operator / packaging
- **Phase 3.B production wiring** — construct `digestPinAutoMergeGate` at runtime (Outstanding §1 above; ~1h; cha-com:1.19.0).
- **OLM / OperatorHub publish** — the OLM bundle + bundle-smoke shipped (Phase 1c); the public **OperatorHub.io submission** is not yet done.
- **Trigger-expansion M7** — operator-mode leftovers → OSS v2.0.0.

### Strategic (GTM-adjacent, mostly Q3–Q4)
- **Public benchmark / eval** — "MTTR with remediation" + "% incidents resolved without human page" (competitive H2).
- **BYO-LLM matrix** — validate the AI tier against multiple LLM providers/endpoints beyond the in-cluster Qwen.
- **SOC2 track** + the broader **FedRAMP / sovereign evidence package** (competitive H2, beyond the P6 SBOM/cosign slice).
- **Community bootstrap** — Discord, contributor page, Awesome-X listing (competitive H1/§2).
- **Knowledge-graph** + **Datadog connector** + **per-tenant RBAC** + **multi-account/multi-cluster federation** — Q4 2026+ per the reconciliation table above.

---

## 🅿️ Parked (by scope decision)

### Phase 3.A — Cross-cluster RAG federation

ICP-gated until first paying customer runs >1 cluster. Substrate is
ready: `pkg/rag.Entry.ClusterID` partitions outcomes. Federation hook
is one wider-list call. Est. ~half-day when triggered.

### Web UI

Explicitly anti-goaled in the Phase 2 master plan ("Slack-only remains
the contract"). Carried forward — no UI work planned.

---

## 🌐 Out of code scope (product/GTM)

Per `project_cha_gtm` memory:

- Day 60: paid landing page
- Days 14-90: anonymized run-log collection
- Days 60-90: first paying customer
- Month 9: 3 paying customers (north star)

No engineering work; product/marketing track.

---

## 🔮 Not formally planned

- Multi-tenant CHA-com SaaS (one binary, N customer clusters)
- Slack-native onboarding (slash commands to manage CHA from Slack)
- Cloud-hosted CHA SaaS offering

These would each need their own master plan before execution.
