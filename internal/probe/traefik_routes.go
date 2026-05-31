// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package probe

import (
	"context"
	"fmt"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// TraefikRoutes inspects Traefik IngressRoute CRDs for configuration drift:
// missing backend Services, unresolved Middleware references, and TLS
// configurations with no cert provisioner.
//
// Auto-skip when traefik.io CRDs are absent — the probe lists
// ingressroutes; a list failure results in SKIPPED without consuming
// any operator noise budget. Safe to register default-on on any cluster.
type TraefikRoutes struct{}

const traefikRoutesName = "Traefik Routes"

// Traefik v3 (k3s 1.30+) uses traefik.io; v2 uses traefik.containo.us.
// We try v3 first; if the list returns an error we fall back to v2.
var (
	gvrTraefikIngressRoute = schema.GroupVersionResource{
		Group: "traefik.io", Version: "v1alpha1", Resource: "ingressroutes",
	}
	gvrTraefikIngressRouteFallback = schema.GroupVersionResource{
		Group: "traefik.containo.us", Version: "v1alpha1", Resource: "ingressroutes",
	}
	gvrTraefikIngressRouteTCP = schema.GroupVersionResource{
		Group: "traefik.io", Version: "v1alpha1", Resource: "ingressroutetcps",
	}
	gvrTraefikIngressRouteTCPFallback = schema.GroupVersionResource{
		Group: "traefik.containo.us", Version: "v1alpha1", Resource: "ingressroutetcps",
	}
	gvrTraefikMiddleware = schema.GroupVersionResource{
		Group: "traefik.io", Version: "v1alpha1", Resource: "middlewares",
	}
	gvrTraefikMiddlewareFallback = schema.GroupVersionResource{
		Group: "traefik.containo.us", Version: "v1alpha1", Resource: "middlewares",
	}
	gvrTraefikTLSStore = schema.GroupVersionResource{
		Group: "traefik.io", Version: "v1alpha1", Resource: "tlsstores",
	}
	gvrTraefikTLSStoreFallback = schema.GroupVersionResource{
		Group: "traefik.containo.us", Version: "v1alpha1", Resource: "tlsstores",
	}

	gvrService = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"}
)

// Name satisfies probe.Probe.
func (TraefikRoutes) Name() string { return traefikRoutesName }

// Run satisfies probe.Probe.
func (TraefikRoutes) Run(ctx context.Context, src snapshot.Source) Result {
	r := Result{Component: ComponentResult{Component: traefikRoutesName}}

	// Try traefik.io (v3) first; fall back to traefik.containo.us (v2).
	irList, tcpList, mwList, tlsList, gvrUsed, ok := listTraefikCRDs(ctx, src)
	if !ok {
		r.Component.Status = "SKIPPED"
		r.Component.Detail = "Traefik CRDs not installed"
		return r
	}
	_ = gvrUsed

	// Build service lookup: "ns/name" → true
	svcSet, err := buildServiceSet(ctx, src)
	if err != nil {
		r.Component.Status = "PROBE_FAILED"
		r.Component.Detail = "list services: " + err.Error()
		return r
	}

	// Build middleware lookup: "ns/name" → true
	mwSet := buildNameSet(mwList)

	// Build TLSStore lookup: "ns/name" → true
	tlsSet := buildNameSet(tlsList)

	var findings []Finding

	// Inspect IngressRoutes.
	for i := range irList.Items {
		ir := &irList.Items[i]
		findings = append(findings, checkIngressRoute(ir, svcSet, mwSet, tlsSet)...)
	}

	// Inspect IngressRouteTCPs (service existence only; no middlewares in v1alpha1 TCP schema).
	for i := range tcpList.Items {
		tcp := &tcpList.Items[i]
		findings = append(findings, checkIngressRouteTCP(tcp, svcSet)...)
	}

	r.Component.Status = rollupComponentStatus(findings)
	if len(findings) == 0 {
		r.Component.Detail = fmt.Sprintf("%d IngressRoute(s) and %d IngressRouteTCP(s) inspected, all healthy",
			len(irList.Items), len(tcpList.Items))
	} else {
		r.Component.Detail = fmt.Sprintf("%d IngressRoute(s) and %d IngressRouteTCP(s) inspected",
			len(irList.Items), len(tcpList.Items))
	}
	r.Findings = findings
	return r
}

// listTraefikCRDs tries traefik.io first, then traefik.containo.us.
// Returns the four lists plus which GVR group was used, and ok=false
// if neither group is installed.
func listTraefikCRDs(ctx context.Context, src snapshot.Source) (
	irList, tcpList, mwList, tlsList *unstructured.UnstructuredList,
	group string, ok bool,
) {
	// Try v3 (traefik.io).
	list, err := src.List(ctx, gvrTraefikIngressRoute, "")
	if err == nil {
		tcps, _ := src.List(ctx, gvrTraefikIngressRouteTCP, "")
		if tcps == nil {
			tcps = &unstructured.UnstructuredList{}
		}
		mws, _ := src.List(ctx, gvrTraefikMiddleware, "")
		if mws == nil {
			mws = &unstructured.UnstructuredList{}
		}
		tls, _ := src.List(ctx, gvrTraefikTLSStore, "")
		if tls == nil {
			tls = &unstructured.UnstructuredList{}
		}
		return list, tcps, mws, tls, "traefik.io", true
	}

	// Fall back to v2 (traefik.containo.us).
	list, err = src.List(ctx, gvrTraefikIngressRouteFallback, "")
	if err != nil {
		return nil, nil, nil, nil, "", false
	}
	tcps, _ := src.List(ctx, gvrTraefikIngressRouteTCPFallback, "")
	if tcps == nil {
		tcps = &unstructured.UnstructuredList{}
	}
	mws, _ := src.List(ctx, gvrTraefikMiddlewareFallback, "")
	if mws == nil {
		mws = &unstructured.UnstructuredList{}
	}
	tls, _ := src.List(ctx, gvrTraefikTLSStoreFallback, "")
	if tls == nil {
		tls = &unstructured.UnstructuredList{}
	}
	return list, tcps, mws, tls, "traefik.containo.us", true
}

func buildServiceSet(ctx context.Context, src snapshot.Source) (map[string]bool, error) {
	list, err := src.List(ctx, gvrService, "")
	if err != nil {
		return nil, err
	}
	set := make(map[string]bool, len(list.Items))
	for _, svc := range list.Items {
		set[svc.GetNamespace()+"/"+svc.GetName()] = true
	}
	return set, nil
}

func buildNameSet(list *unstructured.UnstructuredList) map[string]bool {
	if list == nil {
		return map[string]bool{}
	}
	set := make(map[string]bool, len(list.Items))
	for _, item := range list.Items {
		set[item.GetNamespace()+"/"+item.GetName()] = true
	}
	return set
}

func checkIngressRoute(
	ir *unstructured.Unstructured,
	svcSet, mwSet, tlsSet map[string]bool,
) []Finding {
	irNs := ir.GetNamespace()
	irName := ir.GetName()

	routes, _, _ := unstructured.NestedSlice(ir.Object, "spec", "routes")

	var findings []Finding
	for i, rawRoute := range routes {
		route, ok := rawRoute.(map[string]interface{})
		if !ok {
			continue
		}

		// F1: missing backend Service
		rawSvcs, _, _ := unstructured.NestedSlice(route, "services")
		for _, rawSvc := range rawSvcs {
			svc, ok := rawSvc.(map[string]interface{})
			if !ok {
				continue
			}
			svcName, _ := svc["name"].(string)
			svcNs, _ := svc["namespace"].(string)
			if svcNs == "" {
				svcNs = irNs
			}
			if svcName == "" {
				continue
			}
			key := svcNs + "/" + svcName
			if !svcSet[key] {
				findings = append(findings, Finding{
					Component:   fmt.Sprintf("IngressRoute/%s/%s", irNs, irName),
					Severity:    SeverityCritical,
					Message:     fmt.Sprintf("IngressRoute %s/%s: route[%d] references Service %s/%s which does not exist", irNs, irName, i, svcNs, svcName),
					Remediation: fmt.Sprintf("Create the missing Service or update the IngressRoute's backend: kubectl get svc -n %s", svcNs),
				})
			}
		}

		// F2: missing Middleware reference
		rawMws, _, _ := unstructured.NestedSlice(route, "middlewares")
		for _, rawMw := range rawMws {
			mw, ok := rawMw.(map[string]interface{})
			if !ok {
				continue
			}
			mwName, _ := mw["name"].(string)
			mwNs, _ := mw["namespace"].(string)
			if mwNs == "" {
				mwNs = irNs
			}
			if mwName == "" {
				continue
			}
			key := mwNs + "/" + mwName
			if !mwSet[key] {
				findings = append(findings, Finding{
					Component:   fmt.Sprintf("IngressRoute/%s/%s", irNs, irName),
					Severity:    SeverityWarning,
					Message:     fmt.Sprintf("IngressRoute %s/%s: route[%d] references Middleware %s/%s which does not exist", irNs, irName, i, mwNs, mwName),
					Remediation: fmt.Sprintf("Create the missing Middleware CRD or remove the reference: kubectl get middlewares.traefik.io -n %s", mwNs),
				})
			}
		}
	}

	// F3: TLS enabled but no certResolver, secretName, or TLSStore
	tlsField, hasTLS, _ := unstructured.NestedMap(ir.Object, "spec", "tls")
	if hasTLS && tlsField != nil {
		certResolver, _ := tlsField["certResolver"].(string)
		secretName, _ := tlsField["secretName"].(string)
		hasTLSStore := tlsSet[irNs+"/default"] || tlsSet["default/default"]

		if certResolver == "" && secretName == "" && !hasTLSStore {
			findings = append(findings, Finding{
				Component:   fmt.Sprintf("IngressRoute/%s/%s", irNs, irName),
				Severity:    SeverityWarning,
				Message:     fmt.Sprintf("IngressRoute %s/%s: TLS is enabled but no certResolver, secretName, or default TLSStore found", irNs, irName),
				Remediation: "Set spec.tls.certResolver, spec.tls.secretName, or create a TLSStore default in the route's namespace or the default namespace",
			})
		}
	}

	return findings
}

func checkIngressRouteTCP(
	tcp *unstructured.Unstructured,
	svcSet map[string]bool,
) []Finding {
	tcpNs := tcp.GetNamespace()
	tcpName := tcp.GetName()

	routes, _, _ := unstructured.NestedSlice(tcp.Object, "spec", "routes")

	var findings []Finding
	for i, rawRoute := range routes {
		route, ok := rawRoute.(map[string]interface{})
		if !ok {
			continue
		}
		rawSvcs, _, _ := unstructured.NestedSlice(route, "services")
		for _, rawSvc := range rawSvcs {
			svc, ok := rawSvc.(map[string]interface{})
			if !ok {
				continue
			}
			svcName, _ := svc["name"].(string)
			svcNs, _ := svc["namespace"].(string)
			if svcNs == "" {
				svcNs = tcpNs
			}
			if svcName == "" {
				continue
			}
			key := svcNs + "/" + svcName
			if !svcSet[key] {
				findings = append(findings, Finding{
					Component:   fmt.Sprintf("IngressRouteTCP/%s/%s", tcpNs, tcpName),
					Severity:    SeverityCritical,
					Message:     fmt.Sprintf("IngressRouteTCP %s/%s: route[%d] references Service %s/%s which does not exist", tcpNs, tcpName, i, svcNs, svcName),
					Remediation: fmt.Sprintf("Create the missing Service or update the IngressRouteTCP's backend: kubectl get svc -n %s", svcNs),
				})
			}
		}
	}
	return findings
}
