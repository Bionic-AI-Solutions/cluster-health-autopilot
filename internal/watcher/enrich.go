// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package watcher

import (
	"context"
	"net/url"
	"time"

	"github.com/srenix-ai/agentic-sre/pkg/diagnose"
)

// enrichDiagnostics walks diagnostics and, when the registry has an
// Enricher / FixProposer registered, populates the AI tier fields on
// each Diagnostic. Returns a new slice with the same ordering and the
// AI fields filled in where applicable.
//
// Behavior summary:
//   - No Enricher registered: returns the input slice unchanged (the
//     hot path that OSS users always hit).
//   - Enricher registered: each diagnostic is run through the Enricher
//     in sequence (bounded by enrichmentTimeout). Failures are silent —
//     the deterministic diagnostic still flows; only the enrichment
//     block is omitted.
//   - FixProposer registered: produces ProposedActionID + ApprovalURL
//     fields when the diagnostic matches a whitelisted fixer (implemented
//     below: Propose → Sign → recordApprovalURL).
//
// Enrichment is bounded to enrichmentTimeout total per cycle. Beyond
// that, remaining diagnostics flow without enrichment.
func (w *Watcher) enrichDiagnostics(ctx context.Context, diagnostics []diagnose.Diagnostic) []diagnose.Diagnostic {
	enricher := w.reg.Enricher()
	proposer := w.reg.FixProposer()
	signer := w.reg.Signer()
	approvalBaseURL := w.cfg.ApprovalBaseURL

	if enricher == nil && proposer == nil {
		return diagnostics
	}
	if len(diagnostics) == 0 {
		return diagnostics
	}

	cycleCtx, cancel := context.WithTimeout(ctx, enrichmentTimeout)
	defer cancel()

	out := make([]diagnose.Diagnostic, len(diagnostics))
	copy(out, diagnostics)

	for i := range out {
		if cycleCtx.Err() != nil {
			break
		}

		// T0 enrichment (always when Enricher is registered).
		if enricher != nil {
			if ed, err := enricher.Enrich(cycleCtx, out[i]); err == nil && ed.Enrichment != "" {
				out[i].Enrichment = ed.Enrichment
			}
		}

		// T1 fix proposal (when FixProposer + Signer registered).
		if proposer != nil && signer != nil && approvalBaseURL != "" {
			prop, err := proposer.Propose(cycleCtx, out[i])
			if err != nil || prop == nil {
				continue
			}
			token, err := signer.Sign(*prop)
			if err != nil {
				continue
			}
			out[i].ProposedActionID = prop.ActionID
			// Build approval URL: <baseURL>/approve?token=<jwt>
			u, perr := url.Parse(approvalBaseURL)
			if perr != nil {
				continue
			}
			u.Path = singleSlashJoin(u.Path, "/approve")
			q := u.Query()
			q.Set("token", token)
			u.RawQuery = q.Encode()
			// Store URL on the Diagnostic so the seenEntry → DeltaDiag
			// propagation picks it up. The diagnose.Diagnostic struct
			// doesn't have an ApprovalURL field, so we encode it as a
			// well-known prefix on Enrichment when nothing else is set.
			// Cleaner long-term: add ProposedActionURL to Diagnostic.
			out[i].ProposedActionID = prop.ActionID
			// Use Remediation prefix marker that the reporter can detect
			// to extract a URL. For v1.0.0 we propagate via a side channel
			// stored in the watcher's pendingApprovalURL map (P4 wires this
			// at the buildCurrentState boundary).
			w.recordApprovalURL(prop.ActionID, u.String())
		}
	}

	return out
}

// pendingURLTTL bounds how long a recorded approval URL lingers in the
// pendingURLs map. Entries were previously evicted ONLY on lookup
// (approvalURLFor), so a recorded-but-never-rendered ActionID — e.g. a
// diagnostic that resolved before the next post — persisted for the
// whole process lifetime, leaking memory. 24h is comfortably beyond any
// approval-token TTL (minutes-to-hours), so a legitimately pending URL
// is never swept out from under a still-valid token.
const pendingURLTTL = 24 * time.Hour

// recordApprovalURL stores an approval URL keyed by ActionID. The
// watcher's buildCurrentState reads this to populate seenEntry.approvalURL,
// which in turn populates DeltaDiag.ApprovalURL for rendering.
//
// The cache is bounded two ways: a TTL sweep runs on every insert
// (evicting entries older than pendingURLTTL regardless of whether they
// were ever looked up), and approvalURLFor evicts on access.
func (w *Watcher) recordApprovalURL(actionID, url string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.pendingURLs == nil {
		w.pendingURLs = make(map[string]pendingURL)
	}
	now := w.nowOrDefault()
	// Sweep stale entries first so a write-heavy / read-light workload
	// can't accumulate dead ActionIDs unboundedly.
	for k, e := range w.pendingURLs {
		if now.Sub(e.recordedAt) > pendingURLTTL {
			delete(w.pendingURLs, k)
		}
	}
	w.pendingURLs[actionID] = pendingURL{url: url, recordedAt: now}
}

// approvalURLFor returns the approval URL for actionID, or "" when none
// is registered. Called from buildCurrentState equivalents.
func (w *Watcher) approvalURLFor(actionID string) string {
	w.mu.Lock()
	defer w.mu.Unlock()
	e, ok := w.pendingURLs[actionID]
	if !ok {
		return ""
	}
	if w.nowOrDefault().Sub(e.recordedAt) > pendingURLTTL {
		delete(w.pendingURLs, actionID)
		return ""
	}
	return e.url
}

type pendingURL struct {
	url        string
	recordedAt time.Time
}

func singleSlashJoin(a, b string) string {
	if a == "" {
		return b
	}
	if len(a) > 0 && a[len(a)-1] == '/' {
		a = a[:len(a)-1]
	}
	if len(b) > 0 && b[0] == '/' {
		return a + b
	}
	return a + "/" + b
}

// enrichmentTimeout caps the total wall-clock spent calling the LLM
// during a single watcher cycle. At ~30s for 28 active issues (current
// production scale), this is ~1s per call — well within OpenAI-compatible
// endpoint latency budgets for the small T0 prompt.
const enrichmentTimeout = 30 * time.Second
