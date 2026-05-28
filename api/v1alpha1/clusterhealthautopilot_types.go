// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterHealthAutopilotSpec defines the desired state of a CHA install.
//
// Field shape mirrors the existing Helm `values.yaml`, but as a CRD so
// operators can manage multiple CHA installs declaratively (one CR per
// install) and ArgoCD / Flux can sync them without re-templating Helm.
//
// Phase 1 of the operator port covers WatcherSpec + DiagnoseSpec —
// the two most-deployed CHA modes. RemediateSpec ships alongside but
// can be left disabled. AlertingSpec captures the Slack + Alertmanager
// wiring; TicketingSpec and AISpec ship as opaque pass-throughs in
// Phase 1 (the controller mounts them verbatim onto the watcher
// Deployment) and gain typed validation in Phase 2.
type ClusterHealthAutopilotSpec struct {
	// Image is the container image the watcher + cronjobs run.
	// Required. Operators typically pin a semver tag; the controller
	// does not auto-bump.
	Image ImageSpec `json:"image"`

	// Watcher is the always-on Deployment that streams cluster
	// events through the probe/analyzer/fix loop. Disabled by
	// default — most installs prefer the diagnose CronJob unless
	// they want sub-second triggering.
	// +optional
	Watcher *WatcherSpec `json:"watcher,omitempty"`

	// Diagnose is the periodic read-only sweep. Default schedule
	// 0 9 * * * (daily 09:00 UTC). Always enabled in a typical
	// install — the daily Slack #healthinfo digest depends on it.
	// +optional
	Diagnose *DiagnoseSpec `json:"diagnose,omitempty"`

	// Remediate is the periodic auto-fixer sweep. Disabled by
	// default; operators flip Enabled=true once the team trusts
	// the fixer behavior.
	// +optional
	Remediate *RemediateSpec `json:"remediate,omitempty"`

	// Alerting controls Slack + Alertmanager fan-out. Empty
	// blocks mean "no alerts" — useful for dry-running.
	// +optional
	Alerting *AlertingSpec `json:"alerting,omitempty"`

	// ServiceAccountName overrides the controller-managed SA name.
	// When empty the controller creates and owns <cr-name>-sa.
	//
	// IMPORTANT (RBAC): the operator does not yet provision a reader
	// ClusterRoleBinding for the SA it creates, so the default
	// <cr-name>-sa has NO probe RBAC and the watcher would get
	// `forbidden` on every List. To run an operator-managed watcher
	// today, set this to an existing reader-bound SA (e.g. the chart's
	// `<release>-cluster-health-autopilot` SA). When set, the operator
	// references but does NOT create or own that SA. Operator-owned
	// RBAC is tracked for Phase 1c.
	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty"`
}

// ImageSpec names the container image for all CHA workloads.
type ImageSpec struct {
	// Repository is the OCI image repository (no tag).
	Repository string `json:"repository"`

	// Tag pins the exact image tag. Required — the controller
	// refuses an empty tag rather than silently picking `latest`.
	Tag string `json:"tag"`

	// PullPolicy follows the K8s PullPolicy enum. Defaults to
	// IfNotPresent for semver tags and Always for mutable tags.
	// +optional
	PullPolicy string `json:"pullPolicy,omitempty"`

	// PullSecrets is an optional list of image-pull-secret names
	// in the install namespace.
	// +optional
	PullSecrets []string `json:"pullSecrets,omitempty"`
}

// WatcherSpec configures the watcher Deployment.
type WatcherSpec struct {
	// Enabled is the master switch for the watcher Deployment.
	// When false the controller deletes any existing watcher pod.
	Enabled bool `json:"enabled"`

	// Replicas is the desired replica count. Default 1. Operators
	// typically keep this at 1 because of lease-based leader
	// election — additional replicas stand by.
	// +optional
	Replicas int32 `json:"replicas,omitempty"`

	// Debounce is the duration the watcher coalesces event bursts
	// before triggering a probe pass. Defaults to 10s.
	// +optional
	Debounce string `json:"debounce,omitempty"`

	// ResyncPeriod is the fallback periodic resync interval when
	// no events fire. Defaults to 10m.
	// +optional
	ResyncPeriod string `json:"resyncPeriod,omitempty"`
}

// DiagnoseSpec configures the diagnose CronJob.
type DiagnoseSpec struct {
	// Enabled is the master switch.
	Enabled bool `json:"enabled"`

	// Schedule is a Kubernetes-compatible cron expression.
	// Default "0 9 * * *".
	Schedule string `json:"schedule,omitempty"`

	// BackoffLimit caps Job-level retries. Default 1.
	// +optional
	BackoffLimit int32 `json:"backoffLimit,omitempty"`

	// ActiveDeadlineSeconds caps the Job's wall-clock runtime.
	// Default 120.
	// +optional
	ActiveDeadlineSeconds int64 `json:"activeDeadlineSeconds,omitempty"`
}

// RemediateSpec configures the remediate CronJob.
type RemediateSpec struct {
	// Enabled is the master switch. Default false — teams turn this
	// on after they trust the fixers.
	Enabled bool `json:"enabled"`

	// Schedule is a Kubernetes-compatible cron expression.
	// Default "*/30 * * * *".
	Schedule string `json:"schedule,omitempty"`

	// DryRun, when true, makes fixers log "Refused" without
	// mutating cluster state.
	// +optional
	DryRun bool `json:"dryRun,omitempty"`
}

// AlertingSpec configures the watcher's outbound notification paths.
type AlertingSpec struct {
	// Slack configuration (three channels: alerts / critical /
	// healthinfo). Each entry references a Secret in the install
	// namespace holding the webhook URL.
	// +optional
	Slack *SlackSpec `json:"slack,omitempty"`

	// Alertmanager is the in-cluster Alertmanager URL the watcher
	// posts to. Empty means "no Alertmanager integration."
	// +optional
	Alertmanager *AlertmanagerSpec `json:"alertmanager,omitempty"`
}

// SlackSpec is the three-channel Slack routing block.
type SlackSpec struct {
	// Alerts → #ceph-alerts: event-driven, CHA auto-fixed.
	// +optional
	Alerts *SlackChannelSpec `json:"alerts,omitempty"`
	// Critical → #ceph-critical: event-driven, human action required.
	// +optional
	Critical *SlackChannelSpec `json:"critical,omitempty"`
	// HealthInfo → #healthinfo: daily digest from the diagnose CronJob.
	// +optional
	HealthInfo *SlackChannelSpec `json:"healthInfo,omitempty"`
}

// SlackChannelSpec references the webhook Secret for one channel.
type SlackChannelSpec struct {
	// SecretName is the Kubernetes Secret in the install namespace
	// holding the Slack webhook URL.
	SecretName string `json:"secretName"`

	// SecretKey is the key inside the Secret. Defaults to "WEBHOOK_URL".
	// +optional
	SecretKey string `json:"secretKey,omitempty"`
}

// AlertmanagerSpec configures Alertmanager fan-out.
type AlertmanagerSpec struct {
	// URL is the in-cluster Alertmanager API URL.
	URL string `json:"url"`

	// ClusterName is stamped as the `cluster` label on every alert.
	// Required.
	ClusterName string `json:"clusterName"`
}

// ClusterHealthAutopilotStatus tracks the operator's last-observed
// reconciliation outcome.
type ClusterHealthAutopilotStatus struct {
	// ObservedGeneration is the metadata.generation the controller
	// last reconciled. When status.observedGeneration < metadata.generation
	// the operator is still catching up to the latest spec.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions track high-level operational state. Standard names:
	//   - "Ready"           — all subresources reconciled.
	//   - "WatcherRunning"  — watcher Deployment available replicas > 0.
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

// Standard condition types reported by the controller. Pinned as
// constants so tests and metrics dashboards reference one source.
const (
	// ConditionReady reflects whether the entire CR is reconciled.
	ConditionReady = "Ready"

	// ConditionWatcherRunning is set True when the managed watcher
	// Deployment has at least one available replica.
	ConditionWatcherRunning = "WatcherRunning"
)

// ClusterHealthAutopilot is the Schema for the clusterhealthautopilots
// API. One CR per CHA install — replaces the Helm values.yaml that
// drove pre-operator installs.
//
// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced,shortName=cha
// +kubebuilder:subresource:status
type ClusterHealthAutopilot struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterHealthAutopilotSpec   `json:"spec,omitempty"`
	Status ClusterHealthAutopilotStatus `json:"status,omitempty"`
}

// ClusterHealthAutopilotList contains a list of ClusterHealthAutopilot.
//
// +kubebuilder:object:root=true
type ClusterHealthAutopilotList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterHealthAutopilot `json:"items"`
}
