// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	"context"
	"testing"

	chav1alpha1 "github.com/Bionic-AI-Solutions/cluster-health-autopilot/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// P2.6b — NetworkPolicy restricting approval-server ingress to the
// gateway/oauth2-proxy namespace. Closes the X-Forwarded-User header
// forgery bypass: a pod reaching the approval-server ClusterIP directly
// could inject an arbitrary X-Forwarded-User (the OIDC ingress is the
// only thing that normally sets it), corrupting audit attribution.

// approvalNetpolCR returns a CR with approval + networkPolicy enabled
// and a gateway namespace selector set (required when enabled).
func approvalNetpolCR() *chav1alpha1.ClusterHealthAutopilot {
	cr := approvalCR()
	cr.Spec.Approval.NetworkPolicy = &chav1alpha1.ApprovalNetworkPolicySpec{
		Enabled: true,
		GatewayNamespaceSelector: map[string]string{
			"kubernetes.io/metadata.name": "gateway",
		},
	}
	return cr
}

// --- Builder ---

func TestBuildApprovalServerNetworkPolicy_DisabledReturnsNil(t *testing.T) {
	t.Run("approval off", func(t *testing.T) {
		cr := fullCR()
		if np := BuildApprovalServerNetworkPolicy(cr); np != nil {
			t.Errorf("approval off should produce nil NetworkPolicy; got %v", np)
		}
	})
	t.Run("netpol unset", func(t *testing.T) {
		cr := approvalCR()
		if np := BuildApprovalServerNetworkPolicy(cr); np != nil {
			t.Errorf("netpol unset should produce nil NetworkPolicy; got %v", np)
		}
	})
	t.Run("netpol disabled", func(t *testing.T) {
		cr := approvalCR()
		cr.Spec.Approval.NetworkPolicy = &chav1alpha1.ApprovalNetworkPolicySpec{Enabled: false}
		if np := BuildApprovalServerNetworkPolicy(cr); np != nil {
			t.Errorf("netpol disabled should produce nil NetworkPolicy; got %v", np)
		}
	})
}

func TestBuildApprovalServerNetworkPolicy_Shape(t *testing.T) {
	cr := approvalNetpolCR()
	np := BuildApprovalServerNetworkPolicy(cr)
	if np == nil {
		t.Fatal("netpol enabled must produce a NetworkPolicy")
	}

	// podSelector MUST match the Deployment's pod labels exactly.
	dep := BuildApprovalServerDeployment(cr)
	podLabels := dep.Spec.Template.Labels
	if len(np.Spec.PodSelector.MatchLabels) != len(podLabels) {
		t.Fatalf("podSelector %v does not match Deployment pod labels %v",
			np.Spec.PodSelector.MatchLabels, podLabels)
	}
	for k, v := range podLabels {
		if np.Spec.PodSelector.MatchLabels[k] != v {
			t.Errorf("podSelector missing/mismatched label %s=%s (got %q)", k, v, np.Spec.PodSelector.MatchLabels[k])
		}
	}

	// policyTypes: [Ingress]
	if len(np.Spec.PolicyTypes) != 1 || np.Spec.PolicyTypes[0] != networkingv1.PolicyTypeIngress {
		t.Errorf("policyTypes want [Ingress]; got %v", np.Spec.PolicyTypes)
	}

	// One ingress rule: from gateway namespaceSelector, port 8443/TCP.
	if len(np.Spec.Ingress) != 1 {
		t.Fatalf("want exactly 1 ingress rule; got %d", len(np.Spec.Ingress))
	}
	rule := np.Spec.Ingress[0]
	if len(rule.From) != 1 || rule.From[0].NamespaceSelector == nil {
		t.Fatalf("ingress.from must be a single namespaceSelector; got %+v", rule.From)
	}
	gotSel := rule.From[0].NamespaceSelector.MatchLabels
	if gotSel["kubernetes.io/metadata.name"] != "gateway" {
		t.Errorf("namespaceSelector want kubernetes.io/metadata.name=gateway; got %v", gotSel)
	}
	if len(rule.Ports) != 1 {
		t.Fatalf("want exactly 1 port; got %d", len(rule.Ports))
	}
	if rule.Ports[0].Port == nil || rule.Ports[0].Port.IntValue() != 8443 {
		t.Errorf("port want 8443; got %v", rule.Ports[0].Port)
	}
	if rule.Ports[0].Protocol == nil || *rule.Ports[0].Protocol != corev1.ProtocolTCP {
		t.Errorf("protocol want TCP; got %v", rule.Ports[0].Protocol)
	}

	if np.Namespace != cr.Namespace {
		t.Errorf("namespace want %s; got %s", cr.Namespace, np.Namespace)
	}
	if np.Name != ApprovalServerName(cr) {
		t.Errorf("name want %s; got %s", ApprovalServerName(cr), np.Name)
	}
}

// --- Reconcile ---

func TestReconcile_ApprovalNetworkPolicy_Created(t *testing.T) {
	cr := approvalNetpolCR()
	r, c := newReconciler(t, cr)
	reconcileOnce(t, r, cr)
	reconcileOnce(t, r, cr)

	var np networkingv1.NetworkPolicy
	if err := c.Get(context.Background(),
		types.NamespacedName{Namespace: "cha-system", Name: "bionic-approval-server"},
		&np); err != nil {
		t.Fatalf("approval NetworkPolicy not created: %v", err)
	}
	// podSelector must match the reconciled Deployment's pod labels.
	dep := BuildApprovalServerDeployment(cr)
	for k, v := range dep.Spec.Template.Labels {
		if np.Spec.PodSelector.MatchLabels[k] != v {
			t.Errorf("reconciled podSelector missing %s=%s", k, v)
		}
	}
	if np.Spec.Ingress[0].From[0].NamespaceSelector.MatchLabels["kubernetes.io/metadata.name"] != "gateway" {
		t.Errorf("reconciled namespaceSelector wrong: %v", np.Spec.Ingress[0].From[0].NamespaceSelector)
	}
}

func TestReconcile_ApprovalNetworkPolicy_DisabledNotCreated(t *testing.T) {
	cr := approvalCR() // approval on, netpol not configured
	r, c := newReconciler(t, cr)
	reconcileOnce(t, r, cr)
	reconcileOnce(t, r, cr)

	var np networkingv1.NetworkPolicy
	err := c.Get(context.Background(),
		types.NamespacedName{Namespace: "cha-system", Name: "bionic-approval-server"},
		&np)
	if !apierrors.IsNotFound(err) {
		t.Errorf("NetworkPolicy should not exist; got err=%v", err)
	}
}

func TestReconcile_ApprovalNetworkPolicy_MissingSelector_ReadyFalse(t *testing.T) {
	cr := approvalCR()
	cr.Spec.Approval.NetworkPolicy = &chav1alpha1.ApprovalNetworkPolicySpec{Enabled: true} // no selector
	r, c := newReconciler(t, cr)
	reconcileOnce(t, r, cr)

	cond := readReadyCondition(t, c, cr)
	if cond == nil || cond.Status != metav1.ConditionFalse || cond.Reason != "InvalidSpec" {
		t.Errorf("expected Ready=False/InvalidSpec for missing gatewayNamespaceSelector; got %+v", cond)
	}
}

func TestReconcile_ApprovalNetworkPolicy_DisabledAfterCreate_Deleted(t *testing.T) {
	cr := approvalNetpolCR()
	r, c := newReconciler(t, cr)
	reconcileOnce(t, r, cr)
	reconcileOnce(t, r, cr)

	// Flip netpol off.
	var stored chav1alpha1.ClusterHealthAutopilot
	_ = c.Get(context.Background(),
		types.NamespacedName{Namespace: "cha-system", Name: "bionic"}, &stored)
	stored.Spec.Approval.NetworkPolicy.Enabled = false
	stored.Generation = 2
	if err := c.Update(context.Background(), &stored); err != nil {
		t.Fatalf("update cr: %v", err)
	}
	reconcileOnce(t, r, &stored)

	var np networkingv1.NetworkPolicy
	err := c.Get(context.Background(),
		types.NamespacedName{Namespace: "cha-system", Name: "bionic-approval-server"},
		&np)
	if !apierrors.IsNotFound(err) {
		t.Errorf("NetworkPolicy not deleted after disable; got err=%v", err)
	}
}
