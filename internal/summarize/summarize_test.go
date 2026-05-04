// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package summarize

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/anonymize"
)

func writeJSONL(t *testing.T, dir, filename string, recs []anonymize.RunRecord) {
	t.Helper()
	fh, err := os.Create(filepath.Join(dir, filename))
	if err != nil {
		t.Fatalf("create %s: %v", filename, err)
	}
	defer func() { _ = fh.Close() }()
	enc := json.NewEncoder(fh)
	for _, r := range recs {
		if err := enc.Encode(r); err != nil {
			t.Fatalf("encode: %v", err)
		}
	}
}

func TestFromDir_Empty(t *testing.T) {
	dir := t.TempDir()
	r, err := FromDir(dir)
	if err != nil {
		t.Fatalf("FromDir: %v", err)
	}
	if r.RunCount != 0 {
		t.Errorf("want 0 runs, got %d", r.RunCount)
	}
}

func TestFromDir_SingleFile(t *testing.T) {
	dir := t.TempDir()
	writeJSONL(t, dir, "2026-05-01.jsonl", []anonymize.RunRecord{
		{
			SchemaVersion: "1",
			RunID:         "run-1",
			Timestamp:     "2026-05-01T02:00:00Z",
			ChaVersion:    "0.5.0",
			Summary: anonymize.RunSummary{
				TotalComponents: 4,
				HealthyCount:    3,
				DegradedCount:   1,
				FindingCount:    2,
				DiagnosticCount: 1,
			},
			Diagnostics: []anonymize.AnonDiagnostic{
				{Subject: "missing-key/abcd1234/efgh5678/ijkl9012"},
			},
		},
	})

	r, err := FromDir(dir)
	if err != nil {
		t.Fatalf("FromDir: %v", err)
	}
	if r.RunCount != 1 {
		t.Errorf("run count = %d, want 1", r.RunCount)
	}
	if r.FirstDate != "2026-05-01" || r.LastDate != "2026-05-01" {
		t.Errorf("dates wrong: first=%q last=%q", r.FirstDate, r.LastDate)
	}
	if len(r.TopDiags) != 1 || r.TopDiags[0].Category != "missing-key" {
		t.Errorf("top diag category wrong: %v", r.TopDiags)
	}
}

func TestFromDir_MultipleFiles(t *testing.T) {
	dir := t.TempDir()
	for _, d := range []string{"2026-05-01", "2026-05-02", "2026-05-03"} {
		writeJSONL(t, dir, d+".jsonl", []anonymize.RunRecord{
			{
				SchemaVersion: "1",
				RunID:         "run",
				Timestamp:     d + "T02:00:00Z",
				ChaVersion:    "0.5.0",
				Summary:       anonymize.RunSummary{TotalComponents: 4, HealthyCount: 4},
			},
		})
	}
	r, err := FromDir(dir)
	if err != nil {
		t.Fatalf("FromDir: %v", err)
	}
	if r.RunCount != 3 {
		t.Errorf("run count = %d, want 3", r.RunCount)
	}
	if r.FirstDate != "2026-05-01" || r.LastDate != "2026-05-03" {
		t.Errorf("date range wrong: %q → %q", r.FirstDate, r.LastDate)
	}
}

func TestRender_ContainsExpectedSections(t *testing.T) {
	dir := t.TempDir()
	writeJSONL(t, dir, "2026-05-01.jsonl", []anonymize.RunRecord{
		{
			SchemaVersion: "1",
			RunID:         "r1",
			Timestamp:     "2026-05-01T02:00:00Z",
			ChaVersion:    "0.5.0",
			Summary: anonymize.RunSummary{
				TotalComponents: 3,
				HealthyCount:    2,
				DegradedCount:   1,
				FindingCount:    1,
				DiagnosticCount: 2,
			},
			Diagnostics: []anonymize.AnonDiagnostic{
				{Subject: "missing-key/aa/bb/cc"},
				{Subject: "missing-vault-path/dd/ee"},
			},
			Results: []anonymize.AnonResult{
				{
					Component: anonymize.AnonComponentResult{
						Component: "Ceph Storage",
						Status:    "DEGRADED",
					},
					Findings: []anonymize.AnonFinding{
						{Component: "Ceph Storage", Severity: "warning", Message: "low disk"},
					},
				},
			},
		},
	})
	r, _ := FromDir(dir)
	var sb strings.Builder
	r.Render(&sb)
	md := sb.String()

	for _, want := range []string{
		"# Cluster Health Autopilot",
		"## Health trend",
		"2026-05-01",
		"## Diagnostic patterns",
		"missing-key",
		"missing-vault-path",
		"## Component findings",
		"anonymized",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("rendered SUMMARY.md missing %q", want)
		}
	}
}

func TestRender_EmptyRunsMessage(t *testing.T) {
	r := &Report{}
	var sb strings.Builder
	r.Render(&sb)
	if !strings.Contains(sb.String(), "No runs") {
		t.Errorf("empty report should say 'No runs', got: %s", sb.String())
	}
}

func TestSubjectCategory(t *testing.T) {
	cases := [][2]string{
		{"missing-key/ns/name/key", "missing-key"},
		{"vault-missing-key/p/q/r", "vault-missing-key"},
		{"missing-vault-path/a/b", "missing-vault-path"},
		{"vault-store-rbac-missing", "vault-store-rbac-missing"},
	}
	for _, c := range cases {
		if got := subjectCategory(c[0]); got != c[1] {
			t.Errorf("subjectCategory(%q) = %q, want %q", c[0], got, c[1])
		}
	}
}
