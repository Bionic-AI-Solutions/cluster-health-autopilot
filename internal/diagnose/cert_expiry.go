// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package diagnose

import (
	"context"
	"fmt"
	"strings"
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
		logListFailure("certificates.cert-manager.io", err, true) // silent when the CRD/resource is absent; logs Forbidden etc.
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
		if readyStatus, _ := certReadyCondition(cert); readyStatus == "False" {
			msg, rem := certNotReadyDetail(ctx, src, ns, name)
			out = append(out, Diagnostic{
				Subject:     subject,
				Message:     msg,
				Remediation: rem,
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

// certNotReadyDetail builds a detailed message + remediation by chaining:
// Certificate → failed CertificateRequest → ACME Order → Challenge reason.
// Falls back to the Certificate-level ready message when the chain is absent.
func certNotReadyDetail(ctx context.Context, src snapshot.Source, ns, name string) (msg, remediation string) {
	// Walk: CertificateRequest → Order → Challenge to get the real root-cause.
	crName, crMsg := failedCertRequest(ctx, src, ns, name)
	if crMsg == "" {
		// No failed CR found — use the Certificate-level message only.
		cert, _ := src.List(ctx, snapshot.GVRCertificate, ns)
		for i := range cert.Items {
			if cert.Items[i].GetName() == name {
				_, m := certReadyCondition(cert.Items[i])
				crMsg = m
				break
			}
		}
		return fmt.Sprintf("Certificate `%s/%s` is not Ready: %s. Check Issuer/ClusterIssuer status and cert-manager controller logs.", ns, name, truncate(crMsg, 200)),
			fmt.Sprintf("kubectl describe certificate %s -n %s", name, ns)
	}

	orderName, challengeReason := failedOrderChallenge(ctx, src, ns, crName)

	if challengeReason == "" {
		// CR failed but no challenge detail — use CR message.
		return fmt.Sprintf("Certificate `%s/%s` is not Ready: %s", ns, name, truncate(crMsg, 200)),
			fmt.Sprintf("kubectl delete certificaterequest %s -n %s  # forces cert-manager to retry", crName, ns)
	}

	// Build a concise message from the challenge reason.
	shortReason := truncate(challengeReason, 220)
	deleteCmd := fmt.Sprintf("kubectl delete certificaterequest %s -n %s", crName, ns)
	if orderName != "" {
		deleteCmd = fmt.Sprintf("kubectl delete order %s -n %s  # cert-manager will retry immediately", orderName, ns)
	}

	// Detect DNS-record-missing errors and call out the specific action.
	if strings.Contains(challengeReason, "no valid A records") || strings.Contains(challengeReason, "no valid AAAA records") {
		domain := extractACMEDomain(challengeReason)
		domainHint := ""
		if domain != "" {
			domainHint = fmt.Sprintf(" Add A/AAAA DNS record for *%s* → ingress IP, then: ", domain)
		}
		return fmt.Sprintf("Certificate `%s/%s` is not Ready: ACME HTTP-01 challenge failed — %s", ns, name, shortReason),
			domainHint + deleteCmd
	}

	return fmt.Sprintf("Certificate `%s/%s` is not Ready: ACME challenge failed — %s", ns, name, shortReason),
		deleteCmd
}

// failedCertRequest returns (name, failureMessage) of the most-recently-failed
// CertificateRequest for the given Certificate in the same namespace.
func failedCertRequest(ctx context.Context, src snapshot.Source, ns, certName string) (name, failMsg string) {
	crs, err := src.List(ctx, snapshot.GVRCertificateRequest, ns)
	if err != nil {
		return "", ""
	}
	for i := range crs.Items {
		cr := crs.Items[i]
		if cr.GetLabels()["cert-manager.io/certificate-name"] != certName {
			continue
		}
		// Check failureTime or Ready=False/reason=Failed.
		ft, _, _ := unstructured.NestedString(cr.Object, "status", "failureTime")
		conditions, _, _ := unstructured.NestedSlice(cr.Object, "status", "conditions")
		for _, c := range conditions {
			cm, ok := c.(map[string]any)
			if !ok {
				continue
			}
			if cm["type"] == "Ready" && cm["status"] == "False" && (cm["reason"] == "Failed" || ft != "") {
				msg, _ := cm["message"].(string)
				return cr.GetName(), msg
			}
		}
	}
	return "", ""
}

// failedOrderChallenge returns (orderName, challengeReason) for the failed ACME
// Order owned by the given CertificateRequest, and the first invalid Challenge's reason.
func failedOrderChallenge(ctx context.Context, src snapshot.Source, ns, crName string) (orderName, reason string) {
	orders, err := src.List(ctx, snapshot.GVRCertManagerOrder, ns)
	if err != nil {
		return "", ""
	}
	for i := range orders.Items {
		order := orders.Items[i]
		state, _, _ := unstructured.NestedString(order.Object, "status", "state")
		if state != "errored" && state != "invalid" {
			continue
		}
		for _, ref := range order.GetOwnerReferences() {
			if ref.Name != crName {
				continue
			}
			oName := order.GetName()
			challenges, err := src.List(ctx, snapshot.GVRCertManagerChallenge, ns)
			if err != nil {
				return oName, ""
			}
			for j := range challenges.Items {
				ch := challenges.Items[j]
				for _, cref := range ch.GetOwnerReferences() {
					if cref.Name == oName {
						r, _, _ := unstructured.NestedString(ch.Object, "status", "reason")
						return oName, r
					}
				}
			}
			return oName, ""
		}
	}
	return "", ""
}

// extractACMEDomain parses the domain from an ACME error reason string.
// e.g. "acme: authorization error for foo.example.com: 400 ..."
func extractACMEDomain(reason string) string {
	const marker = "authorization error for "
	idx := strings.Index(reason, marker)
	if idx < 0 {
		return ""
	}
	rest := reason[idx+len(marker):]
	if end := strings.IndexAny(rest, ": "); end > 0 {
		return rest[:end]
	}
	return rest
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
