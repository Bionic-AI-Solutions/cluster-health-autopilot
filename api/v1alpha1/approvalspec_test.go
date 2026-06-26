// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"testing"
)

func TestApprovalSpec_DeepCopy_AllFields(t *testing.T) {
	src := &ApprovalSpec{
		Enabled:        true,
		Replicas:       2,
		AuditNamespace: "audit",
		Image: &ImageSpec{
			Repository:  "docker4zerocool/srenix-enterprise",
			Tag:         "v1.9.4",
			PullPolicy:  "IfNotPresent",
			PullSecrets: []string{"dockerhub"},
		},
		SigningKey: &ApprovalSigningKeySpec{
			SecretName: "srenix-approval-signing-key",
		},
		Store: &ApprovalStoreSpec{
			Backend:          "configmap",
			Namespace:        "srenix-system",
			ReplayConfigMap:  "srenix-approval-replay",
			RunbookConfigMap: "srenix-approval-runbooks",
		},
		Ingress: &ApprovalIngressSpec{
			Enabled:          true,
			IngressClassName: "kong",
			Host:             "approve.srenix.example.com",
			Annotations: map[string]string{
				"cert-manager.io/cluster-issuer": "letsencrypt-prod",
			},
			TLS: &ApprovalIngressTLSSpec{
				Enabled:    true,
				SecretName: "srenix-approval-tls",
			},
		},
	}
	dst := src.DeepCopy()

	if dst == src {
		t.Fatal("DeepCopy should return a new pointer")
	}
	if dst.Image == src.Image {
		t.Error("nested Image pointer not cloned")
	}
	if dst.SigningKey == src.SigningKey {
		t.Error("nested SigningKey pointer not cloned")
	}
	if dst.Store == src.Store {
		t.Error("nested Store pointer not cloned")
	}
	if dst.Ingress == src.Ingress {
		t.Error("nested Ingress pointer not cloned")
	}
	if dst.Ingress.TLS == src.Ingress.TLS {
		t.Error("nested Ingress.TLS pointer not cloned")
	}

	// Mutate the copy; the source must be untouched.
	dst.Image.Tag = "MUTATED"
	dst.SigningKey.SecretName = "MUTATED"
	dst.Store.Backend = "MUTATED"
	dst.Ingress.Host = "MUTATED"
	dst.Ingress.Annotations["cert-manager.io/cluster-issuer"] = "MUTATED"
	dst.Ingress.TLS.SecretName = "MUTATED"
	dst.Image.PullSecrets[0] = "MUTATED"

	if src.Image.Tag == "MUTATED" {
		t.Error("DeepCopy shared Image underlying value")
	}
	if src.SigningKey.SecretName == "MUTATED" {
		t.Error("DeepCopy shared SigningKey underlying value")
	}
	if src.Store.Backend == "MUTATED" {
		t.Error("DeepCopy shared Store underlying value")
	}
	if src.Ingress.Host == "MUTATED" {
		t.Error("DeepCopy shared Ingress underlying value")
	}
	if src.Ingress.Annotations["cert-manager.io/cluster-issuer"] == "MUTATED" {
		t.Error("DeepCopy shared Ingress.Annotations map")
	}
	if src.Ingress.TLS.SecretName == "MUTATED" {
		t.Error("DeepCopy shared Ingress.TLS underlying value")
	}
	if src.Image.PullSecrets[0] == "MUTATED" {
		t.Error("DeepCopy shared Image.PullSecrets slice")
	}
}

func TestApprovalSpec_DeepCopy_Nil(t *testing.T) {
	var s *ApprovalSpec
	if got := s.DeepCopy(); got != nil {
		t.Errorf("DeepCopy on nil receiver should return nil; got %+v", got)
	}
}

func TestApprovalIngressSpec_DeepCopy_Nil(t *testing.T) {
	var s *ApprovalIngressSpec
	if got := s.DeepCopy(); got != nil {
		t.Errorf("DeepCopy on nil receiver should return nil; got %+v", got)
	}
}

func TestAgenticSRESpec_DeepCopy_IncludesApproval(t *testing.T) {
	src := &AgenticSRESpec{
		Image:    ImageSpec{Repository: "a", Tag: "b"},
		Approval: &ApprovalSpec{Enabled: true, Replicas: 3},
	}
	dst := src.DeepCopy()
	if dst.Approval == src.Approval {
		t.Fatal("Spec.DeepCopy should clone Approval pointer")
	}
	dst.Approval.Replicas = 9
	if src.Approval.Replicas == 9 {
		t.Error("Spec.DeepCopy did not deep-copy Approval nested struct")
	}
}
