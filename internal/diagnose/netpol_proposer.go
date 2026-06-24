// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package diagnose

import (
	"context"
	"fmt"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/netpol"
)

// DefaultNetpolMaxPerCycle is the default cap on the number of
// NetworkPolicy proposals emitted per watcher cycle. Without a cap,
// a cluster with many uncovered namespaces generates one Slack
// approval request per namespace — all in the same message — which
// is unusable. Remaining namespaces are summarised in a single
// info-level "N more namespaces need NetworkPolicies" diagnostic so
// operators know work remains without being flooded.
const DefaultNetpolMaxPerCycle = 5

// NetworkPolicyProposer is the OSS-side hook of Phase 2d-β. On clusters
// where the CNI actually enforces NetworkPolicy (Calico / Cilium / AWS
// VPC CNI / etc. — see netpol.DetectCNI), it walks every user namespace
// without a NetworkPolicy and emits a Diagnostic carrying a ready-to-
// apply NetworkPolicy YAML.
//
// On clusters whose CNI does NOT enforce NetworkPolicy (k3s-Flannel-
// only is the canonical case), this analyzer is silent. The companion
// SecurityDrift.checkNetworkPolicyCoverage emits a single info-level
// cluster-scope finding explaining why; the proposer doesn't generate
// noise where applies would be decorative-only.
//
// The proposer NEVER applies the policy itself. It emits the YAML in
// ProposedPolicyYAML; cha-com aiwatch wraps that into an
// ApprovalProposal CR and renders Approve/Deny in Slack (v1.10.4
// pattern). The approval-server's /approve endpoint reads the YAML
// off the CR and applies it. The OSS install never sees Approve/Deny
// buttons — that's the paid AI tier.
type NetworkPolicyProposer struct {
	// Proposer is the implementation that generates the YAML. Defaults
	// to netpol.SnapshotProposer{} when nil. Tests inject fakes.
	Proposer netpol.Proposer

	// MaxPerCycle caps the number of approval proposals emitted per
	// watcher cycle to prevent Slack flooding on clusters with many
	// uncovered namespaces. 0 = use DefaultNetpolMaxPerCycle.
	MaxPerCycle int
}

// Name returns the analyzer identifier.
func (NetworkPolicyProposer) Name() string { return "NetworkPolicyProposer" }

// Run walks user namespaces and emits a Diagnostic with a proposed
// NetworkPolicy YAML for each namespace that has pods + zero NetPols
// on a CNI-enforcing cluster.
//
// Skips entirely when CNI doesn't enforce.
func (a NetworkPolicyProposer) Run(ctx context.Context, src snapshot.Source) []Diagnostic {
	cni := netpol.DetectCNI(ctx, src)
	if !cni.Enforces {
		return nil
	}

	proposer := a.Proposer
	if proposer == nil {
		proposer = netpol.SnapshotProposer{}
	}

	nsList, err := src.List(ctx, gvrNamespace, "")
	if err != nil || nsList == nil {
		logListFailure("namespaces", err, true) // silent when the CRD/resource is absent; logs Forbidden etc.
		return nil
	}

	cap := a.MaxPerCycle
	if cap <= 0 {
		cap = DefaultNetpolMaxPerCycle
	}

	var out []Diagnostic
	var skipped []string
	for i := range nsList.Items {
		ns := &nsList.Items[i]
		name := ns.GetName()
		if _, isSystem := systemSecurityNamespaces[name]; isSystem {
			continue
		}

		proposal, err := proposer.ProposeForNamespace(ctx, src, name)
		if err != nil || proposal == nil {
			continue
		}

		if len(out) >= cap {
			skipped = append(skipped, name)
			continue
		}

		out = append(out, Diagnostic{
			Source:   "NetworkPolicyProposer",
			Subject:  fmt.Sprintf("Namespace/cluster/%s/missing-network-policy", name),
			Severity: "warning",
			Message: fmt.Sprintf(
				"Namespace %s runs pods on a %s cluster (NetPol enforced) but has zero NetworkPolicies. "+
					"Proposed: %s",
				name, cni.CNIName, proposal.Rationale),
			Remediation: fmt.Sprintf(
				"Review the proposed NetworkPolicy below. Approve to apply, Deny to dismiss. "+
					"The proposer is deterministic: default-deny ingress + allow-from same namespace + "+
					"allow-from detected Ingress controllers. Egress is left unrestricted (DNS, "+
					"in-cluster service calls, external APIs all keep working).\n\n%s",
				proposal.PolicyYAML),
			ProposedPolicyYAML: proposal.PolicyYAML,
			ProposedPolicyKind: proposal.PolicyKind,
		})
	}

	// Emit a single summary diagnostic for capped namespaces so operators
	// know work remains without being flooded with approval requests.
	if len(skipped) > 0 {
		out = append(out, Diagnostic{
			Source:   "NetworkPolicyProposer",
			Subject:  "Namespace/cluster/batched/missing-network-policy",
			Severity: "info",
			Message: fmt.Sprintf(
				"%d more namespace(s) on this %s cluster have no NetworkPolicy (batched to avoid Slack flood): %v. "+
					"Approve the current %d proposals; remaining namespaces will be proposed in subsequent cycles.",
				len(skipped), cni.CNIName, skipped, cap),
		})
	}

	return out
}
