// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package report

import (
	"strings"
	"testing"

	"github.com/srenix-ai/agentic-sre/pkg/fix"
)

func TestFormatSlackDelta_ClassButtons_AllPresent(t *testing.T) {
	d := DeltaDiag{
		Subject:         "Pod/prod/srenix-enterprise-xyz",
		Severity:        "warning",
		Message:         "Pod prod/srenix-enterprise mounts container image(s) without digest pin",
		IsNewThisCycle:  true,
		ApprovalURL:     "https://approve.example.com/approve?token=A",
		ApproveClassURL: "https://approve.example.com/approve-class?token=B",
		DenyClassURL:    "https://approve.example.com/deny-class?token=C",
		SilenceClassURL: "https://approve.example.com/silence-class?token=D",
	}
	out := FormatSlackDelta([]DeltaDiag{d}, nil, []fix.Result{}, false)
	body := out.Attachments[0].Text
	for _, w := range []string{
		"✅ <https://approve.example.com/approve?token=A|Approve>",
		"❌ <https://approve.example.com/deny?token=A|Deny>", // existing deny URL derived from approve
		"🧠 <https://approve.example.com/approve-class?token=B|Approve+remember class>",
		"❌ <https://approve.example.com/deny-class?token=C|Deny+remember class>",
		"🔕 <https://approve.example.com/silence-class?token=D|Silence class (90d)>",
	} {
		if !strings.Contains(body, w) {
			t.Errorf("payload missing %q\n--- body ---\n%s", w, body)
		}
	}
}

func TestFormatSlackDelta_ClassButtons_OmittedWhenEmpty(t *testing.T) {
	// Legacy memory-off deploy: ApprovalURL set but class URLs empty.
	// Output stays byte-identical to pre-2.B (no 🧠/🔕 lines).
	d := DeltaDiag{
		Subject:        "Pod/prod/x",
		Severity:       "warning",
		Message:        "anything",
		IsNewThisCycle: true,
		ApprovalURL:    "https://approve.example.com/approve?token=A",
	}
	out := FormatSlackDelta([]DeltaDiag{d}, nil, []fix.Result{}, false)
	body := out.Attachments[0].Text
	// 🔕 silence-24h IS rendered by the existing legacy per-subject
	// silence helper — don't reject that. Only the new Phase 2.B
	// class buttons should be absent.
	for _, banned := range []string{"🧠", "Approve+remember class", "Deny+remember class", "Silence class (7d)"} {
		if strings.Contains(body, banned) {
			t.Errorf("legacy deploy must NOT render class buttons; found %q\n--- body ---\n%s", banned, body)
		}
	}
}
