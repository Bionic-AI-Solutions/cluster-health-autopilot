// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package fix

import (
	"testing"

	"github.com/srenix-ai/agentic-sre/pkg/ai"
)

// fixerFloor pins the compiled-in no-touch floor consumed by every
// fixer guard. Removing an entry here is a breaking safety change.
var fixerFloor = []string{
	"kube-system",
	"kube-public",
	"kube-node-lease",
	"rook-ceph",
	"vault",
	"external-secrets",
	"cnpg-system",
}

func TestIsProtectedNamespace_Floor(t *testing.T) {
	ai.SetExtraProtectedNamespaces()
	t.Cleanup(func() { ai.SetExtraProtectedNamespaces() })

	for _, ns := range fixerFloor {
		if !IsProtectedNamespace(ns) {
			t.Errorf("IsProtectedNamespace(%q) = false; compiled-in floor must always hold", ns)
		}
	}
	if IsProtectedNamespace("") {
		t.Error("IsProtectedNamespace(\"\") = true; cluster-scoped must stay un-gated here")
	}
	if IsProtectedNamespace("default") {
		t.Error("IsProtectedNamespace(default) = true with no extras configured")
	}
}

// TestIsProtectedNamespace_ExtensionVisibleToFixerGuard — the fixer
// guard honors the same append-only extension the AI validator reads
// (SRENIX_PROTECTED_NAMESPACES_EXTRA / ai.SetExtraProtectedNamespaces),
// so one knob protects a namespace on BOTH act-side surfaces.
func TestIsProtectedNamespace_ExtensionVisibleToFixerGuard(t *testing.T) {
	ai.SetExtraProtectedNamespaces("prod-payments")
	t.Cleanup(func() { ai.SetExtraProtectedNamespaces() })

	if !IsProtectedNamespace("prod-payments") {
		t.Error("IsProtectedNamespace(prod-payments) = false; extras must gate fixers too")
	}
	for _, ns := range fixerFloor {
		if !IsProtectedNamespace(ns) {
			t.Errorf("floor entry %q lost while extras are set", ns)
		}
	}
}

// TestIsProtectedNamespace_EnvExtension — end-to-end through the env
// var the chart/operator render onto the watcher + remediate containers.
func TestIsProtectedNamespace_EnvExtension(t *testing.T) {
	t.Setenv(ai.EnvProtectedNamespacesExtra, " tenant-a , ,tenant-a,prod-db ")
	ai.LoadExtraProtectedNamespacesFromEnv()
	t.Cleanup(func() { ai.SetExtraProtectedNamespaces() })

	for _, ns := range []string{"tenant-a", "prod-db"} {
		if !IsProtectedNamespace(ns) {
			t.Errorf("IsProtectedNamespace(%q) = false after env extension", ns)
		}
	}
}

// TestIsProtectedNamespace_GarbageEnvCannotClearFloor — setting the env
// to garbage must never shrink the compiled-in floor.
func TestIsProtectedNamespace_GarbageEnvCannotClearFloor(t *testing.T) {
	t.Setenv(ai.EnvProtectedNamespacesExtra, ",,  ,")
	ai.LoadExtraProtectedNamespacesFromEnv()
	t.Cleanup(func() { ai.SetExtraProtectedNamespaces() })

	for _, ns := range fixerFloor {
		if !IsProtectedNamespace(ns) {
			t.Errorf("garbage env cleared floor entry %q", ns)
		}
	}
}
