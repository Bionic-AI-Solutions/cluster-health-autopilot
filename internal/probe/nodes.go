// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package probe

import (
	"context"
	"fmt"

	"github.com/srenix-ai/agentic-sre/internal/snapshot"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Nodes ports probe_nodes from cluster-health-report.sh:249-274.
//
// The bash version distinguished kubectl-error from empty-output to avoid
// the false-green that tripped us on 2026-04-28 (when the runner image
// shipped without kubectl). In Go this distinction is automatic: a
// snapshot.Source that errored returns err; an empty list is an empty list.
type Nodes struct{}

// Name returns the component label for the report.
func (Nodes) Name() string { return "Cluster Nodes" }

// Run executes the node-readiness probe.
func (Nodes) Run(ctx context.Context, src snapshot.Source) Result {
	r := Result{Component: ComponentResult{Component: "Cluster Nodes"}}
	list, err := src.List(ctx, snapshot.GVRNode, "")
	if err != nil {
		r.Component.Status = "PROBE_FAILED"
		r.Component.Detail = "list nodes: " + err.Error()
		r.Findings = append(r.Findings, Finding{
			Component:   "Cluster Nodes",
			Severity:    SeverityCritical,
			Message:     "Node probe failed (API error)",
			Remediation: "Check API connectivity from health-report pod and SA RBAC",
		})
		return r
	}
	if len(list.Items) == 0 {
		r.Component.Status = "PROBE_FAILED"
		r.Component.Detail = "list nodes returned 0 items"
		r.Findings = append(r.Findings, Finding{
			Component:   "Cluster Nodes",
			Severity:    SeverityCritical,
			Message:     "Node probe returned empty result",
			Remediation: "Check API connectivity and SA RBAC",
		})
		return r
	}
	notReady := 0
	total := len(list.Items)
	for _, n := range list.Items {
		conds, found, _ := getSliceField(n.Object, "status", "conditions")
		if !found {
			notReady++
			continue
		}
		ready := false
		for _, c := range conds {
			cm, ok := c.(map[string]any)
			if !ok {
				continue
			}
			if cm["type"] == "Ready" && cm["status"] == "True" {
				ready = true
				break
			}
		}
		if !ready {
			notReady++
		}
	}
	if notReady == 0 {
		r.Component.Status = "HEALTHY"
		r.Component.Detail = fmt.Sprintf("All %d nodes ready", total)
	} else {
		r.Component.Status = "CRITICAL"
		r.Component.Detail = fmt.Sprintf("%d/%d nodes not ready", notReady, total)
		r.Findings = append(r.Findings, Finding{
			Component:   "Cluster Nodes",
			Severity:    SeverityCritical,
			Message:     fmt.Sprintf("%d node(s) not ready", notReady),
			Remediation: "Check: `kubectl get nodes` and `kubectl describe node <name>`",
		})
	}
	return r
}

// getSliceField walks a nested map and returns a []any at the given path.
func getSliceField(m map[string]any, path ...string) ([]any, bool, error) {
	cur := any(m)
	for i, k := range path {
		mp, ok := cur.(map[string]any)
		if !ok {
			return nil, false, nil
		}
		cur, ok = mp[k]
		if !ok {
			return nil, false, nil
		}
		if i == len(path)-1 {
			sl, ok := cur.([]any)
			if !ok {
				return nil, false, nil
			}
			return sl, true, nil
		}
	}
	return nil, false, nil
}

// isTerminating returns true when the pod has a non-nil deletionTimestamp,
// meaning the kubelet has been asked to stop and delete it. Terminating pods
// are intentionally going away; flagging them as stuck/not-ready/crashlooping
// produces "already resolved" noise and useless AI remediation proposals.
func isTerminating(pod unstructured.Unstructured) bool {
	return pod.GetDeletionTimestamp() != nil
}
