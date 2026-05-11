// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package report

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/fix"
)

// SlackChannels holds webhook URLs for the unified three-channel alert routing model.
//
//   - Alerts   → #ceph-alerts:   CHA acted (fixers ran and resolved issues)
//   - Critical → #ceph-critical: human action required (unfixable or still active)
//
// Either field may be empty; posts are silently skipped for empty URLs.
type SlackChannels struct {
	Alerts   string // #ceph-alerts — event-driven, CHA acted
	Critical string // #ceph-critical — event-driven, needs human
}

// FormatAlertsPayload renders the #ceph-alerts message for a watcher cycle
// where CHA auto-remediation fired. Shows what triggered the fix and what
// actions were taken.
func FormatAlertsPayload(fixedIssues []DeltaDiag, fixResults []fix.Result) SlackPayload {
	now := time.Now().UTC()
	var b strings.Builder

	fmt.Fprintf(&b, "*CHA Auto-Remediation* — %s\n", now.Format("2006-01-02 15:04:05 UTC"))

	if len(fixedIssues) > 0 {
		fmt.Fprintf(&b, "\n*⚡ Triggered by (%d):*\n", len(fixedIssues))
		for _, d := range fixedIssues {
			fmt.Fprintf(&b, "• %s *%s*\n  %s\n", severityWatchIcon(d.Severity), d.Subject, d.Message)
		}
	}

	totalActions := 0
	for _, fr := range fixResults {
		totalActions += len(fr.Actions)
	}
	if totalActions > 0 {
		fmt.Fprintf(&b, "\n*🔧 Actions taken (%d):*\n", totalActions)
		for _, fr := range fixResults {
			for _, a := range fr.Actions {
				fmt.Fprintf(&b, "• %s — `%s`\n", a.Description, a.Object)
			}
		}
	}

	return SlackPayload{
		Username:  "Cluster Health Monitor",
		IconEmoji: ":wrench:",
		Attachments: []SlackAttachment{{
			Color:    "good",
			Text:     b.String(),
			Footer:   "CHA Auto-Remediation",
			Ts:       now.Unix(),
			MrkdwnIn: []string{"text"},
		}},
	}
}

// FormatCriticalPayload renders the #ceph-critical message for a watcher cycle
// where issues require human intervention — either unfixable by CHA or still
// active after fixers ran.
func FormatCriticalPayload(unfixable []DeltaDiag, resolved []ResolvedDiag) SlackPayload {
	now := time.Now().UTC()
	var b strings.Builder

	fmt.Fprintf(&b, "*CHA Alert — Human Action Required* — %s\n", now.Format("2006-01-02 15:04:05 UTC"))

	crits := 0
	diags := 0
	for _, d := range unfixable {
		if d.Severity == "critical" {
			crits++
		} else {
			diags++
		}
	}

	if crits > 0 {
		fmt.Fprintf(&b, "\n*🔴 Critical (%d):*\n", crits)
		for _, d := range unfixable {
			if d.Severity != "critical" {
				continue
			}
			fmt.Fprintf(&b, "• ❌ *%s*\n  %s\n", d.Subject, d.Message)
			if d.Remediation != "" {
				fmt.Fprintf(&b, "  _→ %s_\n", d.Remediation)
			}
		}
	}

	if diags > 0 {
		fmt.Fprintf(&b, "\n*⚠️ Diagnostics (%d):*\n", diags)
		for _, d := range unfixable {
			if d.Severity == "critical" {
				continue
			}
			fmt.Fprintf(&b, "• ⚠️ *%s*\n  %s\n", d.Subject, d.Message)
			if d.Remediation != "" {
				fmt.Fprintf(&b, "  _→ %s_\n", d.Remediation)
			}
		}
	}

	if len(resolved) > 0 {
		fmt.Fprintf(&b, "\n*✅ Resolved (%d):*\n", len(resolved))
		for _, r := range resolved {
			fmt.Fprintf(&b, "• `%s`\n", r.Subject)
			if r.Message != "" {
				fmt.Fprintf(&b, "  _%s_\n", r.Message)
			}
		}
	}

	color := "danger"
	if crits == 0 && diags > 0 {
		color = "warning"
	}
	if crits == 0 && diags == 0 && len(resolved) > 0 {
		color = "good"
	}

	return SlackPayload{
		Username:  "Cluster Health Monitor",
		IconEmoji: ":rotating_light:",
		Attachments: []SlackAttachment{{
			Color:    color,
			Text:     b.String(),
			Footer:   "CHA — Human action required",
			Ts:       now.Unix(),
			MrkdwnIn: []string{"text"},
		}},
	}
}

// RouteAndPost splits watcher cycle results into the correct Slack channels:
//   - issues that disappeared from postFix (fixed by CHA) → channels.Alerts
//   - issues still present in postFix (unfixable) + resolved → channels.Critical
//
// postFixSubjects is the set of subject keys still active after fixers ran.
// Any subject in toPost that is absent from postFixSubjects was fixed this cycle.
// Either channel URL may be empty — posts are silently skipped for empty strings.
func RouteAndPost(
	client *http.Client,
	channels SlackChannels,
	postFixSubjects map[string]bool,
	toPost []DeltaDiag,
	toResolve []ResolvedDiag,
	fixResults []fix.Result,
) {
	var fixedIssues, unfixable []DeltaDiag
	for _, d := range toPost {
		if postFixSubjects[d.Subject] {
			unfixable = append(unfixable, d)
		} else {
			fixedIssues = append(fixedIssues, d)
		}
	}

	hasFixerActions := false
	for _, fr := range fixResults {
		if len(fr.Actions) > 0 {
			hasFixerActions = true
			break
		}
	}

	if channels.Alerts != "" && (len(fixedIssues) > 0 || hasFixerActions) {
		payload := FormatAlertsPayload(fixedIssues, fixResults)
		if err := PostSlack(client, channels.Alerts, payload); err != nil {
			log.Printf("report: slack post to alerts channel: %v", err)
		}
	}

	if channels.Critical != "" && (len(unfixable) > 0 || len(toResolve) > 0) {
		payload := FormatCriticalPayload(unfixable, toResolve)
		if err := PostSlack(client, channels.Critical, payload); err != nil {
			log.Printf("report: slack post to critical channel: %v", err)
		}
	}
}
