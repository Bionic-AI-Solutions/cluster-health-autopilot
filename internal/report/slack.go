// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

// Package report renders cha output for transport to external destinations.
//
// Slack is the first (and currently only) destination. Other webhooks
// (Teams, PagerDuty, custom) can be added by mirroring the FormatSlack →
// Post pattern; the JSON payload shape is intentionally simple and
// destination-neutral.
package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/diagnose"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/fix"
	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/probe"
)

// SlackPayload is the JSON shape Slack incoming-webhooks accept.
//
// We use the legacy attachments shape (color, text, footer, ts) rather
// than blocks/blockKit because: it renders cleanly across desktop/mobile/
// email-summary, the bash version uses the same shape so messages look
// identical, and any custom dispatcher (Teams, internal webhook) can
// trivially down-convert this to its own format.
type SlackPayload struct {
	Username    string            `json:"username,omitempty"`
	IconEmoji   string            `json:"icon_emoji,omitempty"`
	Attachments []SlackAttachment `json:"attachments"`
}

// SlackAttachment is a single coloured block in the message.
type SlackAttachment struct {
	Color    string   `json:"color"` // "good" | "warning" | "danger"
	Text     string   `json:"text"`  // mrkdwn-formatted body
	Footer   string   `json:"footer,omitempty"`
	Ts       int64    `json:"ts,omitempty"`
	MrkdwnIn []string `json:"mrkdwn_in,omitempty"`
}

// FormatSlack renders the diagnose + remediate results into the same
// section structure the in-cluster bash health-report.sh produces:
//
//	*Cluster Health Autopilot* — <date> <time> UTC
//	<emoji> *Overall Status: <state>*
//
//	*Component Status:*
//	• ...
//
//	*🔧 Automated Fixes Applied (N):*  (only when N > 0)
//	  ...
//
//	*🔴 Critical Issues (N) — needs human:*  (only when N > 0)
//	  ...
//
//	*🟡 Warnings (N):*  (only when N > 0)
//	  ...
//
//	*Diagnostics (N):*  (only when N > 0)
//	  🔎 ...
//
// fixResults may be empty (read-only diagnose) or nil.
func FormatSlack(results []probe.Result, diagnostics []diagnose.Diagnostic, fixResults []fix.Result, autopilot bool) SlackPayload {
	overall, color, headline := overallStatus(results)
	now := time.Now().UTC()

	var b strings.Builder
	fmt.Fprintf(&b, "*Cluster Health Autopilot* — %s\n", now.Format("2006-01-02 15:04:05 UTC"))
	fmt.Fprintf(&b, "%s *Overall Status: %s*\n\n", overall.emoji, overall.label)
	fmt.Fprintf(&b, "%s\n\n", headline)

	b.WriteString("*Component Status:*\n")
	for _, r := range results {
		fmt.Fprintf(&b, "• *%s:* %s\n  %s\n", r.Component.Component, statusEmoji(r.Component.Status), r.Component.Detail)
	}

	// Fixes applied (only when any fixer actually acted).
	totalActions := 0
	for _, fr := range fixResults {
		totalActions += len(fr.Actions)
	}
	if totalActions > 0 {
		fmt.Fprintf(&b, "\n*🔧 Automated Fixes Applied (%d):*\n", totalActions)
		for _, fr := range fixResults {
			for _, a := range fr.Actions {
				fmt.Fprintf(&b, "• %s — `%s`\n", a.Description, a.Object)
			}
		}
	}

	// Critical issues per component.
	var criticals []probe.Finding
	var warnings []probe.Finding
	for _, r := range results {
		for _, f := range r.Findings {
			switch f.Severity {
			case probe.SeverityCritical:
				criticals = append(criticals, f)
			case probe.SeverityWarning:
				warnings = append(warnings, f)
			}
		}
	}

	if len(criticals) > 0 {
		fmt.Fprintf(&b, "\n*🔴 Critical Issues (%d) — needs human:*\n", len(criticals))
		for _, f := range criticals {
			fmt.Fprintf(&b, "• *%s:* %s\n", f.Component, f.Message)
			if f.Remediation != "" {
				fmt.Fprintf(&b, "   _Remediation:_ %s\n", f.Remediation)
			}
		}
	}

	if len(warnings) > 0 {
		fmt.Fprintf(&b, "\n*🟡 Warnings (%d):*\n", len(warnings))
		for _, f := range warnings {
			fmt.Fprintf(&b, "• *%s:* %s\n", f.Component, f.Message)
		}
	}

	if len(diagnostics) > 0 {
		fmt.Fprintf(&b, "\n*Diagnostics (%d):*\n", len(diagnostics))
		for _, d := range diagnostics {
			fmt.Fprintf(&b, "• 🔎 %s\n", d.Message)
		}
	}

	footer := "K8s Cluster Health Autopilot"
	if autopilot {
		footer += " (auto-remediation: ON)"
	} else {
		footer += " (read-only)"
	}

	return SlackPayload{
		Username:  "Cluster Health Monitor",
		IconEmoji: ":hospital:",
		Attachments: []SlackAttachment{{
			Color:    color,
			Text:     b.String(),
			Footer:   footer,
			Ts:       now.Unix(),
			MrkdwnIn: []string{"text"},
		}},
	}
}

// PostSlack POSTs the payload as JSON to the given webhook URL and
// returns nil on the canonical "ok" response.
func PostSlack(client *http.Client, webhookURL string, payload SlackPayload) error {
	if webhookURL == "" {
		return fmt.Errorf("slack webhook URL is empty")
	}
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal slack payload: %w", err)
	}
	req, err := http.NewRequest("POST", webhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("post webhook: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("webhook returned HTTP %d: %s", resp.StatusCode, string(respBody))
	}
	if strings.TrimSpace(string(respBody)) != "ok" {
		// Non-Slack webhooks may return JSON; only treat non-200 as fatal.
		return nil
	}
	return nil
}

// overall and helper types — kept package-private since FormatSlack is
// the only consumer.
type statusBadge struct {
	emoji string
	label string
}

// overallStatus inspects component statuses + finding severities and returns
// (badge, attachment-color, one-line headline).
func overallStatus(results []probe.Result) (statusBadge, string, string) {
	hasCritical := false
	hasDegraded := false
	for _, r := range results {
		switch r.Component.Status {
		case "CRITICAL", "PROBE_FAILED":
			hasCritical = true
		case "DEGRADED":
			hasDegraded = true
		}
		for _, f := range r.Findings {
			switch f.Severity {
			case probe.SeverityCritical:
				hasCritical = true
			case probe.SeverityWarning:
				hasDegraded = true
			}
		}
	}
	switch {
	case hasCritical:
		return statusBadge{"❌", "UNHEALTHY"}, "danger", "Critical issues detected. Immediate attention required."
	case hasDegraded:
		return statusBadge{"⚠️", "DEGRADED"}, "warning", "Warnings detected. Review recommended."
	default:
		return statusBadge{"✅", "HEALTHY"}, "good", "All systems operational. No action required."
	}
}

// statusEmoji renders a component's STATUS string with the badge that
// matches the bash version.
func statusEmoji(status string) string {
	switch status {
	case "HEALTHY":
		return "🟢 HEALTHY"
	case "DEGRADED":
		return "🟡 DEGRADED"
	case "CRITICAL":
		return "🔴 CRITICAL"
	case "PROBE_FAILED":
		return "🔴 PROBE_FAILED"
	case "SKIPPED":
		return "⚪ SKIPPED"
	default:
		return status
	}
}
