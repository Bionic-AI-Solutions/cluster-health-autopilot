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
	// optional in-namespace Qdrant. Phase 2 / 2b consume these fields.
	// +optional
	AI *AISpec `json:"ai,omitempty"`

	// Approval configures the CHA-com approval-server Deployment +
	// Service + RBAC + signing-key Secret + optional Ingress.
	//
	// Schema-only in Phase 2c-A (this PR) — the controller does NOT
	// reconcile approval-server objects yet; today's chart users
	// continue to drive the install via the chart's `approval.*` helm
	// values. The schema lands ahead of the reconciler so operator-
	// managed CRs can declare approval-server config forward-compatibly
	// (matches how `AISpec` landed in #107 before Phase 2 picked it up).
	//
	// `spec.ai.approvalServerUrl` and `spec.approval` are independent:
	// the former is the URL the aiwatch points at (could be an
	// out-of-cluster approval-server); the latter is the spec for the
	// in-namespace approval-server the operator manages. Production
	// installs typically set both — `approvalServerUrl =
	// http://<cr>-approval-server.<ns>.svc:8443` when the operator runs
	// the server alongside.
	// +optional
	Approval *ApprovalSpec `json:"approval,omitempty"`

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

	// ConditionAIWatchRunning reflects the CHA-com aiwatch Deployment
	// state when `spec.ai.enabled` is true. False/Disabled when AI is
	// off (the field is still set so dashboards can show "AI disabled"
	// without inferring from a missing condition). True only when the
	// observed Deployment has at least one available replica.
	ConditionAIWatchRunning = "AIWatchRunning"

	// ConditionMemoryStoreReady reflects the in-namespace Qdrant
	// StatefulSet + Service that back the RAG memory loop when
	// `spec.ai.memory.enabled` is true. False/Disabled when memory is
	// off. True only when the StatefulSet reports readyReplicas > 0
	// AND the Service exists. Independent of AIWatchRunning — the
	// store can be Ready before aiwatch even starts, and the
	// aiwatch can run without memory.
	ConditionMemoryStoreReady = "MemoryStoreReady"

	// ConditionApprovalServerReady reflects the in-namespace
	// approval-server Deployment + Service + signing-key Secret when
	// `spec.approval.enabled` is true. False/Disabled when approval
	// is off. True only when the Deployment has availableReplicas > 0,
	// the Service exists, AND the signing-key Secret has a non-empty
	// private key.
	ConditionApprovalServerReady = "ApprovalServerReady"
)

// FinalizerOperatorRBAC tags a ClusterHealthAutopilot CR for the RBAC
// cleanup pass. Cluster-scoped resources (the per-CR ClusterRoleBinding
// the operator provisions) can NOT carry an ownerRef back to a
// namespaced CR — Kubernetes' garbage collector drops the ref and
// orphans the child. The finalizer gives the controller a chance to
// delete the binding before the CR is GC'd, so cluster-scoped state
// stays consistent with CR lifecycle.
//
// Covers BOTH the Phase-1c reader binding AND the Phase-2c-B
// approval-server fixer binding. Single finalizer is enough — the
// finalize handler iterates every managed cluster-scoped binding.
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
// (aiwatch + approval-server + optional RAG store).
//
// Phase 2 / 2b (shipped) — the controller reconciles `spec.ai` into:
//   - an `<cr>-aiwatch` Deployment that mirrors the chart's
//     `aiwatch-deployment.yaml`, including the t3 + memory CLI flags.
//   - an `<cr>-rag` Qdrant StatefulSet + Service when
//     `spec.ai.memory.enabled=true` (mirrors the chart's
//     `rag-qdrant-*.yaml`). The aiwatch's default `--memory-store-url`
//     resolves to this Service.
//
// Still NOT operator-managed (chart-only for now): the approval-server
// Deployment + Ed25519 signing-key Secret for
// `spec.ai.approvalServerUrl`.
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

// ApprovalSpec is the typed schema for the approval-server (the
// in-namespace HTTPS endpoint that validates and applies an SRE's
// approve/deny click on a JWT-signed proposal from the aiwatch).
//
// Schema-only in Phase 2c-A — the controller does NOT reconcile
// approval-server resources yet; the chart-managed install at
// `templates/approval-server-*.yaml` is still the only way to stand
// one up. Phase 2c-B will pick this up and produce:
//   - the `<cr>-approval-server` Deployment + Service
//   - the signing-key Secret (operator-generated Ed25519 keypair,
//     replacing the chart's pre-install keygen Job — Helm hooks
//     don't fit the controller model)
//   - RBAC: ClusterRoleBinding to the watcher's `-remediator` role
//     (approval-server applies fixes), namespace-local Role+Binding
//     for signing-key Secret read, audit Events emission, and
//     optionally the ConfigMap replay/runbook stores
//   - an optional `Ingress` (gated on `ingress.enabled`).
type ApprovalSpec struct {
	// Enabled is the master switch for the approval-server. Default
	// false; matches helm `approval.enabled`.
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// Replicas is the desired replica count. Default 1. Going > 1 is
	// only safe when `Store.Backend = "configmap"` — the in-memory
	// store can't dedupe replayed approval clicks across replicas, so
	// the operator pins the Deployment to Recreate strategy in that
	// mode. With the ConfigMap backend, RollingUpdate is honored.
	// +optional
	Replicas int32 `json:"replicas,omitempty"`

	// Image is the CHA-com binary image (same binary that powers the
	// aiwatch — the approval-server is a subcommand). Defaults to
	// `docker4zerocool/cha-com:v<OSS-tag>`, matching ai.image.
	// +optional
	Image *ImageSpec `json:"image,omitempty"`

	// SigningKey controls the Ed25519 signing-key Secret the
	// approval-server mounts at /etc/cha/keys/signing.key. The
	// aiwatch signs proposal JWTs with the same key; the
	// approval-server verifies them.
	// +optional
	SigningKey *ApprovalSigningKeySpec `json:"signingKey,omitempty"`

	// Store selects the durable replay-store backend. Default
	// `inmemory` (per-replica, suitable for single-replica installs).
	// Set `configmap` for HA installs — the operator stamps RBAC for
	// the named replay + runbook ConfigMaps automatically.
	// +optional
	Store *ApprovalStoreSpec `json:"store,omitempty"`

	// Ingress optionally exposes the approve/deny endpoint publicly
	// (typical for Slack approval flows that need the SRE to click a
	// link). Off by default — production users that drive approval
	// via in-cluster automation can leave it disabled.
	// +optional
	Ingress *ApprovalIngressSpec `json:"ingress,omitempty"`

	// AuditNamespace is the namespace the approval-server emits audit
	// Events into. Empty → CR namespace.
	// +optional
	AuditNamespace string `json:"auditNamespace,omitempty"`
}

// ApprovalSigningKeySpec controls the JWT signing-key Secret.
type ApprovalSigningKeySpec struct {
	// SecretName is the K8s Secret holding `signing.key` (Ed25519
	// private) + `signing.pub` (public). Default
	// `cha-approval-signing-key`. The operator creates this Secret
	// idempotently with a freshly-generated keypair if it doesn't
	// already exist (replacing the chart's pre-install keygen Job).
	// +optional
	SecretName string `json:"secretName,omitempty"`
}

// ApprovalStoreSpec selects the durable replay-store backend for the
// approval-server. `inmemory` (default) holds replay + runbook state in
// the pod's memory — fine for single-replica installs; loses state on
// restart. `configmap` persists both via Kubernetes ConfigMaps that the
// operator stamps RBAC for.
type ApprovalStoreSpec struct {
	// Backend selects the implementation.
	// +kubebuilder:validation:Enum=inmemory;configmap
	// +optional
	Backend string `json:"backend,omitempty"`

	// Namespace is the namespace the backend writes to. Empty → CR
	// namespace.
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// ReplayConfigMap is the name of the ConfigMap holding consumed
	// JWT JTIs (one-shot replay defense). Default
	// `cha-approval-replay`.
	// +optional
	ReplayConfigMap string `json:"replayConfigMap,omitempty"`

	// RunbookConfigMap is the name of the ConfigMap holding runbook
	// state (post-approval execution audit). Default
	// `cha-approval-runbooks`.
	// +optional
	RunbookConfigMap string `json:"runbookConfigMap,omitempty"`
}

// ApprovalIngressSpec optionally exposes the approval-server publicly.
type ApprovalIngressSpec struct {
	// Enabled is the gate. Off by default — production users that
	// drive approval purely via in-cluster Slack/email handlers can
	// leave it disabled.
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// IngressClassName names the Ingress class to bind. Empty → the
	// cluster's default class.
	// +optional
	IngressClassName string `json:"ingressClassName,omitempty"`

	// Host is the public hostname users will hit (e.g.
	// `approve.cha.example.com`). Required when Enabled.
	// +optional
	Host string `json:"host,omitempty"`

	// Annotations passes Ingress annotations through verbatim
	// (cert-manager, oauth2-proxy, kong, etc.).
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// TLS configures the Ingress's TLS termination.
	// +optional
	TLS *ApprovalIngressTLSSpec `json:"tls,omitempty"`
}

// ApprovalIngressTLSSpec is the TLS shape for the Ingress.
type ApprovalIngressTLSSpec struct {
	// Enabled adds the spec.tls block. cert-manager users typically
	// pair this with an annotation that auto-provisions the named
	// Secret.
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// SecretName is the Secret carrying the TLS cert+key. Empty →
	// `<cr>-approval-server-tls`.
	// +optional
	SecretName string `json:"secretName,omitempty"`
}
