// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"

	chav1alpha1 "github.com/Bionic-AI-Solutions/cluster-health-autopilot/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// Phase 2c-B — operator-provisioned approval-server.
//
// The approval-server is the HTTPS endpoint the SRE clicks
// approve/deny on for AI-proposed remediations. It runs as a
// subcommand of the same cha-com binary the aiwatch uses.
//
// Resources reconciled (when spec.approval.enabled=true):
//   - core/v1 ServiceAccount         <cr>-approval-server
//   - apps/v1 Deployment             <cr>-approval-server
//   - core/v1 Service                <cr>-approval-server (ClusterIP, 8443)
//   - core/v1 Secret                 spec.approval.signingKey.secretName
//                                    (Ed25519 keypair, operator-generated)
//   - rbac.authorization.k8s.io/v1 ClusterRole
//                                    cha-operator-approval-fixer (shared)
//   - rbac.authorization.k8s.io/v1 ClusterRoleBinding
//                                    cha-operator-approval-fixer-<ns>-<name>
//                                    (per-CR; cleaned up by finalizer)
//   - rbac.authorization.k8s.io/v1 Role + RoleBinding (3 pairs)
//                                    - signing-key Secret read (resourceNames-scoped)
//                                    - events emission
//                                    - ConfigMap stores (only when
//                                      Store.Backend == "configmap")
//
// The Ingress is shipped separately in Phase 2c-C.
//
// The chart's approach to the signing-key Secret was a Helm pre-install
// keygen Job. The operator generates the Ed25519 keypair in-process via
// crypto/ed25519 (stdlib) and creates the Secret idempotently — Helm
// hooks don't fit the controller model.

const (
	// ApprovalFixerClusterRoleName is the cluster-scoped role granting
	// the mutation verbs every fixer needs (delete pods/jobs/orders/
	// certificaterequests + patch deployments). Shared across all CRs
	// in the cluster; never deleted by the operator (other CRs may
	// still bind to it).
	ApprovalFixerClusterRoleName = "cha-operator-approval-fixer"

	// DefaultApprovalSigningKeySecretName matches the chart's
	// `approval.signingKey.secretName` default. The aiwatch (Phase 2)
	// already mounts the chart's Secret at this name when running
	// alongside a chart install; using the same default lets an
	// operator-managed install drop into an existing cluster.
	DefaultApprovalSigningKeySecretName = "cha-approval-signing-key"

	// DefaultApprovalReplayConfigMap matches the chart's
	// `approval.store.replayConfigMap` default.
	DefaultApprovalReplayConfigMap = "cha-approval-replay"

	// DefaultApprovalRunbookConfigMap matches the chart's
	// `approval.store.runbookConfigMap` default.
	DefaultApprovalRunbookConfigMap = "cha-approval-runbooks"

	approvalServerHTTPPort = int32(8443)
	approvalImageDefault   = defaultAIImageRepo // same docker4zerocool/cha-com binary
)

// ApprovalEnabled is the nil-guarded check.
func ApprovalEnabled(cr *chav1alpha1.ClusterHealthAutopilot) bool {
	return cr.Spec.Approval != nil && cr.Spec.Approval.Enabled
}

// ApprovalSigningKeySecretName returns the name of the Secret holding
// the Ed25519 signing keypair. Honors `spec.approval.signingKey.secretName`
// when set; otherwise the chart-convention default.
func ApprovalSigningKeySecretName(cr *chav1alpha1.ClusterHealthAutopilot) string {
	if cr.Spec.Approval != nil &&
		cr.Spec.Approval.SigningKey != nil &&
		cr.Spec.Approval.SigningKey.SecretName != "" {
		return cr.Spec.Approval.SigningKey.SecretName
	}
	return DefaultApprovalSigningKeySecretName
}

// ApprovalServerName is the shared name for the SA + Deployment +
// Service. Matches the chart's `<release>-approval-server`.
func ApprovalServerName(cr *chav1alpha1.ClusterHealthAutopilot) string {
	return cr.Name + "-approval-server"
}

// ApprovalFixerClusterRoleBindingName is the per-CR fixer binding name.
// `<ns>-<name>` shape keeps it globally unique and trivially derivable
// for the finalizer.
func ApprovalFixerClusterRoleBindingName(cr *chav1alpha1.ClusterHealthAutopilot) string {
	return ApprovalFixerClusterRoleName + "-" + cr.Namespace + "-" + cr.Name
}

// ApprovalAuditNamespace returns the namespace the approval-server
// emits audit events into. Defaults to the CR's own namespace.
func ApprovalAuditNamespace(cr *chav1alpha1.ClusterHealthAutopilot) string {
	if cr.Spec.Approval != nil && cr.Spec.Approval.AuditNamespace != "" {
		return cr.Spec.Approval.AuditNamespace
	}
	return cr.Namespace
}

// approvalReplicas defaults to 1 unless explicitly set higher AND the
// ConfigMap store backend is in use (the in-memory store can't dedupe
// replay across replicas — going > 1 in that mode would race).
func approvalReplicas(cr *chav1alpha1.ClusterHealthAutopilot) int32 {
	if cr.Spec.Approval == nil || cr.Spec.Approval.Replicas == 0 {
		return 1
	}
	return cr.Spec.Approval.Replicas
}

// approvalStoreBackend returns the resolved store backend value or
// "inmemory" by default.
func approvalStoreBackend(cr *chav1alpha1.ClusterHealthAutopilot) string {
	if cr.Spec.Approval != nil &&
		cr.Spec.Approval.Store != nil &&
		cr.Spec.Approval.Store.Backend != "" {
		return cr.Spec.Approval.Store.Backend
	}
	return "inmemory"
}

// approvalStoreConfigMap pair returns the resolved replay + runbook
// ConfigMap names.
func approvalStoreConfigMaps(cr *chav1alpha1.ClusterHealthAutopilot) (replay, runbook string) {
	replay = DefaultApprovalReplayConfigMap
	runbook = DefaultApprovalRunbookConfigMap
	if cr.Spec.Approval != nil && cr.Spec.Approval.Store != nil {
		if cr.Spec.Approval.Store.ReplayConfigMap != "" {
			replay = cr.Spec.Approval.Store.ReplayConfigMap
		}
		if cr.Spec.Approval.Store.RunbookConfigMap != "" {
			runbook = cr.Spec.Approval.Store.RunbookConfigMap
		}
	}
	return replay, runbook
}

// approvalImageRef mirrors aiImageRef — same default repo (cha-com),
// same `v<OSS-tag>` convention.
func approvalImageRef(cr *chav1alpha1.ClusterHealthAutopilot) string {
	repo := approvalImageDefault
	tag := "v" + cr.Spec.Image.Tag
	if ap := cr.Spec.Approval; ap != nil && ap.Image != nil {
		if ap.Image.Repository != "" {
			repo = ap.Image.Repository
		}
		if ap.Image.Tag != "" {
			tag = ap.Image.Tag
		}
	}
	return fmt.Sprintf("%s:%s", repo, tag)
}

// approvalPullPolicy mirrors aiPullPolicy.
func approvalPullPolicy(cr *chav1alpha1.ClusterHealthAutopilot) corev1.PullPolicy {
	if ap := cr.Spec.Approval; ap != nil && ap.Image != nil {
		if ap.Image.PullPolicy != "" {
			return corev1.PullPolicy(ap.Image.PullPolicy)
		}
		switch ap.Image.Tag {
		case "latest", "main", "dev":
			return corev1.PullAlways
		}
	}
	return pullPolicy(cr.Spec.Image)
}

// approvalPullSecrets honors spec.approval.image.pullSecrets first,
// then the OSS image's pullSecrets.
func approvalPullSecrets(cr *chav1alpha1.ClusterHealthAutopilot) []string {
	if ap := cr.Spec.Approval; ap != nil && ap.Image != nil && len(ap.Image.PullSecrets) > 0 {
		return ap.Image.PullSecrets
	}
	return cr.Spec.Image.PullSecrets
}

// BuildApprovalServerServiceAccount returns the SA the approval-server
// container runs under. Distinct from the watcher's SA — the
// approval-server gets its own narrow set of bindings (fixer
// ClusterRoleBinding + 3 namespaced Roles), while the watcher SA gets
// the reader role.
func BuildApprovalServerServiceAccount(cr *chav1alpha1.ClusterHealthAutopilot) *corev1.ServiceAccount {
	if !ApprovalEnabled(cr) {
		return nil
	}
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "ServiceAccount"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      ApprovalServerName(cr),
			Namespace: cr.Namespace,
			Labels:    CommonLabels(cr, "approval-server"),
		},
	}
}

// BuildApprovalServerService returns the ClusterIP Service in front of
// the approval-server pods. Port 8443/http (the binary terminates TLS
// itself when configured; the Ingress fronts plain HTTP for kind
// installs).
func BuildApprovalServerService(cr *chav1alpha1.ClusterHealthAutopilot) *corev1.Service {
	if !ApprovalEnabled(cr) {
		return nil
	}
	labels := CommonLabels(cr, "approval-server")
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Service"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      ApprovalServerName(cr),
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       approvalServerHTTPPort,
					TargetPort: intstr.FromString("http"),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
}

// BuildApprovalServerDeployment returns the approval-server Deployment.
// Mirrors the chart's `templates/approval-server-deployment.yaml`:
//   - `approval-server` subcommand of cha-com.
//   - --signing-key-path mount.
//   - --store-* flags only when Store.Backend != "inmemory" (chart
//     only emits them when the backend is set, so match that).
//   - Strategy = Recreate for inmemory store (replay state would
//     split-brain across replicas), RollingUpdate for ConfigMap.
func BuildApprovalServerDeployment(cr *chav1alpha1.ClusterHealthAutopilot) *appsv1.Deployment {
	if !ApprovalEnabled(cr) {
		return nil
	}
	labels := CommonLabels(cr, "approval-server")
	replicas := approvalReplicas(cr)
	backend := approvalStoreBackend(cr)

	strategy := appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType}
	if backend == "configmap" {
		strategy = appsv1.DeploymentStrategy{
			Type: appsv1.RollingUpdateDeploymentStrategyType,
			RollingUpdate: &appsv1.RollingUpdateDeployment{
				MaxSurge:       intstrPtr(1),
				MaxUnavailable: intstrPtr(0),
			},
		}
	}

	args := []string{
		"approval-server",
		"--listen=:8443",
		"--signing-key-path=/etc/cha/keys/signing.key",
		"--audit-namespace=" + ApprovalAuditNamespace(cr),
	}
	if backend != "inmemory" {
		replayCM, runbookCM := approvalStoreConfigMaps(cr)
		args = append(args, "--store-backend="+backend)
		if cr.Spec.Approval.Store != nil && cr.Spec.Approval.Store.Namespace != "" {
			args = append(args, "--store-namespace="+cr.Spec.Approval.Store.Namespace)
		}
		args = append(args,
			"--store-replay-configmap="+replayCM,
			"--store-runbook-configmap="+runbookCM,
		)
	}

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{APIVersion: "apps/v1", Kind: "Deployment"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      ApprovalServerName(cr),
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Strategy: strategy,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					ServiceAccountName: ApprovalServerName(cr),
					ImagePullSecrets:   pullSecretRefs(approvalPullSecrets(cr)),
					Containers: []corev1.Container{
						{
							Name:            "approval-server",
							Image:           approvalImageRef(cr),
							ImagePullPolicy: approvalPullPolicy(cr),
							Args:            args,
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: approvalServerHTTPPort,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Env: []corev1.EnvVar{
								{Name: "CHA_SIGNING_KEY_PATH", Value: "/etc/cha/keys/signing.key"},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "signing-key",
									MountPath: "/etc/cha/keys",
									ReadOnly:  true,
								},
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/healthz",
										Port: intstr.FromString("http"),
									},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       30,
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/readyz",
										Port: intstr.FromString("http"),
									},
								},
								InitialDelaySeconds: 2,
								PeriodSeconds:       10,
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "signing-key",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: ApprovalSigningKeySecretName(cr),
									Items: []corev1.KeyToPath{
										{
											Key:  "signing.key",
											Path: "signing.key",
											Mode: int32Ptr(0o444),
										},
										{
											Key:  "signing.pub",
											Path: "signing.pub",
											Mode: int32Ptr(0o444),
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// BuildApprovalFixerClusterRole returns the shared mutation role the
// approval-server binds to. Verb set MIRRORS
// `charts/.../templates/clusterrole-remediator.yaml` — same fixer
// surface (no Secret, ConfigMap, or CRD writes).
//
// Returns the role unconditionally (no `enabled` gate) so the
// reconciler can provision it before any per-CR binding lands. The
// chart's tlsSecretMismatch-gated `networking.k8s.io/ingresses patch`
// rule is INCLUDED here unconditionally — TLSSecretMismatch is the
// only fixer that needs Ingress patch and gating it by spec adds
// complexity for a single verb.
func BuildApprovalFixerClusterRole() *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: ApprovalFixerClusterRoleName,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "cha-operator",
				"app.kubernetes.io/name":       "cluster-health-autopilot",
				"app.kubernetes.io/component":  "approval-fixer",
			},
		},
		Rules: []rbacv1.PolicyRule{
			{APIGroups: []string{""}, Resources: []string{"pods"}, Verbs: []string{"delete"}},
			{APIGroups: []string{"batch"}, Resources: []string{"jobs"}, Verbs: []string{"delete"}},
			{APIGroups: []string{"apps"}, Resources: []string{"deployments"}, Verbs: []string{"patch"}},
			{APIGroups: []string{"cert-manager.io"}, Resources: []string{"certificaterequests"}, Verbs: []string{"delete"}},
			{APIGroups: []string{"acme.cert-manager.io"}, Resources: []string{"orders"}, Verbs: []string{"delete"}},
			{APIGroups: []string{"networking.k8s.io"}, Resources: []string{"ingresses"}, Verbs: []string{"patch"}},
		},
	}
}

// BuildApprovalFixerClusterRoleBinding ties the approval-server SA to
// the shared fixer ClusterRole. Per-CR — cleaned up by the operator's
// finalizer.
func BuildApprovalFixerClusterRoleBinding(cr *chav1alpha1.ClusterHealthAutopilot) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: ApprovalFixerClusterRoleBindingName(cr),
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "cha-operator",
				"app.kubernetes.io/name":       "cluster-health-autopilot",
				"app.kubernetes.io/component":  "approval-fixer",
				ManagedByCRLabel:               cr.Name,
				ManagedByCRNamespaceLabel:      cr.Namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     ApprovalFixerClusterRoleName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      ApprovalServerName(cr),
				Namespace: cr.Namespace,
			},
		},
	}
}

// BuildApprovalSigningReaderRole returns the namespace-local Role
// allowing the approval-server to read its own signing key (and ONLY
// that one Secret — resourceNames-scoped). The watcher SA explicitly
// does NOT get this access.
func BuildApprovalSigningReaderRole(cr *chav1alpha1.ClusterHealthAutopilot) *rbacv1.Role {
	if !ApprovalEnabled(cr) {
		return nil
	}
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ApprovalServerName(cr) + "-signing-reader",
			Namespace: cr.Namespace,
			Labels:    CommonLabels(cr, "approval-server"),
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups:     []string{""},
				Resources:     []string{"secrets"},
				ResourceNames: []string{ApprovalSigningKeySecretName(cr)},
				Verbs:         []string{"get", "watch"},
			},
		},
	}
}

// BuildApprovalSigningReaderRoleBinding ties the SA to the signing
// reader Role.
func BuildApprovalSigningReaderRoleBinding(cr *chav1alpha1.ClusterHealthAutopilot) *rbacv1.RoleBinding {
	if !ApprovalEnabled(cr) {
		return nil
	}
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ApprovalServerName(cr) + "-signing-reader",
			Namespace: cr.Namespace,
			Labels:    CommonLabels(cr, "approval-server"),
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     ApprovalServerName(cr) + "-signing-reader",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      ApprovalServerName(cr),
				Namespace: cr.Namespace,
			},
		},
	}
}

// BuildApprovalEventsRole grants events emission for audit trail.
func BuildApprovalEventsRole(cr *chav1alpha1.ClusterHealthAutopilot) *rbacv1.Role {
	if !ApprovalEnabled(cr) {
		return nil
	}
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ApprovalServerName(cr) + "-events",
			Namespace: ApprovalAuditNamespace(cr),
			Labels:    CommonLabels(cr, "approval-server"),
		},
		Rules: []rbacv1.PolicyRule{
			{APIGroups: []string{""}, Resources: []string{"events"}, Verbs: []string{"create", "patch"}},
		},
	}
}

// BuildApprovalEventsRoleBinding ties the SA to the events Role.
func BuildApprovalEventsRoleBinding(cr *chav1alpha1.ClusterHealthAutopilot) *rbacv1.RoleBinding {
	if !ApprovalEnabled(cr) {
		return nil
	}
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ApprovalServerName(cr) + "-events",
			Namespace: ApprovalAuditNamespace(cr),
			Labels:    CommonLabels(cr, "approval-server"),
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     ApprovalServerName(cr) + "-events",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      ApprovalServerName(cr),
				Namespace: cr.Namespace,
			},
		},
	}
}

// BuildApprovalStoresRole grants RW on the replay + runbook ConfigMaps
// when Store.Backend == "configmap". Returns nil for the inmemory
// store (minimum-privilege posture — no extra ConfigMap access).
func BuildApprovalStoresRole(cr *chav1alpha1.ClusterHealthAutopilot) *rbacv1.Role {
	if !ApprovalEnabled(cr) || approvalStoreBackend(cr) != "configmap" {
		return nil
	}
	replayCM, runbookCM := approvalStoreConfigMaps(cr)
	storeNS := cr.Namespace
	if cr.Spec.Approval.Store != nil && cr.Spec.Approval.Store.Namespace != "" {
		storeNS = cr.Spec.Approval.Store.Namespace
	}
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ApprovalServerName(cr) + "-stores",
			Namespace: storeNS,
			Labels:    CommonLabels(cr, "approval-server"),
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups:     []string{""},
				Resources:     []string{"configmaps"},
				ResourceNames: []string{replayCM, runbookCM},
				Verbs:         []string{"get", "update"},
			},
			// `create` cannot be restricted by resourceName.
			{
				APIGroups: []string{""},
				Resources: []string{"configmaps"},
				Verbs:     []string{"create"},
			},
		},
	}
}

// BuildApprovalStoresRoleBinding ties the SA to the stores Role.
// Returns nil for the inmemory store.
func BuildApprovalStoresRoleBinding(cr *chav1alpha1.ClusterHealthAutopilot) *rbacv1.RoleBinding {
	if !ApprovalEnabled(cr) || approvalStoreBackend(cr) != "configmap" {
		return nil
	}
	storeNS := cr.Namespace
	if cr.Spec.Approval.Store != nil && cr.Spec.Approval.Store.Namespace != "" {
		storeNS = cr.Spec.Approval.Store.Namespace
	}
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ApprovalServerName(cr) + "-stores",
			Namespace: storeNS,
			Labels:    CommonLabels(cr, "approval-server"),
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     ApprovalServerName(cr) + "-stores",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      ApprovalServerName(cr),
				Namespace: cr.Namespace,
			},
		},
	}
}

// GenerateSigningKeySecret returns a Secret carrying a fresh Ed25519
// keypair (`signing.key` private + `signing.pub` public, both PEM).
// The operator calls this when the named Secret doesn't yet exist;
// idempotent across replicas via CAS in the create path (a second
// operator replica that finds the Secret already there reads it
// instead of regenerating, so JWT signatures stay verifiable).
//
// `randSource` is the source of randomness. Tests pass a
// deterministic reader for reproducibility; production uses
// crypto/rand.Reader. The crypto/ed25519 stdlib generates a 64-byte
// private key (the first 32 bytes are the seed) and a 32-byte public
// key.
func GenerateSigningKeySecret(cr *chav1alpha1.ClusterHealthAutopilot) (*corev1.Secret, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("ed25519 keygen: %w", err)
	}
	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: priv,
	})
	pubPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pub,
	})
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      ApprovalSigningKeySecretName(cr),
			Namespace: cr.Namespace,
			Labels:    CommonLabels(cr, "approval-server"),
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"signing.key": privPEM,
			"signing.pub": pubPEM,
		},
	}, nil
}

func int32Ptr(i int32) *int32 { return &i }

// ApprovalIngressEnabled reports whether spec.approval.ingress.enabled
// is true. Both `spec.approval` and `spec.approval.ingress` must be
// non-nil first.
func ApprovalIngressEnabled(cr *chav1alpha1.ClusterHealthAutopilot) bool {
	return ApprovalEnabled(cr) &&
		cr.Spec.Approval.Ingress != nil &&
		cr.Spec.Approval.Ingress.Enabled
}

// BuildApprovalServerIngress returns the public-facing Ingress for the
// approval-server, or nil when ingress is disabled. Mirrors the chart's
// `templates/approval-server-ingress.yaml`:
//   - Two paths: `/approve` (the SRE click endpoint) + `/healthz`
//     (so the upstream LB / cert-manager probe can validate the route).
//     Both route to the same approval-server Service on its `http`
//     port (8443 — the binary listens HTTP and lets the Ingress
//     terminate TLS).
//   - Optional TLS block with a Secret reference.
//   - User-provided annotations pass through verbatim
//     (cert-manager, oauth2-proxy, kong, etc.).
//
// The Host field is required at the reconciler door (validated there);
// the builder assumes a non-empty value.
func BuildApprovalServerIngress(cr *chav1alpha1.ClusterHealthAutopilot) *networkingv1.Ingress {
	if !ApprovalIngressEnabled(cr) {
		return nil
	}
	ing := cr.Spec.Approval.Ingress
	labels := CommonLabels(cr, "approval-server")

	pathType := networkingv1.PathTypePrefix
	rules := []networkingv1.IngressRule{
		{
			Host: ing.Host,
			IngressRuleValue: networkingv1.IngressRuleValue{
				HTTP: &networkingv1.HTTPIngressRuleValue{
					Paths: []networkingv1.HTTPIngressPath{
						{
							Path:     "/approve",
							PathType: &pathType,
							Backend: networkingv1.IngressBackend{
								Service: &networkingv1.IngressServiceBackend{
									Name: ApprovalServerName(cr),
									Port: networkingv1.ServiceBackendPort{Name: "http"},
								},
							},
						},
						{
							Path:     "/healthz",
							PathType: &pathType,
							Backend: networkingv1.IngressBackend{
								Service: &networkingv1.IngressServiceBackend{
									Name: ApprovalServerName(cr),
									Port: networkingv1.ServiceBackendPort{Name: "http"},
								},
							},
						},
					},
				},
			},
		},
	}

	out := &networkingv1.Ingress{
		TypeMeta: metav1.TypeMeta{APIVersion: "networking.k8s.io/v1", Kind: "Ingress"},
		ObjectMeta: metav1.ObjectMeta{
			Name:        ApprovalServerName(cr),
			Namespace:   cr.Namespace,
			Labels:      labels,
			Annotations: ing.Annotations,
		},
		Spec: networkingv1.IngressSpec{
			Rules: rules,
		},
	}
	if ing.IngressClassName != "" {
		ic := ing.IngressClassName
		out.Spec.IngressClassName = &ic
	}
	if ing.TLS != nil && ing.TLS.Enabled {
		secretName := ing.TLS.SecretName
		if secretName == "" {
			secretName = ApprovalServerName(cr) + "-tls"
		}
		out.Spec.TLS = []networkingv1.IngressTLS{
			{Hosts: []string{ing.Host}, SecretName: secretName},
		}
	}
	return out
}
