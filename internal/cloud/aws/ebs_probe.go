// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/srenix-ai/agentic-sre/pkg/cloud"
	"github.com/srenix-ai/agentic-sre/pkg/probe"
)

// EBSVolumes flags EBS volumes that are:
//   - in state != "available" / "in-use" → warning (transitional) or critical (error)
//   - detached AND created > 7 days ago → warning (likely abandoned, costing money)
type EBSVolumes struct{}

const ebsName = "aws-ebs"

const detachedAgeThreshold = 7 * 24 * time.Hour

// Name satisfies cloudprobe.Probe.
func (EBSVolumes) Name() string { return ebsName }

// Run satisfies cloudprobe.Probe.
func (EBSVolumes) Run(ctx context.Context, src cloud.Source) probe.Result {
	awsClient := src.AWS()
	if awsClient == nil {
		return skipped(ebsName, "AWS not configured (cloud.aws.enabled=false)")
	}
	vols, err := awsClient.DescribeVolumes(ctx)
	if err != nil {
		return probeFailed(ebsName, "ec2.DescribeVolumes", err)
	}
	var findings []probe.Finding
	for _, v := range vols {
		subject := fmt.Sprintf("aws-ebs/%s/%s", awsClient.Region(), v.VolumeID)
		switch v.State {
		case "in-use", "available":
			// healthy state machine
		case "error":
			findings = append(findings, probe.Finding{
				Component: subject, Severity: probe.SeverityCritical,
				Message:     fmt.Sprintf("EBS volume %s in error state (size %dGB, type %s)", v.VolumeID, v.SizeGB, v.VolumeType),
				Remediation: fmt.Sprintf("aws ec2 describe-volumes --volume-ids %s && aws ec2 describe-volume-status --volume-ids %s", v.VolumeID, v.VolumeID),
			})
		default:
			findings = append(findings, probe.Finding{
				Component: subject, Severity: probe.SeverityWarning,
				Message: fmt.Sprintf("EBS volume %s in transitional state %q", v.VolumeID, v.State),
			})
		}
		if v.State == "available" && v.DetachedDuration > detachedAgeThreshold {
			findings = append(findings, probe.Finding{
				Component: subject, Severity: probe.SeverityWarning,
				Message: fmt.Sprintf("EBS volume %s detached for %s (>%s) — likely abandoned (size %dGB)",
					v.VolumeID, v.DetachedDuration.Round(time.Hour), detachedAgeThreshold, v.SizeGB),
				Remediation: fmt.Sprintf("aws ec2 delete-volume --volume-id %s   # after confirming no snapshots needed", v.VolumeID),
			})
		}
	}
	return probe.Result{
		Component: probe.ComponentResult{
			Component: ebsName,
			Status:    rollupStatus(findings),
			Detail:    fmt.Sprintf("%d volume(s) inspected in %s", len(vols), awsClient.Region()),
		},
		Findings: findings,
	}
}
