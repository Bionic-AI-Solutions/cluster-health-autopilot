# AI Cost Model

How to size LLM endpoint budgets for CHA-com AI tiers and how to
monitor actual usage in production.

**Companion docs**: [AI_TIERS.md](AI_TIERS.md), [AI_AUDIT_TRAIL.md](AI_AUDIT_TRAIL.md)

---

## Token economics per tier

Estimated per-cycle token usage (10-minute watcher resync, 100 active
diagnostics, ~1KB redacted Diagnostic JSON per LLM call):

| Tier | Calls/cycle | Avg input tokens | Avg output tokens | Per-cycle total | Per-hour (6 cycles) |
|---|---|---|---|---|---|
| T0 narration | 100 (one per diag) | ~400 | ~150 | ~55K | ~330K |
| T1 single fix | up to 5 (rate-limited) | ~500 | ~250 | ~3.75K | ~22.5K |
| T2 multi-step | up to 5 plans/hour | ~700 | ~750 | n/a | ~7.25K |
| T3 vault runbook | up to 5 runbooks/hour | ~700 | ~500 | n/a | ~6K |

**Default rate-limit budget**: `ai.rateLimit.actionsPerHour=5`.
**Default token budget**: `ai.rateLimit.tokensPerHour=1000000`.

Round-trip latency budget per call: 30 seconds (`enrichmentTimeout`).
At ~50ms per token on a well-provisioned LLM endpoint, that's ample
for both prompt and response of typical size.

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

**BYOM default applies** — these are not the CHA-com recommended
configurations. The recommended path is an in-cluster open-weight
model (Qwen, Llama, Mistral) on operator-supplied GPU, which means
zero per-token marginal cost beyond your existing infrastructure.

---

## Monitoring actual usage (Prometheus metrics)

When CHA-com is configured with Prometheus scraping enabled (P7
hardening), the following metrics are exposed:

| Metric | Type | Labels | What it tracks |
|---|---|---|---|
| `cha_ai_llm_calls_total` | Counter | `tier`, `phase`, `result` | Total LLM round-trips |
| `cha_ai_llm_input_tokens_total` | Counter | `tier`, `phase` | Input tokens consumed |
| `cha_ai_llm_output_tokens_total` | Counter | `tier`, `phase` | Output tokens generated |
| `cha_ai_llm_duration_seconds` | Histogram | `tier`, `phase` | Round-trip latency |
| `cha_ai_proposals_created_total` | Counter | `tier`, `action_kind` | Successful proposals |
| `cha_ai_proposals_rejected_total` | Counter | `tier`, `reason` | Validator rejections |
| `cha_ai_approvals_granted_total` | Counter | `tier`, `action_kind` | Approved clicks |
| `cha_ai_actions_applied_total` | Counter | `tier`, `action_kind`, `post_apply_verified` | Mutations applied |
| `cha_ai_actions_failed_total` | Counter | `tier`, `action_kind`, `reason` | Mutation failures |
| `cha_ai_rate_limited_total` | Counter | `tier` | Rate-limit denies |
| `cha_ai_circuit_breaker_state` | Gauge | (none) | 0=closed, 1=open |
| `cha_ai_cache_hits_total` | Counter | (none) | Response-cache hits |
| `cha_ai_cache_misses_total` | Counter | (none) | Response-cache misses |

---

## Right-sizing the budget

1. **Start with defaults** (5 actions/hour, 1M tokens/hour). For most
   clusters (≤100 active diagnostics) this is generous.

2. **After 1 week**, check the Prometheus metrics:
   ```promql
   # Hourly token usage by tier
   sum by (tier) (rate(cha_ai_llm_input_tokens_total[1h]) * 3600 + rate(cha_ai_llm_output_tokens_total[1h]) * 3600)

   # Rate-limit pressure
   sum by (tier) (rate(cha_ai_rate_limited_total[1h]) * 3600)
   ```

3. **If you see sustained rate-limits**: raise
   `ai.rateLimit.actionsPerHour`. Each unit represents one Apply Fix
   per hour of headroom.

4. **If you see cache-miss explosion** (`cache_misses_total / cache_hits_total > 10`):
   either diagnostics are highly variable (legit) or the cache TTL is
   too short. Default cache TTL is 5 minutes; raise to 30 min via
   `ai.cache.ttl=30m` for stable clusters.

5. **Cost cap**: if you're using SaaS with a hard budget, set
   `ai.rateLimit.tokensPerHour` to your tolerance. CHA-com auto-soft-
   fails when the bucket exhausts, so the worst case is degraded
   enrichment, not over-spend.
