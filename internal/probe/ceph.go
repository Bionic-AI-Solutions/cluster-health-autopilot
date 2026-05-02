// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package probe

import (
	"context"
	"fmt"
	"strings"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
)

// Ceph ports probe_ceph from cluster-health-report.sh:55-104.
//
// The bash version called `kubectl exec -n rook-ceph deploy/rook-ceph-tools
// -- ceph health detail` and `ceph df`, which requires live cluster access
// and cannot work in zero-trust offline mode. This Go port reads the same
// information from the CephCluster CRD `.status.ceph` fields that
// rook-ceph-operator reconciles every ~60s — works identically online and
// against a captured snapshot.
//
// Status mapping:
//
//	status.ceph.health == HEALTH_OK   → HEALTHY
//	status.ceph.health == HEALTH_WARN → DEGRADED + warning finding
//	status.ceph.health == HEALTH_ERR  → CRITICAL + critical finding
//	status.phase != Ready             → CRITICAL (cluster mid-bringup or wedged)
//	capacity > 80% used               → DEGRADED add-on warning
//	No CephCluster found              → SKIPPED
type Ceph struct{}

// Name returns the component label for the report.
func (Ceph) Name() string { return "Ceph Storage" }

// Run executes the Ceph health probe.
func (Ceph) Run(ctx context.Context, src snapshot.Source) Result {
	r := Result{Component: ComponentResult{Component: "Ceph Storage"}}

	list, err := src.List(ctx, snapshot.GVRCephCluster, "")
	if err != nil {
		r.Component.Status = "PROBE_FAILED"
		r.Component.Detail = "list cephclusters: " + err.Error()
		r.Findings = append(r.Findings, Finding{
			Component:   "Ceph Storage",
			Severity:    SeverityCritical,
			Message:     "CephCluster probe failed (API error)",
			Remediation: "Check API connectivity from health-report pod and SA RBAC for ceph.rook.io/cephclusters",
		})
		return r
	}
	if len(list.Items) == 0 {
		r.Component.Status = "SKIPPED"
		r.Component.Detail = "No CephCluster CRDs found (rook-ceph operator not installed?)"
		return r
	}

	healthy := 0
	degraded := 0
	critical := 0
	var details []string

	for _, c := range list.Items {
		ns := c.GetNamespace()
		name := c.GetName()
		health, _, _ := getStringField(c.Object, "status", "ceph", "health")
		phase, _, _ := getStringField(c.Object, "status", "phase")
		bytesUsed, _, _ := getInt64Field(c.Object, "status", "ceph", "capacity", "bytesUsed")
		bytesTotal, _, _ := getInt64Field(c.Object, "status", "ceph", "capacity", "bytesTotal")
		usedPct := percentUsed(bytesUsed, bytesTotal)

		// Phase has its own failure path (not Ready means the cluster isn't even up).
		if phase != "" && phase != "Ready" {
			critical++
			r.Findings = append(r.Findings, Finding{
				Component:   fmt.Sprintf("Ceph Storage (%s/%s)", ns, name),
				Severity:    SeverityCritical,
				Message:     fmt.Sprintf("Phase=%s (expected Ready)", phase),
				Remediation: fmt.Sprintf("Check: `kubectl describe cephcluster -n %s %s`", ns, name),
			})
			details = append(details, fmt.Sprintf("%s@%s phase=%s", name, ns, phase))
			continue
		}

		switch strings.ToUpper(health) {
		case "HEALTH_OK":
			healthy++
			details = append(details, fmt.Sprintf("%s@%s OK (%.1f%% used)", name, ns, usedPct))
		case "HEALTH_WARN":
			degraded++
			r.Findings = append(r.Findings, Finding{
				Component: fmt.Sprintf("Ceph Storage (%s/%s)", ns, name),
				Severity:  SeverityWarning,
				Message:   "Cluster reports HEALTH_WARN",
				Remediation: fmt.Sprintf(
					"Detail: `kubectl exec -n %s deploy/rook-ceph-tools -- ceph health detail`",
					ns,
				),
			})
			details = append(details, fmt.Sprintf("%s@%s WARN (%.1f%% used)", name, ns, usedPct))
		case "HEALTH_ERR":
			critical++
			r.Findings = append(r.Findings, Finding{
				Component: fmt.Sprintf("Ceph Storage (%s/%s)", ns, name),
				Severity:  SeverityCritical,
				Message:   "Cluster reports HEALTH_ERR",
				Remediation: fmt.Sprintf(
					"Detail: `kubectl exec -n %s deploy/rook-ceph-tools -- ceph health detail` "+
						"and check `CEPH_PG_INCONSISTENCY_RESOLUTION.md` if PGs are inconsistent",
					ns,
				),
			})
			details = append(details, fmt.Sprintf("%s@%s ERR (%.1f%% used)", name, ns, usedPct))
		default:
			// Empty / unknown — treat as critical because we couldn't determine
			// health from the CRD (likely status hasn't populated yet).
			critical++
			r.Findings = append(r.Findings, Finding{
				Component: fmt.Sprintf("Ceph Storage (%s/%s)", ns, name),
				Severity:  SeverityCritical,
				Message:   fmt.Sprintf("status.ceph.health unset or unrecognized (%q)", health),
			})
			details = append(details, fmt.Sprintf("%s@%s health=%q", name, ns, health))
		}

		// Capacity warning is additive to whatever health reported.
		if usedPct > 80.0 {
			degraded++
			r.Findings = append(r.Findings, Finding{
				Component: fmt.Sprintf("Ceph Storage (%s/%s)", ns, name),
				Severity:  SeverityWarning,
				Message:   fmt.Sprintf("Storage %.1f%% full (>80%%)", usedPct),
			})
		}
	}

	switch {
	case critical > 0:
		r.Component.Status = "CRITICAL"
	case degraded > 0:
		r.Component.Status = "DEGRADED"
	default:
		r.Component.Status = "HEALTHY"
	}
	r.Component.Detail = fmt.Sprintf("%d cluster(s): %s", len(list.Items), strings.Join(details, "; "))
	return r
}

// percentUsed safely computes used/total*100 with zero-handling.
func percentUsed(used, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return float64(used) * 100.0 / float64(total)
}
