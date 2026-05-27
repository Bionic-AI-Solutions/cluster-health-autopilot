// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package diagnose

import (
	"context"
	"fmt"
	"strings"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// SecurityDrift surfaces audit-relevant gaps in cluster security
// posture. Each signal is observational — CHA does not enforce; it
// surfaces the gap so an operator can react.
//
// What's surfaced (v1.8 first cut):
//
//   - **Pod Security Standards posture gap** — user namespaces with
//     no `pod-security.kubernetes.io/enforce` label, or with
//     `enforce=privileged`. PSS is the modern replacement for Pod
//     Security Policies; absence of the label means K8s applies the
//     cluster-wide default (typically `privileged`). Warning when
//     missing, warning when explicitly `privileged`.
//
//   - **Mutable image tag (no digest pin)** — Pods whose containers
//     reference images by tag only (`<image>:v1.2.3`) rather than
//     digest (`<image>@sha256:...`). Mutable tags break the
//     attestation story image-signing relies on: even if you sign
//     `<image>:v1.2.3` today, the digest the tag points at can be
//     re-published tomorrow. The pod that ran "the signed image" is
//     not necessarily the pod running the same tag tomorrow.
//     Warning. Skipped for `latest` (the framework's other
//     pull-policy code paths already flag that).
//
//   - **NetworkPolicy coverage gap** — user namespaces running pods
//     with zero NetworkPolicies. K8s default networking is fully
//     permissive: without at least one NetworkPolicy, any pod can
//     reach any other pod cluster-wide. Warning per namespace.
//
// Out of scope (deliberately, for v1.8.x):
//
//   - True PSS-downgrade detection (was-restricted-is-now-baseline)
//     — requires label-history, not derivable from a snapshot.
//
//   - Active Cosign / Notation signature verification — admission-
//     time concern that needs the cluster's policy controller to
//     emit events; observational CHA is the wrong tool.
//
//   - NetworkPolicy egress-coverage analysis (a namespace has
//     NetworkPolicies but none set `policyTypes: [Egress]`) —
//     interesting but more nuanced than v1.8 needs; defer until
//     operator demand surfaces.
//
// Skip rules: kube-system / kube-public / kube-node-lease /
// cnpg-system / rook-ceph / vault / external-secrets — system
// namespaces whose posture is managed by their controllers rather
// than the cluster operator.
type SecurityDrift struct{}

// Name returns the analyzer's identifier. Pinned for metrics +
// dashboards.
func (SecurityDrift) Name() string { return "SecurityDrift" }

var (
	gvrNamespace = schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "namespaces",
	}
	gvrNetworkPolicy = schema.GroupVersionResource{
		Group:    "networking.k8s.io",
		Version:  "v1",
		Resource: "networkpolicies",
	}
)

// systemSecurityNamespaces are namespaces whose security posture is
// managed by the cluster operator at install time (or by the
// platform's own controllers); flagging them is noise.
var systemSecurityNamespaces = map[string]struct{}{
	"kube-system":      {},
	"kube-public":      {},
	"kube-node-lease":  {},
	"cnpg-system":      {},
	"rook-ceph":        {},
	"vault":            {},
	"external-secrets": {},
}

// pssEnforceLabel is the standard label apiserver-side PSS admission
// reads. See:
//
//	https://kubernetes.io/docs/concepts/security/pod-security-admission/
const pssEnforceLabel = "pod-security.kubernetes.io/enforce"

// Run walks namespaces + pods + networkpolicies and emits one
// Diagnostic per drift signal.
func (s SecurityDrift) Run(ctx context.Context, src snapshot.Source) []Diagnostic {
	var out []Diagnostic
	out = append(out, s.checkPSSPosture(ctx, src)...)
	out = append(out, s.checkMutableImageTags(ctx, src)...)
	out = append(out, s.checkNetworkPolicyCoverage(ctx, src)...)
	return out
}

// checkPSSPosture flags user namespaces with no PSS enforce label or
// with `enforce=privileged`.
func (s SecurityDrift) checkPSSPosture(ctx context.Context, src snapshot.Source) []Diagnostic {
	nsList, err := src.List(ctx, gvrNamespace, "")
	if err != nil || nsList == nil {
		return nil
	}
	var out []Diagnostic
	for i := range nsList.Items {
		ns := &nsList.Items[i]
		name := ns.GetName()
		if _, isSystem := systemSecurityNamespaces[name]; isSystem {
			continue
		}
		labels := ns.GetLabels()
		enforce, hasLabel := labels[pssEnforceLabel]
		switch {
		case !hasLabel:
			out = append(out, Diagnostic{
				Source:   "SecurityDrift",
				Subject:  fmt.Sprintf("Namespace/cluster/%s", name),
				Severity: "warning",
				Message: fmt.Sprintf(
					"Namespace %s has no pod-security.kubernetes.io/enforce label; "+
						"admission applies the cluster-wide default (typically privileged)",
					name),
				Remediation: fmt.Sprintf(
					"Label the namespace at the tightest profile its workloads tolerate. "+
						"Start with: `kubectl label namespace %s pod-security.kubernetes.io/enforce=baseline pod-security.kubernetes.io/warn=baseline`. "+
						"Tighten to `restricted` once you've confirmed pods aren't using HostPath / HostNetwork / privileged containers.",
					name),
			})
		case enforce == "privileged":
			out = append(out, Diagnostic{
				Source:   "SecurityDrift",
				Subject:  fmt.Sprintf("Namespace/cluster/%s", name),
				Severity: "warning",
				Message: fmt.Sprintf(
					"Namespace %s explicitly enforces PSS=privileged — the most-permissive profile",
					name),
				Remediation: fmt.Sprintf(
					"Confirm this namespace genuinely needs privileged escalation (host networking, "+
						"hostPath mounts, capabilities like SYS_ADMIN). If not, tighten with "+
						"`kubectl label namespace %s pod-security.kubernetes.io/enforce=baseline --overwrite`. "+
						"If yes, document the reason as a namespace annotation so audit reviewers don't re-flag it.",
					name),
			})
		}
	}
	return out
}

// checkMutableImageTags flags Pods whose containers use a tag-only
// reference (no @sha256:... digest pin). One diagnostic per Pod,
// listing the affected container.
func (s SecurityDrift) checkMutableImageTags(ctx context.Context, src snapshot.Source) []Diagnostic {
	pods, err := src.List(ctx, snapshot.GVRPod, "")
	if err != nil || pods == nil {
		return nil
	}
	var out []Diagnostic
	for i := range pods.Items {
		p := &pods.Items[i]
		ns := p.GetNamespace()
		if _, isSystem := systemSecurityNamespaces[ns]; isSystem {
			continue
		}
		name := p.GetName()

		containers, _, _ := unstructured.NestedSlice(p.Object, "spec", "containers")
		var unpinned []string
		for _, c := range containers {
			ci, ok := c.(map[string]interface{})
			if !ok {
				continue
			}
			cn, _ := ci["name"].(string)
			img, _ := ci["image"].(string)
			if img == "" || strings.Contains(img, "@sha256:") {
				continue
			}
			// `latest` and `:` with no tag are flagged by other
			// code paths; skip to avoid double-flagging.
			if strings.HasSuffix(img, ":latest") || !strings.Contains(img, ":") {
				continue
			}
			unpinned = append(unpinned, fmt.Sprintf("%s=%s", cn, img))
		}
		if len(unpinned) == 0 {
			continue
		}
		out = append(out, Diagnostic{
			Source:   "SecurityDrift",
			Subject:  fmt.Sprintf("Pod/%s/%s", ns, name),
			Severity: "warning",
			Message: fmt.Sprintf(
				"Pod %s/%s mounts %d container image(s) without digest pin: %s",
				ns, name, len(unpinned), strings.Join(unpinned, ", ")),
			Remediation: "Replace mutable :tag references with @sha256:<digest> in the workload's manifest. " +
				"Get the digest from `crane digest <image>:<tag>` or `docker pull <image>:<tag>` (the digest line). " +
				"This pins the runtime image immutably and preserves the image-attestation signature chain — " +
				"the workload can be re-verified against the original sigstore signature at any point.",
		})
	}
	return out
}

// checkNetworkPolicyCoverage flags user namespaces that run pods but
// have zero NetworkPolicies. Default K8s networking is fully open
// without at least one policy.
func (s SecurityDrift) checkNetworkPolicyCoverage(ctx context.Context, src snapshot.Source) []Diagnostic {
	nsList, err := src.List(ctx, gvrNamespace, "")
	if err != nil || nsList == nil {
		return nil
	}

	// Index NetworkPolicies by namespace once. List is cheap; the
	// per-namespace bool lookup avoids a second pass.
	netpolNamespaces := map[string]struct{}{}
	npList, _ := src.List(ctx, gvrNetworkPolicy, "")
	if npList != nil {
		for i := range npList.Items {
			netpolNamespaces[npList.Items[i].GetNamespace()] = struct{}{}
		}
	}

	// Index pod presence by namespace — namespaces with zero pods
	// don't pose a coverage gap (no traffic to govern). Saves
	// flagging brand-new empty namespaces.
	podNamespaces := map[string]struct{}{}
	pods, _ := src.List(ctx, snapshot.GVRPod, "")
	if pods != nil {
		for i := range pods.Items {
			podNamespaces[pods.Items[i].GetNamespace()] = struct{}{}
		}
	}

	var out []Diagnostic
	for i := range nsList.Items {
		ns := &nsList.Items[i]
		name := ns.GetName()
		if _, isSystem := systemSecurityNamespaces[name]; isSystem {
			continue
		}
		if _, hasPods := podNamespaces[name]; !hasPods {
			continue
		}
		if _, hasPolicies := netpolNamespaces[name]; hasPolicies {
			continue
		}
		out = append(out, Diagnostic{
			Source:   "SecurityDrift",
			Subject:  fmt.Sprintf("Namespace/cluster/%s", name),
			Severity: "warning",
			Message: fmt.Sprintf(
				"Namespace %s runs pods but has zero NetworkPolicies; any pod can reach any other pod cluster-wide",
				name),
			Remediation: fmt.Sprintf(
				"Add at least a default-deny NetworkPolicy: "+
					"`kubectl apply -n %s -f - <<EOF\n"+
					"apiVersion: networking.k8s.io/v1\n"+
					"kind: NetworkPolicy\n"+
					"metadata:\n  name: default-deny-all\n"+
					"spec:\n  podSelector: {}\n  policyTypes: [Ingress, Egress]\n"+
					"EOF`\n"+
					"Then add explicit allow-rules for the inter-pod and egress paths the workloads need. "+
					"Helm-managed apps typically ship NetworkPolicies in-chart; missing here usually means the "+
					"`networkPolicy.enabled=true` value wasn't set at install.",
				name),
		})
	}
	return out
}
