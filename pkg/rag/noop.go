// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package rag

import "context"

// NoopReader is the OSS default Reader. Every method returns the
// zero-knowledge answer with no error. Wired by default into every
// probe/analyzer that supports a learnt source — so OSS builds compile
// against the same interface as the paid build, without paying any
// runtime cost.
//
// Calling site contract: probes/analyzers MUST be correct when their
// Learnt Reader returns nothing. The Qdrant-backed paid Reader is a pure
// ADDITION to the existing discovery/CR-override sources, never a
// replacement.
type NoopReader struct{}

// List always returns no entries and no error.
func (NoopReader) List(context.Context, Query) ([]Entry, error) { return nil, nil }

// Get always returns (nil, nil) — no entry, no error.
func (NoopReader) Get(context.Context, EntryKind, string) (*Entry, error) { return nil, nil }

// NoopWriter is the OSS default Writer. Drops every write; never errors.
// Keeps the wiring symmetric with NoopReader so OSS builds compile
// identically to the paid build.
type NoopWriter struct{}

// Upsert drops the entry on the floor.
func (NoopWriter) Upsert(context.Context, Entry) error { return nil }

// AppendSignal drops the event on the floor.
func (NoopWriter) AppendSignal(context.Context, EntryKind, string, SignalEvent) error { return nil }

// Decay is a no-op.
func (NoopWriter) Decay(context.Context) error { return nil }

// Compile-time guards: NoopReader/NoopWriter satisfy Reader/Writer.
var (
	_ Reader = NoopReader{}
	_ Writer = NoopWriter{}
)
