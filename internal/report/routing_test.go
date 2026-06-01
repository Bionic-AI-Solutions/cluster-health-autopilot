// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package report

import (
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
