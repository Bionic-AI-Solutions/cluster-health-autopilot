// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package probe

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
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
type Endpoints struct {
	Targets   []EndpointTarget
	Discovery DiscoveryOptions
	// Timeout is the per-request deadline. Zero defaults to 10 seconds.
	Timeout time.Duration
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

	// Merge static targets with any auto-discovered Ingress hosts.
	allTargets := append([]EndpointTarget{}, e.Targets...)
	discovered := DiscoverIngressTargets(ctx, src, e.Discovery, hostnamesOf(e.Targets))
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

	client := &http.Client{
		Timeout: timeout,
		// TLS verification is ON by default; InsecureSkipVerify stays false.
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
		},
	}

	issues := 0
	healthy := 0

	for _, t := range allTargets {
		reqCtx, cancel := context.WithTimeout(ctx, timeout)
		finding, ok := checkEndpoint(reqCtx, client, t)
		cancel()
		if !ok {
			r.Findings = append(r.Findings, finding)
			issues++
		} else {
			healthy++
		}
	}

	if issues == 0 {
		r.Component.Status = "HEALTHY"
		r.Component.Detail = fmt.Sprintf("All %d endpoints reachable (%d auto-discovered)", healthy, len(discovered))
	} else {
		r.Component.Status = "CRITICAL"
		r.Component.Detail = fmt.Sprintf("%d/%d endpoints failing (%d auto-discovered)", issues, len(allTargets), len(discovered))
	}
	return r
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
	req.Header.Set("User-Agent", "cha-endpoint-probe/1.0")

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
	return Finding{}, true
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
// for this cluster — apex domains with strict status-code contracts, plus any
// host whose probe identity benefits from an explicit display name.
//
// Ingress-exposed hosts not listed here are picked up automatically by
// DiscoverIngressTargets at Run time. Add a host to this set when you need a
// strict ExpectStatus contract or an externally-hosted endpoint that has no
// matching Ingress in the cluster (e.g. apex domains served via Cloudflare).
//
// Extend: Endpoints{Targets: append(DefaultEndpointTargets(), myExtra...)}
// Replace: Endpoints{Targets: myTargets, Discovery: probe.DiscoveryOptions{}}
func DefaultEndpointTargets() []EndpointTarget {
	return []EndpointTarget{
		{URL: "https://bionicaisolutions.com", Name: "Bionic AI Solutions (apex)", ExpectStatus: 200},
		{URL: "https://www.bionicaisolutions.com", Name: "Bionic AI Solutions (www)", ExpectStatus: 200},
		{URL: "https://baisoln.com", Name: "baisoln.com (apex)", ExpectStatus: 200},
		{URL: "https://www.baisoln.com", Name: "baisoln.com (www)", ExpectStatus: 200},
		{URL: "https://auth.bionicaisolutions.com", Name: "Keycloak Auth"},
		{URL: "https://langfuse.bionicaisolutions.com", Name: "Langfuse Observability"},
		{URL: "https://platform.baisoln.com", Name: "Bionic Platform"},
		{URL: "https://mail.bionicaisolutions.com", Name: "Mail Service"},
	}
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
