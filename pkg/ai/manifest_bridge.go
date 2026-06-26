// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"

	"github.com/srenix-ai/agentic-sre/pkg/diagnose"
)

// ManifestBridge turns Diagnostic.ProposedPolicyYAML into a signable
// AIProposedAction with ActionKind=ActionApplyManifest. Used by analyzers
// that emit deterministic remediation YAML (the v1.13.0+ NetworkPolicy
// proposer is the first; others extend the list).
//
// It implements the FixProposer interface so a watcher can register it
// directly (typically as a fallback when no other FixProposer matches):
//
//	reg.RegisterFixProposer(ai.ManifestBridge{})
//
// When the bridge is registered + a Signer + approval-server URL are
// configured, the watcher's enrichDiagnostics path mints signed
// approve/deny URLs that flow through the existing Slack / Alertmanager
// / DriftReport adapters — exactly the same delivery surface as any
// other AI-proposed fix. This closes the gap where ProposedPolicyYAML
// was emitted by OSS analyzers but had no URL minted by OSS, leaving
// the SRE without click-to-fix on otherwise-actionable findings.
//
// Validation is delegated to ValidateManifest — refuses anything
// outside the closed Kind/shape whitelist. Unsafe shapes silently
// drop to a nil proposal rather than mint a URL on a dangerous YAML.
type ManifestBridge struct {
	// ProposalTTL overrides DefaultProposalTTL for the approval link
	// ExpiresAt. Wire from --approval-ttl so on-call SREs have time to
	// respond. Zero = DefaultProposalTTL (15 min).
	ProposalTTL time.Duration
}

// Name satisfies FixProposer.
func (m ManifestBridge) Name() string { return "ManifestBridge" }

// Propose constructs an ApplyManifest action from `d.ProposedPolicyYAML`
// when present and safe. Returns nil when:
//   - the diagnostic has no proposed YAML (the common case — only the
//     handful of YAML-emitting analyzers populate this field)
//   - the YAML fails the safe-apply validator (Egress in policyTypes,
//     non-`0.0.0.0/0` ipBlock, protected namespace, unsupported Kind…)
func (m ManifestBridge) Propose(_ context.Context, d diagnose.Diagnostic) (*AIProposedAction, error) {
	if d.ProposedPolicyYAML == "" {
		return nil, nil
	}
	return BuildApplyManifestProposalWithTTL(d, m.ProposalTTL), nil
}

// BuildApplyManifestProposal is the underlying builder. Exposed for
// callers that want the bridge logic without going through the
// FixProposer interface (e.g. srenix-enterprise's diagnose subcommand renders
// proposals inline). Returns nil when validation refuses the YAML.
//
// Uses DefaultProposalTTL (15 min). For a configurable TTL, use
// BuildApplyManifestProposalWithTTL.
func BuildApplyManifestProposal(d diagnose.Diagnostic) *AIProposedAction {
	return BuildApplyManifestProposalWithTTL(d, 0)
}

// BuildApplyManifestProposalWithTTL is BuildApplyManifestProposal with a
// configurable TTL. ttl=0 falls back to DefaultProposalTTL. Wire
// --approval-ttl here so NetworkPolicy approval links stay alive long
// enough for an on-call SRE to act on them.
func BuildApplyManifestProposalWithTTL(d diagnose.Diagnostic, ttl time.Duration) *AIProposedAction {
	yamlBytes := []byte(d.ProposedPolicyYAML)
	target, kindLabel, err := parseManifestTarget(yamlBytes)
	if err != nil {
		return nil
	}
	if ttl <= 0 {
		ttl = DefaultProposalTTL
	}
	now := time.Now().UTC()
	action := AIProposedAction{
		ActionID:     d.Subject + ":apply-manifest",
		ActionKind:   ActionApplyManifest,
		Target:       ObjectRef(target),
		ManifestYAML: yamlBytes,
		Rationale:    fmt.Sprintf("Apply %s emitted by %s analyzer", kindLabel, d.Source),
		Rollback: RollbackInfo{
			Description: fmt.Sprintf("kubectl delete %s -n %s %s",
				strings.ToLower(target.Kind), target.Namespace, target.Name),
		},
		DiagnosticSubject: d.Subject,
		Tier:              TierT1,
		CreatedAt:         now,
		ExpiresAt:         now.Add(ttl),
	}
	if err := action.Validate(); err != nil {
		return nil
	}
	return &action
}

// manifestTarget is the (Kind, Namespace, Name) parsed from a manifest.
type manifestTarget struct {
	Kind      string
	Namespace string
	Name      string
}

// parseManifestTarget extracts the target from manifest metadata. The
// validator (ValidateManifest) enforces the same invariants — this is
// belt-and-suspenders so the resulting AIProposedAction has a
// non-empty Target without the validator having to NPE on null fields.
func parseManifestTarget(yamlBytes []byte) (manifestTarget, string, error) {
	var raw map[string]any
	if err := yaml.Unmarshal(yamlBytes, &raw); err != nil {
		return manifestTarget{}, "", fmt.Errorf("manifest-bridge: yaml parse: %w", err)
	}
	if raw == nil {
		return manifestTarget{}, "", fmt.Errorf("manifest-bridge: empty manifest")
	}
	u := unstructured.Unstructured{Object: raw}
	t := manifestTarget{
		Kind:      u.GetKind(),
		Namespace: u.GetNamespace(),
		Name:      u.GetName(),
	}
	if t.Kind == "" || t.Namespace == "" || t.Name == "" {
		return manifestTarget{}, "", fmt.Errorf("manifest-bridge: missing kind/namespace/name in metadata")
	}
	return t, t.Kind, nil
}
