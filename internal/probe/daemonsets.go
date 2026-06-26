// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package probe

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/srenix-ai/agentic-sre/internal/snapshot"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// DaemonSets surfaces unhealthy DaemonSets in the system-critical namespaces.
// A broken CNI (Cilium, Calico, Flannel), CSI plugin (rook-ceph, longhorn),
// or kube-proxy pod silently starves the cluster — workloads stay Ready while
// pod-to-pod traffic fails or new mounts hang. The Nodes probe catches the
// downstream symptom (Ready=False after grace period), but the kubelet often
// keeps the node Ready for minutes while the underlying daemon is down.
//
// Default scope: namespaces that ship critical cluster-level DaemonSets.
// Configurable via the SystemNamespaces field — overridable from the
// catalog or via the SRENIX_DAEMONSET_NAMESPACES env when the watcher wires
// it up. Empty SystemNamespaces falls back to DefaultDaemonSetNamespaces.
type DaemonSets struct {
	// SystemNamespaces is the list of namespaces this probe inspects.
	// Empty → DefaultDaemonSetNamespaces (kube-system + common CNI/CSI ns).
	SystemNamespaces []string
}

// DefaultDaemonSetNamespaces lists the namespaces where a broken DaemonSet
// degrades the cluster's ability to run any workload. Order doesn't matter
// for correctness; kept stable for readable test output.
var DefaultDaemonSetNamespaces = []string{
	"kube-system",
	"cilium-system",
	"calico-system",
	"kube-flannel",
	"rook-ceph",
	"longhorn-system",
	"openebs",
	"metallb-system",
}

// Name returns the component label for the report.
func (DaemonSets) Name() string { return "System DaemonSets" }

// Run executes the DaemonSet probe.
func (d DaemonSets) Run(ctx context.Context, src snapshot.Source) Result {
	r := Result{Component: ComponentResult{Component: "System DaemonSets"}}

	ns := d.SystemNamespaces
	if len(ns) == 0 {
		ns = DefaultDaemonSetNamespaces
	}
	nsSet := map[string]struct{}{}
	for _, n := range ns {
		nsSet[n] = struct{}{}
	}

	list, err := src.List(ctx, snapshot.GVRDaemonSet, "")
	if err != nil {
		r.Component.Status = "PROBE_FAILED"
		r.Component.Detail = "list daemonsets: " + err.Error()
		return r
	}

	type dsState struct {
		key         string // ns/name
		desired     int64
		ready       int64
		unavailable int64
	}
	var unhealthy []dsState
	var inspected int
	for _, ds := range list.Items {
		if _, watched := nsSet[ds.GetNamespace()]; !watched {
			continue
		}
		inspected++
		st := dsState{
			key: ds.GetNamespace() + "/" + ds.GetName(),
		}
		st.desired, _, _ = unstructured.NestedInt64(ds.Object, "status", "desiredNumberScheduled")
		st.ready, _, _ = unstructured.NestedInt64(ds.Object, "status", "numberReady")
		st.unavailable, _, _ = unstructured.NestedInt64(ds.Object, "status", "numberUnavailable")
		// Genuine unhealth: at least one node should be scheduled and at
		// least one isn't ready. A DaemonSet with desired=0 (rare; e.g.
		// nodeSelector matches nothing right now) is intentionally idle.
		if st.desired > 0 && st.ready < st.desired {
			unhealthy = append(unhealthy, st)
		}
	}

	if inspected == 0 {
		r.Component.Status = "HEALTHY"
		r.Component.Detail = "No DaemonSets in system namespaces (or none captured)"
		return r
	}
	if len(unhealthy) == 0 {
		r.Component.Status = "HEALTHY"
		r.Component.Detail = fmt.Sprintf("All %d system DaemonSets fully scheduled", inspected)
		return r
	}

	sort.Slice(unhealthy, func(i, j int) bool { return unhealthy[i].key < unhealthy[j].key })
	parts := make([]string, 0, len(unhealthy))
	for _, ds := range unhealthy {
		parts = append(parts, fmt.Sprintf("%s (%d/%d ready, %d unavailable)",
			ds.key, ds.ready, ds.desired, ds.unavailable))
		r.Findings = append(r.Findings, Finding{
			Component: "System DaemonSets",
			Severity:  SeverityCritical,
			Message: fmt.Sprintf("DaemonSet %s is degraded: %d/%d pods ready, %d unavailable",
				ds.key, ds.ready, ds.desired, ds.unavailable),
			Remediation: fmt.Sprintf(
				"Investigate the per-node pods: `kubectl get pods -n %s -l <ds-selector>`. "+
					"Common causes: image pull failure, node taints not tolerated, init-container OOM. "+
					"Check `kubectl describe ds %s` for the controller's last failure event.",
				splitNs(ds.key), splitName(ds.key)),
		})
	}
	r.Component.Status = "CRITICAL"
	r.Component.Detail = fmt.Sprintf("%d/%d system DaemonSets degraded: %s",
		len(unhealthy), inspected, strings.Join(parts, "; "))
	return r
}

func splitNs(key string) string {
	if i := strings.IndexByte(key, '/'); i >= 0 {
		return key[:i]
	}
	return ""
}

func splitName(key string) string {
	if i := strings.IndexByte(key, '/'); i >= 0 {
		return key[i+1:]
	}
	return key
}
