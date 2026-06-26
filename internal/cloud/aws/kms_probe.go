// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/srenix-ai/agentic-sre/pkg/cloud"
	"github.com/srenix-ai/agentic-sre/pkg/probe"
)

// KMSKeys flags:
//   - Keys in PendingDeletion within 7 days → critical (data loss imminent)
//   - Keys in Disabled state → warning (callers will get "AccessDenied")
//   - Keys in Unavailable / PendingImport → warning
type KMSKeys struct{}

const kmsName = "aws-kms-keys"

const kmsDeletionWindowCrit = 7 * 24 * time.Hour

// Name satisfies cloudprobe.Probe.
func (KMSKeys) Name() string { return kmsName }

// Run satisfies cloudprobe.Probe.
func (KMSKeys) Run(ctx context.Context, src cloud.Source) probe.Result {
	awsClient := src.AWS()
	if awsClient == nil {
		return skipped(kmsName, "AWS not configured")
	}
	keys, err := awsClient.ListKMSKeys(ctx)
	if err != nil {
		return probeFailed(kmsName, "kms.ListKeys", err)
	}
	now := time.Now().UTC()
	var findings []probe.Finding
	for _, k := range keys {
		subject := fmt.Sprintf("aws-kms/%s/%s", awsClient.Region(), k.KeyID)
		switch k.State {
		case "Enabled":
			// healthy
		case "Disabled":
			findings = append(findings, probe.Finding{
				Component: subject, Severity: probe.SeverityWarning,
				Message: fmt.Sprintf("KMS key %s is Disabled (%s)", k.KeyID, k.Description),
			})
		case "PendingDeletion":
			ttl := k.DeletionDate.Sub(now)
			severity := probe.SeverityWarning
			if ttl < kmsDeletionWindowCrit {
				severity = probe.SeverityCritical
			}
			findings = append(findings, probe.Finding{
				Component: subject, Severity: severity,
				Message: fmt.Sprintf("KMS key %s scheduled for deletion in %s — anything encrypted with it will be irrecoverable",
					k.KeyID, ttl.Round(time.Hour)),
				Remediation: fmt.Sprintf("aws kms cancel-key-deletion --key-id %s   # if accidental", k.KeyID),
			})
		case "PendingImport", "Unavailable", "PendingReplicaDeletion":
			findings = append(findings, probe.Finding{
				Component: subject, Severity: probe.SeverityWarning,
				Message: fmt.Sprintf("KMS key %s in transitional state %q", k.KeyID, k.State),
			})
		case "Creating":
			// transient; ignore
		default:
			findings = append(findings, probe.Finding{
				Component: subject, Severity: probe.SeverityWarning,
				Message: fmt.Sprintf("KMS key %s in unknown state %q", k.KeyID, k.State),
			})
		}
	}
	return probe.Result{
		Component: probe.ComponentResult{
			Component: kmsName,
			Status:    rollupStatus(findings),
			Detail:    fmt.Sprintf("%d key(s) inspected in %s", len(keys), awsClient.Region()),
		},
		Findings: findings,
	}
}
