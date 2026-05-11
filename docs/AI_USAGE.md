# Does Cluster Health Autopilot use AI?

**No, not in the hot path.** This is a deliberate design choice and a positioning statement, not an oversight. Holds for every release shipped so far, up to and including the current v0.9.5.

## What's deterministic (everything that ships)

| Component | Mechanism | Source |
|---|---|---|
| Probes (Ceph, Nodes, Postgres, PVCs, Services, Endpoints) | Read CRD `.status` fields, count Ready conditions, compute simple ratios; HTTP(S) GET against canonical hostnames | [`internal/probe/`](../internal/probe/) |
| 8 analyzers (`SecretKeyMissing`, `FailingExternalSecrets`, `ProactiveSecretKeyCheck`, `UnprovisionedSecret`, `VaultPathMissing`, `CertExpiry`, `ImagePullAuth`, `IngressCoverage`) | Regex on kubelet event messages, owner-chain walks, ESO target matching, direct Vault key-name lookups, cert-manager status reads, ingress-vs-endpoint set diff | [`internal/diagnose/`](../internal/diagnose/) |
| 4 fixers (`StaleErrorPods`, `StuckJobsWithBadSecretRef`, `StuckRSPods`, `StuckCertificateRequests`) | Pattern-match conditions → call API verbs (`pods/delete`, `jobs/delete`, `deployments/patch`, `certificaterequests/delete`, `orders/delete`) | [`internal/fix/`](../internal/fix/) |
| Watcher event loop | Kubernetes watch on resource set, 10s debounce, run full probe+analyzer stack, fingerprint dedup against DriftReport seen-map | [`internal/watcher/`](../internal/watcher/) |
| Three-channel Slack routing | Subject-prefix dispatch: post-fix CHA-acted set → `#ceph-alerts`; unfixed set → `#ceph-critical`; daily digest → `#healthinfo` | [`internal/report/routing.go`](../internal/report/routing.go) |
| Alertmanager hub integration | POST `/api/v2/alerts` every cycle with label scheme `alertname=cha_issue`/`cha_fixer_acted`; AM handles dedup/silencing/fan-out | [`internal/report/alertmanager.go`](../internal/report/alertmanager.go) |
| Daily digest | Read DriftReport CR history, classify new (firstObserved < 24h) vs persistent vs auto-fixed, format Slack attachment | [`internal/report/daily.go`](../internal/report/daily.go) |

Same input → same output, every time, **auditable from source**. The fix catalog IS the Go source — readable in an afternoon.

## Why no AI in the hot path

1. **Auditability.** "An LLM decided to delete this pod" is not a defensible position at 3 AM. Every fixer's trigger condition and remediation is a few lines of Go that an SRE can read and reason about before opting in.
2. **Whitelist-only safety.** The product's central claim is the catalog of *known-safe* corrections. Letting an LLM propose a remediation breaks that contract — the action surface becomes unbounded and the safety story collapses.
3. **No SaaS dependency.** The README leads with "no telemetry, no SaaS dependency by default." An OpenAI/Anthropic API call on every cluster event would invalidate that promise — and add per-call cost, latency, and a privacy footprint customers don't want.
4. **Privacy.** Customer cluster state (pod names, namespaces, image tags, error messages) is sensitive. Sending it to a third-party LLM would require enterprise-grade DPAs and is a non-starter for many target customers (regulated industries, sovereign-cloud deployments, security-conscious infra teams).

## Where AI could enter later (not today)

The roadmap to v1.0 does not put AI on the critical path. There are three places it could plausibly land in `v2+`, **none of which are in the hot path of the in-cluster install**:

1. **Verified Signature Library curation (offline, asynchronous).** As the OSS catalog grows from customer contributions, an LLM-assisted review pipeline could cluster similar incident reports and propose new fixer signatures. The *signatures themselves* still ship as deterministic Go. The LLM is in the lab, not the customer cluster.
2. **Diagnostic narration (optional, opt-in).** Today's diagnostics are precise structured strings. Some operators may want a narrative ("3 of your 47 services are wedged because Vault path X has no `stripe_api_key` property; here's the kubectl runbook"). That could be a separate `cha narrate --diagnostics-json …` subcommand that hits an LLM on demand and is **never** in the cron path. Customer chooses whether to enable it; the data sent to the LLM is the structured diagnostic output, not raw cluster state.
3. **Triage prioritization in the Fleet Console (the SaaS).** When you're managing 50 clusters with 200 active diagnostics, "which 5 should I fix today" is a ranking problem where light-touch ML helps. That's a **post-fundraise** product and a separate code path from the OSS engine.

## Strategic position

Competitors that lead with "AI-powered remediation" are often masking a thin wrapper of `kubectl exec` + GPT-4 + prayer. Cluster Health Autopilot's pitch is the opposite: **the deterministic engine is the differentiator**. AI is a future feature that customers can opt into, not a load-bearing component of correctness.

When AI eventually does enter the product, it will be:
- **Off by default** in any path that touches cluster state
- **Outside the cron loop** that runs on every customer cluster
- **Auditable** — the LLM-assisted output is a recommendation that goes through the same deterministic safety gates as a hand-written signature
- **Privacy-respecting** — customer cluster state never leaves the cluster without explicit opt-in
