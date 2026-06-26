// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package ai

import (
	"net/url"
	"regexp"
	"strings"
	"time"
)

// ProtectedNamespaces names mirror internal/fix/protected.go. Re-listed
// here so the AI surface enforces the same boundary at proposal time
// without importing the internal package.
//
// This map is the COMPILED-IN FLOOR, not the complete protected set:
// operators may APPEND namespaces via SRENIX_PROTECTED_NAMESPACES_EXTRA
// (or SetExtraProtectedNamespaces) — see protected.go. Nothing can
// remove an entry from this floor at runtime.
var ProtectedNamespaces = map[string]struct{}{
	"kube-system":      {},
	"kube-public":      {},
	"kube-node-lease":  {},
	"rook-ceph":        {},
	"vault":            {},
	"external-secrets": {},
	"cnpg-system":      {},
	// CNI namespaces — a default-deny NetworkPolicy in these namespaces
	// can break node-to-node Calico dataplane traffic and the Tigera
	// operator reconciliation loop, causing cluster-wide connectivity
	// loss. Never propose policies here.
	"calico-system":   {},
	"tigera-operator": {},
	// Srenix's own namespace — never auto-act on ourselves. Deleting the
	// watcher/aiwatch/approval-server pods is self-disruption and a probe
	// blip on a standby replica must not trigger a proposal to delete it.
	"agentic-sre": {},
}

// IsProtectedNamespace reports whether ns is on the no-touch list —
// the compiled-in floor PLUS any operator-appended extras
// (SRENIX_PROTECTED_NAMESPACES_EXTRA / SetExtraProtectedNamespaces).
// Mirrors fix.IsProtectedNamespace; the floor is duplicated to avoid a
// cross-package dependency from the public ai package into
// internal/fix, while the extras are shared (internal/fix consults
// IsExtraProtectedNamespace) so both guards always agree.
func IsProtectedNamespace(ns string) bool {
	if ns == "" {
		return false
	}
	if _, ok := ProtectedNamespaces[ns]; ok {
		return true
	}
	return IsExtraProtectedNamespace(ns)
}

// Validate enforces the structural and policy invariants of an
// AIProposedAction. Called before any proposal is rendered or signed.
//
// Validation rules:
//   - ActionKind must be in the closed enum (ActionKind.IsValid)
//   - Target.Kind, Target.Namespace, Target.Name all non-empty
//   - Target.Namespace must NOT be a protected namespace
//   - Rollback.Description must be non-empty (no proposal without rollback)
//   - PatchPayload must be empty unless ActionKind is a patch verb
//     (ActionPatchDeployment or ActionPatchProbe); ActionPatchProbe must
//     carry a payload and target a Deployment/StatefulSet/DaemonSet
//   - CreatedAt and ExpiresAt must be set; ExpiresAt > CreatedAt
//   - Tier must be a valid AllowsProposals tier (T1/T2/T3)
//
// This is the CREATION/SIGN-time contract: it is the union of the shared
// structural invariants (validateStructural) PLUS the rollback-description
// requirement. The rollback check is a proposal-QUALITY gate — the LLM must
// supply a rollback plan, which is rendered to the approver in Slack/the
// ticket — and is intentionally NOT re-checked at execution time. See
// ValidateForExecution for the execution-time variant.
func (a *AIProposedAction) Validate() error {
	if err := a.validateStructural(); err != nil {
		return err
	}
	if strings.TrimSpace(a.Rollback.Description) == "" {
		return ErrMissingRollback
	}
	return nil
}

// ValidateForExecution enforces every safety and structural invariant of an
// AIProposedAction EXCEPT the rollback-description requirement. It is the
// correct check for executing an already-approved action that the executor
// has RECONSTRUCTED from a signed approval token.
//
// Why rollback is excluded: Rollback.Description is a creation-time QUALITY
// gate — it is enforced at proposal mint (Validate, called before the proposal
// is rendered or signed) and shown to the approver in the Slack message /
// ticket so they can make an informed decision. The signed approval token
// deliberately carries only the action's safety-relevant identity
// (action_id / tier / action_kind / target / diag_subject) and intentionally
// OMITS the rollback description. Re-running the full Validate() against a
// proposal reconstructed from that token would therefore always fail with
// ErrMissingRollback even though the proposal was a fully-valid, human-approved
// action — so execution must use this method instead.
//
// Everything else Validate() enforces is execution-relevant and IS still
// checked here: the action_kind closed enum, target presence/shape, protected-
// namespace boundary, patch-payload/kind pairing, manifest validity for
// ApplyManifest, pull-request URL shape for ProposePullRequest, the expiry
// window, and the proposal-tier check.
//
// Intended caller: the Srenix Enterprise approval-server executor (ai/approval/executor.go
// Execute), which validates the reconstructed proposal immediately before
// applying the mutation.
func (a *AIProposedAction) ValidateForExecution() error {
	return a.validateStructural()
}

// validateStructural runs every AIProposedAction invariant that is relevant
// both at creation/sign time AND at execution time. It deliberately EXCLUDES
// the rollback-description requirement (a creation-time quality gate); callers
// that need it (Validate) add it on top. See Validate and ValidateForExecution.
func (a *AIProposedAction) validateStructural() error {
	if !a.ActionKind.IsValid() {
		return ErrInvalidActionKind
	}
	if a.Target.Kind == "" || a.Target.Namespace == "" || a.Target.Name == "" {
		return ErrEmptyTarget
	}
	if IsProtectedNamespace(a.Target.Namespace) {
		return ErrProtectedNamespace
	}
	// PatchPayload is permitted only for the patch verbs:
	// ActionPatchDeployment (rollout-restart annotation) and
	// ActionPatchProbe (probe-timing scalars). The Srenix Enterprise validator
	// enforces the per-verb closed shape; here we only gate the pairing.
	if len(a.PatchPayload) > 0 &&
		a.ActionKind != ActionPatchDeployment &&
		a.ActionKind != ActionPatchProbe {
		return ErrInvalidActionKind
	}
	// ActionPatchProbe MUST carry a payload (the probe patch) and target a
	// pod-template-bearing workload. The detailed shape/bounds check lives
	// in the Srenix Enterprise approval validator; this is the structural floor.
	if a.ActionKind == ActionPatchProbe {
		if len(a.PatchPayload) == 0 {
			return ErrInvalidActionKind
		}
		switch a.Target.Kind {
		case "Deployment", "StatefulSet", "DaemonSet":
		default:
			return ErrInvalidActionKind
		}
	}
	// v1.15.0: ManifestYAML must be paired with ActionApplyManifest, and
	// when present must pass the safe-apply validator. Per the design,
	// the validator is the load-bearing check: it refuses any YAML that
	// would let an Approve click apply a dangerous mutation.
	if len(a.ManifestYAML) > 0 && a.ActionKind != ActionApplyManifest {
		return ErrInvalidActionKind
	}
	if a.ActionKind == ActionApplyManifest {
		if err := ValidateManifest(a.ManifestYAML); err != nil {
			return err
		}
	}
	// v1.17.0: ActionProposePullRequest carries a URL the
	// approval-server links the SRE to. The URL is opaque from Srenix's
	// point of view (we don't fetch it), but we MUST insist it's a
	// well-formed HTTPS URL — a malformed value would render as a
	// broken / phishing-shaped link in Slack.
	if len(a.PullRequestURL) > 0 && a.ActionKind != ActionProposePullRequest {
		return ErrInvalidActionKind
	}
	if a.ActionKind == ActionProposePullRequest {
		if err := validatePullRequestURL(a.PullRequestURL); err != nil {
			return err
		}
	}
	if a.CreatedAt.IsZero() || a.ExpiresAt.IsZero() || !a.ExpiresAt.After(a.CreatedAt) {
		return ErrTokenExpired
	}
	if !a.Tier.IsValid() || !a.Tier.AllowsProposals() {
		return ErrInvalidTier
	}
	return nil
}

// Validate enforces the structural invariants of an EnrichedDiagnostic.
// Called after LLM response is unmarshalled and before rendering.
//
// Returns ErrEnrichmentTooLong when the narrative exceeds MaxEnrichmentChars.
// Truncates RelatedSignals to MaxRelatedSignals (does not return an error
// for that; truncation is informational).
func (e *EnrichedDiagnostic) Validate() error {
	if len(e.Enrichment) > MaxEnrichmentChars {
		return ErrEnrichmentTooLong
	}
	if len(e.RelatedSignals) > MaxRelatedSignals {
		e.RelatedSignals = e.RelatedSignals[:MaxRelatedSignals]
	}
	return nil
}

// Secret-value heuristics. Any string in a VaultRunbook that matches
// these patterns is treated as a possible leaked secret and the runbook
// is rejected (forces the LLM to re-prompt without values).
//
// Patterns:
//   - Long base64 (≥40 chars from the base64 alphabet)
//   - Long hex (≥32 chars from the hex alphabet)
//   - HashiCorp Vault token prefix `hvs.`
//   - JWT-shape (three dot-separated base64 segments)
//   - GitHub PAT prefixes (`ghp_`, `gho_`, `ghs_`)
//   - AWS access key prefix (`AKIA`)
//   - Slack token prefixes (`xox[bpoa]-`)
var secretHeuristics = []*regexp.Regexp{
	regexp.MustCompile(`[A-Za-z0-9+/]{40,}={0,2}`),
	regexp.MustCompile(`\b[0-9a-fA-F]{32,}\b`),
	regexp.MustCompile(`\bhvs\.[A-Za-z0-9_-]{20,}\b`),
	regexp.MustCompile(`\beyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\b`),
	regexp.MustCompile(`\bghp_[A-Za-z0-9]{20,}\b`),
	regexp.MustCompile(`\bgho_[A-Za-z0-9]{20,}\b`),
	regexp.MustCompile(`\bghs_[A-Za-z0-9]{20,}\b`),
	regexp.MustCompile(`\bAKIA[A-Z0-9]{16,}\b`),
	regexp.MustCompile(`\bxox[bpoa]-[A-Za-z0-9-]{20,}\b`),
}

// ContainsSecretLike reports whether s appears to embed a secret value.
// Used by VaultRunbook.Validate to reject LLM outputs that smuggled
// concrete values past the system prompt instructions.
func ContainsSecretLike(s string) bool {
	for _, re := range secretHeuristics {
		if re.MatchString(s) {
			return true
		}
	}
	return false
}

// Validate enforces invariants on a VaultRunbook before rendering.
//
// Rules:
//   - VaultPath non-empty
//   - At least one KeyName
//   - CommandTemplate must include the ${VALUE substring (placeholder marker)
//   - NO field may pass ContainsSecretLike (LLM must NOT embed concrete values)
//   - CreatedAt and ExpiresAt must be set; ExpiresAt > CreatedAt
func (r *VaultRunbook) Validate() error {
	if strings.TrimSpace(r.VaultPath) == "" {
		return ErrInvalidVaultPath
	}
	if len(r.KeyNames) == 0 {
		return ErrMissingRollback // re-use; "incomplete runbook"
	}
	if !strings.Contains(r.CommandTemplate, "${VALUE") {
		return ErrSecretValueInOutput
	}
	// Aggregate every user-visible string and scan for embedded secrets.
	var sb strings.Builder
	sb.WriteString(r.CommandTemplate)
	sb.WriteString("\n")
	sb.WriteString(r.Rationale)
	sb.WriteString("\n")
	for _, ms := range r.ManualSteps {
		sb.WriteString(ms)
		sb.WriteString("\n")
	}
	for _, kn := range r.KeyNames {
		// Key NAMES are allowed; key values are not. We accept names but
		// still scan them to catch the LLM writing `password=hunter2`
		// where `password` is the name and `hunter2` snuck in.
		sb.WriteString(kn)
		sb.WriteString("\n")
	}
	if ContainsSecretLike(sb.String()) {
		return ErrSecretValueInOutput
	}
	if r.CreatedAt.IsZero() || r.ExpiresAt.IsZero() || !r.ExpiresAt.After(r.CreatedAt) {
		return ErrTokenExpired
	}
	return nil
}

// ValidateT3DualApproval enforces the dual-approval invariants for a
// T3 runbook. The two approvals must be from distinct identities and
// separated by at least MinT3Delay.
func ValidateT3DualApproval(d DualApproval) error {
	if d.First.Approver == "" || d.Second.Approver == "" {
		return ErrTokenInvalid
	}
	if d.First.Approver == d.Second.Approver {
		return ErrSameApprover
	}
	if d.Second.ApprovedAt.Sub(d.First.ApprovedAt) < MinT3Delay {
		return ErrT3DelayNotElapsed
	}
	return nil
}

// validatePullRequestURL enforces the shape rules for an
// ActionProposePullRequest URL. Defensive checks:
//   - non-empty
//   - parseable as a URL
//   - https scheme (never http; downgrades a JWT-signed message to
//     a man-in-the-middle target)
//   - host non-empty (rules out "https:///path" style malformed inputs)
//
// We DON'T enforce a forge host allowlist here — operators run
// self-hosted GitLab / Gitea / Forgejo instances with arbitrary
// hostnames. Allowlist enforcement (if needed) belongs in the
// approval-server's per-CR policy layer.
func validatePullRequestURL(s string) error {
	if strings.TrimSpace(s) == "" {
		return ErrPullRequestURLEmpty
	}
	u, err := url.Parse(s)
	if err != nil {
		return ErrPullRequestURLInvalid
	}
	if u.Scheme != "https" {
		return ErrPullRequestURLInvalid
	}
	if u.Host == "" {
		return ErrPullRequestURLInvalid
	}
	return nil
}

// DefaultProposalTTL is the standard expiry window for a T1/T2 proposal.
const DefaultProposalTTL = 15 * time.Minute

// DefaultRunbookTTL is the standard expiry window for a T3 runbook
// (allows 30 min mandatory delay + 60 min for second approver).
const DefaultRunbookTTL = 90 * time.Minute
