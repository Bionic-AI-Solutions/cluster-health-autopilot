// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

// Package ticketing defines the Sink interface Srenix uses to file
// human-intervention tickets for diagnostics that auto-remediation cannot
// resolve.
//
// A Sink is invoked from internal/report/routing.go on every watcher cycle
// for each unfixable diagnostic. Implementations are expected to be
// idempotent — callers pass a deterministic Fingerprint and re-call Upsert
// on every cycle the diagnostic is still active. The sink decides whether
// that means "create a new ticket", "add a comment to the existing one", or
// "do nothing".
//
// The interface is provider-agnostic. OSS ships an OpenProject
// implementation in pkg/ticketing/openproject; Jira and ServiceNow live in
// Srenix Enterprise.
package ticketing

import (
	"context"
	"time"
)

// Severity mirrors the lowercase severity strings used by
// pkg/diagnose and internal/report (info | warning | critical).
// Defined as constants here so sinks and adapters share a vocabulary
// without importing diagnose.
const (
	SeverityInfo     = "info"
	SeverityWarning  = "warning"
	SeverityCritical = "critical"
)

// Ticket is the provider-agnostic payload a Sink writes to its backing
// issue tracker. Constructed by the routing adapter from a DeltaDiag plus
// runtime context (cluster name, runID).
type Ticket struct {
	// Fingerprint is the dedup key. Same value across cycles for the same
	// underlying diagnostic; implementations MUST treat repeated Upserts
	// with the same Fingerprint as idempotent.
	Fingerprint string

	// Title is a one-line summary (≤200 chars). Typically the diagnostic
	// Subject or "<Source>: <Subject>".
	Title string

	// Body is the long-form markdown description: message, remediation,
	// investigation, cluster, runID, link back to DriftReport.
	Body string

	// Severity is one of SeverityInfo / SeverityWarning / SeverityCritical.
	// Sinks map this to provider-native priority via Helm-provided overrides.
	Severity string

	// Subject is the original diagnose.Diagnostic.Subject — useful for
	// labels and provider-side filtering.
	Subject string

	// Source names the analyzer / probe / fixer that produced the
	// underlying diagnostic.
	Source string

	// Cluster identifies the source cluster. Required when one tracker is
	// shared across multiple Srenix installations.
	Cluster string

	// Labels are provider-native labels/tags. Helm-configured defaults
	// (e.g. ["srenix", "auto-filed"]) get merged in here by the adapter.
	Labels []string

	// OpenedAt is the time this Ticket was assembled. Sinks may use it for
	// the initial creation timestamp; provider-side timestamps win on
	// subsequent updates.
	OpenedAt time.Time
}

// TicketRef identifies an existing ticket in a provider so Srenix can comment
// on or resolve it on future cycles. Stored on
// DriftReport.status.ticket so it persists across Srenix process restarts.
type TicketRef struct {
	Provider string // "openproject" | "jira" | "servicenow"
	Key      string // WP-1287 / OPS-42 / INC0012345
	URL      string // canonical URL the operator clicks through to
}

// Sink is the contract every ticketing implementation satisfies. All
// methods take a context for cancellation and must be safe to call
// concurrently — the routing layer fans out one goroutine per diagnostic.
//
// Implementations MUST be non-fatal on failure: callers log and continue.
// A failed Upsert/Resolve must never abort the surrounding watcher cycle.
type Sink interface {
	// Upsert creates a ticket for t, or returns the ref of an existing
	// one matching t.Fingerprint. Idempotent. The returned TicketRef is
	// stable across calls for the same Fingerprint.
	Upsert(ctx context.Context, t Ticket) (TicketRef, error)

	// Resolve closes/transitions a ticket to a terminal state. reason
	// is appended as a final comment (e.g. "drift cleared by Srenix").
	// Resolving a ticket that is already closed must be a no-op.
	Resolve(ctx context.Context, ref TicketRef, reason string) error

	// Comment appends a comment to an existing ticket. Used for
	// recurrence after a debounce window, severity transitions, and
	// fixer-failure notes.
	Comment(ctx context.Context, ref TicketRef, body string) error

	// Provider returns the string this sink uses for TicketRef.Provider
	// (e.g. "openproject"). Lets the routing layer label DriftReport
	// status correctly.
	Provider() string
}
