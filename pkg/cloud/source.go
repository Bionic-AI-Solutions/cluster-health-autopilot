// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

// Package cloud is the cloud-API counterpart to pkg/snapshot. A
// cloud.Source exposes per-provider sub-clients (AWS / GCP / Azure)
// scoped to the narrow set of read operations cloud probes actually
// need. Probes import only the sub-client(s) they use — a probe that
// only touches AWS holds an aws.Client and never imports gcp or
// azure code paths.
//
// Sub-client packages live alongside this one (pkg/cloud/aws,
// pkg/cloud/gcp, pkg/cloud/azure) and wrap the official Go SDKs.
//
// Mode reports whether the source is backed by a live cloud API
// session or a snapshot bundle captured via `srenix snapshot capture
// --include-cloud`. Probes treat snapshot mode as point-in-time: the
// captured-at timestamp is surfaced in DriftReport messages so
// operators see "as of …" context for cloud findings.
//
// See docs/design/2026-05-cloud-probe-framework.md for the full
// design, rollout phases, and the auth model (IRSA / GCP Workload
// Identity / Azure AAD Workload Identity / static creds escape hatch).
package cloud

import (
	"github.com/srenix-ai/agentic-sre/pkg/cloud/aws"
	"github.com/srenix-ai/agentic-sre/pkg/cloud/azure"
	"github.com/srenix-ai/agentic-sre/pkg/cloud/gcp"
)

// Source is the cloud-API counterpart to snapshot.Source. Probes call
// the sub-client they need; nil returns mean "not configured" and the
// probe should refuse (not panic).
//
// Implementations:
//   - Live (planned, internal/cloud) — wraps real SDK sessions
//   - Snapshot (planned, internal/cloud) — replays captured JSON
//   - Fake (test) — returns canned responses per-method
type Source interface {
	// AWS returns the AWS sub-client, or nil if AWS is not configured.
	// Probes that find nil should mark themselves SKIPPED, never fail.
	AWS() aws.Client

	// GCP returns the GCP sub-client, or nil if GCP is not configured.
	GCP() gcp.Client

	// Azure returns the Azure sub-client, or nil if Azure is not configured.
	Azure() azure.Client

	// Mode reports whether the source is backed by a live API session
	// or a captured snapshot bundle.
	Mode() Mode
}

// Mode reports whether a Source is backed by a live cloud API or a
// snapshot bundle. Mirrors pkg/snapshot.Mode.
type Mode int

// Mode constants.
const (
	// ModeLive — backed by real SDK sessions; results reflect the
	// cloud's current state at call time.
	ModeLive Mode = iota

	// ModeSnapshot — backed by a JSON bundle captured at some prior
	// time. Probes should annotate findings with the capturedAt
	// timestamp so operators interpret them as point-in-time.
	ModeSnapshot
)

// String renders the mode for log lines and DriftReport messages.
func (m Mode) String() string {
	switch m {
	case ModeLive:
		return "live"
	case ModeSnapshot:
		return "snapshot"
	default:
		return "unknown"
	}
}
