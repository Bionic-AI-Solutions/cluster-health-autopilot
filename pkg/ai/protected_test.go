// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package ai

import (
	"errors"
	"reflect"
	"testing"
)

// floorNamespaces is the compiled-in no-touch floor. Pinned here so a
// future edit that REMOVES a floor entry fails loudly — the extension
// mechanism is append-only by contract.
var floorNamespaces = []string{
	"kube-system",
	"kube-public",
	"kube-node-lease",
	"rook-ceph",
	"vault",
	"external-secrets",
	"cnpg-system",
}

func resetExtras(t *testing.T) {
	t.Helper()
	SetExtraProtectedNamespaces()
	t.Cleanup(func() { SetExtraProtectedNamespaces() })
}

func TestProtectedNamespaces_FloorAlwaysPresent(t *testing.T) {
	resetExtras(t)
	for _, ns := range floorNamespaces {
		if !IsProtectedNamespace(ns) {
			t.Errorf("IsProtectedNamespace(%q) = false; compiled-in floor must always hold", ns)
		}
		if _, ok := ProtectedNamespaces[ns]; !ok {
			t.Errorf("ProtectedNamespaces missing floor entry %q", ns)
		}
	}
}

func TestSetExtraProtectedNamespaces_ExtendsFloor(t *testing.T) {
	resetExtras(t)
	SetExtraProtectedNamespaces("prod-payments", "tenant-a")
	for _, ns := range []string{"prod-payments", "tenant-a"} {
		if !IsProtectedNamespace(ns) {
			t.Errorf("IsProtectedNamespace(%q) = false after SetExtraProtectedNamespaces", ns)
		}
		if !IsExtraProtectedNamespace(ns) {
			t.Errorf("IsExtraProtectedNamespace(%q) = false after SetExtraProtectedNamespaces", ns)
		}
	}
	// The floor stays intact alongside the extras.
	for _, ns := range floorNamespaces {
		if !IsProtectedNamespace(ns) {
			t.Errorf("floor entry %q lost after setting extras", ns)
		}
	}
	// Unrelated namespaces stay unprotected.
	if IsProtectedNamespace("default") {
		t.Error("IsProtectedNamespace(default) = true; extras leaked")
	}
}

func TestSetExtraProtectedNamespaces_CannotClearFloor(t *testing.T) {
	resetExtras(t)
	// An operator trying to "replace" the list with garbage (or even
	// with floor names) must never remove a floor entry.
	SetExtraProtectedNamespaces("", "   ", "kube-system", "!!garbage!!")
	for _, ns := range floorNamespaces {
		if !IsProtectedNamespace(ns) {
			t.Errorf("floor entry %q cleared by garbage extras — floor must be append-only", ns)
		}
	}
	// Clearing extras entirely also keeps the floor.
	SetExtraProtectedNamespaces()
	for _, ns := range floorNamespaces {
		if !IsProtectedNamespace(ns) {
			t.Errorf("floor entry %q lost after clearing extras", ns)
		}
	}
}

func TestLoadExtraProtectedNamespacesFromEnv(t *testing.T) {
	resetExtras(t)
	t.Setenv(EnvProtectedNamespacesExtra, " prod-payments, ,tenant-a ,prod-payments,")
	LoadExtraProtectedNamespacesFromEnv()
	t.Cleanup(func() { SetExtraProtectedNamespaces() })

	for _, ns := range []string{"prod-payments", "tenant-a"} {
		if !IsProtectedNamespace(ns) {
			t.Errorf("IsProtectedNamespace(%q) = false after env load", ns)
		}
	}
	if got := ExtraProtectedNamespaces(); !reflect.DeepEqual(got, []string{"prod-payments", "tenant-a"}) {
		t.Errorf("ExtraProtectedNamespaces() = %v; want trimmed+deduped [prod-payments tenant-a]", got)
	}
	for _, ns := range floorNamespaces {
		if !IsProtectedNamespace(ns) {
			t.Errorf("floor entry %q lost after env load", ns)
		}
	}
}

func TestLoadExtraProtectedNamespacesFromEnv_GarbageCannotClearFloor(t *testing.T) {
	resetExtras(t)
	for _, garbage := range []string{"", " , ,, ", ",,,,", "   "} {
		t.Setenv(EnvProtectedNamespacesExtra, garbage)
		LoadExtraProtectedNamespacesFromEnv()
		for _, ns := range floorNamespaces {
			if !IsProtectedNamespace(ns) {
				t.Errorf("env=%q cleared floor entry %q", garbage, ns)
			}
		}
		if got := ExtraProtectedNamespaces(); len(got) != 0 {
			t.Errorf("env=%q produced extras %v; want none", garbage, got)
		}
	}
}

func TestIsProtectedNamespace_LazyEnvLoad(t *testing.T) {
	resetExtras(t)
	// Force the un-initialized state so the first IsProtectedNamespace
	// call performs the lazy env read (the production startup path —
	// no explicit initializer call needed in main()).
	t.Setenv(EnvProtectedNamespacesExtra, "lazy-ns")
	extraMu.Lock()
	extraLoaded = false
	extraProtected = nil
	extraMu.Unlock()
	t.Cleanup(func() { SetExtraProtectedNamespaces() })

	if !IsProtectedNamespace("lazy-ns") {
		t.Error("lazy env load: IsProtectedNamespace(lazy-ns) = false; want true")
	}
}

func TestParseProtectedNamespacesExtra(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{" , ,", nil},
		{"a", []string{"a"}},
		{"a,b", []string{"a", "b"}},
		{" a , b ,a", []string{"a", "b"}},
		{",x,,y,", []string{"x", "y"}},
	}
	for _, c := range cases {
		if got := ParseProtectedNamespacesExtra(c.in); !reflect.DeepEqual(got, c.want) {
			t.Errorf("ParseProtectedNamespacesExtra(%q) = %v; want %v", c.in, got, c.want)
		}
	}
}

// TestProposalValidate_ExtraProtectedNamespace — the AI-action validator
// must reject proposals targeting an operator-extended namespace exactly
// like a compiled-in one.
func TestProposalValidate_ExtraProtectedNamespace(t *testing.T) {
	resetExtras(t)
	SetExtraProtectedNamespaces("prod-payments")

	p := validProposalTargeting(t, "prod-payments")
	if err := p.Validate(); !errors.Is(err, ErrProtectedNamespace) {
		t.Errorf("Validate() on extra-protected ns = %v; want ErrProtectedNamespace", err)
	}

	// And the same proposal passes once the extra is removed.
	SetExtraProtectedNamespaces()
	if err := p.Validate(); err != nil {
		t.Errorf("Validate() after clearing extras = %v; want nil", err)
	}
}

// TestValidateManifest_ExtraProtectedNamespace — the safe-apply manifest
// validator shares the same extended floor.
func TestValidateManifest_ExtraProtectedNamespace(t *testing.T) {
	resetExtras(t)
	SetExtraProtectedNamespaces("prod-payments")

	manifest := []byte(`apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: bad
  namespace: prod-payments
spec: {podSelector: {}}`)
	if err := ValidateManifest(manifest); !errors.Is(err, ErrManifestProtectedNS) {
		t.Errorf("ValidateManifest() on extra-protected ns = %v; want ErrManifestProtectedNS", err)
	}
}

// validProposalTargeting builds a structurally valid proposal targeting
// the given namespace, mirroring newValidProposal in validate_test.go.
func validProposalTargeting(t *testing.T, ns string) AIProposedAction {
	t.Helper()
	p := newValidProposal()
	p.Target.Namespace = ns
	return p
}
