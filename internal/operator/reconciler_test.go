// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	"context"
	"testing"

	chav1alpha1 "github.com/srenix-ai/agentic-sre/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// testScheme builds the scheme the controller-runtime fake client
// needs to know how to encode our types.
func testScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(s))
	utilruntime.Must(chav1alpha1.AddToScheme(s))
	return s
}

// newReconciler stitches together a fake client + scheme + Reconciler.
// Each test gets its own isolated cluster state.
func newReconciler(t *testing.T, initObjects ...client.Object) (*Reconciler, client.Client) {
	t.Helper()
	s := testScheme(t)
	c := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(initObjects...).
		WithStatusSubresource(&chav1alpha1.AgenticSRE{}).
		Build()
	return &Reconciler{Client: c, Scheme: s}, c
}

// fullCR returns a CR with watcher + diagnose enabled and minimal
// image config — the happy-path shape.
func fullCR() *chav1alpha1.AgenticSRE {
	return &chav1alpha1.AgenticSRE{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "bionic",
			Namespace:  "srenix-system",
			Generation: 1,
		},
		Spec: chav1alpha1.AgenticSRESpec{
			Image: chav1alpha1.ImageSpec{
				Repository: "docker4zerocool/agentic-sre",
				Tag:        "1.8.0",
			},
			Watcher:  &chav1alpha1.WatcherSpec{Enabled: true},
			Diagnose: &chav1alpha1.DiagnoseSpec{Enabled: true},
		},
	}
}

func reconcileOnce(t *testing.T, r *Reconciler, cr *chav1alpha1.AgenticSRE) {
	t.Helper()
	if _, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace},
	}); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
}

// --- Happy path: create-all-subresources ---

func TestReconcile_CreatesAllSubresources(t *testing.T) {
	cr := fullCR()
	r, c := newReconciler(t, cr)
	reconcileOnce(t, r, cr)

	// ServiceAccount.
	var sa corev1.ServiceAccount
	if err := c.Get(context.Background(),
		types.NamespacedName{Namespace: "srenix-system", Name: "bionic-sa"},
		&sa); err != nil {
		t.Errorf("SA not created: %v", err)
	}

	// Watcher Deployment.
	var dep appsv1.Deployment
	if err := c.Get(context.Background(),
		types.NamespacedName{Namespace: "srenix-system", Name: "bionic-watcher"},
		&dep); err != nil {
		t.Errorf("watcher Deployment not created: %v", err)
	}

	// Diagnose CronJob.
	var cj batchv1.CronJob
	if err := c.Get(context.Background(),
		types.NamespacedName{Namespace: "srenix-system", Name: "bionic-diagnose"},
		&cj); err != nil {
		t.Errorf("diagnose CronJob not created: %v", err)
	}
}

// --- Owner references ---

func TestReconcile_SetsControllerOwnerRefOnChildren(t *testing.T) {
	cr := fullCR()
	r, c := newReconciler(t, cr)
	reconcileOnce(t, r, cr)

	var dep appsv1.Deployment
	if err := c.Get(context.Background(),
		types.NamespacedName{Namespace: "srenix-system", Name: "bionic-watcher"},
		&dep); err != nil {
		t.Fatalf("get deployment: %v", err)
	}
	if len(dep.OwnerReferences) == 0 {
		t.Fatalf("watcher Deployment has no owner refs")
	}
	owner := dep.OwnerReferences[0]
	if owner.Kind != "AgenticSRE" || owner.Name != "bionic" {
		t.Errorf("owner=%+v; want Kind=AgenticSRE Name=bionic", owner)
	}
	if owner.Controller == nil || !*owner.Controller {
		t.Errorf("controller flag not set; got %+v", owner)
	}
}

// --- Status conditions ---

func TestReconcile_SetsReadyAndWatcherRunningConditions(t *testing.T) {
	cr := fullCR()
	r, c := newReconciler(t, cr)
	reconcileOnce(t, r, cr)

	var got chav1alpha1.AgenticSRE
	if err := c.Get(context.Background(),
		types.NamespacedName{Namespace: "srenix-system", Name: "bionic"},
		&got); err != nil {
		t.Fatalf("get cr: %v", err)
	}
	if got.Status.ObservedGeneration != 1 {
		t.Errorf("observedGeneration=%d want 1", got.Status.ObservedGeneration)
	}
	want := map[string]metav1.ConditionStatus{
		chav1alpha1.ConditionReady:          metav1.ConditionTrue,
		chav1alpha1.ConditionWatcherRunning: metav1.ConditionFalse, // fake client doesn't run the dep
	}
	for cType, expected := range want {
		found := false
		for _, c := range got.Status.Conditions {
			if c.Type == cType {
				found = true
				if c.Status != expected {
					t.Errorf("condition %s=%s want %s (reason=%s msg=%s)",
						cType, c.Status, expected, c.Reason, c.Message)
				}
			}
		}
		if !found {
			t.Errorf("condition %s not set", cType)
		}
	}
}

func TestReconcile_WatcherDeploymentAvailable_ReportsRunningTrue(t *testing.T) {
	cr := fullCR()
	r, c := newReconciler(t, cr)

	// First reconcile creates the Deployment with status.availableReplicas=0.
	reconcileOnce(t, r, cr)

	// Simulate the deployment going Ready.
	var dep appsv1.Deployment
	if err := c.Get(context.Background(),
		types.NamespacedName{Namespace: "srenix-system", Name: "bionic-watcher"},
		&dep); err != nil {
		t.Fatalf("get deployment: %v", err)
	}
	dep.Status.AvailableReplicas = 1
	if err := c.Status().Update(context.Background(), &dep); err != nil {
		t.Fatalf("update deployment status: %v", err)
	}

	// Reconcile again — status should flip.
	reconcileOnce(t, r, cr)

	var got chav1alpha1.AgenticSRE
	_ = c.Get(context.Background(),
		types.NamespacedName{Namespace: "srenix-system", Name: "bionic"},
		&got)
	for _, c := range got.Status.Conditions {
		if c.Type == chav1alpha1.ConditionWatcherRunning {
			if c.Status != metav1.ConditionTrue {
				t.Errorf("WatcherRunning=%s want True (reason=%s msg=%s)", c.Status, c.Reason, c.Message)
			}
			return
		}
	}
	t.Errorf("WatcherRunning condition not set")
}

// --- Spec.Watcher.Enabled=false ---

func TestReconcile_WatcherDisabled_DoesNotCreateDeployment(t *testing.T) {
	cr := fullCR()
	cr.Spec.Watcher.Enabled = false
	r, c := newReconciler(t, cr)
	reconcileOnce(t, r, cr)

	var dep appsv1.Deployment
	err := c.Get(context.Background(),
		types.NamespacedName{Namespace: "srenix-system", Name: "bionic-watcher"},
		&dep)
	if !apierrors.IsNotFound(err) {
		t.Errorf("expected NotFound for disabled watcher Deployment; got err=%v dep=%+v", err, dep)
	}
}

func TestReconcile_WatcherDisabledAfterCreate_DeletesDeployment(t *testing.T) {
	cr := fullCR()
	r, c := newReconciler(t, cr)
	reconcileOnce(t, r, cr) // creates the Deployment.

	// Flip to disabled and reconcile again.
	var stored chav1alpha1.AgenticSRE
	_ = c.Get(context.Background(),
		types.NamespacedName{Namespace: "srenix-system", Name: "bionic"},
		&stored)
	stored.Spec.Watcher.Enabled = false
	stored.Generation = 2
	if err := c.Update(context.Background(), &stored); err != nil {
		t.Fatalf("update cr: %v", err)
	}
	reconcileOnce(t, r, &stored)

	var dep appsv1.Deployment
	err := c.Get(context.Background(),
		types.NamespacedName{Namespace: "srenix-system", Name: "bionic-watcher"},
		&dep)
	if !apierrors.IsNotFound(err) {
		t.Errorf("watcher Deployment not deleted after disable; got err=%v dep=%+v", err, dep)
	}
}

// --- Validation ---

func TestReconcile_EmptyImageTag_SetsReadyFalse(t *testing.T) {
	cr := fullCR()
	cr.Spec.Image.Tag = ""
	r, c := newReconciler(t, cr)
	reconcileOnce(t, r, cr)

	var got chav1alpha1.AgenticSRE
	_ = c.Get(context.Background(),
		types.NamespacedName{Namespace: "srenix-system", Name: "bionic"},
		&got)
	for _, c := range got.Status.Conditions {
		if c.Type == chav1alpha1.ConditionReady {
			if c.Status != metav1.ConditionFalse {
				t.Errorf("Ready=%s want False on empty image.tag", c.Status)
			}
			if c.Reason != "InvalidSpec" {
				t.Errorf("Ready.reason=%s want InvalidSpec", c.Reason)
			}
			return
		}
	}
	t.Errorf("Ready condition not set")
}

// --- NotFound CR ---

func TestReconcile_CRNotFound_NoError(t *testing.T) {
	r, _ := newReconciler(t)
	if _, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "missing", Namespace: "srenix-system"},
	}); err != nil {
		t.Errorf("post-delete reconcile should be silent; got: %v", err)
	}
}

// --- Update path (existing Deployment, spec changed) ---

func TestReconcile_UpdatesExistingDeploymentSpec(t *testing.T) {
	cr := fullCR()
	r, c := newReconciler(t, cr)
	reconcileOnce(t, r, cr) // create.

	// Change replicas in the CR and reconcile.
	var stored chav1alpha1.AgenticSRE
	_ = c.Get(context.Background(),
		types.NamespacedName{Namespace: "srenix-system", Name: "bionic"},
		&stored)
	stored.Spec.Watcher.Replicas = 3
	stored.Generation = 2
	if err := c.Update(context.Background(), &stored); err != nil {
		t.Fatalf("update cr: %v", err)
	}
	reconcileOnce(t, r, &stored)

	var dep appsv1.Deployment
	if err := c.Get(context.Background(),
		types.NamespacedName{Namespace: "srenix-system", Name: "bionic-watcher"},
		&dep); err != nil {
		t.Fatalf("get dep: %v", err)
	}
	if dep.Spec.Replicas == nil || *dep.Spec.Replicas != 3 {
		t.Errorf("replicas=%v want 3 after CR update", dep.Spec.Replicas)
	}
}

// --- Remediate flow ---

func TestReconcile_RemediateEnabled_CreatesCronJob(t *testing.T) {
	cr := fullCR()
	cr.Spec.Remediate = &chav1alpha1.RemediateSpec{Enabled: true, DryRun: true}
	r, c := newReconciler(t, cr)
	reconcileOnce(t, r, cr)

	var cj batchv1.CronJob
	if err := c.Get(context.Background(),
		types.NamespacedName{Namespace: "srenix-system", Name: "bionic-remediate"},
		&cj); err != nil {
		t.Errorf("remediate CronJob not created: %v", err)
	}
}

// --- ServiceAccountName override ---

func TestReconcile_ExplicitServiceAccountName_UsedByDeployment(t *testing.T) {
	cr := fullCR()
	cr.Spec.ServiceAccountName = "shared-srenix-sa"
	r, c := newReconciler(t, cr)
	reconcileOnce(t, r, cr)

	var dep appsv1.Deployment
	_ = c.Get(context.Background(),
		types.NamespacedName{Namespace: "srenix-system", Name: "bionic-watcher"},
		&dep)
	if dep.Spec.Template.Spec.ServiceAccountName != "shared-srenix-sa" {
		t.Errorf("SA on watcher pod=%q want shared-srenix-sa", dep.Spec.Template.Spec.ServiceAccountName)
	}
}

// When the CR pins spec.serviceAccountName (BYO reader-bound SA), the
// operator must NOT create or own that SA — owning it would graft an
// owner-ref onto a pre-existing SA and garbage-collect it on CR delete.
func TestReconcile_ExplicitServiceAccountName_NotCreatedByOperator(t *testing.T) {
	cr := fullCR()
	cr.Spec.ServiceAccountName = "shared-srenix-sa"
	r, c := newReconciler(t, cr)
	reconcileOnce(t, r, cr)

	var sa corev1.ServiceAccount
	err := c.Get(context.Background(),
		types.NamespacedName{Namespace: "srenix-system", Name: "shared-srenix-sa"},
		&sa)
	if err == nil {
		t.Fatalf("operator must not create the BYO SA %q (it belongs to the caller)", "shared-srenix-sa")
	}
}
