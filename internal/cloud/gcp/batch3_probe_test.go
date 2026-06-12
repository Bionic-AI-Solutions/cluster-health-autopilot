// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package gcp

import (
	"context"
	"strings"
	"testing"
	"time"

	pkggcp "github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/cloud/gcp"
)

// --- LoadBalancerBackends ---

func TestLB_SkippedWhenMissing(t *testing.T) {
	if (LoadBalancerBackends{}).Run(context.Background(), &fakeSource{}).Component.Status != "SKIPPED" {
		t.Error("want SKIPPED")
	}
}

func TestLB_AllHealthy_Silent(t *testing.T) {
	got := LoadBalancerBackends{}.Run(context.Background(), &fakeSource{
		gcp: &fakeGCP{project: "p", backends: []pkggcp.BackendService{{Name: "web", HealthyCount: 3, TotalBackends: 3}}},
	})
	if got.Component.Status != "HEALTHY" {
		t.Errorf("status=%s want HEALTHY", got.Component.Status)
	}
}

func TestLB_ZeroHealthy_Critical(t *testing.T) {
	got := LoadBalancerBackends{}.Run(context.Background(), &fakeSource{
		gcp: &fakeGCP{project: "p", backends: []pkggcp.BackendService{{Name: "web", HealthyCount: 0, UnhealthyCount: 3, TotalBackends: 3}}},
	})
	if got.Component.Status != "CRITICAL" {
		t.Errorf("status=%s want CRITICAL", got.Component.Status)
	}
}

// CHA-com RCA join contract (ai/cloudcontext): the 0-healthy message
// carries a " (lb: <forwarding-rule IP or name>)" suffix. See
// internal/cloud/contract_test.go.
func TestLB_ZeroHealthyMessageCarriesForwardingRuleJoinKey(t *testing.T) {
	got := LoadBalancerBackends{}.Run(context.Background(), &fakeSource{
		gcp: &fakeGCP{project: "p", backends: []pkggcp.BackendService{
			{Name: "web", HealthyCount: 0, UnhealthyCount: 3, TotalBackends: 3, ForwardingRule: "203.0.113.7"},
		}},
	})
	if len(got.Findings) != 1 {
		t.Fatalf("want 1 finding got %d", len(got.Findings))
	}
	want := `LB backend service "web" has 0 healthy backends (3 unhealthy); traffic is failing (lb: 203.0.113.7)`
	if got.Findings[0].Message != want {
		t.Errorf("Message=%q want %q", got.Findings[0].Message, want)
	}
}

// No forwarding rule mapped → fall back to the backend-service name as
// the join value (per the CHA-com contract).
func TestLB_ZeroHealthyMessageFallsBackToBackendServiceName(t *testing.T) {
	got := LoadBalancerBackends{}.Run(context.Background(), &fakeSource{
		gcp: &fakeGCP{project: "p", backends: []pkggcp.BackendService{
			{Name: "web", HealthyCount: 0, UnhealthyCount: 3, TotalBackends: 3},
		}},
	})
	if len(got.Findings) != 1 {
		t.Fatalf("want 1 finding got %d", len(got.Findings))
	}
	want := `LB backend service "web" has 0 healthy backends (3 unhealthy); traffic is failing (lb: web)`
	if got.Findings[0].Message != want {
		t.Errorf("Message=%q want %q", got.Findings[0].Message, want)
	}
}

// Guard: if both forwarding rule and name are somehow empty, the suffix
// is omitted entirely — never an empty "(lb: )".
func TestLB_ZeroHealthyMessageNoEmptyLBSuffix(t *testing.T) {
	got := LoadBalancerBackends{}.Run(context.Background(), &fakeSource{
		gcp: &fakeGCP{project: "p", backends: []pkggcp.BackendService{
			{HealthyCount: 0, UnhealthyCount: 3, TotalBackends: 3},
		}},
	})
	if len(got.Findings) != 1 {
		t.Fatalf("want 1 finding got %d", len(got.Findings))
	}
	if strings.Contains(got.Findings[0].Message, "(lb:") {
		t.Errorf("empty join value must omit the suffix; got %q", got.Findings[0].Message)
	}
}

func TestLB_PartialUnhealthy_Warning(t *testing.T) {
	got := LoadBalancerBackends{}.Run(context.Background(), &fakeSource{
		gcp: &fakeGCP{project: "p", backends: []pkggcp.BackendService{{Name: "web", HealthyCount: 2, UnhealthyCount: 1, TotalBackends: 3}}},
	})
	if got.Component.Status != "DEGRADED" {
		t.Errorf("status=%s want DEGRADED", got.Component.Status)
	}
}

// --- ManagedCertificates ---

func TestCert_ActiveFarExpiry_Silent(t *testing.T) {
	got := ManagedCertificates{Now: func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) }}.Run(
		context.Background(), &fakeSource{
			gcp: &fakeGCP{project: "p", certs: []pkggcp.ManagedCertificate{{Name: "c", Status: "ACTIVE", NotAfter: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)}}},
		})
	if got.Component.Status != "HEALTHY" {
		t.Errorf("status=%s want HEALTHY", got.Component.Status)
	}
}

func TestCert_ActiveNearExpiry_Warning(t *testing.T) {
	got := ManagedCertificates{Now: func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) }}.Run(
		context.Background(), &fakeSource{
			gcp: &fakeGCP{project: "p", certs: []pkggcp.ManagedCertificate{{Name: "c", Status: "ACTIVE", NotAfter: time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)}}},
		})
	if got.Component.Status != "DEGRADED" {
		t.Errorf("status=%s want DEGRADED (9d to expiry)", got.Component.Status)
	}
}

func TestCert_ProvisioningFailed_Critical(t *testing.T) {
	got := ManagedCertificates{}.Run(context.Background(), &fakeSource{
		gcp: &fakeGCP{project: "p", certs: []pkggcp.ManagedCertificate{{Name: "c", Status: "PROVISIONING_FAILED_PERMANENTLY"}}},
	})
	if got.Component.Status != "CRITICAL" {
		t.Errorf("status=%s want CRITICAL", got.Component.Status)
	}
}

// --- GCSPublicAccess ---

func TestGCS_Enforced_Silent(t *testing.T) {
	got := GCSPublicAccess{}.Run(context.Background(), &fakeSource{
		gcp: &fakeGCP{project: "p", buckets: []pkggcp.Bucket{{Name: "b", PublicAccessPrevention: "enforced", UniformBucketLevelAccess: true}}},
	})
	if got.Component.Status != "HEALTHY" {
		t.Errorf("status=%s want HEALTHY", got.Component.Status)
	}
}

func TestGCS_AllUsersBinding_Critical(t *testing.T) {
	got := GCSPublicAccess{}.Run(context.Background(), &fakeSource{
		gcp: &fakeGCP{project: "p", buckets: []pkggcp.Bucket{{Name: "public-bucket", HasAllUsersBinding: true}}},
	})
	if got.Component.Status != "CRITICAL" {
		t.Errorf("status=%s want CRITICAL", got.Component.Status)
	}
	if !strings.Contains(got.Findings[0].Message, "public access") {
		t.Errorf("message lacks 'public access': %s", got.Findings[0].Message)
	}
}

func TestGCS_NotEnforced_Warning(t *testing.T) {
	got := GCSPublicAccess{}.Run(context.Background(), &fakeSource{
		gcp: &fakeGCP{project: "p", buckets: []pkggcp.Bucket{{Name: "b", PublicAccessPrevention: "inherited"}}},
	})
	if got.Component.Status != "DEGRADED" {
		t.Errorf("status=%s want DEGRADED", got.Component.Status)
	}
}

// --- KMSKeys ---

func TestKMS_EnabledRotating_Silent(t *testing.T) {
	got := KMSKeys{}.Run(context.Background(), &fakeSource{
		gcp: &fakeGCP{project: "p", kmsKeys: []pkggcp.KMSKey{{Name: "k", PrimaryState: "ENABLED", RotationScheduled: true}}},
	})
	if got.Component.Status != "HEALTHY" {
		t.Errorf("status=%s want HEALTHY", got.Component.Status)
	}
}

func TestKMS_EnabledNoRotation_Warning(t *testing.T) {
	got := KMSKeys{}.Run(context.Background(), &fakeSource{
		gcp: &fakeGCP{project: "p", kmsKeys: []pkggcp.KMSKey{{Name: "k", PrimaryState: "ENABLED", RotationScheduled: false}}},
	})
	if got.Component.Status != "DEGRADED" {
		t.Errorf("status=%s want DEGRADED (no rotation)", got.Component.Status)
	}
}

func TestKMS_Disabled_Warning(t *testing.T) {
	got := KMSKeys{}.Run(context.Background(), &fakeSource{
		gcp: &fakeGCP{project: "p", kmsKeys: []pkggcp.KMSKey{{Name: "k", PrimaryState: "DISABLED"}}},
	})
	if got.Component.Status != "DEGRADED" {
		t.Errorf("status=%s want DEGRADED", got.Component.Status)
	}
}

func TestKMS_DestroyScheduled_Critical(t *testing.T) {
	got := KMSKeys{}.Run(context.Background(), &fakeSource{
		gcp: &fakeGCP{project: "p", kmsKeys: []pkggcp.KMSKey{{Name: "k", PrimaryState: "DESTROY_SCHEDULED"}}},
	})
	if got.Component.Status != "CRITICAL" {
		t.Errorf("status=%s want CRITICAL", got.Component.Status)
	}
}

func TestBatch3_NamesStable(t *testing.T) {
	cases := map[string]string{
		LoadBalancerBackends{}.Name(): "gcp-lb-backends",
		ManagedCertificates{}.Name():  "gcp-managed-certs",
		GCSPublicAccess{}.Name():      "gcp-gcs-public-access",
		KMSKeys{}.Name():              "gcp-kms",
	}
	for got, want := range cases {
		if got != want {
			t.Errorf("Name()=%q want %q", got, want)
		}
	}
}
