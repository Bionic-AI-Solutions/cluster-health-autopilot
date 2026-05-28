// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"
	"strings"
	"testing"
	"time"

	pkgazure "github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/cloud/azure"
)

// --- AppGatewayBackends ---

func TestAppGW_SkippedWhenMissing(t *testing.T) {
	if (AppGatewayBackends{}).Run(context.Background(), &fakeSource{}).Component.Status != "SKIPPED" {
		t.Error("want SKIPPED")
	}
}

func TestAppGW_AllHealthy_Silent(t *testing.T) {
	got := AppGatewayBackends{}.Run(context.Background(), &fakeSource{
		azure: &fakeAzure{subscription: "s", appgw: []pkgazure.AppGatewayBackend{{Gateway: "gw", PoolName: "web", HealthyCount: 3, TotalCount: 3}}},
	})
	if got.Component.Status != "HEALTHY" {
		t.Errorf("status=%s want HEALTHY", got.Component.Status)
	}
}

func TestAppGW_ZeroHealthy_Critical(t *testing.T) {
	got := AppGatewayBackends{}.Run(context.Background(), &fakeSource{
		azure: &fakeAzure{subscription: "s", appgw: []pkgazure.AppGatewayBackend{{Gateway: "gw", PoolName: "web", HealthyCount: 0, UnhealthyCount: 2, TotalCount: 2}}},
	})
	if got.Component.Status != "CRITICAL" {
		t.Errorf("status=%s want CRITICAL", got.Component.Status)
	}
}

func TestAppGW_Partial_Warning(t *testing.T) {
	got := AppGatewayBackends{}.Run(context.Background(), &fakeSource{
		azure: &fakeAzure{subscription: "s", appgw: []pkgazure.AppGatewayBackend{{Gateway: "gw", PoolName: "web", HealthyCount: 2, UnhealthyCount: 1, TotalCount: 3}}},
	})
	if got.Component.Status != "DEGRADED" {
		t.Errorf("status=%s want DEGRADED", got.Component.Status)
	}
}

// HealthyCount = -1 is the live-wrapper "not measured" sentinel. The
// probe must SKIP the health check (not treat the pool as fully
// healthy and silently never fire), stay HEALTHY, and note the gap.
func TestAppGW_Unmeasured_Skipped(t *testing.T) {
	got := AppGatewayBackends{}.Run(context.Background(), &fakeSource{
		azure: &fakeAzure{subscription: "s", appgw: []pkgazure.AppGatewayBackend{{Gateway: "gw", PoolName: "web", HealthyCount: -1, TotalCount: 4}}},
	})
	if len(got.Findings) != 0 {
		t.Errorf("unmeasured pool should be skipped; got: %+v", got.Findings)
	}
	if got.Component.Status != "HEALTHY" {
		t.Errorf("status=%s want HEALTHY", got.Component.Status)
	}
	if !strings.Contains(got.Component.Detail, "not measured") {
		t.Errorf("detail should note unmeasured pools; got: %q", got.Component.Detail)
	}
}

// --- Certificates ---

func TestAzureCert_IssuedFarExpiry_Silent(t *testing.T) {
	got := Certificates{Now: func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) }}.Run(
		context.Background(), &fakeSource{
			azure: &fakeAzure{subscription: "s", certs: []pkgazure.Certificate{{Name: "c", Issued: true, NotAfter: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)}}},
		})
	if got.Component.Status != "HEALTHY" {
		t.Errorf("status=%s want HEALTHY", got.Component.Status)
	}
}

func TestAzureCert_NotIssued_Critical(t *testing.T) {
	got := Certificates{}.Run(context.Background(), &fakeSource{
		azure: &fakeAzure{subscription: "s", certs: []pkgazure.Certificate{{Name: "c", Issued: false}}},
	})
	if got.Component.Status != "CRITICAL" {
		t.Errorf("status=%s want CRITICAL", got.Component.Status)
	}
}

func TestAzureCert_NearExpiry_Warning(t *testing.T) {
	got := Certificates{Now: func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) }}.Run(
		context.Background(), &fakeSource{
			azure: &fakeAzure{subscription: "s", certs: []pkgazure.Certificate{{Name: "c", Issued: true, NotAfter: time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)}}},
		})
	if got.Component.Status != "DEGRADED" {
		t.Errorf("status=%s want DEGRADED (9d)", got.Component.Status)
	}
}

// --- StoragePublicAccess ---

func TestStorage_Locked_Silent(t *testing.T) {
	got := StoragePublicAccess{}.Run(context.Background(), &fakeSource{
		azure: &fakeAzure{subscription: "s", storage: []pkgazure.StorageAccount{{Name: "sa", AllowBlobPublicAccess: false, HTTPSOnly: true}}},
	})
	if got.Component.Status != "HEALTHY" {
		t.Errorf("status=%s want HEALTHY", got.Component.Status)
	}
}

func TestStorage_PublicBlob_Critical(t *testing.T) {
	got := StoragePublicAccess{}.Run(context.Background(), &fakeSource{
		azure: &fakeAzure{subscription: "s", storage: []pkgazure.StorageAccount{{Name: "sa", AllowBlobPublicAccess: true, HTTPSOnly: true}}},
	})
	if got.Component.Status != "CRITICAL" {
		t.Errorf("status=%s want CRITICAL", got.Component.Status)
	}
	if !strings.Contains(got.Findings[0].Message, "public blob access") {
		t.Errorf("message lacks 'public blob access': %s", got.Findings[0].Message)
	}
}

func TestStorage_NoHTTPS_Warning(t *testing.T) {
	got := StoragePublicAccess{}.Run(context.Background(), &fakeSource{
		azure: &fakeAzure{subscription: "s", storage: []pkgazure.StorageAccount{{Name: "sa", AllowBlobPublicAccess: false, HTTPSOnly: false}}},
	})
	if got.Component.Status != "DEGRADED" {
		t.Errorf("status=%s want DEGRADED", got.Component.Status)
	}
}

// --- KeyVaults ---

func TestKeyVault_Protected_Silent(t *testing.T) {
	got := KeyVaults{}.Run(context.Background(), &fakeSource{
		azure: &fakeAzure{subscription: "s", vaults: []pkgazure.KeyVault{{Name: "kv", SoftDelete: true, PurgeProtection: true}}},
	})
	if got.Component.Status != "HEALTHY" {
		t.Errorf("status=%s want HEALTHY", got.Component.Status)
	}
}

func TestKeyVault_NoSoftDelete_Critical(t *testing.T) {
	got := KeyVaults{}.Run(context.Background(), &fakeSource{
		azure: &fakeAzure{subscription: "s", vaults: []pkgazure.KeyVault{{Name: "kv", SoftDelete: false}}},
	})
	if got.Component.Status != "CRITICAL" {
		t.Errorf("status=%s want CRITICAL", got.Component.Status)
	}
}

func TestKeyVault_NoPurgeProtection_Warning(t *testing.T) {
	got := KeyVaults{}.Run(context.Background(), &fakeSource{
		azure: &fakeAzure{subscription: "s", vaults: []pkgazure.KeyVault{{Name: "kv", SoftDelete: true, PurgeProtection: false}}},
	})
	if got.Component.Status != "DEGRADED" {
		t.Errorf("status=%s want DEGRADED", got.Component.Status)
	}
}

func TestAzureBatch3_NamesStable(t *testing.T) {
	cases := map[string]string{
		AppGatewayBackends{}.Name():  "azure-appgw-backends",
		Certificates{}.Name():        "azure-certs",
		StoragePublicAccess{}.Name(): "azure-storage-public-access",
		KeyVaults{}.Name():           "azure-keyvaults",
	}
	for got, want := range cases {
		if got != want {
			t.Errorf("Name()=%q want %q", got, want)
		}
	}
}
