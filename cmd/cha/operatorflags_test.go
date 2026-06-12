// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"strings"
	"testing"

	chav1alpha1 "github.com/Bionic-AI-Solutions/cluster-health-autopilot/api/v1alpha1"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/chartgate"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/operator"
	batchv1 "k8s.io/api/batch/v1"
	"sigs.k8s.io/yaml"
)

// O9 — operator-rendered args ↔ binary FlagSet parity gate.
//
// The sibling gate in chartflags_test.go covers the HELM render path,
// but the operator builders render the same workloads independently —
// and v1.x shipped buildCronJobCommon appending the watch-only flags
// (--alertmanager-url, --cluster-name, --slack-alerts,
// --slack-critical) to the diagnose/remediate CronJobs, which exited 1
// with "unknown flag" on every run. No test compared operator-rendered
// args against the real cobra FlagSets, so CI stayed green.
//
// This gate builds the watcher Deployment + diagnose/remediate
// CronJobs from a MAXIMAL CR (the bundle's full-surface sample with
// every relevant feature force-enabled) and asserts every rendered arg
// parses against the actual `cha <sub>` cobra command built in-process
// via newRootCmd(). That kills the bug CLASS: any future builder that
// emits a flag the subcommand does not register fails here, not in
// production CronJob logs.

// maximalOperatorCR decodes bundle/tests/sample-cr-full.yaml (the
// full-surface CR the bundle-smoke gate maintains — every optional
// subtree populated) and force-enables the arg-bearing features so the
// builders emit their maximal flag surface.
func maximalOperatorCR(t *testing.T) *chav1alpha1.ClusterHealthAutopilot {
	t.Helper()
	// Resolved through the shared chartgate locator so a future
	// bundle/tests move fails loudly in ONE place with a clear message.
	path := chartgate.SampleCRFullPath(t)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read full-surface sample CR: %v", err)
	}
	var cr chav1alpha1.ClusterHealthAutopilot
	if err := yaml.UnmarshalStrict(raw, &cr); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	if cr.Spec.Watcher == nil || cr.Spec.Diagnose == nil || cr.Spec.Remediate == nil ||
		cr.Spec.Alerting == nil || cr.Spec.Ticketing == nil {
		t.Fatalf("%s no longer populates watcher/diagnose/remediate/alerting/ticketing — this gate needs the full surface", path)
	}
	cr.Spec.Watcher.Enabled = true
	cr.Spec.Diagnose.Enabled = true
	cr.Spec.Remediate.Enabled = true
	cr.Spec.Remediate.DryRun = true
	cr.Spec.Ticketing.Enabled = true
	return &cr
}

func TestOperatorBuilderArgs_MatchBinaryFlagSet(t *testing.T) {
	cr := maximalOperatorCR(t)
	root := newRootCmd()

	watcher := operator.BuildWatcherDeployment(cr)
	if watcher == nil {
		t.Fatal("watcher deployment must build")
	}
	cases := []struct {
		sub  string
		args []string
	}{
		{"watch", watcher.Spec.Template.Spec.Containers[0].Args},
		{"diagnose", cronJobArgs(t, operator.BuildDiagnoseCronJob(cr))},
		{"remediate", cronJobArgs(t, operator.BuildRemediateCronJob(cr))},
	}

	for _, tc := range cases {
		cmd := findSub(root, tc.sub)
		if cmd == nil {
			t.Fatalf("cha has no %q subcommand — newRootCmd() shape changed", tc.sub)
		}
		if len(tc.args) == 0 || tc.args[0] != tc.sub {
			t.Errorf("operator-rendered args for %q must start with the subcommand; got %v", tc.sub, tc.args)
			continue
		}
		valid := flagSetNames(cmd)
		for _, a := range tc.args[1:] {
			name := renderedFlagName(a)
			if name == "" {
				t.Errorf("operator renders positional arg %q on the %q container — only --flags are valid after the subcommand", a, tc.sub)
				continue
			}
			if rootInheritedFlags[name] {
				continue
			}
			if !valid[name] {
				t.Errorf("operator renders --%s on the %q container, but `cha %s` does NOT register it — the Job exits 1 with \"unknown flag\" on every run (the v1.26.0 CronJob bug class). Fix internal/operator/builders.go or register the flag.",
					name, tc.sub, tc.sub)
			}
		}
	}
}

// cronJobArgs extracts the single container's args from a built CronJob.
func cronJobArgs(t *testing.T, c *batchv1.CronJob) []string {
	t.Helper()
	if c == nil {
		t.Fatal("CronJob must build (feature enabled on the maximal CR)")
	}
	return c.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Args
}

// renderedFlagName strips "--name=value" / "--name" to "name".
// Returns "" for positional args.
func renderedFlagName(arg string) string {
	if !strings.HasPrefix(arg, "--") {
		return ""
	}
	return strings.SplitN(strings.TrimPrefix(arg, "--"), "=", 2)[0]
}
