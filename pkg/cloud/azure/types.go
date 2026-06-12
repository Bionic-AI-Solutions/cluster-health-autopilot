// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package azure

import "time"

// SQLDatabase is the narrow projection of an Azure SQL Database the
// SQLDatabases probe needs. JSON tags are the snapshot-file wire
// format.
//
// Status values per the Microsoft SQL Database API:
//
//	Online, Offline, Restoring, RecoveryPending, Recovery, Suspect,
//	AutoClosed, Copying, Creating, Inaccessible, Disabled, EmergencyMode,
//	OfflineSecondary, Paused, Pausing, Resuming, Scaling.
//
// Tier examples: Basic, Standard, Premium, GeneralPurpose,
// BusinessCritical, Hyperscale, ServerlessV2.
type SQLDatabase struct {
	Name             string    `json:"name"`
	ResourceGroup    string    `json:"resourceGroup"`
	Server           string    `json:"server"`
	Status           string    `json:"status"`         // Online / Offline / Paused / etc.
	Tier             string    `json:"tier,omitempty"` // service tier
	MaxSizeGB        int64     `json:"maxSizeGB,omitempty"`
	UsedPercent      int       `json:"usedPercent,omitempty"`      // 0 in Snapshot if not captured
	ZoneRedundant    bool      `json:"zoneRedundant,omitempty"`    // multi-AZ posture
	BackupConfigured bool      `json:"backupConfigured,omitempty"` // automated backups on
	GeoBackup        bool      `json:"geoBackup,omitempty"`        // geo-redundant backups
	CreatedAt        time.Time `json:"createdAt,omitempty"`
}

// Disk is the narrow projection of an Azure Managed Disk. ProvisioningState
// values: Creating, Updating, Succeeded, Failed, Deleting. DiskState
// values: Unattached, Attached, Reserved, ActiveSAS, ActiveSASFrozen,
// ReadyToUpload, ActiveUpload.
type Disk struct {
	Name              string        `json:"name"`
	ResourceGroup     string        `json:"resourceGroup"`
	Location          string        `json:"location,omitempty"`
	ProvisioningState string        `json:"provisioningState"`   // Succeeded / Failed / Creating / etc.
	DiskState         string        `json:"diskState,omitempty"` // Attached / Unattached / etc.
	SKU               string        `json:"sku,omitempty"`       // Standard_LRS / Premium_LRS / etc.
	SizeGB            int64         `json:"sizeGB,omitempty"`
	AttachedToVM      string        `json:"attachedToVM,omitempty"`     // empty when detached
	DetachedDuration  time.Duration `json:"detachedDuration,omitempty"` // Live: now - lastDetach; 0 in Snapshot
	CreatedAt         time.Time     `json:"createdAt,omitempty"`
}

// AKSCluster is the narrow projection of an AKS managed cluster.
// ProvisioningState: Succeeded, Failed, Creating, Updating, Deleting.
// PowerState.Code: Running, Stopped.
type AKSCluster struct {
	Name              string    `json:"name"`
	ResourceGroup     string    `json:"resourceGroup"`
	Location          string    `json:"location,omitempty"`
	ProvisioningState string    `json:"provisioningState"`
	PowerState        string    `json:"powerState,omitempty"` // Running / Stopped
	KubernetesVersion string    `json:"kubernetesVersion,omitempty"`
	CreatedAt         time.Time `json:"createdAt,omitempty"`
}

// AKSNodePool is the narrow projection of an AKS agent pool.
// ProvisioningState mirrors the cluster; PowerState too.
type AKSNodePool struct {
	Name              string `json:"name"`
	ClusterName       string `json:"clusterName"`
	ProvisioningState string `json:"provisioningState"`
	PowerState        string `json:"powerState,omitempty"`
	Count             int64  `json:"count,omitempty"`
	Autoscaling       bool   `json:"autoscaling,omitempty"`
}

// ManagedIdentity is the narrow projection of a user-assigned managed
// identity. The drift signal is an identity with no role assignments
// (orphaned) — it's referenced by a workload but grants nothing, so
// the workload silently lacks permissions.
type ManagedIdentity struct {
	Name            string `json:"name"`
	ResourceGroup   string `json:"resourceGroup"`
	ClientID        string `json:"clientId,omitempty"`
	RoleAssignmentN int    `json:"roleAssignmentCount"` // number of role assignments
	FederatedCredsN int    `json:"federatedCredsCount"` // workload-identity federated credentials
}

// Subnet is the narrow projection of a VNet subnet. Drift signal:
// IP-address exhaustion.
type Subnet struct {
	Name             string `json:"name"`
	VNet             string `json:"vnet,omitempty"`
	ResourceGroup    string `json:"resourceGroup,omitempty"`
	AddressPrefix    string `json:"addressPrefix,omitempty"`
	AvailableIPCount int64  `json:"availableIPCount"`
	TotalIPCount     int64  `json:"totalIPCount"`
}

// AppGatewayBackend is the narrow projection of an Application Gateway
// backend pool with health summary inlined. Mirrors AWS ALBTargetGroup
// / GCP BackendService.
type AppGatewayBackend struct {
	Gateway        string `json:"gateway"`
	PoolName       string `json:"poolName"`
	HealthyCount   int    `json:"healthyCount"`
	UnhealthyCount int    `json:"unhealthyCount"`
	TotalCount     int    `json:"totalCount,omitempty"`
	// FrontendHostname is the gateway's public hostname taken from the
	// already-fetched HTTP-listener config (HostName / HostNames).
	// Optional: empty when no listener declares a hostname (including
	// snapshot files captured before this field existed) — the probe
	// then falls back to the gateway name for the "(lb: ...)" message
	// join key CHA-com's RCA matchers parse.
	FrontendHostname string `json:"frontendHostname,omitempty"`
}

// Certificate is the narrow projection of an App Service / managed
// certificate. Mirrors AWS ACMCertificate / GCP ManagedCertificate.
type Certificate struct {
	Name          string    `json:"name"`
	ResourceGroup string    `json:"resourceGroup,omitempty"`
	NotAfter      time.Time `json:"notAfter,omitempty"`
	Issued        bool      `json:"issued,omitempty"` // false = provisioning/failed
	// Domains are the hostnames the certificate covers (SANs/CN from
	// the certificate resource's HostNames). Optional: empty when not
	// surfaced (including snapshot files captured before this field
	// existed) — the probe then omits the "(domains: ...)" message
	// join key CHA-com's RCA matchers parse.
	Domains []string `json:"domains,omitempty"`
}

// StorageAccount is the narrow projection of a Storage account's
// public-access posture. Mirrors AWS S3BucketPAB / GCP Bucket.
type StorageAccount struct {
	Name                  string `json:"name"`
	ResourceGroup         string `json:"resourceGroup,omitempty"`
	AllowBlobPublicAccess bool   `json:"allowBlobPublicAccess"` // true = containers may be public
	HTTPSOnly             bool   `json:"httpsOnly"`             // false = plaintext allowed
	MinTLSVersion         string `json:"minTlsVersion,omitempty"`
}

// KeyVault is the narrow projection of a Key Vault's data-protection
// posture. Mirrors AWS KMSKey / GCP KMSKey but Azure's drift signal is
// soft-delete / purge-protection rather than key state.
type KeyVault struct {
	Name            string `json:"name"`
	ResourceGroup   string `json:"resourceGroup,omitempty"`
	SoftDelete      bool   `json:"softDelete"`      // recovery window for deleted secrets
	PurgeProtection bool   `json:"purgeProtection"` // prevents permanent delete during window
	PublicNetwork   bool   `json:"publicNetwork"`   // reachable from public internet
}
