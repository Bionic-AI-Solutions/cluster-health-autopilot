// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package probe

import (
	"context"
	"strings"
	"testing"
)

const dsAllHealthy = `{
  "apiVersion": "v1", "kind": "List",
  "items": [
    {"apiVersion": "apps/v1", "kind": "DaemonSet",
     "metadata": {"name": "cilium", "namespace": "kube-system"},
     "status": {"desiredNumberScheduled": 4, "numberReady": 4, "numberUnavailable": 0}},
    {"apiVersion": "apps/v1", "kind": "DaemonSet",
     "metadata": {"name": "kube-proxy", "namespace": "kube-system"},
     "status": {"desiredNumberScheduled": 4, "numberReady": 4}},
    {"apiVersion": "apps/v1", "kind": "DaemonSet",
     "metadata": {"name": "csi-rbdplugin", "namespace": "rook-ceph"},
     "status": {"desiredNumberScheduled": 3, "numberReady": 3}}
  ]
}`

const dsCNIDegraded = `{
  "apiVersion": "v1", "kind": "List",
  "items": [
    {"apiVersion": "apps/v1", "kind": "DaemonSet",
     "metadata": {"name": "cilium", "namespace": "kube-system"},
     "status": {"desiredNumberScheduled": 4, "numberReady": 2, "numberUnavailable": 2}},
    {"apiVersion": "apps/v1", "kind": "DaemonSet",
     "metadata": {"name": "kube-proxy", "namespace": "kube-system"},
     "status": {"desiredNumberScheduled": 4, "numberReady": 4}}
  ]
}`

const dsZeroDesired = `{
  "apiVersion": "v1", "kind": "List",
  "items": [
    {"apiVersion": "apps/v1", "kind": "DaemonSet",
     "metadata": {"name": "gpu-driver", "namespace": "kube-system"},
     "status": {"desiredNumberScheduled": 0, "numberReady": 0}}
  ]
}`

const dsIgnoredNamespace = `{
  "apiVersion": "v1", "kind": "List",
  "items": [
    {"apiVersion": "apps/v1", "kind": "DaemonSet",
     "metadata": {"name": "user-ds", "namespace": "demo"},
     "status": {"desiredNumberScheduled": 3, "numberReady": 0, "numberUnavailable": 3}}
  ]
}`

const dsMultipleSystemNs = `{
  "apiVersion": "v1", "kind": "List",
  "items": [
    {"apiVersion": "apps/v1", "kind": "DaemonSet",
     "metadata": {"name": "calico-node", "namespace": "calico-system"},
     "status": {"desiredNumberScheduled": 4, "numberReady": 4}},
    {"apiVersion": "apps/v1", "kind": "DaemonSet",
     "metadata": {"name": "csi-cephfsplugin", "namespace": "rook-ceph"},
     "status": {"desiredNumberScheduled": 3, "numberReady": 1, "numberUnavailable": 2}}
  ]
}`

func TestDaemonSets_AllHealthy(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{"apps-daemonsets.json": dsAllHealthy})
	r := DaemonSets{}.Run(context.Background(), src)
	if r.Component.Status != "HEALTHY" {
		t.Errorf("Status = %q, want HEALTHY (detail=%q)", r.Component.Status, r.Component.Detail)
	}
	if len(r.Findings) != 0 {
		t.Errorf("expected no findings, got %+v", r.Findings)
	}
}

func TestDaemonSets_CNIDegraded_Critical(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{"apps-daemonsets.json": dsCNIDegraded})
	r := DaemonSets{}.Run(context.Background(), src)
	if r.Component.Status != "CRITICAL" {
		t.Errorf("degraded CNI must be CRITICAL, got %q", r.Component.Status)
	}
	if !strings.Contains(r.Component.Detail, "cilium") || !strings.Contains(r.Component.Detail, "2/4") {
		t.Errorf("detail missing cilium/2-of-4: %q", r.Component.Detail)
	}
	if len(r.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %d (%+v)", len(r.Findings), r.Findings)
	}
}

func TestDaemonSets_ZeroDesiredIgnored(t *testing.T) {
	// A DaemonSet whose nodeSelector matches no nodes is intentionally idle.
	// Not a failure.
	src := loadProbeSrc(t, map[string]string{"apps-daemonsets.json": dsZeroDesired})
	r := DaemonSets{}.Run(context.Background(), src)
	if r.Component.Status != "HEALTHY" {
		t.Errorf("zero-desired DS should not be flagged; got %q (detail=%q)",
			r.Component.Status, r.Component.Detail)
	}
}

func TestDaemonSets_IgnoresNonSystemNamespaces(t *testing.T) {
	// A user-namespace DS with all pods down is the workload owner's problem,
	// not a system-level crisis. Don't surface here.
	src := loadProbeSrc(t, map[string]string{"apps-daemonsets.json": dsIgnoredNamespace})
	r := DaemonSets{}.Run(context.Background(), src)
	if r.Component.Status != "HEALTHY" {
		t.Errorf("user-ns DS should not be inspected; got %q", r.Component.Status)
	}
}

func TestDaemonSets_MultipleSystemNamespacesCovered(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{"apps-daemonsets.json": dsMultipleSystemNs})
	r := DaemonSets{}.Run(context.Background(), src)
	if r.Component.Status != "CRITICAL" {
		t.Errorf("rook-ceph CSI degraded should be CRITICAL, got %q (detail=%q)",
			r.Component.Status, r.Component.Detail)
	}
	// Should name the rook-ceph DS but not the healthy calico-system one.
	if !strings.Contains(r.Component.Detail, "rook-ceph") {
		t.Errorf("detail missing rook-ceph: %q", r.Component.Detail)
	}
	if strings.Contains(r.Component.Detail, "calico-node") {
		t.Errorf("healthy calico-node should not appear in detail: %q", r.Component.Detail)
	}
}

func TestDaemonSets_CustomSystemNamespaces(t *testing.T) {
	// Operator overrides the default system-ns list to track only their CNI.
	src := loadProbeSrc(t, map[string]string{"apps-daemonsets.json": dsMultipleSystemNs})
	d := DaemonSets{SystemNamespaces: []string{"calico-system"}}
	r := d.Run(context.Background(), src)
	if r.Component.Status != "HEALTHY" {
		t.Errorf("with calico-system only, broken rook-ceph DS is out of scope; got %q (detail=%q)",
			r.Component.Status, r.Component.Detail)
	}
}

func TestDaemonSets_NoDaemonSetsAtAll(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{})
	r := DaemonSets{}.Run(context.Background(), src)
	// No DS in the snapshot is honestly "we couldn't inspect" rather than
	// failure. Treat as HEALTHY with explanatory detail rather than a false
	// positive PROBE_FAILED.
	if r.Component.Status != "HEALTHY" {
		t.Errorf("empty DS list should be HEALTHY, got %q (detail=%q)",
			r.Component.Status, r.Component.Detail)
	}
}
