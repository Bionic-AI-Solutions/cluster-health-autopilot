// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package probe

import (
	"context"
	"fmt"

	"github.com/salilkadam/cluster-health-autopilot/internal/snapshot"
)

// PVCs ports probe_pvcs from cluster-health-report.sh:276-293.
type PVCs struct{}

func (PVCs) Name() string { return "Storage Claims" }

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
	pending := 0
	total := len(list.Items)
	for _, p := range list.Items {
		phase, _, _ := getStringField(p.Object, "status", "phase")
		if phase == "Pending" {
			pending++
		}
	}
	if pending == 0 {
		r.Component.Status = "HEALTHY"
		r.Component.Detail = fmt.Sprintf("All %d PVCs bound", total)
	} else {
		r.Component.Status = "DEGRADED"
		r.Component.Detail = fmt.Sprintf("%d/%d PVCs pending", pending, total)
		r.Findings = append(r.Findings, Finding{
			Component: "Storage Claims",
			Severity:  SeverityWarning,
			Message:   fmt.Sprintf("%d PVC(s) in Pending state", pending),
		})
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
