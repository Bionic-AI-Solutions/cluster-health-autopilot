// Copyright 2026 Agentic SRE contributors
// SPDX-License-Identifier: Apache-2.0

// Package report renders srenix output for transport to external destinations.
//
// Slack is the first (and currently only) destination. Other webhooks
// (Teams, PagerDuty, custom) can be added by mirroring the FormatSlack →
// Post pattern; the JSON payload shape is intentionally simple and
// destination-neutral.
package report

import (
	"bytes"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/srenix-ai/agentic-sre/internal/diagnose"
	"github.com/srenix-ai/agentic-sre/internal/fix"
	"github.com/srenix-ai/agentic-sre/internal/probe"
	"github.com/srenix-ai/agentic-sre/pkg/ai"
)

// SilenceLinkConfig carries everything FormatSlack needs to mint the
// two signed one-click silence links rendered under each critical
// "needs human" finding. It is OPTIONAL: when nil — or when any of the
// signing key / base URL is absent — FormatSlack emits NO silence links
// and the output is byte-identical to the pre-silence-link render. This
// keeps OSS-only / air-gapped installs (no approval-server, no signing
// key) graceful.
//
// The signer/baseURL/durations are threaded from the watcher Config the
// same way the approval base URL is threaded for click-to-fix links.
type SilenceLinkConfig struct {
	// PrivateKey signs the silence tokens. Required (nil disables links).
	PrivateKey ed25519.PrivateKey
	// KeyID is stamped into the JWT header (kid). Default "default-1".
	KeyID string
	// BaseURL is the approval-server external base
	// (e.g. https://srenix-approve.example.com). Required (empty disables).
	BaseURL string
	// ShortDur is the subject-scoped "Silence 24h" window. Default 24h.
	ShortDur time.Duration
	// LongDur is the class-scoped "Silence class (90d)" window.
	// Default 2160h (90d).
	LongDur time.Duration
}

// Enabled reports whether the config can mint links — a private key of
// the right size AND a non-empty base URL. Used by both the diagnose-path
// renderer (slack.go) and the watcher delta path (internal/watcher) to
// gate silence-link minting; the renderers fall back gracefully when false.
func (c *SilenceLinkConfig) Enabled() bool {
	return c != nil && len(c.PrivateKey) == ed25519.PrivateKeySize && c.BaseURL != ""
}

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
//	*Agentic SRE* — <date> <time> UTC
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
	return FormatSlackWithSilence(results, diagnostics, fixResults, autopilot, nil)
}

// FormatSlackWithSilence is FormatSlack plus optional one-click silence
// links under each critical "needs human" finding. When silenceCfg is
// nil or not fully configured (no signing key / no base URL), it behaves
// exactly like FormatSlack — no links, unchanged output. FormatSlack is
// kept as a thin wrapper so existing callers that don't mint links stay
// untouched.
func FormatSlackWithSilence(results []probe.Result, diagnostics []diagnose.Diagnostic, fixResults []fix.Result, autopilot bool, silenceCfg *SilenceLinkConfig) SlackPayload {
	overall, color, headline := overallStatus(results)
	now := time.Now().UTC()

	var b strings.Builder
	fmt.Fprintf(&b, "*Agentic SRE* — %s\n", now.Format("2006-01-02 15:04:05 UTC"))
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
		now := time.Now()
		for _, f := range criticals {
			fmt.Fprintf(&b, "• *%s:* %s\n", f.Component, f.Message)
			if f.Investigation != "" {
				fmt.Fprintf(&b, "   🔬 _%s_\n", f.Investigation)
			}
			if f.Remediation != "" {
				fmt.Fprintf(&b, "   _Remediation:_ %s\n", f.Remediation)
			}
			// One-click silence links — only when a signer + approval
			// base URL are configured. OSS-only / air-gapped installs
			// (no key, no approval-server) take the no-op path and the
			// render is byte-identical to before.
			if line := silenceLinkLine(silenceCfg, f, now); line != "" {
				b.WriteString(line)
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
			if d.Remediation != "" {
				fmt.Fprintf(&b, "  _→ %s_\n", d.Remediation)
			}
			// Layer-2 investigation summary (OSS rule-based or paid LLM).
			if d.Investigation != "" {
				fmt.Fprintf(&b, "  🔬 _%s_\n", d.Investigation)
			}
			// AI enrichment block — only rendered when Srenix Enterprise has populated
			// it. OSS-only deployments never see this branch fire.
			if d.Enrichment != "" {
				fmt.Fprintf(&b, "  🤖 _%s_\n", d.Enrichment)
			}
		}
	}

	footer := "K8s Agentic SRE"
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

// silenceLinkLine renders the "🔕 Silence 24h · 🔕 Silence class (90d)"
// row for one critical finding, or "" when silence links are disabled
// or minting fails (graceful — a finding never loses its main render
// just because a link couldn't be signed).
//
// probe.Finding has no separate Source/Subject — Component is the
// finding's identity. We use it for BOTH the subject-scoped matcher
// (matcher.{source,subject} = Component → snooze THIS finding) and the
// class-scoped matcher (matcher.source = Component → mute the whole
// class). When the analyzer encodes its name + a per-object subject into
// Component (e.g. "TLSSecretMismatch: ingress/foo"), the class link is
// still Source-only on the same string; this is a defensible default
// until probe.Finding grows distinct Source/Subject fields.
func silenceLinkLine(cfg *SilenceLinkConfig, f probe.Finding, now time.Time) string {
	if !cfg.Enabled() {
		return ""
	}
	short := cfg.ShortDur
	if short <= 0 {
		short = DefaultSilenceShortDuration
	}
	long := cfg.LongDur
	if long <= 0 {
		long = DefaultSilenceLongDuration
	}
	kid := cfg.KeyID
	if kid == "" {
		kid = "default-1"
	}
	links, err := ai.MintSilenceLinks(cfg.PrivateKey, kid, cfg.BaseURL, ai.SilenceLinkRequest{
		Source:   f.Component,
		Subject:  f.Component,
		ShortDur: short,
		LongDur:  long,
	}, now)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("   🔕 <%s|Silence 24h> · 🔕 <%s|Silence class (90d)>\n",
		links.SubjectURL, links.ClassURL)
}

// Default silence windows when SilenceLinkConfig leaves them zero.
const (
	// DefaultSilenceShortDuration is the subject-scoped snooze window.
	DefaultSilenceShortDuration = 24 * time.Hour
	// DefaultSilenceLongDuration is the class-scoped mute window (90d).
	DefaultSilenceLongDuration = 2160 * time.Hour
)

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
	if body := strings.TrimSpace(string(respBody)); body != "ok" {
		return fmt.Errorf("webhook rejected payload: %s", body)
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
