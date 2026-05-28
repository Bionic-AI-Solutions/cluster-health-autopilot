// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package gcp

import (
	"context"
	"fmt"
	"time"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/cloud"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/probe"
)

// LoadBalancerBackends flags LB backend services with no healthy
// backends (critical) or partial unhealth (warning). Mirrors AWS
// ALBTargetHealth.
type LoadBalancerBackends struct{}

const lbBackendsName = "gcp-lb-backends"

// Name satisfies cloudprobe.Probe.
func (LoadBalancerBackends) Name() string { return lbBackendsName }

// Run satisfies cloudprobe.Probe.
func (LoadBalancerBackends) Run(ctx context.Context, src cloud.Source) probe.Result {
	gcpClient := src.GCP()
	if gcpClient == nil {
		return skipped(lbBackendsName, "GCP not configured (cloud.gcp.enabled=false)")
	}
	svcs, err := gcpClient.ListBackendServices(ctx)
	if err != nil {
		return probeFailed(lbBackendsName, "compute.ListBackendServices", err)
	}

	var findings []probe.Finding
	for _, s := range svcs {
		subject := fmt.Sprintf("gcp-lb/%s/%s", gcpClient.Project(), s.Name)
		switch {
		case s.HealthyCount == 0 && (s.UnhealthyCount > 0 || s.TotalBackends > 0):
			findings = append(findings, probe.Finding{
				Component:   subject,
				Severity:    probe.SeverityCritical,
				Message:     fmt.Sprintf("LB backend service %q has 0 healthy backends (%d unhealthy); traffic is failing", s.Name, s.UnhealthyCount),
				Remediation: fmt.Sprintf("gcloud compute backend-services get-health %s --project=%s — check the health-check config + backend instance health.", s.Name, gcpClient.Project()),
			})
		case s.UnhealthyCount > 0:
			findings = append(findings, probe.Finding{
				Component: subject,
				Severity:  probe.SeverityWarning,
				Message:   fmt.Sprintf("LB backend service %q has %d unhealthy backend(s) (%d healthy)", s.Name, s.UnhealthyCount, s.HealthyCount),
			})
		}
	}

	return probe.Result{
		Component: probe.ComponentResult{Component: lbBackendsName, Status: rollupStatus(findings), Detail: fmt.Sprintf("%d backend service(s) inspected in project %s", len(svcs), gcpClient.Project())},
		Findings:  findings,
	}
}

// ManagedCertificates flags Google-managed SSL certs that failed to
// provision/renew (critical) or expire within 21 days (warning).
// Mirrors AWS ACMCertExpiry.
type ManagedCertificates struct {
	// Now overridable in tests.
	Now func() time.Time
}

const managedCertsName = "gcp-managed-certs"

const certExpiryWarnWindow = 21 * 24 * time.Hour

// Name satisfies cloudprobe.Probe.
func (ManagedCertificates) Name() string { return managedCertsName }

// Run satisfies cloudprobe.Probe.
func (m ManagedCertificates) Run(ctx context.Context, src cloud.Source) probe.Result {
	gcpClient := src.GCP()
	if gcpClient == nil {
		return skipped(managedCertsName, "GCP not configured (cloud.gcp.enabled=false)")
	}
	now := m.Now
	if now == nil {
		now = time.Now
	}
	certs, err := gcpClient.ListManagedCertificates(ctx)
	if err != nil {
		return probeFailed(managedCertsName, "compute.ListSslCertificates", err)
	}

	var findings []probe.Finding
	t := now()
	for _, c := range certs {
		subject := fmt.Sprintf("gcp-cert/%s/%s", gcpClient.Project(), c.Name)
		switch c.Status {
		case "ACTIVE":
			if !c.NotAfter.IsZero() && c.NotAfter.Sub(t) < certExpiryWarnWindow {
				findings = append(findings, probe.Finding{
					Component:   subject,
					Severity:    probe.SeverityWarning,
					Message:     fmt.Sprintf("Managed cert %q (%s) expires %s (< 21d); Google-managed renewal may be stuck", c.Name, c.DomainName, c.NotAfter.UTC().Format("2006-01-02")),
					Remediation: fmt.Sprintf("gcloud compute ssl-certificates describe %s --project=%s — confirm the domain's DNS still points at the LB so auto-renewal can validate.", c.Name, gcpClient.Project()),
				})
			}
		case "PROVISIONING_FAILED", "PROVISIONING_FAILED_PERMANENTLY", "RENEWAL_FAILED":
			findings = append(findings, probe.Finding{
				Component:   subject,
				Severity:    probe.SeverityCritical,
				Message:     fmt.Sprintf("Managed cert %q (%s) status=%s", c.Name, c.DomainName, c.Status),
				Remediation: fmt.Sprintf("gcloud compute ssl-certificates describe %s --project=%s — usually a DNS or domain-validation problem.", c.Name, gcpClient.Project()),
			})
		case "PROVISIONING":
			// transient — silent unless it's been stuck (we don't have
			// a timestamp for "since" so we don't flag).
		}
	}

	return probe.Result{
		Component: probe.ComponentResult{Component: managedCertsName, Status: rollupStatus(findings), Detail: fmt.Sprintf("%d managed cert(s) inspected in project %s", len(certs), gcpClient.Project())},
		Findings:  findings,
	}
}
