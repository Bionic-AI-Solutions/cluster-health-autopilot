// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestSnapshotClient_DescribeDBInstances_MissingFileReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	c := NewSnapshotClient(dir, "us-east-1")
	got, err := c.DescribeDBInstances(context.Background())
	if err != nil {
		t.Fatalf("missing file should not error, got %v", err)
	}
	if len(got) != 0 {
		t.Errorf("missing file should yield 0 instances, got %d", len(got))
	}
	if c.Region() != "us-east-1" {
		t.Errorf("Region()=%q want us-east-1", c.Region())
	}
}

func TestSnapshotClient_DescribeDBInstances_RoundTripsCapturedJSON(t *testing.T) {
	dir := t.TempDir()
	awsDir := filepath.Join(dir, "cloud", "aws")
	if err := os.MkdirAll(awsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := `[
		{"identifier":"prod-db","engine":"postgres","status":"available","storageUsedPercent":42,"allocatedStorageGB":100,"multiAZ":true,"endpoint":"prod-db.x.rds.amazonaws.com:5432","arn":"arn:aws:rds:us-east-1:1:db:prod-db"},
		{"identifier":"warn-db","engine":"mysql","status":"available","storageUsedPercent":85,"allocatedStorageGB":50}
	]`
	if err := os.WriteFile(filepath.Join(awsDir, "rds.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	// We deliberately serialize with JSON field names — verify the
	// SnapshotClient honors them. (DBInstance fields don't carry
	// explicit json tags, so this test pins down lowerCamelCase
	// behavior of encoding/json's default field-name handling.)
	c := NewSnapshotClient(dir, "us-east-1")
	got, err := c.DescribeDBInstances(context.Background())
	if err != nil {
		t.Fatalf("DescribeDBInstances: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 instances, got %d", len(got))
	}
	// Default decode uses case-insensitive match, so "identifier" maps
	// to Identifier. Verify a couple of fields:
	if got[0].Identifier == "" {
		t.Errorf("Identifier did not decode from JSON; payload may need json tags. Got: %+v", got[0])
	}
}

func TestSnapshotClient_DescribeDBInstances_MalformedJSON(t *testing.T) {
	dir := t.TempDir()
	awsDir := filepath.Join(dir, "cloud", "aws")
	if err := os.MkdirAll(awsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(awsDir, "rds.json"), []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	c := NewSnapshotClient(dir, "us-east-1")
	if _, err := c.DescribeDBInstances(context.Background()); err == nil {
		t.Error("malformed json should return error")
	}
}
