// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package probe

import (
	"context"
	"strings"

	"github.com/srenix-ai/agentic-sre/pkg/rag"
)

// learntApexTargets queries the rag.Reader for KindApexDomain entries above
// the importance floor and returns them as EndpointTargets, de-duplicated
// against alreadyKnown (the hostnames the static + auto-discovered sources
// will produce). OSS callers pass NoopReader (or nil); the result is then
// empty and the probe behaves identically to pre-2d.
//
// Failure mode: any reader error → return nil and continue. A Qdrant outage
// must not break probes. See pkg/rag.Reader interface contract.
func learntApexTargets(ctx context.Context, r rag.Reader, importanceMin float64, alreadyKnown []string) []EndpointTarget {
	if r == nil {
		return nil
	}
	entries, err := r.List(ctx, rag.Query{
		Kind:          rag.KindApexDomain,
		ImportanceMin: importanceMin,
	})
	if err != nil || len(entries) == 0 {
		return nil
	}

	known := make(map[string]struct{}, len(alreadyKnown))
	for _, h := range alreadyKnown {
		known[strings.ToLower(strings.TrimSpace(h))] = struct{}{}
	}

	out := make([]EndpointTarget, 0, len(entries))
	for _, e := range entries {
		host := strings.ToLower(strings.TrimSpace(e.Key))
		if host == "" {
			continue
		}
		if _, dup := known[host]; dup {
			continue // skip — already covered by static or discovery
		}
		known[host] = struct{}{}

		name, _ := e.Features["display_name"].(string)
		if name == "" {
			name = host + " (learnt)"
		}
		out = append(out, EndpointTarget{
			URL:  "https://" + host,
			Name: name,
		})
	}
	return out
}

// hostOf extracts the lowercased hostname from a URL string. Returns the
// original input on parse failure — the consumer is a de-dup map, so a
// non-host string that round-trips identically is safe.
func hostOf(rawURL string) string {
	h := rawURL
	if i := strings.Index(h, "://"); i >= 0 {
		h = h[i+3:]
	}
	if i := strings.IndexAny(h, "/?"); i >= 0 {
		h = h[:i]
	}
	return strings.ToLower(h)
}
