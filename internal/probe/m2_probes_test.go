// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package probe

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/srenix-ai/agentic-sre/pkg/snapshot"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// errSource is a snapshot.Source whose List always errors — used to
// exercise the "CRD not installed" → SKIPPED path that the live
// dynamic client produces (the file source returns empty, not error,
// so it can't drive that branch).
type errSource struct{}

func (errSource) List(_ context.Context, _ schema.GroupVersionResource, _ string) (*unstructured.UnstructuredList, error) {
	return nil, errors.New(`the server could not find the requested resource`)
}
func (errSource) Get(_ context.Context, _ schema.GroupVersionResource, _, _ string) (*unstructured.Unstructured, error) {
	return nil, errors.New("not found")
}
func (errSource) Mode() snapshot.Mode { return snapshot.ModeLive }

// --- Kong ---

const kongProgrammedFalse = `{
  "apiVersion": "configuration.konghq.com/v1", "kind": "KongPluginList",
  "items": [
    {"apiVersion": "configuration.konghq.com/v1", "kind": "KongPlugin",
     "metadata": {"name": "rate-limit", "namespace": "gw"},
     "status": {"conditions": [{"type": "Programmed", "status": "False", "reason": "InvalidConfig", "message": "limit must be > 0"}]}}
  ]
}`

const kongHealthy = `{
  "apiVersion": "configuration.konghq.com/v1", "kind": "KongPluginList",
  "items": [
    {"apiVersion": "configuration.konghq.com/v1", "kind": "KongPlugin",
     "metadata": {"name": "cors", "namespace": "gw"},
     "status": {"conditions": [{"type": "Programmed", "status": "True"}]}}
  ]
}`

func TestKong_SkippedWhenCRDAbsent(t *testing.T) {
	r := Kong{}.Run(context.Background(), errSource{})
	if r.Component.Status != "SKIPPED" {
		t.Errorf("status=%s want SKIPPED", r.Component.Status)
	}
}

func TestKong_ProgrammedFalse_Critical(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{"kong.json": kongProgrammedFalse})
	r := Kong{}.Run(context.Background(), src)
	if r.Component.Status != "CRITICAL" {
		t.Fatalf("status=%s want CRITICAL (detail=%s)", r.Component.Status, r.Component.Detail)
	}
	if len(r.Findings) != 1 || !strings.Contains(r.Findings[0].Message, "Programmed=False") {
		t.Errorf("expected Programmed=False finding; got: %+v", r.Findings)
	}
}

func TestKong_AllProgrammed_Healthy(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{"kong.json": kongHealthy})
	r := Kong{}.Run(context.Background(), src)
	if r.Component.Status != "HEALTHY" {
		t.Errorf("status=%s want HEALTHY", r.Component.Status)
	}
}

// --- HPAScaling ---

const hpaScalingFalse = `{
  "apiVersion": "autoscaling/v2", "kind": "HorizontalPodAutoscalerList",
  "items": [
    {"apiVersion": "autoscaling/v2", "kind": "HorizontalPodAutoscaler",
     "metadata": {"name": "api", "namespace": "app"},
     "status": {"conditions": [{"type": "ScalingActive", "status": "False", "reason": "FailedGetResourceMetric", "message": "no metrics"}]}}
  ]
}`

const hpaHealthy = `{
  "apiVersion": "autoscaling/v2", "kind": "HorizontalPodAutoscalerList",
  "items": [
    {"apiVersion": "autoscaling/v2", "kind": "HorizontalPodAutoscaler",
     "metadata": {"name": "api", "namespace": "app"},
     "status": {"conditions": [{"type": "ScalingActive", "status": "True"}, {"type": "AbleToScale", "status": "True"}]}}
  ]
}`

// ScalingActive=False with reason=ScalingDisabled is the expected state
// for an intentionally scaled-to-zero / KEDA workload — must be Warning,
// not Critical (false-positive de-noising).
const hpaScalingDisabled = `{
  "apiVersion": "autoscaling/v2", "kind": "HorizontalPodAutoscalerList",
  "items": [
    {"apiVersion": "autoscaling/v2", "kind": "HorizontalPodAutoscaler",
     "metadata": {"name": "keda-hpa-comfyui", "namespace": "comfyui"},
     "status": {"conditions": [{"type": "ScalingActive", "status": "False", "reason": "ScalingDisabled", "message": "scaling is disabled since the replica count of the target is zero"}]}}
  ]
}`

func TestHPAScaling_NoHPAs_Healthy(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{})
	r := HPAScaling{}.Run(context.Background(), src)
	if r.Component.Status != "HEALTHY" {
		t.Errorf("status=%s want HEALTHY (empty cluster)", r.Component.Status)
	}
}

func TestHPAScaling_ScalingActiveFalse_Critical(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{"hpa.json": hpaScalingFalse})
	r := HPAScaling{}.Run(context.Background(), src)
	if r.Component.Status != "CRITICAL" {
		t.Fatalf("status=%s want CRITICAL", r.Component.Status)
	}
	if len(r.Findings) != 1 || !strings.Contains(r.Findings[0].Message, "ScalingActive=False") {
		t.Errorf("expected ScalingActive=False finding; got: %+v", r.Findings)
	}
}

func TestHPAScaling_ScalingDisabled_Warning(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{"hpa.json": hpaScalingDisabled})
	r := HPAScaling{}.Run(context.Background(), src)
	if r.Component.Status == "CRITICAL" {
		t.Fatalf("ScalingDisabled must not be CRITICAL (intentional scale-to-zero); got %s", r.Component.Status)
	}
	if len(r.Findings) != 1 || r.Findings[0].Severity != SeverityWarning {
		t.Fatalf("expected one Warning finding; got: %+v", r.Findings)
	}
	if !strings.Contains(r.Findings[0].Message, "scale-to-zero") {
		t.Errorf("warning message should explain the scale-to-zero expectation; got: %q", r.Findings[0].Message)
	}
}

func TestHPAScaling_Healthy_NoFindings(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{"hpa.json": hpaHealthy})
	r := HPAScaling{}.Run(context.Background(), src)
	if r.Component.Status != "HEALTHY" {
		t.Errorf("status=%s want HEALTHY", r.Component.Status)
	}
}

// --- ArgoCDApplication ---

const argoDegraded = `{
  "apiVersion": "argoproj.io/v1alpha1", "kind": "ApplicationList",
  "items": [
    {"apiVersion": "argoproj.io/v1alpha1", "kind": "Application",
     "metadata": {"name": "web", "namespace": "argocd"},
     "status": {"sync": {"status": "Synced"}, "health": {"status": "Degraded"}}}
  ]
}`

const argoOutOfSync = `{
  "apiVersion": "argoproj.io/v1alpha1", "kind": "ApplicationList",
  "items": [
    {"apiVersion": "argoproj.io/v1alpha1", "kind": "Application",
     "metadata": {"name": "web", "namespace": "argocd"},
     "status": {"sync": {"status": "OutOfSync"}, "health": {"status": "Healthy"}}}
  ]
}`

const argoHealthy = `{
  "apiVersion": "argoproj.io/v1alpha1", "kind": "ApplicationList",
  "items": [
    {"apiVersion": "argoproj.io/v1alpha1", "kind": "Application",
     "metadata": {"name": "web", "namespace": "argocd"},
     "status": {"sync": {"status": "Synced"}, "health": {"status": "Healthy"}}}
  ]
}`

func TestArgoApp_SkippedWhenCRDAbsent(t *testing.T) {
	r := ArgoCDApplication{}.Run(context.Background(), errSource{})
	if r.Component.Status != "SKIPPED" {
		t.Errorf("status=%s want SKIPPED", r.Component.Status)
	}
}

func TestArgoApp_Degraded_Critical(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{"argo.json": argoDegraded})
	r := ArgoCDApplication{}.Run(context.Background(), src)
	if r.Component.Status != "CRITICAL" {
		t.Fatalf("status=%s want CRITICAL", r.Component.Status)
	}
	if len(r.Findings) != 1 || !strings.Contains(r.Findings[0].Message, "health=Degraded") {
		t.Errorf("expected health=Degraded finding; got: %+v", r.Findings)
	}
}

func TestArgoApp_OutOfSync_Warning(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{"argo.json": argoOutOfSync})
	r := ArgoCDApplication{}.Run(context.Background(), src)
	if r.Component.Status != "DEGRADED" {
		t.Fatalf("status=%s want DEGRADED", r.Component.Status)
	}
	if len(r.Findings) != 1 || r.Findings[0].Severity != SeverityWarning {
		t.Errorf("OutOfSync should be warning; got: %+v", r.Findings)
	}
}

func TestArgoApp_Healthy_NoFindings(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{"argo.json": argoHealthy})
	r := ArgoCDApplication{}.Run(context.Background(), src)
	if r.Component.Status != "HEALTHY" {
		t.Errorf("status=%s want HEALTHY", r.Component.Status)
	}
}

// --- Velero ---

func veleroBackupJSON(schedule, phase string, ageHours int) string {
	created := time.Now().Add(-time.Duration(ageHours) * time.Hour).UTC().Format(time.RFC3339)
	return `{
  "apiVersion": "velero.io/v1", "kind": "BackupList",
  "items": [
    {"apiVersion": "velero.io/v1", "kind": "Backup",
     "metadata": {"name": "` + schedule + `-20260528", "namespace": "velero",
                  "labels": {"velero.io/schedule-name": "` + schedule + `"},
                  "creationTimestamp": "` + created + `"},
     "status": {"phase": "` + phase + `"}}
  ]
}`
}

func TestVelero_SkippedWhenCRDAbsent(t *testing.T) {
	r := Velero{}.Run(context.Background(), errSource{})
	if r.Component.Status != "SKIPPED" {
		t.Errorf("status=%s want SKIPPED", r.Component.Status)
	}
}

func TestVelero_RecentCompleted_Healthy(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{"velero.json": veleroBackupJSON("daily", "Completed", 2)})
	r := Velero{}.Run(context.Background(), src)
	if r.Component.Status != "HEALTHY" {
		t.Errorf("status=%s want HEALTHY (2h-old completed backup)", r.Component.Status)
	}
}

func TestVelero_Failed_Critical(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{"velero.json": veleroBackupJSON("daily", "Failed", 2)})
	r := Velero{}.Run(context.Background(), src)
	if r.Component.Status != "CRITICAL" {
		t.Fatalf("status=%s want CRITICAL", r.Component.Status)
	}
	if len(r.Findings) != 1 || !strings.Contains(r.Findings[0].Message, "phase=Failed") {
		t.Errorf("expected phase=Failed finding; got: %+v", r.Findings)
	}
}

func TestVelero_StaleCompleted_Critical(t *testing.T) {
	// Completed but 30h old > 26h SLA.
	src := loadProbeSrc(t, map[string]string{"velero.json": veleroBackupJSON("daily", "Completed", 30)})
	r := Velero{}.Run(context.Background(), src)
	if r.Component.Status != "CRITICAL" {
		t.Fatalf("status=%s want CRITICAL (stale backup)", r.Component.Status)
	}
	if !strings.Contains(r.Findings[0].Message, "more than") {
		t.Errorf("expected stale-SLA finding; got: %+v", r.Findings)
	}
}

func TestVelero_StuckInProgress_Warning(t *testing.T) {
	// InProgress for 6h > 4h stuck threshold.
	src := loadProbeSrc(t, map[string]string{"velero.json": veleroBackupJSON("daily", "InProgress", 6)})
	r := Velero{}.Run(context.Background(), src)
	if r.Component.Status != "DEGRADED" {
		t.Fatalf("status=%s want DEGRADED", r.Component.Status)
	}
	if r.Findings[0].Severity != SeverityWarning {
		t.Errorf("stuck InProgress should be warning; got: %+v", r.Findings)
	}
}

func TestVelero_CustomSLA(t *testing.T) {
	// 10h-old completed backup with a 6h SLA override → critical.
	src := loadProbeSrc(t, map[string]string{"velero.json": veleroBackupJSON("daily", "Completed", 10)})
	r := Velero{BackupSLA: 6 * time.Hour}.Run(context.Background(), src)
	if r.Component.Status != "CRITICAL" {
		t.Errorf("status=%s want CRITICAL with 6h SLA", r.Component.Status)
	}
}

func TestM2Probes_NamesStable(t *testing.T) {
	cases := map[string]string{
		Kong{}.Name():              "Kong",
		HPAScaling{}.Name():        "HPAScaling",
		ArgoCDApplication{}.Name(): "ArgoCD-Application",
		Velero{}.Name():            "Velero",
	}
	for got, want := range cases {
		if got != want {
			t.Errorf("Name()=%q want %q", got, want)
		}
	}
}
