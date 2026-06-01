// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	"context"
	"testing"
	"time"

	chav1alpha1 "github.com/Bionic-AI-Solutions/cluster-health-autopilot/api/v1alpha1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

// --- Phase 1c — operator-provisioned reader RBAC ---

func TestReconcile_CreatesReaderClusterRoleAndBinding(t *testing.T) {
	cr := fullCR()
	r, c := newReconciler(t, cr)
	reconcileOnce(t, r, cr)
	// Second pass: the first pass added the finalizer (which triggers
	// a Status-less Update + early return); the second pass actually
	// creates the children. Tests of "creates X" need two ticks because
	// the Add-finalizer path returns before the subresource reconcile.
	reconcileOnce(t, r, cr)

	var role rbacv1.ClusterRole
	if err := c.Get(context.Background(), types.NamespacedName{Name: ReaderClusterRoleName}, &role); err != nil {
		t.Fatalf("ClusterRole not created: %v", err)
	}
	if len(role.Rules) == 0 {
		t.Errorf("ClusterRole has no rules — verb set lost")
	}
	// Spot-check: the core probe surface must be present.
	if !hasRule(role.Rules, "", "pods") {
		t.Errorf("ClusterRole missing core/pods read rule")
	}
	// v1.10.2 regression guard: watcher uses controller-runtime
	// lease-based leader election. Without this rule the watcher
	// emits a 461-line "cannot get leases" error every 5–10s.
	if !hasRule(role.Rules, "coordination.k8s.io", "leases") {
		t.Errorf("ClusterRole missing coordination.k8s.io/leases for watcher leader election")
	}

	var binding rbacv1.ClusterRoleBinding
	if err := c.Get(context.Background(), types.NamespacedName{Name: ReaderClusterRoleBindingName(cr)}, &binding); err != nil {
		t.Fatalf("ClusterRoleBinding not created: %v", err)
	}
	if binding.RoleRef.Name != ReaderClusterRoleName {
		t.Errorf("Binding.RoleRef.Name = %q; want %q", binding.RoleRef.Name, ReaderClusterRoleName)
	}
	if !bindingTargetsSA(&binding, ServiceAccountNameFor(cr), cr.Namespace) {
		t.Errorf("Binding does not target the CR's SA (subjects=%+v)", binding.Subjects)
	}
	if binding.Labels[ManagedByCRLabel] != cr.Name ||
		binding.Labels[ManagedByCRNamespaceLabel] != cr.Namespace {
		t.Errorf("Binding missing managed-by-cr labels: %+v", binding.Labels)
	}
}

func TestReconcile_AddsFinalizerOnFirstPass(t *testing.T) {
	cr := fullCR()
	r, c := newReconciler(t, cr)
	reconcileOnce(t, r, cr)

	var updated chav1alpha1.ClusterHealthAutopilot
	if err := c.Get(context.Background(), types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}, &updated); err != nil {
		t.Fatalf("get CR: %v", err)
	}
	if !containsString(updated.Finalizers, chav1alpha1.FinalizerOperatorRBAC) {
		t.Errorf("finalizer %q not added; got %v",
			chav1alpha1.FinalizerOperatorRBAC, updated.Finalizers)
	}
}

func TestReconcile_OnDelete_RemovesBindingAndFinalizer(t *testing.T) {
	cr := fullCR()
	r, c := newReconciler(t, cr)
	// Bring the cluster to steady state (finalizer + binding present).
	reconcileOnce(t, r, cr)
	reconcileOnce(t, r, cr)

	// Soft-delete the CR. The fake client honors DeletionTimestamp +
	// preserves Finalizers, so the next reconcile takes the delete path.
	var live chav1alpha1.ClusterHealthAutopilot
	if err := c.Get(context.Background(), types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}, &live); err != nil {
		t.Fatalf("get CR: %v", err)
	}
	if err := c.Delete(context.Background(), &live); err != nil {
		t.Fatalf("delete CR: %v", err)
	}
	reconcileOnce(t, r, cr)

	// Binding should be gone.
	var binding rbacv1.ClusterRoleBinding
	err := c.Get(context.Background(), types.NamespacedName{Name: ReaderClusterRoleBindingName(cr)}, &binding)
	if !apierrors.IsNotFound(err) {
		t.Errorf("binding still present after delete (err=%v); want NotFound", err)
	}
	// The shared ClusterRole MUST survive (other CRs may still need it).
	var role rbacv1.ClusterRole
	if err := c.Get(context.Background(), types.NamespacedName{Name: ReaderClusterRoleName}, &role); err != nil {
		t.Errorf("shared ClusterRole should survive CR delete; got %v", err)
	}
	// CR itself should be fully GC'd by the fake client (finalizer cleared).
	err = c.Get(context.Background(), types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}, &live)
	if !apierrors.IsNotFound(err) {
		t.Errorf("CR should be GC'd after finalizer removed; got %v", err)
	}
}

func TestReconcile_FinalizerSkipsBindingNotLabeledByUs(t *testing.T) {
	// Defense in depth: a manually-created ClusterRoleBinding that
	// happens to share the same name (e.g. `kubectl create
	// clusterrolebinding cha-operator-watcher-<ns>-<name>`) must NOT be
	// garbage-collected by the operator's finalizer.
	cr := fullCR()
	external := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: ReaderClusterRoleBindingName(cr),
			// NO ManagedByCRLabel / ManagedByCRNamespaceLabel set.
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName, Kind: "ClusterRole", Name: "some-other-role",
		},
	}
	// Set up: CR already in the deletion path, finalizer present, external binding pre-existing.
	cr.DeletionTimestamp = &metav1.Time{Time: time.Now()}
	cr.Finalizers = []string{chav1alpha1.FinalizerOperatorRBAC}
	r, c := newReconciler(t, cr, external)

	reconcileOnce(t, r, cr)

	// External binding must still exist — operator only deletes its own.
	var binding rbacv1.ClusterRoleBinding
	if err := c.Get(context.Background(), types.NamespacedName{Name: external.Name}, &binding); err != nil {
		t.Errorf("operator deleted a binding it did NOT manage; defense-in-depth broken (err=%v)", err)
	}
}

func TestReconcile_ReaderRBACReadyCondition_True(t *testing.T) {
	cr := fullCR()
	r, c := newReconciler(t, cr)
	reconcileOnce(t, r, cr)
	reconcileOnce(t, r, cr)

	var updated chav1alpha1.ClusterHealthAutopilot
	_ = c.Get(context.Background(), types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}, &updated)
	cond := findCondition(updated.Status.Conditions, chav1alpha1.ConditionReaderRBACReady)
	if cond == nil {
		t.Fatalf("ReaderRBACReady condition not set")
	}
	if cond.Status != metav1.ConditionTrue {
		t.Errorf("ReaderRBACReady = %q (reason=%s, msg=%s); want True",
			cond.Status, cond.Reason, cond.Message)
	}
}

func TestReconcile_ReaderRBACReady_WrongSubject(t *testing.T) {
	// A binding that exists but targets a different SA must report
	// ReaderRBACReady=False/WrongSubject and gate Ready off.
	cr := fullCR()
	staleBinding := BuildReaderClusterRoleBinding(cr)
	staleBinding.Subjects = []rbacv1.Subject{
		{Kind: rbacv1.ServiceAccountKind, Name: "wrong-sa", Namespace: cr.Namespace},
	}
	role := BuildReaderClusterRole()
	r, c := newReconciler(t, cr, role, staleBinding)

	// On first pass the finalizer add returns early; second pass runs
	// reconcileReaderRBAC which CORRECTS the Subjects.
	reconcileOnce(t, r, cr)
	reconcileOnce(t, r, cr)

	var fixed rbacv1.ClusterRoleBinding
	_ = c.Get(context.Background(), types.NamespacedName{Name: staleBinding.Name}, &fixed)
	if !bindingTargetsSA(&fixed, ServiceAccountNameFor(cr), cr.Namespace) {
		t.Errorf("reconciler did not correct stale Subjects; got %+v", fixed.Subjects)
	}
}

// TestBuildReaderClusterRole_DriftReportsWriteVerbs — issue #139 fix.
// The watcher CREATES and DELETES DriftReports each cycle; pre-1.12.1 the
// reader role only had get/list/watch, producing a 403 every reconcile
// ("cannot delete driftreports"). Verify the role now grants the full
// lifecycle on driftreports + resolutionrecords AND keeps silences
// read-only (SREs own those via kubectl apply).
func TestBuildReaderClusterRole_DriftReportsWriteVerbs(t *testing.T) {
	role := BuildReaderClusterRole()

	type want struct {
		group, resource string
		verbs           []string
	}
	cases := []want{
		{"cha.bionicaisolutions.com", "driftreports",
			[]string{"get", "list", "watch", "create", "update", "patch", "delete"}},
		{"cha.bionicaisolutions.com", "resolutionrecords",
			[]string{"get", "list", "watch", "create", "update", "patch", "delete"}},
		{"cha.bionicaisolutions.com", "silences",
			[]string{"get", "list", "watch"}}, // read-only; SREs own these
	}
	for _, c := range cases {
		var matched *rbacv1.PolicyRule
		for i := range role.Rules {
			r := &role.Rules[i]
			if containsString(r.APIGroups, c.group) && containsString(r.Resources, c.resource) {
				matched = r
				break
			}
		}
		if matched == nil {
			t.Errorf("no rule found for %s/%s", c.group, c.resource)
			continue
		}
		// Required verbs present
		for _, v := range c.verbs {
			if !containsString(matched.Verbs, v) {
				t.Errorf("%s/%s: missing verb %q (got %v)", c.group, c.resource, v, matched.Verbs)
			}
		}
		// silences must NOT have delete — separation-of-authority guard
		if c.resource == "silences" && containsString(matched.Verbs, "delete") {
			t.Errorf("silences must not have 'delete' — SREs own them (got %v)", matched.Verbs)
		}
	}
}

// --- helpers ---

func hasRule(rules []rbacv1.PolicyRule, group, resource string) bool {
	for _, r := range rules {
		gMatch := false
		for _, g := range r.APIGroups {
			if g == group {
				gMatch = true
				break
			}
		}
		if !gMatch {
			continue
		}
		for _, res := range r.Resources {
			if res == resource {
				return true
			}
		}
	}
	return false
}

func containsString(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}

func findCondition(conds []metav1.Condition, ctype string) *metav1.Condition {
	for i := range conds {
		if conds[i].Type == ctype {
			return &conds[i]
		}
	}
	return nil
}

// Compile-time fix-up: import ctrl indirectly so this file
// doesn't grow an unused-import warning for shared scaffolding.
var _ = ctrl.Request{}
