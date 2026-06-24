// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

// Package azure is the Azure sub-client surface that cloud probes
// call. Intentionally narrow — only the read operations the
// M2-Sprint-1 probe set needs (Azure SQL Database, Managed Disks).
// Adding a new resource type should be a deliberate decision.
//
// The Client interface is implementation-agnostic. Three impls exist:
//   - Live (internal/cloud/azure/live.go) wraps azure-sdk-for-go ARM
//     clients + Azure Monitor (azquery for SQL DB storage_percent — PR
//     #104) + the AppGW BackendHealth LRO (PR #105) + a primary-range
//     subnet IP-usage count (PR #106). Shipped v1.7+; Monitor/LRO/IP
//     wiring shipped in v1.9.x.
//   - Snapshot (internal/cloud/azure/snapshot.go) replays captured JSON.
//   - Fake (in _test.go) returns canned responses.
//
// Probes never import azure-sdk-for-go directly.
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

	// ListAppGatewayBackends lists Application Gateway backend pools
	// with their health summary inlined. Returns (nil, nil) when
	// there are none.
	ListAppGatewayBackends(ctx context.Context) ([]AppGatewayBackend, error)

	// ListAppServiceCertificates lists App Service / managed
	// certificates in the bound subscription.
	ListAppServiceCertificates(ctx context.Context) ([]Certificate, error)

	// ListStorageAccounts lists Storage accounts with their
	// public-access posture.
	ListStorageAccounts(ctx context.Context) ([]StorageAccount, error)

	// ListKeyVaults lists Key Vaults with their soft-delete /
	// purge-protection posture.
	ListKeyVaults(ctx context.Context) ([]KeyVault, error)

	// ListKeyVaultItems calls the Key Vault data-plane API to list keys
	// and secrets within a vault. vaultURL is the vault's data-plane base
	// URL (e.g. "https://<name>.vault.azure.net"). Returns (nil, nil, nil)
	// when the vault has no items or RBAC is insufficient (callers should
	// skip silently on error).
	ListKeyVaultItems(ctx context.Context, vaultURL string) ([]KeyVaultKey, []KeyVaultSecret, error)
}
