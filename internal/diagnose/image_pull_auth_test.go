// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package diagnose

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/srenix-ai/agentic-sre/internal/snapshot"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// imagePullSrc is a minimal snapshot.Source for ImagePullAuth tests.
type imagePullSrc struct {
	pods   []map[string]any
	events []map[string]any
}

func (s *imagePullSrc) List(_ context.Context, gvr schema.GroupVersionResource, _ string) (*unstructured.UnstructuredList, error) {
	var raw []map[string]any
	switch gvr {
	case snapshot.GVRPod:
		raw = s.pods
	case snapshot.GVREvent:
		raw = s.events
	}
	list := &unstructured.UnstructuredList{}
	for _, m := range raw {
		u := unstructured.Unstructured{Object: m}
		list.Items = append(list.Items, u)
	}
	return list, nil
}

func (s *imagePullSrc) Get(_ context.Context, _ schema.GroupVersionResource, _, _ string) (*unstructured.Unstructured, error) {
	return nil, nil
}

func (s *imagePullSrc) Mode() snapshot.Mode { return snapshot.ModeSnapshot }

func mustParsePullSrc(t *testing.T, raw string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	return m
}

func TestImagePullAuth_NoPods(t *testing.T) {
	src := &imagePullSrc{}
	got := ImagePullAuth{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("expected no diagnostics, got %d", len(got))
	}
}

func TestImagePullAuth_HealthyPod(t *testing.T) {
	src := &imagePullSrc{
		pods: []map[string]any{
			mustParsePullSrc(t, `{
				"metadata": {"namespace":"default","name":"web"},
				"status": {"containerStatuses": [{"name":"app","image":"nginx:latest","state":{"running":{}}}]}
			}`),
		},
	}
	got := ImagePullAuth{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("expected no diagnostics for running pod, got %d", len(got))
	}
}

func TestImagePullAuth_PullBackoffNoAuthEvent(t *testing.T) {
	// Pod is in ImagePullBackOff but the event says "not found" — not an auth error.
	src := &imagePullSrc{
		pods: []map[string]any{
			mustParsePullSrc(t, `{
				"metadata": {"namespace":"prod","name":"api"},
				"status": {"containerStatuses": [{"name":"api","image":"ghcr.io/org/api:v1","state":{"waiting":{"reason":"ImagePullBackOff"}}}]}
			}`),
		},
		events: []map[string]any{
			mustParsePullSrc(t, `{
				"involvedObject": {"kind":"Pod","name":"api"},
				"reason": "Failed",
				"message": "Failed to pull image: manifest unknown",
				"lastTimestamp": "2026-05-04T10:00:00Z"
			}`),
		},
	}
	got := ImagePullAuth{}.Run(context.Background(), src)
	if len(got) != 0 {
		t.Errorf("non-auth pull failure should not produce a diagnostic, got %d", len(got))
	}
}

func TestImagePullAuth_AuthFailureEmitsDiagnostic(t *testing.T) {
	src := &imagePullSrc{
		pods: []map[string]any{
			mustParsePullSrc(t, `{
				"metadata": {"namespace":"prod","name":"api"},
				"status": {"containerStatuses": [{"name":"api","image":"ghcr.io/org/api:v1","state":{"waiting":{"reason":"ImagePullBackOff"}}}]}
			}`),
		},
		events: []map[string]any{
			mustParsePullSrc(t, `{
				"involvedObject": {"kind":"Pod","name":"api"},
				"reason": "Failed",
				"message": "Failed to pull image \"ghcr.io/org/api:v1\": pull access denied, repository does not exist or may require 'docker login': denied",
				"lastTimestamp": "2026-05-04T10:00:00Z"
			}`),
		},
	}
	got := ImagePullAuth{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(got))
	}
	if got[0].Subject != "Pod/prod/api" {
		t.Errorf("unexpected subject: %s", got[0].Subject)
	}
	if !containsAny(got[0].Message, []string{"imagePullSecret", "auth"}) {
		t.Errorf("message should mention imagePullSecret or auth, got: %s", got[0].Message)
	}
}

func TestImagePullAuth_UnauthorizedKeyword(t *testing.T) {
	src := &imagePullSrc{
		pods: []map[string]any{
			mustParsePullSrc(t, `{
				"metadata": {"namespace":"monitoring","name":"prometheus"},
				"status": {"containerStatuses": [{"name":"prom","image":"registry.k8s.io/prometheus:v2","state":{"waiting":{"reason":"ErrImagePull"}}}]}
			}`),
		},
		events: []map[string]any{
			mustParsePullSrc(t, `{
				"involvedObject": {"kind":"Pod","name":"prometheus"},
				"reason": "Failed",
				"message": "Failed to pull image: unauthorized: authentication required",
				"lastTimestamp": "2026-05-04T10:00:00Z"
			}`),
		},
	}
	got := ImagePullAuth{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("expected 1 diagnostic for 'unauthorized', got %d", len(got))
	}
}

func TestImagePullAuth_InitContainerBackoff(t *testing.T) {
	// Auth failure in an init container.
	src := &imagePullSrc{
		pods: []map[string]any{
			mustParsePullSrc(t, `{
				"metadata": {"namespace":"staging","name":"migrate"},
				"status": {
					"containerStatuses": [{"name":"app","image":"myapp:v1","state":{"waiting":{"reason":"PodInitializing"}}}],
					"initContainerStatuses": [{"name":"init","image":"private.registry.io/init:v1","state":{"waiting":{"reason":"ImagePullBackOff"}}}]
				}
			}`),
		},
		events: []map[string]any{
			mustParsePullSrc(t, `{
				"involvedObject": {"kind":"Pod","name":"migrate"},
				"reason": "Failed",
				"message": "Failed to pull image \"private.registry.io/init:v1\": 401 Unauthorized",
				"lastTimestamp": "2026-05-04T10:00:00Z"
			}`),
		},
	}
	got := ImagePullAuth{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Fatalf("expected 1 diagnostic for init container, got %d", len(got))
	}
	if got[0].Subject != "Pod/staging/migrate" {
		t.Errorf("unexpected subject: %s", got[0].Subject)
	}
}

func TestImagePullAuth_DeduplicatesPerPod(t *testing.T) {
	pod := mustParsePullSrc(t, `{
		"metadata": {"namespace":"default","name":"dup"},
		"status": {"containerStatuses": [{"name":"c","image":"img:v1","state":{"waiting":{"reason":"ImagePullBackOff"}}}]}
	}`)
	event := mustParsePullSrc(t, `{
		"involvedObject": {"kind":"Pod","name":"dup"},
		"reason": "Failed",
		"message": "pull access denied",
		"lastTimestamp": "2026-05-04T10:00:00Z"
	}`)
	src := &imagePullSrc{
		pods:   []map[string]any{pod, pod}, // duplicate pod entries
		events: []map[string]any{event},
	}
	got := ImagePullAuth{}.Run(context.Background(), src)
	if len(got) != 1 {
		t.Errorf("should deduplicate to 1 diagnostic, got %d", len(got))
	}
}
