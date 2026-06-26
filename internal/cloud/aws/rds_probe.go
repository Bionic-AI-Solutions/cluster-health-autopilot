// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

// Package aws contains the AWS implementation of the cloud probe
// surface. Each probe is a single Go file (rds_probe.go, ebs_probe.go,
// etc.) implementing cloudprobe.Probe; all hold the bound
// aws.Client (via cloud.Source.AWS()) and are read-only.
package aws

import (
	"context"
	"fmt"

	"github.com/srenix-ai/agentic-sre/pkg/cloud"
	pkgaws "github.com/srenix-ai/agentic-sre/pkg/cloud/aws"
	"github.com/srenix-ai/agentic-sre/pkg/probe"
)

// RDS reports drift on AWS RDS DBInstances:
//   - any instance whose status is not "available" → warning or critical
//     depending on which non-available state
//   - any instance with StorageUsedPercent >= 80% → warning
//   - any instance with StorageUsedPercent >= 90% → critical
//   - storage-full instances (status == "storage-full") → critical regardless
//
// Multi-AZ instances are not severity-modified — operators care equally
// about storage filling on a Multi-AZ primary.
type RDS struct{}

const rdsName = "aws-rds"

// Storage-utilization thresholds. Tuned for v1; can be made
// configurable later if operators ask.
const (
	storageWarnPercent = 80
	storageCritPercent = 90
)

// Name satisfies cloudprobe.Probe and is rendered in Slack /
// DriftReport / Alertmanager as the component label.
func (RDS) Name() string { return rdsName }

// Run satisfies cloudprobe.Probe.
//
// Run inspects all RDS DBInstances visible to the bound AWS sub-client.
// Returns SKIPPED when the source has no AWS configured.
func (RDS) Run(ctx context.Context, src cloud.Source) probe.Result {
	awsClient := src.AWS()
	if awsClient == nil {
		return probe.Result{
			Component: probe.ComponentResult{
				Component: rdsName,
				Status:    "SKIPPED",
				Detail:    "AWS not configured (cloud.aws.enabled=false)",
			},
		}
	}

	instances, err := awsClient.DescribeDBInstances(ctx)
	if err != nil {
		return probe.Result{
			Component: probe.ComponentResult{
				Component: rdsName,
				Status:    "PROBE_FAILED",
				Detail:    fmt.Sprintf("DescribeDBInstances: %v", err),
			},
		}
	}

	var findings []probe.Finding
	for _, db := range instances {
		findings = append(findings, evaluateInstance(db, awsClient.Region())...)
	}

	return probe.Result{
		Component: probe.ComponentResult{
			Component: rdsName,
			Status:    rollupStatus(findings),
			Detail:    fmt.Sprintf("%d instance(s) inspected in %s", len(instances), awsClient.Region()),
		},
		Findings: findings,
	}
}

// rollupStatus folds per-finding severity into a single component-level
// status string matching the convention used by internal/probe (HEALTHY,
// DEGRADED, CRITICAL).
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

// backupRetentionWarnDays is the minimum recommended backup retention period.
const backupRetentionWarnDays = 7

// evaluateInstance is pure — easy to unit-test exhaustively. Returns
// zero or more findings per instance.
func evaluateInstance(db pkgaws.DBInstance, region string) []probe.Finding {
	subject := fmt.Sprintf("aws-rds/%s/%s", region, db.Identifier)
	var out []probe.Finding

	// Status check
	switch db.Status {
	case "available":
		// healthy; no finding
	case "storage-full":
		out = append(out, probe.Finding{
			Component:   subject,
			Severity:    probe.SeverityCritical,
			Message:     fmt.Sprintf("RDS instance %q (%s) is storage-full — writes will fail until storage is increased", db.Identifier, db.Engine),
			Remediation: fmt.Sprintf("aws rds modify-db-instance --db-instance-identifier %s --allocated-storage <larger> --apply-immediately", db.Identifier),
		})
	case "failed", "incompatible-network", "incompatible-option-group",
		"incompatible-parameters", "incompatible-restore", "restore-error":
		out = append(out, probe.Finding{
			Component:   subject,
			Severity:    probe.SeverityCritical,
			Message:     fmt.Sprintf("RDS instance %q (%s) is in failure state %q", db.Identifier, db.Engine, db.Status),
			Remediation: fmt.Sprintf("aws rds describe-events --source-identifier %s --source-type db-instance --duration 60", db.Identifier),
		})
	default:
		// transitional or unknown — warn but don't critical (we don't
		// know if it's about to recover, so don't page on every
		// "modifying" or "backing-up")
		out = append(out, probe.Finding{
			Component: subject,
			Severity:  probe.SeverityWarning,
			Message:   fmt.Sprintf("RDS instance %q (%s) status is %q (not 'available')", db.Identifier, db.Engine, db.Status),
		})
	}

	// Storage utilization check — emitted independently of status. A
	// storage-full instance gets both the status critical above AND
	// a storage critical (which is fine; same subject, same severity,
	// dedup handles it downstream).
	if db.StorageUsedPercent >= storageCritPercent {
		out = append(out, probe.Finding{
			Component:   subject,
			Severity:    probe.SeverityCritical,
			Message:     fmt.Sprintf("RDS instance %q storage at %d%% of %d GB (>= %d%%)", db.Identifier, db.StorageUsedPercent, db.AllocatedStorageGB, storageCritPercent),
			Remediation: fmt.Sprintf("aws rds modify-db-instance --db-instance-identifier %s --allocated-storage <larger> --apply-immediately", db.Identifier),
		})
	} else if db.StorageUsedPercent >= storageWarnPercent {
		out = append(out, probe.Finding{
			Component: subject,
			Severity:  probe.SeverityWarning,
			Message:   fmt.Sprintf("RDS instance %q storage at %d%% of %d GB (>= %d%%)", db.Identifier, db.StorageUsedPercent, db.AllocatedStorageGB, storageWarnPercent),
		})
	}

	// Multi-AZ check: warn for primary instances (not read replicas) that
	// are not multi-AZ — single point of failure for production workloads.
	isReadReplica := db.ReadReplicaSourceDBInstanceIdentifier != ""
	if !db.MultiAZ && !isReadReplica {
		out = append(out, probe.Finding{
			Component:   subject,
			Severity:    probe.SeverityWarning,
			Message:     fmt.Sprintf("RDS instance %q is not multi-AZ (single-AZ deployment, no automatic failover)", db.Identifier),
			Remediation: fmt.Sprintf("aws rds modify-db-instance --db-instance-identifier %s --multi-az --apply-immediately", db.Identifier),
		})
	}

	// Backup retention check.
	if db.BackupRetentionPeriod == 0 {
		out = append(out, probe.Finding{
			Component:   subject,
			Severity:    probe.SeverityWarning,
			Message:     fmt.Sprintf("RDS instance %q has automated backups disabled (BackupRetentionPeriod=0)", db.Identifier),
			Remediation: fmt.Sprintf("aws rds modify-db-instance --db-instance-identifier %s --backup-retention-period 7 --apply-immediately", db.Identifier),
		})
	} else if db.BackupRetentionPeriod < backupRetentionWarnDays {
		out = append(out, probe.Finding{
			Component: subject,
			Severity:  probe.SeverityInfo,
			Message:   fmt.Sprintf("RDS instance %q backup retention is only %d days (recommended: >=%d days)", db.Identifier, db.BackupRetentionPeriod, backupRetentionWarnDays),
		})
	}

	return out
}
