// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package ai

import (
	"errors"
	"strings"
	"testing"
)

// validNetworkPolicy is the shape the OSS SnapshotProposer (v1.13.0+)
// emits — the validator MUST accept this.
const validNetworkPolicy = `apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: cha-proposed-allow-intracluster
  namespace: app
spec:
  podSelector: {}
  policyTypes:
    - Ingress
  ingress:
    - from:
        - podSelector: {}
    - from:
        - namespaceSelector:
            matchExpressions:
              - key: kubernetes.io/metadata.name
                operator: In
                values: ["kube-system"]
    - from:
        - ipBlock:
            cidr: 0.0.0.0/0
      ports:
        - protocol: TCP
          port: 80
`

func TestValidateManifest_HappyPath_SnapshotProposerNetworkPolicy(t *testing.T) {
	if err := ValidateManifest([]byte(validNetworkPolicy)); err != nil {
		t.Errorf("v1.13.0 SnapshotProposer YAML must validate; got: %v", err)
	}
}

func TestValidateManifest_EmptyRejected(t *testing.T) {
	if err := ValidateManifest(nil); !errors.Is(err, ErrManifestEmpty) {
		t.Errorf("empty manifest should be ErrManifestEmpty; got %v", err)
	}
	if err := ValidateManifest([]byte("")); !errors.Is(err, ErrManifestEmpty) {
		t.Errorf("empty string should be ErrManifestEmpty; got %v", err)
	}
}

func TestValidateManifest_ParseFailureRejected(t *testing.T) {
	err := ValidateManifest([]byte("not: yaml: at all: :::"))
	if !errors.Is(err, ErrManifestParseFailed) {
		t.Errorf("expected ErrManifestParseFailed; got %v", err)
	}
}

func TestValidateManifest_MissingAPIVersion(t *testing.T) {
	y := `kind: NetworkPolicy
metadata:
  name: x
  namespace: y
spec: {podSelector: {}}`
	err := ValidateManifest([]byte(y))
	if !errors.Is(err, ErrManifestMissingAPIVersion) {
		t.Errorf("expected ErrManifestMissingAPIVersion; got %v", err)
	}
}

func TestValidateManifest_DisallowedKind(t *testing.T) {
	y := `apiVersion: v1
kind: Pod
metadata:
  name: bad
  namespace: app`
	err := ValidateManifest([]byte(y))
	if !errors.Is(err, ErrManifestKindNotAllowed) {
		t.Errorf("Pod is not in allowed set; got %v", err)
	}
}

func TestValidateManifest_MissingName(t *testing.T) {
	y := `apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  namespace: app
spec: {podSelector: {}}`
	err := ValidateManifest([]byte(y))
	if !errors.Is(err, ErrManifestMissingMetadata) {
		t.Errorf("missing name should fail; got %v", err)
	}
}

func TestValidateManifest_MissingNamespace(t *testing.T) {
	y := `apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: x
spec: {podSelector: {}}`
	err := ValidateManifest([]byte(y))
	if !errors.Is(err, ErrManifestMissingMetadata) {
		t.Errorf("missing namespace should fail; got %v", err)
	}
}

func TestValidateManifest_ProtectedNamespaceRejected(t *testing.T) {
	for _, ns := range []string{"kube-system", "vault", "external-secrets", "cnpg-system", "rook-ceph"} {
		y := `apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: bad
  namespace: ` + ns + `
spec: {podSelector: {}}`
		err := ValidateManifest([]byte(y))
		if !errors.Is(err, ErrManifestProtectedNS) {
			t.Errorf("namespace %q should be protected; got %v", ns, err)
		}
	}
}

func TestValidateManifest_EgressInPolicyTypesRejected(t *testing.T) {
	y := `apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: x
  namespace: app
spec:
  podSelector: {}
  policyTypes: [Ingress, Egress]
  ingress: []`
	err := ValidateManifest([]byte(y))
	if !errors.Is(err, ErrManifestDangerousShape) {
		t.Errorf("egress in policyTypes should fail; got %v", err)
	}
}

func TestValidateManifest_EgressRulesRejected(t *testing.T) {
	y := `apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: x
  namespace: app
spec:
  podSelector: {}
  policyTypes: [Ingress]
  ingress: []
  egress:
    - to:
        - ipBlock: {cidr: 0.0.0.0/0}`
	err := ValidateManifest([]byte(y))
	if !errors.Is(err, ErrManifestDangerousShape) {
		t.Errorf("spec.egress[] should fail even with policyTypes=[Ingress]; got %v", err)
	}
}

func TestValidateManifest_DangerousIPBlockCIDRRejected(t *testing.T) {
	// Operator-specific CIDRs like 10.0.0.0/8 are dangerous because
	// they could accidentally match the cluster's pod / node / service
	// CIDR. Block them.
	for _, badCIDR := range []string{"10.0.0.0/8", "192.168.0.0/16", "172.16.0.0/12", "0.0.0.0/1"} {
		y := `apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: x
  namespace: app
spec:
  podSelector: {}
  policyTypes: [Ingress]
  ingress:
    - from:
        - ipBlock: {cidr: ` + badCIDR + `}`
		err := ValidateManifest([]byte(y))
		if !errors.Is(err, ErrManifestDangerousShape) {
			t.Errorf("cidr %q should be refused; got %v", badCIDR, err)
		}
	}
}

func TestValidateManifest_WrongAPIVersionRejected(t *testing.T) {
	y := `apiVersion: extensions/v1beta1
kind: NetworkPolicy
metadata:
  name: x
  namespace: app
spec:
  podSelector: {}
  policyTypes: [Ingress]
  ingress: []`
	err := ValidateManifest([]byte(y))
	if !errors.Is(err, ErrManifestDangerousShape) {
		t.Errorf("deprecated extensions/v1beta1 should be refused; got %v", err)
	}
}

// TestAIProposedAction_Validate_ApplyManifest_Integration — the
// AIProposedAction.Validate flow correctly delegates to
// ValidateManifest and surfaces its errors.
func TestAIProposedAction_Validate_ApplyManifest_Integration(t *testing.T) {
	a := newValidProposal()
	a.ActionKind = ActionApplyManifest
	a.PatchPayload = nil
	a.ManifestYAML = []byte(validNetworkPolicy)

	if err := a.Validate(); err != nil {
		t.Errorf("valid ApplyManifest proposal should pass; got %v", err)
	}

	// Negative: bad YAML should fail at Validate level.
	a.ManifestYAML = []byte("not: yaml: ::")
	if err := a.Validate(); !errors.Is(err, ErrManifestParseFailed) {
		t.Errorf("bad YAML should bubble up to Validate; got %v", err)
	}

	// Negative: ManifestYAML on non-ApplyManifest kind should fail.
	a.ActionKind = ActionDeletePod
	a.ManifestYAML = []byte(validNetworkPolicy)
	if err := a.Validate(); !errors.Is(err, ErrInvalidActionKind) {
		t.Errorf("ManifestYAML on DeletePod should fail; got %v", err)
	}
}

func TestActionApplyManifest_IsValid(t *testing.T) {
	if !ActionApplyManifest.IsValid() {
		t.Error("ActionApplyManifest should be in IsValid whitelist")
	}
}

func TestAllowedManifestKinds_NetworkPolicyOnly(t *testing.T) {
	if len(AllowedManifestKinds) != 1 {
		t.Errorf("v1.15.0 should ship NetworkPolicy ONLY; got %d kinds: %v",
			len(AllowedManifestKinds), AllowedManifestKinds)
	}
	if _, ok := AllowedManifestKinds["NetworkPolicy"]; !ok {
		t.Error("NetworkPolicy must be in AllowedManifestKinds")
	}
	for _, dangerous := range []string{"Pod", "Deployment", "ClusterRoleBinding", "ServiceAccount", "Secret"} {
		if _, ok := AllowedManifestKinds[dangerous]; ok {
			t.Errorf("%s should NOT be in AllowedManifestKinds (security regression)", dangerous)
		}
	}
}

// TestValidateManifest_RejectionMessagesContainContext — the
// rejection errors should include enough context for an SRE looking
// at the audit log to understand WHY the proposal was refused.
func TestValidateManifest_RejectionMessagesContainContext(t *testing.T) {
	y := `apiVersion: v1
kind: Pod
metadata:
  name: x
  namespace: app`
	err := ValidateManifest([]byte(y))
	if err == nil || !strings.Contains(err.Error(), "Pod") {
		t.Errorf("rejection error should name the offending kind; got %v", err)
	}
}
