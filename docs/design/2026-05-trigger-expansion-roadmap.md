# Design: Trigger-Class Expansion Roadmap

Status: **Draft / pre-implementation**
Tracked: planned for v1.6–v2.0
Author: investigation 2026-05-16

## Problem

CHA today reacts to **two trigger classes**:

- **A — Resource-change events** via `internal/watcher/watcher.go`, subscribed
  to 14 GVRs (Pod, Node, PVC, Event, Deployment, ReplicaSet, StatefulSet, Job,
  CronJob, ExternalSecret, CNPGCluster, CephCluster, Secret, Certificate)
- **D — Schedule** via the legacy `cha diagnose` cron mode and the watcher's
  `ResyncPeriod` fallback

Three more trigger classes are unaddressed:

- **B — Status-transition optimization** — today the watcher re-runs the *full*
  probe + analyzer stack on any GVR change. There's no fast path for "Node
  flipped to NotReady" that bypasses unrelated probes.
- **C — Prometheus threshold** — CHA *posts* to Alertmanager but never
  *consumes* an alert as a trigger. Drift that doesn't produce a K8s event
  (slow disk fill, error-rate creep, ECC error rate, cert-renewal failure
  caught only by a downstream PromQL alert) waits for the daily resync.
- **E — External webhook** — Vault rotations, ArgoCD `SyncSucceeded` /
  `SyncFailed`, cert-manager `Issued`, Cloudflare cert renewals — all happen
  outside CHA and there's no inbound channel for them to trigger a probe.

Asset-class coverage also has gaps even within the resource-event model:

| asset class | covered today | gap |
|---|---|---|
| Kong routes / KongPlugin | ✗ | no probe; `Ingress` GVR not in `watchedGVRs` |
| GPU health (nvidia-device-plugin, ECC, driver) | ✗ | no probe; `DaemonSet` GVR not watched |
| DNS / CoreDNS canary | ✗ | no probe |
| Workload log-pattern signals (e.g. vLLM `cumem_allocator.cpp:145`) | ✗ | no analyzer class; today's analyzers operate on K8s status, not pod logs |
| HPA / autoscaling | ✗ | no probe; `HorizontalPodAutoscaler` GVR not watched |
| ArgoCD `Application` health | ✗ | no probe; CRD not watched |
| Velero backup recency | ✗ | no probe |
| ResourceQuota utilization | ✗ | no probe |
| Vault server (seal status, audit volume) | ✗ | no probe (analyzers cover ESO downstream) |
| Endpoints L7 health | partial | endpoint probe does TCP+TLS, not HTTP 200 + body match |

## Goal

Expand CHA's trigger surface to all 5 classes and close the 10 asset-class
gaps above, organized into 7 incremental milestones that each ship behind a
feature flag and are individually opt-in.

## Non-goals

- **Replace the watcher.** The watch-driven A-class engine stays as the
  primary trigger; new classes augment it.
- **Replace Alertmanager.** Class C is *consumption* of existing PromQL
  alerts, not a competing alerting layer.
- **Make CHA a metrics scraper.** Probes still read from a `snapshot.Source`;
  if a probe needs Prometheus data it reads from the existing exporter.

## Trigger-class taxonomy

| class | event | example signal | implementation surface |
|---|---|---|---|
| **A** | K8s resource Create / Update / Delete | Secret rotated, Pod CrashLoopBackOff | `internal/watcher/watcher.go` informers (today) |
| **B** | K8s status-condition transition | Node `Ready` flips false, PVC `Bound` → `Pending` | new `internal/watcher/status.go` — filtered subscriber, fires only relevant probe subset |
| **C** | Prometheus threshold breach | `disk_used > 85%`, GPU ECC rate > 0, error-budget burn | new `internal/trigger/prom.go` — Alertmanager `/api/v2/alerts` consumer |
| **D** | Cron / resync ticker | full diagnose pass every 24h | existing — `cha diagnose` + watcher resync |
| **E** | External webhook | Vault rotation, ArgoCD sync, cert-manager Issued | new `internal/server/webhook.go` — HTTP receiver |

A given probe can be wired to multiple classes; the trigger fan-in is
de-duplicated by the watcher's existing debounce window before the probe
fires.

## Milestones

Each milestone is independently shippable, gated by a Helm value, and adds
either a probe, a trigger class, or both. Milestones M1–M3 are pure A-class
extensions (single-file or single-package). M4–M5 add C and E classes. M6 is
the operator refactor that pays for itself once M1–M5 land.

---

### M1 — Ingress + DaemonSet + HPA + ArgoCD Application in `watchedGVRs`

**Trigger class:** A (resource-change)
**Effort:** ~1 day
**Files:** `internal/watcher/watcher.go` (one-line additions per GVR),
`internal/snapshot/snapshot.go` (matching `GVR*` constants + capture entries),
`charts/cluster-health-autopilot/values.yaml` (RBAC additions), test fixtures.

The watcher already auto-discovers Kong-routed services via the Endpoints
probe's `ingress_discovery.go`, but that discovery only re-runs on
`ResyncPeriod`. Adding `Ingress` to `watchedGVRs` means a new public hostname
is probed within the debounce window instead of within 24 h.

`DaemonSet` covers the `nvidia-device-plugin`, `node-exporter`, CSI plugins,
and CNI daemons — high-value resources that don't produce a Pod-level event
when *only their DaemonSet spec* changes (e.g., image bump, NodeSelector
edit).

`HorizontalPodAutoscaler` catches scale-up failures (HPA wants 10 replicas
but the scheduler can't place them — visible in HPA status `conditions`,
not in Pod events).

`ArgoCD Application` (CRD `argoproj.io/v1alpha1`) closes the loop: when a
sync goes `OutOfSync` or `Degraded`, CHA picks it up immediately rather than
waiting for the underlying pod symptom to surface.

**RBAC delta:** `get,list,watch` on `extensions/ingresses`,
`apps/daemonsets`, `autoscaling/horizontalpodautoscalers`,
`argoproj.io/applications`. Each gated by a Helm value (`watcher.gvrs.ingresses=true` etc.) so users can decline what their cluster doesn't run.

**Test:** Add a fixture in `internal/watcher/watcher_test.go` that emits a
fake `Update` event on each new GVR and asserts the watcher's `trigCh`
receives a signal within the debounce window.

---

### M2 — New probe: `probe.KongRoutes{}`

**Trigger class:** A (via M1's Ingress watch)
**Effort:** ~2 days
**Files:** `internal/probe/kong.go`, `internal/probe/kong_test.go`,
`catalog/catalog.go` (register), `internal/snapshot/*.go` (capture
`KongPlugin`, `KongConsumer` CRDs when present).

Probe steps:

1. List `networking.k8s.io/Ingress` objects (already in snapshot from M1).
2. For each Kong-managed Ingress (annotation `konghq.com/strip-path` or
   ingressClassName=kong), resolve its `backend.service.name` and assert the
   target Service has ≥1 ready endpoint.
3. Cross-check any referenced `KongPlugin` (`configuration.konghq.com/KongPlugin`)
   actually exists and has no `lastTransitionTime` failure marker.
4. Optional: if `--kong-admin-url` is set in config, query the Kong admin API
   for each route's upstream health (HEALTHY/DNS_ERROR/HEALTHCHECKS_OFF).

CRDs may be absent (cluster doesn't run Kong); probe must short-circuit
silently per the existing `Probe` contract.

**Tier:** OSS. Single file, no AI dependencies, mirrors `probe.Services`
style.

---

### M3 — New probe: `probe.GPUNodes{}` + new analyzer: `diagnose.LogPatternMatcher{}`

**Trigger class:** A
**Effort:** ~3 days for both
**Files:** `internal/probe/gpu_nodes.go`,
`internal/diagnose/log_pattern.go`, tests, catalog registration.

**`probe.GPUNodes`** asserts:

- Every node labeled `nvidia.com/gpu.present=true` has a Running
  `nvidia-device-plugin` Pod.
- Driver versions match across all GPU nodes (drift = WARNING).
- If a `dcgm-exporter` Service is reachable, recent ECC double-bit error
  count is 0; XID errors are absent.
- Per-GPU memory utilization < 95% at probe time (suggests an OOM-prone
  pod is overcommitting).

**`diagnose.LogPatternMatcher`** is a net-new analyzer *class*. Today's 7
analyzers operate on K8s status fields; this one tails recent logs for a
configured set of workloads and matches them against a regex catalog. Built-in
patterns include:

- `cumem_allocator.cpp:145` → vLLM FP8-KV + sleep-mode wake bug
  (caught the exact incident from the 2026-05-14 session)
- `Killed process .* total-vm` → OOMKill from kernel oom-killer
- `CUDA out of memory` → in-process GPU OOM (vLLM/diffusers)
- `i/o timeout` rate spike → kubelet→containerd tunnel breakage

Workloads to watch are config-driven: `cha.bionicaisolutions.com/watch-logs:
"vllm,latentsync,search-mcp"` annotation on the Deployment, or a Helm value
list. Log read uses the existing `corev1` API (`Pod.GetLogs`) with `tailLines:
500` budget. Patterns are exposed as a Go map so the paid catalog can extend.

**Tier:** OSS probe; OSS analyzer with the built-in pattern set. Paid
catalog may add ML-based anomaly detection alongside.

---

### M4 — Endpoints probe upgrade: optional L7 mode

**Trigger class:** A (existing)
**Effort:** ~1 day
**Files:** `internal/probe/endpoints.go`, `internal/probe/endpoints_test.go`.

The probe today validates TCP+TLS reachability of every auto-discovered
Ingress hostname. Add an optional L7 mode controlled by Ingress annotation:

```yaml
metadata:
  annotations:
    cha.bionicaisolutions.com/probe-l7-path: "/healthz"
    cha.bionicaisolutions.com/probe-l7-expect: '"status":"ok"'
    cha.bionicaisolutions.com/probe-l7-status: "200"
```

When set, the probe additionally issues a `GET <scheme>://<host><path>` with
the configured client cert / API key (read from `cha.bionicaisolutions.com/probe-l7-secretref`), asserts the status code matches, and that the body
contains the expected substring (or matches a regex if prefixed `regex:`).

Closes the "Kong returns 200 but body is wrong" class — the same class that
masked the search-mcp-server async-httpx bug for hours during the 2026-05-16
investigation (degraded but not zero-result).

**RBAC delta:** none (existing).

---

### M5 — Class C: PrometheusRule / Alertmanager consumer

**Trigger class:** C
**Effort:** ~3 days
**New package:** `internal/trigger/prom/`
**Files:** `internal/trigger/prom/client.go`, `prom_test.go`,
`internal/watcher/watcher.go` (additional `trigCh` source),
`charts/cluster-health-autopilot/values.yaml` (config block),
`internal/probe/registry.go` (new `Probe.Triggers() []TriggerRef` method
to express which alerts re-run which probes).

A new long-running goroutine polls Alertmanager's `/api/v2/alerts` (or
subscribes via Alertmanager's webhook if configured) and, for each firing
alert, looks up the registered `Probe.Triggers()` that match the alert's
`alertname` or labels, and pushes the matching probe IDs into the watcher's
existing debounce channel. The watcher's existing dedup + fingerprint logic
takes care of the rest.

Trigger declaration on a probe:

```go
func (p KongRoutes) Triggers() []probe.TriggerRef {
    return []probe.TriggerRef{
        {Source: "prometheus", AlertName: "KongUpstreamUnhealthy"},
        {Source: "prometheus", AlertName: "KongRouteHigh5xx"},
    }
}
```

This is the *highest-value* trigger class for slow-drift assets — disk fill,
cert expiry creep, error-budget burn, GPU ECC accumulation — that produce no
K8s event but do produce a metric.

**RBAC delta:** none for in-cluster Alertmanager; outbound HTTP only.

---

### M6 — Class E: External webhook receiver

**Trigger class:** E
**Effort:** ~3 days
**New package:** `internal/server/webhook/`
**Files:** `internal/server/webhook/handler.go`, tests,
`charts/cluster-health-autopilot/templates/webhook-service.yaml`,
`internal/server/webhook/auth.go` (HMAC-signed bodies; allowed senders
configured via Helm).

A new HTTP endpoint at `/webhook/<source>` with stable shape:

- `POST /webhook/vault` — body `{type: "secret_rotation", path, version}`
  → re-probe ESO consumers of that Vault path.
- `POST /webhook/argocd` — Argo's standard webhook payload → re-probe the
  touched Application's underlying resources.
- `POST /webhook/cert-manager` — `Issued` / `Renewed` event → re-probe
  TLS endpoints referencing the cert.
- `POST /webhook/cloudflare` — origin cert renewal → re-probe public
  endpoints.

Auth: each source has a registered HMAC secret in Vault, mounted via ESO. The
handler rejects unsigned or mis-signed requests with 401.

This unlocks the "rotation → probe within seconds" loop that the Vault-ESO
analyzers currently rely on a daily cron + ESO `refreshInterval` to detect.

**RBAC delta:** none. **Network delta:** a `Service` + `Ingress` for the
webhook endpoint (with its own `KongPlugin` for key-auth and rate limit, as
all other public mcp endpoints have).

---

### M7 — Operator-mode refactor (controller-runtime)

**Trigger class:** A + B (status-aware)
**Effort:** ~10 days, requires consensus on the design from
`project_cha_operator_plan` memory.
**Scope:** Replace the current "rerun everything on any debounced event"
model with a controller-runtime / kubebuilder reconcile loop. Each probe
gains a `Watches(...).Owns(...)` declaration and reconciles per-resource
instead of running the whole catalog.

Triggered now because once M1–M5 land, the watcher's full-pipeline-on-event
model becomes expensive: every Pod change re-runs all 6 probes + all 7+
analyzers, when in practice only the Pod-related probes need to. The
operator's reconciler-per-CR model fixes that.

Out of scope for this roadmap doc — the larger design lives in
`project_cha_operator_plan` memory and a future
`docs/design/2026-Qx-operator-refactor.md`.

---

## Sequencing rationale

```
M1 (4 GVRs to watcher) ─┬─→ M2 (Kong probe — needs Ingress in watcher)
                        │
                        └─→ M3 (GPU probe — needs DaemonSet in watcher)

M4 (L7 endpoints) ───── independent, ship anytime

M5 (Prometheus) ─────── needs the registry.TriggerRef extension; should be in
                        place before adding probes that don't surface in K8s
                        status (Velero backup recency, error-budget burn)

M6 (Webhooks) ───────── independent, depends on a Service+Ingress + Vault
                        for HMAC secrets — assumes the cluster has Kong
                        ingress (it does on the home cluster)

M7 (Operator) ───────── after M1–M5 are stable enough that the
                        rerun-on-everything model is visibly wasteful
```

Suggested order: **M1 → M4 → M2 → M3 → M5 → M6 → M7**, releasing v1.6, v1.7,
v1.8, v1.9, v1.10, v1.11, v2.0 respectively if we honor SemVer.

## Each milestone's "done" criteria

For each Mn:

1. ✅ Code merged behind a Helm value defaulting to `false` for the first
   release that ships it.
2. ✅ Unit tests added under the same `_test.go` convention as siblings,
   including a "feature flag off ⇒ behavior unchanged" test.
3. ✅ `docs/ADVERSARIAL_ANALYSIS.md` section added covering net-new failure
   modes (e.g., M6: replay attack on webhooks, accidental fan-out from
   malformed Argo payload, etc.).
4. ✅ `docs/FAILURE_MODES.md` updated with the new probe's expected
   "WARNING vs CRITICAL" thresholds.
5. ✅ One real-cluster soak: enable in the home cluster, watch for 7 days,
   confirm no Slack flood and no fixer misfires.
6. ✅ Helm value flipped to `true` *by default* in the next release after
   soak.

## OSS / paid tier split

| milestone | OSS | paid (CHA-com) |
|---|---|---|
| M1 (GVR additions) | full | — |
| M2 (Kong probe) | full | — |
| M3 (GPU probe + LogPatternMatcher) | full, with default pattern set | ML-based anomaly detection on the same log stream |
| M4 (L7 endpoints) | full | — |
| M5 (Prometheus trigger) | full, polling Alertmanager | PromQL-aware probe dispatcher (auto-derive triggers from existing PromQL) |
| M6 (Webhook receiver) | full | enterprise auth (mTLS + OIDC instead of HMAC) |
| M7 (Operator) | full | multi-cluster reconciler via ArgoCD ApplicationSet (existing CHA-com path) |

No paid-tier item *requires* a corresponding OSS item to be flagged off —
they all extend, never replace.

## Asset-class coverage after this roadmap

Once M1–M6 ship, the asset-class table from the Problem section flips
fully to ✓:

| asset class | post-roadmap |
|---|---|
| Vault / ESO secrets | ✓ A + D today; **+E webhook (M6)** for sub-second rotation latency |
| ImagePullSecrets | ✓ today; M5 adds threshold-based pull-failure rate |
| TLS certs | ✓ today; M5 adds days-to-expiry threshold; M6 cert-manager webhook |
| Ceph / PVCs | ✓ today; M5 OSD-down PromQL alert as fast-path |
| Nodes | ✓ today; **+B (M7)** for Ready-condition fast path |
| Postgres / CNPG | ✓ today |
| Kong routes / KongPlugin | **+M2** |
| Deployments / Pods / Jobs | ✓ + **M3 log patterns** add depth |
| GPU health | **+M3** |
| DNS / CoreDNS | partial — `probe.Services{}` already targets `kube-dns`; M5 adds DNS-failure PromQL |
| Workload log signals | **+M3 LogPatternMatcher** |
| HPA / autoscaling | **+M1** GVR watch; M3 covers via HPA event matcher |
| ArgoCD Applications | **+M1** GVR watch |
| Velero backup recency | **+M5** (no K8s event; PromQL on `velero_backup_last_successful_timestamp`) |
| ResourceQuota | **+M1** (already a watch GVR group); needs probe in M5-ish |
| Vault server | **+M5** (Vault exports `vault_core_unsealed` etc.) and **+M6** Vault webhook |
| Endpoints L7 | **+M4** |
| mcp.baisoln.com endpoints | ✓ today (TCP+TLS); **+M4** for HTTP-200 + body match |

## Open questions

1. **Probe-to-trigger declaration form** (M5) — Method on `Probe` interface
   vs sidecar config file vs annotation on a `TriggerSpec` CR? Method is
   simplest but couples triggers to compile-time; CR would let operators
   add triggers to existing probes without rebuilding.
2. **Log access surface** (M3 LogPatternMatcher) — `Pod.GetLogs` over the
   apiserver is fine for low cardinality but doesn't scale to a cluster
   of 500 pods. Should the analyzer accept a Loki/Promtail URL as an
   alternate source?
3. **Webhook auth model** (M6) — HMAC (simple, stateless) vs OIDC (proper
   identity, requires a token service). HMAC for OSS, OIDC for paid is the
   tentative split.
4. **Operator-mode migration path** (M7) — Do we keep the `cha watch
   --live` command as a "lite" mode for users who don't want the
   controller-runtime dependency tree, or fully replace? Tentative: keep
   `cha watch --live` as the SHIPPING OSS path for v2.x; the operator is
   the paid CHA-com runtime.

## Risk

- **Watcher CPU under high-churn clusters** (M1): adding 4 GVRs to the
  full-pipeline-rerun watcher could cause noticeable CPU on clusters where
  Ingresses change frequently (devcontainer workflow, GitOps pushes). M7
  mitigates; until then, the debounce window should be tuned conservatively.
- **Log read cost** (M3): tailing 500 lines per watched pod every cycle
  adds non-trivial API server load. Mitigation: log analyzer fires only on
  Pod restart count delta, not every cycle.
- **Alertmanager polling cost** (M5): `/api/v2/alerts` is cheap, but if CHA
  is deployed in a cluster with 10k+ active alerts, payload size matters.
  Mitigation: poll with a label selector (`cha.bionicaisolutions.com/probe-trigger=true`)
  so only opt-in alerts return.
- **Webhook replay / DoS** (M6): mitigated by HMAC + nonce window + a
  per-source rate limiter in the same handler.

## Reference

Investigation history that produced this roadmap:

- 2026-05-16 production session: discovered that the secret-handling story
  is mature (vault → ESO → analyzers + fixers) but no first-class probe for
  Kong, GPU, DNS, ArgoCD, HPA, Velero, Vault-server, or workload log
  patterns. See [[search-mcp-degradation]] memory for the search-stack
  investigation that hit the missing Ingress-watch trigger; the missing
  `cumem_allocator.cpp:145` log analyzer that would have caught the
  2026-05-14 vLLM FP8-KV bug automatically; and the `ai-latentsync`
  cotenant-OOM that needed a GPU-memory probe to catch proactively.
- Current watcher state inspected at `internal/watcher/watcher.go` line 92
  (14 GVRs); current catalog at `catalog/catalog.go` (6 probes, 7 analyzers,
  5 fixers).

## Out of scope (later if needed)

- Cross-cluster trigger correlation (a single probe firing across N clusters
  via the ApplicationSet path) — captured in the operator-plan memory.
- Active synthetic-traffic probes (e.g., synthetic load against
  mcp.baisoln.com to detect cold-start regressions). Possible v2.x.
- Self-probing — CHA reporting on its own pod health to a peer cluster.
  Possible v2.x once multi-cluster is in.
