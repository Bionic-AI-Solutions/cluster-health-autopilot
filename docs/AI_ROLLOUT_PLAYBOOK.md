# AI Rollout Playbook — W1 → W4 Customer Onboarding

CHA-com v1.0.0 ships all four AI tiers day one. This playbook covers
how to introduce them to customers in waves.

**Build vs deploy posture**: Build is single-pass (v1.0.0 has all
tiers). Deploy is staged — tier escalation is a config decision per
customer, not a release cadence.

---

## Wave summary

| Wave | Default tier offered | Trigger | Customer commitment |
|---|---|---|---|
| **W1: Narration** | T0 read-only | v1.0.0 release | "Add LLM enrichment to your Slack/AM alerts" |
| **W2: Click-to-fix** | T1 (opt-in) | First design partner accepts T0 | "Sign off on one-click approved fixes" |
| **W3: Multi-step plans** | T2 (opt-in) | Cascading drift patterns observed | "Step-by-step approved plans" |
| **W4: Break-glass** | T3 (opt-in) | SOC2 + Vault-heavy fleet | "Dual-approval Vault recovery runbooks" |

---

## Per-wave checklists

### W1 — T0 narration (first pilot)

**Pre-wave**:
- [ ] Customer has an LLM endpoint reachable from inside their cluster
      (or accepts SaaS opt-in with audit-logged acknowledgment).
- [ ] Customer has reviewed `AI_USAGE.md` and `AI_TIERS.md`.
- [ ] Helm values prepared: `ai.enabled=true`, `ai.tier=t0`,
      `ai.endpoint=<their-llm>`.

**Activation**:
- [ ] Customer's existing CHA install bumped to image
      `docker4zerocool/cha-com:v1.0.0`.
- [ ] Helm upgrade applied with T0 values.
- [ ] Verify `AIEnrichmentApplied` Events appear within one cycle.
- [ ] Verify `🤖` blocks render correctly in customer's Slack channel.

**Pilot duration**: 2 weeks. Success criteria:
- Zero `AIEnrichmentFailed` events sustained for >1 hour (means LLM
  endpoint is healthy).
- Customer SREs report enrichment is "actually useful" (subjective,
  but the discriminator).

**Post-wave retrospective**:
- Were narratives accurate?
- Were narratives too verbose / too sparse?
- Any false-leading enrichments that masked the deterministic
  remediation?

### W2 — T1 click-to-fix

**Pre-wave**:
- [ ] T0 pilot complete; customer ready to commit to one-click fixes.
- [ ] Customer has an Ingress controller with HTTPS + OIDC capability
      (oauth2-proxy or Kong + JWT).
- [ ] Customer has signed off on the AI safety controls in
      `THREAT_MODEL_AI.md`.
- [ ] DNS + cert for the approval-server Ingress host configured.

**Activation**:
- [ ] Approval-server Deployment + Ingress applied.
- [ ] Signing-key Secret confirmed (`kubectl get secret cha-approval-signing-key`).
- [ ] OIDC route configured for `/approve` paths.
- [ ] (Recommended) OPA Gatekeeper installed.

**Acceptance test**:
- [ ] SRE clicks an Apply Fix URL from a real diagnostic; verifies the
      success page renders.
- [ ] Replay same URL → expects 409 Conflict.
- [ ] Inject Error pod in `kube-system` → verify no Apply Fix button
      emitted (proposer refuses protected NS).

**Pilot duration**: 2 weeks. Success criteria:
- ≥3 successful Apply Fix clicks with `post_apply_verified=true`.
- Zero unwanted mutations (admission rejected all out-of-bounds attempts).
- Customer SREs report the workflow is "easier than kubectl exec".

### W3 — T2 multi-step plans

**Pre-wave**:
- [ ] T1 pilot complete; customer's incident patterns include
      multi-resource cascades.
- [ ] Customer SREs trained on the step-by-step approval UX.

**Activation**:
- [ ] `helm upgrade --set ai.tier=t2`.
- [ ] Verify a synthetic two-step incident generates a plan with
      both step buttons visible (step 2 hidden until step 1 verifies).

**Pilot duration**: 4 weeks (T2 is rarer-firing; longer baseline).

**Post-wave retrospective**:
- Median plan length?
- Plans abandoned mid-execution? Why?
- Did any plan re-propose itself after partial success?

### W4 — T3 break-glass

**Pre-wave**:
- [ ] T2 pilot complete; customer has Vault-heavy fleet (≥10 ESOs).
- [ ] Customer has formal SOC2/ISO27001 audit program.
- [ ] `cha.io/approver` group has ≥2 distinct members in customer's
      RBAC.
- [ ] Customer security team has reviewed `THREAT_MODEL_AI.md §LLM08`
      and signed off on T3 specifically.

**Activation**:
- [ ] `helm upgrade --set ai.tier=t3 --set 'ai.t3.allowedPathPrefixes={...}'`.
- [ ] Customer Vault security team reviews and approves the path
      allowlist.

**Acceptance drills**:
- [ ] Same-approver bypass attempt → 409 expected.
- [ ] Too-early second approval → 409 expected.
- [ ] LLM tries to embed a literal value → runbook auto-rejected
      (synthetic test using prompt-corpus).
- [ ] Full dual-approval flow with two distinct approvers, 30-min
      delay, results in a valid `vault kv patch` template the
      operator runs.

**Pilot duration**: 8 weeks (very low-frequency; need broad coverage).

---

## Cross-wave principles

1. **Never auto-escalate**: tier upgrade is always a customer
   config decision, never automatic.

2. **Audit each wave**: pull the `AI_AUDIT_TRAIL.md` evidence package
   for the pilot period; review for unexpected behavior.

3. **Downshift is always cheap**: customers can drop a tier via
   Helm value at any time. In-flight T2/T3 approvals remain valid
   until their TTL expires.

4. **Tier rollback should not require re-installation**: this is
   tested as part of v1.0.0 acceptance. `helm upgrade --set ai.tier=t0`
   is the canonical downshift; full disable is `ai.enabled=false`.

5. **Document every wave's signoff**: store the customer's wave-
   acceptance retrospective alongside their pilot agreement. This
   becomes part of the customer's SOC2 / ISO27001 evidence.
