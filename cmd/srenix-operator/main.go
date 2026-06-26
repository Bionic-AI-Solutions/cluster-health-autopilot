// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

// srenix-operator is the controller-runtime manager that reconciles
// AgenticSRE CRs into the watcher Deployment + diagnose
// / remediate CronJobs + ServiceAccount the existing chart already
// templates.
//
// Phase 1b ships the manager binary + Reconciler only. Existing
// chart-managed installs continue to work unchanged; the operator
// is opt-in via `operator.enabled=true` in Helm values. Operators
// who do NOT create a AgenticSRE CR see no behavior
// change from this binary running.
package main

import (
	"flag"
	"fmt"
	"os"

	chav1alpha1 "github.com/srenix-ai/agentic-sre/api/v1alpha1"
	chaoperator "github.com/srenix-ai/agentic-sre/internal/operator"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(chav1alpha1.AddToScheme(scheme))
}

// operatorFlags holds the parsed flag targets for the srenix-operator
// manager. registerOperatorFlags binds them onto a FlagSet so the same
// registration is reused by main() and by the chart-args↔binary-flags
// parity gate (internal/chartgate / cmd tests) — keeping the gate's
// notion of "valid operator flags" in lockstep with the real binary
// instead of a hand-maintained copy.
type operatorFlags struct {
	metricsAddr             string
	probeAddr               string
	enableLeaderElection    bool
	leaderElectionNamespace string
}

// registerOperatorFlags binds the srenix-operator flags (excluding the
// zap logger flags, which are bound separately via zap.Options.BindFlags)
// onto fs. Exposed so the parity gate can enumerate the real flag set.
func registerOperatorFlags(fs *flag.FlagSet) *operatorFlags {
	f := &operatorFlags{}
	fs.StringVar(&f.metricsAddr, "metrics-bind-address", ":8080",
		"The address the metric endpoint binds to.")
	fs.StringVar(&f.probeAddr, "health-probe-bind-address", ":8081",
		"The address the probe endpoint binds to.")
	fs.BoolVar(&f.enableLeaderElection, "leader-elect", true,
		"Enable leader election for controller manager — recommended even "+
			"for single-replica installs so a restart sees a clean lease handoff.")
	fs.StringVar(&f.leaderElectionNamespace, "leader-election-namespace", "",
		"The namespace the leader-election Lease lives in. Defaults to the "+
			"namespace the operator pod runs in (read from the downward API).")
	return f
}

func main() {
	f := registerOperatorFlags(flag.CommandLine)

	opts := zap.Options{Development: false}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	metricsAddr := f.metricsAddr
	probeAddr := f.probeAddr
	enableLeaderElection := f.enableLeaderElection
	leaderElectionNamespace := f.leaderElectionNamespace

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	// Fall back to MY_POD_NAMESPACE downward-API env var if the
	// flag wasn't supplied. Mirrors the watcher's lease-bound
	// behavior.
	if leaderElectionNamespace == "" {
		leaderElectionNamespace = os.Getenv("MY_POD_NAMESPACE")
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: metricsAddr,
		},
		HealthProbeBindAddress:  probeAddr,
		LeaderElection:          enableLeaderElection,
		LeaderElectionID:        "srenix-operator.srenix.ai",
		LeaderElectionNamespace: leaderElectionNamespace,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to start manager: %v\n", err)
		os.Exit(1)
	}

	r := &chaoperator.Reconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}
	if err := r.SetupWithManager(mgr); err != nil {
		fmt.Fprintf(os.Stderr, "unable to register reconciler: %v\n", err)
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("ping", healthz.Ping); err != nil {
		fmt.Fprintf(os.Stderr, "unable to set up healthz check: %v\n", err)
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("ping", healthz.Ping); err != nil {
		fmt.Fprintf(os.Stderr, "unable to set up readyz check: %v\n", err)
		os.Exit(1)
	}

	ctrl.Log.Info("starting srenix-operator manager",
		"leader-election", enableLeaderElection,
		"leader-election-namespace", leaderElectionNamespace,
	)
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		fmt.Fprintf(os.Stderr, "manager exited: %v\n", err)
		os.Exit(1)
	}
}
