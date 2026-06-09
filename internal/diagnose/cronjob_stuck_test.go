// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package diagnose

import (
	"context"
	"strings"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func makeCronJob(ns, name, schedule string, suspend bool, lastSuccessAgo, lastScheduleAgo, createdAgo time.Duration) unstructured.Unstructured {
	u := unstructured.Unstructured{}
	u.SetAPIVersion("batch/v1")
	u.SetKind("CronJob")
	u.SetNamespace(ns)
	u.SetName(name)
	u.SetCreationTimestamp(metav1.Time{Time: time.Now().Add(-createdAgo)})
	_ = unstructured.SetNestedField(u.Object, schedule, "spec", "schedule")
	_ = unstructured.SetNestedField(u.Object, suspend, "spec", "suspend")
	if lastSuccessAgo > 0 {
		_ = unstructured.SetNestedField(u.Object,
			time.Now().Add(-lastSuccessAgo).UTC().Format(time.RFC3339),
			"status", "lastSuccessfulTime")
	}
	if lastScheduleAgo > 0 {
		_ = unstructured.SetNestedField(u.Object,
			time.Now().Add(-lastScheduleAgo).UTC().Format(time.RFC3339),
			"status", "lastScheduleTime")
	}
	return u
}

func TestCronJobStuck_NoSuccessInOver24h_Warning(t *testing.T) {
	src := &memSourceDD{byResource: map[string][]unstructured.Unstructured{
		"cronjobs": {
			makeCronJob("data", "etl-nightly", "0 2 * * *", false, 30*time.Hour, 30*time.Hour, 14*24*time.Hour),
		},
	}}
	got := CronJobStuck{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("expected 1 diagnostic; got %d", len(got))
	}
	if got[0].Severity != "warning" {
		t.Errorf("expected warning; got %q", got[0].Severity)
	}
	if !strings.Contains(got[0].Message, "30h") {
		t.Errorf("message should cite age; got %q", got[0].Message)
	}
}

func TestCronJobStuck_NeverSucceeded_Critical(t *testing.T) {
	// CronJob created 3 days ago + has never had a successful run.
	src := &memSourceDD{byResource: map[string][]unstructured.Unstructured{
		"cronjobs": {
			makeCronJob("data", "etl-broken", "0 2 * * *", false, 0, 0, 3*24*time.Hour),
		},
	}}
	got := CronJobStuck{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("expected 1 diagnostic; got %d", len(got))
	}
	if got[0].Severity != "critical" {
		t.Errorf("never-succeeded must be critical; got %q", got[0].Severity)
	}
	if !strings.Contains(got[0].Message, "NEVER had a successful run") {
		t.Errorf("message should distinguish never-succeeded; got %q", got[0].Message)
	}
}

func TestCronJobStuck_Suspended_Warning(t *testing.T) {
	src := &memSourceDD{byResource: map[string][]unstructured.Unstructured{
		"cronjobs": {
			makeCronJob("data", "etl-paused", "0 2 * * *", true, 5*24*time.Hour, 5*24*time.Hour, 7*24*time.Hour),
		},
	}}
	got := CronJobStuck{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("expected 1 diagnostic; got %d", len(got))
	}
	if !strings.Contains(got[0].Message, "spec.suspend=true") {
		t.Errorf("message should call out suspended state; got %q", got[0].Message)
	}
}

func TestCronJobStuck_HealthyRecentSuccess_NoFire(t *testing.T) {
	// Last success was 1h ago — well within grace.
	src := &memSourceDD{byResource: map[string][]unstructured.Unstructured{
		"cronjobs": {
			makeCronJob("data", "etl-healthy", "0 * * * *", false, time.Hour, time.Hour, 14*24*time.Hour),
		},
	}}
	got := CronJobStuck{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("healthy CronJob must NOT fire; got %+v", got)
	}
}

func TestCronJobStuck_NewlyCreatedNoFire(t *testing.T) {
	// CronJob created 1h ago with no status — too young to fire.
	src := &memSourceDD{byResource: map[string][]unstructured.Unstructured{
		"cronjobs": {
			makeCronJob("data", "etl-new", "0 2 * * *", false, 0, 0, time.Hour),
		},
	}}
	got := CronJobStuck{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("brand-new CronJob must NOT fire; got %+v", got)
	}
}

func TestCronJobStuck_Name(t *testing.T) {
	if (CronJobStuck{}).Name() != "CronJobStuck" {
		t.Error("Name mismatch")
	}
}
