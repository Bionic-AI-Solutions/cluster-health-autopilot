// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package report

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

func TestFormatCriticalPayload_SilenceSnippetAlwaysRendered(t *testing.T) {
	payload := FormatCriticalPayload(
		[]DeltaDiag{
			{Subject: "Pod/web/example-1", Severity: "warning", Message: "image not pinned"},
			{Subject: "Probe/CrashLoopBackOff/x", Severity: "critical", Message: "5 restarts"},
		},
		nil,
	)
	body := payload.Attachments[0].Text
	if !strings.Contains(body, "🔕 silence 24h:") {
		t.Errorf("expected silence-snippet on every entry; got:\n%s", body)
	}
	if strings.Count(body, "kubectl apply -f -") != 2 {
		t.Errorf("expected 2 kubectl heredocs (one per entry); got:\n%s", body)
	}
	if !strings.Contains(body, `subject: "Probe/CrashLoopBackOff/x"`) {
		t.Errorf("silence matcher must include exact subject; got:\n%s", body)
	}
}

func TestRenderAIBlocks_ApprovalRendersApproveDenyPair(t *testing.T) {
	var b strings.Builder
	renderAIBlocks(&b, DeltaDiag{
		ApprovalURL: "https://cha-approve.example.com/approve?token=abc",
	})
	out := b.String()
	if !strings.Contains(out, "✅ <https://cha-approve.example.com/approve?token=abc|Approve>") {
		t.Errorf("expected Approve link; got:\n%s", out)
	}
	if !strings.Contains(out, "❌ <https://cha-approve.example.com/deny?token=abc|Deny>") {
		t.Errorf("expected symmetric Deny link with /approve? -> /deny? substitution; got:\n%s", out)
	}
	if strings.Contains(out, "Apply Fix") {
		t.Errorf("legacy 'Apply Fix' button must NOT be rendered; got:\n%s", out)
	}
}

func TestRenderAIBlocks_NoApprovalRendersNothing(t *testing.T) {
	var b strings.Builder
	renderAIBlocks(&b, DeltaDiag{})
	if b.Len() != 0 {
		t.Errorf("no AI fields should render nothing; got:\n%s", b.String())
	}
}

// TestSplitCriticalPayloads_ChunksToStayUnderSlackLimit verifies that
// 200 large-rendered findings (well over Slack's 40K char attachment cap)
// split into multiple payloads, each well under the limit, and that
// every finding makes it into exactly one chunk.
//
// Regression test for the 2026-06-04 outage where 118 findings × ~850 bytes
// rendered as one 115K payload — Slack silently truncated, alphabetically
// late findings (incl. storethesoup-missing-network-policy with a real
// Approve URL) were cut from the displayed message.
func TestSplitCriticalPayloads_ChunksToStayUnderSlackLimit(t *testing.T) {
	// Build 200 findings, each with a long-ish message + remediation so
	// each rendered finding is several hundred bytes. The exact size
	// doesn't matter — we just need to overflow the 35K chunk cap.
	const N = 200
	unfixable := make([]DeltaDiag, 0, N)
	for i := 0; i < N; i++ {
		unfixable = append(unfixable, DeltaDiag{
			Subject:     fmt.Sprintf("Pod/ns-%03d/workload-with-a-realistic-name-suffix", i),
			Severity:    "warning",
			Message:     strings.Repeat("a", 200) + " — synthetic message body for chunking test",
			Remediation: strings.Repeat("b", 200) + " — synthetic remediation body for chunking test",
		})
	}

	payloads := SplitCriticalPayloads(unfixable, nil)
	if len(payloads) < 2 {
		t.Fatalf("expected ≥ 2 chunks for 200 large findings; got %d", len(payloads))
	}

	// Every chunk must stay under Slack's safe limit.
	for i, p := range payloads {
		if len(p.Attachments) != 1 {
			t.Fatalf("chunk %d: expected 1 attachment; got %d", i, len(p.Attachments))
		}
		if got := len(p.Attachments[0].Text); got > maxSlackAttachmentChars {
			t.Errorf("chunk %d: text %d chars exceeds limit %d", i, got, maxSlackAttachmentChars)
		}
	}

	// Every finding must appear in exactly one chunk.
	seen := map[string]int{}
	for _, p := range payloads {
		text := p.Attachments[0].Text
		for i := 0; i < N; i++ {
			subj := fmt.Sprintf("Pod/ns-%03d/workload-with-a-realistic-name-suffix", i)
			if strings.Contains(text, subj) {
				seen[subj]++
			}
		}
	}
	if len(seen) != N {
		t.Errorf("expected all %d findings to appear in some chunk; got %d", N, len(seen))
	}
	for subj, count := range seen {
		if count != 1 {
			t.Errorf("finding %q appeared in %d chunks; want exactly 1", subj, count)
		}
	}

	// Every chunk after the first must carry a (part N/M) marker.
	if len(payloads) > 1 {
		for i, p := range payloads {
			marker := fmt.Sprintf("_(part %d/%d)_", i+1, len(payloads))
			if !strings.Contains(p.Attachments[0].Text, marker) {
				t.Errorf("chunk %d: missing %q marker; first 200 chars:\n%s", i, marker, p.Attachments[0].Text[:200])
			}
		}
	}
}

// TestSplitCriticalPayloads_SmallSetSingleChunk verifies the chunker
// degrades cleanly to a single payload when the rendered content fits.
func TestSplitCriticalPayloads_SmallSetSingleChunk(t *testing.T) {
	unfixable := []DeltaDiag{
		{Subject: "Pod/x/y", Severity: "warning", Message: "broken"},
		{Subject: "Pod/a/b", Severity: "critical", Message: "very broken"},
	}
	payloads := SplitCriticalPayloads(unfixable, nil)
	if len(payloads) != 1 {
		t.Errorf("small set should fit in 1 chunk; got %d", len(payloads))
	}
	if strings.Contains(payloads[0].Attachments[0].Text, "(part") {
		t.Errorf("single-chunk payload should not have (part N/M) marker")
	}
}

// TestRouteAndPost_ActionableFindingsBubbleToTop verifies that findings
// carrying an ApprovalURL sort ahead of findings without one, so the
// approvable Slack message lands in the inline-visible portion (Slack
// collapses long attachments at ~3-4K chars; a lone actionable item
// buried inside a 34K chunk of digest-pin noise renders below the
// fold even with chunking).
//
// Regression test for the 2026-06-04 UX bug where the storethesoup
// NetworkPolicy Approve/Deny line was in the message bytes but Slack
// only displayed the first dozen DNSChainDrift findings inline.
func TestRouteAndPost_ActionableFindingsBubbleToTop(t *testing.T) {
	// 100 noise findings + 2 with URLs, intentionally provided in
	// alphabetical order so a subject-only sort would NOT promote them.
	const N = 100
	unfixable := make([]DeltaDiag, 0, N+2)
	for i := 0; i < N; i++ {
		unfixable = append(unfixable, DeltaDiag{
			Subject:  fmt.Sprintf("Pod/aaa-%03d/noisy", i), // sorts alphabetically before "Pod/zzz-..."
			Severity: "warning",
			Message:  "digest pin missing",
		})
	}
	unfixable = append(unfixable,
		DeltaDiag{
			Subject:     "Pod/zzz-late/actionable",
			Severity:    "warning",
			Message:     "needs human review",
			ApprovalURL: "https://cha-approve.example.com/approve?token=A",
		},
		DeltaDiag{
			Subject:     "Pod/zzz-late2/actionable",
			Severity:    "warning",
			Message:     "needs human review",
			ApprovalURL: "https://cha-approve.example.com/approve?token=B",
		},
	)

	// Apply the same sort that RouteAndPost does.
	sort.Slice(unfixable, func(i, j int) bool {
		iHasURL := unfixable[i].ApprovalURL != ""
		jHasURL := unfixable[j].ApprovalURL != ""
		if iHasURL != jHasURL {
			return iHasURL
		}
		return unfixable[i].Subject < unfixable[j].Subject
	})

	payloads := SplitCriticalPayloads(unfixable, nil)
	if len(payloads) == 0 {
		t.Fatal("expected ≥ 1 payload")
	}

	// The first chunk's text must contain BOTH approvable subjects
	// before any of the noise ones — i.e. the actionable items appear
	// earlier in the text than the first noise subject.
	chunk1 := payloads[0].Attachments[0].Text
	firstActionable := strings.Index(chunk1, "Pod/zzz-late/actionable")
	firstNoise := strings.Index(chunk1, "Pod/aaa-000/noisy")
	if firstActionable < 0 {
		t.Fatalf("first chunk should contain actionable finding; got first 300 chars:\n%s", chunk1[:300])
	}
	if firstNoise >= 0 && firstActionable > firstNoise {
		t.Errorf("actionable finding should appear BEFORE noise; actionable@%d noise@%d", firstActionable, firstNoise)
	}
}

// --- Per-cycle delta render (Phase 1.E) ---
//
// Operators reading #ceph-critical can't tell at-a-glance which findings
// are new this cycle vs. stale-and-re-posted. With 50+ findings per
// digest, the "what should I look at right now" signal drowns. The
// DeltaDiag.IsNewThisCycle flag, set by the watcher's diff(), drives
// a "🆕 New this cycle (N)" section that renders BEFORE the steady-state
// section.

func TestSplitCriticalPayloads_NewThisCycleSectionAppears(t *testing.T) {
	// Two new + three stable critical findings. With Phase 1.E.5
	// stable-collapse, the new section appears in full and the stable
	// findings collapse to a single "…and 3 other stable finding(s)"
	// line so the new section's visibility isn't drowned.
	unfixable := []DeltaDiag{
		{Subject: "Pod/ns/stable-1", Severity: "critical", Message: "still broken"},
		{Subject: "Pod/ns/stable-2", Severity: "critical", Message: "still broken"},
		{Subject: "Pod/ns/new-a", Severity: "critical", Message: "just appeared", IsNewThisCycle: true},
		{Subject: "Pod/ns/stable-3", Severity: "critical", Message: "still broken"},
		{Subject: "Pod/ns/new-b", Severity: "critical", Message: "just appeared", IsNewThisCycle: true},
	}
	payloads := SplitCriticalPayloads(unfixable, nil)
	if len(payloads) == 0 {
		t.Fatal("expected ≥ 1 payload")
	}
	chunk1 := payloads[0].Attachments[0].Text
	if !strings.Contains(chunk1, "🆕 New this cycle (2)") {
		t.Errorf("missing '🆕 New this cycle (2)' header; got:\n%s", chunk1)
	}
	// Both new subjects must appear.
	if !strings.Contains(chunk1, "Pod/ns/new-a") || !strings.Contains(chunk1, "Pod/ns/new-b") {
		t.Errorf("both new-this-cycle subjects must appear; got:\n%s", chunk1)
	}
	// New subjects must appear BEFORE the stable summary line.
	idxNewA := strings.Index(chunk1, "Pod/ns/new-a")
	idxSummary := strings.Index(chunk1, "3 other stable finding")
	if idxSummary < 0 {
		t.Errorf("expected '3 other stable finding' summary; got:\n%s", chunk1)
	} else if idxNewA > idxSummary {
		t.Errorf("new-this-cycle finding should render BEFORE the stable summary; new-a@%d summary@%d", idxNewA, idxSummary)
	}
	// Stable subjects must NOT appear individually (they're collapsed).
	if strings.Contains(chunk1, "Pod/ns/stable-1") {
		t.Errorf("stable subjects should be collapsed when new exists; got:\n%s", chunk1)
	}
}

func TestSplitCriticalPayloads_AllStable_NoNewSection(t *testing.T) {
	// When nothing is new this cycle, the "🆕 New this cycle" section
	// must NOT appear at all — no zero-count clutter.
	unfixable := []DeltaDiag{
		{Subject: "Pod/ns/x", Severity: "critical", Message: "still broken"},
		{Subject: "Pod/ns/y", Severity: "warning", Message: "still broken"},
	}
	payloads := SplitCriticalPayloads(unfixable, nil)
	chunk1 := payloads[0].Attachments[0].Text
	if strings.Contains(chunk1, "🆕 New this cycle") {
		t.Errorf("no new findings should mean no '🆕 New this cycle' section; got:\n%s", chunk1)
	}
}

func TestSplitCriticalPayloads_AllNew_AllUnderNewSection(t *testing.T) {
	// When every finding is new, all of them belong in the new-section
	// and there should be no leftover stable-section header.
	unfixable := []DeltaDiag{
		{Subject: "Pod/ns/a", Severity: "critical", Message: "just appeared", IsNewThisCycle: true},
		{Subject: "Pod/ns/b", Severity: "warning", Message: "just appeared", IsNewThisCycle: true},
	}
	payloads := SplitCriticalPayloads(unfixable, nil)
	chunk1 := payloads[0].Attachments[0].Text
	if !strings.Contains(chunk1, "🆕 New this cycle (2)") {
		t.Errorf("expected '🆕 New this cycle (2)' header; got:\n%s", chunk1)
	}
	// The legacy "🔴 Critical (...)" + "⚠️ Diagnostics (...)" headers
	// should NOT appear independently when there are 0 stable findings —
	// they'd just show "(0)" which is clutter.
	if strings.Contains(chunk1, "Critical (0)") || strings.Contains(chunk1, "Diagnostics (0)") {
		t.Errorf("zero-count section headers should be suppressed; got:\n%s", chunk1)
	}
}

// --- Stable-collapse (Phase 1.E.4-5) ---
//
// When there ARE new findings this cycle, the steady-state list is
// noise the operator already saw last time. Collapse it to a one-liner
// "…and N other stable findings" so the new-section visibility isn't
// drowned. When there are NO new findings, render everything normally
// (the operator is looking at the full re-post, not the delta — every
// finding matters).

func TestSplitCriticalPayloads_StableCollapseWhenNewExists(t *testing.T) {
	// 2 new + 48 stable. Stable section should collapse to a single
	// summary line; the 48 subjects MUST NOT appear individually.
	unfixable := make([]DeltaDiag, 0, 50)
	unfixable = append(unfixable,
		DeltaDiag{Subject: "Pod/ns/new-a", Severity: "critical", Message: "just appeared", IsNewThisCycle: true},
		DeltaDiag{Subject: "Pod/ns/new-b", Severity: "warning", Message: "just appeared", IsNewThisCycle: true},
	)
	for i := 0; i < 48; i++ {
		unfixable = append(unfixable, DeltaDiag{
			Subject:  fmt.Sprintf("Pod/ns/stable-%02d", i),
			Severity: "critical",
			Message:  "long-running broken state",
		})
	}
	payloads := SplitCriticalPayloads(unfixable, nil)
	all := ""
	for _, p := range payloads {
		all += p.Attachments[0].Text
	}
	// Stable-collapse summary line MUST appear.
	if !strings.Contains(all, "48 other stable finding") {
		t.Errorf("expected stable-collapse summary 'N other stable finding'; got:\n%s", all[:min(2000, len(all))])
	}
	// Stable subjects must NOT appear individually.
	for i := 0; i < 48; i++ {
		needle := fmt.Sprintf("Pod/ns/stable-%02d", i)
		if strings.Contains(all, needle) {
			t.Errorf("stable subject %s should have been collapsed; rendered individually", needle)
		}
	}
	// New subjects MUST still appear individually.
	if !strings.Contains(all, "Pod/ns/new-a") || !strings.Contains(all, "Pod/ns/new-b") {
		t.Errorf("new-this-cycle subjects must still appear individually")
	}
}

func TestSplitCriticalPayloads_NoCollapseWhenNoNew(t *testing.T) {
	// 0 new + 50 stable. Stable findings must render fully — the
	// operator is reading the periodic re-post, not a delta.
	unfixable := make([]DeltaDiag, 0, 50)
	for i := 0; i < 50; i++ {
		unfixable = append(unfixable, DeltaDiag{
			Subject:  fmt.Sprintf("Pod/ns/stable-%02d", i),
			Severity: "critical",
			Message:  "broken",
		})
	}
	payloads := SplitCriticalPayloads(unfixable, nil)
	all := ""
	for _, p := range payloads {
		all += p.Attachments[0].Text
	}
	// No collapse line should be present.
	if strings.Contains(all, "other stable finding") {
		t.Errorf("must not collapse when there are no new findings; got summary line")
	}
	// All 50 subjects render.
	for i := 0; i < 50; i++ {
		needle := fmt.Sprintf("Pod/ns/stable-%02d", i)
		if !strings.Contains(all, needle) {
			t.Errorf("subject %s should render when nothing is new", needle)
		}
	}
}

// --- No-change digest (Phase 1.E.6-7) ---
//
// When new == 0, resolved == 0, and stable > 0, the watcher is going
// to post anyway (repeat-interval elapsed). Render a compact "✨ No
// new issues since last cycle (steady state at N findings)" digest
// instead of re-rendering the same 50 findings the operator saw an
// hour ago. Configurable via the new SuppressNoChangeRender option
// (default off — opt-in to preserve byte-identical behaviour for
// existing tests + deployments).

func TestSplitCriticalPayloadsConfig_NoChangeDigest_WhenEnabled(t *testing.T) {
	unfixable := make([]DeltaDiag, 0, 5)
	for i := 0; i < 5; i++ {
		unfixable = append(unfixable, DeltaDiag{
			Subject:  fmt.Sprintf("Pod/ns/stable-%02d", i),
			Severity: "critical",
			Message:  "broken",
		})
	}
	payloads := SplitCriticalPayloadsConfig(unfixable, nil, CriticalRenderConfig{NoChangeDigest: true})
	if len(payloads) != 1 {
		t.Fatalf("no-change digest is a single payload; got %d", len(payloads))
	}
	text := payloads[0].Attachments[0].Text
	if !strings.Contains(text, "✨ No new issues since last cycle") {
		t.Errorf("expected no-change digest header; got:\n%s", text)
	}
	if !strings.Contains(text, "steady state at 5 findings") {
		t.Errorf("expected steady-state count; got:\n%s", text)
	}
	// Individual subjects MUST NOT appear — the whole point is to elide
	// the re-rendered noise.
	for i := 0; i < 5; i++ {
		if strings.Contains(text, fmt.Sprintf("Pod/ns/stable-%02d", i)) {
			t.Errorf("no-change digest must not render individual subjects")
		}
	}
}

func TestSplitCriticalPayloadsConfig_NoChangeDigest_ReturnsNilOnZeroFindings(t *testing.T) {
	// 0 stable + 0 new + 0 resolved → no payload at all (skip the post).
	payloads := SplitCriticalPayloadsConfig(nil, nil, CriticalRenderConfig{NoChangeDigest: true})
	if len(payloads) != 0 {
		t.Errorf("no findings should yield no payload; got %d", len(payloads))
	}
}

func TestSplitCriticalPayloadsConfig_NoChangeDigest_FallsThroughWhenNewExists(t *testing.T) {
	// If anything IS new, even the no-change-digest config still renders
	// the full delta (because there IS a change).
	unfixable := []DeltaDiag{
		{Subject: "Pod/ns/new-a", Severity: "critical", Message: "broken", IsNewThisCycle: true},
		{Subject: "Pod/ns/stable-1", Severity: "critical", Message: "broken"},
	}
	payloads := SplitCriticalPayloadsConfig(unfixable, nil, CriticalRenderConfig{NoChangeDigest: true})
	if len(payloads) == 0 {
		t.Fatal("expected ≥ 1 payload")
	}
	text := payloads[0].Attachments[0].Text
	if strings.Contains(text, "✨ No new issues") {
		t.Errorf("must NOT emit no-change digest when new findings exist; got:\n%s", text)
	}
	if !strings.Contains(text, "Pod/ns/new-a") {
		t.Errorf("new finding must still render; got:\n%s", text)
	}
}

func TestSplitCriticalPayloadsConfig_NoChangeDigest_FallsThroughWhenResolvedExists(t *testing.T) {
	// Resolved transitions are also "changes" — even with no new
	// findings, render the resolved list normally rather than the
	// no-change digest.
	unfixable := []DeltaDiag{
		{Subject: "Pod/ns/stable-1", Severity: "critical", Message: "broken"},
	}
	resolved := []ResolvedDiag{{Subject: "Pod/ns/was-broken", Message: "now fine"}}
	payloads := SplitCriticalPayloadsConfig(unfixable, resolved, CriticalRenderConfig{NoChangeDigest: true})
	if len(payloads) == 0 {
		t.Fatal("expected ≥ 1 payload")
	}
	text := payloads[0].Attachments[0].Text
	if strings.Contains(text, "✨ No new issues") {
		t.Errorf("must NOT emit no-change digest when resolved exists; got:\n%s", text)
	}
	if !strings.Contains(text, "Pod/ns/was-broken") {
		t.Errorf("resolved finding must still render")
	}
}

// --- Conditional title (Part A of fix/advisory-alert-title) ---
//
// Slack alerts must use "Human Action Required" only when an Approve/Deny
// button is present. When ALL findings are purely advisory (no ApprovalURL),
// the title must be "CHA Advisory — Review (no action required)" so on-call
// engineers can triage at a glance without hunting for a non-existent button.

// TestSplitCriticalPayloads_AdvisoryTitleWhenNoActionable asserts that a
// payload containing ONLY advisory findings (no ApprovalURL on any finding)
// renders the softer "CHA Advisory" title, NOT "Human Action Required".
func TestSplitCriticalPayloads_AdvisoryTitleWhenNoActionable(t *testing.T) {
	unfixable := []DeltaDiag{
		{Subject: "ClusterRole/wildcard-role", Severity: "warning", Message: "grants wildcard verb"},
		{Subject: "ServiceAccount/ns/orphan", Severity: "warning", Message: "no RoleBinding"},
	}
	payloads := SplitCriticalPayloads(unfixable, nil)
	if len(payloads) == 0 {
		t.Fatal("expected ≥ 1 payload")
	}
	text := payloads[0].Attachments[0].Text
	if strings.Contains(text, "Human Action Required") {
		t.Errorf("advisory-only payload must NOT use 'Human Action Required' title; got:\n%s", text)
	}
	if !strings.Contains(text, "CHA Advisory") {
		t.Errorf("advisory-only payload must use 'CHA Advisory' title; got:\n%s", text)
	}
	if !strings.Contains(text, "no action required") {
		t.Errorf("advisory title must include 'no action required'; got:\n%s", text)
	}
}

// TestSplitCriticalPayloads_ActionableTitleWhenAnyActionable asserts that a
// payload containing even ONE finding with an ApprovalURL keeps the
// "Human Action Required" title — the operator must click Approve or Deny.
func TestSplitCriticalPayloads_ActionableTitleWhenAnyActionable(t *testing.T) {
	unfixable := []DeltaDiag{
		{Subject: "ClusterRole/wildcard-role", Severity: "warning", Message: "grants wildcard verb"},
		{
			Subject:     "NetworkPolicy/ns/missing",
			Severity:    "warning",
			Message:     "missing egress rule",
			ApprovalURL: "https://cha-approve.example.com/approve?token=abc",
		},
	}
	payloads := SplitCriticalPayloads(unfixable, nil)
	if len(payloads) == 0 {
		t.Fatal("expected ≥ 1 payload")
	}
	text := payloads[0].Attachments[0].Text
	if !strings.Contains(text, "Human Action Required") {
		t.Errorf("payload with actionable finding must use 'Human Action Required' title; got:\n%s", text)
	}
	if strings.Contains(text, "CHA Advisory") {
		t.Errorf("payload with actionable finding must NOT use 'CHA Advisory' title; got:\n%s", text)
	}
}

// TestSplitCriticalPayloads_MixedFindings_HumanActionRequired asserts that
// a mix of advisory and actionable findings → "Human Action Required" title.
func TestSplitCriticalPayloads_MixedFindings_HumanActionRequired(t *testing.T) {
	unfixable := []DeltaDiag{
		{Subject: "Advisory/only/one", Severity: "warning", Message: "advisory"},
		{Subject: "Advisory/only/two", Severity: "critical", Message: "advisory critical"},
		{Subject: "Actionable/one", Severity: "warning", Message: "needs human",
			ApprovalURL: "https://cha-approve.example.com/approve?token=xyz"},
	}
	payloads := SplitCriticalPayloads(unfixable, nil)
	if len(payloads) == 0 {
		t.Fatal("expected ≥ 1 payload")
	}
	text := payloads[0].Attachments[0].Text
	if !strings.Contains(text, "Human Action Required") {
		t.Errorf("mixed payload must use 'Human Action Required'; got:\n%s", text)
	}
}

// TestFormatCriticalPayload_AdvisoryTitle asserts the single-payload legacy
// renderer also uses the advisory title when no findings are actionable.
func TestFormatCriticalPayload_AdvisoryTitle(t *testing.T) {
	payload := FormatCriticalPayload(
		[]DeltaDiag{
			{Subject: "ClusterRole/wildcard-role", Severity: "warning", Message: "wildcard verb"},
		},
		nil,
	)
	text := payload.Attachments[0].Text
	if strings.Contains(text, "Human Action Required") {
		t.Errorf("advisory-only legacy payload must NOT use 'Human Action Required'; got:\n%s", text)
	}
	if !strings.Contains(text, "CHA Advisory") {
		t.Errorf("advisory-only legacy payload must use 'CHA Advisory'; got:\n%s", text)
	}
}

// TestFormatCriticalPayload_ActionableTitle asserts the single-payload
// renderer uses "Human Action Required" when an ApprovalURL is present.
func TestFormatCriticalPayload_ActionableTitle(t *testing.T) {
	payload := FormatCriticalPayload(
		[]DeltaDiag{
			{Subject: "Advisory/one", Severity: "warning", Message: "advisory"},
			{Subject: "Actionable/one", Severity: "warning", Message: "needs human",
				ApprovalURL: "https://cha-approve.example.com/approve?token=tok"},
		},
		nil,
	)
	text := payload.Attachments[0].Text
	if !strings.Contains(text, "Human Action Required") {
		t.Errorf("payload with actionable finding must use 'Human Action Required'; got:\n%s", text)
	}
}

// TestEmitNoChangeDigest_AdvisoryTitle asserts the no-change digest always
// uses the advisory title (there are never Approve/Deny buttons in a digest).
func TestEmitNoChangeDigest_AdvisoryTitle(t *testing.T) {
	// Five stable findings with no actionable items → emitNoChangeDigest path.
	unfixable := make([]DeltaDiag, 5)
	for i := range unfixable {
		unfixable[i] = DeltaDiag{Subject: fmt.Sprintf("Pod/ns/stable-%d", i), Severity: "warning", Message: "broken"}
	}
	payloads := SplitCriticalPayloadsConfig(unfixable, nil, CriticalRenderConfig{NoChangeDigest: true})
	if len(payloads) != 1 {
		t.Fatalf("no-change digest must yield 1 payload; got %d", len(payloads))
	}
	text := payloads[0].Attachments[0].Text
	if strings.Contains(text, "Human Action Required") {
		t.Errorf("no-change digest must NOT use 'Human Action Required'; got:\n%s", text)
	}
	if !strings.Contains(text, "CHA Advisory") {
		t.Errorf("no-change digest must use 'CHA Advisory'; got:\n%s", text)
	}
}

// TestSplitCriticalPayloads_PartMarkerPreservesConditionalTitle asserts that
// multi-chunk payloads keep the correct conditional title on every chunk
// (including the (part N/M) marker format).
func TestSplitCriticalPayloads_PartMarkerPreservesConditionalTitle(t *testing.T) {
	// Build enough advisory findings to force ≥2 chunks.
	const N = 200
	unfixable := make([]DeltaDiag, 0, N)
	for i := 0; i < N; i++ {
		unfixable = append(unfixable, DeltaDiag{
			Subject:     fmt.Sprintf("ClusterRole/rbac-drift-%03d", i),
			Severity:    "warning",
			Message:     strings.Repeat("advisory finding detail ", 10),
			Remediation: strings.Repeat("remediation text here ", 10),
		})
	}
	payloads := SplitCriticalPayloads(unfixable, nil)
	if len(payloads) < 2 {
		t.Fatalf("expected ≥2 chunks for %d findings; got %d", N, len(payloads))
	}
	for i, p := range payloads {
		text := p.Attachments[0].Text
		if strings.Contains(text, "Human Action Required") {
			t.Errorf("chunk %d: advisory-only payload must NOT use 'Human Action Required'; got first 300:\n%s", i, text[:min(300, len(text))])
		}
		if !strings.Contains(text, "CHA Advisory") {
			t.Errorf("chunk %d: advisory-only payload must use 'CHA Advisory'; got first 300:\n%s", i, text[:min(300, len(text))])
		}
		// Part marker must still appear on multi-chunk payloads.
		marker := fmt.Sprintf("_(part %d/%d)_", i+1, len(payloads))
		if !strings.Contains(text, marker) {
			t.Errorf("chunk %d: missing part marker %q; text first 300:\n%s", i, marker, text[:min(300, len(text))])
		}
	}
}
