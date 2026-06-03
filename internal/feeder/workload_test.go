// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package feeder_test

import (
	"context"
	"errors"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/feeder"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/rag"
	pkgsnapshot "github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/snapshot"
)

// memSource is a minimal in-memory snapshot.Source for feeder tests.
// Keyed by GVR.Resource ("deployments", "pods", ...) to match the
// pattern used in internal/diagnose/*_test.go.
type memSource struct {
	byResource map[string][]unstructured.Unstructured
}

func (m *memSource) List(_ context.Context, gvr schema.GroupVersionResource, ns string) (*unstructured.UnstructuredList, error) {
	out := &unstructured.UnstructuredList{}
	for _, u := range m.byResource[gvr.Resource] {
		if ns != "" && u.GetNamespace() != ns {
			continue
		}
		out.Items = append(out.Items, u)
	}
	return out, nil
}

func (m *memSource) Get(_ context.Context, gvr schema.GroupVersionResource, ns, name string) (*unstructured.Unstructured, error) {
	for _, u := range m.byResource[gvr.Resource] {
		if u.GetNamespace() == ns && u.GetName() == name {
			return &u, nil
		}
	}
	return nil, nil
}

func (m *memSource) Mode() pkgsnapshot.Mode { return pkgsnapshot.ModeLive }

// captureWriter records every Upsert in-order. Mirrors the contract
// of cha-com's rag_qdrant.go (idempotent upsert, never errors except
// on degenerate input) so the feeder doesn't see any backend-specific
// behaviour.
type captureWriter struct {
	upserts []rag.Entry
	failOn  string // when non-empty, fail upserts whose Key contains this substring
}

func (c *captureWriter) Upsert(_ context.Context, e rag.Entry) error {
	if c.failOn != "" && contains(e.Key, c.failOn) {
		return errors.New("captureWriter: forced fail")
	}
	c.upserts = append(c.upserts, e)
	return nil
}

func (c *captureWriter) AppendSignal(_ context.Context, _ rag.EntryKind, _ string, _ rag.SignalEvent) error {
	return nil
}

func (c *captureWriter) Decay(_ context.Context) error { return nil }

func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

func makeDeployment(ns, name string, replicas int64, containers []map[string]any) unstructured.Unstructured {
	u := unstructured.Unstructured{}
	u.SetAPIVersion("apps/v1")
	u.SetKind("Deployment")
	u.SetNamespace(ns)
	u.SetName(name)
	cs := make([]any, len(containers))
	for i, c := range containers {
		cs[i] = c
	}
	_ = unstructured.SetNestedSlice(u.Object, cs, "spec", "template", "spec", "containers")
	_ = unstructured.SetNestedField(u.Object, replicas, "spec", "replicas")
	return u
}

func makeStatefulSet(ns, name string, containers []map[string]any) unstructured.Unstructured {
	u := unstructured.Unstructured{}
	u.SetAPIVersion("apps/v1")
	u.SetKind("StatefulSet")
	u.SetNamespace(ns)
	u.SetName(name)
	cs := make([]any, len(containers))
	for i, c := range containers {
		cs[i] = c
	}
	_ = unstructured.SetNestedSlice(u.Object, cs, "spec", "template", "spec", "containers")
	return u
}

func makeDaemonSet(ns, name string, containers []map[string]any) unstructured.Unstructured {
	u := unstructured.Unstructured{}
	u.SetAPIVersion("apps/v1")
	u.SetKind("DaemonSet")
	u.SetNamespace(ns)
	u.SetName(name)
	cs := make([]any, len(containers))
	for i, c := range containers {
		cs[i] = c
	}
	_ = unstructured.SetNestedSlice(u.Object, cs, "spec", "template", "spec", "containers")
	return u
}

func makePodWithStatus(ns, name string, containerStatuses []map[string]any) unstructured.Unstructured {
	u := unstructured.Unstructured{}
	u.SetAPIVersion("v1")
	u.SetKind("Pod")
	u.SetNamespace(ns)
	u.SetName(name)
	cs := make([]any, len(containerStatuses))
	for i, c := range containerStatuses {
		cs[i] = c
	}
	_ = unstructured.SetNestedSlice(u.Object, cs, "status", "containerStatuses")
	return u
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestWorkloadFeeder_NilGuards(t *testing.T) {
	var nilFeeder *feeder.WorkloadFeeder
	if _, err := nilFeeder.RunOnce(context.Background()); err == nil {
		t.Error("nil receiver should error")
	}

	f1 := &feeder.WorkloadFeeder{Source: &memSource{}, Writer: nil}
	if _, err := f1.RunOnce(context.Background()); err == nil {
		t.Error("nil Writer should error")
	}

	f2 := &feeder.WorkloadFeeder{Source: nil, Writer: &captureWriter{}}
	if _, err := f2.RunOnce(context.Background()); err == nil {
		t.Error("nil Source should error")
	}
}

func TestWorkloadFeeder_HappyPath_DeploymentWithDigestPin(t *testing.T) {
	src := &memSource{byResource: map[string][]unstructured.Unstructured{
		"deployments": {makeDeployment("production", "cha-com", 2, []map[string]any{
			{"name": "cha-com", "image": "docker4zerocool/cha-com:1.10.0"},
		})},
		"pods": {makePodWithStatus("production", "cha-com-abc", []map[string]any{
			{"name": "cha-com", "imageID": "docker.io/docker4zerocool/cha-com@sha256:abc123def456"},
		})},
	}}
	w := &captureWriter{}
	f := feeder.NewWorkloadFeeder(src, w, "bionic-cluster")

	res, err := f.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if res.Observed != 1 || res.Upserts != 1 {
		t.Errorf("counts: got Observed=%d Upserts=%d, want 1 / 1", res.Observed, res.Upserts)
	}
	if len(w.upserts) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(w.upserts))
	}
	e := w.upserts[0]
	if e.Kind != rag.KindWorkload {
		t.Errorf("kind: got %v want workload", e.Kind)
	}
	if e.Key != "production/cha-com" {
		t.Errorf("key: got %q want production/cha-com", e.Key)
	}
	if e.ClusterID != "bionic-cluster" {
		t.Errorf("clusterID: got %q want bionic-cluster", e.ClusterID)
	}
	if e.Features["kind"] != "Deployment" {
		t.Errorf("features.kind: got %v want Deployment", e.Features["kind"])
	}
	if e.Features["replicas"] != int64(2) {
		t.Errorf("features.replicas: got %v want 2", e.Features["replicas"])
	}
	containers, ok := e.Features["containers"].([]any)
	if !ok || len(containers) != 1 {
		t.Fatalf("features.containers: shape %T len %d", e.Features["containers"], len(containers))
	}
	c := containers[0].(map[string]any)
	if c["name"] != "cha-com" {
		t.Errorf("container.name: got %v", c["name"])
	}
	if c["image"] != "docker4zerocool/cha-com:1.10.0" {
		t.Errorf("container.image: got %v", c["image"])
	}
	if c["image_digest"] != "sha256:abc123def456" {
		t.Errorf("container.image_digest: got %v want sha256:abc123def456", c["image_digest"])
	}
}

func TestWorkloadFeeder_PodNotRunning_NoDigest(t *testing.T) {
	// Workload exists but no Pod has reported imageID yet (still pulling
	// or ImagePullBackOff). The entry must still upsert; image_digest
	// is just omitted from that container.
	src := &memSource{byResource: map[string][]unstructured.Unstructured{
		"deployments": {makeDeployment("staging", "freshly-created", 1, []map[string]any{
			{"name": "app", "image": "myorg/app:v0.1"},
		})},
		// no pods
	}}
	w := &captureWriter{}
	f := feeder.NewWorkloadFeeder(src, w, "c")

	_, err := f.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if len(w.upserts) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(w.upserts))
	}
	c := w.upserts[0].Features["containers"].([]any)[0].(map[string]any)
	if _, hasDigest := c["image_digest"]; hasDigest {
		t.Errorf("image_digest should be absent when no pod reports imageID; got %v", c["image_digest"])
	}
}

func TestWorkloadFeeder_MultipleControllerKinds(t *testing.T) {
	src := &memSource{byResource: map[string][]unstructured.Unstructured{
		"deployments": {makeDeployment("prod", "web", 3, []map[string]any{
			{"name": "web", "image": "nginx:1.27"},
		})},
		"statefulsets": {makeStatefulSet("prod", "db", []map[string]any{
			{"name": "db", "image": "postgres:17"},
		})},
		"daemonsets": {makeDaemonSet("prod", "logger", []map[string]any{
			{"name": "fluentbit", "image": "fluent/fluent-bit:3.0"},
		})},
	}}
	w := &captureWriter{}
	f := feeder.NewWorkloadFeeder(src, w, "c")

	res, _ := f.RunOnce(context.Background())
	if res.Observed != 3 || res.Upserts != 3 {
		t.Errorf("counts: Observed=%d Upserts=%d, want 3/3", res.Observed, res.Upserts)
	}
	gotKinds := map[string]bool{}
	for _, e := range w.upserts {
		gotKinds[e.Features["kind"].(string)] = true
	}
	for _, k := range []string{"Deployment", "StatefulSet", "DaemonSet"} {
		if !gotKinds[k] {
			t.Errorf("missing kind=%s in upserted entries", k)
		}
	}
}

func TestWorkloadFeeder_SkipsSystemNamespaces(t *testing.T) {
	src := &memSource{byResource: map[string][]unstructured.Unstructured{
		"deployments": {
			makeDeployment("kube-system", "coredns", 2, []map[string]any{
				{"name": "coredns", "image": "registry.k8s.io/coredns:1.11"},
			}),
			makeDeployment("calico-system", "calico-kube-controllers", 1, []map[string]any{
				{"name": "ctlr", "image": "calico/kube-controllers:v3.29"},
			}),
			makeDeployment("user-ns", "app", 1, []map[string]any{
				{"name": "app", "image": "myorg/app:v1"},
			}),
		},
	}}
	w := &captureWriter{}
	f := feeder.NewWorkloadFeeder(src, w, "c")

	res, _ := f.RunOnce(context.Background())
	if res.Observed != 1 {
		t.Errorf("Observed: got %d, want 1 (skipped kube-system + calico-system)", res.Observed)
	}
	if len(w.upserts) != 1 || w.upserts[0].Key != "user-ns/app" {
		t.Errorf("unexpected upserts: %+v", w.upserts)
	}
}

func TestWorkloadFeeder_HelmAnnotations_PopulateOwnerFields(t *testing.T) {
	d := makeDeployment("production", "cha-com", 2, []map[string]any{
		{"name": "cha-com", "image": "docker4zerocool/cha-com:1.10.0"},
	})
	d.SetAnnotations(map[string]string{
		"meta.helm.sh/release-name":      "cha",
		"meta.helm.sh/release-namespace": "cluster-health-autopilot",
	})
	d.SetLabels(map[string]string{
		"helm.sh/chart":          "cluster-health-autopilot-1.16.0",
		"app.kubernetes.io/name": "cluster-health-autopilot",
	})
	src := &memSource{byResource: map[string][]unstructured.Unstructured{
		"deployments": {d},
	}}
	w := &captureWriter{}
	f := feeder.NewWorkloadFeeder(src, w, "c")

	_, _ = f.RunOnce(context.Background())
	if len(w.upserts) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(w.upserts))
	}
	feats := w.upserts[0].Features
	if feats["owner_kind"] != "Helm" {
		t.Errorf("owner_kind: got %v want Helm", feats["owner_kind"])
	}
	if feats["owner_release"] != "cha" {
		t.Errorf("owner_release: got %v want cha", feats["owner_release"])
	}
	if feats["owner_release_namespace"] != "cluster-health-autopilot" {
		t.Errorf("owner_release_namespace: got %v", feats["owner_release_namespace"])
	}
	if feats["owner_chart"] != "cluster-health-autopilot" {
		t.Errorf("owner_chart: should strip trailing version; got %v", feats["owner_chart"])
	}
}

func TestWorkloadFeeder_ArgoCDAnnotation_PopulatesOwner(t *testing.T) {
	d := makeDeployment("apps", "checkout", 1, []map[string]any{
		{"name": "checkout", "image": "myorg/checkout:v3"},
	})
	d.SetAnnotations(map[string]string{
		"argocd.argoproj.io/instance": "argocd_checkout-app",
	})
	src := &memSource{byResource: map[string][]unstructured.Unstructured{
		"deployments": {d},
	}}
	w := &captureWriter{}
	f := feeder.NewWorkloadFeeder(src, w, "c")
	_, _ = f.RunOnce(context.Background())

	feats := w.upserts[0].Features
	if feats["owner_kind"] != "ArgoCD" {
		t.Errorf("owner_kind: got %v want ArgoCD", feats["owner_kind"])
	}
	if feats["owner_release"] != "checkout-app" {
		t.Errorf("owner_release: got %v want checkout-app", feats["owner_release"])
	}
	if feats["owner_release_namespace"] != "argocd" {
		t.Errorf("owner_release_namespace: got %v want argocd", feats["owner_release_namespace"])
	}
}

func TestWorkloadFeeder_NoAnnotations_OmitsOwner(t *testing.T) {
	d := makeDeployment("ns", "raw", 1, []map[string]any{
		{"name": "c", "image": "i:t"},
	})
	src := &memSource{byResource: map[string][]unstructured.Unstructured{
		"deployments": {d},
	}}
	w := &captureWriter{}
	f := feeder.NewWorkloadFeeder(src, w, "c")
	_, _ = f.RunOnce(context.Background())

	feats := w.upserts[0].Features
	if _, has := feats["owner_kind"]; has {
		t.Errorf("owner_kind should be omitted without annotations; got %v", feats["owner_kind"])
	}
}

func TestWorkloadFeeder_MultipleContainers(t *testing.T) {
	src := &memSource{byResource: map[string][]unstructured.Unstructured{
		"deployments": {makeDeployment("ns", "multi", 1, []map[string]any{
			{"name": "app", "image": "myorg/app:v1"},
			{"name": "sidecar", "image": "myorg/sidecar:v2"},
		})},
		"pods": {makePodWithStatus("ns", "multi-xyz", []map[string]any{
			{"name": "app", "imageID": "myorg/app@sha256:aaa"},
			// sidecar imageID missing — only app has been pulled
		})},
	}}
	w := &captureWriter{}
	f := feeder.NewWorkloadFeeder(src, w, "c")
	_, _ = f.RunOnce(context.Background())

	containers := w.upserts[0].Features["containers"].([]any)
	if len(containers) != 2 {
		t.Fatalf("expected 2 containers, got %d", len(containers))
	}
	app := containers[0].(map[string]any)
	if app["image_digest"] != "sha256:aaa" {
		t.Errorf("app.image_digest: got %v want sha256:aaa", app["image_digest"])
	}
	sidecar := containers[1].(map[string]any)
	if _, has := sidecar["image_digest"]; has {
		t.Errorf("sidecar.image_digest should be omitted; got %v", sidecar["image_digest"])
	}
}

func TestWorkloadFeeder_DegenerateWorkload_NoContainers(t *testing.T) {
	d := unstructured.Unstructured{}
	d.SetAPIVersion("apps/v1")
	d.SetKind("Deployment")
	d.SetNamespace("ns")
	d.SetName("empty")
	src := &memSource{byResource: map[string][]unstructured.Unstructured{
		"deployments": {d},
	}}
	w := &captureWriter{}
	f := feeder.NewWorkloadFeeder(src, w, "c")
	res, _ := f.RunOnce(context.Background())
	if res.Observed != 1 || res.Upserts != 0 {
		t.Errorf("expected Observed=1 Upserts=0 for empty workload; got %+v", res)
	}
	if len(w.upserts) != 0 {
		t.Errorf("expected no upsert for degenerate workload; got %+v", w.upserts)
	}
}

func TestWorkloadFeeder_WriterError_DoesNotAbortSweep(t *testing.T) {
	src := &memSource{byResource: map[string][]unstructured.Unstructured{
		"deployments": {
			makeDeployment("ns", "fails", 1, []map[string]any{{"name": "c", "image": "i:t"}}),
			makeDeployment("ns", "succeeds", 1, []map[string]any{{"name": "c", "image": "i:t"}}),
		},
	}}
	w := &captureWriter{failOn: "fails"}
	f := feeder.NewWorkloadFeeder(src, w, "c")
	res, err := f.RunOnce(context.Background())
	if err != nil {
		t.Errorf("RunOnce should not propagate per-entry writer errors; got %v", err)
	}
	if res.Observed != 2 || res.Upserts != 1 {
		t.Errorf("counts: Observed=%d Upserts=%d, want 2/1 (one failed)", res.Observed, res.Upserts)
	}
}

func TestWorkloadFeeder_DigestExtractionVariants(t *testing.T) {
	cases := map[string]struct {
		imageID string
		want    string
	}{
		"docker-io":        {"docker.io/library/redis@sha256:abc", "sha256:abc"},
		"docker-pullable":  {"docker-pullable://reg.example/foo@sha256:def", "sha256:def"},
		"private-registry": {"registry.internal/team/svc@sha256:123abc", "sha256:123abc"},
		"no-digest":        {"docker.io/library/redis:7.2", ""},
		"empty":            {"", ""},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			src := &memSource{byResource: map[string][]unstructured.Unstructured{
				"deployments": {makeDeployment("ns", "x", 1, []map[string]any{
					{"name": "c", "image": "x:tag"},
				})},
				"pods": {makePodWithStatus("ns", "p", []map[string]any{
					{"name": "c", "imageID": c.imageID},
				})},
			}}
			w := &captureWriter{}
			f := feeder.NewWorkloadFeeder(src, w, "cid")
			_, _ = f.RunOnce(context.Background())

			cont := w.upserts[0].Features["containers"].([]any)[0].(map[string]any)
			got, _ := cont["image_digest"].(string)
			if got != c.want {
				t.Errorf("digest extract: got %q want %q", got, c.want)
			}
		})
	}
}

func TestWorkloadFeeder_DefaultImportance(t *testing.T) {
	src := &memSource{byResource: map[string][]unstructured.Unstructured{
		"deployments": {makeDeployment("ns", "x", 1, []map[string]any{
			{"name": "c", "image": "i:t"},
		})},
	}}
	w := &captureWriter{}
	f := &feeder.WorkloadFeeder{Source: src, Writer: w, ClusterID: "c"}
	// MinImportance left zero — feeder must apply the default.
	_, _ = f.RunOnce(context.Background())
	if w.upserts[0].Importance != 0.5 {
		t.Errorf("default importance: got %v want 0.5", w.upserts[0].Importance)
	}
}
