// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package diagnose

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// DisruptionDrift surfaces three quota/disruption-tier signals that
// the basic resource-health probes miss. Each signal here is "this
// workload's disruption + capacity plumbing is broken in a way that
// silently caps it." Phase 2.E roadmap deliverable (paired sibling
// to CapacityDrift which covers HPA/PVC sizing).
//
//   - **PDB blocks all evictions** — `status.disruptionsAllowed == 0`
//     past a grace window. A PodDisruptionBudget set too tight (e.g.
//     `minAvailable: 100%` with 1 replica) means the kubelet can't
//     drain the node for maintenance. Critical (node-maintenance
//     fails; reboots hang).
//
//   - **Stuck Job pods (failed indexes)** — Job with
//     `spec.completionMode=Indexed` whose
//     `status.failedIndexes` is non-empty past grace. The Job is
//     wedged on the failed indexes; the operator probably wants
//     `kubectl create job --from=job/X` with the fixed manifest.
//     Warning (the controller still retries; the operator just
//     can't see the stuck-index list without `kubectl get job -o
//     wide`).
//
//   - **Stale ResourceQuota at 100%** — `status.used == spec.hard`
//     for at least one resource past grace. The namespace's quota
//     is saturated; net-new workloads in the namespace will be
//     refused by the API server with a confusing "exceeded quota"
//     error that doesn't name the saturated resource. Warning (no
//     immediate cluster-wide impact; new workloads in the namespace
//     fail-fast).
//
// Each analyzer is on by default but can be opted out via env var
// for clusters that don't use the targeted asset class:
//
//   - CHA_ANALYZER_DISRUPTION_DRIFT=off — disables the whole bundle
type DisruptionDrift struct{}

// Name returns the analyzer's stable identifier, used in
// Diagnostic.Source + audit logs.
func (DisruptionDrift) Name() string { return "DisruptionDrift" }

// gracePDB / graceQuota / graceJob mirror the CapacityDrift grace-
// window pattern: a brief saturation is "the autoscaler caught up";
// a sustained one is a real signal.
const (
	gracePDBBlocked       = 5 * time.Minute  // long enough to see one HPA tick
	graceQuotaAtMax       = 1 * time.Hour    // resource quotas saturate briefly during rollouts
	graceJobFailedIndexes = 10 * time.Minute // Indexed Job has its own retry backoff
)

// List-failure logging now lives in listlog.go (logListFailure) — the
// once-per-(resource, error-reason) helper introduced here in P1.2 was
// lifted package-wide in P1.7.

// Run executes the three sub-analyzers in order and returns the
// combined diagnostic slice. Always nil-error per the Analyzer
// interface contract.
func (a DisruptionDrift) Run(ctx context.Context, src snapshot.Source) []Diagnostic {
	var out []Diagnostic
	out = append(out, a.runPDB(ctx, src)...)
	out = append(out, a.runStuckJobs(ctx, src)...)
	out = append(out, a.runResourceQuota(ctx, src)...)
	return out
}

// runPDB walks every PDB + surfaces those whose
// status.disruptionsAllowed has been 0 for > gracePDBBlocked.
// We use status.conditions[type=DisruptionAllowed].lastTransitionTime
// as the grace anchor when present; otherwise fall back to
// metadata.creationTimestamp + the grace window (any PDB created
// within the grace window is not yet stable enough to fire).
func (a DisruptionDrift) runPDB(ctx context.Context, src snapshot.Source) []Diagnostic {
	gvr := schema.GroupVersionResource{Group: "policy", Version: "v1", Resource: "poddisruptionbudgets"}
	list, err := src.List(ctx, gvr, "")
	if err != nil {
		logListFailure("poddisruptionbudgets", err, false)
		return nil
	}
	now := time.Now()
	var out []Diagnostic
	for i := range list.Items {
		pdb := &list.Items[i]
		allowed, _, _ := unstructured.NestedInt64(pdb.Object, "status", "disruptionsAllowed")
		if allowed > 0 {
			continue
		}
		// Determine grace anchor — prefer the DisruptionAllowed
		// condition's lastTransitionTime (when the PDB flipped
		// from allowed to blocked); fall back to creationTimestamp.
		anchor := pdbBlockedAnchor(pdb)
		if anchor.IsZero() {
			continue
		}
		if now.Sub(anchor) < gracePDBBlocked {
			continue
		}
		ns := pdb.GetNamespace()
		name := pdb.GetName()
		minAvail, _, _ := unstructured.NestedString(pdb.Object, "spec", "minAvailable")
		maxUnavail, _, _ := unstructured.NestedString(pdb.Object, "spec", "maxUnavailable")
		shape := strings.TrimSpace(minAvail + maxUnavail)
		out = append(out, Diagnostic{
			Source:   "DisruptionDrift",
			Subject:  "PodDisruptionBudget/" + ns + "/" + name,
			Severity: "critical",
			Message: fmt.Sprintf(
				"PodDisruptionBudget %s/%s has blocked ALL evictions for >%s (minAvailable/maxUnavailable=%s). Drains will hang.",
				ns, name, gracePDBBlocked, shape),
			Remediation: fmt.Sprintf(
				"Relax the PDB so at least 1 pod can be evicted at a time:\n\n"+
					"  kubectl -n %s edit pdb %s\n\n"+
					"For a single-replica Deployment, consider whether the PDB belongs at all — a single pod with strict PDB cannot tolerate node maintenance.",
				ns, name),
		})
	}
	return out
}

// pdbBlockedAnchor returns the time the PDB transitioned to
// DisruptionAllowed=False, or its creation timestamp as fallback.
// Returns zero time if neither is available.
func pdbBlockedAnchor(pdb *unstructured.Unstructured) time.Time {
	conds, _, _ := unstructured.NestedSlice(pdb.Object, "status", "conditions")
	for _, c := range conds {
		cm, ok := c.(map[string]any)
		if !ok {
			continue
		}
		if cm["type"] == "DisruptionAllowed" && cm["status"] == "False" {
			if ltt, ok := cm["lastTransitionTime"].(string); ok && ltt != "" {
				if t, err := time.Parse(time.RFC3339, ltt); err == nil {
					return t
				}
			}
		}
	}
	return pdb.GetCreationTimestamp().Time
}

// runStuckJobs walks every Job + flags ones with failed indexes
// past graceJobFailedIndexes. Only Indexed Jobs are eligible —
// non-indexed Jobs handle failures via the controller's backoff
// and the operator's basic resource-health probes catch them.
func (a DisruptionDrift) runStuckJobs(ctx context.Context, src snapshot.Source) []Diagnostic {
	gvr := schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "jobs"}
	list, err := src.List(ctx, gvr, "")
	if err != nil {
		logListFailure("jobs", err, false)
		return nil
	}
	now := time.Now()
	var out []Diagnostic
	for i := range list.Items {
		job := &list.Items[i]
		mode, _, _ := unstructured.NestedString(job.Object, "spec", "completionMode")
		if mode != "Indexed" {
			continue
		}
		failedIdx, _, _ := unstructured.NestedString(job.Object, "status", "failedIndexes")
		if failedIdx == "" {
			continue
		}
		// Use the most recent transition into a non-empty failedIndexes
		// state — we approximate with the Job's startTime + grace.
		startTimeStr, _, _ := unstructured.NestedString(job.Object, "status", "startTime")
		var startTime time.Time
		if startTimeStr != "" {
			if t, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
				startTime = t
			}
		}
		if startTime.IsZero() {
			startTime = job.GetCreationTimestamp().Time
		}
		if startTime.IsZero() || now.Sub(startTime) < graceJobFailedIndexes {
			continue
		}
		ns := job.GetNamespace()
		name := job.GetName()
		out = append(out, Diagnostic{
			Source:   "DisruptionDrift",
			Subject:  "Job/" + ns + "/" + name,
			Severity: "warning",
			Message: fmt.Sprintf(
				"Indexed Job %s/%s has failed indexes [%s] past %s. The controller will keep retrying these indexes; consider whether the manifest needs a fix.",
				ns, name, failedIdx, graceJobFailedIndexes),
			Remediation: fmt.Sprintf(
				"Inspect the failed indexes and decide whether to recreate the Job with a corrected manifest:\n\n"+
					"  kubectl -n %s describe job %s\n"+
					"  kubectl -n %s logs -l job-name=%s --tail=200\n\n"+
					"If the failure is permanent (e.g. bad config), delete + recreate:\n\n"+
					"  kubectl -n %s delete job %s\n"+
					"  kubectl -n %s create -f <fixed-manifest>.yaml",
				ns, name, ns, name, ns, name, ns),
		})
	}
	return out
}

// runResourceQuota walks every ResourceQuota + flags namespaces
// whose `status.used == spec.hard` on at least one resource past
// graceQuotaAtMax. Saturation is normal mid-rollout (briefly
// hits 100% as new pods come up); a sustained 100% means net-new
// workloads will get the "exceeded quota" error.
func (a DisruptionDrift) runResourceQuota(ctx context.Context, src snapshot.Source) []Diagnostic {
	gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "resourcequotas"}
	list, err := src.List(ctx, gvr, "")
	if err != nil {
		logListFailure("resourcequotas", err, false)
		return nil
	}
	now := time.Now()
	var out []Diagnostic
	for i := range list.Items {
		rq := &list.Items[i]
		hard, _, _ := unstructured.NestedMap(rq.Object, "spec", "hard")
		used, _, _ := unstructured.NestedMap(rq.Object, "status", "used")
		if len(hard) == 0 || len(used) == 0 {
			continue
		}
		// Find resources at 100% — exact-string equality is the
		// right comparison here because Kubernetes round-trips
		// resource.Quantity through canonical formatting.
		var saturated []string
		for k, hardVal := range hard {
			if usedVal, ok := used[k]; ok && fmt.Sprintf("%v", usedVal) == fmt.Sprintf("%v", hardVal) {
				saturated = append(saturated, k)
			}
		}
		if len(saturated) == 0 {
			continue
		}
		// Grace anchor: use the RQ's creationTimestamp; resource
		// quotas don't carry per-condition transition times the way
		// PDBs do.
		anchor := rq.GetCreationTimestamp().Time
		if anchor.IsZero() || now.Sub(anchor) < graceQuotaAtMax {
			continue
		}
		ns := rq.GetNamespace()
		name := rq.GetName()
		out = append(out, Diagnostic{
			Source:   "DisruptionDrift",
			Subject:  "ResourceQuota/" + ns + "/" + name,
			Severity: "warning",
			Message: fmt.Sprintf(
				"ResourceQuota %s/%s is at 100%% on resources [%s] for >%s. New workloads in this namespace will be refused with confusing error messages.",
				ns, name, strings.Join(saturated, ","), graceQuotaAtMax),
			Remediation: fmt.Sprintf(
				"Inspect what's consuming the quota and either raise the limit or scale down stale workloads:\n\n"+
					"  kubectl -n %s describe resourcequota %s\n"+
					"  kubectl -n %s get pods --sort-by=.metadata.creationTimestamp\n\n"+
					"To raise:\n\n"+
					"  kubectl -n %s edit resourcequota %s\n"+
					"  # bump the saturated keys",
				ns, name, ns, ns, name),
		})
	}
	return out
}
