// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	"context"
	"testing"

	chav1alpha1 "github.com/Bionic-AI-Solutions/cluster-health-autopilot/api/v1alpha1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// Phase 2c-C — optional Ingress for the approval-server.

// approvalIngressCR returns the happy-path CR with approval + ingress
// both enabled and the required host set.
func approvalIngressCR() *chav1alpha1.ClusterHealthAutopilot {
	cr := approvalCR()
	cr.Spec.Approval.Ingress = &chav1alpha1.ApprovalIngressSpec{
		Enabled: true,
		Host:    "approve.cha.example.com",
	}
	return cr
}

// --- Builder ---

func TestBuildApprovalServerIngress_DisabledReturnsNil(t *testing.T) {
	t.Run("approval off", func(t *testing.T) {
		cr := fullCR()
		if ing := BuildApprovalServerIngress(cr); ing != nil {
			t.Errorf("approval off should produce nil Ingress; got %v", ing)
		}
	})
	t.Run("approval on, ingress off", func(t *testing.T) {
		cr := approvalCR()
		if ing := BuildApprovalServerIngress(cr); ing != nil {
			t.Errorf("ingress off should produce nil Ingress; got %v", ing)
		}
	})
	t.Run("ingress object present, enabled=false", func(t *testing.T) {
		cr := approvalCR()
		cr.Spec.Approval.Ingress = &chav1alpha1.ApprovalIngressSpec{Enabled: false}
		if ing := BuildApprovalServerIngress(cr); ing != nil {
			t.Errorf("ingress.enabled=false should produce nil Ingress; got %v", ing)
		}
	})
}

func TestBuildApprovalServerIngress_BasicShape(t *testing.T) {
	cr := approvalIngressCR()
	ing := BuildApprovalServerIngress(cr)
	if ing == nil {
		t.Fatal("ingress enabled must produce an Ingress")
	}
	if ing.Name != "bionic-approval-server" {
		t.Errorf("ingress name=%q want bionic-approval-server", ing.Name)
	}
	if len(ing.Spec.Rules) != 1 {
		t.Fatalf("expected 1 rule; got %d", len(ing.Spec.Rules))
	}
	if ing.Spec.Rules[0].Host != "approve.cha.example.com" {
		t.Errorf("host=%q want approve.cha.example.com", ing.Spec.Rules[0].Host)
	}
	if ing.Spec.Rules[0].HTTP == nil || len(ing.Spec.Rules[0].HTTP.Paths) != 3 {
		t.Fatalf("expected 3 paths (/approve + /deny + /healthz); got %+v", ing.Spec.Rules[0].HTTP)
	}
	wantPaths := map[string]bool{"/approve": false, "/deny": false, "/healthz": false}
	for _, p := range ing.Spec.Rules[0].HTTP.Paths {
		if _, ok := wantPaths[p.Path]; ok {
			wantPaths[p.Path] = true
		}
		if p.Backend.Service == nil ||
			p.Backend.Service.Name != "bionic-approval-server" ||
			p.Backend.Service.Port.Name != "http" {
			t.Errorf("path %s backend mis-wired: %+v", p.Path, p.Backend)
		}
	}
	for path, ok := range wantPaths {
		if !ok {
			t.Errorf("path %s missing from Ingress", path)
		}
	}
}

func TestBuildApprovalServerIngress_IngressClassNameHonored(t *testing.T) {
	cr := approvalIngressCR()
	cr.Spec.Approval.Ingress.IngressClassName = "kong"
	ing := BuildApprovalServerIngress(cr)
	if ing.Spec.IngressClassName == nil || *ing.Spec.IngressClassName != "kong" {
		t.Errorf("ingressClassName=%v want kong", ing.Spec.IngressClassName)
	}
}

func TestBuildApprovalServerIngress_AnnotationsPassThrough(t *testing.T) {
	cr := approvalIngressCR()
	cr.Spec.Approval.Ingress.Annotations = map[string]string{
		"cert-manager.io/cluster-issuer":       "letsencrypt-prod",
		"konghq.com/strip-path":                "true",
		"nginx.ingress.kubernetes.io/auth-url": "https://oauth2.example/auth",
	}
	ing := BuildApprovalServerIngress(cr)
	for k, v := range cr.Spec.Approval.Ingress.Annotations {
		if got := ing.Annotations[k]; got != v {
			t.Errorf("annotation[%s]=%q want %q", k, got, v)
		}
	}
}

func TestBuildApprovalServerIngress_TLSDefaultSecretName(t *testing.T) {
	cr := approvalIngressCR()
	cr.Spec.Approval.Ingress.TLS = &chav1alpha1.ApprovalIngressTLSSpec{
		Enabled: true,
		// SecretName omitted — default to `<name>-tls`.
	}
	ing := BuildApprovalServerIngress(cr)
	if len(ing.Spec.TLS) != 1 {
		t.Fatalf("expected 1 TLS block; got %d", len(ing.Spec.TLS))
	}
	if ing.Spec.TLS[0].SecretName != "bionic-approval-server-tls" {
		t.Errorf("default TLS secretName=%q want bionic-approval-server-tls",
			ing.Spec.TLS[0].SecretName)
	}
	if len(ing.Spec.TLS[0].Hosts) != 1 || ing.Spec.TLS[0].Hosts[0] != "approve.cha.example.com" {
		t.Errorf("TLS hosts=%v want [approve.cha.example.com]", ing.Spec.TLS[0].Hosts)
	}
}

func TestBuildApprovalServerIngress_TLSExplicitSecretName(t *testing.T) {
	cr := approvalIngressCR()
	cr.Spec.Approval.Ingress.TLS = &chav1alpha1.ApprovalIngressTLSSpec{
		Enabled:    true,
		SecretName: "my-pre-provisioned-cert",
	}
	ing := BuildApprovalServerIngress(cr)
	if ing.Spec.TLS[0].SecretName != "my-pre-provisioned-cert" {
		t.Errorf("explicit TLS secretName not honored; got %q", ing.Spec.TLS[0].SecretName)
	}
}

func TestBuildApprovalServerIngress_TLSDisabledSkipsBlock(t *testing.T) {
	cr := approvalIngressCR()
	cr.Spec.Approval.Ingress.TLS = &chav1alpha1.ApprovalIngressTLSSpec{Enabled: false}
	ing := BuildApprovalServerIngress(cr)
	if len(ing.Spec.TLS) != 0 {
		t.Errorf("TLS disabled should produce no Spec.TLS; got %+v", ing.Spec.TLS)
	}
}

// --- Reconcile-loop wiring ---

func TestReconcile_ApprovalIngress_Created(t *testing.T) {
	cr := approvalIngressCR()
	r, c := newReconciler(t, cr)
	reconcileOnce(t, r, cr)
	reconcileOnce(t, r, cr)

	var ing networkingv1.Ingress
	if err := c.Get(context.Background(),
		types.NamespacedName{Namespace: "cha-system", Name: "bionic-approval-server"},
		&ing); err != nil {
		t.Errorf("approval Ingress not created: %v", err)
	}
}

func TestReconcile_ApprovalIngress_DisabledNotCreated(t *testing.T) {
	cr := approvalCR() // approval on, ingress not configured
	r, c := newReconciler(t, cr)
	reconcileOnce(t, r, cr)
	reconcileOnce(t, r, cr)

	var ing networkingv1.Ingress
	err := c.Get(context.Background(),
		types.NamespacedName{Namespace: "cha-system", Name: "bionic-approval-server"},
		&ing)
	if !apierrors.IsNotFound(err) {
		t.Errorf("ingress should not exist; got err=%v", err)
	}
}

func TestReconcile_ApprovalIngress_MissingHost_ReadyFalse(t *testing.T) {
	cr := approvalCR()
	cr.Spec.Approval.Ingress = &chav1alpha1.ApprovalIngressSpec{Enabled: true} // no Host
	r, c := newReconciler(t, cr)
	reconcileOnce(t, r, cr)

	cond := readReadyCondition(t, c, cr)
	if cond == nil || cond.Status != metav1.ConditionFalse || cond.Reason != "InvalidSpec" {
		t.Errorf("expected Ready=False/InvalidSpec for missing ingress.host; got %+v", cond)
	}
}

func TestReconcile_ApprovalIngress_DisabledAfterCreate_Deleted(t *testing.T) {
	cr := approvalIngressCR()
	r, c := newReconciler(t, cr)
	reconcileOnce(t, r, cr)
	reconcileOnce(t, r, cr)

	// Flip ingress off.
	var stored chav1alpha1.ClusterHealthAutopilot
	_ = c.Get(context.Background(),
		types.NamespacedName{Namespace: "cha-system", Name: "bionic"}, &stored)
	stored.Spec.Approval.Ingress.Enabled = false
	stored.Generation = 2
	if err := c.Update(context.Background(), &stored); err != nil {
		t.Fatalf("update cr: %v", err)
	}
	reconcileOnce(t, r, &stored)

	var ing networkingv1.Ingress
	err := c.Get(context.Background(),
		types.NamespacedName{Namespace: "cha-system", Name: "bionic-approval-server"},
		&ing)
	if !apierrors.IsNotFound(err) {
		t.Errorf("ingress not deleted after disable; got err=%v", err)
	}
}
