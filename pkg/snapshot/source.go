// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

// Package snapshot defines the Source and Mutator interfaces that decouple
// probes, analyzers, and fixers from the underlying data layer.
//
// This package is part of the exported API surface. External pattern catalogs
// (paid tier, community plugins) must import this package to implement
// Analyzers and Fixers — the interfaces here are the only types they need
// from this module besides pkg/diagnose, pkg/fix, and pkg/probe.
package snapshot

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

// Source returns Kubernetes objects to a probe or analyzer.
//
// Two implementations ship in the OSS binary:
//   - Live: backed by a Kubernetes API server via client-go.
//   - File: backed by a directory of kubectl get -o json outputs (zero-trust
//     offline mode — no install, no RBAC, no write permissions).
//
// External pattern authors receive a Source at Run() time; they should never
// need to construct one directly.
type Source interface {
	// List returns all instances of gvr, optionally filtered by namespace.
	// Empty ns means all namespaces for namespaced resources.
	// Returns an empty list (not an error) when the CRD is not installed.
	List(ctx context.Context, gvr schema.GroupVersionResource, ns string) (*unstructured.UnstructuredList, error)

	// Get returns a single instance by namespace and name.
	Get(ctx context.Context, gvr schema.GroupVersionResource, ns, name string) (*unstructured.Unstructured, error)

	// Mode reports whether this source is live or snapshot-backed.
	Mode() Mode
}

// Mutator is the live-only contract that allows fixers to mutate cluster state.
//
// Snapshot sources do not implement this interface; the type system prevents
// fixers from accidentally running against a captured snapshot. Fixers must
// call AsMutator on their Source argument and set Result.Refused when nil.
type Mutator interface {
	// Delete removes a single resource (background propagation).
	Delete(ctx context.Context, gvr schema.GroupVersionResource, ns, name string) error

	// Patch applies a strategic-merge, JSON-merge, or JSON patch.
	Patch(ctx context.Context, gvr schema.GroupVersionResource, ns, name string, patchType types.PatchType, patch []byte) error

	// Create writes a new resource. Scoped to the report writer; fixers must
	// not create resources — only Delete and Patch are permitted to fixers.
	Create(ctx context.Context, gvr schema.GroupVersionResource, ns string, obj *unstructured.Unstructured) error
}

// AsMutator returns s as a Mutator when the source supports live mutation,
// or nil for snapshot-backed sources. Fixers must call this and refuse
// (set Result.Refused) when the return is nil.
func AsMutator(s Source) Mutator {
	m, _ := s.(Mutator)
	return m
}

// Mode reports whether a Source is backed by a live cluster or a snapshot.
type Mode int

// Mode constants.
const (
	ModeLive     Mode = iota
	ModeSnapshot
)

func (m Mode) String() string {
	switch m {
	case ModeLive:
		return "live"
	case ModeSnapshot:
		return "snapshot"
	default:
		return "unknown"
	}
}
