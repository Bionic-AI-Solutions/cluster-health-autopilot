// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package fix

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// gitopsResource is a minimal helper that builds an unstructured Deployment-
// shaped object with the requested annotations / labels / spec for the
// GitOps helper tests. It is intentionally generic — the helper must work on
// any kind, not just Ingress.
func gitopsResource(kind string, annotations, labels map[string]string, spec map[string]any) unstructured.Unstructured {
	obj := map[string]any{
		"apiVersion": "apps/v1",
		"kind":       kind,
		"metadata": map[string]any{
			"name":      "x",
			"namespace": "demo",
		},
	}
	if len(annotations) > 0 {
		md := obj["metadata"].(map[string]any)
		anns := make(map[string]any, len(annotations))
		for k, v := range annotations {
			anns[k] = v
		}
		md["annotations"] = anns
	}
	if len(labels) > 0 {
		md := obj["metadata"].(map[string]any)
		lbls := make(map[string]any, len(labels))
		for k, v := range labels {
			lbls[k] = v
		}
		md["labels"] = lbls
	}
	if spec != nil {
		obj["spec"] = spec
	}
	return unstructured.Unstructured{Object: obj}
}

func TestGitOpsReason_ArgoInstance(t *testing.T) {
	u := gitopsResource("Deployment",
		map[string]string{"argocd.argoproj.io/instance": "billing-app"},
		nil, nil)
	got := GitOpsReason(u)
	if got == "" {
		t.Fatalf("expected non-empty reason for argocd-managed Deployment; got %q", got)
	}
	if !contains(got, "argocd.argoproj.io/instance") {
		t.Errorf("reason should name the annotation; got %q", got)
	}
}

func TestGitOpsReason_ArgoTrackingID(t *testing.T) {
	u := gitopsResource("Deployment",
		map[string]string{"argocd.argoproj.io/tracking-id": "billing-app:apps/v1:Deployment/demo/x"},
		nil, nil)
	if GitOpsReason(u) == "" {
		t.Fatalf("argocd tracking-id should mark resource as managed")
	}
}

func TestGitOpsReason_FluxKustomizeLabel(t *testing.T) {
	u := gitopsResource("Deployment", nil,
		map[string]string{"kustomize.toolkit.fluxcd.io/name": "apps-kustomization"}, nil)
	got := GitOpsReason(u)
	if got == "" {
		t.Fatalf("flux kustomize label should mark resource as managed")
	}
}

func TestGitOpsReason_HelmManagedBy(t *testing.T) {
	for _, v := range []string{"Helm", "helm", "HELM"} {
		u := gitopsResource("Deployment", nil,
			map[string]string{"app.kubernetes.io/managed-by": v}, nil)
		got := GitOpsReason(u)
		if got == "" {
			t.Errorf("managed-by=%q should be detected (case-insensitive); got empty", v)
		}
	}
}

func TestGitOpsReason_ArgocdManagedBy(t *testing.T) {
	u := gitopsResource("Deployment", nil,
		map[string]string{"app.kubernetes.io/managed-by": "argocd"}, nil)
	if GitOpsReason(u) == "" {
		t.Fatalf("managed-by=argocd should be detected")
	}
}

func TestGitOpsReason_FluxManagedBy(t *testing.T) {
	for _, v := range []string{"flux", "fluxcd"} {
		u := gitopsResource("Deployment", nil,
			map[string]string{"app.kubernetes.io/managed-by": v}, nil)
		if GitOpsReason(u) == "" {
			t.Errorf("managed-by=%q should be detected", v)
		}
	}
}

func TestGitOpsReason_PlainResource(t *testing.T) {
	u := gitopsResource("Deployment", nil, nil, nil)
	if got := GitOpsReason(u); got != "" {
		t.Errorf("plain Deployment should not be flagged as GitOps-managed; got %q", got)
	}
}

func TestGitOpsReason_EmptyAnnotationValue(t *testing.T) {
	// argocd annotation present but value is empty string → not actually managed.
	u := gitopsResource("Deployment",
		map[string]string{"argocd.argoproj.io/instance": ""}, nil, nil)
	if got := GitOpsReason(u); got != "" {
		t.Errorf("empty annotation value should not be flagged; got %q", got)
	}
}

func TestGitOpsReason_UnknownManagedByValue(t *testing.T) {
	// managed-by label with a non-GitOps value (e.g. a custom controller).
	u := gitopsResource("Deployment", nil,
		map[string]string{"app.kubernetes.io/managed-by": "my-custom-operator"}, nil)
	if got := GitOpsReason(u); got != "" {
		t.Errorf("unknown managed-by value should not be flagged; got %q", got)
	}
}

func TestGitOpsReason_WorksOnAnyKind(t *testing.T) {
	// The Ingress-only private helper was Ingress-typed; the public helper
	// must accept any resource kind.
	for _, kind := range []string{"StatefulSet", "DaemonSet", "Job", "CronJob", "Ingress"} {
		u := gitopsResource(kind,
			map[string]string{"argocd.argoproj.io/instance": "foo"}, nil, nil)
		if got := GitOpsReason(u); got == "" {
			t.Errorf("kind=%s with argocd annotation should be flagged; got empty", kind)
		}
	}
}

func TestIsPaused_True(t *testing.T) {
	u := gitopsResource("Deployment", nil, nil,
		map[string]any{"paused": true})
	if !IsPaused(u) {
		t.Errorf("Deployment with spec.paused=true should be paused")
	}
}

func TestIsPaused_False(t *testing.T) {
	u := gitopsResource("Deployment", nil, nil,
		map[string]any{"paused": false})
	if IsPaused(u) {
		t.Errorf("Deployment with spec.paused=false should not be paused")
	}
}

func TestIsPaused_AbsentField(t *testing.T) {
	u := gitopsResource("Deployment", nil, nil,
		map[string]any{"replicas": int64(3)})
	if IsPaused(u) {
		t.Errorf("Deployment without spec.paused should not be paused")
	}
}

func TestIsPaused_NoSpec(t *testing.T) {
	u := gitopsResource("Deployment", nil, nil, nil)
	if IsPaused(u) {
		t.Errorf("resource without spec should not be paused")
	}
}

func TestIsPaused_StatefulSetNoFalsePositive(t *testing.T) {
	// StatefulSet doesn't have spec.paused; helper must not panic and must
	// return false regardless of any other spec fields.
	u := gitopsResource("StatefulSet", nil, nil,
		map[string]any{"serviceName": "my-svc"})
	if IsPaused(u) {
		t.Errorf("StatefulSet should not be reported as paused")
	}
}

func TestIsSuspended_True(t *testing.T) {
	u := gitopsResource("CronJob", nil, nil,
		map[string]any{"suspend": true})
	if !IsSuspended(u) {
		t.Errorf("CronJob with spec.suspend=true should be suspended")
	}
}

func TestIsSuspended_False(t *testing.T) {
	u := gitopsResource("CronJob", nil, nil,
		map[string]any{"suspend": false})
	if IsSuspended(u) {
		t.Errorf("CronJob with spec.suspend=false should not be suspended")
	}
}

func TestIsSuspended_AbsentField(t *testing.T) {
	u := gitopsResource("CronJob", nil, nil,
		map[string]any{"schedule": "*/5 * * * *"})
	if IsSuspended(u) {
		t.Errorf("CronJob without spec.suspend should not be suspended")
	}
}

func TestIsSuspended_NonCronJobNoFalsePositive(t *testing.T) {
	// Deployment doesn't have spec.suspend; helper must not panic.
	u := gitopsResource("Deployment", nil, nil,
		map[string]any{"replicas": int64(3)})
	if IsSuspended(u) {
		t.Errorf("Deployment without spec.suspend should not be suspended")
	}
}

// contains is a tiny helper to keep tests free of strings.Contains imports.
func contains(s, sub string) bool {
	return len(s) >= len(sub) && indexOf(s, sub) >= 0
}

func indexOf(s, sub string) int {
	if sub == "" {
		return 0
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
