// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package silence

import (
	"context"
	"encoding/json"
	"fmt"

	chav1alpha1 "github.com/srenix-ai/agentic-sre/api/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// Lister fetches the active Silence set known to the cluster. The
// watch loop calls this once per cycle. nil → no filtering (the
// watcher accepts a nil lister to mean "Silence CRD not installed").
type Lister interface {
	List(ctx context.Context) ([]chav1alpha1.Silence, error)
}

// silenceGVR is the canonical GVR for Silence CRs.
var silenceGVR = schema.GroupVersionResource{
	Group:    chav1alpha1.GroupVersion.Group,
	Version:  chav1alpha1.GroupVersion.Version,
	Resource: "silences",
}

// K8sLister is the production Lister, backed by a dynamic client. It
// lists Silences across ALL namespaces — operators may create them
// wherever the affected workload lives, and Srenix reads them all.
//
// Designed to be cheap: one paginated List per call, no informer or
// cache. The watcher invokes it once per cycle (typically 10s+), so
// the apiserver load is negligible. If a future cluster sprouts
// thousands of silences, swapping in an informer-backed cache is
// straightforward.
type K8sLister struct {
	client dynamic.Interface
}

// NewK8sLister constructs a K8s-backed Lister. nil client returns
// nil (so the watcher falls back to no-filtering rather than panicking
// on a partially-wired install).
func NewK8sLister(client dynamic.Interface) *K8sLister {
	if client == nil {
		return nil
	}
	return &K8sLister{client: client}
}

// List returns every Silence currently in the apiserver across all
// namespaces. Returns (nil, nil) when the CRD is not installed
// (apiserver responds NotFound on the GVR) — same semantics as a
// pre-Silence install: just don't filter anything.
func (l *K8sLister) List(ctx context.Context) ([]chav1alpha1.Silence, error) {
	if l == nil || l.client == nil {
		return nil, nil
	}
	uList, err := l.client.Resource(silenceGVR).Namespace("").List(ctx, metav1.ListOptions{})
	if err != nil {
		// The CRD may not be installed yet (early-adopter cluster on
		// chart < 1.9). Bubble up — callers decide whether to log + skip.
		return nil, fmt.Errorf("silence: list %s: %w", silenceGVR.String(), err)
	}
	out := make([]chav1alpha1.Silence, 0, len(uList.Items))
	for i := range uList.Items {
		raw, err := uList.Items[i].MarshalJSON()
		if err != nil {
			// One bad CR shouldn't kill the whole filter — skip it.
			continue
		}
		var s chav1alpha1.Silence
		if err := json.Unmarshal(raw, &s); err != nil {
			continue
		}
		out = append(out, s)
	}
	return out, nil
}

// StaticLister returns a fixed Silence list, useful for tests and for
// callers that already have the typed objects in memory.
type StaticLister struct {
	Items []chav1alpha1.Silence
}

// List satisfies Lister.
func (s StaticLister) List(_ context.Context) ([]chav1alpha1.Silence, error) {
	return s.Items, nil
}
