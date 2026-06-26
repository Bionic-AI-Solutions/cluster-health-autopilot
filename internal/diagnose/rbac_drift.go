// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package diagnose

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/srenix-ai/agentic-sre/internal/snapshot"
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
//     warning. The following are suppressed (not flagged) because they
//     legitimately hold wildcard verbs:
//
//   - System namespaces: kube-system, kube-public, kube-node-lease,
//     and well-known operator namespaces (calico-system,
//     tigera-operator, minio-operator, kasten-io, olm, operators,
//     rook-ceph, cert-manager, longhorn-system, cnpg-system,
//     external-secrets, vault, local-path-storage, cattle-system,
//     openshift-operators).
//
//   - Name-prefixed system / operator roles: system:, cluster-admin,
//     k10-, kasten-, calico-, tigera-, minio-, olm., olm-, k3s-,
//     local-path-, console-, rook-, cert-manager, velero, longhorn-,
//     cnpg-, external-secrets, vault-, openshift-.
//
//   - Exact canonical names: admin, edit, view, cluster-owner,
//     local-clusterowner (exact match prevents over-suppression of
//     user roles like custom-admin or payments-admin).
//     Operators rarely want a wildcard verb in a user-managed role.
//
//   - **ConfigMap-based allowlist extension** — the built-in allowlist
//     can be extended at runtime by a ConfigMap named `srenix-rbac-allowlist`
//     in the install namespace (POD_NAMESPACE or "agentic-sre").
//     Keys: allowNamespaces, allowRolePrefixes, allowRoleNames (comma/newline
//     separated). See Feature 1 comment in Run() for full details.
//
//   - **ServiceAccount with no RoleBinding** — a ServiceAccount
//     mounted to a Deployment but no RoleBinding / ClusterRoleBinding
//     references it. The pod is running with default-token permissions
//     which is typically far less than the workload needs; symptoms
//     are intermittent "forbidden" errors. Warning. SAs in well-known
//     operator namespaces (same list above) are suppressed.
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
// cluster operator (kubeadm / cloud control plane) or a well-known
// third-party operator — out-of-band edits and unbound SAs there are
// expected and noisy to flag.
var systemRBACNamespaces = map[string]struct{}{
	// Kubernetes core
	"kube-system":     {},
	"kube-public":     {},
	"kube-node-lease": {},
	// Well-known third-party operators whose SAs legitimately exist
	// without user-managed RoleBindings.
	"calico-system":       {},
	"tigera-operator":     {},
	"minio-operator":      {},
	"kasten-io":           {},
	"olm":                 {},
	"operators":           {},
	"rook-ceph":           {},
	"cert-manager":        {},
	"longhorn-system":     {},
	"cnpg-system":         {},
	"external-secrets":    {},
	"vault":               {},
	"local-path-storage":  {},
	"cattle-system":       {},
	"openshift-operators": {},
}

// systemRBACNamePrefixes lists name prefixes for ClusterRoles/Roles
// that the K8s controller manager, admission controllers, or well-known
// third-party operators manage. Wildcard verbs in these roles are
// expected and non-actionable.
var systemRBACNamePrefixes = []string{
	// Kubernetes built-ins
	"system:",
	"cluster-admin",
	// Third-party operators
	"k10-",
	"kasten-",
	"calico-",
	"tigera-",
	"minio-",
	"olm.",
	"olm-",
	"k3s-",
	"local-path-",
	"console-",
	"rook-",
	"cert-manager",
	"velero",
	"longhorn-",
	"cnpg-",
	"external-secrets",
	"vault-",
	"openshift-",
}

// systemRBACExactNames lists exact ClusterRole/Role names that are
// canonical K8s aggregated roles or well-known operator roles.
// Exact-name matching is used (not prefix) so that user roles like
// "custom-admin" or "payments-admin" are still flagged. Prefixing
// "admin" would over-suppress those legitimate user-role signals.
var systemRBACExactNames = map[string]struct{}{
	"admin":              {},
	"edit":               {},
	"view":               {},
	"cluster-owner":      {},
	"local-clusterowner": {},
}

// rbacAllowlist holds the merged built-in + ConfigMap allowlist for a
// single Run() invocation. It is NOT stored on the struct to keep
// RBACDrift constructable as RBACDrift{} with zero config — the list
// is rebuilt from the ConfigMap on every Run call.
type rbacAllowlist struct {
	// extra namespaces to suppress (merged with systemRBACNamespaces)
	extraNamespaces map[string]struct{}
	// extra name prefixes to suppress (merged with systemRBACNamePrefixes)
	extraPrefixes []string
	// extra exact names to suppress (merged with systemRBACExactNames)
	extraExactNames map[string]struct{}
}

// isSystemRBACWithExtra extends the built-in isSystemRBAC check with
// the per-run ConfigMap-sourced allowlist. The caller passes the merged
// allowlist returned by loadRBACAllowlist; built-in vars stay immutable.
func isSystemRBACWithExtra(name, namespace string, extra rbacAllowlist) bool {
	// built-in check first
	if isSystemRBAC(name, namespace) {
		return true
	}
	// ConfigMap-extended namespace check
	if _, ok := extra.extraNamespaces[namespace]; ok {
		return true
	}
	// ConfigMap-extended prefix check
	for _, prefix := range extra.extraPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	// ConfigMap-extended exact-name check
	if _, ok := extra.extraExactNames[name]; ok {
		return true
	}
	return false
}

// installNamespace returns the namespace Srenix is running in.
// Reads POD_NAMESPACE env var; falls back to "agentic-sre".
func installNamespace() string {
	if ns := os.Getenv("POD_NAMESPACE"); ns != "" {
		return ns
	}
	return "agentic-sre"
}

// gvrConfigMap is the GVR for core/v1 ConfigMap objects, used to load
// the srenix-rbac-allowlist ConfigMap at Run time.
var gvrConfigMap = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}

// loadRBACAllowlist attempts a best-effort read of the `srenix-rbac-allowlist`
// ConfigMap from the install namespace. Absent/unreadable/empty → returns
// a zero-value rbacAllowlist (built-in baseline only). No errors are returned;
// callers treat the result as purely additive to the built-in lists.
//
// Supported ConfigMap data keys (comma- and/or newline-separated, spaces trimmed):
//   - allowNamespaces  — extra namespaces to suppress
//   - allowRolePrefixes — extra name prefixes to suppress
//   - allowRoleNames    — extra exact role names to suppress
func loadRBACAllowlist(ctx context.Context, src snapshot.Source) rbacAllowlist {
	ns := installNamespace()
	obj, err := src.Get(ctx, gvrConfigMap, ns, "srenix-rbac-allowlist")
	if err != nil || obj == nil {
		return rbacAllowlist{}
	}
	data, _, _ := unstructured.NestedStringMap(obj.Object, "data")
	if len(data) == 0 {
		return rbacAllowlist{}
	}

	extra := rbacAllowlist{
		extraNamespaces: map[string]struct{}{},
		extraExactNames: map[string]struct{}{},
	}

	parseEntries := func(raw string) []string {
		// split on comma or newline; trim spaces; drop blanks
		parts := strings.FieldsFunc(raw, func(r rune) bool {
			return r == ',' || r == '\n' || r == '\r'
		})
		var out []string
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				out = append(out, p)
			}
		}
		return out
	}

	for _, ns := range parseEntries(data["allowNamespaces"]) {
		extra.extraNamespaces[ns] = struct{}{}
	}
	extra.extraPrefixes = append(extra.extraPrefixes, parseEntries(data["allowRolePrefixes"])...)
	for _, name := range parseEntries(data["allowRoleNames"]) {
		extra.extraExactNames[name] = struct{}{}
	}
	return extra
}

// Run walks the RBAC + ServiceAccount surfaces and emits one
// Diagnostic per drift signal.
//
// Feature 1 — Configurable allowlist via ConfigMap: on each Run, we
// best-effort read the `srenix-rbac-allowlist` ConfigMap from the install
// namespace and merge its entries with the built-in baseline for this
// run only. The built-in package-level vars are never mutated.
//
// Feature 2 — Suppressed-RBAC digest: when ≥1 finding is suppressed
// by the allowlist (built-in or ConfigMap), exactly one `info` diagnostic
// is appended after both checks, summarising what was silenced. This
// collapses N paging warnings into ONE info line for visibility.
// Note: `info` still posts to Slack, but this keeps it as one line vs N.
func (r RBACDrift) Run(ctx context.Context, src snapshot.Source) []Diagnostic {
	extra := loadRBACAllowlist(ctx, src)

	var out []Diagnostic
	var suppressed []string
	var wildcardOut, wildcardSuppressed = r.checkWildcardVerbs(ctx, src, gvrClusterRole, "ClusterRole", extra)
	out = append(out, wildcardOut...)
	suppressed = append(suppressed, wildcardSuppressed...)

	var roleOut, roleSuppressed = r.checkWildcardVerbs(ctx, src, gvrRole, "Role", extra)
	out = append(out, roleOut...)
	suppressed = append(suppressed, roleSuppressed...)

	var saOut, saSuppressed = r.checkUnboundServiceAccounts(ctx, src, extra)
	out = append(out, saOut...)
	suppressed = append(suppressed, saSuppressed...)

	// Feature 2: emit a single info digest when anything was suppressed.
	// The digest itself is NOT counted as a suppressed finding.
	if len(suppressed) > 0 {
		const maxDisplay = 12
		subjects := suppressed
		var overflow int
		if len(subjects) > maxDisplay {
			overflow = len(subjects) - maxDisplay
			subjects = subjects[:maxDisplay]
		}
		msg := fmt.Sprintf("%d expected-system RBAC finding(s) suppressed as allowlisted: %s",
			len(suppressed), strings.Join(subjects, ", "))
		if overflow > 0 {
			msg += fmt.Sprintf(" +%d more", overflow)
		}
		out = append(out, Diagnostic{
			Source:   "RBACDrift",
			Severity: "info",
			Subject:  "RBACDrift/cluster/suppressed-expected-system",
			Message:  msg,
			Remediation: "These are RBAC wildcard/unbound-SA findings on known system/operator components, " +
				"suppressed to reduce noise. To extend or shrink the allowlist, create or edit the " +
				"`srenix-rbac-allowlist` ConfigMap in the Srenix install namespace " +
				"(keys: allowNamespaces / allowRolePrefixes / allowRoleNames, comma- or newline-separated).",
		})
	}

	return out
}

// checkWildcardVerbs walks Role / ClusterRole resources and flags any
// rule whose verbs include `"*"` against a non-system resource and
// the role itself isn't a system canonical role.
//
// Returns (diagnostics, suppressedSubjects). Suppressed subjects are
// collected so Run() can emit the digest diagnostic (Feature 2).
func (r RBACDrift) checkWildcardVerbs(ctx context.Context, src snapshot.Source, gvr schema.GroupVersionResource, kind string, extra rbacAllowlist) ([]Diagnostic, []string) {
	list, err := src.List(ctx, gvr, "")
	if err != nil || list == nil || len(list.Items) == 0 {
		logListFailure(gvr.Resource, err, true) // silent when the CRD/resource is absent; logs Forbidden etc.
		return nil, nil
	}
	var out []Diagnostic
	var suppressed []string
	for i := range list.Items {
		role := &list.Items[i]
		name := role.GetName()
		ns := role.GetNamespace()

		subject := fmt.Sprintf("%s/%s/%s", kind, nsOrCluster(ns), name)
		rules, _, _ := unstructured.NestedSlice(role.Object, "rules")
		hasWildcard := false
		for _, ru := range rules {
			rule, ok := ru.(map[string]interface{})
			if !ok {
				continue
			}
			verbs, _, _ := unstructured.NestedStringSlice(rule, "verbs")
			if !containsString(verbs, "*") {
				continue
			}
			apiGroups, _, _ := unstructured.NestedStringSlice(rule, "apiGroups")
			if onlySystemAPIGroups(apiGroups) {
				continue
			}
			hasWildcard = true
			break
		}
		if !hasWildcard {
			continue
		}

		// Check allowlist AFTER confirming there's a wildcard — so we
		// only collect suppressed subjects for roles that actually had
		// a wildcard finding to suppress (not every role we visited).
		if isSystemRBACWithExtra(name, ns, extra) {
			suppressed = append(suppressed, subject)
			continue
		}

		// Rebuild the rule details for the diagnostic message.
		for _, ru := range rules {
			rule, ok := ru.(map[string]interface{})
			if !ok {
				continue
			}
			verbs, _, _ := unstructured.NestedStringSlice(rule, "verbs")
			if !containsString(verbs, "*") {
				continue
			}
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
	return out, suppressed
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
//
// Returns (diagnostics, suppressedSubjects). Suppressed subjects are
// collected so Run() can emit the digest diagnostic (Feature 2).
func (r RBACDrift) checkUnboundServiceAccounts(ctx context.Context, src snapshot.Source, extra rbacAllowlist) ([]Diagnostic, []string) {
	pods, err := src.List(ctx, snapshot.GVRPod, "")
	if err != nil || pods == nil {
		logListFailure("pods", err, true) // silent when the CRD/resource is absent; logs Forbidden etc.
		return nil, nil
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
		// ConfigMap-extended namespace check for pod collection phase
		if _, ok := extra.extraNamespaces[ns]; ok {
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
		return nil, nil
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
	var suppressed []string
	for ref := range refs {
		if _, isBound := bound[ref]; isBound {
			continue
		}
		subject := fmt.Sprintf("ServiceAccount/%s/%s", ref.ns, ref.name)

		// Check ConfigMap-extended allowlist for SA name exact matches.
		// (Namespace-based suppression was already applied during pod collection above.)
		if _, ok := extra.extraExactNames[ref.name]; ok {
			suppressed = append(suppressed, subject)
			continue
		}
		for _, prefix := range extra.extraPrefixes {
			if strings.HasPrefix(ref.name, prefix) {
				suppressed = append(suppressed, subject)
				goto nextRef
			}
		}

		out = append(out, Diagnostic{
			Source:   "RBACDrift",
			Subject:  subject,
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
	nextRef:
	}
	return out, suppressed
}

// isSystemRBAC reports whether the role/binding is one the K8s control
// plane or a well-known third-party operator manages — wildcard verbs
// and unbound SAs there are expected and non-actionable.
//
// Three suppression paths:
//  1. Namespace is in systemRBACNamespaces (e.g. kube-system,
//     calico-system, kasten-io).
//  2. Name starts with a prefix in systemRBACNamePrefixes (e.g.
//     system:, k10-, calico-).
//  3. Name is an exact match in systemRBACExactNames (e.g. admin,
//     edit, view, cluster-owner). Exact matching is intentional: it
//     prevents over-suppression of user roles like "custom-admin" or
//     "payments-admin" that should still be flagged.
func isSystemRBAC(name, namespace string) bool {
	if _, ok := systemRBACNamespaces[namespace]; ok {
		return true
	}
	for _, prefix := range systemRBACNamePrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	if _, ok := systemRBACExactNames[name]; ok {
		return true
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
