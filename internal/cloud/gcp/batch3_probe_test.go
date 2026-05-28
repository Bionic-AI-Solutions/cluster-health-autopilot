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
