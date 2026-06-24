// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package probe

import (
	"context"
	"fmt"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ArgoCDApplication is the M2 probe-class fast-path for ArgoCD
// Application sync drift. Complements the v1.7 GitOpsDrift analyzer
// (which times signals against a grace window for the agent's
// investigation context) — this probe emits immediately when an
// Application reports OutOfSync or Degraded, no grace.
//
// Auto-skip when the Argo CD CRDs are not installed.
type ArgoCDApplication struct{}

const argoAppName = "ArgoCD-Application"

var gvrArgoApp = schema.GroupVersionResource{
	Group:    "argoproj.io",
	Version:  "v1alpha1",
	Resource: "applications",
}

// Name satisfies probe.Probe.
func (ArgoCDApplication) Name() string { return argoAppName }

// Run satisfies probe.Probe.
func (ArgoCDApplication) Run(ctx context.Context, src snapshot.Source) Result {
	r := Result{Component: ComponentResult{Component: argoAppName}}

	list, err := src.List(ctx, gvrArgoApp, "")
	if err != nil {
		r.Component.Status = "SKIPPED"
		r.Component.Detail = "Argo CD CRDs not installed (list applications failed)"
		return r
	}
	if list == nil || len(list.Items) == 0 {
		r.Component.Status = "HEALTHY"
		r.Component.Detail = "no Argo CD Applications"
		return r
	}

	var findings []Finding
	for i := range list.Items {
		app := &list.Items[i]
		ns := app.GetNamespace()
		name := app.GetName()
		subject := fmt.Sprintf("Application/%s/%s", ns, name)

		syncStatus, _, _ := unstructured.NestedString(app.Object, "status", "sync", "status")
		healthStatus, _, _ := unstructured.NestedString(app.Object, "status", "health", "status")
		operationMessage, _, _ := unstructured.NestedString(app.Object, "status", "operationState", "message")

		switch healthStatus {
		case "Degraded", "Missing":
			findings = append(findings, Finding{
				Component: subject,
				Severity:  SeverityCritical,
				Message: fmt.Sprintf("Argo Application %s/%s health=%s (sync=%s)",
					ns, name, healthStatus, syncStatus),
				Remediation: fmt.Sprintf("argocd app get %s/%s; argocd app sync %s/%s --strategy hook. Last operation message: %s",
					ns, name, ns, name, operationMessage),
			})
			continue
		case "Suspended":
			// Suspended is an intentional operator action (e.g. argocd app pause).
			// It signals "sync is paused" not "app is broken" — warn, do not page.
			findings = append(findings, Finding{
				Component: subject,
				Severity:  SeverityWarning,
				Message: fmt.Sprintf("Argo Application %s/%s health=Suspended (sync=%s) — sync is paused",
					ns, name, syncStatus),
				Remediation: fmt.Sprintf("If this suspension is unintentional: `argocd app resume %s`. "+
					"Otherwise this is expected and can be silenced.", name),
			})
			continue
		}

		switch syncStatus {
		case "OutOfSync":
			findings = append(findings, Finding{
				Component:   subject,
				Severity:    SeverityWarning,
				Message:     fmt.Sprintf("Argo Application %s/%s sync=OutOfSync (health=%s)", ns, name, healthStatus),
				Remediation: fmt.Sprintf("argocd app diff %s/%s — review the divergence then `argocd app sync %s/%s`.", ns, name, ns, name),
			})
		case "Unknown":
			findings = append(findings, Finding{
				Component: subject,
				Severity:  SeverityWarning,
				Message:   fmt.Sprintf("Argo Application %s/%s sync=Unknown (controller hasn't compared yet)", ns, name),
			})
		}
	}

	r.Component.Status = rollupComponentStatus(findings)
	r.Component.Detail = fmt.Sprintf("%d Argo CD Application(s) inspected", len(list.Items))
	r.Findings = findings
	return r
}
