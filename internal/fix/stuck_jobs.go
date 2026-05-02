// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package fix

import (
	"context"
	"strings"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// StuckJobsWithBadSecretRef ports fix_stuck_jobs_with_bad_secret_ref from
// cluster-health-report.sh:354-388.
//
// When a CronJob's spec is updated (e.g. a renamed Secret key), the previous
// Job's pod template is immutable and locks out the corrected CronJob via
// concurrencyPolicy: Forbid. The fixer detects the "couldn't find key"
// signature on a Job-owned pod, verifies the parent CronJob still exists,
// and deletes the frozen Job so the CronJob's next tick respawns clean.
//
// This is the bug class that bit gpu-docker-monitor on 2026-04-28: a
// Secret-key rename rolled the CronJob template forward, but the in-flight
// Job pod kept retrying CreateContainer against the old key for 26 days
// before manual intervention. With this fixer, the next health-report tick
// would have caught and resolved it automatically.
type StuckJobsWithBadSecretRef struct{}

// Name returns the fixer's identifier.
func (StuckJobsWithBadSecretRef) Name() string { return "StuckJobsWithBadSecretRef" }

// Run executes the fixer.
func (StuckJobsWithBadSecretRef) Run(ctx context.Context, src snapshot.Source, m snapshot.Mutator) Result {
	r := Result{Fixer: "StuckJobsWithBadSecretRef"}
	if m == nil {
		r.Refused = "snapshot mode — fixers require live cluster access"
		return r
	}

	pods, err := src.List(ctx, snapshot.GVRPod, "")
	if err != nil || len(pods.Items) == 0 {
		return r
	}

	// Avoid deleting the same Job twice when multiple pods belong to it.
	deletedJobs := map[string]struct{}{}

	for i := range pods.Items {
		pod := pods.Items[i]
		ns := pod.GetNamespace()
		podName := pod.GetName()
		podObj := "Pod/" + ns + "/" + podName

		jobName := ownerNameByKind(pod.GetOwnerReferences(), "Job")
		if jobName == "" {
			continue
		}
		if !podHasMissingSecretKey(pod) {
			continue
		}
		if IsProtectedNamespace(ns) {
			r.Skipped = append(r.Skipped, SkipReason{Object: podObj, Reason: "protected namespace"})
			continue
		}

		jobKey := ns + "/" + jobName
		if _, dup := deletedJobs[jobKey]; dup {
			continue
		}

		// Verify the Job's parent CronJob still exists — without it, deleting
		// the Job won't bring anything back.
		jobObj, err := src.Get(ctx, snapshot.GVRJob, ns, jobName)
		if err != nil {
			r.Skipped = append(r.Skipped, SkipReason{
				Object: "Job/" + jobKey,
				Reason: "couldn't fetch Job (already gone?): " + err.Error(),
			})
			continue
		}
		cronJobName := ownerNameByKind(jobObj.GetOwnerReferences(), "CronJob")
		if cronJobName == "" {
			r.Skipped = append(r.Skipped, SkipReason{
				Object: "Job/" + jobKey,
				Reason: "Job has no CronJob owner; deletion would not auto-respawn",
			})
			continue
		}

		if err := m.Delete(ctx, snapshot.GVRJob, ns, jobName); err != nil {
			r.Skipped = append(r.Skipped, SkipReason{
				Object: "Job/" + jobKey,
				Reason: "delete failed: " + err.Error(),
			})
			continue
		}
		deletedJobs[jobKey] = struct{}{}
		r.Actions = append(r.Actions, Action{
			Description: "Deleted stuck Job (frozen Secret-key ref); CronJob `" +
				ns + "/" + cronJobName + "` will respawn",
			Object: "Job/" + jobKey,
		})
	}
	return r
}

// ownerNameByKind returns the .name of the first ownerReference whose
// .kind matches, or "". Stable across the metav1 API surface.
func ownerNameByKind(refs []metav1.OwnerReference, kind string) string {
	for _, o := range refs {
		if o.Kind == kind {
			return o.Name
		}
	}
	return ""
}

// podInCCE reports whether any container's waiting reason is
// CreateContainerConfigError. Mirrors the bash awk on the STATUS column
// against the structured pod state.
func podInCCE(pod unstructured.Unstructured) bool {
	statuses, _, _ := unstructured.NestedSlice(pod.Object, "status", "containerStatuses")
	for _, s := range statuses {
		sm, ok := s.(map[string]any)
		if !ok {
			continue
		}
		state, _ := sm["state"].(map[string]any)
		waiting, _ := state["waiting"].(map[string]any)
		if reason, _ := waiting["reason"].(string); reason == "CreateContainerConfigError" {
			return true
		}
	}
	return false
}

// podHasMissingSecretKey reports whether any container's waiting message
// contains the kubelet "couldn't find key" signature.
func podHasMissingSecretKey(pod unstructured.Unstructured) bool {
	statuses, _, _ := unstructured.NestedSlice(pod.Object, "status", "containerStatuses")
	for _, s := range statuses {
		sm, ok := s.(map[string]any)
		if !ok {
			continue
		}
		state, _ := sm["state"].(map[string]any)
		waiting, _ := state["waiting"].(map[string]any)
		msg, _ := waiting["message"].(string)
		if strings.Contains(msg, "couldn't find key") || strings.Contains(msg, "couldnt find key") {
			return true
		}
	}
	return false
}
