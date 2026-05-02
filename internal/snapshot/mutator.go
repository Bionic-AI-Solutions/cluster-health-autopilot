// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package snapshot

import (
	"context"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
)

// Mutator is the live-only contract that allows fixers to mutate cluster state.
//
// Snapshot sources (the offline file-backed variant) do NOT implement this
// interface — the type system enforces that no fixer can accidentally run
// against a snapshot. Fixers must call AsMutator on their snapshot.Source
// argument and refuse to act when the result is nil.
//
// The interface is intentionally minimal: only the verbs each whitelisted
// fixer actually needs (Delete, Patch). Adding new verbs is a deliberate
// review event — keep the surface tight.
type Mutator interface {
	// Delete removes a single resource. Background propagation by default.
	Delete(ctx context.Context, gvr schema.GroupVersionResource, ns, name string) error

	// Patch applies a strategic-merge or JSON-merge patch to a single resource.
	// patchType is one of types.{StrategicMergePatchType, MergePatchType, JSONPatchType}.
	Patch(ctx context.Context, gvr schema.GroupVersionResource, ns, name string, patchType types.PatchType, patch []byte) error
}

// AsMutator returns the source as a Mutator if it supports live mutation,
// or nil otherwise. Fixers must check this and refuse on nil.
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
