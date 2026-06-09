// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package diagnose

import (
	"context"
	"strings"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// makePodOOM builds a Pod fixture with one container that has a
// configurable restart count + OOMKilled lastState. Reuses the
// memSourceDD test source from disruption_drift_test.go (same
// package, no need to redefine).
func makePodOOM(ns, name, container string, restartCount int64, lastReason string, finishedAt time.Time) unstructured.Unstructured {
	u := unstructured.Unstructured{}
	u.SetAPIVersion("v1")
	u.SetKind("Pod")
	u.SetNamespace(ns)
	u.SetName(name)
	u.SetCreationTimestamp(metav1.Time{Time: time.Now().Add(-7 * 24 * time.Hour)})
	cs := []any{
		map[string]any{
			"name":         container,
			"restartCount": restartCount,
			"lastState": map[string]any{
				"terminated": map[string]any{
					"reason":     lastReason,
					"finishedAt": finishedAt.UTC().Format(time.RFC3339),
				},
			},
		},
	}
	_ = unstructured.SetNestedSlice(u.Object, cs, "status", "containerStatuses")
	return u
}

func TestOOMKillRecurrence_FiresOnThreeOrMoreOOM(t *testing.T) {
	src := &memSourceDD{byResource: map[string][]unstructured.Unstructured{
		"pods": {
			makePodOOM("prod", "app-1", "main", 4, "OOMKilled", time.Now().Add(-10*time.Minute)),
		},
	}}
	got := OOMKillRecurrence{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("expected 1 diagnostic; got %d", len(got))
	}
	if got[0].Subject != "Pod/prod/app-1" {
		t.Errorf("subject: %q", got[0].Subject)
	}
	if got[0].Severity != "warning" {
		t.Errorf("severity: %q", got[0].Severity)
	}
	if !strings.Contains(got[0].Message, "OOMKilled 4 times") {
		t.Errorf("message should cite count; got %q", got[0].Message)
	}
}

func TestOOMKillRecurrence_LowCountSkipped(t *testing.T) {
	// 2 OOM kills is not yet a recurrence signal — could be noisy
	// neighbor or single memory spike.
	src := &memSourceDD{byResource: map[string][]unstructured.Unstructured{
		"pods": {
			makePodOOM("prod", "app-1", "main", 2, "OOMKilled", time.Now().Add(-10*time.Minute)),
		},
	}}
	got := OOMKillRecurrence{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("restartCount=2 must NOT fire; got %+v", got)
	}
}

func TestOOMKillRecurrence_NonOOMReasonSkipped(t *testing.T) {
	// A pod that crashed for OTHER reasons (Error / Completed)
	// shouldn't fire the OOM signal — handled by other analyzers.
	src := &memSourceDD{byResource: map[string][]unstructured.Unstructured{
		"pods": {
			makePodOOM("prod", "app-1", "main", 5, "Error", time.Now().Add(-10*time.Minute)),
		},
	}}
	got := OOMKillRecurrence{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("non-OOMKilled reason must NOT fire; got %+v", got)
	}
}

func TestOOMKillRecurrence_StaleFinishedAtSkipped(t *testing.T) {
	// Last OOM was 36h ago — outside the 24h recurrence window.
	// The workload might have been fixed since.
	src := &memSourceDD{byResource: map[string][]unstructured.Unstructured{
		"pods": {
			makePodOOM("prod", "app-1", "main", 5, "OOMKilled", time.Now().Add(-36*time.Hour)),
		},
	}}
	got := OOMKillRecurrence{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("stale OOM (>24h) must NOT fire; got %+v", got)
	}
}

func TestOOMKillRecurrence_DedupedPerPod(t *testing.T) {
	// Two containers in the same pod both crossed the threshold.
	// Emit ONE diagnostic per pod (operator's edit pass fixes
	// both containers simultaneously).
	u := unstructured.Unstructured{}
	u.SetAPIVersion("v1")
	u.SetKind("Pod")
	u.SetNamespace("prod")
	u.SetName("multi-container")
	u.SetCreationTimestamp(metav1.Time{Time: time.Now().Add(-7 * 24 * time.Hour)})
	now := time.Now().Add(-5 * time.Minute).UTC().Format(time.RFC3339)
	cs := []any{
		map[string]any{
			"name":         "c1",
			"restartCount": int64(3),
			"lastState": map[string]any{
				"terminated": map[string]any{"reason": "OOMKilled", "finishedAt": now},
			},
		},
		map[string]any{
			"name":         "c2",
			"restartCount": int64(4),
			"lastState": map[string]any{
				"terminated": map[string]any{"reason": "OOMKilled", "finishedAt": now},
			},
		},
	}
	_ = unstructured.SetNestedSlice(u.Object, cs, "status", "containerStatuses")
	src := &memSourceDD{byResource: map[string][]unstructured.Unstructured{"pods": {u}}}

	got := OOMKillRecurrence{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Errorf("multi-container pod should emit 1 dedup'd diagnostic; got %d", len(got))
	}
}

func TestOOMKillRecurrence_Name(t *testing.T) {
	if (OOMKillRecurrence{}).Name() != "OOMKillRecurrence" {
		t.Error("Name mismatch")
	}
}

func TestOOMKillRecurrence_EmptyClusterNoOp(t *testing.T) {
	src := &memSourceDD{byResource: map[string][]unstructured.Unstructured{}}
	got := OOMKillRecurrence{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("empty cluster must NOT fire; got %+v", got)
	}
}
