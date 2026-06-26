// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package gcp

import (
	"context"

	"github.com/srenix-ai/agentic-sre/pkg/cloud"
	pkgaws "github.com/srenix-ai/agentic-sre/pkg/cloud/aws"
	pkgazure "github.com/srenix-ai/agentic-sre/pkg/cloud/azure"
	pkggcp "github.com/srenix-ai/agentic-sre/pkg/cloud/gcp"
)

// fakeGCP implements pkggcp.Client for unit tests. Tests inject the
// per-resource list (and optional per-resource error) directly.
type fakeGCP struct {
	project string
	region  string

	instances    []pkggcp.CloudSQLInstance
	instancesErr error

	disks    []pkggcp.PersistentDisk
	disksErr error

	cluster    *pkggcp.GKECluster
	clusterErr error

	nodePools    []pkggcp.GKENodePool
	nodePoolsErr error

	serviceAccounts    []pkggcp.ServiceAccount
	serviceAccountsErr error

	subnets    []pkggcp.Subnet
	subnetsErr error

	backends    []pkggcp.BackendService
	backendsErr error

	certs    []pkggcp.ManagedCertificate
	certsErr error

	buckets    []pkggcp.Bucket
	bucketsErr error

	kmsKeys    []pkggcp.KMSKey
	kmsKeysErr error
}

func (f *fakeGCP) Project() string { return f.project }
func (f *fakeGCP) Region() string  { return f.region }

func (f *fakeGCP) ListCloudSQLInstances(_ context.Context) ([]pkggcp.CloudSQLInstance, error) {
	return f.instances, f.instancesErr
}

func (f *fakeGCP) ListPersistentDisks(_ context.Context) ([]pkggcp.PersistentDisk, error) {
	return f.disks, f.disksErr
}

func (f *fakeGCP) GetGKECluster(_ context.Context, _ string) (*pkggcp.GKECluster, error) {
	return f.cluster, f.clusterErr
}

func (f *fakeGCP) ListGKENodePools(_ context.Context, _ string) ([]pkggcp.GKENodePool, error) {
	return f.nodePools, f.nodePoolsErr
}

func (f *fakeGCP) ListServiceAccounts(_ context.Context) ([]pkggcp.ServiceAccount, error) {
	return f.serviceAccounts, f.serviceAccountsErr
}

func (f *fakeGCP) ListSubnets(_ context.Context) ([]pkggcp.Subnet, error) {
	return f.subnets, f.subnetsErr
}

func (f *fakeGCP) ListBackendServices(_ context.Context) ([]pkggcp.BackendService, error) {
	return f.backends, f.backendsErr
}

func (f *fakeGCP) ListManagedCertificates(_ context.Context) ([]pkggcp.ManagedCertificate, error) {
	return f.certs, f.certsErr
}

func (f *fakeGCP) ListBuckets(_ context.Context) ([]pkggcp.Bucket, error) {
	return f.buckets, f.bucketsErr
}

func (f *fakeGCP) ListKMSKeys(_ context.Context) ([]pkggcp.KMSKey, error) {
	return f.kmsKeys, f.kmsKeysErr
}

// fakeSource implements cloud.Source for unit tests. GCP is settable;
// other providers default to nil.
type fakeSource struct {
	gcp   pkggcp.Client
	aws   pkgaws.Client
	azure pkgazure.Client
	mode  cloud.Mode
}

func (f *fakeSource) AWS() pkgaws.Client     { return f.aws }
func (f *fakeSource) GCP() pkggcp.Client     { return f.gcp }
func (f *fakeSource) Azure() pkgazure.Client { return f.azure }
func (f *fakeSource) Mode() cloud.Mode       { return f.mode }
