// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package investigator

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/ai"
)

// podRef pulls the first "namespace/pod" token out of a finding/diagnostic
// message (probes render "Pod <ns>/<pod> in CrashLoopBackOff (...)" and the
// like). Returns ("","") when no ns/name pair is present.
var podRefRe = regexp.MustCompile(`([a-z0-9][a-z0-9-]*)/([a-z0-9][a-z0-9.-]*)`)

func parsePodRef(msg string) (namespace, pod string) {
	// Anchor on the word "Pod " when present so we don't grab an unrelated
	// "a/b" token; fall back to the first ns/name pair otherwise.
	search := msg
	if i := strings.Index(msg, "Pod "); i >= 0 {
		search = msg[i+len("Pod "):]
	}
	m := podRefRe.FindStringSubmatch(search)
	if len(m) != 3 {
		return "", ""
	}
	return m[1], m[2]
}

// isCrashClass reports whether a finding/diagnostic message describes a pod
// that crashed / cannot start — the class where reading the container logs is
// the fastest path to root cause.
func isCrashClass(low string) bool {
	for _, k := range []string{
		"crashloopbackoff",
		"terminal failed phase",
		"oomkilled",
		"runcontainererror",
		"createcontainererror",
		"error pulling image", // ImagePull paths still benefit from log/event context
	} {
		if strings.Contains(low, k) {
			return true
		}
	}
	return false
}

// investigateCrash fetches the failed container's logs (previous instance
// first, since a CrashLooping pod's current attempt may not have logged yet)
// and classifies the crash cause from the log tail. This is the capability
// that lets CHA answer "WHY did it crash" in the alert instead of telling the
// operator to run kubectl logs --previous themselves.
func investigateCrash(ctx context.Context, ns, pod, originalMsg string, env ai.Environment) (ai.InvestigationResult, error) {
	res := ai.InvestigationResult{}
	if ns == "" || pod == "" {
		res.Conclusion = ai.ConclusionInsufficientData
		return res, nil
	}

	// Previous-instance logs are the smoking gun for CrashLoopBackOff; fall
	// back to current logs when there is no prior terminated instance.
	logs, _ := env.Logs(ctx, ns, pod, ai.LogsOptions{Previous: true})
	if logs.Error != "" || len(logs.Lines) == 0 {
		logs, _ = env.Logs(ctx, ns, pod, ai.LogsOptions{Previous: false})
	}
	res.Observations = append(res.Observations, ai.Observation{
		Tool:   "pod_logs",
		Args:   fmt.Sprintf("%s/%s previous=%v", ns, pod, logs.Previous),
		Result: logsObsResult(logs),
	})

	// When logs are genuinely unavailable, fall back to events so the alert
	// still carries SOMETHING the operator didn't have to fetch.
	if len(logs.Lines) == 0 {
		evs, _ := env.GetEvents(ctx, ns, "Pod", pod, 0)
		if len(evs) > 0 {
			res.Observations = append(res.Observations, ai.Observation{
				Tool: "get_events", Args: fmt.Sprintf("%s/%s", ns, pod),
				Result: evs[0].Reason + ": " + evs[0].Message,
			})
			res.Conclusion = ai.ConclusionInsufficientData
			res.Summary = fmt.Sprintf("Could not read logs for %s/%s (%s). Most recent event: %s — %s.",
				ns, pod, logs.Error, evs[0].Reason, ai.RedactEventMessage(evs[0].Message))
			return res, nil
		}
		res.Conclusion = ai.ConclusionInsufficientData
		res.Summary = fmt.Sprintf("Could not read logs for %s/%s (%s) and no recent events found.", ns, pod, logs.Error)
		return res, nil
	}

	cause := classifyCrashLogs(logs.Lines)
	res.Conclusion = ai.ConclusionRootCauseIdentified
	res.Summary = fmt.Sprintf("%s/%s: %s", ns, pod, cause)
	return res, nil
}

// classifyCrashLogs pattern-matches a container's log tail to a concrete root
// cause. Ordered most-specific-first. Returns a one-line human explanation
// that names the smoking-gun log line where possible.
func classifyCrashLogs(lines []string) string {
	joinedLow := strings.ToLower(strings.Join(lines, "\n"))
	last := lastNonEmpty(lines)

	switch {
	// CLI binary invoked with no/!valid subcommand — prints usage and exits 0.
	// This is the exact shape of CHA's own mis-deployed runner.
	case containsAll(joinedLow, "usage:", "available commands:"),
		strings.Contains(joinedLow, "use \"") && strings.Contains(joinedLow, "[command]"):
		return "container printed CLI usage/help and exited — its Deployment specifies no (or an invalid) command/args, so the binary has no subcommand to run. Fix the workload's command/args (or run it as a Job, not a Deployment)."

	case strings.Contains(joinedLow, "out of memory"),
		strings.Contains(joinedLow, "oomkilled"),
		strings.Contains(joinedLow, "fatal error: runtime: out of memory"),
		strings.Contains(joinedLow, "cannot allocate memory"):
		return "container was killed for exceeding its memory limit (OOM). Raise resources.limits.memory or fix the workload's memory growth. Last line: " + clip(last, 160)

	case strings.Contains(joinedLow, "panic:"):
		return "container crashed with a Go panic: " + clip(firstMatchingLine(lines, "panic:"), 200)

	case strings.Contains(joinedLow, "traceback (most recent call last)"):
		return "container crashed with an unhandled Python exception: " + clip(last, 200)

	case strings.Contains(joinedLow, "permission denied"):
		return "container failed on a permission error (likely filesystem/securityContext or RBAC). Line: " + clip(firstMatchingLine(lines, "permission denied"), 180)

	case strings.Contains(joinedLow, "no such file or directory"):
		return "container referenced a missing file/path (bad mount, missing config, or wrong entrypoint). Line: " + clip(firstMatchingLine(lines, "no such file"), 180)

	case strings.Contains(joinedLow, "connection refused"),
		strings.Contains(joinedLow, "dial tcp"),
		strings.Contains(joinedLow, "no route to host"),
		strings.Contains(joinedLow, "i/o timeout"):
		return "container could not reach a dependency at startup (DB/cache/API). Line: " + clip(firstMatchingLineAny(lines, "connection refused", "dial tcp", "no route to host", "i/o timeout"), 180)

	case strings.Contains(joinedLow, "address already in use"):
		return "container failed to bind its port (address already in use) — likely a port collision or a previous instance still holding the port. Line: " + clip(firstMatchingLine(lines, "address already in use"), 180)

	case strings.Contains(joinedLow, "error") ||
		strings.Contains(joinedLow, "fatal") ||
		strings.Contains(joinedLow, "failed") ||
		strings.Contains(joinedLow, "exception"):
		return "container exited after an error. Last error line: " + clip(firstErrorLine(lines), 200)

	default:
		return "container exited; last log line: " + clip(last, 200)
	}
}

func logsObsResult(l ai.LogsResult) string {
	if l.Error != "" && len(l.Lines) == 0 {
		return "error: " + l.Error
	}
	n := len(l.Lines)
	tail := l.Lines
	if n > 5 {
		tail = l.Lines[n-5:]
	}
	return fmt.Sprintf("%d line(s); tail: %s", n, clip(strings.Join(tail, " | "), 300))
}

func lastNonEmpty(lines []string) string {
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) != "" {
			return lines[i]
		}
	}
	return ""
}

func firstMatchingLine(lines []string, substr string) string {
	low := strings.ToLower(substr)
	for _, l := range lines {
		if strings.Contains(strings.ToLower(l), low) {
			return l
		}
	}
	return lastNonEmpty(lines)
}

func firstMatchingLineAny(lines []string, subs ...string) string {
	for _, l := range lines {
		ll := strings.ToLower(l)
		for _, s := range subs {
			if strings.Contains(ll, s) {
				return l
			}
		}
	}
	return lastNonEmpty(lines)
}

func firstErrorLine(lines []string) string {
	for _, l := range lines {
		ll := strings.ToLower(l)
		if strings.Contains(ll, "error") || strings.Contains(ll, "fatal") ||
			strings.Contains(ll, "failed") || strings.Contains(ll, "exception") {
			return l
		}
	}
	return lastNonEmpty(lines)
}

func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}

func clip(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
