// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/monitor/azquery"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appservice/armappservice/v4"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v6"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/keyvault/armkeyvault"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sql/armsql"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"

	pkgazure "github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/cloud/azure"
)

// LiveClient implements pkgazure.Client against azure-sdk-for-go.
// Auth flows through DefaultAzureCredential: in-cluster that's AAD
// Workload Identity (the Helm chart annotates the KSA + projects the
// token); locally it's `az login` / env-var service principals.
//
// All calls are READ-ONLY. Clients are constructed once in
// NewLiveClient and reused.
//
// NOTE ON VERIFICATION: this wrapper compiles against the real
// azure-sdk-for-go ARM surface (so API usage is correct), but it is
// NOT integration-tested against a live Azure subscription in CI —
// that requires credentials. The probe evaluation logic it feeds is
// unit-tested independently (see *_probe_test.go).
type LiveClient struct {
	subscription string
	location     string

	// cred is kept for data-plane API calls (e.g. Key Vault keys/secrets
	// endpoint) that require bearer token auth but are not exposed by
	// any ARM SDK client.
	cred azcore.TokenCredential

	sqlServers  *armsql.ServersClient
	sqlDBs      *armsql.DatabasesClient
	disks       *armcompute.DisksClient
	aks         *armcontainerservice.ManagedClustersClient
	aksPools    *armcontainerservice.AgentPoolsClient
	identities  *armmsi.UserAssignedIdentitiesClient
	roleAssigns *armauthorization.RoleAssignmentsClient
	vnets       *armnetwork.VirtualNetworksClient
	subnets     *armnetwork.SubnetsClient
	appgw       *armnetwork.ApplicationGatewaysClient
	certs       *armappservice.CertificatesClient
	storage     *armstorage.AccountsClient
	vaults      *armkeyvault.VaultsClient

	// monitor populates time-series-only signals (Azure SQL storage_percent
	// etc.) that the ARM list APIs don't carry directly. nil is acceptable
	// — callers fall back to the "not measured" sentinel (UsedPercent=-1),
	// preserving the v1.8.x posture for partial credential grants.
	monitor monitoringQuerier

	// appgwHealth wraps the AppGW BackendHealth LRO so ListAppGatewayBackends
	// can populate HealthyCount instead of leaving it at -1. nil → fall
	// back to the "not measured" sentinel.
	appgwHealth backendHealthClient
}

// NewLiveClient constructs a Live Azure client bound to a subscription
// (+ optional location for subject context). One ARM client per
// resource family, all using DefaultAzureCredential.
func NewLiveClient(ctx context.Context, subscription, location string) (*LiveClient, error) {
	if subscription == "" {
		return nil, fmt.Errorf("azure: subscription ID is required")
	}
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("azure: default credential: %w", err)
	}

	sqlServers, err := armsql.NewServersClient(subscription, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("azure: sql servers: %w", err)
	}
	sqlDBs, err := armsql.NewDatabasesClient(subscription, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("azure: sql databases: %w", err)
	}
	disks, err := armcompute.NewDisksClient(subscription, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("azure: disks: %w", err)
	}
	aks, err := armcontainerservice.NewManagedClustersClient(subscription, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("azure: aks: %w", err)
	}
	aksPools, err := armcontainerservice.NewAgentPoolsClient(subscription, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("azure: aks agent pools: %w", err)
	}
	identities, err := armmsi.NewUserAssignedIdentitiesClient(subscription, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("azure: managed identities: %w", err)
	}
	roleAssigns, err := armauthorization.NewRoleAssignmentsClient(subscription, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("azure: role assignments: %w", err)
	}
	vnets, err := armnetwork.NewVirtualNetworksClient(subscription, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("azure: vnets: %w", err)
	}
	subnets, err := armnetwork.NewSubnetsClient(subscription, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("azure: subnets: %w", err)
	}
	appgw, err := armnetwork.NewApplicationGatewaysClient(subscription, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("azure: app gateways: %w", err)
	}
	certs, err := armappservice.NewCertificatesClient(subscription, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("azure: certificates: %w", err)
	}
	storageC, err := armstorage.NewAccountsClient(subscription, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("azure: storage: %w", err)
	}
	vaults, err := armkeyvault.NewVaultsClient(subscription, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("azure: key vaults: %w", err)
	}

	// Azure Monitor metrics client is best-effort: if NewMetricsClient
	// fails (no Monitoring Reader role, network blocked, etc.) we proceed
	// without it and fall back to the "not measured" sentinel for time-
	// series-only signals. Preserves install-success on partial creds.
	var monitor monitoringQuerier
	if mc, mErr := azquery.NewMetricsClient(cred, nil); mErr == nil {
		monitor = newAzureMonitoringQuerier(mc)
	}

	return &LiveClient{
		subscription: subscription,
		location:     location,
		cred:         cred,
		sqlServers:   sqlServers,
		sqlDBs:       sqlDBs,
		disks:        disks,
		aks:          aks,
		aksPools:     aksPools,
		identities:   identities,
		roleAssigns:  roleAssigns,
		vnets:        vnets,
		subnets:      subnets,
		appgw:        appgw,
		certs:        certs,
		storage:      storageC,
		vaults:       vaults,
		monitor:      monitor,
		appgwHealth:  newLiveBackendHealthClient(appgw),
	}, nil
}

// SubscriptionID satisfies pkgazure.Client.
func (l *LiveClient) SubscriptionID() string { return l.subscription }

// Location satisfies pkgazure.Client.
func (l *LiveClient) Location() string { return l.location }

// ListSQLDatabases satisfies pkgazure.Client. SQL databases are nested
// under servers, so we list servers first then databases per server.
func (l *LiveClient) ListSQLDatabases(ctx context.Context) ([]pkgazure.SQLDatabase, error) {
	var out []pkgazure.SQLDatabase
	serverPager := l.sqlServers.NewListPager(nil)
	for serverPager.More() {
		page, err := serverPager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, srv := range page.Value {
			if srv == nil || srv.Name == nil {
				continue
			}
			rg := resourceGroupFromID(deref(srv.ID))
			dbPager := l.sqlDBs.NewListByServerPager(rg, *srv.Name, nil)
			for dbPager.More() {
				dbPage, err := dbPager.NextPage(ctx)
				if err != nil {
					return nil, err
				}
				for _, db := range dbPage.Value {
					if db == nil || db.Name == nil {
						continue
					}
					rec := pkgazure.SQLDatabase{
						Name:          *db.Name,
						ResourceGroup: rg,
						Server:        *srv.Name,
					}
					if db.Properties != nil {
						rec.Status = derefStatus(db.Properties.Status)
						if db.Properties.CurrentServiceObjectiveName != nil {
							rec.Tier = *db.Properties.CurrentServiceObjectiveName
						}
						if db.Properties.MaxSizeBytes != nil {
							rec.MaxSizeGB = *db.Properties.MaxSizeBytes / (1024 * 1024 * 1024)
						}
						if db.Properties.ZoneRedundant != nil {
							rec.ZoneRedundant = *db.Properties.ZoneRedundant
						}
					}
					// UsedPercent is not on the Database resource — it
					// requires Azure Monitor metrics (storage_percent).
					// G9: queried here when the Monitor client is wired.
					// The querier returns -1 when no recent point exists
					// (lag / fresh DB / Monitoring Reader role missing)
					// — the probe then skips the storage check rather
					// than treating it as 0%.
					rec.UsedPercent = -1
					if l.monitor != nil && db.ID != nil {
						if pct, mErr := l.monitor.SQLDatabaseStoragePercent(ctx, *db.ID); mErr == nil {
							rec.UsedPercent = pct
						}
						// Monitoring errors are swallowed: a query
						// failure leaves UsedPercent at -1 (skip the
						// check), same posture as "not measured."
					}
					// Azure SQL automated PITR backup is always on by
					// platform for every non-Basic tier (retention is
					// configurable 1–35 days but never zero), so this is
					// a true platform invariant, not a placeholder. The
					// probe's no-backup warning therefore correctly never
					// fires for live Azure SQL.
					rec.BackupConfigured = true
					out = append(out, rec)
				}
			}
		}
	}
	return out, nil
}

// ListDisks satisfies pkgazure.Client.
func (l *LiveClient) ListDisks(ctx context.Context) ([]pkgazure.Disk, error) {
	var out []pkgazure.Disk
	pager := l.disks.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, d := range page.Value {
			if d == nil || d.Name == nil {
				continue
			}
			rec := pkgazure.Disk{
				Name:          *d.Name,
				ResourceGroup: resourceGroupFromID(deref(d.ID)),
				Location:      deref(d.Location),
			}
			if d.SKU != nil && d.SKU.Name != nil {
				rec.SKU = string(*d.SKU.Name)
			}
			if d.Properties != nil {
				if d.Properties.ProvisioningState != nil {
					rec.ProvisioningState = *d.Properties.ProvisioningState
				}
				if d.Properties.DiskState != nil {
					rec.DiskState = string(*d.Properties.DiskState)
				}
				if d.Properties.DiskSizeGB != nil {
					rec.SizeGB = int64(*d.Properties.DiskSizeGB)
				}
			}
			if d.ManagedBy != nil {
				rec.AttachedToVM = lastSegment(*d.ManagedBy)
			}
			out = append(out, rec)
		}
	}
	return out, nil
}

// GetAKSCluster satisfies pkgazure.Client. The interface passes only a
// name (no resource group), so we list managed clusters across the
// subscription and match by name.
func (l *LiveClient) GetAKSCluster(ctx context.Context, name string) (*pkgazure.AKSCluster, error) {
	pager := l.aks.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, c := range page.Value {
			if c == nil || c.Name == nil || *c.Name != name {
				continue
			}
			rec := &pkgazure.AKSCluster{
				Name:          *c.Name,
				ResourceGroup: resourceGroupFromID(deref(c.ID)),
				Location:      deref(c.Location),
			}
			if c.Properties != nil {
				if c.Properties.ProvisioningState != nil {
					rec.ProvisioningState = *c.Properties.ProvisioningState
				}
				if c.Properties.PowerState != nil && c.Properties.PowerState.Code != nil {
					rec.PowerState = string(*c.Properties.PowerState.Code)
				}
				if c.Properties.KubernetesVersion != nil {
					rec.KubernetesVersion = *c.Properties.KubernetesVersion
				}
			}
			return rec, nil
		}
	}
	return nil, nil
}

// ListAKSNodePools satisfies pkgazure.Client. Needs the cluster's
// resource group, which we resolve by locating the cluster first.
// Also populates Version (currentOrchestratorVersion) and ClusterVersion
// (the control-plane kubernetesVersion) for version-drift comparison.
func (l *LiveClient) ListAKSNodePools(ctx context.Context, clusterName string) ([]pkgazure.AKSNodePool, error) {
	cluster, err := l.GetAKSCluster(ctx, clusterName)
	if err != nil || cluster == nil {
		return nil, err
	}
	var out []pkgazure.AKSNodePool
	pager := l.aksPools.NewListPager(cluster.ResourceGroup, clusterName, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, p := range page.Value {
			if p == nil || p.Name == nil {
				continue
			}
			rec := pkgazure.AKSNodePool{
				Name:           *p.Name,
				ClusterName:    clusterName,
				ClusterVersion: cluster.KubernetesVersion,
			}
			if p.Properties != nil {
				if p.Properties.ProvisioningState != nil {
					rec.ProvisioningState = *p.Properties.ProvisioningState
				}
				if p.Properties.PowerState != nil && p.Properties.PowerState.Code != nil {
					rec.PowerState = string(*p.Properties.PowerState.Code)
				}
				if p.Properties.Count != nil {
					rec.Count = int64(*p.Properties.Count)
				}
				if p.Properties.EnableAutoScaling != nil {
					rec.Autoscaling = *p.Properties.EnableAutoScaling
				}
				if p.Properties.CurrentOrchestratorVersion != nil {
					rec.Version = *p.Properties.CurrentOrchestratorVersion
				}
			}
			out = append(out, rec)
		}
	}
	return out, nil
}

// ListManagedIdentities satisfies pkgazure.Client. Counts role
// assignments per identity by filtering on the identity's principalID.
func (l *LiveClient) ListManagedIdentities(ctx context.Context) ([]pkgazure.ManagedIdentity, error) {
	var out []pkgazure.ManagedIdentity
	pager := l.identities.NewListBySubscriptionPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, id := range page.Value {
			if id == nil || id.Name == nil {
				continue
			}
			rec := pkgazure.ManagedIdentity{
				Name:          *id.Name,
				ResourceGroup: resourceGroupFromID(deref(id.ID)),
			}
			if id.Properties != nil {
				if id.Properties.ClientID != nil {
					rec.ClientID = *id.Properties.ClientID
				}
				if id.Properties.PrincipalID != nil {
					rec.RoleAssignmentN = l.countRoleAssignments(ctx, *id.Properties.PrincipalID)
				}
			}
			out = append(out, rec)
		}
	}
	return out, nil
}

// countRoleAssignments returns the number of role assignments whose
// principal is principalID. Best-effort: a query error yields 0 (the
// probe then flags it as orphaned, which is the safe direction —
// surfaces rather than hides).
func (l *LiveClient) countRoleAssignments(ctx context.Context, principalID string) int {
	filter := fmt.Sprintf("principalId eq '%s'", principalID)
	pager := l.roleAssigns.NewListForSubscriptionPager(&armauthorization.RoleAssignmentsClientListForSubscriptionOptions{Filter: &filter})
	count := 0
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return count
		}
		count += len(page.Value)
	}
	return count
}

// ListSubnets satisfies pkgazure.Client.
//
// Utilization is MEASURED (G9): TotalIPCount comes from the
// address-prefix mask (minus Azure's 5 reserved addresses) and used
// IPs are counted across every subnet-attached resource type — NIC
// IPConfigurations, ApplicationGatewayIPConfigurations,
// IPConfigurationProfiles, PrivateEndpoints — all READ-ONLY fields
// the apiserver populates on the Subnet resource itself (no $expand,
// no extra API call). AvailableIPCount = total - used, clamped at 0,
// so the IP-exhaustion probe fires on real data.
func (l *LiveClient) ListSubnets(ctx context.Context) ([]pkgazure.Subnet, error) {
	var out []pkgazure.Subnet
	vnetPager := l.vnets.NewListAllPager(nil)
	for vnetPager.More() {
		page, err := vnetPager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, vnet := range page.Value {
			if vnet == nil || vnet.Name == nil {
				continue
			}
			rg := resourceGroupFromID(deref(vnet.ID))
			subPager := l.subnets.NewListPager(rg, *vnet.Name, nil)
			for subPager.More() {
				subPage, err := subPager.NextPage(ctx)
				if err != nil {
					return nil, err
				}
				for _, s := range subPage.Value {
					if s == nil || s.Name == nil {
						continue
					}
					rec := pkgazure.Subnet{
						Name:          *s.Name,
						VNet:          *vnet.Name,
						ResourceGroup: rg,
					}
					if s.Properties != nil && s.Properties.AddressPrefix != nil {
						rec.AddressPrefix = *s.Properties.AddressPrefix
						total := usableIPsFromCIDR(*s.Properties.AddressPrefix)
						rec.TotalIPCount = total
						// G9: count consumed IPs across every subnet-attached
						// resource type (NICs, AppGW configs, IP-config
						// profiles, private endpoints). The Subnet's
						// IPConfigurations + sibling fields are READ-ONLY
						// and populated by the apiserver — no $expand
						// needed. Available = total (already minus Azure's
						// 5-IP reservation) - used; never negative.
						used := int64(subnetUsedIPCount(s))
						avail := total - used
						if avail < 0 {
							avail = 0
						}
						rec.AvailableIPCount = avail
					}
					out = append(out, rec)
				}
			}
		}
	}
	return out, nil
}

// ListAppGatewayBackends satisfies pkgazure.Client.
//
// LIMITATION: backend health is a long-running BackendHealth operation
// per gateway; running it for every probe cycle is expensive and
// can't be verified here. We report TotalCount from the backend
// address pool size and set HealthyCount = -1 ("not measured") so the
// probe SKIPS the health check rather than treating every pool as
// fully healthy, which would silently never fire. A follow-up (v1.9)
// can wire BeginBackendHealth for real per-pool health.
func (l *LiveClient) ListAppGatewayBackends(ctx context.Context) ([]pkgazure.AppGatewayBackend, error) {
	var out []pkgazure.AppGatewayBackend
	pager := l.appgw.NewListAllPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, gw := range page.Value {
			if gw == nil || gw.Name == nil || gw.Properties == nil {
				continue
			}
			// G9: query BackendHealth LRO once per gateway and aggregate
			// per pool. Failures are swallowed; HealthyCount stays at -1
			// so the probe skips the check (same posture as "not measured").
			poolHealthByName := map[string]poolHealth{}
			if l.appgwHealth != nil {
				rg := resourceGroupFromID(deref(gw.ID))
				if h, hErr := l.appgwHealth.BackendHealth(ctx, rg, *gw.Name); hErr == nil {
					poolHealthByName = aggregateBackendHealth(h)
				}
			}
			// The gateway's public hostname (from the already-fetched
			// listener config — no extra API call) feeds
			// FrontendHostname, the CHA-com "(lb: ...)" RCA join key;
			// the probe falls back to the gateway name when empty.
			frontendHostname := appGatewayFrontendHostname(gw)
			for _, pool := range gw.Properties.BackendAddressPools {
				if pool == nil || pool.Name == nil || pool.Properties == nil {
					continue
				}
				total := len(pool.Properties.BackendAddresses)
				rec := pkgazure.AppGatewayBackend{
					Gateway:          *gw.Name,
					PoolName:         *pool.Name,
					TotalCount:       total,
					HealthyCount:     -1, // default: not measured
					FrontendHostname: frontendHostname,
				}
				if ph, ok := poolHealthByName[*pool.Name]; ok {
					rec.HealthyCount = ph.Healthy
					// Prefer the BackendHealth-reported total (which counts
					// observed servers) over the spec's pool-size when
					// available; falls back to the spec total otherwise.
					if ph.Total > 0 {
						rec.TotalCount = ph.Total
					}
				}
				out = append(out, rec)
			}
		}
	}
	return out, nil
}

// ListAppServiceCertificates satisfies pkgazure.Client.
func (l *LiveClient) ListAppServiceCertificates(ctx context.Context) ([]pkgazure.Certificate, error) {
	var out []pkgazure.Certificate
	pager := l.certs.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, c := range page.Value {
			if c == nil || c.Name == nil {
				continue
			}
			rec := pkgazure.Certificate{
				Name:          *c.Name,
				ResourceGroup: resourceGroupFromID(deref(c.ID)),
			}
			if c.Properties != nil {
				if c.Properties.ExpirationDate != nil {
					rec.NotAfter = *c.Properties.ExpirationDate
				}
				// A populated thumbprint means the cert is issued.
				rec.Issued = c.Properties.Thumbprint != nil && *c.Properties.Thumbprint != ""
				// SANs/CN the certificate covers — feeds the CHA-com
				// "(domains: ...)" RCA join key (omitted when empty).
				rec.Domains = derefNonEmpty(c.Properties.HostNames)
			}
			out = append(out, rec)
		}
	}
	return out, nil
}

// ListStorageAccounts satisfies pkgazure.Client.
func (l *LiveClient) ListStorageAccounts(ctx context.Context) ([]pkgazure.StorageAccount, error) {
	var out []pkgazure.StorageAccount
	pager := l.storage.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, a := range page.Value {
			if a == nil || a.Name == nil {
				continue
			}
			rec := pkgazure.StorageAccount{
				Name:          *a.Name,
				ResourceGroup: resourceGroupFromID(deref(a.ID)),
			}
			if a.Properties != nil {
				if a.Properties.AllowBlobPublicAccess != nil {
					rec.AllowBlobPublicAccess = *a.Properties.AllowBlobPublicAccess
				}
				if a.Properties.EnableHTTPSTrafficOnly != nil {
					rec.HTTPSOnly = *a.Properties.EnableHTTPSTrafficOnly
				}
				if a.Properties.MinimumTLSVersion != nil {
					rec.MinTLSVersion = string(*a.Properties.MinimumTLSVersion)
				}
			}
			out = append(out, rec)
		}
	}
	return out, nil
}

// ListKeyVaults satisfies pkgazure.Client.
func (l *LiveClient) ListKeyVaults(ctx context.Context) ([]pkgazure.KeyVault, error) {
	var out []pkgazure.KeyVault
	pager := l.vaults.NewListBySubscriptionPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, v := range page.Value {
			if v == nil || v.Name == nil {
				continue
			}
			rec := pkgazure.KeyVault{
				Name:          *v.Name,
				ResourceGroup: resourceGroupFromID(deref(v.ID)),
			}
			if v.Properties != nil {
				if v.Properties.EnableSoftDelete != nil {
					rec.SoftDelete = *v.Properties.EnableSoftDelete
				}
				if v.Properties.EnablePurgeProtection != nil {
					rec.PurgeProtection = *v.Properties.EnablePurgeProtection
				}
				if v.Properties.PublicNetworkAccess != nil {
					rec.PublicNetwork = strings.EqualFold(*v.Properties.PublicNetworkAccess, "Enabled")
				}
				if v.Properties.VaultURI != nil {
					rec.VaultURL = strings.TrimSuffix(*v.Properties.VaultURI, "/")
				}
			}
			out = append(out, rec)
		}
	}
	return out, nil
}

// kvDataPlaneListResponse is the shape of GET {vaultURL}/keys?api-version=7.4
// and GET {vaultURL}/secrets?api-version=7.4.
type kvDataPlaneListResponse struct {
	Value    []kvDataPlaneItem `json:"value"`
	NextLink string            `json:"nextLink,omitempty"`
}

type kvDataPlaneItem struct {
	// ID is the full item URL, e.g.
	// https://<vault>.vault.azure.net/keys/<name>
	ID         string               `json:"id"`
	Attributes kvDataPlaneAttr      `json:"attributes"`
}

type kvDataPlaneAttr struct {
	Enabled bool  `json:"enabled"`
	Exp     *int64 `json:"exp,omitempty"` // Unix timestamp; nil = no expiry
}

// ListKeyVaultItems satisfies pkgazure.Client. It calls the Key Vault
// data-plane API to enumerate keys and secrets in the vault.
// Returns (nil, nil, nil) when the caller lacks data-plane RBAC — the
// probe skips silently.
func (l *LiveClient) ListKeyVaultItems(ctx context.Context, vaultURL string) ([]pkgazure.KeyVaultKey, []pkgazure.KeyVaultSecret, error) {
	if l.cred == nil || vaultURL == "" {
		return nil, nil, nil
	}
	// Acquire bearer token for the Key Vault data-plane scope.
	tok, err := l.cred.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{"https://vault.azure.net/.default"},
	})
	if err != nil {
		// Auth failure — caller skips silently.
		return nil, nil, err
	}
	bearer := "Bearer " + tok.Token

	vaultName := lastSegment(strings.TrimSuffix(vaultURL, "/"))
	keys, err := kvListItems(ctx, bearer, vaultURL+"/keys?api-version=7.4")
	if err != nil {
		return nil, nil, err
	}
	secrets, err := kvListItems(ctx, bearer, vaultURL+"/secrets?api-version=7.4")
	if err != nil {
		return nil, nil, err
	}

	var outKeys []pkgazure.KeyVaultKey
	for _, item := range keys {
		k := pkgazure.KeyVaultKey{
			VaultName: vaultName,
			KeyName:   lastSegment(item.ID),
			Enabled:   item.Attributes.Enabled,
		}
		if item.Attributes.Exp != nil {
			t := time.Unix(*item.Attributes.Exp, 0).UTC()
			k.ExpiresAt = &t
		}
		outKeys = append(outKeys, k)
	}

	var outSecrets []pkgazure.KeyVaultSecret
	for _, item := range secrets {
		s := pkgazure.KeyVaultSecret{
			VaultName:  vaultName,
			SecretName: lastSegment(item.ID),
			Enabled:    item.Attributes.Enabled,
		}
		if item.Attributes.Exp != nil {
			t := time.Unix(*item.Attributes.Exp, 0).UTC()
			s.ExpiresAt = &t
		}
		outSecrets = append(outSecrets, s)
	}

	return outKeys, outSecrets, nil
}

// kvListItems pages through the Key Vault data-plane list endpoint and
// returns all items. Uses raw HTTP with a pre-acquired bearer token.
func kvListItems(ctx context.Context, bearer, url string) ([]kvDataPlaneItem, error) {
	var out []kvDataPlaneItem
	for url != "" {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", bearer)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnauthorized {
			// Insufficient RBAC — caller skips silently.
			return nil, fmt.Errorf("keyvault data-plane: %s (insufficient RBAC)", resp.Status)
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("keyvault data-plane: unexpected status %s", resp.Status)
		}
		var page kvDataPlaneListResponse
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, err
		}
		out = append(out, page.Value...)
		url = page.NextLink
	}
	return out, nil
}

// --- helpers ---

// appGatewayFrontendHostname returns the gateway's public hostname out
// of its HTTP-listener config: the first listener HostName, else the
// first non-empty entry of any listener's HostNames. "" when no
// listener declares a hostname (the probe then falls back to the
// gateway name for the "(lb: ...)" join key).
func appGatewayFrontendHostname(gw *armnetwork.ApplicationGateway) string {
	if gw == nil || gw.Properties == nil {
		return ""
	}
	for _, lis := range gw.Properties.HTTPListeners {
		if lis == nil || lis.Properties == nil {
			continue
		}
		if h := deref(lis.Properties.HostName); h != "" {
			return h
		}
		for _, hn := range lis.Properties.HostNames {
			if h := deref(hn); h != "" {
				return h
			}
		}
	}
	return ""
}

// derefNonEmpty flattens a []*string into the non-empty values.
// Returns nil when nothing usable remains.
func derefNonEmpty(in []*string) []string {
	var out []string
	for _, s := range in {
		if v := deref(s); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// resourceGroupFromID extracts the resourceGroups segment from an ARM
// resource ID: /subscriptions/X/resourceGroups/RG/providers/...
func resourceGroupFromID(id string) string {
	parts := strings.Split(id, "/")
	for i := 0; i < len(parts)-1; i++ {
		if strings.EqualFold(parts[i], "resourceGroups") {
			return parts[i+1]
		}
	}
	return ""
}

// lastSegment returns the final slash-delimited segment of an ARM ID.
func lastSegment(id string) string {
	if i := strings.LastIndex(id, "/"); i >= 0 {
		return id[i+1:]
	}
	return id
}

// deref returns the string a *string points at, or "" if nil.
func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// derefStatus normalizes a *DatabaseStatus (or similar enum pointer) to
// its string form.
func derefStatus[T ~string](s *T) string {
	if s == nil {
		return ""
	}
	return string(*s)
}

// usableIPsFromCIDR returns usable host addresses in an IPv4 CIDR.
// Azure reserves 5 addresses per subnet (first 4 + broadcast), so
// usable = 2^(32-mask) - 5. Returns 0 for unparseable ranges.
func usableIPsFromCIDR(cidr string) int64 {
	i := strings.LastIndex(cidr, "/")
	if i < 0 {
		return 0
	}
	var mask int
	if _, err := fmt.Sscanf(cidr[i+1:], "%d", &mask); err != nil || mask < 0 || mask > 32 {
		return 0
	}
	hostBits := 32 - mask
	if hostBits <= 2 {
		return 0
	}
	total := int64(1) << uint(hostBits)
	usable := total - 5 // Azure reserves 5 per subnet
	if usable < 0 {
		return 0
	}
	return usable
}
