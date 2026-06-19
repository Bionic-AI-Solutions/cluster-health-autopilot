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

// renderSilenceSnippet emits the silence affordance for ONE finding.
//
// Two modes, gated on whether the watcher minted signed one-click links:
//
//   - CONFIGURED (signer + approval base URL present): renders the two
//     click-links — "🔕 Silence <short>" (subject-scoped snooze) and
//     "🔕 Silence class (<long>)" (Source-scoped mute). One click hits the
//     approval-server's /silence endpoint, which materializes a real
//     Silence CR. No CRD schema to memorize, no terminal needed.
//
//   - UNCONFIGURED (OSS-only / air-gapped — no key, no approval-server):
//     falls back to the inline kubectl heredoc that creates a 24h
//     subject-scoped Silence CR. The air-gapped affordance is preserved.
//
// Subject-scoped (exact match); edit spec.matcher to `source` for
// class-wide suppression. Mirrors what `slack_delta.go` does for the
// FormatSlackDelta renderer so both production renderers stay in sync.
func renderSilenceSnippet(b *strings.Builder, d DeltaDiag) {
	if d.SilenceSubjectURL != "" || d.SilenceClassLongURL != "" {
		renderSilenceClickLinks(b, d)
		return
	}
	fmt.Fprintf(b, "  🔕 silence 24h: ```kubectl apply -f - <<EOF\n"+
		"apiVersion: cha.bionicaisolutions.com/v1alpha1\n"+
		"kind: Silence\nmetadata:\n  name: %s\n  namespace: cluster-health-autopilot\n"+
		"spec:\n  matcher:\n    subject: %q\n  until: %q\n  reason: silenced-from-slack\nEOF```\n",
		slackSilenceName(d.Subject), d.Subject,
		time.Now().UTC().Add(24*time.Hour).Format("2006-01-02T15:04:05Z"))
}

// renderSilenceClickLinks emits the "🔕 <url|Silence 24h> · 🔕 <url|Silence
// class (90d)>" row, labelling each link with its actual configured
// duration. Either link may be absent (gated independently).
func renderSilenceClickLinks(b *strings.Builder, d DeltaDiag) {
	b.WriteString("  ")
	sep := ""
	if d.SilenceSubjectURL != "" {
		fmt.Fprintf(b, "🔕 <%s|Silence %s>", d.SilenceSubjectURL, humanizeSilenceDuration(d.SilenceShortDur, DefaultSilenceShortDuration))
		sep = " · "
	}
	if d.SilenceClassLongURL != "" {
		fmt.Fprintf(b, "%s🔕 <%s|Silence class (%s)>", sep, d.SilenceClassLongURL, humanizeSilenceDuration(d.SilenceLongDur, DefaultSilenceLongDuration))
	}
	b.WriteString("\n")
}

// humanizeSilenceDuration renders a silence window as a compact label
// ("24h", "90d", "12h"). Falls back to def when dur <= 0. Multi-day whole
// windows (≥ 48h) collapse to "Nd"; a single day stays "24h" (matches the
// canonical "Silence 24h" affordance); other whole hours render "Nh";
// sub-hour windows keep the stdlib form.
func humanizeSilenceDuration(dur, def time.Duration) string {
	if dur <= 0 {
		dur = def
	}
	if dur >= 48*time.Hour && dur%(24*time.Hour) == 0 {
		return fmt.Sprintf("%dd", dur/(24*time.Hour))
	}
	if dur%time.Hour == 0 {
		return fmt.Sprintf("%dh", dur/time.Hour)
	}
	return dur.String()
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

// CriticalRenderConfig is the typed knob bag for SplitCriticalPayloadsConfig.
// Zero value = identical to legacy SplitCriticalPayloads behaviour, so
// every existing caller is byte-equivalent.
type CriticalRenderConfig struct {
	// NoChangeDigest, when true AND there are zero new findings AND
	// zero resolved transitions, replaces the full re-render of all
	// stable findings with a compact "✨ No new issues since last
	// cycle (steady state at N findings)" digest. Default false to
	// preserve legacy behaviour — opt in when the operator wants the
	// signal-to-noise win on quiet cycles.
	NoChangeDigest bool
}

// SplitCriticalPayloads is the legacy entry point. Renders the same as
// SplitCriticalPayloadsConfig with a zero CriticalRenderConfig. Kept so
// existing callers compile + behave identically.
//
// Without the chunker, posts with >40K rendered chars (≈ 40 digest-pin
// findings) were silently truncated by Slack — alphabetically-late
// findings + their "✅ Approve" links never reached the channel even
// though the OSS render included them correctly. The chunker carries
// the "*CHA Alert — Human Action Required*" header on every chunk and
// adds a "(part N/M)" indicator when more than one chunk results.
// Resolved findings stay in the last chunk (they're typically small).
func SplitCriticalPayloads(unfixable []DeltaDiag, resolved []ResolvedDiag) []SlackPayload {
	return SplitCriticalPayloadsConfig(unfixable, resolved, CriticalRenderConfig{})
}

// SplitCriticalPayloadsConfig is the configurable variant.
//
// Phase 1.E layering, top-down:
//  1. NoChangeDigest mode (opt-in): empty new + empty resolved + N
//     stable → ONE payload with a "✨ No new issues since last cycle"
//     digest. Zero findings entirely → no payload (skip the post).
//  2. "🆕 New this cycle (N):" section ALWAYS renders first when any
//     IsNewThisCycle finding is present.
//  3. Stable-collapse: when new findings exist, the steady-state list
//     collapses to a one-line "…and N other stable findings" summary
//     so the new-section visibility isn't drowned. When no new findings
//     exist, stable renders in full (operator is reading the periodic
//     re-post, not a delta — every finding matters).
//  4. The legacy "🔴 Critical / ⚠️ Diagnostics" sections render only
//     the stable subset when a stable-collapse is NOT in effect.
func SplitCriticalPayloadsConfig(unfixable []DeltaDiag, resolved []ResolvedDiag, cfg CriticalRenderConfig) []SlackPayload {
	// Render each finding to its own string so we can group greedily.
	// Phase 1.E: also partition by IsNewThisCycle so the renderer can
	// surface a "🆕 New this cycle" section ABOVE the steady-state list.
	var critRendered, diagRendered []string
	var newRendered []string
	var newCount int
	var stableCritCount, stableDiagCount int
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
		if d.IsNewThisCycle {
			newRendered = append(newRendered, b.String())
			newCount++
			continue
		}
		if d.Severity == "critical" {
			critRendered = append(critRendered, b.String())
			stableCritCount++
		} else {
			diagRendered = append(diagRendered, b.String())
			stableDiagCount++
		}
	}

	// No-change digest: opted in via cfg, and nothing has changed since
	// the last cycle. Return either a compact digest or nothing at all.
	if cfg.NoChangeDigest && newCount == 0 && len(resolved) == 0 {
		stableTotal := stableCritCount + stableDiagCount
		if stableTotal == 0 {
			// Truly nothing to say. Don't post.
			return nil
		}
		return emitNoChangeDigest(stableTotal)
	}

	// Stable-collapse: when new findings exist, the steady-state list
	// becomes a 1-line summary so the new-section visibility is preserved.
	// When no new findings exist, render stable in full (this is the
	// re-post path; operators need the complete list).
	var stableSummary string
	if newCount > 0 && (stableCritCount+stableDiagCount) > 0 {
		stableSummary = fmt.Sprintf(
			"_…and %d other stable finding%s already posted in earlier cycles._\n",
			stableCritCount+stableDiagCount,
			plural(stableCritCount+stableDiagCount),
		)
		critRendered = nil
		diagRendered = nil
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

	headerLine := fmt.Sprintf("%s — %s\n", alertTitle(hasActionableFindings(unfixable)), time.Now().UTC().Format("2006-01-02 15:04:05 UTC"))
	newHeader := fmt.Sprintf("\n*🆕 New this cycle (%d):*\n", newCount)
	critHeader := fmt.Sprintf("\n*🔴 Critical (%d):*\n", len(critRendered))
	diagHeader := fmt.Sprintf("\n*⚠️ Diagnostics (%d):*\n", len(diagRendered))

	// Stream findings into chunks. Each chunk begins with the global
	// header + (when it's the first chunk) any new-this-cycle findings,
	// then the steady-state critical + diagnostics list. Subsequent
	// chunks add a (part N/M) marker.
	var chunks []string
	var cur strings.Builder
	cur.WriteString(headerLine)
	if newCount > 0 {
		cur.WriteString(newHeader)
		for _, r := range newRendered {
			if cur.Len()+len(r) > maxSlackAttachmentChars {
				chunks = append(chunks, cur.String())
				cur.Reset()
				cur.WriteString(headerLine)
				cur.WriteString(newHeader)
			}
			cur.WriteString(r)
		}
	}
	if len(critRendered) > 0 {
		if cur.Len()+len(critHeader) > maxSlackAttachmentChars {
			chunks = append(chunks, cur.String())
			cur.Reset()
			cur.WriteString(headerLine)
		}
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
	if stableSummary != "" {
		if cur.Len()+len(stableSummary) > maxSlackAttachmentChars {
			chunks = append(chunks, cur.String())
			cur.Reset()
			cur.WriteString(headerLine)
		}
		cur.WriteString("\n")
		cur.WriteString(stableSummary)
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

// emitNoChangeDigest renders the single-payload "✨ No new issues since
// last cycle" digest used when SplitCriticalPayloadsConfig is in
// NoChangeDigest mode and nothing has changed since the prior cycle.
// The header timestamp lets operators confirm at-a-glance that the
// watcher is alive (the alternative — silently skipping the post —
// is indistinguishable from a crashed agent).
func emitNoChangeDigest(stableTotal int) []SlackPayload {
	// No-change digest: nothing new, nothing resolved → purely advisory.
	// There is no Approve/Deny button here; "Human Action Required" would
	// be misleading. Use the advisory title unconditionally.
	text := fmt.Sprintf(
		"%s — %s\n\n"+
			"*✨ No new issues since last cycle* — steady state at %d finding%s. "+
			"_Run `cha diagnose` or check #ceph-critical history for the active list._\n",
		alertTitle(false),
		time.Now().UTC().Format("2006-01-02 15:04:05 UTC"),
		stableTotal, plural(stableTotal),
	)
	return []SlackPayload{{
		Username:  "Cluster Health Monitor",
		IconEmoji: ":rotating_light:",
		Attachments: []SlackAttachment{{
			Color:    "warning", // still problems — just no NEW problems
			Text:     text,
			Footer:   "CHA — Human action required",
			Ts:       time.Now().UTC().Unix(),
			MrkdwnIn: []string{"text"},
		}},
	}}
}

// plural returns "s" when n != 1, empty otherwise. Used by the
// digest/summary lines so "1 finding" stays grammatical without
// special-casing each call site.
func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// hasActionableFindings reports whether any finding in ds carries a signed
// ApprovalURL — i.e. an Approve/Deny button will appear in the Slack post.
// Used to choose between "Human Action Required" (approve/deny buttons
// present) and "Advisory — Review (no action required)" (purely informational
// findings with no interactive controls) as the Slack alert title.
func hasActionableFindings(ds []DeltaDiag) bool {
	for _, d := range ds {
		if d.ApprovalURL != "" {
			return true
		}
	}
	return false
}

// alertTitle returns the Slack header string appropriate for the given finding
// set. When any finding is actionable (carries an ApprovalURL) the operator
// must choose Approve/Deny — use the "Human Action Required" title. When ALL
// findings are purely advisory (no approve/deny buttons) use the softer
// "Advisory — Review" title so on-call engineers can triage at a glance.
func alertTitle(actionable bool) string {
	if actionable {
		return "*CHA Alert — Human Action Required*"
	}
	return "*CHA Advisory — Review (no action required)*"
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

	fmt.Fprintf(&b, "%s — %s\n", alertTitle(hasActionableFindings(unfixable)), now.Format("2006-01-02 15:04:05 UTC"))

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
	RouteAndPostConfig(client, channels, postFixSubjects, toPost, toResolve, fixResults, CriticalRenderConfig{})
}

// RouteAndPostConfig is the configurable variant. Same behaviour as
// RouteAndPost with the addition of cfg, which currently controls the
// no-change digest opt-in. Default-value cfg = identical to RouteAndPost.
func RouteAndPostConfig(
	client *http.Client,
	channels SlackChannels,
	postFixSubjects map[string]bool,
	toPost []DeltaDiag,
	toResolve []ResolvedDiag,
	fixResults []fix.Result,
	cfg CriticalRenderConfig,
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
		payloads := SplitCriticalPayloadsConfig(unfixable, toResolve, cfg)
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
