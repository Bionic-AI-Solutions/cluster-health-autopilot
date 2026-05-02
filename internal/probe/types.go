// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

// Package probe contains read-only health probes that run against a snapshot.Source.
//
// Each probe is responsible for producing a ComponentResult plus zero or more
// Issues / Warnings that the loop-runner aggregates into a final report.
package probe

import (
	"context"

	"github.com/salilkadam/cluster-health-autopilot/internal/snapshot"
)

// Severity describes a finding level.
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
)

// Finding is a single observation surfaced by a probe.
type Finding struct {
	Component   string   `json:"component"`             // "Ceph", "PostgreSQL", "Service: LiveKit SIP", ...
	Severity    Severity `json:"severity"`
	Message     string   `json:"message"`
	Remediation string   `json:"remediation,omitempty"` // Optional kubectl-ready hint
}

// ComponentResult is the per-component status block that ends up in the
// "Component Status" section of the Slack report.
type ComponentResult struct {
	Component string `json:"component"`
	Status    string `json:"status"` // "HEALTHY" / "DEGRADED" / "CRITICAL" / "PROBE_FAILED" / "SKIPPED"
	Detail    string `json:"detail"`
}

// Result is the output of a single probe.
type Result struct {
	Component ComponentResult `json:"component"`
	Findings  []Finding       `json:"findings,omitempty"`
}

// Probe is the contract every health probe implements.
type Probe interface {
	Name() string
	Run(ctx context.Context, src snapshot.Source) Result
}
