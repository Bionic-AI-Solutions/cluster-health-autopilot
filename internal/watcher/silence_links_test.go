// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package watcher

import (
	"strings"
	"testing"
	"time"

	"github.com/srenix-ai/agentic-sre/internal/report"
	pkgai "github.com/srenix-ai/agentic-sre/pkg/ai"
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
				BaseURL:    "https://srenix-approve.example.com",
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

// TestAttachSilenceLinks_WarningAdvisoryFinding verifies that warning/advisory
// findings (e.g. RBAC drift with no ApprovalURL) DO get silence links when the
// SilenceLinks config is populated. This is the regression guard for the
// production report that advisory findings showed only a kubectl snippet instead
// of the one-click Silence button.
//
// Root-cause investigation: the attachSilenceLinks path has NO severity gate and
// NO early-return for advisory findings. The kubectl fallback in renderSilenceSnippet
// only fires when SilenceSubjectURL is empty. If advisory findings show the kubectl
// snippet in production, the root cause is that SilenceLinks is NOT configured in
// the live deployment (no signing key + approval-server URL), not a code defect.
// This test confirms the code-level path is correct for any severity.
func TestAttachSilenceLinks_WarningAdvisoryFinding_GetsLinks(t *testing.T) {
	pub, priv, err := pkgai.GenerateSigningKey()
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}
	fixed := time.Now().UTC()
	w := &Watcher{
		now: func() time.Time { return fixed },
		cfg: Config{
			SilenceLinks: report.SilenceLinkConfig{
				PrivateKey: priv,
				BaseURL:    "https://srenix-approve.example.com",
				ShortDur:   24 * time.Hour,
				LongDur:    90 * 24 * time.Hour,
			},
		},
	}

	// Warning-severity advisory finding — no ApprovalURL (no Approve/Deny button).
	// Mirrors what RBAC drift produces: "ClusterRole X grants wildcard verb".
	e := &seenEntry{
		subject:  "ClusterRole/cluster-admin-wildcard",
		source:   "RBACDrift",
		severity: "warning",
		message:  "ClusterRole grants wildcard verbs on all resources",
	}
	state := map[string]*seenEntry{e.subject: e}
	w.attachApprovalURLs(state)

	if e.silenceSubjectURL == "" {
		t.Errorf("warning/advisory finding must get a subject-scoped silence link; silenceSubjectURL is empty")
	}
	if e.silenceClassLongURL == "" {
		t.Errorf("warning/advisory finding must get a class-scoped silence link; silenceClassLongURL is empty")
	}

	// Verify the subject link points at this specific subject.
	subjClaims := verifySilenceURL(t, pub, e.silenceSubjectURL)
	if subjClaims.Subject != "ClusterRole/cluster-admin-wildcard" {
		t.Errorf("subject link matcher.subject = %q; want exact finding subject", subjClaims.Subject)
	}
	if subjClaims.Source != "RBACDrift" {
		t.Errorf("subject link matcher.source = %q; want 'RBACDrift'", subjClaims.Source)
	}

	// The DeltaDiag projection must carry the links through so the renderer
	// can emit the click-link rather than the kubectl fallback.
	d := seenEntryToDeltaDiag(e)
	if d.SilenceSubjectURL == "" || d.SilenceClassLongURL == "" {
		t.Errorf("DeltaDiag must carry silence URLs for advisory finding: subj=%q class=%q",
			d.SilenceSubjectURL, d.SilenceClassLongURL)
	}
}

// TestAttachSilenceLinks_EmptySource_SubjectFallback verifies that a finding
// with an empty Source (legacy probe findings without an explicit source field)
// still gets silence links — the subject is used as the source fallback so
// MintSilenceLinks can proceed.
func TestAttachSilenceLinks_EmptySource_SubjectFallback(t *testing.T) {
	_, priv, err := pkgai.GenerateSigningKey()
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}
	fixed := time.Now().UTC()
	w := &Watcher{
		now: func() time.Time { return fixed },
		cfg: Config{
			SilenceLinks: report.SilenceLinkConfig{
				PrivateKey: priv,
				BaseURL:    "https://srenix-approve.example.com",
				ShortDur:   24 * time.Hour,
				LongDur:    90 * 24 * time.Hour,
			},
		},
	}

	// Finding with empty Source (e.g. legacy probe or analyzer that doesn't set Source).
	e := &seenEntry{
		subject:  "ServiceAccount/prod/orphan-sa",
		source:   "", // intentionally empty
		severity: "warning",
		message:  "ServiceAccount has no RoleBinding",
	}
	state := map[string]*seenEntry{e.subject: e}
	w.attachApprovalURLs(state)

	if e.silenceSubjectURL == "" {
		t.Errorf("empty-source finding must fall back to subject for silence minting; silenceSubjectURL is empty")
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
