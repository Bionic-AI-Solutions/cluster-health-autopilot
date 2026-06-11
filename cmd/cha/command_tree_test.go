// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
)

// P2.4 — command-tree construction + --help smoke for the cha binary.
//
// Class of bug: CHA-com v1.20.0 shipped a `panic: flag redefined:
// cluster-name` that only fired once a subcommand was attached to the
// root the way main() builds it. No test constructed the full root
// command, so it shipped uncaught. This is the OSS leg of the same
// guard.
//
// newRootCmd() is the single construction path main() uses (P3.1
// extracted it; chartflags_test.go already builds through it). This
// test rebuilds that tree and runs --help on the root AND every
// subcommand:
//
//   - a duplicate-flag panic (the v1.20.0 class) fails at construction
//     of newRootCmd(), before any --help runs.
//   - a subcommand with broken flag wiring / help template fails when
//     --help is executed against it.
//
// Because it goes through newRootCmd(), a future subcommand added in
// main() is automatically covered.

func TestCommandTree_RootConstructsWithoutPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("newRootCmd() panicked at construction (the v1.20.0 flag-redefine class): %v", r)
		}
	}()
	root := newRootCmd()
	if root == nil {
		t.Fatal("newRootCmd() returned nil")
	}
	if len(root.Commands()) == 0 {
		t.Fatal("root has no subcommands — newRootCmd() shape changed")
	}
}

func TestCommandTree_HelpOnRootAndEverySubcommand(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("command-tree construction panicked (the v1.20.0 flag-redefine class): %v", r)
		}
	}()

	// --help on the root.
	runHelp(t, newRootCmd(), nil)

	// --help on every subcommand, including nested ones (e.g. snapshot
	// has a capture child). Walk the tree so a broken grandchild can't
	// hide. Rebuild from newRootCmd() per command for a clean flag state.
	for _, path := range subcommandPaths(newRootCmd(), nil) {
		path := path
		t.Run(joinPath(path), func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("subcommand %q panicked on construction/--help: %v", joinPath(path), r)
				}
			}()
			runHelp(t, newRootCmd(), path)
		})
	}
}

// subcommandPaths returns the arg path for every (transitively nested)
// subcommand under root, excluding cobra's built-in help command.
func subcommandPaths(cmd *cobra.Command, prefix []string) [][]string {
	var out [][]string
	for _, c := range cmd.Commands() {
		if c.Name() == "help" {
			continue
		}
		path := append(append([]string{}, prefix...), c.Name())
		out = append(out, path)
		out = append(out, subcommandPaths(c, path)...)
	}
	return out
}

func joinPath(path []string) string {
	if len(path) == 0 {
		return "root"
	}
	s := path[0]
	for _, p := range path[1:] {
		s += "/" + p
	}
	return s
}

// runHelp executes `<path...> --help` against root, asserting no error.
// --help is a pure construct-and-render path: it never touches the
// cluster, so it exercises flag registration + help templating for the
// targeted command without side effects.
func runHelp(t *testing.T, root *cobra.Command, path []string) {
	t.Helper()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(append(append([]string{}, path...), "--help"))
	if err := root.Execute(); err != nil {
		t.Fatalf("`%s --help` failed: %v\noutput:\n%s", joinPath(path), err, out.String())
	}
}
