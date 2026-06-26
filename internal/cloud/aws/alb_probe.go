// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"fmt"

	intcloud "github.com/srenix-ai/agentic-sre/internal/cloud"
	"github.com/srenix-ai/agentic-sre/pkg/cloud"
	"github.com/srenix-ai/agentic-sre/pkg/probe"
)

// ALBTargetHealth flags target groups with zero healthy targets
// (critical) and target groups where every target is unhealthy
// while at least one used to be healthy (also critical — fresh
// failure). Initial-state targets are not counted as failed.
type ALBTargetHealth struct{}

const albName = "aws-alb-target-health"

// Name satisfies cloudprobe.Probe.
func (ALBTargetHealth) Name() string { return albName }

// Run satisfies cloudprobe.Probe.
func (ALBTargetHealth) Run(ctx context.Context, src cloud.Source) probe.Result {
	awsClient := src.AWS()
	if awsClient == nil {
		return skipped(albName, "AWS not configured")
	}
	tgs, err := awsClient.DescribeALBTargetGroupsWithHealth(ctx)
	if err != nil {
		return probeFailed(albName, "elbv2.DescribeTargetGroups", err)
	}
	var findings []probe.Finding
	for _, tg := range tgs {
		subject := fmt.Sprintf("aws-alb-tg/%s/%s", awsClient.Region(), tg.Name)
		// "Zero healthy + any unhealthy" is the canonical outage signature.
		if tg.HealthyCount == 0 && tg.UnhealthyCount > 0 {
			findings = append(findings, probe.Finding{
				Component: subject, Severity: probe.SeverityCritical,
				// The "(lb: <LB DNS name>)" suffix is the Srenix Enterprise RCA
				// join key (omitted when the LB is unresolved) — see
				// internal/cloud/joinkeys.go.
				// contract: internal/cloud/contract_test.go
				Message: fmt.Sprintf("Target group %s has 0 healthy targets (%d unhealthy) on %s:%d",
					tg.Name, tg.UnhealthyCount, tg.Protocol, tg.Port) + intcloud.JoinKeyLB(tg.LoadBalancerDNS),
				Remediation: fmt.Sprintf("aws elbv2 describe-target-health --target-group-arn %s", tg.ARN),
			})
			continue
		}
		// "Zero healthy + zero unhealthy + zero initial" → empty target group
		// (registration drift). Warn.
		if tg.HealthyCount == 0 && tg.UnhealthyCount == 0 && tg.InitialCount == 0 {
			findings = append(findings, probe.Finding{
				Component: subject, Severity: probe.SeverityWarning,
				Message: fmt.Sprintf("Target group %s has no registered targets (likely registration drift)", tg.Name),
			})
		}
	}
	return probe.Result{
		Component: probe.ComponentResult{
			Component: albName,
			Status:    rollupStatus(findings),
			Detail:    fmt.Sprintf("%d target group(s) inspected in %s", len(tgs), awsClient.Region()),
		},
		Findings: findings,
	}
}
