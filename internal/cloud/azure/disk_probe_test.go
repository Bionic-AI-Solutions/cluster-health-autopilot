// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	pkgazure "github.com/srenix-ai/agentic-sre/pkg/cloud/azure"
	"github.com/srenix-ai/agentic-sre/pkg/probe"
)

func runDisks(disks ...pkgazure.Disk) probe.Result {
	return Disks{}.Run(context.Background(), &fakeSource{
		azure: &fakeAzure{subscription: "sub-1", location: "eastus", disks: disks},
	})
}

func TestAzureDisks_SkippedWhenMissing(t *testing.T) {
	got := Disks{}.Run(context.Background(), &fakeSource{})
	if got.Component.Status != "SKIPPED" {
		t.Errorf("status=%s want SKIPPED", got.Component.Status)
	}
}

func TestAzureDisks_ProbeFailedOnError(t *testing.T) {
	got := Disks{}.Run(context.Background(), &fakeSource{
		azure: &fakeAzure{subscription: "sub-1", disksErr: errors.New("rate limit")},
	})
	if got.Component.Status != "PROBE_FAILED" {
		t.Errorf("status=%s want PROBE_FAILED", got.Component.Status)
	}
}

func TestAzureDisks_AttachedHealthy_Silent(t *testing.T) {
	got := runDisks(pkgazure.Disk{
		Name: "vm-disk-1", ResourceGroup: "rg",
		ProvisioningState: "Succeeded", DiskState: "Attached",
		AttachedToVM: "vm-1", SizeGB: 100, SKU: "Premium_LRS",
	})
	if len(got.Findings) != 0 {
		t.Errorf("attached healthy disk should be silent; got: %+v", got.Findings)
	}
}

func TestAzureDisks_Failed_Critical(t *testing.T) {
	got := runDisks(pkgazure.Disk{
		Name: "broken-1", ResourceGroup: "rg",
		ProvisioningState: "Failed", SizeGB: 100, SKU: "Premium_LRS",
	})
	if len(got.Findings) != 1 || got.Findings[0].Severity != probe.SeverityCritical {
		t.Errorf("Failed should be critical; got: %+v", got.Findings)
	}
	if !strings.Contains(got.Findings[0].Component, "azure-disk/sub-1/rg/broken-1") {
		t.Errorf("subject=%s lacks azure-disk/...", got.Findings[0].Component)
	}
}

func TestAzureDisks_StuckCreating_Warning(t *testing.T) {
	got := runDisks(pkgazure.Disk{
		Name: "stuck-1", ResourceGroup: "rg",
		ProvisioningState: "Creating", SizeGB: 100,
		CreatedAt: time.Now().Add(-2 * time.Hour),
	})
	if len(got.Findings) != 1 || got.Findings[0].Severity != probe.SeverityWarning {
		t.Errorf("stuck Creating should be warning; got: %+v", got.Findings)
	}
	if !strings.Contains(got.Findings[0].Message, "stuck in Creating") {
		t.Errorf("message lacks 'stuck in Creating': %s", got.Findings[0].Message)
	}
}

func TestAzureDisks_FreshlyCreating_Silent(t *testing.T) {
	got := runDisks(pkgazure.Disk{
		Name: "new-1", ResourceGroup: "rg",
		ProvisioningState: "Creating", SizeGB: 100,
		CreatedAt: time.Now().Add(-15 * time.Minute),
	})
	if len(got.Findings) != 0 {
		t.Errorf("fresh Creating within grace should be silent; got: %+v", got.Findings)
	}
}

func TestAzureDisks_DetachedFresh_Silent(t *testing.T) {
	got := runDisks(pkgazure.Disk{
		Name: "orphan-1", ResourceGroup: "rg",
		ProvisioningState: "Succeeded", DiskState: "Unattached",
		AttachedToVM: "", SizeGB: 50,
		DetachedDuration: 2 * time.Hour,
	})
	if len(got.Findings) != 0 {
		t.Errorf("freshly-detached should be silent; got: %+v", got.Findings)
	}
}

func TestAzureDisks_DetachedPastGrace_Warning(t *testing.T) {
	got := runDisks(pkgazure.Disk{
		Name: "orphan-1", ResourceGroup: "rg",
		ProvisioningState: "Succeeded", DiskState: "Unattached",
		AttachedToVM: "", SizeGB: 50, SKU: "Standard_LRS",
		DetachedDuration: 48 * time.Hour,
	})
	if len(got.Findings) != 1 || got.Findings[0].Severity != probe.SeverityWarning {
		t.Errorf("48h-detached should be warning; got: %+v", got.Findings)
	}
	if !strings.Contains(got.Findings[0].Message, "billing leak") {
		t.Errorf("message lacks 'billing leak': %s", got.Findings[0].Message)
	}
}

func TestAzureDisks_UnknownState_Warning(t *testing.T) {
	got := runDisks(pkgazure.Disk{
		Name: "weird-1", ResourceGroup: "rg",
		ProvisioningState: "MysteryState", SizeGB: 50,
	})
	if len(got.Findings) != 1 || got.Findings[0].Severity != probe.SeverityWarning {
		t.Errorf("unknown state should be warning; got: %+v", got.Findings)
	}
}

func TestAzureDisks_NameStable(t *testing.T) {
	if n := (Disks{}).Name(); n != "azure-disks" {
		t.Errorf("Name()=%q want azure-disks", n)
	}
}
