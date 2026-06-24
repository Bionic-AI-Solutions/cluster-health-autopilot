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
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"
	kwwatch "k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/diagnose"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/fix"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/probe"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/report"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/server/webhook"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/trigger/prom"
	pkgai "github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/ai"
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

	// SilenceLinks carries the Ed25519 signer + approval base URL +
	// short/long windows used to mint the two one-click Silence links
	// (subject-scoped snooze + class-scoped mute) attached to every posted
	// finding. When not fully configured (no key / no base URL) the
	// watcher mints nothing and the Slack renderer falls back to the
	// kubectl heredoc — OSS-only / air-gapped installs keep working.
	// Defaults: ShortDur 24h, LongDur 90d (filled by attachApprovalURLs).
	SilenceLinks report.SilenceLinkConfig

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

	// KubeClientset is the typed Kubernetes client used by the Layer-2
	// investigator to stream pod logs (Environment.Logs). When nil the
	// investigator still runs but cannot read container logs — it falls back
	// to describe + events only. Set from cmd/cha via snapshot.BuildKubeClientset.
	KubeClientset kubernetes.Interface

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

	// ApprovalDwellDuration is how long a finding must continuously be
	// present before the watcher attaches Approve/Deny buttons to its
	// Slack message. During the dwell window the finding appears in Slack
	// (for visibility) but without action buttons — operators see the
	// issue without being asked to act until it's clear it won't self-resolve.
	//
	// Findings that clear before the dwell expires never generate an
	// approval request, eliminating the "already resolved" Slack flood
	// seen when many findings are transient. Self-resolved findings are
	// reported normally via PostOnResolved.
	//
	// Zero disables the dwell gate: approval URLs attach immediately on
	// the first cycle (legacy behaviour). Recommended: 5m for most
	// clusters. Set via --approval-dwell or approval.dwellDuration Helm value.
	ApprovalDwellDuration time.Duration

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
	// FAIL-CLOSED: a missing/empty env var disables the SOURCE (not the
	// verification); the only unauthenticated form is the explicit
	// literal "<name>=insecure-no-hmac". Cardinality of the slice =
	// number of /webhook/<src> endpoints the receiver exposes.
	WebhookSourceSpec []string

	// HealthListen is the listen address for the always-on health
	// server that serves GET /healthz (200 = process alive). It is
	// independent of WebhookListen so liveness/readiness probes work
	// even when the M6 webhook receiver is disabled. Empty = default
	// defaultHealthListen (":8081").
	HealthListen string
}

// defaultHealthListen is the health server's listen address when
// Config.HealthListen is empty. A dedicated port (not the webhook
// port) keeps /healthz available unconditionally.
const defaultHealthListen = ":8081"

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
	// EndpointSlice (discovery.k8s.io/v1) — the canonical, non-deprecated
	// endpoint API. An endpoint-membership change (pod becomes ready /
	// unready behind a Service) triggers a diagnose cycle directly instead
	// of relying on the Pod watch to surface it. Deliberately NOT the
	// deprecated core/v1 Endpoints GVR: watching that emits a
	// "v1 Endpoints is deprecated in v1.33+" client-go warning on every
	// (re)connect.
	snapshot.GVREndpointSlice,
}

// seenEntry tracks the last-known fingerprint and Slack-post timestamp for one subject.
type seenEntry struct {
	fp         string
	lastPosted time.Time
	// firstSeen is the wall-clock time of the first cycle this subject appeared.
	// Used by the approval-dwell gate: Approve/Deny URLs are withheld until
	// now-firstSeen >= Config.ApprovalDwellDuration, so transient findings
	// never produce approval requests.
	firstSeen time.Time
	subject   string
	severity    string
	message     string
	remediation string

	// source is the analyzer/probe name that produced this finding. It
	// becomes matcher.source on the class-scoped one-click Silence link
	// (mute the WHOLE class). Carried from diagnose.Diagnostic.Source;
	// for probe findings it defaults to the probe component.
	source string

	// silenceSubjectURL / silenceClassLongURL are the signed one-click
	// Silence links minted by attachApprovalURLs when SilenceLinks is
	// configured. Empty otherwise (renderer falls back to kubectl).
	// silenceShortDur / silenceLongDur carry the configured windows so the
	// renderer can label the links with the real duration.
	silenceSubjectURL   string
	silenceClassLongURL string
	silenceShortDur     time.Duration
	silenceLongDur      time.Duration

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

	// now is the clock seam for TTL-based eviction of pendingURLs.
	// Defaults to time.Now via New(); tests inject a fixed clock. Nil
	// is tolerated (callers that build a Watcher literal without New):
	// nowOrDefault() falls back to time.Now.
	now func() time.Time

	// lastCloudRun rate-limits cloud-probe iteration. Cloud probes are
	// expensive (real API calls per provider) and uncorrelated with K8s
	// events — they run on the initial cycle and then no more often
	// than cfg.CloudCadence. Zero value means "never run yet → run now".
	lastCloudRun time.Time

	// healthOnce + healthLn back the always-on health server. The
	// server is started at most once per Watcher (StartHealthServer is
	// idempotent: cmd/cha starts it BEFORE leader election so standby
	// pods serve /healthz, and Run calls it again defensively for
	// direct callers). healthLn holds the bound listener so tests can
	// discover the ephemeral port. Guarded by healthMu.
	healthOnce sync.Once
	healthMu   sync.Mutex
	healthLn   net.Listener
}

// New returns a configured Watcher. mut may be nil when remediation is disabled.
func New(lv *snapshot.Live, reg *registry.Registry, mut snapshot.Mutator, cfg Config) *Watcher {
	return &Watcher{
		lv:   lv,
		reg:  reg,
		mut:  mut,
		cfg:  cfg,
		seen: make(map[string]*seenEntry),
		now:  time.Now,
	}
}

// nowOrDefault returns the injected clock, falling back to time.Now for
// Watcher literals built without New() (some tests).
func (w *Watcher) nowOrDefault() time.Time {
	if w.now != nil {
		return w.now()
	}
	return time.Now()
}

// healthListenAddr resolves the effective health listen address.
func (w *Watcher) healthListenAddr() string {
	if w.cfg.HealthListen != "" {
		return w.cfg.HealthListen
	}
	return defaultHealthListen
}

// healthHandler returns the always-on health endpoint. GET /healthz
// returns 200 unconditionally — it reports process liveness, not
// readiness of any optional subsystem (webhook/prom triggers are
// best-effort and must not gate the liveness probe).
//
// /readyz is a deliberate alias of /healthz and is also 200 in standby
// (NOT gated on holding the leader lease). Rationale: both the chart
// and the operator point liveness AND readiness at this server, the
// watcher serves no Service traffic that readiness needs to gate, and
// a leadership-gated readiness would re-create the 1.26.0 upgrade
// deadlock under RollingUpdate maxUnavailable=0 — the new standby pod
// could never become Ready while the old leader holds the lease, so
// the rollout would never progress. A standby pod is exactly as
// "ready" as it is supposed to be: alive and waiting for the lease.
func (w *Watcher) healthHandler() http.Handler {
	mux := http.NewServeMux()
	ok := func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(http.StatusOK)
		_, _ = rw.Write([]byte("ok"))
	}
	mux.HandleFunc("/healthz", ok)
	mux.HandleFunc("/readyz", ok)
	return mux
}

// StartHealthServer starts the always-on /healthz (+/readyz alias)
// server and returns once the listener is BOUND — callers know the
// probe target exists the moment this returns. It is idempotent;
// only the first call binds. The server shuts down when ctx ends.
//
// REGRESSION GUARD (O11, production 1.26.0 upgrade incident): cmd/cha
// MUST call this BEFORE RunWithLeader. PR #186 originally started this
// server inside Run — which only executes after the leader lease is
// acquired — so standby pods served nothing, the liveness probe got
// `connection refused`, and rolling upgrades with maxUnavailable=0
// deadlocked (new pod kill-looped while the old leader held the lease).
func (w *Watcher) StartHealthServer(ctx context.Context) error {
	var bindErr error
	w.healthOnce.Do(func() {
		healthAddr := w.healthListenAddr()
		ln, err := net.Listen("tcp", healthAddr)
		if err != nil {
			bindErr = err
			return
		}
		w.healthMu.Lock()
		w.healthLn = ln
		w.healthMu.Unlock()
		healthSrv := &http.Server{
			Handler:           w.healthHandler(),
			ReadHeaderTimeout: 5 * time.Second,
		}
		go func() {
			log.Printf("watcher: health listener on %s", ln.Addr())
			if err := healthSrv.Serve(ln); err != nil && err != http.ErrServerClosed {
				log.Printf("watcher: health server: %v", err)
			}
		}()
		go func() {
			<-ctx.Done()
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = healthSrv.Shutdown(shutdownCtx)
		}()
	})
	return bindErr
}

// healthAddr returns the bound address of the health listener, or ""
// when StartHealthServer has not (successfully) run. Test seam for the
// ephemeral-port (":0") case.
func (w *Watcher) healthAddr() string {
	w.healthMu.Lock()
	defer w.healthMu.Unlock()
	if w.healthLn == nil {
		return ""
	}
	return w.healthLn.Addr().String()
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
		registerWebhookSources(h, w.cfg.WebhookSourceSpec, os.Getenv)
		mux := http.NewServeMux()
		mux.Handle("/webhook/", h)
		// Keep /healthz reachable on the webhook port too (legacy
		// /webhook/health probes and any tooling pinned to the webhook
		// listener). The always-on health server below is the canonical
		// probe target.
		mux.HandleFunc("/healthz", func(rw http.ResponseWriter, _ *http.Request) {
			rw.WriteHeader(http.StatusOK)
			_, _ = rw.Write([]byte("ok"))
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

	// Always-on health server. cmd/cha starts it BEFORE leader election
	// (so standby pods serve /healthz — see StartHealthServer's O11
	// regression note); this call is an idempotent no-op there, and the
	// real start for direct Run callers (leader election disabled).
	if err := w.StartHealthServer(ctx); err != nil {
		log.Printf("watcher: health listener %s: %v", w.healthListenAddr(), err)
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

// insecureNoHMACToken is the literal spec value (in place of an env-var
// name) that registers a webhook source with HMAC verification disabled.
const insecureNoHMACToken = "insecure-no-hmac"

// registerWebhookSources parses each "<name>=<env-var-with-secret>"
// trigger spec and registers it on h. FAIL-CLOSED: a malformed spec
// (no '='), or an env var that is unset/empty, logs an error and the
// source is NOT registered — an unauthenticated source must never be
// created by accident. The only way to register a source without HMAC
// is the explicit literal "<name>=insecure-no-hmac", which logs a loud
// warning. getenv is injectable for tests (os.Getenv in production).
func registerWebhookSources(h *webhook.Handler, specs []string, getenv func(string) string) {
	seen := make(map[string]bool)
	for _, spec := range specs {
		parts := strings.SplitN(spec, "=", 2)
		name := strings.TrimSpace(parts[0])
		if name == "" {
			log.Printf("watcher: ERROR webhook source spec %q has an empty name — source disabled (fail-closed)", spec)
			continue
		}
		if seen[name] {
			log.Printf("watcher: WARNING webhook source %q registered more than once; last spec wins", name)
		}
		seen[name] = true
		if len(parts) != 2 || strings.TrimSpace(parts[1]) == "" {
			log.Printf("watcher: ERROR webhook source %q: spec is missing '=<env-var-with-secret>' — source disabled (fail-closed)", name)
			continue
		}
		envVar := strings.TrimSpace(parts[1])
		if envVar == insecureNoHMACToken {
			log.Printf("watcher: WARNING UNAUTHENTICATED webhook source %s — anyone who can reach the listener can trigger diagnose cycles", name)
			h.RegisterInsecureSource(name)
			continue
		}
		secret := getenv(envVar)
		if secret == "" {
			log.Printf("watcher: ERROR webhook source %q: env var %s is unset or empty — source disabled (fail-closed); mount the secret or use '%s=%s' to explicitly opt out of HMAC", name, envVar, name, insecureNoHMACToken)
			continue
		}
		h.RegisterSource(name, secret)
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
	// clearedSubjects = subjects seen last cycle but absent now. Computed
	// independently of PostOnResolved (which only governs the Slack
	// "resolved" notice) so ticketing resolve-on-clear works regardless of
	// the Slack setting. Captured under the lock BEFORE updateSeen prunes
	// w.seen.
	clearedSubjects := make([]string, 0)
	for subject := range w.seen {
		if _, exists := diffState[subject]; !exists {
			clearedSubjects = append(clearedSubjects, subject)
		}
	}
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
			// Resolve-on-clear (M2) MUST run BEFORE Reconcile — Reconcile
			// deletes the DriftReport CRs for cleared subjects, and the
			// persisted TicketRef lives on that CR's status.ticket. After
			// Reconcile the ref is gone. clearedSubjects is computed
			// unconditionally (subjects seen last cycle, absent now) —
			// unlike the diff's toResolve, which is gated on PostOnResolved.
			if w.cfg.Ticketing.Sink != nil && len(clearedSubjects) > 0 {
				report.RouteResolves(ctx, w.cfg.Ticketing, w.lv, mut, clearedSubjects, time.Now())
			}

			// Retire tickets now below the severity floor (e.g. after the
			// floor was tightened to critical). Runs before Reconcile so the
			// TicketRef on status.ticket is still readable. No-op once drained.
			if w.cfg.Ticketing.Sink != nil {
				report.RouteTicketCleanup(ctx, w.cfg.Ticketing, w.lv, mut, time.Now())
			}

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
			now := time.Now()
			// Reconcile each Silence's status subresource (active /
			// matchCount / lastMatchAt). Best-effort: only changed
			// silences are patched, and failures never abort the cycle
			// — status is observability, not control flow. The Mutator
			// gate comes FIRST so snapshot/dry-run sources skip the
			// per-silence match counting too: counts feed ONLY the
			// status writer, so without one they are dead work.
			if mut := snapshot.AsMutator(w.lv); mut != nil {
				// Match counts on the PRE-filter diagnostics — feeds the
				// status writer (status.matchCount / ACTIVE printer column).
				counts := silence.CountMatches(diags, sil, now)
				if _, uerr := silence.UpdateStatuses(ctx, mut, sil, counts, now); uerr != nil {
					log.Printf("watcher: silence status update: %v", uerr)
				}
			}
			before := len(diags)
			diags = silence.Filter(diags, sil, now)
			if dropped := before - len(diags); dropped > 0 {
				log.Printf("watcher: silenced %d/%d diagnostics this cycle", dropped, before)
			}
			// Also apply the silence filter to probe results — without this,
			// probe findings (Nodes, Ceph, PVCs, Endpoints) bypass Silence CRs
			// entirely because silence.Filter only processes []diagnose.Diagnostic.
			probeDropped := 0
			for i := range results {
				kept := results[i].Findings[:0:0]
				for _, f := range results[i].Findings {
					d := diagnose.Diagnostic{
						Source:   results[i].Component.Component,
						Subject:  "Probe/" + results[i].Component.Component + "/" + f.Component,
						Severity: string(f.Severity),
						Message:  f.Message,
					}
					if !silence.MatchesAny(sil, d, now) {
						kept = append(kept, f)
					} else {
						probeDropped++
					}
				}
				results[i].Findings = kept
			}
			if probeDropped > 0 {
				log.Printf("watcher: silenced %d probe finding(s) this cycle", probeDropped)
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
				// probe.Finding has no distinct Source — the probe
				// component is the class identity for class-scoped silence.
				source: r.Component.Component,
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
			source:           d.Source,
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
		Subject:             e.subject,
		Source:              e.source,
		Severity:            e.severity,
		Message:             e.message,
		Remediation:         e.remediation,
		Investigation:       e.investigation,
		Enrichment:          e.enrichment,
		ProposedActionID:    e.proposedActionID,
		ApprovalURL:         e.approvalURL,
		IsNewThisCycle:      e.isNewThisCycle,
		SilenceSubjectURL:   e.silenceSubjectURL,
		SilenceClassLongURL: e.silenceClassLongURL,
		SilenceShortDur:     e.silenceShortDur,
		SilenceLongDur:      e.silenceLongDur,
	}
}

// firstSeenFor returns the firstSeen timestamp from w.seen for subject, or
// zero if the subject has no history. Called by attachApprovalURLs, which
// runs without w.mu held, so this acquires the lock itself.
func (w *Watcher) firstSeenFor(subject string) time.Time {
	w.mu.Lock()
	defer w.mu.Unlock()
	if e, ok := w.seen[subject]; ok {
		return e.firstSeen
	}
	return time.Time{}
}

// attachApprovalURLs walks state and fills approvalURL from the watcher's
// pendingURLs map for any seenEntry whose ProposedActionID is set.
// Called between buildCurrentState and the Slack/AM emission so that the
// rendered DeltaDiag carries the URL.
//
// Also stamps firstSeen on each entry (carried forward from history, or set
// to now for brand-new findings) and gates approval URL attachment behind
// Config.ApprovalDwellDuration: findings younger than the dwell window appear
// in Slack without Approve/Deny buttons so transient findings self-resolve
// without ever entering the human approval queue.
func (w *Watcher) attachApprovalURLs(state map[string]*seenEntry) {
	now := w.nowOrDefault()
	dwell := w.cfg.ApprovalDwellDuration

	for _, e := range state {
		// Propagate firstSeen: carry forward the original timestamp from
		// w.seen, or stamp now for findings appearing for the first time.
		if prev := w.firstSeenFor(e.subject); !prev.IsZero() {
			e.firstSeen = prev
		} else {
			e.firstSeen = now
		}

		if e.proposedActionID != "" {
			// Gate: only attach Approve/Deny once the finding has persisted
			// through the dwell window. Zero dwell = attach immediately (legacy).
			if dwell <= 0 || now.Sub(e.firstSeen) >= dwell {
				if url := w.approvalURLFor(e.proposedActionID); url != "" {
					e.approvalURL = url
				}
			}
		}
		w.attachSilenceLinks(e)
	}
}

// attachSilenceLinks mints the two signed one-click Silence links for one
// finding when the watcher's SilenceLinks config is fully populated (signer
// + approval base URL). Subject link → matcher.{source,subject}; class link
// → matcher.source only. Soft-fail: a signing/URL error just leaves the
// links empty and the renderer falls back to the kubectl heredoc — a
// finding never loses its render because a link couldn't be signed.
//
// Minted for EVERY posted finding (criticals + the steady-state/diagnostic
// set the renderer surfaces) so the affordance is uniform across the post.
func (w *Watcher) attachSilenceLinks(e *seenEntry) {
	cfg := &w.cfg.SilenceLinks
	if !cfg.Enabled() {
		return
	}
	short := cfg.ShortDur
	if short <= 0 {
		short = report.DefaultSilenceShortDuration
	}
	long := cfg.LongDur
	if long <= 0 {
		long = report.DefaultSilenceLongDuration
	}
	kid := cfg.KeyID
	if kid == "" {
		kid = "default-1"
	}
	// matcher.source: the analyzer/probe name. Fall back to the Subject
	// when an analyzer left Source empty so MintSilenceLinks (which
	// requires a non-empty Source) still produces usable links.
	source := e.source
	if source == "" {
		source = e.subject
	}
	links, err := pkgai.MintSilenceLinks(cfg.PrivateKey, kid, cfg.BaseURL, pkgai.SilenceLinkRequest{
		Source:   source,
		Subject:  e.subject,
		ShortDur: short,
		LongDur:  long,
	}, w.nowOrDefault())
	if err != nil {
		return
	}
	e.silenceSubjectURL = links.SubjectURL
	e.silenceClassLongURL = links.ClassURL
	e.silenceShortDur = short
	e.silenceLongDur = long
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
		// Proceed with an empty seen map (same behavior as before), but
		// say so: a transient apiserver error here means every known
		// finding re-posts to Slack on the first cycle — without this
		// log line that flood looks inexplicable.
		if err == nil {
			log.Printf("watcher: pre-populating seen map from DriftReports failed (Slack may re-post known findings): empty list response")
		} else {
			log.Printf("watcher: pre-populating seen map from DriftReports failed (Slack may re-post known findings): %v", err)
		}
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
		// Restore firstSeen from the DriftReport's creation timestamp so that
		// the dwell gate correctly treats pre-existing findings as already past
		// the dwell window after a pod restart.
		firstSeen := time.Time{}
		if metadata, _ := obj["metadata"].(map[string]interface{}); metadata != nil {
			if ct, _ := metadata["creationTimestamp"].(string); ct != "" {
				if t, err := time.Parse(time.RFC3339, ct); err == nil {
					firstSeen = t
				}
			}
		}
		if firstSeen.IsZero() {
			firstSeen = lastPosted // fall back so dwell is still considered passed
		}
		w.seen[subject] = &seenEntry{
			fp:          fingerprint(subject, severity, message),
			lastPosted:  lastPosted,
			firstSeen:   firstSeen,
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
