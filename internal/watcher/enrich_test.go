// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package watcher

import (
	"context"
	"errors"
	"testing"
	"time"

	pkgai "github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/ai"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/diagnose"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/registry"
)

type stubEnricher struct {
	calls int
	resp  pkgai.EnrichedDiagnostic
	err   error
}

func (s *stubEnricher) Name() string { return "stub" }
func (s *stubEnricher) Enrich(_ context.Context, _ diagnose.Diagnostic) (pkgai.EnrichedDiagnostic, error) {
	s.calls++
	return s.resp, s.err
}

func TestEnrichDiagnostics_NoEnricher(t *testing.T) {
	reg := registry.New()
	w := &Watcher{reg: reg}
	in := []diagnose.Diagnostic{{Subject: "Pod/default/x"}}
	out := w.enrichDiagnostics(context.Background(), in)
	if len(out) != 1 || out[0].Enrichment != "" {
		t.Errorf("expected unchanged diagnostics when no enricher; got %+v", out)
	}
}

func TestEnrichDiagnostics_PopulatesField(t *testing.T) {
	reg := registry.New()
	stub := &stubEnricher{resp: pkgai.EnrichedDiagnostic{Enrichment: "narrative"}}
	reg.RegisterEnricher(stub)
	w := &Watcher{reg: reg}

	in := []diagnose.Diagnostic{
		{Subject: "Pod/default/a"},
		{Subject: "Pod/default/b"},
	}
	out := w.enrichDiagnostics(context.Background(), in)
	if stub.calls != 2 {
		t.Errorf("expected 2 enricher calls; got %d", stub.calls)
	}
	for i, d := range out {
		if d.Enrichment != "narrative" {
			t.Errorf("diagnostic %d not enriched: %+v", i, d)
		}
	}
}

func TestEnrichDiagnostics_PreservesOriginalFields(t *testing.T) {
	reg := registry.New()
	stub := &stubEnricher{resp: pkgai.EnrichedDiagnostic{Enrichment: "n"}}
	reg.RegisterEnricher(stub)
	w := &Watcher{reg: reg}

	in := []diagnose.Diagnostic{{
		Subject:     "Pod/default/x",
		Message:     "msg",
		Remediation: "rem",
		Severity:    "critical",
		Source:      "SomeAnalyzer",
	}}
	out := w.enrichDiagnostics(context.Background(), in)
	if out[0].Subject != "Pod/default/x" || out[0].Message != "msg" ||
		out[0].Remediation != "rem" || out[0].Severity != "critical" ||
		out[0].Source != "SomeAnalyzer" {
		t.Errorf("non-AI fields mutated: %+v", out[0])
	}
}

func TestEnrichDiagnostics_DoesNotMutateInput(t *testing.T) {
	reg := registry.New()
	stub := &stubEnricher{resp: pkgai.EnrichedDiagnostic{Enrichment: "n"}}
	reg.RegisterEnricher(stub)
	w := &Watcher{reg: reg}

	in := []diagnose.Diagnostic{{Subject: "Pod/default/x"}}
	out := w.enrichDiagnostics(context.Background(), in)
	if &in[0] == &out[0] {
		t.Error("input slice was modified in place; caller assumes immutability")
	}
	if in[0].Enrichment != "" {
		t.Errorf("input enrichment was mutated: %q", in[0].Enrichment)
	}
	if out[0].Enrichment != "n" {
		t.Errorf("output not enriched: %q", out[0].Enrichment)
	}
}

func TestEnrichDiagnostics_EnricherFailureIsSilent(t *testing.T) {
	reg := registry.New()
	stub := &stubEnricher{err: errors.New("network")}
	reg.RegisterEnricher(stub)
	w := &Watcher{reg: reg}

	in := []diagnose.Diagnostic{{Subject: "Pod/default/x", Message: "real msg"}}
	out := w.enrichDiagnostics(context.Background(), in)
	if len(out) != 1 || out[0].Message != "real msg" {
		t.Errorf("enricher failure broke deterministic flow: %+v", out)
	}
	if out[0].Enrichment != "" {
		t.Errorf("expected empty enrichment after failure; got %q", out[0].Enrichment)
	}
}

func TestEnrichDiagnostics_EmptyInput(t *testing.T) {
	reg := registry.New()
	stub := &stubEnricher{}
	reg.RegisterEnricher(stub)
	w := &Watcher{reg: reg}
	out := w.enrichDiagnostics(context.Background(), nil)
	if len(out) != 0 {
		t.Errorf("expected empty slice; got %+v", out)
	}
	if stub.calls != 0 {
		t.Errorf("expected no calls for empty input; got %d", stub.calls)
	}
}

func TestEnrichDiagnostics_ContextCancellation(t *testing.T) {
	reg := registry.New()
	stub := &stubEnricher{resp: pkgai.EnrichedDiagnostic{Enrichment: "n"}}
	reg.RegisterEnricher(stub)
	w := &Watcher{reg: reg}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	in := []diagnose.Diagnostic{{Subject: "Pod/default/x"}, {Subject: "Pod/default/y"}}
	out := w.enrichDiagnostics(ctx, in)
	// With cancelled context, enrichment may be skipped entirely; check
	// that deterministic flow continues regardless.
	if len(out) != 2 {
		t.Errorf("expected 2 diagnostics out; got %d", len(out))
	}
}

// P1.9(b) — pendingURLs must not grow unbounded. Entries were only
// evicted on lookup (approvalURLFor); a recorded-but-never-looked-up
// ActionID persisted for the whole process lifetime. recordApprovalURL
// now sweeps entries older than pendingURLTTL on each insert, using the
// injectable `now` clock seam.
func TestRecordApprovalURL_TTLEvictsOnInsert(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clk := base
	w := &Watcher{now: func() time.Time { return clk }}

	w.recordApprovalURL("old-action", "http://example/approve?token=a")
	if got := len(w.pendingURLs); got != 1 {
		t.Fatalf("after first insert len=%d, want 1", got)
	}

	// Advance the clock past the TTL, then insert a fresh entry. The
	// sweep on insert must drop the stale "old-action".
	clk = base.Add(pendingURLTTL + time.Minute)
	w.recordApprovalURL("new-action", "http://example/approve?token=b")

	if _, ok := w.pendingURLs["old-action"]; ok {
		t.Errorf("stale entry old-action survived TTL sweep")
	}
	if _, ok := w.pendingURLs["new-action"]; !ok {
		t.Errorf("fresh entry new-action missing after insert")
	}
	if got := len(w.pendingURLs); got != 1 {
		t.Errorf("after sweep len=%d, want 1 (only new-action)", got)
	}
}
