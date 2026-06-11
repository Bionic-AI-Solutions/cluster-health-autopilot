// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

// default_off_test.go (P3.3a) — default-off discipline gate.
//
// # WHY THIS EXISTS
//
// Three same-week production regressions traced back to one abandoned habit:
// the roadmap's own done-criteria said new triggers ship DEFAULT-OFF, soak for
// 7 days, then flip to default-on the next release. v1.23.0's M5/M6 work shipped
// unwired AND default-on; the moment a new trigger was registered default-on it
// fired against every cluster with no soak. There is no PR-diff available inside
// a unit test, so we approximate the gate with a committed GOLDEN of every
// catalog registration toggle and its default polarity.
//
// # HOW IT WORKS
//
// The test scans catalog/catalog.go for every CHA_* env toggle that GATES a
// probe / analyzer / fixer / investigator registration, derives its default
// polarity from the comparison operator:
//
//	os.Getenv("CHA_X") != "off"   → default-ON  ("on")   registered unless opted out
//	os.Getenv("CHA_X") == "true"  → default-OFF ("off")  registered only when enabled
//
// and compares the derived map against testdata/toggle_defaults.golden. When a
// NET-NEW toggle is added, or an existing toggle's polarity FLIPS, the test
// FAILS and tells the developer to:
//
//  1. confirm the new toggle ships DEFAULT-OFF unless a soak rationale is
//     recorded (the discipline), and
//  2. update the golden consciously.
//
// This makes a default-on addition a REVIEWED act, not a silent one.
//
// GRANDFATHERING: the golden is seeded from the current (pre-discipline) state,
// so the many existing default-on toggles are grandfathered. The gate only
// catches net-new additions and polarity flips — it does NOT retroactively
// demand the existing inventory be default-off.
package chartgate

import (
	"bufio"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
)

const catalogPath = "../../catalog/catalog.go"
const toggleGoldenPath = "testdata/toggle_defaults.golden"

// Matches a registration-gating toggle and captures the key + the comparison
// operator/value, e.g.:
//
//	if os.Getenv("CHA_PROBE_ETCD") != "off" {
//	if os.Getenv("CHA_FIXER_TLS_SECRET_MISMATCH") == "true" {
//
// Toggles read for config behavior (CHA_CRITICAL_SERVICES_REPLACE,
// CHA_K3S_SINGLE_NODE_OK) do NOT gate a registration with this shape and are
// intentionally out of scope — they are documented config strings in the
// toggle_drift_test.go allowlist, not trigger registrations.
var registrationToggleRe = regexp.MustCompile(
	`os\.Getenv\("(CHA_[A-Z0-9_]+)"\)\s*(!=|==)\s*"(off|true)"`,
)

// nonRegistrationToggles are CHA_* keys read in catalog.go with a
// boolean-comparison shape but that modify CONFIG BEHAVIOR rather than gate a
// trigger registration. They are out of scope for the default-off discipline
// (they register nothing) and so are excluded from both the derived map and
// the golden. Each carries a reason. These mirror the free-form config entries
// in toggle_drift_test.go's allowlist.
var nonRegistrationToggles = map[string]string{
	"CHA_CRITICAL_SERVICES_REPLACE": "replace-vs-merge flag for the CHA_CRITICAL_SERVICES target list — config, not a registration gate",
}

// deriveToggleDefaults scans catalog.go and returns key → "on"/"off".
func deriveToggleDefaults(t *testing.T) map[string]string {
	t.Helper()
	raw, err := os.ReadFile(catalogPath)
	if err != nil {
		t.Fatalf("read %s: %v", catalogPath, err)
	}
	out := map[string]string{}
	for _, m := range registrationToggleRe.FindAllStringSubmatch(string(raw), -1) {
		key, op, val := m[1], m[2], m[3]
		if reason, skip := nonRegistrationToggles[key]; skip {
			if strings.TrimSpace(reason) == "" {
				t.Errorf("%s is excluded but carries no reason", key)
			}
			continue
		}
		var polarity string
		switch {
		case op == "!=" && val == "off":
			polarity = "on" // registered unless opted out → default-on
		case op == "==" && val == "true":
			polarity = "off" // registered only when enabled → default-off
		default:
			t.Fatalf("%s: unrecognised toggle comparison %q %q %q — extend the polarity switch", key, key, op, val)
		}
		if existing, ok := out[key]; ok && existing != polarity {
			t.Fatalf("%s read with conflicting polarities (%s vs %s) in catalog.go", key, existing, polarity)
		}
		out[key] = polarity
	}
	if len(out) == 0 {
		t.Fatalf("found zero registration toggles in %s — regex is stale", catalogPath)
	}
	return out
}

// loadToggleGolden parses testdata/toggle_defaults.golden (skipping #-comments
// and blank lines) into key → "on"/"off".
func loadToggleGolden(t *testing.T) map[string]string {
	t.Helper()
	f, err := os.Open(toggleGoldenPath)
	if err != nil {
		t.Fatalf("open golden %s: %v", toggleGoldenPath, err)
	}
	defer f.Close()
	out := map[string]string{}
	sc := bufio.NewScanner(f)
	ln := 0
	for sc.Scan() {
		ln++
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 || (fields[1] != "on" && fields[1] != "off") {
			t.Fatalf("%s:%d malformed golden line %q (want `CHA_X on|off`)", toggleGoldenPath, ln, line)
		}
		out[fields[0]] = fields[1]
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan golden: %v", err)
	}
	return out
}

// TestToggleDefaultsMatchGolden is the P3.3a discipline gate. See file header.
func TestToggleDefaultsMatchGolden(t *testing.T) {
	derived := deriveToggleDefaults(t)
	golden := loadToggleGolden(t)

	const remedy = "\n\nDISCIPLINE: a NET-NEW probe/analyzer/fixer toggle should ship DEFAULT-OFF " +
		"(`os.Getenv(\"CHA_X\") == \"true\"`), soak ~7 days, then flip to default-on next release. " +
		"If you are adding a toggle: make it default-off unless a soak rationale is recorded, then update " +
		toggleGoldenPath + ". If you intentionally flipped polarity: confirm the soak completed and update the golden."

	// Net-new toggles (in code, absent from golden).
	var added []string
	for k := range derived {
		if _, ok := golden[k]; !ok {
			added = append(added, k)
		}
	}
	// Removed toggles (in golden, absent from code).
	var removed []string
	for k := range golden {
		if _, ok := derived[k]; !ok {
			removed = append(removed, k)
		}
	}
	// Polarity flips.
	var flipped []string
	for k, dv := range derived {
		if gv, ok := golden[k]; ok && gv != dv {
			flipped = append(flipped, k+": golden="+gv+" code="+dv)
		}
	}

	if len(added) > 0 {
		sort.Strings(added)
		t.Errorf("NEW catalog toggle(s) not in %s:\n  %s\nDefault polarity in code:%s",
			toggleGoldenPath, strings.Join(annotate(added, derived), "\n  "), remedy)
	}
	if len(flipped) > 0 {
		sort.Strings(flipped)
		t.Errorf("catalog toggle default polarity FLIPPED vs golden:\n  %s%s",
			strings.Join(flipped, "\n  "), remedy)
	}
	if len(removed) > 0 {
		sort.Strings(removed)
		t.Errorf("golden lists toggle(s) no longer found in catalog.go (removed or renamed) — update %s:\n  %s",
			toggleGoldenPath, strings.Join(removed, "\n  "))
	}
}

// annotate appends the derived polarity to each key for a readable failure.
func annotate(keys []string, derived map[string]string) []string {
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, k+" (default-"+derived[k]+")")
	}
	return out
}
