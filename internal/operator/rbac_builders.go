// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	chav1alpha1 "github.com/Bionic-AI-Solutions/cluster-health-autopilot/api/v1alpha1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Phase 1c — operator-provisioned reader RBAC for the watcher SA.
//
// The controller creates two RBAC objects:
//
//  1. ONE shared cluster-scoped `ClusterRole` (`ReaderClusterRoleName`)
//     carrying the same verb set as the chart's
//     `templates/clusterrole-reader.yaml`. Idempotent across all CRs;
//     NOT owned by any CR (cluster-scoped resources can't own from a
//     namespaced parent, and the role survives CR delete on purpose).
//
//  2. ONE per-CR `ClusterRoleBinding` linking the CR's ServiceAccount
//     to the shared ClusterRole. Cleaned up by the operator's
//     finalizer when the CR is deleted (cluster-scoped → no GC).
//
// Side-by-side coexistence with chart-managed installs is the design
// contract: an operator-managed CR can land in a cluster that already
// has the chart's reader binding without disturbing it. RBAC bindings
// union across subjects, so a SA bound by BOTH the chart and the
// operator simply has the union of both verb sets (which are equal,
// by design).

// ReaderClusterRoleName is the cluster-scoped, shared reader role's
// name. Single role per cluster regardless of how many CRs exist —
// every binding points here.
const ReaderClusterRoleName = "cha-operator-watcher-reader"

// ManagedByCRLabel + ManagedByCRNamespaceLabel mark the per-CR
// ClusterRoleBinding so the finalizer can find it without depending
// on a fragile name pattern. Also a defense-in-depth signal: the
// operator only ever deletes RBAC objects it labeled itself.
const (
	ManagedByCRLabel          = "cha.bionicaisolutions.com/managed-by-cr"
	ManagedByCRNamespaceLabel = "cha.bionicaisolutions.com/cr-namespace"
)

// ReaderClusterRoleBindingName returns the per-CR binding name. The
// `<ns>-<name>` shape keeps the name globally unique across CRs in
// any namespace and trivially derivable for the finalizer.
func ReaderClusterRoleBindingName(cr *chav1alpha1.ClusterHealthAutopilot) string {
	return "cha-operator-watcher-" + cr.Namespace + "-" + cr.Name
}

// BuildReaderClusterRole returns the shared cluster-scoped reader
// role. Verb set MIRRORS `charts/.../templates/clusterrole-reader.yaml`
// — any divergence between the two is a regression worth a CI gate
// in a future slice.
//
// Pure: no CR input, no cluster reads.
func BuildReaderClusterRole() *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: ReaderClusterRoleName,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "cha-operator",
				"app.kubernetes.io/name":       "cluster-health-autopilot",
				"app.kubernetes.io/component":  "reader",
			},
		},
		Rules: readerPolicyRules(),
	}
}

// BuildReaderClusterRoleBinding returns the per-CR binding that maps
// the CR's ServiceAccount to the shared reader ClusterRole. The
// binding labels carry the back-pointer the finalizer uses to find
// + delete this binding when the CR is GC'd.
func BuildReaderClusterRoleBinding(cr *chav1alpha1.ClusterHealthAutopilot) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: ReaderClusterRoleBindingName(cr),
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "cha-operator",
				"app.kubernetes.io/name":       "cluster-health-autopilot",
				"app.kubernetes.io/component":  "reader",
				ManagedByCRLabel:               cr.Name,
				ManagedByCRNamespaceLabel:      cr.Namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     ReaderClusterRoleName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      ServiceAccountNameFor(cr),
				Namespace: cr.Namespace,
			},
		},
	}
}

// readerPolicyRules returns the verb set in dependency-stable order so
// `helm template` and `BuildReaderClusterRole()` produce byte-equal
// diffs for the same logical role. Anything added here MUST also be
// added to `templates/clusterrole-reader.yaml` and vice versa.
func readerPolicyRules() []rbacv1.PolicyRule {
	return []rbacv1.PolicyRule{
		// Core probe surface (pods/nodes/PVCs/events/namespaces).
		// Services + endpoints needed by the DNS-chain drift analyzer
		// to walk Ingress → Service → Endpoints chains. Without them
		// every Ingress shows up as orphan-service (the analyzer's
		// src.Get(GVRService) returns 403, which it treats as
		// "doesn't exist"). v1.10.3 regression — caught against the
		// live Bionic cluster where ~10 healthy Ingresses were
		// incorrectly flagged.
		{
			APIGroups: []string{""},
			Resources: []string{"pods", "nodes", "persistentvolumeclaims", "events", "namespaces", "services", "endpoints"},
			Verbs:     []string{"get", "list", "watch"},
		},
		// Secrets — names + keys only (ProactiveSecretKeyCheck doesn't
		// read values). Matches the chart's intent verbatim.
		{
			APIGroups: []string{""},
			Resources: []string{"secrets"},
			Verbs:     []string{"get", "list", "watch"},
		},
		// Workload owner-chain walkers.
		{
			APIGroups: []string{"apps"},
			Resources: []string{"deployments", "replicasets", "statefulsets", "daemonsets"},
			Verbs:     []string{"get", "list", "watch"},
		},
		{
			APIGroups: []string{"batch"},
			Resources: []string{"jobs", "cronjobs"},
			Verbs:     []string{"get", "list", "watch"},
		},
		// external-secrets.
		{
			APIGroups: []string{"external-secrets.io"},
			Resources: []string{"externalsecrets", "secretstores", "clustersecretstores"},
			Verbs:     []string{"get", "list", "watch"},
		},
		// cert-manager.
		{
			APIGroups: []string{"cert-manager.io"},
			Resources: []string{"certificates", "certificaterequests"},
			Verbs:     []string{"get", "list", "watch"},
		},
		{
			APIGroups: []string{"acme.cert-manager.io"},
			Resources: []string{"orders", "challenges"},
			Verbs:     []string{"get", "list", "watch"},
		},
		// GitOpsDrift — Argo CD + Flux.
		{
			APIGroups: []string{"argoproj.io"},
			Resources: []string{"applications"},
			Verbs:     []string{"get", "list", "watch"},
		},
		{
			APIGroups: []string{"kustomize.toolkit.fluxcd.io"},
			Resources: []string{"kustomizations"},
			Verbs:     []string{"get", "list", "watch"},
		},
		{
			APIGroups: []string{"helm.toolkit.fluxcd.io"},
			Resources: []string{"helmreleases"},
			Verbs:     []string{"get", "list", "watch"},
		},
		// RBACDrift — roles + bindings + SAs cluster-wide, read-only.
		{
			APIGroups: []string{"rbac.authorization.k8s.io"},
			Resources: []string{"roles", "rolebindings", "clusterroles", "clusterrolebindings"},
			Verbs:     []string{"get", "list", "watch"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"serviceaccounts"},
			Verbs:     []string{"get", "list", "watch"},
		},
		// ConfigDrift — CRDs.
		{
			APIGroups: []string{"apiextensions.k8s.io"},
			Resources: []string{"customresourcedefinitions"},
			Verbs:     []string{"get", "list", "watch"},
		},
		// CapacityDrift — HPAs.
		{
			APIGroups: []string{"autoscaling"},
			Resources: []string{"horizontalpodautoscalers"},
			Verbs:     []string{"get", "list", "watch"},
		},
		// IngressCoverage + SecurityDrift.
		{
			APIGroups: []string{"networking.k8s.io"},
			Resources: []string{"ingresses", "networkpolicies"},
			Verbs:     []string{"get", "list", "watch"},
		},
		// CloudNativePG.
		{
			APIGroups: []string{"postgresql.cnpg.io"},
			Resources: []string{"clusters"},
			Verbs:     []string{"get", "list", "watch"},
		},
		// rook-ceph.
		{
			APIGroups: []string{"ceph.rook.io"},
			Resources: []string{"cephclusters"},
			Verbs:     []string{"get", "list", "watch"},
		},
		// M2 probe-class CRD reads (Kong / Velero) — no-ops when CRDs absent.
		{
			APIGroups: []string{"configuration.konghq.com"},
			Resources: []string{"kongplugins"},
			Verbs:     []string{"get", "list", "watch"},
		},
		{
			APIGroups: []string{"velero.io"},
			Resources: []string{"backups"},
			Verbs:     []string{"get", "list", "watch"},
		},
		// CHA's own CRDs the watcher reads.
		{
			APIGroups: []string{"cha.bionicaisolutions.com"},
			Resources: []string{"driftreports", "resolutionrecords", "silences"},
			Verbs:     []string{"get", "list", "watch"},
		},
		// Leader-election: the watcher binary uses controller-runtime's
		// lease-based leader election so multi-replica installs don't
		// double-fire. Without these verbs, a single-replica install
		// works (loop continues; lease acquisition is non-fatal) but
		// the watcher logs a 461-line "cannot get leases" error every
		// 5–10s. Verbs scoped to coordination.k8s.io only — these
		// don't expand the probe surface, they're for the watcher's
		// own internal coordination.
		{
			APIGroups: []string{"coordination.k8s.io"},
			Resources: []string{"leases"},
			Verbs:     []string{"get", "list", "watch", "create", "update", "patch"},
		},
	}
}
