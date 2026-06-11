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

// CapacityDrift surfaces capacity-tier signals that the basic
// resource-health probes miss. Each signal here is "the cluster
// can't grow / can't shrink / can't size workloads the way the
// operator thinks they're sized."
//
// What's surfaced (v1.8 first cut):
//
//   - **HPA pinned at maxReplicas** — `status.currentReplicas ==
//     spec.maxReplicas` past the saturation grace window. The
//     workload is chronically under-provisioned and the HPA cannot
//     give it more headroom. Critical.
//
//   - **HPA pinned at minReplicas, not load-driven** — current
//     replicas have been equal to minReplicas for > 30 days AND
//     `spec.maxReplicas > minReplicas + 1`. The HPA's range is
//     effectively decorative; the workload could be a static
//     Deployment for the same outcome. Warning (it's wasted config,
//     not broken).
//
//   - **HPA unable to scale** — `status.conditions[type=AbleToScale,
//     status=False]` past grace. Usually a ResourceQuota cap or a
//     PodDisruptionBudget that's wedged the controller. Critical.
//
//   - **HPA metrics unavailable** — `status.conditions[
//     type=ScalingActive,status=False,reason=FailedGetResourceMetric]`.
//     Metrics-server is missing or unreachable; the HPA cannot
//     decide. Warning — this is the metrics-server-availability
//     signal the v1.8 roadmap R1 risk-mitigation calls for.
//
//   - **PVC expansion stuck** — `status.conditions[
//     type=FileSystemResizePending,status=True]` past grace, OR
//     `spec.resources.requests.storage` > `status.capacity.storage`
//     past grace. Volume-expansion got requested but the CSI driver
//     hasn't completed it. Critical (the workload typically blocks
//     on disk full while expansion hangs).
//
// Out of scope (deliberately, deferred to a v1.8.x follow-up — it
// needs metrics-server integration to add value beyond what HPA's
// own conditions already say):
//
//   - Pod resource-request vs actual-usage divergence (idle waste /
//     OOM risk) — needs metrics-server CPU/memory series.
//
//   - PVC growth-trajectory (linear-fit on PVC fill-rate; flag if
//     free space < 14 days at current rate) — needs metrics-server
//     `kubelet_volume_stats_*` or kube-state-metrics.
//
// Skip rules: kube-system / kube-public / kube-node-lease (their
// HPAs are control-plane-managed and rotations are expected).
type CapacityDrift struct {
	// GracePeriod is how long the "stuck" signals wait before
	// flagging. Zero uses defaultCapacityGrace.
	GracePeriod time.Duration

	// SaturationGrace is the dwell time required before
	// HPA-pinned-at-max is flagged. Zero uses defaultSaturationGrace
	// (24h).
	SaturationGrace time.Duration

	// IdleGrace is the dwell time required before HPA-pinned-at-min
	// is flagged. Zero uses defaultIdleGrace (30 days).
	IdleGrace time.Duration

	// Now returns the current time; overridable in tests.
	Now func() time.Time
}

// Name returns the analyzer's identifier. Pinned for metrics +
// dashboards.
func (CapacityDrift) Name() string { return "CapacityDrift" }

const (
	defaultCapacityGrace   = 15 * time.Minute
	defaultSaturationGrace = 24 * time.Hour
	defaultIdleGrace       = 30 * 24 * time.Hour
)

var (
	gvrHPA = schema.GroupVersionResource{
		Group:    "autoscaling",
		Version:  "v2",
		Resource: "horizontalpodautoscalers",
	}
)

// systemCapacityNamespaces are namespaces whose HPAs/PVCs are
// control-plane-managed; rotations are expected and noisy to flag.
var systemCapacityNamespaces = map[string]struct{}{
	"kube-system":     {},
	"kube-public":     {},
	"kube-node-lease": {},
}

// Run walks HPAs + PVCs and emits one Diagnostic per drift signal.
func (c CapacityDrift) Run(ctx context.Context, src snapshot.Source) []Diagnostic {
	now := c.Now
	if now == nil {
		now = time.Now
	}
	grace := c.GracePeriod
	if grace == 0 {
		grace = defaultCapacityGrace
	}
	satGrace := c.SaturationGrace
	if satGrace == 0 {
		satGrace = defaultSaturationGrace
	}
	idleGrace := c.IdleGrace
	if idleGrace == 0 {
		idleGrace = defaultIdleGrace
	}

	var out []Diagnostic
	out = append(out, c.checkHPAs(ctx, src, now(), grace, satGrace, idleGrace)...)
	out = append(out, c.checkPVCExpansion(ctx, src, now(), grace)...)
	return out
}

// checkHPAs walks autoscaling/v2 HorizontalPodAutoscalers and emits
// diagnostics for pinned-at-max, pinned-at-min, AbleToScale=False,
// and FailedGetResourceMetric.
func (c CapacityDrift) checkHPAs(ctx context.Context, src snapshot.Source, now time.Time, grace, satGrace, idleGrace time.Duration) []Diagnostic {
	list, err := src.List(ctx, gvrHPA, "")
	if err != nil || list == nil {
		logListFailure("horizontalpodautoscalers", err, false)
		return nil
	}
	var out []Diagnostic
	for i := range list.Items {
		hpa := &list.Items[i]
		ns := hpa.GetNamespace()
		if _, isSystem := systemCapacityNamespaces[ns]; isSystem {
			continue
		}
		name := hpa.GetName()
		subject := fmt.Sprintf("HorizontalPodAutoscaler/%s/%s", ns, name)

		minReplicas, _, _ := unstructured.NestedInt64(hpa.Object, "spec", "minReplicas")
		maxReplicas, _, _ := unstructured.NestedInt64(hpa.Object, "spec", "maxReplicas")
		current, _, _ := unstructured.NestedInt64(hpa.Object, "status", "currentReplicas")
		lastScale, _, _ := unstructured.NestedString(hpa.Object, "status", "lastScaleTime")

		// AbleToScale + ScalingActive conditions.
		conds, _, _ := unstructured.NestedSlice(hpa.Object, "status", "conditions")
		ableFalse, ableSince, ableReason, ableMsg := condStatus(conds, "AbleToScale", "False")
		_, _, scalingActiveReason, scalingActiveMsg := condStatus(conds, "ScalingActive", "False")

		// Use lastScaleTime when present (HPA actually scaled at
		// some point) so the dwell-time math reflects how long the
		// HPA has been in its current state. Fall back to the HPA's
		// creationTimestamp.
		dwellAnchor := hpa.GetCreationTimestamp().Time
		if t, err := time.Parse(time.RFC3339, lastScale); err == nil {
			dwellAnchor = t
		}
		dwell := now.Sub(dwellAnchor)

		// 1. AbleToScale=False past grace.
		if ableFalse {
			if ableSinceT, err := time.Parse(time.RFC3339, ableSince); err == nil {
				if now.Sub(ableSinceT) < grace {
					goto checkPinnedMax
				}
			}
			out = append(out, Diagnostic{
				Source:   "CapacityDrift",
				Subject:  subject,
				Severity: "critical",
				Message: fmt.Sprintf(
					"HPA %s/%s reports AbleToScale=False past %s: reason=%s, message=%q",
					ns, name, grace, ableReason, ableMsg),
				Remediation: "The HPA can't scale the target. Common causes: a ResourceQuota in the namespace blocks new pods, a PodDisruptionBudget is wedged, or the target Deployment's selector mismatches its pods. " +
					"`kubectl describe hpa " + name + " -n " + ns + "` shows the controller's most recent decision; `kubectl describe quota -n " + ns + "` shows quota state.",
			})
			continue // Don't pile up pinned-at-max/min on top of AbleToScale=False — too noisy.
		}

	checkPinnedMax:
		// 2. Pinned at maxReplicas past saturation grace.
		// Skip when min==max: the operator configured a static
		// replica count and the HPA has no headroom to grow into
		// even in principle. Flagging is noise.
		if current >= maxReplicas && maxReplicas > 0 && maxReplicas > minReplicas && dwell >= satGrace {
			out = append(out, Diagnostic{
				Source:   "CapacityDrift",
				Subject:  subject,
				Severity: "critical",
				Message: fmt.Sprintf(
					"HPA %s/%s pinned at maxReplicas=%d for >%s; workload is chronically under-provisioned",
					ns, name, maxReplicas, satGrace),
				Remediation: "The HPA wants to scale further but the maxReplicas ceiling is blocking. Raise `spec.maxReplicas` after verifying the underlying nodes have capacity (`kubectl top nodes`). " +
					"If node capacity is the gate, scale the cluster autoscaler or add nodes before raising the HPA cap.",
			})
			continue
		}

		// 3. Pinned at minReplicas past idle grace (not load-driven).
		if current <= minReplicas && minReplicas > 0 && maxReplicas > minReplicas+1 && dwell >= idleGrace {
			out = append(out, Diagnostic{
				Source:   "CapacityDrift",
				Subject:  subject,
				Severity: "warning",
				Message: fmt.Sprintf(
					"HPA %s/%s pinned at minReplicas=%d for >%s with maxReplicas=%d unused; HPA is not load-driven (effectively decorative)",
					ns, name, minReplicas, idleGrace, maxReplicas),
				Remediation: fmt.Sprintf(
					"The HPA's autoscaling range hasn't been exercised in over %s. Options: (a) lower spec.maxReplicas to match actual demand "+
						"(reduces ResourceQuota pressure), (b) convert the workload to a static Deployment with replicas=%d, or "+
						"(c) tune the HPA's target utilization down so it actually scales out under realistic load.",
					idleGrace, minReplicas),
			})
			continue
		}

		// 4. ScalingActive=False, reason=FailedGetResourceMetric.
		if scalingActiveReason == "FailedGetResourceMetric" {
			out = append(out, Diagnostic{
				Source:   "CapacityDrift",
				Subject:  subject,
				Severity: "warning",
				Message: fmt.Sprintf(
					"HPA %s/%s reports FailedGetResourceMetric: metrics-server is missing or unreachable",
					ns, name),
				Remediation: "The HPA cannot decide because the metrics pipeline is down. Verify metrics-server is installed and Ready: " +
					"`kubectl get deploy metrics-server -n kube-system`. " +
					"On managed clusters (EKS / GKE / AKS) metrics-server is typically an opt-in addon — install it from the cluster console, " +
					"or `helm install metrics-server metrics-server/metrics-server -n kube-system`. " +
					"While metrics are unavailable the HPA holds at its current replica count and the workload may saturate without scaling. " +
					"Detail: " + scalingActiveMsg,
			})
		}
	}
	return out
}

// checkPVCExpansion walks PVCs and flags any whose volume-expansion
// got requested but the CSI driver hasn't completed it.
func (c CapacityDrift) checkPVCExpansion(ctx context.Context, src snapshot.Source, now time.Time, grace time.Duration) []Diagnostic {
	list, err := src.List(ctx, snapshot.GVRPVC, "")
	if err != nil || list == nil {
		logListFailure("persistentvolumeclaims", err, false)
		return nil
	}
	var out []Diagnostic
	for i := range list.Items {
		pvc := &list.Items[i]
		ns := pvc.GetNamespace()
		if _, isSystem := systemCapacityNamespaces[ns]; isSystem {
			continue
		}
		name := pvc.GetName()

		// Two signals; only emit once per PVC.
		conds, _, _ := unstructured.NestedSlice(pvc.Object, "status", "conditions")
		// 1. FileSystemResizePending=True past grace.
		pending, since, _, msg := condStatus(conds, "FileSystemResizePending", "True")
		if pending {
			if sinceT, err := time.Parse(time.RFC3339, since); err == nil {
				if now.Sub(sinceT) < grace {
					continue
				}
			}
			out = append(out, Diagnostic{
				Source:   "CapacityDrift",
				Subject:  fmt.Sprintf("PersistentVolumeClaim/%s/%s", ns, name),
				Severity: "critical",
				Message: fmt.Sprintf(
					"PVC %s/%s reports FileSystemResizePending=True past %s; CSI driver hasn't finished the resize",
					ns, name, grace),
				Remediation: "The volume's filesystem hasn't been expanded yet. Typically the resize completes on the next pod restart for offline-expansion CSI drivers. " +
					"For online expansion (most modern drivers): `kubectl describe pvc " + name + " -n " + ns + "` to see the CSI driver's last attempt, and check `kubectl -n kube-system logs deploy/<csi-driver>-controller` for the actual failure. " +
					"Detail: " + msg,
			})
			continue
		}

		// 2. spec.requests.storage > status.capacity.storage past grace.
		req, _, _ := unstructured.NestedString(pvc.Object, "spec", "resources", "requests", "storage")
		cap, _, _ := unstructured.NestedString(pvc.Object, "status", "capacity", "storage")
		if req == "" || cap == "" || req == cap {
			continue
		}
		// PVC creation timestamp gives us age but not "when was
		// resize requested"; without an annotation we use creation
		// + grace as a coarse signal — if the PVC was created over
		// `grace` ago AND requested > capacity, the resize has been
		// pending too long.
		if now.Sub(pvc.GetCreationTimestamp().Time) < grace {
			continue
		}
		out = append(out, Diagnostic{
			Source:   "CapacityDrift",
			Subject:  fmt.Sprintf("PersistentVolumeClaim/%s/%s", ns, name),
			Severity: "critical",
			Message: fmt.Sprintf(
				"PVC %s/%s requests storage=%s but status.capacity=%s past %s; volume expansion is stuck",
				ns, name, req, cap, grace),
			Remediation: renderPVCExpansionRemediation(ctx, src, pvc),
		})
	}
	return out
}

// renderPVCExpansionRemediation looks up the PVC's StorageClass and
// emits an SC-aware remediation. The legacy text told the operator to
// run `kubectl get storageclass <name>` themselves; the AI tier
// surfacing this diagnostic has no way to interpret that placeholder.
// We do the lookup here so the remediation is specific to this PVC's
// SC + allowVolumeExpansion value (branch-collapsed, not both branches
// joined by "if true ... if false ...").
//
// Three paths:
//   - SC named on PVC + found in snapshot + allowVolumeExpansion=true →
//     "SC <X> allows expansion; the CSI driver hasn't completed it.
//     Inspect ..." (no migration advice — the path forward is to
//     unblock the driver).
//   - SC named + found + allowVolumeExpansion=false → "SC <X> does not
//     support expansion. Expansion is impossible — create a new PVC
//     at the larger size and migrate." (no driver-log advice — driver
//     isn't going to do anything).
//   - SC missing from snapshot OR allowVolumeExpansion unset OR PVC has
//     no spec.storageClassName → fall back to a generic text that still
//     names whatever we know (no literal `<name>` token) and gives both
//     branches.
func renderPVCExpansionRemediation(ctx context.Context, src snapshot.Source, pvc *unstructured.Unstructured) string {
	scName, _, _ := unstructured.NestedString(pvc.Object, "spec", "storageClassName")
	if scName == "" {
		return "The PVC has no spec.storageClassName set; the cluster default StorageClass was used implicitly. " +
			"Run `kubectl get storageclass` to identify it, then check `allowVolumeExpansion`. " +
			"If false, expansion is impossible — create a new PVC at the larger size and migrate. " +
			"If true, inspect the CSI driver controller logs."
	}
	sc, err := src.Get(ctx, snapshot.GVRStorageClass, "", scName)
	if err != nil || sc == nil {
		return fmt.Sprintf(
			"The PVC's StorageClass `%s` was not found in the snapshot (likely an RBAC or CRD gap). "+
				"Verify `kubectl get storageclass %s -o jsonpath='{.allowVolumeExpansion}'` directly. "+
				"If false, expansion is impossible — create a new PVC at the larger size and migrate. "+
				"If true, inspect the CSI driver controller logs.",
			scName, scName)
	}
	allow, found, _ := unstructured.NestedBool(sc.Object, "allowVolumeExpansion")
	if !found {
		return fmt.Sprintf(
			"StorageClass `%s` does not declare `allowVolumeExpansion` (defaults to false in Kubernetes). "+
				"Confirm with `kubectl get storageclass %s -o jsonpath='{.allowVolumeExpansion}'`. "+
				"If the field is empty or false, expansion is impossible — create a new PVC at the larger size and migrate. "+
				"If true, inspect the CSI driver controller logs.",
			scName, scName)
	}
	if allow {
		return fmt.Sprintf(
			"StorageClass `%s` allowVolumeExpansion=true, so the resize was accepted but the CSI driver hasn't completed it. "+
				"Inspect the CSI driver controller logs (`kubectl -n kube-system logs deploy/<csi-driver>-controller`) for the actual error. "+
				"Common causes: backend storage out of capacity, controller pod crash-looping, or PVC bound to a PV whose underlying volume can't grow.",
			scName)
	}
	return fmt.Sprintf(
		"StorageClass `%s` allowVolumeExpansion=false, so expansion is impossible on this storage class. "+
			"Create a new PVC at the larger size on an expansion-capable StorageClass and migrate the data (copy via a one-shot Job, or use a backup/restore workflow).",
		scName)
}

// condStatus walks a conditions slice and returns true when a
// condition matching type/status exists, plus its lastTransitionTime,
// reason, and message.
func condStatus(conds []interface{}, condType, condStatusValue string) (matched bool, since, reason, msg string) {
	for _, c := range conds {
		cond, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		t, _ := cond["type"].(string)
		s, _ := cond["status"].(string)
		if t != condType {
			continue
		}
		// Track the reason/message for the matching type even on
		// non-matching status — callers may want to surface
		// "ScalingActive=False" reason regardless.
		reason, _ = cond["reason"].(string)
		msg, _ = cond["message"].(string)
		since, _ = cond["lastTransitionTime"].(string)
		if !strings.EqualFold(s, condStatusValue) {
			continue
		}
		matched = true
		return
	}
	return
}
