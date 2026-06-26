// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package probe

import (
	"context"
	"strings"
	"testing"

	"github.com/srenix-ai/agentic-sre/internal/snapshot"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type memGPUSrc struct {
	nodes []unstructured.Unstructured
}

func (m *memGPUSrc) List(_ context.Context, gvr schema.GroupVersionResource, _ string) (*unstructured.UnstructuredList, error) {
	out := &unstructured.UnstructuredList{}
	if gvr.Resource == "nodes" {
		out.Items = m.nodes
	}
	return out, nil
}
func (m *memGPUSrc) Get(_ context.Context, _ schema.GroupVersionResource, _, _ string) (*unstructured.Unstructured, error) {
	return nil, nil
}
func (m *memGPUSrc) Mode() snapshot.Mode { return snapshot.ModeLive }

func makeGPUNode(name string, ready, unsched bool, gpuKey string, gpuCount string) unstructured.Unstructured {
	u := unstructured.Unstructured{Object: map[string]any{}}
	u.SetAPIVersion("v1")
	u.SetKind("Node")
	u.SetName(name)
	if unsched {
		_ = unstructured.SetNestedField(u.Object, true, "spec", "unschedulable")
	}
	if gpuKey != "" {
		_ = unstructured.SetNestedField(u.Object, gpuCount, "status", "allocatable", gpuKey)
	}
	conds := []any{
		map[string]any{
			"type":   "Ready",
			"status": map[bool]string{true: "True", false: "False"}[ready],
		},
	}
	_ = unstructured.SetNestedSlice(u.Object, conds, "status", "conditions")
	return u
}

func TestGPUNodes_Name(t *testing.T) {
	if (GPUNodes{}).Name() != "GPU Nodes" {
		t.Error("Name mismatch")
	}
}

func TestGPUNodes_NoGPUNodes_OK(t *testing.T) {
	src := &memGPUSrc{nodes: []unstructured.Unstructured{
		makeGPUNode("worker-1", true, false, "", ""), // no GPU resource
	}}
	r := GPUNodes{}.Run(context.Background(), src)
	if r.Component.Status != "OK" {
		t.Errorf("no GPU nodes should be OK; got %q", r.Component.Status)
	}
}

func TestGPUNodes_Healthy(t *testing.T) {
	src := &memGPUSrc{nodes: []unstructured.Unstructured{
		makeGPUNode("gpu-1", true, false, "nvidia.com/gpu", "4"),
	}}
	r := GPUNodes{}.Run(context.Background(), src)
	if r.Component.Status != "OK" || len(r.Findings) != 0 {
		t.Errorf("healthy GPU node should be OK with 0 findings; got status=%q findings=%+v", r.Component.Status, r.Findings)
	}
}

func TestGPUNodes_NotReady_Critical(t *testing.T) {
	src := &memGPUSrc{nodes: []unstructured.Unstructured{
		makeGPUNode("gpu-1", false, false, "nvidia.com/gpu", "4"),
	}}
	r := GPUNodes{}.Run(context.Background(), src)
	if r.Component.Status != "WARNING" {
		t.Errorf("expected WARNING (NotReady); got %q", r.Component.Status)
	}
	if len(r.Findings) == 0 || !strings.Contains(r.Findings[0].Message, "NotReady") {
		t.Errorf("expected NotReady finding; got %+v", r.Findings)
	}
}

func TestGPUNodes_Cordoned_Warning(t *testing.T) {
	src := &memGPUSrc{nodes: []unstructured.Unstructured{
		makeGPUNode("gpu-1", true, true, "nvidia.com/gpu", "4"),
	}}
	r := GPUNodes{}.Run(context.Background(), src)
	if len(r.Findings) != 1 || !strings.Contains(r.Findings[0].Message, "cordoned") {
		t.Errorf("expected cordoned finding; got %+v", r.Findings)
	}
}

func TestGPUNodes_DriverCrash_ZeroAllocatable_Critical(t *testing.T) {
	src := &memGPUSrc{nodes: []unstructured.Unstructured{
		makeGPUNode("gpu-1", true, false, "nvidia.com/gpu", "0"),
	}}
	r := GPUNodes{}.Run(context.Background(), src)
	if len(r.Findings) != 1 {
		t.Fatalf("expected 1 finding (driver crash); got %d", len(r.Findings))
	}
	if !strings.Contains(r.Findings[0].Message, "0 allocatable") {
		t.Errorf("expected driver-crash finding; got %q", r.Findings[0].Message)
	}
}

func TestGPUNodes_AMDGPUDetected(t *testing.T) {
	src := &memGPUSrc{nodes: []unstructured.Unstructured{
		makeGPUNode("amd-1", true, false, "amd.com/gpu", "2"),
	}}
	r := GPUNodes{}.Run(context.Background(), src)
	if r.Component.Status != "OK" || !strings.Contains(r.Component.Detail, "1 GPU node") {
		t.Errorf("AMD GPU should be detected; got status=%q detail=%q", r.Component.Status, r.Component.Detail)
	}
}
