// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package fix

// protectedNamespaces are NEVER touched by any fixer regardless of state.
// Mirrors is_protected_ns from cluster-health-report.sh.
//
// Rationale: these namespaces host platform components whose lifecycle is
// owned by their respective operators (kube-system) or is so security-
// sensitive (vault, external-secrets) that any auto-action could mask a
// real incident. The diagnose analyzers still surface findings in these
// namespaces — only the act-side is gated.
var protectedNamespaces = map[string]struct{}{
	"kube-system":      {},
	"kube-public":      {},
	"kube-node-lease":  {},
	"rook-ceph":        {},
	"vault":            {},
	"external-secrets": {},
	"cnpg-system":      {},
}

// IsProtectedNamespace reports whether the given namespace is on the
// no-touch list. Cluster-scoped resources (ns == "") are never protected
// at the namespace level — fixers must perform their own kind-level
// safety checks for those.
func IsProtectedNamespace(ns string) bool {
	if ns == "" {
		return false
	}
	_, ok := protectedNamespaces[ns]
	return ok
}
