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

	// IsNewThisCycle distinguishes findings that just appeared (new
	// subject OR fingerprint changed since the prior cycle) from
	// findings being re-posted because the repeat-interval elapsed.
	// SplitCriticalPayloads surfaces the new-this-cycle subset in a
	// dedicated "🆕 New this cycle (N)" section that renders BEFORE
	// the steady-state list, so operators reading Slack can tell at-a-
	// glance what changed since their last look. Populated by the
	// watcher's diff() — false on legacy callers (e.g. tests not yet
	// migrated, pkg/report consumers that snapshot DeltaDiag from
	// DriftReport CRs).
	IsNewThisCycle bool

	// Investigation is the Layer-2 investigator's summary. Populated by
	// the OSS rule-based investigator or any registered pkg/ai.Investigator.
	Investigation string

	// AI tier fields — optional, populated only when CHA-com's AI tier
	// is active. OSS deployments never see these set.

	// Enrichment is the LLM-generated narrative addendum (T0+).
	Enrichment string

	// ProposedActionID links to a T1 single-action proposal. When set,
	// the renderer emits an Apply Fix button alongside this entry.
	ProposedActionID string

	// ApprovalURL is the signed click-to-fix URL. Only set in tandem
	// with ProposedActionID.
	ApprovalURL string

	// Phase 2.B.6 — class-action URLs. Pre-signed with class-scoped
	// JWTs that target separate approval-server endpoints; the
	// operator's one click writes a policy AND (for approve-class)
	// executes the action.
	//
	// IMPORTANT: in v1.21.0 these fields are render-only on the OSS
	// watcher path — the OSS internal/watcher/enrich pipeline does NOT
	// yet mint class-action JWTs (the class_token signer lives in
	// CHA-com's ai/approval package). The CHA-com aiwatch emits class
	// buttons via its OWN renderer (cmd/cha-com/render.go), which IS
	// fully wired and verified live since v1.16.0. These fields land
	// here so a future OSS hook (or a shared signer extraction) can
	// populate them without re-touching the render path; until then
	// they stay empty in pure-OSS deploys and the render gates the
	// class-button row on non-empty values.
	ApproveClassURL string
	DenyClassURL    string
	SilenceClassURL string
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
			// To-silence one-liner — paste into a terminal to create a
			// Silence CR that suppresses THIS finding for 24h. Lets
			// SREs distinguish "fix this" from "noisy, mute" without
			// hunting for the CRD schema. Subject-scoped (exact match);
			// edit spec.matcher to `source` for class-wide suppression.
			fmt.Fprintf(&b, "  🔕 silence 24h: ```kubectl apply -f - <<EOF\n"+
				"apiVersion: cha.bionicaisolutions.com/v1alpha1\n"+
				"kind: Silence\nmetadata:\n  name: %s\n  namespace: cluster-health-autopilot\n"+
				"spec:\n  matcher:\n    subject: %q\n  until: %q\n  reason: silenced-from-slack\nEOF```\n",
				slackSilenceName(d.Subject), d.Subject,
				time.Now().UTC().Add(24*time.Hour).Format("2006-01-02T15:04:05Z"))
			if d.Investigation != "" {
				fmt.Fprintf(&b, "  🔬 _%s_\n", d.Investigation)
			}
			if d.Enrichment != "" {
				fmt.Fprintf(&b, "  🤖 _%s_\n", d.Enrichment)
			}
			if d.ApprovalURL != "" {
				// Render symmetric Approve / Deny pair. The deny URL
				// shares the JTI with approve (cha-com #17 symmetric
				// one-shot tokens) — whichever the SRE clicks first
				// wins, the other is burned. Denial records a RAG
				// outcome so the proposer learns from rejections.
				denyURL := strings.Replace(d.ApprovalURL, "/approve?", "/deny?", 1)
				fmt.Fprintf(&b, "  ✅ <%s|Approve> · ❌ <%s|Deny> · 📄 <%s&action=info|Details>\n",
					d.ApprovalURL, denyURL, d.ApprovalURL)
				// Phase 2.B.6 — class-action row. Hidden when class
				// URLs are empty (legacy memory-off deploy stays
				// byte-identical to pre-2.B Slack output).
				if d.ApproveClassURL != "" || d.DenyClassURL != "" || d.SilenceClassURL != "" {
					b.WriteString("  ")
					sep := ""
					if d.ApproveClassURL != "" {
						fmt.Fprintf(&b, "%s🧠 <%s|Approve+remember class>", sep, d.ApproveClassURL)
						sep = " · "
					}
					if d.DenyClassURL != "" {
						fmt.Fprintf(&b, "%s❌ <%s|Deny+remember class>", sep, d.DenyClassURL)
						sep = " · "
					}
					if d.SilenceClassURL != "" {
						fmt.Fprintf(&b, "%s🔕 <%s|Silence class (7d)>", sep, d.SilenceClassURL)
					}
					b.WriteString("\n")
				}
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

// slackSilenceName turns a finding Subject into a K8s-DNS-safe Silence
// CR name. Lowercased, non-[a-z0-9-] characters replaced with '-',
// collapsed dashes, max 50 chars + "silence-" prefix → ≤58 chars
// well within the K8s 63-char name limit.
func slackSilenceName(subject string) string {
	var b strings.Builder
	prev := byte('-')
	for i := 0; i < len(subject); i++ {
		c := subject[i]
		switch {
		case c >= 'A' && c <= 'Z':
			b.WriteByte(c + 32)
			prev = c + 32
		case (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9'):
			b.WriteByte(c)
			prev = c
		default:
			if prev != '-' {
				b.WriteByte('-')
				prev = '-'
			}
		}
	}
	name := strings.Trim(b.String(), "-")
	if len(name) > 50 {
		name = name[:50]
	}
	return "silence-" + name
}
