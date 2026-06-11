// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package report

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/ticketing"
)

// --- Resolve-on-clear (M2) ---

func TestRouteResolvesClosesTicketedCleared(t *testing.T) {
	cr := makeDriftReport("Pod/default/gone", "drift-gone", map[string]any{
		"provider": "openproject",
		"key":      "WP-7",
		"url":      "u",
	})
	src := &tixSource{items: []unstructured.Unstructured{cr}}
	mut := &tixMutator{}
	sink := &recordingSink{}

	RouteResolves(
		context.Background(),
		TicketingConfig{Sink: sink, ResolveOnClear: true},
		src, mut,
		[]string{"Pod/default/gone"},
		time.Now(),
	)

	if len(sink.resolve) != 1 {
		t.Fatalf("resolve calls=%d want 1", len(sink.resolve))
	}
	if sink.resolve[0].Key != "WP-7" {
		t.Errorf("resolved ref key=%q want WP-7", sink.resolve[0].Key)
	}
	// Must stamp resolved=true on status so the next pass is a no-op.
	if len(mut.patches) != 1 || mut.patches[0].sub != "status" {
		t.Fatalf("expected one status patch, got %+v", mut.patches)
	}
	var body map[string]any
	if err := json.Unmarshal(mut.patches[0].body, &body); err != nil {
		t.Fatalf("unmarshal patch: %v", err)
	}
	ticket := body["status"].(map[string]any)["ticket"].(map[string]any)
	if ticket["resolved"] != true {
		t.Errorf("status.ticket.resolved=%v want true", ticket["resolved"])
	}
}

func TestRouteResolvesIdempotentAlreadyResolved(t *testing.T) {
	cr := makeDriftReport("Pod/default/gone", "drift-gone", map[string]any{
		"provider": "openproject",
		"key":      "WP-7",
		"url":      "u",
		"resolved": true,
	})
	src := &tixSource{items: []unstructured.Unstructured{cr}}
	mut := &tixMutator{}
	sink := &recordingSink{}

	RouteResolves(
		context.Background(),
		TicketingConfig{Sink: sink, ResolveOnClear: true},
		src, mut,
		[]string{"Pod/default/gone"},
		time.Now(),
	)

	if len(sink.resolve) != 0 {
		t.Errorf("already-resolved ticket must not be re-resolved, got %d calls", len(sink.resolve))
	}
	if len(mut.patches) != 0 {
		t.Errorf("no patch expected for already-resolved ticket, got %d", len(mut.patches))
	}
}

func TestRouteResolvesSkipsClearedWithoutTicket(t *testing.T) {
	cr := makeDriftReport("Pod/default/gone", "drift-gone", nil) // never ticketed
	src := &tixSource{items: []unstructured.Unstructured{cr}}
	mut := &tixMutator{}
	sink := &recordingSink{}

	RouteResolves(
		context.Background(),
		TicketingConfig{Sink: sink, ResolveOnClear: true},
		src, mut,
		[]string{"Pod/default/gone"},
		time.Now(),
	)
	if len(sink.resolve) != 0 {
		t.Errorf("cleared-but-never-ticketed must not resolve, got %d", len(sink.resolve))
	}
}

func TestRouteResolvesDisabledNoOp(t *testing.T) {
	cr := makeDriftReport("Pod/default/gone", "drift-gone", map[string]any{
		"provider": "openproject", "key": "WP-7", "url": "u",
	})
	src := &tixSource{items: []unstructured.Unstructured{cr}}
	mut := &tixMutator{}
	sink := &recordingSink{}

	RouteResolves(
		context.Background(),
		TicketingConfig{Sink: sink, ResolveOnClear: false}, // disabled
		src, mut,
		[]string{"Pod/default/gone"},
		time.Now(),
	)
	if len(sink.resolve) != 0 {
		t.Errorf("ResolveOnClear=false must be a no-op, got %d resolves", len(sink.resolve))
	}
}

func TestRouteResolvesNilSinkNoOp(t *testing.T) {
	src := &tixSource{}
	mut := &tixMutator{}
	RouteResolves(
		context.Background(),
		TicketingConfig{Sink: nil, ResolveOnClear: true},
		src, mut,
		[]string{"Pod/default/gone"},
		time.Now(),
	)
	if len(mut.patches) != 0 {
		t.Error("nil sink must be a complete no-op")
	}
}

// resolveReason is captured by extending the recording sink for one test.
type reasonSink struct {
	recordingSink
	reasons []string
}

func (r *reasonSink) Resolve(_ context.Context, ref ticketing.TicketRef, reason string) error {
	r.resolve = append(r.resolve, ref)
	r.reasons = append(r.reasons, reason)
	return nil
}

func TestRouteResolvesReasonMentionsCleared(t *testing.T) {
	cr := makeDriftReport("Pod/default/gone", "drift-gone", map[string]any{
		"provider": "openproject", "key": "WP-7", "url": "u",
	})
	src := &tixSource{items: []unstructured.Unstructured{cr}}
	mut := &tixMutator{}
	sink := &reasonSink{}

	RouteResolves(
		context.Background(),
		TicketingConfig{Sink: sink, ResolveOnClear: true},
		src, mut,
		[]string{"Pod/default/gone"},
		time.Now(),
	)
	if len(sink.reasons) != 1 || !strings.Contains(strings.ToLower(sink.reasons[0]), "cleared") {
		t.Errorf("resolve reason=%v want a 'cleared' message", sink.reasons)
	}
}

// --- Comment-on-recurrence (M2) ---

func routeRecurrence(t *testing.T, ticket map[string]any, sev string, interval time.Duration) (*recordingSink, *tixMutator) {
	t.Helper()
	cr := makeDriftReport("Pod/default/flap", "drift-flap", ticket)
	src := &tixSource{items: []unstructured.Unstructured{cr}}
	mut := &tixMutator{}
	sink := &recordingSink{ref: ticketing.TicketRef{Key: "WP-NEW"}}

	RouteTickets(
		context.Background(),
		TicketingConfig{Sink: sink, Cluster: "gpu", CommentInterval: interval},
		src, mut,
		map[string]bool{"Pod/default/flap": true},
		[]DeltaDiag{{Subject: "Pod/default/flap", Severity: sev, Message: "down again"}},
		"run-r",
	)
	return sink, mut
}

func TestRecurrenceCommentsNotNewTicket(t *testing.T) {
	// Existing ticket present, finding still active, CommentInterval set,
	// no prior comment → comment once, never a new Upsert.
	sink, mut := routeRecurrence(t, map[string]any{
		"provider": "openproject", "key": "WP-42", "url": "u",
	}, "critical", time.Hour)

	if len(sink.upserts) != 0 {
		t.Errorf("recurrence must NOT open a new ticket, got %d upserts", len(sink.upserts))
	}
	if len(sink.comment) != 1 {
		t.Fatalf("recurrence comment calls=%d want 1", len(sink.comment))
	}
	if sink.comment[0].ref.Key != "WP-42" {
		t.Errorf("comment ref=%q want WP-42", sink.comment[0].ref.Key)
	}
	// stamp lastCommentedAt for debounce.
	if len(mut.patches) != 1 {
		t.Fatalf("expected one comment-stamp patch, got %d", len(mut.patches))
	}
}

func TestRecurrenceDebouncedWithinInterval(t *testing.T) {
	// lastCommentedAt within the interval → no comment.
	recent := time.Now().UTC().Add(-5 * time.Minute).Format(time.RFC3339)
	sink, mut := routeRecurrence(t, map[string]any{
		"provider": "openproject", "key": "WP-42", "url": "u",
		"lastCommentedAt": recent,
	}, "critical", time.Hour)

	if len(sink.comment) != 0 {
		t.Errorf("comment within debounce window must be suppressed, got %d", len(sink.comment))
	}
	if len(mut.patches) != 0 {
		t.Errorf("no patch expected when debounced, got %d", len(mut.patches))
	}
}

func TestRecurrenceAfterIntervalComments(t *testing.T) {
	// lastCommentedAt older than the interval → comment again (reuse the
	// existing ticket; we do NOT open a new one after the interval).
	old := time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339)
	sink, _ := routeRecurrence(t, map[string]any{
		"provider": "openproject", "key": "WP-42", "url": "u",
		"lastCommentedAt": old,
	}, "critical", time.Hour)

	if len(sink.upserts) != 0 {
		t.Errorf("after-interval recurrence reuses ticket, must not Upsert, got %d", len(sink.upserts))
	}
	if len(sink.comment) != 1 {
		t.Errorf("after-interval recurrence must comment, got %d", len(sink.comment))
	}
}

func TestRecurrenceCommentingDisabledWhenIntervalZero(t *testing.T) {
	// CommentInterval == 0 → M1 parity: already-ticketed is a no-op.
	sink, mut := routeRecurrence(t, map[string]any{
		"provider": "openproject", "key": "WP-42", "url": "u",
	}, "critical", 0)

	if len(sink.comment) != 0 || len(sink.upserts) != 0 {
		t.Errorf("CommentInterval=0 must be a no-op (comments=%d upserts=%d)", len(sink.comment), len(sink.upserts))
	}
	if len(mut.patches) != 0 {
		t.Errorf("no patch expected when commenting disabled, got %d", len(mut.patches))
	}
}

func TestRecurrenceAfterResolveClearsResolvedFlag(t *testing.T) {
	// A resolved ticket whose finding recurs → comment + clear resolved.
	sink, mut := routeRecurrence(t, map[string]any{
		"provider": "openproject", "key": "WP-42", "url": "u",
		"resolved": true,
	}, "critical", time.Hour)

	if len(sink.comment) != 1 {
		t.Fatalf("recurrence-after-resolve must comment, got %d", len(sink.comment))
	}
	if !strings.Contains(strings.ToLower(sink.comment[0].body), "recur") {
		t.Errorf("recurrence comment body should mention recurrence: %q", sink.comment[0].body)
	}
	if len(mut.patches) != 1 {
		t.Fatalf("expected stamp patch, got %d", len(mut.patches))
	}
	var body map[string]any
	if err := json.Unmarshal(mut.patches[0].body, &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	ticket := body["status"].(map[string]any)["ticket"].(map[string]any)
	if ticket["resolved"] != false {
		t.Errorf("recurrence must clear resolved flag, got %v", ticket["resolved"])
	}
}

func TestRecurrenceSeverityTransitionComments(t *testing.T) {
	// Ticket opened at warning; now critical → severity-transition comment.
	sink, _ := routeRecurrence(t, map[string]any{
		"provider": "openproject", "key": "WP-42", "url": "u",
		"severity": "warning",
	}, "critical", time.Hour)

	if len(sink.comment) != 1 {
		t.Fatalf("severity transition must comment, got %d", len(sink.comment))
	}
	b := strings.ToLower(sink.comment[0].body)
	if !strings.Contains(b, "severity") || !strings.Contains(b, "critical") {
		t.Errorf("severity-transition comment should name the new severity: %q", sink.comment[0].body)
	}
}
