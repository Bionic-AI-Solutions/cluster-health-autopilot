// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package diagnose

import (
	"context"
	"strings"
	"testing"
	"time"

	pkgsnapshot "github.com/srenix-ai/agentic-sre/pkg/snapshot"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// memSourceGitOps is a minimal in-memory snapshot.Source for these
// tests. Keyed by Resource name (the lowercase plural of the kind).
type memSourceGitOps struct {
	byResource map[string][]unstructured.Unstructured
}

func (m *memSourceGitOps) List(_ context.Context, gvr schema.GroupVersionResource, ns string) (*unstructured.UnstructuredList, error) {
	out := &unstructured.UnstructuredList{}
	for _, u := range m.byResource[gvr.Resource] {
		if ns != "" && u.GetNamespace() != ns {
			continue
		}
		out.Items = append(out.Items, u)
	}
	return out, nil
}

func (m *memSourceGitOps) Get(_ context.Context, gvr schema.GroupVersionResource, ns, name string) (*unstructured.Unstructured, error) {
	for _, u := range m.byResource[gvr.Resource] {
		if u.GetNamespace() == ns && u.GetName() == name {
			return &u, nil
		}
	}
	return nil, nil
}

func (m *memSourceGitOps) Mode() pkgsnapshot.Mode { return pkgsnapshot.ModeLive }

// argoApp builds a synthetic Argo Application with a status block.
func argoApp(ns, name, syncStatus, healthStatus, revision string, reconciledAtAgo time.Duration) unstructured.Unstructured {
	u := unstructured.Unstructured{}
	u.SetAPIVersion("argoproj.io/v1alpha1")
	u.SetKind("Application")
	u.SetNamespace(ns)
	u.SetName(name)
	reconciledAt := time.Now().Add(-reconciledAtAgo).UTC().Format(time.RFC3339)
	_ = unstructured.SetNestedField(u.Object, syncStatus, "status", "sync", "status")
	_ = unstructured.SetNestedField(u.Object, revision, "status", "sync", "revision")
	_ = unstructured.SetNestedField(u.Object, healthStatus, "status", "health", "status")
	_ = unstructured.SetNestedField(u.Object, reconciledAt, "status", "reconciledAt")
	return u
}

// fluxKustomization builds a synthetic Flux Kustomization with a
// Ready condition. transitionAgo controls lastTransitionTime relative
// to now; condStatus is "True"/"False"; reason is e.g. "ReconciliationSucceeded".
func fluxKustomization(ns, name, condStatus, reason, message string, transitionAgo time.Duration) unstructured.Unstructured {
	return fluxResource("kustomize.toolkit.fluxcd.io/v1", "Kustomization", ns, name, condStatus, reason, message, transitionAgo)
}

func fluxHelmRelease(ns, name, condStatus, reason, message string, transitionAgo time.Duration) unstructured.Unstructured {
	return fluxResource("helm.toolkit.fluxcd.io/v2", "HelmRelease", ns, name, condStatus, reason, message, transitionAgo)
}

func fluxResource(apiVersion, kind, ns, name, condStatus, reason, message string, transitionAgo time.Duration) unstructured.Unstructured {
	u := unstructured.Unstructured{}
	u.SetAPIVersion(apiVersion)
	u.SetKind(kind)
	u.SetNamespace(ns)
	u.SetName(name)
	u.SetCreationTimestamp(metav1.NewTime(time.Now().Add(-2 * 24 * time.Hour)))
	transitionAt := time.Now().Add(-transitionAgo).UTC().Format(time.RFC3339)
	cond := map[string]interface{}{
		"type":               "Ready",
		"status":             condStatus,
		"reason":             reason,
		"message":            message,
		"lastTransitionTime": transitionAt,
	}
	_ = unstructured.SetNestedSlice(u.Object, []interface{}{cond}, "status", "conditions")
	return u
}

// --- Argo Application tests --------------------------------------------------

func TestGitOpsDrift_ArgoSynced_NoDiagnostic(t *testing.T) {
	src := &memSourceGitOps{byResource: map[string][]unstructured.Unstructured{
		"applications": {argoApp("argo", "billing-svc", "Synced", "Healthy", "abc1234567def", 30*time.Minute)},
	}}
	got := GitOpsDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("Synced+Healthy app should produce no diagnostics; got %+v", got)
	}
}

func TestGitOpsDrift_ArgoOutOfSync_PastGrace_Warning(t *testing.T) {
	src := &memSourceGitOps{byResource: map[string][]unstructured.Unstructured{
		"applications": {argoApp("argo", "billing-svc", "OutOfSync", "Healthy", "abc1234567def", 30*time.Minute)},
	}}
	got := GitOpsDrift{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("OutOfSync past grace should emit 1 diagnostic; got %d: %+v", len(got), got)
	}
	d := got[0]
	if d.Severity != "warning" {
		t.Errorf("OutOfSync sync should be warning; got %q", d.Severity)
	}
	if !strings.Contains(d.Message, "OutOfSync") || !strings.Contains(d.Message, "abc1234567") {
		t.Errorf("diagnostic should name status + revision; got %q", d.Message)
	}
	if !strings.Contains(d.Subject, "Application/argo/billing-svc") {
		t.Errorf("diagnostic should name the Application subject; got %q", d.Subject)
	}
	if !strings.Contains(d.Remediation, "argocd app sync") {
		t.Errorf("diagnostic remediation should mention `argocd app sync`; got %q", d.Remediation)
	}
}

func TestGitOpsDrift_ArgoOutOfSync_InsideGrace_Suppressed(t *testing.T) {
	src := &memSourceGitOps{byResource: map[string][]unstructured.Unstructured{
		// 2 minutes < 10-min default grace.
		"applications": {argoApp("argo", "billing-svc", "OutOfSync", "Healthy", "abc1234", 2*time.Minute)},
	}}
	got := GitOpsDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("OutOfSync inside grace should be suppressed; got %+v", got)
	}
}

func TestGitOpsDrift_ArgoDegraded_Critical(t *testing.T) {
	// Synced sync but Degraded health → critical-severity health diagnostic.
	src := &memSourceGitOps{byResource: map[string][]unstructured.Unstructured{
		"applications": {argoApp("argo", "billing-svc", "Synced", "Degraded", "abc1234", 30*time.Minute)},
	}}
	got := GitOpsDrift{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("Degraded health past grace should emit 1 diagnostic; got %d: %+v", len(got), got)
	}
	d := got[0]
	if d.Severity != "critical" {
		t.Errorf("Degraded health should be critical; got %q", d.Severity)
	}
	if !strings.Contains(d.Message, "health=Degraded") {
		t.Errorf("diagnostic should name health status; got %q", d.Message)
	}
}

func TestGitOpsDrift_ArgoBoth_OutOfSyncAndDegraded_TwoDiagnostics(t *testing.T) {
	src := &memSourceGitOps{byResource: map[string][]unstructured.Unstructured{
		"applications": {argoApp("argo", "billing-svc", "OutOfSync", "Degraded", "abc1234", 30*time.Minute)},
	}}
	got := GitOpsDrift{}.Run(context.Background(), src)
	if len(got) != 2 {
		t.Fatalf("Sync + Health both bad should produce 2 diagnostics; got %d: %+v", len(got), got)
	}
}

func TestGitOpsDrift_ArgoProgressing_NoDiagnostic(t *testing.T) {
	// Progressing is a transient state; not a drift signal.
	src := &memSourceGitOps{byResource: map[string][]unstructured.Unstructured{
		"applications": {argoApp("argo", "billing-svc", "Synced", "Progressing", "abc1234", 30*time.Minute)},
	}}
	got := GitOpsDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("Progressing should not emit a diagnostic; got %+v", got)
	}
}

// --- Flux Kustomization tests ------------------------------------------------

func TestGitOpsDrift_FluxKsReady_NoDiagnostic(t *testing.T) {
	src := &memSourceGitOps{byResource: map[string][]unstructured.Unstructured{
		"kustomizations": {fluxKustomization("flux-system", "platform", "True", "ReconciliationSucceeded", "Applied revision abc", 30*time.Minute)},
	}}
	got := GitOpsDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("Ready=True Kustomization should be silent; got %+v", got)
	}
}

func TestGitOpsDrift_FluxKsNotReady_NonFailedReason_Warning(t *testing.T) {
	// DependencyNotReady is the canonical Flux "still waiting on something"
	// reason — it's a NotReady state, but not catastrophic, so default warning.
	src := &memSourceGitOps{byResource: map[string][]unstructured.Unstructured{
		"kustomizations": {fluxKustomization("flux-system", "platform", "False", "DependencyNotReady",
			"depends on flux-system/sources but its Ready condition is not True",
			30*time.Minute)},
	}}
	got := GitOpsDrift{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("Ready=False past grace should emit 1 diagnostic; got %d: %+v", len(got), got)
	}
	d := got[0]
	if d.Severity != "warning" {
		t.Errorf("non-Failed reason should be warning; got %q", d.Severity)
	}
	if !strings.Contains(d.Message, "DependencyNotReady") {
		t.Errorf("diagnostic should surface the reason; got %q", d.Message)
	}
}

func TestGitOpsDrift_FluxKsBuildFailed_Critical(t *testing.T) {
	// BuildFailed = kustomize can't even render the manifests; cluster is
	// silently drifting from git. Escalates to critical.
	src := &memSourceGitOps{byResource: map[string][]unstructured.Unstructured{
		"kustomizations": {fluxKustomization("flux-system", "platform", "False", "BuildFailed",
			"kustomize build failed: accumulating resources: rpc error: code = NotFound",
			30*time.Minute)},
	}}
	got := GitOpsDrift{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("BuildFailed past grace should emit 1 diagnostic; got %d: %+v", len(got), got)
	}
	if got[0].Severity != "critical" {
		t.Errorf("BuildFailed reason should be critical; got %q", got[0].Severity)
	}
	if !strings.Contains(got[0].Message, "kustomize build failed") {
		t.Errorf("diagnostic should surface the controller error; got %q", got[0].Message)
	}
}

func TestGitOpsDrift_FluxKsNotReady_InsideGrace_Suppressed(t *testing.T) {
	src := &memSourceGitOps{byResource: map[string][]unstructured.Unstructured{
		"kustomizations": {fluxKustomization("flux-system", "platform", "False", "Reconciling", "applying", 1*time.Minute)},
	}}
	got := GitOpsDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("Ready=False inside grace should be suppressed; got %+v", got)
	}
}

// --- Flux HelmRelease tests --------------------------------------------------

func TestGitOpsDrift_FluxHRUpgradeFailed_Critical(t *testing.T) {
	// reason containing "Failed" → critical
	src := &memSourceGitOps{byResource: map[string][]unstructured.Unstructured{
		"helmreleases": {fluxHelmRelease("monitoring", "prometheus", "False", "UpgradeFailed",
			"Helm upgrade failed: timed out waiting for the condition", 30*time.Minute)},
	}}
	got := GitOpsDrift{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("UpgradeFailed past grace should emit 1 diagnostic; got %d", len(got))
	}
	d := got[0]
	if d.Severity != "critical" {
		t.Errorf("UpgradeFailed should be critical; got %q", d.Severity)
	}
	if !strings.Contains(d.Subject, "HelmRelease/monitoring/prometheus") {
		t.Errorf("diagnostic should name the HelmRelease; got %q", d.Subject)
	}
}

func TestGitOpsDrift_FluxHRInstallFailed_Critical(t *testing.T) {
	src := &memSourceGitOps{byResource: map[string][]unstructured.Unstructured{
		"helmreleases": {fluxHelmRelease("ai", "vllm", "False", "InstallFailed",
			"Helm install failed: pre-upgrade hooks failed", 30*time.Minute)},
	}}
	got := GitOpsDrift{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("InstallFailed past grace should emit 1 diagnostic; got %d", len(got))
	}
	if got[0].Severity != "critical" {
		t.Errorf("InstallFailed should be critical; got %q", got[0].Severity)
	}
}

// --- multi-resource + edge cases --------------------------------------------

func TestGitOpsDrift_NoCRDs_NoOp(t *testing.T) {
	// Cluster without Argo / Flux installed — source returns empty
	// lists. Analyzer must be silent, not error.
	src := &memSourceGitOps{byResource: map[string][]unstructured.Unstructured{}}
	got := GitOpsDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("empty cluster should produce 0 diagnostics; got %+v", got)
	}
}

func TestGitOpsDrift_TruncatesLongMessage(t *testing.T) {
	// Surface a controller error >200 chars and verify it's truncated
	// so the Slack post stays readable. Full message remains on the
	// CR for kubectl describe.
	longMsg := strings.Repeat("kustomize build failed: accumulating resources: rpc error: code = NotFound desc = ", 4)
	src := &memSourceGitOps{byResource: map[string][]unstructured.Unstructured{
		"kustomizations": {fluxKustomization("flux-system", "platform", "False", "BuildFailed", longMsg, 30*time.Minute)},
	}}
	got := GitOpsDrift{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("want 1 diagnostic; got %d", len(got))
	}
	// Surfaced message in the diagnostic should include the trim marker.
	if !strings.Contains(got[0].Message, "...") {
		t.Errorf("long message should be truncated with ...; got %q", got[0].Message)
	}
}

func TestGitOpsDrift_CustomGracePeriod_HonoursOverride(t *testing.T) {
	// With a 60-minute grace, a 30-minute-old non-Ready Kustomization
	// should still be suppressed.
	src := &memSourceGitOps{byResource: map[string][]unstructured.Unstructured{
		"kustomizations": {fluxKustomization("flux-system", "platform", "False", "BuildFailed", "x", 30*time.Minute)},
	}}
	got := GitOpsDrift{GracePeriod: 60 * time.Minute}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("custom 60-min grace should suppress a 30-min-old failure; got %+v", got)
	}
}
