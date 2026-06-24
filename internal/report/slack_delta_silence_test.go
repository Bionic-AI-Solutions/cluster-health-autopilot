// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package report

import (
	"strings"
	"testing"
	"time"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/fix"
)

// TestDeltaSilence_ClickLinks_WhenMinted asserts that the LIVE critical
// render (SplitCriticalPayloads → renderSilenceSnippet) AND FormatSlackDelta
// both emit the two signed one-click silence links — subject (short) and
// class (long) — with their correct, configurable durations, and DROP the
// kubectl heredoc fallback when the links are present.
func TestDeltaSilence_ClickLinks_WhenMinted(t *testing.T) {
	d := DeltaDiag{
		Subject:             "Pod/prod/api-7f9",
		Source:              "FailingExternalSecrets",
		Severity:            "critical",
		Message:             "ExternalSecret prod/api stuck SecretSyncError",
		IsNewThisCycle:      false, // silence only renders for recurring findings (not brand-new this cycle)
		SilenceSubjectURL:   "https://approve.example.com/silence?token=SUBJ",
		SilenceClassLongURL: "https://approve.example.com/silence?token=CLASS",
		SilenceShortDur:     24 * time.Hour,
		SilenceLongDur:      2160 * time.Hour, // 90d
	}

	wantLinks := []string{
		"🔕 <https://approve.example.com/silence?token=SUBJ|Silence 24h>",
		"🔕 <https://approve.example.com/silence?token=CLASS|Silence class (90d)>",
	}

	// (1) LIVE path.
	for _, p := range SplitCriticalPayloads([]DeltaDiag{d}, nil) {
		body := p.Attachments[0].Text
		for _, w := range wantLinks {
			if !strings.Contains(body, w) {
				t.Errorf("SplitCriticalPayloads missing %q\n--- body ---\n%s", w, body)
			}
		}
		if strings.Contains(body, "kubectl apply") {
			t.Errorf("kubectl fallback must NOT render when click-links present\n--- body ---\n%s", body)
		}
	}

	// (2) FormatSlackDelta path.
	out := FormatSlackDelta([]DeltaDiag{d}, nil, []fix.Result{}, false)
	body := out.Attachments[0].Text
	for _, w := range wantLinks {
		if !strings.Contains(body, w) {
			t.Errorf("FormatSlackDelta missing %q\n--- body ---\n%s", w, body)
		}
	}
	if strings.Contains(body, "kubectl apply") {
		t.Errorf("FormatSlackDelta kubectl fallback must NOT render when click-links present\n--- body ---\n%s", body)
	}
}

// TestDeltaSilence_KubectlFallback_WhenUnconfigured asserts the air-gapped
// affordance survives: with no minted links, both renderers fall back to
// the 24h kubectl heredoc and emit NO click-links.
func TestDeltaSilence_KubectlFallback_WhenUnconfigured(t *testing.T) {
	d := DeltaDiag{
		Subject:        "Pod/prod/api-7f9",
		Source:         "FailingExternalSecrets",
		Severity:       "critical",
		Message:        "ExternalSecret prod/api stuck SecretSyncError",
		IsNewThisCycle: false, // silence only renders for recurring findings (not brand-new this cycle)
	}

	for name, body := range map[string]string{
		"SplitCriticalPayloads": SplitCriticalPayloads([]DeltaDiag{d}, nil)[0].Attachments[0].Text,
		"FormatSlackDelta":      FormatSlackDelta([]DeltaDiag{d}, nil, []fix.Result{}, false).Attachments[0].Text,
	} {
		if !strings.Contains(body, "kubectl apply") {
			t.Errorf("%s: kubectl fallback missing when unconfigured\n--- body ---\n%s", name, body)
		}
		if strings.Contains(body, "/silence?token=") || strings.Contains(body, "|Silence 24h>") {
			t.Errorf("%s: unconfigured render must NOT emit click-links\n--- body ---\n%s", name, body)
		}
	}
}

// TestDeltaSilence_DurationLabelsConfigurable asserts the labels track the
// configured windows (not hardcoded 24h/90d).
func TestDeltaSilence_DurationLabelsConfigurable(t *testing.T) {
	d := DeltaDiag{
		Subject:             "Deployment/prod/web",
		Source:              "ImageDigestPin",
		Severity:            "warning",
		Message:             "no digest pin",
		IsNewThisCycle:      false, // silence only renders for recurring findings (not brand-new this cycle)
		SilenceSubjectURL:   "https://x/silence?token=S",
		SilenceClassLongURL: "https://x/silence?token=C",
		SilenceShortDur:     12 * time.Hour,
		SilenceLongDur:      30 * 24 * time.Hour, // 30d
	}
	body := SplitCriticalPayloads([]DeltaDiag{d}, nil)[0].Attachments[0].Text
	for _, w := range []string{"|Silence 12h>", "|Silence class (30d)>"} {
		if !strings.Contains(body, w) {
			t.Errorf("duration label missing %q\n--- body ---\n%s", w, body)
		}
	}
}
