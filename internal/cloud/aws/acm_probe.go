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

// ACMCertExpiry flags ACM certificates:
//   - in FAILED / VALIDATION_TIMED_OUT / REVOKED state → critical
//   - in EXPIRED state → critical
//   - in PENDING_VALIDATION > 24h → warning (renewal stuck)
//   - expiring within certExpiryWindow (default 14d) → warning, or critical < 3d
type ACMCertExpiry struct{}

const acmName = "aws-acm-cert-expiry"

const (
	certExpiryWindowWarn = 14 * 24 * time.Hour
	certExpiryWindowCrit = 3 * 24 * time.Hour
)

// Name satisfies cloudprobe.Probe.
func (ACMCertExpiry) Name() string { return acmName }

// Run satisfies cloudprobe.Probe.
func (ACMCertExpiry) Run(ctx context.Context, src cloud.Source) probe.Result {
	awsClient := src.AWS()
	if awsClient == nil {
		return skipped(acmName, "AWS not configured")
	}
	certs, err := awsClient.ListACMCertificates(ctx)
	if err != nil {
		return probeFailed(acmName, "acm.ListCertificates", err)
	}
	now := time.Now().UTC()
	var findings []probe.Finding
	for _, c := range certs {
		subject := fmt.Sprintf("aws-acm/%s/%s", awsClient.Region(), c.DomainName)
		switch c.Status {
		case "ISSUED":
			// Healthy state — but check expiry window
		case "PENDING_VALIDATION":
			findings = append(findings, probe.Finding{
				Component: subject, Severity: probe.SeverityWarning,
				Message: fmt.Sprintf("ACM cert for %s stuck in PENDING_VALIDATION (DNS / email validation not yet complete)", c.DomainName),
			})
			continue
		default: // FAILED, EXPIRED, VALIDATION_TIMED_OUT, REVOKED, INACTIVE
			findings = append(findings, probe.Finding{
				Component: subject, Severity: probe.SeverityCritical,
				Message:     fmt.Sprintf("ACM cert for %s in state %q", c.DomainName, c.Status),
				Remediation: fmt.Sprintf("aws acm describe-certificate --certificate-arn %s", c.ARN),
			})
			continue
		}
		if c.NotAfter.IsZero() {
			continue
		}
		ttl := c.NotAfter.Sub(now)
		if ttl <= 0 {
			findings = append(findings, probe.Finding{
				Component: subject, Severity: probe.SeverityCritical,
				Message: fmt.Sprintf("ACM cert for %s EXPIRED at %s", c.DomainName, c.NotAfter.Format(time.RFC3339)),
			})
		} else if ttl < certExpiryWindowCrit {
			findings = append(findings, probe.Finding{
				Component: subject, Severity: probe.SeverityCritical,
				Message: fmt.Sprintf("ACM cert for %s expires in %s (<3d)", c.DomainName, ttl.Round(time.Hour)),
			})
		} else if ttl < certExpiryWindowWarn {
			findings = append(findings, probe.Finding{
				Component: subject, Severity: probe.SeverityWarning,
				Message: fmt.Sprintf("ACM cert for %s expires in %s (<14d)", c.DomainName, ttl.Round(time.Hour)),
			})
		}
	}
	return probe.Result{
		Component: probe.ComponentResult{
			Component: acmName,
			Status:    rollupStatus(findings),
			Detail:    fmt.Sprintf("%d certificate(s) inspected in %s", len(certs), awsClient.Region()),
		},
		Findings: findings,
	}
}
