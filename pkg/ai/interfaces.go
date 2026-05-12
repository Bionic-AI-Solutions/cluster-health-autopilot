// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package ai

import (
	"context"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/diagnose"
)

// Enricher takes a Diagnostic and produces a narrative addendum.
//
// Contract:
//   - MUST NOT mutate cluster state.
//   - MUST NOT block the caller indefinitely; implementations should
//     honor ctx cancellation.
//   - SHOULD return (zero, nil) when the LLM call fails or the response
//     fails validation — deterministic diagnostics must continue to flow.
//   - Implementations live in the paid CHA-com binary. OSS users have
//     no Enricher registered; the watcher skips enrichment when none.
type Enricher interface {
	Name() string
	Enrich(ctx context.Context, d diagnose.Diagnostic) (EnrichedDiagnostic, error)
}

// FixProposer takes a Diagnostic and proposes a single AIProposedAction
// for human approval (T1) or the first step of a multi-step plan (T2).
//
// Contract:
//   - MUST NOT mutate cluster state.
//   - MUST set Proposal.ActionKind to a value satisfying ActionKind.IsValid().
//   - MUST set Proposal.Rollback (non-reversible actions are refused).
//   - MUST honor protected-namespace policy (returns nil for protected
//     namespace targets).
//   - SHOULD return (nil, nil) when the diagnostic has no matching
//     whitelisted fixer or the LLM produced no usable proposal.
//   - Implementations live in the paid CHA-com binary.
type FixProposer interface {
	Name() string
	Propose(ctx context.Context, d diagnose.Diagnostic) (*AIProposedAction, error)
}

// MultiStepPlanner extends FixProposer with the ability to emit a
// multi-action plan (T2). Each step in the plan independently passes
// AIProposedAction.Validate(). Steps are linked by PrerequisiteActionID.
type MultiStepPlanner interface {
	Name() string
	Plan(ctx context.Context, d diagnose.Diagnostic) ([]AIProposedAction, error)
}

// VaultRunbookProposer generates a Vault recovery runbook (T3). The
// runbook is human-executed under dual approval; CHA never executes
// Vault writes itself.
type VaultRunbookProposer interface {
	Name() string
	ProposeRunbook(ctx context.Context, d diagnose.Diagnostic) (*VaultRunbook, error)
}

// Signer produces a JWT token bearing the claims required for an
// approval click. Implementations hold the in-cluster Ed25519 signing
// key and run inside the approval-server sidecar.
type Signer interface {
	// Sign returns a compact JWT for the given proposal. The token
	// contains: action_id (as jti), expires_at, target reference,
	// requester identity, and tier.
	Sign(proposal AIProposedAction) (string, error)

	// SignRunbookApproval produces a token for a T3 runbook
	// acknowledgment click. The token carries the runbook_id and
	// approval slot (first vs second).
	SignRunbookApproval(runbook VaultRunbook, slot int) (string, error)
}

// Verifier validates an approval token and returns the carried claims.
// Implementations enforce signature, expiry, and one-time-use (jti
// recorded in Redis/etcd).
type Verifier interface {
	// VerifyAction parses a JWT, validates signature/expiry/jti, and
	// returns the embedded proposal plus approver identity. Errors:
	// ErrTokenInvalid, ErrTokenExpired, ErrTokenReplay.
	VerifyAction(ctx context.Context, token string) (*AIProposedAction, error)
}

// Approver coordinates the approval flow end-to-end: verifies the
// token, re-validates against admission policy, looks up the matching
// pre-mutation snapshot, and emits an ApprovedAction ready for the
// executor.
type Approver interface {
	// Approve takes a raw token from a click, verifies it, checks
	// admission policy, and returns ApprovedAction on success.
	Approve(ctx context.Context, token, approverID, sourceIP string) (*ApprovedAction, error)

	// ApproveT3 records one slot of a dual-approval flow. Returns
	// (nil, nil) if this is the first approval (waiting for second);
	// returns the DualApproval on the second click. Enforces distinct
	// approvers and MinT3Delay.
	ApproveT3(ctx context.Context, token, approverID, sourceIP string) (*DualApproval, error)
}

// AuditSink writes structured audit events for every AI-related
// operation. Implementations may target Kubernetes Events, Loki, an
// external SIEM, etc.
//
// Implementations MUST NOT block the caller indefinitely. A typical
// implementation buffers events and flushes asynchronously.
type AuditSink interface {
	Write(ctx context.Context, e AuditEvent) error
}

// AuditEvent is one record in the audit trail. Schema is intentionally
// broad so a single sink can carry events for LLM calls, proposals,
// approvals, and applied results.
type AuditEvent struct {
	// Type categorizes the event. Examples: "ai.llm.call",
	// "ai.proposal.created", "ai.proposal.validated", "ai.approval.granted",
	// "ai.approval.rejected", "ai.action.applied", "ai.action.failed",
	// "ai.runbook.dual_approval".
	Type string `json:"type"`

	// CorrelationID links events for the same proposal. Equals the
	// ActionID for action events; the RunbookID for runbook events;
	// a fresh UUID for standalone LLM call events.
	CorrelationID string `json:"correlation_id"`

	// Tier records the active AI tier at event time.
	Tier Tier `json:"tier"`

	// Actor identifies who initiated the event. For LLM calls and
	// proposals this is "cha-com"; for approvals it is the OIDC identity.
	Actor string `json:"actor"`

	// Subject is the Diagnostic.Subject the event relates to. Empty
	// for events that don't relate to a single diagnostic.
	Subject string `json:"subject,omitempty"`

	// Details is a free-form map for type-specific fields. Examples:
	//   ai.llm.call: {model, prompt_hash, prompt_tokens, completion_tokens, duration_ms}
	//   ai.proposal.created: {action_kind, target}
	//   ai.approval.granted: {source_ip}
	//   ai.action.applied: {success, post_apply_verified, diff_summary}
	Details map[string]any `json:"details,omitempty"`
}
