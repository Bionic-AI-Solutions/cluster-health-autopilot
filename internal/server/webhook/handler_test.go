// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newHandlerT(t *testing.T) (*Handler, <-chan struct{}) {
	t.Helper()
	ch := make(chan struct{}, 4)
	h := New(ch)
	return h, ch
}

func TestHealth_GET_200(t *testing.T) {
	h, _ := newHandlerT(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/webhook/health", nil)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("health: got %d want 200", rec.Code)
	}
}

func TestPOST_UnknownSource_404(t *testing.T) {
	h, _ := newHandlerT(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/webhook/unknown", strings.NewReader("{}"))
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("unknown source: got %d want 404", rec.Code)
	}
}

func TestPOST_RegisteredSource_WithBadSig_401(t *testing.T) {
	h, _ := newHandlerT(t)
	h.RegisterSource("vault", "supersecret")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/webhook/vault", strings.NewReader(`{"type":"x"}`))
	req.Header.Set("X-CHA-Signature", "sha256=deadbeef")
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("bad sig: got %d want 401", rec.Code)
	}
}

func TestPOST_RegisteredSource_WithValidSig_202_PushesTrigger(t *testing.T) {
	h, ch := newHandlerT(t)
	h.RegisterSource("vault", "supersecret")
	body := []byte(`{"type":"secret_rotation","path":"secret/x","version":42}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/webhook/vault", bytes.NewReader(body))
	req.Header.Set("X-CHA-Signature", Sign(body, "supersecret"))
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Errorf("good sig: got %d want 202", rec.Code)
	}
	select {
	case <-ch:
	default:
		t.Fatal("expected trigger signal")
	}
}

func TestPOST_NoSecret_NoSigRequired(t *testing.T) {
	// Debug-only path — empty secret disables HMAC.
	h, ch := newHandlerT(t)
	h.RegisterSource("debug", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/webhook/debug", strings.NewReader("{}"))
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Errorf("debug source: got %d want 202", rec.Code)
	}
	select {
	case <-ch:
	default:
		t.Fatal("expected trigger")
	}
}

func TestGET_OnNonHealthPath_405(t *testing.T) {
	h, _ := newHandlerT(t)
	h.RegisterSource("vault", "x")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/webhook/vault", nil)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET non-health: got %d want 405", rec.Code)
	}
}

func TestSign_Verify_RoundTrip(t *testing.T) {
	body := []byte("hello world")
	sig := Sign(body, "k1")
	if !verifyHMAC(body, "k1", sig) {
		t.Error("sign/verify round-trip failed")
	}
	// Tampered body fails.
	if verifyHMAC([]byte("hello WORLD"), "k1", sig) {
		t.Error("tampered body should fail verification")
	}
}

func TestPOST_BodyLargerThanLimit_Truncated(t *testing.T) {
	h, _ := newHandlerT(t)
	h.RegisterSource("vault", "")
	// 128 KiB body — handler reads up to 64 KiB.
	body := bytes.Repeat([]byte("x"), 128*1024)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/webhook/vault", bytes.NewReader(body))
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Errorf("oversized body should still succeed (truncated read); got %d", rec.Code)
	}
}

func TestRoot_404(t *testing.T) {
	h, _ := newHandlerT(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/webhook/", nil)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("root /webhook/ should 404; got %d", rec.Code)
	}
}
