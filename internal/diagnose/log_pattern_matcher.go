// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package diagnose

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/srenix-ai/agentic-sre/internal/snapshot"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// LogPatternMatcher is the M3 analyzer (trigger-expansion roadmap, v1.7+).
//
// Scans recent Event messages cluster-wide for high-signal failure
// patterns that K8s itself doesn't bubble up as a Pod condition:
//
//   - "ImagePullBackOff" — image registry / pull-secret / digest miss
//   - "OOMKilled" — single-shot OOM (OOMKillRecurrence catches the
//     ≥3-restart pattern; this catches the first hit before recurrence)
//   - "Liveness probe failed" / "Readiness probe failed" with a
//     hostname → typically misconfigured probe target
//   - "Failed to attach volume" — CSI driver attach failures (Ceph,
//     EBS, local-path)
//   - "Forbidden" — RBAC mismatches surfaced via events but invisible
//     in pod status until the controller crashes
//
// Each match produces one diagnostic per (involved-object, pattern)
// pair, with the matching event message included verbatim so the
// operator can search for the root cause directly. Opts out via
// SRENIX_ANALYZER_LOG_PATTERN_MATCHER=off.
type LogPatternMatcher struct{}

// Name satisfies the Analyzer contract.
func (LogPatternMatcher) Name() string { return "LogPatternMatcher" }

// logPattern is a compiled pattern + the severity the analyzer
// emits when it matches. The label gets stamped into the
// Diagnostic.Source for downstream filtering (e.g., classify
// "LogPatternMatcher.ImagePullBackOff" separately).
type logPattern struct {
	label    string
	re       *regexp.Regexp
	severity string
	remed    string
	// escalate marks a warning pattern that is promoted to critical when the
	// workload is GENUINELY unhealthy rather than a transient, self-healed
	// blip. Only ProbeFailed uses it: a liveness probe that flaps once and
	// recovers is advisory (Kubernetes' own restart already remediated it),
	// but one that keeps the pod NotReady, drives a restart loop, or recurs
	// many times in the window needs a human. See escalatedSeverity.
	escalate bool
}

// patterns are the canonical, low-false-positive matchers. Anchoring is
// kept loose (`(?i)` case-insensitive, no `^`/`$`) so vendor-specific
// suffixes don't miss the signal.
var logPatterns = []logPattern{
	{
		label: "ImagePullBackOff",
		re:    regexp.MustCompile(`(?i)(ImagePullBackOff|ErrImagePull|manifest unknown)`),
		// Critical: a container kubelet has backed off pulling cannot start —
		// the workload is down for as long as the pull keeps failing. This is
		// the event-based companion to the status-based ImagePullAuth analyzer;
		// both now agree on Critical so a stuck pull reaches the human-action
		// channel regardless of which one wins the per-subject dedup.
		severity: "critical",
		remed:    "Confirm the image tag/digest exists in the registry, then verify the imagePullSecret is mounted and valid: kubectl describe pod <pod>",
	},
	{
		label:    "OOMKilled",
		re:       regexp.MustCompile(`(?i)OOMKilled`),
		severity: "warning",
		remed:    "Raise resources.limits.memory or investigate the workload's memory pressure: kubectl top pod <pod>",
	},
	{
		label:    "ProbeFailed",
		re:       regexp.MustCompile(`(?i)(Liveness probe failed|Readiness probe failed|Startup probe failed)`),
		severity: "warning",
		escalate: true,
		remed:    "Verify the probe URL/port is correct + the workload's startup is fast enough: kubectl describe pod <pod>",
	},
	{
		label:    "VolumeAttachFailed",
		re:       regexp.MustCompile(`(?i)(Failed to attach volume|AttachVolume\.Attach failed|MountVolume\.SetUp failed)`),
		severity: "critical",
		remed:    "Inspect the CSI driver pods + the PV/PVC state: kubectl describe pvc <pvc>; kubectl logs -n <csi-ns> -l app=csi-driver",
	},
	{
		label:    "Forbidden",
		re:       regexp.MustCompile(`(?i)\bForbidden\b.*(?:cannot|unauthorized)`),
		severity: "warning",
		remed:    "RBAC mismatch — inspect the ServiceAccount's bindings: kubectl get rolebinding,clusterrolebinding -A -o wide | grep <sa>",
	},
}

// Run satisfies the Analyzer contract. Walks recent Events and emits
// one diagnostic per (involved-object, pattern) match. Dedups by
// (involved-object, label) so a pod with 30 OOMKilled events surfaces
// once, not 30 times.
func (a LogPatternMatcher) Run(ctx context.Context, src snapshot.Source) []Diagnostic {
	events, err := src.List(ctx, snapshot.GVREvent, "")
	if err != nil {
		logListFailure("events", err, false)
		return nil
	}
	type key struct{ subject, label string }
	type agg struct {
		invKind, invNS, invName, subject string
		pat                              logPattern
		occurrences                      int    // sum of event aggregation counts
		sampleMsg                        string // first matching message (stable)
	}
	aggs := make(map[key]*agg)
	var order []key // preserve first-seen order for deterministic output
	for i := range events.Items {
		e := &events.Items[i]
		msg, _, _ := unstructured.NestedString(e.Object, "message")
		if msg == "" {
			continue
		}
		invKind, _, _ := unstructured.NestedString(e.Object, "involvedObject", "kind")
		invName, _, _ := unstructured.NestedString(e.Object, "involvedObject", "name")
		invNS, _, _ := unstructured.NestedString(e.Object, "involvedObject", "namespace")
		if invKind == "" || invName == "" {
			continue
		}
		subject := invKind + "/" + invNS + "/" + invName
		occ := eventOccurrences(e)
		for idx := range logPatterns {
			p := logPatterns[idx]
			if !p.re.MatchString(msg) {
				continue
			}
			k := key{subject: subject, label: p.label}
			if a, ok := aggs[k]; ok {
				a.occurrences += occ // dedup per (object,label) but COUNT recurrence
				continue
			}
			aggs[k] = &agg{invKind, invNS, invName, subject, p, occ, msg}
			order = append(order, k)
		}
	}

	out := make([]Diagnostic, 0, len(order))
	for _, k := range order {
		a := aggs[k]
		severity := a.pat.severity
		reason := ""
		if a.pat.escalate {
			severity, reason = escalatedSeverity(ctx, src, a.invKind, a.invNS, a.invName, a.occurrences, a.pat.severity)
		}
		// The escalation reason is appended to the message as a STABLE marker
		// (a boolean state, never the fluctuating exact count) so it doesn't
		// churn the cross-cycle dedup fingerprint — only the genuine
		// warning→critical transition re-alerts, which is intended.
		suffix := ""
		if reason != "" {
			suffix = " — escalated: " + reason
		}
		out = append(out, Diagnostic{
			Source:   "LogPatternMatcher." + a.pat.label,
			Subject:  a.subject,
			Severity: severity,
			Message: fmt.Sprintf(
				"%s event on %s — matched pattern %s: %s%s",
				a.invKind, a.subject, a.pat.label, truncateLogMsg(a.sampleMsg, 220), suffix),
			Remediation: a.pat.remed,
		})
	}
	return out
}

// Escalation thresholds for self-healing patterns (ProbeFailed).
const (
	// probeRecurrenceThreshold: matching events in the window beyond which a
	// probe failure is treated as persistent (flapping), not a one-off blip.
	probeRecurrenceThreshold = 5
	// probeRestartThreshold: container restart count indicating a restart loop.
	probeRestartThreshold = 3
)

// eventOccurrences reads an Event's aggregation count (.count, then
// .series.count), defaulting to 1 when absent. Kubelets that aggregate repeats
// report count>1; those that emit a distinct Event per occurrence are summed by
// the caller across the matching events.
func eventOccurrences(e *unstructured.Unstructured) int {
	if c, ok, _ := unstructured.NestedInt64(e.Object, "count"); ok && c > 0 {
		return int(c)
	}
	if c, ok, _ := unstructured.NestedInt64(e.Object, "series", "count"); ok && c > 0 {
		return int(c)
	}
	return 1
}

// escalatedSeverity promotes base (warning) to critical when the workload is
// genuinely unhealthy rather than recovered, and returns a short STABLE reason
// for the message. A one-off probe blip on a now-Ready pod keeps base severity
// (reason ""). Escalates when: the failure recurs past the threshold, OR the
// live Pod is NotReady, OR its restart count indicates a loop. When the live
// Pod can't be read, it does NOT over-escalate (returns base) — recurrence
// alone still escalates.
func escalatedSeverity(ctx context.Context, src snapshot.Source, invKind, ns, name string, occurrences int, base string) (severity, reason string) {
	if occurrences >= probeRecurrenceThreshold {
		return "critical", "recurring"
	}
	if invKind != "Pod" || ns == "" || name == "" {
		return base, ""
	}
	pod, err := src.Get(ctx, snapshot.GVRPod, ns, name)
	if err != nil || pod == nil {
		return base, "" // can't confirm live state → don't over-escalate
	}
	notReady, maxRestarts := podHealthSignals(pod)
	switch {
	case notReady:
		return "critical", "pod NotReady"
	case maxRestarts >= probeRestartThreshold:
		return "critical", "restart-looping"
	}
	return base, ""
}

// podHealthSignals reports whether any container is NOT ready and the maximum
// restartCount across containers. A pod with no containerStatuses yet returns
// (false, 0) so a not-yet-scheduled pod isn't mistaken for unhealthy.
func podHealthSignals(pod *unstructured.Unstructured) (notReady bool, maxRestarts int) {
	css, _, _ := unstructured.NestedSlice(pod.Object, "status", "containerStatuses")
	for _, raw := range css {
		cs, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if ready, _ := cs["ready"].(bool); !ready {
			notReady = true
		}
		switch rc := cs["restartCount"].(type) {
		case int64:
			if int(rc) > maxRestarts {
				maxRestarts = int(rc)
			}
		case float64:
			if int(rc) > maxRestarts {
				maxRestarts = int(rc)
			}
		}
	}
	return notReady, maxRestarts
}

// truncateLogMsg returns msg with a soft cap at n runes — long stack
// traces or container-exit-code dumps would otherwise dominate the
// Slack render budget. Adds an ellipsis when truncated.
func truncateLogMsg(msg string, n int) string {
	msg = strings.ReplaceAll(msg, "\n", " ")
	if len(msg) <= n {
		return msg
	}
	return msg[:n] + "…"
}
