// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

// Package catalog is the OSS pattern catalog for Agentic SRE.
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
	"context"
	"os"

	"github.com/srenix-ai/agentic-sre/internal/diagnose"
	"github.com/srenix-ai/agentic-sre/internal/dns/cloudflare"
	"github.com/srenix-ai/agentic-sre/internal/fix"
	"github.com/srenix-ai/agentic-sre/internal/investigator"
	"github.com/srenix-ai/agentic-sre/internal/probe"
	"github.com/srenix-ai/agentic-sre/pkg/registry"
)

// cfClientAdapter wraps the concrete cloudflare.Client and adapts its
// return types to the diagnose.CloudflareClient interface.
type cfClientAdapter struct {
	inner cloudflare.Client
}

func (a cfClientAdapter) ListZones(ctx context.Context) ([]diagnose.Zone, error) {
	zones, err := a.inner.ListZones(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]diagnose.Zone, len(zones))
	for i, z := range zones {
		out[i] = diagnose.Zone{ID: z.ID, Name: z.Name}
	}
	return out, nil
}

func (a cfClientAdapter) ListDNSRecords(ctx context.Context, zoneID string) ([]diagnose.DNSRecord, error) {
	records, err := a.inner.ListDNSRecords(ctx, zoneID)
	if err != nil {
		return nil, err
	}
	out := make([]diagnose.DNSRecord, len(records))
	for i, r := range records {
		out[i] = diagnose.DNSRecord{
			Name:    r.Name,
			Type:    r.Type,
			Content: r.Content,
			Proxied: r.Proxied,
		}
	}
	return out, nil
}

// RegisterOSS adds all built-in OSS-tier probes, analyzers, and fixers to r.
//
// Sprint 2 added six new probes covering the K8s health blind-spots the
// hardcoded Services target list missed: node pressure, system DaemonSets,
// stuck Pending pods, generic CrashLoopBackOff, ETCD members, and failed
// volume mounts. Each is independently disablable via SRENIX_PROBE_<NAME>=off.
func RegisterOSS(r *registry.Registry) {
	// Services-probe targets: the compiled-in defaults remain the baseline
	// (backward-compat for the Bionic cluster the project was built on),
	// merged with anything supplied via SRENIX_CRITICAL_SERVICES env or the
	// srenix.ai/probe-critical annotation. Operators with
	// non-Bionic clusters override via the env to replace the default set.
	servicesTargets := probe.DefaultTargets()
	if extra := probe.TargetsFromEnv(os.Getenv("SRENIX_CRITICAL_SERVICES")); len(extra) > 0 {
		if os.Getenv("SRENIX_CRITICAL_SERVICES_REPLACE") == "true" {
			servicesTargets = extra
		} else {
			servicesTargets = append(servicesTargets, extra...)
		}
	}

	// Base probes — the original six. Each defaults ON and is
	// independently disablable via SRENIX_PROBE_<NAME>=off, honoring the
	// docs' "each probe independently togglable" promise. The
	// CRITICAL_WORKLOADS toggle gates the Critical Services probe
	// (the documented env name predates the probe's display name).
	if os.Getenv("SRENIX_PROBE_CEPH") != "off" {
		r.RegisterProbe(probe.Ceph{})
	}
	if os.Getenv("SRENIX_PROBE_NODES") != "off" {
		r.RegisterProbe(probe.Nodes{})
	}
	if os.Getenv("SRENIX_PROBE_POSTGRES") != "off" {
		r.RegisterProbe(probe.Postgres{})
	}
	if os.Getenv("SRENIX_PROBE_PVCS") != "off" {
		r.RegisterProbe(probe.PVCs{})
	}
	if os.Getenv("SRENIX_PROBE_CRITICAL_WORKLOADS") != "off" {
		r.RegisterProbe(probe.Services{Targets: servicesTargets})
	}
	if os.Getenv("SRENIX_PROBE_ENDPOINTS") != "off" {
		r.RegisterProbe(probe.NewEndpoints(
			probe.DefaultEndpointTargets(),
			probe.DefaultDiscoveryOptions(),
		))
	}

	// Sprint 2 probes — opt-out via env so a cluster with weird shape can
	// silence individual probes without forking.
	if os.Getenv("SRENIX_PROBE_NODE_PRESSURE") != "off" {
		r.RegisterProbe(probe.NodePressure{})
	}
	if os.Getenv("SRENIX_PROBE_DAEMONSETS") != "off" {
		r.RegisterProbe(probe.DaemonSets{})
	}
	if os.Getenv("SRENIX_PROBE_PENDING_PODS") != "off" {
		r.RegisterProbe(probe.PendingPods{})
	}
	if os.Getenv("SRENIX_PROBE_CRASHLOOP") != "off" {
		r.RegisterProbe(probe.CrashLoopBackOff{})
	}
	if os.Getenv("SRENIX_PROBE_ETCD") != "off" {
		r.RegisterProbe(probe.ETCD{})
	}
	if os.Getenv("SRENIX_PROBE_FAILED_MOUNTS") != "off" {
		r.RegisterProbe(probe.FailedMounts{})
	}
	// M2 probe-class additions (v1.8). Each auto-skips when its CRD
	// is absent (Kong / ArgoCD / Velero) or no-ops on an empty list
	// (HPA), so default-on is safe; the env opt-out is for operators
	// who want to silence a probe on a cluster that does host the
	// CRD but doesn't want Srenix watching it.
	if os.Getenv("SRENIX_PROBE_KONG") != "off" {
		r.RegisterProbe(probe.Kong{})
	}
	// v1.23+ (M2) — KongRoutes verifies each Kong-managed Ingress
	// has a ready backend Endpoint + that KongPlugin/Consumer
	// references resolve. Silent on clusters without Kong-managed
	// Ingresses (no Kong-class/annotation match). Toggle off via
	// SRENIX_PROBE_KONG_ROUTES=off.
	if os.Getenv("SRENIX_PROBE_KONG_ROUTES") != "off" {
		r.RegisterProbe(probe.KongRoutes{})
	}
	// v1.25+ (M3) — GPUNodes detects NotReady / cordoned / zero-
	// allocatable GPU nodes. Silent on CPU-only clusters. Toggle
	// off via SRENIX_PROBE_GPU_NODES=off.
	if os.Getenv("SRENIX_PROBE_GPU_NODES") != "off" {
		r.RegisterProbe(probe.GPUNodes{})
	}
	if os.Getenv("SRENIX_PROBE_HPA_SCALING") != "off" {
		r.RegisterProbe(probe.HPAScaling{})
	}
	if os.Getenv("SRENIX_PROBE_ARGOCD_APP") != "off" {
		r.RegisterProbe(probe.ArgoCDApplication{})
	}
	if os.Getenv("SRENIX_PROBE_VELERO") != "off" {
		r.RegisterProbe(probe.Velero{})
	}
	// k3s-specific probes (v1.10) — safe to register default-on because each
	// auto-skips gracefully when not on a k3s cluster or when the required CRD
	// is absent (TraefikRoutes, K3sDatastore). K3sLocalPathStorage no-ops when
	// there are no local-path PVCs.
	if os.Getenv("SRENIX_PROBE_TRAEFIK_ROUTES") != "off" {
		r.RegisterProbe(probe.TraefikRoutes{})
	}
	if os.Getenv("SRENIX_PROBE_K3S_LOCALPATH") != "off" {
		r.RegisterProbe(probe.K3sLocalPathStorage{})
	}
	if os.Getenv("SRENIX_PROBE_K3S_DATASTORE") != "off" {
		r.RegisterProbe(probe.K3sDatastore{})
	}
	// Core analyzers — the secret-chain / cert / image-auth set the
	// product was built around. Each defaults ON; the env gate exists
	// only so the docs' "disable any analyzer" promise holds. Do NOT
	// flip any of these defaults.
	if os.Getenv("SRENIX_ANALYZER_SECRET_KEY_MISSING") != "off" {
		r.RegisterAnalyzer(diagnose.SecretKeyMissing{})
	}
	if os.Getenv("SRENIX_ANALYZER_FAILING_EXTERNAL_SECRETS") != "off" {
		r.RegisterAnalyzer(diagnose.FailingExternalSecrets{})
	}
	if os.Getenv("SRENIX_ANALYZER_PROACTIVE_SECRET_KEY_CHECK") != "off" {
		r.RegisterAnalyzer(diagnose.ProactiveSecretKeyCheck{})
	}
	if os.Getenv("SRENIX_ANALYZER_UNPROVISIONED_SECRET") != "off" {
		r.RegisterAnalyzer(diagnose.UnprovisionedSecret{})
	}
	if os.Getenv("SRENIX_ANALYZER_IMAGE_PULL_AUTH") != "off" {
		r.RegisterAnalyzer(diagnose.ImagePullAuth{})
	}
	if os.Getenv("SRENIX_ANALYZER_FAILED_PODS") != "off" {
		r.RegisterAnalyzer(diagnose.FailedPods{})
	}
	if os.Getenv("SRENIX_ANALYZER_CERT_EXPIRY") != "off" {
		r.RegisterAnalyzer(diagnose.CertExpiry{})
	}
	if os.Getenv("SRENIX_ANALYZER_TLS_SECRET_MISMATCH") != "off" {
		r.RegisterAnalyzer(diagnose.TLSSecretMismatch{})
	}
	// v1.7 drift-class expansion (Workstreams B1+B2+B3) + v1.8
	// config + capacity + security drift (Workstreams B4+B5+B6).
	// Each opt-out via env var on clusters that don't host the
	// targeted asset class and the operator wants to silence the
	// no-target list cycle.
	if os.Getenv("SRENIX_ANALYZER_GITOPS_DRIFT") != "off" {
		r.RegisterAnalyzer(diagnose.GitOpsDrift{})
	}
	if os.Getenv("SRENIX_ANALYZER_WORKLOAD_STATE_DRIFT") != "off" {
		r.RegisterAnalyzer(diagnose.WorkloadStateDrift{})
	}
	if os.Getenv("SRENIX_ANALYZER_RBAC_DRIFT") != "off" {
		r.RegisterAnalyzer(diagnose.RBACDrift{})
	}
	if os.Getenv("SRENIX_ANALYZER_CONFIG_DRIFT") != "off" {
		r.RegisterAnalyzer(diagnose.ConfigDrift{})
	}
	if os.Getenv("SRENIX_ANALYZER_CAPACITY_DRIFT") != "off" {
		r.RegisterAnalyzer(diagnose.CapacityDrift{})
	}
	if os.Getenv("SRENIX_ANALYZER_SECURITY_DRIFT") != "off" {
		r.RegisterAnalyzer(diagnose.SecurityDrift{})
	}
	// v1.21.0 (Phase 2.E) — disruption-tier quota/PDB/Job signals.
	// Each sub-signal in DisruptionDrift handles its own GVR-absence
	// case gracefully; whole bundle opts out via env var.
	if os.Getenv("SRENIX_ANALYZER_DISRUPTION_DRIFT") != "off" {
		r.RegisterAnalyzer(diagnose.DisruptionDrift{})
	}
	// v1.22.0 (Phase 3.E) — workload-tier signals: OOMKill recurrence
	// (sizing problem masquerading as crash loop), PV orphan (cost
	// leak), CronJob stuck (silent scheduling failure). Each opts
	// out via its own env var.
	if os.Getenv("SRENIX_ANALYZER_OOMKILL_RECURRENCE") != "off" {
		r.RegisterAnalyzer(diagnose.OOMKillRecurrence{})
	}
	if os.Getenv("SRENIX_ANALYZER_PV_ORPHAN") != "off" {
		r.RegisterAnalyzer(diagnose.PVOrphan{})
	}
	if os.Getenv("SRENIX_ANALYZER_CRONJOB_STUCK") != "off" {
		r.RegisterAnalyzer(diagnose.CronJobStuck{})
	}
	// v1.25+ (M3) — LogPatternMatcher scans recent Events for
	// high-signal failure messages (ImagePullBackOff, OOMKilled,
	// VolumeAttachFailed, ProbeFailed, Forbidden). One finding per
	// (involved-object, pattern) — dedup'd so noisy event streams
	// don't dominate. Toggle off via SRENIX_ANALYZER_LOG_PATTERN_MATCHER=off.
	if os.Getenv("SRENIX_ANALYZER_LOG_PATTERN_MATCHER") != "off" {
		r.RegisterAnalyzer(diagnose.LogPatternMatcher{})
	}
	// NetworkPolicyProposer is the Phase 2d-β OSS-side hook. Silent on
	// CNIs that don't enforce NetworkPolicy (k3s-Flannel-only); on
	// enforcing CNIs it emits one warning per uncovered namespace with
	// a deterministic ProposedPolicyYAML attached. srenix-enterprise aiwatch
	// turns the proposal into an ApprovalProposal CR + Slack
	// Approve/Deny pair. Toggle off via SRENIX_ANALYZER_NETPOL_PROPOSER=off.
	if os.Getenv("SRENIX_ANALYZER_NETPOL_PROPOSER") != "off" {
		r.RegisterAnalyzer(diagnose.NetworkPolicyProposer{})
	}
	r.RegisterFixer(
		fix.StaleErrorPods{},
		fix.StuckJobsWithBadSecretRef{},
		fix.StuckRSPods{},
		fix.StuckCertificateRequests{},
	)
	// Opt-in fixers — registered only when explicitly enabled. The matching
	// Helm value flips the env var and adds the required RBAC verbs.
	if os.Getenv("SRENIX_FIXER_TLS_SECRET_MISMATCH") == "true" {
		r.RegisterFixer(fix.TLSSecretMismatch{})
	}

	// Layer-2 investigator: deterministic, rule-based, ships in OSS.
	// Disable with SRENIX_INVESTIGATOR=off; the paid binary may replace it with
	// an LLM-backed implementation after this registration runs.
	if os.Getenv("SRENIX_INVESTIGATOR") != "off" {
		r.RegisterInvestigator(investigator.RuleBased{})
	}

	// DNSChainDrift analyzer — wired when SRENIX_CLOUDFLARE_TOKEN env is set.
	// When absent, the analyzer still runs the K8s-chain hops and emits
	// "external DNS hop not verified" for each host. Opt-out via
	// SRENIX_ANALYZER_DNS_CHAIN_DRIFT=off.
	if os.Getenv("SRENIX_ANALYZER_DNS_CHAIN_DRIFT") != "off" {
		var cfClient diagnose.CloudflareClient
		if tok := os.Getenv("SRENIX_CLOUDFLARE_TOKEN"); tok != "" {
			cfClient = cfClientAdapter{inner: cloudflare.New(tok, "")}
		}
		r.RegisterAnalyzer(diagnose.DNSChainDrift{
			Client:      cfClient,
			SeedTargets: probe.DefaultEndpointHostnames(),
			OptOutAnno:  "srenix.ai/probe-disable",
		})
	}
}

// Default returns a Registry pre-loaded with the OSS pattern set.
func Default() *registry.Registry {
	r := registry.New()
	RegisterOSS(r)
	return r
}
