// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SilenceSpec configures a finding suppression. When the watcher
// observes a diagnostic whose fields equal all NON-EMPTY matcher
// fields, AND the Silence has not expired (`spec.until` is in the
// future), the diagnostic is dropped before downstream emission
// (DriftReport / Slack / Alertmanager / ticketing).
//
// Empty matcher fields act as wildcards. So:
//
//	matcher: { source: "StaleErrorPods" }
//	  → silences EVERY StaleErrorPods finding cluster-wide
//
//	matcher: { source: "StaleErrorPods", subject: "Pod/default/legacy" }
//	  → silences only that pod's StaleErrorPods findings
//
//	matcher: { severity: "warning" }
//	  → silences every warning-severity finding regardless of source
//
// Subject matching is EXACT in Phase 1. Glob / regex matching is a
// Phase 2 concern. An entirely empty matcher is rejected by the CRD
// validation (would silence everything — almost certainly a typo).
type SilenceSpec struct {
	// Matcher selects which diagnostics to suppress. At least one
	// field MUST be set.
	Matcher SilenceMatcher `json:"matcher"`

	// Until is the expiry timestamp. Findings are suppressed only
	// while time.Now() < Until. After expiry the Silence becomes a
	// no-op (status.active flips to false) but the CR is NOT
	// auto-deleted — operators can `kubectl delete` or extend the
	// window by editing `spec.until`. Required.
	Until metav1.Time `json:"until"`

	// Reason is a free-text rationale shown in audit logs and
	// surfaced in the future "active silences" diagnose section.
	// Strongly encouraged so on-call engineers can answer "why is
	// this muted?" without git-archaeology.
	// +optional
	Reason string `json:"reason,omitempty"`

	// CreatedBy is who set the silence (free-text — OIDC subject or
	// ServiceAccount path or human name). Audit metadata only.
	// +optional
	CreatedBy string `json:"createdBy,omitempty"`
}

// SilenceMatcher is the per-field filter applied to each diagnostic.
// Non-empty fields must equal the corresponding Diagnostic field for
// the Silence to match. Empty = wildcard.
type SilenceMatcher struct {
	// Source is the analyzer name that produced the finding (e.g.
	// "StaleErrorPods", "EndpointsProbe"). Exact match. Empty =
	// match all sources.
	// +optional
	Source string `json:"source,omitempty"`

	// Subject is the diagnostic Subject (typically
	// "<kind>/<namespace>/<name>"). Exact match. Empty = match any
	// subject under the other matcher constraints.
	// +optional
	Subject string `json:"subject,omitempty"`

	// Severity narrows by severity (`warning`, `critical`).
	// Empty = any severity.
	// +optional
	Severity string `json:"severity,omitempty"`

	// MessagePattern narrows by Message substring. v1.21.0
	// (Phase 2.B.9) — supports class-scoped silences derived from
	// a "Silence class (7d)" click on a Slack proposal. The CHA-com
	// approval-server's /silence-class handler creates Silences
	// with this field populated from policy.InferMessagePattern,
	// e.g. "without digest pin" for SecurityDrift digest-pin
	// findings.
	//
	// Substring match (NOT regex) — chosen for predictability;
	// operators can audit "what messages will this silence catch?"
	// by reading the value. Empty = any message.
	// +optional
	MessagePattern string `json:"messagePattern,omitempty"`
}

// SilenceStatus tracks the runtime view of a silence.
type SilenceStatus struct {
	// Active is the controller's last observation of whether the
	// Silence's window is currently open (i.e. `spec.until` is in the
	// future). Reset to false when the watcher first observes an
	// expired silence. Operators can leave expired silences in place
	// for audit history or delete them.
	// +optional
	Active bool `json:"active,omitempty"`

	// MatchCount is the running total of diagnostics suppressed by
	// this silence during its active window. Helpful for "did this
	// actually mute anything?" review.
	// +optional
	MatchCount int64 `json:"matchCount,omitempty"`

	// LastMatchAt is when the silence last fired. Empty = never
	// matched.
	// +optional
	LastMatchAt *metav1.Time `json:"lastMatchAt,omitempty"`
}

// Silence is the CHA noise-suppression CRD. One CR per active
// suppression. Operators create these to mute known-benign-but-
// unfixable findings for a bounded window. CHA's watch loop filters
// active silences against incoming diagnostics before downstream
// emission, so a known noisy finding doesn't re-page on every probe
// cycle.
//
// Lifecycle: operators create a Silence, the watcher honors it until
// `spec.until`, then the silence becomes a no-op (it does NOT auto-
// delete — operators can extend or remove on their own).
//
// Until is type=string (NOT type=date): kubectl renders date columns
// as age-since, which is negative — shown as `<invalid>` — for the
// future expiry every active Silence has by definition.
//
// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced,shortName=sil
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Source",type=string,JSONPath=`.spec.matcher.source`
// +kubebuilder:printcolumn:name="Subject",type=string,JSONPath=`.spec.matcher.subject`
// +kubebuilder:printcolumn:name="Until",type=string,JSONPath=`.spec.until`
// +kubebuilder:printcolumn:name="Active",type=boolean,JSONPath=`.status.active`
// +kubebuilder:printcolumn:name="Matched",type=integer,JSONPath=`.status.matchCount`
type Silence struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SilenceSpec   `json:"spec,omitempty"`
	Status SilenceStatus `json:"status,omitempty"`
}

// SilenceList contains a list of Silence.
//
// +kubebuilder:object:root=true
type SilenceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Silence `json:"items"`
}
