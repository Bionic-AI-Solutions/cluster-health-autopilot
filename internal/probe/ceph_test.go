// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package probe

import (
	"context"
	"strings"
	"testing"
)

const cephOK = `{
  "apiVersion": "ceph.rook.io/v1",
  "kind": "CephCluster",
  "metadata": {"name": "rook-ceph", "namespace": "rook-ceph"},
  "status": {
    "phase": "Ready",
    "ceph": {
      "health": "HEALTH_OK",
      "capacity": {"bytesUsed": 1857311535104, "bytesTotal": 15402851770368}
    }
  }
}`

const cephWarn = `{
  "apiVersion": "ceph.rook.io/v1",
  "kind": "CephCluster",
  "metadata": {"name": "rook-ceph", "namespace": "rook-ceph"},
  "status": {
    "phase": "Ready",
    "ceph": {
      "health": "HEALTH_WARN",
      "capacity": {"bytesUsed": 100, "bytesTotal": 1000}
    }
  }
}`

const cephErr = `{
  "apiVersion": "ceph.rook.io/v1",
  "kind": "CephCluster",
  "metadata": {"name": "rook-ceph", "namespace": "rook-ceph"},
  "status": {
    "phase": "Ready",
    "ceph": {
      "health": "HEALTH_ERR",
      "capacity": {"bytesUsed": 100, "bytesTotal": 1000}
    }
  }
}`

const cephFull = `{
  "apiVersion": "ceph.rook.io/v1",
  "kind": "CephCluster",
  "metadata": {"name": "rook-ceph", "namespace": "rook-ceph"},
  "status": {
    "phase": "Ready",
    "ceph": {
      "health": "HEALTH_OK",
      "capacity": {"bytesUsed": 9100, "bytesTotal": 10000}
    }
  }
}`

const cephNotReady = `{
  "apiVersion": "ceph.rook.io/v1",
  "kind": "CephCluster",
  "metadata": {"name": "rook-ceph", "namespace": "rook-ceph"},
  "status": {
    "phase": "Progressing",
    "ceph": {"health": "HEALTH_OK"}
  }
}`

const cephUnknownHealth = `{
  "apiVersion": "ceph.rook.io/v1",
  "kind": "CephCluster",
  "metadata": {"name": "rook-ceph", "namespace": "rook-ceph"},
  "status": {"phase": "Ready", "ceph": {}}
}`

func TestCeph_HealthOK(t *testing.T) {
	src := loadSrc(t, map[string]string{"ceph.json": cephOK})
	r := Ceph{}.Run(context.Background(), src)
	if r.Component.Status != "HEALTHY" {
		t.Fatalf("status=%q detail=%q", r.Component.Status, r.Component.Detail)
	}
	if len(r.Findings) != 0 {
		t.Errorf("expected 0 findings, got %d", len(r.Findings))
	}
}

func TestCeph_HealthWarn(t *testing.T) {
	src := loadSrc(t, map[string]string{"ceph.json": cephWarn})
	r := Ceph{}.Run(context.Background(), src)
	if r.Component.Status != "DEGRADED" {
		t.Fatalf("status=%q", r.Component.Status)
	}
	if len(r.Findings) != 1 || r.Findings[0].Severity != SeverityWarning {
		t.Errorf("expected 1 warning finding, got %+v", r.Findings)
	}
}

func TestCeph_HealthErr(t *testing.T) {
	src := loadSrc(t, map[string]string{"ceph.json": cephErr})
	r := Ceph{}.Run(context.Background(), src)
	if r.Component.Status != "CRITICAL" {
		t.Fatalf("status=%q", r.Component.Status)
	}
	if len(r.Findings) != 1 || r.Findings[0].Severity != SeverityCritical {
		t.Errorf("expected 1 critical finding, got %+v", r.Findings)
	}
}

func TestCeph_CapacityOver80(t *testing.T) {
	src := loadSrc(t, map[string]string{"ceph.json": cephFull})
	r := Ceph{}.Run(context.Background(), src)
	if r.Component.Status != "DEGRADED" {
		t.Fatalf("expected DEGRADED on capacity warning, got %q", r.Component.Status)
	}
	found := false
	for _, f := range r.Findings {
		if strings.Contains(f.Message, "91.0% full") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected '91.0%% full' finding, got: %+v", r.Findings)
	}
}

func TestCeph_PhaseNotReady(t *testing.T) {
	src := loadSrc(t, map[string]string{"ceph.json": cephNotReady})
	r := Ceph{}.Run(context.Background(), src)
	if r.Component.Status != "CRITICAL" {
		t.Fatalf("status=%q", r.Component.Status)
	}
	if len(r.Findings) != 1 || !strings.Contains(r.Findings[0].Message, "Phase=Progressing") {
		t.Errorf("expected Phase=Progressing finding, got %+v", r.Findings)
	}
}

func TestCeph_UnknownHealth(t *testing.T) {
	src := loadSrc(t, map[string]string{"ceph.json": cephUnknownHealth})
	r := Ceph{}.Run(context.Background(), src)
	if r.Component.Status != "CRITICAL" {
		t.Fatalf("status=%q", r.Component.Status)
	}
}

func TestCeph_NoCRD(t *testing.T) {
	src := loadSrc(t, map[string]string{}) // empty
	r := Ceph{}.Run(context.Background(), src)
	if r.Component.Status != "SKIPPED" {
		t.Fatalf("expected SKIPPED on no CephClusters, got %q", r.Component.Status)
	}
}
