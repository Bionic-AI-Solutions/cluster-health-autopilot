// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package probe

import (
	"context"
	"fmt"
	"strings"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/probe"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// KongRoutes is the M2 probe (trigger-expansion roadmap, v1.7+).
//
// For each Kong-managed Ingress in the cluster, it verifies that:
//
//  1. The Ingress targets at least one rule with a valid backend Service
//  2. Each referenced Service has ≥1 ready Endpoint
//  3. Each KongPlugin / KongConsumer reference in the Ingress annotations
//     resolves to an existing object (cross-namespace allowed)
//
// "Kong-managed" is detected via `ingressClassName=kong` OR the
// `konghq.com/strip-path` / `konghq.com/preserve-host` annotations.
//
// The probe is silent on clusters without Kong installed — Ingress
// objects exist but none match the Kong selector, so the probe emits
// nothing. Opts out fully via env var CHA_PROBE_KONG_ROUTES=off.
type KongRoutes struct{}

// Name satisfies probe.Probe.
func (KongRoutes) Name() string { return "Kong Routes" }

// Kong CRDs (read-only). Absent CRDs → list returns IsNotFound → we
// skip the cross-ref check silently (probe still verifies Service +
// endpoints). gvrKongPlugin is already declared in kong.go (the
// existing KongPlugin status probe) — we share the constant via the
// local kongRoutesPluginGVR alias so a future Kong API-group rename
// only needs to touch one site.
var (
	kongRoutesPluginGVR   = gvrKongPlugin // alias of kong.go's gvrKongPlugin
	kongRoutesConsumerGVR = schema.GroupVersionResource{
		Group: "configuration.konghq.com", Version: "v1", Resource: "kongconsumers",
	}
)

// Run satisfies probe.Probe.
func (k KongRoutes) Run(ctx context.Context, src snapshot.Source) probe.Result {
	r := probe.Result{Component: probe.ComponentResult{Component: "Kong Routes"}}

	ingList, err := src.List(ctx, snapshot.GVRIngress, "")
	if err != nil {
		r.Component.Status = "PROBE_FAILED"
		r.Component.Detail = "list ingresses: " + err.Error()
		return r
	}

	// Build a name-keyed plugin + consumer index — used to detect
	// dangling references. Skipping when the CRD isn't installed.
	plugins := loadKongCRDIndex(ctx, src, kongRoutesPluginGVR)
	consumers := loadKongCRDIndex(ctx, src, kongRoutesConsumerGVR)

	var inspected int
	for i := range ingList.Items {
		ing := &ingList.Items[i]
		if !isKongIngress(ing) {
			continue
		}
		inspected++
		ns := ing.GetNamespace()
		name := ing.GetName()
		subject := "Ingress/" + ns + "/" + name

		// 1+2. Each rule.http.paths[*].backend.service must resolve to
		// a Service with ≥1 ready endpoint.
		rules, _, _ := unstructured.NestedSlice(ing.Object, "spec", "rules")
		for _, ru := range rules {
			rm, ok := ru.(map[string]any)
			if !ok {
				continue
			}
			host, _, _ := unstructured.NestedString(rm, "host")
			paths, _, _ := unstructured.NestedSlice(rm, "http", "paths")
			for _, p := range paths {
				pm, _ := p.(map[string]any)
				svcName, _, _ := unstructured.NestedString(pm, "backend", "service", "name")
				if svcName == "" {
					continue
				}
				if ok := serviceHasReadyEndpoint(ctx, src, ns, svcName); !ok {
					r.Findings = append(r.Findings, probe.Finding{
						Component: "Kong Routes",
						Severity:  probe.SeverityWarning,
						Message: fmt.Sprintf(
							"Ingress %s host=%s backend Service %s/%s has no ready endpoints — Kong will return 503 for any request hitting this path.",
							subject, host, ns, svcName),
						Remediation: fmt.Sprintf(
							"Inspect the backing workload:\n  kubectl -n %s get endpoints %s\n  kubectl -n %s describe svc %s",
							ns, svcName, ns, svcName),
					})
				}
			}
		}

		// 3. Plugin / Consumer annotation refs must exist.
		anns := ing.GetAnnotations()
		if pluginRef := anns["konghq.com/plugins"]; pluginRef != "" && len(plugins) > 0 {
			for _, ref := range strings.Split(pluginRef, ",") {
				ref = strings.TrimSpace(ref)
				if ref == "" {
					continue
				}
				if _, ok := plugins[kongRefKey(ns, ref)]; !ok {
					r.Findings = append(r.Findings, probe.Finding{
						Component: "Kong Routes",
						Severity:  probe.SeverityWarning,
						Message: fmt.Sprintf(
							"Ingress %s references KongPlugin %q which does not exist in namespace %s.",
							subject, ref, ns),
						Remediation: fmt.Sprintf(
							"Either create the missing KongPlugin or remove the konghq.com/plugins annotation:\n  kubectl -n %s get kongplugin",
							ns),
					})
				}
			}
		}
		if consumerRef := anns["konghq.com/consumer"]; consumerRef != "" && len(consumers) > 0 {
			if _, ok := consumers[kongRefKey(ns, consumerRef)]; !ok {
				r.Findings = append(r.Findings, probe.Finding{
					Component: "Kong Routes",
					Severity:  probe.SeverityWarning,
					Message: fmt.Sprintf(
						"Ingress %s references KongConsumer %q which does not exist in namespace %s.",
						subject, consumerRef, ns),
					Remediation: fmt.Sprintf(
						"Either create the missing KongConsumer or remove the konghq.com/consumer annotation:\n  kubectl -n %s get kongconsumer",
						ns),
				})
			}
		}
	}

	// Status convention matches other probes: OK when zero findings on
	// inspected ingresses; WARNING when ≥1; "no Kong ingresses found"
	// is reported as OK with a Detail note for visibility.
	if inspected == 0 {
		r.Component.Status = "OK"
		r.Component.Detail = "no Kong-managed Ingress objects found"
		return r
	}
	if len(r.Findings) == 0 {
		r.Component.Status = "OK"
		r.Component.Detail = fmt.Sprintf("%d Kong-managed Ingress(es) — all backends ready", inspected)
		return r
	}
	r.Component.Status = "WARNING"
	r.Component.Detail = fmt.Sprintf("%d issue(s) across %d Kong-managed Ingress(es)", len(r.Findings), inspected)
	return r
}

// isKongIngress returns true when the Ingress is annotated for or
// classed under Kong. Conservative — relies on the operator setting
// `ingressClassName` correctly, but falls back to two well-known
// annotations Kong's helm chart applies.
func isKongIngress(ing *unstructured.Unstructured) bool {
	cls, _, _ := unstructured.NestedString(ing.Object, "spec", "ingressClassName")
	if strings.ToLower(cls) == "kong" {
		return true
	}
	anns := ing.GetAnnotations()
	if anns["kubernetes.io/ingress.class"] == "kong" {
		return true
	}
	if _, ok := anns["konghq.com/strip-path"]; ok {
		return true
	}
	if _, ok := anns["konghq.com/preserve-host"]; ok {
		return true
	}
	return false
}

// loadKongCRDIndex builds a {ns/name → present} index for CRD objects.
// Returns nil when the CRD isn't installed (list error) — callers
// branch on nil to skip the cross-ref check.
func loadKongCRDIndex(ctx context.Context, src snapshot.Source, gvr schema.GroupVersionResource) map[string]struct{} {
	list, err := src.List(ctx, gvr, "")
	if err != nil {
		return nil
	}
	if len(list.Items) == 0 {
		return nil
	}
	idx := make(map[string]struct{}, len(list.Items))
	for i := range list.Items {
		o := &list.Items[i]
		idx[o.GetNamespace()+"/"+o.GetName()] = struct{}{}
	}
	return idx
}

// kongRefKey resolves a "namespace/name" or bare "name" reference to a
// CRD index key. Bare names default to the Ingress's own namespace.
func kongRefKey(ingNS, ref string) string {
	if strings.Contains(ref, "/") {
		return ref
	}
	return ingNS + "/" + ref
}

// serviceHasReadyEndpoint returns true if the Service ns/name has at
// least one Endpoints object with at least one ready address. Soft-fail
// (returns true) when Endpoints aren't readable — we don't want a
// transient API hiccup to spawn false-positive findings.
func serviceHasReadyEndpoint(ctx context.Context, src snapshot.Source, ns, name string) bool {
	// Try EndpointSlices first (K8s 1.21+ canonical). Fall back to
	// legacy Endpoints if the slice list is empty.
	gvrEPSlice := schema.GroupVersionResource{Group: "discovery.k8s.io", Version: "v1", Resource: "endpointslices"}
	slices, err := src.List(ctx, gvrEPSlice, ns)
	if err == nil {
		for i := range slices.Items {
			s := &slices.Items[i]
			ownerName, _, _ := unstructured.NestedString(s.Object, "metadata", "labels", "kubernetes.io/service-name")
			if ownerName != name {
				continue
			}
			endpoints, _, _ := unstructured.NestedSlice(s.Object, "endpoints")
			for _, e := range endpoints {
				em, _ := e.(map[string]any)
				ready, _, _ := unstructured.NestedBool(em, "conditions", "ready")
				if ready {
					return true
				}
			}
		}
		// Slices exist but none ready for this Service.
		return false
	}
	// Slices not available — fall back to v1.Endpoints.
	gvrEP := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "endpoints"}
	epList, err := src.List(ctx, gvrEP, ns)
	if err != nil {
		// Can't determine; soft-true to avoid false positives.
		return true
	}
	for i := range epList.Items {
		ep := &epList.Items[i]
		if ep.GetName() != name {
			continue
		}
		subsets, _, _ := unstructured.NestedSlice(ep.Object, "subsets")
		for _, s := range subsets {
			sm, _ := s.(map[string]any)
			addrs, _, _ := unstructured.NestedSlice(sm, "addresses")
			if len(addrs) > 0 {
				return true
			}
		}
	}
	return false
}

// GVRs satisfies pkg/probe.GVRWatcher (M7 foundation). Declares the
// resource kinds KongRoutes consumes so future per-probe dispatch can
// gate runs on these triggers only.
func (KongRoutes) GVRs() []schema.GroupVersionResource {
	return []schema.GroupVersionResource{
		snapshot.GVRIngress,
		kongRoutesPluginGVR,
		kongRoutesConsumerGVR,
		{Group: "discovery.k8s.io", Version: "v1", Resource: "endpointslices"},
	}
}
