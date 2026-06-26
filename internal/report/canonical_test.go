// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package report

import (
	"strings"
	"testing"
)

func TestRecommendedAction_RootCauseFirst(t *testing.T) {
	got := RecommendedAction("container exited 0 — no subcommand", "kubectl logs --previous")
	if !strings.HasPrefix(got, "Root cause: ") {
		t.Errorf("should lead with root cause; got %q", got)
	}
	if !strings.Contains(got, "Next steps: kubectl logs --previous") {
		t.Errorf("should carry remediation as next steps; got %q", got)
	}
	if RecommendedAction("", "do X") != "do X" {
		t.Error("no investigation → bare remediation")
	}
	if RecommendedAction("cause Y", "") != "Root cause: cause Y" {
		t.Error("no remediation → root cause only")
	}
}

func TestRenderGuidance_RootCauseBeforeRemediation(t *testing.T) {
	var b strings.Builder
	renderGuidance(&b, "the cause", "the fix")
	out := b.String()
	ci := strings.Index(out, "the cause")
	ri := strings.Index(out, "the fix")
	if ci < 0 || ri < 0 || ci > ri {
		t.Errorf("root cause must render before remediation; got %q", out)
	}
	if !strings.Contains(out, "🔬") {
		t.Errorf("root cause should use the 🔬 marker; got %q", out)
	}
}

func TestResolved_CarriesRootCause(t *testing.T) {
	pl := FormatSlackDelta(nil,
		[]ResolvedDiag{{Subject: "Pod/ns/x", Message: "cleared", Investigation: "was OOMKilled"}},
		nil, false)
	body := ""
	if len(pl.Attachments) > 0 {
		body = pl.Attachments[0].Text
	}
	if !strings.Contains(body, "was: was OOMKilled") {
		t.Errorf("resolved message should carry the root cause; got %q", body)
	}
}
