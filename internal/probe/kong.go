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

// Kong reports drift on the Kong ingress controller's KongPlugin
// custom resources — typically the most operator-tunable surface on
// a Kong install. Plugin resources with a non-empty
// status.conditions[type=Programmed,status=False] indicate a plugin
// configuration the controller can't apply; the gateway is then
// serving the upstream traffic without the intended policy
// (rate-limit / auth / CORS / etc.).
//
// Auto-skip when the Kong CRDs are not installed — the probe lists
// configuration.konghq.com/kongplugins; a list failure suggesting
// the resource is unknown results in a SKIPPED component without
// touching the operator's noise budget.
type Kong struct{}

const kongName = "Kong"

var gvrKongPlugin = schema.GroupVersionResource{
	Group:    "configuration.konghq.com",
	Version:  "v1",
	Resource: "kongplugins",
}

// Name satisfies probe.Probe.
func (Kong) Name() string { return kongName }

// Run satisfies probe.Probe.
func (Kong) Run(ctx context.Context, src snapshot.Source) Result {
	r := Result{Component: ComponentResult{Component: kongName}}

	list, err := src.List(ctx, gvrKongPlugin, "")
	if err != nil {
		// Treat any list error as "CRD not installed" — the operator
		// reads SKIPPED and confirms with kubectl get crd | grep kong.
		r.Component.Status = "SKIPPED"
		r.Component.Detail = "Kong CRDs not installed (list kongplugins failed)"
		return r
	}
	if list == nil || len(list.Items) == 0 {
		r.Component.Status = "HEALTHY"
		r.Component.Detail = "no KongPlugin resources"
		return r
	}

	var findings []Finding
	for i := range list.Items {
		p := &list.Items[i]
		ns := p.GetNamespace()
		name := p.GetName()
		subject := fmt.Sprintf("KongPlugin/%s/%s", ns, name)

		conds, _, _ := unstructured.NestedSlice(p.Object, "status", "conditions")
		programmedFalse := false
		var reason, msg string
		for _, c := range conds {
			cond, ok := c.(map[string]interface{})
			if !ok {
				continue
			}
			ctype, _ := cond["type"].(string)
			cstatus, _ := cond["status"].(string)
			if ctype == "Programmed" && cstatus == "False" {
				programmedFalse = true
				reason, _ = cond["reason"].(string)
				msg, _ = cond["message"].(string)
				break
			}
		}
		if programmedFalse {
			findings = append(findings, Finding{
				Component:   subject,
				Severity:    SeverityCritical,
				Message:     fmt.Sprintf("KongPlugin %s/%s reports Programmed=False (reason=%s)", ns, name, reason),
				Remediation: fmt.Sprintf("kubectl describe kongplugin %s -n %s — inspect the controller's last attempt. Detail: %s", name, ns, msg),
			})
		}
	}

	r.Component.Status = rollupComponentStatus(findings)
	r.Component.Detail = fmt.Sprintf("%d KongPlugin resource(s) inspected", len(list.Items))
	r.Findings = findings
	return r
}

// rollupComponentStatus folds findings into HEALTHY / DEGRADED /
// CRITICAL — same convention probes use throughout internal/probe.
func rollupComponentStatus(findings []Finding) string {
	status := "HEALTHY"
	for _, f := range findings {
		if f.Severity == SeverityCritical {
			return "CRITICAL"
		}
		if f.Severity == SeverityWarning {
			status = "DEGRADED"
		}
	}
	return status
}
