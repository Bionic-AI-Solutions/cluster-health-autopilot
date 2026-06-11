// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"html/template"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// TestRenderEscapesXSS proves the viewer is XSS-safe: a DriftReport whose
// message contains a <script> payload is HTML-escaped, never emitted raw.
func TestRenderEscapesXSS(t *testing.T) {
	tmpl := template.Must(template.New("page").Parse(pageTmpl))
	data := pageData{
		Total:     1,
		Refreshed: "now",
		Rows: []row{{
			Subject:  "Deployment/cha-playground/<img src=x>",
			Severity: "critical",
			Source:   "CrashLoopBackOff",
			Message:  `<script>alert('xss')</script>`,
			Count:    3,
			LastObs:  "2026-06-11T00:00:00Z",
		}},
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("execute: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, "<script>alert") {
		t.Fatal("raw <script> payload leaked into output — XSS vulnerable")
	}
	if !strings.Contains(out, "&lt;script&gt;") {
		t.Fatalf("expected escaped payload in output, got: %s", out)
	}
	if strings.Contains(out, "<img src=x>") {
		t.Fatal("raw <img> payload leaked into subject — XSS vulnerable")
	}
}

// TestToRow flattens an unstructured DriftReport into a row.
func TestToRow(t *testing.T) {
	u := &unstructured.Unstructured{Object: map[string]interface{}{
		"spec": map[string]interface{}{
			"subject":  "Secret/cha-playground/playground-config/ABSENT_KEY",
			"severity": "warning",
			"source":   "SecretKeyMissing",
			"message":  "couldn't find key ABSENT_KEY",
		},
		"status": map[string]interface{}{
			"observationCount": int64(5),
			"lastObserved":     "2026-06-11T01:02:03Z",
		},
	}}
	got := toRow(u)
	if got.Source != "SecretKeyMissing" || got.Severity != "warning" || got.Count != 5 {
		t.Fatalf("unexpected row: %+v", got)
	}
}
