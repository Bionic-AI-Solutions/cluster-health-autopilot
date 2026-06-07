// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package diagnose

import (
	"context"
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
	// flagging them is noise.
	src := &memSourceRBAC{byResource: map[string][]unstructured.Unstructured{
		"clusterroles": {
			makeClusterRole("cluster-admin", []string{"*"}, []string{"*"}, []string{"*"}),
			makeClusterRole("system:kube-controller-manager", []string{"*"}, []string{""}, []string{"endpoints"}),
		},
	}}
	got := RBACDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("system roles should be skipped; got %+v", got)
	}
}

func TestRBACDrift_KubeSystemRole_Skipped(t *testing.T) {
	src := &memSourceRBAC{byResource: map[string][]unstructured.Unstructured{
		"roles": {makeRole("kube-system", "extension-apiserver-authentication-reader",
			[]string{"*"}, []string{""}, []string{"configmaps"})},
	}}
	got := RBACDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("kube-system roles should be skipped; got %+v", got)
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
