// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package diagnose

import (
	"context"
	"fmt"
	"strings"
	"testing"

	pkgsnapshot "github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/snapshot"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type memSourceRBAC struct {
	byResource map[string][]unstructured.Unstructured
}

func (m *memSourceRBAC) List(_ context.Context, gvr schema.GroupVersionResource, ns string) (*unstructured.UnstructuredList, error) {
	out := &unstructured.UnstructuredList{}
	for _, u := range m.byResource[gvr.Resource] {
		if ns != "" && u.GetNamespace() != ns {
			continue
		}
		out.Items = append(out.Items, u)
	}
	return out, nil
}

func (m *memSourceRBAC) Get(_ context.Context, gvr schema.GroupVersionResource, ns, name string) (*unstructured.Unstructured, error) {
	for _, u := range m.byResource[gvr.Resource] {
		if u.GetNamespace() == ns && u.GetName() == name {
			return &u, nil
		}
	}
	return nil, nil
}

func (m *memSourceRBAC) Mode() pkgsnapshot.Mode { return pkgsnapshot.ModeLive }

func makeRole(ns, name string, verbs, apiGroups, resources []string) unstructured.Unstructured {
	u := unstructured.Unstructured{}
	u.SetAPIVersion("rbac.authorization.k8s.io/v1")
	u.SetKind("Role")
	u.SetNamespace(ns)
	u.SetName(name)
	rule := map[string]interface{}{
		"verbs":     toIface(verbs),
		"apiGroups": toIface(apiGroups),
		"resources": toIface(resources),
	}
	_ = unstructured.SetNestedSlice(u.Object, []interface{}{rule}, "rules")
	return u
}

func makeClusterRole(name string, verbs, apiGroups, resources []string) unstructured.Unstructured {
	u := unstructured.Unstructured{}
	u.SetAPIVersion("rbac.authorization.k8s.io/v1")
	u.SetKind("ClusterRole")
	u.SetName(name)
	rule := map[string]interface{}{
		"verbs":     toIface(verbs),
		"apiGroups": toIface(apiGroups),
		"resources": toIface(resources),
	}
	_ = unstructured.SetNestedSlice(u.Object, []interface{}{rule}, "rules")
	return u
}

func toIface(ss []string) []interface{} {
	out := make([]interface{}, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}

func makeRBACPod(ns, name, saName string) unstructured.Unstructured {
	u := unstructured.Unstructured{}
	u.SetAPIVersion("v1")
	u.SetKind("Pod")
	u.SetNamespace(ns)
	u.SetName(name)
	_ = unstructured.SetNestedField(u.Object, saName, "spec", "serviceAccountName")
	return u
}

// makeRBACPodWithAutomount mirrors makeRBACPod but sets pod-level
// spec.automountServiceAccountToken. Pod-level overrides SA-level.
func makeRBACPodWithAutomount(ns, name, saName string, automount bool) unstructured.Unstructured {
	u := makeRBACPod(ns, name, saName)
	_ = unstructured.SetNestedField(u.Object, automount, "spec", "automountServiceAccountToken")
	return u
}

// makeServiceAccount builds a core/v1 ServiceAccount.
// automount=nil leaves the field unset (defaults to true on the API
// server). automount=&true / &false sets it explicitly.
func makeServiceAccount(ns, name string, automount *bool) unstructured.Unstructured {
	u := unstructured.Unstructured{}
	u.SetAPIVersion("v1")
	u.SetKind("ServiceAccount")
	u.SetNamespace(ns)
	u.SetName(name)
	if automount != nil {
		_ = unstructured.SetNestedField(u.Object, *automount, "automountServiceAccountToken")
	}
	return u
}

func makeRoleBinding(ns, name, saNamespace, saName string) unstructured.Unstructured {
	u := unstructured.Unstructured{}
	u.SetAPIVersion("rbac.authorization.k8s.io/v1")
	u.SetKind("RoleBinding")
	u.SetNamespace(ns)
	u.SetName(name)
	sub := map[string]interface{}{
		"kind":      "ServiceAccount",
		"name":      saName,
		"namespace": saNamespace,
	}
	_ = unstructured.SetNestedSlice(u.Object, []interface{}{sub}, "subjects")
	return u
}

// --- wildcard verb tests -----------------------------------------------------

func TestRBACDrift_WildcardClusterRole_Warning(t *testing.T) {
	src := &memSourceRBAC{byResource: map[string][]unstructured.Unstructured{
		"clusterroles": {makeClusterRole("app-operator", []string{"*"}, []string{""}, []string{"pods", "configmaps"})},
	}}
	got := RBACDrift{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("wildcard ClusterRole should emit 1 diagnostic; got %d: %+v", len(got), got)
	}
	if got[0].Severity != "warning" {
		t.Errorf("wildcard verbs should be warning; got %q", got[0].Severity)
	}
	if !strings.Contains(got[0].Message, "wildcard verb") {
		t.Errorf("message should call out wildcard; got %q", got[0].Message)
	}
	if !strings.Contains(got[0].Subject, "ClusterRole/cluster/app-operator") {
		t.Errorf("subject should name role + cluster scope; got %q", got[0].Subject)
	}
}

func TestRBACDrift_WildcardNamespacedRole_Warning(t *testing.T) {
	src := &memSourceRBAC{byResource: map[string][]unstructured.Unstructured{
		"roles": {makeRole("app-ns", "app-role", []string{"*"}, []string{""}, []string{"secrets"})},
	}}
	got := RBACDrift{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("wildcard Role should emit 1 diagnostic; got %d: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Subject, "Role/app-ns/app-role") {
		t.Errorf("subject should name namespace + role; got %q", got[0].Subject)
	}
}

func TestRBACDrift_NonWildcard_Silent(t *testing.T) {
	src := &memSourceRBAC{byResource: map[string][]unstructured.Unstructured{
		"roles": {makeRole("app-ns", "app-role", []string{"get", "list", "watch"}, []string{""}, []string{"pods"})},
	}}
	got := RBACDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("explicit verbs should produce 0 diagnostics; got %+v", got)
	}
}

func TestRBACDrift_SystemClusterRole_Skipped(t *testing.T) {
	// cluster-admin and system:* roles have wildcards by design;
	// flagging them is noise. They are suppressed → a suppressed digest
	// (info severity) may be emitted, but no warning must appear.
	src := &memSourceRBAC{byResource: map[string][]unstructured.Unstructured{
		"clusterroles": {
			makeClusterRole("cluster-admin", []string{"*"}, []string{"*"}, []string{"*"}),
			makeClusterRole("system:kube-controller-manager", []string{"*"}, []string{""}, []string{"endpoints"}),
		},
	}}
	got := RBACDrift{}.Run(context.Background(), src)
	for _, d := range got {
		if d.Severity == "warning" {
			t.Errorf("system roles should be skipped (no warning); got %+v", d)
		}
	}
}

func TestRBACDrift_KubeSystemRole_Skipped(t *testing.T) {
	src := &memSourceRBAC{byResource: map[string][]unstructured.Unstructured{
		"roles": {makeRole("kube-system", "extension-apiserver-authentication-reader",
			[]string{"*"}, []string{""}, []string{"configmaps"})},
	}}
	got := RBACDrift{}.Run(context.Background(), src)
	// kube-system roles are suppressed → a digest (info) may appear but no warning.
	for _, d := range got {
		if d.Severity == "warning" {
			t.Errorf("kube-system roles should be skipped (no warning); got %+v", d)
		}
	}
}

func TestRBACDrift_SystemAPIGroupOnly_Skipped(t *testing.T) {
	// A wildcard rule scoped only to authentication.k8s.io / authorization.k8s.io
	// is expected in user-defined roles that wrap kube-controller-manager
	// permissions; don't flag.
	src := &memSourceRBAC{byResource: map[string][]unstructured.Unstructured{
		"clusterroles": {makeClusterRole("metrics-reader",
			[]string{"*"},
			[]string{"authentication.k8s.io", "authorization.k8s.io"},
			[]string{"tokenreviews", "subjectaccessreviews"})},
	}}
	got := RBACDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("system-only api groups should be skipped; got %+v", got)
	}
}

// --- unbound ServiceAccount tests -------------------------------------------

func TestRBACDrift_UnboundSA_Warning(t *testing.T) {
	// Pod mounts SA myapp; no RoleBinding/ClusterRoleBinding references it.
	src := &memSourceRBAC{byResource: map[string][]unstructured.Unstructured{
		"pods": {makeRBACPod("billing", "billing-7d", "myapp")},
	}}
	got := RBACDrift{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("unbound SA referenced by Pod should emit 1 diagnostic; got %d: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Subject, "ServiceAccount/billing/myapp") {
		t.Errorf("subject should name SA; got %q", got[0].Subject)
	}
	if !strings.Contains(got[0].Remediation, "create rolebinding") {
		t.Errorf("remediation should show fix command; got %q", got[0].Remediation)
	}
	// Phase 1.B.4: the legacy `--role=<role-name>` placeholder is gone;
	// rendered remediation must use a NAME-substitute hint instead.
	if strings.Contains(got[0].Remediation, "<role-name>") {
		t.Errorf("remediation must not contain literal <role-name>; got %q", got[0].Remediation)
	}
}

func TestRBACDrift_BoundSA_Silent(t *testing.T) {
	src := &memSourceRBAC{byResource: map[string][]unstructured.Unstructured{
		"pods":         {makeRBACPod("billing", "billing-7d", "myapp")},
		"rolebindings": {makeRoleBinding("billing", "myapp-binding", "billing", "myapp")},
	}}
	got := RBACDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("bound SA should be silent; got %+v", got)
	}
}

func TestRBACDrift_DefaultSA_Skipped(t *testing.T) {
	// "default" SA is the K8s out-of-the-box default; not having a
	// RoleBinding is the design, not drift.
	src := &memSourceRBAC{byResource: map[string][]unstructured.Unstructured{
		"pods": {makeRBACPod("billing", "billing-7d", "default")},
	}}
	got := RBACDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("default SA should be skipped; got %+v", got)
	}
}

// TestRBACDrift_SAWithAutomountFalse_Silent — the v1.11.1 bug fix.
// When the SA has automountServiceAccountToken=false, the API server
// does NOT mount the token into the Pod; a Role binding would be
// useless. Flagging it as "unbound" was producing false-positive
// noise on every Helm chart that ships hardened SAs (langfuse,
// openproject, meilisearch, external-secrets-webhook, etc.).
func TestRBACDrift_SAWithAutomountFalse_Silent(t *testing.T) {
	f := false
	src := &memSourceRBAC{byResource: map[string][]unstructured.Unstructured{
		"pods":            {makeRBACPod("billing", "billing-7d", "myapp")},
		"serviceaccounts": {makeServiceAccount("billing", "myapp", &f)},
	}}
	got := RBACDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("SA with automountServiceAccountToken=false must be silent; got %+v", got)
	}
}

// TestRBACDrift_SAWithAutomountTrueExplicit_StillFlagged — explicit
// automount=true is the original-flagged behavior; ensure the fix
// didn't accidentally silence the true positives.
func TestRBACDrift_SAWithAutomountTrueExplicit_StillFlagged(t *testing.T) {
	tr := true
	src := &memSourceRBAC{byResource: map[string][]unstructured.Unstructured{
		"pods":            {makeRBACPod("billing", "billing-7d", "myapp")},
		"serviceaccounts": {makeServiceAccount("billing", "myapp", &tr)},
	}}
	got := RBACDrift{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Errorf("explicit automount=true unbound SA must still fire; got %d: %+v", len(got), got)
	}
}

// TestRBACDrift_PodAutomountFalseOverridesSA — Pod-level
// automountServiceAccountToken=false wins even if the SA defaults to
// automount=true. Real-world: Helm chart enables automount on the SA
// for compatibility but workload Pod explicitly disables it.
func TestRBACDrift_PodAutomountFalseOverridesSA(t *testing.T) {
	tr := true
	src := &memSourceRBAC{byResource: map[string][]unstructured.Unstructured{
		"pods":            {makeRBACPodWithAutomount("billing", "billing-7d", "myapp", false)},
		"serviceaccounts": {makeServiceAccount("billing", "myapp", &tr)},
	}}
	got := RBACDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("Pod automount=false must override SA automount=true; got %+v", got)
	}
}

// TestRBACDrift_PodAutomountTrueOverridesSA — opposite direction.
// SA defaults to false (chart hardened it) but a Pod explicitly
// re-enables automount → token IS mounted → must flag.
func TestRBACDrift_PodAutomountTrueOverridesSA(t *testing.T) {
	f := false
	src := &memSourceRBAC{byResource: map[string][]unstructured.Unstructured{
		"pods":            {makeRBACPodWithAutomount("billing", "billing-7d", "myapp", true)},
		"serviceaccounts": {makeServiceAccount("billing", "myapp", &f)},
	}}
	got := RBACDrift{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Errorf("Pod automount=true must override SA automount=false; got %d: %+v", len(got), got)
	}
}

// TestRBACDrift_NoSAObject_DefaultsToFlagging — if the SA object
// can't be found in the snapshot (RBAC denied or namespace deleted
// mid-cycle), default to flagging. Conservative: a missing SA object
// is suspicious; we don't want to silently silence findings on
// snapshot-mode errors.
func TestRBACDrift_NoSAObject_DefaultsToFlagging(t *testing.T) {
	src := &memSourceRBAC{byResource: map[string][]unstructured.Unstructured{
		"pods": {makeRBACPod("billing", "billing-7d", "myapp")},
		// no "serviceaccounts" entries — simulate snapshot gap
	}}
	got := RBACDrift{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Errorf("missing SA object should default to flag (conservative); got %d: %+v", len(got), got)
	}
}

func TestRBACDrift_KubeSystemPod_Skipped(t *testing.T) {
	// kube-system Pods often have weird SA setups managed by the
	// cluster operator; skip.
	src := &memSourceRBAC{byResource: map[string][]unstructured.Unstructured{
		"pods": {makeRBACPod("kube-system", "kube-proxy", "kube-proxy")},
	}}
	got := RBACDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("kube-system pods should be skipped; got %+v", got)
	}
}

func TestRBACDrift_ClusterRoleBindingCrossNamespace(t *testing.T) {
	// CRB binds SA from one namespace; analyzer must walk subjects'
	// namespace correctly.
	crb := unstructured.Unstructured{}
	crb.SetAPIVersion("rbac.authorization.k8s.io/v1")
	crb.SetKind("ClusterRoleBinding")
	crb.SetName("myapp-binding")
	_ = unstructured.SetNestedSlice(crb.Object, []interface{}{
		map[string]interface{}{
			"kind":      "ServiceAccount",
			"name":      "myapp",
			"namespace": "billing",
		},
	}, "subjects")

	src := &memSourceRBAC{byResource: map[string][]unstructured.Unstructured{
		"pods":                {makeRBACPod("billing", "billing-7d", "myapp")},
		"clusterrolebindings": {crb},
	}}
	got := RBACDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("SA bound via ClusterRoleBinding should be silent; got %+v", got)
	}
}

func TestRBACDrift_EmptyCluster_NoOp(t *testing.T) {
	src := &memSourceRBAC{byResource: map[string][]unstructured.Unstructured{}}
	got := RBACDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("empty cluster should produce 0 diagnostics; got %+v", got)
	}
}

// --- expanded allowlist: suppressed operator/system ClusterRoles --------

// TestRBACDrift_KnownOperatorPrefixes_Suppressed verifies that wildcard
// ClusterRoles with well-known operator name prefixes (k10-, kasten-,
// calico-, minio-, k3s-, olm., local-path-, console-) are suppressed.
func TestRBACDrift_KnownOperatorPrefixes_Suppressed(t *testing.T) {
	suppressed := []string{
		"k10-admin",
		"kasten-admin",
		"calico-tiered-policy-passthrough",
		"minio-operator",
		"k3s-cloud-controller-manager",
		"local-path-provisioner-role",
		"console-sa-role",
		"olm.og.some-operator",
	}
	for _, name := range suppressed {
		name := name
		t.Run(name, func(t *testing.T) {
			src := &memSourceRBAC{byResource: map[string][]unstructured.Unstructured{
				"clusterroles": {makeClusterRole(name, []string{"*"}, []string{""}, []string{"*"})},
			}}
			got := RBACDrift{}.Run(context.Background(), src)
			// Suppressed roles emit a digest (info), not a warning.
			for _, d := range got {
				if d.Severity == "warning" {
					t.Errorf("operator/system ClusterRole %q should be suppressed (no warning); got: %+v",
						name, d)
				}
			}
		})
	}
}

// TestRBACDrift_CanonicalExactNames_Suppressed verifies that the exact
// names admin, edit, view, cluster-owner, local-clusterowner are
// suppressed regardless of their rules — these are K8s aggregated roles
// or well-known cluster-owner roles that legitimately hold wildcard verbs.
func TestRBACDrift_CanonicalExactNames_Suppressed(t *testing.T) {
	suppressed := []string{
		"admin",
		"edit",
		"view",
		"cluster-owner",
		"local-clusterowner",
	}
	for _, name := range suppressed {
		name := name
		t.Run(name, func(t *testing.T) {
			src := &memSourceRBAC{byResource: map[string][]unstructured.Unstructured{
				"clusterroles": {makeClusterRole(name, []string{"*"}, []string{""}, []string{"*"})},
			}}
			got := RBACDrift{}.Run(context.Background(), src)
			// Suppressed roles emit a digest (info), not a warning.
			for _, d := range got {
				if d.Severity == "warning" {
					t.Errorf("canonical ClusterRole %q should be suppressed (no warning); got: %+v",
						name, d)
				}
			}
		})
	}
}

// TestRBACDrift_UserRoleNotOverSuppressed verifies that user-defined
// roles whose names RESEMBLE system names but are not exact matches or
// exact prefixes are still flagged.  This is the over-suppression guard:
// "custom-admin" is not the canonical "admin", "payments-admin" is not
// "admin", and "my-app-wildcard" has no system prefix.
func TestRBACDrift_UserRoleNotOverSuppressed(t *testing.T) {
	cases := []struct {
		kind string // "clusterroles" or "roles"
		ns   string // namespace (empty for ClusterRole)
		name string
	}{
		{kind: "clusterroles", ns: "", name: "my-app-wildcard"},
		{kind: "roles", ns: "team-x", name: "custom-admin"},
		{kind: "clusterroles", ns: "", name: "payments-admin"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.kind+"/"+tc.ns+"/"+tc.name, func(t *testing.T) {
			var obj unstructured.Unstructured
			if tc.ns == "" {
				obj = makeClusterRole(tc.name, []string{"*"}, []string{""}, []string{"pods"})
			} else {
				obj = makeRole(tc.ns, tc.name, []string{"*"}, []string{""}, []string{"pods"})
			}
			src := &memSourceRBAC{byResource: map[string][]unstructured.Unstructured{
				tc.kind: {obj},
			}}
			got := RBACDrift{}.Run(context.Background(), src)
			if len(got) != 1 {
				t.Errorf("user role %q must still be flagged (not suppressed); got %d diagnostics: %+v",
					tc.name, len(got), got)
			}
		})
	}
}

// TestRBACDrift_OperatorNamespaceUnboundSA_Suppressed verifies that
// unbound ServiceAccounts in well-known operator namespaces (e.g.
// calico-system, tigera-operator) are NOT reported, because those SAs
// are managed by the operator itself and lack user-facing RoleBindings
// by design.
func TestRBACDrift_OperatorNamespaceUnboundSA_Suppressed(t *testing.T) {
	operatorNSCases := []struct {
		ns  string
		sa  string
		pod string
	}{
		{"calico-system", "csi-node-driver", "csi-node-driver-abc"},
		{"tigera-operator", "tigera-operator", "tigera-operator-xyz"},
	}
	for _, tc := range operatorNSCases {
		tc := tc
		t.Run(tc.ns+"/"+tc.sa, func(t *testing.T) {
			src := &memSourceRBAC{byResource: map[string][]unstructured.Unstructured{
				"pods": {makeRBACPod(tc.ns, tc.pod, tc.sa)},
			}}
			got := RBACDrift{}.Run(context.Background(), src)
			if len(got) != 0 {
				t.Errorf("unbound SA in operator namespace %q should be suppressed; got %d diagnostics: %+v",
					tc.ns, len(got), got)
			}
		})
	}
}

// TestRBACDrift_UserNamespaceUnboundSA_StillFlagged verifies that an
// unbound SA in a plain user namespace (e.g. "payments") is still
// reported after the operator-namespace suppression was added.
func TestRBACDrift_UserNamespaceUnboundSA_StillFlagged(t *testing.T) {
	src := &memSourceRBAC{byResource: map[string][]unstructured.Unstructured{
		"pods": {makeRBACPod("payments", "payments-api-xyz", "payments-sa")},
	}}
	got := RBACDrift{}.Run(context.Background(), src)
	// Expect exactly 1 warning (no suppressed items → no digest)
	if len(got) != 1 {
		t.Errorf("unbound SA in user namespace 'payments' must still be flagged; got %d diagnostics: %+v",
			len(got), got)
	}
	if !strings.Contains(got[0].Subject, "ServiceAccount/payments/payments-sa") {
		t.Errorf("subject should name the SA; got %q", got[0].Subject)
	}
}

// --- Feature 1: ConfigMap-based allowlist extension --------------------------

// makeConfigMap builds a core/v1 ConfigMap with arbitrary data.
func makeConfigMap(ns, name string, data map[string]string) unstructured.Unstructured {
	u := unstructured.Unstructured{}
	u.SetAPIVersion("v1")
	u.SetKind("ConfigMap")
	u.SetNamespace(ns)
	u.SetName(name)
	dataIface := map[string]interface{}{}
	for k, v := range data {
		dataIface[k] = v
	}
	_ = unstructured.SetNestedStringMap(u.Object, data, "data")
	return u
}

// TestRBACDrift_ConfigMapAllowlist_ExtendsPrefixes verifies that a
// `cha-rbac-allowlist` ConfigMap with `allowRolePrefixes: "myoperator-"`
// causes a ClusterRole named "myoperator-controller" (which has wildcard
// verbs) to be suppressed instead of flagged.
func TestRBACDrift_ConfigMapAllowlist_ExtendsPrefixes(t *testing.T) {
	cm := makeConfigMap("cluster-health-autopilot", "cha-rbac-allowlist", map[string]string{
		"allowRolePrefixes": "myoperator-",
	})
	src := &memSourceRBAC{byResource: map[string][]unstructured.Unstructured{
		"clusterroles": {makeClusterRole("myoperator-controller", []string{"*"}, []string{""}, []string{"pods"})},
		"configmaps":   {cm},
	}}
	got := RBACDrift{}.Run(context.Background(), src)
	// The wildcard warning should be suppressed; only the info digest may appear.
	for _, d := range got {
		if d.Severity == "warning" && strings.Contains(d.Subject, "myoperator-controller") {
			t.Errorf("myoperator-controller should be suppressed by ConfigMap allowlist; got warning: %+v", d)
		}
	}
}

// TestRBACDrift_ConfigMapAllowlist_ExtendsNamespaces verifies that adding
// "my-ns" to `allowNamespaces` suppresses an unbound SA in that namespace.
func TestRBACDrift_ConfigMapAllowlist_ExtendsNamespaces(t *testing.T) {
	cm := makeConfigMap("cluster-health-autopilot", "cha-rbac-allowlist", map[string]string{
		"allowNamespaces": "my-ns",
	})
	src := &memSourceRBAC{byResource: map[string][]unstructured.Unstructured{
		"pods":       {makeRBACPod("my-ns", "my-pod", "my-sa")},
		"configmaps": {cm},
	}}
	got := RBACDrift{}.Run(context.Background(), src)
	for _, d := range got {
		if d.Severity == "warning" && strings.Contains(d.Subject, "my-ns") {
			t.Errorf("unbound SA in my-ns should be suppressed by ConfigMap allowlist; got warning: %+v", d)
		}
	}
}

// TestRBACDrift_ConfigMapAllowlist_WasAlreadyFlagged confirms the inverse:
// without the ConfigMap, the same role IS flagged.
func TestRBACDrift_ConfigMapAllowlist_WasAlreadyFlagged(t *testing.T) {
	// No ConfigMap in source → myoperator-controller must be flagged.
	src := &memSourceRBAC{byResource: map[string][]unstructured.Unstructured{
		"clusterroles": {makeClusterRole("myoperator-controller", []string{"*"}, []string{""}, []string{"pods"})},
	}}
	got := RBACDrift{}.Run(context.Background(), src)
	var hasWarning bool
	for _, d := range got {
		if d.Severity == "warning" && strings.Contains(d.Subject, "myoperator-controller") {
			hasWarning = true
		}
	}
	if !hasWarning {
		t.Errorf("without ConfigMap, myoperator-controller must be flagged; got: %+v", got)
	}
}

// TestRBACDrift_BuiltinStillWorksNoConfigMap confirms that the built-in
// allowlist (k10-admin etc.) works correctly when no ConfigMap exists:
// k10-admin stays suppressed, my-app-wildcard stays flagged.
func TestRBACDrift_BuiltinStillWorksNoConfigMap(t *testing.T) {
	// memSourceRBAC.Get returns nil,nil for unknown objects → "no ConfigMap"
	src := &memSourceRBAC{byResource: map[string][]unstructured.Unstructured{
		"clusterroles": {
			makeClusterRole("k10-admin", []string{"*"}, []string{""}, []string{"*"}),
			makeClusterRole("my-app-wildcard", []string{"*"}, []string{""}, []string{"pods"}),
		},
	}}
	got := RBACDrift{}.Run(context.Background(), src)
	// k10-admin should be suppressed; my-app-wildcard should be warned
	var hasK10Warning, hasAppWarning bool
	for _, d := range got {
		if d.Severity == "warning" && strings.Contains(d.Subject, "k10-admin") {
			hasK10Warning = true
		}
		if d.Severity == "warning" && strings.Contains(d.Subject, "my-app-wildcard") {
			hasAppWarning = true
		}
	}
	if hasK10Warning {
		t.Errorf("k10-admin should be suppressed by built-in allowlist even without ConfigMap; got warning")
	}
	if !hasAppWarning {
		t.Errorf("my-app-wildcard must be flagged; got: %+v", got)
	}
}

// --- Feature 2: Suppressed-RBAC digest ---------------------------------------

// TestRBACDrift_SuppressedDigest_EmittedWhenSuppressed verifies that when
// at least one finding is suppressed by the allowlist, exactly one `info`
// diagnostic with subject RBACDrift/cluster/suppressed-expected-system is
// emitted, and its message contains the count and at least one suppressed subject.
func TestRBACDrift_SuppressedDigest_EmittedWhenSuppressed(t *testing.T) {
	src := &memSourceRBAC{byResource: map[string][]unstructured.Unstructured{
		"clusterroles": {
			// k10-admin is in the built-in allowlist → will be suppressed
			makeClusterRole("k10-admin", []string{"*"}, []string{""}, []string{"*"}),
		},
	}}
	got := RBACDrift{}.Run(context.Background(), src)
	var digestDiags []Diagnostic
	for _, d := range got {
		if d.Subject == "RBACDrift/cluster/suppressed-expected-system" {
			digestDiags = append(digestDiags, d)
		}
	}
	if len(digestDiags) != 1 {
		t.Fatalf("expected exactly 1 suppressed-digest info diagnostic; got %d: %+v", len(digestDiags), got)
	}
	d := digestDiags[0]
	if d.Severity != "info" {
		t.Errorf("digest diagnostic must have severity 'info'; got %q", d.Severity)
	}
	if !strings.Contains(d.Message, "1 expected-system RBAC finding(s) suppressed") {
		t.Errorf("digest message must contain count; got %q", d.Message)
	}
	if !strings.Contains(d.Message, "k10-admin") {
		t.Errorf("digest message must contain suppressed subject; got %q", d.Message)
	}
	if !strings.Contains(d.Remediation, "cha-rbac-allowlist") {
		t.Errorf("digest remediation must mention ConfigMap name; got %q", d.Remediation)
	}
}

// TestRBACDrift_SuppressedDigest_NotEmittedWhenNothingSuppressed verifies
// that when NO findings are suppressed (all roles are real user roles), the
// digest diagnostic is NOT emitted.
func TestRBACDrift_SuppressedDigest_NotEmittedWhenNothingSuppressed(t *testing.T) {
	src := &memSourceRBAC{byResource: map[string][]unstructured.Unstructured{
		"clusterroles": {
			makeClusterRole("my-app-wildcard", []string{"*"}, []string{""}, []string{"pods"}),
		},
	}}
	got := RBACDrift{}.Run(context.Background(), src)
	for _, d := range got {
		if d.Subject == "RBACDrift/cluster/suppressed-expected-system" {
			t.Errorf("no suppressed findings → digest must NOT be emitted; got: %+v", d)
		}
	}
}

// TestRBACDrift_DigestNotCountedAsSuppressed verifies that the digest
// diagnostic itself is not counted as a suppressed finding (no self-reference
// or infinite recursion).
func TestRBACDrift_DigestNotCountedAsSuppressed(t *testing.T) {
	src := &memSourceRBAC{byResource: map[string][]unstructured.Unstructured{
		"clusterroles": {
			makeClusterRole("k10-admin", []string{"*"}, []string{""}, []string{"*"}),
		},
	}}
	got := RBACDrift{}.Run(context.Background(), src)
	// Expect: 1 info digest. The digest itself must NOT appear in the count
	// embedded in its own message (i.e. "1 ... suppressed" not "2 ... suppressed").
	for _, d := range got {
		if d.Subject == "RBACDrift/cluster/suppressed-expected-system" {
			if strings.Contains(d.Message, "2 expected-system") {
				t.Errorf("digest must not count itself; message: %q", d.Message)
			}
			if !strings.Contains(d.Message, "1 expected-system") {
				t.Errorf("digest message must say 1 suppressed; got %q", d.Message)
			}
		}
	}
}

// TestRBACDrift_DigestMessageCapsAt12 verifies that when more than 12
// subjects are suppressed, the message truncates to ~12 with "+N more".
func TestRBACDrift_DigestMessageCapsAt12(t *testing.T) {
	// Build 15 operator ClusterRoles all with k10- prefix (built-in suppressed)
	var roles []unstructured.Unstructured
	for i := 0; i < 15; i++ {
		roles = append(roles, makeClusterRole(
			fmt.Sprintf("k10-role-%d", i),
			[]string{"*"}, []string{""}, []string{"pods"},
		))
	}
	src := &memSourceRBAC{byResource: map[string][]unstructured.Unstructured{
		"clusterroles": roles,
	}}
	got := RBACDrift{}.Run(context.Background(), src)
	var digest *Diagnostic
	for i := range got {
		if got[i].Subject == "RBACDrift/cluster/suppressed-expected-system" {
			digest = &got[i]
			break
		}
	}
	if digest == nil {
		t.Fatal("expected suppressed digest; got none")
	}
	if !strings.Contains(digest.Message, "+3 more") {
		t.Errorf("digest with 15 suppressions should show '+3 more'; got %q", digest.Message)
	}
	if !strings.Contains(digest.Message, "15 expected-system") {
		t.Errorf("digest message should say 15 suppressed; got %q", digest.Message)
	}
}

// TestRBACDrift_OverSuppressionGuard_AllFlaggedAfterFix repeats the
// over-suppression guard for the new code path: user roles my-app-wildcard,
// custom-admin, payments-admin, and unbound user SA must still be flagged.
func TestRBACDrift_OverSuppressionGuard_AllFlaggedAfterFix(t *testing.T) {
	cases := []struct {
		kind string
		ns   string
		name string
	}{
		{kind: "clusterroles", ns: "", name: "my-app-wildcard"},
		{kind: "roles", ns: "team-x", name: "custom-admin"},
		{kind: "clusterroles", ns: "", name: "payments-admin"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.kind+"/"+tc.ns+"/"+tc.name, func(t *testing.T) {
			var obj unstructured.Unstructured
			if tc.ns == "" {
				obj = makeClusterRole(tc.name, []string{"*"}, []string{""}, []string{"pods"})
			} else {
				obj = makeRole(tc.ns, tc.name, []string{"*"}, []string{""}, []string{"pods"})
			}
			src := &memSourceRBAC{byResource: map[string][]unstructured.Unstructured{
				tc.kind: {obj},
			}}
			got := RBACDrift{}.Run(context.Background(), src)
			var hasWarning bool
			for _, d := range got {
				if d.Severity == "warning" && strings.Contains(d.Subject, tc.name) {
					hasWarning = true
				}
			}
			if !hasWarning {
				t.Errorf("user role %q must still be flagged as warning; got: %+v", tc.name, got)
			}
		})
	}
}
