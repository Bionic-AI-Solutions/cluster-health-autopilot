// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package openproject

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/ticketing"
)

// providerName is the value returned by Sink.Provider() and stored in
// DriftReport.status.ticket.provider. Lowercase, matches the convention
// other CHA components use.
const providerName = "openproject"

// Config holds Sink-construction parameters. Wired from Helm values.yaml
// in cmd/cha or the watcher bootstrap.
//
// All ID fields accept either numeric strings ("6") or names that the
// OpenProject MCP server resolves — match the catalog values discovered
// via list_projects / list_types / list_priorities / list_statuses.
type Config struct {
	// ProjectID is the OpenProject project ID (numeric string, e.g. "6"
	// for the Demo project). REQUIRED for create_work_package.
	ProjectID string

	// TypeID is the OpenProject type ID (numeric string, e.g. "36" for
	// Task). REQUIRED for create_work_package. Look up via list_types.
	TypeID string

	// SeverityPriority maps CHA severities ("critical" / "warning" /
	// "info") to OpenProject priority IDs (e.g. "75" for Immediate,
	// "74" for High, "73" for Normal). Missing severities skip the
	// priority_id field and let OpenProject use the project default.
	SeverityPriority map[string]string

	// ClosedStatusID is the OpenProject status ID to transition to when
	// CHA resolves a ticket. E.g. "82" for "Closed". Look up via
	// list_statuses (rows where is_closed=true).
	ClosedStatusID string

	// Labels are appended to the work package description as a markdown
	// footer line, since OpenProject's create_work_package tool does
	// not accept a native labels field. CHA's global Helm-configured
	// labels (e.g. ["cha", "auto-filed"]) flow through here.
	Labels []string

	// WebURLPrefix is the operator-facing base URL of OpenProject up to
	// (but not including) the work-package path segment. CHA appends
	// "/work_packages/<id>" to build the TicketRef.URL. Example:
	// "https://op.bionicaisolutions.com". Empty leaves TicketRef.URL
	// blank — operators can still open the ticket via key.
	WebURLPrefix string

	// ToolNames lets operators override the MCP tool names if their
	// OpenProject MCP server uses non-default names. Empty fields fall
	// back to defaults documented in DefaultToolNames.
	ToolNames ToolNames

	// DryRun makes the Sink log intended operations without calling
	// the MCP server. Wires NopClient() as the transport.
	DryRun bool
}

// ToolNames lists the MCP tools the Sink invokes. Defaults match the
// docker4zerocool/mcp-servers-openproject image surfaced via tools/list
// at the in-cluster service. Override for forks or community MCP servers
// with different names.
type ToolNames struct {
	CreateWorkPackage       string
	UpdateWorkPackage       string
	UpdateWorkPackageStatus string
	AddWorkPackageComment   string
	GetWorkPackage          string
}

// DefaultToolNames returns the MCP tool names CHA uses unless overridden.
// Verified 2026-05-22 against mcp-openproject-server:8006/mcp tools/list.
func DefaultToolNames() ToolNames {
	return ToolNames{
		CreateWorkPackage:       "create_work_package",
		UpdateWorkPackage:       "update_work_package",
		UpdateWorkPackageStatus: "update_work_package_status",
		AddWorkPackageComment:   "add_work_package_comment",
		GetWorkPackage:          "get_work_package",
	}
}

// Sink is the OpenProject implementation of ticketing.Sink. Construct
// with New.
type Sink struct {
	cfg    Config
	client MCPClient
}

// New constructs a Sink. When cfg.DryRun is true, the supplied client
// is ignored and replaced with NopClient(). Tool name defaults are
// applied for any zero-valued fields in cfg.ToolNames.
func New(cfg Config, client MCPClient) *Sink {
	if cfg.DryRun {
		client = NopClient()
	}
	defaults := DefaultToolNames()
	if cfg.ToolNames.CreateWorkPackage == "" {
		cfg.ToolNames.CreateWorkPackage = defaults.CreateWorkPackage
	}
	if cfg.ToolNames.UpdateWorkPackage == "" {
		cfg.ToolNames.UpdateWorkPackage = defaults.UpdateWorkPackage
	}
	if cfg.ToolNames.UpdateWorkPackageStatus == "" {
		cfg.ToolNames.UpdateWorkPackageStatus = defaults.UpdateWorkPackageStatus
	}
	if cfg.ToolNames.AddWorkPackageComment == "" {
		cfg.ToolNames.AddWorkPackageComment = defaults.AddWorkPackageComment
	}
	if cfg.ToolNames.GetWorkPackage == "" {
		cfg.ToolNames.GetWorkPackage = defaults.GetWorkPackage
	}
	return &Sink{cfg: cfg, client: client}
}

// Provider implements ticketing.Sink.
func (s *Sink) Provider() string { return providerName }

// Upsert implements ticketing.Sink. Because the OpenProject MCP server
// does not expose a portable "find by custom-field" tool CHA can rely
// on, dedup is handled one level up by the routing adapter via
// DriftReport.status.ticket. This method always issues a create call;
// callers must skip it when a TicketRef already exists for the
// fingerprint.
func (s *Sink) Upsert(ctx context.Context, t ticketing.Ticket) (ticketing.TicketRef, error) {
	if t.Fingerprint == "" {
		return ticketing.TicketRef{}, fmt.Errorf("openproject: ticket fingerprint is required")
	}
	if s.cfg.ProjectID == "" {
		return ticketing.TicketRef{}, fmt.Errorf("openproject: ProjectID is required")
	}
	if s.cfg.TypeID == "" {
		return ticketing.TicketRef{}, fmt.Errorf("openproject: TypeID is required (look up via list_types)")
	}

	args := map[string]any{
		"project_id":  s.cfg.ProjectID,
		"subject":     t.Title,
		"type_id":     s.cfg.TypeID,
		"description": appendLabelFooter(t.Body, s.mergeLabels(t.Labels), t.Fingerprint),
	}
	if pri := s.priorityFor(t.Severity); pri != "" {
		args["priority_id"] = pri
	}

	if s.cfg.DryRun {
		log.Printf("ticketing/openproject: DRY-RUN %s args=%v", s.cfg.ToolNames.CreateWorkPackage, args)
	}

	res, err := s.client.CallTool(ctx, s.cfg.ToolNames.CreateWorkPackage, args)
	if err != nil {
		return ticketing.TicketRef{}, fmt.Errorf("openproject: %s: %w", s.cfg.ToolNames.CreateWorkPackage, err)
	}

	wp := extractWorkPackage(res)
	ref := ticketing.TicketRef{Provider: providerName}
	if k, ok := stringField(wp, "id"); ok {
		ref.Key = k
	} else if i, ok := intField(wp, "id"); ok {
		ref.Key = fmt.Sprintf("%d", i)
	}
	if u, ok := stringField(wp, "url"); ok {
		ref.URL = u
	} else if u, ok := stringField(wp, "self_link"); ok {
		ref.URL = u
	} else if u, ok := stringField(wp, "selfLink"); ok {
		ref.URL = u
	}
	if ref.URL == "" && ref.Key != "" && s.cfg.WebURLPrefix != "" {
		ref.URL = strings.TrimRight(s.cfg.WebURLPrefix, "/") + "/work_packages/" + ref.Key
	}

	if ref.Key == "" {
		return ref, fmt.Errorf("openproject: %s returned no work-package id (result=%v)", s.cfg.ToolNames.CreateWorkPackage, res)
	}
	return ref, nil
}

// Resolve implements ticketing.Sink. Transitions the work package to
// the configured Closed status via update_work_package_status and
// passes reason as the optional comment.
func (s *Sink) Resolve(ctx context.Context, ref ticketing.TicketRef, reason string) error {
	if ref.Key == "" {
		return fmt.Errorf("openproject: resolve: empty work_package_id")
	}
	if s.cfg.ClosedStatusID == "" {
		return fmt.Errorf("openproject: resolve: ClosedStatusID not configured (look up via list_statuses)")
	}
	args := map[string]any{
		"work_package_id": ref.Key,
		"status_id":       s.cfg.ClosedStatusID,
	}
	if reason != "" {
		args["comment"] = reason
	}
	if s.cfg.DryRun {
		log.Printf("ticketing/openproject: DRY-RUN %s args=%v", s.cfg.ToolNames.UpdateWorkPackageStatus, args)
	}
	if _, err := s.client.CallTool(ctx, s.cfg.ToolNames.UpdateWorkPackageStatus, args); err != nil {
		return fmt.Errorf("openproject: %s: %w", s.cfg.ToolNames.UpdateWorkPackageStatus, err)
	}
	return nil
}

// Comment implements ticketing.Sink.
func (s *Sink) Comment(ctx context.Context, ref ticketing.TicketRef, body string) error {
	if ref.Key == "" {
		return fmt.Errorf("openproject: comment: empty work_package_id")
	}
	if body == "" {
		return nil
	}
	args := map[string]any{
		"work_package_id": ref.Key,
		"comment":         body,
	}
	if s.cfg.DryRun {
		log.Printf("ticketing/openproject: DRY-RUN %s args=%v", s.cfg.ToolNames.AddWorkPackageComment, args)
	}
	if _, err := s.client.CallTool(ctx, s.cfg.ToolNames.AddWorkPackageComment, args); err != nil {
		return fmt.Errorf("openproject: %s: %w", s.cfg.ToolNames.AddWorkPackageComment, err)
	}
	return nil
}

// priorityFor returns the OpenProject priority ID for a CHA severity,
// or "" to let OpenProject use the project default.
func (s *Sink) priorityFor(severity string) string {
	if v, ok := s.cfg.SeverityPriority[severity]; ok {
		return v
	}
	return ""
}

// mergeLabels combines per-ticket labels with global Config.Labels and
// returns a deduplicated slice. Preserves order: ticket labels first,
// global labels after.
func (s *Sink) mergeLabels(ticketLabels []string) []string {
	seen := make(map[string]struct{}, len(ticketLabels)+len(s.cfg.Labels))
	out := make([]string, 0, len(ticketLabels)+len(s.cfg.Labels))
	for _, l := range ticketLabels {
		if _, ok := seen[l]; ok {
			continue
		}
		seen[l] = struct{}{}
		out = append(out, l)
	}
	for _, l := range s.cfg.Labels {
		if _, ok := seen[l]; ok {
			continue
		}
		seen[l] = struct{}{}
		out = append(out, l)
	}
	return out
}

// appendLabelFooter inserts CHA labels + fingerprint into the
// description body as markdown. OpenProject's create_work_package tool
// does not accept a native labels field, so this is the only place
// they get persisted for filtering.
func appendLabelFooter(body string, labels []string, fingerprint string) string {
	if len(labels) == 0 && fingerprint == "" {
		return body
	}
	var b strings.Builder
	b.WriteString(body)
	if !strings.HasSuffix(body, "\n") {
		b.WriteString("\n")
	}
	b.WriteString("\n---\n\n")
	if len(labels) > 0 {
		b.WriteString("**Labels:** ")
		b.WriteString(strings.Join(labels, ", "))
		b.WriteString("\n\n")
	}
	if fingerprint != "" {
		fmt.Fprintf(&b, "**CHA Fingerprint:** `%s`\n", fingerprint)
	}
	return b.String()
}

// extractWorkPackage normalizes the various shapes the
// create_work_package result may take. The MCP server returns
// {success, work_package: {...}}; raw FastMCP shapes may put the
// payload at the top level. Try both.
func extractWorkPackage(res map[string]any) map[string]any {
	if res == nil {
		return nil
	}
	if wp, ok := res["work_package"].(map[string]any); ok {
		return wp
	}
	return res
}

func stringField(m map[string]any, key string) (string, bool) {
	if m == nil {
		return "", false
	}
	v, ok := m[key]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	if !ok {
		return "", false
	}
	return s, s != ""
}

func intField(m map[string]any, key string) (int64, bool) {
	if m == nil {
		return 0, false
	}
	v, ok := m[key]
	if !ok {
		return 0, false
	}
	switch x := v.(type) {
	case float64:
		return int64(x), true
	case int64:
		return x, true
	case int:
		return int64(x), true
	}
	return 0, false
}
