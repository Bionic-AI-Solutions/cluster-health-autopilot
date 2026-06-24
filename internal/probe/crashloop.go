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

// CrashLoopBackOff catches pods stuck in CrashLoopBackOff anywhere in the
// cluster, not just on the hardcoded critical-services list. The Services
// probe only watches workloads CHA was told about; this probe catches the
// rest.
//
// Severity scaling:
//   - In protected namespaces (kube-system etc.) → CRITICAL.
//   - Elsewhere → WARNING by default; promote to CRITICAL once the pod
//     accumulates more than CriticalRestartThreshold restarts (default 10).
//     This gives a deploy time to settle without immediately paging.
type CrashLoopBackOff struct {
	// CriticalRestartThreshold is the restart count above which a non-
	// protected-namespace pod's loop is escalated from WARNING to CRITICAL.
	// Default: 10.
	CriticalRestartThreshold int64
}

// Name returns the component label for the report.
func (CrashLoopBackOff) Name() string { return "CrashLoopBackOff" }

// Run executes the probe.
func (c CrashLoopBackOff) Run(ctx context.Context, src snapshot.Source) Result {
	r := Result{Component: ComponentResult{Component: "CrashLoopBackOff"}}
	threshold := c.CriticalRestartThreshold
	if threshold == 0 {
		threshold = 10
	}

	pods, err := src.List(ctx, snapshot.GVRPod, "")
	if err != nil {
		r.Component.Status = "PROBE_FAILED"
		r.Component.Detail = "list pods: " + err.Error()
		return r
	}

	type hit struct {
		key      string
		restarts int64
		reason   string
	}
	var critical, warning []hit
	for _, pod := range pods.Items {
		if isTerminating(pod) {
			continue
		}
		// We don't restrict to Running — kubelet sometimes reports
		// Pending/CrashLoopBackOff during init-container retries.
		restarts, found, reason := podMaxRestartCount(pod)
		if !found {
			continue
		}
		if !strings.Contains(reason, "CrashLoopBackOff") {
			continue
		}
		ns := pod.GetNamespace()
		h := hit{
			key:      ns + "/" + pod.GetName(),
			restarts: restarts,
			reason:   reason,
		}
		if IsProtectedNamespace(ns) || restarts > threshold {
			critical = append(critical, h)
		} else {
			warning = append(warning, h)
		}
	}

	if len(critical) == 0 && len(warning) == 0 {
		r.Component.Status = "HEALTHY"
		r.Component.Detail = "No CrashLoopBackOff pods detected"
		return r
	}

	sort.Slice(critical, func(i, j int) bool { return critical[i].key < critical[j].key })
	sort.Slice(warning, func(i, j int) bool { return warning[i].key < warning[j].key })

	emit := func(sev Severity, hits []hit) {
		for _, h := range hits {
			r.Findings = append(r.Findings, Finding{
				Component: "CrashLoopBackOff",
				Severity:  sev,
				Message: fmt.Sprintf("Pod %s in CrashLoopBackOff (%d restarts)",
					h.key, h.restarts),
				Remediation: fmt.Sprintf(
					"Inspect crash cause: `kubectl logs %s -n %s --previous`. "+
						"Common causes: bad config, missing dependency, OOM in the workload, broken liveness probe.",
					splitName(h.key), splitNs(h.key)),
			})
		}
	}
	emit(SeverityCritical, critical)
	emit(SeverityWarning, warning)

	parts := []string{}
	if len(critical) > 0 {
		ks := make([]string, len(critical))
		for i, h := range critical {
			ks[i] = h.key
		}
		parts = append(parts, fmt.Sprintf("%d critical (%s)", len(critical), strings.Join(ks, ",")))
	}
	if len(warning) > 0 {
		ks := make([]string, len(warning))
		for i, h := range warning {
			ks[i] = h.key
		}
		parts = append(parts, fmt.Sprintf("%d warning (%s)", len(warning), strings.Join(ks, ",")))
	}

	if len(critical) > 0 {
		r.Component.Status = "CRITICAL"
	} else {
		r.Component.Status = "WARNING"
	}
	r.Component.Detail = strings.Join(parts, "; ")
	return r
}

// podMaxRestartCount scans the pod's containerStatuses (including init
// containers) and returns the restart count and waiting reason from the
// single container that has the highest restart count while currently in a
// waiting state. This ensures the restart count and reason always describe
// the SAME container — previously they could come from unrelated containers,
// producing misleading messages like "50 restarts" for a CrashLoopBackOff
// that actually had 1 restart on a different container.
//
// The found bool is true only when at least one container is currently
// waiting with a non-empty reason.
func podMaxRestartCount(pod unstructured.Unstructured) (int64, bool, string) {
	statuses, _, _ := getSliceField(pod.Object, "status", "containerStatuses")
	statuses = append(statuses, mustSlice(pod.Object, "status", "initContainerStatuses")...)
	if len(statuses) == 0 {
		return 0, false, ""
	}
	var maxRestarts int64
	var reason string
	found := false
	for _, s := range statuses {
		sm, ok := s.(map[string]any)
		if !ok {
			continue
		}
		state, _ := sm["state"].(map[string]any)
		waiting, _ := state["waiting"].(map[string]any)
		r, _ := waiting["reason"].(string)
		if r == "" {
			// Container is not currently in a waiting state — skip.
			// We only report restarts from containers that are actually waiting.
			continue
		}
		rc := asInt64(sm["restartCount"])
		if !found || rc > maxRestarts {
			maxRestarts = rc
			reason = r
			found = true
		}
	}
	return maxRestarts, found, reason
}
