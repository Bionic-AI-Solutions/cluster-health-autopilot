// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package aws

import "time"

// DBInstance is the narrow projection of an RDS DBInstance the RDS
// probe needs. We deliberately do NOT pass the SDK type through — it
// would force every probe consumer to depend on aws-sdk-go-v2.
//
// Fields cover only what the M1 RDS probe asserts on. Extend
// deliberately as analyzers / fixers ask for more.
type DBInstance struct {
	// Identifier is the DB instance identifier (e.g. "prod-db-1").
	// Used in the DriftReport subject.
	Identifier string

	// Engine is the database engine (e.g. "postgres", "mysql",
	// "aurora-postgresql"). Surfaced in the diagnostic for context.
	Engine string

	// Status is the lifecycle state. Values include: available,
	// backing-up, creating, deleting, failed, incompatible-network,
	// incompatible-option-group, incompatible-parameters,
	// incompatible-restore, modifying, rebooting, renaming,
	// resetting-master-credentials, restore-error, starting, stopped,
	// stopping, storage-full, storage-optimization. The probe flags
	// any status that is not "available".
	Status string

	// AllocatedStorageGB is the requested storage size in GB.
	AllocatedStorageGB int32

	// StorageUsedPercent is the percentage of allocated storage
	// currently consumed. SDK reports this via CloudWatch's
	// FreeStorageSpace metric divided by AllocatedStorageGB; the Live
	// wrapper computes this from the metric. Snapshot mode captures
	// the computed percentage directly. 0 means "unknown" (probe
	// emits warning to surface unknown-state pods).
	StorageUsedPercent int

	// MultiAZ indicates whether the instance is configured for
	// Multi-AZ failover. Probe uses this to scope severity of
	// failover events.
	MultiAZ bool

	// Endpoint is the connection endpoint (host:port). Surfaced in
	// the diagnostic so operators can correlate to client errors.
	Endpoint string

	// ARN is the full Amazon Resource Name. Stamped into
	// DriftReport.spec.resourceRef.cloud for unambiguous reference.
	ARN string

	// CreatedAt is the instance creation time. Provided for age-aware
	// analyzers (not used by M1 RDS probe).
	CreatedAt time.Time
}
