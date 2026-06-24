// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package diagnose

import (
	"context"
	"fmt"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// FailedPods catches pods that have reached the terminal `status.phase=Failed`
// state — the class of failure that no other analyzer covers:
//
//   - reason=UnexpectedAdmissionError — kubelet rejected the pod after it was
//     scheduled (e.g. device-plugin / resource reservation race). The pod is
//     dead but never enters CrashLoop (restartCount stays 0) and is not Pending
//     (the scheduler already placed it), so PendingPods and CrashLoopBackOff
//     both miss it.
//   - reason=Evicted — node-pressure eviction that the workload controller did
//     not replace (orphaned Failed pods linger).
//   - reason=NodeAffinity / NodeLost / Shutdown — terminal placement failures.
//
// Phase=Failed is terminal — a pod does not oscillate in and out of it — so no
// grace period is needed for stability. Pods with a deletionTimestamp are
// skipped: they are already being garbage-collected and reporting them would be
// transient noise.
//
// Severity is Critical: a Failed pod is a workload replica that is permanently
// down until something deletes and reschedules it. Opts out via
// CHA_ANALYZER_FAILED_PODS=off (handled at catalog registration).
type FailedPods struct{}

// Name satisfies the Analyzer contract.
func (FailedPods) Name() string { return "FailedPods" }

// Run walks every pod and emits one Critical diagnostic per pod in the terminal
// Failed phase that is not already being deleted.
func (FailedPods) Run(ctx context.Context, src snapshot.Source) []Diagnostic {
	pods, err := src.List(ctx, snapshot.GVRPod, "")
	if err != nil || pods == nil || len(pods.Items) == 0 {
		logListFailure("pods", err, true) // silent when absent; logs Forbidden etc.
		return nil
	}

	var out []Diagnostic
	for i := range pods.Items {
		pod := pods.Items[i]

		phase, _, _ := unstructured.NestedString(pod.Object, "status", "phase")
		if phase != "Failed" {
			continue
		}
		// Skip pods already terminating — their Failed state is transient
		// cleanup, not a stuck workload.
		if pod.GetDeletionTimestamp() != nil {
			continue
		}

		ns := pod.GetNamespace()
		name := pod.GetName()

		// Pod-level reason/message (e.g. "Evicted", "UnexpectedAdmissionError").
		reason, _, _ := unstructured.NestedString(pod.Object, "status", "reason")
		podMsg, _, _ := unstructured.NestedString(pod.Object, "status", "message")

		// Fall back to the most descriptive container terminated/waiting reason
		// when the pod-level reason is empty (some admission errors only stamp
		// the container status).
		detail := reason
		if cReason := firstFailedContainerReason(pod); cReason != "" {
			if detail == "" {
				detail = cReason
			} else if cReason != detail {
				detail = detail + " / " + cReason
			}
		}
		if detail == "" {
			detail = "unknown"
		}

		msg := fmt.Sprintf("Pod %s/%s is in terminal Failed phase (reason=%s)", ns, name, detail)
		if podMsg != "" {
			msg += ": " + truncate(podMsg, 200)
		}

		out = append(out, Diagnostic{
			Source:   "FailedPods",
			Subject:  fmt.Sprintf("Pod/%s/%s", ns, name),
			Severity: "critical",
			Message:  msg,
			Remediation: fmt.Sprintf(
				"Inspect why the pod was rejected/evicted, then delete it so its controller reschedules: "+
					"kubectl -n %s describe pod %s; kubectl -n %s delete pod %s", ns, name, ns, name),
		})
	}
	return out
}

// firstFailedContainerReason returns the terminated/waiting reason of the first
// container that carries one, across init + regular container statuses. Used to
// enrich the pod-level Failed message (e.g. ContainerStatusUnknown).
func firstFailedContainerReason(pod unstructured.Unstructured) string {
	for _, path := range [][]string{
		{"status", "initContainerStatuses"},
		{"status", "containerStatuses"},
	} {
		statuses, _, _ := unstructured.NestedSlice(pod.Object, path...)
		for _, raw := range statuses {
			cs, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			state, _ := cs["state"].(map[string]any)
			if term, _ := state["terminated"].(map[string]any); term != nil {
				if r, _ := term["reason"].(string); r != "" {
					return r
				}
			}
			if wait, _ := state["waiting"].(map[string]any); wait != nil {
				if r, _ := wait["reason"].(string); r != "" {
					return r
				}
			}
		}
	}
	return ""
}
