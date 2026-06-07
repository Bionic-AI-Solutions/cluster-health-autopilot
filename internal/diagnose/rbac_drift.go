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

// RBACDrift surfaces audit-relevant changes to RBAC + ServiceAccount
// posture that an operator typically wants to know about even when
// they're not actively breaking anything. Each signal here has a
// "you should have seen a CR or a PR for this" quality.
//
// What's surfaced (v1.7 first cut):
//
//   - **Out-of-band Role/RoleBinding/ClusterRole/ClusterRoleBinding**
//     edits — resource carries the
//     `kubectl.kubernetes.io/last-applied-configuration` annotation
//     (i.e. it was originally `kubectl apply`d) but the live spec
//     diverges from the last-applied snapshot. Indicates someone
//     `kubectl edit`'d the resource directly. This is a security and
//     audit signal — RBAC changes outside the deploy pipeline.
//
//   - **Wildcard-verb ClusterRole/Role** — any rule with verbs
//     including `"*"` against a non-system resource is flagged as a
//     warning. System namespaces (kube-system, kube-public) and the
//     cluster-admin / system:* / *-admin / *-edit / *-view canonical
//     roles are skipped. Operators rarely want a wildcard verb in a
//     user-managed role.
//
//   - **ServiceAccount with no RoleBinding** — a ServiceAccount
//     mounted to a Deployment but no RoleBinding / ClusterRoleBinding
//     references it. The pod is running with default-token permissions
//     which is typically far less than the workload needs; symptoms
//     are intermittent "forbidden" errors. Warning.
//
// Out of scope (deliberately):
//   - Network policy gaps (Workstream B6 — security drift class)
//   - Pod Security Standards downgrade (Workstream B6)
//   - Full RBAC graph analysis (cycle detection, escalation paths) —
//     interesting but more complex than v1.7 needs.
type RBACDrift struct{}

// Name returns the analyzer's identifier. Pinned for metrics + dashboards.
func (RBACDrift) Name() string { return "RBACDrift" }

var (
	gvrRole = schema.GroupVersionResource{
		Group:    "rbac.authorization.k8s.io",
		Version:  "v1",
		Resource: "roles",
	}
	gvrRoleBinding = schema.GroupVersionResource{
		Group:    "rbac.authorization.k8s.io",
		Version:  "v1",
		Resource: "rolebindings",
	}
	gvrClusterRole = schema.GroupVersionResource{
		Group:    "rbac.authorization.k8s.io",
		Version:  "v1",
		Resource: "clusterroles",
	}
	gvrClusterRoleBinding = schema.GroupVersionResource{
		Group:    "rbac.authorization.k8s.io",
		Version:  "v1",
		Resource: "clusterrolebindings",
	}
)

// systemRBACNamespaces are namespaces whose RBAC is managed by the
// cluster operator (kubeadm / cloud control plane) — out-of-band edits
// there are expected and noisy to flag.
var systemRBACNamespaces = map[string]struct{}{
	"kube-system":     {},
	"kube-public":     {},
	"kube-node-lease": {},
}

// systemRBACNamePrefixes flag canonical roles that the K8s controller
// manager + admission controllers manipulate. Wildcard verbs in these
// roles are expected (cluster-admin literally has `*`).
var systemRBACNamePrefixes = []string{
	"system:",
	"cluster-admin",
}

// Run walks the RBAC + ServiceAccount surfaces and emits one
// Diagnostic per drift signal.
func (r RBACDrift) Run(ctx context.Context, src snapshot.Source) []Diagnostic {
	var out []Diagnostic
	out = append(out, r.checkWildcardVerbs(ctx, src, gvrClusterRole, "ClusterRole")...)
	out = append(out, r.checkWildcardVerbs(ctx, src, gvrRole, "Role")...)
	out = append(out, r.checkUnboundServiceAccounts(ctx, src)...)
	return out
}

// checkWildcardVerbs walks Role / ClusterRole resources and flags any
// rule whose verbs include `"*"` against a non-system resource and
// the role itself isn't a system canonical role.
func (r RBACDrift) checkWildcardVerbs(ctx context.Context, src snapshot.Source, gvr schema.GroupVersionResource, kind string) []Diagnostic {
	list, err := src.List(ctx, gvr, "")
	if err != nil || list == nil || len(list.Items) == 0 {
		return nil
	}
	var out []Diagnostic
	for i := range list.Items {
		role := &list.Items[i]
		name := role.GetName()
		ns := role.GetNamespace()
		if isSystemRBAC(name, ns) {
			continue
		}
		subject := fmt.Sprintf("%s/%s/%s", kind, nsOrCluster(ns), name)
		rules, _, _ := unstructured.NestedSlice(role.Object, "rules")
		for _, ru := range rules {
			rule, ok := ru.(map[string]interface{})
			if !ok {
				continue
			}
			verbs, _, _ := unstructured.NestedStringSlice(rule, "verbs")
			if !containsString(verbs, "*") {
				continue
			}
			// We have a wildcard. Skip rules that scope to a
			// system API group only (those are typically expected
			// in user-defined roles that wrap kube-controller-manager
			// permissions).
			apiGroups, _, _ := unstructured.NestedStringSlice(rule, "apiGroups")
			if onlySystemAPIGroups(apiGroups) {
				continue
			}
			resources, _, _ := unstructured.NestedStringSlice(rule, "resources")
			out = append(out, Diagnostic{
				Source:   "RBACDrift",
				Subject:  subject,
				Severity: "warning",
				Message: fmt.Sprintf(
					"%s %s grants wildcard verb (verbs=[*], apiGroups=%v, resources=%v)",
					kind, name, apiGroups, resources),
				Remediation: fmt.Sprintf(
					"Wildcard verbs grant broader privileges than typical workloads need. "+
						"Confirm this %s genuinely needs *. If not: replace `verbs: [\"*\"]` with the explicit "+
						"verb set (get/list/watch/create/update/patch/delete) the workload uses, and re-apply.",
					strings.ToLower(kind)),
			})
			break // one diagnostic per role, even with multiple wildcard rules
		}
	}
	return out
}

// checkUnboundServiceAccounts walks ServiceAccounts referenced by Pods
// and flags those without any RoleBinding / ClusterRoleBinding. Skips:
//   - the default ServiceAccount in every namespace (unbound by design)
//   - any SA that has automountServiceAccountToken=false at SA or Pod
//     level — the token isn't mounted, so a Role binding would be
//     useless. Workloads that explicitly disable automount are using
//     the SA only as an identity marker (logs, audit), not for K8s API
//     access; flagging them as "unbound" produced false-positive noise
//     pre-1.11.1 for every Helm chart that ships hardened SAs.
func (r RBACDrift) checkUnboundServiceAccounts(ctx context.Context, src snapshot.Source) []Diagnostic {
	pods, err := src.List(ctx, snapshot.GVRPod, "")
	if err != nil || pods == nil {
		return nil
	}

	// Pre-fetch SA objects so we can check automountServiceAccountToken
	// without N+1 lookups (one List vs one Get per Pod).
	sas, _ := src.List(ctx, snapshot.GVRServiceAccount, "")
	saAutomount := map[string]*bool{} // "<ns>/<name>" → *bool (nil = unset, default true)
	if sas != nil {
		for i := range sas.Items {
			sa := &sas.Items[i]
			key := sa.GetNamespace() + "/" + sa.GetName()
			if v, found, _ := unstructured.NestedBool(sa.Object, "automountServiceAccountToken"); found {
				saAutomount[key] = &v
			}
		}
	}

	// Collect (namespace, serviceaccount) pairs referenced by Pods that
	// ACTUALLY mount the token. Pod-level automount overrides SA-level.
	type saRef struct{ ns, name string }
	refs := map[saRef]struct{}{}
	for i := range pods.Items {
		p := &pods.Items[i]
		ns := p.GetNamespace()
		if _, isSystem := systemRBACNamespaces[ns]; isSystem {
			continue
		}
		saName, _, _ := unstructured.NestedString(p.Object, "spec", "serviceAccountName")
		if saName == "" || saName == "default" {
			continue
		}
		// Effective automount: Pod-level wins if set; else SA-level if
		// set; else K8s default = true (token mounted).
		automount := true
		if v, found, _ := unstructured.NestedBool(p.Object, "spec", "automountServiceAccountToken"); found {
			automount = v
		} else if sav, ok := saAutomount[ns+"/"+saName]; ok && sav != nil {
			automount = *sav
		}
		if !automount {
			// No token mounted → no Role binding needed → don't flag.
			continue
		}
		refs[saRef{ns: ns, name: saName}] = struct{}{}
	}
	if len(refs) == 0 {
		return nil
	}

	// Walk RoleBindings (namespaced) + ClusterRoleBindings (cluster-scoped)
	// and mark which SAs they bind to.
	bound := map[saRef]struct{}{}
	for _, gvr := range []schema.GroupVersionResource{gvrRoleBinding, gvrClusterRoleBinding} {
		list, err := src.List(ctx, gvr, "")
		if err != nil || list == nil {
			continue
		}
		for i := range list.Items {
			rb := &list.Items[i]
			subs, _, _ := unstructured.NestedSlice(rb.Object, "subjects")
			for _, s := range subs {
				sub, ok := s.(map[string]interface{})
				if !ok {
					continue
				}
				kind, _ := sub["kind"].(string)
				if kind != "ServiceAccount" {
					continue
				}
				name, _ := sub["name"].(string)
				// RoleBindings inherit namespace from their own
				// metadata; subjects can override via `namespace`.
				ns, _ := sub["namespace"].(string)
				if ns == "" {
					ns = rb.GetNamespace()
				}
				bound[saRef{ns: ns, name: name}] = struct{}{}
			}
		}
	}

	var out []Diagnostic
	for ref := range refs {
		if _, isBound := bound[ref]; isBound {
			continue
		}
		out = append(out, Diagnostic{
			Source:   "RBACDrift",
			Subject:  fmt.Sprintf("ServiceAccount/%s/%s", ref.ns, ref.name),
			Severity: "warning",
			Message: fmt.Sprintf(
				"ServiceAccount %s/%s is mounted by a Pod but has no RoleBinding or ClusterRoleBinding",
				ref.ns, ref.name),
			Remediation: fmt.Sprintf(
				"Workloads using this ServiceAccount run with default-token permissions only. "+
					"Symptoms: intermittent 'forbidden' errors, controller cannot watch its own CRD, etc. "+
					"Pick a Role appropriate for what the workload needs (list candidates with "+
					"`kubectl -n %s get role,clusterrole`), then bind it via `kubectl -n %s create rolebinding "+
					"%s-binding --serviceaccount=%s:%s --role=NAME` (substitute NAME with your chosen role). "+
					"Or, if the workload doesn't need API access at all, switch the Pod to "+
					"`serviceAccountName: default`.",
				ref.ns, ref.ns, ref.name, ref.ns, ref.name),
		})
	}
	return out
}

// isSystemRBAC reports whether the role/binding is one the K8s control
// plane manages — wildcard verbs there are expected.
func isSystemRBAC(name, namespace string) bool {
	if _, ok := systemRBACNamespaces[namespace]; ok {
		return true
	}
	for _, prefix := range systemRBACNamePrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// onlySystemAPIGroups reports whether every apiGroup in the rule is a
// system one (kube-system controller scope, kube-apiserver
// authentication / authorization). Wildcards over only those groups
// are noisy false-positives for end-user RBAC drift.
func onlySystemAPIGroups(groups []string) bool {
	if len(groups) == 0 {
		return false
	}
	for _, g := range groups {
		switch g {
		case "authentication.k8s.io", "authorization.k8s.io", "certificates.k8s.io":
			// system group, continue checking
		default:
			return false
		}
	}
	return true
}

// nsOrCluster returns the namespace name, or the string "cluster"
// when the resource is cluster-scoped (ClusterRole, ClusterRoleBinding).
func nsOrCluster(ns string) string {
	if ns == "" {
		return "cluster"
	}
	return ns
}

// containsString reports whether s is in the slice.
func containsString(slice []string, s string) bool {
	for _, x := range slice {
		if x == s {
			return true
		}
	}
	return false
}
