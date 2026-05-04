// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

// Package probe contains read-only health probes that run against a snapshot.Source.
//
// Each probe is responsible for producing a ComponentResult plus zero or more
// Issues / Warnings that the loop-runner aggregates into a final report.
//
// The canonical Probe interface and result types live in pkg/probe.
// The aliases below keep all internal implementations compiling unchanged;
// they are identical types, so any implementation here satisfies the
// exported interface expected by pkg/registry.
package probe

import pkgprobe "github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/probe"

// Probe, Finding, ComponentResult, Result, and Severity are re-exported
// from pkg/probe.
type (
	Probe           = pkgprobe.Probe
	Finding         = pkgprobe.Finding
	ComponentResult = pkgprobe.ComponentResult
	Result          = pkgprobe.Result
	Severity        = pkgprobe.Severity
)

// Severity constants re-exported from pkg/probe.
const (
	SeverityInfo     = pkgprobe.SeverityInfo
	SeverityWarning  = pkgprobe.SeverityWarning
	SeverityCritical = pkgprobe.SeverityCritical
)
