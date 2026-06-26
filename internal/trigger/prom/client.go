// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

// Package prom implements the M5 Alertmanager polling trigger source.
//
// A long-running goroutine polls Alertmanager's /api/v2/alerts every
// PollInterval and, on transition from "no firing alerts" → "1+ firing
// alerts" (or on any new fingerprint appearing in the set), pushes a
// signal to the watcher's existing trigCh. The watcher's dedup +
// fingerprint logic absorbs the rest.
//
// Why class C: probes that detect "slow-drift" assets — disk fill,
// cert expiry creep, error-budget burn, GPU ECC accumulation — produce
// NO K8s event but DO produce a Prometheus metric → an Alertmanager
// alert. Class A (resource-change events) and class B (status-aware)
// miss these entirely.
//
// Failure modes are explicitly tolerated:
//   - Alertmanager unreachable → log + retry next interval
//   - 4xx/5xx → log + retry next interval
//   - Malformed JSON → log + retry next interval
//
// The trigger never blocks the watcher; a full trigCh is dropped silently
// (same semantics as the existing trigger sources).
package prom

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// Config controls the Alertmanager polling client. Empty URL → the
// client is a no-op (Run returns immediately) so operators can wire
// it unconditionally and toggle via values.
type Config struct {
	// URL is the Alertmanager base URL (e.g. http://alertmanager:9093).
	// Empty disables the client.
	URL string

	// PollInterval is the polling cadence. Default 30s. Anything below
	// 5s is clamped to 5s to keep Alertmanager unloaded.
	PollInterval time.Duration

	// Timeout is the per-request HTTP timeout. Default 10s.
	Timeout time.Duration

	// HTTP overrides the default client (tests inject a stub).
	HTTP *http.Client

	// AlertNameFilter, when non-empty, limits the alerts that trigger
	// a cycle to those whose alertname is in this slice (case-
	// insensitive). Empty = ANY firing alert triggers.
	AlertNameFilter []string
}

// Client polls Alertmanager + pushes signals to the watcher's trigCh.
type Client struct {
	cfg   Config
	seen  map[string]struct{} // fingerprints we've already triggered on
	trigC chan<- struct{}
}

// New returns a Client; errors on invalid config (negative interval).
// Empty Config.URL yields a no-op Client (Run returns immediately).
func New(cfg Config, trigCh chan<- struct{}) (*Client, error) {
	if cfg.PollInterval < 0 {
		return nil, errors.New("prom: negative PollInterval")
	}
	if cfg.PollInterval == 0 {
		cfg.PollInterval = 30 * time.Second
	}
	if cfg.PollInterval < 5*time.Second {
		cfg.PollInterval = 5 * time.Second
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Second
	}
	if cfg.HTTP == nil {
		cfg.HTTP = &http.Client{Timeout: cfg.Timeout}
	}
	return &Client{
		cfg:   cfg,
		seen:  map[string]struct{}{},
		trigC: trigCh,
	}, nil
}

// Run blocks until ctx is cancelled. Polls /api/v2/alerts every
// PollInterval and pushes one trigger signal per NEW firing-alert
// fingerprint. Soft-fail throughout — transport / parse errors are
// logged and ignored.
func (c *Client) Run(ctx context.Context) {
	if c == nil || c.cfg.URL == "" {
		return // no-op when not configured
	}
	t := time.NewTicker(c.cfg.PollInterval)
	defer t.Stop()
	log.Printf("trigger/prom: polling %s every %s", c.cfg.URL, c.cfg.PollInterval)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			c.tick(ctx)
		}
	}
}

// tick fetches the current alert set + pushes one trigCh signal per
// newly-seen alert. Exported for tests via the test-only invoker.
func (c *Client) tick(ctx context.Context) {
	alerts, err := c.fetchAlerts(ctx)
	if err != nil {
		log.Printf("trigger/prom: poll: %v", err)
		return
	}
	for _, a := range alerts {
		if !c.passesFilter(a.AlertName()) {
			continue
		}
		if _, dup := c.seen[a.Fingerprint]; dup {
			continue
		}
		c.seen[a.Fingerprint] = struct{}{}
		select {
		case c.trigC <- struct{}{}:
		default:
			// Channel full — watcher already has a pending trigger.
			// This is by design: dedup happens downstream.
		}
		log.Printf("trigger/prom: fired on alert %q (fingerprint=%s)", a.AlertName(), a.Fingerprint)
	}
	// Forget fingerprints that no longer appear — so a resolved-then-
	// re-firing alert triggers again on the next firing edge.
	stillFiring := make(map[string]struct{}, len(alerts))
	for _, a := range alerts {
		stillFiring[a.Fingerprint] = struct{}{}
	}
	for fp := range c.seen {
		if _, ok := stillFiring[fp]; !ok {
			delete(c.seen, fp)
		}
	}
}

func (c *Client) passesFilter(name string) bool {
	if len(c.cfg.AlertNameFilter) == 0 {
		return true
	}
	low := strings.ToLower(name)
	for _, f := range c.cfg.AlertNameFilter {
		if strings.ToLower(f) == low {
			return true
		}
	}
	return false
}

// alert is the trimmed shape we read from /api/v2/alerts.
type alert struct {
	Fingerprint string                 `json:"fingerprint"`
	Status      struct{ State string } `json:"status"`
	Labels      map[string]string      `json:"labels"`
}

// AlertName returns the canonical alertname label, falling back to
// the fingerprint when the label is absent.
func (a alert) AlertName() string {
	if n, ok := a.Labels["alertname"]; ok {
		return n
	}
	return a.Fingerprint
}

// fetchAlerts hits /api/v2/alerts and returns the firing alerts only
// (status.state == "active"). Suppressed/resolved alerts are filtered
// out at parse time.
func (c *Client) fetchAlerts(ctx context.Context) ([]alert, error) {
	u := strings.TrimRight(c.cfg.URL, "/") + "/api/v2/alerts?active=true&silenced=false&inhibited=false"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "srenix-prom-trigger/1.0")
	resp, err := c.cfg.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode/100 != 2 {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, fmt.Errorf("alertmanager HTTP %d", resp.StatusCode)
	}
	var raw []alert
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode alerts: %w", err)
	}
	out := raw[:0]
	for _, a := range raw {
		if strings.EqualFold(a.Status.State, "active") {
			out = append(out, a)
		}
	}
	return out, nil
}
