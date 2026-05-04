// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

// Package summarize reads anonymized JSONL run records from the runs/
// directory and produces a human-readable Markdown SUMMARY.md.
//
// The summary is regenerated nightly by the publish-runs GitHub Action and
// linked from the README. It provides the "proof of work" needed for design-
// partner conversations: trend data showing how many issues the analyzer
// surfaces per run, fix rate, and false-positive-free signal over time.
package summarize

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/anonymize"
)

// Stats holds aggregate per-run metrics derived from a RunRecord.
type Stats struct {
	Date            string
	RunID           string
	ChaVersion      string
	TotalComponents int
	HealthyCount    int
	DegradedCount   int
	CriticalCount   int
	FindingCount    int
	DiagnosticCount int
}

// CategoryFreq is a ranked diagnostic/finding category with occurrence count.
type CategoryFreq struct {
	Category string
	Count    int
}

// Report is the aggregated view across all runs in a directory.
type Report struct {
	FirstDate   string
	LastDate    string
	RunCount    int
	Runs        []Stats
	TopDiags    []CategoryFreq // top diagnostic subject prefixes by frequency
	TopFindings []CategoryFreq // top finding severity+component pairs by frequency
}

// FromDir reads all *.jsonl files in dir, parses each line as a RunRecord,
// and returns an aggregated Report. Files are processed in lexicographic
// order (which equals chronological order given YYYY-MM-DD filenames).
func FromDir(dir string) (*Report, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read runs dir %q: %w", dir, err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".jsonl") {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(files)

	r := &Report{}
	diagFreq := map[string]int{}
	findingFreq := map[string]int{}

	for _, f := range files {
		recs, err := readJSONL(f)
		if err != nil {
			return nil, err
		}
		for _, rec := range recs {
			r.RunCount++
			s := Stats{
				Date:            dateFromTimestamp(rec.Timestamp),
				RunID:           rec.RunID,
				ChaVersion:      rec.ChaVersion,
				TotalComponents: rec.Summary.TotalComponents,
				HealthyCount:    rec.Summary.HealthyCount,
				DegradedCount:   rec.Summary.DegradedCount,
				CriticalCount:   rec.Summary.CriticalCount,
				FindingCount:    rec.Summary.FindingCount,
				DiagnosticCount: rec.Summary.DiagnosticCount,
			}
			r.Runs = append(r.Runs, s)

			if r.FirstDate == "" || s.Date < r.FirstDate {
				r.FirstDate = s.Date
			}
			if s.Date > r.LastDate {
				r.LastDate = s.Date
			}

			// Tally diagnostic subject prefixes (first segment = category).
			for _, d := range rec.Diagnostics {
				cat := subjectCategory(d.Subject)
				diagFreq[cat]++
			}
			// Tally finding severity+component.
			for _, res := range rec.Results {
				for _, f := range res.Findings {
					key := f.Severity + "/" + res.Component.Component
					findingFreq[key]++
				}
			}
		}
	}

	r.TopDiags = topN(diagFreq, 10)
	r.TopFindings = topN(findingFreq, 10)
	return r, nil
}

// Render writes a Markdown SUMMARY.md to w.
func (r *Report) Render(w io.Writer) {
	ew := &errWriter{w: w}
	now := time.Now().UTC().Format("2006-01-02 15:04 UTC")
	ew.printf("# Cluster Health Autopilot — Run Summary\n\n")
	ew.printf("_Auto-generated %s · %d run(s) · %s → %s_\n\n",
		now, r.RunCount, r.FirstDate, r.LastDate)

	if r.RunCount == 0 {
		ew.printf("_No runs published yet._\n")
		return
	}

	ew.printf("## Health trend\n\n")
	ew.printf("| Date | Run | Components | Healthy | Degraded | Critical | Findings | Diagnostics |\n")
	ew.printf("|---|---|---|---|---|---|---|---|\n")
	for _, s := range r.Runs {
		ew.printf("| %s | %s | %d | %d | %d | %d | %d | %d |\n",
			s.Date, s.RunID,
			s.TotalComponents, s.HealthyCount, s.DegradedCount, s.CriticalCount,
			s.FindingCount, s.DiagnosticCount,
		)
	}
	ew.printf("\n")

	if len(r.TopDiags) > 0 {
		ew.printf("## Diagnostic patterns (top categories, anonymized)\n\n")
		ew.printf("| Category | Occurrences |\n")
		ew.printf("|---|---|\n")
		for _, c := range r.TopDiags {
			ew.printf("| `%s` | %d |\n", c.Category, c.Count)
		}
		ew.printf("\n")
	}

	if len(r.TopFindings) > 0 {
		ew.printf("## Component findings (top, anonymized)\n\n")
		ew.printf("| Severity/Component | Occurrences |\n")
		ew.printf("|---|---|\n")
		for _, c := range r.TopFindings {
			ew.printf("| `%s` | %d |\n", c.Category, c.Count)
		}
		ew.printf("\n")
	}

	ew.printf("---\n")
	ew.printf("_All namespace, workload, and secret names are anonymized using deterministic SHA-256 hashing._\n")
	ew.printf("_cha version(s) in this dataset: %s_\n", versionList(r.Runs))
	// errWriter.err is intentionally ignored: writing to stdout/file fails silently
	// (the OS will surface the error via the process exit code).
	_ = ew.err
}

// errWriter captures the first write error so callers can use unchecked fmt.Fprintf chains.
type errWriter struct {
	w   io.Writer
	err error
}

func (e *errWriter) printf(format string, args ...any) {
	if e.err != nil {
		return
	}
	_, e.err = fmt.Fprintf(e.w, format, args...)
}

// ── helpers ─────────────────────────────────────────────────────────────────

func readJSONL(path string) ([]anonymize.RunRecord, error) {
	fh, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %q: %w", path, err)
	}
	defer func() { _ = fh.Close() }()

	var out []anonymize.RunRecord
	sc := bufio.NewScanner(fh)
	sc.Buffer(make([]byte, 1<<20), 1<<20) // 1 MB per line
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var rec anonymize.RunRecord
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			return nil, fmt.Errorf("%s line %d: %w", path, lineNo, err)
		}
		out = append(out, rec)
	}
	return out, sc.Err()
}

func dateFromTimestamp(ts string) string {
	if len(ts) >= 10 {
		return ts[:10]
	}
	return ts
}

func subjectCategory(subject string) string {
	if i := strings.Index(subject, "/"); i != -1 {
		return subject[:i]
	}
	return subject
}

func topN(freq map[string]int, n int) []CategoryFreq {
	all := make([]CategoryFreq, 0, len(freq))
	for k, v := range freq {
		all = append(all, CategoryFreq{Category: k, Count: v})
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].Count != all[j].Count {
			return all[i].Count > all[j].Count
		}
		return all[i].Category < all[j].Category
	})
	if len(all) > n {
		all = all[:n]
	}
	return all
}

func versionList(runs []Stats) string {
	seen := map[string]struct{}{}
	for _, r := range runs {
		if r.ChaVersion != "" {
			seen[r.ChaVersion] = struct{}{}
		}
	}
	vs := make([]string, 0, len(seen))
	for v := range seen {
		vs = append(vs, v)
	}
	sort.Strings(vs)
	return strings.Join(vs, ", ")
}
