// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package diagnose

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/srenix-ai/agentic-sre/internal/snapshot"
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
//
// Aggregation: orphaned PVs are not auto-fixable (deleting a PV is a
// destructive, per-volume human decision), and a large cluster can
// accumulate hundreds of them. Emitting one finding per PV floods the
// AI tier — every one is investigated and then refused by the proposer
// each cycle, drowning out actionable findings. So PVOrphan emits ONE
// aggregate finding listing all orphans: a single SRE ticket to triage
// the batch, not N churning approval requests.
type PVOrphan struct{}

// Name returns the analyzer's stable identifier.
func (PVOrphan) Name() string { return "PVOrphan" }

const pvOrphanGrace = 7 * 24 * time.Hour

// pvOrphanListCap bounds how many PV names are inlined in the aggregate
// message; the rest are summarized as "+N more" to keep the Slack/ticket
// body readable.
const pvOrphanListCap = 25

// Run walks every PersistentVolume and emits ONE aggregate warning for all
// orphans (Released past the grace window). Returns nil when there are none.
func (a PVOrphan) Run(ctx context.Context, src snapshot.Source) []Diagnostic {
	list, err := src.List(ctx, snapshot.GVRPV, "")
	if err != nil {
		logListFailure("persistentvolumes", err, false)
		return nil
	}
	now := time.Now()
	type orphan struct {
		name, sc, capacity, reclaim string
	}
	var orphans []orphan
	for i := range list.Items {
		pv := &list.Items[i]
		phase, _, _ := unstructured.NestedString(pv.Object, "status", "phase")
		if phase != "Released" {
			continue
		}
		// Grace anchor: K8s doesn't expose a phase-transition time, so we
		// fall back to creationTimestamp + grace window. A PV created within
		// 7 days that's already Released is fast-churn we skip until it ages.
		anchor := pv.GetCreationTimestamp().Time
		if anchor.IsZero() || now.Sub(anchor) < pvOrphanGrace {
			continue
		}
		sc, _, _ := unstructured.NestedString(pv.Object, "spec", "storageClassName")
		if sc == "" {
			sc = "<default>"
		}
		capacity, _, _ := unstructured.NestedString(pv.Object, "spec", "capacity", "storage")
		if capacity == "" {
			capacity = "<unknown>"
		}
		reclaim, _, _ := unstructured.NestedString(pv.Object, "spec", "persistentVolumeReclaimPolicy")
		orphans = append(orphans, orphan{pv.GetName(), sc, capacity, reclaim})
	}
	if len(orphans) == 0 {
		return nil
	}
	sort.Slice(orphans, func(i, j int) bool { return orphans[i].name < orphans[j].name })

	// Build a readable, capped list of "name (capacity, reclaimPolicy)".
	var lines []string
	for i, o := range orphans {
		if i >= pvOrphanListCap {
			lines = append(lines, fmt.Sprintf("…and %d more", len(orphans)-pvOrphanListCap))
			break
		}
		lines = append(lines, fmt.Sprintf("%s (%s, %s, reclaim=%s)", o.name, o.capacity, o.sc, o.reclaim))
	}

	return []Diagnostic{{
		Source:   "PVOrphan",
		Subject:  "PersistentVolume/cluster/orphaned-released",
		Severity: "warning",
		Message: fmt.Sprintf(
			"%d PersistentVolume(s) have been Released for >%s — their PVCs were deleted but the volumes remain and may still be billing on the underlying cloud disks.",
			len(orphans), pvOrphanGrace),
		Remediation: fmt.Sprintf(
			"Triage the orphaned volumes and delete those no longer needed:\n\n%s\n\n"+
				"  kubectl describe pv <name>   # confirm the data is disposable\n"+
				"  kubectl delete pv <name>\n\n"+
				"On `reclaimPolicy=Retain`, the underlying cloud disk persists after PV deletion — "+
				"clean it up via the provider's console too (EBS / PD / Disk).",
			strings.Join(lines, "\n")),
	}}
}
