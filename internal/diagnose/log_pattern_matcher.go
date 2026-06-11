// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package diagnose

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
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
// CHA_ANALYZER_LOG_PATTERN_MATCHER=off.
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
}

// patterns are the canonical, low-false-positive matchers. Anchoring is
// kept loose (`(?i)` case-insensitive, no `^`/`$`) so vendor-specific
// suffixes don't miss the signal.
var logPatterns = []logPattern{
	{
		label:    "ImagePullBackOff",
		re:       regexp.MustCompile(`(?i)(ImagePullBackOff|ErrImagePull|manifest unknown)`),
		severity: "warning",
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
	seen := make(map[key]struct{})
	var out []Diagnostic
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
		for _, p := range logPatterns {
			if !p.re.MatchString(msg) {
				continue
			}
			k := key{subject: subject, label: p.label}
			if _, dup := seen[k]; dup {
				continue
			}
			seen[k] = struct{}{}
			out = append(out, Diagnostic{
				Source:   "LogPatternMatcher." + p.label,
				Subject:  subject,
				Severity: p.severity,
				Message: fmt.Sprintf(
					"%s event on %s — matched pattern %s: %s",
					invKind, subject, p.label, truncateLogMsg(msg, 220)),
				Remediation: p.remed,
			})
		}
	}
	return out
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
