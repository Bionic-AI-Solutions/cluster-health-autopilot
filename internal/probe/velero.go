// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package probe

import (
	"context"
	"fmt"
	"time"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Velero reports drift on Velero Backup custom resources:
//
//   - Most-recent backup has phase=Failed / PartiallyFailed →
//     critical
//   - Most-recent backup older than backupSLA (default 26h, slightly
//     past a daily schedule) → critical
//   - Backup in InProgress for > 4h → warning (stuck)
//
// "Most recent" is the latest creationTimestamp on a Backup CR
// across all namespaces. Operators with multiple Schedules get one
// finding per Schedule (we group by metadata.labels
// `velero.io/schedule-name` when present, falling back to the
// Backup's own name).
//
// Auto-skip when velero.io CRDs are not installed.
type Velero struct {
	// BackupSLA caps how stale the most-recent backup of each
	// schedule can be before we flag. Zero uses defaultBackupSLA.
	BackupSLA time.Duration

	// Now returns the current time; overridable in tests.
	Now func() time.Time
}

const veleroName = "Velero"

const (
	defaultBackupSLA      = 26 * time.Hour
	veleroStuckInProgress = 4 * time.Hour
)

var gvrVeleroBackup = schema.GroupVersionResource{
	Group:    "velero.io",
	Version:  "v1",
	Resource: "backups",
}

// Name satisfies probe.Probe.
func (Velero) Name() string { return veleroName }

// Run satisfies probe.Probe.
func (v Velero) Run(ctx context.Context, src snapshot.Source) Result {
	r := Result{Component: ComponentResult{Component: veleroName}}
	now := v.Now
	if now == nil {
		now = time.Now
	}
	sla := v.BackupSLA
	if sla == 0 {
		sla = defaultBackupSLA
	}

	list, err := src.List(ctx, gvrVeleroBackup, "")
	if err != nil {
		r.Component.Status = "SKIPPED"
		r.Component.Detail = "Velero CRDs not installed (list backups failed)"
		return r
	}
	if list == nil || len(list.Items) == 0 {
		r.Component.Status = "HEALTHY"
		r.Component.Detail = "no Velero Backup resources"
		return r
	}

	// Group by schedule name; keep most-recent per group.
	type latest struct {
		name      string
		ns        string
		phase     string
		startedAt time.Time
		message   string
	}
	bySchedule := map[string]latest{}
	for i := range list.Items {
		b := &list.Items[i]
		name := b.GetName()
		labels := b.GetLabels()
		schedule := labels["velero.io/schedule-name"]
		if schedule == "" {
			schedule = name // standalone backup; treat as its own group
		}
		phase, _, _ := unstructured.NestedString(b.Object, "status", "phase")
		started := b.GetCreationTimestamp().Time
		message, _, _ := unstructured.NestedString(b.Object, "status", "failureReason")

		cur := bySchedule[schedule]
		if cur.startedAt.IsZero() || started.After(cur.startedAt) {
			bySchedule[schedule] = latest{
				name:      name,
				ns:        b.GetNamespace(),
				phase:     phase,
				startedAt: started,
				message:   message,
			}
		}
	}

	var findings []Finding
	t := now()
	for schedule, l := range bySchedule {
		subject := fmt.Sprintf("Backup/%s/%s", l.ns, schedule)
		switch l.phase {
		case "Completed":
			// Stale-SLA check: even completed backups can drift past
			// their schedule (operator paused the Schedule, or the
			// controller is stalled).
			if t.Sub(l.startedAt) > sla {
				findings = append(findings, Finding{
					Component:   subject,
					Severity:    SeverityCritical,
					Message:     fmt.Sprintf("Velero schedule %q latest backup is older than %s (last: %s)", schedule, sla, l.startedAt.UTC().Format(time.RFC3339)),
					Remediation: fmt.Sprintf("kubectl -n %s get schedule %s -o yaml — confirm the schedule is not paused; `velero schedule create %s --schedule='0 1 * * *' ...` to rebuild if necessary.", l.ns, schedule, schedule),
				})
			}
		case "Failed", "PartiallyFailed":
			findings = append(findings, Finding{
				Component:   subject,
				Severity:    SeverityCritical,
				Message:     fmt.Sprintf("Velero schedule %q latest backup phase=%s (failureReason=%s)", schedule, l.phase, l.message),
				Remediation: fmt.Sprintf("kubectl -n %s describe backup %s — review failureReason + .status.errors; `velero backup logs %s` for the full transcript.", l.ns, l.name, l.name),
			})
		case "InProgress":
			if t.Sub(l.startedAt) > veleroStuckInProgress {
				findings = append(findings, Finding{
					Component:   subject,
					Severity:    SeverityWarning,
					Message:     fmt.Sprintf("Velero schedule %q latest backup stuck InProgress for >%s", schedule, veleroStuckInProgress),
					Remediation: fmt.Sprintf("kubectl -n %s describe backup %s — inspect .status.progress; restic plugin issues commonly stall here.", l.ns, l.name),
				})
			}
		case "":
			// Backup CR exists but the controller hasn't observed
			// phase yet — only flag if it's been quiescent too long.
			if t.Sub(l.startedAt) > veleroStuckInProgress {
				findings = append(findings, Finding{
					Component: subject,
					Severity:  SeverityWarning,
					Message:   fmt.Sprintf("Velero schedule %q latest backup has empty phase for >%s", schedule, veleroStuckInProgress),
				})
			}
		default:
			// FailedValidation / Deleting / etc. — warn.
			findings = append(findings, Finding{
				Component: subject,
				Severity:  SeverityWarning,
				Message:   fmt.Sprintf("Velero schedule %q latest backup phase=%s", schedule, l.phase),
			})
		}
	}

	r.Component.Status = rollupComponentStatus(findings)
	r.Component.Detail = fmt.Sprintf("%d Backup(s) across %d schedule(s)", len(list.Items), len(bySchedule))
	r.Findings = findings
	return r
}
