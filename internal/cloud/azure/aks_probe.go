// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"
	"fmt"
	"os"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/cloud"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/probe"
)

// AKSControlPlane flags the configured AKS cluster (env
// CLOUD_AZURE_AKS_CLUSTER) when ProvisioningState=Failed or
// PowerState=Stopped. Mirrors AWS EKSControlPlane / GCP
// GKEControlPlane.
type AKSControlPlane struct{}

const aksControlPlaneName = "azure-aks-control-plane"
const aksClusterEnv = "CLOUD_AZURE_AKS_CLUSTER"

// Name satisfies cloudprobe.Probe.
func (AKSControlPlane) Name() string { return aksControlPlaneName }

// Run satisfies cloudprobe.Probe.
func (AKSControlPlane) Run(ctx context.Context, src cloud.Source) probe.Result {
	azClient := src.Azure()
	if azClient == nil {
		return skipped(aksControlPlaneName, "Azure not configured (cloud.azure.enabled=false)")
	}
	name := os.Getenv(aksClusterEnv)
	if name == "" {
		return skipped(aksControlPlaneName, "set "+aksClusterEnv+" to enable the AKS control-plane probe")
	}
	c, err := azClient.GetAKSCluster(ctx, name)
	if err != nil {
		return probeFailed(aksControlPlaneName, "containerservice.GetManagedCluster", err)
	}
	if c == nil {
		return probe.Result{
			Component: probe.ComponentResult{Component: aksControlPlaneName, Status: "CRITICAL", Detail: fmt.Sprintf("AKS cluster %q not found", name)},
			Findings: []probe.Finding{{
				Component:   fmt.Sprintf("azure-aks/%s/%s", azClient.SubscriptionID(), name),
				Severity:    probe.SeverityCritical,
				Message:     fmt.Sprintf("Configured AKS cluster %q does not exist", name),
				Remediation: fmt.Sprintf("Confirm CLOUD_AZURE_AKS_CLUSTER=%q: az aks list --subscription %s -o table", name, azClient.SubscriptionID()),
			}},
		}
	}

	subject := fmt.Sprintf("azure-aks/%s/%s/%s", azClient.SubscriptionID(), c.ResourceGroup, c.Name)
	var findings []probe.Finding
	if c.ProvisioningState == "Failed" {
		findings = append(findings, probe.Finding{
			Component:   subject,
			Severity:    probe.SeverityCritical,
			Message:     fmt.Sprintf("AKS cluster %q provisioningState=Failed", c.Name),
			Remediation: fmt.Sprintf("az aks show -g %s -n %s --subscription %s --query provisioningState", c.ResourceGroup, c.Name, azClient.SubscriptionID()),
		})
	} else if c.PowerState == "Stopped" {
		findings = append(findings, probe.Finding{
			Component:   subject,
			Severity:    probe.SeverityCritical,
			Message:     fmt.Sprintf("AKS cluster %q is Stopped — workloads are not scheduled", c.Name),
			Remediation: fmt.Sprintf("az aks start -g %s -n %s --subscription %s", c.ResourceGroup, c.Name, azClient.SubscriptionID()),
		})
	} else if c.ProvisioningState != "Succeeded" {
		findings = append(findings, probe.Finding{
			Component: subject,
			Severity:  probe.SeverityWarning,
			Message:   fmt.Sprintf("AKS cluster %q provisioningState=%s (not Succeeded)", c.Name, c.ProvisioningState),
		})
	}

	return probe.Result{
		Component: probe.ComponentResult{Component: aksControlPlaneName, Status: rollupStatus(findings), Detail: fmt.Sprintf("AKS cluster %q in %s", c.Name, c.Location)},
		Findings:  findings,
	}
}

// AKSNodePools flags AKS agent pools in Failed provisioning or Stopped
// power state. Mirrors AWS EKSNodeGroups / GCP GKENodePools.
type AKSNodePools struct{}

const aksNodePoolsName = "azure-aks-nodepools"

// Name satisfies cloudprobe.Probe.
func (AKSNodePools) Name() string { return aksNodePoolsName }

// Run satisfies cloudprobe.Probe.
func (AKSNodePools) Run(ctx context.Context, src cloud.Source) probe.Result {
	azClient := src.Azure()
	if azClient == nil {
		return skipped(aksNodePoolsName, "Azure not configured (cloud.azure.enabled=false)")
	}
	clusterName := os.Getenv(aksClusterEnv)
	if clusterName == "" {
		return skipped(aksNodePoolsName, "set "+aksClusterEnv+" to enable the AKS node-pool probe")
	}
	pools, err := azClient.ListAKSNodePools(ctx, clusterName)
	if err != nil {
		return probeFailed(aksNodePoolsName, "containerservice.ListAgentPools", err)
	}

	var findings []probe.Finding
	for _, p := range pools {
		subject := fmt.Sprintf("azure-aks-nodepool/%s/%s/%s", azClient.SubscriptionID(), clusterName, p.Name)
		switch {
		case p.ProvisioningState == "Failed":
			findings = append(findings, probe.Finding{
				Component:   subject,
				Severity:    probe.SeverityCritical,
				Message:     fmt.Sprintf("AKS node pool %q (cluster %q) provisioningState=Failed", p.Name, clusterName),
				Remediation: fmt.Sprintf("az aks nodepool show -g <rg> --cluster-name %s -n %s --subscription %s", clusterName, p.Name, azClient.SubscriptionID()),
			})
		case p.PowerState == "Stopped":
			findings = append(findings, probe.Finding{
				Component: subject,
				Severity:  probe.SeverityWarning,
				Message:   fmt.Sprintf("AKS node pool %q is Stopped", p.Name),
			})
		case p.ProvisioningState != "Succeeded":
			findings = append(findings, probe.Finding{
				Component: subject,
				Severity:  probe.SeverityWarning,
				Message:   fmt.Sprintf("AKS node pool %q provisioningState=%s (not Succeeded)", p.Name, p.ProvisioningState),
			})
		}
	}

	return probe.Result{
		Component: probe.ComponentResult{Component: aksNodePoolsName, Status: rollupStatus(findings), Detail: fmt.Sprintf("%d node pool(s) in cluster %q", len(pools), clusterName)},
		Findings:  findings,
	}
}
