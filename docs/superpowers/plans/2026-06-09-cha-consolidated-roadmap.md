# CHA Consolidated Roadmap

**Last updated:** 2026-06-09

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

### 2-8. Trigger-expansion roadmap M1-M7 (paired releases)

These come from `docs/design/2026-05-trigger-expansion-roadmap.md`. M1
partially shipped in v1.6.0; the remaining work below.

| M | Scope | Effort | Tag |
|---|---|---|---|
| **M1** | Add `Ingress`, `DaemonSet`, `HPA`, `ArgoCD Application` to `watchedGVRs` + RBAC + Helm gates + tests | ~1d | OSS v1.23.0 |
| **M2** | New probe `KongRoutes{}` — verify each Kong-managed Ingress resolves to ≥1 ready endpoint + KongPlugin/Consumer refs | ~2d | OSS v1.24.0 |
| **M3** | New probe `GPUNodes{}` + new analyzer `LogPatternMatcher{}` | ~3d | OSS v1.25.0 |
| **M4** | Endpoints probe optional L7 mode (HTTP probe, not just TCP) | ~2d | OSS v1.26.0 |
| **M5** | Class C: PrometheusRule / Alertmanager consumer | ~3d | OSS v1.27.0 |
| **M6** | Class E: External webhook receiver | ~2d | OSS v1.28.0 |
| **M7** | Operator-mode refactor (controller-runtime) — largely done in Phase 1c; this completes leftovers | ~1w | OSS v2.0.0 |

**RBAC philosophy carried through M1-M7:** each new GVR is gated by a
Helm `watcher.gvrs.<kind>=true` value defaulting to true ONLY where the
GVR is broadly safe (no enterprise-only CRDs). Each new probe gets a
`CHA_PROBE_<NAME>=off` env-var opt-out, matching the analyzer pattern.

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
