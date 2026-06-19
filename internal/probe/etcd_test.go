// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package probe

import (
	"context"
	"strings"
	"testing"
)

const etcdAllHealthy = `{
  "apiVersion": "v1", "kind": "PodList",
  "items": [
    {"apiVersion": "v1", "kind": "Pod",
     "metadata": {"name": "etcd-cp-01", "namespace": "kube-system",
                  "labels": {"component": "etcd", "tier": "control-plane"}},
     "status": {"phase": "Running",
                "conditions": [{"type": "Ready", "status": "True"}],
                "containerStatuses": [{"restartCount": 0}]}},
    {"apiVersion": "v1", "kind": "Pod",
     "metadata": {"name": "etcd-cp-02", "namespace": "kube-system",
                  "labels": {"component": "etcd", "tier": "control-plane"}},
     "status": {"phase": "Running",
                "conditions": [{"type": "Ready", "status": "True"}],
                "containerStatuses": [{"restartCount": 0}]}},
    {"apiVersion": "v1", "kind": "Pod",
     "metadata": {"name": "etcd-cp-03", "namespace": "kube-system",
                  "labels": {"component": "etcd", "tier": "control-plane"}},
     "status": {"phase": "Running",
                "conditions": [{"type": "Ready", "status": "True"}],
                "containerStatuses": [{"restartCount": 0}]}}
  ]
}`

const etcdOneMemberDown = `{
  "apiVersion": "v1", "kind": "PodList",
  "items": [
    {"apiVersion": "v1", "kind": "Pod",
     "metadata": {"name": "etcd-cp-01", "namespace": "kube-system",
                  "labels": {"component": "etcd"}},
     "status": {"phase": "Running",
                "conditions": [{"type": "Ready", "status": "True"}],
                "containerStatuses": [{"restartCount": 0}]}},
    {"apiVersion": "v1", "kind": "Pod",
     "metadata": {"name": "etcd-cp-02", "namespace": "kube-system",
                  "labels": {"component": "etcd"}},
     "status": {"phase": "Running",
                "conditions": [{"type": "Ready", "status": "False"}],
                "containerStatuses": [{"restartCount": 0}]}}
  ]
}`

const etcdMemberRestarting = `{
  "apiVersion": "v1", "kind": "PodList",
  "items": [
    {"apiVersion": "v1", "kind": "Pod",
     "metadata": {"name": "etcd-cp-01", "namespace": "kube-system",
                  "labels": {"component": "etcd"}},
     "status": {"phase": "Running",
                "conditions": [{"type": "Ready", "status": "True"}],
                "containerStatuses": [{"restartCount": 7}]}}
  ]
}`

const noEtcdPods = `{
  "apiVersion": "v1", "kind": "PodList",
  "items": [
    {"apiVersion": "v1", "kind": "Pod",
     "metadata": {"name": "coredns-abc", "namespace": "kube-system"},
     "status": {"phase": "Running"}}
  ]
}`

func TestETCD_AllHealthy(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{"pods.json": etcdAllHealthy})
	r := ETCD{}.Run(context.Background(), src)
	if r.Component.Status != "HEALTHY" {
		t.Errorf("Status = %q, want HEALTHY (detail=%q)", r.Component.Status, r.Component.Detail)
	}
	if !strings.Contains(r.Component.Detail, "3") {
		t.Errorf("detail should name member count: %q", r.Component.Detail)
	}
}

func TestETCD_MemberNotReady_Critical(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{"pods.json": etcdOneMemberDown})
	r := ETCD{}.Run(context.Background(), src)
	if r.Component.Status != "CRITICAL" {
		t.Errorf("Ready=False on a member must be CRITICAL, got %q", r.Component.Status)
	}
	if !strings.Contains(r.Component.Detail, "etcd-cp-02") {
		t.Errorf("detail must name the failing member: %q", r.Component.Detail)
	}
}

func TestETCD_MemberRestarting_Critical(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{"pods.json": etcdMemberRestarting})
	r := ETCD{}.Run(context.Background(), src)
	if r.Component.Status != "CRITICAL" {
		t.Errorf("restartCount > 0 on etcd must be CRITICAL, got %q", r.Component.Status)
	}
	if !strings.Contains(r.Component.Detail, "etcd-cp-01") || !strings.Contains(r.Component.Detail, "7") {
		t.Errorf("detail must name the restart count: %q", r.Component.Detail)
	}
}

func TestETCD_ExternalEtcd_Warning(t *testing.T) {
	// No etcd pods in kube-system. External etcd or managed control plane.
	// Must report Warning (probe is blind), not HEALTHY (false-green).
	src := loadProbeSrc(t, map[string]string{"pods.json": noEtcdPods})
	r := ETCD{}.Run(context.Background(), src)
	if r.Component.Status != "WARNING" {
		t.Errorf("no in-cluster etcd should be WARNING (blind probe), got %q (detail=%q)",
			r.Component.Status, r.Component.Detail)
	}
	if !strings.Contains(strings.ToLower(r.Findings[0].Message), "external etcd") {
		t.Errorf("finding should hint external etcd: %q", r.Findings[0].Message)
	}
}

func TestETCD_PodNameBasedMatch(t *testing.T) {
	// Older kubeadm versions don't set component=etcd label but the static
	// pod is named etcd-<node>. Match should still work.
	src := loadProbeSrc(t, map[string]string{"pods.json": `{
  "apiVersion": "v1", "kind": "PodList",
  "items": [{
    "apiVersion": "v1", "kind": "Pod",
    "metadata": {"name": "etcd-cp-01", "namespace": "kube-system"},
    "status": {"phase": "Running",
               "conditions": [{"type": "Ready", "status": "True"}],
               "containerStatuses": [{"restartCount": 0}]}
  }]
}`})
	r := ETCD{}.Run(context.Background(), src)
	if r.Component.Status != "HEALTHY" {
		t.Errorf("etcd-<name> static pod should be matched, got %q (detail=%q)",
			r.Component.Status, r.Component.Detail)
	}
}

func TestETCD_CustomNamespace(t *testing.T) {
	// Some installations isolate the control plane in a dedicated namespace.
	src := loadProbeSrc(t, map[string]string{"pods.json": `{
  "apiVersion": "v1", "kind": "PodList",
  "items": [{
    "apiVersion": "v1", "kind": "Pod",
    "metadata": {"name": "etcd-cp-01", "namespace": "control-plane",
                 "labels": {"component": "etcd"}},
    "status": {"phase": "Running",
               "conditions": [{"type": "Ready", "status": "True"}],
               "containerStatuses": [{"restartCount": 0}]}
  }]
}`})
	r := ETCD{Namespace: "control-plane"}.Run(context.Background(), src)
	if r.Component.Status != "HEALTHY" {
		t.Errorf("custom namespace should be honored, got %q", r.Component.Status)
	}
}

// podsEtcdTerminating has one terminating etcd pod (not-ready, 2 restarts).
// Since it's terminating (being replaced), the probe must not flag it.
const podsEtcdTerminating = `{
  "apiVersion": "v1", "kind": "PodList",
  "items": [
    {"apiVersion": "v1", "kind": "Pod",
     "metadata": {"name": "etcd-node-a", "namespace": "kube-system",
                  "deletionTimestamp": "2026-06-19T10:00:00Z"},
     "status": {"conditions": [{"type": "Ready", "status": "False"}],
                "containerStatuses": [{"restartCount": 2}]}}
  ]
}`

func TestETCD_TerminatingPodSkipped(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{"pods.json": podsEtcdTerminating})
	r := ETCD{}.Run(context.Background(), src)
	// With 0 non-terminating etcd pods, the probe should fall through to the
	// "external etcd" WARNING — NOT to CRITICAL.
	if r.Component.Status == "CRITICAL" {
		t.Errorf("Terminating etcd pod must not cause CRITICAL; got Status=%q Detail=%q",
			r.Component.Status, r.Component.Detail)
	}
	// Specifically must not have a "not ready" or "restarted" finding.
	for _, f := range r.Findings {
		if f.Severity == SeverityCritical {
			t.Errorf("must not produce a CRITICAL finding for terminating pod; got %q", f.Message)
		}
	}
}
