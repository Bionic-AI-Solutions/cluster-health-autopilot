// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package probe

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/srenix-ai/agentic-sre/internal/snapshot"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// FailedMounts catches pods stuck in ContainerCreating because the kubelet
// cannot attach or mount one of their volumes. The PVCs probe sees the
// PVC as Bound; the Services probe sees the workload as 0/N ready; this
// probe is the missing link that names *why*.
//
// Signature:
//   - Pod phase Pending, container status waiting reason=ContainerCreating
//   - Associated Event with reason in {FailedMount, FailedAttachVolume,
//     FailedDetachVolume, ProvisioningFailed}
//
// Severity: CRITICAL once the pod has been Pending past MinAge (default
// 90s). The grace period lets transient mount races resolve without
// pager noise.
type FailedMounts struct {
	MinAge time.Duration
	Now    func() time.Time
}

// Name returns the component label.
func (FailedMounts) Name() string { return "Failed Mounts" }

// Run executes the probe.
func (f FailedMounts) Run(ctx context.Context, src snapshot.Source) Result {
	r := Result{Component: ComponentResult{Component: "Failed Mounts"}}
	minAge := f.MinAge
	if minAge == 0 {
		minAge = 90 * time.Second
	}
	now := time.Now
	if f.Now != nil {
		now = f.Now
	}

	pods, err := src.List(ctx, snapshot.GVRPod, "")
	if err != nil {
		r.Component.Status = "PROBE_FAILED"
		r.Component.Detail = "list pods: " + err.Error()
		return r
	}

	type hit struct {
		key, reason, message string
	}
	var hits []hit

	// Build a quick map of pod key -> mount-related event reasons.
	// We accept any Event whose involvedObject points at a pod we already
	// flagged as stuck ContainerCreating.
	mountReasons := map[string]struct{}{
		"FailedMount":        {},
		"FailedAttachVolume": {},
		"FailedDetachVolume": {},
		"ProvisioningFailed": {},
		"VolumeFailedMount":  {},
	}
	events, _ := src.List(ctx, snapshot.GVREvent, "")

	// Index events by involvedObject (pod) -> latest mount-related event.
	type evt struct{ reason, message string }
	podEvents := map[string]evt{}
	if events != nil {
		for _, e := range events.Items {
			reason, _, _ := unstructured.NestedString(e.Object, "reason")
			if _, ok := mountReasons[reason]; !ok {
				continue
			}
			kind, _, _ := unstructured.NestedString(e.Object, "involvedObject", "kind")
			if kind != "Pod" {
				continue
			}
			ns, _, _ := unstructured.NestedString(e.Object, "involvedObject", "namespace")
			name, _, _ := unstructured.NestedString(e.Object, "involvedObject", "name")
			msg, _, _ := unstructured.NestedString(e.Object, "message")
			podEvents[ns+"/"+name] = evt{reason: reason, message: msg}
		}
	}

	for _, pod := range pods.Items {
		if isTerminating(pod) {
			continue
		}
		phase, _, _ := unstructured.NestedString(pod.Object, "status", "phase")
		if phase != "Pending" {
			continue
		}
		if !podInContainerCreating(pod) {
			continue
		}
		// Grace period — give the kubelet time to mount.
		if minAge >= 0 {
			created := pod.GetCreationTimestamp()
			if !created.IsZero() && now().Sub(created.Time) < minAge {
				continue
			}
		}
		key := pod.GetNamespace() + "/" + pod.GetName()
		e, ok := podEvents[key]
		if !ok {
			// ContainerCreating without a mount event is not in scope —
			// could be image pull, scheduler delay, init container slow.
			continue
		}
		hits = append(hits, hit{key: key, reason: e.reason, message: e.message})
	}

	if len(hits) == 0 {
		r.Component.Status = "HEALTHY"
		r.Component.Detail = "No pods stuck on volume mount"
		return r
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].key < hits[j].key })
	parts := make([]string, 0, len(hits))
	for _, h := range hits {
		parts = append(parts, fmt.Sprintf("%s (%s)", h.key, h.reason))
		msg := h.message
		if len(msg) > 160 {
			msg = msg[:157] + "..."
		}
		r.Findings = append(r.Findings, Finding{
			Component: "Failed Mounts",
			Severity:  SeverityCritical,
			Message: fmt.Sprintf("Pod %s stuck — kubelet event %s: %s",
				h.key, h.reason, msg),
			Remediation: mountRemediation(h.reason),
		})
	}
	r.Component.Status = "CRITICAL"
	r.Component.Detail = fmt.Sprintf("%d pod(s) stuck on mount: %s",
		len(hits), strings.Join(parts, ", "))
	return r
}

// podInContainerCreating returns true when any container is waiting on
// ContainerCreating — the kubelet's "I have the pod spec but can't get
// the volumes mounted (or image pulled)" state. The volume signal comes
// from the associated Event, not this state alone, so this function only
// gates on the pod-level signature.
func podInContainerCreating(pod unstructured.Unstructured) bool {
	statuses, _, _ := getSliceField(pod.Object, "status", "containerStatuses")
	statuses = append(statuses, mustSlice(pod.Object, "status", "initContainerStatuses")...)
	for _, s := range statuses {
		sm, ok := s.(map[string]any)
		if !ok {
			continue
		}
		state, _ := sm["state"].(map[string]any)
		waiting, _ := state["waiting"].(map[string]any)
		if reason, _ := waiting["reason"].(string); reason == "ContainerCreating" {
			return true
		}
	}
	return false
}

func mountRemediation(reason string) string {
	switch reason {
	case "FailedMount", "VolumeFailedMount":
		return "Check the PVC's PV exists and is Available: `kubectl describe pvc <pvc>`, " +
			"`kubectl describe pv <pv>`. For rook-ceph: confirm OSD readiness and rbdplugin DS. " +
			"For NFS: check the export still permits the kubelet's IP."
	case "FailedAttachVolume":
		return "Volume is bound but couldn't be attached to the node: the CSI driver's controller " +
			"pod may be unhealthy. `kubectl get pods -A -l app=csi-<driver>-controller`."
	case "FailedDetachVolume":
		return "Volume is stuck detaching from a previous node. Forced detach often unblocks: " +
			"`kubectl patch pv <pv> --type=json -p='[{\"op\":\"remove\",\"path\":\"/spec/claimRef\"}]'` " +
			"— but verify the previous pod is fully gone first."
	case "ProvisioningFailed":
		return "StorageClass provisioner couldn't create a new PV: check the StorageClass exists " +
			"and the provisioner pod is healthy. Common cause: out-of-quota in the underlying " +
			"storage backend."
	}
	return "Inspect the kubelet/CSI driver logs on the node where the pod is scheduled"
}
