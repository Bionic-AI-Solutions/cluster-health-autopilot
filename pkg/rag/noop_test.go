// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package rag

import (
	"context"
	"testing"
)

// TestNoopReader_ListAlwaysEmpty — the OSS default must never surface a
// learnt entry. This is the load-bearing guarantee that OSS users don't
// silently activate a paid-tier feature.
func TestNoopReader_ListAlwaysEmpty(t *testing.T) {
	r := NoopReader{}
	ctx := context.Background()

	cases := []Query{
		{},
		{Kind: KindApexDomain},
		{Kind: KindWorkload, ImportanceMin: 0.0},
		{Kind: KindFindingOutcome, ImportanceMin: 1.0},
		{Source: "cloudflare_zone", Limit: 100},
	}
	for _, q := range cases {
		got, err := r.List(ctx, q)
		if err != nil {
			t.Errorf("List(%+v) returned err=%v; NoopReader must never error", q, err)
		}
		if len(got) != 0 {
			t.Errorf("List(%+v) returned %d entries; NoopReader must return empty", q, len(got))
		}
	}
}

// TestNoopReader_GetAlwaysNil — Get must never invent entries.
func TestNoopReader_GetAlwaysNil(t *testing.T) {
	r := NoopReader{}
	ctx := context.Background()

	got, err := r.Get(ctx, KindApexDomain, "any.example.com")
	if err != nil {
		t.Errorf("Get returned err=%v; NoopReader must never error", err)
	}
	if got != nil {
		t.Errorf("Get returned %+v; NoopReader must return nil", got)
	}
}

// TestNoopWriter_AllOpsSucceedSilently — writes drop on the floor without
// error. Probes that opportunistically call Upsert/AppendSignal in OSS
// must not see a failure.
func TestNoopWriter_AllOpsSucceedSilently(t *testing.T) {
	w := NoopWriter{}
	ctx := context.Background()

	if err := w.Upsert(ctx, Entry{Kind: KindApexDomain, Key: "x"}); err != nil {
		t.Errorf("Upsert: %v", err)
	}
	if err := w.AppendSignal(ctx, KindFindingOutcome, "y", SignalEvent{Action: "approved"}); err != nil {
		t.Errorf("AppendSignal: %v", err)
	}
	if err := w.Decay(ctx); err != nil {
		t.Errorf("Decay: %v", err)
	}
}

// TestEntryKind_AllConstantsExported — guard against typos / accidental
// renames in the constant set. The kind string is the partition key in
// Qdrant; renames need a migration.
func TestEntryKind_AllConstantsExported(t *testing.T) {
	want := map[EntryKind]struct{}{
		"apex_domain":     {},
		"workload":        {},
		"finding_outcome": {},
		"baseline":        {},
		"silenced_class":  {},
	}
	have := []EntryKind{
		KindApexDomain,
		KindWorkload,
		KindFindingOutcome,
		KindBaseline,
		KindSilencedClass,
	}
	for _, k := range have {
		if _, ok := want[k]; !ok {
			t.Errorf("EntryKind %q is exported but not in the wire-stable set", k)
		}
		delete(want, k)
	}
	for k := range want {
		t.Errorf("EntryKind %q is in the wire-stable set but not exported as a const", k)
	}
}
