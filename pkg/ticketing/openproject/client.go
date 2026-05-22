// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

// Package openproject implements the ticketing.Sink interface against
// an OpenProject MCP server (the docker4zerocool/mcp-servers-openproject
// image deployed in the gpu-deploy cluster, in-cluster service at
// mcp-openproject-server.mcp.svc:8006).
//
// The Sink talks to the MCP server through a minimal MCPClient
// interface so the wire-level transport (Streamable HTTP / SSE+JSON-RPC)
// is pluggable and the Sink itself is unit-testable against a mock.
package openproject

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"time"
)

// MCPClient is the minimum MCP surface this Sink needs. Implementations:
//   - HTTPClient: streamable-HTTP transport for production use.
//   - nopClient: drops all calls; used by dry-run mode.
//   - mock (in _test.go): records calls for assertions.
type MCPClient interface {
	// CallTool invokes a single MCP tools/call request and returns the
	// decoded JSON result. args is marshalled as-is into the JSON-RPC
	// "arguments" object. The returned map mirrors the MCP tool's result
	// payload — callers extract provider-specific fields.
	CallTool(ctx context.Context, name string, args map[string]any) (map[string]any, error)
}

// HTTPClient is a MCPClient that speaks the MCP Streamable HTTP transport
// (single POST endpoint accepting JSON-RPC 2.0 requests, no SSE session
// required). The OpenProject MCP image deployed in-cluster exposes this
// transport at /openproject/mcp.
//
// This is intentionally minimal: enough to call tools/call and parse a
// JSON response. It does NOT implement the full MCP handshake / SSE /
// resource subscription surface — CHA only needs tool invocation.
//
// A future commit may swap this for github.com/mark3labs/mcp-go if we
// need richer protocol support (resources, prompts, completions).
type HTTPClient struct {
	// Endpoint is the full URL to the MCP streamable-HTTP endpoint,
	// e.g. http://mcp-openproject-server.mcp.svc:8006/mcp or
	// https://mcp.baisoln.com/openproject/mcp.
	Endpoint string

	// APIKey is the Kong key-auth value sent as the apikey header.
	// Empty when talking in-cluster (Kong is bypassed).
	APIKey string

	// HTTP is injected so tests and callers can supply custom timeouts,
	// trace transports, etc. Defaults to http.DefaultClient with a 30s
	// timeout if nil.
	HTTP *http.Client

	// id is an atomically-incremented JSON-RPC request id.
	id atomic.Int64
}

type rpcRequest struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      int64          `json:"id"`
	Method  string         `json:"method"`
	Params  map[string]any `json:"params"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (e *rpcError) Error() string {
	return fmt.Sprintf("mcp: %d %s", e.Code, e.Message)
}

// CallTool implements MCPClient.
func (c *HTTPClient) CallTool(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	if c.Endpoint == "" {
		return nil, fmt.Errorf("openproject: MCPClient endpoint not configured")
	}
	cli := c.HTTP
	if cli == nil {
		cli = &http.Client{Timeout: 30 * time.Second}
	}

	req := rpcRequest{
		JSONRPC: "2.0",
		ID:      c.id.Add(1),
		Method:  "tools/call",
		Params: map[string]any{
			"name":      name,
			"arguments": args,
		},
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal mcp request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build mcp request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	if c.APIKey != "" {
		httpReq.Header.Set("apikey", c.APIKey)
	}

	resp, err := cli.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("call mcp tool %q: %w", name, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read mcp response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("mcp tool %q returned HTTP %d: %s", name, resp.StatusCode, truncate(string(raw), 256))
	}

	var rpc rpcResponse
	if err := json.Unmarshal(raw, &rpc); err != nil {
		return nil, fmt.Errorf("decode mcp response: %w (body=%s)", err, truncate(string(raw), 256))
	}
	if rpc.Error != nil {
		return nil, rpc.Error
	}
	if len(rpc.Result) == 0 {
		return map[string]any{}, nil
	}
	var wrapper map[string]any
	if err := json.Unmarshal(rpc.Result, &wrapper); err != nil {
		return nil, fmt.Errorf("decode mcp result: %w", err)
	}
	// MCP tools/call responses wrap the tool's payload. Prefer
	// `structuredContent` (the parsed object); fall back to parsing
	// the first text item in `content` as JSON; fall back to the
	// raw wrapper.
	if sc, ok := wrapper["structuredContent"].(map[string]any); ok {
		return sc, nil
	}
	if contents, ok := wrapper["content"].([]any); ok && len(contents) > 0 {
		if item, ok := contents[0].(map[string]any); ok {
			if text, ok := item["text"].(string); ok && text != "" {
				var parsed map[string]any
				if err := json.Unmarshal([]byte(text), &parsed); err == nil {
					return parsed, nil
				}
			}
		}
	}
	return wrapper, nil
}

// nopClient is a MCPClient that returns success without making any
// network calls. Used by the Sink when dry-run mode is enabled.
type nopClient struct{}

// CallTool implements MCPClient.
func (nopClient) CallTool(_ context.Context, _ string, _ map[string]any) (map[string]any, error) {
	return map[string]any{}, nil
}

// NopClient returns a MCPClient that silently drops all calls. Intended
// for use with Config.DryRun.
func NopClient() MCPClient { return nopClient{} }

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
