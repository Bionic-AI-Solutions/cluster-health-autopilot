// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package ai

import "strings"

// gitOpsAnnotationKeys / gitOpsLabelKeys mark a resource as reconciled by a
// GitOps controller (Argo CD or Flux). A DIRECT live patch on such a resource
// is futile — the controller reverts it on the next sync — so the AI tier must
// route the fix to a pull request against the source repo instead of patching
// the cluster. Helm's managed-by label is deliberately NOT included: a
// Helm-installed resource is not necessarily GitOps-reconciled.
var gitOpsAnnotationKeys = []string{
	"argocd.argoproj.io/tracking-id",
	"argocd.argoproj.io/sync-wave",
	"kustomize.toolkit.fluxcd.io/name",
	"helm.toolkit.fluxcd.io/name",
	"fluxcd.io/sync-checksum",
}

var gitOpsLabelKeys = []string{
	"argocd.argoproj.io/instance",
	"kustomize.toolkit.fluxcd.io/name",
	"helm.toolkit.fluxcd.io/name",
	"app.kubernetes.io/instance", // Argo's default tracking label
}

// IsGitOpsManaged reports whether a workload's labels/annotations indicate it
// is reconciled by Argo CD or Flux. Used by the fix proposer to choose between
// a direct patch (not GitOps → safe to apply live) and a pull request (GitOps →
// patch would drift/revert, so propose the change in source).
func IsGitOpsManaged(labels, annotations map[string]string) bool {
	for _, k := range gitOpsAnnotationKeys {
		if v, ok := annotations[k]; ok && strings.TrimSpace(v) != "" {
			return true
		}
	}
	for _, k := range gitOpsLabelKeys {
		if v, ok := labels[k]; ok && strings.TrimSpace(v) != "" {
			return true
		}
	}
	return false
}
