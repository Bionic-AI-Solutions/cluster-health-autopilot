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

// memSourceWSD is a minimal in-memory snapshot.Source for these
// tests. Keyed by Resource name (the lowercase plural of the kind).
type memSourceWSD struct {
	byResource map[string][]unstructured.Unstructured
}

func (m *memSourceWSD) List(_ context.Context, gvr schema.GroupVersionResource, ns string) (*unstructured.UnstructuredList, error) {
	out := &unstructured.UnstructuredList{}
	for _, u := range m.byResource[gvr.Resource] {
		if ns != "" && u.GetNamespace() != ns {
			continue
		}
		out.Items = append(out.Items, u)
	}
	return out, nil
}

func (m *memSourceWSD) Get(_ context.Context, gvr schema.GroupVersionResource, ns, name string) (*unstructured.Unstructured, error) {
	for _, u := range m.byResource[gvr.Resource] {
		if u.GetNamespace() == ns && u.GetName() == name {
			return &u, nil
		}
	}
	return nil, nil
}

func (m *memSourceWSD) Mode() pkgsnapshot.Mode { return pkgsnapshot.ModeLive }

func cnpgCluster(ns, name, phase, currentPrimary, targetPrimary string, instances, ready int64, ageHours float64) unstructured.Unstructured {
	u := unstructured.Unstructured{}
	u.SetAPIVersion("postgresql.cnpg.io/v1")
	u.SetKind("Cluster")
	u.SetNamespace(ns)
	u.SetName(name)
	u.SetCreationTimestamp(metav1.NewTime(time.Now().Add(-time.Duration(ageHours * float64(time.Hour)))))
	_ = unstructured.SetNestedField(u.Object, instances, "spec", "instances")
	_ = unstructured.SetNestedField(u.Object, ready, "status", "readyInstances")
	if phase != "" {
		_ = unstructured.SetNestedField(u.Object, phase, "status", "phase")
	}
	if currentPrimary != "" {
		_ = unstructured.SetNestedField(u.Object, currentPrimary, "status", "currentPrimary")
	}
	if targetPrimary != "" {
		_ = unstructured.SetNestedField(u.Object, targetPrimary, "status", "targetPrimary")
	}
	return u
}

func sts(ns, name string, desired, ready int64, ageHours float64) unstructured.Unstructured {
	u := unstructured.Unstructured{}
	u.SetAPIVersion("apps/v1")
	u.SetKind("StatefulSet")
	u.SetNamespace(ns)
	u.SetName(name)
	u.SetCreationTimestamp(metav1.NewTime(time.Now().Add(-time.Duration(ageHours * float64(time.Hour)))))
	_ = unstructured.SetNestedField(u.Object, desired, "spec", "replicas")
	_ = unstructured.SetNestedField(u.Object, ready, "status", "readyReplicas")
	_ = unstructured.SetNestedMap(u.Object, map[string]interface{}{"app": name}, "spec", "selector", "matchLabels")
	return u
}

func pod(ns, name string, readyCondition string) unstructured.Unstructured {
	u := unstructured.Unstructured{}
	u.SetAPIVersion("v1")
	u.SetKind("Pod")
	u.SetNamespace(ns)
	u.SetName(name)
	_ = unstructured.SetNestedSlice(u.Object, []interface{}{
		map[string]interface{}{
			"type":   "Ready",
			"status": readyCondition, // "True" / "False"
		},
	}, "status", "conditions")
	return u
}

// --- CNPG tests --------------------------------------------------------------

func TestWorkloadStateDrift_CNPGHealthy_Silent(t *testing.T) {
	src := &memSourceWSD{byResource: map[string][]unstructured.Unstructured{
		"clusters": {cnpgCluster("pg", "pg-main", "Cluster in healthy state", "pg-main-1", "pg-main-1", 3, 3, 24)},
	}}
	got := WorkloadStateDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("healthy CNPG cluster should produce 0 diagnostics; got %+v", got)
	}
}

func TestWorkloadStateDrift_CNPGNonHealthyPhase_Warning(t *testing.T) {
	src := &memSourceWSD{byResource: map[string][]unstructured.Unstructured{
		"clusters": {cnpgCluster("pg", "pg-main", "Setting up primary", "pg-main-1", "pg-main-1", 3, 1, 24)},
	}}
	got := WorkloadStateDrift{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("non-healthy phase should emit 1 diagnostic; got %d", len(got))
	}
	if got[0].Severity != "warning" {
		t.Errorf("setup phase should be warning; got %q", got[0].Severity)
	}
	if !strings.Contains(got[0].Message, "Setting up primary") {
		t.Errorf("diagnostic should surface phase; got %q", got[0].Message)
	}
}

func TestWorkloadStateDrift_CNPGFailoverInProgress_Critical(t *testing.T) {
	src := &memSourceWSD{byResource: map[string][]unstructured.Unstructured{
		"clusters": {cnpgCluster("pg", "pg-main", "Failing over", "pg-main-1", "pg-main-2", 3, 2, 24)},
	}}
	got := WorkloadStateDrift{}.Run(context.Background(), src)
	if len(got) == 0 {
		t.Fatal("failover phase should emit at least 1 diagnostic")
	}
	// At least one of the diagnostics should be critical.
	foundCritical := false
	for _, d := range got {
		if d.Severity == "critical" {
			foundCritical = true
		}
	}
	if !foundCritical {
		t.Errorf("failover should emit critical; got %+v", got)
	}
}

func TestWorkloadStateDrift_CNPGFollowerDegraded_HealthyPhase(t *testing.T) {
	// Cluster reports phase=healthy but ready < instances → followers
	// degraded, warning-class. Important: the phase hasn't flipped
	// (transient), so the operator gets the early signal.
	src := &memSourceWSD{byResource: map[string][]unstructured.Unstructured{
		"clusters": {cnpgCluster("pg", "pg-main", "Cluster in healthy state", "pg-main-1", "pg-main-1", 3, 2, 24)},
	}}
	got := WorkloadStateDrift{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("follower degraded should emit 1 diagnostic; got %d: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Subject, "(followers)") {
		t.Errorf("subject should mark follower scope; got %q", got[0].Subject)
	}
	if got[0].Severity != "warning" {
		t.Errorf("follower degraded should be warning; got %q", got[0].Severity)
	}
}

func TestWorkloadStateDrift_CNPGPrimarySwitchoverStuck_Critical(t *testing.T) {
	src := &memSourceWSD{byResource: map[string][]unstructured.Unstructured{
		"clusters": {cnpgCluster("pg", "pg-main", "Cluster in healthy state", "pg-main-1", "pg-main-2", 3, 3, 24)},
	}}
	got := WorkloadStateDrift{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("primary switchover stuck should emit 1 diagnostic; got %d", len(got))
	}
	if got[0].Severity != "critical" {
		t.Errorf("switchover stuck should be critical; got %q", got[0].Severity)
	}
	if !strings.Contains(got[0].Subject, "(primary)") {
		t.Errorf("subject should mark primary scope; got %q", got[0].Subject)
	}
}

func TestWorkloadStateDrift_CNPGYoungCluster_Suppressed(t *testing.T) {
	// Cluster created 1 minute ago — even if unhealthy, suppress
	// because the operator just deployed it.
	src := &memSourceWSD{byResource: map[string][]unstructured.Unstructured{
		"clusters": {cnpgCluster("pg", "pg-main", "Setting up primary", "", "", 3, 0, 1.0/60.0)},
	}}
	got := WorkloadStateDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("brand-new cluster should be suppressed within grace; got %+v", got)
	}
}

// --- StatefulSet ordinal-zero tests ------------------------------------------

func TestWorkloadStateDrift_STSHealthy_Silent(t *testing.T) {
	src := &memSourceWSD{byResource: map[string][]unstructured.Unstructured{
		"statefulsets": {sts("data", "kafka", 3, 3, 24)},
	}}
	got := WorkloadStateDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("healthy STS should produce 0 diagnostics; got %+v", got)
	}
}

func TestWorkloadStateDrift_STSPod0Missing_Critical(t *testing.T) {
	// STS reports 2/3 ready but pod-0 doesn't exist (terminated,
	// stuck pre-create). Critical.
	src := &memSourceWSD{byResource: map[string][]unstructured.Unstructured{
		"statefulsets": {sts("data", "kafka", 3, 2, 24)},
		"pods": {
			pod("data", "kafka-1", "True"),
			pod("data", "kafka-2", "True"),
			// kafka-0 missing
		},
	}}
	got := WorkloadStateDrift{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("missing pod-0 should emit 1 diagnostic; got %d: %+v", len(got), got)
	}
	if got[0].Severity != "critical" {
		t.Errorf("missing pod-0 should be critical; got %q", got[0].Severity)
	}
	if !strings.Contains(got[0].Message, "missing pod-0") {
		t.Errorf("message should call out missing pod-0; got %q", got[0].Message)
	}
}

func TestWorkloadStateDrift_STSPod0Unready_Warning(t *testing.T) {
	src := &memSourceWSD{byResource: map[string][]unstructured.Unstructured{
		"statefulsets": {sts("data", "kafka", 3, 2, 24)},
		"pods": {
			pod("data", "kafka-0", "False"),
			pod("data", "kafka-1", "True"),
			pod("data", "kafka-2", "True"),
		},
	}}
	got := WorkloadStateDrift{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("pod-0 unready should emit 1 diagnostic; got %d: %+v", len(got), got)
	}
	if got[0].Severity != "warning" {
		t.Errorf("pod-0 unready should be warning; got %q", got[0].Severity)
	}
	if !strings.Contains(got[0].Message, "pod-0 (ordinal-zero) NOT ready") {
		t.Errorf("message should call out pod-0 unready; got %q", got[0].Message)
	}
}

func TestWorkloadStateDrift_STSAllReady_NoFalsePositive(t *testing.T) {
	src := &memSourceWSD{byResource: map[string][]unstructured.Unstructured{
		"statefulsets": {sts("data", "kafka", 3, 3, 24)},
		"pods": {
			pod("data", "kafka-0", "True"),
			pod("data", "kafka-1", "True"),
			pod("data", "kafka-2", "True"),
		},
	}}
	got := WorkloadStateDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("all-ready STS should produce 0 diagnostics; got %+v", got)
	}
}

func TestWorkloadStateDrift_STSYoung_Suppressed(t *testing.T) {
	// STS created 1 minute ago — even if degraded, suppress.
	src := &memSourceWSD{byResource: map[string][]unstructured.Unstructured{
		"statefulsets": {sts("data", "kafka", 3, 0, 1.0/60.0)},
	}}
	got := WorkloadStateDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("brand-new STS should be suppressed within grace; got %+v", got)
	}
}

func TestWorkloadStateDrift_NoCRDs_NoOp(t *testing.T) {
	// Cluster without CNPG installed AND no StatefulSets — silent.
	src := &memSourceWSD{byResource: map[string][]unstructured.Unstructured{}}
	got := WorkloadStateDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("empty cluster should produce 0 diagnostics; got %+v", got)
	}
}
