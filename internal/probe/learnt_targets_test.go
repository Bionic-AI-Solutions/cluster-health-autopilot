// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package probe

import (
	"context"
	"errors"
	"reflect"
	"sort"
	"testing"

	"github.com/srenix-ai/agentic-sre/pkg/rag"
)

// fakeRAGReader is a deterministic Reader for testing the learnt-targets
// fan-in. Returns the pre-seeded entries verbatim from List; ignores the
// Query filter except for an explicit error injection.
type fakeRAGReader struct {
	entries []rag.Entry
	err     error
}

func (f fakeRAGReader) List(context.Context, rag.Query) ([]rag.Entry, error) {
	return f.entries, f.err
}
func (fakeRAGReader) Get(context.Context, rag.EntryKind, string) (*rag.Entry, error) {
	return nil, nil
}

func TestLearntApexTargets_NilReaderReturnsNil(t *testing.T) {
	got := learntApexTargets(context.Background(), nil, 0.3, nil)
	if got != nil {
		t.Errorf("nil Reader must produce no targets; got %+v", got)
	}
}

func TestLearntApexTargets_NoopReaderReturnsEmpty(t *testing.T) {
	got := learntApexTargets(context.Background(), rag.NoopReader{}, 0.3, nil)
	if len(got) != 0 {
		t.Errorf("NoopReader (OSS default) must produce no targets; got %+v", got)
	}
}

func TestLearntApexTargets_HappyPath(t *testing.T) {
	r := fakeRAGReader{
		entries: []rag.Entry{
			{Kind: rag.KindApexDomain, Key: "API.Example.COM", Importance: 0.8, Features: map[string]any{"display_name": "Example API"}},
			{Kind: rag.KindApexDomain, Key: "marketing.example.com", Importance: 0.5},
		},
	}
	got := learntApexTargets(context.Background(), r, 0.3, nil)
	if len(got) != 2 {
		t.Fatalf("expected 2 targets; got %d: %+v", len(got), got)
	}
	if got[0].URL != "https://api.example.com" {
		t.Errorf("expected lowercased URL; got %q", got[0].URL)
	}
	if got[0].Name != "Example API" {
		t.Errorf("expected display_name from features; got %q", got[0].Name)
	}
	if got[1].Name != "marketing.example.com (learnt)" {
		t.Errorf("expected fallback display name; got %q", got[1].Name)
	}
}

func TestLearntApexTargets_DedupesAgainstKnownHosts(t *testing.T) {
	r := fakeRAGReader{
		entries: []rag.Entry{
			{Kind: rag.KindApexDomain, Key: "api.example.com", Importance: 0.9},
			{Kind: rag.KindApexDomain, Key: "fresh.example.com", Importance: 0.9},
		},
	}
	// "api.example.com" is already in alreadyKnown (came from static or
	// ingress discovery), so the learnt fan-in must skip it.
	known := []string{"api.example.com"}
	got := learntApexTargets(context.Background(), r, 0.3, known)
	if len(got) != 1 || got[0].URL != "https://fresh.example.com" {
		t.Errorf("expected only fresh.example.com after de-dup; got %+v", got)
	}
}

func TestLearntApexTargets_ReaderErrorReturnsNil(t *testing.T) {
	// Hard requirement: a Qdrant outage MUST NOT break probes — pkg/rag
	// Reader contract. learntApexTargets returns nil on any error and
	// the probe falls through to its static + ingress-discovered paths.
	r := fakeRAGReader{err: errors.New("qdrant unreachable")}
	got := learntApexTargets(context.Background(), r, 0.3, nil)
	if got != nil {
		t.Errorf("reader error must produce nil targets; got %+v", got)
	}
}

func TestLearntApexTargets_SkipsEmptyKeys(t *testing.T) {
	r := fakeRAGReader{
		entries: []rag.Entry{
			{Kind: rag.KindApexDomain, Key: "", Importance: 0.9},
			{Kind: rag.KindApexDomain, Key: "   ", Importance: 0.9},
			{Kind: rag.KindApexDomain, Key: "real.example.com", Importance: 0.9},
		},
	}
	got := learntApexTargets(context.Background(), r, 0.3, nil)
	urls := make([]string, 0, len(got))
	for _, t := range got {
		urls = append(urls, t.URL)
	}
	sort.Strings(urls)
	want := []string{"https://real.example.com"}
	if !reflect.DeepEqual(urls, want) {
		t.Errorf("expected only the non-empty key to survive; got %+v", urls)
	}
}

func TestHostOf(t *testing.T) {
	cases := map[string]string{
		"https://example.com":            "example.com",
		"https://api.example.com/v1/foo": "api.example.com",
		"http://EXAMPLE.com?x=1":         "example.com",
		"raw-host-no-scheme":             "raw-host-no-scheme",
		"https://example.com:8443/path":  "example.com:8443",
	}
	for in, want := range cases {
		if got := hostOf(in); got != want {
			t.Errorf("hostOf(%q) = %q; want %q", in, got, want)
		}
	}
}
