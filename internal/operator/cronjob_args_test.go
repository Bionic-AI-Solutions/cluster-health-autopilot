// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	"reflect"
	"strings"
	"testing"

	chav1alpha1 "github.com/srenix-ai/agentic-sre/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

// PRODUCTION BUG (fixed v1.26.0): buildCronJobCommon appended
// alertingArgs() — which emits the WATCH-ONLY flags --alertmanager-url,
// --cluster-name, --slack-alerts, --slack-critical — to BOTH CronJobs.
// `srenix diagnose` / `srenix remediate` do not register those flags, so on
// any cluster with spec.alerting configured the bionic-diagnose /
// bionic-remediate Jobs exited 1 with "unknown flag" on EVERY run; the
// CronJobs had never succeeded. The Helm chart templates
// (cronjob-diagnose.yaml / cronjob-remediate.yaml) are the reference:
// diagnose renders `--format=daily` + optional --slack-healthinfo;
// remediate renders no alerting flags at all.
//
// These tests pin the EXACT arg lists so a watch-only flag can never
// leak back in. The bug-CLASS guard (every operator-rendered arg must
// be registered on the real cobra subcommand) lives in
// cmd/srenix/operatorflags_test.go.

// fullAlerting returns an AlertingSpec with every channel configured —
// the shape that triggered the production bug.
func fullAlerting() *chav1alpha1.AlertingSpec {
	return &chav1alpha1.AlertingSpec{
		Alertmanager: &chav1alpha1.AlertmanagerSpec{
			URL:         "http://alertmanager.pg.svc:9093",
			ClusterName: "bionic-cluster",
		},
		Slack: &chav1alpha1.SlackSpec{
			Alerts:     &chav1alpha1.SlackChannelSpec{SecretName: "srenix-alerts"},
			Critical:   &chav1alpha1.SlackChannelSpec{SecretName: "srenix-critical"},
			HealthInfo: &chav1alpha1.SlackChannelSpec{SecretName: "srenix-healthinfo"},
		},
	}
}

func TestBuildDiagnoseCronJob_ArgsExact_FullAlerting(t *testing.T) {
	cr := sampleCR()
	cr.Spec.Diagnose = &chav1alpha1.DiagnoseSpec{Enabled: true}
	cr.Spec.Alerting = fullAlerting()

	c := BuildDiagnoseCronJob(cr)
	if c == nil {
		t.Fatal("enabled diagnose must produce a CronJob")
	}
	args := c.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Args
	want := []string{"diagnose", "--live", "--format=daily", "--slack-healthinfo=$(SLACK_HEALTHINFO_URL)"}
	if !reflect.DeepEqual(args, want) {
		t.Errorf("diagnose CronJob args = %v, want %v (watch-only flags must NOT leak; chart cronjob-diagnose.yaml is the reference)", args, want)
	}
}

func TestBuildDiagnoseCronJob_ArgsExact_NoHealthinfo(t *testing.T) {
	cr := sampleCR()
	cr.Spec.Diagnose = &chav1alpha1.DiagnoseSpec{Enabled: true}
	cr.Spec.Alerting = fullAlerting()
	cr.Spec.Alerting.Slack.HealthInfo = nil

	args := BuildDiagnoseCronJob(cr).Spec.JobTemplate.Spec.Template.Spec.Containers[0].Args
	want := []string{"diagnose", "--live", "--format=daily"}
	if !reflect.DeepEqual(args, want) {
		t.Errorf("diagnose CronJob args = %v, want %v", args, want)
	}
}

func TestBuildRemediateCronJob_ArgsExact_FullAlerting(t *testing.T) {
	cr := sampleCR()
	cr.Spec.Remediate = &chav1alpha1.RemediateSpec{Enabled: true}
	cr.Spec.Alerting = fullAlerting()

	c := BuildRemediateCronJob(cr)
	if c == nil {
		t.Fatal("enabled remediate must produce a CronJob")
	}
	args := c.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Args
	want := []string{"remediate", "--live"}
	if !reflect.DeepEqual(args, want) {
		t.Errorf("remediate CronJob args = %v, want %v (no alerting flags; chart cronjob-remediate.yaml is the reference)", args, want)
	}
}

func TestBuildRemediateCronJob_ArgsExact_DryRunFullAlerting(t *testing.T) {
	cr := sampleCR()
	cr.Spec.Remediate = &chav1alpha1.RemediateSpec{Enabled: true, DryRun: true}
	cr.Spec.Alerting = fullAlerting()

	args := BuildRemediateCronJob(cr).Spec.JobTemplate.Spec.Template.Spec.Containers[0].Args
	want := []string{"remediate", "--live", "--dry-run=true"}
	if !reflect.DeepEqual(args, want) {
		t.Errorf("remediate CronJob args = %v, want %v", args, want)
	}
}

// The diagnose container must carry ONLY the SLACK_HEALTHINFO_URL env
// (the one its --slack-healthinfo arg expands); remediate carries no
// SLACK_* env at all — mirroring the chart, which renders
// srenix.slackHealthinfoEnv on diagnose only.
func TestCronJobAlertingEnv_RoleScoped(t *testing.T) {
	cr := sampleCR()
	cr.Spec.Diagnose = &chav1alpha1.DiagnoseSpec{Enabled: true}
	cr.Spec.Remediate = &chav1alpha1.RemediateSpec{Enabled: true}
	cr.Spec.Alerting = fullAlerting()

	diagEnv := BuildDiagnoseCronJob(cr).Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env
	if !hasEnv(diagEnv, "SLACK_HEALTHINFO_URL") {
		t.Errorf("diagnose CronJob env missing SLACK_HEALTHINFO_URL; have %v", envNames(diagEnv))
	}
	for _, n := range envNames(diagEnv) {
		if n != "SLACK_HEALTHINFO_URL" && strings.HasPrefix(n, "SLACK_") {
			t.Errorf("diagnose CronJob carries stray alerting env %s (only SLACK_HEALTHINFO_URL is consumed)", n)
		}
	}

	remEnv := BuildRemediateCronJob(cr).Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env
	for _, n := range envNames(remEnv) {
		if strings.HasPrefix(n, "SLACK_") {
			t.Errorf("remediate CronJob carries alerting env %s (remediate consumes none)", n)
		}
	}
}

// The WATCHER must not carry a SLACK_HEALTHINFO_URL secretKeyRef:
// nothing in `srenix watch` reads it (--slack-healthinfo is registered on
// the diagnose subcommand only, and there is no direct env read), and
// secretRefEnv emits a NON-optional secretKeyRef — so when the
// healthinfo secret is absent the kubelet hard-fails watcher pod
// creation (CreateContainerConfigError) over an env var that could
// never be consumed. The chart's watcher-deployment.yaml is the
// reference: it renders srenix.slackAlertsEnv + srenix.slackCriticalEnv
// only (srenix.slackHealthinfoEnv appears solely on cronjob-diagnose).
func TestWatcherDeploymentEnv_NoHealthinfoSecretRef(t *testing.T) {
	cr := sampleCR()
	cr.Spec.Watcher = &chav1alpha1.WatcherSpec{Enabled: true}
	cr.Spec.Alerting = fullAlerting()

	d := BuildWatcherDeployment(cr)
	if d == nil {
		t.Fatal("enabled watcher must produce a Deployment")
	}
	env := d.Spec.Template.Spec.Containers[0].Env
	if hasEnv(env, "SLACK_HEALTHINFO_URL") {
		t.Errorf("watcher Deployment carries SLACK_HEALTHINFO_URL — `srenix watch` never reads it, and the non-optional secretKeyRef hard-fails pod creation when the secret is absent; have %v", envNames(env))
	}
	// The two channels the watcher DOES consume must still be present.
	for _, want := range []string{"SLACK_ALERTS_URL", "SLACK_CRITICAL_URL"} {
		if !hasEnv(env, want) {
			t.Errorf("watcher Deployment env missing %s (consumed via --slack-alerts/--slack-critical $(...) expansion); have %v", want, envNames(env))
		}
	}
}

func hasEnv(env []corev1.EnvVar, name string) bool {
	for _, e := range env {
		if e.Name == name {
			return true
		}
	}
	return false
}

func envNames(env []corev1.EnvVar) []string {
	out := make([]string, 0, len(env))
	for _, e := range env {
		out = append(out, e.Name)
	}
	return out
}
