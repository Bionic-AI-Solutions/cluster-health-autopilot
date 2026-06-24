// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package watcher

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/investigator"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/diagnose"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/probe"
)

// investigationTimeout is the hard ceiling for the whole investigation
// pass across all critical findings/diagnostics in one cycle. Individual
// investigations carry their own per-call deadlines via Investigator
// implementations.
const investigationTimeout = 20 * time.Second

// investigateDiagnostics runs the Layer-2 investigator over each Diagnostic
// in the slice and returns a copy with d.Investigation populated for each
// one the investigator could classify. No-op when no investigator is
// registered. Soft-fails on any per-call error — the original Diagnostic
// flows through unchanged.
func (w *Watcher) investigateDiagnostics(ctx context.Context, diagnostics []diagnose.Diagnostic) []diagnose.Diagnostic {
	inv := w.reg.Investigator()
	if inv == nil || len(diagnostics) == 0 {
		return diagnostics
	}
	cycleCtx, cancel := context.WithTimeout(ctx, investigationTimeout)
	defer cancel()
	env := investigator.NewLiveEnvironmentWithLogs(w.lv, w.cfg.KubeClientset)

	out := make([]diagnose.Diagnostic, len(diagnostics))
	copy(out, diagnostics)
	for i := range out {
		if cycleCtx.Err() != nil {
			return out
		}
		// Skip info-level observations — they aren't worth a root-cause pass.
		if strings.EqualFold(out[i].Severity, "info") {
			continue
		}
		res, err := inv.InvestigateDiagnostic(cycleCtx, out[i], env)
		if err != nil {
			log.Printf("watcher: investigate diagnostic %s: %v", out[i].Subject, err)
			continue
		}
		if res.Summary == "" {
			continue
		}
		out[i].Investigation = res.Summary
	}
	return out
}

// investigateProbeResults walks the probe Results and populates the
// Investigation field on every critical Finding. Mutates the result slice
// in place — Findings are value types so we have to reassign explicitly.
func (w *Watcher) investigateProbeResults(ctx context.Context, results []probe.Result) []probe.Result {
	inv := w.reg.Investigator()
	if inv == nil || len(results) == 0 {
		return results
	}
	cycleCtx, cancel := context.WithTimeout(ctx, investigationTimeout)
	defer cancel()
	env := investigator.NewLiveEnvironmentWithLogs(w.lv, w.cfg.KubeClientset)

	for ri := range results {
		for fi := range results[ri].Findings {
			if cycleCtx.Err() != nil {
				return results
			}
			f := results[ri].Findings[fi]
			// Investigate warning + critical; info findings are pure
			// observations and aren't worth a root-cause pass.
			if f.Severity == probe.SeverityInfo {
				continue
			}
			res, err := inv.InvestigateFinding(cycleCtx, f, env)
			if err != nil {
				log.Printf("watcher: investigate finding %s: %v", f.Component, err)
				continue
			}
			if res.Summary == "" {
				continue
			}
			results[ri].Findings[fi].Investigation = res.Summary
		}
	}
	return results
}
