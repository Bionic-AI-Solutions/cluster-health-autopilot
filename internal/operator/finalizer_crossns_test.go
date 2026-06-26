// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	"context"
	"testing"

	chav1alpha1 "github.com/srenix-ai/agentic-sre/api/v1alpha1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

// P1.9(c) — finalizer must clean up the cross-namespace approval events
// Role + RoleBinding.
//
// When a CR pins spec.approval.auditNamespace to a namespace OTHER than
// the CR's own, the operator creates the approval-server "<name>-events"
// Role + RoleBinding in that foreign namespace. Those objects carry NO
// ownerRef (cross-namespace ownerRefs are illegal), so Kubernetes GC
// won't reap them. Pre-fix, the finalizer (finalizeReaderRBAC) deleted
// only the two ClusterRoleBindings — the cross-namespace events RBAC
// leaked on CR deletion. This test asserts the finalizer now deletes
// them.
func TestReconcile_OnDelete_CleansCrossNamespaceEventsRBAC(t *testing.T) {
	cr := approvalCR()
	cr.Spec.Approval.AuditNamespace = "audit-ns" // different from cr.Namespace (srenix-system)

	r, c := newReconciler(t, cr)
	// Steady state: pass 1 adds finalizer, pass 2 reconciles children.
	reconcileOnce(t, r, cr)
	reconcileOnce(t, r, cr)

	eventsName := ApprovalServerName(cr) + "-events"
	auditNS := ApprovalAuditNamespace(cr)
	if auditNS != "audit-ns" {
		t.Fatalf("precondition: audit namespace = %q, want audit-ns", auditNS)
	}

	// Precondition: cross-ns events Role + RoleBinding exist.
	var role rbacv1.Role
	if err := c.Get(context.Background(), types.NamespacedName{Name: eventsName, Namespace: auditNS}, &role); err != nil {
		t.Fatalf("precondition: cross-ns events Role not created: %v", err)
	}
	var binding rbacv1.RoleBinding
	if err := c.Get(context.Background(), types.NamespacedName{Name: eventsName, Namespace: auditNS}, &binding); err != nil {
		t.Fatalf("precondition: cross-ns events RoleBinding not created: %v", err)
	}

	// Delete the CR and run the finalizer pass.
	var live chav1alpha1.AgenticSRE
	if err := c.Get(context.Background(), types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}, &live); err != nil {
		t.Fatalf("get CR: %v", err)
	}
	if err := c.Delete(context.Background(), &live); err != nil {
		t.Fatalf("delete CR: %v", err)
	}
	reconcileOnce(t, r, cr)

	// The cross-namespace events Role + RoleBinding must be gone.
	err := c.Get(context.Background(), types.NamespacedName{Name: eventsName, Namespace: auditNS}, &rbacv1.Role{})
	if !apierrors.IsNotFound(err) {
		t.Errorf("cross-ns events Role survived finalizer (err=%v); want NotFound", err)
	}
	err = c.Get(context.Background(), types.NamespacedName{Name: eventsName, Namespace: auditNS}, &rbacv1.RoleBinding{})
	if !apierrors.IsNotFound(err) {
		t.Errorf("cross-ns events RoleBinding survived finalizer (err=%v); want NotFound", err)
	}
}

// Guard: when auditNamespace is the CR's own namespace, the events
// Role+RoleBinding ARE owner-ref'd and GC handles them — the finalizer's
// extra deletion is a harmless no-op (NotFound is ignored). This test
// makes sure the new cleanup doesn't error in the common same-namespace
// case.
func TestReconcile_OnDelete_SameNamespaceEventsRBAC_NoError(t *testing.T) {
	cr := approvalCR() // default audit namespace == cr.Namespace

	r, c := newReconciler(t, cr)
	reconcileOnce(t, r, cr)
	reconcileOnce(t, r, cr)

	var live chav1alpha1.AgenticSRE
	if err := c.Get(context.Background(), types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}, &live); err != nil {
		t.Fatalf("get CR: %v", err)
	}
	if err := c.Delete(context.Background(), &live); err != nil {
		t.Fatalf("delete CR: %v", err)
	}
	// Must not error even though the same-ns events RBAC may already be
	// GC'd (or still present and then deleted).
	reconcileOnce(t, r, cr)
}
