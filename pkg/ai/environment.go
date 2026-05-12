// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package ai

import (
	"context"
	"time"
)

// Environment is the closed set of read-only tools available to an
// Investigator. Implementations are constructed by the watcher and passed
// to each invocation. The interface is intentionally narrow — every tool
// is read-only and cluster-bounded — so a hostile or malformed LLM output
// cannot escape the safety contract.
//
// Adding a tool to this interface is a versioned design decision. New tools
// require:
//
//   - a clear safety rationale (read-only, bounded I/O),
//   - corresponding RBAC verbs for the watcher ServiceAccount,
//   - an entry in the prompt schema (for LLM-backed investigators),
//   - test coverage exercising the live implementation.
type Environment interface {
	// DNSLookup resolves the given hostname using the cluster's configured
	// resolver. Returns A/AAAA records, the resolver used, and timing.
	DNSLookup(ctx context.Context, host string) (DNSResult, error)

	// HTTPProbe issues one HTTP request to the URL, capturing status, TLS
	// validation outcome, response headers, and timing. Body is NOT read.
	HTTPProbe(ctx context.Context, url string, opts HTTPProbeOpts) (HTTPProbeResult, error)

	// TLSInspect dials host:port and inspects the served certificate chain.
	// Returns SANs, issuer, validity dates, and full chain summary.
	TLSInspect(ctx context.Context, host string, port int) (TLSResult, error)

	// Describe returns a compact human-readable description of a Kubernetes
	// resource (analogous to `kubectl describe`). Uses the watcher's
	// snapshot.Source under the hood — works in both live and snapshot mode.
	Describe(ctx context.Context, kind, namespace, name string) (DescribeResult, error)

	// GetEvents returns recent events involving a specific object, sorted
	// newest-first. Used to surface the "why" behind a failing pod / job /
	// deployment (ImagePullBackOff reasons, FailedScheduling, etc.).
	GetEvents(ctx context.Context, namespace, kind, name string, since time.Duration) ([]EventInfo, error)
}

// DNSResult is one DNS resolution outcome.
type DNSResult struct {
	Host      string        `json:"host"`
	Addresses []string      `json:"addresses,omitempty"`
	Elapsed   time.Duration `json:"elapsed_ms"`
	Error     string        `json:"error,omitempty"`
}

// HTTPProbeOpts tunes one HTTP probe call.
type HTTPProbeOpts struct {
	// Method defaults to GET when empty.
	Method string
	// Timeout defaults to 5 s when zero.
	Timeout time.Duration
	// InsecureSkipVerify disables TLS verification — useful when probing
	// a host that explicitly serves a self-signed or expired cert and the
	// investigator wants to see beyond the TLS error to the HTTP response.
	InsecureSkipVerify bool
	// FollowRedirects defaults to true.
	FollowRedirects *bool
}

// HTTPProbeResult is the captured outcome of one HTTP probe.
type HTTPProbeResult struct {
	URL          string            `json:"url"`
	Method       string            `json:"method,omitempty"`
	StatusCode   int               `json:"status_code,omitempty"`
	TLSVerified  bool              `json:"tls_verified,omitempty"`
	FinalURL     string            `json:"final_url,omitempty"`
	ResponseTime time.Duration     `json:"response_ms,omitempty"`
	DNSTime      time.Duration     `json:"dns_ms,omitempty"`
	ConnectTime  time.Duration     `json:"connect_ms,omitempty"`
	TLSTime      time.Duration     `json:"tls_ms,omitempty"`
	Headers      map[string]string `json:"headers,omitempty"`
	Error        string            `json:"error,omitempty"`
}

// TLSResult describes the certificate chain seen on a TLS handshake.
type TLSResult struct {
	Host         string        `json:"host"`
	Port         int           `json:"port"`
	Subject      string        `json:"subject,omitempty"`
	Issuer       string        `json:"issuer,omitempty"`
	SANs         []string      `json:"sans,omitempty"`
	NotBefore    time.Time     `json:"not_before,omitempty"`
	NotAfter     time.Time     `json:"not_after,omitempty"`
	Expired      bool          `json:"expired,omitempty"`
	HostMatches  bool          `json:"host_matches,omitempty"`
	ChainSummary []string      `json:"chain_summary,omitempty"`
	HandshakeErr string        `json:"handshake_err,omitempty"`
	Elapsed      time.Duration `json:"elapsed_ms,omitempty"`
}

// DescribeResult is a compact view of one Kubernetes resource.
type DescribeResult struct {
	Kind        string            `json:"kind"`
	Namespace   string            `json:"namespace,omitempty"`
	Name        string            `json:"name"`
	Status      string            `json:"status,omitempty"`
	Reason      string            `json:"reason,omitempty"`
	Message     string            `json:"message,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Notes       []string          `json:"notes,omitempty"`
	Error       string            `json:"error,omitempty"`
}

// EventInfo is a compact view of one Kubernetes Event.
type EventInfo struct {
	Type      string    `json:"type"`
	Reason    string    `json:"reason"`
	Message   string    `json:"message"`
	Count     int32     `json:"count,omitempty"`
	FirstSeen time.Time `json:"first_seen,omitempty"`
	LastSeen  time.Time `json:"last_seen,omitempty"`
	Source    string    `json:"source,omitempty"`
}
