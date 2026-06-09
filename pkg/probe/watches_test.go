// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package probe_test

import (
	"context"
	"testing"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/probe"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/snapshot"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// nonWatching is a Probe that does NOT implement GVRWatcher.
type nonWatching struct{}

func (nonWatching) Name() string                            { return "nw" }
func (nonWatching) Run(context.Context, snapshot.Source) probe.Result { return probe.Result{} }

// watching is a Probe that DOES implement GVRWatcher.
type watching struct{}

func (watching) Name() string                            { return "w" }
func (watching) Run(context.Context, snapshot.Source) probe.Result { return probe.Result{} }
func (watching) GVRs() []schema.GroupVersionResource {
	return []schema.GroupVersionResource{
		{Group: "", Version: "v1", Resource: "pods"},
		{Group: "apps", Version: "v1", Resource: "deployments"},
	}
}

func TestGVRsOf_NonWatching_Nil(t *testing.T) {
	if got := probe.GVRsOf(nonWatching{}); got != nil {
		t.Errorf("non-watching probe must yield nil; got %+v", got)
	}
}

func TestGVRsOf_Watching_ReturnsList(t *testing.T) {
	got := probe.GVRsOf(watching{})
	if len(got) != 2 {
		t.Fatalf("expected 2 GVRs; got %d", len(got))
	}
	if got[0].Resource != "pods" || got[1].Resource != "deployments" {
		t.Errorf("got %+v", got)
	}
}
