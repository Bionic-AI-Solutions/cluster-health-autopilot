// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package fix

import (
	"context"
	"fmt"
	"time"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
	"k8s.io/apimachinery/pkg/types"
)

// StuckRSPods ports fix_stuck_rs_pods from cluster-health-report.sh:362-400.
//
// Detects pods stuck in CreateContainerConfigError owned by a ReplicaSet
// whose Deployment has rolled forward. Trigger: live Deployment revision
// != stuck RS revision. Action: kubectl-style rollout restart via a strategic
// merge patch on the Deployment template's restart annotation.
//
// CRITICAL: skips the case where the failure event matches "couldn't find
// key" — a rollout would just create another pod against the same broken
// Secret. The diagnose_secret_key_missing analyzer is the right surface
// for that class.
//
// OWASP K8s Top-10 respected: K01 (Insecure Workload Configurations) — the
// only field written is a restartedAt timestamp annotation; never touches
// PodSpec security fields. See docs/OWASP_MAPPING.md and
// internal/fix/owasp_posture_test.go.
type StuckRSPods struct{}

// Name returns the fixer's identifier.
func (StuckRSPods) Name() string { return "StuckRSPods" }

// Run executes the fixer.
func (StuckRSPods) Run(ctx context.Context, src snapshot.Source, m snapshot.Mutator) Result {
	r := Result{Fixer: "StuckRSPods"}
	if m == nil {
		r.Refused = "snapshot mode — fixers require live cluster access"
		return r
	}

	pods, err := src.List(ctx, snapshot.GVRPod, "")
	if err != nil || len(pods.Items) == 0 {
		return r
	}

	// Restart at most once per Deployment per run.
	restarted := map[string]struct{}{}

	for i := range pods.Items {
		pod := pods.Items[i]
		ns := pod.GetNamespace()
		podObj := "Pod/" + ns + "/" + pod.GetName()

		if !podInCCE(pod) {
			continue
		}
		// "couldn't find key" → rollout would reproduce the same error.
		// The diagnose_secret_key_missing analyzer handles that pattern.
		if podHasMissingSecretKey(pod) {
			r.Skipped = append(r.Skipped, SkipReason{
				Object: podObj,
				Reason: "couldn't find key — rollout will not help; surfaced by diagnose analyzer",
			})
			continue
		}
		if IsProtectedNamespace(ns) {
			r.Skipped = append(r.Skipped, SkipReason{Object: podObj, Reason: "protected namespace"})
			continue
		}

		rsName := ownerNameByKind(pod.GetOwnerReferences(), "ReplicaSet")
		if rsName == "" {
			continue
		}

		rs, err := src.Get(ctx, snapshot.GVRReplicaSet, ns, rsName)
		if err != nil {
			r.Skipped = append(r.Skipped, SkipReason{
				Object: "ReplicaSet/" + ns + "/" + rsName,
				Reason: "couldn't fetch RS: " + err.Error(),
			})
			continue
		}
		deployName := ownerNameByKind(rs.GetOwnerReferences(), "Deployment")
		if deployName == "" {
			continue
		}
		deployKey := ns + "/" + deployName
		if _, dup := restarted[deployKey]; dup {
			continue
		}

		dep, err := src.Get(ctx, snapshot.GVRDeployment, ns, deployName)
		if err != nil {
			r.Skipped = append(r.Skipped, SkipReason{
				Object: "Deployment/" + deployKey,
				Reason: "couldn't fetch Deployment: " + err.Error(),
			})
			continue
		}
		// Refuse to roll-restart a GitOps-managed Deployment — the controller
		// would revert the restart annotation on its next reconcile, locking
		// CHA and the GitOps controller into a fight loop. The fix belongs in
		// the source repo. Dedup against deployKey so we don't emit the same
		// skip for every sibling stuck pod.
		if reason := GitOpsReason(*dep); reason != "" {
			if _, dup := restarted[deployKey]; !dup {
				r.Skipped = append(r.Skipped, SkipReason{
					Object: "Deployment/" + deployKey,
					Reason: "GitOps-managed: " + reason + " — edit the source repo instead",
				})
				restarted[deployKey] = struct{}{}
			}
			continue
		}
		// A paused rollout means an operator deliberately froze updates;
		// forcing a restart violates that intent.
		if IsPaused(*dep) {
			if _, dup := restarted[deployKey]; !dup {
				r.Skipped = append(r.Skipped, SkipReason{
					Object: "Deployment/" + deployKey,
					Reason: "rollout paused (spec.paused=true) — unpause before CHA may restart",
				})
				restarted[deployKey] = struct{}{}
			}
			continue
		}
		curRev := dep.GetAnnotations()["deployment.kubernetes.io/revision"]
		rsRev := rs.GetAnnotations()["deployment.kubernetes.io/revision"]
		if curRev == "" || rsRev == "" {
			continue
		}
		if curRev == rsRev {
			// RS is the live revision — rollout would just reproduce the same
			// stuck pod. (This is the case where the bug is in the live
			// template itself; needs human intervention.)
			r.Skipped = append(r.Skipped, SkipReason{
				Object: "Deployment/" + deployKey,
				Reason: fmt.Sprintf("RS revision %s == live revision; rollout would reproduce the failure", curRev),
			})
			continue
		}

		// Patch the deployment template to add a restart annotation. This is
		// what `kubectl rollout restart` actually does under the hood.
		patch := []byte(fmt.Sprintf(
			`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":%q}}}}}`,
			time.Now().UTC().Format(time.RFC3339),
		))
		if err := m.Patch(ctx, snapshot.GVRDeployment, ns, deployName, types.StrategicMergePatchType, patch); err != nil {
			r.Skipped = append(r.Skipped, SkipReason{
				Object: "Deployment/" + deployKey,
				Reason: "patch failed: " + err.Error(),
			})
			continue
		}
		restarted[deployKey] = struct{}{}
		r.Actions = append(r.Actions, Action{
			Description: fmt.Sprintf("Rolled deploy `%s` (stuck RS rev %s, live rev %s)", deployKey, rsRev, curRev),
			Object:      "Deployment/" + deployKey,
		})
	}
	return r
}
