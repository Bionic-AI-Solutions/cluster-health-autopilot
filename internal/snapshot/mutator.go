// Copyright 2026 Cluster Health Autopilot contributors
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
