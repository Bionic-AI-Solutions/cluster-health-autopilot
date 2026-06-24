// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package diagnose

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// GitOpsDrift surfaces controllers that have failed to reconcile a
// desired state from a Git source. It catches the most common
// "deploy looks healthy but the cluster doesn't match git" failure
// mode that pages oncall.
//
// What it detects (v1.7 first cut):
//
//   - **Argo CD** — Application.status.sync.status not "Synced"
//     past GracePeriod, OR Application.status.health.status not
//     "Healthy" past GracePeriod. Names the comparedTo commit so the
//     operator can git-diff against the live spec.
//   - **Flux Kustomization** — Kustomization.status.conditions[Ready]
//     != True past GracePeriod, with the controller's last-applied
//     revision + error message.
//   - **Flux HelmRelease** — HelmRelease.status.conditions[Ready] !=
//     True past GracePeriod, surfacing the helm-controller's reason
//     (UpgradeFailed / InstallFailed / Retrying / etc.).
//
// All three are operator-policy-safe: this analyzer is read-only. The
// agent's T1 fix proposer would propose a re-sync action (the
// action_kind would land in the operator's allowlist as something
// like "SyncArgoApplication"), but T1 wiring is a separate piece.
//
// Roadmap (v1.8):
//   - Helm-release values drift vs chart-source (this requires reading
//     the Git remote, not just the cluster — deferred to a separate
//     work item).
//
// Reduced scope (deliberately):
//   - We do NOT compare cluster-live to git-stored manifests. That's a
//     different class of work — needs Git remote read access (RBAC +
//     credentials surface) that's out of scope for v1.7. We rely on
//     the controllers' own status conditions instead.
type GitOpsDrift struct {
	// GracePeriod is how long a controller may report a non-Ready /
	// non-Synced status before we flag it. Zero defaults to 10 minutes.
	// Short windows produce noisy alerts (controllers are routinely
	// reconciling); long windows let real drift smolder. 10 minutes is
	// the median grace operators we've talked to live with.
	GracePeriod time.Duration

	// Now is the time-source; tests inject a fixed clock. Defaults to
	// time.Now.
	Now func() time.Time
}

// Name returns the analyzer's identifier. Pinned for metrics + dashboards.
func (GitOpsDrift) Name() string { return "GitOpsDrift" }

var (
	gvrArgoApplication = schema.GroupVersionResource{
		Group:    "argoproj.io",
		Version:  "v1alpha1",
		Resource: "applications",
	}
	gvrFluxKustomization = schema.GroupVersionResource{
		Group:    "kustomize.toolkit.fluxcd.io",
		Version:  "v1",
		Resource: "kustomizations",
	}
	gvrFluxHelmRelease = schema.GroupVersionResource{
		Group:    "helm.toolkit.fluxcd.io",
		Version:  "v2",
		Resource: "helmreleases",
	}
)

// Run walks every Argo Application / Flux Kustomization / Flux
// HelmRelease in the cluster and emits a Diagnostic for each that has
// been Out-of-Sync or NotReady past the GracePeriod. Each diagnostic
// carries the controller's own status message so operators can
// reproduce the underlying error from the CR itself.
//
// CRDs that don't exist in the cluster are silently skipped (a
// cluster without Argo CD shouldn't error from this analyzer).
func (g GitOpsDrift) Run(ctx context.Context, src snapshot.Source) []Diagnostic {
	grace := g.GracePeriod
	if grace == 0 {
		grace = 10 * time.Minute
	}
	now := time.Now
	if g.Now != nil {
		now = g.Now
	}

	var out []Diagnostic
	out = append(out, g.checkArgoApplications(ctx, src, grace, now())...)
	out = append(out, g.checkFluxKustomizations(ctx, src, grace, now())...)
	out = append(out, g.checkFluxHelmReleases(ctx, src, grace, now())...)
	return out
}

func (g GitOpsDrift) checkArgoApplications(ctx context.Context, src snapshot.Source, grace time.Duration, now time.Time) []Diagnostic {
	apps, err := src.List(ctx, gvrArgoApplication, "")
	if err != nil || apps == nil || len(apps.Items) == 0 {
		logListFailure("applications.argoproj.io", err, true) // silent when the CRD/resource is absent; logs Forbidden etc.
		return nil
	}
	var out []Diagnostic
	for i := range apps.Items {
		app := &apps.Items[i]
		ns := app.GetNamespace()
		name := app.GetName()
		subject := fmt.Sprintf("Application/%s/%s", ns, name)

		syncStatus, _, _ := unstructured.NestedString(app.Object, "status", "sync", "status")
		healthStatus, _, _ := unstructured.NestedString(app.Object, "status", "health", "status")
		revision, _, _ := unstructured.NestedString(app.Object, "status", "sync", "revision")

		// Use reconciledAt as the lastReconciled timestamp. Argo also
		// publishes operationState.startedAt for in-flight syncs, but
		// reconciledAt is the canonical "we tried" timestamp.
		reconciledAtRaw, _, _ := unstructured.NestedString(app.Object, "status", "reconciledAt")
		reconciledAt, _ := time.Parse(time.RFC3339, reconciledAtRaw)
		age := time.Duration(0)
		if !reconciledAt.IsZero() {
			age = now.Sub(reconciledAt)
		}

		if syncStatus != "" && syncStatus != "Synced" && age >= grace {
			severity := "warning"
			out = append(out, Diagnostic{
				Source:   "GitOpsDrift",
				Subject:  subject,
				Severity: severity,
				Message: fmt.Sprintf(
					"Argo Application %s/%s out-of-sync (status=%s, last reconciled %s ago, revision %s)",
					ns, name, syncStatus, durationShort(age), shortRev(revision)),
				Remediation: fmt.Sprintf(
					"Inspect: `argocd app diff %s` or `kubectl describe application -n %s %s`. "+
						"If the divergence is intentional, gate this Application on `argocd.argoproj.io/sync-options: ApplyOutOfSync=true` "+
						"or add `app.kubernetes.io/managed-by-out-of-sync` annotation. Otherwise: `argocd app sync %s`.",
					name, ns, name, name),
			})
		}

		// Health degradation is independent of sync — a sync can be
		// "Synced" while the deployed Pods crash-loop. Surface it
		// separately so the operator gets two distinct hints.
		if healthStatus != "" && healthStatus != "Healthy" && healthStatus != "Progressing" && age >= grace {
			severity := "warning"
			if healthStatus == "Degraded" {
				severity = "critical"
			}
			out = append(out, Diagnostic{
				Source:   "GitOpsDrift",
				Subject:  subject,
				Severity: severity,
				Message: fmt.Sprintf(
					"Argo Application %s/%s health=%s (last reconciled %s ago)",
					ns, name, healthStatus, durationShort(age)),
				Remediation: fmt.Sprintf(
					"Argo's health controller is reporting non-Healthy after a successful sync. "+
						"Most common cause: workload-level CrashLoopBackOff / image-pull / probe failure. "+
						"`kubectl get pods -n %s -l app.kubernetes.io/instance=%s` and re-run cha diagnose.",
					ns, name),
			})
		}
	}
	return out
}

func (g GitOpsDrift) checkFluxKustomizations(ctx context.Context, src snapshot.Source, grace time.Duration, now time.Time) []Diagnostic {
	ks, err := src.List(ctx, gvrFluxKustomization, "")
	if err != nil || ks == nil || len(ks.Items) == 0 {
		logListFailure("kustomizations.kustomize.toolkit.fluxcd.io", err, true) // silent when the CRD/resource is absent; logs Forbidden etc.
		return nil
	}
	var out []Diagnostic
	for i := range ks.Items {
		k := &ks.Items[i]
		out = append(out, fluxReadyDiagnostic(k, "Kustomization", grace, now)...)
	}
	return out
}

func (g GitOpsDrift) checkFluxHelmReleases(ctx context.Context, src snapshot.Source, grace time.Duration, now time.Time) []Diagnostic {
	hrs, err := src.List(ctx, gvrFluxHelmRelease, "")
	if err != nil || hrs == nil || len(hrs.Items) == 0 {
		logListFailure("helmreleases.helm.toolkit.fluxcd.io", err, true) // silent when the CRD/resource is absent; logs Forbidden etc.
		return nil
	}
	var out []Diagnostic
	for i := range hrs.Items {
		hr := &hrs.Items[i]
		out = append(out, fluxReadyDiagnostic(hr, "HelmRelease", grace, now)...)
	}
	return out
}

// fluxReadyDiagnostic shares the Ready-condition walk between Flux
// Kustomization and HelmRelease — both use the same status.conditions
// schema with Type=Ready.
func fluxReadyDiagnostic(u *unstructured.Unstructured, kind string, grace time.Duration, now time.Time) []Diagnostic {
	ns := u.GetNamespace()
	name := u.GetName()
	subject := fmt.Sprintf("%s/%s/%s", kind, ns, name)

	conds, _, _ := unstructured.NestedSlice(u.Object, "status", "conditions")
	if len(conds) == 0 {
		return nil
	}
	for _, c := range conds {
		cond, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		ctype, _ := cond["type"].(string)
		if ctype != "Ready" {
			continue
		}
		status, _ := cond["status"].(string)
		if status == "True" {
			return nil
		}
		// Flux conditions carry lastTransitionTime per the K8s
		// metav1.Condition convention.
		ltt, _ := cond["lastTransitionTime"].(string)
		t, _ := time.Parse(time.RFC3339, ltt)
		age := time.Duration(0)
		if !t.IsZero() {
			age = now.Sub(t)
		}
		if age < grace {
			return nil
		}

		reason, _ := cond["reason"].(string)
		msg, _ := cond["message"].(string)
		severity := "warning"
		// HelmRelease UpgradeFailed / InstallFailed are critical — the
		// release is broken until reconciled.
		if strings.Contains(reason, "Failed") {
			severity = "critical"
		}
		return []Diagnostic{{
			Source:   "GitOpsDrift",
			Subject:  subject,
			Severity: severity,
			Message: fmt.Sprintf(
				"Flux %s %s/%s Ready=%s (reason=%s, not reconciled for %s): %s",
				kind, ns, name, status, reason, durationShort(age), trimMessage(msg)),
			Remediation: fmt.Sprintf(
				"Inspect: `kubectl describe %s -n %s %s` for the full controller stack. "+
					"Force a reconcile: `flux reconcile %s -n %s %s`. "+
					"If the source is broken, fix the Git source and the controller will retry automatically.",
				strings.ToLower(kind), ns, name,
				strings.ToLower(kind), ns, name),
		}}
	}
	return nil
}

// shortRev truncates a Git revision string to 10 characters for
// log/Slack legibility. Empty input returns "(unknown)".
func shortRev(rev string) string {
	if rev == "" {
		return "(unknown)"
	}
	if len(rev) > 10 {
		return rev[:10]
	}
	return rev
}

// trimMessage caps a controller error message at 200 chars so it fits
// in a Slack post without pushing the rest of the report off-screen.
// The full message is still on the CR for kubectl describe.
func trimMessage(msg string) string {
	msg = strings.TrimSpace(msg)
	if len(msg) > 200 {
		return msg[:197] + "..."
	}
	return msg
}

// durationShort formats a duration as "5m", "1h2m", etc. for diagnostic
// messages. Shared with other analyzers via this package; the
// statefulset_replica_pressure analyzer in the paid catalog has its
// own copy to avoid a cross-package dep.
func durationShort(d time.Duration) string {
	out := d.Round(time.Second).String()
	if strings.HasSuffix(out, "m0s") {
		out = strings.TrimSuffix(out, "0s")
	}
	if strings.HasSuffix(out, "h0m") {
		out = strings.TrimSuffix(out, "0m")
	}
	return out
}
