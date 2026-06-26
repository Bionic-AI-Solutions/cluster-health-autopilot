// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package resolution

import (
	"context"
	"testing"
	"time"

	"github.com/srenix-ai/agentic-sre/internal/snapshot"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

// fakeMutator captures Create calls; Patch/PatchStatus are no-ops here.
type fakeMutator struct {
	created []*unstructured.Unstructured
	gvrs    []schema.GroupVersionResource
}

func (f *fakeMutator) Create(_ context.Context, gvr schema.GroupVersionResource, _ string, obj *unstructured.Unstructured) error {
	f.created = append(f.created, obj.DeepCopy())
	f.gvrs = append(f.gvrs, gvr)
	return nil
}
func (f *fakeMutator) Patch(_ context.Context, _ schema.GroupVersionResource, _, _ string, _ types.PatchType, _ []byte) error {
	return nil
}
func (f *fakeMutator) PatchStatus(_ context.Context, _ schema.GroupVersionResource, _, _ string, _ types.PatchType, _ []byte) error {
	return nil
}
func (f *fakeMutator) Delete(_ context.Context, _ schema.GroupVersionResource, _, _ string) error {
	return nil
}

func TestFingerprint_StableAndShort(t *testing.T) {
	a := Fingerprint("HPAScaling", "HorizontalPodAutoscaler/pg/postgresql-hpa")
	b := Fingerprint("HPAScaling", "HorizontalPodAutoscaler/pg/postgresql-hpa")
	c := Fingerprint("HPAScaling", "HorizontalPodAutoscaler/pg/other")
	if a != b {
		t.Errorf("fingerprint not stable: %s != %s", a, b)
	}
	if a == c {
		t.Errorf("different subjects must differ: %s == %s", a, c)
	}
	if len(a) != 16 {
		t.Errorf("fingerprint len=%d want 16", len(a))
	}
}

func TestRecorder_NilMutatorIsNoOp(t *testing.T) {
	if err := (Recorder{}).Record(context.Background(), nil, Record{Fingerprint: "abc"}); err != nil {
		t.Errorf("nil mutator should be a no-op, got: %v", err)
	}
}

func TestRecorder_Record_WritesCR(t *testing.T) {
	mut := &fakeMutator{}
	rec := Recorder{Now: func() time.Time { return time.Date(2026, 5, 29, 6, 0, 0, 0, time.UTC) }}
	r := Record{
		Fingerprint:      "deadbeefcafe0001",
		Cluster:          "bionic-cluster",
		Namespace:        "vc-livekit",
		Source:           "CrashLoopBackOff",
		SubjectKind:      "Pod",
		DiagnosticDigest: "Pod vc-livekit/livekit-agent in CrashLoopBackOff (DNS: livekit.srenix.ai unresolved)",
		ActionKind:       "PatchConfigMap",
		Target:           "ConfigMap/vc-livekit/vc-livekit-config",
		Rationale:        "repoint LIVEKIT_URL to the in-cluster service",
		Rollback:         "restore prior LIVEKIT_URL",
		Delivery:         DeliveryHumanApproved,
		Applied:          true,
		Verified:         VerdictCleared,
	}
	if err := rec.Record(context.Background(), mut, r); err != nil {
		t.Fatalf("Record: %v", err)
	}
	if len(mut.created) != 1 {
		t.Fatalf("expected 1 CR created, got %d", len(mut.created))
	}
	cr := mut.created[0]
	if mut.gvrs[0] != snapshot.GVRResolutionRecord {
		t.Errorf("wrong GVR: %v", mut.gvrs[0])
	}
	if cr.GetKind() != "ResolutionRecord" {
		t.Errorf("kind=%s", cr.GetKind())
	}
	spec, _, _ := unstructured.NestedMap(cr.Object, "spec")
	if spec["fingerprint"] != "deadbeefcafe0001" || spec["verified"] != "cleared" || spec["applied"] != true {
		t.Errorf("spec mismatch: %+v", spec)
	}
	prop, _, _ := unstructured.NestedMap(cr.Object, "spec", "proposal")
	if prop["actionKind"] != "PatchConfigMap" {
		t.Errorf("proposal.actionKind=%v", prop["actionKind"])
	}
	labels := cr.GetLabels()
	if labels["srenix.ai/verified"] != "cleared" {
		t.Errorf("verified label=%v", labels["srenix.ai/verified"])
	}
}

func TestRecorder_Record_RequiresFingerprint(t *testing.T) {
	if err := (Recorder{}).Record(context.Background(), &fakeMutator{}, Record{}); err == nil {
		t.Error("expected error when Fingerprint is empty")
	}
}
