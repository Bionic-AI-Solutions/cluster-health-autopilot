// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

// Package azure contains the Azure implementation of the cloud probe
// surface. Each probe is a single Go file (sql_probe.go, disk_probe.go,
// etc.) implementing cloudprobe.Probe; all hold the bound azure.Client
// (via cloud.Source.Azure()) and are read-only.
package azure

import (
	"context"
	"fmt"

	"github.com/srenix-ai/agentic-sre/pkg/cloud"
	pkgazure "github.com/srenix-ai/agentic-sre/pkg/cloud/azure"
	"github.com/srenix-ai/agentic-sre/pkg/probe"
)

// SQLDatabases reports drift on Azure SQL Database resources:
//
//   - Offline / Suspect / EmergencyMode / Inaccessible / Disabled / Failed
//     → critical (database is unreachable)
//   - Paused → critical when not explicitly Serverless tier (Serverless
//     tier auto-pauses by design, so it's expected there; we still
//     warn-only since we can't always tell from the projection)
//   - Restoring / RecoveryPending / Recovery / Scaling / Resuming
//     → warning (transitional; Srenix does not page on routine ops)
//   - UsedPercent >= 80 → warning; >= 90 → critical
//   - BackupConfigured=false → warning (operator-config drift)
type SQLDatabases struct{}

const sqlDatabasesName = "azure-sql"

const (
	storageWarnPercent = 80
	storageCritPercent = 90
)

// Name satisfies cloudprobe.Probe.
func (SQLDatabases) Name() string { return sqlDatabasesName }

// Run satisfies cloudprobe.Probe. Returns SKIPPED when Azure is not
// configured.
func (SQLDatabases) Run(ctx context.Context, src cloud.Source) probe.Result {
	azClient := src.Azure()
	if azClient == nil {
		return probe.Result{
			Component: probe.ComponentResult{
				Component: sqlDatabasesName,
				Status:    "SKIPPED",
				Detail:    "Azure not configured (cloud.azure.enabled=false)",
			},
		}
	}

	dbs, err := azClient.ListSQLDatabases(ctx)
	if err != nil {
		return probe.Result{
			Component: probe.ComponentResult{
				Component: sqlDatabasesName,
				Status:    "PROBE_FAILED",
				Detail:    fmt.Sprintf("ListSQLDatabases: %v", err),
			},
		}
	}

	var findings []probe.Finding
	for _, db := range dbs {
		findings = append(findings, evaluateSQLDatabase(db, azClient.SubscriptionID())...)
	}

	return probe.Result{
		Component: probe.ComponentResult{
			Component: sqlDatabasesName,
			Status:    rollupStatus(findings),
			Detail:    fmt.Sprintf("%d database(s) inspected in subscription %s", len(dbs), azClient.SubscriptionID()),
		},
		Findings: findings,
	}
}

// evaluateSQLDatabase is pure — exhaustively unit-tested.
func evaluateSQLDatabase(db pkgazure.SQLDatabase, subscription string) []probe.Finding {
	subject := fmt.Sprintf("azure-sql/%s/%s/%s/%s",
		subscription, db.ResourceGroup, db.Server, db.Name)
	var out []probe.Finding

	switch db.Status {
	case "Online":
		// healthy
	case "Offline", "Suspect", "EmergencyMode", "Inaccessible", "Disabled":
		out = append(out, probe.Finding{
			Component:   subject,
			Severity:    probe.SeverityCritical,
			Message:     fmt.Sprintf("Azure SQL database %s/%s/%s is in %s state; unreachable", db.ResourceGroup, db.Server, db.Name, db.Status),
			Remediation: fmt.Sprintf("az sql db show -g %s -s %s -n %s --subscription %s — check operations log + portal alerts.", db.ResourceGroup, db.Server, db.Name, subscription),
		})
	case "Paused":
		// Serverless tier auto-pauses; flag as warning so it gets
		// surfaced but doesn't page when expected.
		out = append(out, probe.Finding{
			Component:   subject,
			Severity:    probe.SeverityWarning,
			Message:     fmt.Sprintf("Azure SQL database %s/%s/%s is Paused (expected for Serverless tier; investigate otherwise)", db.ResourceGroup, db.Server, db.Name),
			Remediation: fmt.Sprintf("Confirm tier=%s is Serverless. Otherwise: az sql db resume -g %s -s %s -n %s --subscription %s", db.Tier, db.ResourceGroup, db.Server, db.Name, subscription),
		})
	default:
		// Restoring / Scaling / Creating / etc. — transitional.
		out = append(out, probe.Finding{
			Component: subject,
			Severity:  probe.SeverityWarning,
			Message:   fmt.Sprintf("Azure SQL database %s/%s/%s status=%q (not Online)", db.ResourceGroup, db.Server, db.Name, db.Status),
		})
	}

	// Storage utilization. UsedPercent < 0 means "not measured" (live
	// mode — needs Azure Monitor metrics); skip rather than treat as 0%,
	// which would silently never fire.
	if db.UsedPercent < 0 {
		// no-op: utilization unknown
	} else if db.UsedPercent >= storageCritPercent {
		out = append(out, probe.Finding{
			Component:   subject,
			Severity:    probe.SeverityCritical,
			Message:     fmt.Sprintf("Azure SQL database %s/%s/%s storage at %d%% of %d GB (>= %d%%)", db.ResourceGroup, db.Server, db.Name, db.UsedPercent, db.MaxSizeGB, storageCritPercent),
			Remediation: fmt.Sprintf("az sql db update -g %s -s %s -n %s --max-size <larger> --subscription %s", db.ResourceGroup, db.Server, db.Name, subscription),
		})
	} else if db.UsedPercent >= storageWarnPercent {
		out = append(out, probe.Finding{
			Component: subject,
			Severity:  probe.SeverityWarning,
			Message:   fmt.Sprintf("Azure SQL database %s/%s/%s storage at %d%% (>= %d%%)", db.ResourceGroup, db.Server, db.Name, db.UsedPercent, storageWarnPercent),
		})
	}

	// Backup posture.
	if !db.BackupConfigured {
		out = append(out, probe.Finding{
			Component:   subject,
			Severity:    probe.SeverityWarning,
			Message:     fmt.Sprintf("Azure SQL database %s/%s/%s has no automated backups configured", db.ResourceGroup, db.Server, db.Name),
			Remediation: fmt.Sprintf("az sql db short-term-retention-policy set -g %s -s %s -n %s --retention-days 7 --subscription %s", db.ResourceGroup, db.Server, db.Name, subscription),
		})
	}

	return out
}

// rollupStatus folds per-finding severity into a component-level status.
// Mirrors internal/cloud/aws.rollupStatus + internal/cloud/gcp.rollupStatus.
func rollupStatus(findings []probe.Finding) string {
	status := "HEALTHY"
	for _, f := range findings {
		if f.Severity == probe.SeverityCritical {
			return "CRITICAL"
		}
		if f.Severity == probe.SeverityWarning {
			status = "DEGRADED"
		}
	}
	return status
}
