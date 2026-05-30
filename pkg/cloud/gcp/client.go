// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

// Package gcp is the GCP sub-client surface that cloud probes call.
// Intentionally narrow — only the read operations the M2-Sprint-1
// probe set needs (Cloud SQL, Persistent Disk). Adding a new resource
// type to the surface should be a deliberate decision, not an
// "I needed this from gcloud" reflex.
//
// The Client interface is implementation-agnostic. Three impls exist:
//   - Live (internal/cloud/gcp/live.go) wraps google.golang.org/api/*
//     and the Cloud Monitoring v3 API. Cloud SQL DiskUsedPercent is
//     populated from the cloudsql.googleapis.com/database/disk/
//     utilization metric (shipped v1.9 / PR #103); other live signals
//     come from sqladmin / compute / container / kms / storage / iam
//     (shipped v1.7+).
//   - Snapshot (internal/cloud/gcp/snapshot.go) replays captured JSON.
//   - Fake (in _test.go) returns canned responses.
//
// Probes never import cloud.google.com/go directly.
//
// Mirrors the shape of pkg/cloud/aws so probes share a mental model
// across providers.
package gcp

import "context"

// Client is the GCP sub-client surface. nil-return semantics:
// individual methods return (nil, nil) when the resource type is
// genuinely empty (e.g., no Cloud SQL instances in the project); they
// return (nil, err) when the API call failed. Probes distinguish.
//
// All methods are READ-ONLY by design — cloud probes never mutate.
// Mutation lands in cloud M4 with its own approval-gated surface.
type Client interface {
	// Project returns the GCP project ID this client is bound to.
	// Probes use it to stamp DriftReport subjects like
	// "gcp-cloudsql/my-project/prod-db-1".
	Project() string

	// Region returns the GCP region this client is bound to.
	// Project-scoped global resources (IAM, GCS) ignore this; the
	// field is still surfaced in subjects so the operator sees the
	// bound-region context.
	Region() string

	// ListCloudSQLInstances lists Cloud SQL instances visible to the
	// caller in the bound project. Returns (nil, nil) when the
	// project has zero instances; (nil, err) on API failure.
	ListCloudSQLInstances(ctx context.Context) ([]CloudSQLInstance, error)

	// ListPersistentDisks lists Persistent Disk resources in the
	// bound project + region. Returns (nil, nil) when there are
	// none; (nil, err) on API failure.
	ListPersistentDisks(ctx context.Context) ([]PersistentDisk, error)

	// GetGKECluster fetches a single GKE cluster by name. Returns
	// (nil, nil) when the cluster does not exist (no panic).
	GetGKECluster(ctx context.Context, name string) (*GKECluster, error)

	// ListGKENodePools returns all node pools for the named GKE
	// cluster. Returns (nil, nil) when the cluster has none.
	ListGKENodePools(ctx context.Context, clusterName string) ([]GKENodePool, error)

	// ListServiceAccounts lists IAM service accounts in the bound
	// project. Returns (nil, nil) when there are none.
	ListServiceAccounts(ctx context.Context) ([]ServiceAccount, error)

	// ListSubnets lists VPC subnetworks in the bound project +
	// region. Returns (nil, nil) when there are none.
	ListSubnets(ctx context.Context) ([]Subnet, error)

	// ListBackendServices lists Load Balancer backend services with
	// their backend-health summary inlined. Returns (nil, nil) when
	// there are none.
	ListBackendServices(ctx context.Context) ([]BackendService, error)

	// ListManagedCertificates lists Google-managed SSL certificates
	// in the bound project. Returns (nil, nil) when there are none.
	ListManagedCertificates(ctx context.Context) ([]ManagedCertificate, error)

	// ListBuckets lists GCS buckets with their public-access posture.
	// Returns (nil, nil) when there are none.
	ListBuckets(ctx context.Context) ([]Bucket, error)

	// ListKMSKeys lists Cloud KMS crypto keys in the bound project.
	// Returns (nil, nil) when there are none.
	ListKMSKeys(ctx context.Context) ([]KMSKey, error)
}
