// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

// cha — Cluster Health Autopilot CLI
//
// Subcommands:
//
//	cha diagnose --snapshot <path>   # zero-trust offline mode (no cluster access)
//	cha diagnose --live              # live cluster mode (uses kubeconfig or in-cluster SA)
//	cha snapshot capture             # wraps `kubectl get` into a kubectl-shaped JSON bundle
//	cha remediate --live             # runs the whitelisted auto-fixers (live mode only)
//	cha anonymize [file...]          # anonymize cha diagnose --format json output to JSONL
//	cha version                      # version info
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/catalog"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/anonymize"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/diagnose"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/fix"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/probe"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/report"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/summarize"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/vault"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/watcher"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/ticketing/openproject"
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
	root.AddCommand(watchCmd())
	root.AddCommand(snapshotCmd())
	root.AddCommand(remediateCmd())
	root.AddCommand(anonymizeCmd())
	root.AddCommand(summarizeCmd())
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
		slackWebhook string
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
			if dryRun {
				mut = snapshot.DryRunMutator{}
			} else {
				mut = snapshot.AsMutator(src)
				if mut == nil {
					return fmt.Errorf("source does not support mutation; expected Live source")
				}
			}

			fixers := catalog.Default().Fixers()
			results := make([]fix.Result, 0, len(fixers))
			for _, f := range fixers {
				results = append(results, f.Run(ctx, src, mut))
			}

			// Optional Slack post — summary of actions taken (read-only diagnose
			// is empty here since remediate doesn't probe; the diagnose
			// subcommand is the right surface for the full picture).
			if slackWebhook != "" {
				payload := report.FormatSlack(nil, nil, results, !dryRun)
				if err := report.PostSlack(nil, slackWebhook, payload); err != nil {
					fmt.Fprintln(os.Stderr, "warning: slack post failed:", err)
				}
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
	c.Flags().BoolVar(&dryRun, "dry-run", false, "Evaluate fixers and report what would be done without mutating cluster state")
	c.Flags().StringVar(&slackWebhook, "slack-webhook", "", "Optional Slack incoming-webhook URL — posts a summary of fixes applied")
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
			if dryRun {
				fmt.Printf("    [DRY-RUN] Would: %s [%s]\n", a.Description, a.Object)
			} else {
				fmt.Printf("    🔧 %s [%s]\n", a.Description, a.Object)
			}
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

// anonymizeCmd implements `cha anonymize [file...]`.
// Each input file must be a `cha diagnose --format json` run. When no files
// are given, JSON is read from stdin. Each run is written as one JSONL line
// to stdout with all PII fields hashed.
func anonymizeCmd() *cobra.Command {
	var runID string
	var ts string

	c := &cobra.Command{
		Use:   "anonymize [file...]",
		Short: "Anonymize cha diagnose --format json output to JSONL",
		Long: `anonymize reads one or more cha diagnose --format json files (or stdin)
and writes an anonymized JSONL record per run to stdout.

Namespace names, workload names, secret names, and Vault path segments are
replaced with deterministic short hashes (SHA-256 truncated to 8 hex chars).
The same token always produces the same hash, so time-series comparisons
across daily runs remain coherent.

Typical pipeline (daily cron + nightly GitHub Action):

  cha diagnose --live --format json > /tmp/run.json
  cha anonymize /tmp/run.json >> runs/$(date +%Y-%m-%d).jsonl`,
		RunE: func(cmd *cobra.Command, args []string) error {
			a := anonymize.New()
			enc := json.NewEncoder(os.Stdout)

			processReader := func(r io.Reader, name string) error {
				data, err := io.ReadAll(r)
				if err != nil {
					return fmt.Errorf("read %s: %w", name, err)
				}
				var in anonymize.RunInput
				if err := json.Unmarshal(data, &in); err != nil {
					return fmt.Errorf("parse %s: %w", name, err)
				}
				rid := runID
				if rid == "" {
					rid = filepath.Base(name)
				}
				stamp := ts
				if stamp == "" {
					stamp = time.Now().UTC().Format(time.RFC3339)
				}
				rec := a.Anonymize(in, rid, stamp)
				return enc.Encode(rec)
			}

			if len(args) == 0 {
				return processReader(bufio.NewReader(os.Stdin), "stdin")
			}
			for _, f := range args {
				fh, err := os.Open(f)
				if err != nil {
					return err
				}
				err = processReader(fh, f)
				_ = fh.Close()
				if err != nil {
					return err
				}
			}
			return nil
		},
	}
	c.Flags().StringVar(&runID, "run-id", "", "Run identifier stamped on each JSONL record (default: input filename)")
	c.Flags().StringVar(&ts, "timestamp", "", "RFC3339 timestamp to stamp on each record (default: now)")
	return c
}

// summarizeCmd implements `cha summarize <runs-dir>`.
func summarizeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "summarize <runs-dir>",
		Short: "Generate runs/SUMMARY.md from anonymized JSONL run records",
		Long: `summarize reads all *.jsonl files in <runs-dir>, aggregates the anonymized
run records, and writes a Markdown SUMMARY.md to stdout.

Typical usage (nightly GitHub Action):

  cha summarize runs/ > runs/SUMMARY.md`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			r, err := summarize.FromDir(args[0])
			if err != nil {
				return err
			}
			r.Render(os.Stdout)
			return nil
		},
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

func watchCmd() *cobra.Command {
	var (
		live                    bool
		kubeconfig              string
		debounce                time.Duration
		resyncPeriod            time.Duration
		slackAlerts             string
		slackCritical           string
		postOnResolved          bool
		repeatInterval          time.Duration
		writeDriftReports       bool
		remedy                  bool
		dryRun                  bool
		alertmanagerURL         string
		clusterName             string
		vaultAddr               string
		vaultMount              string
		vaultRole               string
		ticketingProvider       string
		ticketingMCPURL         string
		ticketingProject        string
		ticketingTypeID         string
		ticketingClosedStatusID string
		ticketingPriorityCrit   string
		ticketingPriorityWarn   string
		ticketingPriorityInfo   string
		ticketingWebURLPrefix   string
		ticketingLabels         []string
		ticketingDryRun         bool
	)
	c := &cobra.Command{
		Use:   "watch",
		Short: "Event-driven cluster health watcher (live mode only)",
		Long: `Starts Kubernetes watches for all resource types cha analyzes.

On relevant create/update/delete events a short debounce fires, then the
full probe+analyzer stack runs — identical to cha diagnose --live.

Slack posts are deduplicated by fingerprint: only new/changed/resolved
diagnostics trigger a post. The seen-map is seeded from existing DriftReport
CRs on startup to avoid re-flooding Slack after a pod restart.

With --remedy, fixers run after each diagnose cycle and the report reflects
the post-fix cluster state.`,
		Example: `  # Basic watcher
  cha watch --live

  # With Slack dedup + remediation
  cha watch --live --slack-webhook=$(SLACK_WEBHOOK_URL) --remedy

  # Tune debounce and repeat interval
  cha watch --live --debounce=15s --slack-repeat-interval=6h \
      --slack-webhook=$(SLACK_WEBHOOK_URL)`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !live {
				return fmt.Errorf("watch requires --live")
			}
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			lv, err := snapshot.LoadLive(kubeconfig)
			if err != nil {
				return err
			}

			reg := catalog.Default()
			if vaultAddr != "" {
				vc, verr := buildVaultClient(vaultAddr, vaultMount, vaultRole)
				if verr != nil {
					fmt.Fprintln(os.Stderr, "warning: vault client unavailable:", verr)
				} else {
					reg.RegisterAnalyzer(diagnose.VaultPathMissing{Client: vc})
				}
			}

			var mut snapshot.Mutator
			if remedy {
				if dryRun {
					mut = snapshot.DryRunMutator{}
				} else {
					m := snapshot.AsMutator(lv)
					if m == nil {
						return fmt.Errorf("source does not support mutation; expected Live source")
					}
					mut = m
				}
			}

			ticketingCfg, terr := buildTicketingConfig(ticketingOpts{
				Provider:       ticketingProvider,
				MCPURL:         ticketingMCPURL,
				ProjectID:      ticketingProject,
				TypeID:         ticketingTypeID,
				ClosedStatusID: ticketingClosedStatusID,
				PriorityCrit:   ticketingPriorityCrit,
				PriorityWarn:   ticketingPriorityWarn,
				PriorityInfo:   ticketingPriorityInfo,
				WebURLPrefix:   ticketingWebURLPrefix,
				Cluster:        clusterName,
				Labels:         ticketingLabels,
				DryRun:         ticketingDryRun,
			})
			if terr != nil {
				return terr
			}

			cfg := watcher.Config{
				Debounce:     debounce,
				ResyncPeriod: resyncPeriod,
				SlackChannels: report.SlackChannels{
					Alerts:   slackAlerts,
					Critical: slackCritical,
				},
				PostOnResolved:    postOnResolved,
				RepeatInterval:    repeatInterval,
				WriteDriftReports: writeDriftReports,
				RunRemediation:    remedy,
				DryRun:            dryRun,
				AlertmanagerURL:   alertmanagerURL,
				ClusterName:       clusterName,
				Ticketing:         ticketingCfg,
			}
			w := watcher.New(lv, reg, mut, cfg)
			return w.Run(ctx)
		},
	}
	c.Flags().BoolVar(&live, "live", false, "Run against the live cluster (required)")
	c.Flags().StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig (default: in-cluster, then $KUBECONFIG, then ~/.kube/config)")
	c.Flags().DurationVar(&debounce, "debounce", 10*time.Second, "Debounce window after a Kubernetes event before re-running diagnostics")
	c.Flags().DurationVar(&resyncPeriod, "resync-period", 10*time.Minute, "Full re-diagnose interval regardless of events (catches non-event drift)")
	c.Flags().StringVar(&alertmanagerURL, "alertmanager-url", os.Getenv("ALERTMANAGER_URL"), "Alertmanager base URL (e.g. http://alertmanager.pg:9093). When set, CHA posts all active issues as Prometheus alerts each cycle; AM handles routing to Slack/PagerDuty/etc.")
	c.Flags().StringVar(&clusterName, "cluster-name", envOrDefault("CLUSTER_NAME", "cluster"), "Cluster name stamped on Alertmanager alert labels (default: $CLUSTER_NAME or 'cluster')")
	c.Flags().StringVar(&slackAlerts, "slack-alerts", "", "Slack webhook for #ceph-alerts — CHA acted (auto-fixed issues); used as fallback when --alertmanager-url is not set")
	c.Flags().StringVar(&slackCritical, "slack-critical", "", "Slack webhook for #ceph-critical — human action required; used as fallback when --alertmanager-url is not set")
	c.Flags().BoolVar(&postOnResolved, "slack-post-on-resolved", true, "Post to Slack when a diagnostic resolves")
	c.Flags().DurationVar(&repeatInterval, "slack-repeat-interval", 4*time.Hour, "Re-post still-active diagnostics at this interval (0 = never repeat)")
	c.Flags().BoolVar(&writeDriftReports, "write-driftreports", true, "Upsert DriftReport CRs on every cycle (live mode only)")
	c.Flags().BoolVar(&remedy, "remedy", false, "Run auto-fixers after each diagnose cycle; post-fix state is reported")
	c.Flags().BoolVar(&dryRun, "dry-run", false, "With --remedy: evaluate fixers without applying changes")
	c.Flags().StringVar(&vaultAddr, "vault-addr", os.Getenv("VAULT_ADDR"), "Vault HTTP endpoint (default: $VAULT_ADDR)")
	c.Flags().StringVar(&vaultMount, "vault-kv-mount", envOrDefault("VAULT_KV_MOUNT", "secret"), "Vault KV-v2 mount path")
	c.Flags().StringVar(&vaultRole, "vault-k8s-role", os.Getenv("VAULT_K8S_ROLE"), "Vault kubernetes-auth role (default: $VAULT_K8S_ROLE)")
	c.Flags().StringVar(&ticketingProvider, "ticketing-provider", os.Getenv("TICKETING_PROVIDER"), "Issue tracker sink: 'openproject' (OSS). Empty disables ticketing. Jira/ServiceNow live in CHA-com.")
	c.Flags().StringVar(&ticketingMCPURL, "ticketing-mcp-url", envOrDefault("TICKETING_MCP_URL", "http://mcp-openproject-server.mcp.svc:8006/mcp"), "MCP server URL for the ticketing sink (Streamable-HTTP transport)")
	c.Flags().StringVar(&ticketingProject, "ticketing-project", os.Getenv("TICKETING_PROJECT"), "OpenProject project ID (numeric string, e.g. '6' for Demo project — look up via list_projects)")
	c.Flags().StringVar(&ticketingTypeID, "ticketing-type-id", os.Getenv("TICKETING_TYPE_ID"), "OpenProject work-package type ID (numeric string, e.g. '36' for Task — look up via list_types)")
	c.Flags().StringVar(&ticketingClosedStatusID, "ticketing-closed-status-id", os.Getenv("TICKETING_CLOSED_STATUS_ID"), "OpenProject status ID for resolved tickets (e.g. '82' for Closed — look up via list_statuses)")
	c.Flags().StringVar(&ticketingPriorityCrit, "ticketing-priority-critical", os.Getenv("TICKETING_PRIORITY_CRITICAL"), "OpenProject priority ID for CHA severity=critical (e.g. '75' for Immediate)")
	c.Flags().StringVar(&ticketingPriorityWarn, "ticketing-priority-warning", os.Getenv("TICKETING_PRIORITY_WARNING"), "OpenProject priority ID for CHA severity=warning (e.g. '74' for High)")
	c.Flags().StringVar(&ticketingPriorityInfo, "ticketing-priority-info", os.Getenv("TICKETING_PRIORITY_INFO"), "OpenProject priority ID for CHA severity=info (e.g. '73' for Normal)")
	c.Flags().StringVar(&ticketingWebURLPrefix, "ticketing-web-url-prefix", os.Getenv("TICKETING_WEB_URL_PREFIX"), "OpenProject web base URL (e.g. https://op.example.com) — used to build operator-clickable TicketRef.URL")
	c.Flags().StringSliceVar(&ticketingLabels, "ticketing-labels", []string{"cha", "auto-filed"}, "Labels appended to ticket descriptions for filtering")
	c.Flags().BoolVar(&ticketingDryRun, "ticketing-dry-run", false, "Log intended ticketing operations without calling the MCP server")
	return c
}

type ticketingOpts struct {
	Provider, MCPURL, ProjectID, TypeID, ClosedStatusID string
	PriorityCrit, PriorityWarn, PriorityInfo            string
	WebURLPrefix                                        string
	Cluster                                             string
	Labels                                              []string
	DryRun                                              bool
}

// buildTicketingConfig assembles a report.TicketingConfig from CLI flags.
// Returns an empty config (Sink == nil) when --ticketing-provider is empty —
// the watcher then no-ops the ticketing path. Currently supports
// "openproject"; Jira and ServiceNow plug in here in CHA-com.
func buildTicketingConfig(o ticketingOpts) (report.TicketingConfig, error) {
	if o.Provider == "" {
		return report.TicketingConfig{}, nil
	}
	switch o.Provider {
	case "openproject":
		client := &openproject.HTTPClient{
			Endpoint: o.MCPURL,
			APIKey:   os.Getenv("TICKETING_MCP_API_KEY"),
		}
		sevMap := map[string]string{}
		if o.PriorityCrit != "" {
			sevMap["critical"] = o.PriorityCrit
		}
		if o.PriorityWarn != "" {
			sevMap["warning"] = o.PriorityWarn
		}
		if o.PriorityInfo != "" {
			sevMap["info"] = o.PriorityInfo
		}
		sink := openproject.New(openproject.Config{
			ProjectID:        o.ProjectID,
			TypeID:           o.TypeID,
			ClosedStatusID:   o.ClosedStatusID,
			SeverityPriority: sevMap,
			Labels:           o.Labels,
			WebURLPrefix:     o.WebURLPrefix,
			DryRun:           o.DryRun,
		}, client)
		return report.TicketingConfig{
			Sink:    sink,
			Cluster: o.Cluster,
			Labels:  o.Labels,
		}, nil
	default:
		return report.TicketingConfig{}, fmt.Errorf("unsupported ticketing provider %q (OSS supports: openproject)", o.Provider)
	}
}

func diagnoseCmd() *cobra.Command {
	var (
		snapshotPath      string
		live              bool
		kubeconfig        string
		outputFormat      string
		slackWebhook      string
		slackHealthinfo   string
		writeDriftReports bool
		vaultAddr         string
		vaultMount        string
		vaultRole         string
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

			reg := catalog.Default()
			// VaultPathMissing is opt-in: requires a Vault client; live mode only.
			if vaultAddr != "" && live {
				vc, err := buildVaultClient(vaultAddr, vaultMount, vaultRole)
				if err != nil {
					fmt.Fprintln(os.Stderr, "warning: vault client unavailable:", err)
				} else {
					reg.RegisterAnalyzer(diagnose.VaultPathMissing{Client: vc})
				}
			}

			results := make([]probe.Result, 0, len(reg.Probes()))
			for _, p := range reg.Probes() {
				results = append(results, p.Run(ctx, src))
			}

			var diagnostics []diagnose.Diagnostic
			for _, a := range reg.Analyzers() {
				diagnostics = append(diagnostics, a.Run(ctx, src)...)
			}

			// DriftReport reconcile — only when running live + the CRD-write
			// is opted in. Snapshot mode skips silently since fixers' Mutator
			// requirement is the same gate.
			if writeDriftReports && live {
				if mut := snapshot.AsMutator(src); mut != nil {
					entries := report.AssembleEntries(results, diagnostics, nil)
					runID := time.Now().UTC().Format("20060102-150405")
					c, u, d, err := report.Reconcile(cmd.Context(), src, mut, entries, runID)
					if err != nil {
						fmt.Fprintln(os.Stderr, "warning: driftreport reconcile partial failure:", err)
					}
					fmt.Fprintf(os.Stderr, "driftreports: %d created, %d updated, %d deleted\n", c, u, d)
				}
			}

			// Daily digest — reads DriftReports for the 24h history window,
			// formats for #healthinfo, and optionally posts to Slack.
			if outputFormat == "daily" {
				var drList *report.DriftReportList
				if live {
					if list, err := src.List(ctx, snapshot.GVRDriftReport, ""); err == nil && list != nil {
						drList = &report.DriftReportList{Items: list.Items}
					}
				}
				payload := report.FormatDailyDigest(results, diagnostics, drList)
				if slackHealthinfo != "" {
					if err := report.PostSlack(nil, slackHealthinfo, payload); err != nil {
						fmt.Fprintln(os.Stderr, "warning: slack healthinfo post failed:", err)
					}
				}
				// Render text to stdout for log visibility.
				return printText(results, diagnostics, src.Mode())
			}

			// Standard Slack post (legacy --slack-webhook, used for non-daily cronjobs).
			if slackWebhook != "" {
				payload := report.FormatSlack(results, diagnostics, nil, false)
				if err := report.PostSlack(nil, slackWebhook, payload); err != nil {
					fmt.Fprintln(os.Stderr, "warning: slack post failed:", err)
				}
			}

			switch outputFormat {
			case "json":
				return printJSON(results, diagnostics)
			case "text", "":
				return printText(results, diagnostics, src.Mode())
			default:
				return fmt.Errorf("unknown --format %q (want json, text, or daily)", outputFormat)
			}
		},
	}
	c.Flags().StringVar(&snapshotPath, "snapshot", "", "Path to captured kubectl JSON export (file or directory)")
	c.Flags().BoolVar(&live, "live", false, "Run against the live cluster")
	c.Flags().StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig (default: in-cluster, then $KUBECONFIG, then ~/.kube/config)")
	c.Flags().StringVar(&outputFormat, "format", "text", "Output format: text|json")
	c.Flags().StringVar(&slackWebhook, "slack-webhook", "", "Slack webhook — posts full FormatSlack summary (legacy; use --slack-healthinfo + --format=daily for the daily digest)")
	c.Flags().StringVar(&slackHealthinfo, "slack-healthinfo", "", "Slack webhook for #healthinfo — posts the daily digest (requires --format=daily)")
	c.Flags().BoolVar(&writeDriftReports, "write-driftreports", true, "Upsert DriftReport CRs into the cluster (live mode only; ignored on --snapshot)")
	c.Flags().StringVar(&vaultAddr, "vault-addr", os.Getenv("VAULT_ADDR"), "Vault HTTP endpoint (default: $VAULT_ADDR). Empty disables the VaultPathMissing analyzer.")
	c.Flags().StringVar(&vaultMount, "vault-kv-mount", envOrDefault("VAULT_KV_MOUNT", "secret"), "Vault KV-v2 mount path (default: $VAULT_KV_MOUNT or 'secret')")
	c.Flags().StringVar(&vaultRole, "vault-k8s-role", os.Getenv("VAULT_K8S_ROLE"), "Vault kubernetes-auth role (default: $VAULT_K8S_ROLE). When unset, falls back to $VAULT_TOKEN.")
	return c
}

// buildVaultClient constructs a Vault client honoring the auth precedence:
//  1. $VAULT_TOKEN (development / kubeconfig-style)
//  2. kubernetes auth using the in-cluster SA JWT + the configured role
//
// Returns an error rather than nil so the caller can surface a clear
// reason in the run log; the analyzer itself treats client==nil as
// "Vault probe disabled".
func buildVaultClient(addr, mount, role string) (vault.Client, error) {
	cfg := vault.Config{Address: addr, Mount: mount}
	if tok := os.Getenv("VAULT_TOKEN"); tok != "" {
		cfg.Token = tok
		return vault.New(cfg)
	}
	if role == "" {
		return nil, fmt.Errorf("either VAULT_TOKEN or --vault-k8s-role must be set")
	}
	cfg.KubernetesAuth = &vault.KubernetesAuthConfig{Role: role}
	return vault.New(cfg)
}

func envOrDefault(envVar, def string) string {
	if v := os.Getenv(envVar); v != "" {
		return v
	}
	return def
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
