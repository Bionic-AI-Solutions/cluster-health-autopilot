// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	"context"
	"strings"
	"testing"

	chav1alpha1 "github.com/Bionic-AI-Solutions/cluster-health-autopilot/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Phase 2 — AISpec consumption tests for both the pure builder and the
// reconcile-loop wiring.

// aiCR returns the canonical happy-path CR with AI enabled and the
// minimum required fields populated.
func aiCR() *chav1alpha1.ClusterHealthAutopilot {
	cr := fullCR()
	cr.Spec.AI = &chav1alpha1.AISpec{
		Enabled:  true,
		Endpoint: "https://gpu-ai.svc.cluster.local/v1",
		Model:    "qwen3.6-35b-a3b-fp8",
	}
	return cr
}

// --- BuildAIWatchDeployment ---

func TestBuildAIWatchDeployment_DisabledReturnsNil(t *testing.T) {
	cr := fullCR()
	// nil AISpec: disabled.
	if d := BuildAIWatchDeployment(cr); d != nil {
		t.Errorf("AI nil should produce nil; got %v", d)
	}
	// AISpec present but Enabled=false: disabled.
	cr.Spec.AI = &chav1alpha1.AISpec{Enabled: false}
	if d := BuildAIWatchDeployment(cr); d != nil {
		t.Errorf("AI.Enabled=false should produce nil; got %v", d)
	}
}

func TestBuildAIWatchDeployment_BasicShape(t *testing.T) {
	cr := aiCR()
	d := BuildAIWatchDeployment(cr)
	if d == nil {
		t.Fatal("AI enabled must produce a Deployment")
	}
	if d.Name != "bionic-aiwatch" {
		t.Errorf("name=%q want bionic-aiwatch", d.Name)
	}
	if d.Spec.Replicas == nil || *d.Spec.Replicas != 1 {
		t.Errorf("replicas=%v want 1", d.Spec.Replicas)
	}
	if d.Spec.Strategy.Type != appsv1.RecreateDeploymentStrategyType {
		t.Errorf("strategy=%v want Recreate (chart pins this for leader-lease safety)",
			d.Spec.Strategy.Type)
	}
	if d.Spec.Template.Spec.ServiceAccountName != "bionic-sa" {
		t.Errorf("SA=%q want bionic-sa (aiwatch shares the reader SA)",
			d.Spec.Template.Spec.ServiceAccountName)
	}
	if d.Labels["cha.bionicaisolutions.com/role"] != "aiwatch" {
		t.Errorf("role label=%q want aiwatch", d.Labels["cha.bionicaisolutions.com/role"])
	}
}

func TestBuildAIWatchDeployment_ImageDefaultsToChaCom_WithVPrefix(t *testing.T) {
	// Chart convention: cha-com images carry a leading "v" alongside
	// the OSS image's bare semver (chart appVersion 1.8.0 →
	// cha-com:v1.8.0). The operator mirrors this.
	cr := aiCR()
	cr.Spec.Image.Tag = "1.9.4"
	d := BuildAIWatchDeployment(cr)
	got := d.Spec.Template.Spec.Containers[0].Image
	want := "docker4zerocool/cha-com:v1.9.4"
	if got != want {
		t.Errorf("aiwatch image=%q want %q", got, want)
	}
}

func TestBuildAIWatchDeployment_ImageOverrideWins(t *testing.T) {
	cr := aiCR()
	cr.Spec.AI.Image = &chav1alpha1.ImageSpec{
		Repository: "myco/cha-com",
		Tag:        "custom",
	}
	d := BuildAIWatchDeployment(cr)
	if got := d.Spec.Template.Spec.Containers[0].Image; got != "myco/cha-com:custom" {
		t.Errorf("explicit ai.image not honored; got %q", got)
	}
}

func TestBuildAIWatchDeployment_DefaultArgs(t *testing.T) {
	cr := aiCR()
	d := BuildAIWatchDeployment(cr)
	args := d.Spec.Template.Spec.Containers[0].Args

	mustContain(t, args, "watch")
	mustContain(t, args, "--ai-tier=t0")
	mustContain(t, args, "--ai-endpoint=https://gpu-ai.svc.cluster.local/v1")
	mustContain(t, args, "--ai-model=qwen3.6-35b-a3b-fp8")
	mustContain(t, args, "--interval=60s")

	// Negative checks: optional flags absent when not configured.
	for _, a := range args {
		if strings.HasPrefix(a, "--ai-allow-saas") ||
			strings.HasPrefix(a, "--ai-llm-fixer-matcher") ||
			strings.HasPrefix(a, "--ai-audit-log=") ||
			strings.HasPrefix(a, "--approval-server-url=") ||
			strings.HasPrefix(a, "--memory-") {
			t.Errorf("unexpected default-emitted flag %q", a)
		}
	}
}

func TestBuildAIWatchDeployment_AllOptionalFlagsHonored(t *testing.T) {
	cr := aiCR()
	cr.Spec.AI.Tier = "t1"
	cr.Spec.AI.Interval = "30s"
	cr.Spec.AI.AllowSaaS = true
	cr.Spec.AI.LLMFixerMatcher = true
	cr.Spec.AI.AuditLog = "-"
	cr.Spec.AI.ApprovalServerURL = "https://approval.cha-system.svc"
	cr.Spec.AI.APIKey = &chav1alpha1.AIAPIKeySpec{
		SecretName: "ai-key",
		EnvName:    "OPENAI_API_KEY",
		Header:     "X-API-Key",
	}
	cr.Spec.AI.T3 = &chav1alpha1.AIT3Spec{
		VaultAllowedPrefixes: []string{"secret/cha/", "secret/shared/"},
	}

	d := BuildAIWatchDeployment(cr)
	args := d.Spec.Template.Spec.Containers[0].Args

	mustContain(t, args, "--ai-tier=t1")
	mustContain(t, args, "--interval=30s")
	mustContain(t, args, "--ai-allow-saas")
	mustContain(t, args, "--ai-llm-fixer-matcher")
	mustContain(t, args, "--ai-audit-log=-")
	mustContain(t, args, "--approval-server-url=https://approval.cha-system.svc")
	mustContain(t, args, "--ai-api-key-env=OPENAI_API_KEY")
	mustContain(t, args, "--ai-api-key-header=X-API-Key")
	mustContain(t, args, "--t3-vault-allowed-prefix=secret/cha/")
	mustContain(t, args, "--t3-vault-allowed-prefix=secret/shared/")
}

func TestBuildAIWatchDeployment_MemoryFlags(t *testing.T) {
	cr := aiCR()
	cr.Spec.AI.Memory = &chav1alpha1.AIMemorySpec{
		Enabled: true,
		Embeddings: &chav1alpha1.AIEmbeddingsSpec{
			Endpoint: "https://embeddings.svc/v1",
			Model:    "qwen3-embedding-0.6b",
		},
	}
	d := BuildAIWatchDeployment(cr)
	args := d.Spec.Template.Spec.Containers[0].Args

	// Default storeUrl points at the chart-convention in-namespace
	// Qdrant service even though the operator hasn't stood it up
	// (deferred); operator-managed installs that want memory MUST
	// either install Qdrant via the chart in the same namespace or
	// set spec.ai.memory.storeUrl explicitly.
	mustContain(t, args, "--memory-store-url=http://bionic-rag.cha-system.svc:6333")
	mustContain(t, args, "--memory-embeddings-endpoint=https://embeddings.svc/v1")
	mustContain(t, args, "--memory-embeddings-model=qwen3-embedding-0.6b")
	mustContain(t, args, "--memory-topk=5")
}

func TestBuildAIWatchDeployment_MemoryFlagsHonorOverrides(t *testing.T) {
	cr := aiCR()
	cr.Spec.AI.Memory = &chav1alpha1.AIMemorySpec{
		Enabled:  true,
		StoreURL: "http://external-qdrant.qdrant.svc:6333",
		TopK:     8,
		Embeddings: &chav1alpha1.AIEmbeddingsSpec{
			Model: "text-embedding-3-small",
		},
	}
	d := BuildAIWatchDeployment(cr)
	args := d.Spec.Template.Spec.Containers[0].Args
	mustContain(t, args, "--memory-store-url=http://external-qdrant.qdrant.svc:6333")
	mustContain(t, args, "--memory-topk=8")
}

func TestBuildAIWatchDeployment_APIKeyEnvAndSecretRef(t *testing.T) {
	cr := aiCR()
	cr.Spec.AI.APIKey = &chav1alpha1.AIAPIKeySpec{
		SecretName: "gpu-ai-key",
		// SecretKey + EnvName left empty to exercise defaults.
	}
	d := BuildAIWatchDeployment(cr)
	env := d.Spec.Template.Spec.Containers[0].Env

	var found *string
	for i, e := range env {
		if e.Name == "AI_API_KEY" {
			if e.ValueFrom == nil || e.ValueFrom.SecretKeyRef == nil {
				t.Fatalf("AI_API_KEY env at idx %d missing secretKeyRef", i)
			}
			ref := e.ValueFrom.SecretKeyRef
			if ref.Name != "gpu-ai-key" {
				t.Errorf("secret name=%q want gpu-ai-key", ref.Name)
			}
			if ref.Key != "API_KEY" {
				t.Errorf("secret key=%q want default API_KEY", ref.Key)
			}
			found = &ref.Name
			break
		}
	}
	if found == nil {
		t.Errorf("AI_API_KEY env not set; have: %v", env)
	}
}

func TestBuildAIWatchDeployment_PullSecretsFallbackToOSS(t *testing.T) {
	cr := aiCR()
	cr.Spec.Image.PullSecrets = []string{"oss-pull"}
	d := BuildAIWatchDeployment(cr)
	refs := d.Spec.Template.Spec.ImagePullSecrets
	if len(refs) != 1 || refs[0].Name != "oss-pull" {
		t.Errorf("expected aiwatch to inherit OSS pull secrets; got %v", refs)
	}
}

func TestBuildAIWatchDeployment_PullSecretsAIOverride(t *testing.T) {
	cr := aiCR()
	cr.Spec.Image.PullSecrets = []string{"oss-pull"}
	cr.Spec.AI.Image = &chav1alpha1.ImageSpec{
		Repository:  "myco/cha-com",
		Tag:         "1.0",
		PullSecrets: []string{"chacom-pull"},
	}
	d := BuildAIWatchDeployment(cr)
	refs := d.Spec.Template.Spec.ImagePullSecrets
	if len(refs) != 1 || refs[0].Name != "chacom-pull" {
		t.Errorf("ai.image.pullSecrets should override OSS list; got %v", refs)
	}
}

func TestBuildAIWatchDeployment_ApprovalEnabled_MountsSigningKey(t *testing.T) {
	// When spec.approval.enabled=true, the aiwatch Deployment must mount
	// the Ed25519 signing key and set CHA_SIGNING_KEY_PATH. Without this
	// the cha-com binary exits at startup with
	// "--approval-server-url set but CHA_SIGNING_KEY_PATH is empty".
	cr := aiCR()
	cr.Spec.Approval = &chav1alpha1.ApprovalSpec{Enabled: true}
	d := BuildAIWatchDeployment(cr)
	if d == nil {
		t.Fatal("deployment must not be nil")
	}
	c := d.Spec.Template.Spec.Containers[0]

	var foundEnv, foundMount bool
	for _, e := range c.Env {
		if e.Name == "CHA_SIGNING_KEY_PATH" && e.Value == "/etc/cha/keys/signing.key" {
			foundEnv = true
		}
	}
	for _, m := range c.VolumeMounts {
		if m.Name == "signing-key" && m.MountPath == "/etc/cha/keys" {
			foundMount = true
		}
	}
	if !foundEnv {
		t.Error("CHA_SIGNING_KEY_PATH env not set on aiwatch when approval.enabled=true")
	}
	if !foundMount {
		t.Error("signing-key VolumeMount missing on aiwatch when approval.enabled=true")
	}
	var foundVol bool
	for _, v := range d.Spec.Template.Spec.Volumes {
		if v.Name == "signing-key" {
			foundVol = true
		}
	}
	if !foundVol {
		t.Error("signing-key Volume not added to pod spec when approval.enabled=true")
	}
}

func TestBuildAIWatchDeployment_ApprovalDisabled_NoSigningKey(t *testing.T) {
	cr := aiCR() // no spec.approval
	d := BuildAIWatchDeployment(cr)
	c := d.Spec.Template.Spec.Containers[0]
	for _, e := range c.Env {
		if e.Name == "CHA_SIGNING_KEY_PATH" {
			t.Error("CHA_SIGNING_KEY_PATH should not be set when approval.enabled=false")
		}
	}
	if len(d.Spec.Template.Spec.Volumes) != 0 {
		t.Errorf("no volumes should be added when approval not enabled; got %d", len(d.Spec.Template.Spec.Volumes))
	}
}

func TestNamesFor_IncludesAIWatch(t *testing.T) {
	cr := sampleCR()
	if got := NamesFor(cr).AIWatch; got != "bionic-aiwatch" {
		t.Errorf("AIWatch name=%q want bionic-aiwatch", got)
	}
}

// --- Reconcile-loop wiring ---

func TestReconcile_AIEnabled_CreatesAIWatchDeployment(t *testing.T) {
	cr := aiCR()
	r, c := newReconciler(t, cr)
	reconcileOnce(t, r, cr)
	// Second pass — first pass adds the finalizer and returns early
	// before reconciling children (Phase 1c behavior).
	reconcileOnce(t, r, cr)

	var dep appsv1.Deployment
	if err := c.Get(context.Background(),
		types.NamespacedName{Namespace: "cha-system", Name: "bionic-aiwatch"},
		&dep); err != nil {
		t.Fatalf("aiwatch Deployment not created: %v", err)
	}
	if dep.Spec.Template.Spec.Containers[0].Name != "aiwatch" {
		t.Errorf("container name=%q want aiwatch",
			dep.Spec.Template.Spec.Containers[0].Name)
	}
}

func TestReconcile_AIDisabled_DoesNotCreateAIWatch(t *testing.T) {
	cr := fullCR() // no spec.ai
	r, c := newReconciler(t, cr)
	reconcileOnce(t, r, cr)
	reconcileOnce(t, r, cr)

	var dep appsv1.Deployment
	err := c.Get(context.Background(),
		types.NamespacedName{Namespace: "cha-system", Name: "bionic-aiwatch"},
		&dep)
	if !apierrors.IsNotFound(err) {
		t.Errorf("expected NotFound for AI-off; got err=%v dep=%+v", err, dep)
	}
}

func TestReconcile_AIDisabledAfterCreate_DeletesAIWatch(t *testing.T) {
	cr := aiCR()
	r, c := newReconciler(t, cr)
	reconcileOnce(t, r, cr)
	reconcileOnce(t, r, cr)

	// Flip AI off and reconcile again.
	var stored chav1alpha1.ClusterHealthAutopilot
	_ = c.Get(context.Background(),
		types.NamespacedName{Namespace: "cha-system", Name: "bionic"},
		&stored)
	stored.Spec.AI.Enabled = false
	stored.Generation = 2
	if err := c.Update(context.Background(), &stored); err != nil {
		t.Fatalf("update cr: %v", err)
	}
	reconcileOnce(t, r, &stored)

	var dep appsv1.Deployment
	err := c.Get(context.Background(),
		types.NamespacedName{Namespace: "cha-system", Name: "bionic-aiwatch"},
		&dep)
	if !apierrors.IsNotFound(err) {
		t.Errorf("aiwatch Deployment not deleted after AI disable; got err=%v", err)
	}
}

func TestReconcile_AIEnabled_MissingEndpoint_ReadyFalse(t *testing.T) {
	cr := aiCR()
	cr.Spec.AI.Endpoint = ""
	r, c := newReconciler(t, cr)
	reconcileOnce(t, r, cr)

	cond := readReadyCondition(t, c, cr)
	if cond == nil || cond.Status != metav1.ConditionFalse || cond.Reason != "InvalidSpec" {
		t.Errorf("expected Ready=False/InvalidSpec for missing ai.endpoint; got %+v", cond)
	}
	if !strings.Contains(cond.Message, "spec.ai.endpoint") {
		t.Errorf("Ready.message=%q should reference spec.ai.endpoint", cond.Message)
	}
}

func TestReconcile_AIEnabled_MissingModel_ReadyFalse(t *testing.T) {
	cr := aiCR()
	cr.Spec.AI.Model = ""
	r, c := newReconciler(t, cr)
	reconcileOnce(t, r, cr)

	cond := readReadyCondition(t, c, cr)
	if cond == nil || cond.Status != metav1.ConditionFalse || cond.Reason != "InvalidSpec" {
		t.Errorf("expected Ready=False/InvalidSpec for missing ai.model; got %+v", cond)
	}
}

func TestReconcile_AIMemoryEnabled_MissingEmbeddingsModel_ReadyFalse(t *testing.T) {
	cr := aiCR()
	cr.Spec.AI.Memory = &chav1alpha1.AIMemorySpec{Enabled: true}
	r, c := newReconciler(t, cr)
	reconcileOnce(t, r, cr)

	cond := readReadyCondition(t, c, cr)
	if cond == nil || cond.Status != metav1.ConditionFalse || cond.Reason != "InvalidSpec" {
		t.Errorf("expected Ready=False/InvalidSpec for memory.enabled missing embeddings.model; got %+v", cond)
	}
}

func TestReconcile_AIDisabled_AIWatchRunningCondition_Disabled(t *testing.T) {
	cr := fullCR()
	r, c := newReconciler(t, cr)
	reconcileOnce(t, r, cr)
	reconcileOnce(t, r, cr)

	cond := readCondition(t, c, cr, chav1alpha1.ConditionAIWatchRunning)
	if cond == nil {
		t.Fatal("AIWatchRunning condition not set even when AI is disabled")
	}
	if cond.Status != metav1.ConditionFalse || cond.Reason != "Disabled" {
		t.Errorf("AIWatchRunning=%s/%s; want False/Disabled", cond.Status, cond.Reason)
	}
}

func TestReconcile_AIWatchAvailable_FlipsConditionTrueAndReady(t *testing.T) {
	cr := aiCR()
	r, c := newReconciler(t, cr)
	reconcileOnce(t, r, cr)
	reconcileOnce(t, r, cr)

	// Simulate the aiwatch going Ready.
	var dep appsv1.Deployment
	if err := c.Get(context.Background(),
		types.NamespacedName{Namespace: "cha-system", Name: "bionic-aiwatch"},
		&dep); err != nil {
		t.Fatalf("get aiwatch dep: %v", err)
	}
	dep.Status.AvailableReplicas = 1
	if err := c.Status().Update(context.Background(), &dep); err != nil {
		t.Fatalf("status update: %v", err)
	}

	// Also flip the watcher available (so Ready isn't blocked on
	// WatcherRunning — though Ready doesn't currently AND that in,
	// it's defensive against future fixes).
	var w appsv1.Deployment
	_ = c.Get(context.Background(),
		types.NamespacedName{Namespace: "cha-system", Name: "bionic-watcher"},
		&w)
	w.Status.AvailableReplicas = 1
	_ = c.Status().Update(context.Background(), &w)

	reconcileOnce(t, r, cr)

	cond := readCondition(t, c, cr, chav1alpha1.ConditionAIWatchRunning)
	if cond == nil || cond.Status != metav1.ConditionTrue {
		t.Errorf("AIWatchRunning=%+v; want True", cond)
	}
}

func TestReconcile_AIEnabled_AIWatchMissing_ReadyFalse(t *testing.T) {
	// Even after the operator creates the Deployment, the fake client
	// reports availableReplicas=0 until the test bumps status. With
	// AI enabled, AIWatchRunning=False must propagate to Ready=False.
	cr := aiCR()
	r, c := newReconciler(t, cr)
	reconcileOnce(t, r, cr)
	reconcileOnce(t, r, cr)

	cond := readReadyCondition(t, c, cr)
	if cond == nil {
		t.Fatal("Ready condition missing")
	}
	if cond.Status == metav1.ConditionTrue {
		t.Errorf("Ready=True before aiwatch reports available; got %+v", cond)
	}
}

// --- helpers ---

func readReadyCondition(t *testing.T, c client.Client, cr *chav1alpha1.ClusterHealthAutopilot) *metav1.Condition {
	t.Helper()
	return readCondition(t, c, cr, chav1alpha1.ConditionReady)
}

func readCondition(t *testing.T, c client.Client, cr *chav1alpha1.ClusterHealthAutopilot, ctype string) *metav1.Condition {
	t.Helper()
	var got chav1alpha1.ClusterHealthAutopilot
	if err := c.Get(context.Background(),
		types.NamespacedName{Namespace: cr.Namespace, Name: cr.Name},
		&got); err != nil {
		t.Fatalf("get cr: %v", err)
	}
	for i := range got.Status.Conditions {
		if got.Status.Conditions[i].Type == ctype {
			return &got.Status.Conditions[i]
		}
	}
	return nil
}

// v1.18.0 — extraArgs escape hatch: arbitrary flag list appended
// AFTER the typed args so the operator can pass cha-com flags it
// doesn't yet model (e.g. --cloudflare-feeder, --digest-pin-*).
func TestBuildAIWatchDeployment_ExtraArgs_AppendedAfterTypedArgs(t *testing.T) {
	cr := aiCR()
	cr.Spec.AI.ExtraArgs = []string{
		"--cloudflare-feeder=true",
		"--rag-store-url=http://bionic-rag.cluster-health-autopilot.svc:6333",
		"--digest-pin-proposer=true",
		"--digest-pin-repo-map=docker4zerocool/voice-studio-frontend=Bionic-AI-Solutions/voice-studio-frontend:main",
	}
	d := BuildAIWatchDeployment(cr)
	args := d.Spec.Template.Spec.Containers[0].Args

	for _, want := range cr.Spec.AI.ExtraArgs {
		mustContain(t, args, want)
	}
	// Ordering: extraArgs must come AFTER --ai-tier (typed arg).
	tierIdx := -1
	cfIdx := -1
	for i, a := range args {
		if strings.HasPrefix(a, "--ai-tier=") {
			tierIdx = i
		}
		if a == "--cloudflare-feeder=true" {
			cfIdx = i
		}
	}
	if tierIdx < 0 || cfIdx < 0 || cfIdx < tierIdx {
		t.Errorf("extraArgs must appear AFTER typed args; tierIdx=%d cfIdx=%d", tierIdx, cfIdx)
	}
}

// v1.18.0 — extraEnv escape hatch: secret-backed env vars appended
// after the typed downward-API + APIKey envs.
func TestBuildAIWatchDeployment_ExtraEnv_SecretRefAppended(t *testing.T) {
	cr := aiCR()
	cr.Spec.AI.ExtraEnv = []chav1alpha1.AIExtraEnv{
		{
			Name: "GITHUB_PAT",
			ValueFrom: &chav1alpha1.AIExtraEnvSource{
				SecretKeyRef: &chav1alpha1.AIExtraEnvSecretKeyRef{Name: "cha-github-pat", Key: "PAT"},
			},
		},
		{
			Name:  "LITERAL_VAR",
			Value: "literal-value",
		},
	}
	d := BuildAIWatchDeployment(cr)
	env := d.Spec.Template.Spec.Containers[0].Env

	var foundPAT, foundLiteral bool
	for _, e := range env {
		switch e.Name {
		case "GITHUB_PAT":
			foundPAT = true
			if e.ValueFrom == nil || e.ValueFrom.SecretKeyRef == nil {
				t.Errorf("GITHUB_PAT: want SecretKeyRef; got %+v", e)
			} else if e.ValueFrom.SecretKeyRef.Name != "cha-github-pat" || e.ValueFrom.SecretKeyRef.Key != "PAT" {
				t.Errorf("GITHUB_PAT ref: got name=%q key=%q", e.ValueFrom.SecretKeyRef.Name, e.ValueFrom.SecretKeyRef.Key)
			}
		case "LITERAL_VAR":
			foundLiteral = true
			if e.Value != "literal-value" || e.ValueFrom != nil {
				t.Errorf("LITERAL_VAR: want literal Value; got %+v", e)
			}
		}
	}
	if !foundPAT {
		t.Error("GITHUB_PAT not in env")
	}
	if !foundLiteral {
		t.Error("LITERAL_VAR not in env")
	}
}

// Nil receiver / empty list — defensive coverage.
func TestBuildAIWatchDeployment_ExtraArgsEmpty_NoChange(t *testing.T) {
	cr := aiCR()
	cr.Spec.AI.ExtraArgs = nil
	cr.Spec.AI.ExtraEnv = nil
	d := BuildAIWatchDeployment(cr)
	if d == nil {
		t.Fatal("aiwatch deploy must build with empty extras")
	}
	// args still contain the typed defaults.
	args := d.Spec.Template.Spec.Containers[0].Args
	mustContain(t, args, "--ai-tier=t0")
}
