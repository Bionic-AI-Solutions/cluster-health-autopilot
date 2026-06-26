// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package anonymize

import (
	"strings"
	"testing"

	"github.com/srenix-ai/agentic-sre/internal/diagnose"
	"github.com/srenix-ai/agentic-sre/internal/probe"
)

func TestAnonSubject_KnownSafePassthrough(t *testing.T) {
	if got := anonSubject("vault-store-rbac-missing"); got != "vault-store-rbac-missing" {
		t.Errorf("got %q, want passthrough", got)
	}
}

func TestAnonSubject_MissingKey(t *testing.T) {
	// "missing-key/prod/my-secret/DB_URL" → category preserved, rest hashed.
	got := anonSubject("missing-key/prod/my-secret/DB_URL")
	if !strings.HasPrefix(got, "missing-key/") {
		t.Errorf("category not preserved: %q", got)
	}
	if strings.Contains(got, "prod") || strings.Contains(got, "my-secret") || strings.Contains(got, "DB_URL") {
		t.Errorf("PII not redacted: %q", got)
	}
	// Determinism: same input → same output.
	got2 := anonSubject("missing-key/prod/my-secret/DB_URL")
	if got != got2 {
		t.Errorf("not deterministic: %q != %q", got, got2)
	}
}

func TestAnonSubject_VaultPath(t *testing.T) {
	got := anonSubject("missing-vault-path/team/livekit-sip/credentials")
	if !strings.HasPrefix(got, "missing-vault-path/") {
		t.Errorf("category not preserved: %q", got)
	}
	if strings.Contains(got, "team") || strings.Contains(got, "livekit") {
		t.Errorf("vault path segment not redacted: %q", got)
	}
}

func TestAnonSubject_Determinism(t *testing.T) {
	// Two subjects with the same ns/name must hash the same way so
	// time-series comparisons remain coherent across daily runs.
	s1 := anonSubject("missing-key/prod/my-secret/KEY")
	s2 := anonSubject("missing-key/prod/my-secret/KEY")
	if s1 != s2 {
		t.Errorf("non-deterministic: %q != %q", s1, s2)
	}
	// Different inputs must NOT collide.
	s3 := anonSubject("missing-key/staging/my-secret/KEY")
	if s1 == s3 {
		t.Errorf("collision: staging and prod map to same subject")
	}
}

func TestAnonText_IPv4Redacted(t *testing.T) {
	got := anonText("kubelet at 192.168.1.10 returned error")
	if strings.Contains(got, "192.168") {
		t.Errorf("IPv4 not redacted: %q", got)
	}
	if !strings.Contains(got, "ip-") {
		t.Errorf("IPv4 placeholder missing: %q", got)
	}
}

func TestAnonText_HostnameRedacted(t *testing.T) {
	got := anonText("endpoint https://livekit.example.com unreachable")
	if strings.Contains(got, "livekit.example.com") {
		t.Errorf("hostname not redacted: %q", got)
	}
}

func TestAnonText_BacktickNsName(t *testing.T) {
	// Secret `prod/my-secret` → `HASH/HASH`
	got := anonText("Secret `prod/my-secret` is missing key `DB_URL`")
	if strings.Contains(got, "prod") || strings.Contains(got, "my-secret") {
		t.Errorf("ns/name in backticks not redacted: %q", got)
	}
}

func TestAnonText_KindNsName(t *testing.T) {
	got := anonText("referenced by ExternalSecret/production/my-eso-resource")
	if strings.Contains(got, "production") || strings.Contains(got, "my-eso-resource") {
		t.Errorf("kind/ns/name not redacted: %q", got)
	}
	if !strings.Contains(got, "ExternalSecret/") {
		t.Errorf("kind prefix should be preserved: %q", got)
	}
}

func TestAnonText_NoPIIPreserved(t *testing.T) {
	// Ceph-style messages have no customer PII — they should pass through mostly intact.
	msg := "HEALTH_WARN: 1 nearfull osd(s)"
	got := anonText(msg)
	if got != msg {
		t.Errorf("Ceph message unexpectedly modified: got %q", got)
	}
}

func TestAnonComponent_CategoryPreserved(t *testing.T) {
	for _, name := range []string{"Ceph Storage", "Cluster Nodes", "Storage Claims", "PostgreSQL", "Critical Services"} {
		if got := anonComponent(name); got != name {
			t.Errorf("category %q should be preserved, got %q", name, got)
		}
	}
}

func TestAnonComponent_ServiceHashed(t *testing.T) {
	got := anonComponent("Service: LiveKit SIP")
	if strings.Contains(got, "LiveKit") {
		t.Errorf("service name not hashed: %q", got)
	}
	if !strings.HasPrefix(got, "Service: svc-") {
		t.Errorf("expected Service: svc- prefix, got %q", got)
	}
}

func TestAnonComponent_NsNameHashed(t *testing.T) {
	got := anonComponent("PostgreSQL (cnpg/my-cluster)")
	if strings.Contains(got, "cnpg") || strings.Contains(got, "my-cluster") {
		t.Errorf("ns/name not hashed in component: %q", got)
	}
	if !strings.HasPrefix(got, "PostgreSQL (") {
		t.Errorf("prefix not preserved: %q", got)
	}
}

func TestAnonymize_FullRun(t *testing.T) {
	in := RunInput{
		Version: "0.4.0",
		Results: []probe.Result{
			{
				Component: probe.ComponentResult{
					Component: "PostgreSQL (cnpg/main-db)",
					Status:    "DEGRADED",
					Detail:    "replica at 10.0.0.5 not syncing",
				},
				Findings: []probe.Finding{
					{
						Component:   "PostgreSQL (cnpg/main-db)",
						Severity:    "warning",
						Message:     "Pod cnpg/main-db-1 not ready",
						Remediation: "kubectl rollout restart deployment/cnpg/main-db",
					},
				},
			},
		},
		Diagnostics: []diagnose.Diagnostic{
			{
				Subject: "missing-key/production/db-secret/DB_PASS",
				Message: "Secret `production/db-secret` exists but is missing key `DB_PASS` (referenced by Deployment/production/api-server).",
			},
		},
	}

	a := New()
	rec := a.Anonymize(in, "run-001", "2026-05-04T10:00:00Z")

	if rec.ChaVersion != "0.4.0" {
		t.Errorf("version not preserved: %q", rec.ChaVersion)
	}
	if rec.Summary.TotalComponents != 1 {
		t.Errorf("summary.totalComponents = %d, want 1", rec.Summary.TotalComponents)
	}
	if rec.Summary.DegradedCount != 1 {
		t.Errorf("summary.degradedCount = %d, want 1", rec.Summary.DegradedCount)
	}
	if rec.Summary.DiagnosticCount != 1 {
		t.Errorf("summary.diagnosticCount = %d, want 1", rec.Summary.DiagnosticCount)
	}

	// IP in detail should be redacted.
	if strings.Contains(rec.Results[0].Component.Detail, "10.0.0.5") {
		t.Errorf("IP not redacted in component detail")
	}

	// PII in diagnostic subject and message should be redacted.
	d := rec.Diagnostics[0]
	if strings.Contains(d.Subject, "production") || strings.Contains(d.Subject, "db-secret") || strings.Contains(d.Subject, "DB_PASS") {
		t.Errorf("PII in subject: %q", d.Subject)
	}
	if strings.Contains(d.Message, "production") || strings.Contains(d.Message, "db-secret") {
		t.Errorf("PII in message: %q", d.Message)
	}
}

func TestAnonymize_SchemaVersion(t *testing.T) {
	rec := New().Anonymize(RunInput{Version: "0.4.0"}, "r1", "")
	if rec.SchemaVersion != "1" {
		t.Errorf("schemaVersion = %q, want 1", rec.SchemaVersion)
	}
}
