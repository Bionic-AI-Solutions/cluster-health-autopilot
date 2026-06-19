// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package probe

import (
	"context"
	"strings"
	"testing"
)

const podsNoCrash = `{
  "apiVersion": "v1", "kind": "PodList",
  "items": [
    {"apiVersion": "v1", "kind": "Pod",
     "metadata": {"name": "happy", "namespace": "demo"},
     "status": {"phase": "Running",
                "containerStatuses": [{"restartCount": 0, "state": {"running": {}}}]}}
  ]
}`

const podUserNsLowRestarts = `{
  "apiVersion": "v1", "kind": "PodList",
  "items": [
    {"apiVersion": "v1", "kind": "Pod",
     "metadata": {"name": "wobbly", "namespace": "demo"},
     "status": {"phase": "Running",
                "containerStatuses": [{
                  "restartCount": 3,
                  "state": {"waiting": {"reason": "CrashLoopBackOff"}}
                }]}}
  ]
}`

const podUserNsHighRestarts = `{
  "apiVersion": "v1", "kind": "PodList",
  "items": [
    {"apiVersion": "v1", "kind": "Pod",
     "metadata": {"name": "doomed", "namespace": "demo"},
     "status": {"phase": "Running",
                "containerStatuses": [{
                  "restartCount": 25,
                  "state": {"waiting": {"reason": "CrashLoopBackOff"}}
                }]}}
  ]
}`

const podProtectedNsAnyRestarts = `{
  "apiVersion": "v1", "kind": "PodList",
  "items": [
    {"apiVersion": "v1", "kind": "Pod",
     "metadata": {"name": "coredns", "namespace": "kube-system"},
     "status": {"phase": "Running",
                "containerStatuses": [{
                  "restartCount": 1,
                  "state": {"waiting": {"reason": "CrashLoopBackOff"}}
                }]}}
  ]
}`

const podInitContainerCrashLoop = `{
  "apiVersion": "v1", "kind": "PodList",
  "items": [
    {"apiVersion": "v1", "kind": "Pod",
     "metadata": {"name": "init-broken", "namespace": "demo"},
     "status": {"phase": "Pending",
                "initContainerStatuses": [{
                  "restartCount": 4,
                  "state": {"waiting": {"reason": "CrashLoopBackOff"}}
                }]}}
  ]
}`

const podCrashedThenRecovered = `{
  "apiVersion": "v1", "kind": "PodList",
  "items": [
    {"apiVersion": "v1", "kind": "Pod",
     "metadata": {"name": "recovered", "namespace": "demo"},
     "status": {"phase": "Running",
                "containerStatuses": [{
                  "restartCount": 5,
                  "state": {"running": {}}
                }]}}
  ]
}`

func TestCrashLoop_NoCrash(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{"pods.json": podsNoCrash})
	r := CrashLoopBackOff{}.Run(context.Background(), src)
	if r.Component.Status != "HEALTHY" {
		t.Errorf("Status = %q, want HEALTHY", r.Component.Status)
	}
}

func TestCrashLoop_UserNs_LowRestarts_Warning(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{"pods.json": podUserNsLowRestarts})
	r := CrashLoopBackOff{}.Run(context.Background(), src)
	if r.Component.Status != "WARNING" {
		t.Errorf("low-restart user-ns crash should be WARNING, got %q", r.Component.Status)
	}
	if len(r.Findings) != 1 || r.Findings[0].Severity != SeverityWarning {
		t.Errorf("expected 1 Warning finding, got %+v", r.Findings)
	}
}

func TestCrashLoop_UserNs_HighRestarts_Critical(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{"pods.json": podUserNsHighRestarts})
	r := CrashLoopBackOff{}.Run(context.Background(), src)
	if r.Component.Status != "CRITICAL" {
		t.Errorf("25 restarts > threshold(10) should be CRITICAL, got %q", r.Component.Status)
	}
}

func TestCrashLoop_ProtectedNs_AlwaysCritical(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{"pods.json": podProtectedNsAnyRestarts})
	r := CrashLoopBackOff{}.Run(context.Background(), src)
	if r.Component.Status != "CRITICAL" {
		t.Errorf("kube-system crash with 1 restart should be CRITICAL, got %q", r.Component.Status)
	}
}

func TestCrashLoop_InitContainer(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{"pods.json": podInitContainerCrashLoop})
	r := CrashLoopBackOff{}.Run(context.Background(), src)
	if r.Component.Status == "HEALTHY" {
		t.Errorf("init-container CrashLoop should be flagged; got HEALTHY (detail=%q)",
			r.Component.Detail)
	}
	if !strings.Contains(r.Findings[0].Message, "init-broken") {
		t.Errorf("finding should name the pod: %q", r.Findings[0].Message)
	}
}

func TestCrashLoop_RecoveredPodNotFlagged(t *testing.T) {
	// A pod that crashed (high restart count) but is currently Running is
	// recovered. Don't flag it.
	src := loadProbeSrc(t, map[string]string{"pods.json": podCrashedThenRecovered})
	r := CrashLoopBackOff{}.Run(context.Background(), src)
	if r.Component.Status != "HEALTHY" {
		t.Errorf("recovered pod (currently Running) shouldn't be flagged; got %q",
			r.Component.Status)
	}
}

func TestCrashLoop_CustomThreshold(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{"pods.json": podUserNsLowRestarts})
	// Operator sets threshold=1; this pod has 3 restarts → should escalate.
	r := CrashLoopBackOff{CriticalRestartThreshold: 1}.Run(context.Background(), src)
	if r.Component.Status != "CRITICAL" {
		t.Errorf("threshold=1 + 3 restarts should be CRITICAL, got %q", r.Component.Status)
	}
}

func TestCrashLoop_EmptyCluster(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{})
	r := CrashLoopBackOff{}.Run(context.Background(), src)
	if r.Component.Status != "HEALTHY" {
		t.Errorf("empty cluster should be HEALTHY, got %q", r.Component.Status)
	}
}

// podTerminatingCrashLoop is a pod with deletionTimestamp set AND
// CrashLoopBackOff — should be silently skipped.
const podTerminatingCrashLoop = `{
  "apiVersion": "v1", "kind": "PodList",
  "items": [
    {"apiVersion": "v1", "kind": "Pod",
     "metadata": {"name": "old-pod", "namespace": "demo",
                  "deletionTimestamp": "2026-06-19T10:00:00Z"},
     "status": {"phase": "Running",
                "containerStatuses": [{
                  "restartCount": 5,
                  "state": {"waiting": {"reason": "CrashLoopBackOff"}}
                }]}}
  ]
}`

func TestCrashLoop_TerminatingPodSkipped(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{"pods.json": podTerminatingCrashLoop})
	r := CrashLoopBackOff{}.Run(context.Background(), src)
	if r.Component.Status != "HEALTHY" {
		t.Errorf("Terminating pod with CrashLoopBackOff must be skipped; got Status=%q Detail=%q",
			r.Component.Status, r.Component.Detail)
	}
	if len(r.Findings) != 0 {
		t.Errorf("expected 0 findings for terminating pod, got %+v", r.Findings)
	}
}

func TestCrashLoop_NonTerminatingCrashLoopStillFlagged(t *testing.T) {
	// Regression: removing the deletionTimestamp means the pod IS flagged.
	src := loadProbeSrc(t, map[string]string{"pods.json": podUserNsLowRestarts})
	r := CrashLoopBackOff{}.Run(context.Background(), src)
	if r.Component.Status == "HEALTHY" {
		t.Errorf("non-terminating CrashLoopBackOff pod must still be flagged; got HEALTHY")
	}
}
