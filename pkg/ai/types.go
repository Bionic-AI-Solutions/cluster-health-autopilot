// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

// Package ai defines the interface surface that AI-enhanced CHA tiers
// plug into. The OSS engine ships these interfaces and no implementations —
// the paid CHA-com binary supplies the LLM-backed Enricher, FixProposer,
// Approver, Signer, and Verifier.
//
// SAFETY MODEL — every type in this package is designed so that AI output
// is a RECOMMENDATION, never an ACTION. The Mutator interface (pkg/fix) is
// never called directly from an AI response. Approval tokens are signed,
// expiring, one-time-use, and re-validated against an admission policy
// before mutation.
//
// See docs/AI_TIERS.md for the full capability/safety matrix.
package ai

import (
	"errors"
	"fmt"
	"time"
)

// Tier names the active AI capability level. Higher tiers do not imply
// higher agency — agency is always human-gated. Tiers differ in coverage
// (what AI can analyze and propose), not in autonomy.
type Tier string

const (
	// TierOff disables all AI behavior. Default for OSS users.
	TierOff Tier = "off"
	// TierT0 enables read-only narrative enrichment of diagnostics.
	// No mutation surface; no approval flow.
	TierT0 Tier = "t0"
	// TierT1 enables single-action fix proposals tied to existing
	// whitelisted fixers. Human one-click approval required.
	TierT1 Tier = "t1"
	// TierT2 enables multi-step plan proposals composed of T1 actions.
	// Step-by-step approval required.
	TierT2 Tier = "t2"
	// TierT3 enables Vault recovery runbook proposals. Dual-approval
	// required. CHA itself never writes to Vault in T3; runbooks are
	// executed manually by approvers.
	TierT3 Tier = "t3"
)

// IsValid returns true if t is a recognized tier value.
func (t Tier) IsValid() bool {
	switch t {
	case TierOff, TierT0, TierT1, TierT2, TierT3:
		return true
	}
	return false
}

// AllowsProposals reports whether this tier may surface AI proposals.
// T0 is enrichment only; T1/T2/T3 produce proposals.
func (t Tier) AllowsProposals() bool {
	return t == TierT1 || t == TierT2 || t == TierT3
}

// ActionKind is the closed enum of mutations an AI proposal may request.
// It is intentionally aligned with the existing OSS Fixer whitelist —
// the AI tier can never request a mutation outside this set.
type ActionKind string

// ActionKind values — the closed enum of mutations the AI tier may
// propose. Each entry corresponds to a specific snapshot.Mutator call
// dispatched by the approval-server's executor.
const (
	ActionDeletePod         ActionKind = "DeletePod"
	ActionDeleteJob         ActionKind = "DeleteJob"
	ActionPatchDeployment   ActionKind = "PatchDeployment"
	ActionDeleteCertRequest ActionKind = "DeleteCertRequest"
	ActionDeleteACMEOrder   ActionKind = "DeleteACMEOrder"

	// ActionApplyManifest is v1.15.0 (Phase 2d-δ). It accepts a
	// pre-rendered Kubernetes manifest in `ManifestYAML` and applies
	// it via `kubectl apply -f -`. The safe-apply validator enforces
	// a STRICT whitelist of allowed Kinds + per-Kind shape rules
	// (see pkg/ai/validate_manifest.go) so the LLM (or an OSS
	// analyzer) cannot smuggle arbitrary mutations.
	//
	// Initial allowed Kinds: NetworkPolicy. Extending to additional
	// Kinds requires a per-Kind security review + validator update.
	ActionApplyManifest ActionKind = "ApplyManifest"

	// ActionProposePullRequest is v1.17.0 (Phase 2d-γ). The proposer
	// has ALREADY opened a PR in a release-source repo (Helm chart,
	// Argo CD Application, Kustomize overlay) and embeds the PR URL
	// in `PullRequestURL`. The action carries no cluster mutation;
	// the cluster is only changed when the PR is merged + the next
	// normal deploy runs.
	//
	// Approve semantics (cha-com executor):
	//   - default: post an "SRE approved" comment on the PR; SRE
	//     completes the merge by hand
	//   - opt-in (per-CR): auto-merge the PR via the forge API
	// Deny semantics:
	//   - close the PR + record the outcome to RAG memory so the
	//     proposer doesn't re-propose the same workload until the
	//     running digest changes
	//
	// Used by the v1.17.0+ digest-pin proposer that consumes
	// `kind=workload` entries (pkg/feeder) + release-source
	// detection (pkg/releasesrc) to construct a one-line `:tag` →
	// `@sha256:<digest>` patch.
	ActionProposePullRequest ActionKind = "ProposePullRequest"
)

// IsValid reports whether ak is in the whitelist.
func (ak ActionKind) IsValid() bool {
	switch ak {
	case ActionDeletePod, ActionDeleteJob, ActionPatchDeployment,
		ActionDeleteCertRequest, ActionDeleteACMEOrder,
		ActionApplyManifest, ActionProposePullRequest:
		return true
	}
	return false
}

// ObjectRef identifies the Kubernetes object an action operates on.
type ObjectRef struct {
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

// String renders an ObjectRef as "Kind/namespace/name".
func (o ObjectRef) String() string {
	return fmt.Sprintf("%s/%s/%s", o.Kind, o.Namespace, o.Name)
}

// EnrichedDiagnostic is the T0 output: a narrative addendum to a
// deterministic Diagnostic. Free-form text bounded to 500 chars.
type EnrichedDiagnostic struct {
	// Enrichment is the LLM-generated 2-4 sentence root-cause narrative.
	// Maximum 500 characters; longer responses are truncated by the
	// paid binary's validator.
	Enrichment string `json:"enrichment"`

	// RelatedSignals lists optional follow-up paths the operator may
	// inspect (kubectl commands, dashboard URLs, log queries).
	// Maximum 5 entries; longer lists are truncated.
	RelatedSignals []string `json:"related_signals,omitempty"`
}

// MaxEnrichmentChars bounds enrichment text length. Enforced by the
// validator before AI output is rendered.
const MaxEnrichmentChars = 500

// MaxRelatedSignals bounds the related_signals list length.
const MaxRelatedSignals = 5

// AIProposedAction is a single proposal awaiting human approval.
// It is NEVER executed without a valid ApprovedAction returned from an
// Approver.Verify call.
//
// The "AI" prefix is intentional: it disambiguates from the existing
// fix.Action type (which records a mutation that has already been
// applied). External callers reference these side-by-side, so the
// extra qualifier earns its keep.
//
//revive:disable-next-line:exported
type AIProposedAction struct {
	// ActionID uniquely identifies this proposal. Used as the JWT `jti`
	// claim and as the audit log correlation key.
	ActionID string `json:"action_id"`

	// PlanID is set for T2 multi-step proposals; empty for single-action
	// T1 proposals.
	PlanID string `json:"plan_id,omitempty"`

	// StepN is set for T2 multi-step proposals (1-indexed); zero for T1.
	StepN int `json:"step_n,omitempty"`

	// PrerequisiteActionID identifies the predecessor step (T2 only).
	// An action with a non-empty PrerequisiteActionID may only execute
	// after that predecessor's post-apply verification succeeds.
	PrerequisiteActionID string `json:"prerequisite_action_id,omitempty"`

	// Tier records which tier produced this proposal.
	Tier Tier `json:"tier"`

	// ActionKind is the mutation requested. MUST be in the whitelist.
	ActionKind ActionKind `json:"action_kind"`

	// Target is the Kubernetes object the action operates on.
	Target ObjectRef `json:"target"`

	// PatchPayload is set only when ActionKind == ActionPatchDeployment.
	// It is the strategic-merge-patch JSON the executor will apply.
	// The validator enforces a closed schema on the patch shape so the
	// LLM cannot smuggle arbitrary patches (e.g. it can patch the
	// kubectl.kubernetes.io/restartedAt annotation but not the image
	// or env vars).
	PatchPayload []byte `json:"patch_payload,omitempty"`

	// ManifestYAML is set only when ActionKind == ActionApplyManifest
	// (Phase 2d-δ). It is a ready-to-apply Kubernetes manifest, parsed
	// + safety-checked by the validator before any approval URL is
	// minted. Initial allowed Kinds: NetworkPolicy. Each new Kind
	// requires a security-reviewed validator extension.
	//
	// Use cases:
	//   - OSS analyzer (NetworkPolicyProposer) produces a default-deny
	//     NetPol stub; aiwatch bridges it into an ApplyManifest action
	//     so the SRE can Approve/Deny in Slack rather than copy-paste
	//     kubectl.
	//   - Future analyzers that emit deterministic remediation YAML
	//     (RoleBinding-fix, ConfigMap repair) extend the allow list.
	ManifestYAML []byte `json:"manifest_yaml,omitempty"`

	// PullRequestURL is set only when ActionKind == ActionProposePullRequest
	// (Phase 2d-γ, v1.17.0+). It is the URL of the PR the proposer
	// already opened against the workload's release-source repo
	// (Helm chart, Argo CD Application, Kustomize overlay) —
	// typically a one-line `:tag` → `@sha256:<digest>` patch produced
	// by the digest-pin proposer.
	//
	// The validator requires this URL to use the https scheme and to
	// resolve to a well-formed PR/MR path so the approval-server can
	// safely link out from the Approve UI.
	//
	// The action carries NO cluster mutation: the cluster is only
	// changed when the PR is merged + the next normal deploy runs.
	// Approve = post a comment / auto-merge (per CR policy); Deny =
	// close the PR + record the outcome to RAG memory.
	PullRequestURL string `json:"pull_request_url,omitempty"`

	// Rationale is the LLM-generated explanation for the proposal.
	// Surfaced in the approval UI so the approver can decide.
	Rationale string `json:"rationale"`

	// Rollback describes the inverse action the operator can take to
	// undo this proposal. REQUIRED — the validator rejects any proposal
	// without a rollback. Non-reversible actions are refused at proposal
	// time.
	Rollback RollbackInfo `json:"rollback"`

	// DiagnosticSubject links this proposal to the source Diagnostic.
	// Format matches diagnose.Diagnostic.Subject.
	DiagnosticSubject string `json:"diagnostic_subject"`

	// CreatedAt is when the proposal was generated.
	CreatedAt time.Time `json:"created_at"`

	// ExpiresAt bounds the approval window. Default 15 minutes.
	ExpiresAt time.Time `json:"expires_at"`
}

// RollbackInfo describes how to undo an AIProposedAction.
type RollbackInfo struct {
	// Description is the human-readable rollback instruction.
	Description string `json:"description"`

	// ActionKind is the inverse mutation when one exists (e.g. revert
	// a Deployment patch). May be empty when rollback is purely manual
	// (e.g. "restore from etcd backup").
	ActionKind ActionKind `json:"action_kind,omitempty"`

	// Target is the rollback target. May differ from the original
	// action's Target.
	Target ObjectRef `json:"target,omitempty"`

	// SnapshotRef points to a pre-mutation snapshot held by the
	// approval-server for 24 hours. Empty when no snapshot is needed.
	SnapshotRef string `json:"snapshot_ref,omitempty"`
}

// ApprovedAction is the result of a successful Approver.Verify call.
// It carries the original proposal plus approver identity and timing.
// The executor uses ApprovedAction to apply the mutation via the
// existing Mutator interface.
type ApprovedAction struct {
	// Proposal is the original AIProposedAction the approver acted on.
	Proposal AIProposedAction `json:"proposal"`

	// Approver is the identity of the user who clicked Apply.
	// Sourced from OIDC (verified by the approval-server).
	Approver string `json:"approver"`

	// ApprovedAt is when the approval click was verified.
	ApprovedAt time.Time `json:"approved_at"`

	// SourceIP is the IP the approval click originated from. Logged
	// in audit; does not gate execution.
	SourceIP string `json:"source_ip,omitempty"`
}

// DualApproval records two distinct approvals for a T3 break-glass
// runbook. Both must be present, from distinct approvers, with the
// second occurring at least MinT3Delay after the first.
type DualApproval struct {
	First  ApprovedAction `json:"first"`
	Second ApprovedAction `json:"second"`
}

// MinT3Delay is the mandatory delay between the first and second
// approval for a T3 break-glass runbook. Allows the second approver
// to review.
const MinT3Delay = 30 * time.Minute

// VaultRunbook is the T3 output: a step-by-step recovery procedure for
// a Vault key/path issue. CHA-com NEVER executes Vault writes itself;
// the operator runs the runbook by hand after dual approval.
type VaultRunbook struct {
	// RunbookID uniquely identifies the runbook. Used as the JWT `jti`
	// for both approvals.
	RunbookID string `json:"runbook_id"`

	// VaultPath is the KV-v2 path the runbook targets (e.g.
	// "secret/t6-apps/billing/config"). Validated against the
	// operator-supplied policy allowlist before the runbook is emitted.
	VaultPath string `json:"vault_path"`

	// KeyNames lists the property names the operator must populate.
	// NEVER contains values — only names. Validated for shape (no
	// base64/hex/JWT-like strings that could be leaked secret bytes).
	KeyNames []string `json:"key_names"`

	// CommandTemplate is a `vault kv patch` command with ${VALUE_*}
	// placeholders. Validator rejects any concrete values.
	CommandTemplate string `json:"command_template"`

	// ManualSteps are additional procedural notes (e.g. "rotate the
	// dependent application's API key in its admin UI before patching
	// Vault").
	ManualSteps []string `json:"manual_steps,omitempty"`

	// Rationale explains why the runbook recommends these steps.
	Rationale string `json:"rationale"`

	// CreatedAt is when the runbook was generated.
	CreatedAt time.Time `json:"created_at"`

	// ExpiresAt bounds the dual-approval window. Default 90 minutes
	// (30 minutes minimum gap + 60 minutes for the second approver).
	ExpiresAt time.Time `json:"expires_at"`
}

// Validation errors. Callers check specific errors to distinguish
// "validator rejected" from "transport failure" etc.
var (
	ErrInvalidActionKind   = errors.New("ai: action_kind not in whitelist")
	ErrMissingRollback     = errors.New("ai: proposal lacks rollback info")
	ErrEmptyTarget         = errors.New("ai: target kind/namespace/name required")
	ErrProtectedNamespace  = errors.New("ai: target is in a protected namespace")
	ErrSecretValueInOutput = errors.New("ai: output appears to contain a secret value")
	ErrInvalidVaultPath    = errors.New("ai: vault path not in operator allowlist")
	ErrInvalidTier         = errors.New("ai: tier value not recognized")
	ErrTokenExpired        = errors.New("ai: approval token expired")
	ErrTokenReplay         = errors.New("ai: approval token already used")
	ErrTokenInvalid        = errors.New("ai: approval token signature invalid")
	ErrSameApprover        = errors.New("ai: T3 requires two distinct approvers")
	ErrT3DelayNotElapsed   = errors.New("ai: T3 second approval before 30-minute window")
	ErrEnrichmentTooLong   = errors.New("ai: enrichment exceeds maximum length")

	// v1.17.0 — ActionProposePullRequest validation.
	ErrPullRequestURLEmpty   = errors.New("ai: action ProposePullRequest requires pull_request_url")
	ErrPullRequestURLInvalid = errors.New("ai: pull_request_url must be a well-formed HTTPS URL")
)
