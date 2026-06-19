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

// ETCD inspects the etcd control-plane health via the static pods that
// kubeadm-managed clusters host in kube-system. For external etcd
// deployments (managed services, stacked masters with no in-cluster pods)
// the probe degrades to a Warning that explicitly says "external etcd;
// install etcd-exporter for visibility" — rather than reporting HEALTHY
// for an etcd we can't see.
//
// Heuristic for "is etcd in-cluster":
//   - We look for pods in kube-system labelled component=etcd OR named
//     etcd-<node-name>. This matches kubeadm's static-pod naming.
//   - If any are found, we evaluate them. Any Ready=False or restartCount
//     > 0 on the etcd container is CRITICAL.
//   - If none are found, emit a single Warning so the operator knows the
//     probe is blind, not silently green.
type ETCD struct {
	// Namespace to search for etcd pods. Defaults to "kube-system".
	Namespace string
}

// Name returns the component label for the report.
func (ETCD) Name() string { return "ETCD" }

// Run executes the ETCD probe.
func (e ETCD) Run(ctx context.Context, src snapshot.Source) Result {
	r := Result{Component: ComponentResult{Component: "ETCD"}}
	ns := e.Namespace
	if ns == "" {
		ns = "kube-system"
	}

	pods, err := src.List(ctx, snapshot.GVRPod, ns)
	if err != nil {
		r.Component.Status = "PROBE_FAILED"
		r.Component.Detail = "list pods in " + ns + ": " + err.Error()
		return r
	}

	type member struct {
		name     string
		ready    bool
		restarts int64
	}
	var members []member
	for _, pod := range pods.Items {
		if isTerminating(pod) {
			continue
		}
		if !looksLikeEtcdPod(pod) {
			continue
		}
		m := member{name: pod.GetName()}
		conds, _, _ := getSliceField(pod.Object, "status", "conditions")
		for _, c := range conds {
			cm, ok := c.(map[string]any)
			if !ok {
				continue
			}
			if cm["type"] == "Ready" {
				m.ready = cm["status"] == "True"
			}
		}
		statuses, _, _ := getSliceField(pod.Object, "status", "containerStatuses")
		for _, s := range statuses {
			sm, ok := s.(map[string]any)
			if !ok {
				continue
			}
			if rc := asInt64(sm["restartCount"]); rc > m.restarts {
				m.restarts = rc
			}
		}
		members = append(members, m)
	}

	if len(members) == 0 {
		// External etcd or non-kubeadm install. Emit a Warning — silent
		// green for "we can't see etcd" is the original 2026-04-28 trap
		// the Nodes probe was redesigned to avoid.
		r.Component.Status = "WARNING"
		r.Component.Detail = "No in-cluster etcd pods found in " + ns + " (external etcd or non-kubeadm install)"
		r.Findings = append(r.Findings, Finding{
			Component: "ETCD",
			Severity:  SeverityWarning,
			Message:   "ETCD probe is blind: no in-cluster etcd pods captured. Cluster may be using external etcd.",
			Remediation: "For external etcd: install etcd-exporter on the etcd nodes and scrape via Prometheus. " +
				"For kubeadm in-cluster etcd: confirm static-pod manifests in /etc/kubernetes/manifests/ on control-plane nodes.",
		})
		return r
	}

	sort.Slice(members, func(i, j int) bool { return members[i].name < members[j].name })
	var notReady, restarted []string
	for _, m := range members {
		if !m.ready {
			notReady = append(notReady, m.name)
		}
		if m.restarts > 0 {
			restarted = append(restarted, fmt.Sprintf("%s(%d)", m.name, m.restarts))
		}
	}

	if len(notReady) == 0 && len(restarted) == 0 {
		r.Component.Status = "HEALTHY"
		r.Component.Detail = fmt.Sprintf("All %d etcd member(s) ready", len(members))
		return r
	}

	parts := []string{}
	if len(notReady) > 0 {
		parts = append(parts, fmt.Sprintf("%d not ready: %s", len(notReady), strings.Join(notReady, ",")))
		r.Findings = append(r.Findings, Finding{
			Component: "ETCD",
			Severity:  SeverityCritical,
			Message:   fmt.Sprintf("ETCD member(s) not Ready: %s", strings.Join(notReady, ", ")),
			Remediation: "Inspect the etcd member: `kubectl describe pod <name> -n " + ns + "`. " +
				"Check disk space and IO latency on the host; etcd needs <100ms fsync.",
		})
	}
	if len(restarted) > 0 {
		parts = append(parts, fmt.Sprintf("%d restarted: %s", len(restarted), strings.Join(restarted, ",")))
		r.Findings = append(r.Findings, Finding{
			Component: "ETCD",
			Severity:  SeverityCritical,
			Message:   fmt.Sprintf("ETCD member(s) restarted: %s", strings.Join(restarted, ", ")),
			Remediation: "Check etcd logs on the affected node: `kubectl logs <pod> -n " + ns + " --previous`. " +
				"Common causes: disk IO latency, OOM (raise etcd memory limit), quorum loss.",
		})
	}

	r.Component.Status = "CRITICAL"
	r.Component.Detail = strings.Join(parts, "; ")
	return r
}

func looksLikeEtcdPod(pod unstructured.Unstructured) bool {
	if strings.HasPrefix(pod.GetName(), "etcd-") {
		return true
	}
	labels := pod.GetLabels()
	if labels["component"] == "etcd" {
		return true
	}
	if labels["tier"] == "control-plane" && labels["component"] == "etcd" {
		return true
	}
	return false
}
