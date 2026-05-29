// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package probe

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// DiscoveryOptions controls auto-discovery of ingress hostnames as endpoint
// probe targets. Default behavior probes every public ingress host that lives
// outside a protected namespace, with TCP+TLS reachability as the success
// criterion. Operators opt-out per-Ingress with an annotation.
type DiscoveryOptions struct {
	// Enabled toggles auto-discovery on. When false, only static Targets are probed.
	Enabled bool

	// SkipNamespaces is the no-discover list. Hosts in these namespaces are
	// never auto-probed. Defaults to the same protected-namespace set used by
	// fixers (kube-system, vault, external-secrets, etc.).
	SkipNamespaces map[string]struct{}

	// OptOutAnnotation is the Ingress annotation key that, when set to "true",
	// excludes that Ingress's hosts from auto-discovery. Defaults to
	// "cha.bionicaisolutions.com/probe-disable".
	OptOutAnnotation string

	// Scheme is the URL scheme used to construct probe URLs. Defaults to "https".
	Scheme string
}

// DefaultDiscoveryOptions returns a DiscoveryOptions configured to auto-probe
// every public ingress host outside of protected namespaces.
func DefaultDiscoveryOptions() DiscoveryOptions {
	return DiscoveryOptions{
		Enabled:          true,
		SkipNamespaces:   defaultSkipNamespaces(),
		OptOutAnnotation: "cha.bionicaisolutions.com/probe-disable",
		Scheme:           "https",
	}
}

// defaultSkipNamespaces mirrors the fixer protected-namespace list — hosts
// exposed by platform components in these namespaces are out of scope for
// CHA probing by design.
func defaultSkipNamespaces() map[string]struct{} {
	return map[string]struct{}{
		"kube-system":      {},
		"kube-public":      {},
		"kube-node-lease":  {},
		"rook-ceph":        {},
		"vault":            {},
		"external-secrets": {},
		"cnpg-system":      {},
	}
}

// DiscoverIngressTargets enumerates Ingresses in the cluster (or snapshot) and
// returns auto-generated EndpointTarget entries for every host that:
//
//   - lives outside opts.SkipNamespaces,
//   - is not opted out by opts.OptOutAnnotation,
//   - is not already covered by an explicit target in `existing`.
//
// Discovered targets carry no ExpectStatus — they succeed on any HTTP response
// (TCP connect + TLS validation pass). Explicit static targets with strict
// status expectations are layered on top by the caller and take precedence.
func DiscoverIngressTargets(
	ctx context.Context,
	src snapshot.Source,
	opts DiscoveryOptions,
	existing []string,
) []EndpointTarget {
	if !opts.Enabled {
		return nil
	}
	scheme := opts.Scheme
	if scheme == "" {
		scheme = "https"
	}
	annotation := opts.OptOutAnnotation
	if annotation == "" {
		annotation = "cha.bionicaisolutions.com/probe-disable"
	}

	ingresses, err := src.List(ctx, snapshot.GVRIngress, "")
	if err != nil || ingresses == nil || len(ingresses.Items) == 0 {
		return nil
	}

	existingSet := make(map[string]struct{}, len(existing))
	for _, h := range existing {
		existingSet[strings.ToLower(h)] = struct{}{}
	}

	seen := make(map[string]struct{})
	var out []EndpointTarget

	for i := range ingresses.Items {
		ing := ingresses.Items[i]
		ns := ing.GetNamespace()
		if _, skip := opts.SkipNamespaces[ns]; skip {
			continue
		}
		if isEphemeralIngress(ing.GetName()) {
			// cert-manager spawns short-lived cm-acme-http-solver-*
			// Ingresses during an HTTP-01 challenge and deletes them on
			// completion. Probing them produces churning false-criticals
			// (and ticket spam) for hosts that aren't real services.
			continue
		}
		if v, ok := ing.GetAnnotations()[annotation]; ok && strings.EqualFold(v, "true") {
			continue
		}

		rules, _, _ := unstructured.NestedSlice(ing.Object, "spec", "rules")
		for _, raw := range rules {
			rm, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			host, _ := rm["host"].(string)
			host = strings.TrimSpace(strings.ToLower(host))
			if host == "" {
				continue // catch-all rules have no specific host
			}
			if _, hasExplicit := existingSet[host]; hasExplicit {
				continue // explicit static target overrides; don't double-probe
			}
			if _, dup := seen[host]; dup {
				continue
			}
			seen[host] = struct{}{}
			out = append(out, EndpointTarget{
				URL:  fmt.Sprintf("%s://%s", scheme, host),
				Name: fmt.Sprintf("%s/%s → %s", ns, ing.GetName(), host),
			})
		}
	}

	// Stable order — same snapshot must produce the same target list.
	sort.Slice(out, func(i, j int) bool { return out[i].URL < out[j].URL })
	return out
}

// isEphemeralIngress reports whether an Ingress is a transient artifact
// that should never be probed as a real endpoint. Currently matches
// cert-manager's HTTP-01 challenge solvers (cm-acme-http-solver-*), which
// exist only for the duration of an ACME challenge.
func isEphemeralIngress(name string) bool {
	return strings.HasPrefix(name, "cm-acme-http-solver-")
}

// hostnamesOf returns the hostname component of each target's URL.
// Used internally to seed the "already covered" set when discovery runs
// alongside an explicit static target list.
func hostnamesOf(targets []EndpointTarget) []string {
	out := make([]string, 0, len(targets))
	for _, t := range targets {
		// Trim scheme and any path — we only care about the hostname.
		h := t.URL
		if i := strings.Index(h, "://"); i >= 0 {
			h = h[i+3:]
		}
		if i := strings.IndexAny(h, "/?"); i >= 0 {
			h = h[:i]
		}
		out = append(out, strings.ToLower(h))
	}
	return out
}
