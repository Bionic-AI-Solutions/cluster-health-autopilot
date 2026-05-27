// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package diagnose

import (
	"context"
	"strings"
	"testing"
	"time"

	pkgsnapshot "github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/snapshot"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type memSourceCap struct {
	byResource map[string][]unstructured.Unstructured
}

func (m *memSourceCap) List(_ context.Context, gvr schema.GroupVersionResource, ns string) (*unstructured.UnstructuredList, error) {
	out := &unstructured.UnstructuredList{}
	for _, u := range m.byResource[gvr.Resource] {
		if ns != "" && u.GetNamespace() != ns {
			continue
		}
		out.Items = append(out.Items, u)
	}
	return out, nil
}

func (m *memSourceCap) Get(_ context.Context, gvr schema.GroupVersionResource, ns, name string) (*unstructured.Unstructured, error) {
	for _, u := range m.byResource[gvr.Resource] {
		if u.GetNamespace() == ns && u.GetName() == name {
			return &u, nil
		}
	}
	return nil, nil
}

func (m *memSourceCap) Mode() pkgsnapshot.Mode { return pkgsnapshot.ModeLive }

func makeHPA(ns, name string, minR, maxR, current int64, lastScaleAgo time.Duration, conds []map[string]interface{}) unstructured.Unstructured {
	u := unstructured.Unstructured{}
	u.SetAPIVersion("autoscaling/v2")
	u.SetKind("HorizontalPodAutoscaler")
	u.SetNamespace(ns)
	u.SetName(name)
	u.SetCreationTimestamp(metav1.Time{Time: time.Now().Add(-lastScaleAgo - time.Hour)})
	_ = unstructured.SetNestedField(u.Object, minR, "spec", "minReplicas")
	_ = unstructured.SetNestedField(u.Object, maxR, "spec", "maxReplicas")
	_ = unstructured.SetNestedField(u.Object, current, "status", "currentReplicas")
	if lastScaleAgo > 0 {
		_ = unstructured.SetNestedField(u.Object,
			time.Now().Add(-lastScaleAgo).UTC().Format(time.RFC3339),
			"status", "lastScaleTime")
	}
	if len(conds) > 0 {
		ic := make([]interface{}, len(conds))
		for i, c := range conds {
			ic[i] = c
		}
		_ = unstructured.SetNestedSlice(u.Object, ic, "status", "conditions")
	}
	return u
}

func cond(condType, status, reason, msg string, sinceAgo time.Duration) map[string]interface{} {
	return map[string]interface{}{
		"type":               condType,
		"status":             status,
		"reason":             reason,
		"message":            msg,
		"lastTransitionTime": time.Now().Add(-sinceAgo).UTC().Format(time.RFC3339),
	}
}

func makePVC(ns, name, requested, capacity string, conds []map[string]interface{}, createdAgo time.Duration) unstructured.Unstructured {
	u := unstructured.Unstructured{}
	u.SetAPIVersion("v1")
	u.SetKind("PersistentVolumeClaim")
	u.SetNamespace(ns)
	u.SetName(name)
	u.SetCreationTimestamp(metav1.Time{Time: time.Now().Add(-createdAgo)})
	if requested != "" {
		_ = unstructured.SetNestedField(u.Object, requested, "spec", "resources", "requests", "storage")
	}
	if capacity != "" {
		_ = unstructured.SetNestedField(u.Object, capacity, "status", "capacity", "storage")
	}
	if len(conds) > 0 {
		ic := make([]interface{}, len(conds))
		for i, c := range conds {
			ic[i] = c
		}
		_ = unstructured.SetNestedSlice(u.Object, ic, "status", "conditions")
	}
	return u
}

// --- HPA signals ---

func TestCapacityDrift_HealthyHPA_NoFinding(t *testing.T) {
	// current between min and max, no failure conditions.
	src := &memSourceCap{byResource: map[string][]unstructured.Unstructured{
		"horizontalpodautoscalers": {
			makeHPA("app", "x", 2, 10, 5, time.Hour, nil),
		},
	}}
	got := CapacityDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("healthy HPA should be silent; got: %+v", got)
	}
}

func TestCapacityDrift_HPAPinnedAtMax_Critical(t *testing.T) {
	// current==max, dwell well past saturation grace (default 24h).
	src := &memSourceCap{byResource: map[string][]unstructured.Unstructured{
		"horizontalpodautoscalers": {
			makeHPA("app", "x", 2, 10, 10, 48*time.Hour, nil),
		},
	}}
	got := CapacityDrift{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("expected 1 diagnostic; got %d: %+v", len(got), got)
	}
	if got[0].Severity != "critical" {
		t.Errorf("severity=%s want critical (chronic saturation)", got[0].Severity)
	}
	if !strings.Contains(got[0].Message, "pinned at maxReplicas") {
		t.Errorf("message lacks 'pinned at maxReplicas': %s", got[0].Message)
	}
}

func TestCapacityDrift_HPAPinnedAtMaxWithinGrace_Silent(t *testing.T) {
	// current==max but only 1h dwell — within the 24h saturation grace.
	src := &memSourceCap{byResource: map[string][]unstructured.Unstructured{
		"horizontalpodautoscalers": {
			makeHPA("app", "x", 2, 10, 10, time.Hour, nil),
		},
	}}
	got := CapacityDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("within saturation grace should be silent; got: %+v", got)
	}
}

func TestCapacityDrift_HPAPinnedAtMin_Warning(t *testing.T) {
	// current==min for > 30d with maxReplicas much larger.
	src := &memSourceCap{byResource: map[string][]unstructured.Unstructured{
		"horizontalpodautoscalers": {
			makeHPA("app", "x", 3, 20, 3, 35*24*time.Hour, nil),
		},
	}}
	got := CapacityDrift{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("expected 1 diagnostic; got %d: %+v", len(got), got)
	}
	if got[0].Severity != "warning" {
		t.Errorf("severity=%s want warning (decorative HPA)", got[0].Severity)
	}
	if !strings.Contains(got[0].Message, "not load-driven") {
		t.Errorf("message lacks 'not load-driven': %s", got[0].Message)
	}
}

func TestCapacityDrift_HPAPinnedAtMinWithinIdleGrace_Silent(t *testing.T) {
	// current==min but dwell only 5d — within the 30d idle grace.
	src := &memSourceCap{byResource: map[string][]unstructured.Unstructured{
		"horizontalpodautoscalers": {
			makeHPA("app", "x", 3, 20, 3, 5*24*time.Hour, nil),
		},
	}}
	got := CapacityDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("within idle grace should be silent; got: %+v", got)
	}
}

func TestCapacityDrift_HPAPinnedAtMin_MinEqualsMax_Skipped(t *testing.T) {
	// min == max means the HPA isn't really configured to autoscale —
	// the operator intends a static replica count. Don't flag.
	src := &memSourceCap{byResource: map[string][]unstructured.Unstructured{
		"horizontalpodautoscalers": {
			makeHPA("app", "x", 3, 3, 3, 60*24*time.Hour, nil),
		},
	}}
	got := CapacityDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("min==max is intentional; got: %+v", got)
	}
}

func TestCapacityDrift_HPAAbleToScaleFalse_Critical(t *testing.T) {
	conds := []map[string]interface{}{
		cond("AbleToScale", "False", "FailedUpdateScale", "quota exceeded", time.Hour),
	}
	src := &memSourceCap{byResource: map[string][]unstructured.Unstructured{
		"horizontalpodautoscalers": {
			makeHPA("app", "x", 2, 10, 5, time.Hour, conds),
		},
	}}
	got := CapacityDrift{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("expected 1 diagnostic; got %d: %+v", len(got), got)
	}
	if got[0].Severity != "critical" {
		t.Errorf("severity=%s want critical", got[0].Severity)
	}
	if !strings.Contains(got[0].Message, "AbleToScale=False") {
		t.Errorf("message lacks 'AbleToScale=False': %s", got[0].Message)
	}
}

func TestCapacityDrift_HPAAbleToScaleFalseWithinGrace_Silent(t *testing.T) {
	// AbleToScale=False but only 2 min ago — within the 15 min grace.
	conds := []map[string]interface{}{
		cond("AbleToScale", "False", "FailedUpdateScale", "quota exceeded", 2*time.Minute),
	}
	src := &memSourceCap{byResource: map[string][]unstructured.Unstructured{
		"horizontalpodautoscalers": {
			makeHPA("app", "x", 2, 10, 5, time.Hour, conds),
		},
	}}
	got := CapacityDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("AbleToScale=False within grace should be silent; got: %+v", got)
	}
}

func TestCapacityDrift_HPAMetricsUnavailable_Warning(t *testing.T) {
	conds := []map[string]interface{}{
		cond("ScalingActive", "False", "FailedGetResourceMetric",
			"unable to get metric cpu: no metrics returned", time.Minute),
	}
	src := &memSourceCap{byResource: map[string][]unstructured.Unstructured{
		"horizontalpodautoscalers": {
			makeHPA("app", "x", 2, 10, 5, time.Hour, conds),
		},
	}}
	got := CapacityDrift{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("expected 1 diagnostic; got %d: %+v", len(got), got)
	}
	if got[0].Severity != "warning" {
		t.Errorf("severity=%s want warning", got[0].Severity)
	}
	if !strings.Contains(got[0].Message, "metrics-server is missing") {
		t.Errorf("message lacks 'metrics-server is missing': %s", got[0].Message)
	}
}

func TestCapacityDrift_HPAInKubeSystem_Skipped(t *testing.T) {
	src := &memSourceCap{byResource: map[string][]unstructured.Unstructured{
		"horizontalpodautoscalers": {
			makeHPA("kube-system", "metrics-hpa", 2, 10, 10, 48*time.Hour, nil),
		},
	}}
	got := CapacityDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("kube-system should be skipped; got: %+v", got)
	}
}

// --- PVC expansion signals ---

func TestCapacityDrift_PVCResizePending_Critical(t *testing.T) {
	conds := []map[string]interface{}{
		cond("FileSystemResizePending", "True", "Resizing",
			"pod restart required to complete expansion", time.Hour),
	}
	src := &memSourceCap{byResource: map[string][]unstructured.Unstructured{
		"persistentvolumeclaims": {
			makePVC("app", "data", "20Gi", "10Gi", conds, 2*time.Hour),
		},
	}}
	got := CapacityDrift{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("expected 1 diagnostic; got %d: %+v", len(got), got)
	}
	if got[0].Severity != "critical" {
		t.Errorf("severity=%s want critical", got[0].Severity)
	}
	if !strings.Contains(got[0].Message, "FileSystemResizePending") {
		t.Errorf("message lacks 'FileSystemResizePending': %s", got[0].Message)
	}
}

func TestCapacityDrift_PVCResizePendingWithinGrace_Silent(t *testing.T) {
	conds := []map[string]interface{}{
		cond("FileSystemResizePending", "True", "Resizing",
			"in progress", 2*time.Minute),
	}
	src := &memSourceCap{byResource: map[string][]unstructured.Unstructured{
		"persistentvolumeclaims": {
			makePVC("app", "data", "20Gi", "10Gi", conds, 2*time.Hour),
		},
	}}
	got := CapacityDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("resize within grace should be silent; got: %+v", got)
	}
}

func TestCapacityDrift_PVCRequestExceedsCapacityNoCondition_Critical(t *testing.T) {
	// Older CSI drivers don't always emit FileSystemResizePending —
	// the request/capacity divergence by itself indicates a stuck
	// expansion when the PVC is older than the grace window.
	src := &memSourceCap{byResource: map[string][]unstructured.Unstructured{
		"persistentvolumeclaims": {
			makePVC("app", "data", "50Gi", "10Gi", nil, 2*time.Hour),
		},
	}}
	got := CapacityDrift{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("expected 1 diagnostic; got %d: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Message, "expansion is stuck") {
		t.Errorf("message lacks 'expansion is stuck': %s", got[0].Message)
	}
}

func TestCapacityDrift_PVCRequestMatchesCapacity_NoFinding(t *testing.T) {
	// Steady-state PVC — neither resize-pending nor request>capacity.
	src := &memSourceCap{byResource: map[string][]unstructured.Unstructured{
		"persistentvolumeclaims": {
			makePVC("app", "data", "10Gi", "10Gi", nil, time.Hour),
		},
	}}
	got := CapacityDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("steady-state PVC should be silent; got: %+v", got)
	}
}

func TestCapacityDrift_PVCYoungerThanGrace_Silent(t *testing.T) {
	// Brand-new PVC requesting more storage than current capacity —
	// the expansion may still be in progress. Don't flag immediately.
	src := &memSourceCap{byResource: map[string][]unstructured.Unstructured{
		"persistentvolumeclaims": {
			makePVC("app", "data", "50Gi", "10Gi", nil, 2*time.Minute),
		},
	}}
	got := CapacityDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("PVC younger than grace should be silent; got: %+v", got)
	}
}

// --- Misc ---

func TestCapacityDrift_RunNoOpOnEmptySource(t *testing.T) {
	src := &memSourceCap{byResource: map[string][]unstructured.Unstructured{}}
	got := CapacityDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("empty source should be no-op; got: %+v", got)
	}
}

func TestCapacityDrift_NameStable(t *testing.T) {
	if name := (CapacityDrift{}).Name(); name != "CapacityDrift" {
		t.Errorf("Name()=%q want CapacityDrift", name)
	}
}
