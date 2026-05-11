// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package diagnose

import (
	"context"
	"fmt"
	"sort"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// IngressCoverage detects ingress hostnames that have no corresponding endpoint
// probe. Every public ingress host is a user-facing entry point; an uncovered
// host means silent failures (missing Kong route, cert issue, DNS mis-wiring)
// go undetected — exactly the class of failure that took bionicaisolutions.com
// offline without the health monitor noticing.
//
// Explicit non-removal contract: this analyzer ONLY reports hosts that are
// present in cluster ingresses but absent from KnownHosts. It NEVER suggests
// removing a host from KnownHosts — a monitored host that has no ingress rule
// may be an out-of-cluster service or a host intentionally kept in the probe
// list as a canary. Removals require explicit operator action.
type IngressCoverage struct {
	// KnownHosts is the set of hostnames already covered by endpoint probes.
	// Wire from probe.DefaultEndpointHostnames() in the catalog; extend as needed.
	KnownHosts []string
}

// Name returns the analyzer's identifier.
func (IngressCoverage) Name() string { return "IngressCoverage" }

// Run lists all Ingress resources cluster-wide and emits one Diagnostic per
// host that exists in an ingress rule but has no endpoint probe covering it.
func (ic IngressCoverage) Run(ctx context.Context, src snapshot.Source) []Diagnostic {
	ingresses, err := src.List(ctx, snapshot.GVRIngress, "")
	if err != nil || len(ingresses.Items) == 0 {
		return nil
	}

	known := make(map[string]struct{}, len(ic.KnownHosts))
	for _, h := range ic.KnownHosts {
		known[h] = struct{}{}
	}

	type ingressRef struct {
		ns, name, host string
	}
	dedup := make(map[string]struct{})
	var uncovered []ingressRef

	for i := range ingresses.Items {
		ing := ingresses.Items[i]
		ns := ing.GetNamespace()
		name := ing.GetName()

		rules, _, _ := unstructured.NestedSlice(ing.Object, "spec", "rules")
		for _, raw := range rules {
			rm, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			host, _ := rm["host"].(string)
			if host == "" {
				continue // catch-all rule — no specific host to track
			}
			if _, covered := known[host]; covered {
				continue
			}
			if _, seen := dedup[host]; seen {
				continue
			}
			dedup[host] = struct{}{}
			uncovered = append(uncovered, ingressRef{ns: ns, name: name, host: host})
		}
	}

	if len(uncovered) == 0 {
		return nil
	}

	sort.Slice(uncovered, func(i, j int) bool {
		return uncovered[i].host < uncovered[j].host
	})

	out := make([]Diagnostic, 0, len(uncovered))
	for _, u := range uncovered {
		out = append(out, Diagnostic{
			Subject: fmt.Sprintf("ingress-coverage/%s/%s/%s", u.ns, u.name, u.host),
			Message: fmt.Sprintf(
				"Ingress `%s/%s` exposes host *%s* with no endpoint probe — "+
					"TLS faults, missing Kong routes, and DNS failures will go undetected.",
				u.ns, u.name, u.host,
			),
			Remediation: fmt.Sprintf(
				"Add `{URL: \"https://%s\", Name: \"<display name>\"}` to probe.DefaultEndpointTargets() "+
					"in internal/probe/endpoints.go (removal requires explicit operator action — never auto-removed).",
				u.host,
			),
		})
	}
	return out
}
