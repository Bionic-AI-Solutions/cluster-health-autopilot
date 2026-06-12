// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package chartgate

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// SampleCRFullPath returns the absolute path to the bundle's
// full-surface sample CR (bundle/tests/sample-cr-full.yaml — every
// optional subtree populated; the bundle-smoke gate maintains it).
//
// Centralized here so every parity gate that loads the fixture
// (cmd/cha/operatorflags_test.go and friends) resolves it through ONE
// locator: if bundle/tests ever moves, the gates fail loudly with this
// message instead of each test silently breaking on its own relative
// path.
func SampleCRFullPath(t *testing.T) string {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	path := filepath.Join(filepath.Dir(thisFile), "..", "..", "bundle", "tests", "sample-cr-full.yaml")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("full-surface sample CR not found at %s — did bundle/tests move? Update internal/chartgate.SampleCRFullPath (the shared fixture locator) and every gate resolving through it: %v", path, err)
	}
	return path
}
