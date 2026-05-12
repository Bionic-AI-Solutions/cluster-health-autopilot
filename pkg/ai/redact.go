// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package ai

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"regexp"
	"strings"

	"github.com/Bionic-AI-Solutions/cluster-health-autopilot/pkg/diagnose"
)

// RedactDiagnostic returns a copy of d with identifiers hashed for LLM
// consumption. The caller (paid binary) uses the redacted form as LLM
// input and joins back to the original by Subject after the LLM response.
//
// Redaction policy:
//   - Namespace/name extracted from Subject are hashed (SHA-256 prefix,
//     8 hex chars) and consistently replaced wherever they appear in
//     Message/Remediation. Same identifier → same hash everywhere, so
//     the LLM can still correlate "Pod ns:abc12345 references Secret
//     name:def67890" across fields.
//   - IP addresses replaced with class-tagged labels (<ip:loopback>,
//     <ip:rfc1918>, <ip>)
//   - Internal hostnames (.svc, .local) hashed with trailing label
//     preserved for type signal (services vs node-local)
//   - UUIDs replaced with <uid>
//   - Cluster domain suffix ".cluster.local" collapsed to <cluster>
//
// The result is safe to send to a third-party LLM under standard DPA.
// For an in-cluster LLM, redaction is still applied to maintain the same
// privacy contract uniformly.
func RedactDiagnostic(d diagnose.Diagnostic) diagnose.Diagnostic {
	r := d
	// Extract identifiers from Subject BEFORE hashing it, so we can
	// substitute them everywhere they appear in free-form text.
	ns, name := extractSubjectIdentifiers(d.Subject)
	r.Subject = redactSubject(d.Subject)
	r.Message = redactWithIdentifiers(d.Message, ns, name)
	r.Remediation = redactWithIdentifiers(d.Remediation, ns, name)
	// Source and Severity are enum-like; they remain unredacted.
	// AI fields (Enrichment, ProposedActionID, ProposedRunbookID) are
	// outputs, not inputs — they should never be set when calling
	// RedactDiagnostic.
	return r
}

// extractSubjectIdentifiers pulls (namespace, name) from a Subject of
// shape "<kind>/<namespace>/<name>[/...]". Returns empty strings when
// the Subject does not match the convention.
func extractSubjectIdentifiers(subject string) (ns, name string) {
	parts := strings.SplitN(subject, "/", 4)
	if len(parts) < 3 {
		return "", ""
	}
	return parts[1], parts[2]
}

// redactWithIdentifiers applies the standard text scrubbers PLUS
// targeted replacement of the namespace/name identifiers extracted from
// the Diagnostic's Subject. Identifiers in free-form text are replaced
// with the same hash form used in the redacted Subject, so the LLM sees
// a consistent picture without ever receiving raw names.
func redactWithIdentifiers(s, ns, name string) string {
	if s == "" {
		return s
	}
	out := redactText(s)
	// Replace longer identifiers first so "billing-svc" wins over
	// "billing" when both are present in the text.
	if name != "" {
		out = strings.ReplaceAll(out, name, "name:"+hashShort(name))
	}
	if ns != "" {
		out = strings.ReplaceAll(out, ns, "ns:"+hashShort(ns))
	}
	return out
}

// redactSubject hashes the namespace/name segments of a Subject string
// while preserving the Kind and any trailing key path.
//
// Examples:
//
//	"Pod/billing/billing-svc-abc123"   -> "Pod/ns:a3f0b1c2/name:9e8d7c6b"
//	"Secret/playground/x/STRIPE_API_KEY" -> "Secret/ns:a3f0b1c2/name:9e8d7c6b/STRIPE_API_KEY"
//	"missing-key/billing/billing-svc-secrets/STRIPE_API_KEY"
//	  -> "missing-key/ns:a3f0b1c2/name:9e8d7c6b/STRIPE_API_KEY"
//
// Keys (which are non-identifying) are preserved literally — the LLM
// needs them for case/format reasoning.
func redactSubject(s string) string {
	parts := strings.Split(s, "/")
	if len(parts) < 3 {
		return s
	}
	// Convention: <kind>/<namespace>/<name>[/<key...>]
	parts[1] = "ns:" + hashShort(parts[1])
	parts[2] = "name:" + hashShort(parts[2])
	return strings.Join(parts, "/")
}

// hashShort returns the first 8 hex chars of SHA-256(s).
// Provides a stable identifier across runs without revealing the source.
func hashShort(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:4])
}

// redactText scrubs free-form text (Message, Remediation) of identifiers
// that could be sensitive when shipped to an LLM endpoint.
func redactText(s string) string {
	if s == "" {
		return s
	}
	s = ipRE.ReplaceAllStringFunc(s, redactIP)
	s = uuidRE.ReplaceAllString(s, "<uid>")
	s = clusterDomainRE.ReplaceAllString(s, "<cluster>")
	s = internalHostRE.ReplaceAllStringFunc(s, redactHost)
	return s
}

var (
	ipRE            = regexp.MustCompile(`\b(\d{1,3}\.){3}\d{1,3}\b`)
	uuidRE          = regexp.MustCompile(`\b[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}\b`)
	clusterDomainRE = regexp.MustCompile(`\.cluster\.local\b`)
	// Internal-looking hostnames: a label followed by .svc or .local.
	internalHostRE = regexp.MustCompile(`\b[a-zA-Z0-9-]+(\.[a-zA-Z0-9-]+)+\.(svc|local)\b`)
)

func redactIP(s string) string {
	// Differentiate well-known classes for the LLM's benefit.
	if ip := net.ParseIP(s); ip != nil {
		if ip.IsLoopback() {
			return "<ip:loopback>"
		}
		if ip4 := ip.To4(); ip4 != nil && (ip4[0] == 10 ||
			(ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31) ||
			(ip4[0] == 192 && ip4[1] == 168)) {
			return "<ip:rfc1918>"
		}
	}
	return "<ip>"
}

func redactHost(s string) string {
	// Keep the trailing label (.svc vs .local) so the LLM can reason
	// about service vs node-local lookups, but hash the prefix.
	if strings.HasSuffix(s, ".svc") {
		return fmt.Sprintf("<host:%s>.svc", hashShort(s))
	}
	if strings.HasSuffix(s, ".local") {
		return fmt.Sprintf("<host:%s>.local", hashShort(s))
	}
	return fmt.Sprintf("<host:%s>", hashShort(s))
}

// ScrubInjection removes known prompt-injection patterns from untrusted
// text before it is embedded in an LLM prompt.
//
// This is one layer of defense in depth — the primary defense is the
// system prompt's "treat <observed_data> as untrusted" instruction and
// the structured-output schema. This scrubber catches the most common
// drive-by patterns so they don't even appear in the prompt.
func ScrubInjection(s string) string {
	for _, pat := range injectionPatterns {
		s = pat.ReplaceAllString(s, "[redacted-instruction]")
	}
	return s
}

var injectionPatterns = []*regexp.Regexp{
	// Pattern accepts one OR more modifiers between "ignore/disregard"
	// and "instructions" (e.g. "ignore previous", "ignore any prior").
	regexp.MustCompile(`(?i)\b(ignore|disregard)((?:\s+(?:all|any|every|previous|prior|above|earlier))+)\s+instructions?\b`),
	regexp.MustCompile(`(?i)you are now [^.]+`),
	regexp.MustCompile(`(?i)\bsystem:\s*`),
	regexp.MustCompile(`(?i)\bassistant:\s*`),
	regexp.MustCompile(`(?i)<\|im_start\|>`),
	regexp.MustCompile(`(?i)<\|im_end\|>`),
	regexp.MustCompile(`(?i)\[\[system\]\]`),
	regexp.MustCompile(`(?i)override your (rules|policy|instructions)`),
	regexp.MustCompile(`(?i)pretend (you are|to be)`),
	regexp.MustCompile(`(?i)jailbreak`),
}
