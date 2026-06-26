// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	"os/exec"
	"strings"
	"testing"
)

// P1.9(a) — chart assertions for the watcher Deployment health probes,
// the always-on --health-listen arg, and the replicas × leaderElection
// guard. Skips when helm is absent (local `go test ./...`); CI has helm
// on PATH, so the gate runs there.

func helmTemplateWatcher(t *testing.T, sets ...string) (string, error) {
	t.Helper()
	if _, err := exec.LookPath("helm"); err != nil {
		t.Skipf("helm not on PATH — skipping watcher chart health gate locally (runs in CI)")
	}
	args := []string{"template", "t", "../../charts/agentic-sre",
		"--set", "watcher.enabled=true"}
	for _, s := range sets {
		args = append(args, "--set", s)
	}
	out, err := exec.Command("helm", args...).CombinedOutput()
	return string(out), err
}

func TestWatcherChart_HealthProbesAndArg(t *testing.T) {
	out, err := helmTemplateWatcher(t)
	if err != nil {
		t.Fatalf("helm template failed: %v\n%s", err, out)
	}
	for _, want := range []string{
		"--health-listen=:8081",
		"livenessProbe:",
		"readinessProbe:",
		"path: /healthz",
		"name: health",
		"containerPort: 8081",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered watcher Deployment missing %q", want)
		}
	}
}

func TestWatcherChart_ReplicasGuard_FailsWithoutLeaderElection(t *testing.T) {
	out, err := helmTemplateWatcher(t,
		"watcher.replicas=2", "watcher.leaderElection.enabled=false")
	if err == nil {
		t.Fatalf("expected helm template to FAIL for replicas=2 + leaderElection=false; got success:\n%s", out)
	}
	if !strings.Contains(out, "watcher.replicas > 1 requires watcher.leaderElection.enabled=true") {
		t.Errorf("guard fired but with an unexpected message:\n%s", out)
	}
}

func TestWatcherChart_ReplicasGuard_PassesWithLeaderElection(t *testing.T) {
	out, err := helmTemplateWatcher(t,
		"watcher.replicas=2", "watcher.leaderElection.enabled=true")
	if err != nil {
		t.Fatalf("replicas=2 + leaderElection=true should render; got error: %v\n%s", err, out)
	}
	if !strings.Contains(out, "replicas: 2") {
		t.Errorf("rendered watcher Deployment should carry replicas: 2:\n%s", out)
	}
}
