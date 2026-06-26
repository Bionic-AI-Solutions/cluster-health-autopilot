// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	acmtypes "github.com/aws/aws-sdk-go-v2/service/acm/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	ekstypes "github.com/aws/aws-sdk-go-v2/service/eks/types"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	elbv2types "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	rdstypes "github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	smithy "github.com/aws/smithy-go"

	pkgaws "github.com/srenix-ai/agentic-sre/pkg/cloud/aws"
)

// LiveClient wraps aws-sdk-go-v2 with one service client per resource
// type Srenix probes. Auth flows through aws-sdk-go-v2's default
// credential chain: env vars → shared config → IRSA (Web Identity
// Token) → EC2/ECS instance metadata. For the in-cluster Srenix
// Deployment, IRSA is the default; the Helm chart annotates the
// ServiceAccount with the role-arn the operator configures.
type LiveClient struct {
	region string
	rds    *rds.Client
	ec2    *ec2.Client
	eks    *eks.Client
	iam    *iam.Client
	elbv2  *elasticloadbalancingv2.Client
	acm    *acm.Client
	kms    *kms.Client
	s3     *s3.Client
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
		ec2:    ec2.NewFromConfig(cfg),
		eks:    eks.NewFromConfig(cfg),
		iam:    iam.NewFromConfig(cfg),
		elbv2:  elasticloadbalancingv2.NewFromConfig(cfg),
		acm:    acm.NewFromConfig(cfg),
		kms:    kms.NewFromConfig(cfg),
		s3:     s3.NewFromConfig(cfg),
	}, nil
}

// Region returns the bound AWS region.
func (c *LiveClient) Region() string { return c.region }

// --- RDS ---------------------------------------------------------------

// DescribeDBInstances satisfies pkg/cloud/aws.Client.
func (c *LiveClient) DescribeDBInstances(ctx context.Context) ([]pkgaws.DBInstance, error) {
	var out []pkgaws.DBInstance
	var marker *string
	for {
		resp, err := c.rds.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{Marker: marker})
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

func mapDBInstance(db rdstypes.DBInstance) pkgaws.DBInstance {
	out := pkgaws.DBInstance{
		Identifier:            awssdk.ToString(db.DBInstanceIdentifier),
		Engine:                awssdk.ToString(db.Engine),
		Status:                awssdk.ToString(db.DBInstanceStatus),
		AllocatedStorageGB:    awssdk.ToInt32(db.AllocatedStorage),
		MultiAZ:               awssdk.ToBool(db.MultiAZ),
		ARN:                   awssdk.ToString(db.DBInstanceArn),
		BackupRetentionPeriod: int(awssdk.ToInt32(db.BackupRetentionPeriod)),
		ReadReplicaSourceDBInstanceIdentifier: awssdk.ToString(db.ReadReplicaSourceDBInstanceIdentifier),
	}
	if db.Endpoint != nil {
		out.Endpoint = fmt.Sprintf("%s:%d", awssdk.ToString(db.Endpoint.Address), awssdk.ToInt32(db.Endpoint.Port))
	}
	if db.InstanceCreateTime != nil {
		out.CreatedAt = *db.InstanceCreateTime
	}
	return out
}

// --- EBS ---------------------------------------------------------------

// DescribeVolumes satisfies pkg/cloud/aws.Client.
func (c *LiveClient) DescribeVolumes(ctx context.Context) ([]pkgaws.Volume, error) {
	var out []pkgaws.Volume
	var nextToken *string
	now := time.Now().UTC()
	for {
		resp, err := c.ec2.DescribeVolumes(ctx, &ec2.DescribeVolumesInput{NextToken: nextToken})
		if err != nil {
			return nil, fmt.Errorf("ec2.DescribeVolumes: %w", err)
		}
		for _, v := range resp.Volumes {
			vol := pkgaws.Volume{
				VolumeID:   awssdk.ToString(v.VolumeId),
				State:      string(v.State),
				SizeGB:     awssdk.ToInt32(v.Size),
				VolumeType: string(v.VolumeType),
			}
			if v.CreateTime != nil {
				vol.CreatedAt = *v.CreateTime
			}
			// First attachment wins (EBS volumes are 1:1 with EC2 instances).
			if len(v.Attachments) > 0 {
				vol.AttachedToEC2 = awssdk.ToString(v.Attachments[0].InstanceId)
				// Approximate detached duration as now - CreateTime when
				// state is "available" — we don't get DetachTime from SDK
				// reliably; the probe accepts this approximation.
			}
			if vol.AttachedToEC2 == "" && !vol.CreatedAt.IsZero() {
				vol.DetachedDuration = now.Sub(vol.CreatedAt)
			}
			out = append(out, vol)
		}
		if resp.NextToken == nil || awssdk.ToString(resp.NextToken) == "" {
			break
		}
		nextToken = resp.NextToken
	}
	return out, nil
}

// --- EKS ---------------------------------------------------------------

// DescribeEKSCluster satisfies pkg/cloud/aws.Client.
func (c *LiveClient) DescribeEKSCluster(ctx context.Context, name string) (*pkgaws.EKSCluster, error) {
	resp, err := c.eks.DescribeCluster(ctx, &eks.DescribeClusterInput{Name: awssdk.String(name)})
	if err != nil {
		var notFound *ekstypes.ResourceNotFoundException
		if errors.As(err, &notFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("eks.DescribeCluster: %w", err)
	}
	if resp.Cluster == nil {
		return nil, nil
	}
	cl := resp.Cluster
	out := &pkgaws.EKSCluster{
		Name:    awssdk.ToString(cl.Name),
		Status:  string(cl.Status),
		Version: awssdk.ToString(cl.Version),
		ARN:     awssdk.ToString(cl.Arn),
	}
	if cl.Endpoint != nil {
		out.Endpoint = awssdk.ToString(cl.Endpoint)
	}
	if cl.CreatedAt != nil {
		out.CreatedAt = *cl.CreatedAt
	}
	return out, nil
}

// ListEKSNodeGroups satisfies pkg/cloud/aws.Client.
func (c *LiveClient) ListEKSNodeGroups(ctx context.Context, clusterName string) ([]pkgaws.EKSNodeGroup, error) {
	var names []string
	var nextToken *string
	for {
		resp, err := c.eks.ListNodegroups(ctx, &eks.ListNodegroupsInput{
			ClusterName: awssdk.String(clusterName),
			NextToken:   nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("eks.ListNodegroups: %w", err)
		}
		names = append(names, resp.Nodegroups...)
		if resp.NextToken == nil || awssdk.ToString(resp.NextToken) == "" {
			break
		}
		nextToken = resp.NextToken
	}
	out := make([]pkgaws.EKSNodeGroup, 0, len(names))
	for _, n := range names {
		dn, err := c.eks.DescribeNodegroup(ctx, &eks.DescribeNodegroupInput{
			ClusterName:   awssdk.String(clusterName),
			NodegroupName: awssdk.String(n),
		})
		if err != nil {
			return nil, fmt.Errorf("eks.DescribeNodegroup %s: %w", n, err)
		}
		ng := dn.Nodegroup
		if ng == nil {
			continue
		}
		entry := pkgaws.EKSNodeGroup{
			ClusterName: clusterName,
			Name:        awssdk.ToString(ng.NodegroupName),
			Status:      string(ng.Status),
			Version:     awssdk.ToString(ng.Version),
			ARN:         awssdk.ToString(ng.NodegroupArn),
		}
		if ng.ScalingConfig != nil {
			entry.DesiredSize = awssdk.ToInt32(ng.ScalingConfig.DesiredSize)
			entry.MinSize = awssdk.ToInt32(ng.ScalingConfig.MinSize)
			entry.MaxSize = awssdk.ToInt32(ng.ScalingConfig.MaxSize)
		}
		if ng.Health != nil {
			for _, iss := range ng.Health.Issues {
				entry.HealthIssues = append(entry.HealthIssues, string(iss.Code))
			}
		}
		out = append(out, entry)
	}
	return out, nil
}

// --- IAM ---------------------------------------------------------------

// roleNameFromARN extracts the role name from a role ARN like
// "arn:aws:iam::123456789012:role/MyRole" → "MyRole". Returns input
// when it doesn't look like an ARN.
func roleNameFromARN(s string) string {
	if i := strings.LastIndex(s, "role/"); i >= 0 {
		return s[i+len("role/"):]
	}
	return s
}

// GetIAMRole satisfies pkg/cloud/aws.Client.
func (c *LiveClient) GetIAMRole(ctx context.Context, arnOrName string) (*pkgaws.IAMRole, error) {
	name := roleNameFromARN(arnOrName)
	resp, err := c.iam.GetRole(ctx, &iam.GetRoleInput{RoleName: awssdk.String(name)})
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && apiErr.ErrorCode() == "NoSuchEntity" {
			return &pkgaws.IAMRole{ARN: arnOrName, Name: name, Exists: false}, nil
		}
		var nse *iamtypes.NoSuchEntityException
		if errors.As(err, &nse) {
			return &pkgaws.IAMRole{ARN: arnOrName, Name: name, Exists: false}, nil
		}
		return nil, fmt.Errorf("iam.GetRole: %w", err)
	}
	role := resp.Role
	out := &pkgaws.IAMRole{
		ARN:    awssdk.ToString(role.Arn),
		Name:   awssdk.ToString(role.RoleName),
		Exists: true,
		HasTrustPolicy: role.AssumeRolePolicyDocument != nil &&
			awssdk.ToString(role.AssumeRolePolicyDocument) != "",
	}
	return out, nil
}

// --- ALB / NLB ---------------------------------------------------------

// DescribeALBTargetGroupsWithHealth satisfies pkg/cloud/aws.Client.
func (c *LiveClient) DescribeALBTargetGroupsWithHealth(ctx context.Context) ([]pkgaws.ALBTargetGroup, error) {
	var tgs []elbv2types.TargetGroup
	var marker *string
	for {
		resp, err := c.elbv2.DescribeTargetGroups(ctx, &elasticloadbalancingv2.DescribeTargetGroupsInput{Marker: marker})
		if err != nil {
			return nil, fmt.Errorf("elbv2.DescribeTargetGroups: %w", err)
		}
		tgs = append(tgs, resp.TargetGroups...)
		if resp.NextMarker == nil || awssdk.ToString(resp.NextMarker) == "" {
			break
		}
		marker = resp.NextMarker
	}
	// One DescribeLoadBalancers per probe cycle (NOT per target group)
	// builds the ARN → DNS-name map that feeds LoadBalancerDNS — the
	// Srenix Enterprise "(lb: ...)" RCA join key. Best-effort: a failure leaves
	// the map empty (message enrichment is never worth failing the
	// probe over) and the probe omits the suffix.
	dnsByARN := c.describeLoadBalancerDNS(ctx)
	out := make([]pkgaws.ALBTargetGroup, 0, len(tgs))
	for _, tg := range tgs {
		entry := pkgaws.ALBTargetGroup{
			ARN:             awssdk.ToString(tg.TargetGroupArn),
			Name:            awssdk.ToString(tg.TargetGroupName),
			Protocol:        string(tg.Protocol),
			Port:            awssdk.ToInt32(tg.Port),
			TargetType:      string(tg.TargetType),
			LoadBalancerDNS: firstLoadBalancerDNS(tg.LoadBalancerArns, dnsByARN),
		}
		health, err := c.elbv2.DescribeTargetHealth(ctx, &elasticloadbalancingv2.DescribeTargetHealthInput{
			TargetGroupArn: tg.TargetGroupArn,
		})
		if err != nil {
			return nil, fmt.Errorf("elbv2.DescribeTargetHealth %s: %w", awssdk.ToString(tg.TargetGroupArn), err)
		}
		for _, t := range health.TargetHealthDescriptions {
			if t.TargetHealth == nil {
				continue
			}
			switch t.TargetHealth.State {
			case elbv2types.TargetHealthStateEnumHealthy:
				entry.HealthyCount++
			case elbv2types.TargetHealthStateEnumUnhealthy, elbv2types.TargetHealthStateEnumDraining:
				entry.UnhealthyCount++
			case elbv2types.TargetHealthStateEnumUnused:
				entry.UnusedCount++
			case elbv2types.TargetHealthStateEnumInitial:
				entry.InitialCount++
			}
		}
		out = append(out, entry)
	}
	return out, nil
}

// describeLoadBalancerDNS lists every ALB/NLB once and returns the
// ARN → DNS-name map. Best-effort enrichment helper: returns nil
// (nil-map reads are safe; callers fall back to name / omit the suffix).
func (c *LiveClient) describeLoadBalancerDNS(ctx context.Context) map[string]string {
	var lbs []elbv2types.LoadBalancer
	var marker *string
	for {
		resp, err := c.elbv2.DescribeLoadBalancers(ctx, &elasticloadbalancingv2.DescribeLoadBalancersInput{Marker: marker})
		if err != nil {
			return nil
		}
		lbs = append(lbs, resp.LoadBalancers...)
		if resp.NextMarker == nil || awssdk.ToString(resp.NextMarker) == "" {
			break
		}
		marker = resp.NextMarker
	}
	return lbDNSByARN(lbs)
}

// lbDNSByARN maps load-balancer ARN → DNS name, skipping entries
// missing either side.
func lbDNSByARN(lbs []elbv2types.LoadBalancer) map[string]string {
	out := make(map[string]string, len(lbs))
	for _, lb := range lbs {
		arn := awssdk.ToString(lb.LoadBalancerArn)
		dns := awssdk.ToString(lb.DNSName)
		if arn == "" || dns == "" {
			continue
		}
		out[arn] = dns
	}
	return out
}

// firstLoadBalancerDNS resolves the first of a target group's
// LoadBalancerArns that has a known DNS name ("" when none do — the
// probe then omits the "(lb: ...)" join key).
func firstLoadBalancerDNS(arns []string, dnsByARN map[string]string) string {
	for _, arn := range arns {
		if dns := dnsByARN[arn]; dns != "" {
			return dns
		}
	}
	return ""
}

// --- ACM ---------------------------------------------------------------

// ListACMCertificates satisfies pkg/cloud/aws.Client.
func (c *LiveClient) ListACMCertificates(ctx context.Context) ([]pkgaws.ACMCertificate, error) {
	var summaries []acmtypes.CertificateSummary
	var nextToken *string
	for {
		resp, err := c.acm.ListCertificates(ctx, &acm.ListCertificatesInput{NextToken: nextToken})
		if err != nil {
			return nil, fmt.Errorf("acm.ListCertificates: %w", err)
		}
		summaries = append(summaries, resp.CertificateSummaryList...)
		if resp.NextToken == nil || awssdk.ToString(resp.NextToken) == "" {
			break
		}
		nextToken = resp.NextToken
	}
	out := make([]pkgaws.ACMCertificate, 0, len(summaries))
	for _, s := range summaries {
		entry := pkgaws.ACMCertificate{
			ARN:        awssdk.ToString(s.CertificateArn),
			DomainName: awssdk.ToString(s.DomainName),
			Status:     string(s.Status),
			Type:       string(s.Type),
		}
		if s.NotAfter != nil {
			entry.NotAfter = *s.NotAfter
		}
		out = append(out, entry)
	}
	return out, nil
}

// --- KMS ---------------------------------------------------------------

// ListKMSKeys satisfies pkg/cloud/aws.Client.
func (c *LiveClient) ListKMSKeys(ctx context.Context) ([]pkgaws.KMSKey, error) {
	var out []pkgaws.KMSKey
	var marker *string
	for {
		resp, err := c.kms.ListKeys(ctx, &kms.ListKeysInput{Marker: marker})
		if err != nil {
			return nil, fmt.Errorf("kms.ListKeys: %w", err)
		}
		for _, ks := range resp.Keys {
			dk, derr := c.kms.DescribeKey(ctx, &kms.DescribeKeyInput{KeyId: ks.KeyId})
			if derr != nil {
				return nil, fmt.Errorf("kms.DescribeKey %s: %w", awssdk.ToString(ks.KeyId), derr)
			}
			meta := dk.KeyMetadata
			if meta == nil {
				continue
			}
			entry := pkgaws.KMSKey{
				KeyID:       awssdk.ToString(meta.KeyId),
				ARN:         awssdk.ToString(meta.Arn),
				State:       string(meta.KeyState),
				Enabled:     meta.Enabled,
				Description: awssdk.ToString(meta.Description),
			}
			if meta.DeletionDate != nil {
				entry.DeletionDate = *meta.DeletionDate
			}
			out = append(out, entry)
		}
		if !resp.Truncated {
			break
		}
		marker = resp.NextMarker
	}
	return out, nil
}

// --- S3 ----------------------------------------------------------------

// ListS3BucketPAB satisfies pkg/cloud/aws.Client.
func (c *LiveClient) ListS3BucketPAB(ctx context.Context) ([]pkgaws.S3BucketPAB, error) {
	resp, err := c.s3.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, fmt.Errorf("s3.ListBuckets: %w", err)
	}
	out := make([]pkgaws.S3BucketPAB, 0, len(resp.Buckets))
	for _, b := range resp.Buckets {
		name := awssdk.ToString(b.Name)
		pab, perr := c.s3.GetPublicAccessBlock(ctx, &s3.GetPublicAccessBlockInput{Bucket: awssdk.String(name)})
		entry := pkgaws.S3BucketPAB{Bucket: name}
		if perr != nil {
			// Most common: NoSuchPublicAccessBlockConfiguration. That itself
			// is the drift signal (no PAB configured).
			var apiErr smithy.APIError
			if errors.As(perr, &apiErr) && apiErr.ErrorCode() == "NoSuchPublicAccessBlockConfiguration" {
				entry.HasPolicyError = true
				out = append(out, entry)
				continue
			}
			// Cross-region or denied — record as policy-error so the probe
			// surfaces it without blocking the whole probe.
			entry.HasPolicyError = true
			out = append(out, entry)
			continue
		}
		if pab.PublicAccessBlockConfiguration != nil {
			cfg := pab.PublicAccessBlockConfiguration
			entry.BlockPublicAcls = awssdk.ToBool(cfg.BlockPublicAcls)
			entry.IgnorePublicAcls = awssdk.ToBool(cfg.IgnorePublicAcls)
			entry.BlockPublicPolicy = awssdk.ToBool(cfg.BlockPublicPolicy)
			entry.RestrictPublicBuckets = awssdk.ToBool(cfg.RestrictPublicBuckets)
		}
		out = append(out, entry)
	}
	return out, nil
}

// --- EKS Addons --------------------------------------------------------

// ListEKSAddons satisfies pkg/cloud/aws.Client.
func (c *LiveClient) ListEKSAddons(ctx context.Context, clusterName string) ([]pkgaws.EKSAddon, error) {
	var addonNames []string
	var nextToken *string
	for {
		resp, err := c.eks.ListAddons(ctx, &eks.ListAddonsInput{
			ClusterName: awssdk.String(clusterName),
			NextToken:   nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("eks.ListAddons: %w", err)
		}
		addonNames = append(addonNames, resp.Addons...)
		if resp.NextToken == nil || awssdk.ToString(resp.NextToken) == "" {
			break
		}
		nextToken = resp.NextToken
	}

	out := make([]pkgaws.EKSAddon, 0, len(addonNames))
	for _, name := range addonNames {
		da, err := c.eks.DescribeAddon(ctx, &eks.DescribeAddonInput{
			ClusterName: awssdk.String(clusterName),
			AddonName:   awssdk.String(name),
		})
		if err != nil {
			return nil, fmt.Errorf("eks.DescribeAddon %s: %w", name, err)
		}
		if da.Addon == nil {
			continue
		}
		entry := pkgaws.EKSAddon{
			ClusterName:  clusterName,
			AddonName:    awssdk.ToString(da.Addon.AddonName),
			Status:       string(da.Addon.Status),
			AddonVersion: awssdk.ToString(da.Addon.AddonVersion),
			ARN:          awssdk.ToString(da.Addon.AddonArn),
		}
		// Fetch the latest available version for this addon (informational).
		ver, verr := c.eks.DescribeAddonVersions(ctx, &eks.DescribeAddonVersionsInput{
			AddonName: awssdk.String(name),
		})
		if verr == nil && len(ver.Addons) > 0 && len(ver.Addons[0].AddonVersions) > 0 {
			entry.MarketplaceVersion = awssdk.ToString(ver.Addons[0].AddonVersions[0].AddonVersion)
		}
		out = append(out, entry)
	}
	return out, nil
}

// --- VPC subnets -------------------------------------------------------

// DescribeSubnets satisfies pkg/cloud/aws.Client.
func (c *LiveClient) DescribeSubnets(ctx context.Context) ([]pkgaws.VPCSubnet, error) {
	var out []pkgaws.VPCSubnet
	var nextToken *string
	for {
		resp, err := c.ec2.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{NextToken: nextToken})
		if err != nil {
			return nil, fmt.Errorf("ec2.DescribeSubnets: %w", err)
		}
		for _, s := range resp.Subnets {
			entry := pkgaws.VPCSubnet{
				SubnetID:                  awssdk.ToString(s.SubnetId),
				VPCID:                     awssdk.ToString(s.VpcId),
				CIDRBlock:                 awssdk.ToString(s.CidrBlock),
				AvailabilityZone:          awssdk.ToString(s.AvailabilityZone),
				AvailableIPv4AddressCount: awssdk.ToInt32(s.AvailableIpAddressCount),
			}
			_ = ec2types.Subnet{} // silence unused-import lint when SDK signatures wiggle
			out = append(out, entry)
		}
		if resp.NextToken == nil || awssdk.ToString(resp.NextToken) == "" {
			break
		}
		nextToken = resp.NextToken
	}
	return out, nil
}
