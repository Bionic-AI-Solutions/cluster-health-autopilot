// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// ── helpers ───────────────────────────────────────────────────────────────────

// newTestClient creates a Client pointed at the given test server URL.
func newTestClient(t *testing.T, serverURL string) Client {
	t.Helper()
	return New("test-token", serverURL)
}

// cfSuccessResponse serialises a minimal Cloudflare success envelope with result r.
func cfSuccessResponse(t *testing.T, result any) []byte {
	t.Helper()
	type envelope struct {
		Result     any        `json:"result"`
		ResultInfo resultInfo `json:"result_info"`
		Success    bool       `json:"success"`
		Errors     []cfError  `json:"errors"`
	}
	b, err := json.Marshal(envelope{
		Result:     result,
		ResultInfo: resultInfo{Page: 1, PerPage: 100, TotalPages: 1},
		Success:    true,
	})
	if err != nil {
		t.Fatalf("marshal test envelope: %v", err)
	}
	return b
}

// cfErrorResponse serialises a Cloudflare error envelope.
func cfErrorResponse(t *testing.T, code int, msg string) []byte {
	t.Helper()
	type envelope struct {
		Result  []any     `json:"result"`
		Success bool      `json:"success"`
		Errors  []cfError `json:"errors"`
	}
	b, err := json.Marshal(envelope{
		Result:  nil,
		Success: false,
		Errors:  []cfError{{Code: code, Message: msg}},
	})
	if err != nil {
		t.Fatalf("marshal error envelope: %v", err)
	}
	return b
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestListZones_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/zones") {
			http.Error(w, "unexpected path", http.StatusNotFound)
			return
		}
		payload := []cfZone{{ID: "z1", Name: "example.com"}}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(cfSuccessResponse(t, payload))
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	zones, err := c.ListZones(context.Background())
	if err != nil {
		t.Fatalf("ListZones returned error: %v", err)
	}
	if len(zones) != 1 {
		t.Fatalf("expected 1 zone, got %d", len(zones))
	}
	if zones[0].ID != "z1" || zones[0].Name != "example.com" {
		t.Errorf("unexpected zone: %+v", zones[0])
	}
}

func TestListDNSRecords_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/dns_records") {
			http.Error(w, "unexpected path", http.StatusNotFound)
			return
		}
		payload := []cfDNSRecord{
			{Name: "foo.example.com", Type: "A", Content: "1.2.3.4", Proxied: false},
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(cfSuccessResponse(t, payload))
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	records, err := c.ListDNSRecords(context.Background(), "z1")
	if err != nil {
		t.Fatalf("ListDNSRecords returned error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	r := records[0]
	if r.Name != "foo.example.com" || r.Type != "A" || r.Content != "1.2.3.4" || r.Proxied != false {
		t.Errorf("unexpected record: %+v", r)
	}
}

func TestListZones_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(cfErrorResponse(t, 10000, "Authentication error"))
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.ListZones(context.Background())
	if err == nil {
		t.Fatal("expected error for 401 response, got nil")
	}
	// The response body has success=false and errors=[...], so expect a CF API error.
	if !strings.Contains(err.Error(), "cloudflare") && !strings.Contains(err.Error(), "10000") {
		t.Errorf("unexpected error text: %v", err)
	}
}

func TestClient_Timeout(t *testing.T) {
	// Server blocks until the test is done; client has a 100ms timeout.
	// We use a channel to unblock the handler when the test exits so that
	// srv.Close() does not wait 10 seconds.
	unblock := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-unblock:
		case <-time.After(10 * time.Second):
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer func() {
		close(unblock)
		srv.Close()
	}()

	// Build a client with a very short timeout via a custom *http.Client injected
	// into the struct directly (white-box: package-internal test).
	c := &client{
		token:   "tok",
		baseURL: srv.URL,
		http:    &http.Client{Timeout: 100 * time.Millisecond},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_, err := c.ListZones(ctx)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestListDNSRecords_Pagination(t *testing.T) {
	// Two pages: page 1 has 1 record, page 2 has 1 record.
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		page := r.URL.Query().Get("page")
		type envelope struct {
			Result     []cfDNSRecord `json:"result"`
			ResultInfo resultInfo    `json:"result_info"`
			Success    bool          `json:"success"`
			Errors     []cfError     `json:"errors"`
		}
		var env envelope
		env.Success = true
		if page == "1" {
			env.Result = []cfDNSRecord{{Name: "a.example.com", Type: "A", Content: "1.1.1.1"}}
			env.ResultInfo = resultInfo{Page: 1, PerPage: 1, TotalPages: 2, Count: 1, Total: 2}
		} else {
			env.Result = []cfDNSRecord{{Name: "b.example.com", Type: "A", Content: "2.2.2.2"}}
			env.ResultInfo = resultInfo{Page: 2, PerPage: 1, TotalPages: 2, Count: 1, Total: 2}
		}
		b, _ := json.Marshal(env)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	records, err := c.ListDNSRecords(context.Background(), "zone1")
	if err != nil {
		t.Fatalf("ListDNSRecords: %v", err)
	}
	if len(records) != 2 {
		t.Errorf("expected 2 records from pagination, got %d", len(records))
	}
	if calls < 2 {
		t.Errorf("expected at least 2 HTTP calls for pagination, got %d", calls)
	}
}
