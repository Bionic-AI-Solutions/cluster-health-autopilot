// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"testing"

	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// P2.4 — operator leg of the command-tree/flag construction smoke.
//
// Class of bug: Srenix Enterprise v1.20.0 shipped a `panic: flag redefined`
// that only fired once flags were registered the way main() registers
// them — no test reproduced main()'s construction. The srenix-operator has
// no cobra subcommand tree; its "command tree" is the flag.FlagSet that
// main() builds via registerOperatorFlags(fs) + zap.Options.BindFlags(fs).
// A duplicate flag (own registration colliding with a zap flag, or a
// double registerOperatorFlags) panics flag.FlagSet at startup and
// CrashLoops the operator pod.
//
// This test constructs that exact flag surface and then runs the --help
// path against it (flag.ErrHelp on a ContinueOnError FlagSet), proving:
//
//   - construction (registerOperatorFlags + BindFlags) does not panic on
//     a duplicate/redefined flag.
//   - parsing --help renders usage without error other than the expected
//     flag.ErrHelp — i.e. every registered flag has a valid definition.

func TestOperatorFlags_ConstructWithoutPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("operator flag construction panicked (the v1.20.0 flag-redefine class): %v", r)
		}
	}()
	fs := flag.NewFlagSet("srenix-operator", flag.ContinueOnError)
	registerOperatorFlags(fs)
	opts := zap.Options{}
	opts.BindFlags(fs)

	if fs.Lookup("metrics-bind-address") == nil {
		t.Error("--metrics-bind-address should be registered")
	}
	if fs.Lookup("leader-elect") == nil {
		t.Error("--leader-elect should be registered")
	}
}

func TestOperatorFlags_HelpSmoke(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("operator flag construction panicked (the v1.20.0 flag-redefine class): %v", r)
		}
	}()

	// Build the exact surface main() builds, then exercise the --help
	// render path. ContinueOnError + a discard output keeps the test
	// quiet; --help returns flag.ErrHelp, which is the expected, healthy
	// outcome (a broken flag def would panic during construction above,
	// or error differently here).
	fs := flag.NewFlagSet("srenix-operator", flag.ContinueOnError)
	fs.SetOutput(discardWriter{})
	registerOperatorFlags(fs)
	opts := zap.Options{}
	opts.BindFlags(fs)

	err := fs.Parse([]string{"--help"})
	if err != nil && err != flag.ErrHelp {
		t.Fatalf("`srenix-operator --help` produced unexpected error: %v", err)
	}
}

// discardWriter swallows the FlagSet usage output during the help smoke.
type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }
