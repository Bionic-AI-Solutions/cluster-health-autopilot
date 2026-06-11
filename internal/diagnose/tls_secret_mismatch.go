// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package diagnose

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"strings"
	"time"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// TLSSecretMismatch detects a high-confidence misconfiguration pattern:
//
//	An Ingress references a TLS Secret (`spec.tls[].secretName`) whose cert
//	is expired or expiring soon, while a healthy cert-manager Certificate
//	exists in the SAME namespace targeting a DIFFERENT Secret name with the
//	same hostname in dnsNames. The wires were crossed at install time —
//	cert-manager has been renewing into a Secret nobody references, while
//	the Ingress serves a hand-made cert that has no renewal path.
//
// This pattern silently survives until the served cert expires and end users
// see TLS errors. It is invisible to every other CHA analyzer because each
// piece looks healthy in isolation (Secret exists, Certificate is Ready).
type TLSSecretMismatch struct {
	// ExpiryWindow is how far ahead of expiry to start warning. Zero means
	// "report only already-expired secrets". The default is 14 days.
	ExpiryWindow time.Duration
}

// Name returns the analyzer's identifier.
func (TLSSecretMismatch) Name() string { return "TLSSecretMismatch" }

// Run scans every Ingress, checks its TLS Secret(s), and emits a diagnostic
// when the mismatch pattern is found.
func (a TLSSecretMismatch) Run(ctx context.Context, src snapshot.Source) []Diagnostic {
	window := a.ExpiryWindow
	if window == 0 {
		window = 14 * 24 * time.Hour
	}
	now := time.Now()

	ingresses, err := src.List(ctx, snapshot.GVRIngress, "")
	if err != nil || ingresses == nil || len(ingresses.Items) == 0 {
		logListFailure("ingresses", err, true) // silent when the CRD/resource is absent; logs Forbidden etc.
		return nil
	}

	var out []Diagnostic
	for i := range ingresses.Items {
		ing := ingresses.Items[i]
		ns := ing.GetNamespace()
		ingName := ing.GetName()

		tls, _, _ := unstructured.NestedSlice(ing.Object, "spec", "tls")
		for tlsIdx, raw := range tls {
			tm, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			secretName, _ := tm["secretName"].(string)
			if secretName == "" {
				continue
			}
			hosts := tlsHosts(tm)
			if len(hosts) == 0 {
				continue
			}

			// Inspect the currently-referenced Secret.
			expiry, secretOK := secretCertExpiry(ctx, src, ns, secretName)
			if !secretOK {
				continue // can't read the secret (RBAC) or it's not a TLS secret
			}
			expired := !expiry.IsZero() && now.After(expiry)
			expiringSoon := !expiry.IsZero() && !expired && expiry.Sub(now) < window
			if !expired && !expiringSoon {
				continue // referenced cert is healthy — no mismatch to flag
			}

			// For each host, look for a healthy Certificate in this namespace
			// whose target secret is a DIFFERENT name.
			better := findMismatchedManagedCert(ctx, src, ns, hosts, secretName)
			if better == "" {
				continue // no obvious replacement — surfaces elsewhere (CertExpiry analyzer)
			}

			hostStr := strings.Join(hosts, ", ")
			state := "expired"
			if !expired {
				state = "expiring within window"
			}
			out = append(out, Diagnostic{
				Subject: fmt.Sprintf("tls-secret-mismatch/%s/%s/%d", ns, ingName, tlsIdx),
				Source:  "TLSSecretMismatch",
				Message: fmt.Sprintf(
					"Ingress `%s/%s` host(s) *%s* serve %s cert from Secret `%s` while "+
						"cert-manager is renewing a healthy cert for the same host into Secret `%s` "+
						"in the same namespace. Wires crossed — Kong is serving the wrong Secret.",
					ns, ingName, hostStr, state, secretName, better,
				),
				Remediation: fmt.Sprintf(
					"Patch the Ingress to point at the cert-manager-managed Secret:\n"+
						"  kubectl -n %s patch ingress %s --type=json \\\n"+
						"    -p '[{\"op\":\"replace\",\"path\":\"/spec/tls/%d/secretName\",\"value\":\"%s\"}]'\n"+
						"Then delete the stale Secret `%s` (after confirming nothing else references it).\n"+
						"If the Ingress is GitOps-managed, apply the change in the source repo instead.",
					ns, ingName, tlsIdx, better, secretName,
				),
			})
		}
	}
	return out
}

// tlsHosts pulls the hosts list out of a single Ingress tls[] entry.
func tlsHosts(tm map[string]any) []string {
	raw, ok := tm["hosts"].([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, r := range raw {
		if h, ok := r.(string); ok && h != "" {
			out = append(out, strings.ToLower(h))
		}
	}
	return out
}

// secretCertExpiry returns the notAfter of the tls.crt inside the given Secret.
// Returns (zeroTime, false) if the Secret can't be read or doesn't contain a
// valid x509 cert under tls.crt.
func secretCertExpiry(ctx context.Context, src snapshot.Source, ns, name string) (time.Time, bool) {
	sec, err := src.Get(ctx, snapshot.GVRSecret, ns, name)
	if err != nil || sec == nil {
		return time.Time{}, false
	}
	data, _, _ := unstructured.NestedMap(sec.Object, "data")
	if data == nil {
		return time.Time{}, false
	}
	raw, ok := data["tls.crt"].(string)
	if !ok || raw == "" {
		return time.Time{}, false
	}
	// "data" values are base64 in the wire form; snapshot/Live may decode for us.
	// Try parsing as PEM first; if that fails, treat as base64-encoded PEM.
	pemBytes := []byte(raw)
	if block, _ := pem.Decode(pemBytes); block == nil {
		decoded, derr := base64Decode(raw)
		if derr != nil {
			return time.Time{}, false
		}
		pemBytes = decoded
	}
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return time.Time{}, false
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return time.Time{}, false
	}
	return cert.NotAfter, true
}

// findMismatchedManagedCert looks for a cert-manager Certificate in `ns` whose
// dnsNames cover at least one of `hosts` AND whose target secret is a name
// OTHER than `currentSecret`. The Certificate must also be Ready=True.
// Returns the better secret name, or "" if no candidate is found.
func findMismatchedManagedCert(ctx context.Context, src snapshot.Source, ns string, hosts []string, currentSecret string) string {
	certs, err := src.List(ctx, snapshot.GVRCertificate, ns)
	if err != nil || certs == nil || len(certs.Items) == 0 {
		return ""
	}
	hostSet := make(map[string]struct{}, len(hosts))
	for _, h := range hosts {
		hostSet[h] = struct{}{}
	}

	for i := range certs.Items {
		cert := certs.Items[i]
		target, _, _ := unstructured.NestedString(cert.Object, "spec", "secretName")
		if target == "" || target == currentSecret {
			continue
		}
		// Certificate must be Ready=True for us to recommend it.
		if !certificateIsReady(cert) {
			continue
		}
		dnsNames, _, _ := unstructured.NestedStringSlice(cert.Object, "spec", "dnsNames")
		for _, n := range dnsNames {
			if _, ok := hostSet[strings.ToLower(n)]; ok {
				return target
			}
		}
	}
	return ""
}

func certificateIsReady(cert unstructured.Unstructured) bool {
	conds, _, _ := unstructured.NestedSlice(cert.Object, "status", "conditions")
	for _, raw := range conds {
		cm, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if cm["type"] == "Ready" && cm["status"] == "True" {
			return true
		}
	}
	return false
}

// base64Decode handles the "data" value form from a snapshot capture where
// tls.crt may come through as a base64-encoded PEM string instead of raw PEM.
func base64Decode(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(strings.TrimSpace(s))
}
