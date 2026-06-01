// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package netpol

import (
	"context"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func makePod(ns, name string) unstructured.Unstructured {
	u := unstructured.Unstructured{}
	u.SetAPIVersion("v1")
	u.SetKind("Pod")
	u.SetNamespace(ns)
	u.SetName(name)
	return u
}

func makeNetPol(ns, name string) unstructured.Unstructured {
	u := unstructured.Unstructured{}
	u.SetAPIVersion("networking.k8s.io/v1")
	u.SetKind("NetworkPolicy")
	u.SetNamespace(ns)
	u.SetName(name)
	return u
}

func makeIngress(ns, name, className string) unstructured.Unstructured {
	u := unstructured.Unstructured{}
	u.SetAPIVersion("networking.k8s.io/v1")
	u.SetKind("Ingress")
	u.SetNamespace(ns)
	u.SetName(name)
	if className != "" {
		_ = unstructured.SetNestedField(u.Object, className, "spec", "ingressClassName")
	}
	return u
}

func TestNoopProposer_AlwaysNil(t *testing.T) {
	p := NoopProposer{}
	got, err := p.ProposeForNamespace(context.Background(), &memSourceNet{}, "any")
	if err != nil {
		t.Errorf("NoopProposer must never error; got %v", err)
	}
	if got != nil {
		t.Errorf("NoopProposer must return nil; got %+v", got)
	}
}

func TestSnapshotProposer_SystemNamespaceSkipped(t *testing.T) {
	src := &memSourceNet{byResource: map[string][]unstructured.Unstructured{
		"pods": {makePod("kube-system", "kube-proxy")},
	}}
	p := SnapshotProposer{}
	got, _ := p.ProposeForNamespace(context.Background(), src, "kube-system")
	if got != nil {
		t.Errorf("kube-system must be skipped; got %+v", got)
	}
}

func TestSnapshotProposer_NamespaceWithExistingNetPolSkipped(t *testing.T) {
	src := &memSourceNet{byResource: map[string][]unstructured.Unstructured{
		"pods":            {makePod("app", "a-7d")},
		"networkpolicies": {makeNetPol("app", "existing-policy")},
	}}
	p := SnapshotProposer{}
	got, _ := p.ProposeForNamespace(context.Background(), src, "app")
	if got != nil {
		t.Errorf("ns with existing NetPol must be skipped; got %+v", got)
	}
}

func TestSnapshotProposer_EmptyNamespaceSkipped(t *testing.T) {
	src := &memSourceNet{byResource: map[string][]unstructured.Unstructured{}}
	p := SnapshotProposer{}
	got, _ := p.ProposeForNamespace(context.Background(), src, "empty-ns")
	if got != nil {
		t.Errorf("namespace with zero pods must be skipped; got %+v", got)
	}
}

func TestSnapshotProposer_HappyPath_KongIngress(t *testing.T) {
	src := &memSourceNet{byResource: map[string][]unstructured.Unstructured{
		"pods":      {makePod("app", "a-7d")},
		"ingresses": {makeIngress("app", "a-ing", "kong")},
	}}
	p := SnapshotProposer{}
	got, err := p.ProposeForNamespace(context.Background(), src, "app")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got == nil {
		t.Fatal("expected a Proposal; got nil")
	}
	if got.Namespace != "app" {
		t.Errorf("Namespace=%q want app", got.Namespace)
	}
	if got.PolicyKind != "NetworkPolicy" {
		t.Errorf("PolicyKind=%q want NetworkPolicy", got.PolicyKind)
	}
	if !strings.Contains(got.PolicyYAML, "podSelector: {}") {
		t.Errorf("YAML must include podSelector: {}; got:\n%s", got.PolicyYAML)
	}
	if !strings.Contains(got.PolicyYAML, "policyTypes:\n    - Ingress") {
		t.Errorf("YAML must include Ingress policyType; got:\n%s", got.PolicyYAML)
	}
	if strings.Contains(got.PolicyYAML, "Egress") {
		t.Errorf("YAML must NOT include Egress (egress is intentionally unrestricted); got:\n%s", got.PolicyYAML)
	}
	if !strings.Contains(got.PolicyYAML, `values: ["kong"]`) {
		t.Errorf("YAML must include allow-from kong; got:\n%s", got.PolicyYAML)
	}
	if len(got.AllowSources) < 2 {
		t.Errorf("expected >=2 AllowSources (same-namespace + kong); got %+v", got.AllowSources)
	}
}

func TestSnapshotProposer_NoIngresses_OnlySameNamespaceAllow(t *testing.T) {
	src := &memSourceNet{byResource: map[string][]unstructured.Unstructured{
		"pods": {makePod("backend", "a-7d")},
	}}
	p := SnapshotProposer{}
	got, _ := p.ProposeForNamespace(context.Background(), src, "backend")
	if got == nil {
		t.Fatal("expected proposal")
	}
	if strings.Contains(got.PolicyYAML, "ingress-nginx") || strings.Contains(got.PolicyYAML, "kong") {
		t.Errorf("no-Ingress ns should NOT emit controller allows; got:\n%s", got.PolicyYAML)
	}
	if !strings.Contains(got.PolicyYAML, "podSelector: {}") {
		t.Errorf("same-namespace allow must still be present; got:\n%s", got.PolicyYAML)
	}
}

func TestSnapshotProposer_UnknownIngressClassFallsBackToKong(t *testing.T) {
	src := &memSourceNet{byResource: map[string][]unstructured.Unstructured{
		"pods":      {makePod("app", "a-7d")},
		"ingresses": {makeIngress("app", "a-ing", "exotic-controller")},
	}}
	p := SnapshotProposer{}
	got, _ := p.ProposeForNamespace(context.Background(), src, "app")
	if got == nil {
		t.Fatal("expected proposal")
	}
	if !strings.Contains(got.PolicyYAML, `values: ["kong"]`) {
		t.Errorf("unknown ingressClass should fall back to kong; got:\n%s", got.PolicyYAML)
	}
}

func TestSnapshotProposer_NginxIngressDetected(t *testing.T) {
	src := &memSourceNet{byResource: map[string][]unstructured.Unstructured{
		"pods":      {makePod("app", "a-7d")},
		"ingresses": {makeIngress("app", "a-ing", "nginx")},
	}}
	p := SnapshotProposer{}
	got, _ := p.ProposeForNamespace(context.Background(), src, "app")
	if got == nil || !strings.Contains(got.PolicyYAML, `values: ["ingress-nginx"]`) {
		t.Errorf("nginx ingressClass should map to ingress-nginx; got:\n%v", got)
	}
}

func TestSnapshotProposer_RationaleIncludesContext(t *testing.T) {
	src := &memSourceNet{byResource: map[string][]unstructured.Unstructured{
		"pods":      {makePod("app", "a-7d")},
		"ingresses": {makeIngress("app", "a-ing", "kong")},
	}}
	p := SnapshotProposer{}
	got, _ := p.ProposeForNamespace(context.Background(), src, "app")
	if !strings.Contains(got.Rationale, "default-deny") {
		t.Errorf("Rationale must mention default-deny; got %q", got.Rationale)
	}
	if !strings.Contains(got.Rationale, "kong") {
		t.Errorf("Rationale must mention kong when Kong Ingress detected; got %q", got.Rationale)
	}
}

// Compile-time guard: SnapshotProposer + NoopProposer implement Proposer.
var (
	_ Proposer = SnapshotProposer{}
	_ Proposer = NoopProposer{}
)
