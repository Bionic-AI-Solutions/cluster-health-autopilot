// Copyright 2026 Agentic SRE contributors
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

import pkgprobe "github.com/srenix-ai/agentic-sre/pkg/probe"

// Probe is re-exported from pkg/probe; see that package for the canonical definition.
type Probe = pkgprobe.Probe

// Finding is re-exported from pkg/probe; see that package for the canonical definition.
type Finding = pkgprobe.Finding

// ComponentResult is re-exported from pkg/probe; see that package for the canonical definition.
type ComponentResult = pkgprobe.ComponentResult

// Result is re-exported from pkg/probe; see that package for the canonical definition.
type Result = pkgprobe.Result

// Severity is re-exported from pkg/probe; see that package for the canonical definition.
type Severity = pkgprobe.Severity

// Severity constants re-exported from pkg/probe.
const (
	SeverityInfo     = pkgprobe.SeverityInfo
	SeverityWarning  = pkgprobe.SeverityWarning
	SeverityCritical = pkgprobe.SeverityCritical
)
