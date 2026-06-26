// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"strings"
	"testing"
)

// Sprint 4.2 — required-flag validation.

func TestValidateTicketingOpts_EmptyProviderIsNoop(t *testing.T) {
	// Empty provider returns nil from buildTicketingConfig — no validation
	// reached. The validator itself is only called when Provider is set,
	// so this test guards the public buildTicketingConfig contract.
	cfg, err := buildTicketingConfig(ticketingOpts{Provider: ""})
	if err != nil {
		t.Errorf("empty provider should be a no-op; got err=%v", err)
	}
	if cfg.Sink != nil {
		t.Errorf("empty provider should yield nil Sink; got %+v", cfg)
	}
}

func TestValidateTicketingOpts_OpenProjectRequiresMCPURL(t *testing.T) {
	t.Setenv("TICKETING_MCP_API_KEY", "k")
	err := validateTicketingOpts(ticketingOpts{
		Provider:  "openproject",
		ProjectID: "proj-1",
	})
	if err == nil || !strings.Contains(err.Error(), "ticketing-mcp-url") {
		t.Errorf("missing --ticketing-mcp-url should be flagged; got %v", err)
	}
}

func TestValidateTicketingOpts_OpenProjectRequiresProjectID(t *testing.T) {
	t.Setenv("TICKETING_MCP_API_KEY", "k")
	err := validateTicketingOpts(ticketingOpts{
		Provider: "openproject",
		MCPURL:   "https://mcp.example.com",
	})
	if err == nil || !strings.Contains(err.Error(), "ticketing-project") {
		t.Errorf("missing --ticketing-project should be flagged; got %v", err)
	}
}

func TestValidateTicketingOpts_OpenProjectAcceptsMissingAPIKey(t *testing.T) {
	// In-cluster MCP traffic bypasses Kong key-auth (Kong only enforces
	// auth on external ingress paths). A missing API key is therefore NOT
	// a config error — auth, when needed, fails at the MCP request and
	// surfaces a clear 401/403 in the watcher log.
	_ = os.Unsetenv("TICKETING_MCP_API_KEY")
	err := validateTicketingOpts(ticketingOpts{
		Provider:  "openproject",
		MCPURL:    "http://mcp.svc.cluster.local",
		ProjectID: "proj-1",
	})
	if err != nil {
		t.Errorf("missing API key should NOT block validation for in-cluster MCP; got %v", err)
	}
}

func TestValidateTicketingOpts_OpenProjectFullySet_OK(t *testing.T) {
	t.Setenv("TICKETING_MCP_API_KEY", "k")
	err := validateTicketingOpts(ticketingOpts{
		Provider:  "openproject",
		MCPURL:    "https://mcp.example.com",
		ProjectID: "proj-1",
	})
	if err != nil {
		t.Errorf("fully-configured openproject should validate cleanly; got %v", err)
	}
}

func TestValidateTicketingOpts_UnknownProvider(t *testing.T) {
	err := validateTicketingOpts(ticketingOpts{Provider: "jira"})
	if err == nil || !strings.Contains(err.Error(), "unsupported ticketing provider") {
		t.Errorf("unknown provider should be flagged; got %v", err)
	}
}
