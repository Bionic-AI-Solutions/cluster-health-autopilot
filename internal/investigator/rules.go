// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package investigator

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/srenix-ai/agentic-sre/pkg/ai"
	"github.com/srenix-ai/agentic-sre/pkg/diagnose"
	"github.com/srenix-ai/agentic-sre/pkg/probe"
)

// RuleBased is the deterministic Investigator implementation that ships in
// the OSS catalog. It pattern-matches the failure mode in a Finding /
// Diagnostic and runs a fixed set of follow-up tools via Environment to
// produce a Summary + Observations + Conclusion.
//
// Rules are intentionally narrow: each one targets one well-known failure
// pattern. Unmatched failures return an empty result so the original alert
// surfaces unchanged. The LLM-backed Investigator (paid Srenix Enterprise) handles
// the long tail.
type RuleBased struct{}

// Name returns the investigator's identifier.
func (RuleBased) Name() string { return "RuleBased" }

var _ ai.Investigator = RuleBased{}

// InvestigateFinding routes a probe.Finding to the matching rule.
func (r RuleBased) InvestigateFinding(ctx context.Context, f probe.Finding, env ai.Environment) (ai.InvestigationResult, error) {
	start := time.Now()
	res, err := r.investigateFinding(ctx, f, env)
	res.Cost.WallTime = time.Since(start)
	res.Cost.ToolCalls = len(res.Observations)
	res.Summary = clipSummary(res.Summary)
	return res, err
}

// InvestigateDiagnostic routes a diagnose.Diagnostic to the matching rule.
func (r RuleBased) InvestigateDiagnostic(ctx context.Context, d diagnose.Diagnostic, env ai.Environment) (ai.InvestigationResult, error) {
	start := time.Now()
	res, err := r.investigateDiagnostic(ctx, d, env)
	res.Cost.WallTime = time.Since(start)
	res.Cost.ToolCalls = len(res.Observations)
	res.Summary = clipSummary(res.Summary)
	return res, err
}

// ── Finding rules ──────────────────────────────────────────────────────────

func (RuleBased) investigateFinding(ctx context.Context, f probe.Finding, env ai.Environment) (ai.InvestigationResult, error) {
	msg := f.Message
	low := strings.ToLower(msg)
	target := extractURL(msg)

	switch {
	case isCrashClass(low):
		ns, pod := parsePodRef(msg)
		return investigateCrash(ctx, ns, pod, msg, env)
	case strings.Contains(low, "tls verification failed"):
		return investigateTLS(ctx, target, env, msg)
	case strings.Contains(low, "context deadline exceeded"),
		strings.Contains(low, "connection refused"),
		strings.Contains(low, "i/o timeout"),
		strings.Contains(low, "no such host"),
		strings.Contains(low, "eof"):
		return investigateConnectivity(ctx, target, env, msg)
	case strings.Contains(low, "expected ") && strings.Contains(low, "http "):
		return investigateStatusMismatch(ctx, target, env, msg)
	}
	return ai.InvestigationResult{}, nil
}

// investigateTLS runs tls_inspect + dns_lookup against the failing host and
// classifies the cert problem.
func investigateTLS(ctx context.Context, target string, env ai.Environment, originalMsg string) (ai.InvestigationResult, error) {
	host, port := urlHostPort(target, 443)
	if host == "" {
		return ai.InvestigationResult{Conclusion: ai.ConclusionInsufficientData}, nil
	}
	res := ai.InvestigationResult{}
	tlsR, _ := env.TLSInspect(ctx, host, port)
	res.Observations = append(res.Observations, ai.Observation{
		Tool: "tls_inspect", Args: fmt.Sprintf("%s:%d", host, port),
		Result: tlsObsResult(tlsR), Elapsed: tlsR.Elapsed,
	})
	dnsR, _ := env.DNSLookup(ctx, host)
	res.Observations = append(res.Observations, ai.Observation{
		Tool: "dns_lookup", Args: host,
		Result: dnsObsResult(dnsR), Elapsed: dnsR.Elapsed,
	})

	// Classify
	switch {
	case tlsR.Expired:
		res.Conclusion = ai.ConclusionRootCauseIdentified
		res.Summary = fmt.Sprintf("Cert served on %s expired on %s (issuer %s). cert-manager / Ingress likely points at a stale Secret or has not completed renewal.",
			host, tlsR.NotAfter.Format("2006-01-02"), tlsR.Issuer)
	case !tlsR.HostMatches && len(tlsR.SANs) > 0:
		res.Conclusion = ai.ConclusionRootCauseIdentified
		res.Summary = fmt.Sprintf("Cert served on %s is valid but does not include %s in its SANs (cert SANs: %s). Likely a fallback / wrong-Ingress cert.",
			host, host, strings.Join(tlsR.SANs, ", "))
	case tlsR.HandshakeErr != "":
		res.Conclusion = ai.ConclusionConfirmedOutage
		res.Summary = fmt.Sprintf("TLS handshake to %s:%d failed: %s. Check Kong / Ingress controller logs.",
			host, port, tlsR.HandshakeErr)
	default:
		res.Conclusion = ai.ConclusionInsufficientData
		res.Summary = fmt.Sprintf("TLS handshake to %s:%d succeeded against server-served cert; original verification error may have been transient.",
			host, port)
	}
	return res, nil
}

// investigateConnectivity runs dns_lookup + http_probe (with insecure
// fallback) and distinguishes DNS vs network vs app-layer failures.
func investigateConnectivity(ctx context.Context, target string, env ai.Environment, originalMsg string) (ai.InvestigationResult, error) {
	host, _ := urlHostPort(target, 443)
	res := ai.InvestigationResult{}

	if host != "" {
		dnsR, _ := env.DNSLookup(ctx, host)
		res.Observations = append(res.Observations, ai.Observation{
			Tool: "dns_lookup", Args: host, Result: dnsObsResult(dnsR), Elapsed: dnsR.Elapsed,
		})
		if dnsR.Error != "" {
			res.Conclusion = ai.ConclusionRootCauseIdentified
			res.Summary = fmt.Sprintf("DNS resolution for %s failed: %s. Check Cloudflare / upstream DNS records.",
				host, dnsR.Error)
			return res, nil
		}
		if dnsR.Elapsed > 1500*time.Millisecond {
			res.Conclusion = ai.ConclusionRootCauseIdentified
			res.Summary = fmt.Sprintf("DNS resolution for %s took %v (>1.5s) — likely CoreDNS contention or slow upstream resolver. Probe timeouts under this load are expected.",
				host, dnsR.Elapsed.Round(time.Millisecond))
			return res, nil
		}
	}

	if target != "" {
		// First probe with strict TLS; fall back to insecure to learn HTTP state.
		strict, _ := env.HTTPProbe(ctx, target, ai.HTTPProbeOpts{Timeout: 8 * time.Second})
		res.Observations = append(res.Observations, ai.Observation{
			Tool: "http_probe", Args: target, Result: httpObsResult(strict), Elapsed: strict.ResponseTime,
		})
		if strict.StatusCode > 0 {
			res.Conclusion = ai.ConclusionLikelyTransient
			res.Summary = fmt.Sprintf("Follow-up probe of %s returned HTTP %d in %v — the original error was transient (likely brief network blip or backend pause).",
				target, strict.StatusCode, strict.ResponseTime.Round(time.Millisecond))
			return res, nil
		}
		insecure, _ := env.HTTPProbe(ctx, target, ai.HTTPProbeOpts{Timeout: 8 * time.Second, InsecureSkipVerify: true})
		res.Observations = append(res.Observations, ai.Observation{
			Tool: "http_probe", Args: target + " (TLS skipped)", Result: httpObsResult(insecure), Elapsed: insecure.ResponseTime,
		})
		if insecure.StatusCode > 0 {
			res.Conclusion = ai.ConclusionRootCauseIdentified
			res.Summary = fmt.Sprintf("Connection succeeds with TLS verification disabled (HTTP %d) but fails with TLS on. Cert presentation is the root cause — likely SAN mismatch or expiry.",
				insecure.StatusCode)
			return res, nil
		}
	}

	res.Conclusion = ai.ConclusionConfirmedOutage
	res.Summary = fmt.Sprintf("Host %s remains unreachable on follow-up. DNS resolves but no HTTP response — likely backend (Kong / Service / Pod) outage.", host)
	return res, nil
}

// investigateStatusMismatch confirms whether the HTTP status mismatch
// reproduces or has already cleared.
func investigateStatusMismatch(ctx context.Context, target string, env ai.Environment, originalMsg string) (ai.InvestigationResult, error) {
	if target == "" {
		return ai.InvestigationResult{Conclusion: ai.ConclusionInsufficientData}, nil
	}
	res := ai.InvestigationResult{}
	probeR, _ := env.HTTPProbe(ctx, target, ai.HTTPProbeOpts{Timeout: 5 * time.Second})
	res.Observations = append(res.Observations, ai.Observation{
		Tool: "http_probe", Args: target, Result: httpObsResult(probeR), Elapsed: probeR.ResponseTime,
	})

	expected := extractExpectedStatus(originalMsg)
	got := probeR.StatusCode
	switch {
	case got == 0:
		res.Conclusion = ai.ConclusionConfirmedOutage
		res.Summary = fmt.Sprintf("Follow-up probe of %s failed: %s. Backend may have just gone down further.",
			target, probeR.Error)
	case expected > 0 && got == expected:
		res.Conclusion = ai.ConclusionLikelyTransient
		res.Summary = fmt.Sprintf("%s now returns expected HTTP %d on follow-up — the original status mismatch was transient.", target, got)
	case got >= 500:
		res.Conclusion = ai.ConclusionConfirmedOutage
		res.Summary = fmt.Sprintf("%s returning HTTP %d (server error). Check backend deployment, recent rollouts, and pod restart counts.", target, got)
	case got == 401 || got == 403:
		res.Conclusion = ai.ConclusionRootCauseIdentified
		res.Summary = fmt.Sprintf("%s returns HTTP %d (auth-required). The endpoint expects credentials; the probe is hitting an auth-walled path.", target, got)
	default:
		res.Conclusion = ai.ConclusionConfirmedOutage
		res.Summary = fmt.Sprintf("%s returns HTTP %d (expected %d). Verify Ingress rules and backend service Endpoints.", target, got, expected)
	}
	return res, nil
}

// ── Diagnostic rules ───────────────────────────────────────────────────────

func (RuleBased) investigateDiagnostic(ctx context.Context, d diagnose.Diagnostic, env ai.Environment) (ai.InvestigationResult, error) {
	subj := d.Subject
	msg := d.Message
	src := d.Source

	// Pattern: ExternalSecret/<ns>/<name>
	if strings.HasPrefix(subj, "ExternalSecret/") {
		return investigateExternalSecret(ctx, subj, env, msg)
	}
	// Pattern: ingress-coverage/<ns>/<name>/<host>  (rare since the analyzer
	// was removed, but support kept for paid catalog patterns).
	if strings.HasPrefix(subj, "ingress-coverage/") || strings.HasPrefix(subj, "endpoint-coverage/") {
		return investigateCoverage(ctx, subj, env, msg)
	}
	// Pattern: missing-key/<ns>/<secret>/<key> or missing-secret/<ns>/<name>
	if strings.HasPrefix(subj, "missing-key/") || strings.HasPrefix(subj, "missing-secret/") || strings.HasPrefix(subj, "unprovisioned/") {
		return investigateMissingSecret(ctx, subj, env, msg)
	}
	// Pattern: cert-expiry/<ns>/<name>
	if strings.HasPrefix(subj, "cert-expiry/") {
		return investigateCertExpiry(ctx, subj, env)
	}
	// Pattern: CronJob/<ns>/<name> (CronJobStuck) — read the failed Job pod's
	// logs and report WHY it keeps failing instead of a kubectl recipe.
	if strings.HasPrefix(subj, "CronJob/") {
		ns, name := parseKindNsName(subj)
		return investigateCronJob(ctx, ns, name, msg, env)
	}
	// Crash-class pod findings (FailedPods, log-pattern crashes): read the
	// container logs and classify the cause rather than punting to the human.
	if isCrashClass(strings.ToLower(msg)) {
		ns, pod := parseKindNsName(subj) // structured "Pod/<ns>/<pod>" subject
		if ns == "" {
			ns, pod = parsePodRef(msg)
		}
		return investigateCrash(ctx, ns, pod, msg, env)
	}
	_ = src // reserved for future per-source heuristics
	return ai.InvestigationResult{}, nil
}

func investigateExternalSecret(ctx context.Context, subj string, env ai.Environment, originalMsg string) (ai.InvestigationResult, error) {
	parts := strings.Split(subj, "/")
	if len(parts) < 3 {
		return ai.InvestigationResult{Conclusion: ai.ConclusionInsufficientData}, nil
	}
	ns, name := parts[1], parts[2]
	res := ai.InvestigationResult{}
	desc, _ := env.Describe(ctx, "ExternalSecret", ns, name)
	res.Observations = append(res.Observations, ai.Observation{
		Tool: "describe", Args: fmt.Sprintf("ExternalSecret/%s/%s", ns, name), Result: describeObsResult(desc),
	})
	events, _ := env.GetEvents(ctx, ns, "ExternalSecret", name, 4*time.Hour)
	if len(events) > 0 {
		res.Observations = append(res.Observations, ai.Observation{
			Tool: "events", Args: fmt.Sprintf("ExternalSecret/%s/%s", ns, name),
			Result: eventsObsResult(events),
		})
	}
	if desc.Error != "" {
		res.Conclusion = ai.ConclusionRootCauseIdentified
		res.Summary = fmt.Sprintf("ExternalSecret %s/%s no longer exists in the cluster. The DriftReport is stale and should clear on the next analyzer run.", ns, name)
		return res, nil
	}
	res.Conclusion = ai.ConclusionRootCauseIdentified
	res.Summary = fmt.Sprintf("ExternalSecret %s/%s: status=%q reason=%q. Check the Vault path / property names in spec.data[] — see the original analyzer message for the failing key.", ns, name, desc.Status, desc.Reason)
	return res, nil
}

func investigateCoverage(ctx context.Context, subj string, env ai.Environment, originalMsg string) (ai.InvestigationResult, error) {
	// Best-effort: the original message has the host; probe it lightly.
	host := extractHostnameFromMessage(originalMsg)
	if host == "" {
		return ai.InvestigationResult{Conclusion: ai.ConclusionInsufficientData}, nil
	}
	res := ai.InvestigationResult{}
	dnsR, _ := env.DNSLookup(ctx, host)
	res.Observations = append(res.Observations, ai.Observation{
		Tool: "dns_lookup", Args: host, Result: dnsObsResult(dnsR), Elapsed: dnsR.Elapsed,
	})
	res.Conclusion = ai.ConclusionInsufficientData
	res.Summary = fmt.Sprintf("Coverage gap on %s. DNS resolves to %v — add an external probe or accept this is internal-only.", host, dnsR.Addresses)
	return res, nil
}

func investigateMissingSecret(ctx context.Context, subj string, env ai.Environment, originalMsg string) (ai.InvestigationResult, error) {
	parts := strings.Split(subj, "/")
	if len(parts) < 3 {
		return ai.InvestigationResult{Conclusion: ai.ConclusionInsufficientData}, nil
	}
	ns, name := parts[1], parts[2]
	res := ai.InvestigationResult{}
	desc, _ := env.Describe(ctx, "Secret", ns, name)
	res.Observations = append(res.Observations, ai.Observation{
		Tool: "describe", Args: fmt.Sprintf("Secret/%s/%s", ns, name), Result: describeObsResult(desc),
	})
	if desc.Error != "" {
		res.Conclusion = ai.ConclusionRootCauseIdentified
		res.Summary = fmt.Sprintf("Secret %s/%s does not exist. Create an ExternalSecret pointing at the canonical Vault path, or remove the consuming envFrom reference.", ns, name)
		return res, nil
	}
	res.Conclusion = ai.ConclusionRootCauseIdentified
	res.Summary = fmt.Sprintf("Secret %s/%s exists but key from the analyzer message is missing or case-mismatched. Compare existing Secret keys to the consumer's env reference.", ns, name)
	return res, nil
}

func investigateCertExpiry(ctx context.Context, subj string, env ai.Environment) (ai.InvestigationResult, error) {
	parts := strings.Split(subj, "/")
	if len(parts) < 3 {
		return ai.InvestigationResult{Conclusion: ai.ConclusionInsufficientData}, nil
	}
	ns, name := parts[1], parts[2]
	res := ai.InvestigationResult{}
	desc, _ := env.Describe(ctx, "Certificate", ns, name)
	res.Observations = append(res.Observations, ai.Observation{
		Tool: "describe", Args: fmt.Sprintf("Certificate/%s/%s", ns, name), Result: describeObsResult(desc),
	})
	events, _ := env.GetEvents(ctx, ns, "Certificate", name, 24*time.Hour)
	if len(events) > 0 {
		res.Observations = append(res.Observations, ai.Observation{
			Tool: "events", Args: fmt.Sprintf("Certificate/%s/%s", ns, name),
			Result: eventsObsResult(events),
		})
	}
	res.Conclusion = ai.ConclusionRootCauseIdentified
	res.Summary = fmt.Sprintf("Certificate %s/%s status=%q reason=%q. Inspect the issuing CertificateRequest + ACME Order; check the ClusterIssuer is reachable.", ns, name, desc.Status, desc.Reason)
	return res, nil
}

// ── helpers ────────────────────────────────────────────────────────────────

// extractURL finds the first http(s) URL inside s. Returns "" when none.
var urlRE = regexp.MustCompile(`https?://[^\s]+`)

func extractURL(s string) string {
	m := urlRE.FindString(s)
	return strings.TrimRight(m, ".,)")
}

func urlHostPort(target string, defaultPort int) (string, int) {
	if target == "" {
		return "", 0
	}
	u, err := url.Parse(target)
	if err != nil {
		return "", 0
	}
	host := u.Hostname()
	port := defaultPort
	if p := u.Port(); p != "" {
		if n, err := parseInt(p); err == nil {
			port = n
		}
	}
	return host, port
}

func parseInt(s string) (int, error) {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("not numeric")
		}
		n = n*10 + int(r-'0')
	}
	return n, nil
}

var expectedStatusRE = regexp.MustCompile(`HTTP\s+(\d{3})\s*\(expected\s+(\d{3})`)

func extractExpectedStatus(msg string) int {
	m := expectedStatusRE.FindStringSubmatch(msg)
	if len(m) >= 3 {
		n, _ := parseInt(m[2])
		return n
	}
	return 0
}

var hostInMessageRE = regexp.MustCompile(`\*([a-z0-9][a-z0-9\.\-]+\.[a-z]{2,})\*|host \*?([a-z0-9][a-z0-9\.\-]+\.[a-z]{2,})`)

func extractHostnameFromMessage(msg string) string {
	m := hostInMessageRE.FindStringSubmatch(strings.ToLower(msg))
	for _, g := range m[1:] {
		if g != "" {
			return g
		}
	}
	return ""
}

func tlsObsResult(r ai.TLSResult) string {
	if r.HandshakeErr != "" {
		return "handshake error: " + r.HandshakeErr
	}
	out := fmt.Sprintf("subject=%s issuer=%s notAfter=%s",
		shortSubject(r.Subject), r.Issuer, r.NotAfter.Format("2006-01-02"))
	if r.Expired {
		out += " EXPIRED"
	}
	if !r.HostMatches {
		out += " SAN_MISMATCH"
	}
	if len(r.SANs) > 0 {
		out += " sans=[" + strings.Join(r.SANs, ",") + "]"
	}
	return out
}

func dnsObsResult(r ai.DNSResult) string {
	if r.Error != "" {
		return "error: " + r.Error
	}
	return fmt.Sprintf("→ %s (in %v)", strings.Join(r.Addresses, ","), r.Elapsed.Round(time.Millisecond))
}

func httpObsResult(r ai.HTTPProbeResult) string {
	if r.Error != "" {
		return "error: " + r.Error
	}
	return fmt.Sprintf("HTTP %d in %v (tls_verified=%v)", r.StatusCode, r.ResponseTime.Round(time.Millisecond), r.TLSVerified)
}

func describeObsResult(r ai.DescribeResult) string {
	if r.Error != "" {
		return r.Error
	}
	parts := []string{}
	if r.Status != "" {
		parts = append(parts, "status="+r.Status)
	}
	if r.Reason != "" {
		parts = append(parts, "reason="+r.Reason)
	}
	if r.Message != "" {
		m := r.Message
		if len(m) > 120 {
			m = m[:117] + "..."
		}
		parts = append(parts, "message="+m)
	}
	parts = append(parts, r.Notes...)
	return strings.Join(parts, "; ")
}

func eventsObsResult(events []ai.EventInfo) string {
	parts := []string{}
	for i, ev := range events {
		if i >= 3 {
			parts = append(parts, fmt.Sprintf("…+%d more", len(events)-3))
			break
		}
		parts = append(parts, fmt.Sprintf("%s/%s: %s", ev.Type, ev.Reason, truncate(ev.Message, 80)))
	}
	return strings.Join(parts, " | ")
}

func shortSubject(s string) string {
	for _, part := range strings.Split(s, ",") {
		p := strings.TrimSpace(part)
		if strings.HasPrefix(p, "CN=") {
			return p[3:]
		}
	}
	return s
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

func clipSummary(s string) string {
	if len(s) <= ai.MaxInvestigationSummaryChars {
		return s
	}
	return s[:ai.MaxInvestigationSummaryChars-1] + "…"
}
