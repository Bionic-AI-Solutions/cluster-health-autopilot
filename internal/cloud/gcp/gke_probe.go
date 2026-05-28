// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package gcp

import (
	"context"
	"fmt"
	"os"

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

	return probe.Result{
		Component: probe.ComponentResult{
			Component: gkeControlPlaneName,
			Status:    rollupStatus(findings),
			Detail:    fmt.Sprintf("GKE cluster %q in %s", cluster.Name, cluster.Location),
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
