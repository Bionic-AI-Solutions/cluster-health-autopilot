// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package investigator

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/ai"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/diagnose"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/probe"
)

// fakeEnv implements ai.Environment with hard-coded responses for each tool.
// Tests construct one per scenario and assert the rule engine's output.
type fakeEnv struct {
	dns      ai.DNSResult
	dnsErr   error
	httpResp ai.HTTPProbeResult
	httpAlt  ai.HTTPProbeResult // returned on second HTTP call
	httpN    int
	tlsResp  ai.TLSResult
	desc     ai.DescribeResult
	events   []ai.EventInfo
	logsPrev ai.LogsResult // returned when LogsOptions.Previous
	logsCur  ai.LogsResult // returned otherwise
}

func (f *fakeEnv) DNSLookup(ctx context.Context, host string) (ai.DNSResult, error) {
	if f.dnsErr != nil {
		return ai.DNSResult{}, f.dnsErr
	}
	return f.dns, nil
}
func (f *fakeEnv) HTTPProbe(ctx context.Context, url string, opts ai.HTTPProbeOpts) (ai.HTTPProbeResult, error) {
	f.httpN++
	if f.httpN == 1 {
		return f.httpResp, nil
	}
	return f.httpAlt, nil
}
func (f *fakeEnv) TLSInspect(ctx context.Context, host string, port int) (ai.TLSResult, error) {
	return f.tlsResp, nil
}
func (f *fakeEnv) Describe(ctx context.Context, kind, ns, name string) (ai.DescribeResult, error) {
	return f.desc, nil
}
func (f *fakeEnv) GetEvents(ctx context.Context, ns, kind, name string, since time.Duration) ([]ai.EventInfo, error) {
	return f.events, nil
}
func (f *fakeEnv) Logs(ctx context.Context, ns, pod string, opts ai.LogsOptions) (ai.LogsResult, error) {
	if opts.Previous {
		return f.logsPrev, nil
	}
	return f.logsCur, nil
}

func TestRuleBased_TLSExpiry(t *testing.T) {
	env := &fakeEnv{
		tlsResp: ai.TLSResult{
			Host:        "pg.example.com",
			Port:        443,
			Subject:     "CN=pg.example.com",
			Issuer:      "Let's Encrypt R12",
			SANs:        []string{"pg.example.com"},
			NotAfter:    time.Date(2026, 1, 7, 0, 0, 0, 0, time.UTC),
			Expired:     true,
			HostMatches: true,
		},
		dns: ai.DNSResult{Host: "pg.example.com", Addresses: []string{"1.2.3.4"}, Elapsed: 80 * time.Millisecond},
	}
	finding := probe.Finding{
		Component: "Endpoint: test",
		Severity:  probe.SeverityCritical,
		Message:   "https://pg.example.com: TLS verification failed — tls: failed to verify certificate: x509: certificate has expired",
	}
	r, err := RuleBased{}.InvestigateFinding(context.Background(), finding, env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Conclusion != ai.ConclusionRootCauseIdentified {
		t.Errorf("expected root_cause_identified; got %s", r.Conclusion)
	}
	if !strings.Contains(r.Summary, "expired") {
		t.Errorf("summary should mention expiry; got %q", r.Summary)
	}
	if len(r.Observations) != 2 {
		t.Errorf("expected 2 observations (tls + dns); got %d", len(r.Observations))
	}
}

func TestRuleBased_TLSSANMismatch(t *testing.T) {
	env := &fakeEnv{
		tlsResp: ai.TLSResult{
			Host:        "dashboard.example.com",
			Subject:     "CN=Kong, OU=IT, CN=localhost",
			SANs:        []string{"localhost"},
			NotAfter:    time.Now().Add(60 * 24 * time.Hour),
			Expired:     false,
			HostMatches: false,
		},
		dns: ai.DNSResult{Host: "dashboard.example.com", Addresses: []string{"1.2.3.4"}, Elapsed: 50 * time.Millisecond},
	}
	finding := probe.Finding{
		Severity: probe.SeverityCritical,
		Message:  "https://dashboard.example.com: TLS verification failed — x509: certificate is not valid for any names",
	}
	r, _ := RuleBased{}.InvestigateFinding(context.Background(), finding, env)
	if r.Conclusion != ai.ConclusionRootCauseIdentified {
		t.Errorf("expected root_cause_identified; got %s", r.Conclusion)
	}
	if !strings.Contains(r.Summary, "SAN") && !strings.Contains(r.Summary, "fallback") {
		t.Errorf("summary should mention SAN mismatch / fallback; got %q", r.Summary)
	}
}

func TestRuleBased_TransientRecovery(t *testing.T) {
	// Follow-up probe succeeds → likely transient.
	env := &fakeEnv{
		dns: ai.DNSResult{Host: "x.example.com", Addresses: []string{"1.2.3.4"}, Elapsed: 60 * time.Millisecond},
		httpResp: ai.HTTPProbeResult{
			URL: "https://x.example.com", StatusCode: 200,
			TLSVerified: true, ResponseTime: 200 * time.Millisecond,
		},
	}
	finding := probe.Finding{
		Severity: probe.SeverityCritical,
		Message:  "https://x.example.com: connection failed — context deadline exceeded",
	}
	r, _ := RuleBased{}.InvestigateFinding(context.Background(), finding, env)
	if r.Conclusion != ai.ConclusionLikelyTransient {
		t.Errorf("expected likely_transient; got %s (summary: %s)", r.Conclusion, r.Summary)
	}
}

func TestRuleBased_SlowDNSRootCause(t *testing.T) {
	env := &fakeEnv{
		dns: ai.DNSResult{
			Host:      "slow.example.com",
			Addresses: []string{"1.2.3.4"},
			Elapsed:   2500 * time.Millisecond, // >1.5s threshold
		},
	}
	finding := probe.Finding{
		Severity: probe.SeverityCritical,
		Message:  "https://slow.example.com: connection failed — context deadline exceeded",
	}
	r, _ := RuleBased{}.InvestigateFinding(context.Background(), finding, env)
	if r.Conclusion != ai.ConclusionRootCauseIdentified {
		t.Errorf("expected root_cause_identified; got %s", r.Conclusion)
	}
	if !strings.Contains(r.Summary, "DNS") {
		t.Errorf("summary should mention slow DNS; got %q", r.Summary)
	}
}

func TestRuleBased_DiagnosticExternalSecret(t *testing.T) {
	env := &fakeEnv{
		desc: ai.DescribeResult{
			Kind: "ExternalSecret", Namespace: "ns", Name: "secret-x",
			Status: "Ready=False", Reason: "SecretSyncedError",
		},
	}
	d := diagnose.Diagnostic{
		Subject: "ExternalSecret/ns/secret-x",
		Message: "ExternalSecret not Ready",
	}
	r, _ := RuleBased{}.InvestigateDiagnostic(context.Background(), d, env)
	if r.Conclusion != ai.ConclusionRootCauseIdentified {
		t.Errorf("expected root_cause_identified; got %s", r.Conclusion)
	}
	if !strings.Contains(r.Summary, "Ready=False") {
		t.Errorf("summary should surface status; got %q", r.Summary)
	}
}

func TestRuleBased_UnmatchedFindingReturnsEmpty(t *testing.T) {
	env := &fakeEnv{}
	finding := probe.Finding{
		Severity: probe.SeverityCritical,
		Message:  "some completely unstructured message that no rule matches",
	}
	r, _ := RuleBased{}.InvestigateFinding(context.Background(), finding, env)
	if r.Summary != "" {
		t.Errorf("expected empty summary for unmatched finding; got %q", r.Summary)
	}
	if len(r.Observations) != 0 {
		t.Errorf("expected zero observations; got %d", len(r.Observations))
	}
}

func TestRuleBased_CrashLoop_NoSubcommand(t *testing.T) {
	// The exact shape of CHA's own mis-deployed runner: prints CLI usage and
	// exits. The investigator must identify the missing-subcommand cause.
	env := &fakeEnv{logsPrev: ai.LogsResult{
		Previous: true,
		Lines: []string{
			"Cluster Health Autopilot",
			"Usage:",
			"  cha [command]",
			"Available Commands:",
			"  diagnose    Run probes + analyzers",
			"  watch       Event-driven cluster health watcher",
		},
	}}
	f := probe.Finding{
		Component: "CrashLoopBackOff",
		Severity:  probe.SeverityCritical,
		Message:   "Pod cluster-health-autopilot/cha-runner-xyz in CrashLoopBackOff (64 restarts)",
	}
	res, err := RuleBased{}.InvestigateFinding(context.Background(), f, env)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if res.Conclusion != ai.ConclusionRootCauseIdentified {
		t.Errorf("conclusion: %q", res.Conclusion)
	}
	if !strings.Contains(res.Summary, "command/args") {
		t.Errorf("summary should name the missing-subcommand cause; got: %q", res.Summary)
	}
}

func TestRuleBased_CrashLoop_OOM(t *testing.T) {
	env := &fakeEnv{logsPrev: ai.LogsResult{Previous: true, Lines: []string{
		"server starting",
		"fatal error: runtime: out of memory",
	}}}
	f := probe.Finding{
		Component: "CrashLoopBackOff",
		Severity:  probe.SeverityCritical,
		Message:   "Pod prod/api-7f in CrashLoopBackOff (12 restarts)",
	}
	res, _ := RuleBased{}.InvestigateFinding(context.Background(), f, env)
	if !strings.Contains(strings.ToLower(res.Summary), "memory") {
		t.Errorf("OOM summary expected; got: %q", res.Summary)
	}
}

func TestRuleBased_CrashLoop_LogsUnavailable_FallsBackToEvents(t *testing.T) {
	env := &fakeEnv{
		logsPrev: ai.LogsResult{Error: "previous terminated container not found"},
		logsCur:  ai.LogsResult{Error: "container is waiting to start"},
		events:   []ai.EventInfo{{Reason: "BackOff", Message: "Back-off restarting failed container"}},
	}
	f := probe.Finding{
		Component: "CrashLoopBackOff",
		Severity:  probe.SeverityCritical,
		Message:   "Pod prod/api-7f in CrashLoopBackOff (3 restarts)",
	}
	res, _ := RuleBased{}.InvestigateFinding(context.Background(), f, env)
	if !strings.Contains(res.Summary, "BackOff") {
		t.Errorf("expected event fallback; got: %q", res.Summary)
	}
}
