// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

// Package probe defines the Probe interface and associated types that form
// the exported API surface for the CHA pattern registry.
//
// External pattern catalogs (paid tier, community plugins) implement Probe
// and register their implementations via pkg/registry.
package probe

import (
	"context"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/snapshot"
)

// Severity describes the urgency of a Finding.
type Severity string

// Severity constants.
const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
)

// Finding is a single observation surfaced by a probe.
type Finding struct {
	Component   string   `json:"component"`
	Severity    Severity `json:"severity"`
	Message     string   `json:"message"`
	Remediation string   `json:"remediation,omitempty"`

	// Investigation is the Layer-2 investigator's summary addendum.
	// Populated by the watcher between probe execution and DriftReport
	// creation when a pkg/ai.Investigator is registered. Free-form text
	// bounded by pkg/ai.MaxInvestigationSummaryChars.
	Investigation string `json:"investigation,omitempty"`
}

// ComponentResult is the per-component status block in the Slack report.
type ComponentResult struct {
	Component string `json:"component"`
	// Status is one of: HEALTHY, DEGRADED, CRITICAL, PROBE_FAILED, SKIPPED.
	Status string `json:"status"`
	Detail string `json:"detail"`
}

// Result is the output of a single probe run.
type Result struct {
	Component ComponentResult `json:"component"`
	Findings  []Finding       `json:"findings,omitempty"`
}

// Probe is the contract every health probe must implement.
//
// A Probe is read-only: it must not mutate cluster state.
// It must tolerate any GVR being absent (CRD not installed) without error.
type Probe interface {
	Name() string
	Run(ctx context.Context, src snapshot.Source) Result
}
