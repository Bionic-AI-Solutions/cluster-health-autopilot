// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
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
	req.Header.Set("X-Srenix-Signature", "sha256=deadbeef")
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
	req.Header.Set("X-Srenix-Signature", Sign(body, "supersecret"))
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

// --- P1.1 fail-closed + timestamped-HMAC tests ---

// signTimestamped computes the v2 signature inline (timestamp + "." + body)
// so this test does not depend on a helper that may not exist yet.
func signTimestamped(ts string, body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts + "."))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestPOST_EmptySecretSource_AlwaysRejected_401(t *testing.T) {
	// Defense in depth: a source registered with an empty secret (should
	// never happen post fail-closed registration) must reject ALL
	// requests with 401, never skip verification.
	h, ch := newHandlerT(t)
	h.RegisterSource("misconfigured", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/webhook/misconfigured", strings.NewReader("{}"))
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("empty-secret source: got %d want 401 (fail-closed)", rec.Code)
	}
	select {
	case <-ch:
		t.Fatal("empty-secret source must not push a trigger")
	default:
	}
}

func TestPOST_TimestampedHMAC_Valid_202(t *testing.T) {
	h, ch := newHandlerT(t)
	h.RegisterSource("vault", "fake-test-secret")
	body := []byte(`{"type":"secret_rotation"}`)
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/webhook/vault", bytes.NewReader(body))
	req.Header.Set("X-Srenix-Timestamp", ts)
	req.Header.Set("X-Srenix-Signature", signTimestamped(ts, body, "fake-test-secret"))
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Errorf("valid timestamped HMAC: got %d want 202 (body=%q)", rec.Code, rec.Body.String())
	}
	select {
	case <-ch:
	default:
		t.Fatal("expected trigger signal")
	}
}

func TestPOST_TimestampedHMAC_Stale_401(t *testing.T) {
	h, ch := newHandlerT(t)
	h.RegisterSource("vault", "fake-test-secret")
	body := []byte(`{"type":"secret_rotation"}`)
	// 10 minutes old — outside the 5-minute replay window. Signature
	// itself is valid for that timestamp, so only the window check can
	// reject it.
	ts := strconv.FormatInt(time.Now().Add(-10*time.Minute).Unix(), 10)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/webhook/vault", bytes.NewReader(body))
	req.Header.Set("X-Srenix-Timestamp", ts)
	req.Header.Set("X-Srenix-Signature", signTimestamped(ts, body, "fake-test-secret"))
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("stale timestamp: got %d want 401", rec.Code)
	}
	select {
	case <-ch:
		t.Fatal("stale timestamp must not push a trigger")
	default:
	}
}

func TestPOST_TimestampedHMAC_FutureBeyondWindow_401(t *testing.T) {
	h, _ := newHandlerT(t)
	h.RegisterSource("vault", "fake-test-secret")
	body := []byte("{}")
	ts := strconv.FormatInt(time.Now().Add(10*time.Minute).Unix(), 10)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/webhook/vault", bytes.NewReader(body))
	req.Header.Set("X-Srenix-Timestamp", ts)
	req.Header.Set("X-Srenix-Signature", signTimestamped(ts, body, "fake-test-secret"))
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("future timestamp: got %d want 401", rec.Code)
	}
}

func TestPOST_TimestampedHMAC_MalformedTimestamp_401(t *testing.T) {
	h, _ := newHandlerT(t)
	h.RegisterSource("vault", "fake-test-secret")
	body := []byte("{}")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/webhook/vault", bytes.NewReader(body))
	req.Header.Set("X-Srenix-Timestamp", "not-a-number")
	req.Header.Set("X-Srenix-Signature", Sign(body, "fake-test-secret"))
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("malformed timestamp: got %d want 401", rec.Code)
	}
}

func TestPOST_LegacyBodyOnlyHMAC_Still202(t *testing.T) {
	// Backward compat: no X-Srenix-Timestamp header → body-only HMAC must
	// keep working so existing senders don't break.
	h, ch := newHandlerT(t)
	h.RegisterSource("vault", "fake-test-secret")
	body := []byte(`{"type":"legacy"}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/webhook/vault", bytes.NewReader(body))
	req.Header.Set("X-Srenix-Signature", Sign(body, "fake-test-secret"))
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Errorf("legacy body-only HMAC: got %d want 202", rec.Code)
	}
	select {
	case <-ch:
	default:
		t.Fatal("expected trigger signal")
	}
}

func TestPOST_InsecureSource_NoSigRequired_202(t *testing.T) {
	// Explicit opt-out path: RegisterInsecureSource disables HMAC for
	// the source — unlike an empty secret, which must always 401.
	h, ch := newHandlerT(t)
	h.RegisterInsecureSource("ci")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/webhook/ci", strings.NewReader("{}"))
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Errorf("insecure source: got %d want 202", rec.Code)
	}
	select {
	case <-ch:
	default:
		t.Fatal("expected trigger")
	}
}

func TestSignWithTimestamp_MatchesInlineComputation(t *testing.T) {
	body := []byte("payload")
	const ts = int64(1765432100)
	got := SignWithTimestamp(body, "fake-test-secret", ts)
	want := signTimestamped("1765432100", body, "fake-test-secret")
	if got != want {
		t.Errorf("SignWithTimestamp mismatch: got %s want %s", got, want)
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
	h.RegisterInsecureSource("vault")
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
