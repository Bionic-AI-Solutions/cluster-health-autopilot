// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package report

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/ticketing"
)

// RouteResolves auto-closes tickets whose underlying finding has cleared
// (M2: resolve-on-clear). It MUST run BEFORE DriftReport Reconcile deletes
// the cleared subjects' CRs — that is the only point at which the persisted
// TicketRef on status.ticket is still readable. By the time RouteTickets
// runs (post-Reconcile) the cleared CR, and with it the ref, is gone.
//
// For each cleared subject that has a status.ticket:
//   - already-resolved (status.ticket.resolved == true) → no-op
//     (idempotency: never re-resolve a ticket every cycle), and
//   - otherwise → Sink.Resolve(ref, "CHA: condition cleared as of <ts>")
//     then stamp status.ticket.resolved=true + resolvedAt so a second
//     pass (e.g. a flapping subject re-clearing before the CR is gone)
//     is a no-op.
//
// Disabled (complete no-op) when Sink is nil or cfg.ResolveOnClear is
// false. Failures are logged, never fatal — ticketing is best-effort.
func RouteResolves(
	ctx context.Context,
	cfg TicketingConfig,
	src snapshot.Source,
	mut snapshot.Mutator,
	clearedSubjects []string,
	cycleTime time.Time,
) {
	if cfg.Sink == nil || mut == nil || !cfg.ResolveOnClear {
		return
	}
	if len(clearedSubjects) == 0 {
		return
	}
	bySubject, err := indexDriftReportsBySubject(ctx, src)
	if err != nil {
		log.Printf("ticketing: resolve-on-clear list driftreports: %v", err)
		return
	}
	reason := fmt.Sprintf("CHA: condition cleared as of %s", cycleTime.UTC().Format(time.RFC3339))
	resolved := 0
	for _, subj := range clearedSubjects {
		cr, ok := bySubject[subj]
		if !ok {
			continue // no CR (WriteDriftReports off, or never ticketed)
		}
		ref, ok := readTicketRef(cr)
		if !ok {
			continue // cleared but never ticketed
		}
		if ticketResolved(cr) {
			log.Printf("ticketing: resolve-on-clear skip %s (already resolved %s/%s)", subj, ref.Provider, ref.Key)
			continue
		}
		if err := cfg.Sink.Resolve(ctx, ref, reason); err != nil {
			log.Printf("ticketing: resolve %s (%s/%s): %v", subj, ref.Provider, ref.Key, err)
			continue
		}
		if err := markTicketResolved(ctx, mut, cr.GetName(), cycleTime); err != nil {
			log.Printf("ticketing: persist resolved flag %s on %s: %v", ref.Key, cr.GetName(), err)
			// Resolve already happened; the resolved-state stamp failing
			// only risks a redundant second Resolve, which the sink treats
			// as a no-op. Don't abort.
		}
		log.Printf("ticketing: resolved %s -> %s/%s (condition cleared)", subj, ref.Provider, ref.Key)
		resolved++
	}
	log.Printf("ticketing: resolve-on-clear (cleared=%d resolved=%d)", len(clearedSubjects), resolved)
}

// maybeCommentOnRecurrence handles a still-present finding that already has
// a ticket. It posts a debounced comment to the existing ticket rather than
// opening a new one.
//
// Cases:
//   - CommentInterval == 0 → disabled (M1 parity: no-op).
//   - ticket previously resolved (finding recurred after a clear) → comment
//     a "reopened/recurred" note and clear the resolved flag so the next
//     clear resolves it again. After-interval recurrence policy: we do NOT
//     open a new ticket; we reuse the existing one and comment. The
//     operator's existing investigation history stays in one place; a fresh
//     ticket per flap fragments context. (Documented in the M2 design doc.)
//   - severity changed since last observed on the ticket → comment the
//     transition.
//   - otherwise (stable, still-open) → comment only when the debounce
//     window has elapsed since lastCommentedAt, so a long-running finding
//     gets a periodic "still active" note without spamming.
//
// All comment paths are gated by the CommentInterval debounce keyed on
// status.ticket.lastCommentedAt — at most one comment per window.
func maybeCommentOnRecurrence(
	ctx context.Context,
	cfg TicketingConfig,
	mut snapshot.Mutator,
	cr *unstructured.Unstructured,
	d DeltaDiag,
	ref ticketing.TicketRef,
	runID string,
) {
	if cfg.CommentInterval <= 0 {
		log.Printf("ticketing: skip %s (already ticketed: %s/%s; commenting disabled)", d.Subject, ref.Provider, ref.Key)
		return
	}

	wasResolved := ticketResolved(cr)
	prevSeverity := ticketSeverity(cr)
	sevChanged := prevSeverity != "" && prevSeverity != d.Severity

	now := time.Now().UTC()
	last := ticketLastCommentedAt(cr)
	debounced := !last.IsZero() && now.Sub(last) < cfg.CommentInterval

	// A recurrence-after-clear or a severity transition is materially
	// new information; still honor the debounce window so a fast flap
	// can't spam, but these are the reasons we comment at all on an
	// otherwise-stable finding.
	if debounced {
		log.Printf("ticketing: skip comment %s (%s/%s) — debounced (last %s ago < %s)",
			d.Subject, ref.Provider, ref.Key, now.Sub(last).Truncate(time.Second), cfg.CommentInterval)
		return
	}

	body := buildRecurrenceComment(d, cfg.Cluster, runID, wasResolved, sevChanged, prevSeverity)
	if err := cfg.Sink.Comment(ctx, ref, body); err != nil {
		log.Printf("ticketing: comment %s (%s/%s): %v", d.Subject, ref.Provider, ref.Key, err)
		return
	}
	log.Printf("ticketing: commented %s -> %s/%s (resolved=%v sevChanged=%v)", d.Subject, ref.Provider, ref.Key, wasResolved, sevChanged)

	// Persist the comment timestamp (debounce), the current severity (so
	// the next severity transition is detectable), and clear the resolved
	// flag if this was a recurrence-after-clear so a future clear resolves
	// the ticket again.
	if err := stampTicketComment(ctx, mut, cr.GetName(), now, d.Severity); err != nil {
		log.Printf("ticketing: persist comment stamp %s on %s: %v", ref.Key, cr.GetName(), err)
	}
}

// buildRecurrenceComment renders the markdown body for a recurrence /
// severity-transition / still-active comment.
func buildRecurrenceComment(d DeltaDiag, cluster, runID string, wasResolved, sevChanged bool, prevSeverity string) string {
	now := time.Now().UTC().Format(time.RFC3339)
	switch {
	case wasResolved:
		return fmt.Sprintf(
			"**CHA: condition recurred.** This finding previously cleared and the ticket was resolved; it has reappeared and is active again.\n\n"+
				"**Subject:** `%s`\n**Severity:** %s\n\n%s\n\n---\nCluster: `%s` · Run: `%s` · %s",
			d.Subject, d.Severity, d.Message, cluster, runID, now)
	case sevChanged:
		return fmt.Sprintf(
			"**CHA: severity changed** %s → %s.\n\n**Subject:** `%s`\n\n%s\n\n---\nCluster: `%s` · Run: `%s` · %s",
			prevSeverity, d.Severity, d.Subject, d.Message, cluster, runID, now)
	default:
		return fmt.Sprintf(
			"**CHA: condition still active.**\n\n**Subject:** `%s`\n**Severity:** %s\n\n%s\n\n---\nCluster: `%s` · Run: `%s` · %s",
			d.Subject, d.Severity, d.Message, cluster, runID, now)
	}
}

// ticketResolved reports whether status.ticket.resolved is set true.
func ticketResolved(cr *unstructured.Unstructured) bool {
	t, found, err := unstructured.NestedMap(cr.Object, "status", "ticket")
	if err != nil || !found {
		return false
	}
	b, _ := t["resolved"].(bool)
	return b
}

// ticketSeverity returns the last severity stamped on status.ticket
// (empty if none recorded yet).
func ticketSeverity(cr *unstructured.Unstructured) string {
	t, found, err := unstructured.NestedMap(cr.Object, "status", "ticket")
	if err != nil || !found {
		return ""
	}
	s, _ := t["severity"].(string)
	return s
}

// ticketLastCommentedAt returns the parsed status.ticket.lastCommentedAt,
// or the zero time when absent/unparseable.
func ticketLastCommentedAt(cr *unstructured.Unstructured) time.Time {
	t, found, err := unstructured.NestedMap(cr.Object, "status", "ticket")
	if err != nil || !found {
		return time.Time{}
	}
	s, _ := t["lastCommentedAt"].(string)
	if s == "" {
		return time.Time{}
	}
	ts, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return ts
}

// markTicketResolved stamps status.ticket.resolved=true + resolvedAt.
func markTicketResolved(ctx context.Context, mut snapshot.Mutator, crName string, at time.Time) error {
	patch, err := json.Marshal(map[string]any{
		"status": map[string]any{
			"ticket": map[string]any{
				"resolved":   true,
				"resolvedAt": at.UTC().Format(time.RFC3339),
			},
		},
	})
	if err != nil {
		return err
	}
	return mut.PatchStatus(ctx, snapshot.GVRDriftReport, "", crName, types.MergePatchType, patch)
}

// stampTicketComment records the comment timestamp + current severity and
// clears the resolved flag (recurrence reactivates the ticket).
func stampTicketComment(ctx context.Context, mut snapshot.Mutator, crName string, at time.Time, severity string) error {
	patch, err := json.Marshal(map[string]any{
		"status": map[string]any{
			"ticket": map[string]any{
				"lastCommentedAt": at.UTC().Format(time.RFC3339),
				"severity":        severity,
				"resolved":        false,
			},
		},
	})
	if err != nil {
		return err
	}
	return mut.PatchStatus(ctx, snapshot.GVRDriftReport, "", crName, types.MergePatchType, patch)
}
