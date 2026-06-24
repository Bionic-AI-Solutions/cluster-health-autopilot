// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"
	"fmt"
	"log"
	"time"

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

// kvWarnWindow is the lookahead window for key/secret expiry warnings.
// Keys/secrets expiring within this window generate a Warning finding;
// already-expired items generate a Critical finding.
const kvWarnWindow = 14 * 24 * time.Hour

// KeyVaults checks Key Vault data-protection guardrails AND key/secret
// expiry:
//   - no soft-delete → critical (deleted secrets are unrecoverable)
//   - soft-delete but no purge-protection → warning (an attacker /
//     mistake can still purge within the retention window)
//   - keys/secrets already expired → critical
//   - keys/secrets expiring within 14 days → warning
//
// Mirrors AWS KMSKeys / GCP KMSKeys.
type KeyVaults struct {
	// Now is an optional clock override for testing. Defaults to time.Now.
	Now func() time.Time
}

const keyVaultsName = "azure-keyvaults"

// Name satisfies cloudprobe.Probe.
func (KeyVaults) Name() string { return keyVaultsName }

// Run satisfies cloudprobe.Probe.
func (p KeyVaults) Run(ctx context.Context, src cloud.Source) probe.Result {
	azClient := src.Azure()
	if azClient == nil {
		return skipped(keyVaultsName, "Azure not configured (cloud.azure.enabled=false)")
	}
	vaults, err := azClient.ListKeyVaults(ctx)
	if err != nil {
		return probeFailed(keyVaultsName, "keyvault.ListVaults", err)
	}

	now := time.Now().UTC()
	if p.Now != nil {
		now = p.Now().UTC()
	}

	var findings []probe.Finding
	for _, v := range vaults {
		subject := fmt.Sprintf("azure-keyvault/%s/%s/%s", azClient.SubscriptionID(), v.ResourceGroup, v.Name)

		// --- vault-level policy checks (existing) ---
		if !v.SoftDelete {
			findings = append(findings, probe.Finding{
				Component:   subject,
				Severity:    probe.SeverityCritical,
				Message:     fmt.Sprintf("Key Vault %q has soft-delete disabled; deleted secrets/keys are unrecoverable", v.Name),
				Remediation: fmt.Sprintf("az keyvault update -n %s -g %s --enable-soft-delete true --subscription %s", v.Name, v.ResourceGroup, azClient.SubscriptionID()),
			})
			// Still check items below — don't skip the vault entirely.
		} else if !v.PurgeProtection {
			findings = append(findings, probe.Finding{
				Component:   subject,
				Severity:    probe.SeverityWarning,
				Message:     fmt.Sprintf("Key Vault %q has soft-delete but no purge-protection; secrets can still be purged within the retention window", v.Name),
				Remediation: fmt.Sprintf("az keyvault update -n %s -g %s --enable-purge-protection true --subscription %s", v.Name, v.ResourceGroup, azClient.SubscriptionID()),
			})
		}

		// --- key/secret expiry checks (new) ---
		if v.VaultURL == "" {
			continue // no data-plane URL: skip expiry checks (old snapshot)
		}
		keys, secrets, itemsErr := azClient.ListKeyVaultItems(ctx, v.VaultURL)
		if itemsErr != nil {
			// RBAC insufficient or vault inaccessible — skip silently.
			log.Printf("azure-keyvaults: skipping key/secret expiry for %s: %v", v.Name, itemsErr)
			continue
		}

		for _, k := range keys {
			if k.ExpiresAt == nil {
				continue
			}
			ttl := k.ExpiresAt.Sub(now)
			if ttl <= 0 {
				findings = append(findings, probe.Finding{
					Component: subject,
					Severity:  probe.SeverityCritical,
					Message:   fmt.Sprintf("KeyVault key %q in vault %q expired %s ago", k.KeyName, v.Name, formatDuration(-ttl)),
				})
			} else if ttl <= kvWarnWindow {
				findings = append(findings, probe.Finding{
					Component: subject,
					Severity:  probe.SeverityWarning,
					Message:   fmt.Sprintf("KeyVault key %q in vault %q expires in %s", k.KeyName, v.Name, formatDuration(ttl)),
				})
			}
		}

		for _, s := range secrets {
			if s.ExpiresAt == nil {
				continue
			}
			ttl := s.ExpiresAt.Sub(now)
			if ttl <= 0 {
				findings = append(findings, probe.Finding{
					Component: subject,
					Severity:  probe.SeverityCritical,
					Message:   fmt.Sprintf("KeyVault secret %q in vault %q expired %s ago", s.SecretName, v.Name, formatDuration(-ttl)),
				})
			} else if ttl <= kvWarnWindow {
				findings = append(findings, probe.Finding{
					Component: subject,
					Severity:  probe.SeverityWarning,
					Message:   fmt.Sprintf("KeyVault secret %q in vault %q expires in %s", s.SecretName, v.Name, formatDuration(ttl)),
				})
			}
		}
	}

	return probe.Result{
		Component: probe.ComponentResult{Component: keyVaultsName, Status: rollupStatus(findings), Detail: fmt.Sprintf("%d Key Vault(s) inspected in subscription %s", len(vaults), azClient.SubscriptionID())},
		Findings:  findings,
	}
}

// formatDuration renders a duration in human-readable form (e.g. "2d3h").
func formatDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	if days > 0 {
		return fmt.Sprintf("%dd%dh", days, hours)
	}
	return fmt.Sprintf("%dh", int(d.Hours()))
}
