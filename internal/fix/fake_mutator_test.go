// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package fix

import (
	"context"
	"fmt"
	"sort"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

// fakeMutator is a no-op cluster Mutator used in unit tests.
//
// It records every Delete / Patch call as a stable string so tests can
// assert exact behaviour without spinning up a real API server.
type fakeMutator struct {
	calls []string

	// returnErr lets tests inject failures on a specific call signature.
	// Map key is the call string ("Delete pods/ns/name"); value is the err
	// to return. Calls without an entry succeed.
	returnErr map[string]error
}

func newFakeMutator() *fakeMutator {
	return &fakeMutator{returnErr: map[string]error{}}
}

func (f *fakeMutator) Delete(_ context.Context, gvr schema.GroupVersionResource, ns, name string) error {
	key := fmt.Sprintf("Delete %s/%s/%s", gvr.Resource, ns, name)
	f.calls = append(f.calls, key)
	return f.returnErr[key]
}

func (f *fakeMutator) Patch(_ context.Context, gvr schema.GroupVersionResource, ns, name string, patchType types.PatchType, _ []byte) error {
	key := fmt.Sprintf("Patch %s/%s/%s [%s]", gvr.Resource, ns, name, string(patchType))
	f.calls = append(f.calls, key)
	return f.returnErr[key]
}

// sortedCalls returns the recorded calls in deterministic order — useful
// for asserting the SET of mutations without depending on map iteration
// order (since pod lists from snapshot are stable but map-driven helpers
// elsewhere may not be).
func (f *fakeMutator) sortedCalls() []string {
	cp := append([]string(nil), f.calls...)
	sort.Strings(cp)
	return cp
}
