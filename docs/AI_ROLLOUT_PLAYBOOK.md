# AI Rollout Playbook — W1 → W4 Customer Onboarding

CHA-com v1.0.0 shipped all four LLM tiers day one. v1.4 and v1.5 added
the Layer-1 flake-suppression upgrade to the Endpoints probe and the
Layer-2 Investigator (rule-based in OSS, LLM-backed in CHA-com). This
playbook covers how to introduce them to customers in waves.

**Build vs deploy posture**: Build is single-pass (v1.5.x has all
tiers and both layers). Deploy is staged — tier escalation is a config
decision per customer, not a release cadence.

---

## Wave summary

| Wave | Default tier offered | Trigger | Customer commitment |
|---|---|---|---|
| **W0a: Flake suppression** | Layer-1 (on by default, OSS) | v1.4.0 upgrade | None — auto-engaged on upgrade |
| **W0b: Rule-based investigation** | Layer-2 (on by default, OSS) | v1.5.0 upgrade | None — auto-engaged on upgrade |
| **W0c: Opt-in TLS-secret-mismatch fixer** | OSS fixer (off by default) | v1.3.0+, customer-requested | "Allow CHA to JSON-patch Ingress.spec.tls[].secretName on non-GitOps Ingresses" |
| **W1: Narration** | T0 read-only | v1.0.0 release | "Add LLM enrichment to your Slack/AM alerts" |
| **W1b: LLM-backed investigation** | Layer-2 paid override | After T0 pilot | "Let CHA-com pick investigation tools dynamically instead of by rule" |
| **W2: Click-to-fix** | T1 (opt-in) | First design partner accepts T0 | "Sign off on one-click approved fixes" |
| **W3: Multi-step plans** | T2 (opt-in) | Cascading drift patterns observed | "Step-by-step approved plans" |
| **W4: Break-glass** | T3 (opt-in) | SOC2 + Vault-heavy fleet | "Dual-approval Vault recovery runbooks" |

---

## Per-wave checklists

### W0a — Layer-1 flake suppression (v1.4)

**Default**: On. No customer action required.

**What it does**: The `Endpoints` probe retries transient-class
failures once with a 1.5× timeout, then requires N consecutive failures
(default 2) before escalating to `SeverityCritical`. Deterministic
failures (TLS error, HTTP status mismatch, invalid URL) bypass the
streak counter and emit Critical immediately.

**Acceptance test on upgrade**:
- [ ] Confirm probe results still emit. Look for `[transient, 1/2]` in
      a `SeverityWarning` Finding within the first cycle that follows a
      transient blip — this is the new "first-flake" signature.
- [ ] Confirm that a sustained outage still escalates to Critical
      after the configured streak threshold (default: one extra cycle
      of latency, typically ~10–20 s).

**Tuning** (rare):
- Raise `MinConsecutiveFailures` on `probe.NewEndpoints(...)` if the
  customer's network is consistently flakier than expected.
- Set `RetryOnFlake: false` to disable in-cycle retry entirely (not
  recommended; defeats the purpose).

**Downshift**: Pin to the v1.3 image. There is no Helm value to
disable Layer-1 inside v1.4+ — it is part of the OSS probe contract.

### W0b — Layer-2 rule-based Investigator (v1.5)

**Default**: On. No customer action required. No new RBAC; reuses
the watcher's existing read access to the cluster snapshot.

**What it does**: For every Finding or Diagnostic that escalates to
`SeverityCritical`, the watcher runs the rule-based investigator
after post-fix re-diagnose. The investigator pattern-matches the
failure mode and runs a fixed set of read-only tools (DNS / HTTP /
TLS / describe / events) to produce a one-to-four-sentence summary
and a structured list of observations. The summary is attached to
the DriftReport CR (`spec.investigation`) and rendered under each
finding as a `🔬` block in Slack and Alertmanager.

**Acceptance test on upgrade**:
- [ ] Confirm `🔬` blocks appear on at least one critical finding
      within the first cycle that produces one.
- [ ] Confirm `kubectl get driftreport <name> -o jsonpath='{.spec.investigation}'`
      returns a non-empty summary for that report.
- [ ] Confirm no new audit events of class `ai.*` are emitted by the
      OSS investigator (this is intentional — the rule-based path
      keeps zero-dependency posture).

**Pilot duration**: Two weeks. Success criteria:
- Investigator conclusions are accurate on the customer's known
  failure modes (TLS expiry, ESO target mismatch, slow DNS, etc.).
- No degraded probe latency from the per-cycle 20s investigation cap.

**Downshift**: Set `CHA_INVESTIGATOR=off` on the watcher Deployment
env. The investigator stops running; DriftReports continue without
`spec.investigation`. Behavior matches pre-v1.5 OSS bit-for-bit.

### W0c — Opt-in TLS-secret-mismatch fixer (v1.3)

**Default**: Off. Customer must explicitly opt in.

**Pre-wave**:
- [ ] Customer cluster has at least one `TLSSecretMismatch` analyzer
      finding (otherwise there is no signal to act on).
- [ ] Customer has reviewed the GitOps-skip behavior: Ingresses
      annotated by ArgoCD (`argocd.argoproj.io/instance|tracking-id`),
      Flux (`kustomize.toolkit.fluxcd.io/name|namespace`), or Helm
      (`meta.helm.sh/release-name|namespace`), or carrying
      `app.kubernetes.io/managed-by ∈ {helm, argocd, flux, fluxcd}`,
      are **skipped automatically** — the fixer never patches them.
- [ ] Customer accepts the protected-namespace skip list still applies.

**Activation**:
- [ ] `helm upgrade --set fixers.tlsSecretMismatch.enabled=true`. The
      chart sets `CHA_FIXER_TLS_SECRET_MISMATCH=true` on the watcher
      and adds the `networking.k8s.io/ingresses [patch]` verb to the
      remediator ClusterRole.
- [ ] Verify the fixer registers: `kubectl logs deploy/cha-cluster-health-autopilot`
      should show the fixer in the catalog list at startup.

**Acceptance test**:
- [ ] Inject a synthetic mismatch (Certificate targets `foo-tls-new`,
      Ingress still references `foo-tls-old`). Verify the fixer
      issues exactly one JSON patch and the diagnostic clears in
      the next cycle.
- [ ] Repeat the synthetic case with an `argocd.argoproj.io/instance`
      annotation on the Ingress. Verify the fixer skips and the
      diagnostic remains, with `Reason: GitOps-managed: ...`.

**Pilot duration**: Two weeks. Most clusters see fewer than one
TLS-mismatch incident per week; the value is measured by the
absence of manual `kubectl patch` follow-up rather than by raw
fix count.

**Downshift**: `helm upgrade --set fixers.tlsSecretMismatch.enabled=false`.
The fixer un-registers; the analyzer continues to surface the
mismatch with the suggested `kubectl patch` command.

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

### W1b — LLM-backed Layer-2 Investigator (paid override)

**Pre-wave**:
- [ ] T0 pilot complete (the customer has accepted a redacted
      Diagnostic payload leaving the cluster).
- [ ] Customer has reviewed how the LLM-backed Investigator differs
      from the rule-based one: same `pkg/ai.Investigator` interface,
      same closed `Environment` action surface, same `🔬` rendering —
      only the *tool selection* moves from pattern-match to LLM.

**Activation**:
- [ ] CHA-com image installed; `ai.enabled=true`; `ai.tier=t0` (or
      higher). The paid catalog re-calls `RegisterInvestigator` after
      `catalog.RegisterOSS`, replacing the rule-based one.
- [ ] Verify the override took effect: at least one DriftReport in
      the next cycle should have `spec.investigation` populated with
      output that the rule-based version would not have produced
      (more nuanced summary; tool sequence varies per finding).
- [ ] Verify the new audit events emit:
      `kubectl -n cluster-health-autopilot get events --sort-by=lastTimestamp \
        | grep -E "AIInvestigator(Started|ToolCall|Completed|BudgetExceeded)"`.

**Pilot duration**: 2 weeks. Success criteria:
- Investigator stays within the default 5/hr rate limit.
- Token spend matches the AI_COST_MODEL.md L2 line item within ±50%.
- No `ai.investigator.budget_exceeded` events sustained.

**Downshift**: `--set ai.enabled=false` reverts to the rule-based
investigator. There is no separate `ai.tier` value for L2 — it tracks
whether `ai.enabled=true` and whether the paid binary is the one
running.

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
