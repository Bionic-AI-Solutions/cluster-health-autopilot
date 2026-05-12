# AI in Cluster Health Autopilot

The OSS `cha` binary has **zero AI in the hot path**. This is a positioning
commitment, not a transient state. AI capability ships in the **commercial
CHA-com binary** as an opt-in layered tier system. Both binaries share the
same OSS engine; the AI layer is purely additive.

This document explains:
1. What stays AI-free in OSS, forever
2. What AI adds on top in CHA-com, and how it's gated
3. Why we built it this way

---

## 1. The OSS engine is and stays AI-free

| Component | Mechanism | Source |
|---|---|---|
| 6 probes (Ceph, Nodes, Postgres, PVCs, Services, Endpoints) | CRD `.status` field reads, HTTP(S) GET against canonical hostnames | [`internal/probe/`](../internal/probe/) |
| 8 analyzers (`SecretKeyMissing`, `FailingExternalSecrets`, `ProactiveSecretKeyCheck`, `UnprovisionedSecret`, `VaultPathMissing`, `CertExpiry`, `ImagePullAuth`, `IngressCoverage`) | Regex on kubelet events, owner-chain walks, ESO target matching, direct Vault key-name lookups, cert-manager status reads, ingress-vs-endpoint set diff | [`internal/diagnose/`](../internal/diagnose/) |
| 4 fixers (`StaleErrorPods`, `StuckJobsWithBadSecretRef`, `StuckRSPods`, `StuckCertificateRequests`) | Pattern-match conditions → call API verbs from a closed 5-verb whitelist | [`internal/fix/`](../internal/fix/) |
| Watcher event loop | Kubernetes watch + 10s debounce + full probe/analyze cycle + fingerprint dedup | [`internal/watcher/`](../internal/watcher/) |
| Three-channel Slack routing | Subject-prefix dispatch — post-fix CHA-acted set → `#ceph-alerts`, unfixed → `#ceph-critical`, daily digest → `#healthinfo` | [`internal/report/routing.go`](../internal/report/routing.go) |
| Alertmanager hub integration | POST `/api/v2/alerts` every cycle; AM handles dedup, silencing, fan-out | [`internal/report/alertmanager.go`](../internal/report/alertmanager.go) |
| Daily digest | Reads DriftReport CR history; classifies new/persistent/auto-fixed | [`internal/report/daily.go`](../internal/report/daily.go) |
| JWT signing primitives | Ed25519 (EdDSA) via crypto/ed25519; minimal deps | [`pkg/ai/jwt.go`](../pkg/ai/jwt.go) |

Same input → same diagnosis, every time, **auditable from source**. An
SRE can read the entire fix list in an afternoon.

**Even when CHA-com is installed**, the OSS engine code paths are
unchanged. If `ai.enabled=false` (the default), CHA-com behaves
bit-for-bit identically to the OSS `cha` binary. There is no hidden
AI path that activates without operator opt-in.

---

## 2. CHA-com adds AI as four opt-in tiers

Each tier is documented in [AI_TIERS.md](AI_TIERS.md). Quick overview:

| Tier | What it adds | What it does **NOT** add |
|---|---|---|
| **T0 Narration** | LLM-generated 2–4 sentence root-cause narrative under each diagnostic | Any mutation capability. Tier 0 is read-only enrichment. |
| **T1 Single fix** | One-click signed URLs that apply an existing OSS fixer to a specific target | New verbs, new RBAC, autonomous execution |
| **T2 Multi-step plan** | Sequential multi-action plans with per-step approval | Bypass of step-by-step approval; plans cannot self-modify |
| **T3 Vault runbook** | Generated `vault kv patch` runbook + dual-approval recording | CHA-com NEVER writes to Vault. Runbook is human-run. |

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
            → snapshot.Mutator call (same code path as `cha remediate --live`)
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
| LLM sees raw cluster state | LLM06 — Sensitive Information Disclosure | `RedactDiagnostic()` SHA-256-hashes namespace/name; redacts IPs/UIDs/hostnames; Secret bytes never sent (CHA never reads them anyway) |
| SaaS LLM as default | OSS positioning + GDPR | Default `--ai-endpoint` is operator-supplied (BYOM); SaaS (OpenAI/Anthropic) requires explicit `--ai-allow-saas` opt-in with audit-logged acknowledgment |

Full mapping in [THREAT_MODEL_AI.md](THREAT_MODEL_AI.md).

### What competitors get wrong

Many "AI for SRE" products lead with autonomous remediation: the LLM
decides what to do, executes it, and reports after the fact. This
violates LLM08 and OWASP's repeated guidance to gate any agency
behind explicit human authorization.

CHA-com's tiers explicitly refuse this shape. Even T3 (the most
powerful tier) is recommendation-only: it generates a runbook that
two distinct humans approve and then run themselves with values
neither CHA-com nor the LLM ever see.

### Customer privacy posture

| Data class | OSS engine has | Sent to LLM? |
|---|---|---|
| Diagnostic struct | yes | YES (redacted) |
| K8s event messages | yes (in analyzers) | **NO** — too prompt-injection-prone |
| Pod logs | no | **NO** |
| Secret bytes | no (CHA never reads) | **NO** |
| Secret key NAMES | yes | YES (already in Diagnostic message) |
| Vault values | no | **NO** |
| Vault key NAMES | yes (via VaultPathMissing) | YES |
| Namespace/name | yes | **HASHED** before LLM input |
| Image SHAs | yes | YES |
| Internal hostnames | yes | **REDACTED** |

The redactor lives in CHA-com (`ai/redactor.go`). Unit tests assert
no raw identifier leaks into constructed prompts.

---

## Future: where AI could enter the OSS engine (and where it won't)

Three places AI could plausibly land in OSS `v2+`, *none of which are
in the hot path of the in-cluster install*:

1. **Verified Signature Library curation** (offline, asynchronous):
   LLM-assisted clustering of customer-contributed incident reports to
   propose new fixer signatures. The signatures themselves still ship
   as deterministic Go. The LLM is in our lab, not the customer cluster.

2. **Optional `cha narrate` subcommand**: Same shape as T0 above, but
   shipped to OSS as a CLI tool that hits an LLM on demand against
   a static `cha diagnose --format json` output. Customer chooses
   whether to enable it; never in the cron path.

3. **Triage prioritization in the Fleet Console (the SaaS)**: When
   you're managing 50 clusters with 200 active diagnostics, "which 5
   should I fix today" is a ranking problem where light ML helps.
   This is a separate product, not the OSS engine.

When AI does enter OSS, it will be:
- **Off by default** in any path that touches cluster state
- **Outside the cron loop** that runs on every customer cluster
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
