# AI Tier Threat Model (v1.0.0)

Maps the CHA-com AI tier surface to recognized AI safety frameworks.
Each row identifies a class of risk, the framework that names it, the
control implemented in v1.0.0, and the source/test that validates the
control.

**Companion docs**:
- [AI_TIERS.md](AI_TIERS.md) — tier capability specification
- [AI_USAGE.md](AI_USAGE.md) — positioning and what stays AI-free
- [ADVERSARIAL_ANALYSIS.md](ADVERSARIAL_ANALYSIS.md) — red-team review

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

### LLM03 — Training Data Poisoning

**Risk**: Adversarial fine-tuning data could steer model behavior.

**Controls**: BYOM defaults mean operators control which model is in
play; the system does not depend on any specific model's weights for
safety. The OSS positioning ("zero AI in the hot path") means an
attacker who poisoned a model's training data still cannot break the
deterministic engine.

### LLM04 — Model Denial of Service

**Risk**: An attacker who can trigger many diagnostics could exhaust
the LLM endpoint budget or rate limit.

**Controls**:

| Control | Where | Validated by |
|---|---|---|
| Token-bucket rate limiter per tier | `ai/rate_limit.go:RateLimiter` | `ai/rate_limit_test.go` (capacity, refill, per-tier override) |
| Circuit breaker tripping at N consecutive failures | `ai/circuit_breaker.go` | `ai/circuit_breaker_test.go` (4 cases) |
| Response cache keyed by (system prompt + user message + model) | `ai/client/cache.go:CachingClient` | `ai/client/cache_test.go` (7 cases) |
| Cycle-wide enrichment timeout | `internal/watcher/enrich.go:enrichmentTimeout` (default 30s) | `internal/watcher/enrich_test.go:TestEnrichDiagnostics_ContextCancellation` |
| Soft-fail on rate-limit / transport error keeps deterministic flow | `ai/enricher.go:Enrich` returns `(zero, nil)` on `client.ErrTransport`/`ErrRateLimited` | `ai/enricher_test.go:TestEnricher_LLMFailureDoesNotPropagate` |

### LLM05 — Supply Chain Vulnerabilities

**Risk**: Compromise of an LLM provider's account or proxy.

**Controls**:

| Control | Notes |
|---|---|
| BYOM default: operators run the LLM themselves | `--ai-endpoint` has no default; operator must set; SaaS requires `--ai-allow-saas` opt-in |
| API key in Kubernetes Secret (ESO-friendly), never in env vars or args | `values.yaml:ai.apiKey.secretName` |
| All LLM responses validated against schema before use | (see LLM02) |
| `recommendation-only` invariant means a compromised LLM cannot autonomously mutate state | (see LLM08) |

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

### LLM07 — Insecure Plugin Design

**Risk**: Plugins or tools the LLM can call could expand the action
surface unexpectedly.

**Controls**: CHA-com does NOT use LLM tool-use / function-calling
features. The LLM emits a JSON proposal; the proposal is parsed and
matched against a closed enum. There is no callback path from the
LLM into Kubernetes APIs.

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

### LLM10 — Model Theft

Not directly applicable — CHA-com uses LLM endpoints, doesn't ship a
model. BYOM ensures the operator's model stays under their control.

---

## NIST AI RMF mapping

The NIST AI Risk Management Framework defines four functions: Govern,
Map, Measure, Manage. v1.0.0 controls map as follows:

| Function | v1.0.0 evidence |
|---|---|
| **Govern** | Documented opt-in tier model with explicit defaults (ai.enabled=false). Operator policy via Helm values. Audit log of every AI-related event. |
| **Map** | This document + AI_TIERS.md + ADVERSARIAL_ANALYSIS.md §8 enumerate the risk classes per tier. |
| **Measure** | Prometheus metrics (Phase 7 hardening): `cha_ai_actions_proposed_total`, `cha_ai_approvals_granted_total`, `cha_ai_post_apply_verified_total`, `cha_ai_circuit_breaker_state`. Rate limiter exposes hit/miss counters. |
| **Manage** | Circuit breaker auto-disables on threshold failures. Audit-trail events pageable via Alertmanager when failure events fire. Manual reset via operator endpoint. |

---

## ISO/IEC 42001 readiness

ISO/IEC 42001 (AI Management Systems) maps to:

| Clause | v1.0.0 evidence |
|---|---|
| 6.1 Risk management | This threat model + ADVERSARIAL_ANALYSIS.md §8 |
| 6.2 AI objectives | AI_TIERS.md per-tier capability specifications |
| 7.5 Documented information | Full doc set under `docs/AI_*.md` |
| 8.1 Operational planning | SETUP_GUIDE.md §14 + AI_ROLLOUT_PLAYBOOK.md |
| 8.4 Performance evaluation | Audit log (AI_AUDIT_TRAIL.md schema); Prometheus metrics |
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

The following drills must pass before tagging v1.0.0:

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
