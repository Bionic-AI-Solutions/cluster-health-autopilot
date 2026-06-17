// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package watcher

import (
	"strings"
	"testing"
	"time"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/report"
	pkgai "github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/ai"
)

// TestAttachApprovalURLs_MintsSilenceLinks asserts the watcher mints BOTH
// one-click Silence links onto every posted finding when SilenceLinks is
// configured, that the subject link verifies to matcher.{source,subject}
// = real Subject (NOT Component) for the SHORT window, and the class link
// verifies to matcher.source = the finding's real Source for the LONG
// window. Tokens are signature-verified so we know the scoped claims are
// tamper-proof.
func TestAttachApprovalURLs_MintsSilenceLinks(t *testing.T) {
	pub, priv, err := pkgai.GenerateSigningKey()
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}
	// Use a near-now clock: VerifySilenceToken enforces token exp against
	// the real wall clock, so a far-past fixed clock would mint
	// already-expired links.
	fixed := time.Now().UTC()
	w := &Watcher{
		now: func() time.Time { return fixed },
		cfg: Config{
			SilenceLinks: report.SilenceLinkConfig{
				PrivateKey: priv,
				BaseURL:    "https://cha-approve.example.com",
				ShortDur:   24 * time.Hour,
				LongDur:    90 * 24 * time.Hour,
			},
		},
	}

	e := &seenEntry{
		subject:  "ExternalSecret/prod/api",
		source:   "FailingExternalSecrets",
		severity: "critical",
		message:  "stuck SecretSyncError",
	}
	state := map[string]*seenEntry{e.subject: e}
	w.attachApprovalURLs(state)

	if e.silenceSubjectURL == "" || e.silenceClassLongURL == "" {
		t.Fatalf("both silence links must be minted: subj=%q class=%q", e.silenceSubjectURL, e.silenceClassLongURL)
	}
	if e.silenceShortDur != 24*time.Hour || e.silenceLongDur != 90*24*time.Hour {
		t.Errorf("durations not carried: short=%v long=%v", e.silenceShortDur, e.silenceLongDur)
	}

	subjClaims := verifySilenceURL(t, pub, e.silenceSubjectURL)
	if subjClaims.Scope != pkgai.SilenceScopeSubject {
		t.Errorf("subject link scope = %q, want subject", subjClaims.Scope)
	}
	if subjClaims.Subject != "ExternalSecret/prod/api" {
		t.Errorf("subject matcher.subject = %q, want real Subject", subjClaims.Subject)
	}
	if subjClaims.Source != "FailingExternalSecrets" {
		t.Errorf("subject matcher.source = %q, want real Source (not Component)", subjClaims.Source)
	}

	classClaims := verifySilenceURL(t, pub, e.silenceClassLongURL)
	if classClaims.Scope != pkgai.SilenceScopeClass {
		t.Errorf("class link scope = %q, want class", classClaims.Scope)
	}
	if classClaims.Subject != "" {
		t.Errorf("class link must NOT carry a subject, got %q", classClaims.Subject)
	}
	if classClaims.Source != "FailingExternalSecrets" {
		t.Errorf("class matcher.source = %q, want real Source", classClaims.Source)
	}
	// Class window (until) is ~90d out from the fixed clock; subject ~24h.
	if classClaims.UntilUnix <= subjClaims.UntilUnix {
		t.Errorf("class until (%d) must be later than subject until (%d)", classClaims.UntilUnix, subjClaims.UntilUnix)
	}

	// DeltaDiag projection carries the links + durations through.
	d := seenEntryToDeltaDiag(e)
	if d.SilenceSubjectURL != e.silenceSubjectURL || d.SilenceClassLongURL != e.silenceClassLongURL {
		t.Errorf("DeltaDiag dropped silence URLs: %+v", d)
	}
	if d.Source != "FailingExternalSecrets" {
		t.Errorf("DeltaDiag dropped Source: %q", d.Source)
	}
}

// TestAttachApprovalURLs_NoLinksWhenUnconfigured: OSS-only / air-gapped
// (no signer) leaves the silence URLs empty so the renderer falls back to
// the kubectl heredoc.
func TestAttachApprovalURLs_NoLinksWhenUnconfigured(t *testing.T) {
	w := &Watcher{now: time.Now, cfg: Config{}} // no SilenceLinks
	e := &seenEntry{subject: "Pod/ns/x", source: "Probe", severity: "critical", message: "down"}
	w.attachApprovalURLs(map[string]*seenEntry{e.subject: e})
	if e.silenceSubjectURL != "" || e.silenceClassLongURL != "" {
		t.Errorf("unconfigured watcher must mint NO silence links: subj=%q class=%q", e.silenceSubjectURL, e.silenceClassLongURL)
	}
}

func verifySilenceURL(t *testing.T, pub []byte, raw string) *pkgai.SilenceTokenClaims {
	t.Helper()
	i := strings.Index(raw, "token=")
	if i < 0 {
		t.Fatalf("no token in URL %q", raw)
	}
	tok := raw[i+len("token="):]
	if amp := strings.IndexByte(tok, '&'); amp >= 0 {
		tok = tok[:amp]
	}
	claims, err := pkgai.VerifySilenceToken(pub, tok)
	if err != nil {
		t.Fatalf("verify silence token: %v", err)
	}
	return claims
}
