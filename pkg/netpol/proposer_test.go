// Copyright 2026 Agentic SRE contributors
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

// makeService builds a core/v1 Service with the given type and one
// port. Used for the v1.13.0 LoadBalancer-aware proposer tests.
func makeService(ns, name, svcType string, port int64, proto string) unstructured.Unstructured {
	u := unstructured.Unstructured{}
	u.SetAPIVersion("v1")
	u.SetKind("Service")
	u.SetNamespace(ns)
	u.SetName(name)
	_ = unstructured.SetNestedField(u.Object, svcType, "spec", "type")
	portMap := map[string]any{
		"port":     port,
		"protocol": proto,
	}
	_ = unstructured.SetNestedSlice(u.Object, []any{portMap}, "spec", "ports")
	return u
}

// TestSnapshotProposer_v1_13_AlwaysIncludesKubeSystemAllow — without
// this allow, CoreDNS / kubelet probes break on policy-enforcing CNIs.
// Default-on; pre-1.13.0 omission was implicit (could break apps).
func TestSnapshotProposer_v1_13_AlwaysIncludesKubeSystemAllow(t *testing.T) {
	src := &memSourceNet{byResource: map[string][]unstructured.Unstructured{
		"pods": {makePod("app", "a-7d")},
	}}
	p := SnapshotProposer{}
	got, _ := p.ProposeForNamespace(context.Background(), src, "app")
	if got == nil {
		t.Fatal("expected proposal")
	}
	if !strings.Contains(got.PolicyYAML, `values: ["kube-system"]`) {
		t.Errorf("YAML must include kube-system allow; got:\n%s", got.PolicyYAML)
	}
	foundKubeSystemAllow := false
	for _, a := range got.AllowSources {
		if a.Namespace == "kube-system" {
			foundKubeSystemAllow = true
			break
		}
	}
	if !foundKubeSystemAllow {
		t.Errorf("AllowSources must include kube-system; got %+v", got.AllowSources)
	}
}

// TestSnapshotProposer_v1_13_LoadBalancerEmitsExternalAllow — closes
// the 2026-06-01 outage class. When a Service of type LoadBalancer
// exists, external (0.0.0.0/0) traffic on its ports MUST be allowed
// or policy-enforcing CNIs will silently DROP external requests.
func TestSnapshotProposer_v1_13_LoadBalancerEmitsExternalAllow(t *testing.T) {
	src := &memSourceNet{byResource: map[string][]unstructured.Unstructured{
		"pods":     {makePod("kong", "kong-pod")},
		"services": {makeService("kong", "kong-proxy", "LoadBalancer", 80, "TCP")},
	}}
	p := SnapshotProposer{}
	got, _ := p.ProposeForNamespace(context.Background(), src, "kong")
	if got == nil {
		t.Fatal("expected proposal")
	}
	if !strings.Contains(got.PolicyYAML, "ipBlock:\n            cidr: 0.0.0.0/0") {
		t.Errorf("LoadBalancer ns must include 0.0.0.0/0 ipBlock allow; got:\n%s", got.PolicyYAML)
	}
	if !strings.Contains(got.PolicyYAML, "port: 80") {
		t.Errorf("port-specific allow missing for LoadBalancer port 80; got:\n%s", got.PolicyYAML)
	}
	foundExtAllow := false
	for _, a := range got.AllowSources {
		if a.Kind == "external-ipblock" {
			foundExtAllow = true
			break
		}
	}
	if !foundExtAllow {
		t.Errorf("AllowSources must include external-ipblock; got %+v", got.AllowSources)
	}
}

// TestSnapshotProposer_v1_13_NodePortAlsoEmitsExternalAllow — same
// rule applies to NodePort services (external traffic via node IP +
// nodePort).
func TestSnapshotProposer_v1_13_NodePortAlsoEmitsExternalAllow(t *testing.T) {
	src := &memSourceNet{byResource: map[string][]unstructured.Unstructured{
		"pods":     {makePod("app", "x-7d")},
		"services": {makeService("app", "app-np", "NodePort", 8080, "TCP")},
	}}
	p := SnapshotProposer{}
	got, _ := p.ProposeForNamespace(context.Background(), src, "app")
	if !strings.Contains(got.PolicyYAML, "ipBlock:\n            cidr: 0.0.0.0/0") {
		t.Errorf("NodePort ns must include 0.0.0.0/0 ipBlock allow; got:\n%s", got.PolicyYAML)
	}
}

// TestSnapshotProposer_v1_13_ClusterIPDoesNOTEmitExternalAllow —
// ClusterIP services are in-cluster only; no external traffic, no
// ipBlock rule. Don't over-permit.
func TestSnapshotProposer_v1_13_ClusterIPDoesNOTEmitExternalAllow(t *testing.T) {
	src := &memSourceNet{byResource: map[string][]unstructured.Unstructured{
		"pods":     {makePod("app", "x-7d")},
		"services": {makeService("app", "app-cip", "ClusterIP", 8080, "TCP")},
	}}
	p := SnapshotProposer{}
	got, _ := p.ProposeForNamespace(context.Background(), src, "app")
	if strings.Contains(got.PolicyYAML, "0.0.0.0/0") {
		t.Errorf("ClusterIP-only ns should NOT emit 0.0.0.0/0 allow; got:\n%s", got.PolicyYAML)
	}
}

// TestSnapshotProposer_v1_13_MultiPortLoadBalancerEmitsAllPorts —
// Kong-style LB with 80 + 443 + 5432 must emit all three port allows.
func TestSnapshotProposer_v1_13_MultiPortLoadBalancerEmitsAllPorts(t *testing.T) {
	// Multi-port service: build manually since makeService takes one port.
	u := unstructured.Unstructured{}
	u.SetAPIVersion("v1")
	u.SetKind("Service")
	u.SetNamespace("kong")
	u.SetName("kong-multi")
	_ = unstructured.SetNestedField(u.Object, "LoadBalancer", "spec", "type")
	_ = unstructured.SetNestedSlice(u.Object, []any{
		map[string]any{"port": int64(80), "protocol": "TCP"},
		map[string]any{"port": int64(443), "protocol": "TCP"},
		map[string]any{"port": int64(5432), "protocol": "TCP"},
	}, "spec", "ports")

	src := &memSourceNet{byResource: map[string][]unstructured.Unstructured{
		"pods":     {makePod("kong", "kong-pod")},
		"services": {u},
	}}
	p := SnapshotProposer{}
	got, _ := p.ProposeForNamespace(context.Background(), src, "kong")
	for _, p := range []string{"port: 80", "port: 443", "port: 5432"} {
		if !strings.Contains(got.PolicyYAML, p) {
			t.Errorf("multi-port LoadBalancer must emit %q; got:\n%s", p, got.PolicyYAML)
		}
	}
}

// TestSnapshotProposer_v1_13_OptOutDisablesKubeSystemAllow — operator
// can disable the kube-system allow for hardened-air-gapped setups.
func TestSnapshotProposer_v1_13_OptOutDisablesKubeSystemAllow(t *testing.T) {
	f := false
	src := &memSourceNet{byResource: map[string][]unstructured.Unstructured{
		"pods": {makePod("app", "x-7d")},
	}}
	p := SnapshotProposer{IncludeSystemNamespaceAllow: &f}
	got, _ := p.ProposeForNamespace(context.Background(), src, "app")
	if strings.Contains(got.PolicyYAML, `values: ["kube-system"]`) {
		t.Errorf("with opt-out, kube-system allow should NOT appear; got:\n%s", got.PolicyYAML)
	}
}

// TestSnapshotProposer_v1_13_OptOutDisablesExternalAllow — operator
// can disable the 0.0.0.0/0 rule (rare; only for clusters where
// external traffic is gated upstream of NetworkPolicy).
func TestSnapshotProposer_v1_13_OptOutDisablesExternalAllow(t *testing.T) {
	f := false
	src := &memSourceNet{byResource: map[string][]unstructured.Unstructured{
		"pods":     {makePod("kong", "kong-pod")},
		"services": {makeService("kong", "kong-proxy", "LoadBalancer", 80, "TCP")},
	}}
	p := SnapshotProposer{IncludeLoadBalancerExternalAllow: &f}
	got, _ := p.ProposeForNamespace(context.Background(), src, "kong")
	if strings.Contains(got.PolicyYAML, "0.0.0.0/0") {
		t.Errorf("with opt-out, external 0.0.0.0/0 allow should NOT appear; got:\n%s", got.PolicyYAML)
	}
}

// Compile-time guard: SnapshotProposer + NoopProposer implement Proposer.
var (
	_ Proposer = SnapshotProposer{}
	_ Proposer = NoopProposer{}
)
