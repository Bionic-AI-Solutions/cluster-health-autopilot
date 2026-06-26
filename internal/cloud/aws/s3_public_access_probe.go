// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"fmt"

	"github.com/srenix-ai/agentic-sre/pkg/cloud"
	"github.com/srenix-ai/agentic-sre/pkg/probe"
)

// S3BucketPublicAccess flags buckets where the Public Access Block
// (PAB) is missing or any of its four flags is false. The four flags
// must all be true for the bucket to be considered "publicly-blocked"
// — anything less is potential public-data drift.
//
// Buckets in a different region than the bound region (or that
// errored on GetPublicAccessBlock) are surfaced as warnings rather
// than failing the whole probe.
type S3BucketPublicAccess struct{}

const s3Name = "aws-s3-bucket-public-access"

// Name satisfies cloudprobe.Probe.
func (S3BucketPublicAccess) Name() string { return s3Name }

// Run satisfies cloudprobe.Probe.
func (S3BucketPublicAccess) Run(ctx context.Context, src cloud.Source) probe.Result {
	awsClient := src.AWS()
	if awsClient == nil {
		return skipped(s3Name, "AWS not configured")
	}
	pabs, err := awsClient.ListS3BucketPAB(ctx)
	if err != nil {
		return probeFailed(s3Name, "s3.ListBuckets / GetPublicAccessBlock", err)
	}
	var findings []probe.Finding
	for _, p := range pabs {
		subject := fmt.Sprintf("aws-s3/%s", p.Bucket)
		if p.HasPolicyError {
			findings = append(findings, probe.Finding{
				Component: subject, Severity: probe.SeverityCritical,
				Message:     fmt.Sprintf("S3 bucket %s has no Public Access Block configuration", p.Bucket),
				Remediation: fmt.Sprintf("aws s3api put-public-access-block --bucket %s --public-access-block-configuration BlockPublicAcls=true,IgnorePublicAcls=true,BlockPublicPolicy=true,RestrictPublicBuckets=true", p.Bucket),
			})
			continue
		}
		if !p.BlockPublicAcls || !p.IgnorePublicAcls || !p.BlockPublicPolicy || !p.RestrictPublicBuckets {
			findings = append(findings, probe.Finding{
				Component: subject, Severity: probe.SeverityCritical,
				Message: fmt.Sprintf("S3 bucket %s has partial PAB: BlockPublicAcls=%t IgnorePublicAcls=%t BlockPublicPolicy=%t RestrictPublicBuckets=%t — drift from full lockdown",
					p.Bucket, p.BlockPublicAcls, p.IgnorePublicAcls, p.BlockPublicPolicy, p.RestrictPublicBuckets),
				Remediation: fmt.Sprintf("aws s3api put-public-access-block --bucket %s --public-access-block-configuration BlockPublicAcls=true,IgnorePublicAcls=true,BlockPublicPolicy=true,RestrictPublicBuckets=true", p.Bucket),
			})
		}
	}
	return probe.Result{
		Component: probe.ComponentResult{
			Component: s3Name,
			Status:    rollupStatus(findings),
			Detail:    fmt.Sprintf("%d bucket(s) inspected", len(pabs)),
		},
		Findings: findings,
	}
}
