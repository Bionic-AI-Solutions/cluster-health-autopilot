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
| `cha_ai_investigations_total` | Counter | `implementation`, `conclusion` | Investigations completed; `implementation ∈ {rule_based, llm}` |
| `cha_ai_investigation_duration_seconds` | Histogram | `implementation` | Per-investigation wall-time |
| `cha_ai_investigation_tool_calls_total` | Counter | `implementation`, `tool` | `Environment` method invocations |

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
