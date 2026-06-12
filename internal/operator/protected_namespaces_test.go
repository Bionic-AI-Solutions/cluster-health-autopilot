// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	"testing"

	chav1alpha1 "github.com/Bionic-AI-Solutions/cluster-health-autopilot/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

// spec.protectedNamespacesExtra → CHA_PROTECTED_NAMESPACES_EXTRA on
// every workload that hosts an act-side guard: the watcher Deployment
// and remediate CronJob (internal/fix.IsProtectedNamespace), the
// diagnose CronJob (same binary/catalog), and the aiwatch Deployment
// (pkg/ai.IsProtectedNamespace linked into cha-com).

func envValue(env []corev1.EnvVar, name string) (string, bool) {
	for _, e := range env {
		if e.Name == name {
			return e.Value, true
		}
	}
	return "", false
}

func protectedCR() *chav1alpha1.ClusterHealthAutopilot {
	cr := sampleCR()
	cr.Spec.ProtectedNamespacesExtra = []string{" prod-payments ", "tenant-a", "", "prod-payments"}
	return cr
}

func TestProtectedNamespacesExtra_WatcherEnv(t *testing.T) {
	cr := protectedCR()
	cr.Spec.Watcher = &chav1alpha1.WatcherSpec{Enabled: true}
	d := BuildWatcherDeployment(cr)
	got, ok := envValue(d.Spec.Template.Spec.Containers[0].Env, "CHA_PROTECTED_NAMESPACES_EXTRA")
	if !ok {
		t.Fatal("watcher env missing CHA_PROTECTED_NAMESPACES_EXTRA")
	}
	if got != "prod-payments,tenant-a" {
		t.Errorf("watcher CHA_PROTECTED_NAMESPACES_EXTRA = %q; want trimmed+deduped %q", got, "prod-payments,tenant-a")
	}
}

func TestProtectedNamespacesExtra_DiagnoseAndRemediateEnv(t *testing.T) {
	cr := protectedCR()
	cr.Spec.Diagnose = &chav1alpha1.DiagnoseSpec{Enabled: true}
	cr.Spec.Remediate = &chav1alpha1.RemediateSpec{Enabled: true}

	for name, env := range map[string][]corev1.EnvVar{
		"diagnose":  BuildDiagnoseCronJob(cr).Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env,
		"remediate": BuildRemediateCronJob(cr).Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env,
	} {
		got, ok := envValue(env, "CHA_PROTECTED_NAMESPACES_EXTRA")
		if !ok {
			t.Errorf("%s env missing CHA_PROTECTED_NAMESPACES_EXTRA", name)
			continue
		}
		if got != "prod-payments,tenant-a" {
			t.Errorf("%s CHA_PROTECTED_NAMESPACES_EXTRA = %q; want %q", name, got, "prod-payments,tenant-a")
		}
	}
}

func TestProtectedNamespacesExtra_AIWatchEnv(t *testing.T) {
	cr := protectedCR()
	cr.Spec.AI = &chav1alpha1.AISpec{
		Enabled:  true,
		Endpoint: "https://gpu-ai.example/v1",
		Model:    "qwen3.6-35b-a3b-fp8",
	}
	d := BuildAIWatchDeployment(cr)
	got, ok := envValue(d.Spec.Template.Spec.Containers[0].Env, "CHA_PROTECTED_NAMESPACES_EXTRA")
	if !ok {
		t.Fatal("aiwatch env missing CHA_PROTECTED_NAMESPACES_EXTRA — the cha-com AI validator floor would diverge from the fixer guard")
	}
	if got != "prod-payments,tenant-a" {
		t.Errorf("aiwatch CHA_PROTECTED_NAMESPACES_EXTRA = %q; want %q", got, "prod-payments,tenant-a")
	}
}

func TestProtectedNamespacesExtra_AbsentWhenUnset(t *testing.T) {
	cr := sampleCR() // no ProtectedNamespacesExtra
	cr.Spec.Watcher = &chav1alpha1.WatcherSpec{Enabled: true}
	cr.Spec.Diagnose = &chav1alpha1.DiagnoseSpec{Enabled: true}
	cr.Spec.Remediate = &chav1alpha1.RemediateSpec{Enabled: true}
	cr.Spec.AI = &chav1alpha1.AISpec{Enabled: true, Endpoint: "https://x/v1", Model: "m"}

	for name, env := range map[string][]corev1.EnvVar{
		"watcher":   BuildWatcherDeployment(cr).Spec.Template.Spec.Containers[0].Env,
		"diagnose":  BuildDiagnoseCronJob(cr).Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env,
		"remediate": BuildRemediateCronJob(cr).Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env,
		"aiwatch":   BuildAIWatchDeployment(cr).Spec.Template.Spec.Containers[0].Env,
	} {
		if v, ok := envValue(env, "CHA_PROTECTED_NAMESPACES_EXTRA"); ok {
			t.Errorf("%s renders CHA_PROTECTED_NAMESPACES_EXTRA=%q with no extras set; want absent (byte-identical legacy render)", name, v)
		}
	}
}

func TestProtectedNamespacesExtra_AllGarbageRendersNothing(t *testing.T) {
	cr := sampleCR()
	cr.Spec.ProtectedNamespacesExtra = []string{"", "   ", " "}
	cr.Spec.Watcher = &chav1alpha1.WatcherSpec{Enabled: true}
	d := BuildWatcherDeployment(cr)
	if v, ok := envValue(d.Spec.Template.Spec.Containers[0].Env, "CHA_PROTECTED_NAMESPACES_EXTRA"); ok {
		t.Errorf("all-garbage extras rendered CHA_PROTECTED_NAMESPACES_EXTRA=%q; want absent", v)
	}
}
