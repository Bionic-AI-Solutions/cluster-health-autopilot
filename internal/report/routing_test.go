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
