// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"sort"
	"strings"
	"testing"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/chartgate"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// P3.1 — operator leg of the chart-args↔binary-flags parity gate.
//
// The operator Deployment renders --leader-elect / --metrics-bind-address
// / --health-probe-bind-address. This gate renders the operator
// container args from `helm template` (maximal values, operator.enabled)
// and asserts every flag is registered on the real operator FlagSet —
// built via registerOperatorFlags() + zap.Options.BindFlags(), the exact
// pair main() uses. A template that renders a flag the binary doesn't
// register would CrashLoop the operator pod with green CI without this.
func TestOperatorChartFlags_MatchBinaryFlagSet(t *testing.T) {
	roleArgs := chartgate.RenderMaximalArgs(t)

	args, ok := roleArgs["operator"]
	if !ok {
		t.Fatalf("maximal helm render produced no operator container — expected operator.enabled to render the operator Deployment; update maximal values or chartgate render")
	}

	// Rebuild the operator's real flag surface: the manager flags plus
	// the zap logger flags, exactly as main() binds them.
	fs := flag.NewFlagSet("cha-operator", flag.ContinueOnError)
	registerOperatorFlags(fs)
	opts := zap.Options{}
	opts.BindFlags(fs)

	valid := map[string]bool{}
	fs.VisitAll(func(f *flag.Flag) { valid[f.Name] = true })

	var unknown []string
	for _, a := range args {
		if !valid[a] {
			unknown = append(unknown, a)
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		t.Errorf("chart renders --%s on the operator container, but cha-operator does NOT register it — v1.23.0 CrashLoop class (pod fails with \"flag provided but not defined\"). Register the flag in registerOperatorFlags() or fix the template.",
			strings.Join(unknown, ", --"))
	}
}
