// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package watcher

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

// Sprint 4.3 — leader election via client-go's Lease-based primitive.
//
// Srenix's watcher loop is safe at replicas=1. With replicas>=2, both pods
// would race to apply fixes and post duplicate Slack alerts. The chart
// historically defaulted to one replica with no enforcement; this file
// adds Lease-based leader election so HA deployments are safe.
//
// Design intent: identical durations to controller-manager
// (LeaseDuration 30s / RenewDeadline 20s / RetryPeriod 5s) so the Sprint
// 5 controller-runtime port can keep this lease name and inherit the
// same operational behavior.

// LeaderConfig configures the elector. All fields have defaults.
type LeaderConfig struct {
	// Namespace is where the coordination.k8s.io/v1.Lease object lives.
	// Defaults to MY_POD_NAMESPACE (downward API) or "agentic-sre".
	Namespace string

	// LeaseName is the Lease object's name. Defaults to "srenix-watcher".
	// The Sprint 5 operator port keeps this name so the lease survives.
	LeaseName string

	// Identity uniquely identifies this candidate. Defaults to MY_POD_NAME
	// or hostname.
	Identity string

	// LeaseDuration: how long the lease is valid after acquisition.
	// Default 30s.
	LeaseDuration time.Duration

	// RenewDeadline: how long the leader has to refresh before losing it.
	// Default 20s.
	RenewDeadline time.Duration

	// RetryPeriod: how often candidates poll. Default 5s.
	RetryPeriod time.Duration

	// Clientset is the K8s client used for lease coordination. Required
	// when leader election is enabled; tests inject a fake.
	Clientset kubernetes.Interface

	// Disabled bypasses election entirely — RunWithLeader runs the body
	// immediately without acquiring a lease. Set via SRENIX_LEADER_ELECTION=off
	// or the matching Helm value. Use in single-pod dev or when running
	// outside K8s (srenix snapshot capture, ad-hoc diagnose).
	Disabled bool
}

// withDefaults fills in zero-value fields from env / defaults.
func (c LeaderConfig) withDefaults() LeaderConfig {
	out := c
	if out.Namespace == "" {
		out.Namespace = os.Getenv("MY_POD_NAMESPACE")
	}
	if out.Namespace == "" {
		out.Namespace = "agentic-sre"
	}
	if out.LeaseName == "" {
		out.LeaseName = "srenix-watcher"
	}
	if out.Identity == "" {
		out.Identity = os.Getenv("MY_POD_NAME")
	}
	if out.Identity == "" {
		host, err := os.Hostname()
		if err == nil {
			out.Identity = host
		}
	}
	if out.Identity == "" {
		out.Identity = fmt.Sprintf("srenix-watcher-%d", time.Now().UnixNano())
	}
	if out.LeaseDuration == 0 {
		out.LeaseDuration = 30 * time.Second
	}
	if out.RenewDeadline == 0 {
		out.RenewDeadline = 20 * time.Second
	}
	if out.RetryPeriod == 0 {
		out.RetryPeriod = 5 * time.Second
	}
	return out
}

// ErrLeaderElectionRequiresClientset is returned by RunWithLeader when
// leader election is enabled (Disabled=false) but no Clientset is
// supplied. The caller is expected to set Clientset from the watcher's
// live snapshot.
var ErrLeaderElectionRequiresClientset = errors.New(
	"leader election enabled but no Clientset supplied; set LeaderConfig.Clientset or LeaderConfig.Disabled = true")

// RunWithLeader runs body() exactly once for the lifetime of this
// process while this pod holds the lease. If the lease is lost, body's
// context is cancelled and the goroutine returns; client-go internally
// retries acquisition (next OnStartedLeading fires when we win again).
//
// When cfg.Disabled is true, body runs immediately with the parent ctx
// — no lease created — so single-pod and out-of-cluster invocations
// work unchanged.
//
// The OnStoppedLeading callback fires when leadership is lost or the
// process exits. By convention, ANY loss of leadership results in
// process exit so the caller restarts cleanly via the Deployment
// controller — there is no "graceful re-acquire" branch.
//
// Returns ctx.Err() on shutdown.
func RunWithLeader(ctx context.Context, cfg LeaderConfig, body func(context.Context) error) error {
	cfg = cfg.withDefaults()
	if cfg.Disabled {
		log.Printf("watcher: leader election disabled; running body directly")
		return body(ctx)
	}
	if cfg.Clientset == nil {
		return ErrLeaderElectionRequiresClientset
	}

	lock := &resourcelock.LeaseLock{
		LeaseMeta: leaseObjectMeta(cfg.Namespace, cfg.LeaseName),
		Client:    cfg.Clientset.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: cfg.Identity,
		},
	}

	leaderErr := make(chan error, 1)

	go leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
		Lock:            lock,
		ReleaseOnCancel: true,
		LeaseDuration:   cfg.LeaseDuration,
		RenewDeadline:   cfg.RenewDeadline,
		RetryPeriod:     cfg.RetryPeriod,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(leaderCtx context.Context) {
				log.Printf("watcher: acquired lease %s/%s as %q",
					cfg.Namespace, cfg.LeaseName, cfg.Identity)
				if err := body(leaderCtx); err != nil && !errors.Is(err, context.Canceled) {
					leaderErr <- err
				} else {
					leaderErr <- nil
				}
			},
			OnStoppedLeading: func() {
				log.Printf("watcher: lost lease %s/%s", cfg.Namespace, cfg.LeaseName)
			},
			OnNewLeader: func(identity string) {
				if identity != cfg.Identity {
					log.Printf("watcher: standby; current leader = %q", identity)
				}
			},
		},
	})

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-leaderErr:
		return err
	}
}

// leaderElectionDisabledFromEnv reads SRENIX_LEADER_ELECTION; "off" returns
// true. Used by the srenix watch command to wire the Disabled flag.
func leaderElectionDisabledFromEnv() bool {
	v := os.Getenv("SRENIX_LEADER_ELECTION")
	return v == "off" || v == "false" || v == "0"
}
