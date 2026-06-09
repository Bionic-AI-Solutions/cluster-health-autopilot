// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package diagnose

import (
	"context"
	"fmt"
	"time"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// PVOrphan surfaces PersistentVolumes that have been in the
// `Released` phase for longer than the orphan grace window. A
// Released PV is one whose PVC was deleted but the PV itself
// stuck around (depending on the PV's reclaimPolicy: `Retain`
// → stays forever, `Delete` → eventually cleaned by the
// provisioner). On EBS / GCE-PD / Azure-Disk these orphans
// continue to BILL — operators need to know.
//
// Phase 3.E.2.
//
// Signal: status.phase == "Released" past pvOrphanGrace (7 days).
// Why 7 days — short-lived test workloads (CI, dev iterations)
// shouldn't fire; orphans that have been around a week are
// long-tail forgotten leftovers.
type PVOrphan struct{}

// Name returns the analyzer's stable identifier.
func (PVOrphan) Name() string { return "PVOrphan" }

const pvOrphanGrace = 7 * 24 * time.Hour

// Run walks every PersistentVolume and emits a warning per orphan.
func (a PVOrphan) Run(ctx context.Context, src snapshot.Source) []Diagnostic {
	list, err := src.List(ctx, snapshot.GVRPV, "")
	if err != nil {
		return nil
	}
	now := time.Now()
	var out []Diagnostic
	for i := range list.Items {
		pv := &list.Items[i]
		phase, _, _ := unstructured.NestedString(pv.Object, "status", "phase")
		if phase != "Released" {
			continue
		}
		// Grace anchor: the timestamp of the PV's transition to
		// Released. K8s doesn't expose a phase-transition time
		// directly, so we fall back to creationTimestamp + grace
		// window. A PV created within 7 days that's already
		// Released is a fast-churn anomaly we skip until it ages
		// past the window.
		anchor := pv.GetCreationTimestamp().Time
		if anchor.IsZero() || now.Sub(anchor) < pvOrphanGrace {
			continue
		}
		name := pv.GetName()
		// Surface storage class + capacity in the message so the
		// operator can size up the cost impact at a glance.
		sc, _, _ := unstructured.NestedString(pv.Object, "spec", "storageClassName")
		if sc == "" {
			sc = "<default>"
		}
		capacity, _, _ := unstructured.NestedString(pv.Object, "spec", "capacity", "storage")
		if capacity == "" {
			capacity = "<unknown>"
		}
		reclaim, _, _ := unstructured.NestedString(pv.Object, "spec", "persistentVolumeReclaimPolicy")
		out = append(out, Diagnostic{
			Source:   "PVOrphan",
			Subject:  "PersistentVolume/" + name,
			Severity: "warning",
			Message: fmt.Sprintf(
				"PersistentVolume %s has been Released for >%s (storageClass=%s, capacity=%s, reclaimPolicy=%s). The PVC was deleted; this volume may still be billing on the underlying cloud disk.",
				name, pvOrphanGrace, sc, capacity, reclaim),
			Remediation: fmt.Sprintf(
				"Inspect the PV + decide whether to delete or re-bind:\n\n"+
					"  kubectl describe pv %s\n\n"+
					"If the data is no longer needed:\n\n"+
					"  kubectl delete pv %s\n\n"+
					"On `reclaimPolicy=Retain`, the underlying cloud disk persists after PV deletion — clean it up via the provider's console too (EBS / PD / Disk).",
				name, name),
		})
	}
	return out
}
