// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

// Package audit provides default implementations of the ai.AuditSink
// interface. The OSS engine ships these defaults; the paid CHA-com
// binary can register richer sinks (Loki, OTLP/SIEM) via the registry
// without removing the default.
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

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/ai"
)

// EventsSink writes audit events as Kubernetes Events in the
// cluster-health-autopilot namespace. This is the default sink — it
// requires no additional infrastructure beyond an in-cluster
// ServiceAccount with `events.create` permission.
//
// Events sink semantics:
//   - One Event per AuditEvent.
//   - Reason = event Type (e.g. "AIProposalCreated", "AIApprovalGranted")
//   - Message = compact summary
//   - InvolvedObject = the diagnostic's subject (best-effort parse)
//   - Annotations carry full structured details under
//     "cha.bionicaisolutions.com/audit-details"
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
// Typical ns is the CHA install namespace.
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
			GenerateName: "cha-ai-",
			Namespace:    s.namespace,
			Annotations: map[string]string{
				"cha.bionicaisolutions.com/audit-type":           e.Type,
				"cha.bionicaisolutions.com/audit-correlation-id": e.CorrelationID,
				"cha.bionicaisolutions.com/audit-tier":           string(e.Tier),
				"cha.bionicaisolutions.com/audit-actor":          e.Actor,
				"cha.bionicaisolutions.com/audit-details":        string(details),
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
			Component: "cha-ai",
		},
		ReportingController: "cha.bionicaisolutions.com/ai",
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
		return "AuditScope", defaultNS, "cha"
	}
	parts := strings.SplitN(subject, "/", 4)
	if len(parts) < 3 {
		return "AuditScope", defaultNS, "cha"
	}
	return parts[0], parts[1], parts[2]
}
