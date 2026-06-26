// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package diagnose

import (
	"context"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func makeFailedPod(ns, name, phase, podReason, containerReason string, deleting bool) unstructured.Unstructured {
	u := unstructured.Unstructured{}
	u.SetAPIVersion("v1")
	u.SetKind("Pod")
	u.SetNamespace(ns)
	u.SetName(name)
	if deleting {
		now := metav1.Now()
		u.SetDeletionTimestamp(&now)
	}
	_ = unstructured.SetNestedField(u.Object, phase, "status", "phase")
	if podReason != "" {
		_ = unstructured.SetNestedField(u.Object, podReason, "status", "reason")
	}
	if containerReason != "" {
		cs := []any{map[string]any{
			"name": "engine",
			"state": map[string]any{
				"terminated": map[string]any{"reason": containerReason},
			},
		}}
		_ = unstructured.SetNestedSlice(u.Object, cs, "status", "containerStatuses")
	}
	return u
}

func TestFailedPods_UnexpectedAdmissionError_Critical(t *testing.T) {
	src := &memSourceDD{byResource: map[string][]unstructured.Unstructured{
		"pods": {
			makeFailedPod("immersive", "immersive-engine-x", "Failed", "UnexpectedAdmissionError", "ContainerStatusUnknown", false),
		},
	}}
	got := FailedPods{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("expected 1 diagnostic; got %d", len(got))
	}
	if got[0].Severity != "critical" {
		t.Errorf("severity: %q", got[0].Severity)
	}
	if got[0].Subject != "Pod/immersive/immersive-engine-x" {
		t.Errorf("subject: %q", got[0].Subject)
	}
	if !strings.Contains(got[0].Message, "UnexpectedAdmissionError") {
		t.Errorf("message should name the pod reason: %q", got[0].Message)
	}
	if !strings.Contains(got[0].Message, "ContainerStatusUnknown") {
		t.Errorf("message should enrich with the container reason: %q", got[0].Message)
	}
}

func TestFailedPods_RunningPod_NoFinding(t *testing.T) {
	src := &memSourceDD{byResource: map[string][]unstructured.Unstructured{
		"pods": {makeFailedPod("ns", "ok", "Running", "", "", false)},
	}}
	if got := (FailedPods{}).Run(context.Background(), src); len(got) != 0 {
		t.Fatalf("Running pod must not fire; got %d", len(got))
	}
}

func TestFailedPods_Terminating_Skipped(t *testing.T) {
	src := &memSourceDD{byResource: map[string][]unstructured.Unstructured{
		"pods": {makeFailedPod("ns", "going-away", "Failed", "Evicted", "", true)},
	}}
	if got := (FailedPods{}).Run(context.Background(), src); len(got) != 0 {
		t.Fatalf("terminating Failed pod must be skipped; got %d", len(got))
	}
}

func TestFailedPods_Evicted_Critical(t *testing.T) {
	src := &memSourceDD{byResource: map[string][]unstructured.Unstructured{
		"pods": {makeFailedPod("ns", "evicted-1", "Failed", "Evicted", "", false)},
	}}
	got := FailedPods{}.Run(context.Background(), src)
	if len(got) != 1 || got[0].Severity != "critical" {
		t.Fatalf("Evicted pod should fire critical; got %+v", got)
	}
}

func TestFailedPods_Name(t *testing.T) {
	if (FailedPods{}).Name() != "FailedPods" {
		t.Errorf("name: %q", (FailedPods{}).Name())
	}
}
