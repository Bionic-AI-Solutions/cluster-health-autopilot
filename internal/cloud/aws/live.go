// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"fmt"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	rdstypes "github.com/aws/aws-sdk-go-v2/service/rds/types"

	pkgaws "github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/cloud/aws"
)

// LiveClient wraps aws-sdk-go-v2 for the M1 RDS probe. As more probes
// land (EBS, EKS, IAM, ALB, ACM, KMS, S3, VPC) this struct grows one
// per-service field per resource type; we deliberately keep the
// service clients separate rather than hide them behind a generic
// "AWS service" facade.
//
// Auth is picked up from aws-sdk-go-v2's default credential chain:
// env vars → shared config → IRSA (Web Identity Token) → EC2/ECS
// instance metadata. For the in-cluster CHA Deployment, IRSA is the
// default; the Helm chart annotates the ServiceAccount with the
// role-arn the operator configures.
type LiveClient struct {
	region string
	rds    *rds.Client
}

// NewLiveClient constructs a Live AWS client bound to the given
// region. cfgOpts are forwarded to LoadDefaultConfig so callers can
// inject custom endpoint resolvers (e.g. LocalStack in tests) or
// retry policies.
func NewLiveClient(ctx context.Context, region string, cfgOpts ...func(*awsconfig.LoadOptions) error) (*LiveClient, error) {
	if region == "" {
		return nil, fmt.Errorf("aws: region is required")
	}
	opts := append([]func(*awsconfig.LoadOptions) error{awsconfig.WithRegion(region)}, cfgOpts...)
	cfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("aws: load config: %w", err)
	}
	return &LiveClient{
		region: region,
		rds:    rds.NewFromConfig(cfg),
	}, nil
}

// Region returns the bound AWS region.
func (c *LiveClient) Region() string { return c.region }

// DescribeDBInstances lists all RDS DBInstances in the bound region.
// Paginates over the SDK Marker so we don't truncate on accounts with
// many instances.
//
// NOTE: StorageUsedPercent is left at 0 in M1 — populating it
// requires a CloudWatch GetMetricStatistics call per instance for the
// FreeStorageSpace metric. The RDS probe handles 0 as "unknown" and
// emits no storage finding. CloudWatch wiring lands in M1 follow-up.
func (c *LiveClient) DescribeDBInstances(ctx context.Context) ([]pkgaws.DBInstance, error) {
	var out []pkgaws.DBInstance
	var marker *string
	for {
		resp, err := c.rds.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{
			Marker: marker,
		})
		if err != nil {
			return nil, fmt.Errorf("rds.DescribeDBInstances: %w", err)
		}
		for _, db := range resp.DBInstances {
			out = append(out, mapDBInstance(db))
		}
		if resp.Marker == nil || awssdk.ToString(resp.Marker) == "" {
			break
		}
		marker = resp.Marker
	}
	return out, nil
}

// mapDBInstance projects the SDK type onto our narrow DBInstance.
// Kept here (not in pkg/cloud/aws) so pkg/cloud/aws stays free of
// aws-sdk-go-v2 imports for downstream consumers that only handle
// our type.
func mapDBInstance(db rdstypes.DBInstance) pkgaws.DBInstance {
	out := pkgaws.DBInstance{
		Identifier:         awssdk.ToString(db.DBInstanceIdentifier),
		Engine:             awssdk.ToString(db.Engine),
		Status:             awssdk.ToString(db.DBInstanceStatus),
		AllocatedStorageGB: awssdk.ToInt32(db.AllocatedStorage),
		MultiAZ:            awssdk.ToBool(db.MultiAZ),
		ARN:                awssdk.ToString(db.DBInstanceArn),
	}
	if db.Endpoint != nil {
		out.Endpoint = fmt.Sprintf("%s:%d", awssdk.ToString(db.Endpoint.Address), awssdk.ToInt32(db.Endpoint.Port))
	}
	if db.InstanceCreateTime != nil {
		out.CreatedAt = *db.InstanceCreateTime
	}
	return out
}
