// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package probe

import (
	"context"
	"fmt"
	"strings"

	"github.com/srenix-ai/agentic-sre/internal/snapshot"
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

	// --- Spilo / Patroni fallback (cluster-wide) ---
	//
	// List across all namespaces so deployments outside the legacy "pg"
	// namespace are detected. Group by cluster-name label to support
	// multiple Spilo clusters.
	pods, err := src.List(ctx, snapshot.GVRPod, "")
	if err == nil {
		type spiloCluster struct {
			ns       string
			name     string
			primary  *map[string]any
			replicas int
		}
		clusters := map[string]*spiloCluster{}
		for i := range pods.Items {
			obj := pods.Items[i].Object
			labels := pods.Items[i].GetLabels()
			if labels["application"] != "spilo" {
				continue
			}
			clusterName := labels["cluster-name"]
			if clusterName == "" {
				continue
			}
			ns := pods.Items[i].GetNamespace()
			key := ns + "/" + clusterName
			if clusters[key] == nil {
				clusters[key] = &spiloCluster{ns: ns, name: clusterName}
			}
			switch labels["spilo-role"] {
			case "master":
				p := obj
				clusters[key].primary = &p
			case "replica":
				clusters[key].replicas++
			}
		}
		if len(clusters) > 0 {
			unhealthy := 0
			var details []string
			for _, cl := range clusters {
				if cl.primary == nil {
					unhealthy++
					details = append(details, fmt.Sprintf("%s@%s NO PRIMARY", cl.name, cl.ns))
					r.Findings = append(r.Findings, Finding{
						Component:   fmt.Sprintf("PostgreSQL (%s/%s)", cl.ns, cl.name),
						Severity:    SeverityCritical,
						Message:     fmt.Sprintf("Spilo cluster %s/%s has no primary pod", cl.ns, cl.name),
						Remediation: fmt.Sprintf("Check events: `kubectl describe pod -n %s -l spilo-role=master,cluster-name=%s`", cl.ns, cl.name),
					})
					continue
				}
				phase, _, _ := getStringField(*cl.primary, "status", "phase")
				if phase != "Running" {
					unhealthy++
					details = append(details, fmt.Sprintf("%s@%s UNHEALTHY (primary=%s)", cl.name, cl.ns, phase))
					r.Findings = append(r.Findings, Finding{
						Component:   fmt.Sprintf("PostgreSQL (%s/%s)", cl.ns, cl.name),
						Severity:    SeverityCritical,
						Message:     fmt.Sprintf("Spilo primary not running (%s/%s, status: %s)", cl.ns, cl.name, phase),
						Remediation: fmt.Sprintf("Check events: `kubectl describe pod -n %s -l spilo-role=master,cluster-name=%s`", cl.ns, cl.name),
					})
					continue
				}
				details = append(details, fmt.Sprintf("%s@%s OK (%d replica(s))", cl.name, cl.ns, cl.replicas))
			}
			if unhealthy == 0 {
				r.Component.Status = "HEALTHY"
				r.Component.Detail = fmt.Sprintf("%d Spilo cluster(s): %s", len(clusters), strings.Join(details, "; "))
			} else {
				r.Component.Status = "CRITICAL"
				r.Component.Detail = fmt.Sprintf("%d/%d Spilo cluster(s) unhealthy: %s", unhealthy, len(clusters), strings.Join(details, "; "))
			}
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
