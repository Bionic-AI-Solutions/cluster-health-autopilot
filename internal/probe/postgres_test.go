// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package probe

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/srenix-ai/agentic-sre/internal/snapshot"
)

const cnpgHealthy = `{
  "apiVersion": "postgresql.cnpg.io/v1",
  "kind": "Cluster",
  "metadata": {"name": "pg-ceph", "namespace": "pg"},
  "status": {
    "phase": "Cluster in healthy state",
    "instances": 2,
    "readyInstances": 2,
    "currentPrimary": "pg-ceph-5"
  }
}`

const cnpgUnhealthy = `{
  "apiVersion": "postgresql.cnpg.io/v1",
  "kind": "Cluster",
  "metadata": {"name": "pg-ceph", "namespace": "pg"},
  "status": {
    "phase": "Failover in progress",
    "instances": 2,
    "readyInstances": 1,
    "currentPrimary": "pg-ceph-5"
  }
}`

const spiloMaster = `{
  "apiVersion": "v1",
  "kind": "PodList",
  "items": [
    {
      "apiVersion": "v1",
      "kind": "Pod",
      "metadata": {
        "name": "pg-0",
        "namespace": "pg",
        "labels": {"application": "spilo", "cluster-name": "pg", "spilo-role": "master"}
      },
      "status": {"phase": "Running"}
    },
    {
      "apiVersion": "v1",
      "kind": "Pod",
      "metadata": {
        "name": "pg-1",
        "namespace": "pg",
        "labels": {"application": "spilo", "cluster-name": "pg", "spilo-role": "replica"}
      },
      "status": {"phase": "Running"}
    }
  ]
}`

func loadSrc(t *testing.T, files map[string]string) snapshot.Source {
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

func TestPostgres_CNPGHealthy(t *testing.T) {
	src := loadSrc(t, map[string]string{"cnpg.json": cnpgHealthy})
	r := Postgres{}.Run(context.Background(), src)
	if r.Component.Status != "HEALTHY" {
		t.Fatalf("status = %q, want HEALTHY (detail: %s)", r.Component.Status, r.Component.Detail)
	}
	if len(r.Findings) != 0 {
		t.Errorf("expected no findings, got %d", len(r.Findings))
	}
}

func TestPostgres_CNPGUnhealthy(t *testing.T) {
	src := loadSrc(t, map[string]string{"cnpg.json": cnpgUnhealthy})
	r := Postgres{}.Run(context.Background(), src)
	if r.Component.Status != "CRITICAL" {
		t.Fatalf("status = %q, want CRITICAL (detail: %s)", r.Component.Status, r.Component.Detail)
	}
	if len(r.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(r.Findings))
	}
	if r.Findings[0].Severity != SeverityCritical {
		t.Errorf("severity = %v, want critical", r.Findings[0].Severity)
	}
}

func TestPostgres_SpiloFallback(t *testing.T) {
	src := loadSrc(t, map[string]string{"pods.json": spiloMaster})
	r := Postgres{}.Run(context.Background(), src)
	if r.Component.Status != "HEALTHY" {
		t.Fatalf("status = %q, want HEALTHY (detail: %s)", r.Component.Status, r.Component.Detail)
	}
}

func TestPostgres_NeitherOperator(t *testing.T) {
	src := loadSrc(t, map[string]string{}) // empty snapshot
	r := Postgres{}.Run(context.Background(), src)
	if r.Component.Status != "SKIPPED" {
		t.Fatalf("status = %q, want SKIPPED (detail: %s)", r.Component.Status, r.Component.Detail)
	}
}
