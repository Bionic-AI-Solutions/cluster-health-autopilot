// Copyright 2026 Agentic SRE contributors
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

import pkgdiagnose "github.com/srenix-ai/agentic-sre/pkg/diagnose"

// Diagnostic is re-exported from pkg/diagnose; see that package for the
// canonical definition.
type Diagnostic = pkgdiagnose.Diagnostic

// Analyzer is re-exported from pkg/diagnose; see that package for the
// canonical definition.
type Analyzer = pkgdiagnose.Analyzer
