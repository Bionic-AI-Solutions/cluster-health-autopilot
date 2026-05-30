// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"
	"fmt"
	"strings"

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
				Name:        *p.Name,
				ClusterName: clusterName,
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
// LIMITATION: the Network API Subnet resource does not expose a
// free-IP count. We populate TotalIPCount from the address-prefix
// mask and set AvailableIPCount = TotalIPCount so the IP-exhaustion
// probe never false-positives. Real utilization needs the Network
// usage API — a follow-up.
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
						// Free-IP count needs Azure Monitor metrics; -1 =
						// "not measured" so the probe skips the IP check
						// rather than treating the subnet as 100% free
						// (which would silently never fire). v1.9.
						rec.AvailableIPCount = -1
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
			for _, pool := range gw.Properties.BackendAddressPools {
				if pool == nil || pool.Name == nil || pool.Properties == nil {
					continue
				}
				total := len(pool.Properties.BackendAddresses)
				out = append(out, pkgazure.AppGatewayBackend{
					Gateway:      *gw.Name,
					PoolName:     *pool.Name,
					TotalCount:   total,
					HealthyCount: -1, // not measured — see LIMITATION
				})
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
			}
			out = append(out, rec)
		}
	}
	return out, nil
}

// --- helpers ---

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
