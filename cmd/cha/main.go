// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

// cha — Cluster Health Autopilot CLI
//
// Subcommands (v0.1.x):
//
//	cha diagnose --snapshot <path>   # zero-trust offline mode (no cluster access)
//	cha diagnose --live              # live cluster mode (uses kubeconfig or in-cluster SA)
//	cha version                      # version info
//
// Future:
//
//	cha snapshot capture             # wraps `kubectl get` to produce a snapshot bundle
//	cha report --format slack|json   # post a structured report somewhere
//	cha remediate --live             # gated; runs the whitelisted fixers
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/diagnose"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/probe"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
	"github.com/spf13/cobra"
)

// version is overridden at build time via -ldflags "-X main.version=v0.1.0".
var version = "dev"

func main() {
	root := &cobra.Command{
		Use:   "cha",
		Short: "Cluster Health Autopilot — detect, fix, re-verify, report.",
		Long: `cha runs a battery of probes against your Kubernetes cluster, applies
a whitelist of known-safe fixes, re-verifies, and reports.

Two modes:
  --snapshot   zero-trust offline diagnose against a captured kubectl JSON export.
  --live       in-cluster live mode using kubeconfig or in-cluster SA.

In v0.1.x, fixers are not yet ported (live remediation is week-4 work).
This release is diagnose-only.`,
		SilenceUsage: true,
	}

	root.AddCommand(diagnoseCmd())
	root.AddCommand(versionCmd())

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version",
		RunE: func(_ *cobra.Command, _ []string) error {
			fmt.Println(version)
			return nil
		},
	}
}

func diagnoseCmd() *cobra.Command {
	var (
		snapshotPath string
		live         bool
		kubeconfig   string
		outputFormat string
	)
	c := &cobra.Command{
		Use:   "diagnose",
		Short: "Run probes + analyzers and print findings (read-only)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if (snapshotPath == "" && !live) || (snapshotPath != "" && live) {
				return fmt.Errorf("specify exactly one of --snapshot or --live")
			}

			var src snapshot.Source
			var err error
			if snapshotPath != "" {
				src, err = snapshot.LoadFile(snapshotPath)
			} else {
				src, err = snapshot.LoadLive(kubeconfig)
			}
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			probes := []probe.Probe{
				probe.Nodes{},
				probe.Postgres{},
				probe.PVCs{},
				probe.Services{Targets: probe.DefaultTargets()},
			}

			results := make([]probe.Result, 0, len(probes))
			for _, p := range probes {
				results = append(results, p.Run(ctx, src))
			}

			analyzers := []diagnose.Analyzer{
				diagnose.SecretKeyMissing{},
				diagnose.FailingExternalSecrets{},
			}
			var diagnostics []diagnose.Diagnostic
			for _, a := range analyzers {
				diagnostics = append(diagnostics, a.Run(ctx, src)...)
			}

			switch outputFormat {
			case "json":
				return printJSON(results, diagnostics)
			case "text", "":
				return printText(results, diagnostics, src.Mode())
			default:
				return fmt.Errorf("unknown --format %q (want json or text)", outputFormat)
			}
		},
	}
	c.Flags().StringVar(&snapshotPath, "snapshot", "", "Path to captured kubectl JSON export (file or directory)")
	c.Flags().BoolVar(&live, "live", false, "Run against the live cluster")
	c.Flags().StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig (default: in-cluster, then $KUBECONFIG, then ~/.kube/config)")
	c.Flags().StringVar(&outputFormat, "format", "text", "Output format: text|json")
	return c
}

func printJSON(results []probe.Result, diagnostics []diagnose.Diagnostic) error {
	out := map[string]any{
		"version":     version,
		"results":     results,
		"diagnostics": diagnostics,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func printText(results []probe.Result, diagnostics []diagnose.Diagnostic, mode snapshot.Mode) error {
	fmt.Printf("Cluster Health Autopilot — diagnose (%s mode)\n", mode)
	fmt.Println(repeatRune('=', 60))
	totalFindings := 0
	for _, r := range results {
		fmt.Printf("\n• %s: %s\n", r.Component.Component, statusIcon(r.Component.Status))
		fmt.Printf("  %s\n", r.Component.Detail)
		for _, f := range r.Findings {
			totalFindings++
			fmt.Printf("    %s [%s] %s\n", severityIcon(f.Severity), f.Component, f.Message)
			if f.Remediation != "" {
				fmt.Printf("      → %s\n", f.Remediation)
			}
		}
	}
	if len(diagnostics) > 0 {
		fmt.Printf("\nDiagnostics (%d):\n", len(diagnostics))
		for _, d := range diagnostics {
			fmt.Printf("  🔎 %s\n", d.Message)
		}
	}
	fmt.Println(repeatRune('=', 60))
	fmt.Printf("Total findings: %d, diagnostics: %d\n", totalFindings, len(diagnostics))
	return nil
}

func statusIcon(s string) string {
	switch s {
	case "HEALTHY":
		return "🟢 HEALTHY"
	case "DEGRADED":
		return "🟡 DEGRADED"
	case "CRITICAL":
		return "🔴 CRITICAL"
	case "PROBE_FAILED":
		return "🔴 PROBE_FAILED"
	case "SKIPPED":
		return "⚪ SKIPPED"
	default:
		return s
	}
}

func severityIcon(s probe.Severity) string {
	switch s {
	case probe.SeverityCritical:
		return "❌"
	case probe.SeverityWarning:
		return "⚠️"
	default:
		return "ℹ️"
	}
}

func repeatRune(r rune, n int) string {
	out := make([]rune, n)
	for i := range out {
		out[i] = r
	}
	return string(out)
}
