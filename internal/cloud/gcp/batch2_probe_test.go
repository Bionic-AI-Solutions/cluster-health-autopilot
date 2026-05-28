// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package gcp

import (
	"context"
	"errors"
	"strings"
	"testing"

	pkggcp "github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/cloud/gcp"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/probe"
)

// --- GKEControlPlane ---

func TestGKEControlPlane_SkippedWhenGCPMissing(t *testing.T) {
	got := GKEControlPlane{}.Run(context.Background(), &fakeSource{})
	if got.Component.Status != "SKIPPED" {
		t.Errorf("status=%s want SKIPPED", got.Component.Status)
	}
}

func TestGKEControlPlane_SkippedWhenEnvUnset(t *testing.T) {
	t.Setenv(gkeClusterEnv, "")
	got := GKEControlPlane{}.Run(context.Background(), &fakeSource{gcp: &fakeGCP{project: "p"}})
	if got.Component.Status != "SKIPPED" {
		t.Errorf("status=%s want SKIPPED (env unset)", got.Component.Status)
	}
}

func TestGKEControlPlane_Running_Healthy(t *testing.T) {
	t.Setenv(gkeClusterEnv, "prod")
	got := GKEControlPlane{}.Run(context.Background(), &fakeSource{
		gcp: &fakeGCP{project: "p", cluster: &pkggcp.GKECluster{Name: "prod", Status: "RUNNING", Location: "us-central1"}},
	})
	if got.Component.Status != "HEALTHY" {
		t.Errorf("status=%s want HEALTHY", got.Component.Status)
	}
}

func TestGKEControlPlane_Error_Critical(t *testing.T) {
	t.Setenv(gkeClusterEnv, "prod")
	got := GKEControlPlane{}.Run(context.Background(), &fakeSource{
		gcp: &fakeGCP{project: "p", cluster: &pkggcp.GKECluster{Name: "prod", Status: "ERROR", Location: "us-central1"}},
	})
	if got.Component.Status != "CRITICAL" {
		t.Errorf("status=%s want CRITICAL", got.Component.Status)
	}
}

func TestGKEControlPlane_NotFound_Critical(t *testing.T) {
	t.Setenv(gkeClusterEnv, "ghost")
	got := GKEControlPlane{}.Run(context.Background(), &fakeSource{
		gcp: &fakeGCP{project: "p", cluster: nil},
	})
	if got.Component.Status != "CRITICAL" {
		t.Errorf("status=%s want CRITICAL (cluster not found)", got.Component.Status)
	}
	if !strings.Contains(got.Findings[0].Message, "does not exist") {
		t.Errorf("message lacks 'does not exist': %s", got.Findings[0].Message)
	}
}

func TestGKEControlPlane_ProbeFailed(t *testing.T) {
	t.Setenv(gkeClusterEnv, "prod")
	got := GKEControlPlane{}.Run(context.Background(), &fakeSource{
		gcp: &fakeGCP{project: "p", clusterErr: errors.New("403")},
	})
	if got.Component.Status != "PROBE_FAILED" {
		t.Errorf("status=%s want PROBE_FAILED", got.Component.Status)
	}
}

// --- GKENodePools ---

func TestGKENodePools_RunningHealthy(t *testing.T) {
	t.Setenv(gkeClusterEnv, "prod")
	got := GKENodePools{}.Run(context.Background(), &fakeSource{
		gcp: &fakeGCP{project: "p", nodePools: []pkggcp.GKENodePool{
			{Name: "default", ClusterName: "prod", Status: "RUNNING"},
		}},
	})
	if got.Component.Status != "HEALTHY" {
		t.Errorf("status=%s want HEALTHY", got.Component.Status)
	}
}

func TestGKENodePools_RunningWithError_Critical(t *testing.T) {
	t.Setenv(gkeClusterEnv, "prod")
	got := GKENodePools{}.Run(context.Background(), &fakeSource{
		gcp: &fakeGCP{project: "p", nodePools: []pkggcp.GKENodePool{
			{Name: "gpu", ClusterName: "prod", Status: "RUNNING_WITH_ERROR"},
		}},
	})
	if got.Component.Status != "CRITICAL" {
		t.Errorf("status=%s want CRITICAL", got.Component.Status)
	}
}

// --- IAMServiceAccounts ---

func TestIAMSA_HealthySilent(t *testing.T) {
	got := IAMServiceAccounts{}.Run(context.Background(), &fakeSource{
		gcp: &fakeGCP{project: "p", serviceAccounts: []pkggcp.ServiceAccount{
			{Email: "wi@p.iam", Disabled: false, KeyCount: 0},
		}},
	})
	if got.Component.Status != "HEALTHY" {
		t.Errorf("status=%s want HEALTHY", got.Component.Status)
	}
}

func TestIAMSA_DisabledWithKeys_Warning(t *testing.T) {
	got := IAMServiceAccounts{}.Run(context.Background(), &fakeSource{
		gcp: &fakeGCP{project: "p", serviceAccounts: []pkggcp.ServiceAccount{
			{Email: "old@p.iam", Disabled: true, KeyCount: 1},
		}},
	})
	if got.Component.Status != "DEGRADED" || got.Findings[0].Severity != probe.SeverityWarning {
		t.Errorf("disabled-with-keys should be warning; got: %+v", got.Findings)
	}
}

func TestIAMSA_KeySprawl_Warning(t *testing.T) {
	got := IAMServiceAccounts{}.Run(context.Background(), &fakeSource{
		gcp: &fakeGCP{project: "p", serviceAccounts: []pkggcp.ServiceAccount{
			{Email: "sprawl@p.iam", Disabled: false, KeyCount: 5},
		}},
	})
	if got.Component.Status != "DEGRADED" {
		t.Errorf("status=%s want DEGRADED (key sprawl)", got.Component.Status)
	}
	if !strings.Contains(got.Findings[0].Message, "sprawl") {
		t.Errorf("message lacks 'sprawl': %s", got.Findings[0].Message)
	}
}

func TestIAMSA_TwoKeys_Silent(t *testing.T) {
	// 2 keys == rotation overlap, at the threshold but not over.
	got := IAMServiceAccounts{}.Run(context.Background(), &fakeSource{
		gcp: &fakeGCP{project: "p", serviceAccounts: []pkggcp.ServiceAccount{
			{Email: "rotating@p.iam", Disabled: false, KeyCount: 2},
		}},
	})
	if len(got.Findings) != 0 {
		t.Errorf("2 keys is within threshold; got: %+v", got.Findings)
	}
}

// --- Subnets ---

func TestSubnets_HealthySilent(t *testing.T) {
	got := Subnets{}.Run(context.Background(), &fakeSource{
		gcp: &fakeGCP{project: "p", subnets: []pkggcp.Subnet{
			{Name: "main", Region: "us-central1", TotalIPCount: 1000, AvailableIPCount: 800},
		}},
	})
	if got.Component.Status != "HEALTHY" {
		t.Errorf("status=%s want HEALTHY (80%% free)", got.Component.Status)
	}
}

func TestSubnets_Warn_Below25Percent(t *testing.T) {
	got := Subnets{}.Run(context.Background(), &fakeSource{
		gcp: &fakeGCP{project: "p", subnets: []pkggcp.Subnet{
			{Name: "main", Region: "us-central1", TotalIPCount: 1000, AvailableIPCount: 200},
		}},
	})
	if got.Component.Status != "DEGRADED" {
		t.Errorf("status=%s want DEGRADED (20%% free)", got.Component.Status)
	}
}

func TestSubnets_Critical_Below10Percent(t *testing.T) {
	got := Subnets{}.Run(context.Background(), &fakeSource{
		gcp: &fakeGCP{project: "p", subnets: []pkggcp.Subnet{
			{Name: "main", Region: "us-central1", TotalIPCount: 1000, AvailableIPCount: 50},
		}},
	})
	if got.Component.Status != "CRITICAL" {
		t.Errorf("status=%s want CRITICAL (5%% free)", got.Component.Status)
	}
}

func TestSubnets_ZeroTotal_Skipped(t *testing.T) {
	got := Subnets{}.Run(context.Background(), &fakeSource{
		gcp: &fakeGCP{project: "p", subnets: []pkggcp.Subnet{
			{Name: "weird", Region: "us-central1", TotalIPCount: 0, AvailableIPCount: 0},
		}},
	})
	if len(got.Findings) != 0 {
		t.Errorf("zero-total subnet should be skipped (no div-by-zero); got: %+v", got.Findings)
	}
}

// AvailableIPCount = -1 is the live-wrapper "not measured" sentinel.
// The probe must SKIP the IP check (not treat it as 100% free and
// silently never fire), stay HEALTHY, and note the gap in Detail.
func TestSubnets_Unmeasured_Skipped(t *testing.T) {
	got := Subnets{}.Run(context.Background(), &fakeSource{
		gcp: &fakeGCP{project: "p", subnets: []pkggcp.Subnet{
			{Name: "live", Region: "us-central1", TotalIPCount: 1000, AvailableIPCount: -1},
		}},
	})
	if len(got.Findings) != 0 {
		t.Errorf("unmeasured subnet should be skipped; got: %+v", got.Findings)
	}
	if got.Component.Status != "HEALTHY" {
		t.Errorf("status=%s want HEALTHY", got.Component.Status)
	}
	if !strings.Contains(got.Component.Detail, "not measured") {
		t.Errorf("detail should note unmeasured subnets; got: %q", got.Component.Detail)
	}
}

func TestBatch2_NamesStable(t *testing.T) {
	cases := map[string]string{
		GKEControlPlane{}.Name():    "gcp-gke-control-plane",
		GKENodePools{}.Name():       "gcp-gke-nodepools",
		IAMServiceAccounts{}.Name(): "gcp-iam-serviceaccounts",
		Subnets{}.Name():            "gcp-subnets",
	}
	for got, want := range cases {
		if got != want {
			t.Errorf("Name()=%q want %q", got, want)
		}
	}
}
