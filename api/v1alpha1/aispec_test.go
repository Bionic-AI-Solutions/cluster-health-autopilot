// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"testing"
)

func TestAISpec_DeepCopy_AllFields(t *testing.T) {
	src := &AISpec{
		Enabled:           true,
		Tier:              "t3",
		Endpoint:          "https://gpu-ai.example/v1",
		Model:             "qwen3.6-35b-a3b-fp8",
		Interval:          "60s",
		AllowSaaS:         false,
		LLMFixerMatcher:   true,
		AuditLog:          "-",
		ApprovalServerURL: "https://cha-approve.example",
		Image: &ImageSpec{
			Repository:  "docker4zerocool/cha-com",
			Tag:         "v1.9.3",
			PullPolicy:  "IfNotPresent",
			PullSecrets: []string{"dockerhub"},
		},
		APIKey: &AIAPIKeySpec{
			SecretName: "cha-ai-key",
			SecretKey:  "API_KEY",
			EnvName:    "AI_API_KEY",
			Header:     "X-API-Key",
		},
		T3: &AIT3Spec{
			VaultAllowedPrefixes: []string{"secret/data/cha-recovery/"},
		},
		Memory: &AIMemorySpec{
			Enabled:    true,
			Storage:    &AIMemoryStorageSpec{Size: "10Gi", ClassName: "ssd"},
			Embeddings: &AIEmbeddingsSpec{Endpoint: "https://gpu-ai.example/v1", Model: "qwen3-embedding-0.6b"},
			TopK:       7,
		},
	}
	dst := src.DeepCopy()

	if dst == src {
		t.Fatal("DeepCopy should return a new pointer")
	}
	// Field-by-field scalar check (AISpec now contains a []string —
	// struct value comparison is no longer supported in Go).
	if dst.Tier != src.Tier || dst.Endpoint != src.Endpoint {
		t.Errorf("scalar fields not copied: dst=%+v src=%+v", dst, src)
	}

	// Mutate the copy's nested pointers + slices; the original must be
	// unaffected (this is the whole point of DeepCopy).
	dst.Image.Tag = "MUTATED"
	dst.APIKey.SecretName = "MUTATED"
	dst.T3.VaultAllowedPrefixes[0] = "MUTATED"
	dst.Memory.Storage.Size = "MUTATED"
	dst.Memory.Embeddings.Model = "MUTATED"
	dst.Image.PullSecrets[0] = "MUTATED"

	if src.Image.Tag == "MUTATED" {
		t.Error("DeepCopy of Image shared the underlying value")
	}
	if src.APIKey.SecretName == "MUTATED" {
		t.Error("DeepCopy of APIKey shared the underlying value")
	}
	if src.T3.VaultAllowedPrefixes[0] == "MUTATED" {
		t.Error("DeepCopy of T3.VaultAllowedPrefixes shared the underlying slice")
	}
	if src.Memory.Storage.Size == "MUTATED" {
		t.Error("DeepCopy of Memory.Storage shared the underlying value")
	}
	if src.Memory.Embeddings.Model == "MUTATED" {
		t.Error("DeepCopy of Memory.Embeddings shared the underlying value")
	}
	if src.Image.PullSecrets[0] == "MUTATED" {
		t.Error("DeepCopy of Image.PullSecrets shared the underlying slice")
	}
}

func TestAISpec_DeepCopy_Nil(t *testing.T) {
	var s *AISpec
	if got := s.DeepCopy(); got != nil {
		t.Errorf("DeepCopy on nil receiver should return nil; got %+v", got)
	}
}

func TestClusterHealthAutopilotSpec_DeepCopy_IncludesAI(t *testing.T) {
	src := &ClusterHealthAutopilotSpec{
		Image: ImageSpec{Repository: "a", Tag: "b"},
		AI:    &AISpec{Enabled: true, Tier: "t1"},
	}
	dst := src.DeepCopy()
	if dst.AI == src.AI {
		t.Fatal("Spec.DeepCopy should clone AI pointer")
	}
	dst.AI.Tier = "MUTATED"
	if src.AI.Tier == "MUTATED" {
		t.Error("Spec.DeepCopy did not deep-copy AI nested struct")
	}
}
