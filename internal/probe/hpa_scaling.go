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

// HPAScaling is the M2 probe-class fast-path for HPA failures. It
// complements the v1.8 CapacityDrift analyzer (which evaluates
// long-dwell signals like "pinned at max for 24h"); HPAScaling here
// is the resource-event-driven signal: any HPA reporting
// status.conditions[type=ScalingActive,status=False] OR
// status.conditions[type=AbleToScale,status=False] right now is
// surfaced immediately as CRITICAL, regardless of dwell.
//
// Auto-skip rule: a cluster with zero HPAs is HEALTHY rather than
// SKIPPED — operators don't need to opt out of an empty list.
type HPAScaling struct{}

const hpaScalingName = "HPAScaling"

// Probe-local copy of the GVR — kept narrow rather than reaching into
// internal/diagnose's gvrHPA so internal/probe stays free of any
// internal/diagnose imports.
var gvrHPAv2 = schema.GroupVersionResource{
	Group:    "autoscaling",
	Version:  "v2",
	Resource: "horizontalpodautoscalers",
}

// Name satisfies probe.Probe.
func (HPAScaling) Name() string { return hpaScalingName }

// Run satisfies probe.Probe.
func (HPAScaling) Run(ctx context.Context, src snapshot.Source) Result {
	r := Result{Component: ComponentResult{Component: hpaScalingName}}

	list, err := src.List(ctx, gvrHPAv2, "")
	if err != nil {
		r.Component.Status = "PROBE_FAILED"
		r.Component.Detail = "list horizontalpodautoscalers: " + err.Error()
		return r
	}
	if list == nil || len(list.Items) == 0 {
		r.Component.Status = "HEALTHY"
		r.Component.Detail = "no HorizontalPodAutoscalers"
		return r
	}

	var findings []Finding
	for i := range list.Items {
		h := &list.Items[i]
		ns := h.GetNamespace()
		name := h.GetName()
		subject := fmt.Sprintf("HorizontalPodAutoscaler/%s/%s", ns, name)

		conds, _, _ := unstructured.NestedSlice(h.Object, "status", "conditions")
		for _, c := range conds {
			cond, ok := c.(map[string]interface{})
			if !ok {
				continue
			}
			t, _ := cond["type"].(string)
			s, _ := cond["status"].(string)
			if (t == "ScalingActive" || t == "AbleToScale") && s == "False" {
				reason, _ := cond["reason"].(string)
				msg, _ := cond["message"].(string)
				// ScalingActive=False / reason=ScalingDisabled is the
				// EXPECTED state when the target is intentionally at zero
				// (KEDA scale-to-zero, or a Deployment scaled to 0) — the
				// HPA simply goes dormant. Flagging it CRITICAL is a false
				// positive that erodes trust, so downgrade to Warning
				// (visible, not paging). Every other reason
				// (FailedGetScale, FailedGetResourceMetric, quota/PDB
				// blocks, etc.) stays CRITICAL.
				sev := SeverityCritical
				m := fmt.Sprintf("HPA %s/%s %s=False (reason=%s)", ns, name, t, reason)
				if t == "ScalingActive" && reason == "ScalingDisabled" {
					sev = SeverityWarning
					m = fmt.Sprintf("HPA %s/%s autoscaling inactive (reason=ScalingDisabled) — expected when the target is scaled to zero / KEDA scale-to-zero; not an outage", ns, name)
				}
				findings = append(findings, Finding{
					Component:   subject,
					Severity:    sev,
					Message:     m,
					Remediation: fmt.Sprintf("kubectl describe hpa %s -n %s — controller's last message: %s", name, ns, msg),
				})
				break
			}
		}
	}

	r.Component.Status = rollupComponentStatus(findings)
	r.Component.Detail = fmt.Sprintf("%d HPA(s) inspected", len(list.Items))
	r.Findings = findings
	return r
}
