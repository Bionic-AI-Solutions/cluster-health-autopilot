// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package diagnose

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func makePV(name, phase, storageClass, capacity, reclaimPolicy string, ageDays float64) unstructured.Unstructured {
	u := unstructured.Unstructured{}
	u.SetAPIVersion("v1")
	u.SetKind("PersistentVolume")
	u.SetName(name)
	u.SetCreationTimestamp(metav1.Time{Time: time.Now().Add(-time.Duration(ageDays * 24 * float64(time.Hour)))})
	_ = unstructured.SetNestedField(u.Object, phase, "status", "phase")
	if storageClass != "" {
		_ = unstructured.SetNestedField(u.Object, storageClass, "spec", "storageClassName")
	}
	if capacity != "" {
		_ = unstructured.SetNestedField(u.Object, capacity, "spec", "capacity", "storage")
	}
	if reclaimPolicy != "" {
		_ = unstructured.SetNestedField(u.Object, reclaimPolicy, "spec", "persistentVolumeReclaimPolicy")
	}
	return u
}

func TestPVOrphan_ReleasedPastGrace_Fires(t *testing.T) {
	src := &memSourceDD{byResource: map[string][]unstructured.Unstructured{
		"persistentvolumes": {
			makePV("pv-legacy-ebs-1", "Released", "ebs-gp3", "100Gi", "Retain", 14),
		},
	}}
	got := PVOrphan{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("expected 1 diagnostic; got %d", len(got))
	}
	if got[0].Subject != "PersistentVolume/cluster/orphaned-released" {
		t.Errorf("subject: %q", got[0].Subject)
	}
	if !strings.Contains(got[0].Message, "1 PersistentVolume") || !strings.Contains(got[0].Message, "may still be billing") {
		t.Errorf("message should state count + billing impact; got %q", got[0].Message)
	}
	// Per-PV detail (name, capacity, storageClass) lives in the Remediation list.
	if !strings.Contains(got[0].Remediation, "pv-legacy-ebs-1") ||
		!strings.Contains(got[0].Remediation, "ebs-gp3") || !strings.Contains(got[0].Remediation, "100Gi") {
		t.Errorf("remediation should list the PV with cost-sizing detail; got %q", got[0].Remediation)
	}
}

func TestPVOrphan_AggregatesMany(t *testing.T) {
	// 60 orphaned PVs must collapse into ONE finding (not 60), with the list
	// capped and a "+N more" tail — the whole point of the trim.
	pvs := make([]unstructured.Unstructured, 0, 60)
	for i := 0; i < 60; i++ {
		pvs = append(pvs, makePV(fmt.Sprintf("pv-%02d", i), "Released", "ebs-gp3", "10Gi", "Retain", 14))
	}
	src := &memSourceDD{byResource: map[string][]unstructured.Unstructured{"persistentvolumes": pvs}}
	got := PVOrphan{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("60 orphans must aggregate to 1 finding; got %d", len(got))
	}
	if !strings.Contains(got[0].Message, "60 PersistentVolume") {
		t.Errorf("message should state the count of 60; got %q", got[0].Message)
	}
	if !strings.Contains(got[0].Remediation, "and 35 more") { // 60 - cap(25)
		t.Errorf("remediation should cap the list with a '+N more' tail; got %q", got[0].Remediation)
	}
}

func TestPVOrphan_BoundPhaseSkipped(t *testing.T) {
	src := &memSourceDD{byResource: map[string][]unstructured.Unstructured{
		"persistentvolumes": {
			makePV("pv-healthy", "Bound", "ebs-gp3", "100Gi", "Retain", 30),
		},
	}}
	got := PVOrphan{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("Bound PV must NOT fire; got %+v", got)
	}
}

func TestPVOrphan_RecentReleaseSkipped(t *testing.T) {
	// PV created 2 days ago and already Released — fast-churn dev
	// workload, skip until past 7-day grace.
	src := &memSourceDD{byResource: map[string][]unstructured.Unstructured{
		"persistentvolumes": {
			makePV("pv-dev-1", "Released", "ebs-gp3", "10Gi", "Retain", 2),
		},
	}}
	got := PVOrphan{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("recent Release (<7d) must NOT fire; got %+v", got)
	}
}

func TestPVOrphan_MissingFieldsHaveSafeDefaults(t *testing.T) {
	// A PV without storageClassName / capacity (rare edge case)
	// should still surface, just with <unknown> placeholders.
	src := &memSourceDD{byResource: map[string][]unstructured.Unstructured{
		"persistentvolumes": {
			makePV("pv-bare", "Released", "", "", "", 14),
		},
	}}
	got := PVOrphan{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("bare PV should still fire; got %d", len(got))
	}
	if !strings.Contains(got[0].Remediation, "<default>") || !strings.Contains(got[0].Remediation, "<unknown>") {
		t.Errorf("missing fields should render safe placeholders in the list; got %q", got[0].Remediation)
	}
}

func TestPVOrphan_Name(t *testing.T) {
	if (PVOrphan{}).Name() != "PVOrphan" {
		t.Error("Name mismatch")
	}
}
