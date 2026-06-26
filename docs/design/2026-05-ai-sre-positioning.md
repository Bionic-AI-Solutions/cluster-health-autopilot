# AI SRE Positioning — Implementation Plan (2026-05-27)

> **STATUS: ✅ SHIPPED — Workstreams complete across v1.6.x–v1.8.0 (May 2026)** _(P4.1 honest-header pass, 2026-06-11)_
>
> Workstream B's drift classes landed in **v1.7.0** (PR; CHANGELOG [1.7.0] 2026-05-27: "Closes Workstream B of the AI SRE positioning plan" — `GitOpsDrift`, `WorkloadStateDrift`, `RBACDrift` + `LLMFixerMatcher`); the Layer-2 investigator landed in **v1.5.0** (PR #55); remaining B4/B5/B6 classes + the operator port folded into the **v1.8.0** release (CHANGELOG [1.8.0]). The plan's drift-class-breadth and AI-SRE-repositioning goals are delivered. No material as-shipped delta.
>
> Body below is the original plan, preserved for context.

---

**Status:** Active — drives v1.6.3 (this week) through v1.8 (Q3).
**Driver:** GTM feedback received 2026-05-27 from external reviewer.
**Companion:** `2026-05-trigger-expansion-roadmap.md` (still valid; this plan
adds drift-class breadth on top).

## The feedback in one line

> *"Not advertising a SaaS model, but trying to uplevel from cluster health
> analyzer to an AI SRE play. Extend beyond Vault/ESO. Move away from
> rule-based language."*

Three workstreams below, each broken into PR-sized work items.

---

## Workstream A — Repositioning ("AI SRE for K8s", not "cluster analyzer")

### Goal
Move Srenix's category claim from *cluster monitoring* (Datadog / Prometheus
adjacent) to *agentic SRE* (Komodor / Robusta / Causely / PagerDuty AIOps
adjacent). The product already fits the new category; only the framing is
wrong.

### Differentiation hooks (what we have that the AI-SRE competitive set
doesn't)

  - **In-cluster execution** — no SaaS dependency
  - **Actual mutation** with operator approval — others observe only
  - **Bring-your-own-LLM** (OpenAI-compatible endpoint) — no vendor lock-in
  - **Hash-chained audit + JWT-signed actions** — compliance posture
  - **Open core** — runtime is Apache-2.0; AI tiers are commercial

### Work items (priority order)

A1. **Homepage rewrite.** [`srenix-website/src/pages/index.astro`]
  - Hero: replace *"self-healing operational layer"* with *"Autonomous SRE
    for Kubernetes — policy-bounded, audit-anchored, bring-your-own-LLM."*
  - Above-the-fold: live demo loop showing T0 → T1 → T2 → T3 flow
    (use yesterday's verified output as the source).
  - "How it works" section: agent ↔ runtime ↔ operator-policy diagram
    (instead of the current probe-list grid).
  - **Target: 2026-06-03.** Effort: 0.5 day.

A2. **Pricing tier rename.** [`srenix-website/src/pages/pricing.astro`]
  - `OSS` → `Srenix Open Core (cluster runtime)`
  - `Srenix Enterprise Team` → `AI SRE Team`
  - `Srenix Enterprise Enterprise` → `AI SRE Enterprise`
  - Federal track keeps its name.
  - Update CTAs accordingly.
  - **Target: 2026-06-03.** Effort: 0.25 day.

A3. **Comparison page.** [`srenix-website/src/pages/compare/` — new directory]
  - `index.astro`: tabular comparison vs Komodor / Robusta / Causely on
    in-cluster execution, mutation, audit, LLM bring-your-own,
    open-source posture.
  - Per-competitor sub-pages with longer-form "why Srenix" framing.
  - **Target: 2026-06-10.** Effort: 1 day.

A4. **Demo refresh.** [`agentic-sre/demo/`]
  - Current `run-demo-v2.sh` emphasises deterministic fixers.
  - New `run-demo-v3.sh`: synthetic broken pod → DriftReport →
    T1 fix-proposal flowing into click-to-fix URL → operator approves →
    fix applied → DriftReport reconciled away.
  - Add T3 vault runbook demo (dual-approval timeline, JWT signing).
  - **Target: 2026-06-10.** Effort: 1 day.

A5. **Customer collateral.** [new — slack-able PDF / one-pager]
  - "Why agentic SRE > AIOps observability" — half-page rationale,
    half-page architecture diagram.
  - Live cluster screenshot of T0 enrichment in a Slack alert.
  - **Target: 2026-06-17.** Effort: 0.5 day.

### Acceptance criteria
  - Homepage uses "Autonomous SRE" language; AI tiers are above the fold.
  - Pricing tiers renamed.
  - `/compare` page lists 3+ named competitors with concrete differentiators.
  - Demo runs end-to-end in ≤ 10 minutes against a fresh cluster.

---

## Workstream B — Drift-class expansion (beyond Vault/ESO)

### Goal
Broaden the analyzer/probe catalog from its current secret-heavy weighting
to span six drift classes. Each class becomes a category of probes that
the LLM Investigator can query. The agent's "knowledge of the cluster"
grows accordingly.

### Six drift classes (priority order)

B1. **GitOps drift.** [`internal/diagnose/gitops_drift.go` — new]
  - Argo CD: `Application.status.sync.status` not `Synced`, or
    `health.status` not `Healthy` past a grace window.
  - Flux: `Kustomization` / `HelmRelease` not `Ready=True`, with
    last-reconcile error message.
  - Helm release: `cluster-live` values vs `chart-source` values — flag
    drift on deployments where it matters (no annotation says
    "drift OK").
  - **Target: v1.7 (2026-06-17).** Effort: 4 days.

B2. **Workload state drift.** [`internal/diagnose/workload_state_drift.go`
   — new; extends existing `StatefulSetReplicaPressure` paid analyzer]
  - CNPG: follower replication-lag > threshold, primary divergence.
  - etcd: member-quorum drift (we have an ETCD probe today; expand to
    quorum + leader-election thrash).
  - Kafka (when present): ISR shrink, unclean leader election.
  - StatefulSet ordinal-zero stuck after a failed rollout (currently
    detected as a sub-case of CrashLoopBackOff; promote to its own class).
  - **Target: v1.7 (2026-07-01).** Effort: 5 days.

B3. **RBAC drift.** [`internal/diagnose/rbac_drift.go` — new]
  - Role / RoleBinding / ClusterRoleBinding changed out-of-band
    (`last-applied-configuration` vs live-spec diff).
  - ServiceAccount lost a binding (e.g. workload was bound to a Role
    that no longer exists or no longer references it).
  - NetworkPolicy coverage gaps: workloads in protected namespaces with
    no ingress NetworkPolicy.
  - **Target: v1.7 (2026-07-08).** Effort: 4 days.

B4. **Config drift.** [`internal/diagnose/config_drift.go` — new]
  - ConfigMap content-hash differs across replicas (rolling update never
    completed, or someone `kubectl edit`'d a single replica).
  - CRD version mismatch between cluster-installed CRDs and the version
    referenced in DriftReport custom resources.
  - Helm release values vs cluster-live (overlap with B1 but at the
    individual-resource level).
  - **Target: v1.8 (Q3).** Effort: 4 days.

B5. **Capacity drift.** [`internal/diagnose/capacity_drift.go` — new]
  - Pod resource-request vs actual-usage divergence (idle waste / OOM
    risk). Requires metrics-server or kube-state-metrics.
  - PVC growth-trajectory: linear-fit on PVC fill-rate; flag if free
    space drops below 14 days at current rate.
  - HPA min/max divergence: HPA pinned at minReplicas for > 30 days
    suggests it's not actually being load-driven; HPA pinned at
    maxReplicas means under-provisioned.
  - **Target: v1.8 (Q3).** Effort: 5 days (metrics-server integration is
    the time sink).

B6. **Security drift.** [`internal/diagnose/security_drift.go` — new]
  - Pod Security Standards downgrade: namespaces previously at
    `restricted` slipping to `baseline` / `privileged`.
  - Missing image attestation: workloads running images without a Cosign
    / Notation signature when a policy requires one.
  - NetworkPolicy coverage gaps in production namespaces (overlap with
    B3; this is the policy-gap class specifically).
  - **Target: v1.8 (Q3).** Effort: 4 days.

### Pattern that ties this to the AI SRE narrative
Each new analyzer emits diagnostics that the **LLM Investigator** (Layer 2)
can query AND that the **T1 FixProposer** can match against. We extend the
`DefaultFixerMatcher` keyword whitelist (or replace it with an LLM-classified
matcher; see C5 below) so the agent proposes drift-class-appropriate fixes:
re-sync an Argo Application, scale a StatefulSet, restore a Role, etc.

### Acceptance criteria
  - Each drift class ships with at least one `*_test.go` that exercises
    the analyzer against a synthetic snapshot.
  - Each class is flagged behind a Helm value (`analyzers.gitopsDrift.enabled`,
    etc.) so operators can opt in.
  - T1 FixProposer's `DefaultFixerMatcher` is extended to map at least
    one diagnostic per class to a whitelisted action_kind (or the LLM
    classifier from C5 covers it).
  - Live-cluster verification: each class produces at least one DriftReport
    on the dev cluster within 24h of being enabled.

---

## Workstream C — Language refactor (away from "rule-based")

### Goal
Strip the "rule-based / whitelist / deterministic" framing from public
artifacts. Replace with "policy-bounded autonomy / operator-defined
action policy / audit-anchored execution." The technology doesn't change;
the wrappers do.

### Audit table (places where the old language lives today)

  | File | Phrase to replace | Replacement |
  |------|------------------|-------------|
  | `agentic-sre/README.md` L17 | "Layer-2 rule-based Investigator (OSS)" | "Layer-2 LLM Investigator agent (6-tool action space)" |
  | `agentic-sre/docs/design/2026-05-investigator-agent.md` | "rule-based" mentions | "policy-bounded" / drop where redundant |
  | `agentic-sre/README.md` "Fixers (4 default + 1 opt-in)" line | "whitelist" framing | "operator-policy-bounded action library" |
  | `srenix-website/src/pages/index.astro` | "whitelisted fixers" | "policy-bounded fixers" |
  | `srenix-website/src/pages/pricing.astro` | "All 5 whitelisted fixers" | "All 5 policy-bounded fixers" |
  | `srenix-website/src/pages/features/index.astro` | "rule-based" mentions | drop / reframe |
  | `srenix-website/src/pages/security.astro` | "whitelist of safe actions" | "operator-defined action policy" |
  | All design docs | "deterministic" qualifier on the engine | drop (the engine IS deterministic; we just don't *lead* with that) |

### Work items

C1. **OSS docs sweep.** Replace per the table above in `agentic-sre`.
   PR onto main. **Target: 2026-06-03.** Effort: 0.5 day.

C2. **Website docs sweep.** Same table, but on `srenix-website`. Ship as part
   of the A1 homepage rewrite. **Target: 2026-06-03.** Effort: 0.25 day.

C3. **Srenix Enterprise docs sweep.** `Srenix Enterprise/CHANGELOG.md`, `docs/AI_TIERS.md`
   if any "rule-based" references. **Target: 2026-06-03.** Effort: 0.25 day.

C4. **Action-policy explainer page.** New `srenix-website/src/pages/features/policy.astro`
   that explains the safety envelope as *"the leash you put on the agent"* —
   covers operator-defined action_kinds, target-namespace allowlists,
   approval-threshold thresholds, dual-approval for T3.
   **Target: 2026-06-10.** Effort: 1 day.

C5. **(Stretch) Replace `DefaultFixerMatcher` keyword heuristic with an
   LLM-classified matcher.** The current keyword match (`error` + `pod` +
   `stuck/stale/failed` → `StaleErrorPods`) is genuinely rule-based and
   should become a small LLM classification call. The action_kind whitelist
   stays (that's the policy bound), but the *decision* of which kind to
   propose becomes an LLM call.
   **Target: v1.7 (2026-07-08).** Effort: 2 days.

### Acceptance criteria
  - `grep -i "rule-based\|whitelist" srenix-website/src agentic-sre/{README.md,docs/}`
    returns ≤ 3 hits (down from ~30 today).
  - New `/features/policy` page exists and is linked from pricing.
  - The README's investigator description leads with "LLM agent," not
    "rule-based."
  - C5 ships in v1.7 — the fixer-matching path is end-to-end LLM-classified
    inside a policy-bounded action surface.

---

## Cross-cutting — Demo, collateral, evidence-anchoring

D1. **Refresh the live-evidence claims on the homepage.** Yesterday's
   T0/T1/T2/T3 verification against Qwen 3.6 35B is screenshot-able evidence.
   Embed it as the proof-point on the homepage.
   **Target: 2026-06-03.** Effort: 0.25 day.

D2. **Public roadmap update.** `srenix-website/src/pages/roadmap.astro` —
   add v1.7 "drift-class expansion" block alongside the existing operator
   port + GCP/Azure cloud probes.
   **Target: 2026-06-03.** Effort: 0.25 day.

D3. **New blog post.** "From cluster health analyzer to AI SRE — why we
   re-framed Srenix." Walks through the feedback, the reframe, the safety
   story under the new name, and the v1.7 commitments.
   **Target: 2026-06-17.** Effort: 1 day.

---

## Sequencing & sprint allocation

  | Sprint window | Workstream items | Sprint goal |
  |---------------|------------------|-------------|
  | **2026-05-28 → 2026-06-03** | A1, A2, C1, C2, C3, D1, D2 | Reframe-on-disk: homepage + pricing + docs language refactor live |
  | **2026-06-04 → 2026-06-10** | A3, A4, C4 | Comparison page + demo + action-policy explainer |
  | **2026-06-11 → 2026-06-17** | A5, B1, D3 | First drift class shipped (GitOps drift); v1.7-alpha |
  | **2026-06-18 → 2026-07-01** | B2 | Second drift class (workload state) |
  | **2026-07-02 → 2026-07-08** | B3, C5 | Third drift class (RBAC) + LLM-classified fixer matcher |
  | **2026-07-09 → 2026-07-15** | v1.7 release stabilisation | v1.7 GA — three drift classes + LLM matcher + operator port |
  | **Q3 (2026-08 onward)** | B4, B5, B6 | v1.8 — config, capacity, security drift classes |

## Risks & dependencies

  - **Risk: B5 capacity drift depends on metrics-server.** Mitigation:
    ship metrics-server availability as a required pre-flight check; gracefully
    degrade if absent.
  - **Risk: C5 (LLM-classified fixer matcher) increases LLM-call rate at T1.**
    Mitigation: enforce the existing per-class rate budget; add a 1-call-per-fingerprint
    short-circuit so the same diagnostic doesn't re-call.
  - **Risk: Repositioning could confuse existing pilots.** Mitigation: keep
    the OSS "Srenix" name (the cluster runtime); only rename the paid tier
    public marketing to "AI SRE." Existing pilot binaries and Helm charts
    are unchanged.
  - **Dependency: comparison page (A3) needs current Komodor / Robusta
    feature lists.** Owner: 0.5 day of competitive research before drafting.

## Out of scope (deliberately)

  - **SaaS hosted offering.** The feedback explicitly said *not* to advertise
    a SaaS model. Stays in-cluster.
  - **Switching off the OSS deterministic fixers.** They remain; we just stop
    *leading* with them. The agent uses them under operator policy.
  - **Generic LLM-only "investigate anything" mode.** Out of scope until
    v2.0. The agent needs the structured probe surface from Workstream B to
    do anything useful; pure freeform LLM exploration is a separate research
    spike.

## Success metrics (90 days out)

  - At least one customer-facing artifact (homepage / pricing / demo) uses
    "AI SRE" or "Autonomous SRE" as the primary positioning.
  - Three new drift-class analyzers shipped on `main` and enabled in at
    least one customer's pilot.
  - Public `/compare` page exists with side-by-side data on three named
    competitors.
  - `rule-based` and `whitelist` no longer appear in the README's
    above-the-fold content.
  - At least one feedback round from the original reviewer confirms the
    reframe lands as intended.
