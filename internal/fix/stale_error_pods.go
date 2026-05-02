// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package fix

import (
	"context"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
)

// StaleErrorPods ports fix_stale_error_pods from cluster-health-report.sh:298-319.
//
// Deletes pods whose status.phase == "Failed" (covers both Error and Failed
// shown in the bash STATUS column) when the pod is owned by a Job or has
// no controller owner. Owners that auto-respawn (Deployments, StatefulSets,
// DaemonSets) are skipped — the controller will recover.
//
// This catches stale debug pods (kubectl debug, manual `kubectl run`
// leftovers) and Job-spawned pods that crashed without a retry policy.
type StaleErrorPods struct{}

// Name returns the fixer's identifier.
func (StaleErrorPods) Name() string { return "StaleErrorPods" }

// Run executes the fixer.
func (StaleErrorPods) Run(ctx context.Context, src snapshot.Source, m snapshot.Mutator) Result {
	r := Result{Fixer: "StaleErrorPods"}
	if m == nil {
		r.Refused = "snapshot mode — fixers require live cluster access"
		return r
	}

	pods, err := src.List(ctx, snapshot.GVRPod, "")
	if err != nil || len(pods.Items) == 0 {
		return r
	}

	for i := range pods.Items {
		pod := pods.Items[i]
		ns := pod.GetNamespace()
		name := pod.GetName()
		object := "Pod/" + ns + "/" + name

		if IsProtectedNamespace(ns) {
			r.Skipped = append(r.Skipped, SkipReason{
				Object: object,
				Reason: "protected namespace",
			})
			continue
		}

		// Status phase Failed covers both Error/CrashLoop terminals. The
		// kubectl STATUS column "Error" surfaces the same underlying state.
		phase, _, _ := nestedString(pod.Object, "status", "phase")
		if phase != "Failed" {
			continue
		}

		// Only act on Job-owned or unowned pods. Owners that auto-respawn
		// (Deployment, StatefulSet, DaemonSet) are skipped — let the controller
		// reconcile.
		owners := pod.GetOwnerReferences()
		if len(owners) > 0 {
			ownerKind := owners[0].Kind
			if ownerKind != "Job" {
				r.Skipped = append(r.Skipped, SkipReason{
					Object: object,
					Reason: "owned by " + ownerKind + " (controller will recover)",
				})
				continue
			}
		}

		if err := m.Delete(ctx, snapshot.GVRPod, ns, name); err != nil {
			r.Skipped = append(r.Skipped, SkipReason{
				Object: object,
				Reason: "delete failed: " + err.Error(),
			})
			continue
		}

		desc := "Deleted stale Failed pod"
		if len(owners) > 0 {
			desc = "Deleted stale Failed pod (owner: " + owners[0].Kind + "/" + owners[0].Name + ")"
		}
		r.Actions = append(r.Actions, Action{Description: desc, Object: object})
	}
	return r
}

// nestedString is a tiny helper around the unstructured map shape, kept
// here to avoid pulling probe's helpers across packages. Returns
// (value, found, error). Error is always nil — kept in the signature for
// symmetry with the upstream apimachinery API we may swap to later.
func nestedString(m map[string]any, path ...string) (string, bool, error) {
	cur := any(m)
	for i, k := range path {
		mp, ok := cur.(map[string]any)
		if !ok {
			return "", false, nil
		}
		cur, ok = mp[k]
		if !ok {
			return "", false, nil
		}
		if i == len(path)-1 {
			s, ok := cur.(string)
			if !ok {
				return "", false, nil
			}
			return s, true, nil
		}
	}
	return "", false, nil
}
