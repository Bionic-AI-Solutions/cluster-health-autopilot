// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package feeder

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestDetectOwner_OperatorManaged_v1_25_0(t *testing.T) {
	// Deployment owned by ClusterHealthAutopilot CR (no Helm labels,
	// no ArgoCD annotation) should produce owner_chart synthesized
	// from the CR Kind + name.
	ctrl := true
	obj := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata": map[string]interface{}{
			"name":      "bionic-aiwatch",
			"namespace": "cluster-health-autopilot",
			"ownerReferences": []interface{}{map[string]interface{}{
				"apiVersion": "cha.bionicaisolutions.com/v1alpha1",
				"kind":       "ClusterHealthAutopilot",
				"name":       "bionic",
				"controller": ctrl,
			}},
		},
	}}
	o := detectOwner(obj)
	if o == nil {
		t.Fatal("expected non-nil owner for operator-managed Deployment")
	}
	if o.Kind != "Operator" {
		t.Errorf("Kind=%q want Operator", o.Kind)
	}
	if want := "clusterhealthautopilot-bionic"; o.ChartName != want {
		t.Errorf("ChartName=%q want %q", o.ChartName, want)
	}
	if o.ReleaseName != "bionic" {
		t.Errorf("ReleaseName=%q want bionic", o.ReleaseName)
	}
}

func TestDetectOwner_AppsControllerOwnerRef_StillNil(t *testing.T) {
	// A Deployment owned by apps/v1 ReplicaSet (built-in workload parent)
	// must NOT be treated as operator-managed. The operator-ownerRef
	// rule is specifically for custom-resource parents.
	ctrl := true
	obj := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata": map[string]interface{}{
			"name":      "no-helm-no-argo-no-operator",
			"namespace": "demo",
			"ownerReferences": []interface{}{map[string]interface{}{
				"apiVersion": "apps/v1",
				"kind":       "ReplicaSet",
				"name":       "should-not-match",
				"controller": ctrl,
			}},
		},
	}}
	if o := detectOwner(obj); o != nil {
		t.Errorf("apps/v1 ReplicaSet owner should not synthesize Operator owner; got %+v", o)
	}
}
