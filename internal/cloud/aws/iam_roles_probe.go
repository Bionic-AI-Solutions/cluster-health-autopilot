// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/srenix-ai/agentic-sre/pkg/cloud"
	"github.com/srenix-ai/agentic-sre/pkg/probe"
)

// IAMRoles verifies that a Helm-configured list of IAM role ARNs / names
// actually exist. Typical use: list every IRSA-pointed role from your
// cluster's ServiceAccount annotations so Srenix flags drift between K8s
// expectation and IAM reality (the classic "role got renamed, IRSA
// silently broken" failure mode).
//
// Configured via CLOUD_AWS_IAM_ROLES env var (comma-separated ARNs or
// names). SKIPPED when unset.
type IAMRoles struct{}

const iamRolesName = "aws-iam-roles"

const iamRolesEnv = "CLOUD_AWS_IAM_ROLES"

// Name satisfies cloudprobe.Probe.
func (IAMRoles) Name() string { return iamRolesName }

// Run satisfies cloudprobe.Probe.
func (IAMRoles) Run(ctx context.Context, src cloud.Source) probe.Result {
	awsClient := src.AWS()
	if awsClient == nil {
		return skipped(iamRolesName, "AWS not configured")
	}
	raw := strings.TrimSpace(os.Getenv(iamRolesEnv))
	if raw == "" {
		return skipped(iamRolesName, "set "+iamRolesEnv+" to a comma-separated list of role ARNs to verify")
	}
	roles := strings.Split(raw, ",")
	var findings []probe.Finding
	for _, r := range roles {
		r = strings.TrimSpace(r)
		if r == "" {
			continue
		}
		got, err := awsClient.GetIAMRole(ctx, r)
		if err != nil {
			return probeFailed(iamRolesName, "iam.GetRole "+r, err)
		}
		subject := "aws-iam-role/" + roleSubjectKey(r)
		if !got.Exists {
			findings = append(findings, probe.Finding{
				Component: subject, Severity: probe.SeverityCritical,
				Message:     fmt.Sprintf("IAM role %q does not exist — likely IRSA / service-account drift", r),
				Remediation: "Recreate the role, or fix the K8s ServiceAccount annotation eks.amazonaws.com/role-arn that points at it",
			})
			continue
		}
		if !got.HasTrustPolicy {
			findings = append(findings, probe.Finding{
				Component: subject, Severity: probe.SeverityWarning,
				Message: fmt.Sprintf("IAM role %q exists but has no trust policy (AssumeRolePolicyDocument empty)", got.ARN),
			})
		}
	}
	return probe.Result{
		Component: probe.ComponentResult{
			Component: iamRolesName,
			Status:    rollupStatus(findings),
			Detail:    fmt.Sprintf("%d role(s) verified", len(roles)),
		},
		Findings: findings,
	}
}

// roleSubjectKey strips "arn:aws:iam::ACCOUNT:role/" prefix so the
// subject is operator-readable.
func roleSubjectKey(arnOrName string) string {
	if i := strings.LastIndex(arnOrName, "role/"); i >= 0 {
		return arnOrName[i+len("role/"):]
	}
	return arnOrName
}
