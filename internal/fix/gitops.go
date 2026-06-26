// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package fix

import (
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// GitOpsAnnotations are the annotation keys that signal a resource is
// reconciled by a GitOps controller (Argo CD, Flux, Helm). If any of these
// annotations is present with a non-empty value, fixers should refuse to
// mutate — the edit belongs in the source repo, and mutating in-cluster
// will be reverted on the next reconcile, putting Srenix and the controller
// into a tight fight loop.
//
// This list is intentionally conservative. False negatives (controller we
// don't recognise) are recoverable by the user; false positives (we refuse
// to fix a non-GitOps resource) are silent and frustrating.
var GitOpsAnnotations = []string{
	"argocd.argoproj.io/instance",
	"argocd.argoproj.io/tracking-id",
	"kustomize.toolkit.fluxcd.io/name",
	"kustomize.toolkit.fluxcd.io/namespace",
	"meta.helm.sh/release-name",
	"meta.helm.sh/release-namespace",
}

// gitOpsLabels parallel GitOpsAnnotations for label-based identification.
// The app.kubernetes.io/managed-by label is the common cross-controller
// signal; we only flag it when the value matches a known GitOps tool.
var gitOpsLabels = []string{
	"app.kubernetes.io/managed-by",
	"kustomize.toolkit.fluxcd.io/name", // some Flux versions label rather than annotate
}

// knownGitOpsManagedByValues is the closed set of app.kubernetes.io/managed-by
// label values we treat as "do not touch". Case-insensitive. Any other value
// (e.g. "my-custom-operator") is NOT flagged — operators have their own
// reconciliation, but a wide net here causes too many false positives.
var knownGitOpsManagedByValues = map[string]struct{}{
	"helm":    {},
	"argocd":  {},
	"flux":    {},
	"fluxcd":  {},
	"argo-cd": {},
}

// GitOpsReason returns a human-readable reason string when the given
// resource is reconciled by a known GitOps controller, or empty when Srenix
// may safely mutate it. The reason is intended for log lines and Skipped
// entries on the fixer Result, not for parsing — its exact text may evolve.
//
// Works on any Kubernetes kind (Deployment, StatefulSet, Ingress, etc.).
func GitOpsReason(u unstructured.Unstructured) string {
	annotations := u.GetAnnotations()
	for _, k := range GitOpsAnnotations {
		if v, ok := annotations[k]; ok && v != "" {
			return k + "=" + v
		}
	}
	labels := u.GetLabels()
	for _, k := range gitOpsLabels {
		v, ok := labels[k]
		if !ok || v == "" {
			continue
		}
		if k == "app.kubernetes.io/managed-by" {
			if _, known := knownGitOpsManagedByValues[strings.ToLower(v)]; !known {
				continue
			}
		}
		return k + "=" + v
	}
	return ""
}

// IsPaused reports whether a Deployment-shaped resource has spec.paused set
// to true. Returns false for any other kind (StatefulSet, DaemonSet, etc.)
// and for Deployments where the field is absent or false.
//
// Srenix fixers consult this before rolling-restarting a Deployment: a paused
// rollout means an operator has deliberately frozen updates, and forcing a
// restart violates that intent.
func IsPaused(u unstructured.Unstructured) bool {
	paused, found, err := unstructured.NestedBool(u.Object, "spec", "paused")
	if err != nil || !found {
		return false
	}
	return paused
}

// IsSuspended reports whether a CronJob has spec.suspend set to true.
// Returns false when the field is absent or false. Semantically parallel to
// IsPaused for Deployments — an operator's deliberate "freeze this resource"
// signal that fixers must honor.
func IsSuspended(u unstructured.Unstructured) bool {
	suspended, found, err := unstructured.NestedBool(u.Object, "spec", "suspend")
	if err != nil || !found {
		return false
	}
	return suspended
}
