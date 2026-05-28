// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package gcp

import (
	"context"
	"fmt"
	"time"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/cloud"
	pkggcp "github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/cloud/gcp"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/probe"
)

// PersistentDisks reports drift on GCP Persistent Disks:
//
//   - any disk in non-terminal failure state (FAILED) → critical
//   - any disk in transitional state past a grace period → warning
//     (CREATING / RESTORING / DELETING for > 1h is suspicious)
//   - any detached disk older than the cleanup grace → warning
//     (potential billing leak; orphaned PVs)
//
// Mirrors the shape of internal/cloud/aws.EBSVolumes — same semantics,
// GCP-specific resource model (zonal vs regional disks; attachment
// to a VM rather than a generic instance ID).
type PersistentDisks struct{}

const persistentDisksName = "gcp-persistent-disks"

// detachedCleanupGrace is the dwell time before a detached disk is
// flagged. Tuned for v1 (24h); operators can ignore or set up
// auto-cleanup at their leisure.
const detachedCleanupGrace = 24 * time.Hour

// transitionalGrace is the dwell time before a CREATING / RESTORING /
// DELETING disk is flagged. Most legitimate state transitions resolve
// in seconds; an hour means the provisioner is stuck.
const transitionalGrace = time.Hour

// Name satisfies cloudprobe.Probe.
func (PersistentDisks) Name() string { return persistentDisksName }

// Run satisfies cloudprobe.Probe.
//
// Inspects all Persistent Disks visible to the bound GCP sub-client.
// Returns SKIPPED when GCP is not configured.
func (PersistentDisks) Run(ctx context.Context, src cloud.Source) probe.Result {
	gcpClient := src.GCP()
	if gcpClient == nil {
		return probe.Result{
			Component: probe.ComponentResult{
				Component: persistentDisksName,
				Status:    "SKIPPED",
				Detail:    "GCP not configured (cloud.gcp.enabled=false)",
			},
		}
	}

	disks, err := gcpClient.ListPersistentDisks(ctx)
	if err != nil {
		return probe.Result{
			Component: probe.ComponentResult{
				Component: persistentDisksName,
				Status:    "PROBE_FAILED",
				Detail:    fmt.Sprintf("ListPersistentDisks: %v", err),
			},
		}
	}

	var findings []probe.Finding
	now := time.Now()
	for _, d := range disks {
		findings = append(findings, evaluatePersistentDisk(d, gcpClient.Project(), now)...)
	}

	return probe.Result{
		Component: probe.ComponentResult{
			Component: persistentDisksName,
			Status:    rollupStatus(findings),
			Detail:    fmt.Sprintf("%d disk(s) inspected in project %s", len(disks), gcpClient.Project()),
		},
		Findings: findings,
	}
}

// evaluatePersistentDisk is pure — exhaustively unit-tested.
func evaluatePersistentDisk(d pkggcp.PersistentDisk, project string, now time.Time) []probe.Finding {
	scope := d.Region
	if scope == "" {
		scope = d.Zone
	}
	subject := fmt.Sprintf("gcp-pd/%s/%s/%s", project, scope, d.Name)

	switch d.Status {
	case "FAILED":
		return []probe.Finding{{
			Component:   subject,
			Severity:    probe.SeverityCritical,
			Message:     fmt.Sprintf("Persistent Disk %q (%dGB %s) is in FAILED state", d.Name, d.SizeGB, d.Type),
			Remediation: fmt.Sprintf("gcloud compute disks describe %s --zone=%s --project=%s — inspect operationStatus + sourceImage and recreate if necessary.", d.Name, d.Zone, project),
		}}
	case "CREATING", "RESTORING", "DELETING":
		if !d.CreatedAt.IsZero() && now.Sub(d.CreatedAt) > transitionalGrace {
			return []probe.Finding{{
				Component: subject,
				Severity:  probe.SeverityWarning,
				Message:   fmt.Sprintf("Persistent Disk %q stuck in %s for >%s", d.Name, d.Status, transitionalGrace),
				Remediation: fmt.Sprintf("Check the disk's most recent operation: "+
					"`gcloud compute operations list --filter='targetLink~%s' --project=%s --limit=10`.",
					d.Name, project),
			}}
		}
		// Within grace — silent.
		return nil
	case "READY":
		// Attached disks are healthy. Detached disks past grace are
		// a billing leak / orphaned-PV smell.
		if d.AttachedToVM == "" {
			dwell := d.DetachedDuration
			if dwell == 0 && !d.CreatedAt.IsZero() {
				// Snapshot mode without explicit DetachedDuration —
				// fall back to age. Conservative: only flag if the
				// disk has existed for >cleanupGrace AND is detached
				// (so brand-new disks waiting for attachment don't
				// fire).
				dwell = now.Sub(d.CreatedAt)
			}
			if dwell > detachedCleanupGrace {
				return []probe.Finding{{
					Component:   subject,
					Severity:    probe.SeverityWarning,
					Message:     fmt.Sprintf("Persistent Disk %q (%dGB %s) has been detached for >%s — possible billing leak or orphaned PV", d.Name, d.SizeGB, d.Type, detachedCleanupGrace),
					Remediation: fmt.Sprintf("Confirm the disk is unneeded (no PV references it) then delete: gcloud compute disks delete %s --zone=%s --project=%s", d.Name, d.Zone, project),
				}}
			}
		}
		return nil
	default:
		return []probe.Finding{{
			Component: subject,
			Severity:  probe.SeverityWarning,
			Message:   fmt.Sprintf("Persistent Disk %q has unknown status %q", d.Name, d.Status),
		}}
	}
}
