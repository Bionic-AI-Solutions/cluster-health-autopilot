// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package probe

import (
	"context"
	"fmt"
	"strings"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
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

// DefaultTargets returns the same list as the bash CRITICAL_SERVICES array
// at cluster-health-report.sh:130-167. Provided as a default; callers can
// override via config in a future release.
func DefaultTargets() []ServiceTarget {
	return []ServiceTarget{
		{"redis", "app=redis-cluster-ceph", "Redis Cluster"},
		{"kong", "app.kubernetes.io/name=kong", "Kong Gateway"},
		{"langfuse", "app=web", "Langfuse Web"},
		{"letta", "app.kubernetes.io/name=letta-server", "Letta AI Server"},
		{"letta", "app.kubernetes.io/name=falkordb", "FalkorDB (Letta)"},
		{"letta", "app.kubernetes.io/name=graphiti-service", "Graphiti (Letta)"},
		{"letta", "app.kubernetes.io/name=media-bridge", "Media Bridge (Letta)"},
		{"search-infrastructure", "app=searxng", "SearXNG Search"},
		{"search-infrastructure", "app=crawl4ai", "Crawl4AI"},
		{"mcp", "app=search-mcp-server", "Search MCP"},
		{"mcp", "app=mcp-letta-server", "Letta MCP"},
		{"mcp", "app=mcp-postgres-server", "Postgres MCP"},
		{"mcp", "app=mcp-redis-server", "Redis MCP"},
		{"mcp", "app=mcp-minio-server", "MinIO MCP"},
		{"mcp", "app=mcp-ai-mcp-server", "AI MCP"},
		{"mcp", "app=mcp-calculator-server", "Calculator MCP"},
		{"mcp", "app=mcp-ffmpeg-server", "FFmpeg MCP"},
		{"mcp", "app=mcp-genimage-server", "GenImage MCP"},
		{"mcp", "app=mcp-langfuse-server", "Langfuse MCP"},
		{"mcp", "app=mcp-mail-server", "Mail MCP"},
		{"mcp", "app=mcp-meilisearch-server", "MeiliSearch MCP"},
		{"mcp", "app=mcp-openproject-server", "OpenProject MCP"},
		{"mcp", "app=mcp-pdf-generator-server", "PDF Generator MCP"},
		{"livekit", "app=livekit-server", "LiveKit Server"},
		{"livekit", "app=livekit-egress", "LiveKit Egress"},
		{"livekit", "app=livekit-sip-server", "LiveKit SIP"},
		{"vc-livekit", "app=backend", "VC Backend"},
		{"vc-livekit", "app=frontend", "VC Frontend"},
		{"vc-livekit", "app=livekit-agent", "VC LiveKit Agent"},
		{"nextcloud", "app.kubernetes.io/name=nextcloud", "NextCloud"},
		{"web", "app.kubernetes.io/name=baisoln-web", "Bionic Web"},
		{"web", "app.kubernetes.io/name=contact-api", "Contact API"},
	}
}
