// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"sort"
	"strings"
	"testing"

	"github.com/srenix-ai/agentic-sre/internal/chartgate"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// P3.1 — chart-args ↔ binary-flags parity gate (the v1.23.0 class).
//
// v1.23.0 shipped a chart that rendered a container --flag the binary
// did not register. The container CrashLooped on startup ("unknown
// flag"), but CI was green: no test rendered the chart AND checked the
// flags against the real FlagSet. This gate renders the watcher +
// diagnose + remediate container args from `helm template` with a
// maximal values file and asserts every --flag parses against the
// actual cobra command's FlagSet (built in-process via newRootCmd()).
//
// The operator binary's flags are covered by the sibling gate in
// cmd/srenix-operator/chartflags_test.go (separate main package).

// chaSubcommandFlags maps the workload role label to the srenix subcommand
// whose FlagSet its container args must satisfy.
var chaSubcommandRole = map[string]string{
	"watcher":   "watch",
	"diagnose":  "diagnose",
	"remediate": "remediate",
}

// rootInheritedFlags are persistent/global flags valid on every
// subcommand (currently none beyond cobra's built-in --help, which the
// chart never renders). Kept as an explicit allowlist so a future
// global flag is a one-line addition, not a silent gap.
var rootInheritedFlags = map[string]bool{
	"help": true,
}

func TestChartFlags_MatchBinaryFlagSet(t *testing.T) {
	roleArgs := chartgate.RenderMaximalArgs(t)

	root := newRootCmd()

	for role, sub := range chaSubcommandRole {
		flags, ok := roleArgs[role]
		if !ok {
			t.Errorf("maximal helm render produced no container for role %q — expected the %q subcommand workload; update the maximal values or chartgate render", role, sub)
			continue
		}
		cmd := findSub(root, sub)
		if cmd == nil {
			t.Fatalf("srenix has no %q subcommand — newRootCmd() shape changed", sub)
		}
		valid := flagSetNames(cmd)

		var unknown []string
		for _, f := range flags {
			if rootInheritedFlags[f] {
				continue
			}
			if !valid[f] {
				unknown = append(unknown, f)
			}
		}
		if len(unknown) > 0 {
			sort.Strings(unknown)
			t.Errorf("chart renders --%s on the %q container, but `srenix %s` does NOT register %s — this is the v1.23.0 CrashLoop class (the pod fails with \"unknown flag\" at startup). Register the flag on the cobra command or fix the template.",
				strings.Join(unknown, ", --"), role, sub, pluralFlag(unknown))
		}
	}
}

// findSub returns the named direct subcommand of root, or nil.
func findSub(root *cobra.Command, name string) *cobra.Command {
	for _, c := range root.Commands() {
		if c.Name() == name {
			return c
		}
	}
	return nil
}

// flagSetNames returns the set of flag names a cobra command accepts:
// its own flags plus inherited persistent flags from ancestors.
func flagSetNames(cmd *cobra.Command) map[string]bool {
	out := map[string]bool{}
	cmd.Flags().VisitAll(func(f *pflag.Flag) { out[f.Name] = true })
	cmd.InheritedFlags().VisitAll(func(f *pflag.Flag) { out[f.Name] = true })
	cmd.PersistentFlags().VisitAll(func(f *pflag.Flag) { out[f.Name] = true })
	return out
}

func pluralFlag(s []string) string {
	if len(s) == 1 {
		return "this flag"
	}
	return "these flags"
}
