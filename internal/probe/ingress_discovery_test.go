// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package probe

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
)

// Helper local to this file so we don't depend on test ordering with postgres_test.go.
func loadSrcDisc(t *testing.T, files map[string]string) snapshot.Source {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	src, err := snapshot.LoadFile(dir)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	return src
}

const ingressList = `{
  "apiVersion": "networking.k8s.io/v1",
  "kind": "IngressList",
  "items": [
    {
      "apiVersion": "networking.k8s.io/v1",
      "kind": "Ingress",
      "metadata": {"name": "app-ing", "namespace": "app"},
      "spec": {"rules": [{"host": "app.example.com"}]}
    },
    {
      "apiVersion": "networking.k8s.io/v1",
      "kind": "Ingress",
      "metadata": {"name": "vault-ing", "namespace": "vault"},
      "spec": {"rules": [{"host": "vault.example.com"}]}
    },
    {
      "apiVersion": "networking.k8s.io/v1",
      "kind": "Ingress",
      "metadata": {
        "name":        "internal-ing",
        "namespace":   "internal",
        "annotations": {"cha.bionicaisolutions.com/probe-disable": "true"}
      },
      "spec": {"rules": [{"host": "internal.example.com"}]}
    },
    {
      "apiVersion": "networking.k8s.io/v1",
      "kind": "Ingress",
      "metadata": {"name": "shared-host", "namespace": "billing"},
      "spec": {"rules": [{"host": "billing.example.com"}, {"host": "pay.example.com"}]}
    },
    {
      "apiVersion": "networking.k8s.io/v1",
      "kind": "Ingress",
      "metadata": {"name": "duplicate-app", "namespace": "app2"},
      "spec": {"rules": [{"host": "app.example.com"}]}
    },
    {
      "apiVersion": "networking.k8s.io/v1",
      "kind": "Ingress",
      "metadata": {"name": "already-covered", "namespace": "site"},
      "spec": {"rules": [{"host": "apex.example.com"}]}
    }
  ]
}`

func TestDiscoverIngressTargets_Defaults(t *testing.T) {
	src := loadSrcDisc(t, map[string]string{"ingresses.json": ingressList})

	existing := []string{"https://apex.example.com"}
	// Use the public helper hostnamesOf so behavior matches production wiring.
	got := DiscoverIngressTargets(
		context.Background(),
		src,
		DefaultDiscoveryOptions(),
		hostnamesOf(targetsFromURLs(existing)),
	)

	urls := make([]string, 0, len(got))
	for _, t := range got {
		urls = append(urls, t.URL)
	}
	sort.Strings(urls)

	want := []string{
		"https://app.example.com",     // discovered once even though two Ingresses expose it
		"https://billing.example.com", // multi-host Ingress, each host emits its own target
		"https://pay.example.com",
	}
	sort.Strings(want)

	if !equalSlices(urls, want) {
		t.Fatalf("discovered targets mismatch\n  got:  %v\n  want: %v", urls, want)
	}

	// Confirm: protected namespace excluded, opt-out annotation honored, existing host skipped.
	for _, u := range urls {
		switch u {
		case "https://vault.example.com":
			t.Errorf("vault.example.com leaked through protected-namespace skip")
		case "https://internal.example.com":
			t.Errorf("internal.example.com leaked through opt-out annotation")
		case "https://apex.example.com":
			t.Errorf("apex.example.com leaked through existing-host skip")
		}
	}
}

// cert-manager HTTP-01 solver Ingresses are transient and must never be
// discovered as probe targets (they caused churning false-criticals +
// ticket spam during the langfuse cert issuance).
func TestDiscoverIngressTargets_SkipsAcmeSolver(t *testing.T) {
	const list = `{
  "apiVersion": "networking.k8s.io/v1", "kind": "IngressList",
  "items": [
    {"apiVersion": "networking.k8s.io/v1", "kind": "Ingress",
     "metadata": {"name": "real-ing", "namespace": "app"},
     "spec": {"rules": [{"host": "real.example.com"}]}},
    {"apiVersion": "networking.k8s.io/v1", "kind": "Ingress",
     "metadata": {"name": "cm-acme-http-solver-2qkzp", "namespace": "app"},
     "spec": {"rules": [{"host": "real.example.com"}]}}
  ]
}`
	src := loadSrcDisc(t, map[string]string{"ingresses.json": list})
	got := DiscoverIngressTargets(context.Background(), src, DefaultDiscoveryOptions(), nil)
	for _, tgt := range got {
		if tgt.Name != "" && (tgt.URL == "" || containsSolver(tgt.Name)) {
			t.Errorf("acme solver ingress leaked into targets: %+v", tgt)
		}
	}
	// real.example.com still discovered (via the real ingress).
	if len(got) != 1 || got[0].URL != "https://real.example.com" {
		t.Fatalf("expected only real.example.com; got: %+v", got)
	}
}

func containsSolver(s string) bool { return strings.Contains(s, "cm-acme-http-solver-") }

func TestDiscoverIngressTargets_Disabled(t *testing.T) {
	src := loadSrcDisc(t, map[string]string{"ingresses.json": ingressList})
	opts := DefaultDiscoveryOptions()
	opts.Enabled = false

	got := DiscoverIngressTargets(context.Background(), src, opts, nil)
	if len(got) != 0 {
		t.Errorf("expected no targets when Discovery disabled, got %d", len(got))
	}
}

func TestDiscoverIngressTargets_NoIngressList(t *testing.T) {
	// Empty source — no ingresses at all.
	src := loadSrcDisc(t, map[string]string{})
	got := DiscoverIngressTargets(context.Background(), src, DefaultDiscoveryOptions(), nil)
	if len(got) != 0 {
		t.Errorf("expected no targets when source has no Ingresses, got %d", len(got))
	}
}

func TestHostnamesOf(t *testing.T) {
	in := []EndpointTarget{
		{URL: "https://a.example.com"},
		{URL: "https://b.example.com/path?x=1"},
		{URL: "http://c.example.com:8080"},
	}
	got := hostnamesOf(in)
	want := []string{"a.example.com", "b.example.com", "c.example.com:8080"}
	if !equalSlices(got, want) {
		t.Fatalf("hostnamesOf mismatch\n  got:  %v\n  want: %v", got, want)
	}
}

// ---- test helpers ----------------------------------------------------------

func targetsFromURLs(urls []string) []EndpointTarget {
	out := make([]EndpointTarget, 0, len(urls))
	for _, u := range urls {
		out = append(out, EndpointTarget{URL: u})
	}
	return out
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
