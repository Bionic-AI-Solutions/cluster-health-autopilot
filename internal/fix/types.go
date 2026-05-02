// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

// Package fix contains whitelisted auto-remediation fixers.
//
// Each fixer is a narrowly-scoped, idempotent action targeting a specific
// failure pattern with a known-safe corrective action. Fixers are gated by
// the snapshot.Mutator interface: they refuse to run when the source is
// snapshot-backed (offline mode).
//
// Cluster mutations forbidden by design: Secret writes, ConfigMap writes,
// CRD writes. Resolution for those classes is surfaced through the
// internal/diagnose analyzers instead.
//
// New fixers must:
//  1. Implement the Fixer interface.
//  2. Skip protected namespaces via IsProtectedNamespace().
//  3. Never mutate Secrets, ConfigMaps, or CRDs.
//  4. Be idempotent — repeated runs against an unchanged cluster are no-ops.
//  5. Carry comprehensive tests including a "no-op when nothing matches" case.
package fix

import (
	"context"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
)

// Action records one mutation applied during a fixer run.
type Action struct {
	// Description is the human-readable summary that ends up in the Slack
	// "Automated Fixes Applied" section.
	Description string `json:"description"`

	// Object is "Kind/namespace/name" of the mutated resource (or
	// "Kind/name" for cluster-scoped resources).
	Object string `json:"object"`
}

// SkipReason records a candidate the fixer chose NOT to act on, with reason.
type SkipReason struct {
	Object string `json:"object"`
	Reason string `json:"reason"`
}

// Result is the output of a single fixer run.
type Result struct {
	Fixer   string       `json:"fixer"`
	Actions []Action     `json:"actions,omitempty"`
	Skipped []SkipReason `json:"skipped,omitempty"`

	// Refused is set when the fixer declined to run at all (typically
	// because the source is snapshot-backed). Empty string when not refused.
	Refused string `json:"refused,omitempty"`
}

// Fixer is the contract every whitelisted auto-remediation implements.
//
// Run must be idempotent: repeated invocations against an unchanged cluster
// must produce the same result and not duplicate Actions. Mutator may be
// nil — in that case the fixer must set Result.Refused and return without
// performing any cluster I/O.
type Fixer interface {
	Name() string
	Run(ctx context.Context, src snapshot.Source, m snapshot.Mutator) Result
}
