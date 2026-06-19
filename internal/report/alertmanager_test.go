// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package report

import (
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
