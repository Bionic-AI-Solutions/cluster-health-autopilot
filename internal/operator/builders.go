// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

// Package operator hosts the pure-function builders that translate a
// ClusterHealthAutopilot spec into the concrete Kubernetes manifests
// the controller reconciles toward.
//
// Phase 1 ships the builders only. The controller-runtime Reconcile
// loop that USES the builders lands in Phase 2 (next PR) — separating
// the two keeps this PR digestible and reviewable as pure functions
// without a manager / envtest dependency.
//
// Convention:
//   - Each builder is a pure func: (cr) -> *appsv1.Deployment etc.
//   - The CR's name + namespace are the only sources of object names.
//   - Defaults are applied centrally (in DefaultedSpec) so each
//     builder can assume validated input.
package operator

import (
	"fmt"

	chav1alpha1 "github.com/Bionic-AI-Solutions/cluster-health-autopilot/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	// FieldManager is the field-manager name the controller uses on
	// server-side-apply patches. Pinned so two replicas of the
	// operator (HA via lease-based leader election) agree on
	// ownership.
	FieldManager = "cha-operator"

	// defaultWatcherDebounce / defaultWatcherResync mirror the
	// existing Helm chart defaults — keeps the operator-managed
	// install behavior-identical to the chart-managed install.
	defaultWatcherDebounce = "10s"
	defaultWatcherResync   = "10m"

	defaultDiagnoseSchedule         = "0 9 * * *"
	defaultDiagnoseBackoffLimit     = 1
	defaultDiagnoseActiveDeadlineS  = int64(120)
	defaultRemediateSchedule        = "*/30 * * * *"
	defaultRemediateBackoffLimit    = 1
	defaultRemediateActiveDeadlineS = int64(120)

	defaultSlackSecretKey = "WEBHOOK_URL"

	// AI / aiwatch defaults. Mirror the chart's `cha.aiArgs` helper —
	// the same wire-format the cha-com binary already accepts under
	// chart-managed installs. Any divergence between the chart and
	// these defaults is a Phase 2 regression worth a test.
	defaultAITier                  = "t0"
	defaultAIInterval              = "60s"
	defaultAIAPIKeyEnvName         = "AI_API_KEY"
	defaultAIAPIKeySecretKey       = "API_KEY"
	defaultAIImageRepo             = "docker4zerocool/cha-com"
	defaultAIMemoryTopK      int32 = 5

	// Qdrant defaults — mirror the chart's rag-qdrant-statefulset.yaml.
	defaultQdrantImageRepo   = "qdrant/qdrant"
	defaultQdrantImageTag    = "v1.12.4"
	defaultQdrantStorageSize = "5Gi"
	qdrantHTTPPort           = int32(6333)
	qdrantGRPCPort           = int32(6334)

	// roleLabelKey + roleLabelValue match the chart's labels so
	// existing CronJob/Deployment selectors keep working when an
	// install migrates from Helm to the operator.
	roleLabelKey = "cha.bionicaisolutions.com/role"
)

// ResourceNames are the canonical names the controller manages.
type ResourceNames struct {
	ServiceAccount string
	Watcher        string
	Diagnose       string
	Remediate      string
	AIWatch        string
	// RAG is shared by the Qdrant StatefulSet AND its ClusterIP
	// Service (both named identically — matches the chart). The
	// aiwatch's default `--memory-store-url` resolves to
	// `http://<RAG>.<ns>.svc:6333`.
	RAG string
}

// NamesFor returns the canonical resource names derived from the CR
// name. Pinned so renaming a CR is a forbidden operation (the
// controller refuses).
func NamesFor(cr *chav1alpha1.ClusterHealthAutopilot) ResourceNames {
	return ResourceNames{
		ServiceAccount: cr.Name + "-sa",
		Watcher:        cr.Name + "-watcher",
		Diagnose:       cr.Name + "-diagnose",
		Remediate:      cr.Name + "-remediate",
		AIWatch:        cr.Name + "-aiwatch",
		RAG:            cr.Name + "-rag",
	}
}

// ServiceAccountNameFor returns the SA the controller wires into
// every workload. Honors the explicit spec.ServiceAccountName
// override.
func ServiceAccountNameFor(cr *chav1alpha1.ClusterHealthAutopilot) string {
	if cr.Spec.ServiceAccountName != "" {
		return cr.Spec.ServiceAccountName
	}
	return NamesFor(cr).ServiceAccount
}

// CommonLabels are stamped on every controller-managed object so the
// chart's label selectors continue to match.
func CommonLabels(cr *chav1alpha1.ClusterHealthAutopilot, role string) map[string]string {
	labels := map[string]string{
		"app.kubernetes.io/name":       "cluster-health-autopilot",
		"app.kubernetes.io/instance":   cr.Name,
		"app.kubernetes.io/managed-by": FieldManager,
	}
	if role != "" {
		labels[roleLabelKey] = role
	}
	return labels
}

// imageRef returns the "<repo>:<tag>" reference for the workloads.
// Validation that Tag is non-empty happens in the controller before
// calling any builder.
func imageRef(spec chav1alpha1.ImageSpec) string {
	return fmt.Sprintf("%s:%s", spec.Repository, spec.Tag)
}

// pullSecretRefs converts a list of pull-secret names to the
// corev1.LocalObjectReference shape Kubernetes expects.
func pullSecretRefs(names []string) []corev1.LocalObjectReference {
	if len(names) == 0 {
		return nil
	}
	out := make([]corev1.LocalObjectReference, len(names))
	for i, n := range names {
		out[i] = corev1.LocalObjectReference{Name: n}
	}
	return out
}

// pullPolicy returns the PullPolicy to stamp on the container.
// Honors an explicit spec value; otherwise infers IfNotPresent for
// semver tags and Always for mutable tags (latest / main / dev).
func pullPolicy(spec chav1alpha1.ImageSpec) corev1.PullPolicy {
	if spec.PullPolicy != "" {
		return corev1.PullPolicy(spec.PullPolicy)
	}
	switch spec.Tag {
	case "latest", "main", "dev":
		return corev1.PullAlways
	}
	return corev1.PullIfNotPresent
}

// BuildServiceAccount returns the ServiceAccount the controller owns.
func BuildServiceAccount(cr *chav1alpha1.ClusterHealthAutopilot) *corev1.ServiceAccount {
	names := NamesFor(cr)
	if cr.Spec.ServiceAccountName != "" {
		names.ServiceAccount = cr.Spec.ServiceAccountName
	}
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "ServiceAccount"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      names.ServiceAccount,
			Namespace: cr.Namespace,
			Labels:    CommonLabels(cr, ""),
		},
	}
}

// BuildWatcherDeployment returns the watcher Deployment for the CR,
// or nil when the watcher is disabled.
func BuildWatcherDeployment(cr *chav1alpha1.ClusterHealthAutopilot) *appsv1.Deployment {
	if cr.Spec.Watcher == nil || !cr.Spec.Watcher.Enabled {
		return nil
	}
	names := NamesFor(cr)
	labels := CommonLabels(cr, "watcher")

	replicas := cr.Spec.Watcher.Replicas
	if replicas == 0 {
		replicas = 1
	}
	debounce := cr.Spec.Watcher.Debounce
	if debounce == "" {
		debounce = defaultWatcherDebounce
	}
	resync := cr.Spec.Watcher.ResyncPeriod
	if resync == "" {
		resync = defaultWatcherResync
	}

	args := []string{
		"watch",
		"--live",
		"--debounce=" + debounce,
		"--resync-period=" + resync,
	}
	args = append(args, alertingArgs(cr.Spec.Alerting, false)...)

	env := []corev1.EnvVar{
		{
			Name: "MY_POD_NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"},
			},
		},
		{
			Name: "MY_POD_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"},
			},
		},
	}
	env = append(env, alertingEnv(cr.Spec.Alerting)...)

	// v1.16.0 — When the CR's AI tier has approvalServerUrl set, hand
	// it to the watcher so the OSS binary itself can mint signed
	// approve/deny URLs via the ManifestBridge FixProposer registered
	// by cmd/cha/main.go. Before v1.16.0 only the aiwatch (cha-com)
	// pod minted URLs, but they never reached the watcher's Slack /
	// Alertmanager / ticketing adapters — leaving the SRE without
	// click-to-fix on otherwise-actionable findings.
	var watcherVolumes []corev1.Volume
	var watcherVolumeMounts []corev1.VolumeMount
	if cr.Spec.AI != nil && cr.Spec.AI.ApprovalServerURL != "" && cr.Spec.Approval != nil && cr.Spec.Approval.SigningKey != nil && cr.Spec.Approval.SigningKey.SecretName != "" {
		args = append(args,
			"--approval-server-url="+cr.Spec.AI.ApprovalServerURL,
			"--signing-key-path=/etc/cha/keys/signing.key",
		)
		watcherVolumes = []corev1.Volume{{
			Name: "signing-key",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: cr.Spec.Approval.SigningKey.SecretName,
					Items: []corev1.KeyToPath{{
						Key:  "signing.key",
						Path: "signing.key",
					}},
				},
			},
		}}
		watcherVolumeMounts = []corev1.VolumeMount{{
			Name:      "signing-key",
			MountPath: "/etc/cha/keys",
			ReadOnly:  true,
		}}
	}

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{APIVersion: "apps/v1", Kind: "Deployment"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      names.Watcher,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxSurge:       intstrPtr(1),
					MaxUnavailable: intstrPtr(0),
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					ServiceAccountName: ServiceAccountNameFor(cr),
					ImagePullSecrets:   pullSecretRefs(cr.Spec.Image.PullSecrets),
					Volumes:            watcherVolumes,
					Containers: []corev1.Container{
						{
							Name:            "watcher",
							Image:           imageRef(cr.Spec.Image),
							ImagePullPolicy: pullPolicy(cr.Spec.Image),
							Args:            args,
							Env:             env,
							VolumeMounts:    watcherVolumeMounts,
						},
					},
				},
			},
		},
	}
}

// BuildDiagnoseCronJob returns the diagnose CronJob for the CR, or
// nil when diagnose is disabled.
func BuildDiagnoseCronJob(cr *chav1alpha1.ClusterHealthAutopilot) *batchv1.CronJob {
	if cr.Spec.Diagnose == nil || !cr.Spec.Diagnose.Enabled {
		return nil
	}
	return buildCronJobCommon(cr, "diagnose", NamesFor(cr).Diagnose, cr.Spec.Diagnose.Schedule,
		cr.Spec.Diagnose.BackoffLimit, cr.Spec.Diagnose.ActiveDeadlineSeconds,
		[]string{"diagnose", "--live"},
		defaultDiagnoseSchedule, defaultDiagnoseBackoffLimit, defaultDiagnoseActiveDeadlineS,
	)
}

// BuildRemediateCronJob returns the remediate CronJob for the CR, or
// nil when remediate is disabled.
func BuildRemediateCronJob(cr *chav1alpha1.ClusterHealthAutopilot) *batchv1.CronJob {
	if cr.Spec.Remediate == nil || !cr.Spec.Remediate.Enabled {
		return nil
	}
	args := []string{"remediate", "--live"}
	if cr.Spec.Remediate.DryRun {
		args = append(args, "--dry-run=true")
	}
	return buildCronJobCommon(cr, "remediate", NamesFor(cr).Remediate, cr.Spec.Remediate.Schedule,
		0, 0, args,
		defaultRemediateSchedule, defaultRemediateBackoffLimit, defaultRemediateActiveDeadlineS,
	)
}

// buildCronJobCommon factors the shared shape between diagnose and
// remediate. Each caller supplies its own arg list and defaults so
// the resulting object reflects the role correctly.
func buildCronJobCommon(
	cr *chav1alpha1.ClusterHealthAutopilot,
	role, name, schedule string,
	backoffLimit int32, activeDeadline int64,
	args []string,
	defaultSchedule string, defaultBackoff int32, defaultDeadline int64,
) *batchv1.CronJob {
	if schedule == "" {
		schedule = defaultSchedule
	}
	if backoffLimit == 0 {
		backoffLimit = defaultBackoff
	}
	if activeDeadline == 0 {
		activeDeadline = defaultDeadline
	}
	labels := CommonLabels(cr, role)
	// diagnose posts the daily #healthinfo digest; remediate does not.
	args = append(args, alertingArgs(cr.Spec.Alerting, role == "diagnose")...)

	successfulJobsHistoryLimit := int32(3)
	failedJobsHistoryLimit := int32(3)

	return &batchv1.CronJob{
		TypeMeta: metav1.TypeMeta{APIVersion: "batch/v1", Kind: "CronJob"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: batchv1.CronJobSpec{
			Schedule:                   schedule,
			ConcurrencyPolicy:          batchv1.ForbidConcurrent,
			SuccessfulJobsHistoryLimit: &successfulJobsHistoryLimit,
			FailedJobsHistoryLimit:     &failedJobsHistoryLimit,
			JobTemplate: batchv1.JobTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: batchv1.JobSpec{
					BackoffLimit:          &backoffLimit,
					ActiveDeadlineSeconds: &activeDeadline,
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{Labels: labels},
						Spec: corev1.PodSpec{
							ServiceAccountName: ServiceAccountNameFor(cr),
							RestartPolicy:      corev1.RestartPolicyOnFailure,
							ImagePullSecrets:   pullSecretRefs(cr.Spec.Image.PullSecrets),
							Containers: []corev1.Container{
								{
									Name:            role,
									Image:           imageRef(cr.Spec.Image),
									Env:             alertingEnv(cr.Spec.Alerting),
									ImagePullPolicy: pullPolicy(cr.Spec.Image),
									Args:            args,
								},
							},
						},
					},
				},
			},
		},
	}
}

// alertingArgs turns the AlertingSpec into CLI flags the `cha` binary
// accepts. Slack webhook URLs are passed via K8s env-var expansion
// $(SLACK_*_URL) — the values are injected by alertingEnv() into the
// container's env.
//
// The `cha watch` subcommand only accepts --slack-alerts and
// --slack-critical; --slack-healthinfo is exclusive to `cha diagnose`
// (it posts the daily digest). Pass includeHealthInfo=false for the
// watcher Deployment and true for the diagnose CronJob.
func alertingArgs(a *chav1alpha1.AlertingSpec, includeHealthInfo bool) []string {
	if a == nil {
		return nil
	}
	var out []string
	if a.Alertmanager != nil && a.Alertmanager.URL != "" {
		out = append(out,
			"--alertmanager-url="+a.Alertmanager.URL,
			"--cluster-name="+a.Alertmanager.ClusterName,
		)
	}
	if a.Slack != nil {
		if c := a.Slack.Alerts; c != nil && c.SecretName != "" {
			out = append(out, "--slack-alerts=$(SLACK_ALERTS_URL)")
		}
		if c := a.Slack.Critical; c != nil && c.SecretName != "" {
			out = append(out, "--slack-critical=$(SLACK_CRITICAL_URL)")
		}
		if includeHealthInfo {
			if c := a.Slack.HealthInfo; c != nil && c.SecretName != "" {
				out = append(out, "--slack-healthinfo=$(SLACK_HEALTHINFO_URL)")
			}
		}
	}
	return out
}

// alertingEnv builds the env var slice that provides Slack webhook URLs
// via secretKeyRef so alertingArgs() can reference them as $(ENV_VAR).
// K8s expands $(FOO) in container args from the container's env at
// pod-start time, before the process exec.
func alertingEnv(a *chav1alpha1.AlertingSpec) []corev1.EnvVar {
	if a == nil || a.Slack == nil {
		return nil
	}
	var env []corev1.EnvVar
	if c := a.Slack.Alerts; c != nil && c.SecretName != "" {
		env = append(env, secretRefEnv("SLACK_ALERTS_URL", c.SecretName, defaultSlackSecretKey, c.SecretKey))
	}
	if c := a.Slack.Critical; c != nil && c.SecretName != "" {
		env = append(env, secretRefEnv("SLACK_CRITICAL_URL", c.SecretName, defaultSlackSecretKey, c.SecretKey))
	}
	if c := a.Slack.HealthInfo; c != nil && c.SecretName != "" {
		env = append(env, secretRefEnv("SLACK_HEALTHINFO_URL", c.SecretName, defaultSlackSecretKey, c.SecretKey))
	}
	return env
}

func secretRefEnv(name, secretName, defaultKey, overrideKey string) corev1.EnvVar {
	key := defaultKey
	if overrideKey != "" {
		key = overrideKey
	}
	return corev1.EnvVar{
		Name: name,
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
				Key:                  key,
			},
		},
	}
}

func intstrPtr(i int) *intstr.IntOrString {
	v := intstr.FromInt(i)
	return &v
}

// --- AI / aiwatch (Phase 2) ---

// aiImageRef returns the cha-com image reference. Defaults the repo to
// docker4zerocool/cha-com and the tag to `v<OSS image tag>` — matching
// the chart's convention where the cha-com image is tagged with a
// leading "v" alongside the OSS image's bare semver. Override via
// spec.ai.image.{repository,tag}.
func aiImageRef(cr *chav1alpha1.ClusterHealthAutopilot) string {
	repo := defaultAIImageRepo
	tag := "v" + cr.Spec.Image.Tag
	if ai := cr.Spec.AI; ai != nil && ai.Image != nil {
		if ai.Image.Repository != "" {
			repo = ai.Image.Repository
		}
		if ai.Image.Tag != "" {
			tag = ai.Image.Tag
		}
	}
	return fmt.Sprintf("%s:%s", repo, tag)
}

// aiPullPolicy mirrors pullPolicy() but reads spec.ai.image.pullPolicy
// when set; otherwise falls back to the default policy for the
// resolved aiwatch tag.
func aiPullPolicy(cr *chav1alpha1.ClusterHealthAutopilot) corev1.PullPolicy {
	if ai := cr.Spec.AI; ai != nil && ai.Image != nil {
		if ai.Image.PullPolicy != "" {
			return corev1.PullPolicy(ai.Image.PullPolicy)
		}
		// Same mutable-tag heuristic the OSS pullPolicy() applies.
		switch ai.Image.Tag {
		case "latest", "main", "dev":
			return corev1.PullAlways
		}
	}
	// Use the OSS image's tag as a proxy when ai.image.tag isn't
	// pinned — aiwatch defaults to the same semver, so the same
	// policy is right.
	return pullPolicy(cr.Spec.Image)
}

// AIEnabled reports whether spec.ai opts into the aiwatch Deployment.
// Pulls the nil-guard into one place so callers don't repeat it.
func AIEnabled(cr *chav1alpha1.ClusterHealthAutopilot) bool {
	return cr.Spec.AI != nil && cr.Spec.AI.Enabled
}

// aiArgs builds the `cha-com watch` CLI flags from spec.ai. Mirrors
// the chart's `cha.aiArgs` helper one-for-one — same flag names, same
// defaults, same skip-when-empty semantics. Order is stable so tests
// can match against a known sequence.
func aiArgs(cr *chav1alpha1.ClusterHealthAutopilot) []string {
	ai := cr.Spec.AI
	tier := ai.Tier
	if tier == "" {
		tier = defaultAITier
	}
	interval := ai.Interval
	if interval == "" {
		interval = defaultAIInterval
	}
	args := []string{
		"watch",
		"--ai-tier=" + tier,
		"--ai-endpoint=" + ai.Endpoint,
		"--ai-model=" + ai.Model,
		"--interval=" + interval,
	}
	if ai.APIKey != nil {
		if ai.APIKey.Header != "" {
			args = append(args, "--ai-api-key-header="+ai.APIKey.Header)
		}
		// envName defaults to AI_API_KEY but the chart only emits the
		// flag when an explicit override is set. Match that — keeps
		// the operator-managed install args byte-identical to the
		// chart-managed install for the same CR.
		if ai.APIKey.EnvName != "" {
			args = append(args, "--ai-api-key-env="+ai.APIKey.EnvName)
		}
	}
	if ai.AllowSaaS {
		args = append(args, "--ai-allow-saas")
	}
	if ai.LLMFixerMatcher {
		args = append(args, "--ai-llm-fixer-matcher")
	}
	if ai.AuditLog != "" {
		args = append(args, "--ai-audit-log="+ai.AuditLog)
	}
	if ai.ApprovalServerURL != "" {
		args = append(args, "--approval-server-url="+ai.ApprovalServerURL)
	}
	if ai.T3 != nil {
		for _, p := range ai.T3.VaultAllowedPrefixes {
			args = append(args, "--t3-vault-allowed-prefix="+p)
		}
	}
	if mem := ai.Memory; mem != nil && mem.Enabled {
		storeURL := mem.StoreURL
		if storeURL == "" {
			// Match the chart's in-namespace default. The operator
			// hasn't stood up Qdrant yet (deferred to a follow-up),
			// but if the user installed it via the chart in the same
			// namespace, this URL resolves correctly.
			storeURL = fmt.Sprintf("http://%s-rag.%s.svc:6333", cr.Name, cr.Namespace)
		}
		args = append(args, "--memory-store-url="+storeURL)
		if mem.Embeddings != nil {
			if mem.Embeddings.Endpoint != "" {
				args = append(args, "--memory-embeddings-endpoint="+mem.Embeddings.Endpoint)
			}
			args = append(args, "--memory-embeddings-model="+mem.Embeddings.Model)
		}
		topK := mem.TopK
		if topK == 0 {
			topK = defaultAIMemoryTopK
		}
		args = append(args, fmt.Sprintf("--memory-topk=%d", topK))
	}
	// v1.18.0 — extraArgs escape hatch. Append AFTER typed args so a
	// typed flag (e.g. --ai-tier) wins on duplicate keys (later args
	// override earlier ones in pflag). Useful for cha-com flags the
	// operator schema hasn't typed yet (e.g. --cloudflare-feeder,
	// --rag-store-url, --digest-pin-*).
	args = append(args, ai.ExtraArgs...)
	return args
}

// aiEnv builds the env var slice for the aiwatch container. Just the
// downward-API pair plus the LLM API-key secret reference; everything
// else flows via flags. Mirrors the chart's `cha.aiEnv`.
func aiEnv(cr *chav1alpha1.ClusterHealthAutopilot) []corev1.EnvVar {
	env := []corev1.EnvVar{
		{
			Name: "MY_POD_NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"},
			},
		},
		{
			Name: "MY_POD_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"},
			},
		},
	}
	if k := cr.Spec.AI.APIKey; k != nil && k.SecretName != "" {
		envName := k.EnvName
		if envName == "" {
			envName = defaultAIAPIKeyEnvName
		}
		secretKey := k.SecretKey
		if secretKey == "" {
			secretKey = defaultAIAPIKeySecretKey
		}
		env = append(env, corev1.EnvVar{
			Name: envName,
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: k.SecretName},
					Key:                  secretKey,
				},
			},
		})
	}
	// v1.18.0 — extraEnv escape hatch. Used for cha-com env vars the
	// operator schema doesn't model yet (e.g. GITHUB_PAT for the
	// digest-pin proposer's forge calls, CLOUDFLARE_API_TOKEN for the
	// zone-feeder). Validation enforces Value XOR ValueFrom at the
	// CR-admission layer (kubebuilder validators on the AIExtraEnv type).
	for _, ee := range cr.Spec.AI.ExtraEnv {
		if ee.Name == "" {
			continue
		}
		v := corev1.EnvVar{Name: ee.Name}
		if ee.ValueFrom != nil && ee.ValueFrom.SecretKeyRef != nil && ee.ValueFrom.SecretKeyRef.Name != "" {
			v.ValueFrom = &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: ee.ValueFrom.SecretKeyRef.Name},
					Key:                  ee.ValueFrom.SecretKeyRef.Key,
				},
			}
		} else {
			v.Value = ee.Value
		}
		env = append(env, v)
	}
	return env
}

// BuildAIWatchDeployment returns the aiwatch Deployment for the CR, or
// nil when AI is disabled. Mirrors `templates/aiwatch-deployment.yaml`:
//   - serviceAccount = the CR's reader-bound SA (same one the watcher
//     uses; aiwatch is read-only, every tier is recommendation-only
//     behind human approval).
//   - strategy = Recreate (the chart pins this — the aiwatch holds a
//     leader lease and rolling-update double-runs would split-brain
//     the approval-pair signing).
//   - replicas = 1 (no operator override yet; matches the chart).
//
// Required fields surfaced by the validator: ai.endpoint, ai.model.
// Empty memory.embeddings.model when memory.enabled is the only other
// must-validate.
func BuildAIWatchDeployment(cr *chav1alpha1.ClusterHealthAutopilot) *appsv1.Deployment {
	if !AIEnabled(cr) {
		return nil
	}
	names := NamesFor(cr)
	labels := CommonLabels(cr, "aiwatch")
	replicas := int32(1)

	pod := corev1.PodSpec{
		ServiceAccountName: ServiceAccountNameFor(cr),
		ImagePullSecrets:   pullSecretRefs(aiPullSecrets(cr)),
		Containers: []corev1.Container{
			{
				Name:            "aiwatch",
				Image:           aiImageRef(cr),
				ImagePullPolicy: aiPullPolicy(cr),
				Args:            aiArgs(cr),
				Env:             aiEnv(cr),
			},
		},
	}

	// When the approval-server is enabled, the aiwatch needs the Ed25519
	// signing key to mint click-to-fix JWT URLs. The chart mounts it
	// conditionally under approval.enabled; mirror that here. Without this
	// mount the cha-com binary crashes at startup with:
	//   "--approval-server-url set but CHA_SIGNING_KEY_PATH is empty"
	if ApprovalEnabled(cr) {
		secretName := ApprovalSigningKeySecretName(cr)
		pod.Volumes = append(pod.Volumes, corev1.Volume{
			Name: "signing-key",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: secretName,
					Items: []corev1.KeyToPath{
						// 0444 so the non-root container (UID 65532) can read the
						// private key. fsGroup would be the ideal mechanism but
						// requires the chart/CR to set podSecurityContext.fsGroup.
						{Key: "signing.key", Path: "signing.key", Mode: int32Ptr(0o444)},
						{Key: "signing.pub", Path: "signing.pub", Mode: int32Ptr(0o444)},
					},
				},
			},
		})
		c := &pod.Containers[0]
		c.Env = append(c.Env, corev1.EnvVar{
			Name:  "CHA_SIGNING_KEY_PATH",
			Value: "/etc/cha/keys/signing.key",
		})
		c.VolumeMounts = append(c.VolumeMounts, corev1.VolumeMount{
			Name:      "signing-key",
			MountPath: "/etc/cha/keys",
			ReadOnly:  true,
		})
	}

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{APIVersion: "apps/v1", Kind: "Deployment"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      names.AIWatch,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Strategy: appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec:       pod,
			},
		},
	}
}

// aiPullSecrets honors `spec.ai.image.pullSecrets` when set; otherwise
// falls back to the OSS image's pullSecrets (the chart-managed install
// applies the same list to both images via .Values.image.pullSecrets,
// so this keeps parity for callers that don't differentiate).
func aiPullSecrets(cr *chav1alpha1.ClusterHealthAutopilot) []string {
	if ai := cr.Spec.AI; ai != nil && ai.Image != nil && len(ai.Image.PullSecrets) > 0 {
		return ai.Image.PullSecrets
	}
	return cr.Spec.Image.PullSecrets
}

// --- Qdrant (Phase 2b) ---

// MemoryEnabled reports whether `spec.ai.memory.enabled` is true. The
// memory store is gated independently of AI itself — you can run
// aiwatch without memory (it falls back to in-process retrieval) and
// you can stand up Qdrant without enabling AI (e.g. preheat the index
// before a tier flip). Match the chart's behavior here exactly.
func MemoryEnabled(cr *chav1alpha1.ClusterHealthAutopilot) bool {
	return cr.Spec.AI != nil && cr.Spec.AI.Memory != nil && cr.Spec.AI.Memory.Enabled
}

// qdrantImageRef returns the "<repo>:<tag>" for the Qdrant container.
// Defaults match the chart's rag-qdrant-statefulset.yaml.
func qdrantImageRef(cr *chav1alpha1.ClusterHealthAutopilot) string {
	repo := defaultQdrantImageRepo
	tag := defaultQdrantImageTag
	if m := cr.Spec.AI.Memory; m != nil && m.Image != nil {
		if m.Image.Repository != "" {
			repo = m.Image.Repository
		}
		if m.Image.Tag != "" {
			tag = m.Image.Tag
		}
	}
	return fmt.Sprintf("%s:%s", repo, tag)
}

// qdrantPullPolicy mirrors aiPullPolicy. Defaults to IfNotPresent for
// the qdrant/qdrant semver tag.
func qdrantPullPolicy(cr *chav1alpha1.ClusterHealthAutopilot) corev1.PullPolicy {
	if m := cr.Spec.AI.Memory; m != nil && m.Image != nil {
		if m.Image.PullPolicy != "" {
			return corev1.PullPolicy(m.Image.PullPolicy)
		}
		switch m.Image.Tag {
		case "latest", "main", "dev":
			return corev1.PullAlways
		}
	}
	return corev1.PullIfNotPresent
}

// BuildQdrantService returns the ClusterIP Service the aiwatch reaches
// the vector store on. Two ports (6333 REST + 6334 gRPC) — match the
// chart. Returns nil when memory is disabled.
func BuildQdrantService(cr *chav1alpha1.ClusterHealthAutopilot) *corev1.Service {
	if !MemoryEnabled(cr) {
		return nil
	}
	labels := CommonLabels(cr, "rag")
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Service"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      NamesFor(cr).RAG,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       qdrantHTTPPort,
					TargetPort: intstr.FromString("http"),
					Protocol:   corev1.ProtocolTCP,
				},
				{
					Name:       "grpc",
					Port:       qdrantGRPCPort,
					TargetPort: intstr.FromString("grpc"),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
}

// BuildQdrantStatefulSet returns the Qdrant StatefulSet, or nil when
// memory is disabled. Single replica (chart matches — Qdrant doesn't
// horizontally scale here; the index is rebuildable from
// ResolutionRecord CRs). Storage flows via volumeClaimTemplates.
//
// IMPORTANT for reconcile path: K8s forbids changes to
// spec.{selector,serviceName,volumeClaimTemplates,podManagementPolicy}.
// reconcileQdrant must only mutate spec.{replicas,template,
// updateStrategy} on update.
func BuildQdrantStatefulSet(cr *chav1alpha1.ClusterHealthAutopilot) *appsv1.StatefulSet {
	if !MemoryEnabled(cr) {
		return nil
	}
	names := NamesFor(cr)
	labels := CommonLabels(cr, "rag")
	replicas := int32(1)

	storageSize := defaultQdrantStorageSize
	var storageClass *string
	if m := cr.Spec.AI.Memory; m != nil && m.Storage != nil {
		if m.Storage.Size != "" {
			storageSize = m.Storage.Size
		}
		if m.Storage.ClassName != "" {
			cn := m.Storage.ClassName
			storageClass = &cn
		}
	}
	storageQty, err := parseQuantity(storageSize)
	if err != nil {
		// Reconciler validates this before calling; the fallthrough is
		// defense in depth — we'd rather ship 5Gi than panic.
		storageQty = mustParseQuantity(defaultQdrantStorageSize)
	}

	return &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{APIVersion: "apps/v1", Kind: "StatefulSet"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      names.RAG,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: names.RAG,
			Replicas:    &replicas,
			Selector:    &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					ImagePullSecrets: pullSecretRefs(qdrantPullSecrets(cr)),
					Containers: []corev1.Container{
						{
							Name:            "qdrant",
							Image:           qdrantImageRef(cr),
							ImagePullPolicy: qdrantPullPolicy(cr),
							Ports: []corev1.ContainerPort{
								{Name: "http", ContainerPort: qdrantHTTPPort, Protocol: corev1.ProtocolTCP},
								{Name: "grpc", ContainerPort: qdrantGRPCPort, Protocol: corev1.ProtocolTCP},
							},
							// Match the chart's QDRANT__STORAGE__SNAPSHOTS_PATH +
							// _TEMP_PATH overrides. Without these Qdrant tries
							// to write to /qdrant/snapshots and /qdrant/.qdrant-temp
							// on the read-only image FS and CrashLoops.
							Env: []corev1.EnvVar{
								{Name: "QDRANT__STORAGE__SNAPSHOTS_PATH", Value: "/qdrant/storage/snapshots"},
								{Name: "QDRANT__STORAGE__TEMP_PATH", Value: "/qdrant/storage/temp"},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/readyz",
										Port: intstr.FromString("http"),
									},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       10,
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/livez",
										Port: intstr.FromString("http"),
									},
								},
								InitialDelaySeconds: 15,
								PeriodSeconds:       20,
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "storage", MountPath: "/qdrant/storage"},
							},
						},
					},
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "storage"},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						StorageClassName: storageClass,
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: storageQty,
							},
						},
					},
				},
			},
		},
	}
}

// qdrantPullSecrets honors spec.ai.memory.image.pullSecrets first, then
// falls back to the OSS image's pullSecrets (chart pulls Qdrant from
// the same registry as the OSS image in shared-secret installs).
func qdrantPullSecrets(cr *chav1alpha1.ClusterHealthAutopilot) []string {
	if m := cr.Spec.AI.Memory; m != nil && m.Image != nil && len(m.Image.PullSecrets) > 0 {
		return m.Image.PullSecrets
	}
	return cr.Spec.Image.PullSecrets
}

// parseQuantity wraps k8s resource.ParseQuantity. The Reconciler
// validates spec.ai.memory.storage.size at the door so the builder
// shouldn't see invalid values, but the wrapper keeps the call sites
// terse and the error path explicit.
func parseQuantity(s string) (resource.Quantity, error) {
	return resource.ParseQuantity(s)
}

// mustParseQuantity is the panic-on-failure variant used only with
// compile-time-known strings (the defaultQdrantStorageSize literal).
func mustParseQuantity(s string) resource.Quantity {
	q, err := resource.ParseQuantity(s)
	if err != nil {
		panic(fmt.Sprintf("parse compile-time quantity %q: %v", s, err))
	}
	return q
}
