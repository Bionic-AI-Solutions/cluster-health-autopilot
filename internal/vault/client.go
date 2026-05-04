// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

// Package vault provides a minimal Vault KV-v2 read client used by the
// VaultPathMissing analyzer.
//
// Privacy contract: the public surface of this package only ever returns
// the SET OF KEY NAMES at a Vault path (via ListKeys). Byte values are
// never logged, never returned, never persisted. The analyzer's diagnostic
// output is itself derived from key names alone — see internal/diagnose/
// vault_path_missing.go.
//
// Why a hand-rolled HTTP client instead of the official hashicorp/vault SDK?
// The SDK pulls in retryablehttp, hashicorp/go-cleanhttp, and the hashicorp
// logger surface, which roughly doubles the cha binary size. The KV v2 read
// endpoint we need is two URL templates and a token header — simple enough
// that the dependency tax is not worth paying for v0.2. If a future probe
// needs broader Vault surface (write, list children, lease management),
// migrate to the SDK then; the Client interface in this file is the seam.
package vault

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

// Client reads key names from a Vault KV-v2 path.
//
// Implementations must:
//   - Return ErrPathNotFound when the path does not exist (404).
//   - Return only metadata-level key names — never the byte values stored
//     under them. This is enforced by the interface signature returning
//     []string, not map[string][]byte.
type Client interface {
	// ListKeys returns the KEY NAMES present at the given KV-v2 path.
	// Path is mount-relative (e.g. "team/app", not "secret/data/team/app").
	ListKeys(ctx context.Context, path string) ([]string, error)
}

// ErrPathNotFound is returned by Client.ListKeys when Vault returns 404
// for the path. Distinguished from transport / auth errors so the analyzer
// can emit a precise "path missing in Vault" diagnostic.
var ErrPathNotFound = fmt.Errorf("vault path not found")

// HTTPClient is the live KV-v2 implementation. Construct via New.
type HTTPClient struct {
	addr      string // e.g. "https://vault.example.com" — no trailing slash
	mount     string // KV-v2 mount path, e.g. "secret"
	token     string // Vault token; obtained via Authenticator
	transport *http.Client
}

// Config configures the HTTPClient.
type Config struct {
	// Address is the Vault HTTP(S) endpoint. Required.
	Address string
	// Mount is the KV-v2 mount path. Defaults to "secret".
	Mount string
	// Token is the Vault token. Mutually exclusive with KubernetesAuth.
	Token string
	// KubernetesAuth, if set, performs a `vault login -method=kubernetes`
	// against the given role using the in-cluster ServiceAccount JWT.
	// Mutually exclusive with Token.
	KubernetesAuth *KubernetesAuthConfig
	// Timeout for individual HTTP calls. Defaults to 5 seconds.
	Timeout time.Duration
	// HTTPClient is an optional override (tests inject httptest.Server's client).
	HTTPClient *http.Client
}

// New constructs an HTTPClient and authenticates to Vault.
//
// The returned client holds a live token. If kubernetes-auth is selected,
// the token is acquired immediately; if it later expires, ListKeys will
// surface the 403 error and the analyzer will report it like any other
// transport-level Vault failure (a fresh token next tick is cheap).
// v0.3 may add token-renew on 403 if operators report churn.
func New(cfg Config) (*HTTPClient, error) {
	if cfg.Address == "" {
		return nil, fmt.Errorf("vault: Address is required")
	}
	if (cfg.Token == "") == (cfg.KubernetesAuth == nil) {
		return nil, fmt.Errorf("vault: specify exactly one of Token or KubernetesAuth")
	}
	addr := strings.TrimRight(cfg.Address, "/")
	mount := cfg.Mount
	if mount == "" {
		mount = "secret"
	}
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: timeout}
	}

	c := &HTTPClient{
		addr:      addr,
		mount:     mount,
		token:     cfg.Token,
		transport: httpClient,
	}

	if cfg.KubernetesAuth != nil {
		token, err := loginKubernetes(httpClient, addr, *cfg.KubernetesAuth)
		if err != nil {
			return nil, fmt.Errorf("vault: kubernetes auth: %w", err)
		}
		c.token = token
	}
	return c, nil
}

// ListKeys implements Client.
func (c *HTTPClient) ListKeys(ctx context.Context, secretPath string) ([]string, error) {
	// KV-v2 read URL: {addr}/v1/{mount}/data/{path}
	u, err := url.Parse(c.addr)
	if err != nil {
		return nil, fmt.Errorf("vault: invalid address: %w", err)
	}
	u.Path = path.Join(u.Path, "v1", c.mount, "data", secretPath)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Vault-Token", c.token)

	resp, err := c.transport.Do(req)
	if err != nil {
		return nil, fmt.Errorf("vault: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		// KV-v2 returns 404 both for "path never existed" and "path
		// soft-deleted". Either way, the analyzer's "missing path"
		// diagnostic is correct.
		return nil, ErrPathNotFound
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("vault: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload struct {
		Data struct {
			Data map[string]any `json:"data"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("vault: decode response: %w", err)
	}
	out := make([]string, 0, len(payload.Data.Data))
	for k := range payload.Data.Data {
		out = append(out, k)
	}
	return out, nil
}
