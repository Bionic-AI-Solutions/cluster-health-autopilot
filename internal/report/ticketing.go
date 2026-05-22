// Copyright 2026 Cluster Health Autopilot contributors
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

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/ticketing"
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
	// ["cha", "auto-filed"].
	Labels []string
}

// RouteTickets is the ticketing analogue of RouteAndPost. It runs AFTER
// DriftReport reconcile so it can read the freshly-created CRs and
// persist the resulting TicketRef back onto status.ticket.
//
// Behaviour:
//   - For each unfixable diagnostic with no existing status.ticket,
//     calls Sink.Upsert and patches the ref onto the DriftReport.
//   - For each unfixable diagnostic that already has a status.ticket,
//     this is a no-op for M1 (comment-on-recurrence lands in M2).
//   - Resolve-on-clear is intentionally NOT implemented in M1 — by the
//     time RouteTickets runs, the DriftReport for a cleared subject has
//     already been deleted by Reconcile, so the ref is gone. M2 will
//     thread the ref through differently.
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
		return
	}

	bySubject, err := indexDriftReportsBySubject(ctx, src)
	if err != nil {
		log.Printf("ticketing: list driftreports: %v", err)
		return
	}

	for _, d := range toPost {
		if !postFixSubjects[d.Subject] {
			continue // fixed this cycle — nothing to ticket
		}
		cr, ok := bySubject[d.Subject]
		if !ok {
			// DriftReport CRD disabled or CR not yet visible. CHA still
			// runs Reconcile before us; this only happens when WriteDriftReports
			// is false. In that mode, ticketing is also a no-op.
			continue
		}
		if existing, ok := readTicketRef(cr); ok {
			// M2 will compare severity/message and post a comment; M1 skips.
			_ = existing
			continue
		}

		ticket := ticketFromDelta(d, cfg, runID)
		ref, err := cfg.Sink.Upsert(ctx, ticket)
		if err != nil {
			log.Printf("ticketing: upsert %s: %v", d.Subject, err)
			continue
		}
		if err := writeTicketRef(ctx, mut, cr.GetName(), ref); err != nil {
			log.Printf("ticketing: persist ref %s on %s: %v", ref.Key, cr.GetName(), err)
		}
	}
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
	if d.Remediation != "" {
		fmt.Fprintf(&b, "## Remediation\n\n%s\n\n", d.Remediation)
	}
	if d.Investigation != "" {
		fmt.Fprintf(&b, "## Investigation\n\n%s\n\n", d.Investigation)
	}
	fmt.Fprintf(&b, "---\n\nCluster: `%s` · Run: `%s` · Filed by CHA at %s",
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

func writeTicketRef(ctx context.Context, mut snapshot.Mutator, crName string, ref ticketing.TicketRef) error {
	now := time.Now().UTC().Format(time.RFC3339)
	patch, err := json.Marshal(map[string]any{
		"status": map[string]any{
			"ticket": map[string]any{
				"provider": ref.Provider,
				"key":      ref.Key,
				"url":      ref.URL,
				"openedAt": now,
			},
		},
	})
	if err != nil {
		return err
	}
	return mut.Patch(ctx, snapshot.GVRDriftReport, "", crName, types.MergePatchType, patch)
}
