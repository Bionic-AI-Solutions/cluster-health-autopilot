// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package probe

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/srenix-ai/agentic-sre/internal/snapshot"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// fakeLiveSource is a minimal snapshot.Source that reports ModeLive so the
// Endpoints probe runs its network path. It returns empty lists for any GVR.
type fakeLiveSource struct{}

func (fakeLiveSource) Mode() snapshot.Mode { return snapshot.ModeLive }
func (fakeLiveSource) List(context.Context, schema.GroupVersionResource, string) (*unstructured.UnstructuredList, error) {
	return &unstructured.UnstructuredList{}, nil
}
func (fakeLiveSource) Get(context.Context, schema.GroupVersionResource, string, string) (*unstructured.Unstructured, error) {
	return nil, nil
}

func newTestProbe(srvURL string) Endpoints {
	p := NewEndpoints(
		[]EndpointTarget{{URL: srvURL, Name: "test"}},
		DiscoveryOptions{Enabled: false},
	)
	p.Timeout = 500 * time.Millisecond
	p.RetryOnFlake = false // disabled for deterministic suppression tests
	return p
}

// TestEndpoints_FlakeSuppressed: a single transient failure must NOT promote
// the component to CRITICAL — it should appear as a Warning-level finding.
func TestEndpoints_FlakeSuppressed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, _ := w.(http.Hijacker)
		conn, _, _ := hj.Hijack()
		_ = conn.Close()
	}))
	defer srv.Close()

	p := newTestProbe(srv.URL)
	r := p.Run(context.Background(), fakeLiveSource{})

	if r.Component.Status != "HEALTHY" {
		t.Fatalf("first failure should not promote to CRITICAL; got status=%s detail=%s",
			r.Component.Status, r.Component.Detail)
	}
	if len(r.Findings) != 1 || r.Findings[0].Severity != SeverityWarning {
		t.Errorf("expected 1 warning-level finding; got %+v", r.Findings)
	}
}

// TestEndpoints_PromotesAfterTwo: second consecutive failure must escalate
// to CRITICAL on Status and SeverityCritical on the Finding.
func TestEndpoints_PromotesAfterTwo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, _ := w.(http.Hijacker)
		conn, _, _ := hj.Hijack()
		_ = conn.Close()
	}))
	defer srv.Close()

	p := newTestProbe(srv.URL)
	r1 := p.Run(context.Background(), fakeLiveSource{})
	if r1.Component.Status != "HEALTHY" {
		t.Fatalf("cycle 1 should suppress; got %s", r1.Component.Status)
	}
	r2 := p.Run(context.Background(), fakeLiveSource{})
	if r2.Component.Status != "CRITICAL" {
		t.Fatalf("cycle 2 should escalate to CRITICAL; got %s", r2.Component.Status)
	}
	if len(r2.Findings) != 1 || r2.Findings[0].Severity != SeverityCritical {
		t.Errorf("expected 1 critical finding on cycle 2; got %+v", r2.Findings)
	}
}

// TestEndpoints_SuccessResetsStreak: a healthy probe between failures must
// reset the counter so a later isolated failure is again suppressed.
func TestEndpoints_SuccessResetsStreak(t *testing.T) {
	var fail atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if fail.Load() {
			hj, _ := w.(http.Hijacker)
			conn, _, _ := hj.Hijack()
			_ = conn.Close()
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	p := newTestProbe(srv.URL)

	fail.Store(true)
	if s := p.Run(context.Background(), fakeLiveSource{}).Component.Status; s != "HEALTHY" {
		t.Fatalf("cycle 1 should be suppressed; got %s", s)
	}
	fail.Store(false)
	if r2 := p.Run(context.Background(), fakeLiveSource{}); r2.Component.Status != "HEALTHY" || len(r2.Findings) != 0 {
		t.Fatalf("cycle 2 should be clean; got status=%s findings=%+v", r2.Component.Status, r2.Findings)
	}
	fail.Store(true)
	if s := p.Run(context.Background(), fakeLiveSource{}).Component.Status; s != "HEALTHY" {
		t.Fatalf("cycle 3 should be suppressed again after streak reset; got %s", s)
	}
}

// TestEndpoints_DeterministicNotSuppressed: an HTTP status mismatch is a
// deterministic failure and must alert on the first cycle.
func TestEndpoints_DeterministicNotSuppressed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	p := NewEndpoints(
		[]EndpointTarget{{URL: srv.URL, Name: "test", ExpectStatus: 200}},
		DiscoveryOptions{Enabled: false},
	)
	p.Timeout = 500 * time.Millisecond
	p.RetryOnFlake = false

	r := p.Run(context.Background(), fakeLiveSource{})
	if r.Component.Status != "CRITICAL" {
		t.Errorf("HTTP status mismatch should be CRITICAL on first cycle; got %s", r.Component.Status)
	}
	if len(r.Findings) != 1 || r.Findings[0].Severity != SeverityCritical {
		t.Errorf("expected critical finding; got %+v", r.Findings)
	}
}

// TestEndpoints_RetryRecoversFlake: in-cycle retry must mask a single
// transient flake when RetryOnFlake is true.
func TestEndpoints_RetryRecoversFlake(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) == 1 {
			hj, _ := w.(http.Hijacker)
			conn, _, _ := hj.Hijack()
			_ = conn.Close()
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	p := NewEndpoints(
		[]EndpointTarget{{URL: srv.URL, Name: "test"}},
		DiscoveryOptions{Enabled: false},
	)
	p.Timeout = 800 * time.Millisecond
	p.RetryOnFlake = true

	r := p.Run(context.Background(), fakeLiveSource{})
	if r.Component.Status != "HEALTHY" || len(r.Findings) != 0 {
		t.Fatalf("retry should mask single flake; got status=%s findings=%+v",
			r.Component.Status, r.Findings)
	}
	if got := attempts.Load(); got != 2 {
		t.Errorf("expected exactly 2 attempts; got %d", got)
	}
}

// ---- M4: Layer-7 body assertion ---------------------------------------

func TestReadL7Annotations_NoAnnotation_Nil(t *testing.T) {
	if got := readL7Annotations(nil); got != nil {
		t.Errorf("nil map should yield nil; got %+v", got)
	}
	if got := readL7Annotations(map[string]string{"unrelated": "x"}); got != nil {
		t.Errorf("missing path annotation should yield nil; got %+v", got)
	}
}

func TestReadL7Annotations_PathOnly(t *testing.T) {
	got := readL7Annotations(map[string]string{
		"srenix.ai/probe-l7-path": "/healthz",
	})
	if got == nil || got.Path != "/healthz" || got.ExpectBody != "" || got.ExpectStatus != 0 {
		t.Errorf("path-only L7 spec misparsed: %+v", got)
	}
}

func TestReadL7Annotations_FullAnnotation(t *testing.T) {
	got := readL7Annotations(map[string]string{
		"srenix.ai/probe-l7-path":   "healthz", // missing leading /
		"srenix.ai/probe-l7-expect": `"status":"ok"`,
		"srenix.ai/probe-l7-status": "200",
	})
	if got == nil {
		t.Fatal("expected non-nil L7 spec")
	}
	if got.Path != "/healthz" {
		t.Errorf("path should be normalized with /; got %q", got.Path)
	}
	if got.ExpectBody != `"status":"ok"` {
		t.Errorf("body expect: %q", got.ExpectBody)
	}
	if got.ExpectStatus != 200 {
		t.Errorf("status expect: %d", got.ExpectStatus)
	}
}

func TestMatchBody_Substring(t *testing.T) {
	if !matchBody(`{"status":"ok"}`, `"status":"ok"`) {
		t.Errorf("substring match should succeed")
	}
	if matchBody(`{"status":"degraded"}`, `"status":"ok"`) {
		t.Errorf("substring match should fail when not present")
	}
}

func TestMatchBody_Regex(t *testing.T) {
	if !matchBody(`HTTP/1.1 200 OK`, `regex:^HTTP/[12]\.[0-9]+\s+\d{3}`) {
		t.Errorf("regex should match HTTP status line")
	}
	if matchBody(`hello world`, `regex:^goodbye`) {
		t.Errorf("regex should fail when not matching")
	}
}

func TestMatchBody_BadRegex_FailsClosed(t *testing.T) {
	// Invalid regex pattern → should NOT match (fail-closed) so
	// operator typos don't falsely pass the assertion.
	if matchBody(`anything`, `regex:[invalid`) {
		t.Errorf("invalid regex must fail-closed")
	}
}

func TestTruncate_Bounds(t *testing.T) {
	if got := truncate("short", 10); got != "short" {
		t.Errorf("short string should pass through; got %q", got)
	}
	if got := truncate("0123456789abcdef", 5); got != "01234…" {
		t.Errorf("truncate output: %q", got)
	}
}
