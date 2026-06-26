// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	"context"
	"testing"

	chav1alpha1 "github.com/srenix-ai/agentic-sre/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// Phase 2b — Qdrant StatefulSet + Service tests.

// memCR returns a CR with AI + memory both enabled. embeddings.model
// is required when memory is on; everything else takes defaults.
func memCR() *chav1alpha1.AgenticSRE {
	cr := aiCR()
	cr.Spec.AI.Memory = &chav1alpha1.AIMemorySpec{
		Enabled: true,
		Embeddings: &chav1alpha1.AIEmbeddingsSpec{
			Model: "qwen3-embedding-0.6b",
		},
	}
	return cr
}

// --- BuildQdrantStatefulSet + BuildQdrantService ---

func TestBuildQdrant_DisabledReturnsNil(t *testing.T) {
	cr := aiCR() // AI on, memory off
	if ss := BuildQdrantStatefulSet(cr); ss != nil {
		t.Errorf("memory off should produce nil StatefulSet; got %v", ss)
	}
	if svc := BuildQdrantService(cr); svc != nil {
		t.Errorf("memory off should produce nil Service; got %v", svc)
	}
}

func TestBuildQdrantStatefulSet_BasicShape(t *testing.T) {
	cr := memCR()
	ss := BuildQdrantStatefulSet(cr)
	if ss == nil {
		t.Fatal("memory enabled must produce a StatefulSet")
	}
	if ss.Name != "bionic-rag" {
		t.Errorf("name=%q want bionic-rag", ss.Name)
	}
	if ss.Spec.ServiceName != "bionic-rag" {
		t.Errorf("serviceName=%q want bionic-rag (must match the headless Service)", ss.Spec.ServiceName)
	}
	if ss.Spec.Replicas == nil || *ss.Spec.Replicas != 1 {
		t.Errorf("replicas=%v want 1", ss.Spec.Replicas)
	}
	if ss.Labels["srenix.ai/role"] != "rag" {
		t.Errorf("role label=%q want rag", ss.Labels["srenix.ai/role"])
	}
}

func TestBuildQdrantStatefulSet_DefaultImage(t *testing.T) {
	cr := memCR()
	ss := BuildQdrantStatefulSet(cr)
	got := ss.Spec.Template.Spec.Containers[0].Image
	want := "qdrant/qdrant:v1.12.4"
	if got != want {
		t.Errorf("default image=%q want %q", got, want)
	}
}

func TestBuildQdrantStatefulSet_ImageOverride(t *testing.T) {
	cr := memCR()
	cr.Spec.AI.Memory.Image = &chav1alpha1.ImageSpec{
		Repository: "myco/qdrant", Tag: "v1.13.0",
	}
	ss := BuildQdrantStatefulSet(cr)
	if got := ss.Spec.Template.Spec.Containers[0].Image; got != "myco/qdrant:v1.13.0" {
		t.Errorf("image override not honored; got %q", got)
	}
}

func TestBuildQdrantStatefulSet_PortsAndProbes(t *testing.T) {
	cr := memCR()
	ss := BuildQdrantStatefulSet(cr)
	c := ss.Spec.Template.Spec.Containers[0]
	wantPorts := map[string]int32{"http": 6333, "grpc": 6334}
	got := map[string]int32{}
	for _, p := range c.Ports {
		got[p.Name] = p.ContainerPort
	}
	for name, port := range wantPorts {
		if got[name] != port {
			t.Errorf("port[%s]=%d want %d", name, got[name], port)
		}
	}
	if c.ReadinessProbe == nil || c.ReadinessProbe.HTTPGet == nil ||
		c.ReadinessProbe.HTTPGet.Path != "/readyz" {
		t.Errorf("readinessProbe path missing/wrong; got %+v", c.ReadinessProbe)
	}
	if c.LivenessProbe == nil || c.LivenessProbe.HTTPGet == nil ||
		c.LivenessProbe.HTTPGet.Path != "/livez" {
		t.Errorf("livenessProbe path missing/wrong; got %+v", c.LivenessProbe)
	}
}

func TestBuildQdrantStatefulSet_SnapshotTempEnvOverrides(t *testing.T) {
	// Without these, Qdrant tries to write to /qdrant/snapshots and
	// /qdrant/.qdrant-temp on the read-only image FS and CrashLoops.
	// Chart sets them explicitly; operator MUST mirror.
	cr := memCR()
	ss := BuildQdrantStatefulSet(cr)
	env := ss.Spec.Template.Spec.Containers[0].Env
	want := map[string]string{
		"QDRANT__STORAGE__SNAPSHOTS_PATH": "/qdrant/storage/snapshots",
		"QDRANT__STORAGE__TEMP_PATH":      "/qdrant/storage/temp",
	}
	for _, e := range env {
		if expect, ok := want[e.Name]; ok {
			if e.Value != expect {
				t.Errorf("env[%s]=%q want %q", e.Name, e.Value, expect)
			}
			delete(want, e.Name)
		}
	}
	if len(want) != 0 {
		t.Errorf("missing env: %v", want)
	}
}

func TestBuildQdrantStatefulSet_VolumeClaimTemplateDefault(t *testing.T) {
	cr := memCR()
	ss := BuildQdrantStatefulSet(cr)
	if len(ss.Spec.VolumeClaimTemplates) != 1 {
		t.Fatalf("expected exactly 1 VolumeClaimTemplate; got %d", len(ss.Spec.VolumeClaimTemplates))
	}
	pvc := ss.Spec.VolumeClaimTemplates[0]
	if pvc.Name != "storage" {
		t.Errorf("PVC template name=%q want 'storage'", pvc.Name)
	}
	storage := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
	if storage.String() != "5Gi" {
		t.Errorf("default storage=%s want 5Gi", storage.String())
	}
	if pvc.Spec.StorageClassName != nil {
		t.Errorf("default storageClassName should be nil; got %v", *pvc.Spec.StorageClassName)
	}
}

func TestBuildQdrantStatefulSet_VolumeClaimTemplateOverride(t *testing.T) {
	cr := memCR()
	className := "ceph-rbd"
	cr.Spec.AI.Memory.Storage = &chav1alpha1.AIMemoryStorageSpec{
		Size:      "20Gi",
		ClassName: className,
	}
	ss := BuildQdrantStatefulSet(cr)
	pvc := ss.Spec.VolumeClaimTemplates[0]
	if got := pvc.Spec.Resources.Requests[corev1.ResourceStorage]; got.String() != "20Gi" {
		t.Errorf("storage=%s want 20Gi", got.String())
	}
	if pvc.Spec.StorageClassName == nil || *pvc.Spec.StorageClassName != "ceph-rbd" {
		t.Errorf("storageClassName not honored; got %v", pvc.Spec.StorageClassName)
	}
}

func TestBuildQdrantStatefulSet_VolumeMount(t *testing.T) {
	cr := memCR()
	ss := BuildQdrantStatefulSet(cr)
	mounts := ss.Spec.Template.Spec.Containers[0].VolumeMounts
	if len(mounts) != 1 || mounts[0].Name != "storage" ||
		mounts[0].MountPath != "/qdrant/storage" {
		t.Errorf("volumeMount not wired to /qdrant/storage; got %+v", mounts)
	}
}

func TestBuildQdrantService_BasicShape(t *testing.T) {
	cr := memCR()
	svc := BuildQdrantService(cr)
	if svc == nil {
		t.Fatal("memory enabled must produce a Service")
	}
	if svc.Name != "bionic-rag" {
		t.Errorf("name=%q want bionic-rag", svc.Name)
	}
	if svc.Spec.Type != corev1.ServiceTypeClusterIP {
		t.Errorf("type=%v want ClusterIP", svc.Spec.Type)
	}
	if svc.Spec.Selector["srenix.ai/role"] != "rag" {
		t.Errorf("selector role=%q want rag", svc.Spec.Selector["srenix.ai/role"])
	}
	wantPorts := map[string]int32{"http": 6333, "grpc": 6334}
	got := map[string]int32{}
	for _, p := range svc.Spec.Ports {
		got[p.Name] = p.Port
	}
	for n, p := range wantPorts {
		if got[n] != p {
			t.Errorf("svc port[%s]=%d want %d", n, got[n], p)
		}
	}
}

func TestNamesFor_IncludesRAG(t *testing.T) {
	cr := sampleCR()
	if got := NamesFor(cr).RAG; got != "bionic-rag" {
		t.Errorf("RAG name=%q want bionic-rag", got)
	}
}

// --- Reconcile-loop wiring ---

func TestReconcile_MemoryEnabled_CreatesStatefulSetAndService(t *testing.T) {
	cr := memCR()
	r, c := newReconciler(t, cr)
	reconcileOnce(t, r, cr)
	reconcileOnce(t, r, cr)

	var ss appsv1.StatefulSet
	if err := c.Get(context.Background(),
		types.NamespacedName{Namespace: "srenix-system", Name: "bionic-rag"},
		&ss); err != nil {
		t.Errorf("Qdrant StatefulSet not created: %v", err)
	}
	var svc corev1.Service
	if err := c.Get(context.Background(),
		types.NamespacedName{Namespace: "srenix-system", Name: "bionic-rag"},
		&svc); err != nil {
		t.Errorf("Qdrant Service not created: %v", err)
	}
}

func TestReconcile_MemoryDisabled_NoStatefulSet(t *testing.T) {
	cr := aiCR() // AI on, memory off
	r, c := newReconciler(t, cr)
	reconcileOnce(t, r, cr)
	reconcileOnce(t, r, cr)

	var ss appsv1.StatefulSet
	err := c.Get(context.Background(),
		types.NamespacedName{Namespace: "srenix-system", Name: "bionic-rag"},
		&ss)
	if !apierrors.IsNotFound(err) {
		t.Errorf("expected NotFound for memory-off StatefulSet; got err=%v", err)
	}
}

func TestReconcile_MemoryDisabledAfterCreate_DeletesStatefulSetAndService(t *testing.T) {
	cr := memCR()
	r, c := newReconciler(t, cr)
	reconcileOnce(t, r, cr)
	reconcileOnce(t, r, cr)

	// Flip memory off.
	var stored chav1alpha1.AgenticSRE
	_ = c.Get(context.Background(),
		types.NamespacedName{Namespace: "srenix-system", Name: "bionic"},
		&stored)
	stored.Spec.AI.Memory.Enabled = false
	stored.Generation = 2
	if err := c.Update(context.Background(), &stored); err != nil {
		t.Fatalf("update cr: %v", err)
	}
	reconcileOnce(t, r, &stored)

	var ss appsv1.StatefulSet
	err := c.Get(context.Background(),
		types.NamespacedName{Namespace: "srenix-system", Name: "bionic-rag"},
		&ss)
	if !apierrors.IsNotFound(err) {
		t.Errorf("StatefulSet not deleted after memory disable; got err=%v", err)
	}
	var svc corev1.Service
	err = c.Get(context.Background(),
		types.NamespacedName{Namespace: "srenix-system", Name: "bionic-rag"},
		&svc)
	if !apierrors.IsNotFound(err) {
		t.Errorf("Service not deleted after memory disable; got err=%v", err)
	}
}

func TestReconcile_MemoryEnabled_InvalidStorageSize_ReadyFalse(t *testing.T) {
	cr := memCR()
	cr.Spec.AI.Memory.Storage = &chav1alpha1.AIMemoryStorageSpec{Size: "5gb"} // lower-case b is invalid
	r, c := newReconciler(t, cr)
	reconcileOnce(t, r, cr)

	cond := readReadyCondition(t, c, cr)
	if cond == nil || cond.Status != metav1.ConditionFalse || cond.Reason != "InvalidSpec" {
		t.Errorf("expected Ready=False/InvalidSpec for invalid storage.size; got %+v", cond)
	}
}

func TestReconcile_MemoryDisabled_MemoryStoreReadyCondition_Disabled(t *testing.T) {
	cr := fullCR()
	r, c := newReconciler(t, cr)
	reconcileOnce(t, r, cr)
	reconcileOnce(t, r, cr)

	cond := readCondition(t, c, cr, chav1alpha1.ConditionMemoryStoreReady)
	if cond == nil {
		t.Fatal("MemoryStoreReady condition not set even when memory is off")
	}
	if cond.Status != metav1.ConditionFalse || cond.Reason != "Disabled" {
		t.Errorf("MemoryStoreReady=%s/%s; want False/Disabled", cond.Status, cond.Reason)
	}
}

func TestReconcile_QdrantReady_FlipsConditionTrue(t *testing.T) {
	cr := memCR()
	r, c := newReconciler(t, cr)
	reconcileOnce(t, r, cr)
	reconcileOnce(t, r, cr)

	// Simulate StatefulSet readiness.
	var ss appsv1.StatefulSet
	if err := c.Get(context.Background(),
		types.NamespacedName{Namespace: "srenix-system", Name: "bionic-rag"},
		&ss); err != nil {
		t.Fatalf("get ss: %v", err)
	}
	ss.Status.ReadyReplicas = 1
	if err := c.Status().Update(context.Background(), &ss); err != nil {
		t.Fatalf("ss status update: %v", err)
	}
	// Bump aiwatch too so AIWatchRunning doesn't block this test's
	// observation (AIWatchRunning gates Ready, not MemoryStoreReady,
	// but we want both flips visible for clarity).
	var aw appsv1.Deployment
	_ = c.Get(context.Background(),
		types.NamespacedName{Namespace: "srenix-system", Name: "bionic-aiwatch"},
		&aw)
	aw.Status.AvailableReplicas = 1
	_ = c.Status().Update(context.Background(), &aw)

	reconcileOnce(t, r, cr)

	cond := readCondition(t, c, cr, chav1alpha1.ConditionMemoryStoreReady)
	if cond == nil || cond.Status != metav1.ConditionTrue {
		t.Errorf("MemoryStoreReady=%+v; want True", cond)
	}
}

func TestReconcile_MemoryEnabled_StoreNotReady_ReadyFalse(t *testing.T) {
	// Even when the operator creates StatefulSet + Service, the fake
	// client reports readyReplicas=0 until the test bumps status.
	// With memory enabled, Ready must be False (gated on
	// MemoryStoreReady).
	cr := memCR()
	r, c := newReconciler(t, cr)
	reconcileOnce(t, r, cr)
	reconcileOnce(t, r, cr)

	cond := readReadyCondition(t, c, cr)
	if cond == nil || cond.Status == metav1.ConditionTrue {
		t.Errorf("Ready should not be True before Qdrant reports ready; got %+v", cond)
	}
}

// --- Owner refs ---

func TestReconcile_QdrantChildren_HaveControllerOwnerRef(t *testing.T) {
	cr := memCR()
	r, c := newReconciler(t, cr)
	reconcileOnce(t, r, cr)
	reconcileOnce(t, r, cr)

	var ss appsv1.StatefulSet
	_ = c.Get(context.Background(),
		types.NamespacedName{Namespace: "srenix-system", Name: "bionic-rag"},
		&ss)
	if len(ss.OwnerReferences) == 0 || ss.OwnerReferences[0].Name != "bionic" {
		t.Errorf("StatefulSet missing controller ownerRef; got %+v", ss.OwnerReferences)
	}

	var svc corev1.Service
	_ = c.Get(context.Background(),
		types.NamespacedName{Namespace: "srenix-system", Name: "bionic-rag"},
		&svc)
	if len(svc.OwnerReferences) == 0 || svc.OwnerReferences[0].Name != "bionic" {
		t.Errorf("Service missing controller ownerRef; got %+v", svc.OwnerReferences)
	}
}
