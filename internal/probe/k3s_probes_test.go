// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package probe

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

// ─── K3sDatastore ──────────────────────────────────────────────────────────────

// nodesNonK3s contains a node whose providerID does not start with "k3s://".
const nodesNonK3s = `{
  "apiVersion": "v1", "kind": "NodeList",
  "items": [
    {"apiVersion": "v1", "kind": "Node",
     "metadata": {"name": "node-1"},
     "spec": {"providerID": "aws:///us-east-1a/i-abc123"}}
  ]
}`

// nodesK3sSQLite contains a k3s node (providerID k3s://) — SQLite mode
// because no etcd pods are supplied.
const nodesK3sSQLite = `{
  "apiVersion": "v1", "kind": "NodeList",
  "items": [
    {"apiVersion": "v1", "kind": "Node",
     "metadata": {"name": "k3s-node-1"},
     "spec": {"providerID": "k3s://k3s-node-1"}}
  ]
}`

// nodesK3sAnnotation uses k3s.io/* annotation instead of providerID prefix.
const nodesK3sAnnotation = `{
  "apiVersion": "v1", "kind": "NodeList",
  "items": [
    {"apiVersion": "v1", "kind": "Node",
     "metadata": {
       "name": "k3s-node-2",
       "annotations": {"k3s.io/node-args": "server"}
     }}
  ]
}`

// etcdPodsHealthy contains two k3s etcd static pods in kube-system, both Ready.
const etcdPodsHealthy = `{
  "apiVersion": "v1", "kind": "PodList",
  "items": [
    {"apiVersion": "v1", "kind": "Pod",
     "metadata": {"name": "etcd-k3s-node-1", "namespace": "kube-system",
                  "labels": {"component": "etcd"}},
     "status": {
       "conditions": [{"type": "Ready", "status": "True"}],
       "containerStatuses": [{"restartCount": 0}]
     }},
    {"apiVersion": "v1", "kind": "Pod",
     "metadata": {"name": "etcd-k3s-node-2", "namespace": "kube-system",
                  "labels": {"component": "etcd"}},
     "status": {
       "conditions": [{"type": "Ready", "status": "True"}],
       "containerStatuses": [{"restartCount": 0}]
     }}
  ]
}`

// etcdPodDown contains one k3s etcd pod that is Not Ready.
const etcdPodDown = `{
  "apiVersion": "v1", "kind": "PodList",
  "items": [
    {"apiVersion": "v1", "kind": "Pod",
     "metadata": {"name": "etcd-k3s-node-1", "namespace": "kube-system",
                  "labels": {"component": "etcd"}},
     "status": {
       "conditions": [{"type": "Ready", "status": "False"}],
       "containerStatuses": [{"restartCount": 0}]
     }}
  ]
}`

// snapshotCMRecent returns a ConfigMap list containing a k3s-etcd-snapshot
// ConfigMap created 1 hour ago (well within the 26h SLA).
func snapshotCMRecent() string {
	ts := time.Now().Add(-1 * time.Hour).UTC().Format(time.RFC3339)
	return `{
  "apiVersion": "v1", "kind": "ConfigMapList",
  "items": [
    {"apiVersion": "v1", "kind": "ConfigMap",
     "metadata": {"name": "k3s-etcd-snapshot-12345", "namespace": "kube-system",
                  "creationTimestamp": "` + ts + `"}}
  ]
}`
}

// noSnapshotCMs has no k3s-etcd-snapshot ConfigMaps.
const noSnapshotCMs = `{
  "apiVersion": "v1", "kind": "ConfigMapList",
  "items": []
}`

// TestK3sDatastore_NoK3sNodes — probe skips on non-k3s clusters.
func TestK3sDatastore_NoK3sNodes(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{
		"nodes.json": nodesNonK3s,
		"pods.json":  `{"apiVersion":"v1","kind":"PodList","items":[]}`,
	})
	r := K3sDatastore{}.Run(context.Background(), src)
	if r.Component.Status != "SKIPPED" {
		t.Errorf("non-k3s cluster: want SKIPPED, got %q (detail=%q)", r.Component.Status, r.Component.Detail)
	}
}

// TestK3sDatastore_SQLiteMode — k3s node + no etcd pods → SQLite mode → HEALTHY
// with Info advisory about single-node lack of HA.
func TestK3sDatastore_SQLiteMode(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{
		"nodes.json": nodesK3sSQLite,
		"pods.json":  `{"apiVersion":"v1","kind":"PodList","items":[]}`,
		"cms.json":   noSnapshotCMs,
	})
	r := K3sDatastore{}.Run(context.Background(), src)
	// Info findings roll up to HEALTHY in the rollup function.
	if r.Component.Status != "HEALTHY" {
		t.Errorf("SQLite mode: want HEALTHY (info finding), got %q", r.Component.Status)
	}
	// Must have at least one Info finding about SQLite.
	if len(r.Findings) == 0 {
		t.Fatalf("SQLite mode: expected at least one Info finding, got none")
	}
	found := false
	for _, f := range r.Findings {
		if f.Severity == SeverityInfo && strings.Contains(strings.ToLower(f.Message), "sqlite") {
			found = true
		}
	}
	if !found {
		t.Errorf("SQLite mode finding should mention SQLite: %+v", r.Findings)
	}
}

// TestK3sDatastore_EtcdHealthy — k3s nodes + healthy etcd pods + fresh snapshot → HEALTHY.
func TestK3sDatastore_EtcdHealthy(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{
		"nodes.json": nodesK3sSQLite,
		"pods.json":  etcdPodsHealthy,
		"cms.json":   snapshotCMRecent(),
	})
	r := K3sDatastore{}.Run(context.Background(), src)
	if r.Component.Status != "HEALTHY" {
		t.Errorf("healthy k3s etcd: want HEALTHY, got %q (detail=%q, findings=%+v)",
			r.Component.Status, r.Component.Detail, r.Findings)
	}
	if len(r.Findings) != 0 {
		t.Errorf("healthy etcd: expected no findings, got %+v", r.Findings)
	}
}

// TestK3sDatastore_EtcdDown — Not-Ready etcd pod → CRITICAL finding.
func TestK3sDatastore_EtcdDown(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{
		"nodes.json": nodesK3sSQLite,
		"pods.json":  etcdPodDown,
		"cms.json":   snapshotCMRecent(),
	})
	r := K3sDatastore{}.Run(context.Background(), src)
	if r.Component.Status != "CRITICAL" {
		t.Errorf("down etcd: want CRITICAL, got %q (detail=%q)", r.Component.Status, r.Component.Detail)
	}
	found := false
	for _, f := range r.Findings {
		if f.Severity == SeverityCritical && strings.Contains(f.Message, "etcd-k3s-node-1") {
			found = true
		}
	}
	if !found {
		t.Errorf("CRITICAL finding should name the down pod: %+v", r.Findings)
	}
}

// TestK3sDatastore_AnnotationDetection — k3s.io/* annotation (no providerID) must
// trigger cluster detection (should NOT be SKIPPED).
func TestK3sDatastore_AnnotationDetection(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{
		"nodes.json": nodesK3sAnnotation,
		"pods.json":  `{"apiVersion":"v1","kind":"PodList","items":[]}`,
		"cms.json":   noSnapshotCMs,
	})
	r := K3sDatastore{}.Run(context.Background(), src)
	if r.Component.Status == "SKIPPED" {
		t.Errorf("k3s.io/* annotation should trigger detection; got SKIPPED")
	}
}

// TestK3sDatastore_EtcdStaleSnapshot — healthy etcd pods but snapshot ConfigMap
// older than the SLA → WARNING.
func TestK3sDatastore_EtcdStaleSnapshot(t *testing.T) {
	staleTS := time.Now().Add(-48 * time.Hour).UTC().Format(time.RFC3339)
	staleCMs := `{
  "apiVersion": "v1", "kind": "ConfigMapList",
  "items": [
    {"apiVersion": "v1", "kind": "ConfigMap",
     "metadata": {"name": "k3s-etcd-snapshot-old", "namespace": "kube-system",
                  "creationTimestamp": "` + staleTS + `"}}
  ]
}`
	src := loadProbeSrc(t, map[string]string{
		"nodes.json": nodesK3sSQLite,
		"pods.json":  etcdPodsHealthy,
		"cms.json":   staleCMs,
	})
	r := K3sDatastore{}.Run(context.Background(), src)
	if r.Component.Status != "DEGRADED" {
		t.Errorf("stale snapshot: want DEGRADED, got %q (detail=%q, findings=%+v)",
			r.Component.Status, r.Component.Detail, r.Findings)
	}
	found := false
	for _, f := range r.Findings {
		if f.Severity == SeverityWarning && strings.Contains(f.Message, "k3s-etcd-snapshot-old") {
			found = true
		}
	}
	if !found {
		t.Errorf("stale-snapshot warning should name the ConfigMap: %+v", r.Findings)
	}
}

// ─── K3sDatastore: clustered etcd (multi-member) checks ───────────────────────

// nodes3EtcdMembers — three control-plane nodes, each labelled
// node-role.kubernetes.io/etcd, all with a recent snapshot timestamp.
func nodes3EtcdMembersFresh() string {
	tsRecent := time.Now().Add(-30 * time.Minute).UTC().Format("20060102150405")
	return fmt.Sprintf(`{
  "apiVersion": "v1", "kind": "NodeList",
  "items": [
    {"apiVersion": "v1", "kind": "Node",
     "metadata": {"name": "cp1", "labels": {"node-role.kubernetes.io/etcd": "true"},
                  "annotations": {"etcd.k3s.cattle.io/local-snapshots-timestamp": "%s"}},
     "spec": {"providerID": "k3s://cp1"}},
    {"apiVersion": "v1", "kind": "Node",
     "metadata": {"name": "cp2", "labels": {"node-role.kubernetes.io/etcd": "true"},
                  "annotations": {"etcd.k3s.cattle.io/local-snapshots-timestamp": "%s"}},
     "spec": {"providerID": "k3s://cp2"}},
    {"apiVersion": "v1", "kind": "Node",
     "metadata": {"name": "cp3", "labels": {"node-role.kubernetes.io/etcd": "true"},
                  "annotations": {"etcd.k3s.cattle.io/local-snapshots-timestamp": "%s"}},
     "spec": {"providerID": "k3s://cp3"}}
  ]
}`, tsRecent, tsRecent, tsRecent)
}

// etcd3PodsHealthy — three Ready etcd pods (matches nodes3EtcdMembers).
const etcd3PodsHealthy = `{
  "apiVersion": "v1", "kind": "PodList",
  "items": [
    {"apiVersion": "v1", "kind": "Pod",
     "metadata": {"name": "etcd-cp1", "namespace": "kube-system",
                  "labels": {"component": "etcd"}},
     "status": {"conditions": [{"type": "Ready", "status": "True"}],
                "containerStatuses": [{"restartCount": 0}]}},
    {"apiVersion": "v1", "kind": "Pod",
     "metadata": {"name": "etcd-cp2", "namespace": "kube-system",
                  "labels": {"component": "etcd"}},
     "status": {"conditions": [{"type": "Ready", "status": "True"}],
                "containerStatuses": [{"restartCount": 0}]}},
    {"apiVersion": "v1", "kind": "Pod",
     "metadata": {"name": "etcd-cp3", "namespace": "kube-system",
                  "labels": {"component": "etcd"}},
     "status": {"conditions": [{"type": "Ready", "status": "True"}],
                "containerStatuses": [{"restartCount": 0}]}}
  ]
}`

// TestK3sDatastore_HA3MembersHealthy — full HA configuration, all signals OK.
func TestK3sDatastore_HA3MembersHealthy(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{
		"nodes.json": nodes3EtcdMembersFresh(),
		"pods.json":  etcd3PodsHealthy,
		"cms.json":   snapshotCMRecent(),
	})
	r := K3sDatastore{}.Run(context.Background(), src)
	if r.Component.Status != "HEALTHY" {
		t.Errorf("HA healthy: want HEALTHY, got %q (findings=%+v)", r.Component.Status, r.Findings)
	}
}

// TestK3sDatastore_QuorumAtRisk — 3 declared etcd nodes but only 1 pod Ready (two
// not Ready) → CRITICAL finding mentioning quorum.
func TestK3sDatastore_QuorumAtRisk(t *testing.T) {
	tsRecent := time.Now().Add(-30 * time.Minute).UTC().Format("20060102150405")
	nodes3 := fmt.Sprintf(`{
  "apiVersion": "v1", "kind": "NodeList",
  "items": [
    {"apiVersion": "v1", "kind": "Node",
     "metadata": {"name": "cp1", "labels": {"node-role.kubernetes.io/etcd": "true"},
                  "annotations": {"etcd.k3s.cattle.io/local-snapshots-timestamp": "%s"}},
     "spec": {"providerID": "k3s://cp1"}},
    {"apiVersion": "v1", "kind": "Node",
     "metadata": {"name": "cp2", "labels": {"node-role.kubernetes.io/etcd": "true"},
                  "annotations": {"etcd.k3s.cattle.io/local-snapshots-timestamp": "%s"}},
     "spec": {"providerID": "k3s://cp2"}},
    {"apiVersion": "v1", "kind": "Node",
     "metadata": {"name": "cp3", "labels": {"node-role.kubernetes.io/etcd": "true"},
                  "annotations": {"etcd.k3s.cattle.io/local-snapshots-timestamp": "%s"}},
     "spec": {"providerID": "k3s://cp3"}}
  ]
}`, tsRecent, tsRecent, tsRecent)
	pods := `{
  "apiVersion": "v1", "kind": "PodList",
  "items": [
    {"apiVersion": "v1", "kind": "Pod",
     "metadata": {"name": "etcd-cp1", "namespace": "kube-system",
                  "labels": {"component": "etcd"}},
     "status": {"conditions": [{"type": "Ready", "status": "True"}],
                "containerStatuses": [{"restartCount": 0}]}},
    {"apiVersion": "v1", "kind": "Pod",
     "metadata": {"name": "etcd-cp2", "namespace": "kube-system",
                  "labels": {"component": "etcd"}},
     "status": {"conditions": [{"type": "Ready", "status": "False"}],
                "containerStatuses": [{"restartCount": 0}]}},
    {"apiVersion": "v1", "kind": "Pod",
     "metadata": {"name": "etcd-cp3", "namespace": "kube-system",
                  "labels": {"component": "etcd"}},
     "status": {"conditions": [{"type": "Ready", "status": "False"}],
                "containerStatuses": [{"restartCount": 0}]}}
  ]
}`
	src := loadProbeSrc(t, map[string]string{
		"nodes.json": nodes3,
		"pods.json":  pods,
		"cms.json":   snapshotCMRecent(),
	})
	r := K3sDatastore{}.Run(context.Background(), src)
	if r.Component.Status != "CRITICAL" {
		t.Errorf("quorum loss: want CRITICAL, got %q (findings=%+v)", r.Component.Status, r.Findings)
	}
	found := false
	for _, f := range r.Findings {
		if f.Severity == SeverityCritical && strings.Contains(strings.ToLower(f.Message), "quorum") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a CRITICAL quorum-loss finding; got %+v", r.Findings)
	}
}

// TestK3sDatastore_TwoMemberWarning — declared==2 is a known anti-pattern
// (no fault tolerance) — emit WARNING regardless of pod state.
func TestK3sDatastore_TwoMemberWarning(t *testing.T) {
	tsRecent := time.Now().Add(-30 * time.Minute).UTC().Format("20060102150405")
	nodes2 := fmt.Sprintf(`{
  "apiVersion": "v1", "kind": "NodeList",
  "items": [
    {"apiVersion": "v1", "kind": "Node",
     "metadata": {"name": "cp1", "labels": {"node-role.kubernetes.io/etcd": "true"},
                  "annotations": {"etcd.k3s.cattle.io/local-snapshots-timestamp": "%s"}},
     "spec": {"providerID": "k3s://cp1"}},
    {"apiVersion": "v1", "kind": "Node",
     "metadata": {"name": "cp2", "labels": {"node-role.kubernetes.io/etcd": "true"},
                  "annotations": {"etcd.k3s.cattle.io/local-snapshots-timestamp": "%s"}},
     "spec": {"providerID": "k3s://cp2"}}
  ]
}`, tsRecent, tsRecent)
	pods2 := `{
  "apiVersion": "v1", "kind": "PodList",
  "items": [
    {"apiVersion": "v1", "kind": "Pod",
     "metadata": {"name": "etcd-cp1", "namespace": "kube-system",
                  "labels": {"component": "etcd"}},
     "status": {"conditions": [{"type": "Ready", "status": "True"}],
                "containerStatuses": [{"restartCount": 0}]}},
    {"apiVersion": "v1", "kind": "Pod",
     "metadata": {"name": "etcd-cp2", "namespace": "kube-system",
                  "labels": {"component": "etcd"}},
     "status": {"conditions": [{"type": "Ready", "status": "True"}],
                "containerStatuses": [{"restartCount": 0}]}}
  ]
}`
	src := loadProbeSrc(t, map[string]string{
		"nodes.json": nodes2,
		"pods.json":  pods2,
		"cms.json":   snapshotCMRecent(),
	})
	r := K3sDatastore{}.Run(context.Background(), src)
	found := false
	for _, f := range r.Findings {
		if f.Severity == SeverityWarning && strings.Contains(strings.ToLower(f.Message), "2 voting members") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 2-member fault-tolerance warning; got %+v", r.Findings)
	}
}

// TestK3sDatastore_MemberMissing — 3 etcd-labelled nodes but only 2 pods → CRITICAL.
func TestK3sDatastore_MemberMissing(t *testing.T) {
	tsRecent := time.Now().Add(-30 * time.Minute).UTC().Format("20060102150405")
	nodes3 := fmt.Sprintf(`{
  "apiVersion": "v1", "kind": "NodeList",
  "items": [
    {"apiVersion": "v1", "kind": "Node",
     "metadata": {"name": "cp1", "labels": {"node-role.kubernetes.io/etcd": "true"},
                  "annotations": {"etcd.k3s.cattle.io/local-snapshots-timestamp": "%s"}},
     "spec": {"providerID": "k3s://cp1"}},
    {"apiVersion": "v1", "kind": "Node",
     "metadata": {"name": "cp2", "labels": {"node-role.kubernetes.io/etcd": "true"},
                  "annotations": {"etcd.k3s.cattle.io/local-snapshots-timestamp": "%s"}},
     "spec": {"providerID": "k3s://cp2"}},
    {"apiVersion": "v1", "kind": "Node",
     "metadata": {"name": "cp3", "labels": {"node-role.kubernetes.io/etcd": "true"},
                  "annotations": {"etcd.k3s.cattle.io/local-snapshots-timestamp": "%s"}},
     "spec": {"providerID": "k3s://cp3"}}
  ]
}`, tsRecent, tsRecent, tsRecent)
	pods2 := `{
  "apiVersion": "v1", "kind": "PodList",
  "items": [
    {"apiVersion": "v1", "kind": "Pod",
     "metadata": {"name": "etcd-cp1", "namespace": "kube-system",
                  "labels": {"component": "etcd"}},
     "status": {"conditions": [{"type": "Ready", "status": "True"}],
                "containerStatuses": [{"restartCount": 0}]}},
    {"apiVersion": "v1", "kind": "Pod",
     "metadata": {"name": "etcd-cp2", "namespace": "kube-system",
                  "labels": {"component": "etcd"}},
     "status": {"conditions": [{"type": "Ready", "status": "True"}],
                "containerStatuses": [{"restartCount": 0}]}}
  ]
}`
	src := loadProbeSrc(t, map[string]string{
		"nodes.json": nodes3,
		"pods.json":  pods2,
		"cms.json":   snapshotCMRecent(),
	})
	r := K3sDatastore{}.Run(context.Background(), src)
	found := false
	for _, f := range r.Findings {
		if f.Severity == SeverityCritical && strings.Contains(strings.ToLower(f.Message), "missing member") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected missing-member CRITICAL finding; got %+v", r.Findings)
	}
}

// TestK3sDatastore_MemberSnapshotStale — 3 healthy members but one has a
// local-snapshot annotation that lags >SLA behind the newest → WARNING.
func TestK3sDatastore_MemberSnapshotStale(t *testing.T) {
	newest := time.Now().Add(-30 * time.Minute).UTC().Format("20060102150405")
	stale := time.Now().Add(-72 * time.Hour).UTC().Format("20060102150405")
	nodes3 := fmt.Sprintf(`{
  "apiVersion": "v1", "kind": "NodeList",
  "items": [
    {"apiVersion": "v1", "kind": "Node",
     "metadata": {"name": "cp1", "labels": {"node-role.kubernetes.io/etcd": "true"},
                  "annotations": {"etcd.k3s.cattle.io/local-snapshots-timestamp": "%s"}},
     "spec": {"providerID": "k3s://cp1"}},
    {"apiVersion": "v1", "kind": "Node",
     "metadata": {"name": "cp2", "labels": {"node-role.kubernetes.io/etcd": "true"},
                  "annotations": {"etcd.k3s.cattle.io/local-snapshots-timestamp": "%s"}},
     "spec": {"providerID": "k3s://cp2"}},
    {"apiVersion": "v1", "kind": "Node",
     "metadata": {"name": "cp3", "labels": {"node-role.kubernetes.io/etcd": "true"},
                  "annotations": {"etcd.k3s.cattle.io/local-snapshots-timestamp": "%s"}},
     "spec": {"providerID": "k3s://cp3"}}
  ]
}`, newest, newest, stale)
	src := loadProbeSrc(t, map[string]string{
		"nodes.json": nodes3,
		"pods.json":  etcd3PodsHealthy,
		"cms.json":   snapshotCMRecent(),
	})
	r := K3sDatastore{}.Run(context.Background(), src)
	found := false
	for _, f := range r.Findings {
		if f.Severity == SeverityWarning &&
			strings.Contains(f.Message, "cp3") &&
			strings.Contains(strings.ToLower(f.Message), "lag behind") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected per-member snapshot-lag warning naming cp3; got %+v", r.Findings)
	}
}

// TestK3sDatastore_MemberNeverSnapshotted — etcd-labelled node with no
// local-snapshot annotation → WARNING.
func TestK3sDatastore_MemberNeverSnapshotted(t *testing.T) {
	tsRecent := time.Now().Add(-30 * time.Minute).UTC().Format("20060102150405")
	nodes3 := fmt.Sprintf(`{
  "apiVersion": "v1", "kind": "NodeList",
  "items": [
    {"apiVersion": "v1", "kind": "Node",
     "metadata": {"name": "cp1", "labels": {"node-role.kubernetes.io/etcd": "true"},
                  "annotations": {"etcd.k3s.cattle.io/local-snapshots-timestamp": "%s"}},
     "spec": {"providerID": "k3s://cp1"}},
    {"apiVersion": "v1", "kind": "Node",
     "metadata": {"name": "cp2", "labels": {"node-role.kubernetes.io/etcd": "true"},
                  "annotations": {"etcd.k3s.cattle.io/local-snapshots-timestamp": "%s"}},
     "spec": {"providerID": "k3s://cp2"}},
    {"apiVersion": "v1", "kind": "Node",
     "metadata": {"name": "cp3", "labels": {"node-role.kubernetes.io/etcd": "true"}},
     "spec": {"providerID": "k3s://cp3"}}
  ]
}`, tsRecent, tsRecent)
	src := loadProbeSrc(t, map[string]string{
		"nodes.json": nodes3,
		"pods.json":  etcd3PodsHealthy,
		"cms.json":   snapshotCMRecent(),
	})
	r := K3sDatastore{}.Run(context.Background(), src)
	found := false
	for _, f := range r.Findings {
		if f.Severity == SeverityWarning &&
			strings.Contains(f.Message, "cp3") &&
			strings.Contains(strings.ToLower(f.Message), "never wrote") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected never-snapshotted warning naming cp3; got %+v", r.Findings)
	}
}

// ─── K3sLocalPathStorage ───────────────────────────────────────────────────────

// pvcsNoLocalPath has PVCs but none use the local-path storage class.
const pvcsNoLocalPath = `{
  "apiVersion": "v1", "kind": "PersistentVolumeClaimList",
  "items": [
    {"apiVersion": "v1", "kind": "PersistentVolumeClaim",
     "metadata": {"name": "data", "namespace": "app"},
     "spec": {"storageClassName": "ceph-rbd"},
     "status": {"phase": "Bound"}}
  ]
}`

// pvcsLocalPathBound has a Bound local-path PVC.
const pvcsLocalPathBound = `{
  "apiVersion": "v1", "kind": "PersistentVolumeClaimList",
  "items": [
    {"apiVersion": "v1", "kind": "PersistentVolumeClaim",
     "metadata": {"name": "data", "namespace": "app"},
     "spec": {"storageClassName": "local-path"},
     "status": {"phase": "Bound"}}
  ]
}`

// pvcsLocalPathPending has a Pending local-path PVC.
const pvcsLocalPathPending = `{
  "apiVersion": "v1", "kind": "PersistentVolumeClaimList",
  "items": [
    {"apiVersion": "v1", "kind": "PersistentVolumeClaim",
     "metadata": {"name": "stuck-pvc", "namespace": "app"},
     "spec": {"storageClassName": "local-path"},
     "status": {"phase": "Pending"}}
  ]
}`

// nodesWithEphemeralStorageFraction returns a NodeList JSON with a node that has
// allocFraction of its 100Gi ephemeral-storage reported as allocatable.
// Quantities are expressed as plain byte integers so resource.ParseQuantity can
// always parse them (avoids the "GiB" vs "Gi" suffix mismatch).
func nodesWithEphemeralStorageFraction(allocFraction float64) string {
	const capGiB = 100
	const gib = int64(1) << 30
	cap := int64(capGiB) * gib
	alloc := int64(float64(cap) * allocFraction)
	capStr := fmt.Sprintf("%d", cap)
	allocStr := fmt.Sprintf("%d", alloc)
	return `{
  "apiVersion": "v1", "kind": "NodeList",
  "items": [
    {"apiVersion": "v1", "kind": "Node",
     "metadata": {"name": "k3s-node"},
     "status": {
       "capacity":    {"ephemeral-storage": "` + capStr + `"},
       "allocatable": {"ephemeral-storage": "` + allocStr + `"}
     }}
  ]
}`
}

// TestK3sLocalPathStorage_NoPendingPVCs — no local-path PVCs → HEALTHY no-op.
func TestK3sLocalPathStorage_NoPendingPVCs(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{
		"pvcs.json": pvcsNoLocalPath,
	})
	r := K3sLocalPathStorage{}.Run(context.Background(), src)
	if r.Component.Status != "HEALTHY" {
		t.Errorf("no local-path PVCs: want HEALTHY, got %q (detail=%q)", r.Component.Status, r.Component.Detail)
	}
	if len(r.Findings) != 0 {
		t.Errorf("no local-path PVCs: expected no findings, got %+v", r.Findings)
	}
}

// TestK3sLocalPathStorage_PendingPVC — Pending local-path PVC → WARNING finding.
func TestK3sLocalPathStorage_PendingPVC(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{
		"pvcs.json":  pvcsLocalPathPending,
		"nodes.json": nodesWithEphemeralStorageFraction(0.50),
	})
	r := K3sLocalPathStorage{}.Run(context.Background(), src)
	if r.Component.Status != "DEGRADED" && r.Component.Status != "CRITICAL" {
		t.Errorf("pending PVC: want DEGRADED or CRITICAL, got %q", r.Component.Status)
	}
	found := false
	for _, f := range r.Findings {
		if f.Severity == SeverityWarning && strings.Contains(f.Message, "stuck-pvc") {
			found = true
		}
	}
	if !found {
		t.Errorf("pending PVC finding should name stuck-pvc: %+v", r.Findings)
	}
}

// TestK3sLocalPathStorage_DiskWarning — local-path PVC bound but node disk at 15%
// free (below 20% default threshold) → WARNING.
func TestK3sLocalPathStorage_DiskWarning(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{
		"pvcs.json":  pvcsLocalPathBound,
		"nodes.json": nodesWithEphemeralStorageFraction(0.15),
	})
	r := K3sLocalPathStorage{}.Run(context.Background(), src)
	if r.Component.Status != "DEGRADED" && r.Component.Status != "CRITICAL" {
		t.Errorf("low disk: want DEGRADED or CRITICAL, got %q (findings=%+v)", r.Component.Status, r.Findings)
	}
}

// TestK3sLocalPathStorage_DiskCritical — node disk at 8% free (below 10% critical
// threshold) → CRITICAL.
func TestK3sLocalPathStorage_DiskCritical(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{
		"pvcs.json":  pvcsLocalPathBound,
		"nodes.json": nodesWithEphemeralStorageFraction(0.08),
	})
	r := K3sLocalPathStorage{}.Run(context.Background(), src)
	if r.Component.Status != "CRITICAL" {
		t.Errorf("critical disk: want CRITICAL, got %q (findings=%+v)", r.Component.Status, r.Findings)
	}
}

// TestK3sLocalPathStorage_NodeNoEphemeralMetrics — nodes without ephemeral-storage
// allocatable → Info finding (visibility gap).
func TestK3sLocalPathStorage_NodeNoEphemeralMetrics(t *testing.T) {
	const nodeNoEphemeral = `{
  "apiVersion": "v1", "kind": "NodeList",
  "items": [
    {"apiVersion": "v1", "kind": "Node",
     "metadata": {"name": "k3s-node"},
     "status": {}}
  ]
}`
	src := loadProbeSrc(t, map[string]string{
		"pvcs.json":  pvcsLocalPathBound,
		"nodes.json": nodeNoEphemeral,
	})
	r := K3sLocalPathStorage{}.Run(context.Background(), src)
	found := false
	for _, f := range r.Findings {
		if f.Severity == SeverityInfo {
			found = true
		}
	}
	if !found {
		t.Errorf("missing ephemeral-storage metrics should emit Info finding: status=%q findings=%+v",
			r.Component.Status, r.Findings)
	}
}

// ─── TraefikRoutes ─────────────────────────────────────────────────────────────

// traefikIRHealthy is a valid IngressRoute whose backend Service exists.
// Uses PathPrefix match to avoid backtick-in-raw-string-literal issues.
const traefikIRHealthy = `{
  "apiVersion": "traefik.io/v1alpha1", "kind": "IngressRouteList",
  "items": [
    {"apiVersion": "traefik.io/v1alpha1", "kind": "IngressRoute",
     "metadata": {"name": "myapp", "namespace": "app"},
     "spec": {"routes": [
       {"match": "PathPrefix(\"/\")",
        "kind": "Rule",
        "services": [{"name": "myapp-svc", "port": 80}]}
     ]}}
  ]
}`

// traefikIROrphanService has an IngressRoute pointing to a Service that doesn't exist.
const traefikIROrphanService = `{
  "apiVersion": "traefik.io/v1alpha1", "kind": "IngressRouteList",
  "items": [
    {"apiVersion": "traefik.io/v1alpha1", "kind": "IngressRoute",
     "metadata": {"name": "broken", "namespace": "app"},
     "spec": {"routes": [
       {"match": "PathPrefix(\"/broken\")",
        "kind": "Rule",
        "services": [{"name": "missing-svc", "port": 80}]}
     ]}}
  ]
}`

// svcMyapp is a Service named myapp-svc in namespace app.
const svcMyapp = `{
  "apiVersion": "v1", "kind": "ServiceList",
  "items": [
    {"apiVersion": "v1", "kind": "Service",
     "metadata": {"name": "myapp-svc", "namespace": "app"}}
  ]
}`

// emptySvcList has no Services.
const emptySvcList = `{
  "apiVersion": "v1", "kind": "ServiceList",
  "items": []
}`

// TestTraefikRoutes_NoCRD — errSource (all lists error) → SKIPPED.
func TestTraefikRoutes_NoCRD(t *testing.T) {
	r := TraefikRoutes{}.Run(context.Background(), errSource{})
	if r.Component.Status != "SKIPPED" {
		t.Errorf("no Traefik CRD: want SKIPPED, got %q (detail=%q)", r.Component.Status, r.Component.Detail)
	}
}

// TestTraefikRoutes_OrphanService — IngressRoute references a missing Service
// → CRITICAL finding.
func TestTraefikRoutes_OrphanService(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{
		"ingressroutes.json": traefikIROrphanService,
		"services.json":      emptySvcList,
	})
	r := TraefikRoutes{}.Run(context.Background(), src)
	if r.Component.Status != "CRITICAL" {
		t.Errorf("orphan service: want CRITICAL, got %q (detail=%q, findings=%+v)",
			r.Component.Status, r.Component.Detail, r.Findings)
	}
	found := false
	for _, f := range r.Findings {
		if f.Severity == SeverityCritical && strings.Contains(f.Message, "missing-svc") {
			found = true
		}
	}
	if !found {
		t.Errorf("finding should name missing-svc: %+v", r.Findings)
	}
}

// TestTraefikRoutes_Healthy — IngressRoute → existing Service → HEALTHY.
func TestTraefikRoutes_Healthy(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{
		"ingressroutes.json": traefikIRHealthy,
		"services.json":      svcMyapp,
	})
	r := TraefikRoutes{}.Run(context.Background(), src)
	if r.Component.Status != "HEALTHY" {
		t.Errorf("healthy IngressRoute: want HEALTHY, got %q (detail=%q, findings=%+v)",
			r.Component.Status, r.Component.Detail, r.Findings)
	}
	if len(r.Findings) != 0 {
		t.Errorf("healthy IngressRoute: expected no findings, got %+v", r.Findings)
	}
}

// TestTraefikRoutes_MissingMiddleware — IngressRoute references a Middleware that
// doesn't exist → WARNING finding.
func TestTraefikRoutes_MissingMiddleware(t *testing.T) {
	const irMissingMW = `{
  "apiVersion": "traefik.io/v1alpha1", "kind": "IngressRouteList",
  "items": [
    {"apiVersion": "traefik.io/v1alpha1", "kind": "IngressRoute",
     "metadata": {"name": "guarded", "namespace": "app"},
     "spec": {"routes": [
       {"match": "PathPrefix(\"/guarded\")",
        "kind": "Rule",
        "services": [{"name": "myapp-svc", "port": 80}],
        "middlewares": [{"name": "rate-limit"}]}
     ]}}
  ]
}`
	src := loadProbeSrc(t, map[string]string{
		"ingressroutes.json": irMissingMW,
		"services.json":      svcMyapp,
		// No middlewares in snapshot.
	})
	r := TraefikRoutes{}.Run(context.Background(), src)
	if r.Component.Status != "DEGRADED" && r.Component.Status != "CRITICAL" {
		t.Errorf("missing middleware: want DEGRADED or CRITICAL, got %q", r.Component.Status)
	}
	found := false
	for _, f := range r.Findings {
		if f.Severity == SeverityWarning && strings.Contains(f.Message, "rate-limit") {
			found = true
		}
	}
	if !found {
		t.Errorf("warning finding should name rate-limit middleware: %+v", r.Findings)
	}
}

// TestTraefikRoutes_TLSNoCertResolver — TLS enabled but no certResolver, secretName,
// or TLSStore → WARNING.
func TestTraefikRoutes_TLSNoCertResolver(t *testing.T) {
	const irTLSNoCert = `{
  "apiVersion": "traefik.io/v1alpha1", "kind": "IngressRouteList",
  "items": [
    {"apiVersion": "traefik.io/v1alpha1", "kind": "IngressRoute",
     "metadata": {"name": "tls-route", "namespace": "app"},
     "spec": {
       "routes": [
         {"match": "PathPrefix(\"/secure\")",
          "kind": "Rule",
          "services": [{"name": "myapp-svc", "port": 443}]}
       ],
       "tls": {}
     }}
  ]
}`
	src := loadProbeSrc(t, map[string]string{
		"ingressroutes.json": irTLSNoCert,
		"services.json":      svcMyapp,
	})
	r := TraefikRoutes{}.Run(context.Background(), src)
	if r.Component.Status != "DEGRADED" && r.Component.Status != "CRITICAL" {
		t.Errorf("TLS no cert resolver: want DEGRADED or CRITICAL, got %q", r.Component.Status)
	}
	found := false
	for _, f := range r.Findings {
		if f.Severity == SeverityWarning && strings.Contains(f.Message, "TLS") {
			found = true
		}
	}
	if !found {
		t.Errorf("TLS-no-cert finding not emitted: %+v", r.Findings)
	}
}

// ─── Name stability ────────────────────────────────────────────────────────────

func TestK3sProbes_NamesStable(t *testing.T) {
	cases := map[string]string{
		K3sDatastore{}.Name():        "K3s Datastore",
		K3sLocalPathStorage{}.Name(): "K3s LocalPath Storage",
		TraefikRoutes{}.Name():       "Traefik Routes",
	}
	for got, want := range cases {
		if got != want {
			t.Errorf("Name()=%q want %q", got, want)
		}
	}
}
