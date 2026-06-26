// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package probe

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/srenix-ai/agentic-sre/internal/snapshot"
)

// loadProbeSrc spins up a minimal file-based snapshot.Source for probe tests.
// Re-used by the Sprint 2 probes (NodePressure, DaemonSets, PendingPods,
// CrashLoopBackOff, ETCD, FailedMounts).
func loadProbeSrc(t *testing.T, files map[string]string) snapshot.Source {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	src, err := snapshot.LoadFile(dir)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	return src
}

const nodesAllHealthy = `{
  "apiVersion": "v1", "kind": "NodeList",
  "items": [
    {"apiVersion": "v1", "kind": "Node",
     "metadata": {"name": "node-a"},
     "status": {"conditions": [
       {"type": "Ready", "status": "True"},
       {"type": "DiskPressure", "status": "False"},
       {"type": "MemoryPressure", "status": "False"},
       {"type": "PIDPressure", "status": "False"}
     ]}},
    {"apiVersion": "v1", "kind": "Node",
     "metadata": {"name": "node-b"},
     "status": {"conditions": [
       {"type": "Ready", "status": "True"},
       {"type": "DiskPressure", "status": "False"},
       {"type": "MemoryPressure", "status": "False"}
     ]}}
  ]
}`

const nodesWithDiskPressure = `{
  "apiVersion": "v1", "kind": "NodeList",
  "items": [
    {"apiVersion": "v1", "kind": "Node",
     "metadata": {"name": "node-a"},
     "status": {"conditions": [
       {"type": "Ready", "status": "True"},
       {"type": "DiskPressure", "status": "True"}
     ]}},
    {"apiVersion": "v1", "kind": "Node",
     "metadata": {"name": "node-b"},
     "status": {"conditions": [{"type": "Ready", "status": "True"}]}}
  ]
}`

const nodesWithMemPressureOnly = `{
  "apiVersion": "v1", "kind": "NodeList",
  "items": [
    {"apiVersion": "v1", "kind": "Node",
     "metadata": {"name": "node-a"},
     "status": {"conditions": [
       {"type": "Ready", "status": "True"},
       {"type": "MemoryPressure", "status": "True"}
     ]}}
  ]
}`

const nodesWithNetworkUnavailable = `{
  "apiVersion": "v1", "kind": "NodeList",
  "items": [
    {"apiVersion": "v1", "kind": "Node",
     "metadata": {"name": "node-a"},
     "status": {"conditions": [
       {"type": "Ready", "status": "True"},
       {"type": "NetworkUnavailable", "status": "True"}
     ]}}
  ]
}`

const nodesWithMultiplePressures = `{
  "apiVersion": "v1", "kind": "NodeList",
  "items": [
    {"apiVersion": "v1", "kind": "Node",
     "metadata": {"name": "gpu-01"},
     "status": {"conditions": [
       {"type": "DiskPressure", "status": "True"},
       {"type": "PIDPressure", "status": "True"}
     ]}}
  ]
}`

func TestNodePressure_AllHealthy(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{"nodes.json": nodesAllHealthy})
	r := NodePressure{}.Run(context.Background(), src)
	if r.Component.Status != "HEALTHY" {
		t.Errorf("Status = %q, want HEALTHY (detail=%q)", r.Component.Status, r.Component.Detail)
	}
	if len(r.Findings) != 0 {
		t.Errorf("expected no findings, got %+v", r.Findings)
	}
}

func TestNodePressure_DiskPressure_Critical(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{"nodes.json": nodesWithDiskPressure})
	r := NodePressure{}.Run(context.Background(), src)
	if r.Component.Status != "CRITICAL" {
		t.Errorf("DiskPressure must be CRITICAL, got %q", r.Component.Status)
	}
	if !strings.Contains(r.Component.Detail, "DiskPressure") || !strings.Contains(r.Component.Detail, "node-a") {
		t.Errorf("detail missing DiskPressure/node-a: %q", r.Component.Detail)
	}
	if len(r.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(r.Findings))
	}
	if r.Findings[0].Severity != SeverityCritical {
		t.Errorf("finding severity = %v, want Critical", r.Findings[0].Severity)
	}
}

func TestNodePressure_MemoryPressure_Warning(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{"nodes.json": nodesWithMemPressureOnly})
	r := NodePressure{}.Run(context.Background(), src)
	if r.Component.Status != "WARNING" {
		t.Errorf("MemoryPressure-only should be WARNING, got %q (detail=%q)",
			r.Component.Status, r.Component.Detail)
	}
	if len(r.Findings) != 1 || r.Findings[0].Severity != SeverityWarning {
		t.Errorf("expected 1 Warning finding, got %+v", r.Findings)
	}
}

func TestNodePressure_NetworkUnavailable_Critical(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{"nodes.json": nodesWithNetworkUnavailable})
	r := NodePressure{}.Run(context.Background(), src)
	if r.Component.Status != "CRITICAL" {
		t.Errorf("NetworkUnavailable should be CRITICAL, got %q", r.Component.Status)
	}
}

func TestNodePressure_MultiplePressuresGrouped(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{"nodes.json": nodesWithMultiplePressures})
	r := NodePressure{}.Run(context.Background(), src)
	if r.Component.Status != "CRITICAL" {
		t.Errorf("DiskPressure+PIDPressure should be CRITICAL, got %q", r.Component.Status)
	}
	// Two findings, one per condition class.
	if len(r.Findings) != 2 {
		t.Errorf("expected 2 findings (one per condition class), got %d", len(r.Findings))
	}
	// Detail should list both condition types in stable (sorted) order.
	if !strings.Contains(r.Component.Detail, "DiskPressure") ||
		!strings.Contains(r.Component.Detail, "PIDPressure") {
		t.Errorf("detail missing condition types: %q", r.Component.Detail)
	}
}

func TestNodePressure_NoNodes_ProbeFailed(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{})
	r := NodePressure{}.Run(context.Background(), src)
	if r.Component.Status != "PROBE_FAILED" {
		t.Errorf("empty node list should be PROBE_FAILED, got %q", r.Component.Status)
	}
}
