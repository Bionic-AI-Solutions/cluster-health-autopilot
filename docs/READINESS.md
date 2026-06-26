# Readiness Assessment — Agentic SRE v1.6.0

This document is the srenix team's readiness assessment of the **current
shipping release (v1.6.0)** for design-partner deployment and pilot
customer use. Read alongside [ADVERSARIAL_ANALYSIS.md](./ADVERSARIAL_ANALYSIS.md)
and [design/2026-05-final-adversarial-review.md](design/2026-05-final-adversarial-review.md).

**v1.6.0 readiness delta vs v1.5.2.** Sprints 1-4 closed 22 of 23 items
from the 2026-05-22 adversarial review (95%). The remaining item is
the M2+ probe slice (Kong / HPA / ArgoCD / Velero) — roadmap-class,
not credibility-class. Headline gains:

- **Fixer safety**: all 4 default fixers (+1 opt-in) now consult GitOps
  detection (Argo / Flux / Helm) and operator-intent gates (`spec.paused`,
  `spec.suspend`) before mutating. Closes the fight-loop class.
- **Probe coverage**: 6 net-new probes (NodePressure, DaemonSets,
  PendingPods, CrashLoopBackOff, ETCD, FailedMounts). Total: 12 probes.
- **Critical-workload list**: env + annotation auto-discovery —
  Bionic-cluster defaults no longer required.
- **HA**: Lease-based leader election. Two-replica deployments now
  safe; the chart's `replicas: 1` default still works unchanged.
- **AI safety (paid)**: patch-payload allow-list, independent investigation
  budget, cold-start bucket mitigation, event-message secret scrubbing,
  audit hash chain.
- **Tests**: OSS `internal/fix/` 36→57; `internal/probe/` 0→70 (incl. 6
  new probes); `internal/watcher/` 2→31. Srenix Enterprise 32→94. Chart-level
  helm-unittest with 19 cases.

The original [Vault → Pod Drift solution brief](./vault_pod_drift_solution_brief.docx.pdf)
defined a five-layer detection stack (L1–L5). v0.2 closed all five gaps; v0.3
through v0.9.5 hardened those layers and added the operational features (real-
time mode, multi-channel routing, alert hub integration, auto-remediation)
needed for production deployment. v1.0 introduced the AI tier (Srenix Enterprise).
v1.2–v1.5 added auto-discovery of Ingress hosts, a TLS cert/secret-mismatch
analyzer with opt-in fixer, Layer-1 flake suppression on probe failures,
and a Layer-2 read-only Investigator that attaches root-cause hints to
CRITICAL findings.

## 1. Brief's five-layer stack — coverage matrix

| Layer | Brief's intent | Coverage | How |
|---|---|---|---|
| **L1** Vault stale-Ready window | Catch Vault edits BEFORE the ESO controller refreshes | ✅ | `VaultPathMissing` analyzer queries Vault directly |
| **L2** Failing ExternalSecret detection | Catch ESOs reporting `Ready=False` | ✅ | `FailingExternalSecrets` + `t6-apps/` hierarchy hint |
| **L3** Failing Pod with bad Secret ref | Catch CCE pods, name the missing key | ✅ | `SecretKeyMissing` analyzer with owner-chain resolution |
| **L4** Proactive bipartite-graph drift | Walk Deployment+SS env refs vs. live Secret keys, flag drift before pod restart | ✅ | `ProactiveSecretKeyCheck` (env + envFrom) + case/format variant hint |
| **L5** kubectl-queryable diagnostic objects | Each diagnostic visible as a CR with status + history | ✅ | `DriftReport` CRD + reconciler |

All five layers closed. Tightened in v0.3 (mixed-provider filter,
outage dedup, envFrom walk) and operationalized in v0.9.x. v1.4 added
in-cycle retry + N-of-M streak suppression on top of the L1 surface
to reduce transient-flake noise; v1.5 added a Layer-2 read-only
Investigator that runs on CRITICAL findings only.

---

## 2. Capabilities beyond the original brief (v0.5 – v1.5.2)

The brief targeted detection. Production deployment requires operational
features the brief did not specify. These shipped in v0.5 – v1.5.2:

| Capability | First in | What it does |
|---|---|---|
| **Auto-remediation** | v0.5 | Whitelisted fixers run in `srenix remediate --live` (opt-in) — `StaleErrorPods`, `StuckJobsWithBadSecretRef`, `StuckRSPods` |
| **Slack reporting** | v0.5 | Formatted attachment with component status + diagnostics |
| **UnprovisionedSecret analyzer** | v0.9.1 | Detects workloads referencing Secrets with no ExternalSecret + suggests canonical Vault path |
| **CertExpiry analyzer** | v0.7 | cert-manager Certificate: not-Ready, expired, or expiring within 14 days |
| **ImagePullAuth analyzer** | v0.8 | ImagePullBackOff with auth-signal event messages (401, unauthorized, denied) |
| **StuckCertificateRequests fixer** | v0.9.1 | Deletes terminally-failed CertificateRequest + ACME Order CRs |
| **Event-driven Watcher mode** | v0.9.0 | Long-running Deployment reacts within seconds (10s debounce) instead of waiting for cron tick |
| **DriftReport seeding for Slack dedup** | v0.9.0 | Watcher seeds its seen-map from existing DriftReport CRs on pod startup — no Slack flood after rolling update |
| **Watcher `--remedy`** | v0.9.0 | Fixers run after every diagnose cycle, post-fix state reported |
| **Endpoint reachability probe** | v0.9.x | HTTP(S) GET against canonical hostnames — catches TLS faults, missing Kong routes, DNS failures |
| **Three-channel Slack routing** | v0.9.4 | `#ceph-alerts` (Srenix-fixed) · `#ceph-critical` (needs human) · `#healthinfo` (daily digest) |
| **Daily digest** | v0.9.4 | `--format=daily` reads DriftReport history; reports new/persistent/resolved with age annotations |
| **Alertmanager-as-hub** | v0.9.5 | Direct POST to `/api/v2/alerts` every cycle; AM handles dedup/silencing/fan-out to all configured receivers |
| **Fixed Alertmanager dispatch** | v0.9.5 | `fsGroup` patch on AM pod resolves 3-month-old `permission denied` on nflog/silences |
| **AI tier (Srenix Enterprise)** | v1.0.0 | Recommendation-only LLM enrichment + signed JWT approval-server; OSS engine remains AI-free |
| **Ingress auto-discovery** | v1.2.0 | Endpoint probe enumerates every Ingress host (read-only RBAC), opt-out via `srenix.ai/probe-disable: "true"`; protected namespaces skipped; legacy `IngressCoverage` analyzer removed (pattern-matched on hostnames) |
| **TLSSecretMismatch analyzer** | v1.3.0 | Compares the x509 cert in `Secret.tls.crt` against the target name of the owning `Certificate` CR; read-only |
| **TLSSecretMismatch fixer (opt-in)** | v1.3.0 | Patches `Ingress.spec.tls[].secretName` to the correct Secret; default off; adds the narrow verb `networking.k8s.io/ingresses [patch]` only when enabled; skips protected namespaces AND GitOps-managed Ingresses |
| **Layer-1 flake suppression** | v1.4.0 | Single in-cycle retry (1.5× timeout) on flake-class errors + N-of-M in-memory streak counter — sub-threshold streaks surface as Warning, ≥ threshold escalates to Critical; restart loses the streak by design |
| **Layer-2 Investigator (rule-based, OSS)** | v1.5.0 | `pkg/ai.Investigator` runs on CRITICAL findings against a closed-enum `pkg/ai.Environment` of read-only tools (`DNSLookup`, `HTTPProbe`, `TLSInspect`, `Describe`, `GetEvents`); 20s hard wall-clock cap per cycle; soft-fail per investigation; attaches a root-cause hint to the DriftReport (additive, never replaces the original finding) |
| **Layer-2 Investigator (LLM-backed)** | v1.5.x (Srenix Enterprise) | Same closed-enum `Environment` surface; max 6 tool calls/investigation; same wall-clock cap; rate-limited per the existing AI-tier limiter |

---

## 3. New code surface across the v0.x line

| Surface | Tests | Risk |
|---|---|---|
| `internal/diagnose/` — 7 analyzers (`cert_expiry`, `failing_externalsecrets`, `image_pull_auth`, `proactive_secret_key_check`, `secret_key_missing`, `tls_secret_mismatch`, `unprovisioned_secret`, `vault_path_missing`); the v0.9 `ingress_coverage` analyzer was removed in v1.2 | 35+ | low — pure read-only; privacy contract enforced by code shape |
| `internal/probe/` — 6 probes (Endpoints probe gained Ingress auto-discovery in v1.2, in-cycle retry + streak counter in v1.4) | 18+ | low — read-only; HTTPS GETs respect cluster egress controls; egress surface widened in v1.2 but bounded by existing read-only RBAC |
| `internal/fix/` — 4 default fixers + 1 opt-in (`stale_error_pods`, `stuck_jobs`, `stuck_rs_pods`, `stuck_cert_requests` default; `tls_secret_mismatch` opt-in) | 22+ | medium — type-system gated to live mode; whitelist-only; opt-in fixer adds `networking.k8s.io/ingresses [patch]` only when enabled |
| `internal/watcher/` — long-running event-driven engine | 8 | medium — new persistent attack surface; reviewed in ADVERSARIAL §4.3 |
| `internal/report/` — `routing.go` + `daily.go` + `alertmanager.go` | 12 | low — outbound HTTP only; no inbound listeners |
| `pkg/ai/` — `Investigator` interface, closed-enum `Environment`, rule-based investigator (OSS) | 10+ | low — read-only by interface contract; no mutation methods exposed; reuses watcher RBAC; reviewed in ADVERSARIAL §9 |
| `charts/.../crds/driftreports.yaml` (CRD) | n/a | low — v1alpha1, schema explicitly unstable |
| `charts/.../clusterrole-{reader,remediator,driftreport-writer}.yaml` | n/a | medium — Reader includes cluster-wide `secrets get,list,watch`; documented |

**Aggregate vs v0.1**: 100+ new tests, 1 new CRD, 3 ClusterRoles, 1 Deployment
(watcher), 1 optional Deployment (self-hosted runner), 2 CronJobs, plus the
optional Srenix Enterprise AI tier (approval-server Deployment + isolated SA).

---

## 4. Capability deltas vs. the original brief

| Brief capability | v0.1 | v1.5.2 |
|---|---|---|
| Detect Vault path deletion before pod restart | no | **yes** |
| Detect Vault key removal before pod restart | no | **yes** |
| Detect ExternalSecret/Vault drift with no error in ESO yet | no | **yes** |
| Detect Deployment env reference to missing K8s Secret key (pre-restart) | no | **yes** |
| Detect Deployment env reference to nonexistent K8s Secret (pre-restart) | no | **yes** |
| Detect workload referencing Secret with no ESO at all | no | **yes** (UnprovisionedSecret) |
| Surface diagnostics as queryable cluster objects | no (Slack/JSON only) | **yes** (DriftReport CR) |
| Diagnostic objects show first-observed / last-observed / observation count | no | **yes** (CRD `.status` subresource) |
| Auto-cleanup resolved diagnostics | no | **yes** (reconciler deletes CRs whose subjects no longer reported) |
| Run in zero-trust snapshot mode | yes | yes (Vault probe excluded by design in snapshot) |
| Run live with kubernetes-auth Vault role | n/a | **yes** |
| Run live with VAULT_TOKEN | n/a | **yes** (dev posture) |
| OSS Apache 2.0 engine | yes | yes |
| Helm chart with toggleable probes | yes (diagnose, remediate) | **yes** (+ DriftReport, vaultProbe, watcher, alertmanager, three-channel slack, endpoint probe) |
| Real-time event-driven mode | no | **yes** (Watcher Deployment) |
| Alert hub integration | no | **yes** (Alertmanager direct API) |
| Multi-channel routing | no | **yes** (3 channels) |
| Daily digest | no | **yes** (DriftReport-history aware) |

---

## 5. Gaps that remain

These are **not** addressed in v1.5.2 and are surfaced here so design
partners aren't surprised.

| Gap | Rationale | Target |
|---|---|---|
| Multi-Vault / multi-SecretStore aware analyzer | v0.3 filters by provider; doesn't query multiple Vault instances | v1.0 |
| Self-hosted DriftReport viewer | Currently kubectl + grep; a web UI is post-fundraise scope | v1.0+ (Fleet Console) |
| Trend / time-series storage | DriftReport `.status.observationCount` is closest thing | v1.0+ (Fleet Console scope) |
| Cross-cluster aggregation | Single-cluster scope; multi-cluster is the commercial wedge | post-fundraise |
| Kubernetes Operator (controller-runtime) | Phased plan documented; Phase 1 next sprint | next sprint |
| OLM bundle (OperatorHub publication) | Phase 3 of operator plan | Q3 2026 |
| `partial-object-metadata` Secret listing | Reduces large-cluster bandwidth on `ProactiveSecretKeyCheck` | WILL-FIX (ADVERSARIAL §3.1) |
| Leader election for HA watcher | Currently 1 replica + `Recreate` strategy | bundled with Operator migration |
| Authenticated Alertmanager POST | Operator-supplied NetworkPolicy is the recommended pattern today | when AM upstream adds auth |

---

## 6. Net readiness

> **Are we ready to take Srenix to a design-partner conversation?**

**Yes.** v1.5.2 closes every L1–L5 gap from the original brief and adds
the operational features (real-time mode, multi-channel routing,
Alertmanager hub, auto-remediation, Layer-1 flake suppression, Layer-2
read-only investigation) needed for production deployment.

The [adversarial analysis](./ADVERSARIAL_ANALYSIS.md) flagged zero
must-fix items, one will-fix item (Secret list bandwidth on very large
clusters), and six document items (all captured in SECURITY.md /
SETUP_GUIDE.md / values.yaml).

The honest disclosures we'd make to a design partner:

1. **CRD is v1alpha1.** We will change the schema before v1beta1.
   Consumer scripts should pin on `additionalPrinterColumns` rather
   than full JSON paths.
2. **Reader role grants cluster-wide Secret read.** The code-level
   privacy contract (analyzer iterates `for k := range secret.Data`
   only) is documented in the role manifest. Partners with strict
   data-governance constraints can disable `ProactiveSecretKeyCheck`
   and `UnprovisionedSecret`, then revoke the rule.
3. **Vault role scoping is the operator's responsibility.** The chart
   doesn't install a Vault role; the partner's security team must
   author one scoped to the paths their ExternalSecrets reference.
4. **Alertmanager API is unauthenticated by default.** Production
   deployments should front the AM API with NetworkPolicy. Documented
   in SETUP_GUIDE.md §5.
5. **Watcher continuous remediation widens blast radius vs cron.**
   Recommended posture: `watcher.remedy.dryRun=true` for the first
   week, then enable live remediation.
6. **TLSSecretMismatch fixer is opt-in and adds one verb.** Enabling
   it grants the watcher `networking.k8s.io/ingresses [patch]`. The
   patch surface is narrow (only `spec.tls[].secretName`), and the
   fixer skips protected namespaces and GitOps-managed Ingresses. The
   verb is not in the default Reader/Remediator ClusterRoles.
7. **Layer-2 Investigator widens egress surface modestly.** The OSS
   rule-based investigator can issue outbound DNS, HTTPS, and TCP+TLS
   handshakes from the watcher pod when a CRITICAL finding fires. No
   new RBAC verbs. Existing cluster egress firewall posture applies.
   Hard 20-second wall-clock cap per cycle.

---

## 7. Pre-launch checklist (per release)

- [x] v1.5.2 tag pushed
- [x] GoReleaser workflow green (multi-arch binaries + container images
      on `ghcr.io` and `docker4zerocool`)
- [x] Helm chart `1.5.2` install clean against a production cluster
- [x] Smoke test on production cluster — watcher running, active
      diagnostics flowing to DriftReport CRs + Alertmanager + Slack;
      Ingress auto-discovery enumerates the cluster's hosts; Layer-1
      streak suppression observed on flaky targets; rule-based
      Investigator attaches root-cause hints to CRITICAL findings
- [x] Srenix Enterprise paid binary tracks v1.5.2 OSS dep

## 8. What "ready" does NOT mean

- It does **not** mean we have product-market fit. We have a technically
  defensible product against the brief's scenario AND the operational
  needs of a small fleet.
- It does **not** mean we are SOC 2 ready. SOC 2 is post-fundraise scope.
- It does **not** mean we have a paying customer. Design-partner outreach
  is current scope.
- It does **not** mean v1.5.2 is feature-complete. The operator
  architecture (controller-runtime) is the next major shape change;
  v1.5.2 is the smallest srenix that can credibly run in production
  while shipping flake-tolerant probing and read-only root-cause
  investigation.
