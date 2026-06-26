// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package gcp

import (
	"context"
	"fmt"

	"github.com/srenix-ai/agentic-sre/pkg/cloud"
	"github.com/srenix-ai/agentic-sre/pkg/probe"
)

// GCSPublicAccess flags GCS buckets that allow public access:
//   - an IAM binding to allUsers / allAuthenticatedUsers → critical
//   - PublicAccessPrevention != "enforced" → warning (public ACLs are
//     possible even if none exist yet)
//
// Mirrors AWS S3BucketPublicAccess.
type GCSPublicAccess struct{}

const gcsName = "gcp-gcs-public-access"

// Name satisfies cloudprobe.Probe.
func (GCSPublicAccess) Name() string { return gcsName }

// Run satisfies cloudprobe.Probe.
func (GCSPublicAccess) Run(ctx context.Context, src cloud.Source) probe.Result {
	gcpClient := src.GCP()
	if gcpClient == nil {
		return skipped(gcsName, "GCP not configured (cloud.gcp.enabled=false)")
	}
	buckets, err := gcpClient.ListBuckets(ctx)
	if err != nil {
		return probeFailed(gcsName, "storage.ListBuckets", err)
	}

	var findings []probe.Finding
	for _, b := range buckets {
		subject := fmt.Sprintf("gcp-gcs/%s/%s", gcpClient.Project(), b.Name)
		if b.HasAllUsersBinding {
			findings = append(findings, probe.Finding{
				Component:   subject,
				Severity:    probe.SeverityCritical,
				Message:     fmt.Sprintf("GCS bucket %q grants public access (allUsers / allAuthenticatedUsers IAM binding)", b.Name),
				Remediation: fmt.Sprintf("gcloud storage buckets remove-iam-policy-binding gs://%s --member=allUsers --role=roles/storage.objectViewer (and allAuthenticatedUsers). Confirm no public site depends on it first.", b.Name),
			})
			continue
		}
		if b.PublicAccessPrevention != "enforced" {
			findings = append(findings, probe.Finding{
				Component:   subject,
				Severity:    probe.SeverityWarning,
				Message:     fmt.Sprintf("GCS bucket %q has publicAccessPrevention=%q (not enforced); public ACLs are possible", b.Name, b.PublicAccessPrevention),
				Remediation: fmt.Sprintf("gcloud storage buckets update gs://%s --public-access-prevention", b.Name),
			})
		}
	}

	return probe.Result{
		Component: probe.ComponentResult{Component: gcsName, Status: rollupStatus(findings), Detail: fmt.Sprintf("%d bucket(s) inspected in project %s", len(buckets), gcpClient.Project())},
		Findings:  findings,
	}
}

// KMSKeys flags Cloud KMS keys in a non-ENABLED primary state
// (critical for DESTROY_SCHEDULED / DESTROYED / *_FAILED; warning for
// DISABLED) and keys without automatic rotation configured (warning).
// Mirrors AWS KMSKeys.
type KMSKeys struct{}

const kmsName = "gcp-kms"

// Name satisfies cloudprobe.Probe.
func (KMSKeys) Name() string { return kmsName }

// Run satisfies cloudprobe.Probe.
func (KMSKeys) Run(ctx context.Context, src cloud.Source) probe.Result {
	gcpClient := src.GCP()
	if gcpClient == nil {
		return skipped(kmsName, "GCP not configured (cloud.gcp.enabled=false)")
	}
	keys, err := gcpClient.ListKMSKeys(ctx)
	if err != nil {
		return probeFailed(kmsName, "kms.ListCryptoKeys", err)
	}

	var findings []probe.Finding
	for _, k := range keys {
		subject := fmt.Sprintf("gcp-kms/%s/%s", gcpClient.Project(), k.Name)
		switch k.PrimaryState {
		case "ENABLED":
			// Healthy state. Flag missing rotation as a posture warning.
			if !k.RotationScheduled {
				findings = append(findings, probe.Finding{
					Component:   subject,
					Severity:    probe.SeverityWarning,
					Message:     fmt.Sprintf("KMS key %q has no automatic rotation configured", k.Name),
					Remediation: fmt.Sprintf("gcloud kms keys update %s --rotation-period=90d --next-rotation-time=<ts> (verify the key purpose supports rotation).", k.Name),
				})
			}
		case "DISABLED":
			findings = append(findings, probe.Finding{
				Component: subject,
				Severity:  probe.SeverityWarning,
				Message:   fmt.Sprintf("KMS key %q primary version is DISABLED; encrypt/decrypt with it will fail", k.Name),
			})
		case "DESTROY_SCHEDULED", "DESTROYED", "IMPORT_FAILED", "GENERATION_FAILED":
			findings = append(findings, probe.Finding{
				Component:   subject,
				Severity:    probe.SeverityCritical,
				Message:     fmt.Sprintf("KMS key %q primary version state=%s — data encrypted with it may become unrecoverable", k.Name, k.PrimaryState),
				Remediation: fmt.Sprintf("gcloud kms keys versions list --key=%s --keyring=<ring> --location=<loc> — if DESTROY_SCHEDULED and still needed, restore before the scheduled time.", k.Name),
			})
		}
	}

	return probe.Result{
		Component: probe.ComponentResult{Component: kmsName, Status: rollupStatus(findings), Detail: fmt.Sprintf("%d KMS key(s) inspected in project %s", len(keys), gcpClient.Project())},
		Findings:  findings,
	}
}
