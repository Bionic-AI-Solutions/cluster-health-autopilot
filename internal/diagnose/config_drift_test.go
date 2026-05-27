// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package diagnose

import (
	"context"
	"strings"
	"testing"
	"time"

	pkgsnapshot "github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/snapshot"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// memSourceCfg backs every gvr.Resource by an in-memory slice. Tests
// populate one map per analyzer signal — easier to read than a single
// merged source.
type memSourceCfg struct {
	byResource map[string][]unstructured.Unstructured
}

func (m *memSourceCfg) List(_ context.Context, gvr schema.GroupVersionResource, ns string) (*unstructured.UnstructuredList, error) {
	out := &unstructured.UnstructuredList{}
	for _, u := range m.byResource[gvr.Resource] {
		if ns != "" && u.GetNamespace() != ns {
			continue
		}
		out.Items = append(out.Items, u)
	}
	return out, nil
}

func (m *memSourceCfg) Get(_ context.Context, gvr schema.GroupVersionResource, ns, name string) (*unstructured.Unstructured, error) {
	for _, u := range m.byResource[gvr.Resource] {
		if u.GetNamespace() == ns && u.GetName() == name {
			return &u, nil
		}
	}
	return nil, nil
}

func (m *memSourceCfg) Mode() pkgsnapshot.Mode { return pkgsnapshot.ModeLive }

// makeCRD builds a CRD with the given storedVersions slice.
func makeCRD(name string, storedVersions ...string) unstructured.Unstructured {
	u := unstructured.Unstructured{}
	u.SetAPIVersion("apiextensions.k8s.io/v1")
	u.SetKind("CustomResourceDefinition")
	u.SetName(name)
	if len(storedVersions) > 0 {
		iface := make([]interface{}, len(storedVersions))
		for i, v := range storedVersions {
			iface[i] = v
		}
		_ = unstructured.SetNestedSlice(u.Object, iface, "status", "storedVersions")
	}
	return u
}

// makeDeploy builds a Deployment with generation/observedGen/replicas/
// updatedReplicas/availableReplicas and a creationTimestamp.
func makeDeploy(ns, name string, gen, observed, replicas, updated, available int64, createdAgo time.Duration) unstructured.Unstructured {
	u := unstructured.Unstructured{}
	u.SetAPIVersion("apps/v1")
	u.SetKind("Deployment")
	u.SetNamespace(ns)
	u.SetName(name)
	u.SetCreationTimestamp(metav1.Time{Time: time.Now().Add(-createdAgo)})
	_ = unstructured.SetNestedField(u.Object, gen, "metadata", "generation")
	_ = unstructured.SetNestedField(u.Object, replicas, "spec", "replicas")
	_ = unstructured.SetNestedField(u.Object, observed, "status", "observedGeneration")
	_ = unstructured.SetNestedField(u.Object, updated, "status", "updatedReplicas")
	_ = unstructured.SetNestedField(u.Object, available, "status", "availableReplicas")
	return u
}

// makeReplicaSet builds an RS owned by a Deployment.
func makeReplicaSet(ns, name, ownerDeploy string) unstructured.Unstructured {
	u := unstructured.Unstructured{}
	u.SetAPIVersion("apps/v1")
	u.SetKind("ReplicaSet")
	u.SetNamespace(ns)
	u.SetName(name)
	u.SetOwnerReferences([]metav1.OwnerReference{{Kind: "Deployment", Name: ownerDeploy, APIVersion: "apps/v1"}})
	return u
}

// makePodOwnedBy builds a Pod owned by the given ReplicaSet with an
// optional checksum/config annotation.
func makePodOwnedBy(ns, name, ownerRS, checksum string) unstructured.Unstructured {
	u := unstructured.Unstructured{}
	u.SetAPIVersion("v1")
	u.SetKind("Pod")
	u.SetNamespace(ns)
	u.SetName(name)
	u.SetOwnerReferences([]metav1.OwnerReference{{Kind: "ReplicaSet", Name: ownerRS, APIVersion: "apps/v1"}})
	if checksum != "" {
		u.SetAnnotations(map[string]string{"checksum/config": checksum})
	}
	return u
}

// --- CRD storedVersions ---

func TestConfigDrift_CRDSingleStoredVersion_NoFinding(t *testing.T) {
	src := &memSourceCfg{byResource: map[string][]unstructured.Unstructured{
		"customresourcedefinitions": {makeCRD("widgets.example.com", "v1")},
	}}
	got := ConfigDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("single storedVersion is healthy; got %d diagnostics: %+v", len(got), got)
	}
}

func TestConfigDrift_CRDMultiStoredVersion_Critical(t *testing.T) {
	src := &memSourceCfg{byResource: map[string][]unstructured.Unstructured{
		"customresourcedefinitions": {makeCRD("widgets.example.com", "v1alpha1", "v1")},
	}}
	got := ConfigDrift{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("expected 1 diagnostic; got %d: %+v", len(got), got)
	}
	if got[0].Severity != "critical" {
		t.Errorf("severity=%s want critical", got[0].Severity)
	}
	if !strings.Contains(got[0].Message, "multiple storedVersions") {
		t.Errorf("message lacks 'multiple storedVersions': %s", got[0].Message)
	}
	if !strings.Contains(got[0].Remediation, "storage version migrator") {
		t.Errorf("remediation lacks migrator guidance: %s", got[0].Remediation)
	}
}

func TestConfigDrift_CRDNoStoredVersions_NoFinding(t *testing.T) {
	// Edge case: a brand-new CRD before the apiserver has accepted
	// any objects. status.storedVersions is empty / nil.
	src := &memSourceCfg{byResource: map[string][]unstructured.Unstructured{
		"customresourcedefinitions": {makeCRD("widgets.example.com")},
	}}
	got := ConfigDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("empty storedVersions is healthy; got %d diagnostics", len(got))
	}
}

// --- Deployment rollouts ---

func TestConfigDrift_DeployFullyRolledOut_NoFinding(t *testing.T) {
	// 3/3 updated, 3/3 available, gen==observed, plenty old.
	src := &memSourceCfg{byResource: map[string][]unstructured.Unstructured{
		"deployments": {makeDeploy("app", "x", 7, 7, 3, 3, 3, time.Hour)},
	}}
	got := ConfigDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("healthy deployment; got %d diagnostics: %+v", len(got), got)
	}
}

func TestConfigDrift_DeployScaledToZero_NoFinding(t *testing.T) {
	// spec.replicas=0 is intentional state.
	src := &memSourceCfg{byResource: map[string][]unstructured.Unstructured{
		"deployments": {makeDeploy("app", "x", 7, 7, 0, 0, 0, time.Hour)},
	}}
	got := ConfigDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("scaled-to-zero is healthy; got %d diagnostics", len(got))
	}
}

func TestConfigDrift_DeployWithinGracePeriod_Suppressed(t *testing.T) {
	// updatedReplicas trails but only 5 min old — grace is 15.
	src := &memSourceCfg{byResource: map[string][]unstructured.Unstructured{
		"deployments": {makeDeploy("app", "x", 7, 7, 3, 1, 1, 5*time.Minute)},
	}}
	got := ConfigDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("rollout within grace should be silent; got: %+v", got)
	}
}

func TestConfigDrift_DeployRolloutStuck_Warning(t *testing.T) {
	// updated=1/3, available=1, well past grace.
	src := &memSourceCfg{byResource: map[string][]unstructured.Unstructured{
		"deployments": {makeDeploy("app", "x", 7, 7, 3, 1, 1, time.Hour)},
	}}
	got := ConfigDrift{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("expected 1 diagnostic; got %d: %+v", len(got), got)
	}
	if got[0].Severity != "warning" {
		t.Errorf("severity=%s want warning (some replicas still available)", got[0].Severity)
	}
	if !strings.Contains(got[0].Message, "rollout stuck") {
		t.Errorf("message lacks 'rollout stuck': %s", got[0].Message)
	}
}

func TestConfigDrift_DeployAllReplicasUnavailable_Critical(t *testing.T) {
	// updated=2/3, available=0 — every pod of the new revision is
	// failing. Critical because the workload is down.
	src := &memSourceCfg{byResource: map[string][]unstructured.Unstructured{
		"deployments": {makeDeploy("app", "x", 7, 7, 3, 2, 0, time.Hour)},
	}}
	got := ConfigDrift{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("expected 1 diagnostic; got %d", len(got))
	}
	if got[0].Severity != "critical" {
		t.Errorf("severity=%s want critical (workload fully unavailable)", got[0].Severity)
	}
}

func TestConfigDrift_DeployGenerationSkew_Critical(t *testing.T) {
	// gen=8, observedGen=7 — controller hasn't reconciled the
	// latest spec. Almost always a webhook rejection or paused
	// controller.
	src := &memSourceCfg{byResource: map[string][]unstructured.Unstructured{
		"deployments": {makeDeploy("app", "x", 8, 7, 3, 3, 3, time.Hour)},
	}}
	got := ConfigDrift{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("expected 1 diagnostic; got %d", len(got))
	}
	if got[0].Severity != "critical" {
		t.Errorf("severity=%s want critical (generation skew)", got[0].Severity)
	}
	if !strings.Contains(got[0].Message, "observedGeneration") {
		t.Errorf("message lacks 'observedGeneration': %s", got[0].Message)
	}
}

func TestConfigDrift_DeployInKubeSystem_Skipped(t *testing.T) {
	// kube-system deployments rotate on node lifecycle events; not
	// our signal.
	src := &memSourceCfg{byResource: map[string][]unstructured.Unstructured{
		"deployments": {makeDeploy("kube-system", "metrics-server", 7, 7, 3, 1, 1, time.Hour)},
	}}
	got := ConfigDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("kube-system should be skipped; got: %+v", got)
	}
}

func TestConfigDrift_CustomGracePeriod(t *testing.T) {
	// updatedReplicas trails but only 8 min old. Override grace to
	// 5 min — should flag.
	src := &memSourceCfg{byResource: map[string][]unstructured.Unstructured{
		"deployments": {makeDeploy("app", "x", 7, 7, 3, 1, 1, 8*time.Minute)},
	}}
	got := ConfigDrift{GracePeriod: 5 * time.Minute}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Errorf("with 5-min grace, an 8-min-old rollout should flag; got %d", len(got))
	}
}

// --- checksum/config drift across replicas ---

func TestConfigDrift_PodsAgreeOnChecksum_NoFinding(t *testing.T) {
	src := &memSourceCfg{byResource: map[string][]unstructured.Unstructured{
		"replicasets": {makeReplicaSet("app", "x-abc", "x")},
		"pods": {
			makePodOwnedBy("app", "x-abc-1", "x-abc", "deadbeef"),
			makePodOwnedBy("app", "x-abc-2", "x-abc", "deadbeef"),
			makePodOwnedBy("app", "x-abc-3", "x-abc", "deadbeef"),
		},
	}}
	got := ConfigDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("identical checksums across replicas is healthy; got: %+v", got)
	}
}

func TestConfigDrift_PodsDisagreeOnChecksum_Warning(t *testing.T) {
	// 2 pods on old hash, 1 on new — rolling update stuck.
	src := &memSourceCfg{byResource: map[string][]unstructured.Unstructured{
		"replicasets": {
			makeReplicaSet("app", "x-old", "x"),
			makeReplicaSet("app", "x-new", "x"),
		},
		"pods": {
			makePodOwnedBy("app", "x-old-1", "x-old", "deadbeef"),
			makePodOwnedBy("app", "x-old-2", "x-old", "deadbeef"),
			makePodOwnedBy("app", "x-new-1", "x-new", "cafebabe"),
		},
	}}
	got := ConfigDrift{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("expected 1 diagnostic; got %d: %+v", len(got), got)
	}
	if got[0].Severity != "warning" {
		t.Errorf("severity=%s want warning", got[0].Severity)
	}
	if !strings.Contains(got[0].Message, "checksum/config") {
		t.Errorf("message lacks 'checksum/config': %s", got[0].Message)
	}
	if got[0].Subject != "Deployment/app/x" {
		t.Errorf("subject=%s want Deployment/app/x", got[0].Subject)
	}
}

func TestConfigDrift_PodsWithoutChecksumAnnotation_NoFinding(t *testing.T) {
	// Workload doesn't use the checksum pattern — silent.
	src := &memSourceCfg{byResource: map[string][]unstructured.Unstructured{
		"replicasets": {makeReplicaSet("app", "x-abc", "x")},
		"pods": {
			makePodOwnedBy("app", "x-abc-1", "x-abc", ""),
			makePodOwnedBy("app", "x-abc-2", "x-abc", ""),
		},
	}}
	got := ConfigDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("no-annotation pods should be silent; got: %+v", got)
	}
}

func TestConfigDrift_RunNoOpOnEmptySource(t *testing.T) {
	src := &memSourceCfg{byResource: map[string][]unstructured.Unstructured{}}
	got := ConfigDrift{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("empty source should be no-op; got: %+v", got)
	}
}

func TestConfigDrift_NameStable(t *testing.T) {
	// Pinned for metrics + dashboards.
	if name := (ConfigDrift{}).Name(); name != "ConfigDrift" {
		t.Errorf("Name()=%q want ConfigDrift", name)
	}
}
