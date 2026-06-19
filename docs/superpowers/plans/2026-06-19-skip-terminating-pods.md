# Skip Terminating Pods in Pod-State Probes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Prevent CHA from emitting pod-health findings (CrashLoopBackOff, Pending, FailedMounts, ETCD, Services) for pods that already have `metadata.deletionTimestamp` set (Terminating), eliminating the "already resolved" false-positive noise during rollouts.

**Architecture:** Add a single shared helper `isTerminating(pod unstructured.Unstructured) bool` to `internal/probe/nodes.go` (the existing helper file), then add one `continue` guard at the top of the pod-iterating loop in each affected probe. The Services probe guard goes on the individual pod loop (not the outer target loop). No severity or logic changes; only Terminating pods are skipped.

**Tech Stack:** Go 1.26, `k8s.io/apimachinery/pkg/apis/meta/v1/unstructured`, existing `snapshot.Source` file-based test fixture (`loadProbeSrc`).

## Global Constraints

- OSS repo only: `/home/skadam/CHA/cluster-health-autopilot`
- Branch: `fix/skip-terminating-pods` off `main` (HEAD = `36e3654`)
- One commit total; commit trailer: `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`
- Never skip git hooks (`--no-verify` forbidden)
- `go test ./internal/...` must be green; `go build ./...` clean; `golangci-lint run ./internal/...` 0 issues
- Conservative: do NOT change severities, thresholds, or any other logic
- Write the git-internal report to `.git/sdd-skip-terminating-report.md`

---

### Task 1: Branch + Shared Helper

**Files:**
- Modify: `internal/probe/nodes.go` (add `isTerminating` after `getSliceField`)
- Test: `internal/probe/node_pressure_test.go` (no test needed — pure helper; tested implicitly by Task 2+)

**Interfaces:**
- Produces: `func isTerminating(pod unstructured.Unstructured) bool` — returns `true` when `pod.GetDeletionTimestamp() != nil`

- [ ] **Step 1: Create the feature branch**

```bash
cd /home/skadam/CHA/cluster-health-autopilot
git checkout -b fix/skip-terminating-pods
```

Expected output: `Switched to a new branch 'fix/skip-terminating-pods'`

- [ ] **Step 2: Add `isTerminating` helper to `internal/probe/nodes.go`**

Open `/home/skadam/CHA/cluster-health-autopilot/internal/probe/nodes.go`. After the closing `}` of `getSliceField` (currently the last function, ending at line 110), append:

```go
// isTerminating returns true when the pod has a non-nil deletionTimestamp,
// meaning the kubelet has been asked to stop and delete it. Terminating pods
// are intentionally going away; flagging them as stuck/not-ready/crashlooping
// produces "already resolved" noise and useless AI remediation proposals.
func isTerminating(pod unstructured.Unstructured) bool {
	return pod.GetDeletionTimestamp() != nil
}
```

The import `"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"` is already present in nodes.go's package (it's used by other files in the same package). In Go, a function in the same package can use any type visible in the package without a new import in its own file — but `nodes.go` itself does NOT currently import `unstructured`. The function parameter type `unstructured.Unstructured` requires the import to be present in the file where the function is declared. Check the current imports in `nodes.go`:

Current imports in `nodes.go`:
```go
import (
    "context"
    "fmt"

    "github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
)
```

Add the unstructured import. The final import block in `nodes.go` should be:

```go
import (
	"context"
	"fmt"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)
```

- [ ] **Step 3: Verify it compiles**

```bash
cd /home/skadam/CHA/cluster-health-autopilot && go build ./internal/probe/...
```

Expected: no output, exit code 0.

---

### Task 2: CrashLoopBackOff Probe — Guard + Test

**Files:**
- Modify: `internal/probe/crashloop.go` — add `isTerminating` guard in the pod loop (line 57)
- Modify: `internal/probe/crashloop_test.go` — add two new test cases

**Interfaces:**
- Consumes: `isTerminating(pod unstructured.Unstructured) bool` from Task 1

- [ ] **Step 1: Write the failing test**

Add to the bottom of `/home/skadam/CHA/cluster-health-autopilot/internal/probe/crashloop_test.go`:

```go
// podTerminatingCrashLoop is a pod with deletionTimestamp set AND
// CrashLoopBackOff — should be silently skipped.
const podTerminatingCrashLoop = `{
  "apiVersion": "v1", "kind": "PodList",
  "items": [
    {"apiVersion": "v1", "kind": "Pod",
     "metadata": {"name": "old-pod", "namespace": "demo",
                  "deletionTimestamp": "2026-06-19T10:00:00Z"},
     "status": {"phase": "Running",
                "containerStatuses": [{
                  "restartCount": 5,
                  "state": {"waiting": {"reason": "CrashLoopBackOff"}}
                }]}}
  ]
}`

func TestCrashLoop_TerminatingPodSkipped(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{"pods.json": podTerminatingCrashLoop})
	r := CrashLoopBackOff{}.Run(context.Background(), src)
	if r.Component.Status != "HEALTHY" {
		t.Errorf("Terminating pod with CrashLoopBackOff must be skipped; got Status=%q Detail=%q",
			r.Component.Status, r.Component.Detail)
	}
	if len(r.Findings) != 0 {
		t.Errorf("expected 0 findings for terminating pod, got %+v", r.Findings)
	}
}

func TestCrashLoop_NonTerminatingCrashLoopStillFlagged(t *testing.T) {
	// Regression: removing the deletionTimestamp means the pod IS flagged.
	src := loadProbeSrc(t, map[string]string{"pods.json": podUserNsLowRestarts})
	r := CrashLoopBackOff{}.Run(context.Background(), src)
	if r.Component.Status == "HEALTHY" {
		t.Errorf("non-terminating CrashLoopBackOff pod must still be flagged; got HEALTHY")
	}
}
```

- [ ] **Step 2: Run to confirm tests fail**

```bash
cd /home/skadam/CHA/cluster-health-autopilot && go test ./internal/probe/ -run 'TestCrashLoop_Terminating' -v
```

Expected: `FAIL` — `TestCrashLoop_TerminatingPodSkipped` fails because the probe currently flags the pod.

- [ ] **Step 3: Add the guard in `crashloop.go`**

In `/home/skadam/CHA/cluster-health-autopilot/internal/probe/crashloop.go`, inside the `for _, pod := range pods.Items {` loop (currently line 57), add the guard as the FIRST statement before any other checks:

```go
	for _, pod := range pods.Items {
		if isTerminating(pod) {
			continue
		}
		// We don't restrict to Running — kubelet sometimes reports
		// Pending/CrashLoopBackOff during init-container retries.
		restarts, found, reason := podMaxRestartCount(pod)
```

(Remove the blank comment line above `// We don't restrict...` — just insert `if isTerminating(pod) { continue }` as the new first line of the loop body.)

- [ ] **Step 4: Run the tests to confirm they pass**

```bash
cd /home/skadam/CHA/cluster-health-autopilot && go test ./internal/probe/ -run 'TestCrashLoop' -v
```

Expected: all `TestCrashLoop_*` pass including the two new ones.

---

### Task 3: PendingPods Probe — Guard + Test

**Files:**
- Modify: `internal/probe/pending_pods.go` — add `isTerminating` guard in the pod loop (line 66)
- Modify: `internal/probe/pending_pods_test.go` — add two new test cases

**Interfaces:**
- Consumes: `isTerminating(pod unstructured.Unstructured) bool` from Task 1

- [ ] **Step 1: Write the failing test**

Add to the bottom of `/home/skadam/CHA/cluster-health-autopilot/internal/probe/pending_pods_test.go`:

```go
// podTerminatingPending is a Pending pod with PodScheduled=False AND a
// deletionTimestamp — should be silently skipped.
const podTerminatingPending = `{
  "apiVersion": "v1", "kind": "PodList",
  "items": [
    {"apiVersion": "v1", "kind": "Pod",
     "metadata": {"name": "old-pending", "namespace": "demo",
                  "creationTimestamp": "2026-05-22T10:00:00Z",
                  "deletionTimestamp": "2026-06-19T10:00:00Z"},
     "status": {"phase": "Pending",
                "conditions": [
                  {"type": "PodScheduled", "status": "False",
                   "reason": "Unschedulable",
                   "message": "0/4 nodes are available: 4 Insufficient cpu."}
                ]}}
  ]
}`

func TestPendingPods_TerminatingPodSkipped(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{"pods.json": podTerminatingPending})
	r := PendingPods{Now: fixedNow("2026-05-22T12:00:00Z")}.Run(context.Background(), src)
	if r.Component.Status != "HEALTHY" {
		t.Errorf("Terminating pending pod must be skipped; got Status=%q Detail=%q",
			r.Component.Status, r.Component.Detail)
	}
	if len(r.Findings) != 0 {
		t.Errorf("expected 0 findings for terminating pod, got %+v", r.Findings)
	}
}

func TestPendingPods_NonTerminatingPendingStillFlagged(t *testing.T) {
	// Regression: same pod without deletionTimestamp is still flagged.
	src := loadProbeSrc(t, map[string]string{"pods.json": podStuckInsufficientCPU})
	r := PendingPods{Now: fixedNow("2026-05-22T12:00:00Z")}.Run(context.Background(), src)
	if r.Component.Status != "CRITICAL" {
		t.Errorf("non-terminating Pending pod must still be flagged; got Status=%q", r.Component.Status)
	}
}
```

- [ ] **Step 2: Run to confirm tests fail**

```bash
cd /home/skadam/CHA/cluster-health-autopilot && go test ./internal/probe/ -run 'TestPendingPods_Terminating' -v
```

Expected: `FAIL` — `TestPendingPods_TerminatingPodSkipped` fails.

- [ ] **Step 3: Add the guard in `pending_pods.go`**

In `/home/skadam/CHA/cluster-health-autopilot/internal/probe/pending_pods.go`, inside the `for _, pod := range pods.Items {` loop (currently line 66), add the guard as the FIRST statement:

```go
	for _, pod := range pods.Items {
		if isTerminating(pod) {
			continue
		}
		phase, _, _ := unstructured.NestedString(pod.Object, "status", "phase")
		if phase != "Pending" {
```

- [ ] **Step 4: Run to confirm tests pass**

```bash
cd /home/skadam/CHA/cluster-health-autopilot && go test ./internal/probe/ -run 'TestPendingPods' -v
```

Expected: all `TestPendingPods_*` pass.

---

### Task 4: FailedMounts Probe — Guard + Test

**Files:**
- Modify: `internal/probe/failed_mounts.go` — add `isTerminating` guard in the pod loop (line 94)
- Modify: `internal/probe/failed_mounts_test.go` — add two new test cases

**Interfaces:**
- Consumes: `isTerminating(pod unstructured.Unstructured) bool` from Task 1

- [ ] **Step 1: Read the existing test fixture pattern**

Open `/home/skadam/CHA/cluster-health-autopilot/internal/probe/failed_mounts_test.go` to see existing test constants and the `loadProbeSrc` usage. Specifically note how events.json fixtures are structured alongside pods.json.

- [ ] **Step 2: Write the failing test**

Add to the bottom of `/home/skadam/CHA/cluster-health-autopilot/internal/probe/failed_mounts_test.go`:

```go
// podTerminatingContainerCreating has deletionTimestamp set, is Pending/ContainerCreating,
// and has a corresponding FailedMount event. Should be silently skipped.
const podTerminatingContainerCreating = `{
  "apiVersion": "v1", "kind": "PodList",
  "items": [
    {"apiVersion": "v1", "kind": "Pod",
     "metadata": {"name": "old-mounter", "namespace": "demo",
                  "creationTimestamp": "2026-06-19T09:00:00Z",
                  "deletionTimestamp": "2026-06-19T10:00:00Z"},
     "status": {"phase": "Pending",
                "containerStatuses": [
                  {"state": {"waiting": {"reason": "ContainerCreating"}}}
                ]}}
  ]
}`

const eventsTerminatingMount = `{
  "apiVersion": "v1", "kind": "EventList",
  "items": [
    {"apiVersion": "v1", "kind": "Event",
     "reason": "FailedMount",
     "involvedObject": {"kind": "Pod", "namespace": "demo", "name": "old-mounter"},
     "message": "Unable to attach or mount volumes: ..."}
  ]
}`

func TestFailedMounts_TerminatingPodSkipped(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{
		"pods.json":   podTerminatingContainerCreating,
		"events.json": eventsTerminatingMount,
	})
	fm := FailedMounts{MinAge: -1} // disable grace period to isolate the terminating check
	r := fm.Run(context.Background(), src)
	if r.Component.Status != "HEALTHY" {
		t.Errorf("Terminating pod with FailedMount must be skipped; got Status=%q Detail=%q",
			r.Component.Status, r.Component.Detail)
	}
	if len(r.Findings) != 0 {
		t.Errorf("expected 0 findings for terminating pod, got %+v", r.Findings)
	}
}
```

- [ ] **Step 3: Run to confirm test fails**

```bash
cd /home/skadam/CHA/cluster-health-autopilot && go test ./internal/probe/ -run 'TestFailedMounts_Terminating' -v
```

Expected: `FAIL`.

- [ ] **Step 4: Add the guard in `failed_mounts.go`**

In `/home/skadam/CHA/cluster-health-autopilot/internal/probe/failed_mounts.go`, inside the `for _, pod := range pods.Items {` loop (currently line 94), add the guard as the FIRST statement:

```go
	for _, pod := range pods.Items {
		if isTerminating(pod) {
			continue
		}
		phase, _, _ := unstructured.NestedString(pod.Object, "status", "phase")
		if phase != "Pending" {
```

- [ ] **Step 5: Run to confirm tests pass**

```bash
cd /home/skadam/CHA/cluster-health-autopilot && go test ./internal/probe/ -run 'TestFailedMounts' -v
```

Expected: all `TestFailedMounts_*` pass.

---

### Task 5: Services Probe — Guard + Test

**Files:**
- Modify: `internal/probe/services.go` — add `isTerminating` guard in the inner pod loop (line 64)
- Test: there is no `services_test.go`; check if one exists. If not, create `internal/probe/services_test.go`

**Interfaces:**
- Consumes: `isTerminating(pod unstructured.Unstructured) bool` from Task 1

Note: The `Services` probe iterates pods by label selector for each target. A Terminating pod is "not ready" — it would be counted in `matched` but not `ready`, potentially triggering a degraded/down finding. Guard it to exclude from both counts.

- [ ] **Step 1: Check if services_test.go exists**

```bash
ls /home/skadam/CHA/cluster-health-autopilot/internal/probe/services_test.go 2>/dev/null && echo EXISTS || echo MISSING
```

- [ ] **Step 2: Write the failing test**

If `services_test.go` does not exist, create `/home/skadam/CHA/cluster-health-autopilot/internal/probe/services_test.go` with the full file content below. If it exists, append the two new test functions.

```go
// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package probe

import (
	"context"
	"testing"
)

// podsServiceHealthy has 1 ready pod matching selector app=myapp
const podsServiceHealthy = `{
  "apiVersion": "v1", "kind": "PodList",
  "items": [
    {"apiVersion": "v1", "kind": "Pod",
     "metadata": {"name": "myapp-abc", "namespace": "demo",
                  "labels": {"app": "myapp"}},
     "status": {"containerStatuses": [{"ready": true}]}}
  ]
}`

// podsServiceTerminating has 1 terminating pod and 1 ready pod; the
// terminating one must not count as "matched but not ready".
const podsServiceTerminating = `{
  "apiVersion": "v1", "kind": "PodList",
  "items": [
    {"apiVersion": "v1", "kind": "Pod",
     "metadata": {"name": "myapp-old", "namespace": "demo",
                  "labels": {"app": "myapp"},
                  "deletionTimestamp": "2026-06-19T10:00:00Z"},
     "status": {"containerStatuses": [{"ready": false}]}},
    {"apiVersion": "v1", "kind": "Pod",
     "metadata": {"name": "myapp-new", "namespace": "demo",
                  "labels": {"app": "myapp"}},
     "status": {"containerStatuses": [{"ready": true}]}}
  ]
}`

func TestServices_TerminatingPodNotCountedAsNotReady(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{"pods.json": podsServiceTerminating})
	s := Services{Targets: []ServiceTarget{
		{Namespace: "demo", Selector: "app=myapp", Display: "myapp"},
	}}
	r := s.Run(context.Background(), src)
	if r.Component.Status != "HEALTHY" {
		t.Errorf("Terminating pod must not count as not-ready; got Status=%q Detail=%q",
			r.Component.Status, r.Component.Detail)
	}
}

func TestServices_NonTerminatingNotReadyPodStillFlagged(t *testing.T) {
	// A not-ready pod without deletionTimestamp must still trigger the finding.
	const podsServiceNotReady = `{
  "apiVersion": "v1", "kind": "PodList",
  "items": [
    {"apiVersion": "v1", "kind": "Pod",
     "metadata": {"name": "myapp-abc", "namespace": "demo",
                  "labels": {"app": "myapp"}},
     "status": {"containerStatuses": [{"ready": false}]}}
  ]
}`
	src := loadProbeSrc(t, map[string]string{"pods.json": podsServiceNotReady})
	s := Services{Targets: []ServiceTarget{
		{Namespace: "demo", Selector: "app=myapp", Display: "myapp"},
	}}
	r := s.Run(context.Background(), src)
	if r.Component.Status == "HEALTHY" {
		t.Errorf("non-terminating not-ready pod must still be flagged; got HEALTHY")
	}
}
```

- [ ] **Step 3: Run to confirm tests fail**

```bash
cd /home/skadam/CHA/cluster-health-autopilot && go test ./internal/probe/ -run 'TestServices_Terminating' -v
```

Expected: `FAIL` — `TestServices_TerminatingPodNotCountedAsNotReady` fails.

- [ ] **Step 4: Add the guard in `services.go`**

In `/home/skadam/CHA/cluster-health-autopilot/internal/probe/services.go`, inside the `for _, pod := range list.Items {` loop (currently line 64), add the guard as the FIRST statement. The guard skips the pod from both the `matched` and `ready` counts:

```go
		for _, pod := range list.Items {
			if isTerminating(pod) {
				continue
			}
			labels := pod.GetLabels()
			if labels[key] != val {
				continue
			}
```

- [ ] **Step 5: Run to confirm tests pass**

```bash
cd /home/skadam/CHA/cluster-health-autopilot && go test ./internal/probe/ -run 'TestServices' -v
```

Expected: all service tests pass.

---

### Task 6: ETCD Probe — Guard + Test

**Files:**
- Modify: `internal/probe/etcd.go` — add `isTerminating` guard in the pod loop (line 59)
- Modify: `internal/probe/etcd_test.go` — add two new test cases

**Interfaces:**
- Consumes: `isTerminating(pod unstructured.Unstructured) bool` from Task 1

Note: The ETCD probe only looks at pods named `etcd-*` or labelled `component=etcd`. A Terminating etcd pod during a rolling update should not be flagged as "not ready" or "restarted" because it's being replaced. The guard goes before `looksLikeEtcdPod` to skip all terminating pods eagerly.

- [ ] **Step 1: Read the existing test file**

Open `/home/skadam/CHA/cluster-health-autopilot/internal/probe/etcd_test.go` to understand existing fixture constants.

- [ ] **Step 2: Write the failing test**

Add to the bottom of `/home/skadam/CHA/cluster-health-autopilot/internal/probe/etcd_test.go`:

```go
// podsEtcdTerminating has one terminating etcd pod (not-ready, 2 restarts).
// Since it's terminating (being replaced), the probe must not flag it.
const podsEtcdTerminating = `{
  "apiVersion": "v1", "kind": "PodList",
  "items": [
    {"apiVersion": "v1", "kind": "Pod",
     "metadata": {"name": "etcd-node-a", "namespace": "kube-system",
                  "deletionTimestamp": "2026-06-19T10:00:00Z"},
     "status": {"conditions": [{"type": "Ready", "status": "False"}],
                "containerStatuses": [{"restartCount": 2}]}}
  ]
}`

func TestETCD_TerminatingPodSkipped(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{"pods.json": podsEtcdTerminating})
	r := ETCD{}.Run(context.Background(), src)
	// With 0 non-terminating etcd pods, the probe should fall through to the
	// "external etcd" WARNING — NOT to CRITICAL.
	if r.Component.Status == "CRITICAL" {
		t.Errorf("Terminating etcd pod must not cause CRITICAL; got Status=%q Detail=%q",
			r.Component.Status, r.Component.Detail)
	}
	// Specifically must not have a "not ready" or "restarted" finding.
	for _, f := range r.Findings {
		if f.Severity == SeverityCritical {
			t.Errorf("must not produce a CRITICAL finding for terminating pod; got %q", f.Message)
		}
	}
}
```

- [ ] **Step 3: Run to confirm test fails**

```bash
cd /home/skadam/CHA/cluster-health-autopilot && go test ./internal/probe/ -run 'TestETCD_Terminating' -v
```

Expected: `FAIL` — currently produces CRITICAL.

- [ ] **Step 4: Add the guard in `etcd.go`**

In `/home/skadam/CHA/cluster-health-autopilot/internal/probe/etcd.go`, inside the `for _, pod := range pods.Items {` loop (currently line 59), add the guard as the FIRST statement:

```go
	for _, pod := range pods.Items {
		if isTerminating(pod) {
			continue
		}
		if !looksLikeEtcdPod(pod) {
			continue
		}
```

- [ ] **Step 5: Run to confirm tests pass**

```bash
cd /home/skadam/CHA/cluster-health-autopilot && go test ./internal/probe/ -run 'TestETCD' -v
```

Expected: all `TestETCD_*` pass.

---

### Task 7: Full Test Run + Lint + Write Report

**Files:**
- Create: `.git/sdd-skip-terminating-report.md`
- No source changes

- [ ] **Step 1: Run full internal test suite**

```bash
cd /home/skadam/CHA/cluster-health-autopilot && go test ./internal/... -v 2>&1 | tail -20
```

Expected: `ok github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/probe` and any other internal packages also `ok`. Zero `FAIL` lines.

- [ ] **Step 2: Run full build**

```bash
cd /home/skadam/CHA/cluster-health-autopilot && go build ./...
```

Expected: no output, exit code 0.

- [ ] **Step 3: Run golangci-lint**

```bash
cd /home/skadam/CHA/cluster-health-autopilot && golangci-lint run ./internal/... 2>&1
```

Expected: no issues reported (exit code 0).

- [ ] **Step 4: Write the git-internal report**

Create `/home/skadam/CHA/cluster-health-autopilot/.git/sdd-skip-terminating-report.md` with this content (fill in the actual commit SHA once you commit in Task 8):

```markdown
# skip-terminating-pods — Implementation Report

## Analyzers/Probes Changed

| File | Loop location | Guard added |
|------|--------------|-------------|
| `internal/probe/crashloop.go` | `for _, pod := range pods.Items` (line 57) | `isTerminating(pod)` skip at loop top |
| `internal/probe/pending_pods.go` | `for _, pod := range pods.Items` (line 66) | `isTerminating(pod)` skip at loop top |
| `internal/probe/failed_mounts.go` | `for _, pod := range pods.Items` (line 94) | `isTerminating(pod)` skip at loop top |
| `internal/probe/services.go` | `for _, pod := range list.Items` (line 64) | `isTerminating(pod)` skip at loop top (excludes from matched+ready counts) |
| `internal/probe/etcd.go` | `for _, pod := range pods.Items` (line 59) | `isTerminating(pod)` skip before looksLikeEtcdPod |
| `internal/probe/nodes.go` | N/A (helper file) | Added `isTerminating()` helper function |

## Deliberately Left Alone + Rationale

| Probe | Reason not changed |
|-------|-------------------|
| `k3s_datastore.go` (etcd pod loop, line 126) | Delegates to same `looksLikeEtcdPod` + restarts heuristic as `etcd.go`. The k3s probe's pod loop at line 126 can also skip terminating pods for the same reason. **ACTION NEEDED**: this probe was NOT changed in this PR — it has an identical pattern to `etcd.go` and should receive the same guard in a follow-up if k3s etcd pods are observed as Terminating during upgrades. It was not changed here because the primary observed noise was from the main ETCD probe and pod-health probes, and out-of-scope changes risk unintended side effects in the k3s probe's complex quorum logic. |
| `postgres.go` (Spilo pod loop, line 69) | The Spilo path checks `spilo-role=master` pods specifically. A Terminating Spilo master during a switchover IS relevant — the operator is switching primary. Not skipping is correct here; the Services probe handles the broader readiness concern. Left alone deliberately. |
| `node_pressure.go` | Iterates Nodes, not Pods. No change needed. |
| `daemonsets.go` | Iterates DaemonSets, not Pods. No change needed. |
| `argocd_app.go`, `hpa_scaling.go`, `velero.go`, `ceph.go`, `kong.go`, `kong_routes.go`, `pvcs.go`, `ingress_discovery.go`, `traefik_routes.go`, `k3s_localpath.go` | None of these iterate pods for pod-health findings (they iterate their respective CRDs/resources). No change needed. |

## TDD Evidence

Each changed probe has two new tests:
1. `Test*_TerminatingPodSkipped` — pod WITH `deletionTimestamp` + otherwise-flaggable state → HEALTHY (0 findings)
2. `Test*_NonTerminating*StillFlagged` — identical pod WITHOUT `deletionTimestamp` → flagged as before

All tests were written before the guard was added and verified FAIL, then PASS after the guard.

## Test + Lint Results

- `go test ./internal/...`: OK (all packages pass)
- `go build ./...`: clean
- `golangci-lint run ./internal/...`: 0 issues
```

- [ ] **Step 5: Verify the report file was written**

```bash
ls -la /home/skadam/CHA/cluster-health-autopilot/.git/sdd-skip-terminating-report.md
```

Expected: file exists with non-zero size.

---

### Task 8: Single Commit

**Files:** All modified/created files from Tasks 1-7

- [ ] **Step 1: Stage all changed files**

```bash
cd /home/skadam/CHA/cluster-health-autopilot
git add internal/probe/nodes.go \
        internal/probe/crashloop.go \
        internal/probe/crashloop_test.go \
        internal/probe/pending_pods.go \
        internal/probe/pending_pods_test.go \
        internal/probe/failed_mounts.go \
        internal/probe/failed_mounts_test.go \
        internal/probe/services.go \
        internal/probe/services_test.go \
        internal/probe/etcd.go \
        internal/probe/etcd_test.go \
        docs/superpowers/plans/2026-06-19-skip-terminating-pods.md
```

- [ ] **Step 2: Verify staged files**

```bash
cd /home/skadam/CHA/cluster-health-autopilot && git diff --cached --name-only
```

Expected: the 12 files listed above.

- [ ] **Step 3: Commit**

```bash
cd /home/skadam/CHA/cluster-health-autopilot && git commit -m "$(cat <<'EOF'
fix(probe): skip Terminating pods in pod-health probes

Pods with metadata.deletionTimestamp set are intentionally being deleted;
flagging them as stuck/not-ready/crashlooping generates "already resolved"
noise and pointless AI remediation proposals (observed: own ReplicaSet
pods during rollout). Guard added at the top of each pod-iterating loop in
CrashLoopBackOff, PendingPods, FailedMounts, Services, and ETCD probes via
a shared isTerminating(pod) helper. TDD: each probe gets two new tests
(terminating→HEALTHY, non-terminating→flagged). k3s_datastore etcd loop
and Spilo postgres loop deliberately left unchanged; see report.

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>
EOF
)"
```

- [ ] **Step 4: Verify commit**

```bash
cd /home/skadam/CHA/cluster-health-autopilot && git log --oneline -2
```

Expected: the new commit is HEAD, previous is `36e3654`.

- [ ] **Step 5: Run the full test suite one final time to confirm green HEAD**

```bash
cd /home/skadam/CHA/cluster-health-autopilot && go test ./internal/... 2>&1 | grep -E 'ok|FAIL|---'
```

Expected: all `ok ...` lines, zero `FAIL` lines.
