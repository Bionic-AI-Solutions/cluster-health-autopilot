// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package ai_test

import (
	"context"
	"strings"
	"testing"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/ai"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/diagnose"
)

const goodNetPolYAML = `apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: default-deny
  namespace: workloads
spec:
  podSelector: {}
  policyTypes: [Ingress]
  ingress:
  - from:
    - ipBlock:
        cidr: 0.0.0.0/0
    ports:
    - protocol: TCP
      port: 8080
`

const dangerousEgressYAML = `apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: bad-egress
  namespace: workloads
spec:
  podSelector: {}
  policyTypes: [Ingress, Egress]
`

const unsupportedKindYAML = `apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: rb
  namespace: workloads
roleRef:
  kind: Role
  name: r
  apiGroup: rbac.authorization.k8s.io
subjects:
- kind: ServiceAccount
  name: sa
  namespace: workloads
`

const protectedNSYAML = `apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: bad-ns
  namespace: kube-system
spec:
  podSelector: {}
  policyTypes: [Ingress]
`

func TestManifestBridge_NameSatisfiesFixProposer(t *testing.T) {
	var _ ai.FixProposer = ai.ManifestBridge{}
	if name := (ai.ManifestBridge{}).Name(); name != "ManifestBridge" {
		t.Errorf("Name(): want ManifestBridge got %q", name)
	}
}

func TestManifestBridge_Propose_HappyPath(t *testing.T) {
	d := diagnose.Diagnostic{
		Subject:            "Namespace/cluster/workloads/missing-network-policy",
		Source:             "NetworkPolicyProposer",
		ProposedPolicyYAML: goodNetPolYAML,
	}
	prop, err := (ai.ManifestBridge{}).Propose(context.Background(), d)
	if err != nil {
		t.Fatalf("Propose: %v", err)
	}
	if prop == nil {
		t.Fatal("Propose: nil for valid YAML")
	}
	if prop.ActionKind != ai.ActionApplyManifest {
		t.Errorf("ActionKind: want ApplyManifest got %q", prop.ActionKind)
	}
	if string(prop.ManifestYAML) != goodNetPolYAML {
		t.Errorf("ManifestYAML round-trip mismatch")
	}
	if prop.Target.Kind != "NetworkPolicy" || prop.Target.Namespace != "workloads" || prop.Target.Name != "default-deny" {
		t.Errorf("Target wrong: %+v", prop.Target)
	}
	if !strings.Contains(prop.Rollback.Description, "kubectl delete networkpolicy") {
		t.Errorf("Rollback should suggest kubectl delete; got %q", prop.Rollback.Description)
	}
	if prop.DiagnosticSubject != d.Subject {
		t.Errorf("DiagnosticSubject: want %q got %q", d.Subject, prop.DiagnosticSubject)
	}
	if prop.Tier != ai.TierT1 {
		t.Errorf("Tier: want T1 got %v", prop.Tier)
	}
	// Returned proposal must independently pass Validate (defense-in-depth).
	if verr := prop.Validate(); verr != nil {
		t.Errorf("returned proposal fails Validate: %v", verr)
	}
}

func TestManifestBridge_Propose_NoYAMLReturnsNil(t *testing.T) {
	d := diagnose.Diagnostic{
		Subject: "Pod/x/y",
		Source:  "SomeOtherAnalyzer",
		// no ProposedPolicyYAML
	}
	prop, err := (ai.ManifestBridge{}).Propose(context.Background(), d)
	if err != nil {
		t.Errorf("nil-YAML path should not error; got %v", err)
	}
	if prop != nil {
		t.Errorf("expected nil proposal for empty YAML; got %+v", prop)
	}
}

func TestManifestBridge_Propose_RefusesUnsafeShapes(t *testing.T) {
	cases := map[string]string{
		"egress-in-policyTypes": dangerousEgressYAML,
		"unsupported-kind":      unsupportedKindYAML,
		"protected-namespace":   protectedNSYAML,
		"non-yaml":              "not yaml ::",
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			d := diagnose.Diagnostic{
				Subject:            "Namespace/cluster/x",
				Source:             "test",
				ProposedPolicyYAML: in,
			}
			prop, err := (ai.ManifestBridge{}).Propose(context.Background(), d)
			if err != nil {
				t.Errorf("Propose should drop quietly, not error; got %v", err)
			}
			if prop != nil {
				t.Errorf("%s: expected nil (refused), got %+v", name, prop)
			}
		})
	}
}

func TestBuildApplyManifestProposal_MissingMetadataReturnsNil(t *testing.T) {
	cases := map[string]string{
		"no-name":      `apiVersion: networking.k8s.io/v1` + "\n" + `kind: NetworkPolicy` + "\n" + `metadata: {namespace: workloads}`,
		"no-namespace": `apiVersion: networking.k8s.io/v1` + "\n" + `kind: NetworkPolicy` + "\n" + `metadata: {name: x}`,
		"no-kind":      `apiVersion: networking.k8s.io/v1` + "\n" + `metadata: {name: x, namespace: workloads}`,
		"empty":        ``,
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			d := diagnose.Diagnostic{
				Subject:            "x/y/z",
				Source:             "test",
				ProposedPolicyYAML: in,
			}
			if got := ai.BuildApplyManifestProposal(d); got != nil {
				t.Errorf("%s: expected nil, got %+v", name, got)
			}
		})
	}
}
