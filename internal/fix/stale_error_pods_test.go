// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package fix

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/srenix-ai/agentic-sre/internal/snapshot"
)

func loadSrc(t *testing.T, files map[string]string) snapshot.Source {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	src, err := snapshot.LoadFile(dir)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	return src
}

const podsForFixer = `{
  "apiVersion": "v1",
  "kind": "PodList",
  "items": [
    {
      "apiVersion": "v1", "kind": "Pod",
      "metadata": {"name": "stale-debug", "namespace": "demo"},
      "status": {"phase": "Failed"}
    },
    {
      "apiVersion": "v1", "kind": "Pod",
      "metadata": {
        "name": "job-pod-1",
        "namespace": "batch",
        "ownerReferences": [{"kind": "Job", "name": "nightly-cleanup"}]
      },
      "status": {"phase": "Failed"}
    },
    {
      "apiVersion": "v1", "kind": "Pod",
      "metadata": {
        "name": "deploy-pod-1",
        "namespace": "demo",
        "ownerReferences": [{"kind": "ReplicaSet", "name": "frontend-rs-abc"}]
      },
      "status": {"phase": "Failed"}
    },
    {
      "apiVersion": "v1", "kind": "Pod",
      "metadata": {
        "name": "kube-system-failed",
        "namespace": "kube-system"
      },
      "status": {"phase": "Failed"}
    },
    {
      "apiVersion": "v1", "kind": "Pod",
      "metadata": {"name": "happy", "namespace": "demo"},
      "status": {"phase": "Running"}
    }
  ]
}`

// TestStaleErrorPods_RefusesOnSnapshot — type-system gate: nil Mutator → no I/O.
func TestStaleErrorPods_RefusesOnSnapshot(t *testing.T) {
	src := loadSrc(t, map[string]string{"pods.json": podsForFixer})
	r := StaleErrorPods{}.Run(context.Background(), src, nil)
	if r.Refused == "" {
		t.Errorf("expected Refused to be set when Mutator is nil")
	}
	if len(r.Actions) != 0 {
		t.Errorf("expected no actions when refused, got %d", len(r.Actions))
	}
}

// TestStaleErrorPods_DeletesUnownedAndJobOwned — happy-path delete set.
func TestStaleErrorPods_DeletesUnownedAndJobOwned(t *testing.T) {
	src := loadSrc(t, map[string]string{"pods.json": podsForFixer})
	m := newFakeMutator()
	r := StaleErrorPods{}.Run(context.Background(), src, m)

	if r.Refused != "" {
		t.Errorf("did not expect Refused: %q", r.Refused)
	}
	if got, want := len(r.Actions), 2; got != want {
		t.Fatalf("Actions = %d, want %d (full: %+v)", got, want, r.Actions)
	}
	wantCalls := []string{
		"Delete pods/batch/job-pod-1",
		"Delete pods/demo/stale-debug",
	}
	if got := m.sortedCalls(); !equalStrings(got, wantCalls) {
		t.Errorf("calls = %v, want %v", got, wantCalls)
	}
}

// TestStaleErrorPods_SkipsControllerOwned — RS-owned pods left to controller.
func TestStaleErrorPods_SkipsControllerOwned(t *testing.T) {
	src := loadSrc(t, map[string]string{"pods.json": podsForFixer})
	m := newFakeMutator()
	r := StaleErrorPods{}.Run(context.Background(), src, m)

	foundRS := false
	for _, s := range r.Skipped {
		if s.Object == "Pod/demo/deploy-pod-1" {
			foundRS = true
			if s.Reason != "owned by ReplicaSet (controller will recover)" {
				t.Errorf("RS skip reason = %q", s.Reason)
			}
		}
	}
	if !foundRS {
		t.Errorf("expected Pod/demo/deploy-pod-1 in skipped list, got: %+v", r.Skipped)
	}
}

// TestStaleErrorPods_ProtectedNamespace — kube-system Failed pod left alone.
func TestStaleErrorPods_ProtectedNamespace(t *testing.T) {
	src := loadSrc(t, map[string]string{"pods.json": podsForFixer})
	m := newFakeMutator()
	r := StaleErrorPods{}.Run(context.Background(), src, m)

	for _, c := range m.calls {
		if c == "Delete pods/kube-system/kube-system-failed" {
			t.Errorf("kube-system pod should NOT have been deleted")
		}
	}
	foundProtected := false
	for _, s := range r.Skipped {
		if s.Object == "Pod/kube-system/kube-system-failed" && s.Reason == "protected namespace" {
			foundProtected = true
		}
	}
	if !foundProtected {
		t.Errorf("expected protected-namespace skip entry, got: %+v", r.Skipped)
	}
}

// TestStaleErrorPods_DeleteError — surfaces the error as a skip reason.
func TestStaleErrorPods_DeleteError(t *testing.T) {
	src := loadSrc(t, map[string]string{"pods.json": podsForFixer})
	m := newFakeMutator()
	m.returnErr["Delete pods/demo/stale-debug"] = errors.New("forbidden")

	r := StaleErrorPods{}.Run(context.Background(), src, m)

	foundFail := false
	for _, s := range r.Skipped {
		if s.Object == "Pod/demo/stale-debug" && s.Reason == "delete failed: forbidden" {
			foundFail = true
		}
	}
	if !foundFail {
		t.Errorf("expected delete-failure skip entry; skipped=%+v", r.Skipped)
	}
	// Job-owned pod should still have been deleted.
	if got, want := len(r.Actions), 1; got != want {
		t.Errorf("Actions = %d, want %d", got, want)
	}
}

// TestStaleErrorPods_NothingToDo — empty cluster, Mutator must not be called.
func TestStaleErrorPods_NothingToDo(t *testing.T) {
	src := loadSrc(t, map[string]string{})
	m := newFakeMutator()
	r := StaleErrorPods{}.Run(context.Background(), src, m)
	if len(r.Actions) != 0 {
		t.Errorf("expected 0 actions on empty cluster, got %d", len(r.Actions))
	}
	if len(m.calls) != 0 {
		t.Errorf("expected 0 mutator calls, got %v", m.calls)
	}
}

// podsWithGitOpsOwnedJob holds a Failed pod whose owning Job is reconciled
// by ArgoCD. Deleting the Failed pod could re-trigger Argo's reconciliation
// loop or hide a real failure the GitOps controller would surface.
const podsWithGitOpsOwnedJob = `{
  "apiVersion": "v1", "kind": "PodList",
  "items": [{
    "apiVersion": "v1", "kind": "Pod",
    "metadata": {
      "name": "argo-job-pod",
      "namespace": "ci",
      "ownerReferences": [{"kind": "Job", "name": "argo-pipeline-build"}]
    },
    "status": {"phase": "Failed"}
  }]
}`

const argoOwnedJob = `{
  "apiVersion": "batch/v1", "kind": "Job",
  "metadata": {
    "name": "argo-pipeline-build",
    "namespace": "ci",
    "annotations": {"argocd.argoproj.io/instance": "ci-pipelines"}
  }
}`

const fluxOwnedJob = `{
  "apiVersion": "batch/v1", "kind": "Job",
  "metadata": {
    "name": "argo-pipeline-build",
    "namespace": "ci",
    "labels": {"app.kubernetes.io/managed-by": "Helm"}
  }
}`

const podDirectGitOpsAnnotated = `{
  "apiVersion": "v1", "kind": "PodList",
  "items": [{
    "apiVersion": "v1", "kind": "Pod",
    "metadata": {
      "name": "argo-direct-pod",
      "namespace": "demo",
      "annotations": {"argocd.argoproj.io/instance": "ci-debug"}
    },
    "status": {"phase": "Failed"}
  }]
}`

func TestStaleErrorPods_SkipsArgoOwnedJob(t *testing.T) {
	src := loadSrc(t, map[string]string{
		"pods.json": podsWithGitOpsOwnedJob,
		"job.json":  argoOwnedJob,
	})
	m := newFakeMutator()
	r := StaleErrorPods{}.Run(context.Background(), src, m)

	for _, c := range m.calls {
		if c == "Delete pods/ci/argo-job-pod" {
			t.Errorf("Failed pod owned by Argo-managed Job must NOT be deleted")
		}
	}
	foundGitOps := false
	for _, s := range r.Skipped {
		if s.Object == "Pod/ci/argo-job-pod" &&
			contains(s.Reason, "GitOps-managed") &&
			contains(s.Reason, "argocd") {
			foundGitOps = true
		}
	}
	if !foundGitOps {
		t.Errorf("expected GitOps-managed skip naming argocd; got: %+v", r.Skipped)
	}
}

func TestStaleErrorPods_SkipsHelmOwnedJob(t *testing.T) {
	src := loadSrc(t, map[string]string{
		"pods.json": podsWithGitOpsOwnedJob,
		"job.json":  fluxOwnedJob,
	})
	m := newFakeMutator()
	r := StaleErrorPods{}.Run(context.Background(), src, m)

	for _, c := range m.calls {
		if c == "Delete pods/ci/argo-job-pod" {
			t.Errorf("Failed pod owned by Helm-managed Job must NOT be deleted")
		}
	}
	foundGitOps := false
	for _, s := range r.Skipped {
		if s.Object == "Pod/ci/argo-job-pod" && contains(s.Reason, "GitOps-managed") {
			foundGitOps = true
		}
	}
	if !foundGitOps {
		t.Errorf("expected GitOps-managed skip; got: %+v", r.Skipped)
	}
}

func TestStaleErrorPods_SkipsPodWithDirectGitOpsAnnotation(t *testing.T) {
	src := loadSrc(t, map[string]string{"pods.json": podDirectGitOpsAnnotated})
	m := newFakeMutator()
	r := StaleErrorPods{}.Run(context.Background(), src, m)

	for _, c := range m.calls {
		if c == "Delete pods/demo/argo-direct-pod" {
			t.Errorf("Failed pod with argocd annotation must NOT be deleted directly")
		}
	}
	foundGitOps := false
	for _, s := range r.Skipped {
		if s.Object == "Pod/demo/argo-direct-pod" && contains(s.Reason, "GitOps-managed") {
			foundGitOps = true
		}
	}
	if !foundGitOps {
		t.Errorf("expected GitOps-managed skip; got: %+v", r.Skipped)
	}
}

// TestStaleErrorPods_DeletesWhenOwnerJobMissingFromSnapshot — defensive:
// if the owning Job isn't in the captured snapshot, we cannot evaluate
// GitOps annotations on it. The fixer must err on the side of cleanup
// (Failed pods whose Jobs are gone are garbage), not on the side of
// over-blocking. This guard makes sure the GitOps logic doesn't refuse
// when it should proceed.
func TestStaleErrorPods_DeletesWhenOwnerJobMissingFromSnapshot(t *testing.T) {
	// Pod owned by Job "ghost-job" that we deliberately don't include.
	pods := `{
  "apiVersion": "v1", "kind": "PodList",
  "items": [{
    "apiVersion": "v1", "kind": "Pod",
    "metadata": {
      "name": "orphan-job-pod", "namespace": "batch",
      "ownerReferences": [{"kind": "Job", "name": "ghost-job"}]
    },
    "status": {"phase": "Failed"}
  }]
}`
	src := loadSrc(t, map[string]string{"pods.json": pods})
	m := newFakeMutator()
	r := StaleErrorPods{}.Run(context.Background(), src, m)

	found := false
	for _, c := range m.calls {
		if c == "Delete pods/batch/orphan-job-pod" {
			found = true
		}
	}
	if !found {
		t.Errorf("orphan Failed pod SHOULD be deletable when its owning Job is not in the snapshot; calls=%v actions=%+v",
			m.calls, r.Actions)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
