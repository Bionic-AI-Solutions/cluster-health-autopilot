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
	// Workload Identity SA: OAuth2Bound=true, no user-managed keys — the
	// ideal posture. Must produce no findings.
	got := IAMServiceAccounts{}.Run(context.Background(), &fakeSource{
		gcp: &fakeGCP{project: "p", serviceAccounts: []pkggcp.ServiceAccount{
			{Email: "wi@p.iam", Disabled: false, KeyCount: 0, OAuth2Bound: true},
		}},
	})
	if got.Component.Status != "HEALTHY" {
		t.Errorf("status=%s want HEALTHY", got.Component.Status)
	}
}

func TestIAMSA_KeylessNoBinding_Silent(t *testing.T) {
	// Keyless, enabled SA with NO detected Workload Identity binding
	// (OAuth2Bound=false). This is the case that previously false-
	// positived: it includes both correctly-configured keyless WI SAs
	// whose IAM policy we couldn't read, and SAs awaiting binding. The
	// probe must stay silent — never page on the absence of a binding.
	got := IAMServiceAccounts{}.Run(context.Background(), &fakeSource{
		gcp: &fakeGCP{project: "p", serviceAccounts: []pkggcp.ServiceAccount{
			{Email: "keyless@p.iam", Disabled: false, KeyCount: 0, OAuth2Bound: false},
		}},
	})
	if len(got.Findings) != 0 {
		t.Errorf("keyless SA with no detected WI binding must not be flagged; got: %+v", got.Findings)
	}
	if got.Component.Status != "HEALTHY" {
		t.Errorf("status=%s want HEALTHY", got.Component.Status)
	}
}

func TestIAMSA_MixedConfig_Warning(t *testing.T) {
	// WI binding detected (OAuth2Bound=true) AND user-managed key(s)
	// present: a genuine mixed-mode finding (positive signal).
	got := IAMServiceAccounts{}.Run(context.Background(), &fakeSource{
		gcp: &fakeGCP{project: "p", serviceAccounts: []pkggcp.ServiceAccount{
			{Email: "mixed@p.iam", Disabled: false, KeyCount: 1, OAuth2Bound: true},
		}},
	})
	if got.Component.Status != "DEGRADED" {
		t.Errorf("status=%s want DEGRADED (mixed config)", got.Component.Status)
	}
	if len(got.Findings) != 1 || got.Findings[0].Severity != probe.SeverityWarning {
		t.Fatalf("expected 1 warning; got: %+v", got.Findings)
	}
	if !strings.Contains(got.Findings[0].Message, "mixed configuration") {
		t.Errorf("message lacks 'mixed configuration': %s", got.Findings[0].Message)
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

// Capacity-only contract (O7): unmeasured subnets (live mode) with a
// small primary CIDR are flagged — the IP-exhaustion precondition the
// probe CAN see without the Recommender API.
func TestSubnets_Unmeasured_SmallCIDR_Warns(t *testing.T) {
	got := Subnets{}.Run(context.Background(), &fakeSource{
		gcp: &fakeGCP{project: "p", subnets: []pkggcp.Subnet{
			{Name: "tiny", Region: "us-central1", IPCIDRRange: "10.0.0.0/28", TotalIPCount: 12, AvailableIPCount: -1},
		}},
	})
	if len(got.Findings) != 1 {
		t.Fatalf("expected 1 small-capacity warning; got: %+v", got.Findings)
	}
	if got.Findings[0].Severity != probe.SeverityWarning {
		t.Errorf("severity=%s want warning", got.Findings[0].Severity)
	}
	if !strings.Contains(got.Findings[0].Message, "Network Analyzer") {
		t.Errorf("message must point at Network Analyzer for real utilization; got: %q", got.Findings[0].Message)
	}
}

func TestSubnets_Unmeasured_LargeCIDR_Silent(t *testing.T) {
	got := Subnets{}.Run(context.Background(), &fakeSource{
		gcp: &fakeGCP{project: "p", subnets: []pkggcp.Subnet{
			{Name: "big", Region: "us-central1", IPCIDRRange: "10.0.0.0/24", TotalIPCount: 252, AvailableIPCount: -1},
		}},
	})
	if len(got.Findings) != 0 {
		t.Errorf("unmeasured /24 should not warn; got: %+v", got.Findings)
	}
	if !strings.Contains(got.Component.Detail, "capacity-only") {
		t.Errorf("detail should state the capacity-only contract; got: %q", got.Component.Detail)
	}
}

func TestSubnets_SmallPrefixThreshold_Configurable(t *testing.T) {
	// /27 is small under the default (/26) but fine when the operator
	// relaxes the threshold to /28.
	subnets := []pkggcp.Subnet{
		{Name: "tiny", Region: "us-central1", IPCIDRRange: "10.0.0.0/27", TotalIPCount: 28, AvailableIPCount: -1},
	}
	got := Subnets{}.Run(context.Background(), &fakeSource{gcp: &fakeGCP{project: "p", subnets: subnets}})
	if len(got.Findings) != 1 {
		t.Errorf("default threshold (/26) should flag a /27; got: %+v", got.Findings)
	}
	got = Subnets{SmallPrefixThreshold: 28}.Run(context.Background(), &fakeSource{gcp: &fakeGCP{project: "p", subnets: subnets}})
	if len(got.Findings) != 0 {
		t.Errorf("threshold /28 should not flag a /27; got: %+v", got.Findings)
	}
}

func TestCIDRPrefix(t *testing.T) {
	cases := map[string]int{
		"10.0.0.0/28": 28,
		"10.0.0.0/8":  8,
		"":            -1,
		"10.0.0.0":    -1,
		"10.0.0.0/99": -1,
	}
	for in, want := range cases {
		if got := cidrPrefix(in); got != want {
			t.Errorf("cidrPrefix(%q)=%d want %d", in, got, want)
		}
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
