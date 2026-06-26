// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package probe

import "k8s.io/apimachinery/pkg/runtime/schema"

// GVRWatcher is the optional contract a Probe can satisfy to declare
// which Kubernetes resource kinds it cares about. The watcher reads
// this declaration to scope future per-probe dispatch (M7 — operator-
// mode refactor).
//
// Phase 1: declarations are recorded but the watcher still runs every
// probe on every debounced trigger (matches pre-M7 semantics). The
// declarations are immediately useful for documentation and for the
// `srenix catalog` subcommand to render an accurate "what does this
// probe watch?" column.
//
// Phase 2 (separate PR, future release): the watcher's dispatch loop
// switches to per-probe gating — only probes whose declared GVRs
// intersect the trigger's source GVR are run. Probes that don't
// implement GVRWatcher fall back to "run on every trigger" so the
// change is backwards-compatible.
//
// Phase 3 (further future): controller-runtime / kubebuilder
// reconciler per probe, fully replacing the debounced fan-out model.
type GVRWatcher interface {
	// GVRs returns the resource kinds this probe consumes. Returning
	// nil = "I consume nothing specific; run me on every trigger" —
	// the safe default when a probe is too cross-cutting to enumerate
	// (e.g. it reads Events + Pods + ConfigMaps + ...).
	GVRs() []schema.GroupVersionResource
}

// GVRsOf returns the declared GVRs for p when p satisfies GVRWatcher,
// or nil when it doesn't. Callers should treat nil as "all triggers".
//
// This helper exists so future call sites (watcher dispatcher, catalog
// renderer) don't need a type assertion at every read site.
func GVRsOf(p Probe) []schema.GroupVersionResource {
	if w, ok := p.(GVRWatcher); ok {
		return w.GVRs()
	}
	return nil
}
