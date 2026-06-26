// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package probe

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/srenix-ai/agentic-sre/internal/snapshot"
	"github.com/srenix-ai/agentic-sre/pkg/rag"
)

// EndpointTarget is a single URL to probe externally.
type EndpointTarget struct {
	// URL is the full HTTPS (or HTTP) endpoint to check.
	URL string `yaml:"url" json:"url"`
	// Name is the human-readable display name for reports.
	Name string `yaml:"name" json:"name"`
	// ExpectStatus is the required HTTP response code after following redirects.
	// Zero accepts any HTTP response (connection success + valid TLS is sufficient).
	// Non-zero requires an exact match; mismatches fire as CRITICAL findings.
	ExpectStatus int `yaml:"expectStatus,omitempty" json:"expectStatus,omitempty"`

	// L7 is the optional Layer-7 body assertion (M4). Populated by
	// IngressDiscovery from these Ingress annotations:
	//
	//   srenix.ai/probe-l7-path     → L7.Path
	//   srenix.ai/probe-l7-expect   → L7.ExpectBody (substring or "regex:<pattern>")
	//   srenix.ai/probe-l7-status   → L7.ExpectStatus (overrides top-level)
	//
	// When L7.Path != "", the probe issues a second GET against
	// scheme://host<path> after the base reachability check passes,
	// and asserts both the status code AND the body content.
	//
	// Closes the "Kong returns 200 but body is wrong" failure class —
	// the one that masked the search-mcp-server async-httpx bug for
	// hours during the 2026-05-16 investigation.
	L7 *EndpointL7Spec `yaml:"l7,omitempty" json:"l7,omitempty"`
}

// EndpointL7Spec configures the optional Layer-7 body-assertion check
// run after the base reachability probe succeeds. M4.
type EndpointL7Spec struct {
	// Path is the request path (must start with "/"). Empty disables L7.
	Path string `yaml:"path" json:"path"`

	// ExpectBody is asserted against the response body:
	//   - Plain text   → substring match (Strings.Contains)
	//   - "regex:..."  → regular-expression match
	// Empty = no body assertion (status-only).
	ExpectBody string `yaml:"expectBody,omitempty" json:"expectBody,omitempty"`

	// ExpectStatus, when non-zero, overrides EndpointTarget.ExpectStatus
	// for the L7 check. Useful when /healthz returns 200 but the home
	// page returns 302.
	ExpectStatus int `yaml:"expectStatus,omitempty" json:"expectStatus,omitempty"`
}

// Endpoints probes a list of external HTTP/HTTPS endpoints for reachability,
// TLS validity, and expected HTTP status codes.
//
// This probe is network-active. It returns SKIPPED automatically when running
// against a captured snapshot — no config change required.
//
// When Discovery.Enabled is true (the default in the OSS catalog), every
// public Ingress host in the cluster is auto-added to the probe set at Run
// time. Hosts in protected namespaces and Ingresses carrying the opt-out
// annotation are excluded. Discovered targets succeed on any HTTP response
// (TCP+TLS reachability is the contract); strict status expectations live in
// the explicit Targets slice and are checked separately.
//
// # Flake suppression (v1.4+)
//
// A single failed request does NOT immediately produce a CRITICAL finding.
// The probe retries flake-class failures (context deadline, connection reset)
// once with a longer timeout. If the retry also fails, a consecutive-failure
// counter is incremented per target; the finding is suppressed until the
// counter reaches MinConsecutiveFailures (default 2). Successful probes reset
// the counter.
//
// This eliminates the "alert → 3s later resolved" Slack noise that dominates
// transient cloud / DNS / proxy blips, while preserving fast detection
// (≈ one extra cycle of latency, ~10–20 s) for sustained outages.
//
// Findings emitted for the FIRST failure of a streak are tagged with severity
// SeverityWarning, and the watcher routes them at non-critical urgency. Only
// once the streak hits the threshold does the same subject re-emit at
// SeverityCritical.
type Endpoints struct {
	Targets   []EndpointTarget
	Discovery DiscoveryOptions
	// Timeout is the per-request deadline. Zero defaults to 10 seconds.
	Timeout time.Duration

	// MinConsecutiveFailures is the number of consecutive failed probe cycles
	// for the same target required before a Finding is escalated to
	// SeverityCritical. The first failure emits at SeverityWarning so the
	// signal isn't lost — the watcher just doesn't page on it. Default 2.
	MinConsecutiveFailures int

	// RetryOnFlake controls whether transient-class failures (context deadline,
	// connection reset, EOF) trigger one in-cycle retry with a 1.5× timeout
	// before being recorded as a failure. Default true.
	RetryOnFlake bool

	// Learnt is the paid-tier cluster RAG reader. OSS gets rag.NoopReader
	// (returns nothing) and the probe behaves exactly as it did pre-2d.
	// The paid AI tier wires a Qdrant-backed reader so apex / Cloudflare-
	// only domains learnt from prior cycles fan in alongside static
	// Targets + auto-discovered Ingress hosts.
	//
	// Nil is treated as NoopReader — bare-literal Endpoints structs in
	// existing call sites stay working without modification.
	Learnt rag.Reader

	// LearntImportanceMin is the minimum learned importance for a
	// rag.KindApexDomain entry to be replayed as a probe target. Default
	// 0.3 (include unless explicitly de-ranked). Tunable via the CR's
	// spec.ai.rag.importance.endpointMin field.
	LearntImportanceMin float64

	// streaks tracks consecutive failures per target URL. Required to be
	// non-nil for failure suppression to work — initialized by NewEndpoints
	// or lazily by Run on first use. Pointer is shared across Probe-interface
	// copies of this struct.
	streaks *streakTracker
}

// NewEndpoints returns an Endpoints probe with sensible defaults and a
// fully-initialized internal streak tracker. Prefer this over bare struct
// literals when registering the probe; raw literals work too but rely on
// lazy init inside Run.
func NewEndpoints(targets []EndpointTarget, disc DiscoveryOptions) Endpoints {
	return Endpoints{
		Targets:                targets,
		Discovery:              disc,
		MinConsecutiveFailures: 2,
		RetryOnFlake:           true,
		streaks:                newStreakTracker(),
	}
}

// streakTracker is the per-target consecutive-failure counter. Must be
// referenced via pointer from the Endpoints struct so the count survives
// the value-receiver Run method and any Probe-interface copies.
type streakTracker struct {
	mu     sync.Mutex
	counts map[string]int
}

func newStreakTracker() *streakTracker {
	return &streakTracker{counts: make(map[string]int)}
}

// record updates the streak for one target. If failed is true, the counter
// increments. If false, the counter resets to zero. Returns the new value.
func (s *streakTracker) record(target string, failed bool) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if failed {
		s.counts[target]++
	} else {
		delete(s.counts, target)
	}
	return s.counts[target]
}

// Name returns the component label for the report.
func (Endpoints) Name() string { return "External Endpoints" }

// Run executes endpoint health checks. Skips silently in snapshot mode.
func (e Endpoints) Run(ctx context.Context, src snapshot.Source) Result {
	r := Result{Component: ComponentResult{Component: "External Endpoints"}}

	if src.Mode() == snapshot.ModeSnapshot {
		r.Component.Status = "SKIPPED"
		r.Component.Detail = "network probes skipped in snapshot mode"
		return r
	}

	// Merge static targets with auto-discovered Ingress hosts and any
	// learnt apex/Cloudflare-only domains the paid AI tier has persisted.
	// Precedence: static (operator-supplied) > learnt > Ingress-discovered,
	// de-duplicated by hostname so the same host can't be probed twice.
	allTargets := append([]EndpointTarget{}, e.Targets...)
	knownHosts := hostnamesOf(allTargets)
	learnt := learntApexTargets(ctx, e.Learnt, e.LearntImportanceMin, knownHosts)
	allTargets = append(allTargets, learnt...)
	for _, lt := range learnt {
		knownHosts = append(knownHosts, hostOf(lt.URL))
	}
	discovered := DiscoverIngressTargets(ctx, src, e.Discovery, knownHosts)
	allTargets = append(allTargets, discovered...)

	if len(allTargets) == 0 {
		r.Component.Status = "SKIPPED"
		r.Component.Detail = "no targets configured and auto-discovery returned no hosts"
		return r
	}

	timeout := e.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	minStreak := e.MinConsecutiveFailures
	if minStreak < 1 {
		minStreak = 2
	}
	streaks := e.streaks
	if streaks == nil {
		// Bare-literal Endpoints fell through registration. Use a per-Run
		// tracker so the probe still works — but streak suppression won't
		// persist across cycles. The default-OSS path uses NewEndpoints.
		streaks = newStreakTracker()
	}

	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
		},
	}

	issues := 0
	healthy := 0
	suppressed := 0

	for _, t := range allTargets {
		finding, ok := probeWithRetry(ctx, client, t, timeout, e.RetryOnFlake)

		if ok {
			streaks.record(t.URL, false)
			healthy++
			continue
		}

		// Failure-class findings (invalid URL, TLS errors, status mismatches)
		// are deterministic and emit immediately regardless of streak.
		if isDeterministicFailure(finding) {
			r.Findings = append(r.Findings, finding)
			issues++
			streaks.record(t.URL, true) // still track for completeness
			continue
		}

		streak := streaks.record(t.URL, true)
		if streak < minStreak {
			// Suppress at non-critical severity so the signal exists in logs
			// / observability but doesn't page anyone.
			finding.Severity = SeverityWarning
			finding.Message = fmt.Sprintf("[transient, %d/%d] %s", streak, minStreak, finding.Message)
			r.Findings = append(r.Findings, finding)
			suppressed++
			continue
		}

		// Streak reached threshold — escalate.
		finding.Severity = SeverityCritical
		r.Findings = append(r.Findings, finding)
		issues++
	}

	switch {
	case issues == 0 && suppressed == 0:
		r.Component.Status = "HEALTHY"
		r.Component.Detail = fmt.Sprintf("All %d endpoints reachable (%d auto-discovered)", healthy, len(discovered))
	case issues == 0:
		r.Component.Status = "HEALTHY"
		r.Component.Detail = fmt.Sprintf("All %d endpoints reachable (%d auto-discovered, %d transient under threshold)", healthy, len(discovered), suppressed)
	default:
		r.Component.Status = "CRITICAL"
		r.Component.Detail = fmt.Sprintf("%d/%d endpoints failing (%d auto-discovered, %d transient suppressed)", issues, len(allTargets), len(discovered), suppressed)
	}
	return r
}

// probeWithRetry executes one probe with an optional in-cycle retry for
// transient-class failures. Returns the Finding and ok=true when the (possibly
// retried) probe succeeded.
func probeWithRetry(ctx context.Context, client *http.Client, t EndpointTarget, baseTimeout time.Duration, retryOnFlake bool) (Finding, bool) {
	reqCtx, cancel := context.WithTimeout(ctx, baseTimeout)
	finding, ok := checkEndpoint(reqCtx, client, t)
	cancel()
	if ok || !retryOnFlake || !isTransientFailure(finding) {
		return finding, ok
	}

	// Brief backoff before retry — long enough to let momentary contention clear.
	select {
	case <-ctx.Done():
		return finding, false
	case <-time.After(1500 * time.Millisecond):
	}
	retryTimeout := baseTimeout + baseTimeout/2 // 1.5×
	retryCtx, cancel2 := context.WithTimeout(ctx, retryTimeout)
	retryFinding, retryOK := checkEndpoint(retryCtx, client, t)
	cancel2()
	if retryOK {
		return Finding{}, true
	}
	return retryFinding, false
}

// checkEndpoint probes one target. Returns (finding, ok=true) when healthy.
func checkEndpoint(ctx context.Context, client *http.Client, t EndpointTarget) (Finding, bool) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.URL, nil)
	if err != nil {
		return Finding{
			Component: "Endpoint: " + t.Name,
			Severity:  SeverityCritical,
			Message:   fmt.Sprintf("invalid URL %q: %v", t.URL, err),
		}, false
	}
	req.Header.Set("User-Agent", "srenix-endpoint-probe/1.0")

	resp, err := client.Do(req)
	if err != nil {
		if isTLSError(err) {
			return Finding{
				Component:   "Endpoint: " + t.Name,
				Severity:    SeverityCritical,
				Message:     fmt.Sprintf("%s: TLS verification failed — %v", t.URL, unwrapErr(err)),
				Remediation: "Check cert-manager certificate status and DNS/Cloudflare proxy settings",
			}, false
		}
		return Finding{
			Component:   "Endpoint: " + t.Name,
			Severity:    SeverityCritical,
			Message:     fmt.Sprintf("%s: connection failed — %v", t.URL, unwrapErr(err)),
			Remediation: "Check DNS, Kong ingress route, and pod readiness for this host",
		}, false
	}
	_ = resp.Body.Close()

	if t.ExpectStatus != 0 && resp.StatusCode != t.ExpectStatus {
		return Finding{
			Component:   "Endpoint: " + t.Name,
			Severity:    SeverityCritical,
			Message:     fmt.Sprintf("%s: HTTP %d (expected %d)", t.URL, resp.StatusCode, t.ExpectStatus),
			Remediation: "Check Kong ingress rules, backend deployment readiness, and cert-manager TLS secrets",
		}, false
	}
	// M4 — Layer-7 body assertion. Only runs when the Ingress has
	// the srenix.ai/probe-l7-* annotation set. Closes
	// the "200 OK but body is wrong" class.
	if t.L7 != nil && t.L7.Path != "" {
		if f, ok := checkEndpointL7(ctx, client, t); !ok {
			return f, false
		}
	}
	return Finding{}, true
}

// checkEndpointL7 performs the optional Layer-7 body+status assertion
// described by t.L7. Issues a GET against scheme://host<path>; checks
// status (L7.ExpectStatus, falling back to t.ExpectStatus when zero)
// then optionally the body (substring or regex:<pattern> when prefixed).
//
// Soft-fail semantics match checkEndpoint — connection/TLS errors get
// the same wording and remediation hints.
func checkEndpointL7(ctx context.Context, client *http.Client, t EndpointTarget) (Finding, bool) {
	// Resolve scheme + host from the base URL; append the L7 path.
	base, err := url.Parse(t.URL)
	if err != nil || base.Host == "" {
		return Finding{
			Component: "Endpoint(L7): " + t.Name,
			Severity:  SeverityWarning,
			Message:   fmt.Sprintf("L7 check skipped — invalid base URL %q", t.URL),
		}, false
	}
	u := *base
	u.Path = t.L7.Path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return Finding{
			Component: "Endpoint(L7): " + t.Name,
			Severity:  SeverityWarning,
			Message:   fmt.Sprintf("L7 check: build request: %v", err),
		}, false
	}
	req.Header.Set("User-Agent", "srenix-endpoint-probe-l7/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return Finding{
			Component: "Endpoint(L7): " + t.Name,
			Severity:  SeverityWarning,
			Message:   fmt.Sprintf("%s: L7 connection failed — %v", u.String(), unwrapErr(err)),
		}, false
	}
	defer func() { _ = resp.Body.Close() }()
	// Status check.
	expectStatus := t.L7.ExpectStatus
	if expectStatus == 0 {
		expectStatus = t.ExpectStatus
	}
	if expectStatus != 0 && resp.StatusCode != expectStatus {
		return Finding{
			Component: "Endpoint(L7): " + t.Name,
			Severity:  SeverityCritical,
			Message:   fmt.Sprintf("%s: L7 HTTP %d (expected %d)", u.String(), resp.StatusCode, expectStatus),
		}, false
	}
	// Body assertion.
	if t.L7.ExpectBody != "" {
		body, _ := readBoundedBody(resp.Body)
		if !matchBody(body, t.L7.ExpectBody) {
			return Finding{
				Component:   "Endpoint(L7): " + t.Name,
				Severity:    SeverityCritical,
				Message:     fmt.Sprintf("%s: L7 body assertion failed — expected %q, got first 200 chars %q", u.String(), t.L7.ExpectBody, truncate(body, 200)),
				Remediation: "Backend returned 2xx but body content drifted — inspect the workload's recent deploy or feature-flag state",
			}, false
		}
	}
	return Finding{}, true
}

// isTransientFailure reports whether a finding's failure mode is the kind
// that warrants an in-cycle retry. Hardcoded markers — keep aligned with
// checkEndpoint's error message construction.
func isTransientFailure(f Finding) bool {
	m := f.Message
	if m == "" {
		return false
	}
	if strings.Contains(m, "connection failed") &&
		(strings.Contains(m, "context deadline exceeded") ||
			strings.Contains(m, "connection reset by peer") ||
			strings.Contains(m, "EOF") ||
			strings.Contains(m, "no such host") || // DNS hiccup can be transient
			strings.Contains(m, "i/o timeout")) {
		return true
	}
	return false
}

// isDeterministicFailure reports whether a finding represents a failure mode
// that is reproducible across retries — TLS validation errors, status-code
// mismatches, and invalid URLs. These bypass the streak counter entirely
// because suppressing them just delays a real alert.
func isDeterministicFailure(f Finding) bool {
	m := f.Message
	if m == "" {
		return false
	}
	if strings.Contains(m, "TLS verification failed") {
		return true
	}
	if strings.Contains(m, "expected ") && strings.Contains(m, "HTTP ") {
		return true
	}
	if strings.Contains(m, "invalid URL") {
		return true
	}
	return false
}

func isTLSError(err error) bool {
	s := err.Error()
	return strings.Contains(s, "tls:") ||
		strings.Contains(s, "x509:") ||
		strings.Contains(s, "certificate signed by unknown authority") ||
		strings.Contains(s, "self-signed certificate")
}

func unwrapErr(err error) string {
	var urlErr *url.Error
	if errors.As(err, &urlErr) && urlErr.Err != nil {
		return urlErr.Err.Error()
	}
	return err.Error()
}

// DefaultEndpointTargets returns the canonical set of public-facing endpoints
// to probe. Srenix OSS ships with an EMPTY default — bundling specific hostnames
// here would silently make every OSS install probe whichever organization's
// domains were baked in at build time. That's a cluster-leak.
//
// Sources of probe targets, in precedence order:
//
//  1. spec.probe.endpointTargets on the AgenticSRE CR — operator-
//     supplied for the current cluster.
//  2. DiscoverIngressTargets at Run time — every Ingress host in the cluster
//     is auto-probed (transparent, no config).
//  3. (Phase 2d, paid AI tier) Cluster RAG memory learns apex / Cloudflare-only
//     domains by walking the CF zone listing on the first analyzer run and
//     persisting them under `kind=apex_domain` for replay across cycles.
//
// Extend: Endpoints{Targets: append(DefaultEndpointTargets(), myExtra...)}
// Replace: Endpoints{Targets: myTargets, Discovery: probe.DiscoveryOptions{}}
func DefaultEndpointTargets() []EndpointTarget {
	return nil
}

// DefaultEndpointHostnames returns the hostnames from DefaultEndpointTargets.
// Used by IngressCoverage to determine which ingress hosts are already monitored.
func DefaultEndpointHostnames() []string {
	targets := DefaultEndpointTargets()
	hosts := make([]string, 0, len(targets))
	seen := make(map[string]struct{}, len(targets))
	for _, t := range targets {
		u, err := url.Parse(t.URL)
		if err != nil {
			continue
		}
		h := u.Hostname()
		if h == "" {
			continue
		}
		if _, ok := seen[h]; ok {
			continue
		}
		seen[h] = struct{}{}
		hosts = append(hosts, h)
	}
	return hosts
}

// readBoundedBody reads up to 64 KiB of the response body. Larger
// payloads are truncated — the L7 check is meant for /healthz-style
// endpoints, not for ingesting arbitrary downloads.
func readBoundedBody(r io.Reader) (string, error) {
	const maxBytes = 64 * 1024
	buf := make([]byte, 0, 4096)
	tmp := make([]byte, 4096)
	for len(buf) < maxBytes {
		n, err := r.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return string(buf), err
		}
	}
	return string(buf), nil
}

// matchBody returns true when body satisfies the expect spec:
//   - "regex:<pattern>"      → regular-expression match
//   - any other text         → substring match (case-sensitive)
//
// Patterns that don't compile fall back to substring match.
func matchBody(body, expect string) bool {
	if strings.HasPrefix(expect, "regex:") {
		pattern := strings.TrimPrefix(expect, "regex:")
		re, err := regexp.Compile(pattern)
		if err != nil {
			// Operator typo → fail closed (we don't want to claim
			// success when the assertion itself is malformed).
			return false
		}
		return re.MatchString(body)
	}
	return strings.Contains(body, expect)
}

// truncate caps a string for use in finding messages.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
