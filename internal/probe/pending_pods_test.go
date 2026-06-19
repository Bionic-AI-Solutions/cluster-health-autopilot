// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package probe

import (
	"context"
	"strings"
	"testing"
	"time"
)

// fixedNow returns a clock function the tests can plug into PendingPods.
func fixedNow(s string) func() time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic("bad fixed time: " + err.Error())
	}
	return func() time.Time { return t }
}

const podsAllScheduled = `{
  "apiVersion": "v1", "kind": "PodList",
  "items": [
    {"apiVersion": "v1", "kind": "Pod",
     "metadata": {"name": "running", "namespace": "demo",
                  "creationTimestamp": "2026-05-22T10:00:00Z"},
     "status": {"phase": "Running"}}
  ]
}`

const podStuckInsufficientCPU = `{
  "apiVersion": "v1", "kind": "PodList",
  "items": [
    {"apiVersion": "v1", "kind": "Pod",
     "metadata": {"name": "hungry", "namespace": "demo",
                  "creationTimestamp": "2026-05-22T10:00:00Z"},
     "status": {"phase": "Pending",
                "conditions": [
                  {"type": "PodScheduled", "status": "False",
                   "reason": "Unschedulable",
                   "message": "0/4 nodes are available: 4 Insufficient cpu."}
                ]}}
  ]
}`

const podStuckImagePull = `{
  "apiVersion": "v1", "kind": "PodList",
  "items": [
    {"apiVersion": "v1", "kind": "Pod",
     "metadata": {"name": "bad-image", "namespace": "demo",
                  "creationTimestamp": "2026-05-22T10:00:00Z"},
     "status": {"phase": "Pending",
                "containerStatuses": [
                  {"state": {"waiting": {"reason": "ImagePullBackOff"}}}
                ]}}
  ]
}`

const podPendingButYoung = `{
  "apiVersion": "v1", "kind": "PodList",
  "items": [
    {"apiVersion": "v1", "kind": "Pod",
     "metadata": {"name": "starting", "namespace": "demo",
                  "creationTimestamp": "2026-05-22T11:59:30Z"},
     "status": {"phase": "Pending",
                "conditions": [
                  {"type": "PodScheduled", "status": "False",
                   "reason": "Unschedulable",
                   "message": "0/4 nodes are available: 4 Insufficient cpu."}
                ]}}
  ]
}`

const podPVCUnbound = `{
  "apiVersion": "v1", "kind": "PodList",
  "items": [
    {"apiVersion": "v1", "kind": "Pod",
     "metadata": {"name": "needs-storage", "namespace": "demo",
                  "creationTimestamp": "2026-05-22T10:00:00Z"},
     "status": {"phase": "Pending",
                "conditions": [
                  {"type": "PodScheduled", "status": "False",
                   "reason": "Unschedulable",
                   "message": "pod has unbound immediate PersistentVolumeClaims"}
                ]}}
  ]
}`

const podTaintMismatch = `{
  "apiVersion": "v1", "kind": "PodList",
  "items": [
    {"apiVersion": "v1", "kind": "Pod",
     "metadata": {"name": "non-tol", "namespace": "demo",
                  "creationTimestamp": "2026-05-22T10:00:00Z"},
     "status": {"phase": "Pending",
                "conditions": [
                  {"type": "PodScheduled", "status": "False",
                   "reason": "Unschedulable",
                   "message": "0/4 nodes are available: 4 node(s) had taint {dedicated: gpu}, that the pod didn't tolerate."}
                ]}}
  ]
}`

func TestPendingPods_AllScheduled(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{"pods.json": podsAllScheduled})
	r := PendingPods{Now: fixedNow("2026-05-22T12:00:00Z")}.Run(context.Background(), src)
	if r.Component.Status != "HEALTHY" {
		t.Errorf("Status = %q, want HEALTHY", r.Component.Status)
	}
	if len(r.Findings) != 0 {
		t.Errorf("expected no findings, got %+v", r.Findings)
	}
}

func TestPendingPods_InsufficientCPU_Critical(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{"pods.json": podStuckInsufficientCPU})
	r := PendingPods{Now: fixedNow("2026-05-22T12:00:00Z")}.Run(context.Background(), src)
	if r.Component.Status != "CRITICAL" {
		t.Errorf("Pending pod past grace period must be CRITICAL, got %q", r.Component.Status)
	}
	if len(r.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %+v", r.Findings)
	}
	if !strings.Contains(r.Findings[0].Message, "Insufficient cpu") {
		t.Errorf("message missing CPU detail: %q", r.Findings[0].Message)
	}
	if !strings.Contains(r.Findings[0].Remediation, "CPU") {
		t.Errorf("remediation missing CPU advice: %q", r.Findings[0].Remediation)
	}
}

func TestPendingPods_ImagePullBackOffIgnored(t *testing.T) {
	// ImagePullAuth handles this class; PendingPods must not duplicate the alert.
	src := loadProbeSrc(t, map[string]string{"pods.json": podStuckImagePull})
	r := PendingPods{Now: fixedNow("2026-05-22T12:00:00Z")}.Run(context.Background(), src)
	if r.Component.Status != "HEALTHY" {
		t.Errorf("ImagePullBackOff is owned by ImagePullAuth — must not be flagged here; got %q",
			r.Component.Status)
	}
}

func TestPendingPods_YoungPodSkipped(t *testing.T) {
	// Pod created 30s ago; default grace is 60s. Don't flag.
	src := loadProbeSrc(t, map[string]string{"pods.json": podPendingButYoung})
	r := PendingPods{Now: fixedNow("2026-05-22T12:00:00Z")}.Run(context.Background(), src)
	if r.Component.Status != "HEALTHY" {
		t.Errorf("young Pending pod should be inside grace period; got %q (detail=%q)",
			r.Component.Status, r.Component.Detail)
	}
}

func TestPendingPods_PVCUnbound_NamedInRemediation(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{"pods.json": podPVCUnbound})
	r := PendingPods{Now: fixedNow("2026-05-22T12:00:00Z")}.Run(context.Background(), src)
	if r.Component.Status != "CRITICAL" {
		t.Errorf("unbound PVC should escalate to CRITICAL, got %q", r.Component.Status)
	}
	if !strings.Contains(strings.ToLower(r.Findings[0].Remediation), "pvc") {
		t.Errorf("remediation should mention PVC: %q", r.Findings[0].Remediation)
	}
}

func TestPendingPods_TaintMismatch_NamedInRemediation(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{"pods.json": podTaintMismatch})
	r := PendingPods{Now: fixedNow("2026-05-22T12:00:00Z")}.Run(context.Background(), src)
	if !strings.Contains(strings.ToLower(r.Findings[0].Remediation), "toleration") {
		t.Errorf("remediation should mention toleration: %q", r.Findings[0].Remediation)
	}
}

func TestPendingPods_NoPods(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{})
	r := PendingPods{Now: fixedNow("2026-05-22T12:00:00Z")}.Run(context.Background(), src)
	if r.Component.Status != "HEALTHY" {
		t.Errorf("empty cluster should be HEALTHY, got %q", r.Component.Status)
	}
}

// podTerminatingPending is a Pending pod with PodScheduled=False AND a
// deletionTimestamp — should be silently skipped.
const podTerminatingPending = `{
  "apiVersion": "v1", "kind": "PodList",
  "items": [
    {"apiVersion": "v1", "kind": "Pod",
     "metadata": {"name": "old-pending", "namespace": "demo",
                  "creationTimestamp": "2026-05-22T10:00:00Z",
                  "deletionTimestamp": "2026-06-19T10:00:00Z"},
     "status": {"phase": "Pending",
                "conditions": [
                  {"type": "PodScheduled", "status": "False",
                   "reason": "Unschedulable",
                   "message": "0/4 nodes are available: 4 Insufficient cpu."}
                ]}}
  ]
}`

func TestPendingPods_TerminatingPodSkipped(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{"pods.json": podTerminatingPending})
	r := PendingPods{Now: fixedNow("2026-05-22T12:00:00Z")}.Run(context.Background(), src)
	if r.Component.Status != "HEALTHY" {
		t.Errorf("Terminating pending pod must be skipped; got Status=%q Detail=%q",
			r.Component.Status, r.Component.Detail)
	}
	if len(r.Findings) != 0 {
		t.Errorf("expected 0 findings for terminating pod, got %+v", r.Findings)
	}
}

func TestPendingPods_NonTerminatingPendingStillFlagged(t *testing.T) {
	// Regression: same pod without deletionTimestamp is still flagged.
	src := loadProbeSrc(t, map[string]string{"pods.json": podStuckInsufficientCPU})
	r := PendingPods{Now: fixedNow("2026-05-22T12:00:00Z")}.Run(context.Background(), src)
	if r.Component.Status != "CRITICAL" {
		t.Errorf("non-terminating Pending pod must still be flagged; got Status=%q", r.Component.Status)
	}
}
