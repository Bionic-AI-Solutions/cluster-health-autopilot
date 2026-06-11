# Phase 2d — RAG-Driven Cluster Knowledge Layer

> **STATUS: 🚧 PARTIAL — RAG substrate + READ/outcome layer SHIPPED; full learned-knowledge replay is the open moat.**
> _(P4.1 honest-header pass, 2026-06-11)_
>
> Shipped substrate (OSS + CHA-com): the `pkg/rag` store with typed entry kinds including `KindWorkload` and `KindFindingOutcome` (`pkg/rag/types.go`); the workload feeder upserting per-workload entries each cycle (`internal/feeder` → promoted to `pkg/feeder`, PR #152/#158); CHA-com **Phase 2.A — RAG memory READ + revert detection + per-cycle observability (v1.15.0)** per the consolidated roadmap; and the Approve/Deny/Silence outcome signal captured as `finding_outcome` entries. v1.10.5 (PR #132) removed the hardcoded cluster-specific defaults this doc motivates.
>
> **DELTA — still open:** the doc's core promise — **learned per-cluster knowledge replayed each cycle into the probe/analyzer pipeline and revised continuously from SRE-action signals** (the "moat") — is only partially realized. The store + outcome capture exist; full closed-loop replay (outcomes re-grounding future proposals) is the genuinely-open strategic work tracked under `cha-ai-remediation-direction` / Phase P6 and the Q3 forward plan. Treat the prose below as the target architecture, not as fully-delivered behavior.
>
> Body below is the original design, preserved for context.

---

**Status:** Design draft
**Tier:** Paid (CHA-com / AI tier, gated on `spec.ai.enabled`)
**Author:** opened 2026-06-01
**Companions:** [2026-05-investigator-agent.md](2026-05-investigator-agent.md), [`cha-operator-phase-1c-design`](../../) memory, `cha-ai-remediation-direction` memory

## TL;DR

CHA OSS today auto-discovers Ingress hosts and per-Deployment health, but
cluster-specific knowledge ("which apex domains matter," "which workloads are
critical," "what's normal restart churn here") is either **hardcoded** (v1.10.5
removed all hardcoded defaults from OSS code) or **hand-curated** in CR spec.

This document proposes a third source, available only on the paid AI tier:
**learned, per-cluster knowledge persisted in Qdrant, replayed each cycle into
the probe/analyzer pipeline, and revised continuously from SRE-action signals
(Approve / Deny / Silence outcomes).**

This layer is the moat — it can't be reproduced by reading the OSS code, only
by running CHA in your cluster long enough to learn.

## Why we need it

Three concrete problems v1.10.5 made visible:

1. **Apex domains and Cloudflare-only hosts have no Ingress** — auto-discovery
   misses them. Today the operator must add them to `spec.probe.endpointTargets`
   by hand. Cluster-A's `apex.acme.com` is irrelevant to cluster-B's `corp.io`.
   These should be **learned** by walking the cluster's Cloudflare zones once
   (already done by DNSChainDrift) and persisting the result.

2. **"Critical" vs "noise" is per-cluster** — `Pod restart count > 50` is a
   problem in a stateless web tier but normal in cnpg-system. Static
   thresholds in OSS code can't model this. SREs already correct CHA every
   day by clicking Silence on noisy classes (HPA-ScalingDisabled on KEDA
   scale-to-zero, ExternalName services in oauth2-proxy patterns). Those
   clicks are signal — we throw them away.

3. **Workload importance signals exist but aren't read** — labels like
   `tier: critical`, sustained RPS, `priorityClassName=system-cluster-critical`,
   `prometheus.io/scrape: true`, ownership annotations. The OSS analyzer sees
   them per-cycle and forgets. A learning layer can keep them.

## The "cluster RAG" data model

One Qdrant collection per cluster, sharded by `kind`:

```
cluster_id: "bionic-cluster"     # from CR spec.alerting.alertmanager.clusterName
kind: "apex_domain"              # or workload | finding_outcome | baseline | silenced_class | …
key:  "apex.example.com"         # the natural identifier within kind
value: {
  first_seen: 2026-06-01T00:00:00Z,
  last_seen: 2026-06-01T18:00:00Z,
  observations: 144,             # how many cycles we've seen this
  importance: 0.87,              # learned weight, range [0,1]
  sources: ["cloudflare_zone", "ingress_host", "sre_pin"],
  features: { ... },             # kind-specific
  signal_history: [
    {ts: ..., action: "approved", finding_class: "cert-expiry", actor: "sre1"},
    {ts: ..., action: "silenced", finding_class: "duplicate-host", actor: "sre2"},
    ...
  ]
}
embedding: <vector>              # over (subject + sample messages + features) for RAG retrieval
```

`kind=apex_domain` (the leak v1.10.5 surfaced): populated from a Cloudflare
zone walk plus Ingress host enumeration; replayed as `EndpointTarget`s.

`kind=workload`: per Deployment / StatefulSet / DaemonSet / Job, importance
learned from label hints, traffic, owner-references, and finding history.
Replayed as `ServiceTarget`s — with `criticality` driving severity.

`kind=finding_outcome`: every Approve / Deny / Silence click recorded keyed on
`(cluster_id, finding_class, subject_pattern)`. The next time CHA's analyzer
emits a matching finding it bypasses the noise → SRE-rejected classes
auto-suppress; SRE-approved fix templates rank higher.

`kind=baseline`: per-`(workload, metric)` rolling statistics — restart count
distribution, RPS p50/p95, container OOM frequency. Drives "is this churn
abnormal **for this cluster**" instead of universal thresholds.

`kind=silenced_class`: explicit Silence CRs the SRE created are mirrored
here so the AI tier can propose `kind=silenced_class` extensions on its own
("you silenced HPA-ScalingDisabled on KEDA hosts 3 times; want me to
auto-silence the whole class for KEDA-managed HPAs?").

## Read path — how the analyzer consumes RAG

Today the analyzer:

```
DiscoverIngressTargets() ⨁ DefaultEndpointTargets() → probe → emit findings
```

Tomorrow (`spec.ai.enabled=true` clusters):

```
DiscoverIngressTargets()
  ⨁ LearntEndpointTargets(RAG, kind=apex_domain, importance ≥ τ)
  ⨁ spec.probe.endpointTargets (CR override, always wins)
  → probe → run analyzers → emit findings
  → for each finding: query RAG kind=finding_outcome for similar
  → if SRE has Silenced this class 3+ times: drop severity to info OR auto-silence
  → otherwise: severity from baseline (kind=baseline) + class default
```

OSS behaviour is **literally unchanged** because OSS doesn't talk to Qdrant.
The AI tier adds a `LearntEndpointTargets`/`LearntServiceTargets` source
behind a `pkg/rag.Reader` interface implemented only by `bionic-aiwatch` in
CHA-com. OSS gets a no-op reader.

## Write path — how RAG gets populated

Three feeders, each contained to its own goroutine in `bionic-aiwatch`:

1. **Discovery feeder** (runs at startup + every 24h):
   - Lists Cloudflare zones (already permitted; the credential exists)
   - For each A/AAAA/CNAME, records `kind=apex_domain` with `first_seen`
     and `sources=[cloudflare_zone]`. Merges with Ingress hosts already
     discovered.
   - Walks every Deployment/StatefulSet/DaemonSet — records `kind=workload`
     with the labels and ownerRefs as `features`.

2. **Observation feeder** (per analyzer cycle):
   - For each finding emitted this cycle, records a `kind=finding_outcome`
     stub with `action=open`. Resolution events (delta → `resolved`) update
     the stub.
   - For each workload observed in `kind=workload`, updates rolling stats
     into `kind=baseline`.

3. **Outcome feeder** (real-time, from the approval-server):
   - Every Approve / Deny click on a cha-com #17 signed URL writes a
     `kind=finding_outcome` row with `action ∈ {approved, denied}` and the
     JWT subject claim as the actor. Silence-CR creation events from the
     watch loop write `action=silenced`.

## Operator surfaces

CR spec additions (paid tier only — OSS schema rejects with
`status.conditions[type=AIFieldsInvalid]`):

```yaml
spec:
  ai:
    enabled: true                # gates the entire RAG layer
    rag:
      collection: cluster-rag    # Qdrant collection name (default)
      importance:
        endpointMin: 0.3         # τ for kind=apex_domain replay
        workloadMin: 0.5         # τ for kind=workload → ServiceTarget
      decay:
        halflife: 30d            # how fast "importance" decays without re-observation
      pinned:                    # operator-supplied hard overrides; bypass importance
        endpoints:
          - apex.example.com
        workloads:
          - "core/checkout-api"
```

Status conditions (new):

- `type=RAGReady` — Qdrant collection reachable + readable + has ≥1 row
- `type=RAGStale` — last observation feed ≥ 24h ago (writes blocked)
- `type=LearntTargetsReplaying` — replay loop is feeding probes from RAG

## Failure modes & tests

The proposed gates before this is callable production:

1. **OSS install hits no Qdrant call** — verified by `pkg/rag.NoopReader` being
   the wired-in default; AI-tier wiring activates only when CSV/Helm sets
   `ai.rag.enabled=true`.
2. **Qdrant outage doesn't break probes** — `LearntEndpointTargets` returns
   `nil` on read error; the discovery + CR-override paths still work.
3. **Importance drift can't auto-promote dangerous classes** —
   `finding_outcome` of `action=approved` only RAISES the importance of
   subjects that ALREADY have a CR-pinned class; arbitrary new probe targets
   never originate from approval clicks.
4. **PII / scope** — Qdrant collection is per-cluster and never leaves it.
   Field-level encryption for `signal_history.actor`. No SaaS dependency.

## Phasing

| Phase | Scope | Gate |
|---|---|---|
| 2d-α | `kind=apex_domain` + discovery feeder only. Replay into `LearntEndpointTargets`. Operator override + decay. | OLM-on-kind smoke + bundle parity on the new CR fields. Live cluster: backfill from CF zones + verify learned hostnames appear as probe targets. |
| 2d-β | `kind=workload` + `kind=baseline`. Replay into `LearntServiceTargets` with `criticality` driving severity. | Per-namespace baseline data >= 7 days before any severity remapping takes effect. |
| 2d-γ | `kind=finding_outcome` + outcome feeder + the auto-silence proposer. Wired into `renderAIBlocks` Approve/Deny pair. | Adversarial review for false-positive auto-silence; AI-proposer dry-run on a corpus of past findings before going live. |
| 2d-δ | UI surface (CHA-com web) showing the learned model per cluster, with operator controls to demote / unlearn rows. | Auth wired through Keycloak realm `bionic`; per-cluster RBAC. |

Each phase is its own PR and chart bump. 2d-α can ship as v1.11.0.

## Open questions

1. Where does Qdrant live for OSS users who turn on AI later? Today
   `bionic-rag` is a StatefulSet the operator provisions. Adopting a managed
   Qdrant Cloud is on the table for cha-com.
2. How do we backfill a freshly-installed paid cluster? Discovery + 7-day
   observation gate. UI shows "still learning" with an explicit days-remaining
   indicator.
3. Multi-cluster (CHA-com cohort installs): do per-cluster RAGs ever share?
   No, by design — sharing makes one customer's noise classes affect another's.
   Federation could be a Phase 2e topic.
