// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package report

import (
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/srenix-ai/agentic-sre/internal/diagnose"
	"github.com/srenix-ai/agentic-sre/internal/probe"
)

// DriftReportList is a thin view over the DriftReport CR list used by
// FormatDailyDigest. The caller passes the raw unstructured items from
// snapshot.Source.List.
type DriftReportList struct {
	Items []unstructured.Unstructured
}

// dailyEntry is an extracted view of a single DriftReport CR.
type dailyEntry struct {
	subject       string
	message       string
	remediation   string
	severity      string
	category      string
	firstObserved time.Time
	lastObserved  time.Time
	count         int64
}

// FormatDailyDigest renders the 09:00 #healthinfo daily report. It combines:
//
//  1. Current probe component status (healthy / degraded / critical).
//  2. Active diagnostics (analyzer findings currently in the cluster).
//  3. Issues that first appeared in the last 24 h (new entries).
//  4. Issues that have been open longer than 24 h (persistent entries).
//
// drList may be nil or empty — the history sections are silently omitted.
func FormatDailyDigest(
	results []probe.Result,
	diagnostics []diagnose.Diagnostic,
	drList *DriftReportList,
) SlackPayload {
	now := time.Now().UTC()
	cutoff24h := now.Add(-24 * time.Hour)

	_, color, headline := overallStatus(results)

	var b strings.Builder
	fmt.Fprintf(&b, ":hospital: *Agentic SRE* — Daily Report  %s\n", now.Format("2006-01-02 15:04 UTC"))
	fmt.Fprintf(&b, "%s\n\n", headline)

	// Component status table.
	b.WriteString("*Component Status:*\n")
	for _, r := range results {
		fmt.Fprintf(&b, "• *%s:* %s  %s\n", r.Component.Component, statusEmoji(r.Component.Status), r.Component.Detail)
	}

	// Active diagnostics (from the current run — not from DriftReports).
	if len(diagnostics) > 0 {
		fmt.Fprintf(&b, "\n*Active Diagnostics (%d):*\n", len(diagnostics))
		for _, d := range diagnostics {
			fmt.Fprintf(&b, "• 🔎 %s\n  %s\n", d.Subject, d.Message)
			if d.Remediation != "" {
				fmt.Fprintf(&b, "  _→ %s_\n", d.Remediation)
			}
		}
	}

	// History sections derived from DriftReport CRs.
	if drList != nil && len(drList.Items) > 0 {
		entries := extractDriftEntries(drList.Items)

		var newIssues, persistent []dailyEntry
		for _, e := range entries {
			if e.category == "fixer-action" || e.category == "fixer-skipped" {
				continue
			}
			if e.firstObserved.After(cutoff24h) {
				newIssues = append(newIssues, e)
			} else {
				persistent = append(persistent, e)
			}
		}

		var recentFixes []dailyEntry
		for _, e := range entries {
			if e.category == "fixer-action" && e.lastObserved.After(cutoff24h) {
				recentFixes = append(recentFixes, e)
			}
		}

		if len(newIssues) > 0 {
			fmt.Fprintf(&b, "\n*📋 New Issues (last 24h) — %d:*\n", len(newIssues))
			for _, e := range newIssues {
				age := now.Sub(e.firstObserved).Round(time.Minute)
				fmt.Fprintf(&b, "• 🆕 `%s`  _(appeared %s ago)_\n  %s\n", e.subject, formatAge(age), e.message)
				if e.remediation != "" {
					fmt.Fprintf(&b, "  _→ %s_\n", e.remediation)
				}
			}
		}

		if len(persistent) > 0 {
			fmt.Fprintf(&b, "\n*⏳ Persistent Issues (>24h) — %d:*\n", len(persistent))
			for _, e := range persistent {
				age := now.Sub(e.firstObserved).Round(time.Hour)
				fmt.Fprintf(&b, "• ⏳ `%s`  _(open %s, seen %d times)_\n  %s\n",
					e.subject, formatAge(age), e.count, e.message)
			}
		}

		if len(recentFixes) > 0 {
			fmt.Fprintf(&b, "\n*🔧 Auto-Fixed (last 24h) — %d:*\n", len(recentFixes))
			for _, e := range recentFixes {
				fmt.Fprintf(&b, "• ✅ %s\n", e.message)
			}
		}
	}

	return SlackPayload{
		Username:  "Cluster Health Monitor",
		IconEmoji: ":hospital:",
		Attachments: []SlackAttachment{{
			Color:    color,
			Text:     b.String(),
			Footer:   "K8s Agentic SRE — Daily Digest",
			Ts:       now.Unix(),
			MrkdwnIn: []string{"text"},
		}},
	}
}

// extractDriftEntries parses the raw unstructured DriftReport CRs into
// typed dailyEntry values.
func extractDriftEntries(items []unstructured.Unstructured) []dailyEntry {
	out := make([]dailyEntry, 0, len(items))
	for _, cr := range items {
		spec, _, _ := unstructured.NestedMap(cr.Object, "spec")
		status, _, _ := unstructured.NestedMap(cr.Object, "status")
		if spec == nil {
			continue
		}
		subject, _ := spec["subject"].(string)
		if subject == "" {
			continue
		}
		e := dailyEntry{
			subject:     subject,
			message:     strVal(spec["message"]),
			remediation: strVal(spec["remediation"]),
			severity:    strVal(spec["severity"]),
			category:    strVal(spec["category"]),
		}
		if status != nil {
			e.firstObserved = parseRFC3339(strVal(status["firstObserved"]))
			e.lastObserved = parseRFC3339(strVal(status["lastObserved"]))
			switch v := status["observationCount"].(type) {
			case int64:
				e.count = v
			case float64:
				e.count = int64(v)
			}
		}
		out = append(out, e)
	}
	return out
}

func strVal(v any) string {
	s, _ := v.(string)
	return s
}

func parseRFC3339(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

func formatAge(d time.Duration) string {
	if d < time.Minute {
		return "< 1m"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	if hours == 0 {
		return fmt.Sprintf("%dd", days)
	}
	return fmt.Sprintf("%dd%dh", days, hours)
}
