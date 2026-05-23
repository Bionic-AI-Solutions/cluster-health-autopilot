// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

// Package aws is the AWS sub-client surface that cloud probes call.
// It is intentionally narrow — only the read operations the M1 probe
// set needs (RDS, EBS, EKS, IAM, ALB, ACM, KMS, S3, VPC). Adding a
// new resource type to the surface should be a deliberate decision,
// not an "I needed this from boto" reflex.
//
// The Client interface is implementation-agnostic — Live wraps
// aws-sdk-go-v2, Snapshot replays captured JSON, Fake (in _test.go)
// returns canned responses. Probes never import aws-sdk-go directly.
package aws

import "context"

// Client is the AWS sub-client surface. nil-return semantics:
// individual methods return (nil, nil) when the resource type is
// genuinely empty (e.g., no RDS instances in the account/region);
// they return (nil, err) when the API call failed. Probes distinguish.
//
// This interface is INTENTIONALLY incomplete — only the M1 RDS probe
// is wired today. Methods for the other M1 resource types (EBS, EKS,
// IAM, ALB, ACM, KMS, S3, VPC) land as their probes are implemented.
// Keep the surface narrow; add deliberately.
type Client interface {
	// Region returns the AWS region this client is bound to. Probes
	// use it to stamp DriftReport subjects like
	// "aws-rds/us-east-1/prod-db-1".
	Region() string

	// DescribeDBInstances lists all RDS DBInstances visible to the
	// caller in the bound region. Returns (nil, nil) when the
	// account has zero RDS instances; (nil, err) on API failure.
	DescribeDBInstances(ctx context.Context) ([]DBInstance, error)
}
