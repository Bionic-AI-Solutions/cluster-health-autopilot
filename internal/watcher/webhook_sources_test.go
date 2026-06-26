// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package watcher

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/srenix-ai/agentic-sre/internal/server/webhook"
)

// captureLog redirects the stdlib logger into a buffer for the duration
// of the test and returns a getter for the captured output.
func captureLog(t *testing.T) func() string {
	t.Helper()
	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	return buf.String
}

// post fires a POST at the handler and returns the status code.
func post(h *webhook.Handler, path, sig string) int {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader("{}"))
	if sig != "" {
		req.Header.Set("X-Srenix-Signature", sig)
	}
	h.ServeHTTP(rec, req)
	return rec.Code
}

func TestRegisterWebhookSources_EnvUnset_FailClosed(t *testing.T) {
	logs := captureLog(t)
	ch := make(chan struct{}, 4)
	h := webhook.New(ch)

	registerWebhookSources(h, []string{"vault=SRENIX_TEST_UNSET_ENV_VAR"}, func(string) string { return "" })

	// Source must NOT be registered → 404, not 202/401.
	if code := post(h, "/webhook/vault", ""); code != http.StatusNotFound {
		t.Errorf("env-unset source must not be registered: got %d want 404", code)
	}
	out := logs()
	if !strings.Contains(out, "fail-closed") {
		t.Errorf("expected fail-closed error log, got: %q", out)
	}
	if !strings.Contains(out, "vault") || !strings.Contains(out, "SRENIX_TEST_UNSET_ENV_VAR") {
		t.Errorf("error log must name the source and env var, got: %q", out)
	}
}

func TestRegisterWebhookSources_MalformedSpec_FailClosed(t *testing.T) {
	logs := captureLog(t)
	ch := make(chan struct{}, 4)
	h := webhook.New(ch)

	registerWebhookSources(h, []string{"vault"}, func(string) string { return "should-not-be-read" })

	if code := post(h, "/webhook/vault", ""); code != http.StatusNotFound {
		t.Errorf("malformed spec source must not be registered: got %d want 404", code)
	}
	if out := logs(); !strings.Contains(out, "fail-closed") {
		t.Errorf("expected fail-closed error log for malformed spec, got: %q", out)
	}
}

func TestRegisterWebhookSources_InsecureNoHMAC_RegisteredUnauthenticated(t *testing.T) {
	logs := captureLog(t)
	ch := make(chan struct{}, 4)
	h := webhook.New(ch)

	registerWebhookSources(h, []string{"ci=insecure-no-hmac"}, func(string) string { return "" })

	// Registered with verification disabled → unsigned POST is accepted.
	if code := post(h, "/webhook/ci", ""); code != http.StatusAccepted {
		t.Errorf("insecure-no-hmac source: got %d want 202", code)
	}
	out := logs()
	if !strings.Contains(out, "UNAUTHENTICATED") || !strings.Contains(out, "ci") {
		t.Errorf("expected loud UNAUTHENTICATED warning naming the source, got: %q", out)
	}
}

func TestRegisterWebhookSources_EnvSet_RegisteredWithHMAC(t *testing.T) {
	_ = captureLog(t) // keep test output clean
	ch := make(chan struct{}, 4)
	h := webhook.New(ch)

	registerWebhookSources(h, []string{"vault=SRENIX_TEST_SET_ENV_VAR"}, func(k string) string {
		if k == "SRENIX_TEST_SET_ENV_VAR" {
			return "fake-test-secret"
		}
		return ""
	})

	// Unsigned → 401 (HMAC enforced), correctly signed → 202.
	if code := post(h, "/webhook/vault", ""); code != http.StatusUnauthorized {
		t.Errorf("unsigned POST to HMAC source: got %d want 401", code)
	}
	if code := post(h, "/webhook/vault", webhook.Sign([]byte("{}"), "fake-test-secret")); code != http.StatusAccepted {
		t.Errorf("signed POST to HMAC source: got %d want 202", code)
	}
}
