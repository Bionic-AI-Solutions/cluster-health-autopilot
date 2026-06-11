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
// Phase 1.D: `TicketingSpec` now lives here so operator-managed CRs can
// drive issue-tracker delivery declaratively. The shape is intentionally
// OpenProject-shaped at v1alpha1 (OSS's only supported provider);
// Jira / ServiceNow knobs land in CHA-com via additive optional fields
// when those providers ship.
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

	// ExternalDNS configures the DNS-chain drift analyzer's optional
	// Cloudflare integration. Schema-only if the watcher image predates
	// v1.10; the analyzer reads it at registration time.
	// +optional
	ExternalDNS *ExternalDNSSpec `json:"externalDNS,omitempty"`

	// Ticketing wires the issue-tracker sink (OpenProject in OSS;
	// Jira / ServiceNow in CHA-com). When nil OR `Enabled` is false the
	// watcher behaves byte-identical to pre-1.D installs (no
	// `--ticketing-*` flags emitted). Set to enable per-finding
	// work-package creation + resolve-on-clear.
	// +optional
	Ticketing *TicketingSpec `json:"ticketing,omitempty"`

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

	// Triggers configures the v1.23.0+ external trigger sources
	// (Prometheus poller / webhook receiver). nil = legacy: only
	// class A (Kubernetes informers) + periodic resync drive the
	// diagnose cycle. v1.24.0 (Phase 1.7).
	// +optional
	Triggers *WatcherTriggersSpec `json:"triggers,omitempty"`
}

// WatcherTriggersSpec configures the external trigger sources.
type WatcherTriggersSpec struct {
	// Prom is the M5 Alertmanager polling consumer. nil = disabled.
	// +optional
	Prom *WatcherPromTriggerSpec `json:"prom,omitempty"`

	// Webhook is the M6 HMAC-authed POST receiver. nil = disabled.
	// +optional
	Webhook *WatcherWebhookTriggerSpec `json:"webhook,omitempty"`
}

// WatcherPromTriggerSpec configures the M5 Prometheus class-C trigger.
type WatcherPromTriggerSpec struct {
	// URL is the Alertmanager base URL polled for firing alerts.
	// Required when this stanza is set.
	// +kubebuilder:validation:MinLength=1
	URL string `json:"url"`

	// Interval is the polling cadence. Clamped to ≥5s by the trigger
	// client. Default 30s when zero.
	// +optional
	Interval string `json:"interval,omitempty"`

	// AlertNameFilter limits which alertnames fire the trigger
	// (case-insensitive). Empty = ANY firing alert triggers.
	// +optional
	AlertNameFilter []string `json:"alertNameFilter,omitempty"`
}

// WatcherWebhookTriggerSpec configures the M6 webhook class-E receiver.
type WatcherWebhookTriggerSpec struct {
	// Listen is the HTTP listen address (e.g. ":8090"). Required.
	// +kubebuilder:validation:MinLength=1
	Listen string `json:"listen"`

	// Sources is the operator-supplied list of registered webhook
	// sources, each "<name>=<env-var-with-hmac-secret>". The env-var
	// is looked up at startup; the matching value comes from
	// SecretName below. Empty env-var disables HMAC verification
	// (debug-only).
	// +optional
	Sources []string `json:"sources,omitempty"`

	// SecretName is the K8s Secret carrying the env-var values for
	// each --webhook-source entry. Each key in the Secret must match
	// the env-var name on the right side of <name>=<env-var>.
	// +optional
	SecretName string `json:"secretName,omitempty"`

	// ServiceEnabled renders a ClusterIP Service exposing the
	// receiver inside the cluster. Default false. Set true to make
	// the receiver reachable from cluster Ingress / Kong route.
	// +optional
	ServiceEnabled bool `json:"serviceEnabled,omitempty"`

	// ServicePort is the Service port (when ServiceEnabled).
	// Default 8090 when zero.
	// +optional
	ServicePort int32 `json:"servicePort,omitempty"`
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

	// ActiveDeadlineSeconds caps the per-Job wall-clock budget. Default
	// 120s — fine for healthy clusters where remediate has 0–5 actions.
	// Bump to 600–900s on busy clusters with many SecurityDrift
	// proposals + DigestPin candidates queued up; the analyzer pass
	// inside remediate scales with finding count. v1.25.1.
	// +optional
	ActiveDeadlineSeconds int64 `json:"activeDeadlineSeconds,omitempty"`
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

	// ExtraArgs are passed verbatim as additional command-line
	// arguments to the aiwatch container. Order-preserving append
	// AFTER the typed args (--ai-tier, --ai-endpoint, etc).
	//
	// Escape hatch for cha-com flags the operator's typed schema
	// doesn't yet model — currently used for the v1.11.0+ runtime
	// activations (--cloudflare-feeder, --rag-store-url,
	// --cluster-name, --digest-pin-proposer, --forge-token-env,
	// --digest-pin-repo-map, etc). Each entry is one full CLI token,
	// e.g. ["--cloudflare-feeder=true", "--rag-store-url=http://...",
	// "--digest-pin-repo-map=foo/bar=Org/Bar:main"].
	//
	// Will be replaced by typed fields in future minor releases as
	// the corresponding cha-com surfaces stabilize; the escape hatch
	// stays for forward-compat with experimental flags.
	// +optional
	ExtraArgs []string `json:"extraArgs,omitempty"`

	// ExtraEnv are extra environment variables appended to the
	// aiwatch container. Mirrors the structure of corev1.EnvVar
	// (subset: Name + Value OR Name + ValueFrom.SecretKeyRef).
	//
	// Used for v1.11.0+ external-token references (GITHUB_PAT for
	// the digest-pin proposer's forge calls, CLOUDFLARE_API_TOKEN
	// for the zone-feeder). Each entry must specify EITHER Value
	// OR ValueFrom (not both); the operator rejects the CR at
	// validation time otherwise.
	// +optional
	ExtraEnv []AIExtraEnv `json:"extraEnv,omitempty"`

	// Replicas is the aiwatch deployment replica count. v1.21.0
	// (Phase 2.F) — when >1, the watcher uses coordination.k8s.io/v1
	// leader-election so exactly one replica runs tick() at a time;
	// the others stand by ready to take over within ~30s on lease
	// loss. Default 1 (single-replica, byte-identical to pre-2.F).
	//
	// The chart turns on the --leader-election flag on the aiwatch
	// container when Replicas > 1; with Replicas == 1, the watcher
	// takes the noop elector path (no Lease object created, no RBAC
	// dependency on coordination.k8s.io). RBAC required for >1:
	// the aiwatch ServiceAccount needs verbs {get, list, watch,
	// create, update, patch} on coordination.k8s.io/leases in the
	// install namespace. The chart adds these only when Replicas > 1.
	//
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=5
	// +optional
	Replicas int32 `json:"replicas,omitempty"`

	// DigestPinAttestation enables Phase 2.H cosign-style PR-body
	// attestation on DigestPin-proposer PRs. When set + secretName
	// resolves, the aiwatch container mounts the Ed25519 private key
	// at /etc/cha/keys/attestation.key and signs a canonical payload
	// per PR. Signature + embedded public-key PEM are appended to the
	// PR body so reviewers can verify with `openssl` before approving.
	//
	// nil → no attestation (legacy body; PAT-based auth still gates).
	// Soft-fail on key-read errors: PR opens, just without the block.
	// +optional
	DigestPinAttestation *DigestPinAttestationSpec `json:"digestPinAttestation,omitempty"`

	// Metrics promotes the Phase 2.G chart-only `ai.metrics` fields
	// into the CR so operator-managed installs (ArgoCD/Flux) don't
	// need `--metrics-addr` in extraArgs. The operator reconciler
	// renders the matching container port + Service when set.
	// nil → no /metrics endpoint (legacy single-binary deploy).
	// Phase 3.D.
	// +optional
	Metrics *AIMetricsSpec `json:"metrics,omitempty"`

	// LLMProposer enables the Phase 2.D LLM-driven fallback proposer
	// for diagnostics outside FixProposer's keyword set. nil = off
	// (legacy click-to-fix only path). When `enabled: true`, the
	// proposer uses the same LLM endpoint + API key as the rest of
	// the AI tier — no additional config required.
	// Phase 3.D — promotes the existing CLI flag into a typed field.
	// +optional
	LLMProposer *AILLMProposerSpec `json:"llmProposer,omitempty"`
}

// AIMetricsSpec configures the Phase 2.G /metrics endpoint on the
// aiwatch container. nil = endpoint off. Phase 3.D.
type AIMetricsSpec struct {
	// Addr is the bind address (e.g. ":9090"). Required when set.
	// +kubebuilder:validation:MinLength=1
	Addr string `json:"addr"`

	// Port is the matching container/Service port. Default 9090
	// when zero. Operator reconciles a same-namespace headless
	// Service exposing this port labeled for ServiceMonitor selection.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +optional
	Port int32 `json:"port,omitempty"`
}

// AILLMProposerSpec toggles the Phase 2.D LLM-driven fallback
// proposer. Phase 3.D.
type AILLMProposerSpec struct {
	// Enabled is the master switch. Default false (no fallback;
	// legacy click-to-fix-only behaviour for non-FixProposer
	// diagnostics).
	Enabled bool `json:"enabled"`
}

// DigestPinAttestationSpec configures the Phase 2.H attestation
// signer for DigestPin-proposer PRs. Phase 2.H.
type DigestPinAttestationSpec struct {
	// SecretName references the K8s Secret holding the Ed25519
	// private key. The Secret MUST carry an `attestation.key` data
	// key whose value is the base64-encoded 64-byte ed25519 private
	// key (same on-disk format the approval-server signing key uses).
	// ESO + Vault is the recommended provisioning path.
	// +kubebuilder:validation:MinLength=1
	SecretName string `json:"secretName"`

	// SecretKey is the data key inside SecretName that carries the
	// base64 key. Defaults to "attestation.key" when empty.
	// +optional
	SecretKey string `json:"secretKey,omitempty"`

	// KeyID is the kid stamped into every attestation. Defaults to
	// "cha-digest-pin" when empty.
	// +optional
	KeyID string `json:"keyID,omitempty"`
}

// AIExtraEnv is a minimal env-var spec — supports literal Value or a
// secretKeyRef. ConfigMapKeyRef + FieldRef + ResourceFieldRef are out
// of scope (the aiwatch never needs them in production).
type AIExtraEnv struct {
	// Name is the env var name as seen by the binary.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Value is a literal value. Mutually exclusive with ValueFrom.
	// +optional
	Value string `json:"value,omitempty"`

	// ValueFrom pulls from a K8s Secret. Mutually exclusive with Value.
	// +optional
	ValueFrom *AIExtraEnvSource `json:"valueFrom,omitempty"`
}

// AIExtraEnvSource is the secretKeyRef shape — only the fields the
// aiwatch consumes (no ConfigMap / FieldRef / ResourceFieldRef in
// scope for the operator surface).
type AIExtraEnvSource struct {
	// SecretKeyRef references a K8s Secret in the install namespace.
	// +kubebuilder:validation:Required
	SecretKeyRef *AIExtraEnvSecretKeyRef `json:"secretKeyRef"`
}

// AIExtraEnvSecretKeyRef identifies a (Secret, key) pair.
type AIExtraEnvSecretKeyRef struct {
	// Name of the K8s Secret in the install namespace.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Key inside the Secret.
	// +kubebuilder:validation:MinLength=1
	Key string `json:"key"`
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

	// NetworkPolicy optionally restricts ingress to the approval-server
	// pods to ONLY the gateway/oauth2-proxy namespace, closing the
	// X-Forwarded-User header-forgery bypass (a pod that reaches the
	// ClusterIP directly could forge the identity header the OIDC
	// ingress injects). Off by default — see ApprovalNetworkPolicySpec.
	// +optional
	NetworkPolicy *ApprovalNetworkPolicySpec `json:"networkPolicy,omitempty"`

	// AuditNamespace is the namespace the approval-server emits audit
	// Events into. Empty → CR namespace.
	// +optional
	AuditNamespace string `json:"auditNamespace,omitempty"`
}

// ApprovalNetworkPolicySpec restricts ingress to the approval-server
// pods to traffic originating in the gateway namespace (where
// oauth2-proxy / the OIDC ingress controller live).
//
// Why this matters: the approval-server trusts the `X-Forwarded-User`
// header for audit attribution. That header is injected by oauth2-proxy
// at the OIDC ingress after a successful login. Internal cluster traffic
// that hits the approval-server ClusterIP directly bypasses the ingress
// and can therefore forge an arbitrary `X-Forwarded-User`. The
// approve/deny click still requires a valid one-time signed token, so
// this is defense-in-depth for audit-attribution honesty — but it closes
// a real bypass. With the NetworkPolicy in place, only pods in the
// gateway namespace can reach port 8443, so the only X-Forwarded-User
// the server ever sees is the one oauth2-proxy set.
//
// Default OFF (Enabled=false). A NetworkPolicy is fail-closed by nature:
// once any policy selects a pod, all non-matching ingress is dropped. If
// this defaulted ON and the operator guessed the gateway namespace's
// labels wrong (or the cluster's CNI doesn't enforce NetworkPolicy yet
// labels it as if it does), every approval click would silently 0-route
// — a worse outcome than the header-forgery bug it closes. So it is
// opt-in, with a loud recommendation to enable it in production.
//
// When Enabled, GatewayNamespaceSelector is REQUIRED (validated at the
// reconciler door, fail-closed). There is intentionally no default
// selector: a wrong default would either over-permit (some unrelated
// namespace happens to carry the guessed label) or block everything (no
// namespace matches) — both silent. Forcing the operator to declare the
// gateway namespace's labels makes the trust boundary explicit.
type ApprovalNetworkPolicySpec struct {
	// Enabled is the gate. Off by default. Strongly recommended in
	// production once the gateway namespace's labels are known.
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// GatewayNamespaceSelector is the label selector identifying the
	// namespace(s) allowed to reach the approval-server on port 8443
	// — i.e. where oauth2-proxy / the OIDC ingress controller run.
	// REQUIRED when Enabled. A common value on clusters that enable the
	// `NamespaceDefaultLabelName` feature gate is
	// `{kubernetes.io/metadata.name: <gateway-namespace>}`, which the
	// apiserver auto-stamps on every namespace.
	// +optional
	GatewayNamespaceSelector map[string]string `json:"gatewayNamespaceSelector,omitempty"`
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

// ExternalDNSSpec configures the DNS-chain drift analyzer's optional
// Cloudflare integration. Schema-only if the watcher image predates
// v1.10; the analyzer reads it at registration time.
type ExternalDNSSpec struct {
	// Cloudflare is the Cloudflare-specific DNS drift config.
	// +optional
	Cloudflare *CloudflareSpec `json:"cloudflare,omitempty"`
}

// CloudflareSpec configures the Cloudflare DNS chain drift analyzer.
type CloudflareSpec struct {
	// Enabled is the master switch for Cloudflare-backed DNS drift
	// detection. When false the analyzer is not registered even if
	// APITokenSecretRef is set.
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// APITokenSecretRef references the K8s Secret holding the
	// Cloudflare API token. Required when Enabled.
	// +optional
	APITokenSecretRef *CloudflareSecretRef `json:"apiTokenSecretRef,omitempty"`

	// ZoneIDs is the list of Cloudflare zone IDs to inspect. Empty
	// means "all zones accessible to the token."
	// +optional
	ZoneIDs []string `json:"zoneIDs,omitempty"`

	// ExpectedTargets is the list of hostnames (or CIDR-prefix strings)
	// the analyzer accepts as valid DNS targets. Drift is flagged when
	// a resolved CNAME/A record is not in this list.
	// +optional
	ExpectedTargets []string `json:"expectedTargets,omitempty"`
}

// CloudflareSecretRef points at the K8s Secret holding the Cloudflare
// API token. All fields are plain strings — value-copy is sufficient,
// no explicit DeepCopy needed.
type CloudflareSecretRef struct {
	// Name is the Kubernetes Secret name in the install namespace.
	Name string `json:"name"`

	// Key is the key inside the Secret. Defaults to "token".
	// +optional
	Key string `json:"key,omitempty"`
}

// TicketingSpec controls the issue-tracker sink the watcher drives
// each cycle. When Enabled is true the operator passes the matching
// `--ticketing-*` flags to the watcher container.
//
// Provider-shape note: v1alpha1 is OpenProject-shaped (OSS's only
// supported provider). Jira / ServiceNow knobs land in CHA-com when
// those providers ship, as additive optional fields on this spec.
type TicketingSpec struct {
	// Enabled is the master switch. nil OR false → no `--ticketing-*`
	// flags are emitted (byte-identical to pre-1.D installs).
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// Provider names the issue-tracker. OSS supports `openproject`;
	// CHA-com adds `jira` / `servicenow`. Required when Enabled.
	// +kubebuilder:validation:Enum=openproject;jira;servicenow
	// +optional
	Provider string `json:"provider,omitempty"`

	// Cluster identifies this cluster in ticket bodies (e.g. "bionic").
	// Mirrors alerting.alertmanager.clusterName; defaults to "cluster"
	// when empty.
	// +optional
	Cluster string `json:"cluster,omitempty"`

	// MCPURL is the issue-tracker MCP server endpoint
	// (Streamable-HTTP transport). Required when Enabled. In-cluster
	// default: http://mcp-openproject-server.mcp.svc:8006/mcp
	// +optional
	MCPURL string `json:"mcpURL,omitempty"`

	// Project is the OpenProject project ID (numeric string;
	// discoverable via the MCP server's list_projects tool).
	// +optional
	Project string `json:"project,omitempty"`

	// TypeID is the work-package type ID (e.g. "36" for Task).
	// Required when Enabled — most installs use Task.
	// +optional
	TypeID string `json:"typeID,omitempty"`

	// ClosedStatusID is the status ID applied when a finding
	// auto-resolves (e.g. "82" for Closed). Needed for resolve-on-clear.
	// +optional
	ClosedStatusID string `json:"closedStatusID,omitempty"`

	// WebURLPrefix is the OpenProject web UI base URL (e.g.
	// `https://op.example.com`). Used to render operator-clickable
	// TicketRef.URL on the persisted DriftReport CRs.
	// +optional
	WebURLPrefix string `json:"webURLPrefix,omitempty"`

	// SeverityPriority maps CHA severities → provider priority IDs.
	// Empty values let the provider use its project default.
	// +optional
	SeverityPriority *TicketingPrioritySpec `json:"severityPriority,omitempty"`

	// Labels are appended to every ticket body for filtering.
	// Defaults to ["cha", "auto-filed"] in the chart; the operator
	// passes whatever this slice contains, including empty.
	// +optional
	Labels []string `json:"labels,omitempty"`

	// DryRun=true makes the watcher LOG intended ticket operations
	// without calling the MCP server. Useful for a first deployment.
	// +optional
	DryRun bool `json:"dryRun,omitempty"`

	// ResolveOnClear toggles auto-closing a ticket when its finding
	// clears (M2). Defaults ON: nil → the operator emits
	// `--ticketing-resolve-on-clear=true`. Set to a pointer-to-false to
	// disable. No-op when ticketing is disabled.
	// +optional
	ResolveOnClear *bool `json:"resolveOnClear,omitempty"`

	// CommentInterval is the debounce window for comment-on-recurrence
	// (M2), as a Go duration string (e.g. "1h"). A recurring or
	// severity-changed finding comments on the EXISTING ticket at most
	// once per window. "0" disables recurrence commenting. Empty →
	// chart/binary default ("1h").
	// +optional
	CommentInterval string `json:"commentInterval,omitempty"`

	// Auth optionally configures an MCP API key (typically used when
	// the MCP server is fronted by Kong key-auth). Leave nil for
	// in-cluster HTTP traffic.
	// +optional
	Auth *TicketingAuthSpec `json:"auth,omitempty"`
}

// TicketingPrioritySpec maps CHA severities to provider priority IDs.
type TicketingPrioritySpec struct {
	// Critical priority ID (e.g. "75" for OpenProject Immediate).
	// +optional
	Critical string `json:"critical,omitempty"`
	// Warning priority ID (e.g. "74" for OpenProject High).
	// +optional
	Warning string `json:"warning,omitempty"`
	// Info priority ID (e.g. "73" for OpenProject Normal).
	// +optional
	Info string `json:"info,omitempty"`
}

// TicketingAuthSpec configures an MCP API key for the ticketing sink.
type TicketingAuthSpec struct {
	// Enabled gates whether the operator wires the TICKETING_MCP_API_KEY
	// env var. False = no env var (default; matches in-cluster traffic).
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// SecretName is the K8s Secret holding the API key. Required when
	// Enabled — typically populated by ESO from Vault.
	// +optional
	SecretName string `json:"secretName,omitempty"`

	// SecretKey is the key inside the Secret. Defaults to "api-key".
	// +optional
	SecretKey string `json:"secretKey,omitempty"`
}
