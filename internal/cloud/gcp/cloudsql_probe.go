// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

// Package gcp contains the GCP implementation of the cloud probe
// surface. Each probe is a single Go file (cloudsql_probe.go,
// disk_probe.go, etc.) implementing cloudprobe.Probe; all hold the
// bound gcp.Client (via cloud.Source.GCP()) and are read-only.
package gcp

import (
	"context"
	"fmt"
	"time"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/cloud"
	pkggcp "github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/cloud/gcp"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/probe"
)

// CloudSQL reports drift on GCP Cloud SQL instances:
//
//   - any instance whose state is not RUNNABLE → warning or critical
//     depending on which non-RUNNABLE state
//   - any instance with DiskUsedPercent ≥ 80 → warning
//   - any instance with DiskUsedPercent ≥ 90 → critical (when
//     StorageAutoResize is false; auto-resize-enabled instances are
//     suppressed since GCP grows them automatically)
//   - instances with BackupConfigured=false → warning (operator-config
//     drift; production CloudSQL should always have automated backups)
//
// AvailabilityType is not severity-modified — operators care equally
// about disk filling whether the instance is REGIONAL or ZONAL.
type CloudSQL struct{}

const cloudSQLName = "gcp-cloudsql"

// Storage-utilization thresholds. Tuned for v1; can be made
// configurable later if operators ask.
const (
	storageWarnPercent = 80
	storageCritPercent = 90
)

// Name satisfies cloudprobe.Probe and is rendered in Slack /
// DriftReport / Alertmanager as the component label.
func (CloudSQL) Name() string { return cloudSQLName }

// Run satisfies cloudprobe.Probe.
//
// Inspects all Cloud SQL instances visible to the bound GCP sub-client.
// Returns SKIPPED when the source has no GCP configured.
func (CloudSQL) Run(ctx context.Context, src cloud.Source) probe.Result {
	gcpClient := src.GCP()
	if gcpClient == nil {
		return probe.Result{
			Component: probe.ComponentResult{
				Component: cloudSQLName,
				Status:    "SKIPPED",
				Detail:    "GCP not configured (cloud.gcp.enabled=false)",
			},
		}
	}

	instances, err := gcpClient.ListCloudSQLInstances(ctx)
	if err != nil {
		return probe.Result{
			Component: probe.ComponentResult{
				Component: cloudSQLName,
				Status:    "PROBE_FAILED",
				Detail:    fmt.Sprintf("ListCloudSQLInstances: %v", err),
			},
		}
	}

	var findings []probe.Finding
	for _, db := range instances {
		findings = append(findings, evaluateCloudSQLInstance(db, gcpClient.Project())...)
	}

	return probe.Result{
		Component: probe.ComponentResult{
			Component: cloudSQLName,
			Status:    rollupStatus(findings),
			Detail:    fmt.Sprintf("%d instance(s) inspected in project %s", len(instances), gcpClient.Project()),
		},
		Findings: findings,
	}
}

// evaluateCloudSQLInstance is pure — easy to unit-test exhaustively.
// Returns zero or more findings per instance.
func evaluateCloudSQLInstance(db pkggcp.CloudSQLInstance, project string) []probe.Finding {
	subject := fmt.Sprintf("gcp-cloudsql/%s/%s", project, db.Name)
	var out []probe.Finding

	// State check.
	switch db.State {
	case "RUNNABLE":
		// healthy
	case "FAILED":
		out = append(out, probe.Finding{
			Component:   subject,
			Severity:    probe.SeverityCritical,
			Message:     fmt.Sprintf("Cloud SQL instance %q (%s) is in FAILED state", db.Name, db.DatabaseVersion),
			Remediation: fmt.Sprintf("gcloud sql operations list --instance=%s --project=%s --limit=10 — inspect the most recent failed operations.", db.Name, project),
		})
	case "SUSPENDED":
		out = append(out, probe.Finding{
			Component: subject,
			Severity:  probe.SeverityCritical,
			Message:   fmt.Sprintf("Cloud SQL instance %q is SUSPENDED (typically billing or quota); writes will fail", db.Name),
			Remediation: fmt.Sprintf("Check billing + quota at https://console.cloud.google.com/sql/instances/%s/overview?project=%s",
				db.Name, project),
		})
	default:
		// PENDING_CREATE / PENDING_DELETE / MAINTENANCE / UNKNOWN_STATE
		// — transitional or unknown. Warn but don't critical so we
		// don't page on every routine maintenance window.
		out = append(out, probe.Finding{
			Component: subject,
			Severity:  probe.SeverityWarning,
			Message:   fmt.Sprintf("Cloud SQL instance %q state=%q (not RUNNABLE)", db.Name, db.State),
		})
	}

	// Storage utilization — emitted independently of state. Auto-resize
	// instances are skipped: GCP grows them automatically, and flagging
	// 90% on those is operator-noise.
	if !db.StorageAutoResize {
		if db.DiskUsedPercent >= storageCritPercent {
			out = append(out, probe.Finding{
				Component:   subject,
				Severity:    probe.SeverityCritical,
				Message:     fmt.Sprintf("Cloud SQL instance %q disk at %d%% of %d GB (>= %d%%)", db.Name, db.DiskUsedPercent, db.DiskSizeGB, storageCritPercent),
				Remediation: fmt.Sprintf("gcloud sql instances patch %s --storage-size=<larger> --project=%s", db.Name, project),
			})
		} else if db.DiskUsedPercent >= storageWarnPercent {
			out = append(out, probe.Finding{
				Component: subject,
				Severity:  probe.SeverityWarning,
				Message:   fmt.Sprintf("Cloud SQL instance %q disk at %d%% (>= %d%%)", db.Name, db.DiskUsedPercent, storageWarnPercent),
			})
		}
	}

	// Backup posture — drift signal independent of state. A production
	// instance without automated backups is a future-incident in
	// waiting. Warn unconditionally; the operator decides if the
	// instance is genuinely transient or needs the backup config.
	if !db.BackupConfigured {
		out = append(out, probe.Finding{
			Component:   subject,
			Severity:    probe.SeverityWarning,
			Message:     fmt.Sprintf("Cloud SQL instance %q has no automated backups configured", db.Name),
			Remediation: fmt.Sprintf("gcloud sql instances patch %s --backup-start-time=02:00 --project=%s", db.Name, project),
		})
	}

	return out
}

// rollupStatus folds per-finding severity into a single component-level
// status string matching the convention used by internal/probe (HEALTHY,
// DEGRADED, CRITICAL). Mirrors AWS internal/cloud/aws.rollupStatus.
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

// Ensure time is imported (used by tests via the types package).
var _ = time.Time{}
