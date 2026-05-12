// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package audit

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/ai"
)

func TestEventsSink_Write_Normal(t *testing.T) {
	client := fake.NewClientset()
	s := NewEventsSink(client, "cluster-health-autopilot")
	ctx := context.Background()

	err := s.Write(ctx, ai.AuditEvent{
		Type:          "ai.proposal.created",
		CorrelationID: "act-test-1",
		Tier:          ai.TierT1,
		Actor:         "cha-com",
		Subject:       "Pod/default/demo-abc",
		Details: map[string]any{
			"action_kind": "DeletePod",
			"target":      "Pod/default/demo-abc",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	evs, err := client.CoreV1().Events("cluster-health-autopilot").List(ctx, metav1.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(evs.Items) != 1 {
		t.Fatalf("got %d events; want 1", len(evs.Items))
	}
	ev := evs.Items[0]
	if ev.Reason != "AIProposalCreated" {
		t.Errorf("reason = %q; want AIProposalCreated", ev.Reason)
	}
	if ev.Type != corev1.EventTypeNormal {
		t.Errorf("type = %q; want Normal", ev.Type)
	}
	if ev.InvolvedObject.Kind != "Pod" || ev.InvolvedObject.Namespace != "default" || ev.InvolvedObject.Name != "demo-abc" {
		t.Errorf("involved object = %+v; want Pod/default/demo-abc", ev.InvolvedObject)
	}
	if ev.Annotations["cha.bionicaisolutions.com/audit-tier"] != "t1" {
		t.Errorf("tier annotation = %q; want t1", ev.Annotations["cha.bionicaisolutions.com/audit-tier"])
	}
	if ev.Annotations["cha.bionicaisolutions.com/audit-details"] == "" {
		t.Errorf("missing audit-details annotation")
	}
}

func TestEventsSink_Write_Warning(t *testing.T) {
	client := fake.NewClientset()
	s := NewEventsSink(client, "cha")
	ctx := context.Background()

	err := s.Write(ctx, ai.AuditEvent{
		Type:          "ai.approval.rejected",
		CorrelationID: "act-test-2",
		Tier:          ai.TierT1,
		Actor:         "approval-server",
		Subject:       "Pod/default/demo-abc",
		Details:       map[string]any{"reason": "token_replay"},
	})
	if err != nil {
		t.Fatal(err)
	}

	evs, _ := client.CoreV1().Events("cha").List(ctx, metav1.ListOptions{})
	if evs.Items[0].Type != corev1.EventTypeWarning {
		t.Errorf("expected Warning type for rejection event; got %q", evs.Items[0].Type)
	}
}

func TestEventReason(t *testing.T) {
	cases := map[string]string{
		"ai.proposal.created":      "AIProposalCreated",
		"ai.approval.granted":      "AIApprovalGranted",
		"ai.action.applied":        "AIActionApplied",
		"ai.runbook.dual_approval": "AIRunbookDualApproval",
		"ai.llm.call":              "AILLMCall",
		"":                         "AIEvent",
	}
	for in, want := range cases {
		if got := eventReason(in); got != want {
			t.Errorf("eventReason(%q) = %q; want %q", in, got, want)
		}
	}
}

func TestParseInvolved(t *testing.T) {
	tests := []struct {
		subject string
		wantK   string
		wantNS  string
		wantN   string
	}{
		{"Pod/default/demo-abc", "Pod", "default", "demo-abc"},
		{"missing-key/billing/svc/STRIPE_API_KEY", "missing-key", "billing", "svc"},
		{"", "AuditScope", "default-ns", "cha"},
		{"unparseable", "AuditScope", "default-ns", "cha"},
	}
	for _, tc := range tests {
		k, ns, n := parseInvolved(tc.subject, "default-ns")
		if k != tc.wantK || ns != tc.wantNS || n != tc.wantN {
			t.Errorf("parseInvolved(%q) = (%q,%q,%q); want (%q,%q,%q)",
				tc.subject, k, ns, n, tc.wantK, tc.wantNS, tc.wantN)
		}
	}
}

func TestEventsSink_NilSafety(t *testing.T) {
	var s *EventsSink
	if err := s.Write(context.Background(), ai.AuditEvent{Type: "x"}); err != nil {
		t.Errorf("nil sink should return nil error; got %v", err)
	}
}
