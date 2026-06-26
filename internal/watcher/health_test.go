// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package watcher

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	coordinationv1 "k8s.io/api/coordination/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// P1.9(a) — /healthz must serve UNCONDITIONALLY, independent of the M6
// webhook receiver. Before this fix the only /healthz lived inside the
// `if WebhookListen != ""` branch, so a watcher with no webhook trigger
// had no HTTP health endpoint and the chart could not wire liveness /
// readiness probes.
func TestHealthHandler_ServesOKWithoutWebhookListen(t *testing.T) {
	w := New(nil, nil, nil, Config{}) // WebhookListen empty

	srv := httptest.NewServer(w.healthHandler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /healthz status = %d, want 200", resp.StatusCode)
	}
}

// healthListenAddr resolves the effective health listen address: the
// explicit HealthListen, else the default :8081. (It intentionally does
// NOT fall back to WebhookListen — the health server is always-on and
// must not share the webhook port's lifecycle.)
func TestHealthListenAddr_DefaultsTo8081(t *testing.T) {
	if got := (&Watcher{cfg: Config{}}).healthListenAddr(); got != ":8081" {
		t.Fatalf("default healthListenAddr = %q, want :8081", got)
	}
	if got := (&Watcher{cfg: Config{HealthListen: ":9000"}}).healthListenAddr(); got != ":9000" {
		t.Fatalf("explicit healthListenAddr = %q, want :9000", got)
	}
}

// REGRESSION — production 1.26.0 upgrade incident (O11).
//
// PR #186 introduced the always-on :8081 health server but started it
// INSIDE Watcher.Run, which `srenix watch` wraps in RunWithLeader — so the
// listener only came up inside OnStartedLeading, AFTER the lease was
// acquired. A standby (non-leader) pod served nothing: the liveness
// probe added by the same PR got `connection refused` and kubelet
// kill-looped the pod. Under the operator Deployment's RollingUpdate
// maxUnavailable=0 every upgrade deadlocked: the new pod could never
// go healthy while the old leader held the lease, and the old pod was
// never terminated. Recovery required deleting the old leader by hand.
//
// This test pins the fix: the health server must serve /healthz 200
// BEFORE and WITHOUT lease acquisition. It starts the health server
// the way cmd/srenix does (StartHealthServer before RunWithLeader), with
// the lease already held by another identity so this candidate stays
// standby, and asserts /healthz answers 200 while the body (the watch
// loop) has never run.
func TestStartHealthServer_Serves200WhileStandby_NotLeader(t *testing.T) {
	const (
		ns        = "test-ns"
		leaseName = "test-srenix-standby"
		holder    = "other-pod-the-leader"
	)
	// Lease held by someone else, renewed "now", valid for 5 minutes —
	// our candidate cannot acquire it for the lifetime of this test.
	now := metav1.NewMicroTime(time.Now())
	heldBy := holder
	cs := fake.NewSimpleClientset(&coordinationv1.Lease{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: leaseName},
		Spec: coordinationv1.LeaseSpec{
			HolderIdentity:       &heldBy,
			LeaseDurationSeconds: func(i int32) *int32 { return &i }(300),
			AcquireTime:          &now,
			RenewTime:            &now,
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w := New(nil, nil, nil, Config{HealthListen: "127.0.0.1:0"})

	// 1) Health server starts BEFORE leader election — mirrors cmd/srenix.
	if err := w.StartHealthServer(ctx); err != nil {
		t.Fatalf("StartHealthServer: %v", err)
	}
	addr := w.healthAddr()
	if addr == "" {
		t.Fatal("healthAddr empty after StartHealthServer")
	}

	// 2) Enter leader election as a standby candidate.
	bodyRan := make(chan struct{})
	go func() {
		_ = RunWithLeader(ctx, LeaderConfig{
			Namespace:     ns,
			LeaseName:     leaseName,
			Identity:      "standby-pod",
			LeaseDuration: 200 * time.Millisecond,
			RenewDeadline: 150 * time.Millisecond,
			RetryPeriod:   50 * time.Millisecond,
			Clientset:     cs,
		}, func(_ context.Context) error {
			close(bodyRan) // must never happen — lease is held by another pod
			return nil
		})
	}()

	// Give the elector time to try (and fail) to acquire the lease.
	time.Sleep(300 * time.Millisecond)

	select {
	case <-bodyRan:
		t.Fatal("watch-loop body ran while the lease was held by another identity")
	default:
	}

	// 3) The liveness probe's view: /healthz must be 200 in standby.
	resp, err := http.Get(fmt.Sprintf("http://%s/healthz", addr))
	if err != nil {
		t.Fatalf("GET /healthz while standby: %v (this is the 1.26.0 kill-loop regression)", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /healthz while standby = %d, want 200", resp.StatusCode)
	}
}

// StartHealthServer must be idempotent: Watcher.Run also calls it (so
// direct Run callers and the leader-election-disabled path keep the
// listener), and the second call must not try to re-bind the port.
func TestStartHealthServer_Idempotent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w := New(nil, nil, nil, Config{HealthListen: "127.0.0.1:0"})
	if err := w.StartHealthServer(ctx); err != nil {
		t.Fatalf("first StartHealthServer: %v", err)
	}
	first := w.healthAddr()
	if err := w.StartHealthServer(ctx); err != nil {
		t.Fatalf("second StartHealthServer: %v", err)
	}
	if got := w.healthAddr(); got != first {
		t.Fatalf("second StartHealthServer rebound the listener: %q -> %q", first, got)
	}

	resp, err := http.Get(fmt.Sprintf("http://%s/healthz", first))
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /healthz = %d, want 200", resp.StatusCode)
	}
}
