// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package report

import (
	"fmt"
	"log"
	"net/http"
	"sort"
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

// renderAIBlocks appends optional AI-tier blocks (enrichment, approval URL)
// to a Slack message builder. Renders nothing when no AI fields are populated
// — OSS deployments produce identical output to today.
func renderAIBlocks(b *strings.Builder, d DeltaDiag) {
	if d.Enrichment != "" {
		fmt.Fprintf(b, "  🤖 _%s_\n", d.Enrichment)
	}
	if d.ApprovalURL != "" {
		// Symmetric Approve / Deny pair (cha-com #17 one-shot tokens).
		// Whichever the SRE clicks first wins, the other is burned.
		// Denial records a RAG outcome so the proposer learns from
		// rejections.
		denyURL := strings.Replace(d.ApprovalURL, "/approve?", "/deny?", 1)
		fmt.Fprintf(b, "  ✅ <%s|Approve> · ❌ <%s|Deny> · 📄 <%s&action=info|Details>\n",
			d.ApprovalURL, denyURL, d.ApprovalURL)
	}
}

// renderSilenceSnippet emits the inline kubectl heredoc that creates a
// Silence CR for THIS finding's subject for 24h. Lets SREs distinguish
// "fix this" from "noisy, mute" without hunting for the CRD schema.
// Subject-scoped (exact match); edit spec.matcher to `source` for
// class-wide suppression. Mirrors what `slack_delta.go` does for the
// watcher delta path so both production renderers stay in sync.
func renderSilenceSnippet(b *strings.Builder, d DeltaDiag) {
	fmt.Fprintf(b, "  🔕 silence 24h: ```kubectl apply -f - <<EOF\n"+
		"apiVersion: cha.bionicaisolutions.com/v1alpha1\n"+
		"kind: Silence\nmetadata:\n  name: %s\n  namespace: cluster-health-autopilot\n"+
		"spec:\n  matcher:\n    subject: %q\n  until: %q\n  reason: silenced-from-slack\nEOF```\n",
		slackSilenceName(d.Subject), d.Subject,
		time.Now().UTC().Add(24*time.Hour).Format("2006-01-02T15:04:05Z"))
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
			renderAIBlocks(&b, d)
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

// maxSlackAttachmentChars is the safe cap before Slack silently truncates
// the "text" field of an incoming-webhook attachment. The documented limit
// is 40,000 characters; we leave a margin so the renderer's header +
// per-finding footer text always fits.
const maxSlackAttachmentChars = 35000

// SplitCriticalPayloads renders the unfixable/resolved set into one or
// more SlackPayloads, chunking findings so each payload's attachment
// text stays under maxSlackAttachmentChars.
//
// Without this, posts with >40K rendered chars (≈ 40 digest-pin findings)
// were silently truncated by Slack — alphabetically-late findings + their
// "✅ Approve" links never reached the channel even though the OSS render
// included them correctly. (Discovered 2026-06-04 with a 115K render.)
//
// The first chunk carries the "*CHA Alert — Human Action Required*"
// header and any critical findings; subsequent chunks add a "(part N/M)"
// indicator and continue with diagnostics. Resolved findings stay in the
// last chunk (they're typically small).
func SplitCriticalPayloads(unfixable []DeltaDiag, resolved []ResolvedDiag) []SlackPayload {
	// Render each finding to its own string so we can group greedily.
	var critRendered, diagRendered []string
	for _, d := range unfixable {
		var b strings.Builder
		if d.Severity == "critical" {
			fmt.Fprintf(&b, "• ❌ *%s*\n  %s\n", d.Subject, d.Message)
		} else {
			fmt.Fprintf(&b, "• ⚠️ *%s*\n  %s\n", d.Subject, d.Message)
		}
		if d.Remediation != "" {
			fmt.Fprintf(&b, "  _→ %s_\n", d.Remediation)
		}
		renderSilenceSnippet(&b, d)
		renderAIBlocks(&b, d)
		if d.Severity == "critical" {
			critRendered = append(critRendered, b.String())
		} else {
			diagRendered = append(diagRendered, b.String())
		}
	}

	// Resolved section: small, kept whole.
	var resolvedSection string
	if len(resolved) > 0 {
		var b strings.Builder
		fmt.Fprintf(&b, "\n*✅ Resolved (%d):*\n", len(resolved))
		for _, r := range resolved {
			fmt.Fprintf(&b, "• `%s`\n", r.Subject)
			if r.Message != "" {
				fmt.Fprintf(&b, "  _%s_\n", r.Message)
			}
		}
		resolvedSection = b.String()
	}

	headerLine := fmt.Sprintf("*CHA Alert — Human Action Required* — %s\n", time.Now().UTC().Format("2006-01-02 15:04:05 UTC"))
	critHeader := fmt.Sprintf("\n*🔴 Critical (%d):*\n", len(critRendered))
	diagHeader := fmt.Sprintf("\n*⚠️ Diagnostics (%d):*\n", len(diagRendered))

	// Stream findings into chunks. Each chunk begins with the global
	// header + (when it's the first chunk) any critical findings;
	// subsequent chunks continue diagnostics with a (part N/M) note.
	var chunks []string
	var cur strings.Builder
	cur.WriteString(headerLine)
	if len(critRendered) > 0 {
		cur.WriteString(critHeader)
		for _, r := range critRendered {
			if cur.Len()+len(r) > maxSlackAttachmentChars {
				chunks = append(chunks, cur.String())
				cur.Reset()
				cur.WriteString(headerLine)
				cur.WriteString(critHeader)
			}
			cur.WriteString(r)
		}
	}
	if len(diagRendered) > 0 {
		// Start the diagnostics header inside the current chunk if
		// there's headroom; otherwise flush + start a new chunk.
		if cur.Len()+len(diagHeader) > maxSlackAttachmentChars {
			chunks = append(chunks, cur.String())
			cur.Reset()
			cur.WriteString(headerLine)
		}
		cur.WriteString(diagHeader)
		for _, r := range diagRendered {
			if cur.Len()+len(r) > maxSlackAttachmentChars {
				chunks = append(chunks, cur.String())
				cur.Reset()
				cur.WriteString(headerLine)
				cur.WriteString(diagHeader)
			}
			cur.WriteString(r)
		}
	}
	if resolvedSection != "" {
		if cur.Len()+len(resolvedSection) > maxSlackAttachmentChars {
			chunks = append(chunks, cur.String())
			cur.Reset()
			cur.WriteString(headerLine)
		}
		cur.WriteString(resolvedSection)
	}
	if cur.Len() > 0 {
		chunks = append(chunks, cur.String())
	}

	// Annotate (part N/M) when there's more than one.
	if len(chunks) > 1 {
		for i := range chunks {
			marker := fmt.Sprintf(" _(part %d/%d)_", i+1, len(chunks))
			// Insert the marker right after the first newline (just
			// after the header line) so it stays adjacent to the
			// "CHA Alert" title.
			if nl := strings.Index(chunks[i], "\n"); nl > 0 {
				chunks[i] = chunks[i][:nl] + marker + chunks[i][nl:]
			}
		}
	}

	// Pick color per chunk: danger if any critical, warning if only
	// diagnostics, good if neither (resolved-only).
	color := "danger"
	if len(critRendered) == 0 && len(diagRendered) > 0 {
		color = "warning"
	}
	if len(critRendered) == 0 && len(diagRendered) == 0 {
		color = "good"
	}

	now := time.Now().UTC().Unix()
	out := make([]SlackPayload, 0, len(chunks))
	for _, text := range chunks {
		out = append(out, SlackPayload{
			Username:  "Cluster Health Monitor",
			IconEmoji: ":rotating_light:",
			Attachments: []SlackAttachment{{
				Color:    color,
				Text:     text,
				Footer:   "CHA — Human action required",
				Ts:       now,
				MrkdwnIn: []string{"text"},
			}},
		})
	}
	return out
}

// FormatCriticalPayload renders the #ceph-critical message for a watcher cycle
// where issues require human intervention — either unfixable by CHA or still
// active after fixers ran.
//
// Retained for callers that want a single payload (tests, OSS examples).
// New code should prefer SplitCriticalPayloads which avoids Slack's
// silent 40K-char truncation.
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
			renderSilenceSnippet(&b, d)
			renderAIBlocks(&b, d)
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
			renderSilenceSnippet(&b, d)
			renderAIBlocks(&b, d)
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
		urlCount := 0
		for _, d := range fixedIssues {
			if d.ApprovalURL != "" {
				urlCount++
			}
		}
		log.Printf("report: posting to alerts channel: fixedIssues=%d with_url=%d fixerActions=%d", len(fixedIssues), urlCount, len(fixResults))
		if err := PostSlack(client, channels.Alerts, payload); err != nil {
			log.Printf("report: slack post to alerts channel: %v", err)
		}
	}

	if channels.Critical != "" && (len(unfixable) > 0 || len(toResolve) > 0) {
		// Sort unfixable so actionable findings (those carrying a signed
		// ApprovalURL) surface first within each severity block. Slack's
		// UI auto-collapses long attachments to a ~3-4K inline preview
		// with a "Show more" expand; without this sort the lone
		// approvable finding can land below the fold inside a 34K chunk
		// of digest-pin noise. Alphabetical-by-subject is the
		// secondary key so cycles remain deterministic.
		sort.Slice(unfixable, func(i, j int) bool {
			iHasURL := unfixable[i].ApprovalURL != ""
			jHasURL := unfixable[j].ApprovalURL != ""
			if iHasURL != jHasURL {
				return iHasURL
			}
			return unfixable[i].Subject < unfixable[j].Subject
		})
		payloads := SplitCriticalPayloads(unfixable, toResolve)
		urlCount := 0
		for _, d := range unfixable {
			if d.ApprovalURL != "" {
				urlCount++
			}
		}
		log.Printf("report: posting to critical channel: unfixable=%d with_url=%d resolved=%d chunks=%d",
			len(unfixable), urlCount, len(toResolve), len(payloads))
		for i, payload := range payloads {
			if err := PostSlack(client, channels.Critical, payload); err != nil {
				log.Printf("report: slack post to critical channel (chunk %d/%d): %v", i+1, len(payloads), err)
			}
		}
	}
}
