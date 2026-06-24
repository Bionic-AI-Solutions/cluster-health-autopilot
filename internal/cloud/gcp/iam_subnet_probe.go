// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package gcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/cloud"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/probe"
)

// IAMServiceAccounts flags IAM service-account posture drift:
//   - disabled SAs that still carry user-managed keys (the keys keep
//     working for some Google APIs even while the SA is disabled —
//     a security smell) → warning
//   - SAs with > 2 user-managed keys → warning (long-lived key
//     sprawl; Workload Identity is the recommended keyless path)
//   - SAs with a Workload Identity binding AND user-managed keys →
//     warning (mixed configuration: both auth paths active, which
//     can cause confusion and weakens the keyless posture)
//
// The OAuth2Bound field signals a Workload Identity binding (the live
// client derives it from a roles/iam.workloadIdentityUser binding on the
// SA's IAM policy; see live.go hasWorkloadIdentityBinding). It is treated
// as a *positive* signal only: a finding fires when WI is present AND keys
// coexist. We deliberately do NOT flag the *absence* of a binding — that
// would false-positive on the recommended keyless WI posture and on any
// SA whose IAM policy we couldn't read (best-effort getIamPolicy), which
// would page operators on their best-practice service accounts.
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
		// Workload Identity binding drift check (positive-signal only).
		// OAuth2Bound==true means we positively detected a
		// roles/iam.workloadIdentityUser binding on this SA, so a
		// coexisting user-managed key is a genuine mixed-mode finding.
		// We do NOT flag !OAuth2Bound: a keyless WI SA is the recommended
		// posture, and a getIamPolicy read failure must never page it.
		if sa.OAuth2Bound && sa.KeyCount > 0 {
			// Mixed configuration: SA has both a Workload Identity binding
			// and user-managed keys. The keys undermine the keyless posture
			// and can cause confusion (which credential wins?).
			findings = append(findings, probe.Finding{
				Component:   subject,
				Severity:    probe.SeverityWarning,
				Message:     fmt.Sprintf("GCP service account %s has both Workload Identity binding and %d user-managed key(s) (mixed configuration)", sa.Email, sa.KeyCount),
				Remediation: fmt.Sprintf("Remove user-managed keys to enforce Workload Identity: gcloud iam service-accounts keys list --iam-account=%s", sa.Email),
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

// Subnets flags VPC subnetworks at IP-exhaustion risk. Two modes,
// depending on what the client could measure:
//
// Measured (AvailableIPCount >= 0 — snapshot files / future clients):
//   - < 10% free addresses → critical
//   - < 25% free addresses → warning
//
// Capacity-only (AvailableIPCount = -1 — the live wrapper; GCP exposes
// no cheap used-IP count, see live.go ListSubnets):
//   - primary CIDR smaller than /26 (configurable via
//     SmallPrefixThreshold) → warning. A small subnet is the
//     IP-exhaustion precondition we CAN see without the Recommender
//     API; actual utilization lives in Network Analyzer.
//
// Mirrors the AWS VPCSubnets probe (which measures live).
type Subnets struct {
	// SmallPrefixThreshold is the primary-CIDR prefix length beyond
	// which an unmeasured subnet is flagged as small-capacity (the
	// capacity-only mode above). 0 means the default (/26 → 60 usable
	// IPs). Set to a larger value (e.g. 28) to quiet intentionally
	// tiny subnets, or smaller to be stricter.
	SmallPrefixThreshold int
}

const subnetsName = "gcp-subnets"

const (
	subnetCritFreePercent = 10
	subnetWarnFreePercent = 25
	// defaultSmallPrefixThreshold flags unmeasured subnets smaller
	// than a /26 (60 usable addresses after GCP's 4 reserved).
	defaultSmallPrefixThreshold = 26
)

// Name satisfies cloudprobe.Probe.
func (Subnets) Name() string { return subnetsName }

// Run satisfies cloudprobe.Probe.
func (p Subnets) Run(ctx context.Context, src cloud.Source) probe.Result {
	gcpClient := src.GCP()
	if gcpClient == nil {
		return skipped(subnetsName, "GCP not configured (cloud.gcp.enabled=false)")
	}
	subnets, err := gcpClient.ListSubnets(ctx)
	if err != nil {
		return probeFailed(subnetsName, "compute.ListSubnetworks", err)
	}
	smallPrefix := p.SmallPrefixThreshold
	if smallPrefix <= 0 {
		smallPrefix = defaultSmallPrefixThreshold
	}

	var findings []probe.Finding
	var unmeasured int
	for _, s := range subnets {
		if s.TotalIPCount <= 0 {
			continue // can't compute a percentage; skip rather than divide-by-zero
		}
		subject := fmt.Sprintf("gcp-subnet/%s/%s/%s", gcpClient.Project(), s.Region, s.Name)
		if s.AvailableIPCount < 0 {
			// Free-IP count not measured (the live wrapper — GCP
			// exposes no cheap used-IP count; the allocation-ratio
			// insight is behind the Recommender API). Capacity-only
			// mode: flag small primary ranges instead of pretending
			// the subnet is 100% free (which would silently never
			// fire).
			unmeasured++
			if prefix := cidrPrefix(s.IPCIDRRange); prefix > smallPrefix {
				findings = append(findings, probe.Finding{
					Component:   subject,
					Severity:    probe.SeverityWarning,
					Message:     fmt.Sprintf("Subnet %q primary range %s provides only %d usable IPs (smaller than /%d); per-IP utilization is not exposed by GCP's Compute API — review utilization in Network Analyzer", s.Name, s.IPCIDRRange, s.TotalIPCount, smallPrefix),
					Remediation: fmt.Sprintf("Expand the primary range: gcloud compute networks subnets expand-ip-range %s --region=%s --project=%s --prefix-length=<shorter>; or check Network Analyzer's IP utilization insight for the real allocation ratio.", s.Name, s.Region, gcpClient.Project()),
				})
			}
			continue
		}
		freePercent := int(s.AvailableIPCount * 100 / s.TotalIPCount)
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

	detail := fmt.Sprintf("%d subnet(s) inspected in project %s", len(subnets), gcpClient.Project())
	if unmeasured > 0 {
		detail += fmt.Sprintf("; per-IP utilization not measured for %d — capacity-only (GCP exposes no used-IP count; see Network Analyzer)", unmeasured)
	}
	return probe.Result{
		Component: probe.ComponentResult{
			Component: subnetsName,
			Status:    rollupStatus(findings),
			Detail:    detail,
		},
		Findings: findings,
	}
}

// cidrPrefix returns the prefix length of an IPv4 CIDR ("10.0.0.0/28"
// → 28), or -1 when unparseable / absent — callers then skip the
// capacity check rather than guessing.
func cidrPrefix(cidr string) int {
	i := strings.LastIndex(cidr, "/")
	if i < 0 {
		return -1
	}
	var mask int
	if _, err := fmt.Sscanf(cidr[i+1:], "%d", &mask); err != nil || mask < 0 || mask > 32 {
		return -1
	}
	return mask
}
