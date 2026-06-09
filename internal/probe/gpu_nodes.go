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
)

// GPUNodes is the M3 probe (trigger-expansion roadmap, v1.7+).
//
// Reads every Node, identifies those whose status.allocatable carries
// an `nvidia.com/gpu` (or `amd.com/gpu`) quantity, and verifies:
//
//  1. The node is Ready (status.conditions Ready=True)
//  2. The node is not cordoned (spec.unschedulable=false)
//  3. The allocatable GPU count is non-zero (driver crash → kubelet
//     re-advertises 0 GPUs, the workload becomes unschedulable but
//     the Pod stays Pending forever)
//
// Silent on clusters with no GPU nodes (no `*.gpu` keys in any node's
// allocatable). Toggle off via CHA_PROBE_GPU_NODES=off.
type GPUNodes struct{}

// Name satisfies probe.Probe.
func (GPUNodes) Name() string { return "GPU Nodes" }

// gpuResourceKeys lists the K8s extended-resource keys we treat as GPU
// indicators. Order matters only for stability of reporting.
var gpuResourceKeys = []string{
	"nvidia.com/gpu",
	"amd.com/gpu",
}

// Run satisfies probe.Probe.
func (g GPUNodes) Run(ctx context.Context, src snapshot.Source) probe.Result {
	r := probe.Result{Component: probe.ComponentResult{Component: "GPU Nodes"}}

	nodes, err := src.List(ctx, snapshot.GVRNode, "")
	if err != nil {
		r.Component.Status = "PROBE_FAILED"
		r.Component.Detail = "list nodes: " + err.Error()
		return r
	}

	var inspected int
	for i := range nodes.Items {
		n := &nodes.Items[i]
		gpuKey, gpuCount := nodeGPUAllocatable(n)
		if gpuKey == "" {
			continue
		}
		inspected++
		name := n.GetName()
		subject := "Node/" + name

		// 1. Ready condition.
		if !nodeReady(n) {
			r.Findings = append(r.Findings, probe.Finding{
				Component: "GPU Nodes",
				Severity:  probe.SeverityCritical,
				Message: fmt.Sprintf(
					"GPU node %s is NotReady — every GPU workload scheduled to it will hang in Pending until kubelet recovers.",
					subject),
				Remediation: fmt.Sprintf(
					"Inspect kubelet + node-problem-detector:\n  kubectl describe node %s\n  ssh %s journalctl -u kubelet --since '10 min ago'",
					name, name),
			})
		}
		// 2. Cordoned.
		unsched, _, _ := unstructured.NestedBool(n.Object, "spec", "unschedulable")
		if unsched {
			r.Findings = append(r.Findings, probe.Finding{
				Component: "GPU Nodes",
				Severity:  probe.SeverityWarning,
				Message: fmt.Sprintf(
					"GPU node %s is cordoned. If maintenance is complete, uncordon to restore scheduling.",
					subject),
				Remediation: fmt.Sprintf(
					"  kubectl uncordon %s",
					name),
			})
		}
		// 3. Allocatable GPU count is zero.
		if gpuCount == 0 {
			r.Findings = append(r.Findings, probe.Finding{
				Component: "GPU Nodes",
				Severity:  probe.SeverityCritical,
				Message: fmt.Sprintf(
					"GPU node %s advertises 0 allocatable %s — likely an NVIDIA driver crash or the device-plugin pod is unhealthy. GPU workloads will hang in Pending.",
					subject, gpuKey),
				Remediation: fmt.Sprintf(
					"Inspect the device-plugin DaemonSet pod on this node:\n  kubectl get pods -n kube-system -l name=nvidia-device-plugin --field-selector spec.nodeName=%s\n  kubectl logs -n kube-system <pod> -c nvidia-device-plugin --tail=100\n\nThen check the driver:\n  ssh %s nvidia-smi",
					name, name),
			})
		}
	}

	switch {
	case inspected == 0:
		r.Component.Status = "OK"
		r.Component.Detail = "no GPU nodes detected"
	case len(r.Findings) == 0:
		r.Component.Status = "OK"
		r.Component.Detail = fmt.Sprintf("%d GPU node(s) — all Ready + Schedulable with allocatable GPUs", inspected)
	default:
		r.Component.Status = "WARNING"
		r.Component.Detail = fmt.Sprintf("%d issue(s) across %d GPU node(s)", len(r.Findings), inspected)
	}
	return r
}

// nodeGPUAllocatable returns the first GPU resource key found in the
// node's status.allocatable map plus its integer count (0 if the
// quantity parses to zero). Returns ("", 0) on a non-GPU node.
func nodeGPUAllocatable(n *unstructured.Unstructured) (string, int64) {
	alloc, found, _ := unstructured.NestedMap(n.Object, "status", "allocatable")
	if !found {
		return "", 0
	}
	for _, k := range gpuResourceKeys {
		v, ok := alloc[k]
		if !ok {
			continue
		}
		// status.allocatable values are stringified quantities ("4",
		// "1", etc). Reject anything we can't parse to keep the probe
		// honest about its inputs.
		s, _ := v.(string)
		s = strings.TrimSpace(s)
		if s == "" {
			return k, 0
		}
		var n int64
		_, err := fmt.Sscanf(s, "%d", &n)
		if err != nil {
			return k, 0
		}
		return k, n
	}
	return "", 0
}

// nodeReady returns true when status.conditions has Type=Ready,
// Status=True. Missing conditions → false (operator can't know).
func nodeReady(n *unstructured.Unstructured) bool {
	conds, _, _ := unstructured.NestedSlice(n.Object, "status", "conditions")
	for _, c := range conds {
		cm, ok := c.(map[string]any)
		if !ok {
			continue
		}
		t, _ := cm["type"].(string)
		if t != "Ready" {
			continue
		}
		s, _ := cm["status"].(string)
		return s == "True"
	}
	return false
}
