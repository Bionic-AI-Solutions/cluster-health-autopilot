// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package gcp

import (
	"context"
	"errors"
	"strings"
	"testing"

	pkggcp "github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/cloud/gcp"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/probe"
)

func runCloudSQL(instances ...pkggcp.CloudSQLInstance) probe.Result {
	return CloudSQL{}.Run(context.Background(), &fakeSource{
		gcp: &fakeGCP{
			project:   "my-project",
			region:    "us-central1",
			instances: instances,
		},
	})
}

func TestCloudSQL_SkippedWhenGCPMissing(t *testing.T) {
	got := CloudSQL{}.Run(context.Background(), &fakeSource{})
	if got.Component.Status != "SKIPPED" {
		t.Errorf("status=%s want SKIPPED", got.Component.Status)
	}
	if !strings.Contains(got.Component.Detail, "GCP not configured") {
		t.Errorf("detail lacks 'GCP not configured': %s", got.Component.Detail)
	}
}

func TestCloudSQL_ProbeFailedOnClientError(t *testing.T) {
	got := CloudSQL{}.Run(context.Background(), &fakeSource{
		gcp: &fakeGCP{project: "p", instancesErr: errors.New("API quota exceeded")},
	})
	if got.Component.Status != "PROBE_FAILED" {
		t.Errorf("status=%s want PROBE_FAILED", got.Component.Status)
	}
}

func TestCloudSQL_HealthyInstance_NoFindings(t *testing.T) {
	got := runCloudSQL(pkggcp.CloudSQLInstance{
		Name:             "prod-1",
		State:            "RUNNABLE",
		DatabaseVersion:  "POSTGRES_15",
		DiskSizeGB:       100,
		DiskUsedPercent:  40,
		BackupConfigured: true,
	})
	if len(got.Findings) != 0 {
		t.Errorf("healthy instance should be silent; got: %+v", got.Findings)
	}
	if got.Component.Status != "HEALTHY" {
		t.Errorf("rollup=%s want HEALTHY", got.Component.Status)
	}
}

func TestCloudSQL_FailedState_Critical(t *testing.T) {
	got := runCloudSQL(pkggcp.CloudSQLInstance{
		Name:             "prod-1",
		State:            "FAILED",
		DatabaseVersion:  "POSTGRES_15",
		BackupConfigured: true,
	})
	if len(got.Findings) != 1 {
		t.Fatalf("expected 1 finding; got %d: %+v", len(got.Findings), got.Findings)
	}
	if got.Findings[0].Severity != probe.SeverityCritical {
		t.Errorf("severity=%s want critical", got.Findings[0].Severity)
	}
	if !strings.Contains(got.Findings[0].Component, "gcp-cloudsql/my-project/prod-1") {
		t.Errorf("subject=%s lacks gcp-cloudsql/my-project/prod-1", got.Findings[0].Component)
	}
}

func TestCloudSQL_Suspended_Critical(t *testing.T) {
	got := runCloudSQL(pkggcp.CloudSQLInstance{
		Name:             "prod-1",
		State:            "SUSPENDED",
		BackupConfigured: true,
	})
	if len(got.Findings) != 1 || got.Findings[0].Severity != probe.SeverityCritical {
		t.Errorf("expected 1 critical finding; got: %+v", got.Findings)
	}
	if !strings.Contains(got.Findings[0].Message, "SUSPENDED") {
		t.Errorf("message lacks 'SUSPENDED': %s", got.Findings[0].Message)
	}
}

func TestCloudSQL_TransitionalState_Warning(t *testing.T) {
	got := runCloudSQL(pkggcp.CloudSQLInstance{
		Name:             "prod-1",
		State:            "MAINTENANCE",
		BackupConfigured: true,
	})
	if len(got.Findings) != 1 || got.Findings[0].Severity != probe.SeverityWarning {
		t.Errorf("MAINTENANCE should be warning; got: %+v", got.Findings)
	}
}

func TestCloudSQL_DiskWarn_AtThreshold(t *testing.T) {
	got := runCloudSQL(pkggcp.CloudSQLInstance{
		Name:             "prod-1",
		State:            "RUNNABLE",
		DiskSizeGB:       100,
		DiskUsedPercent:  80,
		BackupConfigured: true,
	})
	if len(got.Findings) != 1 || got.Findings[0].Severity != probe.SeverityWarning {
		t.Errorf("80%% disk should be warning; got: %+v", got.Findings)
	}
}

func TestCloudSQL_DiskCritical_AtThreshold(t *testing.T) {
	got := runCloudSQL(pkggcp.CloudSQLInstance{
		Name:             "prod-1",
		State:            "RUNNABLE",
		DiskSizeGB:       100,
		DiskUsedPercent:  90,
		BackupConfigured: true,
	})
	if len(got.Findings) != 1 || got.Findings[0].Severity != probe.SeverityCritical {
		t.Errorf("90%% disk should be critical; got: %+v", got.Findings)
	}
}

func TestCloudSQL_DiskCritical_SuppressedWhenAutoResize(t *testing.T) {
	got := runCloudSQL(pkggcp.CloudSQLInstance{
		Name:              "prod-1",
		State:             "RUNNABLE",
		DiskSizeGB:        100,
		DiskUsedPercent:   95,
		StorageAutoResize: true, // GCP grows automatically
		BackupConfigured:  true,
	})
	if len(got.Findings) != 0 {
		t.Errorf("auto-resize instance should suppress disk finding; got: %+v", got.Findings)
	}
}

func TestCloudSQL_NoBackup_Warning(t *testing.T) {
	got := runCloudSQL(pkggcp.CloudSQLInstance{
		Name:             "prod-1",
		State:            "RUNNABLE",
		DiskUsedPercent:  40,
		BackupConfigured: false,
	})
	if len(got.Findings) != 1 || got.Findings[0].Severity != probe.SeverityWarning {
		t.Errorf("no-backup should be warning; got: %+v", got.Findings)
	}
	if !strings.Contains(got.Findings[0].Message, "automated backups") {
		t.Errorf("message lacks 'automated backups': %s", got.Findings[0].Message)
	}
}

func TestCloudSQL_FailedAndNoBackup_TwoFindings(t *testing.T) {
	got := runCloudSQL(pkggcp.CloudSQLInstance{
		Name:             "prod-1",
		State:            "FAILED",
		BackupConfigured: false,
	})
	if len(got.Findings) != 2 {
		t.Fatalf("expected 2 findings (state + backup); got %d: %+v", len(got.Findings), got.Findings)
	}
	if got.Component.Status != "CRITICAL" {
		t.Errorf("rollup=%s want CRITICAL", got.Component.Status)
	}
}

func TestCloudSQL_NameStable(t *testing.T) {
	if n := (CloudSQL{}).Name(); n != "gcp-cloudsql" {
		t.Errorf("Name()=%q want gcp-cloudsql", n)
	}
}
