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
//
// The canonical Analyzer interface and Diagnostic type live in pkg/diagnose.
// The aliases below keep all internal implementations compiling unchanged;
// they are identical types, so any implementation here satisfies the
// exported interface expected by pkg/registry.
package diagnose

import pkgdiagnose "github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/diagnose"

// Diagnostic and Analyzer are re-exported from pkg/diagnose.
type (
	Diagnostic = pkgdiagnose.Diagnostic
	Analyzer   = pkgdiagnose.Analyzer
)
