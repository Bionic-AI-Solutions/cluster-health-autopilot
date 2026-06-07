// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package report

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/diagnose"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/fix"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/probe"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
)

// DriftReportEntry is the abstract input the writer consumes — assembled
// from probe findings, diagnose diagnostics, and fixer actions/skips.
type DriftReportEntry struct {
	Subject       string // stable identity for dedup across ticks
	Severity      string // info|warning|critical
	Source        string // probe name, analyzer name, fixer name
	Category      string // probe|analyzer|fixer-action|fixer-skipped
	Message       string
	Remediation   string
	Investigation string // Layer-2 investigator summary (empty when none)
}

// AssembleEntries flattens probe Results + Diagnostics + fix Results into
// the canonical entry list. Pure — no I/O.
func AssembleEntries(
	results []probe.Result,
	diagnostics []diagnose.Diagnostic,
	fixResults []fix.Result,
) []DriftReportEntry {
	var out []DriftReportEntry

	for _, r := range results {
		for _, f := range r.Findings {
			sev := "info"
			switch f.Severity {
			case probe.SeverityCritical:
				sev = "critical"
			case probe.SeverityWarning:
				sev = "warning"
			}
			out = append(out, DriftReportEntry{
				Subject:       "Probe/" + r.Component.Component + "/" + f.Component,
				Severity:      sev,
				Source:        r.Component.Component,
				Category:      "probe",
				Message:       f.Message,
				Remediation:   f.Remediation,
				Investigation: f.Investigation,
			})
		}
	}
	for _, d := range diagnostics {
		// Severity from the diagnostic, falling back to "warning" for
		// backwards compat with analyzers that don't set it.
		sev := d.Severity
		if sev == "" {
			sev = "warning"
		}
		// Source: prefer the analyzer's own ID (`SecurityDrift`,
		// `CapacityDrift`, …) so the DriftReport's `spec.source` is
		// useful for filtering. Fall back to the legacy literal
		// "analyzer" string only when the analyzer hasn't set it.
		src := d.Source
		if src == "" {
			src = "analyzer"
		}
		out = append(out, DriftReportEntry{
			Subject:       d.Subject,
			Severity:      sev,
			Source:        src,
			Category:      "analyzer",
			Message:       d.Message,
			Remediation:   d.Remediation,
			Investigation: d.Investigation,
		})
	}
	for _, fr := range fixResults {
		for _, a := range fr.Actions {
			out = append(out, DriftReportEntry{
				Subject:  "FixerAction/" + fr.Fixer + "/" + a.Object,
				Severity: "info",
				Source:   fr.Fixer,
				Category: "fixer-action",
				Message:  a.Description + " — " + a.Object,
			})
		}
	}
	return out
}

// Reconcile upserts one CR per entry and deletes CRs whose subject is not
// in the current entry set.
//
// runID identifies this cha invocation; it gets stamped into status.runID
// so an operator can tell which cron tick last observed each report.
//
// keep: returns ALL existing CRs in the cluster. Any CR whose subject is
// missing from `entries` is deleted (the drift is gone — celebrate by
// removing the CR).
func Reconcile(
	ctx context.Context,
	src snapshot.Source,
	mut snapshot.Mutator,
	entries []DriftReportEntry,
	runID string,
) (created, updated, deleted int, err error) {
	if mut == nil {
		return 0, 0, 0, fmt.Errorf("DriftReport reconcile requires a Mutator (live mode)")
	}

	wantBySubject := make(map[string]DriftReportEntry, len(entries))
	for _, e := range entries {
		wantBySubject[e.Subject] = e
	}

	// List existing DriftReports cluster-wide. Tolerate the CRD being
	// absent (returns empty list) so the writer is a no-op until the
	// CRD lands or if the operator disabled it.
	existing, listErr := src.List(ctx, snapshot.GVRDriftReport, "")
	if listErr != nil {
		return 0, 0, 0, fmt.Errorf("list driftreports: %w", listErr)
	}
	existingByName := make(map[string]unstructured.Unstructured, len(existing.Items))
	subjectToName := make(map[string]string, len(existing.Items))
	for _, cr := range existing.Items {
		spec, _, _ := unstructured.NestedMap(cr.Object, "spec")
		subj, _ := spec["subject"].(string)
		if subj == "" {
			continue
		}
		existingByName[cr.GetName()] = cr
		subjectToName[subj] = cr.GetName()
	}

	now := time.Now().UTC().Format(time.RFC3339)

	// Upsert: create CRs for new subjects, patch status for known ones.
	for subject, entry := range wantBySubject {
		crName := subjectToName[subject]
		if crName == "" {
			crName = nameForSubject(subject)
		}

		if _, exists := existingByName[crName]; exists {
			obj := existingByName[crName]
			oldStatus, _, _ := unstructured.NestedMap(obj.Object, "status")
			oldCount, _ := oldStatus["observationCount"].(int64)
			if oldCount == 0 {
				if v, ok := oldStatus["observationCount"].(float64); ok {
					oldCount = int64(v)
				}
			}
			newCount := oldCount + 1

			// Two patches because the CRD declares subresources.status:{}.
			// Patches sent to the main resource endpoint silently drop any
			// status fields in the request body; status changes must go
			// through /status.
			//
			// Spec patch refreshes severity/message/remediation/investigation
			// on every cycle so the CR reflects current state (e.g. flake
			// suppression escalating warning→critical, or an investigation
			// summary attaching after the streak counter trips). Subject is
			// the dedup key and never changes.
			specPatch := []byte(fmt.Sprintf(
				`{"spec":{"severity":%q,"message":%q,"remediation":%q,"investigation":%q}}`,
				entry.Severity,
				truncateAt(entry.Message, 4096),
				truncateAt(entry.Remediation, 1024),
				truncateAt(entry.Investigation, 1024),
			))
			if pErr := mut.Patch(ctx, snapshot.GVRDriftReport, "", crName, types.MergePatchType, specPatch); pErr != nil {
				err = pErr
				continue
			}
			// Status patch: lastObserved + observationCount + runID. If this
			// fails the spec is still current; we record the error but still
			// count the CR as updated (matches today's accounting, where the
			// status portion was always silently dropped).
			statusPatch := []byte(fmt.Sprintf(
				`{"status":{"lastObserved":%q,"observationCount":%d,"runID":%q}}`,
				now, newCount, runID,
			))
			if pErr := mut.PatchStatus(ctx, snapshot.GVRDriftReport, "", crName, types.MergePatchType, statusPatch); pErr != nil {
				err = pErr
			}
			updated++
			delete(existingByName, crName)
			continue
		}

		// Create.
		cr := unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "cha.bionicaisolutions.com/v1alpha1",
				"kind":       "DriftReport",
				"metadata": map[string]any{
					"name": crName,
					"labels": map[string]any{
						"cha.bionicaisolutions.com/category": entry.Category,
						"cha.bionicaisolutions.com/severity": entry.Severity,
						"cha.bionicaisolutions.com/source":   sanitizeLabel(entry.Source),
					},
				},
				"spec": map[string]any{
					"subject":       entry.Subject,
					"severity":      entry.Severity,
					"source":        entry.Source,
					"category":      entry.Category,
					"message":       truncateAt(entry.Message, 4096),
					"remediation":   truncateAt(entry.Remediation, 1024),
					"investigation": truncateAt(entry.Investigation, 1024),
				},
			},
		}
		if cErr := mut.Create(ctx, snapshot.GVRDriftReport, "", &cr); cErr != nil {
			err = cErr
			continue
		}
		// Stamp first/last observed via /status. The CRD declares
		// subresources.status:{} so this MUST hit the status subresource —
		// a plain Patch with a status body would be silently dropped.
		patch := []byte(fmt.Sprintf(
			`{"status":{"firstObserved":%q,"lastObserved":%q,"observationCount":1,"runID":%q}}`,
			now, now, runID,
		))
		if pErr := mut.PatchStatus(ctx, snapshot.GVRDriftReport, "", crName, types.MergePatchType, patch); pErr != nil {
			err = pErr
		}
		created++
	}

	// Delete CRs whose subject is no longer in the entry set.
	for crName := range existingByName {
		if dErr := mut.Delete(ctx, snapshot.GVRDriftReport, "", crName); dErr != nil {
			err = dErr
			continue
		}
		deleted++
	}
	return created, updated, deleted, err
}

// nameForSubject hashes the subject to a DNS-1123 valid CR name. Subjects
// like "Secret/mcp/mcp-openproject-secrets/openproject-url" contain
// slashes that aren't valid in resource names, and may be longer than
// the 253-char limit.
func nameForSubject(subject string) string {
	h := sha256.Sum256([]byte(subject))
	return "drift-" + hex.EncodeToString(h[:8])
}

// sanitizeLabel converts arbitrary strings to a label-safe value (lowercase,
// alphanumerics + dashes, ≤63 chars).
func sanitizeLabel(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteRune('-')
		}
	}
	out := b.String()
	out = strings.Trim(out, "-")
	if len(out) > 63 {
		out = out[:63]
		out = strings.TrimRight(out, "-")
	}
	if out == "" {
		out = "unknown"
	}
	return out
}

func truncateAt(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// Compile-time check that we use metav1.ObjectMeta-shaped naming.
var _ = metav1.ObjectMeta{}
