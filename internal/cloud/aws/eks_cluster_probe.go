// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/cloud"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/probe"
)

// EKSControlPlane flags:
//   - EKS cluster status not ACTIVE
//   - version skew between control plane and node groups (> 1 minor version)
//   - add-on staleness (DEGRADED / CREATE_FAILED / UPDATE_FAILED status)
//
// Configured via CLOUD_AWS_EKS_CLUSTER env var; SKIPPED when unset.
type EKSControlPlane struct{}

const eksClusterName = "aws-eks-control-plane"

const eksClusterEnv = "CLOUD_AWS_EKS_CLUSTER"

// Name satisfies cloudprobe.Probe.
func (EKSControlPlane) Name() string { return eksClusterName }

// Run satisfies cloudprobe.Probe.
func (EKSControlPlane) Run(ctx context.Context, src cloud.Source) probe.Result {
	awsClient := src.AWS()
	if awsClient == nil {
		return skipped(eksClusterName, "AWS not configured")
	}
	name := os.Getenv(eksClusterEnv)
	if name == "" {
		return skipped(eksClusterName, "set "+eksClusterEnv+" to enable EKS control-plane probe")
	}
	cluster, err := awsClient.DescribeEKSCluster(ctx, name)
	if err != nil {
		return probeFailed(eksClusterName, "eks.DescribeCluster", err)
	}
	subject := fmt.Sprintf("aws-eks/%s/%s", awsClient.Region(), name)
	if cluster == nil {
		return probe.Result{
			Component: probe.ComponentResult{Component: eksClusterName, Status: "CRITICAL", Detail: "cluster not found"},
			Findings: []probe.Finding{{
				Component: subject, Severity: probe.SeverityCritical,
				Message:     fmt.Sprintf("EKS cluster %q not found in region %s — drift between Helm config (cloud.aws.probes.eksClusterName) and reality", name, awsClient.Region()),
				Remediation: "Verify cluster exists: aws eks describe-cluster --name " + name,
			}},
		}
	}
	var findings []probe.Finding
	switch cluster.Status {
	case "ACTIVE":
		// healthy
	case "CREATING", "UPDATING":
		findings = append(findings, probe.Finding{
			Component: subject, Severity: probe.SeverityWarning,
			Message: fmt.Sprintf("EKS cluster %q is %s (transitional)", name, cluster.Status),
		})
	default:
		findings = append(findings, probe.Finding{
			Component: subject, Severity: probe.SeverityCritical,
			Message:     fmt.Sprintf("EKS cluster %q status is %q (not ACTIVE)", name, cluster.Status),
			Remediation: "aws eks describe-cluster --name " + name,
		})
	}

	// --- version skew: control plane vs node groups ---
	ngs, ngErr := awsClient.ListEKSNodeGroups(ctx, name)
	if ngErr != nil {
		log.Printf("aws-eks-control-plane: skipping version-skew check: %v", ngErr)
	} else {
		for _, ng := range ngs {
			if ng.Version == "" || cluster.Version == "" {
				continue
			}
			skew := minorVersionSkew(cluster.Version, ng.Version)
			if skew > 1 || minorVersionNewer(ng.Version, cluster.Version) {
				ngSubject := fmt.Sprintf("aws-eks-nodegroup/%s/%s/%s", awsClient.Region(), name, ng.Name)
				findings = append(findings, probe.Finding{
					Component: ngSubject, Severity: probe.SeverityWarning,
					Message:     fmt.Sprintf("EKS node group %q runs Kubernetes %s but control plane is %s (version skew)", ng.Name, ng.Version, cluster.Version),
					Remediation: fmt.Sprintf("aws eks update-nodegroup-version --cluster-name %s --nodegroup-name %s", name, ng.Name),
				})
			}
		}
	}

	// --- addon staleness ---
	addons, addonErr := awsClient.ListEKSAddons(ctx, name)
	if addonErr != nil {
		log.Printf("aws-eks-control-plane: skipping addon-staleness check: %v", addonErr)
	} else {
		for _, addon := range addons {
			addonSubject := fmt.Sprintf("aws-eks-addon/%s/%s/%s", awsClient.Region(), name, addon.AddonName)
			switch addon.Status {
			case "DEGRADED", "CREATE_FAILED", "UPDATE_FAILED":
				findings = append(findings, probe.Finding{
					Component: addonSubject, Severity: probe.SeverityWarning,
					Message:     fmt.Sprintf("EKS addon %q is in status %s", addon.AddonName, addon.Status),
					Remediation: fmt.Sprintf("aws eks describe-addon --cluster-name %s --addon-name %s", name, addon.AddonName),
				})
			}
			if addon.MarketplaceVersion != "" && addon.AddonVersion != "" &&
				addon.AddonVersion != addon.MarketplaceVersion {
				findings = append(findings, probe.Finding{
					Component: addonSubject, Severity: probe.SeverityInfo,
					Message: fmt.Sprintf("EKS addon %q version %s has a newer release available (%s)", addon.AddonName, addon.AddonVersion, addon.MarketplaceVersion),
				})
			}
		}
	}

	return probe.Result{
		Component: probe.ComponentResult{
			Component: eksClusterName,
			Status:    rollupStatus(findings),
			Detail:    fmt.Sprintf("cluster %s v%s status=%s", name, cluster.Version, cluster.Status),
		},
		Findings: findings,
	}
}

// minorVersionSkew returns the absolute difference in Kubernetes minor
// versions between two version strings like "1.30" or "1.30.2". Returns
// 0 when either version is unparseable.
func minorVersionSkew(v1, v2 string) int {
	m1 := parseMinor(v1)
	m2 := parseMinor(v2)
	if m1 < 0 || m2 < 0 {
		return 0
	}
	d := m1 - m2
	if d < 0 {
		d = -d
	}
	return d
}

// minorVersionNewer reports whether vA is a newer minor version than vB.
func minorVersionNewer(vA, vB string) bool {
	mA := parseMinor(vA)
	mB := parseMinor(vB)
	if mA < 0 || mB < 0 {
		return false
	}
	return mA > mB
}

// parseMinor extracts the minor version number from a Kubernetes version
// string. "1.30" → 30, "1.30.2" → 30, "" or unparseable → -1.
func parseMinor(v string) int {
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
