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
