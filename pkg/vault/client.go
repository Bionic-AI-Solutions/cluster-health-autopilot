// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

// Package vault exports the read-only Client contract used by
// VaultPathMissing-class analyzers. The live HTTP implementation
// lives in `internal/vault` (not exported); external consumers
// (e.g. the paid CHA-com binary) wire their own implementation
// against this interface.
//
// Privacy contract: the public surface of this package only ever returns
// the SET OF KEY NAMES at a Vault path (via ListKeys). Byte values are
// never logged, never returned, never persisted. Implementations must
// preserve that contract — see internal/vault/client.go for the canonical
// implementation pattern.
package vault

import (
	"context"
	"errors"
)

// Client reads key names from a Vault KV-v2 path.
//
// Implementations must:
//   - Return ErrPathNotFound when the path does not exist (404).
//   - Return only metadata-level key names — never the byte values stored
//     under them. This is enforced by the interface signature returning
//     []string, not map[string][]byte.
type Client interface {
	// ListKeys returns the KEY NAMES present at the given KV-v2 path.
	// Path is mount-relative (e.g. "team/app", not "secret/data/team/app").
	ListKeys(ctx context.Context, path string) ([]string, error)
}

// ErrPathNotFound is returned by Client.ListKeys when Vault returns 404
// for the path. Distinguished from transport / auth errors so the analyzer
// can emit a precise "path missing in Vault" diagnostic.
//
// Use errors.Is(err, ErrPathNotFound) to test for it; implementations
// MAY wrap this sentinel for additional context.
var ErrPathNotFound = errors.New("vault path not found")
