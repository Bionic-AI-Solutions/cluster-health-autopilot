// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package snapshot

import (
	"context"
	"fmt"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	apitypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

// LiveSource is a Source backed by a Kubernetes API server, also
// satisfying Mutator for live cluster mutation.
//
// It uses the dynamic client (untyped) so probes can ask for any GVR
// including CRDs (CNPG, External Secrets, cert-manager) without
// vendoring those CRDs into the build.
//
// This type is the public-facing constructor exposed for paid-tier
// binaries that need to construct a Source/Mutator (e.g. the
// approval-server). The OSS binary continues to use the internal
// implementation; both end up satisfying the same Source/Mutator
// interface contract.
type LiveSource struct {
	client dynamic.Interface
}

// NewLiveSource constructs a LiveSource from an existing rest.Config.
// Useful for paid-tier binaries that build their own client (e.g. for
// custom transport / TLS wiring).
func NewLiveSource(cfg *rest.Config) (*LiveSource, error) {
	dyn, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("dynamic client: %w", err)
	}
	return &LiveSource{client: dyn}, nil
}

// List satisfies Source.
func (l *LiveSource) List(ctx context.Context, gvr schema.GroupVersionResource, ns string) (*unstructured.UnstructuredList, error) {
	if ns == "" {
		return l.client.Resource(gvr).List(ctx, v1.ListOptions{})
	}
	return l.client.Resource(gvr).Namespace(ns).List(ctx, v1.ListOptions{})
}

// Get satisfies Source.
func (l *LiveSource) Get(ctx context.Context, gvr schema.GroupVersionResource, ns, name string) (*unstructured.Unstructured, error) {
	if ns == "" {
		return l.client.Resource(gvr).Get(ctx, name, v1.GetOptions{})
	}
	return l.client.Resource(gvr).Namespace(ns).Get(ctx, name, v1.GetOptions{})
}

// Mode reports live.
func (l *LiveSource) Mode() Mode { return ModeLive }

// Watch returns an apimachinery watch.Interface for the given GVR.
// Used by the watcher to react to cluster events.
func (l *LiveSource) Watch(ctx context.Context, gvr schema.GroupVersionResource) (watch.Interface, error) {
	return l.client.Resource(gvr).Watch(ctx, v1.ListOptions{Watch: true})
}

// Delete satisfies Mutator.
func (l *LiveSource) Delete(ctx context.Context, gvr schema.GroupVersionResource, ns, name string) error {
	if ns == "" {
		return l.client.Resource(gvr).Delete(ctx, name, v1.DeleteOptions{})
	}
	return l.client.Resource(gvr).Namespace(ns).Delete(ctx, name, v1.DeleteOptions{})
}

// Patch satisfies Mutator.
func (l *LiveSource) Patch(ctx context.Context, gvr schema.GroupVersionResource, ns, name string, patchType apitypes.PatchType, patch []byte) error {
	if ns == "" {
		_, err := l.client.Resource(gvr).Patch(ctx, name, patchType, patch, v1.PatchOptions{})
		return err
	}
	_, err := l.client.Resource(gvr).Namespace(ns).Patch(ctx, name, patchType, patch, v1.PatchOptions{})
	return err
}

// Create satisfies Mutator. Restricted to writers (not fixers — fixers
// should call Delete/Patch only).
func (l *LiveSource) Create(ctx context.Context, gvr schema.GroupVersionResource, ns string, obj *unstructured.Unstructured) error {
	if ns == "" {
		_, err := l.client.Resource(gvr).Create(ctx, obj, v1.CreateOptions{})
		return err
	}
	_, err := l.client.Resource(gvr).Namespace(ns).Create(ctx, obj, v1.CreateOptions{})
	return err
}
