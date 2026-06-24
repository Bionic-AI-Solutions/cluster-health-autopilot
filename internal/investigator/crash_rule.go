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

// parseKindNsName splits a "Kind/namespace/name[/...]" finding SUBJECT into its
// namespace + name. Unlike parsePodRef (which scans free-text messages and
// anchors on "Pod "), this is for the structured subject — so "CronJob/ns/name"
// yields ("ns","name"), not ("ob","ns") from the tail of "CronJob". Returns
// ("","") when the subject lacks at least Kind/ns/name.
func parseKindNsName(subject string) (namespace, name string) {
	parts := strings.SplitN(subject, "/", 4)
	if len(parts) < 3 {
		return "", ""
	}
	return parts[1], parts[2]
}

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
		"imagepullbackoff", // LogPatternMatcher renders "matched pattern ImagePullBackOff"
		"errimagepull",
		"error pulling image",
		"terminal failed phase",
		"oomkilled",
		"runcontainererror",
		"createcontainererror",
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

	// No logs — the pod never started: it was rejected (admission/scheduling)
	// or can't pull its image. The cause lives in the pod's STATUS or its
	// EVENTS, not in logs. Reading the right source is what separates a useful
	// answer from "couldn't determine".
	if len(logs.Lines) == 0 {
		// 1) Events carry the MOST SPECIFIC cause for a pod that never started:
		//    the exact image-pull failure ("Failed to pull image X: manifest
		//    unknown" / "401 Unauthorized") or scheduling failure. Prefer this
		//    over the generic pod status ("containers with unready status").
		if ev := informativeFailureEvent(ctx, env, ns, pod); ev != "" {
			if !messageAlreadyExplains(originalMsg, ev) {
				res.Observations = append(res.Observations, ai.Observation{
					Tool: "get_events", Args: fmt.Sprintf("%s/%s", ns, pod), Result: clip(ev, 200),
				})
				res.Conclusion = ai.ConclusionRootCauseIdentified
				res.Summary = fmt.Sprintf("%s/%s: %s", ns, pod, clip(ev, 240))
				return res, nil
			}
		}

		// 2) Pod status carries admission/scheduling rejection reasons
		//    (e.g. "Pod was rejected: Allocate failed ... nvidia.com/gpu
		//    unavailable"). If the finding MESSAGE already states this, the
		//    message speaks for itself — stay silent rather than repeat it.
		desc, _ := env.Describe(ctx, "Pod", ns, pod)
		statusCause := strings.TrimSpace(desc.Message)
		if statusCause == "" {
			statusCause = strings.TrimSpace(desc.Reason)
		}
		if statusCause != "" && !messageAlreadyExplains(originalMsg, statusCause) {
			res.Observations = append(res.Observations, ai.Observation{
				Tool: "describe", Args: fmt.Sprintf("%s/%s", ns, pod), Result: clip(statusCause, 200),
			})
			res.Conclusion = ai.ConclusionRootCauseIdentified
			res.Summary = fmt.Sprintf("%s/%s rejected: %s", ns, pod, clip(statusCause, 240))
			return res, nil
		}

		// 3) Nothing concrete beyond the finding message — stay SILENT. Emitting
		//    "could not determine" contradicts a self-explanatory message and
		//    makes CHA look like it failed to investigate.
		res.Conclusion = ai.ConclusionInsufficientData
		return res, nil
	}

	cause := classifyCrashLogs(logs.Lines)
	res.Conclusion = ai.ConclusionRootCauseIdentified
	res.Summary = fmt.Sprintf("%s/%s: %s", ns, pod, cause)
	return res, nil
}

// messageAlreadyExplains reports whether the finding message already conveys the
// status cause (so the investigator shouldn't repeat it). Compares on a
// distinctive slice of the cause to tolerate truncation/whitespace differences.
func messageAlreadyExplains(message, cause string) bool {
	cause = strings.TrimSpace(cause)
	if cause == "" {
		return false
	}
	probe := cause
	if len(probe) > 40 {
		probe = probe[:40]
	}
	return strings.Contains(strings.ToLower(message), strings.ToLower(probe))
}

// informativeFailureEvent returns the most useful failure event for a pod that
// produced no logs: a "Failed" event (or one carrying image-pull / error
// keywords) in preference to a generic "BackOff"/"Pulling" line. Empty when no
// event adds information.
func informativeFailureEvent(ctx context.Context, env ai.Environment, ns, pod string) string {
	evs, _ := env.GetEvents(ctx, ns, "Pod", pod, 0)
	keywords := []string{"manifest unknown", "not found", "unauthorized", "401", "403",
		"denied", "no such host", "forbidden", "exceeded", "invalid", "failed to pull"}
	var best string
	for _, e := range evs {
		low := strings.ToLower(e.Reason + " " + e.Message)
		// Strongest signal: a Failed event that names a concrete cause.
		if strings.EqualFold(e.Reason, "Failed") || containsAny(low, keywords) {
			return e.Reason + ": " + ai.RedactEventMessage(e.Message)
		}
		if best == "" && e.Reason != "BackOff" && e.Reason != "Pulling" {
			best = e.Reason + ": " + ai.RedactEventMessage(e.Message)
		}
	}
	return best
}

func containsAny(s string, subs []string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// investigateCronJob explains WHY a stuck CronJob keeps failing by reading the
// logs of its most recent (failed) Job pod, instead of telling the operator to
// go list jobs and tail logs by hand. CronJob "<name>" spawns Jobs that spawn
// pods named "<name>-<job>-<pod>", so the pod is found by name prefix.
func investigateCronJob(ctx context.Context, ns, name, originalMsg string, env ai.Environment) (ai.InvestigationResult, error) {
	res := ai.InvestigationResult{}
	if ns == "" || name == "" {
		res.Conclusion = ai.ConclusionInsufficientData
		return res, nil
	}

	// 1) Best source: the failing Job pod's logs (the command's own output).
	if pod, _ := env.LatestByPrefix(ctx, "Pod", ns, name); pod != "" {
		logs, _ := env.Logs(ctx, ns, pod, ai.LogsOptions{Previous: false})
		if logs.Error != "" || len(logs.Lines) == 0 {
			logs, _ = env.Logs(ctx, ns, pod, ai.LogsOptions{Previous: true})
		}
		if len(logs.Lines) > 0 {
			res.Observations = append(res.Observations, ai.Observation{
				Tool: "pod_logs", Args: fmt.Sprintf("%s/%s (cronjob %s)", ns, pod, name), Result: logsObsResult(logs),
			})
			res.Conclusion = ai.ConclusionRootCauseIdentified
			res.Summary = fmt.Sprintf("CronJob %s/%s failing — last run (pod %s): %s", ns, name, pod, classifyCrashLogs(logs.Lines))
			return res, nil
		}
	}

	// 2) Pods garbage-collected — the JOB outlives them and records WHY it
	//    failed: start failures (missing Secret/ConfigMap, quota, RBAC) live in
	//    the Job's events; "BackoffLimitExceeded"/"DeadlineExceeded" live in the
	//    Job's status. This is how we still know the actual issue without logs.
	if job, _ := env.LatestByPrefix(ctx, "Job", ns, name); job != "" {
		// Job events first — they name concrete start failures.
		if ev := informativeFailureEvent2(ctx, env, ns, "Job", job); ev != "" {
			res.Observations = append(res.Observations, ai.Observation{Tool: "job_events", Args: ns + "/" + job, Result: clip(ev, 200)})
			res.Conclusion = ai.ConclusionRootCauseIdentified
			res.Summary = fmt.Sprintf("CronJob %s/%s failing — last Job %s: %s", ns, name, job, clip(ev, 240))
			return res, nil
		}
		// Job status condition (Failed reason/message).
		if reason, msg := jobFailureCondition(ctx, env, ns, job); reason != "" {
			res.Observations = append(res.Observations, ai.Observation{Tool: "describe", Args: ns + "/" + job, Result: reason + ": " + clip(msg, 160)})
			res.Conclusion = ai.ConclusionRootCauseIdentified
			res.Summary = fmt.Sprintf("CronJob %s/%s failing — last Job %s: %s%s. Pod logs were garbage-collected; raise the Job's ttlSecondsAfterFinished / failedJobsHistoryLimit to capture the failing command's output.",
				ns, name, job, reason, sep(": ", clip(msg, 160)))
			return res, nil
		}
	}

	// 3) Nothing survives — say so honestly, and how to retain evidence.
	if evs, _ := env.GetEvents(ctx, ns, "CronJob", name, 0); len(evs) > 0 {
		res.Conclusion = ai.ConclusionInsufficientData
		res.Summary = fmt.Sprintf("CronJob %s/%s: no surviving Job pod to read. Latest CronJob event: %s — %s.",
			ns, name, evs[0].Reason, ai.RedactEventMessage(evs[0].Message))
		return res, nil
	}
	res.Conclusion = ai.ConclusionInsufficientData
	res.Summary = fmt.Sprintf("CronJob %s/%s has not succeeded, but its Jobs/pods were garbage-collected so the failure can't be read. Raise the Job's ttlSecondsAfterFinished / failedJobsHistoryLimit to retain the next failure for diagnosis.", ns, name)
	return res, nil
}

// informativeFailureEvent2 is informativeFailureEvent for an arbitrary kind
// (Job events name missing-Secret/quota/RBAC start failures).
func informativeFailureEvent2(ctx context.Context, env ai.Environment, ns, kind, name string) string {
	evs, _ := env.GetEvents(ctx, ns, kind, name, 0)
	keywords := []string{"not found", "forbidden", "exceeded", "invalid", "unauthorized",
		"denied", "no such", "failedcreate", "deadline", "backofflimit", "couldn't"}
	var best string
	for _, e := range evs {
		low := strings.ToLower(e.Reason + " " + e.Message)
		if strings.EqualFold(e.Reason, "Failed") || strings.EqualFold(e.Reason, "FailedCreate") || containsAny(low, keywords) {
			return e.Reason + ": " + ai.RedactEventMessage(e.Message)
		}
		if best == "" && e.Reason != "SuccessfulCreate" && e.Reason != "Completed" {
			best = e.Reason + ": " + ai.RedactEventMessage(e.Message)
		}
	}
	return best
}

// jobFailureCondition returns the Failed condition's reason+message from a Job's
// status (e.g. "BackoffLimitExceeded": "Job has reached the specified backoff
// limit"), via Describe. Empty when the Job has no Failed condition.
func jobFailureCondition(ctx context.Context, env ai.Environment, ns, job string) (reason, msg string) {
	d, err := env.Describe(ctx, "Job", ns, job)
	if err != nil {
		return "", ""
	}
	// Describe surfaces the most salient condition into Reason/Message.
	return strings.TrimSpace(d.Reason), strings.TrimSpace(d.Message)
}

func sep(s, v string) string {
	if v == "" {
		return ""
	}
	return s + v
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
