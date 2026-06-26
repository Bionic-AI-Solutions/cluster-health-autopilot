// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	"testing"

	rbacv1 "k8s.io/api/rbac/v1"
)

// P1.2 — the DisruptionDrift analyzer lists policy/v1
// poddisruptionbudgets and core resourcequotas. Until v1.26.0 NO RBAC
// surface granted either resource, so the analyzer's soft-fail
// (`if err != nil { return nil }`) made it silently dead on every
// real cluster (fake clients don't enforce RBAC, so unit tests never
// caught it). This test pins the grant into the operator-built reader
// role; the chart parity test (chart_rbac_parity_test.go) extends the
// pin to the chart surface.
func TestBuildReaderClusterRole_GrantsDisruptionDriftResources(t *testing.T) {
	role := BuildReaderClusterRole()

	wants := []struct {
		group    string
		resource string
	}{
		{"policy", "poddisruptionbudgets"},
		{"", "resourcequotas"},
	}
	verbs := []string{"get", "list", "watch"}

	for _, w := range wants {
		for _, v := range verbs {
			if !roleGrants(role.Rules, w.group, w.resource, v) {
				t.Errorf("BuildReaderClusterRole() missing %q on %s/%s — DisruptionDrift analyzer is silently dead without it",
					v, orCore(w.group), w.resource)
			}
		}
	}
}

// roleGrants reports whether the rule set grants verb on group/resource.
func roleGrants(rules []rbacv1.PolicyRule, group, resource, verb string) bool {
	for _, r := range rules {
		if containsStr(r.APIGroups, group) && containsStr(r.Resources, resource) && containsStr(r.Verbs, verb) {
			return true
		}
	}
	return false
}

func containsStr(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}

func orCore(group string) string {
	if group == "" {
		return "core"
	}
	return group
}
