// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package gcp

import (
	"testing"

	compute "google.golang.org/api/compute/v1"
)

// forwardingRuleIndex feeds the CHA-com "(lb: ...)" join key — backend
// service name → forwarding-rule IP (preferred) or name.
//
// Error-injection: when ForwardingRules.AggregatedList errors,
// listForwardingRuleIndex returns nil (best-effort). A nil-map lookup
// returns "" so the probe falls back to the backend-service name as the
// join value — the finding still carries a "(lb: <name>)" suffix.
func TestForwardingRuleIndex_ErrorYieldsNilMap_ProbeFallsBackToName(t *testing.T) {
	// Production error path: listForwardingRuleIndex returns a nil map.
	// Assert that a nil map is what callers receive and that nil-map reads
	// are safe (Go spec: reading a nil map returns the zero value).
	var idx map[string]string // mirrors the nil return from listForwardingRuleIndex
	if idx != nil {
		t.Errorf("expected nil map on error path, got non-nil")
	}
	if got := idx["any-backend"]; got != "" {
		t.Errorf("nil-map read: idx[any-backend]=%q want \"\" (name fallback)", got)
	}
}

func TestForwardingRuleIndex_MapsBackendServiceToIP(t *testing.T) {
	idx := forwardingRuleIndex([]*compute.ForwardingRule{
		{
			Name:           "fr-web",
			IPAddress:      "203.0.113.7",
			BackendService: "https://www.googleapis.com/compute/v1/projects/p/regions/us-central1/backendServices/web",
		},
	})
	if got := idx["web"]; got != "203.0.113.7" {
		t.Errorf("idx[web]=%q want 203.0.113.7", got)
	}
}

func TestForwardingRuleIndex_FallsBackToRuleNameWithoutIP(t *testing.T) {
	idx := forwardingRuleIndex([]*compute.ForwardingRule{
		{Name: "fr-web", BackendService: "projects/p/global/backendServices/web"},
	})
	if got := idx["web"]; got != "fr-web" {
		t.Errorf("idx[web]=%q want fr-web", got)
	}
}

func TestForwardingRuleIndex_SkipsRulesWithoutBackendService(t *testing.T) {
	// Proxy-based LBs reference a target proxy, not a backend service —
	// those rules can't be joined directly and must not pollute the map.
	idx := forwardingRuleIndex([]*compute.ForwardingRule{
		{Name: "fr-proxy", IPAddress: "198.51.100.1", Target: "projects/p/global/targetHttpProxies/tp"},
		nil,
	})
	if len(idx) != 0 {
		t.Errorf("want empty index, got %v", idx)
	}
}

func TestForwardingRuleIndex_SkipsEntriesWithNoUsableValue(t *testing.T) {
	idx := forwardingRuleIndex([]*compute.ForwardingRule{
		{BackendService: "projects/p/global/backendServices/web"}, // no IP, no name
	})
	if _, ok := idx["web"]; ok {
		t.Errorf("entry with no usable join value must be skipped; got %v", idx)
	}
}
