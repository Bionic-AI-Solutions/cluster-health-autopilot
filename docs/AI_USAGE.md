# AI in Agentic SRE

The OSS `srenix` binary keeps **LLMs off the hot path**. This is a positioning
commitment, not a transient state. LLM-driven enrichment, proposal, and
runbook generation ship in the **commercial Srenix Enterprise binary** as an opt-in
layered tier system.

As of v1.5, the OSS engine ships one *deterministic* read-only agent —
the rule-based **Layer-2 Investigator**. It uses no LLM; it pattern-
matches critical findings and runs a fixed set of read-only follow-up
probes (DNS / HTTP / TLS / describe / events). It is auditable line by
line, registered through the same `pkg/ai.Investigator` interface the
paid LLM-backed implementation uses, and replaceable by the paid binary
when Srenix Enterprise is installed. The OSS and paid implementations share the
same closed `Environment` action surface, so the read-only safety
property holds either way.

Both binaries share the same OSS engine; the LLM layer is purely additive
on top of it.

This document explains:
1. What stays LLM-free in OSS, forever (including the rule-based investigator)
2. What LLMs add on top in Srenix Enterprise, and how it's gated
3. Why we built it this way

---

## 1. The OSS engine is and stays LLM-free

| Component | Mechanism | Source |
|---|---|---|
| 6 probes (Ceph, Nodes, Postgres, PVCs, Services, Endpoints) | CRD `.status` field reads, HTTP(S) GET against canonical hostnames. The `Endpoints` probe auto-discovers Ingress hosts (v1.2+) and applies retry / N-of-M streak suppression (v1.4+). | [`internal/probe/`](../internal/probe/) |
| 7 analyzers (`SecretKeyMissing`, `FailingExternalSecrets`, `ProactiveSecretKeyCheck`, `UnprovisionedSecret`, `VaultPathMissing`, `CertExpiry`, `ImagePullAuth`, `TLSSecretMismatch`) | Regex on kubelet events, owner-chain walks, ESO target matching, direct Vault key-name lookups, cert-manager status reads, Ingress↔Certificate secret-name cross-check | [`internal/diagnose/`](../internal/diagnose/) |
| 4 default + 1 opt-in fixer (`StaleErrorPods`, `StuckJobsWithBadSecretRef`, `StuckRSPods`, `StuckCertificateRequests`; opt-in: `TLSSecretMismatch`) | Pattern-match conditions → call API verbs from a closed whitelist. The opt-in TLS fixer JSON-patches Ingress `spec.tls[].secretName`; skips GitOps-managed Ingresses (ArgoCD / Flux / Helm) and protected namespaces. | [`internal/fix/`](../internal/fix/) |
| Layer-2 Investigator (rule-based, v1.5+) | Pattern-match critical Findings / Diagnostics → run a closed set of read-only tools (DNS lookup, HTTP probe, TLS inspect, describe, events) → attach a `🔬` summary and structured observations to the alert and DriftReport CR. No LLM, no new RBAC. | [`internal/investigator/rules.go`](../internal/investigator/rules.go) |
| Watcher event loop | Kubernetes watch + 10s debounce + full probe/analyze cycle + fingerprint dedup. Investigation runs **after** post-fix re-diagnose so fixers can resolve issues before they are investigated. | [`internal/watcher/`](../internal/watcher/) |
| Three-channel Slack routing | Subject-prefix dispatch — post-fix Srenix-acted set → `#ceph-alerts`, unfixed → `#ceph-critical`, daily digest → `#healthinfo` | [`internal/report/routing.go`](../internal/report/routing.go) |
| Alertmanager hub integration | POST `/api/v2/alerts` every cycle; AM handles dedup, silencing, fan-out | [`internal/report/alertmanager.go`](../internal/report/alertmanager.go) |
| Daily digest | Reads DriftReport CR history; classifies new/persistent/auto-fixed | [`internal/report/daily.go`](../internal/report/daily.go) |
| JWT signing primitives | Ed25519 (EdDSA) via crypto/ed25519; minimal deps | [`pkg/ai/jwt.go`](../pkg/ai/jwt.go) |

Same input → same diagnosis, every time, **auditable from source**. An
SRE can read the entire fix list and the investigator rules in an
afternoon.

**Even when Srenix Enterprise is installed**, the OSS engine code paths are
unchanged. If `ai.enabled=false` (the default), Srenix Enterprise behaves
identically to the OSS `srenix` binary at the LLM boundary: no LLM calls,
no `🤖` enrichment, no Apply Fix buttons, no `ai.*` audit events. The
rule-based investigator still runs (it's part of OSS), and Srenix Enterprise can
override it with an LLM-backed implementation via the same
`Registry.RegisterInvestigator` hook — but the OSS path is the default
and the override is opt-in.

---

## 2. Srenix Enterprise adds LLM tiers on top of the OSS engine

Each tier is documented in [AI_TIERS.md](AI_TIERS.md). Quick overview:

| Tier | What it adds | What it does **NOT** add |
|---|---|---|
| **T0 Narration** | LLM-generated 2–4 sentence root-cause narrative under each diagnostic | Any mutation capability. Tier 0 is read-only enrichment. |
| **L2 Investigator (LLM-backed deep-RCA, v0.2.0-alpha.1)** | Replaces the OSS rule-based investigator with an LLM that picks read-only cluster tools dynamically from the same closed `Environment` enum, synthesizes a client-redacted Firecrawl web query for external documentation search, persists the root-cause analysis to the `srenix_investigations` Qdrant collection, and forwards the RCA to every AI tier (T0–T3) via a `<root_cause>` prompt block. | Any new cluster tool, any mutation capability, any new RBAC. Firecrawl is opt-in (key env must be set); the cluster tool surface is unchanged. |
| **T1 Single fix** | One-click signed URLs that apply an existing OSS fixer to a specific target | New verbs, new RBAC, autonomous execution |
| **T2 Multi-step plan** | Sequential multi-action plans with per-step approval | Bypass of step-by-step approval; plans cannot self-modify |
| **T3 Vault runbook** | Generated `vault kv patch` runbook + dual-approval recording | Srenix Enterprise NEVER writes to Vault. Runbook is human-run. |

**Core safety invariant**: AI **proposes**, humans **approve**,
deterministic Go code **executes**. The OSS engine's Mutator interface
is never called from an LLM response. Every mutation passes through:

```
LLM proposal → structured-output validator (closed enum)
            → JWT signer (in-cluster Ed25519 key, watcher SA can't read it)
            → human click via signed expiring URL
            → approval-server verify (signature + expiry + one-time-use + OIDC identity)
            → admission policy re-check (protected NS)
            → OPA/Gatekeeper independently re-validates (defense in depth)
            → snapshot.Mutator call (same code path as `srenix remediate --live`)
            → 60s post-apply verification
            → full audit-trail event chain
```

Higher tiers grow *what AI can analyze and propose*, not *whether a
human is in the loop*.

---

## 3. Why this shape

### Mapping to established AI safety frameworks

| Concern | OWASP LLM Top 10 | Our control |
|---|---|---|
| LLM with autonomous mutation | LLM08 — Excessive Agency | Recommendation-only at every tier; human one-click approval; admission re-check; OPA/Gatekeeper third gate |
| Crafted prompts in event messages | LLM01 — Prompt Injection | All untrusted text wrapped in `<observed_data>` blocks; ScrubInjection() strips known patterns before LLM input; structured-output schema rejects free-form action requests |
| Free-form output executed as commands | LLM02 — Insecure Output Handling | JSON schema validator; closed-enum `action_kind`; protected-namespace re-check before mutation |
| LLM sees raw cluster state | LLM06 — Sensitive Information Disclosure | `RedactDiagnostic()` SHA-256-hashes namespace/name; redacts IPs/UIDs/hostnames; Secret bytes never sent (Srenix never reads them anyway) |
| SaaS LLM as default | OSS positioning + GDPR | Default `--ai-endpoint` is operator-supplied (BYOM); SaaS (OpenAI/Anthropic) requires explicit `--ai-allow-saas` opt-in with audit-logged acknowledgment |

Full mapping in [THREAT_MODEL_AI.md](THREAT_MODEL_AI.md).

### What competitors get wrong

Many "AI for SRE" products lead with autonomous remediation: the LLM
decides what to do, executes it, and reports after the fact. This
violates LLM08 and OWASP's repeated guidance to gate any agency
behind explicit human authorization.

Srenix Enterprise's tiers explicitly refuse this shape. Even T3 (the most
powerful tier) is recommendation-only: it generates a runbook that
two distinct humans approve and then run themselves with values
neither Srenix Enterprise nor the LLM ever see.

### Customer privacy posture

| Data class | OSS engine has | Sent to LLM? |
|---|---|---|
| Diagnostic struct | yes | YES (redacted) |
| K8s event messages | yes (in analyzers) | **Scrubbed via `pkg/ai.RedactEvents` (Sprint 3.4) — identifiers + secret-shaped substrings (AWS keys, Vault hvs.*, JWTs, GitHub PATs, Slack tokens) replaced with `[REDACTED]` before the Layer-2 investigator sees them.** The same redaction now applies to `Diagnostic.Message` so analyzers that copy event text into Findings can't leak secrets through that path either. For Firecrawl: a second LLM synthesizes a generic search query from the already-redacted message; `RedactDiagnostic`/`RedactEventMessage` are the backstop. |
| Pod logs | no | **NO** |
| Secret bytes | no (Srenix never reads) | **NO** |
| Secret key NAMES | yes | YES (already in Diagnostic message) |
| Vault values | no | **NO** |
| Vault key NAMES | yes (via VaultPathMissing) | YES |
| Namespace/name | yes | **HASHED** before LLM input |
| Image SHAs | yes | YES |
| Internal hostnames | yes | **REDACTED** |

The redactor lives in Srenix Enterprise (`ai/redactor.go`). Unit tests assert
no raw identifier leaks into constructed prompts.

---

## New flags in v0.2.0-alpha.1 (Srenix Enterprise)

| Flag | Default | Notes |
|---|---|---|
| `--firecrawl-endpoint` | `https://api.firecrawl.dev` | Firecrawl API base URL; override for self-hosted instances |
| `--firecrawl-enabled` | `true` | Enable web-research in the deep-RCA investigator. Inert without the key env. |
| `--firecrawl-api-key-env` | `FIRECRAWL_API_KEY` | Name of the env var holding the Firecrawl API key |
| `--investigator-web-timeout` | `8s` | Wall-clock cap per Firecrawl HTTP request |
| `--rag-short-circuit` | `true` (previously `false`) | Reuse a prior cleared fix (skip LLM proposal) when cosine similarity ≥ threshold. Inert without `--memory-store-url`. |
| `--rag-short-circuit-threshold` | `0.92` | Minimum cosine similarity for RAG short-circuit reuse |

**ESO setup for the Firecrawl API key** (Vault → ESO → env):
```
Vault:  secret/data/shared/api-keys   key: firecrawl_api_key
ESO:    ExternalSecret srenix-firecrawl-key  (ns: agentic-sre)
        target Secret key: FIRECRAWL_API_KEY
```
See [AI_OPERATOR_RUNBOOK.md §Scenario 8a](AI_OPERATOR_RUNBOOK.md) for the
full ExternalSecret manifest and Deployment patch.

---

## Future: where LLMs could enter the OSS engine (and where they won't)

The rule-based Layer-2 investigator ships in OSS today. It pattern-
matches failures and runs a fixed set of read-only tools; the LLM
versions of the same interface live in Srenix Enterprise. Three additional
places LLMs could plausibly land in OSS `v2+`, *none of which are in
the hot path of the in-cluster install*:

1. **Verified Signature Library curation** (offline, asynchronous):
   LLM-assisted clustering of customer-contributed incident reports to
   propose new fixer signatures. The signatures themselves still ship
   as deterministic Go. The LLM is in our lab, not the customer cluster.

2. **Optional `srenix narrate` subcommand**: Same shape as T0 above, but
   shipped to OSS as a CLI tool that hits an LLM on demand against
   a static `srenix diagnose --format json` output. Customer chooses
   whether to enable it; never in the cron path.

3. **Triage prioritization in the Fleet Console (the SaaS)**: When
   you're managing 50 clusters with 200 active diagnostics, "which 5
   should I fix today" is a ranking problem where light ML helps.
   This is a separate product, not the OSS engine.

When new LLM capability does enter OSS, it will follow the same shape
as today's rule-based Layer-2 investigator:
- **Off by default** in any path that touches cluster state
- **Read-only by construction** — closed action enum, no shell-out
- **Outside the cron loop** that runs on every customer cluster, or
  trivially disabled with a single env var if it lives inside the loop
- **Auditable** — LLM-assisted output goes through the same
  deterministic safety gates as a hand-written signature
- **Privacy-respecting** — customer cluster state never leaves the
  cluster without explicit opt-in

---

## See also

- [AI_TIERS.md](AI_TIERS.md) — definitive tier specification, RBAC matrix, audit event taxonomy
- [THREAT_MODEL_AI.md](THREAT_MODEL_AI.md) — OWASP LLM Top 10 mapping
- [ADVERSARIAL_ANALYSIS.md](ADVERSARIAL_ANALYSIS.md) — full red-team review (AI section in §8)
- [SETUP_GUIDE.md §14](SETUP_GUIDE.md#14-ai-tier-setup) — how to enable AI tiers
- [FAILURE_MODES.md](FAILURE_MODES.md) — the deterministic OSS catalog AI builds on top of
