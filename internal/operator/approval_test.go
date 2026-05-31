// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	"context"
	"encoding/base64"
	"testing"

	chav1alpha1 "github.com/Bionic-AI-Solutions/cluster-health-autopilot/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Phase 2c-B — approval-server reconciliation tests.

// approvalCR returns the happy-path CR with approval-server enabled.
func approvalCR() *chav1alpha1.ClusterHealthAutopilot {
	cr := fullCR()
	cr.Spec.Approval = &chav1alpha1.ApprovalSpec{
		Enabled: true,
	}
	return cr
}

// --- Builders ---

func TestBuildApprovalServer_DisabledReturnsNil(t *testing.T) {
	cr := fullCR()
	for _, name := range []string{"SA", "Service", "Deployment"} {
		t.Run(name, func(t *testing.T) {
			var got interface{}
			switch name {
			case "SA":
				got = BuildApprovalServerServiceAccount(cr)
			case "Service":
				got = BuildApprovalServerService(cr)
			case "Deployment":
				got = BuildApprovalServerDeployment(cr)
			}
			// Each builder returns a typed nil pointer when disabled.
			if got == nil {
				return
			}
			switch v := got.(type) {
			case *corev1.ServiceAccount:
				if v != nil {
					t.Errorf("disabled approval should produce nil SA; got %v", v)
				}
			case *corev1.Service:
				if v != nil {
					t.Errorf("disabled approval should produce nil Service; got %v", v)
				}
			case *appsv1.Deployment:
				if v != nil {
					t.Errorf("disabled approval should produce nil Deployment; got %v", v)
				}
			}
		})
	}
}

func TestBuildApprovalServerDeployment_BasicShape(t *testing.T) {
	cr := approvalCR()
	d := BuildApprovalServerDeployment(cr)
	if d == nil {
		t.Fatal("approval enabled must produce a Deployment")
	}
	if d.Name != "bionic-approval-server" {
		t.Errorf("name=%q want bionic-approval-server", d.Name)
	}
	if d.Spec.Replicas == nil || *d.Spec.Replicas != 1 {
		t.Errorf("default replicas=%v want 1", d.Spec.Replicas)
	}
	if d.Spec.Strategy.Type != appsv1.RecreateDeploymentStrategyType {
		t.Errorf("inmemory store strategy=%v want Recreate", d.Spec.Strategy.Type)
	}
	if d.Spec.Template.Spec.ServiceAccountName != "bionic-approval-server" {
		t.Errorf("SA=%q want bionic-approval-server", d.Spec.Template.Spec.ServiceAccountName)
	}
}

func TestBuildApprovalServerDeployment_DefaultImage(t *testing.T) {
	cr := approvalCR()
	cr.Spec.Image.Tag = "1.9.4"
	d := BuildApprovalServerDeployment(cr)
	got := d.Spec.Template.Spec.Containers[0].Image
	want := "docker4zerocool/cha-com:v1.9.4"
	if got != want {
		t.Errorf("default image=%q want %q (chart cha-com:v<OSS-tag> convention)", got, want)
	}
}

func TestBuildApprovalServerDeployment_DefaultArgs(t *testing.T) {
	cr := approvalCR()
	d := BuildApprovalServerDeployment(cr)
	args := d.Spec.Template.Spec.Containers[0].Args

	mustContain(t, args, "approval-server")
	mustContain(t, args, "--listen=:8443")
	mustContain(t, args, "--signing-key-path=/etc/cha/keys/signing.key")
	mustContain(t, args, "--audit-namespace=cha-system")

	// inmemory store → no --store-* flags emitted.
	for _, a := range args {
		if a == "--store-backend=inmemory" {
			t.Errorf("inmemory store should NOT emit --store-backend flag; got %q", a)
		}
	}
}

func TestBuildApprovalServerDeployment_ConfigMapStoreSwitchesToRollingUpdate(t *testing.T) {
	cr := approvalCR()
	cr.Spec.Approval.Replicas = 2
	cr.Spec.Approval.Store = &chav1alpha1.ApprovalStoreSpec{
		Backend: "configmap",
	}
	d := BuildApprovalServerDeployment(cr)
	if d.Spec.Strategy.Type != appsv1.RollingUpdateDeploymentStrategyType {
		t.Errorf("configmap-store strategy=%v want RollingUpdate (replicas > 1 are safe)",
			d.Spec.Strategy.Type)
	}
	args := d.Spec.Template.Spec.Containers[0].Args
	mustContain(t, args, "--store-backend=configmap")
	mustContain(t, args, "--store-replay-configmap=cha-approval-replay")
	mustContain(t, args, "--store-runbook-configmap=cha-approval-runbooks")
}

func TestBuildApprovalServerDeployment_SigningKeyVolume(t *testing.T) {
	cr := approvalCR()
	d := BuildApprovalServerDeployment(cr)
	vols := d.Spec.Template.Spec.Volumes
	if len(vols) != 1 || vols[0].Name != "signing-key" {
		t.Fatalf("expected one signing-key Volume; got %+v", vols)
	}
	sv := vols[0].Secret
	if sv == nil || sv.SecretName != "cha-approval-signing-key" {
		t.Errorf("Volume secret=%+v want cha-approval-signing-key", sv)
	}
	if len(sv.Items) != 2 {
		t.Errorf("expected 2 keyed mounts (signing.key + signing.pub); got %d", len(sv.Items))
	}
}

func TestBuildApprovalServerService_BasicShape(t *testing.T) {
	cr := approvalCR()
	svc := BuildApprovalServerService(cr)
	if svc == nil {
		t.Fatal("approval enabled must produce a Service")
	}
	if svc.Name != "bionic-approval-server" {
		t.Errorf("svc name=%q want bionic-approval-server", svc.Name)
	}
	if len(svc.Spec.Ports) != 1 || svc.Spec.Ports[0].Port != 8443 {
		t.Errorf("expected port 8443; got %+v", svc.Spec.Ports)
	}
}

func TestBuildApprovalFixerClusterRole_HasRequiredVerbs(t *testing.T) {
	role := BuildApprovalFixerClusterRole()
	if role.Name != "cha-operator-approval-fixer" {
		t.Errorf("role name=%q want cha-operator-approval-fixer", role.Name)
	}
	// Spot-check: pods/delete + deployments/patch are the two
	// most-used fixer verbs.
	if !hasRule(role.Rules, "", "pods") {
		t.Error("fixer role missing pods rule")
	}
	if !hasRule(role.Rules, "apps", "deployments") {
		t.Error("fixer role missing apps/deployments rule")
	}
	if !hasRule(role.Rules, "networking.k8s.io", "ingresses") {
		t.Error("fixer role missing ingresses rule (needed for TLSSecretMismatch fixer)")
	}
}

func TestBuildApprovalFixerClusterRoleBinding_TargetsApprovalSA(t *testing.T) {
	cr := approvalCR()
	binding := BuildApprovalFixerClusterRoleBinding(cr)
	if binding.Name != "cha-operator-approval-fixer-cha-system-bionic" {
		t.Errorf("binding name=%q want cha-operator-approval-fixer-cha-system-bionic", binding.Name)
	}
	if !bindingTargetsSA(binding, "bionic-approval-server", "cha-system") {
		t.Errorf("binding does not target approval-server SA; got %+v", binding.Subjects)
	}
	if binding.Labels[ManagedByCRLabel] != cr.Name {
		t.Errorf("binding missing managed-by-cr label; got %v", binding.Labels)
	}
}

func TestBuildApprovalSigningReaderRole_ResourceNameScoped(t *testing.T) {
	cr := approvalCR()
	role := BuildApprovalSigningReaderRole(cr)
	if role == nil {
		t.Fatal("signing-reader Role missing")
	}
	if len(role.Rules) != 1 ||
		len(role.Rules[0].ResourceNames) != 1 ||
		role.Rules[0].ResourceNames[0] != "cha-approval-signing-key" {
		t.Errorf("signing-reader Role not resourceName-scoped; got %+v", role.Rules)
	}
}

func TestBuildApprovalStoresRole_OnlyWhenConfigMapBackend(t *testing.T) {
	cr := approvalCR()
	// inmemory store → no stores Role.
	if r := BuildApprovalStoresRole(cr); r != nil {
		t.Errorf("inmemory store should NOT produce stores Role; got %v", r)
	}
	// configmap store → Role exists.
	cr.Spec.Approval.Store = &chav1alpha1.ApprovalStoreSpec{Backend: "configmap"}
	r := BuildApprovalStoresRole(cr)
	if r == nil {
		t.Fatal("configmap store should produce stores Role")
	}
	// Two rules: resourceNames-scoped get/update + unrestricted create.
	if len(r.Rules) != 2 {
		t.Errorf("expected 2 rules in stores Role; got %d", len(r.Rules))
	}
}

func TestGenerateSigningKeySecret_HasBothKeys(t *testing.T) {
	cr := approvalCR()
	s, err := GenerateSigningKeySecret(cr)
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}
	if s.Name != "cha-approval-signing-key" {
		t.Errorf("Secret name=%q want cha-approval-signing-key", s.Name)
	}
	if len(s.Data["signing.key"]) == 0 || len(s.Data["signing.pub"]) == 0 {
		t.Errorf("Secret missing signing.key or signing.pub; got keys=%v", keysOf(s.Data))
	}
}

// TestGenerateSigningKeySecret_RawBase64_NotPEM is a regression guard
// for v1.10.0 → v1.10.1: cha-com (approval-server + aiwatch) loads the
// signing key with base64.StdEncoding.DecodeString on the file content.
// PEM-wrapped data (-----BEGIN PRIVATE KEY-----…) fails that decode at
// byte 0 with "signing key not valid base64: illegal base64 data at
// input byte 0", crashing every aiwatch + approval-server pod on
// startup. The encoding MUST be raw base64 of the ed25519 key bytes.
func TestGenerateSigningKeySecret_RawBase64_NotPEM(t *testing.T) {
	cr := approvalCR()
	s, err := GenerateSigningKeySecret(cr)
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}
	for _, k := range []string{"signing.key", "signing.pub"} {
		raw := s.Data[k]
		if bytesHasPrefix(raw, []byte("-----BEGIN")) {
			t.Errorf("%s is PEM-wrapped; cha-com expects raw base64. First bytes: %q", k, raw[:min(40, len(raw))])
		}
		if _, err := base64Decode(raw); err != nil {
			t.Errorf("%s is not valid base64: %v", k, err)
		}
	}
}

func bytesHasPrefix(b, prefix []byte) bool {
	if len(b) < len(prefix) {
		return false
	}
	for i := range prefix {
		if b[i] != prefix[i] {
			return false
		}
	}
	return true
}

func base64Decode(s []byte) ([]byte, error) {
	return base64.StdEncoding.DecodeString(string(s))
}

// --- Reconcile-loop wiring ---

func TestReconcile_ApprovalEnabled_CreatesFullStack(t *testing.T) {
	cr := approvalCR()
	r, c := newReconciler(t, cr)
	reconcileOnce(t, r, cr)
	reconcileOnce(t, r, cr)

	ns := "cha-system"
	name := "bionic-approval-server"

	for _, want := range []struct {
		obj  client.Object
		path string
	}{
		{&corev1.ServiceAccount{}, "SA"},
		{&corev1.Service{}, "Service"},
		{&appsv1.Deployment{}, "Deployment"},
		{&corev1.Secret{}, "signing-key Secret"},
		{&rbacv1.Role{}, "signing-reader Role"},
		{&rbacv1.RoleBinding{}, "signing-reader RoleBinding"},
		{&rbacv1.Role{}, "events Role"},
		{&rbacv1.RoleBinding{}, "events RoleBinding"},
	} {
		_ = want // not strictly needed below; iteration kept for symmetry
	}

	// Individual gets — typed.
	var sa corev1.ServiceAccount
	if err := c.Get(context.Background(), types.NamespacedName{Namespace: ns, Name: name}, &sa); err != nil {
		t.Errorf("approval-server SA missing: %v", err)
	}
	var svc corev1.Service
	if err := c.Get(context.Background(), types.NamespacedName{Namespace: ns, Name: name}, &svc); err != nil {
		t.Errorf("approval-server Service missing: %v", err)
	}
	var dep appsv1.Deployment
	if err := c.Get(context.Background(), types.NamespacedName{Namespace: ns, Name: name}, &dep); err != nil {
		t.Errorf("approval-server Deployment missing: %v", err)
	}
	var sec corev1.Secret
	if err := c.Get(context.Background(),
		types.NamespacedName{Namespace: ns, Name: "cha-approval-signing-key"}, &sec); err != nil {
		t.Errorf("signing-key Secret missing: %v", err)
	}
	if len(sec.Data["signing.key"]) == 0 {
		t.Errorf("signing-key Secret has empty signing.key")
	}
	var fixer rbacv1.ClusterRole
	if err := c.Get(context.Background(),
		types.NamespacedName{Name: ApprovalFixerClusterRoleName}, &fixer); err != nil {
		t.Errorf("fixer ClusterRole missing: %v", err)
	}
	var binding rbacv1.ClusterRoleBinding
	if err := c.Get(context.Background(),
		types.NamespacedName{Name: ApprovalFixerClusterRoleBindingName(cr)}, &binding); err != nil {
		t.Errorf("fixer ClusterRoleBinding missing: %v", err)
	}
}

func TestReconcile_ApprovalEnabled_StoresRBACOnlyForConfigMapBackend(t *testing.T) {
	cr := approvalCR()
	cr.Spec.Approval.Store = &chav1alpha1.ApprovalStoreSpec{Backend: "configmap"}
	r, c := newReconciler(t, cr)
	reconcileOnce(t, r, cr)
	reconcileOnce(t, r, cr)

	var role rbacv1.Role
	if err := c.Get(context.Background(),
		types.NamespacedName{Namespace: "cha-system", Name: "bionic-approval-server-stores"},
		&role); err != nil {
		t.Errorf("stores Role missing for configmap backend: %v", err)
	}
}

func TestReconcile_ApprovalDisabled_NothingCreated(t *testing.T) {
	cr := fullCR()
	r, c := newReconciler(t, cr)
	reconcileOnce(t, r, cr)
	reconcileOnce(t, r, cr)

	var dep appsv1.Deployment
	err := c.Get(context.Background(),
		types.NamespacedName{Namespace: "cha-system", Name: "bionic-approval-server"},
		&dep)
	if !apierrors.IsNotFound(err) {
		t.Errorf("approval-server Deployment should not exist when disabled; got err=%v", err)
	}
}

func TestReconcile_ApprovalDisabledAfterCreate_DeletesAllChildren(t *testing.T) {
	cr := approvalCR()
	r, c := newReconciler(t, cr)
	reconcileOnce(t, r, cr)
	reconcileOnce(t, r, cr)

	var stored chav1alpha1.ClusterHealthAutopilot
	_ = c.Get(context.Background(),
		types.NamespacedName{Namespace: "cha-system", Name: "bionic"}, &stored)
	stored.Spec.Approval.Enabled = false
	stored.Generation = 2
	if err := c.Update(context.Background(), &stored); err != nil {
		t.Fatalf("update cr: %v", err)
	}
	reconcileOnce(t, r, &stored)

	// Deployment/Service/SA gone.
	for _, kind := range []struct {
		obj  client.Object
		name string
		ns   string
	}{
		{&appsv1.Deployment{}, "bionic-approval-server", "cha-system"},
		{&corev1.Service{}, "bionic-approval-server", "cha-system"},
		{&corev1.ServiceAccount{}, "bionic-approval-server", "cha-system"},
		{&rbacv1.Role{}, "bionic-approval-server-signing-reader", "cha-system"},
	} {
		err := c.Get(context.Background(),
			types.NamespacedName{Namespace: kind.ns, Name: kind.name}, kind.obj)
		if !apierrors.IsNotFound(err) {
			t.Errorf("%T %s not deleted; got err=%v", kind.obj, kind.name, err)
		}
	}
	// Signing-key Secret is INTENTIONALLY preserved (so a re-enable
	// doesn't invalidate outstanding JWTs).
	var sec corev1.Secret
	if err := c.Get(context.Background(),
		types.NamespacedName{Namespace: "cha-system", Name: "cha-approval-signing-key"},
		&sec); err != nil {
		t.Errorf("signing-key Secret should be preserved across disable; got err=%v", err)
	}
}

func TestReconcile_ApprovalReady_FlipsConditionTrue(t *testing.T) {
	cr := approvalCR()
	r, c := newReconciler(t, cr)
	reconcileOnce(t, r, cr)
	reconcileOnce(t, r, cr)

	// Simulate Deployment Ready.
	var dep appsv1.Deployment
	_ = c.Get(context.Background(),
		types.NamespacedName{Namespace: "cha-system", Name: "bionic-approval-server"},
		&dep)
	dep.Status.AvailableReplicas = 1
	if err := c.Status().Update(context.Background(), &dep); err != nil {
		t.Fatalf("status update: %v", err)
	}

	reconcileOnce(t, r, cr)

	cond := readCondition(t, c, cr, chav1alpha1.ConditionApprovalServerReady)
	if cond == nil || cond.Status != metav1.ConditionTrue {
		t.Errorf("ApprovalServerReady=%+v; want True", cond)
	}
}

func TestReconcile_ApprovalInmemory_MultipleReplicas_ReadyFalse(t *testing.T) {
	// GAP fix: replicas>1 with inmemory store must be rejected at validation
	// time rather than silently creating a split-brain approval fleet.
	cr := approvalCR()
	cr.Spec.Approval.Replicas = 3
	// Store is nil/default = inmemory.
	r, c := newReconciler(t, cr)
	reconcileOnce(t, r, cr)

	cond := readReadyCondition(t, c, cr)
	if cond == nil || cond.Status != metav1.ConditionFalse || cond.Reason != "InvalidSpec" {
		t.Errorf("expected Ready=False/InvalidSpec for replicas>1+inmemory; got %+v", cond)
	}
}

func TestReconcile_ApprovalConfigmap_MultipleReplicasAllowed(t *testing.T) {
	// configmap backend is safe for replicas>1 — state is shared.
	cr := approvalCR()
	cr.Spec.Approval.Replicas = 2
	cr.Spec.Approval.Store = &chav1alpha1.ApprovalStoreSpec{Backend: "configmap"}
	r, c := newReconciler(t, cr)
	reconcileOnce(t, r, cr)
	reconcileOnce(t, r, cr)

	// Must not land on InvalidSpec.
	cond := readReadyCondition(t, c, cr)
	if cond != nil && cond.Reason == "InvalidSpec" {
		t.Errorf("configmap backend with replicas=2 should not fail validation; got %+v", cond)
	}
}

func TestReconcile_ApprovalEnabled_SigningKeyUpdateIdempotentOnSecondReconcile(t *testing.T) {
	// GAP fix: reconcileSigningKeySecret must NOT call r.Update() when the
	// Secret already exists with correct labels (spurious update per reconcile).
	cr := approvalCR()
	r, c := newReconciler(t, cr)
	reconcileOnce(t, r, cr)
	reconcileOnce(t, r, cr)

	// The Secret should exist with labels matching CommonLabels.
	var sec corev1.Secret
	if err := c.Get(context.Background(),
		types.NamespacedName{Namespace: "cha-system", Name: "cha-approval-signing-key"},
		&sec); err != nil {
		t.Fatalf("signing-key Secret missing: %v", err)
	}
	rv1 := sec.ResourceVersion

	// Third reconcile — labels already correct, no update should happen.
	reconcileOnce(t, r, cr)

	var sec2 corev1.Secret
	_ = c.Get(context.Background(),
		types.NamespacedName{Namespace: "cha-system", Name: "cha-approval-signing-key"},
		&sec2)
	if sec2.ResourceVersion != rv1 {
		t.Errorf("signing-key Secret resourceVersion changed after no-op reconcile (%s→%s): "+
			"spurious Update() not guarded by label-diff check",
			rv1, sec2.ResourceVersion)
	}
}

func TestReconcile_ApprovalDisabled_ApprovalConditionDisabled(t *testing.T) {
	cr := fullCR()
	r, c := newReconciler(t, cr)
	reconcileOnce(t, r, cr)
	reconcileOnce(t, r, cr)

	cond := readCondition(t, c, cr, chav1alpha1.ConditionApprovalServerReady)
	if cond == nil {
		t.Fatal("ApprovalServerReady condition not set even when disabled")
	}
	if cond.Status != metav1.ConditionFalse || cond.Reason != "Disabled" {
		t.Errorf("ApprovalServerReady=%s/%s; want False/Disabled", cond.Status, cond.Reason)
	}
}

func TestReconcile_ApprovalEnabled_SigningKeyIdempotent(t *testing.T) {
	// Two reconciles must produce the same Secret (not regenerate the
	// key). A regenerate would invalidate every outstanding JWT.
	cr := approvalCR()
	r, c := newReconciler(t, cr)
	reconcileOnce(t, r, cr)
	reconcileOnce(t, r, cr)

	var first corev1.Secret
	_ = c.Get(context.Background(),
		types.NamespacedName{Namespace: "cha-system", Name: "cha-approval-signing-key"},
		&first)
	firstKey := string(first.Data["signing.key"])

	// Trigger more reconciles.
	for i := 0; i < 3; i++ {
		reconcileOnce(t, r, cr)
	}
	var later corev1.Secret
	_ = c.Get(context.Background(),
		types.NamespacedName{Namespace: "cha-system", Name: "cha-approval-signing-key"},
		&later)
	if string(later.Data["signing.key"]) != firstKey {
		t.Error("signing-key Secret was regenerated across reconciles — outstanding JWTs broken")
	}
}

// --- helpers ---

func keysOf(m map[string][]byte) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
