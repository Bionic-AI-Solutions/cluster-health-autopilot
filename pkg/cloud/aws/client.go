// Copyright 2026 Agentic SRE contributors
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
// All methods are READ-ONLY by design — cloud probes never mutate.
// Mutation lands in cloud M4 with its own approval-gated surface.
type Client interface {
	// Region returns the AWS region this client is bound to. Probes
	// use it to stamp DriftReport subjects like
	// "aws-rds/us-east-1/prod-db-1".
	Region() string

	// DescribeDBInstances lists all RDS DBInstances visible to the
	// caller in the bound region. Returns (nil, nil) when the
	// account has zero RDS instances; (nil, err) on API failure.
	DescribeDBInstances(ctx context.Context) ([]DBInstance, error)

	// DescribeVolumes lists EBS volumes in the bound region.
	DescribeVolumes(ctx context.Context) ([]Volume, error)

	// DescribeEKSCluster fetches a single EKS cluster by name.
	// Returns (nil, nil) when the cluster does not exist (no panic).
	DescribeEKSCluster(ctx context.Context, name string) (*EKSCluster, error)

	// ListEKSNodeGroups returns all node groups for the named cluster.
	ListEKSNodeGroups(ctx context.Context, clusterName string) ([]EKSNodeGroup, error)

	// GetIAMRole looks up a single IAM role by ARN or name. When the
	// role does not exist returns (&IAMRole{Exists:false, ARN: arnOrName}, nil)
	// — error is reserved for transport/auth failures.
	GetIAMRole(ctx context.Context, arnOrName string) (*IAMRole, error)

	// DescribeALBTargetGroupsWithHealth returns every ALB/NLB target
	// group in the bound region with its current target-health summary
	// inlined. Avoids the per-target-group fan-out the SDK would
	// otherwise force on the probe.
	DescribeALBTargetGroupsWithHealth(ctx context.Context) ([]ALBTargetGroup, error)

	// ListACMCertificates lists ACM-managed certificates in the bound
	// region. NotAfter is populated when the SDK returns it.
	ListACMCertificates(ctx context.Context) ([]ACMCertificate, error)

	// ListKMSKeys lists Customer Master Keys in the bound region.
	// State is populated via the second-call DescribeKey pattern.
	ListKMSKeys(ctx context.Context) ([]KMSKey, error)

	// ListS3BucketPAB returns the public-access-block configuration for
	// every S3 bucket in the account (S3 is global; the bound region
	// is irrelevant for ListBuckets). Buckets without an explicit PAB
	// configuration have HasPolicyError=true.
	ListS3BucketPAB(ctx context.Context) ([]S3BucketPAB, error)

	// DescribeSubnets lists all VPC subnets in the bound region.
	DescribeSubnets(ctx context.Context) ([]VPCSubnet, error)

	// ListEKSAddons lists all managed add-ons for the named cluster,
	// including their version and status.
	ListEKSAddons(ctx context.Context, clusterName string) ([]EKSAddon, error)
}
