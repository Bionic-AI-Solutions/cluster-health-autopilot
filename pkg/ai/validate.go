// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package ai

import (
	"regexp"
	"strings"
	"time"
)

// ProtectedNamespaces names mirror internal/fix/protected.go. Re-listed
// here so the AI surface enforces the same boundary at proposal time
// without importing the internal package.
var ProtectedNamespaces = map[string]struct{}{
	"kube-system":      {},
	"kube-public":      {},
	"kube-node-lease":  {},
	"rook-ceph":        {},
	"vault":            {},
	"external-secrets": {},
	"cnpg-system":      {},
}

// IsProtectedNamespace reports whether ns is on the no-touch list.
// Mirrors fix.IsProtectedNamespace; duplicated to avoid cross-package
// dependency from the public ai package into internal/fix.
func IsProtectedNamespace(ns string) bool {
	if ns == "" {
		return false
	}
	_, ok := ProtectedNamespaces[ns]
	return ok
}

// Validate enforces the structural and policy invariants of an
// AIProposedAction. Called before any proposal is rendered or signed.
//
// Validation rules:
//   - ActionKind must be in the closed enum (ActionKind.IsValid)
//   - Target.Kind, Target.Namespace, Target.Name all non-empty
//   - Target.Namespace must NOT be a protected namespace
//   - Rollback.Description must be non-empty (no proposal without rollback)
//   - PatchPayload must be empty unless ActionKind == ActionPatchDeployment
//   - CreatedAt and ExpiresAt must be set; ExpiresAt > CreatedAt
//   - Tier must be a valid AllowsProposals tier (T1/T2/T3)
func (a *AIProposedAction) Validate() error {
	if !a.ActionKind.IsValid() {
		return ErrInvalidActionKind
	}
	if a.Target.Kind == "" || a.Target.Namespace == "" || a.Target.Name == "" {
		return ErrEmptyTarget
	}
	if IsProtectedNamespace(a.Target.Namespace) {
		return ErrProtectedNamespace
	}
	if strings.TrimSpace(a.Rollback.Description) == "" {
		return ErrMissingRollback
	}
	if len(a.PatchPayload) > 0 && a.ActionKind != ActionPatchDeployment {
		return ErrInvalidActionKind
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

// DefaultProposalTTL is the standard expiry window for a T1/T2 proposal.
const DefaultProposalTTL = 15 * time.Minute

// DefaultRunbookTTL is the standard expiry window for a T3 runbook
// (allows 30 min mandatory delay + 60 min for second approver).
const DefaultRunbookTTL = 90 * time.Minute
