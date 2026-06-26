// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package report

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"

	"github.com/srenix-ai/agentic-sre/internal/snapshot"
	"github.com/srenix-ai/agentic-sre/pkg/ticketing"
)

// TicketingConfig bundles ticket-sink runtime knobs that are not
// expressed via the Sink itself. Constructed at watcher startup from
// Helm values and passed into RouteTickets every cycle.
type TicketingConfig struct {
	// Sink is the ticketing.Sink implementation to invoke. Nil disables
	// the routing entirely (no-op).
	Sink ticketing.Sink

	// Cluster identifies the source cluster in ticket bodies and as
	// part of the Fingerprint. Set to the watcher's cluster name.
	Cluster string

	// Labels are merged into every Ticket.Labels. Typical values:
	// ["srenix", "auto-filed"].
	Labels []string

	// ResolveOnClear toggles auto-closing a ticket when its underlying
	// finding is no longer present in the diagnose cycle. Defaults to ON
	// (the wiring layer sets it true) now that M2 implements it — M1
	// intentionally deferred this. A zero value means "disabled"; the
	// cmd/chart layer is responsible for defaulting it true.
	ResolveOnClear bool

	// CommentInterval is the debounce window for comment-on-recurrence.
	// When a previously-ticketed finding reappears (recurs) — whether the
	// ticket is still open or was already resolved — Srenix adds a comment to
	// the existing ticket instead of opening a new one, but no more often
	// than once per CommentInterval. Zero disables recurrence commenting
	// (the already-ticketed path stays a no-op, matching M1). Typical
	// default: 1h.
	CommentInterval time.Duration

	// MinSeverity is the floor for filing a ticket. A finding is ticketed
	// only when its severity is at least this level — so an issue tracker
	// holds genuine human-action items, not every warning/info observation.
	// Findings below the floor never open a ticket and never re-comment on
	// one. Values: "info" (everything), "warning" (warning+critical),
	// "critical" (critical only — the default the wiring layer sets, since
	// critical is Srenix's human-action / unfixable tier). Empty = "critical".
	MinSeverity string
}

// ticketSeverityRank maps a Srenix severity to a comparable rank. Unknown
// severities sort as warning (the historical default for empty-severity
// diagnostics) so a stray value is never silently treated as critical.
func ticketSeverityRank(s string) int {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "critical":
		return 3
	case "warning":
		return 2
	case "info":
		return 1
	default:
		return 2
	}
}

// meetsTicketThreshold reports whether a finding of the given severity is at
// or above the configured MinSeverity floor. Empty floor defaults to critical.
func (c TicketingConfig) meetsTicketThreshold(severity string) bool {
	floor := c.MinSeverity
	if strings.TrimSpace(floor) == "" {
		floor = "critical"
	}
	return ticketSeverityRank(severity) >= ticketSeverityRank(floor)
}

// RouteTickets is the ticketing analogue of RouteAndPost. It runs AFTER
// DriftReport reconcile so it can read the freshly-created CRs and
// persist the resulting TicketRef back onto status.ticket.
//
// Behaviour:
//   - For each unfixable diagnostic with no existing status.ticket,
//     calls Sink.Upsert and patches the ref onto the DriftReport.
//   - For each unfixable diagnostic that already has a status.ticket
//     (recurrence — still-open or a previously-resolved ticket whose
//     finding reappeared), calls Sink.Comment instead of opening a new
//     ticket, debounced by cfg.CommentInterval so a flapping finding
//     can't spam the tracker. A resolved ticket whose finding recurs is
//     re-flagged active (status.ticket.resolved cleared) so the next
//     clear resolves it again. Disabled when CommentInterval == 0 (M1
//     parity: already-ticketed is then a no-op).
//   - Resolve-on-clear (M2) is NOT handled here — by the time
//     RouteTickets runs, Reconcile has already deleted the DriftReport
//     for a cleared subject, so its ref is gone. The watcher calls
//     RouteResolves BEFORE Reconcile (while the CR + ref still exist) to
//     close cleared findings; see RouteResolves below.
//
// Failures are logged but never aborted: ticketing is a best-effort
// sink, exactly like Slack and Alertmanager.
func RouteTickets(
	ctx context.Context,
	cfg TicketingConfig,
	src snapshot.Source,
	mut snapshot.Mutator,
	postFixSubjects map[string]bool,
	toPost []DeltaDiag,
	runID string,
) {
	if cfg.Sink == nil || mut == nil {
		log.Printf("ticketing: cycle skipped (sink=%v mut=%v)", cfg.Sink != nil, mut != nil)
		return
	}
	log.Printf("ticketing: cycle start (toPost=%d postFix=%d cluster=%q)", len(toPost), len(postFixSubjects), cfg.Cluster)

	bySubject, err := indexDriftReportsBySubject(ctx, src)
	if err != nil {
		log.Printf("ticketing: list driftreports: %v", err)
		return
	}
	log.Printf("ticketing: indexed %d existing driftreports", len(bySubject))

	created := 0
	skippedBelowThreshold := 0
	for _, d := range toPost {
		// Human-action filter: only file tickets for findings at or above the
		// severity floor (default critical). Warnings / info / observations are
		// surfaced via Slack + DriftReports but do NOT clutter the issue tracker.
		if !cfg.meetsTicketThreshold(d.Severity) {
			skippedBelowThreshold++
			continue
		}
		if !postFixSubjects[d.Subject] {
			log.Printf("ticketing: skip %s (fixed this cycle)", d.Subject)
			continue
		}
		cr, ok := bySubject[d.Subject]
		if !ok {
			log.Printf("ticketing: skip %s (no driftreport CR — WriteDriftReports off?)", d.Subject)
			continue
		}
		if existing, ok := readTicketRef(cr); ok {
			// Recurrence: the subject already has a ticket. Comment instead
			// of opening a new one, debounced by CommentInterval, rather
			// than the M1 no-op. A ticket marked resolved that recurs is
			// re-flagged active so the next clear can resolve it again.
			maybeCommentOnRecurrence(ctx, cfg, mut, cr, d, existing, runID)
			continue
		}

		ticket := ticketFromDelta(d, cfg, runID)
		ref, err := cfg.Sink.Upsert(ctx, ticket)
		if err != nil {
			log.Printf("ticketing: upsert %s: %v", d.Subject, err)
			continue
		}
		log.Printf("ticketing: upserted %s -> %s/%s", d.Subject, ref.Provider, ref.Key)
		if err := writeTicketRef(ctx, mut, cr.GetName(), ref, d.Severity); err != nil {
			log.Printf("ticketing: persist ref %s on %s: %v", ref.Key, cr.GetName(), err)
			continue
		}
		created++
	}
	log.Printf("ticketing: cycle end (created=%d, skipped-below-%s=%d)", created, ticketFloorLabel(cfg), skippedBelowThreshold)
}

// ticketFloorLabel returns the effective MinSeverity for log lines.
func ticketFloorLabel(cfg TicketingConfig) string {
	if strings.TrimSpace(cfg.MinSeverity) == "" {
		return "critical"
	}
	return cfg.MinSeverity
}

// ticketFromDelta builds a ticketing.Ticket from a routing DeltaDiag.
func ticketFromDelta(d DeltaDiag, cfg TicketingConfig, runID string) ticketing.Ticket {
	return ticketing.Ticket{
		Fingerprint: ticketing.Fingerprint(d.Subject, cfg.Cluster),
		Title:       truncateAt(d.Subject, 200),
		Body:        buildTicketBody(d, cfg.Cluster, runID),
		Severity:    d.Severity,
		Subject:     d.Subject,
		Cluster:     cfg.Cluster,
		Labels:      append([]string{}, cfg.Labels...),
		OpenedAt:    time.Now().UTC(),
	}
}

func buildTicketBody(d DeltaDiag, cluster, runID string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "**Subject:** `%s`\n\n", d.Subject)
	fmt.Fprintf(&b, "**Severity:** %s\n\n", d.Severity)
	if d.Message != "" {
		fmt.Fprintf(&b, "## Diagnostic\n\n%s\n\n", d.Message)
	}
	// Root-cause-first: the investigator's definitive cause leads, then the
	// remediation steps — same composition order as every Slack adapter.
	if d.Investigation != "" {
		fmt.Fprintf(&b, "## Root cause\n\n%s\n\n", d.Investigation)
	}
	if d.Remediation != "" {
		fmt.Fprintf(&b, "## Recommended action\n\n%s\n\n", d.Remediation)
	}
	fmt.Fprintf(&b, "---\n\nCluster: `%s` · Run: `%s` · Filed by Srenix at %s",
		cluster, runID, time.Now().UTC().Format(time.RFC3339))
	return b.String()
}

func indexDriftReportsBySubject(ctx context.Context, src snapshot.Source) (map[string]*unstructured.Unstructured, error) {
	if src == nil {
		return nil, fmt.Errorf("nil snapshot.Source")
	}
	list, err := src.List(ctx, snapshot.GVRDriftReport, "")
	if err != nil {
		return nil, err
	}
	out := make(map[string]*unstructured.Unstructured, len(list.Items))
	for i := range list.Items {
		cr := &list.Items[i]
		spec, _, _ := unstructured.NestedMap(cr.Object, "spec")
		subj, _ := spec["subject"].(string)
		if subj == "" {
			continue
		}
		out[subj] = cr
	}
	return out, nil
}

func readTicketRef(cr *unstructured.Unstructured) (ticketing.TicketRef, bool) {
	t, found, err := unstructured.NestedMap(cr.Object, "status", "ticket")
	if err != nil || !found {
		return ticketing.TicketRef{}, false
	}
	prov, _ := t["provider"].(string)
	key, _ := t["key"].(string)
	url, _ := t["url"].(string)
	if key == "" {
		return ticketing.TicketRef{}, false
	}
	return ticketing.TicketRef{Provider: prov, Key: key, URL: url}, true
}

func writeTicketRef(ctx context.Context, mut snapshot.Mutator, crName string, ref ticketing.TicketRef, severity string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	patch, err := json.Marshal(map[string]any{
		"status": map[string]any{
			"ticket": map[string]any{
				"provider": ref.Provider,
				"key":      ref.Key,
				"url":      ref.URL,
				"openedAt": now,
				// severity at open-time so the first severity-transition
				// comment (M2) is detectable on a later cycle.
				"severity": severity,
				"resolved": false,
			},
		},
	})
	if err != nil {
		return err
	}
	return mut.PatchStatus(ctx, snapshot.GVRDriftReport, "", crName, types.MergePatchType, patch)
}
