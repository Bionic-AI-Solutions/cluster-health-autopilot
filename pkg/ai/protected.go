// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package ai

import (
	"os"
	"strings"
	"sync"
)

// EnvProtectedNamespacesExtra is the comma-separated list of ADDITIONAL
// protected namespaces an operator may layer on top of the compiled-in
// floor (ProtectedNamespaces / internal/fix's mirror). APPEND-ONLY by
// contract: entries extend the no-touch set; nothing in this env var
// can ever remove a compiled-in namespace. Whitespace around entries is
// trimmed; empty entries are ignored; duplicates collapse.
//
// Rendered by the Helm chart from `protectedNamespaces.extra` and by
// the operator from `spec.protectedNamespacesExtra` onto the watcher,
// diagnose, remediate, and aiwatch containers, so the fixer guard
// (internal/fix.IsProtectedNamespace) and the AI-action validator
// (ai.IsProtectedNamespace) always agree on the boundary.
const EnvProtectedNamespacesExtra = "SRENIX_PROTECTED_NAMESPACES_EXTRA"

var (
	extraMu        sync.RWMutex
	extraProtected map[string]struct{}
	extraList      []string
	extraLoaded    bool
)

// ParseProtectedNamespacesExtra parses the comma-separated extension
// list: entries are whitespace-trimmed, empties dropped, duplicates
// collapsed (first occurrence wins ordering). Returns nil for an empty
// or all-garbage input.
func ParseProtectedNamespacesExtra(raw string) []string {
	var out []string
	seen := map[string]struct{}{}
	for _, part := range strings.Split(raw, ",") {
		ns := strings.TrimSpace(part)
		if ns == "" {
			continue
		}
		if _, dup := seen[ns]; dup {
			continue
		}
		seen[ns] = struct{}{}
		out = append(out, ns)
	}
	return out
}

// SetExtraProtectedNamespaces replaces the extra (operator-extended)
// protected set. The compiled-in floor is NOT touched — extras only
// ever ADD namespaces. Calling with no arguments clears the extras
// back to floor-only. Entries are trimmed/deduped; empties ignored.
//
// Embedders (e.g. the Srenix Enterprise aiwatch) normally don't need to call
// this: the first IsProtectedNamespace check lazily reads
// SRENIX_PROTECTED_NAMESPACES_EXTRA. The setter exists for hosts that
// configure the extension from another source, and for tests.
func SetExtraProtectedNamespaces(namespaces ...string) {
	list, set := buildExtraSet(strings.Join(namespaces, ","))
	extraMu.Lock()
	extraProtected = set
	extraList = list
	extraLoaded = true
	extraMu.Unlock()
}

// buildExtraSet parses raw into the ordered list + lookup set pair
// stored behind extraMu.
func buildExtraSet(raw string) ([]string, map[string]struct{}) {
	list := ParseProtectedNamespacesExtra(raw)
	set := make(map[string]struct{}, len(list))
	for _, ns := range list {
		set[ns] = struct{}{}
	}
	return list, set
}

// LoadExtraProtectedNamespacesFromEnv (re)reads
// SRENIX_PROTECTED_NAMESPACES_EXTRA and replaces the extra set with its
// contents. Happens automatically on first use; exported so hosts and
// tests can force a re-read after mutating the environment.
func LoadExtraProtectedNamespacesFromEnv() {
	SetExtraProtectedNamespaces(os.Getenv(EnvProtectedNamespacesExtra))
}

// IsExtraProtectedNamespace reports whether ns is in the operator-
// extended (append-only) protected set sourced from
// SRENIX_PROTECTED_NAMESPACES_EXTRA or SetExtraProtectedNamespaces. The
// compiled-in floor is NOT consulted here — callers that keep their own
// floor (internal/fix) OR this set with it. Most callers want
// IsProtectedNamespace instead.
func IsExtraProtectedNamespace(ns string) bool {
	if ns == "" {
		return false
	}
	_, ok := extraProtectedSet()[ns]
	return ok
}

// extraProtectedSet returns the current extra set, lazily loading it
// from the environment on first use.
func extraProtectedSet() map[string]struct{} {
	extraMu.RLock()
	if extraLoaded {
		s := extraProtected
		extraMu.RUnlock()
		return s
	}
	extraMu.RUnlock()
	return loadExtraFromEnvIfUnloaded()
}

// loadExtraFromEnvIfUnloaded is the slow path of extraProtectedSet.
// extraLoaded is RE-CHECKED under the write lock (double-checked
// locking): a SetExtraProtectedNamespaces that raced in between the
// caller's RUnlock and this load must win — otherwise the env value
// would silently overwrite the host's explicit configuration.
func loadExtraFromEnvIfUnloaded() map[string]struct{} {
	extraMu.Lock()
	defer extraMu.Unlock()
	if !extraLoaded {
		extraList, extraProtected = buildExtraSet(os.Getenv(EnvProtectedNamespacesExtra))
		extraLoaded = true
	}
	return extraProtected
}

// ExtraProtectedNamespaces returns the operator-extended namespaces
// (floor excluded) in first-seen order. Intended for startup logging —
// the act-side surfaces should announce a widened boundary.
func ExtraProtectedNamespaces() []string {
	extraProtectedSet() // ensure loaded
	extraMu.RLock()
	defer extraMu.RUnlock()
	if len(extraList) == 0 {
		return nil
	}
	out := make([]string, len(extraList))
	copy(out, extraList)
	return out
}
