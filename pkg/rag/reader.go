// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package rag

import "context"

// Reader is the read side of the cluster RAG memory layer.
//
// Implementations:
//
//   - NoopReader (this package, OSS): never returns anything. Wired by
//     default into every probe/analyzer that supports a learnt source.
//     OSS clusters use this; their probes work exactly as they did pre-2d.
//
//   - QdrantReader (srenix-enterprise, paid AI tier): backed by per-cluster Qdrant.
//     Replaces NoopReader at operator startup when spec.ai.enabled=true.
//
// Failure-mode contract: implementations MUST return (nil, err) on
// transport failures, NOT a zero-value entry slice plus err. Callers in the
// probe loop check for nil-or-error and fall through to "no learnt
// entries this cycle" — a Qdrant outage must not break probes.
type Reader interface {
	// List returns entries matching q, sorted by Importance descending.
	// Empty q returns all entries in the cluster scope.
	// Empty result is (nil, nil) — distinguishable from error.
	List(ctx context.Context, q Query) ([]Entry, error)

	// Get returns a single Entry by (Kind, Key) or (nil, nil) when no
	// such entry exists. Errors are reserved for transport failures.
	Get(ctx context.Context, kind EntryKind, key string) (*Entry, error)
}

// Writer is the write side; only the paid AI tier links a real
// implementation. OSS keeps NoopWriter to make wiring symmetric.
//
// Three feeders call Writer in production (see design doc §Write path):
//
//  1. Discovery feeder       — startup + 24h: Cloudflare zones, workload walk
//  2. Observation feeder     — per analyzer cycle: open / resolved events
//  3. Outcome feeder         — real-time: approval-server clicks, Silence CRs
type Writer interface {
	// Upsert creates or updates an Entry. The implementation is responsible
	// for FirstSeen preservation, LastSeen refresh, Observations increment,
	// and importance-smoothing logic.
	Upsert(ctx context.Context, e Entry) error

	// AppendSignal records one SignalEvent against an existing (kind, key)
	// entry. Implementations MAY create the entry on demand if it doesn't
	// exist yet (e.g. first-ever silence event for a never-before-seen
	// finding class). Bounded log retention is the implementation's job.
	AppendSignal(ctx context.Context, kind EntryKind, key string, ev SignalEvent) error

	// Decay applies importance decay to entries whose LastSeen is older
	// than the configured halflife. Called by the operator's periodic
	// maintenance loop, not from the hot path.
	Decay(ctx context.Context) error
}
