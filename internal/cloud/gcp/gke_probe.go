// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package gcp

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/cloud"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/probe"
)

// GKEControlPlane flags a GKE cluster whose status is not RUNNING.
// Configured via CLOUD_GCP_GKE_CLUSTER env var (set by the Helm chart
// from cloud.gcp.probes.gkeClusterName); SKIPPED when unset. Mirrors
// the AWS EKSControlPlane probe.
type GKEControlPlane struct{}

const gkeControlPlaneName = "gcp-gke-control-plane"
const gkeClusterEnv = "CLOUD_GCP_GKE_CLUSTER"

// Name satisfies cloudprobe.Probe.
func (GKEControlPlane) Name() string { return gkeControlPlaneName }

// Run satisfies cloudprobe.Probe.
func (GKEControlPlane) Run(ctx context.Context, src cloud.Source) probe.Result {
	gcpClient := src.GCP()
	if gcpClient == nil {
		return skipped(gkeControlPlaneName, "GCP not configured (cloud.gcp.enabled=false)")
	}
	name := os.Getenv(gkeClusterEnv)
	if name == "" {
		return skipped(gkeControlPlaneName, "set "+gkeClusterEnv+" to enable the GKE control-plane probe")
	}
	cluster, err := gcpClient.GetGKECluster(ctx, name)
	if err != nil {
		return probeFailed(gkeControlPlaneName, "container.GetCluster", err)
	}
	if cluster == nil {
		return probe.Result{
			Component: probe.ComponentResult{
				Component: gkeControlPlaneName,
				Status:    "CRITICAL",
				Detail:    fmt.Sprintf("GKE cluster %q not found in project %s", name, gcpClient.Project()),
			},
			Findings: []probe.Finding{{
				Component:   fmt.Sprintf("gcp-gke/%s/%s", gcpClient.Project(), name),
				Severity:    probe.SeverityCritical,
				Message:     fmt.Sprintf("Configured GKE cluster %q does not exist", name),
				Remediation: fmt.Sprintf("Confirm CLOUD_GCP_GKE_CLUSTER=%q matches a live cluster: gcloud container clusters list --project=%s", name, gcpClient.Project()),
			}},
		}
	}

	subject := fmt.Sprintf("gcp-gke/%s/%s", gcpClient.Project(), cluster.Name)
	var findings []probe.Finding
	switch cluster.Status {
	case "RUNNING":
		// healthy
	case "ERROR", "DEGRADED":
		findings = append(findings, probe.Finding{
			Component:   subject,
			Severity:    probe.SeverityCritical,
			Message:     fmt.Sprintf("GKE cluster %q status=%s", cluster.Name, cluster.Status),
			Remediation: fmt.Sprintf("gcloud container clusters describe %s --location=%s --project=%s", cluster.Name, cluster.Location, gcpClient.Project()),
		})
	default:
		// PROVISIONING / RECONCILING / STOPPING — transitional.
		findings = append(findings, probe.Finding{
			Component: subject,
			Severity:  probe.SeverityWarning,
			Message:   fmt.Sprintf("GKE cluster %q status=%s (not RUNNING)", cluster.Name, cluster.Status),
		})
	}

	// Version information: emit Info when the cluster is RUNNING but has
	// no version data (shouldn't happen, but defensive).
	if cluster.Status == "RUNNING" && cluster.CurrentVersion == "" {
		findings = append(findings, probe.Finding{
			Component: subject,
			Severity:  probe.SeverityInfo,
			Message:   fmt.Sprintf("GKE cluster %q has no version information available", cluster.Name),
		})
	}

	detail := fmt.Sprintf("GKE cluster %q in %s", cluster.Name, cluster.Location)
	if cluster.CurrentVersion != "" {
		detail = fmt.Sprintf("GKE cluster %q in %s (version %s)", cluster.Name, cluster.Location, cluster.CurrentVersion)
	}

	return probe.Result{
		Component: probe.ComponentResult{
			Component: gkeControlPlaneName,
			Status:    rollupStatus(findings),
			Detail:    detail,
		},
		Findings: findings,
	}
}

// GKENodePools flags node pools in a non-RUNNING status for the
// configured GKE cluster (same env var as GKEControlPlane).
type GKENodePools struct{}

const gkeNodePoolsName = "gcp-gke-nodepools"

// Name satisfies cloudprobe.Probe.
func (GKENodePools) Name() string { return gkeNodePoolsName }

// Run satisfies cloudprobe.Probe.
func (GKENodePools) Run(ctx context.Context, src cloud.Source) probe.Result {
	gcpClient := src.GCP()
	if gcpClient == nil {
		return skipped(gkeNodePoolsName, "GCP not configured (cloud.gcp.enabled=false)")
	}
	clusterName := os.Getenv(gkeClusterEnv)
	if clusterName == "" {
		return skipped(gkeNodePoolsName, "set "+gkeClusterEnv+" to enable the GKE node-pool probe")
	}
	pools, err := gcpClient.ListGKENodePools(ctx, clusterName)
	if err != nil {
		return probeFailed(gkeNodePoolsName, "container.ListNodePools", err)
	}

	var findings []probe.Finding
	for _, p := range pools {
		subject := fmt.Sprintf("gcp-gke-nodepool/%s/%s/%s", gcpClient.Project(), clusterName, p.Name)
		switch p.Status {
		case "RUNNING":
			// healthy
		case "ERROR", "RUNNING_WITH_ERROR":
			findings = append(findings, probe.Finding{
				Component:   subject,
				Severity:    probe.SeverityCritical,
				Message:     fmt.Sprintf("GKE node pool %q (cluster %q) status=%s", p.Name, clusterName, p.Status),
				Remediation: fmt.Sprintf("gcloud container node-pools describe %s --cluster=%s --project=%s", p.Name, clusterName, gcpClient.Project()),
			})
		default:
			findings = append(findings, probe.Finding{
				Component: subject,
				Severity:  probe.SeverityWarning,
				Message:   fmt.Sprintf("GKE node pool %q status=%s (not RUNNING)", p.Name, p.Status),
			})
		}

		// Version drift: flag only when the node pool is MORE than one minor
		// version behind the control plane, or newer than it. GKE supports a
		// one-minor-version skew, and node pools routinely lag the control
		// plane by a patch / -gke.NNN suffix during normal rolling upgrades —
		// an exact-string compare would false-positive on every such cluster.
		// We compare minor versions only, matching the EKS/AKS sibling probes.
		if p.Version != "" && p.ClusterVersion != "" {
			skew := gkeMinorVersionSkew(p.Version, p.ClusterVersion)
			newer := gkeMinorVersionNewer(p.Version, p.ClusterVersion)
			if skew > 1 || newer {
				findings = append(findings, probe.Finding{
					Component:   subject,
					Severity:    probe.SeverityWarning,
					Message:     fmt.Sprintf("GKE node pool %q runs %s but control plane is %s (version skew > 1 minor)", p.Name, p.Version, p.ClusterVersion),
					Remediation: fmt.Sprintf("gcloud container clusters upgrade %s --node-pool=%s --project=%s", clusterName, p.Name, gcpClient.Project()),
				})
			}
		}
	}

	return probe.Result{
		Component: probe.ComponentResult{
			Component: gkeNodePoolsName,
			Status:    rollupStatus(findings),
			Detail:    fmt.Sprintf("%d node pool(s) in cluster %q", len(pools), clusterName),
		},
		Findings: findings,
	}
}

// gkeMinorVersionSkew returns the absolute difference in minor versions between
// two GKE version strings (e.g. "1.29.4-gke.1043004" → minor 29). Returns 0
// when either version is unparseable so an unknown version never false-fires.
func gkeMinorVersionSkew(v1, v2 string) int {
	m1 := gkeParseMinor(v1)
	m2 := gkeParseMinor(v2)
	if m1 < 0 || m2 < 0 {
		return 0
	}
	d := m1 - m2
	if d < 0 {
		d = -d
	}
	return d
}

// gkeMinorVersionNewer reports whether vA is a newer minor version than vB.
func gkeMinorVersionNewer(vA, vB string) bool {
	mA := gkeParseMinor(vA)
	mB := gkeParseMinor(vB)
	if mA < 0 || mB < 0 {
		return false
	}
	return mA > mB
}

// gkeParseMinor extracts the minor version integer from a GKE version string.
// "1.29.4-gke.1043004" → 29. Returns -1 when unparseable.
func gkeParseMinor(v string) int {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	if len(parts) < 2 {
		return -1
	}
	n, err := strconv.Atoi(parts[1])
	if err != nil {
		return -1
	}
	return n
}
