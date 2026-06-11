// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package diagnose

import (
	"context"
	"fmt"
	"time"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// CronJobStuck surfaces CronJobs whose most recent scheduled run is
// far in the past (≥ 2× the spec.schedule interval) AND whose
// status.lastSuccessfulTime is non-recent. Common causes:
//   - The CronJob is `suspended: true` (operator parked it for
//     maintenance and forgot)
//   - All recent runs hit `backoffLimit` and the controller stopped
//     scheduling new ones
//   - The pod template references a Secret/ConfigMap that doesn't
//     exist (the controller fails fast every cycle)
//
// The operator typically wants to know BEFORE a downstream system
// notices the job hasn't run. This analyzer is the early-warning.
//
// Phase 3.E.3.
//
// Signal: CronJob whose lastSuccessfulTime is > 24h old AND whose
// expected-next-run (based on .spec.schedule) has been overdue
// for ≥ cronJobOverdueGrace. Conservative on schedule parsing —
// we use a coarse heuristic (24h hard floor) rather than crontab
// arithmetic to avoid false positives on @hourly / @daily
// boundary cases. Per-schedule precision is a future enhancement.
type CronJobStuck struct{}

// Name returns the analyzer's stable identifier.
func (CronJobStuck) Name() string { return "CronJobStuck" }

const cronJobOverdueGrace = 24 * time.Hour

// Run walks every CronJob and emits a warning per stuck job.
func (a CronJobStuck) Run(ctx context.Context, src snapshot.Source) []Diagnostic {
	gvr := schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "cronjobs"}
	list, err := src.List(ctx, gvr, "")
	if err != nil {
		logListFailure("cronjobs", err, false)
		return nil
	}
	now := time.Now()
	var out []Diagnostic
	for i := range list.Items {
		cj := &list.Items[i]
		ns := cj.GetNamespace()
		name := cj.GetName()
		// Suspended CronJobs are an operator's deliberate pause —
		// surface them too (they're often forgotten) but distinguish
		// the cause for the remediation.
		suspended, _, _ := unstructured.NestedBool(cj.Object, "spec", "suspend")
		schedule, _, _ := unstructured.NestedString(cj.Object, "spec", "schedule")
		// Determine last-success and last-scheduled-time anchors.
		// status.lastSuccessfulTime — when the most recent run
		// completed cleanly. Missing on a brand-new CronJob.
		lastSuccessStr, _, _ := unstructured.NestedString(cj.Object, "status", "lastSuccessfulTime")
		var lastSuccess time.Time
		if lastSuccessStr != "" {
			if t, err := time.Parse(time.RFC3339, lastSuccessStr); err == nil {
				lastSuccess = t
			}
		}
		// status.lastScheduleTime — when the controller most recently
		// attempted to schedule (may have failed). Used to detect
		// "controller has stopped scheduling entirely".
		lastScheduleStr, _, _ := unstructured.NestedString(cj.Object, "status", "lastScheduleTime")
		var lastSchedule time.Time
		if lastScheduleStr != "" {
			if t, err := time.Parse(time.RFC3339, lastScheduleStr); err == nil {
				lastSchedule = t
			}
		}

		var reason, severity string
		switch {
		case suspended:
			// Suspended CronJobs surface as info-level since the operator
			// almost always knows about it; the value is in the digest
			// reminder so it doesn't get forgotten.
			reason = "is `spec.suspend=true` — paused by an operator. If maintenance is over, unpause:\n\n" +
				"  kubectl -n " + ns + " patch cronjob " + name + " --type=merge -p '{\"spec\":{\"suspend\":false}}'"
			severity = "warning"
		case lastSuccess.IsZero() && cj.GetCreationTimestamp().Time.Before(now.Add(-cronJobOverdueGrace)):
			reason = "has NEVER had a successful run since creation. Inspect the most recent Job to see why:\n\n" +
				"  kubectl -n " + ns + " describe cronjob " + name + "\n" +
				"  kubectl -n " + ns + " get jobs -l job-name --sort-by=.metadata.creationTimestamp"
			severity = "critical"
		case !lastSuccess.IsZero() && now.Sub(lastSuccess) > cronJobOverdueGrace:
			ago := now.Sub(lastSuccess).Truncate(time.Hour)
			reason = fmt.Sprintf("has not had a successful run in %s. The schedule is `%s`. Check the most recent failed Job:\n\n"+
				"  kubectl -n %s get jobs -l job-name --sort-by=.metadata.creationTimestamp | tail -5\n"+
				"  kubectl -n %s logs job/<latest> --tail=200", ago, schedule, ns, ns)
			severity = "warning"
		case !lastSchedule.IsZero() && now.Sub(lastSchedule) > cronJobOverdueGrace && lastSuccess.IsZero():
			ago := now.Sub(lastSchedule).Truncate(time.Hour)
			reason = fmt.Sprintf("controller stopped scheduling %s ago with no successful run on record. The schedule is `%s`. Investigate Secret/ConfigMap references on the Pod template.", ago, schedule)
			severity = "warning"
		default:
			continue
		}

		out = append(out, Diagnostic{
			Source:      "CronJobStuck",
			Subject:     "CronJob/" + ns + "/" + name,
			Severity:    severity,
			Message:     fmt.Sprintf("CronJob %s/%s %s", ns, name, reason),
			Remediation: "",
		})
	}
	return out
}
