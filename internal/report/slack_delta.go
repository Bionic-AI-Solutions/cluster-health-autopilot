// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package report

import (
	"fmt"
	"strings"
	"time"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/fix"
)

// DeltaDiag is a new or changed diagnostic surfaced in a watcher Slack post.
type DeltaDiag struct {
	Subject     string
	Severity    string // info | warning | critical
	Message     string
	Remediation string
}

// ResolvedDiag is a diagnostic that no longer appears in the current cycle.
type ResolvedDiag struct {
	Subject string
	Message string
}

// FormatSlackDelta renders a condensed watcher-mode message containing only
// the diagnostic delta (new/changed issues, resolved issues, fix actions).
// Called once per cycle where something changed; silent cycles skip this entirely.
func FormatSlackDelta(
	newOrChanged []DeltaDiag,
	resolved []ResolvedDiag,
	fixResults []fix.Result,
	autopilot bool,
) SlackPayload {
	now := time.Now().UTC()
	var b strings.Builder

	fmt.Fprintf(&b, "*Cluster Health Autopilot — Watch* — %s\n", now.Format("2006-01-02 15:04:05 UTC"))

	if len(newOrChanged) > 0 {
		fmt.Fprintf(&b, "\n*🔔 Active Issues (%d):*\n", len(newOrChanged))
		for _, d := range newOrChanged {
			icon := severityWatchIcon(d.Severity)
			fmt.Fprintf(&b, "• %s *%s*\n  %s\n", icon, d.Subject, d.Message)
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

	totalActions := 0
	for _, fr := range fixResults {
		totalActions += len(fr.Actions)
	}
	if totalActions > 0 {
		fmt.Fprintf(&b, "\n*🔧 Fixes Applied (%d):*\n", totalActions)
		for _, fr := range fixResults {
			for _, a := range fr.Actions {
				fmt.Fprintf(&b, "• %s — `%s`\n", a.Description, a.Object)
			}
		}
	}

	color := attachmentColor(newOrChanged, resolved)
	footer := "K8s Cluster Health Autopilot — Watch mode"
	if autopilot {
		footer += " (auto-remediation: ON)"
	}

	return SlackPayload{
		Username:  "Cluster Health Monitor",
		IconEmoji: ":hospital:",
		Attachments: []SlackAttachment{{
			Color:    color,
			Text:     b.String(),
			Footer:   footer,
			Ts:       now.Unix(),
			MrkdwnIn: []string{"text"},
		}},
	}
}

func severityWatchIcon(severity string) string {
	switch severity {
	case "critical":
		return "❌"
	case "warning":
		return "⚠️"
	default:
		return "ℹ️"
	}
}

func attachmentColor(newOrChanged []DeltaDiag, resolved []ResolvedDiag) string {
	for _, d := range newOrChanged {
		if d.Severity == "critical" {
			return "danger"
		}
	}
	if len(newOrChanged) > 0 {
		return "warning"
	}
	if len(resolved) > 0 {
		return "good"
	}
	return "warning"
}
