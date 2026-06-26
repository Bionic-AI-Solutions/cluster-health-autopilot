# AI Cost Model

How to size LLM endpoint budgets for Srenix Enterprise AI tiers and how to
monitor actual usage in production.

**Companion docs**: [AI_TIERS.md](AI_TIERS.md), [AI_AUDIT_TRAIL.md](AI_AUDIT_TRAIL.md)

---

## Token economics per tier

Estimated per-cycle token usage (10-minute watcher resync, 100 active
diagnostics, ~1KB redacted Diagnostic JSON per LLM call):

| Tier | Calls/cycle | Avg input tokens | Avg output tokens | Per-cycle total | Per-hour (6 cycles) |
|---|---|---|---|---|---|
| T0 narration | 100 (one per diag) | ~400 | ~150 | ~55K | ~330K |
| L2 investigator — rule-based (OSS) | up to ~5 (critical only) | 0 | 0 | 0 | 0 |
| L2 investigator — LLM-backed (paid) | up to ~5 (critical only) | ~500 (~2 KB) | ~125 (~500 B) | ~3.1K | ~18.75K |
| T1 single fix | up to 5 (rate-limited) | ~500 | ~250 | ~3.75K | ~22.5K |
| T2 multi-step | up to 5 plans/hour | ~700 | ~750 | n/a | ~7.25K |
| T3 vault runbook | up to 5 runbooks/hour | ~700 | ~500 | n/a | ~6K |

**Default rate-limit budget**: `ai.rateLimit.actionsPerHour=5`. The
LLM-backed Layer-2 investigator shares this budget under the
investigation key; the rule-based investigator is unrate-limited
because it consumes no tokens.
**Default token budget**: `ai.rateLimit.tokensPerHour=1000000`.

Round-trip latency budget per call: 30 seconds (`enrichmentTimeout`).
At ~50ms per token on a well-provisioned LLM endpoint, that's ample
for both prompt and response of typical size.

---

## Layer-2 Investigator: wall-time and token profile

The Layer-2 Investigator is a sibling of T0–T3, not a step on the
T-tier ladder. It runs only on critical findings, after post-fix
re-diagnose, and its cost profile depends entirely on which
implementation is registered.

**Rule-based (OSS, default)**:
- **Token cost**: zero — no LLM is consulted.
- **Wall-time**: ~50–500 ms per investigation, depending on which
  rules fire (TLS handshake + DNS lookup dominates the high end).
- **Per-cycle ceiling**: 20 s of wall-clock for the whole cycle's
  worth of investigations (`investigationTimeout` in
  `internal/watcher/investigate.go`). Excess items are skipped, not
  retried.
- **Network egress**: the same DNS/TCP/TLS traffic the Endpoints
  probe was already issuing; no new outbound destinations.

**LLM-backed (paid, opt-in)**:
- **Input**: ~2 KB per investigation (redacted Finding/Diagnostic +
  tool transcripts accumulated so far). Roughly 500 input tokens.
- **Output**: ~500 B (tool selection JSON or final summary). Roughly
  125 output tokens.
- **Calls per cycle**: bounded by the rate limit (default 5/hr) and
  the per-cycle 20s wall-clock ceiling. Typical real-world load on a
  healthy cluster is <5 investigations per cycle.
- **Default rate-limit**: 5 investigations per hour, shared with the
  T1+ proposal budget. Investigations exceeding this emit
  `ai.investigator.budget_exceeded` and fall back to whatever the
  finding looked like before investigation.

Either implementation lands on the same DriftReport
`spec.investigation` field and the same `🔬` rendering — only the
cost line changes when you swap.

---

## Provider cost translation (illustrative, May 2026)

Using mainstream-provider list prices (your actual costs depend on
contract terms and volume tiers):

| Provider | Model | Input $/1M | Output $/1M | T0 hourly cost (100 issues, 330K tokens) |
|---|---|---|---|---|
| OpenAI | gpt-4-turbo | $10.00 | $30.00 | ~$5.50/h |
| OpenAI | gpt-3.5-turbo | $0.50 | $1.50 | ~$0.30/h |
| Anthropic | claude-sonnet-4 | $3.00 | $15.00 | ~$1.50/h |
| Bedrock | claude-haiku | $0.25 | $1.25 | ~$0.20/h |
| In-cluster vLLM | Qwen3.5-27B | (your GPU cost) | (your GPU cost) | (already paid for) |

**BYOM default applies** — these are not the Srenix Enterprise recommended
configurations. The recommended path is an in-cluster open-weight
model (Qwen, Llama, Mistral) on operator-supplied GPU, which means
zero per-token marginal cost beyond your existing infrastructure.

---

## Monitoring actual usage (Prometheus metrics)

When Srenix Enterprise is configured with Prometheus scraping enabled (P7
hardening), the following metrics are exposed:

| Metric | Type | Labels | What it tracks |
|---|---|---|---|
| `srenix_ai_llm_calls_total` | Counter | `tier`, `phase`, `result` | Total LLM round-trips |
| `srenix_ai_llm_input_tokens_total` | Counter | `tier`, `phase` | Input tokens consumed |
| `srenix_ai_llm_output_tokens_total` | Counter | `tier`, `phase` | Output tokens generated |
| `srenix_ai_llm_duration_seconds` | Histogram | `tier`, `phase` | Round-trip latency |
| `srenix_ai_proposals_created_total` | Counter | `tier`, `action_kind` | Successful proposals |
| `srenix_ai_proposals_rejected_total` | Counter | `tier`, `reason` | Validator rejections |
| `srenix_ai_approvals_granted_total` | Counter | `tier`, `action_kind` | Approved clicks |
| `srenix_ai_actions_applied_total` | Counter | `tier`, `action_kind`, `post_apply_verified` | Mutations applied |
| `srenix_ai_actions_failed_total` | Counter | `tier`, `action_kind`, `reason` | Mutation failures |
| `srenix_ai_rate_limited_total` | Counter | `tier` | Rate-limit denies |
| `srenix_ai_circuit_breaker_state` | Gauge | (none) | 0=closed, 1=open |
| `srenix_ai_cache_hits_total` | Counter | (none) | Response-cache hits |
| `srenix_ai_cache_misses_total` | Counter | (none) | Response-cache misses |
| `srenix_ai_investigations_total` | Counter | `implementation`, `conclusion` | Investigations completed; `implementation ∈ {rule_based, llm}` |
| `srenix_ai_investigation_duration_seconds` | Histogram | `implementation` | Per-investigation wall-time |
| `srenix_ai_investigation_tool_calls_total` | Counter | `implementation`, `tool` | `Environment` method invocations |

---

## Right-sizing the budget

1. **Start with defaults** (5 actions/hour, 1M tokens/hour). For most
   clusters (≤100 active diagnostics) this is generous.

2. **After 1 week**, check the Prometheus metrics:
   ```promql
   # Hourly token usage by tier
   sum by (tier) (rate(srenix_ai_llm_input_tokens_total[1h]) * 3600 + rate(srenix_ai_llm_output_tokens_total[1h]) * 3600)

   # Rate-limit pressure
   sum by (tier) (rate(srenix_ai_rate_limited_total[1h]) * 3600)
   ```

3. **If you see sustained rate-limits**: raise
   `ai.rateLimit.actionsPerHour`. Each unit represents one Apply Fix
   per hour of headroom.

4. **If you see cache-miss explosion** (`cache_misses_total / cache_hits_total > 10`):
   either diagnostics are highly variable (legit) or the cache TTL is
   too short. Default cache TTL is 5 minutes; raise to 30 min via
   `ai.cache.ttl=30m` for stable clusters.

5. **Cost cap**: if you're using SaaS with a hard budget, set
   `ai.rateLimit.tokensPerHour` to your tolerance. Srenix Enterprise auto-soft-
   fails when the bucket exhausts, so the worst case is degraded
   enrichment, not over-spend.

---

## Failure-mode amplification (Sprint 3 hardening)

The token tables above assume **a clean cluster with stable diagnostics**. Real
clusters amplify cost in three specific ways that the 2026-05-22 adversarial
review flagged. Operators must model these before sizing the SaaS budget.

### A1. Per-cycle cache miss on flapping workloads

`pkg/ai.Cache` deduplicates by `(diagnostic_fingerprint, model)`. A workload
that flaps (crash → recover → crash again every ~10 min) generates a **new
fingerprint per occurrence** because the Finding timestamp is part of the
fingerprint. Cache hit rate collapses to ~0.

**Effect**: 1 flapping workload × 144 cycles/day × 1 investigator call =
~144 investigator calls/day from a single resource. At Anthropic Sonnet rates
(~$0.10 per investigation), this is ~$430/month from **one flapping pod**.

### A2. Investigation rate limiter not gating Layer-2 calls (pre-Sprint-3)

The current `ai/rate_limit.go::Take(tier)` gates **fix proposals**, not
**investigations**. The Layer-2 LLM-backed investigator can be called once per
CRITICAL finding per cycle with no separate budget — only the per-cycle
wall-clock ceiling (`investigationTimeout = 20s`) and the implicit
`actionsPerHour` ceiling (which the investigator doesn't consult) apply.

**Closed by**: Sprint 3.2 — adds `TakeInvestigation(class)` keyed on
`(approver_identity, diagnostic_class)`. Until that ships, **do not enable
Layer-2 LLM-backed mode on a SaaS provider without an external monthly
spend cap.**

### A3. Post-apply failure amplification

If a T1 fix applies and the 60-second re-probe still reports the same
diagnostic, the proposer can be re-invoked. The circuit breaker trips after 3
consecutive failures (see [`Srenix Enterprise/ai/circuit_breaker.go`](../../Srenix Enterprise/ai/circuit_breaker.go)),
but those first 3 attempts each generate a fresh T0+T1 pair = **6 LLM calls
per "stuck" incident** before the breaker closes.

**Mitigation**: tune `circuit_breaker.failure_threshold` (default 3) downward
on cost-sensitive deployments. Setting it to 1 means a single failed apply
trips the breaker, which is over-aggressive for normal operations but
protects against runaway cost during a misconfigured rollout.

### Worst-case planning table

For sizing the **monthly LLM budget ceiling** (not the average), use these
multipliers on the steady-state token estimates above:

| Cluster profile | Amplification multiplier | When |
|---|---:|---|
| Stable, no flapping, 10 CRITICAL/month | **1×** | Most production clusters after burn-in |
| Routine deploys, occasional StuckRSPods | **3–5×** | Standard CI/CD pipeline noise |
| Flapping workload(s), pre-Sprint-3 | **20–50×** | One flapping pod can dominate |
| Pre-Sprint-3 + flapping + Layer-2 LLM on | **50–200×** | Genuine cost-blowup scenario |

**Recommendation**: until Sprint 3 ships, treat the **steady-state cost × 20**
as your budget ceiling, not the average. Set the provider-side monthly spend
cap accordingly. See [docs/design/2026-05-hardening-plan.md](design/2026-05-hardening-plan.md)
§3.2 for the investigation-rate-limiter design.

---

## Operator checklist before enabling AI tiers

- [ ] Pick a provider (default: in-cluster vLLM on gpu-ai endpoint = $0 marginal)
- [ ] Set a hard monthly spend cap in the provider console — at least 20× steady-state
- [ ] Verify `ai.cache.ttl` ≥ 5 minutes (default) and Prometheus `cache_hit_ratio` > 0.5 after 1 week
- [ ] Set `ai.rateLimit.actionsPerHour` and `ai.rateLimit.tokensPerHour` explicitly (do not rely on defaults if you customized cycle frequency)
- [ ] Start with **T0 only** for 2 weeks; measure; then add T1
- [ ] Wait for Sprint 3 before enabling Layer-2 LLM-backed mode on SaaS
- [ ] Configure Slack/AM alerts on `ai.budget.warning` and `ai.budget.exceeded` audit events
- [ ] Document who is authorized to raise the budget cap and the approval gate
