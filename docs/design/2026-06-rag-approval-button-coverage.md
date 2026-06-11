# Phase 2d-δ — Approval-Button Coverage for Analyzer-Proposed YAML

> **STATUS: ✅ SHIPPED — OSS URL-minting + class-action buttons landed.**
> _(P4.1 honest-header pass, 2026-06-11)_
>
> Shipped: the OSS watcher now mints approve/deny URLs itself (Phase 2d-δ-3 / Path B, PR #151), closing the root-cause gap this doc names (URLs were only minted in the cha-com aiwatch and never reached the OSS-written Slack/Alertmanager/OpenProject sinks). Approve/Deny + silence-snippet wired into the production critical path in v1.10.4 (PR #131). OSS Slack **class-action buttons** shipped Phase 2.B.6 (v1.21.0, commit d10cf12 / PR #174) — broadening button coverage beyond the original 5-value `ActionKind` whitelist to per-class actions. `ai.replicas` HA aiwatch + observability shipped alongside (v1.21.0).
>
> No material design-vs-shipped scope-shrink. Body below is the original design, preserved for context.

---

**Status:** Design draft
**Tier:** Paid (cha-com aiwatch + approval-server)
**Author:** opened 2026-06-01
**Companions:** [2026-06-rag-cluster-knowledge.md](2026-06-rag-cluster-knowledge.md), [2026-06-rag-networkpolicy-proposer.md](2026-06-rag-networkpolicy-proposer.md), [2026-06-rag-digest-pin-proposer.md](2026-06-rag-digest-pin-proposer.md)

## The gap

End-to-end test 2026-06-01:

  - User clicks: the Slack messages from CHA's `FormatCriticalPayload` are sent (we verified the rendering renders the buttons correctly)
  - BUT no buttons appear in production Slack messages for any of the 200+ findings on the dev cluster
  - **Root cause:** the proposal pipeline only mints approval URLs for findings that produce a valid `pkgai.AIProposedAction`, which has a closed-schema `ActionKind` whitelist of just 5 values:

    | ActionKind | What |
    |---|---|
    | `DeletePod` | Delete a stuck pod |
    | `DeleteJob` | Delete a stuck job |
    | `PatchDeployment` | Patch an annotation (e.g. force rolling restart) |
    | `DeleteCertRequest` | Force cert-manager retry |
    | `DeleteACMEOrder` | Force ACME order retry |

  - The `LLMFixerMatcher` classifier prompt is also restricted to "one of: StaleErrorPods, StuckJobsWithBadSecretRef, StuckRSPods, StuckCertificateRequests, none."
  - For everything else (digest-pin warnings, NetworkPolicy gaps, RBAC findings, HPA structural, DNS-chain, pod-security labels, SA automount): the LLM returns "none" → no proposal → no URL → no buttons.

OSS v1.12.0 added `Diagnostic.ProposedPolicyYAML` + `ProposedPolicyKind` fields so the OSS analyzers (e.g. `NetworkPolicyProposer`) could carry deterministic YAML to cha-com. **cha-com never wired the consumer side**. The bridge exists in OSS contract but not in code.

The result: SREs see "Human Action Required" Slack messages with the silence snippet but no Approve/Deny buttons. The product appears half-built, but it's actually that the action-kind schema is narrower than the diagnostic surface.

## Three candidate fixes

### Option A — Extend the action whitelist with `ApplyManifest`

Add a single new `ActionKind`:

```go
const (
    // ... existing 5 ...
    ActionApplyManifest ActionKind = "ApplyManifest"
)
```

Plus a new payload field:

```go
type AIProposedAction struct {
    // ... existing fields ...

    // ManifestYAML is set only when ActionKind == ActionApplyManifest.
    // The executor on the approval-server applies it via
    // `kubectl apply -f -`. The validator enforces a closed schema
    // on the manifest shape so the LLM (or OSS analyzer) cannot
    // smuggle arbitrary mutations:
    //   - kind: NetworkPolicy | RoleBinding | ConfigMap (initial set)
    //   - no service-account elevation (NO `cluster-admin`,
    //     no `*` verbs in Role/ClusterRole/Binding subjects)
    //   - no privileged: true, hostNetwork: true, hostPath volumes
    //     in PodSpecs (even though we don't propose Pods directly,
    //     guards against future Deployment/StatefulSet shapes)
    //   - no `metadata.namespace: kube-system / kube-public /
    //     kube-node-lease / vault / external-secrets`
    ManifestYAML []byte `json:"manifest_yaml,omitempty"`
}
```

Bridge in `cmd/cha-com/ai_wiring.go::proposeFixes`:

```go
// BEFORE running the LLM matcher, check if the OSS analyzer already proposed:
if d.ProposedPolicyYAML != "" {
    action := pkgai.AIProposedAction{
        ActionID:          generateActionID(d),
        Tier:              "t1",
        ActionKind:        pkgai.ActionApplyManifest,
        ManifestYAML:      []byte(d.ProposedPolicyYAML),
        Target:            parseTargetFromYAML(d.ProposedPolicyYAML),
        Rationale:         d.Remediation,  // OSS analyzer already wrote the why
        DiagnosticSubject: d.Subject,
        Rollback: pkgai.RollbackInfo{
            ActionKind:  pkgai.ActionDeleteManifest, // also new — symmetric undo
            Description: "kubectl delete -f manifest (matched by name+kind+namespace)",
        },
        // ... ExpiresAt etc.
    }
    if err := pkgai.Validate(action); err != nil {
        // Validator rejected. Skip; don't surface a bad proposal.
        continue
    }
    // Mint URL, set d.ApprovalURL, append to proposalRecords.
}
```

**Pros:**
- Single new ActionKind unlocks all current + future analyzer-YAML proposers (NetPol, RoleBinding fix, ConfigMap repair, …)
- Existing `pkg/ai.Signer` / approval-server flow reused verbatim
- Validator centralizes safety checks in one place

**Cons:**
- "kubectl apply" is broader than the existing narrow actions. Validator is the load-bearing safety net; bugs there → arbitrary mutation risk
- Initial allowed-kind list is small (NetworkPolicy + RoleBinding + ConfigMap); broadening is a security review per kind

**Effort:** ~1 sprint. 60% validator + tests, 30% bridge wiring, 10% docs.

---

### Option B — Parallel `ApprovalRequest` CR-based flow

Don't extend the closed schema; instead create a NEW CR type owned by cha-com:

```yaml
apiVersion: cha.bionicaisolutions.com/v1alpha1
kind: ApprovalRequest
metadata:
  name: netpol-app-allow-from-kong
spec:
  source: analyzer
  analyzerName: NetworkPolicyProposer
  diagnosticSubject: Namespace/cluster/app/missing-network-policy
  proposedManifestYAML: |
    apiVersion: networking.k8s.io/v1
    kind: NetworkPolicy
    ...
  rationale: "Namespace app on calico cluster has zero NetworkPolicies; default-deny + observed allows recommended"
  expiresAt: 2026-06-02T18:00:00Z
status:
  outcome: pending  # | approved | denied | applied | reverted
  approver: ""
  approvedAt: null
```

aiwatch creates ApprovalRequest CRs from `Diagnostic.ProposedPolicyYAML`, signs an URL pointing at the approval-server with the CR name as the payload, posts the link to Slack. approval-server's `/approve` endpoint, on click, reads the CR, calls `kubectl apply` on `spec.proposedManifestYAML`, sets `status.outcome=applied`.

**Pros:**
- Doesn't touch the narrow `AIProposedAction` schema (preserves the security posture of T1/T2 fixes)
- ApprovalRequest CR is auditable + replayable (kubectl-native)
- Sets a precedent for future "generic approval workflows" (operator-supplied actions, runbook proposals, …)

**Cons:**
- Two parallel paths for "approve a thing" — schema duplication
- More CRDs to install + RBAC to grant
- Closer to "approval-server reads a CR and trusts its YAML" — same safety burden as Option A, just relocated

**Effort:** ~1 sprint. Similar to A but with new CRD scaffolding (30%) + reconciler (30%) + bridge (40%).

---

### Option C — Per-class structured `ActionKind`s

Each finding class gets its own narrow ActionKind, with structured fields (no YAML blob):

```go
const (
    ActionLabelNamespace          ActionKind = "LabelNamespace"
    ActionSetSAAutomountFalse     ActionKind = "SetSAAutomountFalse"
    ActionCreateNetworkPolicy     ActionKind = "CreateNetworkPolicy"  // structured fields, not YAML
    ActionPinImageDigest          ActionKind = "PinImageDigest"
    // ... one per finding class
)

type AIProposedAction struct {
    // ... existing ...
    LabelNamespace        *LabelNamespacePayload   `json:"label_namespace,omitempty"`
    SetSAAutomountFalse   *SetSAAutomountPayload   `json:"set_sa_automount_false,omitempty"`
    CreateNetworkPolicy   *CreateNetPolPayload     `json:"create_network_policy,omitempty"`
}

type CreateNetPolPayload struct {
    Namespace     string                       `json:"namespace"`
    PolicyName    string                       `json:"policy_name"`
    PodSelector   map[string]string            `json:"pod_selector"`
    Ingress       []IngressRule                `json:"ingress"`
    Egress        []EgressRule                 `json:"egress,omitempty"`
}
```

**Pros:**
- Every ActionKind has its own closed schema — strongest safety posture
- Validator becomes per-kind, easy to audit
- The LLM (or OSS analyzer) cannot smuggle arbitrary YAML into any field

**Cons:**
- N new ActionKinds for N finding classes — labor-intensive
- Each kind needs structured marshaling + validator + executor branch + tests
- New finding classes added later require new ActionKinds (vs Option A: any new analyzer YAML "just works")

**Effort:** ~1 ActionKind per sprint. The first 3 (LabelNamespace, SetSAAutomountFalse, CreateNetworkPolicy) unlock the highest-value classes from today's audit; the rest can come over the next quarter.

---

## Recommendation

**Option A** for the speed-to-value, **gated behind robust validator coverage**.

The validator design is the load-bearing safety check. Initial allowed-kind list:

| Kind | Validator rules |
|---|---|
| `NetworkPolicy` | Egress on policyTypes is opt-in (default deny-ingress only); ipBlock CIDR validated; namespaceSelector matchExpressions must include `kubernetes.io/metadata.name` key (no broad-match accidents) |
| `RoleBinding` | roleRef.name MUST NOT be `cluster-admin` / `*-admin`; subjects must be ServiceAccounts only (no User / Group); roleRef.kind MUST be Role (not ClusterRole) for non-system namespaces |
| `ConfigMap` | data keys must be ≤ N (default 50); no keys matching `*\.key$`, `*\.token$`, `*\.password$` (catch accidental credential placement) |

Extending to additional kinds happens per-PR with security review.

## Phasing

| Phase | Scope | Gate |
|---|---|---|
| 2d-δ-1 | Add `ActionApplyManifest` + validator for kind=NetworkPolicy (the immediate gap) | Adversarial review of validator + bridge tests. Live cluster: real NetworkPolicy proposal emits real Approve link. |
| 2d-δ-2 | Validator extended to kind=RoleBinding (unblocks SA finding remediation) | Same gate, RoleBinding-specific. |
| 2d-δ-3 | Validator extended to kind=ConfigMap (digest-pin update via Helm-values style, or just Kustomize patch) | Same gate. |
| 2d-δ-4 | (Optional) Auto-apply pre-gated kinds for explicit operator allowlist (`spec.ai.autoApplyKinds: [NetworkPolicy]`) | UX review; operator must opt in per-kind. |

## Out of scope (initial cut)

- Deployment / StatefulSet manifest application (the "patch" path covers what we need for image bumps; full deploy replace is bigger surface than initial budget)
- CRD application (CRDs are install-time concerns; we don't want runtime mutations creating CRDs)
- Cluster-scoped resource mutations beyond NetworkPolicy (the namespace-scoped surface is large enough for v1)

## Adjacent decisions captured

- The OSS-side `Diagnostic.ProposedPolicyYAML` field (shipped in v1.12.0) is the right contract. The fix is the cha-com consumer, not the OSS producer.
- The narrow `AIProposedAction.ActionKind` whitelist was a deliberate security choice; Option A widens it but keeps the closed-schema discipline.
- Today's outage (2026-06-01) is the canary case for why YAML proposals need a validator: a permissive NetworkPolicy stub broke production. The validator's job is to refuse YAML that would break workloads.
