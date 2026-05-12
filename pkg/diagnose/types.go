// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

// Package diagnose defines the Analyzer interface and Diagnostic type that
// form the exported API surface for the CHA pattern registry.
//
// External pattern catalogs (paid tier, community plugins) implement Analyzer
// and register their implementations via pkg/registry. The only constraint:
// Run must be read-only — it must never mutate cluster state.
package diagnose

import (
	"context"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/snapshot"
)

// Diagnostic is a single human-readable hint with no auto-applicable action.
type Diagnostic struct {
	// Subject identifies the issue for deduplication across run iterations.
	// Convention: "Kind/namespace/name" or "Kind/namespace/name/key".
	Subject string `json:"subject"`

	// Message is the rendered hint (one line, Slack mrkdwn-formatted).
	Message string `json:"message"`

	// Remediation is an optional actionable fix instruction surfaced in Slack
	// below the message. Empty when no specific action can be recommended.
	Remediation string `json:"remediation,omitempty"`

	// Severity classifies the issue. Optional; analyzers that don't set it
	// receive the watcher's default classification. Used by AI proposers
	// to scope LLM context and by the validator to refuse mutations against
	// info-level diagnostics.
	Severity string `json:"severity,omitempty"`

	// Source names the analyzer that produced this diagnostic
	// (e.g. "FailingExternalSecrets", "IngressCoverage"). Optional;
	// fed to the AI prompt so the LLM knows which analyzer's context to
	// apply.
	Source string `json:"source,omitempty"`

	// AI-tier fields — all optional, all populated only by CHA-com.
	// OSS users never see these set.

	// Enrichment is the LLM-generated narrative addendum (T0+).
	// Free-form text bounded to 500 chars (enforced by pkg/ai validator).
	// Renderers surface this as a separate "🤖" block when present.
	Enrichment string `json:"enrichment,omitempty"`

	// Investigation is the Layer-2 investigator's summary addendum.
	// Populated by the OSS RuleBased investigator or any pkg/ai.Investigator
	// the registry exposes. Renderers surface this as a separate "🔎" block
	// when present, distinct from the LLM-generated Enrichment.
	Investigation string `json:"investigation,omitempty"`

	// ProposedActionID links this diagnostic to an AIProposedAction
	// (T1+). Renderers use the ID to look up the proposal and render the
	// Apply Fix button. Empty when no proposal is attached.
	ProposedActionID string `json:"proposed_action_id,omitempty"`

	// ProposedRunbookID links to a VaultRunbook (T3). Mutually exclusive
	// with ProposedActionID for a given diagnostic.
	ProposedRunbookID string `json:"proposed_runbook_id,omitempty"`
}

// Analyzer is the contract every diagnostic analyzer must implement.
//
// An Analyzer inspects a snapshot and returns zero or more Diagnostics.
// It must:
//   - Never mutate cluster state.
//   - Tolerate any GVR being absent (CRD not installed) without error.
//   - Return nil or an empty slice, never an error, when it has nothing to report.
type Analyzer interface {
	Name() string
	Run(ctx context.Context, src snapshot.Source) []Diagnostic
}
