// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package diagnose

import (
	"context"
	"fmt"
	"time"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// CertExpiry surfaces cert-manager Certificate resources that are:
//   - Not Ready (status.conditions[Ready].status = "False"), or
//   - Expired (status.notAfter in the past), or
//   - Expiring within warnWindow (default 14 days).
//
// Returns nil when cert-manager is not installed (CRD absent). The 14-day
// window sits above cert-manager's default 2/3-of-validity renewal point;
// seeing a diagnostic here means renewal has stalled or the cert is short-lived.
//
// Resolution: check `kubectl describe certificate <name>` for the exact
// renewal error, verify Vault PKI / ACME / CA issuer connectivity, and
// confirm the Issuer/ClusterIssuer is Ready.
type CertExpiry struct {
	// WarnWindow is how far in advance to warn about upcoming expiry.
	// Zero uses the default of 14 days.
	WarnWindow time.Duration
}

// Name returns the analyzer's identifier.
func (CertExpiry) Name() string { return "CertExpiry" }

const defaultWarnWindow = 14 * 24 * time.Hour

// Run lists all cert-manager Certificate resources cluster-wide and emits a
// Diagnostic for any that are not Ready, expired, or expiring within WarnWindow.
func (c CertExpiry) Run(ctx context.Context, src snapshot.Source) []Diagnostic {
	certs, err := src.List(ctx, snapshot.GVRCertificate, "")
	if err != nil || len(certs.Items) == 0 {
		// CRD not installed or no certificates — not an error.
		return nil
	}

	window := c.WarnWindow
	if window == 0 {
		window = defaultWarnWindow
	}
	now := time.Now().UTC()
	var out []Diagnostic

	for i := range certs.Items {
		cert := certs.Items[i]
		ns := cert.GetNamespace()
		name := cert.GetName()
		subject := fmt.Sprintf("cert-expiry/%s/%s", ns, name)

		// Not-Ready condition takes priority — it captures renewal failures,
		// issuer connectivity problems, etc. before the cert actually expires.
		if readyStatus, readyMsg := certReadyCondition(cert); readyStatus == "False" {
			out = append(out, Diagnostic{
				Subject: subject,
				Message: fmt.Sprintf(
					"Certificate `%s/%s` is not Ready: %s. "+
						"Check Issuer/ClusterIssuer status and cert-manager controller logs.",
					ns, name, truncate(readyMsg, 200),
				),
			})
			continue // one diagnostic per certificate is enough
		}

		// Check expiry window from status.notAfter.
		notAfterRaw, _, _ := unstructured.NestedString(cert.Object, "status", "notAfter")
		if notAfterRaw == "" {
			continue // certificate has never been issued yet (handled by not-Ready above)
		}
		notAfter, err := time.Parse(time.RFC3339, notAfterRaw)
		if err != nil {
			continue
		}

		switch {
		case now.After(notAfter):
			out = append(out, Diagnostic{
				Subject: subject,
				Message: fmt.Sprintf(
					"Certificate `%s/%s` EXPIRED at %s. "+
						"Pods consuming this TLS secret will fail on next restart.",
					ns, name, notAfter.Format("2006-01-02 15:04 UTC"),
				),
			})
		case notAfter.Before(now.Add(window)):
			daysLeft := int(time.Until(notAfter).Hours() / 24)
			out = append(out, Diagnostic{
				Subject: subject,
				Message: fmt.Sprintf(
					"Certificate `%s/%s` expires in %d day(s) (%s). "+
						"cert-manager renewal may have stalled — check Issuer status.",
					ns, name, daysLeft, notAfter.Format("2006-01-02"),
				),
			})
		}
	}
	return out
}

// certReadyCondition returns (status, message) of the Ready condition on a
// cert-manager Certificate, or ("", "") if not present.
func certReadyCondition(cert unstructured.Unstructured) (status, message string) {
	conds, _, _ := unstructured.NestedSlice(cert.Object, "status", "conditions")
	for _, raw := range conds {
		cm, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if t, _ := cm["type"].(string); t != "Ready" {
			continue
		}
		s, _ := cm["status"].(string)
		m, _ := cm["message"].(string)
		return s, m
	}
	return "", ""
}
