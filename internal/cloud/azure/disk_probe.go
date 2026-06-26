// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"
	"fmt"
	"time"

	"github.com/srenix-ai/agentic-sre/pkg/cloud"
	pkgazure "github.com/srenix-ai/agentic-sre/pkg/cloud/azure"
	"github.com/srenix-ai/agentic-sre/pkg/probe"
)

// Disks reports drift on Azure Managed Disk resources:
//
//   - ProvisioningState=Failed → critical
//   - ProvisioningState=Creating / Updating past 1h grace → warning
//   - DiskState=Unattached past 24h cleanup grace → warning (billing
//     leak / orphaned PV)
type Disks struct{}

const disksName = "azure-disks"

// detachedCleanupGrace matches the GCP probe — 24h before a detached
// disk is flagged as orphaned.
const detachedCleanupGrace = 24 * time.Hour

// transitionalGrace — 1h dwell before flagging stuck Creating /
// Updating provisioning state.
const transitionalGrace = time.Hour

// Name satisfies cloudprobe.Probe.
func (Disks) Name() string { return disksName }

// Run satisfies cloudprobe.Probe.
func (Disks) Run(ctx context.Context, src cloud.Source) probe.Result {
	azClient := src.Azure()
	if azClient == nil {
		return probe.Result{
			Component: probe.ComponentResult{
				Component: disksName,
				Status:    "SKIPPED",
				Detail:    "Azure not configured (cloud.azure.enabled=false)",
			},
		}
	}

	disks, err := azClient.ListDisks(ctx)
	if err != nil {
		return probe.Result{
			Component: probe.ComponentResult{
				Component: disksName,
				Status:    "PROBE_FAILED",
				Detail:    fmt.Sprintf("ListDisks: %v", err),
			},
		}
	}

	var findings []probe.Finding
	now := time.Now()
	for _, d := range disks {
		findings = append(findings, evaluateDisk(d, azClient.SubscriptionID(), now)...)
	}

	return probe.Result{
		Component: probe.ComponentResult{
			Component: disksName,
			Status:    rollupStatus(findings),
			Detail:    fmt.Sprintf("%d disk(s) inspected in subscription %s", len(disks), azClient.SubscriptionID()),
		},
		Findings: findings,
	}
}

// evaluateDisk is pure.
func evaluateDisk(d pkgazure.Disk, subscription string, now time.Time) []probe.Finding {
	subject := fmt.Sprintf("azure-disk/%s/%s/%s", subscription, d.ResourceGroup, d.Name)

	switch d.ProvisioningState {
	case "Failed":
		return []probe.Finding{{
			Component:   subject,
			Severity:    probe.SeverityCritical,
			Message:     fmt.Sprintf("Azure Managed Disk %s/%s (%dGB %s) is in Failed provisioning state", d.ResourceGroup, d.Name, d.SizeGB, d.SKU),
			Remediation: fmt.Sprintf("az disk show -g %s -n %s --subscription %s — inspect last provisioning operation + recreate if needed.", d.ResourceGroup, d.Name, subscription),
		}}
	case "Creating", "Updating":
		if !d.CreatedAt.IsZero() && now.Sub(d.CreatedAt) > transitionalGrace {
			return []probe.Finding{{
				Component:   subject,
				Severity:    probe.SeverityWarning,
				Message:     fmt.Sprintf("Azure Managed Disk %s/%s stuck in %s for >%s", d.ResourceGroup, d.Name, d.ProvisioningState, transitionalGrace),
				Remediation: fmt.Sprintf("az disk show -g %s -n %s --subscription %s", d.ResourceGroup, d.Name, subscription),
			}}
		}
		return nil
	case "Succeeded":
		// Attached disks are healthy. Unattached past grace is a
		// billing leak / orphaned PV smell.
		if d.DiskState == "Unattached" && d.AttachedToVM == "" {
			dwell := d.DetachedDuration
			if dwell == 0 && !d.CreatedAt.IsZero() {
				dwell = now.Sub(d.CreatedAt)
			}
			if dwell > detachedCleanupGrace {
				return []probe.Finding{{
					Component:   subject,
					Severity:    probe.SeverityWarning,
					Message:     fmt.Sprintf("Azure Managed Disk %s/%s (%dGB %s) has been detached for >%s — possible billing leak or orphaned PV", d.ResourceGroup, d.Name, d.SizeGB, d.SKU, detachedCleanupGrace),
					Remediation: fmt.Sprintf("Confirm the disk is unneeded then delete: az disk delete -g %s -n %s --subscription %s --yes", d.ResourceGroup, d.Name, subscription),
				}}
			}
		}
		return nil
	default:
		return []probe.Finding{{
			Component: subject,
			Severity:  probe.SeverityWarning,
			Message:   fmt.Sprintf("Azure Managed Disk %s/%s has unknown provisioningState=%q", d.ResourceGroup, d.Name, d.ProvisioningState),
		}}
	}
}
