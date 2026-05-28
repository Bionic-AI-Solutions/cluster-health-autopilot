// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"
	"errors"
	"strings"
	"testing"

	pkgazure "github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/cloud/azure"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/probe"
)

// --- AKSControlPlane ---

func TestAKS_SkippedWhenAzureMissing(t *testing.T) {
	got := AKSControlPlane{}.Run(context.Background(), &fakeSource{})
	if got.Component.Status != "SKIPPED" {
		t.Errorf("status=%s want SKIPPED", got.Component.Status)
	}
}

func TestAKS_SkippedWhenEnvUnset(t *testing.T) {
	t.Setenv(aksClusterEnv, "")
	got := AKSControlPlane{}.Run(context.Background(), &fakeSource{azure: &fakeAzure{subscription: "s"}})
	if got.Component.Status != "SKIPPED" {
		t.Errorf("status=%s want SKIPPED (env unset)", got.Component.Status)
	}
}

func TestAKS_SucceededHealthy(t *testing.T) {
	t.Setenv(aksClusterEnv, "prod")
	got := AKSControlPlane{}.Run(context.Background(), &fakeSource{
		azure: &fakeAzure{subscription: "s", cluster: &pkgazure.AKSCluster{Name: "prod", ResourceGroup: "rg", ProvisioningState: "Succeeded", PowerState: "Running"}},
	})
	if got.Component.Status != "HEALTHY" {
		t.Errorf("status=%s want HEALTHY", got.Component.Status)
	}
}

func TestAKS_Failed_Critical(t *testing.T) {
	t.Setenv(aksClusterEnv, "prod")
	got := AKSControlPlane{}.Run(context.Background(), &fakeSource{
		azure: &fakeAzure{subscription: "s", cluster: &pkgazure.AKSCluster{Name: "prod", ResourceGroup: "rg", ProvisioningState: "Failed"}},
	})
	if got.Component.Status != "CRITICAL" {
		t.Errorf("status=%s want CRITICAL", got.Component.Status)
	}
}

func TestAKS_Stopped_Critical(t *testing.T) {
	t.Setenv(aksClusterEnv, "prod")
	got := AKSControlPlane{}.Run(context.Background(), &fakeSource{
		azure: &fakeAzure{subscription: "s", cluster: &pkgazure.AKSCluster{Name: "prod", ResourceGroup: "rg", ProvisioningState: "Succeeded", PowerState: "Stopped"}},
	})
	if got.Component.Status != "CRITICAL" {
		t.Errorf("status=%s want CRITICAL (stopped)", got.Component.Status)
	}
	if !strings.Contains(got.Findings[0].Message, "Stopped") {
		t.Errorf("message lacks 'Stopped': %s", got.Findings[0].Message)
	}
}

func TestAKS_NotFound_Critical(t *testing.T) {
	t.Setenv(aksClusterEnv, "ghost")
	got := AKSControlPlane{}.Run(context.Background(), &fakeSource{azure: &fakeAzure{subscription: "s", cluster: nil}})
	if got.Component.Status != "CRITICAL" {
		t.Errorf("status=%s want CRITICAL (not found)", got.Component.Status)
	}
}

func TestAKS_ProbeFailed(t *testing.T) {
	t.Setenv(aksClusterEnv, "prod")
	got := AKSControlPlane{}.Run(context.Background(), &fakeSource{azure: &fakeAzure{subscription: "s", clusterErr: errors.New("403")}})
	if got.Component.Status != "PROBE_FAILED" {
		t.Errorf("status=%s want PROBE_FAILED", got.Component.Status)
	}
}

// --- AKSNodePools ---

func TestAKSNodePools_Healthy(t *testing.T) {
	t.Setenv(aksClusterEnv, "prod")
	got := AKSNodePools{}.Run(context.Background(), &fakeSource{
		azure: &fakeAzure{subscription: "s", nodePools: []pkgazure.AKSNodePool{{Name: "system", ClusterName: "prod", ProvisioningState: "Succeeded", PowerState: "Running"}}},
	})
	if got.Component.Status != "HEALTHY" {
		t.Errorf("status=%s want HEALTHY", got.Component.Status)
	}
}

func TestAKSNodePools_Failed_Critical(t *testing.T) {
	t.Setenv(aksClusterEnv, "prod")
	got := AKSNodePools{}.Run(context.Background(), &fakeSource{
		azure: &fakeAzure{subscription: "s", nodePools: []pkgazure.AKSNodePool{{Name: "gpu", ClusterName: "prod", ProvisioningState: "Failed"}}},
	})
	if got.Component.Status != "CRITICAL" {
		t.Errorf("status=%s want CRITICAL", got.Component.Status)
	}
}

// --- ManagedIdentities ---

func TestManagedIdentities_WithRoles_Silent(t *testing.T) {
	got := ManagedIdentities{}.Run(context.Background(), &fakeSource{
		azure: &fakeAzure{subscription: "s", identities: []pkgazure.ManagedIdentity{{Name: "wi", ResourceGroup: "rg", RoleAssignmentN: 2}}},
	})
	if got.Component.Status != "HEALTHY" {
		t.Errorf("status=%s want HEALTHY", got.Component.Status)
	}
}

func TestManagedIdentities_NoRoles_Warning(t *testing.T) {
	got := ManagedIdentities{}.Run(context.Background(), &fakeSource{
		azure: &fakeAzure{subscription: "s", identities: []pkgazure.ManagedIdentity{{Name: "orphan", ResourceGroup: "rg", RoleAssignmentN: 0}}},
	})
	if got.Component.Status != "DEGRADED" || got.Findings[0].Severity != probe.SeverityWarning {
		t.Errorf("zero-role identity should be warning; got: %+v", got.Findings)
	}
}

// --- Subnets ---

func TestAzureSubnets_Healthy(t *testing.T) {
	got := Subnets{}.Run(context.Background(), &fakeSource{
		azure: &fakeAzure{subscription: "s", subnets: []pkgazure.Subnet{{Name: "main", VNet: "vnet", TotalIPCount: 1000, AvailableIPCount: 900}}},
	})
	if got.Component.Status != "HEALTHY" {
		t.Errorf("status=%s want HEALTHY", got.Component.Status)
	}
}

func TestAzureSubnets_Warn(t *testing.T) {
	got := Subnets{}.Run(context.Background(), &fakeSource{
		azure: &fakeAzure{subscription: "s", subnets: []pkgazure.Subnet{{Name: "main", VNet: "vnet", TotalIPCount: 1000, AvailableIPCount: 200}}},
	})
	if got.Component.Status != "DEGRADED" {
		t.Errorf("status=%s want DEGRADED (20%% free)", got.Component.Status)
	}
}

func TestAzureSubnets_Critical(t *testing.T) {
	got := Subnets{}.Run(context.Background(), &fakeSource{
		azure: &fakeAzure{subscription: "s", subnets: []pkgazure.Subnet{{Name: "main", VNet: "vnet", TotalIPCount: 1000, AvailableIPCount: 50}}},
	})
	if got.Component.Status != "CRITICAL" {
		t.Errorf("status=%s want CRITICAL (5%% free)", got.Component.Status)
	}
}

func TestAzureSubnets_ZeroTotal_Skipped(t *testing.T) {
	got := Subnets{}.Run(context.Background(), &fakeSource{
		azure: &fakeAzure{subscription: "s", subnets: []pkgazure.Subnet{{Name: "weird", VNet: "vnet", TotalIPCount: 0}}},
	})
	if len(got.Findings) != 0 {
		t.Errorf("zero-total subnet should be skipped; got: %+v", got.Findings)
	}
}

func TestAzureBatch2_NamesStable(t *testing.T) {
	cases := map[string]string{
		AKSControlPlane{}.Name():   "azure-aks-control-plane",
		AKSNodePools{}.Name():      "azure-aks-nodepools",
		ManagedIdentities{}.Name(): "azure-managed-identities",
		Subnets{}.Name():           "azure-subnets",
	}
	for got, want := range cases {
		if got != want {
			t.Errorf("Name()=%q want %q", got, want)
		}
	}
}
