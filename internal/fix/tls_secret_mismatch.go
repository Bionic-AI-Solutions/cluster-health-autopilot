// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package fix

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
	"k8s.io/apimachinery/pkg/types"
)

// TLSSecretMismatch repoints an Ingress's `spec.tls[].secretName` from a
// stale Secret to the cert-manager-managed Secret that holds a healthy cert
// for the same host. Pairs with diagnose.TLSSecretMismatch (which detects).
//
// Safety contract:
//   - OFF by default. Operator opts in by registering this fixer in the
//     catalog or via Helm value `fixers.tlsSecretMismatch.enabled=true`.
//   - Skipped automatically for GitOps-managed Ingresses (ArgoCD / Flux /
//     Helm) — patching those would fight the reconcile loop.
//   - Skipped for protected namespaces.
//   - Requires high-confidence match: stale Secret is actually expired (or
//     expiring within the window), AND a cert-manager Certificate in the
//     same namespace is Ready=True with the same host in dnsNames pointing
//     at a different Secret.
type TLSSecretMismatch struct {
	// ExpiryWindow mirrors diagnose.TLSSecretMismatch — only act on certs
	// already expired or expiring within this window. Default: 14 days.
	ExpiryWindow time.Duration
}

// Name returns the fixer's identifier.
func (TLSSecretMismatch) Name() string { return "TLSSecretMismatch" }

// gitopsAnnotations are the Ingress annotations that signal a GitOps-managed
// resource. If any of these is present, the fixer refuses to mutate — the
// edit must go through the source repo.
var gitopsAnnotations = []string{
	"argocd.argoproj.io/instance",
	"argocd.argoproj.io/tracking-id",
	"kustomize.toolkit.fluxcd.io/name",
	"kustomize.toolkit.fluxcd.io/namespace",
	"meta.helm.sh/release-name",
	"meta.helm.sh/release-namespace",
}

// gitopsLabels parallel gitopsAnnotations for label-based labeling schemes.
var gitopsLabels = []string{
	"app.kubernetes.io/managed-by", // values: "Helm", "argocd", "Flux"
}

// Run executes the fixer.
func (f TLSSecretMismatch) Run(ctx context.Context, src snapshot.Source, m snapshot.Mutator) Result {
	r := Result{Fixer: "TLSSecretMismatch"}
	if m == nil {
		r.Refused = "snapshot mode — fixers require live cluster access"
		return r
	}
	window := f.ExpiryWindow
	if window == 0 {
		window = 14 * 24 * time.Hour
	}
	now := time.Now()

	ingresses, err := src.List(ctx, snapshot.GVRIngress, "")
	if err != nil || len(ingresses.Items) == 0 {
		return r
	}

	for i := range ingresses.Items {
		ing := ingresses.Items[i]
		ns := ing.GetNamespace()
		name := ing.GetName()
		ingObj := "Ingress/" + ns + "/" + name

		if IsProtectedNamespace(ns) {
			r.Skipped = append(r.Skipped, SkipReason{Object: ingObj, Reason: "protected namespace"})
			continue
		}
		if reason := isGitOpsManaged(ing); reason != "" {
			r.Skipped = append(r.Skipped, SkipReason{
				Object: ingObj,
				Reason: "GitOps-managed: " + reason + " — edit the source repo instead",
			})
			continue
		}

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
			hosts := tlsHostsFix(tm)
			if len(hosts) == 0 {
				continue
			}

			expiry, ok := secretCertExpiryFix(ctx, src, ns, secretName)
			if !ok {
				continue
			}
			expired := !expiry.IsZero() && now.After(expiry)
			soon := !expiry.IsZero() && !expired && expiry.Sub(now) < window
			if !expired && !soon {
				continue
			}

			better := findMismatchedCertFix(ctx, src, ns, hosts, secretName)
			if better == "" {
				continue
			}

			patch := []byte(fmt.Sprintf(
				`[{"op":"replace","path":"/spec/tls/%d/secretName","value":%q}]`,
				tlsIdx, better,
			))
			if err := m.Patch(ctx, snapshot.GVRIngress, ns, name, types.JSONPatchType, patch); err != nil {
				r.Skipped = append(r.Skipped, SkipReason{
					Object: ingObj,
					Reason: "patch failed: " + err.Error(),
				})
				continue
			}
			r.Actions = append(r.Actions, Action{
				Description: fmt.Sprintf(
					"Repointed `%s` tls[%d] from Secret `%s` (stale) → `%s` (cert-manager-managed, healthy)",
					ingObj, tlsIdx, secretName, better,
				),
				Object: ingObj,
			})
		}
	}
	return r
}

// isGitOpsManaged inspects annotations and labels for GitOps controllers.
// Returns a human-readable reason string when managed, or "" when CHA may
// safely mutate.
func isGitOpsManaged(ing unstructured.Unstructured) string {
	annotations := ing.GetAnnotations()
	for _, k := range gitopsAnnotations {
		if v, ok := annotations[k]; ok && v != "" {
			return k + "=" + v
		}
	}
	labels := ing.GetLabels()
	for _, k := range gitopsLabels {
		if v, ok := labels[k]; ok {
			// Helm chart releases own the resource via Helm release reconcile.
			// Argo / Flux similarly mark via this label.
			switch strings.ToLower(v) {
			case "helm", "argocd", "flux", "fluxcd":
				return k + "=" + v
			}
		}
	}
	return ""
}

// tlsHostsFix mirrors diagnose.tlsHosts — kept package-local to avoid a
// circular package dependency between diagnose and fix.
func tlsHostsFix(tm map[string]any) []string {
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

// secretCertExpiryFix mirrors diagnose.secretCertExpiry. The cert-parsing
// logic is duplicated to keep the fix package free of cross-internal imports.
func secretCertExpiryFix(ctx context.Context, src snapshot.Source, ns, name string) (time.Time, bool) {
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
	pemBytes := []byte(raw)
	if block, _ := pem.Decode(pemBytes); block == nil {
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(raw))
		if err != nil {
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

// findMismatchedCertFix mirrors diagnose.findMismatchedManagedCert.
func findMismatchedCertFix(ctx context.Context, src snapshot.Source, ns string, hosts []string, currentSecret string) string {
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
		if !certReadyFix(cert) {
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

func certReadyFix(cert unstructured.Unstructured) bool {
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
