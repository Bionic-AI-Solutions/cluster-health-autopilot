# AI Tier Threat Model (v1.5.2)

Maps the CHA AI surfaces to recognized AI safety frameworks. Each row
identifies a class of risk, the framework that names it, the control
implemented in the current shipping release, and the source/test that
validates the control.

There are two distinct AI surfaces in scope:

1. **Mutation AI tier (T0–T3, CHA-com only, since v1.0.0)** — proposes
   mutations that humans then approve and the deterministic engine
   applies. Reviewed in ADVERSARIAL_ANALYSIS.md §8.
2. **Layer-2 Investigator (since v1.5.0)** — a read-only diagnostic
   agent that runs against a closed-enum `pkg/ai.Environment` of
   read-only tools and attaches a root-cause hint to CRITICAL
   DriftReports. **Two implementations**: a deterministic rule-based
   investigator that ships in OSS (no LLM, most LLM-specific rows are
   N/A), and an LLM-backed investigator that ships in CHA-com and
   inherits the same closed-enum Environment surface and the same
   wall-clock bound. Reviewed in ADVERSARIAL_ANALYSIS.md §9.

The Layer-2 Investigator is structurally simpler than the T0–T3 tier
because it has no approval/mutation path: the surface is dominated by
the Environment interface contract, not by JWT/replay/admission.

**Companion docs**:
- [AI_TIERS.md](AI_TIERS.md) — tier capability specification
- [AI_USAGE.md](AI_USAGE.md) — positioning and what stays AI-free
- [ADVERSARIAL_ANALYSIS.md](ADVERSARIAL_ANALYSIS.md) — red-team review
- [design/2026-05-investigator-agent.md](design/2026-05-investigator-agent.md) — Layer-2 architecture rationale

---

## OWASP LLM Top 10 mapping

### LLM01 — Prompt Injection

**Risk**: An attacker who can influence text that ends up in an LLM
prompt (e.g., crafting a malicious event message via a controlled
ExternalSecret) could embed instructions that override system intent.

**Controls**:

| Control | Where | Validated by |
|---|---|---|
| All untrusted text wrapped in `<observed_data>` blocks | `ai/enricher.go:buildEnricherUserMessage`, `ai/fix_proposer.go:buildProposerUserMessage`, `ai/planner.go`, `ai/vault_runbook.go` | `ai/enricher_test.go:TestEnricher_ScrubsInjectionInMessage` |
| System prompt explicitly declares `<observed_data>` is untrusted | `ai/prompts/system_t*.md` (each tier) | Manual prompt review |
| Input scrubber strips known patterns | `pkg/ai/redact.go:ScrubInjection` (11 regexes: ignore previous, system:, you are now, jailbreak, im_start markers, …) | `pkg/ai/redact_test.go:TestScrubInjection` (7 positive + 1 legit-preserved) |
| Structured-output schema (closed-enum action_kind) rejects free-form action requests | `pkg/ai/validate.go:AIProposedAction.Validate` | `pkg/ai/validate_test.go:TestProposalValidate_BadActionKind` |

**Layer-2 Investigator (v1.5)**:

- *Rule-based investigator (OSS)*: **N/A** — no LLM in the path; tool
  selection is deterministic Go code.
- *LLM-backed investigator (CHA-com)*: same defense as above —
  `<observed_data>` wrapping on every tool output before it is fed
  back to the model; closed-enum tool surface (`pkg/ai.Environment`
  has exactly five methods) parsed at decode time so an unrecognized
  tool name is dropped silently; same `pkg/ai/redact.go:ScrubInjection`
  pre-filter applied to user-controllable strings in the finding.

### LLM02 — Insecure Output Handling

**Risk**: Passing raw LLM output into `kubectl` / Mutator calls without
schema enforcement.

**Controls**:

| Control | Where | Validated by |
|---|---|---|
| JSON-mode response_format on every LLM call | `ai/client/openai.go` | `ai/client/openai_test.go:TestOpenAI_HappyPath` (asserts JSON mode flag) |
| Strict JSON schema (`AIProposedAction`) for action proposals | `pkg/ai/types.go` + `Validate()` | `pkg/ai/validate_test.go` (15 cases) |
| Closed-enum `ActionKind` whitelist | `pkg/ai/types.go:ActionKind`, `IsValid()` | `pkg/ai/validate_test.go:TestActionKindIsValid` (8 cases, 5 valid + 3 rejected) |
| Re-validation at executor entry | `ai/approval/executor.go:Execute` calls `p.Validate()` before any Mutator call | `ai/approval/server_test.go` (executor failure path) |
| Markdown fence stripping prevents prompt-leakage attacks via `\`\`\`json` | `ai/enricher.go:parseEnrichmentResponse`, `ai/fix_proposer.go:parseProposerResponse` | `ai/enricher_test.go:TestEnricher_HandlesMarkdownFences` |
| **Patch-payload allow-list (v1.6 / Sprint 3.1)** — closed-enum ActionKind gates verbs; this gates *shape*. `PatchDeployment` permits exactly `spec.template.metadata.annotations.kubectl.kubernetes.io/restartedAt`; everything else (replicas, selector, container images, immutable fields, additional annotations, oversized payloads) returns `ErrPatchForbidden`. Closes the StatefulSet-replicas-zero data-loss vector. | `CHA-com/ai/approval/patch_validator.go` | `CHA-com/ai/approval/patch_validator_test.go` (10 cases) |

**Layer-2 Investigator (v1.5)** (this is the row that maps most
directly to "Improper Output Handling" in the 2025 OWASP rename):

- *Rule-based investigator (OSS)*: **N/A** — output is a typed Go
  struct (`pkg/ai.InvestigationResult`) constructed by the rule
  engine; no parse step from a model's free-form output.
- *LLM-backed investigator (CHA-com)*: the model's response is
  parsed into a closed schema; the tool-selection field is
  validated against the closed `Environment` enum at parse time
  and malformed entries are dropped (not invoked). On repeated
  malformed output the circuit breaker (`ai/circuit_breaker.go`)
  trips and falls back to the rule-based investigator. The
  investigation summary is written into a structured field on the
  DriftReport CR — never into a command, a shell, or a Mutator
  call. The investigation **cannot** propose or apply a mutation:
  `Environment` has no mutation methods (see LLM06 below).

### LLM03 — Training Data Poisoning

**Risk**: Adversarial fine-tuning data could steer model behavior.

**Controls**: BYOM defaults mean operators control which model is in
play; the system does not depend on any specific model's weights for
safety. The OSS positioning ("zero AI in the *mutation* hot path")
means an attacker who poisoned a model's training data still cannot
break the deterministic engine.

**Layer-2 Investigator (v1.5)**: the rule-based investigator in OSS
is the deterministic fallback for this row — even if every LLM
provider's training data were poisoned, the rule-based investigator
continues to produce a correct (or empty) classification, and the
original Critical finding still surfaces regardless. Investigation
output is *additive*, never authoritative.

### LLM04 — Model Denial of Service

**Risk**: An attacker who can trigger many diagnostics could exhaust
the LLM endpoint budget or rate limit.

**Controls**:

| Control | Where | Validated by |
|---|---|---|
| Token-bucket rate limiter per tier | `ai/rate_limit.go:RateLimiter` | `ai/rate_limit_test.go` (capacity, refill, per-tier override) |
| **Independent investigation budget (v1.6 / Sprint 3.2)** — `TakeInvestigation(class)` keyed on `(approver, diagnostic_class)` with default 10/hour. Previously the proposal budget gated `Take(tier)` but Layer-2 investigations had no separate ceiling; a flapping workload could uncapped-burn ~144 investigations/day per resource. | `CHA-com/ai/rate_limit.go::TakeInvestigation` | `CHA-com/ai/rate_limit_test.go::TestTakeInvestigation_*` (4 cases) |
| **Cold-start mitigation (v1.6 / Sprint 3.3)** — new buckets initialize at 0 tokens, not full capacity, so a pod-restart attacker can't extract a free `ActionsPerHour`-sized burst on each restart. `ColdStartFull: true` re-enables legacy burst behavior for stable, long-running deployments. | `CHA-com/ai/rate_limit.go::newScopedBucket` | `CHA-com/ai/rate_limit_test.go::TestColdStart_*` (3 cases) |
| Circuit breaker tripping at N consecutive failures | `ai/circuit_breaker.go` | `ai/circuit_breaker_test.go` (4 cases) |
| Response cache keyed by (system prompt + user message + model) | `ai/client/cache.go:CachingClient` | `ai/client/cache_test.go` (7 cases) |
| Cycle-wide enrichment timeout | `internal/watcher/enrich.go:enrichmentTimeout` (default 30s) | `internal/watcher/enrich_test.go:TestEnrichDiagnostics_ContextCancellation` |
| Soft-fail on rate-limit / transport error keeps deterministic flow | `ai/enricher.go:Enrich` returns `(zero, nil)` on `client.ErrTransport`/`ErrRateLimited` | `ai/enricher_test.go:TestEnricher_LLMFailureDoesNotPropagate` |

**Layer-2 Investigator (v1.5)** (this is now "Unbounded Consumption"
under the OWASP 2025 LLM10 rename — see also LLM10 below):

- *Rule-based investigator (OSS)*: no token cost; bounded by the
  20-second wall-clock cap per cycle (`ctx.Done()` honored). The
  investigator MUST return whatever was gathered when the deadline
  fires — no extended I/O loops.
- *LLM-backed investigator (CHA-com)*: same 20-second wall-clock
  cap; max 6 tool calls per investigation; existing AI-tier
  rate limiter (`ai/rate_limit.go`) applies to investigation calls
  as well. Investigation runs *only* on CRITICAL findings that
  passed the v1.4 streak gate, so a transient-noise attacker cannot
  amplify into investigation cost.
- Soft-fail per investigation: a failed `Investigate` does **not**
  block the rest of the cycle; the original Critical finding still
  surfaces.

### LLM05 — Supply Chain Vulnerabilities

**Risk**: Compromise of an LLM provider's account or proxy.

**Controls**:

| Control | Notes |
|---|---|
| BYOM default: operators run the LLM themselves | `--ai-endpoint` has no default; operator must set; SaaS requires `--ai-allow-saas` opt-in |
| API key in Kubernetes Secret (ESO-friendly), never in env vars or args | `values.yaml:ai.apiKey.secretName` |
| All LLM responses validated against schema before use | (see LLM02) |
| `recommendation-only` invariant means a compromised LLM cannot autonomously mutate state | (see LLM08) |

**Layer-2 Investigator (v1.5)**:

- *Rule-based investigator (OSS)*: deterministic engine; no
  third-party LLM dependency.
- *LLM-backed investigator (CHA-com)*: BYOM default identical to
  the T0–T3 tier; operator-chosen endpoint; the same closed-enum
  Environment contract prevents a compromised provider from
  inducing a mutation. The Investigator interface has no return
  channel into `pkg/ai.Mutator`.

### LLM06 — Sensitive Information Disclosure

**Risk**: LLM provider sees customer cluster state.

**Controls**:

| Control | Where | Validated by |
|---|---|---|
| `RedactDiagnostic`: SHA-256 hash namespace/name before LLM input | `pkg/ai/redact.go:RedactDiagnostic` | `pkg/ai/redact_test.go:TestRedactDiagnostic_NoLeakBackIntoOutput` (round-trip identifier-leak assertion across 7 identifiers) |
| Identifiers consistently replaced across Message/Remediation | `pkg/ai/redact.go:redactWithIdentifiers` | same test |
| IPs class-tagged (loopback/rfc1918/public) | `pkg/ai/redact.go:redactIP` | `pkg/ai/redact_test.go:TestRedactText_IPs` |
| Secret data fields never read by OSS engine, so never available to send | `pkg/ai/interfaces.go` documents "data class boundary"; `internal/diagnose/proactive_secret_key_check.go` iterates `for k := range secret.Data` only | Code-level enforcement; documented privacy contract |
| Vault values never read; only key NAMES via `vault.Client.ListKeys` | `internal/vault/client.go` | Existing OSS posture (v0.2 forward) |
| Internal hostnames hashed, .svc/.local suffix preserved as type signal | `pkg/ai/redact.go:redactHost` | `pkg/ai/redact_test.go:TestRedactText_InternalHosts` |
| Cluster domain → `<cluster>` placeholder | `pkg/ai/redact.go:clusterDomainRE` | `pkg/ai/redact_test.go:TestRedactText_ClusterDomain` |
| **Event-message scrubbing (v1.6 / Sprint 3.4)** — `RedactEvents` applies identifier redaction PLUS secret-heuristic substitution (AWS access keys, Vault `hvs.*` tokens, JWTs, GitHub PATs, Slack tokens, long base64/hex) to Kubernetes event `.Message` fields before they reach the Layer-2 LLM-backed investigator. Wired into `LiveEnvironment.GetEvents`. | `pkg/ai/redact.go:RedactEvents`, `internal/investigator/env_live.go:GetEvents` | `pkg/ai/redact_test.go::TestRedactEventMessage_*` and `TestRedactEvents_*` (7 cases) |
| **`Diagnostic.Message` secret-heuristic scrub (v1.6 / Sprint 3.4b)** — analyzers that copy event text into a Diagnostic's Message field now also pass through the secret-heuristic substitution via the same `redactText` helper. Closes the leak path where an analyzer captures a kubelet event verbatim. | `pkg/ai/redact.go:redactText` (extended) | `pkg/ai/redact_test.go::TestRedactDiagnostic_ScrubsSecretsInMessage`, `TestRedactDiagnostic_ScrubsSecretsInRemediation` |
| **Audit-trail tamper evidence (v1.6 / Sprint 3.6)** — `ChainedSink` wraps any `AuditSink` and embeds `prev_hash` + `entry_hash` SHA-256 fields. `VerifyChain` walks the chain and returns the first broken-link index — detecting content mutation, reordering, and insertion/deletion. Not a signing scheme (tamper *evidence*, not resistance); layer over an append-only Vault audit device for full resistance. | `CHA-com/ai/audit/hash_chain.go` | `CHA-com/ai/audit/hash_chain_test.go` (7 cases) |

**Layer-2 Investigator (v1.5)**:

- *Rule-based investigator (OSS)*: no third-party LLM endpoint; data
  never leaves the watcher pod. RBAC ceiling = watcher SA = read-only
  on namespaced resources. `Environment.Describe` and
  `Environment.GetEvents` route through `snapshot.Source`, which
  preserves the v0.2 privacy contract (`for k := range secret.Data`).
- *LLM-backed investigator (CHA-com)*: every tool output is passed
  through `pkg/ai.ContainsSecretLike` (base64≥40, hex≥32 patterns
  per `pkg/ai/redact.go`) before being added to the prompt; the
  investigation summary is scrubbed before it is written into the
  DriftReport. `Environment` exposes **no Secret read** — there is
  no method that returns Secret values. Vault is not touched.

### LLM07 — Insecure Plugin Design

**Risk**: Plugins or tools the LLM can call could expand the action
surface unexpectedly.

**Controls**: For the T0–T3 mutation tier, CHA-com does NOT use LLM
tool-use / function-calling features. The LLM emits a JSON proposal;
the proposal is parsed and matched against a closed enum. There is no
callback path from the LLM into Kubernetes APIs.

**Layer-2 Investigator (v1.5)**: the LLM-backed investigator *does*
issue tool calls — but the toolbox is the closed-enum
`pkg/ai.Environment` interface (`DNSLookup`, `HTTPProbe`,
`TLSInspect`, `Describe`, `GetEvents`) and **every method on
`Environment` is read-only**. There is no `ApplyManifest`, no
`PatchResource`, no shell, no file write. Adding a tool to
`Environment` is a versioned design decision that requires a code
change, a corresponding RBAC verb in the chart, a prompt-schema
change, and test coverage. The interface contract is the load-bearing
control for this row; see ADVERSARIAL_ANALYSIS.md §9.

In OWASP-2025 vocabulary this is the **"Improper Output Handling"**
defense for tool selection: tool name is parsed against the closed
enum at decode time, malformed entries dropped, circuit-broken after
N consecutive failures.

The rule-based investigator in OSS uses the same `Environment`
interface but selects tools deterministically — it inherits the
same closed-enum boundary by construction.

#### OWASP 2025 LLM07 — System Prompt Leakage (sub-row)

**Risk**: An attacker exfiltrates the system prompt to reverse-engineer
controls or discover safety scaffolding.

**Controls**:

- For the T0–T3 tier, system prompts live in `ai/prompts/system_t*.md`
  and ship in the public OSS repository. They are **not secret**;
  every safety property is implemented in code (validator, admission,
  approval JWT) rather than in the prompt. Prompt leakage discloses
  no privileged information.
- For the Layer-2 Investigator (LLM-backed, CHA-com): the investigator
  prompts will be checked into `ai/prompts/` and are similarly public.
  Every safety property — closed-enum `Environment`, 20-second
  wall-clock cap, read-only RBAC ceiling, scrubber — is enforced in
  Go code outside the prompt. The model cannot escape the
  `Environment` interface by leaking or rewriting its own prompt.

### LLM08 — Excessive Agency

**Risk**: The headline AI/SRE failure mode — giving the LLM
unsupervised mutation capability.

**Controls** (the load-bearing controls for the entire product):

| Control | Where | Validated by |
|---|---|---|
| Mutator interface never called from LLM response path | Code architecture; LLM output → Validator → Signer → human click → Approver → Mutator | `ai/approval/server_test.go` (every test traces the click-through-execute chain) |
| Signed JWT approval URLs with 15-min TTL | `ai/approval/signer.go`, `pkg/ai/jwt.go:VerifyToken` | `pkg/ai/jwt_test.go:TestVerify_Expired` |
| One-time-use JTI replay protection | `ai/approval/replay.go:InMemoryStore.MarkUsed` | `ai/approval/replay_test.go:TestInMemoryStore_MarkOnceThenReject`; `ai/approval/server_test.go:TestApprove_TokenReplay` (replay rejected + executor not re-called) |
| Admission policy re-checks protected NS at executor entry | `ai/approval/executor.go:Execute` calls `DefaultAdmissionPolicy.Admit` | `pkg/ai/validate_test.go:TestProposalValidate_ProtectedNamespace` (every protected NS) |
| OPA/Gatekeeper third gate (optional but recommended) | `charts/.../templates/gatekeeper-constraint.yaml` | Manual deployment verification |
| T3 dual-approval with distinct-approver enforcement + 30-min audit window | `ai/approval/runbook_store.go:RecordApproval` | `ai/approval/runbook_store_test.go:TestRunbookStore_SameApproverRejected`, `TestRunbookStore_TooEarlyRejected` |
| Closed-enum ActionKind whitelist matches existing OSS fixer verbs | `pkg/ai/types.go:ActionKind` | `pkg/ai/validate_test.go:TestActionKindIsValid` |
| Watcher SA cannot read signing-key Secret | `charts/.../templates/approval-server-rbac.yaml` (separate RoleBinding) | Helm template render validation |
| Approval-server SA isolated from watcher SA | `charts/.../templates/approval-server-serviceaccount.yaml` | Helm template structure |
| NetworkPolicy restricts approval-server ingress to the gateway/oauth2-proxy namespace (P2.6b, opt-in) | `internal/operator/approval_builders.go:BuildApprovalServerNetworkPolicy`, `charts/.../templates/approval-server-networkpolicy.yaml` | `internal/operator/approval_networkpolicy_test.go` |

**Header-trust model (X-Forwarded-User).** The approval-server attributes
each approve/deny click to the SRE named in the `X-Forwarded-User`
request header, which **oauth2-proxy injects at the OIDC ingress** after
a successful login. The server's `ClusterIP` Service, however, is
reachable by any pod in the cluster; a pod hitting it directly bypasses
the ingress and can forge an arbitrary `X-Forwarded-User`. Because the
click still requires a valid one-time signed token, this is **not** an
authorization bypass — but it lets a hostile in-cluster workload corrupt
the audit trail's "who approved this" field. The opt-in
`spec.approval.networkPolicy` (default OFF; `gatewayNamespaceSelector`
**required** when enabled — no safe default) closes this by dropping all
ingress to port 8443 except from the gateway namespace, so the only
`X-Forwarded-User` the server ever sees is the one oauth2-proxy set.
**Operator requirement:** label your gateway/oauth2-proxy namespace and
point `gatewayNamespaceSelector` at it (e.g.
`{kubernetes.io/metadata.name: <gateway-ns>}`), and ensure the cluster
CNI actually enforces NetworkPolicy.

**Layer-2 Investigator (v1.5)** — this is the headline failure mode
and the structural reason the Investigator can ship in OSS without an
approval gate:

| Control | Where | Notes |
|---|---|---|
| `Environment` interface exposes ZERO mutation methods | `pkg/ai/environment.go:Environment` | The five methods (`DNSLookup`, `HTTPProbe`, `TLSInspect`, `Describe`, `GetEvents`) all return values; none mutate state |
| Investigator cannot construct or invoke a `Mutator` | Interface contract | `Investigator.Investigate(ctx, finding, env)` returns `InvestigationResult` only — no mutation channel |
| Hallucinated tool name dropped at parse time | LLM-backed investigator parser | Closed-enum matched at decode; unrecognized tools silently dropped (not invoked, not surfaced as an action) |
| Investigation output is *additive*, not authoritative | DriftReport CR shape | The original finding's severity and message remain; the investigation result is attached as a hint |
| No RBAC verbs added by the investigator | `charts/.../clusterrole-reader.yaml` | `Describe`/`GetEvents` route through the watcher's existing read-only RBAC |

The headline AI-SRE failure mode (LLM escapes its sandbox and applies
an unsupervised mutation) is refuted **at the interface level**, not
by an approval gate. The rule-based and LLM-backed implementations
share the same `Environment` and therefore the same ceiling.

### LLM09 — Overreliance

**Risk**: SREs trust LLM-proposed fixes without review.

**Controls**:

| Control | Notes |
|---|---|
| Rationale field in every proposal surfaces the LLM's reasoning to the approver | Proposal flow includes `rationale` shown in approval-server's success page |
| Rollback field is required (validator rejects proposals without it) | `pkg/ai/validate.go:AIProposedAction.Validate` |
| Per-step approval in T2 prevents "rubber-stamping" multi-action plans | Step N+1 button only appears after step N's post-apply verify passes |
| T3 dual-approval forces second-opinion review for Vault recovery | `MinT3Delay=30min` mandatory audit window between approvals |
| Audit log of every approval click (who, when, source_ip) | `ai/approval/server.go:handleApprove` writes `ai.approval.granted` events |
| Post-apply verification re-runs analyzers; failures auto-trip circuit breaker | `ai/post_apply.go:PostApplyVerifier`, `ai/circuit_breaker.go` |

**Layer-2 Investigator (v1.5)** (this row maps to "Misinformation" in
the OWASP 2025 rename):

- The Investigator's classification is a **hint, not authoritative**.
  The original Critical finding remains on the DriftReport with its
  original severity and message regardless of whether the
  investigation succeeded, failed, or produced a wrong root cause.
  An operator who disagrees with the hint still has the underlying
  finding in front of them.
- The rule-based investigator in OSS produces deterministic output —
  same finding produces the same hint. No model drift.
- For the LLM-backed investigator: the conclusion field is bounded
  to a closed enum (`pkg/ai.Conclusion`), so the model cannot
  surface a free-text "root cause" that the UI then renders as
  truth. Free-text reasoning is captured in a separate observations
  array marked as untrusted.

### LLM10 — Unbounded Consumption (Model Theft in pre-2025 OWASP)

For pre-2025 OWASP "Model Theft": not directly applicable — CHA uses
LLM endpoints, doesn't ship a model. BYOM ensures the operator's
model stays under their control.

For OWASP 2025 "Unbounded Consumption" (which absorbed the older
LLM04 DoS scope): see the LLM04 row above, which covers token-bucket
rate limiting, the response cache, circuit-breaker tripping, and the
Layer-2 investigator's 20-second wall-clock cap plus 6-tool-call
ceiling.

---

## NIST AI RMF mapping

The NIST AI Risk Management Framework defines four functions: Govern,
Map, Measure, Manage. v1.0.0 controls map as follows:

| Function | v1.5.2 evidence |
|---|---|
| **Govern** | Documented opt-in tier model with explicit defaults (T0–T3: `ai.enabled=false`; Layer-2 LLM-backed: same gate; Layer-2 rule-based: on by default in OSS but read-only and bounded). Operator policy via Helm values. Audit log of every AI-related event. |
| **Map** | This document + AI_TIERS.md + ADVERSARIAL_ANALYSIS.md §8 (T0–T3) and §9 (Layer-2 Investigator) enumerate the risk classes per surface. |
| **Measure** | Prometheus metrics: `cha_ai_actions_proposed_total`, `cha_ai_approvals_granted_total`, `cha_ai_post_apply_verified_total`, `cha_ai_circuit_breaker_state`. Layer-2 adds `cha_investigations_total{conclusion=...}` and `cha_investigation_duration_seconds`. Rate limiter exposes hit/miss counters. |
| **Manage** | Circuit breaker auto-disables on threshold failures (both T0–T3 and Layer-2 LLM-backed). Audit-trail events pageable via Alertmanager when failure events fire. Manual reset via operator endpoint. Layer-2 soft-fails per investigation without breaking the cycle. |

---

## ISO/IEC 42001 readiness

ISO/IEC 42001 (AI Management Systems) maps to:

| Clause | v1.5.2 evidence |
|---|---|
| 6.1 Risk management | This threat model + ADVERSARIAL_ANALYSIS.md §8 (T0–T3) and §9 (Layer-2 Investigator) |
| 6.2 AI objectives | AI_TIERS.md per-tier capability specifications; design/2026-05-investigator-agent.md for Layer-2 |
| 7.5 Documented information | Full doc set under `docs/AI_*.md` plus `docs/design/2026-05-investigator-agent.md` |
| 8.1 Operational planning | SETUP_GUIDE.md §14 + AI_ROLLOUT_PLAYBOOK.md |
| 8.4 Performance evaluation | Audit log (AI_AUDIT_TRAIL.md schema); Prometheus metrics including Layer-2 counters |
| 9.2 Internal audit | Audit-trail events traceable per correlation_id |
| 9.3 Management review | AI_ROLLOUT_PLAYBOOK.md per-wave retrospective template |

---

## SOC 2 CC mapping

The audit trail (per [AI_AUDIT_TRAIL.md](AI_AUDIT_TRAIL.md)) is
structured to satisfy:

- **CC7.2** Anomaly detection: circuit breaker trip events surface as
  Warning Kubernetes Events, routed by Alertmanager to oncall.
- **CC7.3** Security incident handling: every AI-related event has a
  `correlation_id` linking proposal → approval → action → result.
- **CC6.1** Logical access: approver identity verified via OIDC at the
  Ingress; the approver's `username` is recorded in audit on every
  approval.
- **CC6.2** Authentication: JWT signed by an in-cluster Ed25519 key
  stored in a Secret only the approval-server SA can read.

For long-term audit retention (Events are GC'd at 1h by kubelet), the
Loki/OTLP sink in `ai/audit/` is the recommended production
configuration.

---

## Validation drills

The following drills must pass before tagging the current release:

| Drill | What it asserts | Test |
|---|---|---|
| Prompt-injection corpus | 50 known-malicious patterns rejected by redactor + validator + admission together | `docs/PROMPT_INJECTION_CORPUS.md` (forthcoming, P7 hardening) |
| Cost / rate-limit burst | 1000-diagnostic burst → rate limiter caps actual LLM calls; deterministic output continues | `ai/rate_limit_test.go` (synthetic), production-equivalent in pilot |
| Audit trail integrity | Every LLM call, proposal, approval, applied result traceable; no orphan records | Audit sink unit tests + end-to-end manual verification |
| Replay attack | URL clicked twice → 409, executor not re-called | `ai/approval/server_test.go:TestApprove_TokenReplay` |
| Expiry attack | URL clicked 16 min after issue → 410 | `ai/approval/server_test.go:TestApprove_ExpiredToken` |
| Tampering attack | Payload modified → 403 | `ai/approval/server_test.go:TestApprove_TamperedToken` |
| Protected-NS bypass attempt | Token pointed at `kube-system` → admission rejects | Admission policy unit tests |
| T3 same-approver replay | Same approver clicks both slots → 409 | `ai/approval/runbook_store_test.go:TestRunbookStore_SameApproverRejected` |
| T3 too-early second approval | Second click before 30-min → 409 | `ai/approval/runbook_store_test.go:TestRunbookStore_TooEarlyRejected` |
| LLM endpoint down | Deterministic diagnostics continue unaffected | `ai/enricher_test.go:TestEnricher_LLMFailureDoesNotPropagate` |
| Prompt injection in event message | redactor + validator + admission together reject | Per `pkg/ai/redact_test.go:TestScrubInjection` + admission |

**Layer-2 Investigator drills (v1.5)**:

| Drill | What it asserts | Test |
|---|---|---|
| Wall-clock cap honored | `Investigate(ctx, ...)` with a 20-second deadline returns within bound | `pkg/ai/investigator_test.go` |
| Unknown tool name dropped | LLM-backed parse of an unrecognized tool does NOT invoke anything | Parser unit test (CHA-com) |
| Environment interface is read-only | No method on `pkg/ai.Environment` mutates state | Code review + Go interface assertion |
| Investigator cannot reach `Mutator` | The Investigator interface's signature does not expose a Mutator | Code review |
| Investigation soft-fails | `Investigate` returning an error does NOT block the rest of the cycle; original Critical finding still surfaces | Watcher integration test |
| Streak gate respected | Sub-threshold (Warning) findings never trigger the investigator | `internal/watcher/*_test.go` (Layer-1+Layer-2 wiring) |
| Tool output scrubbed for secrets (LLM-backed) | Outputs containing base64≥40 / hex≥32 substrings are redacted before reaching the model and before being written to the DriftReport | `pkg/ai/redact_test.go` |
| Critical finding always surfaces | Even when the investigator's classification is wrong or empty, the original finding's severity and message are unchanged on the DriftReport | DriftReport reconciler test |
