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
// Signature schemes (X-CHA-Signature header, value "sha256=<hex>"):
//
//   - Timestamped (recommended): the request also carries
//     X-CHA-Timestamp=<unix-seconds> and the signed payload is
//     "<timestamp>.<body>", i.e. sha256=hex(hmac-sha256(secret,
//     timestamp + "." + body)). Requests whose timestamp is more than
//     5 minutes from server time (either direction) are rejected with
//     401, bounding the replay window of a captured request.
//   - Legacy body-only (backward compat): no X-CHA-Timestamp header;
//     the signed payload is just the body. Captured requests replay
//     forever — a once-per-source notice is logged recommending the
//     timestamp header.
//
// Fail-closed: a source registered with an empty secret (a
// misconfiguration — the watcher refuses to register such sources)
// rejects ALL requests with 401. Genuinely-unauthenticated sources must
// be registered explicitly via RegisterInsecureSource.
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
	"strconv"
	"strings"
	"sync"
	"time"
)

// maxTimestampSkew bounds the replay window for timestamped signatures:
// |now - X-CHA-Timestamp| must be within this duration.
const maxTimestampSkew = 5 * time.Minute

// sourceCfg is the per-source auth configuration.
type sourceCfg struct {
	secret string
	// insecure marks a source explicitly registered without
	// authentication (RegisterInsecureSource). This is distinct from an
	// empty secret, which is treated as a misconfiguration and rejects
	// every request (fail-closed).
	insecure bool
}

// Handler is the HTTP handler. Registered sources + their HMAC secrets
// are kept in an internal map; concurrent reads use a RWMutex so a
// future "add source at runtime" CR can mutate safely. Today secrets
// come once at Start time from the chart's ESO-mounted Secret.
type Handler struct {
	mu      sync.RWMutex
	sources map[string]sourceCfg
	trigC   chan<- struct{}

	// legacyNoticed tracks sources that already logged the
	// once-per-source "use X-CHA-Timestamp" recommendation;
	// emptyNoticed does the same for the empty-secret rejection
	// (the endpoint is reachable unauthenticated, so per-request
	// logging would let an attacker spam the logs).
	legacyMu      sync.Mutex
	legacyNoticed map[string]bool
	emptyNoticed  map[string]bool
}

// New returns a Handler with no sources registered. Use RegisterSource
// for each one. trigCh receives a signal per authenticated POST.
func New(trigCh chan<- struct{}) *Handler {
	return &Handler{
		sources:       map[string]sourceCfg{},
		trigC:         trigCh,
		legacyNoticed: map[string]bool{},
		emptyNoticed:  map[string]bool{},
	}
}

// RegisterSource adds a webhook source with HMAC verification. An empty
// secret does NOT disable verification: the source is registered
// fail-closed and rejects every request with 401 (registration-time
// validation in the watcher should prevent this state from ever being
// reached). Use RegisterInsecureSource for an explicitly
// unauthenticated source.
func (h *Handler) RegisterSource(name, hmacSecret string) {
	if name == "" {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.sources[name] = sourceCfg{secret: hmacSecret}
}

// RegisterInsecureSource adds a webhook source with HMAC verification
// DISABLED. Anyone who can reach the listener can trigger diagnose
// cycles for this source — only use behind trusted network boundaries.
func (h *Handler) RegisterInsecureSource(name string) {
	if name == "" {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.sources[name] = sourceCfg{insecure: true}
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
	cfg, ok := h.sources[src]
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

	if !h.authenticate(w, r, src, cfg, body) {
		return // authenticate already wrote the 401
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

// authenticate enforces the per-source auth policy. Returns true when
// the request may proceed; otherwise it writes a 401 and returns false.
func (h *Handler) authenticate(w http.ResponseWriter, r *http.Request, src string, cfg sourceCfg, body []byte) bool {
	if cfg.insecure {
		// Explicitly registered unauthenticated — no verification.
		return true
	}
	if cfg.secret == "" {
		// Defense in depth: registration fail-closes before reaching
		// this state, but if an empty secret slips through we reject
		// everything rather than silently skip HMAC.
		h.noticeEmptySecret(src)
		http.Error(w, "source has no HMAC secret configured", http.StatusUnauthorized)
		return false
	}

	sig := r.Header.Get("X-CHA-Signature")
	tsHdr := r.Header.Get("X-CHA-Timestamp")
	if tsHdr == "" {
		// Legacy body-only scheme — kept for backward compatibility.
		if !verifyHMAC(body, cfg.secret, sig) {
			http.Error(w, "invalid signature", http.StatusUnauthorized)
			return false
		}
		h.noticeLegacy(src)
		return true
	}

	// Timestamped scheme: signed payload is "<timestamp>.<body>".
	ts, err := strconv.ParseInt(tsHdr, 10, 64)
	if err != nil {
		http.Error(w, "invalid X-CHA-Timestamp (want unix seconds)", http.StatusUnauthorized)
		return false
	}
	if skew := time.Since(time.Unix(ts, 0)); skew > maxTimestampSkew || skew < -maxTimestampSkew {
		http.Error(w, "X-CHA-Timestamp outside replay window", http.StatusUnauthorized)
		return false
	}
	payload := make([]byte, 0, len(tsHdr)+1+len(body))
	payload = append(payload, tsHdr...)
	payload = append(payload, '.')
	payload = append(payload, body...)
	if !verifyHMAC(payload, cfg.secret, sig) {
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return false
	}
	return true
}

// noticeLegacy logs a once-per-source recommendation to adopt the
// timestamped signature scheme.
func (h *Handler) noticeLegacy(src string) {
	h.legacyMu.Lock()
	defer h.legacyMu.Unlock()
	if h.legacyNoticed[src] {
		return
	}
	h.legacyNoticed[src] = true
	log.Printf("webhook: source %q authenticated with legacy body-only HMAC — recommend sending X-CHA-Timestamp and signing 'timestamp.body' so captured requests cannot be replayed beyond %s", src, maxTimestampSkew)
}

// noticeEmptySecret logs the empty-secret rejection once per source —
// the endpoint is reachable unauthenticated, so per-request logging
// would be attacker-controlled log spam.
func (h *Handler) noticeEmptySecret(src string) {
	h.legacyMu.Lock()
	defer h.legacyMu.Unlock()
	if h.emptyNoticed[src] {
		return
	}
	h.emptyNoticed[src] = true
	log.Printf("webhook: source %q registered with empty HMAC secret — rejecting all requests (fail-closed)", src)
}

// verifyHMAC validates the X-CHA-Signature header against payload.
// Expected format: "sha256=<hex>". Constant-time comparison via
// hmac.Equal. Empty header → false.
func verifyHMAC(payload []byte, secret, header string) bool {
	if header == "" {
		return false
	}
	header = strings.TrimPrefix(header, "sha256=")
	want, err := hex.DecodeString(header)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	got := mac.Sum(nil)
	return hmac.Equal(want, got)
}

// Sign returns the legacy body-only signature header value for body
// under secret. Used by tests + by external integrators for
// verification. Prefer SignWithTimestamp for new senders.
func Sign(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// SignWithTimestamp returns the timestamped signature header value:
// sha256=hex(hmac-sha256(secret, "<ts>.<body>")), where ts is unix
// seconds and must also be sent as the X-CHA-Timestamp header.
func SignWithTimestamp(body []byte, secret string, ts int64) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strconv.FormatInt(ts, 10)))
	mac.Write([]byte{'.'})
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}
