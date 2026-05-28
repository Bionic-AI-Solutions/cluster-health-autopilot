// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/cloud"
	pkgaws "github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/cloud/aws"
	pkgazure "github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/cloud/azure"
	pkggcp "github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/cloud/gcp"
)

type fakeAzure struct {
	subscription string
	location     string

	dbs    []pkgazure.SQLDatabase
	dbsErr error

	disks    []pkgazure.Disk
	disksErr error

	cluster    *pkgazure.AKSCluster
	clusterErr error

	nodePools    []pkgazure.AKSNodePool
	nodePoolsErr error

	identities    []pkgazure.ManagedIdentity
	identitiesErr error

	subnets    []pkgazure.Subnet
	subnetsErr error
}

func (f *fakeAzure) SubscriptionID() string { return f.subscription }
func (f *fakeAzure) Location() string       { return f.location }

func (f *fakeAzure) ListSQLDatabases(_ context.Context) ([]pkgazure.SQLDatabase, error) {
	return f.dbs, f.dbsErr
}

func (f *fakeAzure) ListDisks(_ context.Context) ([]pkgazure.Disk, error) {
	return f.disks, f.disksErr
}

func (f *fakeAzure) GetAKSCluster(_ context.Context, _ string) (*pkgazure.AKSCluster, error) {
	return f.cluster, f.clusterErr
}

func (f *fakeAzure) ListAKSNodePools(_ context.Context, _ string) ([]pkgazure.AKSNodePool, error) {
	return f.nodePools, f.nodePoolsErr
}

func (f *fakeAzure) ListManagedIdentities(_ context.Context) ([]pkgazure.ManagedIdentity, error) {
	return f.identities, f.identitiesErr
}

func (f *fakeAzure) ListSubnets(_ context.Context) ([]pkgazure.Subnet, error) {
	return f.subnets, f.subnetsErr
}

type fakeSource struct {
	azure pkgazure.Client
	aws   pkgaws.Client
	gcp   pkggcp.Client
	mode  cloud.Mode
}

func (f *fakeSource) AWS() pkgaws.Client     { return f.aws }
func (f *fakeSource) GCP() pkggcp.Client     { return f.gcp }
func (f *fakeSource) Azure() pkgazure.Client { return f.azure }
func (f *fakeSource) Mode() cloud.Mode       { return f.mode }
