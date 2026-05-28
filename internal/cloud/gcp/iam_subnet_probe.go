// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package gcp

import (
	"context"
	"fmt"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/cloud"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/probe"
)

// IAMServiceAccounts flags IAM service-account posture drift:
//   - disabled SAs that still carry user-managed keys (the keys keep
//     working for some Google APIs even while the SA is disabled —
//     a security smell) → warning
//   - SAs with > 2 user-managed keys → warning (long-lived key
//     sprawl; Workload Identity is the recommended keyless path)
//
// Mirrors the AWS IAMRoles probe's "posture, not health" framing.
type IAMServiceAccounts struct{}

const iamSAName = "gcp-iam-serviceaccounts"

// keySprawlThreshold is the user-managed-key count above which we
// warn. 2 allows a rotation overlap; beyond that is sprawl.
const keySprawlThreshold = 2

// Name satisfies cloudprobe.Probe.
func (IAMServiceAccounts) Name() string { return iamSAName }

// Run satisfies cloudprobe.Probe.
func (IAMServiceAccounts) Run(ctx context.Context, src cloud.Source) probe.Result {
	gcpClient := src.GCP()
	if gcpClient == nil {
		return skipped(iamSAName, "GCP not configured (cloud.gcp.enabled=false)")
	}
	sas, err := gcpClient.ListServiceAccounts(ctx)
	if err != nil {
		return probeFailed(iamSAName, "iam.ListServiceAccounts", err)
	}

	var findings []probe.Finding
	for _, sa := range sas {
		subject := fmt.Sprintf("gcp-iam-sa/%s/%s", gcpClient.Project(), sa.Email)
		if sa.Disabled && sa.KeyCount > 0 {
			findings = append(findings, probe.Finding{
				Component:   subject,
				Severity:    probe.SeverityWarning,
				Message:     fmt.Sprintf("Service account %s is disabled but still has %d user-managed key(s)", sa.Email, sa.KeyCount),
				Remediation: fmt.Sprintf("Delete the orphaned keys: gcloud iam service-accounts keys list --iam-account=%s — then `keys delete` each.", sa.Email),
			})
			continue
		}
		if sa.KeyCount > keySprawlThreshold {
			findings = append(findings, probe.Finding{
				Component:   subject,
				Severity:    probe.SeverityWarning,
				Message:     fmt.Sprintf("Service account %s has %d user-managed keys (> %d); long-lived key sprawl", sa.Email, sa.KeyCount, keySprawlThreshold),
				Remediation: fmt.Sprintf("Prefer Workload Identity over keys. Audit: gcloud iam service-accounts keys list --iam-account=%s; migrate workloads to WI and delete keys.", sa.Email),
			})
		}
	}

	return probe.Result{
		Component: probe.ComponentResult{
			Component: iamSAName,
			Status:    rollupStatus(findings),
			Detail:    fmt.Sprintf("%d service account(s) inspected in project %s", len(sas), gcpClient.Project()),
		},
		Findings: findings,
	}
}

// Subnets flags VPC subnetworks approaching IP exhaustion:
//   - < 10% free addresses → critical
//   - < 25% free addresses → warning
//
// Mirrors the AWS VPCSubnets probe.
type Subnets struct{}

const subnetsName = "gcp-subnets"

const (
	subnetCritFreePercent = 10
	subnetWarnFreePercent = 25
)

// Name satisfies cloudprobe.Probe.
func (Subnets) Name() string { return subnetsName }

// Run satisfies cloudprobe.Probe.
func (Subnets) Run(ctx context.Context, src cloud.Source) probe.Result {
	gcpClient := src.GCP()
	if gcpClient == nil {
		return skipped(subnetsName, "GCP not configured (cloud.gcp.enabled=false)")
	}
	subnets, err := gcpClient.ListSubnets(ctx)
	if err != nil {
		return probeFailed(subnetsName, "compute.ListSubnetworks", err)
	}

	var findings []probe.Finding
	for _, s := range subnets {
		if s.TotalIPCount <= 0 {
			continue // can't compute a percentage; skip rather than divide-by-zero
		}
		freePercent := int(s.AvailableIPCount * 100 / s.TotalIPCount)
		subject := fmt.Sprintf("gcp-subnet/%s/%s/%s", gcpClient.Project(), s.Region, s.Name)
		switch {
		case freePercent < subnetCritFreePercent:
			findings = append(findings, probe.Finding{
				Component:   subject,
				Severity:    probe.SeverityCritical,
				Message:     fmt.Sprintf("Subnet %q (%s) has %d%% free IPs (< %d%%); pod/node scheduling will fail soon", s.Name, s.IPCIDRRange, freePercent, subnetCritFreePercent),
				Remediation: fmt.Sprintf("Expand the secondary range or add a new subnet: gcloud compute networks subnets expand-ip-range %s --region=%s --project=%s", s.Name, s.Region, gcpClient.Project()),
			})
		case freePercent < subnetWarnFreePercent:
			findings = append(findings, probe.Finding{
				Component: subject,
				Severity:  probe.SeverityWarning,
				Message:   fmt.Sprintf("Subnet %q (%s) has %d%% free IPs (< %d%%)", s.Name, s.IPCIDRRange, freePercent, subnetWarnFreePercent),
			})
		}
	}

	return probe.Result{
		Component: probe.ComponentResult{
			Component: subnetsName,
			Status:    rollupStatus(findings),
			Detail:    fmt.Sprintf("%d subnet(s) inspected in project %s", len(subnets), gcpClient.Project()),
		},
		Findings: findings,
	}
}
