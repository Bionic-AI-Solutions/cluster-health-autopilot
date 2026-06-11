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

// makePodWithContainersAndStatus is makePodWithContainers plus a
// `status.containerStatuses[]` block. imageIDs maps container name →
// status.containerStatuses[].imageID (the runtime-resolved digest in
// `registry/image@sha256:...` form, possibly with the `docker-pullable://`
// kubelet prefix).
func makePodWithContainersAndStatus(ns, name string, containerImages, imageIDs map[string]string) unstructured.Unstructured {
	u := makePodWithContainers(ns, name, containerImages)
	if len(imageIDs) == 0 {
		return u
	}
	statuses := make([]interface{}, 0, len(imageIDs))
	for cn, iid := range imageIDs {
		img := containerImages[cn]
		statuses = append(statuses, map[string]interface{}{
			"name":    cn,
			"image":   img,
			"imageID": iid,
		})
	}
	_ = unstructured.SetNestedSlice(u.Object, statuses, "status", "containerStatuses")
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

func TestSecurityDrift_PodWithMutableTag_InfoOnTrustedRegistry(t *testing.T) {
	// v1.14.0: trusted-upstream registries (ghcr.io, quay.io, gcr.io,
	// registry.k8s.io, plus canonical docker.io official images) get
	// severity=info instead of warning. Operator can override via
	// CHA_DIGEST_PIN_UNTRUSTED_SEVERITY env.
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
	if got[0].Severity != "info" {
		t.Errorf("trusted-upstream ghcr.io image should be info; got severity=%q", got[0].Severity)
	}
	if !strings.Contains(got[0].Message, "without digest pin") {
		t.Errorf("message lacks 'without digest pin': %s", got[0].Message)
	}
	if !strings.Contains(got[0].Message, "v1.2.3") {
		t.Errorf("message should name the offending image: %s", got[0].Message)
	}
}

// TestSecurityDrift_PodWithMutableTag_WarningOnInHouseRegistry —
// in-house images (e.g., the team's own Docker Hub namespace) stay
// at warning because they're where REAL supply-chain pinning matters.
func TestSecurityDrift_PodWithMutableTag_WarningOnInHouseRegistry(t *testing.T) {
	src := secureBaseline()
	src.byResource["pods"] = []unstructured.Unstructured{
		makePodWithContainers("app", "x-1", map[string]string{
			"main": "docker4zerocool/my-app:v1.2.3", // in-house, not trusted-upstream
		}),
	}
	got := SecurityDrift{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("expected 1 diagnostic; got %d", len(got))
	}
	if got[0].Severity != "warning" {
		t.Errorf("in-house image should be warning; got severity=%q", got[0].Severity)
	}
}

// TestSecurityDrift_PodWithMixedImages_WarningEscalates — when a pod
// has BOTH an upstream image AND an in-house image, the diagnostic
// goes to warning (any single untrusted image escalates the pod).
func TestSecurityDrift_PodWithMixedImages_WarningEscalates(t *testing.T) {
	src := secureBaseline()
	src.byResource["pods"] = []unstructured.Unstructured{
		makePodWithContainers("app", "x-1", map[string]string{
			"trusted":  "ghcr.io/org/sidecar:v0.5",  // trusted-upstream → would be info alone
			"in-house": "docker4zerocool/main:v1.0", // in-house → escalates to warning
		}),
	}
	got := SecurityDrift{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("expected 1 diagnostic; got %d", len(got))
	}
	if got[0].Severity != "warning" {
		t.Errorf("mixed pod should escalate to warning; got severity=%q", got[0].Severity)
	}
}

// TestClassifyDigestPinSeverity exhaustively covers the trust matrix.
func TestClassifyDigestPinSeverity(t *testing.T) {
	cases := map[string]string{
		// Trusted upstream — info
		"ghcr.io/org/app:v1":                 "info",
		"quay.io/operator/foo:1.2":           "info",
		"gcr.io/distroless/static:latest":    "info",
		"registry.k8s.io/pause:3.9":          "info",
		"k8s.gcr.io/pause:3.9":               "info",
		"docker.io/postgres:17":              "info",
		"postgres:17":                        "info", // implicit docker.io
		"docker.io/redis:7-alpine":           "info",
		"redis:7-alpine":                     "info",
		"docker.io/haproxy:2.8-alpine":       "info",
		"docker.io/envoyproxy/envoy:v1.30":   "info",
		"public.ecr.aws/lambda/python:3.11":  "info",
		"mcr.microsoft.com/dotnet/runtime:8": "info",
		// In-house / unknown — warning
		"docker4zerocool/my-app:v1.0":         "warning",
		"someorg/myapp:v1.0":                  "warning",
		"docker.io/someorg/myapp:v1.0":        "warning",
		"my-private-registry.example.com/x:1": "warning",
	}
	for img, want := range cases {
		if got := classifyDigestPinSeverity(img); got != want {
			t.Errorf("classify(%q) = %q; want %q", img, got, want)
		}
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

// --- Digest-pin remediation: substitute the observed digest (Phase 1.B.2) ---
//
// The legacy remediation contained literal `<digest>` / `<image>:<tag>`
// tokens the operator was expected to fill in by hand. The AI tier
// reading the diagnostic has no way to interpret those tokens, and
// operators reading Slack alerts have no way to act without `crane
// digest` round-trips. Kubelet has already resolved every running
// image to a digest and stamped it on `status.containerStatuses[].imageID`
// — substitute that directly.

func TestSecurityDrift_DigestPinRemediation_SubstitutesObservedDigest(t *testing.T) {
	src := secureBaseline()
	src.byResource["pods"] = []unstructured.Unstructured{
		makePodWithContainersAndStatus(
			"app", "x-1",
			map[string]string{"main": "docker4zerocool/cha-com:1.10.0"},
			// kubelet's standard imageID shape: docker-pullable://<repo>@sha256:<hex>
			map[string]string{"main": "docker-pullable://docker4zerocool/cha-com@sha256:abc123def456789012345678901234567890abcd"},
		),
	}
	got := SecurityDrift{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("expected 1 diagnostic; got %d: %+v", len(got), got)
	}
	rem := got[0].Remediation
	// The placeholder tokens must NOT leak.
	for _, tok := range []string{"<digest>", "<image>:<tag>"} {
		if strings.Contains(rem, tok) {
			t.Errorf("remediation must not contain literal %q; got: %s", tok, rem)
		}
	}
	// The actual digest MUST appear, and the `docker-pullable://` prefix
	// MUST be stripped (operators paste the digest into manifest files,
	// not the kubelet prefix).
	if !strings.Contains(rem, "sha256:abc123def456789012345678901234567890abcd") {
		t.Errorf("remediation should embed the observed digest; got: %s", rem)
	}
	if strings.Contains(rem, "docker-pullable://") {
		t.Errorf("remediation must strip the docker-pullable:// prefix from imageID; got: %s", rem)
	}
	// And the original image:tag must appear so operators see what to replace.
	if !strings.Contains(rem, "docker4zerocool/cha-com:1.10.0") {
		t.Errorf("remediation should name the original tag-pinned reference; got: %s", rem)
	}
}

func TestSecurityDrift_DigestPinRemediation_MultipleContainers_SubstitutesAll(t *testing.T) {
	src := secureBaseline()
	src.byResource["pods"] = []unstructured.Unstructured{
		makePodWithContainersAndStatus(
			"app", "x-1",
			map[string]string{
				"main":    "docker4zerocool/cha-com:1.10.0",
				"sidecar": "docker4zerocool/sidecar:0.9",
			},
			map[string]string{
				"main":    "docker4zerocool/cha-com@sha256:1111111111111111111111111111111111111111111111111111111111111111",
				"sidecar": "docker4zerocool/sidecar@sha256:2222222222222222222222222222222222222222222222222222222222222222",
			},
		),
	}
	got := SecurityDrift{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("expected 1 diagnostic; got %d", len(got))
	}
	rem := got[0].Remediation
	if !strings.Contains(rem, "1111111111111111111111111111111111111111111111111111111111111111") ||
		!strings.Contains(rem, "2222222222222222222222222222222222222222222222222222222222222222") {
		t.Errorf("multi-container remediation should embed both observed digests; got: %s", rem)
	}
}

func TestSecurityDrift_DigestPinRemediation_NoStatus_FallsBackWithoutLeakingPlaceholders(t *testing.T) {
	// status.containerStatuses missing (pod not yet scheduled, or
	// image not yet pulled) — the remediation must fall back to a
	// concrete command the operator can run, NOT leak `<digest>` /
	// `<image>:<tag>` as bare tokens.
	src := secureBaseline()
	src.byResource["pods"] = []unstructured.Unstructured{
		makePodWithContainers("app", "x-1", map[string]string{
			"main": "docker4zerocool/cha-com:1.10.0",
		}),
	}
	got := SecurityDrift{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("expected 1 diagnostic; got %d", len(got))
	}
	rem := got[0].Remediation
	for _, tok := range []string{"<digest>", "<image>:<tag>"} {
		if strings.Contains(rem, tok) {
			t.Errorf("fallback remediation must not contain literal %q; got: %s", tok, rem)
		}
	}
	// Should reference the actual image to make crane/skopeo invocation
	// directly copy-pasteable.
	if !strings.Contains(rem, "docker4zerocool/cha-com:1.10.0") {
		t.Errorf("fallback remediation should name the actual image; got: %s", rem)
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

// ---- v1.25.0 — per-workload dedup ---------------------------------------

func TestSecurityDrift_DigestPin_DedupesByWorkloadOwner(t *testing.T) {
	// 3 ReplicaSet-owned Pods that share the same image set should
	// collapse to ONE diagnostic keyed on the ReplicaSet name.
	ctrl := true
	mkPod := func(name string) unstructured.Unstructured {
		return unstructured.Unstructured{Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": "demo",
				"ownerReferences": []interface{}{map[string]interface{}{
					"apiVersion": "apps/v1",
					"kind":       "ReplicaSet",
					"name":       "my-app-6c4c5b5c7b",
					"controller": ctrl,
				}},
			},
			"spec": map[string]interface{}{
				"containers": []interface{}{map[string]interface{}{
					"name":  "app",
					"image": "registry/my-app:v1.0",
				}},
			},
		}}
	}
	src := &memSourceSec{byResource: map[string][]unstructured.Unstructured{
		"pods": {mkPod("my-app-6c4c5b5c7b-aaaaa"), mkPod("my-app-6c4c5b5c7b-bbbbb"), mkPod("my-app-6c4c5b5c7b-ccccc")},
	}}
	got := SecurityDrift{}.checkMutableImageTags(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("expected 1 deduped diagnostic; got %d", len(got))
	}
	want := "Workload/demo/my-app-6c4c5b5c7b"
	if got[0].Subject != want {
		t.Errorf("subject=%q want %q", got[0].Subject, want)
	}
	if !strings.Contains(got[0].Message, "across 3 replica pods") {
		t.Errorf("message should report replica count; got %q", got[0].Message)
	}
}

func TestSecurityDrift_DigestPin_DifferentImageSets_StayDistinct(t *testing.T) {
	// Two ReplicaSets in the same namespace, different image versions
	// (mid-rolling-update). Should emit 2 distinct diagnostics.
	ctrl := true
	mkPod := func(podName, rsName, tag string) unstructured.Unstructured {
		return unstructured.Unstructured{Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]interface{}{
				"name":      podName,
				"namespace": "demo",
				"ownerReferences": []interface{}{map[string]interface{}{
					"apiVersion": "apps/v1",
					"kind":       "ReplicaSet",
					"name":       rsName,
					"controller": ctrl,
				}},
			},
			"spec": map[string]interface{}{
				"containers": []interface{}{map[string]interface{}{
					"name":  "app",
					"image": "registry/my-app:" + tag,
				}},
			},
		}}
	}
	src := &memSourceSec{byResource: map[string][]unstructured.Unstructured{
		"pods": {
			mkPod("my-app-old-aaaaa", "my-app-old-rs", "v1.0"),
			mkPod("my-app-new-bbbbb", "my-app-new-rs", "v2.0"),
		},
	}}
	got := SecurityDrift{}.checkMutableImageTags(context.Background(), src)
	if len(got) != 2 {
		t.Fatalf("expected 2 diagnostics (one per ReplicaSet); got %d", len(got))
	}
}
