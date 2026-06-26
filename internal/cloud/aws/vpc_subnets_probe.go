// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"fmt"

	"github.com/srenix-ai/agentic-sre/pkg/cloud"
	"github.com/srenix-ai/agentic-sre/pkg/probe"
)

// VPCSubnets flags subnets with low IPv4 address availability — the
// most common subnet-related outage in K8s clusters (CNI pod IP
// allocation failing because the subnet is exhausted).
//
// Thresholds: < subnetWarnIPs (10) → warning, < subnetCritIPs (3) →
// critical.
type VPCSubnets struct{}

const vpcName = "aws-vpc-subnets"

const (
	subnetWarnIPs int32 = 10
	subnetCritIPs int32 = 3
)

// Name satisfies cloudprobe.Probe.
func (VPCSubnets) Name() string { return vpcName }

// Run satisfies cloudprobe.Probe.
func (VPCSubnets) Run(ctx context.Context, src cloud.Source) probe.Result {
	awsClient := src.AWS()
	if awsClient == nil {
		return skipped(vpcName, "AWS not configured")
	}
	subnets, err := awsClient.DescribeSubnets(ctx)
	if err != nil {
		return probeFailed(vpcName, "ec2.DescribeSubnets", err)
	}
	var findings []probe.Finding
	for _, s := range subnets {
		subject := fmt.Sprintf("aws-vpc-subnet/%s/%s", awsClient.Region(), s.SubnetID)
		switch {
		case s.AvailableIPv4AddressCount < subnetCritIPs:
			findings = append(findings, probe.Finding{
				Component: subject, Severity: probe.SeverityCritical,
				Message: fmt.Sprintf("Subnet %s (%s, %s) has %d available IPv4 — exhaustion imminent",
					s.SubnetID, s.CIDRBlock, s.AvailabilityZone, s.AvailableIPv4AddressCount),
				Remediation: fmt.Sprintf("Expand the subnet, add a new subnet to the same AZ, or check what's consuming IPs: aws ec2 describe-network-interfaces --filters Name=subnet-id,Values=%s", s.SubnetID),
			})
		case s.AvailableIPv4AddressCount < subnetWarnIPs:
			findings = append(findings, probe.Finding{
				Component: subject, Severity: probe.SeverityWarning,
				Message: fmt.Sprintf("Subnet %s (%s, %s) has %d available IPv4 (<%d threshold)",
					s.SubnetID, s.CIDRBlock, s.AvailabilityZone, s.AvailableIPv4AddressCount, subnetWarnIPs),
			})
		}
	}
	return probe.Result{
		Component: probe.ComponentResult{
			Component: vpcName,
			Status:    rollupStatus(findings),
			Detail:    fmt.Sprintf("%d subnet(s) inspected in %s", len(subnets), awsClient.Region()),
		},
		Findings: findings,
	}
}
