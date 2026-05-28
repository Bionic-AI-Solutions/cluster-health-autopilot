// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

// Package azure is the Azure sub-client surface that cloud probes
// call. Intentionally narrow — only the read operations the
// M2-Sprint-1 probe set needs (Azure SQL Database, Managed Disks).
// Adding a new resource type should be a deliberate decision.
//
// The Client interface is implementation-agnostic — a Live wrapper
// (deferred to a follow-up PR) will wrap azure-sdk-for-go, Snapshot
// will replay captured JSON, Fake (in _test.go) returns canned
// responses. Probes never import azure-sdk-for-go directly.
//
// Mirrors the shape of pkg/cloud/aws and pkg/cloud/gcp so probes
// share a mental model across providers.
package azure

import "context"

// Client is the Azure sub-client surface. nil-return semantics:
// individual methods return (nil, nil) when the resource type is
// genuinely empty; (nil, err) when the API call failed.
//
// All methods are READ-ONLY by design.
type Client interface {
	// SubscriptionID returns the Azure subscription this client is
	// bound to. Probes use it to stamp DriftReport subjects like
	// "azure-sql/<subscription>/<resourceGroup>/<server>/<db>".
	SubscriptionID() string

	// Location returns the Azure region this client is bound to.
	// Surface in subjects for diagnostic clarity.
	Location() string

	// ListSQLDatabases lists Azure SQL Database resources in the
	// bound subscription. Returns (nil, nil) when the subscription
	// has zero databases; (nil, err) on API failure.
	ListSQLDatabases(ctx context.Context) ([]SQLDatabase, error)

	// ListDisks lists Managed Disk resources in the bound
	// subscription. Returns (nil, nil) when there are none.
	ListDisks(ctx context.Context) ([]Disk, error)

	// GetAKSCluster fetches a single AKS managed cluster by name.
	// Returns (nil, nil) when it does not exist.
	GetAKSCluster(ctx context.Context, name string) (*AKSCluster, error)

	// ListAKSNodePools returns the agent pools for the named AKS
	// cluster. Returns (nil, nil) when the cluster has none.
	ListAKSNodePools(ctx context.Context, clusterName string) ([]AKSNodePool, error)

	// ListManagedIdentities lists user-assigned managed identities
	// in the bound subscription. Returns (nil, nil) when there are
	// none.
	ListManagedIdentities(ctx context.Context) ([]ManagedIdentity, error)

	// ListSubnets lists VNet subnets in the bound subscription.
	// Returns (nil, nil) when there are none.
	ListSubnets(ctx context.Context) ([]Subnet, error)
}
