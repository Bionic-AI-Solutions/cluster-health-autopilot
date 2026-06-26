// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package prom

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestServer(t *testing.T, body string, status int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = fmt.Fprint(w, body)
	}))
}

func TestNew_NoOpWhenURLEmpty(t *testing.T) {
	trig := make(chan struct{}, 1)
	c, err := New(Config{}, trig)
	if err != nil {
		t.Fatal(err)
	}
	// Run should return immediately (no-op).
	done := make(chan struct{})
	go func() {
		c.Run(context.Background())
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run should return immediately when URL is empty")
	}
}

func TestNew_NegativeIntervalErrors(t *testing.T) {
	if _, err := New(Config{URL: "x", PollInterval: -time.Second}, make(chan struct{}, 1)); err == nil {
		t.Error("expected error for negative interval")
	}
}

func TestTick_FiringAlert_PushesTrigger(t *testing.T) {
	body := `[{"fingerprint":"fp1","status":{"state":"active"},"labels":{"alertname":"DiskFillUp"}}]`
	srv := newTestServer(t, body, 200)
	defer srv.Close()

	trig := make(chan struct{}, 1)
	c, err := New(Config{URL: srv.URL}, trig)
	if err != nil {
		t.Fatal(err)
	}
	c.tick(context.Background())
	select {
	case <-trig:
	case <-time.After(time.Second):
		t.Fatal("expected trigger signal")
	}
}

func TestTick_DedupSameFingerprint(t *testing.T) {
	body := `[{"fingerprint":"fp1","status":{"state":"active"},"labels":{"alertname":"X"}}]`
	srv := newTestServer(t, body, 200)
	defer srv.Close()

	trig := make(chan struct{}, 2)
	c, _ := New(Config{URL: srv.URL}, trig)

	c.tick(context.Background())
	c.tick(context.Background())

	// Only ONE signal should have been pushed; second tick is a no-op
	// because the fingerprint is in seen.
	select {
	case <-trig:
	default:
		t.Fatal("expected at least one trigger")
	}
	select {
	case <-trig:
		t.Fatal("dedup failed — second trigger pushed on same fingerprint")
	default:
	}
}

func TestTick_ResolvedAlertReFires(t *testing.T) {
	// First poll: alert firing.
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			_, _ = fmt.Fprint(w, `[{"fingerprint":"fp1","status":{"state":"active"},"labels":{"alertname":"X"}}]`)
			return
		}
		if calls == 2 {
			// Alert resolved (no active alerts).
			_, _ = fmt.Fprint(w, `[]`)
			return
		}
		// Alert fires again.
		_, _ = fmt.Fprint(w, `[{"fingerprint":"fp1","status":{"state":"active"},"labels":{"alertname":"X"}}]`)
	}))
	defer srv.Close()

	trig := make(chan struct{}, 5)
	c, _ := New(Config{URL: srv.URL}, trig)

	c.tick(context.Background()) // fires
	c.tick(context.Background()) // resolved → forget fingerprint
	c.tick(context.Background()) // fires again

	signals := 0
loop:
	for {
		select {
		case <-trig:
			signals++
		default:
			break loop
		}
	}
	if signals != 2 {
		t.Errorf("expected 2 trigger signals (initial + re-fire); got %d", signals)
	}
}

func TestTick_NonActiveAlertsIgnored(t *testing.T) {
	// status.state="suppressed" — these are not "firing", so they
	// shouldn't trigger.
	body := `[{"fingerprint":"fp1","status":{"state":"suppressed"},"labels":{"alertname":"X"}}]`
	srv := newTestServer(t, body, 200)
	defer srv.Close()

	trig := make(chan struct{}, 1)
	c, _ := New(Config{URL: srv.URL}, trig)
	c.tick(context.Background())
	select {
	case <-trig:
		t.Fatal("suppressed alert must not trigger")
	default:
	}
}

func TestTick_AlertNameFilter(t *testing.T) {
	body := `[
	  {"fingerprint":"a","status":{"state":"active"},"labels":{"alertname":"AllowMe"}},
	  {"fingerprint":"b","status":{"state":"active"},"labels":{"alertname":"BlockMe"}}
	]`
	srv := newTestServer(t, body, 200)
	defer srv.Close()

	trig := make(chan struct{}, 2)
	c, _ := New(Config{URL: srv.URL, AlertNameFilter: []string{"AllowMe"}}, trig)
	c.tick(context.Background())

	signals := 0
loop:
	for {
		select {
		case <-trig:
			signals++
		default:
			break loop
		}
	}
	if signals != 1 {
		t.Errorf("filter should pass only AllowMe; got %d signals", signals)
	}
}

func TestTick_5xxFromAlertmanager_NoCrash(t *testing.T) {
	srv := newTestServer(t, `whatever`, 503)
	defer srv.Close()
	trig := make(chan struct{}, 1)
	c, _ := New(Config{URL: srv.URL}, trig)
	c.tick(context.Background()) // must not panic
	select {
	case <-trig:
		t.Fatal("5xx must not trigger")
	default:
	}
}

func TestPollIntervalClamping(t *testing.T) {
	c, _ := New(Config{URL: "x", PollInterval: time.Second}, make(chan struct{}, 1))
	if c.cfg.PollInterval != 5*time.Second {
		t.Errorf("expected clamp to 5s; got %v", c.cfg.PollInterval)
	}
}
