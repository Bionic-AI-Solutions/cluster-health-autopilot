// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package aws

import "time"

// DBInstance is the narrow projection of an RDS DBInstance the RDS
// probe needs. We deliberately do NOT pass the SDK type through — it
// would force every probe consumer to depend on aws-sdk-go-v2.
//
// JSON tags are the snapshot-file wire format. Changing a tag is a
// snapshot-file backward-compat break; add new fields with new tags
// instead.
type DBInstance struct {
	Identifier         string    `json:"identifier"`
	Engine             string    `json:"engine"`
	Status             string    `json:"status"`
	AllocatedStorageGB int32     `json:"allocatedStorageGB"`
	StorageUsedPercent int       `json:"storageUsedPercent"`
	MultiAZ            bool      `json:"multiAZ,omitempty"`
	Endpoint           string    `json:"endpoint,omitempty"`
	ARN                string    `json:"arn,omitempty"`
	CreatedAt          time.Time `json:"createdAt,omitempty"`
	// BackupRetentionPeriod is the number of days automated backups are
	// retained. 0 = automated backups disabled.
	BackupRetentionPeriod int `json:"backupRetentionPeriod,omitempty"`
	// ReadReplicaSourceDBInstanceIdentifier is non-empty when this
	// instance is a read replica; empty for primary instances.
	ReadReplicaSourceDBInstanceIdentifier string `json:"readReplicaSourceDBInstanceIdentifier,omitempty"`
}

// Volume is the narrow projection of an EBS volume. State values:
// creating, available, in-use, deleting, deleted, error.
type Volume struct {
	VolumeID         string        `json:"volumeId"`
	State            string        `json:"state"`
	SizeGB           int32         `json:"sizeGB"`
	VolumeType       string        `json:"volumeType,omitempty"`
	AttachedToEC2    string        `json:"attachedToEC2,omitempty"`    // empty when detached
	DetachedDuration time.Duration `json:"detachedDuration,omitempty"` // computed by Live (now - DetachTime); 0 in Snapshot if not captured
	CreatedAt        time.Time     `json:"createdAt,omitempty"`
	ARN              string        `json:"arn,omitempty"`
}

// EKSCluster is the narrow projection of an EKS cluster.
// Status values: CREATING, ACTIVE, DELETING, FAILED, UPDATING, PENDING.
type EKSCluster struct {
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Version   string    `json:"version"`
	Endpoint  string    `json:"endpoint,omitempty"`
	ARN       string    `json:"arn,omitempty"`
	CreatedAt time.Time `json:"createdAt,omitempty"`
}

// EKSNodeGroup is the narrow projection of an EKS managed node group.
// Status values: CREATING, ACTIVE, UPDATING, DELETING, CREATE_FAILED,
// UPDATE_FAILED, DELETE_FAILED, DEGRADED.
type EKSNodeGroup struct {
	ClusterName  string   `json:"clusterName"`
	Name         string   `json:"name"`
	Status       string   `json:"status"`
	Version      string   `json:"version,omitempty"`
	DesiredSize  int32    `json:"desiredSize"`
	MinSize      int32    `json:"minSize"`
	MaxSize      int32    `json:"maxSize"`
	HealthIssues []string `json:"healthIssues,omitempty"` // SDK Health.Issues[*].Code
	ARN          string   `json:"arn,omitempty"`
}

// IAMRole is the narrow projection of an IAM role lookup result.
// Exists==false means GetRole returned NoSuchEntity — caller assumes
// drift (an IRSA pointer to a non-existent role).
type IAMRole struct {
	ARN            string `json:"arn"`
	Name           string `json:"name,omitempty"`
	Exists         bool   `json:"exists"`
	HasTrustPolicy bool   `json:"hasTrustPolicy,omitempty"`
	ErrorMessage   string `json:"errorMessage,omitempty"`
}

// ALBTargetGroup combines an ALB/NLB target group with its current
// target health summary in one record so the probe doesn't have to
// fan out N+1 calls.
type ALBTargetGroup struct {
	ARN            string `json:"arn"`
	Name           string `json:"name"`
	Protocol       string `json:"protocol,omitempty"`
	Port           int32  `json:"port,omitempty"`
	TargetType     string `json:"targetType,omitempty"`
	HealthyCount   int    `json:"healthyCount"`
	UnhealthyCount int    `json:"unhealthyCount"`
	UnusedCount    int    `json:"unusedCount,omitempty"`
	InitialCount   int    `json:"initialCount,omitempty"`
	// LoadBalancerDNS is the DNS name of the load balancer this target
	// group is attached to (first of TargetGroup.LoadBalancerArns,
	// resolved via elbv2.DescribeLoadBalancers). Optional: empty when
	// the TG is unattached or the LB could not be resolved (including
	// snapshot files captured before this field existed) — the probe
	// then omits the "(lb: ...)" message join key Srenix Enterprise's RCA
	// matchers parse.
	LoadBalancerDNS string `json:"loadBalancerDNS,omitempty"`
}

// ACMCertificate is the narrow projection of an ACM cert.
// Status values: PENDING_VALIDATION, ISSUED, INACTIVE, EXPIRED,
// VALIDATION_TIMED_OUT, REVOKED, FAILED.
type ACMCertificate struct {
	ARN        string    `json:"arn"`
	DomainName string    `json:"domainName"`
	Status     string    `json:"status"`
	NotAfter   time.Time `json:"notAfter,omitempty"`
	Type       string    `json:"type,omitempty"` // IMPORTED, AMAZON_ISSUED, PRIVATE
}

// KMSKey is the narrow projection of a KMS Customer Master Key.
// State values: Creating, Enabled, Disabled, PendingDeletion,
// PendingImport, Unavailable, PendingReplicaDeletion.
type KMSKey struct {
	KeyID        string    `json:"keyId"`
	ARN          string    `json:"arn"`
	State        string    `json:"state"`
	Enabled      bool      `json:"enabled"`
	DeletionDate time.Time `json:"deletionDate,omitempty"` // set when State=PendingDeletion
	Description  string    `json:"description,omitempty"`
}

// S3BucketPAB is the public-access-block configuration for one bucket.
// Drift signature: bucket has objects/policy expectations of private
// but PublicAccessBlock is fully or partially disabled.
type S3BucketPAB struct {
	Bucket                string `json:"bucket"`
	BlockPublicAcls       bool   `json:"blockPublicAcls"`
	IgnorePublicAcls      bool   `json:"ignorePublicAcls"`
	BlockPublicPolicy     bool   `json:"blockPublicPolicy"`
	RestrictPublicBuckets bool   `json:"restrictPublicBuckets"`
	HasPolicyError        bool   `json:"hasPolicyError,omitempty"` // GetPublicAccessBlock returned NoSuchPublicAccessBlockConfiguration
}

// VPCSubnet is the narrow projection of a VPC subnet, focused on
// IP-capacity drift (the most common subnet pain).
type VPCSubnet struct {
	SubnetID                  string `json:"subnetId"`
	VPCID                     string `json:"vpcId"`
	CIDRBlock                 string `json:"cidrBlock,omitempty"`
	AvailabilityZone          string `json:"availabilityZone,omitempty"`
	AvailableIPv4AddressCount int32  `json:"availableIPv4AddressCount"`
}

// EKSAddon is the narrow projection of an EKS managed add-on.
// Status values: CREATING, ACTIVE, UPDATE_FAILED, DELETING, DELETE_FAILED,
// DEGRADED, CREATE_FAILED.
type EKSAddon struct {
	ClusterName       string `json:"clusterName"`
	AddonName         string `json:"addonName"`
	Status            string `json:"status"`
	AddonVersion      string `json:"addonVersion,omitempty"`      // currently installed version
	MarketplaceVersion string `json:"marketplaceVersion,omitempty"` // latest available version
	ARN               string `json:"arn,omitempty"`
}
