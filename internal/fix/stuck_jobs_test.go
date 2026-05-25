// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package fix

import (
	"context"
	"testing"
)

const podsAndJobsForFixer = `{
  "apiVersion": "v1",
  "kind": "PodList",
  "items": [
    {
      "apiVersion": "v1", "kind": "Pod",
      "metadata": {
        "name": "gpu-docker-monitor-29622780-rdhww",
        "namespace": "gpu-monitor",
        "ownerReferences": [{"kind": "Job", "name": "gpu-docker-monitor-29622780"}]
      },
      "status": {
        "containerStatuses": [{
          "name": "checker",
          "state": {"waiting": {
            "reason": "CreateContainerConfigError",
            "message": "couldn't find key webhook-url in Secret gpu-monitor/gpu-monitor-slack"
          }}
        }]
      }
    },
    {
      "apiVersion": "v1", "kind": "Pod",
      "metadata": {
        "name": "happy-job-pod",
        "namespace": "batch",
        "ownerReferences": [{"kind": "Job", "name": "nightly-cleanup"}]
      },
      "status": {"phase": "Running"}
    },
    {
      "apiVersion": "v1", "kind": "Pod",
      "metadata": {
        "name": "stuck-cron-without-cronjob",
        "namespace": "demo",
        "ownerReferences": [{"kind": "Job", "name": "one-off-job"}]
      },
      "status": {
        "containerStatuses": [{
          "name": "x",
          "state": {"waiting": {
            "reason": "CreateContainerConfigError",
            "message": "couldn't find key X in Secret demo/missing"
          }}
        }]
      }
    }
  ]
}`

const cronOwnedJob = `{
  "apiVersion": "batch/v1", "kind": "Job",
  "metadata": {
    "name": "gpu-docker-monitor-29622780",
    "namespace": "gpu-monitor",
    "ownerReferences": [{"kind": "CronJob", "name": "gpu-docker-monitor"}]
  }
}`

const oneOffJob = `{
  "apiVersion": "batch/v1", "kind": "Job",
  "metadata": {"name": "one-off-job", "namespace": "demo"}
}`

// TestStuckJobs_RefusesOnSnapshot — type-system gate.
func TestStuckJobs_RefusesOnSnapshot(t *testing.T) {
	src := loadSrc(t, map[string]string{"pods.json": podsAndJobsForFixer, "job.json": cronOwnedJob})
	r := StuckJobsWithBadSecretRef{}.Run(context.Background(), src, nil)
	if r.Refused == "" {
		t.Errorf("expected Refused on snapshot mode")
	}
}

// TestStuckJobs_DeletesCronOwnedJob — happy path.
func TestStuckJobs_DeletesCronOwnedJob(t *testing.T) {
	src := loadSrc(t, map[string]string{
		"pods.json":    podsAndJobsForFixer,
		"job.json":     cronOwnedJob,
		"cronjob.json": activeCronJob,
		"oneoff.json":  oneOffJob,
	})
	m := newFakeMutator()
	r := StuckJobsWithBadSecretRef{}.Run(context.Background(), src, m)

	// Should delete exactly one Job: the CronJob-owned, secret-key-stuck one.
	if got, want := len(r.Actions), 1; got != want {
		t.Fatalf("Actions = %d, want %d (full: %+v)", got, want, r.Actions)
	}
	wantCalls := []string{"Delete jobs/gpu-monitor/gpu-docker-monitor-29622780"}
	if got := m.sortedCalls(); !equalStrings(got, wantCalls) {
		t.Errorf("calls = %v, want %v", got, wantCalls)
	}
}

// TestStuckJobs_SkipsOneOffJob — Job without CronJob parent must NOT be deleted.
func TestStuckJobs_SkipsOneOffJob(t *testing.T) {
	src := loadSrc(t, map[string]string{
		"pods.json":    podsAndJobsForFixer,
		"job.json":     cronOwnedJob,
		"cronjob.json": activeCronJob,
		"oneoff.json":  oneOffJob,
	})
	m := newFakeMutator()
	r := StuckJobsWithBadSecretRef{}.Run(context.Background(), src, m)

	for _, c := range m.calls {
		if c == "Delete jobs/demo/one-off-job" {
			t.Errorf("one-off Job (no CronJob parent) should NOT have been deleted")
		}
	}
	foundSkip := false
	for _, s := range r.Skipped {
		if s.Object == "Job/demo/one-off-job" {
			foundSkip = true
			if s.Reason != "Job has no CronJob owner; deletion would not auto-respawn" {
				t.Errorf("one-off-job skip reason = %q", s.Reason)
			}
		}
	}
	if !foundSkip {
		t.Errorf("expected one-off-job in skipped list, got: %+v", r.Skipped)
	}
}

// TestStuckJobs_SkipsHappyJobPod — running pod must not trigger anything.
func TestStuckJobs_SkipsHappyJobPod(t *testing.T) {
	src := loadSrc(t, map[string]string{
		"pods.json":    podsAndJobsForFixer,
		"job.json":     cronOwnedJob,
		"cronjob.json": activeCronJob,
	})
	m := newFakeMutator()
	StuckJobsWithBadSecretRef{}.Run(context.Background(), src, m)

	for _, c := range m.calls {
		if c == "Delete jobs/batch/nightly-cleanup" {
			t.Errorf("happy job's Job should not be deleted")
		}
	}
}

// TestStuckJobs_DedupesAcrossSiblingPods — multiple pods of same Job → one delete.
func TestStuckJobs_DedupesAcrossSiblingPods(t *testing.T) {
	multiPod := `{
  "apiVersion": "v1", "kind": "PodList",
  "items": [
    {
      "apiVersion": "v1", "kind": "Pod",
      "metadata": {
        "name": "pod-a", "namespace": "gpu-monitor",
        "ownerReferences": [{"kind": "Job", "name": "gpu-docker-monitor-29622780"}]
      },
      "status": {"containerStatuses": [{"state": {"waiting": {"message": "couldn't find key X in Secret y/z"}}}]}
    },
    {
      "apiVersion": "v1", "kind": "Pod",
      "metadata": {
        "name": "pod-b", "namespace": "gpu-monitor",
        "ownerReferences": [{"kind": "Job", "name": "gpu-docker-monitor-29622780"}]
      },
      "status": {"containerStatuses": [{"state": {"waiting": {"message": "couldn't find key X in Secret y/z"}}}]}
    }
  ]
}`
	src := loadSrc(t, map[string]string{
		"pods.json":    multiPod,
		"job.json":     cronOwnedJob,
		"cronjob.json": activeCronJob,
	})
	m := newFakeMutator()
	r := StuckJobsWithBadSecretRef{}.Run(context.Background(), src, m)

	if got, want := len(r.Actions), 1; got != want {
		t.Errorf("expected exactly 1 deduped Action, got %d (%+v)", got, r.Actions)
	}
	if got := len(m.calls); got != 1 {
		t.Errorf("expected 1 mutator Delete call, got %d (%v)", got, m.calls)
	}
}

// TestStuckJobs_ProtectedNamespace — kube-system stuck pod must not trigger Job delete.
func TestStuckJobs_ProtectedNamespace(t *testing.T) {
	protectedPod := `{
  "apiVersion": "v1", "kind": "PodList",
  "items": [{
    "apiVersion": "v1", "kind": "Pod",
    "metadata": {
      "name": "stuck", "namespace": "kube-system",
      "ownerReferences": [{"kind": "Job", "name": "kube-system-job"}]
    },
    "status": {"containerStatuses": [{"state": {"waiting": {"message": "couldn't find key X in Secret y/z"}}}]}
  }]
}`
	src := loadSrc(t, map[string]string{"pods.json": protectedPod})
	m := newFakeMutator()
	r := StuckJobsWithBadSecretRef{}.Run(context.Background(), src, m)

	if len(m.calls) != 0 {
		t.Errorf("kube-system Job should NOT have been deleted, got: %v", m.calls)
	}
	foundProtected := false
	for _, s := range r.Skipped {
		if s.Reason == "protected namespace" {
			foundProtected = true
		}
	}
	if !foundProtected {
		t.Errorf("expected protected-namespace skip entry, got: %+v", r.Skipped)
	}
}

// suspendedCronJob is the parent CronJob marked spec.suspend=true. An
// operator deliberately froze it; CHA must not restart the cycle by
// deleting the broken Job.
const suspendedCronJob = `{
  "apiVersion": "batch/v1", "kind": "CronJob",
  "metadata": {"name": "gpu-docker-monitor", "namespace": "gpu-monitor"},
  "spec": {"suspend": true, "schedule": "*/5 * * * *"}
}`

// activeCronJob is the parent CronJob in normal state — used to satisfy
// the GetCronJob call in tests that need the CronJob to exist but not be
// suspended.
const activeCronJob = `{
  "apiVersion": "batch/v1", "kind": "CronJob",
  "metadata": {"name": "gpu-docker-monitor", "namespace": "gpu-monitor"},
  "spec": {"suspend": false, "schedule": "*/5 * * * *"}
}`

// argoManagedCronJob — same CronJob, but reconciled by ArgoCD. Deleting
// the child Job would prompt a respawn whose template Argo controls; the
// Argo-vs-CHA fight loop the GitOps guard exists to prevent.
const argoManagedCronJob = `{
  "apiVersion": "batch/v1", "kind": "CronJob",
  "metadata": {
    "name": "gpu-docker-monitor", "namespace": "gpu-monitor",
    "annotations": {"argocd.argoproj.io/instance": "infra-apps"}
  },
  "spec": {"schedule": "*/5 * * * *"}
}`

func TestStuckJobs_SkipsSuspendedCronJob(t *testing.T) {
	src := loadSrc(t, map[string]string{
		"pods.json":    podsAndJobsForFixer,
		"job.json":     cronOwnedJob,
		"cronjob.json": suspendedCronJob,
	})
	m := newFakeMutator()
	r := StuckJobsWithBadSecretRef{}.Run(context.Background(), src, m)

	for _, c := range m.calls {
		if c == "Delete jobs/gpu-monitor/gpu-docker-monitor-29622780" {
			t.Errorf("suspended CronJob's child Job should NOT have been deleted")
		}
	}
	foundSuspend := false
	for _, s := range r.Skipped {
		if s.Object == "Job/gpu-monitor/gpu-docker-monitor-29622780" {
			if containsSubstr(s.Reason, "suspend") {
				foundSuspend = true
			}
		}
	}
	if !foundSuspend {
		t.Errorf("expected 'suspended' skip reason on the stuck Job; got: %+v", r.Skipped)
	}
}

func TestStuckJobs_SkipsArgoManagedCronJob(t *testing.T) {
	src := loadSrc(t, map[string]string{
		"pods.json":    podsAndJobsForFixer,
		"job.json":     cronOwnedJob,
		"cronjob.json": argoManagedCronJob,
	})
	m := newFakeMutator()
	r := StuckJobsWithBadSecretRef{}.Run(context.Background(), src, m)

	for _, c := range m.calls {
		if c == "Delete jobs/gpu-monitor/gpu-docker-monitor-29622780" {
			t.Errorf("ArgoCD-managed CronJob's child Job should NOT have been deleted")
		}
	}
	foundGitOps := false
	for _, s := range r.Skipped {
		if s.Object == "Job/gpu-monitor/gpu-docker-monitor-29622780" &&
			containsSubstr(s.Reason, "GitOps-managed") &&
			containsSubstr(s.Reason, "argocd") {
			foundGitOps = true
		}
	}
	if !foundGitOps {
		t.Errorf("expected GitOps-managed skip naming argocd; got: %+v", r.Skipped)
	}
}

// TestStuckJobs_DeletesWhenCronJobActive — make sure the new safety gates
// don't break the happy path when the CronJob exists and is healthy.
func TestStuckJobs_DeletesWhenCronJobActive(t *testing.T) {
	src := loadSrc(t, map[string]string{
		"pods.json":    podsAndJobsForFixer,
		"job.json":     cronOwnedJob,
		"cronjob.json": activeCronJob,
	})
	m := newFakeMutator()
	r := StuckJobsWithBadSecretRef{}.Run(context.Background(), src, m)

	wantCall := "Delete jobs/gpu-monitor/gpu-docker-monitor-29622780"
	found := false
	for _, c := range m.calls {
		if c == wantCall {
			found = true
		}
	}
	if !found {
		t.Errorf("active CronJob's stuck Job SHOULD have been deleted; calls=%v actions=%+v",
			m.calls, r.Actions)
	}
}

// containsSubstr is a tiny indexOf-style helper to keep tests free of
// strings imports already used by gitops_test.go (re-using the helper
// declared there to avoid duplicating it across the package).
func containsSubstr(s, sub string) bool {
	return contains(s, sub)
}
