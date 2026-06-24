// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	pkgaws "github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/cloud/aws"
)

// SnapshotClient replays cloud-resource state captured to disk by
// `cha snapshot capture --include-cloud`. Read-only by construction
// — there is no mutation API.
//
// Captured layout (one file per resource type):
//
//	<snapshot-dir>/cloud/aws/
//	  rds.json     ← []pkgaws.DBInstance
//	  ebs.json     ← []pkgaws.Volume      (M1 follow-up)
//	  eks.json     ← []pkgaws.Cluster     (M1 follow-up)
//	  ...
//
// Each file is a JSON array of the corresponding type. Missing files
// are treated as "no resources of that type" (probes return HEALTHY).
type SnapshotClient struct {
	dir    string // <snapshot-dir>/cloud/aws
	region string // recorded at capture time
}

// NewSnapshotClient constructs a snapshot-backed AWS client rooted at
// the snapshot directory's cloud/aws subdir.
func NewSnapshotClient(snapshotDir, region string) *SnapshotClient {
	return &SnapshotClient{
		dir:    filepath.Join(snapshotDir, "cloud", "aws"),
		region: region,
	}
}

// Region returns the region recorded at capture time.
func (c *SnapshotClient) Region() string { return c.region }

// Per-resource readers. Each is a one-liner over the generic readJSON
// helper. Snapshot file convention: <snapshot-dir>/cloud/aws/<name>.json
// where <name> matches the resource (rds, ebs, eks, eks-nodegroups,
// alb, acm, kms, s3-pab, vpc-subnets). IAM is special: per-role lookup
// keyed by ARN under iam-roles.json.

// DescribeDBInstances satisfies pkg/cloud/aws.Client.
func (c *SnapshotClient) DescribeDBInstances(_ context.Context) ([]pkgaws.DBInstance, error) {
	return readJSON[pkgaws.DBInstance](c.dir, "rds.json")
}

// DescribeVolumes satisfies pkg/cloud/aws.Client.
func (c *SnapshotClient) DescribeVolumes(_ context.Context) ([]pkgaws.Volume, error) {
	return readJSON[pkgaws.Volume](c.dir, "ebs.json")
}

// DescribeEKSCluster returns the cluster matching name from
// eks-clusters.json, or (nil, nil) when not present.
func (c *SnapshotClient) DescribeEKSCluster(_ context.Context, name string) (*pkgaws.EKSCluster, error) {
	clusters, err := readJSON[pkgaws.EKSCluster](c.dir, "eks-clusters.json")
	if err != nil {
		return nil, err
	}
	for i := range clusters {
		if clusters[i].Name == name {
			return &clusters[i], nil
		}
	}
	return nil, nil
}

// ListEKSNodeGroups returns node groups for the named cluster from
// eks-nodegroups.json.
func (c *SnapshotClient) ListEKSNodeGroups(_ context.Context, clusterName string) ([]pkgaws.EKSNodeGroup, error) {
	all, err := readJSON[pkgaws.EKSNodeGroup](c.dir, "eks-nodegroups.json")
	if err != nil {
		return nil, err
	}
	out := make([]pkgaws.EKSNodeGroup, 0, len(all))
	for _, ng := range all {
		if ng.ClusterName == clusterName {
			out = append(out, ng)
		}
	}
	return out, nil
}

// GetIAMRole looks up a role from iam-roles.json by ARN match. Missing
// role yields (&IAMRole{Exists:false, ARN: arnOrName}, nil).
func (c *SnapshotClient) GetIAMRole(_ context.Context, arnOrName string) (*pkgaws.IAMRole, error) {
	roles, err := readJSON[pkgaws.IAMRole](c.dir, "iam-roles.json")
	if err != nil {
		return nil, err
	}
	for i := range roles {
		if roles[i].ARN == arnOrName || roles[i].Name == arnOrName {
			return &roles[i], nil
		}
	}
	return &pkgaws.IAMRole{ARN: arnOrName, Exists: false}, nil
}

// DescribeALBTargetGroupsWithHealth satisfies pkg/cloud/aws.Client.
func (c *SnapshotClient) DescribeALBTargetGroupsWithHealth(_ context.Context) ([]pkgaws.ALBTargetGroup, error) {
	return readJSON[pkgaws.ALBTargetGroup](c.dir, "alb.json")
}

// ListACMCertificates satisfies pkg/cloud/aws.Client.
func (c *SnapshotClient) ListACMCertificates(_ context.Context) ([]pkgaws.ACMCertificate, error) {
	return readJSON[pkgaws.ACMCertificate](c.dir, "acm.json")
}

// ListKMSKeys satisfies pkg/cloud/aws.Client.
func (c *SnapshotClient) ListKMSKeys(_ context.Context) ([]pkgaws.KMSKey, error) {
	return readJSON[pkgaws.KMSKey](c.dir, "kms.json")
}

// ListS3BucketPAB satisfies pkg/cloud/aws.Client.
func (c *SnapshotClient) ListS3BucketPAB(_ context.Context) ([]pkgaws.S3BucketPAB, error) {
	return readJSON[pkgaws.S3BucketPAB](c.dir, "s3-pab.json")
}

// DescribeSubnets satisfies pkg/cloud/aws.Client.
func (c *SnapshotClient) DescribeSubnets(_ context.Context) ([]pkgaws.VPCSubnet, error) {
	return readJSON[pkgaws.VPCSubnet](c.dir, "vpc-subnets.json")
}

// ListEKSAddons returns addons for the named cluster from
// eks-addons.json.
func (c *SnapshotClient) ListEKSAddons(_ context.Context, clusterName string) ([]pkgaws.EKSAddon, error) {
	all, err := readJSON[pkgaws.EKSAddon](c.dir, "eks-addons.json")
	if err != nil {
		return nil, err
	}
	out := make([]pkgaws.EKSAddon, 0, len(all))
	for _, a := range all {
		if a.ClusterName == clusterName {
			out = append(out, a)
		}
	}
	return out, nil
}

// readJSON is a small generic helper so additional Describe* methods
// can be one-liners as more probes land.
func readJSON[T any](dir, file string) ([]T, error) {
	path := filepath.Join(dir, file)
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var out []T
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	return out, nil
}
