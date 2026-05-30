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
// wiring.
//
// AISpec lands the AI-companion (CHA-com) configuration shape at the
// CR level — schema only. The controller does not yet consume these
// fields in Phase 1; the AI watcher and approval-server continue to
// be configured via the chart's `ai.*` and `approval.*` helm values
// until the Phase 2 reconciler picks them up. Ship the schema now so
// operator-managed manifests are forward-compatible.
//
// TicketingSpec is intentionally not yet in v1alpha1 — its shape is
// provider-specific (OpenProject / Jira / ServiceNow) and the typed
// schema lands when ticketing also moves off helm values in Phase 2.
// See `docs/design/2026-05-v1.9-operator-phase-1c.md` for the roadmap.
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

	// AI configures the CHA-com paid-tier AI watcher (aiwatch) +
	// approval-server. Schema only in Phase 1 — the controller does
	// NOT consume these fields yet; configure AI via the chart's
	// `ai.*` helm values today. The schema lands here so an operator-
	// managed CR can declare AI config forward-compatibly. Phase 2
	// wires the reconciler.
	// +optional
	AI *AISpec `json:"ai,omitempty"`

	// ServiceAccountName overrides the controller-managed SA name.
	// When empty the controller creates `<cr-name>-sa` AND provisions a
	// cluster-wide reader ClusterRoleBinding for it (Phase 1c — see
	// `BuildReaderClusterRoleBinding`). So an operator-managed install
	// works greenfield without any chart-installed RBAC.
	//
	// When set, the operator binds the named (BYO) ServiceAccount to
	// the shared reader ClusterRole but does NOT create or own the SA
	// itself — the SA lifecycle belongs to whoever defined it.
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
	// True iff all subresource reconciles succeeded AND every other
	// individual condition is True (currently WatcherRunning +
	// ReaderRBACReady — the AND grows as Phase 2 adds verifiers).
	ConditionReady = "Ready"

	// ConditionWatcherRunning is set True when the managed watcher
	// Deployment has at least one available replica.
	ConditionWatcherRunning = "WatcherRunning"

	// ConditionReaderRBACReady is set True when the operator has
	// provisioned the shared reader ClusterRole AND the per-CR
	// ClusterRoleBinding that grants the watcher SA cluster-wide
	// read on the probe surface. False when either RBAC object is
	// missing — the watcher would get `forbidden` on every List
	// without them.
	ConditionReaderRBACReady = "ReaderRBACReady"
)

// FinalizerOperatorRBAC tags a ClusterHealthAutopilot CR for the RBAC
// cleanup pass. Cluster-scoped resources (the per-CR ClusterRoleBinding
// the operator provisions) can NOT carry an ownerRef back to a
// namespaced CR — Kubernetes' garbage collector drops the ref and
// orphans the child. The finalizer gives the controller a chance to
// delete the binding before the CR is GC'd, so cluster-scoped state
// stays consistent with CR lifecycle.
const FinalizerOperatorRBAC = "cha.bionicaisolutions.com/operator-rbac"

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

// AISpec is the typed schema for the CHA-com paid-tier AI watcher
// (aiwatch + approval-server + optional RAG store). Schema-only in
// Phase 1 — the controller does NOT consume these fields yet. Adding
// the schema now lets operator-managed installs declare AI config on
// the CR ahead of the Phase 2 reconciler wiring; today's installs
// continue to configure AI via the chart's `ai.*` helm values.
type AISpec struct {
	// Enabled is the master switch. Default false; matches helm
	// `ai.enabled`.
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// Tier selects the AI capability level.
	// +kubebuilder:validation:Enum=off;t0;t1;t2;t3
	// +optional
	Tier string `json:"tier,omitempty"`

	// Endpoint is the OpenAI-compatible base URL (e.g.
	// "https://mcp.baisoln.com/gpu-ai/v1"). Required when Enabled.
	// +optional
	Endpoint string `json:"endpoint,omitempty"`

	// Model is the default model identifier passed to the LLM
	// (e.g. "qwen3.6-35b-a3b-fp8"). Required when Enabled.
	// +optional
	Model string `json:"model,omitempty"`

	// Interval is the AI poll cadence; tiers fire only on NEW
	// diagnostics each cycle. Empty → chart default ("60s").
	// +optional
	Interval string `json:"interval,omitempty"`

	// AllowSaaS opts in to non-cluster endpoints (api.openai.com /
	// api.anthropic.com / etc). Off by default; in-cluster vLLM is
	// the BYOM recommendation.
	// +optional
	AllowSaaS bool `json:"allowSaas,omitempty"`

	// LLMFixerMatcher (t1+): use the LLM-classified fixer matcher
	// instead of the keyword heuristic. Falls back to keyword on
	// LLM error / timeout.
	// +optional
	LLMFixerMatcher bool `json:"llmFixerMatcher,omitempty"`

	// AuditLog destination for AI events: empty (off) | "-" (stdout)
	// | file path. Required for t1+ compliance posture.
	// +optional
	AuditLog string `json:"auditLog,omitempty"`

	// ApprovalServerURL (t1+) is the base URL of the approval-server
	// — when set, T1/T2 proposals emit a signed click-to-fix link.
	// +optional
	ApprovalServerURL string `json:"approvalServerUrl,omitempty"`

	// Image is the CHA-com binary image (the aiwatch + approval-server
	// containers). Empty → chart default ("docker4zerocool/cha-com"
	// at `v<AppVersion>`).
	// +optional
	Image *ImageSpec `json:"image,omitempty"`

	// APIKey references the K8s Secret holding the LLM bearer token.
	// +optional
	APIKey *AIAPIKeySpec `json:"apiKey,omitempty"`

	// T3 scopes the LLM runbook proposer's blast radius (Vault path
	// prefixes it may reference). Every t3 proposal is still
	// recommendation-only behind dual human approval.
	// +optional
	T3 *AIT3Spec `json:"t3,omitempty"`

	// Memory is the dedicated RAG (Qdrant + embeddings) config. When
	// enabled, the chart stands up an in-namespace vector store and
	// the aiwatch grounds proposals in prior verified resolutions.
	// +optional
	Memory *AIMemorySpec `json:"memory,omitempty"`
}

// AIAPIKeySpec references the LLM-bearer-token Secret.
type AIAPIKeySpec struct {
	// SecretName is the K8s Secret in the install namespace (ESO-
	// managed in production).
	// +optional
	SecretName string `json:"secretName,omitempty"`

	// SecretKey is the key inside the Secret. Default "API_KEY".
	// +optional
	SecretKey string `json:"secretKey,omitempty"`

	// EnvName is the env var the binary reads. Default "AI_API_KEY".
	// +optional
	EnvName string `json:"envName,omitempty"`

	// Header is the HTTP header for the key. Empty → "Authorization:
	// Bearer <key>". Set "X-API-Key" for Kong key-auth gateways.
	// +optional
	Header string `json:"header,omitempty"`
}

// AIT3Spec is the t3 Vault break-glass configuration.
type AIT3Spec struct {
	// VaultAllowedPrefixes lists the Vault path prefixes the t3
	// runbook proposer may target. Empty → no Vault paths permitted.
	// +optional
	VaultAllowedPrefixes []string `json:"vaultAllowedPrefixes,omitempty"`
}

// AIMemorySpec configures the dedicated RAG vector store + embeddings.
type AIMemorySpec struct {
	// Enabled stands up the in-namespace Qdrant StatefulSet. Off by
	// default; independent of AI.Enabled so it can be rolled
	// separately.
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// Image is the Qdrant container image. Empty → chart default
	// (qdrant/qdrant:v1.12.4).
	// +optional
	Image *ImageSpec `json:"image,omitempty"`

	// Storage configures the Qdrant PVC.
	// +optional
	Storage *AIMemoryStorageSpec `json:"storage,omitempty"`

	// Embeddings configures how aiwatch vectorizes records.
	// +optional
	Embeddings *AIEmbeddingsSpec `json:"embeddings,omitempty"`

	// StoreURL is the Qdrant service URL the aiwatch connects to.
	// Empty → chart-default in-namespace service.
	// +optional
	StoreURL string `json:"storeUrl,omitempty"`

	// TopK is the number of prior resolutions retrieved per finding.
	// 0 → chart default (5).
	// +optional
	TopK int32 `json:"topK,omitempty"`
}

// AIMemoryStorageSpec is the Qdrant PVC shape.
type AIMemoryStorageSpec struct {
	// Size is the PVC size, e.g. "5Gi". Empty → "5Gi".
	// +optional
	Size string `json:"size,omitempty"`

	// ClassName is the storageClassName. Empty → default class.
	// +optional
	ClassName string `json:"className,omitempty"`
}

// AIEmbeddingsSpec configures the embeddings call.
type AIEmbeddingsSpec struct {
	// Endpoint is the OpenAI-compatible /embeddings base URL.
	// Empty → AI.Endpoint.
	// +optional
	Endpoint string `json:"endpoint,omitempty"`

	// Model is the embedding model identifier (e.g.
	// "qwen3-embedding-0.6b"). Required when AIMemorySpec.Enabled.
	// +optional
	Model string `json:"model,omitempty"`
}
