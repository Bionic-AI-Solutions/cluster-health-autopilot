// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package investigator

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/srenix-ai/agentic-sre/pkg/ai"
	"github.com/srenix-ai/agentic-sre/pkg/diagnose"
	"github.com/srenix-ai/agentic-sre/pkg/probe"
)

// fakeEnv implements ai.Environment with hard-coded responses for each tool.
// Tests construct one per scenario and assert the rule engine's output.
type fakeEnv struct {
	dns       ai.DNSResult
	dnsErr    error
	httpResp  ai.HTTPProbeResult
	httpAlt   ai.HTTPProbeResult // returned on second HTTP call
	httpN     int
	tlsResp   ai.TLSResult
	desc      ai.DescribeResult
	events    []ai.EventInfo
	logsPrev  ai.LogsResult // returned when LogsOptions.Previous
	logsCur   ai.LogsResult // returned otherwise
	latestPod string        // returned by LatestByPrefix(kind=Pod)
	latestJob string        // returned by LatestByPrefix(kind=Job)
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
func (f *fakeEnv) LatestByPrefix(ctx context.Context, kind, ns, prefix string) (string, error) {
	if strings.EqualFold(kind, "Job") {
		return f.latestJob, nil
	}
	return f.latestPod, nil
}

// A stuck CronJob whose latest Job is still ACTIVE (no terminal condition)
// because its pod can't start — CreateContainerConfigError on a missing Secret
// key. The cause lives in the pod's container waiting reason, NOT the Job's
// conditions; the investigator must surface it instead of the "can't be read"
// fallback. Reproduces livekit-agents/retention-sweep (2026-06-25).
func TestInvestigateCronJob_PodCantStart_MissingSecretKey(t *testing.T) {
	env := &fakeEnv{
		latestPod: "retention-sweep-29705880-smz9z",
		// Pod never started → no logs.
		logsCur:  ai.LogsResult{},
		logsPrev: ai.LogsResult{},
		desc: ai.DescribeResult{
			Notes: []string{
				"container web waiting: CreateContainerConfigError — couldn't find key postgres_database_url in Secret livekit-agents/agency-api-secrets",
			},
		},
	}
	d := diagnose.Diagnostic{
		Source:  "CronJobStuck",
		Subject: "CronJob/livekit-agents/retention-sweep",
		Message: "CronJob livekit-agents/retention-sweep has not had a successful run in 172h0m0s (schedule 0 2 * * *).",
	}
	r, err := RuleBased{}.InvestigateDiagnostic(context.Background(), d, env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Conclusion != ai.ConclusionRootCauseIdentified {
		t.Fatalf("expected root_cause_identified; got %s (summary=%q)", r.Conclusion, r.Summary)
	}
	if !strings.Contains(r.Summary, "couldn't find key postgres_database_url") {
		t.Errorf("summary must name the missing secret key, not the 'can't be read' fallback; got %q", r.Summary)
	}
	if strings.Contains(r.Summary, "can't be read") || strings.Contains(r.Summary, "garbage-collected") {
		t.Errorf("must NOT fall through to the can't-be-read fallback; got %q", r.Summary)
	}
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
	// The exact shape of Srenix's own mis-deployed runner: prints CLI usage and
	// exits. The investigator must identify the missing-subcommand cause.
	env := &fakeEnv{logsPrev: ai.LogsResult{
		Previous: true,
		Lines: []string{
			"Agentic SRE",
			"Usage:",
			"  srenix [command]",
			"Available Commands:",
			"  diagnose    Run probes + analyzers",
			"  watch       Event-driven cluster health watcher",
		},
	}}
	f := probe.Finding{
		Component: "CrashLoopBackOff",
		Severity:  probe.SeverityCritical,
		Message:   "Pod agentic-sre/srenix-runner-xyz in CrashLoopBackOff (64 restarts)",
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

func TestRuleBased_ImagePull_SurfacesPullErrorEvent(t *testing.T) {
	// No logs (image never pulled) → surface the informative Failed event
	// (manifest unknown), not the generic BackOff.
	env := &fakeEnv{
		logsPrev: ai.LogsResult{Error: "container is waiting to start: ImagePullBackOff"},
		logsCur:  ai.LogsResult{Error: "container is waiting to start: ImagePullBackOff"},
		events: []ai.EventInfo{
			{Reason: "BackOff", Message: "Back-off pulling image"},
			{Reason: "Failed", Message: "Failed to pull image \"x:latest\": manifest unknown"},
		},
	}
	f := probe.Finding{
		Component: "CrashLoopBackOff", Severity: probe.SeverityCritical,
		Message: "Pod prod/api-7f matched pattern ImagePullBackOff",
	}
	res, _ := RuleBased{}.InvestigateFinding(context.Background(), f, env)
	if !strings.Contains(res.Summary, "manifest unknown") {
		t.Errorf("should surface the pull-error event; got: %q", res.Summary)
	}
}

func TestRuleBased_RejectedPod_UsesStatusCause(t *testing.T) {
	// No logs (pod rejected) → surface the status rejection reason from describe.
	env := &fakeEnv{
		logsPrev: ai.LogsResult{Error: "terminated"},
		logsCur:  ai.LogsResult{Error: "terminated"},
		desc:     ai.DescribeResult{Reason: "UnexpectedAdmissionError", Message: "Pod was rejected: Allocate failed — nvidia.com/gpu unavailable"},
	}
	f := probe.Finding{
		Component: "FailedPods", Severity: probe.SeverityCritical,
		Message: "Pod prod/gpu-job is in terminal Failed phase (reason=UnexpectedAdmissionError)",
	}
	res, _ := RuleBased{}.InvestigateFinding(context.Background(), f, env)
	if !strings.Contains(res.Summary, "nvidia.com/gpu") {
		t.Errorf("should surface the GPU rejection cause; got: %q", res.Summary)
	}
}

func TestRuleBased_GenericBackOff_StaysSilent(t *testing.T) {
	// No logs, no status cause, only a generic BackOff event that adds nothing →
	// stay SILENT rather than emit a misleading "couldn't determine" line.
	env := &fakeEnv{
		logsPrev: ai.LogsResult{Error: "terminated"},
		logsCur:  ai.LogsResult{Error: "terminated"},
		events:   []ai.EventInfo{{Reason: "BackOff", Message: "Back-off restarting failed container"}},
	}
	f := probe.Finding{
		Component: "CrashLoopBackOff", Severity: probe.SeverityCritical,
		Message: "Pod prod/api-7f in CrashLoopBackOff (3 restarts)",
	}
	res, _ := RuleBased{}.InvestigateFinding(context.Background(), f, env)
	if res.Summary != "" {
		t.Errorf("generic BackOff should not produce a root-cause line; got: %q", res.Summary)
	}
}

func TestRuleBased_CronJobStuck_ReadsFailedPodLogs(t *testing.T) {
	// CronJobStuck diagnostic → investigator finds the failed Job pod and
	// reports the cause from its logs instead of a kubectl recipe.
	env := &fakeEnv{
		latestPod: "retention-sweep-29012345-abcde",
		logsCur: ai.LogsResult{Lines: []string{
			"connecting to db...",
			"panic: dial tcp 10.0.0.5:5432: connect: connection refused",
		}},
	}
	d := diagnose.Diagnostic{
		Source:   "CronJobStuck",
		Subject:  "CronJob/livekit-agents/retention-sweep",
		Severity: "warning",
		Message:  "CronJob livekit-agents/retention-sweep has not had a successful run in 152h.",
	}
	res, err := RuleBased{}.InvestigateDiagnostic(context.Background(), d, env)
	if err != nil {
		t.Fatal(err)
	}
	if res.Conclusion != ai.ConclusionRootCauseIdentified {
		t.Errorf("conclusion: %q", res.Conclusion)
	}
	if !strings.Contains(res.Summary, "retention-sweep") {
		t.Errorf("summary should name the cronjob; got %q", res.Summary)
	}
	if !strings.Contains(strings.ToLower(res.Summary), "panic") && !strings.Contains(res.Summary, "dependency") {
		t.Errorf("summary should carry the log-derived cause; got %q", res.Summary)
	}
}

func TestRuleBased_CronJobStuck_NoPod_FallsBackToEvents(t *testing.T) {
	env := &fakeEnv{
		latestPod: "", // pods GC'd
		events:    []ai.EventInfo{{Reason: "BackoffLimitExceeded", Message: "Job has reached the specified backoff limit"}},
	}
	d := diagnose.Diagnostic{
		Source:  "CronJobStuck",
		Subject: "CronJob/ns/sweep", Severity: "warning",
		Message: "CronJob ns/sweep has not had a successful run in 152h.",
	}
	res, _ := RuleBased{}.InvestigateDiagnostic(context.Background(), d, env)
	if !strings.Contains(res.Summary, "BackoffLimitExceeded") {
		t.Errorf("should fall back to events; got %q", res.Summary)
	}
}

func TestRuleBased_CronJob_GCdPods_ReadsJobEvents(t *testing.T) {
	// Pods garbage-collected, but the Job survives and its events name the
	// start failure (missing Secret) — the investigator must surface it.
	env := &fakeEnv{
		latestPod: "", // pods GC'd
		latestJob: "wp-verify-weekly-29700240",
		events: []ai.EventInfo{
			{Reason: "FailedCreate", Message: "Error creating: secret \"verify-creds\" not found"},
		},
	}
	d := diagnose.Diagnostic{
		Source: "CronJobStuck", Subject: "CronJob/storethesoup/wp-verify-weekly", Severity: "warning",
		Message: "CronJob storethesoup/wp-verify-weekly has not had a successful run in 85h.",
	}
	res, _ := RuleBased{}.InvestigateDiagnostic(context.Background(), d, env)
	if !strings.Contains(res.Summary, "not found") || !strings.Contains(res.Summary, "verify-creds") {
		t.Errorf("should surface the Job's missing-secret start failure; got %q", res.Summary)
	}
}

func TestRuleBased_ImagePull_PrefersDetailedEvent(t *testing.T) {
	// kubelet emits BOTH a generic "Error: ImagePullBackOff" (newest) and a
	// detailed "Failed to pull ... pull access denied" — surface the detailed one.
	env := &fakeEnv{
		logsPrev: ai.LogsResult{Error: "ImagePullBackOff"},
		logsCur:  ai.LogsResult{Error: "ImagePullBackOff"},
		events: []ai.EventInfo{
			{Reason: "Failed", Message: "Error: ImagePullBackOff"}, // newest, generic
			{Reason: "Failed", Message: "Failed to pull image \"x:v0\": pull access denied, repository does not exist"},
		},
	}
	f := probe.Finding{Component: "CrashLoopBackOff", Severity: probe.SeverityCritical,
		Message: "Pod ns/p matched pattern ImagePullBackOff"}
	res, _ := RuleBased{}.InvestigateFinding(context.Background(), f, env)
	if !strings.Contains(res.Summary, "pull access denied") {
		t.Errorf("should surface the detailed pull error; got %q", res.Summary)
	}
}

func TestContainerWaitingCause_PrefersDetailed(t *testing.T) {
	notes := []string{
		"container app: img",
		"container app waiting: ImagePullBackOff — Back-off pulling image: pull access denied, repository does not exist",
	}
	if got := containerWaitingCause(notes); !strings.Contains(got, "pull access denied") {
		t.Errorf("got %q", got)
	}
}
