// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/internal/fix"
)

// AMAlert is one alert in the Alertmanager v2 API format.
// Alertmanager deduplicates, groups, silences, and routes these to configured
// receivers (Slack, PagerDuty, Teams, email, webhook, …).
//
// Label fingerprinting: Alertmanager identifies an alert by its Labels map.
// Two POSTs with identical labels but different annotations update the same
// alert's TTL — they don't fire duplicate notifications.
type AMAlert struct {
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     time.Time         `json:"startsAt,omitempty"`
	EndsAt       time.Time         `json:"endsAt,omitempty"`
	GeneratorURL string            `json:"generatorURL,omitempty"`
}

// PostAlertmanager fires a batch of alerts to the Alertmanager v2 API.
// url should be the base URL of Alertmanager (e.g. "http://alertmanager.pg:9093").
// Returns nil if Alertmanager accepted the payload (HTTP 200).
func PostAlertmanager(client *http.Client, url string, alerts []AMAlert) error {
	if url == "" || len(alerts) == 0 {
		return nil
	}
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	body, err := json.Marshal(alerts)
	if err != nil {
		return fmt.Errorf("marshal alertmanager payload: %w", err)
	}
	req, err := http.NewRequest("POST", url+"/api/v2/alerts", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build alertmanager request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("post to alertmanager: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("alertmanager returned HTTP %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// BuildActiveAlerts converts the current set of active issues (postFix state)
// into Alertmanager alerts. Every call refreshes the TTL so Alertmanager keeps
// alerts firing as long as CHA keeps posting them. If an issue disappears
// (watcher stops posting it), Alertmanager auto-resolves after ttl expires.
//
// Label scheme:
//
//	alertname  = "cha_issue"
//	severity   = critical | warning | info
//	subject    = the stable issue subject key
//	source     = probe/analyzer name
//	category   = probe | analyzer
//	cluster    = clusterName
func BuildActiveAlerts(active []DeltaDiag, clusterName string, ttl time.Duration) []AMAlert {
	now := time.Now().UTC()
	endsAt := now.Add(ttl)
	out := make([]AMAlert, 0, len(active))
	for _, d := range active {
		if d.Severity == "info" {
			continue // don't fire info-level issues
		}
		annotations := map[string]string{
			"summary":     d.Message,
			"remediation": d.Remediation,
			// `silence_snippet` is a ready-to-run kubectl command an
			// SRE can paste to suppress this exact finding-class for a
			// bounded duration. Surfaces via the Silence CRD (Phase 1c)
			// which the watch loop honors at the analyzer-filter layer.
			// Receivers (Slack/email) render this in the same block as
			// `remediation` so the operator can decide: fix → remediate,
			// noise → silence. Bounded to 24h by default; SRE edits
			// spec.until before applying.
			"silence_snippet": buildSilenceSnippet(d, 24*time.Hour),
		}
		// AI tier annotations — populated only when CHA-com active.
		// Alertmanager templates can reference {{ .Annotations.ai_enrichment }}
		// to render the narrative in receiver-specific formatting.
		if d.Enrichment != "" {
			annotations["ai_enrichment"] = d.Enrichment
		}
		if d.ApprovalURL != "" {
			annotations["proposed_fix_url"] = d.ApprovalURL
			// Symmetric deny URL (cha-com #17 symmetric one-shot
			// tokens). The Slack/email template renders both buttons
			// so the SRE has a one-click Deny alongside Approve.
			// Whichever endpoint the JTI lands at first wins; the
			// other is burned. Denial records a RAG outcome so the
			// proposer learns from rejections.
			annotations["proposed_deny_url"] = strings.Replace(d.ApprovalURL, "/approve?", "/deny?", 1)
		}
		if d.ProposedActionID != "" {
			annotations["proposed_action_id"] = d.ProposedActionID
		}
		out = append(out, AMAlert{
			Labels: map[string]string{
				"alertname": "cha_issue",
				"severity":  d.Severity,
				"subject":   truncateLabel(d.Subject),
				"source":    truncateLabel(d.Severity), // overridden below if Source available
				"cluster":   clusterName,
			},
			Annotations: annotations,
			StartsAt:    now,
			EndsAt:      endsAt,
		})
		// Fill source from Subject prefix (Probe/<name>/... or analyzer/<name>/...)
		if src := subjectSource(d.Subject); src != "" {
			out[len(out)-1].Labels["source"] = src
		}
	}
	return out
}

// BuildFixerAlerts fires one `cha_fixer_acted` alert per fixer that took
// action this cycle. Short TTL — these are informational and auto-resolve
// quickly. Alertmanager routes them to #ceph-alerts.
func BuildFixerAlerts(fixResults []fix.Result, clusterName string) []AMAlert {
	now := time.Now().UTC()
	endsAt := now.Add(30 * time.Minute)
	var out []AMAlert
	for _, fr := range fixResults {
		if len(fr.Actions) == 0 {
			continue
		}
		desc := ""
		if len(fr.Actions) > 0 {
			desc = fr.Actions[0].Description
			if len(fr.Actions) > 1 {
				desc += fmt.Sprintf(" (+%d more)", len(fr.Actions)-1)
			}
		}
		out = append(out, AMAlert{
			Labels: map[string]string{
				"alertname": "cha_fixer_acted",
				"severity":  "info",
				"fixer":     truncateLabel(fr.Fixer),
				"cluster":   clusterName,
			},
			Annotations: map[string]string{
				"summary": fmt.Sprintf("CHA %s applied %d action(s): %s", fr.Fixer, len(fr.Actions), desc),
			},
			StartsAt: now,
			EndsAt:   endsAt,
		})
	}
	return out
}

// PostActiveStateToAM is the high-level call from the watcher: builds and
// posts both the active-issue alerts and the fixer-action alerts in a single
// Alertmanager POST. Errors are logged but not fatal — Slack fallback
// (if configured) still fires.
func PostActiveStateToAM(client *http.Client, amURL string, active []DeltaDiag, fixResults []fix.Result, clusterName string, ttl time.Duration) {
	if amURL == "" {
		return
	}
	alerts := BuildActiveAlerts(active, clusterName, ttl)
	alerts = append(alerts, BuildFixerAlerts(fixResults, clusterName)...)
	if len(alerts) == 0 {
		return
	}
	if err := PostAlertmanager(client, amURL, alerts); err != nil {
		log.Printf("report: alertmanager post: %v", err)
	}
}

// subjectSource extracts a short source name from a DriftReport subject key.
// "Probe/Critical Services/Service: Langfuse Web" → "Critical Services"
// "ingress-coverage/ns/name/host"                 → "IngressCoverage"
// "FailingExternalSecrets/ns/name"                → "FailingExternalSecrets"
func subjectSource(subject string) string {
	for i := 0; i < len(subject); i++ {
		if subject[i] == '/' {
			prefix := subject[:i]
			if prefix == "Probe" && i+1 < len(subject) {
				rest := subject[i+1:]
				for j := 0; j < len(rest); j++ {
					if rest[j] == '/' {
						return rest[:j]
					}
				}
				return rest
			}
			return prefix
		}
	}
	return subject
}

func truncateLabel(s string) string {
	// Alertmanager label values must be ≤ 256 UTF-8 chars (practical limit).
	if len(s) <= 256 {
		return s
	}
	return s[:253] + "..."
}

// silenceSnippetName derives a stable Silence CR name from a finding
// subject. K8s names are ≤63 chars, lowercase alphanumeric + dashes;
// substitute "/" and other illegal characters with "-".
func silenceSnippetName(subject string) string {
	var b strings.Builder
	prev := byte('-')
	for i := 0; i < len(subject); i++ {
		c := subject[i]
		switch {
		case c >= 'A' && c <= 'Z':
			b.WriteByte(c + 32)
			prev = c + 32
		case (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9'):
			b.WriteByte(c)
			prev = c
		default:
			if prev != '-' {
				b.WriteByte('-')
				prev = '-'
			}
		}
	}
	name := strings.Trim(b.String(), "-")
	if len(name) > 50 {
		name = name[:50]
	}
	return "silence-" + name
}

// buildSilenceSnippet returns a kubectl-apply heredoc that, when run,
// creates a Silence CR matching the finding's exact subject for the
// given duration. SREs paste this into a terminal to suppress noise.
//
// Matcher is `subject` (most specific) — silences ONLY this exact
// finding, not the class. If the SRE wants to suppress a whole probe
// they edit the spec.matcher block to use `source` instead.
func buildSilenceSnippet(d DeltaDiag, ttl time.Duration) string {
	until := time.Now().UTC().Add(ttl).Format(time.RFC3339)
	name := silenceSnippetName(d.Subject)
	// `subject` matcher because the Silence CRD's spec.matcher
	// accepts {source, subject, severity}. Subject is the most
	// specific selector — exact match only.
	return fmt.Sprintf(`kubectl apply -f - <<EOF
apiVersion: cha.bionicaisolutions.com/v1alpha1
kind: Silence
metadata:
  name: %s
  namespace: cluster-health-autopilot
spec:
  matcher:
    subject: %q
  until: %q
  reason: "silenced from alert at %s"
EOF`, name, d.Subject, until, time.Now().UTC().Format(time.RFC3339))
}
