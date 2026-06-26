// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

// Package fix contains whitelisted auto-remediation fixers.
//
// Each fixer is a narrowly-scoped, idempotent action targeting a specific
// failure pattern with a known-safe corrective action. Fixers are gated by
// the snapshot.Mutator interface: they refuse to run when the source is
// snapshot-backed (offline mode).
//
// Cluster mutations forbidden by design: Secret writes, ConfigMap writes,
// CRD writes. Resolution for those classes is surfaced through the
// internal/diagnose analyzers instead.
//
// New fixers must:
//  1. Implement the Fixer interface.
//  2. Skip protected namespaces via IsProtectedNamespace().
//  3. Never mutate Secrets, ConfigMaps, or CRDs.
//  4. Be idempotent — repeated runs against an unchanged cluster are no-ops.
//  5. Carry comprehensive tests including a "no-op when nothing matches" case.
//
// The canonical Fixer interface and result types live in pkg/fix.
// The aliases below keep all internal implementations compiling unchanged.
package fix

import pkgfix "github.com/srenix-ai/agentic-sre/pkg/fix"

// Action is re-exported from pkg/fix; see that package for the canonical definition.
type Action = pkgfix.Action

// SkipReason is re-exported from pkg/fix; see that package for the canonical definition.
type SkipReason = pkgfix.SkipReason

// Result is re-exported from pkg/fix; see that package for the canonical definition.
type Result = pkgfix.Result

// Fixer is re-exported from pkg/fix; see that package for the canonical definition.
type Fixer = pkgfix.Fixer
