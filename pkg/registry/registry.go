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
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/diagnose"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/fix"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/probe"
)

// Registry holds the active pattern set.
type Registry struct {
	analyzers []diagnose.Analyzer
	fixers    []fix.Fixer
	probes    []probe.Probe
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

// Analyzers returns registered analyzers in registration order.
func (r *Registry) Analyzers() []diagnose.Analyzer { return r.analyzers }

// Fixers returns registered fixers in registration order.
func (r *Registry) Fixers() []fix.Fixer { return r.fixers }

// Probes returns registered probes in registration order.
func (r *Registry) Probes() []probe.Probe { return r.probes }
