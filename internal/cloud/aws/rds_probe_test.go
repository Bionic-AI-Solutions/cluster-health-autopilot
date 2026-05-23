// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/cloud"
	pkgaws "github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/cloud/aws"
	pkggcp "github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/cloud/gcp"
	pkgazure "github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/cloud/azure"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/probe"
)

// fakeAWS implements pkgaws.Client for unit tests. Tests inject the
// instance list (and optional error) directly.
type fakeAWS struct {
	region    string
	instances []pkgaws.DBInstance
	err       error
}

func (f *fakeAWS) Region() string { return f.region }
func (f *fakeAWS) DescribeDBInstances(_ context.Context) ([]pkgaws.DBInstance, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.instances, nil
}

// fakeSource implements cloud.Source for unit tests. AWS is settable;
// other providers default to nil (matches the cloud.aws-only test case).
type fakeSource struct {
	aws   pkgaws.Client
	gcp   pkggcp.Client
	azure pkgazure.Client
	mode  cloud.Mode
}

func (f *fakeSource) AWS() pkgaws.Client     { return f.aws }
func (f *fakeSource) GCP() pkggcp.Client     { return f.gcp }
func (f *fakeSource) Azure() pkgazure.Client { return f.azure }
func (f *fakeSource) Mode() cloud.Mode       { return f.mode }

func TestRDS_Name(t *testing.T) {
	if got := (RDS{}).Name(); got != "aws-rds" {
		t.Errorf("Name()=%q want aws-rds", got)
	}
}

func TestRDS_SkippedWhenAWSNotConfigured(t *testing.T) {
	src := &fakeSource{} // aws=nil
	r := (RDS{}).Run(context.Background(), src)
	if r.Component.Status != "SKIPPED" {
		t.Errorf("Status=%q want SKIPPED when AWS sub-client is nil", r.Component.Status)
	}
	if len(r.Findings) != 0 {
		t.Errorf("expected no findings when skipped, got %d", len(r.Findings))
	}
}

func TestRDS_ProbeFailedOnAPIError(t *testing.T) {
	src := &fakeSource{aws: &fakeAWS{region: "us-east-1", err: errors.New("boom")}}
	r := (RDS{}).Run(context.Background(), src)
	if r.Component.Status != "PROBE_FAILED" {
		t.Errorf("Status=%q want PROBE_FAILED on API error", r.Component.Status)
	}
	if !strings.Contains(r.Component.Detail, "boom") {
		t.Errorf("Detail must surface the API error, got %q", r.Component.Detail)
	}
}

func TestRDS_HealthyOnAvailableLowStorage(t *testing.T) {
	src := &fakeSource{aws: &fakeAWS{
		region: "us-east-1",
		instances: []pkgaws.DBInstance{
			{Identifier: "happy-db", Engine: "postgres", Status: "available", StorageUsedPercent: 42, AllocatedStorageGB: 100},
		},
	}}
	r := (RDS{}).Run(context.Background(), src)
	if r.Component.Status != "HEALTHY" {
		t.Errorf("Status=%q want HEALTHY for available + low storage", r.Component.Status)
	}
	if len(r.Findings) != 0 {
		t.Errorf("expected no findings, got %d: %+v", len(r.Findings), r.Findings)
	}
}

func TestRDS_StorageWarningTriggersDegraded(t *testing.T) {
	src := &fakeSource{aws: &fakeAWS{
		region: "us-east-1",
		instances: []pkgaws.DBInstance{
			{Identifier: "warm-db", Engine: "postgres", Status: "available", StorageUsedPercent: 82, AllocatedStorageGB: 100},
		},
	}}
	r := (RDS{}).Run(context.Background(), src)
	if r.Component.Status != "DEGRADED" {
		t.Errorf("Status=%q want DEGRADED for storage 82%%", r.Component.Status)
	}
	if len(r.Findings) != 1 || r.Findings[0].Severity != probe.SeverityWarning {
		t.Fatalf("expected 1 warning finding, got %+v", r.Findings)
	}
	if r.Findings[0].Component != "aws-rds/us-east-1/warm-db" {
		t.Errorf("Component=%q want aws-rds/us-east-1/warm-db", r.Findings[0].Component)
	}
}

func TestRDS_StorageCriticalTriggersCritical(t *testing.T) {
	src := &fakeSource{aws: &fakeAWS{
		region: "us-east-1",
		instances: []pkgaws.DBInstance{
			{Identifier: "hot-db", Engine: "postgres", Status: "available", StorageUsedPercent: 95, AllocatedStorageGB: 100},
		},
	}}
	r := (RDS{}).Run(context.Background(), src)
	if r.Component.Status != "CRITICAL" {
		t.Errorf("Status=%q want CRITICAL for storage 95%%", r.Component.Status)
	}
	if len(r.Findings) != 1 || r.Findings[0].Severity != probe.SeverityCritical {
		t.Fatalf("expected 1 critical finding, got %+v", r.Findings)
	}
	if !strings.Contains(r.Findings[0].Remediation, "modify-db-instance") {
		t.Errorf("Remediation should suggest aws rds modify-db-instance, got %q", r.Findings[0].Remediation)
	}
}

func TestRDS_StorageFullStatus(t *testing.T) {
	src := &fakeSource{aws: &fakeAWS{
		region: "us-east-1",
		instances: []pkgaws.DBInstance{
			{Identifier: "full-db", Engine: "postgres", Status: "storage-full", StorageUsedPercent: 100, AllocatedStorageGB: 100},
		},
	}}
	r := (RDS{}).Run(context.Background(), src)
	if r.Component.Status != "CRITICAL" {
		t.Errorf("Status=%q want CRITICAL for storage-full", r.Component.Status)
	}
	// Expect two critical findings: one for status, one for storage threshold.
	if len(r.Findings) != 2 {
		t.Fatalf("expected 2 findings (status + storage), got %d: %+v", len(r.Findings), r.Findings)
	}
}

func TestRDS_FailureStatesCritical(t *testing.T) {
	for _, st := range []string{"failed", "incompatible-network", "incompatible-restore", "restore-error"} {
		t.Run(st, func(t *testing.T) {
			src := &fakeSource{aws: &fakeAWS{
				region:    "us-east-1",
				instances: []pkgaws.DBInstance{{Identifier: "x", Engine: "postgres", Status: st, AllocatedStorageGB: 100, StorageUsedPercent: 10}},
			}}
			r := (RDS{}).Run(context.Background(), src)
			if r.Component.Status != "CRITICAL" {
				t.Errorf("Status=%q want CRITICAL for %q", r.Component.Status, st)
			}
		})
	}
}

func TestRDS_TransitionalStateWarn(t *testing.T) {
	src := &fakeSource{aws: &fakeAWS{
		region:    "us-east-1",
		instances: []pkgaws.DBInstance{{Identifier: "modifying-db", Engine: "postgres", Status: "modifying", AllocatedStorageGB: 100, StorageUsedPercent: 10}},
	}}
	r := (RDS{}).Run(context.Background(), src)
	if r.Component.Status != "DEGRADED" {
		t.Errorf("Status=%q want DEGRADED for transitional 'modifying'", r.Component.Status)
	}
}

func TestRDS_MixedInstancesRollupCritical(t *testing.T) {
	src := &fakeSource{aws: &fakeAWS{
		region: "us-east-1",
		instances: []pkgaws.DBInstance{
			{Identifier: "ok-db", Engine: "postgres", Status: "available", StorageUsedPercent: 30, AllocatedStorageGB: 100},
			{Identifier: "warn-db", Engine: "postgres", Status: "available", StorageUsedPercent: 82, AllocatedStorageGB: 100},
			{Identifier: "fail-db", Engine: "postgres", Status: "failed", AllocatedStorageGB: 100, StorageUsedPercent: 10},
		},
	}}
	r := (RDS{}).Run(context.Background(), src)
	if r.Component.Status != "CRITICAL" {
		t.Errorf("Status=%q want CRITICAL (worst severity wins)", r.Component.Status)
	}
}
