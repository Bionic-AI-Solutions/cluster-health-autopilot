// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

// Package silence is the pure noise-suppression filter for CHA's watch
// loop. Given a slice of diagnostics + the active Silence CRs known to
// the cluster, it drops the diagnostics matched by an unexpired silence.
//
// The package is split into:
//   - filter.go (this file) — pure functions, no K8s deps. Pure +
//     trivially unit-testable.
//   - lister.go              — K8s-backed Lister using dynamic.Interface
//     so the watcher binary can fetch the live Silence set per cycle.
//
// Watcher integration sits in `internal/watcher`: it holds a Lister,
// calls List(ctx) once per cycle, hands the result + current diagnostics
// to Filter, and emits the survivors.
package silence

import (
	"strings"
	"time"

	chav1alpha1 "github.com/Bionic-AI-Solutions/cluster-health-autopilot/api/v1alpha1"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/diagnose"
)

// Matches reports whether the silence matches the diagnostic at time
// `now`. Match semantics:
//   - The silence must be ACTIVE: `spec.until` strictly in the future.
//   - Every NON-EMPTY matcher field must equal the diagnostic field.
//   - Empty matcher fields are wildcards.
//   - An entirely empty matcher (all fields blank) does NOT match —
//     defense-in-depth against a typo silencing everything. The CRD
//     validation also rejects this at admission.
func Matches(s chav1alpha1.Silence, d diagnose.Diagnostic, now time.Time) bool {
	// Expired silences never match — even if all matcher fields align.
	if !s.Spec.Until.After(now) {
		return false
	}

	m := s.Spec.Matcher
	// Empty matcher → no-op safety. The CRD validation should also
	// reject this, but defending the filter is cheap.
	if m.Source == "" && m.Subject == "" && m.Severity == "" && m.MessagePattern == "" {
		return false
	}

	if m.Source != "" && m.Source != d.Source {
		return false
	}
	if m.Subject != "" && m.Subject != d.Subject {
		return false
	}
	if m.Severity != "" && m.Severity != d.Severity {
		return false
	}
	// Phase 2.B.9 — class-scoped silence via MessagePattern substring.
	if m.MessagePattern != "" && !strings.Contains(d.Message, m.MessagePattern) {
		return false
	}
	return true
}

// Filter returns the subset of `diags` not suppressed by any active
// silence. Order is preserved. Diagnostics are scanned once; for each
// the silence list is walked at most once until a match short-circuits.
//
// O(diags × silences) — fine for the realistic sizes (hundreds of
// diags × tens of silences). If volumes grow, a Source-indexed map
// over silences is the obvious next step.
func Filter(diags []diagnose.Diagnostic, silences []chav1alpha1.Silence, now time.Time) []diagnose.Diagnostic {
	if len(silences) == 0 {
		return diags
	}
	out := diags[:0:0] // new backing; do not stomp the caller's slice
	for _, d := range diags {
		if !anyMatch(silences, d, now) {
			out = append(out, d)
		}
	}
	return out
}

// MatchesAny reports whether d is suppressed by any active silence.
// Useful for callers that check one finding at a time (e.g. filtering
// probe.Result findings, which have a different Go type from
// diagnose.Diagnostic and must be projected by the caller).
func MatchesAny(silences []chav1alpha1.Silence, d diagnose.Diagnostic, now time.Time) bool {
	return anyMatch(silences, d, now)
}

func anyMatch(silences []chav1alpha1.Silence, d diagnose.Diagnostic, now time.Time) bool {
	for i := range silences {
		if Matches(silences[i], d, now) {
			return true
		}
	}
	return false
}

// CountMatches reports how many diagnostics would be matched by each
// silence (keyed by namespace/name). Useful for status reporting and
// surfacing "active silences with N suppressions this cycle". The
// returned map carries entries only for silences that matched at least
// once.
func CountMatches(diags []diagnose.Diagnostic, silences []chav1alpha1.Silence, now time.Time) map[string]int {
	counts := map[string]int{}
	if len(silences) == 0 || len(diags) == 0 {
		return counts
	}
	for i := range silences {
		s := silences[i]
		key := s.Namespace + "/" + s.Name
		for _, d := range diags {
			if Matches(s, d, now) {
				counts[key]++
			}
		}
	}
	return counts
}
