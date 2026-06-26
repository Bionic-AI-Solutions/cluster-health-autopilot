// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

// Package resolution persists the outcome of every remediation attempt as
// an append-only ResolutionRecord CR. It is the durable system-of-record
// that the RAG memory layer (see Srenix Enterprise docs/design/2026-05-ai-
// remediation-rag-plan.md) embeds and retrieves: "this fix was applied to
// this kind of finding and it worked / didn't". Both the OSS deterministic
// fixers and the commercial AI proposers write through this recorder.
package resolution

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/srenix-ai/agentic-sre/internal/snapshot"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Verdict is the post-apply verification result for a remediation.
type Verdict string

// Post-apply verification verdicts for a remediation.
const (
	VerdictCleared       Verdict = "cleared"        // re-probe confirmed the finding is gone
	VerdictStillPresent  Verdict = "still-present"  // finding persisted after the fix
	VerdictNotVerifiable Verdict = "not-verifiable" // no probe could confirm (best-effort)
)

// Delivery records how the remediation reached execution.
type Delivery string

// Delivery channels by which a remediation reached execution.
const (
	DeliveryHumanApproved Delivery = "approved-by-human" // signed click-to-fix, human one-click
	DeliveryAutoApplied   Delivery = "auto-applied"      // policy-bounded autonomy (P3)
	DeliveryDeterministic Delivery = "deterministic"     // OSS rule-based fixer
	DeliveryRejected      Delivery = "rejected"          // proposed, declined / never applied
)

// Record is the captured outcome of one remediation. Fields mirror the
// ResolutionRecord CRD spec; the RAG layer embeds DiagnosticDigest and
// filters/weights on Verdict.
type Record struct {
	Fingerprint      string   // stable hash of the originating finding (join key to DriftReport)
	Cluster          string   // cluster name (multi-cluster RAG scoping)
	Namespace        string   // target namespace
	Source           string   // analyzer/probe that produced the finding
	SubjectKind      string   // e.g. Pod, HorizontalPodAutoscaler, Ingress
	DiagnosticDigest string   // redacted one-line summary of what was wrong (the embed text)
	ActionKind       string   // remediation action taken (e.g. DeletePod, PatchDeployment)
	Target           string   // object the action operated on (kind/ns/name)
	Rationale        string   // why this fix (proposer/fixer rationale)
	Rollback         string   // rollback description, if any
	Delivery         Delivery // how it reached execution
	Applied          bool     // whether the action was actually executed
	Verified         Verdict  // post-apply verification result
	HumanEdits       string   // diff vs the original proposal, if a human edited it
}

// Fingerprint derives the stable join key from a finding's source+subject,
// matching the DriftReport dedup identity. Exported so callers can compute
// it consistently before constructing a Record.
func Fingerprint(source, subject string) string {
	sum := sha256.Sum256([]byte(source + "\x00" + subject))
	return hex.EncodeToString(sum[:])[:16]
}

// Recorder writes ResolutionRecords through a snapshot.Mutator. Append-only:
// each verified outcome is a new CR (never upserted), so the memory layer
// sees the full history including repeated/regressed fixes.
type Recorder struct {
	// Now is injected for deterministic tests; defaults to time.Now.
	Now func() time.Time
}

// Record persists r as a new ResolutionRecord CR. A nil Mutator (dry-run /
// snapshot mode) is a no-op so the recorder is safe to call unconditionally.
// The CR name is <fingerprint>-<unix-nanos> so concurrent outcomes for the
// same finding never collide.
func (rec Recorder) Record(ctx context.Context, mut snapshot.Mutator, r Record) error {
	if mut == nil {
		return nil
	}
	now := time.Now
	if rec.Now != nil {
		now = rec.Now
	}
	ts := now().UTC()
	if r.Fingerprint == "" {
		return fmt.Errorf("resolution: Record requires a Fingerprint")
	}
	name := fmt.Sprintf("rr-%s-%d", r.Fingerprint, ts.UnixNano())

	cr := unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "srenix.ai/v1alpha1",
			"kind":       "ResolutionRecord",
			"metadata": map[string]any{
				"name": name,
				"labels": map[string]any{
					"srenix.ai/fingerprint": r.Fingerprint,
					"srenix.ai/verified":    string(r.Verified),
					"srenix.ai/delivery":    string(r.Delivery),
				},
			},
			"spec": map[string]any{
				"fingerprint":      r.Fingerprint,
				"cluster":          r.Cluster,
				"namespace":        r.Namespace,
				"source":           r.Source,
				"subjectKind":      r.SubjectKind,
				"diagnosticDigest": truncate(r.DiagnosticDigest, 2048),
				"proposal": map[string]any{
					"actionKind": r.ActionKind,
					"target":     r.Target,
					"rationale":  truncate(r.Rationale, 1024),
					"rollback":   truncate(r.Rollback, 1024),
				},
				"delivery":   string(r.Delivery),
				"applied":    r.Applied,
				"verified":   string(r.Verified),
				"humanEdits": truncate(r.HumanEdits, 2048),
				"recordedAt": ts.Format(time.RFC3339),
			},
		},
	}
	if err := mut.Create(ctx, snapshot.GVRResolutionRecord, "", &cr); err != nil {
		return fmt.Errorf("create ResolutionRecord %s: %w", name, err)
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
