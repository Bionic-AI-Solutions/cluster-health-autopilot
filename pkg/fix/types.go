// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

// Package fix defines the Fixer interface and associated types that form
// the exported API surface for the CHA pattern registry.
//
// External pattern catalogs (paid tier, community plugins) implement Fixer
// and register their implementations via pkg/registry.
package fix

import (
	"context"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/snapshot"
)

// Action records one mutation applied during a fixer run.
type Action struct {
	// Description is the human-readable summary for the Slack report.
	Description string `json:"description"`

	// Object is "Kind/namespace/name" of the mutated resource.
	Object string `json:"object"`
}

// SkipReason records a candidate the fixer chose not to act on, with reason.
type SkipReason struct {
	Object string `json:"object"`
	Reason string `json:"reason"`
}

// Result is the output of a single fixer run.
type Result struct {
	Fixer   string       `json:"fixer"`
	Actions []Action     `json:"actions,omitempty"`
	Skipped []SkipReason `json:"skipped,omitempty"`

	// Refused is set when the fixer declined to run (e.g. snapshot mode).
	Refused string `json:"refused,omitempty"`
}

// Fixer is the contract every whitelisted auto-remediation must implement.
//
// A Fixer must:
//   - Be idempotent — repeated runs against an unchanged cluster produce the same result.
//   - Skip protected namespaces (kube-system, etc.).
//   - Never mutate Secrets, ConfigMaps, or CRDs.
//   - Set Result.Refused and return without cluster I/O when m is nil (snapshot mode).
type Fixer interface {
	Name() string
	Run(ctx context.Context, src snapshot.Source, m snapshot.Mutator) Result
}
