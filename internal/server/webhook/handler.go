// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

// Package webhook implements the M6 class-E trigger source: external
// systems POST to `/webhook/<source>` to ask CHA to re-run its diagnose
// cycle now, rather than wait for the next resync.
//
// Each registered source has its own HMAC-SHA256 secret; the handler
// rejects unsigned or mis-signed requests with 401. Unrecognized sources
// return 404. Recognized + authenticated requests respond 202 + push a
// signal to the watcher's trigCh.
//
// Closes the "rotation → probe within seconds" loop that the
// Vault-ESO analyzers currently depend on a daily cron + ESO
// refreshInterval to detect.
package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
)

// Handler is the HTTP handler. Registered sources + their HMAC secrets
// are kept in an internal map; concurrent reads use a RWMutex so a
// future "add source at runtime" CR can mutate safely. Today secrets
// come once at Start time from the chart's ESO-mounted Secret.
type Handler struct {
	mu      sync.RWMutex
	sources map[string]string // source-name → HMAC secret
	trigC   chan<- struct{}
}

// New returns a Handler with no sources registered. Use RegisterSource
// for each one. trigCh receives a signal per authenticated POST.
func New(trigCh chan<- struct{}) *Handler {
	return &Handler{
		sources: map[string]string{},
		trigC:   trigCh,
	}
}

// RegisterSource adds a webhook source. Empty secret disables HMAC
// verification for that source (debug-only; production deployments
// must always provide a secret).
func (h *Handler) RegisterSource(name, hmacSecret string) {
	if name == "" {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.sources[name] = hmacSecret
}

// ServeHTTP routes /webhook/<source>.
//
//   - GET /webhook/health  → 200 OK (liveness probe target)
//   - POST /webhook/<src>  → 202 Accepted if HMAC ok, else 401
//   - any other            → 404
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/webhook/health" {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	const prefix = "/webhook/"
	if !strings.HasPrefix(r.URL.Path, prefix) {
		http.NotFound(w, r)
		return
	}
	src := strings.TrimPrefix(r.URL.Path, prefix)
	src = strings.TrimSuffix(src, "/")
	if src == "" {
		http.NotFound(w, r)
		return
	}

	h.mu.RLock()
	secret, ok := h.sources[src]
	h.mu.RUnlock()
	if !ok {
		http.NotFound(w, r)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 64*1024))
	if err != nil {
		http.Error(w, "read body", http.StatusBadRequest)
		return
	}

	if secret != "" {
		sig := r.Header.Get("X-CHA-Signature")
		if !verifyHMAC(body, secret, sig) {
			http.Error(w, "invalid signature", http.StatusUnauthorized)
			return
		}
	}

	// Push trigger; non-blocking — if the channel is full the watcher
	// already has a pending cycle and dedup downstream is fine.
	select {
	case h.trigC <- struct{}{}:
	default:
	}
	log.Printf("webhook: trigger from source=%q body_bytes=%d", src, len(body))

	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write([]byte("accepted"))
}

// verifyHMAC validates the X-CHA-Signature header. Expected format:
// "sha256=<hex>". Constant-time comparison via hmac.Equal. Empty
// header → false.
func verifyHMAC(body []byte, secret, header string) bool {
	if header == "" {
		return false
	}
	header = strings.TrimPrefix(header, "sha256=")
	want, err := hex.DecodeString(header)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	got := mac.Sum(nil)
	return hmac.Equal(want, got)
}

// Sign returns the canonical signature header value for body under
// secret. Used by tests + by external integrators for verification.
func Sign(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}
