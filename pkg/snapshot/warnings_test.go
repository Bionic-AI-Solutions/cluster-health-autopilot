// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package snapshot

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"k8s.io/client-go/rest"
)

// recordingWarningHandler captures every warning forwarded to it.
type recordingWarningHandler struct {
	got []string
}

func (r *recordingWarningHandler) HandleWarningHeader(_ int, _ string, text string) {
	r.got = append(r.got, text)
}

// recordingWarningHandlerWithContext is the context-aware twin.
type recordingWarningHandlerWithContext struct {
	got []string
}

func (r *recordingWarningHandlerWithContext) HandleWarningHeaderWithContext(_ context.Context, _ int, _ string, text string) {
	r.got = append(r.got, text)
}

// The exact text the live API server sends today (plus the version drift
// variant a future server might send).
const (
	endpointsWarning       = "v1 Endpoints is deprecated in v1.33+; use discovery.k8s.io/v1 EndpointSlice"
	endpointsWarningFuture = "v1 Endpoints is deprecated in v1.39+; use discovery.k8s.io/v1 EndpointSlice"
)

func TestSuppressingWarningHandler_DropsEndpointsDeprecation(t *testing.T) {
	rec := &recordingWarningHandler{}
	h := newSuppressingWarningHandler(rec)

	h.HandleWarningHeader(299, "-", endpointsWarning)
	h.HandleWarningHeader(299, "-", endpointsWarningFuture)

	if len(rec.got) != 0 {
		t.Errorf("Endpoints deprecation warnings must be suppressed, but %d leaked: %v", len(rec.got), rec.got)
	}
}

func TestSuppressingWarningHandler_PassesOtherWarningsThrough(t *testing.T) {
	rec := &recordingWarningHandler{}
	h := newSuppressingWarningHandler(rec)

	other := "batch/v1beta1 CronJob is deprecated in v1.21+, unavailable in v1.25+; use batch/v1 CronJob"
	h.HandleWarningHeader(299, "-", endpointsWarning)
	h.HandleWarningHeader(299, "-", other)

	if len(rec.got) != 1 || rec.got[0] != other {
		t.Errorf("non-Endpoints warnings must pass through untouched; got %v", rec.got)
	}
}

// TestSuppressingWarningHandler_DefaultNextDeduplicates pins the production
// wiring contract: warnings that DO pass the filter print once per unique
// message, not once per call (rest.NewWarningWriter with Deduplicate). The
// writer is constructed here exactly as newSuppressingWarningHandler(nil)
// does, but pointed at a buffer so the output is observable.
func TestSuppressingWarningHandler_DefaultNextDeduplicates(t *testing.T) {
	var buf bytes.Buffer
	h := suppressingWarningHandler{
		next: rest.NewWarningWriter(&buf, rest.WarningWriterOptions{Deduplicate: true}),
	}

	other := "policy/v1beta1 PodSecurityPolicy is deprecated in v1.21+, unavailable in v1.25+"
	for i := 0; i < 5; i++ {
		h.HandleWarningHeader(299, "-", other)
		h.HandleWarningHeader(299, "-", endpointsWarning)
	}

	out := buf.String()
	if strings.Contains(out, "Endpoints is deprecated") {
		t.Errorf("Endpoints deprecation leaked to the writer: %q", out)
	}
	if n := strings.Count(out, "PodSecurityPolicy"); n != 1 {
		t.Errorf("expected the passthrough warning to print exactly once (dedup), got %d in %q", n, out)
	}
}

func TestSuppressEndpointsDeprecationWarnings_InstallsHandler(t *testing.T) {
	cfg := &rest.Config{}
	got := SuppressEndpointsDeprecationWarnings(cfg)
	if got != cfg {
		t.Fatalf("expected the same *rest.Config back for chaining")
	}
	if cfg.WarningHandler == nil {
		t.Fatal("WarningHandler not installed on the config")
	}
	if _, ok := cfg.WarningHandler.(suppressingWarningHandler); !ok {
		t.Errorf("installed handler is %T, want suppressingWarningHandler", cfg.WarningHandler)
	}
}

func TestSuppressEndpointsDeprecationWarnings_NilConfigNoop(t *testing.T) {
	if got := SuppressEndpointsDeprecationWarnings(nil); got != nil {
		t.Errorf("nil config must stay nil, got %v", got)
	}
}

// TestSuppressEndpointsDeprecationWarnings_WrapsCallerWarningHandler — the
// function is self-composable: a caller-installed WarningHandler becomes the
// filter's `next` (not replaced), so it still receives every non-suppressed
// warning while the Endpoints deprecation line is dropped.
func TestSuppressEndpointsDeprecationWarnings_WrapsCallerWarningHandler(t *testing.T) {
	rec := &recordingWarningHandler{}
	cfg := &rest.Config{WarningHandler: rec}
	SuppressEndpointsDeprecationWarnings(cfg)

	other := "batch/v1beta1 CronJob is deprecated in v1.21+, unavailable in v1.25+; use batch/v1 CronJob"
	cfg.WarningHandler.HandleWarningHeader(299, "-", endpointsWarning)
	cfg.WarningHandler.HandleWarningHeader(299, "-", other)

	if len(rec.got) != 1 || rec.got[0] != other {
		t.Errorf("caller's handler must receive exactly the non-suppressed warnings; got %v", rec.got)
	}
}

// TestSuppressEndpointsDeprecationWarnings_WrapsCallerWarningHandlerWithContext —
// same contract for the context-aware field, which client-go prefers when
// both are set: the caller's WarningHandlerWithContext is wrapped as `next`.
func TestSuppressEndpointsDeprecationWarnings_WrapsCallerWarningHandlerWithContext(t *testing.T) {
	rec := &recordingWarningHandlerWithContext{}
	cfg := &rest.Config{WarningHandlerWithContext: rec}
	SuppressEndpointsDeprecationWarnings(cfg)

	other := "batch/v1beta1 CronJob is deprecated in v1.21+, unavailable in v1.25+; use batch/v1 CronJob"
	ctx := context.Background()
	cfg.WarningHandlerWithContext.HandleWarningHeaderWithContext(ctx, 299, "-", endpointsWarning)
	cfg.WarningHandlerWithContext.HandleWarningHeaderWithContext(ctx, 299, "-", other)

	if len(rec.got) != 1 || rec.got[0] != other {
		t.Errorf("caller's context handler must receive exactly the non-suppressed warnings; got %v", rec.got)
	}
	if cfg.WarningHandler != nil {
		t.Errorf("legacy WarningHandler field must stay nil when wrapping WarningHandlerWithContext (client-go precedence)")
	}
}

// TestNewLiveSource_WrapsCallerHandlerOnCopy — wrapping is now the contract:
// NewLiveSource applies the filter to an internal COPY of the config, so the
// caller's config is never mutated, and the caller's handler (wrapped inside
// the copy) still receives non-suppressed warnings through the filter.
func TestNewLiveSource_WrapsCallerHandlerOnCopy(t *testing.T) {
	rec := &recordingWarningHandler{}
	cfg := &rest.Config{Host: "https://127.0.0.1:6443", WarningHandler: rec}
	if _, err := NewLiveSource(cfg); err != nil {
		t.Fatalf("NewLiveSource: %v", err)
	}
	if cfg.WarningHandler != rest.WarningHandler(rec) {
		t.Errorf("NewLiveSource must not mutate the caller's config (wrapping happens on an internal copy)")
	}

	// The wrapped handler the internal copy carries forwards non-suppressed
	// warnings to the caller's handler — pin that via the same composition
	// SuppressEndpointsDeprecationWarnings(rest.CopyConfig(cfg)) builds.
	wrapped := SuppressEndpointsDeprecationWarnings(rest.CopyConfig(cfg))
	other := "batch/v1beta1 CronJob is deprecated in v1.21+, unavailable in v1.25+; use batch/v1 CronJob"
	wrapped.WarningHandler.HandleWarningHeader(299, "-", endpointsWarning)
	wrapped.WarningHandler.HandleWarningHeader(299, "-", other)
	if len(rec.got) != 1 || rec.got[0] != other {
		t.Errorf("caller's handler must still receive non-suppressed warnings through the wrapper; got %v", rec.got)
	}

	plain := &rest.Config{Host: "https://127.0.0.1:6443"}
	if _, err := NewLiveSource(plain); err != nil {
		t.Fatalf("NewLiveSource: %v", err)
	}
	if plain.WarningHandler != nil {
		t.Errorf("NewLiveSource must not mutate the caller's config (filter goes on an internal copy)")
	}
}
