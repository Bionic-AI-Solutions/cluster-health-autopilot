// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

package openproject

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/srenix-ai/agentic-sre/pkg/ticketing"
)

// recordingClient is a MCPClient that captures every CallTool invocation
// and returns a canned response. Tests inspect Calls to assert payload
// shape.
type recordingClient struct {
	Response map[string]any
	Err      error
	Calls    []recordedCall
}

type recordedCall struct {
	Name string
	Args map[string]any
}

func (r *recordingClient) CallTool(_ context.Context, name string, args map[string]any) (map[string]any, error) {
	r.Calls = append(r.Calls, recordedCall{Name: name, Args: args})
	if r.Err != nil {
		return nil, r.Err
	}
	return r.Response, nil
}

func sampleTicket() ticketing.Ticket {
	return ticketing.Ticket{
		Fingerprint: "srenix-deadbeef0badc0de",
		Title:       "Secret/mcp/openproject-secrets/openproject-url missing",
		Body:        "## Diagnostic\nKey absent.\n\n## Remediation\nRestore via Vault.",
		Severity:    ticketing.SeverityCritical,
		Subject:     "Secret/mcp/openproject-secrets/openproject-url",
		Source:      "SecretKeyMissing",
		Cluster:     "gpu-cluster",
		Labels:      []string{"ceph"},
		OpenedAt:    time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC),
	}
}

func TestUpsertSendsCorrectMCPSchema(t *testing.T) {
	cli := &recordingClient{
		Response: map[string]any{
			"work_package": map[string]any{
				"id":        float64(1287),
				"self_link": "https://op.example/work_packages/1287",
			},
		},
	}
	sink := New(Config{
		ProjectID: "6",
		TypeID:    "36",
		SeverityPriority: map[string]string{
			ticketing.SeverityCritical: "75",
		},
		Labels: []string{"srenix", "auto-filed"},
	}, cli)

	ref, err := sink.Upsert(context.Background(), sampleTicket())
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if ref.Provider != providerName {
		t.Errorf("provider=%q want %q", ref.Provider, providerName)
	}
	if ref.Key != "1287" {
		t.Errorf("key=%q want 1287", ref.Key)
	}
	if ref.URL != "https://op.example/work_packages/1287" {
		t.Errorf("url=%q want OP work-package URL", ref.URL)
	}

	if len(cli.Calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(cli.Calls))
	}
	call := cli.Calls[0]
	if call.Name != "create_work_package" {
		t.Errorf("tool=%q want create_work_package", call.Name)
	}
	if call.Args["project_id"] != "6" {
		t.Errorf("project_id=%v want 6", call.Args["project_id"])
	}
	if call.Args["type_id"] != "36" {
		t.Errorf("type_id=%v want 36", call.Args["type_id"])
	}
	if call.Args["priority_id"] != "75" {
		t.Errorf("priority_id=%v want 75 (Immediate)", call.Args["priority_id"])
	}
	if call.Args["subject"] == "" {
		t.Error("subject should be set")
	}
	// Labels + fingerprint should be in the description footer.
	desc, _ := call.Args["description"].(string)
	if !strings.Contains(desc, "**Labels:** ceph, srenix, auto-filed") {
		t.Errorf("description missing labels footer: %q", desc)
	}
	if !strings.Contains(desc, "srenix-deadbeef0badc0de") {
		t.Errorf("description missing fingerprint footer: %q", desc)
	}
}

func TestUpsertRejectsMissingRequiredConfig(t *testing.T) {
	cases := []struct {
		name string
		cfg  Config
	}{
		{"no project", Config{TypeID: "36"}},
		{"no type", Config{ProjectID: "6"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sink := New(tc.cfg, &recordingClient{})
			if _, err := sink.Upsert(context.Background(), sampleTicket()); err == nil {
				t.Error("expected error for missing required config")
			}
		})
	}
}

func TestUpsertRejectsEmptyFingerprint(t *testing.T) {
	sink := New(Config{ProjectID: "6", TypeID: "36"}, &recordingClient{})
	tk := sampleTicket()
	tk.Fingerprint = ""
	if _, err := sink.Upsert(context.Background(), tk); err == nil {
		t.Fatal("expected error for empty fingerprint")
	}
}

func TestUpsertFailsWhenResponseMissingID(t *testing.T) {
	sink := New(Config{ProjectID: "6", TypeID: "36"}, &recordingClient{Response: map[string]any{
		"work_package": map[string]any{},
	}})
	if _, err := sink.Upsert(context.Background(), sampleTicket()); err == nil {
		t.Fatal("expected error when MCP returns no id")
	}
}

func TestResolveCallsUpdateStatusTool(t *testing.T) {
	cli := &recordingClient{Response: map[string]any{}}
	sink := New(Config{ClosedStatusID: "82"}, cli)
	ref := ticketing.TicketRef{Provider: providerName, Key: "42", URL: "u"}
	if err := sink.Resolve(context.Background(), ref, "drift cleared"); err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if cli.Calls[0].Name != "update_work_package_status" {
		t.Errorf("tool=%q want update_work_package_status", cli.Calls[0].Name)
	}
	if cli.Calls[0].Args["work_package_id"] != "42" {
		t.Errorf("work_package_id=%v want 42", cli.Calls[0].Args["work_package_id"])
	}
	if cli.Calls[0].Args["status_id"] != "82" {
		t.Errorf("status_id=%v want 82", cli.Calls[0].Args["status_id"])
	}
	if cli.Calls[0].Args["comment"] != "drift cleared" {
		t.Errorf("comment=%v want 'drift cleared'", cli.Calls[0].Args["comment"])
	}
}

func TestResolveRequiresClosedStatusID(t *testing.T) {
	sink := New(Config{}, &recordingClient{})
	if err := sink.Resolve(context.Background(), ticketing.TicketRef{Key: "1"}, "x"); err == nil {
		t.Fatal("expected error when ClosedStatusID not configured")
	}
}

func TestResolveRejectsEmptyKey(t *testing.T) {
	sink := New(Config{ClosedStatusID: "82"}, &recordingClient{})
	if err := sink.Resolve(context.Background(), ticketing.TicketRef{}, "x"); err == nil {
		t.Fatal("expected error for empty key")
	}
}

func TestCommentUsesAddWorkPackageCommentTool(t *testing.T) {
	cli := &recordingClient{Response: map[string]any{}}
	sink := New(Config{}, cli)
	if err := sink.Comment(context.Background(), ticketing.TicketRef{Key: "1"}, "hello"); err != nil {
		t.Fatalf("Comment: %v", err)
	}
	if cli.Calls[0].Name != "add_work_package_comment" {
		t.Errorf("tool=%q want add_work_package_comment", cli.Calls[0].Name)
	}
	if cli.Calls[0].Args["work_package_id"] != "1" {
		t.Errorf("work_package_id=%v want 1", cli.Calls[0].Args["work_package_id"])
	}
	if cli.Calls[0].Args["comment"] != "hello" {
		t.Errorf("comment=%v want hello", cli.Calls[0].Args["comment"])
	}
}

func TestCommentSkipsEmptyBody(t *testing.T) {
	cli := &recordingClient{}
	sink := New(Config{}, cli)
	if err := sink.Comment(context.Background(), ticketing.TicketRef{Key: "1"}, ""); err != nil {
		t.Fatalf("Comment: %v", err)
	}
	if len(cli.Calls) != 0 {
		t.Errorf("empty body should not invoke MCP, got %d calls", len(cli.Calls))
	}
}

func TestDryRunUsesNopClient(t *testing.T) {
	cli := &recordingClient{Err: context.DeadlineExceeded} // would error if used
	sink := New(Config{ProjectID: "6", TypeID: "36", DryRun: true}, cli)
	if _, err := sink.Upsert(context.Background(), sampleTicket()); err == nil {
		t.Fatal("dry-run should still error on missing id (nop returns empty map)")
	}
	if len(cli.Calls) != 0 {
		t.Errorf("dry-run must not call the wrapped client, got %d calls", len(cli.Calls))
	}
}

func TestToolNameOverride(t *testing.T) {
	cli := &recordingClient{Response: map[string]any{"work_package": map[string]any{"id": float64(1)}}}
	sink := New(Config{
		ProjectID: "6",
		TypeID:    "36",
		ToolNames: ToolNames{CreateWorkPackage: "op_create"},
	}, cli)
	if _, err := sink.Upsert(context.Background(), sampleTicket()); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if cli.Calls[0].Name != "op_create" {
		t.Errorf("tool=%q want op_create (override)", cli.Calls[0].Name)
	}
}
