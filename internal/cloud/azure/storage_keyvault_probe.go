// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"
	"fmt"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/cloud"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/probe"
)

// StoragePublicAccess flags Storage accounts that allow public blob
// access (critical) or permit plaintext HTTP (warning). Mirrors AWS
// S3BucketPublicAccess / GCP GCSPublicAccess.
type StoragePublicAccess struct{}

const storageName = "azure-storage-public-access"

// Name satisfies cloudprobe.Probe.
func (StoragePublicAccess) Name() string { return storageName }

// Run satisfies cloudprobe.Probe.
func (StoragePublicAccess) Run(ctx context.Context, src cloud.Source) probe.Result {
	azClient := src.Azure()
	if azClient == nil {
		return skipped(storageName, "Azure not configured (cloud.azure.enabled=false)")
	}
	accts, err := azClient.ListStorageAccounts(ctx)
	if err != nil {
		return probeFailed(storageName, "storage.ListAccounts", err)
	}

	var findings []probe.Finding
	for _, a := range accts {
		subject := fmt.Sprintf("azure-storage/%s/%s/%s", azClient.SubscriptionID(), a.ResourceGroup, a.Name)
		if a.AllowBlobPublicAccess {
			findings = append(findings, probe.Finding{
				Component:   subject,
				Severity:    probe.SeverityCritical,
				Message:     fmt.Sprintf("Storage account %q allows public blob access (allowBlobPublicAccess=true)", a.Name),
				Remediation: fmt.Sprintf("az storage account update -n %s -g %s --allow-blob-public-access false --subscription %s", a.Name, a.ResourceGroup, azClient.SubscriptionID()),
			})
		}
		if !a.HTTPSOnly {
			findings = append(findings, probe.Finding{
				Component:   subject,
				Severity:    probe.SeverityWarning,
				Message:     fmt.Sprintf("Storage account %q permits plaintext HTTP (supportsHttpsTrafficOnly=false)", a.Name),
				Remediation: fmt.Sprintf("az storage account update -n %s -g %s --https-only true --subscription %s", a.Name, a.ResourceGroup, azClient.SubscriptionID()),
			})
		}
	}

	return probe.Result{
		Component: probe.ComponentResult{Component: storageName, Status: rollupStatus(findings), Detail: fmt.Sprintf("%d storage account(s) inspected in subscription %s", len(accts), azClient.SubscriptionID())},
		Findings:  findings,
	}
}

// KeyVaults flags Key Vaults missing data-protection guardrails:
//   - no soft-delete → critical (deleted secrets are unrecoverable)
//   - soft-delete but no purge-protection → warning (an attacker /
//     mistake can still purge within the window)
//
// Mirrors AWS KMSKeys / GCP KMSKeys (Azure's risk surface is the
// vault's recovery posture, not per-key state).
type KeyVaults struct{}

const keyVaultsName = "azure-keyvaults"

// Name satisfies cloudprobe.Probe.
func (KeyVaults) Name() string { return keyVaultsName }

// Run satisfies cloudprobe.Probe.
func (KeyVaults) Run(ctx context.Context, src cloud.Source) probe.Result {
	azClient := src.Azure()
	if azClient == nil {
		return skipped(keyVaultsName, "Azure not configured (cloud.azure.enabled=false)")
	}
	vaults, err := azClient.ListKeyVaults(ctx)
	if err != nil {
		return probeFailed(keyVaultsName, "keyvault.ListVaults", err)
	}

	var findings []probe.Finding
	for _, v := range vaults {
		subject := fmt.Sprintf("azure-keyvault/%s/%s/%s", azClient.SubscriptionID(), v.ResourceGroup, v.Name)
		if !v.SoftDelete {
			findings = append(findings, probe.Finding{
				Component:   subject,
				Severity:    probe.SeverityCritical,
				Message:     fmt.Sprintf("Key Vault %q has soft-delete disabled; deleted secrets/keys are unrecoverable", v.Name),
				Remediation: fmt.Sprintf("az keyvault update -n %s -g %s --enable-soft-delete true --subscription %s", v.Name, v.ResourceGroup, azClient.SubscriptionID()),
			})
			continue
		}
		if !v.PurgeProtection {
			findings = append(findings, probe.Finding{
				Component:   subject,
				Severity:    probe.SeverityWarning,
				Message:     fmt.Sprintf("Key Vault %q has soft-delete but no purge-protection; secrets can still be purged within the retention window", v.Name),
				Remediation: fmt.Sprintf("az keyvault update -n %s -g %s --enable-purge-protection true --subscription %s", v.Name, v.ResourceGroup, azClient.SubscriptionID()),
			})
		}
	}

	return probe.Result{
		Component: probe.ComponentResult{Component: keyVaultsName, Status: rollupStatus(findings), Detail: fmt.Sprintf("%d Key Vault(s) inspected in subscription %s", len(vaults), azClient.SubscriptionID())},
		Findings:  findings,
	}
}
