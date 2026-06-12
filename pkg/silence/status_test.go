// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package silence

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	chav1alpha1 "github.com/Bionic-AI-Solutions/cluster-health-autopilot/api/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

// fakePatcher records every PatchStatus call.
type fakePatcher struct {
	calls []patchCall
	err   error
}

type patchCall struct {
	gvr   schema.GroupVersionResource
	ns    string
	name  string
	pt    types.PatchType
	patch []byte
}

func (f *fakePatcher) PatchStatus(_ context.Context, gvr schema.GroupVersionResource, ns, name string, pt types.PatchType, patch []byte) error {
	f.calls = append(f.calls, patchCall{gvr: gvr, ns: ns, name: name, pt: pt, patch: patch})
	return f.err
}

func mkSilence(ns, name string, until time.Time, active bool, matchCount int64) chav1alpha1.Silence {
	return chav1alpha1.Silence{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: name},
		Spec: chav1alpha1.SilenceSpec{
			Matcher: chav1alpha1.SilenceMatcher{Source: "StaleErrorPods"},
			Until:   metav1.Time{Time: until},
		},
		Status: chav1alpha1.SilenceStatus{Active: active, MatchCount: matchCount},
	}
}

func decodeStatus(t *testing.T, patch []byte) map[string]any {
	t.Helper()
	var body map[string]map[string]any
	if err := json.Unmarshal(patch, &body); err != nil {
		t.Fatalf("patch is not valid JSON: %v", err)
	}
	st, ok := body["status"]
	if !ok {
		t.Fatalf("patch carries no status key: %s", patch)
	}
	return st
}

func TestUpdateStatuses_ActivatesNewSilence(t *testing.T) {
	now := time.Now()
	p := &fakePatcher{}
	// Fresh CR: status.active is the zero value (false) but the window
	// is open — first cycle must flip it to true even with no matches.
	s := mkSilence("default", "mute-1", now.Add(time.Hour), false, 0)
	n, err := UpdateStatuses(context.Background(), p, []chav1alpha1.Silence{s}, nil, now)
	if err != nil || n != 1 {
		t.Fatalf("patched=%d err=%v; want 1, nil", n, err)
	}
	st := decodeStatus(t, p.calls[0].patch)
	if st["active"] != true {
		t.Errorf("active=%v want true", st["active"])
	}
	if _, ok := st["matchCount"]; ok {
		t.Errorf("matchCount must not be patched when nothing matched: %v", st)
	}
	if p.calls[0].gvr != silenceGVR || p.calls[0].ns != "default" || p.calls[0].name != "mute-1" {
		t.Errorf("patch target = %v %s/%s", p.calls[0].gvr, p.calls[0].ns, p.calls[0].name)
	}
}

func TestUpdateStatuses_ActiveFlipsFalseOnExpiry(t *testing.T) {
	now := time.Now()
	p := &fakePatcher{}
	s := mkSilence("default", "expired", now.Add(-time.Minute), true, 7)
	n, err := UpdateStatuses(context.Background(), p, []chav1alpha1.Silence{s}, nil, now)
	if err != nil || n != 1 {
		t.Fatalf("patched=%d err=%v; want 1, nil", n, err)
	}
	st := decodeStatus(t, p.calls[0].patch)
	if st["active"] != false {
		t.Errorf("active=%v want false after expiry", st["active"])
	}
	if _, ok := st["matchCount"]; ok {
		t.Errorf("matchCount must be left alone on a pure expiry flip: %v", st)
	}
}

func TestUpdateStatuses_MatchCountIncrements(t *testing.T) {
	now := time.Now()
	p := &fakePatcher{}
	s := mkSilence("ns1", "busy", now.Add(time.Hour), true, 5)
	counts := map[string]int{"ns1/busy": 3}
	n, err := UpdateStatuses(context.Background(), p, []chav1alpha1.Silence{s}, counts, now)
	if err != nil || n != 1 {
		t.Fatalf("patched=%d err=%v; want 1, nil", n, err)
	}
	st := decodeStatus(t, p.calls[0].patch)
	if got := st["matchCount"]; got != float64(8) { // 5 prior + 3 this cycle
		t.Errorf("matchCount=%v want 8 (running total)", got)
	}
	if _, ok := st["lastMatchAt"]; !ok {
		t.Errorf("lastMatchAt must be stamped when the silence matched: %v", st)
	}
}

func TestUpdateStatuses_NoOpPatchSuppressed(t *testing.T) {
	now := time.Now()
	p := &fakePatcher{}
	steady := []chav1alpha1.Silence{
		mkSilence("a", "active-unchanged", now.Add(time.Hour), true, 4),    // active, already true, no match
		mkSilence("b", "expired-unchanged", now.Add(-time.Hour), false, 0), // expired, already false
	}
	n, err := UpdateStatuses(context.Background(), p, steady, nil, now)
	if err != nil || n != 0 {
		t.Fatalf("patched=%d err=%v; want 0, nil", n, err)
	}
	if len(p.calls) != 0 {
		t.Errorf("steady state must patch nothing; got %d calls", len(p.calls))
	}
}

func TestUpdateStatuses_PatchErrorIsSoftAndScoped(t *testing.T) {
	now := time.Now()
	p := &fakePatcher{err: errors.New("apiserver down")}
	silences := []chav1alpha1.Silence{
		mkSilence("a", "s1", now.Add(time.Hour), false, 0),
		mkSilence("b", "s2", now.Add(time.Hour), false, 0),
	}
	n, err := UpdateStatuses(context.Background(), p, silences, nil, now)
	if n != 0 {
		t.Errorf("patched=%d want 0 (all failed)", n)
	}
	if err == nil {
		t.Fatal("want joined error, got nil")
	}
	if len(p.calls) != 2 {
		t.Errorf("one failure must not stop the walk; got %d calls, want 2", len(p.calls))
	}
}

func TestUpdateStatuses_NilPatcherNoOp(t *testing.T) {
	n, err := UpdateStatuses(context.Background(), nil, []chav1alpha1.Silence{mkSilence("a", "s", time.Now().Add(time.Hour), false, 0)}, nil, time.Now())
	if n != 0 || err != nil {
		t.Errorf("nil patcher must be a no-op; got %d, %v", n, err)
	}
}
