// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"
	"errors"
	"strings"
	"testing"

	pkgazure "github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/cloud/azure"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/probe"
)

func runSQL(dbs ...pkgazure.SQLDatabase) probe.Result {
	return SQLDatabases{}.Run(context.Background(), &fakeSource{
		azure: &fakeAzure{
			subscription: "sub-1",
			location:     "eastus",
			dbs:          dbs,
		},
	})
}

func TestSQLDatabases_SkippedWhenAzureMissing(t *testing.T) {
	got := SQLDatabases{}.Run(context.Background(), &fakeSource{})
	if got.Component.Status != "SKIPPED" {
		t.Errorf("status=%s want SKIPPED", got.Component.Status)
	}
}

func TestSQLDatabases_ProbeFailedOnError(t *testing.T) {
	got := SQLDatabases{}.Run(context.Background(), &fakeSource{
		azure: &fakeAzure{subscription: "sub-1", dbsErr: errors.New("auth failed")},
	})
	if got.Component.Status != "PROBE_FAILED" {
		t.Errorf("status=%s want PROBE_FAILED", got.Component.Status)
	}
}

func TestSQLDatabases_OnlineHealthy_Silent(t *testing.T) {
	got := runSQL(pkgazure.SQLDatabase{
		Name:             "prod-1",
		ResourceGroup:    "rg-prod",
		Server:           "sql-prod",
		Status:           "Online",
		MaxSizeGB:        100,
		UsedPercent:      30,
		BackupConfigured: true,
	})
	if len(got.Findings) != 0 {
		t.Errorf("healthy DB should be silent; got: %+v", got.Findings)
	}
}

func TestSQLDatabases_Offline_Critical(t *testing.T) {
	got := runSQL(pkgazure.SQLDatabase{
		Name:             "prod-1",
		ResourceGroup:    "rg-prod",
		Server:           "sql-prod",
		Status:           "Offline",
		BackupConfigured: true,
	})
	if len(got.Findings) != 1 || got.Findings[0].Severity != probe.SeverityCritical {
		t.Errorf("Offline should be critical; got: %+v", got.Findings)
	}
	if !strings.Contains(got.Findings[0].Component, "azure-sql/sub-1/rg-prod/sql-prod/prod-1") {
		t.Errorf("subject=%s lacks azure-sql/...", got.Findings[0].Component)
	}
}

func TestSQLDatabases_Suspect_Critical(t *testing.T) {
	got := runSQL(pkgazure.SQLDatabase{
		Name: "prod-1", ResourceGroup: "rg", Server: "sql",
		Status: "Suspect", BackupConfigured: true,
	})
	if len(got.Findings) != 1 || got.Findings[0].Severity != probe.SeverityCritical {
		t.Errorf("Suspect should be critical; got: %+v", got.Findings)
	}
}

func TestSQLDatabases_Paused_Warning(t *testing.T) {
	got := runSQL(pkgazure.SQLDatabase{
		Name: "serverless-1", ResourceGroup: "rg", Server: "sql",
		Status: "Paused", Tier: "ServerlessV2", BackupConfigured: true,
	})
	if len(got.Findings) != 1 || got.Findings[0].Severity != probe.SeverityWarning {
		t.Errorf("Paused should be warning; got: %+v", got.Findings)
	}
}

func TestSQLDatabases_TransitionalState_Warning(t *testing.T) {
	got := runSQL(pkgazure.SQLDatabase{
		Name: "prod-1", ResourceGroup: "rg", Server: "sql",
		Status: "Scaling", BackupConfigured: true,
	})
	if len(got.Findings) != 1 || got.Findings[0].Severity != probe.SeverityWarning {
		t.Errorf("Scaling should be warning; got: %+v", got.Findings)
	}
}

func TestSQLDatabases_StorageWarn_AtThreshold(t *testing.T) {
	got := runSQL(pkgazure.SQLDatabase{
		Name: "prod-1", ResourceGroup: "rg", Server: "sql",
		Status: "Online", MaxSizeGB: 100, UsedPercent: 80,
		BackupConfigured: true,
	})
	if len(got.Findings) != 1 || got.Findings[0].Severity != probe.SeverityWarning {
		t.Errorf("80%% should be warning; got: %+v", got.Findings)
	}
}

func TestSQLDatabases_StorageCritical_AtThreshold(t *testing.T) {
	got := runSQL(pkgazure.SQLDatabase{
		Name: "prod-1", ResourceGroup: "rg", Server: "sql",
		Status: "Online", MaxSizeGB: 100, UsedPercent: 90,
		BackupConfigured: true,
	})
	if len(got.Findings) != 1 || got.Findings[0].Severity != probe.SeverityCritical {
		t.Errorf("90%% should be critical; got: %+v", got.Findings)
	}
}

func TestSQLDatabases_NoBackup_Warning(t *testing.T) {
	got := runSQL(pkgazure.SQLDatabase{
		Name: "prod-1", ResourceGroup: "rg", Server: "sql",
		Status: "Online", BackupConfigured: false,
	})
	if len(got.Findings) != 1 || got.Findings[0].Severity != probe.SeverityWarning {
		t.Errorf("no-backup should be warning; got: %+v", got.Findings)
	}
}

func TestSQLDatabases_OfflineAndNoBackup_TwoFindings(t *testing.T) {
	got := runSQL(pkgazure.SQLDatabase{
		Name: "prod-1", ResourceGroup: "rg", Server: "sql",
		Status: "Offline", BackupConfigured: false,
	})
	if len(got.Findings) != 2 {
		t.Fatalf("expected 2 findings (status + backup); got %d", len(got.Findings))
	}
	if got.Component.Status != "CRITICAL" {
		t.Errorf("rollup=%s want CRITICAL", got.Component.Status)
	}
}

func TestSQLDatabases_NameStable(t *testing.T) {
	if n := (SQLDatabases{}).Name(); n != "azure-sql" {
		t.Errorf("Name()=%q want azure-sql", n)
	}
}
