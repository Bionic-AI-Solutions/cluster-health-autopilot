// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package diagnose

import (
	"context"
	"strings"
	"testing"
	"time"

	pkgsnapshot "github.com/srenix-ai/agentic-sre/pkg/snapshot"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// memSourceDD is a tiny in-memory snapshot.Source for DisruptionDrift
// tests. Keys by Resource so each test fixture can populate exactly
// the GVR the analyzer queries.
type memSourceDD struct {
	byResource map[string][]unstructured.Unstructured
}

func (m *memSourceDD) List(_ context.Context, gvr schema.GroupVersionResource, ns string) (*unstructured.UnstructuredList, error) {
	out := &unstructured.UnstructuredList{}
	for _, u := range m.byResource[gvr.Resource] {
		if ns != "" && u.GetNamespace() != ns {
			continue
		}
		out.Items = append(out.Items, u)
	}
	return out, nil
}

func (m *memSourceDD) Get(_ context.Context, gvr schema.GroupVersionResource, ns, name string) (*unstructured.Unstructured, error) {
	for _, u := range m.byResource[gvr.Resource] {
		if u.GetNamespace() == ns && u.GetName() == name {
			return &u, nil
		}
	}
	return nil, nil
}

func (m *memSourceDD) Mode() pkgsnapshot.Mode { return pkgsnapshot.ModeLive }

// ---- PDB fixtures + tests ---------------------------------------------------

func makePDB(ns, name string, disruptionsAllowed int64, minAvailable string, anchor time.Time) unstructured.Unstructured {
	u := unstructured.Unstructured{}
	u.SetAPIVersion("policy/v1")
	u.SetKind("PodDisruptionBudget")
	u.SetNamespace(ns)
	u.SetName(name)
	u.SetCreationTimestamp(metav1.Time{Time: anchor})
	if minAvailable != "" {
		_ = unstructured.SetNestedField(u.Object, minAvailable, "spec", "minAvailable")
	}
	_ = unstructured.SetNestedField(u.Object, disruptionsAllowed, "status", "disruptionsAllowed")
	if !anchor.IsZero() {
		cond := map[string]any{
			"type":               "DisruptionAllowed",
			"status":             "False",
			"lastTransitionTime": anchor.UTC().Format(time.RFC3339),
		}
		_ = unstructured.SetNestedField(u.Object, []any{cond}, "status", "conditions")
	}
	return u
}

func TestDisruptionDrift_PDB_BlocksAll_FiresAfterGrace(t *testing.T) {
	src := &memSourceDD{byResource: map[string][]unstructured.Unstructured{
		"poddisruptionbudgets": {
			makePDB("prod", "blocked", 0, "100%", time.Now().Add(-10*time.Minute)),
		},
	}}
	got := DisruptionDrift{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("expected 1 diagnostic; got %d: %+v", len(got), got)
	}
	if got[0].Subject != "PodDisruptionBudget/prod/blocked" {
		t.Errorf("subject: %q", got[0].Subject)
	}
	if got[0].Severity != "critical" {
		t.Errorf("severity should be critical for blocked PDB; got %q", got[0].Severity)
	}
	if !strings.Contains(got[0].Message, "blocked ALL evictions") {
		t.Errorf("missing key phrase; got %q", got[0].Message)
	}
}

func TestDisruptionDrift_PDB_WithinGrace_Skipped(t *testing.T) {
	src := &memSourceDD{byResource: map[string][]unstructured.Unstructured{
		"poddisruptionbudgets": {
			makePDB("prod", "fresh", 0, "100%", time.Now().Add(-1*time.Minute)),
		},
	}}
	got := DisruptionDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("fresh PDB (within grace) should be skipped; got %+v", got)
	}
}

func TestDisruptionDrift_PDB_Allowed_Skipped(t *testing.T) {
	src := &memSourceDD{byResource: map[string][]unstructured.Unstructured{
		"poddisruptionbudgets": {
			makePDB("prod", "healthy", 1, "50%", time.Now().Add(-30*time.Minute)),
		},
	}}
	got := DisruptionDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("PDB with disruptionsAllowed>0 should be skipped; got %+v", got)
	}
}

// ---- Indexed Job fixtures + tests -------------------------------------------

func makeIndexedJob(ns, name, failedIndexes string, startedAgo time.Duration) unstructured.Unstructured {
	u := unstructured.Unstructured{}
	u.SetAPIVersion("batch/v1")
	u.SetKind("Job")
	u.SetNamespace(ns)
	u.SetName(name)
	u.SetCreationTimestamp(metav1.Time{Time: time.Now().Add(-startedAgo)})
	_ = unstructured.SetNestedField(u.Object, "Indexed", "spec", "completionMode")
	if failedIndexes != "" {
		_ = unstructured.SetNestedField(u.Object, failedIndexes, "status", "failedIndexes")
	}
	_ = unstructured.SetNestedField(u.Object,
		time.Now().Add(-startedAgo).UTC().Format(time.RFC3339),
		"status", "startTime")
	return u
}

func TestDisruptionDrift_StuckJobs_FailedIndexes_PastGrace(t *testing.T) {
	src := &memSourceDD{byResource: map[string][]unstructured.Unstructured{
		"jobs": {
			makeIndexedJob("data", "etl-2026", "0,2,5", 30*time.Minute),
		},
	}}
	got := DisruptionDrift{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("expected 1 diagnostic; got %d", len(got))
	}
	if got[0].Severity != "warning" {
		t.Errorf("severity: %q", got[0].Severity)
	}
	if !strings.Contains(got[0].Message, "0,2,5") {
		t.Errorf("message should list failed indexes; got %q", got[0].Message)
	}
}

func TestDisruptionDrift_StuckJobs_NonIndexed_Skipped(t *testing.T) {
	u := makeIndexedJob("data", "non-idx", "0,1", 30*time.Minute)
	_ = unstructured.SetNestedField(u.Object, "NonIndexed", "spec", "completionMode")
	src := &memSourceDD{byResource: map[string][]unstructured.Unstructured{
		"jobs": {u},
	}}
	got := DisruptionDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("non-indexed Job must NOT be flagged; got %+v", got)
	}
}

func TestDisruptionDrift_StuckJobs_NoFailedIndexes_Skipped(t *testing.T) {
	src := &memSourceDD{byResource: map[string][]unstructured.Unstructured{
		"jobs": {
			makeIndexedJob("data", "healthy", "", 30*time.Minute),
		},
	}}
	got := DisruptionDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("Indexed Job with no failedIndexes must NOT be flagged; got %+v", got)
	}
}

// ---- ResourceQuota fixtures + tests -----------------------------------------

func makeRQ(ns, name string, hard, used map[string]any, ageHours float64) unstructured.Unstructured {
	u := unstructured.Unstructured{}
	u.SetAPIVersion("v1")
	u.SetKind("ResourceQuota")
	u.SetNamespace(ns)
	u.SetName(name)
	u.SetCreationTimestamp(metav1.Time{Time: time.Now().Add(-time.Duration(ageHours * float64(time.Hour)))})
	_ = unstructured.SetNestedMap(u.Object, hard, "spec", "hard")
	_ = unstructured.SetNestedMap(u.Object, used, "status", "used")
	return u
}

func TestDisruptionDrift_ResourceQuota_Saturated_FiresAfterGrace(t *testing.T) {
	src := &memSourceDD{byResource: map[string][]unstructured.Unstructured{
		"resourcequotas": {
			makeRQ("dev", "team-quota",
				map[string]any{"pods": "10", "requests.cpu": "5"},
				map[string]any{"pods": "10", "requests.cpu": "5"},
				2.0),
		},
	}}
	got := DisruptionDrift{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("expected 1 diagnostic; got %d", len(got))
	}
	if !strings.Contains(got[0].Message, "pods") || !strings.Contains(got[0].Message, "requests.cpu") {
		t.Errorf("should list both saturated resources; got %q", got[0].Message)
	}
}

func TestDisruptionDrift_ResourceQuota_HasHeadroom_Skipped(t *testing.T) {
	src := &memSourceDD{byResource: map[string][]unstructured.Unstructured{
		"resourcequotas": {
			makeRQ("dev", "team-quota",
				map[string]any{"pods": "10"},
				map[string]any{"pods": "8"},
				2.0),
		},
	}}
	got := DisruptionDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("RQ with headroom must NOT fire; got %+v", got)
	}
}

func TestDisruptionDrift_ResourceQuota_WithinGrace_Skipped(t *testing.T) {
	src := &memSourceDD{byResource: map[string][]unstructured.Unstructured{
		"resourcequotas": {
			// 100% used but only 30 minutes old — within graceQuotaAtMax.
			makeRQ("dev", "team-quota",
				map[string]any{"pods": "10"},
				map[string]any{"pods": "10"},
				0.5),
		},
	}}
	got := DisruptionDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("fresh saturated RQ (within grace) must NOT fire; got %+v", got)
	}
}

// ---- Registration sanity ----------------------------------------------------

func TestDisruptionDrift_Name(t *testing.T) {
	if (DisruptionDrift{}).Name() != "DisruptionDrift" {
		t.Errorf("Name mismatch")
	}
}

func TestDisruptionDrift_EmptySource_NoOp(t *testing.T) {
	src := &memSourceDD{byResource: map[string][]unstructured.Unstructured{}}
	got := DisruptionDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("empty source should yield no diagnostics; got %+v", got)
	}
}
