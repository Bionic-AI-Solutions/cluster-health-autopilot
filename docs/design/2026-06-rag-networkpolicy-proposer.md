# Phase 2d-β — RAG-Driven NetworkPolicy Proposer

> **STATUS: ✅ SHIPPED — OSS proposer wire-up + CNI gating landed.**
> _(P4.1 honest-header pass, 2026-06-11)_
>
> Shipped: the Phase 2d-β OSS proposer wire-up, CNI-gated + k3s-safe (PR #137); CNI detection hardened to recognize kube-router as a NetPol enforcer (v1.12.3, PR #142, after the 2026-06-01 dev-cluster outage); LoadBalancer-aware + kube-system-aware hardening (v1.13.0, PR #143). The OSS watcher mints Approve/Deny URLs so `ProposedPolicyYAML`-bearing diagnostics reach Slack/Alertmanager/OpenProject (Path B, PR #151 — closes the gap where NetworkPolicyProposer URLs were minted in the srenix-enterprise aiwatch but never reached user-facing sinks). The approval-server NetworkPolicy itself was also tightened to restrict ingress to the gateway ns (P2.6b-OSS).
>
> No material design-vs-shipped scope-shrink. Body below is the original design, preserved for context.

---

**Status:** Design draft
**Tier:** Paid (Srenix Enterprise / AI tier, gated on `spec.ai.enabled`)
**Author:** opened 2026-06-01
**Parent:** [2026-06-rag-cluster-knowledge.md](2026-06-rag-cluster-knowledge.md)

## Why

The Srenix `RBACDrift`/`Namespace` analyzers flag namespaces without NetworkPolicies as a zero-trust gap — and they're right. But **fixing it manually doesn't scale**: each namespace needs its own analysis of what traffic to allow.

Empirical hit on the dev cluster (2026-06-01):

  - 41 namespaces flagged
  - Spans single-pod scratch ns through 22-pod `mcp` ns with 19 Ingresses
  - No single "default-deny + standard allows" template fits all of them — `mcp` needs allow rules for 19 distinct backend services, `default` needs almost nothing, `agentic-sre` needs `auth-proxy` egress, etc.

Doing this by hand means either (a) breaking traffic in production, or (b) spending hours per namespace. **This is exactly the work the AI tier should absorb.**

## What "proposer" means here

The same Approve/Deny pattern v1.10.4 wired for AI-proposed fixes:

  1. Observe each namespace's actual pod-to-pod traffic over N days (via `kind=baseline`)
  2. Generate a `NetworkPolicy` (or set of) per namespace from observation
  3. Render in Slack with the `✅ Approve · ❌ Deny · 📄 Details` triplet — the policy YAML is in the Details link
  4. On Approve: apply the policy. On Deny: record `kind=finding_outcome` so the proposer learns not to re-propose this shape
  5. On Apply: monitor for 24h. If new findings spike (network errors in event streams, pod connection failures), auto-revert and record `verified=false` outcome

This makes "41 namespaces" into "41 click-throughs the SRE reviews over coffee" — each grounded in evidence the SRE can audit.

## Data model additions

Extends [`pkg/rag`](../../pkg/rag) Entry shape, no new `EntryKind` values:

### `kind=baseline` features for network observation

```go
{
  ClusterID: "bionic-cluster",
  Kind: "baseline",
  Key: "network/<namespace>/<workload>",     // e.g. "network/mcp/mcp-api-server"
  Observations: 8640,                         // 30 days × 24h × 12 (5-min sample period)
  Features: {
    "ingress_from": [
      {"namespace": "kong", "pod_label": "app.kubernetes.io/name=kong", "ports": [8080]},
      {"namespace": "vault", "pod_label": "app=vault-agent", "ports": [8200]},
    ],
    "egress_to": [
      {"namespace": "redis", "pod_label": "app=redis-cluster-ceph", "ports": [6379]},
      {"namespace": "pg", "pod_label": "cnpg.io/cluster=pg-ceph", "ports": [5432]},
      {"external_cidr": "0.0.0.0/0", "ports": [443], "reason": "external_api"},
    ],
    "dns_egress": true,
    "first_observed": "2026-05-01T...",
    "last_observed":  "2026-06-01T...",
  }
}
```

### `kind=finding_outcome` for network policy decisions

```go
{
  Kind: "finding_outcome",
  Key: "netpol-proposal:<namespace>",
  SignalHistory: [
    {ts: ..., action: "proposed",  finding_class: "missing-network-policy", actor: "ai-proposer"},
    {ts: ..., action: "approved",  finding_class: "missing-network-policy", actor: "sre1"},
    {ts: ..., action: "verified",  finding_class: "missing-network-policy", actor: "system"},  // 24h post-apply, no spike
  ]
}
```

## Where the network-observation data comes from

Three signal sources, layered:

1. **Cilium / Calico flow logs** (when present): direct pod-to-pod flow data with namespace + label resolution. The gold standard; produces the highest-fidelity baseline.
2. **Kong access logs** (always present in this cluster): every Ingress-mediated request carries `X-Forwarded-For` + upstream service. Use to populate `ingress_from` for Kong-fronted services.
3. **Pod event stream / DNS audit logs** (fallback): if neither of the above is wired, infer egress from DNS lookups + container env vars naming downstream services (e.g. `REDIS_HOST`, `POSTGRES_URL`). Low fidelity; only enough to propose loose policies.

### Operator surface

```yaml
spec:
  ai:
    netpolProposer:
      enabled: true
      observationDays: 30           # minimum baseline before proposing
      sources:
        cilium: { enabled: false }  # set true if Cilium Hubble is running
        kong:   { enabled: true,  logSecretRef: "kong-access-logs" }
        events: { enabled: true }   # always available; fallback
      autoApply:
        namespaces: []              # default: never auto-apply, always require human
      revertOn:
        spikeMultiplier: 3.0        # net-error rate > 3× baseline → auto-revert
        spikeWindow: 24h
```

## Phasing

| Phase | Scope | Gate |
|---|---|---|
| 2d-β-1 | Observation feeders only (Kong logs + DNS events). Populate `kind=baseline` with `network/*` keys for every namespace. No proposals yet. | After 30 days of observation, the cluster has a baseline. Verify on the dev cluster's existing 41 namespaces. |
| 2d-β-2 | Proposer generates draft `NetworkPolicy` per namespace and renders in Slack with Approve/Deny. On approve: apply. On deny: record. No auto-apply. | Proposed policies match what an experienced SRE would write for 80%+ of the 41 namespaces in evaluation. |
| 2d-β-3 | Post-apply verification loop: monitor for 24h, auto-revert on connection-error spike. | False-positive revert rate <1% over 100 applied policies. |
| 2d-β-4 | (Optional, opt-in) Auto-apply mode for explicitly listed namespaces. | Approved by user via `spec.ai.netpolProposer.autoApply.namespaces`. |

## Out-of-scope for now

- Inter-cluster NetworkPolicy federation
- Egress policies for namespaces whose pods make calls to many external SaaS endpoints (high cardinality)
- L7 policies (Cilium-only); 2d-β ships L4 only

## Adjacent decisions captured

- `default-deny` is **never** applied without a corresponding `allow` set. Proposer always pairs them in a single Kustomize.
- Empty namespaces (0 pods) get `default-deny` only — when workloads land, the proposer re-runs and adds allows from observation.
- The Slack/AM render reuses the v1.10.4 Approve/Deny path; the JTI is reused from srenix-enterprise #17.

## Open questions

1. **Privacy of flow data**: pod-to-pod flow logs may contain user-data IPs. Strip / anonymize before persisting to Qdrant? Likely yes for `external_cidr` fields.
2. **Re-proposing after a deny**: if SRE denied a proposed policy, when can the proposer try again? Probably "only when observation diverges significantly" — needs a definition of "significantly".
3. **Cross-cluster sharing of policy shapes**: explicit no (per parent doc Phase 2d-α conclusion). Each cluster's policies are private to its RAG.
