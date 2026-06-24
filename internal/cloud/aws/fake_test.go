// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/cloud"
	pkgaws "github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/cloud/aws"
	pkgazure "github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/cloud/azure"
	pkggcp "github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/cloud/gcp"
)

// fakeAWS implements pkgaws.Client for unit tests. Tests inject the
// per-resource list (and optional per-resource error) directly. Methods
// that take input (DescribeEKSCluster name, GetIAMRole arn) match
// against the injected list.
type fakeAWS struct {
	region string

	instances    []pkgaws.DBInstance
	instancesErr error

	volumes    []pkgaws.Volume
	volumesErr error

	clusters    []pkgaws.EKSCluster
	clustersErr error

	nodeGroups    []pkgaws.EKSNodeGroup
	nodeGroupsErr error

	roles    []pkgaws.IAMRole
	rolesErr error

	targetGroups    []pkgaws.ALBTargetGroup
	targetGroupsErr error

	certs    []pkgaws.ACMCertificate
	certsErr error

	keys    []pkgaws.KMSKey
	keysErr error

	bucketPAB    []pkgaws.S3BucketPAB
	bucketPABErr error

	subnets    []pkgaws.VPCSubnet
	subnetsErr error

	addons    []pkgaws.EKSAddon
	addonsErr error
}

func (f *fakeAWS) Region() string { return f.region }

func (f *fakeAWS) DescribeDBInstances(_ context.Context) ([]pkgaws.DBInstance, error) {
	return f.instances, f.instancesErr
}

func (f *fakeAWS) DescribeVolumes(_ context.Context) ([]pkgaws.Volume, error) {
	return f.volumes, f.volumesErr
}

func (f *fakeAWS) DescribeEKSCluster(_ context.Context, name string) (*pkgaws.EKSCluster, error) {
	if f.clustersErr != nil {
		return nil, f.clustersErr
	}
	for i := range f.clusters {
		if f.clusters[i].Name == name {
			return &f.clusters[i], nil
		}
	}
	return nil, nil
}

func (f *fakeAWS) ListEKSNodeGroups(_ context.Context, clusterName string) ([]pkgaws.EKSNodeGroup, error) {
	if f.nodeGroupsErr != nil {
		return nil, f.nodeGroupsErr
	}
	out := make([]pkgaws.EKSNodeGroup, 0, len(f.nodeGroups))
	for _, ng := range f.nodeGroups {
		if ng.ClusterName == clusterName {
			out = append(out, ng)
		}
	}
	return out, nil
}

func (f *fakeAWS) GetIAMRole(_ context.Context, arnOrName string) (*pkgaws.IAMRole, error) {
	if f.rolesErr != nil {
		return nil, f.rolesErr
	}
	for i := range f.roles {
		if f.roles[i].ARN == arnOrName || f.roles[i].Name == arnOrName {
			return &f.roles[i], nil
		}
	}
	return &pkgaws.IAMRole{ARN: arnOrName, Exists: false}, nil
}

func (f *fakeAWS) DescribeALBTargetGroupsWithHealth(_ context.Context) ([]pkgaws.ALBTargetGroup, error) {
	return f.targetGroups, f.targetGroupsErr
}

func (f *fakeAWS) ListACMCertificates(_ context.Context) ([]pkgaws.ACMCertificate, error) {
	return f.certs, f.certsErr
}

func (f *fakeAWS) ListKMSKeys(_ context.Context) ([]pkgaws.KMSKey, error) {
	return f.keys, f.keysErr
}

func (f *fakeAWS) ListS3BucketPAB(_ context.Context) ([]pkgaws.S3BucketPAB, error) {
	return f.bucketPAB, f.bucketPABErr
}

func (f *fakeAWS) DescribeSubnets(_ context.Context) ([]pkgaws.VPCSubnet, error) {
	return f.subnets, f.subnetsErr
}

func (f *fakeAWS) ListEKSAddons(_ context.Context, clusterName string) ([]pkgaws.EKSAddon, error) {
	if f.addonsErr != nil {
		return nil, f.addonsErr
	}
	out := make([]pkgaws.EKSAddon, 0, len(f.addons))
	for _, a := range f.addons {
		if a.ClusterName == clusterName {
			out = append(out, a)
		}
	}
	return out, nil
}

// fakeSource implements cloud.Source for unit tests. AWS is settable;
// other providers default to nil (matches the M1 "AWS-only" test case).
type fakeSource struct {
	aws   pkgaws.Client
	gcp   pkggcp.Client
	azure pkgazure.Client
	mode  cloud.Mode
}

func (f *fakeSource) AWS() pkgaws.Client     { return f.aws }
func (f *fakeSource) GCP() pkggcp.Client     { return f.gcp }
func (f *fakeSource) Azure() pkgazure.Client { return f.azure }
func (f *fakeSource) Mode() cloud.Mode       { return f.mode }
