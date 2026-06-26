// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package report

import (
	"strings"
	"testing"
	"time"
)

// TestBuildActiveAlerts_ActionableFilter verifies that BuildActiveAlerts only
// forwards critical and actionable (has ApprovalURL) findings to Alertmanager.
// Purely advisory warnings with no ApprovalURL must be suppressed so they
// don't reach the cluster's Slack receiver with a misleading "Human Action
// Required" title.
func TestBuildActiveAlerts_ActionableFilter(t *testing.T) {
	ttl := 5 * time.Minute

	cases := []struct {
		name    string
		diag    DeltaDiag
		wantLen int
	}{
		{
			name: "advisory_warning_no_approval_url_excluded",
			diag: DeltaDiag{
				Subject:  "Probe/Storage/PVCPending",
				Severity: "warning",
				Message:  "2 PVCs in Pending state",
				// ApprovalURL intentionally empty — purely advisory
			},
			wantLen: 0,
		},
		{
			name: "warning_with_approval_url_included",
			diag: DeltaDiag{
				Subject:     "Probe/Critical Services/Service: foo",
				Severity:    "warning",
				Message:     "Deployment foo has 0 ready replicas",
				ApprovalURL: "https://approve.example.com/approve?token=abc",
			},
			wantLen: 1,
		},
		{
			name: "critical_no_approval_url_included",
			diag: DeltaDiag{
				Subject:  "Probe/Critical Services/Service: bar",
				Severity: "critical",
				Message:  "Service bar is completely down",
				// ApprovalURL empty — criticals always page regardless
			},
			wantLen: 1,
		},
		{
			name: "info_excluded",
			diag: DeltaDiag{
				Subject:  "Probe/Info/ClusterVersion",
				Severity: "info",
				Message:  "Cluster is running Kubernetes v1.29.0",
			},
			wantLen: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := BuildActiveAlerts([]DeltaDiag{tc.diag}, "test-cluster", ttl)
			if len(got) != tc.wantLen {
				t.Errorf("BuildActiveAlerts(%q sev=%q approvalURL=%q) returned %d alerts, want %d",
					tc.diag.Subject, tc.diag.Severity, tc.diag.ApprovalURL, len(got), tc.wantLen)
			}
			if tc.wantLen > 0 {
				lbl := got[0].Labels
				if lbl["severity"] != tc.diag.Severity {
					t.Errorf("alert label severity = %q, want %q", lbl["severity"], tc.diag.Severity)
				}
				if lbl["cluster"] != "test-cluster" {
					t.Errorf("alert label cluster = %q, want test-cluster", lbl["cluster"])
				}
			}
		})
	}
}

func TestAMRemediationLeadsWithRootCause(t *testing.T) {
	d := DeltaDiag{
		Subject:       "Probe/CrashLoopBackOff/CrashLoopBackOff",
		Severity:      "critical",
		Message:       "Pod ns/p in CrashLoopBackOff (75 restarts)",
		Remediation:   "Inspect crash cause: kubectl logs p -n ns --previous.",
		Investigation: "ns/p: container printed CLI usage/help and exited — no command/args.",
	}
	got := BuildActiveAlerts([]DeltaDiag{d}, "c", time.Minute)
	if len(got) != 1 {
		t.Fatalf("alerts=%d want 1", len(got))
	}
	ann := got[0].Annotations
	if ann["investigation"] != d.Investigation {
		t.Errorf("investigation annotation = %q", ann["investigation"])
	}
	rem := ann["remediation"]
	if !strings.HasPrefix(rem, "Root cause: ") {
		t.Errorf("remediation should LEAD with root cause; got %q", rem)
	}
	if !strings.Contains(rem, "printed CLI usage/help") {
		t.Errorf("remediation should contain the root cause; got %q", rem)
	}
	if !strings.Contains(rem, "Next steps:") {
		t.Errorf("remediation should still carry next steps; got %q", rem)
	}
}

func TestAMRemediationNoInvestigationUnchanged(t *testing.T) {
	d := DeltaDiag{Subject: "x", Severity: "critical", Message: "m", Remediation: "do the thing"}
	got := BuildActiveAlerts([]DeltaDiag{d}, "c", time.Minute)
	if got[0].Annotations["remediation"] != "do the thing" {
		t.Errorf("no-investigation remediation should be unchanged; got %q", got[0].Annotations["remediation"])
	}
	if _, ok := got[0].Annotations["investigation"]; ok {
		t.Errorf("investigation annotation should be absent when empty")
	}
}
