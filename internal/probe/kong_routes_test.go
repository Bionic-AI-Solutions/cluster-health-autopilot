// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package probe

import (
	"context"
	"strings"
	"testing"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// memKongSrc is a snapshot.Source double for the KongRoutes probe.
type memKongSrc struct {
	ingresses []unstructured.Unstructured
	endpoints []unstructured.Unstructured
	epSlices  []unstructured.Unstructured
	plugins   []unstructured.Unstructured
	consumers []unstructured.Unstructured

	// listErr can be set per-GVR; tests use it to simulate CRD-absent.
	listErr map[string]error
}

func (m *memKongSrc) List(_ context.Context, gvr schema.GroupVersionResource, _ string) (*unstructured.UnstructuredList, error) {
	if err := m.listErr[gvr.Resource]; err != nil {
		return nil, err
	}
	out := &unstructured.UnstructuredList{}
	switch gvr.Resource {
	case "ingresses":
		out.Items = m.ingresses
	case "endpoints":
		out.Items = m.endpoints
	case "endpointslices":
		out.Items = m.epSlices
	case "kongplugins":
		out.Items = m.plugins
	case "kongconsumers":
		out.Items = m.consumers
	}
	return out, nil
}

func (m *memKongSrc) Get(_ context.Context, _ schema.GroupVersionResource, _, _ string) (*unstructured.Unstructured, error) {
	return nil, nil
}
func (m *memKongSrc) Mode() snapshot.Mode { return snapshot.ModeLive }

func makeIngressKong(ns, name, host, backendSvc string, anns map[string]string) unstructured.Unstructured {
	u := unstructured.Unstructured{Object: map[string]any{}}
	u.SetAPIVersion("networking.k8s.io/v1")
	u.SetKind("Ingress")
	u.SetNamespace(ns)
	u.SetName(name)
	if anns == nil {
		anns = map[string]string{}
	}
	u.SetAnnotations(anns)
	_ = unstructured.SetNestedField(u.Object, "kong", "spec", "ingressClassName")
	rules := []any{
		map[string]any{
			"host": host,
			"http": map[string]any{
				"paths": []any{
					map[string]any{
						"path": "/",
						"backend": map[string]any{
							"service": map[string]any{
								"name": backendSvc,
								"port": map[string]any{"number": int64(80)},
							},
						},
					},
				},
			},
		},
	}
	_ = unstructured.SetNestedSlice(u.Object, rules, "spec", "rules")
	return u
}

func makeEndpointSlice(ns, name, svcName string, ready bool) unstructured.Unstructured {
	u := unstructured.Unstructured{Object: map[string]any{}}
	u.SetAPIVersion("discovery.k8s.io/v1")
	u.SetKind("EndpointSlice")
	u.SetNamespace(ns)
	u.SetName(name)
	u.SetLabels(map[string]string{"kubernetes.io/service-name": svcName})
	u.SetCreationTimestamp(metav1.Now())
	endpoints := []any{
		map[string]any{
			"addresses": []any{"10.0.0.1"},
			"conditions": map[string]any{
				"ready": ready,
			},
		},
	}
	_ = unstructured.SetNestedSlice(u.Object, endpoints, "endpoints")
	return u
}

func TestKongRoutes_Name(t *testing.T) {
	if (KongRoutes{}).Name() != "Kong Routes" {
		t.Error("Name mismatch")
	}
}

func TestKongRoutes_NoKongIngresses_IsOK(t *testing.T) {
	src := &memKongSrc{} // empty cluster
	r := KongRoutes{}.Run(context.Background(), src)
	if r.Component.Status != "OK" {
		t.Errorf("empty cluster should be OK; got %q", r.Component.Status)
	}
	if len(r.Findings) != 0 {
		t.Errorf("no findings expected; got %+v", r.Findings)
	}
}

func TestKongRoutes_HealthyBackend_NoFindings(t *testing.T) {
	src := &memKongSrc{
		ingresses: []unstructured.Unstructured{
			makeIngressKong("mcp", "mcp-api", "mcp.example.com", "mcp-api-svc", nil),
		},
		epSlices: []unstructured.Unstructured{
			makeEndpointSlice("mcp", "mcp-api-svc-abc", "mcp-api-svc", true),
		},
	}
	r := KongRoutes{}.Run(context.Background(), src)
	if r.Component.Status != "OK" {
		t.Errorf("healthy backend should be OK; got %q with findings=%+v", r.Component.Status, r.Findings)
	}
}

func TestKongRoutes_NoReadyEndpoints_WarningFires(t *testing.T) {
	src := &memKongSrc{
		ingresses: []unstructured.Unstructured{
			makeIngressKong("mcp", "mcp-api", "mcp.example.com", "mcp-api-svc", nil),
		},
		epSlices: []unstructured.Unstructured{
			makeEndpointSlice("mcp", "mcp-api-svc-abc", "mcp-api-svc", false), // not ready
		},
	}
	r := KongRoutes{}.Run(context.Background(), src)
	if r.Component.Status != "WARNING" {
		t.Errorf("expected WARNING; got %q", r.Component.Status)
	}
	if len(r.Findings) != 1 {
		t.Fatalf("expected 1 finding; got %d", len(r.Findings))
	}
	if !strings.Contains(r.Findings[0].Message, "no ready endpoints") {
		t.Errorf("finding should describe no-ready-endpoints; got %q", r.Findings[0].Message)
	}
}

func TestKongRoutes_NonKongIngressIgnored(t *testing.T) {
	// Build an Ingress with NO Kong markers (annotation/class)
	u := unstructured.Unstructured{Object: map[string]any{}}
	u.SetAPIVersion("networking.k8s.io/v1")
	u.SetKind("Ingress")
	u.SetNamespace("nginx")
	u.SetName("non-kong")
	src := &memKongSrc{ingresses: []unstructured.Unstructured{u}}
	r := KongRoutes{}.Run(context.Background(), src)
	if r.Component.Status != "OK" {
		t.Errorf("non-kong ingress must not trigger findings; got %+v", r)
	}
	if !strings.Contains(r.Component.Detail, "no Kong-managed") {
		t.Errorf("detail should report no Kong-managed; got %q", r.Component.Detail)
	}
}

func TestKongRoutes_DanglingPluginReference(t *testing.T) {
	src := &memKongSrc{
		ingresses: []unstructured.Unstructured{
			makeIngressKong("mcp", "mcp-api", "mcp.example.com", "mcp-api-svc", map[string]string{
				"konghq.com/plugins": "rate-limit",
			}),
		},
		epSlices: []unstructured.Unstructured{
			makeEndpointSlice("mcp", "mcp-api-svc-abc", "mcp-api-svc", true),
		},
		// plugins slice non-empty but no "rate-limit" → dangling ref
		plugins: []unstructured.Unstructured{
			func() unstructured.Unstructured {
				p := unstructured.Unstructured{Object: map[string]any{}}
				p.SetAPIVersion("configuration.konghq.com/v1")
				p.SetKind("KongPlugin")
				p.SetNamespace("mcp")
				p.SetName("cors") // not the one referenced
				return p
			}(),
		},
	}
	r := KongRoutes{}.Run(context.Background(), src)
	if r.Component.Status != "WARNING" {
		t.Errorf("dangling plugin should warn; got %q", r.Component.Status)
	}
	var found bool
	for _, f := range r.Findings {
		if strings.Contains(f.Message, "KongPlugin \"rate-limit\"") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected dangling-plugin finding; got %+v", r.Findings)
	}
}

// isKongIngress unit coverage —
func TestIsKongIngress_DetectionPaths(t *testing.T) {
	cases := []struct {
		name string
		mk   func() unstructured.Unstructured
		want bool
	}{
		{"class=kong", func() unstructured.Unstructured {
			u := unstructured.Unstructured{Object: map[string]any{}}
			_ = unstructured.SetNestedField(u.Object, "kong", "spec", "ingressClassName")
			return u
		}, true},
		{"legacy annotation", func() unstructured.Unstructured {
			u := unstructured.Unstructured{Object: map[string]any{}}
			u.SetAnnotations(map[string]string{"kubernetes.io/ingress.class": "kong"})
			return u
		}, true},
		{"strip-path annotation", func() unstructured.Unstructured {
			u := unstructured.Unstructured{Object: map[string]any{}}
			u.SetAnnotations(map[string]string{"konghq.com/strip-path": "true"})
			return u
		}, true},
		{"unrelated", func() unstructured.Unstructured {
			u := unstructured.Unstructured{Object: map[string]any{}}
			_ = unstructured.SetNestedField(u.Object, "nginx", "spec", "ingressClassName")
			return u
		}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			u := c.mk()
			if got := isKongIngress(&u); got != c.want {
				t.Errorf("got %v want %v", got, c.want)
			}
		})
	}
}
