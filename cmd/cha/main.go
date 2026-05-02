// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

// cha — Cluster Health Autopilot CLI
//
// Subcommands (v0.1.x):
//
//	cha diagnose --snapshot <path>   # zero-trust offline mode (no cluster access)
//	cha diagnose --live              # live cluster mode (uses kubeconfig or in-cluster SA)
//	cha snapshot capture             # wraps `kubectl get` into a kubectl-shaped JSON bundle
//	cha remediate --live             # runs the whitelisted fixers (live mode only)
//	cha version                      # version info
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/diagnose"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/fix"
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
               No install, no RBAC, no write permissions, no cluster access.
  --live       live mode using kubeconfig or in-cluster ServiceAccount.

Subcommands:
  diagnose     run probes + analyzers (read-only; works in both modes)
  snapshot     capture cluster state for offline diagnose
  remediate    run the whitelisted auto-fixers (live mode only)`,
		SilenceUsage: true,
	}

	root.AddCommand(diagnoseCmd())
	root.AddCommand(snapshotCmd())
	root.AddCommand(remediateCmd())
	root.AddCommand(versionCmd())

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func snapshotCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "snapshot",
		Short: "Capture a cluster snapshot for offline diagnose",
	}
	c.AddCommand(snapshotCaptureCmd())
	return c
}

func snapshotCaptureCmd() *cobra.Command {
	var (
		outDir     string
		outTar     string
		kubeconfig string
		quiet      bool
	)
	c := &cobra.Command{
		Use:   "capture",
		Short: "Capture cluster state into a directory or tarball for `cha diagnose --snapshot`",
		Long: `Captures the canonical resource set required by cha diagnose offline:
pods, events, deployments, replicasets, jobs, cronjobs, nodes, pvcs,
externalsecrets, clusters.postgresql.cnpg.io, cephclusters.ceph.rook.io.

Reads only — never modifies cluster state. Output matches kubectl get -o json
shape so the same files round-trip back through cha diagnose --snapshot.`,
		Example: `  # Capture into a directory
  cha snapshot capture --out ./snapshot

  # Capture into a tarball (single artifact, easy to share)
  cha snapshot capture --tar snapshot.tgz

  # Then diagnose offline
  cha diagnose --snapshot ./snapshot`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if (outDir == "" && outTar == "") || (outDir != "" && outTar != "") {
				return fmt.Errorf("specify exactly one of --out or --tar")
			}
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			src, err := snapshot.LoadLive(kubeconfig)
			if err != nil {
				return err
			}
			var summary *snapshot.CaptureSummary
			if outTar != "" {
				summary, err = snapshot.CaptureTarGZ(ctx, src, outTar)
			} else {
				summary, err = snapshot.Capture(ctx, src, outDir)
			}
			if err != nil {
				return err
			}
			if !quiet {
				printCaptureSummary(summary, outTar != "")
			}
			return nil
		},
	}
	c.Flags().StringVar(&outDir, "out", "", "Output directory (mutually exclusive with --tar)")
	c.Flags().StringVar(&outTar, "tar", "", "Output tarball path (mutually exclusive with --out)")
	c.Flags().StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig (default: in-cluster, then $KUBECONFIG, then ~/.kube/config)")
	c.Flags().BoolVar(&quiet, "quiet", false, "Suppress per-resource summary output")
	return c
}

func printCaptureSummary(s *snapshot.CaptureSummary, isTar bool) {
	if isTar {
		fmt.Printf("Wrote tarball: %s\n", s.OutDir)
	} else {
		fmt.Printf("Wrote snapshot to: %s\n", s.OutDir)
	}
	fmt.Println(repeatRune('-', 60))
	totalItems := 0
	skipped := 0
	for _, item := range s.Items {
		if item.SkipErr != "" {
			fmt.Printf("  ⚠️  %-60s skipped: %s\n", item.GVR, item.SkipErr)
			skipped++
			continue
		}
		fmt.Printf("  ✅  %-60s %4d item(s)\n", item.GVR, item.Items)
		totalItems += item.Items
	}
	fmt.Println(repeatRune('-', 60))
	fmt.Printf("Total: %d resources, %d items captured", len(s.Items)-skipped, totalItems)
	if skipped > 0 {
		fmt.Printf(" (%d skipped)", skipped)
	}
	fmt.Println()
}

func remediateCmd() *cobra.Command {
	var (
		live         bool
		kubeconfig   string
		outputFormat string
		dryRun       bool
	)
	c := &cobra.Command{
		Use:   "remediate",
		Short: "Run the whitelisted auto-fixers (live mode only)",
		Long: `Runs the whitelisted auto-remediation fixers against the live cluster.

Refuses to run against snapshots — fixers are live-only by design (the
type system enforces this; snapshot.File does not implement Mutator).

The current fixer set is intentionally small and reversible:
  - StaleErrorPods           delete Failed pods owned by Job or unowned
  - StuckJobsWithBadSecretRef delete a frozen CronJob Job so the cron respawns
  - StuckRSPods              kubectl rollout restart when stuck RS rev != live

Mutations forbidden by design (would need a human + git): edits to
Secrets, ConfigMaps, or any CRD.`,
		Example: `  cha remediate --live
  cha remediate --live --format json
  cha remediate --live --dry-run`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !live {
				return fmt.Errorf("remediate requires --live; fixers refuse to run against snapshots")
			}
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			src, err := snapshot.LoadLive(kubeconfig)
			if err != nil {
				return err
			}
			var mut snapshot.Mutator
			if !dryRun {
				mut = snapshot.AsMutator(src)
				if mut == nil {
					return fmt.Errorf("source does not support mutation; expected Live source")
				}
			}

			fixers := []fix.Fixer{
				fix.StaleErrorPods{},
				fix.StuckJobsWithBadSecretRef{},
				fix.StuckRSPods{},
			}
			results := make([]fix.Result, 0, len(fixers))
			for _, f := range fixers {
				results = append(results, f.Run(ctx, src, mut))
			}

			switch outputFormat {
			case "json":
				return printRemediateJSON(results, dryRun)
			case "text", "":
				printRemediateText(results, dryRun)
				return nil
			default:
				return fmt.Errorf("unknown --format %q (want json or text)", outputFormat)
			}
		},
	}
	c.Flags().BoolVar(&live, "live", false, "Run against the live cluster (required)")
	c.Flags().StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig (default: in-cluster, then $KUBECONFIG, then ~/.kube/config)")
	c.Flags().StringVar(&outputFormat, "format", "text", "Output format: text|json")
	c.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be done without mutating cluster state (each fixer reports Refused)")
	return c
}

func printRemediateJSON(results []fix.Result, dryRun bool) error {
	out := map[string]any{
		"version": version,
		"dryRun":  dryRun,
		"fixers":  results,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func printRemediateText(results []fix.Result, dryRun bool) {
	tag := "live"
	if dryRun {
		tag = "dry-run"
	}
	fmt.Printf("Cluster Health Autopilot — remediate (%s)\n", tag)
	fmt.Println(repeatRune('=', 60))
	totalActions := 0
	totalSkipped := 0
	for _, r := range results {
		fmt.Printf("\n• %s", r.Fixer)
		if r.Refused != "" {
			fmt.Printf(" — refused (%s)\n", r.Refused)
			continue
		}
		fmt.Printf(": %d action(s), %d skipped\n", len(r.Actions), len(r.Skipped))
		for _, a := range r.Actions {
			fmt.Printf("    🔧 %s [%s]\n", a.Description, a.Object)
			totalActions++
		}
		// Print only the first 5 skips per fixer to avoid drowning the output.
		shown := r.Skipped
		if len(shown) > 5 {
			shown = shown[:5]
		}
		for _, s := range shown {
			fmt.Printf("    ⏭️  %s — %s\n", s.Object, s.Reason)
		}
		if len(r.Skipped) > 5 {
			fmt.Printf("    … and %d more skipped\n", len(r.Skipped)-5)
		}
		totalSkipped += len(r.Skipped)
	}
	fmt.Println(repeatRune('=', 60))
	fmt.Printf("Total: %d action(s) applied, %d skipped\n", totalActions, totalSkipped)
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
				probe.Ceph{},
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
