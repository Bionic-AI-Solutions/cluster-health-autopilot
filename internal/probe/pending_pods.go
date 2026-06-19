// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package probe

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// PendingPods catches pods that the scheduler cannot place. Distinguished
// from ImagePullBackOff / CreateContainerConfigError, which the existing
// ImagePullAuth and StuckRSPods analyzers handle — this probe focuses on
// the `status.phase=Pending` + `conditions[type=PodScheduled].status=False`
// signature, which means the kube-scheduler walked the cluster and could
// not find a node that satisfies the pod's requirements.
//
// Common causes (named in the Finding's Remediation):
//   - reason=Unschedulable, message="0/N nodes are available: insufficient cpu"
//   - reason=Unschedulable, message="0/N nodes are available: node(s) had taint"
//   - reason=Unschedulable, message="pod has unbound immediate PersistentVolumeClaims"
//
// Grace period: only pods that have been Pending for at least MinAge are
// reported. New pods are Pending briefly during scheduling; if MinAge is
// 0 the probe defaults to 60s. Set MinAge=-1 to disable.
type PendingPods struct {
	// MinAge is the minimum time a pod must have been in Pending phase
	// (relative to creationTimestamp) before we flag it. Default: 60s.
	MinAge time.Duration
	// Now is the current-time provider; tests inject a fixed clock.
	Now func() time.Time
}

// Name returns the component label for the report.
func (PendingPods) Name() string { return "Pending Pods" }

// Run executes the probe.
func (p PendingPods) Run(ctx context.Context, src snapshot.Source) Result {
	r := Result{Component: ComponentResult{Component: "Pending Pods"}}
	minAge := p.MinAge
	if minAge == 0 {
		minAge = 60 * time.Second
	}
	now := time.Now
	if p.Now != nil {
		now = p.Now
	}

	pods, err := src.List(ctx, snapshot.GVRPod, "")
	if err != nil {
		r.Component.Status = "PROBE_FAILED"
		r.Component.Detail = "list pods: " + err.Error()
		return r
	}

	type pending struct {
		key, reason, message string
	}
	var hits []pending
	for _, pod := range pods.Items {
		if isTerminating(pod) {
			continue
		}
		phase, _, _ := unstructured.NestedString(pod.Object, "status", "phase")
		if phase != "Pending" {
			continue
		}
		// Skip pods whose Pending state is scheduling-success but image-pull
		// failure. ImagePullAuth handles those.
		if isImagePullFailure(pod) {
			continue
		}
		// Find PodScheduled=False, which is the canonical scheduler-couldn't-
		// place signal.
		conds, _, _ := getSliceField(pod.Object, "status", "conditions")
		var schedReason, schedMessage string
		schedFailed := false
		for _, c := range conds {
			cm, ok := c.(map[string]any)
			if !ok {
				continue
			}
			if cm["type"] == "PodScheduled" && cm["status"] == "False" {
				schedFailed = true
				schedReason, _ = cm["reason"].(string)
				schedMessage, _ = cm["message"].(string)
			}
		}
		if !schedFailed {
			continue
		}
		// Grace period — skip pods that just got created.
		if minAge >= 0 {
			created := pod.GetCreationTimestamp()
			if !created.IsZero() && now().Sub(created.Time) < minAge {
				continue
			}
		}
		key := pod.GetNamespace() + "/" + pod.GetName()
		hits = append(hits, pending{key: key, reason: schedReason, message: schedMessage})
	}

	if len(hits) == 0 {
		r.Component.Status = "HEALTHY"
		r.Component.Detail = "No pods Pending past grace period"
		return r
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].key < hits[j].key })
	parts := make([]string, 0, len(hits))
	for _, h := range hits {
		// Truncate scheduler messages — they can be very long.
		msg := h.message
		if len(msg) > 120 {
			msg = msg[:117] + "..."
		}
		parts = append(parts, fmt.Sprintf("%s (%s)", h.key, h.reason))
		r.Findings = append(r.Findings, Finding{
			Component: "Pending Pods",
			Severity:  SeverityCritical,
			Message: fmt.Sprintf("Pod %s stuck Pending — scheduler reason=%s: %s",
				h.key, h.reason, msg),
			Remediation: schedulingRemediation(h.message),
		})
	}
	r.Component.Status = "CRITICAL"
	r.Component.Detail = fmt.Sprintf("%d pod(s) stuck Pending: %s",
		len(hits), strings.Join(parts, ", "))
	return r
}

// isImagePullFailure returns true when the pod is Pending due to image
// pull issues (ImagePullBackOff / ErrImagePull). Handled by ImagePullAuth.
func isImagePullFailure(pod unstructured.Unstructured) bool {
	statuses, _, _ := getSliceField(pod.Object, "status", "containerStatuses")
	statuses = append(statuses, mustSlice(pod.Object, "status", "initContainerStatuses")...)
	for _, s := range statuses {
		sm, ok := s.(map[string]any)
		if !ok {
			continue
		}
		state, _ := sm["state"].(map[string]any)
		waiting, _ := state["waiting"].(map[string]any)
		reason, _ := waiting["reason"].(string)
		switch reason {
		case "ImagePullBackOff", "ErrImagePull", "InvalidImageName":
			return true
		}
	}
	return false
}

func mustSlice(m map[string]any, path ...string) []any {
	s, _, _ := getSliceField(m, path...)
	return s
}

func schedulingRemediation(message string) string {
	m := strings.ToLower(message)
	switch {
	case strings.Contains(m, "insufficient cpu"):
		return "Cluster CPU exhausted: scale up node count, or right-size pod requests (`kubectl describe pod <p>` for the requested CPU)"
	case strings.Contains(m, "insufficient memory"):
		return "Cluster memory exhausted: scale up node count, or right-size pod requests"
	case strings.Contains(m, "node(s) had taint") || strings.Contains(m, "didn't tolerate"):
		return "Pod doesn't tolerate the matching nodes' taints. Either add a toleration on the pod or remove the taint via `kubectl taint nodes <name> <key>-`"
	case strings.Contains(m, "unbound immediate persistentvolumeclaims"):
		return "PVC isn't bound. Check the PVC's StorageClass and provisioner: `kubectl describe pvc <pvc>`; for rook-ceph see [ceph-osd] / OSD readiness"
	case strings.Contains(m, "node selector") || strings.Contains(m, "node affinity"):
		return "No node matches the pod's nodeSelector/affinity. Check labels: `kubectl get nodes --show-labels` against the pod's selector"
	case strings.Contains(m, "had untolerated taint"):
		return "Add a matching toleration or untaint the node"
	default:
		return "Inspect the scheduler verdict: `kubectl describe pod <pod>` and check capacity, taints/tolerations, and PVC binding"
	}
}
