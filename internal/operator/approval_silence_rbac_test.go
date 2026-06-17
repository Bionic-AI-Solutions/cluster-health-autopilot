// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	"testing"
)

func TestBuildApprovalSilenceWriterRole_GrantsCreateSilences(t *testing.T) {
	cr := approvalCR()
	r := BuildApprovalSilenceWriterRole(cr)
	if r == nil {
		t.Fatal("approval-enabled CR should produce a silence-writer Role")
	}
	if r.Namespace != cr.Namespace {
		t.Errorf("silence-writer Role namespace = %q want %q", r.Namespace, cr.Namespace)
	}
	if len(r.Rules) != 1 {
		t.Fatalf("expected 1 rule; got %d", len(r.Rules))
	}
	rule := r.Rules[0]
	if len(rule.APIGroups) != 1 || rule.APIGroups[0] != "cha.bionicaisolutions.com" {
		t.Errorf("apiGroups = %v want [cha.bionicaisolutions.com]", rule.APIGroups)
	}
	if len(rule.Resources) != 1 || rule.Resources[0] != "silences" {
		t.Errorf("resources = %v want [silences]", rule.Resources)
	}
	wantVerbs := map[string]bool{"create": true, "get": true, "list": true}
	for _, v := range rule.Verbs {
		if !wantVerbs[v] {
			t.Errorf("unexpected verb %q", v)
		}
		delete(wantVerbs, v)
	}
	if len(wantVerbs) != 0 {
		t.Errorf("missing verbs: %v", wantVerbs)
	}
}

func TestBuildApprovalSilenceWriterRoleBinding_TiesToApprovalSA(t *testing.T) {
	cr := approvalCR()
	rb := BuildApprovalSilenceWriterRoleBinding(cr)
	if rb == nil {
		t.Fatal("approval-enabled CR should produce a silence-writer RoleBinding")
	}
	if rb.RoleRef.Name != ApprovalServerName(cr)+"-silence-writer" {
		t.Errorf("roleRef name = %q", rb.RoleRef.Name)
	}
	if len(rb.Subjects) != 1 || rb.Subjects[0].Name != ApprovalServerName(cr) {
		t.Errorf("subject not bound to approval-server SA: %+v", rb.Subjects)
	}
}
