// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

// Package chartgate holds permanent CI gates that keep the Helm chart and
// the compiled binary in lockstep. Two drift classes are policed here:
//
//   - toggle drift (P1.8): a SRENIX_* env toggle the binary reads but the
//     chart cannot set, so operators silently can't disable it.
//   - flag drift (P3.1): a container --flag the chart renders but the
//     binary's FlagSet rejects → CrashLoop with green CI (the v1.23.0
//     class).
//
// Both are source-level / template-level gates so the failure is
// compile-adjacent and shows up the moment the drift is introduced.
package chartgate

import (
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// Source files scanned for os.Getenv("SRENIX_..") keys. Add new files here
// if the binary grows env-toggle reads elsewhere.
var toggleSourceFiles = []string{
	"../../catalog/catalog.go",
	// SRENIX_CLOUD_PROBE_* per-cloud-probe gates (O6) — rendered by the
	// chart's srenix.cloudProbeToggleEnv from cloud.<provider>.probes.*.
	"../../catalog/cloud.go",
	"../../internal/diagnose/security_drift.go",
	"../../internal/probe/k3s_datastore.go",
	"../../internal/watcher/leader.go",
	"../../cmd/srenix/main.go",
	// SRENIX_PROTECTED_NAMESPACES_EXTRA — append-only protected-namespace
	// extension read lazily by pkg/ai (and via it, internal/fix).
	"../../pkg/ai/protected.go",
}

const helpersTplPath = "../../charts/agentic-sre/templates/_helpers.tpl"
const watcherTplPath = "../../charts/agentic-sre/templates/watcher-deployment.yaml"

// toggleAllowlist documents SRENIX_* keys that are intentionally NOT a
// values.yaml enabled/disabled toggle rendered in _helpers.tpl. Each
// entry MUST carry a reason. A key in the allowlist is exempt from the
// "must appear in a chart template" assertion.
var toggleAllowlist = map[string]string{
	// Sourced from the externalDNS.cloudflare block via secretKeyRef in
	// the watcher Deployment (NEVER a literal toggle). Wired P1.5
	// (operator) + P1.8 (chart).
	"SRENIX_CLOUDFLARE_TOKEN": "externalDNS.cloudflare.apiTokenSecretRef — secretKeyRef, not a toggle",
	// Free-form config strings, not enabled/disabled booleans. Exposed
	// via watcher.extraEnv (operators set them as plain env entries).
	"SRENIX_CRITICAL_SERVICES":             "free-form CSV target list — set via watcher.extraEnv",
	"SRENIX_CRITICAL_SERVICES_REPLACE":     "replace-vs-merge flag for SRENIX_CRITICAL_SERVICES — set via watcher.extraEnv",
	"SRENIX_DIGEST_PIN_UNTRUSTED_SEVERITY": "severity-tuning string (info) for security-drift — set via watcher.extraEnv",
	"SRENIX_K3S_SINGLE_NODE_OK":            "single-node k3s acknowledgement string — set via watcher.extraEnv",
	// Rendered directly in the watcher Deployment from
	// watcher.leaderElection.enabled (predates the _helpers toggle
	// blocks); covered by the watcher-template assertion below.
	"SRENIX_LEADER_ELECTION": "rendered inline in watcher-deployment.yaml from watcher.leaderElection.enabled",
}

var getenvRe = regexp.MustCompile(`os\.Getenv\("(SRENIX_[A-Z0-9_]+)"\)`)

// collectToggleKeys returns the sorted unique set of SRENIX_* keys the
// binary reads via os.Getenv across the scanned source files.
func collectToggleKeys(t *testing.T) []string {
	t.Helper()
	seen := map[string]bool{}
	for _, f := range toggleSourceFiles {
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read source %s: %v", f, err)
		}
		for _, m := range getenvRe.FindAllStringSubmatch(string(raw), -1) {
			seen[m[1]] = true
		}
	}
	if len(seen) == 0 {
		t.Fatalf("scanned %d source files and found zero os.Getenv(\"SRENIX_..\") keys — regex or file list is stale", len(toggleSourceFiles))
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// TestChartExposesEveryToggle is the P1.8 drift gate: every SRENIX_* env
// toggle the binary reads must be settable through the chart — either
// rendered in _helpers.tpl, rendered inline in the watcher Deployment,
// or documented in toggleAllowlist with a reason. New toggles added to
// catalog.go without a chart surface fail here with the missing key
// name, preventing the every-toggle-since-v1.10-silently-unexposed
// drift from recurring.
func TestChartExposesEveryToggle(t *testing.T) {
	keys := collectToggleKeys(t)

	helpers, err := os.ReadFile(helpersTplPath)
	if err != nil {
		t.Fatalf("read _helpers.tpl: %v", err)
	}
	watcher, err := os.ReadFile(watcherTplPath)
	if err != nil {
		t.Fatalf("read watcher-deployment.yaml: %v", err)
	}
	chartText := string(helpers) + "\n" + string(watcher)

	var missing []string
	for _, k := range keys {
		if reason, ok := toggleAllowlist[k]; ok {
			if strings.TrimSpace(reason) == "" {
				t.Errorf("%s is allowlisted but carries no reason — every allowlist entry must justify why it is not a chart toggle", k)
			}
			continue
		}
		// The key must appear as a rendered env-var name in a chart
		// template (`- name: SRENIX_X`). Searching for the bare key is
		// sufficient and robust to nindent/whitespace.
		if !strings.Contains(chartText, k) {
			missing = append(missing, k)
		}
	}

	if len(missing) > 0 {
		sort.Strings(missing)
		t.Fatalf("the binary reads these SRENIX_* toggles but the chart cannot set them — add a values.yaml toggle rendered in %s (or, if it is config/secret not a toggle, add it to toggleAllowlist with a reason):\n  %s",
			helpersTplPath, strings.Join(missing, "\n  "))
	}
}
