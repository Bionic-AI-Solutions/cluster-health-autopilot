// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package report

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/srenix-ai/agentic-sre/internal/diagnose"
	"github.com/srenix-ai/agentic-sre/internal/fix"
	"github.com/srenix-ai/agentic-sre/internal/probe"
)

// TestFormatSlack_Healthy — clean cluster, all green, no findings.
func TestFormatSlack_Healthy(t *testing.T) {
	results := []probe.Result{
		{Component: probe.ComponentResult{Component: "Cluster Nodes", Status: "HEALTHY", Detail: "All 6 nodes ready"}},
		{Component: probe.ComponentResult{Component: "Ceph Storage", Status: "HEALTHY", Detail: "1 cluster"}},
	}
	p := FormatSlack(results, nil, nil, false)

	if got, want := len(p.Attachments), 1; got != want {
		t.Fatalf("attachments = %d, want %d", got, want)
	}
	a := p.Attachments[0]
	if a.Color != "good" {
		t.Errorf("color = %q, want good", a.Color)
	}
	if !strings.Contains(a.Text, "*Overall Status: HEALTHY*") {
		t.Errorf("text missing healthy banner:\n%s", a.Text)
	}
	if !strings.Contains(a.Text, "All systems operational") {
		t.Errorf("text missing healthy headline:\n%s", a.Text)
	}
	if strings.Contains(a.Text, "Critical Issues") {
		t.Errorf("text shouldn't contain Critical Issues section when none:\n%s", a.Text)
	}
	if strings.Contains(a.Text, "Diagnostics") {
		t.Errorf("text shouldn't contain Diagnostics section when none:\n%s", a.Text)
	}
	if !strings.Contains(a.Footer, "read-only") {
		t.Errorf("footer should report read-only when autopilot=false: %q", a.Footer)
	}
}

// TestFormatSlack_DegradedWithWarnings — some warnings present, no criticals.
func TestFormatSlack_DegradedWithWarnings(t *testing.T) {
	results := []probe.Result{{
		Component: probe.ComponentResult{Component: "Storage Claims", Status: "DEGRADED", Detail: "2 PVCs pending"},
		Findings: []probe.Finding{{
			Component: "Storage Claims",
			Severity:  probe.SeverityWarning,
			Message:   "2 PVC(s) in Pending state",
		}},
	}}
	p := FormatSlack(results, nil, nil, false)

	if p.Attachments[0].Color != "warning" {
		t.Errorf("color = %q, want warning", p.Attachments[0].Color)
	}
	if !strings.Contains(p.Attachments[0].Text, "*Overall Status: DEGRADED*") {
		t.Errorf("text missing degraded banner")
	}
	if !strings.Contains(p.Attachments[0].Text, "*🟡 Warnings (1):*") {
		t.Errorf("text missing warnings section:\n%s", p.Attachments[0].Text)
	}
}

// TestFormatSlack_CriticalWithFixes — fixers applied + criticals + diagnostics.
func TestFormatSlack_CriticalWithFixes(t *testing.T) {
	results := []probe.Result{{
		Component: probe.ComponentResult{Component: "Critical Services", Status: "CRITICAL", Detail: "1 service down"},
		Findings: []probe.Finding{{
			Component:   "Service: foo",
			Severity:    probe.SeverityCritical,
			Message:     "No ready pods (0/1)",
			Remediation: "Check: kubectl get pods -n foo",
		}},
	}}
	diagnostics := []diagnose.Diagnostic{{
		Subject: "Secret/foo/bar/X",
		Message: "Secret `foo/bar` missing key `X` (referenced by Deployment/foo).",
	}}
	fixResults := []fix.Result{{
		Fixer: "StaleErrorPods",
		Actions: []fix.Action{{
			Description: "Deleted stale Failed pod",
			Object:      "Pod/demo/zombie",
		}},
	}}

	p := FormatSlack(results, diagnostics, fixResults, true)
	a := p.Attachments[0]

	if a.Color != "danger" {
		t.Errorf("color = %q, want danger", a.Color)
	}
	for _, want := range []string{
		"*Overall Status: UNHEALTHY*",
		"*🔧 Automated Fixes Applied (1):*",
		"Deleted stale Failed pod",
		"`Pod/demo/zombie`",
		"*🔴 Critical Issues (1) — needs human:*",
		"No ready pods (0/1)",
		"*Diagnostics (1):*",
		"missing key `X`",
	} {
		if !strings.Contains(a.Text, want) {
			t.Errorf("text missing %q:\n%s", want, a.Text)
		}
	}
	if !strings.Contains(a.Footer, "auto-remediation: ON") {
		t.Errorf("footer should reflect autopilot mode: %q", a.Footer)
	}
}

// TestFormatSlack_PayloadIsValidJSON — Slack rejects malformed payloads.
func TestFormatSlack_PayloadIsValidJSON(t *testing.T) {
	p := FormatSlack(
		[]probe.Result{{Component: probe.ComponentResult{Component: "X", Status: "HEALTHY", Detail: "Y"}}},
		nil, nil, false,
	)
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	// Round-trip back to a generic map and check the attachment text is a string.
	var roundTrip map[string]any
	if err := json.Unmarshal(b, &roundTrip); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	atts, ok := roundTrip["attachments"].([]any)
	if !ok || len(atts) != 1 {
		t.Fatalf("attachments lost in round-trip: %+v", roundTrip)
	}
}

// TestPostSlack_Success — webhook returns "ok", no error.
func TestPostSlack_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.Header.Get("Content-Type"), "application/json"; got != want {
			t.Errorf("Content-Type = %q, want %q", got, want)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"username":"Cluster Health Monitor"`) {
			t.Errorf("body missing expected fields: %s", body)
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	p := FormatSlack(
		[]probe.Result{{Component: probe.ComponentResult{Component: "X", Status: "HEALTHY", Detail: "Y"}}},
		nil, nil, false,
	)
	if err := PostSlack(srv.Client(), srv.URL, p); err != nil {
		t.Errorf("PostSlack: %v", err)
	}
}

// TestPostSlack_Failure — non-200 response surfaces as error.
func TestPostSlack_Failure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "invalid payload", http.StatusBadRequest)
	}))
	defer srv.Close()

	p := FormatSlack(nil, nil, nil, false)
	err := PostSlack(srv.Client(), srv.URL, p)
	if err == nil {
		t.Fatal("expected error on 400 response")
	}
	if !strings.Contains(err.Error(), "HTTP 400") {
		t.Errorf("error doesn't mention status: %v", err)
	}
}

// TestPostSlack_RejectedPayload — Slack returns HTTP 200 with a non-"ok" body
// (e.g. "no_service", "channel_not_found"). Must surface as an error so the
// caller logs a warning rather than silently dropping the message.
func TestPostSlack_RejectedPayload(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("no_service"))
	}))
	defer srv.Close()

	p := FormatSlack(nil, nil, nil, false)
	err := PostSlack(srv.Client(), srv.URL, p)
	if err == nil {
		t.Fatal("expected error when Slack returns non-ok body with HTTP 200")
	}
	if !strings.Contains(err.Error(), "no_service") {
		t.Errorf("error should include Slack's response body: %v", err)
	}
}

// TestPostSlack_EmptyURL — empty webhook URL should error early, no network.
func TestPostSlack_EmptyURL(t *testing.T) {
	if err := PostSlack(nil, "", FormatSlack(nil, nil, nil, false)); err == nil {
		t.Errorf("expected error on empty webhook URL")
	}
}
