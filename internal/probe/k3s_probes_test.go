// Copyright 2026 Cluster Health Autopilot contributors
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
