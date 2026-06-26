// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package gcp

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	pkggcp "github.com/srenix-ai/agentic-sre/pkg/cloud/gcp"
	"github.com/srenix-ai/agentic-sre/pkg/probe"
)

func runDisks(disks ...pkggcp.PersistentDisk) probe.Result {
	return PersistentDisks{}.Run(context.Background(), &fakeSource{
		gcp: &fakeGCP{project: "my-project", region: "us-central1", disks: disks},
	})
}

func TestPersistentDisks_SkippedWhenGCPMissing(t *testing.T) {
	got := PersistentDisks{}.Run(context.Background(), &fakeSource{})
	if got.Component.Status != "SKIPPED" {
		t.Errorf("status=%s want SKIPPED", got.Component.Status)
	}
}

func TestPersistentDisks_ProbeFailedOnError(t *testing.T) {
	got := PersistentDisks{}.Run(context.Background(), &fakeSource{
		gcp: &fakeGCP{project: "p", disksErr: errors.New("API down")},
	})
	if got.Component.Status != "PROBE_FAILED" {
		t.Errorf("status=%s want PROBE_FAILED", got.Component.Status)
	}
}

func TestPersistentDisks_AttachedHealthy_Silent(t *testing.T) {
	got := runDisks(pkggcp.PersistentDisk{
		Name:         "vm-disk-1",
		Status:       "READY",
		SizeGB:       100,
		Type:         "pd-ssd",
		Zone:         "us-central1-a",
		AttachedToVM: "instance-1",
	})
	if len(got.Findings) != 0 {
		t.Errorf("attached READY disk should be silent; got: %+v", got.Findings)
	}
}

func TestPersistentDisks_FailedState_Critical(t *testing.T) {
	got := runDisks(pkggcp.PersistentDisk{
		Name:   "vm-disk-1",
		Status: "FAILED",
		SizeGB: 100,
		Type:   "pd-ssd",
		Zone:   "us-central1-a",
	})
	if len(got.Findings) != 1 || got.Findings[0].Severity != probe.SeverityCritical {
		t.Errorf("FAILED should be critical; got: %+v", got.Findings)
	}
	if !strings.Contains(got.Findings[0].Component, "gcp-pd/my-project/us-central1-a/vm-disk-1") {
		t.Errorf("subject=%s lacks gcp-pd/my-project/us-central1-a/vm-disk-1", got.Findings[0].Component)
	}
}

func TestPersistentDisks_DetachedFreshly_Silent(t *testing.T) {
	// Detached but only 1h ago — within the 24h cleanup grace.
	got := runDisks(pkggcp.PersistentDisk{
		Name:             "orphan-1",
		Status:           "READY",
		SizeGB:           50,
		Type:             "pd-standard",
		Zone:             "us-central1-a",
		AttachedToVM:     "", // detached
		DetachedDuration: 1 * time.Hour,
	})
	if len(got.Findings) != 0 {
		t.Errorf("freshly-detached disk should be silent; got: %+v", got.Findings)
	}
}

func TestPersistentDisks_DetachedPastCleanupGrace_Warning(t *testing.T) {
	got := runDisks(pkggcp.PersistentDisk{
		Name:             "orphan-1",
		Status:           "READY",
		SizeGB:           50,
		Type:             "pd-standard",
		Zone:             "us-central1-a",
		AttachedToVM:     "",
		DetachedDuration: 48 * time.Hour, // 2d > 24h grace
	})
	if len(got.Findings) != 1 || got.Findings[0].Severity != probe.SeverityWarning {
		t.Errorf("48h-detached disk should be warning; got: %+v", got.Findings)
	}
	if !strings.Contains(got.Findings[0].Message, "billing leak") {
		t.Errorf("message lacks 'billing leak': %s", got.Findings[0].Message)
	}
}

func TestPersistentDisks_TransitionalWithinGrace_Silent(t *testing.T) {
	// CREATING for only 30 min — within the 1h grace.
	got := runDisks(pkggcp.PersistentDisk{
		Name:      "new-disk",
		Status:    "CREATING",
		SizeGB:    100,
		Zone:      "us-central1-a",
		CreatedAt: time.Now().Add(-30 * time.Minute),
	})
	if len(got.Findings) != 0 {
		t.Errorf("CREATING within grace should be silent; got: %+v", got.Findings)
	}
}

func TestPersistentDisks_TransitionalPastGrace_Warning(t *testing.T) {
	got := runDisks(pkggcp.PersistentDisk{
		Name:      "stuck-disk",
		Status:    "CREATING",
		SizeGB:    100,
		Zone:      "us-central1-a",
		CreatedAt: time.Now().Add(-2 * time.Hour),
	})
	if len(got.Findings) != 1 || got.Findings[0].Severity != probe.SeverityWarning {
		t.Errorf("stuck CREATING should be warning; got: %+v", got.Findings)
	}
	if !strings.Contains(got.Findings[0].Message, "stuck in CREATING") {
		t.Errorf("message lacks 'stuck in CREATING': %s", got.Findings[0].Message)
	}
}

func TestPersistentDisks_UnknownStatus_Warning(t *testing.T) {
	got := runDisks(pkggcp.PersistentDisk{
		Name:   "weird-1",
		Status: "MYSTERY",
		SizeGB: 50,
		Zone:   "us-central1-a",
	})
	if len(got.Findings) != 1 || got.Findings[0].Severity != probe.SeverityWarning {
		t.Errorf("unknown status should be warning; got: %+v", got.Findings)
	}
}

func TestPersistentDisks_RegionalDiskSubject_UsesRegion(t *testing.T) {
	// Regional (HA) disks have Region set instead of Zone.
	got := runDisks(pkggcp.PersistentDisk{
		Name:   "ha-disk-1",
		Status: "FAILED",
		Region: "us-central1",
		SizeGB: 100,
	})
	if !strings.Contains(got.Findings[0].Component, "gcp-pd/my-project/us-central1/ha-disk-1") {
		t.Errorf("regional disk subject=%s should use region (not empty zone)", got.Findings[0].Component)
	}
}

func TestPersistentDisks_NameStable(t *testing.T) {
	if n := (PersistentDisks{}).Name(); n != "gcp-persistent-disks" {
		t.Errorf("Name()=%q want gcp-persistent-disks", n)
	}
}
