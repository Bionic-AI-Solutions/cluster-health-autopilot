// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package netpol

import (
	"context"
	"strings"
	"testing"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/snapshot"
	pkgsnapshot "github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/snapshot"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// memSourceNet is a deterministic in-memory snapshot.Source for CNI
// detection tests. Keyed by GVR.Resource (matches the convention used
// in internal/diagnose tests).
type memSourceNet struct {
	byResource map[string][]unstructured.Unstructured
}

func (m *memSourceNet) List(_ context.Context, gvr schema.GroupVersionResource, _ string) (*unstructured.UnstructuredList, error) {
	out := &unstructured.UnstructuredList{}
	out.Items = append(out.Items, m.byResource[gvr.Resource]...)
	return out, nil
}
func (m *memSourceNet) Get(_ context.Context, gvr schema.GroupVersionResource, _, name string) (*unstructured.Unstructured, error) {
	for _, u := range m.byResource[gvr.Resource] {
		if u.GetName() == name {
			c := u.DeepCopy()
			return c, nil
		}
	}
	return nil, nil
}
func (m *memSourceNet) Mode() pkgsnapshot.Mode { return pkgsnapshot.ModeLive }

func makeDS(ns, name string) unstructured.Unstructured {
	u := unstructured.Unstructured{}
	u.SetAPIVersion("apps/v1")
	u.SetKind("DaemonSet")
	u.SetNamespace(ns)
	u.SetName(name)
	return u
}

func TestDetectCNI_FlannelOnly(t *testing.T) {
	src := &memSourceNet{byResource: map[string][]unstructured.Unstructured{
		"daemonsets": {makeDS("kube-flannel", "kube-flannel-ds")},
	}}
	d := DetectCNI(context.Background(), src)
	if d.Enforces {
		t.Errorf("Flannel-only must NOT enforce; got %+v", d)
	}
	if d.CNIName != "flannel-only" {
		t.Errorf("CNIName=%q want flannel-only", d.CNIName)
	}
}

func TestDetectCNI_Calico(t *testing.T) {
	src := &memSourceNet{byResource: map[string][]unstructured.Unstructured{
		"daemonsets": {
			makeDS("kube-flannel", "kube-flannel-ds"), // Flannel + Calico = Calico wins
			makeDS("calico-system", "calico-node"),
		},
	}}
	d := DetectCNI(context.Background(), src)
	if !d.Enforces {
		t.Errorf("Calico must enforce; got %+v", d)
	}
	if d.CNIName != "calico" {
		t.Errorf("CNIName=%q want calico", d.CNIName)
	}
}

func TestDetectCNI_Cilium(t *testing.T) {
	src := &memSourceNet{byResource: map[string][]unstructured.Unstructured{
		"daemonsets": {makeDS("kube-system", "cilium")},
	}}
	d := DetectCNI(context.Background(), src)
	if !d.Enforces {
		t.Errorf("Cilium must enforce; got %+v", d)
	}
	if d.CNIName != "cilium" {
		t.Errorf("CNIName=%q want cilium", d.CNIName)
	}
}

func TestDetectCNI_AWSVPCCNI(t *testing.T) {
	src := &memSourceNet{byResource: map[string][]unstructured.Unstructured{
		"daemonsets": {makeDS("kube-system", "aws-node")},
	}}
	d := DetectCNI(context.Background(), src)
	if !d.Enforces {
		t.Errorf("AWS VPC CNI must enforce (caveat in evidence); got %+v", d)
	}
	if d.CNIName != "aws-vpc-cni" {
		t.Errorf("CNIName=%q want aws-vpc-cni", d.CNIName)
	}
}

func TestDetectCNI_GKEDataplaneV2(t *testing.T) {
	src := &memSourceNet{byResource: map[string][]unstructured.Unstructured{
		"daemonsets": {makeDS("kube-system", "anetd")},
	}}
	d := DetectCNI(context.Background(), src)
	if !d.Enforces || d.CNIName != "gke-dataplane-v2" {
		t.Errorf("expected gke-dataplane-v2; got %+v", d)
	}
}

func TestDetectCNI_AzureCNI(t *testing.T) {
	src := &memSourceNet{byResource: map[string][]unstructured.Unstructured{
		"daemonsets": {makeDS("kube-system", "azure-npm")},
	}}
	d := DetectCNI(context.Background(), src)
	if !d.Enforces || d.CNIName != "azure-cni" {
		t.Errorf("expected azure-cni; got %+v", d)
	}
}

// TestDetectCNI_KubeRouterAddOn — v1.12.3 fix for a real production
// outage (2026-06-01). kube-router is commonly layered on Flannel as
// a NetPol enforcement add-on. When BOTH are present, NetPol IS
// enforced and any "flannel-only → no enforcement" assumption is
// wrong. The proposer activating on these clusters is correct; the
// SecurityDrift downgrade to info is wrong; and (critically) any
// SRE who APPLIES a NetworkPolicy expecting it to be inert will
// instead break their cluster.
func TestDetectCNI_KubeRouterAddOn(t *testing.T) {
	src := &memSourceNet{byResource: map[string][]unstructured.Unstructured{
		"daemonsets": {
			makeDS("kube-flannel", "kube-flannel-ds"),
			makeDS("kube-system", "kube-router"),
		},
	}}
	d := DetectCNI(context.Background(), src)
	if !d.Enforces {
		t.Errorf("kube-router add-on must enforce; got %+v", d)
	}
	if d.CNIName != "kube-router" {
		t.Errorf("CNIName=%q want kube-router", d.CNIName)
	}
	if !strings.Contains(d.Evidence, "Flannel") {
		t.Errorf("evidence should mention base CNI (Flannel) too; got %q", d.Evidence)
	}
}

// TestDetectCNI_KubeRouterStandalone — kube-router alone (no Flannel)
// also enforces.
func TestDetectCNI_KubeRouterStandalone(t *testing.T) {
	src := &memSourceNet{byResource: map[string][]unstructured.Unstructured{
		"daemonsets": {makeDS("kube-system", "kube-router")},
	}}
	d := DetectCNI(context.Background(), src)
	if !d.Enforces || d.CNIName != "kube-router" {
		t.Errorf("expected kube-router/Enforces=true standalone; got %+v", d)
	}
}

func TestDetectCNI_Unknown(t *testing.T) {
	src := &memSourceNet{byResource: map[string][]unstructured.Unstructured{
		"daemonsets": {makeDS("kube-system", "kube-proxy")}, // not a CNI signal
	}}
	d := DetectCNI(context.Background(), src)
	if d.Enforces {
		t.Errorf("unknown CNI must NOT enforce; got %+v", d)
	}
	if d.CNIName != "unknown" {
		t.Errorf("CNIName=%q want unknown", d.CNIName)
	}
}

func TestDetectCNI_EmptyCluster(t *testing.T) {
	src := &memSourceNet{byResource: map[string][]unstructured.Unstructured{}}
	d := DetectCNI(context.Background(), src)
	if d.Enforces {
		t.Errorf("empty cluster must NOT enforce")
	}
}

// Make sure snapshot.GVRDaemonSet matches our memSourceNet key.
func TestGVRDaemonSetResourceKey(t *testing.T) {
	if snapshot.GVRDaemonSet.Resource != "daemonsets" {
		t.Errorf("GVRDaemonSet.Resource = %q, tests expect 'daemonsets'", snapshot.GVRDaemonSet.Resource)
	}
}
