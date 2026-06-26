// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package snapshot

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCapture_RoundTripDirectory writes a snapshot, then loads the result
// back through LoadFile and verifies the same items come out the other side.
// This is the canonical correctness test: capture → diagnose --snapshot
// must yield the same data the source provided.
func TestCapture_RoundTripDirectory(t *testing.T) {
	// Build a source from in-memory JSON, capture it to a directory,
	// then load that directory and check the round-trip preserves data.
	srcDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(srcDir, "pods.json"), []byte(podListJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "node.json"), []byte(nodeJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	src, err := LoadFile(srcDir)
	if err != nil {
		t.Fatal(err)
	}

	out := t.TempDir()
	summary, err := Capture(context.Background(), src, out)
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	if summary.OutDir != out {
		t.Errorf("OutDir = %q, want %q", summary.OutDir, out)
	}

	// At least the pods + nodes captures should report >0 items.
	gvrItems := map[string]int{}
	for _, c := range summary.Items {
		gvrItems[c.GVR] = c.Items
	}
	if gvrItems["v1/pods"] != 2 {
		t.Errorf("v1/pods captured = %d, want 2 (full summary: %+v)", gvrItems["v1/pods"], summary.Items)
	}
	if gvrItems["v1/nodes"] != 1 {
		t.Errorf("v1/nodes captured = %d, want 1", gvrItems["v1/nodes"])
	}

	// Round-trip: load the captured dir back through LoadFile.
	roundTripped, err := LoadFile(out)
	if err != nil {
		t.Fatalf("LoadFile on captured dir: %v", err)
	}
	pods, err := roundTripped.List(context.Background(), GVRPod, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(pods.Items) != 2 {
		t.Errorf("round-tripped pods = %d, want 2", len(pods.Items))
	}
}

func TestCapture_TarGZ(t *testing.T) {
	srcDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(srcDir, "pods.json"), []byte(podListJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	src, err := LoadFile(srcDir)
	if err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(t.TempDir(), "snapshot.tgz")
	summary, err := CaptureTarGZ(context.Background(), src, out)
	if err != nil {
		t.Fatalf("CaptureTarGZ: %v", err)
	}
	if summary.OutDir != out {
		t.Errorf("OutDir = %q, want %q", summary.OutDir, out)
	}

	// Verify the tarball contains the expected member files and has gzip magic.
	fh, err := os.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = fh.Close() }()
	gz, err := gzip.NewReader(fh)
	if err != nil {
		t.Fatalf("gzip open: %v", err)
	}
	defer func() { _ = gz.Close() }()
	tr := tar.NewReader(gz)
	saw := map[string]bool{}
	for {
		h, err := tr.Next()
		if err != nil {
			break
		}
		saw[h.Name] = true
	}
	wantOneOf := []string{"core-pods.json", "core-nodes.json", "core-events.json"}
	any := false
	for _, n := range wantOneOf {
		if saw[n] {
			any = true
		}
	}
	if !any {
		gotNames := make([]string, 0, len(saw))
		for k := range saw {
			gotNames = append(gotNames, k)
		}
		t.Errorf("tarball missing all expected members; got: %s", strings.Join(gotNames, ", "))
	}
}

func TestCapture_CreatesOutDir(t *testing.T) {
	src, err := LoadFile(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	deeplyNested := filepath.Join(t.TempDir(), "a", "b", "c")
	_, err = Capture(context.Background(), src, deeplyNested)
	if err != nil {
		t.Fatalf("Capture should create nested out dir: %v", err)
	}
	if _, err := os.Stat(deeplyNested); err != nil {
		t.Errorf("out dir not created: %v", err)
	}
}
