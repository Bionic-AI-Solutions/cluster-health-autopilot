// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package probe

import (
	"context"
	"fmt"
	"strings"

	"github.com/srenix-ai/agentic-sre/internal/snapshot"
)

// ServiceTarget describes a Deployment-shaped workload to probe.
//
// Mirrors the CRITICAL_SERVICES list from cluster-health-report.sh:130-167
// but is now data-driven (no longer hardcoded into a probe).
type ServiceTarget struct {
	Namespace string `yaml:"namespace" json:"namespace"`
	Selector  string `yaml:"selector"  json:"selector"` // "app=redis-cluster" — single label match for now
	Display   string `yaml:"display"   json:"display"`
}

// Services ports probe_services from cluster-health-report.sh:168-247.
//
// Critical detail vs. the bash version: counts pods by the READY column
// (X/Y, X==Y) rather than by status.phase=Running. Pods stuck in
// CreateContainerConfigError report phase=Running but never start —
// counting by phase masks the failure.
type Services struct {
	Targets []ServiceTarget
}

// Name returns the component label for the report.
func (Services) Name() string { return "Critical Services" }

// Run executes the per-target service-readiness probe.
func (s Services) Run(ctx context.Context, src snapshot.Source) Result {
	r := Result{Component: ComponentResult{Component: "Critical Services"}}

	issues := 0
	warnings := 0
	healthy := 0

	for _, t := range s.Targets {
		key, val, ok := splitSelector(t.Selector)
		if !ok {
			continue
		}
		list, err := src.List(ctx, snapshot.GVRPod, t.Namespace)
		if err != nil {
			// Per-namespace API error — surface as a probe-level finding but keep
			// going for other targets.
			r.Findings = append(r.Findings, Finding{
				Component: "Service: " + t.Display,
				Severity:  SeverityCritical,
				Message:   fmt.Sprintf("Pod list failed in ns %q: %v", t.Namespace, err),
			})
			issues++
			continue
		}
		matched := 0
		ready := 0
		for _, pod := range list.Items {
			if isTerminating(pod) {
				continue
			}
			labels := pod.GetLabels()
			if labels[key] != val {
				continue
			}
			matched++
			if podIsReady(pod.Object) {
				ready++
			}
		}
		switch {
		case matched == 0:
			// No pods match — silently skip (the workload isn't deployed in this cluster).
			continue
		case ready == 0:
			r.Findings = append(r.Findings, Finding{
				Component:   "Service: " + t.Display,
				Severity:    SeverityCritical,
				Message:     fmt.Sprintf("No ready pods (0/%d)", matched),
				Remediation: fmt.Sprintf("Check: `kubectl get pods -n %s -l %s`", t.Namespace, t.Selector),
			})
			issues++
		case ready < matched:
			r.Findings = append(r.Findings, Finding{
				Component: "Service: " + t.Display,
				Severity:  SeverityWarning,
				Message:   fmt.Sprintf("Degraded (%d/%d pods ready)", ready, matched),
			})
			warnings++
		default:
			healthy++
		}
	}

	switch {
	case issues == 0 && warnings == 0:
		r.Component.Status = "HEALTHY"
		r.Component.Detail = fmt.Sprintf("All %d critical services operational", healthy)
	case issues == 0:
		r.Component.Status = "DEGRADED"
		r.Component.Detail = fmt.Sprintf("%d service(s) degraded, %d healthy", warnings, healthy)
	default:
		r.Component.Status = "CRITICAL"
		r.Component.Detail = fmt.Sprintf("%d service(s) down, %d degraded, %d healthy", issues, warnings, healthy)
	}
	return r
}

// splitSelector handles "key=value" form. Returns (key, value, ok).
// More complex selectors (multiple labels, set-based) are intentionally
// out of scope for v0.1 — the bash equivalent only supported single-label.
func splitSelector(sel string) (string, string, bool) {
	parts := strings.SplitN(sel, "=", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// podIsReady walks a pod's containerStatuses and returns true iff every
// container reports ready=true. This matches the bash awk that counted by
// the READY column "X/Y" requiring X==Y.
func podIsReady(pod map[string]any) bool {
	statuses, ok, _ := getSliceField(pod, "status", "containerStatuses")
	if !ok || len(statuses) == 0 {
		return false
	}
	for _, cs := range statuses {
		csm, ok := cs.(map[string]any)
		if !ok {
			return false
		}
		ready, _ := csm["ready"].(bool)
		if !ready {
			return false
		}
	}
	return true
}

// DefaultTargets returns the OSS-default ServiceTarget list. Srenix OSS ships
// EMPTY — bundling specific workload selectors here would silently make
// every install probe whichever organization's apps were baked in at build
// time. That's a cluster-leak. The pre-1.10.5 default list was a Srenix-
// development-cluster fixture (Letta + MCP fan-out + LiveKit + web app
// selectors) that never belonged in OSS.
//
// Sources of ServiceTargets in production, in precedence order:
//
//  1. Auto-discovery — every Deployment/StatefulSet in the cluster is already
//     a candidate. Critical-service probes are derived from workload state
//     (ReadyReplicas vs Replicas, restart counts, last-seen ready).
//  2. (Phase 2d, paid AI tier) Cluster RAG memory learns workload importance
//     from observation: traffic patterns, label hints (`tier: critical`),
//     finding-history (workloads that have been Approved/Denied by SREs).
//     The learned set replays as ServiceTargets each cycle.
//  3. Operator-supplied override — a future spec.probe.serviceTargets field
//     on the CR will let SREs hand-curate when needed (rare).
//
// Callers that want a literal target list can still pass one in directly via
// the Services{Targets: ...} field — that path is unchanged.
func DefaultTargets() []ServiceTarget {
	return nil
}
