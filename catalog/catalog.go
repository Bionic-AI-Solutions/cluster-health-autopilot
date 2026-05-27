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
	"os"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/diagnose"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/fix"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/investigator"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/probe"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/registry"
)

// RegisterOSS adds all built-in OSS-tier probes, analyzers, and fixers to r.
//
// Sprint 2 added six new probes covering the K8s health blind-spots the
// hardcoded Services target list missed: node pressure, system DaemonSets,
// stuck Pending pods, generic CrashLoopBackOff, ETCD members, and failed
// volume mounts. Each is independently disablable via CHA_PROBE_<NAME>=off.
func RegisterOSS(r *registry.Registry) {
	// Services-probe targets: the compiled-in defaults remain the baseline
	// (backward-compat for the Bionic cluster the project was built on),
	// merged with anything supplied via CHA_CRITICAL_SERVICES env or the
	// cha.bionicaisolutions.com/probe-critical annotation. Operators with
	// non-Bionic clusters override via the env to replace the default set.
	servicesTargets := probe.DefaultTargets()
	if extra := probe.TargetsFromEnv(os.Getenv("CHA_CRITICAL_SERVICES")); len(extra) > 0 {
		if os.Getenv("CHA_CRITICAL_SERVICES_REPLACE") == "true" {
			servicesTargets = extra
		} else {
			servicesTargets = append(servicesTargets, extra...)
		}
	}

	r.RegisterProbe(
		probe.Ceph{},
		probe.Nodes{},
		probe.Postgres{},
		probe.PVCs{},
		probe.Services{Targets: servicesTargets},
		probe.NewEndpoints(
			probe.DefaultEndpointTargets(),
			probe.DefaultDiscoveryOptions(),
		),
	)

	// Sprint 2 probes — opt-out via env so a cluster with weird shape can
	// silence individual probes without forking.
	if os.Getenv("CHA_PROBE_NODE_PRESSURE") != "off" {
		r.RegisterProbe(probe.NodePressure{})
	}
	if os.Getenv("CHA_PROBE_DAEMONSETS") != "off" {
		r.RegisterProbe(probe.DaemonSets{})
	}
	if os.Getenv("CHA_PROBE_PENDING_PODS") != "off" {
		r.RegisterProbe(probe.PendingPods{})
	}
	if os.Getenv("CHA_PROBE_CRASHLOOP") != "off" {
		r.RegisterProbe(probe.CrashLoopBackOff{})
	}
	if os.Getenv("CHA_PROBE_ETCD") != "off" {
		r.RegisterProbe(probe.ETCD{})
	}
	if os.Getenv("CHA_PROBE_FAILED_MOUNTS") != "off" {
		r.RegisterProbe(probe.FailedMounts{})
	}
	r.RegisterAnalyzer(
		diagnose.SecretKeyMissing{},
		diagnose.FailingExternalSecrets{},
		diagnose.ProactiveSecretKeyCheck{},
		diagnose.UnprovisionedSecret{},
		diagnose.ImagePullAuth{},
		diagnose.CertExpiry{},
		diagnose.TLSSecretMismatch{},
	)
	// v1.7 drift-class expansion (Workstream B1). Opt-in via env var
	// because the CRDs aren't installed on every cluster; ESO/cert-mgr
	// analyzers are unconditionally registered because their CRDs are
	// near-ubiquitous in our pilot installs, but Argo/Flux is partial
	// adoption and we don't want a noisy "no resources" lookup cycle
	// on clusters that don't run a GitOps controller.
	if os.Getenv("CHA_ANALYZER_GITOPS_DRIFT") != "off" {
		r.RegisterAnalyzer(diagnose.GitOpsDrift{})
	}
	r.RegisterFixer(
		fix.StaleErrorPods{},
		fix.StuckJobsWithBadSecretRef{},
		fix.StuckRSPods{},
		fix.StuckCertificateRequests{},
	)
	// Opt-in fixers — registered only when explicitly enabled. The matching
	// Helm value flips the env var and adds the required RBAC verbs.
	if os.Getenv("CHA_FIXER_TLS_SECRET_MISMATCH") == "true" {
		r.RegisterFixer(fix.TLSSecretMismatch{})
	}

	// Layer-2 investigator: deterministic, rule-based, ships in OSS.
	// Disable with CHA_INVESTIGATOR=off; the paid binary may replace it with
	// an LLM-backed implementation after this registration runs.
	if os.Getenv("CHA_INVESTIGATOR") != "off" {
		r.RegisterInvestigator(investigator.RuleBased{})
	}
}

// Default returns a Registry pre-loaded with the OSS pattern set.
func Default() *registry.Registry {
	r := registry.New()
	RegisterOSS(r)
	return r
}
