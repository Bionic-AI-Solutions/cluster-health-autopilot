// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"testing"
)

// TestTicketingSpec_DeepCopy_AllFields exercises every nested pointer +
// slice on TicketingSpec so the hand-written DeepCopy in
// zz_generated.deepcopy.go is exhaustively covered. Mutating the source
// after DeepCopy must not affect the destination — proves nothing was
// aliased.
func TestTicketingSpec_DeepCopy_AllFields(t *testing.T) {
	src := &TicketingSpec{
		Enabled:        true,
		Provider:       "openproject",
		Cluster:        "bionic",
		MCPURL:         "http://mcp-openproject-server.mcp.svc:8006/mcp",
		Project:        "6",
		TypeID:         "36",
		ClosedStatusID: "82",
		WebURLPrefix:   "https://op.example.com",
		SeverityPriority: &TicketingPrioritySpec{
			Critical: "75",
			Warning:  "74",
			Info:     "73",
		},
		Labels: []string{"srenix", "auto-filed"},
		DryRun: true,
		Auth: &TicketingAuthSpec{
			Enabled:    true,
			SecretName: "srenix-ticketing-mcp",
			SecretKey:  "api-key",
		},
	}
	dst := src.DeepCopy()

	if dst == src {
		t.Fatal("DeepCopy should return a new pointer")
	}
	if dst.SeverityPriority == src.SeverityPriority {
		t.Error("nested SeverityPriority pointer not cloned")
	}
	if dst.Auth == src.Auth {
		t.Error("nested Auth pointer not cloned")
	}
	if &dst.Labels[0] == &src.Labels[0] {
		t.Error("Labels slice not cloned (header aliased)")
	}

	// Mutate the source and confirm the destination is untouched.
	src.Enabled = false
	src.Provider = "MUTATED"
	src.Project = "MUTATED"
	src.SeverityPriority.Critical = "MUTATED"
	src.Auth.SecretName = "MUTATED"
	src.Labels[0] = "MUTATED"

	if !dst.Enabled {
		t.Error("DeepCopy shared Enabled (bool by value should be independent)")
	}
	if dst.Provider == "MUTATED" {
		t.Error("DeepCopy shared Provider underlying value")
	}
	if dst.Project == "MUTATED" {
		t.Error("DeepCopy shared Project underlying value")
	}
	if dst.SeverityPriority.Critical == "MUTATED" {
		t.Error("DeepCopy shared SeverityPriority underlying value")
	}
	if dst.Auth.SecretName == "MUTATED" {
		t.Error("DeepCopy shared Auth underlying value")
	}
	if dst.Labels[0] == "MUTATED" {
		t.Error("DeepCopy shared Labels slice")
	}
}

func TestTicketingSpec_DeepCopy_Nil(t *testing.T) {
	var s *TicketingSpec
	if got := s.DeepCopy(); got != nil {
		t.Errorf("DeepCopy on nil receiver should return nil; got %+v", got)
	}
}

func TestTicketingSpec_DeepCopy_OptionalsAbsent(t *testing.T) {
	// Realistic minimal spec — only the master switch + provider.
	// DeepCopy must not panic on nil SeverityPriority / Auth / Labels.
	src := &TicketingSpec{
		Enabled:  true,
		Provider: "openproject",
	}
	dst := src.DeepCopy()
	if dst == nil {
		t.Fatal("DeepCopy returned nil for non-nil source")
	}
	if dst.SeverityPriority != nil {
		t.Errorf("nil SeverityPriority leaked as non-nil after DeepCopy")
	}
	if dst.Auth != nil {
		t.Errorf("nil Auth leaked as non-nil after DeepCopy")
	}
	if dst.Labels != nil {
		t.Errorf("nil Labels leaked as non-nil after DeepCopy; got %v", dst.Labels)
	}
}

// TestTicketingSpec_RoundTripThroughAgenticSRE — proves the
// outer Spec's DeepCopy traverses spec.ticketing correctly (catches the
// case where someone adds a field to AgenticSRESpec but
// forgets to DeepCopy it).
func TestTicketingSpec_RoundTripThroughAgenticSRE(t *testing.T) {
	cr := &AgenticSRE{
		Spec: AgenticSRESpec{
			Image: ImageSpec{Repository: "x/y", Tag: "1.0"},
			Ticketing: &TicketingSpec{
				Enabled:          true,
				Provider:         "openproject",
				Project:          "6",
				Labels:           []string{"a", "b"},
				SeverityPriority: &TicketingPrioritySpec{Critical: "75"},
			},
		},
	}
	clone := cr.DeepCopy()
	if clone.Spec.Ticketing == cr.Spec.Ticketing {
		t.Fatal("CR.DeepCopy did not clone Spec.Ticketing pointer")
	}
	if clone.Spec.Ticketing.SeverityPriority == cr.Spec.Ticketing.SeverityPriority {
		t.Fatal("CR.DeepCopy did not clone nested SeverityPriority pointer")
	}

	// Mutate source, confirm clone unchanged.
	cr.Spec.Ticketing.Project = "MUTATED"
	cr.Spec.Ticketing.Labels[0] = "MUTATED"
	cr.Spec.Ticketing.SeverityPriority.Critical = "MUTATED"
	if clone.Spec.Ticketing.Project == "MUTATED" {
		t.Error("CR.DeepCopy clone tracks source Project")
	}
	if clone.Spec.Ticketing.Labels[0] == "MUTATED" {
		t.Error("CR.DeepCopy clone tracks source Labels[0]")
	}
	if clone.Spec.Ticketing.SeverityPriority.Critical == "MUTATED" {
		t.Error("CR.DeepCopy clone tracks source SeverityPriority")
	}
}
