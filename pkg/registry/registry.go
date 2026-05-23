// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

// Package registry holds the active set of probes, analyzers, and fixers
// for a cha run.
//
// The OSS binary seeds it from the catalog package. The paid binary imports
// the same catalog and additionally registers private-tier patterns:
//
//	reg := registry.New()
//	catalog.RegisterOSS(reg)        // public module
//	paidcatalog.Register(reg)       // private module — implements pkg/ interfaces
package registry

import (
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/ai"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/cloudprobe"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/diagnose"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/fix"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/probe"
)

// Registry holds the active pattern set.
//
// The AI fields default to nil; OSS users never see them populated.
// The CHA-com paid binary registers Enricher/FixProposer/etc. at
// process start. The watcher and reporters check for nil before using
// any AI component, so an empty registry produces today's behavior
// bit-for-bit.
type Registry struct {
	analyzers   []diagnose.Analyzer
	fixers      []fix.Fixer
	probes      []probe.Probe
	cloudProbes []cloudprobe.Probe

	// AI components — all optional.
	enricher        ai.Enricher
	fixProposer     ai.FixProposer
	planner         ai.MultiStepPlanner
	runbookProposer ai.VaultRunbookProposer
	signer          ai.Signer
	verifier        ai.Verifier
	approver        ai.Approver
	auditSink       ai.AuditSink
	investigator    ai.Investigator
}

// New returns an empty Registry.
func New() *Registry { return &Registry{} }

// RegisterAnalyzer adds one or more analyzers in registration order.
func (r *Registry) RegisterAnalyzer(a ...diagnose.Analyzer) {
	r.analyzers = append(r.analyzers, a...)
}

// RegisterFixer adds one or more fixers in registration order.
func (r *Registry) RegisterFixer(f ...fix.Fixer) {
	r.fixers = append(r.fixers, f...)
}

// RegisterProbe adds one or more probes in registration order.
func (r *Registry) RegisterProbe(p ...probe.Probe) {
	r.probes = append(r.probes, p...)
}

// RegisterCloudProbe adds one or more cloud-resource probes. Catalog
// wiring (catalog/cloud.go) registers AWS/GCP/Azure probes here.
func (r *Registry) RegisterCloudProbe(p ...cloudprobe.Probe) {
	r.cloudProbes = append(r.cloudProbes, p...)
}

// RegisterEnricher sets the AI enricher (T0+). Only one may be active.
// Passing nil clears it.
func (r *Registry) RegisterEnricher(e ai.Enricher) { r.enricher = e }

// RegisterFixProposer sets the T1 single-action proposer.
func (r *Registry) RegisterFixProposer(p ai.FixProposer) { r.fixProposer = p }

// RegisterPlanner sets the T2 multi-step planner.
func (r *Registry) RegisterPlanner(p ai.MultiStepPlanner) { r.planner = p }

// RegisterRunbookProposer sets the T3 Vault runbook generator.
func (r *Registry) RegisterRunbookProposer(p ai.VaultRunbookProposer) { r.runbookProposer = p }

// RegisterSigner sets the JWT signer (approval-server).
func (r *Registry) RegisterSigner(s ai.Signer) { r.signer = s }

// RegisterVerifier sets the JWT verifier (approval-server).
func (r *Registry) RegisterVerifier(v ai.Verifier) { r.verifier = v }

// RegisterApprover sets the approval coordinator.
func (r *Registry) RegisterApprover(a ai.Approver) { r.approver = a }

// RegisterAuditSink sets the audit log sink.
func (r *Registry) RegisterAuditSink(s ai.AuditSink) { r.auditSink = s }

// RegisterInvestigator sets the Layer-2 read-only investigator. Passing nil
// disables investigation (findings surface unchanged). The OSS catalog
// registers a deterministic rule-based investigator by default; the paid
// binary may replace it with an LLM-backed implementation.
func (r *Registry) RegisterInvestigator(i ai.Investigator) { r.investigator = i }

// Analyzers returns registered analyzers in registration order.
func (r *Registry) Analyzers() []diagnose.Analyzer { return r.analyzers }

// Fixers returns registered fixers in registration order.
func (r *Registry) Fixers() []fix.Fixer { return r.fixers }

// Probes returns registered probes in registration order.
func (r *Registry) Probes() []probe.Probe { return r.probes }

// CloudProbes returns registered cloud-resource probes in registration
// order. Empty slice when no cloud providers are configured.
func (r *Registry) CloudProbes() []cloudprobe.Probe { return r.cloudProbes }

// Enricher returns the registered enricher or nil if AI is off.
func (r *Registry) Enricher() ai.Enricher { return r.enricher }

// FixProposer returns the registered T1 proposer or nil.
func (r *Registry) FixProposer() ai.FixProposer { return r.fixProposer }

// Planner returns the registered T2 planner or nil.
func (r *Registry) Planner() ai.MultiStepPlanner { return r.planner }

// RunbookProposer returns the registered T3 runbook proposer or nil.
func (r *Registry) RunbookProposer() ai.VaultRunbookProposer { return r.runbookProposer }

// Signer returns the registered JWT signer or nil.
func (r *Registry) Signer() ai.Signer { return r.signer }

// Verifier returns the registered JWT verifier or nil.
func (r *Registry) Verifier() ai.Verifier { return r.verifier }

// Approver returns the registered approval coordinator or nil.
func (r *Registry) Approver() ai.Approver { return r.approver }

// AuditSink returns the registered audit sink or nil. Callers should
// fall back to a no-op when nil; pkg/ai/noop.go provides one.
func (r *Registry) AuditSink() ai.AuditSink { return r.auditSink }

// Investigator returns the registered Layer-2 investigator or nil when
// investigation is disabled.
func (r *Registry) Investigator() ai.Investigator { return r.investigator }
