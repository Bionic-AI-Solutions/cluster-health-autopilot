// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package diagnose

import (
	"context"
	"strings"
	"testing"

	"github.com/srenix-ai/agentic-sre/pkg/netpol"
	pkgsnapshot "github.com/srenix-ai/agentic-sre/pkg/snapshot"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// memSourceProposer is the in-memory snapshot.Source used by the
// NetworkPolicyProposer analyzer tests. Keyed by GVR.Resource.
type memSourceProposer struct {
	byResource map[string][]unstructured.Unstructured
}

func (m *memSourceProposer) List(_ context.Context, gvr schema.GroupVersionResource, _ string) (*unstructured.UnstructuredList, error) {
	out := &unstructured.UnstructuredList{}
	out.Items = append(out.Items, m.byResource[gvr.Resource]...)
	return out, nil
}
func (m *memSourceProposer) Get(_ context.Context, gvr schema.GroupVersionResource, _, name string) (*unstructured.Unstructured, error) {
	for _, u := range m.byResource[gvr.Resource] {
		if u.GetName() == name {
			c := u.DeepCopy()
			return c, nil
		}
	}
	return nil, nil
}
func (m *memSourceProposer) Mode() pkgsnapshot.Mode { return pkgsnapshot.ModeLive }

func makeNamespaceProp(name string) unstructured.Unstructured {
	u := unstructured.Unstructured{}
	u.SetAPIVersion("v1")
	u.SetKind("Namespace")
	u.SetName(name)
	return u
}

func makeDaemonSet(ns, name string) unstructured.Unstructured {
	u := unstructured.Unstructured{}
	u.SetAPIVersion("apps/v1")
	u.SetKind("DaemonSet")
	u.SetNamespace(ns)
	u.SetName(name)
	return u
}

func makePodProp(ns, name string) unstructured.Unstructured {
	u := unstructured.Unstructured{}
	u.SetAPIVersion("v1")
	u.SetKind("Pod")
	u.SetNamespace(ns)
	u.SetName(name)
	return u
}

func makeIngressProp(ns, name, class string) unstructured.Unstructured {
	u := unstructured.Unstructured{}
	u.SetAPIVersion("networking.k8s.io/v1")
	u.SetKind("Ingress")
	u.SetNamespace(ns)
	u.SetName(name)
	if class != "" {
		_ = unstructured.SetNestedField(u.Object, class, "spec", "ingressClassName")
	}
	return u
}

// TestNetworkPolicyProposer_FlannelOnly_NoFindings — barebones k3s
// case. CNI doesn't enforce. The proposer must be SILENT.
func TestNetworkPolicyProposer_FlannelOnly_NoFindings(t *testing.T) {
	src := &memSourceProposer{byResource: map[string][]unstructured.Unstructured{
		"daemonsets": {makeDaemonSet("kube-flannel", "kube-flannel-ds")},
		"namespaces": {makeNamespaceProp("app"), makeNamespaceProp("backend")},
		"pods":       {makePodProp("app", "a-7d"), makePodProp("backend", "b-7d")},
	}}
	a := NetworkPolicyProposer{}
	got := a.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("Flannel-only cluster must NOT generate proposals; got %d: %+v", len(got), got)
	}
}

// TestNetworkPolicyProposer_Calico_HappyPath — cloud-typical cluster.
// CNI enforces. Two user namespaces with pods + no NetPols → two
// proposals.
func TestNetworkPolicyProposer_Calico_HappyPath(t *testing.T) {
	src := &memSourceProposer{byResource: map[string][]unstructured.Unstructured{
		"daemonsets": {makeDaemonSet("calico-system", "calico-node")},
		"namespaces": {makeNamespaceProp("app"), makeNamespaceProp("backend")},
		"pods":       {makePodProp("app", "a-7d"), makePodProp("backend", "b-7d")},
		"ingresses":  {makeIngressProp("app", "a-ing", "kong")},
	}}
	a := NetworkPolicyProposer{}
	got := a.Run(context.Background(), src)
	if len(got) != 2 {
		t.Fatalf("expected 2 proposals (app + backend); got %d: %+v", len(got), got)
	}
	for _, d := range got {
		if d.ProposedPolicyYAML == "" {
			t.Errorf("finding %q missing ProposedPolicyYAML", d.Subject)
		}
		if d.ProposedPolicyKind != "NetworkPolicy" {
			t.Errorf("finding %q PolicyKind=%q want NetworkPolicy", d.Subject, d.ProposedPolicyKind)
		}
		if !strings.Contains(d.Message, "calico") {
			t.Errorf("finding %q Message must name the CNI; got %q", d.Subject, d.Message)
		}
	}
}

// TestNetworkPolicyProposer_Cilium_SkipsNamespaceWithExistingNetPol —
// proposer must NOT step on existing policies.
func TestNetworkPolicyProposer_Cilium_SkipsNamespaceWithExistingNetPol(t *testing.T) {
	existingNetPol := unstructured.Unstructured{}
	existingNetPol.SetAPIVersion("networking.k8s.io/v1")
	existingNetPol.SetKind("NetworkPolicy")
	existingNetPol.SetNamespace("app")
	existingNetPol.SetName("existing-policy")

	src := &memSourceProposer{byResource: map[string][]unstructured.Unstructured{
		"daemonsets":      {makeDaemonSet("kube-system", "cilium")},
		"namespaces":      {makeNamespaceProp("app")},
		"pods":            {makePodProp("app", "a-7d")},
		"networkpolicies": {existingNetPol},
	}}
	a := NetworkPolicyProposer{}
	got := a.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("namespace with existing NetPol must be skipped; got %+v", got)
	}
}

// TestNetworkPolicyProposer_SystemNamespacesSkipped — kube-system /
// kube-public / kube-node-lease must never get a proposal.
func TestNetworkPolicyProposer_SystemNamespacesSkipped(t *testing.T) {
	src := &memSourceProposer{byResource: map[string][]unstructured.Unstructured{
		"daemonsets": {makeDaemonSet("kube-system", "cilium")},
		"namespaces": {makeNamespaceProp("kube-system"), makeNamespaceProp("kube-public"), makeNamespaceProp("kube-node-lease")},
		"pods":       {makePodProp("kube-system", "coredns")},
	}}
	a := NetworkPolicyProposer{}
	got := a.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("system namespaces must be skipped; got %+v", got)
	}
}

// TestNetworkPolicyProposer_NoEgressRestriction — the proposed YAML
// must NOT include egress restrictions. User's explicit requirement:
// "intra-cluster communication via internal DNS must keep working."
func TestNetworkPolicyProposer_NoEgressRestriction(t *testing.T) {
	src := &memSourceProposer{byResource: map[string][]unstructured.Unstructured{
		"daemonsets": {makeDaemonSet("calico-system", "calico-node")},
		"namespaces": {makeNamespaceProp("app")},
		"pods":       {makePodProp("app", "a-7d")},
		"ingresses":  {makeIngressProp("app", "a-ing", "kong")},
	}}
	a := NetworkPolicyProposer{}
	got := a.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("expected 1 proposal; got %d", len(got))
	}
	if strings.Contains(got[0].ProposedPolicyYAML, "Egress") {
		t.Errorf("proposed YAML must NOT contain Egress policyType (DNS/in-cluster/external must work); got:\n%s",
			got[0].ProposedPolicyYAML)
	}
}

// Compile-time guard: the NetworkPolicyProposer accepts a netpol.Proposer.
var _ netpol.Proposer = netpol.SnapshotProposer{}
