// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

// Package watcher implements cha watch --live: a long-running event-driven
// diagnose loop.
//
// Kubernetes watches fire for every relevant resource type (pods, events,
// secrets, externalsecrets, certificates, deployments, replicasets, jobs,
// cronjobs, …). A short debounce window collapses burst events before
// re-running the full probe+analyzer stack — identical to cha diagnose --live.
//
// Slack and DriftReport outputs are fingerprint-deduplicated:
//   - A post fires only when a diagnostic is new, its severity/message changed,
//     or the configured repeat interval has expired.
//   - On pod restart the seen-map is pre-populated from existing DriftReport CRs
//     so no Slack flood occurs after a rollout.
//
// If --remedy is set, fixers run after each diagnose pass and the report
// reflects post-fix cluster state.
package watcher

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"
	kwwatch "k8s.io/apimachinery/pkg/watch"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/diagnose"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/fix"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/probe"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/report"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/server/webhook"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/trigger/prom"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/cloud"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/registry"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/silence"
)

// Config controls all tunable watcher behaviours.
type Config struct {
	// Debounce is how long to wait after the last Kubernetes event before
	// re-running diagnostics. Collapses burst updates into one cycle.
	Debounce time.Duration

	// ResyncPeriod triggers a full diagnose cycle regardless of events,
	// catching drift that didn't produce a watch event.
	ResyncPeriod time.Duration

	// SlackChannels holds the two event-driven webhook URLs.
	//   Alerts   → #ceph-alerts:   CHA acted (fixers ran and resolved issues)
	//   Critical → #ceph-critical: human action required (unfixable / still active)
	// Either may be empty — posts are silently skipped for empty strings.
	// When AlertmanagerURL is set, Alertmanager handles routing and these are
	// used only as a fallback.
	SlackChannels  report.SlackChannels
	PostOnResolved bool
	// RepeatInterval re-posts active diagnostics to Slack after this duration.
	// Zero disables repeat posts. Applies to warning + info severities; critical
	// uses CriticalRepeatInterval when that is set.
	RepeatInterval time.Duration

	// CriticalRepeatInterval re-posts active CRITICAL diagnostics to Slack
	// after this duration. Zero falls back to RepeatInterval — preserves the
	// pre-v1.6.1 single-interval behavior for operators who haven't opted in.
	// Use this to keep critical alerts loud (e.g. 4h reminder) while letting
	// warnings calm down (e.g. 24h reminder).
	CriticalRepeatInterval time.Duration

	// AlertmanagerURL is the base URL of the Alertmanager API
	// (e.g. "http://alertmanager.pg.svc.cluster.local:9093").
	// When set, CHA posts the full active-issue state each cycle as Prometheus
	// alerts. Alertmanager handles dedup, grouping, silencing, and routing to
	// all configured receivers (Slack, PagerDuty, Teams, email, …).
	// The TTL for each posted alert is set to 2× ResyncPeriod + 1 minute buffer
	// so alerts auto-resolve if CHA stops refreshing them.
	AlertmanagerURL string
	// ClusterName is stamped as the `cluster` label on every Alertmanager alert.
	// Defaults to "cluster" when empty.
	ClusterName string

	// ApprovalBaseURL is the external base URL of the approval-server
	// (e.g. https://cha-approve.example.com). When set together with a
	// registered FixProposer and Signer, the watcher emits Apply Fix
	// links pointing at <base>/approve?token=<JWT>. When empty, no
	// approval URLs are emitted regardless of registered AI components.
	ApprovalBaseURL string

	WriteDriftReports bool

	// Ticketing wires an optional issue-tracker sink (OpenProject via MCP
	// in OSS; Jira / ServiceNow in CHA-com). When Sink is nil the
	// ticketing path is a no-op and the watcher behaves exactly as
	// before. Ticketing runs after DriftReport reconcile so the resulting
	// TicketRef can be persisted onto status.ticket.
	Ticketing report.TicketingConfig

	// CloudSource is the optional multi-provider cloud-API source
	// (AWS / GCP / Azure). When nil the cloud-probe path is a complete
	// no-op and the watcher behaves exactly as before. Set via
	// `--ticketing-provider` style CLI factory in cmd/cha (look for
	// buildCloudSource).
	CloudSource cloud.Source

	// CloudCadence is the minimum interval between cloud-probe runs.
	// Cloud probes do NOT fire on every K8s event — they only run on
	// the initial cycle and then no more than once per CloudCadence.
	// Default 10 minutes when zero. Set via `--cloud-cadence` CLI flag
	// or `cloud.cadence` Helm value.
	CloudCadence time.Duration

	// RunRemediation runs fixers after each diagnose cycle and re-diagnoses
	// post-fix to report accurate state.
	RunRemediation bool
	DryRun         bool

	// NoChangeSlackDigest enables the compact "✨ No new issues since
	// last cycle" Slack render path. When true AND a cycle produces zero
	// new findings AND zero resolved transitions AND at least one stable
	// re-post would have fired, the renderer emits a single-line
	// steady-state confirmation instead of re-listing every active
	// finding. Default false — operators opt in once they're comfortable
	// the suppression doesn't hide real signal.
	NoChangeSlackDigest bool

	// PromTriggerURL is the Alertmanager base URL used by the M5 class-C
	// trigger source. When set, the watcher spawns a goroutine that polls
	// /api/v2/alerts every PromTriggerInterval and pushes a debounced
	// signal whenever a NEW firing-alert fingerprint appears. Empty
	// disables the trigger source. M5.
	PromTriggerURL string

	// PromTriggerInterval is the polling cadence for the Prometheus
	// trigger. Default 30s when zero. Clamped to ≥5s by the trigger
	// client to keep Alertmanager unloaded.
	PromTriggerInterval time.Duration

	// PromTriggerAlertFilter, when non-empty, limits which alerts fire
	// the trigger (case-insensitive alertname match). Empty = ANY firing
	// alert triggers. Useful when the cluster has high-volume alerts
	// (e.g. node_exporter scraping issues) that shouldn't churn CHA.
	PromTriggerAlertFilter []string

	// WebhookListen is the listen address for the M6 class-E trigger
	// receiver (e.g. ":8090"). Empty = receiver disabled. When set,
	// the watcher spawns an HTTP server alongside the inform-loop
	// goroutines + the Prom poller; POSTs to /webhook/<source> push
	// a debounced signal that re-runs the diagnose cycle (after HMAC
	// verification). M6.
	WebhookListen string

	// WebhookSourceSpec is the operator-supplied list of registered
	// webhook sources. Each entry is "<name>=<env-var-with-hmac-secret>".
	// An empty env-var name disables HMAC verification for that source
	// (debug-only). Cardinality of the slice = number of /webhook/<src>
	// endpoints the receiver exposes.
	WebhookSourceSpec []string
}

// watchedGVRs is the set of resource types that trigger a diagnose cycle on change.
// This mirrors the CaptureGVRs set used by `cha snapshot capture` plus Secrets.
var watchedGVRs = []schema.GroupVersionResource{
	snapshot.GVRPod,
	snapshot.GVRNode,
	snapshot.GVRPVC,
	snapshot.GVREvent,
	snapshot.GVRDeployment,
	snapshot.GVRReplicaSet,
	snapshot.GVRStatefulSet,
	snapshot.GVRDaemonSet,
	snapshot.GVRJob,
	snapshot.GVRCronJob,
	snapshot.GVRExtSecret,
	snapshot.GVRCNPGCluster,
	snapshot.GVRCephCluster,
	snapshot.GVRSecret,
	snapshot.GVRCertificate,
	// Phase 1.7 (M1) — expanded watch set. Each entry is one inform/watch
	// goroutine; cost is minimal because the watcher debounces all signals
	// through trigCh. The ArgoCD CRD is optional — the watcher proceeds
	// without it when the GVR isn't installed.
	snapshot.GVRIngress,
	snapshot.GVRHPA,
	snapshot.GVRArgoCDApplication,
	// v1.24.0 (M1 follow-up) — KEDA ScaledObject. Closes the
	// paused-annotation production hazard (memory: keda-paused-
	// scaledobject). Optional CRD: watcher.watchGVR handles
	// IsNotFound silently when KEDA isn't installed.
	snapshot.GVRScaledObject,
}

// seenEntry tracks the last-known fingerprint and Slack-post timestamp for one subject.
type seenEntry struct {
	fp          string
	lastPosted  time.Time
	subject     string
	severity    string
	message     string
	remediation string

	// Layer-2 investigator summary (OSS rule-based or paid LLM).
	investigation string

	// AI tier fields — optional, populated only when the registry has
	// an Enricher / FixProposer / Approver registered. OSS users never
	// see these set.
	enrichment       string
	proposedActionID string
	approvalURL      string

	// isNewThisCycle is set by diff() when the entry is being posted
	// because it's a new subject or its fingerprint changed since the
	// last cycle (as opposed to a repeat-interval re-post of a stable
	// finding). The routing layer surfaces these in a dedicated "🆕 New
	// this cycle" section so operators can tell at-a-glance what
	// changed since their last look at #ceph-critical.
	isNewThisCycle bool
}

// Watcher runs an event-driven diagnose loop against a live cluster.
type Watcher struct {
	lv  *snapshot.Live
	reg *registry.Registry
	mut snapshot.Mutator // nil when remediation is disabled
	cfg Config

	// silences fetches the active Silence CRD set once per cycle to
	// suppress matched diagnostics before downstream emission. nil =
	// no filtering (Silence CRD not installed / chart < 1.9 / opted
	// out at startup).
	silences silence.Lister

	mu          sync.Mutex
	seen        map[string]*seenEntry
	pendingURLs map[string]pendingURL // ai-tier approval URLs by ActionID

	// lastCloudRun rate-limits cloud-probe iteration. Cloud probes are
	// expensive (real API calls per provider) and uncorrelated with K8s
	// events — they run on the initial cycle and then no more often
	// than cfg.CloudCadence. Zero value means "never run yet → run now".
	lastCloudRun time.Time
}

// New returns a configured Watcher. mut may be nil when remediation is disabled.
func New(lv *snapshot.Live, reg *registry.Registry, mut snapshot.Mutator, cfg Config) *Watcher {
	return &Watcher{
		lv:   lv,
		reg:  reg,
		mut:  mut,
		cfg:  cfg,
		seen: make(map[string]*seenEntry),
	}
}

// WithSilenceLister sets the per-cycle Silence source. Wired by the
// main binary AFTER New() so existing callers (and tests) stay
// untouched. Pass nil to disable filtering.
func (w *Watcher) WithSilenceLister(l silence.Lister) *Watcher {
	w.silences = l
	return w
}

// Run starts the watch loop and blocks until ctx is cancelled.
// It returns ctx.Err() on clean shutdown.
func (w *Watcher) Run(ctx context.Context) error {
	// Raw events from per-GVR watchers.
	trigCh := make(chan struct{}, 1)
	// Debounced signals ready for the diagnose cycle.
	fireCh := make(chan struct{}, 1)

	// Debounce goroutine: collapses bursts from trigCh into a single fireCh signal.
	go func() {
		var timer *time.Timer
		for {
			select {
			case <-ctx.Done():
				return
			case <-trigCh:
				if timer != nil {
					timer.Stop()
				}
				timer = time.AfterFunc(w.cfg.Debounce, func() {
					select {
					case fireCh <- struct{}{}:
					default:
					}
				})
			}
		}
	}()

	for _, gvr := range watchedGVRs {
		go w.watchGVR(ctx, gvr, trigCh)
	}

	// M5 — Prometheus class-C trigger. Polls Alertmanager + pushes to
	// trigCh on new firing-alert fingerprints. No-op when URL is empty.
	if w.cfg.PromTriggerURL != "" {
		promClient, err := prom.New(prom.Config{
			URL:             w.cfg.PromTriggerURL,
			PollInterval:    w.cfg.PromTriggerInterval,
			AlertNameFilter: w.cfg.PromTriggerAlertFilter,
		}, trigCh)
		if err != nil {
			log.Printf("watcher: prom trigger init: %v", err)
		} else {
			go promClient.Run(ctx)
		}
	}

	// M6 — webhook class-E receiver. Spawns an HTTP server that
	// pushes to trigCh on HMAC-verified POSTs. No-op when listen
	// addr is empty. Sources are parsed from WebhookSourceSpec
	// ("<name>=<env-var-with-secret>"); the env-var lookup happens
	// here so secrets never sit in the Config struct.
	if w.cfg.WebhookListen != "" {
		h := webhook.New(trigCh)
		for _, spec := range w.cfg.WebhookSourceSpec {
			parts := strings.SplitN(spec, "=", 2)
			name := strings.TrimSpace(parts[0])
			if name == "" {
				continue
			}
			var secret string
			if len(parts) == 2 {
				secret = os.Getenv(strings.TrimSpace(parts[1]))
			}
			h.RegisterSource(name, secret)
		}
		mux := http.NewServeMux()
		mux.Handle("/webhook/", h)
		mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		})
		srv := &http.Server{
			Addr:              w.cfg.WebhookListen,
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
		}
		go func() {
			log.Printf("watcher: webhook listener on %s", w.cfg.WebhookListen)
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("watcher: webhook server: %v", err)
			}
		}()
		go func() {
			<-ctx.Done()
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = srv.Shutdown(shutdownCtx)
		}()
	}

	resync := time.NewTicker(w.cfg.ResyncPeriod)
	defer resync.Stop()

	// Pre-populate seen from existing DriftReports to avoid re-spamming Slack
	// after a pod restart or rolling update.
	w.loadSeenFromDriftReports(ctx)

	// Initial cycle — report current state immediately on startup.
	log.Println("watcher: initial diagnose cycle")
	w.runCycle(ctx)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-fireCh:
			log.Println("watcher: event-triggered cycle")
			w.runCycle(ctx)
		case <-resync.C:
			log.Println("watcher: resync cycle")
			w.runCycle(ctx)
		}
	}
}

// watchGVR maintains a reconnecting watch for a single GVR.
// Any ADDED/MODIFIED/DELETED event sends to trigCh (non-blocking).
// ResourceNotFound (CRD absent) causes the goroutine to exit silently.
func (w *Watcher) watchGVR(ctx context.Context, gvr schema.GroupVersionResource, trigCh chan<- struct{}) {
	for {
		if ctx.Err() != nil {
			return
		}
		wi, err := w.lv.Watch(ctx, gvr)
		if err != nil {
			if isNotFound(err) {
				return // CRD not installed; skip silently
			}
			log.Printf("watcher: watch %v: %v; retry in 30s", gvr, err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(30 * time.Second):
			}
			continue
		}

		// Consume events until the channel closes (server-side timeout) or error.
		func() {
			defer wi.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case ev, ok := <-wi.ResultChan():
					if !ok || ev.Type == kwwatch.Error {
						return
					}
					select {
					case trigCh <- struct{}{}:
					default:
					}
				}
			}
		}()

		// Brief pause before reconnect (avoids tight loop on repeated errors).
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Second):
		}
	}
}

// runCycle executes one full probe+analyze+(optional fix) pass and posts Slack/DriftReport deltas.
func (w *Watcher) runCycle(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}
	runID := time.Now().UTC().Format("20060102-150405")

	probeResults, diagnostics := w.runDiagnose(ctx)

	// Cloud probes run on their own cadence (not on K8s event triggers)
	// and are NOT re-run by the post-fix re-diagnose below (K8s fixers
	// don't mutate cloud state). Captured here once, re-merged after
	// re-diagnose so findings flow through the same AssembleEntries →
	// DriftReport → sink pipeline as K8s findings.
	cloudResults := w.runCloudDiagnose(ctx)
	probeResults = append(probeResults, cloudResults...)

	// Layer-2 investigation + AI enrichment are applied AFTER the post-fix
	// re-diagnose below (when remediation is on) — otherwise the post-fix
	// re-diagnose overwrites the annotated probeResults and the investigation
	// summary is lost. We still run the pre-fix pass on diagnostics for the
	// preFix Slack-diff baseline; probe findings are handled exclusively
	// after fixers run.
	diagnostics = w.investigateDiagnostics(ctx, diagnostics)
	diagnostics = w.enrichDiagnostics(ctx, diagnostics)

	// Capture pre-fix state for the Slack diff so that issues fixed in this
	// same cycle still appear in the "Active Issues" section of the alert.
	// Without this, fixers delete the pods/resources before buildCurrentState
	// runs, so the diff sees an empty toPost and the alert carries no context.
	preFix := buildCurrentState(probeResults, diagnostics)

	var fixResults []fix.Result
	if w.cfg.RunRemediation && w.mut != nil {
		for _, f := range w.reg.Fixers() {
			fr := f.Run(ctx, w.lv, w.mut)
			for _, a := range fr.Actions {
				log.Printf("watcher: fix[%s]: %s — %s", fr.Fixer, a.Object, a.Description)
			}
			fixResults = append(fixResults, fr)
		}
		// Re-diagnose post-fix so DriftReports reflect actual cluster state.
		probeResults, diagnostics = w.runDiagnose(ctx)
		// Re-merge cloud results — fixers don't mutate cloud state so we
		// preserve the cloud findings captured before fixers ran.
		probeResults = append(probeResults, cloudResults...)
	}

	// Investigation must run on the FINAL probeResults — whether that's the
	// pre-fix set (remediation off) or the post-fix set (remediation on).
	// Same applies to diagnostic-side investigation/enrichment, which is
	// re-applied to the post-fix diagnostics here.
	probeResults = w.investigateProbeResults(ctx, probeResults)
	if w.cfg.RunRemediation && w.mut != nil {
		diagnostics = w.investigateDiagnostics(ctx, diagnostics)
		diagnostics = w.enrichDiagnostics(ctx, diagnostics)
	}

	// Use post-fix state for DriftReport persistence; use pre-fix state for
	// the Slack diff so immediately-fixed issues still generate an alert.
	postFix := buildCurrentState(probeResults, diagnostics)
	w.attachApprovalURLs(postFix)
	diffState := postFix
	if hasActions(fixResults) {
		diffState = preFix
		w.attachApprovalURLs(preFix)
	}

	w.mu.Lock()
	toPost, toResolve := w.diff(diffState)
	w.updateSeen(postFix, toPost)
	w.mu.Unlock()

	// Alertmanager: post the full current state every cycle so Alertmanager
	// refreshes TTLs on active alerts and auto-resolves cleared ones.
	// This runs unconditionally when configured — Alertmanager deduplicates.
	if w.cfg.AlertmanagerURL != "" {
		clusterName := w.cfg.ClusterName
		if clusterName == "" {
			clusterName = "cluster"
		}
		ttl := 2*w.cfg.ResyncPeriod + time.Minute
		allActive := make([]report.DeltaDiag, 0, len(postFix))
		for _, e := range postFix {
			allActive = append(allActive, seenEntryToDeltaDiag(e))
		}
		report.PostActiveStateToAM(nil, w.cfg.AlertmanagerURL, allActive, fixResults, clusterName, ttl)
	}

	// Shared by RouteAndPost (Slack) and RouteTickets (issue tracker) below.
	postFixSubjects := make(map[string]bool, len(postFix))
	for subj := range postFix {
		postFixSubjects[subj] = true
	}
	toPostDiags := make([]report.DeltaDiag, 0, len(toPost))
	for _, e := range toPost {
		toPostDiags = append(toPostDiags, seenEntryToDeltaDiag(e))
	}
	toResolveDiags := make([]report.ResolvedDiag, 0, len(toResolve))
	for _, e := range toResolve {
		toResolveDiags = append(toResolveDiags, report.ResolvedDiag{
			Subject: e.subject,
			Message: e.message,
		})
	}

	needsSlack := len(toPost) > 0 || len(toResolve) > 0 ||
		(w.cfg.RunRemediation && !w.cfg.DryRun && hasActions(fixResults))

	if needsSlack && (w.cfg.SlackChannels.Alerts != "" || w.cfg.SlackChannels.Critical != "") {
		report.RouteAndPostConfig(nil, w.cfg.SlackChannels, postFixSubjects, toPostDiags, toResolveDiags, fixResults,
			report.CriticalRenderConfig{NoChangeDigest: w.cfg.NoChangeSlackDigest})
	}

	if w.cfg.WriteDriftReports {
		if mut := snapshot.AsMutator(w.lv); mut != nil {
			entries := report.AssembleEntries(probeResults, diagnostics, fixResults)
			c, u, d, err := report.Reconcile(ctx, w.lv, mut, entries, runID)
			if err != nil {
				log.Printf("watcher: driftreport reconcile: %v", err)
			}
			log.Printf("watcher: driftreports: %d created, %d updated, %d deleted", c, u, d)

			// Ticketing runs after Reconcile so newly-upserted DriftReports
			// are visible. Sink == nil makes this a no-op.
			if w.cfg.Ticketing.Sink != nil {
				report.RouteTickets(ctx, w.cfg.Ticketing, w.lv, mut, postFixSubjects, toPostDiags, runID)
			}
		}
	}
}

// runDiagnose runs all registered probes and analyzers and returns their results.
// Diagnostics matched by an active Silence CR are dropped before return — the
// downstream emitters (DriftReport, Slack, Alertmanager, ticketing) never see
// a silenced finding, so a known-benign-but-unfixable issue doesn't re-page on
// every probe cycle.
func (w *Watcher) runDiagnose(ctx context.Context) ([]probe.Result, []diagnose.Diagnostic) {
	results := make([]probe.Result, 0, len(w.reg.Probes()))
	for _, p := range w.reg.Probes() {
		results = append(results, p.Run(ctx, w.lv))
	}
	var diags []diagnose.Diagnostic
	for _, a := range w.reg.Analyzers() {
		diags = append(diags, a.Run(ctx, w.lv)...)
	}
	// Silence filter (pre-emission). Soft-fail: a list error is logged
	// once + filtering is skipped for this cycle rather than dropping
	// findings on the floor.
	if w.silences != nil {
		if sil, err := w.silences.List(ctx); err == nil {
			before := len(diags)
			diags = silence.Filter(diags, sil, time.Now())
			if dropped := before - len(diags); dropped > 0 {
				log.Printf("watcher: silenced %d/%d diagnostics this cycle", dropped, before)
			}
		} else {
			log.Printf("watcher: silence list failed (filtering skipped this cycle): %v", err)
		}
	}
	return results, diags
}

// runCloudDiagnose conditionally runs registered cloud probes. It returns
// nil (no results) when:
//   - no cloud source is configured (cfg.CloudSource == nil), OR
//   - no cloud probes are registered, OR
//   - the cadence window hasn't elapsed since the last cloud run.
//
// Otherwise it iterates all registered cloud probes against the source and
// returns their results. The returned slice is meant to be appended to the
// K8s probe results before AssembleEntries so cloud findings flow through
// the existing DriftReport / Slack / Alertmanager / ticketing path.
//
// IMPORTANT: cloud probes only fire on the initial cycle and then no more
// often than cfg.CloudCadence — they are NOT triggered by K8s events.
// This protects cloud-API rate limits and avoids "every Pod creation
// triggers an AWS RDS describe" hot-loops.
func (w *Watcher) runCloudDiagnose(ctx context.Context) []probe.Result {
	if w.cfg.CloudSource == nil {
		return nil
	}
	cloudProbes := w.reg.CloudProbes()
	if len(cloudProbes) == 0 {
		return nil
	}
	cadence := w.cfg.CloudCadence
	if cadence <= 0 {
		cadence = 10 * time.Minute
	}
	w.mu.Lock()
	last := w.lastCloudRun
	w.mu.Unlock()
	if !last.IsZero() && time.Since(last) < cadence {
		return nil
	}

	results := make([]probe.Result, 0, len(cloudProbes))
	for _, p := range cloudProbes {
		results = append(results, p.Run(ctx, w.cfg.CloudSource))
	}

	w.mu.Lock()
	w.lastCloudRun = time.Now()
	w.mu.Unlock()
	log.Printf("watcher: ran %d cloud probe(s)", len(cloudProbes))
	return results
}

// buildCurrentState assembles the subject→seenEntry map for the current cycle.
func buildCurrentState(results []probe.Result, diags []diagnose.Diagnostic) map[string]*seenEntry {
	m := make(map[string]*seenEntry)
	for _, r := range results {
		for _, f := range r.Findings {
			sev := string(f.Severity)
			subject := "Probe/" + r.Component.Component + "/" + f.Component
			m[subject] = &seenEntry{
				fp:            fingerprint(subject, sev, f.Message),
				subject:       subject,
				severity:      sev,
				message:       f.Message,
				remediation:   f.Remediation,
				investigation: f.Investigation,
			}
		}
	}
	for _, d := range diags {
		// Severity defaults to "warning" when the analyzer doesn't set it.
		// Source is also carried through so AI tier can attribute analyzer
		// context in the prompt.
		sev := d.Severity
		if sev == "" {
			sev = "warning"
		}
		m[d.Subject] = &seenEntry{
			fp:               fingerprint(d.Subject, sev, d.Message),
			subject:          d.Subject,
			severity:         sev,
			message:          d.Message,
			remediation:      d.Remediation,
			investigation:    d.Investigation,
			enrichment:       d.Enrichment,
			proposedActionID: d.ProposedActionID,
		}
	}
	return m
}

// seenEntryToDeltaDiag projects a seenEntry into the wire-format
// DeltaDiag used by the Slack / Alertmanager / ticketing emitters.
//
// Single source of truth so the per-destination call sites can't drift —
// a regression in PR #59 (ticketing M1) dropped ProposedActionID +
// ApprovalURL from the Slack-bound mapping while leaving them on the
// Alertmanager-bound one, silently breaking the "✅ Approve · ❌ Deny"
// line for every AI-tier proposal. The collapsed helper makes future
// field additions automatically reach every destination.
func seenEntryToDeltaDiag(e *seenEntry) report.DeltaDiag {
	return report.DeltaDiag{
		Subject:          e.subject,
		Severity:         e.severity,
		Message:          e.message,
		Remediation:      e.remediation,
		Investigation:    e.investigation,
		Enrichment:       e.enrichment,
		ProposedActionID: e.proposedActionID,
		ApprovalURL:      e.approvalURL,
		IsNewThisCycle:   e.isNewThisCycle,
	}
}

// attachApprovalURLs walks state and fills approvalURL from the watcher's
// pendingURLs map for any seenEntry whose ProposedActionID is set.
// Called between buildCurrentState and the Slack/AM emission so that the
// rendered DeltaDiag carries the URL.
func (w *Watcher) attachApprovalURLs(state map[string]*seenEntry) {
	for _, e := range state {
		if e.proposedActionID == "" {
			continue
		}
		if url := w.approvalURLFor(e.proposedActionID); url != "" {
			e.approvalURL = url
		}
	}
}

// diff computes which subjects need a Slack post (new/changed/repeat) and which
// have resolved since the last cycle. Must be called with w.mu held.
//
// Repeat-interval selection is severity-aware (v1.6.1+):
//
//	critical → CriticalRepeatInterval (falls back to RepeatInterval when 0)
//	other    → RepeatInterval
//
// This lets operators keep critical alerts loud (e.g. 4h reminders) while
// letting noisier warnings calm down (e.g. 24h reminders). Pre-v1.6.1
// callers who only set RepeatInterval get identical behavior.
func (w *Watcher) diff(current map[string]*seenEntry) (toPost, toResolve []*seenEntry) {
	now := time.Now()
	for subject, entry := range current {
		existing, seen := w.seen[subject]
		if !seen || existing.fp != entry.fp {
			// First sighting OR fingerprint changed — flag so the
			// routing layer can surface this under "🆕 New this cycle".
			entry.isNewThisCycle = true
			toPost = append(toPost, entry)
			continue
		}
		interval := w.repeatIntervalFor(entry.severity)
		if interval > 0 && now.Sub(existing.lastPosted) >= interval {
			// Re-post of a stable finding — NOT new this cycle.
			entry.isNewThisCycle = false
			toPost = append(toPost, entry)
		}
	}
	if w.cfg.PostOnResolved {
		for subject, entry := range w.seen {
			if _, exists := current[subject]; !exists {
				toResolve = append(toResolve, entry)
			}
		}
	}
	return toPost, toResolve
}

// repeatIntervalFor returns the Slack re-post interval that applies to a
// diagnostic of the given severity. Critical alerts use
// CriticalRepeatInterval when it's set (> 0); everything else (and the
// pre-v1.6.1 backward-compat path) uses RepeatInterval.
func (w *Watcher) repeatIntervalFor(severity string) time.Duration {
	if severity == "critical" && w.cfg.CriticalRepeatInterval > 0 {
		return w.cfg.CriticalRepeatInterval
	}
	return w.cfg.RepeatInterval
}

// updateSeen merges current into the seen map. Must be called with w.mu held.
func (w *Watcher) updateSeen(current map[string]*seenEntry, posted []*seenEntry) {
	now := time.Now()
	postedSet := make(map[string]bool, len(posted))
	for _, e := range posted {
		postedSet[e.subject] = true
	}

	// Remove subjects that no longer appear.
	for subject := range w.seen {
		if _, exists := current[subject]; !exists {
			delete(w.seen, subject)
		}
	}

	// Upsert: preserve lastPosted from existing entry when not re-posted.
	for subject, entry := range current {
		if postedSet[subject] {
			entry.lastPosted = now
			w.seen[subject] = entry
		} else if existing, ok := w.seen[subject]; ok {
			entry.lastPosted = existing.lastPosted
			w.seen[subject] = entry
		} else {
			w.seen[subject] = entry
		}
	}
}

// loadSeenFromDriftReports pre-populates the seen map from existing DriftReport CRs
// so pod restarts do not re-post every known issue to Slack.
func (w *Watcher) loadSeenFromDriftReports(ctx context.Context) {
	list, err := w.lv.List(ctx, snapshot.GVRDriftReport, "")
	if err != nil || list == nil {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, cr := range list.Items {
		obj := cr.Object
		spec, _ := obj["spec"].(map[string]interface{})
		if spec == nil {
			continue
		}
		subject, _ := spec["subject"].(string)
		severity, _ := spec["severity"].(string)
		message, _ := spec["message"].(string)
		remediation, _ := spec["remediation"].(string)
		if subject == "" {
			continue
		}
		status, _ := obj["status"].(map[string]interface{})
		lastPosted := time.Time{}
		if status != nil {
			if s, _ := status["lastObserved"].(string); s != "" {
				if t, err := time.Parse(time.RFC3339, s); err == nil {
					lastPosted = t
				}
			}
		}
		w.seen[subject] = &seenEntry{
			fp:          fingerprint(subject, severity, message),
			lastPosted:  lastPosted,
			subject:     subject,
			severity:    severity,
			message:     message,
			remediation: remediation,
		}
	}
	log.Printf("watcher: pre-populated seen map with %d DriftReports", len(w.seen))
}

func fingerprint(subject, severity, message string) string {
	h := sha256.Sum256([]byte(subject + "|" + severity + "|" + message))
	return hex.EncodeToString(h[:8])
}

func hasActions(fixResults []fix.Result) bool {
	for _, r := range fixResults {
		if len(r.Actions) > 0 {
			return true
		}
	}
	return false
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strContains(msg, "the server could not find the requested resource") ||
		strContains(msg, "no matches for kind")
}

func strContains(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
