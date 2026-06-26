// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package probe

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/srenix-ai/agentic-sre/internal/snapshot"
)

// NodePressure surfaces Node conditions that the basic Nodes probe misses.
// Nodes can be Ready=True while reporting DiskPressure / MemoryPressure /
// PIDPressure — those conditions are how the kubelet says "I can still
// schedule but am about to evict pods." Ignoring them is how a slow-burn
// resource exhaustion turns into a sudden cluster-wide eviction storm.
//
// Sources:
//   - status.conditions[type=DiskPressure].status=True
//   - status.conditions[type=MemoryPressure].status=True
//   - status.conditions[type=PIDPressure].status=True
//   - status.conditions[type=NetworkUnavailable].status=True (also a pressure
//     signal that breaks pod-to-pod traffic, surface as Critical too)
//
// Severity: any pressure condition is reported Warning (the cluster is still
// functional), but DiskPressure escalates to Critical because the kubelet
// will start evicting within minutes once it kicks in.
type NodePressure struct{}

// Name returns the component label for the report.
func (NodePressure) Name() string { return "Node Pressure" }

// Run executes the pressure probe.
func (NodePressure) Run(ctx context.Context, src snapshot.Source) Result {
	r := Result{Component: ComponentResult{Component: "Node Pressure"}}
	list, err := src.List(ctx, snapshot.GVRNode, "")
	if err != nil {
		r.Component.Status = "PROBE_FAILED"
		r.Component.Detail = "list nodes: " + err.Error()
		return r
	}
	if len(list.Items) == 0 {
		r.Component.Status = "PROBE_FAILED"
		r.Component.Detail = "list nodes returned 0 items"
		return r
	}

	// nodes[condition] = sorted list of node names exhibiting the condition.
	pressure := map[string][]string{}
	total := len(list.Items)
	for _, n := range list.Items {
		conds, found, _ := getSliceField(n.Object, "status", "conditions")
		if !found {
			continue
		}
		for _, c := range conds {
			cm, ok := c.(map[string]any)
			if !ok {
				continue
			}
			typ, _ := cm["type"].(string)
			status, _ := cm["status"].(string)
			if status != "True" {
				continue
			}
			switch typ {
			case "DiskPressure", "MemoryPressure", "PIDPressure", "NetworkUnavailable":
				pressure[typ] = append(pressure[typ], n.GetName())
			}
		}
	}

	if len(pressure) == 0 {
		r.Component.Status = "HEALTHY"
		r.Component.Detail = fmt.Sprintf("All %d nodes pressure-clear", total)
		return r
	}

	// Sort condition types so output is stable across runs.
	types := make([]string, 0, len(pressure))
	for k := range pressure {
		types = append(types, k)
	}
	sort.Strings(types)

	// Build a single status detail and one Finding per condition class so
	// downstream rendering can group the alerts cleanly.
	parts := make([]string, 0, len(types))
	maxSev := SeverityWarning
	for _, typ := range types {
		nodes := pressure[typ]
		sort.Strings(nodes)
		parts = append(parts, fmt.Sprintf("%s on %d node(s): %s", typ, len(nodes), strings.Join(nodes, ",")))
		sev := SeverityWarning
		// DiskPressure → eviction within minutes; promote to Critical.
		// NetworkUnavailable → pod-to-pod traffic broken on the node;
		// surfacing it as Warning under-states the impact.
		if typ == "DiskPressure" || typ == "NetworkUnavailable" {
			sev = SeverityCritical
			maxSev = SeverityCritical
		}
		r.Findings = append(r.Findings, Finding{
			Component:   "Node Pressure",
			Severity:    sev,
			Message:     fmt.Sprintf("%d node(s) reporting %s: %s", len(nodes), typ, strings.Join(nodes, ", ")),
			Remediation: nodePressureRemediation(typ),
		})
	}

	if maxSev == SeverityCritical {
		r.Component.Status = "CRITICAL"
	} else {
		r.Component.Status = "WARNING"
	}
	r.Component.Detail = strings.Join(parts, "; ")
	return r
}

func nodePressureRemediation(typ string) string {
	switch typ {
	case "DiskPressure":
		return "Free disk on the node: prune images (`crictl rmi --prune`), clear /var/log, expand the node's root volume, or cordon+drain to reschedule pods"
	case "MemoryPressure":
		return "Investigate node memory: `kubectl top node`, identify heavy pods, consider eviction or moving workloads"
	case "PIDPressure":
		return "Container or process leak on the node: `ps -ef | wc -l`, restart misbehaving daemons, raise kernel.pid_max if needed"
	case "NetworkUnavailable":
		return "Check the node's CNI: `kubectl get pods -n kube-system -l k8s-app=<cni>`, restart the per-node CNI pod, verify routing tables"
	default:
		return "Investigate the named condition on the node"
	}
}
