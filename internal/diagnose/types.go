// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

// Package diagnose contains read-only analyzers that produce precise,
// actionable diagnostic hints for failure patterns the auto-fixer cannot
// safely resolve on its own. Diagnostics are surfaced in the report's
// "Diagnostics" section; they never modify cluster state.
//
// The contract is intentionally distinct from probe: probes report
// component-level health, analyzers report cross-resource correlations
// (e.g. "this Secret is missing key X that this Deployment expects").
package diagnose

import (
	"context"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
)

// Diagnostic is a single human-readable hint with no auto-applicable action.
type Diagnostic struct {
	// Subject is the symbolic identity of the issue (used for de-duplication
	// across iterations). Example: "Secret/mcp/mcp-openproject-secrets/openproject-url"
	Subject string `json:"subject"`

	// Message is the rendered hint (one line, formatted for Slack mrkdwn).
	Message string `json:"message"`
}

// Analyzer is the contract every diagnostic analyzer implements.
//
// Run returns zero or more Diagnostics. It must not mutate cluster state
// and must tolerate any GVR being absent from the snapshot (e.g. CRD not
// installed) without erroring.
type Analyzer interface {
	Name() string
	Run(ctx context.Context, src snapshot.Source) []Diagnostic
}
