// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package probe

import (
	"context"
	"fmt"
	"strings"

	"github.com/srenix-ai/agentic-sre/internal/snapshot"
)

// PVCs ports probe_pvcs from cluster-health-report.sh:276-293.
type PVCs struct{}

// Name returns the component label for the report.
func (PVCs) Name() string { return "Storage Claims" }

// Run executes the PVC-binding probe.
func (PVCs) Run(ctx context.Context, src snapshot.Source) Result {
	r := Result{Component: ComponentResult{Component: "Storage Claims"}}
	list, err := src.List(ctx, snapshot.GVRPVC, "")
	if err != nil {
		r.Component.Status = "PROBE_FAILED"
		r.Component.Detail = "list pvcs: " + err.Error()
		r.Findings = append(r.Findings, Finding{
			Component: "Storage Claims",
			Severity:  SeverityCritical,
			Message:   "PVC probe failed (API error)",
		})
		return r
	}
	total := len(list.Items)
	var lostNames, pendingNames []string
	for _, p := range list.Items {
		phase, _, _ := getStringField(p.Object, "status", "phase")
		name := fmt.Sprintf("%s/%s", p.GetNamespace(), p.GetName())
		switch phase {
		case "Lost":
			lostNames = append(lostNames, name)
		case "Pending":
			pendingNames = append(pendingNames, name)
		}
	}

	if len(lostNames) > 0 {
		r.Component.Status = "CRITICAL"
		r.Component.Detail = fmt.Sprintf("%d/%d PVCs lost (data-loss risk): %s",
			len(lostNames), total, strings.Join(lostNames, ", "))
		r.Findings = append(r.Findings, Finding{
			Component:   "Storage Claims",
			Severity:    SeverityCritical,
			Message:     fmt.Sprintf("%d PVC(s) in Lost phase (backing PV deleted or inaccessible): %s", len(lostNames), strings.Join(lostNames, ", ")),
			Remediation: "Identify the backing PV and restore it, or recreate the PVC from a backup. `kubectl get pvc -A | grep Lost`",
		})
	}
	if len(pendingNames) > 0 {
		if r.Component.Status == "" {
			r.Component.Status = "DEGRADED"
		}
		r.Component.Detail = fmt.Sprintf("%d/%d PVCs pending: %s",
			len(pendingNames), total, strings.Join(pendingNames, ", "))
		r.Findings = append(r.Findings, Finding{
			Component:   "Storage Claims",
			Severity:    SeverityWarning,
			Message:     fmt.Sprintf("%d PVC(s) in Pending state: %s", len(pendingNames), strings.Join(pendingNames, ", ")),
			Remediation: "Check storage class and PV availability. `kubectl describe pvc -A | grep -A5 'Events'`",
		})
	}
	if len(lostNames) == 0 && len(pendingNames) == 0 {
		r.Component.Status = "HEALTHY"
		r.Component.Detail = fmt.Sprintf("All %d PVCs bound", total)
	}
	return r
}

// getStringField walks a nested map and returns a string at the given path.
func getStringField(m map[string]any, path ...string) (string, bool, error) {
	cur := any(m)
	for i, k := range path {
		mp, ok := cur.(map[string]any)
		if !ok {
			return "", false, nil
		}
		cur, ok = mp[k]
		if !ok {
			return "", false, nil
		}
		if i == len(path)-1 {
			s, ok := cur.(string)
			if !ok {
				return "", false, nil
			}
			return s, true, nil
		}
	}
	return "", false, nil
}
