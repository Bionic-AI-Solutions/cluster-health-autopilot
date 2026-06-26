// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package gcp

import "time"

// CloudSQLInstance is the narrow projection of a Cloud SQL instance
// the cloudsql probe needs. We deliberately do NOT pass the SDK type
// through — it would force every probe consumer to depend on
// cloud.google.com/go/sqladmin.
//
// JSON tags are the snapshot-file wire format. Changing a tag is a
// snapshot-file backward-compat break; add new fields with new tags
// instead.
//
// State values per the SQL Admin API:
//
//	RUNNABLE, SUSPENDED, PENDING_DELETE, PENDING_CREATE, MAINTENANCE,
//	FAILED, UNKNOWN_STATE
type CloudSQLInstance struct {
	Name              string    `json:"name"`
	DatabaseVersion   string    `json:"databaseVersion"`  // e.g. POSTGRES_15, MYSQL_8_0
	State             string    `json:"state"`            // RUNNABLE / SUSPENDED / etc.
	Region            string    `json:"region,omitempty"` // e.g. us-central1
	Tier              string    `json:"tier,omitempty"`   // e.g. db-n1-standard-2
	DiskSizeGB        int64     `json:"diskSizeGB,omitempty"`
	DiskUsedPercent   int       `json:"diskUsedPercent,omitempty"`   // 0 in Snapshot if not captured
	StorageAutoResize bool      `json:"storageAutoResize,omitempty"` // when true, GCP auto-scales storage
	AvailabilityType  string    `json:"availabilityType,omitempty"`  // ZONAL / REGIONAL (HA)
	BackupConfigured  bool      `json:"backupConfigured,omitempty"`
	LastBackupAt      time.Time `json:"lastBackupAt,omitempty"`
	ConnectionName    string    `json:"connectionName,omitempty"` // "<project>:<region>:<instance>"
	CreatedAt         time.Time `json:"createdAt,omitempty"`
}

// PersistentDisk is the narrow projection of a Persistent Disk. State
// values per the Compute API: CREATING, RESTORING, FAILED, READY,
// DELETING. Plus our derived semantics: a disk with Users == nil and
// Status == READY is "available" (detached).
type PersistentDisk struct {
	Name             string        `json:"name"`
	Status           string        `json:"status"` // CREATING / READY / FAILED / etc.
	SizeGB           int64         `json:"sizeGB"`
	Type             string        `json:"type,omitempty"`             // pd-ssd / pd-balanced / pd-standard
	Zone             string        `json:"zone,omitempty"`             // zonal disks
	Region           string        `json:"region,omitempty"`           // regional disks (HA)
	AttachedToVM     string        `json:"attachedToVM,omitempty"`     // empty when detached
	DetachedDuration time.Duration `json:"detachedDuration,omitempty"` // computed by Live (now - DetachTime); 0 in Snapshot
	CreatedAt        time.Time     `json:"createdAt,omitempty"`
}

// GKECluster is the narrow projection of a GKE cluster. Status values
// per the Container API: PROVISIONING, RUNNING, RECONCILING,
// STOPPING, ERROR, DEGRADED.
type GKECluster struct {
	Name           string    `json:"name"`
	Status         string    `json:"status"`
	Location       string    `json:"location,omitempty"`       // region or zone
	CurrentVersion string    `json:"currentVersion,omitempty"` // control-plane k8s version
	NodeCount      int64     `json:"nodeCount,omitempty"`
	Endpoint       string    `json:"endpoint,omitempty"`
	CreatedAt      time.Time `json:"createdAt,omitempty"`
}

// GKENodePool is the narrow projection of a GKE node pool. Status
// values: PROVISIONING, RUNNING, RUNNING_WITH_ERROR, RECONCILING,
// STOPPING, ERROR.
type GKENodePool struct {
	Name           string `json:"name"`
	ClusterName    string `json:"clusterName"`
	Status         string `json:"status"`
	NodeCount      int64  `json:"nodeCount,omitempty"`
	Version        string `json:"version,omitempty"`        // node k8s version
	ClusterVersion string `json:"clusterVersion,omitempty"` // control-plane version for drift comparison
	Autoscaling    bool   `json:"autoscaling,omitempty"`
}

// ServiceAccount is the narrow projection of an IAM service account.
// Disabled accounts that are still referenced by a workload are the
// drift signal.
type ServiceAccount struct {
	Email       string `json:"email"`
	DisplayName string `json:"displayName,omitempty"`
	Disabled    bool   `json:"disabled,omitempty"`
	KeyCount    int    `json:"keyCount,omitempty"`    // user-managed keys (long-lived; security smell)
	OAuth2Bound bool   `json:"oauth2Bound,omitempty"` // bound to a GKE workload-identity binding
}

// Subnet is the narrow projection of a VPC subnetwork. The drift
// signal is IP exhaustion: AvailableIPCount low relative to range —
// when measured. The live wrapper cannot measure used IPs cheaply
// (AvailableIPCount = -1, "not measured"); the probe then falls back
// to a capacity-only check on the primary CIDR. See
// internal/cloud/gcp/live.go ListSubnets for the live contract.
type Subnet struct {
	Name             string `json:"name"`
	Network          string `json:"network,omitempty"`
	Region           string `json:"region,omitempty"`
	IPCIDRRange      string `json:"ipCidrRange,omitempty"`
	AvailableIPCount int64  `json:"availableIPCount"` // free addresses; -1 = not measured
	TotalIPCount     int64  `json:"totalIPCount"`     // usable addresses in the primary range
	// SecondaryIPCount is the summed address capacity of the subnet's
	// secondary (alias) ranges — GKE pod/service ranges live here. GCP
	// reserves no addresses in secondary ranges, so this is the full
	// range size. 0 = no secondary ranges (or captured before this
	// field existed).
	//
	// Data-only: no probe consumes it today — it ships purely as
	// snapshot capture surface (context for operators reading captures
	// and for future capacity logic). The Subnets probe's checks read
	// TotalIPCount / AvailableIPCount only.
	SecondaryIPCount int64 `json:"secondaryIPCount,omitempty"`
}

// BackendService is the narrow projection of a Load Balancer backend
// service with its backend-health summary inlined (avoids per-backend
// fan-out). Mirrors AWS ALBTargetGroup.
type BackendService struct {
	Name           string `json:"name"`
	Protocol       string `json:"protocol,omitempty"`
	HealthyCount   int    `json:"healthyCount"`
	UnhealthyCount int    `json:"unhealthyCount"`
	TotalBackends  int    `json:"totalBackends,omitempty"`
	// ForwardingRule is the forwarding-rule IP (preferred) or name of
	// the rule pointing at this backend service (compute
	// ForwardingRules aggregated list, joined on rule.BackendService —
	// passthrough LBs only; proxy-based rules don't reference the
	// backend service directly). Optional: empty when unmapped
	// (including snapshot files captured before this field existed) —
	// the probe then falls back to the backend-service name for the
	// "(lb: ...)" message join key Srenix Enterprise's RCA matchers parse.
	ForwardingRule string `json:"forwardingRule,omitempty"`
}

// ManagedCertificate is the narrow projection of a Google-managed SSL
// certificate. Status: ACTIVE, PROVISIONING, PROVISIONING_FAILED,
// PROVISIONING_FAILED_PERMANENTLY, RENEWAL_FAILED. Mirrors AWS
// ACMCertificate.
type ManagedCertificate struct {
	Name       string    `json:"name"`
	Status     string    `json:"status"`
	DomainName string    `json:"domainName,omitempty"`
	NotAfter   time.Time `json:"notAfter,omitempty"`
}

// Bucket is the narrow projection of a GCS bucket's public-access
// posture. PublicAccessPrevention: "enforced" (good) or "inherited"
// (may allow public ACLs). UniformBucketLevelAccess disables
// object-level ACLs. Mirrors AWS S3BucketPAB.
type Bucket struct {
	Name                     string `json:"name"`
	PublicAccessPrevention   string `json:"publicAccessPrevention,omitempty"` // enforced / inherited
	UniformBucketLevelAccess bool   `json:"uniformBucketLevelAccess,omitempty"`
	HasAllUsersBinding       bool   `json:"hasAllUsersBinding,omitempty"` // IAM grants allUsers/allAuthenticatedUsers
}

// KMSKey is the narrow projection of a Cloud KMS crypto key (primary
// version). State: ENABLED, DISABLED, DESTROYED, DESTROY_SCHEDULED,
// PENDING_GENERATION, IMPORT_FAILED, GENERATION_FAILED. Mirrors AWS
// KMSKey.
type KMSKey struct {
	Name              string    `json:"name"`
	PrimaryState      string    `json:"primaryState"`
	Purpose           string    `json:"purpose,omitempty"`
	RotationScheduled bool      `json:"rotationScheduled,omitempty"` // automatic rotation configured
	NextRotation      time.Time `json:"nextRotation,omitempty"`
}
