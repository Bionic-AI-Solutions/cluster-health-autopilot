// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package audit

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/srenix-ai/agentic-sre/pkg/ai"
)

// memorySink is a slice-backed AuditSink used in tests. Captures the
// events as the chained sink delegates so the verifier has something
// to walk.
type memorySink struct {
	mu     sync.Mutex
	events []ai.AuditEvent
}

func (m *memorySink) Write(_ context.Context, e ai.AuditEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, cloneAuditEvent(e))
	return nil
}

func (m *memorySink) snapshot() []ai.AuditEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]ai.AuditEvent, len(m.events))
	for i, e := range m.events {
		out[i] = cloneAuditEvent(e)
	}
	return out
}

type failingSink struct{}

func (failingSink) Write(_ context.Context, _ ai.AuditEvent) error {
	return errors.New("sink down")
}

// ed25519CheckpointSigner is a test CheckpointSigner backed by a freshly
// generated key (tests NEVER use real key material).
type ed25519CheckpointSigner struct {
	priv ed25519.PrivateKey
	kid  string
}

func newTestSigner(t *testing.T) (*ed25519CheckpointSigner, ed25519.PublicKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("genkey: %v", err)
	}
	return &ed25519CheckpointSigner{priv: priv, kid: "test-1"}, pub
}

func (s *ed25519CheckpointSigner) SignCheckpoint(data []byte) (string, string, error) {
	return base64.StdEncoding.EncodeToString(ed25519.Sign(s.priv, data)), s.kid, nil
}

func writeN(t *testing.T, s *ChainedSink, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		e := ai.AuditEvent{
			Type:    "ai.proposal.created",
			Actor:   "srenix",
			Subject: "Pod/demo/x" + string(rune('0'+i)),
		}
		if err := s.Write(context.Background(), e); err != nil {
			t.Fatalf("write %d: %v", i, err)
		}
	}
}

func TestChainedSink_AnchorIsEmptyHash(t *testing.T) {
	mem := &memorySink{}
	s := NewChainedSink(mem)
	e := ai.AuditEvent{Type: "ai.proposal.created", Actor: "srenix"}
	if err := s.Write(context.Background(), e); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got := mem.events[0].Details["prev_hash"]
	if got != "" {
		t.Errorf("first chain link should anchor to empty hash; got %q", got)
	}
}

func TestChainedSink_LinksAreConsistent(t *testing.T) {
	mem := &memorySink{}
	s := NewChainedSink(mem)
	writeN(t, s, 5)
	snap := mem.snapshot()
	if len(snap) != 5 {
		t.Fatalf("got %d events, want 5", len(snap))
	}
	// Each entry's prev_hash should equal the previous entry's entry_hash.
	for i := 1; i < 5; i++ {
		prevEntry := snap[i].Details["prev_hash"]
		prevHash := snap[i-1].Details["entry_hash"]
		if prevEntry != prevHash {
			t.Errorf("entry %d prev_hash=%v, want %v", i, prevEntry, prevHash)
		}
	}
	// LastHash reflects the final entry's self-hash.
	if got, want := s.LastHash(), snap[4].Details["entry_hash"]; got != want {
		t.Errorf("LastHash=%q, want %v", got, want)
	}
}

func TestVerifyChain_IntactChain(t *testing.T) {
	mem := &memorySink{}
	s := NewChainedSink(mem)
	writeN(t, s, 3)
	idx, err := VerifyChain(mem.snapshot())
	if err != nil {
		t.Fatalf("VerifyChain: %v", err)
	}
	if idx != -1 {
		t.Errorf("intact chain should return -1, got break at index %d", idx)
	}
}

// Tamper detection at EVERY position, for every tamper class: content
// mutation, Details mutation, entry_hash forgery, prev_hash forgery.
func TestVerifyChain_DetectsTamperAtEveryPosition(t *testing.T) {
	const n = 5
	build := func() []ai.AuditEvent {
		mem := &memorySink{}
		s := NewChainedSink(mem)
		writeN(t, s, n)
		return mem.snapshot()
	}
	tampers := map[string]func(e *ai.AuditEvent){
		"struct field": func(e *ai.AuditEvent) { e.Actor = "attacker" },
		"details key":  func(e *ai.AuditEvent) { e.Details["injected"] = true },
		"entry_hash":   func(e *ai.AuditEvent) { e.Details["entry_hash"] = "deadbeef" },
		"prev_hash":    func(e *ai.AuditEvent) { e.Details["prev_hash"] = "deadbeef" },
		"timestamp":    func(e *ai.AuditEvent) { e.Details[EntryTimeKey] = "1999-01-01T00:00:00Z" },
	}
	for name, tamper := range tampers {
		for pos := 0; pos < n; pos++ {
			snap := build()
			tamper(&snap[pos])
			idx, err := VerifyChain(snap)
			if err != nil {
				t.Fatalf("%s@%d: VerifyChain error: %v", name, pos, err)
			}
			if idx != pos {
				t.Errorf("%s@%d: verifier flagged index %d, want %d", name, pos, idx, pos)
			}
		}
	}
}

func TestVerifyChain_DetectsReordering(t *testing.T) {
	mem := &memorySink{}
	s := NewChainedSink(mem)
	writeN(t, s, 3)
	snap := mem.snapshot()
	// Swap entry 1 and entry 2.
	snap[1], snap[2] = snap[2], snap[1]
	idx, _ := VerifyChain(snap)
	if idx < 0 {
		t.Error("reordering should break the chain; verifier said intact")
	}
}

func TestVerifyChain_MissingDetailsIsError(t *testing.T) {
	events := []ai.AuditEvent{{Type: "ai.x"}}
	idx, err := VerifyChain(events)
	if err == nil {
		t.Error("an unchained event (nil Details) should be reported as an error")
	}
	if idx != 0 {
		t.Errorf("idx=%d, want 0", idx)
	}
}

func TestChainedSink_InnerErrorDoesNotAdvanceChain(t *testing.T) {
	s := NewChainedSink(failingSink{})
	if err := s.Write(context.Background(), ai.AuditEvent{Type: "ai.x"}); err == nil {
		t.Fatal("expected failingSink error")
	}
	if s.LastHash() != "" {
		t.Errorf("chain advanced past a failed write; LastHash=%q want \"\"", s.LastHash())
	}
}

func TestChainedSink_NilInnerFallsBackToNoOp(t *testing.T) {
	s := NewChainedSink(nil)
	// Should write successfully (NoOp sink).
	if err := s.Write(context.Background(), ai.AuditEvent{Type: "ai.x"}); err != nil {
		t.Errorf("nil inner should fall back to NoOp, not error; got %v", err)
	}
	if s.LastHash() == "" {
		t.Error("chain should advance even with no-op sink")
	}
}

// Every entry carries an entry_time timestamp inside the hashed payload.
func TestChainedSink_EntriesCarryTimestamp(t *testing.T) {
	mem := &memorySink{}
	s := NewChainedSink(mem)
	writeN(t, s, 3)
	for i, e := range mem.snapshot() {
		ts, _ := e.Details[EntryTimeKey].(string)
		if ts == "" {
			t.Errorf("entry %d has no %s", i, EntryTimeKey)
			continue
		}
		if _, err := time.Parse(time.RFC3339Nano, ts); err != nil {
			t.Errorf("entry %d %s=%q is not RFC3339Nano: %v", i, EntryTimeKey, ts, err)
		}
	}
}

// A replayed / re-chained entry keeps its original entry_time rather
// than being re-stamped.
func TestChainedSink_PreexistingTimestampPreserved(t *testing.T) {
	mem := &memorySink{}
	s := NewChainedSink(mem)
	const orig = "2026-01-02T03:04:05.000000006Z"
	e := ai.AuditEvent{Type: "ai.x", Details: map[string]any{EntryTimeKey: orig}}
	if err := s.Write(context.Background(), e); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if got := mem.snapshot()[0].Details[EntryTimeKey]; got != orig {
		t.Errorf("pre-existing %s was overwritten: got %v, want %q", EntryTimeKey, got, orig)
	}
}

// With a fixed clock and identical inputs, two independent sinks produce
// byte-identical chains — the hash is fully deterministic. The clock is
// injected per-sink (ChainedSink.now), not via package-level state, so
// this test cannot interfere with parallel tests.
func TestChainedSink_DeterministicWithFixedClock(t *testing.T) {
	fixed := time.Date(2026, 6, 12, 10, 0, 0, 0, time.UTC)

	run := func() []ai.AuditEvent {
		mem := &memorySink{}
		s := NewChainedSink(mem)
		s.now = func() time.Time { return fixed }
		writeN(t, s, 3)
		return mem.snapshot()
	}
	a, b := run(), run()
	for i := range a {
		if a[i].Details["entry_hash"] != b[i].Details["entry_hash"] {
			t.Errorf("entry %d hash differs between identical runs: %v vs %v",
				i, a[i].Details["entry_hash"], b[i].Details["entry_hash"])
		}
	}
}

// Canonicalization is independent of map insertion order: encoding/json
// sorts map keys, so two Details maps with the same content but
// different construction order canonicalize identically.
func TestCanonicalJSON_MapOrderStability(t *testing.T) {
	d1 := map[string]any{}
	for _, k := range []string{"alpha", "beta", "gamma", "delta", "zz", "a1"} {
		d1[k] = k + "-v"
	}
	d2 := map[string]any{}
	for _, k := range []string{"a1", "zz", "delta", "gamma", "beta", "alpha"} {
		d2[k] = k + "-v"
	}
	e1 := ai.AuditEvent{Type: "ai.x", Details: d1}
	e2 := ai.AuditEvent{Type: "ai.x", Details: d2}
	c1, err := canonicalJSON(e1)
	if err != nil {
		t.Fatalf("canonicalJSON: %v", err)
	}
	c2, err := canonicalJSON(e2)
	if err != nil {
		t.Fatalf("canonicalJSON: %v", err)
	}
	if !bytes.Equal(c1, c2) {
		t.Errorf("canonical bytes differ by insertion order:\n%s\n%s", c1, c2)
	}
}

// Unicode payloads (multi-byte runes, combining characters, emoji,
// RTL text) canonicalize deterministically across repeated calls and
// survive chain verification.
func TestCanonicalJSON_UnicodeStability(t *testing.T) {
	details := map[string]any{
		"latin-1":   "naïve façade",
		"cjk":       "日本語のテキスト",
		"emoji":     "🚨 alert 🔗",
		"combining": "éclair", // e + COMBINING ACUTE ACCENT
		"rtl":       "שלום עולם",
		"escape":    "line1\nline2\t\"quoted\" <html>",
	}
	e := ai.AuditEvent{Type: "ai.unicode", Actor: "srenix", Details: details}
	first, err := canonicalJSON(e)
	if err != nil {
		t.Fatalf("canonicalJSON: %v", err)
	}
	for i := 0; i < 10; i++ {
		again, err := canonicalJSON(e)
		if err != nil {
			t.Fatalf("canonicalJSON #%d: %v", i, err)
		}
		if !bytes.Equal(first, again) {
			t.Fatalf("canonical bytes unstable on call %d", i)
		}
	}
	// And the full write→verify round trip holds for unicode content.
	mem := &memorySink{}
	s := NewChainedSink(mem)
	if err := s.Write(context.Background(), e); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if idx, err := VerifyChain(mem.snapshot()); err != nil || idx != -1 {
		t.Errorf("unicode chain should verify; idx=%d err=%v", idx, err)
	}
}

// Nested Details values (maps inside maps, slices) canonicalize
// deterministically too — nested map keys are also sorted.
func TestCanonicalJSON_NestedDetailsStability(t *testing.T) {
	build := func(order []string) ai.AuditEvent {
		inner := map[string]any{}
		for _, k := range order {
			inner[k] = len(k)
		}
		return ai.AuditEvent{Type: "ai.x", Details: map[string]any{
			"nested": inner,
			"list":   []any{"a", 1, true},
		}}
	}
	c1, err := canonicalJSON(build([]string{"x", "y", "z"}))
	if err != nil {
		t.Fatalf("canonicalJSON: %v", err)
	}
	c2, err := canonicalJSON(build([]string{"z", "y", "x"}))
	if err != nil {
		t.Fatalf("canonicalJSON: %v", err)
	}
	if !bytes.Equal(c1, c2) {
		t.Errorf("nested canonical bytes differ:\n%s\n%s", c1, c2)
	}
}

// TestCanonicalJSON_FormatContract pins the EXACT canonical bytes — the
// frozen cross-version contract that external verifiers (and the
// production Srenix Enterprise binaries already writing chains in this form) must
// replicate byte-for-byte: struct fields in declaration order, map keys
// sorted, HTML-escaping ON (<, >, & as backslash-u escapes), RFC3339Nano
// timestamps. If this test fails, the canonical format changed and every
// existing chain entry containing <, >, or & would fail verification
// across the version boundary. Do NOT update the golden string to make
// it pass — revert the format change instead.
func TestCanonicalJSON_FormatContract(t *testing.T) {
	e := ai.AuditEvent{
		Type:  "ai.contract",
		Actor: "a<b>&c",
		Details: map[string]any{
			"zeta":  "<script>",
			"alpha": "x&y",
			"html":  `<a href="u">&amp;</a>`,
		},
	}
	const golden = `{"type":"ai.contract","correlation_id":"","tier":"","actor":"a\u003cb\u003e\u0026c","details":{"alpha":"x\u0026y","html":"\u003ca href=\"u\"\u003e\u0026amp;\u003c/a\u003e","zeta":"\u003cscript\u003e"}}`
	got, err := canonicalJSON(e)
	if err != nil {
		t.Fatalf("canonicalJSON: %v", err)
	}
	if string(got) != golden {
		t.Errorf("canonical format changed — this BREAKS verification of existing production chains:\n got: %s\nwant: %s", got, golden)
	}

	// Cross-verifiability spot check: the canonical bytes are exactly
	// plain json.Marshal output — the code path production binaries have
	// always hashed — so an event containing <, >, & hashes identically
	// on the old and new code paths.
	old, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if !bytes.Equal(got, old) {
		t.Errorf("canonicalJSON diverged from json.Marshal:\n new: %s\n old: %s", got, old)
	}
	if sha256.Sum256(got) != sha256.Sum256(old) {
		t.Error("old-path and new-path hashes differ for an event containing <, >, &")
	}
}

// Resumption: a sink constructed with NewChainedSinkResuming continues
// the chain from the prior segment's tail instead of re-anchoring at "".
func TestNewChainedSinkResuming_ContinuesChain(t *testing.T) {
	mem := &memorySink{}
	seg1 := NewChainedSink(mem)
	writeN(t, seg1, 2)
	tail := seg1.LastHash()

	seg2 := NewChainedSinkResuming(mem, tail, ChainOptions{})
	writeN(t, seg2, 2)

	events := mem.snapshot()
	if len(events) != 4 {
		t.Fatalf("got %d events, want 4", len(events))
	}
	// The whole stream — across both sink lifetimes — must verify as one
	// continuous chain. A fresh "" segment in the middle would break it.
	if idx, err := VerifyChain(events); err != nil || idx != -1 {
		t.Fatalf("resumed chain should be continuous; idx=%d err=%v", idx, err)
	}
	// Belt-and-braces: no entry after index 0 may anchor to the empty
	// hash (that would be a chain restart).
	for i := 1; i < len(events); i++ {
		if ph, _ := events[i].Details["prev_hash"].(string); ph == "" {
			t.Errorf("entry %d restarted the chain (prev_hash=\"\")", i)
		}
	}
}

// Resuming from "" behaves exactly like a fresh chain (empty/absent
// prior storage anchors to the empty hash).
func TestNewChainedSinkResuming_EmptyResumeIsFreshAnchor(t *testing.T) {
	mem := &memorySink{}
	s := NewChainedSinkResuming(mem, "", ChainOptions{})
	writeN(t, s, 1)
	if got := mem.snapshot()[0].Details["prev_hash"]; got != "" {
		t.Errorf("prev_hash=%v, want \"\"", got)
	}
}

// The checkpoint cadence emits a chained checkpoint entry every
// CheckpointEvery data events, and the whole stream still verifies.
func TestNewChainedSinkResuming_CheckpointCadence(t *testing.T) {
	mem := &memorySink{}
	signer, _ := newTestSigner(t)
	s := NewChainedSinkResuming(mem, "", ChainOptions{Signer: signer, CheckpointEvery: 2})
	writeN(t, s, 4)

	events := mem.snapshot()
	var checkpoints int
	for _, e := range events {
		if e.Type == CheckpointType {
			checkpoints++
		}
	}
	if checkpoints != 2 {
		t.Errorf("got %d checkpoints, want 2 (cadence=2 over 4 data events)", checkpoints)
	}
	if idx, err := VerifyChain(events); err != nil || idx != -1 {
		t.Errorf("chain with interleaved checkpoints should verify; idx=%d err=%v", idx, err)
	}
}

// Truncating the tail past the last signed checkpoint is reported as an
// unanchored tail by VerifyChainWithCheckpoints, EVEN THOUGH the
// surviving prefix still links cleanly (which plain VerifyChain cannot
// detect).
func TestVerifyChainWithCheckpoints_DetectsTailTruncation(t *testing.T) {
	mem := &memorySink{}
	signer, _ := newTestSigner(t)
	s := NewChainedSinkResuming(mem, "", ChainOptions{Signer: signer, CheckpointEvery: 2})
	writeN(t, s, 6)
	// Final checkpoint anchoring the tail (a closer would emit this).
	s.WriteCheckpoint(context.Background())

	events := mem.snapshot()

	// Sanity: the intact log verifies AND its tail is anchored (last
	// entry is the signed closing checkpoint).
	full := VerifyChainWithCheckpoints(events)
	if full.BrokenIndex != -1 {
		t.Fatalf("intact log should not be broken; got %d", full.BrokenIndex)
	}
	if !full.TailAnchored {
		t.Fatal("intact log tail should be anchored by the closing checkpoint")
	}

	// Simulate an attacker lopping off the tail to erase recent events:
	// truncate so the surviving log ends on a DATA event (every trailing
	// checkpoint anchor is gone).
	lastData := -1
	for i, e := range events {
		if e.Type != CheckpointType {
			lastData = i
		}
	}
	if lastData < 0 {
		t.Fatal("expected at least one data entry")
	}
	cut := events[:lastData+1]
	// The surviving prefix still links cleanly — plain VerifyChain CANNOT
	// detect that the tail was truncated.
	if plainIdx, _ := VerifyChain(cut); plainIdx != -1 {
		t.Fatalf("plain VerifyChain should still see the truncated prefix as intact; got break at %d", plainIdx)
	}
	// ...but the checkpoint-aware verifier reports the tail as unanchored.
	if VerifyChainWithCheckpoints(cut).TailAnchored {
		t.Error("truncating the tail to a data event must leave it UNanchored")
	}
}

// A broken chain reports BrokenIndex AND LastCheckpointIndex together:
// the checkpoint scan covers the verified prefix (entries before the
// break), so the caller learns the last trustworthy anchor even when
// verification fails. Checkpoint at index 1, tamper at index 3 →
// BrokenIndex=3, LastCheckpointIndex=1.
func TestVerifyChainWithCheckpoints_BrokenChainReportsLastCheckpoint(t *testing.T) {
	mem := &memorySink{}
	signer, _ := newTestSigner(t)
	s := NewChainedSinkResuming(mem, "", ChainOptions{Signer: signer, CheckpointEvery: 1})
	writeN(t, s, 2) // cadence=1 → [data0, checkpoint1, data2, checkpoint3]

	events := mem.snapshot()
	if len(events) != 4 {
		t.Fatalf("got %d events, want 4 (data, checkpoint, data, checkpoint)", len(events))
	}
	if events[1].Type != CheckpointType || events[3].Type != CheckpointType {
		t.Fatalf("expected checkpoints at indexes 1 and 3; got types %q, %q", events[1].Type, events[3].Type)
	}

	events[3].Actor = "attacker" // tamper at index 3

	res := VerifyChainWithCheckpoints(events)
	if res.BrokenIndex != 3 {
		t.Errorf("BrokenIndex=%d, want 3", res.BrokenIndex)
	}
	if res.LastCheckpointIndex != 1 {
		t.Errorf("LastCheckpointIndex=%d, want 1 (the checkpoint at the break is untrustworthy and must not count)", res.LastCheckpointIndex)
	}
	if res.TailAnchored {
		t.Error("a broken chain must not report an anchored tail")
	}
}

// Removing the only checkpoint entirely also yields an unanchored tail
// (no anchor → cannot bound truncation).
func TestVerifyChainWithCheckpoints_NoCheckpointIsUnanchored(t *testing.T) {
	mem := &memorySink{}
	s := NewChainedSink(mem) // no signer, no checkpoints
	writeN(t, s, 3)
	res := VerifyChainWithCheckpoints(mem.snapshot())
	if res.BrokenIndex != -1 {
		t.Fatalf("links should be intact; got %d", res.BrokenIndex)
	}
	if res.LastCheckpointIndex != -1 {
		t.Errorf("LastCheckpointIndex=%d, want -1", res.LastCheckpointIndex)
	}
	if res.TailAnchored {
		t.Error("a chain with no checkpoint must report an unanchored tail")
	}
}

// An UNSIGNED checkpoint (no signer) does not anchor the tail: without
// a signature there is no external proof of the head hash.
func TestVerifyChainWithCheckpoints_UnsignedCheckpointNotAnchored(t *testing.T) {
	mem := &memorySink{}
	s := NewChainedSinkResuming(mem, "", ChainOptions{}) // nil signer
	writeN(t, s, 2)
	s.WriteCheckpoint(context.Background()) // unsigned closing checkpoint

	events := mem.snapshot()
	hasCheckpoint := false
	for _, e := range events {
		if e.Type == CheckpointType {
			hasCheckpoint = true
			if sig, _ := e.Details["checkpoint_sig"].(string); sig != "" {
				t.Fatal("expected an UNSIGNED checkpoint, found a signature")
			}
		}
	}
	if !hasCheckpoint {
		t.Fatal("expected a closing checkpoint entry")
	}
	if VerifyChainWithCheckpoints(events).TailAnchored {
		t.Error("an unsigned checkpoint must NOT count as a tail anchor")
	}
}

// The checkpoint signature actually verifies against the signer's public
// key over the head hash bytes (the anchor is real, not cosmetic).
func TestCheckpoint_SignatureVerifiesAgainstHead(t *testing.T) {
	mem := &memorySink{}
	signer, pub := newTestSigner(t)
	s := NewChainedSinkResuming(mem, "", ChainOptions{Signer: signer})
	writeN(t, s, 2)
	s.WriteCheckpoint(context.Background())

	var verified bool
	for _, e := range mem.snapshot() {
		if e.Type != CheckpointType {
			continue
		}
		head, _ := e.Details["checkpoint_head"].(string)
		sigB64, _ := e.Details["checkpoint_sig"].(string)
		sig, derr := base64.StdEncoding.DecodeString(sigB64)
		if derr != nil {
			t.Fatalf("decode sig: %v", derr)
		}
		if !ed25519.Verify(pub, []byte(head), sig) {
			t.Error("checkpoint signature does not verify over the head hash")
		}
		if kid, _ := e.Details["checkpoint_kid"].(string); kid != "test-1" {
			t.Errorf("checkpoint_kid=%q, want test-1", kid)
		}
		verified = true
	}
	if !verified {
		t.Fatal("no signed checkpoint found")
	}
}
