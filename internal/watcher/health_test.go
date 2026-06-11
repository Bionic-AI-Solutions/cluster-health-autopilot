// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package watcher

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// P1.9(a) — /healthz must serve UNCONDITIONALLY, independent of the M6
// webhook receiver. Before this fix the only /healthz lived inside the
// `if WebhookListen != ""` branch, so a watcher with no webhook trigger
// had no HTTP health endpoint and the chart could not wire liveness /
// readiness probes.
func TestHealthHandler_ServesOKWithoutWebhookListen(t *testing.T) {
	w := New(nil, nil, nil, Config{}) // WebhookListen empty

	srv := httptest.NewServer(w.healthHandler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /healthz status = %d, want 200", resp.StatusCode)
	}
}

// healthListenAddr resolves the effective health listen address: the
// explicit HealthListen, else the default :8081. (It intentionally does
// NOT fall back to WebhookListen — the health server is always-on and
// must not share the webhook port's lifecycle.)
func TestHealthListenAddr_DefaultsTo8081(t *testing.T) {
	if got := (&Watcher{cfg: Config{}}).healthListenAddr(); got != ":8081" {
		t.Fatalf("default healthListenAddr = %q, want :8081", got)
	}
	if got := (&Watcher{cfg: Config{HealthListen: ":9000"}}).healthListenAddr(); got != ":9000" {
		t.Fatalf("explicit healthListenAddr = %q, want :9000", got)
	}
}
