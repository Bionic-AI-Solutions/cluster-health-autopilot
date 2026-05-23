// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	pkgaws "github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/cloud/aws"
)

// SnapshotClient replays cloud-resource state captured to disk by
// `cha snapshot capture --include-cloud`. Read-only by construction
// — there is no mutation API.
//
// Captured layout (one file per resource type):
//
//	<snapshot-dir>/cloud/aws/
//	  rds.json     ← []pkgaws.DBInstance
//	  ebs.json     ← []pkgaws.Volume      (M1 follow-up)
//	  eks.json     ← []pkgaws.Cluster     (M1 follow-up)
//	  ...
//
// Each file is a JSON array of the corresponding type. Missing files
// are treated as "no resources of that type" (probes return HEALTHY).
type SnapshotClient struct {
	dir    string // <snapshot-dir>/cloud/aws
	region string // recorded at capture time
}

// NewSnapshotClient constructs a snapshot-backed AWS client rooted at
// the snapshot directory's cloud/aws subdir.
func NewSnapshotClient(snapshotDir, region string) *SnapshotClient {
	return &SnapshotClient{
		dir:    filepath.Join(snapshotDir, "cloud", "aws"),
		region: region,
	}
}

// Region returns the region recorded at capture time.
func (c *SnapshotClient) Region() string { return c.region }

// DescribeDBInstances reads rds.json from the snapshot dir. Missing
// file → (nil, nil). Malformed file → (nil, err).
func (c *SnapshotClient) DescribeDBInstances(_ context.Context) ([]pkgaws.DBInstance, error) {
	return readJSON[pkgaws.DBInstance](c.dir, "rds.json")
}

// readJSON is a small generic helper so additional Describe* methods
// can be one-liners as more probes land.
func readJSON[T any](dir, file string) ([]T, error) {
	path := filepath.Join(dir, file)
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var out []T
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	return out, nil
}
