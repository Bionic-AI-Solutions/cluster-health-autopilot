// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"
	"fmt"
	"time"

	intcloud "github.com/srenix-ai/agentic-sre/internal/cloud"
	"github.com/srenix-ai/agentic-sre/pkg/cloud"
	"github.com/srenix-ai/agentic-sre/pkg/probe"
)

// AppGatewayBackends flags Application Gateway backend pools with no
// healthy members (critical) or partial unhealth (warning). Mirrors
// AWS ALBTargetHealth / GCP LoadBalancerBackends.
type AppGatewayBackends struct{}

const appgwName = "azure-appgw-backends"

// Name satisfies cloudprobe.Probe.
func (AppGatewayBackends) Name() string { return appgwName }

// Run satisfies cloudprobe.Probe.
func (AppGatewayBackends) Run(ctx context.Context, src cloud.Source) probe.Result {
	azClient := src.Azure()
	if azClient == nil {
		return skipped(appgwName, "Azure not configured (cloud.azure.enabled=false)")
	}
	pools, err := azClient.ListAppGatewayBackends(ctx)
	if err != nil {
		return probeFailed(appgwName, "network.ListAppGatewayBackends", err)
	}

	var findings []probe.Finding
	var unmeasured int
	for _, p := range pools {
		if p.HealthyCount < 0 {
			// Backend health not measured (live mode — needs the
			// BackendHealth LRO). Skip rather than treat every pool as
			// healthy, which would silently never fire.
			unmeasured++
			continue
		}
		subject := fmt.Sprintf("azure-appgw/%s/%s/%s", azClient.SubscriptionID(), p.Gateway, p.PoolName)
		switch {
		case p.HealthyCount == 0 && (p.UnhealthyCount > 0 || p.TotalCount > 0):
			// The "(lb: <AppGW public hostname>)" suffix is the Srenix Enterprise
			// RCA join key; gateways without a listener hostname fall
			// back to the gateway name — see internal/cloud/joinkeys.go.
			// contract: internal/cloud/contract_test.go
			lbValue := p.FrontendHostname
			if lbValue == "" {
				lbValue = p.Gateway
			}
			findings = append(findings, probe.Finding{
				Component:   subject,
				Severity:    probe.SeverityCritical,
				Message:     fmt.Sprintf("App Gateway %q backend pool %q has 0 healthy members (%d unhealthy)", p.Gateway, p.PoolName, p.UnhealthyCount) + intcloud.JoinKeyLB(lbValue),
				Remediation: fmt.Sprintf("az network application-gateway show-backend-health --name %s -g <rg> --subscription %s", p.Gateway, azClient.SubscriptionID()),
			})
		case p.UnhealthyCount > 0:
			findings = append(findings, probe.Finding{
				Component: subject,
				Severity:  probe.SeverityWarning,
				Message:   fmt.Sprintf("App Gateway %q backend pool %q has %d unhealthy member(s) (%d healthy)", p.Gateway, p.PoolName, p.UnhealthyCount, p.HealthyCount),
			})
		}
	}

	detail := fmt.Sprintf("%d backend pool(s) inspected in subscription %s", len(pools), azClient.SubscriptionID())
	if unmeasured > 0 {
		detail += fmt.Sprintf("; backend health not measured for %d (needs BackendHealth LRO)", unmeasured)
	}
	return probe.Result{
		Component: probe.ComponentResult{Component: appgwName, Status: rollupStatus(findings), Detail: detail},
		Findings:  findings,
	}
}

// Certificates flags App Service / managed certs that failed to issue
// (critical) or expire within 21 days (warning). Mirrors AWS
// ACMCertExpiry / GCP ManagedCertificates.
type Certificates struct {
	// Now overridable in tests.
	Now func() time.Time
}

const certsName = "azure-certs"

const certExpiryWarnWindow = 21 * 24 * time.Hour

// Name satisfies cloudprobe.Probe.
func (Certificates) Name() string { return certsName }

// Run satisfies cloudprobe.Probe.
func (c Certificates) Run(ctx context.Context, src cloud.Source) probe.Result {
	azClient := src.Azure()
	if azClient == nil {
		return skipped(certsName, "Azure not configured (cloud.azure.enabled=false)")
	}
	now := c.Now
	if now == nil {
		now = time.Now
	}
	certs, err := azClient.ListAppServiceCertificates(ctx)
	if err != nil {
		return probeFailed(certsName, "web.ListCertificates", err)
	}

	var findings []probe.Finding
	t := now()
	for _, cert := range certs {
		subject := fmt.Sprintf("azure-cert/%s/%s/%s", azClient.SubscriptionID(), cert.ResourceGroup, cert.Name)
		// The "(domains: d1,d2)" suffix is the Srenix Enterprise RCA join key
		// (omitted when no domains are known) — see
		// internal/cloud/joinkeys.go.
		// contract: internal/cloud/contract_test.go
		domainsSuffix := intcloud.JoinKeyDomains(cert.Domains)
		if !cert.Issued {
			findings = append(findings, probe.Finding{
				Component:   subject,
				Severity:    probe.SeverityCritical,
				Message:     fmt.Sprintf("Certificate %q is not issued (provisioning failed or pending)", cert.Name) + domainsSuffix,
				Remediation: fmt.Sprintf("az webapp config ssl show --certificate-name %s -g %s --subscription %s — check domain validation.", cert.Name, cert.ResourceGroup, azClient.SubscriptionID()),
			})
			continue
		}
		if !cert.NotAfter.IsZero() && cert.NotAfter.Sub(t) < certExpiryWarnWindow {
			findings = append(findings, probe.Finding{
				Component:   subject,
				Severity:    probe.SeverityWarning,
				Message:     fmt.Sprintf("Certificate %q expires %s (< 21d)", cert.Name, cert.NotAfter.UTC().Format("2006-01-02")) + domainsSuffix,
				Remediation: fmt.Sprintf("Renew before expiry. App Service managed certs auto-renew if the domain binding is intact; confirm the custom-domain binding for %s.", cert.Name),
			})
		}
	}

	return probe.Result{
		Component: probe.ComponentResult{Component: certsName, Status: rollupStatus(findings), Detail: fmt.Sprintf("%d cert(s) inspected in subscription %s", len(certs), azClient.SubscriptionID())},
		Findings:  findings,
	}
}
