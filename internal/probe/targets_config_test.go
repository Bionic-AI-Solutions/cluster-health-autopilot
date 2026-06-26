// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package probe

import (
	"context"
	"testing"
)

func TestTargetsFromEnv_Empty(t *testing.T) {
	for _, in := range []string{"", "   ", "\t\n"} {
		if got := TargetsFromEnv(in); got != nil {
			t.Errorf("TargetsFromEnv(%q) = %+v, want nil", in, got)
		}
	}
}

func TestTargetsFromEnv_SingleEntry(t *testing.T) {
	got := TargetsFromEnv("prod/app=billing")
	if len(got) != 1 {
		t.Fatalf("got %d targets, want 1", len(got))
	}
	if got[0].Namespace != "prod" || got[0].Selector != "app=billing" {
		t.Errorf("parse mismatch: %+v", got[0])
	}
	if got[0].Display != "prod/app=billing" {
		t.Errorf("default display should be ns/selector, got %q", got[0].Display)
	}
}

func TestTargetsFromEnv_WithDisplay(t *testing.T) {
	got := TargetsFromEnv("prod/app=billing|Billing API")
	if got[0].Display != "Billing API" {
		t.Errorf("display = %q, want 'Billing API'", got[0].Display)
	}
}

func TestTargetsFromEnv_MultipleEntries(t *testing.T) {
	got := TargetsFromEnv("ns1/app=a; ns2/app=b|B Display ;ns3/k8s-app=c")
	if len(got) != 3 {
		t.Fatalf("got %d targets, want 3", len(got))
	}
	if got[1].Display != "B Display" {
		t.Errorf("entry 2 display = %q", got[1].Display)
	}
}

func TestTargetsFromEnv_SkipsMalformed(t *testing.T) {
	// "no-slash", missing namespace, missing equals — all dropped silently
	// because the env var is typically operator-edited and we'd rather
	// have a partial config than a panic during probe registration.
	got := TargetsFromEnv("no-slash;/app=x;ns/no-equals;prod/app=ok")
	if len(got) != 1 || got[0].Selector != "app=ok" {
		t.Errorf("expected only the valid entry to survive; got %+v", got)
	}
}

const annotatedDeployments = `{
  "apiVersion": "v1", "kind": "List",
  "items": [
    {"apiVersion": "apps/v1", "kind": "Deployment",
     "metadata": {"name": "billing", "namespace": "prod",
                  "annotations": {"srenix.ai/probe-critical": "true"}},
     "spec": {"selector": {"matchLabels": {"app": "billing"}}}},
    {"apiVersion": "apps/v1", "kind": "Deployment",
     "metadata": {"name": "infra", "namespace": "platform",
                  "annotations": {
                    "srenix.ai/probe-critical": "True",
                    "srenix.ai/probe-display": "Platform Infra"
                  }},
     "spec": {"selector": {"matchLabels": {"app.kubernetes.io/name": "infra"}}}},
    {"apiVersion": "apps/v1", "kind": "Deployment",
     "metadata": {"name": "not-critical", "namespace": "prod"},
     "spec": {"selector": {"matchLabels": {"app": "x"}}}}
  ]
}`

const annotatedStatefulSets = `{
  "apiVersion": "v1", "kind": "List",
  "items": [
    {"apiVersion": "apps/v1", "kind": "StatefulSet",
     "metadata": {"name": "kafka", "namespace": "data",
                  "annotations": {"srenix.ai/probe-critical": "true"}},
     "spec": {"selector": {"matchLabels": {"app": "kafka", "role": "broker"}}}}
  ]
}`

func TestTargetsFromAnnotation_PicksUpOnlyAnnotated(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{
		"apps-deployments.json":  annotatedDeployments,
		"apps-statefulsets.json": annotatedStatefulSets,
	})
	got := TargetsFromAnnotation(context.Background(), src)
	if len(got) != 3 {
		t.Fatalf("expected 3 critical workloads (2 Deployments + 1 StatefulSet), got %d: %+v",
			len(got), got)
	}
}

func TestTargetsFromAnnotation_PrefersAppLabel(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{"apps-deployments.json": annotatedDeployments})
	got := TargetsFromAnnotation(context.Background(), src)
	// billing in prod has app=billing — should be selector "app=billing"
	var billing ServiceTarget
	for _, target := range got {
		if target.Namespace == "prod" {
			billing = target
		}
	}
	if billing.Selector != "app=billing" {
		t.Errorf("billing target selector = %q, want app=billing", billing.Selector)
	}
}

func TestTargetsFromAnnotation_FallsBackToK8sName(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{"apps-deployments.json": annotatedDeployments})
	got := TargetsFromAnnotation(context.Background(), src)
	var infra ServiceTarget
	for _, target := range got {
		if target.Namespace == "platform" {
			infra = target
		}
	}
	if infra.Selector != "app.kubernetes.io/name=infra" {
		t.Errorf("infra target selector = %q, want app.kubernetes.io/name=infra", infra.Selector)
	}
	if infra.Display != "Platform Infra" {
		t.Errorf("infra display = %q, want 'Platform Infra'", infra.Display)
	}
}

func TestTargetsFromAnnotation_StatefulSetSupported(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{
		"apps-statefulsets.json": annotatedStatefulSets,
	})
	got := TargetsFromAnnotation(context.Background(), src)
	if len(got) != 1 || got[0].Namespace != "data" {
		t.Errorf("StatefulSet annotation should produce a target; got %+v", got)
	}
}

func TestTargetsFromAnnotation_AcceptsCaseInsensitiveTrue(t *testing.T) {
	// "True" with capital T should still match — guards against operator
	// typos in the annotation value.
	src := loadProbeSrc(t, map[string]string{"apps-deployments.json": annotatedDeployments})
	got := TargetsFromAnnotation(context.Background(), src)
	foundInfra := false
	for _, target := range got {
		if target.Namespace == "platform" {
			foundInfra = true
		}
	}
	if !foundInfra {
		t.Errorf("expected 'True' annotation to be matched case-insensitively; got %+v", got)
	}
}

func TestTargetsFromAnnotation_NothingMatched(t *testing.T) {
	src := loadProbeSrc(t, map[string]string{})
	got := TargetsFromAnnotation(context.Background(), src)
	if got != nil {
		t.Errorf("expected nil for empty snapshot, got %+v", got)
	}
}
