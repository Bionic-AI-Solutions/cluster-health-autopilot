// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package netpol

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// gvrNetworkPolicy is re-aliased from snapshot pkg to keep this file's
// imports tight (single canonical alias).
var gvrNetworkPolicy = snapshot.GVRNetworkPolicy

// Proposal is what a Proposer emits: a ready-to-apply NetworkPolicy YAML
// plus enough context for an SRE to decide.
//
// The OSS proposer never APPLIES the proposal. The paid AI tier wraps
// it as an ApprovalProposal CR and renders Approve/Deny in Slack
// (v1.10.4 pattern). On Approve click, cha-com's approval-server reads
// the CR's PolicyYAML and applies it.
type Proposal struct {
	// Namespace this proposal targets.
	Namespace string

	// PolicyKind is always "NetworkPolicy" for now. Future proposers may
	// emit other kinds (PodDisruptionBudget, ResourceQuota) — the field
	// is here so the approval-server can route to the right apply path.
	PolicyKind string

	// PolicyName is the spec.metadata.name of the generated resource.
	PolicyName string

	// PolicyYAML is the rendered manifest, ready for `kubectl apply -f -`.
	// One resource per proposal — keep it simple.
	PolicyYAML string

	// Rationale is a human-readable summary of WHY this shape. Goes into
	// the Slack Details link. Examples: "default-deny + allow from same
	// namespace + allow from kong/* (5 Ingresses route to this ns)".
	Rationale string

	// AllowSources is a structured list of the allow rules emitted, for
	// machine consumption + adversarial verification.
	AllowSources []AllowRule
}

// AllowRule is one structured allow entry inside the proposal — useful
// for testing and for the cha-com judge layer that verifies proposals
// before surfacing them.
type AllowRule struct {
	// Kind ∈ {"same-namespace", "controller-namespace", "labeled-pod"}.
	Kind string

	// Namespace is the source namespace name; "*" for same-namespace.
	Namespace string

	// Why is one-sentence justification ("Kong Ingresses found targeting
	// this ns").
	Why string
}

// Proposer generates NetworkPolicy proposals from cluster snapshot
// state. The OSS build ships SnapshotProposer. The paid AI tier
// replaces this with an RAG-grounded proposer that uses kind=baseline
// observations to refine allow lists.
type Proposer interface {
	// ProposeForNamespace returns a single Proposal or (nil, nil) when
	// the proposer chose not to emit (e.g. namespace already has a
	// NetPol; CNI doesn't enforce). Errors are reserved for unexpected
	// snapshot failures.
	ProposeForNamespace(ctx context.Context, src snapshot.Source, namespace string) (*Proposal, error)
}

// NoopProposer is the OSS-default-when-disabled. Returns nothing.
type NoopProposer struct{}

// ProposeForNamespace always returns (nil, nil).
func (NoopProposer) ProposeForNamespace(context.Context, snapshot.Source, string) (*Proposal, error) {
	return nil, nil
}

// SnapshotProposer is the deterministic OSS proposer. Generates a
// safe stub from current cluster state:
//
//  1. Default-deny ingress (policyTypes: [Ingress])
//  2. Allow from same namespace (podSelector: {})
//  3. Allow from each detected Ingress-controller namespace (if any
//     Ingress in the cluster has a backend Service inside the target
//     namespace, we infer that traffic from the controller's ns must be
//     allowed)
//
// Egress is NOT restricted — DNS, in-cluster service calls, and
// external API access all keep working.
//
// This proposer never proposes for namespaces that:
//   - Already have at least one NetworkPolicy (let humans / paid tier
//     refine instead of stomping)
//   - Are system namespaces (kube-system / kube-public / kube-node-lease)
//   - Have zero pods (nothing to govern)
type SnapshotProposer struct {
	// IngressControllerNamespaces are fallback names when we can't
	// detect from Ingress objects. Defaults to ["kong", "ingress-nginx",
	// "traefik", "istio-system", "kong-system"].
	IngressControllerNamespaces []string
}

// systemNamespaces are the kube* namespaces the proposer should never
// touch. Mirrors the SecurityDrift analyzer's skip list intentionally.
var systemNamespaces = map[string]struct{}{
	"kube-system":     {},
	"kube-public":     {},
	"kube-node-lease": {},
}

// commonIngressControllerNamespaces is the fallback list when Ingress
// discovery returns nothing.
var commonIngressControllerNamespaces = []string{
	"kong", "ingress-nginx", "traefik", "istio-system", "kong-system",
}

// ProposeForNamespace implements Proposer.
func (p SnapshotProposer) ProposeForNamespace(ctx context.Context, src snapshot.Source, namespace string) (*Proposal, error) {
	if _, isSystem := systemNamespaces[namespace]; isSystem {
		return nil, nil
	}

	// Skip if there's already a NetworkPolicy in this namespace.
	existing, _ := src.List(ctx, gvrNetworkPolicy, namespace)
	if existing != nil && len(existing.Items) > 0 {
		return nil, nil
	}

	// Skip if zero pods.
	pods, _ := src.List(ctx, snapshot.GVRPod, namespace)
	if pods == nil || len(pods.Items) == 0 {
		return nil, nil
	}

	// Detect Ingress-controller namespaces. Look for Ingress objects
	// CLUSTER-WIDE whose backend Service is in `namespace` — the
	// controller's namespace is the one we should allow from.
	controllerNS := p.detectControllerNamespaces(ctx, src, namespace)
	if len(controllerNS) == 0 {
		// No Ingresses target this ns → only allow from same namespace.
		// Conservative; the proposer can be refined later.
		controllerNS = nil
	}

	policyName := "cha-proposed-allow-intracluster"
	allows := []AllowRule{
		{Kind: "same-namespace", Namespace: "*", Why: "intra-namespace pod-to-pod"},
	}

	var ingressBlocks []string
	ingressBlocks = append(ingressBlocks, `    - from:
        - podSelector: {}                 # allow from any pod in this namespace`)

	for _, cns := range controllerNS {
		allows = append(allows, AllowRule{
			Kind: "controller-namespace", Namespace: cns,
			Why: "Ingress objects route into " + namespace + " via " + cns,
		})
		ingressBlocks = append(ingressBlocks, fmt.Sprintf(`    - from:
        - namespaceSelector:
            matchExpressions:
              - key: kubernetes.io/metadata.name
                operator: In
                values: ["%s"]
      # %s`, cns, "ingress-controller in "+cns))
	}

	yaml := fmt.Sprintf(`apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: %s
  namespace: %s
  annotations:
    cha.bionicaisolutions.com/proposer: snapshot-proposer
    cha.bionicaisolutions.com/policy-tier: default-deny-plus-observed-allows
spec:
  podSelector: {}
  policyTypes:
    - Ingress
  ingress:
%s
`, policyName, namespace, strings.Join(ingressBlocks, "\n"))

	rationale := "default-deny ingress + allow-from same namespace"
	if len(controllerNS) > 0 {
		rationale += " + allow-from " + strings.Join(controllerNS, ",") +
			" (Ingress controllers routing into this ns)"
	}

	return &Proposal{
		Namespace:    namespace,
		PolicyKind:   "NetworkPolicy",
		PolicyName:   policyName,
		PolicyYAML:   yaml,
		Rationale:    rationale,
		AllowSources: allows,
	}, nil
}

// detectControllerNamespaces walks every Ingress in the cluster and
// returns the unique set of Ingress namespaces whose backend services
// live in `targetNS`. In a clean cluster, this is the set of ingress-
// controller namespaces (kong, traefik, etc.) that should be allowed
// to talk to `targetNS`.
//
// Falls back to commonIngressControllerNamespaces when no Ingress data
// is available — better safe than locked-out.
func (p SnapshotProposer) detectControllerNamespaces(ctx context.Context, src snapshot.Source, targetNS string) []string {
	ings, _ := src.List(ctx, snapshot.GVRIngress, "")
	if ings == nil {
		return p.fallbackNamespaces()
	}
	found := map[string]struct{}{}
	for i := range ings.Items {
		ing := &ings.Items[i]
		if ing.GetNamespace() != targetNS {
			// We're looking for Ingresses that point at `targetNS`;
			// in standard K8s an Ingress's backend Service must live
			// in the SAME ns as the Ingress, so the controller-ns is
			// always the Ingress's own ns.
			//
			// Cross-ns backends are exotic (Kubernetes 1.30+
			// PortableHTTPRoute) — out of scope for the deterministic
			// proposer.
			continue
		}
		// Look up the IngressClass → controller namespace mapping if
		// possible. For now, derive from the IngressClassName.
		className, _, _ := unstructured.NestedString(ing.Object, "spec", "ingressClassName")
		switch className {
		case "kong":
			found["kong"] = struct{}{}
		case "nginx":
			found["ingress-nginx"] = struct{}{}
		case "traefik":
			found["traefik"] = struct{}{}
		default:
			// Unknown class — assume kong (most common in this
			// deployment) but operators should refine via the CR.
			found["kong"] = struct{}{}
		}
	}
	if len(found) == 0 {
		return nil // no Ingresses → don't add controller-ns allows
	}
	out := make([]string, 0, len(found))
	for k := range found {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// fallbackNamespaces returns the operator-supplied or default list.
func (p SnapshotProposer) fallbackNamespaces() []string {
	if len(p.IngressControllerNamespaces) > 0 {
		out := append([]string{}, p.IngressControllerNamespaces...)
		sort.Strings(out)
		return out
	}
	out := append([]string{}, commonIngressControllerNamespaces...)
	sort.Strings(out)
	return out
}
