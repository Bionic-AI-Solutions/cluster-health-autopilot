// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"
	"fmt"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/cloud"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/probe"
)

// ManagedIdentities flags user-assigned managed identities with no
// role assignments — orphaned identities referenced by a workload
// that grant nothing, so the workload silently lacks permissions.
// Mirrors AWS IAMRoles / GCP IAMServiceAccounts ("posture, not
// health").
type ManagedIdentities struct{}

const managedIdentitiesName = "azure-managed-identities"

// Name satisfies cloudprobe.Probe.
func (ManagedIdentities) Name() string { return managedIdentitiesName }

// Run satisfies cloudprobe.Probe.
func (ManagedIdentities) Run(ctx context.Context, src cloud.Source) probe.Result {
	azClient := src.Azure()
	if azClient == nil {
		return skipped(managedIdentitiesName, "Azure not configured (cloud.azure.enabled=false)")
	}
	ids, err := azClient.ListManagedIdentities(ctx)
	if err != nil {
		return probeFailed(managedIdentitiesName, "msi.ListUserAssignedIdentities", err)
	}

	var findings []probe.Finding
	for _, id := range ids {
		subject := fmt.Sprintf("azure-mi/%s/%s/%s", azClient.SubscriptionID(), id.ResourceGroup, id.Name)
		if id.RoleAssignmentN == 0 {
			findings = append(findings, probe.Finding{
				Component:   subject,
				Severity:    probe.SeverityWarning,
				Message:     fmt.Sprintf("Managed identity %s/%s has zero role assignments — workloads using it silently lack permissions", id.ResourceGroup, id.Name),
				Remediation: fmt.Sprintf("az role assignment list --assignee %s — if empty, grant the least-privilege role the workload needs, or delete the orphaned identity.", id.ClientID),
			})
		}
	}

	return probe.Result{
		Component: probe.ComponentResult{Component: managedIdentitiesName, Status: rollupStatus(findings), Detail: fmt.Sprintf("%d managed identity(ies) inspected in subscription %s", len(ids), azClient.SubscriptionID())},
		Findings:  findings,
	}
}

// Subnets flags VNet subnets approaching IP exhaustion. Mirrors AWS
// VPCSubnets / GCP Subnets.
type Subnets struct{}

const subnetsName = "azure-subnets"

const (
	subnetCritFreePercent = 10
	subnetWarnFreePercent = 25
)

// Name satisfies cloudprobe.Probe.
func (Subnets) Name() string { return subnetsName }

// Run satisfies cloudprobe.Probe.
func (Subnets) Run(ctx context.Context, src cloud.Source) probe.Result {
	azClient := src.Azure()
	if azClient == nil {
		return skipped(subnetsName, "Azure not configured (cloud.azure.enabled=false)")
	}
	subnets, err := azClient.ListSubnets(ctx)
	if err != nil {
		return probeFailed(subnetsName, "network.ListSubnets", err)
	}

	var findings []probe.Finding
	for _, s := range subnets {
		if s.TotalIPCount <= 0 {
			continue
		}
		freePercent := int(s.AvailableIPCount * 100 / s.TotalIPCount)
		subject := fmt.Sprintf("azure-subnet/%s/%s/%s", azClient.SubscriptionID(), s.VNet, s.Name)
		switch {
		case freePercent < subnetCritFreePercent:
			findings = append(findings, probe.Finding{
				Component:   subject,
				Severity:    probe.SeverityCritical,
				Message:     fmt.Sprintf("Subnet %q (%s) has %d%% free IPs (< %d%%); pod/node scheduling will fail soon", s.Name, s.AddressPrefix, freePercent, subnetCritFreePercent),
				Remediation: fmt.Sprintf("Expand the subnet prefix or add a new subnet to VNet %s.", s.VNet),
			})
		case freePercent < subnetWarnFreePercent:
			findings = append(findings, probe.Finding{
				Component: subject,
				Severity:  probe.SeverityWarning,
				Message:   fmt.Sprintf("Subnet %q (%s) has %d%% free IPs (< %d%%)", s.Name, s.AddressPrefix, freePercent, subnetWarnFreePercent),
			})
		}
	}

	return probe.Result{
		Component: probe.ComponentResult{Component: subnetsName, Status: rollupStatus(findings), Detail: fmt.Sprintf("%d subnet(s) inspected in subscription %s", len(subnets), azClient.SubscriptionID())},
		Findings:  findings,
	}
}
