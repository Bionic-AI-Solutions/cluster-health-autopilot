// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package probe

import "github.com/srenix-ai/agentic-sre/pkg/ai"

// probeProtectedNamespaces mirrors the canonical no-touch list in
// internal/fix/protected.go. Kept in sync by convention; consolidation
// into a shared pkg-level helper is tracked under the Sprint 5 operator
// port (see docs/design/2026-05-hardening-plan.md).
//
// In probe-land, these namespaces aren't "untouched" — they are
// "always-critical." A CrashLoopBackOff in kube-system is more important
// than the same loop in user-ns, not less.
var probeProtectedNamespaces = map[string]struct{}{
	"kube-system":      {},
	"kube-public":      {},
	"kube-node-lease":  {},
	"rook-ceph":        {},
	"vault":            {},
	"external-secrets": {},
	"cnpg-system":      {},
	"cert-manager":     {},
}

// IsProtectedNamespace reports whether the given namespace contains
// platform-critical workloads — the compiled-in floor above plus any
// operator-appended extras (SRENIX_PROTECTED_NAMESPACES_EXTRA, shared via
// pkg/ai with the fixer guard and the AI-action validator). Used by
// probes to escalate severity: a namespace important enough to be
// no-touch on the act side is important enough to be always-critical
// on the detect side.
func IsProtectedNamespace(ns string) bool {
	if ns == "" {
		return false
	}
	if _, ok := probeProtectedNamespaces[ns]; ok {
		return true
	}
	return ai.IsExtraProtectedNamespace(ns)
}
