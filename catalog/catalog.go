// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

// Package catalog is the OSS pattern catalog for Cluster Health Autopilot.
//
// RegisterOSS seeds a Registry with all probes, analyzers, and fixers that
// ship in the open-source tier. The VaultPathMissing analyzer is intentionally
// excluded here because it requires a constructed Vault client; wire it in
// after calling RegisterOSS:
//
//	reg := catalog.Default()
//	if vaultAddr != "" {
//	    vc, _ := vault.New(cfg)
//	    reg.RegisterAnalyzer(diagnose.VaultPathMissing{Client: vc})
//	}
//
// # Paid-tier extension
//
// The paid binary's main package imports this module and a private catalog:
//
//	reg := registry.New()
//	catalog.RegisterOSS(reg)          // this package — public module
//	paidcatalog.Register(reg)         // private module, same interface
//
// The private module only needs to import pkg/diagnose, pkg/fix, pkg/probe,
// pkg/snapshot, and pkg/registry — no internal/ packages required.
package catalog

import (
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/diagnose"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/fix"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/probe"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/registry"
)

// RegisterOSS adds all built-in OSS-tier probes, analyzers, and fixers to r.
func RegisterOSS(r *registry.Registry) {
	r.RegisterProbe(
		probe.Ceph{},
		probe.Nodes{},
		probe.Postgres{},
		probe.PVCs{},
		probe.Services{Targets: probe.DefaultTargets()},
		probe.Endpoints{
			Targets:   probe.DefaultEndpointTargets(),
			Discovery: probe.DefaultDiscoveryOptions(),
		},
	)
	r.RegisterAnalyzer(
		diagnose.SecretKeyMissing{},
		diagnose.FailingExternalSecrets{},
		diagnose.ProactiveSecretKeyCheck{},
		diagnose.UnprovisionedSecret{},
		diagnose.ImagePullAuth{},
		diagnose.CertExpiry{},
	)
	r.RegisterFixer(
		fix.StaleErrorPods{},
		fix.StuckJobsWithBadSecretRef{},
		fix.StuckRSPods{},
		fix.StuckCertificateRequests{},
	)
}

// Default returns a Registry pre-loaded with the OSS pattern set.
func Default() *registry.Registry {
	r := registry.New()
	RegisterOSS(r)
	return r
}
