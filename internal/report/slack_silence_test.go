// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package report

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/srenix-ai/agentic-sre/internal/probe"
	"github.com/srenix-ai/agentic-sre/pkg/ai"
)

func criticalResults() []probe.Result {
	return []probe.Result{
		{
			Component: probe.ComponentResult{Component: "Ceph", Status: "CRITICAL", Detail: "OSD down"},
			Findings: []probe.Finding{
				{Component: "CephOSDDown", Severity: probe.SeverityCritical, Message: "OSD 3 is down"},
			},
		},
	}
}

// TestFormatSlack_NoSilenceLinks_WhenUnconfigured is the regression
// guard: pure-OSS / air-gapped mode (no signer, no base URL) must render
// ZERO silence links and stay byte-identical to FormatSlack.
func TestFormatSlack_NoSilenceLinks_WhenUnconfigured(t *testing.T) {
	results := criticalResults()

	// (1) Old entrypoint — never renders links.
	legacy := FormatSlack(results, nil, nil, false)
	if strings.Contains(legacy.Attachments[0].Text, "Silence") {
		t.Errorf("FormatSlack must never emit silence links:\n%s", legacy.Attachments[0].Text)
	}

	// (2) New entrypoint with nil config — identical output.
	withNil := FormatSlackWithSilence(results, nil, nil, false, nil)
	if withNil.Attachments[0].Text != legacy.Attachments[0].Text {
		t.Errorf("nil-config output diverged from FormatSlack:\nnil:\n%s\nlegacy:\n%s",
			withNil.Attachments[0].Text, legacy.Attachments[0].Text)
	}

	// (3) Config present but no key → still no links (graceful).
	cfg := &SilenceLinkConfig{BaseURL: "https://x.example"} // no PrivateKey
	noKey := FormatSlackWithSilence(results, nil, nil, false, cfg)
	if strings.Contains(noKey.Attachments[0].Text, "Silence") {
		t.Errorf("missing-key config must not emit links:\n%s", noKey.Attachments[0].Text)
	}

	// (4) Key present but no base URL → no links.
	_, priv, _ := ai.GenerateSigningKey()
	noURL := FormatSlackWithSilence(results, nil, nil, false, &SilenceLinkConfig{PrivateKey: priv})
	if strings.Contains(noURL.Attachments[0].Text, "Silence") {
		t.Errorf("missing-baseURL config must not emit links:\n%s", noURL.Attachments[0].Text)
	}
}

// TestFormatSlack_RendersSilenceLinks_WhenConfigured asserts both links
// appear under a critical finding with correct scope/until when a signer
// + base URL are set.
func TestFormatSlack_RendersSilenceLinks_WhenConfigured(t *testing.T) {
	pub, priv, _ := ai.GenerateSigningKey()
	results := criticalResults()
	cfg := &SilenceLinkConfig{
		PrivateKey: priv,
		KeyID:      "kid-1",
		BaseURL:    "https://srenix-approve.example.com",
		ShortDur:   24 * time.Hour,
		LongDur:    2160 * time.Hour,
	}
	now := time.Now()
	p := FormatSlackWithSilence(results, nil, nil, false, cfg)
	text := p.Attachments[0].Text

	if !strings.Contains(text, "Silence 24h") {
		t.Errorf("missing subject silence link:\n%s", text)
	}
	if !strings.Contains(text, "Silence class (90d)") {
		t.Errorf("missing class silence link:\n%s", text)
	}

	// Extract the two /silence?token= URLs and verify their claims.
	tokens := extractSilenceTokens(text)
	if len(tokens) != 2 {
		t.Fatalf("expected 2 silence tokens, got %d:\n%s", len(tokens), text)
	}
	var sawSubject, sawClass bool
	for _, tok := range tokens {
		claims, err := ai.VerifySilenceToken(pub, tok)
		if err != nil {
			t.Fatalf("token verify: %v", err)
		}
		switch claims.Scope {
		case ai.SilenceScopeSubject:
			sawSubject = true
			if claims.Source != "CephOSDDown" || claims.Subject != "CephOSDDown" {
				t.Errorf("subject token matcher wrong: source=%q subject=%q", claims.Source, claims.Subject)
			}
			if delta := claims.UntilUnix - now.Add(24*time.Hour).Unix(); delta < -5 || delta > 5 {
				t.Errorf("subject until off by %ds", delta)
			}
		case ai.SilenceScopeClass:
			sawClass = true
			if claims.Source != "CephOSDDown" || claims.Subject != "" {
				t.Errorf("class token matcher wrong: source=%q subject=%q", claims.Source, claims.Subject)
			}
			if delta := claims.UntilUnix - now.Add(2160*time.Hour).Unix(); delta < -5 || delta > 5 {
				t.Errorf("class until off by %ds", delta)
			}
		}
	}
	if !sawSubject || !sawClass {
		t.Errorf("expected one subject + one class token; subject=%v class=%v", sawSubject, sawClass)
	}
}

// extractSilenceTokens pulls every token= query param out of
// <url|label> Slack links pointing at /silence.
func extractSilenceTokens(text string) []string {
	var out []string
	for _, seg := range strings.Split(text, "<") {
		raw := seg
		if i := strings.Index(raw, "|"); i >= 0 {
			raw = raw[:i]
		}
		if !strings.Contains(raw, "/silence?") {
			continue
		}
		u, err := url.Parse(raw)
		if err != nil {
			continue
		}
		if tok := u.Query().Get("token"); tok != "" {
			out = append(out, tok)
		}
	}
	return out
}
