// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package report

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	"github.com/srenix-ai/agentic-sre/internal/diagnose"
	"github.com/srenix-ai/agentic-sre/internal/fix"
	"github.com/srenix-ai/agentic-sre/internal/probe"
	"github.com/srenix-ai/agentic-sre/internal/snapshot"
)

// fakeMutator records calls and lets the test inspect the create/patch/delete diff.
type fakeMutator struct {
	calls []string
	// patchBodies is keyed by call signature ("Patch driftreports//<name>"
	// or "PatchStatus driftreports//<name>") and holds the most recent patch
	// body sent to that signature. Lets regression tests assert the patch
	// targeted the right subresource AND carried the expected fields.
	patchBodies map[string][]byte
	// created holds every object passed to Create, in call order. Lets
	// regression tests assert the full spec/labels the writer sent.
	created []*unstructured.Unstructured
}

func (f *fakeMutator) Delete(_ context.Context, gvr schema.GroupVersionResource, ns, name string) error {
	f.calls = append(f.calls, fmt.Sprintf("Delete %s/%s/%s", gvr.Resource, ns, name))
	return nil
}
func (f *fakeMutator) Patch(_ context.Context, gvr schema.GroupVersionResource, ns, name string, _ types.PatchType, body []byte) error {
	key := fmt.Sprintf("Patch %s/%s/%s", gvr.Resource, ns, name)
	f.calls = append(f.calls, key)
	if f.patchBodies == nil {
		f.patchBodies = map[string][]byte{}
	}
	f.patchBodies[key] = body
	return nil
}
func (f *fakeMutator) PatchStatus(_ context.Context, gvr schema.GroupVersionResource, ns, name string, _ types.PatchType, body []byte) error {
	key := fmt.Sprintf("PatchStatus %s/%s/%s", gvr.Resource, ns, name)
	f.calls = append(f.calls, key)
	if f.patchBodies == nil {
		f.patchBodies = map[string][]byte{}
	}
	f.patchBodies[key] = body
	return nil
}
func (f *fakeMutator) Create(_ context.Context, gvr schema.GroupVersionResource, ns string, obj *unstructured.Unstructured) error {
	f.calls = append(f.calls, fmt.Sprintf("Create %s/%s/%s", gvr.Resource, ns, obj.GetName()))
	f.created = append(f.created, obj)
	return nil
}

// fakeSrc is a minimal Source that returns a fixed list for GVRDriftReport.
type fakeSrc struct {
	existing []unstructured.Unstructured
}

func (f *fakeSrc) List(_ context.Context, gvr schema.GroupVersionResource, _ string) (*unstructured.UnstructuredList, error) {
	if gvr == snapshot.GVRDriftReport {
		return &unstructured.UnstructuredList{Items: f.existing}, nil
	}
	return &unstructured.UnstructuredList{}, nil
}
func (f *fakeSrc) Get(_ context.Context, _ schema.GroupVersionResource, _, _ string) (*unstructured.Unstructured, error) {
	return nil, fmt.Errorf("not implemented")
}
func (f *fakeSrc) Mode() snapshot.Mode { return snapshot.ModeLive }

func driftCR(subject string) unstructured.Unstructured {
	return unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "srenix.ai/v1alpha1",
			"kind":       "DriftReport",
			"metadata":   map[string]any{"name": nameForSubject(subject)},
			"spec":       map[string]any{"subject": subject},
		},
	}
}

// driftCRWithStatus is driftCR plus a pre-populated status map. Used by
// observationCount-increment tests that need to start from a non-empty
// status to verify the read-modify-write path.
func driftCRWithStatus(subject string, status map[string]any) unstructured.Unstructured {
	cr := driftCR(subject)
	cr.Object["status"] = status
	return cr
}

func decodePatchBody(t *testing.T, body []byte) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("decode patch body: %v (body=%s)", err, string(body))
	}
	return m
}

func TestAssembleEntries(t *testing.T) {
	results := []probe.Result{{
		Component: probe.ComponentResult{Component: "Postgres", Status: "CRITICAL"},
		Findings: []probe.Finding{{
			Component: "Postgres (pg/main)",
			Severity:  probe.SeverityCritical,
			Message:   "phase=Failover",
		}},
	}}
	diag := []diagnose.Diagnostic{{Subject: "Secret/x/y/k", Message: "missing key"}}
	fixR := []fix.Result{{Fixer: "StaleErrorPods", Actions: []fix.Action{{Description: "deleted", Object: "Pod/ns/name"}}}}

	entries := AssembleEntries(results, diag, fixR)
	if len(entries) != 3 {
		t.Fatalf("want 3 entries, got %d", len(entries))
	}
	cats := []string{}
	for _, e := range entries {
		cats = append(cats, e.Category)
	}
	sort.Strings(cats)
	want := []string{"analyzer", "fixer-action", "probe"}
	for i, w := range want {
		if cats[i] != w {
			t.Errorf("entry[%d] category=%q, want %q", i, cats[i], w)
		}
	}
}

// TestAssembleEntries_DiagnosticRemediationCarriedThrough — analyzers
// compute rich Remediation text (e.g. PR #169's per-container digest
// substitution). The DriftReport reconciler previously dropped it on
// the analyzer→entry mapping, leaving spec.remediation empty on every
// digest-pin/etc DriftReport even though Slack/AM rendered it.
func TestAssembleEntries_DiagnosticRemediationCarriedThrough(t *testing.T) {
	diag := []diagnose.Diagnostic{{
		Subject:     "Pod/app/x-1",
		Source:      "SecurityDrift",
		Severity:    "warning",
		Message:     "Pod mounts container image(s) without digest pin",
		Remediation: "Replace `foo:1.2.3` with `foo@sha256:abc123…`",
	}}
	entries := AssembleEntries(nil, diag, nil)
	if len(entries) != 1 {
		t.Fatalf("want 1 entry; got %d", len(entries))
	}
	if entries[0].Remediation == "" {
		t.Errorf("remediation should be carried through; got empty")
	}
	if !strings.Contains(entries[0].Remediation, "sha256:abc123") {
		t.Errorf("remediation should contain the analyzer's text; got %q", entries[0].Remediation)
	}
}

func TestReconcile_CreateNew(t *testing.T) {
	src := &fakeSrc{}
	m := &fakeMutator{}
	entries := []DriftReportEntry{{Subject: "Secret/x/y/k", Severity: "warning", Source: "an", Category: "analyzer", Message: "msg"}}
	c, u, d, err := Reconcile(context.Background(), src, m, entries, "run-1")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if c != 1 || u != 0 || d != 0 {
		t.Errorf("want c=1 u=0 d=0; got c=%d u=%d d=%d", c, u, d)
	}
	// Should have one Create + one PatchStatus (the initial status stamp
	// MUST go through /status because the CRD declares subresources.status:{}
	// — a plain Patch with a status body would be silently dropped).
	if len(m.calls) != 2 {
		t.Fatalf("want 2 calls (create + status patch), got %d: %v", len(m.calls), m.calls)
	}
	if !strings.HasPrefix(m.calls[0], "Create ") {
		t.Errorf("first call should be Create, got: %s", m.calls[0])
	}
	if !strings.HasPrefix(m.calls[1], "PatchStatus ") {
		t.Errorf("second call must be PatchStatus (status subresource), got: %s", m.calls[1])
	}
}

func TestReconcile_UpdateExisting(t *testing.T) {
	subj := "Secret/x/y/k"
	src := &fakeSrc{existing: []unstructured.Unstructured{driftCR(subj)}}
	m := &fakeMutator{}
	entries := []DriftReportEntry{{Subject: subj, Severity: "warning", Source: "an", Category: "analyzer", Message: "msg"}}
	c, u, d, err := Reconcile(context.Background(), src, m, entries, "run-2")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if c != 0 || u != 1 || d != 0 {
		t.Errorf("want c=0 u=1 d=0; got c=%d u=%d d=%d", c, u, d)
	}
	// Should be exactly one Patch (spec) + one PatchStatus (status).
	// The two calls are required because the CRD's subresources.status:{}
	// means a single combined patch would silently drop the status fields.
	if len(m.calls) != 2 {
		t.Fatalf("want 2 calls (spec patch + status patch), got %d: %v", len(m.calls), m.calls)
	}
	if !strings.HasPrefix(m.calls[0], "Patch ") {
		t.Errorf("first call must be Patch (main resource), got: %s", m.calls[0])
	}
	if !strings.HasPrefix(m.calls[1], "PatchStatus ") {
		t.Errorf("second call must be PatchStatus (status subresource), got: %s", m.calls[1])
	}
}

func TestReconcile_DeleteResolved(t *testing.T) {
	subj := "Secret/x/y/k"
	src := &fakeSrc{existing: []unstructured.Unstructured{driftCR(subj)}}
	m := &fakeMutator{}
	c, u, d, err := Reconcile(context.Background(), src, m, nil, "run-3")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if c != 0 || u != 0 || d != 1 {
		t.Errorf("want c=0 u=0 d=1; got c=%d u=%d d=%d", c, u, d)
	}
	if len(m.calls) != 1 || m.calls[0][:6] != "Delete" {
		t.Errorf("want 1 delete, got %v", m.calls)
	}
}

func TestReconcile_RefusesWithoutMutator(t *testing.T) {
	_, _, _, err := Reconcile(context.Background(), &fakeSrc{}, nil, nil, "x")
	if err == nil {
		t.Errorf("expected error when Mutator is nil")
	}
}

// TestReconcile_StatusFieldsLandOnStatusSubresource is the regression test
// for the silent-status-drop bug. The CRD declares subresources.status:{},
// so the API server drops status fields in patches sent to the main
// resource endpoint. Reconcile MUST send status payloads via the /status
// subresource (mut.PatchStatus). This test asserts both the routing AND
// the body shape so the bug can't recur silently.
func TestReconcile_StatusFieldsLandOnStatusSubresource(t *testing.T) {
	subj := "Secret/x/y/k"
	crName := nameForSubject(subj)

	// CREATE path
	t.Run("create", func(t *testing.T) {
		src := &fakeSrc{}
		m := &fakeMutator{}
		entries := []DriftReportEntry{{Subject: subj, Severity: "warning", Source: "an", Category: "analyzer", Message: "msg"}}
		if _, _, _, err := Reconcile(context.Background(), src, m, entries, "run-create"); err != nil {
			t.Fatalf("Reconcile: %v", err)
		}
		key := fmt.Sprintf("PatchStatus driftreports//%s", crName)
		body, ok := m.patchBodies[key]
		if !ok {
			t.Fatalf("no PatchStatus call recorded; calls=%v", m.calls)
		}
		decoded := decodePatchBody(t, body)
		status, _ := decoded["status"].(map[string]any)
		if status == nil {
			t.Fatalf("status patch body has no status field: %s", body)
		}
		for _, field := range []string{"firstObserved", "lastObserved", "observationCount", "runID"} {
			if _, ok := status[field]; !ok {
				t.Errorf("create status patch missing %q (body=%s)", field, body)
			}
		}
		if got, _ := status["observationCount"].(float64); got != 1 {
			t.Errorf("observationCount=%v want 1 on create", status["observationCount"])
		}
		if got, _ := status["runID"].(string); got != "run-create" {
			t.Errorf("runID=%v want run-create", status["runID"])
		}
		// The status body must NOT carry spec fields (would be silently
		// dropped by the /status endpoint, and signals intent confusion).
		if _, hasSpec := decoded["spec"]; hasSpec {
			t.Errorf("status patch must not carry a spec field; body=%s", body)
		}
	})

	// UPDATE path — also asserts that spec patch and status patch are split
	t.Run("update", func(t *testing.T) {
		src := &fakeSrc{existing: []unstructured.Unstructured{driftCR(subj)}}
		m := &fakeMutator{}
		entries := []DriftReportEntry{{Subject: subj, Severity: "critical", Source: "an", Category: "analyzer", Message: "new msg", Remediation: "fix it", Investigation: "looked at logs"}}
		if _, _, _, err := Reconcile(context.Background(), src, m, entries, "run-update"); err != nil {
			t.Fatalf("Reconcile: %v", err)
		}

		// Spec patch: severity/message/remediation/investigation, no status.
		specBody, ok := m.patchBodies[fmt.Sprintf("Patch driftreports//%s", crName)]
		if !ok {
			t.Fatalf("no spec Patch call recorded; calls=%v", m.calls)
		}
		specDecoded := decodePatchBody(t, specBody)
		spec, _ := specDecoded["spec"].(map[string]any)
		if spec == nil {
			t.Fatalf("spec patch missing spec field: %s", specBody)
		}
		if got, _ := spec["severity"].(string); got != "critical" {
			t.Errorf("spec.severity=%v want critical", spec["severity"])
		}
		if got, _ := spec["message"].(string); got != "new msg" {
			t.Errorf("spec.message=%v want 'new msg'", spec["message"])
		}
		if _, hasStatus := specDecoded["status"]; hasStatus {
			t.Errorf("spec patch must not carry status (silent-drop bug); body=%s", specBody)
		}

		// Status patch: lastObserved/observationCount/runID, no spec.
		statusBody, ok := m.patchBodies[fmt.Sprintf("PatchStatus driftreports//%s", crName)]
		if !ok {
			t.Fatalf("no PatchStatus call recorded; calls=%v", m.calls)
		}
		statusDecoded := decodePatchBody(t, statusBody)
		status, _ := statusDecoded["status"].(map[string]any)
		if status == nil {
			t.Fatalf("status patch missing status field: %s", statusBody)
		}
		for _, field := range []string{"lastObserved", "observationCount", "runID"} {
			if _, ok := status[field]; !ok {
				t.Errorf("update status patch missing %q (body=%s)", field, statusBody)
			}
		}
		// firstObserved must NOT be re-stamped on updates — it freezes at creation time.
		if _, hasFirst := status["firstObserved"]; hasFirst {
			t.Errorf("update status patch must not re-stamp firstObserved; body=%s", statusBody)
		}
		if _, hasSpec := statusDecoded["spec"]; hasSpec {
			t.Errorf("status patch must not carry a spec field; body=%s", statusBody)
		}
	})
}

// TestReconcile_ObservationCountIncrements verifies the read-modify-write
// path: an existing CR with status.observationCount = 5 yields a patch
// body with observationCount = 6. Prior to the /status subresource fix
// this counter was effectively stuck at 1 (status patches silently
// dropped, so oldCount always read as 0).
func TestReconcile_ObservationCountIncrements(t *testing.T) {
	subj := "Secret/x/y/k"
	crName := nameForSubject(subj)
	cases := []struct {
		name       string
		oldCount   any // simulate both int64 (typed) and float64 (JSON-decoded) shapes
		wantNewVal float64
	}{
		{"int64", int64(5), 6},
		{"float64", float64(5), 6},
		{"empty status (no field)", nil, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status := map[string]any{}
			if tc.oldCount != nil {
				status["observationCount"] = tc.oldCount
			}
			src := &fakeSrc{existing: []unstructured.Unstructured{driftCRWithStatus(subj, status)}}
			m := &fakeMutator{}
			entries := []DriftReportEntry{{Subject: subj, Severity: "warning", Source: "an", Category: "analyzer", Message: "msg"}}
			if _, _, _, err := Reconcile(context.Background(), src, m, entries, "run-inc"); err != nil {
				t.Fatalf("Reconcile: %v", err)
			}
			body, ok := m.patchBodies[fmt.Sprintf("PatchStatus driftreports//%s", crName)]
			if !ok {
				t.Fatalf("no PatchStatus call recorded; calls=%v", m.calls)
			}
			decoded := decodePatchBody(t, body)
			s, _ := decoded["status"].(map[string]any)
			if got, _ := s["observationCount"].(float64); got != tc.wantNewVal {
				t.Errorf("observationCount=%v want %v (body=%s)", s["observationCount"], tc.wantNewVal, body)
			}
		})
	}
}

func TestSanitizeLabel(t *testing.T) {
	cases := map[string]string{
		"Postgres":             "postgres",
		"Service: LiveKit SIP": "service--livekit-sip",
		"":                     "unknown",
		"AAA-bbb_ccc":          "aaa-bbb-ccc",
	}
	for in, want := range cases {
		if got := sanitizeLabel(in); got != want {
			t.Errorf("sanitizeLabel(%q) = %q, want %q", in, got, want)
		}
	}
}
