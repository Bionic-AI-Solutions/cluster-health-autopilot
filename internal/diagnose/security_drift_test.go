// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package diagnose

import (
	"context"
	"strings"
	"testing"

	pkgsnapshot "github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/snapshot"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type memSourceSec struct {
	byResource map[string][]unstructured.Unstructured
}

func (m *memSourceSec) List(_ context.Context, gvr schema.GroupVersionResource, ns string) (*unstructured.UnstructuredList, error) {
	out := &unstructured.UnstructuredList{}
	for _, u := range m.byResource[gvr.Resource] {
		if ns != "" && u.GetNamespace() != ns {
			continue
		}
		out.Items = append(out.Items, u)
	}
	return out, nil
}

func (m *memSourceSec) Get(_ context.Context, gvr schema.GroupVersionResource, ns, name string) (*unstructured.Unstructured, error) {
	for _, u := range m.byResource[gvr.Resource] {
		if u.GetNamespace() == ns && u.GetName() == name {
			return &u, nil
		}
	}
	return nil, nil
}

func (m *memSourceSec) Mode() pkgsnapshot.Mode { return pkgsnapshot.ModeLive }

func makeNamespace(name string, labels map[string]string) unstructured.Unstructured {
	u := unstructured.Unstructured{}
	u.SetAPIVersion("v1")
	u.SetKind("Namespace")
	u.SetName(name)
	if labels != nil {
		u.SetLabels(labels)
	}
	return u
}

func makePodWithContainers(ns, name string, containerImages map[string]string) unstructured.Unstructured {
	u := unstructured.Unstructured{}
	u.SetAPIVersion("v1")
	u.SetKind("Pod")
	u.SetNamespace(ns)
	u.SetName(name)
	containers := make([]interface{}, 0, len(containerImages))
	for cn, img := range containerImages {
		containers = append(containers, map[string]interface{}{
			"name":  cn,
			"image": img,
		})
	}
	_ = unstructured.SetNestedSlice(u.Object, containers, "spec", "containers")
	return u
}

func makeNetworkPolicy(ns, name string) unstructured.Unstructured {
	u := unstructured.Unstructured{}
	u.SetAPIVersion("networking.k8s.io/v1")
	u.SetKind("NetworkPolicy")
	u.SetNamespace(ns)
	u.SetName(name)
	return u
}

// --- PSS posture ---

func TestSecurityDrift_NamespaceWithoutPSSLabel_Warning(t *testing.T) {
	src := &memSourceSec{byResource: map[string][]unstructured.Unstructured{
		"namespaces": {makeNamespace("app", nil)},
	}}
	got := SecurityDrift{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("expected 1 diagnostic; got %d: %+v", len(got), got)
	}
	if got[0].Severity != "warning" {
		t.Errorf("severity=%s want warning", got[0].Severity)
	}
	if !strings.Contains(got[0].Message, "no pod-security.kubernetes.io/enforce label") {
		t.Errorf("message lacks PSS-label phrasing: %s", got[0].Message)
	}
}

func TestSecurityDrift_NamespacePrivilegedPSS_Warning(t *testing.T) {
	src := &memSourceSec{byResource: map[string][]unstructured.Unstructured{
		"namespaces": {makeNamespace("ai", map[string]string{
			"pod-security.kubernetes.io/enforce": "privileged",
		})},
	}}
	got := SecurityDrift{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("expected 1 diagnostic; got %d: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Message, "PSS=privileged") {
		t.Errorf("message lacks 'PSS=privileged': %s", got[0].Message)
	}
}

func TestSecurityDrift_NamespaceBaselinePSS_Silent(t *testing.T) {
	src := &memSourceSec{byResource: map[string][]unstructured.Unstructured{
		"namespaces": {makeNamespace("app", map[string]string{
			"pod-security.kubernetes.io/enforce": "baseline",
		})},
	}}
	got := SecurityDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("baseline is acceptable; got: %+v", got)
	}
}

func TestSecurityDrift_NamespaceRestrictedPSS_Silent(t *testing.T) {
	src := &memSourceSec{byResource: map[string][]unstructured.Unstructured{
		"namespaces": {makeNamespace("app", map[string]string{
			"pod-security.kubernetes.io/enforce": "restricted",
		})},
	}}
	got := SecurityDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("restricted is the goal; got: %+v", got)
	}
}

func TestSecurityDrift_SystemNamespaceWithoutPSS_Skipped(t *testing.T) {
	// kube-system / vault / etc. are control-plane managed; ignore.
	src := &memSourceSec{byResource: map[string][]unstructured.Unstructured{
		"namespaces": {
			makeNamespace("kube-system", nil),
			makeNamespace("vault", nil),
			makeNamespace("cnpg-system", nil),
		},
	}}
	got := SecurityDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("system namespaces should be skipped; got: %+v", got)
	}
}

// --- Image digest pinning ---

// secureBaseline returns a source pre-populated with a baseline-PSS
// namespace + a NetworkPolicy so PSS-posture and netpol-coverage
// signals are silent. Tests can add Pods to isolate the image-pin
// signal.
func secureBaseline() *memSourceSec {
	return &memSourceSec{byResource: map[string][]unstructured.Unstructured{
		"namespaces":      {makeNamespace("app", map[string]string{"pod-security.kubernetes.io/enforce": "baseline"})},
		"networkpolicies": {makeNetworkPolicy("app", "default-deny")},
	}}
}

func TestSecurityDrift_PodWithDigestPinnedImage_Silent(t *testing.T) {
	src := secureBaseline()
	src.byResource["pods"] = []unstructured.Unstructured{
		makePodWithContainers("app", "x-1", map[string]string{
			"main": "ghcr.io/org/app@sha256:abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789",
		}),
	}
	got := SecurityDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("digest-pinned image is healthy; got: %+v", got)
	}
}

func TestSecurityDrift_PodWithMutableTag_Warning(t *testing.T) {
	src := secureBaseline()
	src.byResource["pods"] = []unstructured.Unstructured{
		makePodWithContainers("app", "x-1", map[string]string{
			"main": "ghcr.io/org/app:v1.2.3",
		}),
	}
	got := SecurityDrift{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("expected 1 diagnostic; got %d: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Message, "without digest pin") {
		t.Errorf("message lacks 'without digest pin': %s", got[0].Message)
	}
	if !strings.Contains(got[0].Message, "v1.2.3") {
		t.Errorf("message should name the offending image: %s", got[0].Message)
	}
}

func TestSecurityDrift_PodWithLatestTag_Silent(t *testing.T) {
	// `:latest` is flagged by other code paths (image-policy / pull-policy);
	// don't double-flag here.
	src := secureBaseline()
	src.byResource["pods"] = []unstructured.Unstructured{
		makePodWithContainers("app", "x-1", map[string]string{
			"main": "ghcr.io/org/app:latest",
		}),
	}
	got := SecurityDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf(":latest is handled elsewhere; got: %+v", got)
	}
}

func TestSecurityDrift_PodWithMultipleContainers_NamesAll(t *testing.T) {
	src := secureBaseline()
	src.byResource["pods"] = []unstructured.Unstructured{
		makePodWithContainers("app", "x-1", map[string]string{
			"main":    "ghcr.io/org/app:v1.2.3",
			"sidecar": "ghcr.io/org/sidecar:0.9",
		}),
	}
	got := SecurityDrift{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("expected 1 diagnostic (one per pod); got %d", len(got))
	}
	// Order is map-iteration-dependent; confirm both image refs land.
	if !strings.Contains(got[0].Message, "v1.2.3") || !strings.Contains(got[0].Message, "0.9") {
		t.Errorf("message should list both containers: %s", got[0].Message)
	}
}

func TestSecurityDrift_PodInSystemNamespace_Skipped(t *testing.T) {
	src := &memSourceSec{byResource: map[string][]unstructured.Unstructured{
		"namespaces": {makeNamespace("kube-system", nil)},
		"pods": {makePodWithContainers("kube-system", "kp-1", map[string]string{
			"main": "k8s.gcr.io/kube-proxy:v1.30.0",
		})},
	}}
	got := SecurityDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("kube-system pods skipped; got: %+v", got)
	}
}

// --- NetworkPolicy coverage ---

func TestSecurityDrift_NamespaceWithPodsNoNetpol_Warning(t *testing.T) {
	// v1.12.0: NetPol coverage warning is gated on CNI enforcement.
	// Fixture includes calico-node DaemonSet so CNI detection sees an
	// enforcing CNI and emits the per-namespace warning. Without this,
	// the analyzer correctly downgrades to a cluster-scope info
	// finding (covered by TestSecurityDrift_NoNetpolOnNonEnforcingCNI_InfoOnly).
	src := &memSourceSec{byResource: map[string][]unstructured.Unstructured{
		"namespaces": {makeNamespace("app", map[string]string{"pod-security.kubernetes.io/enforce": "baseline"})},
		"daemonsets": {makeDaemonSetSec("calico-system", "calico-node")},
		"pods": {makePodWithContainers("app", "x-1", map[string]string{
			"main": "ghcr.io/org/app@sha256:abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789",
		})},
	}}
	got := SecurityDrift{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("expected 1 diagnostic; got %d: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Message, "zero NetworkPolicies") {
		t.Errorf("message lacks 'zero NetworkPolicies': %s", got[0].Message)
	}
	if !strings.Contains(got[0].Remediation, "default-deny") {
		t.Errorf("remediation lacks default-deny guidance: %s", got[0].Remediation)
	}
}

// TestSecurityDrift_NoNetpolOnNonEnforcingCNI_InfoOnly — the v1.12.0
// gate. On Flannel-only / unknown-CNI clusters, even when a namespace
// has pods + no NetPol, the analyzer emits a single info-level
// cluster-scope finding instead of per-namespace warnings. Adding
// NetPols on a non-enforcing CNI is decorative-only.
func TestSecurityDrift_NoNetpolOnNonEnforcingCNI_InfoOnly(t *testing.T) {
	src := &memSourceSec{byResource: map[string][]unstructured.Unstructured{
		"namespaces": {makeNamespace("app", map[string]string{"pod-security.kubernetes.io/enforce": "baseline"})},
		"daemonsets": {makeDaemonSetSec("kube-flannel", "kube-flannel-ds")}, // Flannel-only, doesn't enforce
		"pods": {makePodWithContainers("app", "x-1", map[string]string{
			"main": "ghcr.io/org/app@sha256:abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789",
		})},
	}}
	got := SecurityDrift{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("expected 1 cluster-scope info finding; got %d: %+v", len(got), got)
	}
	if got[0].Severity != "info" {
		t.Errorf("severity=%q want info", got[0].Severity)
	}
	if got[0].Subject != "Cluster/cni-no-netpol-enforcement" {
		t.Errorf("subject=%q want Cluster/cni-no-netpol-enforcement", got[0].Subject)
	}
	if !strings.Contains(got[0].Message, "flannel-only") {
		t.Errorf("message should name flannel-only CNI; got %q", got[0].Message)
	}
}

func makeDaemonSetSec(ns, name string) unstructured.Unstructured {
	u := unstructured.Unstructured{}
	u.SetAPIVersion("apps/v1")
	u.SetKind("DaemonSet")
	u.SetNamespace(ns)
	u.SetName(name)
	return u
}

func TestSecurityDrift_NamespaceWithNetpol_Silent(t *testing.T) {
	src := &memSourceSec{byResource: map[string][]unstructured.Unstructured{
		"namespaces": {makeNamespace("app", map[string]string{"pod-security.kubernetes.io/enforce": "baseline"})},
		"pods": {makePodWithContainers("app", "x-1", map[string]string{
			"main": "ghcr.io/org/app@sha256:abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789",
		})},
		"networkpolicies": {makeNetworkPolicy("app", "default-deny")},
	}}
	got := SecurityDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("namespace with NetPol should be silent; got: %+v", got)
	}
}

func TestSecurityDrift_EmptyNamespaceNoNetpol_Silent(t *testing.T) {
	// A namespace with no pods has nothing to govern — don't flag.
	src := &memSourceSec{byResource: map[string][]unstructured.Unstructured{
		"namespaces": {makeNamespace("app", map[string]string{"pod-security.kubernetes.io/enforce": "baseline"})},
	}}
	got := SecurityDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("empty namespace should be silent; got: %+v", got)
	}
}

func TestSecurityDrift_SystemNamespaceNoNetpol_Skipped(t *testing.T) {
	src := &memSourceSec{byResource: map[string][]unstructured.Unstructured{
		"namespaces": {makeNamespace("kube-system", nil)},
		"pods": {makePodWithContainers("kube-system", "kp-1", map[string]string{
			"main": "k8s.gcr.io/kube-proxy@sha256:abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789",
		})},
	}}
	got := SecurityDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("kube-system NetPol gaps skipped; got: %+v", got)
	}
}

// --- Misc ---

func TestSecurityDrift_RunNoOpOnEmptySource(t *testing.T) {
	src := &memSourceSec{byResource: map[string][]unstructured.Unstructured{}}
	got := SecurityDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("empty source no-op; got: %+v", got)
	}
}

func TestSecurityDrift_NameStable(t *testing.T) {
	if name := (SecurityDrift{}).Name(); name != "SecurityDrift" {
		t.Errorf("Name()=%q want SecurityDrift", name)
	}
}
