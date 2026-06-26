// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package snapshot

import (
	"context"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
)

// AsMutator returns the source as a Mutator if it supports live mutation,
// or nil otherwise. Fixers must check this and refuse on nil.
//
// Defined here for backward compatibility; pkg/snapshot.AsMutator is the
// canonical version available to external catalog packages.
func AsMutator(s Source) Mutator {
	m, _ := s.(Mutator)
	return m
}

// DryRunMutator implements Mutator without performing any real cluster
// mutations. Pass it to fixers when --dry-run is active: they see a non-nil
// Mutator and execute their full evaluation logic (condition checks, candidate
// selection), but every Delete/Patch call is a no-op. The resulting
// fix.Result.Actions list reflects what would have been done.
type DryRunMutator struct{}

// Delete is a no-op in dry-run mode.
func (DryRunMutator) Delete(_ context.Context, _ schema.GroupVersionResource, _, _ string) error {
	return nil
}

// Patch is a no-op in dry-run mode.
func (DryRunMutator) Patch(_ context.Context, _ schema.GroupVersionResource, _, _ string, _ types.PatchType, _ []byte) error {
	return nil
}

// PatchStatus is a no-op in dry-run mode.
func (DryRunMutator) PatchStatus(_ context.Context, _ schema.GroupVersionResource, _, _ string, _ types.PatchType, _ []byte) error {
	return nil
}

// Create is a no-op in dry-run mode.
func (DryRunMutator) Create(_ context.Context, _ schema.GroupVersionResource, _ string, _ *unstructured.Unstructured) error {
	return nil
}

// Delete removes the named resource from the live cluster.
func (l *Live) Delete(ctx context.Context, gvr schema.GroupVersionResource, ns, name string) error {
	var ri dynamic.ResourceInterface
	if ns == "" {
		ri = l.client.Resource(gvr)
	} else {
		ri = l.client.Resource(gvr).Namespace(ns)
	}
	return ri.Delete(ctx, name, v1.DeleteOptions{})
}

// Patch applies the given patch to the named resource on the live cluster.
func (l *Live) Patch(ctx context.Context, gvr schema.GroupVersionResource, ns, name string, patchType types.PatchType, patch []byte) error {
	var ri dynamic.ResourceInterface
	if ns == "" {
		ri = l.client.Resource(gvr)
	} else {
		ri = l.client.Resource(gvr).Namespace(ns)
	}
	_, err := ri.Patch(ctx, name, patchType, patch, v1.PatchOptions{})
	return err
}

// PatchStatus applies the patch to the /status subresource. Required for
// CRDs that declare `subresources.status: {}` — patches to the main
// resource endpoint silently drop status field changes.
func (l *Live) PatchStatus(ctx context.Context, gvr schema.GroupVersionResource, ns, name string, patchType types.PatchType, patch []byte) error {
	var ri dynamic.ResourceInterface
	if ns == "" {
		ri = l.client.Resource(gvr)
	} else {
		ri = l.client.Resource(gvr).Namespace(ns)
	}
	_, err := ri.Patch(ctx, name, patchType, patch, v1.PatchOptions{}, "status")
	return err
}

// Create writes a new resource on the live cluster.
func (l *Live) Create(ctx context.Context, gvr schema.GroupVersionResource, ns string, obj *unstructured.Unstructured) error {
	var ri dynamic.ResourceInterface
	if ns == "" {
		ri = l.client.Resource(gvr)
	} else {
		ri = l.client.Resource(gvr).Namespace(ns)
	}
	_, err := ri.Create(ctx, obj, v1.CreateOptions{})
	return err
}
