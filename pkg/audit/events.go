// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

// Package audit provides the audit-trail building blocks that ship in
// the open-source engine:
//
//   - EventsSink — the default ai.AuditSink, writing Kubernetes Events
//     (no extra infrastructure required).
//   - ChainedSink / VerifyChain / VerifyChainWithCheckpoints — the
//     tamper-evident hash-chain primitive: canonical-JSON hashing,
//     prev_hash/entry_hash linking, chain resumption across restarts,
//     and signed-checkpoint tail anchoring. Fully auditable here, in
//     the open source, so the chain format can be verified before (and
//     independently of) any paid component.
//
// Canonical-form contract (chain hashing): entry hashes are sha256 over
// the event's canonical JSON, where the canonical form is exactly Go
// encoding/json.Marshal output — struct fields in declaration order,
// map keys sorted lexicographically, HTML-escaping ON (the bytes <, >,
// and & are encoded as their backslash-u escapes), and timestamps in
// RFC3339Nano. Production chains are already written in this form, so
// it is frozen for cross-version verifiability. External verifiers MUST
// replicate these rules byte-for-byte; see the golden-bytes contract
// test TestCanonicalJSON_FormatContract and the canonicalJSON doc in
// hash_chain.go.
//
// What stays paid: the richer SINKS the chain can wrap — the JSONL
// chained-file sink with rotation, Loki, and OTLP/SIEM — ship in the
// Srenix Enterprise binary and register via the registry without removing the
// defaults.
//
// Adapter path: the chain primitive here operates directly on
// ai.AuditEvent / ai.AuditSink (the same types Srenix Enterprise's sinks use), so
// Srenix Enterprise adapts by importing this package and constructing its
// file-backed chains via NewChainedSinkResuming(inner, resumeHash,
// ChainOptions{...}) — its store reads the resume hash from the last
// persisted entry's "entry_hash" Details field and calls WriteCheckpoint
// on close to anchor the tail.
package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/srenix-ai/agentic-sre/pkg/ai"
)

// EventsSink writes audit events as Kubernetes Events in the
// agentic-sre namespace. This is the default sink — it
// requires no additional infrastructure beyond an in-cluster
// ServiceAccount with `events.create` permission.
//
// Events sink semantics:
//   - One Event per AuditEvent.
//   - Reason = event Type (e.g. "AIProposalCreated", "AIApprovalGranted")
//   - Message = compact summary
//   - InvolvedObject = the diagnostic's subject (best-effort parse)
//   - Annotations carry full structured details under
//     "srenix.ai/audit-details"
//
// Limitations:
//   - kubectl get events is rate-limited by etcd; the sink coalesces
//     duplicates via the standard EventCorrelator path.
//   - Events older than 1 hour are garbage-collected by the apiserver.
//     For long-term audit retention, register a Loki/SIEM sink alongside.
type EventsSink struct {
	client    kubernetes.Interface
	namespace string
}

// NewEventsSink constructs an EventsSink that writes Events into ns.
// Typical ns is the Srenix install namespace.
func NewEventsSink(client kubernetes.Interface, namespace string) *EventsSink {
	return &EventsSink{client: client, namespace: namespace}
}

// Write emits a Kubernetes Event for e. Non-blocking on failure —
// errors are returned but audit emission must never break the watcher
// loop. The caller (watcher orchestration) logs and continues.
func (s *EventsSink) Write(ctx context.Context, e ai.AuditEvent) error {
	if s == nil || s.client == nil {
		return nil
	}

	reason := eventReason(e.Type)
	msg := eventMessage(e)
	details, err := json.Marshal(e.Details)
	if err != nil {
		details = []byte(`{}`)
	}

	involvedKind, involvedNS, involvedName := parseInvolved(e.Subject, s.namespace)

	now := metav1.NewTime(time.Now())
	ev := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "srenix-ai-",
			Namespace:    s.namespace,
			Annotations: map[string]string{
				"srenix.ai/audit-type":           e.Type,
				"srenix.ai/audit-correlation-id": e.CorrelationID,
				"srenix.ai/audit-tier":           string(e.Tier),
				"srenix.ai/audit-actor":          e.Actor,
				"srenix.ai/audit-details":        string(details),
			},
		},
		InvolvedObject: corev1.ObjectReference{
			Kind:       involvedKind,
			Namespace:  involvedNS,
			Name:       involvedName,
			APIVersion: "v1",
		},
		Reason:         reason,
		Message:        msg,
		Type:           eventLevel(e.Type),
		FirstTimestamp: now,
		LastTimestamp:  now,
		Count:          1,
		Source: corev1.EventSource{
			Component: "srenix-ai",
		},
		ReportingController: "srenix.ai/ai",
		ReportingInstance:   e.Actor,
		EventTime:           metav1.NewMicroTime(now.Time),
	}

	_, err = s.client.CoreV1().Events(s.namespace).Create(ctx, ev, metav1.CreateOptions{})
	return err
}

// eventReason converts a dotted audit type into a CamelCase Reason.
//
//	"ai.proposal.created"   -> "AIProposalCreated"
//	"ai.approval.granted"   -> "AIApprovalGranted"
//	"ai.action.applied"     -> "AIActionApplied"
//	"ai.runbook.dual_approval" -> "AIRunbookDualApproval"
func eventReason(t string) string {
	parts := strings.FieldsFunc(t, func(r rune) bool {
		return r == '.' || r == '_' || r == '-'
	})
	var sb strings.Builder
	for _, p := range parts {
		if p == "" {
			continue
		}
		if strings.EqualFold(p, "ai") || strings.EqualFold(p, "llm") {
			sb.WriteString(strings.ToUpper(p))
			continue
		}
		sb.WriteString(strings.ToUpper(p[:1]))
		sb.WriteString(p[1:])
	}
	if sb.Len() == 0 {
		return "AIEvent"
	}
	return sb.String()
}

// eventLevel maps audit-type prefixes to Kubernetes Event Type
// ("Normal" | "Warning"). Failures and rejections are Warning; everything
// else is Normal.
func eventLevel(t string) string {
	switch {
	case strings.Contains(t, "failed"), strings.Contains(t, "rejected"),
		strings.Contains(t, "expired"), strings.Contains(t, "replay"),
		strings.Contains(t, "circuit_break"):
		return corev1.EventTypeWarning
	default:
		return corev1.EventTypeNormal
	}
}

// eventMessage builds a one-line summary for the Event Message field.
// Full details are still in annotations.
func eventMessage(e ai.AuditEvent) string {
	var sb strings.Builder
	if e.CorrelationID != "" {
		fmt.Fprintf(&sb, "[%s] ", e.CorrelationID)
	}
	if e.Tier != "" {
		fmt.Fprintf(&sb, "tier=%s ", e.Tier)
	}
	if e.Actor != "" {
		fmt.Fprintf(&sb, "actor=%s ", e.Actor)
	}
	if e.Subject != "" {
		fmt.Fprintf(&sb, "subject=%s ", e.Subject)
	}
	// Append a small selection of details to the message for at-a-glance
	// triage when reviewing `kubectl get events`. Full details remain in
	// annotations.
	for _, k := range []string{"model", "action_kind", "success", "approver"} {
		if v, ok := e.Details[k]; ok {
			fmt.Fprintf(&sb, "%s=%v ", k, v)
		}
	}
	return strings.TrimSpace(sb.String())
}

// parseInvolved best-effort parses Subject into a Kubernetes
// (kind, namespace, name) triple for the Event InvolvedObject. Falls
// back to a sentinel "AuditScope" object when Subject is missing or
// not in the standard "Kind/ns/name[/...]" shape.
func parseInvolved(subject, defaultNS string) (kind, ns, name string) {
	if subject == "" {
		return "AuditScope", defaultNS, "srenix"
	}
	parts := strings.SplitN(subject, "/", 4)
	if len(parts) < 3 {
		return "AuditScope", defaultNS, "srenix"
	}
	return parts[0], parts[1], parts[2]
}
