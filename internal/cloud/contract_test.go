// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

// CROSS-REPO CONTRACT TEST — do not change the literal suffix formats
// below without coordinating with CHA-com.
//
// CHA-com's cross-resource RCA matchers (ai/cloudcontext, PR #65) join
// Kubernetes resources to cloud findings by parsing tokens out of the
// finding MESSAGE:
//
//	" (lb: <value>)"        — LB probes (aws-alb-target-health,
//	                          gcp-lb-backends, azure-appgw-backends)
//	" (domains: <d1>,<d2>)" — azure-certs cert findings
//
// CHA-com's fixtures encode this exact shape: single leading space,
// literal "(lb: " / "(domains: ", comma-separated domains with NO
// spaces, closing paren. If a test in this file fails, you are breaking
// the CHA-com RCA join contract — fix the format back or update both
// repos in lockstep.
package cloud

import (
	"fmt"
	"testing"
)

// The literal cross-repo format strings. CHA-com ai/cloudcontext
// matchers parse exactly these.
const (
	contractLBFormat      = " (lb: %s)"
	contractDomainsFormat = " (domains: %s)"
)

func TestJoinKeyLB_MatchesCrossRepoContract(t *testing.T) {
	value := "my-alb-123.us-east-1.elb.amazonaws.com"
	want := fmt.Sprintf(contractLBFormat, value)
	if got := JoinKeyLB(value); got != want {
		t.Errorf("JoinKeyLB(%q) = %q, want %q", value, got, want)
	}
	if want != " (lb: my-alb-123.us-east-1.elb.amazonaws.com)" {
		t.Errorf("contract literal drifted: %q", want)
	}
}

func TestJoinKeyLB_EmptyValueOmitsSuffix(t *testing.T) {
	// Guard: never emit an empty " (lb: )" — absent value means NO suffix.
	if got := JoinKeyLB(""); got != "" {
		t.Errorf("JoinKeyLB(\"\") = %q, want \"\"", got)
	}
}

func TestJoinKeyDomains_MatchesCrossRepoContract(t *testing.T) {
	want := fmt.Sprintf(contractDomainsFormat, "example.com,www.example.com")
	if got := JoinKeyDomains([]string{"example.com", "www.example.com"}); got != want {
		t.Errorf("JoinKeyDomains = %q, want %q", got, want)
	}
	// Comma-separated, NO spaces between domains.
	if want != " (domains: example.com,www.example.com)" {
		t.Errorf("contract literal drifted: %q", want)
	}
}

func TestJoinKeyDomains_SingleDomain(t *testing.T) {
	if got, want := JoinKeyDomains([]string{"example.com"}), " (domains: example.com)"; got != want {
		t.Errorf("JoinKeyDomains = %q, want %q", got, want)
	}
}

func TestJoinKeyDomains_FiltersEmptyEntries(t *testing.T) {
	if got, want := JoinKeyDomains([]string{"", "a.com", "", "b.com"}), " (domains: a.com,b.com)"; got != want {
		t.Errorf("JoinKeyDomains = %q, want %q", got, want)
	}
}

func TestJoinKeyDomains_NoDomainsOmitsSuffix(t *testing.T) {
	// Guard: no known domains → NO suffix (never an empty "(domains: )").
	for _, in := range [][]string{nil, {}, {""}, {"", ""}} {
		if got := JoinKeyDomains(in); got != "" {
			t.Errorf("JoinKeyDomains(%v) = %q, want \"\"", in, got)
		}
	}
}
