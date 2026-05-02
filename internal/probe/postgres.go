// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package probe

import (
	"context"
	"fmt"
	"strings"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
)

// Postgres ports probe_postgres from cluster-health-report.sh:106-166.
//
// Operator-agnostic: tries CloudNativePG (clusters.postgresql.cnpg.io) first,
// falls back to Zalando Spilo (Patroni) by pod label, and reports SKIPPED only
// when neither operator owns any clusters.
type Postgres struct{}

// Name returns the component label for the report.
func (Postgres) Name() string { return "PostgreSQL" }

// Run executes the database-readiness probe.
func (Postgres) Run(ctx context.Context, src snapshot.Source) Result {
	r := Result{Component: ComponentResult{Component: "PostgreSQL"}}

	// --- CNPG path ---
	if cnpgList, err := src.List(ctx, snapshot.GVRCNPGCluster, ""); err == nil && len(cnpgList.Items) > 0 {
		total := 0
		unhealthy := 0
		var details []string
		for _, c := range cnpgList.Items {
			ns := c.GetNamespace()
			name := c.GetName()
			phase, _, _ := getStringField(c.Object, "status", "phase")
			instances, _, _ := getInt64Field(c.Object, "status", "instances")
			ready, _, _ := getInt64Field(c.Object, "status", "readyInstances")
			primary, _, _ := getStringField(c.Object, "status", "currentPrimary")
			total++
			if instances > 0 && ready == instances && strings.Contains(strings.ToLower(phase), "healthy") {
				details = append(details, fmt.Sprintf("%s@%s (%d/%d ready, primary=%s)", name, ns, ready, instances, primary))
				continue
			}
			unhealthy++
			details = append(details, fmt.Sprintf("%s@%s UNHEALTHY (%d/%d ready, phase=%s)", name, ns, ready, instances, phase))
			r.Findings = append(r.Findings, Finding{
				Component:   fmt.Sprintf("PostgreSQL (%s/%s)", ns, name),
				Severity:    SeverityCritical,
				Message:     fmt.Sprintf("CNPG cluster not healthy: phase=%s, %d/%d ready, primary=%s", phase, ready, instances, primary),
				Remediation: fmt.Sprintf("Check: `kubectl get clusters.postgresql.cnpg.io -n %s %s`", ns, name),
			})
		}
		if unhealthy == 0 {
			r.Component.Status = "HEALTHY"
			r.Component.Detail = fmt.Sprintf("%d CNPG cluster(s): %s", total, strings.Join(details, "; "))
		} else {
			r.Component.Status = "CRITICAL"
			r.Component.Detail = fmt.Sprintf("%d/%d CNPG cluster(s) unhealthy: %s", unhealthy, total, strings.Join(details, "; "))
		}
		return r
	}

	// --- Spilo / Patroni fallback ---
	pods, err := src.List(ctx, snapshot.GVRPod, "pg")
	if err == nil {
		var primary *map[string]any
		replicas := 0
		for i := range pods.Items {
			obj := pods.Items[i].Object
			labels := pods.Items[i].GetLabels()
			if labels["application"] != "spilo" || labels["cluster-name"] != "pg" {
				continue
			}
			switch labels["spilo-role"] {
			case "master":
				p := obj
				primary = &p
			case "replica":
				replicas++
			}
		}
		if primary != nil {
			phase, _, _ := getStringField(*primary, "status", "phase")
			if phase != "Running" {
				r.Component.Status = "CRITICAL"
				r.Component.Detail = "Spilo primary pod status: " + phase
				r.Findings = append(r.Findings, Finding{
					Component:   "PostgreSQL",
					Severity:    SeverityCritical,
					Message:     fmt.Sprintf("Primary pod not running (status: %s)", phase),
					Remediation: "Check events: `kubectl describe pod -n pg -l spilo-role=master`",
				})
				return r
			}
			r.Component.Status = "HEALTHY"
			r.Component.Detail = fmt.Sprintf("Spilo primary operational, %d replica(s)", replicas)
			return r
		}
	}

	// --- Neither operator found ---
	r.Component.Status = "SKIPPED"
	r.Component.Detail = "No CNPG cluster and no Spilo primary pod found"
	return r
}

// getInt64Field walks a nested map and returns an int64 at the given path.
// Tolerates float64 (json.Unmarshal default for numbers) and int64.
func getInt64Field(m map[string]any, path ...string) (int64, bool, error) {
	cur := any(m)
	for i, k := range path {
		mp, ok := cur.(map[string]any)
		if !ok {
			return 0, false, nil
		}
		cur, ok = mp[k]
		if !ok {
			return 0, false, nil
		}
		if i == len(path)-1 {
			switch v := cur.(type) {
			case int64:
				return v, true, nil
			case float64:
				return int64(v), true, nil
			case int:
				return int64(v), true, nil
			default:
				return 0, false, nil
			}
		}
	}
	return 0, false, nil
}
