// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package fix

import (
	"context"
	"strings"
	"testing"
)

const stuckRSPod = `{
  "apiVersion": "v1", "kind": "PodList",
  "items": [{
    "apiVersion": "v1", "kind": "Pod",
    "metadata": {
      "name": "frontend-old-rs-abc",
      "namespace": "demo",
      "ownerReferences": [{"kind": "ReplicaSet", "name": "frontend-old-rs"}]
    },
    "status": {
      "containerStatuses": [{
        "name": "web",
        "state": {"waiting": {
          "reason": "CreateContainerConfigError",
          "message": "transient: ImagePull error during init"
        }}
      }]
    }
  }]
}`

const stuckRSPodMissingKey = `{
  "apiVersion": "v1", "kind": "PodList",
  "items": [{
    "apiVersion": "v1", "kind": "Pod",
    "metadata": {
      "name": "broken-deploy-pod",
      "namespace": "demo",
      "ownerReferences": [{"kind": "ReplicaSet", "name": "broken-rs"}]
    },
    "status": {
      "containerStatuses": [{
        "name": "x",
        "state": {"waiting": {
          "reason": "CreateContainerConfigError",
          "message": "couldn't find key X in Secret demo/y"
        }}
      }]
    }
  }]
}`

const oldRS = `{
  "apiVersion": "apps/v1", "kind": "ReplicaSet",
  "metadata": {
    "name": "frontend-old-rs",
    "namespace": "demo",
    "annotations": {"deployment.kubernetes.io/revision": "5"},
    "ownerReferences": [{"kind": "Deployment", "name": "frontend"}]
  }
}`

const liveDeployment = `{
  "apiVersion": "apps/v1", "kind": "Deployment",
  "metadata": {
    "name": "frontend",
    "namespace": "demo",
    "annotations": {"deployment.kubernetes.io/revision": "7"}
  }
}`

const stuckDeploymentSameRev = `{
  "apiVersion": "apps/v1", "kind": "Deployment",
  "metadata": {
    "name": "frontend",
    "namespace": "demo",
    "annotations": {"deployment.kubernetes.io/revision": "5"}
  }
}`

// TestStuckRSPods_RefusesOnSnapshot — type-system gate.
func TestStuckRSPods_RefusesOnSnapshot(t *testing.T) {
	src := loadSrc(t, map[string]string{"pods.json": stuckRSPod, "rs.json": oldRS, "deploy.json": liveDeployment})
	r := StuckRSPods{}.Run(context.Background(), src, nil)
	if r.Refused == "" {
		t.Errorf("expected Refused on snapshot mode")
	}
}

// TestStuckRSPods_RestartsWhenRevisionMismatch — happy path.
func TestStuckRSPods_RestartsWhenRevisionMismatch(t *testing.T) {
	src := loadSrc(t, map[string]string{"pods.json": stuckRSPod, "rs.json": oldRS, "deploy.json": liveDeployment})
	m := newFakeMutator()
	r := StuckRSPods{}.Run(context.Background(), src, m)

	if got, want := len(r.Actions), 1; got != want {
		t.Fatalf("Actions = %d, want %d (full: %+v)", got, want, r.Actions)
	}
	if !strings.Contains(r.Actions[0].Description, "stuck RS rev 5, live rev 7") {
		t.Errorf("Action text doesn't name revisions: %q", r.Actions[0].Description)
	}
	wantPrefix := "Patch deployments/demo/frontend"
	if len(m.calls) != 1 || !strings.HasPrefix(m.calls[0], wantPrefix) {
		t.Errorf("calls = %v, want one starting with %q", m.calls, wantPrefix)
	}
}

// TestStuckRSPods_SkipsWhenRevsMatch — same rev → don't restart, surface as skip.
func TestStuckRSPods_SkipsWhenRevsMatch(t *testing.T) {
	src := loadSrc(t, map[string]string{"pods.json": stuckRSPod, "rs.json": oldRS, "deploy.json": stuckDeploymentSameRev})
	m := newFakeMutator()
	r := StuckRSPods{}.Run(context.Background(), src, m)

	if len(r.Actions) != 0 {
		t.Errorf("expected 0 actions when revisions match, got %d", len(r.Actions))
	}
	if len(m.calls) != 0 {
		t.Errorf("expected 0 mutator calls, got %v", m.calls)
	}
	foundSkip := false
	for _, s := range r.Skipped {
		if strings.Contains(s.Reason, "rollout would reproduce") {
			foundSkip = true
		}
	}
	if !foundSkip {
		t.Errorf("expected revision-match skip reason; got: %+v", r.Skipped)
	}
}

// TestStuckRSPods_SkipsMissingSecretKey — must defer to diagnose analyzer.
func TestStuckRSPods_SkipsMissingSecretKey(t *testing.T) {
	src := loadSrc(t, map[string]string{"pods.json": stuckRSPodMissingKey})
	m := newFakeMutator()
	r := StuckRSPods{}.Run(context.Background(), src, m)

	if len(m.calls) != 0 {
		t.Errorf("must NOT patch when failure is couldn't-find-key, got: %v", m.calls)
	}
	foundSkip := false
	for _, s := range r.Skipped {
		if strings.Contains(s.Reason, "couldn't find key") {
			foundSkip = true
		}
	}
	if !foundSkip {
		t.Errorf("expected missing-key skip reason; got: %+v", r.Skipped)
	}
}

// TestStuckRSPods_DedupesAcrossSiblingPods — multiple stuck pods of same Deploy → one patch.
func TestStuckRSPods_DedupesAcrossSiblingPods(t *testing.T) {
	multi := strings.Replace(stuckRSPod, `"items": [{`, `"items": [{
    "apiVersion": "v1", "kind": "Pod",
    "metadata": {
      "name": "frontend-old-rs-xyz",
      "namespace": "demo",
      "ownerReferences": [{"kind": "ReplicaSet", "name": "frontend-old-rs"}]
    },
    "status": {
      "containerStatuses": [{
        "name": "web",
        "state": {"waiting": {"reason": "CreateContainerConfigError", "message": "image pull"}}
      }]
    }
  },{`, 1)
	src := loadSrc(t, map[string]string{"pods.json": multi, "rs.json": oldRS, "deploy.json": liveDeployment})
	m := newFakeMutator()
	r := StuckRSPods{}.Run(context.Background(), src, m)

	if got, want := len(r.Actions), 1; got != want {
		t.Errorf("expected 1 deduped Action, got %d (%+v)", got, r.Actions)
	}
	if got := len(m.calls); got != 1 {
		t.Errorf("expected 1 patch call, got %d (%v)", got, m.calls)
	}
}

// TestStuckRSPods_ProtectedNamespace — kube-system pod must not trigger anything.
func TestStuckRSPods_ProtectedNamespace(t *testing.T) {
	protected := strings.Replace(stuckRSPod, `"namespace": "demo"`, `"namespace": "kube-system"`, 1)
	src := loadSrc(t, map[string]string{"pods.json": protected})
	m := newFakeMutator()
	r := StuckRSPods{}.Run(context.Background(), src, m)
	if len(m.calls) != 0 {
		t.Errorf("kube-system Deployment should NOT be patched: %v", m.calls)
	}
	foundProtected := false
	for _, s := range r.Skipped {
		if s.Reason == "protected namespace" {
			foundProtected = true
		}
	}
	if !foundProtected {
		t.Errorf("expected protected-namespace skip; got: %+v", r.Skipped)
	}
}
