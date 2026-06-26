// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package snapshot

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

const podListJSON = `{
  "apiVersion": "v1",
  "kind": "PodList",
  "items": [
    {
      "apiVersion": "v1",
      "kind": "Pod",
      "metadata": {"name": "alpha", "namespace": "demo", "labels": {"app": "alpha"}},
      "status": {"phase": "Running"}
    },
    {
      "apiVersion": "v1",
      "kind": "Pod",
      "metadata": {"name": "beta", "namespace": "other", "labels": {"app": "beta"}},
      "status": {"phase": "Pending"}
    }
  ]
}`

const nodeJSON = `{
  "apiVersion": "v1",
  "kind": "Node",
  "metadata": {"name": "node1"},
  "status": {"conditions": [{"type": "Ready", "status": "True"}]}
}`

// writeFile is a tiny helper that fails the test cleanly.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestLoadFile_Directory_ListAndSingleObject(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "pods.json"), podListJSON)
	writeFile(t, filepath.Join(dir, "node.json"), nodeJSON)

	src, err := LoadFile(dir)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if got, want := src.Mode(), ModeSnapshot; got != want {
		t.Errorf("Mode() = %v, want %v", got, want)
	}

	ctx := context.Background()
	pods, err := src.List(ctx, GVRPod, "")
	if err != nil {
		t.Fatalf("List pods: %v", err)
	}
	if got, want := len(pods.Items), 2; got != want {
		t.Errorf("pods all-namespaces: got %d items, want %d", got, want)
	}

	demoPods, err := src.List(ctx, GVRPod, "demo")
	if err != nil {
		t.Fatalf("List pods demo: %v", err)
	}
	if got, want := len(demoPods.Items), 1; got != want {
		t.Errorf("pods ns=demo: got %d items, want %d", got, want)
	}

	nodes, err := src.List(ctx, GVRNode, "")
	if err != nil {
		t.Fatalf("List nodes: %v", err)
	}
	if got, want := len(nodes.Items), 1; got != want {
		t.Errorf("nodes: got %d items, want %d", got, want)
	}
}

func TestLoadFile_SingleFile_List(t *testing.T) {
	f := filepath.Join(t.TempDir(), "pods.json")
	writeFile(t, f, podListJSON)

	src, err := LoadFile(f)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}

	pods, err := src.List(context.Background(), GVRPod, "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got, want := len(pods.Items), 2; got != want {
		t.Fatalf("got %d items, want %d", got, want)
	}
	if name := pods.Items[0].GetName(); name != "alpha" {
		t.Errorf("first pod name = %q, want alpha", name)
	}
}

func TestLoadFile_Get_FoundAndNotFound(t *testing.T) {
	f := filepath.Join(t.TempDir(), "pods.json")
	writeFile(t, f, podListJSON)

	src, err := LoadFile(f)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	ctx := context.Background()

	got, err := src.Get(ctx, GVRPod, "demo", "alpha")
	if err != nil {
		t.Fatalf("Get demo/alpha: %v", err)
	}
	if got.GetName() != "alpha" {
		t.Errorf("Get returned name=%q, want alpha", got.GetName())
	}

	if _, err := src.Get(ctx, GVRPod, "demo", "missing"); err == nil {
		t.Error("Get for missing pod should error, got nil")
	}
}

func TestLoadFile_UnknownPath(t *testing.T) {
	if _, err := LoadFile(filepath.Join(t.TempDir(), "does-not-exist")); err == nil {
		t.Error("LoadFile on missing path should error, got nil")
	}
}
