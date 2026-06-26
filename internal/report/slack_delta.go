// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package report

import (
	"fmt"
	"strings"
	"time"

	"github.com/srenix-ai/agentic-sre/internal/fix"
)

// DeltaDiag is the CANONICAL alert model for the report layer. Every delivery
// adapter — Slack (native), ticketing, and (when enabled) Alertmanager —
// consumes this one struct; none re-derives content or severity. The watcher
// builds it once (seenEntryToDeltaDiag) with the severity already normalized,
// and the render/ticketing layers only FORMAT it. The shared renderGuidance /
// RecommendedAction helpers compose the operator-facing "what's wrong + what to
// do" identically across adapters, root-cause-first.
type DeltaDiag struct {
	Subject     string
	Severity    string // info | warning | critical (normalized at build time)
	Message     string
	Remediation string

	// Source names the analyzer / probe that produced this finding
	// (e.g. "FailingExternalSecrets"). It is the matcher.source for the
	// class-scoped one-click Silence link — muting the WHOLE class, not
	// just this Subject. Empty on legacy callers; the class silence link
	// is only minted/rendered when Source is set.
	Source string

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

	// AI tier fields — optional, populated only when Srenix Enterprise's AI tier
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
	// Srenix Enterprise's ai/approval package). The Srenix Enterprise aiwatch emits class
	// buttons via its OWN renderer (cmd/srenix-enterprise/render.go), which IS
	// fully wired and verified live since v1.16.0. These fields land
	// here so a future OSS hook (or a shared signer extraction) can
	// populate them without re-touching the render path; until then
	// they stay empty in pure-OSS deploys and the render gates the
	// class-button row on non-empty values.
	ApproveClassURL string
	DenyClassURL    string
	SilenceClassURL string

	// One-click signed Silence links — minted by the OSS watcher's
	// attachApprovalURLs when a signing key + approval base URL are
	// configured (see internal/watcher + report.SilenceLinkConfig).
	//
	//   - SilenceSubjectURL   snoozes THIS finding (matcher.subject =
	//     Subject) for the configured SHORT window (default 24h).
	//   - SilenceClassLongURL mutes the finding's whole class
	//     (matcher.source = Source) for the configured LONG window
	//     (default 90d).
	//
	// Both are empty in OSS-only / air-gapped installs (no key / no
	// approval-server). The renderer gates on them: when set it emits the
	// click-links; when unset it falls back to the kubectl one-liner so the
	// air-gapped affordance is never lost.
	//
	// SilenceShortDur / SilenceLongDur carry the configured windows so the
	// renderer can label the links with the real duration ("Silence 24h",
	// "Silence class (90d)") instead of hardcoded text. Zero values fall
	// back to the package defaults when a link URL is nonetheless present.
	SilenceSubjectURL   string
	SilenceClassLongURL string
	SilenceShortDur     time.Duration
	SilenceLongDur      time.Duration
}

// ResolvedDiag is a diagnostic that no longer appears in the current cycle.
// It carries the same canonical content as DeltaDiag so a resolution message
// can say WHAT was wrong (root cause) and that it cleared — not just the bare
// subject. Severity/Remediation/Investigation are empty on legacy callers.
type ResolvedDiag struct {
	Subject       string
	Message       string
	Severity      string
	Remediation   string
	Investigation string
}

// renderGuidance writes the operator-facing guidance for one finding,
// root-cause-FIRST: the Layer-2 investigator's definitive cause (when present)
// before the generic remediation steps. This is the single composition every
// Slack renderer uses, so "definitive cause, not a kubectl recipe" is true
// uniformly. Caller controls indentation via the leading spaces here.
func renderGuidance(b *strings.Builder, investigation, remediation string) {
	if investigation != "" {
		fmt.Fprintf(b, "  🔬 _Root cause: %s_\n", investigation)
	}
	if remediation != "" {
		fmt.Fprintf(b, "  _→ %s_\n", remediation)
	}
}

// renderResolvedRootCause appends the cleared finding's root cause so a
// resolution message says WHAT was wrong, not just that it cleared. No-op when
// the resolved finding carried no investigation.
func renderResolvedRootCause(b *strings.Builder, r ResolvedDiag) {
	if r.Investigation != "" {
		fmt.Fprintf(b, "  🔬 _was: %s_\n", r.Investigation)
	}
}

// RecommendedAction composes a one-string, root-cause-first guidance value for
// non-Slack consumers (ticketing bodies, Alertmanager annotations): the
// investigator's cause leads, then the remediation steps. Falls back to
// whichever side is present.
func RecommendedAction(investigation, remediation string) string {
	switch {
	case investigation != "" && remediation != "":
		return "Root cause: " + investigation + "\n\nNext steps: " + remediation
	case investigation != "":
		return "Root cause: " + investigation
	default:
		return remediation
	}
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

	fmt.Fprintf(&b, "*Agentic SRE — Watch* — %s\n", now.Format("2006-01-02 15:04:05 UTC"))

	if len(newOrChanged) > 0 {
		fmt.Fprintf(&b, "\n*🔔 Active Issues (%d):*\n", len(newOrChanged))
		for _, d := range newOrChanged {
			icon := severityWatchIcon(d.Severity)
			fmt.Fprintf(&b, "• %s *%s*\n  %s\n", icon, d.Subject, d.Message)
			// Root-cause-first guidance (investigation before remediation).
			renderGuidance(&b, d.Investigation, d.Remediation)
			// Silence affordance: only for findings that have been present
			// for at least one cycle. A brand-new problem should be
			// investigated, not immediately silenced.
			if !d.IsNewThisCycle {
				renderSilenceSnippet(&b, d)
			}
			if d.Enrichment != "" {
				fmt.Fprintf(&b, "  🤖 _%s_\n", d.Enrichment)
			}
			if d.ApprovalURL != "" {
				// Render symmetric Approve / Deny pair. The deny URL
				// shares the JTI with approve (srenix-enterprise #17 symmetric
				// one-shot tokens) — whichever the SRE clicks first
				// wins, the other is burned. Denial records a RAG
				// outcome so the proposer learns from rejections.
				denyURL := strings.Replace(d.ApprovalURL, "/approve?", "/deny?", 1)
				fmt.Fprintf(&b, "  ✅ <%s|Approve> · ❌ <%s|Deny> · 📄 <%s&action=info|Details>\n",
					d.ApprovalURL, denyURL, d.ApprovalURL)
				// Phase 2.B.6 — class-action row. Hidden when class
				// URLs are empty (legacy memory-off deploy stays
				// byte-identical to pre-2.B Slack output).
				//
				// The class-scoped Silence link is rendered above by
				// renderSilenceSnippet (SilenceClassLongURL, configurable
				// duration). To keep EXACTLY ONE class silence link, the
				// legacy srenix-enterprise SilenceClassURL is only emitted here when
				// the OSS long link is absent — and labelled with the
				// configurable long duration, never the old hardcoded 7d.
				renderClassSilence := d.SilenceClassURL != "" && d.SilenceClassLongURL == ""
				if d.ApproveClassURL != "" || d.DenyClassURL != "" || renderClassSilence {
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
					if renderClassSilence {
						fmt.Fprintf(&b, "%s🔕 <%s|Silence class (%s)>", sep, d.SilenceClassURL, humanizeSilenceDuration(d.SilenceLongDur, DefaultSilenceLongDuration))
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
			renderResolvedRootCause(&b, r)
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
	footer := "K8s Agentic SRE — Watch mode"
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
