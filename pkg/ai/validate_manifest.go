// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package ai

import (
	"errors"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

// Phase 2d-δ safe-apply validator. Sits between the OSS analyzer's
// `Diagnostic.ProposedPolicyYAML` field and the srenix-enterprise bridge that
// mints approval URLs. Refuses any YAML that doesn't match an allowed
// Kind + per-Kind shape constraint set.
//
// The validator is the load-bearing safety check for ActionApplyManifest.
// If it lets a dangerous manifest through, an approving SRE click
// becomes an arbitrary mutation. Every allowed Kind must have an
// explicit shape validator below.

// AllowedManifestKinds is the closed set of Kinds the ApplyManifest
// validator accepts. Extending this list requires a per-Kind security
// review of the corresponding shape validator.
//
// Initial set (v1.15.0):
//   - NetworkPolicy: the OSS SnapshotProposer's output kind. Validated
//     against the NetworkPolicyProposer's safe-by-default shape.
//
// Future Kinds (per security review):
//   - RoleBinding (with strict roleRef.kind != ClusterRole + subjects
//     must be ServiceAccount only)
//   - ConfigMap (with key-name patterns blocking credential placement)
var AllowedManifestKinds = map[string]struct{}{
	"NetworkPolicy": {},
}

// Errors returned by ValidateManifest.
var (
	ErrManifestEmpty             = errors.New("ai: manifest_yaml is empty")
	ErrManifestParseFailed       = errors.New("ai: manifest_yaml failed to parse as YAML")
	ErrManifestKindNotAllowed    = errors.New("ai: manifest kind is not in the allowed set (see pkg/ai.AllowedManifestKinds)")
	ErrManifestMissingMetadata   = errors.New("ai: manifest is missing metadata.name and/or metadata.namespace")
	ErrManifestProtectedNS       = errors.New("ai: manifest targets a protected namespace")
	ErrManifestDangerousShape    = errors.New("ai: manifest contains a dangerous shape (privileged, hostNetwork, hostPath, wildcard verbs, etc.)")
	ErrManifestMissingAPIVersion = errors.New("ai: manifest is missing apiVersion")
)

// ValidateManifest parses ManifestYAML, refuses anything not in the
// allowed Kind set, and runs Kind-specific shape validators. Returns
// nil only when the manifest is safe to `kubectl apply`.
//
// Defense-in-depth: the validator never mutates the YAML — it only
// reads + rejects. The approval-server's executor applies the exact
// bytes the proposer signed.
func ValidateManifest(yamlBytes []byte) error {
	if len(yamlBytes) == 0 {
		return ErrManifestEmpty
	}

	// Parse into an unstructured object for shape inspection.
	var raw map[string]any
	if err := yaml.Unmarshal(yamlBytes, &raw); err != nil {
		return fmt.Errorf("%w: %v", ErrManifestParseFailed, err)
	}
	if raw == nil {
		return ErrManifestParseFailed
	}
	u := unstructured.Unstructured{Object: raw}

	apiVersion := u.GetAPIVersion()
	kind := u.GetKind()
	if apiVersion == "" {
		return ErrManifestMissingAPIVersion
	}
	if kind == "" {
		return fmt.Errorf("%w: kind not specified", ErrManifestKindNotAllowed)
	}

	// Closed-Kind whitelist.
	if _, ok := AllowedManifestKinds[kind]; !ok {
		return fmt.Errorf("%w: %q", ErrManifestKindNotAllowed, kind)
	}

	// Namespace and name MUST be present (the apply path needs both).
	if u.GetName() == "" {
		return ErrManifestMissingMetadata
	}
	ns := u.GetNamespace()
	if ns == "" {
		return ErrManifestMissingMetadata
	}
	if IsProtectedNamespace(ns) {
		return fmt.Errorf("%w: %q", ErrManifestProtectedNS, ns)
	}

	// Per-Kind shape validation.
	switch kind {
	case "NetworkPolicy":
		return validateNetworkPolicyShape(&u)
	}
	// Should be unreachable given the whitelist check above; if a Kind
	// is in AllowedManifestKinds but has no shape validator, refuse —
	// that's a bug, not a feature.
	return fmt.Errorf("%w: no shape validator for %q (programming error)", ErrManifestKindNotAllowed, kind)
}

// validateNetworkPolicyShape enforces the safe-by-default shape that
// the OSS SnapshotProposer emits (v1.13.0+):
//
//   - apiVersion: networking.k8s.io/v1
//   - spec.policyTypes MAY include Ingress; MAY NOT include Egress
//     (egress restriction is opt-in via spec.policyTypes; default-
//     unrestricted preserves DNS / in-cluster service / external API
//     access — same constraint the v1.13.0 hardening codifies)
//   - spec.ingress[].from[].ipBlock.cidr MUST NOT be a private CIDR
//     other than 0.0.0.0/0 (block accidents like 10.0.0.0/8 unless
//     the operator explicitly authorizes via a future field)
//   - No `spec.egress` rules (egress allowed by absence of Egress
//     in policyTypes; explicit egress rules are out of scope for the
//     OSS proposer)
//
// This list is INTENTIONALLY narrow — the proposer should refuse
// shapes the OSS analyzer doesn't produce. Operators who want broader
// policies write them by hand, not via the AI-approval path.
func validateNetworkPolicyShape(u *unstructured.Unstructured) error {
	if u.GetAPIVersion() != "networking.k8s.io/v1" {
		return fmt.Errorf("%w: NetworkPolicy must use apiVersion=networking.k8s.io/v1, got %q",
			ErrManifestDangerousShape, u.GetAPIVersion())
	}
	policyTypes, _, _ := unstructured.NestedStringSlice(u.Object, "spec", "policyTypes")
	for _, pt := range policyTypes {
		if pt == "Egress" {
			return fmt.Errorf("%w: spec.policyTypes contains Egress; the proposer never restricts egress (DNS / cluster / external must work)",
				ErrManifestDangerousShape)
		}
	}
	// Egress rules forbidden too — defense-in-depth in case policyTypes
	// is empty but spec.egress is set.
	egress, found, _ := unstructured.NestedSlice(u.Object, "spec", "egress")
	if found && len(egress) > 0 {
		return fmt.Errorf("%w: spec.egress[] not allowed in OSS-proposed NetworkPolicies", ErrManifestDangerousShape)
	}

	// ipBlock CIDR scan — 0.0.0.0/0 is fine (the only public-internet
	// allow the proposer uses); other private CIDRs are suspicious.
	ingress, _, _ := unstructured.NestedSlice(u.Object, "spec", "ingress")
	for _, ir := range ingress {
		irMap, ok := ir.(map[string]any)
		if !ok {
			continue
		}
		from, _, _ := unstructured.NestedSlice(irMap, "from")
		for _, f := range from {
			fMap, ok := f.(map[string]any)
			if !ok {
				continue
			}
			cidr, found, _ := unstructured.NestedString(fMap, "ipBlock", "cidr")
			if !found {
				continue
			}
			if !isSafeIPBlockCIDR(cidr) {
				return fmt.Errorf("%w: ipBlock cidr %q not allowed (only 0.0.0.0/0 permitted)",
					ErrManifestDangerousShape, cidr)
			}
		}
	}
	return nil
}

// isSafeIPBlockCIDR returns true when the proposer is allowed to use
// `cidr` in an ingress allow rule. Only 0.0.0.0/0 (the public-internet
// catchall used for LoadBalancer port allows) is currently permitted.
// Private CIDRs (10.0.0.0/8, 192.168.0.0/16, etc.) are blocked because
// the OSS analyzer can't reason about whether a given private CIDR is
// the cluster's actual pod / node / service CIDR.
func isSafeIPBlockCIDR(cidr string) bool {
	s := strings.TrimSpace(cidr)
	return s == "0.0.0.0/0"
}
