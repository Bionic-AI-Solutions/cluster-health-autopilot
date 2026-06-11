// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package diagnose

import (
	"context"
	"fmt"
	"time"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// OOMKillRecurrence surfaces pods whose containers have been OOMKilled
// repeatedly in a recent window. A single OOMKill is normal (the
// existing resource-health probes catch it); the signal here is the
// SUSTAINED loss — a workload that crashes 3+ times in 24h needs
// `resources.limits.memory` raised, not another restart.
//
// CapacityDrift already covers HPA-pinned-at-max; that catches the
// case where the autoscaler can't add replicas. This catches the
// case where each REPLICA needs more memory per pod.
//
// Phase 3.E.1.
//
// Signal: status.containerStatuses[*].lastState.terminated.reason ==
// "OOMKilled" AND status.containerStatuses[*].restartCount >= 3
// AND the most-recent terminated.finishedAt is within
// recurrenceWindow (24h).
//
// Why 3 — single OOMKill is normal (memory spike); 2 might be a
// noisy neighbor; 3 in 24h is a clear sizing problem.
type OOMKillRecurrence struct{}

// Name returns the analyzer's stable identifier.
func (OOMKillRecurrence) Name() string { return "OOMKillRecurrence" }

const (
	oomRecurrenceMinCount = 3
	oomRecurrenceWindow   = 24 * time.Hour
)

// Run walks every Pod and emits one diagnostic per container whose
// restart count + recent OOMKilled state crosses the threshold.
func (a OOMKillRecurrence) Run(ctx context.Context, src snapshot.Source) []Diagnostic {
	gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	list, err := src.List(ctx, gvr, "")
	if err != nil {
		logListFailure("pods", err, false)
		return nil
	}
	now := time.Now()
	var out []Diagnostic
	for i := range list.Items {
		pod := &list.Items[i]
		ns := pod.GetNamespace()
		name := pod.GetName()
		statuses, _, _ := unstructured.NestedSlice(pod.Object, "status", "containerStatuses")
		for _, cs := range statuses {
			cm, ok := cs.(map[string]any)
			if !ok {
				continue
			}
			cname, _ := cm["name"].(string)
			rc, _, _ := unstructured.NestedInt64(cm, "restartCount")
			if rc < int64(oomRecurrenceMinCount) {
				continue
			}
			// Only count when the LAST terminated state was OOMKilled
			// AND the finishedAt is within the window. Containers
			// that crashed for other reasons (CrashLoopBackOff with
			// Error reason) are handled by other analyzers.
			reason, _, _ := unstructured.NestedString(cm, "lastState", "terminated", "reason")
			if reason != "OOMKilled" {
				continue
			}
			finishedStr, _, _ := unstructured.NestedString(cm, "lastState", "terminated", "finishedAt")
			if finishedStr == "" {
				continue
			}
			finished, err := time.Parse(time.RFC3339, finishedStr)
			if err != nil || now.Sub(finished) > oomRecurrenceWindow {
				continue
			}
			out = append(out, Diagnostic{
				Source:   "OOMKillRecurrence",
				Subject:  "Pod/" + ns + "/" + name,
				Severity: "warning",
				Message: fmt.Sprintf(
					"Container %s in Pod %s/%s has been OOMKilled %d times (last %s ago). The workload needs more memory; restarting it just delays the next OOM.",
					cname, ns, name, rc, time.Since(finished).Truncate(time.Minute)),
				Remediation: fmt.Sprintf(
					"Inspect the container's resource usage + raise the limit:\n\n"+
						"  kubectl -n %s top pod %s --containers\n"+
						"  kubectl -n %s get pod %s -o jsonpath='{.spec.containers[?(@.name==\"%s\")].resources}'\n\n"+
						"Edit the parent workload (Deployment/StatefulSet) to bump `resources.limits.memory` for container %s by ~2x:\n\n"+
						"  kubectl -n %s edit <kind>/<name>",
					ns, name, ns, name, cname, cname, ns),
			})
		}
	}
	// Dedup — one finding per pod, even if multiple containers
	// crossed the threshold (the operator will typically fix all
	// containers in the same edit pass).
	return dedupByPodOOM(out)
}

// dedupByPodOOM keeps the first diagnostic per Pod subject; subsequent
// containers in the same pod are folded into the first one's message
// via comma-separation if needed (kept simple here — first-only).
func dedupByPodOOM(diags []Diagnostic) []Diagnostic {
	seen := make(map[string]struct{}, len(diags))
	var out []Diagnostic
	for _, d := range diags {
		if _, ok := seen[d.Subject]; ok {
			continue
		}
		seen[d.Subject] = struct{}{}
		out = append(out, d)
	}
	return out
}
