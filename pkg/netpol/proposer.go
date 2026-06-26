// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package netpol

import (
	"context"
	"fmt"
	"sort"
	"strings"

	pkgai "github.com/srenix-ai/agentic-sre/pkg/ai"
	"github.com/srenix-ai/agentic-sre/internal/snapshot"
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
// (v1.10.4 pattern). On Approve click, srenix-enterprise's approval-server reads
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
// for testing and for the srenix-enterprise judge layer that verifies proposals
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
//  4. **Allow from kube-system** (CoreDNS, kubelet probes, scheduler).
//     Without this, ConfigMap/Secret reload, liveness/readiness probes,
//     and DNS lookups break on policy-enforcing CNIs.
//  5. **Allow from EXTERNAL (0.0.0.0/0) on LoadBalancer / NodePort
//     service ports** — the v1.13.0 hardening that prevents the
//     2026-06-01 class of outage. When a Service of type
//     LoadBalancer or NodePort lives in this namespace, external
//     traffic arrives at the backing pods with the CLIENT source IP
//     (post-MetalLB-ARP) rather than an in-cluster pod IP. A pure
//     `namespaceSelector` allow list will silently DROP that traffic
//     on kube-router / Calico / Cilium, breaking the SaaS surface.
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

	// IncludeSystemNamespaceAllow controls whether the rendered policy
	// includes a `namespaceSelector: kubernetes.io/metadata.name=kube-system`
	// allow rule. Default: true (safe). Set false only for hardened
	// air-gapped deployments where you've explicitly placed kube-system
	// components under their own NetworkPolicies.
	IncludeSystemNamespaceAllow *bool

	// IncludeLoadBalancerExternalAllow controls whether the proposer
	// emits `ipBlock: 0.0.0.0/0` allow rules for ports of Services
	// typed LoadBalancer or NodePort in the namespace. Default: true
	// (REQUIRED for external traffic to reach workloads on policy-
	// enforcing CNIs; pre-1.13.0 omission was the root cause of the
	// 2026-06-01 outage).
	IncludeLoadBalancerExternalAllow *bool
}

// isProtectedNS delegates to the canonical ai.IsProtectedNamespace guard,
// which covers kube-system/public/node-lease + rook-ceph + vault +
// external-secrets + cnpg-system + calico-system + tigera-operator
// and any operator-appended extras (SRENIX_PROTECTED_NAMESPACES_EXTRA).
// This replaces the old local 3-entry map that diverged from the floor.
func isProtectedNS(ns string) bool { return pkgai.IsProtectedNamespace(ns) }

// commonIngressControllerNamespaces is the fallback list when Ingress
// discovery returns nothing.
var commonIngressControllerNamespaces = []string{
	"kong", "ingress-nginx", "traefik", "istio-system", "kong-system",
}

// ProposeForNamespace implements Proposer.
func (p SnapshotProposer) ProposeForNamespace(ctx context.Context, src snapshot.Source, namespace string) (*Proposal, error) {
	if isProtectedNS(namespace) {
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

	policyName := "srenix-proposed-allow-intracluster"
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

	// v1.13.0 hardening #1: allow-from kube-system. CoreDNS, kubelet
	// probes (liveness/readiness), and other system components live in
	// kube-system; without this allow, workloads on policy-enforcing
	// CNIs lose DNS resolution and pod-startup health gates.
	if p.includeSystemNamespaceAllow() {
		allows = append(allows, AllowRule{
			Kind: "controller-namespace", Namespace: "kube-system",
			Why: "CoreDNS + kubelet probes + scheduler health checks",
		})
		ingressBlocks = append(ingressBlocks, `    - from:
        - namespaceSelector:
            matchExpressions:
              - key: kubernetes.io/metadata.name
                operator: In
                values: ["kube-system"]
      # CoreDNS / kubelet probes / scheduler`)
	}

	// v1.13.0 hardening #2: external-traffic allow on LoadBalancer +
	// NodePort service ports. This is the rule the 2026-06-01 outage
	// proved is non-optional on policy-enforcing CNIs.
	if p.includeLoadBalancerExternalAllow() {
		ports := detectExternalPorts(ctx, src, namespace)
		if len(ports) > 0 {
			var portList []string
			for _, p := range ports {
				portList = append(portList, fmt.Sprintf("        - protocol: %s\n          port: %d", p.proto, p.port))
				allows = append(allows, AllowRule{
					Kind:      "external-ipblock",
					Namespace: "0.0.0.0/0",
					Why: fmt.Sprintf("external traffic to %s/%d (LoadBalancer / NodePort exposed)",
						p.proto, p.port),
				})
			}
			ingressBlocks = append(ingressBlocks, fmt.Sprintf(`    - from:
        - ipBlock:
            cidr: 0.0.0.0/0
      ports:
%s
      # external traffic via LoadBalancer / NodePort (MetalLB / cloud LB)`,
				strings.Join(portList, "\n")))
		}
	}

	yaml := fmt.Sprintf(`apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: %s
  namespace: %s
  annotations:
    srenix.ai/proposer: snapshot-proposer
    srenix.ai/policy-tier: default-deny-plus-observed-allows
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

// includeSystemNamespaceAllow returns the effective setting for the
// kube-system allow rule. Defaults to true.
func (p SnapshotProposer) includeSystemNamespaceAllow() bool {
	if p.IncludeSystemNamespaceAllow == nil {
		return true
	}
	return *p.IncludeSystemNamespaceAllow
}

// includeLoadBalancerExternalAllow returns the effective setting for
// the 0.0.0.0/0 external-traffic allow rule on LoadBalancer/NodePort
// ports. Defaults to true (REQUIRED on policy-enforcing CNIs).
func (p SnapshotProposer) includeLoadBalancerExternalAllow() bool {
	if p.IncludeLoadBalancerExternalAllow == nil {
		return true
	}
	return *p.IncludeLoadBalancerExternalAllow
}

// externalPort is one (protocol, port) pair exposed via LoadBalancer
// or NodePort in the target namespace. The deterministic proposer
// emits an `ipBlock: 0.0.0.0/0` allow rule for each one.
type externalPort struct {
	proto string // "TCP" | "UDP" | "SCTP"
	port  int64
}

// detectExternalPorts walks Services in the namespace and returns the
// distinct (proto, port) pairs that are exposed via LoadBalancer or
// NodePort. Returns empty when the namespace has no externally-exposed
// services — in that case the proposer skips the 0.0.0.0/0 rule
// entirely (no external surface to allow).
func detectExternalPorts(ctx context.Context, src snapshot.Source, namespace string) []externalPort {
	svcs, _ := src.List(ctx, snapshot.GVRService, namespace)
	if svcs == nil {
		return nil
	}
	seen := map[externalPort]struct{}{}
	for i := range svcs.Items {
		svc := &svcs.Items[i]
		typ, _, _ := unstructured.NestedString(svc.Object, "spec", "type")
		if typ != "LoadBalancer" && typ != "NodePort" {
			continue
		}
		portsRaw, _, _ := unstructured.NestedSlice(svc.Object, "spec", "ports")
		for _, p := range portsRaw {
			pm, ok := p.(map[string]any)
			if !ok {
				continue
			}
			proto, _ := pm["protocol"].(string)
			if proto == "" {
				proto = "TCP"
			}
			// `port` is required on Service; cast from int64 (the K8s
			// default) or fall through.
			portN, _, _ := unstructured.NestedInt64(pm, "port")
			if portN == 0 {
				continue
			}
			seen[externalPort{proto: proto, port: portN}] = struct{}{}
		}
	}
	if len(seen) == 0 {
		return nil
	}
	out := make([]externalPort, 0, len(seen))
	for ep := range seen {
		out = append(out, ep)
	}
	// Sort for deterministic YAML output.
	sort.Slice(out, func(i, j int) bool {
		if out[i].port != out[j].port {
			return out[i].port < out[j].port
		}
		return out[i].proto < out[j].proto
	})
	return out
}
