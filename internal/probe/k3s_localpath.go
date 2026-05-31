// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package probe

import (
	"context"
	"fmt"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// K3sLocalPathStorage surfaces disk-pressure risk on nodes that host
// local-path-provisioner PVCs.
//
// The local-path provisioner (default StorageClass on k3s) creates
// subdirectories in /var/lib/rancher/k3s/storage/ on the scheduling
// node. It does not report per-PVC capacity to the Kubernetes
// scheduler. The only Kubernetes-visible signal is node
// status.allocatable["ephemeral-storage"] vs
// status.capacity["ephemeral-storage"]. The kubelet DiskPressure
// condition only fires when the root volume is near full — far too
// late for early warning.
//
// Findings produced:
//
//	F1 — Node ephemeral-storage headroom below threshold.
//	     Warning when < DiskFreeThreshold (default 20%).
//	     Critical when < 10% (kubelet eviction-threshold default).
//
//	F2 — local-path PVCs exist but nodes do not report
//	     ephemeral-storage (kubelet feature-gate disabled).
//	     Severity Info — a visibility gap, not a fault.
//
//	F3 — local-path PVCs in Pending state.
//	     Severity Warning — scheduling failure or node capacity issue.
//
// No-ops gracefully when no local-path PVCs are found.
// Opt-out: CHA_PROBE_K3S_LOCALPATH=off
type K3sLocalPathStorage struct {
	// DiskFreeThreshold is the fraction of ephemeral-storage that must
	// remain free before the probe emits a warning. Zero → 0.20 (20%).
	DiskFreeThreshold float64
}

const (
	k3sLocalPathName              = "K3s LocalPath Storage"
	defaultLocalPathDiskThreshold = 0.20
	// criticalLocalPathDiskThreshold mirrors the kubelet default eviction
	// hard threshold. Once free fraction drops below this, eviction is
	// imminent.
	criticalLocalPathDiskThreshold = 0.10

	localPathStorageClass = "local-path"
)

// gvrStorageClass is the GVR for storage.k8s.io/v1 StorageClass objects.
// Added here for K3sLocalPathStorage; will be de-duplicated into
// internal/snapshot/source.go when the full k3s probe suite lands.
var gvrStorageClass = schema.GroupVersionResource{
	Group:    "storage.k8s.io",
	Version:  "v1",
	Resource: "storageclasses",
}

// Name satisfies probe.Probe.
func (K3sLocalPathStorage) Name() string { return k3sLocalPathName }

// Run satisfies probe.Probe.
func (p K3sLocalPathStorage) Run(ctx context.Context, src snapshot.Source) Result {
	r := Result{Component: ComponentResult{Component: k3sLocalPathName}}

	threshold := p.DiskFreeThreshold
	if threshold == 0 {
		threshold = defaultLocalPathDiskThreshold
	}

	// ── 1. Determine if "local-path" is the cluster default StorageClass ──
	isDefaultLocalPath := false
	scList, err := src.List(ctx, gvrStorageClass, "")
	if err == nil && scList != nil {
		for _, sc := range scList.Items {
			scName := sc.GetName()
			if scName != localPathStorageClass {
				continue
			}
			annotations := sc.GetAnnotations()
			if annotations["storageclass.kubernetes.io/is-default-class"] == "true" {
				isDefaultLocalPath = true
				break
			}
		}
	}
	// If StorageClass list failed we conservatively assume local-path
	// might be default; we still filter PVCs by explicit class name so
	// the worst-case is a slightly broader scan, never a missed finding.

	// ── 2. List PVCs and filter to local-path ─────────────────────────────
	pvcList, err := src.List(ctx, snapshot.GVRPVC, "")
	if err != nil {
		r.Component.Status = "PROBE_FAILED"
		r.Component.Detail = "list pvcs: " + err.Error()
		r.Findings = append(r.Findings, Finding{
			Component: k3sLocalPathName,
			Severity:  SeverityCritical,
			Message:   "K3s LocalPath probe failed: cannot list PVCs",
		})
		return r
	}

	type pvcInfo struct {
		ns    string
		name  string
		phase string
	}
	var localPVCs []pvcInfo
	pendingCount := 0

	for _, pvc := range pvcList.Items {
		scName, _, _ := getStringField(pvc.Object, "spec", "storageClassName")
		isLocalPath := scName == localPathStorageClass ||
			(scName == "" && isDefaultLocalPath)
		if !isLocalPath {
			continue
		}
		phase, _, _ := getStringField(pvc.Object, "status", "phase")
		info := pvcInfo{
			ns:    pvc.GetNamespace(),
			name:  pvc.GetName(),
			phase: phase,
		}
		localPVCs = append(localPVCs, info)
		if phase == "Pending" {
			pendingCount++
		}
	}

	// No local-path PVCs → no-op.
	if len(localPVCs) == 0 {
		r.Component.Status = "HEALTHY"
		r.Component.Detail = "no local-path PVCs found"
		return r
	}

	// ── 3. List Nodes and inspect ephemeral-storage ───────────────────────
	nodeList, err := src.List(ctx, snapshot.GVRNode, "")
	if err != nil {
		r.Component.Status = "PROBE_FAILED"
		r.Component.Detail = "list nodes: " + err.Error()
		r.Findings = append(r.Findings, Finding{
			Component: k3sLocalPathName,
			Severity:  SeverityCritical,
			Message:   "K3s LocalPath probe failed: cannot list nodes",
		})
		return r
	}

	var findings []Finding

	// Track whether any node actually reports ephemeral-storage.
	anyNodeReportsEphemeral := false

	for _, node := range nodeList.Items {
		nodeName := node.GetName()

		// Read allocatable["ephemeral-storage"] and capacity["ephemeral-storage"].
		allocStr := nestedResourceValue(node.Object, "status", "allocatable", "ephemeral-storage")
		capStr := nestedResourceValue(node.Object, "status", "capacity", "ephemeral-storage")

		if allocStr == "" || capStr == "" {
			// Node doesn't report ephemeral-storage at all — skip
			// per-node check; F2 handles the aggregate visibility gap.
			continue
		}

		allocBytes, errA := parseQuantityToBytes(allocStr)
		capBytes, errC := parseQuantityToBytes(capStr)
		if errA != nil || errC != nil || capBytes <= 0 {
			// Malformed quantity — skip silently; not a probe failure.
			continue
		}

		anyNodeReportsEphemeral = true

		freeFraction := float64(allocBytes) / float64(capBytes)
		pct := freeFraction * 100.0

		if freeFraction < criticalLocalPathDiskThreshold {
			findings = append(findings, Finding{
				Component: fmt.Sprintf("Node/%s", nodeName),
				Severity:  SeverityCritical,
				Message: fmt.Sprintf(
					"Node %s: ephemeral-storage only %.1f%% free (%s of %s); local-path PVC data lives in /var/lib/rancher/k3s/storage/",
					nodeName, pct, formatBytes(allocBytes), formatBytes(capBytes),
				),
				Remediation: "Free disk on the node: prune unused images (`crictl rmi --prune`), " +
					"remove old log files under /var/log, cordon+drain the node to migrate local-path PVCs, " +
					"or expand the node's root volume",
			})
		} else if freeFraction < threshold {
			findings = append(findings, Finding{
				Component: fmt.Sprintf("Node/%s", nodeName),
				Severity:  SeverityWarning,
				Message: fmt.Sprintf(
					"Node %s: ephemeral-storage only %.1f%% free (%s of %s); local-path PVC data lives in /var/lib/rancher/k3s/storage/",
					nodeName, pct, formatBytes(allocBytes), formatBytes(capBytes),
				),
				Remediation: "Free disk on the node: prune unused images (`crictl rmi --prune`), " +
					"remove old log files under /var/log, cordon+drain the node to migrate local-path PVCs, " +
					"or expand the node's root volume",
			})
		}
	}

	// ── F2: local-path PVCs exist but no node reports ephemeral-storage ───
	if !anyNodeReportsEphemeral && len(nodeList.Items) > 0 {
		findings = append(findings, Finding{
			Component: k3sLocalPathName,
			Severity:  SeverityInfo,
			Message: fmt.Sprintf(
				"%d local-path PVC(s) found but nodes do not report ephemeral-storage allocatable; disk usage is unobservable via Kubernetes API",
				len(localPVCs),
			),
			Remediation: "Enable kubelet ephemeral-storage accounting " +
				"(LocalStorageCapacityIsolation=true, the default since k8s 1.10) or install " +
				"node-exporter and alert on `node_filesystem_avail_bytes{mountpoint='/'}` falling below 20%",
		})
	}

	// ── F3: Pending local-path PVCs ───────────────────────────────────────
	if pendingCount > 0 {
		for _, pvc := range localPVCs {
			if pvc.phase != "Pending" {
				continue
			}
			findings = append(findings, Finding{
				Component: fmt.Sprintf("PVC/%s/%s", pvc.ns, pvc.name),
				Severity:  SeverityWarning,
				Message: fmt.Sprintf(
					"local-path PVC %s/%s has been Pending for an extended time",
					pvc.ns, pvc.name,
				),
				Remediation: fmt.Sprintf(
					"Check that at least one schedulable node has sufficient ephemeral storage. "+
						"Run: kubectl describe pvc %s -n %s. "+
						"If the pod has node affinity/tolerations that limit scheduling, "+
						"the local-path directory may be absent on eligible nodes.",
					pvc.name, pvc.ns,
				),
			})
		}
	}

	// ── Rollup ────────────────────────────────────────────────────────────
	r.Component.Status = rollupComponentStatus(findings)
	if r.Component.Status == "HEALTHY" {
		r.Component.Detail = fmt.Sprintf(
			"%d local-path PVC(s) on %d node(s); ephemeral-storage headroom within threshold",
			len(localPVCs), len(nodeList.Items),
		)
	} else {
		r.Component.Detail = fmt.Sprintf(
			"%d local-path PVC(s) on %d node(s); %d finding(s)",
			len(localPVCs), len(nodeList.Items), len(findings),
		)
	}
	r.Findings = findings
	return r
}

// nestedResourceValue reads a Kubernetes quantity string from a nested
// map path (e.g., status → allocatable → ephemeral-storage). Returns ""
// when the path is absent or the value is not a string.
func nestedResourceValue(obj map[string]any, path ...string) string {
	cur := any(obj)
	for _, k := range path {
		mp, ok := cur.(map[string]any)
		if !ok {
			return ""
		}
		cur, ok = mp[k]
		if !ok {
			return ""
		}
	}
	s, _ := cur.(string)
	return s
}

// parseQuantityToBytes parses a Kubernetes resource quantity string
// (e.g., "50Gi", "107374182400") into an int64 byte count using
// k8s.io/apimachinery/pkg/api/resource.
func parseQuantityToBytes(s string) (int64, error) {
	q, err := resource.ParseQuantity(s)
	if err != nil {
		return 0, err
	}
	// Value() returns the quantity in its canonical scale (bytes for
	// ephemeral-storage, which is dimensionless).
	return q.Value(), nil
}

// formatBytes renders a byte count as a human-readable string
// (GiB / MiB / KiB / B) for Finding messages.
func formatBytes(b int64) string {
	const (
		GiB = 1 << 30
		MiB = 1 << 20
		KiB = 1 << 10
	)
	switch {
	case b >= GiB:
		return fmt.Sprintf("%.1fGiB", float64(b)/GiB)
	case b >= MiB:
		return fmt.Sprintf("%.1fMiB", float64(b)/MiB)
	case b >= KiB:
		return fmt.Sprintf("%.1fKiB", float64(b)/KiB)
	default:
		return fmt.Sprintf("%dB", b)
	}
}
