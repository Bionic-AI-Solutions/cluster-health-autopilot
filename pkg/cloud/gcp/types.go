// Copyright 2026 Cluster Health Autopilot contributors
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
