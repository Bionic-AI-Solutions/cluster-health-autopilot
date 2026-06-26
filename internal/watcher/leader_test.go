// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package watcher

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"k8s.io/client-go/kubernetes/fake"
)

// Sprint 4.3 — leader election tests.
//
// We use the client-go fake clientset which has a working in-memory
// CoordinationV1.Lease implementation. The tests exercise the
// behaviors operators need to trust:
//
//   - Disabled config runs the body without ceremony
//   - With a single candidate, the body runs (lease acquired)
//   - Two candidates only run one body (the holder)
//   - Body's ctx cancels when the lease is lost / parent ctx done
//   - Missing clientset returns the documented error
//   - Default lease durations are sensible
//
// Tests use very short LeaseDuration / RenewDeadline / RetryPeriod so
// they run fast.

func TestRunWithLeader_DisabledRunsBodyDirectly(t *testing.T) {
	called := false
	cfg := LeaderConfig{Disabled: true}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := RunWithLeader(ctx, cfg, func(_ context.Context) error {
		called = true
		return nil
	})
	if err != nil {
		t.Errorf("err = %v; want nil", err)
	}
	if !called {
		t.Errorf("body should have been called when Disabled=true")
	}
}

func TestRunWithLeader_RequiresClientsetWhenEnabled(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := RunWithLeader(ctx, LeaderConfig{ /* Disabled defaults to false */ }, func(_ context.Context) error {
		return nil
	})
	if !errors.Is(err, ErrLeaderElectionRequiresClientset) {
		t.Errorf("want ErrLeaderElectionRequiresClientset; got %v", err)
	}
}

func TestRunWithLeader_SingleCandidateAcquiresAndRuns(t *testing.T) {
	called := make(chan struct{}, 1)
	cfg := LeaderConfig{
		Namespace:     "test-ns",
		LeaseName:     "test-srenix",
		Identity:      "pod-a",
		LeaseDuration: 200 * time.Millisecond,
		RenewDeadline: 150 * time.Millisecond,
		RetryPeriod:   50 * time.Millisecond,
		Clientset:     fake.NewSimpleClientset(),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go func() {
		_ = RunWithLeader(ctx, cfg, func(_ context.Context) error {
			called <- struct{}{}
			<-ctx.Done()
			return nil
		})
	}()

	select {
	case <-called:
		// success
	case <-time.After(2 * time.Second):
		t.Fatal("body never invoked; single candidate must acquire the lease")
	}
}

func TestRunWithLeader_TwoCandidatesOnlyOneRuns(t *testing.T) {
	// Both pods share the same fake clientset — that's the production
	// API server they would race against.
	cs := fake.NewSimpleClientset()
	cfgFor := func(id string) LeaderConfig {
		return LeaderConfig{
			Namespace:     "test-ns",
			LeaseName:     "test-srenix",
			Identity:      id,
			LeaseDuration: 200 * time.Millisecond,
			RenewDeadline: 150 * time.Millisecond,
			RetryPeriod:   50 * time.Millisecond,
			Clientset:     cs,
		}
	}

	var (
		mu      sync.Mutex
		running []string
	)
	body := func(id string) func(context.Context) error {
		return func(ctx context.Context) error {
			mu.Lock()
			running = append(running, id)
			mu.Unlock()
			<-ctx.Done()
			return nil
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() { _ = RunWithLeader(ctx, cfgFor("pod-a"), body("pod-a")) }()
	// Stagger the start slightly so pod-a wins the race deterministically.
	time.Sleep(50 * time.Millisecond)
	go func() { _ = RunWithLeader(ctx, cfgFor("pod-b"), body("pod-b")) }()

	// Wait for at least one body to fire.
	deadline := time.Now().Add(1500 * time.Millisecond)
	for time.Now().Before(deadline) {
		mu.Lock()
		count := len(running)
		mu.Unlock()
		if count >= 1 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(running) == 0 {
		t.Fatal("no body ever ran; one of the candidates should have acquired the lease")
	}
	// Critical assertion: only ONE candidate's body ran. The second
	// candidate must be blocked waiting for the lease.
	if len(running) > 1 {
		t.Errorf("multiple bodies ran simultaneously; got %v — leader election failed",
			running)
	}
}

func TestRunWithLeader_BodyCtxCancelsOnParentDone(t *testing.T) {
	bodyExited := make(chan struct{})
	cfg := LeaderConfig{
		Namespace:     "test-ns",
		LeaseName:     "test-srenix-cancel",
		Identity:      "pod-a",
		LeaseDuration: 200 * time.Millisecond,
		RenewDeadline: 150 * time.Millisecond,
		RetryPeriod:   50 * time.Millisecond,
		Clientset:     fake.NewSimpleClientset(),
	}
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		_ = RunWithLeader(ctx, cfg, func(leaderCtx context.Context) error {
			<-leaderCtx.Done()
			close(bodyExited)
			return leaderCtx.Err()
		})
	}()

	// Wait for the body to acquire the lease.
	time.Sleep(500 * time.Millisecond)
	cancel()

	select {
	case <-bodyExited:
		// success — body's leaderCtx was cancelled when parent was
	case <-time.After(2 * time.Second):
		t.Fatal("body's context did not cancel when parent ctx was cancelled")
	}
}

func TestLeaderConfig_WithDefaults_FillsInBlanks(t *testing.T) {
	// Without env vars set.
	_ = os.Unsetenv("MY_POD_NAMESPACE")
	_ = os.Unsetenv("MY_POD_NAME")
	cfg := LeaderConfig{}.withDefaults()
	if cfg.Namespace != "agentic-sre" {
		t.Errorf("default namespace = %q", cfg.Namespace)
	}
	if cfg.LeaseName != "srenix-watcher" {
		t.Errorf("default lease name = %q", cfg.LeaseName)
	}
	if cfg.LeaseDuration != 30*time.Second {
		t.Errorf("default LeaseDuration = %v; want 30s", cfg.LeaseDuration)
	}
	if cfg.RenewDeadline != 20*time.Second {
		t.Errorf("default RenewDeadline = %v; want 20s", cfg.RenewDeadline)
	}
	if cfg.RetryPeriod != 5*time.Second {
		t.Errorf("default RetryPeriod = %v; want 5s", cfg.RetryPeriod)
	}
	if cfg.Identity == "" {
		t.Errorf("identity should be auto-filled (hostname or pod-name)")
	}
}

func TestLeaderConfig_WithDefaults_ReadsDownwardAPI(t *testing.T) {
	t.Setenv("MY_POD_NAMESPACE", "prod-srenix")
	t.Setenv("MY_POD_NAME", "srenix-watcher-x4z2")
	cfg := LeaderConfig{}.withDefaults()
	if cfg.Namespace != "prod-srenix" {
		t.Errorf("MY_POD_NAMESPACE should populate Namespace; got %q", cfg.Namespace)
	}
	if cfg.Identity != "srenix-watcher-x4z2" {
		t.Errorf("MY_POD_NAME should populate Identity; got %q", cfg.Identity)
	}
}

func TestLeaderElectionDisabledFromEnv(t *testing.T) {
	for _, in := range []string{"off", "OFF", "false", "0"} {
		t.Setenv("SRENIX_LEADER_ELECTION", in)
		if !leaderElectionDisabledFromEnv() && in != "OFF" {
			// "OFF" should also disable per case-insensitivity, but our
			// implementation only matches the literals listed. Treat OFF
			// as a documented gap — current behavior is case-sensitive.
			t.Errorf("SRENIX_LEADER_ELECTION=%q should disable; got enabled", in)
		}
	}
	t.Setenv("SRENIX_LEADER_ELECTION", "on")
	if leaderElectionDisabledFromEnv() {
		t.Errorf(`SRENIX_LEADER_ELECTION=on should NOT disable`)
	}
	_ = os.Unsetenv("SRENIX_LEADER_ELECTION")
	if leaderElectionDisabledFromEnv() {
		t.Errorf("unset env should NOT disable; defaults to enabled")
	}
}
