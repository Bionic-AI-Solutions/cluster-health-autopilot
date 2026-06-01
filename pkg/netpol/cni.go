// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

// Package netpol contains the OSS-side NetworkPolicy proposer plumbing
// for Phase 2d-β. Two layers:
//
//   - CNI enforcement detection (this file): which CNI the cluster runs
//     and whether NetworkPolicies are actually enforced at runtime.
//     Flannel-only clusters STORE NetworkPolicies but don't enforce
//     them; flagging or proposing for those is misleading.
//
//   - Snapshot-based proposer (proposer.go): given a namespace with
//     pods + zero NetPols, generate a safe default-deny stub with
//     allow rules derived from observed Service / Ingress shape.
//
// The proposer never APPLIES anything. It emits a Proposal that
// cha-com aiwatch wraps into an ApprovalProposal CR and surfaces in
// Slack with the v1.10.4 Approve/Deny pair. Click → apply path is
// owned by the approval-server (cha-com).
package netpol

import (
	"context"
	"strings"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
)

// CNIDetection summarizes which CNI the cluster runs and whether
// NetworkPolicy is actually enforced. The proposer skips clusters where
// Enforces=false because applying NetPols there is decorative-only.
type CNIDetection struct {
	// Enforces is true when the detected CNI runtime actively blocks
	// pod-to-pod traffic per NetworkPolicy rules. Flannel-only is the
	// canonical false case.
	Enforces bool

	// CNIName is a short identifier: "calico" | "cilium" | "aws-vpc-cni"
	// | "gke-dataplane-v2" | "azure-cni" | "flannel-only" | "unknown".
	CNIName string

	// Evidence is a human-readable explanation of what was matched
	// (DaemonSet name, CRD presence, etc.). Surfaced in the analyzer
	// finding so operators can verify the detection without rerunning.
	Evidence string
}

// DetectCNI walks the cluster snapshot for known CNI signatures and
// returns the best-effort verdict. Always succeeds — when no
// recognizable CNI is found, returns Enforces=false with CNIName="unknown".
//
// Detection order (first match wins, strongest signal first):
//
//  1. Cilium DaemonSet anywhere (Cilium enforces by default)
//  2. Calico DaemonSet OR calico-system namespace (Calico enforces)
//  3. AWS VPC CNI DaemonSet `aws-node` (enforcement requires
//     ENABLE_NETWORK_POLICY=true env; assumed for now — refine later)
//  4. GKE Dataplane V2: `anetd` DaemonSet (Cilium-based; enforces)
//  5. Azure CNI with policy plugin: `azure-npm` DaemonSet
//  6. Flannel + kube-router as NetPol enforcer: a real production
//     pattern (k3s clusters often layer kube-router on Flannel for
//     enforcement). Add-on enforcement matters even when the
//     base CNI doesn't enforce by itself. v1.12.3 added this after
//     2026-06-01 outage caused by missed kube-router detection.
//  7. Flannel DaemonSet present and none of the above → Flannel-only
//  8. Otherwise → unknown
func DetectCNI(ctx context.Context, src snapshot.Source) CNIDetection {
	dsList, err := src.List(ctx, snapshot.GVRDaemonSet, "")
	if err != nil || dsList == nil {
		return CNIDetection{CNIName: "unknown", Evidence: "DaemonSet list failed or empty"}
	}

	hasFlannel := false
	var flannelEvidence string
	hasKubeRouter := false
	var kubeRouterEvidence string

	for i := range dsList.Items {
		ds := &dsList.Items[i]
		ns := ds.GetNamespace()
		name := ds.GetName()
		lname := strings.ToLower(name)

		switch {
		case strings.Contains(lname, "cilium") && !strings.Contains(lname, "operator"):
			return CNIDetection{
				Enforces: true, CNIName: "cilium",
				Evidence: "DaemonSet " + ns + "/" + name,
			}
		case lname == "anetd" || strings.Contains(lname, "anetd"):
			// GKE Dataplane V2 (Cilium-based).
			return CNIDetection{
				Enforces: true, CNIName: "gke-dataplane-v2",
				Evidence: "DaemonSet " + ns + "/" + name + " (GKE Cilium-based)",
			}
		case strings.HasPrefix(lname, "calico-") || ns == "calico-system" || ns == "tigera-operator":
			return CNIDetection{
				Enforces: true, CNIName: "calico",
				Evidence: "DaemonSet " + ns + "/" + name,
			}
		case lname == "aws-node":
			return CNIDetection{
				Enforces: true, CNIName: "aws-vpc-cni",
				Evidence: "DaemonSet " + ns + "/" + name +
					" (NetPol enforcement requires ENABLE_NETWORK_POLICY=true on the addon)",
			}
		case strings.Contains(lname, "azure-npm"):
			return CNIDetection{
				Enforces: true, CNIName: "azure-cni",
				Evidence: "DaemonSet " + ns + "/" + name,
			}
		case strings.Contains(lname, "kube-router"):
			// kube-router is commonly LAYERED on Flannel as a
			// NetPol enforcement add-on (k3s + kube-router pattern).
			// Even when present alongside Flannel, NetPol IS
			// enforced. Don't return immediately — record and
			// continue to see what base CNI is present too.
			hasKubeRouter = true
			kubeRouterEvidence = "DaemonSet " + ns + "/" + name
		case strings.Contains(lname, "flannel"):
			hasFlannel = true
			flannelEvidence = "DaemonSet " + ns + "/" + name
		}
	}

	// kube-router enforces NetPol regardless of base CNI.
	if hasKubeRouter {
		base := "(standalone)"
		if hasFlannel {
			base = "+ Flannel (" + flannelEvidence + ")"
		}
		return CNIDetection{
			Enforces: true, CNIName: "kube-router",
			Evidence: kubeRouterEvidence + " " + base +
				". kube-router enforces NetworkPolicy via its KUBE-ROUTER-FORWARD chain.",
		}
	}

	if hasFlannel {
		return CNIDetection{
			Enforces: false, CNIName: "flannel-only",
			Evidence: flannelEvidence + ". Flannel-only (no Calico/Cilium/AWS-VPC-CNI/Azure-NPM/" +
				"kube-router signal). Flannel does not enforce NetworkPolicy.",
		}
	}
	return CNIDetection{
		CNIName:  "unknown",
		Evidence: "No recognized CNI DaemonSet pattern matched",
	}
}
