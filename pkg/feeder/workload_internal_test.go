// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package feeder

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestImageRepo(t *testing.T) {
	cases := map[string]struct{ in, want string }{
		"hub-official-shorthand":  {"redis", "docker.io/library/redis"},
		"hub-official-tag":        {"redis:7.2", "docker.io/library/redis"},
		"hub-org-shorthand":       {"myorg/app:v1", "docker.io/myorg/app"},
		"hub-explicit":            {"docker.io/myorg/app:v1", "docker.io/myorg/app"},
		"hub-explicit-official":   {"docker.io/redis:7.2", "docker.io/library/redis"},
		"index-docker-io":         {"index.docker.io/myorg/app:v1", "docker.io/myorg/app"},
		"registry-1-docker-io":    {"registry-1.docker.io/library/nginx:1.25", "docker.io/library/nginx"},
		"localhost-port-registry": {"localhost:5000/app:dev", "localhost:5000/app"},
		"imageid-digest":          {"docker.io/library/redis@sha256:abc", "docker.io/library/redis"},
		"imageid-docker-pullable": {"docker-pullable://reg.example/foo@sha256:def", "reg.example/foo"},
		"private-registry":        {"registry.internal/team/svc:v3", "registry.internal/team/svc"},
		"registry-port":           {"reg.example:5000/team/svc:v3", "reg.example:5000/team/svc"},
		"registry-port-no-tag":    {"reg.example:5000/team/svc", "reg.example:5000/team/svc"},
		"localhost":               {"localhost/foo:1", "localhost/foo"},
		"tag-and-digest":          {"reg.example/foo:1@sha256:abc", "reg.example/foo"},
		"empty":                   {"", ""},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if got := imageRepo(c.in); got != c.want {
				t.Errorf("imageRepo(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestPodWorkloadOwner(t *testing.T) {
	ctrl := true
	mkPod := func(labels map[string]string, refs ...metav1.OwnerReference) *unstructured.Unstructured {
		u := &unstructured.Unstructured{Object: map[string]interface{}{
			"apiVersion": "v1", "kind": "Pod",
			"metadata": map[string]interface{}{"name": "p", "namespace": "ns"},
		}}
		u.SetLabels(labels)
		u.SetOwnerReferences(refs)
		return u
	}
	t.Run("deployment-via-replicaset-hash", func(t *testing.T) {
		p := mkPod(map[string]string{"pod-template-hash": "6d5f9c9b8d"},
			metav1.OwnerReference{APIVersion: "apps/v1", Kind: "ReplicaSet", Name: "web-6d5f9c9b8d", Controller: &ctrl})
		kind, name, ok := podWorkloadOwner(p)
		if !ok || kind != "Deployment" || name != "web" {
			t.Errorf("got (%q,%q,%v) want (Deployment,web,true)", kind, name, ok)
		}
	})
	t.Run("daemonset-direct", func(t *testing.T) {
		p := mkPod(nil, metav1.OwnerReference{APIVersion: "apps/v1", Kind: "DaemonSet", Name: "logger", Controller: &ctrl})
		kind, name, ok := podWorkloadOwner(p)
		if !ok || kind != "DaemonSet" || name != "logger" {
			t.Errorf("got (%q,%q,%v) want (DaemonSet,logger,true)", kind, name, ok)
		}
	})
	t.Run("bare-replicaset-no-hash-label", func(t *testing.T) {
		p := mkPod(nil, metav1.OwnerReference{APIVersion: "apps/v1", Kind: "ReplicaSet", Name: "standalone-rs", Controller: &ctrl})
		if _, _, ok := podWorkloadOwner(p); ok {
			t.Error("bare ReplicaSet pod (no pod-template-hash) must not resolve")
		}
	})
	t.Run("job-owner-skipped", func(t *testing.T) {
		p := mkPod(nil, metav1.OwnerReference{APIVersion: "batch/v1", Kind: "Job", Name: "backup", Controller: &ctrl})
		if _, _, ok := podWorkloadOwner(p); ok {
			t.Error("Job-owned pod must not resolve")
		}
	})
	t.Run("non-controller-ref-ignored", func(t *testing.T) {
		p := mkPod(nil, metav1.OwnerReference{APIVersion: "apps/v1", Kind: "DaemonSet", Name: "logger"})
		if _, _, ok := podWorkloadOwner(p); ok {
			t.Error("non-controller ownerRef must not resolve")
		}
	})
	t.Run("no-owner", func(t *testing.T) {
		if _, _, ok := podWorkloadOwner(mkPod(nil)); ok {
			t.Error("ownerless pod must not resolve")
		}
	})
}

func TestDetectOwner_OperatorManaged_v1_25_0(t *testing.T) {
	// Deployment owned by ClusterHealthAutopilot CR (no Helm labels,
	// no ArgoCD annotation) should produce owner_chart synthesized
	// from the CR Kind + name.
	ctrl := true
	obj := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata": map[string]interface{}{
			"name":      "bionic-aiwatch",
			"namespace": "cluster-health-autopilot",
			"ownerReferences": []interface{}{map[string]interface{}{
				"apiVersion": "cha.bionicaisolutions.com/v1alpha1",
				"kind":       "ClusterHealthAutopilot",
				"name":       "bionic",
				"controller": ctrl,
			}},
		},
	}}
	o := detectOwner(obj)
	if o == nil {
		t.Fatal("expected non-nil owner for operator-managed Deployment")
	}
	if o.Kind != "Operator" {
		t.Errorf("Kind=%q want Operator", o.Kind)
	}
	if want := "clusterhealthautopilot-bionic"; o.ChartName != want {
		t.Errorf("ChartName=%q want %q", o.ChartName, want)
	}
	if o.ReleaseName != "bionic" {
		t.Errorf("ReleaseName=%q want bionic", o.ReleaseName)
	}
}

func TestDetectOwner_AppsControllerOwnerRef_StillNil(t *testing.T) {
	// A Deployment owned by apps/v1 ReplicaSet (built-in workload parent)
	// must NOT be treated as operator-managed. The operator-ownerRef
	// rule is specifically for custom-resource parents.
	ctrl := true
	obj := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata": map[string]interface{}{
			"name":      "no-helm-no-argo-no-operator",
			"namespace": "demo",
			"ownerReferences": []interface{}{map[string]interface{}{
				"apiVersion": "apps/v1",
				"kind":       "ReplicaSet",
				"name":       "should-not-match",
				"controller": ctrl,
			}},
		},
	}}
	if o := detectOwner(obj); o != nil {
		t.Errorf("apps/v1 ReplicaSet owner should not synthesize Operator owner; got %+v", o)
	}
}
